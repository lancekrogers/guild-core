// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// QueryCache implements an LRU cache for query results
type QueryCache struct {
	mu        sync.RWMutex
	config    QueryCacheConfig
	entries   map[string]*CacheEntry
	lru       *lruList
	hitCount  int64
	missCount int64
}

// QueryCacheConfig configures the query cache
type QueryCacheConfig struct {
	MaxEntries      int
	MaxMemoryBytes  int64
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

// DefaultQueryCacheConfig returns default cache configuration
func DefaultQueryCacheConfig() QueryCacheConfig {
	return QueryCacheConfig{
		MaxEntries:      1000,
		MaxMemoryBytes:  100 * 1024 * 1024, // 100MB
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
}

// CacheEntry represents a cached query result
type CacheEntry struct {
	Key         string
	Query       string
	Args        []interface{}
	Result      interface{}
	Size        int64
	CreatedAt   time.Time
	ExpiresAt   time.Time
	AccessCount int64
	LastAccess  time.Time
	prev        *CacheEntry
	next        *CacheEntry
}

// lruList manages LRU ordering
type lruList struct {
	head *CacheEntry
	tail *CacheEntry
	size int
}

// NewQueryCache creates a new query cache
func NewQueryCache(config QueryCacheConfig) (*QueryCache, error) {
	if config.MaxEntries <= 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "max entries must be positive", nil).
			WithComponent("QueryCache")
	}

	cache := &QueryCache{
		config:  config,
		entries: make(map[string]*CacheEntry),
		lru:     &lruList{},
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache, nil
}

// Get retrieves a cached query result
func (c *QueryCache) Get(ctx context.Context, query string, args []interface{}) (interface{}, bool) {
	if err := ctx.Err(); err != nil {
		return nil, false
	}

	key := c.generateKey(query, args)

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.missCount++
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.removeLocked(key)
		c.missCount++
		return nil, false
	}

	// Update access stats
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.hitCount++

	// Move to front of LRU
	c.lru.moveToFront(entry)

	return entry.Result, true
}

// Set stores a query result in cache
func (c *QueryCache) Set(ctx context.Context, query string, args []interface{}, result interface{}, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("QueryCache").
			WithOperation("Set")
	}

	key := c.generateKey(query, args)

	// Estimate size (simplified - in production would be more accurate)
	size := int64(len(query)) + int64(len(args)*8)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.config.MaxEntries {
		c.evictLRU()
	}

	// Create new entry
	now := time.Now()
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	entry := &CacheEntry{
		Key:         key,
		Query:       query,
		Args:        args,
		Result:      result,
		Size:        size,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		AccessCount: 1,
		LastAccess:  now,
	}

	// Add to cache
	c.entries[key] = entry
	c.lru.pushFront(entry)

	return nil
}

// Invalidate removes entries matching a pattern
func (c *QueryCache) Invalidate(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	invalidated := 0
	for key := range c.entries {
		// Simple pattern matching - in production would support more complex patterns
		if pattern == "*" || key == pattern {
			c.removeLocked(key)
			invalidated++
		}
	}

	return invalidated
}

// Clear removes all cache entries
func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.lru = &lruList{}
	c.hitCount = 0
	c.missCount = 0
}

// Stats returns cache statistics
func (c *QueryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalSize := int64(0)
	for _, entry := range c.entries {
		totalSize += entry.Size
	}

	hitRate := float64(0)
	if total := c.hitCount + c.missCount; total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return CacheStats{
		Entries:    len(c.entries),
		SizeBytes:  totalSize,
		HitCount:   c.hitCount,
		MissCount:  c.missCount,
		HitRate:    hitRate,
		MaxEntries: c.config.MaxEntries,
		MaxMemory:  c.config.MaxMemoryBytes,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries    int
	SizeBytes  int64
	HitCount   int64
	MissCount  int64
	HitRate    float64
	MaxEntries int
	MaxMemory  int64
}

// generateKey creates a cache key from query and arguments
func (c *QueryCache) generateKey(query string, args []interface{}) string {
	h := sha256.New()
	h.Write([]byte(query))

	// Add args to hash
	for _, arg := range args {
		h.Write([]byte(fmt.Sprintf("%v", arg)))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// removeLocked removes an entry (must be called with lock held)
func (c *QueryCache) removeLocked(key string) {
	entry, exists := c.entries[key]
	if !exists {
		return
	}

	delete(c.entries, key)
	c.lru.remove(entry)
}

// evictLRU removes the least recently used entry
func (c *QueryCache) evictLRU() {
	if c.lru.tail == nil {
		return
	}

	c.removeLocked(c.lru.tail.Key)
}

// cleanupLoop periodically removes expired entries
func (c *QueryCache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *QueryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			c.removeLocked(key)
		}
	}
}

// LRU list operations

func (l *lruList) pushFront(entry *CacheEntry) {
	entry.prev = nil
	entry.next = l.head

	if l.head != nil {
		l.head.prev = entry
	}
	l.head = entry

	if l.tail == nil {
		l.tail = entry
	}

	l.size++
}

func (l *lruList) remove(entry *CacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		l.head = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		l.tail = entry.prev
	}

	entry.prev = nil
	entry.next = nil
	l.size--
}

func (l *lruList) moveToFront(entry *CacheEntry) {
	if entry == l.head {
		return
	}

	l.remove(entry)
	l.pushFront(entry)
}
