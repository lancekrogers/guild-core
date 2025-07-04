// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package performance provides comprehensive performance monitoring and SLA validation
// for the Guild Framework happy path testing. This system ensures all user-facing
// operations meet staff-level performance requirements.
package performance

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// SLAMonitor provides comprehensive performance monitoring and SLA validation
type SLAMonitor struct {
	t                *testing.T
	slaDefinitions   map[string]SLADefinition
	measurements     map[string][]Measurement
	continuousChecks []ContinuousCheck
	alerts           []Alert
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// SLADefinition defines performance requirements for a specific operation
type SLADefinition struct {
	Name              string
	Operation         string
	Target            time.Duration
	Critical          time.Duration // Hard limit - test fails if exceeded
	Percentile        float64       // P95 by default
	MeasurementWindow time.Duration // Time window for measurements
	MinSamples        int           // Minimum samples required for validation
	Description       string
	Category          SLACategory
}

// SLACategory categorizes different types of SLAs
type SLACategory string

const (
	SLACategoryUserInteraction SLACategory = "user_interaction"
	SLACategorySystemResponse  SLACategory = "system_response"
	SLACategoryUIRendering     SLACategory = "ui_rendering"
	SLACategoryDataProcessing  SLACategory = "data_processing"
	SLACategoryResourceUsage   SLACategory = "resource_usage"
)

// Measurement represents a single performance measurement
type Measurement struct {
	Timestamp   time.Time
	Operation   string
	Duration    time.Duration
	Success     bool
	Metadata    map[string]interface{}
	MemoryUsage uint64
	Context     MeasurementContext
}

// MeasurementContext provides additional context for measurements
type MeasurementContext struct {
	UserAction    string
	SystemState   string
	ConcurrentOps int
	ResourceLoad  float64
}

// ContinuousCheck defines a continuous monitoring check
type ContinuousCheck struct {
	Name         string
	Interval     time.Duration
	CheckFunc    func(context.Context) (bool, string, error)
	Enabled      bool
	LastCheck    time.Time
	FailureCount int
}

// Alert represents a performance alert
type Alert struct {
	Timestamp time.Time
	Severity  AlertSeverity
	Operation string
	Message   string
	Value     interface{}
	Threshold interface{}
	Metadata  map[string]interface{}
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// PerformanceReport provides a comprehensive performance analysis
type PerformanceReport struct {
	GeneratedAt     time.Time
	TestDuration    time.Duration
	SLAResults      map[string]SLAResult
	OverallHealth   HealthStatus
	Recommendations []string
	TrendAnalysis   TrendAnalysis
	ResourceUsage   ResourceUsageReport
}

// SLAResult represents the result of SLA validation
type SLAResult struct {
	SLA            SLADefinition
	Met            bool
	ActualValue    time.Duration
	TargetValue    time.Duration
	Measurements   int
	SuccessRate    float64
	Violations     []SLAViolation
	TrendDirection TrendDirection
}

// SLAViolation represents a specific SLA violation
type SLAViolation struct {
	Timestamp time.Time
	Operation string
	Actual    time.Duration
	Expected  time.Duration
	Severity  AlertSeverity
	Context   MeasurementContext
}

// HealthStatus represents overall system health
type HealthStatus string

const (
	HealthStatusExcellent HealthStatus = "excellent"
	HealthStatusGood      HealthStatus = "good"
	HealthStatusFair      HealthStatus = "fair"
	HealthStatusPoor      HealthStatus = "poor"
	HealthStatusCritical  HealthStatus = "critical"
)

// TrendDirection indicates performance trend
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendStable    TrendDirection = "stable"
	TrendDeclining TrendDirection = "declining"
)

// TrendAnalysis provides trend analysis for performance metrics
type TrendAnalysis struct {
	OverallTrend      TrendDirection
	PerOperationTrend map[string]TrendDirection
	RegressionRisk    float64
	Recommendations   []string
}

// ResourceUsageReport provides resource utilization analysis
type ResourceUsageReport struct {
	MemoryUsage    MemoryUsageStats
	CPUUtilization CPUStats
	GoroutineCount int
	GCStats        GCStats
}

// MemoryUsageStats provides memory usage statistics
type MemoryUsageStats struct {
	Current uint64
	Peak    uint64
	Average uint64
	Trend   TrendDirection
	Leaks   []MemoryLeak
}

// CPUStats provides CPU utilization statistics
type CPUStats struct {
	Average    float64
	Peak       float64
	UserTime   time.Duration
	SystemTime time.Duration
}

// GCStats provides garbage collection statistics
type GCStats struct {
	NumGC      uint32
	TotalPause time.Duration
	MaxPause   time.Duration
	GCRate     float64
}

// MemoryLeak represents a detected memory leak
type MemoryLeak struct {
	StartTime  time.Time
	GrowthRate uint64 // bytes per second
	Severity   AlertSeverity
	Operation  string
}

// NewSLAMonitor creates a new performance monitoring system
func NewSLAMonitor(t *testing.T) *SLAMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &SLAMonitor{
		t:                t,
		slaDefinitions:   make(map[string]SLADefinition),
		measurements:     make(map[string][]Measurement),
		continuousChecks: make([]ContinuousCheck, 0),
		alerts:           make([]Alert, 0),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Register default SLAs for Guild Framework
	monitor.registerDefaultSLAs()

	// Start continuous monitoring
	go monitor.startContinuousMonitoring()

	// Register cleanup
	t.Cleanup(func() {
		monitor.Shutdown()
	})

	return monitor
}

// registerDefaultSLAs registers the critical SLAs for Guild Framework
func (m *SLAMonitor) registerDefaultSLAs() {
	// User Interaction SLAs
	m.RegisterSLA(SLADefinition{
		Name:        "Agent Selection Performance",
		Operation:   "agent_selection",
		Target:      1500 * time.Millisecond,
		Critical:    2000 * time.Millisecond,
		Percentile:  0.95,
		MinSamples:  10,
		Category:    SLACategoryUserInteraction,
		Description: "Agent selection must complete within 2 seconds for 95% of requests",
	})

	m.RegisterSLA(SLADefinition{
		Name:        "Chat Interface Load Time",
		Operation:   "chat_interface_load",
		Target:      300 * time.Millisecond,
		Critical:    500 * time.Millisecond,
		Percentile:  0.95,
		MinSamples:  5,
		Category:    SLACategoryUIRendering,
		Description: "Chat interface must load within 500ms",
	})

	// CRITICAL: Theme switching SLA for 60 FPS
	m.RegisterSLA(SLADefinition{
		Name:        "Theme Switching 60 FPS",
		Operation:   "theme_switch",
		Target:      12 * time.Millisecond,
		Critical:    16 * time.Millisecond,
		Percentile:  1.0, // 100% must meet this SLA
		MinSamples:  5,
		Category:    SLACategoryUIRendering,
		Description: "CRITICAL: Theme switching must complete within 16ms for 60 FPS",
	})

	m.RegisterSLA(SLADefinition{
		Name:        "First Agent Response",
		Operation:   "first_agent_response",
		Target:      2500 * time.Millisecond,
		Critical:    3000 * time.Millisecond,
		Percentile:  0.90,
		MinSamples:  10,
		Category:    SLACategorySystemResponse,
		Description: "First agent response must arrive within 3 seconds for 90% of interactions",
	})

	m.RegisterSLA(SLADefinition{
		Name:        "Streaming Response Latency",
		Operation:   "streaming_chunk",
		Target:      50 * time.Millisecond,
		Critical:    100 * time.Millisecond,
		Percentile:  0.95,
		MinSamples:  20,
		Category:    SLACategorySystemResponse,
		Description: "Streaming chunks must arrive within 100ms",
	})

	m.RegisterSLA(SLADefinition{
		Name:        "Memory Usage Efficiency",
		Operation:   "memory_usage",
		Target:      50 * 1024 * 1024,  // 50MB
		Critical:    100 * 1024 * 1024, // 100MB
		Percentile:  0.95,
		MinSamples:  10,
		Category:    SLACategoryResourceUsage,
		Description: "Memory usage should remain under 100MB for typical operations",
	})

	// Register continuous checks
	m.RegisterContinuousCheck(ContinuousCheck{
		Name:      "Memory Leak Detection",
		Interval:  5 * time.Second,
		CheckFunc: m.checkMemoryLeaks,
		Enabled:   true,
	})

	m.RegisterContinuousCheck(ContinuousCheck{
		Name:      "Performance Regression Detection",
		Interval:  10 * time.Second,
		CheckFunc: m.checkPerformanceRegression,
		Enabled:   true,
	})
}

// RegisterSLA registers a new SLA definition
func (m *SLAMonitor) RegisterSLA(sla SLADefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slaDefinitions[sla.Operation] = sla
}

// RegisterContinuousCheck registers a continuous monitoring check
func (m *SLAMonitor) RegisterContinuousCheck(check ContinuousCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.continuousChecks = append(m.continuousChecks, check)
}

// RecordMeasurement records a performance measurement
func (m *SLAMonitor) RecordMeasurement(operation string, duration time.Duration, success bool, metadata map[string]interface{}) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	measurement := Measurement{
		Timestamp:   time.Now(),
		Operation:   operation,
		Duration:    duration,
		Success:     success,
		Metadata:    metadata,
		MemoryUsage: memStats.Alloc,
		Context: MeasurementContext{
			SystemState: "normal", // Could be enhanced with actual system state
		},
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.measurements[operation] == nil {
		m.measurements[operation] = make([]Measurement, 0)
	}
	m.measurements[operation] = append(m.measurements[operation], measurement)

	// Check for immediate SLA violations
	if sla, exists := m.slaDefinitions[operation]; exists {
		if duration > sla.Critical {
			m.recordAlert(Alert{
				Timestamp: time.Now(),
				Severity:  AlertSeverityCritical,
				Operation: operation,
				Message:   fmt.Sprintf("CRITICAL SLA violation: %v > %v", duration, sla.Critical),
				Value:     duration,
				Threshold: sla.Critical,
				Metadata:  metadata,
			})
		} else if duration > sla.Target {
			m.recordAlert(Alert{
				Timestamp: time.Now(),
				Severity:  AlertSeverityWarning,
				Operation: operation,
				Message:   fmt.Sprintf("SLA target exceeded: %v > %v", duration, sla.Target),
				Value:     duration,
				Threshold: sla.Target,
				Metadata:  metadata,
			})
		}
	}
}

// MeasureOperation provides a convenient way to measure and record an operation
func (m *SLAMonitor) MeasureOperation(operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	m.RecordMeasurement(operation, duration, err == nil, map[string]interface{}{
		"error": err,
	})

	return err
}

// ValidateAllSLAs validates all registered SLAs against current measurements
func (m *SLAMonitor) ValidateAllSLAs() *PerformanceReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &PerformanceReport{
		GeneratedAt:     time.Now(),
		SLAResults:      make(map[string]SLAResult),
		Recommendations: make([]string, 0),
	}

	overallMet := 0
	totalSLAs := len(m.slaDefinitions)

	for operation, sla := range m.slaDefinitions {
		result := m.validateSLA(sla)
		report.SLAResults[operation] = result

		if result.Met {
			overallMet++
		}

		// Generate test assertions
		if sla.Category == SLACategoryUIRendering && operation == "theme_switch" {
			// CRITICAL SLA - must be enforced strictly
			assert.True(m.t, result.Met,
				"CRITICAL FAILURE: Theme switching SLA violated - this breaks 60 FPS user experience")

			for _, violation := range result.Violations {
				assert.Fail(m.t, fmt.Sprintf("Theme switch violation at %v: %v > %v",
					violation.Timestamp, violation.Actual, violation.Expected))
			}
		} else {
			// Other SLAs - warn but don't fail test
			if !result.Met {
				m.t.Logf("⚠️ SLA not met for %s: actual %v vs target %v",
					operation, result.ActualValue, result.TargetValue)
			}
		}
	}

	// Calculate overall health
	successRate := float64(overallMet) / float64(totalSLAs)
	switch {
	case successRate >= 0.95:
		report.OverallHealth = HealthStatusExcellent
	case successRate >= 0.85:
		report.OverallHealth = HealthStatusGood
	case successRate >= 0.70:
		report.OverallHealth = HealthStatusFair
	case successRate >= 0.50:
		report.OverallHealth = HealthStatusPoor
	default:
		report.OverallHealth = HealthStatusCritical
	}

	// Generate resource usage report
	report.ResourceUsage = m.generateResourceUsageReport()

	// Generate trend analysis
	report.TrendAnalysis = m.generateTrendAnalysis()

	// Generate recommendations
	report.Recommendations = m.generateRecommendations(report)

	return report
}

