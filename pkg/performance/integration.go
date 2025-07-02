// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package performance provides integration utilities for performance optimization
// components within the Guild framework.
package performance

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/cache"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/monitoring"
)

// PerformanceManager orchestrates all performance optimization components
type PerformanceManager struct {
	profiler     *PerformanceProfiler
	cache        *cache.IntelligentCache
	memOptimizer *memory.MemoryOptimizer
	monitor      *monitoring.PerformanceMonitor
	config       *PerformanceConfig
	running      bool
}

// PerformanceConfig configures the performance management system
type PerformanceConfig struct {
	// Profiling configuration
	EnableProfiling      bool          `json:"enable_profiling"`
	ProfilingInterval    time.Duration `json:"profiling_interval"`
	
	// Caching configuration
	EnableIntelligentCache bool                  `json:"enable_intelligent_cache"`
	CacheConfig           *cache.CacheConfig     `json:"cache_config"`
	
	// Memory optimization configuration
	EnableMemoryOptimization bool                        `json:"enable_memory_optimization"`
	MemoryConfig            *memory.OptimizerConfig      `json:"memory_config"`
	
	// Monitoring configuration
	EnableMonitoring    bool                           `json:"enable_monitoring"`
	MonitoringConfig    *monitoring.MonitoringConfig   `json:"monitoring_config"`
	
	// Integration settings
	OptimizationInterval time.Duration `json:"optimization_interval"`
	EnableAutoOptimize   bool          `json:"enable_auto_optimize"`
	PerformanceTargets   *PerformanceTargets `json:"performance_targets"`
}

// PerformanceTargets defines performance SLOs
type PerformanceTargets struct {
	MaxP95ResponseTime time.Duration `json:"max_p95_response_time"`
	MaxMemoryUsage     int64         `json:"max_memory_usage"`
	MinCacheHitRate    float64       `json:"min_cache_hit_rate"`
	MaxErrorRate       float64       `json:"max_error_rate"`
}

// DefaultPerformanceConfig returns a default performance configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		EnableProfiling:          true,
		ProfilingInterval:        time.Hour,
		EnableIntelligentCache:   true,
		CacheConfig:             cache.DefaultCacheConfig(),
		EnableMemoryOptimization: true,
		MemoryConfig:            memory.DefaultOptimizerConfig(),
		EnableMonitoring:        true,
		MonitoringConfig:        monitoring.DefaultMonitoringConfig(),
		OptimizationInterval:    time.Minute * 15,
		EnableAutoOptimize:      true,
		PerformanceTargets: &PerformanceTargets{
			MaxP95ResponseTime: time.Millisecond * 100,
			MaxMemoryUsage:     500 * 1024 * 1024, // 500MB
			MinCacheHitRate:    0.90,               // 90%
			MaxErrorRate:       0.01,               // 1%
		},
	}
}

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager(config *PerformanceConfig) (*PerformanceManager, error) {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	pm := &PerformanceManager{
		config: config,
	}

	// Initialize profiler if enabled
	if config.EnableProfiling {
		pm.profiler = NewPerformanceProfiler()
	}

	// Initialize intelligent cache if enabled
	if config.EnableIntelligentCache {
		var err error
		pm.cache, err = cache.NewIntelligentCache(config.CacheConfig)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize intelligent cache").
				WithComponent("performance-manager").
				WithOperation("NewPerformanceManager")
		}
	}

	// Initialize memory optimizer if enabled
	if config.EnableMemoryOptimization {
		pm.memOptimizer = memory.NewMemoryOptimizer(config.MemoryConfig)
	}

	// Initialize performance monitor if enabled
	if config.EnableMonitoring {
		pm.monitor = monitoring.NewPerformanceMonitor(config.MonitoringConfig)
		
		// Set up console alert handler
		consoleHandler := monitoring.NewConsoleAlertHandler()
		pm.monitor.AddHandler(consoleHandler)
	}

	return pm, nil
}

