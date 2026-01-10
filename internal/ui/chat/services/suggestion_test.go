// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents/core"
	"github.com/lancekrogers/guild-core/pkg/commission"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/memory"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/suggestions"
	"github.com/lancekrogers/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements core.EnhancedGuildArtisan for testing
type mockAgent struct {
	id              string
	suggestions     []suggestions.Suggestion
	suggestionError error
	executionResult *core.EnhancedExecutionResult
	executionError  error
}

// Implement interfaces.Agent methods
func (m *mockAgent) Execute(ctx context.Context, request string) (string, error) {
	if m.executionError != nil {
		return "", m.executionError
	}
	return "mock response", nil
}

func (m *mockAgent) GetID() string {
	return m.id
}

func (m *mockAgent) GetName() string {
	return "mock-agent"
}

func (m *mockAgent) GetType() string {
	return "mock"
}

func (m *mockAgent) GetCapabilities() []string {
	return []string{"test"}
}

// Implement core.GuildArtisan methods
func (m *mockAgent) GetToolRegistry() tools.Registry {
	return nil
}

func (m *mockAgent) GetCommissionManager() commission.CommissionManager {
	return nil
}

func (m *mockAgent) GetLLMClient() providers.LLMClient {
	return nil
}

func (m *mockAgent) GetMemoryManager() memory.ChainManager {
	return nil
}

// Implement core.EnhancedGuildArtisan methods
func (m *mockAgent) GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error) {
	if m.suggestionError != nil {
		return nil, m.suggestionError
	}
	return m.suggestions, nil
}

func (m *mockAgent) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*core.EnhancedExecutionResult, error) {
	if m.executionError != nil {
		return nil, m.executionError
	}
	return m.executionResult, nil
}

func (m *mockAgent) GetSuggestionManager() suggestions.SuggestionManager {
	return nil
}

// TestNewSuggestionService tests creation of suggestion service
func TestNewSuggestionService(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		handler *core.ChatSuggestionHandler
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name:    "Valid handler",
			handler: core.NewChatSuggestionHandler(&mockAgent{id: "test-agent"}),
			wantErr: false,
		},
		{
			name:    "Nil handler",
			handler: nil,
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewSuggestionService(ctx, tt.handler)

			if tt.wantErr {
				require.Error(t, err)
				var gerr *gerror.GuildError
				require.True(t, errors.As(err, &gerr))
				assert.Equal(t, tt.errCode, gerr.Code)
				assert.Nil(t, service)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, ctx, service.ctx)
				assert.NotNil(t, service.cache)
				assert.Equal(t, 5*time.Minute, service.cacheTTL)
				assert.Equal(t, 8192, service.tokenLimit)
				assert.Equal(t, 8192, service.tokenLimit)
			}
		})
	}
}

// TestGetSuggestions tests getting suggestions
func TestGetSuggestions(t *testing.T) {
	ctx := context.Background()

	mockSuggestions := []suggestions.Suggestion{
		{
			ID:          "1",
			Content:     "Test suggestion",
			Display:     "Test suggestion",
			Description: "A test suggestion",
			Type:        suggestions.SuggestionTypeCommand,
			Confidence:  0.9,
			Priority:    1,
			Source:      "test",
			CreatedAt:   time.Now(),
		},
	}

	mockAgent := &mockAgent{
		id:          "test-agent",
		suggestions: mockSuggestions,
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Test getting suggestions
	cmd := service.GetSuggestions("test message", nil)
	msg := cmd()

	suggestionsMsg, ok := msg.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.Equal(t, mockSuggestions, suggestionsMsg.Suggestions)
	assert.False(t, suggestionsMsg.FromCache)
	assert.Greater(t, suggestionsMsg.Latency, time.Duration(0))

	// Test cache hit
	cmd2 := service.GetSuggestions("test message", nil)
	msg2 := cmd2()

	cachedMsg, ok := msg2.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.True(t, cachedMsg.FromCache)
	assert.Equal(t, mockSuggestions, cachedMsg.Suggestions)
}

// TestGetSuggestionsWithError tests error handling in GetSuggestions
func TestGetSuggestionsWithError(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:              "test-agent",
		suggestionError: gerror.New(gerror.ErrCodeInternal, "test error", nil),
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	cmd := service.GetSuggestions("test message", nil)
	msg := cmd()

	errorMsg, ok := msg.(SuggestionServiceErrorMsg)
	require.True(t, ok)
	assert.Equal(t, "get_suggestions", errorMsg.Operation)
	assert.Error(t, errorMsg.Error)
}

// TestOptimizeContext tests context optimization for tokens
func TestOptimizeContext(t *testing.T) {
	ctx := context.Background()
	handler := core.NewChatSuggestionHandler(&mockAgent{id: "test"})
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	tests := []struct {
		name                string
		context             string
		tokenLimit          int
		expectReduction     bool
		minReductionPercent float64
		maxReductionPercent float64
	}{
		{
			name:                "Small context gets minimal optimization",
			context:             "This is a small context",
			tokenLimit:          100,
			expectReduction:     true,
			minReductionPercent: 10.0, // Very small contexts get 10% reduction
			maxReductionPercent: 20.0,
		},
		{
			name:                "Large context gets standard optimization",
			context:             string(make([]byte, 20000)), // ~5000 tokens
			tokenLimit:          1000,
			expectReduction:     true,
			minReductionPercent: 15.0, // Large contexts get 15-25% reduction
			maxReductionPercent: 25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.SetTokenLimit(tt.tokenLimit)
			result := service.OptimizeContext(tt.context)

			if tt.expectReduction {
				// Always expect some reduction with our new intelligent optimization
				assert.Less(t, len(result), len(tt.context))

				// Calculate actual reduction percentage
				originalTokens := len(tt.context) / 4
				optimizedTokens := len(result) / 4
				actualReduction := float64(originalTokens-optimizedTokens) / float64(originalTokens) * 100

				assert.GreaterOrEqual(t, actualReduction, tt.minReductionPercent,
					"Reduction %.2f%% should be at least %.2f%%", actualReduction, tt.minReductionPercent)
				assert.LessOrEqual(t, actualReduction, tt.maxReductionPercent,
					"Reduction %.2f%% should be at most %.2f%%", actualReduction, tt.maxReductionPercent)
			}
		})
	}
}

