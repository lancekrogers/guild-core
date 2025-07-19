// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadHandling tests the system under various load conditions
func TestLoadHandling(t *testing.T) {
	ctx := context.Background()

	// Setup system with production-like settings
	extractor := reasoning.NewDefaultExtractor()
	circuitBreaker := reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
		FailureThreshold:  10,
		SuccessThreshold:  5,
		Timeout:           30 * time.Second,
		MaxHalfOpenCalls:  10,
		ObservationWindow: 60 * time.Second,
	})

	rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
		GlobalRPS:       1000,
		PerAgentRPS:     50,
		BurstSize:       10,
		MaxAgents:       100,
		CleanupInterval: 1 * time.Minute,
	})

	retryer := reasoning.NewRetryer(reasoning.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	})

	deadLetter := reasoning.NewDeadLetterQueue(&mockDatabase{}, slog.Default(), reasoning.DeadLetterConfig{
		MaxRetries:      5,
		RetentionPeriod: 24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
	})

	healthChecker := reasoning.NewHealthChecker(slog.Default())
	mockEventBus := &mockEventBus{events: make([]interface{}, 0)}

	registry := reasoning.NewRegistry(
		extractor, circuitBreaker, rateLimiter, retryer,
		deadLetter, healthChecker, slog.Default(), mockEventBus,
	)

	err := registry.Start(ctx)
	require.NoError(t, err)
	defer registry.Stop(ctx)

	t.Run("concurrent extractions", func(t *testing.T) {
		const numAgents = 10
		const requestsPerAgent = 100

		var wg sync.WaitGroup
		var successCount atomic.Int64
		var errorCount atomic.Int64
		var totalDuration atomic.Int64

		startTime := time.Now()

		// Launch concurrent agents
		for i := 0; i < numAgents; i++ {
			agentID := fmt.Sprintf("agent-%d", i)
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				for j := 0; j < requestsPerAgent; j++ {
					content := fmt.Sprintf("<thinking>Processing request %d for %s</thinking>", j, id)
					reqStart := time.Now()

					_, err := registry.Extract(ctx, id, content)

					duration := time.Since(reqStart)
					totalDuration.Add(duration.Nanoseconds())

					if err != nil {
						errorCount.Add(1)
					} else {
						successCount.Add(1)
					}
				}
			}(agentID)
		}

		wg.Wait()
		elapsed := time.Since(startTime)

		// Calculate metrics
		totalRequests := int64(numAgents * requestsPerAgent)
		avgDuration := time.Duration(totalDuration.Load() / totalRequests)
		throughput := float64(totalRequests) / elapsed.Seconds()

		t.Logf("Load Test Results:")
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Successful: %d", successCount.Load())
		t.Logf("  Failed: %d", errorCount.Load())
		t.Logf("  Total time: %v", elapsed)
		t.Logf("  Throughput: %.2f req/s", throughput)
		t.Logf("  Avg latency: %v", avgDuration)

		// Assert performance requirements
		assert.Greater(t, throughput, 100.0, "Throughput should exceed 100 req/s")
		assert.Less(t, avgDuration, 100*time.Millisecond, "Average latency should be under 100ms")
		assert.Less(t, float64(errorCount.Load())/float64(totalRequests), 0.05, "Error rate should be under 5%")
	})

	t.Run("burst handling", func(t *testing.T) {
		const burstSize = 50
		const agentID = "burst-agent"

		var wg sync.WaitGroup
		var accepted atomic.Int64
		var rejected atomic.Int64

		// Send burst of requests
		for i := 0; i < burstSize; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()

				content := fmt.Sprintf("<thinking>Burst request %d</thinking>", idx)
				_, err := registry.Extract(ctx, agentID, content)

				if err != nil {
					rejected.Add(1)
				} else {
					accepted.Add(1)
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Burst Test Results:")
		t.Logf("  Burst size: %d", burstSize)
		t.Logf("  Accepted: %d", accepted.Load())
		t.Logf("  Rejected: %d", rejected.Load())

		// Should accept some but not all in a burst
		assert.Greater(t, accepted.Load(), int64(0), "Should accept some requests")
		assert.Greater(t, rejected.Load(), int64(0), "Should reject some requests due to rate limiting")
	})

	t.Run("sustained load", func(t *testing.T) {
		const duration = 10 * time.Second
		const numAgents = 5

		ctx, cancel := context.WithTimeout(ctx, duration)
		defer cancel()

		var wg sync.WaitGroup
		var requestCount atomic.Int64
		var errorCount atomic.Int64
		stopCh := make(chan struct{})

		// Monitor health during load
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					health := registry.Health(ctx)
					if health.Status != reasoning.HealthStatusHealthy {
						t.Logf("Health degraded: %v", health.Status)
					}
				case <-stopCh:
					return
				}
			}
		}()

		// Generate sustained load
		for i := 0; i < numAgents; i++ {
			agentID := fmt.Sprintf("sustained-agent-%d", i)
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					default:
						content := "<thinking>Sustained load test</thinking>"
						_, err := registry.Extract(context.Background(), id, content)

						requestCount.Add(1)
						if err != nil {
							errorCount.Add(1)
						}

						// Small delay to maintain steady rate
						time.Sleep(10 * time.Millisecond)
					}
				}
			}(agentID)
		}

		// Wait for duration
		<-ctx.Done()
		close(stopCh)
		wg.Wait()

		// Calculate sustained metrics
		totalReqs := requestCount.Load()
		errorRate := float64(errorCount.Load()) / float64(totalReqs)
		sustainedThroughput := float64(totalReqs) / duration.Seconds()

		t.Logf("Sustained Load Results:")
		t.Logf("  Duration: %v", duration)
		t.Logf("  Total requests: %d", totalReqs)
		t.Logf("  Error rate: %.2f%%", errorRate*100)
		t.Logf("  Sustained throughput: %.2f req/s", sustainedThroughput)

		// Assert sustained performance
		assert.Less(t, errorRate, 0.01, "Error rate should be under 1% during sustained load")
		assert.Greater(t, sustainedThroughput, 200.0, "Should maintain >200 req/s sustained")
	})
}

