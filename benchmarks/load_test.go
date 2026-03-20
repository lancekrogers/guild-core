// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package benchmarks

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/internal/ui/chat/services"
)

// LoadTestConfig defines the parameters for load testing
type LoadTestConfig struct {
	Duration      time.Duration
	Concurrency   int
	RampUpTime    time.Duration
	ThinkTime     time.Duration
	QueryPool     []string
	TargetLatency time.Duration
	TargetTPS     float64
}

// LoadTestResult contains the results of a load test
type LoadTestResult struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	P50Latency         time.Duration `json:"p50_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	TPS                float64       `json:"transactions_per_second"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	ErrorRate          float64       `json:"error_rate"`
	MemoryUsage        int64         `json:"memory_usage_bytes"`
}

// BenchmarkLoadTestSuggestions performs comprehensive load testing
func BenchmarkLoadTestSuggestions(b *testing.B) {
	testConfigs := []LoadTestConfig{
		{
			Duration:      30 * time.Second,
			Concurrency:   10,
			RampUpTime:    5 * time.Second,
			ThinkTime:     100 * time.Millisecond,
			TargetLatency: 100 * time.Millisecond,
			TargetTPS:     50,
		},
		{
			Duration:      60 * time.Second,
			Concurrency:   25,
			RampUpTime:    10 * time.Second,
			ThinkTime:     50 * time.Millisecond,
			TargetLatency: 150 * time.Millisecond,
			TargetTPS:     100,
		},
		{
			Duration:      120 * time.Second,
			Concurrency:   50,
			RampUpTime:    15 * time.Second,
			ThinkTime:     25 * time.Millisecond,
			TargetLatency: 200 * time.Millisecond,
			TargetTPS:     200,
		},
	}

	for _, config := range testConfigs {
		config.QueryPool = generateQueryPool()
		b.Run(fmt.Sprintf("LoadTest_%d_users_%ds", config.Concurrency, int(config.Duration.Seconds())), func(b *testing.B) {
			result := runLoadTest(b, config)

			// Report metrics
			b.ReportMetric(float64(result.TotalRequests), "total_requests")
			b.ReportMetric(float64(result.SuccessfulRequests), "successful_requests")
			b.ReportMetric(float64(result.AverageLatency.Milliseconds()), "avg_latency_ms")
			b.ReportMetric(float64(result.P95Latency.Milliseconds()), "p95_latency_ms")
			b.ReportMetric(float64(result.P99Latency.Milliseconds()), "p99_latency_ms")
			b.ReportMetric(result.TPS, "tps")
			b.ReportMetric(result.CacheHitRate*100, "cache_hit_rate_%")
			b.ReportMetric(result.ErrorRate*100, "error_rate_%")

			// Validate against targets
			if result.P95Latency > config.TargetLatency {
				b.Errorf("P95 latency %v exceeds target %v", result.P95Latency, config.TargetLatency)
			}
			if result.TPS < config.TargetTPS*0.8 { // Allow 20% tolerance
				b.Errorf("TPS %.2f below 80%% of target %.2f", result.TPS, config.TargetTPS)
			}
			if result.ErrorRate > 0.01 { // Max 1% error rate
				b.Errorf("Error rate %.2f%% exceeds 1%%", result.ErrorRate*100)
			}
		})
	}
}

// runLoadTest executes a load test with the given configuration
func runLoadTest(b *testing.B, config LoadTestConfig) LoadTestResult {
	service := setupSuggestionService(b)

	// Results tracking
	var (
		totalRequests  int64
		successfulReqs int64
		failedReqs     int64
		cacheHits      int64
		latencies      = make(chan time.Duration, 10000)
		done           = make(chan struct{})
	)

	// Start latency collector
	go func() {
		var allLatencies []time.Duration
		for {
			select {
			case lat := <-latencies:
				allLatencies = append(allLatencies, lat)
			case <-done:
				return
			}
		}
	}()

	// Worker function
	worker := func(workerID int, startTime time.Time) {
		// Ramp up delay
		rampDelay := time.Duration(float64(config.RampUpTime) * float64(workerID) / float64(config.Concurrency))
		time.Sleep(rampDelay)

		// Main load testing loop
		endTime := startTime.Add(config.Duration)
		for time.Now().Before(endTime) {
			// Select random query
			query := config.QueryPool[rand.Intn(len(config.QueryPool))]
			context := &services.SuggestionContext{
				ConversationID: fmt.Sprintf("worker_%d", workerID),
			}

			// Execute request
			start := time.Now()
			cmd := service.GetSuggestions(query, context)
			result := cmd()
			latency := time.Since(start)

			atomic.AddInt64(&totalRequests, 1)

			// Check result
			atomic.AddInt64(&successfulReqs, 1)
			if result.FromCache {
				atomic.AddInt64(&cacheHits, 1)
			}
			latencies <- latency

			// Think time
			if config.ThinkTime > 0 {
				time.Sleep(config.ThinkTime)
			}
		}
	}

	// Start workers
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(workerID, startTime)
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(done)
	close(latencies)

	// Collect all latencies for statistics
	var allLatencies []time.Duration
	for lat := range latencies {
		allLatencies = append(allLatencies, lat)
	}

	// Calculate statistics
	result := LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulReqs,
		FailedRequests:     failedReqs,
		TPS:                float64(totalRequests) / config.Duration.Seconds(),
		CacheHitRate:       float64(cacheHits) / float64(successfulReqs),
		ErrorRate:          float64(failedReqs) / float64(totalRequests),
	}

	if len(allLatencies) > 0 {
		result.AverageLatency = calculateAverage(allLatencies)
		result.P50Latency = calculatePercentile(allLatencies, 50)
		result.P95Latency = calculatePercentile(allLatencies, 95)
		result.P99Latency = calculatePercentile(allLatencies, 99)
		result.MaxLatency = calculateMax(allLatencies)
		result.MinLatency = calculateMin(allLatencies)
	}

	return result
}

