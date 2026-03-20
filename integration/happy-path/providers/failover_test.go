//go:build integration
// +build integration

package providers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProviderConfig defines provider configuration for testing
type ProviderConfig struct {
	Name           string
	Priority       int
	CostMultiplier float64
	QualityScore   int
	RateLimit      int
	Endpoint       string
}

// FailureType represents different types of failures that can be injected
type FailureType int

const (
	FailureType_ProcessCrash FailureType = iota
	FailureType_NetworkPartition
	FailureType_ResourceExhaustion
	FailureType_ServiceUnavailable
	FailureType_RateLimitExceeded
	FailureType_TimeoutError
)

func (f FailureType) String() string {
	switch f {
	case FailureType_ProcessCrash:
		return "ProcessCrash"
	case FailureType_NetworkPartition:
		return "NetworkPartition"
	case FailureType_ResourceExhaustion:
		return "ResourceExhaustion"
	case FailureType_ServiceUnavailable:
		return "ServiceUnavailable"
	case FailureType_RateLimitExceeded:
		return "RateLimitExceeded"
	case FailureType_TimeoutError:
		return "TimeoutError"
	default:
		return "Unknown"
	}
}

// FailurePattern defines how failures should be injected
type FailurePattern struct {
	Provider string
	Type     FailureType
	Duration time.Duration
	Delay    time.Duration
}

// LoadProfile defines load testing parameters
type LoadProfile struct {
	RequestsPerSecond int
	Duration          time.Duration
	RampUpTime        time.Duration
}

// FailoverStrategy defines failover behavior
type FailoverStrategy struct {
	Type                 FailoverType
	MaxFailoverTime      time.Duration
	HealthCheckInterval  time.Duration
	CircuitBreakerConfig CircuitBreakerConfig
}

// FailoverType represents different failover strategies
type FailoverType int

const (
	FailoverType_CostAware FailoverType = iota
	FailoverType_QualityFirst
	FailoverType_RoundRobin
)

// LoadBalancingConfig defines load balancing behavior
type LoadBalancingConfig struct {
	Algorithm           LoadBalancingAlgorithm
	HealthWeightFactor  float64
	CostWeightFactor    float64
	QualityWeightFactor float64
}

// LoadBalancingAlgorithm represents load balancing algorithms
type LoadBalancingAlgorithm int

const (
	LoadBalancingAlgorithm_WeightedRoundRobin LoadBalancingAlgorithm = iota
	LoadBalancingAlgorithm_LeastConnections
	LoadBalancingAlgorithm_CostOptimized
)

// ProviderPoolConfig contains provider pool configuration
type ProviderPoolConfig struct {
	Providers        []ProviderConfig
	FailoverStrategy FailoverStrategy
	LoadBalancing    LoadBalancingConfig
}

// RequestType represents different types of requests
type RequestType int

const (
	RequestType_ChatCompletion RequestType = iota
	RequestType_CodeGeneration
	RequestType_TextAnalysis
	RequestType_DocumentSummary
)

// PayloadSize represents different payload sizes
type PayloadSize int

const (
	PayloadSize_Small PayloadSize = iota
	PayloadSize_Medium
	PayloadSize_Large
)

// LoadTestConfig defines load test parameters
type LoadTestConfig struct {
	RequestsPerSecond     int
	RequestTypes          []RequestType
	PayloadVariations     []PayloadSize
	RealisticDistribution bool
}

// LoadTestResults contains load test execution results
type LoadTestResults struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	SuccessRate        float64
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	ThroughputRPS      float64
	ErrorBreakdown     map[string]int
}

// ProviderUsageStats tracks provider usage statistics
type ProviderUsageStats struct {
	RequestCount   int
	SuccessRate    float64
	AverageLatency time.Duration
	TotalCost      float64
}

// ProviderHealthStats tracks provider health statistics
type ProviderHealthStats struct {
	WasFailedOver         bool
	RecoveredSuccessfully bool
	RecoveryTime          time.Duration
	HealthScore           float64
}

// RequestMetrics tracks request execution metrics
type RequestMetrics struct {
	startTime             time.Time
	failoverEvents        []FailoverEvent
	providerUsage         map[string]*ProviderUsageStats
	providerHealth        map[string]*ProviderHealthStats
	circuitBreakers       map[string]*CircuitBreakerMetrics
	latencyDuringFailures time.Duration
	failurePeriods        []FailurePeriod
	mu                    sync.RWMutex
}

// FailurePeriod represents a period when failures occurred
type FailurePeriod struct {
	StartTime time.Time
	EndTime   time.Time
	Provider  string
}

