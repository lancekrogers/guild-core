// Package vector provides factory functionality for creating vector stores.
// The factory pattern allows for easy registration and creation of different
// vector store implementations while maintaining a consistent interface.
//
// This factory supports the Guild framework's registry pattern, allowing
// new vector store implementations to be registered at runtime.
package vector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// StoreType represents the type of vector store
type StoreType string

const (
	// StoreTypeChromem is an embedded Chromem vector store.
	// Chromem-go provides a pure Go, zero-dependency solution ideal for
	// Guild's embedded use case with good performance for small to medium datasets.
	StoreTypeChromem StoreType = "chromem"

	// StoreTypeChroma is a Chroma vector store.
	// Chroma is a feature-rich vector database with Python bindings,
	// suitable when advanced features are needed.
	StoreTypeChroma StoreType = "chroma"

	// StoreTypeMilvus is a Milvus vector store.
	// Milvus is a distributed vector database designed for billion-scale
	// vector similarity search and is suitable for large-scale deployments.
	StoreTypeMilvus StoreType = "milvus"
)

// StoreConfig represents the configuration for a vector store.
// This configuration is used by the factory to create the appropriate
// vector store implementation with the correct settings.
type StoreConfig struct {
	// Type is the type of vector store to create.
	// Use one of the StoreType constants (e.g., StoreTypeChromem).
	Type StoreType

	// EmbeddingProvider is the AI provider to use for generating embeddings.
	// This is optional - if not provided, the factory will auto-detect available providers.
	// Supports any provider that implements the AIProvider interface.
	EmbeddingProvider interfaces.AIProvider

	// EmbeddingModel is the model to use for embeddings.
	// Common values: 
	// - "nomic-embed-text" (768 dims)
	// - "mxbai-embed-large" (1024 dims)
	// - "all-minilm" (384 dims)
	// If not specified, auto-detects based on provider.
	EmbeddingModel string

	// EmbeddingStrategy specifies how to extract embeddings.
	// Options: "auto", "dedicated", "hidden_state", "mean_pooling", "none"
	// Defaults to "auto" which tries dedicated models first.
	EmbeddingStrategy EmbeddingStrategy

	// ChromemConfig contains Chromem-specific configuration.
	// Only used when Type is StoreTypeChromem.
	ChromemConfig ChromemConfig
	
	// DefaultCollection is the default collection name to use.
	// Collections help organize embeddings by type (e.g., "agent_memories", "corpus_documents").
	DefaultCollection string
}

// ChromemConfig contains Chromem-specific configuration options.
// Chromem is an embedded vector database that can optionally persist to disk.
type ChromemConfig struct {
	// PersistencePath is the path to persist embeddings to disk.
	// If empty, the store will be in-memory only.
	// Example: "./.guild/vectors"
	PersistencePath string

	// DefaultDimension is the default dimension for vectors.
	// This should match the dimension of your embedding model.
	// Common values: 768 (nomic-embed-text), 1024 (mxbai-embed-large), 384 (all-minilm)
	DefaultDimension int
	
	// DefaultCollection overrides the collection name from StoreConfig.
	// If both are set, this one takes precedence.
	DefaultCollection string
}

// NewVectorStore creates a new vector store based on the provided configuration.
// It uses the registry pattern to look up the appropriate factory for the
// requested store type. This allows for easy extension with new store types.
//
// Example:
//   config := &vector.StoreConfig{
//       Type: vector.StoreTypeChromem,
//       ChromemConfig: vector.ChromemConfig{
//           PersistencePath: "./.guild/vectors",
//       },
//   }
//   store, err := vector.NewVectorStore(ctx, config)
func NewVectorStore(ctx context.Context, config *StoreConfig) (VectorStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create embedder
	embedder, err := createEmbedder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Look up factory in registry
	globalRegistry.mu.RLock()
	factory, exists := globalRegistry.factories[config.Type]
	globalRegistry.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("unsupported vector store type: %s (registered types: %v)", 
			config.Type, ListRegisteredStores())
	}
	
	// Use factory to create store
	return factory(ctx, config, embedder)
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
	// If provider is already specified, use it
	if config.EmbeddingProvider != nil {
		opts := []EmbedderOption{
			WithStrategy(config.EmbeddingStrategy),
		}
		if config.EmbeddingModel != "" {
			opts = append(opts, WithModel(config.EmbeddingModel))
		}
		return NewUniversalEmbedder(config.EmbeddingProvider, opts...), nil
	}

	// Auto-detect available providers
	provider, err := detectAvailableProvider()
	if err != nil {
		// No provider available - use NoOpEmbedder for graceful degradation
		return &NoOpEmbedder{}, nil
	}

	opts := []EmbedderOption{
		WithStrategy(config.EmbeddingStrategy),
	}
	if config.EmbeddingModel != "" {
		opts = append(opts, WithModel(config.EmbeddingModel))
	}

	return NewUniversalEmbedder(provider, opts...), nil
}

