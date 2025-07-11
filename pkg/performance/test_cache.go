// test_cache.go provides simple cache implementations for testing and benchmarking
package performance

import (
	"sync"
	"sync/atomic"
	"time"
)

// TestCacheConfig is the configuration for test caches
type TestCacheConfig struct {
	L1Size   int
	L2Size   int
	L2Shards int
	TTL      time.Duration
}

// TestCache is a simple cache interface for tests
type TestCache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	Delete(key K)
	Stats() TestCacheStats
}

// TestCacheStats contains statistics for the test cache
type TestCacheStats struct {
	Sets      uint64
	Gets      uint64
	Hits      uint64
	Misses    uint64
	Deletes   uint64
	Evictions uint64
	L1Size    int
	L2Size    int
}

// testCacheImpl is a simple cache implementation for tests
type testCacheImpl[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]cacheItem[V]
	loader   func(K) (V, error)
	l1Size   int
	l2Size   int
	maxSize  int
	ttl      time.Duration
	stats    testCacheStats
}

type cacheItem[V any] struct {
	value   V
	expires time.Time
}

type testCacheStats struct {
	sets      atomic.Uint64
	gets      atomic.Uint64
	hits      atomic.Uint64
	misses    atomic.Uint64
	deletes   atomic.Uint64
	evictions atomic.Uint64
}

// NewTestCache creates a new test cache
func NewTestCache[K comparable, V any](cfg TestCacheConfig) TestCache[K, V] {
	return &testCacheImpl[K, V]{
		items:   make(map[K]cacheItem[V]),
		l1Size:  cfg.L1Size,
		l2Size:  cfg.L2Size,
		maxSize: cfg.L1Size + cfg.L2Size,
		ttl:     cfg.TTL,
	}
}

// NewTestCacheWithLoader creates a test cache with a loader function
func NewTestCacheWithLoader[K comparable, V any](cfg TestCacheConfig, loader func(K) (V, error)) TestCache[K, V] {
	return &testCacheImpl[K, V]{
		items:   make(map[K]cacheItem[V]),
		loader:  loader,
		l1Size:  cfg.L1Size,
		l2Size:  cfg.L2Size,
		maxSize: cfg.L1Size + cfg.L2Size,
		ttl:     cfg.TTL,
	}
}

// Get retrieves a value from the cache
func (c *testCacheImpl[K, V]) Get(key K) (V, bool) {
	c.stats.gets.Add(1)
	
	// Cleanup expired items periodically (simple implementation)
	c.cleanupExpired()
	
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()
	
	var zero V
	
	if found && (item.expires.IsZero() || time.Now().Before(item.expires)) {
		c.stats.hits.Add(1)
		return item.value, true
	}
	
	// Try loader if available
	if c.loader != nil && !found {
		if value, err := c.loader(key); err == nil {
			c.Set(key, value)
			c.stats.hits.Add(1)
			return value, true
		}
	}
	
	c.stats.misses.Add(1)
	return zero, false
}

// Set stores a value in the cache
func (c *testCacheImpl[K, V]) Set(key K, value V) {
	c.stats.sets.Add(1)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple eviction if over capacity
	if len(c.items) >= c.maxSize {
		// Remove first item found
		for k := range c.items {
			delete(c.items, k)
			c.stats.evictions.Add(1)
			break
		}
	}
	
	var expires time.Time
	if c.ttl > 0 {
		expires = time.Now().Add(c.ttl)
	}
	
	c.items[key] = cacheItem[V]{
		value:   value,
		expires: expires,
	}
}

// Delete removes a value from the cache
func (c *testCacheImpl[K, V]) Delete(key K) {
	c.stats.deletes.Add(1)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// cleanupExpired removes expired items from the cache
func (c *testCacheImpl[K, V]) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for k, item := range c.items {
		if !item.expires.IsZero() && now.After(item.expires) {
			delete(c.items, k)
		}
	}
}

// Stats returns cache statistics
func (c *testCacheImpl[K, V]) Stats() TestCacheStats {
	c.mu.RLock()
	size := len(c.items)
	c.mu.RUnlock()
	
	// Calculate L1 and L2 sizes based on actual items and configured limits
	l1Size := size
	l2Size := 0
	
	if size > c.l1Size {
		l1Size = c.l1Size
		l2Size = size - c.l1Size
		if l2Size > c.l2Size {
			l2Size = c.l2Size
		}
	}
	
	return TestCacheStats{
		Sets:      c.stats.sets.Load(),
		Gets:      c.stats.gets.Load(),
		Hits:      c.stats.hits.Load(),
		Misses:    c.stats.misses.Load(),
		Deletes:   c.stats.deletes.Load(),
		Evictions: c.stats.evictions.Load(),
		L1Size:    l1Size,
		L2Size:    l2Size,
	}
}

// testShardedCache is a sharded cache for tests
type testShardedCache[K comparable, V any] struct {
	shards    []TestCache[K, V]
	numShards int
}

// NewTestShardedCache creates a new sharded test cache
func NewTestShardedCache[K comparable, V any](numShards int, factory func() TestCache[K, V]) *testShardedCache[K, V] {
	shards := make([]TestCache[K, V], numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = factory()
	}
	
	return &testShardedCache[K, V]{
		shards:    shards,
		numShards: numShards,
	}
}

// Get retrieves a value from the appropriate shard
func (sc *testShardedCache[K, V]) Get(key K) (V, bool) {
	shard := sc.getShard(key)
	return sc.shards[shard].Get(key)
}

// Set stores a value in the appropriate shard
func (sc *testShardedCache[K, V]) Set(key K, value V) {
	shard := sc.getShard(key)
	sc.shards[shard].Set(key, value)
}

// Delete removes a value from the appropriate shard
func (sc *testShardedCache[K, V]) Delete(key K) {
	shard := sc.getShard(key)
	sc.shards[shard].Delete(key)
}

// getShard determines which shard to use for a key
func (sc *testShardedCache[K, V]) getShard(key K) int {
	// Simple hash based on string representation
	h := uint32(0)
	keyStr := ""
	switch v := any(key).(type) {
	case string:
		keyStr = v
	case int:
		keyStr = string(rune(v))
	default:
		keyStr = "default"
	}
	
	for _, c := range keyStr {
		h = h*31 + uint32(c)
	}
	
	return int(h % uint32(sc.numShards))
}