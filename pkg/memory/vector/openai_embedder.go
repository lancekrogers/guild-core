package vector

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(apiKey string, model string) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if model == "" {
		model = openai.AdaEmbeddingV2
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbedder{
		client: client,
		model:  model,
	}, nil
}

// Embed generates an embedding from text
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	response, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return response.Data[0].Embedding, nil
}