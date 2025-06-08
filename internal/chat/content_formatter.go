package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ContentFormatter provides high-level content formatting for different message types
type ContentFormatter struct {
	markdownRenderer *MarkdownRenderer
	width            int
	messageStyles    map[string]lipgloss.Style

	// Content optimization
	maxContentLength int  // Maximum content length before truncation
	showMoreEnabled  bool // Whether to enable "show more" functionality
}

// NewContentFormatter creates a new content formatter with medieval theming
func NewContentFormatter(markdownRenderer *MarkdownRenderer, width int) *ContentFormatter {
	// Medieval-themed styles for different message types
	messageStyles := map[string]lipgloss.Style{
		"agent": lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")). // Green
			Bold(true),
		"system": lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")). // Yellow
			Italic(true),
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red
			Bold(true),
		"tool": lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")). // Orange
			Bold(true),
		"user": lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // White
			Bold(false),
		"thinking": lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")). // Purple
			Italic(true),
		"working": lipgloss.NewStyle().
			Foreground(lipgloss.Color("76")). // Bright green
			Bold(true),
	}

	return &ContentFormatter{
		markdownRenderer: markdownRenderer,
		width:            width,
		messageStyles:    messageStyles,
		maxContentLength: 5000, // Default max content length
		showMoreEnabled:  true, // Enable "show more" by default
	}
}

// FormatAgentResponse formats agent responses with rich content rendering
func (cf *ContentFormatter) FormatAgentResponse(content string, agentID string) string {
	// Apply markdown rendering for rich content
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Add agent-specific formatting if needed
	if agentID != "" {
		// Add subtle agent attribution for complex responses
		if len(content) > 200 || strings.Contains(content, "```") {
			attribution := cf.messageStyles["agent"].Render(fmt.Sprintf("🤖 %s", agentID))
			renderedContent = fmt.Sprintf("%s\n%s", attribution, renderedContent)
		}
	}

	return renderedContent
}

// FormatSystemMessage formats system messages with consistent styling
func (cf *ContentFormatter) FormatSystemMessage(content string) string {
	// System messages often contain status updates and notifications
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Add system message formatting
	if cf.isImportantSystemMessage(content) {
		// Highlight important system messages
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("220")). // Gold
			Padding(0, 1).
			Margin(1, 0)
		renderedContent = style.Render(renderedContent)
	}

	return renderedContent
}

// FormatErrorMessage formats error messages with emphasis and helpful styling
func (cf *ContentFormatter) FormatErrorMessage(content string) string {
	// Render any markdown in error content
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Style error messages for visibility
	errorStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red
		Padding(0, 1).
		Margin(1, 0).
		Foreground(lipgloss.Color("196"))

	// Add error icon and formatting
	errorIcon := "❌"
	styledContent := errorStyle.Render(fmt.Sprintf("%s %s", errorIcon, renderedContent))

	return styledContent
}

// FormatToolOutput formats tool execution output with syntax highlighting
func (cf *ContentFormatter) FormatToolOutput(content string, toolName string) string {
	// Tool output often contains code, logs, or structured data
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Add tool-specific formatting
	toolHeader := cf.messageStyles["tool"].Render(fmt.Sprintf("🔧 %s", toolName))

	// Style tool output with distinct borders
	toolStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")). // Orange
		Padding(0, 1).
		Margin(1, 0)

	styledContent := toolStyle.Render(fmt.Sprintf("%s\n%s", toolHeader, renderedContent))

	return styledContent
}

// FormatThinkingMessage formats agent thinking/planning messages
func (cf *ContentFormatter) FormatThinkingMessage(content string, agentID string) string {
	// Thinking messages show agent planning and reasoning
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Add thinking indicators
	thinkingIcon := "🤔"
	if agentID != "" {
		thinkingIcon = fmt.Sprintf("🤔 %s", agentID)
	}

	// Style thinking messages with muted colors
	thinkingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")). // Purple
		Italic(true).
		Padding(0, 1)

	styledContent := thinkingStyle.Render(fmt.Sprintf("%s %s", thinkingIcon, renderedContent))

	return styledContent
}

// FormatWorkingMessage formats agent working/executing messages
func (cf *ContentFormatter) FormatWorkingMessage(content string, agentID string) string {
	// Working messages show active task execution
	renderedContent := cf.markdownRenderer.DetectAndRenderContent(content)

	// Add working indicators
	workingIcon := "⚙️"
	if agentID != "" {
		workingIcon = fmt.Sprintf("⚙️ %s", agentID)
	}

	// Style working messages with active colors
	workingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("76")). // Bright green
		Bold(true).
		Padding(0, 1)

	styledContent := workingStyle.Render(fmt.Sprintf("%s %s", workingIcon, renderedContent))

	return styledContent
}

