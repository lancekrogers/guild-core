// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package memory

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewMemoryOptimizer(t *testing.T) {
	config := DefaultOptimizerConfig()
	optimizer := NewMemoryOptimizer(config)

	if optimizer == nil {
		t.Fatal("Expected optimizer to be non-nil")
	}

	if optimizer.profiler == nil {
		t.Error("Expected profiler to be initialized")
	}

	if optimizer.analyzer == nil {
		t.Error("Expected analyzer to be initialized")
	}

	if optimizer.pools == nil {
		t.Error("Expected pools map to be initialized")
	}

	if optimizer.compactor == nil {
		t.Error("Expected compactor to be initialized")
	}

	if optimizer.interner == nil {
		t.Error("Expected interner to be initialized")
	}

	if optimizer.bufferPool == nil {
		t.Error("Expected buffer pool to be initialized")
	}

	if optimizer.config != config {
		t.Error("Expected config to match input config")
	}
}

func TestDefaultOptimizerConfig(t *testing.T) {
	config := DefaultOptimizerConfig()

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if !config.EnablePooling {
		t.Error("Expected pooling to be enabled by default")
	}

	if !config.EnableStringInterning {
		t.Error("Expected string interning to be enabled by default")
	}

	if !config.EnableCompaction {
		t.Error("Expected compaction to be enabled by default")
	}

	if config.LeakDetectionWindow <= 0 {
		t.Error("Expected leak detection window to be positive")
	}

	if config.MaxPoolSize <= 0 {
		t.Error("Expected max pool size to be positive")
	}
}

