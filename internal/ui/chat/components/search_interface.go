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
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// SearchInterface provides an interactive search UI for corpus content
type SearchInterface struct {
	ctx         context.Context
	query       string
	results     []SearchResult
	selected    int
	showDetails bool
	filters     SearchFilters
	inputMode   bool
	width       int
	height      int
}

// SearchResult represents a search result from the corpus
type SearchResult struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Path        string            `json:"path"`
	Source      string            `json:"source"`
	Tags        []string          `json:"tags"`
	Preview     string            `json:"preview"`
	Score       float64           `json:"score"`
	Type        string            `json:"type"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Highlighted string            `json:"highlighted,omitempty"`
}

// SearchFilters contains filtering options for search
type SearchFilters struct {
	Types            []string   `json:"types,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	Sources          []string   `json:"sources,omitempty"`
	MinScore         float64    `json:"min_score,omitempty"`
	DateRange        *DateRange `json:"date_range,omitempty"`
	InCurrentProject bool       `json:"in_current_project,omitempty"`
	Author           string     `json:"author,omitempty"`
}

// DateRange represents a time range filter
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// SearchMsg represents a search request message
type SearchMsg struct {
	Query   string
	Filters SearchFilters
}

// SearchResultsMsg represents search results
type SearchResultsMsg struct {
	Query    string
	Results  []SearchResult
	Total    int
	Duration time.Duration
	Error    error
}

// NewSearchInterface creates a new search interface
func NewSearchInterface(ctx context.Context) *SearchInterface {
	ctx = observability.WithComponent(ctx, "search_interface")

	return &SearchInterface{
		ctx:      ctx,
		selected: 0,
		filters:  SearchFilters{MinScore: 0.5},
		width:    80,
		height:   24,
	}
}

// SetSize sets the interface dimensions
func (si *SearchInterface) SetSize(width, height int) {
	si.width = width
	si.height = height
}

// SetQuery sets the search query
func (si *SearchInterface) SetQuery(query string) {
	si.query = strings.TrimSpace(query)
}

// SetResults sets the search results
func (si *SearchInterface) SetResults(results []SearchResult) {
	si.results = results
	si.selected = 0
	si.showDetails = false
}

// SetFilters sets the search filters
func (si *SearchInterface) SetFilters(filters SearchFilters) {
	si.filters = filters
}

// GetQuery returns the current query
func (si *SearchInterface) GetQuery() string {
	return si.query
}

// GetFilters returns the current filters
func (si *SearchInterface) GetFilters() SearchFilters {
	return si.filters
}

// View renders the search interface
func (si *SearchInterface) View() tea.View {
	_ = observability.WithOperation(si.ctx, "View")

	var b strings.Builder

	// Header
	b.WriteString(si.renderHeader())
	b.WriteString("\n")

	// Search box
	searchBox := si.renderSearchBox()
	b.WriteString(searchBox)
	b.WriteString("\n\n")

	// Active filters
	if si.hasActiveFilters() {
		filters := si.renderActiveFilters()
		b.WriteString(filters)
		b.WriteString("\n\n")
	}

	// Results
	if len(si.results) > 0 {
		results := si.renderResults()
		b.WriteString(results)
	} else if si.query != "" {
		b.WriteString(si.renderNoResults())
	} else {
		b.WriteString(si.renderWelcome())
	}

	// Footer help
	b.WriteString("\n")
	help := si.renderHelp()
	b.WriteString(help)

	return tea.NewView(b.String())
}

// Update handles input events
func (si *SearchInterface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := observability.WithOperation(si.ctx, "Update")

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return si.handleKeyMsg(ctx, msg)
	case SearchResultsMsg:
		return si.handleSearchResults(ctx, msg)
	case tea.WindowSizeMsg:
		si.SetSize(msg.Width, msg.Height)
	}

	return si, nil
}

// Init initializes the search interface
func (si *SearchInterface) Init() tea.Cmd {
	return nil
}

