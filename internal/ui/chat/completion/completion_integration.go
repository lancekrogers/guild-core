// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package completion

import (
	"context"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/suggestions"
)

// CompletionIntegration provides seamless integration between traditional completions and the suggestion system
type CompletionIntegration struct {
	completionEngine  *CompletionEngine
	suggestionManager suggestions.SuggestionManager
	chatHandler       *core.ChatSuggestionHandler
}

// NewCompletionIntegration creates a new completion integration
func NewCompletionIntegration(engine *CompletionEngine) *CompletionIntegration {
	return &CompletionIntegration{
		completionEngine: engine,
	}
}

// SetSuggestionSystem wires up the suggestion system
func (ci *CompletionIntegration) SetSuggestionSystem(manager suggestions.SuggestionManager, handler *core.ChatSuggestionHandler) {
	ci.suggestionManager = manager
	ci.chatHandler = handler

	// Also update the completion engine
	if ci.completionEngine != nil {
		ci.completionEngine.SuggestionManager = manager
		ci.completionEngine.ChatHandler = handler
	}
}

// Complete provides unified completion with both traditional and suggestion-based results
func (ci *CompletionIntegration) Complete(ctx context.Context, input string, cursorPos int) ([]CompletionResult, error) {
	if ci.completionEngine == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "completion engine not initialized", nil).
			WithComponent("chat.completion_integration").
			WithOperation("Complete")
	}

	// Start with traditional completions
	results := ci.completionEngine.Complete(input, cursorPos)

	// If suggestion system is available, enhance with suggestions
	if ci.hasSuggestionSystem() && len(strings.TrimSpace(input)) > 0 {
		// Add timeout to prevent blocking
		suggCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()

		if suggestions, err := ci.getSuggestionsWithContext(suggCtx, input); err == nil {
			results = append(results, suggestions...)
		}
	}

	// Apply final ranking and deduplication
	results = ci.rankAndDeduplicate(results, input)

	return results, nil
}

// hasSuggestionSystem checks if suggestion system is available
func (ci *CompletionIntegration) hasSuggestionSystem() bool {
	return ci.suggestionManager != nil || ci.chatHandler != nil
}

// getSuggestionsWithContext gets context-aware suggestions
func (ci *CompletionIntegration) getSuggestionsWithContext(ctx context.Context, input string) ([]CompletionResult, error) {
	// Try chat handler first (if available)
	if ci.chatHandler != nil {
		request := core.SuggestionRequest{
			Message:        input,
			MaxSuggestions: 3,
			MinConfidence:  0.5,
			Filter: &suggestions.SuggestionFilter{
				MaxResults: 3,
			},
		}

		response, err := ci.chatHandler.GetSuggestions(ctx, request)
		if err == nil && response.Success {
			return ci.convertSuggestionsToCompletions(response.Suggestions), nil
		}
	}

	// Fall back to direct suggestion manager
	if ci.suggestionManager != nil {
		suggestionCtx := suggestions.SuggestionContext{
			CurrentMessage: input,
			SessionID:      "chat-session",
		}

		filter := &suggestions.SuggestionFilter{
			MaxResults:    3,
			MinConfidence: 0.5,
		}

		suggestions, err := ci.suggestionManager.GetSuggestions(ctx, suggestionCtx, filter)
		if err == nil {
			return ci.convertSuggestionsToCompletions(suggestions), nil
		}
	}

	return []CompletionResult{}, nil
}

// convertSuggestionsToCompletions converts suggestions to completion results
func (ci *CompletionIntegration) convertSuggestionsToCompletions(suggestions []suggestions.Suggestion) []CompletionResult {
	results := make([]CompletionResult, 0, len(suggestions))

	for _, suggestion := range suggestions {
		// Use Display if available, otherwise Content
		content := suggestion.Content
		if suggestion.Display != "" {
			content = suggestion.Display
		}

		result := CompletionResult{
			Content: content,
			AgentID: "suggestion-system",
		}

		// Add metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]string)
		}

		result.Metadata["type"] = "suggestion"
		result.Metadata["suggestion_id"] = suggestion.ID
		result.Metadata["suggestion_source"] = suggestion.Source
		result.Metadata["description"] = suggestion.Description
		result.Metadata["icon"] = ci.getSuggestionIcon(suggestion.Type)
		result.Metadata["category"] = string(suggestion.Type)
		result.Metadata["confidence"] = formatConfidence(suggestion.Confidence)

		results = append(results, result)
	}

	return results
}

