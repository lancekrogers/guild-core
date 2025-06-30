// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// CostOptimizer provides intelligent cost optimization algorithms
type CostOptimizer struct {
	tracker    *CostTracker
	strategies []OptimizationStrategy
	executor   *OptimizationExecutor
	monitor    *SavingsMonitor
	config     *OptimizerConfig
}

// OptimizationStrategy defines the interface for optimization strategies
type OptimizationStrategy interface {
	Analyze(ctx context.Context, usage []Usage) ([]Optimization, error)
	Priority() int
	Name() string
}

// Optimization represents a cost optimization opportunity
type Optimization struct {
	ID          string                 `json:"id"`
	Type        OptimizationType       `json:"type"`
	Description string                 `json:"description"`
	Savings     float64                `json:"savings"`
	Impact      Impact                 `json:"impact"`
	Actions     []Action               `json:"actions"`
	Confidence  float64                `json:"confidence"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OptimizationType defines types of optimizations
type OptimizationType string

const (
	OptimizationModelSwitch OptimizationType = "model_switch"
	OptimizationCaching     OptimizationType = "caching"
	OptimizationBatching    OptimizationType = "batching"
	OptimizationPooling     OptimizationType = "pooling"
	OptimizationCompression OptimizationType = "compression"
	OptimizationScheduling  OptimizationType = "scheduling"
)

// Impact represents the impact of an optimization
type Impact struct {
	Quality     float64 `json:"quality"`     // -1.0 to 1.0 (negative = degradation, positive = improvement)
	Performance float64 `json:"performance"` // -1.0 to 1.0 (negative = slower, positive = faster)
	Reliability float64 `json:"reliability"` // -1.0 to 1.0 (negative = less reliable, positive = more reliable)
}

// Action represents an optimization action
type Action struct {
	Type        ActionType             `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Rollback    map[string]interface{} `json:"rollback,omitempty"`
}

// ActionType defines types of optimization actions
type ActionType string

const (
	ActionUpdateConfig   ActionType = "update_config"
	ActionEnableCache    ActionType = "enable_cache"
	ActionEnableBatching ActionType = "enable_batching"
	ActionCreatePool     ActionType = "create_pool"
	ActionScheduleTask   ActionType = "schedule_task"
	ActionCompressData   ActionType = "compress_data"
)

// OptimizerConfig contains optimizer configuration
type OptimizerConfig struct {
	MinSavingsThreshold float64            `json:"min_savings_threshold"`
	MaxImpactThreshold  float64            `json:"max_impact_threshold"`
	AnalysisWindow      time.Duration      `json:"analysis_window"`
	EnableAutoApply     bool               `json:"enable_auto_apply"`
	StrategyWeights     map[string]float64 `json:"strategy_weights"`
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer(ctx context.Context, tracker *CostTracker, config *OptimizerConfig) (*CostOptimizer, error) {
	ctx = observability.WithComponent(ctx, "cost.optimizer")
	ctx = observability.WithOperation(ctx, "NewCostOptimizer")

	if tracker == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "tracker cannot be nil", nil).
			WithComponent("cost.optimizer").
			WithOperation("NewCostOptimizer")
	}

	if config == nil {
		config = &OptimizerConfig{
			MinSavingsThreshold: 5.0, // $5 minimum savings
			MaxImpactThreshold:  0.2, // Maximum 20% negative impact
			AnalysisWindow:      24 * time.Hour,
			EnableAutoApply:     false,
			StrategyWeights:     make(map[string]float64),
		}
	}

	executor, err := NewOptimizationExecutor(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create optimization executor").
			WithComponent("cost.optimizer").
			WithOperation("NewCostOptimizer")
	}

	monitor, err := NewSavingsMonitor(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create savings monitor").
			WithComponent("cost.optimizer").
			WithOperation("NewCostOptimizer")
	}

	optimizer := &CostOptimizer{
		tracker:  tracker,
		executor: executor,
		monitor:  monitor,
		config:   config,
	}

	// Register optimization strategies
	optimizer.registerStrategies(ctx)

	return optimizer, nil
}

