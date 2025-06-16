// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// VimMode represents the current vim mode
type VimMode int

const (
	ModeInsert VimMode = iota
	ModeNormal
	ModeVisual
	ModeCommand
)

// Vim mode colors
var (
	normalModeColor  = lipgloss.Color("240")
	insertModeColor  = lipgloss.Color("32")
	visualModeColor  = lipgloss.Color("33")
	commandModeColor = lipgloss.Color("202")

	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)
)

// String returns the string representation of the vim mode
func (m VimMode) String() string {
	switch m {
	case ModeInsert:
		return "INSERT"
	case ModeNormal:
		return "NORMAL"
	case ModeVisual:
		return "VISUAL"
	case ModeCommand:
		return "COMMAND"
	default:
		return "UNKNOWN"
	}
}

// VimState manages vim mode state and key handling
type VimState struct {
	Mode           VimMode
	CommandBuffer  string
	VisualStartRow int
	VisualStartCol int
	RepeatCount    int
	LastCommand    string
}

// NewVimState creates a new vim state starting in insert mode
func NewVimState() *VimState {
	return &VimState{
		Mode: ModeInsert,
	}
}

// vimKeyMap extends the chat key map with vim-specific bindings
type vimKeyMap struct {
	// Normal mode navigation
	MoveLeft      key.Binding
	MoveDown      key.Binding
	MoveUp        key.Binding
	MoveRight     key.Binding
	MoveWordNext  key.Binding
	MoveWordPrev  key.Binding
	MoveWordEnd   key.Binding
	MoveLineStart key.Binding
	MoveLineEnd   key.Binding
	MoveFileStart key.Binding
	MoveFileEnd   key.Binding

	// Mode switching
	EnterInsert       key.Binding
	EnterInsertAppend key.Binding
	EnterInsertLine   key.Binding
	EnterVisual       key.Binding
	EnterCommand      key.Binding
	ExitMode          key.Binding

	// Scrolling
	ScrollHalfUp   key.Binding
	ScrollHalfDown key.Binding

	// Search
	SearchForward  key.Binding
	SearchBackward key.Binding
	SearchNext     key.Binding
	SearchPrev     key.Binding

	// Copy/paste
	Yank   key.Binding
	Paste  key.Binding
	Delete key.Binding
}

// newVimKeyMap creates vim-specific key bindings
func newVimKeyMap() vimKeyMap {
	return vimKeyMap{
		// Normal mode navigation
		MoveLeft: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "move left"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "move down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "move up"),
		),
		MoveRight: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "move right"),
		),
		MoveWordNext: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "next word"),
		),
		MoveWordPrev: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "previous word"),
		),
		MoveWordEnd: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "end of word"),
		),
		MoveLineStart: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "start of line"),
		),
		MoveLineEnd: key.NewBinding(
			key.WithKeys("$"),
			key.WithHelp("$", "end of line"),
		),
		MoveFileStart: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "start of file"),
		),
		MoveFileEnd: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "end of file"),
		),

		// Mode switching
		EnterInsert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert mode"),
		),
		EnterInsertAppend: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "append mode"),
		),
		EnterInsertLine: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "insert new line"),
		),
		EnterVisual: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual mode"),
		),
		EnterCommand: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command mode"),
		),
		ExitMode: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal mode"),
		),

		// Scrolling
		ScrollHalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		ScrollHalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),

		// Search
		SearchForward: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search forward"),
		),
		SearchBackward: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "search backward"),
		),
		SearchNext: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		SearchPrev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous match"),
		),

		// Copy/paste
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank/copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "paste"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
	}
}

// HandleVimKey processes a key press based on the current vim mode
func (v *VimState) HandleVimKey(msg tea.KeyMsg, m *ChatModel) (tea.Model, tea.Cmd) {
	switch v.Mode {
	case ModeNormal:
		return v.handleNormalMode(msg, m)
	case ModeInsert:
		return v.handleInsertMode(msg, m)
	case ModeVisual:
		return v.handleVisualMode(msg, m)
	case ModeCommand:
		return v.handleCommandMode(msg, m)
	}
	return m, nil
}

