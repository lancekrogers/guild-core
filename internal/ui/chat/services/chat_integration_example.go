// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build examples
// +build examples

package services

import (
	"context"
	"fmt"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/suggestions"
)

// ExampleChatWithSuggestions demonstrates how to use the integrated chat service
func ExampleChatWithSuggestions() {
	ctx := context.Background()

	// Create dependencies
	client := &mockGuildClient{} // In production, use real gRPC client
	reg := registry.NewComponentRegistry()

	// Create an enhanced agent (in production, get from registry)
	enhancedAgent := createMockEnhancedAgent()

	// Create chat service with integrated suggestions
	chatService, err := NewChatServiceWithSuggestions(ctx, client, reg, enhancedAgent)
	if err != nil {
		panic(err)
	}

	// Configure suggestion behavior
	chatService.SetSuggestionMode(SuggestionModeBoth) // Get suggestions before and after
	chatService.ConfigureSuggestions(true)            // Enable suggestions

	// Example 1: Send message with pre-execution suggestions
	conversationID := "conv-123"
	cmd := chatService.SendMessageWithSuggestions("developer", "How do I implement a REST API?", conversationID)

	// Execute the command (in a real Bubble Tea app, this would be handled by the runtime)
	msg := cmd()

	// Handle different message types
	switch m := msg.(type) {
	case SuggestionsReceivedMsg:
		fmt.Printf("Pre-execution suggestions:\n")
		for _, s := range m.Suggestions {
			fmt.Printf("- %s: %s (confidence: %.2f)\n", s.Type, s.Content, s.Confidence)
		}
	case AgentResponseMsg:
		fmt.Printf("Agent response: %s\n", m.Content)
	}

	// Example 2: Get follow-up suggestions after response
	followUpCmd := chatService.GetPostExecutionSuggestions(
		"How do I implement a REST API?",
		"To implement a REST API, you should start with...",
	)

	if followUpCmd != nil {
		followUpMsg := followUpCmd()
		if sugMsg, ok := followUpMsg.(SuggestionsReceivedMsg); ok {
			fmt.Printf("\nFollow-up suggestions:\n")
			for _, s := range sugMsg.Suggestions {
				fmt.Printf("- %s: %s\n", s.Type, s.Description)
			}
		}
	}

	// Example 3: Process agent response with automatic suggestion generation
	response := AgentResponseMsg{
		AgentID: "developer",
		Content: "Here's how to implement a REST API...",
		Done:    true,
	}

	processCmd := chatService.ProcessAgentResponse(response, "How do I implement a REST API?")
	if processCmd != nil {
		processMsg := processCmd()
		if respWithSuggestions, ok := processMsg.(AgentResponseWithSuggestionsMsg); ok {
			fmt.Printf("\nResponse with suggestions:\n")
			fmt.Printf("Content: %s\n", respWithSuggestions.Content)
			fmt.Printf("Suggestions: %d\n", len(respWithSuggestions.Suggestions))
			fmt.Printf("Tokens used: %d\n", respWithSuggestions.TokensUsed)
		}
	}

	// Example 4: Get statistics
	stats := chatService.GetStats()
	fmt.Printf("\nChat Service Statistics:\n")
	fmt.Printf("Suggestions enabled: %v\n", stats["suggestions_enabled"])
	fmt.Printf("Suggestion mode: %s\n", stats["suggestion_mode"])
	fmt.Printf("Suggestions enabled: %v\n", stats["suggestions_enabled"])
	fmt.Printf("Token used: %d\n", stats["token_used"])

	// Print suggestion service stats if available
	if cacheHitRate, ok := stats["suggestion_cache_hit_rate"]; ok {
		fmt.Printf("Cache hit rate: %s\n", cacheHitRate)
	}
}

// createMockEnhancedAgent creates a mock enhanced agent for the example
func createMockEnhancedAgent() core.EnhancedGuildArtisan {
	return &mockEnhancedGuildArtisan{
		generateSuggestionsFunc: func(ctx context.Context, request core.SuggestionRequest) ([]suggestions.Suggestion, error) {
			// Return contextual suggestions based on the message
			if contains(request.Message, "REST API") {
				return []suggestions.Suggestion{
					{
						Type:        suggestions.SuggestionTypeTool,
						Content:     "Use the HTTP Router tool",
						Description: "Set up routing for your API endpoints",
						Confidence:  0.95,
					},
					{
						Type:        suggestions.SuggestionTypeContext,
						Content:     "Include authentication middleware",
						Description: "Add JWT or OAuth2 for API security",
						Confidence:  0.90,
					},
					{
						Type:        suggestions.SuggestionTypeExample,
						Content:     "GET /api/users",
						Description: "Example endpoint for fetching users",
						Confidence:  0.85,
					},
				}, nil
			}

			// Default suggestions
			return []suggestions.Suggestion{
				{
					Type:        suggestions.SuggestionTypeGeneral,
					Content:     "Provide more context",
					Description: "More details would help provide better suggestions",
					Confidence:  0.7,
				},
			}, nil
		},
	}
}

// contains is a simple helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

// containsMiddle checks if substr appears in the middle of s
func containsMiddle(s, substr string) bool {
	if len(s) <= len(substr) {
		return false
	}
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// DemonstrateSuggestionModes shows how different suggestion modes work
func DemonstrateSuggestionModes(chatService *ChatService) {
	modes := []struct {
		mode        SuggestionMode
		description string
	}{
		{SuggestionModeNone, "No suggestions"},
		{SuggestionModePre, "Pre-execution suggestions only"},
		{SuggestionModePost, "Post-execution suggestions only"},
		{SuggestionModeBoth, "Both pre and post suggestions"},
	}

	for _, m := range modes {
		fmt.Printf("\n=== %s ===\n", m.description)
		chatService.SetSuggestionMode(m.mode)

		// Test pre-execution suggestions
		preCmd := chatService.GetPreExecutionSuggestions("test message", "conv-123")
		if preCmd != nil {
			fmt.Println("✓ Pre-execution suggestions available")
		} else {
			fmt.Println("✗ Pre-execution suggestions not available")
		}

		// Test post-execution suggestions
		postCmd := chatService.GetPostExecutionSuggestions("original", "response")
		if postCmd != nil {
			fmt.Println("✓ Post-execution suggestions available")
		} else {
			fmt.Println("✗ Post-execution suggestions not available")
		}
	}
}

// DemonstrateTokenOptimization shows how token optimization works
func DemonstrateTokenOptimization(chatService *ChatService) {
	fmt.Println("\n=== Token Optimization Demo ===")

	// Configure suggestions with limits
	if chatService.suggestionService != nil {
		chatService.suggestionService.SetTokenLimit(100)
	}

	// Create a very long message
	longMessage := ""
	for i := 0; i < 1000; i++ {
		longMessage += fmt.Sprintf("This is sentence %d. ", i)
	}

	fmt.Printf("Original message length: %d characters\n", len(longMessage))

	// Send the message - it will be optimized automatically
	cmd := chatService.SendMessage("test-agent", longMessage)
	msg := cmd()

	// Check token usage
	stats := chatService.GetStats()
	fmt.Printf("Suggestions enabled: %v\n", stats["suggestions_enabled"])

	// The message was automatically optimized for better efficiency
	if agentResp, ok := msg.(AgentResponseMsg); ok {
		fmt.Printf("Message was sent successfully to %s\n", agentResp.AgentID)
	}
}
