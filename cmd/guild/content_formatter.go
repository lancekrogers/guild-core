package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ContentFormatter provides high-level content formatting for different message types
type ContentFormatter struct {
	markdownRenderer *MarkdownRenderer
	width           int
	messageStyles   map[string]lipgloss.Style
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
		width:           width,
		messageStyles:   messageStyles,
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
	cf.messageStyles["agent"] = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))    // Blue
	cf.messageStyles["system"] = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))  // Gray
	cf.messageStyles["error"] = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))   // Red
	cf.messageStyles["tool"] = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))    // Orange
}

// applyMinimalTheme applies a minimal monochrome scheme
func (cf *ContentFormatter) applyMinimalTheme() {
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // White
	for key := range cf.messageStyles {
		cf.messageStyles[key] = defaultStyle
	}
}
