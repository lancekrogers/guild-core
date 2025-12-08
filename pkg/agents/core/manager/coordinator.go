// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/kanban"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// Coordinator provides intelligent multi-agent coordination with adaptive strategies
type Coordinator struct {
	agents               map[string]core.Agent
	tasks                map[string]*kanban.Task
	capabilityManager    *CapabilityModelManager
	taskAssigner         *TaskAssigner
	progressTracker      *ProgressTracker
	communicationChannel CommunicationChannel
	strategies           []CoordinationStrategy
	db                   *sql.DB
	commissionID         string
	lastCoordination     time.Time
	coordinationHistory  []CoordinationEvent
}

// CoordinationStrategy defines an interface for coordination strategies
type CoordinationStrategy interface {
	// ShouldApply determines if this strategy should be applied given the current context
	ShouldApply(ctx context.Context, context CoordinationContext) bool

	// Apply executes the coordination strategy
	Apply(ctx context.Context, coordinator *Coordinator) error

	// GetPriority returns the priority of this strategy (higher number = higher priority)
	GetPriority() int

	// GetName returns a human-readable name for this strategy
	GetName() string
}

// CoordinationContext provides context for strategy decision making
type CoordinationContext struct {
	TotalTasks         int                `json:"total_tasks"`
	CompletedTasks     int                `json:"completed_tasks"`
	BlockedTasks       int                `json:"blocked_tasks"`
	AgentUtilization   map[string]float64 `json:"agent_utilization"`
	CriticalPathStatus string             `json:"critical_path_status"`
	RecentBottlenecks  []Bottleneck       `json:"recent_bottlenecks"`
	TimeToDeadline     time.Duration      `json:"time_to_deadline"`
	QualityIssues      int                `json:"quality_issues"`
	TeamMorale         float64            `json:"team_morale"`
	Risks              []Risk             `json:"risks"`
}

// CoordinationEvent records a coordination action taken
type CoordinationEvent struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Strategy       string    `json:"strategy"`
	Action         string    `json:"action"`
	AffectedTasks  []string  `json:"affected_tasks"`
	AffectedAgents []string  `json:"affected_agents"`
	Reasoning      string    `json:"reasoning"`
	Outcome        string    `json:"outcome"`
	Success        bool      `json:"success"`
}

// Bottleneck represents a coordination bottleneck
type Bottleneck struct {
	ID            string        `json:"id"`
	Type          string        `json:"type"`     // resource, dependency, approval, technical
	Location      string        `json:"location"` // Where the bottleneck is occurring
	AffectedTasks []string      `json:"affected_tasks"`
	Impact        string        `json:"impact"`   // Description of impact
	Severity      int           `json:"severity"` // 1-5 severity rating
	DetectedAt    time.Time     `json:"detected_at"`
	Duration      time.Duration `json:"duration"`     // How long it's been a bottleneck
	AutoResolve   bool          `json:"auto_resolve"` // Can this be automatically resolved
}

// CommunicationChannel defines interface for agent communication
type CommunicationChannel interface {
	SendMessage(ctx context.Context, from, to string, message string) error
	BroadcastMessage(ctx context.Context, from string, message string) error
	RequestHelp(ctx context.Context, agentID string, taskID string, helpType string) error
	ShareKnowledge(ctx context.Context, agentID string, knowledge string, tags []string) error
}

// NewCoordinator creates a new intelligent coordinator
func NewCoordinator(
	agents map[string]core.Agent,
	capabilityManager *CapabilityModelManager,
	taskAssigner *TaskAssigner,
	progressTracker *ProgressTracker,
	communicationChannel CommunicationChannel,
	db *sql.DB,
	commissionID string,
) *Coordinator {
	coordinator := &Coordinator{
		agents:               agents,
		tasks:                make(map[string]*kanban.Task),
		capabilityManager:    capabilityManager,
		taskAssigner:         taskAssigner,
		progressTracker:      progressTracker,
		communicationChannel: communicationChannel,
		strategies:           []CoordinationStrategy{},
		db:                   db,
		commissionID:         commissionID,
		coordinationHistory:  []CoordinationEvent{},
	}

	// Initialize default strategies
	coordinator.initializeStrategies()

	return coordinator
}

