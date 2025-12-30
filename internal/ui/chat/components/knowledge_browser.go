// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// ViewMode represents different viewing modes for the knowledge browser
type ViewMode int

const (
	ViewModeGraph ViewMode = iota
	ViewModeList
	ViewModeDetail
	ViewModeSearch
)

// KnowledgeBrowser provides visual navigation of the knowledge graph
type KnowledgeBrowser struct {
	ctx         context.Context
	graph       *KnowledgeGraph
	currentNode *KnowledgeNode
	neighbors   []*KnowledgeNode
	history     []string
	viewMode    ViewMode
	width       int
	height      int
	searchQuery string
	searchMode  bool
}

// KnowledgeGraph represents the knowledge graph structure
type KnowledgeGraph struct {
	Nodes map[string]*KnowledgeNode `json:"nodes"`
	Edges map[string]*KnowledgeEdge `json:"edges"`
	Stats KnowledgeGraphStats       `json:"stats"`
}

// KnowledgeNode represents a knowledge node in the graph
type KnowledgeNode struct {
	ID          string                 `json:"id"`
	Type        KnowledgeNodeType      `json:"type"`
	Content     interface{}            `json:"content"`
	Properties  map[string]interface{} `json:"properties"`
	Connections []string               `json:"connections"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Confidence  float64                `json:"confidence"`
	Source      string                 `json:"source"`
}

// KnowledgeEdge represents a connection between nodes
type KnowledgeEdge struct {
	ID         string                 `json:"id"`
	FromNode   string                 `json:"from_node"`
	ToNode     string                 `json:"to_node"`
	Type       KnowledgeEdgeType      `json:"type"`
	Weight     float64                `json:"weight"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	Confidence float64                `json:"confidence"`
}

// KnowledgeNodeType represents different types of knowledge nodes
type KnowledgeNodeType string

const (
	NodeTypePattern   KnowledgeNodeType = "pattern"
	NodeTypeDecision  KnowledgeNodeType = "decision"
	NodeTypeTip       KnowledgeNodeType = "tip"
	NodeTypeReference KnowledgeNodeType = "reference"
	NodeTypeExample   KnowledgeNodeType = "example"
	NodeTypeLesson    KnowledgeNodeType = "lesson"
	NodeTypeConcept   KnowledgeNodeType = "concept"
	NodeTypePerson    KnowledgeNodeType = "person"
	NodeTypeProject   KnowledgeNodeType = "project"
)

// KnowledgeEdgeType represents different types of connections
type KnowledgeEdgeType string

const (
	EdgeTypeRelated    KnowledgeEdgeType = "related"
	EdgeTypeDependsOn  KnowledgeEdgeType = "depends_on"
	EdgeTypeImplements KnowledgeEdgeType = "implements"
	EdgeTypeConflicts  KnowledgeEdgeType = "conflicts"
	EdgeTypeSupersedes KnowledgeEdgeType = "supersedes"
	EdgeTypeSimilar    KnowledgeEdgeType = "similar"
	EdgeTypeExample    KnowledgeEdgeType = "example"
	EdgeTypeAuthor     KnowledgeEdgeType = "authored_by"
	EdgeTypeUsedIn     KnowledgeEdgeType = "used_in"
)

// KnowledgeGraphStats contains graph statistics
type KnowledgeGraphStats struct {
	NodeCount          int                       `json:"node_count"`
	EdgeCount          int                       `json:"edge_count"`
	NodesByType        map[KnowledgeNodeType]int `json:"nodes_by_type"`
	EdgesByType        map[KnowledgeEdgeType]int `json:"edges_by_type"`
	AverageConnections float64                   `json:"average_connections"`
	MostConnectedNode  string                    `json:"most_connected_node"`
	LastUpdated        time.Time                 `json:"last_updated"`
}

// KnowledgeNavigationMsg represents navigation messages
type KnowledgeNavigationMsg struct {
	NodeID string
	Action string
}

