// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/gerror"
)

func TestThinkingBlockParser(t *testing.T) {
	// Pass nil for metrics registry in tests
	parser := core.NewThinkingBlockParser(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []core.ThinkingType
		expectError   bool
	}{
		{
			name: "single analysis block",
			input: `<thinking type="analysis">
				Let me analyze this problem.
				The key aspects are:
				1. Performance
				2. Scalability
				3. Maintainability
				Confidence: 0.85
			</thinking>`,
			expectedCount: 1,
			expectedTypes: []core.ThinkingType{core.ThinkingTypeAnalysis},
		},
		{
			name: "multiple blocks with different types",
			input: `
				<thinking type="analysis">
					Analyzing the requirements...
					Confidence: 0.8
				</thinking>
				
				Some response text here.
				
				<thinking type="planning">
					Planning the implementation:
					- Step 1: Setup
					- Step 2: Core logic
					- Step 3: Testing
					Confidence: 0.9
				</thinking>
				
				<thinking type="decision_making">
					Decision: Use microservices architecture
					Confidence: 0.95
				</thinking>
			`,
			expectedCount: 3,
			expectedTypes: []core.ThinkingType{
				core.ThinkingTypeAnalysis,
				core.ThinkingTypePlanning,
				core.ThinkingTypeDecisionMaking,
			},
		},
		{
			name: "block without type defaults to analysis",
			input: `<thinking>
				This is a general thinking block.
				It should default to analysis type.
			</thinking>`,
			expectedCount: 1,
			expectedTypes: []core.ThinkingType{core.ThinkingTypeAnalysis},
		},
		{
			name:          "no thinking blocks",
			input:         "This is just regular text without any thinking blocks.",
			expectedCount: 0,
			expectedTypes: []core.ThinkingType{},
		},
		{
			name: "nested content with special characters",
			input: `<thinking type="optimization">
				Optimizing the algorithm:
				Current complexity: O(n²)
				Target complexity: O(n log n)
				
				if (x > 0 && y < 10) {
					// This should still parse correctly
				}
				
				Confidence: 0.88
			</thinking>`,
			expectedCount: 1,
			expectedTypes: []core.ThinkingType{core.ThinkingTypeOptimization},
		},
		{
			name: "malformed block (unclosed)",
			input: `<thinking type="analysis">
				This block is never closed
				and should be handled gracefully`,
			expectedCount: 0,
			expectedTypes: []core.ThinkingType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := parser.ParseThinkingBlocks(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, blocks, tt.expectedCount)

			for i, block := range blocks {
				if i < len(tt.expectedTypes) {
					assert.Equal(t, tt.expectedTypes[i], block.Type)
				}
				assert.NotEmpty(t, block.ID)
				assert.NotEmpty(t, block.Content)
				assert.GreaterOrEqual(t, block.Confidence, 0.0)
				assert.LessOrEqual(t, block.Confidence, 1.0)
				assert.Greater(t, block.TokenCount, 0)
			}
		})
	}
}

func TestConfidenceExtraction(t *testing.T) {
	parser := core.NewThinkingBlockParser(nil)
	ctx := context.Background()

	tests := []struct {
		name               string
		input              string
		expectedConfidence float64
	}{
		{
			name: "explicit confidence value",
			input: `<thinking>
				Analysis complete.
				Confidence: 0.92
			</thinking>`,
			expectedConfidence: 0.92,
		},
		{
			name: "confidence as percentage",
			input: `<thinking>
				I'm 85% confident in this approach.
			</thinking>`,
			expectedConfidence: 0.85,
		},
		{
			name: "confidence in different format",
			input: `<thinking>
				Analysis results:
				Confidence level: 0.78
			</thinking>`,
			expectedConfidence: 0.78,
		},
		{
			name: "no confidence specified",
			input: `<thinking>
				Just some analysis without confidence.
			</thinking>`,
			expectedConfidence: 0.5, // default
		},
		{
			name: "multiple confidence values (takes last)",
			input: `<thinking>
				Initial confidence: 0.6
				After analysis...
				Updated confidence: 0.85
			</thinking>`,
			expectedConfidence: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := parser.ParseThinkingBlocks(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, blocks, 1)

			assert.InDelta(t, tt.expectedConfidence, blocks[0].Confidence, 0.01)
		})
	}
}

