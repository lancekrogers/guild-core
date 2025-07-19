// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/registry"
)

// OrchestratorService wraps the orchestrator to integrate with the service framework
type OrchestratorService struct {
	orchestrator orchestrator.Orchestrator
	registry     registry.ComponentRegistry
	eventBus     events.EventBus
	logger       observability.Logger
	config       OrchestratorServiceConfig

	// Service state
	started bool
	mu      sync.RWMutex

	// Event bridge to connect orchestrator events to central event bus
	eventBridge *orchestratorEventBridge

	// Metrics
	commissionsStarted   uint64
	commissionsCompleted uint64
	taskDispatched       uint64
	agentAssignments     uint64
	avgCommissionTime    time.Duration
}

// OrchestratorServiceConfig configures the orchestrator service
type OrchestratorServiceConfig struct {
	MaxConcurrentAgents int
	ManagerAgentID      string
	KanbanBoardID       string
	ExecutionMode       string // "sequential", "parallel", "managed"
	EnableEventBridge   bool
}

// DefaultOrchestratorServiceConfig returns default configuration
func DefaultOrchestratorServiceConfig() OrchestratorServiceConfig {
	return OrchestratorServiceConfig{
		MaxConcurrentAgents: 10,
		ExecutionMode:       "managed",
		EnableEventBridge:   true,
	}
}

// NewOrchestratorService creates a new orchestrator service wrapper
func NewOrchestratorService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config OrchestratorServiceConfig,
) (*OrchestratorService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("OrchestratorService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("OrchestratorService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("OrchestratorService")
	}

	// Create orchestrator configuration
	orchConfig := orchestrator.Config{
		MaxConcurrentAgents: config.MaxConcurrentAgents,
		ManagerAgentID:      config.ManagerAgentID,
		KanbanBoardID:       config.KanbanBoardID,
		ExecutionMode:       config.ExecutionMode,
	}

	// Create orchestrator instance
	// First, we need to get the event bus and dispatcher from registry
	orchRegistry := registry.Orchestrator()
	if orchRegistry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "orchestrator registry not available", nil).
			WithComponent("OrchestratorService")
	}

	// Get default event bus from orchestrator registry
	var orchEventBus orchestrator.EventBus
	if orchReg, ok := orchRegistry.(orchestrator.OrchestratorRegistry); ok {
		var err error
		orchEventBus, err = orchReg.GetDefaultEventBus()
		if err != nil {
			// Create a default event bus if none registered
			orchEventBus = orchestrator.DefaultEventBusFactory()
		}
	} else {
		// Create a default event bus
		orchEventBus = orchestrator.DefaultEventBusFactory()
	}

	// Create dependencies for task dispatcher
	// For now, we'll create minimal dependencies - in production these would come from registry
	// TODO: Get actual kanban board from registry
	kanbanBoard := &kanban.Board{} // Placeholder - should get from registry
	kanbanManager := orchestrator.DefaultKanbanManagerFactory(kanbanBoard)
	agentFactory := &simpleAgentFactory{registry: registry}

	// Create a task dispatcher
	dispatcher := orchestrator.DefaultTaskDispatcherFactory(kanbanManager, agentFactory, orchEventBus, config.MaxConcurrentAgents)

	// Create orchestrator using the factory
	orch := orchestrator.DefaultOrchestratorFactory(&orchConfig, dispatcher, orchEventBus)

	service := &OrchestratorService{
		orchestrator: orch,
		registry:     registry,
		eventBus:     eventBus,
		logger:       logger,
		config:       config,
	}

	// Create event bridge if enabled
	if config.EnableEventBridge {
		service.eventBridge = newOrchestratorEventBridge(orch, eventBus, logger)
	}

	return service, nil
}

// Name returns the service name
func (s *OrchestratorService) Name() string {
	return "orchestrator-service"
}

// Start initializes and starts the service
func (s *OrchestratorService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("OrchestratorService")
	}

	// Start the orchestrator
	if err := s.orchestrator.Start(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start orchestrator").
			WithComponent("OrchestratorService")
	}

	// Start event bridge if enabled
	if s.eventBridge != nil {
		s.eventBridge.Start()
	}

	s.started = true

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"orchestrator-service-started",
		"service.started",
		"orchestrator",
		map[string]interface{}{
			"execution_mode":        s.config.ExecutionMode,
			"max_concurrent_agents": s.config.MaxConcurrentAgents,
			"event_bridge_enabled":  s.config.EnableEventBridge,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Orchestrator service started",
		"execution_mode", s.config.ExecutionMode,
		"max_concurrent_agents", s.config.MaxConcurrentAgents)

	return nil
}

