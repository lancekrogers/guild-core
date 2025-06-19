// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"sync"
	"time"
)

// ModelQueryService queries providers for available models
type ModelQueryService struct {
	cache      map[string]*ModelQueryResult
	cacheMutex sync.RWMutex
	cacheTime  time.Duration
}

// ModelQueryResult contains the result of a model query
type ModelQueryResult struct {
	ProviderName string
	Models       []ModelInfo
	QueryTime    time.Time
	Error        error
}

// NewModelQueryService creates a new model query service
func NewModelQueryService() *ModelQueryService {
	return &ModelQueryService{
		cache:     make(map[string]*ModelQueryResult),
		cacheTime: 5 * time.Minute, // Cache for 5 minutes
	}
}

// GetModelCount gets the number of available models for a provider
func (s *ModelQueryService) GetModelCount(ctx context.Context, providerName string) int {
	// Check cache first
	s.cacheMutex.RLock()
	if result, exists := s.cache[providerName]; exists {
		if time.Since(result.QueryTime) < s.cacheTime {
			s.cacheMutex.RUnlock()
			return len(result.Models)
		}
	}
	s.cacheMutex.RUnlock()

	// Query provider (would be implemented with actual provider APIs)
	count := s.queryProviderModels(ctx, providerName)
	
	// Update cache
	s.cacheMutex.Lock()
	// Handle dynamic counts (like -1 for Ollama)
	models := []ModelInfo{}
	if count > 0 {
		models = make([]ModelInfo, count)
	}
	s.cache[providerName] = &ModelQueryResult{
		ProviderName: providerName,
		Models:       models,
		QueryTime:    time.Now(),
	}
	s.cacheMutex.Unlock()
	
	// Return 0 for dynamic/unknown counts
	if count < 0 {
		return 0
	}
	return count
}

// queryProviderModels queries a provider for available models
// In a real implementation, this would call the provider's API
func (s *ModelQueryService) queryProviderModels(ctx context.Context, providerName string) int {
	// For now, return sensible defaults
	// In the future, this should query the actual provider APIs
	defaults := map[string]int{
		"openai":      8,  // GPT-4, GPT-4-turbo, GPT-3.5, etc.
		"anthropic":   4,  // Claude 3 Opus, Sonnet, Haiku, Claude 2
		"ollama":      -1, // Dynamic based on local models
		"claude_code": 2,  // Claude 3 variants
		"deepseek":    3,  // DeepSeek models
		"deepinfra":   10, // Various open models
	}
	
	if count, exists := defaults[providerName]; exists {
		return count
	}
	
	return 0
}