// FormatUserMessage formats user input messages
func (cf *ContentFormatter) FormatUserMessage(content string) string {
	// User messages might contain commands or queries
	// Apply light markdown processing for user-created content
	return cf.markdownRenderer.DetectAndRenderContent(content)
}

// FormatTimestamp formats timestamps consistently across message types
func (cf *ContentFormatter) FormatTimestamp(timestamp time.Time) string {
	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Italic(true)

	return timestampStyle.Render(timestamp.Format("15:04:05"))
}

// Helper methods

// isImportantSystemMessage determines if a system message needs emphasis
func (cf *ContentFormatter) isImportantSystemMessage(content string) bool {
	importantKeywords := []string{
		"error", "failed", "warning", "critical",
		"completed", "finished", "ready",
		"started", "connecting", "disconnected",
	}

	lowerContent := strings.ToLower(content)
	for _, keyword := range importantKeywords {
		if strings.Contains(lowerContent, keyword) {
			return true
		}
	}

	return false
}

// GetMessageStyle returns the appropriate style for a message type
func (cf *ContentFormatter) GetMessageStyle(messageType string) lipgloss.Style {
	if style, exists := cf.messageStyles[messageType]; exists {
		return style
	}
	return cf.messageStyles["system"] // Default fallback
}

// UpdateWidth adjusts the formatter for new terminal width
func (cf *ContentFormatter) UpdateWidth(newWidth int) {
	cf.width = newWidth
	// Note: MarkdownRenderer width should be updated separately if needed
}

// SetWidth is an alias for UpdateWidth for compatibility
func (cf *ContentFormatter) SetWidth(newWidth int) {
	cf.UpdateWidth(newWidth)
}

// FormatMarkdown formats markdown content using the markdown renderer
func (cf *ContentFormatter) FormatMarkdown(content string) string {
	if cf.markdownRenderer != nil {
		return cf.markdownRenderer.Render(content)
	}
	return content
}

// FormatCodeBlock formats a code block with syntax highlighting
func (cf *ContentFormatter) FormatCodeBlock(content, language string) string {
	if cf.markdownRenderer != nil {
		codeBlock := fmt.Sprintf("```%s\n%s\n```", language, content)
		return cf.markdownRenderer.Render(codeBlock)
	}
	return content
}

// SetTheme allows switching between different visual themes
func (cf *ContentFormatter) SetTheme(theme string) {
	switch theme {
	case "medieval":
		// Already using medieval theme
	case "modern":
		// Could implement a modern theme here
		cf.applyModernTheme()
	case "minimal":
		// Could implement a minimal theme here
		cf.applyMinimalTheme()
	}
}

// applyModernTheme applies a modern color scheme
func (cf *ContentFormatter) applyModernTheme() {
	cf.messageStyles["agent"] = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))   // Blue
	cf.messageStyles["system"] = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Gray
	cf.messageStyles["error"] = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))  // Red
	cf.messageStyles["tool"] = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))   // Orange
}

// applyMinimalTheme applies a minimal monochrome scheme
func (cf *ContentFormatter) applyMinimalTheme() {
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // White
	for key := range cf.messageStyles {
		cf.messageStyles[key] = defaultStyle
	}
}

// ContentType represents the detected type of content
type ContentType int

const (
	ContentTypePlainText ContentType = iota
	ContentTypeMarkdown
	ContentTypeCode
	ContentTypeJSON
	ContentTypeYAML
	ContentTypeMixed
)

// DetectContentType intelligently detects the type of content
func (cf *ContentFormatter) DetectContentType(content string) ContentType {
	// Quick empty check
	if strings.TrimSpace(content) == "" {
		return ContentTypePlainText
	}

	// Check for code blocks first (highest priority)
	if strings.Contains(content, "```") {
		return ContentTypeMixed // Has both markdown and code
	}

	// Check for JSON
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		// Likely JSON
		return ContentTypeJSON
	}

	// Check for YAML
	if strings.Contains(content, ":") && strings.Contains(content, "\n") {
		lines := strings.Split(content, "\n")
		yamlScore := 0
		for _, line := range lines {
			if strings.Contains(line, ": ") || strings.HasSuffix(line, ":") {
				yamlScore++
			}
		}
		if yamlScore > len(lines)/3 {
			return ContentTypeYAML
		}
	}

	// Check for markdown indicators
	markdownIndicators := []string{"#", "*", "_", "[", "`", "1.", "-"}
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			return ContentTypeMarkdown
		}
	}

	// Check if it looks like code (heuristic)
	if cf.looksLikeCode(content) {
		return ContentTypeCode
	}

	return ContentTypePlainText
}

