// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// RenderCache provides efficient caching of rendered card strings
type RenderCache struct {
	mu      sync.RWMutex
	cards   map[string]*CachedRender
	maxAge  time.Duration
	maxSize int
}

// CachedRender contains a cached render result with metadata
type CachedRender struct {
	Content    string
	Timestamp  time.Time
	Hash       uint64 // Content hash for invalidation
	CardID     string
	Status     kanban.TaskStatus
	Width      int       // Rendered width
	Selected   bool      // Whether this was rendered as selected
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
	mu          sync.Mutex
	updates     chan CardUpdate
	batch       []CardUpdate
	ticker      *time.Ticker
	batchSize   int
	interval    time.Duration
	enabled     bool
	lastBatchID int
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

// VirtualNode represents a virtual DOM node for kanban rendering
type VirtualNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "board", "column", "card", "text"
	Content     string                 `json:"content,omitempty"`
	Props       map[string]interface{} `json:"props,omitempty"`
	Children    []*VirtualNode         `json:"children,omitempty"`
	Position    CardPosition           `json:"position"`
	Checksum    uint64                 `json:"checksum"`
	LastUpdated time.Time              `json:"last_updated"`
}

// VirtualDOM manages the virtual representation of the kanban board
type VirtualDOM struct {
	mu          sync.RWMutex
	root        *VirtualNode
	prevRoot    *VirtualNode
	diffPatches []DiffPatch
	nodePool    *sync.Pool
}

// DiffPatch represents a change operation in the virtual DOM
type DiffPatch struct {
	Type     DiffType      `json:"type"`
	Path     []int         `json:"path"`
	Node     *VirtualNode  `json:"node,omitempty"`
	OldNode  *VirtualNode  `json:"old_node,omitempty"`
	Position *CardPosition `json:"position,omitempty"`
}

// DiffType defines the type of diff operation
type DiffType int

const (
	DiffReplace DiffType = iota
	DiffUpdate
	DiffInsert
	DiffRemove
	DiffMove
)

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
	cache      *RenderCache
	batcher    *RenderBatcher
	metrics    *RenderMetrics
	mu         sync.RWMutex
	lowQuality bool // Reduced quality mode for performance
}

// NewOptimizedCardRenderer creates a new optimized renderer
func NewOptimizedCardRenderer() *OptimizedCardRenderer {
	cache := NewRenderCache(500, 5*time.Minute)          // Cache 500 cards for 5 minutes
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
		"cache_hit_rate":      cacheHitRate,
		"total_renders":       metrics.TotalRenders,
		"avg_render_time_ms":  float64(metrics.AverageRenderTime.Nanoseconds()) / 1e6,
		"low_quality_mode":    ocr.lowQuality,
		"cache_entries":       cacheStats["total_entries"],
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

// NewVirtualDOM creates a new virtual DOM for kanban rendering
func NewVirtualDOM() *VirtualDOM {
	return &VirtualDOM{
		nodePool: &sync.Pool{
			New: func() interface{} {
				return &VirtualNode{
					Props:    make(map[string]interface{}),
					Children: make([]*VirtualNode, 0),
				}
			},
		},
		diffPatches: make([]DiffPatch, 0),
	}
}

// BuildTree constructs a virtual DOM tree from kanban board state
func (vdom *VirtualDOM) BuildTree(ctx context.Context, columns []Column, selectedCard string) (*VirtualNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("kanban.render").
			WithOperation("BuildTree")
	}

	vdom.mu.Lock()
	defer vdom.mu.Unlock()

	// Create root node for board
	root := vdom.getNodeFromPool()
	root.ID = "kanban-board"
	root.Type = "board"
	root.LastUpdated = time.Now()

	// Build column nodes
	for colIdx, column := range columns {
		colNode := vdom.getNodeFromPool()
		colNode.ID = fmt.Sprintf("column-%d", colIdx)
		colNode.Type = "column"
		colNode.Props["title"] = column.Title
		colNode.Props["status"] = column.Status
		colNode.Position = CardPosition{ColumnIdx: colIdx}

		// Build card nodes
		for cardIdx, card := range column.Tasks {
			cardNode := vdom.getNodeFromPool()
			cardNode.ID = card.ID
			cardNode.Type = "card"
			cardNode.Content = card.Title
			cardNode.Props["status"] = card.Status
			cardNode.Props["priority"] = card.Priority
			cardNode.Props["selected"] = card.ID == selectedCard
			cardNode.Position = CardPosition{
				ColumnIdx: colIdx,
				CardIdx:   cardIdx,
			}
			cardNode.Checksum = vdom.calculateChecksum(card)
			cardNode.LastUpdated = time.Now()

			colNode.Children = append(colNode.Children, cardNode)
		}

		root.Children = append(root.Children, colNode)
	}

	root.Checksum = vdom.calculateTreeChecksum(root)
	return root, nil
}