// NewKnowledgeBrowser creates a new knowledge browser
func NewKnowledgeBrowser(ctx context.Context) *KnowledgeBrowser {
	ctx = observability.WithComponent(ctx, "knowledge_browser")

	return &KnowledgeBrowser{
		ctx:      ctx,
		history:  make([]string, 0),
		viewMode: ViewModeGraph,
		width:    80,
		height:   24,
	}
}

// SetSize sets the browser dimensions
func (kb *KnowledgeBrowser) SetSize(width, height int) {
	kb.width = width
	kb.height = height
}

// SetGraph sets the knowledge graph
func (kb *KnowledgeBrowser) SetGraph(graph *KnowledgeGraph) {
	kb.graph = graph

	// Set initial node to most connected
	if graph != nil && graph.Stats.MostConnectedNode != "" {
		kb.navigateToNode(graph.Stats.MostConnectedNode)
	}
}

// View renders the knowledge browser
func (kb *KnowledgeBrowser) View() string {
	_ = observability.WithOperation(kb.ctx, "View")

	if kb.graph == nil {
		return kb.renderEmptyState()
	}

	switch kb.viewMode {
	case ViewModeGraph:
		return kb.renderGraphView()
	case ViewModeList:
		return kb.renderListView()
	case ViewModeDetail:
		return kb.renderDetailView()
	case ViewModeSearch:
		return kb.renderSearchView()
	}

	return ""
}

// Update handles input events
func (kb *KnowledgeBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := observability.WithOperation(kb.ctx, "Update")

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return kb.handleKeyMsg(ctx, msg)
	case KnowledgeNavigationMsg:
		return kb.handleNavigation(ctx, msg)
	case tea.WindowSizeMsg:
		kb.SetSize(msg.Width, msg.Height)
	}

	return kb, nil
}

// Init initializes the knowledge browser
func (kb *KnowledgeBrowser) Init() tea.Cmd {
	return nil
}

// handleKeyMsg processes keyboard input
func (kb *KnowledgeBrowser) handleKeyMsg(ctx context.Context, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if kb.searchMode {
		return kb.handleSearchMode(ctx, msg)
	}

	switch msg.String() {
	case "q", "esc":
		return kb, tea.Quit

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Navigate to neighbor by number
		idx, _ := strconv.Atoi(msg.String())
		if idx > 0 && idx <= len(kb.neighbors) {
			return kb, kb.navigateToNode(kb.neighbors[idx-1].ID)
		}

	case "b", "backspace":
		// Go back in history
		if len(kb.history) > 1 {
			kb.history = kb.history[:len(kb.history)-1]
			return kb, kb.navigateToNode(kb.history[len(kb.history)-1])
		}

	case "v":
		// Cycle view modes
		kb.viewMode = (kb.viewMode + 1) % 4

	case "/":
		// Enter search mode
		kb.searchMode = true
		kb.viewMode = ViewModeSearch

	case "g":
		// Go to graph view
		kb.viewMode = ViewModeGraph

	case "l":
		// Go to list view
		kb.viewMode = ViewModeList

	case "d":
		// Go to detail view
		kb.viewMode = ViewModeDetail

	case "r":
		// Refresh/reload graph
		return kb, kb.reloadGraph()

	case "s":
		// Show statistics
		return kb, kb.showStatistics()

	case "h", "?":
		// Show help
		return kb, kb.showHelp()
	}

	return kb, nil
}

// handleSearchMode processes input in search mode
func (kb *KnowledgeBrowser) handleSearchMode(ctx context.Context, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		kb.searchMode = false
		if kb.searchQuery != "" {
			return kb, kb.performSearch(kb.searchQuery)
		}

	case "esc":
		kb.searchMode = false
		kb.viewMode = ViewModeGraph

	case "backspace":
		if len(kb.searchQuery) > 0 {
			kb.searchQuery = kb.searchQuery[:len(kb.searchQuery)-1]
		}

	default:
		// Add character to search query
		if len(msg.String()) == 1 {
			kb.searchQuery += msg.String()
		}
	}

	return kb, nil
}

