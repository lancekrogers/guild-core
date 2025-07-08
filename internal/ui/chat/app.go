// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite" // SQLite driver

	"github.com/lancekrogers/guild/internal/daemonconn"
	"github.com/lancekrogers/guild/internal/ui/chat/agents"
	"github.com/lancekrogers/guild/internal/ui/chat/commands"
	"github.com/lancekrogers/guild/internal/ui/chat/common"
	cfig "github.com/lancekrogers/guild/internal/ui/chat/common/config"
	"github.com/lancekrogers/guild/internal/ui/chat/common/layout"
	"github.com/lancekrogers/guild/internal/ui/chat/common/types"
	"github.com/lancekrogers/guild/internal/ui/chat/common/utils"
	"github.com/lancekrogers/guild/internal/ui/chat/completion"
	"github.com/lancekrogers/guild/internal/ui/chat/managers"
	"github.com/lancekrogers/guild/internal/ui/chat/messages"
	toolmsg "github.com/lancekrogers/guild/internal/ui/chat/messages/tools"
	"github.com/lancekrogers/guild/internal/ui/chat/panes"
	"github.com/lancekrogers/guild/internal/ui/chat/services"
	"github.com/lancekrogers/guild/internal/ui/chat/session"
	"github.com/lancekrogers/guild/internal/ui/formatting"
	uitools "github.com/lancekrogers/guild/internal/ui/tools"
	"github.com/lancekrogers/guild/internal/ui/vim"
	"github.com/lancekrogers/guild/internal/ui/visual"
	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/campaign"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	promptspb "github.com/lancekrogers/guild/pkg/grpc/pb/prompts/v1"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/templates"
	"github.com/lancekrogers/guild/pkg/tools"

	"github.com/lancekrogers/guild/tools/search"
)

// App represents the main chat application
type App struct {
	// Core configuration
	ctx    context.Context
	config *common.ChatConfig

	// Layout and panes
	layoutManager *layout.Manager
	outputPane    panes.OutputPane
	inputPane     panes.InputPane
	statusPane    panes.StatusPane

	// Services
	chatService     *services.ChatService
	daemonService   *services.DaemonService
	providerService *services.ProviderService

	// Agent communication
	agentRouter *agents.AgentRouter

	// Utilities
	styles *utils.Styles
	keys   *utils.KeyBindings

	// gRPC clients
	grpcConn      *grpc.ClientConn
	guildClient   pb.GuildClient
	chatClient    pb.ChatServiceClient
	sessionClient pb.SessionServiceClient
	promptsClient promptspb.PromptServiceClient
	registry      registry.ComponentRegistry

	// Session management
	sessionManager session.SessionManager
	currentSession *session.Session

	// Application state
	messages      []common.ChatMessage
	activeTools   map[string]*common.ToolExecution
	agents        []string
	currentView   common.ViewMode
	searchResults []int
	searchPattern string
	completions   []completion.CompletionResult

	// Command processing
	commandProcessor *commands.CommandProcessor
	commandHistory   *commands.CommandHistory
	templateManager  templates.TemplateManager
	completionEngine *completion.CompletionEngine

	// Real-time suggestions
	suggestionManager *InputSuggestionManager

	// NEW: Suggestion system integration
	suggestionFactory *core.SuggestionAwareAgentFactory
	chatHandler       *core.ChatSuggestionHandler
	enhancedAgent     core.EnhancedGuildArtisan

	// Guild selection
	selectedGuild string

	// Daemon connection management
	connManager      *daemonconn.Manager
	connectionInfo   *daemonconn.ConnectionInfo
	connectionStatus bool
	directMode       bool // True when running without daemon

	// Migrated V1 utilities
	contentFormatter *formatting.ContentFormatter
	markdownRenderer *formatting.MarkdownRenderer
	toolVisualizer   *uitools.ToolVisualizer
	vimModeManager   *vim.VimModeManager
	imageProcessor   *visual.ImageProcessor
	codeRenderer     *visual.CodeRenderer
	mermaidProcessor *visual.MermaidProcessor

	// Feature flags
	initialized bool
	ready       bool
	shouldQuit  bool
	errorState  error
}

// NewApp creates a new chat application (simplified wrapper)
func NewApp(ctx context.Context, guildConfig *config.GuildConfig,
	registry registry.ComponentRegistry,
) *App {
	// Create basic app structure
	app := &App{
		ctx:              ctx,
		registry:         registry,
		messages:         make([]common.ChatMessage, 0),
		activeTools:      make(map[string]*common.ToolExecution),
		agents:           make([]string, 0),
		currentView:      common.ViewModeNormal,
		connManager:      daemonconn.NewManager(ctx),
		connectionStatus: false,
	}

	// Store guild config for later initialization
	app.config = &common.ChatConfig{
		GuildConfig: guildConfig,
		Width:       80,
		Height:      24,
	}

	return app
}

// SetCampaignID sets the campaign ID for the chat session
func (app *App) SetCampaignID(campaignID string) {
	if app.config != nil {
		app.config.CampaignID = campaignID
	}
}

// SetSessionID sets the session ID for the chat session
func (app *App) SetSessionID(sessionID string) {
	if app.config != nil {
		app.config.SessionID = sessionID
	}
}

// Run starts the chat application
func (app *App) Run() error {
	// Initialize components during run
	if err := app.initializeComponents(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize chat").
			WithComponent("chat.core").
			WithOperation("Run")
	}

	// Create and run the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run chat interface").
			WithComponent("chat.core").
			WithOperation("Run")
	}

	return nil
}

