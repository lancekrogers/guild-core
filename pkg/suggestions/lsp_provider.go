// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/lsp"
)

// LSPSuggestionProvider provides code intelligence suggestions using Language Server Protocol
type LSPSuggestionProvider struct {
	lspManager *lsp.Manager
}

// NewLSPSuggestionProvider creates a new LSP suggestion provider
func NewLSPSuggestionProvider(lspManager *lsp.Manager) *LSPSuggestionProvider {
	return &LSPSuggestionProvider{
		lspManager: lspManager,
	}
}

// GetSuggestions returns LSP-based suggestions based on context
func (p *LSPSuggestionProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if p.lspManager == nil {
		return []Suggestion{}, nil
	}

	suggestions := make([]Suggestion, 0)

	// Check if we have file/position context
	if context.FileContext != nil && context.FileContext.FilePath != "" {
		// Get code completions at current position
		completions, err := p.getCompletionSuggestions(ctx, context)
		if err == nil {
			suggestions = append(suggestions, completions...)
		}

		// Get symbol-based suggestions
		symbols, err := p.getSymbolSuggestions(ctx, context)
		if err == nil {
			suggestions = append(suggestions, symbols...)
		}

		// Get navigation suggestions (go to definition, find references)
		nav, err := p.getNavigationSuggestions(ctx, context)
		if err == nil {
			suggestions = append(suggestions, nav...)
		}
	}

	// Get workspace-wide suggestions based on conversation
	workspace, err := p.getWorkspaceSuggestions(ctx, context)
	if err == nil {
		suggestions = append(suggestions, workspace...)
	}

	return suggestions, nil
}

// getCompletionSuggestions gets code completion suggestions at the current position
func (p *LSPSuggestionProvider) getCompletionSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if context.FileContext == nil {
		return []Suggestion{}, nil
	}

	// Get completions from LSP
	completions, err := p.lspManager.GetCompletion(
		ctx,
		context.FileContext.FilePath,
		context.FileContext.Line,
		context.FileContext.Column,
		context.FileContext.TriggerCharacter,
	)
	if err != nil {
		return nil, err
	}

	suggestions := make([]Suggestion, 0)

	// Convert top completions to suggestions
	maxCompletions := 5
	if len(completions.Items) < maxCompletions {
		maxCompletions = len(completions.Items)
	}

	for i := 0; i < maxCompletions; i++ {
		item := completions.Items[i]

		// Calculate confidence based on sort text and kind
		confidence := p.calculateCompletionConfidence(item)

		// Skip low confidence items
		if confidence < 0.3 {
			continue
		}

		// Determine icon based on completion kind
		icon := p.getCompletionIcon(item.Kind)

		suggestion := Suggestion{
			Type:        SuggestionTypeCode,
			Content:     item.InsertText,
			Display:     fmt.Sprintf("%s %s", icon, item.Label),
			Description: item.Detail,
			Confidence:  confidence,
			Priority:    p.getCompletionPriority(item.Kind),
			Action: SuggestionAction{
				Type:   ActionTypeInsert,
				Target: item.InsertText,
				Parameters: map[string]interface{}{
					"completion_kind": item.Kind,
					"file":            context.FileContext.FilePath,
					"position":        fmt.Sprintf("%d:%d", context.FileContext.Line, context.FileContext.Column),
				},
			},
			Tags: []string{"completion", strings.ToLower(p.getKindName(item.Kind))},
			Metadata: map[string]interface{}{
				"lsp_provider": true,
				"kind":         item.Kind,
				"deprecated":   item.Deprecated,
			},
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// getSymbolSuggestions gets suggestions for symbols in the current file
func (p *LSPSuggestionProvider) getSymbolSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	// TODO: Implement when GetDocumentSymbols is added to LSP Manager
	// For now, return empty suggestions
	return []Suggestion{}, nil
}

// getNavigationSuggestions gets navigation-related suggestions
func (p *LSPSuggestionProvider) getNavigationSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if context.FileContext == nil {
		return []Suggestion{}, nil
	}

	suggestions := make([]Suggestion, 0)

	// Suggest "Go to Definition" if cursor is on a symbol
	if context.FileContext.SymbolAtCursor != "" {
		suggestion := Suggestion{
			Type:        SuggestionTypeCommand,
			Content:     fmt.Sprintf("Go to definition of '%s'", context.FileContext.SymbolAtCursor),
			Display:     fmt.Sprintf("📍 Go to definition of '%s'", context.FileContext.SymbolAtCursor),
			Description: "Navigate to where this symbol is defined",
			Confidence:  0.9,
			Priority:    8,
			Action: SuggestionAction{
				Type:   ActionTypeCommand,
				Target: "lsp_definition",
				Parameters: map[string]interface{}{
					"file":   context.FileContext.FilePath,
					"line":   context.FileContext.Line,
					"column": context.FileContext.Column,
				},
			},
			Tags: []string{"navigation", "definition"},
			Metadata: map[string]interface{}{
				"lsp_provider": true,
			},
		}
		suggestions = append(suggestions, suggestion)

		// Also suggest finding references
		refSuggestion := Suggestion{
			Type:        SuggestionTypeCommand,
			Content:     fmt.Sprintf("Find all references to '%s'", context.FileContext.SymbolAtCursor),
			Display:     fmt.Sprintf("🔍 Find references to '%s'", context.FileContext.SymbolAtCursor),
			Description: "Find all places where this symbol is used",
			Confidence:  0.8,
			Priority:    7,
			Action: SuggestionAction{
				Type:   ActionTypeCommand,
				Target: "lsp_references",
				Parameters: map[string]interface{}{
					"file":   context.FileContext.FilePath,
					"line":   context.FileContext.Line,
					"column": context.FileContext.Column,
				},
			},
			Tags: []string{"navigation", "references"},
			Metadata: map[string]interface{}{
				"lsp_provider": true,
			},
		}
		suggestions = append(suggestions, refSuggestion)
	}

	return suggestions, nil
}

