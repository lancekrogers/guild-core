// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2
package ui

import "github.com/guild-framework/guild-core/internal/ui/chat/common"

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

// StatusUpdateMsg represents status bar updates
type StatusUpdateMsg struct {
	Message string
	Level   string // info, warning, error
}
