// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package access

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/security/permissions"
	"github.com/lancekrogers/guild/pkg/tools"
)

// Mock implementations

type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogAllowed(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditLogger) LogDenied(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditLogger) LogExecution(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditLogger) Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]AuditEntry), args.Error(1)
}

func (m *MockAuditLogger) GetStats() AuditStats {
	args := m.Called()
	return args.Get(0).(AuditStats)
}

type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) PublishAccessEvent(ctx context.Context, event AccessEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventBus) PublishSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

type MockTool struct {
	mock.Mock
}

func (m *MockTool) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Schema() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tools.ToolResult), args.Error(1)
}

func (m *MockTool) Examples() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockTool) Category() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) RequiresAuth() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockTool) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

// Test cases

func TestCraftAccessController_CheckAccess_Allowed(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	mockEventBus := &MockEventBus{}

	// Set up permission model
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Create access controller
	ac := NewAccessController(ctx, pm, mockAuditor, mockEventBus)

	// Set up mock expectations
	mockAuditor.On("LogAllowed", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.AgentID == "agent1" && entry.Result == "allowed"
	})).Return(nil)

	// Create access request
	req := AccessRequest{
		AgentID:    "agent1",
		ToolName:   "file",
		Action:     "write",
		Parameters: map[string]interface{}{"path": "/project/main.go"},
		Timestamp:  time.Now(),
	}

	// Check access
	decision, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, "permission granted", decision.Reason)
	assert.Equal(t, "file:/project/main.go", decision.Resource)
	assert.Equal(t, "write", decision.Action)
	assert.False(t, decision.CacheHit) // First access, not cached

	mockAuditor.AssertExpectations(t)
}

func TestGuildAccessController_CheckAccess_Denied(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	mockEventBus := &MockEventBus{}

	// Set up permission model (read-only role)
	err := pm.AssignRole(ctx, "agent1", "read-only", "test")
	require.NoError(t, err)

	// Create access controller
	ac := NewAccessController(ctx, pm, mockAuditor, mockEventBus)

	// Set up mock expectations
	mockAuditor.On("LogDenied", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.AgentID == "agent1" && entry.Result == "denied"
	})).Return(nil)

	mockEventBus.On("PublishAccessEvent", mock.Anything, mock.MatchedBy(func(event AccessEvent) bool {
		return event.Type == "access.denied" && event.AgentID == "agent1"
	})).Return(nil)

	// Create access request (write operation, but agent only has read-only)
	req := AccessRequest{
		AgentID:    "agent1",
		ToolName:   "file",
		Action:     "write",
		Parameters: map[string]interface{}{"path": "/project/main.go"},
		Timestamp:  time.Now(),
	}

	// Check access
	decision, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "no matching permission")
	assert.Equal(t, "file:/project/main.go", decision.Resource)

	mockAuditor.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestJourneymanAccessController_CheckAccess_Cached(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	mockEventBus := &MockEventBus{}

	// Set up permission model
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Create access controller
	ac := NewAccessController(ctx, pm, mockAuditor, mockEventBus)

	// Set up mock expectations (should only be called once due to caching)
	mockAuditor.On("LogAllowed", mock.Anything, mock.Anything).Return(nil).Once()

	// Create access request
	req := AccessRequest{
		AgentID:    "agent1",
		ToolName:   "file",
		Action:     "read",
		Parameters: map[string]interface{}{"path": "/project/main.go"},
		Timestamp:  time.Now(),
	}

	// First access - should hit permission model
	decision1, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.True(t, decision1.Allowed)
	assert.False(t, decision1.CacheHit)

	// Second access - should be cached
	decision2, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.True(t, decision2.Allowed)
	assert.True(t, decision2.CacheHit)

	mockAuditor.AssertExpectations(t)
}

