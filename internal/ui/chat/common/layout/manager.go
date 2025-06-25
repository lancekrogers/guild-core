// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layout

import (
	"context"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Manager manages the three-pane layout for the Guild Chat interface
type Manager struct {
	// Configuration
	config      LayoutConfig
	paneManager *PaneManager

	// Layout state
	currentLayout map[string]Rectangle
	initialized   bool

	// Context
	ctx context.Context
}

// NewManager creates a new layout manager for the chat interface
func NewManager(width, height int) *Manager {
	ctx := context.Background()

	return &Manager{
		config: LayoutConfig{
			Direction:       DirectionColumn,
			MainAxisGap:     1,
			CrossAxisGap:    0,
			ContainerWidth:  width,
			ContainerHeight: height,
		},
		paneManager:   NewPaneManager(ctx),
		currentLayout: make(map[string]Rectangle),
		ctx:           ctx,
	}
}

// Initialize sets up the default three-pane layout for Guild Chat
func (m *Manager) Initialize() error {
	// Define the three-pane layout: output (top), input (middle), status (bottom)
	constraints := map[string]LayoutConstraints{
		"output": OutputPaneConstraints(),
		"input":  InputPaneConstraints(),
		"status": StatusPaneConstraints(),
	}

	// Calculate initial layout
	layout, err := CalculateLayout(m.config, constraints)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate initial layout").
			WithComponent("layout.manager").
			WithOperation("Initialize")
	}

	m.currentLayout = layout
	m.initialized = true

	return nil
}

// Resize updates the layout when the terminal size changes
func (m *Manager) Resize(width, height int) error {
	if width < 40 || height < 10 {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "terminal size too small: %dx%d (minimum 40x10)", width, height).
			WithComponent("layout.manager").
			WithOperation("Resize")
	}

	m.config.ContainerWidth = width
	m.config.ContainerHeight = height

	// Recalculate layout with new dimensions
	return m.RecalculateLayout()
}

// RecalculateLayout recalculates the layout based on current configuration
func (m *Manager) RecalculateLayout() error {
	if !m.initialized {
		return m.Initialize()
	}

	// Get current pane constraints
	constraints := map[string]LayoutConstraints{
		"output": OutputPaneConstraints(),
		"input":  InputPaneConstraints(),
		"status": StatusPaneConstraints(),
	}

	// If we have a pane manager with panes, use their constraints
	if len(m.paneManager.panes) > 0 {
		constraints = m.paneManager.GetPaneConstraints()
	}

	// Calculate new layout
	layout, err := CalculateLayout(m.config, constraints)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to recalculate layout").
			WithComponent("layout.manager").
			WithOperation("RecalculateLayout")
	}

	m.currentLayout = layout

	// Apply layout to managed panes
	if len(m.paneManager.panes) > 0 {
		return m.paneManager.ApplyLayout(layout)
	}

	return nil
}

// GetPaneRect returns the rectangle for a specific pane
func (m *Manager) GetPaneRect(paneID string) Rectangle {
	if rect, exists := m.currentLayout[paneID]; exists {
		return rect
	}

	// Return default rectangle if not found
	return Rectangle{X: 0, Y: 0, Width: 80, Height: 24}
}

// SetPaneConstraints updates constraints for a specific pane
func (m *Manager) SetPaneConstraints(paneID string, constraints LayoutConstraints) error {
	if err := ValidateConstraints(constraints); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid constraints").
			WithComponent("layout.manager").
			WithOperation("SetPaneConstraints").
			WithDetails("pane_id", paneID)
	}

	// Store updated constraints in pane manager if available
	if m.paneManager != nil {
		m.paneManager.UpdatePaneConstraints(paneID, constraints)
	}
	
	// Recalculate layout with new constraints
	return m.RecalculateLayout()
}

// UpdateInputHeight dynamically updates the input pane height based on content
func (m *Manager) UpdateInputHeight(contentLines int) error {
	// Calculate new preferred height (1-8 lines)
	newHeight := contentLines
	if newHeight < 1 {
		newHeight = 1
	}
	if newHeight > 8 {
		newHeight = 8
	}
	
	// Update input pane constraints
	constraints := InputPaneConstraints()
	constraints.PreferredHeight = newHeight
	
	return m.SetPaneConstraints("input", constraints)
}

// UpdateStatusHeight dynamically updates the status pane height based on content
func (m *Manager) UpdateStatusHeight(contentLines int) error {
	// Use the new function to get constraints with content
	constraints := StatusPaneConstraintsWithContent(contentLines)
	return m.SetPaneConstraints("status", constraints)
}

// HideStatusPane hides the status pane by setting height to 0
func (m *Manager) HideStatusPane() error {
	constraints := StatusPaneConstraints() // This returns 0 height constraints
	return m.SetPaneConstraints("status", constraints)
}

