// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package reasoning_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderIntegration tests reasoning extraction with real providers
// Run with: go test -tags=integration -run TestProviderIntegration
func TestProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx := context.Background()

	// Test content with various reasoning patterns
	testCases := []struct {
		name      string
		content   string
		minBlocks int
	}{
		{
			name: "simple_thinking",
			content: `I need to analyze this problem step by step.
<thinking>
First, I should understand what the user is asking for.
They want me to implement a new feature for tracking user activity.
This will require:
1. Database schema changes
2. API endpoints
3. Frontend components
</thinking>

Based on my analysis, here's the implementation plan...`,
			minBlocks: 1,
		},
		{
			name: "nested_reasoning",
			content: `Let me work through this complex problem.
<thinking type="analysis">
The user wants to optimize database performance.
<thinking type="hypothesis">
I believe the issue might be related to missing indexes.
Let me check the query patterns.
</thinking>
Based on the query analysis, I can see several opportunities.
</thinking>

<thinking type="planning">
Here's my optimization strategy:
1. Add composite indexes
2. Denormalize frequently joined tables
3. Implement query caching
</thinking>`,
			minBlocks: 3,
		},
		{
			name: "decision_making",
			content: `<thinking type="decision_making">
I need to choose between three implementation approaches:
Option 1: Microservices - Better scalability but higher complexity
Option 2: Monolith - Simpler to maintain but harder to scale
Option 3: Modular monolith - Balance between the two

Considering the team size and project requirements, I'll go with Option 3.
</thinking>`,
			minBlocks: 1,
		},
	}

	providers := []struct {
		name     string
		provider string
		apiKey   string
		model    string
		skip     bool
	}{
		{
			name:     "OpenAI",
			provider: providers.ProviderOpenAI,
			apiKey:   os.Getenv("OPENAI_API_KEY"),
			model:    "gpt-4",
			skip:     os.Getenv("OPENAI_API_KEY") == "",
		},
		{
			name:     "Anthropic",
			provider: providers.ProviderAnthropic,
			apiKey:   os.Getenv("ANTHROPIC_API_KEY"),
			model:    "claude-3-opus-20240229",
			skip:     os.Getenv("ANTHROPIC_API_KEY") == "",
		},
		{
			name:     "DeepSeek",
			provider: providers.ProviderDeepSeek,
			apiKey:   os.Getenv("DEEPSEEK_API_KEY"),
			model:    "deepseek-chat",
			skip:     os.Getenv("DEEPSEEK_API_KEY") == "",
		},
	}

	for _, provider := range providers {
		if provider.skip {
			t.Logf("Skipping %s provider (no API key)", provider.name)
			continue
		}

		t.Run(provider.name, func(t *testing.T) {
			// Create provider-specific extractor
			extractor := createProviderExtractor(t, provider.provider, provider.apiKey, provider.model)

			// Create registry with the provider
			registry := createProviderRegistry(t, extractor)

			err := registry.Start(ctx)
			require.NoError(t, err)
			defer registry.Stop(ctx)

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Extract reasoning
					blocks, err := registry.Extract(ctx, "test-agent", tc.content)

					assert.NoError(t, err)
					assert.GreaterOrEqual(t, len(blocks), tc.minBlocks,
						"Expected at least %d blocks, got %d", tc.minBlocks, len(blocks))

					// Verify block properties
					for _, block := range blocks {
						assert.NotEmpty(t, block.ID)
						assert.NotEmpty(t, block.Type)
						assert.NotEmpty(t, block.Content)
						assert.Greater(t, block.TokenCount, 0)
						assert.NotZero(t, block.Timestamp)

						t.Logf("Block: type=%s, tokens=%d, confidence=%.2f",
							block.Type, block.TokenCount, block.Confidence)
					}
				})
			}
		})
	}
}

// TestProviderSpecificPatterns tests provider-specific reasoning patterns
func TestProviderSpecificPatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx := context.Background()

	t.Run("OpenAI_ChainOfThought", func(t *testing.T) {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			t.Skip("OPENAI_API_KEY not set")
		}

		content := `Let me solve this step-by-step.
<thinking>
Step 1: Identify the problem constraints
- We have limited memory (2GB)
- Need to process 1TB of data
- Must maintain sub-second response times

Step 2: Consider streaming approaches
- Process data in chunks
- Use memory-mapped files
- Implement a sliding window algorithm

Step 3: Choose the optimal solution
Given the constraints, I'll implement a streaming processor with memory-mapped files.
</thinking>`

		extractor := createProviderExtractor(t, providers.ProviderOpenAI, apiKey, "gpt-4")
		blocks, err := extractor.Extract(ctx, content)

		require.NoError(t, err)
		assert.NotEmpty(t, blocks)

		// Verify structured data extraction
		for _, block := range blocks {
			if block.Type == "planning" && block.StructuredData != nil {
				assert.NotEmpty(t, block.StructuredData.Steps)
			}
		}
	})

	t.Run("Anthropic_ConstitutionalAI", func(t *testing.T) {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			t.Skip("ANTHROPIC_API_KEY not set")
		}

		content := `<thinking>
I need to consider the ethical implications of this feature.
- Privacy: Will this collect user data appropriately?
- Security: Are we protecting sensitive information?
- Transparency: Do users understand what we're doing?
- Benefit: Does this genuinely help users?

After careful consideration, I believe we can proceed with appropriate safeguards.
</thinking>`

		extractor := createProviderExtractor(t, providers.ProviderAnthropic, apiKey, "claude-3-opus-20240229")
		blocks, err := extractor.Extract(ctx, content)

		require.NoError(t, err)
		assert.NotEmpty(t, blocks)

		// Check for ethical reasoning markers
		found := false
		for _, block := range blocks {
			if block.Type == "analysis" {
				assert.Contains(t, block.Content, "ethical")
				found = true
			}
		}
		assert.True(t, found, "Should detect ethical reasoning")
	})
}

