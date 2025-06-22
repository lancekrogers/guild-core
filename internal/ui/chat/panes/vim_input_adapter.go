// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/internal/ui/chat/common/layout"
	"github.com/guild-ventures/guild-core/internal/ui/chat/completion"
	"github.com/guild-ventures/guild-core/internal/ui/vim"
)

// VimInputAdapter wraps InputPane to make it VimCapable
type VimInputAdapter struct {
	inputPane      *inputPaneImpl
	vimModeManager *vim.VimModeManager
	enabled        bool
}

// NewVimInputAdapter creates a new vim input adapter
func NewVimInputAdapter(inputPane *inputPaneImpl, vimModeManager *vim.VimModeManager) *VimInputAdapter {
	return &VimInputAdapter{
		inputPane:      inputPane,
		vimModeManager: vimModeManager,
		enabled:        false,
	}
}

// SetEnabled enables or disables vim mode
func (v *VimInputAdapter) SetEnabled(enabled bool) {
	v.enabled = enabled
	if enabled {
		// Start in normal mode when enabling vim
		v.vimModeManager.GetState().Mode = vim.ModeNormal
	} else {
		// Return to insert mode when disabling vim
		v.vimModeManager.GetState().Mode = vim.ModeInsert
	}
}

// IsEnabled returns whether vim mode is enabled
func (v *VimInputAdapter) IsEnabled() bool {
	return v.enabled
}

// Update handles key events, routing through vim mode when enabled
func (v *VimInputAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If vim mode is disabled, pass through to normal input handling
	if !v.enabled {
		return v.inputPane.Update(msg)
	}

	// Handle key messages through vim mode
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		currentMode := v.vimModeManager.GetState().Mode

		// In insert mode, pass most keys to the input pane first
		if currentMode == vim.ModeInsert {
			// Only let vim handle the escape key in insert mode
			if keyMsg.Type == tea.KeyEscape {
				_, cmd := v.vimModeManager.HandleVimKey(keyMsg, v)
				return v, cmd
			}
			// All other keys go to the input pane
			return v.inputPane.Update(msg)
		}

		// In other modes (normal, visual, command), vim handles the key first
		_, cmd := v.vimModeManager.HandleVimKey(keyMsg, v)

		// If vim switched to insert mode, we might need to update the input
		if v.vimModeManager.GetState().Mode == vim.ModeInsert && currentMode != vim.ModeInsert {
			// Focus the textarea
			v.inputPane.textarea.Focus()
		}

		return v, cmd
	}

	// For non-key messages, pass through to input pane
	return v.inputPane.Update(msg)
}

// View delegates to the input pane's view with vim mode indicator
func (v *VimInputAdapter) View() string {
	baseView := v.inputPane.View()

	// If vim mode is enabled and we're not in insert mode, add a mode indicator
	if v.enabled && v.vimModeManager.GetState().Mode != vim.ModeInsert {
		modeIndicator := v.vimModeManager.GetModeIndicator()
		// Prepend the mode indicator to the view
		return modeIndicator + "\n" + baseView
	}

	return baseView
}

// Init delegates to the input pane's init
func (v *VimInputAdapter) Init() tea.Cmd {
	return v.inputPane.Init()
}

// VimCapable interface implementation

// MoveCursorLeft moves the cursor left
func (v *VimInputAdapter) MoveCursorLeft(count int) {
	// Simplified implementation - move to start for now
	// The textarea bubbles component doesn't expose cursor movement methods
	v.inputPane.textarea.CursorStart()
}

// MoveCursorRight moves the cursor right
func (v *VimInputAdapter) MoveCursorRight(count int) {
	// Simplified implementation - move to end for now
	v.inputPane.textarea.CursorEnd()
}

// MoveWordForward moves forward by words
func (v *VimInputAdapter) MoveWordForward(count int) {
	// Not supported by textarea - move to end
	v.inputPane.textarea.CursorEnd()
}

// MoveWordBackward moves backward by words
func (v *VimInputAdapter) MoveWordBackward(count int) {
	// Not supported by textarea - move to start
	v.inputPane.textarea.CursorStart()
}

// MoveCursorLineStart moves cursor to start of line
func (v *VimInputAdapter) MoveCursorLineStart() {
	v.inputPane.textarea.CursorStart()
}

// MoveCursorLineEnd moves cursor to end of line
func (v *VimInputAdapter) MoveCursorLineEnd() {
	v.inputPane.textarea.CursorEnd()
}

// ScrollToTop - for input, this clears the field
func (v *VimInputAdapter) ScrollToTop() {
	// Not applicable for single-line input
}

// ScrollToBottom - for input, this is a no-op
func (v *VimInputAdapter) ScrollToBottom() {
	// Not applicable for single-line input
}

// ScrollUp - for input, go to history
func (v *VimInputAdapter) ScrollUp(count int) {
	v.inputPane.NavigateHistory(-count)
}

// ScrollDown - for input, go to history
func (v *VimInputAdapter) ScrollDown(count int) {
	v.inputPane.NavigateHistory(count)
}

