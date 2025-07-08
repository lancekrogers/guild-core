// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package monitoring

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewPerformanceMonitor(t *testing.T) {
	config := DefaultMonitoringConfig()
	monitor := NewPerformanceMonitor(config)

	if monitor == nil {
		t.Fatal("Expected monitor to be non-nil")
	}

	if monitor.metrics == nil {
		t.Error("Expected metrics collector to be initialized")
	}

	if monitor.alerts == nil {
		t.Error("Expected alert manager to be initialized")
	}

	if monitor.dashboard == nil {
		t.Error("Expected dashboard to be initialized")
	}

	if monitor.sloMonitor == nil {
		t.Error("Expected SLO monitor to be initialized")
	}

	if monitor.tracing == nil {
		t.Error("Expected tracing integration to be initialized")
	}

	if monitor.config != config {
		t.Error("Expected config to match input config")
	}
}

func TestDefaultMonitoringConfig(t *testing.T) {
	config := DefaultMonitoringConfig()

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.MetricsInterval <= 0 {
		t.Error("Expected metrics interval to be positive")
	}

	if config.AlertCheckInterval <= 0 {
		t.Error("Expected alert check interval to be positive")
	}

	if config.DashboardRefresh <= 0 {
		t.Error("Expected dashboard refresh to be positive")
	}

	if config.RetentionPeriod <= 0 {
		t.Error("Expected retention period to be positive")
	}

	if config.MaxMetricSamples <= 0 {
		t.Error("Expected max metric samples to be positive")
	}
}

func TestPerformanceMonitorStartStop(t *testing.T) {
	monitor := NewPerformanceMonitor(nil)

	// Test starting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !monitor.running {
		t.Error("Expected monitor to be running after start")
	}

	// Test starting again (should fail)
	err = monitor.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running monitor")
	}

	// Test stopping
	err = monitor.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if monitor.running {
		t.Error("Expected monitor to be stopped after stop")
	}

	// Test stopping again (should fail)
	err = monitor.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped monitor")
	}
}

func TestRecordOperation(t *testing.T) {
	monitor := NewPerformanceMonitor(nil)

	operation := "test-operation"
	duration := time.Millisecond * 50

	// Test without error
	monitor.RecordOperation(operation, duration, nil)

	// Test with error
	monitor.RecordOperation(operation, duration, fmt.Errorf("test error"))

	// Verify metrics were recorded
	metrics := monitor.GetCurrentMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be non-nil")
	}

	if metrics.Throughput.Value() <= 0 {
		t.Error("Expected throughput to be recorded")
	}
}

func TestGetCurrentMetrics(t *testing.T) {
	monitor := NewPerformanceMonitor(nil)

	metrics := monitor.GetCurrentMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be non-nil")
	}

	if metrics.ResponseTime == nil {
		t.Error("Expected response time metric to be non-nil")
	}

	if metrics.Throughput == nil {
		t.Error("Expected throughput metric to be non-nil")
	}

	if metrics.ErrorRate == nil {
		t.Error("Expected error rate metric to be non-nil")
	}

	if metrics.CPUUsage == nil {
		t.Error("Expected CPU usage metric to be non-nil")
	}

	if metrics.MemoryUsage == nil {
		t.Error("Expected memory usage metric to be non-nil")
	}

	if metrics.GoroutineCount == nil {
		t.Error("Expected goroutine count metric to be non-nil")
	}

	if metrics.GCPauses == nil {
		t.Error("Expected GC pauses metric to be non-nil")
	}

	if metrics.CacheHitRate == nil {
		t.Error("Expected cache hit rate metric to be non-nil")
	}
}

func TestRenderDashboard(t *testing.T) {
	monitor := NewPerformanceMonitor(nil)

	// Record some metrics first
	monitor.RecordOperation("test", time.Millisecond*50, nil)

	dashboard := monitor.RenderDashboard()
	if dashboard == "" {
		t.Error("Expected dashboard to be non-empty")
	}

	// Dashboard should contain key sections
	if !contains(dashboard, "PERFORMANCE DASHBOARD") {
		t.Error("Expected dashboard to contain title")
	}

	if !contains(dashboard, "Response Time") {
		t.Error("Expected dashboard to contain response time section")
	}

	if !contains(dashboard, "Throughput") {
		t.Error("Expected dashboard to contain throughput section")
	}

	if !contains(dashboard, "Resource Usage") {
		t.Error("Expected dashboard to contain resource usage section")
	}
}

