// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// OpenAITokenCounter implements token counting for OpenAI models
type OpenAITokenCounter struct {
	// In production, this would use tiktoken-go
	modelConfigs map[string]ModelLimits
	cache        *tokenCache
}

// NewOpenAITokenCounter creates a new OpenAI token counter
func NewOpenAITokenCounter() *OpenAITokenCounter {
	tc := &OpenAITokenCounter{
		modelConfigs: make(map[string]ModelLimits),
		cache:        newTokenCache(1000, 5*time.Minute),
	}

	// Register model configurations
	tc.modelConfigs["gpt-4"] = ModelLimits{
		MaxContextTokens:  8192,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 1000,
		CostPer1KTokens:   0.03,
		Provider:          "openai",
		Model:             "gpt-4",
	}

	tc.modelConfigs["gpt-4-turbo"] = ModelLimits{
		MaxContextTokens:  128000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 2000,
		CostPer1KTokens:   0.01,
		Provider:          "openai",
		Model:             "gpt-4-turbo",
	}

	tc.modelConfigs["gpt-3.5-turbo"] = ModelLimits{
		MaxContextTokens:  16384,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 1000,
		CostPer1KTokens:   0.0015,
		Provider:          "openai",
		Model:             "gpt-3.5-turbo",
	}

	return tc
}

// CountTokens counts tokens in text
func (tc *OpenAITokenCounter) CountTokens(text string) (int, error) {
	// Check cache
	if count, found := tc.cache.get(text); found {
		return count, nil
	}

	// In production, use tiktoken-go for accurate counting
	// This is a simplified approximation
	count := tc.approximateTokenCount(text)

	// Cache result
	tc.cache.set(text, count)

	return count, nil
}

// CountMessages counts tokens in messages
func (tc *OpenAITokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	total := 0

	// OpenAI has message overhead
	messageOverhead := 4 // tokens per message

	for _, msg := range messages {
		// Count content
		contentTokens, err := tc.CountTokens(msg.Content)
		if err != nil {
			return 0, err
		}

		// Add role tokens
		roleTokens := len(strings.Fields(msg.Role))

		total += contentTokens + roleTokens + messageOverhead

		// TODO: Handle function calls when ToolCalls is added to ContextMessage
		// if msg.ToolCalls != nil {
		// 	toolTokens, err := tc.countToolTokens(msg.ToolCalls)
		// 	if err != nil {
		// 		return 0, err
		// 	}
		// 	total += toolTokens
		// }
	}

	return total, nil
}

// GetModelLimits returns limits for the current model
func (tc *OpenAITokenCounter) GetModelLimits() ModelLimits {
	// Default to GPT-4 Turbo
	if limits, ok := tc.modelConfigs["gpt-4-turbo"]; ok {
		return limits
	}

	return ModelLimits{
		MaxContextTokens:  128000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 2000,
		Provider:          "openai",
	}
}

// approximateTokenCount provides rough token estimation
func (tc *OpenAITokenCounter) approximateTokenCount(text string) int {
	// GPT tokenizer approximation:
	// - Average English word: ~1.3 tokens
	// - Spaces and punctuation: ~0.5 tokens each
	// - Numbers: 1 token per 3-4 digits

	words := strings.Fields(text)
	wordCount := len(words)

	// Count punctuation
	punctCount := 0
	for _, r := range text {
		if strings.ContainsRune(".,!?;:()[]{}\"'", r) {
			punctCount++
		}
	}

	// Rough calculation
	tokens := int(float64(wordCount)*1.3 + float64(punctCount)*0.5)

	// Add buffer for special tokens
	tokens = int(float64(tokens) * 1.1)

	return tokens
}

// countToolTokens counts tokens in tool calls
func (tc *OpenAITokenCounter) countToolTokens(toolCalls interface{}) (int, error) {
	// Serialize to JSON to count
	data, err := json.Marshal(toolCalls)
	if err != nil {
		return 0, err
	}

	return tc.CountTokens(string(data))
}

// AnthropicTokenCounter implements token counting for Anthropic models
type AnthropicTokenCounter struct {
	modelConfigs map[string]ModelLimits
	cache        *tokenCache
}