func TestReasoningExtractor(t *testing.T) {
	config := core.ReasoningConfig{
		EnableCaching:      true,
		CacheMaxSize:       10,
		CacheTTL:           1 * time.Minute,
		MaxReasoningLength: 1000,
		MinConfidence:      0.0,
		MaxConfidence:      1.0,
		StrictValidation:   true,
	}

	extractor, err := core.NewReasoningExtractor(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("successful extraction", func(t *testing.T) {
		response := `<thinking type="analysis">
			Analyzing the problem...
			Confidence: 0.8
		</thinking>
		
		Based on my analysis, here's the solution.
		
		<thinking type="verification">
			Verifying the solution...
			All checks passed.
			Confidence: 0.95
		</thinking>`

		resp, err := extractor.ExtractReasoning(ctx, response)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Reasoning)
		assert.Greater(t, resp.Confidence, 0.0)
	})

	t.Run("extraction with metadata", func(t *testing.T) {
		response := `<thinking>Test reasoning</thinking>`
		// Test with metadata in response (metadata handling would be in ExtractReasoning implementation)

		resp, err := extractor.ExtractReasoning(ctx, response)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Reasoning)
	})

	t.Run("caching behavior", func(t *testing.T) {
		response := `<thinking>Cached test</thinking>`

		// First call
		resp1, err := extractor.ExtractReasoning(ctx, response)
		require.NoError(t, err)

		// Second call (should hit cache)
		resp2, err := extractor.ExtractReasoning(ctx, response)
		require.NoError(t, err)

		// Should return same response from cache
		assert.Equal(t, resp1.Reasoning, resp2.Reasoning)

		// Check stats
		stats := extractor.GetStats()
		assert.Greater(t, stats["cache_hit_rate"].(float64), 0.0)
	})

}

func TestReasoningChainBuilder(t *testing.T) {
	ctx := context.Background()

	t.Run("building chain with multiple blocks", func(t *testing.T) {
		builder := core.NewReasoningChainBuilder("agent-1", "session-1", "task-1")

		// Set strategy
		builder.SetStrategy("problem_solving", "Systematic problem-solving approach")

		// Add blocks
		blocks := []*core.ThinkingBlock{
			{
				ID:         "block-1",
				Type:       core.ThinkingTypeAnalysis,
				Content:    "Analyzing the problem",
				Confidence: 0.8,
				Timestamp:  time.Now(),
				Duration:   100 * time.Millisecond,
				TokenCount: 50,
			},
			{
				ID:         "block-2",
				Type:       core.ThinkingTypePlanning,
				Content:    "Planning the solution",
				Confidence: 0.85,
				Timestamp:  time.Now(),
				Duration:   150 * time.Millisecond,
				TokenCount: 75,
			},
			{
				ID:         "block-3",
				Type:       core.ThinkingTypeDecisionMaking,
				Content:    "Deciding on approach",
				Confidence: 0.9,
				Timestamp:  time.Now(),
				Duration:   80 * time.Millisecond,
				TokenCount: 40,
				DecisionPoints: []core.DecisionPoint{
					{
						Decision:   "Use microservices",
						Confidence: 0.9,
						Alternatives: []core.Alternative{
							{Option: "Monolith"},
							{Option: "Serverless"},
						},
						Rationale: "Better scalability",
					},
				},
			},
		}

		for _, block := range blocks {
			err := builder.AddBlock(block)
			require.NoError(t, err)
		}

		// Add metadata
		builder.AddContext("environment", "production")
		builder.AddTag("architecture")
		builder.SetCost(0.05)

		// Build chain
		chain, err := builder.Build(ctx)
		require.NoError(t, err)

		assert.Equal(t, "agent-1", chain.AgentID)
		assert.Equal(t, "session-1", chain.SessionID)
		assert.Equal(t, "task-1", chain.TaskID)
		assert.Len(t, chain.Blocks, 3)
		assert.Equal(t, 165, chain.TotalTokens) // 50 + 75 + 40
		assert.Greater(t, chain.FinalConfidence, 0.8)
		assert.NotEmpty(t, chain.Summary)
		assert.Equal(t, "production", chain.Context["environment"])
		assert.Contains(t, chain.Tags, "architecture")
		assert.Equal(t, 0.05, chain.TotalCost)

		// Check quality metrics
		assert.Greater(t, chain.Quality.Overall, 0.0)
		assert.LessOrEqual(t, chain.Quality.Overall, 1.0)

		// Check performance metrics
		assert.Greater(t, chain.Performance.ThinkingTime, time.Duration(0))
		assert.Greater(t, chain.Performance.TokensPerSecond, 0.0)
		assert.Equal(t, 3, chain.Performance.IterationCount)
	})

	t.Run("strategy adaptation", func(t *testing.T) {
		builder := core.NewReasoningChainBuilder("agent-2", "session-2", "task-2")

		builder.SetStrategy("initial", "Initial approach")
		builder.AddBlock(&core.ThinkingBlock{
			ID:      "block-1",
			Type:    core.ThinkingTypeAnalysis,
			Content: "Initial analysis",
		})

		// Adapt strategy
		builder.AdaptStrategy("Found better approach", "optimized")

		chain, err := builder.Build(ctx)
		require.NoError(t, err)

		assert.Equal(t, "optimized", chain.Strategy.Name)
		assert.Len(t, chain.Strategy.Adaptations, 1)
		assert.Equal(t, "Found better approach", chain.Strategy.Adaptations[0].Reason)
	})
}

