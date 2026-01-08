// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"path/filepath"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/guild-framework/guild-core/internal/setup"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/providers"
)

// InitTUIModelV2 represents the improved initialization TUI with better practices
type InitTUIModelV2 struct {
	// Core dependencies
	ctx           context.Context
	configManager ConfigurationManager
	projectInit   ProjectInitializer
	demoGen       DemoGenerator
	validator     Validator
	daemonManager DaemonManager

	// Configuration
	config Config

	// UI state
	state  InitState
	styles *Styles

	// Input components
	inputs      map[string]textinput.Model
	activeInput string

	// Display components
	spinner  spinner.Model
	progress progress.Model
	help     help.Model
	renderer *glamour.TermRenderer

	// Data
	campaignName string
	projectName  string
	demoType     setup.DemoCommissionType
	demoOptions  []setup.DemoCommissionType
	selectedDemo int

	// Results
	validationResults  []ValidationResult
	providerResults    []providers.DetectionResult
	bestProvider       *providers.DetectionResult
	enhancedAgentCount int
	err                error

	// UI dimensions
	width  int
	height int

	useAltScreen bool
	mouseMode    tea.MouseMode
}

// Config holds initialization configuration
type Config struct {
	ProjectPath    string
	QuickMode      bool
	Force          bool
	ProviderOnly   string
	SkipValidation bool
}

// NewInitTUIModelV2 creates an improved initialization TUI
func NewInitTUIModelV2(ctx context.Context, cfg Config, deps InitDependencies, ttyAvailable bool) (*InitTUIModelV2, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during initialization").
			WithComponent("InitTUIV2").
			WithOperation("NewInitTUIModelV2")
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(cfg.ProjectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve project path").
			WithComponent("InitTUIV2").
			WithOperation("NewInitTUIModelV2").
			WithDetails("path", cfg.ProjectPath)
	}
	cfg.ProjectPath = absPath

	// Create glamour renderer for markdown based on TTY availability
	var renderer *glamour.TermRenderer

	if ttyAvailable {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithEnvironmentConfig(),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create markdown renderer").
				WithComponent("InitTUIV2").
				WithOperation("NewInitTUIModelV2")
		}
	} else {
		// For non-TTY environments, try a basic style and fall back to nil if that fails
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("notty"),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			// If even basic rendering fails, we'll work without glamour
			renderer = nil
		}
	}

	// Initialize components
	styles := NewStyles()
	inputs := createInputs(absPath)

	// Create spinner with medieval theme
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = styles.Spinner

	// Create progress bar
	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithWidth(60),
	)

	// Get demo options
	demoOptions := deps.DemoGen.GetAvailableTypes()

	useAltScreen := ttyAvailable && !cfg.QuickMode
	mouseMode := tea.MouseModeNone
	if useAltScreen {
		mouseMode = tea.MouseModeCellMotion
	}

	return &InitTUIModelV2{
		ctx:           ctx,
		configManager: deps.ConfigManager,
		projectInit:   deps.ProjectInit,
		demoGen:       deps.DemoGen,
		validator:     deps.Validator,
		daemonManager: deps.DaemonManager,
		config:        cfg,
		state:         StateWelcome,
		styles:        styles,
		inputs:        inputs,
		activeInput:   "campaign",
		spinner:       s,
		progress:      p,
		help:          help.New(),
		renderer:      renderer,
		demoOptions:   demoOptions,
		width:         80,
		height:        24,
		useAltScreen:  useAltScreen,
		mouseMode:     mouseMode,
	}, nil
}

// Init initializes the model
func (m *InitTUIModelV2) Init() tea.Cmd {
	// Check context
	if err := m.ctx.Err(); err != nil {
		m.err = gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled").
			WithComponent("InitTUIV2").
			WithOperation("Init")
		m.state = StateError
		return nil
	}

	if m.config.QuickMode {
		return m.runQuickMode()
	}

	return textinput.Blink
}

// Update handles all messages
func (m *InitTUIModelV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always check context first
	if err := m.ctx.Err(); err != nil {
		m.err = gerror.Wrap(err, gerror.ErrCodeCancelled, "operation cancelled").
			WithComponent("InitTUIV2").
			WithOperation("Update")
		m.state = StateError
		return m, nil
	}

	// Handle window resize
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.progress.SetWidth(min(msg.Width-10, 60))
		return m, nil
	}

	// Route to state-specific handlers
	switch m.state {
	case StateWelcome:
		return m.updateWelcome(msg)
	case StateCampaignInput:
		return m.updateCampaignInput(msg)
	case StateProjectInput:
		return m.updateProjectInput(msg)
	case StateConfirmation:
		return m.updateConfirmation(msg)
	case StateInitializing:
		return m.updateInitializing(msg)
	case StateDemoQuestion:
		return m.updateDemoQuestion(msg)
	case StateDemoSelection:
		return m.updateDemoSelection(msg)
	case StateValidating:
		return m.updateValidating(msg)
	case StateComplete:
		return m.updateComplete(msg)
	case StateError:
		return m.updateError(msg)
	default:
		return m, nil
	}
}

