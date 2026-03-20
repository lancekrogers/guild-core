// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"

	"github.com/lancekrogers/guild-core/pkg/suggestions"
	"github.com/lancekrogers/guild-core/pkg/tools"
	fsTool "github.com/lancekrogers/guild-core/tools/fs"
)

func main() {
	ctx := context.Background()

	fmt.Println("🔮 Guild Suggestion-Tool Integration Demo")
	fmt.Println("========================================")

	// Step 1: Create tool registry using the pkg/tools version (implements interface)
	fmt.Println("\n🔧 Step 1: Creating tool registry...")

	toolRegistry := tools.NewToolRegistry()

	// Register a tool
	globTool := fsTool.NewGlobTool("/workspace")
	if err := toolRegistry.RegisterTool(globTool.Name(), globTool); err != nil {
		fmt.Printf("Error registering tool: %v\n", err)
		return
	}

	fmt.Printf("✅ Registered tool: %s\n", globTool.Name())

	// Step 2: Create suggestion manager with tool provider
	fmt.Println("\n📋 Step 2: Creating suggestion manager with tool provider...")

	manager := suggestions.NewSuggestionManager()

	// Register command provider
	commandProvider := suggestions.NewCommandSuggestionProvider()
	if err := manager.RegisterProvider(commandProvider); err != nil {
		fmt.Printf("Error registering command provider: %v\n", err)
		return
	}

	// Register tool provider using the embedded concrete registry
	toolProvider := suggestions.NewToolSuggestionProvider(toolRegistry.ToolRegistry)
	if err := manager.RegisterProvider(toolProvider); err != nil {
		fmt.Printf("Error registering tool provider: %v\n", err)
		return
	}

	fmt.Println("✅ All providers registered successfully")

	// Step 3: Test tool suggestions
	fmt.Println("\n🔮 Step 3: Testing tool suggestions...")

	// Test file search context
	context1 := suggestions.SuggestionContext{
		CurrentMessage: "I need to find all Go files in my project",
		ProjectContext: suggestions.ProjectContext{
			Language:    "go",
			ProjectType: "library",
		},
	}

	suggestions1, err := manager.GetSuggestions(ctx, context1, nil)
	if err != nil {
		fmt.Printf("Error getting suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d suggestions for file search:\n", len(suggestions1))
		for i, s := range suggestions1 {
			fmt.Printf("  %d. %s: %s (%.2f confidence, type: %s)\n",
				i+1, s.Display, s.Description, s.Confidence, s.Type)
		}
	}

	// Test tool-specific filtering
	fmt.Println("\n🔍 Step 4: Testing tool-specific filtering...")

	filter := &suggestions.SuggestionFilter{
		Types:         []suggestions.SuggestionType{suggestions.SuggestionTypeTool},
		MinConfidence: 0.3,
	}

	toolSuggestions, err := manager.GetSuggestions(ctx, context1, filter)
	if err != nil {
		fmt.Printf("Error getting tool suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d tool-specific suggestions:\n", len(toolSuggestions))
		for i, s := range toolSuggestions {
			fmt.Printf("  %d. 🔧 %s - %s\n", i+1, s.Content, s.Description)
		}
	}

	// Test different context
	fmt.Println("\n--- Different Context ---")
	context2 := suggestions.SuggestionContext{
		CurrentMessage: "show me commands to help with my project",
	}

	suggestions2, err := manager.GetSuggestions(ctx, context2, nil)
	if err != nil {
		fmt.Printf("Error getting suggestions: %v\n", err)
	} else {
		fmt.Printf("Got %d mixed suggestions:\n", len(suggestions2))
		for i, s := range suggestions2 {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, s.Display, s.Type, s.Description)
		}
	}

	fmt.Println("\n🎉 Integration demo completed successfully!")
	fmt.Println("\n📚 Key Points Demonstrated:")
	fmt.Println("  • Tool registry integration working correctly")
	fmt.Println("  • Tool suggestions based on user context")
	fmt.Println("  • Multiple suggestion providers working together")
	fmt.Println("  • Filtering by suggestion type")
	fmt.Println("  • Ready for agent system integration")
}
