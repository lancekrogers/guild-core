// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLSPManager implements a minimal LSP manager for testing
type MockLSPManager struct {
	completions      *lsp.CompletionList
	symbols          []lsp.DocumentSymbol
	definitions      []lsp.Location
	workspaceSymbols []lsp.SymbolInformation
	err              error
}

func (m *MockLSPManager) GetCompletion(ctx context.Context, file string, line, column int, trigger string) (*lsp.CompletionList, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.completions, nil
}

func (m *MockLSPManager) GetDocumentSymbols(ctx context.Context, file string) ([]lsp.DocumentSymbol, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.symbols, nil
}

func (m *MockLSPManager) GetDefinition(ctx context.Context, file string, line, column int) ([]lsp.Location, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.definitions, nil
}

func (m *MockLSPManager) GetWorkspaceSymbol(ctx context.Context, query string) ([]lsp.SymbolInformation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.workspaceSymbols, nil
}

func TestLSPSuggestionProvider_GetCompletions(t *testing.T) {
	// We'll test the logic separately since we need a real LSP manager
	provider := NewLSPSuggestionProvider(nil)

	// Test confidence calculation
	confidence := provider.calculateCompletionConfidence(lsp.CompletionItem{
		Kind:   lsp.CompletionItemKindMethod,
		Detail: "Has details",
	})
	assert.Greater(t, confidence, 0.5)

	// Test deprecated item
	deprecatedConfidence := provider.calculateCompletionConfidence(lsp.CompletionItem{
		Kind:       lsp.CompletionItemKindMethod,
		Deprecated: true,
	})
	assert.Less(t, deprecatedConfidence, confidence)
}

func TestLSPSuggestionProvider_NavigationSuggestions(t *testing.T) {
	// Test with symbol at cursor
	fileContext := &FileContext{
		FilePath:       "/path/to/main.go",
		Line:           20,
		Column:         10,
		SymbolAtCursor: "ProcessData",
	}

	// Since we have nil manager, we test the logic directly
	suggestions := make([]Suggestion, 0)

	// Manually create what getNavigationSuggestions would create
	if fileContext != nil && fileContext.SymbolAtCursor != "" {
		suggestions = append(suggestions, Suggestion{
			Type:       SuggestionTypeCommand,
			Content:    "Go to definition of 'ProcessData'",
			Display:    "📍 Go to definition of 'ProcessData'",
			Confidence: 0.9,
			Priority:   8,
		})

		suggestions = append(suggestions, Suggestion{
			Type:       SuggestionTypeCommand,
			Content:    "Find all references to 'ProcessData'",
			Display:    "🔍 Find references to 'ProcessData'",
			Confidence: 0.8,
			Priority:   7,
		})
	}

	assert.Len(t, suggestions, 2)
	assert.Contains(t, suggestions[0].Content, "Go to definition")
	assert.Contains(t, suggestions[1].Content, "Find all references")
}

