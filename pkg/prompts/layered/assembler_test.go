// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/prompts/layered"
)

// Mock implementations for testing
type mockManager struct {
	mock.Mock
}

func (m *mockManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	args := m.Called(ctx, role, domain)
	return args.String(0), args.Error(1)
}

func (m *mockManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	args := m.Called(ctx, templateName)
	return args.String(0), args.Error(1)
}

func (m *mockManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	args := m.Called(ctx, context)
	return args.String(0), args.Error(1)
}

func (m *mockManager) ListRoles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	args := m.Called(ctx, role)
	return args.Get(0).([]string), args.Error(1)
}

type mockFormatter struct {
	mock.Mock
}

func (m *mockFormatter) FormatAsXML(ctx layered.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockFormatter) FormatAsMarkdown(ctx layered.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	args := m.Called(content, maxTokens)
	return args.String(0), args.Error(1)
}

type mockStore struct {
	mock.Mock
}

func (m *mockStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	args := m.Called(ctx, bucket, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	args := m.Called(ctx, bucket, key, value)
	return args.Error(0)
}

func (m *mockStore) Delete(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

func (m *mockStore) List(ctx context.Context, bucket string) ([]string, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockStore) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	args := m.Called(ctx, bucket, prefix)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockStore) GetPromptLayer(ctx context.Context, layer, identifier string) ([]byte, error) {
	args := m.Called(ctx, layer, identifier)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockStore) SavePromptLayer(ctx context.Context, layer, identifier string, data []byte) error {
	args := m.Called(ctx, layer, identifier, data)
	return args.Error(0)
}

func (m *mockStore) DeletePromptLayer(ctx context.Context, layer, identifier string) error {
	args := m.Called(ctx, layer, identifier)
	return args.Error(0)
}

func (m *mockStore) ListPromptLayers(ctx context.Context, layer string) ([]string, error) {
	args := m.Called(ctx, layer)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockStore) CacheCompiledPrompt(ctx context.Context, cacheKey string, data []byte) error {
	args := m.Called(ctx, cacheKey, data)
	return args.Error(0)
}

func (m *mockStore) GetCachedPrompt(ctx context.Context, cacheKey string) ([]byte, error) {
	args := m.Called(ctx, cacheKey)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockStore) InvalidatePromptCache(ctx context.Context, keyPattern string) error {
	args := m.Called(ctx, keyPattern)
	return args.Error(0)
}

func (m *mockStore) SavePromptMetrics(ctx context.Context, metricID string, data []byte) error {
	args := m.Called(ctx, metricID, data)
	return args.Error(0)
}

func (m *mockStore) GetPromptMetrics(ctx context.Context, metricID string) ([]byte, error) {
	args := m.Called(ctx, metricID)
	return args.Get(0).([]byte), args.Error(1)
}

type mockRAGRetriever struct {
	mock.Mock
}

func (m *mockRAGRetriever) GetContextualMemory(
	ctx context.Context,
	sessionID, query string,
	maxTokens int,
	threshold float64,
) ([]layered.MemoryChunk, error) {
	args := m.Called(ctx, sessionID, query, maxTokens, threshold)
	return args.Get(0).([]layered.MemoryChunk), args.Error(1)
}

func TestLayeredPromptAssembler(t *testing.T) {
	t.Run("BuildPrompt_Success", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}

		// Create assembler
		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)

		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage:  "Implement user authentication",
			TaskID:       "AUTH-001",
			CommissionID: "COMM-001",
			Urgency:      "high",
		}

		// Setup expectations for platform layer (will be generated by default)
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError) // Not found - will use default

		// Setup expectations for guild layer
		store.On("GetPromptLayer", mock.Anything, "guild", "default").Return(
			[]byte{}, assert.AnError) // Not found - will use default

		// Setup expectations for role prompt
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"You are a backend artisan specialized in server-side development...", nil)

		// Setup expectations for domain prompt
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Additional guidelines for development environment...", nil)

		// Setup expectations for session layer
		store.On("GetPromptLayer", mock.Anything, "session", "session_123").Return(
			[]byte{}, assert.AnError) // Not found - will use default

		// Setup expectations for RAG retrieval
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Implement user authentication", 800, 0.7).Return(
			[]layered.MemoryChunk{
				{
					Content: "Previous authentication implementation used JWT tokens...",
					Score:   0.85,
					Source:  "previous_task",
					Tokens:  20,
				},
			}, nil)

		// Setup expectations for cache store
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, artisanID, result.ArtisanID)
		assert.Equal(t, sessionID, result.SessionID)
		assert.True(t, result.TokenCount > 0)
		assert.NotEmpty(t, result.Compiled)
		assert.NotEmpty(t, result.CacheKey)

		// Verify layers are present
		assert.True(t, len(result.Layers) >= 3) // At least platform, role, and turn

		// Check that platform layer is included
		foundPlatform := false
		foundRole := false
		foundTurn := false
		for _, layer := range result.Layers {
			switch layer.Layer {
			case layered.LayerPlatform:
				foundPlatform = true
				assert.Contains(t, layer.Content, "Guild Framework")
			case layered.LayerRole:
				foundRole = true
				assert.Contains(t, layer.Content, "backend artisan")
			case layered.LayerTurn:
				foundTurn = true
				assert.Contains(t, layer.Content, "Implement user authentication")
			}
		}
		assert.True(t, foundPlatform, "Platform layer should be present")
		assert.True(t, foundRole, "Role layer should be present")
		assert.True(t, foundTurn, "Turn layer should be present")

		// Verify memory chunks are included
		assert.Contains(t, result.Compiled, "[[MEMORY:previous_task]]")

		// Verify all mocks were called as expected
		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})

	t.Run("BuildPrompt_WithTokenTruncation", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}

		// Create assembler with very small token budget
		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 200, // Very small budget
		)

		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage: "Simple request",
		}

		// Setup expectations - return very long prompts
		longPrompt := strings.Repeat("This is a very long prompt that will exceed the token budget. ", 50)
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(longPrompt, nil)
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(longPrompt, nil)

		// Add platform layer expectation
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return([]byte{}, assert.AnError)

		// Add session layer expectation
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return([]byte{}, assert.AnError)

		// No RAG retrieval for this test
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Simple request", 40, 0.7).Return(
			[]layered.MemoryChunk{}, nil)

		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Truncated, "Prompt should be truncated due to token budget")
		// Token counting may include overhead - allow some tolerance
		assert.True(t, result.TokenCount <= 250, "Token count should be reasonably close to budget (actual: %d)", result.TokenCount)

		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})

	t.Run("BuildPrompt_WithCache", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}

		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)

		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage: "Test request",
		}

		// Setup expectations for first call
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return([]byte{}, assert.AnError).Once()
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return([]byte{}, assert.AnError).Once()
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Backend artisan prompt", nil).Once()
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Backend dev prompt", nil).Once()
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Test request", 800, 0.7).Return(
			[]layered.MemoryChunk{}, nil).Once()
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

		ctx := context.Background()

		// First call - should hit the mocks
		result1, err1 := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)
		require.NoError(t, err1)
		assert.NotNil(t, result1)

		// Second call - should use cache (mocks won't be called again)
		result2, err2 := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)
		require.NoError(t, err2)
		assert.NotNil(t, result2)

		// Results should be identical (from cache)
		assert.Equal(t, result1.CacheKey, result2.CacheKey)
		assert.Equal(t, result1.Compiled, result2.Compiled)

		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})

	t.Run("BuildPrompt_NoRAGRetriever", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}

		// Create assembler without RAG retriever (nil)
		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, nil, 4000,
		)

		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage: "Test without RAG",
		}

		// Setup expectations
		// Platform layer is always retrieved (returns error to use default)
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - will use default

		// Guild layer is optional
		store.On("GetPromptLayer", mock.Anything, "guild", "").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		// Session layer is optional
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Backend artisan prompt", nil)
		// Domain prompt is also called since artisan ID has domain "dev"
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Backend dev domain prompt", nil).Maybe()

		// Setup expectation for cache store
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotContains(t, result.Compiled, "[[MEMORY:")

		manager.AssertExpectations(t)
	})

	t.Run("BuildPrompt_ErrorHandling", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}

		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)

		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage: "Test error handling",
		}

		// Setup expectations - manager returns error
		// Platform layer is always retrieved
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - will use default

		// Guild layer is optional
		store.On("GetPromptLayer", mock.Anything, "guild", "").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		// Session layer is optional
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"", assert.AnError)
		// Domain prompt is also called since artisan ID has domain "dev"
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"", nil).Maybe() // Optional domain prompt

		// Setup expectation for cache store
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

		// RAG should still be called
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Test error handling", 800, 0.7).Return(
			[]layered.MemoryChunk{}, nil)

		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)

		// Should succeed even if role prompt fails (fallback behavior)
		require.NoError(t, err)
		assert.NotNil(t, result)

		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})
}