// handleNavigation processes navigation messages
func (kb *KnowledgeBrowser) handleNavigation(ctx context.Context, msg KnowledgeNavigationMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case "navigate":
		return kb, kb.navigateToNode(msg.NodeID)
	case "select":
		kb.currentNode = kb.graph.Nodes[msg.NodeID]
		kb.updateNeighbors()
	}

	return kb, nil
}

// navigateToNode navigates to a specific node
func (kb *KnowledgeBrowser) navigateToNode(nodeID string) tea.Cmd {
	return func() tea.Msg {
		if kb.graph == nil || kb.graph.Nodes[nodeID] == nil {
			return nil
		}

		// Add to history if not already the current node
		if kb.currentNode == nil || kb.currentNode.ID != nodeID {
			kb.history = append(kb.history, nodeID)
		}

		kb.currentNode = kb.graph.Nodes[nodeID]
		kb.updateNeighbors()

		return KnowledgeNavigationMsg{
			NodeID: nodeID,
			Action: "select",
		}
	}
}

// updateNeighbors updates the list of connected neighbors
func (kb *KnowledgeBrowser) updateNeighbors() {
	if kb.currentNode == nil || kb.graph == nil {
		kb.neighbors = nil
		return
	}

	neighbors := make([]*KnowledgeNode, 0)

	// Find connected nodes
	for _, connection := range kb.currentNode.Connections {
		if node := kb.graph.Nodes[connection]; node != nil {
			neighbors = append(neighbors, node)
		}
	}

	// Sort by connection strength or relevance
	// For now, sort by node type and then by name
	// TODO: Implement proper relevance scoring

	kb.neighbors = neighbors
}

// Rendering methods

// renderEmptyState renders the empty state
func (kb *KnowledgeBrowser) renderEmptyState() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		Padding(2).
		Width(kb.width)

	content := `🧠 Knowledge Graph Browser

No knowledge graph data available.

**Getting Started:**
The knowledge graph is built from your corpus documents and chat interactions.

• Add documents to your corpus using '/corpus add'
• Chat with agents to generate knowledge
• Use '/index rebuild' to process documents into knowledge nodes

**What you'll see here:**
• Visual navigation of knowledge connections
• Related concepts and patterns
• Decision trees and dependencies  
• Learning paths and examples

The knowledge graph will grow as you use Guild!`

	return style.Render(content)
}

// renderGraphView renders the graph visualization
func (kb *KnowledgeBrowser) renderGraphView() string {
	var b strings.Builder

	// Header
	b.WriteString(kb.renderHeader())
	b.WriteString("\n\n")

	if kb.currentNode == nil {
		b.WriteString(kb.renderNoCurrentNode())
		return b.String()
	}

	// Current node (center)
	center := kb.renderNode(kb.currentNode, true)
	b.WriteString(center)
	b.WriteString("\n\n")

	// Connected nodes
	if len(kb.neighbors) > 0 {
		b.WriteString("🔗 **Connected Knowledge:**\n\n")
		for i, neighbor := range kb.neighbors {
			if i >= 9 { // Limit to 9 connections for keyboard navigation
				remaining := len(kb.neighbors) - 9
				b.WriteString(fmt.Sprintf("   ... and %d more connections\n", remaining))
				break
			}

			edge := kb.getEdge(kb.currentNode.ID, neighbor.ID)

			prefix := fmt.Sprintf("  %d. ", i+1)
			edgeType := kb.formatEdgeType(edge)

			node := kb.renderNode(neighbor, false)

			line := fmt.Sprintf("%s%s → %s",
				prefix, edgeType, node)
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("No connections found.\n")
	}

	// Navigation breadcrumb
	if len(kb.history) > 1 {
		b.WriteString("\n")
		b.WriteString(kb.renderBreadcrumb())
	}

	// Help
	b.WriteString("\n")
	b.WriteString(kb.renderGraphHelp())

	return b.String()
}

// renderListView renders the list view of all nodes
func (kb *KnowledgeBrowser) renderListView() string {
	var b strings.Builder

	b.WriteString(kb.renderHeader())
	b.WriteString("\n\n")

	b.WriteString("📋 **All Knowledge Nodes**\n\n")

	if kb.graph == nil || len(kb.graph.Nodes) == 0 {
		b.WriteString("No knowledge nodes available.\n")
		return b.String()
	}

	// Group by type
	nodesByType := make(map[KnowledgeNodeType][]*KnowledgeNode)
	for _, node := range kb.graph.Nodes {
		nodesByType[node.Type] = append(nodesByType[node.Type], node)
	}

	// Display by type
	for nodeType, nodes := range nodesByType {
		if len(nodes) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("## %s (%d)\n\n", kb.getNodeTypeDisplayName(nodeType), len(nodes)))

		for _, node := range nodes {
			icon := kb.getNodeTypeIcon(node.Type)
			preview := kb.getNodePreview(node)
			connections := len(node.Connections)

			b.WriteString(fmt.Sprintf("- %s **%s** (%d connections)\n",
				icon, preview, connections))
			b.WriteString(fmt.Sprintf("  _Updated: %s | Confidence: %.0f%%_\n\n",
				node.UpdatedAt.Format("Jan 2, 2006"), node.Confidence*100))
		}
	}

	b.WriteString(kb.renderListHelp())

	return b.String()
}

