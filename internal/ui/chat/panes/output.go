// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guild-ventures/guild-core/internal/ui/chat/common"
	"github.com/guild-ventures/guild-core/internal/ui/chat/common/layout"
	"github.com/guild-ventures/guild-core/internal/ui/formatting"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// OutputPane handles the display of messages and rich content
type OutputPane interface {
	layout.PaneInterface

	// Message management
	AddMessage(message common.ChatMessage)
	GetMessages() []common.ChatMessage
	Clear()

	// Search functionality
	Search(pattern string) []int
	HighlightMatches(matches []int)

	// Content formatting
	SetMarkdownEnabled(enabled bool)
	SetRichContentEnabled(enabled bool)
	SetContentFormatter(formatter *formatting.ContentFormatter)

	// Scrolling
	ScrollToBottom()
	ScrollToTop()
	ScrollUp(lines int)
	ScrollDown(lines int)
}

// outputPaneImpl implements the OutputPane interface
type outputPaneImpl struct {
	*layout.BasePane

	// Message storage
	messages []common.ChatMessage

	// Viewport for scrolling
	viewport viewport.Model

	// Rich content settings
	markdownEnabled    bool
	richContentEnabled bool

	// Search state
	searchPattern string
	searchMatches []int
	currentMatch  int

	// Styling
	messageStyles map[common.MessageType]lipgloss.Style

	// Visual components
	contentFormatter *formatting.ContentFormatter

	// Context
	ctx context.Context
}

// NewOutputPane creates a new output pane
func NewOutputPane(width, height int, richContentEnabled bool) (OutputPane, error) {
	if width < 20 || height < 5 {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "output pane dimensions too small: %dx%d", width, height).
			WithComponent("panes.output").
			WithOperation("NewOutputPane")
	}

	ctx := context.Background()
	basePane := layout.NewBasePane(ctx, "output", width, height)
	basePane.SetConstraints(layout.OutputPaneConstraints())
	basePane.ApplyDefaultStyling()

	// Initialize viewport
	vp := viewport.New(width-2, height-2) // Account for borders
	vp.SetContent("")

	pane := &outputPaneImpl{
		BasePane:           basePane,
		messages:           make([]common.ChatMessage, 0),
		viewport:           vp,
		markdownEnabled:    true,
		richContentEnabled: richContentEnabled,
		searchMatches:      make([]int, 0),
		messageStyles:      createMessageStyles(),
		ctx:                ctx,
	}

	return pane, nil
}

// SetContentFormatter sets the content formatter for rich content rendering
func (op *outputPaneImpl) SetContentFormatter(formatter *formatting.ContentFormatter) {
	op.contentFormatter = formatter
}

// createMessageStyles creates styled message formatting
func createMessageStyles() map[common.MessageType]lipgloss.Style {
	return map[common.MessageType]lipgloss.Style{
		common.MsgUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")). // Purple
			Bold(true),
		common.MsgAgent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")), // Green
		common.MsgSystem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Gray
			Italic(true),
		common.MsgError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red
			Bold(true),
		common.MsgToolStart: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange
		common.MsgToolProgress: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Italic(true),
		common.MsgToolComplete: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Bright green
		common.MsgAgentThinking: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true),
		common.MsgAgentWorking: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		common.MsgPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")), // Purple
		common.MsgToolError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		common.MsgToolAuth: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")), // Orange
	}
}

// AddMessage adds a new message to the output pane
func (op *outputPaneImpl) AddMessage(message common.ChatMessage) {
	op.messages = append(op.messages, message)
	op.updateViewportContent()
	op.ScrollToBottom()
}

// GetMessages returns all messages
func (op *outputPaneImpl) GetMessages() []common.ChatMessage {
	return op.messages
}

// Clear removes all messages
func (op *outputPaneImpl) Clear() {
	op.messages = make([]common.ChatMessage, 0)
	op.updateViewportContent()
}