// View renders the current state
func (m *InitTUIModelV2) View() tea.View {
	// Always use lipgloss for consistent rendering
	content := m.renderCurrentState()

	// Apply container styling
	view := tea.NewView(lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	))
	view.AltScreen = m.useAltScreen
	view.MouseMode = m.mouseMode
	return view
}

// GetError returns any error that occurred
func (m *InitTUIModelV2) GetError() error {
	return m.err
}

// Helper methods

func createInputs(projectPath string) map[string]textinput.Model {
	inputs := make(map[string]textinput.Model)

	// Campaign input
	campaign := textinput.New()
	campaign.Placeholder = "guild-demo"
	campaign.CharLimit = 50
	campaign.SetWidth(40)
	campaign.Prompt = "📋 "
	inputs["campaign"] = campaign

	// Project input
	project := textinput.New()
	project.Placeholder = filepath.Base(projectPath)
	project.CharLimit = 50
	project.SetWidth(40)
	project.Prompt = "📁 "
	inputs["project"] = project

	return inputs
}

func (m *InitTUIModelV2) runQuickMode() tea.Cmd {
	return func() tea.Msg {
		// Set defaults
		m.campaignName = "guild-demo"
		m.projectName = filepath.Base(m.config.ProjectPath)
		m.state = StateInitializing

		// Start initialization
		return m.doInitialization()()
	}
}

func (m *InitTUIModelV2) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(k, keys.Enter):
			m.state = StateCampaignInput
			campaign := m.inputs["campaign"]
			cmd := campaign.Focus()
			m.inputs["campaign"] = campaign
			return m, cmd
		case key.Matches(k, keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *InitTUIModelV2) updateCampaignInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(k, keys.Enter):
			m.campaignName = m.inputs["campaign"].Value()
			if m.campaignName == "" {
				m.campaignName = m.inputs["campaign"].Placeholder
			}
			m.state = StateProjectInput
			campaign := m.inputs["campaign"]
			campaign.Blur()
			m.inputs["campaign"] = campaign
			project := m.inputs["project"]
			cmd := project.Focus()
			m.inputs["project"] = project
			m.activeInput = "project"
			return m, cmd
		case key.Matches(k, keys.Quit):
			return m, tea.Quit
		}
	}

	m.inputs["campaign"], cmd = m.inputs["campaign"].Update(msg)
	return m, cmd
}

func (m *InitTUIModelV2) renderCurrentState() string {
	switch m.state {
	case StateWelcome:
		return m.renderWelcome()
	case StateCampaignInput:
		return m.renderCampaignInput()
	case StateProjectInput:
		return m.renderProjectInput()
	case StateConfirmation:
		return m.renderConfirmation()
	case StateInitializing:
		return m.renderInitializing()
	case StateDemoQuestion:
		return m.renderDemoQuestion()
	case StateDemoSelection:
		return m.renderDemoSelection()
	case StateValidating:
		return m.renderValidating()
	case StateComplete:
		return m.renderComplete()
	case StateError:
		return m.renderError()
	default:
		return "Unknown state"
	}
}

func (m *InitTUIModelV2) renderWelcome() string {
	content := m.styles.RenderHeader(
		"Welcome to Guild Framework",
		"Let's forge your development guild together",
	)

	// Enhanced medieval-themed introduction featuring Elena
	intro := `
🏰 **Welcome to the Guild Framework**

The Guild awaits your command, Master Artisan. You are about to establish 
a legendary development guild that will transform how you create software.

**Meet Elena, Your Guild Master:**
Elena the Guild Master will be your trusted coordinator - a wise and experienced 
leader who specializes in orchestrating teams of AI specialists. With 18 years 
of experience guiding diverse teams to create legendary software works, Elena 
will help you organize, plan, and execute your development projects with grace 
and efficiency.

**Your Guild Will Include:**
  🧙 **Elena the Guild Master** - Project coordination and team leadership
  ⚔️  **Marcus the Code Artisan** - Master craftsman of digital logic
  🛡️  **Vera the Quality Guardian** - Protector of software excellence
  🎯 **Smart AI Provider Detection** - Automatic setup of available AI services

Together, we shall establish:
  📋 A mighty **Campaign** to guide your quest
  🏗️  A stalwart **Project** to house your works  
  🤖 Wise **AI Providers** automatically detected and configured
  👥 Skilled **Agents** with rich personalities and expertise

Press Enter to begin forging your legendary development guild...
`

	rendered, _ := m.renderer.Render(intro)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		content,
		m.styles.Section.Render(rendered),
		m.renderHelp(),
	)
}

func (m *InitTUIModelV2) renderHelp() string {
	return m.help.ShortHelpView([]key.Binding{
		keys.Enter,
		keys.Quit,
	})
}

// Key bindings
var keys = keyBindings{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c/esc", "quit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
}

type keyBindings struct {
	Enter key.Binding
	Quit  key.Binding
	Up    key.Binding
	Down  key.Binding
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InitDependencies holds all dependencies for the init TUI
type InitDependencies struct {
	ConfigManager ConfigurationManager
	ProjectInit   ProjectInitializer
	DemoGen       DemoGenerator
	Validator     Validator
	DaemonManager DaemonManager
}
