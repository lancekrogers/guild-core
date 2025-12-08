// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guild-framework/guild-core/tools/search"
)

// GlobalSearchKeyMap defines key bindings for the global search
type GlobalSearchKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Tab     key.Binding
	Preview key.Binding
	Refresh key.Binding
}

// DefaultGlobalSearchKeyMap returns the default key bindings
func DefaultGlobalSearchKeyMap() GlobalSearchKeyMap {
	return GlobalSearchKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "jump to location"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "close search"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle preview"),
		),
		Preview: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "toggle preview"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh search"),
		),
	}
}

// GlobalSearchModel represents the global search UI state
type GlobalSearchModel struct {
	// UI components
	input  textinput.Model
	keyMap GlobalSearchKeyMap
	width  int
	height int

	// State
	agTool         *search.AgTool
	results        []search.AgSearchResult
	selectedIndex  int
	showPreview    bool
	previewContent string
	loading        bool
	err            error
	lastQuery      string

	// Callbacks
	onLocationSelected func(filePath string, line int, column int)
	onClose            func()

	// Styles
	listStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	previewStyle  lipgloss.Style
	headerStyle   lipgloss.Style
	footerStyle   lipgloss.Style
	errorStyle    lipgloss.Style
	matchStyle    lipgloss.Style
}

// GlobalSearchOption configures the global search model
type GlobalSearchOption func(*GlobalSearchModel)

// WithLocationSelectedCallback sets the callback for when a location is selected
func WithLocationSelectedCallback(callback func(filePath string, line int, column int)) GlobalSearchOption {
	return func(m *GlobalSearchModel) {
		m.onLocationSelected = callback
	}
}

// WithGlobalSearchCloseCallback sets the callback for when the search is closed
func WithGlobalSearchCloseCallback(callback func()) GlobalSearchOption {
	return func(m *GlobalSearchModel) {
		m.onClose = callback
	}
}

// NewGlobalSearchModel creates a new global search model
func NewGlobalSearchModel(workingDir string, opts ...GlobalSearchOption) *GlobalSearchModel {
	input := textinput.New()
	input.Placeholder = "Enter search pattern (regex supported)..."
	input.Focus()
	input.CharLimit = 256
	input.Width = 50

	agTool := search.NewAgTool(workingDir)

	m := &GlobalSearchModel{
		input:       input,
		keyMap:      DefaultGlobalSearchKeyMap(),
		agTool:      agTool,
		showPreview: true,

		// Default styles
		listStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			MarginRight(1),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("13")).
			Bold(true),

		previewStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Bold(true).
			MarginBottom(1),

		footerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true),

		matchStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("11")).
			Bold(true),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Init initializes the global search model
func (m *GlobalSearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m *GlobalSearchModel) Update(msg tea.Msg) (*GlobalSearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Escape):
			if m.onClose != nil {
				m.onClose()
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Enter):
			if len(m.results) > 0 && m.selectedIndex < len(m.results) {
				selectedResult := m.results[m.selectedIndex]
				if m.onLocationSelected != nil {
					m.onLocationSelected(selectedResult.File, selectedResult.Line, selectedResult.Column)
				}
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
				return m, m.updatePreview()
			}

		case key.Matches(msg, m.keyMap.Down):
			if m.selectedIndex < len(m.results)-1 {
				m.selectedIndex++
				return m, m.updatePreview()
			}

		case key.Matches(msg, m.keyMap.Tab, m.keyMap.Preview):
			m.showPreview = !m.showPreview
			if m.showPreview {
				return m, m.updatePreview()
			}

		case key.Matches(msg, m.keyMap.Refresh):
			if m.lastQuery != "" {
				return m, m.performSearch(m.lastQuery)
			}

		default:
			// Update input and trigger search
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

			// Trigger search on Enter or when query changes significantly
			query := strings.TrimSpace(m.input.Value())
			if query != m.lastQuery && len(query) > 1 {
				m.selectedIndex = 0
				m.lastQuery = query
				cmds = append(cmds, m.performSearch(query))
			}
		}

	case globalSearchResultsMsg:
		m.results = msg.results
		m.err = msg.err
		m.loading = false
		if m.showPreview && len(m.results) > 0 {
			cmds = append(cmds, m.updatePreview())
		}

	case globalSearchPreviewMsg:
		m.previewContent = string(msg)

	case globalSearchErrorMsg:
		m.err = error(msg)
		m.loading = false
	}

	return m, tea.Batch(cmds...)
}

