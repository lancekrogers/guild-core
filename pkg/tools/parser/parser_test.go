// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRobustParser tests the main parser with format detection
func TestRobustParser(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name           string
		input          string
		wantFormat     ProviderFormat
		wantCalls      int
		wantConfidence float64
	}{
		{
			name:           "auto-detect JSON",
			input:          `{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test_func", "arguments": "{}"}}]}`,
			wantFormat:     ProviderFormatOpenAI,
			wantCalls:      1,
			wantConfidence: 0.85,
		},
		{
			name:           "auto-detect XML",
			input:          `<function_calls><invoke name="test_func"><parameter name="arg">value</parameter></invoke></function_calls>`,
			wantFormat:     ProviderFormatAnthropic,
			wantCalls:      1,
			wantConfidence: 0.85,
		},
		{
			name: "JSON in mixed content",
			input: `Here's my response:

{"tool_calls": [{"id": "mixed", "type": "function", "function": {"name": "analyze", "arguments": "{}"}}]}

Processing...`,
			wantFormat:     ProviderFormatOpenAI,
			wantCalls:      1,
			wantConfidence: 0.7,
		},
		{
			name:           "no tool calls",
			input:          "This is just a regular message with no tool calls.",
			wantFormat:     ProviderFormatUnknown,
			wantCalls:      0,
			wantConfidence: 0,
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
				assert.InDelta(t, tt.wantConfidence, confidence, 0.25)
			}

			// Test extraction
			calls, err := parser.ExtractToolCalls(tt.input)
			require.NoError(t, err) // No error even if no calls found
			assert.Len(t, calls, tt.wantCalls)
		})
	}
}

// TestParserRobustness tests edge cases and error handling
func TestParserRobustness(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "empty input",
			input:     "",
			shouldErr: false, // Empty is not an error
		},
		{
			name:      "whitespace only",
			input:     "   \n\t  ",
			shouldErr: false,
		},
		{
			name:      "truncated JSON",
			input:     `{"tool_calls": [{"id": "test", "type": "func`,
			shouldErr: false, // No calls found, but not an error
		},
		{
			name:      "invalid characters",
			input:     string([]byte{0xFF, 0xFE, 0xFD}),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parser.ExtractToolCalls(tt.input)

			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				// Not finding tool calls is not an error
				assert.NoError(t, err)
				if err == nil {
					assert.NotNil(t, calls) // Should return empty slice, not nil
				}
			}
		})
	}
}

// TestParserOptions tests parser configuration options
func TestParserOptions(t *testing.T) {
	// Test with strict validation
	strictParser := NewResponseParser(WithStrictValidation(true))
	assert.NotNil(t, strictParser)

	// Test with custom timeout
	timeoutParser := NewResponseParser(WithTimeout(1000))
	assert.NotNil(t, timeoutParser)

	// Test with max input size
	sizeParser := NewResponseParser(WithMaxInputSize(1024))
	assert.NotNil(t, sizeParser)
}