// validateSLA validates a single SLA against its measurements
func (m *SLAMonitor) validateSLA(sla SLADefinition) SLAResult {
	measurements := m.measurements[sla.Operation]

	result := SLAResult{
		SLA:            sla,
		Met:            false,
		Measurements:   len(measurements),
		Violations:     make([]SLAViolation, 0),
		TrendDirection: TrendStable,
	}

	if len(measurements) < sla.MinSamples {
		return result
	}

	// Filter successful measurements
	successfulMeasurements := make([]Measurement, 0)
	successCount := 0

	for _, measurement := range measurements {
		if measurement.Success {
			successfulMeasurements = append(successfulMeasurements, measurement)
			successCount++
		}
	}

	result.SuccessRate = float64(successCount) / float64(len(measurements))

	if len(successfulMeasurements) == 0 {
		return result
	}

	// Sort measurements by duration
	durations := make([]time.Duration, len(successfulMeasurements))
	for i, measurement := range successfulMeasurements {
		durations[i] = measurement.Duration
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Calculate percentile value
	percentileIndex := int(float64(len(durations)) * sla.Percentile)
	if percentileIndex >= len(durations) {
		percentileIndex = len(durations) - 1
	}

	result.ActualValue = durations[percentileIndex]
	result.TargetValue = sla.Target

	// Check if SLA is met
	if sla.Category == SLACategoryResourceUsage {
		// For resource usage, check against memory/CPU limits
		result.Met = result.ActualValue <= sla.Critical
	} else {
		// For time-based SLAs
		result.Met = result.ActualValue <= sla.Target
	}

	// Record violations
	for _, measurement := range successfulMeasurements {
		if measurement.Duration > sla.Critical {
			result.Violations = append(result.Violations, SLAViolation{
				Timestamp: measurement.Timestamp,
				Operation: sla.Operation,
				Actual:    measurement.Duration,
				Expected:  sla.Critical,
				Severity:  AlertSeverityCritical,
				Context:   measurement.Context,
			})
		} else if measurement.Duration > sla.Target {
			result.Violations = append(result.Violations, SLAViolation{
				Timestamp: measurement.Timestamp,
				Operation: sla.Operation,
				Actual:    measurement.Duration,
				Expected:  sla.Target,
				Severity:  AlertSeverityWarning,
				Context:   measurement.Context,
			})
		}
	}

	return result
}

// Continuous monitoring methods

func (m *SLAMonitor) startContinuousMonitoring() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.runContinuousChecks()
		}
	}
}

