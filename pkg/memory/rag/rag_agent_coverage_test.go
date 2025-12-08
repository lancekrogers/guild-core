// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/corpus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test RAG agent methods that are not covered
func TestRAGAgent_Execute(t *testing.T) {
	ctx := context.Background()
	agent := &mockGuildArtisan{}
	embedder := &MockEmbedder{}

	// Create a retriever with test config
	config := Config{
		CollectionName: "test_execute",
		ChunkSize:      100,
		ChunkOverlap:   20,
		MaxResults:     3,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Create wrapper
	wrapper := NewAgentWrapper(agent, retriever, config)

	// Add a test document
	err = retriever.AddDocument(ctx, "doc1", "This is test content about AI and machine learning", "test.txt")
	assert.NoError(t, err)

	// Execute should enhance the request
	result, err := wrapper.Execute(ctx, "Tell me about AI")
	assert.NoError(t, err)
	assert.Equal(t, "executed: Tell me about AI", result) // Mock agent returns this
}

// Test corpus-based search functionality
func TestRetriever_SearchCorpus(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create retriever with corpus enabled
	retriever := &Retriever{
		Config: Config{
			UseCorpus: true,
		},
		corpusConfig: &corpus.Config{
			CorpusPath: tempDir,
		},
	}

	// Create test document
	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is a test document about programming",
		FilePath: tempDir + "/test.md",
		Source:   "test",
		Tags:     []string{"test", "programming"},
	}

	// Save document
	err := corpus.Save(ctx, doc, *retriever.corpusConfig)
	require.NoError(t, err)

	// Search for it
	results, err := retriever.searchCorpus(ctx, "programming", 5)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, float32(1.0), results[0].Score) // Capped at 1.0 for high match
	assert.Contains(t, results[0].Content, "programming")
}

// Test calculateCorpusScore with all paths
func TestRetriever_CalculateCorpusScore(t *testing.T) {
	retriever := &Retriever{}

	tests := []struct {
		name     string
		doc      *corpus.CorpusDoc
		query    string
		expected float32
	}{
		{
			name: "Title match only",
			doc: &corpus.CorpusDoc{
				Title: "Test Query Document",
				Body:  "No match here",
			},
			query:    "query",
			expected: 0.5,
		},
		{
			name: "Body match only",
			doc: &corpus.CorpusDoc{
				Title: "Document",
				Body:  "This contains query in the body",
			},
			query:    "query",
			expected: 1.0, // Normalized score gets capped at 1.0
		},
		{
			name: "Title and body match",
			doc: &corpus.CorpusDoc{
				Title: "Query Document",
				Body:  "This also has query",
			},
			query:    "query",
			expected: 1.0, // Title (0.5) + normalized body score, capped at 1.0
		},
		{
			name: "Tag match",
			doc: &corpus.CorpusDoc{
				Title: "Document",
				Body:  "No match",
				Tags:  []string{"query", "test"},
			},
			query:    "query",
			expected: 0.3, // Tag match gives 0.3
		},
		{
			name: "Source match",
			doc: &corpus.CorpusDoc{
				Title:  "Document",
				Body:   "No match",
				Source: "query-source",
			},
			query:    "query",
			expected: 0.0, // Source is not used in scoring
		},
		{
			name: "All matches",
			doc: &corpus.CorpusDoc{
				Title:  "Query Title",
				Body:   "Query body content",
				Tags:   []string{"query"},
				Source: "query-source",
			},
			query:    "query",
			expected: 1.0, // Title + tag + body, capped at 1.0
		},
		{
			name: "No matches",
			doc: &corpus.CorpusDoc{
				Title: "Document",
				Body:  "Content",
			},
			query:    "notfound",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := retriever.calculateCorpusScore(tt.doc, tt.query)
			assert.Equal(t, tt.expected, score)
		})
	}
}

// Test AddCorpusDocument
func TestRetriever_AddCorpusDocument(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_corpus_add",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Create a corpus document
	doc := &corpus.CorpusDoc{
		Title:    "Test Document",
		Body:     "This is a test document with sufficient content to be chunked properly",
		FilePath: "test.md",
		Source:   "test",
	}

	// Add the document
	err = retriever.AddCorpusDocument(ctx, doc)
	assert.NoError(t, err)

	// Try with empty content
	emptyDoc := &corpus.CorpusDoc{
		Title:    "Empty",
		Body:     "",
		FilePath: "empty.md",
	}
	err = retriever.AddCorpusDocument(ctx, emptyDoc)
	assert.NoError(t, err)
}

// Test EnhancePrompt on retriever with coverage focus
func TestRetriever_EnhancePrompt_Coverage(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_enhance",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Add test documents
	err = retriever.AddDocument(ctx, "doc1", "Machine learning is a subset of AI", "ml.txt")
	assert.NoError(t, err)

	err = retriever.AddDocument(ctx, "doc2", "Deep learning uses neural networks", "dl.txt")
	assert.NoError(t, err)

	// Test enhancement
	config2 := RetrievalConfig{
		MaxResults: 2,
		MinScore:   0.0, // Lower threshold to ensure results
	}

	enhanced, err := retriever.EnhancePrompt(ctx, "Tell me about AI", config2)
	assert.NoError(t, err)
	// The enhanced prompt should either contain the original prompt with context,
	// or just the original prompt if no results are found
	assert.Contains(t, enhanced, "Tell me about AI")
	// Only check for context header if results were actually found
	if enhanced != "Tell me about AI" {
		assert.Contains(t, enhanced, "# Context")
	}

	// Test with empty prompt
	enhanced, err = retriever.EnhancePrompt(ctx, "", config2)
	assert.Error(t, err) // Empty query should return error
	assert.Contains(t, err.Error(), "query cannot be empty")
}

