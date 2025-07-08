// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
)

// Viewport manages the visible portion of the kanban board for performance optimization
type Viewport struct {
	// Display dimensions
	Width  int
	Height int

	// Scroll position
	ScrollX int
	ScrollY int

	// Card dimensions
	CardHeight  int
	CardWidth   int
	CardSpacing int

	// Column layout
	ColumnCount   int
	ColumnSpacing int

	// Performance settings
	CullingEnabled bool
	BufferRows     int // Extra rows to render outside viewport for smooth scrolling
}

// CardRenderInfo contains position and visibility data for a card
type CardRenderInfo struct {
	Card      *kanban.Task
	X         int // Screen X coordinate
	Y         int // Screen Y coordinate
	Column    string
	ColumnIdx int
	CardIdx   int
	Visible   bool
	InBuffer  bool // Card is in buffer zone (not visible but rendered for performance)
}

// ColumnLayout contains position and dimension data for a column
type ColumnLayout struct {
	Index        int
	X            int
	Width        int
	ScrollY      int
	VisibleCards int
	TotalCards   int
	StartCard    int
	EndCard      int
}

// ViewportUpdate contains changes to viewport state
type ViewportUpdate struct {
	ScrollDeltaX int
	ScrollDeltaY int
	SizeChanged  bool
	NewWidth     int
	NewHeight    int
}

// NewViewport creates a new viewport with default settings
func NewViewport(width, height int) *Viewport {
	return &Viewport{
		Width:          width,
		Height:         height,
		CardHeight:     4, // 3 lines + 1 for spacing
		CardWidth:      20,
		CardSpacing:    2,
		ColumnCount:    5,
		ColumnSpacing:  1,
		CullingEnabled: true,
		BufferRows:     2, // Render 2 extra rows above/below viewport
	}
}

// Resize updates viewport dimensions
func (v *Viewport) Resize(ctx context.Context, width, height int) error {
	if width <= 0 || height <= 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid viewport dimensions", nil).
			WithComponent("kanban.viewport").
			WithOperation("Resize").
			WithDetails("width", width).
			WithDetails("height", height)
	}

	v.Width = width
	v.Height = height

	// Recalculate column width based on new viewport width
	availableWidth := v.Width - (v.ColumnCount-1)*v.ColumnSpacing
	v.CardWidth = availableWidth / v.ColumnCount
	if v.CardWidth < 15 {
		v.CardWidth = 15 // Minimum readable width
	}

	return nil
}

// Scroll updates the viewport scroll position
func (v *Viewport) Scroll(ctx context.Context, deltaX, deltaY int) {
	// Update scroll position with bounds checking
	v.ScrollX += deltaX
	v.ScrollY += deltaY

	// Clamp scroll position to valid bounds
	if v.ScrollX < 0 {
		v.ScrollX = 0
	}
	if v.ScrollY < 0 {
		v.ScrollY = 0
	}

	// Note: Upper bounds checking would require knowledge of total content size
	// This is handled by the calling code in the kanban model
}

// CalculateColumnLayouts computes the layout for all columns
func (v *Viewport) CalculateColumnLayouts(ctx context.Context, columns []*Column) ([]ColumnLayout, error) {
	if len(columns) != v.ColumnCount {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "column count mismatch", nil).
			WithComponent("kanban.viewport").
			WithOperation("CalculateColumnLayouts").
			WithDetails("expected", v.ColumnCount).
			WithDetails("actual", len(columns))
	}

	layouts := make([]ColumnLayout, v.ColumnCount)

	// Calculate header space (title, borders, etc.)
	headerHeight := 6                            // Board header + column headers + separators
	contentHeight := v.Height - headerHeight - 2 // -2 for status bar and bottom border
	if contentHeight < 1 {
		contentHeight = 1
	}

	visibleRows := contentHeight / v.CardHeight
	if visibleRows < 1 {
		visibleRows = 1
	}

	for i, col := range columns {
		// Calculate column X position
		colX := i * (v.CardWidth + v.ColumnSpacing)

		// Calculate visible card range for this column
		firstVisibleCard := col.ScrollOffset
		lastVisibleCard := firstVisibleCard + visibleRows
		if lastVisibleCard > col.TotalTasks {
			lastVisibleCard = col.TotalTasks
		}

		// Add buffer rows if culling is enabled
		bufferStart := firstVisibleCard
		bufferEnd := lastVisibleCard

		if v.CullingEnabled && v.BufferRows > 0 {
			bufferStart = firstVisibleCard - v.BufferRows
			bufferEnd = lastVisibleCard + v.BufferRows

			// Clamp to valid range
			if bufferStart < 0 {
				bufferStart = 0
			}
			if bufferEnd > col.TotalTasks {
				bufferEnd = col.TotalTasks
			}
		}

		layouts[i] = ColumnLayout{
			Index:        i,
			X:            colX,
			Width:        v.CardWidth,
			ScrollY:      col.ScrollOffset * v.CardHeight,
			VisibleCards: visibleRows,
			TotalCards:   col.TotalTasks,
			StartCard:    bufferStart,
			EndCard:      bufferEnd,
		}
	}

	return layouts, nil
}

