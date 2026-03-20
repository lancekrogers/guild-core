// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
	"github.com/lancekrogers/guild-core/pkg/registry"
)

// RealProviderIntegrationFramework provides real provider integration testing
type RealProviderIntegrationFramework struct {
	t        TestingT
	registry registry.ComponentRegistry
	manager  *ProviderManager
	selector *IntelligentProviderSelector
	failover *FailoverManager
	cost     *CostOptimizer
	security *SecurityManager
	cleanup  []func()
	mu       sync.RWMutex
}

// TestingT interface for testing compatibility
type TestingT interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
	Helper()
}

// NewRealProviderIntegrationFramework creates a real provider integration framework
func NewRealProviderIntegrationFramework(t TestingT) (*RealProviderIntegrationFramework, error) {
	reg := registry.NewComponentRegistry()

	framework := &RealProviderIntegrationFramework{
		t:        t,
		registry: reg,
		cleanup:  make([]func(), 0),
	}

	// Initialize provider manager
	manager, err := NewProviderManager(reg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider manager")
	}
	framework.manager = manager

	// Initialize intelligent provider selector
	selector, err := NewIntelligentProviderSelector(manager)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider selector")
	}
	framework.selector = selector

	// Initialize failover manager
	failover, err := NewFailoverManager(manager, selector)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create failover manager")
	}
	framework.failover = failover

	// Initialize cost optimizer
	cost, err := NewCostOptimizer(manager)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost optimizer")
	}
	framework.cost = cost

	// Initialize security manager
	security, err := NewSecurityManager(reg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create security manager")
	}
	framework.security = security

	framework.cleanup = append(framework.cleanup, func() {
		framework.manager.Stop()
		framework.failover.Stop()
		framework.cost.Stop()
		framework.security.Stop()
	})

	return framework, nil
}

// ProviderManager manages multiple AI providers
type ProviderManager struct {
	registry     registry.ComponentRegistry
	providers    map[providers.ProviderType]interfaces.AIProvider
	capabilities map[providers.ProviderType]interfaces.ProviderCapabilities
	health       map[providers.ProviderType]*ProviderHealth
	performance  map[providers.ProviderType]*ProviderPerformance
	running      bool
	mu           sync.RWMutex
}

// ProviderHealth tracks provider health metrics
type ProviderHealth struct {
	LastCheck    time.Time
	Healthy      bool
	ResponseTime time.Duration
	ErrorRate    float64
	FailureCount int
	RecoveryTime time.Duration
	CircuitState CircuitState
	mu           sync.RWMutex
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreakerMetrics tracks circuit breaker statistics
type CircuitBreakerMetrics struct {
	WasTriggered             bool
	FailedRequestsBeforeOpen int
	RecoverySuccessRate      float64
	OpenDuration             time.Duration
	HalfOpenAttempts         int
}

// ProviderPerformance tracks provider performance metrics
type ProviderPerformance struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	QualityScore       float64
	CostEfficiency     float64
	LastUpdated        time.Time
	latencies          []time.Duration
	mu                 sync.RWMutex
}

