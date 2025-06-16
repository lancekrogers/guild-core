// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package execution

import (
	"time"
)

// CachedPromptBuilder wraps PromptBuilder with caching
type CachedPromptBuilder struct {
	*PromptBuilder
	cache *PromptCache
}

// NewCachedPromptBuilder creates a new cached prompt builder
func NewCachedPromptBuilder() (*CachedPromptBuilder, error) {
	builder, err := NewPromptBuilder()
	if err != nil {
		return nil, err
	}

	return &CachedPromptBuilder{
		PromptBuilder: builder,
		cache:         NewPromptCache(100, 5*time.Minute), // Cache up to 100 prompts for 5 minutes
	}, nil
}

// BuildPromptCached builds a prompt with caching
func (b *CachedPromptBuilder) BuildPromptCached(layers []Layer, data map[string]interface{}) (string, error) {
	// Generate cache key
	key, err := GenerateKey(layers, data)
	if err != nil {
		// If we can't generate a key, just build without caching
		return b.BuildPrompt(layers, data)
	}

	// Check cache
	if prompt, found := b.cache.Get(key); found {
		return prompt, nil
	}

	// Build prompt
	prompt, err := b.BuildPrompt(layers, data)
	if err != nil {
		return "", err
	}

	// Cache the result
	b.cache.Set(key, prompt)

	return prompt, nil
}

// BuildFullExecutionPromptCached builds a full execution prompt with caching
func (b *CachedPromptBuilder) BuildFullExecutionPromptCached(data ExecutionPromptData) (string, error) {
	// For full execution prompts, we typically don't cache because execution state changes frequently
	// But we could cache the base layers
	return b.BuildFullExecutionPrompt(data)
}

// BuildPlanningPromptCached builds a planning prompt with caching
func (b *CachedPromptBuilder) BuildPlanningPromptCached(data ExecutionPromptData) (string, error) {
	// Planning prompts are good candidates for caching since they don't include execution state
	key, err := GenerateKey([]Layer{LayerBase, LayerContext, LayerTask, LayerTool}, data)
	if err != nil {
		return b.BuildPlanningPrompt(data)
	}

	if prompt, found := b.cache.Get(key); found {
		return prompt, nil
	}

	prompt, err := b.BuildPlanningPrompt(data)
	if err != nil {
		return "", err
	}

	b.cache.Set(key, prompt)
	return prompt, nil
}

// ClearCache clears the prompt cache
func (b *CachedPromptBuilder) ClearCache() {
	b.cache.Clear()
}

// GetCacheStats returns cache statistics
func (b *CachedPromptBuilder) GetCacheStats() CacheStats {
	return b.cache.Stats()
}
