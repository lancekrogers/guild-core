// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/kanban"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// TaskAssigner implements intelligent task assignment using agent capability models
type TaskAssigner struct {
	capabilityManager *CapabilityModelManager
	constraints       AssignmentConstraints
	db                *sql.DB
}

// AssignmentConstraints defines rules for task assignment
type AssignmentConstraints struct {
	MaxTasksPerAgent  int     `json:"max_tasks_per_agent"`
	PreferSpecialists bool    `json:"prefer_specialists"`
	BalanceWorkload   bool    `json:"balance_workload"`
	CostOptimization  bool    `json:"cost_optimization"`
	MinScoreThreshold float64 `json:"min_score_threshold"` // Minimum score to consider assignment
	MaxWorkloadRatio  float64 `json:"max_workload_ratio"`  // Maximum workload before agent is considered overloaded
}

// TaskAssignment represents an assignment of a task to an agent
type TaskAssignment struct {
	TaskID        string        `json:"task_id"`
	AgentID       string        `json:"agent_id"`
	Score         float64       `json:"score"`
	Confidence    float64       `json:"confidence"` // How confident we are in this assignment
	Reasoning     string        `json:"reasoning"`  // Why this agent was chosen
	EstimatedTime time.Duration `json:"estimated_time"`
	AssignedAt    time.Time     `json:"assigned_at"`
}

// AssignmentPlan represents a complete assignment plan for multiple tasks
type AssignmentPlan struct {
	Assignments        []TaskAssignment `json:"assignments"`
	UnassignedTasks    []string         `json:"unassigned_tasks"`    // Tasks that couldn't be assigned
	OverloadedAgents   []string         `json:"overloaded_agents"`   // Agents that are near capacity
	ConstraintsApplied []string         `json:"constraints_applied"` // Which constraints were used
	TotalScore         float64          `json:"total_score"`         // Overall quality of the plan
	GeneratedAt        time.Time        `json:"generated_at"`
}

// NewTaskAssigner creates a new intelligent task assigner
func NewTaskAssigner(capabilityManager *CapabilityModelManager, db *sql.DB) *TaskAssigner {
	return &TaskAssigner{
		capabilityManager: capabilityManager,
		db:                db,
		constraints: AssignmentConstraints{
			MaxTasksPerAgent:  5,
			PreferSpecialists: true,
			BalanceWorkload:   true,
			CostOptimization:  false,
			MinScoreThreshold: 0.3,
			MaxWorkloadRatio:  0.8,
		},
	}
}

// SetConstraints updates the assignment constraints
func (ta *TaskAssigner) SetConstraints(constraints AssignmentConstraints) {
	ta.constraints = constraints
}

// AssignTasks creates an assignment plan for multiple tasks
func (ta *TaskAssigner) AssignTasks(ctx context.Context, tasks []*kanban.Task, availableAgentIDs []string) (*AssignmentPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("TaskAssigner").
			WithOperation("AssignTasks")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "TaskAssigner")
	ctx = observability.WithOperation(ctx, "AssignTasks")

	logger.InfoContext(ctx, "Starting intelligent task assignment",
		"tasks_count", len(tasks),
		"available_agents", len(availableAgentIDs),
		"constraints", ta.constraints)

	// Load agent models
	agentModels := make(map[string]*AgentCapabilityModel)
	for _, agentID := range availableAgentIDs {
		model, err := ta.capabilityManager.LoadAgentModel(ctx, agentID)
		if err != nil {
			logger.WarnContext(ctx, "Failed to load agent model", "agent_id", agentID, "error", err)
			continue
		}
		agentModels[agentID] = model
	}

	if len(agentModels) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no valid agent models available", nil).
			WithComponent("TaskAssigner").
			WithOperation("AssignTasks")
	}

	// Sort tasks by priority and dependencies
	sortedTasks, err := ta.topologicalSort(ctx, tasks)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to sort tasks").
			WithComponent("TaskAssigner").
			WithOperation("AssignTasks")
	}

	// Create assignment plan
	plan := &AssignmentPlan{
		Assignments:        []TaskAssignment{},
		UnassignedTasks:    []string{},
		OverloadedAgents:   []string{},
		ConstraintsApplied: []string{},
		GeneratedAt:        time.Now(),
	}

	// Track assignments for workload balancing
	agentAssignments := make(map[string][]TaskAssignment)

	// Assign each task
	for _, task := range sortedTasks {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during assignment").
				WithComponent("TaskAssigner").
				WithOperation("AssignTasks")
		}

		assignment, err := ta.assignSingleTask(ctx, task, agentModels, agentAssignments)
		if err != nil {
			logger.WarnContext(ctx, "Failed to assign task", "task_id", task.ID, "error", err)
			plan.UnassignedTasks = append(plan.UnassignedTasks, task.ID)
			continue
		}

		if assignment != nil {
			plan.Assignments = append(plan.Assignments, *assignment)
			agentAssignments[assignment.AgentID] = append(agentAssignments[assignment.AgentID], *assignment)

			logger.InfoContext(ctx, "Task assigned successfully",
				"task_id", task.ID,
				"agent_id", assignment.AgentID,
				"score", assignment.Score,
				"reasoning", assignment.Reasoning)
		} else {
			plan.UnassignedTasks = append(plan.UnassignedTasks, task.ID)
		}
	}

	// Identify overloaded agents
	plan.OverloadedAgents = ta.identifyOverloadedAgents(agentModels, agentAssignments)

	// Calculate total plan score
	plan.TotalScore = ta.calculatePlanScore(plan.Assignments)

	// Record applied constraints
	plan.ConstraintsApplied = ta.getAppliedConstraints()

	logger.InfoContext(ctx, "Task assignment completed",
		"assigned_count", len(plan.Assignments),
		"unassigned_count", len(plan.UnassignedTasks),
		"overloaded_agents", len(plan.OverloadedAgents),
		"total_score", plan.TotalScore)

	return plan, nil
}

