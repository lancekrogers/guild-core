package vector

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbedder is a mock implementation of the Embedder interface for testing
type MockEmbedder struct {
	embeddings map[string][]float32
}

// NewMockEmbedder creates a new mock embedder
func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		embeddings: make(map[string][]float32),
	}
}

// Embed returns a mock embedding for the given text
func (m *MockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	// Check if we have a predefined embedding for this text
	if embedding, ok := m.embeddings[text]; ok {
		return embedding, nil
	}

	// Otherwise, generate a simple mock embedding
	// For testing, we'll use the length of the text as the first value
	// and some arbitrary values for the rest
	embedding := make([]float32, 4)
	embedding[0] = float32(len(text))
	embedding[1] = 0.5
	embedding[2] = 0.3
	embedding[3] = 0.2

	// Save it for future use
	m.embeddings[text] = embedding

	return embedding, nil
}

// SetEmbedding sets a predefined embedding for the given text
func (m *MockEmbedder) SetEmbedding(text string, embedding []float32) {
	m.embeddings[text] = embedding
}

func TestChromemStore(t *testing.T) {
	// Create a temporary directory for persistence
	tempDir, err := os.MkdirTemp("", "chromem-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock embedder
	embedder := NewMockEmbedder()

	// Set up some test embeddings with specific distances
	embedder.SetEmbedding("query", []float32{1.0, 0.0, 0.0, 0.0})
	embedder.SetEmbedding("exact match", []float32{1.0, 0.0, 0.0, 0.0})
	embedder.SetEmbedding("close match", []float32{0.9, 0.1, 0.0, 0.0})
	embedder.SetEmbedding("distant match", []float32{0.5, 0.5, 0.0, 0.0})
	embedder.SetEmbedding("unrelated", []float32{0.0, 0.0, 1.0, 0.0})

	// Create a Chromem store with persistence
	config := Config{
		Embedder:         embedder,
		PersistencePath:  tempDir,
		DefaultDimension: 4,
	}

	store, err := NewChromemStore(config)
	require.NoError(t, err)
	defer store.Close()

	// Add test data
	ctx := context.Background()
	embeddings := []Embedding{
		{
			ID:        "1",
			Text:      "exact match",
			Source:    "test",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"collection": "test",
				"type":       "document",
			},
		},
		{
			ID:        "2",
			Text:      "close match",
			Source:    "test",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"collection": "test",
				"type":       "document",
			},
		},
		{
			ID:        "3",
			Text:      "distant match",
			Source:    "test",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"collection": "test",
				"type":       "document",
			},
		},
		{
			ID:        "4",
			Text:      "unrelated",
			Source:    "test",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"collection": "test",
				"type":       "document",
			},
		},
	}

	// Save embeddings
	for _, embedding := range embeddings {
		err := store.SaveEmbedding(ctx, embedding)
		require.NoError(t, err)
	}

	// Test search
	t.Run("QueryEmbeddings", func(t *testing.T) {
		results, err := store.QueryEmbeddings(ctx, "query", 3)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Results should be sorted by score (highest first)
		assert.Equal(t, "exact match", results[0].Text)
		assert.Equal(t, "close match", results[1].Text)
		assert.Equal(t, "distant match", results[2].Text)

		// Check scores are ordered (descending)
		assert.Greater(t, results[0].Score, results[1].Score)
		assert.Greater(t, results[1].Score, results[2].Score)
		// Ensure all scores are positive
		assert.Greater(t, results[0].Score, float32(0.0))
		assert.Greater(t, results[1].Score, float32(0.0))
		assert.Greater(t, results[2].Score, float32(0.0))
	})

	// Test query with limit
	t.Run("QueryWithLimit", func(t *testing.T) {
		results, err := store.QueryEmbeddings(ctx, "query", 2)
		require.NoError(t, err)
		require.Len(t, results, 2)

		// Only the top 2 results should be returned
		assert.Equal(t, "exact match", results[0].Text)
		assert.Equal(t, "close match", results[1].Text)
	})

}