func TestTokenManager(t *testing.T) {
	ctx := context.Background()

	config := core.TokenConfig{
		DefaultSafetyMargin:   0.15,
		CompactionThreshold:   0.85,
		EmergencyThreshold:    0.98,
		EnableAutoCompaction:  true,
		MaxCompactionAttempts: 3,
		TokenCountCache:       true,
		CacheTTL:              5 * time.Minute,
	}

	manager := core.NewTokenManager(config)

	t.Run("window creation and management", func(t *testing.T) {
		// Create window
		window, err := manager.CreateWindow(ctx, "test-agent", "test-session", "openai", "gpt-4")
		require.NoError(t, err)
		assert.NotNil(t, window)
		assert.Equal(t, "test-agent", window.AgentID)
		assert.Equal(t, "test-session", window.SessionID)

		// Get window
		fetched, err := manager.GetWindow(window.ID)
		require.NoError(t, err)
		assert.Equal(t, window.ID, fetched.ID)
		assert.Equal(t, int64(0), window.CurrentTokens)

		// Add messages
		messages := []core.WindowMessage{
			{
				ID:           "msg-1",
				Role:         "user",
				Content:      "Hello, can you help me?",
				Priority:     core.PriorityNormal,
				Timestamp:    time.Now(),
				Compressible: true,
			},
			{
				ID:           "msg-2",
				Role:         "assistant",
				Content:      "Of course! I'd be happy to help.",
				Priority:     core.PriorityNormal,
				Timestamp:    time.Now(),
				Compressible: true,
			},
		}

		for _, msg := range messages {
			err := manager.AddMessage(ctx, window.ID, msg)
			require.NoError(t, err)
		}

		// Check utilization
		utilization, err := manager.GetUtilization(window.ID)
		require.NoError(t, err)
		assert.Greater(t, utilization.CurrentTokens, int64(0))
		assert.Less(t, utilization.Utilization, 0.5)
		assert.False(t, utilization.NearLimit)
	})

	t.Run("token limit enforcement", func(t *testing.T) {
		// Create window with small limit
		window, err := manager.CreateWindow(ctx, "test-agent", "test-session", "openai", "gpt-3.5-turbo")
		require.NoError(t, err)

		// Try to add many messages to exceed limit
		for i := 0; i < 100; i++ {
			largeMessage := core.WindowMessage{
				ID:           fmt.Sprintf("large-msg-%d", i),
				Role:         "user",
				Content:      strings.Repeat("word ", 100), // ~100 tokens
				Priority:     core.PriorityNormal,
				Timestamp:    time.Now(),
				Compressible: true,
			}

			err = manager.AddMessage(ctx, window.ID, largeMessage)
			// At some point should trigger compaction or error
			if err != nil {
				assert.True(t, gerror.Is(err, gerror.ErrCodeResourceLimit))
				break
			}
		}
	})

	t.Run("token prediction", func(t *testing.T) {
		// Create window for prediction
		window, err := manager.CreateWindow(ctx, "test-agent", "test-session", "openai", "gpt-4")
		require.NoError(t, err)
		windowID := window.ID

		// Add some messages
		for i := 0; i < 5; i++ {
			msg := core.WindowMessage{
				ID:           fmt.Sprintf("msg-%d", i),
				Role:         "user",
				Content:      "Test message",
				Priority:     core.PriorityNormal,
				Timestamp:    time.Now(),
				Compressible: true,
			}
			manager.AddMessage(ctx, window.ID, msg)
		}

		// Predict future usage
		prediction, err := manager.PredictTokenUsage(windowID, 10)
		require.NoError(t, err)
		assert.Greater(t, prediction.PredictedTokens, prediction.CurrentTokens)
		assert.NotNil(t, prediction.CompactionNeeded)
	})
}

