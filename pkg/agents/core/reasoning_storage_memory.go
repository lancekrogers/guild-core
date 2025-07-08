// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// MemoryReasoningStorage provides in-memory storage for reasoning chains
// This is suitable for testing and development, not for production use
type MemoryReasoningStorage struct {
	mu       sync.RWMutex
	chains   map[string]*ReasoningChain
	patterns map[string]*ReasoningPattern
	config   ReasoningStorageConfig
}

// NewMemoryReasoningStorage creates a new in-memory reasoning storage
func NewMemoryReasoningStorage(config ReasoningStorageConfig) (*MemoryReasoningStorage, error) {
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid storage configuration").
			WithComponent("memory_reasoning_storage").
			WithOperation("NewMemoryReasoningStorage")
	}

	return &MemoryReasoningStorage{
		chains:   make(map[string]*ReasoningChain),
		patterns: make(map[string]*ReasoningPattern),
		config:   config,
	}, nil
}

// Store saves a reasoning chain
func (m *MemoryReasoningStorage) Store(ctx context.Context, chain *ReasoningChain) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("Store")
	}

	if chain == nil {
		return gerror.New(gerror.ErrCodeValidation, "chain cannot be nil", nil).
			WithComponent("memory_reasoning_storage").
			WithOperation("Store")
	}

	if chain.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "chain ID cannot be empty", nil).
			WithComponent("memory_reasoning_storage").
			WithOperation("Store")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clone the chain to avoid external modifications
	storedChain := *chain
	m.chains[chain.ID] = &storedChain

	// Clean up old chains if retention is set
	if m.config.RetentionDays > 0 {
		m.cleanupOldChainsLocked(ctx)
	}

	return nil
}

// Get retrieves a reasoning chain by ID
func (m *MemoryReasoningStorage) Get(ctx context.Context, id string) (*ReasoningChain, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("Get")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, exists := m.chains[id]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "reasoning chain not found: %s", id).
			WithComponent("memory_reasoning_storage").
			WithOperation("Get").
			WithDetails("chain_id", id)
	}

	// Return a copy to prevent external modifications
	chainCopy := *chain
	return &chainCopy, nil
}

// Query searches for reasoning chains based on criteria
func (m *MemoryReasoningStorage) Query(ctx context.Context, query *ReasoningQuery) ([]*ReasoningChain, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("Query")
	}

	if query == nil {
		query = &ReasoningQuery{
			Limit:   m.config.MaxChainsPerQuery,
			OrderBy: "created_at",
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter chains
	var filtered []*ReasoningChain
	for _, chain := range m.chains {
		if m.matchesQuery(chain, query) {
			// Create a copy
			chainCopy := *chain
			filtered = append(filtered, &chainCopy)
		}
	}

	// Sort results
	m.sortChains(filtered, query.OrderBy, query.Ascending)

	// Apply pagination
	start := query.Offset
	if start > len(filtered) {
		return []*ReasoningChain{}, nil
	}

	end := start + query.Limit
	if end > len(filtered) || query.Limit == 0 {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// GetStats returns aggregated statistics
func (m *MemoryReasoningStorage) GetStats(ctx context.Context, agentID string, startTime, endTime time.Time) (*ReasoningStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("GetStats")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ReasoningStats{
		ConfidenceDistrib: make(map[string]int),
		TaskTypeDistrib:   make(map[string]int),
		TimeDistrib:       make(map[string]int),
		Metadata:          make(map[string]interface{}),
	}

	var totalConfidence float64
	var totalDuration time.Duration
	var successCount int64

	for _, chain := range m.chains {
		// Filter by agent and time range
		if agentID != "" && chain.AgentID != agentID {
			continue
		}
		if !startTime.IsZero() && chain.CreatedAt.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && chain.CreatedAt.After(endTime) {
			continue
		}

		stats.TotalChains++
		totalConfidence += chain.Confidence
		totalDuration += chain.Duration

		if chain.Success {
			successCount++
		}

		// Update distributions
		confBucket := m.getConfidenceBucket(chain.Confidence)
		stats.ConfidenceDistrib[confBucket]++

		if chain.TaskType != "" {
			stats.TaskTypeDistrib[chain.TaskType]++
		}

		hourBucket := chain.CreatedAt.Format("2006-01-02 15:00")
		stats.TimeDistrib[hourBucket]++
	}

	// Calculate averages
	if stats.TotalChains > 0 {
		stats.AvgConfidence = totalConfidence / float64(stats.TotalChains)
		stats.AvgDuration = totalDuration / time.Duration(stats.TotalChains)
		stats.SuccessRate = float64(successCount) / float64(stats.TotalChains)
	}

	return stats, nil
}

// GetPatterns returns learned reasoning patterns
func (m *MemoryReasoningStorage) GetPatterns(ctx context.Context, taskType string, limit int) ([]*ReasoningPattern, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("GetPatterns")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var patterns []*ReasoningPattern
	for _, pattern := range m.patterns {
		if taskType == "" || pattern.TaskType == taskType {
			// Create a copy
			patternCopy := *pattern
			patterns = append(patterns, &patternCopy)
		}
	}

	// Sort by occurrences (most common first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Occurrences > patterns[j].Occurrences
	})

	// Apply limit
	if limit > 0 && len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// UpdatePattern updates or creates a reasoning pattern
func (m *MemoryReasoningStorage) UpdatePattern(ctx context.Context, pattern *ReasoningPattern) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("UpdatePattern")
	}

	if pattern == nil || pattern.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "pattern and pattern ID cannot be empty", nil).
			WithComponent("memory_reasoning_storage").
			WithOperation("UpdatePattern")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clone the pattern
	storedPattern := *pattern
	storedPattern.UpdatedAt = time.Now()
	m.patterns[pattern.ID] = &storedPattern

	return nil
}

