// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/orchestrator/interfaces"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusReady     TaskStatus = "ready"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ResourceRequirements defines what resources a task needs
type ResourceRequirements struct {
	CPUCores    float64           `json:"cpu_cores"`
	MemoryMB    int64             `json:"memory_mb"`
	GPURequired bool              `json:"gpu_required"`
	APIQuotas   map[string]int    `json:"api_quotas"` // provider -> requests/min
	Custom      map[string]interface{} `json:"custom"`
}

// SchedulableTask represents a task that can be scheduled
type SchedulableTask struct {
	ID           string
	CommissionID string
	Priority     int
	Dependencies []string
	Resources    ResourceRequirements
	Estimated    time.Duration
	Agent        string
	Payload      interface{}
}

// RunningTask tracks a currently executing task
type RunningTask struct {
	Task      *SchedulableTask
	Executor  interfaces.AgentExecutor
	StartTime time.Time
	Context   context.Context
	Cancel    context.CancelFunc
	Progress  chan ProgressUpdate
}

// ProgressUpdate represents progress information from a running task
type ProgressUpdate struct {
	TaskID      string
	Percentage  float64
	Message     string
	Details     map[string]interface{}
	Timestamp   time.Time
}

// TaskResult represents the outcome of a task execution
type TaskResult struct {
	TaskID    string
	Status    TaskStatus
	Output    interface{}
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Metrics   map[string]interface{}
}

// TaskScheduler manages concurrent task execution with dependencies and resource allocation
type TaskScheduler struct {
	executors    map[string]interfaces.AgentExecutor
	taskQueue    *PriorityQueue
	runningTasks map[string]*RunningTask
	dependencies *DependencyGraph
	resources    *ResourceManager
	progress     *ProgressAggregator
	config       *SchedulerConfig
	
	// Progress tracking
	progressChan chan ProgressUpdate
	resultChan   chan TaskResult
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// SchedulerConfig contains configuration for the task scheduler
type SchedulerConfig struct {
	MaxConcurrentTasks int
	ScheduleInterval   time.Duration
	DefaultTimeout     time.Duration
	EnableMetrics      bool
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		MaxConcurrentTasks: 10,
		ScheduleInterval:   100 * time.Millisecond,
		DefaultTimeout:     30 * time.Minute,
		EnableMetrics:      true,
	}
}

// NewTaskScheduler creates a new task scheduler
func NewTaskScheduler(ctx context.Context, config *SchedulerConfig) (*TaskScheduler, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("NewTaskScheduler")
	}

	if config == nil {
		config = DefaultSchedulerConfig()
	}

	schedCtx, cancel := context.WithCancel(ctx)

	scheduler := &TaskScheduler{
		executors:    make(map[string]interfaces.AgentExecutor),
		taskQueue:    NewPriorityQueue(),
		runningTasks: make(map[string]*RunningTask),
		dependencies: NewDependencyGraph(),
		resources:    NewResourceManager(config.MaxConcurrentTasks),
		progress:     NewProgressAggregator(),
		config:       config,
		progressChan: make(chan ProgressUpdate, 1000),
		resultChan:   make(chan TaskResult, 100),
		ctx:          schedCtx,
		cancel:       cancel,
	}

	return scheduler, nil
}

// RegisterExecutor adds an agent executor to the scheduler
func (ts *TaskScheduler) RegisterExecutor(agentID string, executor interfaces.AgentExecutor) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if executor == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "executor cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("RegisterExecutor")
	}

	ts.executors[agentID] = executor
	return nil
}

// SubmitTask adds a task to the scheduler queue
func (ts *TaskScheduler) SubmitTask(ctx context.Context, task *SchedulableTask) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("SubmitTask")
	}

	if task == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "task cannot be nil", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("SubmitTask")
	}

	// Add to dependency graph
	if err := ts.dependencies.AddTask(task.ID, task.Dependencies); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add task to dependency graph").
			WithComponent("orchestrator.scheduler").
			WithOperation("SubmitTask").
			WithDetails("task_id", task.ID)
	}

	// Add to priority queue
	ts.taskQueue.Push(task)

	// Update progress tracking
	ts.progress.UpdateTaskStatus(task.ID, task.CommissionID, TaskStatusPending)

	return nil
}

