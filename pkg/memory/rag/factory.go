// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// Factory creates RAG components
type Factory struct {
	retriever RetrieverInterface
	embedder  vector.Embedder
}

// newFactory creates a new RAG factory (private constructor)
func newFactory(ctx context.Context, embedder vector.Embedder, config Config) (*Factory, error) {
	// Create retriever
	retriever, err := newRetriever(ctx, embedder, config)
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
func (f *Factory) GetRetriever() RetrieverInterface {
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

// DefaultFactoryFactory creates a factory for registry use
func DefaultFactoryFactory(ctx context.Context, embedder vector.Embedder, config Config) (FactoryInterface, error) {
	return newFactory(ctx, embedder, config)
}
