// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// Object pools for frequently allocated structs to reduce GC pressure
var (
	cardPool = sync.Pool{
		New: func() interface{} {
			return &kanban.Task{}
		},
	}

	columnPool = sync.Pool{
		New: func() interface{} {
			return &Column{}
		},
	}

	renderInfoPool = sync.Pool{
		New: func() interface{} {
			return &CardRenderInfo{}
		},
	}

	slicePool = sync.Pool{
		New: func() interface{} {
			return make([]*kanban.Task, 0, 50)
		},
	}
)

// GetCard retrieves a card from the object pool
func GetCard() *kanban.Task {
	return cardPool.Get().(*kanban.Task)
}

// PutCard returns a card to the object pool
func PutCard(card *kanban.Task) {
	if card != nil {
		// Reset card fields to prevent memory leaks
		*card = kanban.Task{}
		cardPool.Put(card)
	}
}

// GetColumn retrieves a column from the object pool
func GetColumn() *Column {
	return columnPool.Get().(*Column)
}

// PutColumn returns a column to the object pool
func PutColumn(col *Column) {
	if col != nil {
		// Reset column fields
		col.Status = ""
		col.Title = ""
		col.Tasks = nil
		col.ScrollOffset = 0
		col.TotalTasks = 0
		columnPool.Put(col)
	}
}

// GetRenderInfo retrieves a render info from the object pool
func GetRenderInfo() *CardRenderInfo {
	return renderInfoPool.Get().(*CardRenderInfo)
}

// PutRenderInfo returns a render info to the object pool
func PutRenderInfo(info *CardRenderInfo) {
	if info != nil {
		// Reset render info fields
		info.Card = nil
		info.X = 0
		info.Y = 0
		info.Column = ""
		info.ColumnIdx = 0
		info.CardIdx = 0
		info.Visible = false
		info.InBuffer = false
		renderInfoPool.Put(info)
	}
}

// GetTaskSlice retrieves a task slice from the object pool
func GetTaskSlice() []*kanban.Task {
	slice := slicePool.Get().([]*kanban.Task)
	return slice[:0] // Reset length but keep capacity
}

// PutTaskSlice returns a task slice to the object pool
func PutTaskSlice(slice []*kanban.Task) {
	if slice != nil {
		// Clear references to prevent memory leaks
		for i := range slice {
			slice[i] = nil
		}
		slice = slice[:0]
		slicePool.Put(slice)
	}
}

// VirtualWindow manages a sliding window of cards for memory efficiency
type VirtualWindow struct {
	mu          sync.RWMutex
	cards       []*kanban.Task
	startIndex  int
	endIndex    int
	totalCount  int
	windowSize  int
	centerIndex int
	lastAccess  time.Time
	isDirty     bool
}

// NewVirtualWindow creates a new virtual window
func NewVirtualWindow(windowSize int) *VirtualWindow {
	return &VirtualWindow{
		windowSize: windowSize,
		lastAccess: time.Now(),
	}
}

// LoadWindow loads a window of cards centered around the given index
func (vw *VirtualWindow) LoadWindow(ctx context.Context, allCards []*kanban.Task, center int) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	vw.totalCount = len(allCards)
	vw.centerIndex = center
	vw.lastAccess = time.Now()

	// Calculate window bounds
	halfWindow := vw.windowSize / 2
	vw.startIndex = center - halfWindow
	vw.endIndex = center + halfWindow

	// Clamp to valid bounds
	if vw.startIndex < 0 {
		vw.startIndex = 0
	}
	if vw.endIndex > len(allCards) {
		vw.endIndex = len(allCards)
	}

	// Adjust start if we're near the end
	if vw.endIndex-vw.startIndex < vw.windowSize && vw.startIndex > 0 {
		vw.startIndex = vw.endIndex - vw.windowSize
		if vw.startIndex < 0 {
			vw.startIndex = 0
		}
	}

	windowLen := vw.endIndex - vw.startIndex

	// Reuse existing slice if possible to reduce allocations
	if cap(vw.cards) >= windowLen {
		vw.cards = vw.cards[:windowLen]
	} else {
		vw.cards = make([]*kanban.Task, windowLen)
	}

	// Copy cards into window
	copy(vw.cards, allCards[vw.startIndex:vw.endIndex])
	vw.isDirty = false

	return nil
}

// GetCards returns the current window of cards
func (vw *VirtualWindow) GetCards(ctx context.Context) ([]*kanban.Task, error) {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	vw.lastAccess = time.Now()

	// Return a copy to prevent external modification
	result := make([]*kanban.Task, len(vw.cards))
	copy(result, vw.cards)

	return result, nil
}