// getWorkspaceSuggestions gets workspace-wide suggestions based on conversation
func (p *LSPSuggestionProvider) getWorkspaceSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if context.CurrentMessage == "" {
		return []Suggestion{}, nil
	}

	suggestions := make([]Suggestion, 0)

	// Extract potential symbol names from the message
	symbols := p.extractSymbolsFromMessage(context.CurrentMessage)

	for _, symbol := range symbols {
		// Suggest workspace symbol search
		suggestion := Suggestion{
			Type:        SuggestionTypeCommand,
			Content:     fmt.Sprintf("Search for '%s' in workspace", symbol),
			Display:     fmt.Sprintf("🔎 Find '%s' in workspace", symbol),
			Description: "Search for this symbol across all files",
			Confidence:  0.6,
			Priority:    5,
			Action: SuggestionAction{
				Type:   ActionTypeCommand,
				Target: "workspace_symbol",
				Parameters: map[string]interface{}{
					"query": symbol,
				},
			},
			Tags: []string{"workspace", "search"},
			Metadata: map[string]interface{}{
				"lsp_provider": true,
			},
		}
		suggestions = append(suggestions, suggestion)
	}

	// Language-specific suggestions
	if context.ProjectContext.Language != "" {
		langSuggestions := p.getLanguageSpecificSuggestions(context)
		suggestions = append(suggestions, langSuggestions...)
	}

	return suggestions, nil
}

