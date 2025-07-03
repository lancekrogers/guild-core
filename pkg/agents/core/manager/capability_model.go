// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
)

// AgentCapabilityModel represents the intelligence profile of an agent
type AgentCapabilityModel struct {
	AgentID      string                      `json:"agent_id"`
	Capabilities map[string]float64          `json:"capabilities"` // capability -> proficiency (0.0-1.0)
	Performance  PerformanceMetrics          `json:"performance"`
	Availability AgentAvailability           `json:"availability"`
	Specialties  map[string]SpecialtyMetrics `json:"specialties"`
	Config       *config.EnhancedAgentConfig `json:"config,omitempty"`
}

// PerformanceMetrics tracks historical performance data
type PerformanceMetrics struct {
	TasksCompleted   int           `json:"tasks_completed"`
	AverageTime      time.Duration `json:"average_time"`
	SuccessRate      float64       `json:"success_rate"`      // 0.0-1.0
	ComplexityHandle float64       `json:"complexity_handle"` // avg complexity handled (1-10)
	LastUpdated      time.Time     `json:"last_updated"`
}

// SpecialtyMetrics tracks performance in specific domains
type SpecialtyMetrics struct {
	Proficiency     float64   `json:"proficiency"` // 0.0-1.0
	TasksCompleted  int       `json:"tasks_completed"`
	SuccessRate     float64   `json:"success_rate"` // 0.0-1.0
	LastImprovement time.Time `json:"last_improvement"`
}

// AgentAvailability tracks current workload and capacity
type AgentAvailability struct {
	CurrentLoad        float64   `json:"current_load"` // 0.0-1.0
	ActiveTasks        int       `json:"active_tasks"`
	MaxConcurrentTasks int       `json:"max_concurrent_tasks"`
	LastAssignment     time.Time `json:"last_assignment"`
	Status             string    `json:"status"` // available, busy, offline
}

// TaskRequirements represents what a task needs from an agent
type TaskRequirements struct {
	RequiredCapabilities []string   `json:"required_capabilities"`
	PreferredSpecialties []string   `json:"preferred_specialties"`
	Complexity           int        `json:"complexity"` // 1-10
	Priority             string     `json:"priority"`   // high, medium, low
	EstimatedHours       float64    `json:"estimated_hours"`
	Deadline             *time.Time `json:"deadline,omitempty"`
	TaskType             string     `json:"task_type"` // coding, testing, documentation, etc.
}

// CapabilityModelManager manages agent capability models
type CapabilityModelManager struct {
	db     *sql.DB
	models map[string]*AgentCapabilityModel // agentID -> model
}

// NewCapabilityModelManager creates a new capability model manager
func NewCapabilityModelManager(db *sql.DB) *CapabilityModelManager {
	return &CapabilityModelManager{
		db:     db,
		models: make(map[string]*AgentCapabilityModel),
	}
}

// LoadAgentModel loads or creates an agent capability model
func (cmm *CapabilityModelManager) LoadAgentModel(ctx context.Context, agentID string) (*AgentCapabilityModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CapabilityModelManager").
			WithOperation("LoadAgentModel")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "CapabilityModelManager")
	ctx = observability.WithOperation(ctx, "LoadAgentModel")

	// Check cache first
	if model, exists := cmm.models[agentID]; exists {
		logger.DebugContext(ctx, "Loaded agent model from cache", "agent_id", agentID)
		return model, nil
	}

	// Load from database
	model, err := cmm.loadFromDatabase(ctx, agentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load agent model from database").
			WithComponent("CapabilityModelManager").
			WithOperation("LoadAgentModel").
			WithDetails("agent_id", agentID)
	}

	// Cache the model
	cmm.models[agentID] = model

	logger.InfoContext(ctx, "Loaded agent capability model",
		"agent_id", agentID,
		"capabilities_count", len(model.Capabilities),
		"specialties_count", len(model.Specialties),
		"tasks_completed", model.Performance.TasksCompleted)

	return model, nil
}

