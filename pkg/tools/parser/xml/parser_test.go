// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package xml

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/tools/parser/types"
)

func TestXMLParser_Parse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		input     string
		wantCalls []types.ToolCall
		wantErr   bool
	}{
		{
			name: "single function call",
			input: `<function_calls>
				<invoke name="search">
					<parameter name="query">golang testing</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "search",
					},
				},
			},
		},
		{
			name: "multiple function calls",
			input: `<function_calls>
				<invoke name="search">
					<parameter name="query">golang</parameter>
					<parameter name="limit">10</parameter>
				</invoke>
				<invoke name="write_file">
					<parameter name="path">/tmp/test.txt</parameter>
					<parameter name="content">hello world</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "search",
					},
				},
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "write_file",
					},
				},
			},
		},
		{
			name: "empty parameters",
			input: `<function_calls>
				<invoke name="get_time">
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "get_time",
					},
				},
			},
		},
		{
			name: "JSON value in parameter",
			input: `<function_calls>
				<invoke name="process">
					<parameter name="config">{"key": "value", "num": 42}</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "process",
					},
				},
			},
		},
		{
			name: "XML from code block",
			input: "Here's the function call:\n\n```xml\n" + `<function_calls>
				<invoke name="analyze">
					<parameter name="type">comprehensive</parameter>
				</invoke>
			</function_calls>` + "\n```\n\nThat should work.",
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "analyze",
					},
				},
			},
		},
		{
			name: "alternative tool tag",
			input: `<function_calls>
				<tool name="calculator">
					<param name="x">10</param>
					<param name="y">20</param>
				</tool>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "calculator",
					},
				},
			},
		},
		{
			name: "special characters in parameters",
			input: `<function_calls>
				<invoke name="echo">
					<parameter name="message">&lt;Hello &amp; "World"&gt;</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{
				{
					Type: "function",
					Function: types.FunctionCall{
						Name: "echo",
					},
				},
			},
		},
		{
			name: "malformed XML",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="broken"
				</invoke>
			</function_calls>`,
			wantErr: true,
		},
		{
			name:    "not XML at all",
			input:   "This is just plain text with no XML content",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name: "missing name attribute",
			input: `<function_calls>
				<invoke>
					<parameter name="test">value</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: []types.ToolCall{}, // Should skip invalid calls
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
				assert.Equal(t, want.Type, got.Type)
				assert.Equal(t, want.Function.Name, got.Function.Name)
				assert.NotEmpty(t, got.ID) // ID should be generated

				// Check that arguments were converted to JSON
				if len(got.Function.Arguments) > 0 {
					var args map[string]interface{}
					err := json.Unmarshal(got.Function.Arguments, &args)
					assert.NoError(t, err, "Arguments should be valid JSON")
				}
			}
		})
	}
}

func TestXMLParser_ParameterHandling(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	tests := []struct {
		name       string
		input      string
		wantParams map[string]interface{}
	}{
		{
			name: "string parameters",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="str1">hello</parameter>
					<parameter name="str2">world</parameter>
				</invoke>
			</function_calls>`,
			wantParams: map[string]interface{}{
				"str1": "hello",
				"str2": "world",
			},
		},
		{
			name: "JSON parameters",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="num">42</parameter>
					<parameter name="bool">true</parameter>
					<parameter name="array">[1, 2, 3]</parameter>
					<parameter name="obj">{"key": "value"}</parameter>
				</invoke>
			</function_calls>`,
			wantParams: map[string]interface{}{
				"num":   float64(42),
				"bool":  true,
				"array": []interface{}{float64(1), float64(2), float64(3)},
				"obj":   map[string]interface{}{"key": "value"},
			},
		},
		{
			name: "empty parameter values",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="empty"></parameter>
					<parameter name="whitespace">   </parameter>
				</invoke>
			</function_calls>`,
			wantParams: map[string]interface{}{
				"empty":      "",
				"whitespace": "",
			},
		},
		{
			name: "parameter with CDATA",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="cdata"><![CDATA[<html>content</html>]]></parameter>
				</invoke>
			</function_calls>`,
			wantParams: map[string]interface{}{
				"cdata": "<html>content</html>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.Parse(ctx, []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, calls, 1)

			// Parse arguments
			var gotParams map[string]interface{}
			err = json.Unmarshal(calls[0].Function.Arguments, &gotParams)
			require.NoError(t, err)

			assert.Equal(t, tt.wantParams, gotParams)
		})
	}
}

