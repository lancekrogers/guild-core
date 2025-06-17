// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/guild-ventures/guild-core/tools"
)

// MockTool implements the Tool interface for testing
type MockTool struct {
	name        string
	description string
	category    string
	examples    []string
	schema      map[string]interface{}
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) Category() string    { return m.category }
func (m *MockTool) Examples() []string  { return m.examples }
func (m *MockTool) Schema() map[string]interface{} { return m.schema }
func (m *MockTool) RequiresAuth() bool  { return false }
func (m *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return tools.NewToolResult("mock output", nil, nil, nil), nil
}

func TestToolSuggestionProvider_GetSuggestions(t *testing.T) {
	// Create mock tools
	mockTools := []tools.Tool{
		&MockTool{
			name:        "glob",
			description: "Find files matching glob patterns like *.go or src/**/*.js",
			category:    "file",
			examples:    []string{`{"pattern": "**/*.go"}`, `{"pattern": "src/**/*.js"}`},
		},
		&MockTool{
			name:        "grep",
			description: "Search for text patterns within files",
			category:    "search",
			examples:    []string{`{"pattern": "TODO", "path": "."}`},
		},
		&MockTool{
			name:        "git_log",
			description: "View git commit history and search through commits",
			category:    "git",
			examples:    []string{`{"limit": 10}`, `{"author": "john"}`},
		},
		&MockTool{
			name:        "webfetch",
			description: "Fetch and scrape content from websites",
			category:    "web",
			examples:    []string{`{"url": "https://example.com"}`},
		},
	}

	// Create tool registry and register tools
	registry := tools.NewToolRegistry()
	for _, tool := range mockTools {
		err := registry.RegisterTool(tool)
		require.NoError(t, err)
	}

	provider := NewToolSuggestionProvider(registry)
	ctx := context.Background()

	tests := []struct {
		name           string
		context        SuggestionContext
		expectedTools  []string
		minConfidence  float64
	}{
		{
			name: "file search context",
			context: SuggestionContext{
				CurrentMessage: "find all go files in the project",
			},
			expectedTools: []string{"glob"},
			minConfidence: 0.6,
		},
		{
			name: "text search context",
			context: SuggestionContext{
				CurrentMessage: "search for TODO comments",
			},
			expectedTools: []string{"grep"},
			minConfidence: 0.6,
		},
		{
			name: "git history context",
			context: SuggestionContext{
				CurrentMessage: "show me the recent commits",
			},
			expectedTools: []string{"git_log"},
			minConfidence: 0.5,
		},
		{
			name: "web scraping context",
			context: SuggestionContext{
				CurrentMessage: "fetch content from a website",
			},
			expectedTools: []string{"webfetch"},
			minConfidence: 0.6,
		},
		{
			name: "multiple tool matches",
			context: SuggestionContext{
				CurrentMessage: "search for files and text",
			},
			expectedTools: []string{"glob", "grep"},
			minConfidence: 0.4,
		},
		{
			name: "project context boost",
			context: SuggestionContext{
				CurrentMessage: "analyze the code",
				ProjectContext: ProjectContext{
					ProjectType: "git",
					Language:    "go",
				},
			},
			expectedTools: []string{}, // Any git tool would be boosted
			minConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := provider.GetSuggestions(ctx, tt.context)
			require.NoError(t, err)

			// Check if expected tools are suggested
			suggestedTools := make(map[string]bool)
			for _, s := range suggestions {
				assert.Equal(t, SuggestionTypeTool, s.Type)
				assert.Contains(t, s.Display, "🔧")
				suggestedTools[s.Content] = true
				
				// Check confidence
				if tt.minConfidence > 0 {
					assert.GreaterOrEqual(t, s.Confidence, tt.minConfidence)
				}
			}

			// Verify expected tools are present
			for _, expectedTool := range tt.expectedTools {
				assert.True(t, suggestedTools[expectedTool], 
					"Expected tool %s not found in suggestions", expectedTool)
			}
		})
	}
}

