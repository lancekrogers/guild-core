// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package happypath

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/lancekrogers/guild/pkg/tools/executor"
	"github.com/lancekrogers/guild/pkg/tools/parser"
	"github.com/lancekrogers/guild/pkg/tools/parser/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleToolExecution tests basic tool execution without external dependencies
func TestSimpleToolExecution(t *testing.T) {
	t.Run("Parse OpenAI Format", func(t *testing.T) {
		p := parser.NewResponseParser()

		response := `{
			"tool_calls": [
				{
					"id": "call_123",
					"type": "function",
					"function": {
						"name": "test_tool",
						"arguments": "{\"input\": \"hello\"}"
					}
				}
			]
		}`

		toolCalls, err := p.ExtractToolCalls(response)
		require.NoError(t, err)
		require.Len(t, toolCalls, 1)

		assert.Equal(t, "call_123", toolCalls[0].ID)
		assert.Equal(t, "test_tool", toolCalls[0].Function.Name)
	})

	t.Run("Parse Anthropic XML Format", func(t *testing.T) {
		p := parser.NewResponseParser()

		response := `<function_calls>
<invoke name="test_tool">
<parameter name="input">hello</parameter>
</invoke>
</function_calls>`

		toolCalls, err := p.ExtractToolCalls(response)
		require.NoError(t, err)
		require.Len(t, toolCalls, 1)

		assert.Equal(t, "test_tool", toolCalls[0].Function.Name)

		// Check arguments
		var args map[string]interface{}
		err = json.Unmarshal(toolCalls[0].Function.Arguments, &args)
		require.NoError(t, err)
		assert.Equal(t, "hello", args["input"])
	})

	t.Run("Execute Simple Tool", func(t *testing.T) {
		ctx := context.Background()
		registry := tools.NewToolRegistry()

		// Register a simple test tool
		testTool := &simpleTestTool{}
		err := registry.RegisterTool("simple_test", testTool)
		require.NoError(t, err)

		// Create executor
		exec := executor.NewToolExecutor(registry)

		// Execute tool
		toolCall := types.ToolCall{
			ID:   "test_call",
			Type: "function",
			Function: types.FunctionCall{
				Name:      "simple_test",
				Arguments: json.RawMessage(`{"message": "test"}`),
			},
		}

		result, err := exec.Execute(ctx, toolCall)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "Received: test", result.Content)
	})
}

// simpleTestTool is a basic tool for testing
type simpleTestTool struct{}

func (t *simpleTestTool) Name() string        { return "simple_test" }
func (t *simpleTestTool) Description() string { return "A simple test tool" }
func (t *simpleTestTool) Category() string    { return "test" }
func (t *simpleTestTool) Examples() []string  { return []string{`{"message": "hello"}`} }
func (t *simpleTestTool) RequiresAuth() bool  { return false }
func (t *simpleTestTool) HealthCheck() error  { return nil }

func (t *simpleTestTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "A test message",
			},
		},
		"required": []string{"message"},
	}
}

func (t *simpleTestTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return nil, err
	}

	return &tools.ToolResult{
		Output:  "Received: " + args.Message,
		Success: true,
	}, nil
}