// NewAnthropicTokenCounter creates a new Anthropic token counter
func NewAnthropicTokenCounter() *AnthropicTokenCounter {
	tc := &AnthropicTokenCounter{
		modelConfigs: make(map[string]ModelLimits),
		cache:        newTokenCache(1000, 5*time.Minute),
	}

	// Register model configurations
	tc.modelConfigs["claude-3-opus"] = ModelLimits{
		MaxContextTokens:  200000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 2000,
		CostPer1KTokens:   0.015,
		Provider:          "anthropic",
		Model:             "claude-3-opus",
	}

	tc.modelConfigs["claude-3-sonnet"] = ModelLimits{
		MaxContextTokens:  200000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 2000,
		CostPer1KTokens:   0.003,
		Provider:          "anthropic",
		Model:             "claude-3-sonnet",
	}

	tc.modelConfigs["claude-3-haiku"] = ModelLimits{
		MaxContextTokens:  200000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 1000,
		CostPer1KTokens:   0.00025,
		Provider:          "anthropic",
		Model:             "claude-3-haiku",
	}

	return tc
}

// CountTokens counts tokens in text
func (tc *AnthropicTokenCounter) CountTokens(text string) (int, error) {
	// Check cache
	if count, found := tc.cache.get(text); found {
		return count, nil
	}

	// Anthropic's tokenization is similar but not identical to OpenAI
	// This is an approximation
	count := tc.approximateTokenCount(text)

	// Cache result
	tc.cache.set(text, count)

	return count, nil
}

// CountMessages counts tokens in messages
func (tc *AnthropicTokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	total := 0

	// Anthropic has different message formatting
	for _, msg := range messages {
		// Count content
		contentTokens, err := tc.CountTokens(msg.Content)
		if err != nil {
			return 0, err
		}

		// Anthropic uses Human/Assistant format
		roleTokens := 2 // Simplified

		total += contentTokens + roleTokens
	}

	return total, nil
}

// GetModelLimits returns limits for the current model
func (tc *AnthropicTokenCounter) GetModelLimits() ModelLimits {
	// Default to Claude 3 Sonnet
	if limits, ok := tc.modelConfigs["claude-3-sonnet"]; ok {
		return limits
	}

	return ModelLimits{
		MaxContextTokens:  200000,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 2000,
		Provider:          "anthropic",
	}
}

// approximateTokenCount provides rough token estimation for Anthropic
func (tc *AnthropicTokenCounter) approximateTokenCount(text string) int {
	// Claude tokenizer approximation (slightly different from OpenAI)
	words := strings.Fields(text)
	wordCount := len(words)

	// Claude tends to use slightly fewer tokens
	tokens := int(float64(wordCount) * 1.25)

	// Add overhead
	tokens = int(float64(tokens) * 1.05)

	return tokens
}

// OllamaTokenCounter implements token counting for Ollama models
type OllamaTokenCounter struct {
	modelConfigs map[string]ModelLimits
	cache        *tokenCache
}

// NewOllamaTokenCounter creates a new Ollama token counter
func NewOllamaTokenCounter() *OllamaTokenCounter {
	tc := &OllamaTokenCounter{
		modelConfigs: make(map[string]ModelLimits),
		cache:        newTokenCache(500, 5*time.Minute),
	}

	// Register common Ollama models
	tc.modelConfigs["llama2:7b"] = ModelLimits{
		MaxContextTokens:  4096,
		MaxResponseTokens: 2048,
		RecommendedBuffer: 500,
		CostPer1KTokens:   0, // Local model
		Provider:          "ollama",
		Model:             "llama2:7b",
	}

	tc.modelConfigs["llama2:13b"] = ModelLimits{
		MaxContextTokens:  4096,
		MaxResponseTokens: 2048,
		RecommendedBuffer: 500,
		CostPer1KTokens:   0,
		Provider:          "ollama",
		Model:             "llama2:13b",
	}

	tc.modelConfigs["mixtral:8x7b"] = ModelLimits{
		MaxContextTokens:  32768,
		MaxResponseTokens: 4096,
		RecommendedBuffer: 1000,
		CostPer1KTokens:   0,
		Provider:          "ollama",
		Model:             "mixtral:8x7b",
	}

	tc.modelConfigs["mistral:7b"] = ModelLimits{
		MaxContextTokens:  8192,
		MaxResponseTokens: 2048,
		RecommendedBuffer: 500,
		CostPer1KTokens:   0,
		Provider:          "ollama",
		Model:             "mistral:7b",
	}

	return tc
}

// CountTokens counts tokens in text
func (tc *OllamaTokenCounter) CountTokens(text string) (int, error) {
	// Check cache
	if count, found := tc.cache.get(text); found {
		return count, nil
	}

	// Llama tokenization approximation
	count := tc.approximateTokenCount(text)

	// Cache result
	tc.cache.set(text, count)

	return count, nil
}

// CountMessages counts tokens in messages
func (tc *OllamaTokenCounter) CountMessages(messages []ContextMessage) (int, error) {
	total := 0

	for _, msg := range messages {
		contentTokens, err := tc.CountTokens(msg.Content)
		if err != nil {
			return 0, err
		}

		// Ollama uses simple formatting
		total += contentTokens + 5 // Small overhead
	}

	return total, nil
}

