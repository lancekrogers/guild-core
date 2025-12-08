// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package core_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// TestReasoningSystemFullIntegration tests the complete reasoning system
func TestReasoningSystemFullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "reasoning_integration_test")

	// Setup test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reasoning.db")

	// Create reasoning system
	systemConfig := core.ReasoningSystemConfig{
		ExtractorConfig: core.ReasoningConfig{
			EnableCaching:      true,
			CacheMaxSize:       100,
			CacheTTL:           1 * time.Minute,
			MaxReasoningLength: 10000,
			MinConfidence:      0.0,
			MaxConfidence:      1.0,
			StrictValidation:   true,
		},
		StorageConfig: core.ReasoningStorageConfig{
			RetentionDays:     30,
			MaxChainsPerQuery: 1000,
			EnableCompression: true,
			StorageBackend:    "sqlite",
		},
		EnableAnalytics: true,
		DatabasePath:    dbPath,
	}

	system, err := core.NewReasoningSystem(ctx, systemConfig)
	require.NoError(t, err)
	defer system.Close(ctx)

	// Start maintenance tasks
	system.StartMaintenance(ctx)

	t.Run("ExtractAndStoreReasoning", func(t *testing.T) {
		// Test response with multiple thinking blocks
		response := `<thinking type="analysis">
Let me analyze this problem step by step.
The key components are:
1. Data ingestion
2. Processing pipeline
3. Output generation

Confidence: 0.85
</thinking>

I'll help you build a data processing pipeline.

<thinking type="planning">
Here's my implementation plan:
- Set up data ingestion from multiple sources
- Create transformation pipeline with error handling
- Implement output formatters for different targets
- Add monitoring and logging

This should take about 3-4 hours to implement properly.
Confidence: 0.9
</thinking>

<thinking type="decision_making">
I need to decide on the technology stack:
- Language: Go (for performance and concurrency)
- Message Queue: NATS (lightweight and fast)
- Storage: PostgreSQL with TimescaleDB extension
- Monitoring: Prometheus + Grafana

Decision: Go with cloud-native stack
Confidence: 0.95
</thinking>

Here's the implementation plan...`

		// Extract reasoning
		chain, err := system.Extractor.Extract(ctx, response)
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.Len(t, chain.Blocks, 3)

		// Verify block types
		assert.Equal(t, core.ThinkingTypeAnalysis, chain.Blocks[0].Type)
		assert.Equal(t, core.ThinkingTypePlanning, chain.Blocks[1].Type)
		assert.Equal(t, core.ThinkingTypeDecisionMaking, chain.Blocks[2].Type)

		// Verify confidence extraction
		assert.InDelta(t, 0.85, chain.Blocks[0].Confidence, 0.01)
		assert.InDelta(t, 0.9, chain.Blocks[1].Confidence, 0.01)
		assert.InDelta(t, 0.95, chain.Blocks[2].Confidence, 0.01)

		// Store chain with metadata
		enhancedChain := &core.ReasoningChainEnhanced{
			ID:              chain.ID,
			AgentID:         "test-agent-001",
			SessionID:       "session-001",
			TaskID:          "task-001",
			Blocks:          chain.Blocks,
			Summary:         "Data pipeline implementation planning",
			FinalConfidence: chain.Confidence,
			StartTime:       time.Now().Add(-5 * time.Minute),
			EndTime:         time.Now(),
			TotalTokens:     523,
			Context: map[string]interface{}{
				"project": "data-pipeline",
				"phase":   "planning",
			},
			Tags: []string{"planning", "architecture", "go"},
		}

		err = system.Storage.Store(ctx, enhancedChain)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := system.Storage.Get(ctx, chain.ID)
		require.NoError(t, err)
		assert.Equal(t, enhancedChain.ID, retrieved.ID)
		assert.Equal(t, enhancedChain.AgentID, retrieved.AgentID)
		assert.Len(t, retrieved.Blocks, 3)
	})

	t.Run("StreamingReasoningExtraction", func(t *testing.T) {
		// Create streaming components
		parser := core.NewThinkingBlockParser()
		builder := core.NewReasoningChainBuilder("test-agent-002", "session-002", "task-002")
		streamer := core.NewReasoningStreamer(parser, builder)

		// Simulate streaming response
		streamContent := `Starting analysis...

<thinking type="hypothesis">
I hypothesize that the performance issue is related to:
1. Inefficient database queries
2. Lack of caching
3. Synchronous processing of large datasets

Let me verify each hypothesis.
Confidence: 0.7
</thinking>

Let me check the database queries first...

<thinking type="verification">
After analyzing the query logs:
- Found N+1 query pattern in user fetching
- Missing indexes on frequently queried columns
- No query result caching

These findings confirm hypothesis #1.
Confidence: 0.92
</thinking>

The main issue is with database queries.`

		reader := strings.NewReader(streamContent)

		// Collect events
		var events []core.StreamEvent
		done := make(chan bool)

		go func() {
			for event := range streamer.EventChannel() {
				events = append(events, event)
			}
			done <- true
		}()

		// Stream the content
		err := streamer.Stream(ctx, reader)
		require.NoError(t, err)

		// Wait for events
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for stream events")
		}

		// Verify events
		assert.Greater(t, len(events), 0)

		// Count event types
		eventCounts := make(map[core.StreamEventType]int)
		for _, event := range events {
			eventCounts[event.Type]++
		}

		assert.Equal(t, 2, eventCounts[core.StreamEventThinkingStart])
		assert.Equal(t, 2, eventCounts[core.StreamEventThinkingComplete])
		assert.Greater(t, eventCounts[core.StreamEventThinkingUpdate], 0)

		// Get final chain
		chain, err := streamer.GetChain(ctx)
		require.NoError(t, err)
		assert.Len(t, chain.Blocks, 2)
		assert.Equal(t, core.ThinkingTypeHypothesis, chain.Blocks[0].Type)
		assert.Equal(t, core.ThinkingTypeVerification, chain.Blocks[1].Type)
	})

	t.Run("TokenManagementAndCompaction", func(t *testing.T) {
		// Create token manager
		tokenConfig := core.TokenConfig{
			DefaultLimit:    1000,
			SafetyMargin:    0.15,
			EmergencyMargin: 0.02,
			CompactionConfig: core.CompactionConfig{
				Strategies:      []string{"priority", "temporal", "summarization"},
				TargetReduction: 0.5,
			},
		}

		manager := core.NewTokenManager(tokenConfig)
		windowID := "test-window"

		// Create window
		err := manager.CreateWindow(ctx, windowID, 1000)
		require.NoError(t, err)

		// Add messages until approaching limit
		messages := []core.ContextMessage{
			{
				ID:        "msg-1",
				Role:      "user",
				Content:   "Please analyze the system architecture and identify potential bottlenecks.",
				Priority:  8,
				Timestamp: time.Now().Add(-30 * time.Minute),
			},
			{
				ID:        "msg-2",
				Role:      "assistant",
				Content:   generateLongResponse(200), // ~200 tokens
				Priority:  6,
				Timestamp: time.Now().Add(-25 * time.Minute),
			},
			{
				ID:        "msg-3",
				Role:      "user",
				Content:   "Can you elaborate on the database bottlenecks?",
				Priority:  9,
				Timestamp: time.Now().Add(-20 * time.Minute),
			},
			{
				ID:        "msg-4",
				Role:      "assistant",
				Content:   generateLongResponse(300), // ~300 tokens
				Priority:  7,
				Timestamp: time.Now().Add(-15 * time.Minute),
			},
			{
				ID:        "msg-5",
				Role:      "user",
				Content:   "What about caching strategies?",
				Priority:  9,
				Timestamp: time.Now().Add(-10 * time.Minute),
			},
			{
				ID:        "msg-6",
				Role:      "assistant",
				Content:   generateLongResponse(250), // ~250 tokens
				Priority:  8,
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
		}

		// Add messages
		for _, msg := range messages {
			err := manager.AddMessage(ctx, windowID, msg)
			require.NoError(t, err)
		}

		// Check safety
		safety, err := manager.CheckSafety(windowID)
		require.NoError(t, err)
		assert.Greater(t, safety.PercentUsed, 0.7)

		// Trigger compaction
		err = manager.CompactWindow(ctx, windowID)
		require.NoError(t, err)

		// Verify compaction
		window, err := manager.GetWindow(windowID)
		require.NoError(t, err)
		assert.Less(t, len(window.Messages), len(messages))

		// Verify high priority messages preserved
		hasHighPriority := false
		for _, msg := range window.Messages {
			if msg.Priority >= 9 {
				hasHighPriority = true
				break
			}
		}
		assert.True(t, hasHighPriority)
	})

	t.Run("PatternLearningAndApplication", func(t *testing.T) {
		// Create multiple reasoning chains with patterns
		chains := createTestReasoningChains(t, system.Storage)

		// Learn patterns
		learner := core.NewPatternLearner(nil, nil) // Using in-memory repository
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

		// Find analysis->planning->implementation pattern
		var analysisPattern *core.LearnedPattern
		for _, pattern := range patterns {
			if len(pattern.Signature) >= 3 &&
				pattern.Signature[0] == core.ThinkingTypeAnalysis &&
				pattern.Signature[1] == core.ThinkingTypePlanning {
				analysisPattern = pattern
				break
			}
		}
		require.NotNil(t, analysisPattern, "Should find analysis->planning pattern")

		// Apply pattern
		currentContext := "User asked to implement a new feature"
		application, err := learner.ApplyPattern(ctx, analysisPattern, currentContext)
		require.NoError(t, err)
		assert.NotNil(t, application)
		assert.Greater(t, application.Confidence, 0.6)

		// Simulate feedback
		feedback := &core.PatternFeedback{
			PatternID: analysisPattern.ID,
			Success:   true,
			Outcome:   "Pattern successfully guided implementation",
			Rating:    0.9,
		}

		err = learner.RefinePattern(ctx, analysisPattern, feedback)
		require.NoError(t, err)
	})

	t.Run("ReasoningAnalytics", func(t *testing.T) {
		// Get analytics for test agent
		stats, err := system.Storage.GetStats(ctx, "test-agent-001",
			time.Now().Add(-1*time.Hour), time.Now())
		require.NoError(t, err)

		assert.Greater(t, stats.TotalChains, int64(0))
		assert.Greater(t, stats.AvgConfidence, 0.0)
		assert.LessOrEqual(t, stats.AvgConfidence, 1.0)

		// Generate insights
		if system.Analyzer != nil {
			insights, err := system.GetInsights(ctx, "test-agent-001")
			require.NoError(t, err)
			assert.NotEmpty(t, insights)
		}
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Test concurrent extraction and storage
		responses := []string{
			`<thinking>Concurrent test 1</thinking>Result 1`,
			`<thinking>Concurrent test 2</thinking>Result 2`,
			`<thinking>Concurrent test 3</thinking>Result 3`,
			`<thinking>Concurrent test 4</thinking>Result 4`,
			`<thinking>Concurrent test 5</thinking>Result 5`,
		}

		type result struct {
			chain *core.ReasoningChain
			err   error
		}

		results := make(chan result, len(responses))

		// Extract concurrently
		for i, response := range responses {
			go func(idx int, resp string) {
				chain, err := system.Extractor.Extract(ctx, resp)
				results <- result{chain: chain, err: err}
			}(i, response)
		}

		// Collect results
		var chains []*core.ReasoningChain
		for i := 0; i < len(responses); i++ {
			res := <-results
			require.NoError(t, res.err)
			chains = append(chains, res.chain)
		}

		assert.Len(t, chains, len(responses))

		// Store concurrently
		storeResults := make(chan error, len(chains))
		for i, chain := range chains {
			go func(idx int, c *core.ReasoningChain) {
				enhanced := &core.ReasoningChainEnhanced{
					ID:        c.ID,
					AgentID:   fmt.Sprintf("concurrent-agent-%d", idx),
					SessionID: "concurrent-session",
					TaskID:    fmt.Sprintf("concurrent-task-%d", idx),
					Blocks:    c.Blocks,
					StartTime: time.Now(),
					EndTime:   time.Now(),
				}
				storeResults <- system.Storage.Store(ctx, enhanced)
			}(i, chain)
		}

		// Verify storage
		for i := 0; i < len(chains); i++ {
			err := <-storeResults
			require.NoError(t, err)
		}
	})

	t.Run("ErrorRecovery", func(t *testing.T) {
		// Test malformed thinking blocks
		malformedResponse := `<thinking>Unclosed thinking block
		
		This will cause issues`

		chain, err := system.Extractor.Extract(ctx, malformedResponse)
		// Should handle gracefully
		assert.NoError(t, err)
		assert.NotNil(t, chain)

		// Test token limit exceeded
		veryLongResponse := "<thinking>" + generateLongResponse(10000) + "</thinking>"
		chain, err = system.Extractor.Extract(ctx, veryLongResponse)
		// Should truncate or handle appropriately
		if err != nil {
			assert.True(t, gerror.Is(err, gerror.ErrCodeResourceLimit))
		}
	})
}

