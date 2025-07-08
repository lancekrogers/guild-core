// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cache

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// CacheStats tracks comprehensive cache statistics
type CacheStats struct {
	l1Stats      *LevelMetrics
	l2Stats      *LevelMetrics
	distStats    *LevelMetrics
	overallStats *OverallMetrics
	startTime    time.Time
	mu           sync.RWMutex
}

// NewCacheStats creates a new cache statistics tracker
func NewCacheStats() *CacheStats {
	return &CacheStats{
		l1Stats:      NewLevelMetrics("L1"),
		l2Stats:      NewLevelMetrics("L2"),
		distStats:    NewLevelMetrics("Distributed"),
		overallStats: NewOverallMetrics(),
		startTime:    time.Now(),
	}
}

// LevelMetrics tracks metrics for a single cache level
type LevelMetrics struct {
	Name      string    `json:"name"`
	Hits      int64     `json:"hits"`
	Misses    int64     `json:"misses"`
	Sets      int64     `json:"sets"`
	Deletes   int64     `json:"deletes"`
	Evictions int64     `json:"evictions"`
	Errors    int64     `json:"errors"`
	LastReset time.Time `json:"last_reset"`
	mu        sync.RWMutex
}

// NewLevelMetrics creates new level metrics
func NewLevelMetrics(name string) *LevelMetrics {
	return &LevelMetrics{
		Name:      name,
		LastReset: time.Now(),
	}
}

// RecordHit records a cache hit
func (lm *LevelMetrics) RecordHit() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Hits++
}

// RecordMiss records a cache miss
func (lm *LevelMetrics) RecordMiss() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Misses++
}

// RecordSet records a cache set operation
func (lm *LevelMetrics) RecordSet() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Sets++
}

// RecordDelete records a cache delete operation
func (lm *LevelMetrics) RecordDelete() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Deletes++
}

// RecordEviction records a cache eviction
func (lm *LevelMetrics) RecordEviction() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Evictions++
}

// RecordError records a cache error
func (lm *LevelMetrics) RecordError() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.Errors++
}

// GetSnapshot returns a snapshot of current metrics
func (lm *LevelMetrics) GetSnapshot() *LevelMetrics {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return &LevelMetrics{
		Name:      lm.Name,
		Hits:      lm.Hits,
		Misses:    lm.Misses,
		Sets:      lm.Sets,
		Deletes:   lm.Deletes,
		Evictions: lm.Evictions,
		Errors:    lm.Errors,
		LastReset: lm.LastReset,
	}
}

// GetHitRate calculates the hit rate
func (lm *LevelMetrics) GetHitRate() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	total := lm.Hits + lm.Misses
	if total == 0 {
		return 0.0
	}
	return float64(lm.Hits) / float64(total)
}

// Reset resets all metrics
func (lm *LevelMetrics) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.Hits = 0
	lm.Misses = 0
	lm.Sets = 0
	lm.Deletes = 0
	lm.Evictions = 0
	lm.Errors = 0
	lm.LastReset = time.Now()
}

// OverallMetrics tracks overall cache system metrics
type OverallMetrics struct {
	TotalRequests     int64                `json:"total_requests"`
	TotalHits         int64                `json:"total_hits"`
	TotalMisses       int64                `json:"total_misses"`
	TotalErrors       int64                `json:"total_errors"`
	ResponseTimes     *ResponseTimeMetrics `json:"response_times"`
	ThroughputMetrics *ThroughputMetrics   `json:"throughput_metrics"`
	StartTime         time.Time            `json:"start_time"`
	mu                sync.RWMutex
}

// NewOverallMetrics creates new overall metrics
func NewOverallMetrics() *OverallMetrics {
	return &OverallMetrics{
		ResponseTimes:     NewResponseTimeMetrics(),
		ThroughputMetrics: NewThroughputMetrics(),
		StartTime:         time.Now(),
	}
}

// ResponseTimeMetrics tracks response time statistics
type ResponseTimeMetrics struct {
	samples    []time.Duration
	maxSamples int
	mu         sync.RWMutex
}

// NewResponseTimeMetrics creates new response time metrics
func NewResponseTimeMetrics() *ResponseTimeMetrics {
	return &ResponseTimeMetrics{
		samples:    make([]time.Duration, 0),
		maxSamples: 10000, // Keep last 10K samples
	}
}

// RecordResponseTime records a response time sample
func (rtm *ResponseTimeMetrics) RecordResponseTime(duration time.Duration) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()

	rtm.samples = append(rtm.samples, duration)

	// Keep only recent samples
	if len(rtm.samples) > rtm.maxSamples {
		rtm.samples = rtm.samples[len(rtm.samples)-rtm.maxSamples:]
	}
}

