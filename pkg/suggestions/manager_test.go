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

// MockProvider implements SuggestionProvider for testing
type MockProvider struct {
	name        string
	suggestions []Suggestion
	err         error
}

func (m *MockProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.suggestions, nil
}

func (m *MockProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	return nil
}

func (m *MockProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeCommand, SuggestionTypeTemplate}
}

func (m *MockProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "Mock provider for testing",
	}
}

func TestDefaultSuggestionManager_RegisterProvider(t *testing.T) {
	manager := NewSuggestionManager()
	
	// Test registering a valid provider
	provider := &MockProvider{name: "TestProvider"}
	err := manager.RegisterProvider(provider)
	assert.NoError(t, err)
	assert.Len(t, manager.providers, 1)
	
	// Test registering duplicate provider
	err = manager.RegisterProvider(provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	
	// Test registering nil provider
	err = manager.RegisterProvider(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestDefaultSuggestionManager_GetSuggestions(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Register mock providers
	provider1 := &MockProvider{
		name: "Provider1",
		suggestions: []Suggestion{
			{
				Type:        SuggestionTypeCommand,
				Content:     "test command",
				Confidence:  0.8,
				Priority:    5,
			},
		},
	}
	
	provider2 := &MockProvider{
		name: "Provider2",
		suggestions: []Suggestion{
			{
				Type:        SuggestionTypeTemplate,
				Content:     "test template",
				Confidence:  0.9,
				Priority:    7,
			},
		},
	}
	
	require.NoError(t, manager.RegisterProvider(provider1))
	require.NoError(t, manager.RegisterProvider(provider2))
	
	// Get suggestions without filter
	context := SuggestionContext{
		CurrentMessage: "test",
	}
	
	suggestions, err := manager.GetSuggestions(ctx, context, nil)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 2)
	
	// Check that suggestions are sorted by priority and confidence
	assert.Equal(t, "test template", suggestions[0].Content) // Higher priority
	assert.Equal(t, "test command", suggestions[1].Content)
	
	// Verify sources are set
	assert.Equal(t, "Provider2", suggestions[0].Source)
	assert.Equal(t, "Provider1", suggestions[1].Source)
}

func TestDefaultSuggestionManager_GetSuggestionsWithFilter(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Register provider with various suggestions
	provider := &MockProvider{
		name: "TestProvider",
		suggestions: []Suggestion{
			{
				Type:        SuggestionTypeCommand,
				Content:     "command1",
				Confidence:  0.8,
				Priority:    5,
				Tags:        []string{"dev"},
			},
			{
				Type:        SuggestionTypeTemplate,
				Content:     "template1",
				Confidence:  0.4,
				Priority:    3,
				Tags:        []string{"docs"},
			},
			{
				Type:        SuggestionTypeCommand,
				Content:     "command2",
				Confidence:  0.9,
				Priority:    7,
				Tags:        []string{"dev", "test"},
			},
		},
	}
	
	require.NoError(t, manager.RegisterProvider(provider))
	
	context := SuggestionContext{
		CurrentMessage: "test",
	}
	
	// Test type filter
	filter := &SuggestionFilter{
		Types: []SuggestionType{SuggestionTypeCommand},
	}
	
	suggestions, err := manager.GetSuggestions(ctx, context, filter)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 2)
	assert.Equal(t, SuggestionTypeCommand, suggestions[0].Type)
	assert.Equal(t, SuggestionTypeCommand, suggestions[1].Type)
	
	// Test confidence filter
	filter = &SuggestionFilter{
		MinConfidence: 0.5,
	}
	
	suggestions, err = manager.GetSuggestions(ctx, context, filter)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 2) // Only high confidence suggestions
	
	// Test tag filter
	filter = &SuggestionFilter{
		Tags: []string{"test"},
	}
	
	suggestions, err = manager.GetSuggestions(ctx, context, filter)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 1)
	assert.Equal(t, "command2", suggestions[0].Content)
	
	// Test max results
	filter = &SuggestionFilter{
		MaxResults: 1,
	}
	
	suggestions, err = manager.GetSuggestions(ctx, context, filter)
	assert.NoError(t, err)
	assert.Len(t, suggestions, 1)
}

