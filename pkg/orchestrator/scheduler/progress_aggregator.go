// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ResourceUsage is a placeholder for metrics - replaced by agent metrics
type ResourceUsage struct {
	AgentsAvailable int                    `json:"agents_available"`
	TasksRunning    int                    `json:"tasks_running"`
	AgentMetrics    map[string]interface{} `json:"agent_metrics"`
}

// CommissionProgress tracks overall progress for a commission
type CommissionProgress struct {
	CommissionID    string
	TotalTasks      int
	CompletedTasks  int
	FailedTasks     int
	RunningTasks    int
	PendingTasks    int
	OverallProgress float64
	StartTime       *time.Time
	EstimatedEnd    *time.Time
	TaskProgress    map[string]*TaskProgress
	mu              sync.RWMutex
}

// TaskProgress tracks progress for an individual task
type TaskProgress struct {
	TaskID        string
	Status        TaskStatus
	Percentage    float64
	Message       string
	StartTime     *time.Time
	LastUpdate    time.Time
	EstimatedEnd  *time.Time
	SubTasks      map[string]*SubTaskProgress
	ResourceUsage *ResourceUsage
}

// SubTaskProgress tracks progress for sub-tasks
type SubTaskProgress struct {
	ID         string
	Name       string
	Percentage float64
	Status     string
	Message    string
}

// ProgressAggregator collects and aggregates progress from multiple agents
type ProgressAggregator struct {
	commissions map[string]*CommissionProgress
	tasks       map[string]*TaskProgress
	subscribers []chan<- ProgressSnapshot
	mu          sync.RWMutex
}

// ProgressSnapshot represents a point-in-time view of progress
type ProgressSnapshot struct {
	Timestamp    time.Time
	Commissions  map[string]CommissionSummary
	RunningTasks []TaskSummary
	Metrics      ProgressMetrics
}

// CommissionSummary summarizes progress for a commission
type CommissionSummary struct {
	CommissionID       string
	Title              string
	OverallProgress    float64
	TaskCounts         TaskCounts
	Duration           time.Duration
	EstimatedRemaining time.Duration
}

// TaskCounts tracks task counts by status
type TaskCounts struct {
	Total     int
	Pending   int
	Running   int
	Completed int
	Failed    int
}

// TaskSummary summarizes a running task
type TaskSummary struct {
	TaskID        string
	Agent         string
	Progress      float64
	Message       string
	Duration      time.Duration
	ResourceUsage ResourceUsage
}

// ProgressMetrics contains aggregate metrics
type ProgressMetrics struct {
	TotalTasksProcessed int
	AverageTaskDuration time.Duration
	TasksPerMinute      float64
	SuccessRate         float64
	ResourceUtilization float64
}

// NewProgressAggregator creates a new progress aggregator
func NewProgressAggregator() *ProgressAggregator {
	return &ProgressAggregator{
		commissions: make(map[string]*CommissionProgress),
		tasks:       make(map[string]*TaskProgress),
		subscribers: make([]chan<- ProgressSnapshot, 0),
	}
}

// RegisterCommission registers a new commission for tracking
func (pa *ProgressAggregator) RegisterCommission(ctx context.Context, commissionID string, totalTasks int) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("orchestrator.scheduler").
			WithOperation("RegisterCommission")
	}
	pa.mu.Lock()
	defer pa.mu.Unlock()

	now := time.Now()
	pa.commissions[commissionID] = &CommissionProgress{
		CommissionID: commissionID,
		TotalTasks:   totalTasks,
		PendingTasks: totalTasks,
		StartTime:    &now,
		TaskProgress: make(map[string]*TaskProgress),
	}
	return nil
}

// UpdateTaskProgress updates progress for a specific task
func (pa *ProgressAggregator) UpdateTaskProgress(update ProgressUpdate) error {
	// Progress updates are time-sensitive, so we don't check context to avoid dropping updates
	pa.mu.Lock()

	// Update task progress
	task, exists := pa.tasks[update.TaskID]
	if !exists {
		task = &TaskProgress{
			TaskID:   update.TaskID,
			SubTasks: make(map[string]*SubTaskProgress),
		}
		pa.tasks[update.TaskID] = task
	}

	task.Percentage = update.Percentage
	task.Message = update.Message
	task.LastUpdate = update.Timestamp

	// Update sub-tasks if provided
	if subTasks, ok := update.Details["subtasks"].([]SubTaskProgress); ok {
		for _, st := range subTasks {
			task.SubTasks[st.ID] = &st
		}
	}

	pa.mu.Unlock()

	// Notify subscribers after releasing lock to avoid deadlock
	pa.notifySubscribers()

	return nil
}

