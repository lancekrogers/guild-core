// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_ExtractFromResponse(t *testing.T) {
	extractor := NewExtractor()
	ctx := context.Background()

	tests := []struct {
		name     string
		response *interfaces.ChatResponse
		expected int
	}{
		{
			name:     "nil response",
			response: nil,
			expected: 0,
		},
		{
			name: "empty choices",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{},
			},
			expected: 0,
		},
		{
			name: "response with thinking tags",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{
					{
						Message: interfaces.ChatMessage{
							Content: "Let me think <thinking>Processing the request</thinking> The answer is 42.",
						},
					},
				},
				Usage: interfaces.UsageInfo{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			expected: 1,
		},
		{
			name: "response with multiple reasoning types",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{
					{
						Message: interfaces.ChatMessage{
							Content: "<thinking>Initial thoughts</thinking> then <reasoning>Deep analysis</reasoning> and <analysis>Final review</analysis>",
						},
					},
				},
				Usage: interfaces.UsageInfo{},
			},
			expected: 3,
		},
		{
			name: "multiple choices with reasoning",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{
					{
						Message: interfaces.ChatMessage{
							Content: "<thinking>First choice reasoning</thinking>",
						},
					},
					{
						Message: interfaces.ChatMessage{
							Content: "<reasoning>Second choice reasoning</reasoning>",
						},
					},
				},
				Usage: interfaces.UsageInfo{},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := extractor.ExtractFromResponse(ctx, tt.response)
			require.NoError(t, err)
			assert.Len(t, blocks, tt.expected)

			// Check that reasoning tokens are updated
			if tt.expected > 0 && tt.response != nil {
				assert.Greater(t, tt.response.Usage.ReasoningTokens, 0)
			}

			// Verify block properties
			for _, block := range blocks {
				assert.NotEmpty(t, block.ID)
				assert.NotEmpty(t, block.Type)
				assert.NotEmpty(t, block.Content)
				assert.NotZero(t, block.Timestamp)
				assert.GreaterOrEqual(t, block.TokenCount, 0)
			}
		})
	}
}

func TestExtractor_ExtractFromContent(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		content  string
		expected int
		types    []string
	}{
		{
			name:     "no reasoning blocks",
			content:  "This is regular content without any reasoning blocks.",
			expected: 0,
			types:    []string{},
		},
		{
			name:     "single thinking block",
			content:  "Response text <thinking>This is my reasoning process</thinking> final answer",
			expected: 1,
			types:    []string{"thinking"},
		},
		{
			name:     "nested content",
			content:  "<thinking>Complex reasoning with <nested>tags</nested> inside</thinking>",
			expected: 1,
			types:    []string{"thinking"},
		},
		{
			name:     "all three types",
			content:  "<thinking>Think</thinking> <reasoning>Reason</reasoning> <analysis>Analyze</analysis>",
			expected: 3,
			types:    []string{"thinking", "reasoning", "analysis"},
		},
		{
			name:     "thinking block with attributes",
			content:  `<thinking id="123" type="deep">Detailed analysis here</thinking>`,
			expected: 1,
			types:    []string{"thinking"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := extractor.ExtractFromContent(tt.content)
			assert.Len(t, blocks, tt.expected)

			// Check block types
			for i, block := range blocks {
				if i < len(tt.types) {
					assert.Equal(t, tt.types[i], block.Type)
				}
			}
		})
	}
}

func TestExtractor_ContextCancellation(t *testing.T) {
	extractor := NewExtractor()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	response := &interfaces.ChatResponse{
		Choices: []interfaces.ChatChoice{
			{
				Message: interfaces.ChatMessage{
					Content: "Test content",
				},
			},
		},
	}

	_, err := extractor.ExtractFromResponse(ctx, response)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestStreamExtractor_ProcessChunk(t *testing.T) {
	extractor := NewExtractor()
	streamExtractor := NewStreamExtractor(extractor)
	ctx := context.Background()

	// Simulate streaming chunks
	chunks := []string{
		"Let me think about this ",
		"<thinking>I need to ",
		"analyze the problem ",
		"step by step</thinking> ",
		"The answer is 42.",
	}

	receivedBlocks := 0
	done := make(chan bool)

	// Listen for blocks
	go func() {
		for range streamExtractor.Channel() {
			receivedBlocks++
		}
		done <- true
	}()

	// Process chunks
	for _, chunk := range chunks {
		streamExtractor.ProcessChunk(ctx, chunk)
	}

	// Close and wait
	streamExtractor.Close()
	<-done

	assert.Equal(t, 1, receivedBlocks)
}

func TestStreamExtractor_MultipleBlocks(t *testing.T) {
	extractor := NewExtractor()
	streamExtractor := NewStreamExtractor(extractor)
	ctx := context.Background()

	// Simulate streaming with multiple reasoning blocks
	chunks := []string{
		"<thinking>First thought</thinking> ",
		"middle text ",
		"<reasoning>Deep ",
		"analysis here</reasoning> ",
		"<analysis>Final review</analysis>",
	}

	var receivedBlocks []*interfaces.ReasoningBlock
	done := make(chan bool)

	// Listen for blocks
	go func() {
		for block := range streamExtractor.Channel() {
			receivedBlocks = append(receivedBlocks, block)
		}
		done <- true
	}()

	// Process chunks
	for _, chunk := range chunks {
		streamExtractor.ProcessChunk(ctx, chunk)
	}

	// Close and wait
	streamExtractor.Close()
	<-done

	assert.Len(t, receivedBlocks, 3)
	assert.Equal(t, "thinking", receivedBlocks[0].Type)
	assert.Equal(t, "reasoning", receivedBlocks[1].Type)
	assert.Equal(t, "analysis", receivedBlocks[2].Type)
}

// BenchmarkExtractor_ExtractFromContent benchmarks extraction performance
func BenchmarkExtractor_ExtractFromContent(b *testing.B) {
	extractor := NewExtractor()

	content := `
This is some initial content.
<thinking>
This is the first thinking block with some reasoning content.
It spans multiple lines and contains various thoughts.
</thinking>
Some middle content here.
<reasoning>
This is a reasoning block with deep analysis.
It also has multiple lines of content.
</reasoning>
<analysis>
Final analysis and conclusions go here.
This wraps up our reasoning process.
</analysis>
Final content after all reasoning blocks.
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractFromContent(content)
	}
}