// handleNormalMode processes keys in normal mode
func (v *VimState) handleNormalMode(msg tea.KeyMsg, m *ChatModel) (tea.Model, tea.Cmd) {
	// Check for repeat count (e.g., 5j to move down 5 lines)
	if msg.String() >= "1" && msg.String() <= "9" {
		if v.RepeatCount == 0 {
			v.RepeatCount = int(msg.String()[0] - '0')
		} else {
			v.RepeatCount = v.RepeatCount*10 + int(msg.String()[0]-'0')
		}
		return m, nil
	}

	// Get repeat count (default to 1)
	count := v.RepeatCount
	if count == 0 {
		count = 1
	}
	v.RepeatCount = 0 // Reset after use

	vimKeys := m.vimKeys

	switch {
	// Navigation
	case key.Matches(msg, vimKeys.MoveLeft):
		m.moveCursorLeft(count)
	case key.Matches(msg, vimKeys.MoveDown):
		m.scrollDown(count)
	case key.Matches(msg, vimKeys.MoveUp):
		m.scrollUp(count)
	case key.Matches(msg, vimKeys.MoveRight):
		m.moveCursorRight(count)
	case key.Matches(msg, vimKeys.MoveWordNext):
		m.moveWordForward(count)
	case key.Matches(msg, vimKeys.MoveWordPrev):
		m.moveWordBackward(count)
	case key.Matches(msg, vimKeys.MoveLineStart):
		m.moveCursorLineStart()
	case key.Matches(msg, vimKeys.MoveLineEnd):
		m.moveCursorLineEnd()
	case msg.String() == "g":
		// Handle gg command
		if v.LastCommand == "g" {
			m.scrollToTop()
			v.LastCommand = ""
		} else {
			v.LastCommand = "g"
		}
	case key.Matches(msg, vimKeys.MoveFileEnd):
		m.scrollToBottom()

	// Scrolling
	case key.Matches(msg, vimKeys.ScrollHalfUp):
		m.scrollHalfPageUp()
	case key.Matches(msg, vimKeys.ScrollHalfDown):
		m.scrollHalfPageDown()

	// Mode switching
	case key.Matches(msg, vimKeys.EnterInsert):
		v.Mode = ModeInsert
		m.input.Focus()
	case key.Matches(msg, vimKeys.EnterInsertAppend):
		v.Mode = ModeInsert
		m.moveCursorRight(1)
		m.input.Focus()
	case key.Matches(msg, vimKeys.EnterInsertLine):
		v.Mode = ModeInsert
		m.insertNewLine()
		m.input.Focus()
	case key.Matches(msg, vimKeys.EnterVisual):
		v.Mode = ModeVisual
		v.VisualStartRow = m.viewport.YOffset + m.cursorY
		v.VisualStartCol = m.cursorX
	case key.Matches(msg, vimKeys.EnterCommand):
		v.Mode = ModeCommand
		v.CommandBuffer = ":"

	// Search
	case key.Matches(msg, vimKeys.SearchForward):
		v.Mode = ModeCommand
		v.CommandBuffer = "/"
	case key.Matches(msg, vimKeys.SearchBackward):
		v.Mode = ModeCommand
		v.CommandBuffer = "?"

	default:
		// Reset last command if unrecognized
		if v.LastCommand != "" && msg.String() != "g" {
			v.LastCommand = ""
		}
	}

	return m, nil
}

// handleInsertMode processes keys in insert mode
func (v *VimState) handleInsertMode(msg tea.KeyMsg, m *ChatModel) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.vimKeys.ExitMode) {
		v.Mode = ModeNormal
		m.input.Blur()
		return m, nil
	}

	// Pass through to normal text input handling
	return m, nil
}

// handleVisualMode processes keys in visual mode
func (v *VimState) handleVisualMode(msg tea.KeyMsg, m *ChatModel) (tea.Model, tea.Cmd) {
	vimKeys := m.vimKeys

	switch {
	case key.Matches(msg, vimKeys.ExitMode):
		v.Mode = ModeNormal
		m.clearSelection()
	case key.Matches(msg, vimKeys.Yank):
		m.yankSelection()
		v.Mode = ModeNormal
	case key.Matches(msg, vimKeys.Delete):
		m.deleteSelection()
		v.Mode = ModeNormal
	default:
		// Allow navigation in visual mode
		return v.handleNormalMode(msg, m)
	}

	return m, nil
}

