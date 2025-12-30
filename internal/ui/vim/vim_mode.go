// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package vim

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// VimMode represents the current vim mode
type VimMode int

const (
	ModeInsert VimMode = iota
	ModeNormal
	ModeVisual
	ModeCommand
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

// Vim mode colors
var (
	normalModeColor  = lipgloss.Color("240")
	insertModeColor  = lipgloss.Color("32")
	visualModeColor  = lipgloss.Color("33")
	commandModeColor = lipgloss.Color("196")
)

// Mode style
var modeStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("15")).
	Padding(0, 1)

// VimCapable defines the interface for models that can work with vim mode
type VimCapable interface {
	tea.Model
	MoveCursorLeft(count int)
	MoveCursorRight(count int)
	MoveWordForward(count int)
	MoveWordBackward(count int)
	MoveCursorLineStart()
	MoveCursorLineEnd()
	ScrollToTop()
	ScrollToBottom()
	ScrollUp(count int)
	ScrollDown(count int)
	ScrollHalfPageUp()
	ScrollHalfPageDown()
	InsertNewLine()
	ClearSelection()
	YankSelection()
	DeleteSelection()
	SaveChat() tea.Cmd
	SearchForward(pattern string) tea.Cmd
	SearchBackward(pattern string) tea.Cmd
}

// VimState holds the current vim state
type VimState struct {
	Mode          VimMode
	RepeatCount   int
	CommandBuffer string
	LastCommand   string
	ShowNumbers   bool
	WrapLines     bool
}

// NewVimState creates a new vim state in insert mode
func NewVimState() *VimState {
	return &VimState{
		Mode:        ModeInsert,
		RepeatCount: 0,
		ShowNumbers: false,
		WrapLines:   true,
	}
}

// VimModeManager manages vim mode state and key handling
type VimModeManager struct {
	state *VimState
}

// NewVimModeManager creates a new vim mode manager
func NewVimModeManager() *VimModeManager {
	return &VimModeManager{
		state: NewVimState(),
	}
}

// GetState returns the current vim state
func (v *VimModeManager) GetState() *VimState {
	return v.state
}

// GetModeIndicator returns a styled mode indicator for the status line
func (v *VimModeManager) GetModeIndicator() string {
	style := modeStyle

	switch v.state.Mode {
	case ModeInsert:
		style = style.Background(insertModeColor)
	case ModeNormal:
		style = style.Background(normalModeColor)
	case ModeVisual:
		style = style.Background(visualModeColor)
	case ModeCommand:
		return style.Background(commandModeColor).Render(v.state.CommandBuffer)
	}

	return style.Render(fmt.Sprintf(" %s ", v.state.Mode.String()))
}

// HandleVimKey processes a key press based on the current vim mode
func (v *VimModeManager) HandleVimKey(msg tea.KeyMsg, m VimCapable) (tea.Model, tea.Cmd) {
	switch v.state.Mode {
	case ModeNormal:
		return v.handleNormalMode(msg, m)
	case ModeInsert:
		return v.handleInsertMode(msg, m)
	case ModeVisual:
		return v.handleVisualMode(msg, m)
	case ModeCommand:
		return v.handleCommandMode(msg, m)
	default:
		return m, nil
	}
}

// handleNormalMode processes keys in normal mode using simple key matching
func (v *VimModeManager) handleNormalMode(msg tea.KeyMsg, m VimCapable) (tea.Model, tea.Cmd) {
	// Check for repeat count (e.g., 5j to move down 5 lines)
	if msg.String() >= "1" && msg.String() <= "9" {
		if v.state.RepeatCount == 0 {
			v.state.RepeatCount = int(msg.String()[0] - '0')
		} else {
			v.state.RepeatCount = v.state.RepeatCount*10 + int(msg.String()[0]-'0')
		}
		return m, nil
	}

	count := v.state.RepeatCount
	if count == 0 {
		count = 1
	}
	v.state.RepeatCount = 0 // Reset after use

	switch msg.String() {
	// Mode changes
	case "i":
		v.state.Mode = ModeInsert
		return m, nil
	case "v":
		v.state.Mode = ModeVisual
		return m, nil
	case ":":
		v.state.Mode = ModeCommand
		v.state.CommandBuffer = ":"
		return m, nil

	// Navigation
	case "h":
		m.MoveCursorLeft(count)
	case "j":
		m.ScrollDown(count)
	case "k":
		m.ScrollUp(count)
	case "l":
		m.MoveCursorRight(count)
	case "w":
		m.MoveWordForward(count)
	case "b":
		m.MoveWordBackward(count)
	case "0":
		m.MoveCursorLineStart()
	case "$":
		m.MoveCursorLineEnd()
	case "g":
		// Handle gg (go to top)
		if v.state.LastCommand == "g" {
			m.ScrollToTop()
			v.state.LastCommand = ""
		} else {
			v.state.LastCommand = "g"
		}
		return m, nil
	case "G":
		m.ScrollToBottom()

	// Scrolling
	case "ctrl+u":
		m.ScrollHalfPageUp()
	case "ctrl+d":
		m.ScrollHalfPageDown()

	// Editing
	case "o":
		m.InsertNewLine()
		v.state.Mode = ModeInsert
	case "x":
		m.DeleteSelection()

	default:
		v.state.LastCommand = ""
	}

	return m, nil
}

// handleInsertMode processes keys in insert mode
func (v *VimModeManager) handleInsertMode(msg tea.KeyMsg, m VimCapable) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEscape {
		v.state.Mode = ModeNormal
		return m, nil
	}
	// In insert mode, pass through to normal input handling
	return m, nil
}

// handleVisualMode processes keys in visual mode
func (v *VimModeManager) handleVisualMode(msg tea.KeyMsg, m VimCapable) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		v.state.Mode = ModeNormal
		m.ClearSelection()
		return m, nil
	}

	switch msg.String() {
	case "y":
		m.YankSelection()
		v.state.Mode = ModeNormal
		m.ClearSelection()
	case "d":
		m.DeleteSelection()
		v.state.Mode = ModeNormal
	}

	return m, nil
}

// handleCommandMode processes keys in command mode
func (v *VimModeManager) handleCommandMode(msg tea.KeyMsg, m VimCapable) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		v.state.Mode = ModeNormal
		v.state.CommandBuffer = ""
		return m, nil
	case tea.KeyEnter:
		cmd := v.executeCommand(v.state.CommandBuffer[1:], m) // Remove the ":"
		v.state.Mode = ModeNormal
		v.state.CommandBuffer = ""
		return m, cmd
	case tea.KeyBackspace:
		if len(v.state.CommandBuffer) > 1 { // Keep the ":"
			v.state.CommandBuffer = v.state.CommandBuffer[:len(v.state.CommandBuffer)-1]
		}
		return m, nil
	default:
		if msg.String() != "" {
			v.state.CommandBuffer += msg.String()
		}
		return m, nil
	}
}

// executeCommand executes a vim command
func (v *VimModeManager) executeCommand(cmd string, m VimCapable) tea.Cmd {
	if len(cmd) == 0 {
		return nil
	}

	switch cmd {
	case "w", "write":
		return m.SaveChat()
	case "q", "quit":
		return tea.Quit
	case "wq":
		return tea.Batch(m.SaveChat(), tea.Quit)
	default:
		// Handle search commands
		if len(cmd) > 0 && cmd[0] == '/' {
			pattern := cmd[1:]
			return m.SearchForward(pattern)
		}
		if len(cmd) > 0 && cmd[0] == '?' {
			pattern := cmd[1:]
			return m.SearchBackward(pattern)
		}
	}

	return nil
}