// GetCard returns a specific card by its position in the window
func (vw *VirtualWindow) GetCard(ctx context.Context, index int) (*kanban.Task, error) {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	if index < vw.startIndex || index >= vw.endIndex {
		return nil, gerror.New(gerror.ErrCodeNotFound, "card index outside window", nil).
			WithComponent("kanban.data").
			WithOperation("GetCard").
			WithDetails("index", index).
			WithDetails("start", vw.startIndex).
			WithDetails("end", vw.endIndex)
	}

	windowIndex := index - vw.startIndex
	if windowIndex < 0 || windowIndex >= len(vw.cards) {
		return nil, gerror.New(gerror.ErrCodeNotFound, "invalid window index", nil).
			WithComponent("kanban.data").
			WithOperation("GetCard").
			WithDetails("window_index", windowIndex)
	}

	vw.lastAccess = time.Now()
	return vw.cards[windowIndex], nil
}

// Contains checks if the window contains a specific card index
func (vw *VirtualWindow) Contains(index int) bool {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	return index >= vw.startIndex && index < vw.endIndex
}

// GetBounds returns the start and end indices of the current window
func (vw *VirtualWindow) GetBounds() (start, end, total int) {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	return vw.startIndex, vw.endIndex, vw.totalCount
}

// ShouldReload determines if the window should be reloaded based on access patterns
func (vw *VirtualWindow) ShouldReload(ctx context.Context, requestedIndex int) bool {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	// Reload if requested index is outside current window
	if requestedIndex < vw.startIndex || requestedIndex >= vw.endIndex {
		return true
	}

	// Reload if we're close to the window edge (prefetch)
	bufferZone := vw.windowSize / 4
	if requestedIndex-vw.startIndex < bufferZone || vw.endIndex-requestedIndex < bufferZone {
		return true
	}

	// Reload if window is stale
	if vw.isDirty {
		return true
	}

	return false
}

// MarkDirty marks the window as needing reload
func (vw *VirtualWindow) MarkDirty() {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	vw.isDirty = true
}

// GetMemoryUsage estimates the memory usage of the virtual window
func (vw *VirtualWindow) GetMemoryUsage() int64 {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	// Rough estimation
	const taskSize = 200 // Average bytes per task
	return int64(len(vw.cards)) * taskSize
}

// CompactCardCache provides efficient storage for large numbers of cards
type CompactCardCache struct {
	mu          sync.RWMutex
	cards       map[string]*CompactCard
	byStatus    map[kanban.TaskStatus][]string // Card IDs by status
	maxAge      time.Duration
	maxSize     int
	lastCleanup time.Time
}

// CompactCard is a memory-efficient representation of a task
type CompactCard struct {
	ID         string
	Title      string
	Status     kanban.TaskStatus
	Priority   kanban.TaskPriority
	AssignedTo string
	Progress   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Hash       uint64 // For change detection
	LastAccess time.Time
}

// NewCompactCardCache creates a new compact card cache
func NewCompactCardCache(maxSize int, maxAge time.Duration) *CompactCardCache {
	return &CompactCardCache{
		cards:       make(map[string]*CompactCard),
		byStatus:    make(map[kanban.TaskStatus][]string),
		maxAge:      maxAge,
		maxSize:     maxSize,
		lastCleanup: time.Now(),
	}
}

// AddCard adds or updates a card in the cache
func (ccc *CompactCardCache) AddCard(ctx context.Context, task *kanban.Task) error {
	if task == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "task cannot be nil", nil).
			WithComponent("kanban.data").
			WithOperation("AddCard")
	}

	ccc.mu.Lock()
	defer ccc.mu.Unlock()

	// Check if we need to cleanup old entries
	if time.Since(ccc.lastCleanup) > time.Minute {
		ccc.cleanupExpiredEntries()
	}

	// Convert to compact representation
	compact := &CompactCard{
		ID:         task.ID,
		Title:      task.Title,
		Status:     task.Status,
		Priority:   task.Priority,
		AssignedTo: task.AssignedTo,
		Progress:   task.Progress,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		Hash:       ccc.calculateHash(task),
		LastAccess: time.Now(),
	}

	// Remove from old status if it exists
	if existing, exists := ccc.cards[task.ID]; exists {
		ccc.removeFromStatus(existing.Status, task.ID)
	}

	// Add to cache
	ccc.cards[task.ID] = compact

	// Add to status index
	if ccc.byStatus[task.Status] == nil {
		ccc.byStatus[task.Status] = make([]string, 0, 10)
	}
	ccc.byStatus[task.Status] = append(ccc.byStatus[task.Status], task.ID)

	// Evict oldest entries if cache is full
	if len(ccc.cards) > ccc.maxSize {
		ccc.evictOldestEntries(len(ccc.cards) - ccc.maxSize)
	}

	return nil
}

