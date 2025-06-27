// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/corpus"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/stretchr/testify/assert"
)

// Test ChunkWithMetadata function
func TestChunkWithMetadata_Coverage(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    100,
		ChunkOverlap: 20,
		Strategy:     ChunkByParagraph,
	}
	chunker := newChunker(config)

	text := "This is a test document that should be chunked properly."
	chunks := chunker.ChunkWithMetadata(text)

	assert.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0].Content)
	assert.Equal(t, 0, chunks[0].Index)
	assert.Equal(t, 0, chunks[0].Metadata["chunk_index"])
	assert.Equal(t, 1, chunks[0].Metadata["total_chunks"])
	assert.Equal(t, "paragraph", chunks[0].Metadata["strategy"])
	assert.Equal(t, 100, chunks[0].Metadata["chunk_size"])
	assert.Equal(t, 20, chunks[0].Metadata["overlap"])
}

// Test SearchCorpus function
func TestSearchCorpus_Additional(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	corpusConfig := corpus.Config{
		CorpusPath: tempDir,
	}

	// Create a simple document
	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is a test document with some content",
		FilePath: tempDir + "/test.md",
	}

	err := corpus.Save(ctx, doc, corpusConfig)
	assert.NoError(t, err)

	// Search for it
	results, err := SearchCorpus(ctx, "test", corpusConfig, 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, float32(0.9), results[0].Score)
	assert.Contains(t, results[0].Content, "test document")
}

// Test retriever methods that are not covered
func TestRetriever_Methods_Coverage(t *testing.T) {
	// Test sortResultsByScore
	retriever := &Retriever{}
	results := []SearchResult{
		{Score: 0.3, Content: "Low"},
		{Score: 0.9, Content: "High"},
		{Score: 0.6, Content: "Medium"},
	}

	retriever.sortResultsByScore(results)
	assert.Equal(t, "High", results[0].Content)
	assert.Equal(t, "Medium", results[1].Content)
	assert.Equal(t, "Low", results[2].Content)

	// Test with empty slice
	emptyResults := []SearchResult{}
	retriever.sortResultsByScore(emptyResults)
	assert.Empty(t, emptyResults)
}

// Test the retriever with different chunk strategies
func TestRetriever_NewWithStrategies(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	strategies := []struct {
		input    string
		expected ChunkStrategy
	}{
		{"sentence", ChunkBySentence},
		{"fixed", ChunkByFixedSize},
		{"markdown", ChunkByMarkdownHeader},
		{"unknown", ChunkByParagraph}, // Default
	}

	for _, test := range strategies {
		config := Config{
			CollectionName: "test_" + test.input,
			ChunkSize:      500,
			ChunkOverlap:   50,
			ChunkStrategy:  test.input,
		}

		retriever, err := newRetriever(ctx, embedder, config)
		assert.NoError(t, err)
		assert.NotNil(t, retriever)
		assert.Equal(t, test.expected, retriever.chunker.Config.Strategy)
		retriever.Close()
	}
}

// Test AddDocument method
func TestRetriever_AddDocument_Additional(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_add",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)
	defer retriever.Close()

	// Test with empty content
	err = retriever.AddDocument(ctx, "doc1", "", "empty.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document content cannot be empty")

	// Test with small content
	err = retriever.AddDocument(ctx, "doc2", "Small content", "small.txt")
	assert.NoError(t, err)
}

// Test RemoveDocument method
func TestRetriever_RemoveDocument_Additional(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_remove",
	}

	retriever, err := newRetriever(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)
	defer retriever.Close()

	// Remove a document (even if it doesn't exist)
	err = retriever.RemoveDocument(ctx, "doc123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document removal not yet implemented")
}

// Test EnhancePrompt method
func TestRetriever_EnhancePrompt_Additional(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_enhance",
	}

	retriever, err := newRetriever(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)
	defer retriever.Close()

	// Test with empty prompt
	result, err := retriever.EnhancePrompt(ctx, "", RetrievalConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
	assert.Equal(t, "", result)

	// Test with simple prompt (will have no results)
	result, err = retriever.EnhancePrompt(ctx, "test prompt", RetrievalConfig{
		MaxResults: 3,
		MinScore:   0.5,
	})
	assert.NoError(t, err)
	assert.Equal(t, "test prompt", result) // No enhancement without results
}