// registerStrategies registers all optimization strategies
func (co *CostOptimizer) registerStrategies(ctx context.Context) {
	co.strategies = []OptimizationStrategy{
		NewModelOptimizer(ctx, co.tracker),
		NewCacheOptimizer(ctx, co.tracker),
		NewBatchOptimizer(ctx, co.tracker),
		NewResourcePoolOptimizer(ctx, co.tracker),
		NewCompressionOptimizer(ctx, co.tracker),
		NewSchedulingOptimizer(ctx, co.tracker),
	}
}

// Analyze performs comprehensive cost optimization analysis
func (co *CostOptimizer) Analyze(ctx context.Context) ([]Optimization, error) {
	ctx = observability.WithComponent(ctx, "cost.optimizer")
	ctx = observability.WithOperation(ctx, "Analyze")

	// Get usage data for analysis window
	period := TimePeriod{
		Start: time.Now().Add(-co.config.AnalysisWindow),
		End:   time.Now(),
	}

	// This would get usage from storage - for now, we'll simulate
	usage := co.getUsageForAnalysis(ctx, period)

	var allOptimizations []Optimization

	// Run all optimization strategies
	for _, strategy := range co.strategies {
		optimizations, err := strategy.Analyze(ctx, usage)
		if err != nil {
			// Log error but continue with other strategies
			continue
		}

		// Filter optimizations based on thresholds
		for _, opt := range optimizations {
			if co.shouldIncludeOptimization(opt) {
				allOptimizations = append(allOptimizations, opt)
			}
		}
	}

	// Sort optimizations by priority and savings
	co.sortOptimizations(allOptimizations)

	return allOptimizations, nil
}

// shouldIncludeOptimization checks if optimization meets thresholds
func (co *CostOptimizer) shouldIncludeOptimization(opt Optimization) bool {
	// Check minimum savings threshold
	if opt.Savings < co.config.MinSavingsThreshold {
		return false
	}

	// Check maximum negative impact threshold
	maxNegativeImpact := math.Max(
		math.Max(-opt.Impact.Quality, -opt.Impact.Performance),
		-opt.Impact.Reliability,
	)

	if maxNegativeImpact > co.config.MaxImpactThreshold {
		return false
	}

	return true
}

// sortOptimizations sorts optimizations by priority and savings
func (co *CostOptimizer) sortOptimizations(optimizations []Optimization) {
	sort.Slice(optimizations, func(i, j int) bool {
		optI, optJ := optimizations[i], optimizations[j]

		// First by priority (higher is better)
		if optI.Priority != optJ.Priority {
			return optI.Priority > optJ.Priority
		}

		// Then by savings (higher is better)
		return optI.Savings > optJ.Savings
	})
}

// getUsageForAnalysis gets usage data for analysis (simulated for now)
func (co *CostOptimizer) getUsageForAnalysis(ctx context.Context, period TimePeriod) []Usage {
	// In production, this would query the storage for actual usage data
	// For now, return simulated usage
	return []Usage{
		{
			AgentID:   "elena",
			Provider:  "openai",
			Resource:  "completion",
			Quantity:  1000,
			Unit:      "tokens",
			Timestamp: time.Now().Add(-time.Hour),
			Metadata: map[string]interface{}{
				"model":         "gpt-4",
				"input_tokens":  800,
				"output_tokens": 200,
				"total_cost":    0.048,
				"success":       true,
				"task_type":     "code_review",
			},
		},
		{
			AgentID:   "marcus",
			Provider:  "anthropic",
			Resource:  "completion",
			Quantity:  2000,
			Unit:      "tokens",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Metadata: map[string]interface{}{
				"model":         "claude-3-opus-20240229",
				"input_tokens":  1500,
				"output_tokens": 500,
				"total_cost":    0.06,
				"success":       true,
				"task_type":     "documentation",
			},
		},
	}
}

// ModelOptimizer implements model selection optimization
type ModelOptimizer struct {
	tracker     *CostTracker
	performance *PerformanceTracker
	costs       *CostCalculator
}

// NewModelOptimizer creates a new model optimizer
func NewModelOptimizer(ctx context.Context, tracker *CostTracker) *ModelOptimizer {
	return &ModelOptimizer{
		tracker:     tracker,
		performance: NewPerformanceTracker(ctx),
		costs:       &CostCalculator{},
	}
}

// Name returns the strategy name
func (mo *ModelOptimizer) Name() string {
	return "model_optimizer"
}

// Priority returns the strategy priority
func (mo *ModelOptimizer) Priority() int {
	return 100 // High priority
}

