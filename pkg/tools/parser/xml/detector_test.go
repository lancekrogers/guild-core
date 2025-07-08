// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package xml

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetector tests the XML format detector
func TestDetector(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	tests := []struct {
		name           string
		input          string
		wantConfidence float64
		wantDetected   bool
	}{
		{
			name: "valid Anthropic XML",
			input: `<function_calls>
<invoke name="search_database">
<parameter name="query">test query</parameter>
<parameter name="limit">10</parameter>
</invoke>
</function_calls>`,
			wantConfidence: 0.85,
			wantDetected:   true,
		},
		{
			name: "XML in mixed content",
			input: `I'll search the database for you.

<function_calls>
<invoke name="search">
<parameter name="q">golang parser</parameter>
</invoke>
</function_calls>

The search is running...`,
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
			name:           "no function calls",
			input:          `<data><item>test</item></data>`,
			wantConfidence: 0,
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
		})
	}
}
