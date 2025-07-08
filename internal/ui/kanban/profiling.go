// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// KanbanProfiler provides performance profiling for the kanban TUI
type KanbanProfiler struct {
	mu           sync.RWMutex
	renderTimes  []time.Duration
	allocations  []runtime.MemStats
	frameDrops   int
	targetFPS    int
	maxSamples   int
	enabled      bool
	startTime    time.Time
	totalFrames  int64
	lastGCPause  time.Duration
	peakMemUsage uint64
}

// FrameMetrics captures performance data for a single frame
type FrameMetrics struct {
	RenderTime    time.Duration
	AllocBytes    uint64
	HeapSize      uint64
	NumGoroutines int
	GCPauseTime   time.Duration
	DroppedFrame  bool
}

// ProfileReport contains aggregated performance metrics
type ProfileReport struct {
	AvgRenderTime    time.Duration
	MaxRenderTime    time.Duration
	MinRenderTime    time.Duration
	FrameDropRate    float64
	AvgMemUsage      uint64
	PeakMemUsage     uint64
	TotalFrames      int64
	ProfileDuration  time.Duration
	TargetFPS        int
	ActualFPS        float64
	RecommendedCards int
}

// NewKanbanProfiler creates a new performance profiler
func NewKanbanProfiler(targetFPS int) *KanbanProfiler {
	return &KanbanProfiler{
		targetFPS:  targetFPS,
		maxSamples: 1000, // Keep last 1000 samples
		enabled:    false,
		startTime:  time.Now(),
	}
}

// Enable activates profiling
func (kp *KanbanProfiler) Enable() {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	kp.enabled = true
	kp.startTime = time.Now()
	kp.renderTimes = make([]time.Duration, 0, kp.maxSamples)
	kp.allocations = make([]runtime.MemStats, 0, kp.maxSamples)
	kp.frameDrops = 0
	kp.totalFrames = 0
}

// Disable deactivates profiling
func (kp *KanbanProfiler) Disable() {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	kp.enabled = false
}

// IsEnabled returns whether profiling is active
func (kp *KanbanProfiler) IsEnabled() bool {
	kp.mu.RLock()
	defer kp.mu.RUnlock()
	return kp.enabled
}

// StartFrame begins profiling a render frame and returns a completion function
func (kp *KanbanProfiler) StartFrame(ctx context.Context) func() {
	if !kp.IsEnabled() {
		return func() {} // No-op when disabled
	}

	start := time.Now()
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	_ = runtime.NumGoroutine() // Track goroutines but don't use in this implementation

	return func() {
		if !kp.IsEnabled() {
			return
		}

		duration := time.Since(start)
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)

		kp.mu.Lock()
		defer kp.mu.Unlock()

		// Check for context cancellation
		if ctx.Err() != nil {
			return
		}

		kp.totalFrames++

		// Store render time (with circular buffer)
		if len(kp.renderTimes) >= kp.maxSamples {
			kp.renderTimes = kp.renderTimes[1:]
		}
		kp.renderTimes = append(kp.renderTimes, duration)

		// Store memory stats (with circular buffer)
		if len(kp.allocations) >= kp.maxSamples {
			kp.allocations = kp.allocations[1:]
		}
		kp.allocations = append(kp.allocations, endMem)

		// Track peak memory usage
		if endMem.HeapInuse > kp.peakMemUsage {
			kp.peakMemUsage = endMem.HeapInuse
		}

		// Check if we missed target FPS
		targetFrameTime := time.Second / time.Duration(kp.targetFPS)
		if duration > targetFrameTime {
			kp.frameDrops++
		}

		// Track GC pause time
		if endMem.PauseTotalNs > startMem.PauseTotalNs {
			kp.lastGCPause = time.Duration(endMem.PauseTotalNs - startMem.PauseTotalNs)
		}

		// Log warning if frame time is excessive
		if duration > 100*time.Millisecond {
			// This would be logged to the observability system
			// gerror.Warn().WithComponent("kanban.profiler").
			//   WithOperation("StartFrame").
			//   WithDetails("frame_time", duration.String()).
			//   Log("Excessive frame render time detected")
		}
	}
}

// GetCurrentMetrics returns the latest frame metrics
func (kp *KanbanProfiler) GetCurrentMetrics(ctx context.Context) (*FrameMetrics, error) {
	if !kp.IsEnabled() {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "profiler not enabled", nil).
			WithComponent("kanban.profiler").
			WithOperation("GetCurrentMetrics")
	}

	kp.mu.RLock()
	defer kp.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var lastRenderTime time.Duration
	if len(kp.renderTimes) > 0 {
		lastRenderTime = kp.renderTimes[len(kp.renderTimes)-1]
	}

	targetFrameTime := time.Second / time.Duration(kp.targetFPS)
	droppedFrame := lastRenderTime > targetFrameTime

	return &FrameMetrics{
		RenderTime:    lastRenderTime,
		AllocBytes:    mem.TotalAlloc,
		HeapSize:      mem.HeapInuse,
		NumGoroutines: runtime.NumGoroutine(),
		GCPauseTime:   kp.lastGCPause,
		DroppedFrame:  droppedFrame,
	}, nil
}