// Analyze analyzes model efficiency and suggests optimizations
func (mo *ModelOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	ctx = observability.WithComponent(ctx, "cost.model_optimizer")
	ctx = observability.WithOperation(ctx, "Analyze")

	var optimizations []Optimization

	// Group usage by task type
	taskGroups := mo.groupByTaskType(usage)

	for taskType, usages := range taskGroups {
		analysis := mo.analyzeModelEfficiency(ctx, taskType, usages)

		if analysis.HasOptimization() {
			opt := Optimization{
				ID:          generateOptID(),
				Type:        OptimizationModelSwitch,
				Description: fmt.Sprintf("Switch to %s for %s tasks", analysis.RecommendedModel, taskType),
				Savings:     analysis.EstimatedSavings,
				Impact: Impact{
					Quality:     analysis.QualityImpact,
					Performance: analysis.PerformanceImpact,
					Reliability: analysis.ReliabilityImpact,
				},
				Actions: []Action{
					{
						Type:        ActionUpdateConfig,
						Description: fmt.Sprintf("Update agent configuration to use %s for %s tasks", analysis.RecommendedModel, taskType),
						Config: map[string]interface{}{
							"task_type": taskType,
							"model":     analysis.RecommendedModel,
							"reason":    "cost_optimization",
						},
						Rollback: map[string]interface{}{
							"task_type": taskType,
							"model":     analysis.CurrentModel,
						},
					},
				},
				Confidence: analysis.Confidence,
				Priority:   mo.Priority(),
				CreatedAt:  time.Now(),
				Metadata: map[string]interface{}{
					"task_type":         taskType,
					"current_model":     analysis.CurrentModel,
					"recommended_model": analysis.RecommendedModel,
					"cost_per_task":     analysis.CostPerTask,
				},
			}

			optimizations = append(optimizations, opt)
		}
	}

	return optimizations, nil
}

// groupByTaskType groups usage by task type
func (mo *ModelOptimizer) groupByTaskType(usage []Usage) map[string][]Usage {
	groups := make(map[string][]Usage)

	for _, u := range usage {
		taskType := "general"
		if tt, ok := u.Metadata["task_type"]; ok {
			if ttStr, ok := tt.(string); ok {
				taskType = ttStr
			}
		}

		groups[taskType] = append(groups[taskType], u)
	}

	return groups
}

// analyzeModelEfficiency analyzes model efficiency for a task type
func (mo *ModelOptimizer) analyzeModelEfficiency(ctx context.Context, taskType string, usages []Usage) *ModelAnalysis {
	// Calculate cost per successful task by model
	modelStats := make(map[string]*ModelStat)

	for _, usage := range usages {
		model := "unknown"
		if m, ok := usage.Metadata["model"]; ok {
			if mStr, ok := m.(string); ok {
				model = mStr
			}
		}

		success := true
		if s, ok := usage.Metadata["success"]; ok {
			if sBool, ok := s.(bool); ok {
				success = sBool
			}
		}

		cost := 0.0
		if c, ok := usage.Metadata["total_cost"]; ok {
			if cFloat, ok := c.(float64); ok {
				cost = cFloat
			}
		}

		if _, ok := modelStats[model]; !ok {
			modelStats[model] = &ModelStat{}
		}

		stat := modelStats[model]
		stat.TotalCost += cost
		stat.TaskCount++
		if success {
			stat.SuccessCount++
		}
	}

	// Find most cost-effective model
	var bestModel string
	bestEfficiency := math.MaxFloat64
	currentModel := mo.getCurrentModel(taskType)

	for model, stat := range modelStats {
		if stat.SuccessCount == 0 {
			continue
		}

		efficiency := stat.TotalCost / float64(stat.SuccessCount)
		if efficiency < bestEfficiency {
			bestEfficiency = efficiency
			bestModel = model
		}
	}

	if bestModel == "" || bestModel == currentModel {
		return &ModelAnalysis{HasOptimization: func() bool { return false }}
	}

	// Calculate potential savings
	currentStat, exists := modelStats[currentModel]
	if !exists {
		return &ModelAnalysis{HasOptimization: func() bool { return false }}
	}

	currentEfficiency := currentStat.TotalCost / float64(currentStat.SuccessCount)
	savings := (currentEfficiency - bestEfficiency) * float64(currentStat.TaskCount)

	return &ModelAnalysis{
		CurrentModel:      currentModel,
		RecommendedModel:  bestModel,
		EstimatedSavings:  savings,
		QualityImpact:     mo.assessQualityImpact(currentModel, bestModel),
		PerformanceImpact: mo.assessPerformanceImpact(currentModel, bestModel),
		ReliabilityImpact: mo.assessReliabilityImpact(currentModel, bestModel),
		Confidence:        mo.calculateConfidence(modelStats, taskType),
		CostPerTask:       map[string]float64{currentModel: currentEfficiency, bestModel: bestEfficiency},
		HasOptimization:   func() bool { return savings > 0 },
	}
}

