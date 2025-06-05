// pkg/ui/commission/view.go
package commission

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Define medieval Guild-themed styles
var (
	// Colors inspired by medieval manuscripts and guild halls
	burgundy  = lipgloss.Color("#8B0000") // Guild Master's robes
	parchment = lipgloss.Color("#F5F5DC") // Guild records
	gold      = lipgloss.Color("#D4AF37") // Guild seal
	darkOak   = lipgloss.Color("#3E2723") // Guild hall panels
	ivory     = lipgloss.Color("#FFFFF0") // Parchment text
	emerald   = lipgloss.Color("#2E8B57") // Success states
	ruby      = lipgloss.Color("#9B111E") // Error states

	// Title bar styled as a Guild banner
	bannerStyle = lipgloss.NewStyle().
		Foreground(gold).
		Background(burgundy).
		Bold(true).
		Padding(0, 1).
		Width(80)

	// Section headers like manuscript titles
	manuscriptStyle = lipgloss.NewStyle().
		Foreground(gold).
		Background(darkOak).
		Bold(true).
		Padding(0, 2)

	// Normal text like parchment scrolls
	scrollStyle = lipgloss.NewStyle().
		Foreground(ivory).
		Background(darkOak).
		Padding(0, 2)

	// Command text like a scribe's notes
	scribeStyle = lipgloss.NewStyle().
		Foreground(parchment).
		Italic(true)

	// Error message like a warning seal
	warningStyle = lipgloss.NewStyle().
		Foreground(ruby).
		Bold(true)

	// Success message like a master craftsman's approval
	approvalStyle = lipgloss.NewStyle().
		Foreground(emerald).
		Bold(true)

	// Help text like an apprentice's guide
	guideStyle = lipgloss.NewStyle().
		Foreground(gold).
		Faint(true)

	// Border styles for different components
	guildHallStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(burgundy).
		Padding(1, 2)

	chamberStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(gold).
		Padding(1, 2)

	workshopStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(darkOak).
		Padding(0, 1)
)

// View renders the UI
func (m ObjectiveChamber) View() string {
	var b strings.Builder

	// Get title based on current objective
	var titleText string
	if m.currentCommission != nil {
		titleText = fmt.Sprintf(" Guild Hall - %s ", m.currentCommission.Title)
	} else {
		titleText = " Guild Hall - Objective Chamber "
	}

	// Title banner
	title := bannerStyle.Copy().Width(m.hallWidth).Render(titleText)
	b.WriteString(title)
	b.WriteString("\n")

	// Render different views based on state
	switch m.chamberState {
	case stateViewing:
		b.WriteString(m.renderViewingState())
	case stateContext:
		b.WriteString(m.renderContextState())
	case statePreview:
		b.WriteString(m.renderPreviewState())
	case stateCommands:
		b.WriteString(m.renderCommandState())
	case stateDashboard:
		b.WriteString(m.renderDashboardState())
	case stateCreating:
		b.WriteString(m.renderCreatingState())
	}

	// Town Crier's Proclamation (Status message)
	proclamationStyle := scrollStyle
	if m.guildError != nil {
		proclamationStyle = warningStyle
		m.proclamation = "Warning: " + m.guildError.Error()
	} else if strings.Contains(m.proclamation, "success") || 
	          strings.Contains(m.proclamation, "completed") ||
	          strings.Contains(m.proclamation, "approved") {
		proclamationStyle = approvalStyle
	}
	
	// Status footer with Town Crier's message
	footer := workshopStyle.Copy().
		Width(m.hallWidth - 4).
		Render("𝕿𝖔𝖜𝖓 𝕮𝖗𝖎𝖊𝖗: " + proclamationStyle.Render(m.proclamation))
	b.WriteString("\n")
	b.WriteString(footer)

	// Apprentice's guidance (help)
	if m.helpScroll.ShowAll {
		helpView := guideStyle.Render(m.helpScroll.View(m.keymap))
		b.WriteString("\n")
		b.WriteString(helpView)
	} else {
		helpHint := guideStyle.Render("Press ? for guidance")
		b.WriteString("\n")
		b.WriteString(helpHint)
	}

	return b.String()
}

