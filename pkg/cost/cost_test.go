// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCostTracker tests the cost tracking system
func TestCostTracker(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCostTracker", func(t *testing.T) {
		config := &TrackerConfig{
			UpdateInterval:       5 * time.Second,
			RetentionPeriod:      24 * time.Hour,
			AggregationWindow:    5 * time.Minute,
			EnableRealTimeAlerts: true,
			BudgetLimits:         map[string]float64{"monthly": 1000.0},
		}

		tracker, err := NewCostTracker(ctx, config)
		assert.NoError(t, err)
		assert.NotNil(t, tracker)
		assert.Equal(t, config.UpdateInterval, tracker.config.UpdateInterval)
	})

	t.Run("RegisterProvider", func(t *testing.T) {
		tracker := setupTestTracker(t)

		provider, err := NewOpenAICostProvider(ctx, "test-api-key")
		require.NoError(t, err)

		err = tracker.RegisterProvider(ctx, provider)
		assert.NoError(t, err)

		// Test duplicate registration (should work)
		err = tracker.RegisterProvider(ctx, provider)
		assert.NoError(t, err)
	})

	t.Run("TrackUsage", func(t *testing.T) {
		tracker := setupTestTracker(t)
		setupTestProviders(t, tracker)

		usage := Usage{
			AgentID:   "test-agent",
			Provider:  "openai",
			Resource:  "completion",
			Quantity:  1000,
			Unit:      "tokens",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"model":         "gpt-4",
				"input_tokens":  800,
				"output_tokens": 200,
				"total_cost":    0.048,
			},
		}

		err := tracker.TrackUsage(ctx, usage)
		assert.NoError(t, err)
	})

	t.Run("GetCurrentCosts", func(t *testing.T) {
		tracker := setupTestTracker(t)
		setupTestProviders(t, tracker)

		// Track some usage first
		usages := []Usage{
			{
				AgentID:   "elena",
				Provider:  "openai",
				Resource:  "completion",
				Quantity:  1000,
				Unit:      "tokens",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"model":         "gpt-4",
					"input_tokens":  800,
					"output_tokens": 200,
					"total_cost":    0.048,
				},
			},
			{
				AgentID:   "marcus",
				Provider:  "anthropic",
				Resource:  "completion",
				Quantity:  2000,
				Unit:      "tokens",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"model":         "claude-3-opus-20240229",
					"input_tokens":  1500,
					"output_tokens": 500,
					"total_cost":    0.06,
				},
			},
		}

		for _, usage := range usages {
			err := tracker.TrackUsage(ctx, usage)
			require.NoError(t, err)
		}

		summary, err := tracker.GetCurrentCosts(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, "USD", summary.Currency)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		tracker := setupTestTracker(t)

		// Test invalid usage
		invalidUsages := []Usage{
			{
				// Missing AgentID
				Provider:  "openai",
				Resource:  "completion",
				Quantity:  1000,
				Unit:      "tokens",
				Timestamp: time.Now(),
			},
			{
				AgentID: "test",
				// Missing Provider
				Resource:  "completion",
				Quantity:  1000,
				Unit:      "tokens",
				Timestamp: time.Now(),
			},
			{
				AgentID:  "test",
				Provider: "openai",
				// Missing Resource
				Quantity:  1000,
				Unit:      "tokens",
				Timestamp: time.Now(),
			},
			{
				AgentID:   "test",
				Provider:  "openai",
				Resource:  "completion",
				Quantity:  -1000, // Negative quantity
				Unit:      "tokens",
				Timestamp: time.Now(),
			},
		}

		for _, usage := range invalidUsages {
			err := tracker.TrackUsage(ctx, usage)
			assert.Error(t, err)
		}
	})
}

// TestCostProviders tests cost provider implementations
func TestCostProviders(t *testing.T) {
	ctx := context.Background()

	t.Run("OpenAICostProvider", func(t *testing.T) {
		provider, err := NewOpenAICostProvider(ctx, "test-api-key")
		require.NoError(t, err)
		assert.Equal(t, "openai", provider.GetProviderName())

		// Test rate card
		rates, err := provider.GetRates(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, rates)
		assert.Equal(t, "openai", rates.Provider)
		assert.Contains(t, rates.Rates, "gpt-4")

		// Test usage tracking
		usage := Usage{
			AgentID:   "test-agent",
			Provider:  "openai",
			Resource:  "completion",
			Quantity:  1000,
			Unit:      "tokens",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"model":         "gpt-4",
				"input_tokens":  800,
				"output_tokens": 200,
			},
		}

		err = provider.TrackUsage(ctx, usage)
		assert.NoError(t, err)
	})

	t.Run("AnthropicCostProvider", func(t *testing.T) {
		provider, err := NewAnthropicCostProvider(ctx, "test-api-key")
		require.NoError(t, err)
		assert.Equal(t, "anthropic", provider.GetProviderName())

		// Test rate card
		rates, err := provider.GetRates(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, rates)
		assert.Equal(t, "anthropic", rates.Provider)
		assert.Contains(t, rates.Rates, "claude-3-opus-20240229")
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		_, err := NewOpenAICostProvider(ctx, "")
		assert.Error(t, err)

		_, err = NewAnthropicCostProvider(ctx, "")
		assert.Error(t, err)
	})
}

