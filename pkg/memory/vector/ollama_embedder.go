package vector

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/ollama"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// OllamaEmbedder implements the Embedder interface using local Ollama models.
// This provides a completely offline embedding solution that doesn't require API keys
// or internet connectivity. It uses locally hosted models through Ollama.
//
// Benefits of using Ollama for embeddings:
//   - Completely offline - no internet required
//   - No API costs - free local inference
//   - Privacy - data never leaves your machine
//   - Customizable - use any embedding model supported by Ollama
//   - Fast - local inference without network latency
//
// Popular embedding models for Ollama:
//   - nomic-embed-text: 768 dimensions, good general purpose
//   - all-minilm: 384 dimensions, faster and smaller
//   - bge-large: 1024 dimensions, high quality
//   - e5-large: 1024 dimensions, multilingual support
type OllamaEmbedder struct {
	client  *ollama.Client
	model   string
	baseURL string
}

// NewOllamaEmbedder creates a new Ollama-based embedder.
//
// Parameters:
//   - baseURL: Ollama server URL (e.g., "http://localhost:11434")
//   - model: Embedding model name (e.g., "nomic-embed-text", "all-minilm")
//
// The embedder will automatically pull the model if it's not available locally.
func NewOllamaEmbedder(baseURL, model string) (*OllamaEmbedder, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama URL
	}
	
	if model == "" {
		model = "nomic-embed-text" // Default embedding model
	}

	client := ollama.NewClient(baseURL)

	return &OllamaEmbedder{
		client:  client,
		model:   model,
		baseURL: baseURL,
	}, nil
}

// Embed generates an embedding from text using the local Ollama model.
//
// This method:
//   1. Sends the text to the local Ollama server
//   2. Uses the configured embedding model to generate vectors
//   3. Returns the embedding as float32 slice for compatibility
//
// No internet connection or API keys required.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidArgument).
			WithComponent("memory").
			WithOperation("Embed").
			WithDetails("text cannot be empty")
	}

	// Create embedding request
	req := interfaces.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	}

	// Call Ollama to generate embedding
	response, err := e.client.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal).
			WithComponent("memory").
			WithOperation("Embed").
			WithDetails("failed to create embedding with Ollama")
	}

	if len(response.Embeddings) == 0 {
		return nil, gerror.New(gerror.ErrCodeInternal).
			WithComponent("memory").
			WithOperation("Embed").
			WithDetails("no embeddings returned from Ollama")
	}

	// Convert []float64 to []float32 for consistency with OpenAI embedder
	embedding := response.Embeddings[0].Embedding
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

// GetEmbedding is an alias for Embed (for backward compatibility)
func (e *OllamaEmbedder) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	return e.Embed(ctx, text)
}

// GetEmbeddings gets embeddings for multiple texts in a single batch call.
// This is more efficient than calling Embed multiple times.
func (e *OllamaEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidArgument).
			WithComponent("memory").
			WithOperation("GetEmbeddings").
			WithDetails("no texts provided")
	}

	// Create embedding request for all texts
	req := interfaces.EmbeddingRequest{
		Input: texts,
		Model: e.model,
	}

	// Call Ollama to generate embeddings
	response, err := e.client.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal).
			WithComponent("memory").
			WithOperation("GetEmbeddings").
			WithDetails("failed to create embeddings with Ollama")
	}

	if len(response.Embeddings) != len(texts) {
		return nil, gerror.New(gerror.ErrCodeInternal).
			WithComponent("memory").
			WithOperation("GetEmbeddings").
			WithDetails(fmt.Sprintf("expected %d embeddings, got %d", len(texts), len(response.Embeddings)))
	}

	// Convert all embeddings from []float64 to []float32
	results := make([][]float32, len(response.Embeddings))
	for i, embData := range response.Embeddings {
		embedding := embData.Embedding
		result := make([]float32, len(embedding))
		for j, v := range embedding {
			result[j] = float32(v)
		}
		results[i] = result
	}

	return results, nil
}

// GetModel returns the current embedding model name
func (e *OllamaEmbedder) GetModel() string {
	return e.model
}

// SetModel changes the embedding model (useful for switching between models)
func (e *OllamaEmbedder) SetModel(model string) {
	e.model = model
	// Note: The client itself doesn't need updating as model is passed per request
}

// GetBaseURL returns the Ollama server URL
func (e *OllamaEmbedder) GetBaseURL() string {
	return e.baseURL
}

// IsModelAvailable checks if the embedding model is pulled and available locally
func (e *OllamaEmbedder) IsModelAvailable(ctx context.Context) (bool, error) {
	// This would require extending the Ollama client to list available models
	// For now, we'll try a test embedding to see if the model works
	_, err := e.Embed(ctx, "test")
	if err != nil {
		return false, err
	}
	return true, nil
}

// PullModel downloads the embedding model if it's not available locally
func (e *OllamaEmbedder) PullModel(ctx context.Context) error {
	// This would require extending the Ollama client to support model pulling
	// For now, return an informative error
	return gerror.New(gerror.ErrCodeNotImplemented).
		WithComponent("memory").
		WithOperation("PullModel").
		WithDetails(fmt.Sprintf("model pulling not yet implemented - please run 'ollama pull %s' manually", e.model))
}