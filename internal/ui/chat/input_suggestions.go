// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/guild-framework/guild-core/internal/ui/chat/completion"
)

// InputSuggestionManager handles real-time input suggestions with debouncing
type InputSuggestionManager struct {
	app           *App
	debounceTimer *time.Timer
	debounceDelay time.Duration
	lastInput     string
	isRequesting  bool
}

// NewInputSuggestionManager creates a new suggestion manager
func NewInputSuggestionManager(app *App, debounceDelay time.Duration) *InputSuggestionManager {
	return &InputSuggestionManager{
		app:           app,
		debounceDelay: debounceDelay,
		lastInput:     "",
		isRequesting:  false,
	}
}

// HandleInputChange processes input changes and triggers suggestions
func (ism *InputSuggestionManager) HandleInputChange(input string) tea.Cmd {
	// Skip if input hasn't really changed or is empty
	if input == ism.lastInput || strings.TrimSpace(input) == "" {
		// Hide suggestions for empty input
		if strings.TrimSpace(input) == "" {
			ism.app.inputPane.HideCompletions()
		}
		return nil
	}

	ism.lastInput = input

	// Cancel existing timer
	if ism.debounceTimer != nil {
		ism.debounceTimer.Stop()
	}

	// Hide suggestions immediately if input is getting shorter (user is deleting)
	if len(input) < len(ism.lastInput) {
		ism.app.inputPane.HideCompletions()
	}

	// Create new debounced command
	return ism.createDebouncedSuggestionCmd(input)
}

// createDebouncedSuggestionCmd creates a debounced command for suggestions
func (ism *InputSuggestionManager) createDebouncedSuggestionCmd(input string) tea.Cmd {
	return func() tea.Msg {
		// Set timer for debounce
		ism.debounceTimer = time.NewTimer(ism.debounceDelay)
		<-ism.debounceTimer.C

		// Request suggestions after debounce
		return completion.SuggestionRequestMsg{
			Input:     input,
			Timestamp: time.Now(),
		}
	}
}

// ProcessSuggestionRequest handles the actual suggestion request
func (ism *InputSuggestionManager) ProcessSuggestionRequest(input string) tea.Cmd {
	// Skip if already requesting or input changed
	if ism.isRequesting || input != ism.app.inputPane.GetValue() {
		return nil
	}

	ism.isRequesting = true

	return func() tea.Msg {
		defer func() { ism.isRequesting = false }()

		// Use completion engine to get suggestions
		if ism.app.completionEngine != nil {
			results := ism.app.completionEngine.Complete(input, len(input))

			// Filter and enhance results based on context
			enhancedResults := ism.enhanceCompletionResults(results, input)

			return completion.CompletionResultMsg{
				Results:   enhancedResults,
				ForInput:  input,
				Timestamp: time.Now(),
			}
		}

		return completion.CompletionResultMsg{
			Results:   []completion.CompletionResult{},
			ForInput:  input,
			Timestamp: time.Now(),
		}
	}
}

// enhanceCompletionResults enhances completion results with additional context
func (ism *InputSuggestionManager) enhanceCompletionResults(results []completion.CompletionResult, input string) []completion.CompletionResult {
	enhanced := make([]completion.CompletionResult, 0, len(results))

	for _, result := range results {
		// Add input context to metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]string)
		}
		result.Metadata["original_input"] = input

		// Add relevance score based on match quality
		result.Metadata["relevance"] = ism.calculateRelevance(result.Content, input)

		// Add preview of how the completion would look
		result.Metadata["preview"] = ism.generatePreview(result.Content, input)

		enhanced = append(enhanced, result)
	}

	// Sort by relevance
	ism.sortByRelevance(enhanced)

	// Limit to top suggestions
	if len(enhanced) > 10 {
		enhanced = enhanced[:10]
	}

	return enhanced
}

// calculateRelevance calculates how relevant a suggestion is to the input
func (ism *InputSuggestionManager) calculateRelevance(suggestion, input string) string {
	// Simple relevance calculation
	lowerSuggestion := strings.ToLower(suggestion)
	lowerInput := strings.ToLower(input)

	if strings.HasPrefix(lowerSuggestion, lowerInput) {
		return "high"
	} else if strings.Contains(lowerSuggestion, lowerInput) {
		return "medium"
	}
	return "low"
}

// generatePreview shows how the suggestion would complete the input
func (ism *InputSuggestionManager) generatePreview(suggestion, input string) string {
	// For commands and mentions, show the full suggestion
	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "@") {
		return suggestion
	}

	// For other inputs, show how it would complete
	if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(input)) {
		return suggestion
	}

	// Default to showing the suggestion as-is
	return suggestion
}

// sortByRelevance sorts suggestions by their relevance score
func (ism *InputSuggestionManager) sortByRelevance(results []completion.CompletionResult) {
	// Simple bubble sort for small lists
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if ism.compareRelevance(results[i], results[j]) < 0 {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// compareRelevance compares two results by relevance
func (ism *InputSuggestionManager) compareRelevance(a, b completion.CompletionResult) int {
	aRel := a.Metadata["relevance"]
	bRel := b.Metadata["relevance"]

	// Convert relevance to numeric score
	scoreMap := map[string]int{
		"high":   3,
		"medium": 2,
		"low":    1,
		"":       0,
	}

	aScore := scoreMap[aRel]
	bScore := scoreMap[bRel]

	return aScore - bScore
}
