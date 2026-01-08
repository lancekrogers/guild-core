// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/guild-framework/guild-core/internal/daemonconn"
	"github.com/guild-framework/guild-core/internal/ui/chat/agents"
	"github.com/guild-framework/guild-core/internal/ui/chat/commands"
	"github.com/guild-framework/guild-core/internal/ui/chat/common"
	cfig "github.com/guild-framework/guild-core/internal/ui/chat/common/config"
	"github.com/guild-framework/guild-core/internal/ui/chat/common/layout"
	"github.com/guild-framework/guild-core/internal/ui/chat/common/types"
	"github.com/guild-framework/guild-core/internal/ui/chat/common/utils"
	"github.com/guild-framework/guild-core/internal/ui/chat/completion"
	"github.com/guild-framework/guild-core/internal/ui/chat/managers"
	"github.com/guild-framework/guild-core/internal/ui/chat/messages"
	toolmsg "github.com/guild-framework/guild-core/internal/ui/chat/messages/tools"
	"github.com/guild-framework/guild-core/internal/ui/chat/panes"
	"github.com/guild-framework/guild-core/internal/ui/chat/services"
	"github.com/guild-framework/guild-core/internal/ui/chat/session"
	"github.com/guild-framework/guild-core/internal/ui/formatting"
	uitools "github.com/guild-framework/guild-core/internal/ui/tools"
	viewutil "github.com/guild-framework/guild-core/internal/ui/view"
	"github.com/guild-framework/guild-core/internal/ui/vim"
	"github.com/guild-framework/guild-core/internal/ui/visual"
	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/campaign"
	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-framework/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/paths"
	"github.com/guild-framework/guild-core/pkg/preferences"
	"github.com/guild-framework/guild-core/pkg/project"
	globalproj "github.com/guild-framework/guild-core/pkg/project/global"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/registry"
	pkgsession "github.com/guild-framework/guild-core/pkg/session"
	"github.com/guild-framework/guild-core/pkg/storage"
	"github.com/guild-framework/guild-core/pkg/templates"
	"github.com/guild-framework/guild-core/pkg/tools"

	"github.com/guild-framework/guild-core/tools/search"
)

const (
	daemonStartupRPCTimeout = 2 * time.Second
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

	// Enhanced state management (Sprint 4)
	stateManager   *StateManager
	stateHistory   []AppState
	stateListeners []StateListener

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

	// Sprint 2: Persistence & State Management
	prefService     *preferences.Service
	storageRegistry storage.StorageRegistry
	sessionStore    pkgsession.SessionStore
	recoveryManager *pkgsession.RecoveryManager
	multiSessionMgr *pkgsession.MultiSessionManager
	sessionResumer  *pkgsession.SessionResumer
	pendingRecovery []pkgsession.CheckpointInfo

	// Feature flags
	initialized bool
	ready       bool
	shouldQuit  bool
	useAltScreen bool
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

		// UI defaults
		Width:           80,
		Height:          24,
		MarkdownEnabled: true,
		WrapLines:       true,

		// Feature defaults (enabled)
		EnableCompletion:    true,
		EnableHistory:       true,
		EnableStatusDisplay: true,
		EnableRichContent:   true,

		// Theme defaults
		Theme:          "dark",
		FontSize:       12,
		ColorScheme:    "default",
		ShowTimestamps: true,
		CompactMode:    false,

		// AI defaults
		DefaultProvider:  "anthropic",
		DefaultModel:     "claude-3-sonnet-20240229",
		Temperature:      0.7,
		MaxTokens:        4096,
		StreamingEnabled: true,
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

// SetUserID sets the user ID for preferences (Sprint 2)
func (app *App) SetUserID(userID string) {
	if app.config != nil {
		app.config.UserID = userID
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

	// Create and run the Bubble Tea program.
	//
	// Prefer a dedicated TTY for interactive I/O so chat can run even when
	// stdin/stdout are redirected, and so stray prints to stdio can't corrupt the
	// TUI renderer.
	opts := []tea.ProgramOption{
		tea.WithContext(app.ctx),
	}

	var ttyOut *os.File
	if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		ttyOut = tty
		defer ttyOut.Close()

		// Prevent any stray writes to stdout/stderr from corrupting the TUI by
		// temporarily redirecting them away from the terminal. Disable this for
		// debugging by setting `GUILD_CHAT_DEBUG_STDIO=1`.
		if os.Getenv("GUILD_CHAT_DEBUG_STDIO") == "" {
			if restore, err := suppressStdIO(); err == nil {
				defer restore()
			}
		}

		opts = append(opts,
			tea.WithInput(ttyOut),
			tea.WithOutput(ttyOut),
		)
		app.useAltScreen = true
	} else if term.IsTerminal(int(os.Stdout.Fd())) {
		app.useAltScreen = true
	} else {
		// No usable terminal: fall back to simple output mode.
		opts = append(opts, tea.WithoutRenderer())
	}

	p := tea.NewProgram(app, opts...)
	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run chat interface").
			WithComponent("chat.core").
			WithOperation("Run")
	}

	return nil
}