// View renders the global search interface
func (m *GlobalSearchModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var sections []string

	// Header
	header := m.headerStyle.Render("🔍 Global Search")
	sections = append(sections, header)

	// Input
	sections = append(sections, m.input.View())

	// Error display
	if m.err != nil {
		errorText := m.errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		sections = append(sections, errorText)
	}

	// Main content area
	if m.showPreview {
		mainContent := m.renderSplitView()
		sections = append(sections, mainContent)
	} else {
		resultsList := m.renderResultsList()
		sections = append(sections, resultsList)
	}

	// Footer with help
	footer := m.renderFooter()
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSplitView renders the results list and preview side by side
func (m *GlobalSearchModel) renderSplitView() string {
	listWidth := m.width/2 - 2
	previewWidth := m.width/2 - 2

	// Results list
	resultsList := m.renderResultsListWithWidth(listWidth)
	styledList := m.listStyle.Width(listWidth).Render(resultsList)

	// Preview
	preview := m.renderPreviewWithWidth(previewWidth)
	styledPreview := m.previewStyle.Width(previewWidth).Render(preview)

	return lipgloss.JoinHorizontal(lipgloss.Top, styledList, styledPreview)
}

// renderResultsList renders the search results list
func (m *GlobalSearchModel) renderResultsList() string {
	return m.renderResultsListWithWidth(m.width - 4)
}

// renderResultsListWithWidth renders the results list with specified width
func (m *GlobalSearchModel) renderResultsListWithWidth(width int) string {
	if m.loading {
		return "Searching..."
	}

	if len(m.results) == 0 {
		if m.input.Value() == "" {
			return "Enter a search pattern to find text in files"
		}
		return "No matches found"
	}

	var lines []string
	maxVisible := m.height - 8 // Account for header, input, footer

	start := 0
	end := len(m.results)

	// Scroll to keep selected item visible
	if len(m.results) > maxVisible {
		if m.selectedIndex >= maxVisible/2 {
			start = m.selectedIndex - maxVisible/2
			end = start + maxVisible
			if end > len(m.results) {
				end = len(m.results)
				start = end - maxVisible
				if start < 0 {
					start = 0
				}
			}
		} else {
			end = maxVisible
		}
	}

	for i := start; i < end; i++ {
		result := m.results[i]
		line := m.formatSearchResult(result, width-2)

		if i == m.selectedIndex {
			line = m.selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	// Add scroll indicator if needed
	if len(m.results) > maxVisible {
		scrollInfo := fmt.Sprintf("(%d-%d of %d matches)", start+1, end, len(m.results))
		lines = append(lines, scrollInfo)
	}

	return strings.Join(lines, "\n")
}

// renderPreviewWithWidth renders the search preview with specified width
func (m *GlobalSearchModel) renderPreviewWithWidth(width int) string {
	if len(m.results) == 0 || m.selectedIndex >= len(m.results) {
		return "No preview available"
	}

	selectedResult := m.results[m.selectedIndex]

	var content strings.Builder
	content.WriteString(fmt.Sprintf("📄 %s:%d:%d\n", selectedResult.File, selectedResult.Line, selectedResult.Column))
	content.WriteString(fmt.Sprintf("🔍 Match: %s\n", selectedResult.Match))
	content.WriteString("\n")

	if m.previewContent != "" {
		// Show context around the match
		lines := strings.Split(m.previewContent, "\n")
		maxLines := m.height - 12 // Account for file info and other UI elements

		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, "...")
		}

		for i, line := range lines {
			// Truncate long lines
			if len(line) > width-4 {
				line = line[:width-7] + "..."
			}

			// Highlight the line containing the match
			if strings.Contains(line, selectedResult.Match) {
				line = m.highlightMatch(line, selectedResult.Match)
			}

			// Add line numbers
			lineNum := selectedResult.Line - len(lines)/2 + i
			if lineNum > 0 {
				content.WriteString(fmt.Sprintf("%4d: %s\n", lineNum, line))
			}
		}
	} else {
		content.WriteString("Loading preview...")
	}

	return content.String()
}

// renderFooter renders the help footer
func (m *GlobalSearchModel) renderFooter() string {
	help := []string{
		"↑/↓: navigate",
		"Enter: jump to location",
		"Tab: toggle preview",
		"Ctrl+R: refresh",
		"Esc: close",
	}

	statusInfo := fmt.Sprintf("Results: %d", len(m.results))
	if m.lastQuery != "" {
		statusInfo += fmt.Sprintf(" | Pattern: %s", m.lastQuery)
	}

	helpText := strings.Join(help, " • ")

	return m.footerStyle.Render(helpText + "\n" + statusInfo)
}

// formatSearchResult formats a search result for display
func (m *GlobalSearchModel) formatSearchResult(result search.AgSearchResult, width int) string {
	// Format: filename:line:column match_text
	location := fmt.Sprintf("%s:%d:%d", result.File, result.Line, result.Column)

	// Truncate file path if too long
	maxLocationLength := width / 2
	if len(location) > maxLocationLength {
		location = "..." + location[len(location)-maxLocationLength+3:]
	}

	// Format match text
	matchText := strings.TrimSpace(result.Context)
	if matchText == "" {
		matchText = result.Match
	}

	// Truncate match text if needed
	remainingWidth := width - len(location) - 3 // Account for spacing
	if len(matchText) > remainingWidth {
		matchText = matchText[:remainingWidth-3] + "..."
	}

	return fmt.Sprintf("%-*s  %s", len(location), location, matchText)
}

// highlightMatch highlights the search match in a line
func (m *GlobalSearchModel) highlightMatch(line, match string) string {
	if match == "" {
		return line
	}

	// Simple highlighting - can be enhanced with proper regex matching
	highlighted := strings.ReplaceAll(line, match, m.matchStyle.Render(match))
	return highlighted
}

// updateLayout updates the layout based on current dimensions
func (m *GlobalSearchModel) updateLayout() {
	// Update input width
	inputWidth := m.width - 4
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.input.Width = inputWidth
}

// Message types
type globalSearchResultsMsg struct {
	results []search.AgSearchResult
	err     error
}

type globalSearchPreviewMsg string
type globalSearchErrorMsg error

// Commands

// performSearch performs a global search using the ag tool
func (m *GlobalSearchModel) performSearch(pattern string) tea.Cmd {
	return func() tea.Msg {
		m.loading = true
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create search input
		searchInput := search.AgToolInput{
			Pattern:    pattern,
			MaxResults: 100,
			Context:    2, // Include context lines
			Timeout:    10,
		}

		// Execute search
		result, err := m.agTool.Execute(ctx, toJSON(searchInput))
		if err != nil {
			return globalSearchErrorMsg(err)
		}

		// Parse results
		if result.Error != "" {
			return globalSearchErrorMsg(fmt.Errorf("search error: %s", result.Error))
		}

		agResult, ok := result.ExtraData["structured_results"].(*search.AgToolResult)
		if !ok {
			return globalSearchErrorMsg(fmt.Errorf("invalid search results"))
		}

		return globalSearchResultsMsg{
			results: agResult.Results,
			err:     nil,
		}
	}
}

// updatePreview updates the search result preview
func (m *GlobalSearchModel) updatePreview() tea.Cmd {
	if len(m.results) == 0 || m.selectedIndex >= len(m.results) {
		return nil
	}

	selectedResult := m.results[m.selectedIndex]

	return func() tea.Msg {
		// Read file content around the match for preview
		content, err := readFileContext(selectedResult.File, selectedResult.Line, 10)
		if err != nil {
			return globalSearchPreviewMsg(fmt.Sprintf("Cannot preview: %v", err))
		}

		return globalSearchPreviewMsg(content)
	}
}

// Helper functions

// readFileContext reads file content around a specific line for preview
func readFileContext(filePath string, lineNumber, contextLines int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string

	// Read all lines (simplified version - for production should be more efficient)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	allLines := strings.Split(string(content), "\n")

	// Extract context around the target line
	start := lineNumber - contextLines - 1
	if start < 0 {
		start = 0
	}

	end := lineNumber + contextLines
	if end > len(allLines) {
		end = len(allLines)
	}

	lines = allLines[start:end]
	return strings.Join(lines, "\n"), nil
}

// toJSON converts AgToolInput to JSON string
func toJSON(input search.AgToolInput) string {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		// Fallback to simple format
		return fmt.Sprintf(`{"pattern": "%s", "max_results": %d, "context": %d, "timeout": %d}`,
			input.Pattern, input.MaxResults, input.Context, input.Timeout)
	}
	return string(jsonBytes)
}