// initializeComponents initializes all application components
func (app *App) initializeComponents() error {
	// Initialize utilities
	app.styles = utils.NewStyles()
	app.keys = utils.NewKeyBindings()

	// Initialize command history
	app.commandHistory = commands.NewCommandHistory(1000)

	// Initialize visual components
	if err := app.initializeVisualComponents(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize visual components").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize session management first
	if err := app.initializeSessionManagement(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize session management").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize template manager (for now, nil - would need database setup)
	app.templateManager = nil

	// Initialize suggestion-aware agent system
	if err := app.initializeSuggestionSystem(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize suggestion system").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize completion engine with suggestion support
	projectRoot := "." // TODO: Get actual project root

	// Try to get project root from registry if available
	// Note: There's a type mismatch between registry.ProjectContext interface
	// and the actual project.Context implementation, so we'll skip this for now
	// and use the default project root

	// Create enhanced completion engine with direct suggestion provider integration
	enhancedEngine, err := completion.NewCompletionEngineEnhanced(app.config.GuildConfig, projectRoot)
	if err == nil {
		app.completionEngine = enhancedEngine.CompletionEngine
	} else {
		// Fall back to basic completion engine if enhanced fails
		app.completionEngine = completion.NewCompletionEngine(app.config.GuildConfig, projectRoot)
	}

	// Integrate agent-based suggestion system if available
	if app.enhancedAgent != nil {
		// Set the enhanced agent directly on the completion engine
		app.completionEngine.SetEnhancedAgent(app.enhancedAgent, app.chatHandler)
	}

	if app.registry != nil {
		app.completionEngine.SetRegistry(app.registry)
	}

	// Initialize command processor with session manager and template manager
	app.commandProcessor = commands.NewCommandProcessor(app.ctx, app.config, app.commandHistory,
		app.sessionManager, app.currentSession, app.templateManager, app.guildClient)

	// Connect completion engine to command processor for live command updates
	if app.completionEngine != nil && app.commandProcessor != nil {
		app.completionEngine.SetCommandProcessor(app.commandProcessor)
	}

	// Initialize services
	if err := app.initializeServices(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize services").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize agent router
	app.agentRouter = agents.NewAgentRouter(app.ctx, app.guildClient)

	// Initialize layout manager
	app.layoutManager = layout.NewManager(app.config.Width, app.config.Height)
	if err := app.layoutManager.Initialize(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize layout manager").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize panes
	if err := app.initializePanes(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize panes").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize daemon connection
	if err := app.initializeDaemonConnection(); err != nil {
		// Enable direct mode fallback
		app.enableDirectMode()
	}

	app.initialized = true
	return nil
}

// initializeSessionManagement initializes session persistence
func (app *App) initializeSessionManagement() error {
	// Open database connection
	db, err := sql.Open("sqlite3", app.config.DatabasePath)
	if err != nil {
		// Continue without session persistence if database fails
		fmt.Printf("Warning: Failed to open session database: %v\n", err)
		return nil
	}

	// Create session store and manager
	store := session.NewSQLiteStore(db)
	app.sessionManager = session.NewManager(store)

	// Load or create session
	if app.config.SessionID != "" {
		// Try to load existing session
		session, err := app.sessionManager.LoadSession(app.config.SessionID)
		if err != nil {
			// Create new session if load fails
			name := fmt.Sprintf("Chat Session %s", app.config.SessionID[:8])
			session, err = app.sessionManager.NewSession(name, &app.config.CampaignID)
			if err != nil {
				fmt.Printf("Warning: Failed to create session: %v\n", err)
				return nil
			}
			app.config.SessionID = session.ID
		}
		app.currentSession = session
	}

	// Load existing messages if we have a session
	if app.currentSession != nil {
		messages, err := app.sessionManager.GetContext(app.currentSession.ID, 50)
		if err == nil && len(messages) > 0 {
			// Convert session messages to chat messages
			for _, msg := range messages {
				chatMsg := app.convertSessionMessage(msg)
				app.messages = append(app.messages, chatMsg)
			}
		}
	}

	return nil
}

// initializeServices initializes the service layer
func (app *App) initializeServices() error {
	// Initialize chat service with suggestion support if available
	var chatService *services.ChatService
	var err error

	if app.enhancedAgent != nil {
		// Create chat service with suggestion support
		chatService, err = services.NewChatServiceWithSuggestions(
			app.ctx,
			app.guildClient,
			app.registry,
			app.enhancedAgent,
		)
	} else {
		// Create regular chat service
		chatService, err = services.NewChatService(app.ctx, app.guildClient, app.registry)
	}

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chat service").
			WithComponent("chat.app").
			WithOperation("initializeServices")
	}
	app.chatService = chatService

	// Initialize daemon service
	daemonService, err := services.NewDaemonService(app.ctx, app.config.CampaignID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create daemon service").
			WithComponent("chat.app").
			WithOperation("initializeServices")
	}
	app.daemonService = daemonService

	// Initialize provider service
	providerService, err := services.NewProviderService(app.ctx, app.config.GuildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider service").
			WithComponent("chat.app").
			WithOperation("initializeServices")
	}
	app.providerService = providerService

	return nil
}

// initializePanes initializes the UI panes
func (app *App) initializePanes() error {
	// Calculate pane dimensions using layout manager
	outputRect := app.layoutManager.GetPaneRect("output")
	inputRect := app.layoutManager.GetPaneRect("input")
	statusRect := app.layoutManager.GetPaneRect("status")

	// Initialize output pane
	outputPane, err := panes.NewOutputPane(outputRect.Width, outputRect.Height, app.config.EnableRichContent)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create output pane").
			WithComponent("chat.app").
			WithOperation("initializePanes")
	}
	app.outputPane = outputPane

	// Set content formatter on output pane if available
	if app.contentFormatter != nil {
		app.outputPane.SetContentFormatter(app.contentFormatter)
	}

	// Initialize input pane with vim support
	inputPane, err := panes.NewVimEnabledInputPane(inputRect.Width, inputRect.Height, app.config.EnableCompletion, app.vimModeManager)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create input pane").
			WithComponent("chat.app").
			WithOperation("initializePanes")
	}
	app.inputPane = inputPane

	// Set up input callbacks
	app.setupInputCallbacks()

	// Initialize status pane
	statusPane, err := panes.NewStatusPane(statusRect.Width, statusRect.Height)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create status pane").
			WithComponent("chat.app").
			WithOperation("initializePanes")
	}
	app.statusPane = statusPane

	return nil
}

// setupInputCallbacks sets up callbacks for the input pane
func (app *App) setupInputCallbacks() {
	// Initialize suggestion manager with 300ms debounce
	app.suggestionManager = NewInputSuggestionManager(app, 300*time.Millisecond)

	// Set up OnChange callback for dynamic resizing and suggestions
	app.inputPane.OnChange(func(input string) {
		// Update dynamic height based on content
		lines := strings.Count(input, "\n") + 1
		if lines < 1 {
			lines = 1
		}

		// Update layout manager with new input height
		if app.layoutManager != nil {
			if err := app.layoutManager.UpdateInputHeight(lines); err == nil {
				// Resize panes based on new layout
				inputRect := app.layoutManager.GetPaneRect("input")
				app.inputPane.Resize(inputRect.Width, inputRect.Height)

				// Update other panes that might be affected
				outputRect := app.layoutManager.GetPaneRect("output")
				app.outputPane.Resize(outputRect.Width, outputRect.Height)

				statusRect := app.layoutManager.GetPaneRect("status")
				app.statusPane.Resize(statusRect.Width, statusRect.Height)
			}
		}

		// Trigger suggestion request
		if app.suggestionManager != nil {
			// This will be handled in the main Update loop to avoid blocking
		}
	})

	// Set up OnSubmit callback
	app.inputPane.OnSubmit(func(input string) {
		// Hide suggestions when submitting
		app.inputPane.HideCompletions()
		app.statusPane.HideCompletions()

		// Process the input through command processor
		if app.commandProcessor != nil {
			isCommand, cmd := app.commandProcessor.ProcessInput(input)
			if isCommand {
				// Execute command and get result
				if cmd != nil {
					// We can't execute the command directly here since we're in a callback
					// Instead, add to output pane indicating command was processed
					userMsg := common.ChatMessage{
						Type:      common.MsgUser,
						Content:   input,
						Timestamp: time.Now(),
					}
					app.outputPane.AddMessage(userMsg)
				}
			} else {
				// Regular message - should be sent to agents
				userMsg := common.ChatMessage{
					Type:      common.MsgUser,
					Content:   input,
					Timestamp: time.Now(),
				}
				app.outputPane.AddMessage(userMsg)

				// Send to agent router for processing
				if app.agentRouter != nil {
					// Parse input for agent mentions
					target, err := app.agentRouter.ParseInput(input)
					if err != nil {
						// Handle parse error
						errorMsg := common.ChatMessage{
							Type:      common.MsgSystem,
							Content:   fmt.Sprintf("❌ Error parsing agent mention: %s", err.Error()),
							AgentID:   "system",
							Timestamp: time.Now(),
							Metadata:  make(map[string]string),
						}
						app.outputPane.AddMessage(errorMsg)
						return
					}

					// Schedule agent communication as a separate command
					// Since we're in a callback, we need to trigger this via the program
					go func() {
						var cmd tea.Cmd
						if target != nil {
							// Route to specific agent or broadcast
							if target.IsBroadcast {
								cmd = app.agentRouter.BroadcastToAll(target.Message)
							} else {
								cmd = app.agentRouter.SendToAgent(target.ID, target.Message)
							}
						} else {
							// No agent mention, send to default agent (elena-guild-master)
							cmd = app.agentRouter.SendToAgent("elena-guild-master", input)
						}

						// Execute the command and handle the result
						if cmd != nil {
							if msg := cmd(); msg != nil {
								// Send the message to the main program
								// Note: This is a simplified approach - in a full implementation
								// we would need proper message passing
							}
						}
					}()
				} else {
					// No agent router available
					errorMsg := common.ChatMessage{
						Type:      common.MsgSystem,
						Content:   "❌ Agent router not available. Please check guild daemon connection.",
						AgentID:   "system",
						Timestamp: time.Now(),
						Metadata:  make(map[string]string),
					}
					app.outputPane.AddMessage(errorMsg)
				}
			}
		}
	})
}

// convertSessionMessage converts a session message to a chat message
func (app *App) convertSessionMessage(msg *session.Message) common.ChatMessage {
	var msgType common.MessageType
	var agentID string

	switch msg.Role {
	case session.RoleUser:
		msgType = common.MsgUser
	case session.RoleAssistant:
		msgType = common.MsgAgent
		// Extract agent ID from metadata if available
		if msg.Metadata != nil {
			if id, ok := msg.Metadata["agent_id"].(string); ok {
				agentID = id
			}
		}
	case session.RoleSystem:
		msgType = common.MsgSystem
	default:
		msgType = common.MsgSystem
	}

	return common.ChatMessage{
		Type:      msgType,
		Content:   msg.Content,
		AgentID:   agentID,
		Timestamp: msg.CreatedAt,
		Metadata:  make(map[string]string),
	}
}

// Implement tea.Model interface

// Init initializes the Bubble Tea model
func (app *App) Init() tea.Cmd {
	if !app.initialized {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Application not properly initialized",
				Level:   "error",
			}
		}
	}

	// Initialize all panes
	cmds := []tea.Cmd{
		app.outputPane.Init(),
		app.inputPane.Init(),
		app.statusPane.Init(),
	}

	// Start services
	if app.chatService != nil {
		cmds = append(cmds, app.chatService.Start())
	}
	if app.daemonService != nil {
		cmds = append(cmds, app.daemonService.Start())
	}
	if app.providerService != nil {
		cmds = append(cmds, app.providerService.Start())
	}

	// Start agent router by refreshing agent list and status updates
	if app.agentRouter != nil {
		cmds = append(cmds, app.agentRouter.RefreshAgentList())
		cmds = append(cmds, app.startPeriodicStatusUpdates())
	}

	// Show welcome message
	welcomeMsg := app.generateWelcomeMessage()
	app.outputPane.AddMessage(welcomeMsg)

	app.ready = true
	return tea.Batch(cmds...)
}

