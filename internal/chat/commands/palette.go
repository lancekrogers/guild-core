// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaletteCommand represents a command in the command palette
type PaletteCommand struct {
	Name        string
	Description string
	Category    string
	Shortcut    string
	Action      func() tea.Cmd
}

// CommandPalette manages the command palette interface
type CommandPalette struct {
	commands      []PaletteCommand
	filteredCmds  []PaletteCommand
	searchQuery   string
	selectedIndex int
	isOpen        bool
	width         int
	height        int
}

// NewCommandPalette creates a new command palette
func NewCommandPalette() *CommandPalette {
	cp := &CommandPalette{
		commands:      make([]PaletteCommand, 0),
		filteredCmds:  make([]PaletteCommand, 0),
		selectedIndex: 0,
		isOpen:        false,
	}

	// Register default commands
	cp.registerDefaultCommands()

	return cp
}

// registerDefaultCommands registers all available commands
func (cp *CommandPalette) registerDefaultCommands() {
	commands := []PaletteCommand{
		// General Commands
		{
			Name:        "Show Help",
			Description: "Display available commands and shortcuts",
			Category:    "General",
			Shortcut:    "/help",
		},
		{
			Name:        "Clear Chat",
			Description: "Clear all messages from the chat",
			Category:    "General",
			Shortcut:    "/clear",
		},
		{
			Name:        "Exit Guild",
			Description: "Exit the Guild chat interface",
			Category:    "General",
			Shortcut:    "/exit",
		},

		// Agent Commands
		{
			Name:        "List Agents",
			Description: "Show all available Guild artisans",
			Category:    "Agents",
			Shortcut:    "/agents",
		},
		{
			Name:        "Select Agent",
			Description: "Choose a specific agent to work with",
			Category:    "Agents",
			Shortcut:    "/agent select",
		},
		{
			Name:        "Agent Status",
			Description: "View detailed status of all agents",
			Category:    "Agents",
			Shortcut:    "/status",
		},

		// Prompt Commands
		{
			Name:        "List Prompts",
			Description: "Show all prompt layers",
			Category:    "Prompts",
			Shortcut:    "/prompt list",
		},
		{
			Name:        "Get Prompt Layer",
			Description: "View content of a specific prompt layer",
			Category:    "Prompts",
			Shortcut:    "/prompt get",
		},
		{
			Name:        "Set Prompt Layer",
			Description: "Update content for a prompt layer",
			Category:    "Prompts",
			Shortcut:    "/prompt set",
		},

		// Tool Commands
		{
			Name:        "List Tools",
			Description: "Show all available tools",
			Category:    "Tools",
			Shortcut:    "/tools list",
		},
		{
			Name:        "Search Tools",
			Description: "Search for tools by name or capability",
			Category:    "Tools",
			Shortcut:    "/tools search",
		},
		{
			Name:        "Tool Info",
			Description: "Get detailed information about a tool",
			Category:    "Tools",
			Shortcut:    "/tools info",
		},
		{
			Name:        "Execute Tool",
			Description: "Run a tool directly",
			Category:    "Tools",
			Shortcut:    "/tool",
		},

		// Campaign Commands
		{
			Name:        "Campaign Info",
			Description: "View current campaign details",
			Category:    "Campaign",
			Shortcut:    "/campaign info",
		},

		// Test Commands
		{
			Name:        "Test Markdown",
			Description: "Test markdown rendering capabilities",
			Category:    "Testing",
			Shortcut:    "/test markdown",
		},
		{
			Name:        "Test Code Highlighting",
			Description: "Test syntax highlighting for code",
			Category:    "Testing",
			Shortcut:    "/test code",
		},
		{
			Name:        "Test Mixed Content",
			Description: "Test mixed markdown and code rendering",
			Category:    "Testing",
			Shortcut:    "/test mixed",
		},
	}

	cp.commands = commands
	cp.filteredCmds = commands
}

// Open opens the command palette
func (cp *CommandPalette) Open() {
	cp.isOpen = true
	cp.searchQuery = ""
	cp.selectedIndex = 0
	cp.filteredCmds = cp.commands
}

// Close closes the command palette
func (cp *CommandPalette) Close() {
	cp.isOpen = false
	cp.searchQuery = ""
	cp.selectedIndex = 0
}

// IsOpen returns whether the palette is open
func (cp *CommandPalette) IsOpen() bool {
	return cp.isOpen
}

// SetDimensions sets the viewport dimensions
func (cp *CommandPalette) SetDimensions(width, height int) {
	cp.width = width
	cp.height = height
}

// UpdateSearch updates the search query and filters commands
func (cp *CommandPalette) UpdateSearch(query string) {
	cp.searchQuery = query
	cp.selectedIndex = 0

	if query == "" {
		cp.filteredCmds = cp.commands
		return
	}

	// Filter commands based on fuzzy search
	cp.filteredCmds = cp.filterCommands(query)
}