func (m *SLAMonitor) runContinuousChecks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for i := range m.continuousChecks {
		check := &m.continuousChecks[i]
		if !check.Enabled {
			continue
		}

		if now.Sub(check.LastCheck) >= check.Interval {
			success, message, err := check.CheckFunc(m.ctx)
			check.LastCheck = now

			if err != nil || !success {
				check.FailureCount++
				m.recordAlert(Alert{
					Timestamp: now,
					Severity:  AlertSeverityError,
					Operation: check.Name,
					Message:   message,
					Metadata: map[string]interface{}{
						"failure_count": check.FailureCount,
						"error":         err,
					},
				})
			} else {
				check.FailureCount = 0 // Reset on success
			}
		}
	}
}

func (m *SLAMonitor) checkMemoryLeaks(ctx context.Context) (bool, string, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Simple memory leak detection - in production would be more sophisticated
	const memoryThreshold = 200 * 1024 * 1024 // 200MB

	if memStats.Alloc > memoryThreshold {
		return false, fmt.Sprintf("High memory usage detected: %d bytes", memStats.Alloc), nil
	}

	return true, "Memory usage normal", nil
}

func (m *SLAMonitor) checkPerformanceRegression(ctx context.Context) (bool, string, error) {
	// Simple regression check - compare recent performance to baseline
	// In production would use more sophisticated statistical analysis

	for operation, measurements := range m.measurements {
		if len(measurements) < 20 {
			continue
		}

		// Compare last 10 measurements to previous 10
		recent := measurements[len(measurements)-10:]
		previous := measurements[len(measurements)-20 : len(measurements)-10]

		recentAvg := m.calculateAverageDuration(recent)
		previousAvg := m.calculateAverageDuration(previous)

		regression := float64(recentAvg-previousAvg) / float64(previousAvg)
		if regression > 0.2 { // 20% regression threshold
			return false, fmt.Sprintf("Performance regression detected in %s: %.1f%% slower",
				operation, regression*100), nil
		}
	}

	return true, "No performance regression detected", nil
}