// Update handles Bubble Tea messages
func (app *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return app.handleWindowSize(msg)

	case tea.KeyMsg:
		return app.handleKeyPress(msg)

	case agents.AgentStreamMsg:
		return app.handleAgentStream(msg)

	case panes.StatusUpdateMsg:
		return app.handleStatusUpdate(msg)

	case panes.LayoutUpdateMsg:
		return app.handleLayoutUpdate(msg)

	case panes.PaneUpdateMsg:
		return app.handlePaneUpdate(msg)

	case panes.ViewModeChangeMsg:
		return app.handleViewModeChange(msg)

	case toolmsg.ToolExecutionStartMsg:
		return app.handleToolExecutionStart(msg)

	case toolmsg.ToolExecutionProgressMsg:
		return app.handleToolExecutionProgress(msg)

	case toolmsg.ToolExecutionCompleteMsg:
		return app.handleToolExecutionComplete(msg)

	case struct {
		Type  string
		Input string
	}:
		if msg.Type == "completion_request" {
			return app.handleCompletionRequestAnon(msg.Input)
		}

	case completion.SuggestionRequestMsg:
		return app.handleSuggestionRequest(msg)

	case completion.CompletionResultMsg:
		return app.handleCompletionResult(msg)

	case messages.SearchMsg:
		return app.handleSearch(msg)

	case messages.VimModeToggleMsg:
		return app.handleVimModeToggle(msg)

	case agents.AgentResponseMsg:
		return app.handleAgentResponse(msg)

	case agents.BroadcastResponseMsg:
		return app.handleBroadcastResponse(msg)

	case agents.AgentErrorMsg:
		return app.handleAgentError(msg)

	case agents.AgentListUpdatedMsg:
		return app.handleAgentListUpdated(msg)

	case agents.AgentStatusMsg:
		return app.handleAgentStatusUpdated(msg)
	}

	// Update panes
	var paneCmd tea.Cmd
	var updatedModel tea.Model

	updatedModel, paneCmd = app.outputPane.Update(msg)
	if outputPane, ok := updatedModel.(panes.OutputPane); ok {
		app.outputPane = outputPane
	}
	if paneCmd != nil {
		cmds = append(cmds, paneCmd)
	}

	// Store old input value before update
	oldInputValue := app.inputPane.GetValue()

	updatedModel, paneCmd = app.inputPane.Update(msg)
	if inputPane, ok := updatedModel.(panes.InputPane); ok {
		app.inputPane = inputPane
	}
	if paneCmd != nil {
		cmds = append(cmds, paneCmd)
	}

	// Check if input changed and trigger suggestions and dynamic resize
	newInputValue := app.inputPane.GetValue()
	if oldInputValue != newInputValue {
		// Handle dynamic input resizing
		lines := strings.Count(newInputValue, "\n") + 1
		if lines < 1 {
			lines = 1
		}

		// Update layout manager with new input height
		if app.layoutManager != nil {
			if err := app.layoutManager.UpdateInputHeight(lines); err == nil {
				// Resize panes based on new layout
				inputRect := app.layoutManager.GetPaneRect("input")
				app.inputPane.Resize(inputRect.Width, inputRect.Height)

				// Update other panes that might be affected
				outputRect := app.layoutManager.GetPaneRect("output")
				app.outputPane.Resize(outputRect.Width, outputRect.Height)

				statusRect := app.layoutManager.GetPaneRect("status")
				app.statusPane.Resize(statusRect.Width, statusRect.Height)
			}
		}

		// Handle suggestions
		if app.suggestionManager != nil {
			if suggestionCmd := app.suggestionManager.HandleInputChange(newInputValue); suggestionCmd != nil {
				cmds = append(cmds, suggestionCmd)
			}
		}
	}

	updatedModel, paneCmd = app.statusPane.Update(msg)
	if statusPane, ok := updatedModel.(panes.StatusPane); ok {
		app.statusPane = statusPane
	}
	if paneCmd != nil {
		cmds = append(cmds, paneCmd)
	}

	return app, tea.Batch(cmds...)
}

// View renders the application
func (app *App) View() string {
	if !app.ready {
		return "Initializing Guild Chat..."
	}

	if app.shouldQuit {
		return "Goodbye! 🏰"
	}

	if app.errorState != nil {
		return fmt.Sprintf("Error: %v", app.errorState)
	}

	// Get pane views
	outputView := app.outputPane.View()
	inputView := app.inputPane.View()
	statusView := app.statusPane.View()

	// Use layout manager to compose the final view
	return app.layoutManager.Render(map[string]string{
		"output": outputView,
		"input":  inputView,
		"status": statusView,
	})
}

