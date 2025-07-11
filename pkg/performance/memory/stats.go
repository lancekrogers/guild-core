package memory

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Stats provides comprehensive memory statistics and monitoring
type Stats struct {
	allocations      atomic.Uint64
	deallocations    atomic.Uint64
	bytesAllocated   atomic.Uint64
	bytesDeallocated atomic.Uint64
	peakUsage        atomic.Uint64
	currentUsage     atomic.Uint64

	// GC stats
	gcCount      atomic.Uint64
	gcPauseTotal atomic.Uint64
	lastGCTime   atomic.Int64

	// Pool stats
	poolHits   atomic.Uint64
	poolMisses atomic.Uint64

	startTime time.Time
}

// NewStats creates a new memory statistics tracker
func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

// RecordAllocation records a memory allocation
func (s *Stats) RecordAllocation(size uint64) {
	s.allocations.Add(1)
	s.bytesAllocated.Add(size)

	current := s.currentUsage.Add(size)

	// Update peak usage
	for {
		peak := s.peakUsage.Load()
		if current <= peak {
			break
		}
		if s.peakUsage.CompareAndSwap(peak, current) {
			break
		}
	}
}

// RecordDeallocation records a memory deallocation
func (s *Stats) RecordDeallocation(size uint64) {
	s.deallocations.Add(1)
	s.bytesDeallocated.Add(size)
	s.currentUsage.Add(^(size - 1)) // Atomic subtract
}

// RecordGC records garbage collection statistics
func (s *Stats) RecordGC(count uint64, pauseNs uint64) {
	s.gcCount.Store(count)
	s.gcPauseTotal.Store(pauseNs)
	s.lastGCTime.Store(time.Now().UnixNano())
}

// RecordPoolHit records a pool cache hit
func (s *Stats) RecordPoolHit() {
	s.poolHits.Add(1)
}

// RecordPoolMiss records a pool cache miss
func (s *Stats) RecordPoolMiss() {
	s.poolMisses.Add(1)
}

// Snapshot returns a snapshot of current statistics
func (s *Stats) Snapshot() StatsSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return StatsSnapshot{
		// Custom stats
		Allocations:      s.allocations.Load(),
		Deallocations:    s.deallocations.Load(),
		BytesAllocated:   s.bytesAllocated.Load(),
		BytesDeallocated: s.bytesDeallocated.Load(),
		PeakUsage:        s.peakUsage.Load(),
		CurrentUsage:     s.currentUsage.Load(),

		// Pool stats
		PoolHits:   s.poolHits.Load(),
		PoolMisses: s.poolMisses.Load(),

		// Runtime stats
		RuntimeStats: RuntimeMemStats{
			Alloc:         m.Alloc,
			TotalAlloc:    m.TotalAlloc,
			Sys:           m.Sys,
			Lookups:       m.Lookups,
			Mallocs:       m.Mallocs,
			Frees:         m.Frees,
			HeapAlloc:     m.HeapAlloc,
			HeapSys:       m.HeapSys,
			HeapIdle:      m.HeapIdle,
			HeapInuse:     m.HeapInuse,
			HeapReleased:  m.HeapReleased,
			HeapObjects:   m.HeapObjects,
			StackInuse:    m.StackInuse,
			StackSys:      m.StackSys,
			MSpanInuse:    m.MSpanInuse,
			MSpanSys:      m.MSpanSys,
			MCacheInuse:   m.MCacheInuse,
			MCacheSys:     m.MCacheSys,
			BuckHashSys:   m.BuckHashSys,
			GCSys:         m.GCSys,
			OtherSys:      m.OtherSys,
			NextGC:        m.NextGC,
			LastGC:        m.LastGC,
			PauseTotalNs:  m.PauseTotalNs,
			NumGC:         uint64(m.NumGC),
			NumForcedGC:   uint64(m.NumForcedGC),
			GCCPUFraction: m.GCCPUFraction,
		},

		// Calculated fields
		PoolHitRate: s.calculatePoolHitRate(),
		Uptime:      time.Since(s.startTime),
		Timestamp:   time.Now(),
	}
}

