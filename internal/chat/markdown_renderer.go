package chat

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer provides rich content rendering for Guild chat
type MarkdownRenderer struct {
	renderer        *glamour.TermRenderer
	width           int
	codeStyle       lipgloss.Style
	lineNumberStyle lipgloss.Style
	formatter       chroma.Formatter
	style           *chroma.Style

	// Performance optimization
	renderCache  sync.Map // Cache for rendered content
	maxCacheSize int      // Maximum cache entries
	cacheHits    int64    // Performance metrics
	cacheMisses  int64

	// Error handling
	errorFallback bool  // Whether to use fallback on errors
	lastError     error // Last rendering error for debugging
}

// NewMarkdownRenderer creates a new markdown renderer with medieval theming
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	// Ensure minimum width for proper rendering
	if width < 40 {
		width = 40
	}
	if width > 200 {
		width = 200 // Cap maximum width for readability
	}

	// Create glamour renderer with medieval-themed styling
	var renderer *glamour.TermRenderer
	var lastErr error

	// Try multiple style configurations for robustness
	configs := []func() (*glamour.TermRenderer, error){
		func() (*glamour.TermRenderer, error) {
			// Primary: Custom medieval theme
			return glamour.NewTermRenderer(
				glamour.WithStylePath("dracula"), // Purple-themed style
				glamour.WithWordWrap(width-8),    // Account for borders and padding
				glamour.WithEmoji(),
				glamour.WithPreservedNewLines(),
			)
		},
		func() (*glamour.TermRenderer, error) {
			// Fallback 1: Auto style
			return glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(width-8),
				glamour.WithEmoji(),
			)
		},
		func() (*glamour.TermRenderer, error) {
			// Fallback 2: Basic style
			return glamour.NewTermRenderer(
				glamour.WithWordWrap(width - 8),
			)
		},
	}

	for _, config := range configs {
		r, err := config()
		if err == nil {
			renderer = r
			break
		}
		lastErr = err
	}

	// Create chroma formatter for syntax highlighting
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Get("terminal")
	}
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Use a medieval-themed style (monokai/dracula have purple accents)
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Get("dracula")
	}
	if style == nil {
		style = styles.Get("native")
	}
	if style == nil {
		style = styles.Fallback
	}

	// Medieval-themed style for code blocks with consistent purple borders
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")). // Consistent medieval purple
		Padding(0, 1).
		Margin(1, 0).
		MaxWidth(width - 8) // Ensure proper wrapping with extra padding

	// Line number style
	lineNumberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Dark gray
		MarginRight(1)

	return &MarkdownRenderer{
		renderer:        renderer,
		width:           width,
		codeStyle:       codeStyle,
		lineNumberStyle: lineNumberStyle,
		formatter:       formatter,
		style:           style,
		maxCacheSize:    100, // Cache up to 100 rendered items
		errorFallback:   true,
		lastError:       lastErr,
	}, nil
}

// Render processes content with markdown and syntax highlighting
func (m *MarkdownRenderer) Render(content string) string {
	// Quick check if content needs markdown processing
	if !m.needsMarkdownProcessing(content) {
		return content
	}

	// First, extract and process code blocks separately for better syntax highlighting
	processedContent := m.processCodeBlocks(content)

	// Then render the markdown
	rendered, err := m.renderer.Render(processedContent)
	if err != nil {
		// Fallback to original content if rendering fails
		return content
	}

	return rendered
}

// needsMarkdownProcessing checks if content contains markdown elements
func (m *MarkdownRenderer) needsMarkdownProcessing(content string) bool {
	// Quick heuristics to avoid unnecessary processing
	markdownIndicators := []string{"```", "#", "*", "_", "[", "`", "1.", "-"}
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// processCodeBlocks extracts code blocks and applies syntax highlighting
func (m *MarkdownRenderer) processCodeBlocks(content string) string {
	// Regex to match fenced code blocks
	codeBlockRegex := regexp.MustCompile("```(\\w+)?\\n([\\s\\S]*?)```")

	return codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		language := parts[1]
		code := parts[2]

		// Apply syntax highlighting
		highlighted := m.highlightCode(code, language)

		// Add line numbers for code blocks with more than 5 lines
		lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
		if len(lines) > 5 {
			highlighted = m.addLineNumbers(highlighted, len(lines))
		}

		// Add language label if specified
		if language != "" {
			langStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("141")). // Purple for medieval theme
				Bold(true).
				Margin(0, 0, 0, 1)

			langLabel := langStyle.Render(language)
			highlighted = langLabel + "\n" + highlighted
		}

		// Wrap in styled border
		return m.codeStyle.Render(highlighted)
	})
}