// handleCommandMode processes keys in command mode
func (v *VimState) handleCommandMode(msg tea.KeyMsg, m *ChatModel) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		v.Mode = ModeNormal
		v.CommandBuffer = ""
		return m, nil

	case tea.KeyEnter:
		cmd := v.CommandBuffer
		v.CommandBuffer = ""
		v.Mode = ModeNormal

		// Execute command
		return m, v.executeCommand(cmd, m)

	case tea.KeyBackspace:
		if len(v.CommandBuffer) > 1 {
			v.CommandBuffer = v.CommandBuffer[:len(v.CommandBuffer)-1]
		} else {
			v.Mode = ModeNormal
			v.CommandBuffer = ""
		}

	default:
		// Add character to command buffer
		v.CommandBuffer += msg.String()
	}

	return m, nil
}

// executeCommand executes a vim command
func (v *VimState) executeCommand(cmd string, m *ChatModel) tea.Cmd {
	if len(cmd) == 0 {
		return nil
	}

	// Remove the command prefix (: or / or ?)
	prefix := cmd[0]
	cmd = cmd[1:]

	switch prefix {
	case ':':
		// Handle ex commands
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return nil
		}

		switch parts[0] {
		case "q", "quit":
			return tea.Quit
		case "w", "write":
			return m.saveChat()
		case "wq":
			m.saveChat()
			return tea.Quit
		case "set":
			if len(parts) > 1 {
				return v.handleSetCommand(parts[1:], m)
			}
		}

	case '/':
		// Forward search
		return m.searchForward(cmd)

	case '?':
		// Backward search
		return m.searchBackward(cmd)
	}

	return nil
}

// handleSetCommand handles :set commands
func (v *VimState) handleSetCommand(args []string, m *ChatModel) tea.Cmd {
	for _, arg := range args {
		switch arg {
		case "number", "nu":
			m.showLineNumbers = true
		case "nonumber", "nonu":
			m.showLineNumbers = false
		case "wrap":
			m.wrapLines = true
		case "nowrap":
			m.wrapLines = false
		}
	}
	return nil
}

// GetModeIndicator returns a styled mode indicator for the status line
func (v *VimState) GetModeIndicator() string {
	style := modeStyle

	switch v.Mode {
	case ModeInsert:
		style = style.Background(insertModeColor)
	case ModeNormal:
		style = style.Background(normalModeColor)
	case ModeVisual:
		style = style.Background(visualModeColor)
	case ModeCommand:
		return style.Background(commandModeColor).Render(v.CommandBuffer)
	}

	return style.Render(fmt.Sprintf(" %s ", v.Mode.String()))
}

// Helper methods implemented for ChatModel

func (m *ChatModel) moveCursorLeft(count int) {
	// Move cursor left in the input text area
	for i := 0; i < count; i++ {
		lineInfo := m.input.LineInfo()
		if lineInfo.ColumnOffset > 0 {
			m.input.SetCursor(lineInfo.ColumnOffset - 1)
		}
	}
}

func (m *ChatModel) moveCursorRight(count int) {
	// Move cursor right in the input text area
	for i := 0; i < count; i++ {
		lineInfo := m.input.LineInfo()
		text := m.input.Value()
		if lineInfo.ColumnOffset < len(text) {
			m.input.SetCursor(lineInfo.ColumnOffset + 1)
		}
	}
}

func (m *ChatModel) moveWordForward(count int) {
	// Move cursor to next word boundary
	text := m.input.Value()
	lineInfo := m.input.LineInfo()
	pos := lineInfo.ColumnOffset

	for i := 0; i < count; i++ {
		// Find next word boundary
		for pos < len(text) && text[pos] != ' ' && text[pos] != '\n' && text[pos] != '\t' {
			pos++
		}
		// Skip whitespace
		for pos < len(text) && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\t') {
			pos++
		}
	}

	m.input.SetCursor(pos)
}

func (m *ChatModel) moveWordBackward(count int) {
	// Move cursor to previous word boundary
	text := m.input.Value()
	lineInfo := m.input.LineInfo()
	pos := lineInfo.ColumnOffset

	for i := 0; i < count; i++ {
		if pos > 0 {
			pos--
		}
		// Skip current word
		for pos > 0 && text[pos] != ' ' && text[pos] != '\n' && text[pos] != '\t' {
			pos--
		}
		// Skip whitespace
		for pos > 0 && (text[pos] == ' ' || text[pos] == '\n' || text[pos] == '\t') {
			pos--
		}
		// Go to start of word
		for pos > 0 && text[pos-1] != ' ' && text[pos-1] != '\n' && text[pos-1] != '\t' {
			pos--
		}
	}

	m.input.SetCursor(pos)
}

