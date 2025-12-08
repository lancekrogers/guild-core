// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// PatternLearner learns and applies reasoning patterns
type PatternLearner struct {
	repository       PatternRepository
	analyzer         *PatternAnalyzer
	applicator       *PatternApplicator
	evaluator        *PatternEvaluator
	featureExtractor *FeatureExtractor

	// Learning configuration
	config PatternLearningConfig

	// Active patterns
	activePatterns sync.Map // map[string]*LearnedPattern

	// Learning queue
	learningQueue chan *LearningTask

	// Metrics
	// metrics         *observability.Metrics // TODO: Update to use MetricsRegistry
}

// PatternLearningConfig configures pattern learning
type PatternLearningConfig struct {
	MinOccurrences      int           `json:"min_occurrences"`
	MinConfidence       float64       `json:"min_confidence"`
	MinSuccessRate      float64       `json:"min_success_rate"`
	LearningRate        float64       `json:"learning_rate"`
	DecayFactor         float64       `json:"decay_factor"`
	MaxActivePatterns   int           `json:"max_active_patterns"`
	EvaluationWindow    time.Duration `json:"evaluation_window"`
	AdaptationThreshold float64       `json:"adaptation_threshold"`
	EnableAutoApply     bool          `json:"enable_auto_apply"`
}

// DefaultPatternLearningConfig returns default configuration
func DefaultPatternLearningConfig() PatternLearningConfig {
	return PatternLearningConfig{
		MinOccurrences:      3,
		MinConfidence:       0.7,
		MinSuccessRate:      0.6,
		LearningRate:        0.1,
		DecayFactor:         0.95,
		MaxActivePatterns:   50,
		EvaluationWindow:    24 * time.Hour,
		AdaptationThreshold: 0.8,
		EnableAutoApply:     true,
	}
}

// LearnedPattern represents a pattern learned from successful reasoning
type LearnedPattern struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Pattern structure
	Signature     PatternSignature `json:"signature"`
	Features      []PatternFeature `json:"features"`
	Prerequisites []string         `json:"prerequisites"`

	// Learning data
	Examples   []PatternExample  `json:"examples"`
	Statistics PatternStatistics `json:"statistics"`

	// Application
	Template      ReasoningTemplate  `json:"template"`
	Applicability ApplicabilityScore `json:"applicability"`

	// Metadata
	CreatedAt   time.Time     `json:"created_at"`
	LastUsed    time.Time     `json:"last_used"`
	LastUpdated time.Time     `json:"last_updated"`
	Version     int           `json:"version"`
	Tags        []string      `json:"tags"`
	Status      PatternStatus `json:"status"`
}

// PatternSignature defines the structural signature of a pattern
type PatternSignature struct {
	InputTypes      []ThinkingType      `json:"input_types"`
	OutputTypes     []ThinkingType      `json:"output_types"`
	TypicalSequence []ThinkingType      `json:"typical_sequence"`
	MinBlocks       int                 `json:"min_blocks"`
	MaxBlocks       int                 `json:"max_blocks"`
	Constraints     []PatternConstraint `json:"constraints"`
}

// PatternFeature represents a distinguishing feature
type PatternFeature struct {
	Name       string      `json:"name"`
	Type       FeatureType `json:"type"`
	Weight     float64     `json:"weight"`
	Value      interface{} `json:"value"`
	Importance float64     `json:"importance"`
}

// FeatureType categorizes pattern features
type FeatureType string

const (
	FeatureTypeStructural  FeatureType = "structural"
	FeatureTypeSemantic    FeatureType = "semantic"
	FeatureTypeTemporal    FeatureType = "temporal"
	FeatureTypeContextual  FeatureType = "contextual"
	FeatureTypePerformance FeatureType = "performance"
)

// PatternExample represents an example usage
type PatternExample struct {
	ID          string                 `json:"id"`
	ChainID     string                 `json:"chain_id"`
	Context     map[string]interface{} `json:"context"`
	Input       []ThinkingBlock        `json:"input"`
	Output      []ThinkingBlock        `json:"output"`
	Success     bool                   `json:"success"`
	Performance PerformanceMetrics     `json:"performance"`
	Timestamp   time.Time              `json:"timestamp"`
}