// generateWelcomeMessage creates the welcome message for new sessions
func (app *App) generateWelcomeMessage() common.ChatMessage {
	var content string

	// Always show Elena's welcome for a rich experience
	content = app.getElenaWelcomeMessage()

	return common.ChatMessage{
		Type:      common.MsgSystem,
		Content:   content,
		AgentID:   "system",
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
}

// getElenaWelcomeMessage returns Elena's personalized welcome message
func (app *App) getElenaWelcomeMessage() string {
	// Get Elena's welcome from the campaign defaults
	return campaign.GetDefaultElenaWelcome()
}

// Event handlers - these will be implemented as the components are built

func (app *App) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// Update config dimensions
	configManager := cfig.NewConfigManager(app.ctx)
	configManager.UpdateDimensions(app.config, msg.Width, msg.Height)

	// Update layout manager
	app.layoutManager.Resize(msg.Width, msg.Height)

	// Update panes with new dimensions
	outputRect := app.layoutManager.GetPaneRect("output")
	inputRect := app.layoutManager.GetPaneRect("input")
	statusRect := app.layoutManager.GetPaneRect("status")

	var cmds []tea.Cmd

	// Resize panes
	app.outputPane.Resize(outputRect.Width, outputRect.Height)
	app.inputPane.Resize(inputRect.Width, inputRect.Height)
	app.statusPane.Resize(statusRect.Width, statusRect.Height)

	// Update visual components with new width
	// Note: MarkdownRenderer doesn't have a SetWidth method - would need to recreate it
	// if app.markdownRenderer != nil {
	//     app.markdownRenderer.SetWidth(msg.Width)
	// }
	if app.contentFormatter != nil {
		app.contentFormatter.UpdateWidth(msg.Width)
	}
	if app.imageProcessor != nil {
		app.imageProcessor.SetASCIISize(msg.Width-10, 30)
	}
	if app.codeRenderer != nil {
		app.codeRenderer.SetMaxWidth(msg.Width - 10)
	}
	if app.mermaidProcessor != nil {
		app.mermaidProcessor.SetASCIISize(msg.Width-10, 30)
	}

	return app, tea.Batch(cmds...)
}

func (app *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle completion navigation if completions are showing
	if len(app.completions) > 0 {
		switch msg.String() {
		case "up", "ctrl+p":
			// Navigate up in completions
			currentIndex := 0
			for i, comp := range app.completions {
				if comp.Content == app.inputPane.GetCurrentCompletion() {
					currentIndex = i
					break
				}
			}
			if currentIndex > 0 {
				currentIndex--
			} else {
				currentIndex = len(app.completions) - 1
			}
			app.statusPane.ShowCompletions(getCompletionStrings(app.completions), currentIndex)
			return app, nil

		case "down", "ctrl+n":
			// Navigate down in completions
			currentIndex := 0
			for i, comp := range app.completions {
				if comp.Content == app.inputPane.GetCurrentCompletion() {
					currentIndex = i
					break
				}
			}
			if currentIndex < len(app.completions)-1 {
				currentIndex++
			} else {
				currentIndex = 0
			}
			app.statusPane.ShowCompletions(getCompletionStrings(app.completions), currentIndex)
			return app, nil

		case "tab", "enter":
			// Accept current completion
			if len(app.completions) > 0 {
				currentIndex := 0
				for i, comp := range app.completions {
					if comp.Content == app.inputPane.GetCurrentCompletion() {
						currentIndex = i
						break
					}
				}
				app.inputPane.SetValue(app.completions[currentIndex].Content)
				app.completions = []completion.CompletionResult{}
				app.statusPane.HideCompletions()
				return app, nil
			}

		case "esc":
			// Hide completions
			app.completions = []completion.CompletionResult{}
			app.statusPane.HideCompletions()
			return app, nil
		}
	}

	// Handle global shortcuts
	switch {
	case key.Matches(msg, app.keys.Quit):
		app.shouldQuit = true
		return app, tea.Quit

	case key.Matches(msg, app.keys.Submit):
		return app.handleSubmit()

	case key.Matches(msg, app.keys.CommandPalette):
		return app.handleCommandPalette()

	case key.Matches(msg, app.keys.GlobalSearch):
		return app.handleGlobalSearch()

	case key.Matches(msg, app.keys.Help):
		return app.handleHelp()
	}

	// Let the currently focused pane handle the key
	// For now, assume input pane is always focused
	var cmd tea.Cmd
	updatedModel, cmd := app.inputPane.Update(msg)
	if inputPane, ok := updatedModel.(panes.InputPane); ok {
		app.inputPane = inputPane
	}

	return app, cmd
}

// Helper methods - placeholders for now, will be implemented with components

