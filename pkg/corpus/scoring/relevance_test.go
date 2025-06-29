// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scoring

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/retrieval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

type MockTokenizer struct {
	mock.Mock
}

func (m *MockTokenizer) Tokenize(text string) []string {
	args := m.Called(text)
	return args.Get(0).([]string)
}

func (m *MockTokenizer) CountTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

type MockDomainModel struct {
	mock.Mock
}

func (m *MockDomainModel) Score(ctx context.Context, doc retrieval.Document, context Context) float64 {
	args := m.Called(ctx, doc, context)
	return args.Get(0).(float64)
}

func (m *MockDomainModel) Domain() string {
	args := m.Called()
	return args.String(0)
}

func TestCraftRelevanceEngine_Creation(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	mockTokenizer := &MockTokenizer{}

	engine := NewRelevanceEngine(mockEmbedder, mockTokenizer)

	assert.NotNil(t, engine)
	assert.Equal(t, mockEmbedder, engine.embedder)
	assert.Equal(t, mockTokenizer, engine.tokenizer)
	assert.NotNil(t, engine.domainModels)
}

func TestGuildRelevanceEngine_DomainModelRegistration(t *testing.T) {
	engine := NewRelevanceEngine(nil, nil)
	mockDomain := &MockDomainModel{}

	mockDomain.On("Domain").Return("test-domain")

	err := engine.RegisterDomainModel(mockDomain)
	assert.NoError(t, err)
	assert.Equal(t, mockDomain, engine.domainModels["test-domain"])

	// Test nil domain model
	err = engine.RegisterDomainModel(nil)
	assert.Error(t, err)
}

func TestJourneymanRelevanceEngine_ComprehensiveScoring(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	mockTokenizer := &MockTokenizer{}
	mockDomain := &MockDomainModel{}

	engine := NewRelevanceEngine(mockEmbedder, mockTokenizer)
	
	// Register domain model
	mockDomain.On("Domain").Return("golang")
	engine.RegisterDomainModel(mockDomain)

	// Test document
	now := time.Now()
	doc := retrieval.Document{
		ID:      "test-doc",
		Content: "This is a golang testing framework for authentication systems",
		Metadata: map[string]interface{}{
			"last_modified":   now,
			"citation_count":  5.0,
			"embedding":       []float32{0.1, 0.2, 0.3, 0.4},
			"title":          "Go Testing Guide",
		},
	}

	scoreContext := Context{
		Query:         "golang testing authentication",
		Domain:        "golang",
		RequiresFresh: false,
		CurrentFiles:  []string{"test_auth.go"},
		Tags:          []string{"golang", "testing"},
	}

	// Set up mocks
	queryEmbedding := []float32{0.1, 0.2, 0.3, 0.4}
	mockEmbedder.On("Embed", mock.Anything, "golang testing authentication").Return(queryEmbedding, nil)

	mockTokenizer.On("Tokenize", "this is a golang testing framework for authentication systems").
		Return([]string{"golang", "testing", "framework", "authentication", "systems"})
	mockTokenizer.On("Tokenize", "golang testing authentication").
		Return([]string{"golang", "testing", "authentication"})

	mockDomain.On("Score", mock.Anything, doc, scoreContext).Return(0.8)

	// Calculate relevance
	score, err := engine.CalculateRelevance(context.Background(), doc, scoreContext)

	require.NoError(t, err)
	assert.Greater(t, score.Overall, 0.0)
	assert.Less(t, score.Overall, 1.0)

	// Check all components are present
	assert.Contains(t, score.Components, "semantic")
	assert.Contains(t, score.Components, "keyword")
	assert.Contains(t, score.Components, "temporal")
	assert.Contains(t, score.Components, "domain")
	assert.Contains(t, score.Components, "citation")

	// Check semantic component (perfect match)
	assert.Equal(t, 1.0, score.Components["semantic"])

	// Check domain component
	assert.Equal(t, 0.8, score.Components["domain"])

	// Check reasoning is provided
	assert.NotEmpty(t, score.Reasoning)
	assert.Contains(t, score.Reasoning, "Semantic similarity")
	assert.Contains(t, score.Reasoning, "Domain relevance")

	mockEmbedder.AssertExpectations(t)
	mockTokenizer.AssertExpectations(t)
	mockDomain.AssertExpectations(t)
}

