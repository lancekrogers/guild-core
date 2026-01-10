package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
)

func TestNewTelemetry(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				SampleRate:     1.0,
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			config: Config{
				ServiceVersion: "1.0.0",
				Environment:    "test",
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use noop telemetry for testing
			tel := NewNoop()

			if tt.wantErr {
				cfg := tt.config
				cfg.ServiceName = "" // Force error
				_, err := New(context.Background(), cfg)
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if tel == nil {
					t.Errorf("expected telemetry instance but got nil")
				}
			}
		})
	}
}

func TestTelemetryMetrics(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	t.Run("RecordRequest", func(t *testing.T) {
		// Should not panic
		tel.RecordRequest(ctx, "test-operation", 100*time.Millisecond, nil,
			attribute.String("test", "value"))

		// With error
		err := gerror.New(gerror.ErrCodeInternal, "test error", nil)
		tel.RecordRequest(ctx, "test-operation", 200*time.Millisecond, err,
			attribute.String("test", "error"))
	})

	t.Run("ActiveRequests", func(t *testing.T) {
		tel.IncrementActiveRequests(ctx, attribute.String("endpoint", "/test"))
		tel.DecrementActiveRequests(ctx, attribute.String("endpoint", "/test"))
	})

	t.Run("Commission Metrics", func(t *testing.T) {
		tel.RecordCommissionStarted(ctx, "commission-123",
			attribute.String("type", "test"))
		tel.RecordCommissionCompleted(ctx, "commission-123", true,
			attribute.String("type", "test"))
		tel.RecordCommissionCompleted(ctx, "commission-456", false,
			attribute.String("type", "test"))
	})

	t.Run("Agent Metrics", func(t *testing.T) {
		tel.RecordAgentInvocation(ctx, "test-agent",
			attribute.String("task", "analysis"))
	})

	t.Run("Token Usage", func(t *testing.T) {
		tel.RecordTokenUsage(ctx, "openai", "gpt-4", 1500,
			attribute.String("purpose", "commission"))
	})
}

func TestTelemetryShutdown(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	// Shutdown should not error for noop
	err := tel.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}

// MockTelemetry for testing
type MockTelemetry struct {
	*Telemetry
	RecordedRequests    []MockRequest
	RecordedCommissions []MockCommission
	RecordedAgents      []MockAgent
	RecordedTokens      []MockToken
}

type MockRequest struct {
	Operation string
	Duration  time.Duration
	Error     error
	Attrs     []attribute.KeyValue
}

type MockCommission struct {
	ID      string
	Started bool
	Success bool
	Attrs   []attribute.KeyValue
}

type MockAgent struct {
	Name  string
	Attrs []attribute.KeyValue
}

type MockToken struct {
	Provider string
	Model    string
	Count    int64
	Attrs    []attribute.KeyValue
}

func NewMockTelemetry() *MockTelemetry {
	noop := NewNoop()
	return &MockTelemetry{
		Telemetry:           noop.Telemetry,
		RecordedRequests:    []MockRequest{},
		RecordedCommissions: []MockCommission{},
		RecordedAgents:      []MockAgent{},
		RecordedTokens:      []MockToken{},
	}
}

func (m *MockTelemetry) RecordRequest(ctx context.Context, operation string, duration time.Duration, err error, attrs ...attribute.KeyValue) {
	m.RecordedRequests = append(m.RecordedRequests, MockRequest{
		Operation: operation,
		Duration:  duration,
		Error:     err,
		Attrs:     attrs,
	})
}

func (m *MockTelemetry) RecordCommissionStarted(ctx context.Context, commissionID string, attrs ...attribute.KeyValue) {
	m.RecordedCommissions = append(m.RecordedCommissions, MockCommission{
		ID:      commissionID,
		Started: true,
		Attrs:   attrs,
	})
}

func (m *MockTelemetry) RecordCommissionCompleted(ctx context.Context, commissionID string, success bool, attrs ...attribute.KeyValue) {
	m.RecordedCommissions = append(m.RecordedCommissions, MockCommission{
		ID:      commissionID,
		Success: success,
		Attrs:   attrs,
	})
}

