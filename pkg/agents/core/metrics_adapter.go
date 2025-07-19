// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"time"

	"github.com/lancekrogers/guild/pkg/observability"
)

// MetricsAdapter adapts the observability.MetricsRegistry to provide
// the methods expected by the agents/core package
type MetricsAdapter struct {
	registry *observability.MetricsRegistry
}

// NewMetricsAdapter creates a new metrics adapter
func NewMetricsAdapter(registry *observability.MetricsRegistry) *MetricsAdapter {
	return &MetricsAdapter{registry: registry}
}

// RecordCounter records a counter metric
func (m *MetricsAdapter) RecordCounter(name string, value float64, labels ...string) {
	// Map to appropriate observability methods
	switch name {
	case "reasoning_stream_errors":
		m.registry.RecordError("reasoning_stream", "agents_core", "stream_error")
	case "reasoning_stream_events":
		if len(labels) >= 2 && labels[0] == "type" {
			m.registry.RecordAgentTask("reasoning_stream", "event", labels[1])
		}
	default:
		// Use task retry as a generic counter
		m.registry.RecordTaskRetry(name)
	}
}

// RecordHistogram records a histogram metric
func (m *MetricsAdapter) RecordHistogram(name string, value float64, labels ...string) {
	// Map to duration metrics
	if name == "thinking_block_parsing_seconds" {
		m.registry.RecordTaskDuration("thinking_block_parsing", time.Duration(value*float64(time.Second)))
	}
}

// RecordGauge records a gauge metric
func (m *MetricsAdapter) RecordGauge(name string, value float64, labels ...string) {
	// Map to appropriate gauge-like methods
	if name == "thinking_blocks_parsed" {
		m.registry.SetTaskQueueSize("thinking_blocks", int(value))
	}
}