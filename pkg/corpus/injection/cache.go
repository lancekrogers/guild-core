// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package injection

import (
	"sync"
	"time"
)

// CacheEntry represents a cached injection result
type CacheEntry struct {
	Value       *InjectedPrompt `json:"value"`
	ExpiresAt   time.Time       `json:"expires_at"`
	AccessedAt  time.Time       `json:"accessed_at"`
	AccessCount int             `json:"access_count"`
}

// ContextCache implements a thread-safe LRU cache with TTL for injection results
type ContextCache struct {
	entries map[string]*CacheEntry
	ttl     time.Duration
	maxSize int
	mu      sync.RWMutex
}

// NewContextCache creates a new context cache with the specified TTL
func NewContextCache(ttl time.Duration) *ContextCache {
	return &ContextCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: 1000, // Default max cache size
		mu:      sync.RWMutex{},
	}
}

// NewContextCacheWithSize creates a new context cache with specified TTL and max size
func NewContextCacheWithSize(ttl time.Duration, maxSize int) *ContextCache {
	return &ContextCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		mu:      sync.RWMutex{},
	}
}

// Get retrieves a value from the cache
func (cc *ContextCache) Get(key string) *InjectedPrompt {
	cc.mu.RLock()
	entry, exists := cc.entries[key]
	cc.mu.RUnlock()

	if !exists {
		return nil
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		cc.mu.Lock()
		delete(cc.entries, key)
		cc.mu.Unlock()
		return nil
	}

	// Update access metadata
	cc.mu.Lock()
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	cc.mu.Unlock()

	return entry.Value
}

// Set stores a value in the cache
func (cc *ContextCache) Set(key string, value *InjectedPrompt) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Check if we need to evict entries
	if len(cc.entries) >= cc.maxSize {
		cc.evictLRU()
	}

	// Create new entry
	entry := &CacheEntry{
		Value:       value,
		ExpiresAt:   time.Now().Add(cc.ttl),
		AccessedAt:  time.Now(),
		AccessCount: 1,
	}

	cc.entries[key] = entry
}

// Delete removes a value from the cache
func (cc *ContextCache) Delete(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.entries, key)
}

// Clear removes all entries from the cache
func (cc *ContextCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.entries = make(map[string]*CacheEntry)
}

// Size returns the current number of entries in the cache
func (cc *ContextCache) Size() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return len(cc.entries)
}

// evictLRU removes the least recently used entry
func (cc *ContextCache) evictLRU() {
	if len(cc.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range cc.entries {
		if first || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(cc.entries, oldestKey)
	}
}

// CleanupExpired removes all expired entries from the cache
func (cc *ContextCache) CleanupExpired() int {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for key, entry := range cc.entries {
		if now.After(entry.ExpiresAt) {
			delete(cc.entries, key)
			expiredCount++
		}
	}

	return expiredCount
}

// Stats returns cache statistics
func (cc *ContextCache) Stats() CacheStats {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	stats := CacheStats{
		Size:    len(cc.entries),
		MaxSize: cc.maxSize,
		TTL:     cc.ttl,
	}

	now := time.Now()
	for _, entry := range cc.entries {
		stats.TotalAccesses += entry.AccessCount
		if now.After(entry.ExpiresAt) {
			stats.ExpiredEntries++
		}
	}

	return stats
}

// CacheStats contains statistics about the cache
type CacheStats struct {
	Size           int           `json:"size"`
	MaxSize        int           `json:"max_size"`
	TTL            time.Duration `json:"ttl"`
	TotalAccesses  int           `json:"total_accesses"`
	ExpiredEntries int           `json:"expired_entries"`
}

// StartCleanupRoutine starts a background goroutine to periodically clean expired entries
func (cc *ContextCache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			cc.CleanupExpired()
		}
	}()
}
