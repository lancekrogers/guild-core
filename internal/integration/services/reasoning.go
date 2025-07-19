// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/lancekrogers/guild/pkg/component"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/reasoning"
	"github.com/lancekrogers/guild/pkg/registry"
)

// ReasoningService provides reasoning extraction and analysis capabilities
type ReasoningService struct {
	BaseService
	registry *reasoning.Registry
}

// ReasoningServiceConfig holds configuration for the reasoning service
type ReasoningServiceConfig struct {
	// Circuit breaker settings
	CircuitBreakerFailureThreshold int `json:"circuit_breaker_failure_threshold"`
	CircuitBreakerSuccessThreshold int `json:"circuit_breaker_success_threshold"`
	CircuitBreakerTimeout          int `json:"circuit_breaker_timeout_seconds"`

	// Rate limiter settings
	GlobalRateLimit   int `json:"global_rate_limit"`
	GlobalBurst       int `json:"global_burst"`
	PerAgentRateLimit int `json:"per_agent_rate_limit"`
	PerAgentBurst     int `json:"per_agent_burst"`

	// Performance settings
	MaxWorkers      int `json:"max_workers"`
	CleanupInterval int `json:"cleanup_interval_seconds"`
	MetricsInterval int `json:"metrics_interval_seconds"`
}

// DefaultReasoningServiceConfig returns default configuration
func DefaultReasoningServiceConfig() ReasoningServiceConfig {
	return ReasoningServiceConfig{
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerSuccessThreshold: 2,
		CircuitBreakerTimeout:          30,
		GlobalRateLimit:                1000,
		GlobalBurst:                    100,
		PerAgentRateLimit:              100,
		PerAgentBurst:                  10,
		MaxWorkers:                     4,
		CleanupInterval:                3600, // 1 hour
		MetricsInterval:                60,   // 1 minute
	}
}

// NewReasoningService creates a new reasoning service
func NewReasoningService(
	componentRegistry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	db *sql.DB,
	config ReasoningServiceConfig,
) (*ReasoningService, error) {
	// Validate inputs
	if componentRegistry == nil {
		return nil, gerror.New("component registry is required").
			WithCode(gerror.ErrCodeInvalidArgument).
			WithComponent("reasoning_service")
	}
	if eventBus == nil {
		return nil, gerror.New("event bus is required").
			WithCode(gerror.ErrCodeInvalidArgument).
			WithComponent("reasoning_service")
	}
	if logger == nil {
		return nil, gerror.New("logger is required").
			WithCode(gerror.ErrCodeInvalidArgument).
			WithComponent("reasoning_service")
	}
	if db == nil {
		return nil, gerror.New("database is required").
			WithCode(gerror.ErrCodeInvalidArgument).
			WithComponent("reasoning_service")
	}

	// Create metrics registry
	metricsRegistry := observability.NewMetricsRegistry()

	// Create extractor
	extractor := reasoning.NewExtractor()

	// Convert event bus to orchestrator.EventBus interface
	var orchEventBus orchestrator.EventBus
	if adapter, ok := eventBus.(orchestrator.EventBus); ok {
		orchEventBus = adapter
	} else {
		// Create adapter if needed
		orchEventBus = &eventBusAdapter{eventBus: eventBus}
	}

	// Create slog logger from observability logger
	slogLogger := slog.New(&observabilityLogHandler{logger: logger})

	// Create reasoning registry config
	reasoningConfig := reasoning.Config{
		CircuitBreakerFailureThreshold: config.CircuitBreakerFailureThreshold,
		CircuitBreakerSuccessThreshold: config.CircuitBreakerSuccessThreshold,
		CircuitBreakerTimeout:          time.Duration(config.CircuitBreakerTimeout) * time.Second,
		GlobalRateLimit:                config.GlobalRateLimit,
		GlobalBurst:                    config.GlobalBurst,
		PerAgentRateLimit:              config.PerAgentRateLimit,
		PerAgentBurst:                  config.PerAgentBurst,
		MaxWorkers:                     config.MaxWorkers,
		CleanupInterval:                time.Duration(config.CleanupInterval) * time.Second,
		MetricsInterval:                time.Duration(config.MetricsInterval) * time.Second,
	}

	// Create reasoning registry
	reasoningRegistry, err := reasoning.NewRegistry(
		extractor,
		orchEventBus,
		metricsRegistry,
		slogLogger,
		db,
		reasoningConfig,
	)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to create reasoning registry").
			WithComponent("reasoning_service")
	}

	return &ReasoningService{
		BaseService: BaseService{
			name:        "reasoning",
			description: "Reasoning extraction and analysis",
			status:      ServiceStatusStopped,
			logger:      logger,
		},
		registry: reasoningRegistry,
	}, nil
}