// getLanguageSpecificSuggestions provides language-specific code suggestions
func (p *LSPSuggestionProvider) getLanguageSpecificSuggestions(context SuggestionContext) []Suggestion {
	suggestions := make([]Suggestion, 0)

	switch strings.ToLower(context.ProjectContext.Language) {
	case "go":
		if strings.Contains(context.CurrentMessage, "test") {
			suggestions = append(suggestions, Suggestion{
				Type:        SuggestionTypeCommand,
				Content:     "Generate Go test",
				Display:     "🧪 Generate Go test",
				Description: "Create a test function for the current Go function",
				Confidence:  0.7,
				Priority:    6,
				Action: SuggestionAction{
					Type:   ActionTypeTemplate,
					Target: "go-test-generator",
				},
				Tags: []string{"go", "testing"},
			})
		}
		if strings.Contains(context.CurrentMessage, "interface") {
			suggestions = append(suggestions, Suggestion{
				Type:        SuggestionTypeCommand,
				Content:     "Find interface implementations",
				Display:     "🔗 Find interface implementations",
				Description: "Locate all implementations of the current interface",
				Confidence:  0.6,
				Priority:    5,
				Action: SuggestionAction{
					Type:   ActionTypeCommand,
					Target: "find_implementations",
				},
				Tags: []string{"go", "interface"},
			})
		}

	case "python":
		if strings.Contains(context.CurrentMessage, "import") {
			suggestions = append(suggestions, Suggestion{
				Type:        SuggestionTypeCommand,
				Content:     "Organize Python imports",
				Display:     "📦 Organize imports",
				Description: "Sort and organize import statements",
				Confidence:  0.7,
				Priority:    5,
				Action: SuggestionAction{
					Type:   ActionTypeCommand,
					Target: "organize_imports",
				},
				Tags: []string{"python", "imports"},
			})
		}

	case "typescript", "javascript":
		if strings.Contains(context.CurrentMessage, "type") || strings.Contains(context.CurrentMessage, "interface") {
			suggestions = append(suggestions, Suggestion{
				Type:        SuggestionTypeCommand,
				Content:     "Generate TypeScript types",
				Display:     "📝 Generate types",
				Description: "Generate TypeScript type definitions",
				Confidence:  0.6,
				Priority:    5,
				Action: SuggestionAction{
					Type:   ActionTypeCommand,
					Target: "generate_types",
				},
				Tags: []string{"typescript", "types"},
			})
		}
	}

	return suggestions
}

// Helper methods