// Diff calculates differences between previous and current virtual DOM trees
func (vdom *VirtualDOM) Diff(ctx context.Context, newRoot *VirtualNode) ([]DiffPatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("kanban.render").
			WithOperation("Diff")
	}

	vdom.mu.Lock()
	defer vdom.mu.Unlock()

	// Clear previous patches
	vdom.diffPatches = vdom.diffPatches[:0]

	if vdom.prevRoot == nil {
		// First render - everything is new
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type: DiffInsert,
			Node: newRoot,
			Path: []int{},
		})
	} else {
		// Calculate differences
		vdom.diffNodes(vdom.prevRoot, newRoot, []int{})
	}

	// Update references
	vdom.prevRoot = vdom.root
	vdom.root = newRoot

	return vdom.diffPatches, nil
}

// diffNodes recursively calculates differences between nodes
func (vdom *VirtualDOM) diffNodes(oldNode, newNode *VirtualNode, path []int) {
	// Both nil - no change
	if oldNode == nil && newNode == nil {
		return
	}

	// Node removed
	if oldNode != nil && newNode == nil {
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type:    DiffRemove,
			OldNode: oldNode,
			Path:    append([]int{}, path...),
		})
		return
	}

	// Node added
	if oldNode == nil && newNode != nil {
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type: DiffInsert,
			Node: newNode,
			Path: append([]int{}, path...),
		})
		return
	}

	// Type changed - replace entire subtree
	if oldNode.Type != newNode.Type {
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type:    DiffReplace,
			OldNode: oldNode,
			Node:    newNode,
			Path:    append([]int{}, path...),
		})
		return
	}

	// Check for content/property changes
	if oldNode.Checksum != newNode.Checksum {
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type:    DiffUpdate,
			OldNode: oldNode,
			Node:    newNode,
			Path:    append([]int{}, path...),
		})
	}

	// Check for position changes (card moved)
	if oldNode.Position != newNode.Position {
		vdom.diffPatches = append(vdom.diffPatches, DiffPatch{
			Type:     DiffMove,
			OldNode:  oldNode,
			Node:     newNode,
			Path:     append([]int{}, path...),
			Position: &newNode.Position,
		})
	}

	// Diff children
	vdom.diffChildren(oldNode.Children, newNode.Children, path)
}

// diffChildren handles child node diffing with optimization
func (vdom *VirtualDOM) diffChildren(oldChildren, newChildren []*VirtualNode, parentPath []int) {
	// Build ID maps for efficient lookup
	oldMap := make(map[string]int)
	for i, child := range oldChildren {
		if child != nil {
			oldMap[child.ID] = i
		}
	}

	newMap := make(map[string]int)
	for i, child := range newChildren {
		if child != nil {
			newMap[child.ID] = i
		}
	}

	// Process new children
	for newIdx, newChild := range newChildren {
		if newChild == nil {
			continue
		}

		childPath := append(parentPath, newIdx)

		if oldIdx, exists := oldMap[newChild.ID]; exists {
			// Child exists in both - check for updates
			vdom.diffNodes(oldChildren[oldIdx], newChild, childPath)
		} else {
			// New child
			vdom.diffNodes(nil, newChild, childPath)
		}
	}

	// Find removed children
	for oldIdx, oldChild := range oldChildren {
		if oldChild == nil {
			continue
		}

		if _, exists := newMap[oldChild.ID]; !exists {
			childPath := append(parentPath, oldIdx)
			vdom.diffNodes(oldChild, nil, childPath)
		}
	}
}

