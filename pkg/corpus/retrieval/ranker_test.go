// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftResultRanker_Creation(t *testing.T) {
	ranker := NewResultRanker()

	assert.NotNil(t, ranker)
	assert.Len(t, ranker.scorers, 4) // Default scorers: Recency, Relevance, Contextual, Authority
	assert.Len(t, ranker.weights, 4)

	// Check default weights sum to 1.0
	totalWeight := 0.0
	for _, weight := range ranker.weights {
		totalWeight += weight
	}
	assert.InDelta(t, 1.0, totalWeight, 0.01) // Allow small floating point errors
}

func TestGuildResultRanker_AddScorer(t *testing.T) {
	ranker := &ResultRanker{
		scorers: make([]Scorer, 0),
		weights: make(map[string]float64),
	}

	// Test adding valid scorer
	scorer := &RecencyScorer{}
	err := ranker.AddScorer(scorer, 0.5)
	assert.NoError(t, err)
	assert.Len(t, ranker.scorers, 1)
	assert.Equal(t, 0.5, ranker.weights["recency"])

	// Test adding nil scorer
	err = ranker.AddScorer(nil, 0.5)
	assert.Error(t, err)

	// Test invalid weights
	err = ranker.AddScorer(&RecencyScorer{}, -0.1)
	assert.Error(t, err)

	err = ranker.AddScorer(&RecencyScorer{}, 1.1)
	assert.Error(t, err)
}

func TestJourneymanResultRanker_Ranking(t *testing.T) {
	ranker := NewResultRanker()

	// Create test documents with different characteristics
	now := time.Now()
	oldTime := now.Add(-7 * 24 * time.Hour) // 1 week ago

	docs := []Document{
		{
			ID:      "recent_doc",
			Content: "Recent document about golang programming with authentication",
			Score:   0.8,
			Metadata: map[string]interface{}{
				"last_modified": now,
				"title":         "Recent Go Guide",
				"tags":          []string{"golang", "programming"},
				"author":        "test-agent",
			},
		},
		{
			ID:      "old_doc",
			Content: "Old document about programming",
			Score:   0.9, // Higher base score but old
			Metadata: map[string]interface{}{
				"last_modified": oldTime,
				"title":         "Old Programming Guide",
				"tags":          []string{"programming"},
				"author":        "other-agent",
			},
		},
	}

	query := Query{
		Text: "golang programming authentication",
		Context: QueryContext{
			AgentID:      "test-agent",
			CurrentFiles: []string{"auth.go"},
			Tags:         []string{"golang"},
		},
	}

	ranked := ranker.Rank(docs, query)

	require.Len(t, ranked, 2)

	// Recent document should rank higher due to recency and contextual relevance
	assert.Equal(t, "recent_doc", ranked[0].ID)
	assert.Equal(t, "old_doc", ranked[1].ID)

	// Check that final scores are calculated
	assert.Greater(t, ranked[0].FinalScore, 0.0)
	assert.Greater(t, ranked[1].FinalScore, 0.0)
	assert.Greater(t, ranked[0].FinalScore, ranked[1].FinalScore)

	// Check score details
	assert.Contains(t, ranked[0].ScoreDetails, "recency")
	assert.Contains(t, ranked[0].ScoreDetails, "relevance")
	assert.Contains(t, ranked[0].ScoreDetails, "contextual")
	assert.Contains(t, ranked[0].ScoreDetails, "authority")
}

func TestCraftRecencyScorer_Scoring(t *testing.T) {
	scorer := &RecencyScorer{}
	assert.Equal(t, "recency", scorer.Name())

	now := time.Now()

	tests := []struct {
		name     string
		doc      Document
		expected float64
		delta    float64
	}{
		{
			name: "recent document",
			doc: Document{
				Metadata: map[string]interface{}{
					"last_modified": now,
				},
			},
			expected: 1.0,
			delta:    0.01,
		},
		{
			name: "one week old document",
			doc: Document{
				Metadata: map[string]interface{}{
					"last_modified": now.Add(-7 * 24 * time.Hour),
				},
			},
			expected: 0.5, // Approximately half score due to weekly decay
			delta:    0.1,
		},
		{
			name: "no timestamp",
			doc: Document{
				Metadata: map[string]interface{}{},
			},
			expected: 0.5,
			delta:    0.01,
		},
		{
			name: "string timestamp",
			doc: Document{
				Metadata: map[string]interface{}{
					"last_modified": now.Format(time.RFC3339),
				},
			},
			expected: 1.0,
			delta:    0.01,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score := scorer.Score(test.doc, Query{})
			assert.InDelta(t, test.expected, score, test.delta)
		})
	}
}

func TestGuildRelevanceScorer_VectorScore(t *testing.T) {
	scorer := &RelevanceScorer{}
	assert.Equal(t, "relevance", scorer.Name())

	// Test with vector score in metadata
	doc := Document{
		Content: "Test document",
		Metadata: map[string]interface{}{
			"vector_score": 0.85,
		},
	}

	query := Query{Text: "test query"}
	score := scorer.Score(doc, query)
	assert.Equal(t, 0.85, score)
}

func TestJourneymanRelevanceScorer_KeywordOverlap(t *testing.T) {
	scorer := &RelevanceScorer{}

	doc := Document{
		Content:  "This document is about golang programming and testing frameworks",
		Metadata: map[string]interface{}{}, // No vector score
	}

	query := Query{Text: "golang testing"}
	score := scorer.Score(doc, query)

	// Should get perfect overlap (2/2 keywords match)
	assert.Equal(t, 1.0, score)

	// Test partial overlap
	query = Query{Text: "golang authentication"}
	score = scorer.Score(doc, query)

	// Should get 0.5 (1/2 keywords match)
	assert.Equal(t, 0.5, score)
}

