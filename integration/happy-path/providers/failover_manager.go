// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
)

// FailoverManager manages automatic failover between providers
type FailoverManager struct {
	manager           *ProviderManager
	selector          *IntelligentProviderSelector
	circuitBreakers   map[providers.ProviderType]*CircuitBreaker
	failoverHistory   []FailoverEvent
	config            FailoverConfig
	running           bool
	mu                sync.RWMutex
}

// FailoverConfig configures failover behavior
type FailoverConfig struct {
	MaxFailoverTime      time.Duration
	HealthCheckInterval  time.Duration
	CircuitBreakerConfig CircuitBreakerConfig
	RetryConfig          RetryConfig
	QualityThreshold     float64
	LatencyThreshold     time.Duration
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold   int
	RecoveryTimeout    time.Duration
	HalfOpenRequests   int
	SuccessThreshold   int
	ErrorRateThreshold float64
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffMultiplier float64
	JitterEnabled   bool
}

// CircuitBreaker implements circuit breaker pattern for providers
type CircuitBreaker struct {
	provider      providers.ProviderType
	state         CircuitState
	failures      int
	requests      int
	successes     int
	lastFailure   time.Time
	lastStateChange time.Time
	config        CircuitBreakerConfig
	mu            sync.RWMutex
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID               string
	Timestamp        time.Time
	FromProvider     providers.ProviderType
	ToProvider       providers.ProviderType
	Reason           FailoverReason
	FailoverDuration time.Duration
	Success          bool
	QualityImpact    float64
	Context          map[string]interface{}
}

// FailoverReason represents reasons for failover
type FailoverReason int

const (
	FailoverReasonProviderFailure FailoverReason = iota
	FailoverReasonRateLimitExceeded
	FailoverReasonHealthCheck
	FailoverReasonCostOptimization
	FailoverReasonQualityDegradation
	FailoverReasonLatencyThreshold
	FailoverReasonCircuitBreaker
)

func (r FailoverReason) String() string {
	switch r {
	case FailoverReasonProviderFailure:
		return "ProviderFailure"
	case FailoverReasonRateLimitExceeded:
		return "RateLimitExceeded"
	case FailoverReasonHealthCheck:
		return "HealthCheck"
	case FailoverReasonCostOptimization:
		return "CostOptimization"
	case FailoverReasonQualityDegradation:
		return "QualityDegradation"
	case FailoverReasonLatencyThreshold:
		return "LatencyThreshold"
	case FailoverReasonCircuitBreaker:
		return "CircuitBreaker"
	default:
		return "Unknown"
	}
}

// RequestContext contains context for failover decisions
type RequestContext struct {
	RequestID    string
	StartTime    time.Time
	Requirements TaskRequirements
	MaxRetries   int
	Timeout      time.Duration
}

// FailoverResult contains the result of a failover attempt
type FailoverResult struct {
	Success          bool
	FinalProvider    providers.ProviderType
	TotalDuration    time.Duration
	FailoverCount    int
	FailoverEvents   []FailoverEvent
	QualityImpact    float64
	Response         interface{}
	Error            error
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(manager *ProviderManager, selector *IntelligentProviderSelector) (*FailoverManager, error) {
	config := FailoverConfig{
		MaxFailoverTime:     2 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold:   5,
			RecoveryTimeout:    30 * time.Second,
			HalfOpenRequests:   3,
			SuccessThreshold:   2,
			ErrorRateThreshold: 0.5,
		},
		RetryConfig: RetryConfig{
			MaxRetries:        3,
			InitialDelay:      100 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			BackoffMultiplier: 2.0,
			JitterEnabled:     true,
		},
		QualityThreshold: 0.8,
		LatencyThreshold: 10 * time.Second,
	}

	fm := &FailoverManager{
		manager:         manager,
		selector:        selector,
		circuitBreakers: make(map[providers.ProviderType]*CircuitBreaker),
		failoverHistory: make([]FailoverEvent, 0),
		config:          config,
	}

	// Initialize circuit breakers for all providers
	availableProviders := manager.GetAvailableProviders()
	for _, provider := range availableProviders {
		fm.circuitBreakers[provider] = NewCircuitBreaker(provider, config.CircuitBreakerConfig)
	}

	return fm, nil
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(provider providers.ProviderType, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		provider:        provider,
		state:           CircuitClosed,
		config:          config,
		lastStateChange: time.Now(),
	}
}

