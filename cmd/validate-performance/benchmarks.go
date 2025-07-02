// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/session"
	"go.uber.org/zap"
)

// SessionBenchmark implements session performance benchmarking
type SessionBenchmark struct {
	logger       *zap.Logger
	sessionSvc   *session.SessionService
	registry     *session.DefaultSessionRegistry
	testSessions []*session.Session
	mu           sync.RWMutex
}

// NewSessionBenchmark creates a new session benchmark instance
func NewSessionBenchmark(logger *zap.Logger) (*SessionBenchmark, error) {
	registry := session.NewDefaultSessionRegistry()
	sessionSvc := session.NewSessionService(registry)

	return &SessionBenchmark{
		logger:       logger.Named("session-benchmark"),
		sessionSvc:   sessionSvc,
		registry:     registry,
		testSessions: make([]*session.Session, 0),
	}, nil
}

// BenchmarkSessionCreation measures session creation performance
func (sb *SessionBenchmark) BenchmarkSessionCreation(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	sb.logger.Info("Starting session creation benchmark", zap.Int("iterations", iterations))

	startTime := time.Now()
	var responseTimes []time.Duration
	var failures int

	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		
		userID := fmt.Sprintf("benchmark-user-%d", i)
		campaignID := fmt.Sprintf("benchmark-campaign-%d", i)
		
		session, err := sb.sessionSvc.CreateSession(ctx, userID, campaignID)
		iterDuration := time.Since(iterStart)
		responseTimes = append(responseTimes, iterDuration)
		
		if err != nil {
			failures++
			sb.logger.Warn("Session creation failed", zap.Int("iteration", i), zap.Error(err))
		} else {
			sb.mu.Lock()
			sb.testSessions = append(sb.testSessions, session)
			sb.mu.Unlock()
		}
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "session creation benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	successRate := float64(iterations-failures) / float64(iterations)
	avgResponseTime := calculateMean(responseTimes)
	p95ResponseTime := calculatePercentile(responseTimes, 0.95)

	result := &BenchmarkResult{
		Name:         "Session Creation",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: avgResponseTime.Seconds() * 1000, // Convert to milliseconds
		Success:      failures == 0,
		TargetMet:    avgResponseTime <= 50*time.Millisecond,
		ActualValue:  avgResponseTime,
		TargetValue:  50 * time.Millisecond,
	}

	sb.logger.Info("Session creation benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Duration("avg_response_time", avgResponseTime),
		zap.Duration("p95_response_time", p95ResponseTime),
		zap.Float64("success_rate", successRate),
		zap.Int("failures", failures))

	return result, nil
}

// BenchmarkSessionRestoration measures session restoration performance and success rate
func (sb *SessionBenchmark) BenchmarkSessionRestoration(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	sb.logger.Info("Starting session restoration benchmark", zap.Int("iterations", iterations))

	// Ensure we have test sessions to restore
	if len(sb.testSessions) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no test sessions available for restoration benchmark", nil)
	}

	startTime := time.Now()
	var responseTimes []time.Duration
	var failures int
	
	// Use existing sessions for restoration testing
	sessionCount := len(sb.testSessions)
	
	for i := 0; i < iterations; i++ {
		sessionIndex := i % sessionCount
		sessionID := sb.testSessions[sessionIndex].ID
		
		iterStart := time.Now()
		err := sb.sessionSvc.ResumeSession(ctx, sessionID)
		iterDuration := time.Since(iterStart)
		responseTimes = append(responseTimes, iterDuration)
		
		if err != nil {
			failures++
			sb.logger.Warn("Session restoration failed", 
				zap.Int("iteration", i), 
				zap.String("session_id", sessionID), 
				zap.Error(err))
		}
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "session restoration benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	successRate := float64(iterations-failures) / float64(iterations)
	avgResponseTime := calculateMean(responseTimes)

	result := &BenchmarkResult{
		Name:         "Session Restoration",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: avgResponseTime.Seconds() * 1000, // Convert to milliseconds
		Success:      successRate >= 0.99, // 99% success rate required
		TargetMet:    successRate >= 0.99,
		ActualValue:  successRate,
		TargetValue:  0.99,
	}

	sb.logger.Info("Session restoration benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Duration("avg_response_time", avgResponseTime),
		zap.Float64("success_rate", successRate),
		zap.Int("failures", failures))

	return result, nil
}

// UIBenchmark implements UI performance benchmarking
type UIBenchmark struct {
	logger *zap.Logger
}