// ScoreForTask calculates how well-suited an agent is for a specific task
func (acm *AgentCapabilityModel) ScoreForTask(ctx context.Context, task *kanban.Task) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentCapabilityModel").
			WithOperation("ScoreForTask")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "AgentCapabilityModel")
	ctx = observability.WithOperation(ctx, "ScoreForTask")

	// Extract requirements from task
	requirements := acm.extractTaskRequirements(task)

	score := 0.0
	maxScore := 1.0

	logger.DebugContext(ctx, "Calculating task score",
		"agent_id", acm.AgentID,
		"task_id", task.ID,
		"task_title", task.Title,
		"required_capabilities", len(requirements.RequiredCapabilities))

	// 1. Capability matching (40% weight)
	capabilityScore := acm.calculateCapabilityScore(requirements)
	score += capabilityScore * 0.4

	// 2. Workload consideration (20% weight)
	workloadScore := acm.calculateWorkloadScore()
	score += workloadScore * 0.2

	// 3. Specialty matching (25% weight)
	specialtyScore := acm.calculateSpecialtyScore(requirements)
	score += specialtyScore * 0.25

	// 4. Complexity matching (15% weight)
	complexityScore := acm.calculateComplexityScore(requirements)
	score += complexityScore * 0.15

	// Normalize to 0-1 range
	normalizedScore := score / maxScore

	logger.DebugContext(ctx, "Task scoring completed",
		"agent_id", acm.AgentID,
		"task_id", task.ID,
		"capability_score", capabilityScore,
		"workload_score", workloadScore,
		"specialty_score", specialtyScore,
		"complexity_score", complexityScore,
		"final_score", normalizedScore)

	return normalizedScore, nil
}

// UpdatePerformance updates the agent's performance metrics based on task completion
func (acm *AgentCapabilityModel) UpdatePerformance(ctx context.Context, taskID string, success bool, duration time.Duration, complexity int) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentCapabilityModel").
			WithOperation("UpdatePerformance")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "AgentCapabilityModel")
	ctx = observability.WithOperation(ctx, "UpdatePerformance")

	// Update task completion count
	acm.Performance.TasksCompleted++

	// Update average time (exponential moving average)
	if acm.Performance.AverageTime == 0 {
		acm.Performance.AverageTime = duration
	} else {
		alpha := 0.3 // smoothing factor
		newAvg := time.Duration(float64(acm.Performance.AverageTime)*(1-alpha) + float64(duration)*alpha)
		acm.Performance.AverageTime = newAvg
	}

	// Update success rate (exponential moving average)
	successValue := 0.0
	if success {
		successValue = 1.0
	}

	if acm.Performance.TasksCompleted == 1 {
		acm.Performance.SuccessRate = successValue
	} else {
		alpha := 0.3
		acm.Performance.SuccessRate = acm.Performance.SuccessRate*(1-alpha) + successValue*alpha
	}

	// Update complexity handling (exponential moving average)
	complexityValue := float64(complexity)
	if acm.Performance.TasksCompleted == 1 {
		acm.Performance.ComplexityHandle = complexityValue
	} else {
		alpha := 0.3
		acm.Performance.ComplexityHandle = acm.Performance.ComplexityHandle*(1-alpha) + complexityValue*alpha
	}

	acm.Performance.LastUpdated = time.Now()

	logger.InfoContext(ctx, "Updated agent performance metrics",
		"agent_id", acm.AgentID,
		"task_id", taskID,
		"success", success,
		"duration_hours", duration.Hours(),
		"complexity", complexity,
		"new_success_rate", acm.Performance.SuccessRate,
		"new_avg_time_hours", acm.Performance.AverageTime.Hours(),
		"new_complexity_handle", acm.Performance.ComplexityHandle)

	return nil
}