// TestCostAggregator tests the cost aggregation system
func TestCostAggregator(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCostAggregator", func(t *testing.T) {
		tracker := setupTestTracker(t)

		aggregator, err := NewCostAggregator(ctx, tracker, 5*time.Minute)
		assert.NoError(t, err)
		assert.NotNil(t, aggregator)
	})

	t.Run("GetCurrentCosts", func(t *testing.T) {
		tracker := setupTestTracker(t)
		aggregator, err := NewCostAggregator(ctx, tracker, 5*time.Minute)
		require.NoError(t, err)

		summary, err := aggregator.GetCurrentCosts(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, "USD", summary.Currency)
	})

	t.Run("Subscribe", func(t *testing.T) {
		tracker := setupTestTracker(t)
		aggregator, err := NewCostAggregator(ctx, tracker, 5*time.Minute)
		require.NoError(t, err)

		updateCh := make(chan CostUpdate, 10)
		aggregator.Subscribe(updateCh)

		// Start aggregator briefly
		go aggregator.Start(ctx)
		time.Sleep(100 * time.Millisecond)

		err = aggregator.Stop(ctx)
		assert.NoError(t, err)
	})

	t.Run("ProjectionCalculation", func(t *testing.T) {
		tracker := setupTestTracker(t)
		aggregator, err := NewCostAggregator(ctx, tracker, 5*time.Minute)
		require.NoError(t, err)

		projections := aggregator.GetProjections(ctx)
		assert.NotNil(t, projections)
		assert.GreaterOrEqual(t, projections.Confidence, 0.0)
		assert.LessOrEqual(t, projections.Confidence, 1.0)
	})
}

// TestCostCalculator tests the cost calculation utilities
func TestCostCalculator(t *testing.T) {
	ctx := context.Background()
	calculator, err := NewCostCalculator(ctx)
	require.NoError(t, err)

	t.Run("CalculateTokenCost", func(t *testing.T) {
		// Test valid calculation
		cost, err := calculator.CalculateTokenCost(ctx, 1000, 30.0) // 1000 tokens at $30 per 1M
		assert.NoError(t, err)
		assert.Equal(t, 0.03, cost) // $0.03

		// Test zero tokens
		cost, err = calculator.CalculateTokenCost(ctx, 0, 30.0)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, cost)

		// Test negative tokens
		_, err = calculator.CalculateTokenCost(ctx, -1000, 30.0)
		assert.Error(t, err)

		// Test negative rate
		_, err = calculator.CalculateTokenCost(ctx, 1000, -30.0)
		assert.Error(t, err)
	})

	t.Run("EstimateCostForTask", func(t *testing.T) {
		estimate, err := calculator.EstimateCostForTask(ctx, 5, "gpt-4", 2000)
		assert.NoError(t, err)
		assert.NotNil(t, estimate)
		assert.Equal(t, "gpt-4", estimate.ModelType)
		assert.Equal(t, 2000, estimate.EstimatedTokens)
		assert.Greater(t, estimate.TotalCost, 0.0)
		assert.GreaterOrEqual(t, estimate.Confidence, 0.0)
		assert.LessOrEqual(t, estimate.Confidence, 1.0)

		// Test unknown model
		_, err = calculator.EstimateCostForTask(ctx, 5, "unknown-model", 2000)
		assert.Error(t, err)
	})

	t.Run("CalculateProjectedCost", func(t *testing.T) {
		// Test with empty usage
		projection, err := calculator.CalculateProjectedCost(ctx, []Usage{}, time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, projection)
		assert.Equal(t, 0.0, projection.HourlyRate)

		// Test with sample usage
		usage := []Usage{
			{
				AgentID:   "test",
				Timestamp: time.Now().Add(-time.Hour),
				Metadata: map[string]interface{}{
					"total_cost": 0.05,
				},
			},
			{
				AgentID:   "test",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"total_cost": 0.03,
				},
			},
		}

		projection, err = calculator.CalculateProjectedCost(ctx, usage, time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, projection)
		assert.Greater(t, projection.HourlyRate, 0.0)
	})

	t.Run("CalculateSavings", func(t *testing.T) {
		currentCost := &CostEstimate{
			TotalCost:  0.10,
			Confidence: 0.9,
			Currency:   "USD",
		}

		optimizedCost := &CostEstimate{
			TotalCost:  0.06,
			Confidence: 0.8,
			Currency:   "USD",
		}

		savings, err := calculator.CalculateSavings(ctx, currentCost, optimizedCost)
		assert.NoError(t, err)
		assert.NotNil(t, savings)
		assert.InDelta(t, 0.04, savings.AbsoluteSavings, 0.001)
		assert.InDelta(t, 40.0, savings.PercentageSavings, 0.001)
		assert.Equal(t, 0.8, savings.Confidence) // Min of both
	})
}

