// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guild-ventures/guild-core/internal/setup"
	"github.com/guild-ventures/guild-core/pkg/config"
)

// WizardInterface defines the interface for the setup wizard
type WizardInterface interface {
	DetectProviders(ctx context.Context) ([]setup.DetectedProvider, error)
	ConfigureProvider(ctx context.Context, provider setup.DetectedProvider) (*setup.ConfiguredProvider, error)
	CreateAgents(ctx context.Context, providers []setup.ConfiguredProvider) ([]config.AgentConfig, error)
	SaveConfiguration(ctx context.Context, providers []setup.ConfiguredProvider, agents []config.AgentConfig) error
}

// Wizard states using medieval Guild terminology
type WizardState int

const (
	stateWelcome WizardState = iota
	stateProviderDetection
	stateProviderSelection
	stateModelSelection
	stateAgentCreation
	stateProgress
	stateCompletion
	stateError
)

// WizardTUIModel represents the Bubble Tea model for the setup wizard
type WizardTUIModel struct {
	// Core dependencies
	ctx    context.Context
	wizard WizardInterface

	// Current state
	state       WizardState
	currentStep int
	totalSteps  int

	// UI Components
	list        list.Model
	textInput   textinput.Model
	progressBar progress.Model
	help        help.Model
	keys        WizardKeyMap

	// Data
	detectedProviders    []setup.DetectedProvider
	selectedProviders    []setup.DetectedProvider
	configuredProviders  []setup.ConfiguredProvider
	createdAgents        []config.AgentConfig
	currentProvider      *setup.DetectedProvider
	currentProviderIndex int

	// UI State
	width         int
	height        int
	errorMessage  string
	statusMessage string

	// Progress tracking
	progressComplete bool
	progressCurrent  int
	progressMax      int
}

// WizardKeyMap defines key bindings for the wizard
type WizardKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Space    key.Binding
	Escape   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Quit     key.Binding
	Help     key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k WizardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k WizardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Space},
		{k.Tab, k.ShiftTab, k.Escape},
		{k.Help, k.Quit},
	}
}

// DefaultWizardKeyMap returns the default key bindings
func DefaultWizardKeyMap() WizardKeyMap {
	return WizardKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/continue"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle selection"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// Messages
type (
	// providerDetectedMsg signals provider detection completion
	providerDetectedMsg struct {
		providers []setup.DetectedProvider
		err       error
	}

	// providerConfiguredMsg signals provider configuration completion
	providerConfiguredMsg struct {
		provider setup.ConfiguredProvider
		err      error
	}

	// agentsCreatedMsg signals agent creation completion
	agentsCreatedMsg struct {
		agents []config.AgentConfig
		err    error
	}

	// progressMsg updates progress
	progressMsg struct {
		current int
		max     int
		message string
	}

	// configSavedMsg indicates configuration save result
	configSavedMsg struct {
		err error
	}

	// completeMsg signals wizard completion
	completeMsg struct{}

	// errorMsg signals an error occurred
	errorMsg struct {
		err error
	}
)

// NewWizardTUIModel creates a new Bubble Tea model for the setup wizard
func NewWizardTUIModel(ctx context.Context, wizard WizardInterface) *WizardTUIModel {
	// Initialize list component
	listItems := []list.Item{}
	l := list.New(listItems, NewProviderDelegate(), 0, 0)
	l.Title = "🏰 Guild Setup Wizard"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter your choice..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Initialize progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	// Initialize help
	h := help.New()
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.ShortDesc = helpDescStyle

	return &WizardTUIModel{
		ctx:         ctx,
		wizard:      wizard,
		state:       stateWelcome,
		totalSteps:  6, // Welcome, Detection, Selection, Models, Agents, Complete
		list:        l,
		textInput:   ti,
		progressBar: prog,
		help:        h,
		keys:        DefaultWizardKeyMap(),
	}
}

// Init initializes the wizard model
func (m *WizardTUIModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.detectProviders(),
	)
}

// Update handles messages and state transitions
func (m *WizardTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 8)
		m.progressBar.Width = msg.Width - 10
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case providerDetectedMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMessage = fmt.Sprintf("Failed to detect providers: %v", msg.err)
			return m, nil
		}
		m.detectedProviders = msg.providers
		m.state = stateProviderSelection
		m.currentStep = 2
		return m, m.updateProviderList()

	case providerConfiguredMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMessage = fmt.Sprintf("Failed to configure provider: %v", msg.err)
			return m, nil
		}
		m.configuredProviders = append(m.configuredProviders, msg.provider)
		return m, m.handleNextProvider()

	case agentsCreatedMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMessage = fmt.Sprintf("Failed to create agents: %v", msg.err)
			return m, nil
		}
		m.createdAgents = msg.agents
		// Save configuration before marking as complete
		return m, m.saveConfiguration()

	case progressMsg:
		m.progressCurrent = msg.current
		m.progressMax = msg.max
		m.statusMessage = msg.message
		if msg.current >= msg.max {
			m.progressComplete = true
		}
		return m, nil

	case configSavedMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMessage = fmt.Sprintf("Failed to save configuration: %v", msg.err)
			return m, nil
		}
		m.state = stateCompletion
		m.currentStep = 6
		return m, nil

	case completeMsg:
		// Wizard is complete, signal to exit
		return m, tea.Quit
	}

	// Update sub-components
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the current state of the wizard
func (m *WizardTUIModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	switch m.state {
	case stateWelcome:
		return m.renderWelcome()
	case stateProviderDetection:
		return m.renderProviderDetection()
	case stateProviderSelection:
		return m.renderProviderSelection()
	case stateModelSelection:
		return m.renderModelSelection()
	case stateAgentCreation:
		return m.renderAgentCreation()
	case stateProgress:
		return m.renderProgress()
	case stateCompletion:
		return m.renderCompletion()
	case stateError:
		return m.renderError()
	default:
		return "Unknown state"
	}
}

// Styling
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Background(lipgloss.Color("#282828")).
			Padding(0, 1).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Margin(1, 0)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4757")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA726")).
			Bold(true)

	paginationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Margin(1, 0)
)

// Helper methods will be implemented in the next part...
