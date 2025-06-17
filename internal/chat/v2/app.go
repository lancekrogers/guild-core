// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package v2

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	_ "modernc.org/sqlite" // SQLite driver

	"github.com/guild-ventures/guild-core/internal/chat/v2/layout"
	"github.com/guild-ventures/guild-core/internal/chat/v2/panes"
	"github.com/guild-ventures/guild-core/internal/chat/v2/services"
	"github.com/guild-ventures/guild-core/internal/chat/v2/utils"
	"github.com/guild-ventures/guild-core/internal/chat/session"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/templates"
)

// App represents the main chat application
type App struct {
	// Core configuration
	ctx      context.Context
	config   *ChatConfig
	
	// Layout and panes
	layoutManager *layout.Manager
	outputPane    panes.OutputPane
	inputPane     panes.InputPane
	statusPane    panes.StatusPane
	
	// Services
	chatService     *services.ChatService
	daemonService   *services.DaemonService
	providerService *services.ProviderService
	
	// Utilities
	styles *utils.Styles
	keys   *utils.KeyBindings
	
	// gRPC clients
	grpcConn      *grpc.ClientConn
	guildClient   pb.GuildClient
	promptsClient promptspb.PromptServiceClient
	registry      registry.ComponentRegistry
	
	// Session management
	sessionManager session.SessionManager
	currentSession *session.Session
	
	// Application state
	messages       []ChatMessage
	activeTools    map[string]*ToolExecution
	agents         []string
	currentView    ViewMode
	searchResults  []int
	searchPattern  string
	completions    []CompletionResult
	
	// Command processing
	commandProcessor *CommandProcessor
	commandHistory   *CommandHistory
	templateManager  templates.TemplateManager
	completionEngine *CompletionEngine
	
	// Feature flags
	initialized bool
	ready       bool
	shouldQuit  bool
	errorState  error
}

// NewApp creates a new chat application (simplified wrapper)
func NewApp(ctx context.Context, guildConfig *config.GuildConfig, 
	conn *grpc.ClientConn, guildClient pb.GuildClient, 
	promptsClient promptspb.PromptServiceClient, 
	registry registry.ComponentRegistry) *App {
	
	// Create basic app structure
	app := &App{
		ctx:           ctx,
		grpcConn:      conn,
		guildClient:   guildClient,
		promptsClient: promptsClient,
		registry:      registry,
		messages:      make([]ChatMessage, 0),
		activeTools:   make(map[string]*ToolExecution),
		agents:        make([]string, 0),
		currentView:   ViewModeNormal,
	}
	
	// Store guild config for later initialization
	app.config = &ChatConfig{
		GuildConfig: guildConfig,
		Width:       80,
		Height:      24,
	}
	
	return app
}

