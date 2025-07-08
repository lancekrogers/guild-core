// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractReasoning(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedContent    string
		expectedReasoning  string
		expectedConfidence float64
	}{
		{
			name: "extract thinking with confidence",
			input: `Let me think about this.
<thinking>
The user wants a REST API.
I should suggest a framework.
Confidence: 0.8
</thinking>
Here's how to implement a REST API...`,
			expectedContent:    "Let me think about this.\n\nHere's how to implement a REST API...",
			expectedReasoning:  "The user wants a REST API.\nI should suggest a framework.\nConfidence: 0.8",
			expectedConfidence: 0.8,
		},
		{
			name: "multiple thinking blocks",
			input: `Starting analysis.
<thinking>
First consideration.
</thinking>
Middle content.
<thinking>
Second consideration.
Confidence: 0.9
</thinking>
Final response.`,
			expectedContent:    "Starting analysis.\n\nMiddle content.\n\nFinal response.",
			expectedReasoning:  "First consideration.\n\nSecond consideration.\nConfidence: 0.9",
			expectedConfidence: 0.9,
		},
		{
			name:               "no thinking blocks",
			input:              `This is a simple response without any thinking blocks.`,
			expectedContent:    "This is a simple response without any thinking blocks.",
			expectedReasoning:  "",
			expectedConfidence: 0.5,
		},
		{
			name: "thinking without confidence",
			input: `<thinking>
Analyzing the request...
This seems complex.
</thinking>
The answer is 42.`,
			expectedContent:    "The answer is 42.",
			expectedReasoning:  "Analyzing the request...\nThis seems complex.",
			expectedConfidence: 0.5,
		},
		{
			name:               "empty thinking block",
			input:              `<thinking></thinking>Response text.`,
			expectedContent:    "Response text.",
			expectedReasoning:  "",
			expectedConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, reasoning, confidence := ExtractReasoning(tt.input)
			assert.Equal(t, tt.expectedContent, content)
			assert.Equal(t, tt.expectedReasoning, reasoning)
			assert.Equal(t, tt.expectedConfidence, confidence)
		})
	}
}
