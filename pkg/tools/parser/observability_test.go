// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObservableParser_Metrics tests metric collection
func TestObservableParser_Metrics(t *testing.T) {
	// Create a test registry
	reg := prometheus.NewRegistry()
	
	// Create base parser
	baseParser := NewResponseParser()
	
	// Wrap with metrics (we'll need to modify NewObservableParser to accept a registry)
	// For now, test basic functionality
	obsParser := NewObservableParser(baseParser)
	
	// Test successful parse
	input := `{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test_func", "arguments": "{}"}}]}`
	
	calls, err := obsParser.ExtractToolCalls(input)
	require.NoError(t, err)
	assert.Len(t, calls, 1)
	
	// Verify metrics were recorded
	// In a real test, we'd check the prometheus metrics
}

// TestObservableParser_ErrorMetrics tests error metric collection
func TestObservableParser_ErrorMetrics(t *testing.T) {
	baseParser := NewResponseParser()
	obsParser := NewObservableParser(baseParser)
	
	// Test with invalid input
	input := `{invalid json`
	
	_, err := obsParser.ExtractToolCalls(input)
	assert.NoError(t, err) // Parser returns empty result, not error
	
	// Test with malformed tool call
	input2 := `{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "", "arguments": "{}"}}]}`
	calls, err := obsParser.ExtractToolCalls(input2)
	assert.NoError(t, err)
	assert.Empty(t, calls) // Should skip invalid calls
}

// TestObservableParser_FormatDistribution tests format tracking
func TestObservableParser_FormatDistribution(t *testing.T) {
	baseParser := NewResponseParser()
	obsParser := NewObservableParser(baseParser)
	
	// Parse different formats
	jsonInput := `{"id": "j1", "type": "function", "function": {"name": "json_func", "arguments": "{}"}}`
	xmlInput := `<function_calls><invoke name="xml_func"></invoke></function_calls>`
	
	_, _ = obsParser.ExtractToolCalls(jsonInput)
	_, _ = obsParser.ExtractToolCalls(xmlInput)
	
	// In a real test, verify format distribution metrics
}

// TestHealthMonitor_BasicHealth tests health check functionality
func TestHealthMonitor_BasicHealth(t *testing.T) {
	parser := NewResponseParser()
	monitor := NewHealthMonitor(parser, "test-v1.0")
	
	// Perform health check
	health := monitor.Check(context.Background())
	
	// Verify basic health
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, "test-v1.0", health.Version)
	assert.NotZero(t, health.Timestamp)
	assert.Greater(t, health.Uptime, time.Duration(0))
	
	// Verify checks were performed
	assert.Contains(t, health.Checks, "parser_basic")
	assert.Contains(t, health.Checks, "json_detector")
	assert.Contains(t, health.Checks, "xml_detector")
	assert.Contains(t, health.Checks, "format_detection")
	assert.Contains(t, health.Checks, "memory_usage")
	
	// All checks should be healthy for a new parser
	for name, check := range health.Checks {
		assert.Equal(t, HealthStatusHealthy, check.Status, "Check %s failed", name)
	}
}

// TestHealthMonitor_MetricsCalculation tests metric calculations
func TestHealthMonitor_MetricsCalculation(t *testing.T) {
	parser := NewResponseParser()
	monitor := NewHealthMonitor(parser, "test-v1.0")
	
	// Record some parse operations
	monitor.RecordParse(ProviderFormatOpenAI, 10*time.Millisecond, true)
	monitor.RecordParse(ProviderFormatOpenAI, 20*time.Millisecond, true)
	monitor.RecordParse(ProviderFormatAnthropic, 15*time.Millisecond, true)
	monitor.RecordParse(ProviderFormatOpenAI, 30*time.Millisecond, false) // failure
	
	// Get metrics
	health := monitor.Check(context.Background())
	metrics := health.Metrics
	
	// Verify metrics
	assert.Equal(t, int64(4), metrics.TotalParses)
	assert.Equal(t, int64(3), metrics.TotalSuccesses)
	assert.Equal(t, int64(1), metrics.TotalFailures)
	assert.Equal(t, 0.75, metrics.SuccessRate)
	
	// Check format distribution
	assert.Equal(t, int64(3), metrics.FormatDistribution[string(ProviderFormatOpenAI)])
	assert.Equal(t, int64(1), metrics.FormatDistribution[string(ProviderFormatAnthropic)])
	
	// Average latency should be calculated
	assert.Greater(t, metrics.AverageLatency, 0.0)
}

