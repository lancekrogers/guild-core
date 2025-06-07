package context

import (
	"context"
	"testing"
	"time"
)

func TestNewGuildContext(t *testing.T) {
	ctx := NewGuildContext(context.Background())

	// Check that request ID is set
	requestID := GetRequestID(ctx)
	if requestID == "" {
		t.Error("Expected request ID to be set")
	}

	// Check that trace ID is set
	traceID := GetTraceID(ctx)
	if traceID == "" {
		t.Error("Expected trace ID to be set")
	}

	// Check that span ID is set
	spanID := GetSpanID(ctx)
	if spanID == "" {
		t.Error("Expected span ID to be set")
	}

	// Check that request info is available
	requestInfo := GetRequestInfo(ctx)
	if requestInfo == nil {
		t.Error("Expected request info to be available")
	}

	if requestInfo.ID != requestID {
		t.Errorf("Request info ID mismatch: got %s, want %s", requestInfo.ID, requestID)
	}
}

func TestNewOperationContext(t *testing.T) {
	parentCtx := NewGuildContext(context.Background())

	// Test with timeout
	opCtx, cancel := NewOperationContext(parentCtx, "test-operation", 5*time.Second)
	defer cancel()

	operation := GetOperation(opCtx)
	if operation != "test-operation" {
		t.Errorf("Expected operation 'test-operation', got %s", operation)
	}

	// Check that parent context values are preserved
	requestID := GetRequestID(opCtx)
	parentRequestID := GetRequestID(parentCtx)
	if requestID != parentRequestID {
		t.Error("Request ID should be preserved from parent context")
	}

	// Test without timeout
	opCtx2, cancel2 := NewOperationContext(parentCtx, "no-timeout", 0)
	defer cancel2()

	operation2 := GetOperation(opCtx2)
	if operation2 != "no-timeout" {
		t.Errorf("Expected operation 'no-timeout', got %s", operation2)
	}
}

func TestRegistryContextMethods(t *testing.T) {
	ctx := context.Background()

	// Test getting registry when none exists
	_, err := GetRegistry(ctx)
	if err == nil {
		t.Error("Expected error when getting registry from empty context")
	}

	// Note: We can't test WithRegistry/GetRegistry properly without a real registry
	// This will be tested when we integrate with the actual registry package
}

func TestRequestTracking(t *testing.T) {
	ctx := context.Background()

	// Test request ID
	ctx = WithRequestID(ctx, "test-request-123")
	requestID := GetRequestID(ctx)
	if requestID != "test-request-123" {
		t.Errorf("Expected request ID 'test-request-123', got %s", requestID)
	}

	// Test session ID
	ctx = WithSessionID(ctx, "session-456")
	sessionID := GetSessionID(ctx)
	if sessionID != "session-456" {
		t.Errorf("Expected session ID 'session-456', got %s", sessionID)
	}

	// Test operation
	ctx = WithOperation(ctx, "test-operation")
	operation := GetOperation(ctx)
	if operation != "test-operation" {
		t.Errorf("Expected operation 'test-operation', got %s", operation)
	}
}

func TestComponentIdentification(t *testing.T) {
	ctx := context.Background()

	// Test agent ID
	ctx = WithAgentID(ctx, "agent-001")
	agentID := GetAgentID(ctx)
	if agentID != "agent-001" {
		t.Errorf("Expected agent ID 'agent-001', got %s", agentID)
	}

	// Test provider
	ctx = WithProvider(ctx, "openai")
	provider := GetProvider(ctx)
	if provider != "openai" {
		t.Errorf("Expected provider 'openai', got %s", provider)
	}

	// Test tool
	ctx = WithTool(ctx, "file-tool")
	tool := GetTool(ctx)
	if tool != "file-tool" {
		t.Errorf("Expected tool 'file-tool', got %s", tool)
	}
}

func TestTracingAndDebugging(t *testing.T) {
	ctx := context.Background()

	// Test trace ID
	ctx = WithTraceID(ctx, "trace-123")
	traceID := GetTraceID(ctx)
	if traceID != "trace-123" {
		t.Errorf("Expected trace ID 'trace-123', got %s", traceID)
	}

	// Test span ID
	ctx = WithSpanID(ctx, "span-456")
	spanID := GetSpanID(ctx)
	if spanID != "span-456" {
		t.Errorf("Expected span ID 'span-456', got %s", spanID)
	}

	// Test logger (when none is set)
	logger := GetLogger(ctx)
	if logger != nil {
		t.Error("Expected nil logger when none is set")
	}

	// Note: Testing WithLogger/GetLogger properly requires a logger implementation
}

