package rag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/guild-ventures/guild-core/pkg/corpus"
)

// Test the private newFactory function
func TestNewFactory_Coverage(t *testing.T) {
	ctx := context.Background()
	
	// Test with nil embedder
	_, err := newFactory(ctx, nil, Config{})
	assert.Error(t, err)
	
	// Test with valid embedder
	embedder := &MockEmbedder{}
	factory, err := newFactory(ctx, embedder, Config{
		CollectionName: "test",
	})
	assert.NoError(t, err)
	assert.NotNil(t, factory)
	assert.NotNil(t, factory.retriever)
	assert.Equal(t, embedder, factory.embedder)
	
	err = factory.Close()
	assert.NoError(t, err)
}

// Test the private newRetriever function
func TestNewRetriever_Coverage(t *testing.T) {
	ctx := context.Background()
	
	// Test with nil embedder
	_, err := newRetriever(ctx, nil, Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedder is required")
	
	// Test with valid embedder
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_collection",
		ChunkSize:      1500,
		ChunkOverlap:   250,
		ChunkStrategy:  "sentence",
		UseCorpus:      true,
		CorpusPath:     "/test/path",
	}
	
	retriever, err := newRetriever(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)
	assert.NotNil(t, retriever.vectorStore)
	assert.NotNil(t, retriever.chunker)
	assert.Equal(t, embedder, retriever.embedder)
	assert.Equal(t, config.ChunkSize, retriever.chunker.Config.ChunkSize)
	assert.Equal(t, config.ChunkOverlap, retriever.chunker.Config.ChunkOverlap)
	
	err = retriever.Close()
	assert.NoError(t, err)
}

// Test corpus score calculation
func TestCalculateCorpusScore_Coverage(t *testing.T) {
	retriever := &Retriever{}
	
	// Test title + body match
	doc1 := &corpus.CorpusDoc{
		Title: "Test Document About Query",
		Body:  "This body also contains query information",
	}
	score := retriever.calculateCorpusScore(doc1, "query")
	assert.Equal(t, float32(0.8), score) // 0.5 (title) + 0.3 (body)
	
	// Test tag match
	doc2 := &corpus.CorpusDoc{
		Title: "Document",
		Body:  "No match here",
		Tags:  []string{"query", "test"},
	}
	score = retriever.calculateCorpusScore(doc2, "query")
	assert.Greater(t, score, float32(0.0))
	
	// Test source match
	doc3 := &corpus.CorpusDoc{
		Title:  "Document",
		Body:   "No match",
		Source: "query-source",
	}
	score = retriever.calculateCorpusScore(doc3, "query")
	assert.Greater(t, score, float32(0.0))
}

// Test sortResultsByScore with empty slice
func TestSortResultsByScore_Empty(t *testing.T) {
	retriever := &Retriever{}
	
	// Empty slice
	results := []SearchResult{}
	retriever.sortResultsByScore(results)
	assert.Empty(t, results)
	
	// Single element
	results = []SearchResult{{Score: 0.5}}
	retriever.sortResultsByScore(results)
	assert.Len(t, results, 1)
}

// Test retriever searchCorpus method
func TestSearchCorpus_Coverage(t *testing.T) {
	ctx := context.Background()
	
	// Create retriever without corpus config
	retriever := &Retriever{
		Config: Config{
			UseCorpus: false,
		},
	}
	
	// Should return empty results when corpus is disabled
	results, err := retriever.searchCorpus(ctx, "test", 5)
	assert.NoError(t, err)
	assert.Empty(t, results)
	
	// With corpus config but no path
	retriever.Config.UseCorpus = true
	retriever.corpusConfig = &corpus.Config{}
	
	results, err = retriever.searchCorpus(ctx, "test", 5)
	assert.Error(t, err) // Should fail without valid corpus path
}

// Test AddCorpusDocument
func TestAddCorpusDocument_Coverage(t *testing.T) {
	ctx := context.Background()
	
	// Create retriever without vector store (will fail)
	retriever := &Retriever{
		Config: Config{
			CollectionName: "test",
		},
		chunker: newChunker(ChunkerConfig{
			ChunkSize:    500,
			ChunkOverlap: 50,
		}),
	}
	
	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is test content that should be chunked",
		FilePath: "test.md",
		Source:   "test",
	}
	
	// Will fail without vector store, but tests the code path
	err := retriever.AddCorpusDocument(ctx, doc)
	assert.Error(t, err)
}

// Test DefaultRetrieverWithStoreFactory
func TestDefaultRetrieverWithStoreFactory_Coverage(t *testing.T) {
	// This function returns a function, not a retriever
	config := Config{
		CollectionName: "test",
	}
	
	factoryFunc := DefaultRetrieverWithStoreFactory(nil, config)
	assert.NotNil(t, factoryFunc)
	
	// The returned function would create a retriever when called with proper parameters
}

// Test AgentWrapper enhanceRequestWithRAG
func TestEnhanceRequestWithRAG_Coverage(t *testing.T) {
	ctx := context.Background()
	
	// Test with nil retriever
	wrapper := &AgentWrapper{
		retriever: nil,
	}
	
	result, err := wrapper.enhanceRequestWithRAG(ctx, "test request")
	assert.NoError(t, err)
	assert.Equal(t, "test request", result)
	
	// Test with retriever but no results
	retriever := &Retriever{
		Config: DefaultConfig(),
	}
	wrapper.retriever = retriever
	wrapper.config = DefaultConfig()
	
	// This will fail to retrieve but should handle gracefully
	result, err = wrapper.enhanceRequestWithRAG(ctx, "test request")
	assert.Error(t, err) // Expected to fail without proper setup
}

// Test EnhancePrompt method on AgentWrapper
func TestAgentWrapper_EnhancePromptMethod(t *testing.T) {
	ctx := context.Background()
	
	// Create wrapper with minimal retriever
	retriever := &Retriever{
		Config: DefaultConfig(),
	}
	
	wrapper := &AgentWrapper{
		retriever: retriever,
		config:    DefaultConfig(),
	}
	
	// This will fail without proper retriever setup, but tests the path
	_, err := wrapper.EnhancePrompt(ctx, "prompt", "query", RetrievalConfig{
		MaxResults: 3,
	})
	assert.Error(t, err) // Expected to fail
}