// Search finds messages matching the pattern
func (op *outputPaneImpl) Search(pattern string) []int {
	if pattern == "" {
		op.searchMatches = make([]int, 0)
		return op.searchMatches
	}

	op.searchPattern = pattern
	op.searchMatches = make([]int, 0)

	pattern = strings.ToLower(pattern)

	for i, msg := range op.messages {
		content := strings.ToLower(msg.Content)
		if strings.Contains(content, pattern) {
			op.searchMatches = append(op.searchMatches, i)
		}
	}

	op.currentMatch = 0
	op.updateViewportContent()

	return op.searchMatches
}

// HighlightMatches highlights search matches in the display
func (op *outputPaneImpl) HighlightMatches(matches []int) {
	op.searchMatches = matches
	op.updateViewportContent()
}

// SetMarkdownEnabled enables or disables markdown rendering
func (op *outputPaneImpl) SetMarkdownEnabled(enabled bool) {
	op.markdownEnabled = enabled
	op.updateViewportContent()
}

// SetRichContentEnabled enables or disables rich content rendering
func (op *outputPaneImpl) SetRichContentEnabled(enabled bool) {
	op.richContentEnabled = enabled
	op.updateViewportContent()
}

// ScrollToBottom scrolls to the bottom of the message history
func (op *outputPaneImpl) ScrollToBottom() {
	op.viewport.GotoBottom()
}

// ScrollToTop scrolls to the top of the message history
func (op *outputPaneImpl) ScrollToTop() {
	op.viewport.GotoTop()
}

// ScrollUp scrolls up by the specified number of lines
func (op *outputPaneImpl) ScrollUp(lines int) {
	for i := 0; i < lines; i++ {
		op.viewport.LineUp(1)
	}
}

// ScrollDown scrolls down by the specified number of lines
func (op *outputPaneImpl) ScrollDown(lines int) {
	for i := 0; i < lines; i++ {
		op.viewport.LineDown(1)
	}
}

// Resize updates the pane dimensions
func (op *outputPaneImpl) Resize(width, height int) {
	op.BasePane.Resize(width, height)

	// Update viewport dimensions
	innerWidth, innerHeight := op.GetInnerDimensions()
	op.viewport.Width = innerWidth
	op.viewport.Height = innerHeight

	// Regenerate content for new width
	op.updateViewportContent()
}

// Update handles Bubble Tea messages
func (op *outputPaneImpl) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update base pane first
	_, cmd := op.BasePane.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return op.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		op.Resize(msg.Width, msg.Height)
	}

	// Update viewport
	var vpCmd tea.Cmd
	op.viewport, vpCmd = op.viewport.Update(msg)

	return op, tea.Batch(cmd, vpCmd)
}

// handleKeyPress handles keyboard input for the output pane
func (op *outputPaneImpl) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !op.IsFocused() {
		return op, nil
	}

	switch msg.String() {
	case "j", "down":
		op.ScrollDown(1)
	case "k", "up":
		op.ScrollUp(1)
	case "g":
		op.ScrollToTop()
	case "G":
		op.ScrollToBottom()
	case "ctrl+u":
		op.ScrollUp(10)
	case "ctrl+d":
		op.ScrollDown(10)
	case "pgup":
		op.viewport.ViewUp()
	case "pgdown":
		op.viewport.ViewDown()
	case "home":
		op.ScrollToTop()
	case "end":
		op.ScrollToBottom()
	case "n":
		// Next search match
		if len(op.searchMatches) > 0 {
			op.currentMatch = (op.currentMatch + 1) % len(op.searchMatches)
			op.scrollToMatch()
		}
	case "N":
		// Previous search match
		if len(op.searchMatches) > 0 {
			op.currentMatch = (op.currentMatch - 1 + len(op.searchMatches)) % len(op.searchMatches)
			op.scrollToMatch()
		}
	}

	return op, nil
}

