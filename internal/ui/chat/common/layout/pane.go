// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layout

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// PaneInterface defines the common interface for all panes
type PaneInterface interface {
	// Bubble Tea model interface
	tea.Model

	// Layout interface
	Resize(width, height int)
	GetRect() Rectangle
	SetRect(rect Rectangle)
	GetConstraints() LayoutConstraints
	SetConstraints(constraints LayoutConstraints)

	// Pane identification
	ID() string
	SetID(id string)

	// Focus management
	IsFocused() bool
	SetFocus(focused bool)

	// Content management
	SetContent(content string)
	GetContent() string

	// Style management
	SetStyle(style lipgloss.Style)
	GetStyle() lipgloss.Style
}

// BasePane provides common functionality for all panes
type BasePane struct {
	// Identity
	id string

	// Layout properties
	rect        Rectangle
	constraints LayoutConstraints

	// State
	focused bool
	visible bool

	// Content
	content string

	// Styling
	style       lipgloss.Style
	focusStyle  lipgloss.Style
	borderStyle lipgloss.Style

	// Context
	ctx context.Context
}

// NewBasePane creates a new base pane
func NewBasePane(ctx context.Context, id string, width, height int) *BasePane {
	return &BasePane{
		id:      id,
		ctx:     ctx,
		visible: true,
		rect: Rectangle{
			X:      0,
			Y:      0,
			Width:  width,
			Height: height,
		},
		constraints: DefaultConstraints(),
		style:       lipgloss.NewStyle(),
		focusStyle:  lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("141")),
		borderStyle: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
	}
}

// Implement PaneInterface

func (bp *BasePane) ID() string {
	return bp.id
}

func (bp *BasePane) SetID(id string) {
	bp.id = id
}

func (bp *BasePane) Resize(width, height int) {
	bp.rect.Width = width
	bp.rect.Height = height
}

func (bp *BasePane) GetRect() Rectangle {
	return bp.rect
}

func (bp *BasePane) SetRect(rect Rectangle) {
	bp.rect = rect
}

func (bp *BasePane) GetConstraints() LayoutConstraints {
	return bp.constraints
}

func (bp *BasePane) SetConstraints(constraints LayoutConstraints) {
	bp.constraints = constraints
}

func (bp *BasePane) IsFocused() bool {
	return bp.focused
}

func (bp *BasePane) SetFocus(focused bool) {
	bp.focused = focused
}

func (bp *BasePane) SetContent(content string) {
	bp.content = content
}

func (bp *BasePane) GetContent() string {
	return bp.content
}

func (bp *BasePane) SetStyle(style lipgloss.Style) {
	bp.style = style
}

func (bp *BasePane) GetStyle() lipgloss.Style {
	if bp.focused {
		return bp.focusStyle
	}
	return bp.style
}

// Implement tea.Model interface

func (bp *BasePane) Init() tea.Cmd {
	return nil
}

func (bp *BasePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		bp.Resize(msg.Width, msg.Height)
	}
	return bp, nil
}

func (bp *BasePane) View() string {
	if !bp.visible {
		return ""
	}

	// Apply styling based on focus state
	style := bp.GetStyle()

	// Set dimensions
	style = style.Width(bp.rect.Width).Height(bp.rect.Height)

	// Render content
	return style.Render(bp.content)
}

// Additional utility methods

func (bp *BasePane) IsVisible() bool {
	return bp.visible
}

func (bp *BasePane) SetVisible(visible bool) {
	bp.visible = visible
}

func (bp *BasePane) GetDimensions() (int, int) {
	return bp.rect.Width, bp.rect.Height
}

func (bp *BasePane) GetPosition() (int, int) {
	return bp.rect.X, bp.rect.Y
}

func (bp *BasePane) SetPosition(x, y int) {
	bp.rect.X = x
	bp.rect.Y = y
}

func (bp *BasePane) GetInnerDimensions() (int, int) {
	// Calculate inner dimensions accounting for padding and borders
	padding := bp.constraints.Padding

	innerWidth := bp.rect.Width - padding.Left - padding.Right
	innerHeight := bp.rect.Height - padding.Top - padding.Bottom

	// Account for borders if present
	// Note: Border detection simplified - in production would check actual border
	if bp.focused {
		innerWidth -= 2  // Left and right borders
		innerHeight -= 2 // Top and bottom borders
	}

	// Ensure minimum dimensions
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	return innerWidth, innerHeight
}

