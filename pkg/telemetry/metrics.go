package telemetry

import (
	"context"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/telemetry/collectors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry provides comprehensive observability for Guild
type Telemetry struct {
	meter        metric.Meter
	tracer       trace.Tracer
	serviceName  string
	shutdownFunc func(context.Context) error

	// Request metrics
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
	errorCount      metric.Int64Counter

	// Business metrics
	commissionsStarted   metric.Int64Counter
	commissionsCompleted metric.Int64Counter
	agentInvocations     metric.Int64Counter
	tokenUsage           metric.Int64Counter

	// System metrics are registered via callbacks
	systemCollector *collectors.SystemCollector
}

// Config holds telemetry configuration
type Config struct {
	ServiceName        string
	ServiceVersion     string
	Environment        string
	OTLPEndpoint       string
	PrometheusEndpoint string
	JaegerEndpoint     string
	SampleRate         float64
}

// Validate checks if the config is valid
func (c Config) Validate() error {
	if c.ServiceName == "" {
		return gerror.New(gerror.ErrCodeValidation, "service name is required", nil)
	}
	if c.SampleRate < 0 || c.SampleRate > 1 {
		return gerror.New(gerror.ErrCodeValidation, "sampling rate must be between 0 and 1", nil)
	}
	return nil
}

// MetricName constants for consistency
const (
	MetricRequestDuration      = "guild.request.duration"
	MetricActiveRequests       = "guild.request.active"
	MetricErrors               = "guild.errors.total"
	MetricCommissionsStarted   = "guild.commissions.started"
	MetricCommissionsCompleted = "guild.commissions.completed"
	MetricAgentInvocations     = "guild.agents.invocations"
	MetricTokenUsage           = "guild.tokens.used"
)

// New creates a new telemetry instance with configured exporters
func New(ctx context.Context, cfg Config) (*Telemetry, error) {
	if cfg.ServiceName == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "service name is required", nil)
	}

	// Initialize providers and exporters via separate setup
	shutdown, err := setupOTelProviders(ctx, cfg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup OpenTelemetry providers")
	}

	meter := otel.Meter(cfg.ServiceName)
	tracer := otel.Tracer(cfg.ServiceName)

	t := &Telemetry{
		meter:        meter,
		tracer:       tracer,
		serviceName:  cfg.ServiceName,
		shutdownFunc: shutdown,
	}

	// Initialize metrics
	if err := t.initializeMetrics(); err != nil {
		// Try to cleanup on error
		_ = shutdown(ctx)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize metrics")
	}

	// Initialize system collector
	t.systemCollector = collectors.NewSystemCollector()
	if err := t.systemCollector.Register(meter); err != nil {
		_ = shutdown(ctx)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register system collector")
	}

	return t, nil
}

// initializeMetrics creates all metric instruments
func (t *Telemetry) initializeMetrics() error {
	var err error

	// Request metrics
	t.requestDuration, err = t.meter.Float64Histogram(
		MetricRequestDuration,
		metric.WithDescription("Duration of requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create request duration histogram")
	}

	t.activeRequests, err = t.meter.Int64UpDownCounter(
		MetricActiveRequests,
		metric.WithDescription("Number of active requests"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create active requests counter")
	}

	t.errorCount, err = t.meter.Int64Counter(
		MetricErrors,
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create error counter")
	}

	// Business metrics
	t.commissionsStarted, err = t.meter.Int64Counter(
		MetricCommissionsStarted,
		metric.WithDescription("Number of commissions started"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create commissions started counter")
	}

	t.commissionsCompleted, err = t.meter.Int64Counter(
		MetricCommissionsCompleted,
		metric.WithDescription("Number of commissions completed"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create commissions completed counter")
	}

	t.agentInvocations, err = t.meter.Int64Counter(
		MetricAgentInvocations,
		metric.WithDescription("Number of agent invocations"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent invocations counter")
	}

	t.tokenUsage, err = t.meter.Int64Counter(
		MetricTokenUsage,
		metric.WithDescription("Number of tokens used"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create token usage counter")
	}

	return nil
}

// RecordRequest records metrics for a request
func (t *Telemetry) RecordRequest(ctx context.Context, operation string, duration time.Duration, err error, attrs ...attribute.KeyValue) {
	// Add operation attribute
	allAttrs := append([]attribute.KeyValue{
		attribute.String("operation", operation),
	}, attrs...)

	// Record duration
	t.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(allAttrs...))

	// Record error if present
	if err != nil {
		errorAttrs := append(allAttrs, attribute.String("error", err.Error()))
		t.errorCount.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

// IncrementActiveRequests increments the active request counter
func (t *Telemetry) IncrementActiveRequests(ctx context.Context, attrs ...attribute.KeyValue) {
	t.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveRequests decrements the active request counter
func (t *Telemetry) DecrementActiveRequests(ctx context.Context, attrs ...attribute.KeyValue) {
	t.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordCommissionStarted records a commission start event
func (t *Telemetry) RecordCommissionStarted(ctx context.Context, commissionID string, attrs ...attribute.KeyValue) {
	allAttrs := append([]attribute.KeyValue{
		attribute.String("commission.id", commissionID),
	}, attrs...)
	t.commissionsStarted.Add(ctx, 1, metric.WithAttributes(allAttrs...))
}

// RecordCommissionCompleted records a commission completion event
func (t *Telemetry) RecordCommissionCompleted(ctx context.Context, commissionID string, success bool, attrs ...attribute.KeyValue) {
	allAttrs := append([]attribute.KeyValue{
		attribute.String("commission.id", commissionID),
		attribute.Bool("commission.success", success),
	}, attrs...)
	t.commissionsCompleted.Add(ctx, 1, metric.WithAttributes(allAttrs...))
}

// RecordAgentInvocation records an agent invocation
func (t *Telemetry) RecordAgentInvocation(ctx context.Context, agentName string, attrs ...attribute.KeyValue) {
	allAttrs := append([]attribute.KeyValue{
		attribute.String("agent.name", agentName),
	}, attrs...)
	t.agentInvocations.Add(ctx, 1, metric.WithAttributes(allAttrs...))
}

// RecordTokenUsage records token usage
func (t *Telemetry) RecordTokenUsage(ctx context.Context, provider string, model string, count int64, attrs ...attribute.KeyValue) {
	allAttrs := append([]attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}, attrs...)
	t.tokenUsage.Add(ctx, count, metric.WithAttributes(allAttrs...))
}

// Tracer returns the configured tracer
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// Meter returns the configured meter
func (t *Telemetry) Meter() metric.Meter {
	return t.meter
}

// Shutdown gracefully shuts down the telemetry system
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.shutdownFunc != nil {
		return t.shutdownFunc(ctx)
	}
	return nil
}