// renderDetailView renders detailed view of current node
func (kb *KnowledgeBrowser) renderDetailView() string {
	var b strings.Builder

	b.WriteString(kb.renderHeader())
	b.WriteString("\n\n")

	if kb.currentNode == nil {
		b.WriteString(kb.renderNoCurrentNode())
		return b.String()
	}

	node := kb.currentNode

	// Node header
	icon := kb.getNodeTypeIcon(node.Type)
	b.WriteString(fmt.Sprintf("%s **%s**\n", icon, kb.getNodePreview(node)))
	b.WriteString(fmt.Sprintf("_Type: %s | ID: %s_\n\n",
		kb.getNodeTypeDisplayName(node.Type), node.ID))

	// Content
	b.WriteString("## Content\n\n")
	content := kb.formatNodeContent(node.Content)
	b.WriteString(content)
	b.WriteString("\n\n")

	// Properties
	if len(node.Properties) > 0 {
		b.WriteString("## Properties\n\n")
		for key, value := range node.Properties {
			b.WriteString(fmt.Sprintf("- **%s:** %v\n", key, value))
		}
		b.WriteString("\n")
	}

	// Metadata
	b.WriteString("## Metadata\n\n")
	b.WriteString(fmt.Sprintf("- **Created:** %s\n", node.CreatedAt.Format("January 2, 2006 at 15:04")))
	b.WriteString(fmt.Sprintf("- **Updated:** %s\n", node.UpdatedAt.Format("January 2, 2006 at 15:04")))
	b.WriteString(fmt.Sprintf("- **Source:** %s\n", node.Source))
	b.WriteString(fmt.Sprintf("- **Confidence:** %.0f%%\n", node.Confidence*100))
	b.WriteString(fmt.Sprintf("- **Connections:** %d\n\n", len(node.Connections)))

	// Connections detail
	if len(kb.neighbors) > 0 {
		b.WriteString("## Connections\n\n")
		for i, neighbor := range kb.neighbors {
			edge := kb.getEdge(node.ID, neighbor.ID)
			b.WriteString(fmt.Sprintf("%d. %s **%s** %s\n",
				i+1,
				kb.getNodeTypeIcon(neighbor.Type),
				kb.getNodePreview(neighbor),
				kb.formatEdgeType(edge)))
		}
		b.WriteString("\n")
	}

	b.WriteString(kb.renderDetailHelp())

	return b.String()
}

// renderSearchView renders the search interface
func (kb *KnowledgeBrowser) renderSearchView() string {
	var b strings.Builder

	b.WriteString(kb.renderHeader())
	b.WriteString("\n\n")

	// Search box
	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(0, 1).
		Width(kb.width - 4)

	query := kb.searchQuery
	if kb.searchMode {
		query += "█" // Cursor
	}

	if query == "" {
		query = "Type to search knowledge..."
		searchStyle = searchStyle.Foreground(lipgloss.Color("240"))
	}

	b.WriteString(searchStyle.Render("Search: " + query))
	b.WriteString("\n\n")

	// Search results would go here
	if kb.searchQuery != "" {
		b.WriteString("🔍 **Search Results** (placeholder)\n\n")
		b.WriteString("Search functionality is under development.\n")
		b.WriteString("Will search through node content, properties, and connections.\n\n")
	}

	b.WriteString(kb.renderSearchHelp())

	return b.String()
}