// Start starts all performance optimization components
func (pm *PerformanceManager) Start(ctx context.Context) error {
	if pm.running {
		return gerror.New(gerror.ErrCodeConflict, "performance manager already running", nil).
			WithComponent("performance-manager").
			WithOperation("Start")
	}

	// Start monitoring first
	if pm.monitor != nil {
		if err := pm.monitor.Start(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start performance monitor").
				WithComponent("performance-manager").
				WithOperation("Start")
		}
	}

	// Start auto-optimization if enabled
	if pm.config.EnableAutoOptimize {
		go pm.runAutoOptimization(ctx)
	}

	// Start periodic profiling if enabled
	if pm.profiler != nil {
		go pm.runPeriodicProfiling(ctx)
	}

	pm.running = true
	return nil
}

// Stop stops all performance optimization components
func (pm *PerformanceManager) Stop() error {
	if !pm.running {
		return gerror.New(gerror.ErrCodeConflict, "performance manager not running", nil).
			WithComponent("performance-manager").
			WithOperation("Stop")
	}

	// Stop monitoring
	if pm.monitor != nil {
		if err := pm.monitor.Stop(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop performance monitor").
				WithComponent("performance-manager").
				WithOperation("Stop")
		}
	}

	pm.running = false
	return nil
}

// runAutoOptimization runs automatic optimization at configured intervals
func (pm *PerformanceManager) runAutoOptimization(ctx context.Context) {
	ticker := time.NewTicker(pm.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			// Run optimization cycle
			if err := pm.optimize(ctx); err != nil {
				// Log error but continue
				// In production, this would use proper logging
				continue
			}
		}
	}
}

