// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents/core"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ReasoningMetrics contains comprehensive reasoning analytics
type ReasoningMetrics struct {
	TotalReasoningTokens int                       `json:"total_reasoning_tokens"`
	ReasoningDepth       float64                   `json:"reasoning_depth"`
	DecisionQuality      float64                   `json:"decision_quality"`
	PatternComplexity    float64                   `json:"pattern_complexity"`
	ReasoningEfficiency  float64                   `json:"reasoning_efficiency"`
	TimeDistribution     []ReasoningTimePoint      `json:"time_distribution"`
	PatternBreakdown     map[string]PatternMetric  `json:"pattern_breakdown"`
	QualityTrend         []DataPoint               `json:"quality_trend"`
	AgentReasoningStyles map[string]ReasoningStyle `json:"agent_reasoning_styles"`
}

// ReasoningTimePoint represents reasoning usage at a point in time
type ReasoningTimePoint struct {
	Timestamp      time.Time `json:"timestamp"`
	ReasoningRatio float64   `json:"reasoning_ratio"`
	Quality        float64   `json:"quality"`
	Depth          int       `json:"depth"`
}

// PatternMetric contains metrics for a specific reasoning pattern
type PatternMetric struct {
	Pattern        string  `json:"pattern"`
	Frequency      int     `json:"frequency"`
	AverageQuality float64 `json:"average_quality"`
	SuccessRate    float64 `json:"success_rate"`
}

// ReasoningStyle describes an agent's reasoning approach
type ReasoningStyle struct {
	AgentID           string   `json:"agent_id"`
	PreferredPatterns []string `json:"preferred_patterns"`
	AverageDepth      float64  `json:"average_depth"`
	ConsistencyScore  float64  `json:"consistency_score"`
	AdaptabilityScore float64  `json:"adaptability_score"`
}

// ReasoningInsightGenerator generates insights from reasoning data
type ReasoningInsightGenerator struct {
	analyzer  *core.DefaultReasoningAnalyzer
	extractor *reasoningMetricsExtractor
}

// reasoningMetricsExtractor extracts metrics from reasoning data
type reasoningMetricsExtractor struct{}

// NewReasoningInsightGenerator creates a new reasoning insight generator
func NewReasoningInsightGenerator(analyzer *core.DefaultReasoningAnalyzer) *ReasoningInsightGenerator {
	return &ReasoningInsightGenerator{
		analyzer:  analyzer,
		extractor: &reasoningMetricsExtractor{},
	}
}

// GenerateReasoningInsights generates insights from analytics data
func (rig *ReasoningInsightGenerator) GenerateReasoningInsights(ctx context.Context, analytics *AnalyticsData) ([]Insight, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	var insights []Insight

	// Analyze reasoning efficiency
	if analytics.TokenUsage.ReasoningRatio > 0 {
		efficiency := rig.calculateReasoningEfficiency(analytics)
		if efficiency < 0.5 {
			insights = append(insights, Insight{
				Type:     InsightEfficiency,
				Title:    "Low Reasoning Efficiency",
				Message:  fmt.Sprintf("Reasoning efficiency is %.0f%%. Consider more focused prompts.", efficiency*100),
				Priority: InsightPriorityHigh,
				Actions: []Action{
					{
						Title:       "Optimize Prompts",
						Description: "Use clearer, more specific instructions",
						Type:        "optimization",
					},
					{
						Title:       "Review Reasoning Patterns",
						Description: "Check for unnecessary reasoning loops",
						Type:        "analysis",
					},
				},
			})
		}
	}

	// Analyze decision quality
	qualityScore := rig.calculateDecisionQuality(analytics)
	if qualityScore < 0.7 {
		insights = append(insights, Insight{
			Type:     InsightProductivity,
			Title:    "Decision Quality Below Target",
			Message:  fmt.Sprintf("Decision quality score is %.0f%%. Review reasoning patterns.", qualityScore*100),
			Priority: InsightPriorityMedium,
			Actions: []Action{
				{
					Title:       "Enhance Context",
					Description: "Provide more relevant context for decisions",
					Type:        "enhancement",
				},
			},
		})
	}

	// Analyze reasoning patterns
	patternInsights := rig.analyzeReasoningPatterns(analytics)
	insights = append(insights, patternInsights...)

	return insights, nil
}

// CalculateReasoningMetrics calculates comprehensive reasoning metrics
func (rig *ReasoningInsightGenerator) CalculateReasoningMetrics(ctx context.Context, analytics *AnalyticsData) (*ReasoningMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	metrics := &ReasoningMetrics{
		TotalReasoningTokens: analytics.TokenUsage.Reasoning,
		ReasoningEfficiency:  rig.calculateReasoningEfficiency(analytics),
		DecisionQuality:      rig.calculateDecisionQuality(analytics),
		PatternComplexity:    rig.calculatePatternComplexity(analytics),
		PatternBreakdown:     make(map[string]PatternMetric),
		AgentReasoningStyles: make(map[string]ReasoningStyle),
	}

	// Calculate reasoning depth
	metrics.ReasoningDepth = rig.calculateReasoningDepth(analytics)

	// Generate time distribution
	metrics.TimeDistribution = rig.generateTimeDistribution(analytics)

	// Analyze agent reasoning styles
	for agentID, usage := range analytics.AgentUsage {
		style := rig.analyzeAgentReasoningStyle(agentID, usage, analytics)
		metrics.AgentReasoningStyles[agentID] = style
	}

	return metrics, nil
}

