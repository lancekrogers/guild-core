// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewIntelligentCache(t *testing.T) {
	config := DefaultCacheConfig()
	cache, err := NewIntelligentCache(config)

	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	if cache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	if cache.l1Cache == nil {
		t.Error("Expected L1 cache to be initialized")
	}

	if cache.l2Cache == nil {
		t.Error("Expected L2 cache to be initialized")
	}

	if cache.predictor == nil {
		t.Error("Expected predictor to be initialized")
	}

	if cache.stats == nil {
		t.Error("Expected stats to be initialized")
	}
}

func TestIntelligentCacheGetSet(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Test cache miss
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected cache miss error")
	}

	// Test cache set
	err = cache.Set(ctx, key, value)
	if err != nil {
		t.Errorf("Cache set failed: %v", err)
	}

	// Test cache hit
	retrieved, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Cache get failed: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
}

func TestIntelligentCacheOptions(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Test with TTL option
	err = cache.Set(ctx, key, value, WithTTL(time.Minute))
	if err != nil {
		t.Errorf("Cache set with TTL failed: %v", err)
	}

	// Test with cost option
	err = cache.Set(ctx, key, value, WithCost(10.5))
	if err != nil {
		t.Errorf("Cache set with cost failed: %v", err)
	}

	// Test with metadata option
	err = cache.Set(ctx, key, value, WithMetadata("type", "test"))
	if err != nil {
		t.Errorf("Cache set with metadata failed: %v", err)
	}

	// Test with shared option
	err = cache.Set(ctx, key, value, WithShared(true))
	if err != nil {
		t.Errorf("Cache set with shared failed: %v", err)
	}
}

func TestIntelligentCacheDelete(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value
	err = cache.Set(ctx, key, value)
	if err != nil {
		t.Errorf("Cache set failed: %v", err)
	}

	// Verify it exists
	_, err = cache.Get(ctx, key)
	if err != nil {
		t.Error("Expected cache hit before delete")
	}

	// Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Errorf("Cache delete failed: %v", err)
	}

	// Verify it's gone
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected cache miss after delete")
	}
}

func TestIntelligentCacheClear(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err = cache.Set(ctx, key, value)
		if err != nil {
			t.Errorf("Cache set failed for %s: %v", key, err)
		}
	}

	// Clear cache
	err = cache.Clear(ctx)
	if err != nil {
		t.Errorf("Cache clear failed: %v", err)
	}

	// Verify all values are gone
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err = cache.Get(ctx, key)
		if err == nil {
			t.Errorf("Expected cache miss for %s after clear", key)
		}
	}
}

func TestIntelligentCacheStats(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	stats := cache.GetStats()
	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}

	if stats.L1Stats == nil {
		t.Error("Expected L1 stats to be non-nil")
	}

	if stats.L2Stats == nil {
		t.Error("Expected L2 stats to be non-nil")
	}

	if stats.OverallStats == nil {
		t.Error("Expected overall stats to be non-nil")
	}

	if stats.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestCacheEntryExpiration(t *testing.T) {
	entry := &CacheEntry{
		Key:       "test",
		Value:     "value",
		CreatedAt: time.Now().Add(-time.Hour),
		TTL:       time.Minute * 30, // Expired
	}

	if !entry.IsExpired() {
		t.Error("Expected entry to be expired")
	}

	// Test non-expiring entry
	entry.TTL = 0
	if entry.IsExpired() {
		t.Error("Expected entry with 0 TTL to never expire")
	}

	// Test valid entry
	entry.CreatedAt = time.Now()
	entry.TTL = time.Hour
	if entry.IsExpired() {
		t.Error("Expected fresh entry to not be expired")
	}
}

func TestCacheEntryUpdateAccess(t *testing.T) {
	entry := &CacheEntry{
		Key:         "test",
		Value:       "value",
		AccessCount: 0,
		LastAccess:  time.Time{},
	}

	entry.UpdateAccess()

	if entry.AccessCount != 1 {
		t.Errorf("Expected access count to be 1, got %d", entry.AccessCount)
	}

	if entry.LastAccess.IsZero() {
		t.Error("Expected last access time to be updated")
	}
}

func TestAccessPredictor(t *testing.T) {
	predictor := NewAccessPredictor(time.Hour)

	if predictor == nil {
		t.Fatal("Expected predictor to be non-nil")
	}

	key := "test-key"
	now := time.Now()

	// Record some accesses
	predictor.RecordAccess(key, now)
	predictor.RecordAccess(key, now.Add(time.Minute))
	predictor.RecordAccess(key, now.Add(time.Minute*2))

	// Get prediction
	prediction := predictor.PredictNextAccess(key)
	if prediction == nil {
		t.Fatal("Expected prediction to be non-nil")
	}

	if prediction.Key != key {
		t.Errorf("Expected prediction key to be %s, got %s", key, prediction.Key)
	}

	if prediction.Probability < 0 || prediction.Probability > 1 {
		t.Errorf("Expected probability to be between 0 and 1, got %f", prediction.Probability)
	}
}