func TestOptimizeMemoryUsage(t *testing.T) {
	optimizer := NewMemoryOptimizer(nil)
	ctx := context.Background()

	report, err := optimizer.OptimizeMemoryUsage(ctx)
	if err != nil {
		t.Fatalf("OptimizeMemoryUsage failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.StartTime.IsZero() {
		t.Error("Expected start time to be set")
	}

	if report.Duration <= 0 {
		t.Error("Expected duration to be positive")
	}

	if report.BeforeStats == nil {
		t.Error("Expected before stats to be non-nil")
	}

	if report.AfterStats == nil {
		t.Error("Expected after stats to be non-nil")
	}

	if report.Optimizations == nil {
		t.Error("Expected optimizations slice to be non-nil")
	}

	if report.LeaksDetected == nil {
		t.Error("Expected leaks detected slice to be non-nil")
	}

	if report.Recommendations == nil {
		t.Error("Expected recommendations slice to be non-nil")
	}
}

func TestOptimizeMemoryUsageWithCancellation(t *testing.T) {
	optimizer := NewMemoryOptimizer(nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	report, err := optimizer.OptimizeMemoryUsage(ctx)

	// Should still complete because the context is only checked during optimization steps
	if err != nil && report == nil {
		t.Error("Expected optimization to handle cancellation gracefully")
	}
}

func TestCreatePool(t *testing.T) {
	optimizer := NewMemoryOptimizer(nil)

	// Test slice pool creation
	sliceType := reflect.TypeOf([]int{})
	if !optimizer.createPool(sliceType) {
		t.Error("Expected slice pool creation to succeed")
	}

	// Test struct pool creation
	structType := reflect.TypeOf(struct{ X int }{})
	if !optimizer.createPool(structType) {
		t.Error("Expected struct pool creation to succeed")
	}

	// Test map pool creation
	mapType := reflect.TypeOf(map[string]int{})
	if !optimizer.createPool(mapType) {
		t.Error("Expected map pool creation to succeed")
	}

	// Test unsupported type
	intType := reflect.TypeOf(42)
	if optimizer.createPool(intType) {
		t.Error("Expected int pool creation to fail")
	}

	// Test duplicate pool creation
	if optimizer.createPool(sliceType) {
		t.Error("Expected duplicate pool creation to fail")
	}
}

func TestGetFromPoolPutToPool(t *testing.T) {
	optimizer := NewMemoryOptimizer(nil)

	// Create pool for slice type
	sliceType := reflect.TypeOf([]int{})
	optimizer.createPool(sliceType)

	// Get from pool
	obj := optimizer.GetFromPool(sliceType)
	if obj == nil {
		t.Error("Expected object from pool")
	}

	// Verify it's the right type
	if reflect.TypeOf(obj) != sliceType {
		t.Error("Expected object to be of slice type")
	}

	// Put back to pool
	optimizer.PutToPool(obj)

	// Test with non-existent pool
	intType := reflect.TypeOf(42)
	obj = optimizer.GetFromPool(intType)
	if obj != nil {
		t.Error("Expected nil for non-existent pool")
	}
}

func TestStringInterner(t *testing.T) {
	interner := NewStringInterner()

	if interner == nil {
		t.Fatal("Expected interner to be non-nil")
	}

	s1 := "test string"
	s2 := "test string"

	// Intern both strings
	interned1 := interner.Intern(s1)
	interned2 := interner.Intern(s2)

	// They should be the same reference
	if interned1 != interned2 {
		t.Error("Expected interned strings to be the same reference")
	}

	// Test stats
	stats := interner.GetStats()
	if stats == nil {
		t.Error("Expected stats to be non-nil")
	}

	if internedCount, ok := stats["interned_strings"].(int); !ok || internedCount <= 0 {
		t.Error("Expected positive interned strings count")
	}
}

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()

	if pool == nil {
		t.Fatal("Expected buffer pool to be non-nil")
	}

	// Test getting buffers of different sizes
	sizes := []int{64, 128, 256, 512, 1024}

	for _, size := range sizes {
		buf := pool.Get(size)
		if buf == nil {
			t.Errorf("Expected buffer for size %d", size)
			continue
		}

		if buf.Cap() < size {
			t.Errorf("Expected buffer capacity >= %d, got %d", size, buf.Cap())
		}

		// Write some data
		buf.WriteString("test data")

		// Put back to pool
		pool.Put(buf)

		// Get again - should be reset
		buf2 := pool.Get(size)
		if buf2.Len() != 0 {
			t.Error("Expected buffer to be reset when retrieved from pool")
		}

		pool.Put(buf2)
	}

	// Test with nil buffer
	pool.Put(nil) // Should not panic
}

func TestMemoryProfiler(t *testing.T) {
	profiler := NewMemoryProfiler()

	if profiler == nil {
		t.Fatal("Expected profiler to be non-nil")
	}

	profile := profiler.Profile()
	if profile == nil {
		t.Fatal("Expected profile to be non-nil")
	}

	if profile.Stats == nil {
		t.Error("Expected stats to be non-nil")
	}

	if profile.Allocations == nil {
		t.Error("Expected allocations to be non-nil")
	}

	if profile.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	// Verify some basic stats are present
	stats := profile.Stats
	if stats.Allocated == 0 {
		t.Error("Expected allocated memory to be non-zero")
	}

	if stats.TotalAlloc == 0 {
		t.Error("Expected total allocated memory to be non-zero")
	}
}

func TestAllocationAnalyzer(t *testing.T) {
	analyzer := NewAllocationAnalyzer()

	if analyzer == nil {
		t.Fatal("Expected analyzer to be non-nil")
	}

	// Create a mock profile
	profile := &MemoryProfile{
		Stats:       &MemoryStats{Allocated: 1024},
		Allocations: []Allocation{},
		Timestamp:   time.Now(),
	}

	allocations := analyzer.AnalyzeAllocations(profile)
	if allocations == nil {
		t.Error("Expected allocations to be non-nil")
	}

	// The mock implementation returns some default allocations
	if len(allocations) == 0 {
		t.Error("Expected some default allocations")
	}

	// Verify allocation structure
	for _, alloc := range allocations {
		if alloc.Type == nil {
			t.Error("Expected allocation type to be non-nil")
		}

		if alloc.Size <= 0 {
			t.Error("Expected allocation size to be positive")
		}

		if alloc.Count <= 0 {
			t.Error("Expected allocation count to be positive")
		}

		if alloc.TotalSize <= 0 {
			t.Error("Expected total allocation size to be positive")
		}

		if alloc.Location == "" {
			t.Error("Expected allocation location to be non-empty")
		}
	}
}

func TestObjectCompactor(t *testing.T) {
	compactor := NewObjectCompactor()

	if compactor == nil {
		t.Fatal("Expected compactor to be non-nil")
	}

	err := compactor.Compact()
	if err != nil {
		t.Errorf("Compact failed: %v", err)
	}
}

func TestCalculateLeakSeverity(t *testing.T) {
	optimizer := NewMemoryOptimizer(nil)

	tests := []struct {
		size     int64
		expected string
	}{
		{200 * 1024 * 1024, "critical"}, // 200MB
		{50 * 1024 * 1024, "high"},      // 50MB
		{5 * 1024 * 1024, "medium"},     // 5MB
		{500 * 1024, "low"},             // 500KB
	}

	for _, test := range tests {
		result := optimizer.calculateLeakSeverity(test.size)
		if result != test.expected {
			t.Errorf("calculateLeakSeverity(%d) = %s, expected %s",
				test.size, result, test.expected)
		}
	}
}

func TestOptimizationStats(t *testing.T) {
	stats := NewOptimizationStats()

	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}

	// Create a mock report
	report := &OptimizationReport{
		StartTime:        time.Now(),
		Duration:         time.Millisecond * 100,
		MemorySaved:      1024 * 1024, // 1MB
		ReductionPercent: 10.0,
	}

	stats.RecordOptimization(report)

	// Get stats
	statMap := stats.GetStats()
	if statMap == nil {
		t.Error("Expected stat map to be non-nil")
	}

	if total, ok := statMap["total_optimizations"].(int64); !ok || total != 1 {
		t.Error("Expected total optimizations to be 1")
	}

	if saved, ok := statMap["total_memory_saved"].(int64); !ok || saved != 1024*1024 {
		t.Error("Expected total memory saved to be 1MB")
	}
}