// TestReasoningWithRealProviders tests with actual LLM providers if configured
func TestReasoningWithRealProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping provider integration test in short mode")
	}

	// Check for provider credentials
	providers := []struct {
		name     string
		envVar   string
		provider string
		model    string
	}{
		{"OpenAI", "OPENAI_API_KEY", "openai", "gpt-4"},
		{"Anthropic", "ANTHROPIC_API_KEY", "anthropic", "claude-3-opus-20240229"},
		{"Ollama", "OLLAMA_HOST", "ollama", "llama2"},
	}

	ctx := context.Background()

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			// Skip if not configured
			if os.Getenv(p.envVar) == "" {
				t.Skipf("Skipping %s test: %s not set", p.name, p.envVar)
			}

			// Create provider client
			providerConfig := &config.ProviderConfig{
				Type:  p.provider,
				Model: p.model,
			}

			client, err := providers.NewClient(ctx, providerConfig)
			require.NoError(t, err)

			// Create system
			system, err := core.NewReasoningSystem(ctx, core.DefaultReasoningSystemConfig())
			require.NoError(t, err)
			defer system.Close(ctx)

			// Test prompt that triggers reasoning
			prompt := `Please analyze the following code and provide improvement suggestions:

func processItems(items []string) []string {
	var result []string
	for i := 0; i < len(items); i++ {
		item := items[i]
		if item != "" {
			result = append(result, strings.ToUpper(item))
		}
	}
	return result
}

Think through your analysis step by step.`

			// Get response from provider
			response, err := client.Complete(ctx, prompt)
			require.NoError(t, err)

			// Extract reasoning
			chain, err := system.Extractor.Extract(ctx, response)
			require.NoError(t, err)

			// Verify reasoning was extracted
			assert.NotNil(t, chain)
			assert.NotEmpty(t, chain.Blocks)

			// Log results
			t.Logf("%s extracted %d thinking blocks", p.name, len(chain.Blocks))
			for i, block := range chain.Blocks {
				t.Logf("  Block %d: Type=%s, Confidence=%.2f, Tokens=%d",
					i+1, block.Type, block.Confidence, block.TokenCount)
			}

			// Store for analytics
			enhanced := &core.ReasoningChainEnhanced{
				ID:        chain.ID,
				AgentID:   fmt.Sprintf("test-%s", p.provider),
				SessionID: "provider-test",
				TaskID:    "code-analysis",
				Blocks:    chain.Blocks,
				StartTime: time.Now(),
				EndTime:   time.Now(),
				Context: map[string]interface{}{
					"provider": p.provider,
					"model":    p.model,
				},
			}

			err = system.Storage.Store(ctx, enhanced)
			require.NoError(t, err)
		})
	}
}

