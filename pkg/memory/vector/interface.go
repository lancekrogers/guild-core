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