package objective_ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers/generator"
)

// Define key mappings
type keyMap struct {
	Add        key.Binding
	Regenerate key.Binding
	Suggest    key.Binding
	Ready      key.Binding
	Preview    key.Binding
	Quit       key.Binding
	Help       key.Binding
}

// Define UI state
type Model struct {
	// Session state
	planner   *objective.Planner
	session   *objective.PlanningSession
	generator generator.LLMGenerator

	// UI components
	textarea textarea.Model
	viewport viewport.Model
	help     help.Model
	keymap   keyMap

	// UI state
	width, height int
	ready         bool
	err           error
	inputMode     bool
	previewMode   bool
	statusMsg     string

	// Content
	preview string
}

// NewModel creates a new Bubble Tea model for the objective planner
func NewModel(planner *objective.Planner, generator generator.LLMGenerator) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter context or commands (e.g., @spec/path/to/file.md)"
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Guild Objective Planner")

	help := help.New()

	km := keyMap{
		Add: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "add context"),
		),
		Regenerate: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "regenerate"),
		),
		// Other key bindings...
	}

	return Model{
		planner:   planner,
		session:   planner.GetSession(),
		generator: generator,
		textarea:  ta,
		viewport:  vp,
		help:      help,
		keymap:    km,
		inputMode: true,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}