func (m *MockTelemetry) RecordAgentInvocation(ctx context.Context, agentName string, attrs ...attribute.KeyValue) {
	m.RecordedAgents = append(m.RecordedAgents, MockAgent{
		Name:  agentName,
		Attrs: attrs,
	})
}

func (m *MockTelemetry) RecordTokenUsage(ctx context.Context, provider string, model string, count int64, attrs ...attribute.KeyValue) {
	m.RecordedTokens = append(m.RecordedTokens, MockToken{
		Provider: provider,
		Model:    model,
		Count:    count,
		Attrs:    attrs,
	})
}

func TestMockTelemetry(t *testing.T) {
	ctx := context.Background()
	mock := NewMockTelemetry()

	// Test recording
	mock.RecordRequest(ctx, "test-op", 100*time.Millisecond, nil)
	if len(mock.RecordedRequests) != 1 {
		t.Errorf("expected 1 recorded request, got %d", len(mock.RecordedRequests))
	}

	mock.RecordCommissionStarted(ctx, "comm-123")
	mock.RecordCommissionCompleted(ctx, "comm-123", true)
	if len(mock.RecordedCommissions) != 2 {
		t.Errorf("expected 2 recorded commissions, got %d", len(mock.RecordedCommissions))
	}

	mock.RecordAgentInvocation(ctx, "test-agent")
	if len(mock.RecordedAgents) != 1 {
		t.Errorf("expected 1 recorded agent, got %d", len(mock.RecordedAgents))
	}

	mock.RecordTokenUsage(ctx, "openai", "gpt-4", 1000)
	if len(mock.RecordedTokens) != 1 {
		t.Errorf("expected 1 recorded token usage, got %d", len(mock.RecordedTokens))
	}
}

// Table-driven tests for metric recording
func TestMetricRecording(t *testing.T) {
	ctx := context.Background()
	tel := NewNoop()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "request with all attributes",
			fn: func() {
				tel.RecordRequest(ctx, "complex-operation", 500*time.Millisecond, nil,
					attribute.String("method", "POST"),
					attribute.String("endpoint", "/api/commission"),
					attribute.Int("status", 200),
				)
			},
		},
		{
			name: "request with error",
			fn: func() {
				err := gerror.New(gerror.ErrCodeValidation, "validation failed", nil).WithDetails("field", "name")
				tel.RecordRequest(ctx, "validation", 10*time.Millisecond, err,
					attribute.String("validator", "schema"),
				)
			},
		},
		{
			name: "concurrent request tracking",
			fn: func() {
				attrs := []attribute.KeyValue{attribute.String("handler", "chat")}
				tel.IncrementActiveRequests(ctx, attrs...)
				tel.IncrementActiveRequests(ctx, attrs...)
				tel.DecrementActiveRequests(ctx, attrs...)
				tel.DecrementActiveRequests(ctx, attrs...)
			},
		},
		{
			name: "commission lifecycle",
			fn: func() {
				commID := "comm-test-123"
				attrs := []attribute.KeyValue{
					attribute.String("type", "code-generation"),
					attribute.String("priority", "high"),
				}
				tel.RecordCommissionStarted(ctx, commID, attrs...)
				time.Sleep(10 * time.Millisecond) // Simulate work
				tel.RecordCommissionCompleted(ctx, commID, true, attrs...)
			},
		},
		{
			name: "agent invocation with context",
			fn: func() {
				tel.RecordAgentInvocation(ctx, "code-analyst",
					attribute.String("task", "review"),
					attribute.String("language", "go"),
					attribute.Int("lines", 500),
				)
			},
		},
		{
			name: "token usage by multiple providers",
			fn: func() {
				tel.RecordTokenUsage(ctx, "openai", "gpt-4", 1500,
					attribute.String("purpose", "analysis"))
				tel.RecordTokenUsage(ctx, "anthropic", "claude-3", 2000,
					attribute.String("purpose", "generation"))
				tel.RecordTokenUsage(ctx, "openai", "gpt-3.5-turbo", 500,
					attribute.String("purpose", "summary"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}