// handleKeyMsg processes keyboard input
func (si *SearchInterface) handleKeyMsg(ctx context.Context, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if si.inputMode {
		return si.handleInputMode(ctx, msg)
	}

	switch msg.String() {
	case "q", "esc":
		return si, tea.Quit

	case "enter", "/":
		si.inputMode = true
		return si, nil

	case "j", "down":
		si.moveSelection(1)

	case "k", "up":
		si.moveSelection(-1)

	case "space", "tab":
		si.toggleDetails()

	case "f":
		return si, si.showFilterHelp()

	case "c":
		si.clearFilters()

	case "r":
		if si.query != "" {
			return si, si.performSearch()
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx, _ := strconv.Atoi(msg.String())
		if idx > 0 && idx <= len(si.results) {
			si.selected = idx - 1
			si.showDetails = true
		}
	}

	return si, nil
}

// handleInputMode processes input when in search input mode
func (si *SearchInterface) handleInputMode(ctx context.Context, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		si.inputMode = false
		if si.query != "" {
			return si, si.performSearch()
		}

	case "esc":
		si.inputMode = false

	case "backspace":
		if len(si.query) > 0 {
			si.query = si.query[:len(si.query)-1]
		}

	default:
		// Add character to query
		if len(msg.String()) == 1 {
			si.query += msg.String()
		}
	}

	return si, nil
}

// handleSearchResults processes search results
func (si *SearchInterface) handleSearchResults(ctx context.Context, msg SearchResultsMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		// Handle error - could show an error message
		return si, nil
	}

	si.SetResults(msg.Results)
	return si, nil
}

// moveSelection moves the selection up or down
func (si *SearchInterface) moveSelection(delta int) {
	if len(si.results) == 0 {
		return
	}

	si.selected += delta
	if si.selected < 0 {
		si.selected = len(si.results) - 1
	} else if si.selected >= len(si.results) {
		si.selected = 0
	}
}

// toggleDetails toggles detail view for selected result
func (si *SearchInterface) toggleDetails() {
	if len(si.results) > 0 {
		si.showDetails = !si.showDetails
	}
}

// clearFilters clears all active filters
func (si *SearchInterface) clearFilters() {
	si.filters = SearchFilters{MinScore: 0.5}
}

// performSearch initiates a search
func (si *SearchInterface) performSearch() tea.Cmd {
	return func() tea.Msg {
		// This would integrate with the actual search service
		// For now, return a placeholder command
		return SearchMsg{
			Query:   si.query,
			Filters: si.filters,
		}
	}
}

// showFilterHelp shows filter configuration help
func (si *SearchInterface) showFilterHelp() tea.Cmd {
	return func() tea.Msg {
		// Could show a filter configuration dialog
		return nil
	}
}

// Rendering methods

// renderHeader renders the interface header
func (si *SearchInterface) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		Padding(0, 1)

	title := "🔍 Guild Corpus Search"
	if len(si.results) > 0 {
		title += fmt.Sprintf(" (%d results)", len(si.results))
	}

	return style.Render(title)
}

// renderSearchBox renders the search input box
func (si *SearchInterface) renderSearchBox() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(si.width - 4)

	if si.inputMode {
		style = style.BorderForeground(lipgloss.Color("69"))
	}

	query := si.query
	if si.inputMode {
		query += "█" // Cursor
	}

	if query == "" {
		query = "Type to search corpus..."
		style = style.Foreground(lipgloss.Color("240"))
	}

	return style.Render("Search: " + query)
}

// renderActiveFilters renders active search filters
func (si *SearchInterface) renderActiveFilters() string {
	var filters []string

	if len(si.filters.Types) > 0 {
		filters = append(filters, fmt.Sprintf("types:%s", strings.Join(si.filters.Types, ",")))
	}

	if len(si.filters.Tags) > 0 {
		filters = append(filters, fmt.Sprintf("tags:%s", strings.Join(si.filters.Tags, ",")))
	}

	if len(si.filters.Sources) > 0 {
		filters = append(filters, fmt.Sprintf("sources:%s", strings.Join(si.filters.Sources, ",")))
	}

	if si.filters.MinScore > 0.5 {
		filters = append(filters, fmt.Sprintf("score:>%.1f", si.filters.MinScore))
	}

	if si.filters.Author != "" {
		filters = append(filters, fmt.Sprintf("author:%s", si.filters.Author))
	}

	if si.filters.InCurrentProject {
		filters = append(filters, "scope:current-project")
	}

	if len(filters) == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	return style.Render("Active filters: " + strings.Join(filters, " | "))
}