// Run starts the chat application
func (app *App) Run() error {
	// Initialize components during run
	if err := app.initializeComponents(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize chat").
			WithComponent("chat.v2").
			WithOperation("Run")
	}
	
	// Create and run the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run chat interface").
			WithComponent("chat.v2").
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
	app.commandHistory = NewCommandHistory(1000)
	
	// Initialize session management first
	if err := app.initializeSessionManagement(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize session management").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}
	
	// Initialize template manager (for now, nil - would need database setup)
	app.templateManager = nil
	
	// Initialize completion engine
	projectRoot := "." // TODO: Get actual project root
	app.completionEngine = NewCompletionEngine(app.config.GuildConfig, projectRoot)
	if app.registry != nil {
		app.completionEngine.SetRegistry(app.registry)
	}
	
	// Initialize command processor with session manager and template manager
	app.commandProcessor = NewCommandProcessor(app.ctx, app.config, app.commandHistory, 
		app.sessionManager, app.currentSession, app.templateManager)
	
	// Initialize services
	if err := app.initializeServices(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize services").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}
	
	// Initialize layout manager
	app.layoutManager = layout.NewManager(app.config.Width, app.config.Height)
	
	// Initialize panes
	if err := app.initializePanes(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize panes").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
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
	// Initialize chat service
	chatService, err := services.NewChatService(app.ctx, app.guildClient, app.registry)
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
	
	// Initialize input pane
	inputPane, err := panes.NewInputPane(inputRect.Width, inputRect.Height, app.config.EnableCompletion)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create input pane").
			WithComponent("chat.app").
			WithOperation("initializePanes")
	}
	app.inputPane = inputPane
	
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

// convertSessionMessage converts a session message to a chat message
func (app *App) convertSessionMessage(msg *session.Message) ChatMessage {
	var msgType MessageType
	var agentID string
	
	switch msg.Role {
	case session.RoleUser:
		msgType = MsgUser
	case session.RoleAssistant:
		msgType = MsgAgent
		// Extract agent ID from metadata if available
		if msg.Metadata != nil {
			if id, ok := msg.Metadata["agent_id"].(string); ok {
				agentID = id
			}
		}
	case session.RoleSystem:
		msgType = MsgSystem
	default:
		msgType = MsgSystem
	}
	
	return ChatMessage{
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
			return StatusUpdateMsg{
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
		
	case AgentStreamMsg:
		return app.handleAgentStream(msg)
		
	case StatusUpdateMsg:
		return app.handleStatusUpdate(msg)
		
	case LayoutUpdateMsg:
		return app.handleLayoutUpdate(msg)
		
	case PaneUpdateMsg:
		return app.handlePaneUpdate(msg)
		
	case ViewModeChangeMsg:
		return app.handleViewModeChange(msg)
		
	case ToolExecutionStartMsg:
		return app.handleToolExecutionStart(msg)
		
	case ToolExecutionProgressMsg:
		return app.handleToolExecutionProgress(msg)
		
	case ToolExecutionCompleteMsg:
		return app.handleToolExecutionComplete(msg)
		
	case struct {
		Type  string
		Input string
	}:
		if msg.Type == "completion_request" {
			return app.handleCompletionRequestAnon(msg.Input)
		}
		
	case CompletionResultMsg:
		return app.handleCompletionResult(msg)
		
	case SearchMsg:
		return app.handleSearch(msg)
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
	
	updatedModel, paneCmd = app.inputPane.Update(msg)
	if inputPane, ok := updatedModel.(panes.InputPane); ok {
		app.inputPane = inputPane
	}
	if paneCmd != nil {
		cmds = append(cmds, paneCmd)
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
func (app *App) generateWelcomeMessage() ChatMessage {
	content := `🏰 ═══════════════════════════════════════════ 🏰
   Welcome to the Guild Chat Chamber!

   ⚔️  Your agents await your commands
   🛡️  Type /help to see available commands
   👑  Use @agent-name to message specific agents
   📜  Use @all to broadcast to all agents

   Ready to craft great software together!
🏰 ═══════════════════════════════════════════ 🏰

Rich content rendering is ACTIVE! ✨
Try these commands to see visual features:
• /test markdown - See styled headers and formatting
• /test code go - View syntax highlighted code
• /status - View real-time agent status panel`

	return ChatMessage{
		Type:      MsgSystem,
		Content:   content,
		AgentID:   "system",
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
}

// Event handlers - these will be implemented as the components are built

func (app *App) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// Update config dimensions
	configManager := NewConfigManager(app.ctx)
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
	
	return app, tea.Batch(cmds...)
}

func (app *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle global shortcuts first
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
	
	// Process input through command processor
	isCommand, cmd := app.commandProcessor.ProcessInput(input)
	
	// Add user message to output if it's not a command
	if !isCommand {
		userMsg := ChatMessage{
			Type:      MsgUser,
			Content:   input,
			AgentID:   "user",
			Timestamp: app.GetCurrentTime(),
			Metadata:  make(map[string]string),
		}
		app.outputPane.AddMessage(userMsg)
	}
	
	return app, cmd
}

func (app *App) handleCommandPalette() (tea.Model, tea.Cmd) {
	// TODO: Implement command palette
	return app, nil
}

func (app *App) handleGlobalSearch() (tea.Model, tea.Cmd) {
	// TODO: Implement global search
	return app, nil
}

func (app *App) handleHelp() (tea.Model, tea.Cmd) {
	return app, app.commandProcessor.processCommand("help")
}

func (app *App) handleAgentStream(msg AgentStreamMsg) (tea.Model, tea.Cmd) {
	// Add agent message to output
	chatMsg := ChatMessage{
		Type:      MsgAgent,
		Content:   msg.Content,
		AgentID:   msg.AgentID,
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(chatMsg)
	
	return app, nil
}

func (app *App) handleStatusUpdate(msg StatusUpdateMsg) (tea.Model, tea.Cmd) {
	app.statusPane.UpdateStatus(msg.Message, msg.Level)
	return app, nil
}

func (app *App) handleLayoutUpdate(msg LayoutUpdateMsg) (tea.Model, tea.Cmd) {
	app.layoutManager.Resize(msg.Width, msg.Height)
	return app, nil
}

func (app *App) handlePaneUpdate(msg PaneUpdateMsg) (tea.Model, tea.Cmd) {
	switch msg.PaneID {
	case "output":
		if msg.Data == "clear" {
			app.outputPane.Clear()
		} else if msg.Content != "" {
			systemMsg := ChatMessage{
				Type:      MsgSystem,
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

func (app *App) handleViewModeChange(msg ViewModeChangeMsg) (tea.Model, tea.Cmd) {
	app.currentView = msg.Mode
	return app, nil
}

func (app *App) handleToolExecutionStart(msg ToolExecutionStartMsg) (tea.Model, tea.Cmd) {
	// Track tool execution
	execution := &ToolExecution{
		ID:          msg.ExecutionID,
		ToolName:    msg.ToolName,
		AgentID:     msg.AgentID,
		StartTime:   app.GetCurrentTime(),
		Status:      "starting",
		Metadata:    map[string]string{"parameters": fmt.Sprintf("%v", msg.Parameters)},
	}
	app.activeTools[msg.ExecutionID] = execution
	
	// Add tool start message
	toolMsg := ChatMessage{
		Type:      MsgToolStart,
		Content:   fmt.Sprintf("🔨 Starting tool: %s", msg.ToolName),
		AgentID:   msg.AgentID,
		Timestamp: app.GetCurrentTime(),
		Metadata:  make(map[string]string),
	}
	app.outputPane.AddMessage(toolMsg)
	
	return app, nil
}

func (app *App) handleToolExecutionProgress(msg ToolExecutionProgressMsg) (tea.Model, tea.Cmd) {
	if execution, exists := app.activeTools[msg.ExecutionID]; exists {
		execution.Progress = msg.Progress
		execution.Status = "running"
	}
	return app, nil
}

func (app *App) handleToolExecutionComplete(msg ToolExecutionCompleteMsg) (tea.Model, tea.Cmd) {
	if execution, exists := app.activeTools[msg.ExecutionID]; exists {
		now := app.GetCurrentTime()
		execution.EndTime = &now
		execution.Status = "completed"
		execution.Result = msg.Result
		
		// Add completion message
		toolMsg := ChatMessage{
			Type:      MsgToolComplete,
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
			return CompletionResultMsg{
				Results: results,
			}
		}
	}
	
	return app, nil
}

func (app *App) handleCompletionResult(msg CompletionResultMsg) (tea.Model, tea.Cmd) {
	app.completions = msg.Results
	app.inputPane.ShowCompletions(msg.Results)
	return app, nil
}

func (app *App) handleSearch(msg SearchMsg) (tea.Model, tea.Cmd) {
	app.searchPattern = msg.Pattern
	app.searchResults = msg.Results
	// TODO: Update UI to show search results
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