// extractTaskRequirements extracts requirements from a kanban task
func (acm *AgentCapabilityModel) extractTaskRequirements(task *kanban.Task) TaskRequirements {
	requirements := TaskRequirements{
		RequiredCapabilities: []string{},
		PreferredSpecialties: []string{},
		Complexity:           1,
		Priority:             string(task.Priority),
		EstimatedHours:       task.EstimatedHours,
		Deadline:             task.DueDate,
		TaskType:             "general",
	}

	// Extract from tags
	for _, tag := range task.Tags {
		tagLower := strings.ToLower(tag)

		// Capability tags
		if strings.HasPrefix(tagLower, "cap:") {
			cap := strings.TrimPrefix(tagLower, "cap:")
			requirements.RequiredCapabilities = append(requirements.RequiredCapabilities, cap)
		}

		// Specialty tags
		if strings.HasPrefix(tagLower, "spec:") {
			spec := strings.TrimPrefix(tagLower, "spec:")
			requirements.PreferredSpecialties = append(requirements.PreferredSpecialties, spec)
		}

		// Task type inference
		switch tagLower {
		case "backend", "api", "database":
			requirements.TaskType = "backend"
		case "frontend", "ui", "ux":
			requirements.TaskType = "frontend"
		case "testing", "qa", "validation":
			requirements.TaskType = "testing"
		case "documentation", "docs":
			requirements.TaskType = "documentation"
		case "devops", "deployment", "infrastructure":
			requirements.TaskType = "devops"
		}
	}

	// Extract from metadata
	if complexityStr, exists := task.Metadata["complexity"]; exists {
		if complexity := parseComplexity(complexityStr); complexity > 0 {
			requirements.Complexity = complexity
		}
	}

	// Extract from description patterns
	descLower := strings.ToLower(task.Description)
	if strings.Contains(descLower, "test") || strings.Contains(descLower, "validate") {
		requirements.RequiredCapabilities = append(requirements.RequiredCapabilities, "testing")
	}
	if strings.Contains(descLower, "api") || strings.Contains(descLower, "backend") {
		requirements.RequiredCapabilities = append(requirements.RequiredCapabilities, "backend")
	}
	if strings.Contains(descLower, "ui") || strings.Contains(descLower, "frontend") {
		requirements.RequiredCapabilities = append(requirements.RequiredCapabilities, "frontend")
	}

	return requirements
}

// calculateCapabilityScore calculates how well the agent matches required capabilities
func (acm *AgentCapabilityModel) calculateCapabilityScore(requirements TaskRequirements) float64 {
	if len(requirements.RequiredCapabilities) == 0 {
		return 0.5 // neutral score for tasks with no specific requirements
	}

	totalScore := 0.0
	matchedCapabilities := 0

	for _, reqCap := range requirements.RequiredCapabilities {
		if proficiency, exists := acm.Capabilities[reqCap]; exists {
			totalScore += proficiency
			matchedCapabilities++
		}
	}

	if matchedCapabilities == 0 {
		return 0.1 // low score if no capabilities match
	}

	return totalScore / float64(matchedCapabilities)
}

// calculateWorkloadScore calculates score based on current workload (higher load = lower score)
func (acm *AgentCapabilityModel) calculateWorkloadScore() float64 {
	// Invert current load so that lower load = higher score
	return 1.0 - acm.Availability.CurrentLoad
}

// calculateSpecialtyScore calculates how well the agent matches preferred specialties
func (acm *AgentCapabilityModel) calculateSpecialtyScore(requirements TaskRequirements) float64 {
	if len(requirements.PreferredSpecialties) == 0 {
		return 0.5 // neutral score for tasks with no specialty preferences
	}

	totalScore := 0.0
	matchedSpecialties := 0

	for _, reqSpec := range requirements.PreferredSpecialties {
		if specialty, exists := acm.Specialties[reqSpec]; exists {
			// Weight by both proficiency and success rate
			score := (specialty.Proficiency + specialty.SuccessRate) / 2.0
			totalScore += score
			matchedSpecialties++
		}
	}

	if matchedSpecialties == 0 {
		return 0.2 // low score if no specialties match
	}

	return totalScore / float64(matchedSpecialties)
}

