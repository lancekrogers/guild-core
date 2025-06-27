// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/suggestions"
	"github.com/lancekrogers/guild/pkg/tools"
)

// EnhancedGuildArtisan extends GuildArtisan with suggestion capabilities
type EnhancedGuildArtisan interface {
	GuildArtisan

	// GetSuggestionManager returns the suggestion manager for context-aware suggestions
	GetSuggestionManager() suggestions.SuggestionManager

	// GetSuggestionsForContext provides suggestions for the current context
	GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error)

	// ExecuteWithSuggestions executes a task with suggestion assistance
	ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*EnhancedExecutionResult, error)
}

// EnhancedExecutionResult includes both the execution result and contextual suggestions
type EnhancedExecutionResult struct {
	Response    string                   `json:"response"`
	Suggestions []suggestions.Suggestion `json:"suggestions,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
	Success     bool                     `json:"success"`
	Error       string                   `json:"error,omitempty"`
}

// SuggestionAwareWorkerAgent extends WorkerAgent with suggestion capabilities
type SuggestionAwareWorkerAgent struct {
	*WorkerAgent
	suggestionManager suggestions.SuggestionManager
}

// NewSuggestionAwareWorkerAgent creates a new suggestion-aware worker agent
func NewSuggestionAwareWorkerAgent(
	id, name string,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
	suggestionManager suggestions.SuggestionManager,
) *SuggestionAwareWorkerAgent {

	baseAgent := newWorkerAgent(id, name, llmClient, memoryManager, toolRegistry, commissionManager, costManager)

	return &SuggestionAwareWorkerAgent{
		WorkerAgent:       baseAgent,
		suggestionManager: suggestionManager,
	}
}

// GetSuggestionManager returns the suggestion manager
func (a *SuggestionAwareWorkerAgent) GetSuggestionManager() suggestions.SuggestionManager {
	return a.suggestionManager
}

// GetSuggestionsForContext provides suggestions for the current context
func (a *SuggestionAwareWorkerAgent) GetSuggestionsForContext(ctx context.Context, message string, filter *suggestions.SuggestionFilter) ([]suggestions.Suggestion, error) {
	if a.suggestionManager == nil {
		return []suggestions.Suggestion{}, nil
	}

	// Build suggestion context from agent state
	suggestionContext := a.buildSuggestionContext(ctx, message)

	return a.suggestionManager.GetSuggestions(ctx, suggestionContext, filter)
}

// ExecuteWithSuggestions executes a task with suggestion assistance
func (a *SuggestionAwareWorkerAgent) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*EnhancedExecutionResult, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("agent.suggestion_aware").
		WithOperation("ExecuteWithSuggestions").
		With("agent_id", a.ID, "enable_suggestions", enableSuggestions)

	// Execute the base request
	response, err := a.Execute(ctx, request)

	result := &EnhancedExecutionResult{
		Response: response,
		Success:  err == nil,
		Metadata: make(map[string]interface{}),
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Get suggestions if enabled
	if enableSuggestions && a.suggestionManager != nil {
		logger.DebugContext(ctx, "Getting suggestions for executed task")

		suggestions, sugErr := a.GetSuggestionsForContext(ctx, request, nil)
		if sugErr != nil {
			logger.WarnContext(ctx, "Failed to get suggestions", "error", sugErr)
			// Don't fail the main operation for suggestion errors
		} else {
			result.Suggestions = suggestions
			result.Metadata["suggestion_count"] = len(suggestions)

			logger.InfoContext(ctx, "Added suggestions to execution result",
				"suggestion_count", len(suggestions))
		}
	}

	return result, err
}

// buildSuggestionContext builds a suggestion context from the current agent state
func (a *SuggestionAwareWorkerAgent) buildSuggestionContext(ctx context.Context, message string) suggestions.SuggestionContext {
	context := suggestions.SuggestionContext{
		CurrentMessage: message,
		AvailableTools: a.getAvailableTools(),
		ProjectContext: a.getProjectContext(ctx),
	}

	// Get conversation history from memory manager if available
	if a.MemoryManager != nil {
		if history, err := a.getConversationHistory(ctx); err == nil {
			context.ConversationHistory = history
		}
	}

	return context
}

// getAvailableTools converts tool registry tools to suggestion format
func (a *SuggestionAwareWorkerAgent) getAvailableTools() []suggestions.Tool {
	if a.ToolRegistry == nil {
		return []suggestions.Tool{}
	}

	tools := []suggestions.Tool{}

	// Get tools from registry (this would need the registry interface to be extended)
	// For now, return empty slice - this would be implemented when integrating with tool registry

	return tools
}

// getProjectContext extracts project context for suggestions
func (a *SuggestionAwareWorkerAgent) getProjectContext(ctx context.Context) suggestions.ProjectContext {
	// This would extract project information from the current context
	// Implementation would depend on how project context is stored

	return suggestions.ProjectContext{
		// Would populate with actual project detection logic
	}
}

// getConversationHistory gets recent conversation history from memory
func (a *SuggestionAwareWorkerAgent) getConversationHistory(ctx context.Context) ([]suggestions.ChatMessage, error) {
	// This would integrate with the memory manager to get recent conversations
	// Implementation depends on the memory manager interface

	return []suggestions.ChatMessage{}, nil
}