// VisibleCards returns only the cards that should be rendered for performance
func (v *Viewport) VisibleCards(ctx context.Context, columns []*Column) ([]CardRenderInfo, error) {
	if !v.CullingEnabled {
		// If culling is disabled, return all cards
		return v.getAllCards(ctx, columns)
	}

	layouts, err := v.CalculateColumnLayouts(ctx, columns)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate column layouts").
			WithComponent("kanban.viewport").
			WithOperation("VisibleCards")
	}

	// Pre-allocate slice with estimated capacity
	estimatedCards := 0
	for _, layout := range layouts {
		cardsInLayout := layout.EndCard - layout.StartCard
		if cardsInLayout > 0 && cardsInLayout < 10000 { // Sanity check
			estimatedCards += cardsInLayout
		}
	}
	if estimatedCards < 0 || estimatedCards > 10000 { // Safety limit
		estimatedCards = 100
	}
	visible := make([]CardRenderInfo, 0, estimatedCards)

	for i, col := range columns {
		layout := layouts[i]

		// Skip columns that are completely outside the viewport
		if layout.X+layout.Width < v.ScrollX || layout.X > v.ScrollX+v.Width {
			continue
		}

		// Calculate screen X position for this column
		screenX := layout.X - v.ScrollX

		// Process cards in the visible/buffer range
		for cardIdx := layout.StartCard; cardIdx < layout.EndCard && cardIdx < len(col.Tasks); cardIdx++ {
			card := col.Tasks[cardIdx]
			if card == nil {
				continue
			}

			// Calculate screen Y position
			cardY := cardIdx * v.CardHeight
			screenY := cardY - layout.ScrollY

			// Determine if card is actually visible or just in buffer
			isVisible := screenY >= 0 && screenY < v.Height-v.CardHeight
			isInBuffer := !isVisible && cardIdx >= layout.StartCard && cardIdx < layout.EndCard

			visible = append(visible, CardRenderInfo{
				Card:      card,
				X:         screenX,
				Y:         screenY,
				Column:    col.Title,
				ColumnIdx: i,
				CardIdx:   cardIdx,
				Visible:   isVisible,
				InBuffer:  isInBuffer,
			})
		}
	}

	return visible, nil
}

// getAllCards returns all cards without culling (for debugging or when disabled)
func (v *Viewport) getAllCards(ctx context.Context, columns []*Column) ([]CardRenderInfo, error) {
	var allCards []CardRenderInfo

	for colIdx, col := range columns {
		colX := colIdx * (v.CardWidth + v.ColumnSpacing)
		screenX := colX - v.ScrollX

		for cardIdx, card := range col.Tasks {
			if card == nil {
				continue
			}

			cardY := cardIdx * v.CardHeight
			screenY := cardY - (col.ScrollOffset * v.CardHeight)

			allCards = append(allCards, CardRenderInfo{
				Card:      card,
				X:         screenX,
				Y:         screenY,
				Column:    col.Title,
				ColumnIdx: colIdx,
				CardIdx:   cardIdx,
				Visible:   true,
				InBuffer:  false,
			})
		}
	}

	return allCards, nil
}

// EstimateMemoryUsage calculates approximate memory usage for the current viewport
func (v *Viewport) EstimateMemoryUsage(ctx context.Context, columns []*Column) (int64, error) {
	visibleCards, err := v.VisibleCards(ctx, columns)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get visible cards").
			WithComponent("kanban.viewport").
			WithOperation("EstimateMemoryUsage")
	}

	// Rough estimation of memory per card
	const avgCardSize = 500        // bytes (task data + rendered strings)
	const avgRenderCacheSize = 200 // bytes per cached render

	memoryUsage := int64(len(visibleCards)) * (avgCardSize + avgRenderCacheSize)

	// Add viewport overhead
	const viewportOverhead = 1024 // bytes
	memoryUsage += viewportOverhead

	return memoryUsage, nil
}

// GetScrollBounds calculates the maximum scroll values for current content
func (v *Viewport) GetScrollBounds(ctx context.Context, columns []*Column) (maxX, maxY int, err error) {
	if len(columns) == 0 {
		return 0, 0, nil
	}

	// Calculate maximum horizontal scroll
	totalContentWidth := v.ColumnCount*v.CardWidth + (v.ColumnCount-1)*v.ColumnSpacing
	maxX = totalContentWidth - v.Width
	if maxX < 0 {
		maxX = 0
	}

	// Calculate maximum vertical scroll (find tallest column)
	maxCards := 0
	for _, col := range columns {
		if col.TotalTasks > maxCards {
			maxCards = col.TotalTasks
		}
	}

	totalContentHeight := maxCards * v.CardHeight
	visibleHeight := v.Height - 8 // Account for headers and status bar
	maxY = totalContentHeight - visibleHeight
	if maxY < 0 {
		maxY = 0
	}

	return maxX, maxY, nil
}

