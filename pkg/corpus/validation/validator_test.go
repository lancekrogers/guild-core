// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package validation

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/extraction"
	"github.com/lancekrogers/guild/pkg/corpus/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftKnowledgeValidator tests the creation of a new knowledge validator
func TestCraftKnowledgeValidator(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.rules)
	assert.NotNil(t, validator.conflictDetector)
	assert.NotNil(t, validator.factChecker)
}

// TestJourneymanValidation tests the core validation functionality
func TestJourneymanValidation(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	tests := []struct {
		name      string
		knowledge extraction.ExtractedKnowledge
		expectValid bool
		expectIssues int
	}{
		{
			name: "valid knowledge",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "valid-1",
				Type:    extraction.KnowledgeDecision,
				Content: "Use PostgreSQL for data persistence because of ACID compliance",
				Confidence: 0.8,
				Timestamp:  time.Now(),
				Source: extraction.Source{
					Type: "documentation",
					Timestamp: time.Now(),
				},
			},
			expectValid: true,
			expectIssues: 0,
		},
		{
			name: "low confidence knowledge",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "low-conf-1",
				Type:    extraction.KnowledgeDecision,
				Content: "Maybe use some database",
				Confidence: 0.2, // Below minimum threshold
				Timestamp:  time.Now(),
			},
			expectValid: false,
			expectIssues: 1,
		},
		{
			name: "empty content",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "empty-1",
				Type:    extraction.KnowledgeDecision,
				Content: "", // Empty content
				Confidence: 0.8,
				Timestamp:  time.Now(),
			},
			expectValid: false,
			expectIssues: 1,
		},
		{
			name: "absolute technical claim",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "absolute-1",
				Type:    extraction.KnowledgeSolution,
				Content: "This solution always works and never fails", // Absolute claims
				Confidence: 0.8,
				Timestamp:  time.Now(),
			},
			expectValid: false,
			expectIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(ctx, tt.knowledge)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Len(t, result.Issues, tt.expectIssues)
			assert.Greater(t, result.Confidence, 0.0)
			assert.False(t, result.ValidatedAt.IsZero())
		})
	}
}

// TestScribeConflictDetection tests conflict detection functionality
func TestScribeConflictDetection(t *testing.T) {
	ctx := context.Background()
	
	// Create knowledge graph with existing knowledge
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	// Add existing knowledge
	existing := extraction.ExtractedKnowledge{
		ID:      "existing-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Use React for frontend development",
		Confidence: 0.9,
		Timestamp:  time.Now().Add(-24 * time.Hour),
	}
	err = knowledgeGraph.AddKnowledge(ctx, existing)
	require.NoError(t, err)
	
	// Create validator with the knowledge graph
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	// Test conflicting knowledge
	conflicting := extraction.ExtractedKnowledge{
		ID:      "conflicting-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Avoid React for frontend development", // Conflicts with existing
		Confidence: 0.8,
		Timestamp:  time.Now(),
	}

	result, err := validator.Validate(ctx, conflicting)
	require.NoError(t, err)
	
	// Should detect conflict
	hasConflictIssue := false
	for _, issue := range result.Issues {
		if issue.Type == "conflict" {
			hasConflictIssue = true
			break
		}
	}
	assert.True(t, hasConflictIssue, "Should detect semantic conflict")
}

// TestGuildFactChecking tests fact checking functionality  
func TestGuildFactChecking(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	tests := []struct {
		name           string
		knowledge      extraction.ExtractedKnowledge
		expectVerified bool
	}{
		{
			name: "reasonable technical claim",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "reasonable-1",
				Type:    extraction.KnowledgeSolution,
				Content: "Use Redis for caching to improve response times",
				Confidence: 0.8,
			},
			expectVerified: true,
		},
		{
			name: "unrealistic claim",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "unrealistic-1", 
				Type:    extraction.KnowledgeSolution,
				Content: "This solution is 100% secure and has zero overhead",
				Confidence: 0.8,
			},
			expectVerified: false,
		},
		{
			name: "code with syntax errors",
			knowledge: extraction.ExtractedKnowledge{
				ID:      "syntax-error-1",
				Type:    extraction.KnowledgeSolution,
				Content: "```go\nfunc( {\n    return\n}\n```",
				Confidence: 0.8,
			},
			expectVerified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(ctx, tt.knowledge)
			require.NoError(t, err)
			
			// Check if fact checking passed
			factCheckPassed := true
			for _, issue := range result.Issues {
				if issue.Type == "fact_check" && issue.Severity == "error" {
					factCheckPassed = false
					break
				}
			}
			
			assert.Equal(t, tt.expectVerified, factCheckPassed)
		})
	}
}