// highlightCode applies syntax highlighting to code content
func (m *MarkdownRenderer) highlightCode(code, language string) string {
	// Handle empty code
	if strings.TrimSpace(code) == "" {
		return code
	}

	// Get lexer for the language
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
		// Try common language aliases
		if lexer == nil {
			aliases := map[string]string{
				"golang": "go",
				"js":     "javascript",
				"py":     "python",
				"sh":     "bash",
				"yml":    "yaml",
			}
			if alias, exists := aliases[language]; exists {
				lexer = lexers.Get(alias)
			}
		}
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Fallback to original code
	}

	// Format with syntax highlighting
	var buf strings.Builder
	err = m.formatter.Format(&buf, m.style, iterator)
	if err != nil {
		return code // Fallback to original code
	}

	return buf.String()
}

// RenderInlineCode applies highlighting to inline code snippets
func (m *MarkdownRenderer) RenderInlineCode(code string) string {
	inlineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("8")).  // Dark gray background
		Foreground(lipgloss.Color("15")). // Bright white text
		Padding(0, 1)

	return inlineStyle.Render(code)
}

// DetectAndRenderContent automatically detects content type and applies appropriate rendering
func (m *MarkdownRenderer) DetectAndRenderContent(content string) string {
	// Check if content looks like markdown
	hasMarkdown := strings.Contains(content, "```") ||
		strings.Contains(content, "#") ||
		strings.Contains(content, "*") ||
		strings.Contains(content, "_") ||
		strings.Contains(content, "[") ||
		strings.Contains(content, "`")

	if hasMarkdown {
		return m.Render(content)
	}

	// For plain text, just return as-is (already styled by lipgloss)
	return content
}

// generateCacheKey creates a unique key for caching rendered content
func (m *MarkdownRenderer) generateCacheKey(content string) string {
	// Simple hash-based key generation
	// In production, this could use crypto/md5 for better distribution
	return content[:minInt(len(content), 32)]
}

// getCacheSize returns the approximate size of the cache
func (m *MarkdownRenderer) getCacheSize() int {
	size := 0
	m.renderCache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

// cleanCache removes old entries from the cache
func (m *MarkdownRenderer) cleanCache() {
	// Simple implementation: clear half the cache
	// In production, this could use LRU or time-based eviction
	count := 0
	target := m.maxCacheSize / 2
	m.renderCache.Range(func(key, value interface{}) bool {
		if count < target {
			m.renderCache.Delete(key)
			count++
		}
		return count < target
	})
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// addLineNumbers adds line numbers to code content
func (m *MarkdownRenderer) addLineNumbers(content string, lineCount int) string {
	lines := strings.Split(content, "\n")
	numberedLines := make([]string, 0, len(lines))

	// Calculate width for line number padding
	width := len(fmt.Sprintf("%d", lineCount))

	for i, line := range lines {
		// Format with proper padding and separator
		format := fmt.Sprintf("%%-%dd│", width)
		lineNumStr := fmt.Sprintf(format, i+1)

		// Apply style to line number
		styledLineNum := m.lineNumberStyle.Render(lineNumStr)

		// Combine line number with code line
		numberedLines = append(numberedLines, styledLineNum+line)
	}

	return strings.Join(numberedLines, "\n")
}

// GetCacheStats returns cache performance statistics
func (mr *MarkdownRenderer) GetCacheStats() string {
	total := mr.cacheHits + mr.cacheMisses
	if total == 0 {
		return "Cache stats: No cache activity yet"
	}

	ratio := float64(mr.cacheHits) / float64(total) * 100
	return fmt.Sprintf("Cache hits: %d, misses: %d, ratio: %.2f%%",
		mr.cacheHits, mr.cacheMisses, ratio)
}