func (p *LSPSuggestionProvider) calculateCompletionConfidence(item lsp.CompletionItem) float64 {
	confidence := 0.5

	// Boost confidence for certain kinds
	switch item.Kind {
	case lsp.CompletionItemKindMethod, lsp.CompletionItemKindFunction:
		confidence += 0.2
	case lsp.CompletionItemKindVariable, lsp.CompletionItemKindField:
		confidence += 0.15
	case lsp.CompletionItemKindClass, lsp.CompletionItemKindInterface:
		confidence += 0.1
	}

	// Reduce confidence for deprecated items
	if item.Deprecated {
		confidence *= 0.5
	}

	// Boost if it has detailed documentation
	if item.Detail != "" {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (p *LSPSuggestionProvider) getCompletionPriority(kind lsp.CompletionItemKind) int {
	switch kind {
	case lsp.CompletionItemKindMethod, lsp.CompletionItemKindFunction:
		return 8
	case lsp.CompletionItemKindVariable, lsp.CompletionItemKindField:
		return 7
	case lsp.CompletionItemKindClass, lsp.CompletionItemKindInterface:
		return 6
	case lsp.CompletionItemKindKeyword:
		return 5
	default:
		return 4
	}
}

func (p *LSPSuggestionProvider) getCompletionIcon(kind lsp.CompletionItemKind) string {
	switch kind {
	case lsp.CompletionItemKindMethod, lsp.CompletionItemKindFunction:
		return "🔧"
	case lsp.CompletionItemKindVariable, lsp.CompletionItemKindField:
		return "📊"
	case lsp.CompletionItemKindClass:
		return "🏛️"
	case lsp.CompletionItemKindInterface:
		return "🔗"
	case lsp.CompletionItemKindModule:
		return "📦"
	case lsp.CompletionItemKindProperty:
		return "🏷️"
	case lsp.CompletionItemKindKeyword:
		return "🔑"
	case lsp.CompletionItemKindSnippet:
		return "📝"
	default:
		return "💡"
	}
}

// getSymbolIcon returns an icon for a symbol kind
// Note: Currently commented out as LSP Manager doesn't expose SymbolKind
// func (p *LSPSuggestionProvider) getSymbolIcon(kind lsp.SymbolKind) string {
// 	switch kind {
// 	case lsp.SymbolKindFunction, lsp.SymbolKindMethod:
// 		return "🔧"
// 	case lsp.SymbolKindClass:
// 		return "🏛️"
// 	case lsp.SymbolKindInterface:
// 		return "🔗"
// 	case lsp.SymbolKindStruct:
// 		return "🏗️"
// 	case lsp.SymbolKindVariable:
// 		return "📊"
// 	case lsp.SymbolKindConstant:
// 		return "🔒"
// 	case lsp.SymbolKindPackage:
// 		return "📦"
// 	default:
// 		return "📍"
// 	}
// }

func (p *LSPSuggestionProvider) getKindName(kind lsp.CompletionItemKind) string {
	switch kind {
	case lsp.CompletionItemKindMethod:
		return "method"
	case lsp.CompletionItemKindFunction:
		return "function"
	case lsp.CompletionItemKindVariable:
		return "variable"
	case lsp.CompletionItemKindField:
		return "field"
	case lsp.CompletionItemKindClass:
		return "class"
	case lsp.CompletionItemKindInterface:
		return "interface"
	default:
		return "code"
	}
}

// getSymbolKindName returns the name of a symbol kind
// Note: Currently commented out as LSP Manager doesn't expose SymbolKind
// func (p *LSPSuggestionProvider) getSymbolKindName(kind lsp.SymbolKind) string {
// 	switch kind {
// 	case lsp.SymbolKindFunction:
// 		return "function"
// 	case lsp.SymbolKindMethod:
// 		return "method"
// 	case lsp.SymbolKindClass:
// 		return "class"
// 	case lsp.SymbolKindInterface:
// 		return "interface"
// 	case lsp.SymbolKindStruct:
// 		return "struct"
// 	case lsp.SymbolKindVariable:
// 		return "variable"
// 	default:
// 		return "symbol"
// 	}
// }

func (p *LSPSuggestionProvider) extractSymbolsFromMessage(message string) []string {
	// Simple heuristic to extract potential symbol names
	symbols := []string{}
	words := strings.Fields(message)

	// Common English words to skip
	commonWords := map[string]bool{
		"The": true, "Where": true, "Find": true, "Look": true,
		"Get": true, "Set": true, "For": true, "And": true,
		"Or": true, "In": true, "Is": true, "Are": true,
		"Can": true, "How": true, "What": true, "When": true,
		"Show": true, "List": true, "Check": true, "Search": true,
	}

	for _, word := range words {
		// Clean up common punctuation first
		cleaned := strings.Trim(word, ".,!?()[]{}\"'")
		if cleaned == "" || len(cleaned) <= 2 {
			continue
		}

		// Look for snake_case patterns
		if strings.Contains(cleaned, "_") {
			symbols = append(symbols, cleaned)
			continue
		}

		// Look for PascalCase or camelCase patterns
		if cleaned[0] >= 'A' && cleaned[0] <= 'Z' {
			// Check if it has more capitals (like HTTPClient)
			if strings.ContainsAny(cleaned[1:], "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				symbols = append(symbols, cleaned)
			} else if !commonWords[cleaned] {
				// If not a common word, treat as identifier
				symbols = append(symbols, cleaned)
			}
		}
	}

	return symbols
}

// UpdateContext updates the provider's context (no-op for stateless provider)
func (p *LSPSuggestionProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	// This provider is stateless
	return nil
}

// SupportedTypes returns the suggestion types this provider handles
func (p *LSPSuggestionProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeCode, SuggestionTypeProject, SuggestionTypeCommand}
}

// GetMetadata returns provider metadata
func (p *LSPSuggestionProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        "LSPSuggestionProvider",
		Version:     "1.0.0",
		Description: "Provides code intelligence suggestions using Language Server Protocol",
		Capabilities: []string{
			"code_completion",
			"goto_definition",
			"find_references",
			"workspace_symbols",
			"document_symbols",
			"language_specific",
		},
	}
}