// assignSingleTask assigns a single task to the best available agent
func (ta *TaskAssigner) assignSingleTask(ctx context.Context, task *kanban.Task, agentModels map[string]*AgentCapabilityModel, currentAssignments map[string][]TaskAssignment) (*TaskAssignment, error) {
	logger := observability.GetLogger(ctx)

	// Score each agent for this task
	scores := make(map[string]float64)
	reasonings := make(map[string]string)

	for agentID, model := range agentModels {
		// Check if agent can take more tasks
		if !ta.canAssignToAgent(agentID, model, currentAssignments) {
			logger.DebugContext(ctx, "Agent cannot take more tasks",
				"agent_id", agentID,
				"current_assignments", len(currentAssignments[agentID]))
			continue
		}

		score, err := model.ScoreForTask(ctx, task)
		if err != nil {
			logger.WarnContext(ctx, "Failed to score task for agent",
				"agent_id", agentID, "task_id", task.ID, "error", err)
			continue
		}

		// Apply constraints to adjust score
		adjustedScore := ta.applyConstraintsToScore(score, agentID, model, currentAssignments)

		if adjustedScore >= ta.constraints.MinScoreThreshold {
			scores[agentID] = adjustedScore
			reasonings[agentID] = ta.generateReasoning(score, adjustedScore, model, task)
		}

		logger.DebugContext(ctx, "Agent scored for task",
			"agent_id", agentID,
			"task_id", task.ID,
			"raw_score", score,
			"adjusted_score", adjustedScore)
	}

	if len(scores) == 0 {
		logger.WarnContext(ctx, "No suitable agents found for task", "task_id", task.ID)
		return nil, nil
	}

	// Select best agent
	bestAgentID, bestScore := ta.selectBestAgent(scores)

	// Create assignment
	assignment := &TaskAssignment{
		TaskID:        task.ID,
		AgentID:       bestAgentID,
		Score:         bestScore,
		Confidence:    ta.calculateConfidence(bestScore, scores),
		Reasoning:     reasonings[bestAgentID],
		EstimatedTime: ta.estimateTaskTime(task, agentModels[bestAgentID]),
		AssignedAt:    time.Now(),
	}

	return assignment, nil
}

