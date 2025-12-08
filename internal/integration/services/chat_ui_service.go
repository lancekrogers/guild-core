// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	chatui "github.com/guild-framework/guild-core/internal/ui/chat"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/registry"
)

// ChatUIService wraps the Bubble Tea chat application to integrate with the service framework
type ChatUIService struct {
	app      *chatui.App
	program  *tea.Program
	registry registry.ComponentRegistry
	eventBus events.EventBus
	logger   observability.Logger
	config   ChatUIServiceConfig

	// Service state
	started bool
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex

	// UI state
	activeView string
	sessionID  string

	// Metrics
	messagesProcessed uint64
	commandsExecuted  uint64
	errorsCount       uint64
	uptimeStart       time.Time
}

// ChatUIServiceConfig configures the chat UI service
type ChatUIServiceConfig struct {
	GuildConfig   *config.GuildConfig
	CampaignID    string
	SessionID     string
	UserID        string
	SelectedGuild string
	EnableLogging bool
	LogPath       string
}

// DefaultChatUIServiceConfig returns default configuration
func DefaultChatUIServiceConfig() ChatUIServiceConfig {
	return ChatUIServiceConfig{
		EnableLogging: true,
		LogPath:       ".guild/logs/chat-ui.log",
		UserID:        "default",
	}
}

// NewChatUIService creates a new chat UI service wrapper
func NewChatUIService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config ChatUIServiceConfig,
) (*ChatUIService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("ChatUIService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("ChatUIService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("ChatUIService")
	}
	if config.GuildConfig == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "guild config cannot be nil", nil).
			WithComponent("ChatUIService")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ChatUIService{
		registry:  registry,
		eventBus:  eventBus,
		logger:    logger,
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		sessionID: config.SessionID,
	}, nil
}

// Name returns the service name
func (s *ChatUIService) Name() string {
	return "chat-ui-service"
}

// Start initializes and starts the service
func (s *ChatUIService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("ChatUIService")
	}

	// Create the chat app
	s.app = chatui.NewApp(s.ctx, s.config.GuildConfig, s.registry)

	// Configure the app
	s.app.SetSelectedGuild(s.config.SelectedGuild)
	s.app.SetCampaignID(s.config.CampaignID)
	s.app.SetSessionID(s.config.SessionID)
	s.app.SetUserID(s.config.UserID)

	// Create the Bubble Tea program
	// Note: We don't start it immediately as it takes control of the terminal
	s.program = tea.NewProgram(s.app, tea.WithContext(s.ctx))

	// Subscribe to relevant events
	if err := s.subscribeToEvents(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to events").
			WithComponent("ChatUIService")
	}

	s.started = true
	s.uptimeStart = time.Now()

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"chat-ui-service-started",
		"service.started",
		"chat-ui",
		map[string]interface{}{
			"session_id":     s.config.SessionID,
			"campaign_id":    s.config.CampaignID,
			"selected_guild": s.config.SelectedGuild,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Chat UI service started",
		"session_id", s.config.SessionID,
		"campaign_id", s.config.CampaignID)

	return nil
}

// Run starts the actual UI program (blocks until exit)
func (s *ChatUIService) Run() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("ChatUIService")
	}
	if s.running {
		s.mu.Unlock()
		return gerror.New(gerror.ErrCodeAlreadyExists, "UI already running", nil).
			WithComponent("ChatUIService")
	}
	s.running = true
	s.mu.Unlock()

	// Emit UI running event
	if err := s.eventBus.Publish(s.ctx, events.NewBaseEvent(
		"chat-ui-running",
		"ui.running",
		"chat-ui",
		map[string]interface{}{
			"session_id": s.sessionID,
		},
	)); err != nil {
		s.logger.WarnContext(s.ctx, "Failed to publish UI running event", "error", err)
	}

	// Run the Bubble Tea program (blocks)
	_, err := s.program.Run()

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	if err != nil {
		s.errorsCount++
		return gerror.Wrap(err, gerror.ErrCodeInternal, "UI program error").
			WithComponent("ChatUIService")
	}

	return nil
}

