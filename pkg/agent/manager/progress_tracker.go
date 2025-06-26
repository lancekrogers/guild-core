// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// ProgressTracker provides real-time monitoring and progress tracking for commissions and tasks
type ProgressTracker struct {
	db           *sql.DB
	commissionID string
	tasks        map[string]*TaskProgress
	agents       map[string]*AgentProgress
	timeline     *Timeline
	risks        []Risk
	lastUpdate   time.Time
}

// TaskProgress represents the progress state of a single task
type TaskProgress struct {
	TaskID         string                 `json:"task_id"`
	Status         kanban.TaskStatus      `json:"status"`
	StartTime      *time.Time             `json:"start_time,omitempty"`
	UpdatedAt      time.Time              `json:"updated_at"`
	BlockedInfo    *BlockedInfo           `json:"blocked_info,omitempty"`
	Completion     float64                `json:"completion"`     // 0.0-1.0
	EstimatedTime  time.Duration          `json:"estimated_time"`
	ActualTime     time.Duration          `json:"actual_time"`
	AssignedAgent  string                 `json:"assigned_agent"`
	Dependencies   []string               `json:"dependencies"`
	Dependents     []string               `json:"dependents"`     // Tasks that depend on this one
	Velocity       float64                `json:"velocity"`       // Progress per hour
	Metadata       map[string]interface{} `json:"metadata"`
}

// AgentProgress represents the progress state of an agent
type AgentProgress struct {
	AgentID       string        `json:"agent_id"`
	ActiveTasks   []string      `json:"active_tasks"`
	TasksToday    int           `json:"tasks_today"`
	HoursWorked   float64       `json:"hours_worked"`
	Efficiency    float64       `json:"efficiency"`    // Tasks completed vs estimated
	CurrentLoad   float64       `json:"current_load"`  // 0.0-1.0
	LastActivity  time.Time     `json:"last_activity"`
	BlockedTasks  []string      `json:"blocked_tasks"`
	Status        string        `json:"status"`        // active, idle, blocked, offline
}

// BlockedInfo contains information about why a task is blocked
type BlockedInfo struct {
	Reason         string    `json:"reason"`
	BlockedSince   time.Time `json:"blocked_since"`
	BlockingTasks  []string  `json:"blocking_tasks"`
	Resolution     string    `json:"resolution,omitempty"`  // Suggested resolution
	Severity       string    `json:"severity"`              // low, medium, high, critical
	AutoResolvable bool      `json:"auto_resolvable"`
}

// Timeline represents the project timeline with milestones and predictions
type Timeline struct {
	StartTime           time.Time            `json:"start_time"`
	EstimatedCompletion time.Time            `json:"estimated_completion"`
	OriginalEstimate    time.Time            `json:"original_estimate"`
	Milestones          []Milestone          `json:"milestones"`
	CriticalPath        []string             `json:"critical_path"`        // Task IDs on critical path
	ProjectionAccuracy  float64              `json:"projection_accuracy"`  // How accurate our estimates have been
	Confidence          float64              `json:"confidence"`           // Confidence in current projection
}

// Milestone represents a significant point in the project
type Milestone struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	TargetDate      time.Time `json:"target_date"`
	CompletionDate  *time.Time `json:"completion_date,omitempty"`
	DependentTasks  []string  `json:"dependent_tasks"`
	Status          string    `json:"status"`     // upcoming, at_risk, completed, overdue
	CriticalLevel   int       `json:"critical_level"` // 1-5, how critical this milestone is
}

// Risk represents a potential risk to project completion
type Risk struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // delay, quality, resource, dependency
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Probability float64   `json:"probability"` // 0.0-1.0
	Impact      string    `json:"impact"`      // Description of potential impact
	AffectedTasks []string `json:"affected_tasks"`
	Mitigation  string    `json:"mitigation"`  // Suggested mitigation strategy
	DetectedAt  time.Time `json:"detected_at"`
	Status      string    `json:"status"`      // active, mitigated, resolved
}

// StatusReport provides a comprehensive status overview
type StatusReport struct {
	OverallProgress     float64            `json:"overall_progress"`     // 0.0-1.0
	TasksComplete       int                `json:"tasks_complete"`
	TasksInProgress     int                `json:"tasks_in_progress"`
	TasksBlocked        int                `json:"tasks_blocked"`
	TasksBacklog        int                `json:"tasks_backlog"`
	EstimatedComplete   time.Time          `json:"estimated_complete"`
	Risks               []Risk             `json:"risks"`
	AgentUtilization    map[string]float64 `json:"agent_utilization"`    // agentID -> utilization
	CriticalPathStatus  string             `json:"critical_path_status"` // on_track, at_risk, delayed
	Velocity            float64            `json:"velocity"`             // Tasks completed per day
	QualityMetrics      QualityMetrics     `json:"quality_metrics"`
	Recommendations     []string           `json:"recommendations"`
	GeneratedAt         time.Time          `json:"generated_at"`
}

