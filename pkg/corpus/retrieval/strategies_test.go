// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCraftVectorSearchStrategy_BasicOperation(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	strategy := NewVectorSearchStrategy(mockVectorStore, 0.8)

	assert.Equal(t, "vector_search", strategy.Name())
	assert.Equal(t, 0.8, strategy.Weight())

	// Set up mock expectation
	expectedResults := []VectorResult{
		{
			DocID:   "doc1",
			Content: "Test document about Go programming",
			Score:   0.9,
			Metadata: map[string]interface{}{
				"title": "Go Guide",
			},
		},
	}

	query := Query{
		Text:       "golang programming",
		MaxResults: 10,
		Context: QueryContext{
			CurrentFiles: []string{"main.go"},
			Tags:         []string{"golang"},
		},
	}

	// Enhanced query should include context
	mockVectorStore.On("Search", mock.Anything, mock.MatchedBy(func(q string) bool {
		return q == "golang programming files: main.go tags: golang"
	}), 20).Return(expectedResults, nil)

	docs, err := strategy.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
	assert.Equal(t, "vector_search", docs[0].Metadata["strategy"])
	assert.Equal(t, 0.9, docs[0].Metadata["vector_score"])

	mockVectorStore.AssertExpectations(t)
}

func TestGuildVectorSearchStrategy_QueryEnhancement(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	strategy := NewVectorSearchStrategy(mockVectorStore, 0.8)

	query := Query{
		Text:       "implement authentication",
		MaxResults: 5,
		Context: QueryContext{
			CurrentFiles: []string{"auth.go", "middleware.go"},
			Tags:         []string{"security", "auth"},
			MessageHistory: []Message{
				{Role: "user", Content: "I need help with JWT tokens"},
				{Role: "assistant", Content: "I can help with that"},
				{Role: "user", Content: "How do I validate tokens?"},
			},
		},
	}

	enhanced := strategy.enhanceQuery(query)
	
	// Should include files, tags, and recent messages
	assert.Contains(t, enhanced, "implement authentication")
	assert.Contains(t, enhanced, "files: auth.go middleware.go")
	assert.Contains(t, enhanced, "tags: security auth")
	assert.Contains(t, enhanced, "I need help with JWT tokens")
	assert.Contains(t, enhanced, "How do I validate tokens?")
}

func TestJourneymanKeywordSearchStrategy_BasicOperation(t *testing.T) {
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.6)

	assert.Equal(t, "keyword_search", strategy.Name())
	assert.Equal(t, 0.6, strategy.Weight())

	expectedDocs := []Document{
		{
			ID:      "doc1",
			Content: "Document about golang testing",
			Score:   0.8,
		},
	}

	query := Query{
		Text:       "golang testing framework",
		MaxResults: 10,
	}

	// Should extract keywords and search
	mockIndexer.On("Search", mock.Anything, []string{"golang", "testing", "framework"}, 10).Return(expectedDocs, nil)

	docs, err := strategy.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
	assert.Equal(t, "keyword_search", docs[0].Metadata["strategy"])
	assert.Equal(t, []string{"golang", "testing", "framework"}, docs[0].Metadata["keywords"])

	mockIndexer.AssertExpectations(t)
}

func TestGuildKeywordSearchStrategy_KeywordExtraction(t *testing.T) {
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.6)

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "How do I implement authentication in Go?",
			expected: []string{"implement", "authentication"},
		},
		{
			input:    "The quick brown fox jumps over the lazy dog",
			expected: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
		},
		{
			input:    "API, testing, and performance",
			expected: []string{"api", "testing", "performance"},
		},
		{
			input:    "",
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := strategy.extractKeywords(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", test.input)
	}
}

func TestCraftKeywordSearchStrategy_EmptyQuery(t *testing.T) {
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.6)

	query := Query{
		Text:       "",
		MaxResults: 10,
	}

	docs, err := strategy.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Empty(t, docs)

	// Should not call search if no keywords
	mockIndexer.AssertNotCalled(t, "Search")
}

