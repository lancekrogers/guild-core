// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"time"

	"github.com/lancekrogers/guild/pkg/observability"
)

// MetricsAdapter provides a compatibility layer for reasoning metrics
type MetricsAdapter struct {
	registry *observability.MetricsRegistry
}

// NewMetricsAdapter creates a new metrics adapter
func NewMetricsAdapter(registry *observability.MetricsRegistry) *MetricsAdapter {
	return &MetricsAdapter{
		registry: registry,
	}
}

// RecordCounter is a helper method to record counter metrics
func (ma *MetricsAdapter) RecordCounter(name string, value float64, labels ...string) {
	// Map to existing metrics API based on name
	switch name {
	case "reasoning_rate_limit_hit":
		if len(labels) >= 2 && labels[0] == "agent_id" {
			ma.registry.RecordError("RATE_LIMIT", "reasoning", labels[1])
		}
	case "reasoning_retry_attempts":
		ma.registry.RecordTaskRetry("reasoning_extraction")
	case "reasoning_dead_letter_entries":
		if len(labels) >= 2 && labels[0] == "error_code" {
			ma.registry.RecordError(labels[1], "reasoning", "extraction")
		}
	}
}

// RecordGauge is a helper method to record gauge metrics
func (ma *MetricsAdapter) RecordGauge(name string, value float64, labels ...string) {
	// Map to existing metrics API based on name
	switch name {
	case "reasoning_circuit_breaker_state":
		// No direct mapping, but we can track as utilization
		if len(labels) >= 2 && labels[0] == "provider" {
			// Map circuit breaker states: 0=closed (0%), 1=open (100%), 2=half-open (50%)
			utilization := value * 50.0 // 0->0%, 1->50%, 2->100%
			ma.registry.SetAgentUtilization("circuit_breaker", labels[1], utilization)
		}
	case "reasoning_rate_limiter_usage_ratio":
		if len(labels) >= 2 && labels[0] == "agent_id" {
			ma.registry.SetAgentUtilization(labels[1], "rate_limiter", value*100)
		}
	}
}

// RecordHistogram is a helper method to record histogram metrics
func (ma *MetricsAdapter) RecordHistogram(name string, value float64, labels ...string) {
	// Map to existing metrics API based on name
	switch name {
	case "reasoning_extraction_duration_seconds":
		if len(labels) >= 2 && labels[0] == "provider" {
			ma.registry.RecordProviderDuration(labels[1], "reasoning", time.Duration(value*float64(time.Second)))
		}
	}
}

// RegisterCounter is a no-op for compatibility
func (ma *MetricsAdapter) RegisterCounter(name, help string, labels []string) {
	// No-op - registration not needed with current API
}

// RegisterGauge is a no-op for compatibility
func (ma *MetricsAdapter) RegisterGauge(name, help string, labels []string) {
	// No-op - registration not needed with current API
}

// RegisterHistogram is a no-op for compatibility
func (ma *MetricsAdapter) RegisterHistogram(name, help string, labels []string, buckets []float64) {
	// No-op - registration not needed with current API
}

// Flush flushes any buffered metrics
func (ma *MetricsAdapter) Flush() {
	// No-op - current API doesn't buffer
}
