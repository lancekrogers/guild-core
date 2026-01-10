// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTemplateManager implements templates.TemplateManager for testing
type MockTemplateManager struct {
	templates         []*templates.Template
	searchResults     []*templates.TemplateSearchResult
	contextualResults []*templates.Template
	err               error
}

func (m *MockTemplateManager) Create(ctx context.Context, template *templates.Template) error {
	return m.err
}

func (m *MockTemplateManager) Get(ctx context.Context, id string) (*templates.Template, error) {
	for _, t := range m.templates {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, m.err
}

func (m *MockTemplateManager) GetByName(ctx context.Context, name string) (*templates.Template, error) {
	for _, t := range m.templates {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, m.err
}

func (m *MockTemplateManager) List(ctx context.Context, filter *templates.TemplateFilter) ([]*templates.Template, error) {
	return m.templates, m.err
}

func (m *MockTemplateManager) Update(ctx context.Context, template *templates.Template) error {
	return m.err
}

func (m *MockTemplateManager) Delete(ctx context.Context, id string) error {
	return m.err
}

func (m *MockTemplateManager) Search(ctx context.Context, query string) ([]*templates.Template, error) {
	return m.templates, m.err
}

func (m *MockTemplateManager) GetByCategory(ctx context.Context, category string) ([]*templates.Template, error) {
	var result []*templates.Template
	for _, t := range m.templates {
		if t.Category == category {
			result = append(result, t)
		}
	}
	return result, m.err
}

func (m *MockTemplateManager) GetMostUsed(ctx context.Context, limit int) ([]*templates.Template, error) {
	return m.templates, m.err
}

func (m *MockTemplateManager) GetVariables(ctx context.Context, templateID string) ([]*templates.TemplateVariable, error) {
	return []*templates.TemplateVariable{}, m.err
}

func (m *MockTemplateManager) SetVariables(ctx context.Context, templateID string, variables []*templates.TemplateVariable) error {
	return m.err
}

func (m *MockTemplateManager) Render(ctx context.Context, templateID string, variables map[string]interface{}) (string, error) {
	return "", m.err
}

func (m *MockTemplateManager) RenderContent(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	return content, m.err
}

func (m *MockTemplateManager) RecordUsage(ctx context.Context, templateID string, campaignID *string, variables map[string]interface{}, context string) error {
	return m.err
}

func (m *MockTemplateManager) GetUsageStats(ctx context.Context, templateID string) (*templates.UsageStats, error) {
	return nil, m.err
}

func (m *MockTemplateManager) ListCategories(ctx context.Context) ([]*templates.TemplateCategory, error) {
	return []*templates.TemplateCategory{}, m.err
}

func (m *MockTemplateManager) CreateCategory(ctx context.Context, category *templates.TemplateCategory) error {
	return m.err
}

func (m *MockTemplateManager) Export(ctx context.Context, templateIDs []string) ([]byte, error) {
	return []byte{}, m.err
}

func (m *MockTemplateManager) Import(ctx context.Context, data []byte, overwrite bool) (*templates.ImportResult, error) {
	return nil, m.err
}

func (m *MockTemplateManager) InstallBuiltInTemplates(ctx context.Context) error {
	return m.err
}

func (m *MockTemplateManager) GetBuiltInTemplates() []*templates.Template {
	var result []*templates.Template
	for _, t := range m.templates {
		if t.IsBuiltIn {
			result = append(result, t)
		}
	}
	return result
}

func (m *MockTemplateManager) GetContextualSuggestions(context map[string]interface{}) ([]*templates.Template, error) {
	if m.contextualResults != nil {
		return m.contextualResults, nil
	}
	return m.templates, m.err
}

func (m *MockTemplateManager) RenderTemplate(templateID string, variables map[string]interface{}) (string, error) {
	return "", m.err
}

func (m *MockTemplateManager) SearchTemplates(query string, limit int) ([]*templates.TemplateSearchResult, error) {
	return m.searchResults, m.err
}

func TestTemplateSuggestionProvider_GetSuggestions(t *testing.T) {
	mockManager := &MockTemplateManager{
		templates: []*templates.Template{
			{
				ID:          "test-review",
				Name:        "Code Review",
				Description: "Template for code review",
				Category:    "code review",
				UseCount:    15,
			},
			{
				ID:          "test-debug",
				Name:        "Debug Helper",
				Description: "Template for debugging issues",
				Category:    "debugging",
				UseCount:    5,
			},
			{
				ID:          "test-docs",
				Name:        "Documentation",
				Description: "Template for writing documentation",
				Category:    "documentation",
				UseCount:    20,
			},
		},
	}

	provider := NewTemplateSuggestionProvider(mockManager)
	ctx := context.Background()

	tests := []struct {
		name          string
		context       SuggestionContext
		expectedCount int
		expectedFirst string
		minConfidence float64
	}{
		{
			name: "code review context",
			context: SuggestionContext{
				CurrentMessage: "can you review this code?",
			},
			expectedCount: 1, // Only one matches with high enough confidence
			expectedFirst: "Code Review",
			minConfidence: 0.2,
		},
		{
			name: "debugging context",
			context: SuggestionContext{
				CurrentMessage: "help me debug this error",
			},
			expectedCount: 0, // No contextual suggestions returned, confidence too low
			expectedFirst: "",
			minConfidence: 0.2,
		},
		{
			name: "documentation context",
			context: SuggestionContext{
				CurrentMessage: "write some docs",
			},
			expectedCount: 0, // No contextual suggestions returned, confidence too low
			expectedFirst: "",
			minConfidence: 0.2,
		},
		{
			name: "empty context",
			context: SuggestionContext{
				CurrentMessage: "",
			},
			expectedCount: 0, // No suggestions without search query
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := provider.GetSuggestions(ctx, tt.context)
			require.NoError(t, err)

			assert.Len(t, suggestions, tt.expectedCount)

			if tt.expectedCount > 0 && tt.expectedFirst != "" {
				// Find the expected suggestion (order may vary due to priority/confidence)
				found := false
				for _, s := range suggestions {
					if s.Content == tt.expectedFirst {
						found = true
						assert.Equal(t, SuggestionTypeTemplate, s.Type)
						assert.GreaterOrEqual(t, s.Confidence, tt.minConfidence)
						assert.Equal(t, ActionTypeTemplate, s.Action.Type)
						assert.NotEmpty(t, s.Action.Target)
						assert.NotEmpty(t, s.Tags)
						break
					}
				}
				assert.True(t, found, "Expected suggestion %s not found", tt.expectedFirst)
			}
		})
	}
}

func TestTemplateSuggestionProvider_SearchResults(t *testing.T) {
	mockManager := &MockTemplateManager{
		templates: []*templates.Template{},
		searchResults: []*templates.TemplateSearchResult{
			{
				Template: &templates.Template{
					ID:          "search-1",
					Name:        "Search Result 1",
					Description: "A search result template",
					Category:    "general",
				},
				Relevance: 0.8,
				Matches:   []string{"search", "result"},
			},
			{
				Template: &templates.Template{
					ID:          "search-2",
					Name:        "Search Result 2",
					Description: "Another search result",
					Category:    "general",
				},
				Relevance: 0.2, // Below threshold
				Matches:   []string{"search"},
			},
		},
	}

	provider := NewTemplateSuggestionProvider(mockManager)
	ctx := context.Background()

	context := SuggestionContext{
		CurrentMessage: "search for something",
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should only include search result above threshold
	assert.Len(t, suggestions, 1)
	assert.Equal(t, "Search Result 1", suggestions[0].Content)
	assert.Contains(t, suggestions[0].Display, "🔍")
}

func TestTemplateSuggestionProvider_ProjectContext(t *testing.T) {
	mockManager := &MockTemplateManager{
		templates: []*templates.Template{
			{
				ID:          "test-go",
				Name:        "Go Test Template",
				Description: "Template for Go testing",
				Category:    "go",
				UseCount:    10,
			},
			{
				ID:          "test-python",
				Name:        "Python Test Template",
				Description: "Template for Python testing",
				Category:    "python",
				UseCount:    10,
			},
		},
		// Return the Go template as a contextual suggestion
		contextualResults: []*templates.Template{
			{
				ID:          "test-go",
				Name:        "Go Test Template",
				Description: "Template for Go testing",
				Category:    "go",
				UseCount:    10,
			},
		},
		// Also add search results for the "test" query
		searchResults: []*templates.TemplateSearchResult{
			{
				Template: &templates.Template{
					ID:          "test-go",
					Name:        "Go Test Template",
					Description: "Template for Go testing",
					Category:    "go",
					UseCount:    10,
				},
				Relevance: 0.8,
				Matches:   []string{"test"},
			},
		},
	}

	provider := NewTemplateSuggestionProvider(mockManager)
	ctx := context.Background()

	// Test with Go project context
	context := SuggestionContext{
		CurrentMessage: "test",
		ProjectContext: ProjectContext{
			ProjectType: "go",
			Language:    "go",
		},
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should have at least one suggestion
	assert.NotEmpty(t, suggestions)

	// Go template should have higher priority/confidence
	found := false
	for _, s := range suggestions {
		if s.Content == "Go Test Template" {
			found = true
			assert.GreaterOrEqual(t, s.Priority, 3)
			break
		}
	}
	assert.True(t, found)
}

func TestTemplateSuggestionProvider_NilManager(t *testing.T) {
	provider := NewTemplateSuggestionProvider(nil)
	ctx := context.Background()

	context := SuggestionContext{
		CurrentMessage: "test",
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)
	assert.Empty(t, suggestions)
}

func TestTemplateSuggestionProvider_Metadata(t *testing.T) {
	provider := NewTemplateSuggestionProvider(nil)

	metadata := provider.GetMetadata()
	assert.Equal(t, "TemplateSuggestionProvider", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Capabilities, "template_search")
	assert.Contains(t, metadata.Capabilities, "context_matching")
	assert.Contains(t, metadata.Capabilities, "usage_tracking")
}