// GetModelLimits returns limits for the current model
func (tc *OllamaTokenCounter) GetModelLimits() ModelLimits {
	// Default to llama2:7b
	if limits, ok := tc.modelConfigs["llama2:7b"]; ok {
		return limits
	}

	return ModelLimits{
		MaxContextTokens:  4096,
		MaxResponseTokens: 2048,
		RecommendedBuffer: 500,
		Provider:          "ollama",
	}
}

// approximateTokenCount provides rough token estimation for Llama models
func (tc *OllamaTokenCounter) approximateTokenCount(text string) int {
	// Llama tokenizer tends to be more efficient
	words := strings.Fields(text)
	wordCount := len(words)

	// Llama uses SentencePiece, roughly 1.2 tokens per word
	tokens := int(float64(wordCount) * 1.2)

	return tokens
}

// tokenCache provides simple caching for token counts
type tokenCache struct {
	mu       sync.RWMutex
	items    map[string]cacheItem
	maxItems int
	ttl      time.Duration
}

type cacheItem struct {
	count     int
	timestamp time.Time
}

func newTokenCache(maxItems int, ttl time.Duration) *tokenCache {
	tc := &tokenCache{
		items:    make(map[string]cacheItem),
		maxItems: maxItems,
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go tc.cleanup()

	return tc
}

func (tc *tokenCache) get(key string) (int, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	item, exists := tc.items[key]
	if !exists {
		return 0, false
	}

	// Check if expired
	if time.Since(item.timestamp) > tc.ttl {
		return 0, false
	}

	return item.count, true
}

func (tc *tokenCache) set(key string, count int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Simple eviction if at capacity
	if len(tc.items) >= tc.maxItems {
		// Remove oldest item
		var oldestKey string
		oldestTime := time.Now()

		for k, v := range tc.items {
			if v.timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.timestamp
			}
		}

		delete(tc.items, oldestKey)
	}

	tc.items[key] = cacheItem{
		count:     count,
		timestamp: time.Now(),
	}
}

func (tc *tokenCache) cleanup() {
	ticker := time.NewTicker(tc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		tc.mu.Lock()
		now := time.Now()

		for key, item := range tc.items {
			if now.Sub(item.timestamp) > tc.ttl {
				delete(tc.items, key)
			}
		}

		tc.mu.Unlock()
	}
}

// UniversalTokenCounter provides a unified interface for all providers
type UniversalTokenCounter struct {
	counters map[string]TokenCounter
	mu       sync.RWMutex
}

// NewUniversalTokenCounter creates a universal token counter
func NewUniversalTokenCounter() *UniversalTokenCounter {
	utc := &UniversalTokenCounter{
		counters: make(map[string]TokenCounter),
	}

	// Register all providers
	utc.RegisterProvider("openai", NewOpenAITokenCounter())
	utc.RegisterProvider("anthropic", NewAnthropicTokenCounter())
	utc.RegisterProvider("ollama", NewOllamaTokenCounter())

	return utc
}

// RegisterProvider registers a token counter for a provider
func (utc *UniversalTokenCounter) RegisterProvider(provider string, counter TokenCounter) {
	utc.mu.Lock()
	defer utc.mu.Unlock()

	utc.counters[provider] = counter
}

// CountTokens counts tokens for a specific provider
func (utc *UniversalTokenCounter) CountTokens(provider, text string) (int, error) {
	utc.mu.RLock()
	counter, exists := utc.counters[provider]
	utc.mu.RUnlock()

	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, "token counter not found", nil).
			WithComponent("universal_token_counter").
			WithDetails("provider", provider)
	}

	return counter.CountTokens(text)
}

// GetModelLimits gets limits for a specific provider
func (utc *UniversalTokenCounter) GetModelLimits(provider string) (ModelLimits, error) {
	utc.mu.RLock()
	counter, exists := utc.counters[provider]
	utc.mu.RUnlock()

	if !exists {
		return ModelLimits{}, gerror.New(gerror.ErrCodeNotFound, "token counter not found", nil).
			WithComponent("universal_token_counter").
			WithDetails("provider", provider)
	}

	return counter.GetModelLimits(), nil
}

// EstimateTokens provides a quick estimation without provider-specific logic
func EstimateTokens(text string) int {
	// Universal approximation
	words := len(strings.Fields(text))
	chars := len(text)

	// Average of different methods
	byWords := int(float64(words) * 1.3)
	byChars := chars / 4

	return (byWords + byChars) / 2
}
