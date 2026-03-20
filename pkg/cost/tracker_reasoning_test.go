// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReasoningTokenCostTracking(t *testing.T) {
	ctx := context.Background()

	t.Run("OpenAI o1 reasoning cost", func(t *testing.T) {
		provider, err := NewOpenAICostProvider(ctx, "test-key")
		require.NoError(t, err)

		usage := Usage{
			AgentID:   "test-agent",
			Provider:  "openai",
			Resource:  "completion",
			Quantity:  1,
			Unit:      "request",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"model":            "o1-preview",
				"input_tokens":     1000,
				"output_tokens":    2000,
				"reasoning_tokens": 5000, // Significant reasoning
			},
		}

		err = provider.TrackUsage(ctx, usage)
		assert.NoError(t, err)

		// Verify costs were calculated correctly
		assert.NotNil(t, usage.Metadata["input_cost"])
		assert.NotNil(t, usage.Metadata["output_cost"])
		assert.NotNil(t, usage.Metadata["reasoning_cost"])
		assert.NotNil(t, usage.Metadata["total_cost"])

		// Check specific values
		inputCost := usage.Metadata["input_cost"].(float64)
		outputCost := usage.Metadata["output_cost"].(float64)
		reasoningCost := usage.Metadata["reasoning_cost"].(float64)
		totalCost := usage.Metadata["total_cost"].(float64)

		// o1-preview: $15/1M input, $60/1M output, $15/1M reasoning
		assert.InDelta(t, 0.015, inputCost, 0.0001)     // 1000 tokens * $15/1M
		assert.InDelta(t, 0.12, outputCost, 0.0001)     // 2000 tokens * $60/1M
		assert.InDelta(t, 0.075, reasoningCost, 0.0001) // 5000 tokens * $15/1M
		assert.InDelta(t, 0.21, totalCost, 0.0001)      // Sum of all costs
	})

	t.Run("Anthropic Claude extended thinking cost", func(t *testing.T) {
		provider, err := NewAnthropicCostProvider(ctx, "test-key")
		require.NoError(t, err)

		usage := Usage{
			AgentID:   "test-agent",
			Provider:  "anthropic",
			Resource:  "completion",
			Quantity:  1,
			Unit:      "request",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"model":            "claude-3-opus-extended",
				"input_tokens":     1000,
				"output_tokens":    2000,
				"reasoning_tokens": 3000,
			},
		}

		err = provider.TrackUsage(ctx, usage)
		assert.NoError(t, err)

		// Verify costs were calculated correctly
		assert.NotNil(t, usage.Metadata["input_cost"])
		assert.NotNil(t, usage.Metadata["output_cost"])
		assert.NotNil(t, usage.Metadata["reasoning_cost"])
		assert.NotNil(t, usage.Metadata["total_cost"])

		// Check specific values
		inputCost := usage.Metadata["input_cost"].(float64)
		outputCost := usage.Metadata["output_cost"].(float64)
		reasoningCost := usage.Metadata["reasoning_cost"].(float64)
		totalCost := usage.Metadata["total_cost"].(float64)

		// claude-3-opus-extended: $15/1M input, $75/1M output, $15/1M reasoning
		assert.InDelta(t, 0.015, inputCost, 0.0001)     // 1000 tokens * $15/1M
		assert.InDelta(t, 0.15, outputCost, 0.0001)     // 2000 tokens * $75/1M
		assert.InDelta(t, 0.045, reasoningCost, 0.0001) // 3000 tokens * $15/1M
		assert.InDelta(t, 0.21, totalCost, 0.0001)      // Sum of all costs
	})

	t.Run("No reasoning tokens", func(t *testing.T) {
		provider, err := NewOpenAICostProvider(ctx, "test-key")
		require.NoError(t, err)

		usage := Usage{
			AgentID:   "test-agent",
			Provider:  "openai",
			Resource:  "completion",
			Quantity:  1,
			Unit:      "request",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"model":         "gpt-4",
				"input_tokens":  1000,
				"output_tokens": 2000,
				// No reasoning_tokens field
			},
		}

		err = provider.TrackUsage(ctx, usage)
		assert.NoError(t, err)

		// Verify no reasoning cost was added
		assert.NotNil(t, usage.Metadata["input_cost"])
		assert.NotNil(t, usage.Metadata["output_cost"])
		assert.NotNil(t, usage.Metadata["total_cost"])
		assert.Nil(t, usage.Metadata["reasoning_cost"])

		// Check specific values
		inputCost := usage.Metadata["input_cost"].(float64)
		outputCost := usage.Metadata["output_cost"].(float64)
		totalCost := usage.Metadata["total_cost"].(float64)

		// gpt-4: $30/1M input, $60/1M output
		assert.InDelta(t, 0.03, inputCost, 0.0001)  // 1000 tokens * $30/1M
		assert.InDelta(t, 0.12, outputCost, 0.0001) // 2000 tokens * $60/1M
		assert.InDelta(t, 0.15, totalCost, 0.0001)  // Sum of input + output only
	})
}

func TestRateCardReasoningSupport(t *testing.T) {
	ctx := context.Background()

	t.Run("OpenAI rate card has reasoning rates", func(t *testing.T) {
		provider, err := NewOpenAICostProvider(ctx, "test-key")
		require.NoError(t, err)

		rates, err := provider.GetRates(ctx)
		require.NoError(t, err)

		// Check o1 models have reasoning rates
		assert.Contains(t, rates.Rates, "o1-preview")
		assert.Contains(t, rates.Rates["o1-preview"], "reasoning")
		assert.Equal(t, 15.0, rates.Rates["o1-preview"]["reasoning"])

		assert.Contains(t, rates.Rates, "o1-mini")
		assert.Contains(t, rates.Rates["o1-mini"], "reasoning")
		assert.Equal(t, 3.0, rates.Rates["o1-mini"]["reasoning"])
	})

	t.Run("Anthropic rate card has reasoning rates", func(t *testing.T) {
		provider, err := NewAnthropicCostProvider(ctx, "test-key")
		require.NoError(t, err)

		rates, err := provider.GetRates(ctx)
		require.NoError(t, err)

		// Check extended models have reasoning rates
		assert.Contains(t, rates.Rates, "claude-3-opus-extended")
		assert.Contains(t, rates.Rates["claude-3-opus-extended"], "reasoning")
		assert.Equal(t, 15.0, rates.Rates["claude-3-opus-extended"]["reasoning"])

		assert.Contains(t, rates.Rates, "claude-3-sonnet-extended")
		assert.Contains(t, rates.Rates["claude-3-sonnet-extended"], "reasoning")
		assert.Equal(t, 3.0, rates.Rates["claude-3-sonnet-extended"]["reasoning"])
	})
}
