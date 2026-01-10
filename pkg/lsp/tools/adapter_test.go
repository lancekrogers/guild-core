// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/lsp"
	lsptools "github.com/lancekrogers/guild-core/pkg/lsp/tools"
	"github.com/lancekrogers/guild-core/tools"
)

func TestRegistryAdapter(t *testing.T) {
	// Create a mock LSP manager
	manager, err := lsp.NewManager("")
	require.NoError(t, err)
	defer manager.Shutdown(context.Background())

	// Create an LSP tool
	completionTool := lsptools.NewCompletionTool(manager)

	// Convert to registry tool
	registryTool := lsptools.ToRegistryTool(completionTool)

	// Verify it implements the tools.Tool interface
	var _ tools.Tool = registryTool

	// Test basic properties
	assert.Equal(t, "lsp_completion", registryTool.Name())
	assert.Equal(t, "Get code completions at a specific position in a file using Language Server Protocol", registryTool.Description())
	assert.Equal(t, "code", registryTool.Category())
	assert.False(t, registryTool.RequiresAuth())

	// Test schema
	schema := registryTool.Schema()
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	// Test examples
	examples := registryTool.Examples()
	assert.Len(t, examples, 2)
	assert.Contains(t, examples[0], "main.go")
	assert.Contains(t, examples[1], "app.ts")
}

func TestAllLSPToolsAdapter(t *testing.T) {
	// Create a mock LSP manager
	manager, err := lsp.NewManager("")
	require.NoError(t, err)
	defer manager.Shutdown(context.Background())

	// Test all LSP tools can be adapted
	tools := []struct {
		name string
		tool tools.Tool
	}{
		{"lsp_completion", lsptools.ToRegistryTool(lsptools.NewCompletionTool(manager))},
		{"lsp_definition", lsptools.ToRegistryTool(lsptools.NewDefinitionTool(manager))},
		{"lsp_references", lsptools.ToRegistryTool(lsptools.NewReferencesTool(manager))},
		{"lsp_hover", lsptools.ToRegistryTool(lsptools.NewHoverTool(manager))},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, tt.tool.Name())
			assert.NotEmpty(t, tt.tool.Description())
			assert.NotNil(t, tt.tool.Schema())
			assert.NotEmpty(t, tt.tool.Examples())

			// Verify they all belong to the "code" category
			assert.Equal(t, "code", tt.tool.Category())
		})
	}
}

func TestAdapterExecution(t *testing.T) {
	ctx := context.Background()

	// Create a mock LSP manager
	manager, err := lsp.NewManager("")
	require.NoError(t, err)
	defer manager.Shutdown(ctx)

	// Create and adapt a tool
	completionTool := lsptools.NewCompletionTool(manager)
	registryTool := lsptools.ToRegistryTool(completionTool)

	// Test with invalid input (should fail gracefully)
	result, err := registryTool.Execute(ctx, "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test with valid but incomplete input
	result, err = registryTool.Execute(ctx, `{"file": "test.go"}`)
	assert.Error(t, err)
	assert.Nil(t, result)
}
