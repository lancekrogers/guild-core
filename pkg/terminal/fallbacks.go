package terminal

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// FallbackStrategy defines a text transformation strategy
type FallbackStrategy struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
	From        string
	To          string
	When        func(caps Capabilities) bool
	Priority    int // Higher priority strategies are applied first
}

// FallbackProcessor handles graceful degradation of terminal output
type FallbackProcessor struct {
	mu         sync.RWMutex
	strategies []FallbackStrategy
	caps       Capabilities
	cache      map[string]string
	cacheSize  int
}

// NewFallbackProcessor creates a new fallback processor
func NewFallbackProcessor(caps Capabilities) *FallbackProcessor {
	fp := &FallbackProcessor{
		caps:      caps,
		cache:     make(map[string]string),
		cacheSize: 1000, // Cache up to 1000 transformations
	}

	// Register default strategies
	fp.registerDefaultStrategies()

	return fp
}

// registerDefaultStrategies sets up the default fallback strategies
func (fp *FallbackProcessor) registerDefaultStrategies() {
	// Unicode to ASCII fallbacks
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "checkmark",
		Description: "Unicode checkmark to ASCII",
		From:        "✓",
		To:          "[OK]",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    100,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "cross",
		Description: "Unicode cross to ASCII",
		From:        "✗",
		To:          "[X]",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    100,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "arrow-right",
		Description: "Unicode arrow to ASCII",
		From:        "→",
		To:          "->",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    90,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "arrow-left",
		Description: "Unicode arrow to ASCII",
		From:        "←",
		To:          "<-",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    90,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "bullet",
		Description: "Unicode bullet to ASCII",
		From:        "•",
		To:          "*",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    80,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "ellipsis",
		Description: "Unicode ellipsis to ASCII",
		From:        "…",
		To:          "...",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    80,
	})

	// Box drawing to ASCII
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "box-horizontal",
		Description: "Box drawing horizontal to ASCII",
		Pattern:     regexp.MustCompile(`[─━═]`),
		To:          "-",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    70,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "box-vertical",
		Description: "Box drawing vertical to ASCII",
		Pattern:     regexp.MustCompile(`[│┃║]`),
		To:          "|",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    70,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "box-corners",
		Description: "Box drawing corners to ASCII",
		Pattern:     regexp.MustCompile(`[┌┐└┘╔╗╚╝┏┓┗┛╭╮╰╯]`),
		To:          "+",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    70,
	})

	// Progress indicators
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "progress-blocks",
		Description: "Progress blocks to ASCII",
		Pattern:     regexp.MustCompile(`[▏▎▍▌▋▊▉█]`),
		To:          "#",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    60,
	})

	// Emoji fallbacks
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "emoji-success",
		Description: "Success emoji to text",
		Pattern:     regexp.MustCompile(`[✅🎉🎯💚]`),
		To:          "[SUCCESS]",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    50,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "emoji-error",
		Description: "Error emoji to text",
		Pattern:     regexp.MustCompile(`[❌🚫⛔💔]`),
		To:          "[ERROR]",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    50,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "emoji-warning",
		Description: "Warning emoji to text",
		Pattern:     regexp.MustCompile(`[⚠️🔶🟡⚡]`),
		To:          "[WARNING]",
		When:        func(caps Capabilities) bool { return !caps.Unicode },
		Priority:    50,
	})

	// Color to text indicators
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "color-success",
		Description: "Green color to success indicator",
		Pattern:     regexp.MustCompile(`\x1b\[(?:38;[25];)?(?:32|92)m`),
		To:          "[+] ",
		When:        func(caps Capabilities) bool { return caps.Colors == NoColor },
		Priority:    40,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "color-error",
		Description: "Red color to error indicator",
		Pattern:     regexp.MustCompile(`\x1b\[(?:38;[25];)?(?:31|91)m`),
		To:          "[-] ",
		When:        func(caps Capabilities) bool { return caps.Colors == NoColor },
		Priority:    40,
	})

	fp.RegisterStrategy(FallbackStrategy{
		Name:        "color-warning",
		Description: "Yellow color to warning indicator",
		Pattern:     regexp.MustCompile(`\x1b\[(?:38;[25];)?(?:33|93)m`),
		To:          "[!] ",
		When:        func(caps Capabilities) bool { return caps.Colors == NoColor },
		Priority:    40,
	})

	// Strip all ANSI codes for no-color terminals
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "strip-ansi",
		Description: "Remove all ANSI escape codes",
		Pattern:     regexp.MustCompile(`\x1b\[[0-9;]*m`),
		To:          "",
		When:        func(caps Capabilities) bool { return caps.Colors == NoColor },
		Priority:    10,
	})

	// Hyperlink stripping
	fp.RegisterStrategy(FallbackStrategy{
		Name:        "strip-hyperlinks",
		Description: "Remove hyperlink sequences",
		Pattern:     regexp.MustCompile(`\x1b\]8;;[^\x1b]*\x1b\\([^\x1b]*)\x1b\]8;;\x1b\\`),
		To:          "$1",
		When:        func(caps Capabilities) bool { return !caps.Hyperlinks },
		Priority:    30,
	})
}