// getCurrentModel returns the current model for a task type
func (mo *ModelOptimizer) getCurrentModel(taskType string) string {
	// This would come from agent configuration in production
	// For now, use defaults based on task type
	switch taskType {
	case "code_review", "debugging":
		return "gpt-4"
	case "documentation", "writing":
		return "claude-3-opus-20240229"
	case "simple_qa", "chat":
		return "gpt-3.5-turbo"
	default:
		return "gpt-4"
	}
}

// assessQualityImpact assesses quality impact of model switch
func (mo *ModelOptimizer) assessQualityImpact(currentModel, newModel string) float64 {
	// Model quality rankings (higher is better)
	quality := map[string]float64{
		"gpt-4":                    1.0,
		"claude-3-opus-20240229":   0.95,
		"gpt-4-turbo":              0.9,
		"claude-3-sonnet-20240229": 0.8,
		"gpt-3.5-turbo":            0.7,
		"claude-3-haiku-20240307":  0.6,
	}

	currentQuality := quality[currentModel]
	newQuality := quality[newModel]

	// Return relative change (-1.0 to 1.0)
	if currentQuality == 0 {
		return 0
	}

	return (newQuality - currentQuality) / currentQuality
}

// assessPerformanceImpact assesses performance impact of model switch
func (mo *ModelOptimizer) assessPerformanceImpact(currentModel, newModel string) float64 {
	// Model speed rankings (higher is faster)
	speed := map[string]float64{
		"gpt-3.5-turbo":            1.0,
		"claude-3-haiku-20240307":  0.9,
		"gpt-4-turbo":              0.7,
		"claude-3-sonnet-20240229": 0.6,
		"gpt-4":                    0.5,
		"claude-3-opus-20240229":   0.4,
	}

	currentSpeed := speed[currentModel]
	newSpeed := speed[newModel]

	// Return relative change (-1.0 to 1.0)
	if currentSpeed == 0 {
		return 0
	}

	return (newSpeed - currentSpeed) / currentSpeed
}

// assessReliabilityImpact assesses reliability impact of model switch
func (mo *ModelOptimizer) assessReliabilityImpact(currentModel, newModel string) float64 {
	// Model reliability rankings (higher is more reliable)
	reliability := map[string]float64{
		"gpt-4":                    1.0,
		"gpt-4-turbo":              0.95,
		"claude-3-opus-20240229":   0.9,
		"claude-3-sonnet-20240229": 0.85,
		"gpt-3.5-turbo":            0.8,
		"claude-3-haiku-20240307":  0.75,
	}

	currentReliability := reliability[currentModel]
	newReliability := reliability[newModel]

	// Return relative change (-1.0 to 1.0)
	if currentReliability == 0 {
		return 0
	}

	return (newReliability - currentReliability) / currentReliability
}

// calculateConfidence calculates confidence in the analysis
func (mo *ModelOptimizer) calculateConfidence(modelStats map[string]*ModelStat, taskType string) float64 {
	// Base confidence on amount of data and success rates
	totalTasks := 0
	for _, stat := range modelStats {
		totalTasks += stat.TaskCount
	}

	// More data = higher confidence
	dataConfidence := math.Min(float64(totalTasks)/100.0, 1.0)

	// Task type familiarity
	familiarityConfidence := 0.8 // Default confidence
	if strings.Contains(taskType, "code") || strings.Contains(taskType, "debug") {
		familiarityConfidence = 0.9 // Higher confidence for well-understood tasks
	}

	return (dataConfidence + familiarityConfidence) / 2.0
}