func (app *App) handleSubmit() (tea.Model, tea.Cmd) {
	input := app.inputPane.GetValue()
	if input == "" {
		return app, nil
	}

	// Clear input
	app.inputPane.SetValue("")

	// 1. Check for slash commands first
	isCommand, cmd := app.commandProcessor.ProcessInput(input)
	if isCommand {
		return app, cmd
	}

	// 2. Check for agent mentions (@agent message)
	if agentTarget, err := app.agentRouter.ParseInput(input); err != nil {
		// Show error for invalid agent mention
		errorMsg := common.ChatMessage{
			Type:      common.MsgSystem,
			Content:   fmt.Sprintf("❌ %s", err.Error()),
			AgentID:   "system",
			Timestamp: app.GetCurrentTime(),
			Metadata:  make(map[string]string),
		}
		app.outputPane.AddMessage(errorMsg)
		return app, nil
	} else if agentTarget != nil {
		// Valid agent mention - add user message and route to agent
		userMsg := common.ChatMessage{
			Type:      common.MsgUser,
			Content:   input,
			AgentID:   "user",
			Timestamp: app.GetCurrentTime(),
			Metadata:  make(map[string]string),
		}
		app.outputPane.AddMessage(userMsg)

		// Route to agent or broadcast
		if agentTarget.IsBroadcast {
			return app, app.agentRouter.BroadcastToAll(agentTarget.Message)
		} else {
			return app, app.agentRouter.SendToAgent(agentTarget.ID, agentTarget.Message)
		}
	}

	// 3. Default: treat as general message to all agents
	userMsg := common.ChatMessage{
		Type:      common.MsgUser,
		Content:   input,
		AgentID:   "user",
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(userMsg)

	return app, app.agentRouter.BroadcastToAll(input)
}

func (app *App) handleCommandPalette() (tea.Model, tea.Cmd) {
	if app.commandProcessor == nil || app.outputPane == nil {
		return app, nil
	}

	commands := app.commandProcessor.GetAvailableCommands()
	if len(commands) == 0 {
		msg := common.ChatMessage{
			Type:      common.MsgSystem,
			Content:   "No commands available",
			Timestamp: app.GetCurrentTime(),
			Metadata:  map[string]string{"source": "palette"},
		}
		app.outputPane.AddMessage(msg)
		return app, nil
	}

	var b strings.Builder
	b.WriteString("\u2728 Available Commands:\n")
	for _, c := range commands {
		b.WriteString(fmt.Sprintf(" • /%s - %s\n", c.Name, c.Description))
	}

	msg := common.ChatMessage{
		Type:      common.MsgSystem,
		Content:   b.String(),
		Timestamp: app.GetCurrentTime(),
		Metadata:  map[string]string{"source": "palette"},
	}
	app.outputPane.AddMessage(msg)
	return app, nil
}

func (app *App) handleGlobalSearch() (tea.Model, tea.Cmd) {
	if app.inputPane == nil || app.outputPane == nil {
		return app, nil
	}

	pattern := strings.TrimSpace(app.inputPane.GetValue())
	if pattern == "" {
		app.statusPane.UpdateStatus("Enter a search pattern first", "warning")
		return app, nil
	}

	ag := search.NewAgTool(app.config.ProjectRoot)
	input := fmt.Sprintf(`{"pattern": "%s"}`, pattern)
	result, err := ag.Execute(app.ctx, input)
	if err != nil {
		app.statusPane.UpdateStatus("search failed", "error")
		return app, nil
	}
	if result.Metadata["error"] == "ag_not_installed" {
		app.statusPane.UpdateStatus("ag tool not installed", "error")
		return app, nil
	}

	var indices []int
	if res, ok := result.ExtraData["structured_results"].(*search.AgToolResult); ok {
		for _, r := range res.Results {
			line := fmt.Sprintf("%s:%d:%s", r.File, r.Line, strings.TrimSpace(r.Match))
			msg := common.ChatMessage{
				Type:      common.MsgSystem,
				Content:   line,
				Timestamp: app.GetCurrentTime(),
				Metadata:  map[string]string{"source": "search"},
			}
			app.outputPane.AddMessage(msg)
			indices = append(indices, r.Line)
		}
	}

	return app, func() tea.Msg {
		return messages.SearchMsg{Pattern: pattern, Results: indices}
	}
}

func (app *App) handleHelp() (tea.Model, tea.Cmd) {
	return app, app.commandProcessor.ProcessCommand("help")
}

func (app *App) handleAgentStream(msg agents.AgentStreamMsg) (tea.Model, tea.Cmd) {
	// Add agent message to output
	chatMsg := common.ChatMessage{
		Type:      common.MsgAgent,
		Content:   msg.Content,
		AgentID:   msg.AgentID,
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(chatMsg)

	return app, nil
}

func (app *App) handleStatusUpdate(msg panes.StatusUpdateMsg) (tea.Model, tea.Cmd) {
	app.statusPane.UpdateStatus(msg.Message, msg.Level)
	return app, nil
}

func (app *App) handleLayoutUpdate(msg panes.LayoutUpdateMsg) (tea.Model, tea.Cmd) {
	app.layoutManager.Resize(msg.Width, msg.Height)
	return app, nil
}

func (app *App) handlePaneUpdate(msg panes.PaneUpdateMsg) (tea.Model, tea.Cmd) {
	switch msg.PaneID {
	case "output":
		if msg.Data == "clear" {
			app.outputPane.Clear()
		} else if msg.Content != "" {
			systemMsg := common.ChatMessage{
				Type:      common.MsgSystem,
				Content:   msg.Content,
				AgentID:   "system",
				Timestamp: app.GetCurrentTime(),
				Metadata:  make(map[string]string),
			}
			app.outputPane.AddMessage(systemMsg)
		}
	}
	return app, nil
}

func (app *App) handleViewModeChange(msg panes.ViewModeChangeMsg) (tea.Model, tea.Cmd) {
	app.currentView = msg.Mode
	return app, nil
}

func (app *App) handleToolExecutionStart(msg toolmsg.ToolExecutionStartMsg) (tea.Model, tea.Cmd) {
	// Track tool execution
	execution := &common.ToolExecution{
		ID:        msg.ExecutionID,
		ToolName:  msg.ToolName,
		AgentID:   msg.AgentID,
		StartTime: app.GetCurrentTime(),
		Status:    "starting",
		Metadata:  map[string]string{"parameters": fmt.Sprintf("%v", msg.Parameters)},
	}
	app.activeTools[msg.ExecutionID] = execution

	// Add tool start message
	toolMsg := common.ChatMessage{
		Type:      common.MsgToolStart,
		Content:   fmt.Sprintf("🔨 Starting tool: %s", msg.ToolName),
		AgentID:   msg.AgentID,
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(toolMsg)

	return app, nil
}

func (app *App) handleToolExecutionProgress(msg toolmsg.ToolExecutionProgressMsg) (tea.Model, tea.Cmd) {
	if execution, exists := app.activeTools[msg.ExecutionID]; exists {
		execution.Progress = msg.Progress
		execution.Status = "running"
	}
	return app, nil
}

func (app *App) handleToolExecutionComplete(msg toolmsg.ToolExecutionCompleteMsg) (tea.Model, tea.Cmd) {
	if execution, exists := app.activeTools[msg.ExecutionID]; exists {
		now := app.GetCurrentTime()
		execution.EndTime = &now
		execution.Status = "completed"
		execution.Result = msg.Result

		// Add completion message
		toolMsg := common.ChatMessage{
			Type:      common.MsgToolComplete,
			Content:   fmt.Sprintf("✅ Tool completed: %s\n%s", execution.ToolName, msg.Result),
			AgentID:   execution.AgentID,
			Timestamp: now,
			Metadata:  make(map[string]string),
		}
		app.outputPane.AddMessage(toolMsg)

		// Remove from active tools
		delete(app.activeTools, msg.ExecutionID)
	}
	return app, nil
}

func (app *App) handleCompletionRequestAnon(input string) (tea.Model, tea.Cmd) {
	// Use completion engine to generate suggestions
	if app.completionEngine != nil {
		results := app.completionEngine.Complete(input, len(input))
		return app, func() tea.Msg {
			return completion.CompletionResultMsg{
				Results: results,
			}
		}
	}

	return app, nil
}

func (app *App) handleCompletionResult(msg completion.CompletionResultMsg) (tea.Model, tea.Cmd) {
	// Only show completions if they're for the current input
	currentInput := app.inputPane.GetValue()

	// Check if the suggestions are stale (ForInput field was added to CompletionResultMsg)
	if msg.ForInput != "" && msg.ForInput != currentInput {
		return app, nil // Stale suggestions, ignore
	}

	app.completions = msg.Results

	// Show completions in status pane
	if len(msg.Results) > 0 {
		// Extract completion strings
		completionStrings := make([]string, len(msg.Results))
		for i, result := range msg.Results {
			completionStrings[i] = result.Content
		}
		app.statusPane.ShowCompletions(completionStrings, 0)

		// Dynamically size status pane for completions (2 lines header + number of results, max 8)
		statusHeight := 2 + len(msg.Results)
		if statusHeight > 8 {
			statusHeight = 8
		}
		if app.layoutManager != nil {
			app.layoutManager.ShowStatusPane(statusHeight)

			// Update pane sizes based on new layout
			statusRect := app.layoutManager.GetPaneRect("status")
			app.statusPane.Resize(statusRect.Width, statusRect.Height)

			// Update other panes that might be affected
			outputRect := app.layoutManager.GetPaneRect("output")
			app.outputPane.Resize(outputRect.Width, outputRect.Height)

			inputRect := app.layoutManager.GetPaneRect("input")
			app.inputPane.Resize(inputRect.Width, inputRect.Height)
		}
	} else {
		app.statusPane.HideCompletions()

		// Hide status pane when no completions
		if app.layoutManager != nil {
			app.layoutManager.HideStatusPane()

			// Update pane sizes based on new layout
			statusRect := app.layoutManager.GetPaneRect("status")
			app.statusPane.Resize(statusRect.Width, statusRect.Height)

			// Update other panes that might be affected
			outputRect := app.layoutManager.GetPaneRect("output")
			app.outputPane.Resize(outputRect.Width, outputRect.Height)

			inputRect := app.layoutManager.GetPaneRect("input")
			app.inputPane.Resize(inputRect.Width, inputRect.Height)
		}
	}

	return app, nil
}

// handleSuggestionRequest processes a suggestion request
func (app *App) handleSuggestionRequest(msg completion.SuggestionRequestMsg) (tea.Model, tea.Cmd) {
	if app.suggestionManager != nil {
		return app, app.suggestionManager.ProcessSuggestionRequest(msg.Input)
	}
	return app, nil
}

func (app *App) handleSearch(msg messages.SearchMsg) (tea.Model, tea.Cmd) {
	app.searchPattern = msg.Pattern
	app.searchResults = msg.Results
	// TODO: Update UI to show search results
	return app, nil
}

func (app *App) handleVimModeToggle(msg messages.VimModeToggleMsg) (tea.Model, tea.Cmd) {
	// Toggle vim mode on the vim manager and input adapter
	if app.vimModeManager != nil && app.inputPane != nil {
		// Check if the input pane is a vim adapter
		if vimAdapter, ok := app.inputPane.(*panes.VimInputAdapter); ok {
			// Toggle vim mode state
			isEnabled := vimAdapter.IsEnabled()
			vimAdapter.SetEnabled(!isEnabled)

			// Update status
			if !isEnabled {
				app.statusPane.UpdateStatus("Vim mode enabled (NORMAL)", "info")
				app.statusPane.SetVimMode("NORMAL")
				app.statusPane.SetInputMode("vim")
			} else {
				app.statusPane.UpdateStatus("Vim mode disabled (INSERT)", "info")
				app.statusPane.SetVimMode("")
				app.statusPane.SetInputMode("insert")
			}
		} else {
			app.statusPane.UpdateStatus("Vim mode not available", "warning")
		}
	}
	return app, nil
}

// Utility methods

func (app *App) GetCurrentTime() time.Time {
	return time.Now()
}

func (app *App) IsReady() bool {
	return app.ready
}

func (app *App) ShouldQuit() bool {
	return app.shouldQuit
}

func (app *App) GetError() error {
	return app.errorState
}

func (app *App) SetError(err error) {
	app.errorState = err
}

// Agent message handlers

func (app *App) handleAgentResponse(msg agents.AgentResponseMsg) (tea.Model, tea.Cmd) {
	// Add agent response to output
	agentMsg := common.ChatMessage{
		Type:      common.MsgAgent,
		Content:   msg.Content,
		AgentID:   msg.AgentID,
		Timestamp: msg.Timestamp,
		Metadata:  map[string]string{"message_id": msg.MessageID},
	}
	app.outputPane.AddMessage(agentMsg)

	// Save to session if available
	if app.sessionManager != nil && app.currentSession != nil {
		// Convert to the expected format (no tool calls for regular agent responses)
		app.sessionManager.AppendMessage(app.currentSession.ID, session.RoleAssistant, msg.Content, nil)
	}

	return app, nil
}

func (app *App) handleBroadcastResponse(msg agents.BroadcastResponseMsg) (tea.Model, tea.Cmd) {
	// Add responses from all agents
	for _, response := range msg.Responses {
		agentMsg := common.ChatMessage{
			Type:      common.MsgAgent,
			Content:   response.Response,
			AgentID:   response.AgentId,
			Timestamp: msg.Timestamp,
			Metadata:  map[string]string{"message_id": msg.MessageID},
		}
		app.outputPane.AddMessage(agentMsg)

		// Save each broadcast response to session if available
		if app.sessionManager != nil && app.currentSession != nil {
			// Convert to the expected format (no tool calls for broadcast responses)
			app.sessionManager.AppendMessage(app.currentSession.ID, session.RoleAssistant, response.Response, nil)
		}
	}

	return app, nil
}

func (app *App) handleAgentError(msg agents.AgentErrorMsg) (tea.Model, tea.Cmd) {
	// Display error message
	errorMsg := common.ChatMessage{
		Type:      common.MsgSystem,
		Content:   fmt.Sprintf("❌ Agent %s error: %s", msg.AgentID, msg.Error.Error()),
		AgentID:   "system",
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(errorMsg)

	return app, nil
}

func (app *App) handleAgentListUpdated(msg agents.AgentListUpdatedMsg) (tea.Model, tea.Cmd) {
	// Update agent list in status pane or show notification
	statusMsg := fmt.Sprintf("🔄 Agent list updated: %d agents available", len(msg.Agents))
	app.statusPane.UpdateStatus(statusMsg, "info")

	return app, nil
}

func (app *App) handleAgentStatusUpdated(msg agents.AgentStatusMsg) (tea.Model, tea.Cmd) {
	// Update agent status display
	statusIcon := agents.GetStatusIcon(msg.Status.State)
	statusMsg := fmt.Sprintf("%s Agent %s: %s", statusIcon, msg.AgentID, msg.Status.CurrentTask)
	app.statusPane.UpdateStatus(statusMsg, "info")

	return app, nil
}

// refreshAllAgentStatuses refreshes the status of all available agents
func (app *App) refreshAllAgentStatuses() tea.Cmd {
	if app.agentRouter == nil {
		return nil
	}

	return tea.Batch(
		app.agentRouter.RefreshAgentList(),
		func() tea.Msg {
			// Get status for all known agents
			availableAgents := app.agentRouter.GetAvailableAgents()
			var cmds []tea.Cmd
			for _, agentID := range availableAgents {
				cmds = append(cmds, app.agentRouter.GetAgentStatus(agentID))
			}

			// Schedule next refresh in 10 seconds
			time.AfterFunc(10*time.Second, func() {
				// This would trigger the next refresh cycle in a real implementation
				// For now, we'll leave it as a placeholder since we need proper
				// message scheduling in the TUI framework
			})

			return tea.Batch(cmds...)()
		},
	)
}

// startPeriodicStatusUpdates initiates periodic agent status updates
func (app *App) startPeriodicStatusUpdates() tea.Cmd {
	return app.refreshAllAgentStatuses()
}

// initializeSuggestionSystem initializes the suggestion-aware agent system
func (app *App) initializeSuggestionSystem() error {
	// Get components from registry
	if app.registry == nil {
		// Graceful fallback - continue without suggestions
		return nil
	}

	// Get LLM client
	llmClient, err := app.getLLMClient()
	if err != nil {
		// Continue without suggestions if LLM client fails
		fmt.Printf("Warning: Failed to get LLM client for suggestions: %v\n", err)
		return nil
	}

	// Get memory manager
	memoryManager, err := app.getMemoryManager()
	if err != nil {
		// Continue without suggestions if memory manager fails
		fmt.Printf("Warning: Failed to get memory manager for suggestions: %v\n", err)
		return nil
	}

	// Get tool registry
	toolRegistry, err := app.getToolRegistry()
	if err != nil {
		// Continue without suggestions if tool registry fails
		fmt.Printf("Warning: Failed to get tool registry for suggestions: %v\n", err)
		return nil
	}

	// Get commission manager
	commissionManager, err := app.getCommissionManager()
	if err != nil {
		// Continue without suggestions if commission manager fails
		fmt.Printf("Warning: Failed to get commission manager for suggestions: %v\n", err)
		return nil
	}

	// Get cost manager (placeholder for now)
	costManager := &managers.MinimalCostManager{}

	// Create suggestion-aware agent factory
	app.suggestionFactory = core.NewSuggestionAwareAgentFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create enhanced agent for chat
	app.enhancedAgent = app.suggestionFactory.CreateWorkerAgent("chat-agent", "Chat Assistant")

	// Create chat suggestion handler
	app.chatHandler = core.NewChatSuggestionHandler(app.enhancedAgent)

	return nil
}

// Helper methods to get components from registry
func (app *App) getLLMClient() (providers.LLMClient, error) {
	if app.registry == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "registry not available", nil).
			WithComponent("chat.app").
			WithOperation("getLLMClient")
	}

	// Try to get default provider from registry
	if app.registry.Providers() != nil {
		provider, err := app.registry.Providers().GetDefaultProvider()
		if err == nil && provider != nil {
			return provider, nil
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "LLM client not available", nil).
		WithComponent("chat.app").
		WithOperation("getLLMClient")
}

func (app *App) getMemoryManager() (memory.ChainManager, error) {
	if app.registry == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "registry not available", nil).
			WithComponent("chat.app").
			WithOperation("getMemoryManager")
	}

	// Try to get default chain manager from registry
	if app.registry.Memory() != nil {
		manager, err := app.registry.Memory().GetDefaultChainManager()
		if err == nil && manager != nil {
			return manager, nil
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "memory chain manager not available", nil).
		WithComponent("chat.app").
		WithOperation("getMemoryManager")
}

func (app *App) getToolRegistry() (tools.Registry, error) {
	if app.registry == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "registry not available", nil).
			WithComponent("chat.app").
			WithOperation("getToolRegistry")
	}

	// The component registry Tools() returns a registry.ToolRegistry interface
	// We need to get the actual tools from it and create a new tools.Registry
	if app.registry.Tools() != nil {
		// Create a new tool registry that implements tools.Registry
		toolRegistry := tools.NewToolRegistry()

		// Get all tools from the component registry and register them
		for _, toolName := range app.registry.Tools().ListTools() {
			tool, err := app.registry.Tools().GetTool(toolName)
			if err == nil && tool != nil {
				// Register the tool in our local registry
				_ = toolRegistry.RegisterTool(toolName, tool)
			}
		}

		return toolRegistry, nil
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "tool registry not available", nil).
		WithComponent("chat.app").
		WithOperation("getToolRegistry")
}