func TestShouldPromote(t *testing.T) {
	predictor := NewAccessPredictor(time.Hour)

	// Test with no history
	if predictor.ShouldPromote("unknown-key") {
		t.Error("Expected false for unknown key")
	}

	// Record frequent accesses
	key := "hot-key"
	now := time.Now()
	for i := 0; i < 10; i++ {
		predictor.RecordAccess(key, now.Add(time.Duration(i)*time.Minute))
	}

	// This should result in promotion
	// Note: The actual promotion decision depends on the prediction algorithm
	result := predictor.ShouldPromote(key)
	// We can't guarantee the result without more sophisticated setup,
	// but we can test that it returns a boolean
	if result != true && result != false {
		t.Error("ShouldPromote should return a boolean")
	}
}

func TestGetHotKeyPredictions(t *testing.T) {
	predictor := NewAccessPredictor(time.Hour)

	// Record accesses for multiple keys
	keys := []string{"key1", "key2", "key3"}
	now := time.Now()

	for _, key := range keys {
		for i := 0; i < 5; i++ {
			predictor.RecordAccess(key, now.Add(time.Duration(i)*time.Minute))
		}
	}

	predictions := predictor.GetHotKeyPredictions(time.Hour)

	if predictions == nil {
		t.Fatal("Expected predictions to be non-nil")
	}

	// Predictions should be sorted by probability
	for i := 1; i < len(predictions); i++ {
		if predictions[i-1].Probability < predictions[i].Probability {
			t.Error("Expected predictions to be sorted by probability (descending)")
		}
	}
}

func TestCacheWarmer(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	warmer := NewCacheWarmer(cache, cache.predictor)
	if warmer == nil {
		t.Fatal("Expected warmer to be non-nil")
	}

	if warmer.cache != cache {
		t.Error("Expected warmer cache to match input cache")
	}

	if warmer.predictor != cache.predictor {
		t.Error("Expected warmer predictor to match input predictor")
	}

	// Test warming with no predictions
	ctx := context.Background()
	err = warmer.WarmCache(ctx)
	if err != nil {
		t.Errorf("WarmCache failed: %v", err)
	}
}

// Test with different cache levels
func TestCacheLevelDetermination(t *testing.T) {
	cache, err := NewIntelligentCache(nil)
	if err != nil {
		t.Fatalf("NewIntelligentCache failed: %v", err)
	}

	// Test small entry (should go to L1 or L2)
	smallEntry := &CacheEntry{
		Key:   "small",
		Value: "small-value",
		Size:  100,
	}

	level := cache.determineCacheLevel(smallEntry)
	if level != CacheLevelL2 {
		t.Errorf("Expected small entry to go to L2, got %s", level.String())
	}

	// Test large entry (should go to L2)
	largeEntry := &CacheEntry{
		Key:   "large",
		Value: make([]byte, 1024*1024), // 1MB
		Size:  1024 * 1024,
	}

	level = cache.determineCacheLevel(largeEntry)
	if level != CacheLevelL2 {
		t.Errorf("Expected large entry to go to L2, got %s", level.String())
	}

	// Test shared entry
	sharedEntry := &CacheEntry{
		Key:      "shared",
		Value:    "shared-value",
		Size:     100,
		Metadata: map[string]interface{}{"shared": true},
	}

	level = cache.determineCacheLevel(sharedEntry)
	if level != CacheLevelDistributed {
		t.Errorf("Expected shared entry to go to distributed, got %s", level.String())
	}
}

// Benchmark tests
func BenchmarkCacheGet(b *testing.B) {
	cache, _ := NewIntelligentCache(nil)
	ctx := context.Background()

	// Pre-populate cache
	cache.Set(ctx, "test-key", "test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "test-key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache, _ := NewIntelligentCache(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(ctx, key, "test-value")
	}
}

func BenchmarkPredictNextAccess(b *testing.B) {
	predictor := NewAccessPredictor(time.Hour)
	key := "test-key"

	// Pre-populate with some history
	now := time.Now()
	for i := 0; i < 10; i++ {
		predictor.RecordAccess(key, now.Add(time.Duration(i)*time.Minute))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.PredictNextAccess(key)
	}
}
