// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layout

import (
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// LayoutConstraints define how panes should be arranged and sized
type LayoutConstraints struct {
	MinWidth   int
	MinHeight  int
	MaxWidth   int     // 0 means no limit
	MaxHeight  int     // 0 means no limit
	FlexGrow   float64 // How much extra space this pane should consume
	FlexShrink float64 // How much this pane should shrink when space is limited
	Padding    Padding
	Margin     Margin
}

// Padding defines internal spacing for a pane
type Padding struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// Margin defines external spacing around a pane
type Margin struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// Rectangle represents a positioned area in the layout
type Rectangle struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutDirection defines how panes are arranged
type LayoutDirection int

const (
	DirectionColumn LayoutDirection = iota // Vertical stack
	DirectionRow                           // Horizontal arrangement
)

// LayoutConfig defines the overall layout configuration
type LayoutConfig struct {
	Direction       LayoutDirection
	MainAxisGap     int // Space between panes along main axis
	CrossAxisGap    int // Space between panes along cross axis
	ContainerWidth  int
	ContainerHeight int
}

// DefaultConstraints returns sensible default constraints for a pane
func DefaultConstraints() LayoutConstraints {
	return LayoutConstraints{
		MinWidth:   20,
		MinHeight:  3,
		MaxWidth:   0, // No limit
		MaxHeight:  0, // No limit
		FlexGrow:   1.0,
		FlexShrink: 1.0,
		Padding:    Padding{Top: 0, Right: 1, Bottom: 0, Left: 1},
		Margin:     Margin{Top: 0, Right: 0, Bottom: 0, Left: 0},
	}
}

// OutputPaneConstraints returns constraints optimized for the output pane
func OutputPaneConstraints() LayoutConstraints {
	constraints := DefaultConstraints()
	constraints.MinHeight = 10
	constraints.FlexGrow = 2.0 // Take most of the space
	constraints.Padding = Padding{Top: 1, Right: 1, Bottom: 1, Left: 1}
	return constraints
}

// InputPaneConstraints returns constraints optimized for the input pane
func InputPaneConstraints() LayoutConstraints {
	constraints := DefaultConstraints()
	constraints.MinHeight = 3
	constraints.MaxHeight = 10   // Limit input area height
	constraints.FlexGrow = 0.0   // Don't grow
	constraints.FlexShrink = 0.0 // Don't shrink
	constraints.Padding = Padding{Top: 0, Right: 1, Bottom: 0, Left: 1}
	return constraints
}

// StatusPaneConstraints returns constraints optimized for the status bar
func StatusPaneConstraints() LayoutConstraints {
	constraints := DefaultConstraints()
	constraints.MinHeight = 1
	constraints.MaxHeight = 1    // Fixed height status bar
	constraints.FlexGrow = 0.0   // Don't grow
	constraints.FlexShrink = 0.0 // Don't shrink
	constraints.Padding = Padding{Top: 0, Right: 1, Bottom: 0, Left: 1}
	return constraints
}

// CalculateLayout calculates the layout for a set of panes with constraints
func CalculateLayout(config LayoutConfig, paneConstraints map[string]LayoutConstraints) (map[string]Rectangle, error) {
	if len(paneConstraints) == 0 {
		return make(map[string]Rectangle), nil
	}

	// Validate container dimensions
	if config.ContainerWidth < 1 || config.ContainerHeight < 1 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "container dimensions must be positive", nil).
			WithComponent("layout.constraints").
			WithOperation("CalculateLayout")
	}

	switch config.Direction {
	case DirectionColumn:
		return calculateColumnLayout(config, paneConstraints)
	case DirectionRow:
		return calculateRowLayout(config, paneConstraints)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported layout direction", nil).
			WithComponent("layout.constraints").
			WithOperation("CalculateLayout")
	}
}

// calculateColumnLayout calculates layout for vertical stacking
func calculateColumnLayout(config LayoutConfig, paneConstraints map[string]LayoutConstraints) (map[string]Rectangle, error) {
	results := make(map[string]Rectangle)

	// Calculate available space
	availableHeight := config.ContainerHeight
	availableWidth := config.ContainerWidth

	// Account for gaps between panes
	if len(paneConstraints) > 1 {
		availableHeight -= config.MainAxisGap * (len(paneConstraints) - 1)
	}

	// First pass: allocate minimum required space and calculate margins/padding
	var paneOrder []string
	totalMinHeight := 0
	totalFlexGrow := 0.0

	for paneID, constraints := range paneConstraints {
		paneOrder = append(paneOrder, paneID)

		// Account for margins and padding in height calculation
		marginHeight := constraints.Margin.Top + constraints.Margin.Bottom
		paddingHeight := constraints.Padding.Top + constraints.Padding.Bottom
		totalMinHeight += constraints.MinHeight + marginHeight + paddingHeight

		if constraints.FlexGrow > 0 {
			totalFlexGrow += constraints.FlexGrow
		}
	}

	// Check if minimum requirements can be satisfied
	if totalMinHeight > availableHeight {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "insufficient space for minimum pane requirements", nil).
			WithComponent("layout.constraints").
			WithOperation("calculateColumnLayout")
	}

	// Second pass: distribute remaining space based on flex-grow
	remainingHeight := availableHeight - totalMinHeight
	currentY := 0

	for _, paneID := range paneOrder {
		constraints := paneConstraints[paneID]

		// Calculate pane dimensions
		marginHeight := constraints.Margin.Top + constraints.Margin.Bottom
		paddingHeight := constraints.Padding.Top + constraints.Padding.Bottom
		marginWidth := constraints.Margin.Left + constraints.Margin.Right
		paddingWidth := constraints.Padding.Left + constraints.Padding.Right

		// Start with minimum height
		paneHeight := constraints.MinHeight

		// Add flex space if applicable
		if constraints.FlexGrow > 0 && totalFlexGrow > 0 {
			flexSpace := int(float64(remainingHeight) * (constraints.FlexGrow / totalFlexGrow))
			paneHeight += flexSpace
		}

		// Apply maximum height constraint if set
		if constraints.MaxHeight > 0 && paneHeight > constraints.MaxHeight {
			paneHeight = constraints.MaxHeight
		}

		// Calculate width (full container width minus margins and padding)
		paneWidth := availableWidth - marginWidth - paddingWidth
		if constraints.MinWidth > paneWidth {
			paneWidth = constraints.MinWidth
		}
		if constraints.MaxWidth > 0 && paneWidth > constraints.MaxWidth {
			paneWidth = constraints.MaxWidth
		}

		// Position the pane accounting for margins
		x := constraints.Margin.Left + constraints.Padding.Left
		y := currentY + constraints.Margin.Top + constraints.Padding.Top

		results[paneID] = Rectangle{
			X:      x,
			Y:      y,
			Width:  paneWidth,
			Height: paneHeight,
		}

		// Move to next position
		currentY += paneHeight + marginHeight + paddingHeight + config.MainAxisGap
	}

	return results, nil
}