// calculateReasoningEfficiency calculates efficiency of reasoning usage
func (rig *ReasoningInsightGenerator) calculateReasoningEfficiency(analytics *AnalyticsData) float64 {
	if analytics.TokenUsage.Total == 0 {
		return 1.0
	}

	// Base efficiency on ratio and productivity
	reasoningRatio := analytics.TokenUsage.ReasoningRatio
	productivity := analytics.ProductivityScore / 100.0

	// Lower reasoning ratio with high productivity = high efficiency
	// High reasoning ratio with low productivity = low efficiency
	efficiency := (1.0 - reasoningRatio) * productivity

	// Adjust based on task completion
	if analytics.TaskMetrics.CompletionRate > 0 {
		efficiency = efficiency*0.7 + analytics.TaskMetrics.CompletionRate*0.3
	}

	return math.Max(0, math.Min(1, efficiency))
}

// calculateDecisionQuality estimates quality of decisions made
func (rig *ReasoningInsightGenerator) calculateDecisionQuality(analytics *AnalyticsData) float64 {
	quality := 0.0
	factors := 0

	// Factor 1: Task completion rate
	if analytics.TaskMetrics.TasksCreated > 0 {
		quality += analytics.TaskMetrics.CompletionRate
		factors++
	}

	// Factor 2: Agent success rates
	totalSuccess := 0.0
	agentCount := 0
	for _, usage := range analytics.AgentUsage {
		totalSuccess += usage.SuccessRate
		agentCount++
	}
	if agentCount > 0 {
		quality += totalSuccess / float64(agentCount)
		factors++
	}

	// Factor 3: Error rate (inverse)
	totalMessages := analytics.MessageCount
	totalErrors := 0
	for _, usage := range analytics.AgentUsage {
		totalErrors += usage.ErrorCount
	}
	if totalMessages > 0 {
		errorRate := float64(totalErrors) / float64(totalMessages)
		quality += (1.0 - errorRate)
		factors++
	}

	if factors == 0 {
		return 0.5 // Default to neutral
	}

	return quality / float64(factors)
}

// calculatePatternComplexity calculates complexity of reasoning patterns
func (rig *ReasoningInsightGenerator) calculatePatternComplexity(analytics *AnalyticsData) float64 {
	// Simple heuristic based on command diversity and reasoning ratio
	commandDiversity := float64(len(analytics.CommandUsage))
	reasoningIntensity := analytics.TokenUsage.ReasoningRatio

	// More diverse commands with moderate reasoning = higher complexity
	complexity := math.Min(1.0, commandDiversity/10.0) * (0.5 + reasoningIntensity*0.5)

	return complexity
}

// calculateReasoningDepth estimates depth of reasoning chains
func (rig *ReasoningInsightGenerator) calculateReasoningDepth(analytics *AnalyticsData) float64 {
	if analytics.TokenUsage.Reasoning == 0 {
		return 0
	}

	// Estimate depth based on reasoning tokens per message
	avgReasoningPerMessage := float64(analytics.TokenUsage.Reasoning) / float64(analytics.MessageCount)

	// Map tokens to estimated depth (heuristic)
	// <100 tokens = shallow (1-2 levels)
	// 100-500 tokens = moderate (2-4 levels)
	// >500 tokens = deep (4+ levels)
	if avgReasoningPerMessage < 100 {
		return 1.0 + avgReasoningPerMessage/100.0
	} else if avgReasoningPerMessage < 500 {
		return 2.0 + (avgReasoningPerMessage-100)/200.0
	}
	return 4.0 + math.Log10(avgReasoningPerMessage/500.0)
}

// generateTimeDistribution creates time-based reasoning distribution
func (rig *ReasoningInsightGenerator) generateTimeDistribution(analytics *AnalyticsData) []ReasoningTimePoint {
	// For now, return a simple distribution
	// In production, this would analyze message timestamps
	points := []ReasoningTimePoint{
		{
			Timestamp:      time.Now().Add(-time.Hour),
			ReasoningRatio: 0.2,
			Quality:        0.8,
			Depth:          2,
		},
		{
			Timestamp:      time.Now().Add(-30 * time.Minute),
			ReasoningRatio: analytics.TokenUsage.ReasoningRatio,
			Quality:        rig.calculateDecisionQuality(analytics),
			Depth:          int(rig.calculateReasoningDepth(analytics)),
		},
	}

	return points
}