// filterCommands filters commands based on the search query
func (cp *CommandPalette) filterCommands(query string) []PaletteCommand {
	var filtered []PaletteCommand
	query = strings.ToLower(query)

	for _, cmd := range cp.commands {
		// Check if query matches name, description, category, or shortcut
		if fuzzyMatch(strings.ToLower(cmd.Name), query) ||
			fuzzyMatch(strings.ToLower(cmd.Description), query) ||
			fuzzyMatch(strings.ToLower(cmd.Category), query) ||
			fuzzyMatch(strings.ToLower(cmd.Shortcut), query) {
			filtered = append(filtered, cmd)
		}
	}

	// Sort by relevance
	sort.Slice(filtered, func(i, j int) bool {
		// Prioritize exact matches in name
		iNameMatch := strings.Contains(strings.ToLower(filtered[i].Name), query)
		jNameMatch := strings.Contains(strings.ToLower(filtered[j].Name), query)

		if iNameMatch != jNameMatch {
			return iNameMatch
		}

		// Then by shortcut match
		iShortcutMatch := strings.Contains(strings.ToLower(filtered[i].Shortcut), query)
		jShortcutMatch := strings.Contains(strings.ToLower(filtered[j].Shortcut), query)

		if iShortcutMatch != jShortcutMatch {
			return iShortcutMatch
		}

		// Finally by name alphabetically
		return filtered[i].Name < filtered[j].Name
	})

	return filtered
}

// MoveUp moves selection up
func (cp *CommandPalette) MoveUp() {
	if cp.selectedIndex > 0 {
		cp.selectedIndex--
	} else {
		cp.selectedIndex = len(cp.filteredCmds) - 1
	}
}

// MoveDown moves selection down
func (cp *CommandPalette) MoveDown() {
	if cp.selectedIndex < len(cp.filteredCmds)-1 {
		cp.selectedIndex++
	} else {
		cp.selectedIndex = 0
	}
}

// GetSelectedCommand returns the currently selected command
func (cp *CommandPalette) GetSelectedCommand() *PaletteCommand {
	if cp.selectedIndex < len(cp.filteredCmds) {
		return &cp.filteredCmds[cp.selectedIndex]
	}
	return nil
}

// View returns the command palette view
func (cp *CommandPalette) View() string {
	if !cp.isOpen {
		return ""
	}

	// Styles
	paletteStyle := lipgloss.NewStyle().
		Width(cp.width - 4).
		MaxWidth(80).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")).
		Padding(1)

	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("141")).
		Foreground(lipgloss.Color("235")).
		Bold(true).
		Width(cp.width - 8)

	normalStyle := lipgloss.NewStyle().
		Width(cp.width - 8)

	categoryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	shortcutStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("71")).
		Bold(true)

	// Build content
	var content strings.Builder

	// Search box
	content.WriteString(searchStyle.Render("🔍 Search: "))
	content.WriteString(cp.searchQuery)
	if len(cp.searchQuery) > 0 {
		content.WriteString("_") // Cursor
	}
	content.WriteString("\n\n")

	// Results count
	if len(cp.filteredCmds) == 0 {
		content.WriteString("No commands found")
	} else {
		content.WriteString(fmt.Sprintf("Found %d commands\n\n", len(cp.filteredCmds)))

		// Commands list
		maxVisible := 10
		startIdx := 0

		// Adjust start index to keep selected item visible
		if cp.selectedIndex >= maxVisible {
			startIdx = cp.selectedIndex - maxVisible + 1
		}

		endIdx := startIdx + maxVisible
		if endIdx > len(cp.filteredCmds) {
			endIdx = len(cp.filteredCmds)
		}

		// Group by category
		var lastCategory string
		for i := startIdx; i < endIdx; i++ {
			cmd := cp.filteredCmds[i]

			// Show category header
			if cmd.Category != lastCategory {
				content.WriteString(categoryStyle.Render(cmd.Category))
				content.WriteString("\n")
				lastCategory = cmd.Category
			}

			// Command line
			var line strings.Builder
			line.WriteString("  ")
			line.WriteString(cmd.Name)
			line.WriteString(" - ")
			line.WriteString(cmd.Description)
			if cmd.Shortcut != "" {
				line.WriteString(" ")
				line.WriteString(shortcutStyle.Render(cmd.Shortcut))
			}

			// Apply selection style
			if i == cp.selectedIndex {
				content.WriteString(selectedStyle.Render(line.String()))
			} else {
				content.WriteString(normalStyle.Render(line.String()))
			}
			content.WriteString("\n")
		}

		// Scroll indicator
		if len(cp.filteredCmds) > maxVisible {
			content.WriteString("\n")
			content.WriteString(categoryStyle.Render(
				fmt.Sprintf("Showing %d-%d of %d (↑↓ to navigate)",
					startIdx+1, endIdx, len(cp.filteredCmds))))
		}
	}

	// Help footer
	content.WriteString("\n\n")
	content.WriteString(categoryStyle.Render("Enter: Select • Esc: Cancel • Tab: Next"))

	return paletteStyle.Render(content.String())
}

// fuzzyMatch performs fuzzy string matching
func fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternIdx := 0
	for _, ch := range text {
		if patternIdx < len(pattern) && ch == rune(pattern[patternIdx]) {
			patternIdx++
		}
	}

	return patternIdx == len(pattern)
}

// AddCommand adds a new command to the palette
func (cp *CommandPalette) AddCommand(cmd PaletteCommand) {
	cp.commands = append(cp.commands, cmd)
	if cp.searchQuery == "" {
		cp.filteredCmds = cp.commands
	} else {
		cp.filteredCmds = cp.filterCommands(cp.searchQuery)
	}
}

// GetCommands returns all registered commands
func (cp *CommandPalette) GetCommands() []PaletteCommand {
	return cp.commands
}

// GetFilteredCommands returns currently filtered commands
func (cp *CommandPalette) GetFilteredCommands() []PaletteCommand {
	return cp.filteredCmds
}

// GetSearchQuery returns the current search query
func (cp *CommandPalette) GetSearchQuery() string {
	return cp.searchQuery
}
