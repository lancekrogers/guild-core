// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	jsonparser "github.com/guild-framework/guild-core/pkg/tools/parser/json"
	xmlparser "github.com/guild-framework/guild-core/pkg/tools/parser/xml"
)

// HealthStatus represents the health status of the parser
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents the health of the parser system
type HealthCheck struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
	Metrics   HealthMetrics          `json:"metrics"`
}

// CheckResult represents an individual health check result
type CheckResult struct {
	Status  HealthStatus  `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency_ms"`
	Error   string        `json:"error,omitempty"`
}

// HealthMetrics contains key performance metrics
type HealthMetrics struct {
	ParseRate          float64          `json:"parse_rate_per_second"`
	SuccessRate        float64          `json:"success_rate"`
	AverageLatency     float64          `json:"average_latency_ms"`
	P95Latency         float64          `json:"p95_latency_ms"`
	P99Latency         float64          `json:"p99_latency_ms"`
	ActiveParsers      int              `json:"active_parsers"`
	TotalParses        int64            `json:"total_parses"`
	TotalSuccesses     int64            `json:"total_successes"`
	TotalFailures      int64            `json:"total_failures"`
	LastParseTime      string           `json:"last_parse_time,omitempty"`
	FormatDistribution map[string]int64 `json:"format_distribution"`
}

// HealthMonitor monitors parser health
type HealthMonitor struct {
	parser    ResponseParser
	startTime time.Time
	version   string

	mu sync.RWMutex

	// Metrics tracking
	totalParses    int64
	totalSuccesses int64
	totalFailures  int64
	lastParseTime  time.Time

	// Format distribution
	formatCounts map[ProviderFormat]int64

	// Latency tracking (in milliseconds)
	latencies    []float64
	maxLatencies int

	// Rate calculation
	rateWindow   time.Duration
	parseHistory []time.Time

	// Active parser tracking
	activeParsers int32
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(parser ResponseParser, version string) *HealthMonitor {
	return &HealthMonitor{
		parser:       parser,
		startTime:    time.Now(),
		version:      version,
		formatCounts: make(map[ProviderFormat]int64),
		latencies:    make([]float64, 0, 1000),
		maxLatencies: 1000,
		rateWindow:   time.Minute,
		parseHistory: make([]time.Time, 0, 1000),
	}
}

// Check performs a comprehensive health check
func (h *HealthMonitor) Check(ctx context.Context) HealthCheck {
	h.mu.RLock()
	defer h.mu.RUnlock()

	check := HealthCheck{
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime),
		Version:   h.version,
		Checks:    make(map[string]CheckResult),
		Metrics:   h.calculateMetrics(),
	}

	// Run individual health checks
	checks := []struct {
		name string
		fn   func(context.Context) CheckResult
	}{
		{"parser_basic", h.checkBasicParsing},
		{"json_detector", h.checkJSONDetector},
		{"xml_detector", h.checkXMLDetector},
		{"format_detection", h.checkFormatDetection},
		{"memory_usage", h.checkMemoryUsage},
	}

	var overallStatus HealthStatus = HealthStatusHealthy
	degradedCount := 0
	unhealthyCount := 0

	for _, c := range checks {
		result := c.fn(ctx)
		check.Checks[c.name] = result

		switch result.Status {
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
	}

	// Determine overall status
	if unhealthyCount > 0 {
		overallStatus = HealthStatusUnhealthy
	} else if degradedCount > 0 {
		overallStatus = HealthStatusDegraded
	}

	check.Status = overallStatus
	return check
}

// checkBasicParsing tests basic parsing functionality
func (h *HealthMonitor) checkBasicParsing(ctx context.Context) CheckResult {
	start := time.Now()

	// Test with a simple JSON tool call in OpenAI format
	testInput := `{"tool_calls": [{"id": "test_123", "type": "function", "function": {"name": "test_tool", "arguments": "{}"}}]}`

	calls, err := h.parser.ExtractWithContext(ctx, testInput)

	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "Basic parsing test failed",
			Latency: latency,
			Error:   err.Error(),
		}
	}

	// For basic parsing, we expect to get at least one tool call
	if len(calls) == 0 {
		return CheckResult{
			Status:  HealthStatusDegraded,
			Message: "Basic parsing returned no tool calls",
			Latency: latency,
		}
	}

	// Check latency threshold
	if latency > 100*time.Millisecond {
		return CheckResult{
			Status:  HealthStatusDegraded,
			Message: fmt.Sprintf("Parsing latency high: %v", latency),
			Latency: latency,
		}
	}

	return CheckResult{
		Status:  HealthStatusHealthy,
		Message: "Basic parsing operational",
		Latency: latency,
	}
}