// Helper functions

func generateLongResponse(approxTokens int) string {
	// Approximate 1 token = 4 characters
	chars := approxTokens * 4
	words := []string{
		"analyze", "implement", "optimize", "configure", "database",
		"performance", "architecture", "scalability", "reliability",
		"monitoring", "deployment", "integration", "testing", "security",
	}

	var result strings.Builder
	for result.Len() < chars {
		word := words[result.Len()%len(words)]
		result.WriteString(word)
		result.WriteString(" ")
	}

	return result.String()
}

func createTestReasoningChains(t *testing.T, storage core.ReasoningStorage) []*core.ReasoningChainEnhanced {
	ctx := context.Background()
	chains := make([]*core.ReasoningChainEnhanced, 0)

	// Pattern 1: Analysis -> Planning -> Implementation
	patterns := [][]core.ThinkingType{
		{core.ThinkingTypeAnalysis, core.ThinkingTypePlanning, core.ThinkingTypeOptimization},
		{core.ThinkingTypeAnalysis, core.ThinkingTypePlanning, core.ThinkingTypeVerification},
		{core.ThinkingTypeHypothesis, core.ThinkingTypeVerification, core.ThinkingTypeDecisionMaking},
		{core.ThinkingTypeAnalysis, core.ThinkingTypeDecisionMaking, core.ThinkingTypeOptimization},
		{core.ThinkingTypeErrorRecovery, core.ThinkingTypeAnalysis, core.ThinkingTypePlanning},
	}

	for i, pattern := range patterns {
		blocks := make([]*core.ThinkingBlock, len(pattern))
		for j, thinkingType := range pattern {
			blocks[j] = &core.ThinkingBlock{
				ID:         fmt.Sprintf("block-%d-%d", i, j),
				Type:       thinkingType,
				Content:    fmt.Sprintf("Test content for %s", thinkingType),
				Confidence: 0.7 + float64(j)*0.1,
				Timestamp:  time.Now(),
				TokenCount: 50 + j*10,
			}
		}

		chain := &core.ReasoningChainEnhanced{
			ID:              fmt.Sprintf("chain-%d", i),
			AgentID:         "pattern-test-agent",
			SessionID:       fmt.Sprintf("session-%d", i),
			TaskID:          fmt.Sprintf("task-%d", i),
			Blocks:          blocks,
			FinalConfidence: 0.8 + float64(i)*0.02,
			StartTime:       time.Now().Add(-time.Duration(i) * time.Hour),
			EndTime:         time.Now().Add(-time.Duration(i) * time.Hour).Add(5 * time.Minute),
			TotalTokens:     200 + i*50,
		}

		err := storage.Store(ctx, chain)
		require.NoError(t, err)
		chains = append(chains, chain)
	}

	return chains
}

// MockLLMClient for testing without real providers
type MockLLMClient struct {
	responses []string
	index     int
}

func NewMockLLMClient(responses []string) *MockLLMClient {
	return &MockLLMClient{
		responses: responses,
		index:     0,
	}
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	if m.index >= len(m.responses) {
		return "", io.EOF
	}

	response := m.responses[m.index]
	m.index++
	return response, nil
}

func (m *MockLLMClient) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)

		response, err := m.Complete(ctx, prompt)
		if err != nil {
			return
		}

		// Simulate streaming by sending chunks
		words := strings.Fields(response)
		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case ch <- word:
				if i < len(words)-1 {
					ch <- " "
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return ch, nil
}