// looksLikeCode uses heuristics to detect if content looks like code
func (cf *ContentFormatter) looksLikeCode(content string) bool {
	codeIndicators := []string{
		"func ", "function ", "def ", "class ", "import ", "const ", "var ", "let ",
		"if (", "for (", "while (", "return ", "package ", "public ", "private ",
		"=>", "==", "!=", "&&", "||", ":=", "++", "--",
	}

	indicatorCount := 0
	for _, indicator := range codeIndicators {
		if strings.Contains(content, indicator) {
			indicatorCount++
		}
	}

	// If we find multiple code indicators, it's likely code
	return indicatorCount >= 2
}

// InferLanguage attempts to infer the programming language from code content
func (cf *ContentFormatter) InferLanguage(code string) string {
	// Language detection heuristics
	languagePatterns := map[string][]string{
		"go":         {"package ", "func ", ":=", "import (", "go mod", "defer ", "chan "},
		"python":     {"def ", "import ", "from ", "__init__", "class ", "self.", "pip "},
		"javascript": {"function ", "const ", "let ", "var ", "=>", "require(", "export "},
		"typescript": {"interface ", "type ", ": string", ": number", "export class", "import {"},
		"json":       {"\":", "\": ", "{\n", "[\n", "},", "],"},
		"yaml":       {"- ", ": ", "---", "...", "!!", "<<:"},
		"bash":       {"#!/bin/bash", "#!/bin/sh", "if [", "then", "fi", "do", "done", "echo", "export"},
		"sql":        {"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "CREATE TABLE", "ALTER", "DROP"},
		"rust":       {"fn ", "let ", "mut ", "impl ", "trait ", "use ", "pub ", "mod ", "match"},
		"java":       {"public class", "private ", "public static", "import java", "extends", "implements"},
		"ruby":       {"def ", "end", "class ", "module ", "require ", "puts ", "attr_"},
		"dockerfile": {"FROM ", "RUN ", "CMD ", "EXPOSE ", "ENV ", "COPY ", "WORKDIR"},
		"makefile":   {"PHONY:", "all:", "clean:", "install:", "$(", "@echo", "CFLAGS"},
	}

	scores := make(map[string]int)
	for lang, patterns := range languagePatterns {
		for _, pattern := range patterns {
			if strings.Contains(code, pattern) {
				scores[lang]++
			}
		}
	}

	// Find language with highest score
	maxScore := 0
	detectedLang := ""
	for lang, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedLang = lang
		}
	}

	return detectedLang
}

// OptimizeContentLength truncates long content with "show more" indicator
func (cf *ContentFormatter) OptimizeContentLength(content string) string {
	if !cf.showMoreEnabled || len(content) <= cf.maxContentLength {
		return content
	}

	// Find a good truncation point (end of line or sentence)
	truncateAt := cf.maxContentLength

	// Try to find end of line
	if idx := strings.LastIndex(content[:truncateAt], "\n"); idx > truncateAt*3/4 {
		truncateAt = idx
	} else if idx := strings.LastIndex(content[:truncateAt], ". "); idx > truncateAt*3/4 {
		// Try to find end of sentence
		truncateAt = idx + 1
	}

	truncated := content[:truncateAt]
	remaining := len(content) - truncateAt

	// Add "show more" indicator
	showMoreStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")). // Purple
		Italic(true)

	showMore := showMoreStyle.Render(fmt.Sprintf("\n... (%d more characters) ...", remaining))

	return truncated + showMore
}

// ProcessContent applies the full content processing pipeline
func (cf *ContentFormatter) ProcessContent(content string) string {
	// Detect content type
	contentType := cf.DetectContentType(content)

	// Apply appropriate processing based on type
	var processed string
	switch contentType {
	case ContentTypeCode:
		// Wrap in code block with inferred language
		lang := cf.InferLanguage(content)
		processed = fmt.Sprintf("```%s\n%s\n```", lang, content)
		processed = cf.markdownRenderer.Render(processed)

	case ContentTypeJSON:
		// Format as JSON code block
		processed = fmt.Sprintf("```json\n%s\n```", content)
		processed = cf.markdownRenderer.Render(processed)

	case ContentTypeYAML:
		// Format as YAML code block
		processed = fmt.Sprintf("```yaml\n%s\n```", content)
		processed = cf.markdownRenderer.Render(processed)

	case ContentTypeMarkdown, ContentTypeMixed:
		// Process with markdown renderer
		processed = cf.markdownRenderer.Render(content)

	default:
		// Plain text - no special processing
		processed = content
	}

	// Apply content length optimization
	return cf.OptimizeContentLength(processed)
}

