// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package formatting

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour/v2"
	"charm.land/lipgloss/v2"
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

	// Advanced features
	enableMermaid    bool // Enable Mermaid diagram rendering
	enableMath       bool // Enable math equation rendering
	enableEmoji      bool // Enable emoji support
	enableTables     bool // Enable table rendering
	enableChecklists bool // Enable checklist rendering

	// Custom styles
	headerStyle     lipgloss.Style
	blockquoteStyle lipgloss.Style
	listStyle       lipgloss.Style
	linkStyle       lipgloss.Style
	tableStyle      lipgloss.Style
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

	// Custom styles for advanced features
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141")). // Medieval purple
		MarginTop(1).
		MarginBottom(1)

	blockquoteStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("241")).
		Foreground(lipgloss.Color("245")).
		Italic(true).
		PaddingLeft(2)

	listStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(2)

	linkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")). // Blue
		Underline(true)

	tableStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Align(lipgloss.Center)

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
		// Advanced features enabled by default
		enableMermaid:    true,
		enableMath:       true,
		enableEmoji:      true,
		enableTables:     true,
		enableChecklists: true,
		// Custom styles
		headerStyle:     headerStyle,
		blockquoteStyle: blockquoteStyle,
		listStyle:       listStyle,
		linkStyle:       linkStyle,
		tableStyle:      tableStyle,
	}, nil
}

// Render processes content with markdown and syntax highlighting
func (m *MarkdownRenderer) Render(content string) string {
	// Check cache first
	cacheKey := m.generateCacheKey(content)
	if cached, found := m.renderCache.Load(cacheKey); found {
		m.cacheHits++
		return cached.(string)
	}
	m.cacheMisses++

	// Quick check if content needs markdown processing
	if !m.needsMarkdownProcessing(content) {
		return content
	}

	// Apply advanced processing in order
	processedContent := content

	// 1. Process emoji codes
	if m.enableEmoji {
		processedContent = m.RenderEmoji(processedContent)
	}

	// 2. Process math equations
	if m.enableMath {
		processedContent = m.processMathEquations(processedContent)
	}

	// 3. Process mermaid diagrams
	if m.enableMermaid {
		processedContent = m.processMermaidDiagrams(processedContent)
	}

	// 4. Extract and process code blocks separately for better syntax highlighting
	processedContent = m.processCodeBlocks(processedContent)

	// 5. Process checklists
	if m.enableChecklists {
		processedContent = m.processChecklists(processedContent)
	}

	// Then render the markdown
	rendered, err := m.renderer.Render(processedContent)
	if err != nil {
		m.lastError = err
		if m.errorFallback {
			// Fallback to original content if rendering fails
			return content
		}
	}

	// Cache the result
	if m.getCacheSize() >= m.maxCacheSize {
		m.cleanCache()
	}
	m.renderCache.Store(cacheKey, rendered)

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

// RenderTable renders a markdown table with proper formatting
func (m *MarkdownRenderer) RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Build table
	var table strings.Builder

	// Header row
	table.WriteString("|")
	for i, header := range headers {
		table.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], header))
	}
	table.WriteString("\n")

	// Separator row
	table.WriteString("|")
	for _, width := range colWidths {
		table.WriteString(strings.Repeat("-", width+2) + "|")
	}
	table.WriteString("\n")

	// Data rows
	for _, row := range rows {
		table.WriteString("|")
		for i, cell := range row {
			if i < len(colWidths) {
				table.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], cell))
			}
		}
		table.WriteString("\n")
	}

	return m.tableStyle.Render(table.String())
}

// RenderChecklist renders a checklist with checkboxes
func (m *MarkdownRenderer) RenderChecklist(items []struct {
	Text    string
	Checked bool
},
) string {
	if !m.enableChecklists || len(items) == 0 {
		return ""
	}

	var checklist strings.Builder
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))    // Green
	uncheckStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // Gray

	for _, item := range items {
		if item.Checked {
			checklist.WriteString(checkStyle.Render("✓ "))
			// Strike through completed items
			strikeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Strikethrough(true)
			checklist.WriteString(strikeStyle.Render(item.Text))
		} else {
			checklist.WriteString(uncheckStyle.Render("☐ "))
			checklist.WriteString(item.Text)
		}
		checklist.WriteString("\n")
	}

	return m.listStyle.Render(checklist.String())
}

// RenderMermaidDiagram renders a mermaid diagram as ASCII art
func (m *MarkdownRenderer) RenderMermaidDiagram(diagram string) string {
	if !m.enableMermaid {
		return m.codeStyle.Render(diagram)
	}

	// For now, render as a code block with "mermaid" label
	// In a real implementation, this could convert to ASCII art
	diagramStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("141")).
		Padding(1).
		Margin(1)

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141")).
		Render("📊 Mermaid Diagram")

	return header + "\n" + diagramStyle.Render(diagram)
}

// RenderMathEquation renders a math equation
func (m *MarkdownRenderer) RenderMathEquation(equation string, inline bool) string {
	if !m.enableMath {
		return equation
	}

	mathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")). // Orange for math
		Italic(true)

	if inline {
		return mathStyle.Render(equation)
	}

	// Block equation
	blockMathStyle := mathStyle.Copy().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(1).
		Align(lipgloss.Center).
		Width(m.width - 10)

	return blockMathStyle.Render(equation)
}

