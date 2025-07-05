// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cache

import (
	"sort"
	"sync"
	"time"
)

// Cache interface for eviction policies
type Cache interface {
	GetEntries() map[string]*CacheEntry
}

// EvictionPolicy defines the interface for cache eviction strategies
type EvictionPolicy interface {
	Evict(cache Cache, requiredSpace int64) []string
	GetName() string
}

// LRUPolicy implements Least Recently Used eviction
type LRUPolicy struct {
	name string
}

// NewLRUPolicy creates a new LRU eviction policy
func NewLRUPolicy() *LRUPolicy {
	return &LRUPolicy{
		name: "LRU",
	}
}

// Evict selects entries for eviction based on LRU strategy
func (lru *LRUPolicy) Evict(cache Cache, requiredSpace int64) []string {
	entries := cache.GetEntries()
	if len(entries) == 0 {
		return []string{}
	}

	// Sort entries by last access time (oldest first)
	type entryInfo struct {
		key        string
		lastAccess time.Time
		size       int64
	}

	var sortedEntries []entryInfo
	for key, entry := range entries {
		sortedEntries = append(sortedEntries, entryInfo{
			key:        key,
			lastAccess: entry.LastAccess,
			size:       entry.Size,
		})
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].lastAccess.Before(sortedEntries[j].lastAccess)
	})

	// Select entries to evict
	var candidates []string
	var freedSpace int64

	for _, entry := range sortedEntries {
		candidates = append(candidates, entry.key)
		freedSpace += entry.size

		if freedSpace >= requiredSpace {
			break
		}
	}

	return candidates
}

// GetName returns the name of the eviction policy
func (lru *LRUPolicy) GetName() string {
	return lru.name
}

// LFUPolicy implements Least Frequently Used eviction
type LFUPolicy struct {
	name string
}

// NewLFUPolicy creates a new LFU eviction policy
func NewLFUPolicy() *LFUPolicy {
	return &LFUPolicy{
		name: "LFU",
	}
}

// Evict selects entries for eviction based on LFU strategy
func (lfu *LFUPolicy) Evict(cache Cache, requiredSpace int64) []string {
	entries := cache.GetEntries()
	if len(entries) == 0 {
		return []string{}
	}

	// Sort entries by access count (least frequent first)
	type entryInfo struct {
		key         string
		accessCount int
		size        int64
	}

	var sortedEntries []entryInfo
	for key, entry := range entries {
		sortedEntries = append(sortedEntries, entryInfo{
			key:         key,
			accessCount: entry.AccessCount,
			size:        entry.Size,
		})
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].accessCount < sortedEntries[j].accessCount
	})

	// Select entries to evict
	var candidates []string
	var freedSpace int64

	for _, entry := range sortedEntries {
		candidates = append(candidates, entry.key)
		freedSpace += entry.size

		if freedSpace >= requiredSpace {
			break
		}
	}

	return candidates
}

// GetName returns the name of the eviction policy
func (lfu *LFUPolicy) GetName() string {
	return lfu.name
}

// AdaptiveEvictionPolicy combines multiple strategies based on workload
type AdaptiveEvictionPolicy struct {
	strategies []EvictionStrategy
	selector   *StrategySelector
	monitor    *PerformanceMonitor
	mu         sync.RWMutex
	name       string
}

// EvictionStrategy represents a single eviction strategy with performance metrics
type EvictionStrategy struct {
	Policy      EvictionPolicy
	Weight      float64
	Performance *StrategyPerformance
}

// StrategyPerformance tracks the performance of an eviction strategy
type StrategyPerformance struct {
	HitRateAfterEviction float64
	EvictionTime         time.Duration
	UsageCount           int64
	SuccessRate          float64
}

// NewAdaptiveEvictionPolicy creates a new adaptive eviction policy
func NewAdaptiveEvictionPolicy() *AdaptiveEvictionPolicy {
	aep := &AdaptiveEvictionPolicy{
		selector: NewStrategySelector(),
		monitor:  NewPerformanceMonitor(),
		name:     "Adaptive",
	}

	// Initialize with multiple strategies
	aep.strategies = []EvictionStrategy{
		{
			Policy: NewLRUPolicy(),
			Weight: 0.4,
			Performance: &StrategyPerformance{
				HitRateAfterEviction: 0.8,
				SuccessRate:          0.9,
			},
		},
		{
			Policy: NewLFUPolicy(),
			Weight: 0.3,
			Performance: &StrategyPerformance{
				HitRateAfterEviction: 0.75,
				SuccessRate:          0.85,
			},
		},
		{
			Policy: NewSizeBasedPolicy(),
			Weight: 0.2,
			Performance: &StrategyPerformance{
				HitRateAfterEviction: 0.7,
				SuccessRate:          0.8,
			},
		},
		{
			Policy: NewCostBasedPolicy(),
			Weight: 0.1,
			Performance: &StrategyPerformance{
				HitRateAfterEviction: 0.85,
				SuccessRate:          0.7,
			},
		},
	}

	return aep
}

