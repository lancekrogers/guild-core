//go:build example
// +build example

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/guild-ventures/guild-core/tools"
	"github.com/guild-ventures/guild-core/tools/fs"
)

// This example demonstrates how to use the grep tool
func main() {
	// Create a grep tool instance
	grepTool := fs.NewGrepTool(".")

	// Create a tool registry and register the grep tool
	registry := tools.NewToolRegistry()
	if err := registry.RegisterTool(grepTool); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Example 1: Search for TODO comments
	fmt.Println("=== Example 1: Finding TODO comments ===")
	result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "TODO"}`)
	if err != nil {
		log.Fatal(err)
	}
	printResult(result)

	// Example 2: Search for function definitions in Go files
	fmt.Println("\n=== Example 2: Finding Go functions ===")
	result, err = registry.ExecuteTool(ctx, "grep", `{"pattern": "func\\s+\\w+", "include": "*.go"}`)
	if err != nil {
		log.Fatal(err)
	}
	printResult(result)

	// Example 3: Search in a specific directory
	fmt.Println("\n=== Example 3: Search in specific directory ===")
	result, err = registry.ExecuteTool(ctx, "grep", `{"pattern": "import", "path": "./pkg", "include": "*.go"}`)
	if err != nil {
		log.Fatal(err)
	}
	printResult(result)

	// Example 4: Using brace expansion for multiple file types
	fmt.Println("\n=== Example 4: Search in multiple file types ===")
	result, err = registry.ExecuteTool(ctx, "grep", `{"pattern": "error", "include": "*.{go,js,ts}"}`)
	if err != nil {
		log.Fatal(err)
	}
	printResult(result)
}

func printResult(result *tools.ToolResult) {
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		return
	}

	// Parse the JSON output
	var grepResult fs.GrepResult
	if err := json.Unmarshal([]byte(result.Output), &grepResult); err != nil {
		log.Printf("Failed to parse result: %v", err)
		return
	}

	fmt.Printf("Found %d files matching pattern '%s'\n", grepResult.Count, grepResult.Pattern)
	if grepResult.Include != "" {
		fmt.Printf("File filter: %s\n", grepResult.Include)
	}
	fmt.Printf("Search directory: %s\n", grepResult.SearchDir)
	
	// Show first 5 matches
	for i, file := range grepResult.Files {
		if i >= 5 {
			fmt.Printf("... and %d more files\n", len(grepResult.Files)-5)
			break
		}
		fmt.Printf("  - %s (%d matches)\n", file.RelativePath, file.MatchCount)
	}
}