// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package graph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/corpus/extraction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftKnowledgeGraph tests the creation of a new knowledge graph
func TestCraftKnowledgeGraph(t *testing.T) {
	ctx := context.Background()

	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.NotNil(t, graph.nodes)
	assert.NotNil(t, graph.edges)
	assert.NotNil(t, graph.index)
}

// TestJourneymanAddKnowledge tests adding knowledge to the graph
func TestJourneymanAddKnowledge(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	knowledge := extraction.ExtractedKnowledge{
		ID:      "test-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Use React for frontend development because of its large ecosystem",
		Source: extraction.Source{
			Type:      "test",
			Timestamp: time.Now(),
		},
		Entities: []extraction.Entity{
			{Name: "React", Type: "technology", Confidence: 0.9},
			{Name: "frontend", Type: "domain", Confidence: 0.8},
		},
		Relations: []extraction.Relation{
			{Subject: "React", Predicate: "used_for", Object: "frontend", Confidence: 0.9},
		},
		Confidence: 0.85,
		Timestamp:  time.Now(),
	}

	err = graph.AddKnowledge(ctx, knowledge)
	require.NoError(t, err)

	// Verify the knowledge was added by querying for it
	query := GraphQuery{
		Text:      "frontend architecture decision",
		NodeTypes: []NodeType{NodeDecision},
		MaxDepth:  0, // Only return start nodes, don't traverse edges
	}
	results, err := graph.Query(ctx, query)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	if len(results) > 0 {
		assert.Equal(t, "test-1", results[0].ID)
		assert.Equal(t, NodeDecision, results[0].Type)
		assert.Equal(t, knowledge.Content, results[0].Content)
	}
}

// TestGuildQueryKnowledge tests querying knowledge from the graph
func TestGuildQueryKnowledge(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	// Add test knowledge
	knowledgeItems := []extraction.ExtractedKnowledge{
		{
			ID:      "decision-1",
			Type:    extraction.KnowledgeDecision,
			Content: "Use PostgreSQL for data persistence",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			ID:      "solution-1",
			Type:    extraction.KnowledgeSolution,
			Content: "Fix database connection issues by increasing timeout",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
		{
			ID:      "preference-1",
			Type:    extraction.KnowledgePreference,
			Content: "Prefer TypeScript over JavaScript for type safety",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Confidence: 0.7,
			Timestamp:  time.Now(),
		},
	}

	for _, k := range knowledgeItems {
		err = graph.AddKnowledge(ctx, k)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		query       GraphQuery
		expectCount int
	}{
		{
			name: "text search",
			query: GraphQuery{
				Text:  "database",
				Limit: 10,
			},
			expectCount: 1, // Should find connection fix (contains "database")
		},
		{
			name: "type filter",
			query: GraphQuery{
				NodeTypes: []NodeType{NodeDecision},
				Limit:     10,
			},
			expectCount: 1, // Should find only decision
		},
		{
			name: "confidence filter",
			query: GraphQuery{
				MinConfidence: 0.85,
				Limit:         10,
			},
			expectCount: 1, // Should find only high-confidence items
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := graph.Query(ctx, tt.query)
			require.NoError(t, err)
			assert.Len(t, results, tt.expectCount)
		})
	}
}