// GetCard retrieves a card from the cache
func (ccc *CompactCardCache) GetCard(ctx context.Context, cardID string) (*CompactCard, error) {
	ccc.mu.RLock()
	defer ccc.mu.RUnlock()

	card, exists := ccc.cards[cardID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "card not found", nil).
			WithComponent("kanban.data").
			WithOperation("GetCard").
			WithDetails("card_id", cardID)
	}

	// Update access time
	card.LastAccess = time.Now()
	return card, nil
}

// GetCardsByStatus returns card IDs for a specific status
func (ccc *CompactCardCache) GetCardsByStatus(ctx context.Context, status kanban.TaskStatus) ([]string, error) {
	ccc.mu.RLock()
	defer ccc.mu.RUnlock()

	cardIDs := ccc.byStatus[status]
	if cardIDs == nil {
		return []string{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]string, len(cardIDs))
	copy(result, cardIDs)
	return result, nil
}

// RemoveCard removes a card from the cache
func (ccc *CompactCardCache) RemoveCard(ctx context.Context, cardID string) error {
	ccc.mu.Lock()
	defer ccc.mu.Unlock()

	card, exists := ccc.cards[cardID]
	if !exists {
		return nil // Already removed
	}

	// Remove from status index
	ccc.removeFromStatus(card.Status, cardID)

	// Remove from main cache
	delete(ccc.cards, cardID)

	return nil
}

// removeFromStatus removes a card ID from the status index
func (ccc *CompactCardCache) removeFromStatus(status kanban.TaskStatus, cardID string) {
	cardIDs := ccc.byStatus[status]
	for i, id := range cardIDs {
		if id == cardID {
			// Remove by swapping with last element
			cardIDs[i] = cardIDs[len(cardIDs)-1]
			ccc.byStatus[status] = cardIDs[:len(cardIDs)-1]
			break
		}
	}
}

// cleanupExpiredEntries removes old entries from the cache
func (ccc *CompactCardCache) cleanupExpiredEntries() {
	now := time.Now()
	var toRemove []string

	for cardID, card := range ccc.cards {
		if now.Sub(card.LastAccess) > ccc.maxAge {
			toRemove = append(toRemove, cardID)
		}
	}

	for _, cardID := range toRemove {
		card := ccc.cards[cardID]
		ccc.removeFromStatus(card.Status, cardID)
		delete(ccc.cards, cardID)
	}

	ccc.lastCleanup = now
}

// evictOldestEntries removes the oldest entries from the cache
func (ccc *CompactCardCache) evictOldestEntries(count int) {
	type cardAge struct {
		id     string
		access time.Time
	}

	var oldest []cardAge
	for cardID, card := range ccc.cards {
		oldest = append(oldest, cardAge{id: cardID, access: card.LastAccess})
	}

	// Sort by access time (oldest first)
	for i := 0; i < len(oldest)-1; i++ {
		for j := i + 1; j < len(oldest); j++ {
			if oldest[i].access.After(oldest[j].access) {
				oldest[i], oldest[j] = oldest[j], oldest[i]
			}
		}
	}

	// Remove oldest entries
	for i := 0; i < count && i < len(oldest); i++ {
		cardID := oldest[i].id
		card := ccc.cards[cardID]
		ccc.removeFromStatus(card.Status, cardID)
		delete(ccc.cards, cardID)
	}
}

// calculateHash computes a simple hash for change detection
func (ccc *CompactCardCache) calculateHash(task *kanban.Task) uint64 {
	// Simple hash based on key fields
	const fnvPrime = 1099511628211
	const fnvOffset = 14695981039346656037

	data := task.Title + string(task.Status) + string(task.Priority) + task.AssignedTo
	hash := uint64(fnvOffset)
	for _, b := range []byte(data) {
		hash ^= uint64(b)
		hash *= fnvPrime
	}
	return hash
}

// GetStats returns cache statistics
func (ccc *CompactCardCache) GetStats() map[string]interface{} {
	ccc.mu.RLock()
	defer ccc.mu.RUnlock()

	statusCounts := make(map[string]int)
	for status, cards := range ccc.byStatus {
		statusCounts[string(status)] = len(cards)
	}

	return map[string]interface{}{
		"total_cards":   len(ccc.cards),
		"max_size":      ccc.maxSize,
		"status_counts": statusCounts,
		"last_cleanup":  ccc.lastCleanup,
	}
}

// Clear removes all entries from the cache
func (ccc *CompactCardCache) Clear() {
	ccc.mu.Lock()
	defer ccc.mu.Unlock()

	ccc.cards = make(map[string]*CompactCard)
	ccc.byStatus = make(map[kanban.TaskStatus][]string)
	ccc.lastCleanup = time.Now()
}
