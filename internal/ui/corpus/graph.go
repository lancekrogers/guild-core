// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/lancekrogers/guild/pkg/corpus"
)

// GraphView handles rendering the document relationship graph
type GraphView struct {
	graph       corpus.Graph
	width       int
	height      int
	nodes       []graphNode
	cursorX     int
	cursorY     int
	selectedIdx int
}

// graphNode represents a node in the graph visualization
type graphNode struct {
	doc   corpus.CorpusDoc
	x     int
	y     int
	edges []int // indices of connected nodes
}

// NewGraphView creates a new graph visualization
func NewGraphView(graph corpus.Graph, width, height int) *GraphView {
	return &GraphView{
		graph:  graph,
		width:  width,
		height: height,
	}
}

// Layout positions the nodes in the graph for visualization
func (g *GraphView) Layout() {
	if len(g.graph.Nodes) == 0 {
		return
	}

	// Clear existing layout
	g.nodes = make([]graphNode, 0, len(g.graph.Nodes))

	// Simple force-directed layout algorithm
	// For a small number of nodes, we'll use a circular layout
	if len(g.graph.Nodes) <= 8 {
		g.layoutCircular()
	} else {
		g.layoutForceDirected()
	}
}

// layoutCircular arranges nodes in a circle
func (g *GraphView) layoutCircular() {
	center := struct{ x, y int }{g.width / 2, g.height / 2}
	radius := int(math.Min(float64(g.width), float64(g.height)) * 0.4)

	// Convert map keys to a slice for indexing
	nodeNames := make([]string, 0, len(g.graph.Nodes))
	for name := range g.graph.Nodes {
		nodeNames = append(nodeNames, name)
	}

	for i, nodeName := range nodeNames {
		// Calculate position on circle
		angle := 2 * math.Pi * float64(i) / float64(len(nodeNames))
		x := center.x + int(float64(radius)*math.Cos(angle))
		y := center.y + int(float64(radius)*math.Sin(angle))

		// Create graph node (using empty CorpusDoc with just the title)
		gNode := graphNode{
			doc: corpus.CorpusDoc{Title: nodeName},
			x:   x,
			y:   y / 2, // Compensate for terminal aspect ratio
		}

		// Add edges
		for _, edge := range g.graph.Edges {
			if edge.From == nodeName {
				// Find the target node index
				for j, targetName := range nodeNames {
					if targetName == edge.To {
						gNode.edges = append(gNode.edges, j)
						break
					}
				}
			}
		}

		g.nodes = append(g.nodes, gNode)
	}
}

// layoutForceDirected implements a simple force-directed layout algorithm
func (g *GraphView) layoutForceDirected() {
	// Convert map keys to a slice for indexing
	nodeNames := make([]string, 0, len(g.graph.Nodes))
	for name := range g.graph.Nodes {
		nodeNames = append(nodeNames, name)
	}

	// Initialize with random positions
	for _, nodeName := range nodeNames {
		// Generate a pseudorandom position based on title hash
		hash := 0
		for _, c := range nodeName {
			hash = (hash*31 + int(c)) % 1000
		}

		x := (hash % g.width) * 8 / 10
		y := ((hash / g.width) % g.height) * 8 / 10

		gNode := graphNode{
			doc: corpus.CorpusDoc{Title: nodeName},
			x:   x,
			y:   y / 2, // Compensate for terminal aspect ratio
		}

		// Add edges
		for _, edge := range g.graph.Edges {
			if edge.From == nodeName {
				// Find the target node index
				for j, targetName := range nodeNames {
					if targetName == edge.To {
						gNode.edges = append(gNode.edges, j)
						break
					}
				}
			}
		}

		g.nodes = append(g.nodes, gNode)
	}

	// Run several iterations of force-directed placement
	iterations := 100
	for i := 0; i < iterations; i++ {
		g.applyForces()
	}
}