// CoordinateAgents performs intelligent coordination of all agents
func (c *Coordinator) CoordinateAgents(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Coordinator").
			WithOperation("CoordinateAgents")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "Coordinator")
	ctx = observability.WithOperation(ctx, "CoordinateAgents")

	logger.InfoContext(ctx, "Starting agent coordination",
		"commission_id", c.commissionID,
		"agents_count", len(c.agents),
		"tasks_count", len(c.tasks),
		"strategies_count", len(c.strategies))

	// Gather coordination context
	context, err := c.gatherCoordinationContext(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to gather coordination context").
			WithComponent("Coordinator").
			WithOperation("CoordinateAgents")
	}

	// Apply strategies in priority order
	applicableStrategies := c.getApplicableStrategies(ctx, context)

	logger.InfoContext(ctx, "Applying coordination strategies",
		"applicable_count", len(applicableStrategies),
		"context", context)

	for _, strategy := range applicableStrategies {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during strategy application").
				WithComponent("Coordinator").
				WithOperation("CoordinateAgents")
		}

		logger.InfoContext(ctx, "Applying coordination strategy", "strategy", strategy.GetName())

		event := CoordinationEvent{
			ID:        generateID(),
			Timestamp: time.Now(),
			Strategy:  strategy.GetName(),
		}

		err := strategy.Apply(ctx, c)
		if err != nil {
			event.Success = false
			event.Outcome = fmt.Sprintf("Failed: %v", err)
			logger.WarnContext(ctx, "Strategy application failed",
				"strategy", strategy.GetName(),
				"error", err)
		} else {
			event.Success = true
			event.Outcome = "Applied successfully"
			logger.InfoContext(ctx, "Strategy applied successfully", "strategy", strategy.GetName())
		}

		c.coordinationHistory = append(c.coordinationHistory, event)
	}

	c.lastCoordination = time.Now()

	logger.InfoContext(ctx, "Agent coordination completed",
		"strategies_applied", len(applicableStrategies))

	return nil
}

// initializeStrategies sets up the default coordination strategies
func (c *Coordinator) initializeStrategies() {
	c.strategies = []CoordinationStrategy{
		&ParallelizationStrategy{},
		&BottleneckMitigationStrategy{},
		&WorkloadBalancingStrategy{},
		&QualityAssuranceStrategy{},
		&DeadlineManagementStrategy{},
		&CommunicationOptimizationStrategy{},
	}

	// Sort strategies by priority
	sort.Slice(c.strategies, func(i, j int) bool {
		return c.strategies[i].GetPriority() > c.strategies[j].GetPriority()
	})
}

// gatherCoordinationContext collects current state for strategy decisions
func (c *Coordinator) gatherCoordinationContext(ctx context.Context) (CoordinationContext, error) {
	logger := observability.GetLogger(ctx)

	// Get status report from progress tracker
	statusReport, err := c.progressTracker.GetStatusReport(ctx)
	if err != nil {
		return CoordinationContext{}, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get status report").
			WithComponent("Coordinator").
			WithOperation("gatherCoordinationContext")
	}

	// Detect bottlenecks
	bottlenecks := c.detectBottlenecks(ctx)

	context := CoordinationContext{
		TotalTasks:         len(c.tasks),
		CompletedTasks:     statusReport.TasksComplete,
		BlockedTasks:       statusReport.TasksBlocked,
		AgentUtilization:   statusReport.AgentUtilization,
		CriticalPathStatus: statusReport.CriticalPathStatus,
		RecentBottlenecks:  bottlenecks,
		QualityIssues:      0,   // Would be calculated from quality metrics
		TeamMorale:         0.8, // Would be calculated from performance data
		Risks:              statusReport.Risks,
	}

	// Calculate time to deadline (if available)
	if !statusReport.EstimatedComplete.IsZero() {
		context.TimeToDeadline = time.Until(statusReport.EstimatedComplete)
	}

	logger.DebugContext(ctx, "Coordination context gathered",
		"total_tasks", context.TotalTasks,
		"completed_tasks", context.CompletedTasks,
		"blocked_tasks", context.BlockedTasks,
		"bottlenecks", len(context.RecentBottlenecks))

	return context, nil
}