// Start begins the scheduler's execution loops
func (ts *TaskScheduler) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("Start")
	}

	// Start scheduler loop
	ts.wg.Add(1)
	go ts.schedulerLoop()

	// Start progress aggregator
	ts.wg.Add(1)
	go ts.progressAggregator()

	// Start resource monitor
	ts.wg.Add(1)
	go ts.resourceMonitor()

	return nil
}

// Stop gracefully shuts down the scheduler
func (ts *TaskScheduler) Stop(ctx context.Context) error {
	// Cancel all running tasks
	ts.mu.Lock()
	for _, rt := range ts.runningTasks {
		rt.Cancel()
	}
	ts.mu.Unlock()

	// Cancel scheduler context
	ts.cancel()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		ts.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout waiting for scheduler shutdown").
			WithComponent("orchestrator.scheduler").
			WithOperation("Stop")
	}
}

// schedulerLoop is the main scheduling loop
func (ts *TaskScheduler) schedulerLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(ts.config.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return

		case <-ticker.C:
			ts.scheduleNext()
		}
	}
}

// scheduleNext attempts to schedule ready tasks
func (ts *TaskScheduler) scheduleNext() {
	// Check context cancellation
	if err := ts.ctx.Err(); err != nil {
		return
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check if we can run more tasks
	if len(ts.runningTasks) >= ts.config.MaxConcurrentTasks {
		return
	}

	// Get available executors
	available := ts.getAvailableExecutors()
	if len(available) == 0 {
		return
	}

	// Get ready tasks
	readyTasks := ts.getReadyTasks()
	if len(readyTasks) == 0 {
		return
	}

	// Match tasks to executors
	assignments := ts.matchTasksToExecutors(readyTasks, available)

	// Start executions
	for task, executor := range assignments {
		ts.startTaskExecution(task, executor)
	}
}

// getAvailableExecutors returns executors that can accept tasks
func (ts *TaskScheduler) getAvailableExecutors() []interfaces.AgentExecutor {
	var available []interfaces.AgentExecutor

	for agentID, executor := range ts.executors {
		// Check if executor is already running a task
		busy := false
		for _, rt := range ts.runningTasks {
			if rt.Task.Agent == agentID {
				busy = true
				break
			}
		}

		if !busy {
			available = append(available, executor)
		}
	}

	return available
}

// getReadyTasks returns tasks that are ready to execute
func (ts *TaskScheduler) getReadyTasks() []*SchedulableTask {
	var ready []*SchedulableTask

	// Check each queued task
	ts.taskQueue.Range(func(task *SchedulableTask) bool {
		// Check dependencies
		if !ts.dependencies.AreSatisfied(task.ID) {
			return true // continue
		}

		// Check resources
		if !ts.resources.CanAllocate(ts.ctx, task.Resources) {
			return true // continue
		}

		ready = append(ready, task)
		return true
	})

	// Sort by priority (highest first)
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].Priority > ready[j].Priority
	})

	return ready
}

// matchTasksToExecutors assigns tasks to available executors
func (ts *TaskScheduler) matchTasksToExecutors(tasks []*SchedulableTask, executors []interfaces.AgentExecutor) map[*SchedulableTask]interfaces.AgentExecutor {
	assignments := make(map[*SchedulableTask]interfaces.AgentExecutor)

	// Simple assignment for now - can be made more sophisticated
	for i, task := range tasks {
		if i < len(executors) {
			// Find executor for the agent
			for _, executor := range executors {
				// TODO: Match based on agent capabilities
				assignments[task] = executor
				break
			}
		}
	}

	return assignments
}