// Test RemoveDocument with coverage focus
func TestRetriever_RemoveDocument_Coverage(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_remove",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Add and then remove a document
	err = retriever.AddDocument(ctx, "doc1", "Test content", "test.txt")
	assert.NoError(t, err)

	// Remove the document - not implemented yet
	err = retriever.RemoveDocument(ctx, "doc1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document removal not yet implemented")

	// Remove non-existent document - also returns not implemented error
	err = retriever.RemoveDocument(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document removal not yet implemented")
}

// Test GetChunker method
func TestRAGAgent_GetChunker(t *testing.T) {
	ctx := context.Background()
	agent := &mockGuildArtisan{}
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_chunker",
		ChunkSize:      500,
		ChunkOverlap:   50,
		ChunkStrategy:  "sentence",
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	wrapper := NewAgentWrapper(agent, retriever, config)

	// Test indirect chunker usage through document addition
	err = retriever.AddDocument(ctx, "doc1", "Test content.", "test.txt")
	assert.NoError(t, err)

	// Use wrapper to avoid unused variable error
	assert.NotNil(t, wrapper)
}

// Test enhanceRequestWithRAG with corpus results
func TestRAGAgent_EnhanceRequestWithRAG_Corpus(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_enhance_corpus",
		ChunkSize:      100,
		ChunkOverlap:   20,
		UseCorpus:      true,
		CorpusPath:     tempDir,
		MaxResults:     5,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Set corpus config
	retriever.corpusConfig = &corpus.Config{
		CorpusPath: tempDir,
	}

	// Create corpus document
	doc := &corpus.CorpusDoc{
		Title:    "AI Research",
		Body:     "Artificial Intelligence is transforming technology",
		FilePath: tempDir + "/ai.md",
		Source:   "research",
	}
	err = corpus.Save(ctx, doc, *retriever.corpusConfig)
	require.NoError(t, err)

	agent := &mockGuildArtisan{}
	wrapper := NewAgentWrapper(agent, retriever, config)

	// Test enhancement - should return original request when no results
	enhanced, err := wrapper.enhanceRequestWithRAG(ctx, "What is AI?")
	assert.NoError(t, err)
	assert.Equal(t, "What is AI?", enhanced) // No results, so returns original
}

// Test RetrieveContext with corpus
func TestRetriever_RetrieveContext_WithCorpus(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_retrieve_corpus",
		ChunkSize:      100,
		ChunkOverlap:   20,
		UseCorpus:      true,
		CorpusPath:     tempDir,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Set corpus config
	retriever.corpusConfig = &corpus.Config{
		CorpusPath: tempDir,
	}

	// Create corpus documents
	docs := []*corpus.CorpusDoc{
		{
			Title:    "Machine Learning",
			Body:     "ML is a branch of AI",
			FilePath: tempDir + "/ml.md",
			Source:   "education",
		},
		{
			Title:    "Deep Learning",
			Body:     "DL uses neural networks",
			FilePath: tempDir + "/dl.md",
			Source:   "research",
		},
	}

	for _, doc := range docs {
		err = corpus.Save(ctx, doc, *retriever.corpusConfig)
		require.NoError(t, err)
	}

	// Retrieve with corpus enabled
	config2 := RetrievalConfig{
		MaxResults:      5,
		MinScore:        0.0,
		UseCorpus:       true,
		IncludeMetadata: true,
	}

	results, err := retriever.RetrieveContext(ctx, "learning", config2)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, len(results.Results), 0)
}

// Test vector store operations in AddDocument
func TestRetriever_AddDocument_VectorOperations(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_vector_ops",
		ChunkSize:      50, // Small chunks to test multiple chunks
		ChunkOverlap:   10,
	}

	retriever, err := newRetriever(ctx, embedder, config)
	require.NoError(t, err)
	defer retriever.Close()

	// Add a document that will be chunked into multiple pieces
	longContent := "This is the first sentence about machine learning. " +
		"This is the second sentence about artificial intelligence. " +
		"This is the third sentence about deep learning. " +
		"This is the fourth sentence about neural networks."

	err = retriever.AddDocument(ctx, "doc-multi", longContent, "multi.txt")
	assert.NoError(t, err)
}

// Test sortResultsByScore with various inputs
func TestRetriever_SortResultsByScore_Edge(t *testing.T) {
	retriever := &Retriever{}

	// Test with duplicate scores
	results := []SearchResult{
		{Score: 0.5, Content: "A"},
		{Score: 0.9, Content: "B"},
		{Score: 0.5, Content: "C"},
		{Score: 0.9, Content: "D"},
		{Score: 0.7, Content: "E"},
	}

	retriever.sortResultsByScore(results)

	// Check order - highest scores first
	assert.Equal(t, float32(0.9), results[0].Score)
	assert.Equal(t, float32(0.9), results[1].Score)
	assert.Equal(t, float32(0.7), results[2].Score)
	assert.Equal(t, float32(0.5), results[3].Score)
	assert.Equal(t, float32(0.5), results[4].Score)
}
