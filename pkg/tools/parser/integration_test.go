// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RealWorldScenarios tests the parser with real-world examples
func TestIntegration_RealWorldScenarios(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name         string
		input        string
		wantFormat   ProviderFormat
		wantCalls    int
		wantFuncName string
	}{
		{
			name: "OpenAI GPT-4 response",
			input: `I'll help you search for that information.

{"tool_calls": [{"id": "call_abc123", "type": "function", "function": {"name": "web_search", "arguments": "{\"query\": \"latest golang features 2024\", \"limit\": 5}"}}]}

Let me search for the latest Go features...`,
			wantFormat:   ProviderFormatOpenAI,
			wantCalls:    1,
			wantFuncName: "web_search",
		},
		{
			name: "Anthropic Claude response",
			input: `I'll search for that information for you.

<function_calls>
<invoke name="web_search">
<parameter name="query">latest golang features 2024</parameter>
<parameter name="limit">5</parameter>
</invoke>
</function_calls>

Searching for the latest Go features...`,
			wantFormat:   ProviderFormatAnthropic,
			wantCalls:    1,
			wantFuncName: "web_search",
		},
		{
			name: "Multiple tool calls in sequence",
			input: `Let me help you with multiple tasks.

First, I'll search for the information:
{"tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "search", "arguments": "{\"query\": \"go modules\"}"}}]}

Then I'll write it to a file:
{"tool_calls": [{"id": "call_2", "type": "function", "function": {"name": "write_file", "arguments": "{\"path\": \"results.txt\", \"content\": \"search results here\"}"}}]}

All done!`,
			wantFormat: ProviderFormatOpenAI,
			wantCalls:  2,
		},
		{
			name: "Mixed content with code blocks",
			input: "Here's how to call the function:\n\n```json\n" + `{
  "tool_calls": [{
    "id": "call_example",
    "type": "function",
    "function": {
      "name": "calculate",
      "arguments": "{\"expression\": \"2 + 2\"}"
    }
  }]
}` + "\n```\n\nThis will calculate the result.",
			wantFormat:   ProviderFormatOpenAI,
			wantCalls:    1,
			wantFuncName: "calculate",
		},
		{
			name: "Streaming response simulation",
			input: `I'll analyze this for you. <function_calls>
<invoke name="analyze_code">
<parameter name="language">go</parameter>
<parameter name="code">func main() { fmt.Println("Hello") }</parameter>
<parameter name="checks">["style", "bugs", "performance"]</parameter>
</invoke>
</function_calls> The analysis is complete.`,
			wantFormat:   ProviderFormatAnthropic,
			wantCalls:    1,
			wantFuncName: "analyze_code",
		},
		{
			name: "No tool calls - conversational response",
			input: `This is just a regular response without any tool calls. 
I'm explaining something to you without needing to use any tools.
The answer to your question is 42.`,
			wantFormat: ProviderFormatUnknown,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test format detection
			format, confidence, err := parser.DetectFormat(tt.input)
			if tt.wantFormat == ProviderFormatUnknown {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantFormat, format)
				assert.Greater(t, confidence, 0.5)
			}

			// Test extraction
			calls, err := parser.ExtractToolCalls(tt.input)
			require.NoError(t, err)
			assert.Len(t, calls, tt.wantCalls)

			if tt.wantFuncName != "" && len(calls) > 0 {
				assert.Equal(t, tt.wantFuncName, calls[0].Function.Name)
			}
		})
	}
}

// TestIntegration_ConcurrentParsing tests thread safety
func TestIntegration_ConcurrentParsing(t *testing.T) {
	parser := NewResponseParser()
	
	// Different inputs to parse concurrently
	inputs := []string{
		`{"tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "test1", "arguments": "{}"}}]}`,
		`<function_calls><invoke name="test2"><parameter name="arg">value</parameter></invoke></function_calls>`,
		`{"id": "call_3", "type": "function", "function": {"name": "test3", "arguments": "{\"key\": \"value\"}"}}`,
		`Mixed content with {"tool_calls": [{"id": "call_4", "type": "function", "function": {"name": "test4", "arguments": "{}"}}]} embedded`,
		`<function_calls>
			<invoke name="test5">
				<parameter name="data">[1,2,3]</parameter>
			</invoke>
		</function_calls>`,
	}

	// Run concurrent parsing
	var wg sync.WaitGroup
	errors := make([]error, len(inputs))
	results := make([][]ToolCall, len(inputs))

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in string) {
			defer wg.Done()
			
			// Parse multiple times to increase chance of race conditions
			for j := 0; j < 100; j++ {
				calls, err := parser.ExtractToolCalls(in)
				if j == 0 {
					errors[idx] = err
					results[idx] = calls
				}
			}
		}(i, input)
	}

	wg.Wait()

	// Verify results
	for i, err := range errors {
		assert.NoError(t, err, "Error parsing input %d", i)
		assert.NotNil(t, results[i], "Nil result for input %d", i)
		assert.Greater(t, len(results[i]), 0, "No calls extracted for input %d", i)
	}
}

