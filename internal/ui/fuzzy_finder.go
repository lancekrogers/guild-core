// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/guild-framework/guild-core/pkg/search"
)

// FuzzyFinderKeyMap defines key bindings for the fuzzy finder
type FuzzyFinderKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Tab     key.Binding
	Preview key.Binding
	Refresh key.Binding
}

// DefaultFuzzyFinderKeyMap returns the default key bindings
func DefaultFuzzyFinderKeyMap() FuzzyFinderKeyMap {
	return FuzzyFinderKeyMap{
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
			key.WithHelp("enter", "select file"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "close finder"),
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
			key.WithHelp("ctrl+r", "refresh index"),
		),
	}
}

// FuzzyFinderModel represents the fuzzy finder UI state
type FuzzyFinderModel struct {
	// UI components
	input  textinput.Model
	keyMap FuzzyFinderKeyMap
	width  int
	height int

	// State
	fuzzyFinder    *search.FuzzyFinder
	results        []search.FileResult
	selectedIndex  int
	showPreview    bool
	previewContent string
	loading        bool
	err            error

	// Callbacks
	onFileSelected func(filePath string)
	onClose        func()

	// Styles
	listStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	previewStyle  lipgloss.Style
	headerStyle   lipgloss.Style
	footerStyle   lipgloss.Style
	errorStyle    lipgloss.Style
}

// FuzzyFinderOption configures the fuzzy finder model
type FuzzyFinderOption func(*FuzzyFinderModel)

// WithFileSelectedCallback sets the callback for when a file is selected
func WithFileSelectedCallback(callback func(filePath string)) FuzzyFinderOption {
	return func(m *FuzzyFinderModel) {
		m.onFileSelected = callback
	}
}

// WithCloseCallback sets the callback for when the finder is closed
func WithCloseCallback(callback func()) FuzzyFinderOption {
	return func(m *FuzzyFinderModel) {
		m.onClose = callback
	}
}

// NewFuzzyFinderModel creates a new fuzzy finder model
func NewFuzzyFinderModel(workingDir string, opts ...FuzzyFinderOption) *FuzzyFinderModel {
	input := textinput.New()
	input.Placeholder = "Type to search files..."
	input.Focus()
	input.CharLimit = 256
	input.Width = 50

	fuzzyFinder := search.NewFuzzyFinder(search.FuzzyFinderConfig{
		WorkingDir:      workingDir,
		ExcludePatterns: search.DefaultExcludePatterns(),
		MaxResults:      20,
		IndexTimeout:    5 * time.Second,
	})

	m := &FuzzyFinderModel{
		input:       input,
		keyMap:      DefaultFuzzyFinderKeyMap(),
		fuzzyFinder: fuzzyFinder,
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
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Init initializes the fuzzy finder model
func (m *FuzzyFinderModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.searchFiles(),
	)
}

// Update handles messages and updates the model
func (m *FuzzyFinderModel) Update(msg tea.Msg) (*FuzzyFinderModel, tea.Cmd) {
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
				selectedFile := m.results[m.selectedIndex]
				m.fuzzyFinder.MarkRecentFile(selectedFile.Path)
				if m.onFileSelected != nil {
					m.onFileSelected(selectedFile.AbsPath)
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
			return m, m.refreshIndex()

		default:
			// Update input and trigger search
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

			// Reset selection when search changes
			m.selectedIndex = 0
			cmds = append(cmds, m.searchFiles())
		}

	case searchResultsMsg:
		m.results = msg.results
		m.err = msg.err
		m.loading = false
		if m.showPreview && len(m.results) > 0 {
			cmds = append(cmds, m.updatePreview())
		}

	case previewContentMsg:
		m.previewContent = string(msg)

	case indexRefreshedMsg:
		m.loading = false
		cmds = append(cmds, m.searchFiles())

	case errorMsg:
		m.err = error(msg)
		m.loading = false
	}

	return m, tea.Batch(cmds...)
}

// View renders the fuzzy finder interface
func (m *FuzzyFinderModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var sections []string

	// Header
	header := m.headerStyle.Render("📁 Fuzzy File Finder")
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
		fileList := m.renderFileList()
		sections = append(sections, fileList)
	}

	// Footer with help
	footer := m.renderFooter()
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSplitView renders the file list and preview side by side
func (m *FuzzyFinderModel) renderSplitView() string {
	listWidth := m.width/2 - 2
	previewWidth := m.width/2 - 2

	// File list
	fileList := m.renderFileListWithWidth(listWidth)
	styledList := m.listStyle.Width(listWidth).Render(fileList)

	// Preview
	preview := m.renderPreviewWithWidth(previewWidth)
	styledPreview := m.previewStyle.Width(previewWidth).Render(preview)

	return lipgloss.JoinHorizontal(lipgloss.Top, styledList, styledPreview)
}

// renderFileList renders the file list
func (m *FuzzyFinderModel) renderFileList() string {
	return m.renderFileListWithWidth(m.width - 4)
}