// Start starts the failover manager
func (f *FailoverManager) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return gerror.New(gerror.ErrCodeConflict, "failover manager already running", nil)
	}

	// Start circuit breaker monitoring
	go f.monitorCircuitBreakers(ctx)

	// Start provider health monitoring
	go f.monitorProviderHealth(ctx)

	f.running = true
	return nil
}

// Stop stops the failover manager
func (f *FailoverManager) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.running = false
	return nil
}

// ExecuteWithFailover executes a request with automatic failover
func (f *FailoverManager) ExecuteWithFailover(ctx context.Context, req interfaces.ChatRequest, reqCtx RequestContext) (*FailoverResult, error) {
	start := time.Now()
	result := &FailoverResult{
		FailoverEvents: make([]FailoverEvent, 0),
	}

	// Select initial provider
	provider, reason, err := f.selector.SelectProvider(ctx, reqCtx.Requirements)
	if err != nil {
		result.Error = err
		return result, err
	}

	currentProvider := provider
	attempts := 0
	maxAttempts := f.config.RetryConfig.MaxRetries + 1

	for attempts < maxAttempts {
		attempts++
		
		// Check circuit breaker state
		circuitBreaker := f.getCircuitBreaker(currentProvider)
		if !circuitBreaker.CanExecute() {
			// Circuit breaker is open - attempt failover
			failoverProvider, failoverReason, failoverErr := f.attemptFailover(ctx, currentProvider, FailoverReasonCircuitBreaker, reqCtx.Requirements)
			if failoverErr != nil {
				result.Error = failoverErr
				break
			}

			// Record failover event
			event := f.recordFailoverEvent(currentProvider, failoverProvider, failoverReason, time.Since(start))
			result.FailoverEvents = append(result.FailoverEvents, event)
			result.FailoverCount++
			
			currentProvider = failoverProvider
		}

		// Execute request
		response, execErr := f.executeRequest(ctx, currentProvider, req)
		
		// Record execution in circuit breaker
		circuitBreaker.RecordExecution(execErr == nil)

		if execErr == nil {
			// Success
			result.Success = true
			result.FinalProvider = currentProvider
			result.Response = response
			result.TotalDuration = time.Since(start)
			
			// Calculate quality impact if we failed over
			if result.FailoverCount > 0 {
				result.QualityImpact = f.calculateQualityImpact(provider, currentProvider)
			}
			
			return result, nil
		}

		// Request failed - check if we should failover
		if f.shouldFailover(execErr, currentProvider) && attempts < maxAttempts {
			failoverReason := f.determineFailoverReason(execErr)
			failoverProvider, failoverReasonActual, failoverErr := f.attemptFailover(ctx, currentProvider, failoverReason, reqCtx.Requirements)
			
			if failoverErr == nil {
				// Record failover event
				event := f.recordFailoverEvent(currentProvider, failoverProvider, failoverReasonActual, time.Since(start))
				result.FailoverEvents = append(result.FailoverEvents, event)
				result.FailoverCount++
				
				currentProvider = failoverProvider
				
				// Apply retry delay
				if attempts < maxAttempts {
					delay := f.calculateRetryDelay(attempts)
					select {
					case <-time.After(delay):
					case <-ctx.Done():
						result.Error = ctx.Err()
						return result, ctx.Err()
					}
				}
				continue
			}
		}

		// No more failover options or max attempts reached
		result.Error = execErr
		break
	}

	result.TotalDuration = time.Since(start)
	result.FinalProvider = currentProvider
	
	return result, result.Error
}

// executeRequest executes a request against a specific provider
func (f *FailoverManager) executeRequest(ctx context.Context, provider providers.ProviderType, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	providerInstance, err := f.manager.GetProvider(provider)
	if err != nil {
		return nil, err
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, f.config.LatencyThreshold)
	defer cancel()

	start := time.Now()
	response, err := providerInstance.ChatCompletion(execCtx, req)
	latency := time.Since(start)

	// Record metrics
	if err == nil {
		f.manager.RecordProviderMetrics(provider, latency, true, 0.0) // Cost would be calculated separately
	} else {
		f.manager.RecordProviderMetrics(provider, latency, false, 0.0)
	}

	return response, err
}

