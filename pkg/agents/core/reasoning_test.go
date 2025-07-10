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
	parser := core.NewThinkingBlockParser()
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
	parser := core.NewThinkingBlockParser()
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

		chain, err := extractor.Extract(ctx, response)
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.Len(t, chain.Blocks, 2)
		assert.NotEmpty(t, chain.ID)
		assert.Greater(t, chain.Confidence, 0.0)
	})

	t.Run("extraction with metadata", func(t *testing.T) {
		response := `<thinking>Test reasoning</thinking>`
		metadata := map[string]interface{}{
			"task_id":    "test-123",
			"agent_name": "test-agent",
		}

		chain, err := extractor.ExtractWithMetadata(ctx, response, metadata)
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.Equal(t, metadata["task_id"], chain.Metadata["task_id"])
		assert.Equal(t, metadata["agent_name"], chain.Metadata["agent_name"])
	})

	t.Run("caching behavior", func(t *testing.T) {
		response := `<thinking>Cached test</thinking>`

		// First call
		chain1, err := extractor.Extract(ctx, response)
		require.NoError(t, err)

		// Second call (should hit cache)
		chain2, err := extractor.Extract(ctx, response)
		require.NoError(t, err)

		// Should return same chain ID from cache
		assert.Equal(t, chain1.ID, chain2.ID)

		// Check stats
		stats := extractor.GetStats()
		assert.Greater(t, stats["cache_hit_rate"].(float64), 0.0)
	})

	t.Run("validation", func(t *testing.T) {
		// Test invalid confidence
		invalidChain := &core.ReasoningChain{
			Blocks: []*core.ThinkingBlock{
				{
					ID:         "test",
					Type:       core.ThinkingTypeAnalysis,
					Content:    "Test",
					Confidence: 1.5, // Invalid
				},
			},
		}

		err := extractor.ValidateChain(invalidChain)
		assert.Error(t, err)
		assert.True(t, gerror.Is(err, gerror.ErrCodeValidation))
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
						Decision:     "Use microservices",
						Confidence:   0.9,
						Alternatives: []string{"Monolith", "Serverless"},
						Reasoning:    "Better scalability",
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
		DefaultLimit:    1000,
		SafetyMargin:    0.15,
		EmergencyMargin: 0.02,
		CompactionConfig: core.CompactionConfig{
			Strategies:      []string{"priority"},
			TargetReduction: 0.5,
		},
	}

	manager := core.NewTokenManager(config)

	t.Run("window creation and management", func(t *testing.T) {
		windowID := "test-window"
		
		// Create window
		err := manager.CreateWindow(ctx, windowID, 1000)
		require.NoError(t, err)

		// Get window
		window, err := manager.GetWindow(windowID)
		require.NoError(t, err)
		assert.Equal(t, windowID, window.ID)
		assert.Equal(t, 1000, window.MaxTokens)
		assert.Equal(t, 0, window.CurrentUsage)

		// Add messages
		messages := []core.ContextMessage{
			{
				ID:        "msg-1",
				Role:      "user",
				Content:   "Hello, can you help me?",
				Priority:  5,
				Timestamp: time.Now(),
			},
			{
				ID:        "msg-2",
				Role:      "assistant",
				Content:   "Of course! I'd be happy to help.",
				Priority:  5,
				Timestamp: time.Now(),
			},
		}

		for _, msg := range messages {
			err := manager.AddMessage(ctx, windowID, msg)
			require.NoError(t, err)
		}

		// Check safety
		safety, err := manager.CheckSafety(windowID)
		require.NoError(t, err)
		assert.Greater(t, safety.TokensUsed, 0)
		assert.Less(t, safety.PercentUsed, 0.5)
		assert.True(t, safety.IsSafe)
	})

	t.Run("token limit enforcement", func(t *testing.T) {
		windowID := "small-window"
		
		// Create small window
		err := manager.CreateWindow(ctx, windowID, 100)
		require.NoError(t, err)

		// Try to add message that exceeds limit
		largeMessage := core.ContextMessage{
			ID:      "large-msg",
			Role:    "user",
			Content: strings.Repeat("word ", 100), // ~100 tokens
			Priority: 5,
			Timestamp: time.Now(),
		}

		err = manager.AddMessage(ctx, windowID, largeMessage)
		// Should trigger compaction or error
		if err != nil {
			assert.True(t, gerror.Is(err, gerror.ErrCodeResourceLimit))
		}
	})

	t.Run("token prediction", func(t *testing.T) {
		windowID := "prediction-window"
		
		err := manager.CreateWindow(ctx, windowID, 1000)
		require.NoError(t, err)

		// Add some messages
		for i := 0; i < 5; i++ {
			msg := core.ContextMessage{
				ID:        fmt.Sprintf("msg-%d", i),
				Role:      "user",
				Content:   "Test message",
				Priority:  5,
				Timestamp: time.Now(),
			}
			manager.AddMessage(ctx, windowID, msg)
		}

		// Predict future usage
		prediction, err := manager.PredictUsage(windowID, 200)
		require.NoError(t, err)
		assert.Greater(t, prediction.PredictedTotal, prediction.CurrentUsage)
		assert.NotNil(t, prediction.CompactionNeeded)
	})
}

