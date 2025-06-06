package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// taskDispatcher is responsible for assigning tasks to agents
type taskDispatcher struct {
	kanbanManager KanbanManager
	agentFactory  AgentFactory
	agentPool     map[string]agent.Agent
	activeAgents  map[string]agent.Agent
	agentTasks    map[string]*kanban.Task // Maps agent ID to their current task
	maxAgents     int
	mu            sync.Mutex
	eventBus      EventBus
}

// newTaskDispatcher creates a new task dispatcher (private constructor)
func newTaskDispatcher(kanbanManager KanbanManager, agentFactory AgentFactory, eventBus EventBus, maxAgents int) *taskDispatcher {
	return &taskDispatcher{
		kanbanManager: kanbanManager,
		agentFactory:  agentFactory,
		agentPool:     make(map[string]agent.Agent),
		activeAgents:  make(map[string]agent.Agent),
		agentTasks:    make(map[string]*kanban.Task),
		maxAgents:     maxAgents,
		eventBus:      eventBus,
	}
}

// DefaultTaskDispatcherFactory creates a task dispatcher for registry use
func DefaultTaskDispatcherFactory(kanbanManager KanbanManager, agentFactory AgentFactory, eventBus EventBus, maxAgents int) TaskDispatcher {
	return newTaskDispatcher(kanbanManager, agentFactory, eventBus, maxAgents)
}

// RegisterAgent registers an agent with the dispatcher
func (d *taskDispatcher) RegisterAgent(agent agent.Agent) {
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
func (d *taskDispatcher) UnregisterAgent(agentID string) {
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
func (d *taskDispatcher) DispatchTasks(ctx context.Context) error {
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
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to list tasks").
			WithComponent("orchestrator").
			WithOperation("DispatchTasks")
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
func (d *taskDispatcher) StartAgent(ctx context.Context, agentID string) error {
	d.mu.Lock()
	agent, exists := d.activeAgents[agentID]
	d.mu.Unlock()
	
	if !exists {
		return gerror.New(gerror.ErrCodeAgentNotFound, "agent not found or not active", nil).
			WithComponent("orchestrator").
			WithOperation("StartAgent").
			WithDetails("agent_id", agentID)
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
func (d *taskDispatcher) GetActiveAgents() []agent.Agent {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	var agents []agent.Agent
	for _, agent := range d.activeAgents {
		agents = append(agents, agent)
	}
	
	return agents
}

// GetAvailableAgents returns the list of available agents
func (d *taskDispatcher) GetAvailableAgents() []agent.Agent {
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

// ListAvailableAgents returns agents that can accept tasks (implements interface)
func (d *taskDispatcher) ListAvailableAgents() []agent.Agent {
	return d.GetAvailableAgents()
}

// Dispatch assigns a task to an available agent (implements interface)
func (d *taskDispatcher) Dispatch(ctx context.Context, task *kanban.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Find an available agent
	for id, agent := range d.agentPool {
		if _, active := d.activeAgents[id]; !active {
			// Assign task to agent
			d.activeAgents[id] = agent
			d.agentTasks[id] = task
			
			// Start the agent
			go d.StartAgent(ctx, id)
			
			return nil
		}
	}
	
	return gerror.New(gerror.ErrCodeNoAvailableAgent, "no available agents to handle task", nil).
		WithComponent("orchestrator").
		WithOperation("Dispatch").
		WithDetails("task_id", task.ID)
}

// GetTaskStatus returns the current status of a task (implements interface)
func (d *taskDispatcher) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Find task in agent assignments
	for agentID, task := range d.agentTasks {
		if task.ID == taskID {
			return TaskStatus{
				TaskID:    taskID,
				AgentID:   agentID,
				Status:    string(task.Status),
				StartTime: time.Now(), // This would need proper tracking
				Error:     nil,
			}, nil
		}
	}
	
	return TaskStatus{}, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
		WithComponent("orchestrator").
		WithOperation("GetTaskStatus").
		WithDetails("task_id", taskID)
}

// GetAgentStatus returns the current status of an agent (implements interface)
func (d *taskDispatcher) GetAgentStatus(agentID string) AgentStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	_, isActive := d.activeAgents[agentID]
	currentTask := ""
	if task, exists := d.agentTasks[agentID]; exists {
		currentTask = task.ID
	}
	
	return AgentStatus{
		AgentID:      agentID,
		Available:    !isActive,
		CurrentTask:  currentTask,
		TasksHandled: 0, // This would need proper tracking
	}
}

// Stop gracefully shuts down the dispatcher (implements interface)
func (d *taskDispatcher) Stop(ctx context.Context) error {
	// Signal all active agents to stop
	// This would need proper implementation
	return nil
}

// Run runs the dispatcher in a loop
func (d *taskDispatcher) Run(ctx context.Context, interval time.Duration) error {
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