// Delete removes old reasoning chains
func (m *MemoryReasoningStorage) Delete(ctx context.Context, beforeTime time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("memory_reasoning_storage").
			WithOperation("Delete")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for id, chain := range m.chains {
		if chain.CreatedAt.Before(beforeTime) {
			delete(m.chains, id)
			deleted++
		}
	}

	logger := observability.GetLogger(ctx).
		WithComponent("memory_reasoning_storage").
		WithOperation("Delete")

	logger.InfoContext(ctx, "Deleted old reasoning chains",
		"deleted_count", deleted,
		"before_time", beforeTime)

	return deleted, nil
}

// Close closes the storage connection
func (m *MemoryReasoningStorage) Close(ctx context.Context) error {
	// Nothing to close for in-memory storage
	return nil
}

// Helper methods

func (m *MemoryReasoningStorage) matchesQuery(chain *ReasoningChain, query *ReasoningQuery) bool {
	if query.AgentID != "" && chain.AgentID != query.AgentID {
		return false
	}
	if query.SessionID != "" && chain.SessionID != query.SessionID {
		return false
	}
	if query.TaskType != "" && chain.TaskType != query.TaskType {
		return false
	}
	if query.MinConfidence > 0 && chain.Confidence < query.MinConfidence {
		return false
	}
	if query.MaxConfidence > 0 && chain.Confidence > query.MaxConfidence {
		return false
	}
	if !query.StartTime.IsZero() && chain.CreatedAt.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && chain.CreatedAt.After(query.EndTime) {
		return false
	}
	return true
}

func (m *MemoryReasoningStorage) sortChains(chains []*ReasoningChain, orderBy string, ascending bool) {
	sort.Slice(chains, func(i, j int) bool {
		var less bool
		switch orderBy {
		case "confidence":
			less = chains[i].Confidence < chains[j].Confidence
		case "duration":
			less = chains[i].Duration < chains[j].Duration
		default: // created_at
			less = chains[i].CreatedAt.Before(chains[j].CreatedAt)
		}

		if ascending {
			return less
		}
		return !less
	})
}

func (m *MemoryReasoningStorage) getConfidenceBucket(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "very_high"
	case confidence >= 0.7:
		return "high"
	case confidence >= 0.5:
		return "medium"
	case confidence >= 0.3:
		return "low"
	default:
		return "very_low"
	}
}

func (m *MemoryReasoningStorage) cleanupOldChainsLocked(ctx context.Context) {
	if m.config.RetentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -m.config.RetentionDays)

	for id, chain := range m.chains {
		if chain.CreatedAt.Before(cutoffTime) {
			delete(m.chains, id)
		}
	}
}
