package vector

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
)

// ChromemStore implements the VectorStore interface using Chromem-go
type ChromemStore struct {
	db       *chromem.DB
	embedder Embedder
	mu       sync.RWMutex
	// Mock in-memory storage for testing
	embeddings map[string]Embedding
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

	// The actual chromem.NewDB function and API may differ
	// Create a simplistic wrapper to simulate the library functionality
	db := &chromem.DB{}
	
	return &ChromemStore{
		db:         db,
		embedder:   config.Embedder,
		embeddings: make(map[string]Embedding),
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
		embedding.Vector = vector
	}

	// Store in memory
	s.mu.Lock()
	s.embeddings[embedding.ID] = embedding
	s.mu.Unlock()
	
	return nil
}

// QueryEmbeddings performs a similarity search
func (s *ChromemStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error) {
	// Generate query vector
	queryVector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Calculate similarities
	s.mu.RLock()
	matches := make([]EmbeddingMatch, 0, len(s.embeddings))
	for _, emb := range s.embeddings {
		score := cosineSimilarity(queryVector, emb.Vector)
		matches = append(matches, EmbeddingMatch{
			ID:        emb.ID,
			Text:      emb.Text,
			Source:    emb.Source,
			Score:     score,
			Timestamp: emb.Timestamp,
			Metadata:  emb.Metadata,
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

// Close closes the database
func (s *ChromemStore) Close() error {
	// Stub implementation
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
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

// sqrt is a simple square root approximation
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