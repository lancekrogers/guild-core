// Package components provides UI components for the objective UI
package components

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/objective"
)

// StatusColors defines the colors for each objective status
var StatusColors = map[string]lipgloss.Style{
	objective.StatusEmpty:       lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
	objective.StatusDraft:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")),
	objective.StatusInProgress:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF")),
	objective.StatusReady:       lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00")),
	objective.StatusImplementing: lipgloss.NewStyle().Foreground(lipgloss.Color("#CC00FF")),
	objective.StatusCompleted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
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