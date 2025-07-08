// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package json

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetector tests the JSON format detector
func TestDetector(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	tests := []struct {
		name           string
		input          string
		wantConfidence float64
		wantDetected   bool
		description    string
	}{
		{
			name: "valid OpenAI tool calls",
			input: `{
				"tool_calls": [
					{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "test_function",
							"arguments": "{\"param\": \"value\"}"
						}
					}
				]
			}`,
			wantConfidence: 0.95,
			wantDetected:   true,
		},
		{
			name: "mixed content with JSON",
			input: `Here's the analysis you requested:
			
			{"tool_calls": [{"id": "call_456", "type": "function", "function": {"name": "analyze", "arguments": "{}"}}]}
			
			The results will be processed.`,
			wantConfidence: 0.85,
			wantDetected:   true,
		},
		{
			name:           "empty input",
			input:          "",
			wantConfidence: 0,
			wantDetected:   false,
		},
		{
			name:           "no tool calls",
			input:          `{"message": "Hello world"}`,
			wantConfidence: 0.1,
			wantDetected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := detector.Detect(ctx, []byte(tt.input))
			require.NoError(t, err)

			if tt.wantDetected {
				assert.Greater(t, result.Confidence, 0.5, "Expected detection but confidence too low")
				assert.InDelta(t, tt.wantConfidence, result.Confidence, 0.15, "Confidence mismatch")
			} else {
				assert.Less(t, result.Confidence, 0.5, "Should not detect with high confidence")
			}

			// Check metadata
			assert.NotNil(t, result.Metadata)
			if tt.input != "" {
				assert.Contains(t, result.Metadata, "original_size")
			}
		})
	}
}
