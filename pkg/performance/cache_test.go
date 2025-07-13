package performance

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 10,
		L2Size: 20,
	})

	// Test Set and Get
	cache.Set("key1", 100)
	value, found := cache.Get("key1")
	if !found || value != 100 {
		t.Errorf("Expected (100, true), got (%d, %t)", value, found)
	}

	// Test missing key
	_, found = cache.Get("missing")
	if found {
		t.Error("Expected false for missing key")
	}

	// Test Delete
	cache.Delete("key1")
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected false after delete")
	}
}

func TestCacheWithLoader(t *testing.T) {
	loadCount := 0
	loader := func(key string) (int, error) {
		loadCount++
		val, _ := strconv.Atoi(key)
		return val * 2, nil
	}

	cache := NewTestCacheWithLoader[string, int](TestCacheConfig{
		L1Size: 5,
		L2Size: 10,
	}, loader)

	// First access should load
	value, found := cache.Get("42")
	if !found || value != 84 {
		t.Errorf("Expected (84, true), got (%d, %t)", value, found)
	}
	if loadCount != 1 {
		t.Errorf("Expected 1 load, got %d", loadCount)
	}

	// Second access should hit cache
	value, found = cache.Get("42")
	if !found || value != 84 {
		t.Errorf("Expected (84, true), got (%d, %t)", value, found)
	}
	if loadCount != 1 {
		t.Errorf("Expected 1 load (cached), got %d", loadCount)
	}
}

func TestCacheEviction(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 2,
		L2Size: 3,
	})

	// Fill beyond L1 capacity
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3) // Should evict from L1 to L2

	// Access key1 to check if it's still available (in L2)
	value, found := cache.Get("key1")
	if !found || value != 1 {
		t.Errorf("Expected key1 to be in L2, got (%d, %t)", value, found)
	}

	// Add more items to trigger L2 eviction
	cache.Set("key4", 4)
	cache.Set("key5", 5)
	cache.Set("key6", 6) // Should evict oldest from L2

	// Some items should be evicted
	stats := cache.Stats()
	if stats.L1Size+stats.L2Size > 5 {
		t.Errorf("Cache size exceeded limits: L1=%d, L2=%d", stats.L1Size, stats.L2Size)
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 100,
		L2Size: 200,
	})

	const numGoroutines = 50
	const itemsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				key := strconv.Itoa(goroutineID*itemsPerGoroutine + j)
				value := goroutineID*itemsPerGoroutine + j

				cache.Set(key, value)

				retrieved, found := cache.Get(key)
				if found && retrieved != value {
					t.Errorf("Concurrent access failure: expected %d, got %d", value, retrieved)
				}
			}
		}(i)
	}

	wg.Wait()

	stats := cache.Stats()
	if stats.Sets == 0 || stats.Gets == 0 {
		t.Errorf("Expected non-zero stats, got %+v", stats)
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 10,
		L2Size: 20,
		TTL:    100 * time.Millisecond,
	})

	cache.Set("key1", 100)

	// Should be available immediately
	value, found := cache.Get("key1")
	if !found || value != 100 {
		t.Errorf("Expected (100, true), got (%d, %t)", value, found)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected item to be expired", nil)
	}
}

