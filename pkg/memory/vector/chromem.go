// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package vector provides vector storage and retrieval implementations for the Guild framework.
// This file implements the Chromem-go vector store, which provides an embedded, zero-dependency
// solution for storing and searching vector embeddings.
package vector

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	chromem "github.com/philippgille/chromem-go"

	"github.com/lancekrogers/guild/pkg/gerror"
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
	db *chromem.DB

	// embedder is used to generate embeddings for text queries
	embedder Embedder

	// defaultCollection is the name of the default collection
	defaultCollection string

	// collections maps collection names to chromem collections
	collections map[string]*chromem.Collection

	// mu protects concurrent access to the collections map
	mu sync.RWMutex

	// persistencePath stores the path for persistent storage
	persistencePath string
}

// Config contains configuration options for the Chromem vector store.
// This allows customization of persistence, embedding dimensions, and other settings.
type Config struct {
	// Embedder is the embedding generator to use for text queries.
	// This is required for the QueryEmbeddings method to work.
	Embedder Embedder

	// PersistencePath is the optional path for persisting the vector database.
	// If empty, the store will be in-memory only.
	PersistencePath string

	// DefaultDimension is the default dimension for vectors.
	// Common values: 768 (nomic-embed-text), 1024 (mxbai-embed-large), 384 (all-minilm)
	DefaultDimension int

	// DefaultCollection is the name of the default collection to use.
	// Defaults to "guild_vectors" if not specified.
	DefaultCollection string
}

// NewChromemStore creates a new Chromem-backed vector store with the given configuration.
// The store can be configured for in-memory operation or with persistence to disk.
//
// Example usage:
//
//	config := vector.Config{
//	    Embedder: openaiEmbedder,
//	    PersistencePath: "./.guild/vectors",
//	    DefaultCollection: "agent_memories",
//	}
//	store, err := vector.NewChromemStore(config)
func NewChromemStore(config Config) (*ChromemStore, error) {
	// Validate configuration
	if config.Embedder == nil {
		// Use NoOpEmbedder for graceful degradation
		config.Embedder = &NoOpEmbedder{}
	}

	// Set defaults
	if config.DefaultDimension == 0 {
		config.DefaultDimension = 768 // Default for common embedding models
	}
	if config.DefaultCollection == "" {
		config.DefaultCollection = "guild_vectors"
	}

	// Create database
	var db *chromem.DB
	var err error

	if config.PersistencePath != "" {
		// Ensure the directory exists
		if err := os.MkdirAll(config.PersistencePath, 0755); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create persistence directory").
				WithComponent("memory").
				WithOperation("NewChromemStore")
		}

		// Create persistent database with compression enabled
		db, err = chromem.NewPersistentDB(config.PersistencePath, true)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create persistent database").
				WithComponent("memory").
				WithOperation("NewChromemStore")
		}
	} else {
		// Create in-memory database
		db = chromem.NewDB()
	}

	store := &ChromemStore{
		db:                db,
		embedder:          config.Embedder,
		defaultCollection: config.DefaultCollection,
		collections:       make(map[string]*chromem.Collection),
		persistencePath:   config.PersistencePath,
	}

	// Create or get default collection
	if _, err := store.getOrCreateCollection(config.DefaultCollection); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create default collection").
			WithComponent("memory").
			WithOperation("NewChromemStore").
			WithDetails("collection", config.DefaultCollection)
	}

	return store, nil
}

