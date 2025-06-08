package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Colors inspired by medieval guild themes
var (
	burgundy  = lipgloss.Color("#8B0000") // Guild Master's robes
	parchment = lipgloss.Color("#F5F5DC") // Guild records
	gold      = lipgloss.Color("#D4AF37") // Guild seal
	ivory     = lipgloss.Color("#FFFFF0") // Parchment text
	emerald   = lipgloss.Color("#2E8B57") // Success states
	// Unused colors removed - can be re-added when needed
	// darkOak   = lipgloss.Color("#3E2723") // Guild hall panels
	// ruby      = lipgloss.Color("#9B111E") // Error states
)

// UI Styles for the Guild Hall demo
var (
	// Main container style (Guild Hall)
	hallStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(burgundy).
			Padding(1, 2)

	// Section container style (Guild Chamber)
	chamberStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(gold).
			Padding(1, 2)

	// Title style
	titleStyle = lipgloss.NewStyle().
			Foreground(gold).
			Bold(true).
			Padding(0, 1)

	// Normal text style
	textStyle = lipgloss.NewStyle().
			Foreground(ivory)

	// Selected text style (unused for now)
	// selectedStyle = lipgloss.NewStyle().
	//		Foreground(darkOak).
	//		Background(parchment).
	//		Bold(true)

	// Status style
	statusStyle = lipgloss.NewStyle().
			Foreground(parchment).
			Background(burgundy).
			Padding(0, 1)
)

// Item represents an item in our demo list
type item struct {
	title       string
	description string
	status      string
}

// FilterValue implements the list.Item interface
func (i item) FilterValue() string { return i.title }

// Title returns the item's title
func (i item) Title() string { return i.title }

// Description returns the item's description
func (i item) Description() string { return i.description }

// GuildHall is our main UI model
type GuildHall struct {
	objectives list.Model
	details    viewport.Model
	selected   string
	width      int
	height     int
}

// createDemoItems creates some demo items with Guild theme
func createDemoItems() []list.Item {
	return []list.Item{
		item{
			title:       "Guild Charter: Framework Development",
			description: "Establish the core framework for Guild operations",
			status:      "in_progress",
		},
		item{
			title:       "Guild Treasury: Cost Management",
			description: "Implement token tracking and optimization",
			status:      "pending",
		},
		item{
			title:       "Craftsmen Workshop: Vector Store",
			description: "Develop the vector storage system for Guild memory",
			status:      "in_progress",
		},
		item{
			title:       "Master's Directive: CLI Tools",
			description: "Create CLI tools for Guild interaction",
			status:      "completed",
		},
		item{
			title:       "Journeyman's Task: Documentation",
			description: "Write comprehensive documentation for Guild users",
			status:      "pending",
		},
	}
}

// Initialize creates our initial model
func Initialize() *GuildHall {
	// Create the list of objectives
	items := createDemoItems()
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Guild Objectives Ledger"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.TitleBar = titleStyle

	// Create the details viewport
	d := viewport.New(0, 0)
	d.Style = textStyle

	return &GuildHall{
		objectives: l,
		details:    d,
		selected:   "",
	}
}

// Init initializes the model
func (m GuildHall) Init() tea.Cmd {
	return nil
}

// Update handles UI messages
func (m GuildHall) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve room for the status bar
		statusHeight := 1
		mainHeight := m.height - statusHeight

		// Left panel gets 40% of the width
		leftWidth := m.width * 40 / 100
		rightWidth := m.width - leftWidth - 5 // 5 for padding/border

		m.objectives.SetWidth(leftWidth)
		m.objectives.SetHeight(mainHeight)

		m.details.Width = rightWidth
		m.details.Height = mainHeight

		// If we have selection, update viewport content
		if i, ok := m.objectives.SelectedItem().(item); ok {
			m.selected = i.title
			m.updateDetailsContent(i)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		// Handle list key presses
		if m.objectives.FilterState() != list.Filtering {
			prevSelection := ""
			if i, ok := m.objectives.SelectedItem().(item); ok {
				prevSelection = i.title
			}

			// Let the list handle the key press
			m.objectives, cmd = m.objectives.Update(msg)
			cmds = append(cmds, cmd)

			// If selection changed, update details
			if i, ok := m.objectives.SelectedItem().(item); ok && i.title != prevSelection {
				m.selected = i.title
				m.updateDetailsContent(i)
			}
		}

		// Let the details viewport handle scrolling
		m.details, cmd = m.details.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateDetailsContent updates the content in the details panel
func (m *GuildHall) updateDetailsContent(i item) {
	// Simulate a detailed objective content
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n",
		titleStyle.Render("📜 "+i.title),
		textStyle.Render("Description:"),
		textStyle.Render("  "+i.description),
		textStyle.Render("Status:"),
		getStatusStyle(i.status).Render("  "+i.status),
		textStyle.Render("Guild Master:"),
		textStyle.Render("  Master Craftsman Claude"),
		textStyle.Render("This objective seeks to advance the Guild's capabilities through diligent craft and dedicated artisanship. The work shall be conducted according to the ancient traditions of guild practice, with attention to detail and commitment to excellence."),
	)
	m.details.SetContent(content)
}

// getStatusStyle returns a style based on status
func getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "completed":
		return lipgloss.NewStyle().
			Foreground(emerald).
			Bold(true)
	case "in_progress":
		return lipgloss.NewStyle().
			Foreground(gold).
			Bold(true)
	case "pending":
		return lipgloss.NewStyle().
			Foreground(ivory)
	default:
		return textStyle
	}
}

// View renders the UI
func (m GuildHall) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Render the left panel (Objectives list)
	leftPanel := chamberStyle.Copy().
		Width(m.objectives.Width()).
		Height(m.objectives.Height()).
		Render(m.objectives.View())

	// Render the right panel (Details panel)
	detailsTitle := titleStyle.Render("Objective Details")
	detailsContent := m.details.View()
	rightPanel := chamberStyle.Copy().
		Width(m.details.Width).
		Height(m.details.Height).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			detailsTitle,
			detailsContent,
		))

	// Combine panels
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	// Add status bar
	statusBar := statusStyle.Copy().
		Width(m.width).
		Render("Press q to exit • arrow keys to navigate • enter to select")

	// Wrap everything in the main hall style
	return hallStyle.Copy().
		Width(m.width - hallStyle.GetHorizontalFrameSize()).
		Height(m.height - hallStyle.GetVerticalFrameSize()).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(gold).Render("🏰 Guild Hall - Objective Management Chamber"),
			mainContent,
			statusBar,
		))
}

func main() {
	// Initialize our model
	hall := Initialize()

	// Start the Bubble Tea program
	p := tea.NewProgram(hall, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