// renderResults renders the search results
func (si *SearchInterface) renderResults() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Found %d results:\n\n", len(si.results)))

	visibleStart := 0
	visibleEnd := len(si.results)

	// Implement pagination if needed
	maxVisible := si.height - 10 // Reserve space for header/footer
	if maxVisible > 0 && len(si.results) > maxVisible {
		// Center the selection in the visible window
		half := maxVisible / 2
		visibleStart = si.selected - half
		if visibleStart < 0 {
			visibleStart = 0
		}
		visibleEnd = visibleStart + maxVisible
		if visibleEnd > len(si.results) {
			visibleEnd = len(si.results)
			visibleStart = visibleEnd - maxVisible
			if visibleStart < 0 {
				visibleStart = 0
			}
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		result := si.results[i]
		selected := i == si.selected

		// Result card
		card := si.renderResultCard(result, selected, i+1)
		b.WriteString(card)

		// Show details if selected and expanded
		if selected && si.showDetails {
			details := si.renderResultDetails(result)
			b.WriteString(details)
		}

		b.WriteString("\n")
	}

	// Show pagination info if needed
	if visibleEnd < len(si.results) {
		remaining := len(si.results) - visibleEnd
		b.WriteString(fmt.Sprintf("... and %d more results\n", remaining))
	}

	return b.String()
}

// renderResultCard renders a single search result card
func (si *SearchInterface) renderResultCard(result SearchResult, selected bool, index int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(si.width - 6)

	if selected {
		style = style.BorderForeground(lipgloss.Color("69"))
	} else {
		style = style.BorderForeground(lipgloss.Color("240"))
	}

	// Build content
	var content strings.Builder

	// Index and title with score
	title := fmt.Sprintf("%d. %s (%.0f%%)",
		index, result.Title, result.Score*100)
	titleStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		titleStyle = titleStyle.Foreground(lipgloss.Color("69"))
	}
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")

	// Preview
	preview := si.truncate(result.Preview, 80)
	if result.Highlighted != "" {
		preview = result.Highlighted
	}
	content.WriteString(preview)
	content.WriteString("\n")

	// Metadata line - include path for better visibility
	meta := fmt.Sprintf("📁 %s · %s · %s · %s",
		result.Type,
		result.Source,
		si.formatTime(result.UpdatedAt),
		result.Path)

	if len(result.Tags) > 0 {
		meta += fmt.Sprintf(" · `%s`", strings.Join(result.Tags, "` `"))
	}

	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(metaStyle.Render(meta))

	return style.Render(content.String())
}

// renderResultDetails renders detailed view of a result
func (si *SearchInterface) renderResultDetails(result SearchResult) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1).
		Width(si.width - 6).
		MarginLeft(2)

	var content strings.Builder

	content.WriteString("📄 **Full Details**\n\n")

	// Path and metadata
	content.WriteString(fmt.Sprintf("**Path:** %s\n", result.Path))
	content.WriteString(fmt.Sprintf("**Updated:** %s\n", result.UpdatedAt.Format("January 2, 2006 at 15:04")))
	content.WriteString(fmt.Sprintf("**Score:** %.2f (%.0f%%)\n\n", result.Score, result.Score*100))

	// Extended preview - always show if we have a preview
	if result.Preview != "" {
		content.WriteString("**Extended Preview:**\n")
		content.WriteString(result.Preview)
		content.WriteString("\n\n")
	}

	// Metadata
	if len(result.Metadata) > 0 {
		content.WriteString("**Metadata:**\n")
		for key, value := range result.Metadata {
			content.WriteString(fmt.Sprintf("  • %s: %s\n", key, value))
		}
		content.WriteString("\n")
	}

	content.WriteString("_Press [space] to collapse details_")

	return style.Render(content.String())
}

// renderNoResults renders the no results state
func (si *SearchInterface) renderNoResults() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		Padding(1)

	content := `No results found for your search.

**Suggestions:**
• Try different keywords
• Remove some filters
• Check spelling
• Use broader search terms
• Try searching for tags or document types

**Available commands:**
• [c] Clear filters
• [f] Configure filters  
• [/] New search`

	return style.Render(content)
}

// renderWelcome renders the welcome state
func (si *SearchInterface) renderWelcome() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("69")).
		Padding(1)

	content := `Welcome to Guild Corpus Search!

**Getting Started:**
• Press [/] or [enter] to start searching
• Use [f] to configure search filters
• Navigate results with [j/k] or arrow keys
• Press [space] to expand result details

**Search Tips:**
• Search by content, titles, tags, or types
• Use specific terms for better results
• Filter by document type, source, or date range
• Results are ranked by relevance

**Examples:**
• "authentication patterns"
• "JWT token validation"  
• "repository pattern"
• "testing strategies"`

	return style.Render(content)
}