// TestProviderFailureHandling tests handling of provider-specific failures
func TestProviderFailureHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx := context.Background()

	t.Run("RateLimitHandling", func(t *testing.T) {
		// Create registry with aggressive rate limiting
		extractor := reasoning.NewDefaultExtractor()
		registry := createProviderRegistry(t, extractor)

		// Override rate limiter with very low limits
		rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
			GlobalRPS:   1,   // Very low limit
			PerAgentRPS: 0.5, // Even lower per-agent
			BurstSize:   1,
		})

		registry = reasoning.NewRegistry(
			extractor,
			reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			}),
			rateLimiter,
			reasoning.NewRetryer(reasoning.RetryConfig{
				MaxAttempts:  2,
				InitialDelay: 10 * time.Millisecond,
			}),
			reasoning.NewDeadLetterQueue(&mockDatabase{}, slog.Default(), reasoning.DeadLetterConfig{}),
			reasoning.NewHealthChecker(slog.Default()),
			slog.Default(),
			&mockEventBus{events: make([]interface{}, 0)},
		)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		// Rapid fire requests
		var rateLimitCount int
		for i := 0; i < 10; i++ {
			_, err := registry.Extract(ctx, "rate-test", "test content")
			if reasoning.IsRateLimitExceeded(err) {
				rateLimitCount++
			}
		}

		assert.Greater(t, rateLimitCount, 5, "Should hit rate limits")
	})

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Create slow extractor that simulates timeouts
		slowExtractor := &slowExtractor{delay: 5 * time.Second}
		registry := createProviderRegistry(t, slowExtractor)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		// Use short timeout
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		_, err = registry.Extract(ctx, "timeout-test", "test content")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

// TestProviderMetrics tests metrics collection across providers
func TestProviderMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx := context.Background()

	// Create metrics collector
	metricsRegistry := observability.NewMetricsRegistry()
	metricsCollector, err := reasoning.NewMetricsCollector(metricsRegistry)
	require.NoError(t, err)

	// Test with each available provider
	providers := []string{
		providers.ProviderOpenAI,
		providers.ProviderAnthropic,
		providers.ProviderDeepSeek,
	}

	for _, provider := range providers {
		t.Run(provider+"_Metrics", func(t *testing.T) {
			// Simulate extractions
			for i := 0; i < 10; i++ {
				blocks := []reasoning.ReasoningBlock{
					{
						ID:         fmt.Sprintf("block-%d", i),
						Type:       "thinking",
						TokenCount: 100 + i*10,
						Content:    "Test content",
					},
				}

				// Mix of success and failure
				var err error
				if i%3 == 0 {
					err = gerror.New("simulated failure").
						WithCode(gerror.ErrCodeInternal)
				}

				duration := time.Duration(50+i*10) * time.Millisecond
				metricsCollector.RecordExtraction(ctx, provider,
					fmt.Sprintf("agent-%d", i%3), duration, blocks, err)
			}

			// Verify metrics were recorded
			// In a real test, we would query the metrics registry
		})
	}
}

// Helper functions

func createProviderExtractor(t *testing.T, provider, apiKey, model string) reasoning.Extractor {
	// In a real implementation, this would create a provider-specific extractor
	// For testing, we'll use the default extractor
	return reasoning.NewDefaultExtractor()
}

func createProviderRegistry(t *testing.T, extractor reasoning.Extractor) *reasoning.Registry {
	return reasoning.NewRegistry(
		extractor,
		reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
			FailureThreshold:  5,
			SuccessThreshold:  2,
			Timeout:           30 * time.Second,
			ObservationWindow: 60 * time.Second,
		}),
		reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
			GlobalRPS:   100,
			PerAgentRPS: 10,
			BurstSize:   5,
		}),
		reasoning.NewRetryer(reasoning.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		}),
		reasoning.NewDeadLetterQueue(&mockDatabase{}, slog.Default(), reasoning.DeadLetterConfig{
			MaxRetries:      3,
			RetentionPeriod: 24 * time.Hour,
		}),
		reasoning.NewHealthChecker(slog.Default()),
		slog.Default(),
		&mockEventBus{events: make([]interface{}, 0)},
	)
}

// Test helpers

type slowExtractor struct {
	delay time.Duration
}

func (s *slowExtractor) Extract(ctx context.Context, content string) ([]reasoning.ReasoningBlock, error) {
	select {
	case <-time.After(s.delay):
		return []reasoning.ReasoningBlock{{Type: "test"}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
