// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-framework/guild-core/internal/ui/chat/common/layout"
	"github.com/guild-framework/guild-core/internal/ui/chat/completion"
	"github.com/guild-framework/guild-core/internal/ui/vim"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// InputPane handles user input with auto-completion and history
type InputPane interface {
	layout.PaneInterface

	// Value management
	GetValue() string
	SetValue(value string)

	// Completion
	ShowCompletions(results []completion.CompletionResult)
	HideCompletions()
	GetCurrentCompletion() string
	AcceptCompletion()

	// History
	SetHistory(history []string)
	NavigateHistory(direction int) // -1 for previous, 1 for next

	// Input modes
	SetMultilineEnabled(enabled bool)
	SetPlaceholder(placeholder string)

	// Events
	OnSubmit(callback func(string))
	OnChange(callback func(string))
}

// inputPaneImpl implements the InputPane interface
type inputPaneImpl struct {
	*layout.BasePane

	// Text input
	textarea textarea.Model

	// Completion state
	showingCompletions bool
	completions        []completion.CompletionResult
	completionIndex    int

	// History state
	history       []string
	historyIndex  int
	originalValue string // Store original input when navigating history

	// Settings
	multilineEnabled bool
	placeholder      string

	// Callbacks
	onSubmit func(string)
	onChange func(string)

	// Styling
	normalStyle     lipgloss.Style
	completionStyle lipgloss.Style
	historyStyle    lipgloss.Style

	// Context
	ctx context.Context
}

// NewInputPane creates a new input pane
func NewInputPane(width, height int, completionEnabled bool) (InputPane, error) {
	if width < 20 || height < 3 {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "input pane dimensions too small: %dx%d", width, height).
			WithComponent("panes.input").
			WithOperation("NewInputPane")
	}

	ctx := context.Background()
	basePane := layout.NewBasePane(ctx, "input", width, height)
	basePane.SetConstraints(layout.InputPaneConstraints())
	basePane.ApplyDefaultStyling()

	// Initialize textarea
	ta := textarea.New()
	ta.Placeholder = "Message agents with @agent-name or use /commands..."
	ta.Focus()
	ta.SetHeight(height - 2) // Account for borders
	ta.SetWidth(width - 2)   // Account for borders
	ta.ShowLineNumbers = false

	pane := &inputPaneImpl{
		BasePane:         basePane,
		textarea:         ta,
		completions:      make([]completion.CompletionResult, 0),
		history:          make([]string, 0),
		historyIndex:     -1,
		multilineEnabled: false,
		placeholder:      "Message agents with @agent-name or use /commands...",
		normalStyle:      createInputStyle(),
		completionStyle:  createCompletionStyle(),
		historyStyle:     createHistoryStyle(),
		ctx:              ctx,
	}

	return pane, nil
}

// NewVimEnabledInputPane creates a new input pane with vim mode support
func NewVimEnabledInputPane(width, height int, completionEnabled bool, vimModeManager *vim.VimModeManager) (InputPane, error) {
	// Create the base input pane
	basePane, err := NewInputPane(width, height, completionEnabled)
	if err != nil {
		return nil, err
	}

	// Cast to implementation type
	inputPaneImpl, ok := basePane.(*inputPaneImpl)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "failed to cast input pane to implementation type", nil).
			WithComponent("panes.input").
			WithOperation("NewVimEnabledInputPane")
	}

	// Wrap with vim adapter
	vimAdapter := NewVimInputAdapter(inputPaneImpl, vimModeManager)

	// Return as InputPane interface
	return vimAdapter, nil
}

// createInputStyle creates the normal input styling
func createInputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")). // Purple
		Padding(0, 1)
}

// createCompletionStyle creates styling for completion mode
func createCompletionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Bright purple
		Padding(0, 1)
}

// createHistoryStyle creates styling for history mode
func createHistoryStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")). // Orange
		Padding(0, 1)
}

// GetValue returns the current input value
func (ip *inputPaneImpl) GetValue() string {
	return ip.textarea.Value()
}

// SetValue sets the input value
func (ip *inputPaneImpl) SetValue(value string) {
	ip.textarea.SetValue(value)
	ip.triggerOnChange()
}

// ShowCompletions displays auto-completion suggestions
func (ip *inputPaneImpl) ShowCompletions(results []completion.CompletionResult) {
	ip.completions = results
	ip.showingCompletions = len(results) > 0
	ip.completionIndex = 0
}

