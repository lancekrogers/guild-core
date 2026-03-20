// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	// Global rate limit
	GlobalRate  rate.Limit
	GlobalBurst int

	// Per-agent rate limit
	AgentRate  rate.Limit
	AgentBurst int

	// Cleanup settings
	MaxAgents        int
	CleanupThreshold time.Duration

	// Callbacks
	OnLimitHit func(agentID string, limitType string)
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	// Configuration (immutable)
	config RateLimiterConfig

	// Global limiter
	global *rate.Limiter

	// Per-agent limiters
	mu       sync.RWMutex
	agents   map[string]*agentLimiter
	lru      *list.List
	lruIndex map[string]*list.Element
}

// agentLimiter tracks rate limiting for a specific agent
type agentLimiter struct {
	limiter    *rate.Limiter
	agentID    string
	lastAccess time.Time
	tokenUsed  int64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	// Set defaults
	if config.GlobalRate <= 0 {
		config.GlobalRate = 1000
	}
	if config.GlobalBurst <= 0 {
		config.GlobalBurst = 100
	}
	if config.AgentRate <= 0 {
		config.AgentRate = 100
	}
	if config.AgentBurst <= 0 {
		config.AgentBurst = 10
	}
	if config.MaxAgents <= 0 {
		config.MaxAgents = 1000
	}
	if config.CleanupThreshold <= 0 {
		config.CleanupThreshold = time.Hour
	}

	return &RateLimiter{
		config:   config,
		global:   rate.NewLimiter(config.GlobalRate, config.GlobalBurst),
		agents:   make(map[string]*agentLimiter),
		lru:      list.New(),
		lruIndex: make(map[string]*list.Element),
	}
}

// Allow checks if a request is allowed without blocking
func (rl *RateLimiter) Allow(ctx context.Context, agentID string) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled").
			WithComponent("rate_limiter")
	}

	// Check global limit
	if !rl.global.Allow() {
		if rl.config.OnLimitHit != nil {
			rl.config.OnLimitHit(agentID, "global")
		}
		return gerror.New(gerror.ErrCodeResourceExhausted, "global rate limit exceeded", nil).
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("limit_type", "global").
			WithDetails("rate", float64(rl.config.GlobalRate)).
			WithDetails("burst", rl.config.GlobalBurst)
	}

	// Get or create agent limiter
	limiter, err := rl.getAgentLimiter(agentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get agent limiter").
			WithDetails("agent_id", agentID).
			WithComponent("rate_limiter")
	}

	// Check agent limit
	if !limiter.limiter.Allow() {
		if rl.config.OnLimitHit != nil {
			rl.config.OnLimitHit(agentID, "agent")
		}
		return gerror.New(gerror.ErrCodeResourceExhausted, "agent rate limit exceeded", nil).
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("limit_type", "agent").
			WithDetails("rate", float64(rl.config.AgentRate)).
			WithDetails("burst", rl.config.AgentBurst)
	}

	// Update access time and token count
	rl.updateAgentAccess(agentID)

	return nil
}

// Wait blocks until a request is allowed or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context, agentID string) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled").
			WithComponent("rate_limiter")
	}

	// Wait for global limit
	if err := rl.global.Wait(ctx); err != nil {
		if rl.config.OnLimitHit != nil {
			rl.config.OnLimitHit(agentID, "global")
		}
		return gerror.Wrap(err, gerror.ErrCodeTimeout, "global rate limit wait failed").
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("limit_type", "global")
	}

	// Get or create agent limiter
	limiter, err := rl.getAgentLimiter(agentID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get agent limiter").
			WithDetails("agent_id", agentID).
			WithComponent("rate_limiter")
	}

	// Wait for agent limit
	if err := limiter.limiter.Wait(ctx); err != nil {
		if rl.config.OnLimitHit != nil {
			rl.config.OnLimitHit(agentID, "agent")
		}
		return gerror.Wrap(err, gerror.ErrCodeTimeout, "agent rate limit wait failed").
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("limit_type", "agent")
	}

	// Update access time and token count
	rl.updateAgentAccess(agentID)

	return nil
}

// Reserve reserves tokens for future use
func (rl *RateLimiter) Reserve(ctx context.Context, agentID string, tokens int) (*rate.Reservation, error) {
	// Check context first
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled").
			WithComponent("rate_limiter")
	}

	// Reserve from global limiter
	globalReserve := rl.global.ReserveN(time.Now(), tokens)
	if !globalReserve.OK() {
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "global rate limit cannot reserve tokens", nil).
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("tokens", tokens).
			WithDetails("limit_type", "global")
	}

	// Get or create agent limiter
	limiter, err := rl.getAgentLimiter(agentID)
	if err != nil {
		globalReserve.Cancel() // Cancel global reservation
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get agent limiter").
			WithDetails("agent_id", agentID).
			WithComponent("rate_limiter")
	}

	// Reserve from agent limiter
	agentReserve := limiter.limiter.ReserveN(time.Now(), tokens)
	if !agentReserve.OK() {
		globalReserve.Cancel() // Cancel global reservation
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "agent rate limit cannot reserve tokens", nil).
			WithComponent("rate_limiter").
			WithDetails("agent_id", agentID).
			WithDetails("tokens", tokens).
			WithDetails("limit_type", "agent")
	}

	// Update token count
	rl.mu.Lock()
	limiter.tokenUsed += int64(tokens)
	rl.mu.Unlock()

	// Return the agent reservation (caller should check delay)
	return agentReserve, nil
}

