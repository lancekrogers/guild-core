// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// MetricsCollector handles all reasoning system metrics
type MetricsCollector struct {
	registry *observability.MetricsRegistry

	// Counters
	extractionTotal     observability.Counter
	extractionErrors    observability.Counter
	tokensProcessed     observability.Counter
	retryAttempts       observability.Counter
	deadLetterEntries   observability.Counter
	circuitBreakerTrips observability.Counter
	rateLimitHits       observability.Counter
	streamEvents        observability.Counter

	// Histograms
	extractionDuration   observability.Histogram
	blockSize            observability.Histogram
	tokenCount           observability.Histogram
	retryDelay           observability.Histogram
	streamProcessingTime observability.Histogram

	// Gauges
	circuitBreakerState  observability.Gauge
	rateLimiterUsage     observability.Gauge
	deadLetterQueueSize  observability.Gauge
	activeExtractions    observability.Gauge
	reasoningBlocksInMem observability.Gauge
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(registry *observability.MetricsRegistry) (*MetricsCollector, error) {
	mc := &MetricsCollector{registry: registry}

	// Register counters
	mc.extractionTotal = registry.RegisterCounter(
		"reasoning_extraction_total",
		"Total number of reasoning extractions",
		[]string{"provider", "agent_id", "status"},
	)

	mc.extractionErrors = registry.RegisterCounter(
		"reasoning_extraction_errors_total",
		"Total number of extraction errors",
		[]string{"provider", "agent_id", "error_type"},
	)

	mc.tokensProcessed = registry.RegisterCounter(
		"reasoning_tokens_processed_total",
		"Total reasoning tokens processed",
		[]string{"provider", "agent_id", "block_type"},
	)

	mc.retryAttempts = registry.RegisterCounter(
		"reasoning_retry_attempts_total",
		"Total retry attempts",
		[]string{"attempt", "provider"},
	)

	mc.deadLetterEntries = registry.RegisterCounter(
		"reasoning_dead_letter_entries_total",
		"Total entries added to dead letter queue",
		[]string{"error_code"},
	)

	mc.circuitBreakerTrips = registry.RegisterCounter(
		"reasoning_circuit_breaker_trips_total",
		"Circuit breaker state changes",
		[]string{"from_state", "to_state"},
	)

	mc.rateLimitHits = registry.RegisterCounter(
		"reasoning_rate_limit_hits_total",
		"Rate limit hits",
		[]string{"agent_id", "limit_type"},
	)

	mc.streamEvents = registry.RegisterCounter(
		"reasoning_stream_events_total",
		"Stream events emitted",
		[]string{"type"},
	)

	// Register histograms
	mc.extractionDuration = registry.RegisterHistogram(
		"reasoning_extraction_duration_seconds",
		"Duration of reasoning extraction in seconds",
		[]string{"provider", "agent_id"},
		observability.DefaultBuckets,
	)

	mc.blockSize = registry.RegisterHistogram(
		"reasoning_block_size_bytes",
		"Size of reasoning blocks in bytes",
		[]string{"block_type"},
		[]float64{100, 500, 1000, 5000, 10000, 50000},
	)

	mc.tokenCount = registry.RegisterHistogram(
		"reasoning_block_tokens",
		"Token count per reasoning block",
		[]string{"block_type"},
		[]float64{10, 50, 100, 200, 500, 1000, 2000},
	)

	mc.retryDelay = registry.RegisterHistogram(
		"reasoning_retry_delay_seconds",
		"Delay between retry attempts",
		[]string{"attempt"},
		[]float64{0.1, 0.5, 1, 2, 5, 10},
	)

	mc.streamProcessingTime = registry.RegisterHistogram(
		"reasoning_stream_processing_seconds",
		"Time to process streaming chunks",
		[]string{"provider"},
		observability.DefaultBuckets,
	)

	// Register gauges
	mc.circuitBreakerState = registry.RegisterGauge(
		"reasoning_circuit_breaker_state",
		"Current state of circuit breaker (0=closed, 1=open, 2=half-open)",
		[]string{"provider"},
	)

	mc.rateLimiterUsage = registry.RegisterGauge(
		"reasoning_rate_limiter_usage_ratio",
		"Rate limiter usage as ratio of limit",
		[]string{"agent_id", "type"},
	)

	mc.deadLetterQueueSize = registry.RegisterGauge(
		"reasoning_dead_letter_queue_size",
		"Number of entries in dead letter queue",
		[]string{"status"},
	)

	mc.activeExtractions = registry.RegisterGauge(
		"reasoning_active_extractions",
		"Number of active extractions",
		[]string{"provider"},
	)

	mc.reasoningBlocksInMem = registry.RegisterGauge(
		"reasoning_blocks_in_memory",
		"Number of reasoning blocks in memory",
		[]string{"type"},
	)

	return mc, nil
}

// RecordExtraction records metrics for an extraction
func (mc *MetricsCollector) RecordExtraction(
	ctx context.Context,
	provider, agentID string,
	duration time.Duration,
	blocks []ReasoningBlock,
	err error,
) {
	labels := map[string]string{
		"provider": provider,
		"agent_id": agentID,
	}

	// Record attempt
	statusLabel := map[string]string{
		"provider": provider,
		"agent_id": agentID,
		"status":   "success",
	}
	if err != nil {
		statusLabel["status"] = "error"
	}
	mc.extractionTotal.Inc(statusLabel)

	// Record duration
	mc.extractionDuration.Observe(duration.Seconds(), labels)

	if err != nil {
		// Record error
		errorType := "unknown"
		if gerr, ok := err.(*gerror.Error); ok {
			errorType = gerr.Code
		}
		mc.extractionErrors.Inc(map[string]string{
			"provider":   provider,
			"agent_id":   agentID,
			"error_type": errorType,
		})
	} else {
		// Record block metrics
		for _, block := range blocks {
			// Token count
			mc.tokensProcessed.Add(float64(block.TokenCount), map[string]string{
				"provider":   provider,
				"agent_id":   agentID,
				"block_type": block.Type,
			})

			// Block size
			mc.blockSize.Observe(float64(len(block.Content)), map[string]string{
				"block_type": block.Type,
			})

			// Token histogram
			mc.tokenCount.Observe(float64(block.TokenCount), map[string]string{
				"block_type": block.Type,
			})
		}
	}
}

// RecordCircuitBreakerStateChange records circuit breaker state changes
func (mc *MetricsCollector) RecordCircuitBreakerStateChange(from, to CircuitState) {
	mc.circuitBreakerTrips.Inc(map[string]string{
		"from_state": from.String(),
		"to_state":   to.String(),
	})
}

// UpdateCircuitBreakerState updates the current state gauge
func (mc *MetricsCollector) UpdateCircuitBreakerState(provider string, state CircuitState) {
	mc.circuitBreakerState.Set(float64(state), map[string]string{
		"provider": provider,
	})
}

// RecordRateLimitHit records a rate limit hit
func (mc *MetricsCollector) RecordRateLimitHit(agentID, limitType string) {
	mc.rateLimitHits.Inc(map[string]string{
		"agent_id":   agentID,
		"limit_type": limitType,
	})
}

// UpdateRateLimiterUsage updates rate limiter usage gauges
func (mc *MetricsCollector) UpdateRateLimiterUsage(usage map[string]float64) {
	for agentID, ratio := range usage {
		limitType := "agent"
		if agentID == "global" {
			limitType = "global"
		}
		mc.rateLimiterUsage.Set(ratio, map[string]string{
			"agent_id": agentID,
			"type":     limitType,
		})
	}
}

// RecordRetryAttempt records a retry attempt
func (mc *MetricsCollector) RecordRetryAttempt(attempt int, provider string, delay time.Duration) {
	mc.retryAttempts.Inc(map[string]string{
		"attempt":  fmt.Sprintf("%d", attempt),
		"provider": provider,
	})

	mc.retryDelay.Observe(delay.Seconds(), map[string]string{
		"attempt": fmt.Sprintf("%d", attempt),
	})
}

// RecordDeadLetterEntry records an entry added to dead letter queue
func (mc *MetricsCollector) RecordDeadLetterEntry(errorCode string) {
	mc.deadLetterEntries.Inc(map[string]string{
		"error_code": errorCode,
	})
}

// UpdateDeadLetterQueueSize updates the queue size gauge
func (mc *MetricsCollector) UpdateDeadLetterQueueSize(total, unprocessed int) {
	mc.deadLetterQueueSize.Set(float64(total), map[string]string{
		"status": "total",
	})
	mc.deadLetterQueueSize.Set(float64(unprocessed), map[string]string{
		"status": "unprocessed",
	})
}

// RecordStreamEvent records a stream event
func (mc *MetricsCollector) RecordStreamEvent(eventType string) {
	mc.streamEvents.Inc(map[string]string{
		"type": eventType,
	})
}

// RecordStreamProcessing records stream processing time
func (mc *MetricsCollector) RecordStreamProcessing(provider string, duration time.Duration) {
	mc.streamProcessingTime.Observe(duration.Seconds(), map[string]string{
		"provider": provider,
	})
}

// UpdateActiveExtractions updates the active extractions gauge
func (mc *MetricsCollector) UpdateActiveExtractions(provider string, count int) {
	mc.activeExtractions.Set(float64(count), map[string]string{
		"provider": provider,
	})
}

// UpdateReasoningBlocksInMemory updates the blocks in memory gauge
func (mc *MetricsCollector) UpdateReasoningBlocksInMemory(blockType string, count int) {
	mc.reasoningBlocksInMem.Set(float64(count), map[string]string{
		"type": blockType,
	})
}

// GetRegistry returns the underlying metrics registry
func (mc *MetricsCollector) GetRegistry() *observability.MetricsRegistry {
	return mc.registry
}