func TestJourneymanGraphTraversalStrategy_BasicOperation(t *testing.T) {
	mockGraph := &MockKnowledgeGraph{}
	strategy := NewGraphTraversalStrategy(mockGraph, 0.7)

	assert.Equal(t, "graph_traversal", strategy.Name())
	assert.Equal(t, 0.7, strategy.Weight())

	entryNodes := []GraphNode{
		{
			ID:      "node1",
			Type:    "concept",
			Content: "Authentication concept",
		},
	}

	relatedNodes := []GraphNode{
		{
			ID:      "node1",
			Type:    "concept",
			Content: "Authentication concept",
		},
		{
			ID:      "node2",
			Type:    "implementation",
			Content: "JWT implementation",
		},
	}

	query := Query{
		Text:       "authentication implementation",
		MaxResults: 10,
	}

	mockGraph.On("FindEntryNodes", mock.Anything, query).Return(entryNodes, nil)
	mockGraph.On("TraverseRelated", mock.Anything, entryNodes, 2).Return(relatedNodes, nil)

	docs, err := strategy.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "node1", docs[0].ID)
	assert.Equal(t, "node2", docs[1].ID)
	assert.Equal(t, "graph_traversal", docs[0].Metadata["strategy"])
	assert.Equal(t, 1, docs[0].Metadata["entry_nodes_count"])
	assert.Equal(t, 2, docs[0].Metadata["traversal_hops"])

	mockGraph.AssertExpectations(t)
}

func TestCraftGraphTraversalStrategy_NoEntryNodes(t *testing.T) {
	mockGraph := &MockKnowledgeGraph{}
	strategy := NewGraphTraversalStrategy(mockGraph, 0.7)

	query := Query{
		Text:       "nonexistent concept",
		MaxResults: 10,
	}

	// No entry nodes found
	mockGraph.On("FindEntryNodes", mock.Anything, query).Return([]GraphNode{}, nil)

	docs, err := strategy.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Empty(t, docs)

	// Should not call TraverseRelated if no entry nodes
	mockGraph.AssertNotCalled(t, "TraverseRelated")
}

func TestGuildGraphTraversalStrategy_NodeConversion(t *testing.T) {
	mockGraph := &MockKnowledgeGraph{}
	strategy := NewGraphTraversalStrategy(mockGraph, 0.7)

	nodes := []GraphNode{
		{
			ID:      "node1",
			Type:    "concept",
			Content: "Test content",
			Metadata: map[string]interface{}{
				"importance": 0.8,
			},
		},
		{
			ID:   "node2",
			Type: "example",
			Content: "Example content",
		},
	}

	docs := strategy.nodesToDocuments(nodes)

	assert.Len(t, docs, 2)
	assert.Equal(t, "node1", docs[0].ID)
	assert.Equal(t, "Test content", docs[0].Content)
	assert.Equal(t, 1.0, docs[0].Score) // Base score
	assert.Equal(t, "concept", docs[0].Metadata["node_type"])
	assert.Equal(t, 0.8, docs[0].Metadata["importance"])

	assert.Equal(t, "node2", docs[1].ID)
	assert.Equal(t, "example", docs[1].Metadata["node_type"])
}

// Performance and edge case tests

func BenchmarkCraftVectorSearch_QueryEnhancement(b *testing.B) {
	mockVectorStore := &MockVectorStore{}
	strategy := NewVectorSearchStrategy(mockVectorStore, 0.8)

	query := Query{
		Text:       "implement authentication system",
		MaxResults: 10,
		Context: QueryContext{
			CurrentFiles: []string{"auth.go", "middleware.go", "jwt.go"},
			Tags:         []string{"security", "auth", "jwt", "middleware"},
			MessageHistory: []Message{
				{Role: "user", Content: "I need help with authentication"},
				{Role: "user", Content: "How do I implement JWT validation?"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.enhanceQuery(query)
	}
}

func BenchmarkJourneymanKeywordExtraction_LargeText(b *testing.B) {
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.6)

	// Large text with many words
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "This is a test sentence with some important keywords like authentication, security, and implementation. "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.extractKeywords(largeText)
	}
}

func TestScribeGraphTraversalStrategy_ErrorHandling(t *testing.T) {
	mockGraph := &MockKnowledgeGraph{}
	strategy := NewGraphTraversalStrategy(mockGraph, 0.7)

	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	// Test error in FindEntryNodes
	mockGraph.On("FindEntryNodes", mock.Anything, query).Return([]GraphNode{}, assert.AnError)

	docs, err := strategy.Retrieve(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, docs)

	// Reset mock
	mockGraph.ExpectedCalls = nil

	// Test error in TraverseRelated
	entryNodes := []GraphNode{{ID: "node1"}}
	mockGraph.On("FindEntryNodes", mock.Anything, query).Return(entryNodes, nil)
	mockGraph.On("TraverseRelated", mock.Anything, entryNodes, 2).Return([]GraphNode{}, assert.AnError)

	docs, err = strategy.Retrieve(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, docs)
}

func TestCraftVectorSearchStrategy_ContextCancellation(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	strategy := NewVectorSearchStrategy(mockVectorStore, 0.8)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	docs, err := strategy.Retrieve(ctx, query)
	assert.Error(t, err)
	assert.Nil(t, docs)
}