// getAgentLimiter gets or creates a limiter for an agent
func (rl *RateLimiter) getAgentLimiter(agentID string) (*agentLimiter, error) {
	// Fast path - read lock
	rl.mu.RLock()
	if limiter, exists := rl.agents[agentID]; exists {
		rl.mu.RUnlock()
		return limiter, nil
	}
	rl.mu.RUnlock()

	// Slow path - write lock
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.agents[agentID]; exists {
		// Move to front of LRU
		rl.lru.MoveToFront(rl.lruIndex[agentID])
		return limiter, nil
	}

	// Check if we need to evict
	if len(rl.agents) >= rl.config.MaxAgents {
		if err := rl.evictOldest(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeResourceExhausted, "failed to evict oldest agent").
				WithComponent("rate_limiter")
		}
	}

	// Create new limiter
	limiter := &agentLimiter{
		limiter:    rate.NewLimiter(rl.config.AgentRate, rl.config.AgentBurst),
		agentID:    agentID,
		lastAccess: time.Now(),
		tokenUsed:  0,
	}

	// Add to map and LRU
	rl.agents[agentID] = limiter
	element := rl.lru.PushFront(agentID)
	rl.lruIndex[agentID] = element

	return limiter, nil
}

// updateAgentAccess updates the last access time for an agent
func (rl *RateLimiter) updateAgentAccess(agentID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.agents[agentID]; exists {
		limiter.lastAccess = time.Now()
		// Move to front of LRU
		if element, exists := rl.lruIndex[agentID]; exists {
			rl.lru.MoveToFront(element)
		}
	}
}

// evictOldest removes the least recently used agent
func (rl *RateLimiter) evictOldest() error {
	if rl.lru.Len() == 0 {
		return gerror.New(gerror.ErrCodeInternal, "no agents to evict", nil).
			WithComponent("rate_limiter")
	}

	// Get oldest element
	oldest := rl.lru.Back()
	if oldest == nil {
		return gerror.New(gerror.ErrCodeInternal, "LRU list corrupted", nil).
			WithComponent("rate_limiter")
	}

	agentID := oldest.Value.(string)

	// Remove from maps
	delete(rl.agents, agentID)
	delete(rl.lruIndex, agentID)
	rl.lru.Remove(oldest)

	return nil
}

// Cleanup removes inactive agent limiters
func (rl *RateLimiter) Cleanup() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.config.CleanupThreshold)
	removed := 0

	// Iterate through agents and remove inactive ones
	for agentID, limiter := range rl.agents {
		if limiter.lastAccess.Before(threshold) {
			// Remove from maps
			delete(rl.agents, agentID)
			if element, exists := rl.lruIndex[agentID]; exists {
				rl.lru.Remove(element)
				delete(rl.lruIndex, agentID)
			}
			removed++
		}
	}

	return nil
}

// GetUsageStats returns usage statistics for all agents
func (rl *RateLimiter) GetUsageStats() map[string]float64 {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := make(map[string]float64)

	// Global usage (approximate based on available tokens)
	globalTokens := float64(rl.global.Tokens())
	globalMax := float64(rl.config.GlobalBurst)
	if globalMax > 0 {
		stats["global"] = 1.0 - (globalTokens / globalMax)
	}

	// Per-agent usage
	for agentID, limiter := range rl.agents {
		agentTokens := float64(limiter.limiter.Tokens())
		agentMax := float64(rl.config.AgentBurst)
		if agentMax > 0 {
			stats[agentID] = 1.0 - (agentTokens / agentMax)
		}
	}

	return stats
}

// Statistics returns detailed statistics
func (rl *RateLimiter) Statistics() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := map[string]interface{}{
		"global_rate":       float64(rl.config.GlobalRate),
		"global_burst":      rl.config.GlobalBurst,
		"agent_rate":        float64(rl.config.AgentRate),
		"agent_burst":       rl.config.AgentBurst,
		"active_agents":     len(rl.agents),
		"max_agents":        rl.config.MaxAgents,
		"cleanup_threshold": rl.config.CleanupThreshold.String(),
	}

	// Calculate total token usage
	var totalTokens int64
	for _, limiter := range rl.agents {
		totalTokens += limiter.tokenUsed
	}
	stats["total_tokens_used"] = totalTokens

	return stats
}

// Reset resets all rate limiters
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Reset global limiter
	rl.global = rate.NewLimiter(rl.config.GlobalRate, rl.config.GlobalBurst)

	// Clear all agent limiters
	rl.agents = make(map[string]*agentLimiter)
	rl.lru = list.New()
	rl.lruIndex = make(map[string]*list.Element)
}
