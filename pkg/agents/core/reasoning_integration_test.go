// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// TestReasoningExtractorIntegration tests the complete reasoning extraction pipeline
func TestReasoningExtractorIntegration(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "test")

	config := DefaultReasoningConfig()
	extractor, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	tests := []struct {
		name               string
		input              string
		expectedContent    string
		expectedHasReason  bool
		expectedConfidence float64
		expectError        bool
	}{
		{
			name: "standard reasoning extraction",
			input: `Let me analyze this request.
<thinking>
The user is asking about API implementation.
I should provide a structured approach.
Key considerations:
- RESTful design principles
- Error handling
- Authentication
Confidence: 0.85
</thinking>

Here's a comprehensive guide to implementing a REST API:

1. Design your endpoints following RESTful conventions
2. Implement proper error handling with status codes
3. Add authentication and authorization
4. Document your API with OpenAPI/Swagger`,
			expectedContent: `Let me analyze this request.

Here's a comprehensive guide to implementing a REST API:

1. Design your endpoints following RESTful conventions
2. Implement proper error handling with status codes
3. Add authentication and authorization
4. Document your API with OpenAPI/Swagger`,
			expectedHasReason:  true,
			expectedConfidence: 0.85,
		},
		{
			name: "multiple reasoning blocks",
			input: `Initial analysis needed.
<thinking>
First, let me understand the problem space.
This appears to be about microservices.
</thinking>

Continuing analysis...

<thinking>
On second thought, this is about monolithic architecture.
The key challenges are different.
Confidence: 0.7
</thinking>

Based on my analysis, here's the recommendation...`,
			expectedContent: `Initial analysis needed.

Continuing analysis...

Based on my analysis, here's the recommendation...`,
			expectedHasReason:  true,
			expectedConfidence: 0.7,
		},
		{
			name:               "no reasoning blocks",
			input:              "This is a simple response without any thinking blocks.",
			expectedContent:    "This is a simple response without any thinking blocks.",
			expectedHasReason:  false,
			expectedConfidence: 0.5,
		},
		{
			name: "malformed confidence",
			input: `<thinking>
Analyzing the request.
Confidence: not-a-number
</thinking>
Response content.`,
			expectedContent:    "Response content.",
			expectedHasReason:  true,
			expectedConfidence: 0.5, // default when parsing fails
		},
		{
			name:               "empty input",
			input:              "",
			expectedContent:    "",
			expectedHasReason:  false,
			expectedConfidence: 0.5,
		},
		{
			name: "nested thinking blocks (edge case)",
			input: `<thinking>
Outer thinking
<thinking>
Should not parse nested
</thinking>
Still in outer
</thinking>
Response`,
			expectedContent:   "Response",
			expectedHasReason: true,
			// The regex should handle this, but exact behavior depends on implementation
		},
		{
			name: "confidence out of range",
			input: `<thinking>
Analysis complete.
Confidence: 1.5
</thinking>
Response`,
			expectedContent:    "Response",
			expectedHasReason:  true,
			expectedConfidence: 1.0, // should be clamped to max
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := extractor.ExtractReasoning(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response)

			// Check content
			assert.Equal(t, tt.expectedContent, response.Content)

			// Check reasoning presence
			hasReasoning := response.Reasoning != ""
			assert.Equal(t, tt.expectedHasReason, hasReasoning)

			// Check confidence
			assert.InDelta(t, tt.expectedConfidence, response.Confidence, 0.01)

			// Check metadata
			assert.NotNil(t, response.Metadata)
			assert.Contains(t, response.Metadata, "extraction_time_ms")
			assert.Equal(t, tt.expectedHasReason, response.Metadata["has_reasoning"])
		})
	}
}

// TestReasoningExtractorConcurrency tests thread safety
func TestReasoningExtractorConcurrency(t *testing.T) {
	ctx := context.Background()
	config := DefaultReasoningConfig()
	extractor, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	// Test concurrent extraction
	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				input := fmt.Sprintf(`Worker %d operation %d
<thinking>
Processing request.
Confidence: %.2f
</thinking>
Response from worker %d`, workerID, j, float64(j%10)/10, workerID)

				response, err := extractor.ExtractReasoning(ctx, input)
				if err != nil {
					errors <- err
					continue
				}

				// Verify response
				if response == nil {
					errors <- fmt.Errorf("nil response from worker %d operation %d", workerID, j)
					continue
				}

				if !strings.Contains(response.Content, fmt.Sprintf("Response from worker %d", workerID)) {
					errors <- fmt.Errorf("incorrect content for worker %d operation %d", workerID, j)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
		if errorCount > 10 {
			t.Fatal("Too many concurrent errors, stopping")
		}
	}

	assert.Equal(t, 0, errorCount, "Expected no errors in concurrent operations")
}

// TestReasoningExtractorContextCancellation tests context cancellation handling
func TestReasoningExtractorContextCancellation(t *testing.T) {
	config := DefaultReasoningConfig()
	extractor, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	// Test immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = extractor.ExtractReasoning(ctx, "test input")
	require.Error(t, err)

	gerr, ok := err.(*gerror.GuildError)
	require.True(t, ok)
	assert.Equal(t, gerror.ErrCodeCancelled, gerr.Code)

	// Test cancellation during processing
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel2()

	// Use a large input to increase processing time
	largeInput := strings.Repeat("<thinking>test</thinking>content", 1000)

	_, err = extractor.ExtractReasoning(ctx2, largeInput)
	// May or may not error depending on timing, but should handle gracefully
	if err != nil {
		gerr, ok := err.(*gerror.GuildError)
		if ok {
			assert.Equal(t, gerror.ErrCodeCancelled, gerr.Code)
		}
	}
}

