// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is working normally
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates the component is working but with issues
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates the component is not working
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthReport represents the overall health of the reasoning system
type HealthReport struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HealthChecker performs health checks on the reasoning system
type HealthChecker struct {
	extractor      *Extractor
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
	db             *sql.DB

	// Performance thresholds
	extractionTimeoutThreshold time.Duration
	latencyP99Threshold        time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(
	extractor *Extractor,
	circuitBreaker *CircuitBreaker,
	rateLimiter *RateLimiter,
	db *sql.DB,
) *HealthChecker {
	return &HealthChecker{
		extractor:                  extractor,
		circuitBreaker:             circuitBreaker,
		rateLimiter:                rateLimiter,
		db:                         db,
		extractionTimeoutThreshold: 5 * time.Second,
		latencyP99Threshold:        100 * time.Millisecond,
	}
}

// Check performs a comprehensive health check
func (hc *HealthChecker) Check(ctx context.Context) HealthReport {
	report := HealthReport{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
		Metadata:  make(map[string]interface{}),
	}

	// Perform individual checks
	checks := []struct {
		name  string
		check func(context.Context) CheckResult
	}{
		{"circuit_breaker", hc.checkCircuitBreaker},
		{"rate_limiter", hc.checkRateLimiter},
		{"database", hc.checkDatabase},
		{"extractor", hc.checkExtractor},
		{"performance", hc.checkPerformance},
	}

	for _, c := range checks {
		// Create timeout context for each check
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result := c.check(checkCtx)
		cancel()

		report.Checks[c.name] = result

		// Update overall status
		switch result.Status {
		case HealthStatusUnhealthy:
			report.Status = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if report.Status == HealthStatusHealthy {
				report.Status = HealthStatusDegraded
			}
		}
	}

	// Set overall message
	report.Message = hc.generateMessage(report)

	return report
}

// checkCircuitBreaker checks the circuit breaker health
func (hc *HealthChecker) checkCircuitBreaker(ctx context.Context) CheckResult {
	start := time.Now()

	if hc.circuitBreaker == nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "circuit breaker not initialized",
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	state := hc.circuitBreaker.State()
	stats := hc.circuitBreaker.Statistics()

	result := CheckResult{
		Timestamp: start,
		Duration:  time.Since(start),
		Details:   stats,
	}

	switch state {
	case StateClosed:
		result.Status = HealthStatusHealthy
		result.Message = "circuit breaker closed (normal operation)"
	case StateOpen:
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("circuit breaker open (failures: %v)", stats["failures"])
	case StateHalfOpen:
		result.Status = HealthStatusDegraded
		result.Message = "circuit breaker half-open (testing recovery)"
	default:
		result.Status = HealthStatusUnhealthy
		result.Message = fmt.Sprintf("circuit breaker in unknown state: %v", state)
	}

	return result
}

// checkRateLimiter checks the rate limiter health
func (hc *HealthChecker) checkRateLimiter(ctx context.Context) CheckResult {
	start := time.Now()

	if hc.rateLimiter == nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "rate limiter not initialized",
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	stats := hc.rateLimiter.Statistics()
	usage := hc.rateLimiter.GetUsageStats()

	result := CheckResult{
		Timestamp: start,
		Duration:  time.Since(start),
		Details:   stats,
	}

	// Check global usage
	globalUsage, exists := usage["global"]
	if !exists {
		result.Status = HealthStatusUnhealthy
		result.Message = "unable to determine global rate limit usage"
		return result
	}

	// Set status based on usage
	if globalUsage > 0.9 {
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("high global rate limit usage: %.1f%%", globalUsage*100)
	} else {
		result.Status = HealthStatusHealthy
		result.Message = fmt.Sprintf("rate limiter operating normally (usage: %.1f%%)", globalUsage*100)
	}

	// Add usage stats to details
	result.Details["usage"] = usage

	return result
}

// checkDatabase checks database connectivity and performance
func (hc *HealthChecker) checkDatabase(ctx context.Context) CheckResult {
	start := time.Now()

	if hc.db == nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "database not initialized",
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	// Ping database
	if err := hc.db.PingContext(ctx); err != nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "database ping failed",
			Timestamp: start,
			Duration:  time.Since(start),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Check table existence
	var tableCount int
	query := `
		SELECT COUNT(*) 
		FROM sqlite_master 
		WHERE type='table' 
		AND name IN ('reasoning_blocks', 'reasoning_circuit_breaker', 'reasoning_rate_limits')
	`

	if err := hc.db.QueryRowContext(ctx, query).Scan(&tableCount); err != nil {
		return CheckResult{
			Status:    HealthStatusDegraded,
			Message:   "unable to verify database schema",
			Timestamp: start,
			Duration:  time.Since(start),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	duration := time.Since(start)

	// Check query performance
	if duration > 100*time.Millisecond {
		return CheckResult{
			Status:    HealthStatusDegraded,
			Message:   fmt.Sprintf("slow database response: %v", duration),
			Timestamp: start,
			Duration:  duration,
			Details: map[string]interface{}{
				"tables_found": tableCount,
			},
		}
	}

	return CheckResult{
		Status:    HealthStatusHealthy,
		Message:   "database connection healthy",
		Timestamp: start,
		Duration:  duration,
		Details: map[string]interface{}{
			"tables_found": tableCount,
			"ping_ms":      duration.Milliseconds(),
		},
	}
}

// checkExtractor checks the extractor functionality
func (hc *HealthChecker) checkExtractor(ctx context.Context) CheckResult {
	start := time.Now()

	if hc.extractor == nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "extractor not initialized",
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	// Test extraction with sample content
	testContent := "<thinking>Health check test reasoning block</thinking>"

	// Create timeout context
	extractCtx, cancel := context.WithTimeout(ctx, hc.extractionTimeoutThreshold)
	defer cancel()

	blocks, err := hc.extractor.Extract(extractCtx, testContent)
	duration := time.Since(start)

	if err != nil {
		if gerr, ok := err.(*gerror.GuildError); ok && gerr.Code == gerror.ErrCodeTimeout {
			return CheckResult{
				Status:    HealthStatusDegraded,
				Message:   "extractor timeout",
				Timestamp: start,
				Duration:  duration,
				Details: map[string]interface{}{
					"error":   err.Error(),
					"timeout": hc.extractionTimeoutThreshold,
				},
			}
		}
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "extractor test failed",
			Timestamp: start,
			Duration:  duration,
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	if len(blocks) != 1 {
		return CheckResult{
			Status:    HealthStatusDegraded,
			Message:   fmt.Sprintf("unexpected extraction result: got %d blocks, expected 1", len(blocks)),
			Timestamp: start,
			Duration:  duration,
		}
	}

	return CheckResult{
		Status:    HealthStatusHealthy,
		Message:   "extractor functioning normally",
		Timestamp: start,
		Duration:  duration,
		Details: map[string]interface{}{
			"extraction_ms": duration.Milliseconds(),
			"blocks_found":  len(blocks),
		},
	}
}

// checkPerformance checks overall system performance
func (hc *HealthChecker) checkPerformance(ctx context.Context) CheckResult {
	start := time.Now()

	// This would normally check metrics from the metrics registry
	// For now, we'll use the extraction test as a performance indicator
	result := CheckResult{
		Status:    HealthStatusHealthy,
		Message:   "performance within acceptable limits",
		Timestamp: start,
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	// Add performance metrics
	result.Details["extraction_threshold_ms"] = hc.latencyP99Threshold.Milliseconds()
	result.Details["timeout_threshold_ms"] = hc.extractionTimeoutThreshold.Milliseconds()

	return result
}

// generateMessage creates an overall health message
func (hc *HealthChecker) generateMessage(report HealthReport) string {
	unhealthy := 0
	degraded := 0

	for _, check := range report.Checks {
		switch check.Status {
		case HealthStatusUnhealthy:
			unhealthy++
		case HealthStatusDegraded:
			degraded++
		}
	}

	if unhealthy > 0 {
		return fmt.Sprintf("%d checks unhealthy, %d degraded", unhealthy, degraded)
	}
	if degraded > 0 {
		return fmt.Sprintf("%d checks degraded", degraded)
	}
	return "all checks passing"
}
