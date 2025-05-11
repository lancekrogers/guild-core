package ollama

import (
	"context"
	"fmt"

	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
)

// Client implements the LLMClient interface for Ollama
type Client struct {
	apiKey string
	model  string
}

// NewClient creates a new Ollama client
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("Stub response for prompt: %s", prompt), nil
}

// CreateCompletion is a lower-level method to create a completion
func (c *Client) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	return &interfaces.CompletionResponse{
		Text: fmt.Sprintf("Stub response for prompt: %s", req.Prompt),
		TokensUsed: 10,
		TokensInput: 5,
		TokensOutput: 5,
		ModelUsed: c.model,
	}, nil
}

// CreateEmbedding generates an embedding for the given text
func (c *Client) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return &interfaces.EmbeddingResponse{
		Embedding: []float32{0.1, 0.2, 0.3},
		Dimensions: 3,
	}, nil
}

// CreateEmbeddings generates embeddings for multiple texts
func (c *Client) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return &interfaces.EmbeddingResponse{
		Embeddings: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
		Dimensions: 3,
	}, nil
}
