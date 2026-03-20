// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/lancekrogers/guild-core/pkg/orchestrator/interfaces"
)

// Type definitions moved to types.go to avoid import cycles

// RateLimiter manages API quota usage
type RateLimiter struct {
	provider    string
	maxRequests int
	window      time.Duration
	requests    []time.Time
	mu          sync.Mutex
}

// AgentPool manages the available agents
type AgentPool struct {
	agents    map[string]*AgentInfo
	executors map[string]interfaces.AgentExecutor
	mu        sync.RWMutex
}

// KanbanClient interface moved to types.go

// AgentOrchestrator replaces ResourceManager with Guild-specific orchestration
type AgentOrchestrator struct {
	managerAgent ManagerAgentClient
	agentPool    *AgentPool
	kanbanClient KanbanClient
	rateLimiters map[string]*RateLimiter

	// Assignment tracking
	assignments   map[string]*TaskAssignment
	assignmentsMu sync.RWMutex

	// Metrics
	metrics *OrchestratorMetrics

	// Configuration
	config *OrchestratorConfig
}

// Config and metrics types moved to types.go

// NewAgentOrchestrator creates a new agent orchestrator
func NewAgentOrchestrator(ctx context.Context, config *OrchestratorConfig, managerAgent ManagerAgentClient, kanbanClient KanbanClient) (*AgentOrchestrator, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("NewAgentOrchestrator")
	}

	if config == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "config cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("NewAgentOrchestrator")
	}

	if managerAgent == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "managerAgent cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("NewAgentOrchestrator")
	}

	if kanbanClient == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "kanbanClient cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("NewAgentOrchestrator")
	}

	ao := &AgentOrchestrator{
		managerAgent: managerAgent,
		agentPool: &AgentPool{
			agents:    make(map[string]*AgentInfo),
			executors: make(map[string]interfaces.AgentExecutor),
		},
		kanbanClient: kanbanClient,
		rateLimiters: make(map[string]*RateLimiter),
		assignments:  make(map[string]*TaskAssignment),
		metrics: &OrchestratorMetrics{
			AgentUtilization: make(map[string]float64),
		},
		config: config,
	}

	// Initialize rate limiters
	for _, rlConfig := range config.RateLimitConfigs {
		ao.rateLimiters[rlConfig.Provider] = &RateLimiter{
			provider:    rlConfig.Provider,
			maxRequests: rlConfig.MaxRequests,
			window:      rlConfig.Window,
			requests:    make([]time.Time, 0, rlConfig.MaxRequests),
		}
	}

	return ao, nil
}

// RegisterAgent adds an agent to the pool
func (ao *AgentOrchestrator) RegisterAgent(ctx context.Context, agentID string, executor interfaces.AgentExecutor, capabilities []AgentCapability) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("RegisterAgent")
	}

	if agentID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agentID cannot be empty", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RegisterAgent")
	}

	if executor == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "executor cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RegisterAgent")
	}

	ao.agentPool.mu.Lock()
	defer ao.agentPool.mu.Unlock()

	ao.agentPool.agents[agentID] = &AgentInfo{
		AgentID:      agentID,
		Capabilities: capabilities,
		IsAvailable:  true,
		TasksHandled: 0,
		ErrorRate:    0.0,
	}
	ao.agentPool.executors[agentID] = executor

	return nil
}

// UnregisterAgent removes an agent from the pool
func (ao *AgentOrchestrator) UnregisterAgent(ctx context.Context, agentID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("UnregisterAgent")
	}

	ao.agentPool.mu.Lock()
	defer ao.agentPool.mu.Unlock()

	if _, exists := ao.agentPool.agents[agentID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "agent not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("UnregisterAgent").
			WithDetails("agent_id", agentID)
	}

	delete(ao.agentPool.agents, agentID)
	delete(ao.agentPool.executors, agentID)

	return nil
}

