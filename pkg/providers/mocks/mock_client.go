package mocks

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// MockClient provides a mock implementation of the LLMClient interface for testing
type MockClient struct {
	CompletionResponses []string
	EmbeddingResponses  [][]float32
	CallCount           int
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		CompletionResponses: []string{"Mock response"},
		EmbeddingResponses:  [][]float32{{0.1, 0.2, 0.3}},
	}
}

// Complete generates a completion for the given prompt
func (c *MockClient) Complete(ctx context.Context, prompt string) (string, error) {
	c.CallCount++
	if len(c.CompletionResponses) > 0 {
		return c.CompletionResponses[0], nil
	}
	return fmt.Sprintf("Mock response for: %s", prompt), nil
}

// CreateCompletion is a lower-level method to create a completion
func (c *MockClient) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	c.CallCount++
	return &interfaces.CompletionResponse{
		Text: fmt.Sprintf("Mock response for: %s", req.Prompt),
		TokensUsed: 10,
		TokensInput: 5,
		TokensOutput: 5,
		ModelUsed: "mock-model",
	}, nil
}

// CreateEmbedding generates an embedding for the given text
func (c *MockClient) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	c.CallCount++
	return &interfaces.EmbeddingResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
		Dimensions: 3,
	}, nil
}

// CreateEmbeddings generates embeddings for multiple texts
func (c *MockClient) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	c.CallCount++
	return &interfaces.EmbeddingResponse{
		Embeddings: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
		Dimensions: 3,
	}, nil
}