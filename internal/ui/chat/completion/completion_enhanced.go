// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package completion

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
)

// CompletionEngineEnhanced provides a more robust integration with the suggestion system
type CompletionEngineEnhanced struct {
	*CompletionEngine
	directManager suggestions.SuggestionManager
	providers     map[string]suggestions.SuggestionProvider
}

// NewCompletionEngineEnhanced creates an enhanced completion engine with direct suggestion provider integration
func NewCompletionEngineEnhanced(guildConfig *config.GuildConfig, projectRoot string) (*CompletionEngineEnhanced, error) {
	if guildConfig == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "guild config is required", nil).
			WithComponent("chat.completion").
			WithOperation("NewCompletionEngineEnhanced")
	}

	// Create base engine
	baseEngine := NewCompletionEngine(guildConfig, projectRoot)

	// Create enhanced engine
	enhanced := &CompletionEngineEnhanced{
		CompletionEngine: baseEngine,
		providers:        make(map[string]suggestions.SuggestionProvider),
	}

	// Initialize suggestion providers directly
	if err := enhanced.initializeProviders(guildConfig, projectRoot); err != nil {
		// Log but don't fail - we can still use traditional completions
		// In production, you might want to use proper logging here
		_ = err
	}

	return enhanced, nil
}

// initializeProviders sets up all suggestion providers
func (ce *CompletionEngineEnhanced) initializeProviders(guildConfig *config.GuildConfig, projectRoot string) error {
	// Create suggestion manager
	ce.directManager = suggestions.NewSuggestionManager()

	// Initialize command provider
	commandProvider := suggestions.NewCommandSuggestionProvider()

	if err := ce.directManager.RegisterProvider(commandProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register command provider").
			WithComponent("chat.completion").
			WithOperation("initializeProviders")
	}
	ce.providers["command"] = commandProvider

	// Initialize template provider (requires template manager)
	// For now, skip template provider as it requires a template manager
	// templateProvider := suggestions.NewTemplateSuggestionProvider(templateManager)
	// Template provider registration commented out until template manager is available
	// if err := ce.directManager.RegisterProvider(templateProvider); err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register template provider").
	// 		WithComponent("chat.completion").
	// 		WithOperation("initializeProviders")
	// }
	// ce.providers["template"] = templateProvider

	// Initialize follow-up provider
	followUpProvider := suggestions.NewFollowUpSuggestionProvider()
	if err := ce.directManager.RegisterProvider(followUpProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register followup provider").
			WithComponent("chat.completion").
			WithOperation("initializeProviders")
	}
	ce.providers["followup"] = followUpProvider

	// Initialize tool provider (requires tool registry)
	// For now, skip tool provider as it requires a tool registry
	// toolProvider := suggestions.NewToolSuggestionProvider(toolRegistry)
	// Tool provider registration commented out until tool registry is available
	// if err := ce.directManager.RegisterProvider(toolProvider); err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool provider").
	// 		WithComponent("chat.completion").
	// 		WithOperation("initializeProviders")
	// }
	// ce.providers["tool"] = toolProvider

	// Initialize LSP provider (requires LSP manager)
	// For now, skip LSP provider as it requires an LSP manager
	// lspProvider := suggestions.NewLSPSuggestionProvider(lspManager)
	// LSP provider registration commented out until LSP manager is available
	// if err := ce.directManager.RegisterProvider(lspProvider); err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register LSP provider").
	// 		WithComponent("chat.completion").
	// 		WithOperation("initializeProviders")
	// }
	// ce.providers["lsp"] = lspProvider

	// Update the base engine's suggestion manager
	ce.suggestionManager = ce.directManager

	return nil
}

// GetDirectSuggestions gets suggestions directly from the manager without needing ChatHandler
func (ce *CompletionEngineEnhanced) GetDirectSuggestions(ctx context.Context, input string) ([]CompletionResult, error) {
	if ce.directManager == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "suggestion manager not initialized", nil).
			WithComponent("chat.completion").
			WithOperation("GetDirectSuggestions")
	}

	// Build suggestion context
	suggestionCtx := suggestions.SuggestionContext{
		CurrentMessage:      input,
		SessionID:           "chat-session", // Would come from actual session
		ProjectContext:      ce.buildProjectContext(),
		ConversationHistory: ce.conversationHist,
	}

	// Create filter
	filter := &suggestions.SuggestionFilter{
		MaxResults:    5,
		MinConfidence: 0.3,
	}

	// Get suggestions
	suggestions, err := ce.directManager.GetSuggestions(ctx, suggestionCtx, filter)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get suggestions").
			WithComponent("chat.completion").
			WithOperation("GetDirectSuggestions")
	}

	// Convert to completion results
	return ce.convertSuggestionsToCompletions(suggestions), nil
}

// Complete overrides the base Complete method to use direct suggestions
func (ce *CompletionEngineEnhanced) Complete(input string, cursorPos int) []CompletionResult {
	var results []CompletionResult

	// Get traditional completions first
	traditionalResults := ce.getTraditionalCompletions(input, cursorPos)
	results = append(results, traditionalResults...)

	// Add direct suggestions if available
	if ce.directManager != nil && len(input) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		if directSuggestions, err := ce.GetDirectSuggestions(ctx, input); err == nil {
			results = append(results, directSuggestions...)
		}
	}

	// Merge, rank, and deduplicate results
	results = ce.mergeAndRankResults(results, input)

	// If still no results, provide helpful suggestions based on context
	if len(results) == 0 {
		results = ce.getHelpfulSuggestions(input)
	}

	return results
}

// EnableProvider enables a specific suggestion provider
func (ce *CompletionEngineEnhanced) EnableProvider(providerName string) error {
	provider, exists := ce.providers[providerName]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "provider not found", nil).
			WithComponent("chat.completion").
			WithOperation("EnableProvider").
			WithDetails("provider", providerName)
	}

	// Re-register the provider to ensure it's active
	return ce.directManager.RegisterProvider(provider)
}

// DisableProvider would remove a provider (not implemented in the interface, but useful)
func (ce *CompletionEngineEnhanced) DisableProvider(providerName string) error {
	// This would require extending the SuggestionManager interface
	// For now, just remove from our local map
	delete(ce.providers, providerName)
	return nil
}

// UpdateProjectContext updates the project context for better suggestions
func (ce *CompletionEngineEnhanced) UpdateProjectContext(projectPath string) {
	ce.projectRoot = projectPath

	// Update LSP provider if it exists
	// Note: LSP provider would need methods to update project path
	// For now, just update our internal state
	_ = projectPath
}

// GetProviderStatus returns the status of all registered providers
func (ce *CompletionEngineEnhanced) GetProviderStatus() map[string]bool {
	status := make(map[string]bool)
	for name := range ce.providers {
		status[name] = true
	}
	return status
}