// analyzeAgentReasoningStyle analyzes an agent's reasoning approach
func (rig *ReasoningInsightGenerator) analyzeAgentReasoningStyle(agentID string, usage AgentUsage, analytics *AnalyticsData) ReasoningStyle {
	// Calculate reasoning tokens for this agent
	reasoningTokens := 0
	if tokens, ok := analytics.TokenUsage.ReasoningByAgent[agentID]; ok {
		reasoningTokens = tokens
	}

	totalTokens := 0
	if tokens, ok := analytics.TokenUsage.ByAgent[agentID]; ok {
		totalTokens = tokens
	}

	averageDepth := 0.0
	if totalTokens > 0 && usage.MessageCount > 0 {
		avgReasoningPerMessage := float64(reasoningTokens) / float64(usage.MessageCount)
		averageDepth = rig.estimateDepthFromTokens(avgReasoningPerMessage)
	}

	return ReasoningStyle{
		AgentID:           agentID,
		PreferredPatterns: []string{"analytical", "systematic"}, // Placeholder
		AverageDepth:      averageDepth,
		ConsistencyScore:  usage.SuccessRate,
		AdaptabilityScore: 0.75, // Placeholder
	}
}

// estimateDepthFromTokens estimates reasoning depth from token count
func (rig *ReasoningInsightGenerator) estimateDepthFromTokens(tokens float64) float64 {
	if tokens < 50 {
		return 1.0
	} else if tokens < 200 {
		return 2.0
	} else if tokens < 500 {
		return 3.0
	}
	return 4.0 + math.Log10(tokens/500.0)
}

// analyzeReasoningPatterns analyzes patterns in reasoning
func (rig *ReasoningInsightGenerator) analyzeReasoningPatterns(analytics *AnalyticsData) []Insight {
	var insights []Insight

	// Check for reasoning imbalance across agents
	maxRatio := 0.0
	minRatio := 1.0
	for agentID, totalTokens := range analytics.TokenUsage.ByAgent {
		if totalTokens > 0 {
			reasoningTokens := analytics.TokenUsage.ReasoningByAgent[agentID]
			ratio := float64(reasoningTokens) / float64(totalTokens)
			if ratio > maxRatio {
				maxRatio = ratio
			}
			if ratio < minRatio {
				minRatio = ratio
			}
		}
	}

	if maxRatio-minRatio > 0.5 {
		insights = append(insights, Insight{
			Type:     InsightUsage,
			Title:    "Reasoning Imbalance Detected",
			Message:  "Some agents use significantly more reasoning than others. Consider balancing workload.",
			Priority: InsightPriorityLow,
		})
	}

	return insights
}

// GenerateReasoningTimeline creates a timeline of reasoning events
func (rig *ReasoningInsightGenerator) GenerateReasoningTimeline(ctx context.Context, messages []Message) ([]ReasoningTimePoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	var timeline []ReasoningTimePoint

	for _, msg := range messages {
		if reasoningTokens, ok := msg.Metadata["reasoning_tokens"].(int); ok && reasoningTokens > 0 {
			totalTokens := 0
			if tokens, ok := msg.Metadata["tokens"].(int); ok {
				totalTokens = tokens
			}

			ratio := 0.0
			if totalTokens > 0 {
				ratio = float64(reasoningTokens) / float64(totalTokens)
			}

			point := ReasoningTimePoint{
				Timestamp:      msg.Timestamp,
				ReasoningRatio: ratio,
				Quality:        0.8, // Placeholder - would calculate from actual metrics
				Depth:          int(rig.estimateDepthFromTokens(float64(reasoningTokens))),
			}

			timeline = append(timeline, point)
		}
	}

	// Sort by timestamp
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})

	return timeline, nil
}

// GenerateReasoningHeatmap creates heatmap data for reasoning visualization
func (rig *ReasoningInsightGenerator) GenerateReasoningHeatmap(ctx context.Context, analytics *AnalyticsData) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	heatmap := map[string]interface{}{
		"type": "reasoning_intensity",
		"data": []map[string]interface{}{},
	}

	// Generate hourly buckets for the session duration
	buckets := make(map[int]float64) // hour -> reasoning intensity

	// Simulate data (in production, would analyze actual message timestamps)
	for hour := 0; hour < 24; hour++ {
		intensity := 0.0
		if hour >= 9 && hour <= 17 { // Business hours
			intensity = 0.3 + math.Sin(float64(hour-9)*math.Pi/8)*0.4
		}
		buckets[hour] = intensity
	}

	// Convert to heatmap format
	for hour, intensity := range buckets {
		heatmap["data"] = append(heatmap["data"].([]map[string]interface{}), map[string]interface{}{
			"hour":      hour,
			"intensity": intensity,
			"label":     fmt.Sprintf("%02d:00", hour),
		})
	}

	return heatmap, nil
}