func (bp *BasePane) ValidateDimensions() error {
	if bp.rect.Width < bp.constraints.MinWidth {
		return gerror.Newf(gerror.ErrCodeInvalidInput,
			"pane width %d is less than minimum %d",
			bp.rect.Width, bp.constraints.MinWidth).
			WithComponent("layout.pane").
			WithOperation("ValidateDimensions").
			WithDetails("pane_id", bp.id)
	}

	if bp.rect.Height < bp.constraints.MinHeight {
		return gerror.Newf(gerror.ErrCodeInvalidInput,
			"pane height %d is less than minimum %d",
			bp.rect.Height, bp.constraints.MinHeight).
			WithComponent("layout.pane").
			WithOperation("ValidateDimensions").
			WithDetails("pane_id", bp.id)
	}

	if bp.constraints.MaxWidth > 0 && bp.rect.Width > bp.constraints.MaxWidth {
		return gerror.Newf(gerror.ErrCodeInvalidInput,
			"pane width %d exceeds maximum %d",
			bp.rect.Width, bp.constraints.MaxWidth).
			WithComponent("layout.pane").
			WithOperation("ValidateDimensions").
			WithDetails("pane_id", bp.id)
	}

	if bp.constraints.MaxHeight > 0 && bp.rect.Height > bp.constraints.MaxHeight {
		return gerror.Newf(gerror.ErrCodeInvalidInput,
			"pane height %d exceeds maximum %d",
			bp.rect.Height, bp.constraints.MaxHeight).
			WithComponent("layout.pane").
			WithOperation("ValidateDimensions").
			WithDetails("pane_id", bp.id)
	}

	return nil
}

// ApplyConstraints applies layout constraints to the pane dimensions
func (bp *BasePane) ApplyConstraints() {
	// Apply minimum constraints
	if bp.rect.Width < bp.constraints.MinWidth {
		bp.rect.Width = bp.constraints.MinWidth
	}
	if bp.rect.Height < bp.constraints.MinHeight {
		bp.rect.Height = bp.constraints.MinHeight
	}

	// Apply maximum constraints
	if bp.constraints.MaxWidth > 0 && bp.rect.Width > bp.constraints.MaxWidth {
		bp.rect.Width = bp.constraints.MaxWidth
	}
	if bp.constraints.MaxHeight > 0 && bp.rect.Height > bp.constraints.MaxHeight {
		bp.rect.Height = bp.constraints.MaxHeight
	}
}

// SetFocusStyle sets the style to use when the pane is focused
func (bp *BasePane) SetFocusStyle(style lipgloss.Style) {
	bp.focusStyle = style
}

// SetBorderStyle sets the style to use for borders when not focused
func (bp *BasePane) SetBorderStyle(style lipgloss.Style) {
	bp.borderStyle = style
}

// GetBorderStyle gets the appropriate border style based on focus state
func (bp *BasePane) GetBorderStyle() lipgloss.Style {
	if bp.focused {
		return bp.focusStyle
	}
	return bp.borderStyle
}

// Helper methods for common styling patterns

// ApplyDefaultStyling applies default Guild Chat styling
func (bp *BasePane) ApplyDefaultStyling() {
	bp.style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	bp.focusStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")).
		Padding(0, 1)
}

// ApplyMedievalStyling applies medieval-themed styling
func (bp *BasePane) ApplyMedievalStyling() {
	bp.style = lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("208")). // Orange/amber
		Padding(0, 1)

	bp.focusStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("141")). // Purple
		Padding(0, 1).
		Bold(true)
}

// ApplyMinimalStyling applies minimal styling (no borders)
func (bp *BasePane) ApplyMinimalStyling() {
	bp.style = lipgloss.NewStyle().
		Padding(0, 1)

	bp.focusStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark gray background
		Padding(0, 1)
}

// RenderWithBorder renders the pane content with appropriate border
func (bp *BasePane) RenderWithBorder(content string) string {
	style := bp.GetBorderStyle()

	// Set dimensions
	style = style.Width(bp.rect.Width).Height(bp.rect.Height)

	return style.Render(content)
}

