// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ContextKey represents a key for storing values in context
type ContextKey string

// Core context keys for the Guild framework
const (
	// Registry and component access
	RegistryKey ContextKey = "guild.registry"
	ConfigKey   ContextKey = "guild.config"

	// Request tracking
	RequestIDKey ContextKey = "guild.request_id"
	SessionIDKey ContextKey = "guild.session_id"
	OperationKey ContextKey = "guild.operation"

	// Component identification
	AgentIDKey  ContextKey = "guild.agent_id"
	ProviderKey ContextKey = "guild.provider"
	ToolKey     ContextKey = "guild.tool"

	// Execution context
	TimeoutKey    ContextKey = "guild.timeout"
	RetryCountKey ContextKey = "guild.retry_count"

	// Debugging and observability
	TraceIDKey ContextKey = "guild.trace_id"
	SpanIDKey  ContextKey = "guild.span_id"
	LoggerKey  ContextKey = "guild.logger"

	// Cost and resource tracking
	CostBudgetKey    ContextKey = "guild.cost_budget"
	ResourceLimitKey ContextKey = "guild.resource_limit"
)

// Registry interface (forward declaration to avoid import cycles)
type ComponentRegistry interface {
	// We'll define this properly when we integrate
}

// Config interface (forward declaration)
type Config interface {
	// We'll define this properly when we integrate
}

// Logger interface for context-aware logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	With(fields ...interface{}) Logger
}

// RequestInfo contains metadata about the current request
type RequestInfo struct {
	ID        string
	SessionID string
	Operation string
	TraceID   string
	SpanID    string
	StartTime time.Time
	UserID    string
	Metadata  map[string]interface{}
}

// OperationInfo contains metadata about the current operation
type OperationInfo struct {
	Name       string
	Component  string
	Action     string
	Timeout    time.Duration
	RetryCount int
	Metadata   map[string]interface{}
}

// CostInfo tracks cost and resource usage
type CostInfo struct {
	Budget     float64
	Used       float64
	Currency   string
	TokenLimit int
	TokensUsed int
}

// ResourceInfo tracks resource limits and usage
type ResourceInfo struct {
	MaxConcurrency int
	CurrentActive  int
	MemoryLimit    int64
	MemoryUsed     int64
}

// ==============================================================================
// Context Creation Functions
// ==============================================================================

// NewGuildContext creates a new Guild context with a unique request ID
func NewGuildContext(parent context.Context) context.Context {
	requestID := uuid.New().String()
	traceID := uuid.New().String()

	ctx := context.WithValue(parent, RequestIDKey, requestID)
	ctx = context.WithValue(ctx, TraceIDKey, traceID)
	ctx = context.WithValue(ctx, SpanIDKey, uuid.New().String())

	// Add request info
	requestInfo := &RequestInfo{
		ID:        requestID,
		TraceID:   traceID,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	ctx = context.WithValue(ctx, "guild.request_info", requestInfo)

	return ctx
}

// NewOperationContext creates a context for a specific operation
func NewOperationContext(parent context.Context, operation string, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.WithValue(parent, OperationKey, operation)
	ctx = context.WithValue(ctx, SpanIDKey, uuid.New().String())

	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}

	return ctx, func() {}
}

// ==============================================================================
// Registry and Configuration
// ==============================================================================

// WithRegistry adds a component registry to the context
func WithRegistry(ctx context.Context, registry ComponentRegistry) context.Context {
	return context.WithValue(ctx, RegistryKey, registry)
}

// GetRegistry retrieves the component registry from context
func GetRegistry(ctx context.Context) (ComponentRegistry, error) {
	if registry, ok := ctx.Value(RegistryKey).(ComponentRegistry); ok {
		return registry, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "no component registry found in context", nil).WithComponent("context").WithOperation("GetRegistry")
}

// WithConfig adds configuration to the context
func WithConfig(ctx context.Context, config Config) context.Context {
	return context.WithValue(ctx, ConfigKey, config)
}

// GetConfig retrieves configuration from context
func GetConfig(ctx context.Context) (Config, error) {
	if config, ok := ctx.Value(ConfigKey).(Config); ok {
		return config, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "no configuration found in context", nil).WithComponent("context").WithOperation("GetConfig")
}

// ==============================================================================
// Request Tracking
// ==============================================================================

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// GetSessionID retrieves the session ID from context
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// WithOperation adds operation information to the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// GetOperation retrieves the current operation from context
func GetOperation(ctx context.Context) string {
	if op, ok := ctx.Value(OperationKey).(string); ok {
		return op
	}
	return ""
}

// ==============================================================================
// Component Identification
// ==============================================================================

// WithAgentID adds an agent ID to the context
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, AgentIDKey, agentID)
}