// View renders the output pane
func (op *outputPaneImpl) View() string {
	// Get viewport content
	viewportContent := op.viewport.View()

	// Apply border and styling
	return op.RenderWithBorder(viewportContent)
}

// updateViewportContent regenerates the viewport content from messages
func (op *outputPaneImpl) updateViewportContent() {
	if len(op.messages) == 0 {
		op.viewport.SetContent("")
		return
	}

	var content strings.Builder

	for i, msg := range op.messages {
		formattedMsg := op.formatMessage(msg, i)
		content.WriteString(formattedMsg)

		// Add newline between messages
		if i < len(op.messages)-1 {
			content.WriteString("\n")
		}
	}

	op.viewport.SetContent(content.String())
}

// formatMessage formats a single message for display
func (op *outputPaneImpl) formatMessage(msg common.ChatMessage, index int) string {
	// Check if this message matches the search
	isMatch := op.isSearchMatch(index)

	// Get message style
	style := op.messageStyles[msg.Type]

	// Apply search highlighting if needed
	if isMatch && index == op.getCurrentSearchMatch() {
		style = style.Background(lipgloss.Color("236")) // Dark gray background for current match
	} else if isMatch {
		style = style.Background(lipgloss.Color("238")) // Slightly lighter for other matches
	}

	// Format timestamp
	timestamp := msg.Timestamp.Format("15:04:05")

	// Format agent/user prefix
	var prefix string
	var messageType string
	switch msg.Type {
	case common.MsgUser:
		prefix = "👤 You"
		messageType = "user"
	case common.MsgAgent:
		if msg.AgentID != "" {
			prefix = fmt.Sprintf("🤖 %s", msg.AgentID)
		} else {
			prefix = "🤖 Agent"
		}
		messageType = "agent"
	case common.MsgSystem:
		prefix = "🏰 System"
		messageType = "system"
	case common.MsgError:
		prefix = "❌ Error"
		messageType = "error"
	case common.MsgToolStart, common.MsgToolProgress, common.MsgToolComplete:
		prefix = "🔨 Tool"
		messageType = "tool"
	case common.MsgAgentThinking:
		prefix = "🤔 Agent"
		messageType = "thinking"
	case common.MsgAgentWorking:
		prefix = "⚙️ Agent"
		messageType = "working"
	case common.MsgPrompt:
		prefix = "📜 Prompt"
		messageType = "system"
	case common.MsgToolError:
		prefix = "❌ Tool"
		messageType = "error"
	case common.MsgToolAuth:
		prefix = "🔐 Auth"
		messageType = "system"
	default:
		prefix = "📝 Message"
		messageType = "agent"
	}

	// Format the complete message
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render(fmt.Sprintf("[%s] %s", timestamp, prefix))

	// Process content based on rich content settings
	content := op.processContentWithType(msg.Content, messageType, msg.Metadata)

	// Apply message style to content
	styledContent := style.Render(content)

	return fmt.Sprintf("%s\n%s", header, styledContent)
}

// processContent processes message content based on enabled features
func (op *outputPaneImpl) processContent(content string) string {
	return op.processContentWithType(content, "agent", nil)
}

// processContentWithType processes message content with specific type
func (op *outputPaneImpl) processContentWithType(content string, messageType string, metadata map[string]string) string {
	if !op.richContentEnabled {
		return content
	}

	// Use ContentFormatter if available for rich content rendering
	if op.contentFormatter != nil && op.markdownEnabled {
		formatted := op.contentFormatter.FormatMessage(messageType, content, metadata)
		return formatted
	}

	return content
}

// isSearchMatch checks if a message index is in the search results
func (op *outputPaneImpl) isSearchMatch(index int) bool {
	for _, match := range op.searchMatches {
		if match == index {
			return true
		}
	}
	return false
}

// getCurrentSearchMatch returns the index of the current search match
func (op *outputPaneImpl) getCurrentSearchMatch() int {
	if len(op.searchMatches) == 0 || op.currentMatch >= len(op.searchMatches) {
		return -1
	}
	return op.searchMatches[op.currentMatch]
}