// ShowStatusPane shows the status pane with content
func (m *Manager) ShowStatusPane(contentLines int) error {
	return m.UpdateStatusHeight(contentLines)
}

// Render composites the views from all panes into the final layout
func (m *Manager) Render(paneViews map[string]string) string {
	if !m.initialized {
		return "Layout not initialized"
	}

	// Create a canvas to compose the layout
	canvas := make([][]rune, m.config.ContainerHeight)
	for i := range canvas {
		canvas[i] = make([]rune, m.config.ContainerWidth)
		// Fill with spaces
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// Render each pane into the canvas
	paneOrder := []string{"output", "input", "status"} // Output first, then input, status last for overlay effect

	for _, paneID := range paneOrder {
		rect, exists := m.currentLayout[paneID]
		if !exists {
			continue
		}

		// Skip rendering status pane if it has 0 height (hidden)
		if paneID == "status" && rect.Height == 0 {
			continue
		}

		view, exists := paneViews[paneID]
		if !exists {
			continue
		}

		// Skip rendering if view is empty and pane is status
		if paneID == "status" && strings.TrimSpace(view) == "" {
			continue
		}

		m.renderPaneToCanvas(canvas, rect, view)
	}

	// Convert canvas to string
	return m.canvasToString(canvas)
}

// renderPaneToCanvas renders a pane view into the canvas at the specified rectangle
func (m *Manager) renderPaneToCanvas(canvas [][]rune, rect Rectangle, view string) {
	lines := strings.Split(view, "\n")

	for lineIdx, line := range lines {
		y := rect.Y + lineIdx
		if y >= len(canvas) || y < 0 {
			continue
		}

		runes := []rune(line)
		for runeIdx, r := range runes {
			x := rect.X + runeIdx
			if x >= len(canvas[y]) || x < 0 {
				continue
			}
			canvas[y][x] = r
		}
	}
}

// canvasToString converts the canvas to a string
func (m *Manager) canvasToString(canvas [][]rune) string {
	var lines []string
	for _, row := range canvas {
		lines = append(lines, string(row))
	}
	return strings.Join(lines, "\n")
}

// GetLayout returns the current layout rectangles
func (m *Manager) GetLayout() map[string]Rectangle {
	// Return a copy to prevent external modification
	layout := make(map[string]Rectangle)
	for k, v := range m.currentLayout {
		layout[k] = v
	}
	return layout
}

// SetLayoutDirection changes the layout direction (column or row)
func (m *Manager) SetLayoutDirection(direction LayoutDirection) error {
	m.config.Direction = direction
	return m.RecalculateLayout()
}

// SetGaps updates the spacing between panes
func (m *Manager) SetGaps(mainAxisGap, crossAxisGap int) error {
	if mainAxisGap < 0 || crossAxisGap < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "gaps cannot be negative", nil).
			WithComponent("layout.manager").
			WithOperation("SetGaps")
	}

	m.config.MainAxisGap = mainAxisGap
	m.config.CrossAxisGap = crossAxisGap

	return m.RecalculateLayout()
}

// GetDimensions returns the current container dimensions
func (m *Manager) GetDimensions() (int, int) {
	return m.config.ContainerWidth, m.config.ContainerHeight
}

// IsInitialized returns whether the layout manager has been initialized
func (m *Manager) IsInitialized() bool {
	return m.initialized
}

// Validate checks if the current layout is valid
func (m *Manager) Validate() error {
	if !m.initialized {
		return gerror.New(gerror.ErrCodeInternal, "layout manager not initialized", nil).
			WithComponent("layout.manager").
			WithOperation("Validate")
	}

	if m.config.ContainerWidth < 1 || m.config.ContainerHeight < 1 {
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid container dimensions", nil).
			WithComponent("layout.manager").
			WithOperation("Validate")
	}

	// Validate that all expected panes have rectangles
	expectedPanes := []string{"output", "input", "status"}
	for _, paneID := range expectedPanes {
		if _, exists := m.currentLayout[paneID]; !exists {
			return gerror.Newf(gerror.ErrCodeInternal, "missing layout for pane: %s", paneID).
				WithComponent("layout.manager").
				WithOperation("Validate")
		}
	}

	return nil
}

// GetPaneAtPosition returns the pane ID at the given coordinates (if any)
func (m *Manager) GetPaneAtPosition(x, y int) string {
	for paneID, rect := range m.currentLayout {
		if x >= rect.X && x < rect.X+rect.Width &&
			y >= rect.Y && y < rect.Y+rect.Height {
			return paneID
		}
	}
	return ""
}