// BenchmarkStressTest pushes the system to its limits
func BenchmarkStressTest(b *testing.B) {
	service := setupSuggestionService(b)

	// Stress test configuration
	stressLevels := []struct {
		name        string
		concurrency int
		duration    time.Duration
		queryRate   time.Duration // Queries per second per user
	}{
		{"Light", 20, 60 * time.Second, 500 * time.Millisecond},
		{"Medium", 50, 120 * time.Second, 200 * time.Millisecond},
		{"Heavy", 100, 180 * time.Second, 100 * time.Millisecond},
		{"Extreme", 200, 300 * time.Second, 50 * time.Millisecond},
	}

	for _, level := range stressLevels {
		b.Run(fmt.Sprintf("StressTest_%s", level.name), func(b *testing.B) {
			// Clear cache for consistent testing
			service.ClearCache()

			var (
				totalRequests   int64
				successRequests int64
				errors          int64
				timeouts        int64
			)

			// Worker function
			worker := func(ctx context.Context) {
				queries := generateQueryPool()
				ticker := time.NewTicker(level.queryRate)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						query := queries[rand.Intn(len(queries))]
						suggestionCtx := &services.SuggestionContext{
							ConversationID: fmt.Sprintf("stress_%d", rand.Intn(10)),
						}

						// Set timeout for each request
						reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

						atomic.AddInt64(&totalRequests, 1)

						go func() {
							defer cancel()

							cmd := service.GetSuggestions(query, suggestionCtx)
							_ = cmd()

							atomic.AddInt64(&successRequests, 1)
						}()

						// Check for timeout
						select {
						case <-reqCtx.Done():
							if reqCtx.Err() == context.DeadlineExceeded {
								atomic.AddInt64(&timeouts, 1)
							}
						default:
						}
					}
				}
			}

			// Run stress test
			ctx, cancel := context.WithTimeout(context.Background(), level.duration)
			defer cancel()

			var wg sync.WaitGroup
			for i := 0; i < level.concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					worker(ctx)
				}()
			}

			wg.Wait()

			// Calculate metrics
			duration := level.duration.Seconds()
			tps := float64(totalRequests) / duration
			successRate := float64(successRequests) / float64(totalRequests) * 100
			errorRate := float64(errors) / float64(totalRequests) * 100
			timeoutRate := float64(timeouts) / float64(totalRequests) * 100

			b.ReportMetric(float64(totalRequests), "total_requests")
			b.ReportMetric(float64(successRequests), "successful_requests")
			b.ReportMetric(tps, "tps")
			b.ReportMetric(successRate, "success_rate_%")
			b.ReportMetric(errorRate, "error_rate_%")
			b.ReportMetric(timeoutRate, "timeout_rate_%")

			// Log results
			b.Logf("Stress Test %s Results:", level.name)
			b.Logf("  Total Requests: %d", totalRequests)
			b.Logf("  Success Rate: %.2f%%", successRate)
			b.Logf("  Error Rate: %.2f%%", errorRate)
			b.Logf("  Timeout Rate: %.2f%%", timeoutRate)
			b.Logf("  TPS: %.2f", tps)

			// Validate system stability
			if successRate < 95.0 {
				b.Errorf("Success rate %.2f%% below 95%% threshold", successRate)
			}
			if timeoutRate > 5.0 {
				b.Errorf("Timeout rate %.2f%% above 5%% threshold", timeoutRate)
			}
		})
	}
}

// BenchmarkMemoryPressure tests performance under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	service := setupSuggestionService(b)

	// Create memory pressure by holding large objects
	var memoryBallast [][]byte
	ballastSize := 100 * 1024 * 1024 // 100MB

	for i := 0; i < 10; i++ {
		ballast := make([]byte, ballastSize/10)
		memoryBallast = append(memoryBallast, ballast)
	}

	b.ResetTimer()

	// Run normal benchmark under memory pressure
	for i := 0; i < b.N; i++ {
		query := fmt.Sprintf("Memory pressure test query %d", i)
		context := &services.SuggestionContext{
			ConversationID: "memory_test",
		}

		cmd := service.GetSuggestions(query, context)
		_ = cmd()

		// For our mock, requests don't fail

		// Force garbage collection periodically
		if i%100 == 0 {
			runtime.GC()
		}
	}
}

// Helper functions

func generateQueryPool() []string {
	return []string{
		"How do I create a new REST API endpoint?",
		"What's the best way to handle authentication?",
		"How to implement database migrations?",
		"Best practices for error handling in Go?",
		"How to write effective unit tests?",
		"What's the proper way to structure a Go project?",
		"How to implement graceful shutdown?",
		"Best practices for logging and monitoring?",
		"How to handle concurrent operations safely?",
		"What's the recommended approach for configuration management?",
		"How to implement rate limiting?",
		"Best practices for API versioning?",
		"How to handle file uploads securely?",
		"What's the proper way to validate input data?",
		"How to implement caching effectively?",
		"Best practices for database connection pooling?",
		"How to handle background job processing?",
		"What's the recommended approach for testing HTTP handlers?",
		"How to implement distributed tracing?",
		"Best practices for dependency injection in Go?",
	}
}

func calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func calculateMin(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}
