// pkg/ui/commission/update.go
package commission

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/internal/ui/commission/components"
	commissionpkg "github.com/guild-ventures/guild-core/pkg/commission"
)

// Custom message types with Guild-themed names
type ContextCraftedMsg struct {
	Content string
	Success bool
	Error   error
}

// Reference to availableCommands from model.go
var availableCommands string

type DocumentRefiningMsg struct {
	Success bool
	Error   error
}

type MasterSuggestionMsg struct {
	Suggestions string
	Success     bool
	Error       error
}

type CommissionReadyMsg struct {
	Success bool
	Error   error
}

type CommissionLoadedMsg struct {
	Commission string
	Success    bool
	Error      error
}

// CommandMsg represents a command entered in the command input
type CommandMsg struct {
	Command string
}

// Update handles UI events and state changes
func (m CommissionChamber) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// First check global keybindings that work in all states
		switch {
		case key.Matches(msg, m.keymap.LeaveHall):
			m.proclamation = "Departing the Guild Hall..."
			return m, tea.Quit

		case key.Matches(msg, m.keymap.SeekGuidance):
			// Toggle help display
			m.helpScroll.ShowAll = !m.helpScroll.ShowAll
			return m, nil
		}

		// State-specific key handling
		switch m.chamberState {
		case stateViewing:
			// Viewing state key bindings
			switch {
			case key.Matches(msg, m.keymap.Craft):
				m.chamberState = stateContext
				m.scribe.Focus()
				m.proclamation = "The Guild scribe prepares to record your context..."

			case key.Matches(msg, m.keymap.Refine):
				m.proclamation = "Summoning the Guild's craftsmen to refine the documents..."
				return m, generateDocumentsCmd(&m)

			case key.Matches(msg, m.keymap.ConsultMaster):
				m.proclamation = "Seeking the Guild Master's counsel on improvements..."
				return m, requestSuggestionsCmd(&m)

			case key.Matches(msg, m.keymap.ApproveWork):
				m.proclamation = "Preparing to seal the work with the Guild's mark of approval..."
				return m, markCommissionReadyCmd(&m)

			case key.Matches(msg, m.keymap.ExamineDocs):
				m.chamberState = statePreview
				m.proclamation = "Unrolling the document scrolls for examination..."

			case key.Matches(msg, m.keymap.ToggleView):
				m.chamberState = stateDashboard
				m.proclamation = "Opening the Guild's objective ledger..."
				return m, loadCommissionsCmd(&m)

			case key.Matches(msg, m.keymap.EnterHall):
				m.chamberState = stateCommands
				m.parchment.Focus()
				m.proclamation = "Enter thy command, and the Guild shall obey..."
			}

		case stateContext:
			// Adding context state key bindings
			switch {
			case key.Matches(msg, m.keymap.NavigateUp),
				key.Matches(msg, m.keymap.NavigateDown),
				key.Matches(msg, m.keymap.NavigateLeft),
				key.Matches(msg, m.keymap.NavigateRight):
				// Pass these navigation keys to the textarea
				m.scribe, cmd = m.scribe.Update(msg)
				cmds = append(cmds, cmd)

			case msg.Type == tea.KeyEsc:
				// Return to viewing state
				m.chamberState = stateViewing
				m.scribe.Blur()
				m.proclamation = "Context crafting cancelled. Returned to the main chamber."

			case msg.Type == tea.KeyEnter && key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+enter"))):
				// Submit context with Ctrl+Enter
				content := m.scribe.Value()
				if content != "" {
					m.contextHistory = append(m.contextHistory, content)
					m.scribe.Reset()
					m.chamberState = stateViewing
					m.proclamation = "Context added to the Guild's knowledge. The craftsmen ponder..."
					return m, addContextCmd(&m, content)
				}
			}

		case statePreview:
			// Preview docs state key bindings
			switch {
			case key.Matches(msg, m.keymap.NavigateUp),
				key.Matches(msg, m.keymap.NavigateDown):
				// Scroll viewport
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)

			case msg.Type == tea.KeyEsc, key.Matches(msg, m.keymap.ExamineDocs):
				// Return to viewing state
				m.chamberState = stateViewing
				m.proclamation = "Returned to the main chamber."
			}

		case stateDashboard:
			// Dashboard state key bindings
			switch {
			case key.Matches(msg, m.keymap.NavigateUp),
				key.Matches(msg, m.keymap.NavigateDown):
				// Navigate the ledger list
				m.ledger, cmd = m.ledger.Update(msg)
				cmds = append(cmds, cmd)

			case key.Matches(msg, m.keymap.ToggleView),
				msg.Type == tea.KeyEsc:
				// Return to viewing state
				m.chamberState = stateViewing
				m.proclamation = "Closed the Guild's objective ledger."

			case msg.Type == tea.KeyEnter:
				// Select objective from ledger
				// TODO: Get selected objective and load it
				m.chamberState = stateViewing
				m.proclamation = "A new objective scroll unfurls before you..."
			}

		case stateCommands:
			// Command input state key bindings
			switch {
			case msg.Type == tea.KeyEsc:
				// Return to viewing state
				m.chamberState = stateViewing
				m.parchment.Blur()
				m.proclamation = "Command cancelled. Returned to the main chamber."

			case msg.Type == tea.KeyEnter:
				// Process command
				command := m.parchment.Value()
				m.parchment.Reset()
				m.parchment.Blur()
				m.chamberState = stateViewing

				if command == "" {
					m.proclamation = "Command was empty. Returned to the main chamber."
				} else {
					m.proclamation = fmt.Sprintf("Executing command: %s", command)
					return m, executeCommandCmd(&m, command)
				}
			}

		case stateCreating:
			// Creating a new objective state key bindings
			switch {
			case key.Matches(msg, m.keymap.NavigateUp),
				key.Matches(msg, m.keymap.NavigateDown),
				key.Matches(msg, m.keymap.NavigateLeft),
				key.Matches(msg, m.keymap.NavigateRight):
				// Pass these navigation keys to the textarea
				m.scribe, cmd = m.scribe.Update(msg)
				cmds = append(cmds, cmd)

			case msg.Type == tea.KeyEsc:
				// Return to viewing state
				m.chamberState = stateViewing
				m.scribe.Blur()
				m.proclamation = "Objective creation cancelled. Returned to the main chamber."

			case msg.Type == tea.KeyEnter && key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+enter"))):
				// Submit new objective with Ctrl+Enter
				content := m.scribe.Value()
				if content != "" {
					m.scribe.Reset()
					m.chamberState = stateViewing
					m.proclamation = "The Guild craftsmen begin to shape your objective..."
					return m, createCommissionCmd(&m, content)
				}
			}
		}

	case tea.WindowSizeMsg:
		// Handle window resizing with Guild-themed variable names
		m.hallWidth = msg.Width
		m.hallHeight = msg.Height

		// Update viewport dimensions
		headerHeight := 2 // Title
		footerHeight := 4 // Status + help
		contentHeight := m.hallHeight - headerHeight - footerHeight

		m.viewport.Width = m.hallWidth
		m.viewport.Height = contentHeight

		// Update textarea width
		m.scribe.SetWidth(m.hallWidth)

		// Update list dimensions
		m.ledger.SetSize(m.hallWidth, contentHeight)

		// Update help width
		m.helpScroll.Width = m.hallWidth

	case ContextCraftedMsg:
		// Handle context processing results
		if !msg.Success {
			m.guildError = msg.Error
			m.proclamation = "The scribes failed to record your wisdom: " + msg.Error.Error()
		} else {
			m.proclamation = "Your knowledge has been added to the Guild's records."

			// Update preview content if we have one
			if m.currentCommission != nil {
				m.commissionPreview = formatCommissionPreview(m.currentCommission)
			}
		}

	case DocumentRefiningMsg:
		// Handle regeneration results
		if !msg.Success {
			m.guildError = msg.Error
			m.proclamation = "The craftsmen failed to refine the documents: " + msg.Error.Error()
		} else {
			m.proclamation = "The documents have been skillfully refined by our artisans."
			// Update previews with new content
			// This would update aiDocsPreview and specsPreview
		}

	case MasterSuggestionMsg:
		// Handle suggestions from the Guild Master
		if !msg.Success {
			m.guildError = msg.Error
			m.proclamation = "The Guild Master's counsel could not be obtained: " + msg.Error.Error()
		} else {
			m.proclamation = "The Guild Master has offered wisdom to improve your objective."
			// Set viewport content to show suggestions
			m.viewport.SetContent(fmt.Sprintf("Guild Master's Wisdom:\n\n%s", msg.Suggestions))
			m.chamberState = statePreview
		}

	case CommissionReadyMsg:
		// Handle marking objective as ready
		if !msg.Success {
			m.guildError = msg.Error
			m.proclamation = "Could not mark the work as masterful: " + msg.Error.Error()
		} else {
			m.readyForMaster = true
			m.proclamation = "The work has been sealed with the Guild's mark of approval!"
		}

	case CommissionLoadedMsg:
		// Handle loading an objective
		if !msg.Success {
			m.guildError = msg.Error
			m.proclamation = "The objective scroll could not be retrieved: " + msg.Error.Error()
		} else {
			m.commissionPreview = msg.Commission
			m.proclamation = "The objective scroll has been unfurled before you."
			// Update the view with the objective content
			m.viewport.SetContent(msg.Commission)
		}

	case CommandMsg:
		// Handle command execution results
		// Parse the command and execute appropriate action
		// This would be a more complex implementation based on command parsing
	}

	// Always update active components based on state
	switch m.chamberState {
	case stateContext, stateCreating:
		m.scribe, cmd = m.scribe.Update(msg)
		cmds = append(cmds, cmd)

	case stateCommands:
		m.parchment, cmd = m.parchment.Update(msg)
		cmds = append(cmds, cmd)

	case statePreview:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case stateDashboard:
		m.ledger, cmd = m.ledger.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// Custom commands for objective operations

// addContextCmd creates a command to add context to the objective
func addContextCmd(m *CommissionChamber, content string) tea.Cmd {
	return func() tea.Msg {
		// If planner is available, use it
		if m.planner != nil && m.planner.GetSession().Commission != nil {
			// Create a context for the operation
			ctx := m.ctx

			// Add context to the objective
			err := m.planner.AddContext(ctx, content)
			if err != nil {
				return ContextCraftedMsg{
					Content: content,
					Success: false,
					Error:   err,
				}
			}

			return ContextCraftedMsg{
				Content: content,
				Success: true,
				Error:   nil,
			}
		}

		// Fallback to mock behavior if planner not available
		return ContextCraftedMsg{
			Content: content,
			Success: true,
			Error:   nil,
		}
	}
}

// generateDocumentsCmd creates a command to regenerate documents
func generateDocumentsCmd(m *CommissionChamber) tea.Cmd {
	return func() tea.Msg {
		// If planner is available, use it
		if m.planner != nil && m.planner.GetSession().Commission != nil {
			// Create a context for the operation
			ctx := m.ctx

			// Regenerate documents
			err := m.planner.Regenerate(ctx)
			if err != nil {
				return DocumentRefiningMsg{
					Success: false,
					Error:   err,
				}
			}

			return DocumentRefiningMsg{
				Success: true,
				Error:   nil,
			}
		}

		// Fallback to mock behavior if planner not available
		return DocumentRefiningMsg{
			Success: true,
			Error:   nil,
		}
	}
}

// requestSuggestionsCmd creates a command to get suggestions for the objective
func requestSuggestionsCmd(m *CommissionChamber) tea.Cmd {
	return func() tea.Msg {
		// If planner is available, use it
		if m.planner != nil && m.planner.GetSession().Commission != nil {
			// Create a context for the operation
			ctx := m.ctx

			// Get suggestions
			suggestions, err := m.planner.GetSuggestions(ctx)
			if err != nil {
				return MasterSuggestionMsg{
					Suggestions: "",
					Success:     false,
					Error:       err,
				}
			}

			return MasterSuggestionMsg{
				Suggestions: suggestions,
				Success:     true,
				Error:       nil,
			}
		}

		// Fallback to mock behavior if planner not available
		sampleSuggestions := `Consider the following improvements to your objective:

1. Add more specific success criteria to the Requirements section
2. Include technical constraints if applicable
3. Clarify the intended audience or users
4. Consider adding relevant tags for better organization`

		return MasterSuggestionMsg{
			Suggestions: sampleSuggestions,
			Success:     true,
			Error:       nil,
		}
	}
}

// markCommissionReadyCmd creates a command to mark the objective as ready
func markCommissionReadyCmd(m *CommissionChamber) tea.Cmd {
	return func() tea.Msg {
		// If planner is available, use it
		if m.planner != nil && m.planner.GetSession().Commission != nil {
			// Create a context for the operation
			ctx := m.ctx

			// Mark objective as ready
			err := m.planner.MarkReady(ctx)
			if err != nil {
				return CommissionReadyMsg{
					Success: false,
					Error:   err,
				}
			}

			return CommissionReadyMsg{
				Success: true,
				Error:   nil,
			}
		}

		// Fallback to mock behavior if planner not available
		return CommissionReadyMsg{
			Success: true,
			Error:   nil,
		}
	}
}

// loadCommissionsCmd creates a command to load all objectives for the dashboard
func loadCommissionsCmd(m *CommissionChamber) tea.Cmd {
	return func() tea.Msg {
		// If objective manager is available, use it
		if m.commissionManager != nil {
			// Create a context for the operation
			ctx := m.ctx

			// List all objectives
			commissions, err := m.commissionManager.ListCommissions(ctx)
			if err != nil {
				return nil // Handle error
			}

			// Convert commissions to list items
			items := make([]list.Item, 0, len(commissions))
			for _, obj := range commissions {
				items = append(items, components.CommissionItem{
					ID:          obj.ID,
					Title:       obj.Title,
					Status:      string(obj.Status),
					Path:        obj.FilePath,
					Iterations:  obj.Iteration,
					CreatedAt:   obj.CreatedAt,
					ModifiedAt:  obj.UpdatedAt,
					Tags:        obj.Tags,
					Completion:  obj.Completion,
					Description: obj.Description,
				})
			}

			// Update the list
			m.ledger.SetItems(items)
			return nil
		}

		// Fallback to mock data if manager not available
		items := []list.Item{
			components.CommissionItem{
				ID:          "mock-1",
				Title:       "Build a RESTful API service",
				Status:      "in_progress",
				Path:        "/objectives/api-service.md",
				Iterations:  3,
				CreatedAt:   time.Now().Add(-72 * time.Hour),
				ModifiedAt:  time.Now().Add(-24 * time.Hour),
				Tags:        []string{"api", "backend"},
				Completion:  0.7,
				Description: "Create a RESTful API service for the application",
			},
			components.CommissionItem{
				ID:          "mock-2",
				Title:       "Design user authentication system",
				Status:      "draft",
				Path:        "/objectives/auth-system.md",
				Iterations:  1,
				CreatedAt:   time.Now().Add(-48 * time.Hour),
				ModifiedAt:  time.Now().Add(-48 * time.Hour),
				Tags:        []string{"auth", "security"},
				Completion:  0.3,
				Description: "Design a secure user authentication system",
			},
			components.CommissionItem{
				ID:          "mock-3",
				Title:       "Implement database schema",
				Status:      "completed",
				Path:        "/objectives/db-schema.md",
				Iterations:  5,
				CreatedAt:   time.Now().Add(-96 * time.Hour),
				ModifiedAt:  time.Now().Add(-12 * time.Hour),
				Tags:        []string{"database", "schema"},
				Completion:  1.0,
				Description: "Design and implement the database schema",
			},
		}
		m.ledger.SetItems(items)
		return nil
	}
}

// createCommissionCmd creates a command to create a new objective
func createCommissionCmd(m *CommissionChamber, description string) tea.Cmd {
	return func() tea.Msg {
		// If planner is available, use it
		if m.planner != nil {
			// Create a context for the operation
			ctx := m.ctx

			// Create objective
			err := m.planner.CreateCommission(ctx, description)
			if err != nil {
				return CommissionLoadedMsg{
					Commission: "",
					Success:    false,
					Error:      err,
				}
			}

			// Get the objective content
			session := m.planner.GetSession()
			if session.Commission != nil {
				objectiveContent := ""
				if session.Commission.Content != "" {
					objectiveContent = session.Commission.Content
				} else {
					// Format the content if it's not available
					objectiveContent = formatCommissionContent(session.Commission)
				}

				return CommissionLoadedMsg{
					Commission: objectiveContent,
					Success:    true,
					Error:      nil,
				}
			}
		}

		// Fallback to mock behavior if planner not available
		objectiveContent := fmt.Sprintf(`# 🧠 Goal

%s

# 📂 Context

This objective was created in the Guild Hall.

# 🔧 Requirements

- Requirement 1
- Requirement 2
- Requirement 3

# 📌 Tags

- new
- objective

# 🔗 Related

- None yet
`, description)

		return CommissionLoadedMsg{
			Commission: objectiveContent,
			Success:    true,
			Error:      nil,
		}
	}
}

// Helper function to format objective content
func formatCommissionContent(obj *commissionpkg.Commission) string {
	if obj == nil {
		return "No objective available"
	}

	content := fmt.Sprintf(`# 🧠 Goal

%s

# 📂 Context

%s

# 🔧 Requirements

`, obj.Goal, obj.Description)

	// Add requirements
	if len(obj.Requirements) > 0 {
		for _, req := range obj.Requirements {
			content += fmt.Sprintf("- %s\n", req)
		}
	} else {
		content += "- No requirements defined yet\n"
	}

	content += "\n# 📌 Tags\n\n"

	// Add tags
	if len(obj.Tags) > 0 {
		for _, tag := range obj.Tags {
			content += fmt.Sprintf("- %s\n", tag)
		}
	} else {
		content += "- No tags defined yet\n"
	}

	content += "\n# 🔗 Related\n\n"

	// Add related
	if len(obj.Related) > 0 {
		for _, rel := range obj.Related {
			content += fmt.Sprintf("- %s\n", rel)
		}
	} else {
		content += "- None yet\n"
	}

	return content
}

// executeCommandCmd creates a command to execute a command string
func executeCommandCmd(m *CommissionChamber, command string) tea.Cmd {
	return func() tea.Msg {
		// Parse the command and execute appropriate action
		// Format: command [args...]
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return nil
		}

		cmd := parts[0]
		args := parts[1:]

		// Handle built-in UI commands
		switch cmd {
		case "add-context", "craft":
			if len(args) > 0 {
				context := strings.Join(args, " ")
				return addContextCmd(m, context)()
			}

		case "regenerate", "refine":
			return generateDocumentsCmd(m)()

		case "suggest":
			return requestSuggestionsCmd(m)()

		case "ready":
			return markCommissionReadyCmd(m)()

		case "list":
			// Switch to dashboard view which shows the objective list
			m.chamberState = stateDashboard
			return loadCommissionsCmd(m)()

		case "create":
			// If arguments provided, create objective from them
			if len(args) > 0 {
				description := strings.Join(args, " ")
				return createCommissionCmd(m, description)()
			}
			// Otherwise switch to create view
			m.chamberState = stateCreating
			m.scribe.Focus()
			m.proclamation = "Describe your new objective..."
			return nil

		case "help":
			// Show help view and command reference
			m.helpScroll.ShowAll = true

			// Also show command reference in viewport
			m.viewport.SetContent(availableCommands)
			m.chamberState = statePreview
			return nil

		case "view":
			// If path provided, attempt to load objective
			if len(args) > 0 && m.commissionManager != nil {
				path := args[0]
				ctx := m.ctx

				// Try to load the objective
				obj, err := m.commissionManager.LoadCommissionFromFile(ctx, path)
				if err != nil {
					return CommissionLoadedMsg{
						Commission: "",
						Success:    false,
						Error:      err,
					}
				}

				// If planner exists, set up the planning session
				if m.planner != nil {
					err := m.planner.SetCommission(ctx, obj.ID)
					if err != nil {
						return CommissionLoadedMsg{
							Commission: "",
							Success:    false,
							Error:      err,
						}
					}
				}

				m.currentCommission = obj
				content := ""
				if obj.Content != "" {
					content = obj.Content
				} else {
					content = formatCommissionContent(obj)
				}

				return CommissionLoadedMsg{
					Commission: content,
					Success:    true,
					Error:      nil,
				}
			}
			return nil

		// Guild CLI passthrough commands - these will run the actual CLI commands as processes
		case "exec":
			// Format: exec <subcommand> [args...]
			if len(args) > 0 {
				return executeExternalCommandCmd(strings.Join(args, " "))()
			}

		// Direct subcommand passthrough
		case "list-all", "create-obj", "view-obj", "agent":
			// These are direct passthroughs to guild CLI commands
			// Construct the command based on the UI command
			var cliCmd string
			switch cmd {
			case "list-all":
				cliCmd = "objective list"
			case "create-obj":
				if len(args) > 0 {
					cliCmd = "objective create " + strings.Join(args, " ")
				} else {
					cliCmd = "objective create"
				}
			case "view-obj":
				if len(args) > 0 {
					cliCmd = "objective view " + strings.Join(args, " ")
				} else {
					cliCmd = "objective view"
				}
			case "agent":
				if len(args) > 0 {
					cliCmd = "agent " + strings.Join(args, " ")
				} else {
					cliCmd = "agent"
				}
			}

			if cliCmd != "" {
				return executeExternalCommandCmd(cliCmd)()
			}
		}

		return nil
	}
}