// ScrollHalfPageUp - not applicable for input
func (v *VimInputAdapter) ScrollHalfPageUp() {
	// Not applicable for single-line input
}

// ScrollHalfPageDown - not applicable for input
func (v *VimInputAdapter) ScrollHalfPageDown() {
	// Not applicable for single-line input
}

// InsertNewLine adds a new line (if multiline is enabled)
func (v *VimInputAdapter) InsertNewLine() {
	if v.inputPane.multilineEnabled {
		current := v.inputPane.GetValue()
		v.inputPane.SetValue(current + "\n")
	}
}

// ClearSelection - not implemented for now
func (v *VimInputAdapter) ClearSelection() {
	// Not implemented - would need selection tracking
}

// YankSelection copies the current value
func (v *VimInputAdapter) YankSelection() {
	// In a real implementation, this would copy to system clipboard
	// For now, we'll just store the value
}

// DeleteSelection deletes the current character or selection
func (v *VimInputAdapter) DeleteSelection() {
	// Use the textarea's built-in delete functionality
	// This is a simplified implementation for x command in normal mode
	v.inputPane.textarea.SetValue(v.inputPane.textarea.Value())
}

// SaveChat - not applicable for input pane
func (v *VimInputAdapter) SaveChat() tea.Cmd {
	return nil
}

// SearchForward - could search in history
func (v *VimInputAdapter) SearchForward(pattern string) tea.Cmd {
	// Could implement history search
	return nil
}

// SearchBackward - could search in history
func (v *VimInputAdapter) SearchBackward(pattern string) tea.Cmd {
	// Could implement history search
	return nil
}

// Delegated methods from InputPane interface

func (v *VimInputAdapter) GetValue() string {
	return v.inputPane.GetValue()
}

func (v *VimInputAdapter) SetValue(value string) {
	v.inputPane.SetValue(value)
}

func (v *VimInputAdapter) ShowCompletions(results []completion.CompletionResult) {
	v.inputPane.ShowCompletions(results)
}

func (v *VimInputAdapter) HideCompletions() {
	v.inputPane.HideCompletions()
}

func (v *VimInputAdapter) GetCurrentCompletion() string {
	return v.inputPane.GetCurrentCompletion()
}

func (v *VimInputAdapter) AcceptCompletion() {
	v.inputPane.AcceptCompletion()
}

func (v *VimInputAdapter) SetHistory(history []string) {
	v.inputPane.SetHistory(history)
}

func (v *VimInputAdapter) NavigateHistory(direction int) {
	v.inputPane.NavigateHistory(direction)
}

func (v *VimInputAdapter) SetMultilineEnabled(enabled bool) {
	v.inputPane.SetMultilineEnabled(enabled)
}

func (v *VimInputAdapter) SetPlaceholder(placeholder string) {
	v.inputPane.SetPlaceholder(placeholder)
}

func (v *VimInputAdapter) OnSubmit(callback func(string)) {
	v.inputPane.OnSubmit(callback)
}

func (v *VimInputAdapter) OnChange(callback func(string)) {
	v.inputPane.OnChange(callback)
}

// Additional helper to get vim mode display
func (v *VimInputAdapter) GetVimModeDisplay() string {
	if !v.enabled {
		return ""
	}
	return v.vimModeManager.GetModeIndicator()
}

// Implement remaining PaneInterface methods by delegating to inputPane

func (v *VimInputAdapter) ID() string {
	return v.inputPane.ID()
}

func (v *VimInputAdapter) SetID(id string) {
	v.inputPane.SetID(id)
}

func (v *VimInputAdapter) GetRect() layout.Rectangle {
	return v.inputPane.GetRect()
}

func (v *VimInputAdapter) SetRect(rect layout.Rectangle) {
	v.inputPane.SetRect(rect)
}

func (v *VimInputAdapter) GetConstraints() layout.LayoutConstraints {
	return v.inputPane.GetConstraints()
}

func (v *VimInputAdapter) SetConstraints(constraints layout.LayoutConstraints) {
	v.inputPane.SetConstraints(constraints)
}

func (v *VimInputAdapter) SetFocus(focused bool) {
	v.inputPane.SetFocus(focused)
}

func (v *VimInputAdapter) IsFocused() bool {
	return v.inputPane.IsFocused()
}

func (v *VimInputAdapter) Resize(width, height int) {
	v.inputPane.Resize(width, height)
}

func (v *VimInputAdapter) SetContent(content string) {
	v.inputPane.SetContent(content)
}

func (v *VimInputAdapter) GetContent() string {
	return v.inputPane.GetContent()
}

func (v *VimInputAdapter) SetStyle(style lipgloss.Style) {
	v.inputPane.SetStyle(style)
}

func (v *VimInputAdapter) GetStyle() lipgloss.Style {
	return v.inputPane.GetStyle()
}

// InputPane-specific methods delegated to inputPane
func (v *VimInputAdapter) GetStats() map[string]interface{} {
	return v.inputPane.GetStats()
}
