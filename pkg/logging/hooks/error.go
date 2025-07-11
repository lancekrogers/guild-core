package hooks

import (
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// ErrorHook tracks errors and adds additional context
type ErrorHook struct {
	mu         sync.RWMutex
	errorCount map[string]int
	lastSeen   map[string]time.Time

	// Configuration
	includeStackTrace bool
	trackFrequency    bool
	dedupeWindow      time.Duration
}

// ErrorInfo contains error tracking information
type ErrorInfo struct {
	Count     int
	LastSeen  time.Time
	FirstSeen time.Time
}

// NewErrorHook creates a new error tracking hook
func NewErrorHook(includeStackTrace, trackFrequency bool) *ErrorHook {
	return &ErrorHook{
		errorCount:        make(map[string]int),
		lastSeen:          make(map[string]time.Time),
		includeStackTrace: includeStackTrace,
		trackFrequency:    trackFrequency,
		dedupeWindow:      5 * time.Minute,
	}
}

// Process implements the Hook interface
func (h *ErrorHook) Process(record *slog.Record) *slog.Record {
	// Only process error level logs
	if record.Level < slog.LevelError {
		return record
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Look for error attribute
	var errorFound bool
	var errorKey string
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "error" || a.Key == "err" {
			if err, ok := a.Value.Any().(error); ok && err != nil {
				errorFound = true
				errorKey = err.Error()
				return false
			}
		}
		return true
	})

	if !errorFound {
		// Use message as error key if no error attribute
		errorKey = record.Message
	}

	// Track error frequency
	attrs := make([]slog.Attr, 0)
	if h.trackFrequency {
		h.errorCount[errorKey]++
		h.lastSeen[errorKey] = record.Time

		attrs = append(attrs,
			slog.Int("error_count", h.errorCount[errorKey]),
			slog.Duration("since_last", record.Time.Sub(h.lastSeen[errorKey])),
		)
	}

	// Add stack trace if enabled
	if h.includeStackTrace {
		attrs = append(attrs, slog.String("stack_trace", h.captureStackTrace()))
	}

	// Add error metadata
	attrs = append(attrs,
		slog.String("error_key", errorKey),
		slog.Time("error_time", record.Time),
	)

	// Create new record with additional attributes
	if len(attrs) > 0 {
		newRecord := *record
		newRecord.AddAttrs(attrs...)
		return &newRecord
	}

	return record
}

// GetErrorInfo returns tracking info for a specific error
func (h *ErrorHook) GetErrorInfo(errorKey string) (ErrorInfo, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count, hasCount := h.errorCount[errorKey]
	lastSeen, hasLastSeen := h.lastSeen[errorKey]

	if !hasCount || !hasLastSeen {
		return ErrorInfo{}, false
	}

	return ErrorInfo{
		Count:    count,
		LastSeen: lastSeen,
	}, true
}

// GetTopErrors returns the most frequent errors
func (h *ErrorHook) GetTopErrors(limit int) []struct {
	Error string
	Count int
} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create slice for sorting
	type errorCount struct {
		Error string
		Count int
	}
	errors := make([]errorCount, 0, len(h.errorCount))
	for err, count := range h.errorCount {
		errors = append(errors, errorCount{Error: err, Count: count})
	}

	// Sort by count (descending)
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[j].Count > errors[i].Count {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	// Return top N
	if limit > len(errors) {
		limit = len(errors)
	}

	result := make([]struct {
		Error string
		Count int
	}, limit)
	for i := 0; i < limit; i++ {
		result[i].Error = errors[i].Error
		result[i].Count = errors[i].Count
	}

	return result
}

// Reset clears all error tracking data
func (h *ErrorHook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errorCount = make(map[string]int)
	h.lastSeen = make(map[string]time.Time)
}

// CleanOld removes error entries older than the specified duration
func (h *ErrorHook) CleanOld(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for key, lastSeen := range h.lastSeen {
		if now.Sub(lastSeen) > maxAge {
			delete(h.errorCount, key)
			delete(h.lastSeen, key)
		}
	}
}

// captureStackTrace captures the current stack trace
func (h *ErrorHook) captureStackTrace() string {
	const maxStackDepth = 32
	pc := make([]uintptr, maxStackDepth)
	n := runtime.Callers(4, pc) // Skip runtime.Callers, captureStackTrace, Process, and log call

	if n == 0 {
		return "no stack trace available"
	}

	frames := runtime.CallersFrames(pc[:n])
	var trace string

	for {
		frame, more := frames.Next()
		trace += frame.Function + "\n\t" + frame.File + ":" + string(rune(frame.Line)) + "\n"
		if !more {
			break
		}
	}

	return trace
}
