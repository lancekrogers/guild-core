// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandSuggestionProvider_GetSuggestions(t *testing.T) {
	provider := NewCommandSuggestionProvider()
	ctx := context.Background()

	tests := []struct {
		name          string
		context       SuggestionContext
		expectedCount int
		expectedFirst string
		minConfidence float64
	}{
		{
			name: "help command match",
			context: SuggestionContext{
				CurrentMessage: "how do i use this?",
			},
			expectedCount: 1,
			expectedFirst: "help",
			minConfidence: 0.7,
		},
		{
			name: "init command match",
			context: SuggestionContext{
				CurrentMessage: "create a new project",
			},
			expectedCount: 1,
			expectedFirst: "init",
			minConfidence: 0.6,
		},
		{
			name: "multiple matches",
			context: SuggestionContext{
				CurrentMessage: "i want to test my code",
			},
			expectedCount: 1,
			expectedFirst: "test",
			minConfidence: 0.5,
		},
		{
			name: "search command match",
			context: SuggestionContext{
				CurrentMessage: "where is the config file?",
			},
			expectedCount: 1,
			expectedFirst: "search",
			minConfidence: 0.6,
		},
		{
			name: "template command match",
			context: SuggestionContext{
				CurrentMessage: "I need a template for this",
			},
			expectedCount: 1,
			expectedFirst: "template",
			minConfidence: 0.5,
		},
		{
			name: "no matches",
			context: SuggestionContext{
				CurrentMessage: "random text without keywords",
			},
			expectedCount: 0,
		},
		{
			name: "conversation history context",
			context: SuggestionContext{
				CurrentMessage: "yes",
				ConversationHistory: []ChatMessage{
					{
						Role:    "user",
						Content: "can you help me debug this?",
					},
				},
			},
			expectedCount: 1,
			expectedFirst: "debug",
			minConfidence: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := provider.GetSuggestions(ctx, tt.context)
			require.NoError(t, err)

			assert.Len(t, suggestions, tt.expectedCount)

			if tt.expectedCount > 0 && len(suggestions) > 0 {
				assert.Equal(t, tt.expectedFirst, suggestions[0].Content)
				assert.Equal(t, SuggestionTypeCommand, suggestions[0].Type)
				assert.GreaterOrEqual(t, suggestions[0].Confidence, tt.minConfidence)
				assert.Equal(t, ActionTypeExecute, suggestions[0].Action.Type)
				assert.Equal(t, tt.expectedFirst, suggestions[0].Action.Target)
				assert.NotEmpty(t, suggestions[0].Tags)
			}
		})
	}
}

func TestCommandSuggestionProvider_ProjectContext(t *testing.T) {
	provider := NewCommandSuggestionProvider()
	ctx := context.Background()

	// Test with Go project context
	context := SuggestionContext{
		CurrentMessage: "build this",
		ProjectContext: ProjectContext{
			Language: "go",
		},
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	assert.NotEmpty(t, suggestions)
	if len(suggestions) > 0 {
		// Build command should have boosted confidence for Go project
		buildSuggestion := suggestions[0]
		assert.Equal(t, "build", buildSuggestion.Content)
		assert.Greater(t, buildSuggestion.Confidence, 0.5)
	}
}

func TestCommandSuggestionProvider_Metadata(t *testing.T) {
	provider := NewCommandSuggestionProvider()

	metadata := provider.GetMetadata()
	assert.Equal(t, "CommandSuggestionProvider", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Capabilities, "context_analysis")
	assert.Contains(t, metadata.Capabilities, "pattern_matching")
	assert.Contains(t, metadata.Capabilities, "keyword_detection")
}

func TestCommandSuggestionProvider_SupportedTypes(t *testing.T) {
	provider := NewCommandSuggestionProvider()

	types := provider.SupportedTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, SuggestionTypeCommand, types[0])
}

func TestCommandSuggestionProvider_UpdateContext(t *testing.T) {
	provider := NewCommandSuggestionProvider()
	ctx := context.Background()

	// Should be a no-op for stateless provider
	err := provider.UpdateContext(ctx, SuggestionContext{})
	assert.NoError(t, err)
}
