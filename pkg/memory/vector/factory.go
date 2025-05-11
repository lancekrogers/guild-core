package vector

import (
	"context"
	"fmt"
	"os"

	"github.com/lancerogers/guild/pkg/providers"
)

// StoreType represents the type of vector store
type StoreType string

const (
	// StoreTypeChromem is an embedded Chromem vector store
	StoreTypeChromem StoreType = "chromem"

	// StoreTypeChroma is a Chroma vector store
	StoreTypeChroma StoreType = "chroma"

	// StoreTypeMilvus is a Milvus vector store (not implemented yet)
	StoreTypeMilvus StoreType = "milvus"
)

// StoreConfig represents the configuration for a vector store
type StoreConfig struct {
	// Type is the type of vector store
	Type StoreType

	// EmbeddingProvider is the provider to use for generating embeddings
	EmbeddingProvider providers.LLMClient

	// EmbeddingModel is the model to use for embeddings
	EmbeddingModel string

	// ChromemConfig contains Chromem-specific configuration
	ChromemConfig ChromemConfig

	// OpenAIApiKey is the API key for OpenAI (for embeddings)
	OpenAIApiKey string
}

// ChromemConfig contains Chromem-specific configuration
type ChromemConfig struct {
	// PersistencePath is the path to persist embeddings to disk
	PersistencePath string

	// DefaultDimension is the default dimension for vectors
	DefaultDimension int
}

// NewVectorStore creates a new vector store based on the provided configuration
func NewVectorStore(ctx context.Context, config *StoreConfig) (VectorStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create embedder
	embedder, err := createEmbedder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	switch config.Type {
	case StoreTypeChromem:
		return createChromemStore(ctx, config, embedder)
	case StoreTypeChroma:
		return nil, fmt.Errorf("chroma vector store not implemented yet")
	case StoreTypeMilvus:
		return nil, fmt.Errorf("milvus vector store not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", config.Type)
	}
}

// createChromemStore creates a Chromem vector store
func createChromemStore(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error) {
	chromemConfig := Config{
		Embedder:        embedder,
		PersistencePath: config.ChromemConfig.PersistencePath,
		DefaultDimension: config.ChromemConfig.DefaultDimension,
	}

	if chromemConfig.DefaultDimension == 0 {
		chromemConfig.DefaultDimension = 1536 // Default for OpenAI embeddings
	}

	return NewChromemStore(chromemConfig)
}

// createEmbedder creates an embedder based on the configuration
func createEmbedder(config *StoreConfig) (Embedder, error) {
	// Try to get OpenAI API key
	apiKey := config.OpenAIApiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required for embeddings")
	}

	// Default to OpenAI's text-embedding-ada-002 model
	model := "text-embedding-ada-002"
	if config.EmbeddingModel != "" {
		model = config.EmbeddingModel
	}

	return NewOpenAIEmbedder(apiKey, model)
}