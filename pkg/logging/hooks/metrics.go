package hooks

import (
	"log/slog"
	"sync"
	"time"
)

// MetricsHook emits metrics based on log data
type MetricsHook struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64

	// Metrics emission function
	emitFunc MetricsEmitter

	// Configuration
	emitInterval time.Duration
	stopCh       chan struct{}
}

// MetricsEmitter defines the interface for emitting metrics
type MetricsEmitter func(name string, value float64, tags map[string]string)

// NewMetricsHook creates a new metrics emission hook
func NewMetricsHook(emitter MetricsEmitter, emitInterval time.Duration) *MetricsHook {
	h := &MetricsHook{
		counters:     make(map[string]int64),
		gauges:       make(map[string]float64),
		emitFunc:     emitter,
		emitInterval: emitInterval,
		stopCh:       make(chan struct{}),
	}

	// Start emission goroutine
	go h.emitLoop()

	return h
}

// Process implements the Hook interface
func (h *MetricsHook) Process(record *slog.Record) *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Count logs by level
	levelKey := "logs.count." + record.Level.String()
	h.counters[levelKey]++

	// Extract metrics from attributes
	record.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "duration", "latency":
			if d, ok := a.Value.Any().(time.Duration); ok {
				h.gauges["latency."+record.Message] = d.Seconds()
			}
		case "queue_size":
			if size, ok := getNumericValue(a.Value); ok {
				h.gauges["queue.size"] = size
			}
		case "tokens_used":
			if tokens, ok := getNumericValue(a.Value); ok {
				h.counters["tokens.used"] += int64(tokens)
			}
		case "cost":
			if cost, ok := getNumericValue(a.Value); ok {
				h.gauges["cost.total"] += cost
			}
		case "cache_hit":
			if hit, ok := a.Value.Any().(bool); ok {
				if hit {
					h.counters["cache.hits"]++
				} else {
					h.counters["cache.misses"]++
				}
			}
		case "error":
			if a.Value.Any() != nil {
				h.counters["errors.total"]++
			}
		}
		return true
	})

	return record
}

// Stop stops the metrics emission loop
func (h *MetricsHook) Stop() {
	close(h.stopCh)
}

// emitLoop periodically emits accumulated metrics
func (h *MetricsHook) emitLoop() {
	ticker := time.NewTicker(h.emitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.emit()
		case <-h.stopCh:
			return
		}
	}
}

// emit sends all accumulated metrics
func (h *MetricsHook) emit() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.emitFunc == nil {
		return
	}

	// Emit counters
	for name, value := range h.counters {
		h.emitFunc(name, float64(value), nil)
	}

	// Emit gauges
	for name, value := range h.gauges {
		h.emitFunc(name, value, nil)
	}

	// Calculate and emit derived metrics
	if hits, hasHits := h.counters["cache.hits"]; hasHits {
		if misses, hasMisses := h.counters["cache.misses"]; hasMisses {
			total := float64(hits + misses)
			if total > 0 {
				hitRate := float64(hits) / total
				h.emitFunc("cache.hit_rate", hitRate, nil)
			}
		}
	}

	// Reset counters (gauges persist)
	h.counters = make(map[string]int64)
}

// GetMetrics returns current metric values
func (h *MetricsHook) GetMetrics() (counters map[string]int64, gauges map[string]float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Deep copy counters
	counters = make(map[string]int64)
	for k, v := range h.counters {
		counters[k] = v
	}

	// Deep copy gauges
	gauges = make(map[string]float64)
	for k, v := range h.gauges {
		gauges[k] = v
	}

	return counters, gauges
}

// getNumericValue extracts a numeric value from a slog.Value
func getNumericValue(v slog.Value) (float64, bool) {
	switch v.Kind() {
	case slog.KindInt64:
		return float64(v.Int64()), true
	case slog.KindFloat64:
		return v.Float64(), true
	default:
		// Try to convert from Any
		switch val := v.Any().(type) {
		case int:
			return float64(val), true
		case int32:
			return float64(val), true
		case int64:
			return float64(val), true
		case float32:
			return float64(val), true
		case float64:
			return val, true
		default:
			return 0, false
		}
	}
}
