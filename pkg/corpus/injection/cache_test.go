// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package injection

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftContextCache_BasicOperations(t *testing.T) {
	cache := NewContextCache(time.Hour)

	// Test initial state
	assert.Equal(t, 0, cache.Size())
	assert.Nil(t, cache.Get("nonexistent"))

	// Test set and get
	value := &InjectedPrompt{
		SystemPrompt: "Test prompt",
		Contexts: map[InjectionPoint]string{
			InjectionSystemPrompt: "test context",
		},
	}

	cache.Set("key1", value)
	assert.Equal(t, 1, cache.Size())

	retrieved := cache.Get("key1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test prompt", retrieved.SystemPrompt)
	assert.Equal(t, "test context", retrieved.Contexts[InjectionSystemPrompt])
}

func TestGuildContextCache_TTL(t *testing.T) {
	// Use very short TTL for testing
	cache := NewContextCache(10 * time.Millisecond)

	value := &InjectedPrompt{
		SystemPrompt: "Test prompt",
	}

	cache.Set("key1", value)
	assert.Equal(t, 1, cache.Size())

	// Should be available immediately
	retrieved := cache.Get("key1")
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Should be expired and removed
	retrieved = cache.Get("key1")
	assert.Nil(t, retrieved)
	assert.Equal(t, 0, cache.Size()) // Should be cleaned up on access
}

func TestJourneymanContextCache_AccessCounting(t *testing.T) {
	cache := NewContextCache(time.Hour)

	value := &InjectedPrompt{
		SystemPrompt: "Test prompt",
	}

	cache.Set("key1", value)

	// Access multiple times
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")

	// Check internal state (we need to access the entry directly for testing)
	cache.mu.RLock()
	entry := cache.entries["key1"]
	cache.mu.RUnlock()

	require.NotNil(t, entry)
	assert.Equal(t, 4, entry.AccessCount) // 1 from Set + 3 from Get calls
}

func TestCraftContextCache_LRUEviction(t *testing.T) {
	cache := NewContextCacheWithSize(time.Hour, 2) // Max size of 2

	value1 := &InjectedPrompt{SystemPrompt: "Prompt 1"}
	value2 := &InjectedPrompt{SystemPrompt: "Prompt 2"}
	value3 := &InjectedPrompt{SystemPrompt: "Prompt 3"}

	// Fill cache to capacity
	cache.Set("key1", value1)
	cache.Set("key2", value2)
	assert.Equal(t, 2, cache.Size())

	// Access key1 to make it more recently used
	cache.Get("key1")

	// Add third item, should evict key2 (least recently used)
	cache.Set("key3", value3)
	assert.Equal(t, 2, cache.Size())

	// key1 and key3 should exist, key2 should be evicted
	assert.NotNil(t, cache.Get("key1"))
	assert.Nil(t, cache.Get("key2"))
	assert.NotNil(t, cache.Get("key3"))
}

func TestGuildContextCache_Delete(t *testing.T) {
	cache := NewContextCache(time.Hour)

	value := &InjectedPrompt{
		SystemPrompt: "Test prompt",
	}

	cache.Set("key1", value)
	assert.Equal(t, 1, cache.Size())

	cache.Delete("key1")
	assert.Equal(t, 0, cache.Size())
	assert.Nil(t, cache.Get("key1"))

	// Delete non-existent key should not error
	cache.Delete("nonexistent")
}

func TestJourneymanContextCache_Clear(t *testing.T) {
	cache := NewContextCache(time.Hour)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		value := &InjectedPrompt{SystemPrompt: "Prompt " + string(rune(i))}
		cache.Set(key, value)
	}

	assert.Equal(t, 5, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())

	// All keys should be gone
	for i := 0; i < 5; i++ {
		key := "key" + string(rune(i))
		assert.Nil(t, cache.Get(key))
	}
}

func TestCraftContextCache_CleanupExpired(t *testing.T) {
	cache := NewContextCache(20 * time.Millisecond)

	// Add entries
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}
	cache.Set("key1", value)
	cache.Set("key2", value)
	cache.Set("key3", value)
	assert.Equal(t, 3, cache.Size())

	// Wait for expiration
	time.Sleep(25 * time.Millisecond)

	// Manually cleanup expired entries
	expiredCount := cache.CleanupExpired()
	assert.Equal(t, 3, expiredCount)
	assert.Equal(t, 0, cache.Size())
}