func (app *App) getCommissionManager() (commission.CommissionManager, error) {
	if app.registry == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "registry not available", nil).
			WithComponent("chat.app").
			WithOperation("getCommissionManager")
	}

	// Commission manager is not directly available in the registry interface
	// Create a minimal implementation that satisfies the interface
	// This is a placeholder that allows the suggestion system to work
	return &managers.MinimalCommissionManager{}, nil
}

// SetSelectedGuild sets the selected guild for the chat session
func (app *App) SetSelectedGuild(guildName string) {
	app.selectedGuild = guildName
}

// GetSelectedGuild returns the selected guild for the chat session
func (app *App) GetSelectedGuild() string {
	return app.selectedGuild
}

// initializeVisualComponents initializes all visual and formatting utilities
func (app *App) initializeVisualComponents() error {
	// Initialize markdown renderer with current terminal width
	width := app.config.Width
	if width == 0 {
		width = 80 // Default width
	}

	// Create markdown renderer
	markdownRenderer, err := formatting.NewMarkdownRenderer(width)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create markdown renderer").
			WithComponent("chat.app").
			WithOperation("initializeVisualComponents")
	}
	app.markdownRenderer = markdownRenderer

	// Get project workspace path
	workspacePath := "."
	// TODO: Get actual project workspace path from registry or config
	// For now, use current directory as workspace

	// Initialize content formatter with markdown renderer
	app.contentFormatter = formatting.NewContentFormatter(markdownRenderer, width, workspacePath)

	// Initialize tool visualizer
	app.toolVisualizer = uitools.NewToolVisualizer()

	// Initialize vim mode manager
	app.vimModeManager = vim.NewVimModeManager()

	// Initialize visual processors
	app.imageProcessor = visual.NewImageProcessor()
	app.imageProcessor.SetASCIISize(width-10, 30) // Adjust for chat width

	app.codeRenderer = visual.NewCodeRenderer()
	app.codeRenderer.SetMaxWidth(width - 10)

	app.mermaidProcessor = visual.NewMermaidProcessor()
	app.mermaidProcessor.SetASCIISize(width-10, 30)

	return nil
}

