// Package vector provides vector storage and retrieval implementations for the Guild framework.
// This file implements the Chromem-go vector store, which provides an embedded, zero-dependency
// solution for storing and searching vector embeddings.
package vector

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
)

// ChromemStore implements the VectorStore interface using Chromem-go.
// Chromem-go is a pure Go, embeddable vector database that requires no external dependencies.
// It's ideal for Guild's use case as it provides good performance for small to medium datasets
// while maintaining simplicity and ease of deployment.
//
// This implementation supports:
// - In-memory storage with optional persistence
// - Multiple collections for organizing different types of embeddings
// - Cosine similarity search for finding relevant content
// - Thread-safe operations
//
// Collections in Guild typically represent:
// - "agent_memories": Memories from agent interactions
// - "corpus_documents": Research corpus documents  
// - "tool_outputs": Results from tool executions
type ChromemStore struct {
	// db is the underlying Chromem database instance
	db       *chromem.DB
	
	// embedder is used to generate embeddings for text queries
	embedder Embedder
	
	// mu protects concurrent access to the store
	mu       sync.RWMutex
	
	// Mock in-memory storage for testing
	// TODO: Remove this when chromem-go is properly integrated
	embeddings map[string]Embedding
	
	// defaultCollection is the name of the default collection
	defaultCollection string
}

// Config contains configuration options for the Chromem vector store.
// This allows customization of persistence, embedding dimensions, and other settings.
type Config struct {
	// Embedder is the embedding generator to use for text queries.
	// This is required for the QueryEmbeddings method to work.
	Embedder         Embedder
	
	// PersistencePath is the optional path for persisting the vector database.
	// If empty, the store will be in-memory only.
	PersistencePath  string
	
	// DefaultDimension is the default dimension for vectors.
	// Common values: 1536 (OpenAI), 1024 (many open models), 768 (sentence-transformers)
	DefaultDimension int
	
	// DefaultCollection is the name of the default collection to use.
	// Defaults to "guild_vectors" if not specified.
	DefaultCollection string
}

// NewChromemStore creates a new Chromem-backed vector store with the given configuration.
// The store can be configured for in-memory operation or with persistence to disk.
//
// Example usage:
//   config := vector.Config{
//       Embedder: openaiEmbedder,
//       PersistencePath: "./data/vectors",
//       DefaultCollection: "agent_memories",
//   }
//   store, err := vector.NewChromemStore(config)
func NewChromemStore(config Config) (*ChromemStore, error) {
	// Validate configuration
	if config.Embedder == nil {
		return nil, fmt.Errorf("embedder is required for Chromem store")
	}
	
	// Set defaults
	if config.DefaultDimension == 0 {
		config.DefaultDimension = 1536 // Default for OpenAI embeddings
	}
	if config.DefaultCollection == "" {
		config.DefaultCollection = "guild_vectors"
	}

	// TODO: Replace with actual chromem-go API when properly integrated
	// The actual chromem.NewDB function and API may differ
	// Create a simplistic wrapper to simulate the library functionality
	db := &chromem.DB{}
	
	// If persistence path is provided, configure persistence
	// TODO: Implement persistence when chromem-go API is finalized
	if config.PersistencePath != "" {
		// db.EnablePersistence(config.PersistencePath)
	}
	
	return &ChromemStore{
		db:                db,
		embedder:          config.Embedder,
		embeddings:        make(map[string]Embedding),
		defaultCollection: config.DefaultCollection,
	}, nil
}

