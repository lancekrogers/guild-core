package chat

import (
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
)

// handleUpKey handles up arrow for history navigation or completion navigation
func (m ChatModel) handleUpKey() (ChatModel, tea.Cmd) {
	// If showing completion popup, navigate up in completion list
	if m.showingCompletion && len(m.completionResults) > 0 {
		if m.completionIndex > 0 {
			m.completionIndex--
		} else {
			m.completionIndex = len(m.completionResults) - 1 // Wrap to bottom
		}
		return m, nil
	}

	// Enter history mode if not already in it
	// TODO: Implement history mode when fields are added

	// Navigate command history
	if previousCommand := m.history.Previous(); previousCommand != "" {
		m.input.SetValue(previousCommand)
		m.input.CursorEnd()
	}

	return m, nil
}

// handleDownKey handles down arrow for history navigation or completion navigation
func (m ChatModel) handleDownKey() (ChatModel, tea.Cmd) {
	// If showing completion popup, navigate down in completion list
	if m.showingCompletion && len(m.completionResults) > 0 {
		if m.completionIndex < len(m.completionResults)-1 {
			m.completionIndex++
		} else {
			m.completionIndex = 0 // Wrap to top
		}
		return m, nil
	}

	// Enter history mode if not already in it
	// TODO: Implement history mode

	// Navigate command history
	if nextCommand := m.history.Next(); nextCommand != "" {
		m.input.SetValue(nextCommand)
		m.input.CursorEnd()
	} else {
		// At the end of history, restore original input
		m.input.SetValue("") // TODO: Add originalInput field
		m.input.CursorEnd()
		// TODO: Set historyMode = false when field is added
	}

	return m, nil
}

// handleSearchHistory handles Ctrl+R for fuzzy history search
func (m ChatModel) handleSearchHistory() (ChatModel, tea.Cmd) {
	// Get current input as search term
	searchTerm := m.input.Value()

	// If input is empty, show recent commands
	if searchTerm == "" {
		recent := m.history.GetRecent(5)
		if len(recent) > 0 {
			// Show first recent command
			m.input.SetValue(recent[0])
			m.input.CursorEnd()
		}
		return m, nil
	}

	// Perform fuzzy search
	results := m.history.Search(searchTerm)
	if len(results) > 0 {
		// Set first result
		m.input.SetValue(results[0])
		m.input.CursorEnd()

		// Store as completion results for cycling
		m.completionResults = make([]CompletionResult, len(results))
		for i, result := range results {
			m.completionResults[i] = CompletionResult{
				Content:  result,
				AgentID:  "history",
				Metadata: map[string]string{"type": "history"},
			}
		}
		m.completionIndex = 0
		m.showingCompletion = true
	}

	return m, nil
}

// handleEscape handles escape key for canceling completion or search
func (m ChatModel) handleEscape() (ChatModel, tea.Cmd) {
	// Cancel completion popup
	if m.showingCompletion {
		m.showingCompletion = false
		m.completionResults = nil
		m.completionIndex = 0
		return m, nil
	}

	// Clear current input as fallback
	m.input.Reset()
	return m, nil
}

// handleTabCompletion handles tab key for completion
func (m ChatModel) handleTabCompletion() (ChatModel, tea.Cmd) {
	if m.completionEng == nil {
		// No completion engine available
		return m, nil
	}

	// Get current input
	input := m.input.Value()
	cursorPos := len(input) // Use length as approximation for cursor position

	// If already showing completions, cycle through them
	if m.showingCompletion && len(m.completionResults) > 0 {
		m.completionIndex = (m.completionIndex + 1) % len(m.completionResults)
		m.input.SetValue(m.completionResults[m.completionIndex].Content)
		m.input.CursorEnd()
		return m, nil
	}

	// Get completion suggestions
	completions := m.completionEng.Complete(input, cursorPos)
	if len(completions) == 0 {
		// No completions available - just insert tab spaces
		return m, nil
	}

	// Store completions
	m.completionResults = completions
	m.completionIndex = 0
	m.showingCompletion = true

	// If only one completion, auto-apply it
	if len(completions) == 1 {
		completion := completions[0]
		m.input.SetValue(completion.Content)
		m.input.CursorEnd()
		m.showingCompletion = false
		m.completionResults = nil
	}

	return m, nil
}

// handleCompletionKey handles keyboard input during completion mode
func (m ChatModel) handleCompletionKey(msg tea.KeyMsg) (ChatModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel completion
		m.showingCompletion = false
		m.completionResults = nil
		return m, nil

	case "enter":
		// Accept current completion
		if len(m.completionResults) > 0 {
			m.input.SetValue(m.completionResults[m.completionIndex].Content)
			m.input.CursorEnd()
		}
		m.showingCompletion = false
		m.completionResults = nil
		return m, nil

	case "tab":
		// Cycle through completions
		if len(m.completionResults) > 0 {
			m.completionIndex = (m.completionIndex + 1) % len(m.completionResults)
			// Apply current completion
			m.input.SetValue(m.completionResults[m.completionIndex].Content)
			m.input.CursorEnd()
		}
		return m, nil

	default:
		// Any other key cancels completion and processes normally
		m.showingCompletion = false
		m.completionResults = nil
		// Let the key be processed normally
		m.input, _ = m.input.Update(msg)
		return m, nil
	}
}

// handleCommandPalette handles opening and managing the command palette
func (m ChatModel) handleCommandPalette() (ChatModel, tea.Cmd) {
	if m.commandPalette == nil {
		return m, nil
	}
	
	if m.commandPalette.IsOpen() {
		// Close palette
		m.commandPalette.Close()
	} else {
		// Open palette
		m.commandPalette.Open()
		m.commandPalette.SetDimensions(m.width, m.height)
	}
	
	return m, nil
}

// handleCommandPaletteKey handles key input when command palette is open
func (m ChatModel) handleCommandPaletteKey(msg tea.KeyMsg) (ChatModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		// Close palette
		m.commandPalette.Close()
		return m, nil
		
	case "enter":
		// Execute selected command
		if cmd := m.commandPalette.GetSelectedCommand(); cmd != nil {
			// Close palette
			m.commandPalette.Close()
			
			// Execute the command by setting the input
			m.input.SetValue(cmd.Shortcut)
			m.input.CursorEnd()
			
			// Optionally auto-submit for certain commands
			if strings.HasPrefix(cmd.Shortcut, "/help") || 
			   strings.HasPrefix(cmd.Shortcut, "/status") ||
			   strings.HasPrefix(cmd.Shortcut, "/agents") {
				return m.handleSendMessage()
			}
		}
		return m, nil
		
	case "up", "k", "shift+tab":
		// Move up in list
		m.commandPalette.MoveUp()
		return m, nil
		
	case "down", "j", "tab":
		// Move down in list
		m.commandPalette.MoveDown()
		return m, nil
		
	default:
		// Update search query with typed characters
		if len(msg.String()) == 1 || msg.String() == "backspace" {
			currentQuery := ""
			// Get current search query (we need to track this in the model)
			// For now, just handle character input
			if len(msg.String()) == 1 {
				m.commandPalette.UpdateSearch(currentQuery + msg.String())
			}
		}
		return m, nil
	}
}