// detectAvailableProvider attempts to auto-detect available AI providers
func detectAvailableProvider() (interfaces.AIProvider, error) {
	factory := providers.NewFactoryV2()
	
	// Check for Ollama first (preferred for offline operation)
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}
	
	// Try to connect to Ollama
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ollamaHost + "/api/tags")
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		// Ollama is available
		provider, err := factory.CreateAIProvider(providers.ProviderOllama, ollamaHost)
		if err == nil {
			return provider, nil
		}
	}
	
	// Check for other providers via environment variables
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, apiKey)
		if err == nil {
			return provider, nil
		}
	}
	
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider, err := factory.CreateAIProvider(providers.ProviderAnthropic, apiKey)
		if err == nil {
			return provider, nil
		}
	}
	
	return nil, fmt.Errorf("no AI provider available: check Ollama installation or set API keys")
}

// VectorStoreFactory is a function that creates a VectorStore
type VectorStoreFactory func(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error)

// Registry manages vector store factories following the Guild registry pattern.
// This allows new vector store implementations to be registered at runtime
// and selected based on configuration.
type Registry struct {
	mu        sync.RWMutex
	factories map[StoreType]VectorStoreFactory
}

// globalRegistry is the default registry instance
var globalRegistry = &Registry{
	factories: make(map[StoreType]VectorStoreFactory),
}

// init registers the built-in vector store implementations
func init() {
	// Register Chromem store
	RegisterVectorStore(StoreTypeChromem, func(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error) {
		chromemConfig := Config{
			Embedder:         embedder,
			PersistencePath:  config.ChromemConfig.PersistencePath,
			DefaultDimension: config.ChromemConfig.DefaultDimension,
		}

		// Set default dimension if not specified
		if chromemConfig.DefaultDimension == 0 {
			chromemConfig.DefaultDimension = 768 // Default for common embedding models
		}
		
		// Set default collection, preferring ChromemConfig over StoreConfig
		if config.ChromemConfig.DefaultCollection != "" {
			chromemConfig.DefaultCollection = config.ChromemConfig.DefaultCollection
		} else if config.DefaultCollection != "" {
			chromemConfig.DefaultCollection = config.DefaultCollection
		} else {
			chromemConfig.DefaultCollection = "guild_vectors"
		}

		return NewChromemStore(chromemConfig)
	})

	// Register placeholder for Chroma
	RegisterVectorStore(StoreTypeChroma, func(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error) {
		return nil, fmt.Errorf("chroma vector store not implemented yet")
	})

	// Register placeholder for Milvus
	RegisterVectorStore(StoreTypeMilvus, func(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error) {
		return nil, fmt.Errorf("milvus vector store not implemented yet")
	})
}

// RegisterVectorStore registers a new vector store factory.
// This follows the Guild registry pattern and allows for runtime extension
// of available vector store types.
//
// Example:
//   vector.RegisterVectorStore("custom", func(ctx context.Context, config *StoreConfig, embedder Embedder) (VectorStore, error) {
//       return NewCustomStore(config, embedder)
//   })
func RegisterVectorStore(storeType StoreType, factory VectorStoreFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	
	globalRegistry.factories[storeType] = factory
}

// GetVectorStore creates a vector store using the registered factory.
// This is an alternative to NewVectorStore that emphasizes the registry pattern.
func GetVectorStore(ctx context.Context, config *StoreConfig) (VectorStore, error) {
	return NewVectorStore(ctx, config)
}

// ListRegisteredStores returns a list of all registered vector store types.
// This is useful for discovery and validation of configuration.
func ListRegisteredStores() []StoreType {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	types := make([]StoreType, 0, len(globalRegistry.factories))
	for t := range globalRegistry.factories {
		types = append(types, t)
	}
	
	return types
}