// HideCompletions hides the completion popup
func (ip *inputPaneImpl) HideCompletions() {
	ip.showingCompletions = false
	ip.completions = make([]completion.CompletionResult, 0)
	ip.completionIndex = 0
}

// GetCurrentCompletion returns the currently selected completion
func (ip *inputPaneImpl) GetCurrentCompletion() string {
	if !ip.showingCompletions || len(ip.completions) == 0 || ip.completionIndex >= len(ip.completions) {
		return ""
	}
	return ip.completions[ip.completionIndex].Content
}

// AcceptCompletion accepts the current completion suggestion
func (ip *inputPaneImpl) AcceptCompletion() {
	if completion := ip.GetCurrentCompletion(); completion != "" {
		ip.SetValue(completion)
		ip.HideCompletions()
	}
}

// SetHistory sets the command history
func (ip *inputPaneImpl) SetHistory(history []string) {
	ip.history = make([]string, len(history))
	copy(ip.history, history)
	ip.historyIndex = len(ip.history)
}

// NavigateHistory navigates through command history
func (ip *inputPaneImpl) NavigateHistory(direction int) {
	if len(ip.history) == 0 {
		return
	}

	// Store current input when starting history navigation
	if ip.historyIndex == len(ip.history) {
		ip.originalValue = ip.GetValue()
	}

	// Navigate
	ip.historyIndex += direction

	// Clamp to valid range
	if ip.historyIndex < 0 {
		ip.historyIndex = 0
	} else if ip.historyIndex >= len(ip.history) {
		ip.historyIndex = len(ip.history)
		// Restore original value
		ip.SetValue(ip.originalValue)
		return
	}

	// Set history value
	ip.SetValue(ip.history[ip.historyIndex])
}

// SetMultilineEnabled enables or disables multiline input
func (ip *inputPaneImpl) SetMultilineEnabled(enabled bool) {
	ip.multilineEnabled = enabled

	if enabled {
		ip.textarea.SetHeight(ip.GetRect().Height - 2)
	} else {
		ip.textarea.SetHeight(3) // Single line input
	}
}

// SetPlaceholder sets the input placeholder text
func (ip *inputPaneImpl) SetPlaceholder(placeholder string) {
	ip.placeholder = placeholder
	ip.textarea.Placeholder = placeholder
}

// OnSubmit sets the submit callback
func (ip *inputPaneImpl) OnSubmit(callback func(string)) {
	ip.onSubmit = callback
}

// OnChange sets the change callback
func (ip *inputPaneImpl) OnChange(callback func(string)) {
	ip.onChange = callback
}

// GetPreferredHeight calculates the preferred height based on content
func (ip *inputPaneImpl) GetPreferredHeight() int {
	content := ip.GetValue()
	if content == "" {
		return 3 // Minimum height for empty input
	}

	// Count lines in content
	lines := strings.Count(content, "\n") + 1

	// Add height for borders (2) and padding
	preferredHeight := lines + 2

	// Limit to reasonable maximum (1/3 of typical terminal height)
	maxHeight := 15
	if preferredHeight > maxHeight {
		preferredHeight = maxHeight
	}

	// Ensure minimum height
	if preferredHeight < 3 {
		preferredHeight = 3
	}

	return preferredHeight
}

// UpdateConstraints updates the pane's layout constraints based on content
func (ip *inputPaneImpl) UpdateConstraints() {
	constraints := ip.GetConstraints()
	constraints.PreferredHeight = ip.GetPreferredHeight()
	ip.SetConstraints(constraints)
}

// triggerOnChange calls the onChange callback if set and updates layout
func (ip *inputPaneImpl) triggerOnChange() {
	// Update layout constraints based on new content
	ip.UpdateConstraints()

	if ip.onChange != nil {
		ip.onChange(ip.GetValue())
	}
}

// Resize updates the pane dimensions
func (ip *inputPaneImpl) Resize(width, height int) {
	ip.BasePane.Resize(width, height)

	// Update textarea dimensions
	ip.textarea.SetWidth(width - 2)
	if ip.multilineEnabled {
		ip.textarea.SetHeight(height - 2)
	} else {
		ip.textarea.SetHeight(3)
	}
}

// Update handles Bubble Tea messages
func (ip *inputPaneImpl) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update base pane first
	_, cmd := ip.BasePane.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return ip.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		ip.Resize(msg.Width, msg.Height)
	}

	// Update textarea
	var taCmd tea.Cmd
	ip.textarea, taCmd = ip.textarea.Update(msg)

	return ip, tea.Batch(cmd, taCmd)
}

