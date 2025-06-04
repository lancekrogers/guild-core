package prompts_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/prompts"
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

func (m *mockManager) FormatContext(ctx context.Context, context prompts.Context) (string, error) {
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

func (m *mockFormatter) FormatAsXML(ctx prompts.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockFormatter) FormatAsMarkdown(ctx prompts.Context) (string, error) {
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
) ([]prompts.MemoryChunk, error) {
	args := m.Called(ctx, sessionID, query, maxTokens, threshold)
	return args.Get(0).([]prompts.MemoryChunk), args.Error(1)
}

func TestLayeredPromptAssembler(t *testing.T) {
	t.Run("BuildPrompt_Success", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}
		
		// Create assembler
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
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
			[]prompts.MemoryChunk{
				{
					Content: "Previous authentication implementation used JWT tokens...",
					Score:   0.85,
					Source:  "previous_task",
					Tokens:  20,
				},
			}, nil)
		
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
			case prompts.LayerPlatform:
				foundPlatform = true
				assert.Contains(t, layer.Content, "Guild Framework")
			case prompts.LayerRole:
				foundRole = true
				assert.Contains(t, layer.Content, "backend artisan")
			case prompts.LayerTurn:
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
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 200, // Very small budget
		)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
			UserMessage: "Simple request",
		}
		
		// Setup expectations - return very long prompts
		longPrompt := strings.Repeat("This is a very long prompt that will exceed the token budget. ", 50)
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(longPrompt, nil)
		
		// No RAG retrieval for this test
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Simple request", 40, 0.7).Return(
			[]prompts.MemoryChunk{}, nil)
		
		// Execute
		ctx := context.Background()
		result, err := assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)
		
		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Truncated, "Prompt should be truncated due to token budget")
		assert.True(t, result.TokenCount <= 200, "Token count should be within budget")
		
		manager.AssertExpectations(t)
		ragRetriever.AssertExpectations(t)
	})
	
	t.Run("BuildPrompt_WithCache", func(t *testing.T) {
		// Setup mocks
		manager := &mockManager{}
		formatter := &mockFormatter{}
		store := &mockStore{}
		ragRetriever := &mockRAGRetriever{}
		
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
			UserMessage: "Test request",
		}
		
		// Setup expectations for first call
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Backend artisan prompt", nil).Once()
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Test request", 800, 0.7).Return(
			[]prompts.MemoryChunk{}, nil).Once()
		
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
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, nil, 4000,
		)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
			UserMessage: "Test without RAG",
		}
		
		// Setup expectations
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Backend artisan prompt", nil)
		
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
		
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
			UserMessage: "Test error handling",
		}
		
		// Setup expectations - manager returns error
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"", assert.AnError)
		
		// RAG should still be called
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, "Test error handling", 800, 0.7).Return(
			[]prompts.MemoryChunk{}, nil)
		
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
		
		assembler := prompts.NewLayeredPromptAssembler(
			manager, formatter, store, ragRetriever, 4000,
		)
		
		// Test data with complex context
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := prompts.TurnContext{
			UserMessage:  "Complex request with multiple instructions",
			TaskID:       "TASK-001",
			CommissionID: "COMM-001",
			Urgency:      "medium",
			Instructions: []string{"Be detailed", "Include examples"},
		}
		
		// Setup expectations
		manager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"Role prompt content", nil)
		manager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Domain prompt content", nil)
		ragRetriever.On("GetContextualMemory", mock.Anything, sessionID, mock.Anything, mock.Anything, mock.Anything).Return(
			[]prompts.MemoryChunk{}, nil)
		
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