// TestCostOptimizer tests the optimization algorithms
func TestCostOptimizer(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCostOptimizer", func(t *testing.T) {
		tracker := setupTestTracker(t)
		config := &OptimizerConfig{
			MinSavingsThreshold: 5.0,
			MaxImpactThreshold:  0.2,
			AnalysisWindow:      24 * time.Hour,
			EnableAutoApply:     false,
		}

		optimizer, err := NewCostOptimizer(ctx, tracker, config)
		assert.NoError(t, err)
		assert.NotNil(t, optimizer)
		assert.Len(t, optimizer.strategies, 6) // All strategies registered
	})

	t.Run("AnalyzeOptimizations", func(t *testing.T) {
		tracker := setupTestTracker(t)
		optimizer, err := NewCostOptimizer(ctx, tracker, nil)
		require.NoError(t, err)

		optimizations, err := optimizer.Analyze(ctx)
		assert.NoError(t, err)
		// optimizations may be empty if no opportunities found
		_ = optimizations
	})

	t.Run("ModelOptimizer", func(t *testing.T) {
		tracker := setupTestTracker(t)
		modelOpt := NewModelOptimizer(ctx, tracker)

		assert.Equal(t, "model_optimizer", modelOpt.Name())
		assert.Equal(t, 100, modelOpt.Priority())

		// Test with sample usage
		usage := []Usage{
			{
				AgentID:   "test",
				Provider:  "openai",
				Resource:  "completion",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"model":      "gpt-4",
					"task_type":  "simple_qa",
					"success":    true,
					"total_cost": 0.10,
				},
			},
		}

		optimizations, err := modelOpt.Analyze(ctx, usage)
		assert.NoError(t, err)
		// optimizations may be empty if no opportunities found
		_ = optimizations
	})
}

// TestCostStorage tests the storage system
func TestCostStorage(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCostStorage", func(t *testing.T) {
		// Skip if database not available
		if testing.Short() {
			t.Skip("Skipping storage test in short mode")
		}

		storage, err := NewCostStorage(ctx)
		if err != nil {
			t.Skip("Database not available, skipping storage tests")
		}
		assert.NotNil(t, storage)
	})

	// Additional storage tests would go here if database is available
}

// BenchmarkCostTracking benchmarks cost tracking performance
func BenchmarkCostTracking(b *testing.B) {
	ctx := context.Background()
	tracker := setupBenchmarkTracker(b)

	usage := Usage{
		AgentID:   "bench-agent",
		Provider:  "openai",
		Resource:  "completion",
		Quantity:  1000,
		Unit:      "tokens",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"model":         "gpt-4",
			"input_tokens":  800,
			"output_tokens": 200,
			"total_cost":    0.048,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		usage.Timestamp = time.Now()
		err := tracker.TrackUsage(ctx, usage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptimization benchmarks optimization analysis
func BenchmarkOptimization(b *testing.B) {
	ctx := context.Background()
	tracker := setupBenchmarkTracker(b)
	optimizer, err := NewCostOptimizer(ctx, tracker, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := optimizer.Analyze(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for tests

// setupTestTracker creates a test cost tracker
func setupTestTracker(t *testing.T) *CostTracker {
	ctx := context.Background()
	config := &TrackerConfig{
		UpdateInterval:       time.Second,
		RetentionPeriod:      time.Hour,
		AggregationWindow:    time.Minute,
		EnableRealTimeAlerts: false,
		BudgetLimits:         map[string]float64{"monthly": 1000.0},
	}

	tracker, err := NewCostTracker(ctx, config)
	require.NoError(t, err)
	return tracker
}

// setupBenchmarkTracker creates a tracker for benchmarking
func setupBenchmarkTracker(b *testing.B) *CostTracker {
	ctx := context.Background()
	config := &TrackerConfig{
		UpdateInterval:       time.Second,
		RetentionPeriod:      time.Hour,
		AggregationWindow:    time.Minute,
		EnableRealTimeAlerts: false,
		BudgetLimits:         map[string]float64{"monthly": 10000.0},
	}

	tracker, err := NewCostTracker(ctx, config)
	if err != nil {
		b.Fatal(err)
	}
	return tracker
}

// setupTestProviders registers test providers
func setupTestProviders(t *testing.T, tracker *CostTracker) {
	ctx := context.Background()

	// Setup OpenAI provider
	openaiProvider, err := NewOpenAICostProvider(ctx, "test-openai-key")
	require.NoError(t, err)
	err = tracker.RegisterProvider(ctx, openaiProvider)
	require.NoError(t, err)

	// Setup Anthropic provider
	anthropicProvider, err := NewAnthropicCostProvider(ctx, "test-anthropic-key")
	require.NoError(t, err)
	err = tracker.RegisterProvider(ctx, anthropicProvider)
	require.NoError(t, err)
}

// setupTestDatabase creates a test database (if needed)
func setupTestDatabase(t *testing.T) {
	// Set test database path
	os.Setenv("GUILD_DB_PATH", ":memory:")
}

// cleanupTestDatabase cleans up test database
func cleanupTestDatabase(t *testing.T) {
	os.Unsetenv("GUILD_DB_PATH")
}
