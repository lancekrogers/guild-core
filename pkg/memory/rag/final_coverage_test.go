package rag

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create a fully functional mock vector store
type functionalMockVectorStore struct {
	embeddings map[string][]vector.Embedding
}

func newFunctionalMockVectorStore() *functionalMockVectorStore {
	return &functionalMockVectorStore{
		embeddings: make(map[string][]vector.Embedding),
	}
}

func (m *functionalMockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	// Store by collection (use ID as collection key for simplicity)
	collection := "default"
	if embedding.ID != "" {
		collection = embedding.ID[:1] // Use first char as collection
	}
	m.embeddings[collection] = append(m.embeddings[collection], embedding)
	return nil
}

func (m *functionalMockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	results := []vector.EmbeddingMatch{}

	// Return mock matches based on query
	for _, embeddings := range m.embeddings {
		for i, emb := range embeddings {
			if i < limit {
				results = append(results, vector.EmbeddingMatch{
					ID:        emb.ID,
					Text:      emb.Text,
					Source:    emb.Source,
					Score:     0.8,
					Timestamp: emb.Timestamp,
					Metadata:  emb.Metadata,
				})
			}
		}
	}

	return results, nil
}

func (m *functionalMockVectorStore) Close() error {
	return nil
}

// Test config functions
func TestConfigFunctions(t *testing.T) {
	// Test DefaultConfig
	config := DefaultConfig()
	assert.Equal(t, "rag_embeddings", config.CollectionName)
	assert.Equal(t, 1000, config.ChunkSize)
	assert.Equal(t, 200, config.ChunkOverlap)
	assert.Equal(t, "paragraph", config.ChunkStrategy)
	assert.Equal(t, 5, config.MaxResults)
	assert.Equal(t, "", config.VectorStorePath)
	assert.Equal(t, "", config.CorpusPath)
	assert.False(t, config.UseCorpus)

	// Test DefaultRetrievalConfig
	retrieval := DefaultRetrievalConfig()
	assert.Equal(t, 5, retrieval.MaxResults)
	assert.Equal(t, float32(0.0), retrieval.MinScore)
	assert.False(t, retrieval.IncludeMetadata)
	assert.False(t, retrieval.UseCorpus)
}

// Test GetConfig and GetChunker
func TestChunkerMethods(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    1000,
		ChunkOverlap: 100,
		Strategy:     ChunkBySentence,
	}

	chunker := newChunker(config)

	// Test GetConfig
	gotConfig := chunker.GetConfig()
	assert.Equal(t, config, gotConfig)

	// chunker doesn't have GetChunker method, skip this test
}