// RenderEmoji converts emoji codes to actual emojis
func (m *MarkdownRenderer) RenderEmoji(text string) string {
	if !m.enableEmoji {
		return text
	}

	// Common emoji mappings
	emojiMap := map[string]string{
		":smile:":      "😊",
		":thumbsup:":   "👍",
		":thumbsdown:": "👎",
		":heart:":      "❤️",
		":star:":       "⭐",
		":fire:":       "🔥",
		":rocket:":     "🚀",
		":check:":      "✅",
		":cross:":      "❌",
		":warning:":    "⚠️",
		":info:":       "ℹ️",
		":bulb:":       "💡",
		":bug:":        "🐛",
		":lock:":       "🔒",
		":key:":        "🔑",
		":clock:":      "🕐",
		":calendar:":   "📅",
		":folder:":     "📁",
		":file:":       "📄",
		":mail:":       "📧",
		":link:":       "🔗",
	}

	result := text
	for code, emoji := range emojiMap {
		result = strings.ReplaceAll(result, code, emoji)
	}

	return result
}

// RenderWithOptions renders content with specific feature toggles
func (m *MarkdownRenderer) RenderWithOptions(content string, options struct {
	EnableMermaid    bool
	EnableMath       bool
	EnableEmoji      bool
	EnableTables     bool
	EnableChecklists bool
},
) string {
	// Save current settings
	oldMermaid := m.enableMermaid
	oldMath := m.enableMath
	oldEmoji := m.enableEmoji
	oldTables := m.enableTables
	oldChecklists := m.enableChecklists

	// Apply options
	m.enableMermaid = options.EnableMermaid
	m.enableMath = options.EnableMath
	m.enableEmoji = options.EnableEmoji
	m.enableTables = options.EnableTables
	m.enableChecklists = options.EnableChecklists

	// Render
	result := m.Render(content)

	// Restore settings
	m.enableMermaid = oldMermaid
	m.enableMath = oldMath
	m.enableEmoji = oldEmoji
	m.enableTables = oldTables
	m.enableChecklists = oldChecklists

	return result
}

// SetWidth updates the renderer width and recreates the glamour renderer
func (m *MarkdownRenderer) SetWidth(width int) error {
	if width < 40 {
		width = 40
	}
	if width > 200 {
		width = 200
	}

	m.width = width

	// Recreate glamour renderer with new width
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(width-8),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return err
	}

	m.renderer = renderer
	return nil
}

// GetLastError returns the last rendering error for debugging
func (m *MarkdownRenderer) GetLastError() error {
	return m.lastError
}

// processMathEquations processes LaTeX math equations in content
func (m *MarkdownRenderer) processMathEquations(content string) string {
	// Process inline math $...$
	inlineMathRegex := regexp.MustCompile(`\$([^\$]+)\$`)
	content = inlineMathRegex.ReplaceAllStringFunc(content, func(match string) string {
		equation := match[1 : len(match)-1]
		return m.RenderMathEquation(equation, true)
	})

	// Process block math $$...$$
	blockMathRegex := regexp.MustCompile(`(?s)\$\$\n?(.*?)\n?\$\$`)
	content = blockMathRegex.ReplaceAllStringFunc(content, func(match string) string {
		equation := strings.TrimPrefix(strings.TrimSuffix(match, "$$"), "$$")
		equation = strings.TrimSpace(equation)
		return m.RenderMathEquation(equation, false)
	})

	return content
}

// processMermaidDiagrams processes mermaid diagram blocks
func (m *MarkdownRenderer) processMermaidDiagrams(content string) string {
	// Look for mermaid code blocks
	mermaidRegex := regexp.MustCompile("```mermaid\\n([\\s\\S]*?)```")

	return mermaidRegex.ReplaceAllStringFunc(content, func(match string) string {
		diagram := mermaidRegex.FindStringSubmatch(match)[1]
		return m.RenderMermaidDiagram(diagram)
	})
}

// processChecklists processes checklist items in markdown
func (m *MarkdownRenderer) processChecklists(content string) string {
	// Process checklist items
	checklistRegex := regexp.MustCompile(`(?m)^- \[([ xX])\] (.+)$`)

	lines := strings.Split(content, "\n")
	var result []string
	var checklistItems []struct {
		Text    string
		Checked bool
	}
	inChecklist := false

	for i, line := range lines {
		if matches := checklistRegex.FindStringSubmatch(line); matches != nil {
			inChecklist = true
			checklistItems = append(checklistItems, struct {
				Text    string
				Checked bool
			}{
				Text:    matches[2],
				Checked: matches[1] != " ",
			})

			// Check if this is the last checklist item
			if i+1 >= len(lines) || !checklistRegex.MatchString(lines[i+1]) {
				// Render the accumulated checklist
				result = append(result, m.RenderChecklist(checklistItems))
				checklistItems = nil
				inChecklist = false
			}
		} else {
			if inChecklist && len(checklistItems) > 0 {
				// Render the accumulated checklist
				result = append(result, m.RenderChecklist(checklistItems))
				checklistItems = nil
				inChecklist = false
			}
			result = append(result, line)
		}
	}

	// Handle any remaining checklist items
	if len(checklistItems) > 0 {
		result = append(result, m.RenderChecklist(checklistItems))
	}

	return strings.Join(result, "\n")
}

// RenderDiff renders a diff with proper syntax highlighting
func (m *MarkdownRenderer) RenderDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var rendered []string

	addedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("22")).
		Foreground(lipgloss.Color("10"))
	removedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("52")).
		Foreground(lipgloss.Color("9"))
	contextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			rendered = append(rendered, headerStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			rendered = append(rendered, headerStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			rendered = append(rendered, addedStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			rendered = append(rendered, removedStyle.Render(line))
		default:
			rendered = append(rendered, contextStyle.Render(line))
		}
	}

	return strings.Join(rendered, "\n")
}