// initializeComponents initializes all application components
func (app *App) initializeComponents() error {
	// Ensure config is complete before any component init that relies on it.
	if err := app.ensureChatConfigDefaults(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build chat configuration").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize utilities
	app.styles = utils.NewStyles()
	app.keys = utils.NewKeyBindings()

	// Initialize command history
	app.commandHistory = commands.NewCommandHistory(1000)

	// Initialize state manager (Sprint 4)
	if err := app.initializeStateManager(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize state manager").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize visual components
	if err := app.initializeVisualComponents(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize visual components").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize session management with Sprint 2 enhancements
	if err := app.initializeSessionManagement(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize session management").
			WithComponent("chat.app").
			WithOperation("initializeComponents")
	}

	// Initialize preferences service (Sprint 2)
	if err := app.initializePreferences(); err != nil {
		// Log but don't fail - preferences are optional
		observability.GetLogger(app.ctx).Warn("Failed to initialize preferences", "error", err)
	}

	// Initialize daemon-backed or local backend before services/routers.
	if err := app.initializeBackend(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize backend").
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
	projectRoot := app.config.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

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

	// Set preferences service on command processor (Sprint 2)
	if app.prefService != nil && app.commandProcessor != nil {
		app.commandProcessor.SetPreferencesService(app.prefService)
	}

	// Connect completion engine to command processor for live command updates
	if app.completionEngine != nil && app.commandProcessor != nil {
		app.completionEngine.SetCommandProcessor(app.commandProcessor)
	}

	// Initialize services (optional in direct mode / global mode)
	if err := app.initializeServices(); err != nil {
		observability.GetLogger(app.ctx).Warn("Failed to initialize chat services", "error", err)
	}

	// Initialize agent router (requires backend)
	if app.guildClient != nil {
		app.agentRouter = agents.NewAgentRouter(app.ctx, app.guildClient)

		// Set agent router on command processor for campaign commands
		if app.commandProcessor != nil {
			app.commandProcessor.SetAgentRouter(app.agentRouter)
		}
	}

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

	// Update connection status display now that panes exist.
	app.updateConnectionStatus()

	app.initialized = true
	return nil
}

func (app *App) ensureChatConfigDefaults() error {
	if app.config == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "chat config is nil", nil).
			WithComponent("chat.app").
			WithOperation("ensureChatConfigDefaults")
	}

	// Basic UI defaults
	if app.config.Width <= 0 {
		app.config.Width = 80
	}
	if app.config.Height <= 0 {
		app.config.Height = 24
	}

	// Feature defaults (opt-out later via preferences)
	app.config.MarkdownEnabled = true
	app.config.WrapLines = true
	app.config.EnableCompletion = true
	app.config.EnableHistory = true
	app.config.EnableStatusDisplay = true
	app.config.EnableRichContent = true

	// Ensure we have a usable project root (used for tools/search/completions)
	if app.config.ProjectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			app.config.ProjectRoot = cwd
		} else {
			app.config.ProjectRoot = "."
		}
	}

	// Resolve campaign root if present; otherwise, run in global mode.
	if root, err := project.FindProjectRoot(app.config.ProjectRoot); err == nil && root != "" {
		app.config.ProjectRoot = root
		if app.config.DatabasePath == "" {
			app.config.DatabasePath = filepath.Join(root, paths.DefaultCampaignDir, paths.DefaultMemoryDB)
		}
		if app.config.HistoryPath == "" {
			app.config.HistoryPath = filepath.Join(root, paths.DefaultCampaignDir, "chat_history.txt")
		}
		if app.config.ConfigPath == "" {
			app.config.ConfigPath = filepath.Join(root, paths.DefaultCampaignDir, "campaign.yaml")
		}
	} else {
		// Global mode uses the global Guild directory for state.
		globalDir := globalproj.GlobalGuildDir()
		if err := globalproj.EnsureGlobalInitialized(); err != nil {
			// Don't block chat startup on global initialization issues.
			// Fall back to in-memory persistence and disable file-based history.
			if app.config.DatabasePath == "" {
				app.config.DatabasePath = ":memory:"
			}
			if app.config.HistoryPath == "" {
				app.config.HistoryPath = ""
			}
			if app.config.ConfigPath == "" {
				app.config.ConfigPath = filepath.Join(globalDir, "config.yaml")
			}
		} else {
			if app.config.DatabasePath == "" {
				app.config.DatabasePath = filepath.Join(globalDir, paths.DefaultMemoryDB)
			}
			if app.config.HistoryPath == "" {
				app.config.HistoryPath = filepath.Join(globalDir, "chat_history.txt")
			}
			if app.config.ConfigPath == "" {
				app.config.ConfigPath = filepath.Join(globalDir, "config.yaml")
			}
		}
	}

	// Ensure GuildConfig exists (NewApp should set it, but keep safe defaults).
	if app.config.GuildConfig == nil {
		app.config.GuildConfig = config.DefaultGuildTemplate()
	}

	return nil
}

func (app *App) initializeBackend() error {
	// Prefer daemon when available, but always fall back to direct mode.
	if app.config != nil && app.config.CampaignID != "" {
		if err := app.initializeDaemonConnection(); err == nil {
			app.directMode = false
			return nil
		}
	}

	app.directMode = true
	app.connectionStatus = false
	return app.initializeDirectMode()
}