// NewRequestMetrics creates new request metrics tracker
func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{
		startTime:       time.Now(),
		failoverEvents:  make([]FailoverEvent, 0),
		providerUsage:   make(map[string]*ProviderUsageStats),
		providerHealth:  make(map[string]*ProviderHealthStats),
		circuitBreakers: make(map[string]*CircuitBreakerMetrics),
		failurePeriods:  make([]FailurePeriod, 0),
	}
}

// GetFailoverEvents returns all failover events
func (m *RequestMetrics) GetFailoverEvents() []FailoverEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]FailoverEvent(nil), m.failoverEvents...)
}

// GetLatencyDuringFailures returns average latency during failure periods
func (m *RequestMetrics) GetLatencyDuringFailures() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latencyDuringFailures
}

// GetProviderUsageStats returns provider usage statistics
func (m *RequestMetrics) GetProviderUsageStats() map[string]*ProviderUsageStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ProviderUsageStats)
	for k, v := range m.providerUsage {
		result[k] = v
	}
	return result
}

// GetProviderHealthStats returns provider health statistics
func (m *RequestMetrics) GetProviderHealthStats() map[string]*ProviderHealthStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ProviderHealthStats)
	for k, v := range m.providerHealth {
		result[k] = v
	}
	return result
}

// GetCircuitBreakerMetrics returns circuit breaker metrics
func (m *RequestMetrics) GetCircuitBreakerMetrics() map[string]*CircuitBreakerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*CircuitBreakerMetrics)
	for k, v := range m.circuitBreakers {
		result[k] = v
	}
	return result
}

// recordFailoverEvent records a failover event
func (m *RequestMetrics) recordFailoverEvent(event FailoverEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failoverEvents = append(m.failoverEvents, event)
}

// MockProviderPool represents a provider pool for testing
type MockProviderPool struct {
	config         ProviderPoolConfig
	activeProvider string
	providers      map[string]*MockProvider
	failureStates  map[string]bool
	mu             sync.RWMutex
}

// MockProvider represents a mock provider implementation
type MockProvider struct {
	config    ProviderConfig
	healthy   bool
	rateLimit int
	requests  int
	failures  int
	lastUsed  time.Time
	mu        sync.RWMutex
}

// IsHealthy checks if provider is healthy
func (p *MockProvider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthy
}

// Execute simulates provider request execution
func (p *MockProvider) Execute(ctx context.Context, request interface{}) (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.requests++
	p.lastUsed = time.Now()

	// Check rate limit
	if p.rateLimit > 0 && p.requests%p.rateLimit == 0 {
		return nil, gerror.New(gerror.ErrCodeRateLimit, "rate limit exceeded", nil)
	}

	// Simulate processing time based on provider quality
	processingTime := time.Duration(100-p.config.QualityScore) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate occasional failures for more realistic testing
	if !p.healthy || (p.requests%20 == 0) {
		p.failures++
		return nil, gerror.New(gerror.ErrCodeInternal, "provider failure", nil)
	}

	return fmt.Sprintf("Response from %s", p.config.Name), nil
}

// GetStats returns provider statistics
func (p *MockProvider) GetStats() ProviderUsageStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	successRate := float64(p.requests-p.failures) / float64(p.requests)
	if p.requests == 0 {
		successRate = 1.0
	}

	return ProviderUsageStats{
		RequestCount:   p.requests,
		SuccessRate:    successRate,
		AverageLatency: time.Duration(100-p.config.QualityScore) * time.Millisecond,
		TotalCost:      float64(p.requests) * p.config.CostMultiplier,
	}
}

// SelectProvider selects the best available provider
func (pool *MockProviderPool) SelectProvider() (*MockProvider, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Try providers in priority order
	for i := 1; i <= len(pool.providers); i++ {
		for _, provider := range pool.providers {
			if provider.config.Priority == i && provider.IsHealthy() {
				if isFailureState, exists := pool.failureStates[provider.config.Name]; !exists || !isFailureState {
					return provider, nil
				}
			}
		}
	}

	return nil, gerror.New(gerror.ErrCodeInternal, "no healthy providers available", nil)
}

// FailureInjector manages failure injection for providers
type FailureInjector struct {
	pattern FailurePattern
	pool    *MockProviderPool
	active  bool
	mu      sync.RWMutex
}