// TestCacheManagement tests cache operations
func TestCacheManagement(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id: "test-agent",
		suggestions: []suggestions.Suggestion{
			{ID: "1", Content: "Test", Display: "Test", Type: suggestions.SuggestionTypeCommand, Source: "test", CreatedAt: time.Now()},
		},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Set short TTL for testing
	service.SetCacheTTL(100 * time.Millisecond)

	// Get suggestions to populate cache
	cmd := service.GetSuggestions("test", nil)
	_ = cmd()

	// Verify cache hit
	stats := service.GetStats()
	assert.Equal(t, 1, stats["total_requests"])
	assert.Equal(t, 0, stats["cache_hits"])

	// Second request should hit cache
	cmd2 := service.GetSuggestions("test", nil)
	msg := cmd2()
	cached, ok := msg.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.True(t, cached.FromCache)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be cache miss now
	cmd3 := service.GetSuggestions("test", nil)
	msg3 := cmd3()
	notCached, ok := msg3.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.False(t, notCached.FromCache)

	// Test cache clear
	service.ClearCache()
	stats2 := service.GetStats()
	assert.Equal(t, 0, stats2["cache_size"])
}

// TestFollowUpSuggestions tests follow-up suggestion generation
func TestFollowUpSuggestions(t *testing.T) {
	ctx := context.Background()

	mockSuggestions := []suggestions.Suggestion{
		{
			ID:        "followup-1",
			Content:   "Follow-up suggestion",
			Display:   "Follow-up suggestion",
			Type:      suggestions.SuggestionTypeFollowUp,
			Source:    "test",
			CreatedAt: time.Now(),
		},
	}

	mockAgent := &mockAgent{
		id:          "test-agent",
		suggestions: mockSuggestions,
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	cmd := service.GetFollowUpSuggestions("previous message", "previous response")
	msg := cmd()

	suggestionsMsg, ok := msg.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.Equal(t, mockSuggestions, suggestionsMsg.Suggestions)
}

// TestStatistics tests statistics tracking
func TestStatistics(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:          "test-agent",
		suggestions: []suggestions.Suggestion{{ID: "1", Content: "Test", Display: "Test", Source: "test", CreatedAt: time.Now()}},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Make several requests
	for i := 0; i < 5; i++ {
		cmd := service.GetSuggestions("test", nil)
		_ = cmd()
	}

	stats := service.GetStats()
	assert.Equal(t, 5, stats["total_requests"])
	assert.Equal(t, 4, stats["cache_hits"]) // First is miss, rest are hits
	assert.Equal(t, 1, stats["cache_misses"])
	assert.Equal(t, "80.00%", stats["cache_hit_rate"])
	assert.Equal(t, 8192, stats["token_limit"]) // Check default token limit
}

// TestPeriodicCleanup tests periodic cache cleanup
func TestPeriodicCleanup(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{id: "test-agent"}
	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Test cleanup command
	cmd := service.StartPeriodicCleanup()
	assert.NotNil(t, cmd)

	// Test cleanup handler
	cleanupCmd := service.HandleCacheCleanup()
	msg := cleanupCmd()

	cleanedMsg, ok := msg.(CacheCleanedMsg)
	require.True(t, ok)
	assert.GreaterOrEqual(t, cleanedMsg.ItemsRemoved, 0)
}

// TestSuggestionContext tests suggestion context handling
func TestSuggestionContext(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:          "test-agent",
		suggestions: []suggestions.Suggestion{{ID: "1", Content: "Context-aware", Display: "Context-aware", Source: "test", CreatedAt: time.Now()}},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	context := &SuggestionContext{
		ConversationID: "conv-123",
		FileContext: &suggestions.FileContext{
			FilePath: "/test/file.go",
		},
		IsFollowUp: true,
	}

	cmd := service.GetSuggestions("test with context", context)
	msg := cmd()

	suggestionsMsg, ok := msg.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.NotNil(t, suggestionsMsg.Suggestions)
}

// TestTokenLimitManagement tests token limit tracking
func TestTokenLimitManagement(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id: "test-agent",
		suggestions: []suggestions.Suggestion{
			{ID: "1", Content: string(make([]byte, 1000)), Display: "Large", Description: string(make([]byte, 1000)), Source: "test", CreatedAt: time.Now()},
		},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Set low token limit
	service.SetTokenLimit(100)

	// Make request with large message
	largeMessage := string(make([]byte, 1000))
	cmd := service.GetSuggestions(largeMessage, nil)
	msg := cmd()

	suggestionsMsg, ok := msg.(SuggestionsReceivedMsg)
	require.True(t, ok)
	assert.Greater(t, suggestionsMsg.TokensUsed, 0)

	stats := service.GetStats()
	assert.Equal(t, 100, stats["token_limit"]) // Check modified token limit
}

// TestConcurrentAccess tests concurrent access to the service
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:          "test-agent",
		suggestions: []suggestions.Suggestion{{ID: "1", Content: "Concurrent", Display: "Concurrent", Source: "test", CreatedAt: time.Now()}},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	// Run concurrent operations
	done := make(chan bool, 3)

	// Concurrent gets
	go func() {
		for i := 0; i < 10; i++ {
			cmd := service.GetSuggestions("test", nil)
			_ = cmd()
		}
		done <- true
	}()

	// Concurrent stats
	go func() {
		for i := 0; i < 10; i++ {
			_ = service.GetStats()
		}
		done <- true
	}()

	// Concurrent cache operations
	go func() {
		for i := 0; i < 5; i++ {
			service.ClearCache()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify service is still functional
	finalCmd := service.GetSuggestions("final test", nil)
	finalMsg := finalCmd()
	_, ok := finalMsg.(SuggestionsReceivedMsg)
	assert.True(t, ok)
}

// TestStartCommand tests the Start command
func TestStartCommand(t *testing.T) {
	ctx := context.Background()

	mockAgent := &mockAgent{id: "test-agent"}
	handler := core.NewChatSuggestionHandler(mockAgent)
	service, err := NewSuggestionService(ctx, handler)
	require.NoError(t, err)

	cmd := service.Start()
	msg := cmd()

	startMsg, ok := msg.(SuggestionServiceStartedMsg)
	require.True(t, ok)
	assert.NotNil(t, startMsg.Config)
	assert.True(t, startMsg.Config.EnableSuggestions)
}

// BenchmarkGetSuggestions benchmarks suggestion retrieval
func BenchmarkGetSuggestions(b *testing.B) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:          "bench-agent",
		suggestions: []suggestions.Suggestion{{ID: "1", Content: "Benchmark", Display: "Benchmark", Source: "test", CreatedAt: time.Now()}},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, _ := NewSuggestionService(ctx, handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := service.GetSuggestions("benchmark test", nil)
		_ = cmd()
	}
}

// BenchmarkCacheHit benchmarks cache hit performance
func BenchmarkCacheHit(b *testing.B) {
	ctx := context.Background()

	mockAgent := &mockAgent{
		id:          "bench-agent",
		suggestions: []suggestions.Suggestion{{ID: "1", Content: "Benchmark", Display: "Benchmark", Source: "test", CreatedAt: time.Now()}},
	}

	handler := core.NewChatSuggestionHandler(mockAgent)
	service, _ := NewSuggestionService(ctx, handler)

	// Populate cache
	cmd := service.GetSuggestions("cached", nil)
	_ = cmd()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := service.GetSuggestions("cached", nil)
		_ = cmd()
	}
}