// initializeSessionManagement initializes enhanced session persistence from Sprint 2
func (app *App) initializeSessionManagement() error {
	// Initialize storage registry if not provided
	if app.storageRegistry == nil {
		storageReg, memStore, err := storage.InitializeSQLiteStorageForRegistry(app.ctx, app.config.DatabasePath)
		if err != nil {
			// Fall back to an in-memory DB so the chat UI can still start.
			storageReg, memStore, err = storage.InitializeSQLiteStorageForRegistry(app.ctx, ":memory:")
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize storage registry").
					WithComponent("chat.app").
					WithOperation("initializeSessionManagement")
			}
			app.config.DatabasePath = ":memory:"
		}
		app.storageRegistry = storageReg
		// memStore is available if needed for memory operations
		_ = memStore
	}

	// Get session repository from storage registry
	sessionRepo := app.storageRegistry.GetSessionRepository()
	if sessionRepo == nil {
		return gerror.New(gerror.ErrCodeInternal, "session repository not available", nil).
			WithComponent("chat.app").
			WithOperation("initializeSessionManagement")
	}

	// Create session store adapter for pkg/session
	app.sessionStore = &sessionStoreAdapter{
		repo: sessionRepo,
		ctx:  app.ctx,
	}

	// Create enhanced session manager with Sprint 2 features
	sessionManager := pkgsession.NewSessionManager(app.sessionStore,
		pkgsession.WithAutoSaveInterval(30*time.Second))

	// Create recovery manager for crash recovery (skip for in-memory DBs)
	if app.config.DatabasePath != ":memory:" {
		recoveryDir := filepath.Join(filepath.Dir(app.config.DatabasePath), "recovery")
		recoveryManager, err := pkgsession.NewRecoveryManager(sessionManager, recoveryDir)
		if err != nil {
			// Log but don't fail - recovery is optional
			observability.GetLogger(app.ctx).Warn("Failed to create recovery manager", "error", err)
		}
		app.recoveryManager = recoveryManager
	}

	// Create multi-session manager
	app.multiSessionMgr = pkgsession.NewMultiSessionManager(sessionManager, nil, 10)

	// Create session resumer (we'll create adapters for UI integration later)
	app.sessionResumer = pkgsession.NewSessionResumer(sessionManager, nil, nil, nil)

	// Check for crash recovery before creating new session
	if app.recoveryManager != nil {
		checkpoints := app.recoveryManager.GetCheckpointInfo()
		if len(checkpoints) > 0 {
			// We'll handle recovery prompt in Init() method
			app.pendingRecovery = checkpoints
		}
	}

	// Create compatibility wrapper for existing code
	app.sessionManager = &sessionManagerAdapter{
		multiSessionMgr: app.multiSessionMgr,
		sessionManager:  sessionManager,
	}

	// Load or create session
	if app.config.SessionID != "" {
		// Try to switch to existing session
		session, err := app.multiSessionMgr.SwitchSession(app.ctx, app.config.SessionID)
		if err != nil {
			// Create new session if switch fails
			session, err = app.multiSessionMgr.CreateSession(app.ctx, app.config.UserID, app.config.CampaignID)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create session").
					WithComponent("chat.app").
					WithOperation("initializeSessionManagement")
			}
			app.config.SessionID = session.ID
		}
		app.currentSession = convertPkgSession(session)
	}

	// Load existing messages if we have a session
	if app.currentSession != nil {
		messages, err := app.multiSessionMgr.GetSessionMessages(app.currentSession.ID, 50)
		if err == nil && len(messages) > 0 {
			// Convert session messages to chat messages
			for _, msg := range messages {
				chatMsg := app.convertPkgSessionMessage(msg)
				if app.stateManager != nil {
					app.addMessageWithState(chatMsg)
				} else {
					app.messages = append(app.messages, chatMsg)
				}
			}
		}
	}

	return nil
}