func TestMMapCache(t *testing.T) {
	cache := NewMMapCache()

	if cache == nil {
		t.Fatal("Expected mmap cache to be non-nil")
	}

	// Test mapping non-existent file
	_, err := cache.Map("/non/existent/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Benchmark tests
func BenchmarkOptimizeMemoryUsage(b *testing.B) {
	optimizer := NewMemoryOptimizer(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.OptimizeMemoryUsage(ctx)
	}
}

func BenchmarkStringInterner(b *testing.B) {
	interner := NewStringInterner()
	testString := "benchmark test string"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interner.Intern(testString)
	}
}

func BenchmarkBufferPoolGet(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(1024)
		pool.Put(buf)
	}
}

func BenchmarkMemoryProfile(b *testing.B) {
	profiler := NewMemoryProfiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.Profile()
	}
}

// Test edge cases
func TestOptimizeMemoryUsageEdgeCases(t *testing.T) {
	// Test with disabled optimizations
	config := &OptimizerConfig{
		EnablePooling:         false,
		EnableStringInterning: false,
		EnableCompaction:      false,
		EnableMMap:            false,
		LeakDetectionWindow:   time.Hour,
		PoolGCInterval:        time.Minute,
		MaxPoolSize:           1000,
		CompactionThreshold:   0.3,
	}

	optimizer := NewMemoryOptimizer(config)
	ctx := context.Background()

	report, err := optimizer.OptimizeMemoryUsage(ctx)
	if err != nil {
		t.Errorf("OptimizeMemoryUsage with disabled optimizations failed: %v", err)
	}

	if report == nil {
		t.Error("Expected report even with disabled optimizations")
	}

	// Should have fewer or no optimizations applied
	optimizationCount := len(report.Optimizations)
	if optimizationCount > 1 {
		t.Errorf("Expected fewer optimizations with disabled config, got %d", optimizationCount)
	}
}

func TestBufferPoolEdgeCases(t *testing.T) {
	pool := NewBufferPool()

	// Test with very large size
	buf := pool.Get(10 * 1024 * 1024) // 10MB
	if buf == nil {
		t.Error("Expected buffer even for large size")
	}
	pool.Put(buf)

	// Test with zero size
	buf = pool.Get(0)
	if buf == nil {
		t.Error("Expected buffer even for zero size")
	}
	pool.Put(buf)
}

func TestStringInternerConcurrency(t *testing.T) {
	interner := NewStringInterner()
	testString := "concurrent test string"

	// Run multiple goroutines concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				interner.Intern(testString)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// All should have the same reference
	result1 := interner.Intern(testString)
	result2 := interner.Intern(testString)

	if result1 != result2 {
		t.Error("Expected same reference after concurrent access")
	}
}
