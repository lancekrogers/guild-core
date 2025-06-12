package base

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/stretchr/testify/assert"
)

func TestNewOpenAICompatible(t *testing.T) {
	// Test URL normalization
	provider := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		"https://api.test.com",
		map[string]string{},
		interfaces.ProviderCapabilities{},
	)
	assert.Equal(t, "https://api.test.com/", provider.baseURL)
	assert.NotNil(t, provider.client)

	// Test with trailing slash
	provider2 := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		"https://api.test.com/",
		map[string]string{},
		interfaces.ProviderCapabilities{},
	)
	assert.Equal(t, "https://api.test.com/", provider2.baseURL)
}

func TestChatCompletionBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "Test response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		server.URL,
		map[string]string{},
		interfaces.ProviderCapabilities{},
	)

	resp, err := provider.ChatCompletion(context.Background(), interfaces.ChatRequest{
		Model: "test-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "Test response", resp.Choices[0].Message.Content)
}

func TestStreamChatCompletionBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send simple SSE data
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		server.URL,
		map[string]string{},
		interfaces.ProviderCapabilities{},
	)

	stream, err := provider.StreamChatCompletion(context.Background(), interfaces.ChatRequest{
		Model: "test-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	})

	assert.NoError(t, err)
	assert.NotNil(t, stream)

	// Read one chunk
	chunk, err := stream.Next()
	if err == nil {
		assert.NotEmpty(t, chunk.Delta.Content)
	}

	stream.Close()
}

func TestCreateEmbeddingBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": []float64{0.1, 0.2, 0.3},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		server.URL,
		map[string]string{},
		interfaces.ProviderCapabilities{},
	)

	resp, err := provider.CreateEmbedding(context.Background(), interfaces.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []string{"Hello"},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Embeddings, 1)
	assert.Len(t, resp.Embeddings[0].Embedding, 3)
}

func TestGetCapabilitiesBase(t *testing.T) {
	caps := interfaces.ProviderCapabilities{
		MaxTokens:      4096,
		ContextWindow:  128000,
		SupportsVision: true,
	}

	provider := NewOpenAICompatibleProvider(
		"test",
		"test-key",
		"https://api.test.com",
		map[string]string{},
		caps,
	)

	got := provider.GetCapabilities()
	assert.Equal(t, caps.MaxTokens, got.MaxTokens)
	assert.Equal(t, caps.ContextWindow, got.ContextWindow)
	assert.Equal(t, caps.SupportsVision, got.SupportsVision)
}

func TestParseErrorBasic(t *testing.T) {
	provider := &OpenAICompatibleProvider{}

	// Test 400 error
	err := provider.parseError(400, []byte(`{"error": {"message": "Bad request"}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad request")

	// Test 429 rate limit
	err = provider.parseError(429, []byte("Rate limit"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Rate limit")
}