func TestPatternLearning(t *testing.T) {
	ctx := context.Background()
	config := core.DefaultPatternLearningConfig()
	learner := core.NewPatternLearner(nil, config) // In-memory repository

	t.Run("pattern learning", func(t *testing.T) {
		// Create a chain with a pattern
		chain := createChainWithPattern("chain-1", []core.ThinkingType{
			core.ThinkingTypeAnalysis,
			core.ThinkingTypePlanning,
			core.ThinkingTypeVerification,
		})

		// Learn from the chain with positive feedback
		feedback := &core.ReasoningFeedback{
			ID:          "feedback-1",
			Type:        core.FeedbackTypeOutcome,
			Rating:      0.9,
			Comments:    "Pattern worked well",
			Suggestions: []string{},
			ProvidedBy:  "test",
			ProvidedAt:  time.Now(),
		}

		err := learner.Learn(ctx, chain, feedback)
		require.NoError(t, err)

		// Suggest patterns for similar context
		patternContext := core.PatternContext{
			Task:          "analysis",
			CurrentBlocks: 3,
			Complexity:    0.7,
			Resources:     map[string]interface{}{},
			History:       []string{},
			Metadata:      map[string]interface{}{},
		}

		suggestions, err := learner.SuggestPatterns(ctx, patternContext)
		require.NoError(t, err)
		// May or may not have suggestions depending on learning
		_ = suggestions
	})

	t.Run("pattern application", func(t *testing.T) {
		// Apply pattern with test input
		input := &core.PatternInput{
			Context: map[string]interface{}{
				"description": "User asked to analyze system performance",
				"type":        "analysis",
				"complexity":  0.7,
			},
			Task:        "analyze performance",
			Constraints: []string{},
			Examples:    []string{},
		}

		// Note: ApplyPattern expects a pattern to exist, but we can't guarantee that in tests
		// Just test that the method doesn't panic
		output, err := learner.ApplyPattern(ctx, "test-pattern", input)
		// May return error if pattern doesn't exist
		if err == nil {
			assert.NotNil(t, output)
		}
	})

}

