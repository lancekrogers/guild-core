// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseParser_ExtractToolCalls(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		response string
		wantLen  int
		validate func(t *testing.T, calls []ToolCall)
	}{
		{
			name: "OpenAI JSON format",
			response: `{
				"tool_calls": [
					{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "read_file",
							"arguments": "{\"path\": \"/test.txt\"}"
						}
					}
				]
			}`,
			wantLen: 1,
			validate: func(t *testing.T, calls []ToolCall) {
				assert.Equal(t, "call_123", calls[0].ID)
				assert.Equal(t, "function", calls[0].Type)
				assert.Equal(t, "read_file", calls[0].Function.Name)
				
				var args map[string]interface{}
				err := json.Unmarshal(calls[0].Function.Arguments, &args)
				require.NoError(t, err)
				assert.Equal(t, "/test.txt", args["path"])
			},
		},
		{
			name: "Anthropic XML format",
			response: `I'll help you read that file.

<function_calls>
	<invoke name="read_file">
		<parameter name="path">/test.txt</parameter>
	</invoke>
</function_calls>`,
			wantLen: 1,
			validate: func(t *testing.T, calls []ToolCall) {
				assert.Equal(t, "function", calls[0].Type)
				assert.Equal(t, "read_file", calls[0].Function.Name)
				
				var args map[string]interface{}
				err := json.Unmarshal(calls[0].Function.Arguments, &args)
				require.NoError(t, err)
				assert.Equal(t, "/test.txt", args["path"])
			},
		},
		{
			name: "Multiple Anthropic tool calls",
			response: `Let me check the file and then run the tests.

<function_calls>
	<invoke name="read_file">
		<parameter name="path">/src/main.go</parameter>
	</invoke>
	<invoke name="run_tests">
		<parameter name="path">/src</parameter>
		<parameter name="verbose">true</parameter>
	</invoke>
</function_calls>`,
			wantLen: 2,
			validate: func(t *testing.T, calls []ToolCall) {
				assert.Equal(t, "read_file", calls[0].Function.Name)
				assert.Equal(t, "run_tests", calls[1].Function.Name)
				
				// Check second call parameters
				var args map[string]interface{}
				err := json.Unmarshal(calls[1].Function.Arguments, &args)
				require.NoError(t, err)
				assert.Equal(t, "/src", args["path"])
				assert.Equal(t, "true", args["verbose"])
			},
		},
		{
			name:     "No tool calls",
			response: "I can help you with that task. Let me explain how to do it manually.",
			wantLen:  0,
		},
		{
			name: "Mixed content without valid tool calls",
			response: `Sure, I'll read that file for you. The tool call is:
			
			"tool_calls": [{"id": "tc_001", "type": "function", "function": {"name": "read_file", "arguments": "{\"path\": \"README.md\"}"}}]
			
			This will read the README file.`,
			wantLen: 0, // This is not valid JSON, so no tool calls should be extracted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.ExtractToolCalls(tt.response)
			
			// For mixed content, we expect no tool calls (nil response)
			if tt.name == "Mixed content without valid tool calls" {
				// The parser should return nil, nil when no valid tool calls are found
				assert.Nil(t, err)
				assert.Nil(t, calls)
				return
			}
			
			require.NoError(t, err)
			
			if tt.wantLen == 0 {
				assert.Nil(t, calls)
			} else {
				require.Len(t, calls, tt.wantLen)
				if tt.validate != nil {
					tt.validate(t, calls)
				}
			}
		})
	}
}

func TestResponseParser_ContainsToolCalls(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "Contains tool_calls",
			response: `{"tool_calls": []}`,
			want:     true,
		},
		{
			name:     "Contains function_call",
			response: `The function_call is ready`,
			want:     true,
		},
		{
			name:     "Contains Anthropic format",
			response: `<function_calls>test</function_calls>`,
			want:     true,
		},
		{
			name:     "Contains invoke tag",
			response: `<invoke name="test">`,
			want:     true,
		},
		{
			name:     "No tool calls",
			response: `Just a regular response without any tool references`,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.ContainsToolCalls(tt.response)
			assert.Equal(t, tt.want, got)
		})
	}
}

