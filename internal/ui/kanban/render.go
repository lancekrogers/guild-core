// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
)

// RenderCache provides efficient caching of rendered card strings
type RenderCache struct {
	mu     sync.RWMutex
	cards  map[string]*CachedRender
	maxAge time.Duration
	maxSize int
}

// CachedRender contains a cached render result with metadata
type CachedRender struct {
	Content   string
	Timestamp time.Time
	Hash      uint64    // Content hash for invalidation
	CardID    string
	Status    kanban.TaskStatus
	Width     int       // Rendered width
	Selected  bool      // Whether this was rendered as selected
	AccessTime time.Time // For LRU eviction
}

// CardUpdate represents a change to a card that may require re-rendering
type CardUpdate struct {
	CardID   string
	Card     *kanban.Task
	Type     UpdateType
	Position *CardPosition
}

// UpdateType indicates the type of card update
type UpdateType int

const (
	UpdateTypeCreate UpdateType = iota
	UpdateTypeModify
	UpdateTypeMove
	UpdateTypeDelete
	UpdateTypeSelect
)

// CardPosition represents the position of a card in the layout
type CardPosition struct {
	ColumnIdx int
	CardIdx   int
	X         int
	Y         int
}

// BatchUpdateMsg is a Bubble Tea message for batched render updates
type BatchUpdateMsg struct {
	Updates []CardUpdate
	BatchID string
}

// RenderBatcher manages batched render updates for performance
type RenderBatcher struct {
	mu           sync.Mutex
	updates      chan CardUpdate
	batch        []CardUpdate
	ticker       *time.Ticker
	batchSize    int
	interval     time.Duration
	enabled      bool
	lastBatchID  int
}

// RenderMetrics tracks rendering performance
type RenderMetrics struct {
	CacheHits         int64
	CacheMisses       int64
	TotalRenders      int64
	BatchedUpdates    int64
	AverageRenderTime time.Duration
	LastRenderTime    time.Time
}

// NewRenderCache creates a new render cache
func NewRenderCache(maxSize int, maxAge time.Duration) *RenderCache {
	return &RenderCache{
		cards:   make(map[string]*CachedRender),
		maxAge:  maxAge,
		maxSize: maxSize,
	}
}

// GetOrRender retrieves cached content or renders it if not cached
func (rc *RenderCache) GetOrRender(ctx context.Context, card *kanban.Task, width int, selected bool, renderer func(*kanban.Task, int, bool) string) (string, error) {
	if card == nil {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "card cannot be nil", nil).
			WithComponent("kanban.render").
			WithOperation("GetOrRender")
	}

	// Create cache key including rendering parameters
	cacheKey := fmt.Sprintf("%s:%d:%t", card.ID, width, selected)
	
	// Calculate content hash for cache invalidation
	contentHash := rc.calculateHash(card)

	rc.mu.RLock()
	if cached, exists := rc.cards[cacheKey]; exists {
		// Check if cache entry is still valid
		if time.Since(cached.Timestamp) < rc.maxAge && cached.Hash == contentHash {
			// Update access time for LRU
			cached.AccessTime = time.Now()
			rc.mu.RUnlock()
			return cached.Content, nil
		}
	}
	rc.mu.RUnlock()

	// Cache miss or invalid - render new content
	rendered := renderer(card, width, selected)

	// Store in cache
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Evict old entries if cache is full
	if len(rc.cards) >= rc.maxSize {
		rc.evictOldestEntry()
	}

	rc.cards[cacheKey] = &CachedRender{
		Content:    rendered,
		Timestamp:  time.Now(),
		Hash:       contentHash,
		CardID:     card.ID,
		Status:     card.Status,
		Width:      width,
		Selected:   selected,
		AccessTime: time.Now(),
	}

	return rendered, nil
}