// NewProviderManager creates a new provider manager
func NewProviderManager(registry registry.ComponentRegistry) (*ProviderManager, error) {
	manager := &ProviderManager{
		registry:     registry,
		providers:    make(map[providers.ProviderType]interfaces.AIProvider),
		capabilities: make(map[providers.ProviderType]interfaces.ProviderCapabilities),
		health:       make(map[providers.ProviderType]*ProviderHealth),
		performance:  make(map[providers.ProviderType]*ProviderPerformance),
	}

	// Initialize with mock providers for testing
	err := manager.initializeTestProviders()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

// initializeTestProviders creates mock providers for testing
func (m *ProviderManager) initializeTestProviders() error {
	// Create mock providers for all supported types
	providerTypes := []providers.ProviderType{
		providers.ProviderOpenAI,
		providers.ProviderAnthropic,
		providers.ProviderOllama,
		providers.ProviderDeepSeek,
		providers.ProviderOra,
	}

	for _, providerType := range providerTypes {
		mockProvider := NewMockAIProvider(providerType)
		m.providers[providerType] = mockProvider
		m.capabilities[providerType] = mockProvider.GetCapabilities()

		// Initialize health tracking
		m.health[providerType] = &ProviderHealth{
			LastCheck:    time.Now(),
			Healthy:      true,
			ResponseTime: time.Millisecond * 100,
			ErrorRate:    0.01,
			CircuitState: CircuitClosed,
		}

		// Initialize performance tracking
		m.performance[providerType] = &ProviderPerformance{
			QualityScore:   0.95,
			CostEfficiency: 1.0,
			LastUpdated:    time.Now(),
			latencies:      make([]time.Duration, 0, 1000),
		}
	}

	return nil
}

// Start starts the provider manager
func (m *ProviderManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return gerror.New(gerror.ErrCodeConflict, "provider manager already running", nil)
	}

	// Start health monitoring
	go m.monitorProviderHealth(ctx)

	// Start capability discovery
	go m.discoverCapabilities(ctx)

	m.running = true
	return nil
}

// Stop stops the provider manager
func (m *ProviderManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.running = false
	return nil
}

// GetAvailableProviders returns all available providers
func (m *ProviderManager) GetAvailableProviders() []providers.ProviderType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []providers.ProviderType
	for providerType, health := range m.health {
		if health.Healthy && health.CircuitState != CircuitOpen {
			available = append(available, providerType)
		}
	}
	return available
}

// GetProvider returns a specific provider
func (m *ProviderManager) GetProvider(providerType providers.ProviderType) (interfaces.AIProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[providerType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("provider %s not found", providerType), nil)
	}

	// Check health
	health, exists := m.health[providerType]
	if !exists || !health.Healthy || health.CircuitState == CircuitOpen {
		return nil, gerror.New(gerror.ErrCodeProvider, fmt.Sprintf("provider %s unhealthy", providerType), nil)
	}

	return provider, nil
}

// GetProviderCapabilities returns provider capabilities
func (m *ProviderManager) GetProviderCapabilities(providerType providers.ProviderType) (interfaces.ProviderCapabilities, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	capabilities, exists := m.capabilities[providerType]
	if !exists {
		return interfaces.ProviderCapabilities{}, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("capabilities for %s not found", providerType), nil)
	}

	return capabilities, nil
}

// GetProviderHealth returns provider health
func (m *ProviderManager) GetProviderHealth(providerType providers.ProviderType) (*ProviderHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health, exists := m.health[providerType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("health for %s not found", providerType), nil)
	}

	// Return a copy to avoid race conditions
	health.mu.RLock()
	defer health.mu.RUnlock()

	return &ProviderHealth{
		LastCheck:    health.LastCheck,
		Healthy:      health.Healthy,
		ResponseTime: health.ResponseTime,
		ErrorRate:    health.ErrorRate,
		FailureCount: health.FailureCount,
		RecoveryTime: health.RecoveryTime,
		CircuitState: health.CircuitState,
	}, nil
}

// GetProviderPerformance returns provider performance metrics
func (m *ProviderManager) GetProviderPerformance(providerType providers.ProviderType) (*ProviderPerformance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	perf, exists := m.performance[providerType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("performance for %s not found", providerType), nil)
	}

	// Return a copy to avoid race conditions
	perf.mu.RLock()
	defer perf.mu.RUnlock()

	return &ProviderPerformance{
		TotalRequests:      perf.TotalRequests,
		SuccessfulRequests: perf.SuccessfulRequests,
		FailedRequests:     perf.FailedRequests,
		AverageLatency:     perf.AverageLatency,
		P95Latency:         perf.P95Latency,
		P99Latency:         perf.P99Latency,
		QualityScore:       perf.QualityScore,
		CostEfficiency:     perf.CostEfficiency,
		LastUpdated:        perf.LastUpdated,
	}, nil
}