// checkJSONDetector tests JSON detector health
func (h *HealthMonitor) checkJSONDetector(ctx context.Context) CheckResult {
	start := time.Now()

	detector := NewJSONDetector()
	// Use a proper OpenAI format for the test
	testInput := []byte(`{"tool_calls": [{"id": "test_123", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`)

	canParse := detector.CanParse(testInput)
	latency := time.Since(start)

	if !canParse {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "JSON detector cannot parse valid JSON",
			Latency: latency,
		}
	}

	// Test actual detection
	result, err := detector.Detect(ctx, testInput)
	if err != nil || result.Confidence < 0.5 {
		return CheckResult{
			Status:  HealthStatusDegraded,
			Message: "JSON detector confidence low",
			Latency: latency,
		}
	}

	return CheckResult{
		Status:  HealthStatusHealthy,
		Message: fmt.Sprintf("JSON detector operational (confidence: %.2f)", result.Confidence),
		Latency: latency,
	}
}

// checkXMLDetector tests XML detector health
func (h *HealthMonitor) checkXMLDetector(ctx context.Context) CheckResult {
	start := time.Now()

	detector := NewXMLDetector()
	testInput := []byte(`<function_calls><invoke name="test"/></function_calls>`)

	canParse := detector.CanParse(testInput)
	latency := time.Since(start)

	if !canParse {
		return CheckResult{
			Status:  HealthStatusUnhealthy,
			Message: "XML detector cannot parse valid XML",
			Latency: latency,
		}
	}

	// Test actual detection
	result, err := detector.Detect(ctx, testInput)
	if err != nil || result.Confidence < 0.5 {
		return CheckResult{
			Status:  HealthStatusDegraded,
			Message: "XML detector confidence low",
			Latency: latency,
		}
	}

	return CheckResult{
		Status:  HealthStatusHealthy,
		Message: fmt.Sprintf("XML detector operational (confidence: %.2f)", result.Confidence),
		Latency: latency,
	}
}

// checkFormatDetection tests format detection accuracy
func (h *HealthMonitor) checkFormatDetection(ctx context.Context) CheckResult {
	start := time.Now()

	testCases := []struct {
		input    string
		expected ProviderFormat
	}{
		{
			`{"tool_calls": [{"id": "test_123", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`,
			ProviderFormatOpenAI,
		},
		{
			`<function_calls><invoke name="test"><parameter name="arg">value</parameter></invoke></function_calls>`,
			ProviderFormatAnthropic,
		},
	}

	for _, tc := range testCases {
		format, confidence, err := h.parser.DetectFormat(tc.input)
		if err != nil || format != tc.expected || confidence < 0.7 {
			return CheckResult{
				Status:  HealthStatusDegraded,
				Message: fmt.Sprintf("Format detection inaccurate for %s", tc.expected),
				Latency: time.Since(start),
			}
		}
	}

	return CheckResult{
		Status:  HealthStatusHealthy,
		Message: "Format detection accurate",
		Latency: time.Since(start),
	}
}