// ClampScrollPosition ensures scroll position is within valid bounds
func (v *Viewport) ClampScrollPosition(ctx context.Context, columns []*Column) error {
	maxX, maxY, err := v.GetScrollBounds(ctx, columns)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get scroll bounds").
			WithComponent("kanban.viewport").
			WithOperation("ClampScrollPosition")
	}

	if v.ScrollX < 0 {
		v.ScrollX = 0
	} else if v.ScrollX > maxX {
		v.ScrollX = maxX
	}

	if v.ScrollY < 0 {
		v.ScrollY = 0
	} else if v.ScrollY > maxY {
		v.ScrollY = maxY
	}

	return nil
}

// IsCardVisible determines if a specific card is currently visible
func (v *Viewport) IsCardVisible(ctx context.Context, columnIdx, cardIdx int, columns []*Column) (bool, error) {
	if columnIdx < 0 || columnIdx >= len(columns) {
		return false, gerror.New(gerror.ErrCodeInvalidInput, "invalid column index", nil).
			WithComponent("kanban.viewport").
			WithOperation("IsCardVisible").
			WithDetails("column_idx", columnIdx)
	}

	col := columns[columnIdx]
	if cardIdx < 0 || cardIdx >= len(col.Tasks) {
		return false, nil // Card doesn't exist
	}

	// Calculate column position
	colX := columnIdx * (v.CardWidth + v.ColumnSpacing)

	// Check horizontal visibility
	if colX+v.CardWidth < v.ScrollX || colX > v.ScrollX+v.Width {
		return false, nil
	}

	// Check vertical visibility
	cardY := cardIdx * v.CardHeight
	screenY := cardY - (col.ScrollOffset * v.CardHeight)

	if screenY < 0 || screenY >= v.Height-v.CardHeight {
		return false, nil
	}

	return true, nil
}

// GetOptimalBufferSize calculates the optimal buffer size based on performance
func (v *Viewport) GetOptimalBufferSize(ctx context.Context, avgRenderTime float64) int {
	// Adjust buffer size based on render performance
	targetFrameTime := 16.67 // 60 FPS in milliseconds

	if avgRenderTime < targetFrameTime/2 {
		return 4 // High performance, larger buffer for smoother scrolling
	} else if avgRenderTime < targetFrameTime {
		return 2 // Good performance, moderate buffer
	} else {
		return 1 // Poor performance, minimal buffer
	}
}

// UpdateBufferSize dynamically adjusts buffer size for optimal performance
func (v *Viewport) UpdateBufferSize(ctx context.Context, newBufferSize int) error {
	if newBufferSize < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "buffer size cannot be negative", nil).
			WithComponent("kanban.viewport").
			WithOperation("UpdateBufferSize").
			WithDetails("buffer_size", newBufferSize)
	}

	v.BufferRows = newBufferSize
	return nil
}

// GetViewportStats returns statistics about current viewport state
func (v *Viewport) GetViewportStats(ctx context.Context, columns []*Column) (map[string]interface{}, error) {
	visibleCards, err := v.VisibleCards(ctx, columns)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get visible cards").
			WithComponent("kanban.viewport").
			WithOperation("GetViewportStats")
	}

	memUsage, _ := v.EstimateMemoryUsage(ctx, columns)
	maxX, maxY, _ := v.GetScrollBounds(ctx, columns)

	totalCards := 0
	for _, col := range columns {
		totalCards += col.TotalTasks
	}

	visibleCount := 0
	bufferCount := 0
	for _, card := range visibleCards {
		if card.Visible {
			visibleCount++
		} else if card.InBuffer {
			bufferCount++
		}
	}

	return map[string]interface{}{
		"viewport_width":  v.Width,
		"viewport_height": v.Height,
		"scroll_x":        v.ScrollX,
		"scroll_y":        v.ScrollY,
		"max_scroll_x":    maxX,
		"max_scroll_y":    maxY,
		"card_width":      v.CardWidth,
		"card_height":     v.CardHeight,
		"buffer_rows":     v.BufferRows,
		"culling_enabled": v.CullingEnabled,
		"total_cards":     totalCards,
		"visible_cards":   visibleCount,
		"buffer_cards":    bufferCount,
		"rendered_cards":  len(visibleCards),
		"memory_usage_mb": float64(memUsage) / 1024 / 1024,
		"culling_ratio":   float64(len(visibleCards)) / float64(totalCards),
	}, nil
}