func TestPatternLearning(t *testing.T) {
	ctx := context.Background()
	learner := core.NewPatternLearner(nil, nil) // In-memory repository

	t.Run("pattern discovery", func(t *testing.T) {
		// Create chains with repeated patterns
		chains := []*core.ReasoningChainEnhanced{
			createChainWithPattern("chain-1", []core.ThinkingType{
				core.ThinkingTypeAnalysis,
				core.ThinkingTypePlanning,
				core.ThinkingTypeVerification,
			}),
			createChainWithPattern("chain-2", []core.ThinkingType{
				core.ThinkingTypeAnalysis,
				core.ThinkingTypePlanning,
				core.ThinkingTypeVerification,
			}),
			createChainWithPattern("chain-3", []core.ThinkingType{
				core.ThinkingTypeAnalysis,
				core.ThinkingTypePlanning,
				core.ThinkingTypeOptimization,
			}),
			createChainWithPattern("chain-4", []core.ThinkingType{
				core.ThinkingTypeHypothesis,
				core.ThinkingTypeVerification,
				core.ThinkingTypeDecisionMaking,
			}),
		}

		config := core.PatternLearningConfig{
			MinOccurrences:       2,
			MinConfidence:        0.7,
			MaxPatternsPerBatch:  10,
			LearningRate:         0.1,
			DecayFactor:          0.95,
			CrossDomainThreshold: 0.6,
		}

		patterns, err := learner.LearnPatterns(ctx, chains, config)
		require.NoError(t, err)
		assert.NotEmpty(t, patterns)

		// Should find the analysis->planning pattern
		found := false
		for _, pattern := range patterns {
			if len(pattern.Signature) >= 2 &&
				pattern.Signature[0] == core.ThinkingTypeAnalysis &&
				pattern.Signature[1] == core.ThinkingTypePlanning {
				found = true
				assert.GreaterOrEqual(t, pattern.Performance.SuccessRate, 0.5)
				assert.Greater(t, pattern.Performance.OccurrenceCount, 1)
			}
		}
		assert.True(t, found, "Should find analysis->planning pattern")
	})

	t.Run("pattern application", func(t *testing.T) {
		// Create a known pattern
		pattern := &core.LearnedPattern{
			ID:          "test-pattern",
			Name:        "Analysis-Planning-Verification",
			Description: "Standard analysis workflow",
			Signature: []core.ThinkingType{
				core.ThinkingTypeAnalysis,
				core.ThinkingTypePlanning,
				core.ThinkingTypeVerification,
			},
			Performance: core.PatternPerformance{
				SuccessRate:     0.85,
				OccurrenceCount: 10,
				LastUsed:        time.Now(),
			},
		}

		// Apply pattern
		context := "User asked to analyze system performance"
		application, err := learner.ApplyPattern(ctx, pattern, context)
		require.NoError(t, err)
		assert.NotNil(t, application)
		assert.Equal(t, pattern.ID, application.PatternID)
		assert.Greater(t, application.Confidence, 0.5)
		assert.NotEmpty(t, application.SuggestedSteps)
	})

	t.Run("pattern refinement", func(t *testing.T) {
		pattern := &core.LearnedPattern{
			ID:   "refine-pattern",
			Name: "Test Pattern",
			Performance: core.PatternPerformance{
				SuccessRate:     0.7,
				OccurrenceCount: 5,
			},
		}

		// Positive feedback
		feedback := &core.PatternFeedback{
			PatternID: pattern.ID,
			Success:   true,
			Outcome:   "Pattern worked well",
			Rating:    0.9,
		}

		err := learner.RefinePattern(ctx, pattern, feedback)
		require.NoError(t, err)
		// Pattern should be strengthened
		assert.Greater(t, pattern.Performance.SuccessRate, 0.7)
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
		chain := &core.ReasoningChainEnhanced{
			ID:        "test-chain-1",
			AgentID:   "agent-1",
			SessionID: "session-1",
			TaskID:    "task-1",
			Blocks: []*core.ThinkingBlock{
				{
					ID:         "block-1",
					Type:       core.ThinkingTypeAnalysis,
					Content:    "Test analysis",
					Confidence: 0.85,
				},
			},
			FinalConfidence: 0.85,
			StartTime:       time.Now().Add(-5 * time.Minute),
			EndTime:         time.Now(),
			TotalTokens:     100,
		}

		// Store
		err := storage.Store(ctx, chain)
		require.NoError(t, err)

		// Retrieve
		retrieved, err := storage.Get(ctx, chain.ID)
		require.NoError(t, err)
		assert.Equal(t, chain.ID, retrieved.ID)
		assert.Equal(t, chain.AgentID, retrieved.AgentID)
		assert.Len(t, retrieved.Blocks, 1)
	})

	t.Run("query chains", func(t *testing.T) {
		// Store multiple chains
		for i := 0; i < 5; i++ {
			chain := &core.ReasoningChainEnhanced{
				ID:              fmt.Sprintf("query-chain-%d", i),
				AgentID:         "agent-1",
				SessionID:       "session-1",
				TaskID:          fmt.Sprintf("task-%d", i),
				Blocks:          []*core.ThinkingBlock{},
				FinalConfidence: 0.7 + float64(i)*0.05,
				StartTime:       time.Now().Add(-time.Duration(i) * time.Hour),
				EndTime:         time.Now().Add(-time.Duration(i) * time.Hour).Add(5 * time.Minute),
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
			assert.GreaterOrEqual(t, chains[i-1].FinalConfidence, chains[i].FinalConfidence)
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
	parser := core.NewThinkingBlockParser()
	builder := core.NewReasoningChainBuilder("agent-1", "session-1", "task-1")
	streamer := core.NewReasoningStreamer(parser, builder)

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

func TestCompactionStrategies(t *testing.T) {
	ctx := context.Background()

	t.Run("priority compaction", func(t *testing.T) {
		strategy := core.NewPriorityStrategy()
		
		window := &core.ContextWindow{
			ID:        "test",
			MaxTokens: 1000,
			Messages: []core.ContextMessage{
				{ID: "1", Content: "Low priority", Priority: 3},
				{ID: "2", Content: "High priority", Priority: 9},
				{ID: "3", Content: "Medium priority", Priority: 5},
				{ID: "4", Content: "Critical", Priority: 10},
			},
		}

		compacted, err := strategy.Compact(ctx, window)
		require.NoError(t, err)
		
		// Should keep high priority messages
		foundCritical := false
		foundHigh := false
		for _, msg := range compacted.Messages {
			if msg.Priority == 10 {
				foundCritical = true
			}
			if msg.Priority == 9 {
				foundHigh = true
			}
		}
		assert.True(t, foundCritical)
		assert.True(t, foundHigh)
	})

	t.Run("temporal compaction", func(t *testing.T) {
		strategy := core.NewTemporalStrategy()
		
		now := time.Now()
		window := &core.ContextWindow{
			ID:        "test",
			MaxTokens: 1000,
			Messages: []core.ContextMessage{
				{ID: "1", Content: "Very old", Timestamp: now.Add(-2 * time.Hour)},
				{ID: "2", Content: "Old", Timestamp: now.Add(-1 * time.Hour)},
				{ID: "3", Content: "Recent", Timestamp: now.Add(-5 * time.Minute)},
				{ID: "4", Content: "Very recent", Timestamp: now.Add(-1 * time.Minute)},
			},
		}

		compacted, err := strategy.Compact(ctx, window)
		require.NoError(t, err)
		
		// Should keep recent messages
		assert.Less(t, len(compacted.Messages), len(window.Messages))
		
		// Verify recent messages are kept
		for _, msg := range compacted.Messages {
			age := now.Sub(msg.Timestamp)
			assert.Less(t, age, 30*time.Minute)
		}
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
						Steps: []string{"Step 1", "Step 2", "Step 3"},
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