// getApplicableStrategies returns strategies that should be applied
func (c *Coordinator) getApplicableStrategies(ctx context.Context, context CoordinationContext) []CoordinationStrategy {
	var applicable []CoordinationStrategy

	for _, strategy := range c.strategies {
		if strategy.ShouldApply(ctx, context) {
			applicable = append(applicable, strategy)
		}
	}

	return applicable
}

// detectBottlenecks identifies current bottlenecks in the workflow
func (c *Coordinator) detectBottlenecks(ctx context.Context) []Bottleneck {
	var bottlenecks []Bottleneck

	// Detect resource bottlenecks (overloaded agents)
	for agentID, utilization := range c.getAgentUtilization() {
		if utilization > 0.9 {
			bottleneck := Bottleneck{
				ID:            generateID(),
				Type:          "resource",
				Location:      fmt.Sprintf("Agent: %s", agentID),
				AffectedTasks: c.getAgentTasks(agentID),
				Impact:        "Agent overload may cause delays and quality issues",
				Severity:      4,
				DetectedAt:    time.Now(),
				AutoResolve:   true,
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	// Detect dependency bottlenecks (tasks waiting on dependencies)
	for taskID, task := range c.tasks {
		if task.Status == kanban.StatusBlocked && len(task.Dependencies) > 0 {
			bottleneck := Bottleneck{
				ID:            generateID(),
				Type:          "dependency",
				Location:      fmt.Sprintf("Task: %s", taskID),
				AffectedTasks: append([]string{taskID}, c.getDependentTasks(taskID)...),
				Impact:        "Blocked task is preventing dependent work",
				Severity:      3,
				DetectedAt:    time.Now(),
				AutoResolve:   false,
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

// Strategy Implementations

// ParallelizationStrategy identifies and parallelizes independent tasks
type ParallelizationStrategy struct{}

func (ps *ParallelizationStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	// Apply if there are available agents and parallelizable tasks
	availableAgents := 0
	for _, utilization := range context.AgentUtilization {
		if utilization < 0.7 {
			availableAgents++
		}
	}
	return availableAgents > 0 && context.TotalTasks > context.CompletedTasks
}

func (ps *ParallelizationStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying parallelization strategy")

	// Find independent tasks that can be parallelized
	independentTasks := coordinator.findIndependentTasks()

	// Get available agents
	availableAgents := coordinator.getAvailableAgents()

	if len(independentTasks) == 0 || len(availableAgents) == 0 {
		return nil
	}

	// Assign independent tasks to available agents
	assigned := 0
	for i, task := range independentTasks {
		if i >= len(availableAgents) {
			break
		}

		agentID := availableAgents[i]
		err := coordinator.assignTaskToAgent(ctx, task.ID, agentID, "parallelization")
		if err != nil {
			logger.WarnContext(ctx, "Failed to assign task for parallelization",
				"task_id", task.ID, "agent_id", agentID, "error", err)
			continue
		}
		assigned++
	}

	logger.InfoContext(ctx, "Parallelization strategy completed", "tasks_assigned", assigned)
	return nil
}

func (ps *ParallelizationStrategy) GetPriority() int { return 8 }
func (ps *ParallelizationStrategy) GetName() string  { return "Parallelization" }

// BottleneckMitigationStrategy addresses identified bottlenecks
type BottleneckMitigationStrategy struct{}

func (bms *BottleneckMitigationStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	return len(context.RecentBottlenecks) > 0
}

func (bms *BottleneckMitigationStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying bottleneck mitigation strategy")

	mitigated := 0
	for _, bottleneck := range coordinator.detectBottlenecks(ctx) {
		switch bottleneck.Type {
		case "resource":
			// Try to redistribute tasks from overloaded agents
			if coordinator.redistributeTasksFromOverloadedAgent(ctx, bottleneck.Location) {
				mitigated++
			}
		case "dependency":
			// Try to resolve dependency or find alternative approach
			if coordinator.resolveDependencyBottleneck(ctx, bottleneck.AffectedTasks[0]) {
				mitigated++
			}
		}
	}

	logger.InfoContext(ctx, "Bottleneck mitigation completed", "bottlenecks_mitigated", mitigated)
	return nil
}

func (bms *BottleneckMitigationStrategy) GetPriority() int { return 9 }
func (bms *BottleneckMitigationStrategy) GetName() string  { return "BottleneckMitigation" }

// WorkloadBalancingStrategy balances work across agents
type WorkloadBalancingStrategy struct{}

func (wbs *WorkloadBalancingStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	// Calculate workload variance
	if len(context.AgentUtilization) < 2 {
		return false
	}

	var max, min float64 = 0, 1
	for _, utilization := range context.AgentUtilization {
		if utilization > max {
			max = utilization
		}
		if utilization < min {
			min = utilization
		}
	}

	// Apply if there's significant imbalance
	return (max - min) > 0.3
}

func (wbs *WorkloadBalancingStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying workload balancing strategy")

	balanced := coordinator.balanceWorkload(ctx)

	logger.InfoContext(ctx, "Workload balancing completed", "tasks_rebalanced", balanced)
	return nil
}

func (wbs *WorkloadBalancingStrategy) GetPriority() int { return 6 }
func (wbs *WorkloadBalancingStrategy) GetName() string  { return "WorkloadBalancing" }

// QualityAssuranceStrategy ensures quality standards
type QualityAssuranceStrategy struct{}

func (qas *QualityAssuranceStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	return context.QualityIssues > 2 // Apply if quality issues detected
}

func (qas *QualityAssuranceStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying quality assurance strategy")

	// Implement quality improvements
	improved := coordinator.implementQualityMeasures(ctx)

	logger.InfoContext(ctx, "Quality assurance completed", "measures_implemented", improved)
	return nil
}

func (qas *QualityAssuranceStrategy) GetPriority() int { return 7 }
func (qas *QualityAssuranceStrategy) GetName() string  { return "QualityAssurance" }

// DeadlineManagementStrategy manages deadline pressures
type DeadlineManagementStrategy struct{}

func (dms *DeadlineManagementStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	// Apply if deadline is approaching and progress is behind
	if context.TimeToDeadline <= 0 {
		return false
	}

	expectedProgress := 1.0 - (context.TimeToDeadline.Hours() / (7 * 24)) // Assuming 1 week projects
	actualProgress := float64(context.CompletedTasks) / float64(context.TotalTasks)

	return actualProgress < expectedProgress
}

func (dms *DeadlineManagementStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying deadline management strategy")

	actions := coordinator.implementDeadlineActions(ctx)

	logger.InfoContext(ctx, "Deadline management completed", "actions_taken", actions)
	return nil
}

func (dms *DeadlineManagementStrategy) GetPriority() int { return 10 }
func (dms *DeadlineManagementStrategy) GetName() string  { return "DeadlineManagement" }

// CommunicationOptimizationStrategy optimizes team communication
type CommunicationOptimizationStrategy struct{}

func (cos *CommunicationOptimizationStrategy) ShouldApply(ctx context.Context, context CoordinationContext) bool {
	// Apply if there are communication gaps or blocked tasks
	return context.BlockedTasks > 0 || context.TeamMorale < 0.7
}

func (cos *CommunicationOptimizationStrategy) Apply(ctx context.Context, coordinator *Coordinator) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Applying communication optimization strategy")

	improvements := coordinator.optimizeCommunication(ctx)

	logger.InfoContext(ctx, "Communication optimization completed", "improvements_made", improvements)
	return nil
}

func (cos *CommunicationOptimizationStrategy) GetPriority() int { return 5 }
func (cos *CommunicationOptimizationStrategy) GetName() string  { return "CommunicationOptimization" }

// Helper methods for coordinator operations

func (c *Coordinator) findIndependentTasks() []*kanban.Task {
	var independent []*kanban.Task

	for _, task := range c.tasks {
		if task.Status == kanban.StatusTodo && len(task.Dependencies) == 0 && task.AssignedTo == "" {
			independent = append(independent, task)
		}
	}

	return independent
}

func (c *Coordinator) getAvailableAgents() []string {
	var available []string

	utilization := c.getAgentUtilization()
	for agentID, util := range utilization {
		if util < 0.7 { // Consider agents with <70% utilization as available
			available = append(available, agentID)
		}
	}

	return available
}

func (c *Coordinator) getAgentUtilization() map[string]float64 {
	utilization := make(map[string]float64)

	// Count tasks per agent
	taskCounts := make(map[string]int)
	for _, task := range c.tasks {
		if task.AssignedTo != "" && task.Status != kanban.StatusDone {
			taskCounts[task.AssignedTo]++
		}
	}

	// Calculate utilization (assuming max 5 concurrent tasks)
	for agentID := range c.agents {
		count := taskCounts[agentID]
		utilization[agentID] = float64(count) / 5.0
	}

	return utilization
}

func (c *Coordinator) getAgentTasks(agentID string) []string {
	var tasks []string
	for taskID, task := range c.tasks {
		if task.AssignedTo == agentID {
			tasks = append(tasks, taskID)
		}
	}
	return tasks
}

func (c *Coordinator) getDependentTasks(taskID string) []string {
	var dependents []string
	for id, task := range c.tasks {
		for _, dep := range task.Dependencies {
			if dep == taskID {
				dependents = append(dependents, id)
				break
			}
		}
	}
	return dependents
}

func (c *Coordinator) assignTaskToAgent(ctx context.Context, taskID, agentID, reason string) error {
	logger := observability.GetLogger(ctx)

	task, exists := c.tasks[taskID]
	if !exists {
		return gerror.New(gerror.ErrCodeValidation, "task not found", nil).
			WithDetails("task_id", taskID)
	}

	task.AssignedTo = agentID
	task.UpdatedAt = time.Now()

	logger.InfoContext(ctx, "Task assigned to agent",
		"task_id", taskID,
		"agent_id", agentID,
		"reason", reason)

	return nil
}

func (c *Coordinator) redistributeTasksFromOverloadedAgent(ctx context.Context, location string) bool {
	// Extract agent ID from location string
	agentID := strings.TrimPrefix(location, "Agent: ")

	// Find tasks that can be redistributed
	redistributableTasks := c.getRedistributableTasks(agentID)
	if len(redistributableTasks) == 0 {
		return false
	}

	// Find available agents
	availableAgents := c.getAvailableAgents()
	if len(availableAgents) == 0 {
		return false
	}

	// Redistribute tasks
	for i, taskID := range redistributableTasks {
		if i >= len(availableAgents) {
			break
		}
		c.assignTaskToAgent(ctx, taskID, availableAgents[i], "load_balancing")
	}

	return true
}

func (c *Coordinator) getRedistributableTasks(agentID string) []string {
	var redistributable []string

	for taskID, task := range c.tasks {
		if task.AssignedTo == agentID &&
			task.Status == kanban.StatusTodo &&
			len(task.Dependencies) == 0 {
			redistributable = append(redistributable, taskID)
		}
	}

	return redistributable
}

func (c *Coordinator) resolveDependencyBottleneck(ctx context.Context, taskID string) bool {
	// Placeholder - would implement dependency resolution logic
	return false
}

func (c *Coordinator) balanceWorkload(ctx context.Context) int {
	// Placeholder - would implement workload balancing logic
	return 0
}

func (c *Coordinator) implementQualityMeasures(ctx context.Context) int {
	// Placeholder - would implement quality improvement measures
	return 0
}

func (c *Coordinator) implementDeadlineActions(ctx context.Context) int {
	// Placeholder - would implement deadline management actions
	return 0
}

func (c *Coordinator) optimizeCommunication(ctx context.Context) int {
	// Placeholder - would implement communication optimization
	return 0
}