func TestCraftAccessController_BuildResourceIdentifier(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	ac := NewAccessController(ctx, pm, nil, nil)

	tests := []struct {
		name     string
		req      AccessRequest
		expected string
	}{
		{
			name: "file tool with path",
			req: AccessRequest{
				ToolName:   "file",
				Parameters: map[string]interface{}{"path": "/project/main.go"},
			},
			expected: "file:/project/main.go",
		},
		{
			name: "git tool with command",
			req: AccessRequest{
				ToolName:   "git",
				Parameters: map[string]interface{}{"command": "commit"},
			},
			expected: "git:commit",
		},
		{
			name: "shell tool with command",
			req: AccessRequest{
				ToolName:   "shell",
				Parameters: map[string]interface{}{"command": "ls -la"},
			},
			expected: "shell:ls -la",
		},
		{
			name: "unknown tool",
			req: AccessRequest{
				ToolName:   "unknown",
				Parameters: map[string]interface{}{},
			},
			expected: "unknown:*",
		},
		{
			name: "file tool without path",
			req: AccessRequest{
				ToolName:   "file",
				Parameters: map[string]interface{}{},
			},
			expected: "file:*",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ac.buildResourceIdentifier(test.req)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestGuildAccessController_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	ac := NewAccessController(ctx, pm, mockAuditor, nil)

	// Set up permission model
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	mockAuditor.On("LogAllowed", mock.Anything, mock.Anything).Return(nil)

	// Create access request
	req := AccessRequest{
		AgentID:    "agent1",
		ToolName:   "file",
		Action:     "read",
		Parameters: map[string]interface{}{"path": "/project/main.go"},
	}

	// Check access and cache
	decision1, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.False(t, decision1.CacheHit)

	// Second check should be cached
	decision2, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.True(t, decision2.CacheHit)

	// Invalidate cache for agent
	ac.InvalidateCache("agent1")

	// Third check should not be cached
	decision3, err := ac.CheckAccess(ctx, req)
	require.NoError(t, err)
	assert.False(t, decision3.CacheHit)
}

func TestJourneymanAccessController_GetStats(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	ac := NewAccessController(ctx, pm, nil, nil)

	// Get initial stats
	stats := ac.GetStats()
	assert.Equal(t, float64(0), stats.CacheHitRate)
	assert.Equal(t, 0, stats.CacheStats.Size)
}

func TestCraftToolInterceptor_Execute_Allowed(t *testing.T) {
	ctx := context.WithValue(context.Background(), "agent_id", "agent1")
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	mockTool := &MockTool{}

	// Set up permission model
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Create access controller and tool interceptor
	ac := NewAccessController(ctx, pm, mockAuditor, nil)
	interceptor := NewToolInterceptor(ctx, mockTool, ac, "file")

	// Set up mock expectations
	mockAuditor.On("LogAllowed", mock.Anything, mock.Anything).Return(nil)
	mockAuditor.On("LogExecution", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.Result == "success"
	})).Return(nil)

	expectedResult := &tools.ToolResult{
		Success: true,
		Output:  "file content",
	}
	mockTool.On("Execute", mock.Anything, "test input").Return(expectedResult, nil)

	// Execute tool
	result, err := interceptor.Execute(ctx, "test input")
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	mockTool.AssertExpectations(t)
	mockAuditor.AssertExpectations(t)
}

func TestGuildToolInterceptor_Execute_Denied(t *testing.T) {
	ctx := context.WithValue(context.Background(), "agent_id", "agent1")
	pm := permissions.NewPermissionModel(ctx)
	mockAuditor := &MockAuditLogger{}
	mockEventBus := &MockEventBus{}
	mockTool := &MockTool{}

	// Set up permission model (read-only role, but interceptor represents write operation)
	err := pm.AssignRole(ctx, "agent1", "read-only", "test")
	require.NoError(t, err)

	// Create access controller and tool interceptor
	ac := NewAccessController(ctx, pm, mockAuditor, mockEventBus)
	interceptor := NewToolInterceptor(ctx, mockTool, ac, "file")

	// Set up mock expectations
	mockAuditor.On("LogDenied", mock.Anything, mock.Anything).Return(nil)
	mockEventBus.On("PublishAccessEvent", mock.Anything, mock.Anything).Return(nil)

	// Execute tool (should be denied)
	result, err := interceptor.Execute(ctx, "test input")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")

	// Tool.Execute should not be called
	mockTool.AssertNotCalled(t, "Execute")
	mockAuditor.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestJourneymanToolInterceptor_NoAgentID(t *testing.T) {
	ctx := context.Background() // No agent_id in context
	pm := permissions.NewPermissionModel(ctx)
	mockTool := &MockTool{}

	ac := NewAccessController(ctx, pm, nil, nil)
	interceptor := NewToolInterceptor(ctx, mockTool, ac, "file")

	// Execute tool without agent ID should fail
	result, err := interceptor.Execute(ctx, "test input")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "agent ID not found in context")

	// Tool.Execute should not be called
	mockTool.AssertNotCalled(t, "Execute")
}

func TestCraftPermissionCache_BasicOperations(t *testing.T) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 100, 5*time.Minute)

	// Test miss
	decision := cache.Get("nonexistent")
	assert.Nil(t, decision)

	// Test set and get
	testDecision := &AccessDecision{
		Allowed:   true,
		Reason:    "test",
		Timestamp: time.Now(),
	}
	cache.Set("test-key", testDecision, 5*time.Minute)

	retrieved := cache.Get("test-key")
	assert.NotNil(t, retrieved)
	assert.Equal(t, testDecision.Allowed, retrieved.Allowed)
	assert.Equal(t, testDecision.Reason, retrieved.Reason)
}

