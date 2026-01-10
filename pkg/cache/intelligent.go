// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package cache provides intelligent multi-layer caching capabilities
// for the Guild framework, featuring predictive algorithms and adaptive eviction.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Cache error codes for the Guild framework
const (
	ErrCodeCacheMiss         = "CACHE-1001"
	ErrCodeCacheCorruption   = "CACHE-1002"
	ErrCodeEvictionFailed    = "CACHE-1003"
	ErrCodePredictionFailed  = "CACHE-1004"
	ErrCodeSizeLimitExceeded = "CACHE-1005"
)

// CacheLevel represents different cache layers
type CacheLevel int

const (
	CacheLevelL1 CacheLevel = iota
	CacheLevelL2
	CacheLevelDistributed
)

// String returns the string representation of CacheLevel
func (cl CacheLevel) String() string {
	switch cl {
	case CacheLevelL1:
		return "L1"
	case CacheLevelL2:
		return "L2"
	case CacheLevelDistributed:
		return "Distributed"
	default:
		return "Unknown"
	}
}

// IntelligentCache provides multi-layer caching with predictive algorithms
type IntelligentCache struct {
	l1Cache     *L1Cache
	l2Cache     *L2Cache
	distributed *RedisCache
	predictor   *AccessPredictor
	eviction    EvictionPolicy
	stats       *CacheStats
	mu          sync.RWMutex
	config      *CacheConfig
}

// CacheConfig holds configuration for the intelligent cache
type CacheConfig struct {
	L1MaxSize        int64         `json:"l1_max_size"`
	L2MaxSize        int64         `json:"l2_max_size"`
	DefaultTTL       time.Duration `json:"default_ttl"`
	PredictionWindow time.Duration `json:"prediction_window"`
	WarmupThreshold  float64       `json:"warmup_threshold"`
	EvictionPolicy   string        `json:"eviction_policy"`
	DistributedURL   string        `json:"distributed_url"`
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		L1MaxSize:        100 * 1024 * 1024,  // 100MB
		L2MaxSize:        1024 * 1024 * 1024, // 1GB
		DefaultTTL:       time.Hour,
		PredictionWindow: time.Hour * 24,
		WarmupThreshold:  0.7,
		EvictionPolicy:   "adaptive",
	}
}

