// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package messages

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Bubble Tea messages for event-driven communication between components

// SearchMsg represents search-related messages
type SearchMsg struct {
	Pattern string
	Results []int
}

// Command represents a chat command
type Command struct {
	Name        string
	Description string
	Category    string
	Action      func() tea.Cmd
	Shortcut    string
}

// VimModeToggleMsg is sent when vim mode should be toggled
type VimModeToggleMsg struct{}
