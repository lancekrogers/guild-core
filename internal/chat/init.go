package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"

	"github.com/guild-ventures/guild-core/pkg/config"
	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// NewChatModel creates a new chat model
func NewChatModel(guildConfig *config.GuildConfig, conn *grpc.ClientConn, promptsConn *grpc.ClientConn, campaignID string) ChatModel {
	// Initialize textarea
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.CharLimit = 1000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Single line input

	// Initialize viewport
	vp := viewport.New(80, 20)
	vp.SetContent("🏰 Welcome to Guild Chat\n\nType '/help' for available commands.")

	// Create gRPC clients
	grpcClient := pb.NewGuildServiceClient(conn)
	promptsClient := promptspb.NewPromptsServiceClient(promptsConn)

	// Initialize command history
	var historyFile string
	projectInfo, err := project.DetectProject(".")
	if err == nil && projectInfo != nil {
		historyFile = filepath.Join(projectInfo.Root, ".guild", "chat_history.txt")
	} else {
		// Fallback to home directory
		homeDir, _ := os.UserHomeDir()
		historyFile = filepath.Join(homeDir, ".guild", "chat_history.txt")
	}

	// Create new model
	m := ChatModel{
		messages:      []Message{},
		input:         ta,
		viewport:      vp,
		ready:         false,
		grpcClient:    grpcClient,
		promptsClient: promptsClient,
		campaignID:    campaignID,
		sessionID:     fmt.Sprintf("chat-%d", time.Now().Unix()),
		guildConfig:   guildConfig,
		keys:          DefaultKeyMap(),
		help:          help.New(),
		err:           nil,
		width:         80,
		height:        24,
		viewMode:      chatModeNormal,
		promptLayers:  []string{},
		activeTools:   make(map[string]*toolExecution),
		costConsent:   make(map[string]bool),
		
		// Initialize history
		history: NewCommandHistory(historyFile),
		
		// Completion state
		showingCompletion: false,
		completionResults: nil,
		completionIndex:   0,
		
		// Integration flags
		integrationFlags: make(map[string]bool),
	}

	// Add initial system message
	m.messages = append(m.messages, Message{
		Type:      msgSystem,
		Content:   fmt.Sprintf("📜 Campaign: %s | Session: %s", campaignID, m.sessionID),
		Timestamp: time.Now(),
	})

	return m
}

// Init implements tea.Model
func (m ChatModel) Init() tea.Cmd {
	// Initialize sub-components asynchronously
	go func() {
		if err := m.InitializeAllComponents(); err != nil {
			log.Printf("Warning: Some components failed to initialize: %v", err)
		}
	}()

	// Set up initial state
	return tea.Batch(
		textarea.Blink,
		m.listenForAgentUpdates(),
	)
}

// initializeMarkdownRenderer sets up the markdown rendering components
func (m *ChatModel) initializeMarkdownRenderer() error {
	renderer, err := NewMarkdownRenderer(m.width)
	if err != nil {
		return gerror.Wrap(err, "failed to create markdown renderer",
			gerror.WithComponent("chat"),
			gerror.WithOperation("initializeMarkdownRenderer"))
	}
	
	m.markdownRenderer = renderer
	m.contentFormatter = NewContentFormatter(renderer, m.width)
	
	return nil
}

// initializeStatusDisplay sets up the agent status display
func (m *ChatModel) initializeStatusDisplay() error {
	if m.guildConfig == nil {
		return gerror.New(gerror.ErrCodeMissingRequired, "guild config not provided",
			gerror.WithComponent("chat"),
			gerror.WithOperation("initializeStatusDisplay"))
	}
	
	// Create agent status tracker
	m.agentStatusTracker = NewAgentStatusTracker(m.guildConfig)
	
	// Create status display
	m.statusDisplay = NewStatusDisplay(m.agentStatusTracker, m.width, m.height)
	
	// Create agent indicators
	m.agentIndicators = NewAgentIndicators()
	
	return nil
}