func TestCraftRelevanceEngine_SemanticSimilarity(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	engine := NewRelevanceEngine(mockEmbedder, nil)

	// Test document with cached embedding
	doc := retrieval.Document{
		Content: "Test document content",
		Metadata: map[string]interface{}{
			"embedding": []float32{1.0, 0.0, 0.0},
		},
	}

	scoreContext := Context{
		Query: "test query",
	}

	// Query embedding
	queryEmbedding := []float32{1.0, 0.0, 0.0} // Perfect match
	mockEmbedder.On("Embed", mock.Anything, "test query").Return(queryEmbedding, nil)

	similarity, err := engine.semanticSimilarity(context.Background(), doc, scoreContext)

	require.NoError(t, err)
	assert.Equal(t, 1.0, similarity) // Perfect cosine similarity

	mockEmbedder.AssertExpectations(t)
}

func TestGuildRelevanceEngine_SemanticSimilarityComputation(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	engine := NewRelevanceEngine(mockEmbedder, nil)

	// Test document without cached embedding
	doc := retrieval.Document{
		Content:  "Test document content",
		Metadata: map[string]interface{}{},
	}

	scoreContext := Context{
		Query: "test query",
	}

	// Document embedding
	docEmbedding := []float32{0.7, 0.7, 0.0}
	mockEmbedder.On("Embed", mock.Anything, "Test document content").Return(docEmbedding, nil)

	// Query embedding
	queryEmbedding := []float32{1.0, 0.0, 0.0}
	mockEmbedder.On("Embed", mock.Anything, "test query").Return(queryEmbedding, nil)

	similarity, err := engine.semanticSimilarity(context.Background(), doc, scoreContext)

	require.NoError(t, err)
	assert.Greater(t, similarity, 0.0)
	assert.Less(t, similarity, 1.0)

	mockEmbedder.AssertExpectations(t)
}

func TestJourneymanRelevanceEngine_KeywordRelevance(t *testing.T) {
	mockTokenizer := &MockTokenizer{}
	engine := NewRelevanceEngine(nil, mockTokenizer)

	doc := retrieval.Document{
		Content: "golang programming testing framework",
	}

	scoreContext := Context{
		Query: "golang testing",
	}

	// Set up tokenizer mocks
	mockTokenizer.On("Tokenize", "golang programming testing framework").
		Return([]string{"golang", "programming", "testing", "framework"})
	mockTokenizer.On("Tokenize", "golang testing").
		Return([]string{"golang", "testing"})

	relevance := engine.keywordRelevance(doc, scoreContext)

	// Perfect overlap: 2/2 query terms found
	assert.Equal(t, 1.0, relevance)

	mockTokenizer.AssertExpectations(t)
}

func TestCraftRelevanceEngine_KeywordRelevancePartial(t *testing.T) {
	mockTokenizer := &MockTokenizer{}
	engine := NewRelevanceEngine(nil, mockTokenizer)

	doc := retrieval.Document{
		Content: "golang programming framework",
	}

	scoreContext := Context{
		Query: "golang testing authentication",
	}

	// Set up tokenizer mocks
	mockTokenizer.On("Tokenize", "golang programming framework").
		Return([]string{"golang", "programming", "framework"})
	mockTokenizer.On("Tokenize", "golang testing authentication").
		Return([]string{"golang", "testing", "authentication"})

	relevance := engine.keywordRelevance(doc, scoreContext)

	// Partial overlap: 1/3 query terms found
	assert.InDelta(t, 0.333, relevance, 0.001)

	mockTokenizer.AssertExpectations(t)
}

func TestGuildRelevanceEngine_TemporalRelevance(t *testing.T) {
	engine := NewRelevanceEngine(nil, nil)

	now := time.Now()

	tests := []struct {
		name         string
		doc          retrieval.Document
		scoreContext Context
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name: "recent document",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{
					"last_modified": now,
				},
			},
			scoreContext: Context{
				RequiresFresh: false,
			},
			expectedMin: 0.95,
			expectedMax: 1.0,
		},
		{
			name: "old document, fresh required",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{
					"last_modified": now.Add(-7 * 24 * time.Hour), // 1 week old
				},
			},
			scoreContext: Context{
				RequiresFresh: true, // Daily decay
			},
			expectedMin: 0.0,
			expectedMax: 0.1,
		},
		{
			name: "no timestamp",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{},
			},
			scoreContext: Context{},
			expectedMin: 0.5,
			expectedMax: 0.5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			relevance := engine.temporalRelevance(test.doc, test.scoreContext)
			assert.GreaterOrEqual(t, relevance, test.expectedMin)
			assert.LessOrEqual(t, relevance, test.expectedMax)
		})
	}
}