// Helper methods

func (vdom *VirtualDOM) getNodeFromPool() *VirtualNode {
	node := vdom.nodePool.Get().(*VirtualNode)
	// Reset node state
	node.ID = ""
	node.Type = ""
	node.Content = ""
	node.Children = node.Children[:0]
	node.Checksum = 0
	for k := range node.Props {
		delete(node.Props, k)
	}
	return node
}

func (vdom *VirtualDOM) returnNodeToPool(node *VirtualNode) {
	if node != nil {
		// Return children first
		for _, child := range node.Children {
			vdom.returnNodeToPool(child)
		}
		vdom.nodePool.Put(node)
	}
}

func (vdom *VirtualDOM) calculateChecksum(card *kanban.Task) uint64 {
	// Simple checksum based on card content
	var sum uint64
	for _, r := range card.Title {
		sum = sum*31 + uint64(r)
	}
	// Use string hash for status
	for _, r := range string(card.Status) {
		sum = sum*17 + uint64(r)
	}
	// Use string hash for priority
	for _, r := range string(card.Priority) {
		sum = sum*13 + uint64(r)
	}
	return sum
}

func (vdom *VirtualDOM) calculateTreeChecksum(node *VirtualNode) uint64 {
	if node == nil {
		return 0
	}

	sum := node.Checksum
	for _, child := range node.Children {
		sum = sum*31 + vdom.calculateTreeChecksum(child)
	}
	return sum
}

// ApplyPatches applies diff patches to optimize rendering
func (vdom *VirtualDOM) ApplyPatches(ctx context.Context, patches []DiffPatch, renderer func(patch DiffPatch) error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("kanban.render").
			WithOperation("ApplyPatches")
	}

	// Group patches by type for optimized application
	grouped := make(map[DiffType][]DiffPatch)
	for _, patch := range patches {
		grouped[patch.Type] = append(grouped[patch.Type], patch)
	}

	// Apply in optimal order: removes first, then moves, updates, and inserts
	order := []DiffType{DiffRemove, DiffMove, DiffUpdate, DiffReplace, DiffInsert}

	for _, patchType := range order {
		if patches, exists := grouped[patchType]; exists {
			for _, patch := range patches {
				if err := renderer(patch); err != nil {
					return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to apply patch").
						WithComponent("kanban.render").
						WithOperation("ApplyPatches")
				}
			}
		}
	}

	return nil
}

// GetStats returns virtual DOM statistics
func (vdom *VirtualDOM) GetStats() map[string]interface{} {
	vdom.mu.RLock()
	defer vdom.mu.RUnlock()

	stats := map[string]interface{}{
		"patches_count": len(vdom.diffPatches),
		"has_root":      vdom.root != nil,
		"has_prev_root": vdom.prevRoot != nil,
	}

	if vdom.root != nil {
		stats["total_nodes"] = vdom.countNodes(vdom.root)
	}

	return stats
}

// countNodes recursively counts nodes in the tree
func (vdom *VirtualDOM) countNodes(node *VirtualNode) int {
	if node == nil {
		return 0
	}

	count := 1
	for _, child := range node.Children {
		count += vdom.countNodes(child)
	}
	return count
}

// Cleanup releases resources and returns nodes to pool
func (vdom *VirtualDOM) Cleanup() {
	vdom.mu.Lock()
	defer vdom.mu.Unlock()

	if vdom.prevRoot != nil {
		vdom.returnNodeToPool(vdom.prevRoot)
		vdom.prevRoot = nil
	}

	if vdom.root != nil {
		vdom.returnNodeToPool(vdom.root)
		vdom.root = nil
	}

	vdom.diffPatches = vdom.diffPatches[:0]
}
