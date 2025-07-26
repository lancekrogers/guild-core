// Package bridges provides integration bridges between different components
package bridges

import (
	"context"
	"sync"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// AgentRegistrationBridge manages agent registration with the task dispatcher
type AgentRegistrationBridge struct {
	eventBus      events.EventBus
	logger        observability.Logger
	subscriptions map[string]events.SubscriptionID
	mu            sync.RWMutex

	// Dependencies
	AgentRegistry  registry.AgentRegistry
	AgentFactory   AgentFactory
	TaskDispatcher TaskDispatcher
	config         *AgentRegistrationConfig

	// State tracking
	registeredAgents map[string]core.Agent
}

// AgentRegistrationConfig configures the agent registration bridge
type AgentRegistrationConfig struct {
	Enabled               bool
	AutoRegisterOnStartup bool
	LoadFromGuildConfig   bool
	GuildConfigPath       string
	MaxAgents             int
}

// AgentFactory creates agent instances
type AgentFactory interface {
	CreateAgent(agentType, name string, options ...interface{}) (core.Agent, error)
}

// NewAgentRegistrationBridge creates a new agent registration bridge
func NewAgentRegistrationBridge(
	eventBus events.EventBus,
	logger observability.Logger,
	config AgentRegistrationConfig,
	agentRegistry registry.AgentRegistry,
	agentFactory AgentFactory,
	taskDispatcher TaskDispatcher,
) *AgentRegistrationBridge {
	return &AgentRegistrationBridge{
		eventBus:         eventBus,
		logger:           logger.WithComponent("AgentRegistrationBridge"),
		subscriptions:    make(map[string]events.SubscriptionID),
		config:           &config,
		AgentRegistry:    agentRegistry,
		AgentFactory:     agentFactory,
		TaskDispatcher:   taskDispatcher,
		registeredAgents: make(map[string]core.Agent),
	}
}

// Start starts the bridge and subscribes to events
func (b *AgentRegistrationBridge) Start(ctx context.Context) error {
	if !b.config.Enabled {
		b.logger.InfoContext(ctx, "Agent registration bridge disabled")
		return nil
	}

	b.logger.InfoContext(ctx, "Starting agent registration bridge")

	// Subscribe to agent discovery events (emitted by other bridges)
	if err := b.subscribeToEvents(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to events").
			WithComponent("AgentRegistrationBridge")
	}

	// Auto-register agents on startup if enabled
	if b.config.AutoRegisterOnStartup {
		if err := b.registerAgentsFromConfig(ctx); err != nil {
			b.logger.WithError(err).WarnContext(ctx, "Failed to auto-register agents")
			// Don't fail startup - agents can be registered later
		}
	}

	b.logger.InfoContext(ctx, "Agent registration bridge started")
	return nil
}

// Stop stops the bridge and unsubscribes from events
func (b *AgentRegistrationBridge) Stop(ctx context.Context) error {
	b.logger.InfoContext(ctx, "Stopping agent registration bridge")

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Unsubscribe from all events
	for eventType, subID := range b.subscriptions {
		if err := b.eventBus.Unsubscribe(ctx, subID); err != nil {
			b.logger.WithError(err).ErrorContext(ctx, "Failed to unsubscribe from event",
				"event_type", eventType)
		}
	}

	b.logger.InfoContext(ctx, "Agent registration bridge stopped")
	return nil
}

// Health returns nil if the bridge is healthy
func (b *AgentRegistrationBridge) Health(ctx context.Context) error {
	if b.AgentRegistry == nil {
		return gerror.New(gerror.ErrCodeInternal, "agent registry not available", nil).
			WithComponent("AgentRegistrationBridge")
	}
	if b.TaskDispatcher == nil {
		return gerror.New(gerror.ErrCodeInternal, "task dispatcher not available", nil).
			WithComponent("AgentRegistrationBridge")
	}
	return nil
}

// Ready returns nil if the bridge is ready to process events
func (b *AgentRegistrationBridge) Ready(ctx context.Context) error {
	if !b.config.Enabled {
		return nil // Always ready if disabled
	}
	return b.Health(ctx)
}

// Name returns the service name
func (b *AgentRegistrationBridge) Name() string {
	return "agent-registration-bridge"
}

// subscribeToEvents sets up event subscriptions
func (b *AgentRegistrationBridge) subscribeToEvents(ctx context.Context) error {
	// Subscribe to agent discovery events (emitted by orchestrator campaign bridge)
	subID, err := b.eventBus.Subscribe(ctx, "agent.discovered", func(ctx context.Context, e events.CoreEvent) error {
		return b.handleAgentDiscovered(ctx, e)
	})
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.subscriptions["agent.discovered"] = subID
	b.mu.Unlock()

	// Subscribe to commission process requested events to register agents on-demand
	subID, err = b.eventBus.Subscribe(ctx, "commission.process_requested", func(ctx context.Context, e events.CoreEvent) error {
		return b.handleCommissionProcessRequested(ctx, e)
	})
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.subscriptions["commission.process_requested"] = subID
	b.mu.Unlock()

	return nil
}

// handleAgentDiscovered processes agent discovery events
func (b *AgentRegistrationBridge) handleAgentDiscovered(ctx context.Context, e events.CoreEvent) error {
	data := e.GetData()
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent_id not found in event", nil).
			WithComponent("AgentRegistrationBridge")
	}

	agentType, _ := data["agent_type"].(string)
	agentName, _ := data["agent_name"].(string)

	b.logger.InfoContext(ctx, "Agent discovered, registering with dispatcher",
		"agent_id", agentID,
		"agent_name", agentName,
		"agent_type", agentType)

	return b.registerAgent(ctx, agentID, agentType, agentName)
}

