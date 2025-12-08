package collectors

import (
	"context"
	"runtime"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// SystemCollector collects system-level metrics
type SystemCollector struct {
	cpuUsage    metric.Float64ObservableGauge
	memoryUsage metric.Int64ObservableGauge
	goroutines  metric.Int64ObservableGauge
	gcPauseTime metric.Float64ObservableGauge
	heapAlloc   metric.Int64ObservableGauge
	heapSys     metric.Int64ObservableGauge
	heapInuse   metric.Int64ObservableGauge
	stackInuse  metric.Int64ObservableGauge
	lastGCTime  metric.Int64ObservableGauge
	gcCount     metric.Int64ObservableCounter

	lastNumGC      uint32
	lastPauseTotal time.Duration
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

// Register registers all system metrics with the meter
func (s *SystemCollector) Register(meter metric.Meter) error {
	var err error

	// CPU metrics
	s.cpuUsage, err = meter.Float64ObservableGauge(
		"guild.system.cpu.usage",
		metric.WithDescription("Current CPU usage percentage"),
		metric.WithUnit("%"),
		metric.WithFloat64Callback(s.observeCPU),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create CPU usage gauge")
	}

	// Memory metrics
	s.memoryUsage, err = meter.Int64ObservableGauge(
		"guild.system.memory.usage",
		metric.WithDescription("Current memory usage in bytes"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(s.observeMemory),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create memory usage gauge")
	}

	// Heap metrics
	s.heapAlloc, err = meter.Int64ObservableGauge(
		"guild.system.heap.alloc",
		metric.WithDescription("Heap bytes allocated and in use"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(s.observeHeap),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create heap alloc gauge")
	}

	s.heapSys, err = meter.Int64ObservableGauge(
		"guild.system.heap.sys",
		metric.WithDescription("Heap bytes obtained from system"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(s.observeHeap),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create heap sys gauge")
	}

	s.heapInuse, err = meter.Int64ObservableGauge(
		"guild.system.heap.inuse",
		metric.WithDescription("Heap bytes in use"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(s.observeHeap),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create heap inuse gauge")
	}

	// Stack metrics
	s.stackInuse, err = meter.Int64ObservableGauge(
		"guild.system.stack.inuse",
		metric.WithDescription("Stack bytes in use"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(s.observeStack),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stack inuse gauge")
	}

	// Goroutine metrics
	s.goroutines, err = meter.Int64ObservableGauge(
		"guild.system.goroutines",
		metric.WithDescription("Number of goroutines"),
		metric.WithInt64Callback(s.observeGoroutines),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create goroutines gauge")
	}

	// GC metrics
	s.gcPauseTime, err = meter.Float64ObservableGauge(
		"guild.system.gc.pause",
		metric.WithDescription("GC pause duration"),
		metric.WithUnit("ms"),
		metric.WithFloat64Callback(s.observeGCPause),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create GC pause gauge")
	}

	s.lastGCTime, err = meter.Int64ObservableGauge(
		"guild.system.gc.last",
		metric.WithDescription("Time since last GC"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(s.observeLastGCTime),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create last GC time gauge")
	}

	s.gcCount, err = meter.Int64ObservableCounter(
		"guild.system.gc.count",
		metric.WithDescription("Number of completed GC cycles"),
		metric.WithInt64Callback(s.observeGCCount),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create GC count counter")
	}

	return nil
}

// observeCPU observes CPU metrics
func (s *SystemCollector) observeCPU(_ context.Context, observer metric.Float64Observer) error {
	// This is a simplified version. In production, you'd use more sophisticated CPU monitoring
	// For now, we'll report the number of CPUs as a proxy
	observer.Observe(float64(runtime.NumCPU()), metric.WithAttributes(
		attribute.String("type", "available_cores"),
	))
	return nil
}

// observeMemory observes memory metrics
func (s *SystemCollector) observeMemory(_ context.Context, observer metric.Int64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	observer.Observe(int64(m.Alloc), metric.WithAttributes(
		attribute.String("type", "alloc"),
	))
	observer.Observe(int64(m.TotalAlloc), metric.WithAttributes(
		attribute.String("type", "total_alloc"),
	))
	observer.Observe(int64(m.Sys), metric.WithAttributes(
		attribute.String("type", "sys"),
	))

	return nil
}

// observeHeap observes heap metrics
func (s *SystemCollector) observeHeap(_ context.Context, observer metric.Int64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	observer.Observe(int64(m.HeapAlloc))
	observer.Observe(int64(m.HeapSys))
	observer.Observe(int64(m.HeapInuse))

	return nil
}

// observeStack observes stack metrics
func (s *SystemCollector) observeStack(_ context.Context, observer metric.Int64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	observer.Observe(int64(m.StackInuse))

	return nil
}

// observeGoroutines observes goroutine count
func (s *SystemCollector) observeGoroutines(_ context.Context, observer metric.Int64Observer) error {
	observer.Observe(int64(runtime.NumGoroutine()))
	return nil
}

// observeGCPause observes GC pause time
func (s *SystemCollector) observeGCPause(_ context.Context, observer metric.Float64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate average pause time
	if m.NumGC > s.lastNumGC {
		gcCycles := m.NumGC - s.lastNumGC
		pauseTime := m.PauseTotalNs - uint64(s.lastPauseTotal)
		avgPause := float64(pauseTime) / float64(gcCycles) / 1e6 // Convert to milliseconds

		observer.Observe(avgPause)

		s.lastNumGC = m.NumGC
		s.lastPauseTotal = time.Duration(m.PauseTotalNs)
	}

	return nil
}

// observeLastGCTime observes time since last GC
func (s *SystemCollector) observeLastGCTime(_ context.Context, observer metric.Int64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.LastGC > 0 {
		lastGCTime := time.Since(time.Unix(0, int64(m.LastGC)))
		observer.Observe(int64(lastGCTime.Seconds()))
	}

	return nil
}

// observeGCCount observes GC count
func (s *SystemCollector) observeGCCount(_ context.Context, observer metric.Int64Observer) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	observer.Observe(int64(m.NumGC))
	return nil
}