// renderHelp renders the help footer
func (si *SearchInterface) renderHelp() string {
	var helpItems []string

	if si.inputMode {
		helpItems = []string{
			"[enter] Search",
			"[esc] Cancel",
			"[backspace] Delete",
		}
	} else {
		helpItems = []string{
			"[/] Search",
			"[j/k] Navigate",
			"[space] Details",
			"[f] Filters",
			"[c] Clear",
			"[r] Refresh",
			"[q] Quit",
		}
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render(strings.Join(helpItems, " | "))
}

// Helper methods

// hasActiveFilters checks if any filters are active
func (si *SearchInterface) hasActiveFilters() bool {
	return len(si.filters.Types) > 0 ||
		len(si.filters.Tags) > 0 ||
		len(si.filters.Sources) > 0 ||
		si.filters.MinScore > 0.5 ||
		si.filters.Author != "" ||
		si.filters.InCurrentProject ||
		si.filters.DateRange != nil
}

// truncate truncates text to specified length
func (si *SearchInterface) truncate(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Find last space before cutoff
	cutoff := maxLength - 3
	for cutoff > 0 && text[cutoff] != ' ' {
		cutoff--
	}

	if cutoff == 0 {
		cutoff = maxLength - 3
	}

	return text[:cutoff] + "..."
}

// formatTime formats a time for display
func (si *SearchInterface) formatTime(t time.Time) string {
	now := time.Now()

	if t.Year() != now.Year() {
		return t.Format("Jan 2, 2006")
	}

	if t.YearDay() != now.YearDay() {
		return t.Format("Jan 2")
	}

	return t.Format("15:04")
}

// GetSelectedResult returns the currently selected result
func (si *SearchInterface) GetSelectedResult() *SearchResult {
	if si.selected >= 0 && si.selected < len(si.results) {
		return &si.results[si.selected]
	}
	return nil
}

// SetInputMode sets whether the interface is in input mode
func (si *SearchInterface) SetInputMode(inputMode bool) {
	si.inputMode = inputMode
}

// IsInputMode returns whether the interface is in input mode
func (si *SearchInterface) IsInputMode() bool {
	return si.inputMode
}

// AddFilter adds a filter to the current filters
func (si *SearchInterface) AddFilter(filterType, value string) error {
	_ = observability.WithOperation(si.ctx, "AddFilter")

	switch filterType {
	case "type":
		if !contains(si.filters.Types, value) {
			si.filters.Types = append(si.filters.Types, value)
		}
	case "tag":
		if !contains(si.filters.Tags, value) {
			si.filters.Tags = append(si.filters.Tags, value)
		}
	case "source":
		if !contains(si.filters.Sources, value) {
			si.filters.Sources = append(si.filters.Sources, value)
		}
	case "author":
		si.filters.Author = value
	case "score":
		if score, err := strconv.ParseFloat(value, 64); err == nil {
			si.filters.MinScore = score
		} else {
			return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid score filter").
				WithComponent("search_interface").
				WithOperation("AddFilter").
				WithDetails("filter_type", filterType).
				WithDetails("value", value)
		}
	case "project":
		si.filters.InCurrentProject = value == "current" || value == "true"
	default:
		return gerror.New(gerror.ErrCodeValidation, "unknown filter type", nil).
			WithComponent("search_interface").
			WithOperation("AddFilter").
			WithDetails("filter_type", filterType)
	}

	return nil
}

// RemoveFilter removes a filter from the current filters
func (si *SearchInterface) RemoveFilter(filterType, value string) {
	switch filterType {
	case "type":
		si.filters.Types = removeString(si.filters.Types, value)
	case "tag":
		si.filters.Tags = removeString(si.filters.Tags, value)
	case "source":
		si.filters.Sources = removeString(si.filters.Sources, value)
	case "author":
		if si.filters.Author == value {
			si.filters.Author = ""
		}
	case "project":
		si.filters.InCurrentProject = false
	}
}

// Helper functions

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// removeString removes a value from a string slice
func removeString(slice []string, value string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != value {
			result = append(result, item)
		}
	}
	return result
}