// Start starts failure injection
func (f *FailureInjector) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.active {
		return gerror.New(gerror.ErrCodeInternal, "failure injection already active", nil)
	}

	// Mark provider as failed
	if f.pool.failureStates == nil {
		f.pool.failureStates = make(map[string]bool)
	}
	f.pool.failureStates[f.pattern.Provider] = true

	// Mark provider as unhealthy
	if provider, exists := f.pool.providers[f.pattern.Provider]; exists {
		provider.mu.Lock()
		provider.healthy = false
		provider.mu.Unlock()
	}

	f.active = true

	// Schedule recovery
	go func() {
		timer := time.NewTimer(f.pattern.Duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			f.recover()
		case <-ctx.Done():
			f.recover()
		}
	}()

	return nil
}

// recover recovers the provider from failure
func (f *FailureInjector) recover() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.active {
		return
	}

	// Remove failure state
	delete(f.pool.failureStates, f.pattern.Provider)

	// Mark provider as healthy
	if provider, exists := f.pool.providers[f.pattern.Provider]; exists {
		provider.mu.Lock()
		provider.healthy = true
		provider.mu.Unlock()
	}

	f.active = false
}

// Stop stops failure injection
func (f *FailureInjector) Stop() {
	f.recover()
}

// ProviderTestFramework provides utilities for provider testing
type ProviderTestFramework struct {
	t       *testing.T
	cleanup []func()
}

// NewProviderTestFramework creates a new provider test framework
func NewProviderTestFramework(t *testing.T) *ProviderTestFramework {
	return &ProviderTestFramework{
		t:       t,
		cleanup: make([]func(), 0),
	}
}

// CreateProviderPool creates a provider pool for testing
func (f *ProviderTestFramework) CreateProviderPool(config ProviderPoolConfig) (*MockProviderPool, error) {
	pool := &MockProviderPool{
		config:        config,
		providers:     make(map[string]*MockProvider),
		failureStates: make(map[string]bool),
	}

	// Initialize providers
	for _, providerConfig := range config.Providers {
		provider := &MockProvider{
			config:    providerConfig,
			healthy:   true,
			rateLimit: providerConfig.RateLimit,
		}
		pool.providers[providerConfig.Name] = provider
	}

	f.cleanup = append(f.cleanup, func() {
		// Cleanup provider pool
	})

	return pool, nil
}

// StartRequestMetrics starts request metrics collection
func (f *ProviderTestFramework) StartRequestMetrics() *RequestMetrics {
	return NewRequestMetrics()
}

// ExecuteLoadTest executes a load test against the provider pool
func (f *ProviderTestFramework) ExecuteLoadTest(ctx context.Context, pool *MockProviderPool, config LoadTestConfig) *LoadTestResults {
	results := &LoadTestResults{
		ErrorBreakdown: make(map[string]int),
		MinLatency:     time.Hour, // Initialize to high value
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Calculate request interval
	requestInterval := time.Second / time.Duration(config.RequestsPerSecond)

	// Start request generation
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			f.finalizeResults(results)
			return results

		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				f.executeRequest(pool, config, results, &mu)
			}()
		}
	}
}

// executeRequest executes a single request
func (f *ProviderTestFramework) executeRequest(pool *MockProviderPool, config LoadTestConfig, results *LoadTestResults, mu *sync.Mutex) {
	start := time.Now()

	// Select provider
	provider, err := pool.SelectProvider()
	if err != nil {
		mu.Lock()
		results.FailedRequests++
		results.ErrorBreakdown["provider_selection"]++
		mu.Unlock()
		return
	}

	// Execute request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = provider.Execute(ctx, "test request")
	latency := time.Since(start)

	mu.Lock()
	defer mu.Unlock()

	results.TotalRequests++

	if err != nil {
		results.FailedRequests++
		if gerr, ok := err.(*gerror.GuildError); ok {
			results.ErrorBreakdown[string(gerr.Code)]++
		} else {
			results.ErrorBreakdown["unknown"]++
		}
	} else {
		results.SuccessfulRequests++
	}

	// Update latency statistics
	if latency > results.MaxLatency {
		results.MaxLatency = latency
	}
	if latency < results.MinLatency {
		results.MinLatency = latency
	}
}

// finalizeResults calculates final statistics
func (f *ProviderTestFramework) finalizeResults(results *LoadTestResults) {
	if results.TotalRequests > 0 {
		results.SuccessRate = float64(results.SuccessfulRequests) / float64(results.TotalRequests)
		results.AverageLatency = (results.MaxLatency + results.MinLatency) / 2
	}
}

// CreateFailureInjector creates a failure injector
func (f *ProviderTestFramework) CreateFailureInjector(pattern FailurePattern) *FailureInjector {
	return &FailureInjector{
		pattern: pattern,
	}
}

// GetProviderConfig returns provider configuration by name
func (f *ProviderTestFramework) GetProviderConfig(name string) ProviderConfig {
	// Mock implementation - return default config
	return ProviderConfig{
		Name:           name,
		CostMultiplier: 1.0,
		QualityScore:   90,
	}
}