// addSystemMessage adds a system message to the chat
func (app *App) addSystemMessage(message string) {
	if app.statusPane != nil {
		app.statusPane.AddNotification(message, "info")
	}

	// Also add to messages array if it exists
	if app.messages != nil {
		systemMsg := common.ChatMessage{
			Type:      types.MsgSystem,
			Content:   message,
			Timestamp: time.Now(),
			Metadata:  map[string]string{"source": "daemon"},
		}
		app.messages = append(app.messages, systemMsg)
	}
}

// initializeDaemonConnection establishes connection to the Guild daemon
func (app *App) initializeDaemonConnection() error {
	// Try to connect to daemon
	err := app.connManager.Connect(app.ctx)
	if err != nil {
		app.connectionStatus = false
		app.updateConnectionStatus()
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("chat.app").
			WithOperation("initializeDaemonConnection")
	}

	// Get connection info
	conn, info := app.connManager.GetConnection()
	if conn != nil && info != nil {
		app.grpcConn = conn
		app.connectionInfo = info
		app.connectionStatus = true

		// Create gRPC clients
		app.guildClient = pb.NewGuildClient(conn)
		app.chatClient = pb.NewChatServiceClient(conn)
		app.sessionClient = pb.NewSessionServiceClient(conn)
		app.promptsClient = promptspb.NewPromptServiceClient(conn)

		// Update status display
		app.updateConnectionStatus()

		// Load session from daemon
		if err := app.loadSessionFromDaemon(); err != nil {
			// Log but don't fail
			app.addSystemMessage("Warning: Failed to load session from daemon")
		}
	}

	return nil
}

// updateConnectionStatus updates the UI with current connection status
func (app *App) updateConnectionStatus() {
	if app.statusPane == nil {
		return
	}

	app.statusPane.SetConnectionStatus(app.connectionStatus)

	if app.connectionStatus && app.connectionInfo != nil {
		latency := app.connManager.GetLatency(app.ctx)
		statusMsg := daemonconn.FormatConnectionStatus(app.connectionInfo, latency)
		app.statusPane.UpdateStatus(statusMsg, "success")
	} else {
		app.statusPane.UpdateStatus("🔴 Daemon offline", "error")
	}
}

// loadSessionFromDaemon loads previous session from the daemon
func (app *App) loadSessionFromDaemon() error {
	if app.sessionClient == nil {
		return gerror.New(gerror.ErrCodeNotFound, "session client not available", nil).
			WithComponent("chat.app").
			WithOperation("loadSessionFromDaemon")
	}

	campaignID := app.config.CampaignID
	if campaignID == "" {
		// No campaign specified, create a default session
		return app.createNewSession()
	}

	// Try to find existing session for this campaign
	listReq := &pb.ListSessionsRequest{
		CampaignId: &campaignID,
		Limit:      1, // Get most recent session
	}

	listResp, err := app.sessionClient.ListSessions(app.ctx, listReq)
	if err != nil {
		// Can't list sessions, create new one
		return app.createNewSession()
	}

	var session *pb.Session
	if len(listResp.Sessions) > 0 {
		// Use existing session
		session = listResp.Sessions[0]
		app.addSystemMessage(fmt.Sprintf("Restored session: %s", session.Name))
	} else {
		// Create new session
		return app.createNewSession()
	}

	// Load message history for this session
	return app.loadMessagesFromSession(session.Id)
}

// createNewSession creates a new session for the current campaign
func (app *App) createNewSession() error {
	// Check if session client is available
	if app.sessionClient == nil {
		return gerror.New(gerror.ErrCodeConnection, "session client not available", nil).
			WithComponent("chat.app").
			WithOperation("createNewSession")
	}

	createReq := &pb.CreateSessionRequest{
		Name:       fmt.Sprintf("Chat-%d", time.Now().Unix()),
		CampaignId: &app.config.CampaignID,
		Metadata: map[string]string{
			"client":     "guild-chat",
			"created_by": "user",
		},
	}

	session, err := app.sessionClient.CreateSession(app.ctx, createReq)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create new session").
			WithComponent("chat.app").
			WithOperation("createNewSession")
	}

	app.config.SessionID = session.Id
	app.addSystemMessage(fmt.Sprintf("Created new session: %s", session.Name))
	return nil
}

// loadMessagesFromSession loads message history from a specific session
func (app *App) loadMessagesFromSession(sessionID string) error {
	// Get messages from last 24 hours
	since := timestamppb.New(time.Now().Add(-24 * time.Hour))

	streamReq := &pb.StreamMessagesRequest{
		SessionId: sessionID,
		Since:     since,
	}

	stream, err := app.sessionClient.StreamMessages(app.ctx, streamReq)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to stream messages").
			WithComponent("chat.app").
			WithOperation("loadMessagesFromSession").
			WithDetails("session_id", sessionID)
	}

	var messageCount int
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return gerror.Wrap(err, gerror.ErrCodeConnection, "stream receive error").
				WithComponent("chat.app").
				WithOperation("loadMessagesFromSession")
		}

		// Convert protobuf message to internal format
		chatMsg := app.convertProtoMessage(msg)
		app.messages = append(app.messages, chatMsg)
		messageCount++
	}

	if messageCount > 0 {
		app.addSystemMessage(fmt.Sprintf("Loaded %d messages from previous session", messageCount))
	}

	return nil
}

