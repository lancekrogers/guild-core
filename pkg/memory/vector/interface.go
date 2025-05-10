package vector

import (
	"context"
)

// Vector represents a high-dimensional vector used for similarity search
type Vector []float32

// Document represents a document with its vector embedding
type Document struct {
	// ID is the unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional information about the document
	Metadata map[string]string

	// Embedding is the vector representation of the document
	Embedding Vector
}

// QueryResult represents a document returned from a vector query
type QueryResult struct {
	// Document is the matched document
	Document *Document

	// Score is the similarity score (higher is more similar)
	Score float32
}

// SearchResult represents a low-level search result with ID and similarity score
type SearchResult struct {
	// ID is the unique identifier for the vector
	ID string

	// Vector is the vector data
	Vector Vector

	// Score is the similarity score (higher is more similar)
	Score float32

	// Metadata contains additional information about the vector
	Metadata map[string]string
}

// VectorStore is the interface for vector storage and retrieval
type VectorStore interface {
	// Store stores a document with its embedding
	Store(ctx context.Context, doc *Document) error

	// StoreMany stores multiple documents with their embeddings
	StoreMany(ctx context.Context, docs []*Document) error

	// Retrieve retrieves a document by ID
	Retrieve(ctx context.Context, id string) (*Document, error)

	// Query performs a similarity search on the vector store
	Query(ctx context.Context, embedding Vector, limit int) ([]*QueryResult, error)

	// QueryByText performs a similarity search using text
	// This assumes the store can generate embeddings for the query
	QueryByText(ctx context.Context, text string, limit int) ([]*QueryResult, error)

	// Delete removes a document from the vector store
	Delete(ctx context.Context, id string) error

	// DeleteMany removes multiple documents from the vector store
	DeleteMany(ctx context.Context, ids []string) error

	// Close closes the vector store
	Close() error
}

// EmbeddingProvider generates embeddings from text
type EmbeddingProvider interface {
	// GetEmbedding generates an embedding for the given text
	GetEmbedding(ctx context.Context, text string) (Vector, error)

	// GetEmbeddings generates embeddings for multiple texts
	GetEmbeddings(ctx context.Context, texts []string) ([]Vector, error)
}