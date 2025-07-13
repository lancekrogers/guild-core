package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	tests := []struct {
		name      string
		spanName  string
		opts      []TraceOption
		wantAttrs int
	}{
		{
			name:      "basic span",
			spanName:  "test-operation",
			opts:      nil,
			wantAttrs: 0,
		},
		{
			name:     "span with attributes",
			spanName: "test-with-attrs",
			opts: []TraceOption{
				WithAttributes(
					attribute.String("key1", "value1"),
					attribute.Int("key2", 42),
				),
			},
			wantAttrs: 2,
		},
		{
			name:     "span with kind",
			spanName: "test-server-span",
			opts: []TraceOption{
				WithSpanKind(SpanKindServer),
			},
			wantAttrs: 0,
		},
		{
			name:     "span with multiple options",
			spanName: "test-complex-span",
			opts: []TraceOption{
				WithSpanKind(SpanKindClient),
				WithAttributes(
					attribute.String("service", "test-service"),
					attribute.Bool("test", true),
				),
			},
			wantAttrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCtx, span := tel.StartSpan(ctx, tt.spanName, tt.opts...)
			assert.NotNil(t, span)
			assert.NotEqual(t, ctx, newCtx)

			// Verify span is in context
			spanFromCtx := trace.SpanFromContext(newCtx)
			assert.Equal(t, span, spanFromCtx)

			span.End()
		})
	}
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// No span in context
	span := SpanFromContext(ctx)
	assert.NotNil(t, span)
	assert.False(t, span.IsRecording())

	// With span in context
	ctx2, span2 := tel.StartSpan(ctx, "test-span")
	spanFromCtx := SpanFromContext(ctx2)
	assert.Equal(t, span2, spanFromCtx)
	span2.End()
}

func TestRecordError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		err   error
		attrs []attribute.KeyValue
	}{
		{
			name:  "nil error",
			err:   nil,
			attrs: []attribute.KeyValue{},
		},
		{
			name: "error with attributes",
			err:  errors.New("test error"),
			attrs: []attribute.KeyValue{
				attribute.String("component", "test"),
				attribute.Int("retry_count", 3),
			},
		},
		{
			name:  "gerror",
			err:   gerror.New(gerror.ErrCodeTimeout, "operation timed out", nil),
			attrs: []attribute.KeyValue{attribute.String("operation", "fetch")},
		},
	}

	tel := NewNoop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, span := tel.StartSpan(ctx, "test-span")

			assert.NotPanics(t, func() {
				RecordError(ctx, tt.err, tt.attrs...)
			})

			span.End()
		})
	}
}

func TestSetSpanAttributes(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()
	ctx, span := tel.StartSpan(ctx, "test-span")
	defer span.End()

	// Test various attribute types
	attrs := []attribute.KeyValue{
		attribute.String("string_key", "value"),
		attribute.Int("int_key", 42),
		attribute.Int64("int64_key", int64(9999999999)),
		attribute.Float64("float_key", 3.14),
		attribute.Bool("bool_key", true),
		attribute.StringSlice("string_slice", []string{"a", "b", "c"}),
		attribute.IntSlice("int_slice", []int{1, 2, 3}),
	}

	assert.NotPanics(t, func() {
		SetSpanAttributes(ctx, attrs...)
	})

	// Test empty attributes
	assert.NotPanics(t, func() {
		SetSpanAttributes(ctx)
	})

	// Test with no span in context
	emptyCtx := context.Background()
	assert.NotPanics(t, func() {
		SetSpanAttributes(emptyCtx, attribute.String("key", "value"))
	})
}

func TestAddEvent(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	tests := []struct {
		name  string
		event string
		attrs []attribute.KeyValue
	}{
		{
			name:  "simple event",
			event: "checkpoint_reached",
			attrs: nil,
		},
		{
			name:  "event with attributes",
			event: "error_recovered",
			attrs: []attribute.KeyValue{
				attribute.String("error_type", "timeout"),
				attribute.Int("retry_count", 2),
			},
		},
		{
			name:  "empty event name",
			event: "",
			attrs: []attribute.KeyValue{attribute.String("key", "value")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, span := tel.StartSpan(ctx, "test-span")

			assert.NotPanics(t, func() {
				AddEvent(ctx, tt.event, tt.attrs...)
			})

			span.End()
		})
	}

	// Test with no span in context
	t.Run("no span in context", func(t *testing.T) {
		emptyCtx := context.Background()
		assert.NotPanics(t, func() {
			AddEvent(emptyCtx, "event", attribute.String("key", "value"))
		})
	})
}