// renderFileListWithWidth renders the file list with specified width
func (m *FuzzyFinderModel) renderFileListWithWidth(width int) string {
	if m.loading {
		return "Searching..."
	}

	if len(m.results) == 0 {
		if m.input.Value() == "" {
			return "Type to search files, or press Enter to see recent files"
		}
		return "No files found"
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
		line := m.formatFileResult(result, width-2)

		if i == m.selectedIndex {
			line = m.selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	// Add scroll indicator if needed
	if len(m.results) > maxVisible {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", start+1, end, len(m.results))
		lines = append(lines, scrollInfo)
	}

	return strings.Join(lines, "\n")
}

// renderPreviewWithWidth renders the file preview with specified width
func (m *FuzzyFinderModel) renderPreviewWithWidth(width int) string {
	if len(m.results) == 0 || m.selectedIndex >= len(m.results) {
		return "No preview available"
	}

	selectedFile := m.results[m.selectedIndex]

	var content strings.Builder
	content.WriteString(fmt.Sprintf("📄 %s\n", selectedFile.Name))
	content.WriteString(fmt.Sprintf("📁 %s\n", selectedFile.Path))
	content.WriteString(fmt.Sprintf("📊 %s\n", formatFileSize(selectedFile.Size)))

	if selectedFile.IsRecent {
		content.WriteString("⭐ Recent file\n")
	}

	content.WriteString("\n")

	if m.previewContent != "" {
		// Limit preview content to available space
		lines := strings.Split(m.previewContent, "\n")
		maxLines := m.height - 12 // Account for file info and other UI elements

		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, "...")
		}

		for _, line := range lines {
			// Truncate long lines
			if len(line) > width-4 {
				line = line[:width-7] + "..."
			}
			content.WriteString(line + "\n")
		}
	} else {
		content.WriteString("Loading preview...")
	}

	return content.String()
}

// renderFooter renders the help footer
func (m *FuzzyFinderModel) renderFooter() string {
	help := []string{
		"↑/↓: navigate",
		"Enter: select",
		"Tab: toggle preview",
		"Ctrl+R: refresh",
		"Esc: close",
	}

	stats := m.fuzzyFinder.GetIndexStats()
	totalFiles := stats["total_files"].(int)
	recentFiles := stats["recent_files"].(int)

	statusInfo := fmt.Sprintf("Files: %d | Recent: %d | Results: %d",
		totalFiles, recentFiles, len(m.results))

	helpText := strings.Join(help, " • ")

	return m.footerStyle.Render(helpText + "\n" + statusInfo)
}

// formatFileResult formats a file result for display
func (m *FuzzyFinderModel) formatFileResult(result search.FileResult, width int) string {
	var parts []string

	// File icon
	icon := "📄"
	if result.IsDirectory {
		icon = "📁"
	}

	// Recent indicator
	if result.IsRecent {
		icon = "⭐"
	}

	// Format path
	displayPath := result.Path
	maxPathLength := width - 20 // Reserve space for icon, size, etc.
	if len(displayPath) > maxPathLength {
		displayPath = "..." + displayPath[len(displayPath)-maxPathLength+3:]
	}

	parts = append(parts, icon, displayPath)

	// Add size for files
	if !result.IsDirectory {
		size := formatFileSize(result.Size)
		parts = append(parts, fmt.Sprintf("(%s)", size))
	}

	line := strings.Join(parts, " ")

	// Ensure line doesn't exceed width
	if len(line) > width {
		line = line[:width-3] + "..."
	}

	return line
}

// updateLayout updates the layout based on current dimensions
func (m *FuzzyFinderModel) updateLayout() {
	// Update input width
	inputWidth := m.width - 4
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.input.Width = inputWidth
}

// Message types
type searchResultsMsg struct {
	results []search.FileResult
	err     error
}

type (
	previewContentMsg string
	indexRefreshedMsg struct{}
	errorMsg          error
)

// Commands

// searchFiles performs a file search
func (m *FuzzyFinderModel) searchFiles() tea.Cmd {
	return func() tea.Msg {
		m.loading = true
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pattern := m.input.Value()
		results, err := m.fuzzyFinder.Search(ctx, pattern)

		return searchResultsMsg{
			results: results,
			err:     err,
		}
	}
}

// updatePreview updates the file preview
func (m *FuzzyFinderModel) updatePreview() tea.Cmd {
	if len(m.results) == 0 || m.selectedIndex >= len(m.results) {
		return nil
	}

	selectedFile := m.results[m.selectedIndex]

	return func() tea.Msg {
		// Don't preview directories or very large files
		if selectedFile.IsDirectory || selectedFile.Size > 1024*1024 {
			return previewContentMsg("Directory or file too large to preview")
		}

		content, err := readFilePreview(selectedFile.AbsPath, 50) // First 50 lines
		if err != nil {
			return previewContentMsg(fmt.Sprintf("Cannot preview file: %v", err))
		}

		return previewContentMsg(content)
	}
}

// refreshIndex refreshes the file index
func (m *FuzzyFinderModel) refreshIndex() tea.Cmd {
	return func() tea.Msg {
		m.loading = true
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.fuzzyFinder.RefreshIndex(ctx)
		if err != nil {
			return errorMsg(err)
		}

		return indexRefreshedMsg{}
	}
}

// Helper functions

// readFilePreview reads the first N lines of a file for preview
func readFilePreview(filePath string, maxLines int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 8KB to check if it's a text file
	buffer := make([]byte, 8192)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Simple binary file detection
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return "Binary file - cannot preview", nil
		}
	}

	// Reset file position
	file.Seek(0, 0)

	var lines []string
	content := string(buffer[:n])

	for _, line := range strings.Split(content, "\n") {
		if len(lines) >= maxLines {
			break
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