// IsRichContent checks if content contains rich formatting elements
func (cf *ContentFormatter) IsRichContent(content string) bool {
	contentType := cf.DetectContentType(content)
	return contentType != ContentTypePlainText
}

// FormatMessage formats a generic message with intelligent content processing
func (cf *ContentFormatter) FormatMessage(messageType, content string, metadata map[string]string) string {
	// Apply error boundaries - never crash on malformed content
	defer func() {
		if r := recover(); r != nil {
			// Log the error (in production, this would go to a logger)
			fmt.Printf("Error formatting message: %v\n", r)
		}
	}()

	// Process content through the pipeline
	processed := cf.ProcessContent(content)

	// Apply message type specific styling
	switch messageType {
	case "agent":
		if agentID, ok := metadata["agentID"]; ok {
			return cf.FormatAgentResponse(processed, agentID)
		}
		return cf.FormatAgentResponse(processed, "")

	case "system":
		return cf.FormatSystemMessage(processed)

	case "error":
		return cf.FormatErrorMessage(processed)

	case "tool":
		if toolName, ok := metadata["toolName"]; ok {
			return cf.FormatToolOutput(processed, toolName)
		}
		return cf.FormatToolOutput(processed, "Tool")

	case "thinking":
		if agentID, ok := metadata["agentID"]; ok {
			return cf.FormatThinkingMessage(processed, agentID)
		}
		return cf.FormatThinkingMessage(processed, "")

	case "working":
		if agentID, ok := metadata["agentID"]; ok {
			return cf.FormatWorkingMessage(processed, agentID)
		}
		return cf.FormatWorkingMessage(processed, "")

	case "user":
		return cf.FormatUserMessage(processed)

	default:
		return processed
	}
}

// ContentFormatterInterface defines the contract for content formatting
type ContentFormatterInterface interface {
	FormatAgentResponse(content string, agentID string) string
	FormatSystemMessage(content string) string
	FormatErrorMessage(content string) string
	FormatToolOutput(content string, toolName string) string
	FormatThinkingMessage(content string, agentID string) string
	FormatWorkingMessage(content string, agentID string) string
	FormatUserMessage(content string) string
	FormatTimestamp(timestamp time.Time) string
	UpdateWidth(newWidth int)
}

// PlainTextFormatter provides fallback plain text formatting when rich rendering fails
type PlainTextFormatter struct {
	width int
}

// NewPlainTextFormatter creates a plain text formatter for graceful degradation
func NewPlainTextFormatter(width int) *PlainTextFormatter {
	return &PlainTextFormatter{
		width: width,
	}
}

// FormatAgentResponse formats agent responses with simple text formatting
func (ptf *PlainTextFormatter) FormatAgentResponse(content string, agentID string) string {
	if agentID != "" {
		return fmt.Sprintf("🤖 %s: %s", agentID, content)
	}
	return fmt.Sprintf("🤖 %s", content)
}

// FormatSystemMessage formats system messages with simple text formatting
func (ptf *PlainTextFormatter) FormatSystemMessage(content string) string {
	return fmt.Sprintf("⚙️ System: %s", content)
}

// FormatErrorMessage formats error messages with simple text formatting
func (ptf *PlainTextFormatter) FormatErrorMessage(content string) string {
	return fmt.Sprintf("❌ Error: %s", content)
}

// FormatToolOutput formats tool execution output with simple text formatting
func (ptf *PlainTextFormatter) FormatToolOutput(content string, toolName string) string {
	return fmt.Sprintf("🔧 %s: %s", toolName, content)
}

// FormatThinkingMessage formats agent thinking/planning messages with simple text formatting
func (ptf *PlainTextFormatter) FormatThinkingMessage(content string, agentID string) string {
	if agentID != "" {
		return fmt.Sprintf("🤔 %s: %s", agentID, content)
	}
	return fmt.Sprintf("🤔 %s", content)
}

// FormatWorkingMessage formats agent working/executing messages with simple text formatting
func (ptf *PlainTextFormatter) FormatWorkingMessage(content string, agentID string) string {
	if agentID != "" {
		return fmt.Sprintf("⚙️ %s: %s", agentID, content)
	}
	return fmt.Sprintf("⚙️ %s", content)
}

// FormatUserMessage formats user input messages with simple text formatting
func (ptf *PlainTextFormatter) FormatUserMessage(content string) string {
	return content // User messages don't need special formatting in plain text mode
}

// FormatTimestamp formats timestamps with simple text formatting
func (ptf *PlainTextFormatter) FormatTimestamp(timestamp time.Time) string {
	return timestamp.Format("15:04:05")
}

// UpdateWidth adjusts the formatter for new terminal width
func (ptf *PlainTextFormatter) UpdateWidth(newWidth int) {
	ptf.width = newWidth
}
