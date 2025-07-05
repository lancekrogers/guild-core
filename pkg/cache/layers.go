// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// L1Cache represents the fastest, in-memory cache layer
type L1Cache struct {
	data     map[string]interface{}
	metadata map[string]*CacheEntry
	maxSize  int64
	curSize  int64
	mu       sync.RWMutex
	stats    *LevelStats
}

// NewL1Cache creates a new L1 cache
func NewL1Cache(maxSize int64) *L1Cache {
	return &L1Cache{
		data:     make(map[string]interface{}),
		metadata: make(map[string]*CacheEntry),
		maxSize:  maxSize,
		stats:    &LevelStats{},
	}
}

// Get retrieves a value from L1 cache
func (l1 *L1Cache) Get(key string) (interface{}, bool) {
	l1.mu.RLock()
	defer l1.mu.RUnlock()

	if entry, exists := l1.metadata[key]; exists {
		if entry.IsExpired() {
			// Remove expired entry
			l1.mu.RUnlock()
			l1.mu.Lock()
			delete(l1.data, key)
			delete(l1.metadata, key)
			l1.curSize -= entry.Size
			l1.mu.Unlock()
			l1.mu.RLock()
			l1.stats.Misses++
			return nil, false
		}

		entry.UpdateAccess()
		l1.stats.Hits++
		return l1.data[key], true
	}

	l1.stats.Misses++
	return nil, false
}

// Set stores a value in L1 cache
func (l1 *L1Cache) Set(key string, value interface{}) error {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Size:       calculateItemSize(value),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		AccessCount: 1,
	}

	// Check if we have space
	if l1.curSize+entry.Size > l1.maxSize {
		return gerror.New(gerror.ErrCodeResourceExhausted, "L1 cache full", nil).
			WithComponent("l1-cache").
			WithOperation("Set")
	}

	// Remove existing entry if updating
	if existingEntry, exists := l1.metadata[key]; exists {
		l1.curSize -= existingEntry.Size
	}

	l1.data[key] = value
	l1.metadata[key] = entry
	l1.curSize += entry.Size

	return nil
}

// Delete removes a value from L1 cache
func (l1 *L1Cache) Delete(key string) {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	if entry, exists := l1.metadata[key]; exists {
		delete(l1.data, key)
		delete(l1.metadata, key)
		l1.curSize -= entry.Size
	}
}

// Clear removes all entries from L1 cache
func (l1 *L1Cache) Clear() {
	l1.mu.Lock()
	defer l1.mu.Unlock()

	l1.data = make(map[string]interface{})
	l1.metadata = make(map[string]*CacheEntry)
	l1.curSize = 0
}

// GetStats returns L1 cache statistics
func (l1 *L1Cache) GetStats() *LevelStats {
	l1.mu.RLock()
	defer l1.mu.RUnlock()

	total := l1.stats.Hits + l1.stats.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(l1.stats.Hits) / float64(total)
	}

	return &LevelStats{
		Hits:       l1.stats.Hits,
		Misses:     l1.stats.Misses,
		HitRate:    hitRate,
		Size:       l1.curSize,
		EntryCount: len(l1.data),
		Evictions:  l1.stats.Evictions,
	}
}

// GetEntries returns all cache entries for eviction algorithms
func (l1 *L1Cache) GetEntries() map[string]*CacheEntry {
	l1.mu.RLock()
	defer l1.mu.RUnlock()

	entries := make(map[string]*CacheEntry)
	for k, v := range l1.metadata {
		entries[k] = v
	}
	return entries
}

// L2Cache represents the disk-based cache layer
type L2Cache struct {
	data     map[string]interface{}
	metadata map[string]*CacheEntry
	maxSize  int64
	curSize  int64
	mu       sync.RWMutex
	stats    *LevelStats
}

// NewL2Cache creates a new L2 cache
func NewL2Cache(maxSize int64) *L2Cache {
	return &L2Cache{
		data:     make(map[string]interface{}),
		metadata: make(map[string]*CacheEntry),
		maxSize:  maxSize,
		stats:    &LevelStats{},
	}
}

// Get retrieves a value from L2 cache
func (l2 *L2Cache) Get(key string) (interface{}, bool) {
	l2.mu.RLock()
	defer l2.mu.RUnlock()

	if entry, exists := l2.metadata[key]; exists {
		if entry.IsExpired() {
			// Remove expired entry
			l2.mu.RUnlock()
			l2.mu.Lock()
			delete(l2.data, key)
			delete(l2.metadata, key)
			l2.curSize -= entry.Size
			l2.mu.Unlock()
			l2.mu.RLock()
			l2.stats.Misses++
			return nil, false
		}

		entry.UpdateAccess()
		l2.stats.Hits++
		return l2.data[key], true
	}

	l2.stats.Misses++
	return nil, false
}

