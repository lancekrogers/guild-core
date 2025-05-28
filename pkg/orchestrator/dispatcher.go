package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// TaskDispatcher is responsible for assigning tasks to agents
type TaskDispatcher struct {
	kanbanManager KanbanManager
	agentFactory  AgentFactory
	agentPool     map[string]agent.Agent
	activeAgents  map[string]agent.Agent
	agentTasks    map[string]*kanban.Task // Maps agent ID to their current task
	maxAgents     int
	mu            sync.Mutex
	eventBus      *EventBus
}

// NewTaskDispatcher creates a new task dispatcher
func NewTaskDispatcher(kanbanManager KanbanManager, agentFactory AgentFactory, eventBus *EventBus, maxAgents int) *TaskDispatcher {
	return &TaskDispatcher{
		kanbanManager: kanbanManager,
		agentFactory:  agentFactory,
		agentPool:     make(map[string]agent.Agent),
		activeAgents:  make(map[string]agent.Agent),
		agentTasks:    make(map[string]*kanban.Task),
		maxAgents:     maxAgents,
		eventBus:      eventBus,
	}
}

// RegisterAgent registers an agent with the dispatcher
func (d *TaskDispatcher) RegisterAgent(agent agent.Agent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.agentPool[agent.GetID()] = agent
	
	// Emit agent added event
	d.eventBus.Publish(Event{
		Type:   EventType(EventAgentAdded),
		Source: "dispatcher",
		Data:   map[string]interface{}{
			"agent_id":   agent.GetID(),
			"agent_name": agent.GetName(),
		},
	})
}

// UnregisterAgent unregisters an agent from the dispatcher
func (d *TaskDispatcher) UnregisterAgent(agentID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.agentPool, agentID)
	delete(d.activeAgents, agentID)
	
	// Emit agent removed event
	d.eventBus.Publish(Event{
		Type:   EventType(EventAgentRemoved),
		Source: "dispatcher",
		Data:   map[string]interface{}{"agent_id": agentID},
	})
}

// DispatchTasks assigns tasks to available agents
func (d *TaskDispatcher) DispatchTasks(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if we can run more agents
	availableSlots := d.maxAgents - len(d.activeAgents)
	if availableSlots <= 0 {
		return nil // No slots available
	}

	// Get todo tasks
	// TODO: The boardID should be configurable
	tasks, err := d.kanbanManager.ListTasksByStatus(ctx, "default", kanban.StatusTodo)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil // No tasks to dispatch
	}
	
	// Sort tasks by priority

	// Find available agents
	var availableAgents []agent.Agent
	for id, agent := range d.agentPool {
		if _, active := d.activeAgents[id]; !active {
			availableAgents = append(availableAgents, agent)
			if len(availableAgents) >= availableSlots {
				break
			}
		}
	}

	// Dispatch tasks to available agents
	for i, task := range tasks {
		if i >= len(availableAgents) {
			break
		}

		agent := availableAgents[i]
		
		// Store task assignment
		d.agentTasks[agent.GetID()] = task

		// Mark the agent as active
		d.activeAgents[agent.GetID()] = agent
		
		// Update the task in the kanban board
		if err := d.kanbanManager.UpdateTaskStatus(ctx, task.ID, string(kanban.StatusInProgress), agent.GetID(), "Assigned to agent"); err != nil {
			fmt.Printf("Error updating task status: %v\n", err)
		}
		
		// Emit task assigned event
		d.eventBus.Publish(Event{
			Type:   EventType(EventTaskAssigned),
			Source: "dispatcher",
			Data: map[string]interface{}{
				"task_id":   task.ID,
				"agent_id":  agent.GetID(),
				"task_title": task.Title,
			},
		})
	}

	return nil
}

// StartAgent starts an agent's execution
func (d *TaskDispatcher) StartAgent(ctx context.Context, agentID string) error {
	d.mu.Lock()
	agent, exists := d.activeAgents[agentID]
	d.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("agent %s not found or not active", agentID)
	}
	
	// Start the agent's execution in a goroutine
	go func() {
		// Create a new context with timeout
		execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		
		// Emit agent started event
		d.eventBus.Publish(Event{
			Type:   EventType(EventAgentStarted),
			Source: "dispatcher",
			Data:   map[string]interface{}{"agent_id": agentID},
		})
		
		// Get the task assigned to this agent
		task, hasTask := d.agentTasks[agentID]
		taskRequest := "Execute assigned task"
		if hasTask && task != nil {
			taskRequest = fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
		}
		
		// Execute the agent with the task details
		_, err := agent.Execute(execCtx, taskRequest)
		
		// Agent execution completed
		d.mu.Lock()
		delete(d.activeAgents, agentID)
		delete(d.agentTasks, agentID)
		d.mu.Unlock()
		
		if err != nil {
			// Emit agent failed event
			d.eventBus.Publish(Event{
				Type:   EventType(EventAgentFailed),
				Source: "dispatcher",
				Data: map[string]interface{}{
					"agent_id": agentID,
					"error":    err.Error(),
				},
			})
		} else {
			// Emit agent completed event
			d.eventBus.Publish(Event{
				Type:   EventType(EventAgentCompleted),
				Source: "dispatcher",
				Data:   map[string]interface{}{"agent_id": agentID},
			})
		}
	}()
	
	return nil
}

// GetActiveAgents returns the list of active agents
func (d *TaskDispatcher) GetActiveAgents() []agent.Agent {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	var agents []agent.Agent
	for _, agent := range d.activeAgents {
		agents = append(agents, agent)
	}
	
	return agents
}

// GetAvailableAgents returns the list of available agents
func (d *TaskDispatcher) GetAvailableAgents() []agent.Agent {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	var agents []agent.Agent
	for id, agent := range d.agentPool {
		if _, active := d.activeAgents[id]; !active {
			agents = append(agents, agent)
		}
	}
	
	return agents
}

// Run runs the dispatcher in a loop
func (d *TaskDispatcher) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.DispatchTasks(ctx); err != nil {
				// Log and continue
				fmt.Printf("Error dispatching tasks: %v\n", err)
			}
			
			// Start any agents that are ready
			for _, agent := range d.GetActiveAgents() {
				if _, isActive := d.activeAgents[agent.GetID()]; isActive {
					// Agent is active, start execution
					d.StartAgent(ctx, agent.GetID())
				}
			}
		}
	}
}