// PatternStatistics tracks pattern performance
type PatternStatistics struct {
	TotalUsages      int            `json:"total_usages"`
	SuccessfulUsages int            `json:"successful_usages"`
	FailedUsages     int            `json:"failed_usages"`
	SuccessRate      float64        `json:"success_rate"`
	AverageScore     float64        `json:"average_score"`
	RecentScores     []float64      `json:"recent_scores"`
	Trend            TrendDirection `json:"trend"`
	LastEvaluation   time.Time      `json:"last_evaluation"`
}

// TrendDirection indicates performance trend
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendStable    TrendDirection = "stable"
	TrendDeclining TrendDirection = "declining"
)

// ReasoningTemplate defines how to apply a pattern
type ReasoningTemplate struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Prompt components
	SystemPrompt        string `json:"system_prompt"`
	InstructionTemplate string `json:"instruction_template"`
	ExampleFormat       string `json:"example_format"`

	// Guidance
	Steps          []TemplateStep `json:"steps"`
	Hints          []string       `json:"hints"`
	CommonPitfalls []string       `json:"common_pitfalls"`

	// Constraints
	RequiredElements  []string `json:"required_elements"`
	ForbiddenElements []string `json:"forbidden_elements"`
}

// TemplateStep represents a step in applying a pattern
type TemplateStep struct {
	Order           int          `json:"order"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	ExpectedType    ThinkingType `json:"expected_type"`
	Required        bool         `json:"required"`
	ValidationRules []string     `json:"validation_rules"`
}

// ApplicabilityScore measures how well a pattern fits
type ApplicabilityScore struct {
	Overall        float64 `json:"overall"`
	ContextMatch   float64 `json:"context_match"`
	StructureMatch float64 `json:"structure_match"`
	HistoryMatch   float64 `json:"history_match"`
	Confidence     float64 `json:"confidence"`
}

// PatternStatus indicates pattern lifecycle state
type PatternStatus string

const (
	PatternStatusLearning   PatternStatus = "learning"
	PatternStatusActive     PatternStatus = "active"
	PatternStatusDeprecated PatternStatus = "deprecated"
	PatternStatusArchived   PatternStatus = "archived"
)

// PatternConstraint defines applicability constraints
type PatternConstraint struct {
	Type     ConstraintType `json:"type"`
	Field    string         `json:"field"`
	Operator string         `json:"operator"`
	Value    interface{}    `json:"value"`
}

// ConstraintType categorizes constraints
type ConstraintType string

const (
	ConstraintTypeContext    ConstraintType = "context"
	ConstraintTypeComplexity ConstraintType = "complexity"
	ConstraintTypeResource   ConstraintType = "resource"
	ConstraintTypeTemporal   ConstraintType = "temporal"
)

// LearningTask represents a pattern learning task
type LearningTask struct {
	ID          string
	Type        LearningTaskType
	SourceChain *ReasoningChainEnhanced
	Feedback    *ReasoningFeedback
	Context     map[string]interface{}
}

// LearningTaskType categorizes learning tasks
type LearningTaskType string

const (
	LearningTaskDiscover  LearningTaskType = "discover"
	LearningTaskReinforce LearningTaskType = "reinforce"
	LearningTaskAdapt     LearningTaskType = "adapt"
	LearningTaskEvaluate  LearningTaskType = "evaluate"
)

// NewPatternLearner creates a new pattern learner
func NewPatternLearner(
	repository PatternRepository,
	config PatternLearningConfig,
) *PatternLearner {
	pl := &PatternLearner{
		repository:       repository,
		analyzer:         NewPatternAnalyzer(),
		applicator:       NewPatternApplicator(),
		evaluator:        NewPatternEvaluator(),
		featureExtractor: NewFeatureExtractor(),
		config:           config,
		learningQueue:    make(chan *LearningTask, 100),
	}

	// Start learning worker
	go pl.learningWorker()

	// Load active patterns
	pl.loadActivePatterns()

	return pl
}

// Learn processes a reasoning chain for pattern learning
func (pl *PatternLearner) Learn(ctx context.Context, chain *ReasoningChainEnhanced, feedback *ReasoningFeedback) error {
	logger := observability.GetLogger(ctx)

	// Create learning task
	task := &LearningTask{
		ID:          uuid.New().String(),
		Type:        pl.determineLearningType(chain, feedback),
		SourceChain: chain,
		Feedback:    feedback,
		Context: map[string]interface{}{
			"timestamp": time.Now(),
			"agent_id":  chain.AgentID,
		},
	}

	// Queue for processing
	select {
	case pl.learningQueue <- task:
		logger.DebugContext(ctx, "Queued learning task",
			"task_id", task.ID,
			"type", task.Type)
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "learning cancelled").
			WithComponent("pattern_learner")
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "learning queue full", nil).
			WithComponent("pattern_learner")
	}
}

// SuggestPatterns suggests applicable patterns for a context
func (pl *PatternLearner) SuggestPatterns(ctx context.Context, context PatternContext) ([]PatternSuggestion, error) {
	suggestions := make([]PatternSuggestion, 0)

	// Extract features from context
	features := pl.featureExtractor.ExtractContextFeatures(context)

	// Score all active patterns
	pl.activePatterns.Range(func(key, value interface{}) bool {
		pattern := value.(*LearnedPattern)

		// Check if pattern is applicable
		if pattern.Status != PatternStatusActive {
			return true
		}

		// Calculate applicability score
		score := pl.calculateApplicability(pattern, features, context)

		if score.Overall >= pl.config.MinConfidence {
			suggestions = append(suggestions, PatternSuggestion{
				Pattern:       pattern,
				Score:         score,
				Rationale:     pl.generateRationale(pattern, score),
				EstimatedGain: pl.estimateGain(pattern, context),
			})
		}

		return true
	})

	// Sort by score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score.Overall > suggestions[j].Score.Overall
	})

	// Limit to top suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions, nil
}

// ApplyPattern applies a learned pattern to enhance reasoning
func (pl *PatternLearner) ApplyPattern(ctx context.Context, patternID string, input *PatternInput) (*PatternOutput, error) {
	// Get pattern
	patternI, exists := pl.activePatterns.Load(patternID)
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "pattern not found", nil).
			WithComponent("pattern_learner").
			WithDetails("pattern_id", patternID)
	}

	pattern := patternI.(*LearnedPattern)

	// Apply pattern
	output, err := pl.applicator.Apply(ctx, pattern, input)
	if err != nil {
		// Record failure
		pl.recordApplication(pattern, false, 0)
		return nil, err
	}

	// Evaluate application
	score := pl.evaluator.EvaluateApplication(pattern, input, output)

	// Record application
	pl.recordApplication(pattern, score > pl.config.MinSuccessRate, score)

	// Update pattern statistics
	pl.updatePatternStats(pattern, score)

	return output, nil
}

// determineLearningType determines the type of learning task
func (pl *PatternLearner) determineLearningType(chain *ReasoningChainEnhanced, feedback *ReasoningFeedback) LearningTaskType {
	// High-quality chains with good feedback -> discover new patterns
	if chain.Quality.Overall > 0.8 && (feedback == nil || feedback.Rating > 0.8) {
		return LearningTaskDiscover
	}

	// Existing pattern with feedback -> reinforce or adapt
	if len(chain.Patterns) > 0 && feedback != nil {
		if feedback.Rating > 0.7 {
			return LearningTaskReinforce
		}
		return LearningTaskAdapt
	}

	// Default to evaluation
	return LearningTaskEvaluate
}

// learningWorker processes learning tasks
func (pl *PatternLearner) learningWorker() {
	for task := range pl.learningQueue {
		ctx := context.Background()

		switch task.Type {
		case LearningTaskDiscover:
			pl.discoverPatterns(ctx, task)
		case LearningTaskReinforce:
			pl.reinforcePatterns(ctx, task)
		case LearningTaskAdapt:
			pl.adaptPatterns(ctx, task)
		case LearningTaskEvaluate:
			pl.evaluatePatterns(ctx, task)
		}

		// TODO: Update to use MetricsRegistry
		// pl.metrics.RecordCounter("pattern_learning_tasks", 1,
		// 	"type", string(task.Type))
	}
}

// discoverPatterns discovers new patterns from successful chains
func (pl *PatternLearner) discoverPatterns(ctx context.Context, task *LearningTask) {
	logger := observability.GetLogger(ctx)
	chain := task.SourceChain

	// Analyze chain for patterns
	candidates := pl.analyzer.DiscoverPatterns(chain)

	for _, candidate := range candidates {
		// Check if pattern already exists
		if pl.patternExists(candidate) {
			continue
		}

		// Extract features
		features := pl.featureExtractor.ExtractPatternFeatures(candidate, chain)

		// Create learned pattern
		pattern := &LearnedPattern{
			ID:          uuid.New().String(),
			Name:        candidate.Name,
			Description: candidate.Description,
			Signature:   candidate.Signature,
			Features:    features,
			Examples: []PatternExample{{
				ID:          uuid.New().String(),
				ChainID:     chain.ID,
				Context:     task.Context,
				Success:     true,
				Performance: chain.Performance,
				Timestamp:   time.Now(),
			}},
			Statistics: PatternStatistics{
				TotalUsages:      1,
				SuccessfulUsages: 1,
				SuccessRate:      1.0,
				AverageScore:     chain.Quality.Overall,
				RecentScores:     []float64{chain.Quality.Overall},
				Trend:            TrendStable,
				LastEvaluation:   time.Now(),
			},
			CreatedAt:   time.Now(),
			LastUsed:    time.Now(),
			LastUpdated: time.Now(),
			Version:     1,
			Status:      PatternStatusLearning,
		}

		// Generate template
		pattern.Template = pl.generateTemplate(pattern)

		// Save pattern
		if err := pl.repository.SavePattern(ctx, pattern); err != nil {
			logger.ErrorContext(ctx, "Failed to save discovered pattern",
				"error", err,
				"pattern_name", pattern.Name)
			continue
		}

		logger.InfoContext(ctx, "Discovered new pattern",
			"pattern_id", pattern.ID,
			"pattern_name", pattern.Name)

		// Add to active patterns if meets criteria
		if pattern.Statistics.TotalUsages >= pl.config.MinOccurrences {
			pattern.Status = PatternStatusActive
			pl.activePatterns.Store(pattern.ID, pattern)
		}
	}
}

// reinforcePatterns strengthens successful patterns
func (pl *PatternLearner) reinforcePatterns(ctx context.Context, task *LearningTask) {
	logger := observability.GetLogger(ctx)
	chain := task.SourceChain
	feedback := task.Feedback

	// Update patterns used in this chain
	for _, patternMatch := range chain.Patterns {
		patternI, exists := pl.activePatterns.Load(patternMatch.PatternID)
		if !exists {
			continue
		}

		pattern := patternI.(*LearnedPattern)

		// Add successful example
		example := PatternExample{
			ID:          uuid.New().String(),
			ChainID:     chain.ID,
			Context:     task.Context,
			Success:     feedback.Rating > pl.config.MinSuccessRate,
			Performance: chain.Performance,
			Timestamp:   time.Now(),
		}

		pattern.Examples = append(pattern.Examples, example)

		// Update statistics with reinforcement
		pl.reinforceStatistics(pattern, feedback.Rating)

		// Update features based on success
		pl.updateFeatures(pattern, chain, feedback.Rating)

		pattern.LastUsed = time.Now()
		pattern.LastUpdated = time.Now()
		pattern.Version++

		// Save updated pattern
		if err := pl.repository.SavePattern(ctx, pattern); err != nil {
			logger.ErrorContext(ctx, "Failed to reinforce pattern",
				"error", err,
				"pattern_id", pattern.ID)
		}
	}
}

// adaptPatterns adapts patterns based on feedback
func (pl *PatternLearner) adaptPatterns(ctx context.Context, task *LearningTask) {
	logger := observability.GetLogger(ctx)
	chain := task.SourceChain
	feedback := task.Feedback

	// Patterns that need adaptation
	for _, patternMatch := range chain.Patterns {
		if patternMatch.Confidence < pl.config.AdaptationThreshold {
			continue
		}

		patternI, exists := pl.activePatterns.Load(patternMatch.PatternID)
		if !exists {
			continue
		}

		pattern := patternI.(*LearnedPattern)

		// Analyze what went wrong
		issues := pl.analyzer.AnalyzeFailure(pattern, chain, feedback)

		// Adapt pattern based on issues
		for _, issue := range issues {
			switch issue.Type {
			case "constraint_violation":
				pl.adaptConstraints(pattern, issue)
			case "feature_mismatch":
				pl.adaptFeatures(pattern, issue)
			case "template_inadequate":
				pl.adaptTemplate(pattern, issue)
			}
		}

		// Mark for re-evaluation
		pattern.Status = PatternStatusLearning
		pattern.LastUpdated = time.Now()
		pattern.Version++

		// Save adapted pattern
		if err := pl.repository.SavePattern(ctx, pattern); err != nil {
			logger.ErrorContext(ctx, "Failed to adapt pattern",
				"error", err,
				"pattern_id", pattern.ID)
		}
	}
}

// evaluatePatterns evaluates pattern performance
func (pl *PatternLearner) evaluatePatterns(ctx context.Context, task *LearningTask) {
	logger := observability.GetLogger(ctx)
	now := time.Now()

	pl.activePatterns.Range(func(key, value interface{}) bool {
		pattern := value.(*LearnedPattern)

		// Skip if recently evaluated
		if now.Sub(pattern.Statistics.LastEvaluation) < pl.config.EvaluationWindow {
			return true
		}

		// Evaluate performance
		performance := pl.evaluator.EvaluatePattern(pattern)

		// Update status based on performance
		if performance.SuccessRate < pl.config.MinSuccessRate {
			if pattern.Status == PatternStatusActive {
				pattern.Status = PatternStatusDeprecated
				logger.WarnContext(ctx, "Pattern deprecated due to poor performance",
					"pattern_id", pattern.ID,
					"success_rate", performance.SuccessRate)
			}
		} else if performance.TotalUsages >= pl.config.MinOccurrences {
			if pattern.Status == PatternStatusLearning {
				pattern.Status = PatternStatusActive
				logger.InfoContext(ctx, "Pattern promoted to active",
					"pattern_id", pattern.ID,
					"success_rate", performance.SuccessRate)
			}
		}

		// Update statistics
		pattern.Statistics = performance
		pattern.Statistics.LastEvaluation = now

		// Apply decay to old patterns
		if now.Sub(pattern.LastUsed) > 7*24*time.Hour {
			pl.applyDecay(pattern)
		}

		// Save updated pattern
		pl.repository.SavePattern(ctx, pattern)

		return true
	})
}

// Helper methods

func (pl *PatternLearner) patternExists(candidate PatternCandidate) bool {
	exists := false

	pl.activePatterns.Range(func(key, value interface{}) bool {
		pattern := value.(*LearnedPattern)
		if pl.isSimilarPattern(pattern, candidate) {
			exists = true
			return false
		}
		return true
	})

	return exists
}

func (pl *PatternLearner) isSimilarPattern(pattern *LearnedPattern, candidate PatternCandidate) bool {
	// Compare signatures
	if len(pattern.Signature.InputTypes) != len(candidate.Signature.InputTypes) {
		return false
	}

	for i, t := range pattern.Signature.InputTypes {
		if t != candidate.Signature.InputTypes[i] {
			return false
		}
	}

	// Compare features
	featureSimilarity := pl.compareFeatures(pattern.Features, candidate.Features)

	return featureSimilarity > 0.8
}

func (pl *PatternLearner) compareFeatures(features1, features2 []PatternFeature) float64 {
	if len(features1) == 0 || len(features2) == 0 {
		return 0
	}

	// Simple feature comparison
	matches := 0
	for _, f1 := range features1 {
		for _, f2 := range features2 {
			if f1.Name == f2.Name && f1.Type == f2.Type {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(max(len(features1), len(features2)))
}

func (pl *PatternLearner) calculateApplicability(pattern *LearnedPattern, features []ContextFeature, context PatternContext) ApplicabilityScore {
	// Context match
	contextMatch := pl.matchContext(pattern, context)

	// Structure match
	structureMatch := pl.matchStructure(pattern, context)

	// History match
	historyMatch := pl.matchHistory(pattern, context)

	// Calculate overall score
	overall := (contextMatch*0.4 + structureMatch*0.4 + historyMatch*0.2)

	// Apply confidence based on pattern statistics
	confidence := pattern.Statistics.SuccessRate * pattern.Statistics.AverageScore

	return ApplicabilityScore{
		Overall:        overall * confidence,
		ContextMatch:   contextMatch,
		StructureMatch: structureMatch,
		HistoryMatch:   historyMatch,
		Confidence:     confidence,
	}
}

func (pl *PatternLearner) generateRationale(pattern *LearnedPattern, score ApplicabilityScore) string {
	rationale := fmt.Sprintf("Pattern '%s' is %.0f%% applicable. ",
		pattern.Name, score.Overall*100)

	if score.ContextMatch > 0.8 {
		rationale += "Strong context match. "
	}
	if score.StructureMatch > 0.8 {
		rationale += "Problem structure aligns well. "
	}
	if score.HistoryMatch > 0.7 {
		rationale += "Similar patterns successful in past. "
	}

	return rationale
}

func (pl *PatternLearner) estimateGain(pattern *LearnedPattern, context PatternContext) EstimatedGain {
	// Estimate based on historical performance
	avgImprovement := 0.0
	for _, example := range pattern.Examples {
		if example.Success {
			avgImprovement += 0.2 // Simplified
		}
	}

	if len(pattern.Examples) > 0 {
		avgImprovement /= float64(len(pattern.Examples))
	}

	return EstimatedGain{
		QualityImprovement: avgImprovement,
		TimeReduction:      pattern.Statistics.AverageScore * 0.3,
		ConfidenceBoost:    pattern.Statistics.SuccessRate * 0.2,
	}
}

func (pl *PatternLearner) recordApplication(pattern *LearnedPattern, success bool, score float64) {
	pattern.Statistics.TotalUsages++
	if success {
		pattern.Statistics.SuccessfulUsages++
	} else {
		pattern.Statistics.FailedUsages++
	}

	// Update success rate
	pattern.Statistics.SuccessRate = float64(pattern.Statistics.SuccessfulUsages) /
		float64(pattern.Statistics.TotalUsages)

	// Update recent scores
	pattern.Statistics.RecentScores = append(pattern.Statistics.RecentScores, score)
	if len(pattern.Statistics.RecentScores) > 10 {
		pattern.Statistics.RecentScores = pattern.Statistics.RecentScores[1:]
	}

	// Update average score
	sum := 0.0
	for _, s := range pattern.Statistics.RecentScores {
		sum += s
	}
	pattern.Statistics.AverageScore = sum / float64(len(pattern.Statistics.RecentScores))

	// Determine trend
	if len(pattern.Statistics.RecentScores) >= 5 {
		recent := pattern.Statistics.RecentScores[len(pattern.Statistics.RecentScores)-5:]
		if pl.isImproving(recent) {
			pattern.Statistics.Trend = TrendImproving
		} else if pl.isDeclining(recent) {
			pattern.Statistics.Trend = TrendDeclining
		} else {
			pattern.Statistics.Trend = TrendStable
		}
	}
}

func (pl *PatternLearner) isImproving(scores []float64) bool {
	if len(scores) < 2 {
		return false
	}

	// Simple linear regression
	sum := 0.0
	for i := 1; i < len(scores); i++ {
		sum += scores[i] - scores[i-1]
	}

	return sum > 0.1
}

func (pl *PatternLearner) isDeclining(scores []float64) bool {
	if len(scores) < 2 {
		return false
	}

	sum := 0.0
	for i := 1; i < len(scores); i++ {
		sum += scores[i] - scores[i-1]
	}

	return sum < -0.1
}

func (pl *PatternLearner) updatePatternStats(pattern *LearnedPattern, score float64) {
	// Update with learning rate
	pattern.Statistics.AverageScore = pattern.Statistics.AverageScore*(1-pl.config.LearningRate) +
		score*pl.config.LearningRate

	pattern.LastUsed = time.Now()
}

func (pl *PatternLearner) reinforceStatistics(pattern *LearnedPattern, rating float64) {
	// Reinforce based on feedback
	boost := (rating - 0.5) * pl.config.LearningRate

	// Update scores with boost
	for i := range pattern.Statistics.RecentScores {
		pattern.Statistics.RecentScores[i] *= (1 + boost)
		pattern.Statistics.RecentScores[i] = math.Min(1.0, pattern.Statistics.RecentScores[i])
	}
}

func (pl *PatternLearner) updateFeatures(pattern *LearnedPattern, chain *ReasoningChainEnhanced, rating float64) {
	// Extract current features
	currentFeatures := pl.featureExtractor.ExtractChainFeatures(chain)

	// Update pattern features based on success
	for i, pf := range pattern.Features {
		for _, cf := range currentFeatures {
			if pf.Name == cf.Name {
				// Adjust weight based on rating
				adjustment := (rating - 0.5) * pl.config.LearningRate
				pattern.Features[i].Weight *= (1 + adjustment)
				pattern.Features[i].Weight = math.Max(0.1, math.Min(1.0, pattern.Features[i].Weight))
				break
			}
		}
	}
}

func (pl *PatternLearner) generateTemplate(pattern *LearnedPattern) ReasoningTemplate {
	template := ReasoningTemplate{
		ID:    uuid.New().String(),
		Name:  pattern.Name + "_template",
		Steps: make([]TemplateStep, 0),
	}

	// Generate steps from signature
	for i, thinkingType := range pattern.Signature.TypicalSequence {
		step := TemplateStep{
			Order:        i + 1,
			Name:         fmt.Sprintf("Step %d: %s", i+1, thinkingType),
			Description:  pl.getStepDescription(thinkingType),
			ExpectedType: thinkingType,
			Required:     i < 3, // First 3 steps required
		}
		template.Steps = append(template.Steps, step)
	}

	// Add hints based on successful examples
	template.Hints = pl.extractHints(pattern)

	// Add common pitfalls
	template.CommonPitfalls = pl.identifyPitfalls(pattern)

	return template
}

func (pl *PatternLearner) getStepDescription(thinkingType ThinkingType) string {
	descriptions := map[ThinkingType]string{
		ThinkingTypeAnalysis:       "Analyze the problem and identify key components",
		ThinkingTypePlanning:       "Create a structured plan with clear steps",
		ThinkingTypeDecisionMaking: "Make informed decisions based on analysis",
		ThinkingTypeToolSelection:  "Select appropriate tools for the task",
		ThinkingTypeVerification:   "Verify the approach and results",
		ThinkingTypeHypothesis:     "Form hypotheses about the solution",
		ThinkingTypeErrorRecovery:  "Recover from errors and adapt approach",
	}

	if desc, ok := descriptions[thinkingType]; ok {
		return desc
	}
	return "Perform reasoning step"
}

func (pl *PatternLearner) extractHints(pattern *LearnedPattern) []string {
	hints := []string{}

	// Analyze successful examples
	for _, example := range pattern.Examples {
		if example.Success && example.Performance.ThinkingTime < 5*time.Second {
			hints = append(hints, "Quick decision-making led to success")
		}
	}

	// Add feature-based hints
	for _, feature := range pattern.Features {
		if feature.Importance > 0.8 {
			hints = append(hints, fmt.Sprintf("Focus on %s", feature.Name))
		}
	}

	return hints
}

func (pl *PatternLearner) identifyPitfalls(pattern *LearnedPattern) []string {
	pitfalls := []string{}

	// Analyze failed examples
	failureReasons := make(map[string]int)
	for _, example := range pattern.Examples {
		if !example.Success {
			// Simplified - in production, analyze failure reasons
			failureReasons["incomplete_analysis"]++
		}
	}

	for reason, count := range failureReasons {
		if count > 2 {
			pitfalls = append(pitfalls, fmt.Sprintf("Avoid %s", reason))
		}
	}

	return pitfalls
}

func (pl *PatternLearner) applyDecay(pattern *LearnedPattern) {
	// Apply decay to weights and scores
	for i := range pattern.Features {
		pattern.Features[i].Weight *= pl.config.DecayFactor
	}

	// Decay recent scores
	for i := range pattern.Statistics.RecentScores {
		pattern.Statistics.RecentScores[i] *= pl.config.DecayFactor
	}
}

func (pl *PatternLearner) loadActivePatterns() {
	ctx := context.Background()
	logger := observability.GetLogger(ctx)

	// Handle nil repository (for testing)
	if pl.repository == nil {
		logger.InfoContext(ctx, "No repository provided, skipping pattern loading")
		return
	}

	patterns, err := pl.repository.GetActivePatterns(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to load active patterns", "error", err)
		return
	}

	for _, pattern := range patterns {
		pl.activePatterns.Store(pattern.ID, pattern)
	}

	logger.InfoContext(ctx, "Loaded active patterns", "count", len(patterns))
}

// Helper functions

func (pl *PatternLearner) matchContext(pattern *LearnedPattern, context PatternContext) float64 {
	// Simplified context matching
	score := 0.5

	// Check constraints
	for _, constraint := range pattern.Signature.Constraints {
		if pl.checkConstraint(constraint, context) {
			score += 0.1
		} else {
			score -= 0.2
		}
	}

	return math.Max(0, math.Min(1, score))
}

func (pl *PatternLearner) matchStructure(pattern *LearnedPattern, context PatternContext) float64 {
	// Check if current structure matches pattern signature
	if context.CurrentBlocks < pattern.Signature.MinBlocks {
		return 0.3
	}

	if context.CurrentBlocks > pattern.Signature.MaxBlocks {
		return 0.5
	}

	return 0.8
}

func (pl *PatternLearner) matchHistory(pattern *LearnedPattern, context PatternContext) float64 {
	// Check if similar contexts succeeded with this pattern
	successCount := 0
	for _, example := range pattern.Examples {
		if pl.isSimilarContext(example.Context, context.Metadata) {
			if example.Success {
				successCount++
			}
		}
	}

	if len(pattern.Examples) == 0 {
		return 0.5
	}

	return float64(successCount) / float64(len(pattern.Examples))
}

func (pl *PatternLearner) checkConstraint(constraint PatternConstraint, context PatternContext) bool {
	// Simplified constraint checking
	switch constraint.Type {
	case ConstraintTypeComplexity:
		if constraint.Field == "task_complexity" {
			// Check if complexity matches
			return true // Simplified
		}
	case ConstraintTypeContext:
		// Check context fields
		return true // Simplified
	}

	return false
}

func (pl *PatternLearner) isSimilarContext(ctx1, ctx2 map[string]interface{}) bool {
	// Simple similarity check
	matches := 0
	for k, v1 := range ctx1 {
		if v2, ok := ctx2[k]; ok && v1 == v2 {
			matches++
		}
	}

	return matches > len(ctx1)/2
}

func (pl *PatternLearner) adaptConstraints(pattern *LearnedPattern, issue PatternIssue) {
	// Adjust constraints based on failure
	for i, constraint := range pattern.Signature.Constraints {
		if constraint.Field == issue.Field {
			// Relax constraint
			pattern.Signature.Constraints[i] = pl.relaxConstraint(constraint)
		}
	}
}

func (pl *PatternLearner) relaxConstraint(constraint PatternConstraint) PatternConstraint {
	// Simplified constraint relaxation
	relaxed := constraint

	switch constraint.Operator {
	case ">":
		if val, ok := constraint.Value.(float64); ok {
			relaxed.Value = val * 0.9
		}
	case "<":
		if val, ok := constraint.Value.(float64); ok {
			relaxed.Value = val * 1.1
		}
	}

	return relaxed
}

func (pl *PatternLearner) adaptFeatures(pattern *LearnedPattern, issue PatternIssue) {
	// Adjust feature weights
	for i, feature := range pattern.Features {
		if feature.Name == issue.Feature {
			pattern.Features[i].Weight *= 0.8
			pattern.Features[i].Importance *= 0.9
		}
	}
}

func (pl *PatternLearner) adaptTemplate(pattern *LearnedPattern, issue PatternIssue) {
	// Add new step or modify existing
	if issue.SuggestedFix != "" {
		pattern.Template.Hints = append(pattern.Template.Hints, issue.SuggestedFix)
	}
}

// Supporting types

type PatternContext struct {
	Task          string
	CurrentBlocks int
	Complexity    float64
	Resources     map[string]interface{}
	History       []string
	Metadata      map[string]interface{}
}

type PatternSuggestion struct {
	Pattern       *LearnedPattern
	Score         ApplicabilityScore
	Rationale     string
	EstimatedGain EstimatedGain
}

type EstimatedGain struct {
	QualityImprovement float64
	TimeReduction      float64
	ConfidenceBoost    float64
}

type PatternInput struct {
	Context     map[string]interface{}
	Task        string
	Constraints []string
	Examples    []string
}

type PatternOutput struct {
	EnhancedPrompt string
	GuidedSteps    []GuidedStep
	Confidence     float64
	Explanation    string
}

type GuidedStep struct {
	Order       int
	Instruction string
	Hints       []string
	Examples    []string
}

type PatternCandidate struct {
	Name        string
	Description string
	Signature   PatternSignature
	Features    []PatternFeature
}

type ContextFeature struct {
	Name  string
	Value interface{}
	Type  FeatureType
}

type PatternIssue struct {
	Type         string
	Field        string
	Feature      string
	Description  string
	SuggestedFix string
}

// PatternRepository interface for pattern storage
type PatternRepository interface {
	SavePattern(ctx context.Context, pattern *LearnedPattern) error
	GetPattern(ctx context.Context, id string) (*LearnedPattern, error)
	GetActivePatterns(ctx context.Context) ([]*LearnedPattern, error)
	SearchPatterns(ctx context.Context, query PatternQuery) ([]*LearnedPattern, error)
	UpdatePattern(ctx context.Context, pattern *LearnedPattern) error
	DeletePattern(ctx context.Context, id string) error
}

type PatternQuery struct {
	Status   PatternStatus
	MinScore float64
	Tags     []string
	Limit    int
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