// handleKeyPress handles keyboard input
func (ip *inputPaneImpl) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return ip, tea.Quit

	case "enter":
		if ip.showingCompletions {
			ip.AcceptCompletion()
			return ip, nil
		}

		if !ip.multilineEnabled {
			return ip.handleSubmit()
		}
		// In multiline mode, enter adds a new line (handled by textarea)

	case "tab":
		if ip.showingCompletions {
			ip.cyclePreviousCompletion()
			return ip, nil
		}
		// Request completions
		return ip.requestCompletions()

	case "shift+tab":
		if ip.showingCompletions {
			ip.cycleNextCompletion()
			return ip, nil
		}

	case "esc":
		if ip.showingCompletions {
			ip.HideCompletions()
			return ip, nil
		}

	case "up":
		if ip.showingCompletions {
			ip.cyclePreviousCompletion()
			return ip, nil
		}
		ip.NavigateHistory(-1)
		return ip, nil

	case "down":
		if ip.showingCompletions {
			ip.cycleNextCompletion()
			return ip, nil
		}
		ip.NavigateHistory(1)
		return ip, nil

	case "ctrl+r":
		// History search - TODO: implement
		return ip, nil

	case "ctrl+l":
		// Clear input
		ip.SetValue("")
		ip.HideCompletions()
		return ip, nil

	case "ctrl+u":
		// Delete to beginning of line
		ip.SetValue("")
		return ip, nil

	case "ctrl+k":
		// Delete to end of line
		// TODO: implement partial line deletion
		return ip, nil

	case "ctrl+w":
		// Delete previous word
		// TODO: implement word deletion
		return ip, nil
	}

	// For other keys, check if the value changed and trigger onChange
	oldValue := ip.GetValue()

	// Let textarea handle the key
	var cmd tea.Cmd
	ip.textarea, cmd = ip.textarea.Update(msg)

	// Check if value changed
	newValue := ip.GetValue()
	if oldValue != newValue {
		ip.triggerOnChange()

		// Hide completions if input changed significantly
		if ip.showingCompletions && !strings.HasPrefix(newValue, oldValue) {
			ip.HideCompletions()
		}
	}

	return ip, cmd
}

// handleSubmit processes input submission
func (ip *inputPaneImpl) handleSubmit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(ip.GetValue())
	if value == "" {
		return ip, nil
	}

	// Add to history
	if len(ip.history) == 0 || ip.history[len(ip.history)-1] != value {
		ip.history = append(ip.history, value)
	}
	ip.historyIndex = len(ip.history)

	// Clear input
	ip.SetValue("")
	ip.HideCompletions()

	// Trigger submit callback
	if ip.onSubmit != nil {
		ip.onSubmit(value)
	}

	return ip, nil
}

// requestCompletions requests auto-completion suggestions
func (ip *inputPaneImpl) requestCompletions() (tea.Model, tea.Cmd) {
	currentValue := ip.GetValue()

	// Simple completion logic - this would be more sophisticated in practice
	if strings.HasPrefix(currentValue, "/") {
		// Command completions
		return ip, func() tea.Msg {
			// Using anonymous struct to avoid import dependencies
			return struct {
				Type  string
				Input string
			}{
				Type:  "completion_request",
				Input: currentValue,
			}
		}
	} else if strings.HasPrefix(currentValue, "@") {
		// Agent completions
		return ip, func() tea.Msg {
			// Using anonymous struct to avoid import dependencies
			return struct {
				Type  string
				Input string
			}{
				Type:  "completion_request",
				Input: currentValue,
			}
		}
	}

	return ip, nil
}

// cyclePreviousCompletion moves to the previous completion
func (ip *inputPaneImpl) cyclePreviousCompletion() {
	if len(ip.completions) == 0 {
		return
	}

	ip.completionIndex--
	if ip.completionIndex < 0 {
		ip.completionIndex = len(ip.completions) - 1
	}

	// Update input with current completion
	if completion := ip.GetCurrentCompletion(); completion != "" {
		ip.textarea.SetValue(completion)
		ip.textarea.CursorEnd()
	}
}

// cycleNextCompletion moves to the next completion
func (ip *inputPaneImpl) cycleNextCompletion() {
	if len(ip.completions) == 0 {
		return
	}

	ip.completionIndex++
	if ip.completionIndex >= len(ip.completions) {
		ip.completionIndex = 0
	}

	// Update input with current completion
	if completion := ip.GetCurrentCompletion(); completion != "" {
		ip.textarea.SetValue(completion)
		ip.textarea.CursorEnd()
	}
}