// calculateComplexityScore calculates how well the agent matches the task complexity
func (acm *AgentCapabilityModel) calculateComplexityScore(requirements TaskRequirements) float64 {
	complexityDiff := math.Abs(float64(requirements.Complexity) - acm.Performance.ComplexityHandle)

	// Normalize the difference (0 = perfect match, 9 = maximum mismatch)
	normalizedDiff := complexityDiff / 9.0

	// Convert to score (0 diff = 1.0 score, max diff = 0.1 score)
	return 1.0 - (normalizedDiff * 0.9)
}

// loadFromDatabase loads agent model from the database
func (cmm *CapabilityModelManager) loadFromDatabase(ctx context.Context, agentID string) (*AgentCapabilityModel, error) {
	model := &AgentCapabilityModel{
		AgentID:      agentID,
		Capabilities: make(map[string]float64),
		Specialties:  make(map[string]SpecialtyMetrics),
	}

	// Load performance metrics
	err := cmm.loadPerformanceMetrics(ctx, model)
	if err != nil {
		return nil, err
	}

	// Load capabilities
	err = cmm.loadCapabilities(ctx, model)
	if err != nil {
		return nil, err
	}

	// Load specialties
	err = cmm.loadSpecialties(ctx, model)
	if err != nil {
		return nil, err
	}

	// Load availability
	err = cmm.loadAvailability(ctx, model)
	if err != nil {
		return nil, err
	}

	return model, nil
}

// loadPerformanceMetrics loads performance data from database
func (cmm *CapabilityModelManager) loadPerformanceMetrics(ctx context.Context, model *AgentCapabilityModel) error {
	query := `
		SELECT tasks_completed, average_time_hours, success_rate, complexity_handle, updated_at
		FROM agent_performance 
		WHERE agent_id = ?`

	var avgTimeHours float64
	var updatedAt time.Time

	err := cmm.db.QueryRowContext(ctx, query, model.AgentID).Scan(
		&model.Performance.TasksCompleted,
		&avgTimeHours,
		&model.Performance.SuccessRate,
		&model.Performance.ComplexityHandle,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// Initialize default performance metrics
		model.Performance = PerformanceMetrics{
			TasksCompleted:   0,
			AverageTime:      0,
			SuccessRate:      0.5,
			ComplexityHandle: 1.0,
			LastUpdated:      time.Now(),
		}

		// Insert default record
		return cmm.createDefaultPerformanceRecord(ctx, model.AgentID)
	}

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load performance metrics").
			WithComponent("CapabilityModelManager").
			WithOperation("loadPerformanceMetrics").
			WithDetails("agent_id", model.AgentID)
	}

	model.Performance.AverageTime = time.Duration(avgTimeHours * float64(time.Hour))
	model.Performance.LastUpdated = updatedAt

	return nil
}

// loadCapabilities loads capability proficiency data from database
func (cmm *CapabilityModelManager) loadCapabilities(ctx context.Context, model *AgentCapabilityModel) error {
	query := `
		SELECT capability, proficiency
		FROM agent_capabilities 
		WHERE agent_id = ?`

	rows, err := cmm.db.QueryContext(ctx, query, model.AgentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query capabilities").
			WithComponent("CapabilityModelManager").
			WithOperation("loadCapabilities").
			WithDetails("agent_id", model.AgentID)
	}
	defer rows.Close()

	for rows.Next() {
		var capability string
		var proficiency float64

		if err := rows.Scan(&capability, &proficiency); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan capability row").
				WithComponent("CapabilityModelManager").
				WithOperation("loadCapabilities").
				WithDetails("agent_id", model.AgentID)
		}

		model.Capabilities[capability] = proficiency
	}

	return rows.Err()
}