// GetPercentile calculates the specified percentile
func (rtm *ResponseTimeMetrics) GetPercentile(percentile float64) time.Duration {
	rtm.mu.RLock()
	defer rtm.mu.RUnlock()

	if len(rtm.samples) == 0 {
		return 0
	}

	// Make a copy and sort
	sorted := make([]time.Duration, len(rtm.samples))
	copy(sorted, rtm.samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// GetAverage calculates the average response time
func (rtm *ResponseTimeMetrics) GetAverage() time.Duration {
	rtm.mu.RLock()
	defer rtm.mu.RUnlock()

	if len(rtm.samples) == 0 {
		return 0
	}

	var total time.Duration
	for _, sample := range rtm.samples {
		total += sample
	}

	return total / time.Duration(len(rtm.samples))
}

// ThroughputMetrics tracks throughput statistics
type ThroughputMetrics struct {
	requests  []time.Time
	maxWindow time.Duration
	mu        sync.RWMutex
}

// NewThroughputMetrics creates new throughput metrics
func NewThroughputMetrics() *ThroughputMetrics {
	return &ThroughputMetrics{
		requests:  make([]time.Time, 0),
		maxWindow: time.Minute * 5, // Keep 5-minute window
	}
}

// RecordRequest records a request timestamp
func (tm *ThroughputMetrics) RecordRequest() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	tm.requests = append(tm.requests, now)

	// Remove old requests outside the window
	cutoff := now.Add(-tm.maxWindow)
	i := 0
	for i < len(tm.requests) && tm.requests[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		tm.requests = tm.requests[i:]
	}
}

// GetRequestsPerSecond calculates requests per second
func (tm *ThroughputMetrics) GetRequestsPerSecond() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if len(tm.requests) < 2 {
		return 0.0
	}

	// Calculate over the last minute
	now := time.Now()
	cutoff := now.Add(-time.Minute)

	count := 0
	for _, req := range tm.requests {
		if req.After(cutoff) {
			count++
		}
	}

	return float64(count) / 60.0 // requests per second
}

// RecordHit records a hit for the specified cache level
func (cs *CacheStats) RecordHit(level CacheLevel) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.overallStats.TotalHits++
	cs.overallStats.TotalRequests++
	cs.overallStats.ThroughputMetrics.RecordRequest()

	switch level {
	case CacheLevelL1:
		cs.l1Stats.RecordHit()
	case CacheLevelL2:
		cs.l2Stats.RecordHit()
	case CacheLevelDistributed:
		cs.distStats.RecordHit()
	}
}

// RecordMiss records a cache miss
func (cs *CacheStats) RecordMiss() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.overallStats.TotalMisses++
	cs.overallStats.TotalRequests++
	cs.overallStats.ThroughputMetrics.RecordRequest()
}

// RecordResponseTime records a response time
func (cs *CacheStats) RecordResponseTime(duration time.Duration) {
	cs.overallStats.ResponseTimes.RecordResponseTime(duration)
}

// GetOverallStats returns overall cache statistics
func (cs *CacheStats) GetOverallStats() *OverallStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	total := cs.overallStats.TotalHits + cs.overallStats.TotalMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(cs.overallStats.TotalHits) / float64(total)
	}

	return &OverallStats{
		TotalHits:      cs.overallStats.TotalHits,
		TotalMisses:    cs.overallStats.TotalMisses,
		OverallHitRate: hitRate,
		MemoryUsage:    0, // Would be calculated from cache sizes
		Uptime:         time.Since(cs.startTime),
	}
}

// GetDetailedStats returns comprehensive cache statistics
func (cs *CacheStats) GetDetailedStats() *DetailedCacheStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return &DetailedCacheStats{
		L1Metrics:          cs.l1Stats.GetSnapshot(),
		L2Metrics:          cs.l2Stats.GetSnapshot(),
		DistributedMetrics: cs.distStats.GetSnapshot(),
		OverallMetrics:     cs.overallStats,
		ResponseTimes: &ResponseTimeSummary{
			P50: cs.overallStats.ResponseTimes.GetPercentile(50),
			P95: cs.overallStats.ResponseTimes.GetPercentile(95),
			P99: cs.overallStats.ResponseTimes.GetPercentile(99),
			Avg: cs.overallStats.ResponseTimes.GetAverage(),
		},
		Throughput: cs.overallStats.ThroughputMetrics.GetRequestsPerSecond(),
		Timestamp:  time.Now(),
	}
}