// startTaskExecution begins executing a task
func (ts *TaskScheduler) startTaskExecution(task *SchedulableTask, executor interfaces.AgentExecutor) {
	// Remove from queue
	ts.taskQueue.Remove(task.ID)

	// Allocate resources
	if err := ts.resources.Allocate(ts.ctx, task.ID, task.Resources); err != nil {
		ts.handleTaskError(task, err)
		return
	}

	// Create context with timeout
	timeout := task.Estimated * 2
	if timeout == 0 {
		timeout = ts.config.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ts.ctx, timeout)

	// Track running task
	running := &RunningTask{
		Task:      task,
		Executor:  executor,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
		Progress:  make(chan ProgressUpdate, 100),
	}

	ts.runningTasks[task.ID] = running

	// Update progress tracking
	ts.progress.UpdateTaskStatus(task.ID, task.CommissionID, TaskStatusRunning)

	// Start execution in goroutine
	ts.wg.Add(1)
	go ts.executeTask(running)
}

// executeTask runs a task to completion
func (ts *TaskScheduler) executeTask(rt *RunningTask) {
	defer ts.wg.Done()
	
	// Check context at start
	if err := rt.Context.Err(); err != nil {
		return
	}
	
	defer func() {
		// Cleanup - safely handle nil checks
		if rt != nil && rt.Task != nil {
			ts.mu.Lock()
			delete(ts.runningTasks, rt.Task.ID)
			_ = ts.resources.Release(ts.ctx, rt.Task.ID) // Ignore error during cleanup
			ts.mu.Unlock()
		}

		if rt != nil && rt.Cancel != nil {
			rt.Cancel()
		}
		
		// Safe channel close
		if rt != nil && rt.Progress != nil {
			select {
			case <-rt.Progress:
				// Already closed
			default:
				close(rt.Progress)
			}
		}
	}()

	// Forward progress updates
	go func() {
		for update := range rt.Progress {
			select {
			case ts.progressChan <- update:
			case <-rt.Context.Done():
				return
			}
		}
	}()

	// Execute the task
	startTime := time.Now()
	result, err := rt.Executor.Execute(rt.Context, rt.Task.ID, rt.Task.Payload)
	endTime := time.Now()

	// Update dependency graph
	if err == nil {
		ts.dependencies.MarkComplete(rt.Task.ID)
	}

	// Create task result
	taskResult := TaskResult{
		TaskID:    rt.Task.ID,
		StartTime: startTime,
		EndTime:   endTime,
		Output:    result,
		Error:     err,
	}

	if err != nil {
		taskResult.Status = TaskStatusFailed
		ts.progress.UpdateTaskStatus(rt.Task.ID, rt.Task.CommissionID, TaskStatusFailed)
		ts.handleTaskError(rt.Task, err)
	} else {
		taskResult.Status = TaskStatusCompleted
		ts.progress.UpdateTaskStatus(rt.Task.ID, rt.Task.CommissionID, TaskStatusCompleted)
		ts.handleTaskSuccess(rt.Task, result)
	}

	// Send result
	select {
	case ts.resultChan <- taskResult:
	case <-ts.ctx.Done():
	}
}

// handleTaskError processes task failures
func (ts *TaskScheduler) handleTaskError(task *SchedulableTask, err error) {
	// Wrap error with context
	wrappedErr := gerror.Wrap(err, gerror.ErrCodeInternal, "task execution failed").
		WithComponent("orchestrator.scheduler").
		WithOperation("handleTaskError").
		WithDetails("task_id", task.ID).
		WithDetails("commission_id", task.CommissionID)
	
	// Update progress tracking
	ts.progress.UpdateTaskStatus(task.ID, task.CommissionID, TaskStatusFailed)
	
	// Mark dependencies as blocked
	ts.dependencies.MarkFailed(task.ID)
	
	// TODO: Implement retry logic based on error type
	// TODO: Notify orchestrator of failure for human intervention
	_ = wrappedErr // Use the error for logging when observability is ready
}

// handleTaskSuccess processes successful task completion  
func (ts *TaskScheduler) handleTaskSuccess(task *SchedulableTask, result interface{}) {
	// Store result for dependent tasks
	ts.mu.Lock()
	// TODO: Implement result storage when memory layer is ready
	ts.mu.Unlock()
	
	// Re-evaluate ready tasks now that dependencies may be satisfied
	go ts.scheduleNext()
}