func TestGuildContextCache_Stats(t *testing.T) {
	cache := NewContextCacheWithSize(time.Hour, 100)

	// Add some entries
	for i := 0; i < 5; i++ {
		key := "key" + strconv.Itoa(i)
		value := &InjectedPrompt{SystemPrompt: "Prompt " + strconv.Itoa(i)}
		cache.Set(key, value)
	}

	// Access some entries multiple times
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")

	stats := cache.Stats()
	assert.Equal(t, 5, stats.Size)
	assert.Equal(t, 100, stats.MaxSize)
	assert.Equal(t, time.Hour, stats.TTL)
	assert.Equal(t, 8, stats.TotalAccesses)  // 5 from Set + 3 from Get
	assert.Equal(t, 0, stats.ExpiredEntries) // None expired yet
}

func TestJourneymanContextCache_StatsWithExpiredEntries(t *testing.T) {
	cache := NewContextCache(10 * time.Millisecond)

	// Add entries
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}
	cache.Set("key1", value)
	cache.Set("key2", value)

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	stats := cache.Stats()
	assert.Equal(t, 2, stats.Size)           // Still counted in size
	assert.Equal(t, 2, stats.ExpiredEntries) // But marked as expired
}

func TestCraftContextCache_ConcurrentAccess(t *testing.T) {
	cache := NewContextCache(time.Hour)
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key" + string(rune(id))
			cache.Set(key, value)
			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, cache.Size())

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key" + string(rune(id))
			retrieved := cache.Get(key)
			assert.NotNil(t, retrieved)
			done <- true
		}(i)
	}

	// Wait for all reads to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGuildContextCache_EmptyCache(t *testing.T) {
	cache := NewContextCache(time.Hour)

	// Test operations on empty cache
	assert.Equal(t, 0, cache.Size())
	assert.Nil(t, cache.Get("nonexistent"))

	stats := cache.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, 0, stats.TotalAccesses)
	assert.Equal(t, 0, stats.ExpiredEntries)

	// Cleanup on empty cache should return 0
	expiredCount := cache.CleanupExpired()
	assert.Equal(t, 0, expiredCount)

	// Clear on empty cache should not error
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestScribeContextCache_InjectionPointString(t *testing.T) {
	tests := []struct {
		point    InjectionPoint
		expected string
	}{
		{InjectionSystemPrompt, "system_prompt"},
		{InjectionUserMessage, "user_message"},
		{InjectionToolContext, "tool_context"},
		{InjectionPoint(999), "unknown"},
	}

	for _, test := range tests {
		result := test.point.String()
		assert.Equal(t, test.expected, result)
	}
}

// Benchmark tests

func BenchmarkCraftCache_Set(b *testing.B) {
	cache := NewContextCache(time.Hour)
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000)) // Cycle through 1000 keys
		cache.Set(key, value)
	}
}

func BenchmarkJourneymanCache_Get(b *testing.B) {
	cache := NewContextCache(time.Hour)
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		cache.Get(key)
	}
}

func BenchmarkGuildCache_SetGetMixed(b *testing.B) {
	cache := NewContextCache(time.Hour)
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%100))
		if i%2 == 0 {
			cache.Set(key, value)
		} else {
			cache.Get(key)
		}
	}
}

func TestCraftContextCache_CleanupRoutine(t *testing.T) {
	cache := NewContextCache(50 * time.Millisecond)

	// Add an entry
	value := &InjectedPrompt{SystemPrompt: "Test prompt"}
	cache.Set("key1", value)
	assert.Equal(t, 1, cache.Size())

	// Start cleanup routine with short interval
	cache.StartCleanupRoutine(30 * time.Millisecond)

	// Wait for entry to expire and be cleaned up
	time.Sleep(100 * time.Millisecond)

	// Entry should be cleaned up by the routine
	assert.Equal(t, 0, cache.Size())
	assert.Nil(t, cache.Get("key1"))
}