// RegisterStrategy adds a new fallback strategy
func (fp *FallbackProcessor) RegisterStrategy(strategy FallbackStrategy) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.strategies = append(fp.strategies, strategy)
	fp.sortStrategies()
}

// sortStrategies sorts strategies by priority (highest first)
func (fp *FallbackProcessor) sortStrategies() {
	for i := 0; i < len(fp.strategies)-1; i++ {
		for j := i + 1; j < len(fp.strategies); j++ {
			if fp.strategies[j].Priority > fp.strategies[i].Priority {
				fp.strategies[i], fp.strategies[j] = fp.strategies[j], fp.strategies[i]
			}
		}
	}
}

// Process applies all applicable fallback strategies to the input
func (fp *FallbackProcessor) Process(input string) string {
	fp.mu.RLock()

	// Check cache first
	if cached, ok := fp.cache[input]; ok {
		fp.mu.RUnlock()
		return cached
	}

	strategies := make([]FallbackStrategy, len(fp.strategies))
	copy(strategies, fp.strategies)
	caps := fp.caps
	fp.mu.RUnlock()

	result := input

	// Apply each strategy in priority order
	for _, strategy := range strategies {
		if strategy.When(caps) {
			result = fp.applyStrategy(result, strategy)
		}
	}

	// Cache the result
	fp.mu.Lock()
	if len(fp.cache) < fp.cacheSize {
		fp.cache[input] = result
	}
	fp.mu.Unlock()

	return result
}

// applyStrategy applies a single fallback strategy
func (fp *FallbackProcessor) applyStrategy(input string, strategy FallbackStrategy) string {
	if strategy.Pattern != nil {
		return strategy.Pattern.ReplaceAllString(input, strategy.To)
	}

	return strings.ReplaceAll(input, strategy.From, strategy.To)
}

// ProcessContext applies fallbacks with context cancellation support
func (fp *FallbackProcessor) ProcessContext(ctx context.Context, input string) (string, error) {
	// Check context before processing
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// For large inputs, check context periodically
	if len(input) > 10000 {
		result := fp.processLargeInput(ctx, input)
		if err := ctx.Err(); err != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during processing")
		}
		return result, nil
	}

	return fp.Process(input), nil
}

// processLargeInput processes large inputs with periodic context checks
func (fp *FallbackProcessor) processLargeInput(ctx context.Context, input string) string {
	// Split into chunks for periodic context checking
	const chunkSize = 1000
	chunks := []string{}

	for i := 0; i < len(input); i += chunkSize {
		// Check context
		if ctx.Err() != nil {
			return input // Return original on cancellation
		}

		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}

		chunk := fp.Process(input[i:end])
		chunks = append(chunks, chunk)
	}

	return strings.Join(chunks, "")
}

// ClearCache clears the transformation cache
func (fp *FallbackProcessor) ClearCache() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.cache = make(map[string]string)
}

// UpdateCapabilities updates the capabilities used for fallback decisions
func (fp *FallbackProcessor) UpdateCapabilities(caps Capabilities) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.caps = caps
	fp.cache = make(map[string]string) // Clear cache when capabilities change
}

// GetStrategies returns all registered strategies
func (fp *FallbackProcessor) GetStrategies() []FallbackStrategy {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	strategies := make([]FallbackStrategy, len(fp.strategies))
	copy(strategies, fp.strategies)
	return strategies
}

// CommonFallbacks provides convenience functions for common transformations
type CommonFallbacks struct {
	processor *FallbackProcessor
}

// NewCommonFallbacks creates a new common fallbacks helper
func NewCommonFallbacks(caps Capabilities) *CommonFallbacks {
	return &CommonFallbacks{
		processor: NewFallbackProcessor(caps),
	}
}

// Success formats a success message
func (cf *CommonFallbacks) Success(message string) string {
	if cf.processor.caps.Unicode {
		message = "✓ " + message
	} else {
		message = "[OK] " + message
	}

	if cf.processor.caps.Colors >= Basic16 {
		return "\x1b[32m" + message + "\x1b[0m"
	}

	return message
}

// Error formats an error message
func (cf *CommonFallbacks) Error(message string) string {
	if cf.processor.caps.Unicode {
		message = "✗ " + message
	} else {
		message = "[X] " + message
	}

	if cf.processor.caps.Colors >= Basic16 {
		return "\x1b[31m" + message + "\x1b[0m"
	}

	return message
}

// Warning formats a warning message
func (cf *CommonFallbacks) Warning(message string) string {
	if cf.processor.caps.Unicode {
		message = "⚠ " + message
	} else {
		message = "[!] " + message
	}

	if cf.processor.caps.Colors >= Basic16 {
		return "\x1b[33m" + message + "\x1b[0m"
	}

	return message
}

// Info formats an info message
func (cf *CommonFallbacks) Info(message string) string {
	if cf.processor.caps.Unicode {
		message = "ℹ " + message
	} else {
		message = "[i] " + message
	}

	if cf.processor.caps.Colors >= Basic16 {
		return "\x1b[36m" + message + "\x1b[0m"
	}

	return message
}
