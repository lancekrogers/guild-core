package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/agent"
	"github.com/blockhead-consulting/Guild/pkg/objective"
)

// BaseOrchestrator implements the Orchestrator interface
type BaseOrchestrator struct {
	status          Status
	agents          map[string]agent.Agent
	eventBus        *EventBus
	dispatcher      *TaskDispatcher
	currentObjective *objective.Objective
	config          *Config
	mu              sync.RWMutex
	cancelFunc      context.CancelFunc
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(config *Config, dispatcher *TaskDispatcher, eventBus *EventBus) *BaseOrchestrator {
	return &BaseOrchestrator{
		status:     StatusIdle,
		agents:     make(map[string]agent.Agent),
		eventBus:   eventBus,
		dispatcher: dispatcher,
		config:     config,
	}
}

// Start starts the orchestrator
func (o *BaseOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status == StatusRunning {
		return fmt.Errorf("orchestrator is already running")
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
				Type:   EventOrchestratorError,
				Source: "orchestrator",
				Data:   fmt.Sprintf("Dispatcher error: %v", err),
			})
		}
	}()

	// Update status
	o.status = StatusRunning
	
	// Emit started event
	o.eventBus.Publish(Event{
		Type:   EventOrchestratorStarted,
		Source: "orchestrator",
		Data:   o.config,
	})

	return nil
}

// Stop stops the orchestrator
func (o *BaseOrchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusRunning && o.status != StatusPaused {
		return fmt.Errorf("orchestrator is not running or paused")
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
		Type:   EventOrchestratorStopped,
		Source: "orchestrator",
		Data:   nil,
	})

	return nil
}

// Pause pauses the orchestrator
func (o *BaseOrchestrator) Pause(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusRunning {
		return fmt.Errorf("orchestrator is not running")
	}

	// Update status
	o.status = StatusPaused
	
	// Emit paused event
	o.eventBus.Publish(Event{
		Type:   EventOrchestratorPaused,
		Source: "orchestrator",
		Data:   nil,
	})

	return nil
}

// Resume resumes the orchestrator
func (o *BaseOrchestrator) Resume(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.status != StatusPaused {
		return fmt.Errorf("orchestrator is not paused")
	}

	// Update status
	o.status = StatusRunning
	
	// Emit resumed event
	o.eventBus.Publish(Event{
		Type:   EventOrchestratorResumed,
		Source: "orchestrator",
		Data:   nil,
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
	if _, exists := o.agents[agent.ID()]; exists {
		return fmt.Errorf("agent %s already exists", agent.ID())
	}

	// Add agent to the orchestrator
	o.agents[agent.ID()] = agent
	
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
		return fmt.Errorf("agent %s not found", agentID)
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
func (o *BaseOrchestrator) SetObjective(objective *objective.Objective) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.currentObjective = objective
	
	// Emit objective set event
	o.eventBus.Publish(Event{
		Type:   EventObjectiveSet,
		Source: "orchestrator",
		Data: map[string]string{
			"objective_id":    objective.ID,
			"objective_title": objective.Title,
		},
	})

	return nil
}

// GetObjective gets the current objective
func (o *BaseOrchestrator) GetObjective() *objective.Objective {
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