// initializeAutoCompletion sets up the auto-completion engine
func (m *ChatModel) initializeAutoCompletion() error {
	// Create completion engine with guild context
	engine := NewCompletionEngine()
	
	// Register command completions
	engine.RegisterCommands([]string{
		"/help", "/status", "/agents", "/prompt", "/tools",
		"/test", "/clear", "/exit", "/quit",
	})
	
	// Register agent completions from guild config
	if m.guildConfig != nil {
		for _, agent := range m.guildConfig.Agents {
			engine.RegisterAgent("@" + agent.ID, agent.Name)
		}
	}
	
	m.completionEng = engine
	m.commandProc = NewCommandProcessor(m.guildConfig)
	
	return nil
}

// initializeCommandHistory sets up command history management
func (m *ChatModel) initializeCommandHistory() error {
	if m.history == nil {
		// This shouldn't happen as it's initialized in NewChatModel
		return gerror.New(gerror.ErrCodeInternalError, "command history not initialized",
			gerror.WithComponent("chat"),
			gerror.WithOperation("initializeCommandHistory"))
	}
	
	// History is already initialized in NewChatModel
	return nil
}

// InitializeAllComponents initializes and validates all chat components
func (m *ChatModel) InitializeAllComponents() error {
	// Initialize integration state tracking
	if m.integrationFlags == nil {
		m.integrationFlags = make(map[string]bool)
	}

	// Initialize markdown renderer (Agent 1's component)
	if err := m.initializeMarkdownRenderer(); err != nil {
		m.integrationFlags["markdown_failed"] = true
		// Continue with graceful degradation
	} else {
		m.integrationFlags["markdown_enabled"] = true
	}

	// Initialize status display (Agent 1's component)
	if err := m.initializeStatusDisplay(); err != nil {
		m.integrationFlags["status_display_failed"] = true
		// Continue with graceful degradation
	} else {
		m.integrationFlags["status_display_enabled"] = true
	}

	// Initialize auto-completion (Agent 4's component)
	if err := m.initializeAutoCompletion(); err != nil {
		m.integrationFlags["auto_complete_failed"] = true
		// Continue with graceful degradation
	} else {
		m.integrationFlags["auto_complete_enabled"] = true
	}

	// Initialize command history (Agent 4's component)
	if err := m.initializeCommandHistory(); err != nil {
		m.integrationFlags["command_history_failed"] = true
		// Continue with graceful degradation
	} else {
		m.integrationFlags["command_history_enabled"] = true
	}

	return m.ValidateAllComponents()
}

// ValidateAllComponents ensures all components are properly configured
func (m *ChatModel) ValidateAllComponents() error {
	var validationErrors []string

	// Validate markdown components
	if m.integrationFlags["markdown_enabled"] {
		if m.markdownRenderer == nil || m.contentFormatter == nil {
			validationErrors = append(validationErrors, "markdown components not properly initialized")
		}
	}

	// Validate status display
	if m.integrationFlags["status_display_enabled"] {
		if m.statusDisplay == nil {
			validationErrors = append(validationErrors, "status display not initialized")
		} else {
			// Test render to ensure it works
			if testRender := m.statusDisplay.RenderCompactStatus(); testRender == "" {
				validationErrors = append(validationErrors, "status display render test failed")
			}
		}
	}

	// Validate auto-completion
	if m.integrationFlags["auto_complete_enabled"] {
		if m.completionEng == nil {
			validationErrors = append(validationErrors, "auto-completion not initialized")
		}
	}

	// Validate command history
	if m.integrationFlags["command_history_enabled"] {
		if m.history == nil {
			validationErrors = append(validationErrors, "command history not initialized")
		}
	}

	// Check for critical failures
	criticalComponents := []string{"markdown_enabled", "status_display_enabled"}
	allCriticalFailed := true
	for _, component := range criticalComponents {
		if m.integrationFlags[component] {
			allCriticalFailed = false
			break
		}
	}

	if allCriticalFailed {
		return fmt.Errorf("all critical visual components failed to initialize")
	}

	if len(validationErrors) > 0 {
		// Log warnings but don't fail - graceful degradation
		log.Printf("Some components had validation warnings: %v", validationErrors)
	}

	// Mark integration as ready
	m.integrationFlags["enhanced_view"] = true
	m.integrationFlags["integrated_processing"] = true
	
	return nil
}

// listenForAgentUpdates subscribes to agent status updates
func (m ChatModel) listenForAgentUpdates() tea.Cmd {
	return func() tea.Msg {
		// This would connect to the actual agent status stream
		// For now, return nil to prevent blocking
		return nil
	}
}