// Start implements Service interface
func (s *ReasoningService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == ServiceStatusRunning {
		return gerror.New("service already running").
			WithCode(gerror.ErrCodeAlreadyExists).
			WithComponent("reasoning_service")
	}

	s.logger.InfoContext(ctx, "Starting reasoning service")

	// Start the reasoning registry
	if err := s.registry.Start(ctx); err != nil {
		return gerror.Wrap(err, "failed to start reasoning registry").
			WithComponent("reasoning_service")
	}

	s.status = ServiceStatusRunning
	s.logger.InfoContext(ctx, "Reasoning service started successfully")

	return nil
}

// Stop implements Service interface
func (s *ReasoningService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != ServiceStatusRunning {
		return gerror.New("service not running").
			WithCode(gerror.ErrCodeFailedPrecondition).
			WithComponent("reasoning_service")
	}

	s.logger.InfoContext(ctx, "Stopping reasoning service")

	// Stop the reasoning registry
	if err := s.registry.Stop(ctx); err != nil {
		return gerror.Wrap(err, "failed to stop reasoning registry").
			WithComponent("reasoning_service")
	}

	s.status = ServiceStatusStopped
	s.logger.InfoContext(ctx, "Reasoning service stopped successfully")

	return nil
}

// Health returns the health status of the service
func (s *ReasoningService) Health(ctx context.Context) (*HealthStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status != ServiceStatusRunning {
		return &HealthStatus{
			Healthy: false,
			Message: "service not running",
		}, nil
	}

	// Get health from reasoning registry
	health := s.registry.Health(ctx)

	return &HealthStatus{
		Healthy: health.Status == component.StatusHealthy,
		Message: health.Message,
		Details: health.Details,
	}, nil
}

// GetRegistry returns the reasoning registry for direct access
func (s *ReasoningService) GetRegistry() *reasoning.Registry {
	return s.registry
}

// eventBusAdapter adapts events.EventBus to orchestrator.EventBus
type eventBusAdapter struct {
	eventBus events.EventBus
}

func (a *eventBusAdapter) Publish(ctx context.Context, event interface{}) error {
	if e, ok := event.(events.Event); ok {
		return a.eventBus.Publish(ctx, e)
	}
	// Wrap non-Event types
	wrappedEvent := &genericEvent{
		eventType: fmt.Sprintf("%T", event),
		payload:   event,
	}
	return a.eventBus.Publish(ctx, wrappedEvent)
}

func (a *eventBusAdapter) Subscribe(eventType string, handler orchestrator.EventHandler) error {
	// Adapt the handler
	adaptedHandler := func(ctx context.Context, event events.Event) error {
		handler(event)
		return nil
	}
	return a.eventBus.Subscribe(eventType, adaptedHandler)
}

func (a *eventBusAdapter) Unsubscribe(eventType string, handler orchestrator.EventHandler) error {
	// This is tricky since we need to match the original handler
	// For now, return unimplemented
	return gerror.New("unsubscribe not implemented for adapter").
		WithCode(gerror.ErrCodeNotImplemented).
		WithComponent("event_bus_adapter")
}

// genericEvent wraps any type as an Event
type genericEvent struct {
	eventType string
	payload   interface{}
}

func (e *genericEvent) Type() string {
	return e.eventType
}

func (e *genericEvent) Timestamp() time.Time {
	return time.Now()
}

func (e *genericEvent) Source() string {
	return "reasoning_service"
}

func (e *genericEvent) Data() interface{} {
	return e.payload
}

// observabilityLogHandler adapts observability.Logger to slog.Handler
type observabilityLogHandler struct {
	logger observability.Logger
}

func (h *observabilityLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Map slog levels to our logger
	switch level {
	case slog.LevelDebug:
		return true // Assume debug is enabled
	case slog.LevelInfo:
		return true
	case slog.LevelWarn:
		return true
	case slog.LevelError:
		return true
	default:
		return true
	}
}

func (h *observabilityLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Extract fields
	fields := make(map[string]interface{})
	record.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	// Log based on level
	switch record.Level {
	case slog.LevelDebug:
		h.logger.DebugContext(ctx, record.Message, fields)
	case slog.LevelInfo:
		h.logger.InfoContext(ctx, record.Message, fields)
	case slog.LevelWarn:
		h.logger.WarnContext(ctx, record.Message, fields)
	case slog.LevelError:
		h.logger.ErrorContext(ctx, record.Message, fields)
	}

	return nil
}

func (h *observabilityLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, return same handler
	return h
}

func (h *observabilityLogHandler) WithGroup(name string) slog.Handler {
	// For simplicity, return same handler
	return h
}
