// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/observability"
)

// ReasoningChainEnhanced represents a complete reasoning flow with rich metadata
type ReasoningChainEnhanced struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`
	TaskID    string `json:"task_id"`

	// Content
	Blocks          []*ThinkingBlock `json:"blocks"`
	Summary         string           `json:"summary"`
	FinalConfidence float64          `json:"final_confidence"`

	// Metadata
	Strategy    ReasoningStrategy  `json:"strategy"`
	Quality     QualityMetrics     `json:"quality"`
	Performance PerformanceMetrics `json:"performance"`
	Insights    []Insight          `json:"insights"`
	Patterns    []PatternMatch     `json:"patterns"`

	// Tracking
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	TotalTokens int       `json:"total_tokens"`
	TotalCost   float64   `json:"total_cost"`

	// Learning
	Feedback     *ReasoningFeedback `json:"feedback,omitempty"`
	Improvements []Improvement      `json:"improvements,omitempty"`

	// Context
	Context map[string]interface{} `json:"context,omitempty"`
	Tags    []string               `json:"tags,omitempty"`
}

// ReasoningStrategy represents the approach taken
type ReasoningStrategy struct {
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	StepsPlanned  int                  `json:"steps_planned"`
	StepsExecuted int                  `json:"steps_executed"`
	Adaptations   []StrategyAdaptation `json:"adaptations,omitempty"`
}

// StrategyAdaptation represents a change in approach
type StrategyAdaptation struct {
	Reason       string `json:"reason"`
	FromStrategy string `json:"from_strategy"`
	ToStrategy   string `json:"to_strategy"`
	AtStep       int    `json:"at_step"`
	Impact       string `json:"impact"`
}

// QualityMetrics measures reasoning quality
type QualityMetrics struct {
	Coherence    float64 `json:"coherence"`    // 0-1: How well thoughts connect
	Completeness float64 `json:"completeness"` // 0-1: Coverage of problem space
	Depth        float64 `json:"depth"`        // 0-1: Level of analysis
	Accuracy     float64 `json:"accuracy"`     // 0-1: Correctness of reasoning
	Innovation   float64 `json:"innovation"`   // 0-1: Novel approaches
	Overall      float64 `json:"overall"`      // 0-1: Weighted average
}

// PerformanceMetrics tracks efficiency
type PerformanceMetrics struct {
	ThinkingTime    time.Duration `json:"thinking_time"`
	TokensPerSecond float64       `json:"tokens_per_second"`
	DecisionSpeed   time.Duration `json:"decision_speed"`
	BacktrackCount  int           `json:"backtrack_count"`
	IterationCount  int           `json:"iteration_count"`
	ParallelPaths   int           `json:"parallel_paths"`
}

// Insight represents a key learning or observation
type Insight struct {
	Type        InsightType `json:"type"`
	Description string      `json:"description"`
	Confidence  float64     `json:"confidence"`
	Source      string      `json:"source"` // Block ID or "analysis"
	Actionable  bool        `json:"actionable"`
	Actions     []string    `json:"actions,omitempty"`
}

// InsightType categorizes insights
type InsightType string

const (
	InsightTypePattern      InsightType = "pattern"
	InsightTypeAnomaly      InsightType = "anomaly"
	InsightTypeOptimization InsightType = "optimization"
	InsightTypeRisk         InsightType = "risk"
	InsightTypeOpportunity  InsightType = "opportunity"
)

// PatternMatch represents a recognized pattern
type PatternMatch struct {
	PatternID    string              `json:"pattern_id"`
	PatternName  string              `json:"pattern_name"`
	Confidence   float64             `json:"confidence"`
	Occurrences  []PatternOccurrence `json:"occurrences"`
	Implications []string            `json:"implications"`
}

// PatternOccurrence represents where a pattern was found
type PatternOccurrence struct {
	BlockID  string  `json:"block_id"`
	Location string  `json:"location"`
	Strength float64 `json:"strength"`
}

// ReasoningFeedback represents human or system feedback
type ReasoningFeedback struct {
	ID          string       `json:"id"`
	Type        FeedbackType `json:"type"`
	Rating      float64      `json:"rating"` // 0-1
	Comments    string       `json:"comments"`
	Suggestions []string     `json:"suggestions"`
	ProvidedBy  string       `json:"provided_by"`
	ProvidedAt  time.Time    `json:"provided_at"`
}

// FeedbackType categorizes feedback
type FeedbackType string

const (
	FeedbackTypeHuman     FeedbackType = "human"
	FeedbackTypeSystem    FeedbackType = "system"
	FeedbackTypeOutcome   FeedbackType = "outcome"
	FeedbackTypeAutomatic FeedbackType = "automatic"
)

// Improvement represents a suggested enhancement
type Improvement struct {
	Area            string  `json:"area"`
	Description     string  `json:"description"`
	Priority        int     `json:"priority"`
	EstimatedImpact float64 `json:"estimated_impact"`
	Implementation  string  `json:"implementation"`
}

// ReasoningChainBuilder constructs reasoning chains with analysis
type ReasoningChainBuilder struct {
	chain            *ReasoningChainEnhanced
	analyzer         *ChainAnalyzer
	patternMatcher   *PatternMatcher
	qualityScorer    *QualityScorer
	insightExtractor *InsightExtractor
	mu               sync.Mutex
}

// NewReasoningChainBuilder creates a new chain builder
func NewReasoningChainBuilder(agentID, sessionID, taskID string) *ReasoningChainBuilder {
	return &ReasoningChainBuilder{
		chain: &ReasoningChainEnhanced{
			ID:        uuid.New().String(),
			AgentID:   agentID,
			SessionID: sessionID,
			TaskID:    taskID,
			Blocks:    make([]*ThinkingBlock, 0),
			Insights:  make([]Insight, 0),
			Patterns:  make([]PatternMatch, 0),
			StartTime: time.Now(),
			Context:   make(map[string]interface{}),
			Tags:      make([]string, 0),
		},
		analyzer:         NewChainAnalyzer(),
		patternMatcher:   NewPatternMatcher(),
		qualityScorer:    NewQualityScorer(),
		insightExtractor: NewInsightExtractor(),
	}
}

// AddBlock adds a thinking block to the chain
func (b *ReasoningChainBuilder) AddBlock(block *ThinkingBlock) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add block
	b.chain.Blocks = append(b.chain.Blocks, block)

	// Update token count
	b.chain.TotalTokens += block.TokenCount

	// Extract insights from this block
	insights := b.insightExtractor.ExtractFromBlock(block)
	b.chain.Insights = append(b.chain.Insights, insights...)

	// Check for patterns
	if len(b.chain.Blocks) > 1 {
		patterns := b.patternMatcher.FindPatterns(b.chain.Blocks)
		b.chain.Patterns = patterns
	}

	return nil
}

// SetStrategy sets the reasoning strategy
func (b *ReasoningChainBuilder) SetStrategy(name, description string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chain.Strategy = ReasoningStrategy{
		Name:        name,
		Description: description,
		Adaptations: make([]StrategyAdaptation, 0),
	}
}

// AdaptStrategy records a strategy change
func (b *ReasoningChainBuilder) AdaptStrategy(reason, toStrategy string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	adaptation := StrategyAdaptation{
		Reason:       reason,
		FromStrategy: b.chain.Strategy.Name,
		ToStrategy:   toStrategy,
		AtStep:       len(b.chain.Blocks),
		Impact:       "unknown", // Will be evaluated later
	}

	b.chain.Strategy.Adaptations = append(b.chain.Strategy.Adaptations, adaptation)
	b.chain.Strategy.Name = toStrategy
}

// Build finalizes the chain with analysis
func (b *ReasoningChainBuilder) Build(ctx context.Context) (*ReasoningChainEnhanced, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set end time
	b.chain.EndTime = time.Now()

	// Calculate performance metrics
	b.chain.Performance = b.calculatePerformance()

	// Score quality
	quality, err := b.qualityScorer.Score(ctx, b.chain)
	if err != nil {
		logger := observability.GetLogger(ctx)
		logger.WarnContext(ctx, "Failed to score quality", "error", err)
		quality = b.defaultQualityMetrics()
	}
	b.chain.Quality = quality

	// Generate summary
	b.chain.Summary = b.generateSummary()

	// Calculate final confidence
	b.chain.FinalConfidence = b.calculateFinalConfidence()

	// Extract final insights
	finalInsights := b.insightExtractor.ExtractFromChain(b.chain)
	b.chain.Insights = append(b.chain.Insights, finalInsights...)

	// Identify improvements
	b.chain.Improvements = b.identifyImprovements()

	return b.chain, nil
}

// calculatePerformance computes performance metrics
func (b *ReasoningChainBuilder) calculatePerformance() PerformanceMetrics {
	duration := b.chain.EndTime.Sub(b.chain.StartTime)

	// Count decision blocks
	decisionCount := 0
	totalDecisionTime := time.Duration(0)
	backtrackCount := 0

	for _, block := range b.chain.Blocks {
		if block.Type == ThinkingTypeDecisionMaking {
			decisionCount++
			totalDecisionTime += block.Duration
		}

		// Count backtracks (blocks that reference previous errors)
		if block.ErrorContext != nil {
			backtrackCount++
		}
	}

	avgDecisionSpeed := time.Duration(0)
	if decisionCount > 0 {
		avgDecisionSpeed = totalDecisionTime / time.Duration(decisionCount)
	}

	tokensPerSecond := float64(b.chain.TotalTokens) / duration.Seconds()

	return PerformanceMetrics{
		ThinkingTime:    duration,
		TokensPerSecond: tokensPerSecond,
		DecisionSpeed:   avgDecisionSpeed,
		BacktrackCount:  backtrackCount,
		IterationCount:  len(b.chain.Blocks),
		ParallelPaths:   b.countParallelPaths(),
	}
}

// countParallelPaths counts independent reasoning paths
func (b *ReasoningChainBuilder) countParallelPaths() int {
	// Count blocks with no parent as independent paths
	paths := 0
	for _, block := range b.chain.Blocks {
		if block.ParentID == nil {
			paths++
		}
	}
	return paths
}

// generateSummary creates a concise summary of the reasoning
func (b *ReasoningChainBuilder) generateSummary() string {
	if len(b.chain.Blocks) == 0 {
		return "No reasoning recorded"
	}

	// Find key decisions and conclusions
	keyPoints := []string{}

	for _, block := range b.chain.Blocks {
		// Add decisions
		for _, dp := range block.DecisionPoints {
			if dp.Confidence > 0.7 {
				keyPoints = append(keyPoints, fmt.Sprintf("Decided: %s (%.0f%% confident)",
					dp.Decision, dp.Confidence*100))
			}
		}

		// Add conclusions from analysis
		if block.Type == ThinkingTypeAnalysis && block.StructuredData != nil {
			for _, conclusion := range block.StructuredData.Conclusions {
				keyPoints = append(keyPoints, fmt.Sprintf("Concluded: %s", conclusion))
			}
		}
	}

	// Add insights
	for _, insight := range b.chain.Insights {
		if insight.Confidence > 0.8 && insight.Actionable {
			keyPoints = append(keyPoints, fmt.Sprintf("Insight: %s", insight.Description))
		}
	}

	// Combine into summary
	summary := fmt.Sprintf("Reasoning chain with %d thinking blocks. ", len(b.chain.Blocks))
	if len(keyPoints) > 0 {
		summary += "Key points: " + joinStrings(keyPoints, "; ")
	}

	return summary
}

// calculateFinalConfidence computes overall confidence
func (b *ReasoningChainBuilder) calculateFinalConfidence() float64 {
	if len(b.chain.Blocks) == 0 {
		return 0
	}

	// Weighted average based on block importance
	totalWeight := 0.0
	weightedSum := 0.0

	for _, block := range b.chain.Blocks {
		weight := 1.0

		// Decision blocks are more important
		if block.Type == ThinkingTypeDecisionMaking {
			weight = 2.0
		}

		// Verification blocks validate confidence
		if block.Type == ThinkingTypeVerification {
			weight = 1.5
		}

		// Error recovery reduces confidence
		if block.Type == ThinkingTypeErrorRecovery {
			weight = 0.8
		}

		totalWeight += weight
		weightedSum += block.Confidence * weight
	}

	return weightedSum / totalWeight
}

// defaultQualityMetrics returns default quality scores
func (b *ReasoningChainBuilder) defaultQualityMetrics() QualityMetrics {
	return QualityMetrics{
		Coherence:    0.5,
		Completeness: 0.5,
		Depth:        0.5,
		Accuracy:     0.5,
		Innovation:   0.5,
		Overall:      0.5,
	}
}

// identifyImprovements suggests enhancements
func (b *ReasoningChainBuilder) identifyImprovements() []Improvement {
	improvements := []Improvement{}

	// Check for low confidence
	if b.chain.FinalConfidence < 0.6 {
		improvements = append(improvements, Improvement{
			Area:            "confidence",
			Description:     "Improve confidence through additional verification steps",
			Priority:        1,
			EstimatedImpact: 0.3,
			Implementation:  "Add verification blocks after key decisions",
		})
	}

	// Check for high backtrack count
	if b.chain.Performance.BacktrackCount > 2 {
		improvements = append(improvements, Improvement{
			Area:            "planning",
			Description:     "Reduce backtracks through better initial analysis",
			Priority:        2,
			EstimatedImpact: 0.25,
			Implementation:  "Enhance upfront problem analysis and constraint checking",
		})
	}

	// Check for shallow depth
	if b.chain.Quality.Depth < 0.5 {
		improvements = append(improvements, Improvement{
			Area:            "analysis",
			Description:     "Increase analysis depth for complex problems",
			Priority:        3,
			EstimatedImpact: 0.2,
			Implementation:  "Add multi-level analysis with sub-problems",
		})
	}

	return improvements
}

// AddContext adds context information to the chain
func (b *ReasoningChainBuilder) AddContext(key string, value interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chain.Context[key] = value
}

// AddTag adds a tag to the chain
func (b *ReasoningChainBuilder) AddTag(tag string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chain.Tags = append(b.chain.Tags, tag)
}

// SetCost sets the total cost of the reasoning
func (b *ReasoningChainBuilder) SetCost(cost float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chain.TotalCost = cost
}

// MarshalJSON implements custom JSON marshaling
func (rc *ReasoningChainEnhanced) MarshalJSON() ([]byte, error) {
	type Alias ReasoningChainEnhanced
	return json.Marshal(&struct {
		*Alias
		Duration string `json:"duration"`
	}{
		Alias:    (*Alias)(rc),
		Duration: rc.EndTime.Sub(rc.StartTime).String(),
	})
}

// joinStrings joins strings with a separator (helper function)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