// runPeriodicProfiling runs profiling at configured intervals
func (pm *PerformanceManager) runPeriodicProfiling(ctx context.Context) {
	ticker := time.NewTicker(pm.config.ProfilingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			// Run profiling session
			if err := pm.profile(ctx); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}

// optimize runs a full optimization cycle
func (pm *PerformanceManager) optimize(ctx context.Context) error {
	// Check if optimization is needed based on current metrics
	if !pm.shouldOptimize() {
		return nil
	}

	// Run memory optimization
	if pm.memOptimizer != nil {
		_, err := pm.memOptimizer.OptimizeMemoryUsage(ctx)
		if err != nil {
			return gerror.Wrap(err, ErrCodeOptimizationFailed, "memory optimization failed").
				WithComponent("performance-manager").
				WithOperation("optimize")
		}
	}

	return nil
}

// profile runs a profiling session
func (pm *PerformanceManager) profile(ctx context.Context) error {
	if pm.profiler == nil {
		return nil
	}

	// Run short profiling session
	duration := time.Second * 10
	_, err := pm.profiler.ProfileApplication(ctx, duration)
	if err != nil {
		return gerror.Wrap(err, ErrCodeProfilingFailed, "profiling session failed").
			WithComponent("performance-manager").
			WithOperation("profile")
	}

	return nil
}

// shouldOptimize determines if optimization is needed based on current metrics
func (pm *PerformanceManager) shouldOptimize() bool {
	if pm.monitor == nil || pm.config.PerformanceTargets == nil {
		return true // Optimize if we can't measure
	}

	metrics := pm.monitor.GetCurrentMetrics()
	targets := pm.config.PerformanceTargets

	// Check response time
	p95ResponseTime := time.Duration(metrics.ResponseTime.Percentile(0.95)) * time.Millisecond
	if p95ResponseTime > targets.MaxP95ResponseTime {
		return true
	}

	// Check memory usage
	memoryUsage := int64(metrics.MemoryUsage.Value() * 1024 * 1024) // Convert MB to bytes
	if memoryUsage > targets.MaxMemoryUsage {
		return true
	}

	// Check cache hit rate
	cacheHitRate := metrics.CacheHitRate.Value()
	if cacheHitRate < targets.MinCacheHitRate {
		return true
	}

	// Check error rate
	errorRate := metrics.ErrorRate.Value()
	if errorRate > targets.MaxErrorRate {
		return true
	}

	return false
}

// GetCache returns the intelligent cache instance
func (pm *PerformanceManager) GetCache() *cache.IntelligentCache {
	return pm.cache
}

// GetMonitor returns the performance monitor instance
func (pm *PerformanceManager) GetMonitor() *monitoring.PerformanceMonitor {
	return pm.monitor
}

// GetProfiler returns the profiler instance
func (pm *PerformanceManager) GetProfiler() *PerformanceProfiler {
	return pm.profiler
}

// GetMemoryOptimizer returns the memory optimizer instance
func (pm *PerformanceManager) GetMemoryOptimizer() *memory.MemoryOptimizer {
	return pm.memOptimizer
}

// InstrumentedOperation wraps an operation with performance monitoring
func (pm *PerformanceManager) InstrumentedOperation(ctx context.Context, name string, operation func(context.Context) error) error {
	start := time.Now()
	
	// Add tracing if monitoring is enabled
	if pm.monitor != nil && pm.monitor.GetTracing() != nil {
		return pm.monitor.GetTracing().TraceOperation(ctx, name, operation)
	}

	// Fallback to basic instrumentation
	err := operation(ctx)
	duration := time.Since(start)

	// Record metrics if monitoring is enabled
	if pm.monitor != nil {
		pm.monitor.RecordOperation(name, duration, err)
	}

	return err
}

// GetPerformanceReport generates a comprehensive performance report
func (pm *PerformanceManager) GetPerformanceReport(ctx context.Context) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp: time.Now(),
		Targets:   pm.config.PerformanceTargets,
	}

	// Get monitoring stats
	if pm.monitor != nil {
		metrics := pm.monitor.GetCurrentMetrics()
		report.CurrentMetrics = &CurrentMetrics{
			P95ResponseTime: time.Duration(metrics.ResponseTime.Percentile(0.95)) * time.Millisecond,
			MemoryUsage:     int64(metrics.MemoryUsage.Value() * 1024 * 1024),
			CacheHitRate:    metrics.CacheHitRate.Value(),
			ErrorRate:       metrics.ErrorRate.Value(),
			Throughput:      metrics.Throughput.Value(),
			GoroutineCount:  int64(metrics.GoroutineCount.Value()),
		}

		// Render dashboard
		report.Dashboard = pm.monitor.RenderDashboard()
	}

	// Get cache stats
	if pm.cache != nil {
		report.CacheStats = pm.cache.GetStats()
	}

	// Get profiler stats
	if pm.profiler != nil {
		report.ProfilerStats = pm.profiler.GetStats()
	}

	// Get memory optimizer stats
	if pm.memOptimizer != nil {
		report.MemoryStats = pm.memOptimizer.GetStats()
	}

	// Calculate overall health score
	report.HealthScore = pm.calculateHealthScore(report.CurrentMetrics)
	
	// Generate recommendations
	report.Recommendations = pm.generateRecommendations(report.CurrentMetrics)

	return report, nil
}

// PerformanceReport contains comprehensive performance information
type PerformanceReport struct {
	Timestamp       time.Time                     `json:"timestamp"`
	Targets         *PerformanceTargets           `json:"targets"`
	CurrentMetrics  *CurrentMetrics               `json:"current_metrics"`
	CacheStats      *cache.CacheStatistics        `json:"cache_stats,omitempty"`
	ProfilerStats   map[string]interface{}        `json:"profiler_stats,omitempty"`
	MemoryStats     map[string]interface{}        `json:"memory_stats,omitempty"`
	HealthScore     float64                       `json:"health_score"`
	Dashboard       string                        `json:"dashboard"`
	Recommendations []string                      `json:"recommendations"`
}