func (m *ChatModel) moveCursorLineStart() {
	// Move cursor to start of current line
	text := m.input.Value()
	lineInfo := m.input.LineInfo()
	pos := lineInfo.ColumnOffset

	// Find start of current line
	for pos > 0 && text[pos-1] != '\n' {
		pos--
	}

	m.input.SetCursor(pos)
}

func (m *ChatModel) moveCursorLineEnd() {
	// Move cursor to end of current line
	text := m.input.Value()
	lineInfo := m.input.LineInfo()
	pos := lineInfo.ColumnOffset

	// Find end of current line
	for pos < len(text) && text[pos] != '\n' {
		pos++
	}

	m.input.SetCursor(pos)
}

func (m *ChatModel) scrollToTop() {
	m.viewport.GotoTop()
}

func (m *ChatModel) scrollToBottom() {
	m.viewport.GotoBottom()
}

func (m *ChatModel) scrollUp(count int) {
	for i := 0; i < count; i++ {
		m.viewport.LineUp(1)
	}
}

func (m *ChatModel) scrollDown(count int) {
	for i := 0; i < count; i++ {
		m.viewport.LineDown(1)
	}
}

func (m *ChatModel) scrollHalfPageUp() {
	m.viewport.HalfViewUp()
}

func (m *ChatModel) scrollHalfPageDown() {
	m.viewport.HalfViewDown()
}

func (m *ChatModel) insertNewLine() {
	// Insert a new line at current cursor position
	currentValue := m.input.Value()
	lineInfo := m.input.LineInfo()
	cursorPos := lineInfo.ColumnOffset

	newValue := currentValue[:cursorPos] + "\n" + currentValue[cursorPos:]
	m.input.SetValue(newValue)
	m.input.SetCursor(cursorPos + 1)
}

func (m *ChatModel) clearSelection() {
	// Clear visual selection (implementation depends on visual selection system)
	// For now, just reset visual state
	if m.vimState != nil {
		m.vimState.VisualStartRow = 0
		m.vimState.VisualStartCol = 0
	}
}

func (m *ChatModel) yankSelection() {
	// Copy selected text to vim register
	// In a full vim implementation, this would work with visual selection
	// For now, we'll copy the current input text
	text := m.input.Value()
	if text != "" {
		// Add message indicating yank operation
		msg := Message{
			Type:      msgSystem,
			Content:   fmt.Sprintf("📋 Yanked %d characters", len(text)),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
		m.updateMessagesView()
	}
}

func (m *ChatModel) deleteSelection() {
	// Delete selected text
	// In a full vim implementation, this would work with visual selection
	// For now, we'll clear the input
	m.input.SetValue("")
	m.input.SetCursor(0)
}

func (m *ChatModel) saveChat() tea.Cmd {
	// Save chat session
	return func() tea.Msg {
		// In a real implementation, this would save to file
		msg := Message{
			Type:      msgSystem,
			Content:   "💾 Chat session saved",
			Timestamp: time.Now(),
		}
		return msg
	}
}

func (m *ChatModel) searchForward(pattern string) tea.Cmd {
	// Search forward in chat messages
	return func() tea.Msg {
		// Find matches in message history
		matches := 0
		for _, msg := range m.messages {
			if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(pattern)) {
				matches++
			}
		}

		responseMsg := Message{
			Type:      msgSystem,
			Content:   fmt.Sprintf("🔍 Forward search for '%s': found %d matches", pattern, matches),
			Timestamp: time.Now(),
		}
		return responseMsg
	}
}

func (m *ChatModel) searchBackward(pattern string) tea.Cmd {
	// Search backward in chat messages
	return func() tea.Msg {
		// Find matches in message history (reverse order)
		matches := 0
		for i := len(m.messages) - 1; i >= 0; i-- {
			if strings.Contains(strings.ToLower(m.messages[i].Content), strings.ToLower(pattern)) {
				matches++
			}
		}

		responseMsg := Message{
			Type:      msgSystem,
			Content:   fmt.Sprintf("🔍 Backward search for '%s': found %d matches", pattern, matches),
			Timestamp: time.Now(),
		}
		return responseMsg
	}
}
