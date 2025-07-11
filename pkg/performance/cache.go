package performance

import (
	"container/list"
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Cache provides a multi-level cache with L1/L2/L3 hierarchy
type Cache[K comparable, V any] struct {
	l1     *lruCache[K, V]
	l2     *shardedCache[K, V]
	l3     CacheBackend[K, V]
	loader LoaderFunc[K, V]
	stats  CacheStats
	config CacheConfig
}

// LoaderFunc loads a value for a key when not found in cache
type LoaderFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// CacheBackend represents an external cache backend (Redis, disk, etc.)
type CacheBackend[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, error)
	Put(ctx context.Context, key K, value V, ttl time.Duration) error
	Delete(ctx context.Context, key K) error
	Clear(ctx context.Context) error
}

// CacheStats tracks cache performance across all levels
type CacheStats struct {
	L1Hits     atomic.Uint64
	L1Misses   atomic.Uint64
	L2Hits     atomic.Uint64
	L2Misses   atomic.Uint64
	L3Hits     atomic.Uint64
	L3Misses   atomic.Uint64
	Loads      atomic.Uint64
	LoadErrors atomic.Uint64
	Evictions  atomic.Uint64
	Promotions atomic.Uint64
}

// CacheStatsSnapshot represents a point-in-time snapshot of cache statistics
type CacheStatsSnapshot struct {
	L1Hits     uint64
	L1Misses   uint64
	L2Hits     uint64
	L2Misses   uint64
	L3Hits     uint64
	L3Misses   uint64
	Loads      uint64
	LoadErrors uint64
	Evictions  uint64
	Promotions uint64
}

// CacheConfig configures cache behavior
type CacheConfig struct {
	L1Size      int
	L2Size      int
	L2Shards    int
	TTL         time.Duration
	EnableStats bool
}

// NewCache creates a new multi-level cache
func NewCache[K comparable, V any](cfg CacheConfig, loader LoaderFunc[K, V]) *Cache[K, V] {
	if cfg.L1Size <= 0 {
		cfg.L1Size = 1000
	}
	if cfg.L2Size <= 0 {
		cfg.L2Size = 10000
	}
	if cfg.L2Shards <= 0 {
		cfg.L2Shards = 16
	}
	if cfg.TTL == 0 {
		cfg.TTL = 10 * time.Minute
	}

	return &Cache[K, V]{
		l1:     newLRUCache[K, V](cfg.L1Size),
		l2:     newShardedCache[K, V](cfg.L2Size, cfg.L2Shards),
		loader: loader,
		config: cfg,
	}
}

// WithL3Backend adds an L3 backend to the cache
func (c *Cache[K, V]) WithL3Backend(backend CacheBackend[K, V]) *Cache[K, V] {
	c.l3 = backend
	return c
}

// Get retrieves a value from the cache hierarchy
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	// Check L1 cache (fastest)
	if value, ok := c.l1.Get(key); ok {
		c.stats.L1Hits.Add(1)
		return value, nil
	}
	c.stats.L1Misses.Add(1)

	// Check L2 cache
	if value, ok := c.l2.Get(key); ok {
		c.stats.L2Hits.Add(1)
		// Promote to L1
		c.l1.Put(key, value, c.config.TTL)
		c.stats.Promotions.Add(1)
		return value, nil
	}
	c.stats.L2Misses.Add(1)

	// Check L3 cache if available
	if c.l3 != nil {
		if value, err := c.l3.Get(ctx, key); err == nil {
			c.stats.L3Hits.Add(1)
			// Promote to L1 and L2
			c.promote(key, value)
			return value, nil
		}
		c.stats.L3Misses.Add(1)
	}

	// Load from source
	if c.loader == nil {
		return zero, gerror.New(gerror.ErrCodeInternal, "no loader function provided", nil)
	}

	c.stats.Loads.Add(1)
	value, err := c.loader(ctx, key)
	if err != nil {
		c.stats.LoadErrors.Add(1)
		return zero, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load value")
	}

	// Store in all cache levels
	c.putAll(key, value)
	return value, nil
}

// Put stores a value in all cache levels
func (c *Cache[K, V]) Put(ctx context.Context, key K, value V) {
	c.putAll(key, value)

	// Also store in L3 if available
	if c.l3 != nil {
		c.l3.Put(ctx, key, value, c.config.TTL)
	}
}

// Delete removes a value from all cache levels
func (c *Cache[K, V]) Delete(ctx context.Context, key K) {
	c.l1.Delete(key)
	c.l2.Delete(key)

	if c.l3 != nil {
		c.l3.Delete(ctx, key)
	}
}

