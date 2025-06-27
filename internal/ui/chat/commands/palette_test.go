// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/lancekrogers/guild/internal/ui/chat/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPaletteModel wraps CommandPalette to make it a proper Bubble Tea model for testing
type testPaletteModel struct {
	palette     *commands.CommandPalette
	width       int
	height      int
	quitting    bool
	selectedCmd *commands.PaletteCommand
	executed    bool
}

// newTestPaletteModel creates a test model with the command palette
func newTestPaletteModel() *testPaletteModel {
	return &testPaletteModel{
		palette: commands.NewCommandPalette(),
		width:   80,
		height:  24,
	}
}

// Init implements tea.Model
func (m *testPaletteModel) Init() tea.Cmd {
	// Open the palette immediately for testing
	m.palette.Open()
	m.palette.SetDimensions(m.width, m.height)
	return nil
}

// Update implements tea.Model
func (m *testPaletteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.palette.SetDimensions(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		if !m.palette.IsOpen() {
			switch msg.String() {
			case "ctrl+p", ":":
				m.palette.Open()
				return m, nil
			case "q", "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle palette keys when open
		switch msg.String() {
		case "esc":
			m.palette.Close()
			return m, nil
		case "enter":
			if selected := m.palette.GetSelectedCommand(); selected != nil {
				m.selectedCmd = selected
				m.executed = true
				m.palette.Close()
				return m, tea.Quit
			}
			return m, nil
		case "up", "k":
			m.palette.MoveUp()
			return m, nil
		case "down", "j":
			m.palette.MoveDown()
			return m, nil
		case "backspace":
			query := m.palette.GetSearchQuery()
			if len(query) > 0 {
				m.palette.UpdateSearch(query[:len(query)-1])
			}
			return m, nil
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			// Handle character input for search
			if len(msg.Runes) == 1 {
				char := string(msg.Runes[0])
				if char >= " " && char <= "~" { // Printable ASCII
					currentQuery := m.palette.GetSearchQuery()
					m.palette.UpdateSearch(currentQuery + char)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// View implements tea.Model
func (m *testPaletteModel) View() string {
	if m.quitting {
		if m.executed && m.selectedCmd != nil {
			return "Command executed: " + m.selectedCmd.Name + "\n"
		}
		return "Goodbye!\n"
	}

	if !m.palette.IsOpen() {
		return "Press Ctrl+P or : to open command palette\nPress Q to quit\n"
	}

	return m.palette.View()
}

// GetSearchQuery returns the current search query (helper for palette access)
func (m *testPaletteModel) GetSearchQuery() string {
	return m.palette.GetSearchQuery()
}

// Unit Tests for CommandPalette Data Structure

func TestNewCommandPalette(t *testing.T) {
	palette := commands.NewCommandPalette()

	assert.NotNil(t, palette)
	assert.False(t, palette.IsOpen())

	// Should have default commands registered
	commands := palette.GetCommands()
	assert.Greater(t, len(commands), 0, "Should have default commands")

	// Check for some expected commands
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}

	assert.True(t, commandNames["Show Help"], "Should have Show Help command")
	assert.True(t, commandNames["List Agents"], "Should have List Agents command")
	assert.True(t, commandNames["Clear Chat"], "Should have Clear Chat command")
}

func TestCommandPaletteOpenClose(t *testing.T) {
	palette := commands.NewCommandPalette()

	// Initially closed
	assert.False(t, palette.IsOpen())

	// Open
	palette.Open()
	assert.True(t, palette.IsOpen())
	assert.Equal(t, "", palette.GetSearchQuery())

	// Close
	palette.Close()
	assert.False(t, palette.IsOpen())
	assert.Equal(t, "", palette.GetSearchQuery())
}

func TestCommandPaletteSearch(t *testing.T) {
	palette := commands.NewCommandPalette()
	palette.Open()

	// Initially shows all commands
	allCmds := palette.GetFilteredCommands()
	assert.Equal(t, len(palette.GetCommands()), len(allCmds))

	// Search for "help"
	palette.UpdateSearch("help")
	filtered := palette.GetFilteredCommands()
	assert.Less(t, len(filtered), len(allCmds), "Should filter commands")

	// All filtered commands should match "help"
	for _, cmd := range filtered {
		matched := false
		if fuzzyMatch(cmd.Name, "help") ||
			fuzzyMatch(cmd.Description, "help") ||
			fuzzyMatch(cmd.Category, "help") ||
			fuzzyMatch(cmd.Shortcut, "help") {
			matched = true
		}
		assert.True(t, matched, "Command %s should match 'help'", cmd.Name)
	}

	// Clear search
	palette.UpdateSearch("")
	cleared := palette.GetFilteredCommands()
	assert.Equal(t, len(palette.GetCommands()), len(cleared))
}

func TestCommandPaletteNavigation(t *testing.T) {
	palette := commands.NewCommandPalette()
	palette.Open()

	// Should start at index 0
	selected := palette.GetSelectedCommand()
	require.NotNil(t, selected)

	firstCmd := palette.GetFilteredCommands()[0]
	assert.Equal(t, firstCmd.Name, selected.Name)

	// Move down
	palette.MoveDown()
	selected = palette.GetSelectedCommand()
	require.NotNil(t, selected)

	secondCmd := palette.GetFilteredCommands()[1]
	assert.Equal(t, secondCmd.Name, selected.Name)

	// Move up (should wrap to first)
	palette.MoveUp()
	selected = palette.GetSelectedCommand()
	require.NotNil(t, selected)
	assert.Equal(t, firstCmd.Name, selected.Name)
}

func TestCommandPaletteWrapping(t *testing.T) {
	palette := commands.NewCommandPalette()
	palette.Open()

	commands := palette.GetFilteredCommands()
	if len(commands) < 2 {
		t.Skip("Need at least 2 commands for wrapping test")
	}

	// Move up from first position should wrap to last
	palette.MoveUp()
	selected := palette.GetSelectedCommand()
	require.NotNil(t, selected)

	lastCmd := commands[len(commands)-1]
	assert.Equal(t, lastCmd.Name, selected.Name)

	// Move down should wrap to first
	palette.MoveDown()
	selected = palette.GetSelectedCommand()
	require.NotNil(t, selected)

	firstCmd := commands[0]
	assert.Equal(t, firstCmd.Name, selected.Name)
}

// Integration Tests using teatest

func TestCommandPalette_Integration_BasicOpen(t *testing.T) {
	model := newTestPaletteModel()

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for palette to open and render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "🔍 Search:") && contains(bts, "Show Help")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Send escape to close palette first, then q to quit the model
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait a moment for the escape to be processed
	time.Sleep(100 * time.Millisecond)

	// Send 'q' to quit the application
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify final output
	finalOutput := tm.FinalOutput(t)
	outputBytes, err := io.ReadAll(finalOutput)
	require.NoError(t, err)

	t.Logf("Final output: %s", string(outputBytes))
}

func TestCommandPalette_Integration_SearchAndFilter(t *testing.T) {
	model := newTestPaletteModel()

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "🔍 Search:")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Type "help" to search
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})

	// Wait for search results
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "help") && contains(bts, "Show Help")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Exit - first escape to close palette, then 'q' to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	finalOutput := tm.FinalOutput(t)
	outputBytes, err := io.ReadAll(finalOutput)
	require.NoError(t, err)

	t.Logf("Search test output: %s", string(outputBytes))
}

func TestCommandPalette_Integration_NavigationAndSelection(t *testing.T) {
	model := newTestPaletteModel()

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "🔍 Search:")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Navigate down a few items
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Wait a bit for navigation to process
	time.Sleep(100 * time.Millisecond)

	// Select current item
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The enter key should execute the command and quit automatically
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify command was executed
	finalModel := tm.FinalModel(t)
	if testModel, ok := finalModel.(*testPaletteModel); ok {
		assert.True(t, testModel.executed, "Command should have been executed")
		assert.NotNil(t, testModel.selectedCmd, "A command should have been selected")
	}
}

