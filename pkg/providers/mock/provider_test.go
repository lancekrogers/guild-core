package mock

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

func TestMockProviderEnvironmentActivation(t *testing.T) {
	t.Run("Provider disabled without environment variable", func(t *testing.T) {
		// Ensure environment variable is not set
		os.Unsetenv("GUILD_MOCK_PROVIDER")
		
		provider, err := NewProvider()
		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.False(t, provider.enabled)
		
		// Should error when trying to use
		req := interfaces.ChatRequest{
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Test message"},
			},
		}
		
		_, err = provider.ChatCompletion(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})
	
	t.Run("Provider enabled with environment variable", func(t *testing.T) {
		// Enable mock provider
		os.Setenv("GUILD_MOCK_PROVIDER", "true")
		defer os.Unsetenv("GUILD_MOCK_PROVIDER")
		
		provider, err := NewProvider()
		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.True(t, provider.enabled)
		
		// Should work when enabled
		req := interfaces.ChatRequest{
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Test message"},
			},
		}
		
		resp, err := provider.ChatCompletion(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Choices)
	})
}

func TestMockProviderYAMLPatternMatching(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)
	
	tests := []struct {
		name     string
		prompt   string
		expected string
	}{
		{
			name:     "commission pattern",
			prompt:   "Create a commission for building a REST API",
			expected: "Commission Analysis",
		},
		{
			name:     "implementation pattern",
			prompt:   "Implement the user authentication service",
			expected: "implement this with clean",
		},
		{
			name:     "test pattern",
			prompt:   "Write unit tests for the service",
			expected: "comprehensive tests",
		},
		{
			name:     "architecture pattern",
			prompt:   "Design a system architecture for microservices",
			expected: "System Architecture",
		},
		{
			name:     "error pattern",
			prompt:   "There's a bug in the authentication code",
			expected: "Error Analysis",
		},
		{
			name:     "help pattern",
			prompt:   "Can you help me with this feature?",
			expected: "I can assist with",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := interfaces.ChatRequest{
				Messages: []interfaces.ChatMessage{
					{Role: "user", Content: tt.prompt},
				},
				Model: "mock-model-v1",
			}
			
			resp, err := provider.ChatCompletion(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotEmpty(t, resp.Choices)
			
			content := resp.Choices[0].Message.Content
			assert.Contains(t, content, tt.expected, "Response should contain expected pattern match")
			assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
			assert.Equal(t, "stop", resp.Choices[0].FinishReason)
		})
	}
}

func TestMockProviderDefaultResponse(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	// Test with unmatched prompt
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Random unmatched prompt that should not match any pattern"},
		},
		Model: "mock-model-v1",
	}
	
	resp, err := provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Choices)
	
	content := resp.Choices[0].Message.Content
	assert.Contains(t, content, "testing purposes", "Should use default Guild response")
	assert.Contains(t, content, "Analysis", "Should contain structured response")
}

func TestMockProviderStreaming(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Help me design a system"},
		},
		Model: "mock-model-v1",
	}
	
	stream, err := provider.StreamChatCompletion(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)
	
	// Collect streamed chunks
	var chunks []string
	for {
		chunk, err := stream.Next()
		if err != nil {
			break // Should be io.EOF when done
		}
		chunks = append(chunks, chunk.Delta.Content)
	}
	
	// Verify we got multiple chunks
	assert.Greater(t, len(chunks), 3, "Should receive multiple streaming chunks")
	
	// Verify content is meaningful
	fullContent := strings.Join(chunks, "")
	assert.NotEmpty(t, fullContent)
	assert.Contains(t, fullContent, "design", "Content should be relevant to request")
}

func TestMockProviderCapabilities(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	caps := provider.GetCapabilities()
	
	assert.True(t, caps.SupportsStream, "Should support streaming")
	assert.True(t, caps.SupportsEmbeddings, "Should support embeddings")
	assert.Equal(t, 8192, caps.ContextWindow, "Should have correct context window")
	assert.Len(t, caps.Models, 3, "Should have 3 mock models")
	
	// Check model names
	modelNames := make([]string, len(caps.Models))
	for i, model := range caps.Models {
		modelNames[i] = model.ID
	}
	assert.Contains(t, modelNames, "mock-model-v1")
	assert.Contains(t, modelNames, "mock-model-fast")
	assert.Contains(t, modelNames, "mock-model-smart")
}

func TestMockProviderEmbeddings(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	req := interfaces.EmbeddingRequest{
		Model: "mock-model-v1",
		Input: []string{"test text 1", "test text 2"},
	}
	
	resp, err := provider.CreateEmbedding(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	
	assert.Len(t, resp.Embeddings, 2, "Should return embeddings for both inputs")
	assert.Equal(t, req.Model, resp.Model)
	
	// Check embedding structure
	for i, embedding := range resp.Embeddings {
		assert.Equal(t, i, embedding.Index)
		assert.Len(t, embedding.Embedding, 384, "Should return 384-dimensional embeddings")
		
		// Check that embeddings are deterministic but not all zeros
		nonZero := false
		for _, val := range embedding.Embedding {
			if val != 0 {
				nonZero = true
				break
			}
		}
		assert.True(t, nonZero, "Embeddings should contain non-zero values")
	}
}

func TestMockProviderContextCancellation(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	// Set a delay to test cancellation
	provider.SetDelay(100 * time.Millisecond)
	
	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "This should be cancelled"},
		},
	}
	
	_, err = provider.ChatCompletion(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestMockProviderBuilderPattern(t *testing.T) {
	// Test legacy builder pattern still works
	builder, err := NewBuilder()
	require.NoError(t, err)
	
	provider := builder.
		WithResponse("test", "custom response").
		WithDefaultResponse("custom default").
		WithDelay(50 * time.Millisecond).
		Build()
	
	// Force enable for testing
	provider.enabled = true
	
	// Test custom response
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}
	
	resp, err := provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.Choices[0].Message.Content, "custom response")
}

func TestMockProviderLegacyCompatibility(t *testing.T) {
	// Test the legacy testing interface
	provider := NewProviderForTesting()
	assert.True(t, provider.enabled, "Should be enabled for testing")
	
	// Test Complete method (legacy interface)
	response, err := provider.Complete(context.Background(), "test prompt")
	require.NoError(t, err)
	assert.NotEmpty(t, response)
	
	// Test call recording
	calls := provider.GetCalls()
	assert.Len(t, calls, 1, "Should record the call")
	assert.Equal(t, "ChatCompletion", calls[0].Method)
}

func TestMockProviderTokenCounting(t *testing.T) {
	// Enable mock provider for testing
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")
	
	provider, err := NewProvider()
	require.NoError(t, err)
	
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Create a commission for building an API"},
		},
		Model: "mock-model-v1",
	}
	
	resp, err := provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	
	// Should use tokens from YAML configuration for commission pattern
	assert.Equal(t, 250, resp.Usage.CompletionTokens, "Should use YAML-defined token count")
	assert.Greater(t, resp.Usage.TotalTokens, 0, "Should calculate total tokens")
}