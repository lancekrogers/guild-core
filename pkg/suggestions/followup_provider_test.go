// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFollowUpSuggestionProvider_GetSuggestions(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()
	ctx := context.Background()

	tests := []struct {
		name           string
		context        SuggestionContext
		expectedCount  int
		expectedTypes  []string
	}{
		{
			name: "after assistant provides solution",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{
					{
						Role:    "assistant",
						Content: "Here's how you can solve this problem...",
					},
				},
			},
			expectedCount: 2, // Should match default patterns
			expectedTypes: []string{"clarification", "alternatives"},
		},
		{
			name: "after user asks question",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{
					{
						Role:    "user",
						Content: "How do I implement this feature?",
					},
				},
			},
			expectedCount: 1,
			expectedTypes: []string{"example"},
		},
		{
			name: "error discussion",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{
					{
						Role:    "user",
						Content: "I'm getting an error when running the code",
					},
				},
			},
			expectedCount: 2, // Pattern match suggestions
			expectedTypes: []string{"debugging", "error"},
		},
		{
			name: "implementation discussion",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{
					{
						Role:    "user",
						Content: "Let's implement a new feature",
					},
				},
			},
			expectedCount: 2, // Pattern match suggestions
			expectedTypes: []string{"requirements", "planning"},
		},
		{
			name: "code provided by assistant",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{
					{
						Role:    "user",
						Content: "Can you write a function for me?",
					},
					{
						Role:    "assistant",
						Content: "Sure! Here's the function:\n```python\ndef hello():\n    print('Hello')\n```",
					},
				},
			},
			expectedCount: 2, // Dynamic suggestions for code
			expectedTypes: []string{"testing", "code", "execution"},
		},
		{
			name: "empty history",
			context: SuggestionContext{
				ConversationHistory: []ChatMessage{},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := provider.GetSuggestions(ctx, tt.context)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(suggestions), tt.expectedCount)

			if tt.expectedCount > 0 {
				// Verify all are follow-up type
				for _, s := range suggestions {
					assert.Equal(t, SuggestionTypeFollowUp, s.Type)
					assert.Equal(t, ActionTypeInsert, s.Action.Type)
					assert.NotEmpty(t, s.Display)
					assert.Contains(t, s.Display, "➡️")
				}

				// Check expected tags
				allTags := []string{}
				for _, s := range suggestions {
					allTags = append(allTags, s.Tags...)
				}
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, tag := range allTags {
						if tag == expectedType {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected tag %s not found", expectedType)
				}
			}
		})
	}
}

func TestFollowUpSuggestionProvider_UserPreferences(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()
	ctx := context.Background()

	// Test with "always" preference
	context := SuggestionContext{
		ConversationHistory: []ChatMessage{
			{
				Role:    "user",
				Content: "How do I do this?",
			},
		},
		UserPreferences: UserPreferences{
			SuggestionFrequency: "always",
		},
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should have boosted confidence
	assert.NotEmpty(t, suggestions)
	for _, s := range suggestions {
		assert.Greater(t, s.Confidence, 0.5)
	}

	// Test with "minimal" preference
	context.UserPreferences.SuggestionFrequency = "minimal"
	suggestions, err = provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should have reduced confidence
	assert.NotEmpty(t, suggestions)
}

func TestFollowUpSuggestionProvider_LongConversation(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()
	ctx := context.Background()

	// Create a long conversation history
	history := make([]ChatMessage, 15)
	for i := 0; i < 15; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history[i] = ChatMessage{
			Role:      role,
			Content:   "Message content",
			Timestamp: time.Now().Add(-time.Duration(15-i) * time.Minute),
		}
	}
	history[14].Content = "How do I fix this error?"

	context := SuggestionContext{
		ConversationHistory: history,
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should have suggestions but with slightly reduced confidence
	assert.NotEmpty(t, suggestions)
}

func TestFollowUpSuggestionProvider_DynamicSuggestions(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()
	ctx := context.Background()

	// Test dynamic code suggestions - need at least 2 messages in history
	context := SuggestionContext{
		ConversationHistory: []ChatMessage{
			{
				Role:    "user",
				Content: "Can you write a function?",
			},
			{
				Role:    "assistant",
				Content: "Here's the implementation:\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			},
		},
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should include testing and execution suggestions
	testFound := false
	runFound := false
	for _, s := range suggestions {
		if s.Content == "Can you help me test this code?" {
			testFound = true
		}
		if s.Content == "How do I run this?" {
			runFound = true
		}
	}
	assert.True(t, testFound, "Testing suggestion not found")
	assert.True(t, runFound, "Run suggestion not found")
}

func TestFollowUpSuggestionProvider_Metadata(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()

	metadata := provider.GetMetadata()
	assert.Equal(t, "FollowUpSuggestionProvider", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Capabilities, "conversation_analysis")
	assert.Contains(t, metadata.Capabilities, "pattern_matching")
	assert.Contains(t, metadata.Capabilities, "dynamic_generation")
}

func TestFollowUpSuggestionProvider_SupportedTypes(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()

	types := provider.SupportedTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, SuggestionTypeFollowUp, types[0])
}

func TestFollowUpSuggestionProvider_UpdateContext(t *testing.T) {
	provider := NewFollowUpSuggestionProvider()
	ctx := context.Background()

	// Should be a no-op for stateless provider
	err := provider.UpdateContext(ctx, SuggestionContext{})
	assert.NoError(t, err)
}