// UpdateTaskStatus updates the status of a task
func (pa *ProgressAggregator) UpdateTaskStatus(taskID, commissionID string, status TaskStatus) {
	// Status updates are critical for correctness, so we don't check context

	// First update task status
	pa.mu.Lock()
	task, exists := pa.tasks[taskID]
	var previousStatus TaskStatus
	var isNewTask bool
	if !exists {
		task = &TaskProgress{
			TaskID:   taskID,
			Status:   TaskStatusPending, // New tasks start as pending
			SubTasks: make(map[string]*SubTaskProgress),
		}
		pa.tasks[taskID] = task
		previousStatus = TaskStatusPending
		isNewTask = true
	} else {
		previousStatus = task.Status
		isNewTask = false
	}

	task.Status = status
	task.LastUpdate = time.Now()

	// Update start time
	if status == TaskStatusRunning && task.StartTime == nil {
		now := time.Now()
		task.StartTime = &now
	}

	// Get commission reference while holding lock
	commission := pa.commissions[commissionID]
	pa.mu.Unlock()

	// Update commission progress without holding pa.mu
	if commission != nil {
		commission.mu.Lock()

		// For new tasks, we "consume" one pending task from the commission pool
		if isNewTask {
			// New task consumes one pending task slot
			if commission.PendingTasks > 0 {
				commission.PendingTasks--
			}
		} else {
			// Update task counts based on status change for existing tasks
			switch previousStatus {
			case TaskStatusPending:
				commission.PendingTasks--
			case TaskStatusRunning:
				commission.RunningTasks--
			case TaskStatusCompleted:
				commission.CompletedTasks--
			case TaskStatusFailed:
				commission.FailedTasks--
			}
		}

		// Always update the new status count
		switch status {
		case TaskStatusPending:
			commission.PendingTasks++
		case TaskStatusRunning:
			commission.RunningTasks++
		case TaskStatusCompleted:
			commission.CompletedTasks++
		case TaskStatusFailed:
			commission.FailedTasks++
		}

		// Update overall progress
		if commission.TotalTasks > 0 {
			commission.OverallProgress = float64(commission.CompletedTasks) / float64(commission.TotalTasks) * 100
		}

		// Update estimated end time
		pa.updateEstimatedEnd(commission)

		commission.mu.Unlock()
	}

	// Notify subscribers
	pa.notifySubscribers()
}

// GetProgressSnapshot returns current progress snapshot
func (pa *ProgressAggregator) GetProgressSnapshot() ProgressSnapshot {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	snapshot := ProgressSnapshot{
		Timestamp:    time.Now(),
		Commissions:  make(map[string]CommissionSummary),
		RunningTasks: make([]TaskSummary, 0),
		Metrics:      pa.calculateMetrics(),
	}

	// Build commission summaries
	for id, commission := range pa.commissions {
		commission.mu.RLock()

		summary := CommissionSummary{
			CommissionID:    id,
			OverallProgress: commission.OverallProgress,
			TaskCounts: TaskCounts{
				Total:     commission.TotalTasks,
				Pending:   commission.PendingTasks,
				Running:   commission.RunningTasks,
				Completed: commission.CompletedTasks,
				Failed:    commission.FailedTasks,
			},
		}

		if commission.StartTime != nil {
			summary.Duration = time.Since(*commission.StartTime)
		}

		if commission.EstimatedEnd != nil && commission.EstimatedEnd.After(time.Now()) {
			summary.EstimatedRemaining = commission.EstimatedEnd.Sub(time.Now())
		}

		commission.mu.RUnlock()
		snapshot.Commissions[id] = summary
	}

	// Build running task summaries
	for _, task := range pa.tasks {
		if task.Status == TaskStatusRunning {
			summary := TaskSummary{
				TaskID:   task.TaskID,
				Progress: task.Percentage,
				Message:  task.Message,
			}

			if task.StartTime != nil {
				summary.Duration = time.Since(*task.StartTime)
			}

			if task.ResourceUsage != nil {
				summary.ResourceUsage = *task.ResourceUsage
			}

			snapshot.RunningTasks = append(snapshot.RunningTasks, summary)
		}
	}

	return snapshot
}

