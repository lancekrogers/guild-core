// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"golang.org/x/time/rate"
)

// Registry provides centralized reasoning extraction with resilience patterns
type Registry struct {
	// Core components
	extractor      *Extractor
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
	retryer        *Retryer
	deadLetter     *DeadLetterQueue
	eventBus       events.EventBus
	metrics        *MetricsWrapper
	logger         *slog.Logger
	db             *sql.DB

	// State management
	mu          sync.RWMutex
	started     bool
	stopping    bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
	healthCheck *HealthChecker

	// Configuration
	config Config
}

// Config holds configuration for the reasoning registry
type Config struct {
	// Circuit breaker settings
	CircuitBreakerFailureThreshold int           `json:"circuit_breaker_failure_threshold"`
	CircuitBreakerSuccessThreshold int           `json:"circuit_breaker_success_threshold"`
	CircuitBreakerTimeout          time.Duration `json:"circuit_breaker_timeout"`

	// Rate limiter settings
	GlobalRateLimit   int `json:"global_rate_limit"`
	GlobalBurst       int `json:"global_burst"`
	PerAgentRateLimit int `json:"per_agent_rate_limit"`
	PerAgentBurst     int `json:"per_agent_burst"`

	// Performance settings
	MaxWorkers      int           `json:"max_workers"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerSuccessThreshold: 2,
		CircuitBreakerTimeout:          30 * time.Second,
		GlobalRateLimit:                1000,
		GlobalBurst:                    100,
		PerAgentRateLimit:              100,
		PerAgentBurst:                  10,
		MaxWorkers:                     4,
		CleanupInterval:                time.Hour,
		MetricsInterval:                time.Minute,
	}
}

// NewRegistry creates a new reasoning registry with all dependencies
func NewRegistry(
	extractor *Extractor,
	eventBus events.EventBus,
	metrics *observability.MetricsRegistry,
	logger *slog.Logger,
	db *sql.DB,
	config Config,
) (*Registry, error) {
	// Validate dependencies
	if extractor == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "extractor is required", nil).
			WithComponent("reasoning_registry")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "event bus is required", nil).
			WithComponent("reasoning_registry")
	}
	if metrics == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "metrics registry is required", nil).
			WithComponent("reasoning_registry")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "logger is required", nil).
			WithComponent("reasoning_registry")
	}
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("reasoning_registry")
	}

	// Create metrics wrapper
	metricsWrapper := NewMetricsWrapper(metrics)

	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: config.CircuitBreakerFailureThreshold,
		SuccessThreshold: config.CircuitBreakerSuccessThreshold,
		Timeout:          config.CircuitBreakerTimeout,
		OnStateChange: func(from, to CircuitState) {
			logger.Info("circuit breaker state changed",
				"from", from.String(),
				"to", to.String())

			// Emit event for state change
			eventBus.Publish(context.Background(), &CircuitBreakerStateChangeEvent{
				BaseEvent: *events.NewBaseEvent(
					uuid.New().String(),
					EventCircuitBreakerStateChange,
					"reasoning-registry",
					map[string]interface{}{"provider": "default"},
				),
				From:      from,
				To:        to,
				Timestamp: time.Now(),
			})
		},
	})

	// Create rate limiter
	rateLimiter := NewRateLimiter(RateLimiterConfig{
		GlobalRate:  rate.Limit(config.GlobalRateLimit),
		GlobalBurst: config.GlobalBurst,
		AgentRate:   rate.Limit(config.PerAgentRateLimit),
		AgentBurst:  config.PerAgentBurst,
		OnLimitHit: func(agentID string, limitType string) {
			metricsWrapper.RecordCounter("reasoning_rate_limit_hit", 1,
				"agent_id", agentID,
				"limit_type", limitType)
		},
	})

	// Create retryer
	retryConfig := DefaultRetryConfig()
	retryConfig.OnRetry = func(attempt int, err error, delay time.Duration) {
		logger.Warn("retrying reasoning extraction",
			"attempt", attempt,
			"error", err,
			"delay_ms", delay.Milliseconds())

		metricsWrapper.RecordCounter("reasoning_retry_attempts", 1,
			"attempt", fmt.Sprintf("%d", attempt))
	}
	retryer := NewRetryer(retryConfig)

	// Create dead letter queue
	deadLetter := NewDeadLetterQueue(db, 1000)
	deadLetter.SetOnChange(func(entry DeadLetterEntry) {
		logger.Error("reasoning extraction failed, added to dead letter queue",
			"entry_id", entry.ID,
			"agent_id", entry.AgentID,
			"error", entry.Error,
			"attempts", entry.Attempts)

		metricsWrapper.RecordCounter("reasoning_dead_letter_entries", 1,
			"error_code", entry.ErrorCode)
	})

	// Create health checker
	healthChecker := NewHealthChecker(extractor, circuitBreaker, rateLimiter, db)

	return &Registry{
		extractor:      extractor,
		circuitBreaker: circuitBreaker,
		rateLimiter:    rateLimiter,
		retryer:        retryer,
		deadLetter:     deadLetter,
		eventBus:       eventBus,
		metrics:        metricsWrapper,
		logger:         logger,
		db:             db,
		config:         config,
		stopCh:         make(chan struct{}),
		healthCheck:    healthChecker,
	}, nil
}

// Start initializes and starts the registry
func (r *Registry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "reasoning registry already started", nil).
			WithComponent("reasoning_registry")
	}

	r.logger.Info("starting reasoning registry")

	// Initialize metrics
	if err := r.initializeMetrics(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize metrics").
			WithComponent("reasoning_registry")
	}

	// Load circuit breaker state from database
	if err := r.loadCircuitBreakerState(ctx); err != nil {
		r.logger.Warn("failed to load circuit breaker state",
			"error", err)
		// Non-fatal, continue with default state
	}

	// Start background workers
	r.wg.Add(4)
	go r.metricsCollector()
	go r.rateLimiterCleaner()
	go r.circuitBreakerPersister()
	go r.deadLetterCleaner()

	// No longer using OnExtraction callback - events are published directly in Extract method

	r.started = true
	r.logger.Info("reasoning registry started successfully")

	return nil
}

// Stop gracefully shuts down the registry
func (r *Registry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return gerror.New(gerror.ErrCodeInvalidTransition, "reasoning registry not started", nil).
			WithComponent("reasoning_registry")
	}

	if r.stopping {
		return gerror.New(gerror.ErrCodeAlreadyExists, "reasoning registry already stopping", nil).
			WithComponent("reasoning_registry")
	}

	r.stopping = true
	r.logger.Info("stopping reasoning registry")

	// Signal shutdown to workers
	close(r.stopCh)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("all workers stopped")
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout waiting for workers").
			WithComponent("reasoning_registry")
	}

	// Save circuit breaker state
	if err := r.saveCircuitBreakerState(context.Background()); err != nil {
		r.logger.Warn("failed to save circuit breaker state",
			"error", err)
		// Non-fatal, continue shutdown
	}

	// Flush metrics
	r.metrics.Flush()

	r.started = false
	r.logger.Info("reasoning registry stopped successfully")

	return nil
}

// Health returns the health status of all components
func (r *Registry) Health(ctx context.Context) HealthReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.started {
		return HealthReport{
			Status:  HealthStatusUnhealthy,
			Message: "reasoning registry not started",
			Checks: map[string]CheckResult{
				"registry": {
					Status:    HealthStatusUnhealthy,
					Message:   "not started",
					Timestamp: time.Now(),
				},
			},
			Timestamp: time.Now(),
		}
	}

	// Perform comprehensive health check
	health := r.healthCheck.Check(ctx)

	// Return the health report directly
	return health
}

// Extract performs reasoning extraction with full protection
func (r *Registry) Extract(ctx context.Context, agentID, content string) ([]ReasoningBlock, error) {
	start := time.Now()

	// Check if started
	r.mu.RLock()
	if !r.started {
		r.mu.RUnlock()
		return nil, gerror.New(gerror.ErrCodeInvalidTransition, "reasoning registry not started", nil).
			WithComponent("reasoning_registry")
	}
	r.mu.RUnlock()

	// Rate limiting
	if err := r.rateLimiter.Wait(ctx, agentID); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeResourceExhausted, "rate limit exceeded").
			WithDetails("agent_id", agentID).
			WithComponent("reasoning_registry")
	}

	// Execute with retry and circuit breaker protection
	var blocks []ReasoningBlock
	err := r.retryer.Execute(ctx, func() error {
		// Circuit breaker protection
		return r.circuitBreaker.Execute(ctx, func() error {
			var extractErr error
			blocks, extractErr = r.extractor.Extract(ctx, content)
			return extractErr
		})
	})
	if err != nil {
		// Add to dead letter queue on final failure
		metadata := map[string]interface{}{
			"content_length": len(content),
			"provider":       "default",
		}

		if dlqErr := r.deadLetter.Add(ctx, agentID, content, err, r.retryer.config.MaxAttempts, metadata); dlqErr != nil {
			r.logger.Error("failed to add to dead letter queue",
				"error", dlqErr,
				"original_error", err,
				"agent_id", agentID)
		}

		// Publish failure event
		r.eventBus.Publish(ctx, &ReasoningFailedEvent{
			BaseEvent: *events.NewBaseEvent(
				uuid.New().String(),
				EventReasoningFailed,
				"reasoning-registry",
				map[string]interface{}{"provider": "default"},
			),
			AgentID:   agentID,
			Error:     err,
			Timestamp: time.Now(),
		})

		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "extraction failed after retries").
			WithDetails("agent_id", agentID).
			WithDetails("attempts", r.retryer.config.MaxAttempts).
			WithComponent("reasoning_registry")
	}

	// Publish success event
	r.eventBus.Publish(ctx, &ReasoningExtractedEvent{
		BaseEvent: *events.NewBaseEvent(
			uuid.New().String(),
			EventReasoningExtracted,
			"reasoning-registry",
			map[string]interface{}{"provider": "default"},
		),
		AgentID:         agentID,
		Blocks:          blocks,
		TokensExtracted: calculateTotalTokens(blocks),
		Duration:        time.Since(start),
		Timestamp:       time.Now(),
	})

	return blocks, nil
}

// ExtractStream performs streaming reasoning extraction
func (r *Registry) ExtractStream(ctx context.Context, agentID string, reader io.Reader) (<-chan ReasoningBlock, <-chan error) {
	blockCh := make(chan ReasoningBlock, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(blockCh)
		defer close(errCh)

		// Check if started
		r.mu.RLock()
		if !r.started {
			r.mu.RUnlock()
			errCh <- gerror.New(gerror.ErrCodeInvalidTransition, "reasoning registry not started", nil).
				WithComponent("reasoning_registry")
			return
		}
		r.mu.RUnlock()

		// Rate limiting
		if err := r.rateLimiter.Wait(ctx, agentID); err != nil {
			errCh <- gerror.Wrap(err, gerror.ErrCodeResourceExhausted, "rate limit exceeded").
				WithDetails("agent_id", agentID).
				WithComponent("reasoning_registry")
			return
		}

		// Stream with circuit breaker protection
		err := r.circuitBreaker.Execute(ctx, func() error {
			streamCh, streamErrCh := r.extractor.ExtractStream(ctx, reader)

			for {
				select {
				case block, ok := <-streamCh:
					if !ok {
						return nil
					}
					select {
					case blockCh <- block:
					case <-ctx.Done():
						return ctx.Err()
					}
				case err := <-streamErrCh:
					if err != nil {
						return err
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
		if err != nil {
			errCh <- gerror.Wrap(err, gerror.ErrCodeInternal, "stream extraction failed").
				WithDetails("agent_id", agentID).
				WithComponent("reasoning_registry")
		}
	}()

	return blockCh, errCh
}

// initializeMetrics registers all metrics with the registry
func (r *Registry) initializeMetrics() error {
	// Metrics are registered via the adapter's Record* methods
	// No explicit registration needed with current observability API
	return nil
}

// Background workers

func (r *Registry) metricsCollector() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.collectMetrics()
		case <-r.stopCh:
			return
		}
	}
}

func (r *Registry) rateLimiterCleaner() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.rateLimiter.Cleanup(); err != nil {
				r.logger.Warn("rate limiter cleanup failed",
					"error", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

func (r *Registry) circuitBreakerPersister() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.saveCircuitBreakerState(context.Background()); err != nil {
				r.logger.Warn("failed to persist circuit breaker state",
					"error", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

func (r *Registry) deadLetterCleaner() {
	defer r.wg.Done()

	// Clean dead letter queue daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()

			// Clean entries older than 30 days
			cleaned, err := r.deadLetter.Clean(ctx, 30*24*time.Hour)
			if err != nil {
				r.logger.Warn("failed to clean dead letter queue",
					"error", err)
			} else if cleaned > 0 {
				r.logger.Info("cleaned dead letter queue",
					"entries_removed", cleaned)
				r.metrics.RecordCounter("reasoning_dead_letter_cleaned", float64(cleaned))
			}

			// Log statistics
			stats, err := r.deadLetter.Statistics(ctx)
			if err != nil {
				r.logger.Warn("failed to get dead letter statistics",
					"error", err)
			} else {
				r.logger.Info("dead letter queue statistics",
					"total_entries", stats["total_entries"],
					"unprocessed_entries", stats["unprocessed_entries"])
			}
		case <-r.stopCh:
			return
		}
	}
}

func (r *Registry) collectMetrics() {
	// Circuit breaker state
	state := r.circuitBreaker.State()
	r.metrics.RecordGauge("reasoning_circuit_breaker_state", float64(state),
		"provider", "default")

	// Rate limiter usage
	usage := r.rateLimiter.GetUsageStats()
	for agentID, ratio := range usage {
		r.metrics.RecordGauge("reasoning_rate_limiter_usage_ratio", ratio,
			"agent_id", agentID,
			"type", "tokens")
	}
}

// Database operations

func (r *Registry) loadCircuitBreakerState(ctx context.Context) error {
	query := `
		SELECT state, failures, last_failure_time, last_success_time
		FROM reasoning_circuit_breaker
		WHERE provider = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var state int
	var failures int
	var lastFailure, lastSuccess sql.NullTime

	err := r.db.QueryRowContext(ctx, query, "default").Scan(
		&state, &failures, &lastFailure, &lastSuccess)
	if err == sql.ErrNoRows {
		// No saved state, use defaults
		return nil
	}
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to query circuit breaker state").
			WithComponent("reasoning_registry")
	}

	// Restore state
	r.circuitBreaker.RestoreState(CircuitState(state), failures,
		lastFailure.Time, lastSuccess.Time)

	return nil
}

func (r *Registry) saveCircuitBreakerState(ctx context.Context) error {
	state, failures, lastFailure, lastSuccess := r.circuitBreaker.GetState()

	query := `
		INSERT INTO reasoning_circuit_breaker (
			id, provider, state, failures, 
			last_failure_time, last_success_time,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		) ON CONFLICT (provider) DO UPDATE SET
			state = EXCLUDED.state,
			failures = EXCLUDED.failures,
			last_failure_time = EXCLUDED.last_failure_time,
			last_success_time = EXCLUDED.last_success_time,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query,
		generateID(), "default", int(state), failures,
		toNullTime(lastFailure), toNullTime(lastSuccess))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to save circuit breaker state").
			WithComponent("reasoning_registry")
	}

	return nil
}

// Helper functions

func calculateTotalTokens(blocks []ReasoningBlock) int {
	total := 0
	for _, block := range blocks {
		total += block.TokenCount
	}
	return total
}

func toNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

func generateID() string {
	return uuid.New().String()
}
