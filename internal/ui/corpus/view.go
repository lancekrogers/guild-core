package corpus

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/internal/corpus"
)

var (
	// Colors inspired by scholarly and archival aesthetics
	archivePaper   = lipgloss.Color("#F5F5DC") // Light beige for backgrounds
	archiveInk     = lipgloss.Color("#2F2F2F") // Dark grey for text
	archiveAccent  = lipgloss.Color("#8B4513") // Brown for accents
	archiveLink    = lipgloss.Color("#436B95") // Blue for links
	archiveHeading = lipgloss.Color("#8B0000") // Dark red for headings
	archiveTag     = lipgloss.Color("#006400") // Green for tags
	archiveWarn    = lipgloss.Color("#A52A2A") // Brown-red for warnings

	// Base styles
	docStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(archiveAccent).
		Padding(1, 2).
		Background(archivePaper).
		Foreground(archiveInk)

	titleStyle = lipgloss.NewStyle().
		Foreground(archiveHeading).
		Bold(true).
		MarginBottom(1)

	tagStyle = lipgloss.NewStyle().
		Foreground(archiveTag).
		Italic(true)

	linkStyle = lipgloss.NewStyle().
		Foreground(archiveLink).
		Underline(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(archiveWarn).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(archiveInk).
		Background(archivePaper)

	statusStyle = lipgloss.NewStyle().
		Foreground(archiveInk).
		Background(archivePaper).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(archiveAccent).
		PaddingLeft(1).
		PaddingRight(1)

	// Layout elements
	appStyle = lipgloss.NewStyle().
		Margin(1, 2)
)

// View renders the current UI state
func (m CorpusModel) View() string {
	// If we're still initializing, show a loading message
	if !m.ready {
		return "Loading Corpus..."
	}

	var s string

	// Render the appropriate view based on the current mode
	switch m.mode {
	case ModeList:
		s = m.renderList()
	case ModeView:
		s = m.renderDocument()
	case ModeSearch:
		s = m.renderSearch()
	case ModeTags:
		s = m.renderTags()
	case ModeGraph:
		s = m.renderGraph()
	case ModeCommand:
		s = m.renderCommand()
	case ModeHelp:
		s = m.renderHelp()
	default:
		s = "Unknown mode: " + m.mode
	}

	// Add status bar
	s = lipgloss.JoinVertical(lipgloss.Left, s, m.renderStatus())

	// Add error message if there is one
	if m.err != nil {
		s = lipgloss.JoinVertical(
			lipgloss.Left,
			s,
			errorStyle.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	}

	// Add help bar if enabled
	if m.helpView.ShowAll {
		s = lipgloss.JoinVertical(
			lipgloss.Left,
			s,
			m.helpView.View(m.keys),
		)
	}

	// Apply overall styles and return
	return appStyle.Render(s)
}

// renderList displays the document list
func (m CorpusModel) renderList() string {
	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(m.docList.View())
}

// renderTags displays the tag list
func (m CorpusModel) renderTags() string {
	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(m.tagList.View())
}

// renderDocument displays a single document
func (m CorpusModel) renderDocument() string {
	if m.currentDoc.Title == "" {
		return docStyle.
			Width(m.width - 4).
			Height(m.height - 6).
			Render("No document selected")
	}

	// Set up the viewport with the document content
	contentView := docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(m.viewPort.View())

	// Add title and metadata above the content
	header := renderDocumentHeader(&m.currentDoc)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentView,
	)
}

// renderDocumentHeader renders the title and metadata of a document
func renderDocumentHeader(doc *corpus.CorpusDoc) string {
	// Render title
	title := titleStyle.Render(doc.Title)

	// Render metadata
	meta := fmt.Sprintf("Source: %s | Author: %s:%s | Created: %s",
		doc.Source,
		doc.GuildID,
		doc.AgentID,
		doc.CreatedAt.Format("2006-01-02"),
	)

	// Render tags
	tags := ""
	if len(doc.Tags) > 0 {
		tagTexts := make([]string, len(doc.Tags))
		for i, tag := range doc.Tags {
			tagTexts[i] = tagStyle.Render("#" + tag)
		}
		tags = strings.Join(tagTexts, " ")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		meta,
		tags,
	)
}

// renderSearch displays search interface
func (m CorpusModel) renderSearch() string {
	// Render search input
	searchBar := m.searchInput.View()

	// Render search results (using the list)
	results := m.docList.View()

	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"Search Documents:",
			searchBar,
			"",
			results,
		))
}

// renderGraph displays graph visualization
func (m CorpusModel) renderGraph() string {
	if len(m.graph.Nodes) == 0 {
		return docStyle.
			Width(m.width - 4).
			Height(m.height - 6).
			Render("Graph visualization not available")
	}

	// This is a placeholder for the actual graph visualization
	// In a real implementation, we would render a proper graph visualization
	// using ASCII/ANSI art, or use a more sophisticated approach
	
	var sb strings.Builder
	sb.WriteString("Corpus Graph Visualization\n\n")
	
	// Simple node-link representation
	nodeCount := 0
	for node, links := range m.graph.Nodes {
		if nodeCount >= m.graphOffset && nodeCount < m.graphOffset+20 {
			sb.WriteString(fmt.Sprintf("• %s\n", node))
			for _, link := range links {
				sb.WriteString(fmt.Sprintf("  └─→ %s\n", link))
			}
			sb.WriteString("\n")
		}
		nodeCount++
	}
	
	sb.WriteString(fmt.Sprintf("\nShowing %d/%d nodes (use j/k to scroll)", 
		min(20, nodeCount-m.graphOffset), 
		nodeCount))

	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(sb.String())
}

// renderCommand displays command input interface
func (m CorpusModel) renderCommand() string {
	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"Command:",
			m.commandInput.View(),
			"",
			"Available Commands:",
			" open [path]  - Open a document",
			" tag [tags]   - Filter by tags",
			" refresh      - Refresh corpus data",
			" graph        - Show graph visualization",
			" help         - Show help",
		))
}

// renderHelp displays help information
func (m CorpusModel) renderHelp() string {
	help := `
Guild Corpus Browser

Navigation:
  ↑/k, ↓/j     Move up/down
  PgUp, PgDown  Page up/down
  Enter         View selected document
  Esc, q        Return/Quit

Views:
  1             Document list
  2             Tags view
  3             Graph view
  /             Search documents

Commands:
  :             Enter command mode
  r             Refresh data
  b             Show backlinks
  ?             Toggle this help
`

	return docStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(help)
}

// renderStatus displays the status bar
func (m CorpusModel) renderStatus() string {
	// Mode indicator
	modeText := fmt.Sprintf("Mode: %s", strings.ToUpper(m.mode))
	
	// Document count
	countText := fmt.Sprintf("Documents: %d", len(m.docs))
	
	// Current user
	userText := fmt.Sprintf("User: %s", m.config.CurrentUser)
	
	// Join status elements with spacing
	status := lipgloss.JoinHorizontal(
		lipgloss.Left,
		modeText,
		strings.Repeat(" ", 5),
		countText,
		strings.Repeat(" ", 5),
		userText,
	)
	
	return statusStyle.
		Width(m.width).
		Render(status)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}