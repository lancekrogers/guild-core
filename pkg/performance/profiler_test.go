// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package performance

import (
	"context"
	"testing"
	"time"
)

func TestNewPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler()

	if profiler == nil {
		t.Fatal("Expected profiler to be non-nil")
	}

	if profiler.cpuProfiler == nil {
		t.Error("Expected CPU profiler to be initialized")
	}

	if profiler.memProfiler == nil {
		t.Error("Expected memory profiler to be initialized")
	}

	if profiler.traceProfiler == nil {
		t.Error("Expected trace profiler to be initialized")
	}

	if profiler.benchmarks == nil {
		t.Error("Expected benchmarks map to be initialized")
	}

	if profiler.hotspots == nil {
		t.Error("Expected hotspots detector to be initialized")
	}
}

func TestProfileApplication(t *testing.T) {
	profiler := NewPerformanceProfiler()
	ctx := context.Background()

	// Test with very short duration to avoid long test times
	duration := time.Millisecond * 100

	report, err := profiler.ProfileApplication(ctx, duration)
	if err != nil {
		t.Fatalf("ProfileApplication failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.StartTime.IsZero() {
		t.Error("Expected report start time to be set")
	}

	if report.Duration < duration {
		t.Error("Expected report duration to be at least the requested duration")
	}

	// Test cancellation
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = profiler.ProfileApplication(cancelCtx, time.Second)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestProfileApplicationConcurrency(t *testing.T) {
	profiler := NewPerformanceProfiler()
	ctx := context.Background()

	// Start first profiling session
	go func() {
		profiler.ProfileApplication(ctx, time.Millisecond*100)
	}()

	// Wait a bit to ensure first session starts
	time.Sleep(time.Millisecond * 10)

	// Try to start second session - should fail
	_, err := profiler.ProfileApplication(ctx, time.Millisecond*50)
	if err == nil {
		t.Error("Expected error when trying to start concurrent profiling session")
	}
}

func TestRunBenchmark(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Simple benchmark function
	benchFunc := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i * 2
		}
	}

	result := profiler.RunBenchmark("test-benchmark", benchFunc)

	if result == nil {
		t.Fatal("Expected benchmark result to be non-nil")
	}

	if result.N <= 0 {
		t.Error("Expected benchmark iterations to be positive")
	}

	if result.NsPerOp <= 0 {
		t.Error("Expected nanoseconds per operation to be positive")
	}
}

func TestCompareBenchmarks(t *testing.T) {
	profiler := NewPerformanceProfiler()

	benchFunc := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i * 2
		}
	}

	// Run initial benchmark
	profiler.RunBenchmark("test-benchmark", benchFunc)

	// Set baseline
	err := profiler.SetBaseline("test-benchmark")
	if err != nil {
		t.Fatalf("SetBaseline failed: %v", err)
	}

	// Run benchmark again
	profiler.RunBenchmark("test-benchmark", benchFunc)

	// Compare results
	comparison := profiler.CompareBenchmarks("test-benchmark")
	if comparison == nil {
		t.Fatal("Expected comparison to be non-nil")
	}

	if comparison.Name != "test-benchmark" {
		t.Error("Expected comparison name to match benchmark name")
	}

	if comparison.Current == nil {
		t.Error("Expected current result to be non-nil")
	}

	if comparison.Baseline == nil {
		t.Error("Expected baseline result to be non-nil")
	}
}

func TestSetBaseline(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Try to set baseline for non-existent benchmark
	err := profiler.SetBaseline("non-existent")
	if err == nil {
		t.Error("Expected error when setting baseline for non-existent benchmark")
	}

	// Run benchmark and set baseline
	benchFunc := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i * 2
		}
	}

	profiler.RunBenchmark("test-benchmark", benchFunc)

	err = profiler.SetBaseline("test-benchmark")
	if err != nil {
		t.Errorf("SetBaseline failed: %v", err)
	}
}

func TestHotPathOptimizer(t *testing.T) {
	profiler := NewPerformanceProfiler()
	optimizer := NewHotPathOptimizer(profiler)

	if optimizer == nil {
		t.Fatal("Expected optimizer to be non-nil")
	}

	if optimizer.profiler != profiler {
		t.Error("Expected optimizer profiler to match input profiler")
	}

	// Test optimization with empty hotspots
	ctx := context.Background()
	results, err := optimizer.OptimizeHotPaths(ctx, []Hotspot{})

	if err != nil {
		t.Errorf("OptimizeHotPaths failed: %v", err)
	}

	if results == nil {
		t.Error("Expected results to be non-nil")
	}

	if len(results) != 0 {
		t.Error("Expected empty results for empty hotspots")
	}
}