// RecordProviderMetrics records performance metrics for a provider
func (m *ProviderManager) RecordProviderMetrics(providerType providers.ProviderType, latency time.Duration, success bool, cost float64) {
	m.mu.RLock()
	perf, exists := m.performance[providerType]
	health, healthExists := m.health[providerType]
	m.mu.RUnlock()

	if !exists || !healthExists {
		return
	}

	// Update performance metrics
	perf.mu.Lock()
	perf.TotalRequests++
	if success {
		perf.SuccessfulRequests++
	} else {
		perf.FailedRequests++
	}

	// Update latency tracking
	perf.latencies = append(perf.latencies, latency)
	if len(perf.latencies) > 1000 {
		perf.latencies = perf.latencies[len(perf.latencies)-1000:]
	}

	// Calculate average latency
	if len(perf.latencies) > 0 {
		var total time.Duration
		for _, l := range perf.latencies {
			total += l
		}
		perf.AverageLatency = total / time.Duration(len(perf.latencies))

		// Calculate percentiles
		perf.P95Latency = m.calculatePercentile(perf.latencies, 0.95)
		perf.P99Latency = m.calculatePercentile(perf.latencies, 0.99)
	}

	perf.LastUpdated = time.Now()
	perf.mu.Unlock()

	// Update health metrics
	health.mu.Lock()
	if !success {
		health.FailureCount++
		health.ErrorRate = float64(perf.FailedRequests) / float64(perf.TotalRequests)
	}
	health.ResponseTime = latency
	health.LastCheck = time.Now()

	// Update circuit breaker state
	if health.ErrorRate > 0.1 && health.FailureCount > 5 {
		health.CircuitState = CircuitOpen
		health.Healthy = false
	} else if health.CircuitState == CircuitOpen && health.ErrorRate < 0.05 {
		health.CircuitState = CircuitHalfOpen
	} else if health.CircuitState == CircuitHalfOpen && success {
		health.CircuitState = CircuitClosed
		health.Healthy = true
	}

	health.mu.Unlock()
}

// calculatePercentile calculates the specified percentile from latencies
func (m *ProviderManager) calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation - in production would use proper sorting
	index := int(float64(len(latencies)) * percentile)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

// monitorProviderHealth continuously monitors provider health
func (m *ProviderManager) monitorProviderHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.running {
				return
			}
			m.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks performs health checks on all providers
func (m *ProviderManager) performHealthChecks(ctx context.Context) {
	m.mu.RLock()
	providers := make(map[providers.ProviderType]interfaces.AIProvider)
	for k, v := range m.providers {
		providers[k] = v
	}
	m.mu.RUnlock()

	for providerType, provider := range providers {
		go m.checkProviderHealth(ctx, providerType, provider)
	}
}

// checkProviderHealth checks the health of a specific provider
func (m *ProviderManager) checkProviderHealth(ctx context.Context, providerType providers.ProviderType, provider interfaces.AIProvider) {
	start := time.Now()

	// Simple health check - create a minimal request
	req := interfaces.ChatRequest{
		Model: "health-check",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "health check"},
		},
		MaxTokens: 1,
	}

	_, err := provider.ChatCompletion(ctx, req)
	latency := time.Since(start)

	m.mu.RLock()
	health, exists := m.health[providerType]
	m.mu.RUnlock()

	if !exists {
		return
	}

	health.mu.Lock()
	defer health.mu.Unlock()

	health.LastCheck = time.Now()
	health.ResponseTime = latency

	if err != nil {
		health.FailureCount++
		health.Healthy = false
		if health.CircuitState == CircuitClosed {
			health.CircuitState = CircuitOpen
		}
	} else {
		health.Healthy = true
		if health.CircuitState == CircuitOpen {
			health.CircuitState = CircuitHalfOpen
		} else if health.CircuitState == CircuitHalfOpen {
			health.CircuitState = CircuitClosed
		}
	}
}

// discoverCapabilities continuously discovers provider capabilities
func (m *ProviderManager) discoverCapabilities(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.running {
				return
			}
			m.updateCapabilities()
		}
	}
}