// scrollToMatch scrolls to the current search match
func (op *outputPaneImpl) scrollToMatch() {
	if len(op.searchMatches) == 0 {
		return
	}

	matchIndex := op.getCurrentSearchMatch()
	if matchIndex < 0 {
		return
	}

	// Calculate which line the match is on
	// This is a simplified calculation - in a real implementation,
	// you'd need to account for message wrapping and formatting
	totalLines := len(op.messages) * 2 // Assume 2 lines per message on average
	matchLine := matchIndex * 2

	// Calculate viewport position
	targetPosition := float64(matchLine) / float64(totalLines)

	op.viewport.SetYOffset(int(targetPosition * float64(op.viewport.TotalLineCount())))
}

// GetSearchInfo returns information about the current search
func (op *outputPaneImpl) GetSearchInfo() (pattern string, matches int, current int) {
	return op.searchPattern, len(op.searchMatches), op.currentMatch + 1
}

// ClearSearch clears the current search state
func (op *outputPaneImpl) ClearSearch() {
	op.searchPattern = ""
	op.searchMatches = make([]int, 0)
	op.currentMatch = 0
	op.updateViewportContent()
}

// GetStats returns statistics about the output pane
func (op *outputPaneImpl) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["total_messages"] = len(op.messages)
	stats["viewport_height"] = op.viewport.Height
	stats["viewport_width"] = op.viewport.Width
	stats["scroll_position"] = op.viewport.YOffset
	stats["total_lines"] = op.viewport.TotalLineCount()
	stats["markdown_enabled"] = op.markdownEnabled
	stats["rich_content_enabled"] = op.richContentEnabled
	stats["search_pattern"] = op.searchPattern
	stats["search_matches"] = len(op.searchMatches)
	stats["current_match"] = op.currentMatch + 1

	// Message type counts
	typeCounts := make(map[common.MessageType]int)
	for _, msg := range op.messages {
		typeCounts[msg.Type]++
	}
	stats["message_type_counts"] = typeCounts

	return stats
}

// ExportMessages exports all messages to a string format
func (op *outputPaneImpl) ExportMessages(format string) (string, error) {
	if len(op.messages) == 0 {
		return "", nil
	}

	switch format {
	case "text", "txt":
		return op.exportAsText(), nil
	case "markdown", "md":
		return op.exportAsMarkdown(), nil
	case "json":
		return op.exportAsJSON()
	default:
		return "", gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported export format: %s", format).
			WithComponent("panes.output").
			WithOperation("ExportMessages")
	}
}

// exportAsText exports messages as plain text
func (op *outputPaneImpl) exportAsText() string {
	var content strings.Builder

	content.WriteString("Guild Chat Export\n")
	content.WriteString("================\n\n")

	for _, msg := range op.messages {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		content.WriteString(fmt.Sprintf("[%s] %s: %s\n\n", timestamp, msg.AgentID, msg.Content))
	}

	return content.String()
}

// exportAsMarkdown exports messages as markdown
func (op *outputPaneImpl) exportAsMarkdown() string {
	var content strings.Builder

	content.WriteString("# Guild Chat Export\n\n")

	for _, msg := range op.messages {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		content.WriteString(fmt.Sprintf("## %s - %s\n\n", timestamp, msg.AgentID))
		content.WriteString(fmt.Sprintf("%s\n\n---\n\n", msg.Content))
	}

	return content.String()
}

// exportAsJSON exports messages as JSON
func (op *outputPaneImpl) exportAsJSON() (string, error) {
	// This would use json.Marshal in a real implementation
	// For now, return a placeholder
	return "JSON export not implemented", gerror.New(gerror.ErrCodeInternal, "JSON export not implemented", nil).
		WithComponent("panes.output").
		WithOperation("exportAsJSON")
}
