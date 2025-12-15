// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// MetricsCollector handles all reasoning system metrics
type MetricsCollector struct {
	registry *observability.MetricsRegistry
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(registry *observability.MetricsRegistry) (*MetricsCollector, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "metrics registry is required", nil).
			WithComponent("reasoning_metrics")
	}

	return &MetricsCollector{registry: registry}, nil
}

// RecordExtraction records metrics for an extraction
func (mc *MetricsCollector) RecordExtraction(ctx context.Context, provider, agentID string,
	duration time.Duration, blocks []ReasoningBlock, err error,
) {
	status := "success"
	if err != nil {
		status = "error"
	}

	// Record using existing observability API
	mc.registry.RecordProviderRequest(provider, "reasoning", status)
	mc.registry.RecordProviderDuration(provider, "reasoning", duration)

	// Record tokens
	totalTokens := 0
	for _, block := range blocks {
		totalTokens += block.TokenCount
	}
	if totalTokens > 0 {
		mc.registry.RecordProviderTokens(provider, "reasoning", "extracted", totalTokens)
	}

	// Record errors
	if err != nil {
		errorType := "unknown"
		if gerr, ok := err.(*gerror.GuildError); ok {
			errorType = string(gerr.Code)
		}
		mc.registry.RecordProviderError(provider, "reasoning", errorType)
	}
}

// RecordCircuitBreakerTrip records circuit breaker state changes
func (mc *MetricsCollector) RecordCircuitBreakerTrip(provider string, from, to CircuitState) {
	// Map to provider error tracking
	if to == StateOpen {
		mc.registry.RecordProviderError(provider, "reasoning", "circuit_breaker_open")
	}
}

// RecordRateLimitHit records rate limit hits
func (mc *MetricsCollector) RecordRateLimitHit(agentID string, limitType string) {
	// Use agent task recording with error status
	mc.registry.RecordAgentTask(agentID, "reasoning", "rate_limited")
}

// RecordRetry records retry attempts
func (mc *MetricsCollector) RecordRetry(attempt int, provider string) {
	mc.registry.RecordTaskRetry("reasoning_extraction")
}

// RecordDeadLetterEntry records entries added to dead letter queue
func (mc *MetricsCollector) RecordDeadLetterEntry(agentID string, errorCode string) {
	// Record as storage operation
	mc.registry.RecordStorageOperation("insert", "reasoning_dead_letter", "error")
	mc.registry.RecordStorageError("insert", "reasoning_dead_letter", errorCode)
}

// RecordStreamEvent records streaming events
func (mc *MetricsCollector) RecordStreamEvent(eventType string, agentID, provider string) {
	// Use agent task recording
	mc.registry.RecordAgentTask(agentID, "reasoning_stream", eventType)
}

// RecordBlockProcessed records processing of individual blocks
func (mc *MetricsCollector) RecordBlockProcessed(block ReasoningBlock, processingTime time.Duration) {
	// Record as task duration
	mc.registry.RecordTaskDuration("reasoning_block_"+block.Type, processingTime)
}

// UpdateCircuitBreakerState updates circuit breaker gauge
func (mc *MetricsCollector) UpdateCircuitBreakerState(provider string, state CircuitState) {
	// Map to agent utilization: closed=0%, open=100%, half-open=50%
	utilization := float64(state) * 50.0
	mc.registry.SetAgentUtilization("circuit_breaker_"+provider, "reasoning", utilization)
}

// UpdateRateLimiterUsage updates rate limiter usage ratio
func (mc *MetricsCollector) UpdateRateLimiterUsage(agentID string, ratio float64) {
	// Use agent utilization
	mc.registry.SetAgentUtilization(agentID, "rate_limiter", ratio*100)
}

// UpdateDeadLetterQueueSize updates dead letter queue size
func (mc *MetricsCollector) UpdateDeadLetterQueueSize(unprocessed, total int) {
	// Use task queue size
	mc.registry.SetTaskQueueSize("dead_letter_unprocessed", unprocessed)
	mc.registry.SetTaskQueueSize("dead_letter_total", total)
}

// UpdateActiveExtractions updates number of active extractions
func (mc *MetricsCollector) UpdateActiveExtractions(provider string, count int) {
	// Map to task queue size
	mc.registry.SetTaskQueueSize("reasoning_active_"+provider, count)
}
