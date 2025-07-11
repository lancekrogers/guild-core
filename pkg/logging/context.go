package logging

import (
	"context"
)

// Context keys for logging metadata
type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	userIDKey       contextKey = "user_id"
	sessionIDKey    contextKey = "session_id"
	commissionIDKey contextKey = "commission_id"
	agentIDKey      contextKey = "agent_id"
	taskIDKey       contextKey = "task_id"
	traceIDKey      contextKey = "trace_id"
	spanIDKey       contextKey = "span_id"
)

// Context enrichment functions

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// WithCommissionID adds a commission ID to the context
func WithCommissionID(ctx context.Context, commissionID string) context.Context {
	return context.WithValue(ctx, commissionIDKey, commissionID)
}

// WithAgentID adds an agent ID to the context
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDKey, agentID)
}

// WithTaskID adds a task ID to the context
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, taskIDKey, taskID)
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithSpanID adds a span ID to the context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, spanIDKey, spanID)
}

// Context extraction functions

// RequestID extracts the request ID from context
func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

// UserID extracts the user ID from context
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// SessionID extracts the session ID from context
func SessionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey).(string)
	return id, ok
}

// CommissionID extracts the commission ID from context
func CommissionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(commissionIDKey).(string)
	return id, ok
}

// AgentID extracts the agent ID from context
func AgentID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(agentIDKey).(string)
	return id, ok
}

// TaskID extracts the task ID from context
func TaskID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(taskIDKey).(string)
	return id, ok
}

// TraceID extracts the trace ID from context
func TraceID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(traceIDKey).(string)
	return id, ok
}

// SpanID extracts the span ID from context
func SpanID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(spanIDKey).(string)
	return id, ok
}

// extractContextFields extracts all logging-relevant fields from context
func extractContextFields(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}

	var fields []Field

	if requestID, ok := RequestID(ctx); ok && requestID != "" {
		fields = append(fields, String("request_id", requestID))
	}

	if userID, ok := UserID(ctx); ok && userID != "" {
		fields = append(fields, String("user_id", userID))
	}

	if sessionID, ok := SessionID(ctx); ok && sessionID != "" {
		fields = append(fields, String("session_id", sessionID))
	}

	if commissionID, ok := CommissionID(ctx); ok && commissionID != "" {
		fields = append(fields, String("commission_id", commissionID))
	}

	if agentID, ok := AgentID(ctx); ok && agentID != "" {
		fields = append(fields, String("agent_id", agentID))
	}

	if taskID, ok := TaskID(ctx); ok && taskID != "" {
		fields = append(fields, String("task_id", taskID))
	}

	if traceID, ok := TraceID(ctx); ok && traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}

	if spanID, ok := SpanID(ctx); ok && spanID != "" {
		fields = append(fields, String("span_id", spanID))
	}

	return fields
}

// EnrichContext adds multiple metadata fields to context at once
type ContextEnricher struct {
	RequestID    string
	UserID       string
	SessionID    string
	CommissionID string
	AgentID      string
	TaskID       string
	TraceID      string
	SpanID       string
}

// Apply enriches the context with all non-empty fields
func (e ContextEnricher) Apply(ctx context.Context) context.Context {
	if e.RequestID != "" {
		ctx = WithRequestID(ctx, e.RequestID)
	}
	if e.UserID != "" {
		ctx = WithUserID(ctx, e.UserID)
	}
	if e.SessionID != "" {
		ctx = WithSessionID(ctx, e.SessionID)
	}
	if e.CommissionID != "" {
		ctx = WithCommissionID(ctx, e.CommissionID)
	}
	if e.AgentID != "" {
		ctx = WithAgentID(ctx, e.AgentID)
	}
	if e.TaskID != "" {
		ctx = WithTaskID(ctx, e.TaskID)
	}
	if e.TraceID != "" {
		ctx = WithTraceID(ctx, e.TraceID)
	}
	if e.SpanID != "" {
		ctx = WithSpanID(ctx, e.SpanID)
	}
	return ctx
}
