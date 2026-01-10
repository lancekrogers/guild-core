// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package json

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/tools/parser/types"
)

func TestJSONParser_Parse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		input     string
		wantCalls []types.ToolCall
		wantErr   bool
	}{
		{
			name: "single function call",
			input: `{
				"id": "call_123",
				"type": "function",
				"function": {
					"name": "search",
					"arguments": "{\"query\": \"test\"}"
				}
			}`,
			wantCalls: []types.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "search",
						Arguments: json.RawMessage(`{"query": "test"}`),
					},
				},
			},
		},
		{
			name: "multiple function calls",
			input: `{
				"tool_calls": [
					{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "search",
							"arguments": "{\"query\": \"golang\"}"
						}
					},
					{
						"id": "call_2",
						"type": "function",
						"function": {
							"name": "write_file",
							"arguments": "{\"path\": \"/tmp/test.txt\", \"content\": \"hello\"}"
						}
					}
				]
			}`,
			wantCalls: []types.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "search",
						Arguments: json.RawMessage(`{"query": "golang"}`),
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "write_file",
						Arguments: json.RawMessage(`{"path": "/tmp/test.txt", "content": "hello"}`),
					},
				},
			},
		},
		{
			name: "arguments as object",
			input: `{
				"id": "call_obj",
				"type": "function",
				"function": {
					"name": "calculate",
					"arguments": {"x": 10, "y": 20, "operation": "add"}
				}
			}`,
			wantCalls: []types.ToolCall{
				{
					ID:   "call_obj",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "calculate",
						Arguments: json.RawMessage(`{"x":10,"y":20,"operation":"add"}`),
					},
				},
			},
		},
		{
			name: "empty arguments",
			input: `{
				"id": "call_empty",
				"type": "function",
				"function": {
					"name": "get_time",
					"arguments": ""
				}
			}`,
			wantCalls: []types.ToolCall{
				{
					ID:   "call_empty",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "get_time",
						Arguments: json.RawMessage(`{}`),
					},
				},
			},
		},
		{
			name: "null arguments",
			input: `{
				"id": "call_null",
				"type": "function",
				"function": {
					"name": "clear_cache",
					"arguments": null
				}
			}`,
			wantCalls: []types.ToolCall{
				{
					ID:   "call_null",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "clear_cache",
						Arguments: json.RawMessage(`{}`),
					},
				},
			},
		},
		{
			name: "JSON from code block",
			input: "Here's the function call:\n\n```json\n" + `{
				"id": "call_block",
				"type": "function",
				"function": {
					"name": "analyze",
					"arguments": "{}"
				}
			}` + "\n```\n\nThat should work.",
			wantCalls: []types.ToolCall{
				{
					ID:   "call_block",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "analyze",
						Arguments: json.RawMessage(`{}`),
					},
				},
			},
		},
		{
			name: "malformed JSON",
			input: `{
				"id": "call_bad",
				"type": "function",
				"function": {
					"name": "test",
					"arguments": "{"broken": json}"
				}
			}`,
			wantErr: true,
		},
		{
			name:    "not JSON at all",
			input:   "This is just plain text with no JSON content",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			calls, err := parser.Parse(ctx, []byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, calls, len(tt.wantCalls))

			for i, want := range tt.wantCalls {
				got := calls[i]
				assert.Equal(t, want.ID, got.ID)
				assert.Equal(t, want.Type, got.Type)
				assert.Equal(t, want.Function.Name, got.Function.Name)

				// Compare arguments as JSON
				var wantArgs, gotArgs interface{}
				json.Unmarshal(want.Function.Arguments, &wantArgs)
				json.Unmarshal(got.Function.Arguments, &gotArgs)
				assert.Equal(t, wantArgs, gotArgs)
			}
		})
	}
}