// convertProtoMessage converts a protobuf Message to internal ChatMessage format
func (app *App) convertProtoMessage(msg *pb.Message) common.ChatMessage {
	var msgType types.MessageType

	switch msg.Role {
	case pb.Message_SYSTEM:
		msgType = types.MsgSystem
	case pb.Message_USER:
		msgType = types.MsgUser
	case pb.Message_ASSISTANT:
		msgType = types.MsgAgent
	case pb.Message_TOOL:
		msgType = types.MsgToolComplete
	default:
		msgType = types.MsgSystem
	}

	return common.ChatMessage{
		Type:      msgType,
		Content:   msg.Content,
		AgentID:   extractAgentID(msg.Metadata),
		Timestamp: msg.CreatedAt.AsTime(),
		Metadata:  msg.Metadata,
	}
}

// extractAgentID extracts agent ID from message metadata
func extractAgentID(metadata map[string]string) string {
	if agentID, ok := metadata["agent_id"]; ok {
		return agentID
	}
	if senderID, ok := metadata["sender_id"]; ok {
		return senderID
	}
	return "system"
}

// enableDirectMode switches to direct orchestrator mode
func (app *App) enableDirectMode() {
	app.directMode = true
	app.connectionStatus = false
	app.updateConnectionStatus()

	// Initialize direct mode components
	if err := app.initializeDirectMode(); err != nil {
		app.addSystemMessage("⚠️ Failed to initialize direct mode: " + err.Error())
		return
	}

	app.addSystemMessage("⚠️ Daemon unavailable, using direct mode")
}

// initializeDirectMode sets up direct orchestrator for when daemon is unavailable
func (app *App) initializeDirectMode() error {
	// Create mock/direct gRPC clients for local operation
	// This would integrate with the existing orchestrator package
	// For now, we'll create a simple local implementation

	// NOTE: This is where we'd integrate with pkg/orchestrator for direct mode
	// The orchestrator package already has local execution capabilities

	app.addSystemMessage("Direct mode initialized - commands will run locally")
	return nil
}

// sendMessageDirect handles message sending in direct mode
func (app *App) sendMessageDirect(ctx context.Context, content string) error {
	if !app.directMode {
		return gerror.New(gerror.ErrCodeInvalidInput, "not in direct mode", nil).
			WithComponent("chat.app").
			WithOperation("sendMessageDirect")
	}

	// Add user message to chat
	userMsg := common.ChatMessage{
		Type:      types.MsgUser,
		Content:   content,
		AgentID:   "user",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"mode": "direct"},
	}
	app.messages = append(app.messages, userMsg)

	// Process message locally using orchestrator
	// This would integrate with the existing orchestrator package
	response := fmt.Sprintf("Direct mode response to: %s", content)

	// Add system response
	responseMsg := common.ChatMessage{
		Type:      types.MsgSystem,
		Content:   response,
		AgentID:   "system",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"mode": "direct"},
	}
	app.messages = append(app.messages, responseMsg)

	return nil
}

// isConnectedToDaemon returns true if connected to daemon, false if in direct mode
func (app *App) isConnectedToDaemon() bool {
	// Check basic connection status
	if !app.connectionStatus || app.directMode {
		return false
	}

	// If we have a connection manager, check its status
	if app.connManager != nil {
		return app.connManager.IsConnected()
	}

	// Otherwise, check if we have gRPC connection
	return app.grpcConn != nil
}

// sendMessage sends a message via daemon or direct mode
func (app *App) sendMessage(ctx context.Context, content string) error {
	if app.isConnectedToDaemon() {
		// Send via daemon
		return app.sendMessageViaDaemon(ctx, content)
	} else {
		// Enable direct mode if not already enabled
		if !app.directMode {
			app.enableDirectMode()
		}
		// Send via direct mode
		return app.sendMessageDirect(ctx, content)
	}
}

// sendMessageViaDaemon sends message through daemon gRPC service
func (app *App) sendMessageViaDaemon(ctx context.Context, content string) error {
	if app.chatClient == nil {
		return gerror.New(gerror.ErrCodeNotFound, "chat client not available", nil).
			WithComponent("chat.app").
			WithOperation("sendMessageViaDaemon")
	}

	// Create bidirectional stream for chat
	stream, err := app.chatClient.Chat(ctx)
	if err != nil {
		// Connection failed, trigger reconnect and fallback
		app.connManager.TriggerReconnect()
		app.enableDirectMode()
		return app.sendMessageDirect(ctx, content)
	}

	// Send chat message
	chatReq := &pb.ChatRequest{
		Request: &pb.ChatRequest_Message{
			Message: &pb.ChatMessage{
				SessionId:  app.config.SessionID,
				SenderId:   "user",
				SenderName: "User",
				Content:    content,
				Type:       pb.ChatMessage_USER_MESSAGE,
				Timestamp:  time.Now().Unix(),
				Metadata:   map[string]string{"client": "guild-chat"},
			},
		},
	}

	if err := stream.Send(chatReq); err != nil {
		// Send failed, fallback to direct mode
		app.enableDirectMode()
		return app.sendMessageDirect(ctx, content)
	}

	// Handle responses (this would be done in a goroutine in real implementation)
	go app.handleChatResponses(stream)

	return nil
}

// handleChatResponses processes streaming responses from daemon
func (app *App) handleChatResponses(stream pb.ChatService_ChatClient) {
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Stream ended or error occurred
			if err.Error() != "EOF" {
				app.addSystemMessage("Connection to daemon lost: " + err.Error())
				app.enableDirectMode()
			}
			return
		}

		// Process response based on type
		switch r := resp.Response.(type) {
		case *pb.ChatResponse_Message:
			// Add message to chat
			msg := app.convertChatMessage(r.Message)
			app.messages = append(app.messages, msg)
		case *pb.ChatResponse_Thinking:
			// Show agent thinking indicator
			app.addSystemMessage(fmt.Sprintf("%s is %s", r.Thinking.AgentName, r.Thinking.State.String()))
		case *pb.ChatResponse_Error:
			// Handle error
			app.addSystemMessage(fmt.Sprintf("Error: %s", r.Error.Message))
		}
	}
}

// convertChatMessage converts gRPC ChatMessage to internal format
func (app *App) convertChatMessage(msg *pb.ChatMessage) common.ChatMessage {
	var msgType types.MessageType

	switch msg.Type {
	case pb.ChatMessage_USER_MESSAGE:
		msgType = types.MsgUser
	case pb.ChatMessage_AGENT_RESPONSE:
		msgType = types.MsgAgent
	case pb.ChatMessage_SYSTEM_MESSAGE:
		msgType = types.MsgSystem
	case pb.ChatMessage_TOOL_REQUEST:
		msgType = types.MsgToolStart
	case pb.ChatMessage_TOOL_RESULT:
		msgType = types.MsgToolComplete
	default:
		msgType = types.MsgSystem
	}

	return common.ChatMessage{
		Type:      msgType,
		Content:   msg.Content,
		AgentID:   msg.SenderId,
		Timestamp: time.Unix(msg.Timestamp, 0),
		Metadata:  msg.Metadata,
	}
}

// getCompletionStrings extracts string content from completion results
func getCompletionStrings(completions []completion.CompletionResult) []string {
	strings := make([]string, len(completions))
	for i, comp := range completions {
		strings[i] = comp.Content
	}
	return strings
}
