// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCase represents a parser test case
type TestCase struct {
	Name           string
	Input          string
	WantFormat     ProviderFormat
	WantCalls      []ToolCall
	WantError      bool
	WantConfidence float64
}

// RunParserTest runs a standard parser test case
func RunParserTest(t *testing.T, parser ResponseParser, tc TestCase) {
	t.Helper()

	// Test format detection if format is specified
	if tc.WantFormat != "" {
		format, confidence, err := parser.DetectFormat(tc.Input)
		if tc.WantFormat == ProviderFormatUnknown {
			assert.Error(t, err, "Expected error for unknown format")
		} else {
			require.NoError(t, err, "Format detection failed")
			assert.Equal(t, tc.WantFormat, format, "Wrong format detected")
			if tc.WantConfidence > 0 {
				assert.InDelta(t, tc.WantConfidence, confidence, 0.1, "Confidence mismatch")
			}
		}
	}

	// Test extraction
	calls, err := parser.ExtractToolCalls(tc.Input)

	if tc.WantError {
		assert.Error(t, err, "Expected error but got none")
		return
	}

	require.NoError(t, err, "Extraction failed")
	assert.Len(t, calls, len(tc.WantCalls), "Wrong number of calls extracted")

	// Compare calls
	for i, want := range tc.WantCalls {
		if i >= len(calls) {
			break
		}
		got := calls[i]

		if want.ID != "" {
			assert.Equal(t, want.ID, got.ID, "ID mismatch at call %d", i)
		}
		if want.Type != "" {
			assert.Equal(t, want.Type, got.Type, "Type mismatch at call %d", i)
		}
		assert.Equal(t, want.Function.Name, got.Function.Name, "Function name mismatch at call %d", i)

		// Compare arguments if specified
		if len(want.Function.Arguments) > 0 {
			CompareArguments(t, want.Function.Arguments, got.Function.Arguments)
		}
	}
}

// CompareArguments compares JSON arguments
func CompareArguments(t *testing.T, want, got json.RawMessage) {
	t.Helper()

	var wantData, gotData interface{}

	err := json.Unmarshal(want, &wantData)
	require.NoError(t, err, "Failed to unmarshal expected arguments")

	err = json.Unmarshal(got, &gotData)
	require.NoError(t, err, "Failed to unmarshal actual arguments")

	assert.Equal(t, wantData, gotData, "Arguments mismatch")
}

// GenerateJSONToolCall generates a JSON tool call for testing
func GenerateJSONToolCall(id, funcName string, args map[string]interface{}) string {
	argsJSON, _ := json.Marshal(args)
	return fmt.Sprintf(`{
		"id": "%s",
		"type": "function",
		"function": {
			"name": "%s",
			"arguments": %q
		}
	}`, id, funcName, string(argsJSON))
}

// GenerateJSONToolCalls generates multiple JSON tool calls
func GenerateJSONToolCalls(calls []map[string]interface{}) string {
	var jsonCalls []string
	for _, call := range calls {
		id := call["id"].(string)
		name := call["name"].(string)
		args := call["args"].(map[string]interface{})
		jsonCalls = append(jsonCalls, GenerateJSONToolCall(id, name, args))
	}
	return fmt.Sprintf(`{"tool_calls": [%s]}`, strings.Join(jsonCalls, ","))
}

// GenerateXMLToolCall generates an XML tool call for testing
func GenerateXMLToolCall(funcName string, params map[string]string) string {
	var paramXML []string
	for name, value := range params {
		paramXML = append(paramXML, fmt.Sprintf(`<parameter name="%s">%s</parameter>`, name, value))
	}
	return fmt.Sprintf(`<invoke name="%s">%s</invoke>`, funcName, strings.Join(paramXML, "\n"))
}

// GenerateXMLToolCalls generates multiple XML tool calls
func GenerateXMLToolCalls(calls []map[string]interface{}) string {
	var xmlCalls []string
	for _, call := range calls {
		name := call["name"].(string)
		params := call["params"].(map[string]string)
		xmlCalls = append(xmlCalls, GenerateXMLToolCall(name, params))
	}
	return fmt.Sprintf(`<function_calls>%s</function_calls>`, strings.Join(xmlCalls, "\n"))
}

// WrapInContext wraps tool calls in conversational context
func WrapInContext(toolCalls string, prefix, suffix string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s", prefix, toolCalls, suffix)
}

// WrapInCodeBlock wraps content in a code block
func WrapInCodeBlock(content, language string) string {
	return fmt.Sprintf("```%s\n%s\n```", language, content)
}

// CreateMockParser creates a mock parser for testing
type MockParser struct {
	ExtractFunc      func(string) ([]ToolCall, error)
	DetectFormatFunc func(string) (ProviderFormat, float64, error)
}

func (m *MockParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(response)
	}
	return nil, nil
}

func (m *MockParser) ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	return m.ExtractToolCalls(response)
}

func (m *MockParser) DetectFormat(response string) (ProviderFormat, float64, error) {
	if m.DetectFormatFunc != nil {
		return m.DetectFormatFunc(response)
	}
	return ProviderFormatUnknown, 0, fmt.Errorf("no format detected")
}

// AssertToolCallsEqual asserts that two slices of tool calls are equal
func AssertToolCallsEqual(t *testing.T, expected, actual []ToolCall) {
	t.Helper()

	require.Len(t, actual, len(expected), "Different number of tool calls")

	for i := range expected {
		assert.Equal(t, expected[i].Type, actual[i].Type, "Type mismatch at index %d", i)
		assert.Equal(t, expected[i].Function.Name, actual[i].Function.Name, "Function name mismatch at index %d", i)

		// Compare arguments as JSON
		var expectedArgs, actualArgs interface{}
		json.Unmarshal(expected[i].Function.Arguments, &expectedArgs)
		json.Unmarshal(actual[i].Function.Arguments, &actualArgs)
		assert.Equal(t, expectedArgs, actualArgs, "Arguments mismatch at index %d", i)
	}
}

// CreateTestRegistry creates a test prometheus registry
func CreateTestRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// Common test inputs for reuse
var (
	ValidJSONSingle = `{"id": "test1", "type": "function", "function": {"name": "test_func", "arguments": "{}"}}`

	ValidJSONMultiple = `{"tool_calls": [
		{"id": "call1", "type": "function", "function": {"name": "func1", "arguments": "{}"}},
		{"id": "call2", "type": "function", "function": {"name": "func2", "arguments": "{\"key\": \"value\"}"}}
	]}`

	ValidXMLSingle = `<function_calls>
		<invoke name="test_func">
			<parameter name="arg1">value1</parameter>
		</invoke>
	</function_calls>`

	ValidXMLMultiple = `<function_calls>
		<invoke name="func1">
			<parameter name="arg1">value1</parameter>
		</invoke>
		<invoke name="func2">
			<parameter name="arg2">value2</parameter>
		</invoke>
	</function_calls>`

	InvalidJSON = `{"broken": json}`
	InvalidXML  = `<broken>`

	MixedContentJSON = `Let me help you with that.
	
	` + ValidJSONSingle + `
	
	The function has been called.`

	MixedContentXML = `I'll process your request.
	
	` + ValidXMLSingle + `
	
	Processing complete.`
)
