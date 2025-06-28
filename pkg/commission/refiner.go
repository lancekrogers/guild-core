// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers"
)

// Refiner implements intelligent commission refinement
type Refiner struct {
	parser      *MarkdownParser
	analyzer    *RequirementAnalyzer
	decomposer  *TaskDecomposer
	estimator   *ComplexityEstimator
	llmClient   providers.LLMClient
	agentLoader *config.HierarchicalLoader
}

// RefinedCommission represents a commission that has been refined into actionable tasks
type RefinedCommission struct {
	Original          *Commission               `json:"original"`
	Analysis          *Analysis                 `json:"analysis"`
	Tasks             []*RefinedTask            `json:"tasks"`
	Timeline          *Timeline                 `json:"timeline"`
	Assignments       map[string][]*RefinedTask `json:"assignments"` // agent_id -> tasks
	AgentResources    *AgentResourceSummary     `json:"agent_resources"`
	EstimatedDuration time.Duration             `json:"estimated_duration"`
	TotalComplexity   int                       `json:"total_complexity"`
	RefinedAt         time.Time                 `json:"refined_at"`
}

// RefinedTask represents a task with intelligent breakdown and assignment
type RefinedTask struct {
	ID             string            `json:"id"`
	CommissionID   string            `json:"commission_id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Type           string            `json:"type"` // design, implementation, testing, documentation
	Status         string            `json:"status"`
	Complexity     int               `json:"complexity"` // 1-8 scale
	AssignedAgent  string            `json:"assigned_agent"`
	EstimatedHours float64           `json:"estimated_hours"`
	Dependencies   []string          `json:"dependencies"`
	Prerequisites  []string          `json:"prerequisites"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Analysis represents the analyzed commission requirements
type Analysis struct {
	Requirements    []Requirement `json:"requirements"`
	TechnicalStack  []string      `json:"technical_stack"`
	Scope           string        `json:"scope"`            // small, medium, large
	EstimatedEffort string        `json:"estimated_effort"` // days/weeks
	RiskFactors     []string      `json:"risk_factors"`
	SuccessCriteria []string      `json:"success_criteria"`
	KeyDeliverables []string      `json:"key_deliverables"`
}

// Requirement represents a parsed requirement
type Requirement struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`     // functional, non-functional, technical
	Priority    string            `json:"priority"` // high, medium, low
	Description string            `json:"description"`
	Acceptance  []string          `json:"acceptance_criteria"`
	Metadata    map[string]string `json:"metadata"`
}

// Timeline represents the project timeline
type Timeline struct {
	StartDate    time.Time   `json:"start_date"`
	EndDate      time.Time   `json:"end_date"`
	Milestones   []Milestone `json:"milestones"`
	CriticalPath []string    `json:"critical_path"` // Task IDs
	BufferDays   int         `json:"buffer_days"`
}

// Milestone represents a project milestone
type Milestone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TargetDate  time.Time `json:"target_date"`
	TaskIDs     []string  `json:"task_ids"`
}

// AgentResourceSummary provides information about available agents for the manager
type AgentResourceSummary struct {
	AvailableAgents []AgentResource `json:"available_agents"`
	TotalAgents     int             `json:"total_agents"`
	TotalCapacity   float64         `json:"total_capacity"` // hours/week
	CostRange       string          `json:"cost_range"`
	GeneratedAt     time.Time       `json:"generated_at"`
}

// AgentResource represents an agent's capabilities for assignment decisions
type AgentResource struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Specialty     string   `json:"specialty"`
	Capabilities  []string `json:"capabilities"`
	CostMagnitude int      `json:"cost_magnitude"`
	Languages     []string `json:"languages"`
	Frameworks    []string `json:"frameworks"`
	Availability  string   `json:"availability"` // available, busy, overloaded
}

// NewRefiner creates a new commission refiner
func NewRefiner(ctx context.Context, llmClient providers.LLMClient, agentConfigPath string) (*Refiner, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.refiner").
			WithOperation("NewRefiner")
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating commission refiner",
		"agent_config_path", agentConfigPath)

	// Create parser with default options
	parser := NewMarkdownParser(DefaultParseOptions())

	// Create requirement analyzer
	analyzer := NewRequirementAnalyzer(llmClient)

	// Create task decomposer
	decomposer := NewTaskDecomposer(llmClient)

	// Create complexity estimator
	estimator := NewComplexityEstimator()

	// Load agent configurations for resource analysis
	agentLoader := config.NewHierarchicalLoader()

	return &Refiner{
		parser:      parser,
		analyzer:    analyzer,
		decomposer:  decomposer,
		estimator:   estimator,
		llmClient:   llmClient,
		agentLoader: agentLoader,
	}, nil
}

// Refine refines a commission into actionable tasks with intelligent agent assignment
func (r *Refiner) Refine(ctx context.Context, commission *Commission) (*RefinedCommission, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.refiner").
			WithOperation("Refine")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "commission.refiner")
	ctx = observability.WithOperation(ctx, "Refine")

	startTime := time.Now()
	logger.InfoContext(ctx, "Starting commission refinement",
		"commission_id", commission.ID,
		"commission_title", commission.Title)

	// 1. Parse structure if not already done
	var parsedCommission *Commission
	if commission.Content != "" {
		var err error
		parsedCommission, err = r.parser.Parse(commission.Content, commission.Source)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse commission content").
				WithComponent("commission.refiner").
				WithOperation("Refine").
				WithDetails("commission_id", commission.ID)
		}
		// Merge parsed data with original
		parsedCommission.ID = commission.ID
		parsedCommission.CampaignID = commission.CampaignID
	} else {
		parsedCommission = commission
	}

	// 2. Analyze requirements using AI
	logger.DebugContext(ctx, "Analyzing commission requirements")
	analysis, err := r.analyzer.Analyze(ctx, parsedCommission)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to analyze commission requirements").
			WithComponent("commission.refiner").
			WithOperation("Refine").
			WithDetails("commission_id", commission.ID)
	}

	// 3. Load agent resources for intelligent assignment
	logger.DebugContext(ctx, "Loading agent resources")
	agentResources, err := r.loadAgentResources(ctx)
	if err != nil {
		logger.WarnContext(ctx, "Failed to load agent resources, continuing without intelligent assignment",
			"error", err.Error())
		agentResources = &AgentResourceSummary{
			AvailableAgents: []AgentResource{},
			TotalAgents:     0,
			GeneratedAt:     time.Now(),
		}
	}

	// 4. Decompose into tasks with agent context
	logger.DebugContext(ctx, "Decomposing into tasks")
	tasks, err := r.decomposer.Decompose(ctx, analysis, agentResources)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decompose commission into tasks").
			WithComponent("commission.refiner").
			WithOperation("Refine").
			WithDetails("commission_id", commission.ID)
	}

	// 5. Estimate complexity for each task
	logger.DebugContext(ctx, "Estimating task complexity")
	for i := range tasks {
		complexity := r.estimator.Estimate(tasks[i])
		tasks[i].Complexity = complexity

		// Estimate hours based on complexity
		tasks[i].EstimatedHours = r.estimateHours(complexity, tasks[i].Type)

		// Set metadata
		tasks[i].CommissionID = commission.ID
		tasks[i].CreatedAt = time.Now()
		tasks[i].UpdatedAt = time.Now()
	}

	// 6. Create timeline
	logger.DebugContext(ctx, "Creating project timeline")
	timeline := r.createTimeline(tasks)

	// 7. Assign tasks to agents
	logger.DebugContext(ctx, "Assigning tasks to agents")
	assignments := r.assignTasks(ctx, tasks, agentResources)

	// Calculate totals
	totalComplexity := 0
	var totalDuration time.Duration
	for _, task := range tasks {
		totalComplexity += task.Complexity
		totalDuration += time.Duration(task.EstimatedHours * float64(time.Hour))
	}

	refined := &RefinedCommission{
		Original:          parsedCommission,
		Analysis:          analysis,
		Tasks:             tasks,
		Timeline:          timeline,
		Assignments:       assignments,
		AgentResources:    agentResources,
		EstimatedDuration: totalDuration,
		TotalComplexity:   totalComplexity,
		RefinedAt:         time.Now(),
	}

	duration := time.Since(startTime)
	logger.InfoContext(ctx, "Commission refinement completed",
		"commission_id", commission.ID,
		"tasks_generated", len(tasks),
		"agents_assigned", len(assignments),
		"total_complexity", totalComplexity,
		"estimated_duration_hours", totalDuration.Hours(),
		"refinement_duration_ms", duration.Milliseconds())

	return refined, nil
}

// createTimeline creates a project timeline from tasks
func (r *Refiner) createTimeline(tasks []*RefinedTask) *Timeline {
	startDate := time.Now()

	// Calculate end date based on task complexity and dependencies
	totalHours := 0.0
	for _, task := range tasks {
		totalHours += task.EstimatedHours
	}

	// Assume 6 productive hours per day, with some buffer
	workingDays := int(totalHours/6) + 2 // 2-day buffer
	endDate := startDate.AddDate(0, 0, workingDays)

	// Create milestones based on task types
	milestones := r.createMilestones(tasks, startDate, endDate)

	// Identify critical path (simplified - longest dependency chain)
	criticalPath := r.identifyCriticalPath(tasks)

	return &Timeline{
		StartDate:    startDate,
		EndDate:      endDate,
		Milestones:   milestones,
		CriticalPath: criticalPath,
		BufferDays:   2,
	}
}

// createMilestones creates project milestones
func (r *Refiner) createMilestones(tasks []*RefinedTask, startDate, endDate time.Time) []Milestone {
	milestones := []Milestone{}

	// Group tasks by type for milestone creation
	tasksByType := make(map[string][]*RefinedTask)
	for _, task := range tasks {
		tasksByType[task.Type] = append(tasksByType[task.Type], task)
	}

	milestoneOrder := []string{"design", "implementation", "testing", "documentation"}
	currentDate := startDate

	for i, taskType := range milestoneOrder {
		if typeTasks, exists := tasksByType[taskType]; exists {
			taskIDs := make([]string, len(typeTasks))
			for j, task := range typeTasks {
				taskIDs[j] = task.ID
			}

			// Calculate milestone date
			totalHours := 0.0
			for _, task := range typeTasks {
				totalHours += task.EstimatedHours
			}
			days := int(totalHours / 6)
			if days == 0 {
				days = 1
			}

			milestoneDate := currentDate.AddDate(0, 0, days)
			if milestoneDate.After(endDate) {
				milestoneDate = endDate
			}

			milestone := Milestone{
				ID:          fmt.Sprintf("milestone-%d", i+1),
				Name:        fmt.Sprintf("%s Complete", capitalize(taskType)),
				Description: fmt.Sprintf("All %s tasks completed", taskType),
				TargetDate:  milestoneDate,
				TaskIDs:     taskIDs,
			}

			milestones = append(milestones, milestone)
			currentDate = milestoneDate
		}
	}

	return milestones
}

// identifyCriticalPath identifies the critical path through tasks
func (r *Refiner) identifyCriticalPath(tasks []*RefinedTask) []string {
	// Simplified critical path - return tasks with most dependencies
	var criticalTasks []*RefinedTask
	maxDeps := 0

	for _, task := range tasks {
		if len(task.Dependencies) > maxDeps {
			maxDeps = len(task.Dependencies)
			criticalTasks = []*RefinedTask{task}
		} else if len(task.Dependencies) == maxDeps && maxDeps > 0 {
			criticalTasks = append(criticalTasks, task)
		}
	}

	path := make([]string, len(criticalTasks))
	for i, task := range criticalTasks {
		path[i] = task.ID
	}

	return path
}

// assignTasks assigns tasks to agents based on capabilities and cost optimization
func (r *Refiner) assignTasks(ctx context.Context, tasks []*RefinedTask, resources *AgentResourceSummary) map[string][]*RefinedTask {
	assignments := make(map[string][]*RefinedTask)

	// If no agents available, return empty assignments
	if len(resources.AvailableAgents) == 0 {
		return assignments
	}

	logger := observability.GetLogger(ctx)

	// Create agent capability map for efficient lookup
	agentCapabilities := make(map[string][]string)
	agentCosts := make(map[string]int)
	for _, agent := range resources.AvailableAgents {
		agentCapabilities[agent.ID] = agent.Capabilities
		agentCosts[agent.ID] = agent.CostMagnitude
	}

	// Assign tasks using intelligent matching
	for _, task := range tasks {
		bestAgent := r.findBestAgent(task, resources.AvailableAgents)
		if bestAgent != "" {
			task.AssignedAgent = bestAgent
			assignments[bestAgent] = append(assignments[bestAgent], task)

			logger.DebugContext(ctx, "Assigned task to agent",
				"task_id", task.ID,
				"task_title", task.Title,
				"agent_id", bestAgent,
				"task_complexity", task.Complexity)
		} else {
			// Fallback to first available agent
			if len(resources.AvailableAgents) > 0 {
				fallbackAgent := resources.AvailableAgents[0].ID
				task.AssignedAgent = fallbackAgent
				assignments[fallbackAgent] = append(assignments[fallbackAgent], task)

				logger.WarnContext(ctx, "No perfect match found, using fallback agent",
					"task_id", task.ID,
					"fallback_agent", fallbackAgent)
			}
		}
	}

	return assignments
}

// findBestAgent finds the best agent for a task based on capabilities and cost
func (r *Refiner) findBestAgent(task *RefinedTask, agents []AgentResource) string {
	var bestAgent string
	bestScore := -1

	for _, agent := range agents {
		score := r.calculateAgentScore(task, agent)
		if score > bestScore {
			bestScore = score
			bestAgent = agent.ID
		}
	}

	return bestAgent
}

// calculateAgentScore calculates how well an agent matches a task
func (r *Refiner) calculateAgentScore(task *RefinedTask, agent AgentResource) int {
	score := 0

	// Base score for agent type matching task type
	switch task.Type {
	case "design":
		if agent.Type == "specialist" || agent.Type == "manager" {
			score += 10
		}
	case "implementation":
		if agent.Type == "worker" || agent.Type == "specialist" {
			score += 10
		}
	case "testing":
		if agent.Type == "worker" {
			score += 10
		}
	case "documentation":
		if agent.Type == "worker" {
			score += 8
		}
	}

	// Bonus for relevant capabilities
	taskKeywords := extractTaskKeywords(task.Title + " " + task.Description)
	for _, capability := range agent.Capabilities {
		for _, keyword := range taskKeywords {
			if containsIgnoreCase(capability, keyword) {
				score += 5
			}
		}
	}

	// Cost efficiency bonus (prefer lower cost for simpler tasks)
	if task.Complexity <= 3 && agent.CostMagnitude <= 2 {
		score += 3
	} else if task.Complexity >= 6 && agent.CostMagnitude >= 3 {
		score += 2
	}

	// Availability bonus
	switch agent.Availability {
	case "available":
		score += 5
	case "busy":
		score += 0
	case "overloaded":
		score -= 5
	}

	return score
}

// estimateHours estimates hours based on complexity and task type
func (r *Refiner) estimateHours(complexity int, taskType string) float64 {
	baseHours := map[string]float64{
		"design":         2.0,
		"implementation": 4.0,
		"testing":        3.0,
		"documentation":  1.5,
	}

	base, exists := baseHours[taskType]
	if !exists {
		base = 3.0 // Default
	}

	// Scale by complexity (1-8 Fibonacci scale)
	multiplier := map[int]float64{
		1: 0.5, // Very simple
		2: 1.0, // Simple
		3: 1.5, // Medium-simple
		5: 2.5, // Medium-complex
		8: 4.0, // Very complex
	}

	mult, exists := multiplier[complexity]
	if !exists {
		mult = 1.0 // Default
	}

	return base * mult
}

// loadAgentResources loads available agent configurations for resource planning
func (r *Refiner) loadAgentResources(ctx context.Context) (*AgentResourceSummary, error) {
	// This would load from the agent configuration directory
	// For now, return a basic structure
	// TODO: Implement actual agent config loading

	agents := []AgentResource{
		{
			ID:            "elena-guild-master",
			Name:          "Elena",
			Type:          "manager",
			Specialty:     "project management",
			Capabilities:  []string{"planning", "coordination", "analysis"},
			CostMagnitude: 3,
			Availability:  "available",
		},
		{
			ID:            "marcus-code-artisan",
			Name:          "Marcus",
			Type:          "worker",
			Specialty:     "software development",
			Capabilities:  []string{"coding", "implementation", "debugging"},
			CostMagnitude: 2,
			Languages:     []string{"go", "javascript", "python"},
			Availability:  "available",
		},
		{
			ID:            "vera-test-guardian",
			Name:          "Vera",
			Type:          "specialist",
			Specialty:     "quality assurance",
			Capabilities:  []string{"testing", "validation", "quality"},
			CostMagnitude: 2,
			Availability:  "available",
		},
	}

	return &AgentResourceSummary{
		AvailableAgents: agents,
		TotalAgents:     len(agents),
		TotalCapacity:   40.0 * float64(len(agents)), // 40 hours/week per agent
		CostRange:       "1-5 magnitude",
		GeneratedAt:     time.Now(),
	}, nil
}

// Utility functions

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func extractTaskKeywords(text string) []string {
	// Simplified keyword extraction
	// In production, this could use NLP libraries
	keywords := []string{}
	commonWords := []string{"api", "endpoint", "database", "ui", "frontend", "backend", "test", "auth", "security"}

	textLower := strings.ToLower(text)
	for _, word := range commonWords {
		if strings.Contains(textLower, word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