// calculatePoolHitRate calculates the pool hit rate
func (s *Stats) calculatePoolHitRate() float64 {
	hits := s.poolHits.Load()
	misses := s.poolMisses.Load()
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

// StatsSnapshot represents a point-in-time snapshot of memory statistics
type StatsSnapshot struct {
	// Custom allocation tracking
	Allocations      uint64
	Deallocations    uint64
	BytesAllocated   uint64
	BytesDeallocated uint64
	PeakUsage        uint64
	CurrentUsage     uint64

	// Pool statistics
	PoolHits    uint64
	PoolMisses  uint64
	PoolHitRate float64

	// Runtime statistics
	RuntimeStats RuntimeMemStats

	// Metadata
	Uptime    time.Duration
	Timestamp time.Time
}

// RuntimeMemStats mirrors runtime.MemStats with selected fields
type RuntimeMemStats struct {
	Alloc         uint64
	TotalAlloc    uint64
	Sys           uint64
	Lookups       uint64
	Mallocs       uint64
	Frees         uint64
	HeapAlloc     uint64
	HeapSys       uint64
	HeapIdle      uint64
	HeapInuse     uint64
	HeapReleased  uint64
	HeapObjects   uint64
	StackInuse    uint64
	StackSys      uint64
	MSpanInuse    uint64
	MSpanSys      uint64
	MCacheInuse   uint64
	MCacheSys     uint64
	BuckHashSys   uint64
	GCSys         uint64
	OtherSys      uint64
	NextGC        uint64
	LastGC        uint64
	PauseTotalNs  uint64
	NumGC         uint64
	NumForcedGC   uint64
	GCCPUFraction float64
}

// MemoryPressure calculates memory pressure as a percentage
func (s *StatsSnapshot) MemoryPressure() float64 {
	if s.RuntimeStats.NextGC == 0 {
		return 0
	}

	return float64(s.RuntimeStats.HeapAlloc) / float64(s.RuntimeStats.NextGC) * 100
}

// AllocationRate returns allocations per second
func (s *StatsSnapshot) AllocationRate() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}

	return float64(s.Allocations) / s.Uptime.Seconds()
}

// Monitor provides continuous memory monitoring
type Monitor struct {
	stats    *Stats
	interval time.Duration
	handlers []MonitorHandler
	done     chan struct{}
}

// MonitorHandler handles monitoring events
type MonitorHandler interface {
	OnSnapshot(snapshot StatsSnapshot)
	OnThreshold(threshold ThresholdEvent)
}

// ThresholdEvent represents a threshold violation
type ThresholdEvent struct {
	Type      ThresholdType
	Threshold uint64
	Current   uint64
	Timestamp time.Time
}

// ThresholdType represents the type of threshold
type ThresholdType int

const (
	ThresholdMemoryUsage ThresholdType = iota
	ThresholdAllocationRate
	ThresholdGCFrequency
	ThresholdPoolMissRate
)

// NewMonitor creates a new memory monitor
func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		stats:    NewStats(),
		interval: interval,
		done:     make(chan struct{}),
	}
}

// AddHandler adds a monitoring handler
func (m *Monitor) AddHandler(handler MonitorHandler) {
	m.handlers = append(m.handlers, handler)
}

// Start begins monitoring
func (m *Monitor) Start() {
	go m.monitorLoop()
}

// Stop stops monitoring
func (m *Monitor) Stop() {
	close(m.done)
}

// monitorLoop runs the monitoring loop
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snapshot := m.stats.Snapshot()

			// Notify handlers
			for _, handler := range m.handlers {
				handler.OnSnapshot(snapshot)
			}

			// Check thresholds
			m.checkThresholds(snapshot)

		case <-m.done:
			return
		}
	}
}