func TestJSONParser_ContextCancellation(t *testing.T) {
	parser := NewParser()

	// Create a large input that would take time to process
	largeInput := `{"tool_calls": [`
	for i := 0; i < 1000; i++ {
		if i > 0 {
			largeInput += ","
		}
		largeInput += `{"id": "call_` + string(rune(i)) + `", "type": "function", "function": {"name": "test", "arguments": "{}"}}`
	}
	largeInput += `]}`

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := parser.Parse(ctx, []byte(largeInput))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestJSONParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		input      string
		wantValid  bool
		wantErrors int
	}{
		{
			name: "valid single call",
			input: `{
				"id": "call_123",
				"type": "function",
				"function": {
					"name": "search",
					"arguments": "{\"query\": \"test\"}"
				}
			}`,
			wantValid: true,
		},
		{
			name: "valid array format",
			input: `{
				"tool_calls": [{
					"id": "call_123",
					"type": "function",
					"function": {
						"name": "search",
						"arguments": "{\"query\": \"test\"}"
					}
				}]
			}`,
			wantValid: true,
		},
		{
			name: "missing function name",
			input: `{
				"id": "call_123",
				"type": "function",
				"function": {
					"arguments": "{}"
				}
			}`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "missing id",
			input: `{
				"type": "function",
				"function": {
					"name": "test",
					"arguments": "{}"
				}
			}`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "invalid arguments JSON",
			input: `{
				"id": "call_123",
				"type": "function",
				"function": {
					"name": "test",
					"arguments": "{broken json"
				}
			}`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name:       "not JSON",
			input:      "plain text",
			wantValid:  false,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Validate([]byte(tt.input))

			assert.Equal(t, tt.wantValid, result.Valid)
			if tt.wantErrors > 0 {
				assert.Len(t, result.Errors, tt.wantErrors)
			} else {
				assert.Empty(t, result.Errors)
			}

			// Schema should always be set
			assert.NotEmpty(t, result.SchemaUsed)
		})
	}
}

func TestJSONParser_EdgeCases(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     string
		wantCalls int
	}{
		{
			name: "escaped quotes in arguments",
			input: `{
				"id": "call_escape",
				"type": "function",
				"function": {
					"name": "echo",
					"arguments": "{\"message\": \"He said \\\"Hello\\\"\"}"
				}
			}`,
			wantCalls: 1,
		},
		{
			name: "unicode in function name",
			input: `{
				"id": "call_unicode",
				"type": "function",
				"function": {
					"name": "测试_function",
					"arguments": "{}"
				}
			}`,
			wantCalls: 1,
		},
		{
			name: "very long arguments",
			input: `{
				"id": "call_long",
				"type": "function",
				"function": {
					"name": "process",
					"arguments": "{\"data\": \"` + strings.Repeat("x", 10000) + `\"}"
				}
			}`,
			wantCalls: 1,
		},
		{
			name: "nested JSON in arguments",
			input: `{
				"id": "call_nested",
				"type": "function",
				"function": {
					"name": "complex",
					"arguments": "{\"config\": {\"nested\": {\"deep\": {\"value\": 42}}}}"
				}
			}`,
			wantCalls: 1,
		},
		{
			name: "array arguments",
			input: `{
				"id": "call_array",
				"type": "function",
				"function": {
					"name": "batch",
					"arguments": "[1, 2, 3, 4, 5]"
				}
			}`,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.Parse(ctx, []byte(tt.input))
			require.NoError(t, err)
			assert.Len(t, calls, tt.wantCalls)
		})
	}
}

func TestJSONParser_PerformanceRegression(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	// Test with a reasonably complex input
	input := `{
		"tool_calls": [
			{
				"id": "call_perf_1",
				"type": "function",
				"function": {
					"name": "analyze_data",
					"arguments": "{\"data\": [1,2,3,4,5,6,7,8,9,10], \"method\": \"statistical\"}"
				}
			}
		]
	}`

	// Warm up
	for i := 0; i < 10; i++ {
		_, _ = parser.Parse(ctx, []byte(input))
	}

	// Measure performance
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		calls, err := parser.Parse(ctx, []byte(input))
		require.NoError(t, err)
		require.Len(t, calls, 1)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Ensure parsing is fast enough (adjust threshold as needed)
	assert.Less(t, avgTime, 200*time.Microsecond, "Parsing is too slow: %v per operation", avgTime)
}
