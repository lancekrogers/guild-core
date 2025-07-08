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

// Mock implementations for testing

type MockVectorStore struct {
	mock.Mock
}

func (m *MockVectorStore) Search(ctx context.Context, query string, limit int) ([]VectorResult, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]VectorResult), args.Error(1)
}

type MockMetadataStore struct {
	mock.Mock
}

func (m *MockMetadataStore) GetDocument(ctx context.Context, id string) (*Document, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*Document), args.Error(1)
}

func (m *MockMetadataStore) ListDocuments(ctx context.Context, filters []Filter) ([]Document, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]Document), args.Error(1)
}

func (m *MockMetadataStore) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	args := m.Called(ctx, id, metadata)
	return args.Error(0)
}

type MockKeywordIndexer struct {
	mock.Mock
}

func (m *MockKeywordIndexer) Search(ctx context.Context, keywords []string, limit int) ([]Document, error) {
	args := m.Called(ctx, keywords, limit)
	return args.Get(0).([]Document), args.Error(1)
}

func (m *MockKeywordIndexer) Index(ctx context.Context, doc Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

type MockKnowledgeGraph struct {
	mock.Mock
}

func (m *MockKnowledgeGraph) FindEntryNodes(ctx context.Context, query Query) ([]GraphNode, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]GraphNode), args.Error(1)
}

func (m *MockKnowledgeGraph) TraverseRelated(ctx context.Context, nodes []GraphNode, hops int) ([]GraphNode, error) {
	args := m.Called(ctx, nodes, hops)
	return args.Get(0).([]GraphNode), args.Error(1)
}

type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, event Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// Test functions

func TestCraftRetriever_Creation(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)

	require.NoError(t, err)
	assert.NotNil(t, retriever)

	// Cast to concrete type to access fields
	impl, ok := retriever.(*RetrieverImpl)
	require.True(t, ok)
	assert.Equal(t, mockVectorStore, impl.vectorStore)
	assert.Equal(t, mockMetadataStore, impl.metadataStore)
	assert.Equal(t, mockRanker, impl.ranker)
}

func TestCraftRetriever_CreationValidation(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	// Test nil vector store
	retriever, err := NewRetriever(context.Background(), nil, mockMetadataStore, mockRanker)
	assert.Error(t, err)
	assert.Nil(t, retriever)

	// Test nil metadata store
	retriever, err = NewRetriever(context.Background(), mockVectorStore, nil, mockRanker)
	assert.Error(t, err)
	assert.Nil(t, retriever)

	// Test nil ranker
	retriever, err = NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, nil)
	assert.Error(t, err)
	assert.Nil(t, retriever)
}

func TestGuildRetriever_StrategyManagement(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	// Test adding strategy
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.5)

	err = retriever.AddStrategy(strategy)
	assert.NoError(t, err)
	assert.Len(t, retriever.GetStrategies(), 1)

	// Test adding nil strategy
	err = retriever.AddStrategy(nil)
	assert.Error(t, err)
}

func TestJourneymanRetriever_Retrieve(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	// Add a mock strategy
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.7)
	retriever.AddStrategy(strategy)

	// Set up mocks
	expectedDocs := []Document{
		{
			ID:      "doc1",
			Content: "This is a test document about Go programming",
			Score:   0.8,
			Metadata: map[string]interface{}{
				"title": "Go Programming Guide",
				"tags":  []string{"golang", "programming"},
			},
		},
	}

	mockIndexer.On("Search", mock.Anything, []string{"test", "golang"}, 10).Return(expectedDocs, nil)

	// Test retrieval
	query := Query{
		Text:       "test golang",
		MaxResults: 10,
		MinScore:   0.1,
		Context: QueryContext{
			AgentID: "test-agent",
			Tags:    []string{"golang"},
		},
	}

	results, err := retriever.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Contains(t, results[0].Content, "Go programming")

	mockIndexer.AssertExpectations(t)
}

func TestJourneymanRetriever_EmptyQuery(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	// Test empty query
	query := Query{
		Text:       "",
		MaxResults: 10,
	}

	results, err := retriever.Retrieve(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestJourneymanRetriever_NoStrategies(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	// Test retrieval with no strategies
	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	results, err := retriever.Retrieve(context.Background(), query)
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestGuildRetriever_EventPublishing(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()
	mockEventBus := &MockEventBus{}

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	retriever.SetEventBus(mockEventBus)

	// Add a mock strategy
	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.7)
	retriever.AddStrategy(strategy)

	// Set up mocks
	mockIndexer.On("Search", mock.Anything, mock.Anything, mock.Anything).Return([]Document{}, nil)
	mockEventBus.On("Publish", mock.Anything, mock.MatchedBy(func(event Event) bool {
		return event.Type == "retrieval.completed"
	})).Return(nil)

	// Test retrieval
	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	_, err = retriever.Retrieve(context.Background(), query)
	assert.NoError(t, err)

	mockEventBus.AssertExpectations(t)
}

func TestCraftRetriever_ContextCancellation(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	results, err := retriever.Retrieve(ctx, query)
	assert.Error(t, err)
	assert.Nil(t, results)
}

// Benchmark tests

func BenchmarkCraftRetrieval_SmallDataset(b *testing.B) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, _ := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)

	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.7)
	retriever.AddStrategy(strategy)

	docs := make([]Document, 10)
	for i := 0; i < 10; i++ {
		docs[i] = Document{
			ID:      "doc" + string(rune(i)),
			Content: "This is test document number " + string(rune(i)),
			Score:   0.8,
		}
	}

	mockIndexer.On("Search", mock.Anything, mock.Anything, mock.Anything).Return(docs, nil)

	query := Query{
		Text:       "test query",
		MaxResults: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retriever.Retrieve(context.Background(), query)
	}
}

func BenchmarkJourneymanRetrieval_LargeDataset(b *testing.B) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, _ := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)

	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.7)
	retriever.AddStrategy(strategy)

	docs := make([]Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = Document{
			ID:      "doc" + string(rune(i)),
			Content: "This is test document number " + string(rune(i)),
			Score:   0.8,
		}
	}

	mockIndexer.On("Search", mock.Anything, mock.Anything, mock.Anything).Return(docs, nil)

	query := Query{
		Text:       "test query",
		MaxResults: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retriever.Retrieve(context.Background(), query)
	}
}

// Edge case tests

func TestJourneymanRetriever_ScoreFiltering(t *testing.T) {
	mockVectorStore := &MockVectorStore{}
	mockMetadataStore := &MockMetadataStore{}
	mockRanker := NewResultRanker()

	retriever, err := NewRetriever(context.Background(), mockVectorStore, mockMetadataStore, mockRanker)
	require.NoError(t, err)

	mockIndexer := &MockKeywordIndexer{}
	strategy := NewKeywordSearchStrategy(mockIndexer, 0.7)
	retriever.AddStrategy(strategy)

	// Documents with varying scores
	docs := []Document{
		{ID: "doc1", Content: "High score doc", Score: 0.9},
		{ID: "doc2", Content: "Medium score doc", Score: 0.5},
		{ID: "doc3", Content: "Low score doc", Score: 0.1},
	}

	mockIndexer.On("Search", mock.Anything, mock.Anything, mock.Anything).Return(docs, nil)

	// Test with minimum score filter
	query := Query{
		Text:       "test query",
		MaxResults: 10,
		MinScore:   0.4, // Should filter out doc3
	}

	results, err := retriever.Retrieve(context.Background(), query)
	require.NoError(t, err)

	// Should get 2 results (doc1 and doc2 have scores >= 0.4)
	assert.Len(t, results, 2)
}