// InvalidateCard removes all cached renders for a specific card
func (rc *RenderCache) InvalidateCard(ctx context.Context, cardID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	keysToDelete := make([]string, 0)
	for key, cached := range rc.cards {
		if cached.CardID == cardID {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(rc.cards, key)
	}

	return nil
}

// Clear removes all cached entries
func (rc *RenderCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.cards = make(map[string]*CachedRender)
}

// GetStats returns cache performance statistics
func (rc *RenderCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	totalEntries := len(rc.cards)
	validEntries := 0
	now := time.Now()
	
	for _, cached := range rc.cards {
		if now.Sub(cached.Timestamp) < rc.maxAge {
			validEntries++
		}
	}

	return map[string]interface{}{
		"total_entries": totalEntries,
		"valid_entries": validEntries,
		"max_size":      rc.maxSize,
		"max_age_ms":    rc.maxAge.Milliseconds(),
	}
}

// calculateHash computes a hash of card content for cache invalidation
func (rc *RenderCache) calculateHash(card *kanban.Task) uint64 {
	// Simple hash based on card fields that affect rendering
	content := fmt.Sprintf("%s:%s:%s:%s:%s:%d:%d",
		card.ID, card.Title, card.Description, string(card.Priority),
		card.AssignedTo, card.Progress, card.UpdatedAt.Unix())
	
	// Basic FNV-1a hash
	const fnvPrime = 1099511628211
	const fnvOffset = 14695981039346656037
	
	hash := uint64(fnvOffset)
	for _, b := range []byte(content) {
		hash ^= uint64(b)
		hash *= fnvPrime
	}
	
	return hash
}

// evictOldestEntry removes the least recently used cache entry
func (rc *RenderCache) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, cached := range rc.cards {
		if oldestKey == "" || cached.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.AccessTime
		}
	}
	
	if oldestKey != "" {
		delete(rc.cards, oldestKey)
	}
}

// NewRenderBatcher creates a new render batcher
func NewRenderBatcher(batchSize int, interval time.Duration) *RenderBatcher {
	return &RenderBatcher{
		updates:   make(chan CardUpdate, batchSize*2), // Buffer for 2x batch size
		batch:     make([]CardUpdate, 0, batchSize),
		batchSize: batchSize,
		interval:  interval,
		enabled:   true,
	}
}

// Start begins the render batching process
func (rb *RenderBatcher) Start(ctx context.Context) tea.Cmd {
	if !rb.enabled {
		return nil
	}

	rb.mu.Lock()
	rb.ticker = time.NewTicker(rb.interval)
	rb.mu.Unlock()

	return func() tea.Msg {
		for {
			select {
			case <-ctx.Done():
				return nil

			case update := <-rb.updates:
				rb.mu.Lock()
				rb.batch = append(rb.batch, update)
				
				// Send batch if it's full
				if len(rb.batch) >= rb.batchSize {
					batch := rb.createBatchMessage()
					rb.batch = rb.batch[:0] // Reset batch
					rb.mu.Unlock()
					return batch
				}
				rb.mu.Unlock()

			case <-rb.ticker.C:
				rb.mu.Lock()
				if len(rb.batch) > 0 {
					batch := rb.createBatchMessage()
					rb.batch = rb.batch[:0] // Reset batch
					rb.mu.Unlock()
					return batch
				}
				rb.mu.Unlock()
			}
		}
	}
}

// QueueUpdate adds an update to the batch queue
func (rb *RenderBatcher) QueueUpdate(ctx context.Context, update CardUpdate) error {
	if !rb.enabled {
		return nil
	}

	select {
	case rb.updates <- update:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Channel is full, this indicates performance issues
		return gerror.New(gerror.ErrCodeConnection, "render update queue full", nil).
			WithComponent("kanban.render").
			WithOperation("QueueUpdate")
	}
}

// Stop terminates the render batcher
func (rb *RenderBatcher) Stop() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	rb.enabled = false
	if rb.ticker != nil {
		rb.ticker.Stop()
	}
	close(rb.updates)
}