// QualityMetrics tracks quality-related metrics
type QualityMetrics struct {
	DefectRate        float64 `json:"defect_rate"`         // Defects per task
	ReworkRate        float64 `json:"rework_rate"`         // Tasks requiring rework
	ReviewPassRate    float64 `json:"review_pass_rate"`    // Tasks passing first review
	TestCoverage      float64 `json:"test_coverage"`       // Test coverage percentage
	CodeQualityScore  float64 `json:"code_quality_score"`  // Overall code quality (0-100)
}

// ProgressUpdate represents an update to task progress
type ProgressUpdate struct {
	TaskID     string                 `json:"task_id"`
	Status     kanban.TaskStatus      `json:"status"`
	Completion float64                `json:"completion"`
	Notes      string                 `json:"notes"`
	UpdatedBy  string                 `json:"updated_by"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// NewProgressTracker creates a new progress tracker for a commission
func NewProgressTracker(db *sql.DB, commissionID string) *ProgressTracker {
	return &ProgressTracker{
		db:           db,
		commissionID: commissionID,
		tasks:        make(map[string]*TaskProgress),
		agents:       make(map[string]*AgentProgress),
		timeline:     &Timeline{},
		risks:        []Risk{},
		lastUpdate:   time.Now(),
	}
}

// Initialize loads existing progress data and sets up tracking
func (pt *ProgressTracker) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProgressTracker").
			WithOperation("Initialize")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ProgressTracker")
	ctx = observability.WithOperation(ctx, "Initialize")

	logger.InfoContext(ctx, "Initializing progress tracker", "commission_id", pt.commissionID)

	// Load tasks for this commission
	if err := pt.loadTasks(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load tasks").
			WithComponent("ProgressTracker").
			WithOperation("Initialize").
			WithDetails("commission_id", pt.commissionID)
	}

	// Load agent progress
	if err := pt.loadAgentProgress(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load agent progress").
			WithComponent("ProgressTracker").
			WithOperation("Initialize").
			WithDetails("commission_id", pt.commissionID)
	}

	// Initialize timeline
	if err := pt.initializeTimeline(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize timeline").
			WithComponent("ProgressTracker").
			WithOperation("Initialize").
			WithDetails("commission_id", pt.commissionID)
	}

	// Detect initial risks
	pt.detectRisks(ctx)

	logger.InfoContext(ctx, "Progress tracker initialized",
		"commission_id", pt.commissionID,
		"tasks_count", len(pt.tasks),
		"agents_count", len(pt.agents),
		"risks_count", len(pt.risks))

	return nil
}

// UpdateTaskProgress updates the progress of a specific task
func (pt *ProgressTracker) UpdateTaskProgress(ctx context.Context, taskID string, update ProgressUpdate) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProgressTracker").
			WithOperation("UpdateTaskProgress")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ProgressTracker")
	ctx = observability.WithOperation(ctx, "UpdateTaskProgress")

	logger.DebugContext(ctx, "Updating task progress",
		"task_id", taskID,
		"new_status", update.Status,
		"completion", update.Completion,
		"updated_by", update.UpdatedBy)

	// Get or create task progress
	progress, exists := pt.tasks[taskID]
	if !exists {
		progress = &TaskProgress{
			TaskID:    taskID,
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
		pt.tasks[taskID] = progress
	}

	// Track status transitions
	oldStatus := progress.Status
	progress.Status = update.Status
	progress.Completion = update.Completion
	progress.UpdatedAt = update.Timestamp

	// Handle status-specific logic
	if err := pt.handleStatusTransition(ctx, progress, oldStatus, update); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to handle status transition").
			WithComponent("ProgressTracker").
			WithOperation("UpdateTaskProgress").
			WithDetails("task_id", taskID)
	}

	// Update velocity calculation
	pt.updateTaskVelocity(progress)

	// Persist to database
	if err := pt.persistTaskProgress(ctx, progress); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to persist task progress").
			WithComponent("ProgressTracker").
			WithOperation("UpdateTaskProgress").
			WithDetails("task_id", taskID)
	}

	// Update overall progress
	pt.calculateOverallProgress()

	// Check for new delays or risks
	if pt.isDelayed(taskID) {
		pt.notifyDelay(ctx, taskID)
	}

	// Update timeline projections
	pt.updateTimeline(ctx)

	// Detect new risks
	pt.detectRisks(ctx)

	pt.lastUpdate = time.Now()

	logger.InfoContext(ctx, "Task progress updated successfully",
		"task_id", taskID,
		"old_status", oldStatus,
		"new_status", update.Status,
		"completion", update.Completion)

	return nil
}

// GetStatusReport generates a comprehensive status report
func (pt *ProgressTracker) GetStatusReport(ctx context.Context) (*StatusReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ProgressTracker").
			WithOperation("GetStatusReport")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ProgressTracker")
	ctx = observability.WithOperation(ctx, "GetStatusReport")

	logger.DebugContext(ctx, "Generating status report", "commission_id", pt.commissionID)

	report := &StatusReport{
		OverallProgress:     pt.calculateOverallProgress(),
		TasksComplete:       pt.countByStatus(kanban.StatusDone),
		TasksInProgress:     pt.countByStatus(kanban.StatusInProgress),
		TasksBlocked:        pt.countByStatus(kanban.StatusBlocked),
		TasksBacklog:        pt.countByStatus(kanban.StatusBacklog),
		EstimatedComplete:   pt.timeline.EstimatedCompletion,
		Risks:               pt.risks,
		AgentUtilization:    pt.calculateAgentUtilization(),
		CriticalPathStatus:  pt.assessCriticalPathStatus(),
		Velocity:            pt.calculateVelocity(),
		QualityMetrics:      pt.calculateQualityMetrics(),
		Recommendations:     pt.generateRecommendations(ctx),
		GeneratedAt:         time.Now(),
	}

	logger.InfoContext(ctx, "Status report generated",
		"commission_id", pt.commissionID,
		"overall_progress", report.OverallProgress,
		"risks_count", len(report.Risks),
		"velocity", report.Velocity)

	return report, nil
}

// handleStatusTransition handles logic for status changes
func (pt *ProgressTracker) handleStatusTransition(ctx context.Context, progress *TaskProgress, oldStatus kanban.TaskStatus, update ProgressUpdate) error {
	now := time.Now()

	switch update.Status {
	case kanban.StatusInProgress:
		if progress.StartTime == nil {
			progress.StartTime = &now
		}
		// Update agent status
		if progress.AssignedAgent != "" {
			pt.updateAgentStatus(progress.AssignedAgent, "active")
		}

	case kanban.StatusDone:
		progress.Completion = 1.0
		if progress.StartTime != nil {
			progress.ActualTime = now.Sub(*progress.StartTime)
		}
		// Update agent workload
		if progress.AssignedAgent != "" {
			pt.updateAgentTaskCompletion(progress.AssignedAgent, progress.TaskID)
		}

	case kanban.StatusBlocked:
		// Create or update blocked info
		if progress.BlockedInfo == nil {
			progress.BlockedInfo = &BlockedInfo{
				BlockedSince: now,
				Severity:     "medium",
			}
		}
		progress.BlockedInfo.Reason = update.Notes

	default:
		// Remove blocked info if moving out of blocked status
		if oldStatus == kanban.StatusBlocked {
			progress.BlockedInfo = nil
		}
	}

	return nil
}

// updateTaskVelocity updates the velocity calculation for a task
func (pt *ProgressTracker) updateTaskVelocity(progress *TaskProgress) {
	if progress.StartTime == nil {
		progress.Velocity = 0
		return
	}

	elapsed := time.Since(*progress.StartTime)
	if elapsed > 0 {
		progress.Velocity = progress.Completion / elapsed.Hours()
	}
}

// calculateOverallProgress calculates the overall progress percentage
func (pt *ProgressTracker) calculateOverallProgress() float64 {
	if len(pt.tasks) == 0 {
		return 0.0
	}

	totalCompletion := 0.0
	for _, task := range pt.tasks {
		totalCompletion += task.Completion
	}

	return totalCompletion / float64(len(pt.tasks))
}

// countByStatus counts tasks with a specific status
func (pt *ProgressTracker) countByStatus(status kanban.TaskStatus) int {
	count := 0
	for _, task := range pt.tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

// isDelayed checks if a task is behind schedule
func (pt *ProgressTracker) isDelayed(taskID string) bool {
	task, exists := pt.tasks[taskID]
	if !exists || task.StartTime == nil {
		return false
	}

	// Simple heuristic: if actual time > estimated time and not complete
	if task.EstimatedTime > 0 && task.Completion < 1.0 {
		elapsed := time.Since(*task.StartTime)
		expectedCompletion := task.EstimatedTime * time.Duration(1.0/task.Completion)
		return elapsed > expectedCompletion
	}

	return false
}

// notifyDelay handles notification when a task is delayed
func (pt *ProgressTracker) notifyDelay(ctx context.Context, taskID string) {
	logger := observability.GetLogger(ctx)
	
	task := pt.tasks[taskID]
	delay := time.Since(*task.StartTime) - task.EstimatedTime
	
	logger.WarnContext(ctx, "Task delay detected",
		"task_id", taskID,
		"assigned_agent", task.AssignedAgent,
		"delay_hours", delay.Hours(),
		"completion", task.Completion)

	// Create a risk entry
	risk := Risk{
		ID:            generateID(),
		Type:          "delay",
		Severity:      "medium",
		Probability:   0.8,
		Impact:        "Task is behind schedule and may affect dependent tasks",
		AffectedTasks: []string{taskID},
		Mitigation:    "Consider reassigning resources or breaking down the task",
		DetectedAt:    time.Now(),
		Status:        "active",
	}

	// Add dependents to affected tasks
	risk.AffectedTasks = append(risk.AffectedTasks, task.Dependents...)

	pt.risks = append(pt.risks, risk)
}

// updateTimeline updates timeline projections based on current progress
func (pt *ProgressTracker) updateTimeline(ctx context.Context) {
	logger := observability.GetLogger(ctx)

	// Calculate new estimated completion based on current velocity
	totalTasks := len(pt.tasks)
	completedTasks := pt.countByStatus(kanban.StatusDone)
	inProgressTasks := pt.countByStatus(kanban.StatusInProgress)

	if totalTasks == 0 {
		return
	}

	// Calculate average velocity
	avgVelocity := pt.calculateVelocity()
	if avgVelocity <= 0 {
		return
	}

	remainingTasks := totalTasks - completedTasks
	estimatedDays := float64(remainingTasks) / avgVelocity

	pt.timeline.EstimatedCompletion = time.Now().Add(time.Duration(estimatedDays * 24) * time.Hour)

	// Update projection accuracy
	if !pt.timeline.OriginalEstimate.IsZero() {
		originalDays := pt.timeline.OriginalEstimate.Sub(pt.timeline.StartTime).Hours() / 24
		currentDays := pt.timeline.EstimatedCompletion.Sub(pt.timeline.StartTime).Hours() / 24
		
		if originalDays > 0 {
			pt.timeline.ProjectionAccuracy = 1.0 - math.Abs(currentDays-originalDays)/originalDays
		}
	}

	// Calculate confidence based on data quality
	dataPoints := completedTasks + inProgressTasks
	pt.timeline.Confidence = math.Min(1.0, float64(dataPoints)/10.0) // Full confidence with 10+ data points

	logger.DebugContext(ctx, "Timeline updated",
		"estimated_completion", pt.timeline.EstimatedCompletion,
		"velocity", avgVelocity,
		"confidence", pt.timeline.Confidence)
}

// detectRisks analyzes current state to detect potential risks
func (pt *ProgressTracker) detectRisks(ctx context.Context) {
	logger := observability.GetLogger(ctx)

	// Clear old risks
	pt.risks = []Risk{}

	// Detect blocked task risks
	for taskID, task := range pt.tasks {
		if task.Status == kanban.StatusBlocked && task.BlockedInfo != nil {
			severity := task.BlockedInfo.Severity
			if severity == "" {
				severity = "medium"
			}

			risk := Risk{
				ID:            generateID(),
				Type:          "dependency",
				Severity:      severity,
				Probability:   0.9,
				Impact:        "Blocked task may delay dependent work",
				AffectedTasks: append([]string{taskID}, task.Dependents...),
				Mitigation:    "Resolve blocking dependencies or find alternative approach",
				DetectedAt:    time.Now(),
				Status:        "active",
			}
			pt.risks = append(pt.risks, risk)
		}
	}

	// Detect resource utilization risks
	overloadedAgents := 0
	for _, agent := range pt.agents {
		if agent.CurrentLoad > 0.9 {
			overloadedAgents++
		}
	}

	if overloadedAgents > len(pt.agents)/2 {
		risk := Risk{
			ID:          generateID(),
			Type:        "resource",
			Severity:    "high",
			Probability: 0.8,
			Impact:      "Team is overloaded and may experience burnout or quality issues",
			Mitigation:  "Redistribute workload or consider additional resources",
			DetectedAt:  time.Now(),
			Status:      "active",
		}
		pt.risks = append(pt.risks, risk)
	}

	logger.DebugContext(ctx, "Risk detection completed", "risks_found", len(pt.risks))
}

// calculateAgentUtilization calculates utilization for each agent
func (pt *ProgressTracker) calculateAgentUtilization() map[string]float64 {
	utilization := make(map[string]float64)
	
	for agentID, agent := range pt.agents {
		utilization[agentID] = agent.CurrentLoad
	}
	
	return utilization
}

// assessCriticalPathStatus assesses the status of the critical path
func (pt *ProgressTracker) assessCriticalPathStatus() string {
	if len(pt.timeline.CriticalPath) == 0 {
		return "unknown"
	}

	delayedTasks := 0
	totalTasks := len(pt.timeline.CriticalPath)

	for _, taskID := range pt.timeline.CriticalPath {
		if pt.isDelayed(taskID) {
			delayedTasks++
		}
	}

	delayRatio := float64(delayedTasks) / float64(totalTasks)
	
	if delayRatio == 0 {
		return "on_track"
	} else if delayRatio < 0.3 {
		return "at_risk"
	} else {
		return "delayed"
	}
}

// calculateVelocity calculates the team's velocity (tasks per day)
func (pt *ProgressTracker) calculateVelocity() float64 {
	completedTasks := pt.countByStatus(kanban.StatusDone)
	
	if completedTasks == 0 || pt.timeline.StartTime.IsZero() {
		return 0.0
	}

	elapsed := time.Since(pt.timeline.StartTime)
	days := elapsed.Hours() / 24.0
	
	if days <= 0 {
		return 0.0
	}

	return float64(completedTasks) / days
}

// calculateQualityMetrics calculates quality-related metrics
func (pt *ProgressTracker) calculateQualityMetrics() QualityMetrics {
	// This would be enhanced with actual quality data
	return QualityMetrics{
		DefectRate:       0.05, // Default values - would be calculated from real data
		ReworkRate:       0.10,
		ReviewPassRate:   0.85,
		TestCoverage:     0.75,
		CodeQualityScore: 85.0,
	}
}

// generateRecommendations generates actionable recommendations
func (pt *ProgressTracker) generateRecommendations(ctx context.Context) []string {
	var recommendations []string

	// Check for blocked tasks
	blockedCount := pt.countByStatus(kanban.StatusBlocked)
	if blockedCount > 0 {
		recommendations = append(recommendations, 
			"Address blocked tasks to prevent further delays")
	}

	// Check for overloaded agents
	overloadedCount := 0
	for _, agent := range pt.agents {
		if agent.CurrentLoad > 0.8 {
			overloadedCount++
		}
	}
	
	if overloadedCount > 0 {
		recommendations = append(recommendations,
			"Rebalance workload among team members")
	}

	// Check velocity
	velocity := pt.calculateVelocity()
	if velocity < 1.0 {
		recommendations = append(recommendations,
			"Consider breaking down large tasks to improve velocity")
	}

	// Check critical path
	if pt.assessCriticalPathStatus() != "on_track" {
		recommendations = append(recommendations,
			"Focus resources on critical path tasks")
	}

	return recommendations
}

// Placeholder methods for database operations
func (pt *ProgressTracker) loadTasks(ctx context.Context) error {
	// Implementation would load tasks from database
	return nil
}

func (pt *ProgressTracker) loadAgentProgress(ctx context.Context) error {
	// Implementation would load agent progress from database
	return nil
}

func (pt *ProgressTracker) initializeTimeline(ctx context.Context) error {
	// Implementation would initialize timeline from commission data
	pt.timeline.StartTime = time.Now()
	return nil
}

func (pt *ProgressTracker) persistTaskProgress(ctx context.Context, progress *TaskProgress) error {
	// Implementation would persist to database
	return nil
}

func (pt *ProgressTracker) updateAgentStatus(agentID, status string) {
	if agent, exists := pt.agents[agentID]; exists {
		agent.Status = status
		agent.LastActivity = time.Now()
	}
}

func (pt *ProgressTracker) updateAgentTaskCompletion(agentID, taskID string) {
	if agent, exists := pt.agents[agentID]; exists {
		agent.TasksToday++
		// Remove from active tasks
		for i, activeTask := range agent.ActiveTasks {
			if activeTask == taskID {
				agent.ActiveTasks = append(agent.ActiveTasks[:i], agent.ActiveTasks[i+1:]...)
				break
			}
		}
		// Update current load
		agent.CurrentLoad = float64(len(agent.ActiveTasks)) / 5.0 // Assuming max 5 concurrent tasks
	}
}