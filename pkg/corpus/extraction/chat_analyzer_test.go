// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/storage/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftChatAnalyzer tests the creation of a new chat analyzer
func TestCraftChatAnalyzer(t *testing.T) {
	ctx := context.Background()

	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.nlp)
	assert.NotNil(t, analyzer.patternMatcher)
	assert.NotNil(t, analyzer.knowledgeBuilder)
}

// TestJourneymanChatAnalysis tests the core analysis functionality
func TestJourneymanChatAnalysis(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)

	tests := []struct {
		name               string
		messages           []db.ChatMessage
		wantKnowledgeTypes []KnowledgeType
	}{
		{
			name: "decision conversation",
			messages: []db.ChatMessage{
				{
					ID:        "1",
					Content:   "Should I use React or Vue for this project?",
					Role:      "user",
					CreatedAt: &[]time.Time{time.Now()}[0],
				},
				{
					ID:        "2",
					Content:   "I recommend using React because it has better ecosystem support and more job opportunities.",
					Role:      "assistant",
					CreatedAt: &[]time.Time{time.Now().Add(time.Minute)}[0],
				},
			},
			wantKnowledgeTypes: []KnowledgeType{KnowledgeDecision},
		},
		{
			name: "solution conversation",
			messages: []db.ChatMessage{
				{
					ID:        "1",
					Content:   "I'm getting a 'connection refused' error when trying to connect to the database.",
					Role:      "user",
					CreatedAt: &[]time.Time{time.Now()}[0],
				},
				{
					ID:        "2",
					Content:   "This error typically occurs when the database service isn't running. Try starting the service with 'systemctl start postgresql'.",
					Role:      "assistant",
					CreatedAt: &[]time.Time{time.Now().Add(time.Minute)}[0],
				},
			},
			wantKnowledgeTypes: []KnowledgeType{KnowledgeSolution},
		},
		{
			name: "preference statement",
			messages: []db.ChatMessage{
				{
					ID:        "1",
					Content:   "I prefer using TypeScript over JavaScript for large projects because of better type safety.",
					Role:      "user",
					CreatedAt: &[]time.Time{time.Now()}[0],
				},
			},
			wantKnowledgeTypes: []KnowledgeType{KnowledgePreference},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			knowledge, err := analyzer.AnalyzeConversation(ctx, tt.messages)
			require.NoError(t, err)

			// Check that we extracted some knowledge
			assert.Greater(t, len(knowledge), 0, "Should extract at least one piece of knowledge")

			// Check that we got the expected knowledge types
			foundTypes := make(map[KnowledgeType]bool)
			for _, k := range knowledge {
				foundTypes[k.Type] = true
			}

			for _, expectedType := range tt.wantKnowledgeTypes {
				assert.True(t, foundTypes[expectedType], "Should find knowledge type %v", expectedType)
			}
		})
	}
}

// TestGuildExchangeGrouping tests the message grouping functionality
func TestGuildExchangeGrouping(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)

	messages := []db.ChatMessage{
		{ID: "1", Content: "First question", Role: "user", CreatedAt: &[]time.Time{time.Now()}[0]},
		{ID: "2", Content: "First answer", Role: "assistant", CreatedAt: &[]time.Time{time.Now().Add(time.Minute)}[0]},
		{ID: "3", Content: "Second question", Role: "user", CreatedAt: &[]time.Time{time.Now().Add(2 * time.Minute)}[0]},
		{ID: "4", Content: "Second answer", Role: "assistant", CreatedAt: &[]time.Time{time.Now().Add(3 * time.Minute)}[0]},
	}

	exchanges, err := analyzer.groupIntoExchanges(ctx, messages)
	require.NoError(t, err)

	// Should have 2 exchanges
	assert.Len(t, exchanges, 2)

	// First exchange should have messages 1 and 2
	assert.Len(t, exchanges[0].Messages, 2)
	assert.Equal(t, "1", exchanges[0].Messages[0].ID)
	assert.Equal(t, "2", exchanges[0].Messages[1].ID)

	// Second exchange should have messages 3 and 4
	assert.Len(t, exchanges[1].Messages, 2)
	assert.Equal(t, "3", exchanges[1].Messages[0].ID)
	assert.Equal(t, "4", exchanges[1].Messages[1].ID)
}

// TestScribePatternDetection tests the pattern detection capabilities
func TestScribePatternDetection(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)

	tests := []struct {
		name        string
		exchange    Exchange
		shouldMatch string
	}{
		{
			name: "decision pattern",
			exchange: Exchange{
				Messages: []db.ChatMessage{
					{Content: "What should I choose?", Role: "user"},
					{Content: "I recommend option A because it's more reliable.", Role: "assistant"},
				},
			},
			shouldMatch: "decision",
		},
		{
			name: "solution pattern",
			exchange: Exchange{
				Messages: []db.ChatMessage{
					{Content: "I have an error with my code", Role: "user"},
					{Content: "Here's the solution: fix the import statement", Role: "assistant"},
				},
			},
			shouldMatch: "solution",
		},
		{
			name: "preference pattern",
			exchange: Exchange{
				Messages: []db.ChatMessage{
					{Content: "I prefer using spaces over tabs for indentation", Role: "user"},
				},
			},
			shouldMatch: "preference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.shouldMatch {
			case "decision":
				assert.True(t, analyzer.isDecisionPoint(ctx, tt.exchange))
			case "solution":
				assert.True(t, analyzer.isSolutionPattern(ctx, tt.exchange))
			case "preference":
				assert.True(t, analyzer.isPreferenceStatement(ctx, tt.exchange))
			}
		})
	}
}

// TestGuildContextCancellation tests context cancellation handling
func TestGuildContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	analyzer, err := NewChatAnalyzer(context.Background())
	require.NoError(t, err)

	// Should handle cancelled context gracefully
	knowledge, err := analyzer.AnalyzeConversation(ctx, []db.ChatMessage{})
	assert.Error(t, err)
	assert.Nil(t, knowledge)
}

// TestJourneymanEmptyMessages tests handling of empty message lists
func TestJourneymanEmptyMessages(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)

	knowledge, err := analyzer.AnalyzeConversation(ctx, []db.ChatMessage{})
	require.NoError(t, err)
	assert.Empty(t, knowledge)
}

// TestCraftConfidenceCalculation tests confidence calculation
func TestCraftConfidenceCalculation(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewChatAnalyzer(ctx)
	require.NoError(t, err)

	// Create an exchange with clear indicators that should increase confidence
	exchange := Exchange{
		Messages: []db.ChatMessage{
			{Content: "How do I fix this error?", Role: "user"},
			{Content: "I'm certain this solution will work perfectly.", Role: "assistant"},
			{Content: "Thank you, that worked!", Role: "user"},
			{Content: "Glad I could help!", Role: "assistant"},
		},
	}

	confidence := analyzer.calculateConfidence(ctx, exchange)

	// Should have high confidence due to:
	// - Multiple messages (>3)
	// - User question
	// - Certainty indicators
	assert.Greater(t, confidence, 0.7)
	assert.LessOrEqual(t, confidence, 0.95) // Should cap at 0.95
}