// updateCapabilities updates provider capabilities
func (m *ProviderManager) updateCapabilities() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for providerType, provider := range m.providers {
		capabilities := provider.GetCapabilities()
		m.capabilities[providerType] = capabilities
	}
}

// IntelligentProviderSelector selects optimal providers based on requirements
type IntelligentProviderSelector struct {
	manager          *ProviderManager
	selectionHistory []ProviderSelection
	mu               sync.RWMutex
}

// ProviderSelection tracks provider selection decisions
type ProviderSelection struct {
	Timestamp    time.Time
	Provider     providers.ProviderType
	Reason       SelectionReason
	Requirements TaskRequirements
	Performance  ProviderPerformance
}

// SelectionReason represents why a provider was selected
type SelectionReason int

const (
	SelectionReasonOptimal SelectionReason = iota
	SelectionReasonCost
	SelectionReasonQuality
	SelectionReasonAvailability
	SelectionReasonFailover
)

// TaskRequirements defines what a task needs
type TaskRequirements struct {
	Complexity      TaskComplexity
	MaxLatency      time.Duration
	MaxCost         float64
	QualityRequired float64
	ModelFeatures   []string
}

// TaskComplexity represents task complexity levels
type TaskComplexity int

const (
	TaskComplexitySimple TaskComplexity = iota
	TaskComplexityModerate
	TaskComplexityComplex
	TaskComplexityExpert
)

// NewIntelligentProviderSelector creates a new intelligent provider selector
func NewIntelligentProviderSelector(manager *ProviderManager) (*IntelligentProviderSelector, error) {
	return &IntelligentProviderSelector{
		manager:          manager,
		selectionHistory: make([]ProviderSelection, 0),
	}, nil
}

// SelectProvider selects the optimal provider for given requirements
func (s *IntelligentProviderSelector) SelectProvider(ctx context.Context, requirements TaskRequirements) (providers.ProviderType, SelectionReason, error) {
	start := time.Now()
	defer func() {
		selectionTime := time.Since(start)
		if selectionTime > 100*time.Millisecond {
			// Log slow selection
		}
	}()

	availableProviders := s.manager.GetAvailableProviders()
	if len(availableProviders) == 0 {
		return "", SelectionReasonAvailability, gerror.New(gerror.ErrCodeProvider, "no providers available", nil)
	}

	// Score each provider
	scores := make(map[providers.ProviderType]float64)
	for _, providerType := range availableProviders {
		score := s.scoreProvider(providerType, requirements)
		scores[providerType] = score
	}

	// Select highest scoring provider
	var bestProvider providers.ProviderType
	var bestScore float64
	var reason SelectionReason = SelectionReasonOptimal

	for providerType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestProvider = providerType
		}
	}

	if bestProvider == "" {
		return "", SelectionReasonAvailability, gerror.New(gerror.ErrCodeProvider, "no suitable provider found", nil)
	}

	// Record selection
	s.recordSelection(bestProvider, reason, requirements)

	return bestProvider, reason, nil
}

// scoreProvider calculates a score for a provider based on requirements
func (s *IntelligentProviderSelector) scoreProvider(providerType providers.ProviderType, requirements TaskRequirements) float64 {
	// Get provider metrics
	capabilities, err := s.manager.GetProviderCapabilities(providerType)
	if err != nil {
		return 0
	}

	performance, err := s.manager.GetProviderPerformance(providerType)
	if err != nil {
		return 0
	}

	health, err := s.manager.GetProviderHealth(providerType)
	if err != nil {
		return 0
	}

	// Calculate composite score
	var score float64

	// Quality score (30%)
	qualityScore := performance.QualityScore
	if qualityScore >= requirements.QualityRequired {
		score += 0.3 * qualityScore
	} else {
		score += 0.1 * qualityScore // Penalty for not meeting quality requirements
	}

	// Cost efficiency score (25%)
	costScore := 1.0 / performance.CostEfficiency
	if requirements.MaxCost > 0 {
		if performance.CostEfficiency <= requirements.MaxCost {
			score += 0.25 * costScore
		} else {
			score += 0.1 * costScore // Penalty for exceeding cost requirements
		}
	} else {
		score += 0.25 * costScore
	}

	// Performance score (25%)
	performanceScore := 1.0
	if requirements.MaxLatency > 0 && performance.AverageLatency > requirements.MaxLatency {
		performanceScore = 0.5 // Penalty for high latency
	}
	score += 0.25 * performanceScore

	// Health score (20%)
	healthScore := 1.0
	if !health.Healthy {
		healthScore = 0.1
	} else if health.ErrorRate > 0.05 {
		healthScore = 0.7
	}
	score += 0.2 * healthScore

	// Capability matching bonus
	capabilityBonus := s.calculateCapabilityMatch(capabilities, requirements)
	score += 0.1 * capabilityBonus

	return score
}