// shouldFailover determines if we should attempt failover for a given error
func (f *FailoverManager) shouldFailover(err error, provider providers.ProviderType) bool {
	if err == nil {
		return false
	}

	// Check if error is retryable
	if providerErr, ok := err.(*interfaces.ProviderError); ok {
		return providerErr.Retryable
	}

	// Check for specific error types that warrant failover
	if guildErr, ok := err.(*gerror.GuildError); ok {
		switch guildErr.Code {
		case gerror.ErrCodeInternal:
		case gerror.ErrCodeTimeout:
		case gerror.ErrCodeRateLimit:
		case gerror.ErrCodeInternal:
			return true
		default:
			return false
		}
	}

	return true // Default to allowing failover for unknown errors
}

// determineFailoverReason determines the reason for failover based on error
func (f *FailoverManager) determineFailoverReason(err error) FailoverReason {
	if providerErr, ok := err.(*interfaces.ProviderError); ok {
		switch providerErr.Type {
		case interfaces.ErrorTypeRateLimit:
			return FailoverReasonRateLimitExceeded
		case interfaces.ErrorTypeServer:
			return FailoverReasonProviderFailure
		case interfaces.ErrorTypeAuth:
			return FailoverReasonProviderFailure
		default:
			return FailoverReasonProviderFailure
		}
	}

	if guildErr, ok := err.(*gerror.GuildError); ok {
		switch guildErr.Code {
		case gerror.ErrCodeTimeout:
			return FailoverReasonLatencyThreshold
		case gerror.ErrCodeRateLimit:
			return FailoverReasonRateLimitExceeded
		default:
			return FailoverReasonProviderFailure
		}
	}

	return FailoverReasonProviderFailure
}

// attemptFailover attempts to failover to an alternative provider
func (f *FailoverManager) attemptFailover(ctx context.Context, currentProvider providers.ProviderType, reason FailoverReason, requirements TaskRequirements) (providers.ProviderType, FailoverReason, error) {
	// Get available providers excluding current one
	availableProviders := f.manager.GetAvailableProviders()
	var alternatives []providers.ProviderType
	
	for _, provider := range availableProviders {
		if provider != currentProvider {
			circuitBreaker := f.getCircuitBreaker(provider)
			if circuitBreaker.CanExecute() {
				alternatives = append(alternatives, provider)
			}
		}
	}

	if len(alternatives) == 0 {
		return "", reason, gerror.New(gerror.ErrCodeInternal, "no alternative providers available", nil)
	}

	// Select best alternative provider
	bestProvider := ""
	bestScore := 0.0

	for _, provider := range alternatives {
		score := f.selector.scoreProvider(provider, requirements)
		if score > bestScore {
			bestScore = score
			bestProvider = string(provider)
		}
	}

	if bestProvider == "" {
		return "", reason, gerror.New(gerror.ErrCodeInternal, "no suitable alternative provider found", nil)
	}

	return providers.ProviderType(bestProvider), reason, nil
}

// recordFailoverEvent records a failover event
func (f *FailoverManager) recordFailoverEvent(fromProvider, toProvider providers.ProviderType, reason FailoverReason, duration time.Duration) FailoverEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	event := FailoverEvent{
		ID:               fmt.Sprintf("failover-%d", time.Now().UnixNano()),
		Timestamp:        time.Now(),
		FromProvider:     fromProvider,
		ToProvider:       toProvider,
		Reason:           reason,
		FailoverDuration: duration,
		Success:          true,
		Context:          make(map[string]interface{}),
	}

	f.failoverHistory = append(f.failoverHistory, event)
	
	// Keep only last 1000 events
	if len(f.failoverHistory) > 1000 {
		f.failoverHistory = f.failoverHistory[len(f.failoverHistory)-1000:]
	}

	return event
}

