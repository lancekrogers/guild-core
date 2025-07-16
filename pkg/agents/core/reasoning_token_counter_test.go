// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"strings"
	"testing"

	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenCounter for testing
type mockTokenCounter struct {
	tokenCount int
}

func (m *mockTokenCounter) CountTokens(text string) (int, error) {
	// Simple mock: count words * 1.3
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3), nil
}

func (m *mockTokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	total := 0
	for _, msg := range messages {
		count, _ := m.CountTokens(msg.Content)
		total += count
	}
	return total, nil
}

func (m *mockTokenCounter) GetModelLimits() ModelLimits {
	return ModelLimits{
		MaxContextTokens:  100000,
		MaxResponseTokens: 4096,
		Provider:          "mock",
	}
}

func TestReasoningTokenCounter_CountTokensWithBreakdown(t *testing.T) {
	baseCounter := &mockTokenCounter{}
	rtc := NewReasoningTokenCounter(baseCounter)

	tests := []struct {
		name          string
		text          string
		wantTotal     int
		wantReasoning int
		wantContent   int
		wantRatioMin  float64
		wantRatioMax  float64
	}{
		{
			name:          "no reasoning",
			text:          "This is a simple response without any reasoning blocks.",
			wantTotal:     11, // ~9 words * 1.3
			wantReasoning: 0,
			wantContent:   11,
			wantRatioMin:  0.0,
			wantRatioMax:  0.0,
		},
		{
			name:          "single thinking block",
			text:          "Let me think <thinking>This is my reasoning process</thinking> The answer is 42.",
			wantTotal:     15, // ~12 words * 1.3
			wantReasoning: 7,  // "This is my reasoning process" = 28 chars * 0.25
			wantContent:   8,
			wantRatioMin:  0.4,
			wantRatioMax:  0.5,
		},
		{
			name:          "multiple reasoning types",
			text:          "<thinking>Step 1</thinking> then <reasoning>Step 2</reasoning> finally <analysis>Step 3</analysis>",
			wantTotal:     10, // ~8 words * 1.3
			wantReasoning: 3,  // 3 blocks: "Step 1" (6*0.25=1), "Step 2" (6*0.25=1), "Step 3" (6*0.25=1)
			wantContent:   7,
			wantRatioMin:  0.2,
			wantRatioMax:  0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown, err := rtc.CountTokensWithBreakdown(tt.text)
			require.NoError(t, err)

			assert.Equal(t, tt.wantTotal, breakdown.TotalTokens, "total tokens")
			assert.Equal(t, tt.wantReasoning, breakdown.ReasoningTokens, "reasoning tokens")
			assert.Equal(t, tt.wantContent, breakdown.ContentTokens, "content tokens")
			assert.GreaterOrEqual(t, breakdown.ReasoningRatio, tt.wantRatioMin, "ratio min")
			assert.LessOrEqual(t, breakdown.ReasoningRatio, tt.wantRatioMax, "ratio max")
		})
	}
}

func TestReasoningTokenCounter_CountMessagesWithBreakdown(t *testing.T) {
	baseCounter := &mockTokenCounter{}
	rtc := NewReasoningTokenCounter(baseCounter)

	messages := []ContextMessage{
		{
			Role:    "user",
			Content: "Help me solve this problem",
		},
		{
			Role:    "assistant",
			Content: "<thinking>Let me analyze this step by step</thinking> The solution is simple.",
		},
		{
			Role:    "assistant",
			Content: "<reasoning>Further analysis needed</reasoning> Here's the final answer.",
		},
	}

	breakdown, err := rtc.CountMessagesWithBreakdown(messages)
	require.NoError(t, err)

	assert.Greater(t, breakdown.TotalTokens, 0)
	assert.Greater(t, breakdown.ReasoningTokens, 0)
	assert.Greater(t, breakdown.ContentTokens, 0)
	assert.Greater(t, breakdown.ReasoningRatio, 0.0)
	assert.Less(t, breakdown.ReasoningRatio, 1.0)
}

