// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
)

// SuggestionService provides intelligent suggestions for chat interactions
type SuggestionService struct {
	ctx              context.Context
	suggestionHandler *agent.ChatSuggestionHandler
	
	// Caching
	cache     map[string]*cachedSuggestion
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
	
	// Token optimization
	tokenLimit      int
	tokenBudget     int
	tokenUsed       int
	
	// Configuration
	config          agent.ChatSuggestionConfig
	maxSuggestions  int
	minConfidence   float64
	
	// Statistics
	totalRequests   int
	cacheHits       int
	cacheMisses     int
	avgLatency      time.Duration
	lastRequest     time.Time
}

// cachedSuggestion represents a cached suggestion result
type cachedSuggestion struct {
	suggestions []suggestions.Suggestion
	metadata    map[string]interface{}
	timestamp   time.Time
	tokenCost   int
}

// NewSuggestionService creates a new suggestion service
func NewSuggestionService(ctx context.Context, handler *agent.ChatSuggestionHandler) (*SuggestionService, error) {
	if handler == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "suggestion handler cannot be nil", nil).
			WithComponent("services.suggestion").
			WithOperation("NewSuggestionService")
	}
	
	return &SuggestionService{
		ctx:               ctx,
		suggestionHandler: handler,
		cache:             make(map[string]*cachedSuggestion),
		cacheTTL:          5 * time.Minute,
		tokenLimit:        8192,
		tokenBudget:       4096,
		config:            agent.DefaultChatSuggestionConfig(),
		maxSuggestions:    5,
		minConfidence:     0.5,
	}, nil
}

// Start initializes the suggestion service
func (s *SuggestionService) Start() tea.Cmd {
	return func() tea.Msg {
		// Initialize any required resources
		s.cleanupCache()
		
		return SuggestionServiceStartedMsg{
			Config: s.config,
		}
	}
}

// GetSuggestions retrieves suggestions for the given context
func (s *SuggestionService) GetSuggestions(message string, context *SuggestionContext) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		s.totalRequests++
		
		// Check cache first
		cacheKey := s.buildCacheKey(message, context)
		if cached := s.getFromCache(cacheKey); cached != nil {
			s.cacheHits++
			return SuggestionsReceivedMsg{
				Suggestions: cached.suggestions,
				Metadata:    cached.metadata,
				FromCache:   true,
				Latency:     time.Since(start),
			}
		}
		
		s.cacheMisses++
		
		// Build optimized request
		request := s.buildOptimizedRequest(message, context)
		
		// Get suggestions from handler
		response, err := s.suggestionHandler.GetSuggestions(s.ctx, request)
		if err != nil {
			return SuggestionServiceErrorMsg{
				Operation: "get_suggestions",
				Error:     err,
			}
		}
		
		// Calculate token usage
		tokenCost := s.estimateTokenCost(request, response)
		s.tokenUsed += tokenCost
		
		// Cache the result
		s.putInCache(cacheKey, response.Suggestions, response.Metadata, tokenCost)
		
		// Update statistics
		latency := time.Since(start)
		s.updateLatency(latency)
		s.lastRequest = time.Now()
		
		return SuggestionsReceivedMsg{
			Suggestions: response.Suggestions,
			Metadata:    response.Metadata,
			FromCache:   false,
			Latency:     latency,
			TokensUsed:  tokenCost,
		}
	}
}

// GetFollowUpSuggestions gets suggestions based on previous interaction
func (s *SuggestionService) GetFollowUpSuggestions(previousMessage string, response string) tea.Cmd {
	return func() tea.Msg {
		context := &SuggestionContext{
			PreviousMessage: previousMessage,
			PreviousResponse: response,
			IsFollowUp: true,
		}
		
		// Use the main GetSuggestions logic
		cmd := s.GetSuggestions("", context)
		return cmd()
	}
}

// OptimizeContext builds an optimized context for token efficiency
func (s *SuggestionService) OptimizeContext(fullContext string) string {
	logger := observability.GetLogger(s.ctx).
		WithComponent("services.suggestion").
		WithOperation("OptimizeContext")
	
	// Simple token estimation (rough approximation)
	estimatedTokens := len(fullContext) / 4
	
	if estimatedTokens <= s.tokenBudget {
		return fullContext
	}
	
	// Truncate to fit within budget
	maxChars := s.tokenBudget * 4
	if len(fullContext) > maxChars {
		logger.Debug("Truncating context for token optimization",
			"original_length", len(fullContext),
			"truncated_length", maxChars)
		
		// Keep the most recent content
		return fullContext[len(fullContext)-maxChars:]
	}
	
	return fullContext
}

// SetTokenBudget sets the token budget for suggestions
func (s *SuggestionService) SetTokenBudget(budget int) {
	s.tokenBudget = budget
}

// SetCacheTTL sets the cache time-to-live
func (s *SuggestionService) SetCacheTTL(ttl time.Duration) {
	s.cacheTTL = ttl
}

// SetConfig updates the suggestion configuration
func (s *SuggestionService) SetConfig(config agent.ChatSuggestionConfig) {
	s.config = config
}