// Stop gracefully shuts down the service
func (s *ChatUIService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("ChatUIService")
	}

	// Cancel context to signal shutdown
	s.cancel()

	// If UI is running, send quit command
	if s.running && s.program != nil {
		s.program.Quit()
	}

	// Calculate uptime
	uptime := time.Since(s.uptimeStart)

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"chat-ui-service-stopped",
		"service.stopped",
		"chat-ui",
		map[string]interface{}{
			"messages_processed": s.messagesProcessed,
			"commands_executed":  s.commandsExecuted,
			"errors_count":       s.errorsCount,
			"uptime_seconds":     uptime.Seconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false

	s.logger.InfoContext(ctx, "Chat UI service stopped",
		"messages_processed", s.messagesProcessed,
		"uptime", uptime)

	return nil
}

// Health checks if the service is healthy
func (s *ChatUIService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("ChatUIService")
	}

	// Check if context is still valid
	if err := s.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "service context cancelled").
			WithComponent("ChatUIService")
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *ChatUIService) Ready(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("ChatUIService")
	}

	// UI service is ready once started
	return nil
}

// subscribeToEvents subscribes to relevant events from the event bus
func (s *ChatUIService) subscribeToEvents(ctx context.Context) error {
	// Subscribe to system events that might affect UI
	systemEvents := []string{
		"agent.state.changed",
		"task.progress",
		"commission.status.changed",
		"memory.search.results",
	}

	for _, eventType := range systemEvents {
		handler := s.createEventHandler(eventType)
		if _, err := s.eventBus.Subscribe(ctx, eventType, handler); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to event").
				WithComponent("ChatUIService").
				WithDetails("event_type", eventType)
		}
	}

	return nil
}

// createEventHandler creates an event handler for a specific event type
func (s *ChatUIService) createEventHandler(eventType string) events.EventHandler {
	return func(ctx context.Context, event events.CoreEvent) error {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if !s.running || s.program == nil {
			return nil // UI not running, ignore events
		}

		// Convert event to UI message
		// This would normally send a tea.Msg to the program
		// For now, we'll just log it
		s.logger.DebugContext(ctx, "Received event for UI",
			"event_type", eventType,
			"event_id", event.GetID())

		// In a real implementation, we would:
		// s.program.Send(uiEventMsg{Event: event})

		return nil
	}
}

// HandleMessage processes a chat message with event emission
func (s *ChatUIService) HandleMessage(ctx context.Context, message string) error {
	s.mu.Lock()
	s.messagesProcessed++
	s.mu.Unlock()

	// Emit message event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"chat-message-"+s.sessionID,
		"chat.message.sent",
		"chat-ui",
		map[string]interface{}{
			"session_id":     s.sessionID,
			"message_length": len(message),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish message event", "error", err)
	}

	return nil
}

// HandleCommand processes a UI command with event emission
func (s *ChatUIService) HandleCommand(ctx context.Context, command string) error {
	s.mu.Lock()
	s.commandsExecuted++
	s.mu.Unlock()

	// Emit command event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"chat-command-"+command,
		"chat.command.executed",
		"chat-ui",
		map[string]interface{}{
			"session_id": s.sessionID,
			"command":    command,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish command event", "error", err)
	}

	return nil
}

// GetMetrics returns service metrics
func (s *ChatUIService) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uptime := time.Duration(0)
	if s.started {
		uptime = time.Since(s.uptimeStart)
	}

	return map[string]interface{}{
		"messages_processed": s.messagesProcessed,
		"commands_executed":  s.commandsExecuted,
		"errors_count":       s.errorsCount,
		"uptime_seconds":     uptime.Seconds(),
		"is_running":         s.running,
	}
}
