package observability

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ContextKey is a type for context keys
type ContextKey string

// Context keys for observability
const (
	RequestIDKey    ContextKey = "request_id"
	TraceIDKey      ContextKey = "trace_id"
	SpanIDKey       ContextKey = "span_id"
	UserIDKey       ContextKey = "user_id"
	SessionIDKey    ContextKey = "session_id"
	AgentIDKey      ContextKey = "agent_id"
	TaskIDKey       ContextKey = "task_id"
	CommissionIDKey ContextKey = "commission_id"
	CampaignIDKey   ContextKey = "campaign_id"
	ComponentKey    ContextKey = "component"
	OperationKey    ContextKey = "operation"
)

// RequestContext holds request-scoped values
type RequestContext struct {
	RequestID    string
	TraceID      string
	SpanID       string
	UserID       string
	SessionID    string
	AgentID      string
	TaskID       string
	CommissionID string
	CampaignID   string
	Component    string
	Operation    string
}

// NewRequestContext creates a new request context with generated IDs
func NewRequestContext() *RequestContext {
	return &RequestContext{
		RequestID: GenerateRequestID(),
		TraceID:   GenerateTraceID(),
		SpanID:    GenerateSpanID(),
	}
}

// ToContext adds the request context values to a context
func (rc *RequestContext) ToContext(ctx context.Context) context.Context {
	if rc.RequestID != "" {
		ctx = context.WithValue(ctx, RequestIDKey, rc.RequestID)
	}
	if rc.TraceID != "" {
		ctx = context.WithValue(ctx, TraceIDKey, rc.TraceID)
	}
	if rc.SpanID != "" {
		ctx = context.WithValue(ctx, SpanIDKey, rc.SpanID)
	}
	if rc.UserID != "" {
		ctx = context.WithValue(ctx, UserIDKey, rc.UserID)
	}
	if rc.SessionID != "" {
		ctx = context.WithValue(ctx, SessionIDKey, rc.SessionID)
	}
	if rc.AgentID != "" {
		ctx = context.WithValue(ctx, AgentIDKey, rc.AgentID)
	}
	if rc.TaskID != "" {
		ctx = context.WithValue(ctx, TaskIDKey, rc.TaskID)
	}
	if rc.CommissionID != "" {
		ctx = context.WithValue(ctx, CommissionIDKey, rc.CommissionID)
	}
	if rc.CampaignID != "" {
		ctx = context.WithValue(ctx, CampaignIDKey, rc.CampaignID)
	}
	if rc.Component != "" {
		ctx = context.WithValue(ctx, ComponentKey, rc.Component)
	}
	if rc.Operation != "" {
		ctx = context.WithValue(ctx, OperationKey, rc.Operation)
	}
	return ctx
}

// FromContext extracts request context from a context
func FromContext(ctx context.Context) *RequestContext {
	rc := &RequestContext{}

	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		rc.RequestID = v
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		rc.TraceID = v
	}
	if v, ok := ctx.Value(SpanIDKey).(string); ok {
		rc.SpanID = v
	}
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		rc.UserID = v
	}
	if v, ok := ctx.Value(SessionIDKey).(string); ok {
		rc.SessionID = v
	}
	if v, ok := ctx.Value(AgentIDKey).(string); ok {
		rc.AgentID = v
	}
	if v, ok := ctx.Value(TaskIDKey).(string); ok {
		rc.TaskID = v
	}
	if v, ok := ctx.Value(CommissionIDKey).(string); ok {
		rc.CommissionID = v
	}
	if v, ok := ctx.Value(CampaignIDKey).(string); ok {
		rc.CampaignID = v
	}
	if v, ok := ctx.Value(ComponentKey).(string); ok {
		rc.Component = v
	}
	if v, ok := ctx.Value(OperationKey).(string); ok {
		rc.Operation = v
	}

	return rc
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID gets the request ID from context
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID gets the trace ID from context
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID gets the user ID from context
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// WithComponent adds a component name to the context
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

// GetComponent gets the component from context
func GetComponent(ctx context.Context) string {
	if v, ok := ctx.Value(ComponentKey).(string); ok {
		return v
	}
	return ""
}

// WithOperation adds an operation name to the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// GetOperation gets the operation from context
func GetOperation(ctx context.Context) string {
	if v, ok := ctx.Value(OperationKey).(string); ok {
		return v
	}
	return ""
}

// WithGuildContext adds Guild-specific context values
func WithGuildContext(ctx context.Context, agentID, taskID, commissionID, campaignID string) context.Context {
	if agentID != "" {
		ctx = context.WithValue(ctx, AgentIDKey, agentID)
	}
	if taskID != "" {
		ctx = context.WithValue(ctx, TaskIDKey, taskID)
	}
	if commissionID != "" {
		ctx = context.WithValue(ctx, CommissionIDKey, commissionID)
	}
	if campaignID != "" {
		ctx = context.WithValue(ctx, CampaignIDKey, campaignID)
	}
	return ctx
}

// GenerateRequestID generates a new request ID
func GenerateRequestID() string {
	return fmt.Sprintf("req_%s", uuid.New().String())
}

// GenerateTraceID generates a new trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateSpanID generates a new span ID
func GenerateSpanID() string {
	return fmt.Sprintf("%016x", uuid.New().ID())
}

// GenerateSessionID generates a new session ID
func GenerateSessionID() string {
	return fmt.Sprintf("sess_%s", uuid.New().String())
}

// EnsureRequestContext ensures the context has request tracking IDs
func EnsureRequestContext(ctx context.Context) context.Context {
	// Check if we already have a request ID
	if GetRequestID(ctx) == "" {
		ctx = WithRequestID(ctx, GenerateRequestID())
	}

	// Check if we already have a trace ID
	if GetTraceID(ctx) == "" {
		ctx = WithTraceID(ctx, GenerateTraceID())
	}

	return ctx
}

// PropagateContext copies observability context from one context to another
func PropagateContext(from, to context.Context) context.Context {
	rc := FromContext(from)
	return rc.ToContext(to)
}