// GetAgentID retrieves the agent ID from context
func GetAgentID(ctx context.Context) string {
	if id, ok := ctx.Value(AgentIDKey).(string); ok {
		return id
	}
	return ""
}

// WithProvider adds provider information to the context
func WithProvider(ctx context.Context, provider string) context.Context {
	return context.WithValue(ctx, ProviderKey, provider)
}

// GetProvider retrieves the provider from context
func GetProvider(ctx context.Context) string {
	if provider, ok := ctx.Value(ProviderKey).(string); ok {
		return provider
	}
	return ""
}

// WithTool adds tool information to the context
func WithTool(ctx context.Context, tool string) context.Context {
	return context.WithValue(ctx, ToolKey, tool)
}

// GetTool retrieves the tool from context
func GetTool(ctx context.Context) string {
	if tool, ok := ctx.Value(ToolKey).(string); ok {
		return tool
	}
	return ""
}

// ==============================================================================
// Tracing and Debugging
// ==============================================================================

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

// WithSpanID adds a span ID to the context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetSpanID retrieves the span ID from context
func GetSpanID(ctx context.Context) string {
	if id, ok := ctx.Value(SpanIDKey).(string); ok {
		return id
	}
	return ""
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// GetLogger retrieves the logger from context
func GetLogger(ctx context.Context) Logger {
	if logger, ok := ctx.Value(LoggerKey).(Logger); ok {
		return logger
	}
	return nil // Return nil, caller should handle gracefully
}

// ==============================================================================
// Cost and Resource Tracking
// ==============================================================================

// WithCostBudget adds cost budget information to the context
func WithCostBudget(ctx context.Context, budget float64, currency string) context.Context {
	costInfo := &CostInfo{
		Budget:   budget,
		Currency: currency,
		Used:     0.0,
	}
	return context.WithValue(ctx, CostBudgetKey, costInfo)
}

// GetCostInfo retrieves cost information from context
func GetCostInfo(ctx context.Context) *CostInfo {
	if costInfo, ok := ctx.Value(CostBudgetKey).(*CostInfo); ok {
		return costInfo
	}
	return nil
}

// WithResourceLimit adds resource limit information to the context
func WithResourceLimit(ctx context.Context, maxConcurrency int, memoryLimit int64) context.Context {
	resourceInfo := &ResourceInfo{
		MaxConcurrency: maxConcurrency,
		MemoryLimit:    memoryLimit,
		CurrentActive:  0,
		MemoryUsed:     0,
	}
	return context.WithValue(ctx, ResourceLimitKey, resourceInfo)
}

// GetResourceInfo retrieves resource information from context
func GetResourceInfo(ctx context.Context) *ResourceInfo {
	if resourceInfo, ok := ctx.Value(ResourceLimitKey).(*ResourceInfo); ok {
		return resourceInfo
	}
	return nil
}

// ==============================================================================
// Helper Functions
// ==============================================================================

// GetRequestInfo retrieves comprehensive request information from context
func GetRequestInfo(ctx context.Context) *RequestInfo {
	if requestInfo, ok := ctx.Value("guild.request_info").(*RequestInfo); ok {
		return requestInfo
	}

	// Fallback: construct from individual fields
	return &RequestInfo{
		ID:        GetRequestID(ctx),
		SessionID: GetSessionID(ctx),
		Operation: GetOperation(ctx),
		TraceID:   GetTraceID(ctx),
		SpanID:    GetSpanID(ctx),
		StartTime: time.Now(), // Won't be accurate, but better than nothing
		Metadata:  make(map[string]interface{}),
	}
}

// LogFields returns structured logging fields from context
func LogFields(ctx context.Context) []interface{} {
	fields := []interface{}{}

	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, "request_id", requestID)
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}
	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, "span_id", spanID)
	}
	if agentID := GetAgentID(ctx); agentID != "" {
		fields = append(fields, "agent_id", agentID)
	}
	if provider := GetProvider(ctx); provider != "" {
		fields = append(fields, "provider", provider)
	}
	if tool := GetTool(ctx); tool != "" {
		fields = append(fields, "tool", tool)
	}
	if operation := GetOperation(ctx); operation != "" {
		fields = append(fields, "operation", operation)
	}

	return fields
}

// ContextSummary returns a human-readable summary of context values
func ContextSummary(ctx context.Context) string {
	requestID := GetRequestID(ctx)
	operation := GetOperation(ctx)
	agentID := GetAgentID(ctx)
	provider := GetProvider(ctx)

	summary := fmt.Sprintf("Request[%s]", requestID)
	if operation != "" {
		summary += fmt.Sprintf(" Op[%s]", operation)
	}
	if agentID != "" {
		summary += fmt.Sprintf(" Agent[%s]", agentID)
	}
	if provider != "" {
		summary += fmt.Sprintf(" Provider[%s]", provider)
	}

	return summary
}