func TestCommandPalette_Integration_BackspaceInSearch(t *testing.T) {
	model := newTestPaletteModel()

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "🔍 Search:")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Type some text
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Wait for search to show "agen"
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "agen")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Use backspace to remove characters
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})

	// Wait for search to show "ag"
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "ag") && !contains(bts, "agen")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Exit - first escape to close palette, then 'q' to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCommandPalette_Integration_EmptySearchResults(t *testing.T) {
	model := newTestPaletteModel()

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "🔍 Search:")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Type a search that won't match anything
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})

	// Wait for "No commands found"
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "No commands found")
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(2*time.Second),
	)

	// Exit - first escape to close palette, then 'q' to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// Benchmarks for Performance Testing

func BenchmarkCommandPaletteSearch(b *testing.B) {
	palette := commands.NewCommandPalette()
	palette.Open()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		palette.UpdateSearch("help")
		palette.UpdateSearch("agent")
		palette.UpdateSearch("tool")
		palette.UpdateSearch("")
	}
}

func BenchmarkCommandPaletteNavigation(b *testing.B) {
	palette := commands.NewCommandPalette()
	palette.Open()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		palette.MoveDown()
		palette.MoveDown()
		palette.MoveUp()
		palette.MoveUp()
	}
}

func BenchmarkCommandPaletteView(b *testing.B) {
	palette := commands.NewCommandPalette()
	palette.Open()
	palette.SetDimensions(80, 24)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = palette.View()
	}
}

// Helper functions

// contains checks if bytes contain a string (case-insensitive)
func contains(data []byte, s string) bool {
	return bytes.Contains(bytes.ToLower(data), bytes.ToLower([]byte(s)))
}

// fuzzyMatch helper function for testing (duplicated from implementation)
func fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternIdx := 0
	for _, ch := range text {
		if patternIdx < len(pattern) && ch == rune(pattern[patternIdx]) {
			patternIdx++
		}
	}

	return patternIdx == len(pattern)
}