func TestJourneymanRelevanceEngine_CitationRelevance(t *testing.T) {
	engine := NewRelevanceEngine(nil, nil)

	tests := []struct {
		name     string
		doc      retrieval.Document
		expected float64
	}{
		{
			name: "with citation count",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{
					"citation_count": 10.0,
				},
			},
			expected: 0.521, // log(11)/log(100) ≈ 0.521
		},
		{
			name: "with view count",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{
					"view_count": 25.0,
				},
			},
			expected: 0.5, // 25/50 = 0.5
		},
		{
			name: "with usage frequency",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{
					"usage_frequency": 0.8,
				},
			},
			expected: 0.8,
		},
		{
			name: "no citation data",
			doc: retrieval.Document{
				Metadata: map[string]interface{}{},
			},
			expected: 0.5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			relevance := engine.citationRelevance(test.doc, Context{})
			assert.InDelta(t, test.expected, relevance, 0.01)
		})
	}
}

func TestCraftCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0},
			expected: 0.0,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "zero vectors",
			a:        []float32{0.0, 0.0},
			b:        []float32{0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := cosineSimilarity(test.a, test.b)
			assert.InDelta(t, test.expected, result, 0.001)
		})
	}
}

func TestGuildSimpleTokenizer(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			input:    "golang   testing\tframework",
			expected: []string{"golang", "testing", "framework"},
		},
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "   \t\n  ",
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := tokenizer.Tokenize(test.input)
		assert.Equal(t, test.expected, result)

		count := tokenizer.CountTokens(test.input)
		assert.Equal(t, len(test.expected), count)
	}
}

func TestJourneymanGolangDomainModel(t *testing.T) {
	model := &GolangDomainModel{}
	assert.Equal(t, "golang", model.Domain())

	doc := retrieval.Document{
		Content: "This document covers golang programming patterns, testing, and interface design",
	}

	scoreContext := Context{
		Query:        "golang testing patterns",
		CurrentFiles: []string{"main.go", "auth_test.go"},
	}

	score := model.Score(context.Background(), doc, scoreContext)

	// Should get points for:
	// - Go terms in content: ~0.3 (golang, testing)
	// - Go file in current files: 0.2
	// - Testing-related content when query mentions test: 0.15
	// Total: ~0.65
	assert.Greater(t, score, 0.6)
	assert.LessOrEqual(t, score, 1.0)
}

func TestCraftGolangDomainModel_TestingContext(t *testing.T) {
	model := &GolangDomainModel{}

	doc := retrieval.Document{
		Content: "This covers golang testing frameworks, assert functions and mock patterns",
	}

	scoreContext := Context{
		Query:        "how to test golang code",
		CurrentFiles: []string{"user.go"},
	}

	score := model.Score(context.Background(), doc, scoreContext)

	// Should get extra points for testing-related content
	assert.Greater(t, score, 0.0)
}

// Performance and edge case tests

func BenchmarkCraftRelevanceEngine_FullScoring(b *testing.B) {
	engine := NewRelevanceEngine(nil, &SimpleTokenizer{})

	doc := retrieval.Document{
		ID:      "test-doc",
		Content: "This is a golang programming guide with authentication examples and testing frameworks",
		Metadata: map[string]interface{}{
			"last_modified":  time.Now(),
			"citation_count": 10.0,
		},
	}

	scoreContext := Context{
		Query:        "golang authentication testing",
		Domain:       "golang",
		CurrentFiles: []string{"auth.go", "test.go"},
		Tags:         []string{"golang", "auth"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateRelevance(context.Background(), doc, scoreContext)
	}
}

func BenchmarkGuildCosineSimilarity_LargeVectors(b *testing.B) {
	// Create large vectors (typical embedding size)
	a := make([]float32, 1536)
	bb := make([]float32, 1536)
	
	for i := 0; i < 1536; i++ {
		a[i] = float32(i) / 1536.0
		bb[i] = float32(1536-i) / 1536.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(a, bb)
	}
}

func TestScribeRelevanceEngine_ContextCancellation(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	engine := NewRelevanceEngine(mockEmbedder, nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	doc := retrieval.Document{
		Content: "Test document",
	}

	scoreContext := Context{
		Query: "test query",
	}

	score, err := engine.CalculateRelevance(ctx, doc, scoreContext)
	assert.Error(t, err)
	assert.Equal(t, RelevanceScore{}, score)
}

func TestCraftRelevanceEngine_EmbedderError(t *testing.T) {
	mockEmbedder := &MockEmbedder{}
	engine := NewRelevanceEngine(mockEmbedder, nil)

	doc := retrieval.Document{
		Content:  "Test document",
		Metadata: map[string]interface{}{},
	}

	scoreContext := Context{
		Query: "test query",
	}

	// Mock embedder to return error
	mockEmbedder.On("Embed", mock.Anything, "Test document").Return([]float32{}, assert.AnError)

	similarity, err := engine.semanticSimilarity(context.Background(), doc, scoreContext)
	assert.Error(t, err)
	assert.Equal(t, 0.0, similarity)
}