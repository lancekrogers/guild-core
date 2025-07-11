package logging

import (
	"context"
	"testing"
)

func TestContextEnrichment(t *testing.T) {
	ctx := context.Background()

	// Test individual context functions
	ctx = WithRequestID(ctx, "req-123")
	if id, ok := RequestID(ctx); !ok || id != "req-123" {
		t.Errorf("RequestID() = %v, %v; want req-123, true", id, ok)
	}

	ctx = WithUserID(ctx, "user-456")
	if id, ok := UserID(ctx); !ok || id != "user-456" {
		t.Errorf("UserID() = %v, %v; want user-456, true", id, ok)
	}

	ctx = WithSessionID(ctx, "session-789")
	if id, ok := SessionID(ctx); !ok || id != "session-789" {
		t.Errorf("SessionID() = %v, %v; want session-789, true", id, ok)
	}

	ctx = WithCommissionID(ctx, "comm-001")
	if id, ok := CommissionID(ctx); !ok || id != "comm-001" {
		t.Errorf("CommissionID() = %v, %v; want comm-001, true", id, ok)
	}

	ctx = WithAgentID(ctx, "agent-002")
	if id, ok := AgentID(ctx); !ok || id != "agent-002" {
		t.Errorf("AgentID() = %v, %v; want agent-002, true", id, ok)
	}

	ctx = WithTaskID(ctx, "task-003")
	if id, ok := TaskID(ctx); !ok || id != "task-003" {
		t.Errorf("TaskID() = %v, %v; want task-003, true", id, ok)
	}

	ctx = WithTraceID(ctx, "trace-004")
	if id, ok := TraceID(ctx); !ok || id != "trace-004" {
		t.Errorf("TraceID() = %v, %v; want trace-004, true", id, ok)
	}

	ctx = WithSpanID(ctx, "span-005")
	if id, ok := SpanID(ctx); !ok || id != "span-005" {
		t.Errorf("SpanID() = %v, %v; want span-005, true", id, ok)
	}
}

func TestContextExtraction(t *testing.T) {
	// Test empty context
	ctx := context.Background()
	fields := extractContextFields(ctx)
	if len(fields) != 0 {
		t.Errorf("extractContextFields() returned %d fields for empty context, want 0", len(fields))
	}

	// Test nil context
	fields = extractContextFields(nil)
	if fields != nil {
		t.Error("extractContextFields(nil) should return nil")
	}

	// Test context with all fields
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithSessionID(ctx, "session-789")
	ctx = WithCommissionID(ctx, "comm-001")
	ctx = WithAgentID(ctx, "agent-002")
	ctx = WithTaskID(ctx, "task-003")
	ctx = WithTraceID(ctx, "trace-004")
	ctx = WithSpanID(ctx, "span-005")

	fields = extractContextFields(ctx)
	if len(fields) != 8 {
		t.Errorf("extractContextFields() returned %d fields, want 8", len(fields))
	}

	// Verify each field
	fieldMap := make(map[string]string)
	for _, f := range fields {
		fieldMap[f.Key] = f.Value.String()
	}

	expectedFields := map[string]string{
		"request_id":    "req-123",
		"user_id":       "user-456",
		"session_id":    "session-789",
		"commission_id": "comm-001",
		"agent_id":      "agent-002",
		"task_id":       "task-003",
		"trace_id":      "trace-004",
		"span_id":       "span-005",
	}

	for key, expectedValue := range expectedFields {
		if value, ok := fieldMap[key]; !ok || value != expectedValue {
			t.Errorf("Field %s = %v, want %v", key, value, expectedValue)
		}
	}
}

func TestContextEnricher(t *testing.T) {
	ctx := context.Background()

	enricher := ContextEnricher{
		RequestID:    "req-123",
		UserID:       "user-456",
		SessionID:    "session-789",
		CommissionID: "comm-001",
		AgentID:      "agent-002",
		TaskID:       "task-003",
		TraceID:      "trace-004",
		SpanID:       "span-005",
	}

	ctx = enricher.Apply(ctx)

	// Verify all fields were set
	if id, ok := RequestID(ctx); !ok || id != "req-123" {
		t.Errorf("RequestID() = %v, %v; want req-123, true", id, ok)
	}
	if id, ok := UserID(ctx); !ok || id != "user-456" {
		t.Errorf("UserID() = %v, %v; want user-456, true", id, ok)
	}
	if id, ok := SessionID(ctx); !ok || id != "session-789" {
		t.Errorf("SessionID() = %v, %v; want session-789, true", id, ok)
	}
	if id, ok := CommissionID(ctx); !ok || id != "comm-001" {
		t.Errorf("CommissionID() = %v, %v; want comm-001, true", id, ok)
	}
	if id, ok := AgentID(ctx); !ok || id != "agent-002" {
		t.Errorf("AgentID() = %v, %v; want agent-002, true", id, ok)
	}
	if id, ok := TaskID(ctx); !ok || id != "task-003" {
		t.Errorf("TaskID() = %v, %v; want task-003, true", id, ok)
	}
	if id, ok := TraceID(ctx); !ok || id != "trace-004" {
		t.Errorf("TraceID() = %v, %v; want trace-004, true", id, ok)
	}
	if id, ok := SpanID(ctx); !ok || id != "span-005" {
		t.Errorf("SpanID() = %v, %v; want span-005, true", id, ok)
	}
}

func TestPartialContextEnricher(t *testing.T) {
	ctx := context.Background()

	// Test with only some fields set
	enricher := ContextEnricher{
		RequestID: "req-123",
		UserID:    "", // Empty, should not be set
		SessionID: "session-789",
	}

	ctx = enricher.Apply(ctx)

	// Verify only non-empty fields were set
	if id, ok := RequestID(ctx); !ok || id != "req-123" {
		t.Errorf("RequestID() = %v, %v; want req-123, true", id, ok)
	}
	if _, ok := UserID(ctx); ok {
		t.Error("UserID() should not be set for empty value")
	}
	if id, ok := SessionID(ctx); !ok || id != "session-789" {
		t.Errorf("SessionID() = %v, %v; want session-789, true", id, ok)
	}
}

func TestContextOverwrite(t *testing.T) {
	ctx := context.Background()

	// Set initial value
	ctx = WithRequestID(ctx, "req-123")

	// Overwrite with new value
	ctx = WithRequestID(ctx, "req-456")

	// Verify new value
	if id, ok := RequestID(ctx); !ok || id != "req-456" {
		t.Errorf("RequestID() = %v, %v; want req-456, true", id, ok)
	}
}

func TestContextWithoutValues(t *testing.T) {
	ctx := context.Background()

	// Test extraction when values are not set
	tests := []struct {
		name      string
		extractor func(context.Context) (string, bool)
	}{
		{"RequestID", RequestID},
		{"UserID", UserID},
		{"SessionID", SessionID},
		{"CommissionID", CommissionID},
		{"AgentID", AgentID},
		{"TaskID", TaskID},
		{"TraceID", TraceID},
		{"SpanID", SpanID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if value, ok := tt.extractor(ctx); ok || value != "" {
				t.Errorf("%s() = %v, %v; want empty, false", tt.name, value, ok)
			}
		})
	}
}
