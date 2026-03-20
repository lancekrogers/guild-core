// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dashboard

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/cost"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// DataVisualizer creates visualizations for cost data
type DataVisualizer struct {
	config *VisualizerConfig
}

// VisualizerConfig contains visualizer configuration
type VisualizerConfig struct {
	DefaultWidth  int    `json:"default_width"`
	DefaultHeight int    `json:"default_height"`
	ColorScheme   string `json:"color_scheme"`
	AnimationFPS  int    `json:"animation_fps"`
}

// NewDataVisualizer creates a new data visualizer
func NewDataVisualizer(ctx context.Context) (*DataVisualizer, error) {
	ctx = observability.WithComponent(ctx, "cost.visualizer")
	ctx = observability.WithOperation(ctx, "NewDataVisualizer")

	config := &VisualizerConfig{
		DefaultWidth:  80,
		DefaultHeight: 20,
		ColorScheme:   "default",
		AnimationFPS:  30,
	}

	return &DataVisualizer{
		config: config,
	}, nil
}

// GenerateTrend creates a trend visualization
func (dv *DataVisualizer) GenerateTrend(historical *HistoricalData, interval, window string) Visualization {
	return Visualization{
		Type:  VisualizationTrend,
		Title: "Cost Trend",
		Data:  historical.DataPoints,
		Config: map[string]interface{}{
			"interval": interval,
			"window":   window,
			"width":    dv.config.DefaultWidth,
			"height":   dv.config.DefaultHeight,
		},
		GeneratedAt: time.Now(),
	}
}

// GeneratePieChart creates a pie chart visualization
func (dv *DataVisualizer) GeneratePieChart(data map[string]float64, title string) Visualization {
	// Convert map to sorted slices for consistent ordering
	type pieSlice struct {
		Label      string  `json:"label"`
		Value      float64 `json:"value"`
		Percentage float64 `json:"percentage"`
	}

	total := 0.0
	for _, value := range data {
		total += value
	}

	var slices []pieSlice
	for label, value := range data {
		percentage := (value / total) * 100
		slices = append(slices, pieSlice{
			Label:      label,
			Value:      value,
			Percentage: percentage,
		})
	}

	// Sort by value (largest first)
	sort.Slice(slices, func(i, j int) bool {
		return slices[i].Value > slices[j].Value
	})

	return Visualization{
		Type:  VisualizationPieChart,
		Title: title,
		Data:  slices,
		Config: map[string]interface{}{
			"total":       total,
			"show_labels": true,
			"show_values": true,
		},
		GeneratedAt: time.Now(),
	}
}

// GenerateBarChart creates a bar chart visualization
func (dv *DataVisualizer) GenerateBarChart(data map[string]float64, title string) Visualization {
	// Convert map to sorted slices
	type barData struct {
		Label string  `json:"label"`
		Value float64 `json:"value"`
	}

	var bars []barData
	maxValue := 0.0
	for label, value := range data {
		bars = append(bars, barData{
			Label: label,
			Value: value,
		})
		if value > maxValue {
			maxValue = value
		}
	}

	// Sort by value (largest first)
	sort.Slice(bars, func(i, j int) bool {
		return bars[i].Value > bars[j].Value
	})

	return Visualization{
		Type:  VisualizationBarChart,
		Title: title,
		Data:  bars,
		Config: map[string]interface{}{
			"max_value":   maxValue,
			"horizontal":  true,
			"show_values": true,
			"bar_width":   20,
		},
		GeneratedAt: time.Now(),
	}
}

// GenerateSavingsChart creates a savings opportunities visualization
func (dv *DataVisualizer) GenerateSavingsChart(optimizations []cost.Optimization) Visualization {
	// Convert optimizations to savings data
	type savingsData struct {
		Description string  `json:"description"`
		Savings     float64 `json:"savings"`
		Type        string  `json:"type"`
		Priority    int     `json:"priority"`
		Confidence  float64 `json:"confidence"`
	}

	var savings []savingsData
	totalPotentialSavings := 0.0

	for _, opt := range optimizations {
		savings = append(savings, savingsData{
			Description: opt.Description,
			Savings:     opt.Savings,
			Type:        string(opt.Type),
			Priority:    opt.Priority,
			Confidence:  opt.Confidence,
		})
		totalPotentialSavings += opt.Savings
	}

	return Visualization{
		Type:  VisualizationSavings,
		Title: "Cost Optimization Opportunities",
		Data:  savings,
		Config: map[string]interface{}{
			"total_savings": totalPotentialSavings,
			"max_items":     10,
			"sort_by":       "savings",
		},
		GeneratedAt: time.Now(),
	}
}

