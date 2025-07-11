package hooks

import (
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

// Common patterns for sensitive data
var (
	// Credit card patterns
	creditCardPattern = regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`)

	// Email pattern
	emailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Z|a-z]{2,}\b`)

	// SSN pattern
	ssnPattern = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Phone number patterns
	phonePattern = regexp.MustCompile(`\b(?:\+?1[\s.-]?)?\(?[0-9]{3}\)?[\s.-]?[0-9]{3}[\s.-]?[0-9]{4}\b`)

	// API key/token patterns
	apiKeyPattern = regexp.MustCompile(`\b(api[_\-]?key|token|bearer|secret)["\s:=]+["']?[A-Za-z0-9\-._~+/]+["']?\b`)

	// IPv4 address pattern
	ipv4Pattern = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

	// AWS keys
	awsKeyPattern    = regexp.MustCompile(`\b(AKIA[0-9A-Z]{16})\b`)
	awsSecretPattern = regexp.MustCompile(`\b[A-Za-z0-9/+=]{40}\b`)

	// JWT tokens
	jwtPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

// DefaultPatterns contains all default sensitive patterns
var DefaultPatterns = []*regexp.Regexp{
	creditCardPattern,
	emailPattern,
	ssnPattern,
	phonePattern,
	apiKeyPattern,
	ipv4Pattern,
	awsKeyPattern,
	jwtPattern,
}

// SensitiveDataHook scrubs PII and sensitive data from logs
type SensitiveDataHook struct {
	patterns []*regexp.Regexp
	redactor RedactorFunc
	mu       sync.RWMutex
}

// RedactorFunc defines how to redact sensitive data
type RedactorFunc func(match string) string

// DefaultRedactor replaces sensitive data with [REDACTED]
func DefaultRedactor(match string) string {
	return "[REDACTED]"
}

// MaskingRedactor shows first and last characters
func MaskingRedactor(match string) string {
	if len(match) <= 2 {
		return "[REDACTED]"
	}
	if len(match) <= 4 {
		return match[:1] + "***"
	}
	return match[:2] + strings.Repeat("*", len(match)-4) + match[len(match)-2:]
}

// TypedRedactor shows the type of redacted data
func TypedRedactor(match string) string {
	switch {
	case creditCardPattern.MatchString(match):
		return "[REDACTED-CC]"
	case emailPattern.MatchString(match):
		return "[REDACTED-EMAIL]"
	case ssnPattern.MatchString(match):
		return "[REDACTED-SSN]"
	case phonePattern.MatchString(match):
		return "[REDACTED-PHONE]"
	case apiKeyPattern.MatchString(match):
		return "[REDACTED-KEY]"
	case ipv4Pattern.MatchString(match):
		return "[REDACTED-IP]"
	case awsKeyPattern.MatchString(match):
		return "[REDACTED-AWS-KEY]"
	case jwtPattern.MatchString(match):
		return "[REDACTED-JWT]"
	default:
		return "[REDACTED]"
	}
}

// NewSensitiveDataHook creates a new sensitive data hook with default patterns
func NewSensitiveDataHook() *SensitiveDataHook {
	return &SensitiveDataHook{
		patterns: DefaultPatterns,
		redactor: DefaultRedactor,
	}
}

// NewCustomSensitiveDataHook creates a hook with custom patterns
func NewCustomSensitiveDataHook(patterns []*regexp.Regexp, redactor RedactorFunc) *SensitiveDataHook {
	if redactor == nil {
		redactor = DefaultRedactor
	}
	return &SensitiveDataHook{
		patterns: patterns,
		redactor: redactor,
	}
}

// Process implements the Hook interface
func (h *SensitiveDataHook) Process(record *slog.Record) *slog.Record {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Scrub message
	scrubbed := h.scrubString(record.Message)
	if scrubbed != record.Message {
		newRecord := *record
		newRecord.Message = scrubbed
		record = &newRecord
	}

	// Scrub attributes
	var hasChanges bool
	var newAttrs []slog.Attr

	record.Attrs(func(a slog.Attr) bool {
		scrubbedAttr := h.scrubAttr(a)
		if !attributesEqual(a, scrubbedAttr) {
			hasChanges = true
		}
		newAttrs = append(newAttrs, scrubbedAttr)
		return true
	})

	if hasChanges {
		// Create new record with scrubbed attributes
		newRecord := &slog.Record{
			Time:    record.Time,
			Level:   record.Level,
			Message: record.Message,
			PC:      record.PC,
		}
		for _, attr := range newAttrs {
			newRecord.AddAttrs(attr)
		}
		return newRecord
	}

	return record
}

// AddPattern adds a new pattern to check for
func (h *SensitiveDataHook) AddPattern(pattern *regexp.Regexp) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.patterns = append(h.patterns, pattern)
}

// SetRedactor sets a custom redactor function
func (h *SensitiveDataHook) SetRedactor(redactor RedactorFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.redactor = redactor
}

// scrubString scrubs sensitive data from a string
func (h *SensitiveDataHook) scrubString(s string) string {
	result := s
	for _, pattern := range h.patterns {
		result = pattern.ReplaceAllStringFunc(result, h.redactor)
	}
	return result
}

// scrubAttr recursively scrubs sensitive data from attributes
func (h *SensitiveDataHook) scrubAttr(a slog.Attr) slog.Attr {
	// Scrub key
	scrubbedKey := h.scrubString(a.Key)

	// Scrub value
	switch a.Value.Kind() {
	case slog.KindString:
		scrubbedValue := h.scrubString(a.Value.String())
		if scrubbedValue != a.Value.String() || scrubbedKey != a.Key {
			return slog.String(scrubbedKey, scrubbedValue)
		}
	case slog.KindGroup:
		group := a.Value.Group()
		var changed bool
		newGroup := make([]slog.Attr, len(group))
		for i, ga := range group {
			newGroup[i] = h.scrubAttr(ga)
			if !attributesEqual(ga, newGroup[i]) {
				changed = true
			}
		}
		if changed || scrubbedKey != a.Key {
			return slog.Group(scrubbedKey, attrsToAny(newGroup)...)
		}
	case slog.KindAny:
		// Special handling for error types
		if err, ok := a.Value.Any().(error); ok {
			scrubbedMsg := h.scrubString(err.Error())
			if scrubbedMsg != err.Error() || scrubbedKey != a.Key {
				return slog.String(scrubbedKey, scrubbedMsg)
			}
		}
	}

	if scrubbedKey != a.Key {
		return slog.Attr{Key: scrubbedKey, Value: a.Value}
	}

	return a
}

// attributesEqual checks if two attributes are equal
func attributesEqual(a, b slog.Attr) bool {
	if a.Key != b.Key {
		return false
	}
	return a.Value.Equal(b.Value)
}

// attrsToAny converts attributes to []any for slog.Group
func attrsToAny(attrs []slog.Attr) []any {
	result := make([]any, 0, len(attrs)*2)
	for _, a := range attrs {
		result = append(result, a.Key, a.Value.Any())
	}
	return result
}
