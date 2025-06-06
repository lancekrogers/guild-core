package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/internal/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// BaseOrchestrator implements the Orchestrator interface
type BaseOrchestrator struct {
	status          Status
	agents          map[string]agent.Agent
	eventBus        *EventBus
	dispatcher      *TaskDispatcher
	currentObjective *commission.Commission
	config          *Config
	mu              sync.RWMutex
	cancelFunc      context.CancelFunc
}

// newOrchestrator creates a new orchestrator (private constructor)
func newOrchestrator(config *Config, dispatcher *TaskDispatcher, eventBus *EventBus) *BaseOrchestrator {
	return &BaseOrchestrator{
		status:     StatusIdle,
		agents:     make(map[string]agent.Agent),
		eventBus:   eventBus,
		dispatcher: dispatcher,
		config:     config,
	}
}

// DefaultOrchestratorFactory creates an orchestrator factory for registry use
func DefaultOrchestratorFactory(config *Config, dispatcher *TaskDispatcher, eventBus *EventBus) *BaseOrchestrator {
	return newOrchestrator(config, dispatcher, eventBus)
}

// Start starts the orchestrator
func (o *BaseOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status == StatusRunning {
		return gerror.New(gerror.ErrCodeValidation, "orchestrator is already running", nil).
			WithComponent("orchestrator").
			WithOperation("Start")
	}

	// Create a cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	o.cancelFunc = cancel

	// Start the dispatcher
	go func() {
		if err := o.dispatcher.Run(runCtx, 5*time.Second); err != nil && err != context.Canceled {
			fmt.Printf("Dispatcher error: %v\n", err)
			
			// Emit error event
			o.eventBus.Publish(Event{
				Type:   EventType(EventOrchestratorError),
				Source: "orchestrator",
				Data:   map[string]interface{}{"error": err.Error()},
			})
		}
	}()

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
func (o *BaseOrchestrator) AddAgent(agent agent.Agent) error {
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
func (o *BaseOrchestrator) GetAgent(agentID string) (agent.Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.agents[agentID]
	return agent, exists
}

// SetObjective sets the current objective
func (o *BaseOrchestrator) SetObjective(objective *commission.Commission) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.currentObjective = objective
	
	// Emit objective set event
	o.eventBus.Publish(Event{
		Type:   EventType(EventObjectiveSet),
		Source: "orchestrator",
		Data: map[string]interface{}{
			"objective_id":    objective.ID,
			"objective_title": objective.Title,
		},
	})

	return nil
}

// GetObjective gets the current objective
func (o *BaseOrchestrator) GetObjective() *commission.Commission {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	return o.currentObjective
}

// AddEventHandler adds an event handler
func (o *BaseOrchestrator) AddEventHandler(handler EventHandler) {
	o.eventBus.SubscribeAll(handler)
}

// EmitEvent emits an event
func (o *BaseOrchestrator) EmitEvent(event Event) {
	o.eventBus.Publish(event)
}