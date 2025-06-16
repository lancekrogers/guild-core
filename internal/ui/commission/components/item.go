// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	// "fmt" - temporarily unused due to commented out formatTimeAgo function
	"time"
)

// CommissionItem represents an objective in the ledger list
type CommissionItem struct {
	ID          string
	Title       string
	Status      string
	Path        string
	Iterations  int
	CreatedAt   time.Time
	ModifiedAt  time.Time
	Tags        []string
	Completion  float64 // 0.0-1.0 representing completion percentage
	Description string
}

// FilterValue implements list.Item interface
func (i CommissionItem) FilterValue() string {
	return i.Title
}

// formatTimeAgo formats a time in a relative manner (e.g., "2 days ago")
// TODO: This function will be used for displaying commission timestamps in the UI
// Temporarily commented out to avoid "unused" linter warnings
/*
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
*/