func TestReasoningTokenCounter_AnalyzeResponse(t *testing.T) {
	baseCounter := &mockTokenCounter{}
	rtc := NewReasoningTokenCounter(baseCounter)
	ctx := context.Background()

	tests := []struct {
		name     string
		response *interfaces.ChatResponse
		wantErr  bool
	}{
		{
			name:     "nil response",
			response: nil,
			wantErr:  false,
		},
		{
			name: "response with reasoning",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{
					{
						Message: interfaces.ChatMessage{
							Content: "Let me think <thinking>Processing request</thinking> Answer is 42.",
						},
					},
				},
				Usage: interfaces.UsageInfo{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
				},
			},
			wantErr: false,
		},
		{
			name: "response with provider reasoning tokens",
			response: &interfaces.ChatResponse{
				Choices: []interfaces.ChatChoice{
					{
						Message: interfaces.ChatMessage{
							Content: "The answer is 42.",
						},
					},
				},
				Usage: interfaces.UsageInfo{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					ReasoningTokens:  30, // Provider reported
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown, err := rtc.AnalyzeResponse(ctx, tt.response)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, breakdown)

				if tt.response != nil && tt.response.Usage.ReasoningTokens > 0 {
					// Should use provider-reported reasoning tokens
					assert.Equal(t, tt.response.Usage.ReasoningTokens, breakdown.ReasoningTokens)
				}
			}
		})
	}
}

func TestReasoningTokenCounter_ContextCancellation(t *testing.T) {
	baseCounter := &mockTokenCounter{}
	rtc := NewReasoningTokenCounter(baseCounter)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	response := &interfaces.ChatResponse{
		Choices: []interfaces.ChatChoice{
			{Message: interfaces.ChatMessage{Content: "test"}},
		},
	}

	_, err := rtc.AnalyzeResponse(ctx, response)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestExtractContentWithoutReasoning(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no reasoning",
			input:    "This is plain text",
			expected: "This is plain text",
		},
		{
			name:     "single thinking block",
			input:    "Start <thinking>hidden content</thinking> end",
			expected: "Start  end",
		},
		{
			name:     "multiple reasoning types",
			input:    "A <thinking>think</thinking> B <reasoning>reason</reasoning> C <analysis>analyze</analysis> D",
			expected: "A  B  C  D",
		},
		{
			name:     "nested tags preserved",
			input:    "Text <thinking>has <nested>tags</nested> inside</thinking> more",
			expected: "Text  more",
		},
		{
			name:     "attributes in tags",
			input:    `Start <thinking id="123">content</thinking> end`,
			expected: "Start  end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractContentWithoutReasoning(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUniversalReasoningTokenCounter(t *testing.T) {
	urtc := NewUniversalReasoningTokenCounter()

	tests := []struct {
		provider string
		text     string
	}{
		{
			provider: "openai",
			text:     "Test with <thinking>OpenAI reasoning</thinking> content",
		},
		{
			provider: "anthropic",
			text:     "Test with <reasoning>Claude reasoning</reasoning> content",
		},
		{
			provider: "unknown",
			text:     "Test with unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			breakdown, err := urtc.CountTokensWithBreakdown(tt.provider, tt.text)

			if tt.provider == "unknown" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Greater(t, breakdown.TotalTokens, 0)
			}
		})
	}
}

func TestUniversalReasoningTokenCounter_AnalyzeResponse(t *testing.T) {
	urtc := NewUniversalReasoningTokenCounter()
	ctx := context.Background()

	response := &interfaces.ChatResponse{
		Choices: []interfaces.ChatChoice{
			{
				Message: interfaces.ChatMessage{
					Content: "Answer with <thinking>reasoning</thinking> included",
				},
			},
		},
		Usage: interfaces.UsageInfo{
			CompletionTokens: 10,
		},
	}

	// Test with known provider
	breakdown, err := urtc.AnalyzeResponse(ctx, "openai", response)
	require.NoError(t, err)
	assert.Greater(t, breakdown.ReasoningTokens, 0)

	// Test with unknown provider (should fallback to generic)
	breakdown, err = urtc.AnalyzeResponse(ctx, "unknown", response)
	require.NoError(t, err)
	assert.Greater(t, breakdown.ReasoningTokens, 0)
}

// Benchmarks
func BenchmarkReasoningTokenCounter_CountTokensWithBreakdown(b *testing.B) {
	baseCounter := &mockTokenCounter{}
	rtc := NewReasoningTokenCounter(baseCounter)

	text := `Let me analyze this problem step by step.
<thinking>
First, I need to understand what's being asked.
The user wants a comprehensive solution.
I should break this down into manageable parts.
</thinking>
Based on my analysis, here's the solution:
<reasoning>
The key insight is that we need to approach this systematically.
By breaking down the problem, we can solve each part independently.
</reasoning>
The final answer incorporates all these elements.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rtc.CountTokensWithBreakdown(text)
	}
}

func BenchmarkExtractContentWithoutReasoning(b *testing.B) {
	text := `Complex response with multiple reasoning blocks.
<thinking>First reasoning block with substantial content that needs to be removed</thinking>
Some content in between.
<reasoning>Another reasoning block that should be extracted</reasoning>
More content.
<analysis>Final analysis block</analysis>
Conclusion.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractContentWithoutReasoning(text)
	}
}
