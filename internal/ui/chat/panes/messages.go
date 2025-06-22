// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import "github.com/guild-ventures/guild-core/internal/ui/chat/common"

// LayoutUpdateMsg represents layout dimension changes
type LayoutUpdateMsg struct {
	Width  int
	Height int
}

// PaneUpdateMsg represents updates to individual panes
type PaneUpdateMsg struct {
	PaneID  string
	Content string
	Data    any
}

// ViewModeChangeMsg represents view mode changes
type ViewModeChangeMsg struct {
	Mode common.ViewMode
}

// Message types for status updates

// StatusUpdateMsg represents a status update
type StatusUpdateMsg struct {
	Message string
	Level   string // info, warning, error
}

// AgentStatusMsg represents an agent status update
type AgentStatusMsg struct {
	AgentID string
	Status  string
}

// NotificationMsg represents a notification
type NotificationMsg struct {
	Message string
	Level   string
}