// renderNode renders a single knowledge node
func (kb *KnowledgeBrowser) renderNode(node *KnowledgeNode, detailed bool) string {
	if detailed {
		style := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(1).
			Width(kb.width - 4)

		var content strings.Builder

		// Type and ID
		icon := kb.getNodeTypeIcon(node.Type)
		header := fmt.Sprintf("%s %s", icon, kb.getNodePreview(node))
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
		content.WriteString(titleStyle.Render(header))
		content.WriteString("\n\n")

		// Content preview
		nodeContent := kb.formatNodeContent(node.Content)
		if len(nodeContent) > 200 {
			nodeContent = nodeContent[:200] + "..."
		}
		content.WriteString(nodeContent)
		content.WriteString("\n\n")

		// Quick stats
		content.WriteString(fmt.Sprintf("**Type:** %s | **Connections:** %d | **Confidence:** %.0f%%",
			kb.getNodeTypeDisplayName(node.Type),
			len(node.Connections),
			node.Confidence*100))

		return style.Render(content.String())
	}

	// Simple view
	icon := kb.getNodeTypeIcon(node.Type)
	preview := kb.getNodePreview(node)
	if len(preview) > 40 {
		preview = preview[:40] + "..."
	}
	return fmt.Sprintf("%s %s", icon, preview)
}

// Helper methods

// renderHeader renders the browser header
func (kb *KnowledgeBrowser) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69"))

	title := "🧠 Knowledge Browser"

	// Add view mode indicator
	modeNames := []string{"Graph", "List", "Detail", "Search"}
	if int(kb.viewMode) < len(modeNames) {
		title += fmt.Sprintf(" (%s)", modeNames[kb.viewMode])
	}

	// Add graph stats
	if kb.graph != nil {
		title += fmt.Sprintf(" - %d nodes, %d connections",
			kb.graph.Stats.NodeCount, kb.graph.Stats.EdgeCount)
	}

	return style.Render(title)
}

// renderNoCurrentNode renders message when no node is selected
func (kb *KnowledgeBrowser) renderNoCurrentNode() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	return style.Render("No node selected. Use [l] for list view to browse all nodes.")
}

// renderBreadcrumb renders navigation breadcrumb
func (kb *KnowledgeBrowser) renderBreadcrumb() string {
	if len(kb.history) <= 1 {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	path := make([]string, 0, len(kb.history))
	for _, nodeID := range kb.history {
		if node := kb.graph.Nodes[nodeID]; node != nil {
			preview := kb.getNodePreview(node)
			if len(preview) > 20 {
				preview = preview[:20] + "..."
			}
			path = append(path, preview)
		}
	}

	return style.Render("📍 " + strings.Join(path, " → "))
}

// Help rendering methods

// renderGraphHelp renders help for graph view
func (kb *KnowledgeBrowser) renderGraphHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render("[1-9] Navigate | [b] Back | [v] View | [l] List | [d] Detail | [/] Search | [q] Quit")
}

// renderListHelp renders help for list view
func (kb *KnowledgeBrowser) renderListHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render("[v] View mode | [g] Graph | [d] Detail | [/] Search | [q] Quit")
}

// renderDetailHelp renders help for detail view
func (kb *KnowledgeBrowser) renderDetailHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render("[g] Graph | [l] List | [v] View mode | [/] Search | [b] Back | [q] Quit")
}

// renderSearchHelp renders help for search view
func (kb *KnowledgeBrowser) renderSearchHelp() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render("[enter] Search | [esc] Cancel | [backspace] Delete")
}

// Utility methods

