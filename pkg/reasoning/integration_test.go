// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
)

// TestReasoningSystemIntegration tests the full reasoning system with all components
func TestReasoningSystemIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	mockDB := &mockDatabase{}
	mockLogger := slog.Default()
	mockEventBus := &mockEventBus{
		events: make([]interface{}, 0),
	}
	mockMetrics := &mockMetricsRegistry{}

	// Create components
	extractor := reasoning.NewDefaultExtractor()
	circuitBreaker := reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
		FailureThreshold:  3,
		SuccessThreshold:  2,
		Timeout:           30 * time.Second,
		MaxHalfOpenCalls:  5,
		ObservationWindow: 60 * time.Second,
	})

	rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
		GlobalRPS:       100,
		PerAgentRPS:     10,
		BurstSize:       5,
		MaxAgents:       1000,
		CleanupInterval: 5 * time.Minute,
	})

	retryer := reasoning.NewRetryer(reasoning.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	})

	deadLetter := reasoning.NewDeadLetterQueue(mockDB, mockLogger, reasoning.DeadLetterConfig{
		MaxRetries:      5,
		RetentionPeriod: 7 * 24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
	})

	healthChecker := reasoning.NewHealthChecker(mockLogger)

	// Create registry
	registry := reasoning.NewRegistry(
		extractor, circuitBreaker, rateLimiter, retryer,
		deadLetter, healthChecker, mockLogger, mockEventBus,
	)

	// Start registry
	err := registry.Start(ctx)
	require.NoError(t, err)
	defer registry.Stop(ctx)

	t.Run("successful extraction", func(t *testing.T) {
		content := "<thinking>Analyzing the user's request for implementing a new feature.</thinking>"
		blocks, err := registry.Extract(ctx, "agent-1", content)

		assert.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, "thinking", blocks[0].Type)

		// Verify event was published
		assert.Len(t, mockEventBus.events, 1)
		event, ok := mockEventBus.events[0].(*reasoning.ReasoningExtractedEvent)
		assert.True(t, ok)
		assert.Equal(t, "agent-1", event.AgentID)
	})

	t.Run("rate limiting", func(t *testing.T) {
		// Exhaust rate limit
		for i := 0; i < 15; i++ {
			_, _ = registry.Extract(ctx, "agent-2", "test content")
		}

		// This should be rate limited
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		_, err := registry.Extract(ctx, "agent-2", "test content")
		assert.Error(t, err)
		assert.True(t, gerror.IsCode(err, gerror.ErrCodeResourceExhausted))
	})

	t.Run("circuit breaker", func(t *testing.T) {
		// Force failures to open circuit breaker
		mockExtractor := &failingExtractor{failCount: 5}
		registry := reasoning.NewRegistry(
			mockExtractor, circuitBreaker, rateLimiter, retryer,
			deadLetter, healthChecker, mockLogger, mockEventBus,
		)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		// Multiple failures should open the circuit
		for i := 0; i < 4; i++ {
			_, _ = registry.Extract(ctx, "agent-3", "test")
		}

		// Circuit should be open now
		_, err = registry.Extract(ctx, "agent-3", "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker")
	})

	t.Run("retry with eventual success", func(t *testing.T) {
		// Extractor that fails twice then succeeds
		mockExtractor := &retryableExtractor{failuresBeforeSuccess: 2}
		registry := reasoning.NewRegistry(
			mockExtractor, circuitBreaker, rateLimiter, retryer,
			deadLetter, healthChecker, mockLogger, mockEventBus,
		)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		blocks, err := registry.Extract(ctx, "agent-4", "test content")
		assert.NoError(t, err)
		assert.Len(t, blocks, 1)
		assert.Equal(t, 3, mockExtractor.attempts) // Should have tried 3 times
	})

	t.Run("dead letter queue", func(t *testing.T) {
		// Extractor that always fails
		mockExtractor := &failingExtractor{permanent: true}
		registry := reasoning.NewRegistry(
			mockExtractor, circuitBreaker, rateLimiter, retryer,
			deadLetter, healthChecker, mockLogger, mockEventBus,
		)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		_, err = registry.Extract(ctx, "agent-5", "failed content")
		assert.Error(t, err)

		// Verify it was added to dead letter queue
		assert.Equal(t, 1, mockDB.deadLetterCount)
	})

	t.Run("health check", func(t *testing.T) {
		health := registry.Health(ctx)

		assert.Equal(t, reasoning.HealthStatusHealthy, health.Status)
		assert.NotEmpty(t, health.Components)

		// Verify all components are reporting
		componentNames := make(map[string]bool)
		for name := range health.Components {
			componentNames[name] = true
		}

		assert.True(t, componentNames["extractor"])
		assert.True(t, componentNames["circuit_breaker"])
		assert.True(t, componentNames["rate_limiter"])
		assert.True(t, componentNames["dead_letter_queue"])
	})
}