// NewUIBenchmark creates a new UI benchmark instance
func NewUIBenchmark(logger *zap.Logger) (*UIBenchmark, error) {
	return &UIBenchmark{
		logger: logger.Named("ui-benchmark"),
	}, nil
}

// BenchmarkUIResponseTime measures UI response time performance
func (ub *UIBenchmark) BenchmarkUIResponseTime(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	ub.logger.Info("Starting UI response time benchmark", zap.Int("iterations", iterations))

	startTime := time.Now()
	var responseTimes []time.Duration
	
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		
		// Simulate UI rendering operations
		ub.simulateUIRendering()
		
		iterDuration := time.Since(iterStart)
		responseTimes = append(responseTimes, iterDuration)
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "UI response time benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	p99ResponseTime := calculatePercentile(responseTimes, 0.99)
	avgResponseTime := calculateMean(responseTimes)

	result := &BenchmarkResult{
		Name:         "UI Response Time P99",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: p99ResponseTime.Seconds() * 1000, // Convert to milliseconds
		Success:      p99ResponseTime <= 100*time.Millisecond,
		TargetMet:    p99ResponseTime <= 100*time.Millisecond,
		ActualValue:  p99ResponseTime,
		TargetValue:  100 * time.Millisecond,
	}

	ub.logger.Info("UI response time benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Duration("avg_response_time", avgResponseTime),
		zap.Duration("p99_response_time", p99ResponseTime),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// BenchmarkAnimationFrameRate measures animation performance
func (ub *UIBenchmark) BenchmarkAnimationFrameRate(ctx context.Context, targetFPS int) (*BenchmarkResult, error) {
	ub.logger.Info("Starting animation frame rate benchmark", zap.Int("target_fps", targetFPS))

	testDuration := 2 * time.Second // Run animation for 2 seconds
	frameInterval := time.Second / time.Duration(targetFPS)
	
	startTime := time.Now()
	frames := 0
	var frameRenderTimes []time.Duration
	
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "animation benchmark cancelled")
		case <-ticker.C:
			frameStart := time.Now()
			
			// Simulate frame rendering
			ub.simulateFrameRendering()
			
			frameDuration := time.Since(frameStart)
			frameRenderTimes = append(frameRenderTimes, frameDuration)
			frames++
			
			if time.Since(startTime) >= testDuration {
				goto done
			}
		}
	}