// RequestTaskAssignment requests the manager agent to assign a task
func (ao *AgentOrchestrator) RequestTaskAssignment(ctx context.Context, task *kanban.Task) (*TaskAssignment, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("RequestTaskAssignment")
	}

	if task == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "task cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RequestTaskAssignment")
	}

	// Get available agents
	availableAgents := ao.getAvailableAgents()
	if len(availableAgents) == 0 {
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "no available agents", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RequestTaskAssignment").
			WithDetails("task_id", task.ID)
	}

	// Ask manager agent to make assignment decision
	ctx, cancel := context.WithTimeout(ctx, ao.config.ManagerAgentTimeout)
	defer cancel()

	assignment, err := ao.managerAgent.RequestAssignment(ctx, task, availableAgents)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "manager agent assignment failed").
			WithComponent("orchestrator.scheduler").
			WithOperation("RequestTaskAssignment").
			WithDetails("task_id", task.ID)
	}

	// Validate assignment
	if assignment.AgentID == "" {
		return nil, gerror.New(gerror.ErrCodeInternal, "manager returned empty agent assignment", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RequestTaskAssignment").
			WithDetails("task_id", task.ID)
	}

	// Check rate limits if API provider specified
	if assignment.APIProvider != "" {
		if !ao.checkRateLimit(assignment.APIProvider) {
			return nil, gerror.New(gerror.ErrCodeResourceExhausted, "API rate limit exceeded", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("RequestTaskAssignment").
				WithDetails("provider", assignment.APIProvider)
		}
	}

	// Store assignment
	ao.assignmentsMu.Lock()
	ao.assignments[task.ID] = assignment
	ao.assignmentsMu.Unlock()

	// Update metrics
	ao.metrics.mu.Lock()
	ao.metrics.TasksAssigned++
	ao.metrics.mu.Unlock()

	return assignment, nil
}

// AssignTask atomically assigns a task to an agent
func (ao *AgentOrchestrator) AssignTask(ctx context.Context, assignment *TaskAssignment) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("AssignTask")
	}

	if assignment == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "assignment cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("AssignTask")
	}

	// Use database transaction for atomic assignment
	return ao.kanbanClient.WithTransaction(ctx, func(txCtx context.Context) error {
		// Get task with lock
		task, err := ao.kanbanClient.GetTaskForUpdate(txCtx, assignment.TaskID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get task for update").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTask").
				WithDetails("task_id", assignment.TaskID)
		}

		// Check if already assigned
		if task.AssignedTo != "" {
			return gerror.New(gerror.ErrCodeConflict, "task already assigned", nil).
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTask").
				WithDetails("task_id", assignment.TaskID).
				WithDetails("existing_assignee", task.AssignedTo)
		}

		// Atomically assign task
		if err := ao.kanbanClient.AssignTaskAtomic(txCtx, assignment.TaskID, assignment.AgentID); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to assign task atomically").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTask").
				WithDetails("task_id", assignment.TaskID).
				WithDetails("agent_id", assignment.AgentID)
		}

		// Update agent availability
		ao.agentPool.mu.Lock()
		if agentInfo, exists := ao.agentPool.agents[assignment.AgentID]; exists {
			agentInfo.IsAvailable = false
			agentInfo.CurrentTask = assignment.TaskID
			agentInfo.LastAssigned = time.Now()
		}
		ao.agentPool.mu.Unlock()

		// Update task status to in_progress
		if err := ao.kanbanClient.UpdateTaskStatusAtomic(txCtx, assignment.TaskID, kanban.StatusInProgress); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update task status").
				WithComponent("orchestrator.scheduler").
				WithOperation("AssignTask").
				WithDetails("task_id", assignment.TaskID)
		}

		return nil
	})
}