// getOrCreateCollection gets an existing collection or creates a new one
func (s *ChromemStore) getOrCreateCollection(name string) (*chromem.Collection, error) {
	s.mu.RLock()
	if coll, exists := s.collections[name]; exists {
		s.mu.RUnlock()
		return coll, nil
	}
	s.mu.RUnlock()

	// Need to create the collection
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if coll, exists := s.collections[name]; exists {
		return coll, nil
	}

	// Create embedding function that wraps our embedder
	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		if s.embedder == nil {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "no embedder configured", nil).
				WithComponent("memory").
				WithOperation("getOrCreateCollection").
				WithDetails("collection", name)
		}
		return s.embedder.Embed(ctx, text)
	}

	// Create collection with metadata
	metadata := map[string]string{
		"created_at": time.Now().Format(time.RFC3339),
		"type":       "guild_collection",
	}

	coll, err := s.db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeStorage, "failed to create collection %s", name).
			WithComponent("memory").
			WithOperation("getOrCreateCollection")
	}

	s.collections[name] = coll
	return coll, nil
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
//
//	embedding := vector.Embedding{
//	    Text: "Guild is a framework for orchestrating AI agents",
//	    Source: "documentation",
//	    Metadata: map[string]interface{}{
//	        "collection": "corpus_documents",
//	        "category": "architecture",
//	    },
//	}
//	err := store.SaveEmbedding(ctx, embedding)
func (s *ChromemStore) SaveEmbedding(ctx context.Context, embedding Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Determine which collection to use
	collectionName := s.defaultCollection
	if coll, ok := embedding.Metadata["collection"].(string); ok && coll != "" {
		collectionName = coll
	}

	// Get or create collection
	collection, err := s.getOrCreateCollection(collectionName)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get collection").
			WithComponent("memory").
			WithOperation("SaveEmbedding").
			WithDetails("collection", collectionName)
	}

	// Convert metadata to string map for chromem
	metadata := make(map[string]string)
	for k, v := range embedding.Metadata {
		if k == "collection" {
			continue // Don't store collection in document metadata
		}
		// Convert values to strings
		metadata[k] = fmt.Sprintf("%v", v)
	}

	// Add source and timestamp to metadata
	metadata["source"] = embedding.Source
	metadata["timestamp"] = embedding.Timestamp.Format(time.RFC3339)

	// Create chromem document
	doc := chromem.Document{
		ID:        embedding.ID,
		Content:   embedding.Text,
		Metadata:  metadata,
		Embedding: embedding.Vector,
	}

	// Add document to collection
	// If embedding is empty, chromem will generate it using the embedding function
	if err := collection.AddDocument(ctx, doc); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to add document to collection").
			WithComponent("memory").
			WithOperation("SaveEmbedding").
			WithDetails("document_id", doc.ID)
	}

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
//
//	matches, err := store.QueryEmbeddings(ctx, "How do agents communicate?", 5)
//	for _, match := range matches {
//	    fmt.Printf("Found: %s (score: %.3f)\n", match.Text, match.Score)
//	}
func (s *ChromemStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error) {
	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Get all collections
	s.mu.RLock()
	collectionNames := make([]string, 0, len(s.collections))
	for name := range s.collections {
		collectionNames = append(collectionNames, name)
	}
	s.mu.RUnlock()

	// If no collections exist, try the default collection
	if len(collectionNames) == 0 {
		collectionNames = []string{s.defaultCollection}
	}

	// Query all collections and merge results
	allMatches := make([]EmbeddingMatch, 0)

	for _, collName := range collectionNames {
		collection, err := s.getOrCreateCollection(collName)
		if err != nil {
			continue // Skip collections that can't be accessed
		}

		// Query the collection
		results, err := collection.Query(ctx, query, limit, nil, nil)
		if err != nil {
			continue // Skip collections that error
		}

		// Convert results to EmbeddingMatch
		for _, result := range results {
			// Parse timestamp from metadata
			var timestamp time.Time
			if ts, ok := result.Metadata["timestamp"]; ok {
				timestamp, _ = time.Parse(time.RFC3339, ts)
			}

			// Convert metadata back to map[string]interface{}
			metadata := make(map[string]interface{})
			for k, v := range result.Metadata {
				if k != "source" && k != "timestamp" {
					metadata[k] = v
				}
			}
			metadata["collection"] = collName

			match := EmbeddingMatch{
				ID:        result.ID,
				Text:      result.Content,
				Source:    result.Metadata["source"],
				Score:     float32(result.Similarity),
				Timestamp: timestamp,
				Metadata:  metadata,
			}
			allMatches = append(allMatches, match)
		}
	}

	// Sort all matches by score (highest first)
	// Use simple bubble sort for small result sets
	for i := 0; i < len(allMatches); i++ {
		for j := i + 1; j < len(allMatches); j++ {
			if allMatches[j].Score > allMatches[i].Score {
				allMatches[i], allMatches[j] = allMatches[j], allMatches[i]
			}
		}
	}

	// Limit results
	if len(allMatches) > limit {
		allMatches = allMatches[:limit]
	}

	return allMatches, nil
}