// GenerateReport creates a comprehensive performance report
func (kp *KanbanProfiler) GenerateReport(ctx context.Context) (*ProfileReport, error) {
	if !kp.IsEnabled() {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "profiler not enabled", nil).
			WithComponent("kanban.profiler").
			WithOperation("GenerateReport")
	}

	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if len(kp.renderTimes) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no profiling data available", nil).
			WithComponent("kanban.profiler").
			WithOperation("GenerateReport")
	}

	// Calculate render time statistics
	var totalTime time.Duration
	minTime := kp.renderTimes[0]
	maxTime := kp.renderTimes[0]

	for _, t := range kp.renderTimes {
		totalTime += t
		if t < minTime {
			minTime = t
		}
		if t > maxTime {
			maxTime = t
		}
	}

	avgRenderTime := totalTime / time.Duration(len(kp.renderTimes))

	// Calculate memory statistics
	var totalMem uint64
	if len(kp.allocations) > 0 {
		for _, mem := range kp.allocations {
			totalMem += mem.HeapInuse
		}
	}
	avgMemUsage := totalMem / uint64(len(kp.allocations))

	// Calculate actual FPS
	profileDuration := time.Since(kp.startTime)
	actualFPS := float64(kp.totalFrames) / profileDuration.Seconds()

	// Calculate frame drop rate
	frameDropRate := float64(kp.frameDrops) / float64(kp.totalFrames) * 100.0

	// Calculate recommended card count based on performance
	recommendedCards := kp.calculateRecommendedCards(avgRenderTime)

	return &ProfileReport{
		AvgRenderTime:    avgRenderTime,
		MaxRenderTime:    maxTime,
		MinRenderTime:    minTime,
		FrameDropRate:    frameDropRate,
		AvgMemUsage:      avgMemUsage,
		PeakMemUsage:     kp.peakMemUsage,
		TotalFrames:      kp.totalFrames,
		ProfileDuration:  profileDuration,
		TargetFPS:        kp.targetFPS,
		ActualFPS:        actualFPS,
		RecommendedCards: recommendedCards,
	}, nil
}

// calculateRecommendedCards estimates optimal card count based on render performance
func (kp *KanbanProfiler) calculateRecommendedCards(avgRenderTime time.Duration) int {
	targetFrameTime := time.Second / time.Duration(kp.targetFPS)

	// If we're hitting target consistently, we can handle more cards
	if avgRenderTime < targetFrameTime/2 {
		return 500 // Can handle large boards
	} else if avgRenderTime < targetFrameTime {
		return 200 // Optimal performance range
	} else if avgRenderTime < targetFrameTime*2 {
		return 100 // Reduce card count for better performance
	} else {
		return 50 // Performance issues, minimal cards recommended
	}
}

// Reset clears all profiling data
func (kp *KanbanProfiler) Reset() {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	kp.renderTimes = kp.renderTimes[:0]
	kp.allocations = kp.allocations[:0]
	kp.frameDrops = 0
	kp.totalFrames = 0
	kp.startTime = time.Now()
	kp.peakMemUsage = 0
	kp.lastGCPause = 0
}

// GetMemoryPressure returns current memory pressure level (0.0 to 1.0)
func (kp *KanbanProfiler) GetMemoryPressure(ctx context.Context) float64 {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// Simple heuristic: ratio of heap in use to system memory
	// This is a basic implementation - in production, you'd want more sophisticated metrics
	systemMemLimit := uint64(1024 * 1024 * 1024) // 1GB baseline
	if mem.Sys > systemMemLimit {
		systemMemLimit = mem.Sys
	}

	pressure := float64(mem.HeapInuse) / float64(systemMemLimit)
	if pressure > 1.0 {
		pressure = 1.0
	}

	return pressure
}

// ShouldReduceQuality suggests if render quality should be reduced for performance
func (kp *KanbanProfiler) ShouldReduceQuality(ctx context.Context) bool {
	if !kp.IsEnabled() {
		return false
	}

	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Check recent frame drops
	recentDrops := 0
	recentFrames := 10
	if len(kp.renderTimes) >= recentFrames {
		targetFrameTime := time.Second / time.Duration(kp.targetFPS)
		start := len(kp.renderTimes) - recentFrames

		for i := start; i < len(kp.renderTimes); i++ {
			if kp.renderTimes[i] > targetFrameTime {
				recentDrops++
			}
		}
	}

	// Reduce quality if more than 30% of recent frames were dropped
	dropRate := float64(recentDrops) / float64(recentFrames)
	return dropRate > 0.3
}

// GetDebugInfo returns debug information for display in the TUI
func (kp *KanbanProfiler) GetDebugInfo(ctx context.Context) map[string]interface{} {
	if !kp.IsEnabled() {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	kp.mu.RLock()
	defer kp.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	info := map[string]interface{}{
		"enabled":        true,
		"total_frames":   kp.totalFrames,
		"frame_drops":    kp.frameDrops,
		"sample_count":   len(kp.renderTimes),
		"heap_size_mb":   float64(mem.HeapInuse) / 1024 / 1024,
		"peak_memory_mb": float64(kp.peakMemUsage) / 1024 / 1024,
		"goroutines":     runtime.NumGoroutine(),
		"target_fps":     kp.targetFPS,
	}

	if len(kp.renderTimes) > 0 {
		latest := kp.renderTimes[len(kp.renderTimes)-1]
		info["last_render_ms"] = float64(latest.Nanoseconds()) / 1e6
	}

	return info
}
