// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/cmd/guild/chat/common"
)

// Bubble Tea messages for event-driven communication between components

// AgentStreamMsg represents streaming content from an agent
type AgentStreamMsg struct {
	AgentID string
	Content string
	Done    bool
}

// AgentStatusMsg represents agent status changes
type AgentStatusMsg struct {
	AgentID string
	Status  string
}

// AgentErrorMsg represents agent errors
type AgentErrorMsg struct {
	AgentID string
	Err     error
}

// ToolExecutionStartMsg represents the start of tool execution
type ToolExecutionStartMsg struct {
	ExecutionID string
	ToolName    string
	AgentID     string
	Parameters  map[string]string
}

// ToolExecutionProgressMsg represents tool execution progress updates
type ToolExecutionProgressMsg struct {
	ExecutionID string
	Progress    float32
}

// ToolExecutionCompleteMsg represents completed tool execution
type ToolExecutionCompleteMsg struct {
	ExecutionID string
	Result      string
}

// ToolExecutionErrorMsg represents tool execution errors
type ToolExecutionErrorMsg struct {
	ExecutionID string
	Err         error
}

// ToolAuthRequiredMsg represents tool authorization requirements
type ToolAuthRequiredMsg struct {
	ToolName string
	AuthURL  string
	Message  string
}

// CompletionResultMsg represents completion results
type CompletionResultMsg struct {
	Results []CompletionResult
}

// LayoutUpdateMsg represents layout dimension changes
type LayoutUpdateMsg struct {
	Width  int
	Height int
}

// PaneUpdateMsg represents updates to individual panes
type PaneUpdateMsg struct {
	PaneID  string
	Content string
	Data    interface{}
}

// SearchMsg represents search-related messages
type SearchMsg struct {
	Pattern string
	Results []int
}

// ViewModeChangeMsg represents view mode changes
type ViewModeChangeMsg struct {
	Mode ViewMode
}

// StatusUpdateMsg represents status bar updates
type StatusUpdateMsg struct {
	Message string
	Level   string // info, warning, error
}

// Command represents a chat command
type Command struct {
	Name        string
	Description string
	Category    string
	Action      func() tea.Cmd
	Shortcut    string
}