// promote moves a value up the cache hierarchy
func (c *Cache[K, V]) promote(key K, value V) {
	c.l1.Put(key, value, c.config.TTL)
	c.l2.Put(key, value, c.config.TTL)
	c.stats.Promotions.Add(1)
}

// putAll stores a value in L1 and L2 caches
func (c *Cache[K, V]) putAll(key K, value V) {
	c.l1.Put(key, value, c.config.TTL)
	c.l2.Put(key, value, c.config.TTL)
}

// Stats returns current cache statistics
func (c *Cache[K, V]) Stats() CacheStatsSnapshot {
	return CacheStatsSnapshot{
		L1Hits:     c.stats.L1Hits.Load(),
		L1Misses:   c.stats.L1Misses.Load(),
		L2Hits:     c.stats.L2Hits.Load(),
		L2Misses:   c.stats.L2Misses.Load(),
		L3Hits:     c.stats.L3Hits.Load(),
		L3Misses:   c.stats.L3Misses.Load(),
		Loads:      c.stats.Loads.Load(),
		LoadErrors: c.stats.LoadErrors.Load(),
		Promotions: c.stats.Promotions.Load(),
		Evictions:  c.stats.Evictions.Load(),
	}
}

// HitRate returns the overall cache hit rate
func (c *Cache[K, V]) HitRate() float64 {
	hits := c.stats.L1Hits.Load() + c.stats.L2Hits.Load() + c.stats.L3Hits.Load()
	misses := c.stats.L1Misses.Load() + c.stats.L2Misses.Load() + c.stats.L3Misses.Load()
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

// lruCache implements a simple LRU cache
type lruCache[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	items    map[K]*list.Element
	lru      *list.List
}

// lruItem represents an item in the LRU cache
type lruItem[K comparable, V any] struct {
	key     K
	value   V
	expires time.Time
}

// newLRUCache creates a new LRU cache
func newLRUCache[K comparable, V any](capacity int) *lruCache[K, V] {
	return &lruCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a value from the LRU cache
func (c *lruCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	elem, ok := c.items[key]
	if !ok {
		return zero, false
	}

	item := elem.Value.(*lruItem[K, V])

	// Check expiration
	if time.Now().After(item.expires) {
		c.removeElement(elem)
		return zero, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)
	return item.value, true
}

// Put stores a value in the LRU cache
func (c *lruCache[K, V]) Put(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expires := time.Now().Add(ttl)

	// Update existing item
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*lruItem[K, V])
		item.value = value
		item.expires = expires
		c.lru.MoveToFront(elem)
		return
	}

	// Add new item
	item := &lruItem[K, V]{
		key:     key,
		value:   value,
		expires: expires,
	}

	elem := c.lru.PushFront(item)
	c.items[key] = elem

	// Evict if over capacity
	if c.lru.Len() > c.capacity {
		c.evictOldest()
	}
}

// Delete removes a value from the LRU cache
func (c *lruCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache
func (c *lruCache[K, V]) removeElement(elem *list.Element) {
	item := elem.Value.(*lruItem[K, V])
	delete(c.items, item.key)
	c.lru.Remove(elem)
}

// evictOldest removes the least recently used item
func (c *lruCache[K, V]) evictOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// shardedCache implements a sharded cache for reduced contention
type shardedCache[K comparable, V any] struct {
	shards    []*lruCache[K, V]
	shardMask uint32
}

// newShardedCache creates a new sharded cache
func newShardedCache[K comparable, V any](totalSize int, shards int) *shardedCache[K, V] {
	// Ensure shards is a power of 2 for efficient masking
	if shards&(shards-1) != 0 {
		panic("shards must be a power of 2")
	}

	shardSize := totalSize / shards
	if shardSize < 1 {
		shardSize = 1
	}

	sc := &shardedCache[K, V]{
		shards:    make([]*lruCache[K, V], shards),
		shardMask: uint32(shards - 1),
	}

	for i := 0; i < shards; i++ {
		sc.shards[i] = newLRUCache[K, V](shardSize)
	}

	return sc
}

// getShard returns the shard for a given key
func (sc *shardedCache[K, V]) getShard(key K) *lruCache[K, V] {
	hash := hashKey(key)
	return sc.shards[hash&sc.shardMask]
}

// Get retrieves a value from the sharded cache
func (sc *shardedCache[K, V]) Get(key K) (V, bool) {
	return sc.getShard(key).Get(key)
}

// Put stores a value in the sharded cache
func (sc *shardedCache[K, V]) Put(key K, value V, ttl time.Duration) {
	sc.getShard(key).Put(key, value, ttl)
}

// Delete removes a value from the sharded cache
func (sc *shardedCache[K, V]) Delete(key K) {
	sc.getShard(key).Delete(key)
}

// hashKey computes a hash for a key
func hashKey[K comparable](key K) uint32 {
	h := fnv.New32a()

	// For simple types, convert to string and hash
	// This is safer than using unsafe
	switch v := any(key).(type) {
	case string:
		h.Write([]byte(v))
	case int:
		h.Write([]byte(fmt.Sprintf("%d", v)))
	case int64:
		h.Write([]byte(fmt.Sprintf("%d", v)))
	case uint64:
		h.Write([]byte(fmt.Sprintf("%d", v)))
	default:
		// For other types, use reflection to get string representation
		h.Write([]byte(fmt.Sprintf("%v", v)))
	}

	return h.Sum32()
}

// MemoryBackend implements a simple in-memory L3 backend
type MemoryBackend[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]memoryItem[V]
}

