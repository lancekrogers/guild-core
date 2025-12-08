// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"sync"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// BaseOrchestrator implements the Orchestrator interface
type BaseOrchestrator struct {
	status            Status
	agents            map[string]core.Agent
	eventBus          EventBus
	dispatcher        TaskDispatcher
	currentCommission *commission.Commission
	config            *Config
	mu                sync.RWMutex
	cancelFunc        context.CancelFunc
}

// newOrchestrator creates a new orchestrator (private constructor)
func newOrchestrator(config *Config, dispatcher TaskDispatcher, eventBus EventBus) *BaseOrchestrator {
	return &BaseOrchestrator{
		status:     StatusIdle,
		agents:     make(map[string]core.Agent),
		eventBus:   eventBus,
		dispatcher: dispatcher,
		config:     config,
	}
}

// DefaultOrchestratorFactory creates an orchestrator factory for registry use
func DefaultOrchestratorFactory(config *Config, dispatcher TaskDispatcher, eventBus EventBus) *BaseOrchestrator {
	return newOrchestrator(config, dispatcher, eventBus)
}

// Start starts the orchestrator
func (o *BaseOrchestrator) Start(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("Start")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status == StatusRunning {
		return gerror.New(gerror.ErrCodeValidation, "orchestrator is already running", nil).
			WithComponent("orchestrator").
			WithOperation("Start")
	}

	// Create a cancellable context for cleanup
	_, cancel := context.WithCancel(ctx)
	o.cancelFunc = cancel

	// Dispatcher is now ready to accept tasks via Dispatch method
	// No background running needed as it operates on-demand

	// Update status
	o.status = StatusRunning

	// Emit started event
	o.eventBus.Publish(Event{
		Type:   EventType(EventOrchestratorStarted),
		Source: "orchestrator",
		Data:   map[string]interface{}{"config": o.config},
	})

	return nil
}

// Stop stops the orchestrator
func (o *BaseOrchestrator) Stop(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("Stop")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusRunning && o.status != StatusPaused {
		return gerror.New(gerror.ErrCodeValidation, "orchestrator is not running or paused", nil).
			WithComponent("orchestrator").
			WithOperation("Stop")
	}

	// Cancel the run context
	if o.cancelFunc != nil {
		o.cancelFunc()
		o.cancelFunc = nil
	}

	// Update status
	o.status = StatusIdle

	// Emit stopped event
	o.eventBus.Publish(Event{
		Type:   EventType(EventOrchestratorStopped),
		Source: "orchestrator",
		Data:   map[string]interface{}{},
	})

	return nil
}

// Pause pauses the orchestrator
func (o *BaseOrchestrator) Pause(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator").
			WithOperation("Pause")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusRunning {
		return gerror.New(gerror.ErrCodeValidation, "orchestrator is not running", nil).
			WithComponent("orchestrator").
			WithOperation("Pause")
	}

	// Update status
	o.status = StatusPaused

	// Emit paused event
	o.eventBus.Publish(Event{
		Type:   EventType(EventOrchestratorPaused),
		Source: "orchestrator",
		Data:   map[string]interface{}{},
	})

	return nil
}

// Resume resumes the orchestrator
func (o *BaseOrchestrator) Resume(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusPaused {
		return gerror.New(gerror.ErrCodeValidation, "orchestrator is not paused", nil).
			WithComponent("orchestrator").
			WithOperation("Resume")
	}

	// Update status
	o.status = StatusRunning

	// Emit resumed event
	o.eventBus.Publish(Event{
		Type:   EventType(EventOrchestratorResumed),
		Source: "orchestrator",
		Data:   map[string]interface{}{},
	})

	return nil
}

// Status returns the current status
func (o *BaseOrchestrator) Status() Status {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.status
}

// AddAgent adds an agent to the orchestrator
func (o *BaseOrchestrator) AddAgent(agent core.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if agent already exists
	if _, exists := o.agents[agent.GetID()]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "agent already exists", nil).
			WithComponent("orchestrator").
			WithOperation("AddAgent").
			WithDetails("agent_id", agent.GetID())
	}

	// Add agent to the orchestrator
	o.agents[agent.GetID()] = agent

	// Add agent to the dispatcher
	o.dispatcher.RegisterAgent(agent)

	return nil
}

// RemoveAgent removes an agent from the orchestrator
func (o *BaseOrchestrator) RemoveAgent(agentID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if agent exists
	if _, exists := o.agents[agentID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "agent not found", nil).
			WithComponent("orchestrator").
			WithOperation("RemoveAgent").
			WithDetails("agent_id", agentID)
	}

	// Remove agent from the orchestrator
	delete(o.agents, agentID)

	// Remove agent from the dispatcher
	o.dispatcher.UnregisterAgent(agentID)

	return nil
}

// GetAgent gets an agent by ID
func (o *BaseOrchestrator) GetAgent(agentID string) (core.Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.agents[agentID]
	return agent, exists
}

// SetCommission sets the current commission
func (o *BaseOrchestrator) SetCommission(commission *commission.Commission) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.currentCommission = commission

	// Emit commission set event
	o.eventBus.Publish(Event{
		Type:   EventType(EventCommissionSet),
		Source: "orchestrator",
		Data: map[string]interface{}{
			"commission_id":    commission.ID,
			"commission_title": commission.Title,
		},
	})

	return nil
}

// GetCommission gets the current commission
func (o *BaseOrchestrator) GetCommission() *commission.Commission {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.currentCommission
}

// AddEventHandler adds an event handler
func (o *BaseOrchestrator) AddEventHandler(handler EventHandler) {
	o.eventBus.SubscribeAll(handler)
}

// EmitEvent emits an event
func (o *BaseOrchestrator) EmitEvent(event Event) {
	o.eventBus.Publish(event)
}