func TestDefaultSuggestionManager_RecordUsage(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Create and cache a suggestion
	suggestion := Suggestion{
		ID:      "test-id",
		Type:    SuggestionTypeCommand,
		Content: "test command",
	}
	manager.suggestions[suggestion.ID] = suggestion
	
	// Record acceptance
	err := manager.RecordUsage(ctx, suggestion.ID, true)
	assert.NoError(t, err)
	assert.Len(t, manager.history, 1)
	assert.True(t, manager.history[0].Accepted)
	
	// Record rejection
	err = manager.RecordUsage(ctx, suggestion.ID, false)
	assert.NoError(t, err)
	assert.Len(t, manager.history, 2)
	assert.False(t, manager.history[1].Accepted)
	
	// Try to record usage for non-existent suggestion
	err = manager.RecordUsage(ctx, "non-existent", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultSuggestionManager_GetHistory(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Add some history
	now := time.Now()
	manager.history = []SuggestionHistory{
		{
			ID:           "h1",
			SuggestionID: "s1",
			Type:         SuggestionTypeCommand,
			Content:      "command1",
			Accepted:     true,
			UsedAt:       now.Add(-2 * time.Hour),
			Context:      SuggestionContext{SessionID: "session1"},
		},
		{
			ID:           "h2",
			SuggestionID: "s2",
			Type:         SuggestionTypeTemplate,
			Content:      "template1",
			Accepted:     false,
			UsedAt:       now.Add(-1 * time.Hour),
			Context:      SuggestionContext{SessionID: "session2"},
		},
		{
			ID:           "h3",
			SuggestionID: "s3",
			Type:         SuggestionTypeCommand,
			Content:      "command2",
			Accepted:     true,
			UsedAt:       now,
			Context:      SuggestionContext{SessionID: "session1"},
		},
	}
	
	// Get all history
	history, err := manager.GetHistory(ctx, "", 0)
	assert.NoError(t, err)
	assert.Len(t, history, 3)
	assert.Equal(t, "h3", history[0].ID) // Most recent first
	
	// Get history for specific session
	history, err = manager.GetHistory(ctx, "session1", 0)
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	
	// Get history with limit
	history, err = manager.GetHistory(ctx, "", 1)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "h3", history[0].ID)
}

func TestDefaultSuggestionManager_ProvideFeedback(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Add history item
	manager.history = []SuggestionHistory{
		{
			ID:           "h1",
			SuggestionID: "s1",
			Type:         SuggestionTypeCommand,
			Content:      "test command",
			Accepted:     true,
			UsedAt:       time.Now(),
		},
	}
	
	// Provide feedback
	feedback := UserFeedback{
		Helpful: true,
		Rating:  5,
		Comment: "Very helpful",
	}
	
	err := manager.ProvideFeedback(ctx, "h1", feedback)
	assert.NoError(t, err)
	assert.NotNil(t, manager.history[0].UserFeedback)
	assert.Equal(t, 5, manager.history[0].UserFeedback.Rating)
	assert.Equal(t, "Very helpful", manager.history[0].UserFeedback.Comment)
	
	// Try to provide feedback for non-existent history
	err = manager.ProvideFeedback(ctx, "non-existent", feedback)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultSuggestionManager_GetAnalytics(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	// Add history with various data
	manager.history = []SuggestionHistory{
		{
			ID:           "h1",
			SuggestionID: "s1",
			Type:         SuggestionTypeCommand,
			Content:      "test",
			Accepted:     true,
			UserFeedback: &UserFeedback{Rating: 5},
		},
		{
			ID:           "h2",
			SuggestionID: "s2",
			Type:         SuggestionTypeCommand,
			Content:      "test",
			Accepted:     true,
			UserFeedback: &UserFeedback{Rating: 4},
		},
		{
			ID:           "h3",
			SuggestionID: "s3",
			Type:         SuggestionTypeTemplate,
			Content:      "template",
			Accepted:     false,
		},
		{
			ID:           "h4",
			SuggestionID: "s4",
			Type:         SuggestionTypeCommand,
			Content:      "other",
			Accepted:     true,
		},
	}
	
	// Cache suggestions for provider breakdown
	manager.suggestions["s1"] = Suggestion{Source: "Provider1"}
	manager.suggestions["s2"] = Suggestion{Source: "Provider1"}
	manager.suggestions["s3"] = Suggestion{Source: "Provider2"}
	manager.suggestions["s4"] = Suggestion{Source: "Provider1"}
	
	analytics, err := manager.GetAnalytics(ctx)
	assert.NoError(t, err)
	
	// Check basic metrics
	assert.Equal(t, int64(4), analytics.TotalSuggestions)
	assert.Equal(t, int64(3), analytics.AcceptedSuggestions)
	assert.Equal(t, 0.75, analytics.AcceptanceRate)
	
	// Check type breakdown
	assert.Equal(t, int64(3), analytics.TypeBreakdown[SuggestionTypeCommand])
	assert.Equal(t, int64(1), analytics.TypeBreakdown[SuggestionTypeTemplate])
	
	// Check provider breakdown
	assert.Equal(t, int64(3), analytics.ProviderBreakdown["Provider1"])
	assert.Equal(t, int64(1), analytics.ProviderBreakdown["Provider2"])
	
	// Check user satisfaction
	assert.Equal(t, 4.5, analytics.UserSatisfaction) // (5+4)/2
	
	// Check top suggestions
	assert.NotEmpty(t, analytics.TopSuggestions)
	// Find the "test" pattern which should have 2 uses
	var testPattern *SuggestionStat
	for i := range analytics.TopSuggestions {
		// Pattern format is "Type:Content"
		if analytics.TopSuggestions[i].Pattern == "command:test" {
			testPattern = &analytics.TopSuggestions[i]
			break
		}
	}
	require.NotNil(t, testPattern)
	assert.Equal(t, int64(2), testPattern.UsageCount)
}

func TestDefaultSuggestionManager_EmptyProviders(t *testing.T) {
	manager := NewSuggestionManager()
	ctx := context.Background()
	
	context := SuggestionContext{
		CurrentMessage: "test",
	}
	
	// Should return empty list without error
	suggestions, err := manager.GetSuggestions(ctx, context, nil)
	assert.NoError(t, err)
	assert.Empty(t, suggestions)
}