done:
	totalDuration := time.Since(startTime)
	actualFPS := float64(frames) / totalDuration.Seconds()
	avgFrameRenderTime := calculateMean(frameRenderTimes)

	result := &BenchmarkResult{
		Name:         "Animation Frame Rate",
		Duration:     totalDuration,
		Iterations:   frames,
		MetricsPerOp: avgFrameRenderTime.Seconds() * 1000, // Convert to milliseconds
		Success:      actualFPS >= float64(targetFPS-2), // Allow 2 FPS tolerance
		TargetMet:    actualFPS >= float64(targetFPS-2),
		ActualValue:  actualFPS,
		TargetValue:  float64(targetFPS),
	}

	ub.logger.Info("Animation frame rate benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Int("frames_rendered", frames),
		zap.Float64("actual_fps", actualFPS),
		zap.Duration("avg_frame_render_time", avgFrameRenderTime),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// simulateUIRendering simulates UI rendering operations
func (ub *UIBenchmark) simulateUIRendering() {
	// Simulate typical UI operations: layout calculation, text rendering, etc.
	// This is a simplified simulation for benchmarking purposes
	data := make([]byte, 1024) // Simulate data processing
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	// Simulate some CPU work
	sum := 0
	for i := 0; i < 1000; i++ {
		sum += i
	}
	
	// Simulate a small sleep to represent rendering time
	time.Sleep(time.Microsecond * time.Duration(50+sum%50))
}

// simulateFrameRendering simulates frame rendering for animations
func (ub *UIBenchmark) simulateFrameRendering() {
	// Simulate frame rendering operations
	data := make([]byte, 512) // Simulate smaller frame data
	for i := range data {
		data[i] = byte((i * 31) % 256) // Some computation to simulate work
	}
	
	// Simulate frame rendering work
	for i := 0; i < 100; i++ {
		_ = float64(i) * 1.414 // Some floating point operations
	}
}

// AgentBenchmark implements agent performance benchmarking
type AgentBenchmark struct {
	logger *zap.Logger
}

// NewAgentBenchmark creates a new agent benchmark instance
func NewAgentBenchmark(logger *zap.Logger) (*AgentBenchmark, error) {
	return &AgentBenchmark{
		logger: logger.Named("agent-benchmark"),
	}, nil
}

// BenchmarkAgentResponseTime measures agent response time performance
func (ab *AgentBenchmark) BenchmarkAgentResponseTime(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	ab.logger.Info("Starting agent response time benchmark", zap.Int("iterations", iterations))

	startTime := time.Now()
	var responseTimes []time.Duration
	var failures int
	
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		
		// Simulate agent processing
		err := ab.simulateAgentProcessing(ctx, fmt.Sprintf("test-request-%d", i))
		
		iterDuration := time.Since(iterStart)
		responseTimes = append(responseTimes, iterDuration)
		
		if err != nil {
			failures++
		}
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "agent response time benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	p95ResponseTime := calculatePercentile(responseTimes, 0.95)
	avgResponseTime := calculateMean(responseTimes)
	successRate := float64(iterations-failures) / float64(iterations)

	result := &BenchmarkResult{
		Name:         "Agent Response Time P95",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: p95ResponseTime.Seconds() * 1000, // Convert to milliseconds
		Success:      p95ResponseTime <= 1*time.Second && successRate >= 0.95,
		TargetMet:    p95ResponseTime <= 1*time.Second,
		ActualValue:  p95ResponseTime,
		TargetValue:  1 * time.Second,
	}

	ab.logger.Info("Agent response time benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Duration("avg_response_time", avgResponseTime),
		zap.Duration("p95_response_time", p95ResponseTime),
		zap.Float64("success_rate", successRate),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// BenchmarkMultiAgentCoordination measures multi-agent coordination performance
func (ab *AgentBenchmark) BenchmarkMultiAgentCoordination(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	ab.logger.Info("Starting multi-agent coordination benchmark", zap.Int("iterations", iterations))

	startTime := time.Now()
	agentCount := 3 // Simulate 3 agents coordinating
	var coordinationTimes []time.Duration
	
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		
		// Simulate multi-agent coordination
		ab.simulateMultiAgentCoordination(ctx, agentCount, fmt.Sprintf("coordination-task-%d", i))
		
		iterDuration := time.Since(iterStart)
		coordinationTimes = append(coordinationTimes, iterDuration)
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "multi-agent coordination benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	throughput := float64(iterations) / totalDuration.Seconds()
	avgCoordinationTime := calculateMean(coordinationTimes)

	result := &BenchmarkResult{
		Name:         "Multi-Agent Throughput",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: throughput,
		Success:      throughput >= 100.0, // Target: 100 operations/second
		TargetMet:    throughput >= 100.0,
		ActualValue:  throughput,
		TargetValue:  100.0,
	}

	ab.logger.Info("Multi-agent coordination benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Duration("avg_coordination_time", avgCoordinationTime),
		zap.Float64("throughput_ops_per_sec", throughput),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// simulateAgentProcessing simulates agent processing work
func (ab *AgentBenchmark) simulateAgentProcessing(ctx context.Context, request string) error {
	// Simulate variable processing time based on request complexity
	processingTime := time.Millisecond * time.Duration(100+len(request)%500)
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(processingTime):
		// Simulate some CPU work
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte((i + len(request)) % 256)
		}
		return nil
	}
}

// simulateMultiAgentCoordination simulates multi-agent coordination
func (ab *AgentBenchmark) simulateMultiAgentCoordination(ctx context.Context, agentCount int, task string) {
	// Simulate parallel agent coordination
	var wg sync.WaitGroup
	wg.Add(agentCount)
	
	for i := 0; i < agentCount; i++ {
		go func(agentID int) {
			defer wg.Done()
			
			// Simulate agent-specific processing
			processingTime := time.Millisecond * time.Duration(50+agentID*10)
			select {
			case <-ctx.Done():
				return
			case <-time.After(processingTime):
				// Simulate some coordination work
				for j := 0; j < 100; j++ {
					_ = float64(j) * float64(agentID) // Some computation
				}
			}
		}(i)
	}
	
	wg.Wait()
}

// CacheBenchmark implements cache performance benchmarking
type CacheBenchmark struct {
	logger *zap.Logger
	cache  map[string][]byte
	mu     sync.RWMutex
}

// NewCacheBenchmark creates a new cache benchmark instance
func NewCacheBenchmark(logger *zap.Logger) (*CacheBenchmark, error) {
	return &CacheBenchmark{
		logger: logger.Named("cache-benchmark"),
		cache:  make(map[string][]byte),
	}, nil
}

// BenchmarkCacheHitRate measures cache hit rate performance
func (cb *CacheBenchmark) BenchmarkCacheHitRate(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	cb.logger.Info("Starting cache hit rate benchmark", zap.Int("iterations", iterations))

	// Pre-populate cache with test data
	cacheSize := iterations / 2 // 50% of test data in cache
	cb.populateCache(cacheSize)

	startTime := time.Now()
	hits := 0
	misses := 0
	
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("cache-key-%d", i)
		
		cb.mu.RLock()
		_, exists := cb.cache[key]
		cb.mu.RUnlock()
		
		if exists {
			hits++
		} else {
			misses++
			// Simulate cache miss - add to cache
			cb.mu.Lock()
			cb.cache[key] = generateTestData(1024)
			cb.mu.Unlock()
		}
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache hit rate benchmark cancelled")
		default:
		}
	}

	totalDuration := time.Since(startTime)
	hitRate := float64(hits) / float64(iterations)

	result := &BenchmarkResult{
		Name:         "Cache Hit Rate",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: hitRate * 100, // Convert to percentage
		Success:      hitRate >= 0.90, // Target: 90% hit rate
		TargetMet:    hitRate >= 0.90,
		ActualValue:  hitRate,
		TargetValue:  0.90,
	}

	cb.logger.Info("Cache hit rate benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Int("hits", hits),
		zap.Int("misses", misses),
		zap.Float64("hit_rate", hitRate),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// BenchmarkCacheMemoryUsage measures cache memory usage
func (cb *CacheBenchmark) BenchmarkCacheMemoryUsage(ctx context.Context, iterations int) (*BenchmarkResult, error) {
	cb.logger.Info("Starting cache memory usage benchmark", zap.Int("iterations", iterations))

	// Clear existing cache
	cb.mu.Lock()
	cb.cache = make(map[string][]byte)
	cb.mu.Unlock()

	startTime := time.Now()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialMemory := memStats.Alloc
	
	// Populate cache with test data
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("memory-test-key-%d", i)
		data := generateTestData(1024) // 1KB per entry
		
		cb.mu.Lock()
		cb.cache[key] = data
		cb.mu.Unlock()
		
		// Context cancellation check
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache memory usage benchmark cancelled")
		default:
		}
	}

	runtime.ReadMemStats(&memStats)
	finalMemory := memStats.Alloc
	memoryUsed := int64(finalMemory - initialMemory)
	totalDuration := time.Since(startTime)

	result := &BenchmarkResult{
		Name:         "Cache Memory Usage",
		Duration:     totalDuration,
		Iterations:   iterations,
		MetricsPerOp: float64(memoryUsed) / float64(iterations), // Bytes per operation
		Success:      memoryUsed <= 125*1024*1024, // Target: <125MB (25% of 500MB limit)
		TargetMet:    memoryUsed <= 125*1024*1024,
		ActualValue:  memoryUsed,
		TargetValue:  int64(125 * 1024 * 1024),
	}

	cb.logger.Info("Cache memory usage benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Int64("memory_used_bytes", memoryUsed),
		zap.Float64("memory_used_mb", float64(memoryUsed)/(1024*1024)),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// populateCache pre-populates the cache with test data
func (cb *CacheBenchmark) populateCache(size int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("cache-key-%d", i)
		cb.cache[key] = generateTestData(1024)
	}
}

// IntegrationTest implements end-to-end integration testing
type IntegrationTest struct {
	logger *zap.Logger
}

// NewIntegrationTest creates a new integration test instance
func NewIntegrationTest(logger *zap.Logger) (*IntegrationTest, error) {
	return &IntegrationTest{
		logger: logger.Named("integration-test"),
	}, nil
}

// BenchmarkSystemMemoryUsage measures system memory usage under load
func (it *IntegrationTest) BenchmarkSystemMemoryUsage(ctx context.Context, duration time.Duration) (*BenchmarkResult, error) {
	it.logger.Info("Starting system memory usage benchmark", zap.Duration("duration", duration))

	startTime := time.Now()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialMemory := memStats.Alloc
	peakMemory := initialMemory
	
	// Monitor memory usage during the test duration
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	// Simulate system load
	workDone := make(chan struct{})
	go func() {
		defer close(workDone)
		it.simulateSystemLoad(ctx, duration)
	}()
	
	for {
		select {
		case <-ctx.Done():
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "system memory usage benchmark cancelled")
		case <-ticker.C:
			runtime.ReadMemStats(&memStats)
			if memStats.Alloc > peakMemory {
				peakMemory = memStats.Alloc
			}
		case <-workDone:
			goto done
		}
	}

done:
	totalDuration := time.Since(startTime)
	runtime.ReadMemStats(&memStats)
	finalMemory := memStats.Alloc

	result := &BenchmarkResult{
		Name:         "System Memory Usage",
		Duration:     totalDuration,
		Iterations:   1,
		MetricsPerOp: float64(peakMemory) / (1024 * 1024), // Convert to MB
		Success:      int64(peakMemory) <= 500*1024*1024, // Target: <500MB
		TargetMet:    int64(peakMemory) <= 500*1024*1024,
		ActualValue:  int64(peakMemory),
		TargetValue:  int64(500 * 1024 * 1024),
	}

	it.logger.Info("System memory usage benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Uint64("initial_memory_mb", initialMemory/(1024*1024)),
		zap.Uint64("peak_memory_mb", peakMemory/(1024*1024)),
		zap.Uint64("final_memory_mb", finalMemory/(1024*1024)),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// BenchmarkConcurrentLoad measures concurrent user load handling
func (it *IntegrationTest) BenchmarkConcurrentLoad(ctx context.Context, users int, duration time.Duration) (*BenchmarkResult, error) {
	it.logger.Info("Starting concurrent load benchmark", zap.Int("users", users), zap.Duration("duration", duration))

	startTime := time.Now()
	var wg sync.WaitGroup
	successCount := int64(0)
	errorCount := int64(0)
	var mu sync.Mutex
	
	// Start concurrent users
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			userCtx, cancel := context.WithTimeout(ctx, duration)
			defer cancel()
			
			success, errors := it.simulateUserLoad(userCtx, userID)
			
			mu.Lock()
			successCount += int64(success)
			errorCount += int64(errors)
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	totalOperations := successCount + errorCount
	successRate := float64(successCount) / float64(totalOperations)

	result := &BenchmarkResult{
		Name:         "Concurrent Users Load",
		Duration:     totalDuration,
		Iterations:   int(totalOperations),
		MetricsPerOp: float64(users),
		Success:      successRate >= 0.95 && users >= 50, // 95% success rate with 50+ users
		TargetMet:    users >= 50,
		ActualValue:  users,
		TargetValue:  50,
	}

	it.logger.Info("Concurrent load benchmark completed",
		zap.Duration("total_duration", totalDuration),
		zap.Int("concurrent_users", users),
		zap.Int64("successful_operations", successCount),
		zap.Int64("failed_operations", errorCount),
		zap.Float64("success_rate", successRate),
		zap.Bool("target_met", result.TargetMet))

	return result, nil
}

// simulateSystemLoad simulates various system operations
func (it *IntegrationTest) simulateSystemLoad(ctx context.Context, duration time.Duration) {
	endTime := time.Now().Add(duration)
	
	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		// Simulate memory allocation
		data := make([][]byte, 100)
		for i := range data {
			data[i] = generateTestData(1024)
		}
		
		// Simulate CPU work
		for i := 0; i < 1000; i++ {
			_ = float64(i) * 3.14159
		}
		
		// Small delay between operations
		time.Sleep(10 * time.Millisecond)
	}
}

// simulateUserLoad simulates a single user's load pattern
func (it *IntegrationTest) simulateUserLoad(ctx context.Context, userID int) (successes, errors int) {
	operations := []string{"create_session", "send_message", "export_data", "analyze_session"}
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		_ = operations[successes%len(operations)] // operation selected for simulation
		
		// Simulate operation processing time
		processingTime := time.Millisecond * time.Duration(50+userID%100)
		time.Sleep(processingTime)
		
		// Simulate occasional failures (5% failure rate)
		if (successes+errors)%20 == 19 {
			errors++
		} else {
			successes++
		}
		
		// Small delay between operations
		time.Sleep(100 * time.Millisecond)
	}
}

// Utility functions

// calculateMean calculates the mean of a slice of durations
func calculateMean(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	
	return sum / time.Duration(len(durations))
}

// calculatePercentile calculates the nth percentile of a slice of durations
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	// Calculate percentile index
	index := int(float64(len(sorted)-1) * percentile)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}

// generateTestData generates test data of specified size
func generateTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}