// Evict selects the best strategy and performs eviction
func (aep *AdaptiveEvictionPolicy) Evict(cache Cache, requiredSpace int64) []string {
	aep.mu.Lock()
	defer aep.mu.Unlock()

	// Select best strategy based on current workload
	strategy := aep.selector.SelectStrategy(aep.strategies, aep.getWorkloadStats(cache))

	// Record eviction start time
	startTime := time.Now()

	// Get eviction candidates
	candidates := strategy.Policy.Evict(cache, requiredSpace)

	// Update strategy performance
	evictionTime := time.Since(startTime)
	aep.updateStrategyPerformance(strategy, evictionTime, len(candidates))

	// Monitor eviction impact
	aep.monitor.RecordEviction(strategy.Policy.GetName(), candidates)

	return candidates
}

// GetName returns the name of the eviction policy
func (aep *AdaptiveEvictionPolicy) GetName() string {
	return aep.name
}

// getWorkloadStats analyzes current cache workload
func (aep *AdaptiveEvictionPolicy) getWorkloadStats(cache Cache) *WorkloadStats {
	entries := cache.GetEntries()
	
	stats := &WorkloadStats{
		EntryCount: len(entries),
		TotalSize:  0,
		AvgAccessCount: 0,
		RecentAccesses: 0,
	}

	if len(entries) == 0 {
		return stats
	}

	recentThreshold := time.Now().Add(-time.Hour)
	totalAccesses := 0

	for _, entry := range entries {
		stats.TotalSize += entry.Size
		totalAccesses += entry.AccessCount

		if entry.LastAccess.After(recentThreshold) {
			stats.RecentAccesses++
		}
	}

	stats.AvgAccessCount = float64(totalAccesses) / float64(len(entries))
	
	return stats
}

// updateStrategyPerformance updates performance metrics for a strategy
func (aep *AdaptiveEvictionPolicy) updateStrategyPerformance(strategy *EvictionStrategy, evictionTime time.Duration, evictedCount int) {
	strategy.Performance.EvictionTime = evictionTime
	strategy.Performance.UsageCount++

	// Update success rate based on eviction efficiency
	if evictedCount > 0 {
		strategy.Performance.SuccessRate = (strategy.Performance.SuccessRate*0.9) + (0.1*1.0)
	} else {
		strategy.Performance.SuccessRate = (strategy.Performance.SuccessRate*0.9) + (0.1*0.0)
	}
}

// WorkloadStats represents current cache workload characteristics
type WorkloadStats struct {
	EntryCount     int     `json:"entry_count"`
	TotalSize      int64   `json:"total_size"`
	AvgAccessCount float64 `json:"avg_access_count"`
	RecentAccesses int     `json:"recent_accesses"`
	HitRate        float64 `json:"hit_rate"`
	MissRate       float64 `json:"miss_rate"`
}

// StrategySelector selects the best eviction strategy for current workload
type StrategySelector struct {
	mu sync.RWMutex
}

// NewStrategySelector creates a new strategy selector
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{}
}

// SelectStrategy selects the best strategy for the given workload
func (ss *StrategySelector) SelectStrategy(strategies []EvictionStrategy, stats *WorkloadStats) *EvictionStrategy {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Calculate scores for each strategy based on workload
	bestStrategy := &strategies[0]
	bestScore := ss.calculateScore(&strategies[0], stats)

	for i := 1; i < len(strategies); i++ {
		score := ss.calculateScore(&strategies[i], stats)
		if score > bestScore {
			bestScore = score
			bestStrategy = &strategies[i]
		}
	}

	return bestStrategy
}

// calculateScore calculates a score for a strategy given current workload
func (ss *StrategySelector) calculateScore(strategy *EvictionStrategy, stats *WorkloadStats) float64 {
	score := strategy.Weight * strategy.Performance.SuccessRate

	// Adjust score based on workload characteristics
	if stats.RecentAccesses > stats.EntryCount/2 {
		// High recent activity - favor LRU
		if strategy.Policy.GetName() == "LRU" {
			score *= 1.2
		}
	}

	if stats.AvgAccessCount > 10 {
		// High access frequency - favor LFU
		if strategy.Policy.GetName() == "LFU" {
			score *= 1.15
		}
	}

	if stats.TotalSize > 1024*1024*1024 { // > 1GB
		// Large cache size - favor size-based eviction
		if strategy.Policy.GetName() == "Size-Based" {
			score *= 1.1
		}
	}

	return score
}

// PerformanceMonitor monitors eviction performance
type PerformanceMonitor struct {
	evictions map[string][]EvictionEvent
	mu        sync.RWMutex
}

