package vector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
)

// ChromemStore implements the VectorStore interface using Chromem-go
type ChromemStore struct {
	db       *chromem.DB
	embedder Embedder
}

// Config contains Chromem configuration
type Config struct {
	Embedder         Embedder
	PersistencePath  string // Optional path for persistence
	DefaultDimension int    // Default dimension for vectors
}

// NewChromemStore creates a new Chromem store
func NewChromemStore(config Config) (*ChromemStore, error) {
	// Set defaults
	if config.DefaultDimension == 0 {
		config.DefaultDimension = 1536 // Default for OpenAI embeddings
	}

	// Configure options
	opts := []chromem.Option{
		chromem.WithDefaultDimension(config.DefaultDimension),
	}

	// Add persistence if path provided
	if config.PersistencePath != "" {
		opts = append(opts, chromem.WithPersistence(config.PersistencePath))
	}

	// Create DB
	db, err := chromem.NewDB(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Chromem DB: %w", err)
	}

	return &ChromemStore{
		db:       db,
		embedder: config.Embedder,
	}, nil
}

// SaveEmbedding stores a vector embedding
func (s *ChromemStore) SaveEmbedding(ctx context.Context, embedding Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Generate vector if not provided
	vector := embedding.Vector
	if len(vector) == 0 && s.embedder != nil {
		var err error
		vector, err = s.embedder.Embed(ctx, embedding.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Create collection if it doesn't exist
	collection := "default"
	if src, ok := embedding.Metadata["collection"]; ok {
		if colName, ok := src.(string); ok && colName != "" {
			collection = colName
		}
	}

	// Create metadata
	metadata := map[string]any{
		"text":      embedding.Text,
		"source":    embedding.Source,
		"timestamp": embedding.Timestamp.Format(time.RFC3339),
	}

	// Add custom metadata
	for k, v := range embedding.Metadata {
		if k != "collection" { // Skip collection as we already used it
			metadata[k] = v
		}
	}

	// Add the embedding
	err := s.db.UpsertEmbedding(ctx, collection, embedding.ID, vector, metadata)
	if err != nil {
		return fmt.Errorf("failed to upsert embedding: %w", err)
	}

	return nil
}

// QueryEmbeddings performs a similarity search
func (s *ChromemStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error) {
	// Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Search (in all collections)
	results, err := s.db.QueryAllCollections(ctx, vector, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}

	// Convert results
	matches := make([]EmbeddingMatch, 0, len(results))
	for _, result := range results {
		match := EmbeddingMatch{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract fields from metadata
		if text, ok := result.Metadata["text"].(string); ok {
			match.Text = text
		}
		if source, ok := result.Metadata["source"].(string); ok {
			match.Source = source
		}
		if ts, ok := result.Metadata["timestamp"].(string); ok {
			timestamp, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				match.Timestamp = timestamp
			}
		}

		// Extract other metadata
		for k, v := range result.Metadata {
			if k == "text" || k == "source" || k == "timestamp" {
				continue
			}
			match.Metadata[k] = v
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// QueryCollection performs a similarity search within a specific collection
func (s *ChromemStore) QueryCollection(ctx context.Context, collection, query string, limit int) ([]EmbeddingMatch, error) {
	// Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Search
	results, err := s.db.Query(ctx, collection, vector, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results
	matches := make([]EmbeddingMatch, 0, len(results))
	for _, result := range results {
		match := EmbeddingMatch{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract fields from metadata
		if text, ok := result.Metadata["text"].(string); ok {
			match.Text = text
		}
		if source, ok := result.Metadata["source"].(string); ok {
			match.Source = source
		}
		if ts, ok := result.Metadata["timestamp"].(string); ok {
			timestamp, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				match.Timestamp = timestamp
			}
		}

		// Extract other metadata
		for k, v := range result.Metadata {
			if k == "text" || k == "source" || k == "timestamp" {
				continue
			}
			match.Metadata[k] = v
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// ListCollections returns all collections
func (s *ChromemStore) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := s.db.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, nil
}

// Close closes the database
func (s *ChromemStore) Close() error {
	return s.db.Close()
}