// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package happypath

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-framework/guild-core/pkg/tools"
	"github.com/guild-framework/guild-core/pkg/tools/executor"
	"github.com/guild-framework/guild-core/pkg/tools/parser"
	"github.com/guild-framework/guild-core/pkg/tools/parser/types"
	"github.com/guild-framework/guild-core/tools/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealToolExecution tests with actual file system tools
func TestRealToolExecution(t *testing.T) {
	t.Run("Execute File Tool", func(t *testing.T) {
		// Create a temp directory for testing
		tmpDir, err := os.MkdirTemp("", "guild-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a test file
		testFile := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(testFile, []byte("Hello from Guild!"), 0o644)
		require.NoError(t, err)

		// Setup
		ctx := context.Background()
		registry := tools.NewToolRegistry()

		// Register file tool
		fileTool := fs.NewFileTool(tmpDir)
		err = registry.RegisterTool("file", fileTool)
		require.NoError(t, err)

		// Create executor
		exec := executor.NewToolExecutor(registry)

		// Create tool call to read the file (use relative path)
		toolCall := types.ToolCall{
			ID:   "read_file",
			Type: "function",
			Function: types.FunctionCall{
				Name:      "file",
				Arguments: json.RawMessage(`{"operation": "read", "path": "test.txt"}`),
			},
		}

		// Execute
		result, err := exec.Execute(ctx, toolCall)
		require.NoError(t, err)

		// Debug output
		t.Logf("Result Success: %v", result.Success)
		t.Logf("Result Content: %q", result.Content)
		t.Logf("Result Error: %q", result.Error)

		assert.True(t, result.Success, "Expected success, got error: %s", result.Error)
		assert.Contains(t, result.Content, "Hello from Guild!")
	})

	t.Run("Execute Glob Tool", func(t *testing.T) {
		// Create a temp directory with test files
		tmpDir, err := os.MkdirTemp("", "guild-glob-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create test files
		for i := 0; i < 3; i++ {
			filename := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".go")
			err = os.WriteFile(filename, []byte("package test"), 0o644)
			require.NoError(t, err)
		}

		// Setup
		ctx := context.Background()
		registry := tools.NewToolRegistry()

		// Register glob tool
		globTool := fs.NewGlobTool(tmpDir)
		err = registry.RegisterTool("glob", globTool)
		require.NoError(t, err)

		// Create executor
		exec := executor.NewToolExecutor(registry)

		// Create tool call to glob files
		toolCall := types.ToolCall{
			ID:   "glob_files",
			Type: "function",
			Function: types.FunctionCall{
				Name:      "glob",
				Arguments: json.RawMessage(`{"pattern": "*.go", "path": "` + tmpDir + `"}`),
			},
		}

		// Execute
		result, err := exec.Execute(ctx, toolCall)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Content, "test0.go")
		assert.Contains(t, result.Content, "test1.go")
		assert.Contains(t, result.Content, "test2.go")
	})
}

// TestProviderResponseWithTools simulates provider responses with tool calls
func TestProviderResponseWithTools(t *testing.T) {
	t.Run("OpenAI Style Response", func(t *testing.T) {
		// Simulate a full OpenAI response with tool calls
		openAIResponse := `{
  "tool_calls": [
    {
      "id": "call_file_read",
      "type": "function",
      "function": {
        "name": "file",
        "arguments": "{\"operation\": \"read\", \"path\": \"/tmp/test.txt\"}"
      }
    }
  ]
}`

		// Parse the response
		p := parser.NewResponseParser()
		toolCalls, err := p.ExtractToolCalls(openAIResponse)
		require.NoError(t, err)
		require.Len(t, toolCalls, 1)

		// Verify the parsed tool call
		assert.Equal(t, "call_file_read", toolCalls[0].ID)
		assert.Equal(t, "file", toolCalls[0].Function.Name)

		// Parse arguments
		var args map[string]interface{}
		err = json.Unmarshal(toolCalls[0].Function.Arguments, &args)
		require.NoError(t, err)
		assert.Equal(t, "read", args["operation"])
		assert.Equal(t, "/tmp/test.txt", args["path"])
	})

	t.Run("Anthropic Style Response", func(t *testing.T) {
		// Simulate Anthropic response with XML tool calls
		anthropicResponse := `Let me search for files matching that pattern.

<function_calls>
<invoke name="glob">
<parameter name="pattern">**/*.go</parameter>
<parameter name="path">/project/src</parameter>
</invoke>
</function_calls>

I'll look for all Go files in the source directory.`

		// Parse the response
		p := parser.NewResponseParser()
		toolCalls, err := p.ExtractToolCalls(anthropicResponse)
		require.NoError(t, err)
		require.Len(t, toolCalls, 1)

		// Verify the parsed tool call
		assert.Equal(t, "glob", toolCalls[0].Function.Name)

		// Parse arguments
		var args map[string]interface{}
		err = json.Unmarshal(toolCalls[0].Function.Arguments, &args)
		require.NoError(t, err)
		assert.Equal(t, "**/*.go", args["pattern"])
		assert.Equal(t, "/project/src", args["path"])
	})
}
