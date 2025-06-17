// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/suggestions"
)

func main() {
	ctx := context.Background()
	
	fmt.Println("🔮 Guild Suggestion System Demo")
	fmt.Println("===============================")
	
	// Step 1: Create suggestion manager
	fmt.Println("\n📋 Step 1: Creating suggestion manager...")
	
	manager := suggestions.NewSuggestionManager()
	
	// Step 2: Register providers
	fmt.Println("\n🔧 Step 2: Registering suggestion providers...")
	
	// Register command provider
	commandProvider := suggestions.NewCommandSuggestionProvider()
	if err := manager.RegisterProvider(commandProvider); err != nil {
		fmt.Printf("Error registering command provider: %v\n", err)
		return
	}
	fmt.Println("✅ Command provider registered")
	
	// Register follow-up provider
	followUpProvider := suggestions.NewFollowUpSuggestionProvider()
	if err := manager.RegisterProvider(followUpProvider); err != nil {
		fmt.Printf("Error registering follow-up provider: %v\n", err)
		return
	}
	fmt.Println("✅ Follow-up provider registered")
	
	// Step 3: Test different suggestion scenarios
	fmt.Println("\n🔮 Step 3: Testing suggestion scenarios...")
	
	// Scenario 1: Help request
	fmt.Println("\n--- Scenario 1: Help Request ---")
	context1 := suggestions.SuggestionContext{
		CurrentMessage: "I need help with my project",
		ConversationHistory: []suggestions.ChatMessage{
			{Role: "user", Content: "I'm starting a new project", Timestamp: 1},
		},
	}
	
	suggestions1, err := manager.GetSuggestions(ctx, context1, nil)
	if err != nil {
		fmt.Printf("Error getting suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d suggestions:\n", len(suggestions1))
		for i, s := range suggestions1 {
			fmt.Printf("  %d. %s: %s (%.2f confidence)\n", i+1, s.Display, s.Description, s.Confidence)
		}
	}
	
	// Scenario 2: Template request
	fmt.Println("\n--- Scenario 2: Template Request ---")
	context2 := suggestions.SuggestionContext{
		CurrentMessage: "show me templates for documentation",
		ConversationHistory: []suggestions.ChatMessage{
			{Role: "user", Content: "I need to write documentation", Timestamp: 1},
			{Role: "assistant", Content: "I can help you with documentation", Timestamp: 2},
		},
	}
	
	suggestions2, err := manager.GetSuggestions(ctx, context2, nil)
	if err != nil {
		fmt.Printf("Error getting suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d suggestions:\n", len(suggestions2))
		for i, s := range suggestions2 {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, s.Content, s.Type, s.Description)
		}
	}
	
	// Scenario 3: Follow-up context
	fmt.Println("\n--- Scenario 3: Follow-up Context ---")
	context3 := suggestions.SuggestionContext{
		CurrentMessage: "what should I do next?",
		ConversationHistory: []suggestions.ChatMessage{
			{Role: "user", Content: "I just finished implementing the login feature", Timestamp: 1},
			{Role: "assistant", Content: "Great! The login feature looks good", Timestamp: 2},
		},
	}
	
	suggestions3, err := manager.GetSuggestions(ctx, context3, nil)
	if err != nil {
		fmt.Printf("Error getting suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d suggestions:\n", len(suggestions3))
		for i, s := range suggestions3 {
			fmt.Printf("  %d. %s - %s\n", i+1, s.Display, s.Description)
		}
	}
	
	// Step 4: Test filtering
	fmt.Println("\n🔍 Step 4: Testing suggestion filtering...")
	
	filter := &suggestions.SuggestionFilter{
		Types:         []suggestions.SuggestionType{suggestions.SuggestionTypeCommand},
		MinConfidence: 0.5,
		MaxResults:    3,
	}
	
	filteredSuggestions, err := manager.GetSuggestions(ctx, context1, filter)
	if err != nil {
		fmt.Printf("Error getting filtered suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d filtered command suggestions:\n", len(filteredSuggestions))
		for i, s := range filteredSuggestions {
			fmt.Printf("  %d. %s (confidence: %.2f)\n", i+1, s.Display, s.Confidence)
		}
	}
	
	// Step 5: Test analytics and usage tracking
	fmt.Println("\n📊 Step 5: Testing analytics...")
	
	// Record some usage
	err = manager.RecordUsage(ctx, "help", suggestions.SuggestionUsage{
		SuggestionID: "cmd_help",
		Accepted:     true,
		Context:      "user_help_request",
	})
	if err != nil {
		fmt.Printf("Error recording usage: %v\n", err)
	} else {
		fmt.Println("✅ Usage recorded")
	}
	
	// Get analytics
	analytics, err := manager.GetAnalytics(ctx)
	if err != nil {
		fmt.Printf("Error getting analytics: %v\n", err)
	} else {
		fmt.Printf("Analytics: %+v\n", analytics)
	}
	
	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n📚 Summary:")
	fmt.Println("  • Suggestion system working with multiple providers")
	fmt.Println("  • Command suggestions based on user input")
	fmt.Println("  • Follow-up suggestions based on conversation history")
	fmt.Println("  • Filtering by type, confidence, and result count")
	fmt.Println("  • Usage tracking and analytics")
	fmt.Println("  • Ready for integration with agent system")
}