// createBatchMessage creates a batch update message
func (rb *RenderBatcher) createBatchMessage() BatchUpdateMsg {
	rb.lastBatchID++
	batchCopy := make([]CardUpdate, len(rb.batch))
	copy(batchCopy, rb.batch)
	
	return BatchUpdateMsg{
		Updates: batchCopy,
		BatchID: fmt.Sprintf("batch_%d", rb.lastBatchID),
	}
}

// OptimizedCardRenderer provides high-performance card rendering
type OptimizedCardRenderer struct {
	cache        *RenderCache
	batcher      *RenderBatcher
	metrics      *RenderMetrics
	mu           sync.RWMutex
	lowQuality   bool // Reduced quality mode for performance
}

// NewOptimizedCardRenderer creates a new optimized renderer
func NewOptimizedCardRenderer() *OptimizedCardRenderer {
	cache := NewRenderCache(500, 5*time.Minute) // Cache 500 cards for 5 minutes
	batcher := NewRenderBatcher(20, 16*time.Millisecond) // 60 FPS batching
	
	return &OptimizedCardRenderer{
		cache:   cache,
		batcher: batcher,
		metrics: &RenderMetrics{},
	}
}

// RenderCard renders a single card with caching
func (ocr *OptimizedCardRenderer) RenderCard(ctx context.Context, card *kanban.Task, width int, selected bool) (string, error) {
	start := time.Now()
	defer func() {
		ocr.mu.Lock()
		ocr.metrics.TotalRenders++
		ocr.metrics.AverageRenderTime = time.Since(start)
		ocr.metrics.LastRenderTime = time.Now()
		ocr.mu.Unlock()
	}()

	rendered, err := ocr.cache.GetOrRender(ctx, card, width, selected, ocr.renderCardContent)
	if err != nil {
		ocr.mu.Lock()
		ocr.metrics.CacheMisses++
		ocr.mu.Unlock()
		return "", err
	}

	// Check if this was a cache hit
	if time.Since(start) < time.Millisecond {
		ocr.mu.Lock()
		ocr.metrics.CacheHits++
		ocr.mu.Unlock()
	} else {
		ocr.mu.Lock()
		ocr.metrics.CacheMisses++
		ocr.mu.Unlock()
	}

	return rendered, nil
}

// renderCardContent performs the actual card rendering
func (ocr *OptimizedCardRenderer) renderCardContent(card *kanban.Task, width int, selected bool) string {
	// Priority indicator
	var priorityIcon string
	if !ocr.lowQuality {
		switch card.Priority {
		case kanban.PriorityHigh:
			priorityIcon = "🔴"
		case kanban.PriorityMedium:
			priorityIcon = "🟡"
		case kanban.PriorityLow:
			priorityIcon = "🟢"
		}
	} else {
		// Low quality mode - use simple characters
		switch card.Priority {
		case kanban.PriorityHigh:
			priorityIcon = "H"
		case kanban.PriorityMedium:
			priorityIcon = "M"
		case kanban.PriorityLow:
			priorityIcon = "L"
		}
	}

	// Progress indicator for in-progress tasks
	var progressBar string
	if card.Status == kanban.StatusInProgress && card.Progress > 0 && !ocr.lowQuality {
		barWidth := 8
		filled := int(float64(barWidth) * float64(card.Progress) / 100.0)
		progressBar = fmt.Sprintf(" [%s%s] %d%%",
			strings.Repeat("█", filled),
			strings.Repeat("░", barWidth-filled),
			card.Progress,
		)
	} else if card.Status == kanban.StatusInProgress && card.Progress > 0 {
		// Low quality progress
		progressBar = fmt.Sprintf(" %d%%", card.Progress)
	}

	// Task ID and title
	titleWidth := width - 4 // Account for padding and priority
	if progressBar != "" {
		titleWidth -= len(progressBar)
	}
	if titleWidth < 5 {
		titleWidth = 5 // Minimum width for readability
	}

	title := fmt.Sprintf("[%s]", card.ID)
	if len(title) > titleWidth && titleWidth > 3 {
		title = title[:titleWidth-3] + "..."
	}

	// Task description (second line)
	desc := card.Title
	descWidth := width - 4
	if descWidth > 7 && len(desc) > descWidth {
		desc = desc[:descWidth-7] + "..."
	}

	// Assignee (third line) - skip in low quality mode to save space
	assignee := ""
	if !ocr.lowQuality && card.AssignedTo != "" {
		assignee = "@" + card.AssignedTo
		if len(assignee) > width-4 {
			assignee = assignee[:width-7] + "..."
		}
	}

	// Build task content
	lines := []string{
		fmt.Sprintf("%s %s%s", priorityIcon, title, progressBar),
		"  " + desc,
	}
	if assignee != "" {
		lines = append(lines, "  "+assignee)
	}

	// Apply style
	var style lipgloss.Style
	if !ocr.lowQuality {
		style = taskStyle.Copy().Width(width).MaxHeight(3)
		if selected {
			style = selectedTaskStyle.Copy().Width(width).MaxHeight(3)
		}

		// Special styling for blocked tasks
		if card.Status == kanban.StatusBlocked {
			style = style.Foreground(lipgloss.Color("9"))
		}
	} else {
		// Low quality mode - minimal styling
		style = lipgloss.NewStyle().Width(width).MaxHeight(3)
		if selected {
			style = style.Bold(true)
		}
	}

	return style.Render(strings.Join(lines, "\n"))
}

