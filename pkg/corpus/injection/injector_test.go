// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package injection

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/pkg/corpus/retrieval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock retriever for testing
type MockRetriever struct {
	mock.Mock
}

func (m *MockRetriever) Retrieve(ctx context.Context, query retrieval.Query) ([]retrieval.RankedDocument, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]retrieval.RankedDocument), args.Error(1)
}

func (m *MockRetriever) AddStrategy(strategy retrieval.RetrievalStrategy) error {
	args := m.Called(strategy)
	return args.Error(0)
}

func (m *MockRetriever) SetEventBus(eventBus retrieval.EventBus) {
	m.Called(eventBus)
}

func (m *MockRetriever) GetStrategies() []retrieval.RetrievalStrategy {
	args := m.Called()
	return args.Get(0).([]retrieval.RetrievalStrategy)
}

func (m *MockRetriever) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCraftContextInjector_Creation(t *testing.T) {
	mockRetriever := &MockRetriever{}

	injector, err := NewContextInjector(mockRetriever, 4000)

	require.NoError(t, err)
	assert.NotNil(t, injector)
	assert.Equal(t, 4000, injector.maxTokens)
	assert.True(t, injector.cacheEnabled)
}

func TestGuildContextInjector_CreationWithDefaults(t *testing.T) {
	mockRetriever := &MockRetriever{}

	// Test with zero max tokens (should use default)
	injector, err := NewContextInjector(mockRetriever, 0)

	require.NoError(t, err)
	assert.Equal(t, 4000, injector.maxTokens) // Should use default
}

func TestJourneymanContextInjector_BasicInjection(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	// Mock retrieval results
	mockDocs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "This is documentation about Go authentication patterns",
				Metadata: map[string]interface{}{
					"title": "Go Auth Guide",
					"type":  "architecture",
				},
			},
			FinalScore: 0.9,
		},
		{
			Document: retrieval.Document{
				ID:      "doc2",
				Content: "Example: func ValidateToken(token string) error { ... }",
				Metadata: map[string]interface{}{
					"title": "JWT Example",
					"type":  "example",
				},
			},
			FinalScore: 0.8,
		},
	}

	mockRetriever.On("Retrieve", mock.Anything, mock.MatchedBy(func(q retrieval.Query) bool {
		return q.Text == "How do I implement JWT authentication?"
	})).Return(mockDocs, nil)

	// Test injection request
	req := InjectionRequest{
		OriginalPrompt: Prompt{
			System: "You are a helpful coding assistant.",
			User:   "How do I implement JWT authentication?",
		},
		Query: retrieval.Query{
			Text:       "How do I implement JWT authentication?",
			MaxResults: 5,
		},
		InjectionPoints: []InjectionPoint{InjectionSystemPrompt, InjectionUserMessage},
		DisableCache:    true, // Disable cache for testing
	}

	result, err := injector.InjectContext(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that context was injected
	assert.Contains(t, result.SystemPrompt, "Go Auth Guide")
	assert.Contains(t, result.UserMessage, "JWT Example")
	assert.Equal(t, "How do I implement JWT authentication?", result.Original.User)

	// Check metadata
	assert.Equal(t, 2, result.Metadata["documents_used"])
	assert.Contains(t, result.Metadata, "injection_timestamp")

	mockRetriever.AssertExpectations(t)
}

func TestCraftContextInjector_CacheKey(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			System: "System prompt",
			User:   "User message",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionSystemPrompt},
	}

	// Generate cache key
	key1 := injector.generateCacheKey(req)
	assert.NotEmpty(t, key1)

	// Same request should generate same key
	key2 := injector.generateCacheKey(req)
	assert.Equal(t, key1, key2)

	// Different request should generate different key
	req.Query.Text = "different query"
	key3 := injector.generateCacheKey(req)
	assert.NotEqual(t, key1, key3)
}

func TestGuildContextInjector_Caching(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	mockDocs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "Test content",
			},
			FinalScore: 0.9,
		},
	}

	// First call should hit retriever
	mockRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(mockDocs, nil).Once()

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			User: "test query",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionUserMessage},
		CacheKey:        "test-key",
	}

	// First call
	result1, err := injector.InjectContext(context.Background(), req)
	require.NoError(t, err)

	// Second call should use cache (retriever should not be called again)
	result2, err := injector.InjectContext(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, result1.UserMessage, result2.UserMessage)
	mockRetriever.AssertExpectations(t)
}

func TestJourneymanContextInjector_DisabledCache(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	injector.SetCacheEnabled(false)

	mockDocs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "Test content",
			},
			FinalScore: 0.9,
		},
	}

	// Should call retriever both times when cache is disabled
	mockRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(mockDocs, nil).Twice()

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			User: "test query",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionUserMessage},
		CacheKey:        "test-key",
	}

	// Both calls should hit retriever
	_, err = injector.InjectContext(context.Background(), req)
	require.NoError(t, err)

	_, err = injector.InjectContext(context.Background(), req)
	require.NoError(t, err)

	mockRetriever.AssertExpectations(t)
}