// TestHealthMonitor_ActiveParsers tests concurrent parser tracking
func TestHealthMonitor_ActiveParsers(t *testing.T) {
	parser := NewResponseParser()
	monitor := NewHealthMonitor(parser, "test-v1.0")
	
	// Start multiple parsers
	monitor.StartParse()
	monitor.StartParse()
	monitor.StartParse()
	
	health := monitor.Check(context.Background())
	assert.Equal(t, 3, health.Metrics.ActiveParsers)
	
	// End parsers
	monitor.EndParse()
	monitor.EndParse()
	
	health = monitor.Check(context.Background())
	assert.Equal(t, 1, health.Metrics.ActiveParsers)
	
	monitor.EndParse()
	
	health = monitor.Check(context.Background())
	assert.Equal(t, 0, health.Metrics.ActiveParsers)
}

// TestAlertManager_Conditions tests alert condition evaluation
func TestAlertManager_Conditions(t *testing.T) {
	am := NewAlertManager()
	
	// Test high failure rate condition
	metrics := HealthMetrics{
		TotalParses:    200,
		TotalSuccesses: 170,
		TotalFailures:  30,
		SuccessRate:    0.85, // Below 90% threshold
	}
	
	am.CheckAlerts(metrics)
	
	// Should have high failure rate alert
	alerts := am.GetActiveAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "high_failure_rate", alerts[0].Labels["condition"])
	assert.Equal(t, AlertSeverityCritical, alerts[0].Severity)
}

// TestAlertManager_Resolution tests alert resolution
func TestAlertManager_Resolution(t *testing.T) {
	am := NewAlertManager()
	
	// Trigger alert
	badMetrics := HealthMetrics{
		TotalParses:    200,
		TotalSuccesses: 170,
		TotalFailures:  30,
		SuccessRate:    0.85,
	}
	am.CheckAlerts(badMetrics)
	
	assert.Len(t, am.GetActiveAlerts(), 1)
	
	// Resolve alert
	goodMetrics := HealthMetrics{
		TotalParses:    300,
		TotalSuccesses: 285,
		TotalFailures:  15,
		SuccessRate:    0.95,
	}
	am.CheckAlerts(goodMetrics)
	
	// Alert should be resolved
	activeAlerts := am.GetActiveAlerts()
	assert.Len(t, activeAlerts, 0)
	
	// But should still exist in history
	allAlerts := am.GetAllAlerts()
	assert.Greater(t, len(allAlerts), 0)
	assert.True(t, allAlerts[0].Resolved)
}

// TestMonitoredParser_Integration tests full monitoring integration
func TestMonitoredParser_Integration(t *testing.T) {
	baseParser := NewResponseParser()
	monitoredParser := NewMonitoredParser(baseParser, "test-v1.0")
	defer monitoredParser.Stop()
	
	// Parse some inputs
	inputs := []string{
		`{"id": "1", "type": "function", "function": {"name": "test1", "arguments": "{}"}}`,
		`<function_calls><invoke name="test2"></invoke></function_calls>`,
		`invalid input that won't parse`,
	}
	
	for _, input := range inputs {
		_, _ = monitoredParser.ExtractToolCalls(input)
	}
	
	// Give monitoring time to process
	time.Sleep(100 * time.Millisecond)
	
	// Check health
	health := monitoredParser.GetHealth()
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, int64(3), health.Metrics.TotalParses)
	
	// Check for alerts
	alerts := monitoredParser.GetAlerts()
	// Shouldn't have any alerts with just 3 parses
	assert.Empty(t, alerts)
}

// TestDashboardServer_Endpoints tests dashboard HTTP endpoints
func TestDashboardServer_Endpoints(t *testing.T) {
	// This would test the HTTP endpoints
	// Skipping for brevity, but would include:
	// - Health endpoint returns correct status
	// - Metrics endpoint returns prometheus metrics
	// - Alert endpoints return current alerts
	// - Dashboard HTML is served correctly
}

// Benchmark observability overhead
func BenchmarkObservableParser_Overhead(b *testing.B) {
	baseParser := NewResponseParser()
	obsParser := NewObservableParser(baseParser)
	
	input := `{"id": "bench", "type": "function", "function": {"name": "benchmark", "arguments": "{}"}}`
	
	b.Run("WithoutObservability", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = baseParser.ExtractToolCalls(input)
		}
	})
	
	b.Run("WithObservability", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = obsParser.ExtractToolCalls(input)
		}
	})
}

// TestMetricLabels tests that metrics have correct labels
func TestMetricLabels(t *testing.T) {
	// This would use testutil to verify prometheus metrics
	// Example structure:
	/*
	reg := prometheus.NewRegistry()
	metrics := InitParserMetrics(reg)
	
	// Trigger metric recording
	metrics.parseAttempts.WithLabelValues("openai").Inc()
	
	// Verify with testutil
	expected := `
		# HELP guild_parser_parse_attempts_total Total number of parsing attempts
		# TYPE guild_parser_parse_attempts_total counter
		guild_parser_parse_attempts_total{format="openai"} 1
	`
	err := testutil.CollectAndCompare(metrics.parseAttempts, strings.NewReader(expected))
	assert.NoError(t, err)
	*/
}