// GetStats returns statistics about the suggestion service
func (s *SuggestionService) GetStats() map[string]interface{} {
	s.cacheMu.RLock()
	cacheSize := len(s.cache)
	s.cacheMu.RUnlock()
	
	hitRate := float64(0)
	if s.totalRequests > 0 {
		hitRate = float64(s.cacheHits) / float64(s.totalRequests) * 100
	}
	
	stats := map[string]interface{}{
		"total_requests":   s.totalRequests,
		"cache_hits":       s.cacheHits,
		"cache_misses":     s.cacheMisses,
		"cache_hit_rate":   fmt.Sprintf("%.2f%%", hitRate),
		"cache_size":       cacheSize,
		"cache_ttl":        s.cacheTTL.String(),
		"token_limit":      s.tokenLimit,
		"token_budget":     s.tokenBudget,
		"token_used":       s.tokenUsed,
		"avg_latency":      s.avgLatency.String(),
		"last_request":     s.lastRequest.Format(time.RFC3339),
	}
	
	return stats
}

// ClearCache clears the suggestion cache
func (s *SuggestionService) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	s.cache = make(map[string]*cachedSuggestion)
}

// buildCacheKey creates a cache key from request parameters
func (s *SuggestionService) buildCacheKey(message string, context *SuggestionContext) string {
	key := fmt.Sprintf("%s", message)
	
	if context != nil {
		if context.ConversationID != "" {
			key += fmt.Sprintf(":%s", context.ConversationID)
		}
		if context.IsFollowUp {
			key += ":followup"
		}
	}
	
	return key
}

// getFromCache retrieves a cached suggestion if available
func (s *SuggestionService) getFromCache(key string) *cachedSuggestion {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	
	cached, exists := s.cache[key]
	if !exists {
		return nil
	}
	
	// Check if cache entry is still valid
	if time.Since(cached.timestamp) > s.cacheTTL {
		return nil
	}
	
	return cached
}

// putInCache stores a suggestion result in cache
func (s *SuggestionService) putInCache(key string, suggestions []suggestions.Suggestion, metadata map[string]interface{}, tokenCost int) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	s.cache[key] = &cachedSuggestion{
		suggestions: suggestions,
		metadata:    metadata,
		timestamp:   time.Now(),
		tokenCost:   tokenCost,
	}
}

// cleanupCache removes expired cache entries
func (s *SuggestionService) cleanupCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	now := time.Now()
	for key, cached := range s.cache {
		if now.Sub(cached.timestamp) > s.cacheTTL {
			delete(s.cache, key)
		}
	}
}

// buildOptimizedRequest creates an optimized suggestion request
func (s *SuggestionService) buildOptimizedRequest(message string, context *SuggestionContext) agent.SuggestionRequest {
	request := agent.SuggestionRequest{
		Message:        message,
		MaxSuggestions: s.maxSuggestions,
		MinConfidence:  s.minConfidence,
	}
	
	if context != nil {
		request.ConversationID = context.ConversationID
		
		// Add file context if available
		if context.FileContext != nil {
			request.FileContext = context.FileContext
		}
		
		// Build filter based on context
		filter := &suggestions.SuggestionFilter{
			MaxResults:    s.maxSuggestions,
			MinConfidence: s.minConfidence,
		}
		
		if s.config.EnabledTypes != nil {
			filter.Types = s.config.EnabledTypes
		}
		
		request.Filter = filter
	}
	
	// Apply configuration
	s.suggestionHandler.ApplyConfig(s.config, &request)
	
	return request
}

// estimateTokenCost estimates the token cost of a request/response
func (s *SuggestionService) estimateTokenCost(request agent.SuggestionRequest, response *agent.SuggestionResponse) int {
	// Simple estimation: 1 token per 4 characters
	cost := len(request.Message) / 4
	
	if response != nil {
		for _, suggestion := range response.Suggestions {
			cost += len(suggestion.Content) / 4
			cost += len(suggestion.Description) / 4
		}
	}
	
	return cost
}

// updateLatency updates the average latency calculation
func (s *SuggestionService) updateLatency(latency time.Duration) {
	if s.avgLatency == 0 {
		s.avgLatency = latency
	} else {
		// Simple moving average
		s.avgLatency = (s.avgLatency + latency) / 2
	}
}

// StartPeriodicCleanup starts periodic cache cleanup
func (s *SuggestionService) StartPeriodicCleanup() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return CacheCleanupMsg{Timestamp: t}
	})
}

// HandleCacheCleanup handles cache cleanup tick
func (s *SuggestionService) HandleCacheCleanup() tea.Cmd {
	return func() tea.Msg {
		s.cleanupCache()
		return CacheCleanedMsg{
			ItemsRemoved: s.totalRequests - s.cacheHits,
		}
	}
}

// SuggestionContext provides context for suggestion generation
type SuggestionContext struct {
	ConversationID   string
	FileContext      *suggestions.FileContext
	PreviousMessage  string
	PreviousResponse string
	IsFollowUp       bool
	UserPreferences  map[string]interface{}
}

// Message types for suggestion service

// SuggestionServiceStartedMsg indicates the service has started
type SuggestionServiceStartedMsg struct {
	Config agent.ChatSuggestionConfig
}

// SuggestionServiceErrorMsg represents a service error
type SuggestionServiceErrorMsg struct {
	Operation string
	Error     error
}

// SuggestionsReceivedMsg contains received suggestions
type SuggestionsReceivedMsg struct {
	Suggestions []suggestions.Suggestion
	Metadata    map[string]interface{}
	FromCache   bool
	Latency     time.Duration
	TokensUsed  int
}

// CacheCleanupMsg triggers cache cleanup
type CacheCleanupMsg struct {
	Timestamp time.Time
}

// CacheCleanedMsg indicates cache was cleaned
type CacheCleanedMsg struct {
	ItemsRemoved int
}