func TestShardedCache(t *testing.T) {
	cache := NewTestShardedCache[string, int](16, func() TestCache[string, int] {
		return NewTestCache[string, int](TestCacheConfig{
			L1Size: 10,
			L2Size: 20,
		})
	})

	// Test basic operations
	cache.Set("key1", 100)
	value, found := cache.Get("key1")
	if !found || value != 100 {
		t.Errorf("Expected (100, true), got (%d, %t)", value, found)
	}

	// Test concurrent access
	const numGoroutines = 10
	const itemsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				key := strconv.Itoa(goroutineID*itemsPerGoroutine + j)
				value := goroutineID*itemsPerGoroutine + j

				cache.Set(key, value)

				retrieved, found := cache.Get(key)
				if found && retrieved != value {
					t.Errorf("Sharded cache failure: expected %d, got %d", value, retrieved)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheStatsTracking(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 5,
		L2Size: 10,
	})

	// Generate some cache activity
	for i := 0; i < 20; i++ {
		key := strconv.Itoa(i)
		cache.Set(key, i)
	}

	for i := 0; i < 10; i++ {
		key := strconv.Itoa(i)
		cache.Get(key)
	}

	// Access non-existent keys
	for i := 100; i < 105; i++ {
		key := strconv.Itoa(i)
		cache.Get(key)
	}

	stats := cache.Stats()

	if stats.Sets != 20 {
		t.Errorf("Expected 20 sets, got %d", stats.Sets)
	}

	if stats.Gets != 15 {
		t.Errorf("Expected 15 gets, got %d", stats.Gets)
	}

	if stats.Misses == 0 {
		t.Error("Expected some misses", nil)
	}

	hitRate := float64(stats.Hits) / float64(stats.Gets)
	if hitRate <= 0 {
		t.Errorf("Expected positive hit rate, got %f", hitRate)
	}
}

func TestCacheWithContext(t *testing.T) {
	loader := func(key string) (int, error) {
		time.Sleep(50 * time.Millisecond) // Simulate slow load
		val, _ := strconv.Atoi(key)
		return val, nil
	}

	cache := NewTestCacheWithLoader[string, int](TestCacheConfig{
		L1Size: 5,
		L2Size: 10,
	}, loader)

	// Should succeed within timeout
	value, found := cache.Get("42")
	if !found || value != 42 {
		t.Errorf("Expected (42, true), got (%d, %t)", value, found)
	}

	// Test with cancelled context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	cancel() // Cancel immediately
	_ = ctx  // ctx is not used in test cache

	// This would normally work since the item is cached
	value, found = cache.Get("42")
	if !found || value != 42 {
		t.Errorf("Expected cached value (42, true), got (%d, %t)", value, found)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 1000,
		L2Size: 2000,
	})

	// Populate cache
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := strconv.Itoa(i % 500)
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 1000,
		L2Size: 2000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := strconv.Itoa(i)
			cache.Set(key, i)
			i++
		}
	})
}

func BenchmarkShardedCacheGet(b *testing.B) {
	cache := NewTestShardedCache[string, int](16, func() TestCache[string, int] {
		return NewTestCache[string, int](TestCacheConfig{
			L1Size: 100,
			L2Size: 200,
		})
	})

	// Populate cache
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := strconv.Itoa(i % 500)
			cache.Get(key)
			i++
		}
	})
}

func TestCacheCleanup(t *testing.T) {
	cache := NewTestCache[string, int](TestCacheConfig{
		L1Size: 5,
		L2Size: 10,
		TTL:    50 * time.Millisecond,
	})

	// Add items
	for i := 0; i < 10; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	initialStats := cache.Stats()
	if initialStats.L1Size+initialStats.L2Size == 0 {
		t.Error("Cache should have items", nil)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Trigger cleanup by accessing cache
	cache.Get("0")

	// Check if expired items were cleaned up
	finalStats := cache.Stats()
	if finalStats.L1Size+finalStats.L2Size >= initialStats.L1Size+initialStats.L2Size {
		t.Error("Expected cleanup to reduce cache size", nil)
	}
}

func TestCacheMemoryPressure(t *testing.T) {
	cache := NewTestCache[string, []byte](TestCacheConfig{
		L1Size: 10,
		L2Size: 20,
	})

	// Add large items to trigger eviction
	for i := 0; i < 50; i++ {
		key := strconv.Itoa(i)
		value := make([]byte, 1024) // 1KB per item
		cache.Set(key, value)
	}

	stats := cache.Stats()

	// Cache should not exceed its size limits
	if stats.L1Size > 10 {
		t.Errorf("L1 size exceeded limit: %d", stats.L1Size)
	}
	if stats.L2Size > 20 {
		t.Errorf("L2 size exceeded limit: %d", stats.L2Size)
	}

	// Should have evictions
	if stats.Evictions == 0 {
		t.Error("Expected evictions due to memory pressure", nil)
	}
}