func TestCraftContextInjector_InjectionPoints(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	mockDocs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "System context content",
				Metadata: map[string]interface{}{
					"type": "architecture",
				},
			},
			FinalScore: 0.9,
		},
		{
			Document: retrieval.Document{
				ID:      "doc2",
				Content: "API documentation content",
				Metadata: map[string]interface{}{
					"type": "api",
				},
			},
			FinalScore: 0.8,
		},
		{
			Document: retrieval.Document{
				ID:      "doc3",
				Content: "Example code content",
				Metadata: map[string]interface{}{
					"type": "example",
				},
			},
			FinalScore: 0.7,
		},
	}

	mockRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(mockDocs, nil)

	originalPrompt := Prompt{
		System: "Original system prompt",
		User:   "Original user message",
		Tools:  "Original tool context",
	}

	// Test all injection points
	req := InjectionRequest{
		OriginalPrompt: originalPrompt,
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{
			InjectionSystemPrompt,
			InjectionUserMessage,
			InjectionToolContext,
		},
		DisableCache: true,
	}

	result, err := injector.InjectContext(context.Background(), req)
	require.NoError(t, err)

	// System prompt should be enhanced
	assert.NotEqual(t, originalPrompt.System, result.SystemPrompt)
	assert.Contains(t, result.SystemPrompt, "Original system prompt")

	// User message should be enhanced
	assert.NotEqual(t, originalPrompt.User, result.UserMessage)
	assert.Contains(t, result.UserMessage, "Original user message")

	// Tool context should be set
	assert.NotEqual(t, originalPrompt.Tools, result.ToolContext)

	// Check contexts are recorded
	assert.Len(t, result.Contexts, 3)
	assert.Contains(t, result.Contexts, InjectionSystemPrompt)
	assert.Contains(t, result.Contexts, InjectionUserMessage)
	assert.Contains(t, result.Contexts, InjectionToolContext)
}

func TestGuildContextInjector_ContextCancellation(t *testing.T) {
	mockRetriever := &MockRetriever{}
	injector, err := NewContextInjector(mockRetriever, 2000)
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			User: "test query",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionUserMessage},
	}

	result, err := injector.InjectContext(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestScribeContextFormatter_DocumentGrouping(t *testing.T) {
	formatter := NewContextFormatter()

	docs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "Architecture documentation",
				Metadata: map[string]interface{}{
					"type": "architecture",
				},
			},
		},
		{
			Document: retrieval.Document{
				ID:      "doc2",
				Content: "API reference",
				Metadata: map[string]interface{}{
					"type": "api",
				},
			},
		},
		{
			Document: retrieval.Document{
				ID:      "doc3",
				Content: "Code example",
				Metadata: map[string]interface{}{
					"type": "example",
				},
			},
		},
		{
			Document: retrieval.Document{
				ID:      "doc4",
				Content: "Design patterns guide",
			},
		},
	}

	grouped := formatter.groupByType(docs)

	assert.Len(t, grouped["system"], 1)  // Architecture doc
	assert.Len(t, grouped["tool"], 1)    // API doc
	assert.Len(t, grouped["example"], 2) // Example doc + unknown type (default)

	assert.Equal(t, "doc1", grouped["system"][0].ID)
	assert.Equal(t, "doc2", grouped["tool"][0].ID)
}

func TestCraftContextFormatter_TokenEstimation(t *testing.T) {
	formatter := NewContextFormatter()

	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"test", 1},
		{"this is a test", 4},
		{"this is a longer test with more words", 8},
	}

	for _, test := range tests {
		result := formatter.estimateTokens(test.text)
		assert.Equal(t, test.expected, result)
	}
}

func TestGuildContextFormatter_DocumentTitles(t *testing.T) {
	formatter := NewContextFormatter()

	tests := []struct {
		name     string
		doc      retrieval.RankedDocument
		expected string
	}{
		{
			name: "with title",
			doc: retrieval.RankedDocument{
				Document: retrieval.Document{
					ID: "doc1",
					Metadata: map[string]interface{}{
						"title": "Test Document",
					},
				},
			},
			expected: "Test Document",
		},
		{
			name: "with source",
			doc: retrieval.RankedDocument{
				Document: retrieval.Document{
					ID: "doc1",
					Metadata: map[string]interface{}{
						"source": "test-source",
					},
				},
			},
			expected: "test-source",
		},
		{
			name: "fallback to ID",
			doc: retrieval.RankedDocument{
				Document: retrieval.Document{
					ID:       "doc1",
					Metadata: map[string]interface{}{},
				},
			},
			expected: "Document doc1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatter.getDocumentTitle(test.doc)
			assert.Equal(t, test.expected, result)
		})
	}
}

// Performance and integration tests

func BenchmarkCraftContextInjection_SmallContext(b *testing.B) {
	mockRetriever := &MockRetriever{}
	injector, _ := NewContextInjector(mockRetriever, 2000)

	docs := []retrieval.RankedDocument{
		{
			Document: retrieval.Document{
				ID:      "doc1",
				Content: "Small test document",
			},
			FinalScore: 0.9,
		},
	}

	mockRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(docs, nil)

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			User: "test query",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionUserMessage},
		DisableCache:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		injector.InjectContext(context.Background(), req)
	}
}

func BenchmarkJourneymanContextInjection_LargeContext(b *testing.B) {
	mockRetriever := &MockRetriever{}
	injector, _ := NewContextInjector(mockRetriever, 8000)

	// Create large documents
	docs := make([]retrieval.RankedDocument, 20)
	for i := 0; i < 20; i++ {
		content := ""
		for j := 0; j < 100; j++ {
			content += "This is a large document with lots of content to test performance. "
		}

		docs[i] = retrieval.RankedDocument{
			Document: retrieval.Document{
				ID:      "doc" + string(rune(i)),
				Content: content,
			},
			FinalScore: 0.8,
		}
	}

	mockRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(docs, nil)

	req := InjectionRequest{
		OriginalPrompt: Prompt{
			User: "test query",
		},
		Query: retrieval.Query{
			Text: "test query",
		},
		InjectionPoints: []InjectionPoint{InjectionSystemPrompt, InjectionUserMessage},
		DisableCache:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		injector.InjectContext(context.Background(), req)
	}
}