// SetLowQualityMode enables/disables low quality rendering for performance
func (ocr *OptimizedCardRenderer) SetLowQualityMode(ctx context.Context, enabled bool) error {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()
	
	if ocr.lowQuality != enabled {
		ocr.lowQuality = enabled
		// Clear cache when quality mode changes
		ocr.cache.Clear()
	}
	
	return nil
}

// InvalidateCard removes cached renders for a specific card
func (ocr *OptimizedCardRenderer) InvalidateCard(ctx context.Context, cardID string) error {
	return ocr.cache.InvalidateCard(ctx, cardID)
}

// GetMetrics returns rendering performance metrics
func (ocr *OptimizedCardRenderer) GetMetrics() *RenderMetrics {
	ocr.mu.RLock()
	defer ocr.mu.RUnlock()
	
	// Return a copy to avoid data races
	return &RenderMetrics{
		CacheHits:         ocr.metrics.CacheHits,
		CacheMisses:       ocr.metrics.CacheMisses,
		TotalRenders:      ocr.metrics.TotalRenders,
		BatchedUpdates:    ocr.metrics.BatchedUpdates,
		AverageRenderTime: ocr.metrics.AverageRenderTime,
		LastRenderTime:    ocr.metrics.LastRenderTime,
	}
}

// GetDebugInfo returns debug information about the render system
func (ocr *OptimizedCardRenderer) GetDebugInfo(ctx context.Context) map[string]interface{} {
	metrics := ocr.GetMetrics()
	cacheStats := ocr.cache.GetStats()
	
	cacheHitRate := float64(0)
	if metrics.CacheHits+metrics.CacheMisses > 0 {
		cacheHitRate = float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
	}

	info := map[string]interface{}{
		"cache_hit_rate":     cacheHitRate,
		"total_renders":      metrics.TotalRenders,
		"avg_render_time_ms": float64(metrics.AverageRenderTime.Nanoseconds()) / 1e6,
		"low_quality_mode":   ocr.lowQuality,
		"cache_entries":      cacheStats["total_entries"],
		"valid_cache_entries": cacheStats["valid_entries"],
	}

	return info
}

// Cleanup performs cleanup operations
func (ocr *OptimizedCardRenderer) Cleanup(ctx context.Context) error {
	if ocr.batcher != nil {
		ocr.batcher.Stop()
	}
	
	if ocr.cache != nil {
		ocr.cache.Clear()
	}
	
	return nil
}