// checkThresholds checks various memory thresholds
func (m *Monitor) checkThresholds(snapshot StatsSnapshot) {
	// Check memory usage threshold (example: 1GB)
	if snapshot.RuntimeStats.HeapAlloc > 1024*1024*1024 {
		event := ThresholdEvent{
			Type:      ThresholdMemoryUsage,
			Threshold: 1024 * 1024 * 1024,
			Current:   snapshot.RuntimeStats.HeapAlloc,
			Timestamp: time.Now(),
		}

		for _, handler := range m.handlers {
			handler.OnThreshold(event)
		}
	}

	// Check pool miss rate threshold (example: 10%)
	if snapshot.PoolHitRate < 0.9 && snapshot.PoolHits+snapshot.PoolMisses > 100 {
		event := ThresholdEvent{
			Type:      ThresholdPoolMissRate,
			Threshold: 90, // 90% hit rate
			Current:   uint64(snapshot.PoolHitRate * 100),
			Timestamp: time.Now(),
		}

		for _, handler := range m.handlers {
			handler.OnThreshold(event)
		}
	}
}

// LoggingHandler provides a simple logging handler
type LoggingHandler struct{}

// OnSnapshot logs memory snapshots
func (lh *LoggingHandler) OnSnapshot(snapshot StatsSnapshot) {
	// Implementation would log the snapshot
	// Using a structured logger in production
}

// OnThreshold logs threshold violations
func (lh *LoggingHandler) OnThreshold(event ThresholdEvent) {
	// Implementation would log the threshold event
	// Using a structured logger in production
}

// Profiler provides memory profiling utilities
type Profiler struct {
	samples    []StatsSnapshot
	maxSamples int
}

// NewProfiler creates a new memory profiler
func NewProfiler(maxSamples int) *Profiler {
	if maxSamples <= 0 {
		maxSamples = 1000
	}

	return &Profiler{
		samples:    make([]StatsSnapshot, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// Sample takes a memory profile sample
func (p *Profiler) Sample(stats *Stats) {
	snapshot := stats.Snapshot()

	if len(p.samples) >= p.maxSamples {
		// Remove oldest sample
		copy(p.samples, p.samples[1:])
		p.samples = p.samples[:len(p.samples)-1]
	}

	p.samples = append(p.samples, snapshot)
}

// Analysis returns analysis of collected samples
func (p *Profiler) Analysis() ProfileAnalysis {
	if len(p.samples) == 0 {
		return ProfileAnalysis{}
	}

	first := p.samples[0]
	last := p.samples[len(p.samples)-1]

	return ProfileAnalysis{
		SampleCount:        len(p.samples),
		Duration:           last.Timestamp.Sub(first.Timestamp),
		AllocationGrowth:   last.Allocations - first.Allocations,
		PeakMemoryUsage:    p.findPeakUsage(),
		AveragePoolHitRate: p.calculateAveragePoolHitRate(),
	}
}

// findPeakUsage finds the peak memory usage across samples
func (p *Profiler) findPeakUsage() uint64 {
	var peak uint64

	for _, sample := range p.samples {
		if sample.RuntimeStats.HeapAlloc > peak {
			peak = sample.RuntimeStats.HeapAlloc
		}
	}

	return peak
}

// calculateAveragePoolHitRate calculates average pool hit rate
func (p *Profiler) calculateAveragePoolHitRate() float64 {
	if len(p.samples) == 0 {
		return 0
	}

	total := 0.0
	for _, sample := range p.samples {
		total += sample.PoolHitRate
	}

	return total / float64(len(p.samples))
}

// ProfileAnalysis provides analysis of memory profile samples
type ProfileAnalysis struct {
	SampleCount        int
	Duration           time.Duration
	AllocationGrowth   uint64
	PeakMemoryUsage    uint64
	AveragePoolHitRate float64
}