// getSuggestionIcon returns an icon for the suggestion type
func (ci *CompletionIntegration) getSuggestionIcon(suggestionType suggestions.SuggestionType) string {
	switch suggestionType {
	case suggestions.SuggestionTypeCommand:
		return "⚡"
	case suggestions.SuggestionTypeTool:
		return "🔧"
	case suggestions.SuggestionTypeTemplate:
		return "📝"
	case suggestions.SuggestionTypeFollowUp:
		return "💡"
	case suggestions.SuggestionTypeCode:
		return "💻"
	default:
		return "✨"
	}
}

// rankAndDeduplicate applies final ranking and removes duplicates
func (ci *CompletionIntegration) rankAndDeduplicate(results []CompletionResult, input string) []CompletionResult {
	if len(results) == 0 {
		return results
	}

	// Deduplicate by content (case-insensitive)
	seen := make(map[string]bool)
	deduplicated := make([]CompletionResult, 0, len(results))

	for _, result := range results {
		key := strings.ToLower(result.Content)
		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, result)
		}
	}

	// Apply scoring and sort
	scored := make([]scoredResult, 0, len(deduplicated))
	for _, result := range deduplicated {
		score := ci.calculateScore(result, input)
		scored = append(scored, scoredResult{
			result: result,
			score:  score,
		})
	}

	// Sort by score (descending)
	sortByScore(scored)

	// Extract results
	final := make([]CompletionResult, 0, len(scored))
	for _, sr := range scored {
		final = append(final, sr.result)
	}

	// Limit to reasonable number
	if len(final) > 10 {
		final = final[:10]
	}

	return final
}

// calculateScore calculates a relevance score for a completion result
func (ci *CompletionIntegration) calculateScore(result CompletionResult, input string) float64 {
	score := 0.0
	inputLower := strings.ToLower(input)
	contentLower := strings.ToLower(result.Content)

	// Exact match gets highest score
	if contentLower == inputLower {
		score += 1.0
	}

	// Prefix match gets high score
	if strings.HasPrefix(contentLower, inputLower) {
		score += 0.8
	}

	// Contains match gets medium score
	if strings.Contains(contentLower, inputLower) {
		score += 0.5
	}

	// Boost suggestions with high confidence
	if confidence, ok := result.Metadata["confidence"]; ok {
		if conf, err := parseConfidence(confidence); err == nil {
			score += conf * 0.3
		}
	}

	// Boost by type priority
	if resultType, ok := result.Metadata["type"]; ok {
		switch resultType {
		case "command":
			score += 0.2
		case "suggestion":
			score += 0.15
		case "agent":
			score += 0.1
		}
	}

	return score
}

// Helper types and functions

type scoredResult struct {
	result CompletionResult
	score  float64
}

func sortByScore(results []scoredResult) {
	// Simple bubble sort for small arrays
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if results[j].score < results[j+1].score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

func formatConfidence(conf float64) string {
	// Simple confidence formatting
	return floatToString(conf, 2)
}

func floatToString(f float64, decimals int) string {
	// Simple float to string conversion
	// In production, use fmt.Sprintf("%.2f", f)
	whole := int(f)
	fraction := int((f - float64(whole)) * 100)
	if fraction < 0 {
		fraction = -fraction
	}
	if decimals == 2 {
		if fraction < 10 {
			return string(rune('0'+whole)) + ".0" + string(rune('0'+fraction))
		}
		return string(rune('0'+whole)) + "." + string(rune('0'+fraction/10)) + string(rune('0'+fraction%10))
	}
	return string(rune('0' + whole))
}

func parseConfidence(s string) (float64, error) {
	// Simple confidence parsing
	// In production, use strconv.ParseFloat
	if s == "1" || s == "1.0" || s == "1.00" {
		return 1.0, nil
	}
	if s == "0" || s == "0.0" || s == "0.00" {
		return 0.0, nil
	}
	// Default medium confidence
	return 0.5, nil
}

// UpdateConversationContext updates the conversation context for better suggestions
func (ci *CompletionIntegration) UpdateConversationContext(messages []suggestions.ChatMessage) {
	if ci.completionEngine != nil {
		ci.completionEngine.UpdateConversationHistory(messages)
	}
}

// SetProjectRoot updates the project root for context-aware suggestions
func (ci *CompletionIntegration) SetProjectRoot(projectRoot string) {
	if ci.completionEngine != nil {
		ci.completionEngine.ProjectRoot = projectRoot
	}
}