func TestToolSuggestionProvider_CustomKeywords(t *testing.T) {
	registry := tools.NewToolRegistry()
	
	// Register a custom tool
	customTool := &MockTool{
		name:        "my_custom_analyzer",
		description: "Custom code analysis tool",
		category:    "analysis",
		examples:    []string{},
	}
	err := registry.RegisterTool(customTool)
	require.NoError(t, err)

	// Create provider with custom keywords
	provider := NewToolSuggestionProvider(registry).
		WithCustomKeywords("my_custom_analyzer", []string{"analyze", "inspect", "review"}).
		WithMinConfidence(0.5)

	ctx := context.Background()
	context := SuggestionContext{
		CurrentMessage: "can you inspect this code for issues?",
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)

	// Should find our custom tool
	found := false
	for _, s := range suggestions {
		if s.Content == "my_custom_analyzer" {
			found = true
			assert.GreaterOrEqual(t, s.Confidence, 0.5)
			break
		}
	}
	assert.True(t, found, "Custom tool not found with custom keywords")
}

func TestToolSuggestionProvider_ExtractActionWords(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)

	tests := []struct {
		description string
		expected    []string
	}{
		{
			description: "Search for files and parse their content",
			expected:    []string{"search", "parse"},
		},
		{
			description: "Create, read, update and delete records",
			expected:    []string{"create", "read", "delete"},
		},
		{
			description: "Fetch data from API and analyze results",
			expected:    []string{"fetch", "analyze"},
		},
	}

	for _, tt := range tests {
		actions := provider.extractActionWords(tt.description)
		for _, exp := range tt.expected {
			found := false
			for _, action := range actions {
				if action == exp {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected action %s not found", exp)
		}
	}
}

func TestToolSuggestionProvider_ExtractSignificantWords(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)

	text := "The quick brown fox jumps over the lazy dog"
	words := provider.extractSignificantWords(text)

	// Should exclude common words like "the", "over"
	assert.NotContains(t, words, "the")
	assert.NotContains(t, words, "over")
	
	// Should include significant words
	assert.Contains(t, words, "quick")
	assert.Contains(t, words, "brown")
	assert.Contains(t, words, "fox")
	assert.Contains(t, words, "jumps")
	assert.Contains(t, words, "lazy")
	assert.Contains(t, words, "dog")
}

func TestToolSuggestionProvider_MatchesExamples(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)

	tool := &MockTool{
		name:     "test_tool",
		examples: []string{
			`{"pattern": "*.go", "path": "/src"}`,
			`{"query": "SELECT * FROM users"}`,
			`{"url": "https://api.example.com/data"}`,
		},
	}

	tests := []struct {
		message  string
		expected bool
	}{
		{
			message:  "I need files with pattern *.go",
			expected: true, // Matches first example
		},
		{
			message:  "query the users table",
			expected: true, // Matches second example
		},
		{
			message:  "do something completely different",
			expected: false, // No match
		},
	}

	for _, tt := range tests {
		result := provider.matchesExamples(tool, tt.message, SuggestionContext{})
		assert.Equal(t, tt.expected, result, "Message: %s", tt.message)
	}
}

func TestToolSuggestionProvider_Priority(t *testing.T) {
	registry := tools.NewToolRegistry()
	
	// Create tools with different categories
	tools := []tools.Tool{
		&MockTool{name: "search_tool", category: "search"},
		&MockTool{name: "file_tool", category: "file"},
		&MockTool{name: "shell_tool", category: "shell"},
		&MockTool{name: "web_tool", category: "web"},
	}

	for _, tool := range tools {
		err := registry.RegisterTool(tool)
		require.NoError(t, err)
	}

	provider := NewToolSuggestionProvider(registry)

	// Test priority calculation
	for _, tool := range tools {
		priority := provider.calculatePriority(tool, SuggestionContext{})
		
		switch tool.Category() {
		case "search":
			assert.Equal(t, 8, priority) // Search tools have high priority
		case "file":
			assert.Equal(t, 7, priority)
		case "shell":
			assert.Equal(t, 4, priority) // Shell tools have lower priority
		case "web":
			assert.Equal(t, 5, priority)
		}
	}
}

func TestToolSuggestionProvider_Metadata(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)

	metadata := provider.GetMetadata()
	assert.Equal(t, "ToolSuggestionProvider", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Capabilities, "tool_matching")
	assert.Contains(t, metadata.Capabilities, "category_analysis")
}

func TestToolSuggestionProvider_SupportedTypes(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)

	types := provider.SupportedTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, SuggestionTypeTool, types[0])
}

func TestToolSuggestionProvider_NilRegistry(t *testing.T) {
	provider := NewToolSuggestionProvider(nil)
	ctx := context.Background()

	suggestions, err := provider.GetSuggestions(ctx, SuggestionContext{
		CurrentMessage: "find files",
	})
	
	require.NoError(t, err)
	assert.Empty(t, suggestions)
}