// calculateRowLayout calculates layout for horizontal arrangement
func calculateRowLayout(config LayoutConfig, paneConstraints map[string]LayoutConstraints) (map[string]Rectangle, error) {
	results := make(map[string]Rectangle)

	// Calculate available space
	availableWidth := config.ContainerWidth
	availableHeight := config.ContainerHeight

	// Account for gaps between panes
	if len(paneConstraints) > 1 {
		availableWidth -= config.MainAxisGap * (len(paneConstraints) - 1)
	}

	// First pass: allocate minimum required space
	var paneOrder []string
	totalMinWidth := 0
	totalFlexGrow := 0.0

	for paneID, constraints := range paneConstraints {
		paneOrder = append(paneOrder, paneID)

		// Account for margins and padding in width calculation
		marginWidth := constraints.Margin.Left + constraints.Margin.Right
		paddingWidth := constraints.Padding.Left + constraints.Padding.Right
		totalMinWidth += constraints.MinWidth + marginWidth + paddingWidth

		if constraints.FlexGrow > 0 {
			totalFlexGrow += constraints.FlexGrow
		}
	}

	// Check if minimum requirements can be satisfied
	if totalMinWidth > availableWidth {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "insufficient space for minimum pane requirements", nil).
			WithComponent("layout.constraints").
			WithOperation("calculateRowLayout")
	}

	// Second pass: distribute remaining space based on flex-grow
	remainingWidth := availableWidth - totalMinWidth
	currentX := 0

	for _, paneID := range paneOrder {
		constraints := paneConstraints[paneID]

		// Calculate pane dimensions
		marginWidth := constraints.Margin.Left + constraints.Margin.Right
		paddingWidth := constraints.Padding.Left + constraints.Padding.Right
		marginHeight := constraints.Margin.Top + constraints.Margin.Bottom
		paddingHeight := constraints.Padding.Top + constraints.Padding.Bottom

		// Start with minimum width
		paneWidth := constraints.MinWidth

		// Add flex space if applicable
		if constraints.FlexGrow > 0 && totalFlexGrow > 0 {
			flexSpace := int(float64(remainingWidth) * (constraints.FlexGrow / totalFlexGrow))
			paneWidth += flexSpace
		}

		// Apply maximum width constraint if set
		if constraints.MaxWidth > 0 && paneWidth > constraints.MaxWidth {
			paneWidth = constraints.MaxWidth
		}

		// Calculate height (full container height minus margins and padding)
		paneHeight := availableHeight - marginHeight - paddingHeight
		if constraints.MinHeight > paneHeight {
			paneHeight = constraints.MinHeight
		}
		if constraints.MaxHeight > 0 && paneHeight > constraints.MaxHeight {
			paneHeight = constraints.MaxHeight
		}

		// Position the pane accounting for margins
		x := currentX + constraints.Margin.Left + constraints.Padding.Left
		y := constraints.Margin.Top + constraints.Padding.Top

		results[paneID] = Rectangle{
			X:      x,
			Y:      y,
			Width:  paneWidth,
			Height: paneHeight,
		}

		// Move to next position
		currentX += paneWidth + marginWidth + paddingWidth + config.MainAxisGap
	}

	return results, nil
}

// ValidateConstraints validates a set of layout constraints
func ValidateConstraints(constraints LayoutConstraints) error {
	if constraints.MinWidth < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "minimum width cannot be negative", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	if constraints.MinHeight < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "minimum height cannot be negative", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	if constraints.MaxWidth > 0 && constraints.MaxWidth < constraints.MinWidth {
		return gerror.New(gerror.ErrCodeInvalidInput, "maximum width cannot be less than minimum width", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	if constraints.MaxHeight > 0 && constraints.MaxHeight < constraints.MinHeight {
		return gerror.New(gerror.ErrCodeInvalidInput, "maximum height cannot be less than minimum height", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	if constraints.FlexGrow < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "flex grow cannot be negative", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	if constraints.FlexShrink < 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "flex shrink cannot be negative", nil).
			WithComponent("layout.constraints").
			WithOperation("ValidateConstraints")
	}

	return nil
}