// NewIntelligentCache creates a new intelligent cache system
func NewIntelligentCache(config *CacheConfig) (*IntelligentCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	ic := &IntelligentCache{
		l1Cache:   NewL1Cache(config.L1MaxSize),
		l2Cache:   NewL2Cache(config.L2MaxSize),
		predictor: NewAccessPredictor(config.PredictionWindow),
		stats:     NewCacheStats(),
		config:    config,
	}

	// Initialize eviction policy
	switch config.EvictionPolicy {
	case "lru":
		ic.eviction = NewLRUPolicy()
	case "lfu":
		ic.eviction = NewLFUPolicy()
	case "adaptive":
		ic.eviction = NewAdaptiveEvictionPolicy()
	default:
		ic.eviction = NewAdaptiveEvictionPolicy()
	}

	// Initialize distributed cache if configured
	if config.DistributedURL != "" {
		distCache, err := NewRedisCache(config.DistributedURL)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to distributed cache").
				WithComponent("intelligent-cache").
				WithOperation("NewIntelligentCache")
		}
		ic.distributed = distCache
	}

	return ic, nil
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key         string                 `json:"key"`
	Value       interface{}            `json:"value"`
	Size        int64                  `json:"size"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	TTL         time.Duration          `json:"ttl"`
	Cost        float64                `json:"cost"`
	Metadata    map[string]interface{} `json:"metadata"`
	Level       CacheLevel             `json:"level"`
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	if ce.TTL == 0 {
		return false // No expiration
	}
	return time.Since(ce.CreatedAt) > ce.TTL
}

// UpdateAccess updates access statistics
func (ce *CacheEntry) UpdateAccess() {
	ce.AccessCount++
	ce.LastAccess = time.Now()
}

// Get retrieves a value from the cache with intelligent promotion
func (ic *IntelligentCache) Get(ctx context.Context, key string) (interface{}, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache get cancelled").
			WithComponent("intelligent-cache").
			WithOperation("Get")
	default:
	}

	// Record access for prediction
	ic.recordAccess(key)

	// Check L1 cache first (fastest)
	if value, found := ic.l1Cache.Get(key); found {
		ic.stats.RecordHit(CacheLevelL1)
		return value, nil
	}

	// Check L2 cache
	if value, found := ic.l2Cache.Get(key); found {
		ic.stats.RecordHit(CacheLevelL2)

		// Promote to L1 if predictor suggests it's hot
		if ic.predictor.ShouldPromote(key) {
			ic.l1Cache.Set(key, value)
		}

		return value, nil
	}

	// Check distributed cache if available
	if ic.distributed != nil {
		if value, err := ic.distributed.Get(ctx, key); err == nil {
			ic.stats.RecordHit(CacheLevelDistributed)

			// Cache locally based on prediction
			ic.cacheLocally(key, value)

			return value, nil
		}
	}

	ic.stats.RecordMiss()
	return nil, gerror.New(gerror.ErrCodeNotFound, "cache miss", nil).
		WithComponent("intelligent-cache").
		WithOperation("Get")
}

// Set stores a value in the cache with intelligent placement
func (ic *IntelligentCache) Set(ctx context.Context, key string, value interface{}, opts ...CacheOption) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache set cancelled").
			WithComponent("intelligent-cache").
			WithOperation("Set")
	default:
	}

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Size:       ic.calculateSize(value),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		TTL:        ic.config.DefaultTTL,
		Metadata:   make(map[string]interface{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(entry)
	}

	// Determine optimal cache level
	level := ic.determineCacheLevel(entry)
	entry.Level = level

	switch level {
	case CacheLevelL1:
		return ic.setL1(entry)
	case CacheLevelL2:
		return ic.setL2(entry)
	case CacheLevelDistributed:
		if ic.distributed != nil {
			return ic.distributed.Set(ctx, key, value, entry.TTL)
		}
		// Fallback to L2 if distributed cache is not available
		return ic.setL2(entry)
	default:
		return ic.setL1(entry)
	}
}

// setL1 sets an entry in L1 cache with eviction handling
func (ic *IntelligentCache) setL1(entry *CacheEntry) error {
	if err := ic.l1Cache.Set(entry.Key, entry.Value); err != nil {
		// L1 full, evict based on policy
		if err := ic.evictFromL1(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeResourceExhausted, "failed to evict from L1 cache").
				WithComponent("intelligent-cache").
				WithOperation("setL1")
		}
		// Retry after eviction
		return ic.l1Cache.Set(entry.Key, entry.Value)
	}
	return nil
}

// setL2 sets an entry in L2 cache with eviction handling
func (ic *IntelligentCache) setL2(entry *CacheEntry) error {
	if err := ic.l2Cache.Set(entry.Key, entry.Value); err != nil {
		// L2 full, evict based on policy
		if err := ic.evictFromL2(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeResourceExhausted, "failed to evict from L2 cache").
				WithComponent("intelligent-cache").
				WithOperation("setL2")
		}
		// Retry after eviction
		return ic.l2Cache.Set(entry.Key, entry.Value)
	}
	return nil
}

// determineCacheLevel determines the optimal cache level for an entry
func (ic *IntelligentCache) determineCacheLevel(entry *CacheEntry) CacheLevel {
	// Check entry size constraints
	if entry.Size > ic.config.L1MaxSize/10 {
		return CacheLevelL2 // Large objects go to L2
	}

	// Check access prediction
	prediction := ic.predictor.PredictNextAccess(entry.Key)
	if prediction.Probability > ic.config.WarmupThreshold {
		return CacheLevelL1 // Hot data goes to L1
	}

	// Check if this is shared data
	if shared, ok := entry.Metadata["shared"].(bool); ok && shared {
		return CacheLevelDistributed
	}

	// Default to L2 for warm data
	return CacheLevelL2
}

// evictFromL1 evicts entries from L1 cache
func (ic *IntelligentCache) evictFromL1() error {
	requiredSpace := ic.config.L1MaxSize / 10 // Evict 10% of capacity
	candidates := ic.eviction.Evict(ic.l1Cache, requiredSpace)

	for _, key := range candidates {
		// Move to L2 before evicting from L1
		if value, found := ic.l1Cache.Get(key); found {
			ic.l2Cache.Set(key, value)
		}
		ic.l1Cache.Delete(key)
	}

	return nil
}

// evictFromL2 evicts entries from L2 cache
func (ic *IntelligentCache) evictFromL2() error {
	requiredSpace := ic.config.L2MaxSize / 10 // Evict 10% of capacity
	candidates := ic.eviction.Evict(ic.l2Cache, requiredSpace)

	for _, key := range candidates {
		ic.l2Cache.Delete(key)
	}

	return nil
}

// recordAccess records an access for prediction algorithms
func (ic *IntelligentCache) recordAccess(key string) {
	ic.predictor.RecordAccess(key, time.Now())
}

// cacheLocally decides where to cache data retrieved from distributed cache
func (ic *IntelligentCache) cacheLocally(key string, value interface{}) {
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Size:       ic.calculateSize(value),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		TTL:        ic.config.DefaultTTL,
	}

	level := ic.determineCacheLevel(entry)
	switch level {
	case CacheLevelL1:
		ic.l1Cache.Set(key, value)
	case CacheLevelL2:
		ic.l2Cache.Set(key, value)
	}
}

// calculateSize estimates the size of a value in bytes
func (ic *IntelligentCache) calculateSize(value interface{}) int64 {
	if value == nil {
		return 8 // Size of pointer
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return int64(len(v.String()))
	case reflect.Slice, reflect.Array:
		return int64(v.Len()) * 8 // Rough estimate
	case reflect.Map:
		return int64(v.Len()) * 16 // Rough estimate for map entries
	case reflect.Struct:
		return int64(v.Type().Size())
	case reflect.Ptr:
		if !v.IsNil() {
			return ic.calculateSize(v.Elem().Interface())
		}
		return 8
	default:
		// Try JSON marshalling for complex types
		if data, err := json.Marshal(value); err == nil {
			return int64(len(data))
		}
		return 64 // Default estimate
	}
}

// Delete removes an entry from all cache levels
func (ic *IntelligentCache) Delete(ctx context.Context, key string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache delete cancelled").
			WithComponent("intelligent-cache").
			WithOperation("Delete")
	default:
	}

	// Delete from all levels
	ic.l1Cache.Delete(key)
	ic.l2Cache.Delete(key)

	if ic.distributed != nil {
		if err := ic.distributed.Delete(ctx, key); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to delete from distributed cache: %v\n", err)
		}
	}

	return nil
}

// Clear removes all entries from the cache
func (ic *IntelligentCache) Clear(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cache clear cancelled").
			WithComponent("intelligent-cache").
			WithOperation("Clear")
	default:
	}

	ic.l1Cache.Clear()
	ic.l2Cache.Clear()

	if ic.distributed != nil {
		if err := ic.distributed.Clear(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to clear distributed cache").
				WithComponent("intelligent-cache").
				WithOperation("Clear")
		}
	}

	return nil
}

// GetStats returns comprehensive cache statistics
func (ic *IntelligentCache) GetStats() *CacheStatistics {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	return &CacheStatistics{
		L1Stats:            ic.l1Cache.GetStats(),
		L2Stats:            ic.l2Cache.GetStats(),
		OverallStats:       ic.stats.GetOverallStats(),
		PredictionAccuracy: ic.predictor.GetAccuracy(),
		Timestamp:          time.Now(),
	}
}

// CacheStatistics aggregates statistics from all cache levels
type CacheStatistics struct {
	L1Stats            *LevelStats   `json:"l1_stats"`
	L2Stats            *LevelStats   `json:"l2_stats"`
	OverallStats       *OverallStats `json:"overall_stats"`
	PredictionAccuracy float64       `json:"prediction_accuracy"`
	Timestamp          time.Time     `json:"timestamp"`
}

// LevelStats contains statistics for a single cache level
type LevelStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	Size       int64   `json:"size"`
	EntryCount int     `json:"entry_count"`
	Evictions  int64   `json:"evictions"`
}

// OverallStats contains overall cache statistics
type OverallStats struct {
	TotalHits      int64         `json:"total_hits"`
	TotalMisses    int64         `json:"total_misses"`
	OverallHitRate float64       `json:"overall_hit_rate"`
	MemoryUsage    int64         `json:"memory_usage"`
	Uptime         time.Duration `json:"uptime"`
}

// CacheOption allows customization of cache entries
type CacheOption func(*CacheEntry)

// WithTTL sets the time-to-live for a cache entry
func WithTTL(ttl time.Duration) CacheOption {
	return func(entry *CacheEntry) {
		entry.TTL = ttl
	}
}

// WithCost sets the cost metric for a cache entry
func WithCost(cost float64) CacheOption {
	return func(entry *CacheEntry) {
		entry.Cost = cost
	}
}

// WithMetadata sets metadata for a cache entry
func WithMetadata(key string, value interface{}) CacheOption {
	return func(entry *CacheEntry) {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]interface{})
		}
		entry.Metadata[key] = value
	}
}

// WithShared marks an entry as shared across instances
func WithShared(shared bool) CacheOption {
	return WithMetadata("shared", shared)
}

// WithLevel forces a specific cache level
func WithLevel(level CacheLevel) CacheOption {
	return func(entry *CacheEntry) {
		entry.Level = level
	}
}
