package vector

import (
	"context"
	"time"
)

// Vector represents a high-dimensional vector used for similarity search
type Vector []float32

// Embedding represents a document with its vector embedding
type Embedding struct {
	// ID is the unique identifier for the document
	ID string

	// Text is the original text content
	Text string

	// Vector is the embedding vector
	Vector []float32

	// Source is where the text came from
	Source string

	// Timestamp is when the text was embedded
	Timestamp time.Time

	// Metadata contains additional information about the document
	Metadata map[string]interface{}
}

// EmbeddingMatch represents a result from a similarity search
type EmbeddingMatch struct {
	// ID is the unique identifier for the document
	ID string

	// Text is the matched text
	Text string

	// Source is where the text came from
	Source string

	// Score is the similarity score (higher is more similar)
	Score float32

	// Timestamp is when the text was embedded
	Timestamp time.Time

	// Metadata contains additional information about the document
	Metadata map[string]interface{}
}

// VectorStore is the interface for vector storage and retrieval
type VectorStore interface {
	// SaveEmbedding stores a vector embedding
	SaveEmbedding(ctx context.Context, embedding Embedding) error

	// QueryEmbeddings performs a similarity search
	QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error)

	// Close closes the vector store
	Close() error
}

// Embedder generates embeddings from text
type Embedder interface {
	// Embed generates an embedding from text
	Embed(ctx context.Context, text string) ([]float32, error)
}

// EmbeddingProvider is a deprecated alias for Embedder, kept for backward compatibility
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	// GetEmbedding is an alias for Embed for backward compatibility
	GetEmbedding(ctx context.Context, text string) ([]float32, error)
	// GetEmbeddings gets embeddings for multiple texts
	GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// Document represents a document with its embedding
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  interface{}            `json:"metadata,omitempty"`
}

// QueryResult represents a search result from the vector store
type QueryResult struct {
	ID       string                 `json:"id"`
	Document *Document              `json:"document"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents a search result (deprecated, use EmbeddingMatch instead)
type SearchResult struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Score     float32                `json:"score"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}