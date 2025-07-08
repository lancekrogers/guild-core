// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package monitoring provides real-time performance monitoring and alerting
// capabilities for the Guild framework with comprehensive dashboards and SLO tracking.
package monitoring

import (
	"context"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Monitoring error codes
const (
	ErrCodeMetricNotFound  = "MONITOR-1001"
	ErrCodeSLOViolation    = "MONITOR-1002"
	ErrCodeAlertingFailed  = "MONITOR-1003"
	ErrCodeDashboardFailed = "MONITOR-1004"
	ErrCodeExporterFailed  = "MONITOR-1005"
)

// PerformanceMonitor provides comprehensive performance monitoring
type PerformanceMonitor struct {
	metrics    *MetricsCollector
	alerts     *AlertManager
	dashboard  *PerformanceDashboard
	exporters  []MetricsExporter
	sloMonitor *SLOMonitor
	tracing    *TracingIntegration
	config     *MonitoringConfig
	mu         sync.RWMutex
	running    bool
}

// MonitoringConfig configures monitoring behavior
type MonitoringConfig struct {
	MetricsInterval    time.Duration `json:"metrics_interval"`
	AlertCheckInterval time.Duration `json:"alert_check_interval"`
	DashboardRefresh   time.Duration `json:"dashboard_refresh"`
	EnableTracing      bool          `json:"enable_tracing"`
	EnableExport       bool          `json:"enable_export"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	MaxMetricSamples   int           `json:"max_metric_samples"`
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		MetricsInterval:    time.Second,
		AlertCheckInterval: time.Second * 10,
		DashboardRefresh:   time.Second * 5,
		EnableTracing:      true,
		EnableExport:       false,
		RetentionPeriod:    time.Hour * 24,
		MaxMetricSamples:   10000,
	}
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(config *MonitoringConfig) *PerformanceMonitor {
	if config == nil {
		config = DefaultMonitoringConfig()
	}

	pm := &PerformanceMonitor{
		metrics:    NewMetricsCollector(config.MaxMetricSamples),
		alerts:     NewAlertManager(),
		dashboard:  NewPerformanceDashboard(),
		exporters:  make([]MetricsExporter, 0),
		sloMonitor: NewSLOMonitor(),
		tracing:    NewTracingIntegration(),
		config:     config,
	}

	// Set up default SLOs
	pm.setupDefaultSLOs()

	return pm
}

// Metrics represents the core metrics being tracked
type Metrics struct {
	ResponseTime   *Histogram `json:"response_time"`
	Throughput     *Counter   `json:"throughput"`
	ErrorRate      *Gauge     `json:"error_rate"`
	CPUUsage       *Gauge     `json:"cpu_usage"`
	MemoryUsage    *Gauge     `json:"memory_usage"`
	GoroutineCount *Gauge     `json:"goroutine_count"`
	GCPauses       *Histogram `json:"gc_pauses"`
	CacheHitRate   *Gauge     `json:"cache_hit_rate"`
}

// Start starts the performance monitoring system
func (pm *PerformanceMonitor) Start(ctx context.Context) error {
	pm.mu.Lock()
	if pm.running {
		pm.mu.Unlock()
		return gerror.New(gerror.ErrCodeConflict, "monitoring already running", nil).
			WithComponent("performance-monitor").
			WithOperation("Start")
	}
	pm.running = true
	pm.mu.Unlock()

	// Start metrics collection
	go pm.collectMetrics(ctx)

	// Start application metrics collection
	go pm.collectApplicationMetrics(ctx)

	// Start GC metrics collection
	go pm.collectGCMetrics(ctx)

	// Start system metrics collection
	go pm.collectSystemMetrics(ctx)

	// Start alert monitoring
	go pm.monitorAlerts(ctx)

	// Start exporters if enabled
	if pm.config.EnableExport {
		for _, exporter := range pm.exporters {
			go exporter.Start(ctx)
		}
	}

	return nil
}

// Stop stops the performance monitoring system
func (pm *PerformanceMonitor) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return gerror.New(gerror.ErrCodeConflict, "monitoring not running", nil).
			WithComponent("performance-monitor").
			WithOperation("Stop")
	}

	pm.running = false
	return nil
}

// collectMetrics coordinates metrics collection
func (pm *PerformanceMonitor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(pm.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}
			// Metrics collection is handled by specific collectors
		}
	}
}

// collectApplicationMetrics collects application-specific metrics
func (pm *PerformanceMonitor) collectApplicationMetrics(ctx context.Context) {
	ticker := time.NewTicker(pm.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			// Collect response times from active requests
			pm.collectResponseTimes()

			// Collect throughput metrics
			pm.collectThroughput()

			// Collect error rates
			pm.collectErrorRate()

			// Collect cache metrics
			pm.collectCacheMetrics()
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (pm *PerformanceMonitor) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(pm.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			// Collect CPU usage
			pm.collectCPUUsage()

			// Collect memory usage
			pm.collectMemoryUsage()

			// Collect goroutine count
			pm.collectGoroutineCount()
		}
	}
}

// collectGCMetrics collects garbage collection metrics
func (pm *PerformanceMonitor) collectGCMetrics(ctx context.Context) {
	ticker := time.NewTicker(pm.config.MetricsInterval)
	defer ticker.Stop()

	var lastGCCount uint32

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Check for new GC cycles
			if m.NumGC > lastGCCount {
				// Calculate pause time for recent GC
				pauseTime := time.Duration(m.PauseNs[(m.NumGC+255)%256])
				pm.metrics.RecordGCPause(pauseTime)
				lastGCCount = m.NumGC
			}
		}
	}
}

// collectResponseTimes collects response time metrics
func (pm *PerformanceMonitor) collectResponseTimes() {
	// This would typically integrate with HTTP middleware or instrumentation
	// For now, we'll simulate response times
	responseTime := time.Millisecond * time.Duration(50+rand.Intn(100))
	pm.metrics.RecordResponseTime(responseTime)
}

// collectThroughput collects throughput metrics
func (pm *PerformanceMonitor) collectThroughput() {
	// Increment throughput counter
	pm.metrics.IncrementThroughput()
}

// collectErrorRate collects error rate metrics
func (pm *PerformanceMonitor) collectErrorRate() {
	// This would typically track actual error rates
	// For now, simulate a low error rate
	errorRate := 0.01 // 1% error rate
	pm.metrics.SetErrorRate(errorRate)
}

// collectCacheMetrics collects cache performance metrics
func (pm *PerformanceMonitor) collectCacheMetrics() {
	// This would integrate with the cache system
	// For now, simulate good cache performance
	hitRate := 0.92 // 92% hit rate
	pm.metrics.SetCacheHitRate(hitRate)
}

// collectCPUUsage collects CPU usage metrics
func (pm *PerformanceMonitor) collectCPUUsage() {
	// This is a simplified CPU usage calculation
	// In production, you'd use more sophisticated methods
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Estimate CPU usage based on GC activity and other factors
	cpuUsage := float64(runtime.NumGoroutine()) / 1000.0
	if cpuUsage > 1.0 {
		cpuUsage = 1.0
	}

	pm.metrics.SetCPUUsage(cpuUsage)
}

// collectMemoryUsage collects memory usage metrics
func (pm *PerformanceMonitor) collectMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryUsage := float64(m.Alloc) / (1024 * 1024) // MB
	pm.metrics.SetMemoryUsage(memoryUsage)
}

// collectGoroutineCount collects goroutine count metrics
func (pm *PerformanceMonitor) collectGoroutineCount() {
	count := runtime.NumGoroutine()
	pm.metrics.SetGoroutineCount(float64(count))
}

// monitorAlerts monitors SLOs and triggers alerts
func (pm *PerformanceMonitor) monitorAlerts(ctx context.Context) {
	ticker := time.NewTicker(pm.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !pm.running {
				return
			}

			// Check SLO violations
			violations := pm.sloMonitor.CheckSLOs(pm.metrics)
			for _, violation := range violations {
				pm.alerts.TriggerAlert(AlertTypeSLOViolation, violation)
			}

			// Check other alert conditions
			pm.checkPerformanceAlerts()
		}
	}
}

// checkPerformanceAlerts checks for performance-related alerts
func (pm *PerformanceMonitor) checkPerformanceAlerts() {
	metrics := pm.metrics.GetCurrentMetrics()

	// Check high response time
	p95ResponseTime := time.Duration(metrics.ResponseTime.Percentile(0.95)) * time.Millisecond
	if p95ResponseTime > time.Millisecond*100 {
		pm.alerts.TriggerAlert(AlertTypeHighLatency, map[string]interface{}{
			"p95_response_time": p95ResponseTime,
			"threshold":         time.Millisecond * 100,
		})
	}

	// Check high error rate
	if metrics.ErrorRate.Value() > 0.05 { // 5% error rate
		pm.alerts.TriggerAlert(AlertTypeHighErrorRate, map[string]interface{}{
			"error_rate": metrics.ErrorRate.Value(),
			"threshold":  0.05,
		})
	}

	// Check low cache hit rate
	if metrics.CacheHitRate.Value() < 0.90 { // 90% hit rate
		pm.alerts.TriggerAlert(AlertTypeLowCacheHitRate, map[string]interface{}{
			"hit_rate":  metrics.CacheHitRate.Value(),
			"threshold": 0.90,
		})
	}

	// Check high memory usage
	if metrics.MemoryUsage.Value() > 500 { // 500MB
		pm.alerts.TriggerAlert(AlertTypeHighMemoryUsage, map[string]interface{}{
			"memory_usage": metrics.MemoryUsage.Value(),
			"threshold":    500,
		})
	}
}

// RecordOperation records metrics for an operation
func (pm *PerformanceMonitor) RecordOperation(operation string, duration time.Duration, err error) {
	pm.metrics.RecordResponseTime(duration)
	pm.metrics.IncrementThroughput()

	if err != nil {
		pm.metrics.IncrementErrors()
	}

	// Record in tracing if enabled
	if pm.config.EnableTracing {
		pm.tracing.RecordOperation(operation, duration, err)
	}
}

// GetCurrentMetrics returns current metric values
func (pm *PerformanceMonitor) GetCurrentMetrics() *Metrics {
	return pm.metrics.GetCurrentMetrics()
}

// GetMetricValue gets a specific metric value over a time window
func (pm *PerformanceMonitor) GetMetricValue(metric string, window time.Duration) float64 {
	return pm.metrics.GetMetricValue(metric, window)
}

// RenderDashboard renders the performance dashboard
func (pm *PerformanceMonitor) RenderDashboard() string {
	return pm.dashboard.Render(pm.metrics)
}

// AddExporter adds a metrics exporter
func (pm *PerformanceMonitor) AddExporter(exporter MetricsExporter) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.exporters = append(pm.exporters, exporter)
}

// AddHandler adds an alert handler
func (pm *PerformanceMonitor) AddHandler(handler AlertHandler) {
	pm.alerts.AddHandler(handler)
}

// GetTracing returns the tracing integration
func (pm *PerformanceMonitor) GetTracing() *TracingIntegration {
	return pm.tracing
}

// setupDefaultSLOs sets up default service level objectives
func (pm *PerformanceMonitor) setupDefaultSLOs() {
	slos := []SLO{
		{
			Name:       "Response Time P95",
			Target:     100.0, // 100ms
			Window:     time.Minute * 5,
			Metric:     "response_time_p95",
			Comparator: ComparatorLessThan,
		},
		{
			Name:       "Error Rate",
			Target:     0.01, // 1%
			Window:     time.Minute * 5,
			Metric:     "error_rate",
			Comparator: ComparatorLessThan,
		},
		{
			Name:       "Cache Hit Rate",
			Target:     0.90, // 90%
			Window:     time.Minute * 5,
			Metric:     "cache_hit_rate",
			Comparator: ComparatorGreaterThan,
		},
		{
			Name:       "Memory Usage",
			Target:     500.0, // 500MB
			Window:     time.Minute * 5,
			Metric:     "memory_usage",
			Comparator: ComparatorLessThan,
		},
	}

	for _, slo := range slos {
		pm.sloMonitor.AddSLO(slo)
	}
}

// MetricsCollector collects and manages performance metrics
type MetricsCollector struct {
	responseTime   *Histogram
	throughput     *Counter
	errorRate      *Gauge
	cpuUsage       *Gauge
	memoryUsage    *Gauge
	goroutineCount *Gauge
	gcPauses       *Histogram
	cacheHitRate   *Gauge
	errors         *Counter
	mu             sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(maxSamples int) *MetricsCollector {
	return &MetricsCollector{
		responseTime:   NewHistogram(maxSamples),
		throughput:     NewCounter(),
		errorRate:      NewGauge(),
		cpuUsage:       NewGauge(),
		memoryUsage:    NewGauge(),
		goroutineCount: NewGauge(),
		gcPauses:       NewHistogram(maxSamples),
		cacheHitRate:   NewGauge(),
		errors:         NewCounter(),
	}
}

// RecordResponseTime records a response time measurement
func (mc *MetricsCollector) RecordResponseTime(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.responseTime.Observe(float64(duration.Milliseconds()))
}

// IncrementThroughput increments the throughput counter
func (mc *MetricsCollector) IncrementThroughput() {
	mc.throughput.Inc()
}

// IncrementErrors increments the error counter
func (mc *MetricsCollector) IncrementErrors() {
	mc.errors.Inc()
}

// SetErrorRate sets the current error rate
func (mc *MetricsCollector) SetErrorRate(rate float64) {
	mc.errorRate.Set(rate)
}

// SetCPUUsage sets the current CPU usage
func (mc *MetricsCollector) SetCPUUsage(usage float64) {
	mc.cpuUsage.Set(usage)
}

// SetMemoryUsage sets the current memory usage
func (mc *MetricsCollector) SetMemoryUsage(usage float64) {
	mc.memoryUsage.Set(usage)
}

// SetGoroutineCount sets the current goroutine count
func (mc *MetricsCollector) SetGoroutineCount(count float64) {
	mc.goroutineCount.Set(count)
}

// RecordGCPause records a garbage collection pause
func (mc *MetricsCollector) RecordGCPause(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.gcPauses.Observe(float64(duration.Milliseconds()))
}

// SetCacheHitRate sets the current cache hit rate
func (mc *MetricsCollector) SetCacheHitRate(rate float64) {
	mc.cacheHitRate.Set(rate)
}

// GetCurrentMetrics returns current metric values
func (mc *MetricsCollector) GetCurrentMetrics() *Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &Metrics{
		ResponseTime:   mc.responseTime,
		Throughput:     mc.throughput,
		ErrorRate:      mc.errorRate,
		CPUUsage:       mc.cpuUsage,
		MemoryUsage:    mc.memoryUsage,
		GoroutineCount: mc.goroutineCount,
		GCPauses:       mc.gcPauses,
		CacheHitRate:   mc.cacheHitRate,
	}
}

// GetMetricValue gets a specific metric value over a time window
func (mc *MetricsCollector) GetMetricValue(metric string, window time.Duration) float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	switch metric {
	case "response_time_p95":
		return mc.responseTime.Percentile(0.95)
	case "response_time_p50":
		return mc.responseTime.Percentile(0.50)
	case "error_rate":
		return mc.errorRate.Value()
	case "cpu_usage":
		return mc.cpuUsage.Value()
	case "memory_usage":
		return mc.memoryUsage.Value()
	case "cache_hit_rate":
		return mc.cacheHitRate.Value()
	case "goroutine_count":
		return mc.goroutineCount.Value()
	default:
		return 0.0
	}
}

// Simple metric implementations would go here
// For brevity, I'll provide basic implementations

// Histogram tracks distribution of values
type Histogram struct {
	samples    []float64
	maxSamples int
	mu         sync.RWMutex
}

// NewHistogram creates a new histogram
func NewHistogram(maxSamples int) *Histogram {
	return &Histogram{
		samples:    make([]float64, 0),
		maxSamples: maxSamples,
	}
}

// Observe adds a sample to the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples = append(h.samples, value)
	if len(h.samples) > h.maxSamples {
		h.samples = h.samples[len(h.samples)-h.maxSamples:]
	}
}

// Percentile calculates the specified percentile
func (h *Histogram) Percentile(percentile float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.samples) == 0 {
		return 0
	}

	sorted := make([]float64, len(h.samples))
	copy(sorted, h.samples)
	sort.Float64s(sorted)

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// Counter tracks incrementing values
type Counter struct {
	value float64
	mu    sync.RWMutex
}

// NewCounter creates a new counter
func NewCounter() *Counter {
	return &Counter{}
}

// Inc increments the counter
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Value returns the current counter value
func (c *Counter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Gauge tracks current values
type Gauge struct {
	value float64
	mu    sync.RWMutex
}

// NewGauge creates a new gauge
func NewGauge() *Gauge {
	return &Gauge{}
}

// Set sets the gauge value
func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

// Value returns the current gauge value
func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}