// loadSpecialties loads specialty metrics from database
func (cmm *CapabilityModelManager) loadSpecialties(ctx context.Context, model *AgentCapabilityModel) error {
	query := `
		SELECT specialty, proficiency, tasks_completed, success_rate, updated_at
		FROM agent_specialties 
		WHERE agent_id = ?`

	rows, err := cmm.db.QueryContext(ctx, query, model.AgentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query specialties").
			WithComponent("CapabilityModelManager").
			WithOperation("loadSpecialties").
			WithDetails("agent_id", model.AgentID)
	}
	defer rows.Close()

	for rows.Next() {
		var specialty string
		var metrics SpecialtyMetrics

		if err := rows.Scan(&specialty, &metrics.Proficiency, &metrics.TasksCompleted, &metrics.SuccessRate, &metrics.LastImprovement); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan specialty row").
				WithComponent("CapabilityModelManager").
				WithOperation("loadSpecialties").
				WithDetails("agent_id", model.AgentID)
		}

		model.Specialties[specialty] = metrics
	}

	return rows.Err()
}

// loadAvailability loads availability data from database
func (cmm *CapabilityModelManager) loadAvailability(ctx context.Context, model *AgentCapabilityModel) error {
	query := `
		SELECT current_load, active_tasks, max_concurrent_tasks, last_assignment, status
		FROM agent_availability 
		WHERE agent_id = ?`

	var lastAssignment sql.NullTime

	err := cmm.db.QueryRowContext(ctx, query, model.AgentID).Scan(
		&model.Availability.CurrentLoad,
		&model.Availability.ActiveTasks,
		&model.Availability.MaxConcurrentTasks,
		&lastAssignment,
		&model.Availability.Status,
	)

	if err == sql.ErrNoRows {
		// Initialize default availability
		model.Availability = AgentAvailability{
			CurrentLoad:        0.0,
			ActiveTasks:        0,
			MaxConcurrentTasks: 3,
			Status:             "available",
		}

		// Insert default record
		return cmm.createDefaultAvailabilityRecord(ctx, model.AgentID)
	}

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load availability").
			WithComponent("CapabilityModelManager").
			WithOperation("loadAvailability").
			WithDetails("agent_id", model.AgentID)
	}

	if lastAssignment.Valid {
		model.Availability.LastAssignment = lastAssignment.Time
	}

	return nil
}

// createDefaultPerformanceRecord creates a default performance record for a new agent
func (cmm *CapabilityModelManager) createDefaultPerformanceRecord(ctx context.Context, agentID string) error {
	query := `
		INSERT INTO agent_performance (id, agent_id, tasks_completed, average_time_hours, success_rate, complexity_handle)
		VALUES (?, ?, 0, 0.0, 0.5, 1.0)`

	id := generateID()
	_, err := cmm.db.ExecContext(ctx, query, id, agentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create default performance record").
			WithComponent("CapabilityModelManager").
			WithOperation("createDefaultPerformanceRecord").
			WithDetails("agent_id", agentID)
	}

	return nil
}

// createDefaultAvailabilityRecord creates a default availability record for a new agent
func (cmm *CapabilityModelManager) createDefaultAvailabilityRecord(ctx context.Context, agentID string) error {
	query := `
		INSERT INTO agent_availability (id, agent_id, current_load, active_tasks, max_concurrent_tasks, status)
		VALUES (?, ?, 0.0, 0, 3, 'available')`

	id := generateID()
	_, err := cmm.db.ExecContext(ctx, query, id, agentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create default availability record").
			WithComponent("CapabilityModelManager").
			WithOperation("createDefaultAvailabilityRecord").
			WithDetails("agent_id", agentID)
	}

	return nil
}

// parseComplexity parses complexity from string to int
func parseComplexity(complexityStr string) int {
	switch strings.ToLower(complexityStr) {
	case "1", "trivial", "very easy":
		return 1
	case "2", "easy":
		return 2
	case "3", "simple":
		return 3
	case "4", "medium":
		return 4
	case "5", "moderate":
		return 5
	case "6", "complex":
		return 6
	case "7", "hard":
		return 7
	case "8", "very hard":
		return 8
	case "9", "expert":
		return 9
	case "10", "legendary":
		return 10
	default:
		return 1
	}
}

// generateID generates a unique ID for database records
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