// progressAggregator collects and processes progress updates
func (ts *TaskScheduler) progressAggregator() {
	defer ts.wg.Done()

	for {
		select {
		case <-ts.ctx.Done():
			return

		case update := <-ts.progressChan:
			// Update progress aggregator
			if err := ts.progress.UpdateTaskProgress(update); err != nil {
				// Wrap and log error but continue processing
				wrappedErr := gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update progress").
					WithComponent("orchestrator.scheduler").
					WithOperation("progressAggregator").
					WithDetails("task_id", update.TaskID)
				_ = wrappedErr // TODO: Send to observability when ready
				continue
			}
		}
	}
}

// resourceMonitor tracks resource usage
func (ts *TaskScheduler) resourceMonitor() {
	defer ts.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return

		case <-ticker.C:
			// Monitor resource usage
			// TODO: Implement resource monitoring
		}
	}
}

// GetProgress returns the progress channel
func (ts *TaskScheduler) GetProgress() <-chan ProgressUpdate {
	return ts.progressChan
}

// GetResults returns the results channel
func (ts *TaskScheduler) GetResults() <-chan TaskResult {
	return ts.resultChan
}

// GetRunningTasks returns currently executing tasks
func (ts *TaskScheduler) GetRunningTasks() []*SchedulableTask {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	tasks := make([]*SchedulableTask, 0, len(ts.runningTasks))
	for _, rt := range ts.runningTasks {
		tasks = append(tasks, rt.Task)
	}

	return tasks
}

// GetQueuedTasks returns tasks waiting to be scheduled
func (ts *TaskScheduler) GetQueuedTasks() []*SchedulableTask {
	var tasks []*SchedulableTask
	ts.taskQueue.Range(func(task *SchedulableTask) bool {
		tasks = append(tasks, task)
		return true
	})
	return tasks
}

// GetTaskStatus returns the current status of a task
func (ts *TaskScheduler) GetTaskStatus(taskID string) (TaskStatus, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Check if running
	if _, ok := ts.runningTasks[taskID]; ok {
		return TaskStatusRunning, nil
	}

	// Check if completed
	if ts.dependencies.IsCompleted(taskID) {
		return TaskStatusCompleted, nil
	}

	// Check if queued
	if ts.taskQueue.Contains(taskID) {
		if ts.dependencies.AreSatisfied(taskID) {
			return TaskStatusReady, nil
		}
		return TaskStatusPending, nil
	}

	return "", gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
		WithComponent("orchestrator.scheduler").
		WithOperation("GetTaskStatus").
		WithDetails("task_id", taskID)
}

// GetSchedulerStats returns scheduler statistics
func (ts *TaskScheduler) GetSchedulerStats() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return map[string]interface{}{
		"running_tasks":   len(ts.runningTasks),
		"queued_tasks":    ts.taskQueue.Len(),
		"total_executors": len(ts.executors),
		"resource_usage":  ts.resources.GetUsage(),
	}
}

// RegisterCommission registers a new commission with the scheduler
func (ts *TaskScheduler) RegisterCommission(ctx context.Context, commissionID string, totalTasks int) error {
	return ts.progress.RegisterCommission(ctx, commissionID, totalTasks)
}

// GetCommissionProgress returns progress for a specific commission
func (ts *TaskScheduler) GetCommissionProgress(commissionID string) (*CommissionProgress, error) {
	return ts.progress.GetCommissionProgress(commissionID)
}

// GetProgressSnapshot returns current progress snapshot
func (ts *TaskScheduler) GetProgressSnapshot() ProgressSnapshot {
	return ts.progress.GetProgressSnapshot()
}

// SubscribeToProgress subscribes to progress updates
func (ts *TaskScheduler) SubscribeToProgress(ctx context.Context) <-chan ProgressSnapshot {
	return ts.progress.Subscribe(ctx)
}