// calculateCapabilityMatch calculates how well provider capabilities match requirements
func (s *IntelligentProviderSelector) calculateCapabilityMatch(capabilities interfaces.ProviderCapabilities, requirements TaskRequirements) float64 {
	score := 1.0

	// Check context window requirements
	if requirements.Complexity == TaskComplexityComplex && capabilities.ContextWindow < 8000 {
		score -= 0.3
	} else if requirements.Complexity == TaskComplexityExpert && capabilities.ContextWindow < 32000 {
		score -= 0.3
	}

	// Check feature requirements
	for _, feature := range requirements.ModelFeatures {
		switch feature {
		case "vision":
			if !capabilities.SupportsVision {
				score -= 0.2
			}
		case "tools":
			if !capabilities.SupportsTools {
				score -= 0.2
			}
		case "streaming":
			if !capabilities.SupportsStream {
				score -= 0.1
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// recordSelection records a provider selection decision
func (s *IntelligentProviderSelector) recordSelection(provider providers.ProviderType, reason SelectionReason, requirements TaskRequirements) {
	s.mu.Lock()
	defer s.mu.Unlock()

	selection := ProviderSelection{
		Timestamp:    time.Now(),
		Provider:     provider,
		Reason:       reason,
		Requirements: requirements,
	}

	s.selectionHistory = append(s.selectionHistory, selection)

	// Keep only last 1000 selections
	if len(s.selectionHistory) > 1000 {
		s.selectionHistory = s.selectionHistory[len(s.selectionHistory)-1000:]
	}
}

// GetSelectionHistory returns the provider selection history
func (s *IntelligentProviderSelector) GetSelectionHistory() []ProviderSelection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]ProviderSelection, len(s.selectionHistory))
	copy(history, s.selectionHistory)
	return history
}

// GetSelectionAccuracy calculates selection accuracy metrics
func (s *IntelligentProviderSelector) GetSelectionAccuracy() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.selectionHistory) < 10 {
		return 1.0 // Assume perfect for small samples
	}

	// Calculate how often we selected optimal vs suboptimal providers
	optimalSelections := 0
	for _, selection := range s.selectionHistory {
		if selection.Reason == SelectionReasonOptimal {
			optimalSelections++
		}
	}

	return float64(optimalSelections) / float64(len(s.selectionHistory))
}

// Cleanup performs cleanup
func (f *RealProviderIntegrationFramework) Cleanup() {
	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// GetManager returns the provider manager
func (f *RealProviderIntegrationFramework) GetManager() *ProviderManager {
	return f.manager
}

// GetSelector returns the provider selector
func (f *RealProviderIntegrationFramework) GetSelector() *IntelligentProviderSelector {
	return f.selector
}

// GetFailover returns the failover manager
func (f *RealProviderIntegrationFramework) GetFailover() *FailoverManager {
	return f.failover
}

// GetCostOptimizer returns the cost optimizer
func (f *RealProviderIntegrationFramework) GetCostOptimizer() *CostOptimizer {
	return f.cost
}

// GetSecurityManager returns the security manager
func (f *RealProviderIntegrationFramework) GetSecurityManager() *SecurityManager {
	return f.security
}