// initializePreferences initializes the preferences service from Sprint 2
func (app *App) initializePreferences() error {
	// Get preferences repository from storage registry
	if app.storageRegistry == nil {
		// Storage registry should be initialized by session management
		return gerror.New(gerror.ErrCodeInternal, "storage registry not initialized", nil).
			WithComponent("chat.app").
			WithOperation("initializePreferences")
	}

	prefRepo := app.storageRegistry.GetPreferencesRepository()
	if prefRepo == nil {
		return gerror.New(gerror.ErrCodeInternal, "preferences repository not available", nil).
			WithComponent("chat.app").
			WithOperation("initializePreferences")
	}

	// Create preferences service
	app.prefService = preferences.NewService(prefRepo)

	// Load user preferences into config
	if app.config.UserID != "" {
		configManager := cfig.NewConfigManagerWithPreferences(app.ctx, app.prefService)

		// Reload config with preferences
		enhancedConfig, err := configManager.LoadChatConfig(app.ctx, app.config.UserID,
			app.config.CampaignID, app.config.SessionID)
		if err == nil {
			// Preserve core runtime paths/IDs but apply preference fields.
			enhancedConfig.Width = app.config.Width
			enhancedConfig.Height = app.config.Height
			enhancedConfig.ProjectRoot = app.config.ProjectRoot
			enhancedConfig.DatabasePath = app.config.DatabasePath
			enhancedConfig.HistoryPath = app.config.HistoryPath
			enhancedConfig.ConfigPath = app.config.ConfigPath
			enhancedConfig.GuildConfig = app.config.GuildConfig
			app.config = enhancedConfig
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
							// No agent mention: prefer the configured manager, otherwise broadcast.
							defaultAgentID := ""
							if app.config != nil && app.config.GuildConfig != nil {
								if app.config.GuildConfig.Manager.Override != "" {
									defaultAgentID = app.config.GuildConfig.Manager.Override
								} else if app.config.GuildConfig.Manager.Default != "" {
									defaultAgentID = app.config.GuildConfig.Manager.Default
								} else if len(app.config.GuildConfig.Agents) > 0 {
									defaultAgentID = app.config.GuildConfig.Agents[0].ID
								}
							}

							if defaultAgentID == "" || defaultAgentID == "all" {
								cmd = app.agentRouter.BroadcastToAll(input)
							} else {
								cmd = app.agentRouter.SendToAgent(defaultAgentID, input)
							}
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

// Sprint 2: Crash Recovery Methods

// promptForRecovery creates a recovery prompt for the user
func (app *App) promptForRecovery() tea.Cmd {
	return func() tea.Msg {
		// Show recovery information
		recoveryInfo := fmt.Sprintf("🔄 Previous session detected:\n")
		for _, checkpoint := range app.pendingRecovery {
			recoveryInfo += fmt.Sprintf("  • Session %s (last active: %s)\n",
				checkpoint.SessionID[:8],
				checkpoint.Timestamp.Format("Jan 2 15:04"))
		}
		recoveryInfo += "\nWould you like to recover? Type /recover to restore or /new to start fresh."

		return common.ChatMessage{
			Type:      common.MsgSystem,
			Content:   recoveryInfo,
			AgentID:   "system",
			Timestamp: time.Now(),
			Metadata:  map[string]string{"type": "recovery_prompt"},
		}
	}
}

// handleRecoveryCommand handles the user's recovery decision
func (app *App) handleRecoveryCommand(recover bool) tea.Cmd {
	if !recover || app.recoveryManager == nil || len(app.pendingRecovery) == 0 {
		// Clear pending recovery and continue normally
		app.pendingRecovery = nil
		return nil
	}

	// Recover the most recent session
	checkpoint := app.pendingRecovery[0]
	return func() tea.Msg {
		session, err := app.recoveryManager.RecoverSession(app.ctx, checkpoint.SessionID)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to recover session: %v", err),
				Level:   "error",
			}
		}

		// Update app state with recovered session
		app.currentSession = convertPkgSession(session)
		app.config.SessionID = session.ID

		// Clear pending recovery
		app.pendingRecovery = nil

		return panes.StatusUpdateMsg{
			Message: "✅ Session recovered successfully",
			Level:   "success",
		}
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

	// Check for crash recovery
	if len(app.pendingRecovery) > 0 {
		// Create recovery prompt
		cmds = append(cmds, app.promptForRecovery())
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

	case common.RecoveryCommandMsg:
		// Handle recovery decision
		cmd := app.handleRecoveryCommand(msg.Recover)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return app, tea.Batch(cmds...)

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
func (app *App) View() tea.View {
	if !app.ready {
		view := tea.NewView("Initializing Guild Chat...")
		view.AltScreen = app.useAltScreen
		return view
	}

	if app.shouldQuit {
		view := tea.NewView("Goodbye! 🏰")
		view.AltScreen = app.useAltScreen
		return view
	}

	if app.errorState != nil {
		view := tea.NewView(fmt.Sprintf("Error: %v", app.errorState))
		view.AltScreen = app.useAltScreen
		return view
	}

	// Get pane views
	outputView := viewutil.String(app.outputPane.View())
	inputView := viewutil.String(app.inputPane.View())
	statusView := viewutil.String(app.statusPane.View())

	// Use layout manager to compose the final view
	view := tea.NewView(app.layoutManager.Render(map[string]string{
		"output": outputView,
		"input":  inputView,
		"status": statusView,
	}))
	view.AltScreen = app.useAltScreen
	return view
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

			batch := tea.Batch(cmds...)
			if batch == nil {
				return nil
			}

			// Schedule next refresh in 10 seconds
			time.AfterFunc(10*time.Second, func() {
				// This would trigger the next refresh cycle in a real implementation
				// For now, we'll leave it as a placeholder since we need proper
				// message scheduling in the TUI framework
			})

			return batch()
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
		observability.GetLogger(app.ctx).Warn("Failed to get LLM client for suggestions", "error", err)
		return nil
	}

	// Get memory manager
	memoryManager, err := app.getMemoryManager()
	if err != nil {
		// Continue without suggestions if memory manager fails
		observability.GetLogger(app.ctx).Warn("Failed to get memory manager for suggestions", "error", err)
		return nil
	}

	// Get tool registry
	toolRegistry, err := app.getToolRegistry()
	if err != nil {
		// Continue without suggestions if tool registry fails
		observability.GetLogger(app.ctx).Warn("Failed to get tool registry for suggestions", "error", err)
		return nil
	}

	// Get commission manager
	commissionManager, err := app.getCommissionManager()
	if err != nil {
		// Continue without suggestions if commission manager fails
		observability.GetLogger(app.ctx).Warn("Failed to get commission manager for suggestions", "error", err)
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
		if app.stateManager != nil {
			app.addMessageWithState(systemMsg)
		} else {
			app.messages = append(app.messages, systemMsg)
		}
	}
}

// initializeDaemonConnection establishes connection to the Guild daemon
func (app *App) initializeDaemonConnection() error {
	// Try to connect to daemon
	err := app.connManager.ConnectForCampaign(app.ctx, app.config.CampaignID)
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

	rpcCtx, cancel := context.WithTimeout(app.ctx, daemonStartupRPCTimeout)
	defer cancel()

	listResp, err := app.sessionClient.ListSessions(rpcCtx, listReq)
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

	rpcCtx, cancel := context.WithTimeout(app.ctx, daemonStartupRPCTimeout)
	defer cancel()

	session, err := app.sessionClient.CreateSession(rpcCtx, createReq)
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

	rpcCtx, cancel := context.WithTimeout(app.ctx, daemonStartupRPCTimeout)
	defer cancel()

	resp, err := app.sessionClient.GetMessagesAfter(rpcCtx, &pb.GetMessagesAfterRequest{
		SessionId: sessionID,
		After:     since,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to load messages").
			WithComponent("chat.app").
			WithOperation("loadMessagesFromSession").
			WithDetails("session_id", sessionID)
	}

	var messageCount int
	for _, msg := range resp.Messages {
		// Convert protobuf message to internal format
		chatMsg := app.convertProtoMessage(msg)
		if app.stateManager != nil {
			app.addMessageWithState(chatMsg)
		} else {
			app.messages = append(app.messages, chatMsg)
		}
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
	client, err := newLocalGuildClient(app.config.GuildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize local guild client").
			WithComponent("chat.app").
			WithOperation("initializeDirectMode")
	}

	app.guildClient = client
	app.chatClient = nil
	app.sessionClient = nil
	app.promptsClient = nil

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
	if app.stateManager != nil {
		app.addMessageWithState(userMsg)
	} else {
		app.messages = append(app.messages, userMsg)
	}

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
	if app.stateManager != nil {
		app.addMessageWithState(responseMsg)
	} else {
		app.messages = append(app.messages, responseMsg)
	}

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
			if app.stateManager != nil {
				app.addMessageWithState(msg)
			} else {
				app.messages = append(app.messages, msg)
			}
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

// Sprint 2 Integration: Adapter types for compatibility

// sessionStoreAdapter adapts storage.SessionRepository to pkgsession.SessionStore interface
type sessionStoreAdapter struct {
	repo storage.SessionRepository
	ctx  context.Context
}

func (s *sessionStoreAdapter) GetSession(ctx context.Context, sessionID string) (*storage.ChatSession, error) {
	return s.repo.GetSession(ctx, sessionID)
}

func (s *sessionStoreAdapter) UpsertSession(ctx context.Context, session *storage.ChatSession, stateData []byte) error {
	// Store state data in metadata
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}
	session.Metadata["state_data"] = string(stateData)
	return s.repo.UpdateSession(ctx, session)
}

func (s *sessionStoreAdapter) SaveMessage(ctx context.Context, sessionID string, message *storage.ChatMessage) error {
	// Ensure message has the session ID set
	message.SessionID = sessionID
	return s.repo.SaveMessage(ctx, message)
}

func (s *sessionStoreAdapter) GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error) {
	return s.repo.GetMessages(ctx, sessionID)
}

func (s *sessionStoreAdapter) ListSessions(ctx context.Context, options pkgsession.ListOptions) ([]*storage.ChatSession, error) {
	// Convert options to storage format
	return s.repo.ListSessions(ctx, int32(options.Limit), int32(options.Offset))
}

func (s *sessionStoreAdapter) Begin() (pkgsession.Transaction, error) {
	// For now, return a no-op transaction
	return &noOpTransaction{store: s}, nil
}

// noOpTransaction provides a simple transaction implementation
type noOpTransaction struct {
	store *sessionStoreAdapter
}

func (t *noOpTransaction) UpsertSession(session *storage.ChatSession, stateData []byte) error {
	return t.store.UpsertSession(context.Background(), session, stateData)
}

func (t *noOpTransaction) SaveMessage(sessionID string, message *storage.ChatMessage) error {
	return t.store.SaveMessage(context.Background(), sessionID, message)
}

func (t *noOpTransaction) Commit() error {
	return nil
}

func (t *noOpTransaction) Rollback() error {
	return nil
}

// sessionManagerAdapter adapts the new multi-session manager to the old interface
type sessionManagerAdapter struct {
	multiSessionMgr *pkgsession.MultiSessionManager
	sessionManager  *pkgsession.SessionManager
}

func (s *sessionManagerAdapter) LoadSession(sessionID string) (*session.Session, error) {
	sess, err := s.multiSessionMgr.SwitchSession(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	return convertPkgSession(sess), nil
}

func (s *sessionManagerAdapter) NewSession(name string, campaignID *string) (*session.Session, error) {
	userID := "default" // TODO: Get from context or config
	campaign := ""
	if campaignID != nil {
		campaign = *campaignID
	}
	sess, err := s.multiSessionMgr.CreateSession(context.Background(), userID, campaign)
	if err != nil {
		return nil, err
	}
	return convertPkgSession(sess), nil
}

func (s *sessionManagerAdapter) SaveSession(session *session.Session) error {
	// Convert back to pkg session format
	campaignID := ""
	if session.CampaignID != nil {
		campaignID = *session.CampaignID
	}

	pkgSess := &pkgsession.Session{
		ID:             session.ID,
		UserID:         "default", // TODO: Get from session metadata
		CampaignID:     campaignID,
		StartTime:      session.CreatedAt,
		LastActiveTime: session.UpdatedAt,
		Messages:       convertToPkgMessages(session.Messages),
		Metadata:       session.Metadata,
	}
	return s.sessionManager.SaveSession(context.Background(), pkgSess)
}

func (s *sessionManagerAdapter) GetContext(sessionID string, limit int) ([]*session.Message, error) {
	messages, err := s.multiSessionMgr.GetSessionMessages(sessionID, limit)
	if err != nil {
		return nil, err
	}
	return convertFromPkgMessages(messages), nil
}

func (s *sessionManagerAdapter) ExportSession(sessionID string, format session.ExportFormat) ([]byte, error) {
	// TODO: Implement using pkg session export
	return nil, nil
}

func (s *sessionManagerAdapter) ImportSession(data []byte, format session.ExportFormat) (*session.Session, error) {
	// TODO: Implement using pkg session import
	return nil, nil
}

func (s *sessionManagerAdapter) AppendMessage(sessionID string, role session.MessageRole, content string, toolCalls []session.ToolCall) (*session.Message, error) {
	// Convert role to pkg session format
	var msgType pkgsession.MessageType
	switch role {
	case session.RoleUser:
		msgType = pkgsession.MessageTypeUser
	case session.RoleAssistant:
		msgType = pkgsession.MessageTypeAgent
	case session.RoleSystem:
		msgType = pkgsession.MessageTypeSystem
	default:
		msgType = pkgsession.MessageTypeUser
	}

	// Create message
	msg := &pkgsession.Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Agent:     "chat", // Default agent for UI messages
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add to session
	sess, err := s.multiSessionMgr.GetActiveSession()
	if err != nil {
		return nil, err
	}

	sess.Messages = append(sess.Messages, *msg)

	// Save the session
	if err := s.sessionManager.SaveSession(context.Background(), sess); err != nil {
		return nil, err
	}

	// Convert back to session format
	return &session.Message{
		ID:        msg.ID,
		Role:      role,
		Content:   content,
		CreatedAt: msg.Timestamp,
	}, nil
}

func (s *sessionManagerAdapter) StreamMessage(sessionID string, role session.MessageRole) (session.MessageStream, error) {
	// TODO: Implement streaming message support
	return nil, fmt.Errorf("streaming messages not yet implemented")
}

func (s *sessionManagerAdapter) ClearContext(sessionID string) error {
	// Clear messages from the session
	sess, err := s.multiSessionMgr.GetActiveSession()
	if err != nil {
		return err
	}

	// Keep system messages, clear user/assistant messages
	var filteredMessages []pkgsession.Message
	for _, msg := range sess.Messages {
		if msg.Type == pkgsession.MessageTypeSystem {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	sess.Messages = filteredMessages
	return s.sessionManager.SaveSession(context.Background(), sess)
}

func (s *sessionManagerAdapter) ExportSessionWithOptions(sessionID string, format session.ExportFormat, options *session.ExportOptions) ([]byte, error) {
	// For now, delegate to regular export
	return s.ExportSession(sessionID, format)
}

func (s *sessionManagerAdapter) ForkSession(sourceID string, newName string) (*session.Session, error) {
	// TODO: Implement session forking
	return nil, fmt.Errorf("session forking not yet implemented")
}

// Conversion functions between package types

func convertPkgSession(s *pkgsession.Session) *session.Session {
	var campaignID *string
	if s.CampaignID != "" {
		campaignID = &s.CampaignID
	}

	return &session.Session{
		ID:         s.ID,
		Name:       fmt.Sprintf("Session %s", s.ID[:8]),
		CampaignID: campaignID,
		CreatedAt:  s.StartTime,
		UpdatedAt:  s.LastActiveTime,
		Messages:   convertFromPkgMessages(toPkgMessagePtrs(s.Messages)),
		Metadata:   s.Metadata,
	}
}

func toPkgMessagePtrs(messages []pkgsession.Message) []*pkgsession.Message {
	ptrs := make([]*pkgsession.Message, len(messages))
	for i := range messages {
		ptrs[i] = &messages[i]
	}
	return ptrs
}

func convertPkgMessageType(msgType pkgsession.MessageType) session.MessageRole {
	switch msgType {
	case pkgsession.MessageTypeUser:
		return session.RoleUser
	case pkgsession.MessageTypeAgent:
		return session.RoleAssistant
	case pkgsession.MessageTypeSystem:
		return session.RoleSystem
	default:
		return session.RoleUser
	}
}

func convertFromPkgMessages(messages []*pkgsession.Message) []*session.Message {
	result := make([]*session.Message, len(messages))
	for i, msg := range messages {
		result[i] = &session.Message{
			ID:        msg.ID,
			Role:      convertPkgMessageType(msg.Type),
			Content:   msg.Content,
			CreatedAt: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
	}
	return result
}

func convertToPkgMessages(messages []*session.Message) []pkgsession.Message {
	result := make([]pkgsession.Message, len(messages))
	for i, msg := range messages {
		// Extract agent from metadata if available
		agent := "chat" // default
		if msg.Metadata != nil {
			if a, ok := msg.Metadata["agent"].(string); ok {
				agent = a
			}
		}

		// Convert role to message type
		msgType := pkgsession.MessageTypeUser
		switch msg.Role {
		case session.RoleUser:
			msgType = pkgsession.MessageTypeUser
		case session.RoleAssistant:
			msgType = pkgsession.MessageTypeAgent
		case session.RoleSystem:
			msgType = pkgsession.MessageTypeSystem
		}

		result[i] = pkgsession.Message{
			ID:        msg.ID,
			Agent:     agent,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
			Type:      msgType,
			Metadata:  msg.Metadata,
		}
	}
	return result
}

func (app *App) convertPkgSessionMessage(msg *pkgsession.Message) common.ChatMessage {
	msgType := types.MsgSystem
	switch msg.Type {
	case pkgsession.MessageTypeUser:
		msgType = types.MsgUser
	case pkgsession.MessageTypeAgent:
		msgType = types.MsgAgent
	case pkgsession.MessageTypeSystem:
		msgType = types.MsgSystem
	}

	// Convert metadata from map[string]interface{} to map[string]string
	metadata := make(map[string]string)
	if msg.Metadata != nil {
		for k, v := range msg.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return common.ChatMessage{
		Type:      msgType,
		Content:   msg.Content,
		AgentID:   msg.Agent,
		Timestamp: msg.Timestamp,
		Metadata:  metadata,
	}
}

// Sprint 4: Enhanced State Management

// AppState represents a snapshot of the application state
type AppState struct {
	ID          string                           `json:"id"`
	Timestamp   time.Time                        `json:"timestamp"`
	Messages    []common.ChatMessage             `json:"messages"`
	ActiveTools map[string]*common.ToolExecution `json:"active_tools"`
	ViewMode    common.ViewMode                  `json:"view_mode"`
	SessionID   string                           `json:"session_id"`
	Context     map[string]interface{}           `json:"context"`
}

// StateListener is notified of state changes
type StateListener interface {
	OnStateChange(oldState, newState *AppState)
}

// StateManager manages application state with history and listeners
type StateManager struct {
	ctx            context.Context
	currentState   *AppState
	history        []AppState
	maxHistory     int
	listeners      []StateListener
	mu             sync.RWMutex
	persistenceKey string
}

// NewStateManager creates a new state manager
func NewStateManager(ctx context.Context) *StateManager {
	return &StateManager{
		ctx:        ctx,
		maxHistory: 100, // Keep last 100 states
		history:    make([]AppState, 0, 100),
		listeners:  make([]StateListener, 0),
		currentState: &AppState{
			ID:          fmt.Sprintf("state-%d", time.Now().UnixNano()),
			Timestamp:   time.Now(),
			Messages:    make([]common.ChatMessage, 0),
			ActiveTools: make(map[string]*common.ToolExecution),
			ViewMode:    common.ViewModeNormal,
			Context:     make(map[string]interface{}),
		},
	}
}

// GetCurrentState returns a copy of the current state
func (sm *StateManager) GetCurrentState() *AppState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.copyCurrentStateLocked()
}

func (sm *StateManager) copyCurrentStateLocked() *AppState {
	// Deep copy the state
	stateCopy := *sm.currentState
	stateCopy.Messages = make([]common.ChatMessage, len(sm.currentState.Messages))
	copy(stateCopy.Messages, sm.currentState.Messages)

	stateCopy.ActiveTools = make(map[string]*common.ToolExecution)
	for k, v := range sm.currentState.ActiveTools {
		toolCopy := *v
		stateCopy.ActiveTools[k] = &toolCopy
	}

	stateCopy.Context = make(map[string]interface{})
	for k, v := range sm.currentState.Context {
		stateCopy.Context[k] = v
	}

	return &stateCopy
}

// UpdateState updates the current state and notifies listeners
func (sm *StateManager) UpdateState(updater func(*AppState) *AppState) error {
	if err := sm.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("state-manager").
			WithOperation("UpdateState")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create a copy of current state
	oldState := sm.copyCurrentStateLocked()

	// Apply the update
	newState := updater(oldState)
	if newState == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "updater returned nil state", nil).
			WithComponent("state-manager").
			WithOperation("UpdateState")
	}

	// Set new state ID and timestamp
	newState.ID = fmt.Sprintf("state-%d", time.Now().UnixNano())
	newState.Timestamp = time.Now()

	// Update current state
	sm.currentState = newState

	// Add to history
	sm.history = append(sm.history, *newState)
	if len(sm.history) > sm.maxHistory {
		sm.history = sm.history[len(sm.history)-sm.maxHistory:]
	}

	// Notify listeners
	for _, listener := range sm.listeners {
		go listener.OnStateChange(oldState, newState)
	}

	return nil
}

// AddListener adds a state change listener
func (sm *StateManager) AddListener(listener StateListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// RemoveListener removes a state change listener
func (sm *StateManager) RemoveListener(listener StateListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, l := range sm.listeners {
		if l == listener {
			sm.listeners = append(sm.listeners[:i], sm.listeners[i+1:]...)
			break
		}
	}
}

// GetHistory returns the state history
func (sm *StateManager) GetHistory() []AppState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	historyCopy := make([]AppState, len(sm.history))
	copy(historyCopy, sm.history)
	return historyCopy
}

// Restore restores a previous state by ID
func (sm *StateManager) Restore(stateID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := len(sm.history) - 1; i >= 0; i-- {
		if sm.history[i].ID == stateID {
			sm.currentState = &sm.history[i]
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "state not found", nil).
		WithComponent("state-manager").
		WithOperation("Restore").
		WithDetails("state_id", stateID)
}

// Enhanced app methods for state management

// initializeStateManager initializes the enhanced state management
func (app *App) initializeStateManager() error {
	app.stateManager = NewStateManager(app.ctx)

	// Add app as a listener to its own state changes
	app.stateManager.AddListener(app)

	// Initialize state from current app data
	return app.stateManager.UpdateState(func(state *AppState) *AppState {
		state.Messages = app.messages
		state.ActiveTools = app.activeTools
		state.ViewMode = app.currentView
		state.SessionID = app.config.SessionID
		return state
	})
}

// OnStateChange implements StateListener interface
func (app *App) OnStateChange(oldState, newState *AppState) {
	// Auto-save session on state change
	if app.sessionManager != nil && app.currentSession != nil {
		go func() {
			if err := app.sessionManager.SaveSession(app.currentSession); err != nil {
				// Log error but don't fail
				if app.statusPane != nil {
					app.statusPane.UpdateStatus("Failed to auto-save session", "warning")
				}
			}
		}()
	}

	// Update UI components if needed
	if oldState.ViewMode != newState.ViewMode {
		// View mode changes are handled by the panes themselves
		// during the regular Update cycle
	}

	// Persist state for recovery
	if app.recoveryManager != nil {
		go func() {
			if err := app.persistStateForRecovery(newState); err != nil {
				// Log error but don't fail
				observability.GetLogger(app.ctx).Warn("Failed to persist state for recovery", "error", err)
			}
		}()
	}
}

// persistStateForRecovery saves state for crash recovery
func (app *App) persistStateForRecovery(state *AppState) error {
	if app.recoveryManager == nil || app.currentSession == nil {
		return nil
	}

	// Get the active session from the multi-session manager
	pkgSession, err := app.multiSessionMgr.GetActiveSession()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get active session for checkpoint").
			WithComponent("chat.app").
			WithOperation("persistStateForRecovery")
	}

	// Create checkpoint using the current session
	return app.recoveryManager.CreateCheckpoint(app.ctx, pkgSession)
}

// updateAppState updates the app state through the state manager
func (app *App) updateAppState(updater func(*AppState) *AppState) error {
	if app.stateManager == nil {
		return gerror.New(gerror.ErrCodeNotFound, "state manager not initialized", nil).
			WithComponent("chat.app").
			WithOperation("updateAppState")
	}

	return app.stateManager.UpdateState(updater)
}

// addMessageWithState adds a message and updates state atomically
func (app *App) addMessageWithState(msg common.ChatMessage) error {
	// Update local state
	app.messages = append(app.messages, msg)

	// Update managed state
	return app.updateAppState(func(state *AppState) *AppState {
		state.Messages = append(state.Messages, msg)
		return state
	})
}

// updateToolStateWithState updates tool execution state atomically
func (app *App) updateToolStateWithState(toolID string, execution *common.ToolExecution) error {
	// Update local state
	if execution == nil {
		delete(app.activeTools, toolID)
	} else {
		app.activeTools[toolID] = execution
	}

	// Update managed state
	return app.updateAppState(func(state *AppState) *AppState {
		if execution == nil {
			delete(state.ActiveTools, toolID)
		} else {
			state.ActiveTools[toolID] = execution
		}
		return state
	})
}

// setViewModeWithState sets view mode and updates state atomically
func (app *App) setViewModeWithState(mode common.ViewMode) error {
	// Update local state
	app.currentView = mode

	// Update managed state
	return app.updateAppState(func(state *AppState) *AppState {
		state.ViewMode = mode
		return state
	})
}

// getStateHistory returns the application state history
func (app *App) getStateHistory() []AppState {
	if app.stateManager == nil {
		return []AppState{}
	}
	return app.stateManager.GetHistory()
}

// restoreState restores a previous application state
func (app *App) restoreState(stateID string) error {
	if app.stateManager == nil {
		return gerror.New(gerror.ErrCodeNotFound, "state manager not initialized", nil).
			WithComponent("chat.app").
			WithOperation("restoreState")
	}

	if err := app.stateManager.Restore(stateID); err != nil {
		return err
	}

	// Update local state from restored state
	restoredState := app.stateManager.GetCurrentState()
	app.messages = restoredState.Messages
	app.activeTools = restoredState.ActiveTools
	app.currentView = restoredState.ViewMode

	// Update UI by clearing and re-adding messages
	if app.outputPane != nil {
		app.outputPane.Clear()
		for _, msg := range app.messages {
			app.outputPane.AddMessage(msg)
		}
	}
	// View mode changes will be handled in the next Update cycle

	return nil
}
