package components

import (
	"fmt"
	"time"
)

// ObjectiveItem represents an objective in the ledger list
type ObjectiveItem struct {
	ID           string
	Title        string
	Status       string
	Path         string
	Iterations   int
	CreatedAt    time.Time
	ModifiedAt   time.Time
	Tags         []string
	Completion   float64 // 0.0-1.0 representing completion percentage
	Description  string
}

// FilterValue implements list.Item interface
func (i ObjectiveItem) FilterValue() string { 
	return i.Title 
}

// Title returns the item's title for the list
func (i ObjectiveItem) Title() string { 
	return i.Title 
}

// Description returns a formatted description for the list item
func (i ObjectiveItem) Description() string {
	// Format modified time as relative time
	timeAgo := formatTimeAgo(i.ModifiedAt)
	
	// Format status with emoji
	statusEmoji := "🔄" // Default - in progress
	switch i.Status {
	case "not_started":
		statusEmoji = "🆕"
	case "draft":
		statusEmoji = "📝"
	case "in_progress":
		statusEmoji = "🔄"
	case "finalized":
		statusEmoji = "✅"
	}
	
	// Format completion percentage
	completionStr := fmt.Sprintf("%.0f%%", i.Completion*100)
	
	return fmt.Sprintf("%s %s | Modified: %s | Completion: %s", 
		statusEmoji, 
		i.Status, 
		timeAgo, 
		completionStr,
	)
}

// formatTimeAgo formats a time in a relative manner (e.g., "2 days ago")
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