// RenderWithoutBorder renders the pane content without border
func (bp *BasePane) RenderWithoutBorder(content string) string {
	style := bp.style

	// Set dimensions
	style = style.Width(bp.rect.Width).Height(bp.rect.Height)

	return style.Render(content)
}

// PaneManager manages a collection of panes
type PaneManager struct {
	panes map[string]PaneInterface
	order []string
	ctx   context.Context
}

// NewPaneManager creates a new pane manager
func NewPaneManager(ctx context.Context) *PaneManager {
	return &PaneManager{
		panes: make(map[string]PaneInterface),
		order: make([]string, 0),
		ctx:   ctx,
	}
}

// AddPane adds a pane to the manager
func (pm *PaneManager) AddPane(pane PaneInterface) error {
	if pane == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "pane cannot be nil", nil).
			WithComponent("layout.pane").
			WithOperation("AddPane")
	}

	id := pane.ID()
	if id == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "pane ID cannot be empty", nil).
			WithComponent("layout.pane").
			WithOperation("AddPane")
	}

	if _, exists := pm.panes[id]; exists {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "pane with ID %s already exists", id).
			WithComponent("layout.pane").
			WithOperation("AddPane")
	}

	pm.panes[id] = pane
	pm.order = append(pm.order, id)

	return nil
}

// RemovePane removes a pane from the manager
func (pm *PaneManager) RemovePane(id string) error {
	if _, exists := pm.panes[id]; !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "pane with ID %s not found", id).
			WithComponent("layout.pane").
			WithOperation("RemovePane")
	}

	delete(pm.panes, id)

	// Remove from order
	for i, paneID := range pm.order {
		if paneID == id {
			pm.order = append(pm.order[:i], pm.order[i+1:]...)
			break
		}
	}

	return nil
}

// GetPane retrieves a pane by ID
func (pm *PaneManager) GetPane(id string) (PaneInterface, error) {
	pane, exists := pm.panes[id]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "pane with ID %s not found", id).
			WithComponent("layout.pane").
			WithOperation("GetPane")
	}

	return pane, nil
}

// GetAllPanes returns all panes in order
func (pm *PaneManager) GetAllPanes() []PaneInterface {
	var panes []PaneInterface
	for _, id := range pm.order {
		if pane, exists := pm.panes[id]; exists {
			panes = append(panes, pane)
		}
	}
	return panes
}

// GetPaneConstraints returns a map of pane constraints for layout calculation
func (pm *PaneManager) GetPaneConstraints() map[string]LayoutConstraints {
	constraints := make(map[string]LayoutConstraints)
	for id, pane := range pm.panes {
		constraints[id] = pane.GetConstraints()
	}
	return constraints
}

// ApplyLayout applies calculated layout rectangles to panes
func (pm *PaneManager) ApplyLayout(layout map[string]Rectangle) error {
	for id, rect := range layout {
		pane, exists := pm.panes[id]
		if !exists {
			continue // Skip panes that don't exist
		}

		pane.SetRect(rect)

		// Validate dimensions after applying layout
		if basePane, ok := pane.(*BasePane); ok {
			if err := basePane.ValidateDimensions(); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "layout validation failed").
					WithComponent("layout.pane").
					WithOperation("ApplyLayout").
					WithDetails("pane_id", id)
			}
		}
	}

	return nil
}

// FocusPane sets focus to a specific pane
func (pm *PaneManager) FocusPane(id string) error {
	// Remove focus from all panes
	for _, pane := range pm.panes {
		pane.SetFocus(false)
	}

	// Set focus to specified pane
	pane, exists := pm.panes[id]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "pane with ID %s not found", id).
			WithComponent("layout.pane").
			WithOperation("FocusPane")
	}

	pane.SetFocus(true)
	return nil
}

// GetFocusedPane returns the currently focused pane
func (pm *PaneManager) GetFocusedPane() PaneInterface {
	for _, pane := range pm.panes {
		if pane.IsFocused() {
			return pane
		}
	}
	return nil
}

// UpdateAll updates all panes with a Bubble Tea message
func (pm *PaneManager) UpdateAll(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	for _, pane := range pm.panes {
		var cmd tea.Cmd
		_, cmd = pane.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}