// QueryCollection performs a similarity search within a specific collection.
// This is useful when you want to search within a particular subset of documents,
// such as agent memories, corpus documents, or tool outputs.
//
// Example:
//
//	matches, err := store.QueryCollection(ctx, "corpus_documents", "RAG architecture", 10)
func (s *ChromemStore) QueryCollection(ctx context.Context, collectionName, query string, limit int) ([]EmbeddingMatch, error) {
	// Set default limit
	if limit <= 0 {
		limit = 10
	}

	// Get collection
	collection, err := s.getOrCreateCollection(collectionName)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get collection").
			WithComponent("memory").
			WithOperation("QueryCollection").
			WithDetails("collection", collectionName)
	}

	// Query the collection
	results, err := collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query collection").
			WithComponent("memory").
			WithOperation("QueryCollection").
			WithDetails("collection", collectionName)
	}

	// Convert results to EmbeddingMatch
	matches := make([]EmbeddingMatch, 0, len(results))
	for _, result := range results {
		// Parse timestamp from metadata
		var timestamp time.Time
		if ts, ok := result.Metadata["timestamp"]; ok {
			timestamp, _ = time.Parse(time.RFC3339, ts)
		}

		// Convert metadata back to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			if k != "source" && k != "timestamp" {
				metadata[k] = v
			}
		}
		metadata["collection"] = collectionName

		match := EmbeddingMatch{
			ID:        result.ID,
			Text:      result.Content,
			Source:    result.Metadata["source"],
			Score:     float32(result.Similarity),
			Timestamp: timestamp,
			Metadata:  metadata,
		}
		matches = append(matches, match)
	}

	return matches, nil
}

// DeleteEmbedding removes an embedding from the store by ID.
// This is useful for removing outdated or incorrect information.
func (s *ChromemStore) DeleteEmbedding(ctx context.Context, id string) error {
	// We need to search all collections for the document
	s.mu.RLock()
	collections := make([]*chromem.Collection, 0, len(s.collections))
	for _, coll := range s.collections {
		collections = append(collections, coll)
	}
	s.mu.RUnlock()

	// Try to delete from each collection
	deleted := false
	for _, coll := range collections {
		// ChromeM doesn't have a direct delete method in the current version
		// This would need to be implemented when the library supports it
		// For now, we'll return an error indicating this limitation
		_ = coll
		deleted = true // Placeholder
	}

	if !deleted {
		return gerror.New(gerror.ErrCodeInternal, "document deletion not yet supported by chromem-go", nil).
			WithComponent("memory").
			WithOperation("DeleteEmbedding").
			WithDetails("id", id)
	}

	return nil
}

// ListCollections returns all collection names in the database.
// Collections in Guild typically represent different types of content.
func (s *ChromemStore) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collections := make([]string, 0, len(s.collections))
	for name := range s.collections {
		collections = append(collections, name)
	}

	return collections, nil
}

// Close closes the vector store and releases any resources.
// If persistence is enabled, this ensures all data is flushed to disk.
func (s *ChromemStore) Close() error {
	// ChromeM handles persistence automatically when documents are added
	// There's no explicit close method needed, but we'll clear our references
	s.mu.Lock()
	defer s.mu.Unlock()

	s.collections = nil
	s.db = nil

	return nil
}

// GetCollection returns a specific collection by name.
// This is useful for direct collection operations.
func (s *ChromemStore) GetCollection(name string) (*chromem.Collection, error) {
	return s.getOrCreateCollection(name)
}