func TestCostAndResourceTracking(t *testing.T) {
	ctx := context.Background()

	// Test cost budget
	ctx = WithCostBudget(ctx, 10.0, "USD")
	costInfo := GetCostInfo(ctx)
	if costInfo == nil {
		t.Error("Expected cost info to be set")
	}
	if costInfo.Budget != 10.0 {
		t.Errorf("Expected budget 10.0, got %f", costInfo.Budget)
	}
	if costInfo.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got %s", costInfo.Currency)
	}
	if costInfo.Used != 0.0 {
		t.Errorf("Expected used amount 0.0, got %f", costInfo.Used)
	}

	// Test resource limits
	ctx = WithResourceLimit(ctx, 5, 1024*1024*1024) // 5 concurrent, 1GB memory
	resourceInfo := GetResourceInfo(ctx)
	if resourceInfo == nil {
		t.Error("Expected resource info to be set")
	}
	if resourceInfo.MaxConcurrency != 5 {
		t.Errorf("Expected max concurrency 5, got %d", resourceInfo.MaxConcurrency)
	}
	if resourceInfo.MemoryLimit != 1024*1024*1024 {
		t.Errorf("Expected memory limit 1GB, got %d", resourceInfo.MemoryLimit)
	}
}

func TestLogFields(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")
	ctx = WithAgentID(ctx, "agent-789")
	ctx = WithProvider(ctx, "anthropic")
	ctx = WithOperation(ctx, "completion")

	fields := LogFields(ctx)

	// Check that we have the expected number of fields (key-value pairs)
	expectedFieldCount := 10 // 5 fields * 2 (key + value)
	if len(fields) != expectedFieldCount {
		t.Errorf("Expected %d log fields, got %d", expectedFieldCount, len(fields))
	}

	// Check that specific fields exist
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fieldMap[fields[i].(string)] = fields[i+1]
		}
	}

	if fieldMap["request_id"] != "req-123" {
		t.Errorf("Expected request_id 'req-123', got %v", fieldMap["request_id"])
	}
	if fieldMap["trace_id"] != "trace-456" {
		t.Errorf("Expected trace_id 'trace-456', got %v", fieldMap["trace_id"])
	}
	if fieldMap["agent_id"] != "agent-789" {
		t.Errorf("Expected agent_id 'agent-789', got %v", fieldMap["agent_id"])
	}
	if fieldMap["provider"] != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %v", fieldMap["provider"])
	}
	if fieldMap["operation"] != "completion" {
		t.Errorf("Expected operation 'completion', got %v", fieldMap["operation"])
	}
}

func TestContextSummary(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithOperation(ctx, "test-op")
	ctx = WithAgentID(ctx, "agent-456")
	ctx = WithProvider(ctx, "openai")

	summary := ContextSummary(ctx)
	expected := "Request[req-123] Op[test-op] Agent[agent-456] Provider[openai]"

	if summary != expected {
		t.Errorf("Expected summary '%s', got '%s'", expected, summary)
	}

	// Test with minimal context
	minCtx := WithRequestID(context.Background(), "minimal")
	minSummary := ContextSummary(minCtx)
	expectedMin := "Request[minimal]"

	if minSummary != expectedMin {
		t.Errorf("Expected minimal summary '%s', got '%s'", expectedMin, minSummary)
	}
}

func TestEmptyContextValues(t *testing.T) {
	ctx := context.Background()

	// Test that empty contexts return empty strings/nil values
	if GetRequestID(ctx) != "" {
		t.Error("Expected empty request ID from empty context")
	}
	if GetSessionID(ctx) != "" {
		t.Error("Expected empty session ID from empty context")
	}
	if GetOperation(ctx) != "" {
		t.Error("Expected empty operation from empty context")
	}
	if GetAgentID(ctx) != "" {
		t.Error("Expected empty agent ID from empty context")
	}
	if GetProvider(ctx) != "" {
		t.Error("Expected empty provider from empty context")
	}
	if GetTool(ctx) != "" {
		t.Error("Expected empty tool from empty context")
	}
	if GetTraceID(ctx) != "" {
		t.Error("Expected empty trace ID from empty context")
	}
	if GetSpanID(ctx) != "" {
		t.Error("Expected empty span ID from empty context")
	}
	if GetLogger(ctx) != nil {
		t.Error("Expected nil logger from empty context")
	}
	if GetCostInfo(ctx) != nil {
		t.Error("Expected nil cost info from empty context")
	}
	if GetResourceInfo(ctx) != nil {
		t.Error("Expected nil resource info from empty context")
	}
}