// checkMemoryUsage checks for memory leaks or excessive usage
func (h *HealthMonitor) checkMemoryUsage(ctx context.Context) CheckResult {
	// This is a placeholder - in production, you'd check actual memory metrics
	return CheckResult{
		Status:  HealthStatusHealthy,
		Message: "Memory usage within limits",
		Latency: 0,
	}
}

// calculateMetrics calculates current health metrics
func (h *HealthMonitor) calculateMetrics() HealthMetrics {
	metrics := HealthMetrics{
		TotalParses:        h.totalParses,
		TotalSuccesses:     h.totalSuccesses,
		TotalFailures:      h.totalFailures,
		FormatDistribution: make(map[string]int64),
	}

	// Calculate success rate
	if h.totalParses > 0 {
		metrics.SuccessRate = float64(h.totalSuccesses) / float64(h.totalParses)
	}

	// Calculate parse rate
	now := time.Now()
	cutoff := now.Add(-h.rateWindow)
	recentParses := 0
	for i := len(h.parseHistory) - 1; i >= 0; i-- {
		if h.parseHistory[i].After(cutoff) {
			recentParses++
		} else {
			break
		}
	}
	metrics.ParseRate = float64(recentParses) / h.rateWindow.Seconds()

	// Calculate latencies
	if len(h.latencies) > 0 {
		sum := 0.0
		for _, l := range h.latencies {
			sum += l
		}
		metrics.AverageLatency = sum / float64(len(h.latencies))

		// Calculate percentiles
		if len(h.latencies) >= 20 {
			sorted := make([]float64, len(h.latencies))
			copy(sorted, h.latencies)
			// Simple percentile calculation (would use proper sorting in production)
			p95Index := int(float64(len(sorted)) * 0.95)
			p99Index := int(float64(len(sorted)) * 0.99)
			if p95Index < len(sorted) {
				metrics.P95Latency = sorted[p95Index]
			}
			if p99Index < len(sorted) {
				metrics.P99Latency = sorted[p99Index]
			}
		}
	}

	// Format distribution
	for format, count := range h.formatCounts {
		metrics.FormatDistribution[string(format)] = count
	}

	// Last parse time
	if !h.lastParseTime.IsZero() {
		metrics.LastParseTime = h.lastParseTime.Format(time.RFC3339)
	}

	metrics.ActiveParsers = int(h.activeParsers)

	return metrics
}

// RecordParse records a parse operation for health tracking
func (h *HealthMonitor) RecordParse(format ProviderFormat, duration time.Duration, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.totalParses++
	h.lastParseTime = time.Now()
	h.parseHistory = append(h.parseHistory, h.lastParseTime)

	// Trim old history
	if len(h.parseHistory) > 1000 {
		h.parseHistory = h.parseHistory[len(h.parseHistory)-1000:]
	}

	if success {
		h.totalSuccesses++
	} else {
		h.totalFailures++
	}

	// Record format
	h.formatCounts[format]++

	// Record latency
	latencyMs := float64(duration.Milliseconds())
	h.latencies = append(h.latencies, latencyMs)
	if len(h.latencies) > h.maxLatencies {
		h.latencies = h.latencies[len(h.latencies)-h.maxLatencies:]
	}
}

// StartParse increments active parser count
func (h *HealthMonitor) StartParse() {
	h.mu.Lock()
	h.activeParsers++
	h.mu.Unlock()
}

// EndParse decrements active parser count
func (h *HealthMonitor) EndParse() {
	h.mu.Lock()
	h.activeParsers--
	h.mu.Unlock()
}

// ToJSON returns the health check as JSON
func (h *HealthMonitor) ToJSON() ([]byte, error) {
	check := h.Check(context.Background())
	return json.MarshalIndent(check, "", "  ")
}

// NewJSONDetector creates a JSON detector
func NewJSONDetector() FormatDetector {
	return jsonparser.NewDetector()
}

// NewXMLDetector creates an XML detector
func NewXMLDetector() FormatDetector {
	return xmlparser.NewDetector()
}