func TestCraftContextualScorer_Scoring(t *testing.T) {
	scorer := &ContextualScorer{}
	assert.Equal(t, "contextual", scorer.Name())

	doc := Document{
		Content: "Document about auth.go implementation",
		Metadata: map[string]interface{}{
			"tags":   []string{"golang", "authentication"},
			"author": "test-agent",
			"type":   "golang-guide",
		},
	}

	query := Query{
		Context: QueryContext{
			AgentID:      "test-agent",
			CurrentFiles: []string{"auth.go", "main.go"},
			Tags:         []string{"golang", "security"},
		},
	}

	score := scorer.Score(doc, query)

	// Should get points for:
	// - File mention (auth.go): 0.2
	// - Tag match (golang): 0.15
	// - Same author: 0.1
	// - Task type match: 0.1
	// Total: 0.55
	assert.InDelta(t, 0.55, score, 0.01)
}

func TestGuildContextualScorer_TagConversion(t *testing.T) {
	scorer := &ContextualScorer{}

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string slice",
			input:    []string{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "interface slice",
			input:    []interface{}{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "single string",
			input:    "single-tag",
			expected: []string{"single-tag"},
		},
		{
			name:     "invalid type",
			input:    123,
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := scorer.convertToStringSlice(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestJourneymanContextualScorer_TaskTypeInference(t *testing.T) {
	scorer := &ContextualScorer{}

	tests := []struct {
		files    []string
		expected string
	}{
		{
			files:    []string{"main.go", "auth.go"},
			expected: "golang",
		},
		{
			files:    []string{"index.js", "package.json"},
			expected: "javascript",
		},
		{
			files:    []string{"main.py", "requirements.txt"},
			expected: "python",
		},
		{
			files:    []string{"README.md", "docs.md"},
			expected: "documentation",
		},
		{
			files:    []string{"test_auth.py", "main.py"},
			expected: "testing",
		},
		{
			files:    []string{"config.yaml"},
			expected: "",
		},
	}

	for _, test := range tests {
		result := scorer.inferTaskType(test.files)
		assert.Equal(t, test.expected, result)
	}
}

type MockCitationGraph struct {
	authorities map[string]float64
}

func (mcg *MockCitationGraph) GetAuthority(documentID string) float64 {
	if mcg.authorities == nil {
		return 0.5
	}
	if auth, exists := mcg.authorities[documentID]; exists {
		return auth
	}
	return 0.5
}

func TestCraftAuthorityScorer_WithCitationGraph(t *testing.T) {
	mockGraph := &MockCitationGraph{
		authorities: map[string]float64{
			"doc1": 0.9,
			"doc2": 0.3,
		},
	}

	scorer := NewAuthorityScorer(mockGraph)
	assert.Equal(t, "authority", scorer.Name())

	doc1 := Document{ID: "doc1"}
	doc2 := Document{ID: "doc2"}
	doc3 := Document{ID: "doc3"} // Not in map

	assert.Equal(t, 0.9, scorer.Score(doc1, Query{}))
	assert.Equal(t, 0.3, scorer.Score(doc2, Query{}))
	assert.Equal(t, 0.5, scorer.Score(doc3, Query{})) // Default
}

func TestGuildAuthorityScorer_WithoutCitationGraph(t *testing.T) {
	scorer := NewAuthorityScorer(nil)

	// Test with view count
	doc := Document{
		Metadata: map[string]interface{}{
			"view_count": 75.0,
		},
	}

	score := scorer.Score(doc, Query{})
	assert.Equal(t, 0.75, score) // 75/100 = 0.75

	// Test with high view count (should cap at 1.0)
	doc.Metadata["view_count"] = 150.0
	score = scorer.Score(doc, Query{})
	assert.Equal(t, 1.0, score)

	// Test with no metadata
	doc = Document{Metadata: map[string]interface{}{}}
	score = scorer.Score(doc, Query{})
	assert.Equal(t, 0.5, score) // Default neutral score
}

// Benchmark tests

func BenchmarkCraftRanking_SmallDataset(b *testing.B) {
	ranker := NewResultRanker()

	docs := make([]Document, 10)
	for i := 0; i < 10; i++ {
		docs[i] = Document{
			ID:      "doc" + string(rune(i)),
			Content: "Test document number " + string(rune(i)),
			Score:   0.8,
			Metadata: map[string]interface{}{
				"last_modified": time.Now(),
				"tags":          []string{"test"},
			},
		}
	}

	query := Query{
		Text: "test query",
		Context: QueryContext{
			Tags: []string{"test"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ranker.Rank(docs, query)
	}
}

func BenchmarkJourneymanRanking_LargeDataset(b *testing.B) {
	ranker := NewResultRanker()

	docs := make([]Document, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = Document{
			ID:      "doc" + string(rune(i)),
			Content: "Test document number " + string(rune(i)),
			Score:   0.8,
			Metadata: map[string]interface{}{
				"last_modified": time.Now(),
				"tags":          []string{"test"},
			},
		}
	}

	query := Query{
		Text: "test query",
		Context: QueryContext{
			Tags: []string{"test"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ranker.Rank(docs, query)
	}
}