func TestGuildPermissionCache_Eviction(t *testing.T) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 2, 5*time.Minute) // Small cache size

	// Fill cache to capacity
	decision1 := &AccessDecision{Allowed: true, Reason: "test1"}
	decision2 := &AccessDecision{Allowed: true, Reason: "test2"}
	decision3 := &AccessDecision{Allowed: true, Reason: "test3"}

	cache.Set("key1", decision1, 5*time.Minute)
	cache.Set("key2", decision2, 5*time.Minute)

	// Both should be in cache
	assert.NotNil(t, cache.Get("key1"))
	assert.NotNil(t, cache.Get("key2"))

	// Adding third should evict one
	cache.Set("key3", decision3, 5*time.Minute)

	// Should still have key3
	assert.NotNil(t, cache.Get("key3"))

	// At least one of the others should be evicted
	stats := cache.GetStats()
	assert.Greater(t, stats.Evictions, int64(0))
}

func TestJourneymanPermissionCache_Expiration(t *testing.T) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 100, 1*time.Millisecond) // Very short TTL

	testDecision := &AccessDecision{
		Allowed: true,
		Reason:  "test",
	}

	cache.Set("test-key", testDecision, 1*time.Millisecond)

	// Should be available immediately
	retrieved := cache.Get("test-key")
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	retrieved = cache.Get("test-key")
	assert.Nil(t, retrieved)
}

func TestCraftPermissionCache_InvalidateAgent(t *testing.T) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 100, 5*time.Minute)

	// Add decisions for multiple agents
	agent1Decision := &AccessDecision{AgentID: "agent1", Allowed: true}
	agent2Decision := &AccessDecision{AgentID: "agent2", Allowed: true}

	cache.Set("agent1-key", agent1Decision, 5*time.Minute)
	cache.Set("agent2-key", agent2Decision, 5*time.Minute)

	// Both should be in cache
	assert.NotNil(t, cache.Get("agent1-key"))
	assert.NotNil(t, cache.Get("agent2-key"))

	// Invalidate agent1
	cache.InvalidateAgent("agent1")

	// Agent1 decision should be gone, agent2 should remain
	assert.Nil(t, cache.Get("agent1-key"))
	assert.NotNil(t, cache.Get("agent2-key"))
}

func TestGuildPermissionCache_Stats(t *testing.T) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 100, 5*time.Minute)

	// Check initial stats
	stats := cache.GetStats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)

	// Add some entries and perform operations
	testDecision := &AccessDecision{Allowed: true}
	cache.Set("key1", testDecision, 5*time.Minute)
	cache.Set("key2", testDecision, 5*time.Minute)

	// Test hits and misses
	cache.Get("key1")    // hit
	cache.Get("key2")    // hit
	cache.Get("missing") // miss

	// Check updated stats
	stats = cache.GetStats()
	assert.Equal(t, 2, stats.Size)
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)

	// Check hit rate
	hitRate := cache.GetHitRate()
	assert.InDelta(t, 2.0/3.0, hitRate, 0.01)
}

func TestJourneymanAccessController_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	ac := NewAccessController(ctx, pm, nil, nil)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	req := AccessRequest{
		AgentID:  "agent1",
		ToolName: "file",
		Action:   "read",
	}

	// Should handle cancellation gracefully
	decision, err := ac.CheckAccess(cancelledCtx, req)
	assert.Error(t, err)
	assert.Nil(t, decision)
	assert.Contains(t, err.Error(), "context cancelled")
}

// Benchmark tests

func BenchmarkCraftAccessController_CheckAccess(b *testing.B) {
	ctx := context.Background()
	pm := permissions.NewPermissionModel(ctx)
	ac := NewAccessController(ctx, pm, nil, nil)

	// Set up permission
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(b, err)

	req := AccessRequest{
		AgentID:    "agent1",
		ToolName:   "file",
		Action:     "read",
		Parameters: map[string]interface{}{"path": "/project/main.go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.CheckAccess(ctx, req)
	}
}

func BenchmarkJourneymanPermissionCache_GetSet(b *testing.B) {
	ctx := context.Background()
	cache := NewPermissionCache(ctx, 1000, 5*time.Minute)

	decision := &AccessDecision{
		Allowed: true,
		Reason:  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100) // Rotate through 100 keys
		cache.Set(key, decision, 5*time.Minute)
		cache.Get(key)
	}
}