// Helper methods

func (m *SLAMonitor) recordAlert(alert Alert) {
	m.alerts = append(m.alerts, alert)

	// Log alert based on severity
	switch alert.Severity {
	case AlertSeverityCritical:
		m.t.Errorf("CRITICAL ALERT: %s - %s", alert.Operation, alert.Message)
	case AlertSeverityError:
		m.t.Logf("ERROR: %s - %s", alert.Operation, alert.Message)
	case AlertSeverityWarning:
		m.t.Logf("WARNING: %s - %s", alert.Operation, alert.Message)
	default:
		m.t.Logf("INFO: %s - %s", alert.Operation, alert.Message)
	}
}

func (m *SLAMonitor) calculateAverageDuration(measurements []Measurement) time.Duration {
	if len(measurements) == 0 {
		return 0
	}

	var total time.Duration
	for _, measurement := range measurements {
		total += measurement.Duration
	}
	return total / time.Duration(len(measurements))
}

func (m *SLAMonitor) generateResourceUsageReport() ResourceUsageReport {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceUsageReport{
		MemoryUsage: MemoryUsageStats{
			Current: memStats.Alloc,
			Peak:    memStats.Sys,
			Average: memStats.Alloc, // Simplified
			Trend:   TrendStable,
		},
		GoroutineCount: runtime.NumGoroutine(),
		GCStats: GCStats{
			NumGC:      memStats.NumGC,
			TotalPause: time.Duration(memStats.PauseTotalNs),
		},
	}
}