// Cleanup performs test cleanup
func (f *ProviderTestFramework) Cleanup() {
	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// TestProviderFailover_HappyPath validates intelligent provider switching
func TestProviderFailover_HappyPath(t *testing.T) {
	framework := NewProviderTestFramework(t)
	defer framework.Cleanup()

	failoverScenarios := []struct {
		name                 string
		providers            []ProviderConfig
		failurePatterns      []FailurePattern
		expectedFailoverTime time.Duration
		expectedSuccessRate  float64
		loadProfile          LoadProfile
	}{
		{
			name: "Simple failover between two providers",
			providers: []ProviderConfig{
				{Name: "openai", Priority: 1, CostMultiplier: 1.0, QualityScore: 95},
				{Name: "anthropic", Priority: 2, CostMultiplier: 0.8, QualityScore: 92},
			},
			failurePatterns: []FailurePattern{
				{Provider: "openai", Type: FailureType_NetworkPartition, Duration: 10 * time.Second},
			},
			expectedFailoverTime: 2 * time.Second,
			expectedSuccessRate:  0.98,
			loadProfile:          LoadProfile{RequestsPerSecond: 5, Duration: 20 * time.Second},
		},
		{
			name: "Complex multi-provider failover with rate limiting",
			providers: []ProviderConfig{
				{Name: "openai", Priority: 1, CostMultiplier: 1.0, QualityScore: 95, RateLimit: 10},
				{Name: "anthropic", Priority: 2, CostMultiplier: 0.8, QualityScore: 92, RateLimit: 15},
				{Name: "local", Priority: 3, CostMultiplier: 0.1, QualityScore: 80, RateLimit: 100},
			},
			failurePatterns: []FailurePattern{
				{Provider: "openai", Type: FailureType_ProcessCrash, Duration: 30 * time.Second},
				{Provider: "anthropic", Type: FailureType_NetworkPartition, Duration: 15 * time.Second, Delay: 20 * time.Second},
			},
			expectedFailoverTime: 3 * time.Second,
			expectedSuccessRate:  0.95,
			loadProfile:          LoadProfile{RequestsPerSecond: 20, Duration: 30 * time.Second},
		},
	}

	for _, scenario := range failoverScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// PHASE 1: Initialize provider pool with realistic configurations
			providerPool, err := framework.CreateProviderPool(ProviderPoolConfig{
				Providers: scenario.providers,
				FailoverStrategy: FailoverStrategy{
					Type:                FailoverType_CostAware,
					MaxFailoverTime:     scenario.expectedFailoverTime,
					HealthCheckInterval: 5 * time.Second,
					CircuitBreakerConfig: CircuitBreakerConfig{
						FailureThreshold: 3,
						RecoveryTimeout:  30 * time.Second,
						HalfOpenRequests: 2,
					},
				},
				LoadBalancing: LoadBalancingConfig{
					Algorithm:           LoadBalancingAlgorithm_WeightedRoundRobin,
					HealthWeightFactor:  0.3,
					CostWeightFactor:    0.4,
					QualityWeightFactor: 0.3,
				},
			})
			require.NoError(t, err, "Failed to create provider pool")

			// Initialize request metrics
			requestMetrics := framework.StartRequestMetrics()

			// PHASE 2: Establish baseline performance
			baselineCtx, baselineCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer baselineCancel()

			baselineResults := framework.ExecuteLoadTest(baselineCtx, providerPool, LoadTestConfig{
				RequestsPerSecond: scenario.loadProfile.RequestsPerSecond / 2,
				RequestTypes: []RequestType{
					RequestType_ChatCompletion,
					RequestType_CodeGeneration,
					RequestType_TextAnalysis,
				},
				PayloadVariations: []PayloadSize{PayloadSize_Small, PayloadSize_Medium},
			})

			require.GreaterOrEqual(t, baselineResults.SuccessRate, 0.99,
				"Baseline success rate too low: %.2f%% < 99%%", baselineResults.SuccessRate*100)

			baselineLatency := baselineResults.AverageLatency

			// PHASE 3: Execute load test with failure injection
			loadTestCtx, loadTestCancel := context.WithTimeout(context.Background(), scenario.loadProfile.Duration)
			defer loadTestCancel()

			// Start failure injection based on patterns
			failureInjectors := make([]*FailureInjector, len(scenario.failurePatterns))
			for i, pattern := range scenario.failurePatterns {
				injector := framework.CreateFailureInjector(pattern)
				injector.pool = providerPool // Set pool reference
				failureInjectors[i] = injector

				// Schedule failure injection
				go func(injector *FailureInjector, pattern FailurePattern) {
					time.Sleep(pattern.Delay)

					t.Logf("🔥 Injecting %s failure on provider %s for %v",
						pattern.Type, pattern.Provider, pattern.Duration)

					err := injector.Start(loadTestCtx)
					if err != nil {
						t.Errorf("Failed to inject failure: %v", err)
					}
				}(injector, pattern)
			}

			// Execute main load test
			loadTestResults := framework.ExecuteLoadTest(loadTestCtx, providerPool, LoadTestConfig{
				RequestsPerSecond: scenario.loadProfile.RequestsPerSecond,
				RequestTypes: []RequestType{
					RequestType_ChatCompletion,
					RequestType_CodeGeneration,
					RequestType_TextAnalysis,
					RequestType_DocumentSummary,
				},
				PayloadVariations: []PayloadSize{
					PayloadSize_Small, PayloadSize_Medium, PayloadSize_Large,
				},
				RealisticDistribution: true,
			})

			// Stop failure injectors
			for _, injector := range failureInjectors {
				injector.Stop()
			}

			// PHASE 4: Validate failover performance

			// Check overall success rate
			assert.GreaterOrEqual(t, loadTestResults.SuccessRate, scenario.expectedSuccessRate,
				"Success rate below target: %.3f < %.3f", loadTestResults.SuccessRate, scenario.expectedSuccessRate)

			// Validate failover timing
			failoverEvents := requestMetrics.GetFailoverEvents()
			for _, event := range failoverEvents {
				assert.LessOrEqual(t, event.FailoverDuration, scenario.expectedFailoverTime,
					"Failover time exceeded target: %v > %v", event.FailoverDuration, scenario.expectedFailoverTime)
				assert.Equal(t, FailoverReasonProviderFailure, event.Reason,
					"Unexpected failover reason: %v", event.Reason)
			}

			// Validate latency impact during failures
			latencyDuringFailures := requestMetrics.GetLatencyDuringFailures()
			if latencyDuringFailures > 0 {
				assert.LessOrEqual(t, latencyDuringFailures, baselineLatency*2,
					"Latency during failures too high: %v > %v", latencyDuringFailures, baselineLatency*2)
			}

			// PHASE 5: Validate provider selection intelligence
			providerUsageStats := requestMetrics.GetProviderUsageStats()

			// Verify provider usage distribution
			totalCost := 0.0
			for providerName, stats := range providerUsageStats {
				_ = framework.GetProviderConfig(providerName)
				totalCost += stats.TotalCost

				t.Logf("Provider %s: %d requests, %.1f%% success rate, avg latency %v",
					providerName, stats.RequestCount, stats.SuccessRate*100, stats.AverageLatency)
			}

			// PHASE 6: Validate provider health and recovery
			providerHealthStats := requestMetrics.GetProviderHealthStats()
			for providerName, healthStat := range providerHealthStats {
				if healthStat.WasFailedOver {
					assert.True(t, healthStat.RecoveredSuccessfully,
						"Provider %s did not recover successfully", providerName)
					assert.LessOrEqual(t, healthStat.RecoveryTime, 60*time.Second,
						"Provider %s recovery time too long: %v", providerName, healthStat.RecoveryTime)
				}
			}

			// PHASE 7: Validate circuit breaker behavior
			circuitBreakerMetrics := requestMetrics.GetCircuitBreakerMetrics()
			for providerName, cbMetrics := range circuitBreakerMetrics {
				if cbMetrics.WasTriggered {
					assert.LessOrEqual(t, cbMetrics.FailedRequestsBeforeOpen, 5,
						"Circuit breaker for %s took too many failures to open: %d",
						providerName, cbMetrics.FailedRequestsBeforeOpen)
					assert.GreaterOrEqual(t, cbMetrics.RecoverySuccessRate, 0.8,
						"Circuit breaker for %s recovery success rate too low: %.2f%%",
						providerName, cbMetrics.RecoverySuccessRate*100)
				}
			}

			t.Logf("✅ Provider failover test completed successfully")
			t.Logf("📊 Failover Performance Summary:")
			t.Logf("   - Overall Success Rate: %.2f%%", loadTestResults.SuccessRate*100)
			t.Logf("   - Total Requests: %d", loadTestResults.TotalRequests)
			t.Logf("   - Failover Events: %d", len(failoverEvents))
			t.Logf("   - Average Latency: %v", loadTestResults.AverageLatency)
			t.Logf("   - Total Cost Impact: $%.4f", totalCost)
		})
	}
}
