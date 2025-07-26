// Package bridges provides integration bridges between different components
package bridges

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// OrchestratorCampaignBridge connects campaigns to the orchestration system
type OrchestratorCampaignBridge struct {
	eventBus      events.EventBus
	logger        observability.Logger
	subscriptions map[string]events.SubscriptionID
	mu            sync.RWMutex

	// Dependencies injected via constructor
	campaignManager       CampaignManager
	commissionManager     CommissionManager
	taskDispatcher        TaskDispatcher
	agentRegistry         AgentRegistry
	config                *OrchestratorCampaignConfig
	processCommissionFunc ProcessCommissionFunc
}

// OrchestratorCampaignConfig configures the orchestrator campaign bridge
type OrchestratorCampaignConfig struct {
	Enabled                bool
	ProcessCommissionsSync bool
	MaxConcurrentAgents    int
}

// Minimal interfaces to avoid import cycles
type CampaignManager interface {
	Get(ctx context.Context, id string) (*Campaign, error)
	MarkReady(ctx context.Context, id string) error
}

type CommissionManager interface {
	Get(ctx context.Context, id string) (*Commission, error)
}

type TaskDispatcher interface {
	DispatchTasks(ctx context.Context) error
	RegisterAgent(agent core.Agent) error
}

type AgentRegistry interface {
	GetRegisteredAgents() []AgentConfig
}

type Agent interface {
	GetID() string
	GetType() string
	GetCapabilities() []string
}

type AgentConfig struct {
	ID           string
	Name         string
	Type         string
	Provider     string
	Model        string
	Capabilities []string
}

type Campaign struct {
	ID          string
	Name        string
	Status      string
	Commissions []string
}

type Commission struct {
	ID          string
	Title       string
	Description string
}

// ProcessCommissionFunc is a function that processes a commission into tasks
type ProcessCommissionFunc func(ctx context.Context, commissionID string) error

// NewOrchestratorCampaignBridge creates a new orchestrator campaign bridge
func NewOrchestratorCampaignBridge(
	eventBus events.EventBus,
	logger observability.Logger,
	config OrchestratorCampaignConfig,
	campaignManager CampaignManager,
	commissionManager CommissionManager,
	taskDispatcher TaskDispatcher,
	agentRegistry AgentRegistry,
) *OrchestratorCampaignBridge {
	return &OrchestratorCampaignBridge{
		eventBus:          eventBus,
		logger:            logger.WithComponent("OrchestratorCampaignBridge"),
		subscriptions:     make(map[string]events.SubscriptionID),
		config:            &config,
		campaignManager:   campaignManager,
		commissionManager: commissionManager,
		taskDispatcher:    taskDispatcher,
		agentRegistry:     agentRegistry,
	}
}

// SetProcessCommissionFunc sets the function used to process commissions
// This allows injection of the actual processing logic without import cycles
func (b *OrchestratorCampaignBridge) SetProcessCommissionFunc(fn ProcessCommissionFunc) {
	b.processCommissionFunc = fn
}

// Start starts the bridge and subscribes to events
func (b *OrchestratorCampaignBridge) Start(ctx context.Context) error {
	if !b.config.Enabled {
		b.logger.InfoContext(ctx, "Orchestrator campaign bridge disabled")
		return nil
	}

	b.logger.InfoContext(ctx, "Starting orchestrator campaign bridge")

	// Subscribe to campaign events
	if err := b.subscribeToCampaignEvents(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to campaign events").
			WithComponent("OrchestratorCampaignBridge")
	}

	// Initialize agents from registry
	if err := b.initializeAgents(ctx); err != nil {
		b.logger.WithError(err).WarnContext(ctx, "Failed to initialize agents")
		// Don't fail startup - agents can be registered later
	}

	b.logger.InfoContext(ctx, "Orchestrator campaign bridge started")
	return nil
}

// Stop stops the bridge and unsubscribes from events
func (b *OrchestratorCampaignBridge) Stop(ctx context.Context) error {
	b.logger.InfoContext(ctx, "Stopping orchestrator campaign bridge")

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Unsubscribe from all events
	for eventType, subID := range b.subscriptions {
		if err := b.eventBus.Unsubscribe(ctx, subID); err != nil {
			b.logger.WithError(err).ErrorContext(ctx, "Failed to unsubscribe from event",
				"event_type", eventType)
		}
	}

	b.logger.InfoContext(ctx, "Orchestrator campaign bridge stopped")
	return nil
}

// Health returns nil if the bridge is healthy
func (b *OrchestratorCampaignBridge) Health(ctx context.Context) error {
	// Check if we have the required dependencies
	if b.campaignManager == nil {
		return gerror.New(gerror.ErrCodeInternal, "campaign manager not available", nil).
			WithComponent("OrchestratorCampaignBridge")
	}
	if b.taskDispatcher == nil {
		return gerror.New(gerror.ErrCodeInternal, "task dispatcher not available", nil).
			WithComponent("OrchestratorCampaignBridge")
	}
	return nil
}

// Ready returns nil if the bridge is ready to process events
func (b *OrchestratorCampaignBridge) Ready(ctx context.Context) error {
	if !b.config.Enabled {
		return nil // Always ready if disabled
	}
	return b.Health(ctx)
}