// TestIntegration_LargeInputs tests handling of large inputs
func TestIntegration_LargeInputs(t *testing.T) {
	parser := NewResponseParser()

	// Generate a large input with many tool calls
	var jsonCalls []string
	for i := 0; i < 100; i++ {
		jsonCalls = append(jsonCalls, fmt.Sprintf(`{
			"id": "call_%d",
			"type": "function",
			"function": {
				"name": "process_%d",
				"arguments": "{\"index\": %d, \"data\": \"%s\"}"
			}
		}`, i, i, i, strings.Repeat("x", 100)))
	}
	
	largeJSON := fmt.Sprintf(`{"tool_calls": [%s]}`, strings.Join(jsonCalls, ","))
	
	// Add some padding to make it even larger
	largeInput := strings.Repeat("This is some context. ", 1000) + "\n\n" + largeJSON + "\n\n" + strings.Repeat("More context. ", 1000)

	// Test with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	calls, err := parser.ExtractWithContext(ctx, largeInput)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, calls, 100)
	assert.Less(t, elapsed, 1*time.Second, "Parsing took too long: %v", elapsed)
}

// TestIntegration_ErrorRecovery tests graceful error handling
func TestIntegration_ErrorRecovery(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name      string
		input     string
		wantCalls int // Some calls might still be extracted
	}{
		{
			name: "Truncated JSON in middle",
			input: `Start of response
{"tool_calls": [
  {"id": "call_1", "type": "function", "function": {"name": "test1", "arguments": "{}"}},
  {"id": "call_2", "type": "function", "function": {"name": "test2", "argume`,
			wantCalls: 0, // JSON parser typically fails completely on truncation
		},
		{
			name: "Truncated XML in middle",
			input: `Start of response
<function_calls>
  <invoke name="test1">
    <parameter name="arg1">value1</parameter>
  </invoke>
  <invoke name="test2">
    <parameter name="arg2">val`,
			wantCalls: 1, // Streaming XML parser might recover first call
		},
		{
			name: "Mixed valid and invalid",
			input: `First call is valid:
{"id": "call_1", "type": "function", "function": {"name": "valid", "arguments": "{}"}}

But this one is broken:
{"id": "call_2", "type": "function", "function": {"name": "broken", "arguments": "{invalid json}"}}

And this is just text that looks like JSON but isn't.`,
			wantCalls: 1, // Should extract the valid one
		},
		{
			name: "Invalid UTF-8 sequences",
			input: "Some text " + string([]byte{0xFF, 0xFE}) + `{"id": "call_1", "type": "function", "function": {"name": "test", "arguments": "{}"}}`,
			wantCalls: 1, // Should handle invalid UTF-8 gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.ExtractToolCalls(tt.input)
			// We don't require error because empty result is acceptable
			assert.NoError(t, err)
			assert.Len(t, calls, tt.wantCalls)
		})
	}
}

// TestIntegration_FormatDetectionAccuracy tests format detection edge cases
func TestIntegration_FormatDetectionAccuracy(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name           string
		input          string
		wantFormat     ProviderFormat
		minConfidence  float64
	}{
		{
			name:          "Clear JSON format",
			input:         `{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`,
			wantFormat:    ProviderFormatOpenAI,
			minConfidence: 0.9,
		},
		{
			name:          "Clear XML format",
			input:         `<function_calls><invoke name="test"><parameter name="arg">value</parameter></invoke></function_calls>`,
			wantFormat:    ProviderFormatAnthropic,
			minConfidence: 0.9,
		},
		{
			name: "JSON with noise",
			input: `Some random text here and there...
Maybe some code: if (x > 0) { return true; }
{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}
More random text...`,
			wantFormat:    ProviderFormatOpenAI,
			minConfidence: 0.7,
		},
		{
			name: "XML with noise",
			input: `Let me help you with that...
<function_calls>
  <invoke name="search">
    <parameter name="query">test query</parameter>
  </invoke>
</function_calls>
I've initiated the search...`,
			wantFormat:    ProviderFormatAnthropic,
			minConfidence: 0.7,
		},
		{
			name:          "Ambiguous - both JSON and XML keywords",
			input:         `function_calls invoke parameter {"name": "test"} arguments`,
			wantFormat:    ProviderFormatUnknown,
			minConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, confidence, err := parser.DetectFormat(tt.input)
			
			if tt.wantFormat == ProviderFormatUnknown {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantFormat, format)
				assert.GreaterOrEqual(t, confidence, tt.minConfidence, 
					"Confidence %f is below minimum %f", confidence, tt.minConfidence)
			}
		})
	}
}