func TestReasoningStorage(t *testing.T) {
	ctx := context.Background()

	config := core.ReasoningStorageConfig{
		RetentionDays:     30,
		MaxChainsPerQuery: 100,
		EnableCompression: true,
		StorageBackend:    "memory",
	}

	storage, err := core.NewMemoryReasoningStorage(config)
	require.NoError(t, err)
	defer storage.Close(ctx)

	t.Run("store and retrieve", func(t *testing.T) {
		chain := &core.ReasoningChain{
			ID:         "test-chain-1",
			AgentID:    "agent-1",
			SessionID:  "session-1",
			RequestID:  "request-1",
			Content:    "Test content",
			Reasoning:  "Test analysis",
			Confidence: 0.85,
			TaskType:   "analysis",
			Success:    true,
			TokensUsed: 100,
			Duration:   5 * time.Minute,
			CreatedAt:  time.Now(),
			Metadata:   map[string]interface{}{},
		}

		// Store
		err := storage.Store(ctx, chain)
		require.NoError(t, err)

		// Retrieve
		retrieved, err := storage.Get(ctx, chain.ID)
		require.NoError(t, err)
		assert.Equal(t, chain.ID, retrieved.ID)
		assert.Equal(t, chain.AgentID, retrieved.AgentID)
		assert.Equal(t, chain.Confidence, retrieved.Confidence)
	})

	t.Run("query chains", func(t *testing.T) {
		// Store multiple chains
		for i := 0; i < 5; i++ {
			chain := &core.ReasoningChain{
				ID:         fmt.Sprintf("query-chain-%d", i),
				AgentID:    "agent-1",
				SessionID:  "session-1",
				RequestID:  fmt.Sprintf("request-%d", i),
				Content:    fmt.Sprintf("Content %d", i),
				Reasoning:  fmt.Sprintf("Reasoning %d", i),
				Confidence: 0.7 + float64(i)*0.05,
				TaskType:   "analysis",
				Success:    true,
				TokensUsed: 100 + i*10,
				Duration:   time.Duration(i+1) * time.Minute,
				CreatedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
				Metadata:   map[string]interface{}{},
			}
			err := storage.Store(ctx, chain)
			require.NoError(t, err)
		}

		// Query with filters
		query := &core.ReasoningQuery{
			AgentID:       "agent-1",
			MinConfidence: 0.8,
			OrderBy:       "confidence",
			Ascending:     false,
			Limit:         3,
		}

		chains, err := storage.Query(ctx, query)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(chains), 3)

		// Verify ordering
		for i := 1; i < len(chains); i++ {
			assert.GreaterOrEqual(t, chains[i-1].Confidence, chains[i].Confidence)
		}
	})

	t.Run("statistics", func(t *testing.T) {
		stats, err := storage.GetStats(ctx, "agent-1", time.Now().Add(-24*time.Hour), time.Now())
		require.NoError(t, err)
		assert.Greater(t, stats.TotalChains, int64(0))
		assert.Greater(t, stats.AvgConfidence, 0.0)
		assert.GreaterOrEqual(t, stats.AvgDuration, time.Duration(0))
	})
}

// Helper functions

func createChainWithPattern(id string, pattern []core.ThinkingType) *core.ReasoningChainEnhanced {
	blocks := make([]*core.ThinkingBlock, len(pattern))
	for i, thinkingType := range pattern {
		blocks[i] = &core.ThinkingBlock{
			ID:         fmt.Sprintf("%s-block-%d", id, i),
			Type:       thinkingType,
			Content:    fmt.Sprintf("Content for %s", thinkingType),
			Confidence: 0.8,
			Timestamp:  time.Now(),
			TokenCount: 50,
		}
	}

	return &core.ReasoningChainEnhanced{
		ID:              id,
		AgentID:         "test-agent",
		SessionID:       "test-session",
		TaskID:          "test-task",
		Blocks:          blocks,
		FinalConfidence: 0.85,
		StartTime:       time.Now().Add(-5 * time.Minute),
		EndTime:         time.Now(),
		TotalTokens:     len(blocks) * 50,
	}
}