// Debug returns debug information about the current layout
func (m *Manager) Debug() map[string]interface{} {
	debug := make(map[string]interface{})

	debug["initialized"] = m.initialized
	debug["container_width"] = m.config.ContainerWidth
	debug["container_height"] = m.config.ContainerHeight
	debug["direction"] = m.config.Direction
	debug["main_axis_gap"] = m.config.MainAxisGap
	debug["cross_axis_gap"] = m.config.CrossAxisGap

	layoutInfo := make(map[string]interface{})
	for paneID, rect := range m.currentLayout {
		layoutInfo[paneID] = map[string]interface{}{
			"x":      rect.X,
			"y":      rect.Y,
			"width":  rect.Width,
			"height": rect.Height,
		}
	}
	debug["layout"] = layoutInfo

	return debug
}

// ApplyTheme applies a visual theme to the layout
func (m *Manager) ApplyTheme(theme string) error {
	// This method can be used to apply different visual themes
	// For now, it's a placeholder for future theme support
	switch theme {
	case "default", "guild", "medieval":
		// Valid themes
		return nil
	default:
		return gerror.Newf(gerror.ErrCodeInvalidInput, "unknown theme: %s", theme).
			WithComponent("layout.manager").
			WithOperation("ApplyTheme")
	}
}

// CreateThreePaneLayout is a helper that creates the standard Guild Chat layout
func CreateThreePaneLayout(width, height int) (*Manager, error) {
	manager := NewManager(width, height)

	if err := manager.Initialize(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize three-pane layout").
			WithComponent("layout.manager").
			WithOperation("CreateThreePaneLayout")
	}

	return manager, nil
}

// LayoutPreset represents a predefined layout configuration
type LayoutPreset struct {
	Name         string
	Direction    LayoutDirection
	MainAxisGap  int
	CrossAxisGap int
	Constraints  map[string]LayoutConstraints
}

// GetPreset returns a predefined layout preset
func GetPreset(name string) (*LayoutPreset, error) {
	presets := map[string]LayoutPreset{
		"default": {
			Name:         "Default Guild Chat",
			Direction:    DirectionColumn,
			MainAxisGap:  1,
			CrossAxisGap: 0,
			Constraints: map[string]LayoutConstraints{
				"output": OutputPaneConstraints(),
				"input":  InputPaneConstraints(),
				"status": StatusPaneConstraints(),
			},
		},
		"minimal": {
			Name:         "Minimal Layout",
			Direction:    DirectionColumn,
			MainAxisGap:  0,
			CrossAxisGap: 0,
			Constraints: map[string]LayoutConstraints{
				"output": func() LayoutConstraints {
					c := OutputPaneConstraints()
					c.Padding = Padding{} // No padding
					return c
				}(),
				"input": func() LayoutConstraints {
					c := InputPaneConstraints()
					c.Padding = Padding{} // No padding
					return c
				}(),
				"status": func() LayoutConstraints {
					c := StatusPaneConstraints()
					c.Padding = Padding{} // No padding
					return c
				}(),
			},
		},
		"compact": {
			Name:         "Compact Layout",
			Direction:    DirectionColumn,
			MainAxisGap:  0,
			CrossAxisGap: 0,
			Constraints: map[string]LayoutConstraints{
				"output": func() LayoutConstraints {
					c := OutputPaneConstraints()
					c.MinHeight = 5 // Smaller output area
					return c
				}(),
				"input":  InputPaneConstraints(),
				"status": StatusPaneConstraints(),
			},
		},
	}

	preset, exists := presets[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "layout preset not found: %s", name).
			WithComponent("layout.manager").
			WithOperation("GetPreset")
	}

	return &preset, nil
}

// ApplyPreset applies a predefined layout preset
func (m *Manager) ApplyPreset(presetName string) error {
	preset, err := GetPreset(presetName)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get preset").
			WithComponent("layout.manager").
			WithOperation("ApplyPreset")
	}

	// Apply preset configuration
	m.config.Direction = preset.Direction
	m.config.MainAxisGap = preset.MainAxisGap
	m.config.CrossAxisGap = preset.CrossAxisGap

	// Calculate layout with preset constraints
	layout, err := CalculateLayout(m.config, preset.Constraints)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate preset layout").
			WithComponent("layout.manager").
			WithOperation("ApplyPreset")
	}

	m.currentLayout = layout
	m.initialized = true

	return nil
}

// RenderWithStyles renders the layout with lipgloss styling
func (m *Manager) RenderWithStyles(paneViews map[string]string, styles map[string]lipgloss.Style) string {
	if !m.initialized {
		return "Layout not initialized"
	}

	styledViews := make(map[string]string)

	// Apply styles to each pane view
	for paneID, view := range paneViews {
		if style, exists := styles[paneID]; exists {
			rect := m.GetPaneRect(paneID)
			styledView := style.Width(rect.Width).Height(rect.Height).Render(view)
			styledViews[paneID] = styledView
		} else {
			styledViews[paneID] = view
		}
	}

	return m.Render(styledViews)
}