// EvictionEvent represents a single eviction event
type EvictionEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Strategy  string    `json:"strategy"`
	KeyCount  int       `json:"key_count"`
	Duration  time.Duration `json:"duration"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		evictions: make(map[string][]EvictionEvent),
	}
}

// RecordEviction records an eviction event
func (pm *PerformanceMonitor) RecordEviction(strategy string, keys []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	event := EvictionEvent{
		Timestamp: time.Now(),
		Strategy:  strategy,
		KeyCount:  len(keys),
	}

	pm.evictions[strategy] = append(pm.evictions[strategy], event)

	// Keep only recent events (last 1000)
	if len(pm.evictions[strategy]) > 1000 {
		pm.evictions[strategy] = pm.evictions[strategy][len(pm.evictions[strategy])-1000:]
	}
}

// GetEvictionStats returns eviction statistics
func (pm *PerformanceMonitor) GetEvictionStats() map[string]*EvictionStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]*EvictionStats)

	for strategy, events := range pm.evictions {
		if len(events) == 0 {
			continue
		}

		totalKeys := 0
		var totalDuration time.Duration

		for _, event := range events {
			totalKeys += event.KeyCount
			totalDuration += event.Duration
		}

		stats[strategy] = &EvictionStats{
			TotalEvictions: len(events),
			TotalKeys:      totalKeys,
			AvgKeysPerEviction: float64(totalKeys) / float64(len(events)),
			AvgDuration:    totalDuration / time.Duration(len(events)),
		}
	}

	return stats
}

// EvictionStats contains statistics about evictions
type EvictionStats struct {
	TotalEvictions     int           `json:"total_evictions"`
	TotalKeys          int           `json:"total_keys"`
	AvgKeysPerEviction float64       `json:"avg_keys_per_eviction"`
	AvgDuration        time.Duration `json:"avg_duration"`
}

// SizeBasedPolicy evicts largest entries first
type SizeBasedPolicy struct {
	name string
}

// NewSizeBasedPolicy creates a new size-based eviction policy
func NewSizeBasedPolicy() *SizeBasedPolicy {
	return &SizeBasedPolicy{
		name: "Size-Based",
	}
}

// Evict selects entries for eviction based on size
func (sbp *SizeBasedPolicy) Evict(cache Cache, requiredSpace int64) []string {
	entries := cache.GetEntries()
	if len(entries) == 0 {
		return []string{}
	}

	// Sort entries by size (largest first)
	type entryInfo struct {
		key  string
		size int64
	}

	var sortedEntries []entryInfo
	for key, entry := range entries {
		sortedEntries = append(sortedEntries, entryInfo{
			key:  key,
			size: entry.Size,
		})
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].size > sortedEntries[j].size
	})

	// Select entries to evict
	var candidates []string
	var freedSpace int64

	for _, entry := range sortedEntries {
		candidates = append(candidates, entry.key)
		freedSpace += entry.size

		if freedSpace >= requiredSpace {
			break
		}
	}

	return candidates
}

// GetName returns the name of the eviction policy
func (sbp *SizeBasedPolicy) GetName() string {
	return sbp.name
}

// CostBasedPolicy evicts entries based on cost-benefit analysis
type CostBasedPolicy struct {
	name string
}

// NewCostBasedPolicy creates a new cost-based eviction policy
func NewCostBasedPolicy() *CostBasedPolicy {
	return &CostBasedPolicy{
		name: "Cost-Based",
	}
}

// Evict selects entries for eviction based on cost-benefit ratio
func (cbp *CostBasedPolicy) Evict(cache Cache, requiredSpace int64) []string {
	entries := cache.GetEntries()
	if len(entries) == 0 {
		return []string{}
	}

	// Calculate cost-benefit ratio for each entry
	type entryInfo struct {
		key       string
		size      int64
		ratio     float64
	}

	var sortedEntries []entryInfo
	for key, entry := range entries {
		// Cost-benefit ratio: size / (access_count * recency_factor)
		recencyFactor := 1.0
		timeSinceAccess := time.Since(entry.LastAccess)
		if timeSinceAccess > time.Hour {
			recencyFactor = 0.5
		}

		accessWeight := float64(entry.AccessCount) * recencyFactor
		if accessWeight == 0 {
			accessWeight = 0.1 // Avoid division by zero
		}

		ratio := float64(entry.Size) / accessWeight

		sortedEntries = append(sortedEntries, entryInfo{
			key:   key,
			size:  entry.Size,
			ratio: ratio,
		})
	}

	// Sort by ratio (highest ratio = best candidates for eviction)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].ratio > sortedEntries[j].ratio
	})

	// Select entries to evict
	var candidates []string
	var freedSpace int64

	for _, entry := range sortedEntries {
		candidates = append(candidates, entry.key)
		freedSpace += entry.size

		if freedSpace >= requiredSpace {
			break
		}
	}

	return candidates
}

// GetName returns the name of the eviction policy
func (cbp *CostBasedPolicy) GetName() string {
	return cbp.name
}