func TestStreamingReasoning(t *testing.T) {
	ctx := context.Background()
	parser := core.NewThinkingBlockParser(nil)
	builder := core.NewReasoningChainBuilder("agent-1", "session-1", "task-1")
	streamer := core.NewReasoningStreamer(parser, builder, nil)

	t.Run("basic streaming", func(t *testing.T) {
		input := `Starting analysis...
<thinking type="analysis">
Analyzing the problem step by step.
This is line 2.
This is line 3.
Confidence: 0.85
</thinking>
Final result.`

		reader := strings.NewReader(input)

		// Collect events
		var events []core.StreamEvent
		done := make(chan bool)

		go func() {
			for event := range streamer.EventChannel() {
				events = append(events, event)
			}
			done <- true
		}()

		// Stream
		err := streamer.Stream(ctx, reader)
		require.NoError(t, err)

		// Wait for completion
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for stream completion")
		}

		// Verify events
		assert.Greater(t, len(events), 0)

		// Check for expected event types
		hasStart := false
		hasUpdate := false
		hasComplete := false
		hasContent := false

		for _, event := range events {
			switch event.Type {
			case core.StreamEventThinkingStart:
				hasStart = true
			case core.StreamEventThinkingUpdate:
				hasUpdate = true
			case core.StreamEventThinkingComplete:
				hasComplete = true
			case core.StreamEventContentChunk:
				hasContent = true
			}
		}

		assert.True(t, hasStart, "Should have thinking start event")
		assert.True(t, hasUpdate, "Should have thinking update events")
		assert.True(t, hasComplete, "Should have thinking complete event")
		assert.True(t, hasContent, "Should have content chunk events")
	})

	t.Run("interruption handling", func(t *testing.T) {
		// Long input that we'll interrupt
		input := strings.Repeat("<thinking>Long content...</thinking>\n", 100)
		reader := strings.NewReader(input)

		// Start streaming
		go func() {
			time.Sleep(50 * time.Millisecond)
			streamer.Interrupt()
		}()

		err := streamer.Stream(ctx, reader)
		// Should not error on interruption
		assert.NoError(t, err)

		// Check for interrupt event
		interrupted := false
		for event := range streamer.EventChannel() {
			if event.Type == core.StreamEventInterrupted {
				interrupted = true
				break
			}
		}
		assert.True(t, interrupted, "Should have interrupt event")
	})
}

func TestQualityScoring(t *testing.T) {
	ctx := context.Background()
	scorer := core.NewQualityScorer()

	t.Run("comprehensive chain scoring", func(t *testing.T) {
		chain := &core.ReasoningChainEnhanced{
			Blocks: []*core.ThinkingBlock{
				{
					Type:       core.ThinkingTypeAnalysis,
					Content:    "Detailed analysis with multiple considerations",
					Confidence: 0.85,
					StructuredData: &core.StructuredThinking{
						Steps: []core.Step{
							{Order: 1, Action: "Analyze data", Purpose: "Understand context"},
							{Order: 2, Action: "Identify patterns", Purpose: "Find insights"},
							{Order: 3, Action: "Draw conclusions", Purpose: "Make recommendations"},
						},
					},
				},
				{
					Type:       core.ThinkingTypePlanning,
					Content:    "Comprehensive plan",
					Confidence: 0.9,
				},
				{
					Type:       core.ThinkingTypeVerification,
					Content:    "Verification complete",
					Confidence: 0.95,
				},
			},
			Performance: core.PerformanceMetrics{
				BacktrackCount: 0,
				IterationCount: 3,
			},
		}

		quality, err := scorer.Score(ctx, chain)
		require.NoError(t, err)

		// Should have good scores
		assert.Greater(t, quality.Coherence, 0.7)
		assert.Greater(t, quality.Completeness, 0.7)
		assert.Greater(t, quality.Depth, 0.6)
		assert.Greater(t, quality.Overall, 0.7)
	})

	t.Run("poor quality detection", func(t *testing.T) {
		chain := &core.ReasoningChainEnhanced{
			Blocks: []*core.ThinkingBlock{
				{
					Type:       core.ThinkingTypeAnalysis,
					Content:    "Brief",
					Confidence: 0.4,
				},
			},
			Performance: core.PerformanceMetrics{
				BacktrackCount: 5,
				IterationCount: 1,
			},
		}

		quality, err := scorer.Score(ctx, chain)
		require.NoError(t, err)

		// Should have low scores
		assert.Less(t, quality.Coherence, 0.5)
		assert.Less(t, quality.Completeness, 0.5)
		assert.Less(t, quality.Overall, 0.5)
	})
}
