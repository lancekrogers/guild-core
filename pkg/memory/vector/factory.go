package vector

import (
	"context"
	"fmt"

	"github.com/blockhead-consulting/guild/pkg/providers"
)

// StoreType represents the type of vector store
type StoreType string

const (
	// StoreTypeQdrant is a Qdrant vector store
	StoreTypeQdrant StoreType = "qdrant"

	// StoreTypeChroma is a Chroma vector store
	StoreTypeChroma StoreType = "chroma"

	// StoreTypeMilvus is a Milvus vector store (not implemented yet)
	StoreTypeMilvus StoreType = "milvus"
)

// StoreConfig represents the configuration for a vector store
type StoreConfig struct {
	// Type is the type of vector store
	Type StoreType

	// URL is the address of the vector store
	URL string

	// Collection is the collection/namespace to use
	Collection string

	// EmbeddingProvider is the provider to use for generating embeddings
	EmbeddingProvider providers.LLMClient

	// EmbeddingModel is the model to use for embeddings
	EmbeddingModel string

	// AdditionalConfig contains additional configuration specific to the vector store type
	AdditionalConfig map[string]interface{}
}

// NewVectorStore creates a new vector store based on the provided configuration
func NewVectorStore(ctx context.Context, config *StoreConfig) (VectorStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("embedding provider cannot be nil")
	}

	embeddingProvider := &providerAdapter{
		provider:     config.EmbeddingProvider,
		defaultModel: config.EmbeddingModel,
	}

	switch config.Type {
	case StoreTypeQdrant:
		return createQdrantStore(ctx, config, embeddingProvider)
	case StoreTypeChroma:
		return createChromaStore(ctx, config, embeddingProvider)
	case StoreTypeMilvus:
		return nil, fmt.Errorf("milvus vector store not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", config.Type)
	}
}

// createQdrantStore creates a Qdrant vector store
func createQdrantStore(ctx context.Context, config *StoreConfig, embeddingProvider EmbeddingProvider) (*QdrantStore, error) {
	qdrantConfig := &QdrantConfig{
		Address:           config.URL,
		Collection:        config.Collection,
		EmbeddingProvider: embeddingProvider,
	}

	// Set dimension from embedding provider if not specified
	if dim := config.EmbeddingProvider.GetEmbeddingDimension(config.EmbeddingModel); dim > 0 {
		qdrantConfig.Dimension = uint64(dim)
	}

	// Apply additional configuration
	if config.AdditionalConfig != nil {
		if val, ok := config.AdditionalConfig["dimension"].(int); ok && val > 0 {
			qdrantConfig.Dimension = uint64(val)
		}
	}

	return NewQdrantStore(ctx, qdrantConfig)
}

// createChromaStore creates a Chroma vector store
func createChromaStore(ctx context.Context, config *StoreConfig, embeddingProvider EmbeddingProvider) (*ChromaStore, error) {
	chromaConfig := &ChromaConfig{
		URL:                config.URL,
		CollectionName:     config.Collection,
		EmbeddingProvider:  embeddingProvider,
	}

	return NewChromaStore(ctx, chromaConfig)
}

// providerAdapter adapts an LLMClient to the EmbeddingProvider interface
type providerAdapter struct {
	provider     providers.LLMClient
	defaultModel string
}

// GetEmbedding generates an embedding for the given text
func (p *providerAdapter) GetEmbedding(ctx context.Context, text string) (Vector, error) {
	req := &providers.EmbeddingRequest{
		Text:  text,
		Model: p.defaultModel,
	}

	resp, err := p.provider.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	// Convert to Vector
	embedding := make(Vector, len(resp.Embedding))
	for i, v := range resp.Embedding {
		embedding[i] = v
	}

	return embedding, nil
}

// GetEmbeddings generates embeddings for multiple texts
func (p *providerAdapter) GetEmbeddings(ctx context.Context, texts []string) ([]Vector, error) {
	if len(texts) == 0 {
		return []Vector{}, nil
	}

	req := &providers.EmbeddingRequest{
		Texts: texts,
		Model: p.defaultModel,
	}

	resp, err := p.provider.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Convert to Vectors
	embeddings := make([]Vector, len(resp.Embeddings))
	for i, e := range resp.Embeddings {
		embedding := make(Vector, len(e))
		for j, v := range e {
			embedding[j] = v
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}