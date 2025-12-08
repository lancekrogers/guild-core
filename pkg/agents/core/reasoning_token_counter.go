// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"strings"
	"sync"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
	"github.com/guild-framework/guild-core/pkg/reasoning"
)

// TokenBreakdown provides detailed token usage information
type TokenBreakdown struct {
	TotalTokens     int     `json:"total_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	ContentTokens   int     `json:"content_tokens"`
	ReasoningRatio  float64 `json:"reasoning_ratio"` // Percentage of tokens used for reasoning
}

// ReasoningTokenCounter extends token counting with reasoning awareness
type ReasoningTokenCounter struct {
	baseCounter TokenCounter
	extractor   *reasoning.Extractor
	mu          sync.RWMutex
}

// NewReasoningTokenCounter creates a new reasoning-aware token counter
func NewReasoningTokenCounter(baseCounter TokenCounter) *ReasoningTokenCounter {
	return &ReasoningTokenCounter{
		baseCounter: baseCounter,
		extractor:   reasoning.NewExtractor(),
	}
}

// CountTokens counts total tokens (delegates to base counter)
func (rtc *ReasoningTokenCounter) CountTokens(text string) (int, error) {
	return rtc.baseCounter.CountTokens(text)
}

// CountMessages counts tokens in messages (delegates to base counter)
func (rtc *ReasoningTokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	return rtc.baseCounter.CountMessages(messages)
}

// GetModelLimits returns model limits (delegates to base counter)
func (rtc *ReasoningTokenCounter) GetModelLimits() ModelLimits {
	return rtc.baseCounter.GetModelLimits()
}

// CountTokensWithBreakdown provides detailed token breakdown including reasoning
func (rtc *ReasoningTokenCounter) CountTokensWithBreakdown(text string) (*TokenBreakdown, error) {
	// Count total tokens
	totalTokens, err := rtc.baseCounter.CountTokens(text)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to count total tokens").
			WithComponent("reasoning_token_counter")
	}

	// Extract reasoning blocks
	blocks := rtc.extractor.ExtractFromContent(text)

	// Count reasoning tokens
	reasoningTokens := 0
	for _, block := range blocks {
		// Use the token count from the block (already calculated by extractor)
		reasoningTokens += block.TokenCount
	}

	// Calculate content tokens
	contentTokens := totalTokens - reasoningTokens
	if contentTokens < 0 {
		contentTokens = 0 // Handle edge case
	}

	// Calculate ratio
	reasoningRatio := 0.0
	if totalTokens > 0 {
		reasoningRatio = float64(reasoningTokens) / float64(totalTokens)
	}

	return &TokenBreakdown{
		TotalTokens:     totalTokens,
		ReasoningTokens: reasoningTokens,
		ContentTokens:   contentTokens,
		ReasoningRatio:  reasoningRatio,
	}, nil
}

// CountMessagesWithBreakdown provides detailed token breakdown for messages
func (rtc *ReasoningTokenCounter) CountMessagesWithBreakdown(messages []ContextMessage) (*TokenBreakdown, error) {
	totalBreakdown := &TokenBreakdown{}

	for _, msg := range messages {
		msgBreakdown, err := rtc.CountTokensWithBreakdown(msg.Content)
		if err != nil {
			return nil, err
		}

		totalBreakdown.TotalTokens += msgBreakdown.TotalTokens
		totalBreakdown.ReasoningTokens += msgBreakdown.ReasoningTokens
		totalBreakdown.ContentTokens += msgBreakdown.ContentTokens
	}

	// Recalculate ratio for combined messages
	if totalBreakdown.TotalTokens > 0 {
		totalBreakdown.ReasoningRatio = float64(totalBreakdown.ReasoningTokens) / float64(totalBreakdown.TotalTokens)
	}

	return totalBreakdown, nil
}

// AnalyzeResponse analyzes a provider response for token breakdown
func (rtc *ReasoningTokenCounter) AnalyzeResponse(ctx context.Context, response *interfaces.ChatResponse) (*TokenBreakdown, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_token_counter")
	}

	if response == nil || len(response.Choices) == 0 {
		return &TokenBreakdown{}, nil
	}

	// Extract reasoning blocks and update response
	blocks, err := rtc.extractor.ExtractFromResponse(ctx, response)
	if err != nil {
		return nil, err
	}

	// Get total tokens from response or calculate
	totalTokens := response.Usage.TotalTokens
	if totalTokens == 0 && response.Usage.CompletionTokens > 0 {
		totalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	}

	// If provider reports reasoning tokens, use them; otherwise use our calculation
	reasoningTokens := response.Usage.ReasoningTokens
	if reasoningTokens == 0 && len(blocks) > 0 {
		// Calculate from extracted blocks
		for _, block := range blocks {
			reasoningTokens += block.TokenCount
		}
	}

	// Calculate content tokens
	contentTokens := response.Usage.CompletionTokens - reasoningTokens
	if contentTokens < 0 {
		contentTokens = response.Usage.CompletionTokens
	}

	// Calculate ratio
	reasoningRatio := 0.0
	if response.Usage.CompletionTokens > 0 {
		reasoningRatio = float64(reasoningTokens) / float64(response.Usage.CompletionTokens)
	}

	return &TokenBreakdown{
		TotalTokens:     totalTokens,
		ReasoningTokens: reasoningTokens,
		ContentTokens:   contentTokens,
		ReasoningRatio:  reasoningRatio,
	}, nil
}

// UniversalReasoningTokenCounter provides reasoning-aware counting for all providers
type UniversalReasoningTokenCounter struct {
	*UniversalTokenCounter
	reasoningCounters map[string]*ReasoningTokenCounter
	mu                sync.RWMutex
}

// NewUniversalReasoningTokenCounter creates a universal reasoning token counter
func NewUniversalReasoningTokenCounter() *UniversalReasoningTokenCounter {
	base := NewUniversalTokenCounter()
	urtc := &UniversalReasoningTokenCounter{
		UniversalTokenCounter: base,
		reasoningCounters:     make(map[string]*ReasoningTokenCounter),
	}

	// Create reasoning counters for each provider
	urtc.mu.Lock()
	for provider, counter := range base.counters {
		urtc.reasoningCounters[provider] = NewReasoningTokenCounter(counter)
	}
	urtc.mu.Unlock()

	return urtc
}

// CountTokensWithBreakdown counts tokens with reasoning breakdown for a provider
func (urtc *UniversalReasoningTokenCounter) CountTokensWithBreakdown(provider, text string) (*TokenBreakdown, error) {
	urtc.mu.RLock()
	counter, exists := urtc.reasoningCounters[provider]
	urtc.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "reasoning token counter not found", nil).
			WithComponent("universal_reasoning_token_counter").
			WithDetails("provider", provider)
	}

	return counter.CountTokensWithBreakdown(text)
}

// AnalyzeResponse analyzes a provider response
func (urtc *UniversalReasoningTokenCounter) AnalyzeResponse(ctx context.Context, provider string, response *interfaces.ChatResponse) (*TokenBreakdown, error) {
	urtc.mu.RLock()
	counter, exists := urtc.reasoningCounters[provider]
	urtc.mu.RUnlock()

	if !exists {
		// Fallback to provider-agnostic analysis
		counter = NewReasoningTokenCounter(&genericTokenCounter{})
	}

	return counter.AnalyzeResponse(ctx, response)
}

// genericTokenCounter provides basic token counting when provider is unknown
type genericTokenCounter struct{}

func (g *genericTokenCounter) CountTokens(text string) (int, error) {
	return EstimateTokens(text), nil
}

func (g *genericTokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	total := 0
	for _, msg := range messages {
		count, _ := g.CountTokens(msg.Content)
		total += count + 5 // Small overhead
	}
	return total, nil
}

func (g *genericTokenCounter) GetModelLimits() ModelLimits {
	return ModelLimits{
		MaxContextTokens:  100000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 1000,
		Provider:          "generic",
	}
}

// ExtractContentWithoutReasoning removes reasoning blocks from text
func ExtractContentWithoutReasoning(text string) string {
	// Simple implementation - remove common reasoning tags
	patterns := []struct{ start, end string }{
		{"<thinking", "</thinking>"},
		{"<reasoning", "</reasoning>"},
		{"<analysis", "</analysis>"},
	}

	result := text
	for _, pattern := range patterns {
		for {
			startIdx := strings.Index(result, pattern.start)
			if startIdx == -1 {
				break
			}

			// Find end of opening tag
			tagEnd := strings.Index(result[startIdx:], ">")
			if tagEnd == -1 {
				break
			}

			// Find closing tag
			endIdx := strings.Index(result[startIdx:], pattern.end)
			if endIdx == -1 {
				break
			}
			endIdx += startIdx + len(pattern.end)

			// Remove the block
			result = result[:startIdx] + result[endIdx:]
		}
	}

	return strings.TrimSpace(result)
}