// topologicalSort sorts tasks based on dependencies and priority
func (ta *TaskAssigner) topologicalSort(ctx context.Context, tasks []*kanban.Task) ([]*kanban.Task, error) {
	// Create a map for quick lookup
	taskMap := make(map[string]*kanban.Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjacencyList := make(map[string][]string)

	for _, task := range tasks {
		inDegree[task.ID] = 0
		adjacencyList[task.ID] = []string{}
	}

	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if _, exists := taskMap[depID]; exists {
				adjacencyList[depID] = append(adjacencyList[depID], task.ID)
				inDegree[task.ID]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	var result []*kanban.Task
	queue := make([]*kanban.Task, 0)

	// Start with tasks that have no dependencies
	for _, task := range tasks {
		if inDegree[task.ID] == 0 {
			queue = append(queue, task)
		}
	}

	// Sort queue by priority
	sort.Slice(queue, func(i, j int) bool {
		return ta.getTaskPriorityValue(queue[i]) > ta.getTaskPriorityValue(queue[j])
	})

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Update dependencies
		for _, dependentID := range adjacencyList[current.ID] {
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				dependent := taskMap[dependentID]
				// Insert in priority order
				inserted := false
				for i, task := range queue {
					if ta.getTaskPriorityValue(dependent) > ta.getTaskPriorityValue(task) {
						queue = append(queue[:i], append([]*kanban.Task{dependent}, queue[i:]...)...)
						inserted = true
						break
					}
				}
				if !inserted {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(tasks) {
		return nil, gerror.New(gerror.ErrCodeValidation, "circular dependencies detected in task graph", nil).
			WithComponent("TaskAssigner").
			WithOperation("topologicalSort")
	}

	return result, nil
}

// canAssignToAgent checks if an agent can take another task
func (ta *TaskAssigner) canAssignToAgent(agentID string, model *AgentCapabilityModel, currentAssignments map[string][]TaskAssignment) bool {
	currentTasks := len(currentAssignments[agentID])

	// Check max tasks per agent
	if currentTasks >= ta.constraints.MaxTasksPerAgent {
		return false
	}

	// Check if agent is available
	if model.Availability.Status != "available" {
		return false
	}

	// Check workload ratio
	projectedLoad := model.Availability.CurrentLoad + float64(currentTasks)/float64(model.Availability.MaxConcurrentTasks)
	if projectedLoad > ta.constraints.MaxWorkloadRatio {
		return false
	}

	return true
}

// applyConstraintsToScore applies assignment constraints to adjust the raw score
func (ta *TaskAssigner) applyConstraintsToScore(rawScore float64, agentID string, model *AgentCapabilityModel, currentAssignments map[string][]TaskAssignment) float64 {
	adjustedScore := rawScore

	// Workload balancing - penalize agents with more assignments
	if ta.constraints.BalanceWorkload {
		currentTasks := len(currentAssignments[agentID])
		penalty := float64(currentTasks) * 0.1 // 10% penalty per current task
		adjustedScore -= penalty
	}

	// Cost optimization - prefer cheaper agents if scores are close
	if ta.constraints.CostOptimization && model.Config != nil {
		costMagnitude := model.Config.GetEffectiveCostMagnitude()
		// Higher cost = lower score adjustment
		costPenalty := float64(costMagnitude) * 0.05 // 5% penalty per cost magnitude point
		adjustedScore -= costPenalty
	}

	// Specialist preference - boost specialists for their domains
	if ta.constraints.PreferSpecialists {
		// Check if agent has high proficiency in relevant specialties
		hasSpecialty := false
		for _, metrics := range model.Specialties {
			if metrics.Proficiency > 0.7 && metrics.SuccessRate > 0.7 {
				hasSpecialty = true
				break
			}
		}
		if hasSpecialty {
			adjustedScore += 0.1 // 10% boost for specialists
		}
	}

	// Ensure score doesn't go below 0
	if adjustedScore < 0 {
		adjustedScore = 0
	}

	return adjustedScore
}

// selectBestAgent selects the agent with the highest score
func (ta *TaskAssigner) selectBestAgent(scores map[string]float64) (string, float64) {
	var bestAgent string
	var bestScore float64

	for agentID, score := range scores {
		if score > bestScore {
			bestAgent = agentID
			bestScore = score
		}
	}

	return bestAgent, bestScore
}

// calculateConfidence calculates confidence in the assignment
func (ta *TaskAssigner) calculateConfidence(bestScore float64, allScores map[string]float64) float64 {
	if len(allScores) <= 1 {
		return bestScore
	}

	// Find second best score
	var secondBest float64
	for _, score := range allScores {
		if score > secondBest && score < bestScore {
			secondBest = score
		}
	}

	// Confidence is based on the gap between best and second best
	gap := bestScore - secondBest
	confidence := bestScore + (gap * 0.5) // Boost confidence with larger gaps

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateReasoning creates a human-readable explanation for the assignment
func (ta *TaskAssigner) generateReasoning(rawScore, adjustedScore float64, model *AgentCapabilityModel, task *kanban.Task) string {
	reasoning := ""

	if rawScore >= 0.8 {
		reasoning = "Excellent capability match"
	} else if rawScore >= 0.6 {
		reasoning = "Good capability match"
	} else if rawScore >= 0.4 {
		reasoning = "Adequate capability match"
	} else {
		reasoning = "Basic capability match"
	}

	if model.Availability.CurrentLoad < 0.3 {
		reasoning += ", low workload"
	} else if model.Availability.CurrentLoad > 0.7 {
		reasoning += ", high workload"
	}

	if model.Performance.SuccessRate > 0.8 {
		reasoning += ", strong track record"
	}

	if adjustedScore != rawScore {
		if adjustedScore > rawScore {
			reasoning += ", constraint bonus applied"
		} else {
			reasoning += ", constraint penalty applied"
		}
	}

	return reasoning
}

// estimateTaskTime estimates how long a task will take for an agent
func (ta *TaskAssigner) estimateTaskTime(task *kanban.Task, model *AgentCapabilityModel) time.Duration {
	// Start with task's estimated hours
	if task.EstimatedHours > 0 {
		baseTime := time.Duration(task.EstimatedHours * float64(time.Hour))

		// Adjust based on agent's historical performance
		if model.Performance.TasksCompleted > 0 && model.Performance.AverageTime > 0 {
			// Use average time as a factor
			avgHours := model.Performance.AverageTime.Hours()
			if avgHours > 0 {
				factor := avgHours / task.EstimatedHours
				// Clamp factor to reasonable range
				if factor > 2.0 {
					factor = 2.0
				} else if factor < 0.5 {
					factor = 0.5
				}
				return time.Duration(float64(baseTime) * factor)
			}
		}

		return baseTime
	}

	// Fallback to agent's average time
	if model.Performance.AverageTime > 0 {
		return model.Performance.AverageTime
	}

	// Default estimate
	return 4 * time.Hour
}

// getTaskPriorityValue converts task priority to numeric value for sorting
func (ta *TaskAssigner) getTaskPriorityValue(task *kanban.Task) int {
	switch task.Priority {
	case kanban.PriorityHigh:
		return 3
	case kanban.PriorityMedium:
		return 2
	case kanban.PriorityLow:
		return 1
	default:
		return 1
	}
}

// identifyOverloadedAgents identifies agents that are approaching capacity
func (ta *TaskAssigner) identifyOverloadedAgents(agentModels map[string]*AgentCapabilityModel, assignments map[string][]TaskAssignment) []string {
	var overloaded []string

	for agentID, model := range agentModels {
		currentAssignments := len(assignments[agentID])
		projectedLoad := model.Availability.CurrentLoad + float64(currentAssignments)/float64(model.Availability.MaxConcurrentTasks)

		if projectedLoad > ta.constraints.MaxWorkloadRatio {
			overloaded = append(overloaded, agentID)
		}
	}

	return overloaded
}

// calculatePlanScore calculates an overall quality score for the assignment plan
func (ta *TaskAssigner) calculatePlanScore(assignments []TaskAssignment) float64 {
	if len(assignments) == 0 {
		return 0.0
	}

	totalScore := 0.0
	totalConfidence := 0.0

	for _, assignment := range assignments {
		totalScore += assignment.Score
		totalConfidence += assignment.Confidence
	}

	// Average score weighted by confidence
	avgScore := totalScore / float64(len(assignments))
	avgConfidence := totalConfidence / float64(len(assignments))

	return (avgScore + avgConfidence) / 2.0
}

// getAppliedConstraints returns a list of constraints that were applied
func (ta *TaskAssigner) getAppliedConstraints() []string {
	var applied []string

	if ta.constraints.MaxTasksPerAgent > 0 {
		applied = append(applied, "max_tasks_per_agent")
	}
	if ta.constraints.PreferSpecialists {
		applied = append(applied, "prefer_specialists")
	}
	if ta.constraints.BalanceWorkload {
		applied = append(applied, "balance_workload")
	}
	if ta.constraints.CostOptimization {
		applied = append(applied, "cost_optimization")
	}
	if ta.constraints.MinScoreThreshold > 0 {
		applied = append(applied, "min_score_threshold")
	}
	if ta.constraints.MaxWorkloadRatio > 0 {
		applied = append(applied, "max_workload_ratio")
	}

	return applied
}