// handleCommissionProcessRequested ensures agents are registered before processing
func (b *AgentRegistrationBridge) handleCommissionProcessRequested(ctx context.Context, e events.CoreEvent) error {
	b.logger.InfoContext(ctx, "Commission processing requested, ensuring agents are registered")

	// Register all available agents to handle the commission
	return b.registerAgentsFromConfig(ctx)
}

// registerAgent creates and registers a single agent with the dispatcher
func (b *AgentRegistrationBridge) registerAgent(ctx context.Context, agentID, agentType, agentName string) error {
	// Check if agent is already registered
	b.mu.RLock()
	if _, exists := b.registeredAgents[agentID]; exists {
		b.mu.RUnlock()
		b.logger.InfoContext(ctx, "Agent already registered", "agent_id", agentID)
		return nil
	}
	b.mu.RUnlock()

	// Create agent instance using the factory
	agent, err := b.AgentFactory.CreateAgent(agentType, agentName)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent").
			WithComponent("AgentRegistrationBridge").
			WithDetails("agent_id", agentID).
			WithDetails("agent_type", agentType)
	}

	// Register with task dispatcher
	if err := b.TaskDispatcher.RegisterAgent(agent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent with dispatcher").
			WithComponent("AgentRegistrationBridge").
			WithDetails("agent_id", agentID)
	}

	// Track registered agent
	b.mu.Lock()
	b.registeredAgents[agentID] = agent
	b.mu.Unlock()

	// Emit registration success event
	event := events.NewBaseEvent(
		generateEventID(),
		"agent.registered_with_dispatcher",
		"agent_registration_bridge",
		map[string]interface{}{
			"agent_id":   agentID,
			"agent_name": agentName,
			"agent_type": agentType,
		},
	)
	b.eventBus.Publish(ctx, event)

	b.logger.InfoContext(ctx, "Agent registered with dispatcher",
		"agent_id", agentID,
		"agent_name", agentName,
		"agent_type", agentType)

	return nil
}

// registerAgentsFromConfig loads and registers agents from guild configuration
func (b *AgentRegistrationBridge) registerAgentsFromConfig(ctx context.Context) error {
	if !b.config.LoadFromGuildConfig {
		b.logger.InfoContext(ctx, "Loading from guild config disabled")
		return nil
	}

	// Get registered agents from the agent registry
	if b.AgentRegistry == nil {
		return gerror.New(gerror.ErrCodeInternal, "agent registry not available", nil).
			WithComponent("AgentRegistrationBridge")
	}

	// Get all registered agent configurations
	agentConfigs := b.AgentRegistry.GetRegisteredAgents()
	b.logger.InfoContext(ctx, "Found registered agents", "count", len(agentConfigs))

	registered := 0
	for _, agentConfig := range agentConfigs {
		// Check max agents limit
		if b.config.MaxAgents > 0 && registered >= b.config.MaxAgents {
			b.logger.InfoContext(ctx, "Reached max agents limit", "max_agents", b.config.MaxAgents)
			break
		}

		// Register the agent
		if err := b.registerAgent(ctx, agentConfig.ID, agentConfig.Type, agentConfig.Name); err != nil {
			b.logger.WithError(err).ErrorContext(ctx, "Failed to register agent",
				"agent_id", agentConfig.ID,
				"agent_type", agentConfig.Type)
			// Continue with other agents
			continue
		}
		registered++
	}

	b.logger.InfoContext(ctx, "Agent registration complete",
		"total_available", len(agentConfigs),
		"registered", registered,
		"max_agents", b.config.MaxAgents)

	return nil
}

// GetRegisteredAgentCount returns the number of registered agents
func (b *AgentRegistrationBridge) GetRegisteredAgentCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.registeredAgents)
}

// GetRegisteredAgents returns a copy of registered agents
func (b *AgentRegistrationBridge) GetRegisteredAgents() map[string]core.Agent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]core.Agent)
	for id, agent := range b.registeredAgents {
		result[id] = agent
	}
	return result
}