// Test SearchCorpus functionality
func TestSearchCorpusFunction(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	corpusConfig := corpus.Config{
		CorpusPath: tempDir,
	}

	// Create test documents
	docs := []*corpus.CorpusDoc{
		{
			Title:    "Go Programming Guide",
			Body:     "Learn Go programming language basics",
			FilePath: tempDir + "/go.md",
			Source:   "tutorial",
		},
		{
			Title:    "Python Tutorial",
			Body:     "Python is a high-level programming language",
			FilePath: tempDir + "/python.md",
			Source:   "tutorial",
		},
	}

	// Save documents
	for _, doc := range docs {
		err := corpus.Save(ctx, doc, corpusConfig)
		require.NoError(t, err)
	}

	// Search
	results, err := SearchCorpus(ctx, "programming", corpusConfig, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify results
	for _, result := range results {
		assert.Equal(t, float32(0.9), result.Score)
		assert.NotEmpty(t, result.Content)
		assert.NotEmpty(t, result.Source)
	}

	// Test containsIgnoreCase directly
	assert.True(t, containsIgnoreCase("Hello World", "hello"))
	assert.True(t, containsIgnoreCase("HELLO", "hello"))
	assert.False(t, containsIgnoreCase("Hello", "world"))
}

// Test retriever.searchCorpus method
func TestRetrieverSearchCorpus(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	retriever := &Retriever{
		Config: Config{
			UseCorpus: true,
		},
		corpusConfig: &corpus.Config{
			CorpusPath: tempDir,
		},
	}

	// Create a test document
	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is test content",
		FilePath: tempDir + "/test.md",
		Source:   "test",
		Tags:     []string{"test"},
	}

	err := corpus.Save(ctx, doc, *retriever.corpusConfig)
	require.NoError(t, err)

	// Search
	results, err := retriever.searchCorpus(ctx, "test", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Greater(t, results[0].Score, float32(0))

	// Test with corpus disabled - searchCorpus is internal and still works
	// UseCorpus is checked at a higher level in RetrieveContext
	retriever.Config.UseCorpus = false
	results, err = retriever.searchCorpus(ctx, "test", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1) // searchCorpus still returns results
}

// Test calculateCorpusScore
func TestCalculateCorpusScore(t *testing.T) {
	retriever := &Retriever{}

	doc := &corpus.CorpusDoc{
		Title:  "Machine Learning Guide",
		Body:   "Introduction to machine learning concepts",
		Tags:   []string{"ml", "ai", "learning"},
		Source: "tutorial",
	}

	// Test title match
	score := retriever.calculateCorpusScore(doc, "machine")
	assert.Greater(t, score, float32(0))

	// Test body match
	score = retriever.calculateCorpusScore(doc, "introduction")
	assert.Greater(t, score, float32(0))

	// Test tag match
	score = retriever.calculateCorpusScore(doc, "ml")
	assert.Greater(t, score, float32(0))

	// Test no match
	score = retriever.calculateCorpusScore(doc, "notfound")
	assert.Equal(t, float32(0), score)
}

// Test RetrieveContext
func TestRetrieverRetrieveContext(t *testing.T) {
	ctx := context.Background()

	// Create retriever with mock store
	mockStore := newFunctionalMockVectorStore()
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
		},
	}

	// Add test embeddings
	testEmbeddings := []vector.Embedding{
		{
			ID:        "doc1",
			Text:      "Machine learning is fascinating",
			Source:    "ml.txt",
			Vector:    []float32{0.1, 0.2, 0.3},
			Timestamp: time.Now(),
		},
		{
			ID:        "doc2",
			Text:      "Deep learning uses neural networks",
			Source:    "dl.txt",
			Vector:    []float32{0.4, 0.5, 0.6},
			Timestamp: time.Now(),
		},
	}

	for _, emb := range testEmbeddings {
		err := mockStore.SaveEmbedding(ctx, emb)
		assert.NoError(t, err)
	}

	// Test retrieval
	config := RetrievalConfig{
		MaxResults:      5,
		MinScore:        0.5,
		IncludeMetadata: true,
	}

	results, err := retriever.RetrieveContext(ctx, "learning", config)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, len(results.Results), 0)

	// Test with empty query
	results, err = retriever.RetrieveContext(ctx, "", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

// Test EnhancePrompt on retriever
func TestRetrieverEnhancePrompt(t *testing.T) {
	ctx := context.Background()

	mockStore := newFunctionalMockVectorStore()
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
		},
	}

	// Add test data
	err := mockStore.SaveEmbedding(ctx, vector.Embedding{
		ID:     "doc1",
		Text:   "AI is transforming technology",
		Source: "ai.txt",
	})
	assert.NoError(t, err)

	config := RetrievalConfig{
		MaxResults: 3,
		MinScore:   0.5,
	}

	// Test enhancement
	enhanced, err := retriever.EnhancePrompt(ctx, "Tell me about AI", config)
	assert.NoError(t, err)
	assert.Contains(t, enhanced, "Tell me about AI")
	assert.Contains(t, enhanced, "# Context")

	// Test with empty prompt
	enhanced, err = retriever.EnhancePrompt(ctx, "", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

// Test AddCorpusDocument
func TestAddCorpusDocument(t *testing.T) {
	ctx := context.Background()

	mockStore := newFunctionalMockVectorStore()
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
		},
		chunker: newChunker(ChunkerConfig{
			ChunkSize:    100,
			ChunkOverlap: 20,
		}),
	}

	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is a test document with enough content to be processed",
		FilePath: "test.md",
		Source:   "test",
	}

	err := retriever.AddCorpusDocument(ctx, doc)
	assert.NoError(t, err)

	// Test with empty content
	emptyDoc := &corpus.CorpusDoc{
		Title:    "Empty",
		Body:     "",
		FilePath: "empty.md",
	}

	err = retriever.AddCorpusDocument(ctx, emptyDoc)
	assert.NoError(t, err) // Should handle gracefully
}

// Test RemoveDocument
func TestRemoveDocument(t *testing.T) {
	retriever := &Retriever{}

	err := retriever.RemoveDocument(context.Background(), "doc1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

// Test DefaultRetrieverWithStoreFactory
func TestDefaultRetrieverWithStoreFactory(t *testing.T) {
	store := newFunctionalMockVectorStore()
	config := Config{
		CollectionName: "test",
	}

	factoryFunc := DefaultRetrieverWithStoreFactory(store, config)
	assert.NotNil(t, factoryFunc)

	// The factory function would be called with context and embedder
	// We're just testing that it returns a function
}

// Test DefaultRetrieverFactory
func TestDefaultRetrieverFactory(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_factory",
	}

	retriever, err := DefaultRetrieverFactory(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)

	// Clean up
	err = retriever.Close()
	assert.NoError(t, err)
}

// Test the enhanced request with RAG
func TestEnhanceRequestWithRAG(t *testing.T) {
	ctx := context.Background()

	mockStore := newFunctionalMockVectorStore()
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
			MaxResults:     3,
		},
	}

	// Add test data
	err := mockStore.SaveEmbedding(ctx, vector.Embedding{
		ID:     "doc1",
		Text:   "Machine learning algorithms",
		Source: "ml.txt",
	})
	assert.NoError(t, err)

	agent := &mockGuildArtisan{}
	wrapper := NewAgentWrapper(agent, retriever, retriever.Config)

	// Test enhancement
	enhanced, err := wrapper.enhanceRequestWithRAG(ctx, "Tell me about ML")
	assert.NoError(t, err)
	assert.Contains(t, enhanced, "relevant context")
	assert.Contains(t, enhanced, "Tell me about ML")
}