// ASCIIChart provides ASCII chart generation capabilities
type ASCIIChart struct {
	Width  int
	Height int
	Title  string
}

// Plot generates an ASCII line chart
func (chart *ASCIIChart) Plot(values []float64) string {
	if len(values) == 0 {
		return "No data available"
	}

	// Find min and max values
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Avoid division by zero
	if maxVal == minVal {
		maxVal = minVal + 1
	}

	// Create the chart grid
	grid := make([][]rune, chart.Height)
	for i := range grid {
		grid[i] = make([]rune, chart.Width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot the line
	for i, value := range values {
		if i >= chart.Width {
			break
		}

		// Normalize value to chart height
		normalized := (value - minVal) / (maxVal - minVal)
		y := int((1.0 - normalized) * float64(chart.Height-1))

		// Ensure y is within bounds
		if y < 0 {
			y = 0
		} else if y >= chart.Height {
			y = chart.Height - 1
		}

		// Plot point
		if i == 0 {
			grid[y][i] = '╭'
		} else if i == len(values)-1 || i == chart.Width-1 {
			grid[y][i] = '╮'
		} else {
			grid[y][i] = '─'
		}

		// Connect to previous point if not first
		if i > 0 && i < len(values) {
			prevValue := values[i-1]
			prevNormalized := (prevValue - minVal) / (maxVal - minVal)
			prevY := int((1.0 - prevNormalized) * float64(chart.Height-1))

			// Ensure prevY is within bounds
			if prevY < 0 {
				prevY = 0
			} else if prevY >= chart.Height {
				prevY = chart.Height - 1
			}

			// Draw vertical connection if needed
			startY, endY := prevY, y
			if startY > endY {
				startY, endY = endY, startY
			}

			for connY := startY; connY <= endY; connY++ {
				if grid[connY][i] == ' ' {
					if connY == startY || connY == endY {
						grid[connY][i] = '│'
					} else {
						grid[connY][i] = '│'
					}
				}
			}
		}
	}

	// Convert grid to string
	var result []string
	for _, row := range grid {
		result = append(result, string(row))
	}

	return joinStrings(result, "\n")
}

// PlotBarChart generates an ASCII bar chart
func (chart *ASCIIChart) PlotBarChart(data map[string]float64) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Convert to sorted slices
	type barItem struct {
		label string
		value float64
	}

	var items []barItem
	maxValue := 0.0
	for label, value := range data {
		items = append(items, barItem{label, value})
		if value > maxValue {
			maxValue = value
		}
	}

	// Sort by value (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].value > items[j].value
	})

	// Limit to chart height
	if len(items) > chart.Height {
		items = items[:chart.Height]
	}

	// Generate bars
	var result []string
	maxLabelWidth := 0
	for _, item := range items {
		if len(item.label) > maxLabelWidth {
			maxLabelWidth = len(item.label)
		}
	}

	// Ensure label width doesn't exceed reasonable limit
	if maxLabelWidth > 15 {
		maxLabelWidth = 15
	}

	for _, item := range items {
		// Truncate label if too long
		label := item.label
		if len(label) > maxLabelWidth {
			label = label[:maxLabelWidth-3] + "..."
		}

		// Calculate bar length
		barLength := int((item.value / maxValue) * float64(chart.Width-maxLabelWidth-10))
		if barLength < 0 {
			barLength = 0
		} else if barLength > chart.Width-maxLabelWidth-10 {
			barLength = chart.Width - maxLabelWidth - 10
		}

		// Create bar
		bar := repeatString("█", barLength)

		// Format line
		line := padRight(label, maxLabelWidth) + " " + bar + " $" + formatFloat(item.value, 2)
		result = append(result, line)
	}

	return joinStrings(result, "\n")
}

// Utility functions

// joinStrings joins string slice with separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	n := len(sep) * (len(strs) - 1)
	for i := 0; i < len(strs); i++ {
		n += len(strs[i])
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(strs[0])
	for _, s := range strs[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) * n)
	for i := 0; i < n; i++ {
		b.WriteString(s)
	}
	return b.String()
}

// padRight pads string to width with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + repeatString(" ", width-len(s))
}

// formatFloat formats float to specified decimal places
func formatFloat(f float64, decimals int) string {
	multiplier := math.Pow(10, float64(decimals))
	return fmt.Sprintf("%."+fmt.Sprintf("%d", decimals)+"f", math.Round(f*multiplier)/multiplier)
}