// calculateQualityImpact calculates the quality impact of failover
func (f *FailoverManager) calculateQualityImpact(originalProvider, currentProvider providers.ProviderType) float64 {
	originalPerf, err := f.manager.GetProviderPerformance(originalProvider)
	if err != nil {
		return 0.0
	}

	currentPerf, err := f.manager.GetProviderPerformance(currentProvider)
	if err != nil {
		return 0.0
	}

	return originalPerf.QualityScore - currentPerf.QualityScore
}

// calculateRetryDelay calculates retry delay with exponential backoff
func (f *FailoverManager) calculateRetryDelay(attempt int) time.Duration {
	delay := f.config.RetryConfig.InitialDelay
	
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * f.config.RetryConfig.BackoffMultiplier)
		if delay > f.config.RetryConfig.MaxDelay {
			delay = f.config.RetryConfig.MaxDelay
			break
		}
	}

	// Add jitter if enabled
	if f.config.RetryConfig.JitterEnabled {
		jitter := time.Duration(float64(delay) * 0.1) // 10% jitter
		delay += time.Duration(float64(jitter) * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
	}

	return delay
}

// getCircuitBreaker gets or creates a circuit breaker for a provider
func (f *FailoverManager) getCircuitBreaker(provider providers.ProviderType) *CircuitBreaker {
	f.mu.RLock()
	cb, exists := f.circuitBreakers[provider]
	f.mu.RUnlock()

	if !exists {
		f.mu.Lock()
		// Double-check after acquiring write lock
		if cb, exists = f.circuitBreakers[provider]; !exists {
			cb = NewCircuitBreaker(provider, f.config.CircuitBreakerConfig)
			f.circuitBreakers[provider] = cb
		}
		f.mu.Unlock()
	}

	return cb
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should move to half-open
		if time.Since(cb.lastStateChange) >= cb.config.RecoveryTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			// Double-check after acquiring write lock
			if cb.state == CircuitOpen && time.Since(cb.lastStateChange) >= cb.config.RecoveryTimeout {
				cb.state = CircuitHalfOpen
				cb.successes = 0
				cb.lastStateChange = time.Now()
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == CircuitHalfOpen
		}
		return false
	case CircuitHalfOpen:
		return cb.requests < cb.config.HalfOpenRequests
	default:
		return false
	}
}

// RecordExecution records the result of an execution
func (cb *CircuitBreaker) RecordExecution(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.requests++

	if success {
		cb.successes++
		
		if cb.state == CircuitHalfOpen && cb.successes >= cb.config.SuccessThreshold {
			// Move to closed state
			cb.state = CircuitClosed
			cb.failures = 0
			cb.requests = 0
			cb.successes = 0
			cb.lastStateChange = time.Now()
		}
	} else {
		cb.failures++
		cb.lastFailure = time.Now()

		// Check if we should move to open state
		if cb.state == CircuitClosed || cb.state == CircuitHalfOpen {
			errorRate := float64(cb.failures) / float64(cb.requests)
			if cb.failures >= cb.config.FailureThreshold || errorRate >= cb.config.ErrorRateThreshold {
				cb.state = CircuitOpen
				cb.lastStateChange = time.Now()
			}
		}
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	recoverySuccessRate := 0.0
	if cb.requests > 0 {
		recoverySuccessRate = float64(cb.successes) / float64(cb.requests)
	}

	return CircuitBreakerMetrics{
		WasTriggered:             cb.state != CircuitClosed,
		FailedRequestsBeforeOpen: cb.failures,
		RecoverySuccessRate:      recoverySuccessRate,
		OpenDuration:             time.Since(cb.lastStateChange),
		HalfOpenAttempts:         cb.requests,
	}
}

// monitorCircuitBreakers monitors circuit breaker states
func (f *FailoverManager) monitorCircuitBreakers(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !f.running {
				return
			}
			f.checkCircuitBreakerHealth()
		}
	}
}