// Set stores a value in L2 cache
func (l2 *L2Cache) Set(key string, value interface{}) error {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Size:       calculateItemSize(value),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		AccessCount: 1,
	}

	// Check if we have space
	if l2.curSize+entry.Size > l2.maxSize {
		return gerror.New(gerror.ErrCodeResourceExhausted, "L2 cache full", nil).
			WithComponent("l2-cache").
			WithOperation("Set")
	}

	// Remove existing entry if updating
	if existingEntry, exists := l2.metadata[key]; exists {
		l2.curSize -= existingEntry.Size
	}

	l2.data[key] = value
	l2.metadata[key] = entry
	l2.curSize += entry.Size

	return nil
}

// Delete removes a value from L2 cache
func (l2 *L2Cache) Delete(key string) {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	if entry, exists := l2.metadata[key]; exists {
		delete(l2.data, key)
		delete(l2.metadata, key)
		l2.curSize -= entry.Size
	}
}

// Clear removes all entries from L2 cache
func (l2 *L2Cache) Clear() {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	l2.data = make(map[string]interface{})
	l2.metadata = make(map[string]*CacheEntry)
	l2.curSize = 0
}

// GetStats returns L2 cache statistics
func (l2 *L2Cache) GetStats() *LevelStats {
	l2.mu.RLock()
	defer l2.mu.RUnlock()

	total := l2.stats.Hits + l2.stats.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(l2.stats.Hits) / float64(total)
	}

	return &LevelStats{
		Hits:       l2.stats.Hits,
		Misses:     l2.stats.Misses,
		HitRate:    hitRate,
		Size:       l2.curSize,
		EntryCount: len(l2.data),
		Evictions:  l2.stats.Evictions,
	}
}

// GetEntries returns all cache entries for eviction algorithms
func (l2 *L2Cache) GetEntries() map[string]*CacheEntry {
	l2.mu.RLock()
	defer l2.mu.RUnlock()

	entries := make(map[string]*CacheEntry)
	for k, v := range l2.metadata {
		entries[k] = v
	}
	return entries
}

// RedisCache represents the distributed cache layer
type RedisCache struct {
	client RedisClient
	mu     sync.RWMutex
}

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(url string) (*RedisCache, error) {
	// In a real implementation, this would connect to Redis
	// For now, we'll return a mock implementation
	return &RedisCache{
		client: &MockRedisClient{},
	}, nil
}

// Get retrieves a value from Redis cache
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	value, err := rc.client.Get(ctx, key)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "redis get failed").
			WithComponent("redis-cache").
			WithOperation("Get")
	}
	return value, nil
}

// Set stores a value in Redis cache
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Convert value to string (in real implementation, use JSON or other serialization)
	valueStr := fmt.Sprintf("%v", value)
	
	err := rc.client.Set(ctx, key, valueStr, ttl)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "redis set failed").
			WithComponent("redis-cache").
			WithOperation("Set")
	}
	return nil
}

// Delete removes a value from Redis cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	err := rc.client.Delete(ctx, key)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "redis delete failed").
			WithComponent("redis-cache").
			WithOperation("Delete")
	}
	return nil
}

// Clear removes all entries from Redis cache
func (rc *RedisCache) Clear(ctx context.Context) error {
	err := rc.client.Clear(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "redis clear failed").
			WithComponent("redis-cache").
			WithOperation("Clear")
	}
	return nil
}

// MockRedisClient provides a mock implementation for testing
type MockRedisClient struct {
	data map[string]string
	mu   sync.RWMutex
}

// Get retrieves a value from mock Redis
func (mrc *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	mrc.mu.RLock()
	defer mrc.mu.RUnlock()
	
	if mrc.data == nil {
		mrc.data = make(map[string]string)
	}
	
	value, exists := mrc.data[key]
	if !exists {
		return "", gerror.New(gerror.ErrCodeNotFound, "key not found", nil)
	}
	return value, nil
}

// Set stores a value in mock Redis
func (mrc *MockRedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()
	
	if mrc.data == nil {
		mrc.data = make(map[string]string)
	}
	
	mrc.data[key] = value
	return nil
}

// Delete removes a value from mock Redis
func (mrc *MockRedisClient) Delete(ctx context.Context, key string) error {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()
	
	if mrc.data == nil {
		return nil
	}
	
	delete(mrc.data, key)
	return nil
}

// Clear removes all entries from mock Redis
func (mrc *MockRedisClient) Clear(ctx context.Context) error {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()
	
	mrc.data = make(map[string]string)
	return nil
}

// calculateItemSize estimates the size of an item
func calculateItemSize(value interface{}) int64 {
	// This is a simplified size calculation
	// In production, you might want more accurate size calculation
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	default:
		return 64 // Default estimate
	}
}