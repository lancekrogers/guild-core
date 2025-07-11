// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package preferences

import (
	"sync"
	"time"
)

// cacheEntry represents a cached preference value
type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// PreferenceCache provides an in-memory cache for preferences
type PreferenceCache struct {
	items       map[string]cacheEntry
	mu          sync.RWMutex
	defaultTTL  time.Duration
	cleanupTick time.Duration
	stopCleanup chan struct{}
}

// NewPreferenceCache creates a new preference cache
func NewPreferenceCache(defaultTTL, cleanupInterval time.Duration) *PreferenceCache {
	cache := &PreferenceCache{
		items:       make(map[string]cacheEntry),
		defaultTTL:  defaultTTL,
		cleanupTick: cleanupInterval,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a value from the cache
func (c *PreferenceCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiration) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache with the default TTL
func (c *PreferenceCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *PreferenceCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (c *PreferenceCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all values from the cache
func (c *PreferenceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheEntry)
}

// Size returns the number of items in the cache
func (c *PreferenceCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Stop stops the cleanup goroutine
func (c *PreferenceCache) Stop() {
	close(c.stopCleanup)
}

// cleanupLoop periodically removes expired entries
func (c *PreferenceCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries
func (c *PreferenceCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.items {
		if now.After(entry.expiration) {
			delete(c.items, key)
		}
	}
}
