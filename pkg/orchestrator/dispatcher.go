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
	kanbanManager *kanban.Manager
	agentFactory  *agent.Factory
	agentPool     map[string]agent.Agent
	activeAgents  map[string]agent.Agent
	maxAgents     int
	mu            sync.Mutex
	eventBus      *EventBus
}

// NewTaskDispatcher creates a new task dispatcher
func NewTaskDispatcher(kanbanManager *kanban.Manager, agentFactory *agent.Factory, eventBus *EventBus, maxAgents int) *TaskDispatcher {
	return &TaskDispatcher{
		kanbanManager: kanbanManager,
		agentFactory:  agentFactory,
		agentPool:     make(map[string]agent.Agent),
		activeAgents:  make(map[string]agent.Agent),
		maxAgents:     maxAgents,
		eventBus:      eventBus,
	}
}

// RegisterAgent registers an agent with the dispatcher
func (d *TaskDispatcher) RegisterAgent(agent agent.Agent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.agentPool[agent.ID()] = agent
	
	// Emit agent added event
	d.eventBus.Publish(Event{
		Type:   EventAgentAdded,
		Source: "dispatcher",
		Data:   agent.ID(),
		Metadata: map[string]string{
			"agent_name": agent.Name(),
			"agent_type": agent.Type(),
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
		Type:   EventAgentRemoved,
		Source: "dispatcher",
		Data:   agentID,
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
	tasks, err := d.kanbanManager.ListTasksByStatus(ctx, kanban.StatusTodo)
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
		if _, active := d.activeAgents[id]; !active && agent.Status() == agent.StatusIdle {
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
		
		// Assign the task to the agent
		if err := agent.AssignTask(ctx, task); err != nil {
			// Log and continue with other tasks
			fmt.Printf("Error assigning task %s to agent %s: %v\n", task.ID, agent.ID(), err)
			continue
		}

		// Mark the agent as active
		d.activeAgents[agent.ID()] = agent
		
		// Update the task in the kanban board
		if err := d.kanbanManager.UpdateTaskStatus(ctx, task.ID, kanban.StatusInProgress, agent.ID(), "Assigned to agent"); err != nil {
			fmt.Printf("Error updating task status: %v\n", err)
		}
		
		// Emit task assigned event
		d.eventBus.Publish(Event{
			Type:   EventTaskAssigned,
			Source: "dispatcher",
			Data: map[string]string{
				"task_id":   task.ID,
				"agent_id":  agent.ID(),
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
			Type:   EventAgentStarted,
			Source: "dispatcher",
			Data:   agentID,
		})
		
		// Execute the agent
		err := agent.Execute(execCtx)
		
		// Agent execution completed
		d.mu.Lock()
		delete(d.activeAgents, agentID)
		d.mu.Unlock()
		
		if err != nil {
			// Emit agent failed event
			d.eventBus.Publish(Event{
				Type:   EventAgentFailed,
				Source: "dispatcher",
				Data:   agentID,
				Metadata: map[string]string{
					"error": err.Error(),
				},
			})
		} else {
			// Emit agent completed event
			d.eventBus.Publish(Event{
				Type:   EventAgentCompleted,
				Source: "dispatcher",
				Data:   agentID,
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
		if _, active := d.activeAgents[id]; !active && agent.Status() == agent.StatusIdle {
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
				if agent.Status() == agent.StatusWorking {
					// This agent is already working, check if it has a task
					// This might need refinement depending on how agent status is tracked
					if state := agent.GetState(); state != nil && state.CurrentTask != "" && state.Status == agent.StatusWorking {
						// Agent has a task but might not be executing yet
						d.StartAgent(ctx, agent.ID())
					}
				}
			}
		}
	}
}