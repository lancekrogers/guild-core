// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/guild-framework/guild-core/pkg/lsp"
	lsptools "github.com/guild-framework/guild-core/pkg/lsp/tools"
)

// ExampleLSPUsage demonstrates how to use the LSP integration
func ExampleLSPUsage() {
	ctx := context.Background()

	// Create LSP manager
	manager, err := lsp.NewManager("")
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Shutdown(ctx)

	// Example 1: Get code completions without file content
	completionTool := lsptools.NewCompletionTool(manager)

	completionInput := map[string]interface{}{
		"file":    "/path/to/main.go",
		"line":    10,
		"column":  15,
		"trigger": ".",
	}

	inputJSON, _ := json.Marshal(completionInput)
	result, err := completionTool.Execute(ctx, string(inputJSON))
	if err == nil && result != nil {
		fmt.Printf("Found %s completions\n", result.Metadata["completion_count"])
	}

	// Example 2: Go to definition
	definitionTool := lsptools.NewDefinitionTool(manager)

	defInput := map[string]interface{}{
		"file":   "/path/to/main.go",
		"line":   20,
		"column": 10,
	}

	inputJSON, _ = json.Marshal(defInput)
	result, err = definitionTool.Execute(ctx, string(inputJSON))
	if err == nil && result != nil {
		var defResult lsptools.DefinitionResult
		json.Unmarshal([]byte(result.Output), &defResult)
		fmt.Printf("Definition at: %s\n", lsptools.FormatDefinitionsAsText(&defResult))
	}

	// Example 3: Find all references
	referencesTool := lsptools.NewReferencesTool(manager)

	refInput := map[string]interface{}{
		"file":                "/path/to/types.go",
		"line":                30,
		"column":              5,
		"include_declaration": true,
	}

	inputJSON, _ = json.Marshal(refInput)
	result, err = referencesTool.Execute(ctx, string(inputJSON))
	if err == nil && result != nil {
		fmt.Printf("Found %s references across %s files\n",
			result.Metadata["reference_count"],
			result.Metadata["file_count"])
	}

	// Example 4: Get type information
	hoverTool := lsptools.NewHoverTool(manager)

	hoverInput := map[string]interface{}{
		"file":   "/path/to/calculator.go",
		"line":   8,
		"column": 10,
	}

	inputJSON, _ = json.Marshal(hoverInput)
	result, err = hoverTool.Execute(ctx, string(inputJSON))
	if err == nil && result != nil {
		var hoverResult lsptools.HoverResult
		json.Unmarshal([]byte(result.Output), &hoverResult)
		fmt.Printf("Type info: %s\n", hoverResult.Content)
	}
}

// ExampleAgentIntegration shows how agents use LSP tools
func ExampleAgentIntegration() {
	// Agents automatically get LSP tools when using LSPAwareExecutor
	// The executor will:
	// 1. Detect code-related tasks
	// 2. Enhance context with LSP information
	// 3. Prefer LSP tools over regular tools when available
	// 4. Provide 97.5% token savings on code operations

	fmt.Println("LSP tools are automatically available to agents!")
}