// Helper methods for update...
func formatCommissionPreview(obj *commissionpkg.Commission) string {
	// Create a formatted preview of the objective
	// This would format the objective details into a nice display
	// using actual objective data

	// Mock implementation for now
	if obj == nil {
		return "No objective loaded"
	}

	return fmt.Sprintf(`# %s

Goal: %s

Status: %s
Iterations: %d

Tags: %s
`,
		obj.Title,
		obj.Description,
		obj.Status,
		obj.Iteration,
		strings.Join(obj.Tags, ", "),
	)
}

// init initializes variables needed by this package
func init() {
	// Initialize command reference
	availableCommands = `
Guild Hall Command Reference:

UI Commands:
  help                   - Show this guidance
  create [description]   - Create a new objective
  view [path]            - View an objective by path
  list                   - List all objectives
  add-context [text]     - Add context to current objective
  regenerate             - Regenerate documents
  suggest                - Get improvement suggestions
  ready                  - Mark objective as ready

CLI Passthrough:
  exec [command]         - Execute any guild command
  list-all               - List all objectives (CLI)
  create-obj [desc]      - Create objective (CLI)
  view-obj [id]          - View objective (CLI)
  agent [subcommand]     - Run agent commands

Use ctrl+enter to submit in text areas.
`
}

// executeExternalCommandCmd creates a command to execute an external Guild CLI command
func executeExternalCommandCmd(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		// Execute the external command
		result := components.ExecuteExternalCommand(cmdStr)

		// Format the output for display
		outputMsg := fmt.Sprintf("Command: guild %s\n\n", cmdStr)
		if result.Success {
			outputMsg += fmt.Sprintf("Success! Output:\n%s", result.Output)
		} else {
			outputMsg += fmt.Sprintf("Error (code %d):\n%s\n\nOutput:\n%s",
				result.ExitCode,
				result.Error.Error(),
				result.Output)
		}

		// Return a message that will display this output in the viewport
		return CommissionLoadedMsg{
			Commission: outputMsg,
			Success:    true,
			Error:      nil,
		}
	}
}