// applyForces applies spring and repulsive forces between nodes
func (g *GraphView) applyForces() {
	// Force parameters
	k := float64(g.width) / math.Sqrt(float64(len(g.nodes))) // optimal distance

	// Calculate repulsive forces between all nodes
	forces := make([]struct{ x, y float64 }, len(g.nodes))

	// Repulsive forces between all nodes
	for i := range g.nodes {
		for j := range g.nodes {
			if i == j {
				continue
			}

			// Calculate distance
			dx := float64(g.nodes[i].x - g.nodes[j].x)
			dy := float64(g.nodes[i].y-g.nodes[j].y) * 2 // Compensate for terminal aspect ratio
			dist := math.Max(1.0, math.Sqrt(dx*dx+dy*dy))

			// Repulsive force is inversely proportional to distance
			force := k * k / dist
			forces[i].x += (dx / dist) * force
			forces[i].y += (dy / dist) * force
		}
	}

	// Attractive forces along edges
	for i, node := range g.nodes {
		for _, j := range node.edges {
			// Calculate distance
			dx := float64(g.nodes[i].x - g.nodes[j].x)
			dy := float64(g.nodes[i].y-g.nodes[j].y) * 2 // Compensate for terminal aspect ratio
			dist := math.Max(1.0, math.Sqrt(dx*dx+dy*dy))

			// Attractive force is proportional to distance
			force := dist * dist / k
			forces[i].x -= (dx / dist) * force
			forces[i].y -= (dy / dist) * force
		}
	}

	// Apply forces with dampening
	dampening := 0.1
	for i := range g.nodes {
		// Apply force with dampening
		g.nodes[i].x += int(forces[i].x * dampening)
		g.nodes[i].y += int(forces[i].y * dampening)

		// Keep within bounds
		g.nodes[i].x = clamp(g.nodes[i].x, 2, g.width-3)
		g.nodes[i].y = clamp(g.nodes[i].y, 1, g.height-2)
	}
}

