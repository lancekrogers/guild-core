// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package components provides UI components for the commission UI
package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	commissionpkg "github.com/lancekrogers/guild-core/pkg/commission"
)

// StatusColors defines the colors for each commission status
var StatusColors = map[string]lipgloss.Style{
	string(commissionpkg.CommissionStatusDraft):     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")),
	string(commissionpkg.CommissionStatusActive):    lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF")),
	string(commissionpkg.CommissionStatusCompleted): lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
	string(commissionpkg.CommissionStatusCancelled): lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")),
}

// StatusBadge renders a status badge with appropriate color
func StatusBadge(status string) string {
	style, exists := StatusColors[status]
	if !exists {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	}

	// Format the status text
	formattedStatus := strings.ToUpper(status)

	return style.Render(fmt.Sprintf("[%s]", formattedStatus))
}