// subscribeToCampaignEvents sets up event subscriptions
func (b *OrchestratorCampaignBridge) subscribeToCampaignEvents(ctx context.Context) error {
	// Subscribe to campaign started event
	subID, err := b.eventBus.Subscribe(ctx, "campaign.started", func(ctx context.Context, e events.CoreEvent) error {
		return b.handleCampaignStarted(ctx, e)
	})
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.subscriptions["campaign.started"] = subID
	b.mu.Unlock()

	// Subscribe to campaign planning started event
	subID, err = b.eventBus.Subscribe(ctx, "campaign.planning_started", func(ctx context.Context, e events.CoreEvent) error {
		return b.handleCampaignPlanningStarted(ctx, e)
	})
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.subscriptions["campaign.planning_started"] = subID
	b.mu.Unlock()

	return nil
}

// handleCampaignStarted processes campaign start events
func (b *OrchestratorCampaignBridge) handleCampaignStarted(ctx context.Context, e events.CoreEvent) error {
	data := e.GetData()
	campaignID, ok := data["campaign_id"].(string)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign_id not found in event", nil).
			WithComponent("OrchestratorCampaignBridge")
	}

	b.logger.InfoContext(ctx, "Campaign started, processing commissions", "campaign_id", campaignID)

	// Get campaign
	campaign, err := b.campaignManager.Get(ctx, campaignID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get campaign").
			WithComponent("OrchestratorCampaignBridge").
			WithDetails("campaign_id", campaignID)
	}

	// Process each commission
	if b.config.ProcessCommissionsSync {
		for _, commissionID := range campaign.Commissions {
			if err := b.processCommission(ctx, commissionID, campaignID); err != nil {
				b.logger.WithError(err).ErrorContext(ctx, "Failed to process commission",
					"commission_id", commissionID,
					"campaign_id", campaignID)
				// Continue with other commissions
			}
		}
	}

	// Start task dispatcher
	go func() {
		dispatchCtx := context.Background()
		if err := b.taskDispatcher.DispatchTasks(dispatchCtx); err != nil {
			b.logger.WithError(err).ErrorContext(dispatchCtx, "Failed to dispatch tasks")
		}
	}()

	return nil
}

// handleCampaignPlanningStarted processes campaign planning events
func (b *OrchestratorCampaignBridge) handleCampaignPlanningStarted(ctx context.Context, e events.CoreEvent) error {
	data := e.GetData()
	campaignID, ok := data["campaign_id"].(string)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign_id not found in event", nil).
			WithComponent("OrchestratorCampaignBridge")
	}

	b.logger.InfoContext(ctx, "Campaign planning started", "campaign_id", campaignID)

	// Get campaign
	campaign, err := b.campaignManager.Get(ctx, campaignID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get campaign").
			WithComponent("OrchestratorCampaignBridge").
			WithDetails("campaign_id", campaignID)
	}

	// Create tasks for each commission (planning phase)
	for _, commissionID := range campaign.Commissions {
		b.logger.InfoContext(ctx, "Planning tasks for commission",
			"commission_id", commissionID,
			"campaign_id", campaignID)

		if err := b.processCommission(ctx, commissionID, campaignID); err != nil {
			b.logger.WithError(err).ErrorContext(ctx, "Failed to plan commission",
				"commission_id", commissionID,
				"campaign_id", campaignID)
		}
	}

	// Mark campaign as ready
	if err := b.campaignManager.MarkReady(ctx, campaignID); err != nil {
		b.logger.WithError(err).ErrorContext(ctx, "Failed to mark campaign as ready",
			"campaign_id", campaignID)
	}

	return nil
}

// processCommission processes a commission (delegates to injected function)
func (b *OrchestratorCampaignBridge) processCommission(ctx context.Context, commissionID, campaignID string) error {
	// Use the injected processing function if available
	if b.processCommissionFunc != nil {
		return b.processCommissionFunc(ctx, commissionID)
	}

	// Otherwise, just emit an event that commission needs processing
	event := events.NewBaseEvent(
		generateEventID(),
		"commission.process_requested",
		"orchestrator_campaign_bridge",
		map[string]interface{}{
			"commission_id": commissionID,
			"campaign_id":   campaignID,
		},
	)
	b.eventBus.Publish(ctx, event)

	return nil
}

// initializeAgents initializes agents from the registry
func (b *OrchestratorCampaignBridge) initializeAgents(ctx context.Context) error {
	if b.agentRegistry == nil {
		return gerror.New(gerror.ErrCodeInternal, "agent registry not available", nil).
			WithComponent("OrchestratorCampaignBridge")
	}

	agents := b.agentRegistry.GetRegisteredAgents()
	b.logger.InfoContext(ctx, "Initializing agents", "count", len(agents))

	for _, agentConfig := range agents {
		b.logger.InfoContext(ctx, "Agent configuration found",
			"agent_id", agentConfig.ID,
			"agent_name", agentConfig.Name,
			"agent_type", agentConfig.Type)

		// Emit event for agent discovery
		event := events.NewBaseEvent(
			generateEventID(),
			"agent.discovered",
			"orchestrator_campaign_bridge",
			map[string]interface{}{
				"agent_id":     agentConfig.ID,
				"agent_name":   agentConfig.Name,
				"agent_type":   agentConfig.Type,
				"provider":     agentConfig.Provider,
				"model":        agentConfig.Model,
				"capabilities": agentConfig.Capabilities,
			},
		)
		b.eventBus.Publish(ctx, event)
	}

	return nil
}

// Name returns the service name
func (b *OrchestratorCampaignBridge) Name() string {
	return "orchestrator-campaign-bridge"
}

// generateEventID creates a simple random event ID
func generateEventID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("event-%x", b)
}