// TestChaosEngineering tests system resilience
func TestChaosEngineering(t *testing.T) {
	ctx := context.Background()

	t.Run("intermittent failures", func(t *testing.T) {
		// Create extractor that fails 30% of the time
		chaosExtractor := &chaosExtractor{
			failureRate:   0.3,
			baseExtractor: reasoning.NewDefaultExtractor(),
		}

		circuitBreaker := reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
			FailureThreshold:  5,
			SuccessThreshold:  3,
			Timeout:           10 * time.Second,
			ObservationWindow: 30 * time.Second,
		})

		rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
			GlobalRPS:   100,
			PerAgentRPS: 10,
		})

		retryer := reasoning.NewRetryer(reasoning.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
		})

		registry := reasoning.NewRegistry(
			chaosExtractor, circuitBreaker, rateLimiter, retryer,
			reasoning.NewDeadLetterQueue(&mockDatabase{}, slog.Default(), reasoning.DeadLetterConfig{}),
			reasoning.NewHealthChecker(slog.Default()),
			slog.Default(),
			&mockEventBus{events: make([]interface{}, 0)},
		)

		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		// Run requests and measure resilience
		const numRequests = 100
		var successCount int
		var circuitOpenCount int

		for i := 0; i < numRequests; i++ {
			content := fmt.Sprintf("<thinking>Chaos test %d</thinking>", i)
			_, err := registry.Extract(ctx, "chaos-agent", content)

			if err == nil {
				successCount++
			} else if reasoning.IsCircuitBreakerOpen(err) {
				circuitOpenCount++
			}

			// Small delay between requests
			time.Sleep(10 * time.Millisecond)
		}

		successRate := float64(successCount) / float64(numRequests)
		t.Logf("Chaos Test Results:")
		t.Logf("  Success rate: %.2f%%", successRate*100)
		t.Logf("  Circuit opened: %d times", circuitOpenCount)

		// With 30% failure rate and retries, we should still achieve decent success
		assert.Greater(t, successRate, 0.7, "Should achieve >70% success with retries")
	})

	t.Run("memory pressure", func(t *testing.T) {
		// Test with large content that could cause memory issues
		largeContent := make([]byte, 1024*1024) // 1MB
		for i := range largeContent {
			largeContent[i] = byte('a' + (i % 26))
		}

		content := fmt.Sprintf("<thinking>%s</thinking>", string(largeContent))

		registry := createTestRegistry()
		err := registry.Start(ctx)
		require.NoError(t, err)
		defer registry.Stop(ctx)

		// Should handle large content gracefully
		blocks, err := registry.Extract(ctx, "memory-test", content)
		assert.NoError(t, err)
		assert.NotEmpty(t, blocks)
	})
}

// Benchmark tests

func BenchmarkReasoningExtraction(b *testing.B) {
	ctx := context.Background()
	registry := createTestRegistry()

	err := registry.Start(ctx)
	require.NoError(b, err)
	defer registry.Stop(ctx)

	content := "<thinking>Benchmark test content for performance measurement</thinking>"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentID := fmt.Sprintf("bench-agent-%d", time.Now().UnixNano())
		for pb.Next() {
			_, _ = registry.Extract(ctx, agentID, content)
		}
	})
}

func BenchmarkRateLimiter(b *testing.B) {
	rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
		GlobalRPS:   10000,
		PerAgentRPS: 1000,
		BurstSize:   100,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentID := fmt.Sprintf("bench-%d", time.Now().UnixNano())
		for pb.Next() {
			_ = rateLimiter.Allow(ctx, agentID)
		}
	})
}

// Helper functions

func createTestRegistry() *reasoning.Registry {
	return reasoning.NewRegistry(
		reasoning.NewDefaultExtractor(),
		reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		}),
		reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
			GlobalRPS:   1000,
			PerAgentRPS: 100,
		}),
		reasoning.NewRetryer(reasoning.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
		}),
		reasoning.NewDeadLetterQueue(&mockDatabase{}, slog.Default(), reasoning.DeadLetterConfig{}),
		reasoning.NewHealthChecker(slog.Default()),
		slog.Default(),
		&mockEventBus{events: make([]interface{}, 0)},
	)
}

// Chaos extractor for testing

type chaosExtractor struct {
	failureRate   float64
	baseExtractor reasoning.Extractor
	mu            sync.Mutex
	requestCount  int
}

func (c *chaosExtractor) Extract(ctx context.Context, content string) ([]reasoning.ReasoningBlock, error) {
	c.mu.Lock()
	c.requestCount++
	shouldFail := (c.requestCount % int(1/c.failureRate)) == 0
	c.mu.Unlock()

	if shouldFail {
		return nil, gerror.New("chaos failure").WithCode(gerror.ErrCodeInternal)
	}

	return c.baseExtractor.Extract(ctx, content)
}