// ModelAnalysis contains model analysis results
type ModelAnalysis struct {
	CurrentModel      string             `json:"current_model"`
	RecommendedModel  string             `json:"recommended_model"`
	EstimatedSavings  float64            `json:"estimated_savings"`
	QualityImpact     float64            `json:"quality_impact"`
	PerformanceImpact float64            `json:"performance_impact"`
	ReliabilityImpact float64            `json:"reliability_impact"`
	Confidence        float64            `json:"confidence"`
	CostPerTask       map[string]float64 `json:"cost_per_task"`
	HasOptimization   func() bool        `json:"-"`
}

// ModelStat contains statistics for a model
type ModelStat struct {
	TotalCost    float64 `json:"total_cost"`
	TaskCount    int     `json:"task_count"`
	SuccessCount int     `json:"success_count"`
}

// Utility functions and other optimizers would follow similar patterns...

// generateOptID generates a unique optimization ID
func generateOptID() string {
	return fmt.Sprintf("opt_%d", time.Now().UnixNano())
}

// PerformanceTracker tracks model performance (simplified implementation)
type PerformanceTracker struct{}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(ctx context.Context) *PerformanceTracker {
	return &PerformanceTracker{}
}

// OptimizationExecutor executes optimizations (simplified implementation)
type OptimizationExecutor struct{}

// NewOptimizationExecutor creates a new optimization executor
func NewOptimizationExecutor(ctx context.Context) (*OptimizationExecutor, error) {
	return &OptimizationExecutor{}, nil
}

// SavingsMonitor monitors optimization savings (simplified implementation)
type SavingsMonitor struct{}

// NewSavingsMonitor creates a new savings monitor
func NewSavingsMonitor(ctx context.Context) (*SavingsMonitor, error) {
	return &SavingsMonitor{}, nil
}

// Placeholder implementations for other optimizers
func NewCacheOptimizer(ctx context.Context, tracker *CostTracker) OptimizationStrategy {
	return &CacheOptimizer{tracker: tracker}
}

func NewBatchOptimizer(ctx context.Context, tracker *CostTracker) OptimizationStrategy {
	return &BatchOptimizer{tracker: tracker}
}

func NewResourcePoolOptimizer(ctx context.Context, tracker *CostTracker) OptimizationStrategy {
	return &ResourcePoolOptimizer{tracker: tracker}
}

func NewCompressionOptimizer(ctx context.Context, tracker *CostTracker) OptimizationStrategy {
	return &CompressionOptimizer{tracker: tracker}
}

func NewSchedulingOptimizer(ctx context.Context, tracker *CostTracker) OptimizationStrategy {
	return &SchedulingOptimizer{tracker: tracker}
}

// Simplified implementations of other optimizers
type CacheOptimizer struct{ tracker *CostTracker }

func (co *CacheOptimizer) Name() string  { return "cache_optimizer" }
func (co *CacheOptimizer) Priority() int { return 80 }
func (co *CacheOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	return []Optimization{}, nil // TODO: Implement
}

type BatchOptimizer struct{ tracker *CostTracker }

func (bo *BatchOptimizer) Name() string  { return "batch_optimizer" }
func (bo *BatchOptimizer) Priority() int { return 70 }
func (bo *BatchOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	return []Optimization{}, nil // TODO: Implement
}

type ResourcePoolOptimizer struct{ tracker *CostTracker }

func (rpo *ResourcePoolOptimizer) Name() string  { return "resource_pool_optimizer" }
func (rpo *ResourcePoolOptimizer) Priority() int { return 60 }
func (rpo *ResourcePoolOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	return []Optimization{}, nil // TODO: Implement
}

type CompressionOptimizer struct{ tracker *CostTracker }

func (co *CompressionOptimizer) Name() string  { return "compression_optimizer" }
func (co *CompressionOptimizer) Priority() int { return 50 }
func (co *CompressionOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	return []Optimization{}, nil // TODO: Implement
}

type SchedulingOptimizer struct{ tracker *CostTracker }

func (so *SchedulingOptimizer) Name() string  { return "scheduling_optimizer" }
func (so *SchedulingOptimizer) Priority() int { return 40 }
func (so *SchedulingOptimizer) Analyze(ctx context.Context, usage []Usage) ([]Optimization, error) {
	return []Optimization{}, nil // TODO: Implement
}