func TestAddExporter(t *testing.T) {
	monitor := NewPerformanceMonitor(nil)

	// Add Prometheus exporter
	prometheusExporter := NewPrometheusExporter("http://localhost:9090", time.Second*30)
	monitor.AddExporter(prometheusExporter)

	if len(monitor.exporters) != 1 {
		t.Error("Expected one exporter to be added")
	}

	// Add Jaeger exporter
	jaegerExporter := NewJaegerExporter("http://localhost:14268")
	monitor.AddExporter(jaegerExporter)

	if len(monitor.exporters) != 2 {
		t.Error("Expected two exporters to be added")
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(1000)

	if collector == nil {
		t.Fatal("Expected collector to be non-nil")
	}

	// Test recording response time
	collector.RecordResponseTime(time.Millisecond * 100)
	metrics := collector.GetCurrentMetrics()

	if metrics.ResponseTime.Percentile(0.50) <= 0 {
		t.Error("Expected response time to be recorded")
	}

	// Test incrementing throughput
	collector.IncrementThroughput()
	if metrics.Throughput.Value() <= 0 {
		t.Error("Expected throughput to be incremented")
	}

	// Test setting error rate
	collector.SetErrorRate(0.05)
	if metrics.ErrorRate.Value() != 0.05 {
		t.Error("Expected error rate to be set correctly")
	}

	// Test setting CPU usage
	collector.SetCPUUsage(0.75)
	if metrics.CPUUsage.Value() != 0.75 {
		t.Error("Expected CPU usage to be set correctly")
	}

	// Test setting memory usage
	collector.SetMemoryUsage(512.0)
	if metrics.MemoryUsage.Value() != 512.0 {
		t.Error("Expected memory usage to be set correctly")
	}

	// Test setting goroutine count
	collector.SetGoroutineCount(150.0)
	if metrics.GoroutineCount.Value() != 150.0 {
		t.Error("Expected goroutine count to be set correctly")
	}

	// Test recording GC pause
	collector.RecordGCPause(time.Millisecond * 5)
	if metrics.GCPauses.Percentile(0.50) <= 0 {
		t.Error("Expected GC pause to be recorded")
	}

	// Test setting cache hit rate
	collector.SetCacheHitRate(0.92)
	if metrics.CacheHitRate.Value() != 0.92 {
		t.Error("Expected cache hit rate to be set correctly")
	}
}

func TestHistogram(t *testing.T) {
	histogram := NewHistogram(100)

	if histogram == nil {
		t.Fatal("Expected histogram to be non-nil")
	}

	// Test observing values
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, value := range values {
		histogram.Observe(value)
	}

	// Test percentiles
	p50 := histogram.Percentile(0.50)
	if p50 < 40 || p50 > 60 {
		t.Errorf("Expected P50 to be around 50, got %f", p50)
	}

	p95 := histogram.Percentile(0.95)
	if p95 < 90 {
		t.Errorf("Expected P95 to be around 95, got %f", p95)
	}

	// Test with empty histogram
	emptyHistogram := NewHistogram(100)
	if emptyHistogram.Percentile(0.50) != 0 {
		t.Error("Expected 0 percentile for empty histogram")
	}
}

func TestCounter(t *testing.T) {
	counter := NewCounter()

	if counter == nil {
		t.Fatal("Expected counter to be non-nil")
	}

	// Initial value should be 0
	if counter.Value() != 0 {
		t.Error("Expected initial counter value to be 0")
	}

	// Test incrementing
	counter.Inc()
	if counter.Value() != 1 {
		t.Error("Expected counter value to be 1 after increment")
	}

	// Test multiple increments
	for i := 0; i < 9; i++ {
		counter.Inc()
	}

	if counter.Value() != 10 {
		t.Error("Expected counter value to be 10 after 10 increments")
	}
}

func TestGauge(t *testing.T) {
	gauge := NewGauge()

	if gauge == nil {
		t.Fatal("Expected gauge to be non-nil")
	}

	// Initial value should be 0
	if gauge.Value() != 0 {
		t.Error("Expected initial gauge value to be 0")
	}

	// Test setting values
	gauge.Set(42.5)
	if gauge.Value() != 42.5 {
		t.Error("Expected gauge value to be 42.5")
	}

	// Test setting negative values
	gauge.Set(-10.0)
	if gauge.Value() != -10.0 {
		t.Error("Expected gauge value to be -10.0")
	}
}

func TestGetMetricValue(t *testing.T) {
	collector := NewMetricsCollector(1000)

	// Set some metric values
	collector.RecordResponseTime(time.Millisecond * 50)
	collector.SetErrorRate(0.02)
	collector.SetCPUUsage(0.65)
	collector.SetMemoryUsage(400.0)
	collector.SetCacheHitRate(0.95)
	collector.SetGoroutineCount(200.0)

	window := time.Minute * 5

	// Test getting different metric values
	p95 := collector.GetMetricValue("response_time_p95", window)
	if p95 <= 0 {
		t.Error("Expected P95 response time to be positive")
	}

	p50 := collector.GetMetricValue("response_time_p50", window)
	if p50 <= 0 {
		t.Error("Expected P50 response time to be positive")
	}

	errorRate := collector.GetMetricValue("error_rate", window)
	if errorRate != 0.02 {
		t.Errorf("Expected error rate to be 0.02, got %f", errorRate)
	}

	cpuUsage := collector.GetMetricValue("cpu_usage", window)
	if cpuUsage != 0.65 {
		t.Errorf("Expected CPU usage to be 0.65, got %f", cpuUsage)
	}

	memoryUsage := collector.GetMetricValue("memory_usage", window)
	if memoryUsage != 400.0 {
		t.Errorf("Expected memory usage to be 400.0, got %f", memoryUsage)
	}

	cacheHitRate := collector.GetMetricValue("cache_hit_rate", window)
	if cacheHitRate != 0.95 {
		t.Errorf("Expected cache hit rate to be 0.95, got %f", cacheHitRate)
	}

	goroutineCount := collector.GetMetricValue("goroutine_count", window)
	if goroutineCount != 200.0 {
		t.Errorf("Expected goroutine count to be 200.0, got %f", goroutineCount)
	}

	// Test unknown metric
	unknown := collector.GetMetricValue("unknown_metric", window)
	if unknown != 0.0 {
		t.Error("Expected 0.0 for unknown metric")
	}
}

// Benchmark tests
func BenchmarkRecordOperation(b *testing.B) {
	monitor := NewPerformanceMonitor(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordOperation("test-op", time.Millisecond*50, nil)
	}
}

func BenchmarkHistogramObserve(b *testing.B) {
	histogram := NewHistogram(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogram.Observe(float64(i % 100))
	}
}

func BenchmarkHistogramPercentile(b *testing.B) {
	histogram := NewHistogram(1000)

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		histogram.Observe(float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		histogram.Percentile(0.95)
	}
}

func BenchmarkCounterInc(b *testing.B) {
	counter := NewCounter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Inc()
	}
}

func BenchmarkGaugeSet(b *testing.B) {
	gauge := NewGauge()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gauge.Set(float64(i))
	}
}

func BenchmarkRenderDashboard(b *testing.B) {
	monitor := NewPerformanceMonitor(nil)

	// Pre-populate with some metrics
	for i := 0; i < 100; i++ {
		monitor.RecordOperation("test-op", time.Millisecond*50, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RenderDashboard()
	}
}

// Test concurrent access
func TestMetricsCollectorConcurrency(t *testing.T) {
	collector := NewMetricsCollector(1000)

	done := make(chan bool, 10)

	// Run multiple goroutines concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				collector.RecordResponseTime(time.Millisecond * time.Duration(j))
				collector.IncrementThroughput()
				collector.SetErrorRate(float64(j) / 1000.0)
				collector.SetCPUUsage(float64(j) / 100.0)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics are still accessible
	metrics := collector.GetCurrentMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be accessible after concurrent access")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
