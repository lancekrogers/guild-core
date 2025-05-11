package vector

import (
	"context"
	"fmt"
)

// EmbedderWrapper wraps an Embedder to provide the EmbeddingProvider interface
type EmbedderWrapper struct {
	embedder Embedder
}

// NewEmbedderWrapper creates a new EmbedderWrapper
func NewEmbedderWrapper(embedder Embedder) *EmbedderWrapper {
	return &EmbedderWrapper{
		embedder: embedder,
	}
}

// Embed calls the wrapped embedder's Embed method
func (w *EmbedderWrapper) Embed(ctx context.Context, text string) ([]float32, error) {
	return w.embedder.Embed(ctx, text)
}

// GetEmbedding is an alias for Embed for backward compatibility
func (w *EmbedderWrapper) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	return w.embedder.Embed(ctx, text)
}

// GetEmbeddings gets embeddings for multiple texts
func (w *EmbedderWrapper) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	// Call Embed for each text
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := w.embedder.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}