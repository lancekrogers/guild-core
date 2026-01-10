package telemetry

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindClient   SpanKind = "client"
	SpanKindServer   SpanKind = "server"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// TraceOption allows customization of trace operations
type TraceOption func(*traceConfig)

type traceConfig struct {
	spanKind   trace.SpanKind
	attributes []attribute.KeyValue
}

// WithSpanKind sets the span kind
func WithSpanKind(kind SpanKind) TraceOption {
	return func(cfg *traceConfig) {
		switch kind {
		case SpanKindClient:
			cfg.spanKind = trace.SpanKindClient
		case SpanKindServer:
			cfg.spanKind = trace.SpanKindServer
		case SpanKindProducer:
			cfg.spanKind = trace.SpanKindProducer
		case SpanKindConsumer:
			cfg.spanKind = trace.SpanKindConsumer
		default:
			cfg.spanKind = trace.SpanKindInternal
		}
	}
}

// WithAttributes adds attributes to the span
func WithAttributes(attrs ...attribute.KeyValue) TraceOption {
	return func(cfg *traceConfig) {
		cfg.attributes = append(cfg.attributes, attrs...)
	}
}

// TraceOperation executes a function within a traced span
func (t *Telemetry) TraceOperation(ctx context.Context, operation string, fn func(context.Context) error, opts ...TraceOption) error {
	cfg := &traceConfig{
		spanKind: trace.SpanKindInternal,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx, span := t.tracer.Start(ctx, operation,
		trace.WithSpanKind(cfg.spanKind),
		trace.WithAttributes(cfg.attributes...),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TraceFunc wraps a function to automatically create spans
func (t *Telemetry) TraceFunc(operation string, opts ...TraceOption) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		return t.TraceOperation(ctx, operation, fn, opts...)
	}
}

// StartSpan starts a new span that must be manually ended
func (t *Telemetry) StartSpan(ctx context.Context, operation string, opts ...TraceOption) (context.Context, trace.Span) {
	cfg := &traceConfig{
		spanKind: trace.SpanKindInternal,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return t.tracer.Start(ctx, operation,
		trace.WithSpanKind(cfg.spanKind),
		trace.WithAttributes(cfg.attributes...),
	)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithAttributes(attrs...))
}

// TracedError wraps an error with span recording
func TracedError(ctx context.Context, err error, msg string) error {
	if err == nil {
		return nil
	}
	RecordError(ctx, err, attribute.String("error.message", msg))
	return gerror.Wrap(err, gerror.ErrCodeInternal, msg)
}

// Link represents a link to another span
type Link struct {
	TraceID    string
	SpanID     string
	Attributes []attribute.KeyValue
}

// ExtractTraceContext extracts trace context for propagation
func ExtractTraceContext(ctx context.Context) (traceID, spanID string) {
	// First check for a real span
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}

	// Check for injected context (for testing)
	if tc, ok := ctx.Value(traceContextKey{}).(struct{ TraceID, SpanID string }); ok {
		return tc.TraceID, tc.SpanID
	}

	// Return zero-filled IDs when no valid span
	return "00000000000000000000000000000000", "0000000000000000"
}

// traceContextKey is used to store trace context in context values
type traceContextKey struct{}

// InjectTraceContext creates a new context with injected trace information
func InjectTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	// Store the trace context in the context for testing
	// In production, you'd use proper propagators and create a real span
	return context.WithValue(ctx, traceContextKey{}, struct{ TraceID, SpanID string }{traceID, spanID})
}