// Test DefaultRetrieverFactory
func TestDefaultRetrieverFactory_Coverage(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_factory",
		ChunkSize:      1000,
		ChunkOverlap:   100,
		ChunkStrategy:  "sentence",
	}

	retriever, err := DefaultRetrieverFactory(ctx, embedder, config)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)

	// Verify it implements the interface
	_, ok := retriever.(RetrieverInterface)
	assert.True(t, ok)

	err = retriever.Close()
	assert.NoError(t, err)
}

// Test DefaultRetrieverWithStoreFactory
func TestDefaultRetrieverWithStoreFactory_Additional(t *testing.T) {
	// Create a mock store
	store := &mockVectorStore{}

	config := Config{
		CollectionName: "test_store",
		ChunkSize:      500,
		ChunkOverlap:   50,
	}

	retrieverFunc := DefaultRetrieverWithStoreFactory(store, config)
	assert.NotNil(t, retrieverFunc)

	// The function returns a factory function, not a retriever directly
	// This is the expected behavior based on the implementation
}

// Test agent wrapper methods
func TestAgentWrapper_Methods(t *testing.T) {
	ctx := context.Background()

	// Test with nil retriever
	wrapper := &AgentWrapper{
		retriever: nil,
		config:    DefaultConfig(),
	}

	result, err := wrapper.enhanceRequestWithRAG(ctx, "test request")
	assert.NoError(t, err)
	assert.Equal(t, "test request", result)

	// Test NewAgentWrapper
	agent := &mockGuildArtisan{}
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_wrapper",
	}

	retriever, _ := newRetriever(ctx, embedder, config)
	wrapper2 := NewAgentWrapper(agent, retriever, config)
	assert.NotNil(t, wrapper2)
	assert.Equal(t, agent, wrapper2.agent)
	assert.Equal(t, retriever, wrapper2.retriever)
	assert.Equal(t, config, wrapper2.config)
}

// Mock GuildArtisan for testing
type mockGuildArtisan struct{}

func (m *mockGuildArtisan) Execute(ctx context.Context, request string) (string, error) {
	return "executed: " + request, nil
}
func (m *mockGuildArtisan) GetID() string                                      { return "mock-agent" }
func (m *mockGuildArtisan) GetName() string                                    { return "Mock Agent" }
func (m *mockGuildArtisan) GetToolRegistry() tools.Registry                    { return nil }
func (m *mockGuildArtisan) GetCommissionManager() commission.CommissionManager { return nil }
func (m *mockGuildArtisan) GetLLMClient() providers.LLMClient                  { return nil }
func (m *mockGuildArtisan) GetMemoryManager() memory.ChainManager              { return nil }
func (m *mockGuildArtisan) GetType() string                                    { return "mock" }
func (m *mockGuildArtisan) GetCapabilities() []string                          { return []string{"testing"} }

// Test the rag agent wrapper Execute method
func TestAgentWrapper_Execute_Additional(t *testing.T) {
	ctx := context.Background()

	agent := &mockGuildArtisan{}
	wrapper := &AgentWrapper{
		agent:     agent,
		retriever: nil, // No retriever
		config:    DefaultConfig(),
	}

	// Execute should work even without retriever
	result, err := wrapper.Execute(ctx, "test request")
	assert.NoError(t, err)
	assert.Equal(t, "executed: test request", result)

	// Test other delegated methods
	assert.Equal(t, "mock-agent", wrapper.GetID())
	assert.Equal(t, "Mock Agent", wrapper.GetName())
	assert.Nil(t, wrapper.GetToolRegistry())
	assert.Nil(t, wrapper.GetCommissionManager())
	assert.Nil(t, wrapper.GetLLMClient())
	assert.Nil(t, wrapper.GetMemoryManager())
}