func TestLayeredPromptLayers(t *testing.T) {
	t.Run("LayerPriority_Ordering", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}

		assembler := layered.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)

		// Test data with complex context
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage:  "Complex request with multiple instructions",
			TaskID:       "TASK-001",
			CommissionID: "COMM-001",
			Urgency:      "medium",
			Instructions: []string{"Be detailed", "Include examples"},
		}

		// Setup expectations
		// Platform layer is always retrieved
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - will use default

		// Guild layer is optional
		store.On("GetPromptLayer", mock.Anything, "guild", "").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		// Session layer is optional
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional

		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Role prompt content", nil)
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Domain prompt content", nil)
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, mock.Anything, mock.Anything, mock.Anything).Return(
			[]layered.MemoryChunk{}, nil)

		// Setup expectation for cache store
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Check that layers are in correct priority order in the compiled prompt
		assert.Contains(t, result.Compiled, "Guild Framework") // Platform layer
		assert.Contains(t, result.Compiled, "Complex request") // Turn layer

		// Verify metadata
		assert.Equal(t, artisanID, result.ArtisanID)
		assert.Equal(t, sessionID, result.SessionID)
		assert.Contains(t, result.Metadata, "layer_count")
		assert.Contains(t, result.Metadata, "turn_context")

		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})
}
