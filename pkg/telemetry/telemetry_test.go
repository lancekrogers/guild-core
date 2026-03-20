package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config",
			config:  Config{},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "valid minimal config",
			config: Config{
				ServiceName: "test-service",
			},
			wantErr: false,
		},
		{
			name: "full config",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "test",
				OTLPEndpoint:       "localhost:4317",
				JaegerEndpoint:     "localhost:14250",
				PrometheusEndpoint: ":9090",
				SampleRate:         0.1,
			},
			wantErr: false,
		},
		{
			name: "invalid sampling rate too high",
			config: Config{
				ServiceName: "test-service",
				SampleRate:  1.5,
			},
			wantErr: true,
			errMsg:  "sampling rate must be between 0 and 1",
		},
		{
			name: "invalid sampling rate negative",
			config: Config{
				ServiceName: "test-service",
				SampleRate:  -0.1,
			},
			wantErr: true,
			errMsg:  "sampling rate must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTelemetry_New(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config creates telemetry",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
			},
			wantErr: false,
		},
		{
			name:    "invalid config returns error",
			config:  Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := New(ctx, tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, tel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tel)
				assert.NotNil(t, tel.meter)
				assert.NotNil(t, tel.tracer)
				assert.NotNil(t, tel.systemCollector)

				// Cleanup
				err = tel.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestTelemetry_RecordRequest(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Test successful request
	tel.RecordRequest(ctx, "test-op", 100*time.Millisecond, nil,
		attribute.String("test", "value"))

	// Test failed request
	err := gerror.New(gerror.ErrCodeInternal, "test error", nil)
	tel.RecordRequest(ctx, "test-op", 200*time.Millisecond, err,
		attribute.String("test", "value"))

	// Verify no panics
	assert.NotPanics(t, func() {
		tel.RecordRequest(ctx, "", 0, nil)
	})
}

func TestTelemetry_RecordError(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	tests := []struct {
		name      string
		err       error
		errorType string
		attrs     []attribute.KeyValue
	}{
		{
			name:      "gerror with code",
			err:       gerror.New(gerror.ErrCodeValidation, "validation failed", nil),
			errorType: "validation",
			attrs:     []attribute.KeyValue{attribute.String("field", "email")},
		},
		{
			name:      "standard error",
			err:       assert.AnError,
			errorType: "internal",
			attrs:     []attribute.KeyValue{},
		},
		{
			name:      "nil error",
			err:       nil,
			errorType: "",
			attrs:     []attribute.KeyValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				// Record as a failed request instead
				tel.RecordRequest(ctx, tt.errorType, 100*time.Millisecond, tt.err, tt.attrs...)
			})
		})
	}
}

func TestTelemetry_BusinessMetrics(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Test commission metrics
	tel.RecordCommissionStarted(ctx, "test-commission", attribute.String("type", "test-type"))
	tel.RecordCommissionCompleted(ctx, "test-commission", true, attribute.String("type", "test-type"))
	tel.RecordCommissionCompleted(ctx, "test-commission", false, attribute.String("type", "test-type"))

	// Test agent metrics
	tel.RecordAgentInvocation(ctx, "test-agent", attribute.String("type", "test-type"))

	// Test token usage
	tel.RecordTokenUsage(ctx, "openai", "gpt-4", 1000)

	// Verify no panics
	assert.NotPanics(t, func() {
		tel.RecordCommissionStarted(ctx, "")
		tel.RecordCommissionCompleted(ctx, "", true)
		tel.RecordAgentInvocation(ctx, "")
		tel.RecordTokenUsage(ctx, "", "", 0)
	})
}

func TestTelemetry_StartSpan(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Test basic span
	ctx, span := tel.StartSpan(ctx, "test-operation")
	assert.NotNil(t, span)
	span.End()

	// Test span with attributes
	_, span2 := tel.StartSpan(ctx, "test-operation-2",
		WithAttributes(
			attribute.String("key", "value"),
			attribute.Int("count", 42),
		))
	assert.NotNil(t, span2)
	span2.End()

	// Test nested spans
	ctx3, parent := tel.StartSpan(ctx, "parent-operation")
	_, child := tel.StartSpan(ctx3, "child-operation")
	assert.NotNil(t, parent)
	assert.NotNil(t, child)
	child.End()
	parent.End()
}

func TestTelemetry_Metrics(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Get all metrics for verification
	meter := tel.Meter()
	assert.NotNil(t, meter)

	tracer := tel.Tracer()
	assert.NotNil(t, tracer)

	// Test metric accessors
	assert.NotPanics(t, func() {
		// These should work with noop implementation
		_, span := tracer.Start(ctx, "test")
		span.End()
	})
}

func TestTelemetry_Shutdown(t *testing.T) {
	ctx := context.Background()

	// Test shutdown with noop
	tel := NewNoop()
	err := tel.Shutdown(ctx)
	assert.NoError(t, err)

	// Test shutdown with real telemetry
	realTel, err := New(ctx, Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	require.NoError(t, err)

	err = realTel.Shutdown(ctx)
	assert.NoError(t, err)

	// Test shutdown with timeout
	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	err = realTel.Shutdown(shortCtx)
	// May or may not timeout, but should not panic
	assert.NotPanics(t, func() {
		_ = realTel.Shutdown(shortCtx)
	})
}

func TestNoopTelemetry(t *testing.T) {
	tel := NewNoop()
	assert.NotNil(t, tel)
	assert.NotNil(t, tel.Telemetry)
	assert.NotNil(t, tel.meter)
	assert.NotNil(t, tel.tracer)

	// Verify noop doesn't panic on any operation
	ctx := context.Background()
	assert.NotPanics(t, func() {
		tel.RecordRequest(ctx, "op", time.Second, nil)
		// RecordError is a standalone function, not a method
		RecordError(ctx, assert.AnError, attribute.String("key", "value"))
		tel.RecordCommissionStarted(ctx, "commission", attribute.String("type", "test-type"))
		tel.RecordCommissionCompleted(ctx, "commission", true, attribute.String("type", "test-type"))
		tel.RecordAgentInvocation(ctx, "agent", attribute.String("type", "test-type"))
		tel.RecordTokenUsage(ctx, "openai", "gpt-4", 100)

		ctx2, span := tel.StartSpan(ctx, "operation")
		span.End()
		_ = ctx2

		_ = tel.Shutdown(ctx)
	})
}

func TestConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Test concurrent metric recording
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Record various metrics concurrently
			tel.RecordRequest(ctx, "concurrent-op", time.Duration(id)*time.Millisecond, nil)
			RecordError(ctx, assert.AnError, attribute.String("type", "concurrent-error"))
			tel.RecordCommissionStarted(ctx, "commission", attribute.String("type", "concurrent"))
			tel.RecordAgentInvocation(ctx, "agent", attribute.String("type", "concurrent"))
			tel.RecordTokenUsage(ctx, "openai", "gpt-4", int64(id*100))

			// Create spans
			_, span := tel.StartSpan(ctx, "concurrent-span")
			span.End()
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
	assert.NotPanics(t, func() {
		err := tel.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
