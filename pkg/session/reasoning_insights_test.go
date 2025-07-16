// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReasoningInsightGenerator(t *testing.T) {
	ctx := context.Background()
	generator := NewReasoningInsightGenerator(nil)

	t.Run("CalculateReasoningMetrics", func(t *testing.T) {
		analytics := &AnalyticsData{
			SessionID:    "test-session",
			Duration:     30 * time.Minute,
			MessageCount: 20,
			TokenUsage: TokenUsage{
				Total:            10000,
				Reasoning:        3000,
				ReasoningRatio:   0.3,
				ByAgent:          map[string]int{"agent1": 5000, "agent2": 5000},
				ReasoningByAgent: map[string]int{"agent1": 1500, "agent2": 1500},
			},
			TaskMetrics: TaskMetrics{
				TasksCreated:   10,
				TasksCompleted: 8,
				CompletionRate: 0.8,
			},
			ProductivityScore: 75.0,
			AgentUsage: map[string]AgentUsage{
				"agent1": {
					MessageCount: 10,
					SuccessRate:  0.9,
				},
				"agent2": {
					MessageCount: 10,
					SuccessRate:  0.8,
				},
			},
		}

		metrics, err := generator.CalculateReasoningMetrics(ctx, analytics)
		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// Check basic metrics
		assert.Equal(t, 3000, metrics.TotalReasoningTokens)
		assert.Greater(t, metrics.ReasoningEfficiency, 0.0)
		assert.LessOrEqual(t, metrics.ReasoningEfficiency, 1.0)
		assert.Greater(t, metrics.DecisionQuality, 0.0)
		assert.Greater(t, metrics.ReasoningDepth, 0.0)

		// Check agent reasoning styles
		assert.Len(t, metrics.AgentReasoningStyles, 2)
		assert.Contains(t, metrics.AgentReasoningStyles, "agent1")
		assert.Contains(t, metrics.AgentReasoningStyles, "agent2")
	})

	t.Run("GenerateReasoningInsights with high reasoning", func(t *testing.T) {
		analytics := &AnalyticsData{
			SessionID:    "test-session",
			Duration:     30 * time.Minute,
			MessageCount: 20,
			TokenUsage: TokenUsage{
				Total:          10000,
				Reasoning:      6000, // 60% reasoning
				ReasoningRatio: 0.6,
			},
			TaskMetrics: TaskMetrics{
				CompletionRate: 0.5, // Low completion
			},
			ProductivityScore: 40.0, // Low productivity
		}

		insights, err := generator.GenerateReasoningInsights(ctx, analytics)
		require.NoError(t, err)

		// Should generate efficiency insight
		found := false
		for _, insight := range insights {
			if insight.Type == InsightEfficiency && insight.Title == "Low Reasoning Efficiency" {
				found = true
				assert.Equal(t, InsightPriorityHigh, insight.Priority)
				assert.NotEmpty(t, insight.Actions)
			}
		}
		assert.True(t, found, "Should generate low efficiency insight")
	})

	t.Run("GenerateReasoningInsights with low decision quality", func(t *testing.T) {
		analytics := &AnalyticsData{
			SessionID:    "test-session",
			Duration:     30 * time.Minute,
			MessageCount: 20,
			TokenUsage: TokenUsage{
				Total:          10000,
				Reasoning:      2000,
				ReasoningRatio: 0.2,
			},
			TaskMetrics: TaskMetrics{
				TasksCreated:   10,
				TasksCompleted: 3,
				CompletionRate: 0.3, // Very low completion
			},
			ProductivityScore: 30.0,
			AgentUsage: map[string]AgentUsage{
				"agent1": {
					MessageCount: 10,
					SuccessRate:  0.4, // Low success
					ErrorCount:   6,
				},
			},
		}

		insights, err := generator.GenerateReasoningInsights(ctx, analytics)
		require.NoError(t, err)

		// Should generate decision quality insight
		found := false
		for _, insight := range insights {
			if insight.Type == InsightProductivity && insight.Title == "Decision Quality Below Target" {
				found = true
				assert.Equal(t, InsightPriorityMedium, insight.Priority)
			}
		}
		assert.True(t, found, "Should generate decision quality insight")
	})

	t.Run("CalculateReasoningEfficiency", func(t *testing.T) {
		tests := []struct {
			name      string
			analytics *AnalyticsData
			minEff    float64
			maxEff    float64
		}{
			{
				name: "high efficiency",
				analytics: &AnalyticsData{
					TokenUsage: TokenUsage{
						Total:          10000,
						ReasoningRatio: 0.1, // Low reasoning
					},
					ProductivityScore: 90.0, // High productivity
					TaskMetrics: TaskMetrics{
						CompletionRate: 0.95,
					},
				},
				minEff: 0.8,
				maxEff: 1.0,
			},
			{
				name: "low efficiency",
				analytics: &AnalyticsData{
					TokenUsage: TokenUsage{
						Total:          10000,
						ReasoningRatio: 0.8, // High reasoning
					},
					ProductivityScore: 30.0, // Low productivity
					TaskMetrics: TaskMetrics{
						CompletionRate: 0.2,
					},
				},
				minEff: 0.0,
				maxEff: 0.3,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				efficiency := generator.calculateReasoningEfficiency(tt.analytics)
				assert.GreaterOrEqual(t, efficiency, tt.minEff)
				assert.LessOrEqual(t, efficiency, tt.maxEff)
			})
		}
	})

	t.Run("GenerateReasoningTimeline", func(t *testing.T) {
		messages := []Message{
			{
				ID:        "msg1",
				Agent:     "agent1",
				Type:      MessageTypeAgent,
				Content:   "Test message 1",
				Timestamp: time.Now().Add(-30 * time.Minute),
				Metadata: map[string]interface{}{
					"tokens":           1000,
					"reasoning_tokens": 200,
				},
			},
			{
				ID:        "msg2",
				Agent:     "agent1",
				Type:      MessageTypeAgent,
				Content:   "Test message 2",
				Timestamp: time.Now().Add(-20 * time.Minute),
				Metadata: map[string]interface{}{
					"tokens":           2000,
					"reasoning_tokens": 1000,
				},
			},
			{
				ID:        "msg3",
				Agent:     "agent2",
				Type:      MessageTypeAgent,
				Content:   "Test message 3",
				Timestamp: time.Now().Add(-10 * time.Minute),
				Metadata: map[string]interface{}{
					"tokens": 500,
					// No reasoning tokens
				},
			},
		}

		timeline, err := generator.GenerateReasoningTimeline(ctx, messages)
		require.NoError(t, err)
		assert.Len(t, timeline, 2) // Only messages with reasoning tokens

		// Check timeline is sorted
		assert.True(t, timeline[0].Timestamp.Before(timeline[1].Timestamp))

		// Check reasoning ratios
		assert.InDelta(t, 0.2, timeline[0].ReasoningRatio, 0.01) // 200/1000
		assert.InDelta(t, 0.5, timeline[1].ReasoningRatio, 0.01) // 1000/2000
	})

	t.Run("GenerateReasoningHeatmap", func(t *testing.T) {
		analytics := &AnalyticsData{
			SessionID: "test-session",
			TokenUsage: TokenUsage{
				Total:          10000,
				Reasoning:      3000,
				ReasoningRatio: 0.3,
			},
		}

		heatmap, err := generator.GenerateReasoningHeatmap(ctx, analytics)
		require.NoError(t, err)
		assert.NotNil(t, heatmap)

		assert.Equal(t, "reasoning_intensity", heatmap["type"])
		assert.NotNil(t, heatmap["data"])

		// Check data structure
		data, ok := heatmap["data"].([]map[string]interface{})
		assert.True(t, ok)
		assert.NotEmpty(t, data)
	})
}