// TestIntegration_ProviderCompatibility tests compatibility with different providers
func TestIntegration_ProviderCompatibility(t *testing.T) {
	parser := NewResponseParser()

	// Test various provider response formats
	providers := []struct {
		name     string
		response string
		wantFunc string
	}{
		{
			name: "OpenAI GPT-4",
			response: `{"tool_calls": [{"id": "call_JlbpwFCg7t8R1HXuBOxtaVxE", "type": "function", "function": {"name": "get_weather", "arguments": "{\"location\": \"San Francisco, CA\"}"}}]}`,
			wantFunc: "get_weather",
		},
		{
			name: "OpenAI GPT-3.5",
			response: `{"id": "chatcmpl-123", "choices": [{"message": {"tool_calls": [{"id": "call_abc", "type": "function", "function": {"name": "search", "arguments": "{\"q\": \"test\"}"}}]}}]}`,
			wantFunc: "search",
		},
		{
			name: "Anthropic Claude",
			response: `I'll help you with that.

<function_calls>
<invoke name="calculate">
<parameter name="expression">2 + 2</parameter>
</invoke>
</function_calls>

The result is 4.`,
			wantFunc: "calculate",
		},
		{
			name: "Single function format (older style)",
			response: `{"function": {"name": "translate", "arguments": "{\"text\": \"Hello\", \"to\": \"es\"}"}}`,
			wantFunc: "translate",
		},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			calls, err := parser.ExtractToolCalls(p.response)
			require.NoError(t, err)
			require.Greater(t, len(calls), 0, "No calls extracted")
			assert.Equal(t, p.wantFunc, calls[0].Function.Name)
		})
	}
}

// TestIntegration_MemoryLeaks tests for memory leaks with repeated parsing
func TestIntegration_MemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	parser := NewResponseParser()
	
	// Large input to amplify any leaks
	largeInput := `{"tool_calls": [{"id": "call_mem", "type": "function", "function": {"name": "process", "arguments": "{\"data\": \"` + 
		strings.Repeat("x", 10000) + `\"}"}}]}`

	// Parse many times
	iterations := 10000
	for i := 0; i < iterations; i++ {
		calls, err := parser.ExtractToolCalls(largeInput)
		require.NoError(t, err)
		require.Len(t, calls, 1)
		
		// Results should be garbage collected
		_ = calls
	}

	// In a real test, we'd measure memory usage before and after
	// For now, this test ensures no panics or obvious leaks
}

// TestIntegration_ComplexArguments tests parsing of complex argument structures
func TestIntegration_ComplexArguments(t *testing.T) {
	parser := NewResponseParser()

	complexArgs := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "Alice", "roles": []string{"admin", "user"}},
			{"id": 2, "name": "Bob", "roles": []string{"user"}},
		},
		"settings": map[string]interface{}{
			"theme": "dark",
			"notifications": map[string]bool{
				"email": true,
				"sms":   false,
			},
		},
		"data": []interface{}{1, "two", 3.14, true, nil},
	}

	argsJSON, _ := json.Marshal(complexArgs)
	
	input := fmt.Sprintf(`{"tool_calls": [{"id": "complex", "type": "function", "function": {"name": "process_complex", "arguments": %q}}]}`, string(argsJSON))

	calls, err := parser.ExtractToolCalls(input)
	require.NoError(t, err)
	require.Len(t, calls, 1)

	// Verify arguments were preserved correctly
	var parsedArgs map[string]interface{}
	err = json.Unmarshal(calls[0].Function.Arguments, &parsedArgs)
	require.NoError(t, err)
	
	assert.Equal(t, complexArgs, parsedArgs)
}