// DetailedCacheStats contains comprehensive cache statistics
type DetailedCacheStats struct {
	L1Metrics          *LevelMetrics        `json:"l1_metrics"`
	L2Metrics          *LevelMetrics        `json:"l2_metrics"`
	DistributedMetrics *LevelMetrics        `json:"distributed_metrics"`
	OverallMetrics     *OverallMetrics      `json:"overall_metrics"`
	ResponseTimes      *ResponseTimeSummary `json:"response_times"`
	Throughput         float64              `json:"throughput"`
	Timestamp          time.Time            `json:"timestamp"`
}

// ResponseTimeSummary contains response time percentiles
type ResponseTimeSummary struct {
	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
	Avg time.Duration `json:"avg"`
}

// CacheWarmer provides intelligent cache warming capabilities
type CacheWarmer struct {
	cache     *IntelligentCache
	predictor *AccessPredictor
	scheduler *WarmingScheduler
	mu        sync.RWMutex
	config    *WarmingConfig
}

// WarmingConfig configures cache warming behavior
type WarmingConfig struct {
	Enabled             bool          `json:"enabled"`
	WarmingInterval     time.Duration `json:"warming_interval"`
	PredictionThreshold float64       `json:"prediction_threshold"`
	MaxWarmingKeys      int           `json:"max_warming_keys"`
	ConcurrentWarmers   int           `json:"concurrent_warmers"`
	WarmingTimeout      time.Duration `json:"warming_timeout"`
}

// DefaultWarmingConfig returns default warming configuration
func DefaultWarmingConfig() *WarmingConfig {
	return &WarmingConfig{
		Enabled:             true,
		WarmingInterval:     time.Minute * 15,
		PredictionThreshold: 0.7,
		MaxWarmingKeys:      100,
		ConcurrentWarmers:   3,
		WarmingTimeout:      time.Second * 30,
	}
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *IntelligentCache, predictor *AccessPredictor) *CacheWarmer {
	return &CacheWarmer{
		cache:     cache,
		predictor: predictor,
		scheduler: NewWarmingScheduler(),
		config:    DefaultWarmingConfig(),
	}
}

// WarmCache performs intelligent cache warming
func (cw *CacheWarmer) WarmCache(ctx context.Context) error {
	if !cw.config.Enabled {
		return nil
	}

	// Get predicted hot keys
	predictions := cw.predictor.GetHotKeyPredictions(cw.config.WarmingInterval)

	// Filter by prediction threshold
	var hotPredictions []*AccessPrediction
	for _, pred := range predictions {
		if pred.Probability >= cw.config.PredictionThreshold {
			hotPredictions = append(hotPredictions, pred)
		}
	}

	// Limit the number of keys to warm
	if len(hotPredictions) > cw.config.MaxWarmingKeys {
		hotPredictions = hotPredictions[:cw.config.MaxWarmingKeys]
	}

	// Warm cache with predicted data
	for _, pred := range hotPredictions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if key is already in cache
		if _, err := cw.cache.Get(ctx, pred.Key); err == nil {
			continue // Already cached
		}

		// Fetch and cache data (placeholder implementation)
		if data, err := cw.fetchData(ctx, pred.Key); err == nil {
			cw.cache.Set(ctx, pred.Key, data, WithTTL(pred.ExpectedDuration))
		}
	}

	return nil
}

// fetchData fetches data for cache warming (placeholder)
func (cw *CacheWarmer) fetchData(ctx context.Context, key string) (interface{}, error) {
	// This would typically fetch data from the original source
	// For now, return a placeholder
	return fmt.Sprintf("warmed-data-for-%s", key), nil
}

// StartWarming starts the cache warming scheduler
func (cw *CacheWarmer) StartWarming(ctx context.Context) {
	go cw.scheduler.Start(ctx, cw)
}

// WarmingScheduler schedules cache warming operations
type WarmingScheduler struct {
	ticker *time.Ticker
	mu     sync.RWMutex
}

// NewWarmingScheduler creates a new warming scheduler
func NewWarmingScheduler() *WarmingScheduler {
	return &WarmingScheduler{}
}

// Start starts the warming scheduler
func (ws *WarmingScheduler) Start(ctx context.Context, warmer *CacheWarmer) {
	ws.mu.Lock()
	ws.ticker = time.NewTicker(warmer.config.WarmingInterval)
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		if ws.ticker != nil {
			ws.ticker.Stop()
		}
		ws.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.ticker.C:
			// Create a timeout context for warming
			warmCtx, cancel := context.WithTimeout(ctx, warmer.config.WarmingTimeout)
			if err := warmer.WarmCache(warmCtx); err != nil {
				// Log error but continue
				fmt.Printf("Cache warming failed: %v\n", err)
			}
			cancel()
		}
	}
}

// Stop stops the warming scheduler
func (ws *WarmingScheduler) Stop() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.ticker != nil {
		ws.ticker.Stop()
		ws.ticker = nil
	}
}
