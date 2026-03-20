// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// MetricsWrapper wraps MetricsRegistry to provide simple counter/gauge methods
type MetricsWrapper struct {
	registry *observability.MetricsRegistry
}

// NewMetricsWrapper creates a new metrics wrapper
func NewMetricsWrapper(registry *observability.MetricsRegistry) *MetricsWrapper {
	return &MetricsWrapper{registry: registry}
}

// RecordCounter records a counter metric
func (m *MetricsWrapper) RecordCounter(name string, value float64, labels ...string) {
	// Convert to appropriate MetricsRegistry method
	// Since we don't have direct counter recording, use the most appropriate method
	if name == "reasoning_rate_limit_hit" {
		m.registry.RecordTaskRetry("reasoning_rate_limited")
	} else if name == "reasoning_retry_attempts" {
		m.registry.RecordTaskRetry("reasoning_extraction")
	} else if name == "reasoning_dead_letter_entries" {
		m.registry.RecordStorageError("insert", "reasoning_dead_letter", "failed")
	} else if name == "reasoning_dead_letter_cleaned" {
		m.registry.RecordStorageOperation("delete", "reasoning_dead_letter", "success")
	}
}

// RecordGauge records a gauge metric
func (m *MetricsWrapper) RecordGauge(name string, value float64, labels ...string) {
	// Convert to appropriate MetricsRegistry method
	if name == "reasoning_circuit_breaker_state" {
		// Map to agent utilization
		m.registry.SetAgentUtilization("circuit_breaker", "reasoning", value*50.0)
	} else if name == "reasoning_rate_limiter_usage_ratio" {
		if len(labels) >= 2 {
			agentID := labels[1] // Assuming agent_id is second label
			m.registry.SetAgentUtilization(agentID, "rate_limiter", value*100)
		}
	}
}

// Flush flushes any buffered metrics
func (m *MetricsWrapper) Flush() {
	// MetricsRegistry doesn't have a flush method
}