// renderViewingState renders the main objective viewing state
func (m ObjectiveChamber) renderViewingState() string {
	var content string
	
	// If we have an objective, show its details
	if m.currentCommission != nil || m.commissionPreview != "" {
		objectiveContent := m.commissionPreview
		if objectiveContent == "" && m.currentCommission != nil {
			objectiveContent = formatCommissionPreview(m.currentCommission)
		}
		
		// Render the objective in a guild chamber
		content = chamberStyle.Copy().
			Width(m.hallWidth - 4).
			Render(fmt.Sprintf(
				"%s\n\n%s",
				manuscriptStyle.Render("Objective Scroll"),
				m.viewport.View(),
			))
	} else {
		// No objective loaded, show welcome message
		content = chamberStyle.Copy().
			Width(m.hallWidth - 4).
			Render(fmt.Sprintf(
				"%s\n\n%s",
				manuscriptStyle.Render("Welcome to the Guild Hall"),
				scrollStyle.Render(
					"Press 'c' to craft a new objective\n"+
					"Press 'tab' to view existing objectives\n"+
					"Press ':' to enter command mode",
				),
			))
	}
	
	// Show current status and available actions
	status := ""
	if m.currentCommission != nil {
		status = fmt.Sprintf(
			"Status: %s | Iterations: %d | Ready: %v",
			m.currentCommission.Status,
			m.currentCommission.Iteration,
			m.readyForMaster,
		)
	}
	
	if status != "" {
		content += "\n" + status
	}
	
	return content
}

// renderContextState renders the context adding state
func (m ObjectiveChamber) renderContextState() string {
	// Show the scribe's text area for adding context
	header := manuscriptStyle.Render("The Guild Scribe's Parchment")
	instructions := scrollStyle.Render(
		"Enter context or reference documents for your objective.\n" +
		"Use @spec/path/to/file.md or @ai_docs/path/to/file.md to reference existing documents.\n" +
		"Press Ctrl+Enter to submit or Esc to cancel.",
	)
	
	scribeArea := m.scribe.View()
	
	content := chamberStyle.Copy().
		Width(m.hallWidth - 4).
		Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			header,
			instructions,
			scribeArea,
		))
	
	return content
}

// renderPreviewState renders the document preview state
func (m ObjectiveChamber) renderPreviewState() string {
	// Show the viewport with document preview
	header := manuscriptStyle.Render("Guild Document Archives")
	
	content := chamberStyle.Copy().
		Width(m.hallWidth - 4).
		Height(m.hallHeight - 10).
		Render(fmt.Sprintf(
			"%s\n\n%s",
			header,
			m.viewport.View(),
		))
	
	return content
}

// renderCommandState renders the command input state
func (m ObjectiveChamber) renderCommandState() string {
	// Show the command input for entering commands
	header := manuscriptStyle.Render("Guild Master's Command Hall")
	instructions := scrollStyle.Render(
		"Enter a command to execute:\n" +
		"  add-context \"<text>\" - Add context to the objective\n" +
		"  regenerate - Rebuild documents from current objective\n" +
		"  suggest - Request improvement suggestions\n" +
		"  ready - Mark the objective as ready\n" +
		"Press Enter to execute or Esc to cancel.",
	)
	
	commandArea := scribeStyle.Render("> " + m.parchment.View())
	
	content := chamberStyle.Copy().
		Width(m.hallWidth - 4).
		Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			header,
			instructions,
			commandArea,
		))
	
	return content
}

// renderDashboardState renders the objectives dashboard state
func (m ObjectiveChamber) renderDashboardState() string {
	// Show the list of objectives
	header := manuscriptStyle.Render("Guild Objective Ledger")
	
	ledgerView := m.ledger.View()
	if ledgerView == "" || !strings.Contains(ledgerView, "item") {
		ledgerView = scrollStyle.Render("No objectives recorded in the Guild ledger.")
	}
	
	content := chamberStyle.Copy().
		Width(m.hallWidth - 4).
		Height(m.hallHeight - 10).
		Render(fmt.Sprintf(
			"%s\n\n%s",
			header,
			ledgerView,
		))
	
	return content
}

// renderCreatingState renders the objective creation state
func (m ObjectiveChamber) renderCreatingState() string {
	// Show the textarea for creating a new objective
	header := manuscriptStyle.Render("Crafting a New Objective")
	instructions := scrollStyle.Render(
		"Describe your objective in natural language.\n" +
		"The Guild craftsmen will shape it into a proper objective structure.\n" +
		"Press Ctrl+Enter to submit or Esc to cancel.",
	)
	
	creationArea := m.scribe.View()
	
	content := chamberStyle.Copy().
		Width(m.hallWidth - 4).
		Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			header,
			instructions,
			creationArea,
		))
	
	return content
}