func TestHotPathOptimizerWithHotspots(t *testing.T) {
	profiler := NewPerformanceProfiler()
	optimizer := NewHotPathOptimizer(profiler)

	hotspots := []Hotspot{
		{
			Function:   "testFunction",
			File:       "test.go",
			Line:       42,
			CPUTime:    time.Millisecond * 500,
			Percentage: 15.0,
			Calls:      1000,
			Severity:   "high",
		},
	}

	ctx := context.Background()
	results, err := optimizer.OptimizeHotPaths(ctx, hotspots)

	if err != nil {
		t.Errorf("OptimizeHotPaths failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected optimization results for hotspots")
	}

	// Check that we get at least one optimization suggestion
	foundOptimization := false
	for _, result := range results {
		if result.Type != "" && result.Description != "" {
			foundOptimization = true
			break
		}
	}

	if !foundOptimization {
		t.Error("Expected at least one valid optimization result")
	}
}

func TestIsActive(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Initially should not be active
	if profiler.IsActive() {
		t.Error("Expected profiler to be inactive initially")
	}

	// Start profiling in goroutine
	ctx := context.Background()
	go func() {
		profiler.ProfileApplication(ctx, time.Millisecond*100)
	}()

	// Wait a bit and check if active
	time.Sleep(time.Millisecond * 10)
	if !profiler.IsActive() {
		t.Error("Expected profiler to be active during profiling")
	}

	// Wait for profiling to complete
	time.Sleep(time.Millisecond * 200)
	if profiler.IsActive() {
		t.Error("Expected profiler to be inactive after profiling completes")
	}
}

func TestGetStats(t *testing.T) {
	profiler := NewPerformanceProfiler()

	stats := profiler.GetStats()
	if stats == nil {
		t.Fatal("Expected stats to be non-nil")
	}

	// Check required fields
	if _, ok := stats["active"]; !ok {
		t.Error("Expected active field in stats")
	}

	if _, ok := stats["benchmark_count"]; !ok {
		t.Error("Expected benchmark_count field in stats")
	}

	if _, ok := stats["goroutines"]; !ok {
		t.Error("Expected goroutines field in stats")
	}

	if _, ok := stats["memory_allocated"]; !ok {
		t.Error("Expected memory_allocated field in stats")
	}
}

func TestCalculateImprovement(t *testing.T) {
	profiler := NewPerformanceProfiler()

	tests := []struct {
		speedupFactor float64
		expected      string
	}{
		{2.5, "excellent"},
		{1.8, "good"},
		{1.2, "moderate"},
		{1.0, "neutral"},
		{0.8, "regression"},
	}

	for _, test := range tests {
		result := profiler.calculateImprovement(test.speedupFactor)
		if result != test.expected {
			t.Errorf("calculateImprovement(%.1f) = %s, expected %s",
				test.speedupFactor, result, test.expected)
		}
	}
}

func TestProfilerStop(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Test stopping when not active
	err := profiler.Stop()
	if err == nil {
		t.Error("Expected error when stopping inactive profiler")
	}

	// Start profiling and stop
	ctx := context.Background()
	go func() {
		profiler.ProfileApplication(ctx, time.Second)
	}()

	// Wait for profiling to start
	time.Sleep(time.Millisecond * 10)

	err = profiler.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if profiler.IsActive() {
		t.Error("Expected profiler to be inactive after stop")
	}
}

// Benchmark tests
func BenchmarkProfilerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewPerformanceProfiler()
	}
}

func BenchmarkGetStats(b *testing.B) {
	profiler := NewPerformanceProfiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = profiler.GetStats()
	}
}

// Test helper functions
func TestCalculateHotspotSeverity(t *testing.T) {
	profiler := NewPerformanceProfiler()

	tests := []struct {
		percentage float64
		expected   string
	}{
		{25.0, "critical"},
		{15.0, "high"},
		{8.0, "medium"},
		{3.0, "low"},
	}

	for _, test := range tests {
		result := profiler.calculateHotspotSeverity(test.percentage)
		if result != test.expected {
			t.Errorf("calculateHotspotSeverity(%.1f) = %s, expected %s",
				test.percentage, result, test.expected)
		}
	}
}