// Stop gracefully shuts down the service
func (s *OrchestratorService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("OrchestratorService")
	}

	// Stop event bridge first
	if s.eventBridge != nil {
		s.eventBridge.Stop()
	}

	// Stop the orchestrator
	if err := s.orchestrator.Stop(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Failed to stop orchestrator", "error", err)
	}

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"orchestrator-service-stopped",
		"service.stopped",
		"orchestrator",
		map[string]interface{}{
			"commissions_started":   s.commissionsStarted,
			"commissions_completed": s.commissionsCompleted,
			"tasks_dispatched":      s.taskDispatched,
			"agent_assignments":     s.agentAssignments,
			"avg_commission_time":   s.avgCommissionTime.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false

	s.logger.InfoContext(ctx, "Orchestrator service stopped",
		"total_commissions", s.commissionsStarted)

	return nil
}

// Health checks if the service is healthy
func (s *OrchestratorService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("OrchestratorService")
	}

	// Check orchestrator status
	status := s.orchestrator.Status()
	if status == orchestrator.StatusError {
		return gerror.New(gerror.ErrCodeInternal, "orchestrator in error state", nil).
			WithComponent("OrchestratorService")
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *OrchestratorService) Ready(ctx context.Context) error {
	if err := s.Health(ctx); err != nil {
		return err
	}

	// Check if orchestrator is running
	status := s.orchestrator.Status()
	if status != orchestrator.StatusRunning {
		return gerror.New(gerror.ErrCodeResourceExhausted, "orchestrator not running", nil).
			WithComponent("OrchestratorService").
			WithDetails("status", string(status))
	}

	return nil
}

// SetCommission sets the current commission with event emission
func (s *OrchestratorService) SetCommission(ctx context.Context, comm *commission.Commission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("OrchestratorService")
	}

	// Set the commission
	if err := s.orchestrator.SetCommission(comm); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set commission").
			WithComponent("OrchestratorService")
	}

	// Update metrics
	s.commissionsStarted++

	// Emit commission started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"commission-"+comm.ID+"-started",
		"commission.started",
		"orchestrator",
		map[string]interface{}{
			"commission_id": comm.ID,
			"title":         comm.Title,
			"start_time":    time.Now(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish commission started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Commission started",
		"commission_id", comm.ID,
		"title", comm.Title)

	return nil
}

// AddAgent adds an agent to the orchestrator with event emission
func (s *OrchestratorService) AddAgent(ctx context.Context, agent core.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("OrchestratorService")
	}

	// Add the agent
	if err := s.orchestrator.AddAgent(agent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add agent").
			WithComponent("OrchestratorService")
	}

	// Update metrics
	s.agentAssignments++

	// Emit agent added event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agent-"+agent.GetID()+"-added",
		"agent.added",
		"orchestrator",
		map[string]interface{}{
			"agent_id":   agent.GetID(),
			"agent_type": agent.GetType(),
			"add_time":   time.Now(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish agent added event", "error", err)
	}

	return nil
}

// GetOrchestrator returns the wrapped orchestrator (for compatibility)
func (s *OrchestratorService) GetOrchestrator() orchestrator.Orchestrator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.orchestrator
}

// orchestratorEventBridge bridges orchestrator events to the central event bus
type orchestratorEventBridge struct {
	orchestrator orchestrator.Orchestrator
	eventBus     events.EventBus
	logger       observability.Logger
	stopCh       chan struct{}
}

func newOrchestratorEventBridge(
	orch orchestrator.Orchestrator,
	eventBus events.EventBus,
	logger observability.Logger,
) *orchestratorEventBridge {
	return &orchestratorEventBridge{
		orchestrator: orch,
		eventBus:     eventBus,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

func (b *orchestratorEventBridge) Start() {
	// Add event handler to orchestrator
	b.orchestrator.AddEventHandler(b.handleOrchestratorEvent)
}

func (b *orchestratorEventBridge) Stop() {
	close(b.stopCh)
}

func (b *orchestratorEventBridge) handleOrchestratorEvent(event orchestrator.Event) {
	ctx := context.Background()

	// Convert orchestrator event to standard event
	standardEvent := events.NewBaseEvent(
		event.ID,
		"orchestrator."+string(event.Type),
		"orchestrator",
		event.Data,
	)

	// Publish to central event bus
	if err := b.eventBus.Publish(ctx, standardEvent); err != nil {
		b.logger.WarnContext(ctx, "Failed to bridge orchestrator event",
			"event_type", event.Type,
			"error", err)
	}
}

// simpleAgentFactory is a basic agent factory implementation
type simpleAgentFactory struct {
	registry registry.ComponentRegistry
}

// CreateAgent creates an agent by type
func (f *simpleAgentFactory) CreateAgent(agentType, name string, options ...interface{}) (core.Agent, error) {
	// Get agent from registry
	agentReg := f.registry.Agents()
	if agentReg == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "agent registry not available", nil).
			WithComponent("simpleAgentFactory")
	}

	agent, err := agentReg.GetAgent(agentType)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent").
			WithComponent("simpleAgentFactory").
			WithDetails("agent_type", agentType)
	}

	return agent, nil
}