// TestScribeRelationships tests relationship handling in the graph
func TestScribeRelationships(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	// Add knowledge with entities and relations
	knowledge := extraction.ExtractedKnowledge{
		ID:      "rel-test-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Use Redis for caching to improve performance",
		Source: extraction.Source{
			Type:      "test",
			Timestamp: time.Now(),
		},
		Entities: []extraction.Entity{
			{Name: "Redis", Type: "technology", Confidence: 0.9},
			{Name: "caching", Type: "technique", Confidence: 0.8},
			{Name: "performance", Type: "quality", Confidence: 0.9},
		},
		Relations: []extraction.Relation{
			{Subject: "Redis", Predicate: "improves", Object: "performance", Confidence: 0.9},
			{Subject: "caching", Predicate: "technique_for", Object: "performance", Confidence: 0.8},
		},
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err = graph.AddKnowledge(ctx, knowledge)
	require.NoError(t, err)

	// Verify relationships were created by querying
	query := GraphQuery{
		Text:      "Python machine learning",
		NodeTypes: []NodeType{NodeEntity},
		MaxDepth:  2,
	}
	results, err := graph.Query(ctx, query)
	require.NoError(t, err)
	assert.Greater(t, len(results), 0, "Should find entities from the knowledge")
}

// TestCraftGraphTraversal tests graph traversal capabilities
func TestCraftGraphTraversal(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	// Create a small knowledge network
	knowledgeItems := []extraction.ExtractedKnowledge{
		{
			ID:      "node-1",
			Type:    extraction.KnowledgeDecision,
			Content: "Choose React for frontend",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Entities:   []extraction.Entity{{Name: "React", Type: "technology", Confidence: 0.9}},
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			ID:      "node-2",
			Type:    extraction.KnowledgeSolution,
			Content: "Use React hooks for state management",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Entities:   []extraction.Entity{{Name: "React", Type: "technology", Confidence: 0.9}},
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
		{
			ID:      "node-3",
			Type:    extraction.KnowledgePattern,
			Content: "React component patterns for reusability",
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Entities:   []extraction.Entity{{Name: "React", Type: "technology", Confidence: 0.9}},
			Confidence: 0.7,
			Timestamp:  time.Now(),
		},
	}

	for _, k := range knowledgeItems {
		err = graph.AddKnowledge(ctx, k)
		require.NoError(t, err)
	}

	// Test traversal by querying for React-related nodes
	query := GraphQuery{
		Text:      "React",
		NodeTypes: []NodeType{NodeEntity, NodeDecision},
		MaxDepth:  2,
	}
	results, err := graph.Query(ctx, query)
	require.NoError(t, err)

	// Should find multiple React-related nodes
	assert.Greater(t, len(results), 1)
}

// TestJourneymanIndexing tests the search indexing functionality
func TestJourneymanIndexing(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	knowledge := extraction.ExtractedKnowledge{
		ID:      "index-test-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Use microservices architecture for scalability and maintainability",
		Source: extraction.Source{
			Type:      "test",
			Timestamp: time.Now(),
		},
		Confidence: 0.8,
		Timestamp:  time.Now(),
	}

	err = graph.AddKnowledge(ctx, knowledge)
	require.NoError(t, err)

	// Test search functionality
	results, err := graph.index.Search(ctx, "microservices scalability", 5)
	require.NoError(t, err)

	// Should find the indexed knowledge
	assert.Greater(t, len(results), 0)
	assert.Equal(t, "index-test-1", results[0].NodeID)
}

// TestGuildContextCancellation tests context cancellation handling
func TestGuildContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	graph, err := NewKnowledgeGraph(context.Background())
	require.NoError(t, err)

	knowledge := extraction.ExtractedKnowledge{
		ID:      "cancel-test",
		Type:    extraction.KnowledgeDecision,
		Content: "Test knowledge",
		Source: extraction.Source{
			Type:      "test",
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Should handle cancelled context gracefully
	err = graph.AddKnowledge(ctx, knowledge)
	assert.Error(t, err)
}

// TestScribeGraphStatistics tests graph statistics functionality
func TestScribeGraphStatistics(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	// Add various types of knowledge
	knowledgeTypes := []extraction.KnowledgeType{
		extraction.KnowledgeDecision,
		extraction.KnowledgeSolution,
		extraction.KnowledgePreference,
		extraction.KnowledgePattern,
	}

	for i, kType := range knowledgeTypes {
		knowledge := extraction.ExtractedKnowledge{
			ID:      fmt.Sprintf("stats-test-%d", i+1),
			Type:    kType,
			Content: fmt.Sprintf("Test content for %s", kType),
			Source: extraction.Source{
				Type:      "test",
				Timestamp: time.Now(),
			},
			Confidence: 0.8,
			Timestamp:  time.Now(),
		}
		err = graph.AddKnowledge(ctx, knowledge)
		require.NoError(t, err)
	}

	stats, err := graph.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, 4, stats.NodeCount)
	assert.Equal(t, 4, len(stats.NodeTypes))
	// EdgeTypes should be 0 since test knowledge items have no entities/relations
	assert.Equal(t, 0, len(stats.EdgeTypes))
}

// TestCraftQueryBuilder tests the query builder functionality
func TestCraftQueryBuilder(t *testing.T) {
	query := NewQueryBuilder().
		WithText("React components").
		WithNodeTypes(NodeDecision, NodeSolution).
		WithMinConfidence(0.7).
		WithTimeRange(time.Now().Add(-24 * time.Hour)).
		WithLimit(10).
		Build()

	assert.Equal(t, "React components", query.Text)
	assert.Len(t, query.NodeTypes, 2)
	assert.Equal(t, 0.7, query.MinConfidence)
	assert.Equal(t, 10, query.Limit)
	assert.NotNil(t, query.TimeRange)
}

// TestJourneymanConcurrentAccess tests concurrent access to the graph
func TestJourneymanConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	graph, err := NewKnowledgeGraph(ctx)
	require.NoError(t, err)

	// Test concurrent writes
	const numGoroutines = 10
	const itemsPerGoroutine = 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			for j := 0; j < itemsPerGoroutine; j++ {
				knowledge := extraction.ExtractedKnowledge{
					ID:      fmt.Sprintf("concurrent-%d-%d", routineID, j),
					Type:    extraction.KnowledgeDecision,
					Content: fmt.Sprintf("Concurrent test %d-%d", routineID, j),
					Source: extraction.Source{
						Type:      "test",
						Timestamp: time.Now(),
					},
					Confidence: 0.8,
					Timestamp:  time.Now(),
				}
				err := graph.AddKnowledge(ctx, knowledge)
				assert.NoError(t, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all items were added
	stats, err := graph.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines*itemsPerGoroutine, stats.NodeCount)
}