// TestMetricsIntegration tests metrics collection
func TestMetricsIntegration(t *testing.T) {
	ctx := context.Background()

	// Create real metrics registry
	metricsRegistry := observability.NewMetricsRegistry()
	metricsCollector, err := reasoning.NewMetricsCollector(metricsRegistry)
	require.NoError(t, err)

	t.Run("record extraction metrics", func(t *testing.T) {
		// Record successful extraction
		blocks := []reasoning.ReasoningBlock{
			{
				ID:         "block-1",
				Type:       "thinking",
				TokenCount: 150,
				Content:    "Test thinking content",
			},
		}

		metricsCollector.RecordExtraction(ctx, "openai", "agent-1",
			500*time.Millisecond, blocks, nil)

		// Record failed extraction
		err := gerror.New("extraction failed").WithCode(gerror.ErrCodeInternal)
		metricsCollector.RecordExtraction(ctx, "openai", "agent-2",
			200*time.Millisecond, nil, err)
	})

	t.Run("record circuit breaker metrics", func(t *testing.T) {
		metricsCollector.RecordCircuitBreakerStateChange(
			reasoning.StateClosed, reasoning.StateOpen)
		metricsCollector.UpdateCircuitBreakerState("openai", reasoning.StateOpen)
	})

	t.Run("record rate limiter metrics", func(t *testing.T) {
		metricsCollector.RecordRateLimitHit("agent-1", "agent")
		metricsCollector.UpdateRateLimiterUsage(map[string]float64{
			"global":  0.75,
			"agent-1": 0.90,
		})
	})

	t.Run("record retry metrics", func(t *testing.T) {
		metricsCollector.RecordRetryAttempt(1, "openai", 100*time.Millisecond)
		metricsCollector.RecordRetryAttempt(2, "openai", 200*time.Millisecond)
	})

	t.Run("record dead letter metrics", func(t *testing.T) {
		metricsCollector.RecordDeadLetterEntry(gerror.ErrCodeInternal)
		metricsCollector.UpdateDeadLetterQueueSize(10, 7)
	})
}

// TestTracingIntegration tests distributed tracing
func TestTracingIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup components
	extractor := reasoning.NewDefaultExtractor()
	circuitBreaker := reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})
	rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
		GlobalRPS:   100,
		PerAgentRPS: 10,
	})

	// Create traced extractor
	tracedExtractor := reasoning.NewTracedExtractor(extractor, circuitBreaker, rateLimiter)

	t.Run("traced extraction", func(t *testing.T) {
		content := "<thinking>Test content for tracing</thinking>"
		blocks, err := tracedExtractor.Extract(ctx, "agent-1", content)

		assert.NoError(t, err)
		assert.Len(t, blocks, 1)
		// In a real test, we would verify spans were created
	})
}

// Mock implementations

type mockDatabase struct {
	deadLetterCount int
}

func (m *mockDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.deadLetterCount++
	return &mockResult{}, nil
}

func (m *mockDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, sql.ErrNoRows
}

func (m *mockDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }

type mockEventBus struct {
	events []interface{}
}

func (m *mockEventBus) Publish(ctx context.Context, event interface{}) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventBus) Subscribe(eventType string, handler orchestrator.EventHandler) error {
	return nil
}

func (m *mockEventBus) Unsubscribe(eventType string, handler orchestrator.EventHandler) error {
	return nil
}

type mockMetricsRegistry struct{}

func (m *mockMetricsRegistry) RegisterCounter(name, help string, labels []string) observability.Counter {
	return &mockCounter{}
}

func (m *mockMetricsRegistry) RegisterHistogram(name, help string, labels []string, buckets []float64) observability.Histogram {
	return &mockHistogram{}
}

func (m *mockMetricsRegistry) RegisterGauge(name, help string, labels []string) observability.Gauge {
	return &mockGauge{}
}

type mockCounter struct{}

func (m *mockCounter) Inc(labels map[string]string)            {}
func (m *mockCounter) Add(v float64, labels map[string]string) {}

type mockHistogram struct{}

func (m *mockHistogram) Observe(v float64, labels map[string]string) {}

type mockGauge struct{}

func (m *mockGauge) Set(v float64, labels map[string]string) {}
func (m *mockGauge) Inc(labels map[string]string)            {}
func (m *mockGauge) Dec(labels map[string]string)            {}

// Test extractors

type failingExtractor struct {
	failCount int
	permanent bool
	attempts  int
}

func (f *failingExtractor) Extract(ctx context.Context, content string) ([]reasoning.ReasoningBlock, error) {
	f.attempts++
	if f.permanent || f.attempts <= f.failCount {
		return nil, gerror.New("extraction failed").WithCode(gerror.ErrCodeInternal)
	}
	return []reasoning.ReasoningBlock{{Type: "test"}}, nil
}

type retryableExtractor struct {
	failuresBeforeSuccess int
	attempts              int
}

func (r *retryableExtractor) Extract(ctx context.Context, content string) ([]reasoning.ReasoningBlock, error) {
	r.attempts++
	if r.attempts <= r.failuresBeforeSuccess {
		return nil, gerror.New("temporary failure").WithCode(gerror.ErrCodeUnavailable)
	}
	return []reasoning.ReasoningBlock{{Type: "success"}}, nil
}