// memoryItem represents an item in the memory backend
type memoryItem[V any] struct {
	value   V
	expires time.Time
}

// NewMemoryBackend creates a new memory backend
func NewMemoryBackend[K comparable, V any]() *MemoryBackend[K, V] {
	return &MemoryBackend[K, V]{
		items: make(map[K]memoryItem[V]),
	}
}

// Get retrieves a value from the memory backend
func (mb *MemoryBackend[K, V]) Get(ctx context.Context, key K) (V, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var zero V
	item, ok := mb.items[key]
	if !ok {
		return zero, gerror.New(gerror.ErrCodeInternal, "key not found", nil)
	}

	if time.Now().After(item.expires) {
		return zero, gerror.New(gerror.ErrCodeInternal, "key expired", nil)
	}

	return item.value, nil
}

// Put stores a value in the memory backend
func (mb *MemoryBackend[K, V]) Put(ctx context.Context, key K, value V, ttl time.Duration) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.items[key] = memoryItem[V]{
		value:   value,
		expires: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from the memory backend
func (mb *MemoryBackend[K, V]) Delete(ctx context.Context, key K) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	delete(mb.items, key)
	return nil
}

// Clear removes all values from the memory backend
func (mb *MemoryBackend[K, V]) Clear(ctx context.Context) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.items = make(map[K]memoryItem[V])
	return nil
}

// AdaptiveCache implements an adaptive cache that adjusts sizes based on hit rates
type AdaptiveCache[K comparable, V any] struct {
	cache       *Cache[K, V]
	hitRates    [3]float64 // L1, L2, L3 hit rates
	adjustments atomic.Int32
	lastAdjust  atomic.Int64
}

// NewAdaptiveCache creates a new adaptive cache
func NewAdaptiveCache[K comparable, V any](cfg CacheConfig, loader LoaderFunc[K, V]) *AdaptiveCache[K, V] {
	return &AdaptiveCache[K, V]{
		cache: NewCache(cfg, loader),
	}
}

// Get retrieves a value and tracks performance
func (ac *AdaptiveCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	value, err := ac.cache.Get(ctx, key)
	ac.maybeAdjust()
	return value, err
}

// maybeAdjust periodically adjusts cache sizes based on performance
func (ac *AdaptiveCache[K, V]) maybeAdjust() {
	now := time.Now().Unix()
	if now-ac.lastAdjust.Load() > 60 { // Adjust every minute
		if ac.lastAdjust.CompareAndSwap(ac.lastAdjust.Load(), now) {
			go ac.adjust()
		}
	}
}

// adjust modifies cache sizes based on hit rates
func (ac *AdaptiveCache[K, V]) adjust() {
	stats := ac.cache.Stats()

	l1Total := stats.L1Hits + stats.L1Misses
	l2Total := stats.L2Hits + stats.L2Misses
	l3Total := stats.L3Hits + stats.L3Misses

	if l1Total > 0 {
		ac.hitRates[0] = float64(stats.L1Hits) / float64(l1Total)
	}
	if l2Total > 0 {
		ac.hitRates[1] = float64(stats.L2Hits) / float64(l2Total)
	}
	if l3Total > 0 {
		ac.hitRates[2] = float64(stats.L3Hits) / float64(l3Total)
	}

	ac.adjustments.Add(1)
	// Implementation of actual size adjustment would go here
	// This would involve creating new cache instances with adjusted sizes
}
