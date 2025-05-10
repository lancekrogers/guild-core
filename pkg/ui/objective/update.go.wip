// pkg/ui/objective/update.go
package objective_ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ProcessContextMsg is sent when context processing is complete
type ProcessContextMsg struct {
	Success bool
	Error   error
}

// RegenerateMsg is sent when regeneration is complete
type RegenerateMsg struct {
	Success bool
	Error   error
}

// Update handles UI events and state changes
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle key presses
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Add):
			if m.inputMode {
				// Process the text in the textarea
				content := m.textarea.Value()
				if content != "" {
					return m, tea.Batch(
						func() tea.Msg {
							err := m.planner.AddContext(content)
							return ProcessContextMsg{
								Success: err == nil,
								Error:   err,
							}
						},
					)
				}
			}

		case key.Matches(msg, m.keymap.Regenerate):
			return m, tea.Batch(
				func() tea.Msg {
					err := m.planner.Regenerate()
					return RegenerateMsg{
						Success: err == nil,
						Error:   err,
					}
				},
			)

			// Handle other key bindings...
		}

	case tea.WindowSizeMsg:
		// Handle window resizing
		m.height = msg.Height
		m.width = msg.Width
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10 // Leave room for input & status
		m.textarea.SetWidth(msg.Width)

	case ProcessContextMsg:
		// Handle context processing results
		if !msg.Success {
			m.err = msg.Error
			m.statusMsg = "Error adding context: " + msg.Error.Error()
		} else {
			m.statusMsg = "Context added successfully."
			m.textarea.Reset()
			// Update preview content
			m.preview = m.formatSessionPreview()
		}

	case RegenerateMsg:
		// Handle regeneration results
		// Similar to ProcessContextMsg...

		// Handle other message types...
	}

	// Update sub-components
	if m.inputMode {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// Helper methods for update...
func (m Model) formatSessionPreview() string {
	// Create a formatted preview of the session state
	// Include objective details, ai_docs stubs, specs stubs
	// ...
}
