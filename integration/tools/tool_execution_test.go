// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/providers/anthropic"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/lancekrogers/guild/pkg/tools/executor"
	"github.com/lancekrogers/guild/pkg/tools/parser"
)

// MockTool implements a simple test tool
type MockTool struct {
	*tools.BaseTool
	executeCalls []string
}

func NewMockTool() *MockTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Test message",
			},
		},
		"required": []string{"message"},
	}

	return &MockTool{
		BaseTool: tools.NewBaseTool(
			"test_tool",
			"A mock tool for testing",
			schema,
			"testing",
			false,
			[]string{`{"message": "hello"}`},
		),
		executeCalls: []string{},
	}
}

func (t *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	t.executeCalls = append(t.executeCalls, input)
	
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	message := params["message"].(string)
	return &tools.ToolResult{
		Success: true,
		Output:  "Mock response: " + message,
	}, nil
}

func TestToolExecutionFlow(t *testing.T) {
	// Create tool registry and register mock tool
	registry := tools.NewToolRegistry()
	mockTool := NewMockTool()
	err := registry.RegisterTool(mockTool)
	require.NoError(t, err)

	// Create tool executor
	toolExec := executor.NewToolExecutor(registry)

	// Test getting available tools
	availableTools := toolExec.GetAvailableTools()
	assert.Len(t, availableTools, 1)
	assert.Equal(t, "test_tool", availableTools[0].Function.Name)

	// Test executing a tool call
	toolCall := parser.ToolCall{
		ID:   "test_call_1",
		Type: "function",
		Function: parser.Function{
			Name:      "test_tool",
			Arguments: json.RawMessage(`{"message": "Hello from test"}`),
		},
	}

	result, err := toolExec.Execute(context.Background(), toolCall)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "Mock response: Hello from test", result.Content)
	assert.Equal(t, "test_call_1", result.ID)

	// Verify tool was called
	assert.Len(t, mockTool.executeCalls, 1)
	assert.Equal(t, `{"message": "Hello from test"}`, mockTool.executeCalls[0])
}

func TestResponseParserIntegration(t *testing.T) {
	responseParser := parser.NewResponseParser()

	// Test parsing various response formats
	testCases := []struct {
		name     string
		response string
		wantCalls int
	}{
		{
			name: "Anthropic XML format",
			response: `I'll help you with that task.

<function_calls>
	<invoke name="test_tool">
		<parameter name="message">Testing XML format</parameter>
	</invoke>
</function_calls>`,
			wantCalls: 1,
		},
		{
			name: "OpenAI JSON format embedded",
			response: `Sure, let me process that for you. Here's the tool call:
			
			"tool_calls": [{
				"id": "call_test",
				"type": "function",
				"function": {
					"name": "test_tool",
					"arguments": "{\"message\": \"Testing JSON format\"}"
				}
			}]`,
			wantCalls: 1,
		},
		{
			name:      "No tool calls",
			response:  "I can explain that concept without using any tools.",
			wantCalls: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			calls, err := responseParser.ExtractToolCalls(tc.response)
			require.NoError(t, err)
			
			if tc.wantCalls == 0 {
				assert.Nil(t, calls)
			} else {
				require.Len(t, calls, tc.wantCalls)
				assert.Equal(t, "test_tool", calls[0].Function.Name)
			}
		})
	}
}

func TestAnthropicProviderToolSupport(t *testing.T) {
	t.Skip("Requires API key - enable for integration testing")
	
	// This test demonstrates how the Anthropic provider would handle tools
	client := anthropic.NewClient("")
	
	// Check capabilities
	caps := client.GetCapabilities()
	assert.True(t, caps.SupportsTools)

	// Create a mock request with tools
	tools := []interfaces.ToolDefinition{
		{
			Type: "function",
			Function: interfaces.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"location": {
							"type": "string",
							"description": "The city and state, e.g. San Francisco, CA"
						}
					},
					"required": ["location"]
				}`),
			},
		},
	}

	req := interfaces.ChatRequestWithTools{
		ChatRequest: interfaces.ChatRequest{
			Model: anthropic.Claude35Haiku,
			Messages: []interfaces.ChatMessage{
				{
					Role:    "user",
					Content: "What's the weather in San Francisco?",
				},
			},
			MaxTokens: 1000,
		},
		Tools: tools,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.ChatCompletionWithTools(ctx, req)
	require.NoError(t, err)
	
	// Check if tool was called
	assert.NotEmpty(t, resp.Choices)
	if len(resp.ToolCalls) > 0 {
		assert.Equal(t, "get_weather", resp.ToolCalls[0].Function.Name)
		
		// Verify arguments
		var args map[string]interface{}
		err = json.Unmarshal(resp.ToolCalls[0].Function.Arguments, &args)
		require.NoError(t, err)
		assert.Contains(t, args["location"], "San Francisco")
	}
}

func TestToolExecutorBatch(t *testing.T) {
	// Create registry with multiple tools
	registry := tools.NewToolRegistry()
	
	tool1 := NewMockTool()
	tool1.BaseTool = tools.NewBaseTool(
		"tool1",
		"First test tool",
		nil,
		"testing",
		false,
		nil,
	)
	
	tool2 := NewMockTool()
	tool2.BaseTool = tools.NewBaseTool(
		"tool2",
		"Second test tool",
		nil,
		"testing",
		false,
		nil,
	)
	
	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)

	toolExec := executor.NewToolExecutor(registry)

	// Create multiple tool calls
	calls := []parser.ToolCall{
		{
			ID:   "call1",
			Type: "function",
			Function: parser.Function{
				Name:      "tool1",
				Arguments: json.RawMessage(`{"message": "First"}`),
			},
		},
		{
			ID:   "call2",
			Type: "function",
			Function: parser.Function{
				Name:      "tool2",
				Arguments: json.RawMessage(`{"message": "Second"}`),
			},
		},
	}

	results, err := toolExec.ExecuteBatch(context.Background(), calls)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	
	assert.Equal(t, "call1", results[0].ID)
	assert.True(t, results[0].Success)
	assert.Equal(t, "Mock response: First", results[0].Content)
	
	assert.Equal(t, "call2", results[1].ID)
	assert.True(t, results[1].Success)
	assert.Equal(t, "Mock response: Second", results[1].Content)
}