func TestTracedError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		err     error
		msg     string
		wantNil bool
	}{
		{
			name:    "nil error",
			err:     nil,
			msg:     "should return nil",
			wantNil: true,
		},
		{
			name:    "standard error",
			err:     errors.New("test error"),
			msg:     "wrapped with message",
			wantNil: false,
		},
		{
			name:    "gerror",
			err:     gerror.New(gerror.ErrCodeNotFound, "not found", nil),
			msg:     "resource not found",
			wantNil: false,
		},
	}

	tel := NewNoop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, span := tel.StartSpan(ctx, "test-span")
			defer span.End()

			result := TracedError(ctx, tt.err, tt.msg)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.Error(t, result)
				assert.Contains(t, result.Error(), tt.msg)
			}
		})
	}
}

func TestExtractTraceContext(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Create a span to get trace context
	ctx, span := tel.StartSpan(ctx, "test-span")
	defer span.End()

	traceID, spanID := ExtractTraceContext(ctx)
	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)

	// Test with no span
	emptyCtx := context.Background()
	emptyTraceID, emptySpanID := ExtractTraceContext(emptyCtx)
	assert.Equal(t, "00000000000000000000000000000000", emptyTraceID)
	assert.Equal(t, "0000000000000000", emptySpanID)
}

func TestInjectTraceContext(t *testing.T) {
	tel := NewNoop()
	// Create a trace context
	originalCtx := context.Background()
	originalCtx, span := tel.StartSpan(originalCtx, "original-span")
	traceID, spanID := ExtractTraceContext(originalCtx)
	span.End()

	// Inject into a new context
	newCtx := context.Background()
	injectedCtx := InjectTraceContext(newCtx, traceID, spanID)

	// Verify the trace context was injected
	newTraceID, newSpanID := ExtractTraceContext(injectedCtx)
	assert.Equal(t, traceID, newTraceID)
	assert.Equal(t, spanID, newSpanID)
}

func TestLink(t *testing.T) {
	link := &Link{
		TraceID: "trace-123",
		SpanID:  "span-456",
		Attributes: []attribute.KeyValue{
			attribute.String("link_type", "parent"),
			attribute.Int("sequence", 1),
		},
	}

	assert.Equal(t, "trace-123", link.TraceID)
	assert.Equal(t, "span-456", link.SpanID)
	assert.Len(t, link.Attributes, 2)
}

func TestNestedSpans(t *testing.T) {
	ctx := context.Background()

	tel := NewNoop()
	// Create parent span
	parentCtx, parentSpan := tel.StartSpan(ctx, "parent-operation",
		WithAttributes(attribute.String("level", "parent")))

	// Create child span
	childCtx, childSpan := tel.StartSpan(parentCtx, "child-operation",
		WithAttributes(attribute.String("level", "child")))

	// Create grandchild span
	grandchildCtx, grandchildSpan := tel.StartSpan(childCtx, "grandchild-operation",
		WithAttributes(attribute.String("level", "grandchild")))

	// Verify spans are different
	assert.NotEqual(t, parentSpan, childSpan)
	assert.NotEqual(t, childSpan, grandchildSpan)
	assert.NotEqual(t, parentSpan, grandchildSpan)

	// Clean up in reverse order
	grandchildSpan.End()
	childSpan.End()
	parentSpan.End()

	_ = grandchildCtx // Use to avoid unused variable warning
}

func TestConcurrentSpanOperations(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()
	ctx, span := tel.StartSpan(ctx, "concurrent-test")
	defer span.End()

	// Run concurrent operations on the same span
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Set attributes
			SetSpanAttributes(ctx,
				attribute.Int("goroutine_id", id),
				attribute.String("operation", "concurrent"))

			// Add events
			AddEvent(ctx, "processing",
				attribute.Int("item", id))

			// Record errors
			if id%2 == 0 {
				RecordError(ctx, errors.New("test error"),
					attribute.Int("error_id", id))
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
}