// checkCircuitBreakerHealth checks circuit breaker health
func (f *FailoverManager) checkCircuitBreakerHealth() {
	f.mu.RLock()
	circuitBreakers := make(map[providers.ProviderType]*CircuitBreaker)
	for k, v := range f.circuitBreakers {
		circuitBreakers[k] = v
	}
	f.mu.RUnlock()

	for provider, cb := range circuitBreakers {
		state := cb.GetState()
		
		// If circuit has been open for too long, try to reset it
		if state == CircuitOpen {
			cb.mu.RLock()
			openDuration := time.Since(cb.lastStateChange)
			cb.mu.RUnlock()
			
			if openDuration > f.config.CircuitBreakerConfig.RecoveryTimeout*2 {
				// Force circuit to half-open for health check
				cb.mu.Lock()
				cb.state = CircuitHalfOpen
				cb.requests = 0
				cb.successes = 0
				cb.lastStateChange = time.Now()
				cb.mu.Unlock()
			}
		}
	}
}

// monitorProviderHealth monitors provider health for failover decisions
func (f *FailoverManager) monitorProviderHealth(ctx context.Context) {
	ticker := time.NewTicker(f.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !f.running {
				return
			}
			f.checkProviderHealthForFailover()
		}
	}
}

// checkProviderHealthForFailover checks provider health and triggers failover if needed
func (f *FailoverManager) checkProviderHealthForFailover() {
	availableProviders := f.manager.GetAvailableProviders()
	
	for _, provider := range availableProviders {
		health, err := f.manager.GetProviderHealth(provider)
		if err != nil {
			continue
		}

		performance, err := f.manager.GetProviderPerformance(provider)
		if err != nil {
			continue
		}

		// Check for quality degradation
		if performance.QualityScore < f.config.QualityThreshold {
			// Provider quality has degraded
			cb := f.getCircuitBreaker(provider)
			cb.RecordExecution(false)
		}

		// Check for latency threshold violations
		if performance.AverageLatency > f.config.LatencyThreshold {
			// Provider latency is too high
			cb := f.getCircuitBreaker(provider)
			cb.RecordExecution(false)
		}

		// Check for high error rates
		if health.ErrorRate > f.config.CircuitBreakerConfig.ErrorRateThreshold {
			// Provider error rate is too high
			cb := f.getCircuitBreaker(provider)
			cb.RecordExecution(false)
		}
	}
}

// GetFailoverHistory returns failover history
func (f *FailoverManager) GetFailoverHistory() []FailoverEvent {
	f.mu.RLock()
	defer f.mu.RUnlock()

	history := make([]FailoverEvent, len(f.failoverHistory))
	copy(history, f.failoverHistory)
	return history
}

// GetCircuitBreakerMetrics returns circuit breaker metrics for all providers
func (f *FailoverManager) GetCircuitBreakerMetrics() map[providers.ProviderType]CircuitBreakerMetrics {
	f.mu.RLock()
	defer f.mu.RUnlock()

	metrics := make(map[providers.ProviderType]CircuitBreakerMetrics)
	for provider, cb := range f.circuitBreakers {
		metrics[provider] = cb.GetMetrics()
	}
	return metrics
}

// GetFailoverStats returns failover statistics
func (f *FailoverManager) GetFailoverStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	totalFailovers := len(f.failoverHistory)
	successfulFailovers := 0
	failoversByReason := make(map[string]int)

	for _, event := range f.failoverHistory {
		if event.Success {
			successfulFailovers++
		}
		failoversByReason[event.Reason.String()]++
	}

	stats := map[string]interface{}{
		"total_failovers":      totalFailovers,
		"successful_failovers": successfulFailovers,
		"success_rate":         0.0,
		"failovers_by_reason":  failoversByReason,
		"circuit_breaker_states": make(map[string]string),
	}

	if totalFailovers > 0 {
		stats["success_rate"] = float64(successfulFailovers) / float64(totalFailovers)
	}

	// Add circuit breaker states
	cbStates := stats["circuit_breaker_states"].(map[string]string)
	for provider, cb := range f.circuitBreakers {
		state := cb.GetState()
		switch state {
		case CircuitClosed:
			cbStates[string(provider)] = "closed"
		case CircuitOpen:
			cbStates[string(provider)] = "open"
		case CircuitHalfOpen:
			cbStates[string(provider)] = "half-open"
		}
	}

	return stats
}