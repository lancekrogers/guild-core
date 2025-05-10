package vector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const (
	// DefaultQdrantCollection is the default collection name
	DefaultQdrantCollection = "guild_documents"
	
	// DefaultQdrantDimension is the default embedding dimension
	DefaultQdrantDimension = 1536 // OpenAI embedding dimension
)

// QdrantConfig contains configuration for the Qdrant vector store
type QdrantConfig struct {
	// Address is the address of the Qdrant server (e.g., "localhost:6334")
	Address string

	// Collection is the name of the collection to use
	Collection string

	// Dimension is the dimension of the vector embeddings
	Dimension uint64

	// EmbeddingProvider is the provider used to generate embeddings
	EmbeddingProvider EmbeddingProvider
}

// QdrantStore implements VectorStore using Qdrant
type QdrantStore struct {
	config            *QdrantConfig
	embeddingProvider EmbeddingProvider
	documents         map[string]*Document // In-memory document storage for stub implementation
}

// NewQdrantStore creates a new Qdrant vector store
func NewQdrantStore(ctx context.Context, config *QdrantConfig) (*QdrantStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	if config.Collection == "" {
		config.Collection = DefaultQdrantCollection
	}

	if config.Dimension == 0 {
		config.Dimension = DefaultQdrantDimension
	}

	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("embedding provider cannot be nil")
	}

	return &QdrantStore{
		config:            config,
		embeddingProvider: config.EmbeddingProvider,
		documents:         make(map[string]*Document),
	}, nil
}

// Store stores a document with its embedding
func (s *QdrantStore) Store(ctx context.Context, doc *Document) error {
	var err error
	var embedding Vector

	// If the document doesn't have an embedding, generate one
	if doc.Embedding == nil {
		embedding, err = s.embeddingProvider.GetEmbedding(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		doc.Embedding = embedding
	}

	// Ensure the document has an ID
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}

	// Store the document in memory
	s.documents[doc.ID] = doc

	return nil
}

// StoreMany stores multiple documents with their embeddings
func (s *QdrantStore) StoreMany(ctx context.Context, docs []*Document) error {
	for _, doc := range docs {
		if err := s.Store(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// Retrieve retrieves a document by ID
func (s *QdrantStore) Retrieve(ctx context.Context, id string) (*Document, error) {
	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("document with ID %s not found", id)
	}
	return doc, nil
}

// Query performs a similarity search on the vector store
func (s *QdrantStore) Query(ctx context.Context, embedding Vector, limit int) ([]*QueryResult, error) {
	// This is a stub implementation that doesn't actually perform vector similarity search
	// In a real implementation, this would call the Qdrant API
	
	if len(s.documents) == 0 {
		return []*QueryResult{}, nil
	}
	
	// Return some dummy results
	results := make([]*QueryResult, 0, min(limit, len(s.documents)))
	count := 0
	
	for _, doc := range s.documents {
		if count >= limit {
			break
		}
		
		results = append(results, &QueryResult{
			Document: doc,
			Score:    0.5, // Dummy score
		})
		count++
	}
	
	return results, nil
}

// QueryByText performs a similarity search using text
func (s *QdrantStore) QueryByText(ctx context.Context, text string, limit int) ([]*QueryResult, error) {
	// Generate embedding for the query text
	embedding, err := s.embeddingProvider.GetEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Search using the embedding
	return s.Query(ctx, embedding, limit)
}

// Delete removes a document from the vector store
func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	delete(s.documents, id)
	return nil
}

// DeleteMany removes multiple documents from the vector store
func (s *QdrantStore) DeleteMany(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.documents, id)
	}
	return nil
}

// Close closes the vector store
func (s *QdrantStore) Close() error {
	// Nothing to do for the stub implementation
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}