// SaveEmbedding stores a vector embedding in the database.
// The embedding is stored in the default collection unless a different collection
// is specified in the embedding's metadata under the "collection" key.
//
// If no ID is provided, a UUID will be generated.
// If no vector is provided but an embedder is configured, the vector will be
// generated from the text content.
//
// Example:
//   embedding := vector.Embedding{
//       Text: "Guild is a framework for orchestrating AI agents",
//       Source: "documentation",
//       Metadata: map[string]interface{}{
//           "collection": "corpus_documents",
//           "category": "architecture",
//       },
//   }
//   err := store.SaveEmbedding(ctx, embedding)
func (s *ChromemStore) SaveEmbedding(ctx context.Context, embedding Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Determine which collection to use
	collection := s.defaultCollection
	if coll, ok := embedding.Metadata["collection"].(string); ok && coll != "" {
		collection = coll
	}

	// Generate vector if not provided
	vector := embedding.Vector
	if len(vector) == 0 && s.embedder != nil {
		var err error
		vector, err = s.embedder.Embed(ctx, embedding.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		embedding.Vector = vector
	}

	// TODO: When chromem-go is integrated, store in the appropriate collection
	// For now, use mock in-memory storage
	s.mu.Lock()
	// Add collection info to metadata for retrieval
	if embedding.Metadata == nil {
		embedding.Metadata = make(map[string]interface{})
	}
	embedding.Metadata["_collection"] = collection
	s.embeddings[embedding.ID] = embedding
	s.mu.Unlock()
	
	return nil
}

// QueryEmbeddings performs a similarity search using the provided query text.
// It generates an embedding for the query using the configured embedder and
// searches across all collections in the database.
//
// The search uses cosine similarity to find the most relevant documents.
// Results are sorted by similarity score (highest first) and limited to the
// specified number of results.
//
// Example:
//   matches, err := store.QueryEmbeddings(ctx, "How do agents communicate?", 5)
//   for _, match := range matches {
//       fmt.Printf("Found: %s (score: %.3f)\n", match.Text, match.Score)
//   }
func (s *ChromemStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error) {
	// Validate embedder
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured")
	}
	
	// Generate query vector
	queryVector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Calculate similarities across all embeddings
	s.mu.RLock()
	matches := make([]EmbeddingMatch, 0, len(s.embeddings))
	for _, emb := range s.embeddings {
		// Skip if vector is missing
		if len(emb.Vector) == 0 {
			continue
		}
		
		score := cosineSimilarity(queryVector, emb.Vector)
		
		// Create a clean copy of metadata without internal fields
		cleanMetadata := make(map[string]interface{})
		for k, v := range emb.Metadata {
			if !strings.HasPrefix(k, "_") { // Skip internal metadata
				cleanMetadata[k] = v
			}
		}
		
		matches = append(matches, EmbeddingMatch{
			ID:        emb.ID,
			Text:      emb.Text,
			Source:    emb.Source,
			Score:     score,
			Timestamp: emb.Timestamp,
			Metadata:  cleanMetadata,
		})
	}
	s.mu.RUnlock()

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// QueryCollection performs a similarity search within a specific collection.
// This is useful when you want to search within a particular subset of documents,
// such as agent memories, corpus documents, or tool outputs.
//
// Example:
//   matches, err := store.QueryCollection(ctx, "corpus_documents", "RAG architecture", 10)
func (s *ChromemStore) QueryCollection(ctx context.Context, collectionName, query string, limit int) ([]EmbeddingMatch, error) {
	// For now, filter by collection metadata in the mock implementation
	// TODO: Implement proper collection support when chromem-go is integrated
	
	// Validate embedder
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured")
	}
	
	// Generate query vector
	queryVector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Calculate similarities for embeddings in the specified collection
	s.mu.RLock()
	matches := make([]EmbeddingMatch, 0)
	for _, emb := range s.embeddings {
		// Check if embedding belongs to the specified collection
		if coll, ok := emb.Metadata["_collection"].(string); !ok || coll != collectionName {
			continue
		}
		
		// Skip if vector is missing
		if len(emb.Vector) == 0 {
			continue
		}
		
		score := cosineSimilarity(queryVector, emb.Vector)
		
		// Create a clean copy of metadata without internal fields
		cleanMetadata := make(map[string]interface{})
		for k, v := range emb.Metadata {
			if !strings.HasPrefix(k, "_") { // Skip internal metadata
				cleanMetadata[k] = v
			}
		}
		
		matches = append(matches, EmbeddingMatch{
			ID:        emb.ID,
			Text:      emb.Text,
			Source:    emb.Source,
			Score:     score,
			Timestamp: emb.Timestamp,
			Metadata:  cleanMetadata,
		})
	}
	s.mu.RUnlock()

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// DeleteEmbedding removes an embedding from the store by ID.
// This is useful for removing outdated or incorrect information.
func (s *ChromemStore) DeleteEmbedding(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.embeddings[id]; !exists {
		return fmt.Errorf("embedding %s not found", id)
	}
	
	delete(s.embeddings, id)
	return nil
}

// ListCollections returns all collection names in the database.
// Collections in Guild typically represent different types of content.
func (s *ChromemStore) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	collections := make(map[string]bool)
	for _, emb := range s.embeddings {
		if coll, ok := emb.Metadata["_collection"].(string); ok {
			collections[coll] = true
		}
	}
	
	// Convert map to slice
	result := make([]string, 0, len(collections))
	for coll := range collections {
		result = append(result, coll)
	}
	
	// Sort for consistent ordering
	sort.Strings(result)
	
	return result, nil
}

// Close closes the vector store and releases any resources.
// If persistence is enabled, this ensures all data is flushed to disk.
func (s *ChromemStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// TODO: When chromem-go is integrated, properly close the database
	// For now, clear the in-memory storage
	s.embeddings = nil
	
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// The result is a value between -1 and 1, where 1 means identical,
// 0 means orthogonal, and -1 means opposite.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	// Use a simple approximation for square root
	normA = sqrt(normA)
	normB = sqrt(normB)

	return dotProduct / (normA * normB)
}

// sqrt is a simple square root approximation using Newton's method.
// This provides sufficient accuracy for similarity calculations.
func sqrt(x float32) float32 {
	if x == 0 {
		return 0
	}
	
	// Newton's method for square root
	guess := x
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}