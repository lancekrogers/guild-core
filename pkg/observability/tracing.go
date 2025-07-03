// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// TracerProvider wraps OpenTelemetry tracer provider
type TracerProvider struct {
	provider trace.TracerProvider
	shutdown func(context.Context) error
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string // OTLP endpoint (e.g., "localhost:4317")
	Insecure       bool   // Use insecure connection
	Enabled        bool   // Enable tracing
	SampleRate     float64
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		ServiceName:    getEnv("GUILD_SERVICE", "guild"),
		ServiceVersion: getEnv("GUILD_VERSION", "unknown"),
		Environment:    getEnv("GUILD_ENV", "development"),
		Endpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Insecure:       getEnv("OTEL_EXPORTER_OTLP_INSECURE", "true") == "true",
		Enabled:        getEnv("GUILD_TRACING_ENABLED", "true") == "true",
		SampleRate:     1.0, // Sample all traces by default
	}
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(ctx context.Context, config *TracingConfig) (*TracerProvider, error) {
	if config == nil {
		config = DefaultTracingConfig()
	}

	if !config.Enabled {
		// Return no-op provider
		return &TracerProvider{
			provider: otel.GetTracerProvider(),
			shutdown: func(ctx context.Context) error { return nil },
		}, nil
	}

	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
		otlptracegrpc.WithTimeout(30 * time.Second),
	}

	if config.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create trace exporter").
			WithComponent("observability").
			WithOperation("InitTracing").
			WithDetails("endpoint", config.Endpoint)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create resource").
			WithComponent("observability").
			WithOperation("InitTracing").
			WithDetails("service_name", config.ServiceName)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: provider,
		shutdown: provider.Shutdown,
	}, nil
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.shutdown != nil {
		return tp.shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/lancekrogers/guild")
	return tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with attributes
func StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, trace.Span) {
	var attributes []attribute.KeyValue
	for k, v := range attrs {
		attributes = append(attributes, attributeFromValue(k, v))
	}

	return StartSpan(ctx, name, trace.WithAttributes(attributes...))
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	// Extract Guild error details
	var gerr *gerror.GuildError
	if gerror.As(err, &gerr) {
		span.SetAttributes(
			attribute.String("error.code", string(gerr.Code)),
			attribute.String("error.component", gerr.Component),
			attribute.String("error.operation", gerr.Operation),
			attribute.Bool("error.retryable", gerr.Retryable),
		)

		if gerr.Details != nil {
			for k, v := range gerr.Details {
				span.SetAttributes(attributeFromValue(fmt.Sprintf("error.details.%s", k), v))
			}
		}
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	var attributes []attribute.KeyValue
	for k, v := range attrs {
		attributes = append(attributes, attributeFromValue(k, v))
	}

	span.SetAttributes(attributes...)
}

// TraceOperation runs a function with a traced span
func TraceOperation(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
	}

	return err
}

// TraceOperationWithResult runs a function with a traced span and returns a result
func TraceOperationWithResult[T any](ctx context.Context, name string, fn func(context.Context) (T, error)) (T, error) {
	ctx, span := StartSpan(ctx, name)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		RecordError(ctx, err)
	}

	return result, err
}

// ExtractTraceContext extracts trace context from a map
func ExtractTraceContext(ctx context.Context, carrier map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.MapCarrier(carrier))
}

// InjectTraceContext injects trace context into a map
func InjectTraceContext(ctx context.Context) map[string]string {
	carrier := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(carrier))
	return carrier
}

// attributeFromValue converts a value to an OpenTelemetry attribute
func attributeFromValue(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	default:
		// Convert to string as fallback
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// Guild-specific span helpers

// StartAgentSpan starts a span for agent operations
func StartAgentSpan(ctx context.Context, agentID, operation string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, fmt.Sprintf("core.%s", operation), map[string]interface{}{
		"core.id":        agentID,
		"core.operation": operation,
	})
}

// StartTaskSpan starts a span for task operations
func StartTaskSpan(ctx context.Context, taskID, operation string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, fmt.Sprintf("task.%s", operation), map[string]interface{}{
		"task.id":        taskID,
		"task.operation": operation,
	})
}

// StartProviderSpan starts a span for provider operations
func StartProviderSpan(ctx context.Context, provider, operation string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, fmt.Sprintf("provider.%s", operation), map[string]interface{}{
		"provider.name":      provider,
		"provider.operation": operation,
	})
}

// StartStorageSpan starts a span for storage operations
func StartStorageSpan(ctx context.Context, operation string, table string) (context.Context, trace.Span) {
	return StartSpanWithAttributes(ctx, fmt.Sprintf("storage.%s", operation), map[string]interface{}{
		"storage.operation": operation,
		"storage.table":     table,
	})
}
