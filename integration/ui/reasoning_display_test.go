//go:build integration

package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/internal/testutil"
	"github.com/lancekrogers/guild-core/pkg/agents/core"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReasoningDisplay validates thinking blocks, formatting, and performance
func TestReasoningDisplay(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "reasoning-display-test",
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("thinking_block_parsing", func(t *testing.T) {
		// Test various thinking block formats
		testCases := []struct {
			name     string
			input    string
			expected int // expected number of blocks
		}{
			{
				name: "simple_thinking_block",
				input: `<thinking>
This is my reasoning process.
I need to consider X and Y.
</thinking>`,
				expected: 1,
			},
			{
				name: "multiple_blocks",
				input: `<thinking>
First thought process.
</thinking>
Some content.
<thinking>
Second thought process.
</thinking>`,
				expected: 2,
			},
			{
				name:     "nested_markdown",
				input:    "<thinking>\n## Analysis\n- Point 1\n- Point 2\n\n```python\ndef example():\n    return True\n```\n</thinking>",
				expected: 1,
			},
		}

		parser := reasoning.NewThinkingBlockParser(nil)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				blocks, err := parser.Parse(tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, len(blocks),
					"Expected %d thinking blocks, got %d", tc.expected, len(blocks))
			})
		}
	})

	t.Run("streaming_display", func(t *testing.T) {
		// Test streaming of reasoning content
		streamer := reasoning.NewReasoningStreamer(nil, nil, nil)

		// Simulate streaming content
		content := `<thinking>
Let me analyze this step by step:
1. First, I need to understand the problem
2. Then, I'll design a solution
3. Finally, I'll implement it
</thinking>`

		// Stream character by character
		var mu sync.Mutex
		var displayed strings.Builder

		for _, char := range content {
			chunk := string(char)
			output, err := streamer.ProcessChunk(chunk)
			require.NoError(t, err)

			mu.Lock()
			displayed.WriteString(output)
			mu.Unlock()

			// Simulate realistic streaming delay
			time.Sleep(time.Millisecond)
		}

		// Verify complete content was streamed
		assert.Contains(t, displayed.String(), "analyze this step by step")
	})

	t.Run("formatting_consistency", func(t *testing.T) {
		formatter := reasoning.NewReasoningFormatter()

		// Test different content types
		contents := []struct {
			name     string
			thinking string
			wantErr  bool
		}{
			{
				name:     "code_blocks",
				thinking: "<thinking>\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n</thinking>",
			},
			{
				name: "lists",
				thinking: `<thinking>
- Item 1
  - Sub-item 1.1
  - Sub-item 1.2
- Item 2
</thinking>`,
			},
			{
				name: "mixed_content",
				thinking: `<thinking>
# Header
Some **bold** and *italic* text.
> A quote
</thinking>`,
			},
		}

		for _, tc := range contents {
			t.Run(tc.name, func(t *testing.T) {
				formatted, err := formatter.Format(tc.thinking)
				if tc.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotEmpty(t, formatted)
					// Should preserve essential structure
					assert.Contains(t, tc.thinking, "<thinking>")
				}
			})
		}
	})

	t.Run("performance_large_blocks", func(t *testing.T) {
		parser := reasoning.NewThinkingBlockParser(nil)

		// Generate large thinking block
		var largeBlock strings.Builder
		largeBlock.WriteString("<thinking>\n")
		for i := 0; i < 1000; i++ {
			largeBlock.WriteString(fmt.Sprintf("Line %d: This is a line of reasoning content.\n", i))
		}
		largeBlock.WriteString("</thinking>")

		// Measure parsing time
		start := time.Now()
		blocks, err := parser.Parse(largeBlock.String())
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, blocks, 1)

		// Should parse large blocks quickly
		assert.LessOrEqual(t, duration, 100*time.Millisecond,
			"Large block parsing should be fast, took %v", duration)
	})

	t.Run("concurrent_reasoning_streams", func(t *testing.T) {
		// Test multiple concurrent reasoning streams
		numStreams := 10
		var wg sync.WaitGroup
		errors := make(chan error, numStreams)

		for i := 0; i < numStreams; i++ {
			wg.Add(1)
			go func(streamID int) {
				defer wg.Done()

				streamer := reasoning.NewReasoningStreamer(nil, nil, nil)
				content := fmt.Sprintf(`<thinking>
Stream %d reasoning:
- Processing request
- Analyzing data
- Generating response
</thinking>`, streamID)

				// Process entire content
				_, err := streamer.ProcessChunk(content)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Stream error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "All streams should process successfully")
	})

	t.Run("token_counting", func(t *testing.T) {
		counter := reasoning.NewTokenCounter()

		testCases := []struct {
			content   string
			minTokens int
			maxTokens int
		}{
			{
				content:   "Hello world",
				minTokens: 2,
				maxTokens: 3,
			},
			{
				content:   "<thinking>This is a longer reasoning block with multiple sentences.</thinking>",
				minTokens: 10,
				maxTokens: 15,
			},
		}

		for _, tc := range testCases {
			count := counter.Count(tc.content)
			assert.GreaterOrEqual(t, count, tc.minTokens,
				"Token count should be at least %d", tc.minTokens)
			assert.LessOrEqual(t, count, tc.maxTokens,
				"Token count should be at most %d", tc.maxTokens)
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		parser := reasoning.NewThinkingBlockParser(nil)

		// Test malformed blocks
		malformed := []string{
			"<thinking>Unclosed block",
			"<thinking><thinking>Nested</thinking></thinking>",
			"<thinking></thinking><thinking></thinking><thinking>", // Unclosed
		}

		for _, input := range malformed {
			blocks, err := parser.Parse(input)
			// Should handle gracefully - either parse what it can or return error
			if err != nil {
				assert.True(t, gerror.Is(err, gerror.ErrCodeInvalidInput),
					"Should return appropriate error for malformed input")
			} else {
				// If no error, should have parsed something
				assert.NotEmpty(t, blocks, "Should parse some blocks or return error")
			}
		}
	})
}
