// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	commissionpkg "github.com/guild-ventures/guild-core/pkg/commission"
)

// Define UI states
const (
	stateViewing   = "viewing"   // Viewing commission details
	stateEditing   = "editing"   // Editing commission content
	stateCreating  = "creating"  // Creating a new commission
	stateContext   = "context"   // Adding context
	statePreview   = "preview"   // Previewing generated docs
	stateCommands  = "commands"  // Command input mode
	stateDashboard = "dashboard" // Commissions dashboard
)

// Define key mappings using Guild lore-inspired names
type GuildHallKeyMap struct {
	// Navigation
	NavigateUp    key.Binding
	NavigateDown  key.Binding
	NavigateLeft  key.Binding
	NavigateRight key.Binding

	// Actions
	Craft         key.Binding // Add context (create)
	Refine        key.Binding // Regenerate (refine)
	ConsultMaster key.Binding // Suggest improvements
	ApproveWork   key.Binding // Mark as ready
	ExamineDocs   key.Binding // Preview docs

	// UI Controls
	EnterHall    key.Binding // Enter command mode
	LeaveHall    key.Binding // Exit
	SeekGuidance key.Binding // Help
	ToggleView   key.Binding // Toggle between views
}

// Define UI state using Guild metaphors
type CommissionChamber struct {
	// Session state
	ctx               context.Context           // Context for operations
	commissionManager CommissionManager         // Manages commissions - interface dependency
	planner           CommissionPlanner         // Plans commissions - interface dependency
	currentCommission *commissionpkg.Commission // Current commission
	generator         CommissionGenerator       // LLM generator for commissions - interface dependency
	commissionPath    string                    // Path to current commission file

	// UI components
	scribe     textarea.Model  // Text input for longer content (medieval scribe)
	parchment  textinput.Model // Text input for commands (writing on parchment)
	viewport   viewport.Model  // Content viewing area (viewing the scroll)
	ledger     list.Model      // Commissions list (guild ledger)
	helpScroll help.Model      // Help display (instruction scroll)
	keymap     GuildHallKeyMap // Key bindings

	// UI state
	hallWidth, hallHeight int    // Terminal dimensions (hall dimensions)
	chamberState          string // Current UI state (which chamber we're in)
	readyForMaster        bool   // Whether commission is ready (ready for guildmaster)
	proclamation          string // Status message (town crier's proclamation)
	guildError            error  // Error state

	// Content
	aiDocsPreview     string   // Preview of generated ai_docs
	specsPreview      string   // Preview of generated specs
	commissionPreview string   // Preview of current commission
	contextHistory    []string // History of added context
}

// DefaultKeyMap returns the default key mappings with Guild-themed help text
func DefaultKeyMap() GuildHallKeyMap {
	return GuildHallKeyMap{
		NavigateUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move northward"),
		),
		NavigateDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move southward"),
		),
		NavigateLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move westward"),
		),
		NavigateRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move eastward"),
		),
		Craft: key.NewBinding(
			key.WithKeys("a", "c"),
			key.WithHelp("a/c", "craft context"),
		),
		Refine: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refine documents"),
		),
		ConsultMaster: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "seek master's counsel"),
		),
		ApproveWork: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mark work as masterful"),
		),
		ExamineDocs: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "peruse documents"),
		),
		EnterHall: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "enter the command hall"),
		),
		LeaveHall: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "leave the guild hall"),
		),
		SeekGuidance: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "seek apprentice guidance"),
		),
		ToggleView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "change chamber view"),
		),
	}
}

// ShortHelp returns key bindings to be shown in the mini help view.
func (k GuildHallKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.SeekGuidance, k.LeaveHall}
}

// FullHelp returns keybindings for the expanded help view.
func (k GuildHallKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NavigateUp, k.NavigateDown, k.NavigateLeft, k.NavigateRight},
		{k.Craft, k.Refine, k.ConsultMaster, k.ApproveWork},
		{k.ExamineDocs, k.ToggleView, k.EnterHall},
		{k.SeekGuidance, k.LeaveHall},
	}
}

// NewModel creates a new Guild Hall model for commission planning
func NewModel(ctx context.Context, commissionPath string, manager CommissionManager, planner CommissionPlanner, generator CommissionGenerator) *CommissionChamber {
	// Initialize textarea for context input
	scribe := textarea.New()
	scribe.Placeholder = "Enter context or reference documents (e.g., @spec/path/to/file.md)"
	scribe.Focus()
	scribe.SetHeight(10)
	scribe.ShowLineNumbers = false

	// Initialize command input
	parchment := textinput.New()
	parchment.Placeholder = "Enter command or : to begin"
	parchment.CharLimit = 250

	// Initialize viewport for displaying content
	viewport := viewport.New(80, 20)
	viewport.SetContent("Welcome to the Guild Hall Commission Chamber")

	// Initialize commissions list
	ledger := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ledger.Title = "Guild Commissions Ledger"
	ledger.SetShowStatusBar(false)
	ledger.SetFilteringEnabled(false)

	// Initialize help
	helpScroll := help.New()

	// Create the model with Guild-themed names
	chamber := &CommissionChamber{
		ctx:               ctx,
		commissionPath:    commissionPath,
		commissionManager: manager,
		planner:           planner,
		generator:         generator,
		scribe:            scribe,
		parchment:         parchment,
		viewport:          viewport,
		ledger:            ledger,
		helpScroll:        helpScroll,
		keymap:            DefaultKeyMap(),
		chamberState:      stateViewing,
		proclamation:      "Welcome to the Guild Commission Chamber. How may we assist your planning?",
		contextHistory:    []string{},
	}

	// If commission path is provided, load that commission
	if commissionPath != "" && manager != nil {
		// Load the commission using the manager
		obj, err := manager.LoadCommissionFromFile(ctx, commissionPath)
		if err != nil {
			chamber.proclamation = "Failed to load commission scroll: " + err.Error()
		} else {
			chamber.currentCommission = obj
			chamber.proclamation = "Examining the commission scroll: " + obj.Title

			// If planner exists, set up the planning session
			if planner != nil {
				err := planner.SetCommission(ctx, obj.ID)
				if err != nil {
					chamber.proclamation = "Failed to start planning session: " + err.Error()
				}
			}
		}
	} else {
		// No commission or no manager, offer to create one
		chamber.chamberState = stateCreating
		chamber.proclamation = "A blank parchment awaits. Describe your commission to begin crafting."
	}

	return chamber
}

// Init implements tea.Model
func (m CommissionChamber) Init() tea.Cmd {
	// Start with multiple initialization commands
	return tea.Batch(
		textarea.Blink,
		textinput.Blink,
	)
}