// TestReasoningExtractorCaching tests cache functionality
func TestReasoningExtractorCaching(t *testing.T) {
	ctx := context.Background()
	config := DefaultReasoningConfig()
	config.EnableCaching = true
	config.CacheTTL = 1 * time.Second

	extractor, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	input := `<thinking>
Test reasoning for caching.
Confidence: 0.75
</thinking>
Cached response content.`

	// First call - should miss cache
	response1, err := extractor.ExtractReasoning(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, response1)
	assert.False(t, response1.Metadata["cached"].(bool))

	// Second call - should hit cache
	response2, err := extractor.ExtractReasoning(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, response2)

	// Verify responses are identical
	assert.Equal(t, response1.Content, response2.Content)
	assert.Equal(t, response1.Reasoning, response2.Reasoning)
	assert.Equal(t, response1.Confidence, response2.Confidence)

	// Check cache stats
	stats := extractor.GetStats()
	assert.Greater(t, stats["cache_hits"].(uint64), uint64(0))

	// Wait for cache expiration
	time.Sleep(2 * time.Second)

	// Third call - should miss cache again
	response3, err := extractor.ExtractReasoning(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, response3)
	assert.False(t, response3.Metadata["cached"].(bool))
}

// TestReasoningExtractorValidation tests input validation
func TestReasoningExtractorValidation(t *testing.T) {
	ctx := context.Background()

	// Test invalid configurations
	invalidConfigs := []struct {
		name   string
		config ReasoningConfig
	}{
		{
			name: "negative cache size",
			config: ReasoningConfig{
				CacheMaxSize: -1,
			},
		},
		{
			name: "invalid confidence range",
			config: ReasoningConfig{
				MinConfidence: 1.5,
				MaxConfidence: 2.0,
			},
		},
		{
			name: "min > max confidence",
			config: ReasoningConfig{
				MinConfidence: 0.8,
				MaxConfidence: 0.3,
			},
		},
	}

	for _, tc := range invalidConfigs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewReasoningExtractor(tc.config)
			assert.Error(t, err)

			gerr, ok := err.(*gerror.GuildError)
			require.True(t, ok)
			assert.Equal(t, gerror.ErrCodeValidation, gerr.Code)
		})
	}

	// Test valid extractor with strict validation
	config := DefaultReasoningConfig()
	config.StrictValidation = true
	_, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	// Test extraction with very long reasoning (should truncate)
	config.MaxReasoningLength = 100
	extractor2, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	longReasoning := strings.Repeat("This is a very long reasoning text. ", 50)
	input := fmt.Sprintf(`<thinking>%s</thinking>Short response.`, longReasoning)

	response, err := extractor2.ExtractReasoning(ctx, input)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(response.Reasoning), config.MaxReasoningLength)
}

// TestReasoningExtractorMetrics tests metrics collection
func TestReasoningExtractorMetrics(t *testing.T) {
	ctx := context.Background()
	config := DefaultReasoningConfig()
	extractor, err := NewReasoningExtractor(config)
	require.NoError(t, err)

	// Perform several extractions with different characteristics
	inputs := []string{
		`<thinking>Short reasoning. Confidence: 0.3</thinking>Response.`,
		`<thinking>Medium length reasoning with more detail. Confidence: 0.6</thinking>Response.`,
		`<thinking>Very detailed reasoning with lots of analysis and consideration of multiple factors. Confidence: 0.9</thinking>Response.`,
		`No reasoning in this response.`,
	}

	for _, input := range inputs {
		_, err := extractor.ExtractReasoning(ctx, input)
		require.NoError(t, err)
	}

	// Verify metrics were recorded
	// Note: In a real implementation, you'd check the actual metrics backend
	stats := extractor.GetStats()
	assert.NotNil(t, stats)
}

// BenchmarkReasoningExtraction benchmarks the extraction performance
func BenchmarkReasoningExtraction(b *testing.B) {
	ctx := context.Background()
	config := DefaultReasoningConfig()
	config.EnableCaching = false // Disable caching for accurate benchmarks

	extractor, err := NewReasoningExtractor(config)
	require.NoError(b, err)

	// Prepare test inputs of different sizes
	smallInput := `<thinking>Small reasoning. Confidence: 0.8</thinking>Small response.`

	mediumInput := `Initial thoughts here.
<thinking>
This is a medium-sized reasoning block with several considerations:
- First point to consider
- Second point to analyze
- Third aspect to evaluate
Overall assessment suggests a moderate approach.
Confidence: 0.75
</thinking>
Here's a medium-length response with multiple paragraphs and detailed explanation.`

	largeInput := strings.Repeat(`<thinking>
Detailed analysis block with extensive reasoning.
Multiple considerations and factors.
Confidence: 0.9
</thinking>
Response content with details.
`, 10)

	b.Run("Small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := extractor.ExtractReasoning(ctx, smallInput)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Medium", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := extractor.ExtractReasoning(ctx, mediumInput)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Large", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := extractor.ExtractReasoning(ctx, largeInput)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		cachedExtractor, _ := NewReasoningExtractor(DefaultReasoningConfig())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use same input to test cache performance
			_, err := cachedExtractor.ExtractReasoning(ctx, mediumInput)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