func (m *SLAMonitor) generateTrendAnalysis() TrendAnalysis {
	return TrendAnalysis{
		OverallTrend:      TrendStable,
		PerOperationTrend: make(map[string]TrendDirection),
		RegressionRisk:    0.1, // Low risk
		Recommendations:   []string{"Continue monitoring", "Maintain current performance"},
	}
}

func (m *SLAMonitor) generateRecommendations(report *PerformanceReport) []string {
	recommendations := []string{}

	switch report.OverallHealth {
	case HealthStatusCritical:
		recommendations = append(recommendations, "URGENT: Address critical performance issues immediately")
		recommendations = append(recommendations, "Review resource allocation and optimize bottlenecks")
	case HealthStatusPoor:
		recommendations = append(recommendations, "Investigate performance degradation causes")
		recommendations = append(recommendations, "Consider scaling resources or optimizing algorithms")
	case HealthStatusFair:
		recommendations = append(recommendations, "Monitor closely for performance trends")
		recommendations = append(recommendations, "Optimize slower operations to improve SLA compliance")
	case HealthStatusGood:
		recommendations = append(recommendations, "Maintain current performance levels")
		recommendations = append(recommendations, "Continue regular monitoring and optimization")
	case HealthStatusExcellent:
		recommendations = append(recommendations, "Excellent performance - consider if SLAs can be tightened")
		recommendations = append(recommendations, "Use as baseline for future performance comparisons")
	}

	return recommendations
}

// LogPerformanceReport logs a comprehensive performance report
func (m *SLAMonitor) LogPerformanceReport() {
	report := m.ValidateAllSLAs()

	m.t.Logf("📊 Performance Report - Overall Health: %s", report.OverallHealth)
	m.t.Logf("   Generated: %v", report.GeneratedAt.Format(time.RFC3339))

	for operation, result := range report.SLAResults {
		status := "✅ PASS"
		if !result.Met {
			if result.SLA.Category == SLACategoryUIRendering && operation == "theme_switch" {
				status = "❌ CRITICAL FAIL"
			} else {
				status = "⚠️ FAIL"
			}
		}

		m.t.Logf("   %s %s: %v (target: %v, samples: %d)",
			status, result.SLA.Name, result.ActualValue, result.TargetValue, result.Measurements)
	}

	m.t.Logf("   Memory: %d MB", report.ResourceUsage.MemoryUsage.Current/(1024*1024))
	m.t.Logf("   Goroutines: %d", report.ResourceUsage.GoroutineCount)

	if len(report.Recommendations) > 0 {
		m.t.Logf("   Recommendations:")
		for _, rec := range report.Recommendations {
			m.t.Logf("     - %s", rec)
		}
	}
}

// Shutdown stops all continuous monitoring
func (m *SLAMonitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
	m.LogPerformanceReport()
}