// getEdge gets the edge between two nodes
func (kb *KnowledgeBrowser) getEdge(fromID, toID string) *KnowledgeEdge {
	if kb.graph == nil {
		return nil
	}

	// Try both directions
	edgeID1 := fmt.Sprintf("%s->%s", fromID, toID)
	edgeID2 := fmt.Sprintf("%s->%s", toID, fromID)

	if edge := kb.graph.Edges[edgeID1]; edge != nil {
		return edge
	}
	if edge := kb.graph.Edges[edgeID2]; edge != nil {
		return edge
	}

	// Return a default edge if none found
	return &KnowledgeEdge{
		Type:   EdgeTypeRelated,
		Weight: 0.5,
	}
}

// formatEdgeType formats an edge type for display
func (kb *KnowledgeBrowser) formatEdgeType(edge *KnowledgeEdge) string {
	if edge == nil {
		return "→"
	}

	switch edge.Type {
	case EdgeTypeRelated:
		return "↔"
	case EdgeTypeDependsOn:
		return "⬇"
	case EdgeTypeImplements:
		return "⚙"
	case EdgeTypeConflicts:
		return "⚡"
	case EdgeTypeSupersedes:
		return "⬆"
	case EdgeTypeSimilar:
		return "≈"
	case EdgeTypeExample:
		return "💡"
	case EdgeTypeAuthor:
		return "✍"
	case EdgeTypeUsedIn:
		return "🔧"
	default:
		return "→"
	}
}

// getNodeTypeIcon returns an icon for a node type
func (kb *KnowledgeBrowser) getNodeTypeIcon(nodeType KnowledgeNodeType) string {
	switch nodeType {
	case NodeTypePattern:
		return "🏗"
	case NodeTypeDecision:
		return "⚖"
	case NodeTypeTip:
		return "💡"
	case NodeTypeReference:
		return "📖"
	case NodeTypeExample:
		return "🔍"
	case NodeTypeLesson:
		return "🎓"
	case NodeTypeConcept:
		return "🧠"
	case NodeTypePerson:
		return "👤"
	case NodeTypeProject:
		return "📁"
	default:
		return "📝"
	}
}

// getNodeTypeDisplayName returns a display name for a node type
func (kb *KnowledgeBrowser) getNodeTypeDisplayName(nodeType KnowledgeNodeType) string {
	switch nodeType {
	case NodeTypePattern:
		return "Patterns"
	case NodeTypeDecision:
		return "Decisions"
	case NodeTypeTip:
		return "Tips"
	case NodeTypeReference:
		return "References"
	case NodeTypeExample:
		return "Examples"
	case NodeTypeLesson:
		return "Lessons"
	case NodeTypeConcept:
		return "Concepts"
	case NodeTypePerson:
		return "People"
	case NodeTypeProject:
		return "Projects"
	default:
		return "Unknown"
	}
}

// getNodePreview returns a preview string for a node
func (kb *KnowledgeBrowser) getNodePreview(node *KnowledgeNode) string {
	content := kb.formatNodeContent(node.Content)

	// Try to extract a title from the content
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		title := strings.TrimSpace(lines[0])
		if len(title) > 0 {
			return title
		}
	}

	// Fallback to node ID
	return node.ID
}

// formatNodeContent formats node content for display
func (kb *KnowledgeBrowser) formatNodeContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case map[string]interface{}:
		if title, ok := v["title"].(string); ok {
			return title
		}
		if description, ok := v["description"].(string); ok {
			return description
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Command methods

// reloadGraph reloads the knowledge graph
func (kb *KnowledgeBrowser) reloadGraph() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement graph reloading
		return nil
	}
}

// performSearch performs a knowledge search
func (kb *KnowledgeBrowser) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement knowledge search
		return nil
	}
}

// showStatistics shows graph statistics
func (kb *KnowledgeBrowser) showStatistics() tea.Cmd {
	return func() tea.Msg {
		// TODO: Show detailed statistics
		return nil
	}
}

// showHelp shows detailed help
func (kb *KnowledgeBrowser) showHelp() tea.Cmd {
	return func() tea.Msg {
		// TODO: Show comprehensive help
		return nil
	}
}