// Subscribe adds a subscriber for progress updates
func (pa *ProgressAggregator) Subscribe(ctx context.Context) <-chan ProgressSnapshot {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ch := make(chan ProgressSnapshot, 10)
	pa.subscribers = append(pa.subscribers, ch)

	// Send initial snapshot
	go func() {
		snapshot := pa.GetProgressSnapshot()
		select {
		case ch <- snapshot:
		case <-ctx.Done():
		}
	}()

	// Remove subscriber when context is done
	go func() {
		<-ctx.Done()
		pa.mu.Lock()
		defer pa.mu.Unlock()

		// Remove from subscribers
		for i, sub := range pa.subscribers {
			if sub == ch {
				pa.subscribers = append(pa.subscribers[:i], pa.subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}()

	return ch
}

// GetCommissionProgress returns progress for a specific commission
func (pa *ProgressAggregator) GetCommissionProgress(commissionID string) (*CommissionProgress, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	commission, exists := pa.commissions[commissionID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "commission not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("GetCommissionProgress").
			WithDetails("commission_id", commissionID)
	}

	// Return a copy to avoid race conditions
	commission.mu.RLock()
	defer commission.mu.RUnlock()

	result := &CommissionProgress{
		CommissionID:    commission.CommissionID,
		TotalTasks:      commission.TotalTasks,
		CompletedTasks:  commission.CompletedTasks,
		FailedTasks:     commission.FailedTasks,
		RunningTasks:    commission.RunningTasks,
		PendingTasks:    commission.PendingTasks,
		OverallProgress: commission.OverallProgress,
		StartTime:       commission.StartTime,
		EstimatedEnd:    commission.EstimatedEnd,
		TaskProgress:    make(map[string]*TaskProgress),
	}

	// Copy task progress
	for k, v := range commission.TaskProgress {
		result.TaskProgress[k] = v
	}

	return result, nil
}

// updateEstimatedEnd calculates estimated completion time
func (pa *ProgressAggregator) updateEstimatedEnd(commission *CommissionProgress) {
	if commission.CompletedTasks == 0 || commission.StartTime == nil {
		return
	}

	// Calculate average task duration
	elapsed := time.Since(*commission.StartTime)
	avgTaskDuration := elapsed / time.Duration(commission.CompletedTasks)

	// Estimate remaining time
	remainingTasks := commission.TotalTasks - commission.CompletedTasks - commission.FailedTasks
	if remainingTasks > 0 {
		estimatedRemaining := avgTaskDuration * time.Duration(remainingTasks)
		estimatedEnd := time.Now().Add(estimatedRemaining)
		commission.EstimatedEnd = &estimatedEnd
	}
}

// calculateMetrics calculates aggregate metrics
func (pa *ProgressAggregator) calculateMetrics() ProgressMetrics {
	metrics := ProgressMetrics{}

	totalCompleted := 0
	totalFailed := 0
	totalDuration := time.Duration(0)
	taskCount := 0

	for _, commission := range pa.commissions {
		commission.mu.RLock()
		totalCompleted += commission.CompletedTasks
		totalFailed += commission.FailedTasks
		commission.mu.RUnlock()
	}

	for _, task := range pa.tasks {
		if task.Status == TaskStatusCompleted && task.StartTime != nil {
			duration := task.LastUpdate.Sub(*task.StartTime)
			totalDuration += duration
			taskCount++
		}
	}

	metrics.TotalTasksProcessed = totalCompleted + totalFailed

	if taskCount > 0 {
		metrics.AverageTaskDuration = totalDuration / time.Duration(taskCount)
	}

	if totalCompleted+totalFailed > 0 {
		metrics.SuccessRate = float64(totalCompleted) / float64(totalCompleted+totalFailed) * 100
	}

	// Calculate tasks per minute
	oldestStart := time.Now()
	for _, commission := range pa.commissions {
		if commission.StartTime != nil && commission.StartTime.Before(oldestStart) {
			oldestStart = *commission.StartTime
		}
	}

	if elapsed := time.Since(oldestStart); elapsed > 0 {
		metrics.TasksPerMinute = float64(metrics.TotalTasksProcessed) / elapsed.Minutes()
	}

	return metrics
}

// notifySubscribers sends progress updates to all subscribers
func (pa *ProgressAggregator) notifySubscribers() {
	snapshot := pa.GetProgressSnapshot()

	for _, ch := range pa.subscribers {
		select {
		case ch <- snapshot:
		default:
			// Skip if channel is full
		}
	}
}

// Clear removes all progress data
func (pa *ProgressAggregator) Clear() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.commissions = make(map[string]*CommissionProgress)
	pa.tasks = make(map[string]*TaskProgress)
}

// GetTaskProgress returns progress for a specific task
func (pa *ProgressAggregator) GetTaskProgress(taskID string) (*TaskProgress, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	task, exists := pa.tasks[taskID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
			WithComponent("orchestrator.scheduler").
			WithOperation("GetTaskProgress").
			WithDetails("task_id", taskID)
	}

	// Return a copy
	result := &TaskProgress{
		TaskID:        task.TaskID,
		Status:        task.Status,
		Percentage:    task.Percentage,
		Message:       task.Message,
		StartTime:     task.StartTime,
		LastUpdate:    task.LastUpdate,
		EstimatedEnd:  task.EstimatedEnd,
		ResourceUsage: task.ResourceUsage,
		SubTasks:      make(map[string]*SubTaskProgress),
	}

	// Copy sub-tasks
	for k, v := range task.SubTasks {
		result.SubTasks[k] = v
	}

	return result, nil
}