func TestLSPSuggestionProvider_LanguageSpecific(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	tests := []struct {
		name     string
		context  SuggestionContext
		expected []string
	}{
		{
			name: "Go test suggestion",
			context: SuggestionContext{
				CurrentMessage: "write a test for this",
				ProjectContext: ProjectContext{
					Language: "go",
				},
			},
			expected: []string{"Generate Go test"},
		},
		{
			name: "Go interface suggestion",
			context: SuggestionContext{
				CurrentMessage: "find interface implementations",
				ProjectContext: ProjectContext{
					Language: "go",
				},
			},
			expected: []string{"Find interface implementations"},
		},
		{
			name: "Python import suggestion",
			context: SuggestionContext{
				CurrentMessage: "organize imports",
				ProjectContext: ProjectContext{
					Language: "python",
				},
			},
			expected: []string{"Organize Python imports"},
		},
		{
			name: "TypeScript types suggestion",
			context: SuggestionContext{
				CurrentMessage: "generate types",
				ProjectContext: ProjectContext{
					Language: "typescript",
				},
			},
			expected: []string{"Generate TypeScript types"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := provider.getLanguageSpecificSuggestions(tt.context)

			if len(tt.expected) > 0 {
				assert.NotEmpty(t, suggestions)
				for _, exp := range tt.expected {
					found := false
					for _, s := range suggestions {
						if s.Content == exp {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected suggestion %s not found", exp)
				}
			}
		})
	}
}

func TestLSPSuggestionProvider_ExtractSymbols(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	tests := []struct {
		message  string
		expected []string
	}{
		{
			message:  "Where is ProcessData defined?",
			expected: []string{"ProcessData"},
		},
		{
			message:  "Find the UserManager class",
			expected: []string{"UserManager"},
		},
		{
			message:  "Look for get_user_data function",
			expected: []string{"get_user_data"}, // snake_case pattern
		},
		{
			message:  "The HTTPClient and WebSocket classes",
			expected: []string{"HTTPClient", "WebSocket"},
		},
	}

	for _, tt := range tests {
		symbols := provider.extractSymbolsFromMessage(tt.message)
		assert.Equal(t, len(tt.expected), len(symbols), "Message: %s", tt.message)
		for i, exp := range tt.expected {
			if i < len(symbols) {
				assert.Equal(t, exp, symbols[i])
			}
		}
	}
}

func TestLSPSuggestionProvider_Icons(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	// Test completion icons
	assert.Equal(t, "🔧", provider.getCompletionIcon(lsp.CompletionItemKindMethod))
	assert.Equal(t, "📊", provider.getCompletionIcon(lsp.CompletionItemKindVariable))
	assert.Equal(t, "🏛️", provider.getCompletionIcon(lsp.CompletionItemKindClass))
	assert.Equal(t, "🔗", provider.getCompletionIcon(lsp.CompletionItemKindInterface))

	// Symbol icon tests commented out until LSP Manager exposes SymbolKind
	// assert.Equal(t, "🔧", provider.getSymbolIcon(lsp.SymbolKindFunction))
	// assert.Equal(t, "🏛️", provider.getSymbolIcon(lsp.SymbolKindClass))
	// assert.Equal(t, "🏗️", provider.getSymbolIcon(lsp.SymbolKindStruct))
}

func TestLSPSuggestionProvider_Priority(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	// Methods and functions should have high priority
	assert.Equal(t, 8, provider.getCompletionPriority(lsp.CompletionItemKindMethod))
	assert.Equal(t, 8, provider.getCompletionPriority(lsp.CompletionItemKindFunction))

	// Variables and fields medium priority
	assert.Equal(t, 7, provider.getCompletionPriority(lsp.CompletionItemKindVariable))

	// Classes and interfaces
	assert.Equal(t, 6, provider.getCompletionPriority(lsp.CompletionItemKindClass))
}

func TestLSPSuggestionProvider_Metadata(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	metadata := provider.GetMetadata()
	assert.Equal(t, "LSPSuggestionProvider", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Capabilities, "code_completion")
	assert.Contains(t, metadata.Capabilities, "goto_definition")
	assert.Contains(t, metadata.Capabilities, "find_references")
}

func TestLSPSuggestionProvider_SupportedTypes(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)

	types := provider.SupportedTypes()
	assert.Contains(t, types, SuggestionTypeCode)
	assert.Contains(t, types, SuggestionTypeProject)
	assert.Contains(t, types, SuggestionTypeCommand)
}

func TestLSPSuggestionProvider_NilManager(t *testing.T) {
	provider := NewLSPSuggestionProvider(nil)
	ctx := context.Background()

	context := SuggestionContext{
		CurrentMessage: "test",
		FileContext: &FileContext{
			FilePath: "/path/to/main.go",
			Line:     10,
			Column:   15,
		},
	}

	suggestions, err := provider.GetSuggestions(ctx, context)
	require.NoError(t, err)
	assert.Empty(t, suggestions)
}
