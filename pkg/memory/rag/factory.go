package rag

import (
	"context"
	
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
)

// Factory creates RAG components
type Factory struct {
	retriever *Retriever
	embedder  vector.Embedder
}

// NewFactory creates a new RAG factory
func NewFactory(ctx context.Context, embedder vector.Embedder, config Config) (*Factory, error) {
	// Create retriever
	retriever, err := NewRetriever(ctx, embedder, config)
	if err != nil {
		return nil, err
	}
	
	// Create factory
	factory := &Factory{
		retriever: retriever,
		embedder:  embedder,
	}
	
	return factory, nil
}

// GetRetriever returns the retriever
func (f *Factory) GetRetriever() *Retriever {
	return f.retriever
}

// GetEmbedder returns the embedder
func (f *Factory) GetEmbedder() vector.Embedder {
	return f.embedder
}

// Close closes the factory and all its resources
func (f *Factory) Close() error {
	if f.retriever != nil {
		return f.retriever.Close()
	}
	return nil
}