// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/guild-framework/guild-core/tools"
)

// ExampleAgToolUsage demonstrates how to use the Silver Searcher (ag) tool
func ExampleAgToolUsage() {
	// Create an ag tool instance
	agTool := NewAgTool("/path/to/your/workspace")

	// Example 1: Basic text search
	basicSearch := AgToolInput{
		Pattern: "func main",
	}
	runSearchExample("Basic Search", agTool, basicSearch)

	// Example 2: Search with file type filtering
	fileTypeSearch := AgToolInput{
		Pattern:   "import",
		FileTypes: []string{"go", "js"},
	}
	runSearchExample("File Type Filtering", agTool, fileTypeSearch)

	// Example 3: Case-sensitive search
	caseSensitiveSearch := AgToolInput{
		Pattern:       "TODO",
		CaseSensitive: true,
	}
	runSearchExample("Case Sensitive Search", agTool, caseSensitiveSearch)

	// Example 4: Search with context lines
	contextSearch := AgToolInput{
		Pattern: "error",
		Context: 2,
	}
	runSearchExample("Search with Context", agTool, contextSearch)

	// Example 5: Search with ignore patterns
	ignoreSearch := AgToolInput{
		Pattern:        "test",
		IgnorePatterns: []string{"*.test.go", "testdata"},
	}
	runSearchExample("Search with Ignore Patterns", agTool, ignoreSearch)

	// Example 6: Literal string search (no regex)
	literalSearch := AgToolInput{
		Pattern: "func()",
		Literal: true,
	}
	runSearchExample("Literal Search", agTool, literalSearch)

	// Example 7: Limited results search
	limitedSearch := AgToolInput{
		Pattern:    "struct",
		MaxResults: 5,
	}
	runSearchExample("Limited Results", agTool, limitedSearch)
}

// runSearchExample executes a search example and prints results
func runSearchExample(name string, tool *AgTool, input AgToolInput) {
	fmt.Printf("\n=== %s ===\n", name)

	// Convert input to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		log.Printf("Failed to marshal input: %v", err)
		return
	}

	// Execute the search
	ctx := context.Background()
	result, err := tool.Execute(ctx, string(inputJSON))
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	// Handle the case where ag is not installed
	if result.Metadata["error"] == "ag_not_installed" {
		fmt.Println("Silver Searcher (ag) is not installed. Please install it to use this tool.")
		return
	}

	// Print basic metadata
	fmt.Printf("Pattern: %s\n", result.Metadata["pattern"])
	fmt.Printf("Total results: %s\n", result.Metadata["total"])
	fmt.Printf("Duration: %s\n", result.Metadata["duration"])

	// Parse and display structured results
	if result.Error == "" {
		var agResult AgToolResult
		if err := json.Unmarshal([]byte(result.Output), &agResult); err != nil {
			log.Printf("Failed to parse results: %v", err)
			return
		}

		// Display first few results
		displayCount := min(len(agResult.Results), 3)
		for i, searchResult := range agResult.Results[:displayCount] {
			fmt.Printf("  %d. %s:%d:%d - %s\n",
				i+1,
				searchResult.File,
				searchResult.Line,
				searchResult.Column,
				truncateString(searchResult.Match, 60))
		}

		if len(agResult.Results) > displayCount {
			fmt.Printf("  ... and %d more results\n", len(agResult.Results)-displayCount)
		}

		if agResult.Truncated {
			fmt.Println("  (Results truncated)")
		}
	} else {
		fmt.Printf("Error: %s\n", result.Error)
	}
}

// ExampleRegistryIntegration demonstrates how to register the ag tool with Guild's registry
func ExampleRegistryIntegration() {
	// Create a tool registry
	registry := tools.NewToolRegistry()

	// Register search tools
	workspacePath := "/path/to/workspace"
	err := RegisterSearchTools(registry, workspacePath)
	if err != nil {
		log.Fatalf("Failed to register search tools: %v", err)
	}

	// Get the ag tool from registry
	agTool, exists := registry.GetTool("ag")
	if !exists {
		log.Fatal("ag tool not found in registry")
	}

	// Use the tool
	ctx := context.Background()
	input := `{"pattern": "TODO", "file_types": ["go"]}`
	result, err := agTool.Execute(ctx, input)
	if err != nil {
		log.Fatalf("Tool execution failed: %v", err)
	}

	fmt.Printf("Search completed: %s results found\n", result.Metadata["total"])
}

// ExampleCostAwareRegistration demonstrates cost-aware tool registration
func ExampleCostAwareRegistration() {
	// This would typically be done with Guild's ComponentRegistry
	// that supports cost-aware tool registration

	workspacePath := "/path/to/workspace"
	searchTools := GetSearchTools(workspacePath)

	for _, toolInfo := range searchTools {
		fmt.Printf("Tool: %s\n", toolInfo.Tool.Name())
		fmt.Printf("Cost Magnitude: %d (Fibonacci scale)\n", toolInfo.CostMagnitude)
		fmt.Printf("Capabilities: %v\n", toolInfo.Capabilities)
		fmt.Println("---")
	}
}

// ExampleValidateInstallation demonstrates how to check if ag is installed
func ExampleValidateInstallation() {
	err := ValidateAgInstallation()
	if err != nil {
		fmt.Printf("ag validation failed: %v\n", err)
		fmt.Println("Installation instructions:")
		fmt.Println("  macOS: brew install the_silver_searcher")
		fmt.Println("  Ubuntu: apt-get install silversearcher-ag")
		fmt.Println("  CentOS/RHEL: yum install the_silver_searcher")
	} else {
		fmt.Println("Silver Searcher (ag) is properly installed and available.")
	}
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ExampleSearchPatterns demonstrates common search patterns
func ExampleSearchPatterns() {
	patterns := map[string]string{
		"Find functions":          `{"pattern": "func \\w+", "file_types": ["go"]}`,
		"Find TODO comments":      `{"pattern": "TODO|FIXME|XXX", "case_sensitive": true}`,
		"Find imports":            `{"pattern": "^import", "file_types": ["go", "js", "py"]}`,
		"Find error handling":     `{"pattern": "if.*err.*!=.*nil", "file_types": ["go"]}`,
		"Find struct definitions": `{"pattern": "type.*struct", "file_types": ["go"], "whole_word": true}`,
		"Find JSON files":         `{"pattern": "\\{.*\\}", "file_types": ["json"]}`,
		"Find configuration":      `{"pattern": "config|Config", "file_types": ["go", "yaml", "json"]}`,
		"Find test functions":     `{"pattern": "func Test", "file_types": ["go"], "path": "./test"}`,
	}

	fmt.Println("Common ag search patterns:")
	for description, pattern := range patterns {
		fmt.Printf("  %s:\n    %s\n", description, pattern)
	}
}
