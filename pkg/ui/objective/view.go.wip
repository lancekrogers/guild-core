// pkg/ui/objective/view.go
package objective_ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Define styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000"))

	statusSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Render
)

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Title bar with objective info
	objectiveTitle := "No objective"
	if m.session.Objective != nil {
		objectiveTitle = m.session.Objective.Title
	}

	title := titleStyle.Render(fmt.Sprintf(" Guild Objective Planner - %s ", objectiveTitle))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Status bar with iteration count
	status := fmt.Sprintf("Status: %s | Iterations: %d",
		m.session.Objective.Status,
		m.session.Objective.Iterations)
	b.WriteString(status)
	b.WriteString("\n\n")

	// Main content area - either preview or viewport
	if m.previewMode {
		b.WriteString(m.viewport.View())
	} else {
		// Show objective details and planning status
		b.WriteString(m.formatObjectiveView())
	}

	b.WriteString("\n\n")

	// Status message (errors, success messages)
	statusMsg := m.statusMsg
	if m.err != nil {
		statusMsg = statusErrorStyle.Render("Error: " + m.err.Error())
	} else if m.statusMsg != "" {
		statusMsg = statusSuccessStyle.Render(m.statusMsg)
	}
	b.WriteString(statusMsg)
	b.WriteString("\n\n")

	// Input area
	if m.inputMode {
		b.WriteString("Enter context or command:\n")
		b.WriteString(m.textarea.View())
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(helpStyle(m.help.View(m.keymap)))

	return b.String()
}

// Helper methods for view...
func (m Model) formatObjectiveView() string {
	// Format the objective details for display
	// ...
}