// TestCraftValidationRules tests individual validation rules
func TestCraftValidationRules(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		rule      ValidationRule
		knowledge extraction.ExtractedKnowledge
		expectValid bool
	}{
		{
			name: "completeness rule",
			rule: &CompletenessRule{},
			knowledge: extraction.ExtractedKnowledge{
				Content: "Use PostgreSQL for persistence",
				Type:    extraction.KnowledgeDecision,
			},
			expectValid: true,
		},
		{
			name: "confidence threshold rule",
			rule: &ConfidenceThresholdRule{threshold: 0.5},
			knowledge: extraction.ExtractedKnowledge{
				Content:    "Test knowledge",
				Type:       extraction.KnowledgeDecision,
				Confidence: 0.3, // Below threshold
			},
			expectValid: false,
		},
		{
			name: "relevance rule",
			rule: &RelevanceRule{},
			knowledge: extraction.ExtractedKnowledge{
				Content: "Test knowledge",
				Type:    extraction.KnowledgeDecision,
				Source: extraction.Source{
					Type: "documentation",
				},
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.Validate(ctx, tt.knowledge)
			assert.Equal(t, tt.expectValid, result.Valid)
		})
	}
}

// TestJourneymanBatchValidation tests batch validation functionality
func TestJourneymanBatchValidation(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	// Create batch of knowledge items
	knowledgeItems := []extraction.ExtractedKnowledge{
		{ID: "batch-1", Type: extraction.KnowledgeDecision, Content: "Valid decision 1", Confidence: 0.8},
		{ID: "batch-2", Type: extraction.KnowledgeSolution, Content: "Valid solution 1", Confidence: 0.9},
		{ID: "batch-3", Type: extraction.KnowledgePreference, Content: "", Confidence: 0.7}, // Invalid
		{ID: "batch-4", Type: extraction.KnowledgePattern, Content: "Valid pattern 1", Confidence: 0.6},
	}

	results, err := validator.ValidateBatch(ctx, knowledgeItems)
	require.NoError(t, err)
	
	assert.Len(t, results, len(knowledgeItems))
	
	// Check that invalid item was caught
	assert.False(t, results[2].Valid) // Empty content should be invalid
	assert.Greater(t, len(results[2].Issues), 0)
}

// TestScribeValidationCaching tests validation result caching
func TestScribeValidationCaching(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	knowledge := extraction.ExtractedKnowledge{
		ID:      "cache-test-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Test caching functionality",
		Confidence: 0.8,
	}

	// First validation
	start := time.Now()
	result1, err := validator.Validate(ctx, knowledge)
	require.NoError(t, err)
	duration1 := time.Since(start)

	// Second validation (should be cached)
	start = time.Now()
	result2, err := validator.Validate(ctx, knowledge)
	require.NoError(t, err)
	duration2 := time.Since(start)

	// Results should be the same
	assert.Equal(t, result1.Valid, result2.Valid)
	assert.Equal(t, result1.Confidence, result2.Confidence)
	
	// Second validation should be faster (cached)
	assert.Less(t, duration2, duration1)
}

// TestGuildContextCancellation tests context cancellation handling
func TestGuildContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(context.Background())
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(context.Background(), knowledgeGraph)
	require.NoError(t, err)
	
	knowledge := extraction.ExtractedKnowledge{
		ID:      "cancel-test",
		Type:    extraction.KnowledgeDecision,
		Content: "Test knowledge",
	}
	
	// Should handle cancelled context gracefully
	result, err := validator.Validate(ctx, knowledge)
	assert.Error(t, err)
	assert.Equal(t, ValidationResult{}, result)
}

// TestCraftValidationStatistics tests validation statistics tracking
func TestCraftValidationStatistics(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	// Validate multiple items
	knowledgeItems := []extraction.ExtractedKnowledge{
		{ID: "stats-1", Type: extraction.KnowledgeDecision, Content: "Valid 1", Confidence: 0.8},
		{ID: "stats-2", Type: extraction.KnowledgeSolution, Content: "Valid 2", Confidence: 0.9},
		{ID: "stats-3", Type: extraction.KnowledgePreference, Content: "", Confidence: 0.7}, // Invalid
	}

	for _, k := range knowledgeItems {
		_, err := validator.Validate(ctx, k)
		require.NoError(t, err)
	}

	stats, err := validator.GetValidationStats(ctx)
	require.NoError(t, err)
	
	assert.Equal(t, 3, stats.TotalValidated)
	assert.Equal(t, 2, stats.PassedValidation)
	assert.Equal(t, 1, stats.FailedValidation)
	assert.Greater(t, stats.AverageConfidence, 0.0)
	assert.Greater(t, len(stats.CommonIssues), 0)
}

// TestJourneymanQualityMetrics tests quality metrics calculation
func TestJourneymanQualityMetrics(t *testing.T) {
	ctx := context.Background()
	
	// Create a knowledge graph first
	knowledgeGraph, err := graph.NewKnowledgeGraph(ctx)
	require.NoError(t, err)
	
	validator, err := NewKnowledgeValidator(ctx, knowledgeGraph)
	require.NoError(t, err)

	knowledge := extraction.ExtractedKnowledge{
		ID:      "quality-test-1",
		Type:    extraction.KnowledgeDecision,
		Content: "Use PostgreSQL for persistent data storage due to ACID compliance and reliability",
		Confidence: 0.9,
		Timestamp:  time.Now(),
		Source: extraction.Source{
			Type: "documentation",
			Timestamp: time.Now(),
		},
	}

	result, err := validator.Validate(ctx, knowledge)
	require.NoError(t, err)

	// Should have good validation result
	assert.True(t, result.Valid)
	assert.Greater(t, result.Confidence, 0.7)
	assert.Empty(t, result.Issues) // No validation issues for high-quality knowledge
}