// View renders the input pane
func (ip *inputPaneImpl) View() string {
	// Choose style based on current mode
	var style lipgloss.Style
	var modeIndicator string

	if ip.showingCompletions {
		style = ip.completionStyle
		modeIndicator = "⚡ Completions"
	} else if ip.historyIndex < len(ip.history) {
		style = ip.historyStyle
		modeIndicator = "📜 History"
	} else {
		style = ip.normalStyle
	}

	// Get textarea view
	textareaView := ip.textarea.View()

	// Add completion popup if showing
	if ip.showingCompletions {
		completionView := ip.renderCompletions()
		textareaView = lipgloss.JoinVertical(lipgloss.Left, completionView, textareaView)
	}

	// Add mode indicator if present
	if modeIndicator != "" {
		indicatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
		indicator := indicatorStyle.Render(modeIndicator)
		textareaView = lipgloss.JoinVertical(lipgloss.Left, indicator, textareaView)
	}

	// Apply final styling
	rect := ip.GetRect()
	return style.Width(rect.Width).Height(rect.Height).Render(textareaView)
}

// renderCompletions renders the completion popup
func (ip *inputPaneImpl) renderCompletions() string {
	if len(ip.completions) == 0 {
		return ""
	}

	var items []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)
	header := headerStyle.Render("⚔️ Guild Suggestions")
	items = append(items, header)

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	separator := separatorStyle.Render(strings.Repeat("─", 25))
	items = append(items, separator)

	// Completion items
	for i, result := range ip.completions {
		icon := getCompletionIcon(result.Metadata["type"])

		var nameStyle, descStyle lipgloss.Style

		if i == ip.completionIndex {
			// Selected item
			nameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63"))
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("254")).
				Background(lipgloss.Color("63"))
		} else {
			// Unselected items
			nameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141"))
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))
		}

		name := nameStyle.Render(result.Content)
		description := descStyle.Render(result.Metadata["description"])

		var itemLine string
		if i == ip.completionIndex {
			itemLine = fmt.Sprintf("⚡ %s %s  %s", icon, name, description)
		} else {
			itemLine = fmt.Sprintf("  %s %s  %s", icon, name, description)
		}

		items = append(items, itemLine)
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	footer := footerStyle.Render(fmt.Sprintf("⚡ %d of %d suggestions",
		ip.completionIndex+1, len(ip.completions)))
	items = append(items, footer)

	// Create popup
	popup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Background(lipgloss.Color("0")).
		Padding(0, 1).
		Render(strings.Join(items, "\n"))

	return popup
}

// getCompletionIcon returns an icon for the completion type
func getCompletionIcon(completionType string) string {
	switch completionType {
	case "command":
		return "💻"
	case "agent":
		return "🤖"
	case "tool":
		return "🔨"
	case "file":
		return "📁"
	case "variable":
		return "📝"
	default:
		return "⭐"
	}
}

// GetStats returns statistics about the input pane
func (ip *inputPaneImpl) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["current_value"] = ip.GetValue()
	stats["value_length"] = len(ip.GetValue())
	stats["multiline_enabled"] = ip.multilineEnabled
	stats["showing_completions"] = ip.showingCompletions
	stats["completion_count"] = len(ip.completions)
	stats["completion_index"] = ip.completionIndex
	stats["history_count"] = len(ip.history)
	stats["history_index"] = ip.historyIndex
	stats["placeholder"] = ip.placeholder

	return stats
}

// AddToHistory adds a command to the history
func (ip *inputPaneImpl) AddToHistory(command string) {
	if command == "" {
		return
	}

	// Don't add duplicates
	if len(ip.history) > 0 && ip.history[len(ip.history)-1] == command {
		return
	}

	ip.history = append(ip.history, command)
	ip.historyIndex = len(ip.history)

	// Limit history size
	maxHistory := 1000
	if len(ip.history) > maxHistory {
		ip.history = ip.history[len(ip.history)-maxHistory:]
		ip.historyIndex = len(ip.history)
	}
}

// ClearHistory clears the command history
func (ip *inputPaneImpl) ClearHistory() {
	ip.history = make([]string, 0)
	ip.historyIndex = 0
	ip.originalValue = ""
}

// GetHistorySize returns the number of items in history
func (ip *inputPaneImpl) GetHistorySize() int {
	return len(ip.history)
}