// ReleaseAgent marks an agent as available after task completion
func (ao *AgentOrchestrator) ReleaseAgent(ctx context.Context, agentID string, taskCompleted bool) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("ReleaseAgent")
	}

	ao.agentPool.mu.Lock()
	defer ao.agentPool.mu.Unlock()

	agentInfo, exists := ao.agentPool.agents[agentID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "agent not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("ReleaseAgent").
			WithDetails("agent_id", agentID)
	}

	// Update agent info
	agentInfo.IsAvailable = true
	agentInfo.CurrentTask = ""
	agentInfo.TasksHandled++

	// Update error rate if task failed
	if !taskCompleted {
		totalTasks := float64(agentInfo.TasksHandled)
		currentErrors := agentInfo.ErrorRate * (totalTasks - 1)
		agentInfo.ErrorRate = (currentErrors + 1) / totalTasks
	}

	// Update metrics
	ao.metrics.mu.Lock()
	if taskCompleted {
		ao.metrics.TasksCompleted++
	} else {
		ao.metrics.TasksFailed++
	}
	ao.metrics.mu.Unlock()

	return nil
}

// getAvailableAgents returns agents that can accept tasks
func (ao *AgentOrchestrator) getAvailableAgents() []*AgentInfo {
	ao.agentPool.mu.RLock()
	defer ao.agentPool.mu.RUnlock()

	var available []*AgentInfo
	for _, agent := range ao.agentPool.agents {
		if agent.IsAvailable && agent.ErrorRate < 0.5 { // Skip agents with >50% error rate
			// Create a copy to avoid race conditions
			agentCopy := &AgentInfo{
				AgentID:      agent.AgentID,
				Capabilities: append([]AgentCapability{}, agent.Capabilities...),
				IsAvailable:  agent.IsAvailable,
				TasksHandled: agent.TasksHandled,
				ErrorRate:    agent.ErrorRate,
				LastAssigned: agent.LastAssigned,
			}
			available = append(available, agentCopy)
		}
	}

	return available
}

// checkRateLimit checks if we can make a request to the given provider
func (ao *AgentOrchestrator) checkRateLimit(provider string) bool {
	limiter, exists := ao.rateLimiters[provider]
	if !exists {
		return true // No limit configured
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-limiter.window)

	// Remove old requests
	validRequests := make([]time.Time, 0, len(limiter.requests))
	for _, reqTime := range limiter.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	limiter.requests = validRequests

	// Check if we can add another request
	if len(limiter.requests) < limiter.maxRequests {
		limiter.requests = append(limiter.requests, now)
		return true
	}

	return false
}

// GetMetrics returns current orchestrator metrics
func (ao *AgentOrchestrator) GetMetrics() OrchestratorMetrics {
	ao.metrics.mu.RLock()
	defer ao.metrics.mu.RUnlock()

	// Calculate agent utilization
	ao.agentPool.mu.RLock()
	utilization := make(map[string]float64)
	for agentID, agent := range ao.agentPool.agents {
		if agent.TasksHandled > 0 {
			utilization[agentID] = 1.0 - agent.ErrorRate
		}
	}
	ao.agentPool.mu.RUnlock()

	return OrchestratorMetrics{
		TasksAssigned:    ao.metrics.TasksAssigned,
		TasksCompleted:   ao.metrics.TasksCompleted,
		TasksFailed:      ao.metrics.TasksFailed,
		AssignmentTime:   ao.metrics.AssignmentTime,
		AverageWaitTime:  ao.metrics.AverageWaitTime,
		AgentUtilization: utilization,
	}
}

// GetAgentExecutor returns the executor for an agent
func (ao *AgentOrchestrator) GetAgentExecutor(agentID string) (interfaces.AgentExecutor, error) {
	ao.agentPool.mu.RLock()
	defer ao.agentPool.mu.RUnlock()

	executor, exists := ao.agentPool.executors[agentID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "agent executor not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("GetAgentExecutor").
			WithDetails("agent_id", agentID)
	}

	return executor, nil
}

// GetAssignment retrieves a task assignment
func (ao *AgentOrchestrator) GetAssignment(taskID string) (*TaskAssignment, error) {
	ao.assignmentsMu.RLock()
	defer ao.assignmentsMu.RUnlock()

	assignment, exists := ao.assignments[taskID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "assignment not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("GetAssignment").
			WithDetails("task_id", taskID)
	}

	return assignment, nil
}
