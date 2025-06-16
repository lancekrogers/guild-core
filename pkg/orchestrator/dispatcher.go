// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/observability"
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
		Data: map[string]interface{}{
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
	// Initialize observability for multi-agent task dispatch
	logger := observability.GetLogger(ctx).
		WithComponent("orchestrator").
		WithOperation("DispatchTasks")

	// Start dispatch timing
	start := time.Now()
	logger.InfoContext(ctx, "Starting task dispatch",
		"max_agents", d.maxAgents,
	)

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if we can run more agents
	availableSlots := d.maxAgents - len(d.activeAgents)
	currentActiveAgents := len(d.activeAgents)
	poolSize := len(d.agentPool)

	logger.DebugContext(ctx, "Agent capacity analysis",
		"active_agents", currentActiveAgents,
		"agent_pool_size", poolSize,
		"available_slots", availableSlots,
		"max_agents", d.maxAgents,
	)

	if availableSlots <= 0 {
		logger.DebugContext(ctx, "No available agent slots - dispatch skipped")
		return nil // No slots available
	}

	// Get todo tasks
	taskListStart := time.Now()
	// TODO: The boardID should be configurable
	tasks, err := d.kanbanManager.ListTasksByStatus(ctx, "default", kanban.StatusTodo)
	taskListDuration := time.Since(taskListStart)

	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "Failed to list todo tasks",
			"task_list_duration_ms", taskListDuration.Milliseconds(),
		)
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to list tasks").
			WithComponent("orchestrator").
			WithOperation("DispatchTasks")
	}

	logger.DebugContext(ctx, "Retrieved todo tasks",
		"todo_tasks_count", len(tasks),
		"task_list_duration_ms", taskListDuration.Milliseconds(),
	)

	if len(tasks) == 0 {
		logger.DebugContext(ctx, "No tasks to dispatch - returning")
		return nil // No tasks to dispatch
	}

	// Sort tasks by priority
	logger.DebugContext(ctx, "TODO: Task priority sorting not yet implemented")

	// Find available agents
	agentSelectionStart := time.Now()
	var availableAgents []agent.Agent
	for id, agent := range d.agentPool {
		if _, active := d.activeAgents[id]; !active {
			availableAgents = append(availableAgents, agent)
			if len(availableAgents) >= availableSlots {
				break
			}
		}
	}
	agentSelectionDuration := time.Since(agentSelectionStart)

	logger.DebugContext(ctx, "Agent selection completed",
		"available_agents_count", len(availableAgents),
		"agents_to_dispatch", min(len(tasks), len(availableAgents)),
		"agent_selection_duration_ms", agentSelectionDuration.Milliseconds(),
	)

	// Dispatch tasks to available agents
	var dispatchedTasks []string
	var dispatchedAgents []string

	for i, task := range tasks {
		if i >= len(availableAgents) {
			break
		}

		agent := availableAgents[i]
		assignmentStart := time.Now()

		logger.DebugContext(ctx, "Assigning task to agent",
			"task_id", task.ID,
			"task_title", task.Title,
			"agent_id", agent.GetID(),
			"agent_name", agent.GetName(),
		)

		// Store task assignment
		d.agentTasks[agent.GetID()] = task

		// Mark the agent as active
		d.activeAgents[agent.GetID()] = agent

		// Update the task in the kanban board
		kanbanUpdateStart := time.Now()
		if err := d.kanbanManager.UpdateTaskStatus(ctx, task.ID, string(kanban.StatusInProgress), agent.GetID(), "Assigned to agent"); err != nil {
			kanbanUpdateDuration := time.Since(kanbanUpdateStart)
			logger.WithError(err).WarnContext(ctx, "Failed to update task status in kanban",
				"task_id", task.ID,
				"agent_id", agent.GetID(),
				"kanban_update_duration_ms", kanbanUpdateDuration.Milliseconds(),
			)
		} else {
			kanbanUpdateDuration := time.Since(kanbanUpdateStart)
			logger.DebugContext(ctx, "Kanban status updated",
				"task_id", task.ID,
				"new_status", kanban.StatusInProgress,
				"kanban_update_duration_ms", kanbanUpdateDuration.Milliseconds(),
			)
		}

		// Emit task assigned event
		eventStart := time.Now()
		d.eventBus.Publish(Event{
			Type:   EventType(EventTaskAssigned),
			Source: "dispatcher",
			Data: map[string]interface{}{
				"task_id":    task.ID,
				"agent_id":   agent.GetID(),
				"task_title": task.Title,
			},
		})
		eventDuration := time.Since(eventStart)

		assignmentDuration := time.Since(assignmentStart)
		logger.DebugContext(ctx, "Task assignment completed",
			"task_id", task.ID,
			"agent_id", agent.GetID(),
			"assignment_duration_ms", assignmentDuration.Milliseconds(),
			"event_publish_duration_ms", eventDuration.Milliseconds(),
		)

		dispatchedTasks = append(dispatchedTasks, task.ID)
		dispatchedAgents = append(dispatchedAgents, agent.GetID())
	}

	// Log dispatch completion with comprehensive metrics
	duration := time.Since(start)
	logger.InfoContext(ctx, "Task dispatch completed successfully",
		"duration_ms", duration.Milliseconds(),
		"tasks_dispatched", len(dispatchedTasks),
		"agents_activated", len(dispatchedAgents),
		"total_active_agents", len(d.activeAgents),
		"remaining_available_slots", d.maxAgents-len(d.activeAgents),
		"dispatched_tasks", dispatchedTasks,
		"dispatched_agents", dispatchedAgents,
	)

	// Log performance metrics for monitoring
	logger.Duration("orchestrator.task_dispatch", duration,
		"success", true,
		"tasks_dispatched", len(dispatchedTasks),
		"agents_activated", len(dispatchedAgents),
		"task_list_duration_ms", taskListDuration.Milliseconds(),
		"agent_selection_duration_ms", agentSelectionDuration.Milliseconds(),
	)

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StartAgent starts an agent's execution
func (d *taskDispatcher) StartAgent(ctx context.Context, agentID string) error {
	// Initialize observability for agent startup
	logger := observability.GetLogger(ctx).
		WithComponent("orchestrator").
		WithOperation("StartAgent").
		With("agent_id", agentID)

	logger.InfoContext(ctx, "Starting agent execution",
		"agent_id", agentID,
	)

	d.mu.Lock()
	agent, exists := d.activeAgents[agentID]
	currentActiveCount := len(d.activeAgents)
	d.mu.Unlock()

	logger.DebugContext(ctx, "Agent lookup completed",
		"agent_exists", exists,
		"current_active_agents", currentActiveCount,
	)

	if !exists {
		logger.ErrorContext(ctx, "Agent not found or not active")
		return gerror.New(gerror.ErrCodeAgentNotFound, "agent not found or not active", nil).
			WithComponent("orchestrator").
			WithOperation("StartAgent").
			WithDetails("agent_id", agentID)
	}

	// Start the agent's execution in a goroutine
	go func() {
		// Initialize observability for agent goroutine execution
		goroutineLogger := observability.GetLogger(ctx).
			WithComponent("orchestrator").
			WithOperation("StartAgent.execution").
			With("agent_id", agentID)

		// Start execution timing
		executionStart := time.Now()
		goroutineLogger.InfoContext(ctx, "Agent execution goroutine started")

		// Create a new context with timeout
		execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Add observability context to execution context
		execCtx = observability.WithLogger(execCtx, goroutineLogger)
		execCtx = context.WithValue(execCtx, "agent_id", agentID)

		// Emit agent started event
		eventStart := time.Now()
		d.eventBus.Publish(Event{
			Type:   EventType(EventAgentStarted),
			Source: "dispatcher",
			Data:   map[string]interface{}{"agent_id": agentID},
		})
		eventDuration := time.Since(eventStart)

		goroutineLogger.DebugContext(ctx, "Agent started event published",
			"event_publish_duration_ms", eventDuration.Milliseconds(),
		)

		// Get the task assigned to this agent
		taskLookupStart := time.Now()
		task, hasTask := d.agentTasks[agentID]
		taskRequest := "Execute assigned task"
		var taskTitle, taskDescription string

		if hasTask && task != nil {
			taskTitle = task.Title
			taskDescription = task.Description
			taskRequest = fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
		}
		taskLookupDuration := time.Since(taskLookupStart)

		goroutineLogger.DebugContext(ctx, "Task assignment retrieved",
			"has_task", hasTask,
			"task_title", taskTitle,
			"task_description_length", len(taskDescription),
			"task_request_length", len(taskRequest),
			"task_lookup_duration_ms", taskLookupDuration.Milliseconds(),
		)

		// Execute the agent with the task details
		agentExecStart := time.Now()
		response, err := agent.Execute(execCtx, taskRequest)
		agentExecDuration := time.Since(agentExecStart)

		goroutineLogger.DebugContext(ctx, "Agent execution completed",
			"execution_duration_ms", agentExecDuration.Milliseconds(),
			"response_length", len(response),
			"execution_successful", err == nil,
		)

		// Agent execution completed - cleanup
		cleanupStart := time.Now()
		d.mu.Lock()
		delete(d.activeAgents, agentID)
		delete(d.agentTasks, agentID)
		remainingActiveAgents := len(d.activeAgents)
		d.mu.Unlock()
		cleanupDuration := time.Since(cleanupStart)

		goroutineLogger.DebugContext(ctx, "Agent cleanup completed",
			"cleanup_duration_ms", cleanupDuration.Milliseconds(),
			"remaining_active_agents", remainingActiveAgents,
		)

		// Publish completion/failure events
		finalEventStart := time.Now()
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

			goroutineLogger.WithError(err).ErrorContext(ctx, "Agent execution failed",
				"execution_duration_ms", agentExecDuration.Milliseconds(),
				"remaining_active_agents", remainingActiveAgents,
			)
		} else {
			// Emit agent completed event
			d.eventBus.Publish(Event{
				Type:   EventType(EventAgentCompleted),
				Source: "dispatcher",
				Data:   map[string]interface{}{"agent_id": agentID},
			})

			goroutineLogger.InfoContext(ctx, "Agent execution completed successfully",
				"execution_duration_ms", agentExecDuration.Milliseconds(),
				"response_length", len(response),
				"remaining_active_agents", remainingActiveAgents,
			)
		}
		finalEventDuration := time.Since(finalEventStart)

		// Log comprehensive execution metrics
		totalDuration := time.Since(executionStart)
		goroutineLogger.InfoContext(ctx, "Agent execution session completed",
			"total_duration_ms", totalDuration.Milliseconds(),
			"agent_exec_duration_ms", agentExecDuration.Milliseconds(),
			"cleanup_duration_ms", cleanupDuration.Milliseconds(),
			"final_event_duration_ms", finalEventDuration.Milliseconds(),
			"execution_successful", err == nil,
			"remaining_active_agents", remainingActiveAgents,
		)

		// Log performance metrics for monitoring
		goroutineLogger.Duration("orchestrator.agent_execution", totalDuration,
			"agent_id", agentID,
			"success", err == nil,
			"response_size", len(response),
			"agent_exec_duration_ms", agentExecDuration.Milliseconds(),
			"has_task", hasTask,
		)
	}()

	logger.InfoContext(ctx, "Agent execution goroutine launched successfully")
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