// clamp keeps a value within min and max
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Render draws the graph
func (g *GraphView) Render() string {
	if len(g.nodes) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No documents to visualize in graph")
	}

	// We'll use a simple ASCII-based approach for the graph visualization
	// This can be enhanced with more sophisticated rendering in the future

	// Create a 2D grid for the canvas
	canvas := make([][]string, g.height)
	for i := range canvas {
		canvas[i] = make([]string, g.width)
		for j := range canvas[i] {
			canvas[i][j] = " "
		}
	}

	// Draw edges first so they appear behind nodes
	for i, node := range g.nodes {
		for _, edgeIdx := range node.edges {
			target := g.nodes[edgeIdx]

			// Draw a simple line from node to target
			// This is a basic implementation - a better line algorithm would be nicer
			drawLine(canvas, node.x, node.y, target.x, target.y, "·")
		}

		// Highlight edges from selected node with a different character
		if i == g.selectedIdx {
			for _, edgeIdx := range node.edges {
				target := g.nodes[edgeIdx]
				drawLine(canvas, node.x, node.y, target.x, target.y, "•")
			}
		}
	}

	// Draw nodes over the edges
	for i, node := range g.nodes {
		// Truncate node title to fit
		title := node.doc.Title
		if len(title) > 20 {
			title = truncate.StringWithTail(title, 20, "…")
		}

		// Draw the node
		nodeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Padding(0, 1).
			Bold(true)

		// If this is the selected node, highlight it
		if i == g.selectedIdx {
			nodeStyle = nodeStyle.Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("6"))
		}

		// Draw title at node position
		drawString(canvas, node.x, node.y, nodeStyle.Render(title))

		// For tagged nodes, show a small indicator
		if len(node.doc.Tags) > 0 {
			indicator := fmt.Sprintf("(%d)", len(node.doc.Tags))
			drawString(canvas, node.x+len(title)+2, node.y,
				lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(indicator))
		}
	}

	// Convert canvas to string
	var builder strings.Builder
	for _, row := range canvas {
		for _, cell := range row {
			builder.WriteString(cell)
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// drawLine places characters to form a line between two points
func drawLine(canvas [][]string, x1, y1, x2, y2 int, char string) {
	// Simple line drawing algorithm (Bresenham's algorithm)
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		// Only draw if within canvas bounds
		if y1 >= 0 && y1 < len(canvas) && x1 >= 0 && x1 < len(canvas[0]) {
			// Only draw if the cell is empty or already has our line character
			if canvas[y1][x1] == " " || canvas[y1][x1] == char {
				canvas[y1][x1] = char
			}
		}

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawString places a string on the canvas at given position
func drawString(canvas [][]string, x, y int, s string) {
	// Check if y is within bounds
	if y < 0 || y >= len(canvas) {
		return
	}

	// Handle ANSI escape sequences properly
	inEscapeSeq := false
	escapeBuffer := ""

	for i, char := range s {
		// Check if we're in an escape sequence
		if char == '\x1b' {
			inEscapeSeq = true
			escapeBuffer = string(char)
			continue
		}

		if inEscapeSeq {
			escapeBuffer += string(char)
			// Check if this is the end of the escape sequence
			if char == 'm' {
				inEscapeSeq = false
				// Apply escape sequence to all cells it will affect
				for j := x + i; j < x+len(s); j++ {
					if j >= 0 && j < len(canvas[y]) {
						canvas[y][j] = escapeBuffer + canvas[y][j]
					}
				}
			}
			continue
		}

		// Regular character, place it on the canvas
		pos := x + i
		if pos >= 0 && pos < len(canvas[y]) {
			canvas[y][pos] = string(char)
		}
	}
}

// MoveUp moves the cursor up in the graph
func (g *GraphView) MoveUp() {
	// Find the nearest node above the current selection
	if len(g.nodes) == 0 {
		return
	}

	currY := g.nodes[g.selectedIdx].y
	best := g.selectedIdx
	bestDist := g.height

	for i, node := range g.nodes {
		if node.y < currY {
			dist := currY - node.y
			if dist < bestDist {
				bestDist = dist
				best = i
			}
		}
	}

	g.selectedIdx = best
}

// MoveDown moves the cursor down in the graph
func (g *GraphView) MoveDown() {
	// Find the nearest node below the current selection
	if len(g.nodes) == 0 {
		return
	}

	currY := g.nodes[g.selectedIdx].y
	best := g.selectedIdx
	bestDist := g.height

	for i, node := range g.nodes {
		if node.y > currY {
			dist := node.y - currY
			if dist < bestDist {
				bestDist = dist
				best = i
			}
		}
	}

	g.selectedIdx = best
}

// MoveLeft moves the cursor left in the graph
func (g *GraphView) MoveLeft() {
	// Find the nearest node to the left of the current selection
	if len(g.nodes) == 0 {
		return
	}

	currX := g.nodes[g.selectedIdx].x
	best := g.selectedIdx
	bestDist := g.width

	for i, node := range g.nodes {
		if node.x < currX {
			dist := currX - node.x
			if dist < bestDist {
				bestDist = dist
				best = i
			}
		}
	}

	g.selectedIdx = best
}

// MoveRight moves the cursor right in the graph
func (g *GraphView) MoveRight() {
	// Find the nearest node to the right of the current selection
	if len(g.nodes) == 0 {
		return
	}

	currX := g.nodes[g.selectedIdx].x
	best := g.selectedIdx
	bestDist := g.width

	for i, node := range g.nodes {
		if node.x > currX {
			dist := node.x - currX
			if dist < bestDist {
				bestDist = dist
				best = i
			}
		}
	}

	g.selectedIdx = best
}

// GetSelectedDoc returns the currently selected document
func (g *GraphView) GetSelectedDoc() *corpus.CorpusDoc {
	if len(g.nodes) == 0 || g.selectedIdx < 0 || g.selectedIdx >= len(g.nodes) {
		return nil
	}

	// Return a copy of the document
	doc := g.nodes[g.selectedIdx].doc
	return &doc
}

// absolute value function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