func TestXMLParser_ContextCancellation(t *testing.T) {
	parser := NewParser()

	// Create a large input that would take time to process
	largeInput := `<function_calls>`
	for i := 0; i < 1000; i++ {
		largeInput += `<invoke name="test` + string(rune(i)) + `"><parameter name="arg">value</parameter></invoke>`
	}
	largeInput += `</function_calls>`

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := parser.Parse(ctx, []byte(largeInput))
	assert.Error(t, err)
	// Note: XML parsing might not always respect context cancellation immediately
}

func TestXMLParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		input      string
		wantValid  bool
		wantErrors int
	}{
		{
			name: "valid XML",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="arg">value</parameter>
				</invoke>
			</function_calls>`,
			wantValid: true,
		},
		{
			name:       "missing root element",
			input:      `<invoke name="test"><parameter name="arg">value</parameter></invoke>`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "missing closing tag",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="arg">value</parameter>
				</invoke>`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "missing name attribute",
			input: `<function_calls>
				<invoke>
					<parameter name="arg">value</parameter>
				</invoke>
			</function_calls>`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "parameter without name",
			input: `<function_calls>
				<invoke name="test">
					<parameter>value</parameter>
				</invoke>
			</function_calls>`,
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name:       "not XML",
			input:      "plain text",
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "empty invoke",
			input: `<function_calls>
				<invoke name="test"></invoke>
			</function_calls>`,
			wantValid: true, // Empty parameters are valid
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

func TestXMLParser_EdgeCases(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     string
		wantCalls int
	}{
		{
			name: "unicode in function name",
			input: `<function_calls>
				<invoke name="测试_function">
					<parameter name="参数">值</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: 1,
		},
		{
			name: "very long parameter value",
			input: `<function_calls>
				<invoke name="process">
					<parameter name="data">` + strings.Repeat("x", 10000) + `</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: 1,
		},
		{
			name: "nested XML entities",
			input: `<function_calls>
				<invoke name="echo">
					<parameter name="msg">&lt;tag&gt;&amp;nbsp;&lt;/tag&gt;</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: 1,
		},
		{
			name: "multiple parameters same name",
			input: `<function_calls>
				<invoke name="multi">
					<parameter name="item">first</parameter>
					<parameter name="item">second</parameter>
					<parameter name="item">third</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: 1, // Should handle gracefully
		},
		{
			name: "attributes in parameter tag",
			input: `<function_calls>
				<invoke name="test">
					<parameter name="config" type="json">{"key": "value"}</parameter>
				</invoke>
			</function_calls>`,
			wantCalls: 1, // Should ignore extra attributes
		},
		{
			name: "namespace prefixes",
			input: `<ns:function_calls xmlns:ns="http://example.com">
				<ns:invoke name="test">
					<ns:parameter name="arg">value</ns:parameter>
				</ns:invoke>
			</ns:function_calls>`,
			wantCalls: 1, // Parser handles namespaces correctly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.Parse(ctx, []byte(tt.input))
			if tt.wantCalls > 0 {
				require.NoError(t, err)
			}
			assert.Len(t, calls, tt.wantCalls)
		})
	}
}

func TestXMLParser_StreamingMode(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	// Test that streaming parser handles partial/malformed XML gracefully
	input := `<function_calls>
		<invoke name="test1">
			<parameter name="arg1">value1</parameter>
		</invoke>
		<invoke name="test2">
			<parameter name="arg2">value2</parameter>
			<!-- Incomplete XML below -->
			<parameter name="broken"`

	calls, err := parser.Parse(ctx, []byte(input))
	// Should parse what it can
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(calls), 1) // At least first invoke should parse
	if len(calls) > 0 {
		assert.Equal(t, "test1", calls[0].Function.Name)
	}
}

func TestXMLParser_PerformanceRegression(t *testing.T) {
	parser := NewParser()
	ctx := context.Background()

	// Test with a reasonably complex input
	input := `<function_calls>
		<invoke name="analyze_data">
			<parameter name="data">[1,2,3,4,5,6,7,8,9,10]</parameter>
			<parameter name="method">statistical</parameter>
			<parameter name="options">{"verbose": true, "format": "json"}</parameter>
		</invoke>
	</function_calls>`

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