// CurrentMetrics contains current performance metrics
type CurrentMetrics struct {
	P95ResponseTime time.Duration `json:"p95_response_time"`
	MemoryUsage     int64         `json:"memory_usage"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	ErrorRate       float64       `json:"error_rate"`
	Throughput      float64       `json:"throughput"`
	GoroutineCount  int64         `json:"goroutine_count"`
}

// calculateHealthScore calculates an overall health score (0-100)
func (pm *PerformanceManager) calculateHealthScore(metrics *CurrentMetrics) float64 {
	if metrics == nil || pm.config.PerformanceTargets == nil {
		return 50.0 // Neutral score if no data
	}

	targets := pm.config.PerformanceTargets

	// Response time score (0-25 points)
	responseScore := 25.0
	if metrics.P95ResponseTime > targets.MaxP95ResponseTime {
		responseScore = 25.0 * (1.0 - float64(metrics.P95ResponseTime-targets.MaxP95ResponseTime)/float64(targets.MaxP95ResponseTime))
		if responseScore < 0 {
			responseScore = 0
		}
	}

	// Memory usage score (0-25 points)
	memoryScore := 25.0
	if metrics.MemoryUsage > targets.MaxMemoryUsage {
		memoryScore = 25.0 * (1.0 - float64(metrics.MemoryUsage-targets.MaxMemoryUsage)/float64(targets.MaxMemoryUsage))
		if memoryScore < 0 {
			memoryScore = 0
		}
	}

	// Cache hit rate score (0-25 points)
	cacheScore := 25.0
	if metrics.CacheHitRate < targets.MinCacheHitRate {
		cacheScore = 25.0 * (metrics.CacheHitRate / targets.MinCacheHitRate)
	}

	// Error rate score (0-25 points)
	errorScore := 25.0
	if metrics.ErrorRate > targets.MaxErrorRate {
		errorScore = 25.0 * (1.0 - (metrics.ErrorRate-targets.MaxErrorRate)/targets.MaxErrorRate)
		if errorScore < 0 {
			errorScore = 0
		}
	}

	return responseScore + memoryScore + cacheScore + errorScore
}

// generateRecommendations generates performance recommendations
func (pm *PerformanceManager) generateRecommendations(metrics *CurrentMetrics) []string {
	var recommendations []string

	if metrics == nil || pm.config.PerformanceTargets == nil {
		return recommendations
	}

	targets := pm.config.PerformanceTargets

	// Response time recommendations
	if metrics.P95ResponseTime > targets.MaxP95ResponseTime {
		recommendations = append(recommendations, 
			"P95 response time exceeds target - consider optimizing slow operations or adding caching")
	}

	// Memory usage recommendations
	if metrics.MemoryUsage > targets.MaxMemoryUsage {
		recommendations = append(recommendations, 
			"Memory usage exceeds target - run memory optimization or investigate memory leaks")
	}

	// Cache hit rate recommendations
	if metrics.CacheHitRate < targets.MinCacheHitRate {
		recommendations = append(recommendations, 
			"Cache hit rate below target - review cache strategy and warming policies")
	}

	// Error rate recommendations
	if metrics.ErrorRate > targets.MaxErrorRate {
		recommendations = append(recommendations, 
			"Error rate exceeds target - investigate and fix error sources")
	}

	// Goroutine count recommendations
	if metrics.GoroutineCount > 1000 {
		recommendations = append(recommendations, 
			"High goroutine count detected - check for goroutine leaks")
	}

	// Throughput recommendations
	if metrics.Throughput < 100 { // requests per minute
		recommendations = append(recommendations, 
			"Low throughput detected - consider performance optimizations")
	}

	return recommendations
}

// IsRunning returns whether the performance manager is running
func (pm *PerformanceManager) IsRunning() bool {
	return pm.running
}

// GetStats returns performance manager statistics
func (pm *PerformanceManager) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"running":         pm.running,
		"profiling":       pm.config.EnableProfiling,
		"caching":         pm.config.EnableIntelligentCache,
		"memory_opt":      pm.config.EnableMemoryOptimization,
		"monitoring":      pm.config.EnableMonitoring,
		"auto_optimize":   pm.config.EnableAutoOptimize,
	}

	if pm.config.PerformanceTargets != nil {
		stats["max_p95_response_ms"] = pm.config.PerformanceTargets.MaxP95ResponseTime.Milliseconds()
		stats["max_memory_mb"] = pm.config.PerformanceTargets.MaxMemoryUsage / (1024 * 1024)
		stats["min_cache_hit_rate"] = pm.config.PerformanceTargets.MinCacheHitRate
		stats["max_error_rate"] = pm.config.PerformanceTargets.MaxErrorRate
	}

	return stats
}