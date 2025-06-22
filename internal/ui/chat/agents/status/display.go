// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// agentDisplay implements the AgentDisplay interface
type agentDisplay struct {
	styles map[string]lipgloss.Style
}

// NewAgentDisplay creates a new agent display formatter
func NewAgentDisplay() AgentDisplay {
	return &agentDisplay{
		styles: createDisplayStyles(),
	}
}

// createDisplayStyles creates the lipgloss styles for different statuses
func createDisplayStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"idle": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
		"thinking": lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		"working": lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
		"offline": lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Gray
		"starting": lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")), // Purple
		"stopping": lipgloss.NewStyle().
			Foreground(lipgloss.Color("202")), // Light red
		"header": lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("254")), // White
		"dim": lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Gray
		"task": lipgloss.NewStyle().
			Foreground(lipgloss.Color("147")), // Light purple
		"count": lipgloss.NewStyle().
			Foreground(lipgloss.Color("111")), // Blue
	}
}

// FormatAgentStatus formats detailed agent status
func (d *agentDisplay) FormatAgentStatus(info AgentInfo) string {
	style := d.getStyleForStatus(info.Status)
	icon := d.GetStatusIcon(info.Status)

	var parts []string

	// Agent identifier
	parts = append(parts, fmt.Sprintf("%s %s (%s)",
		icon,
		d.styles["header"].Render(info.Name),
		d.styles["dim"].Render(info.Type)))

	// Status
	statusText := style.Render(string(info.Status))
	parts = append(parts, fmt.Sprintf("Status: %s", statusText))

	// Current task
	if info.CurrentTask != "" {
		parts = append(parts, fmt.Sprintf("Task: %s",
			d.styles["task"].Render(info.CurrentTask)))
	}

	// Task count
	if info.TaskCount > 0 {
		parts = append(parts, fmt.Sprintf("Tasks: %s",
			d.styles["count"].Render(fmt.Sprintf("%d", info.TaskCount))))
	}

	// Error info
	if info.ErrorCount > 0 {
		errorText := d.styles["error"].Render(fmt.Sprintf("%d errors", info.ErrorCount))
		if info.LastError != "" {
			errorText += fmt.Sprintf(" (last: %s)", info.LastError)
		}
		parts = append(parts, errorText)
	}

	// Last seen
	idleTime := time.Since(info.LastSeen)
	if idleTime > time.Minute {
		parts = append(parts, d.styles["dim"].Render(
			fmt.Sprintf("Last seen: %s ago", formatDuration(idleTime))))
	}

	return strings.Join(parts, " | ")
}

// FormatAgentCompact formats compact agent status
func (d *agentDisplay) FormatAgentCompact(info AgentInfo) string {
	style := d.getStyleForStatus(info.Status)
	icon := d.GetStatusIcon(info.Status)

	base := fmt.Sprintf("%s %s", icon, info.Name)

	// Add status if not idle
	if info.Status != StatusIdle {
		base += fmt.Sprintf(" [%s]", style.Render(string(info.Status)))
	}

	// Add task count if any
	if info.TaskCount > 0 {
		base += fmt.Sprintf(" (%d)", info.TaskCount)
	}

	return base
}

// FormatAgentList formats a list of agents
func (d *agentDisplay) FormatAgentList(agents []AgentInfo) string {
	if len(agents) == 0 {
		return d.styles["dim"].Render("No agents registered")
	}

	var lines []string

	// Group by status
	statusGroups := make(map[AgentStatus][]AgentInfo)
	for _, agent := range agents {
		statusGroups[agent.Status] = append(statusGroups[agent.Status], agent)
	}

	// Format each status group
	statusOrder := []AgentStatus{
		StatusWorking, StatusThinking, StatusIdle,
		StatusStarting, StatusStopping, StatusError, StatusOffline,
	}

	for _, status := range statusOrder {
		agentList := statusGroups[status]
		if len(agentList) == 0 {
			continue
		}

		// Status header
		style := d.getStyleForStatus(status)
		header := style.Render(fmt.Sprintf("%s %s (%d)",
			d.GetStatusIcon(status), string(status), len(agentList)))
		lines = append(lines, header)

		// Agent details
		for _, agent := range agentList {
			detail := fmt.Sprintf("  %s", d.FormatAgentCompact(agent))
			lines = append(lines, detail)
		}
	}

	return strings.Join(lines, "\n")
}

// FormatAgentSummary formats a summary of agent statuses
func (d *agentDisplay) FormatAgentSummary(agents []AgentInfo) string {
	if len(agents) == 0 {
		return d.styles["dim"].Render("No agents")
	}

	// Count by status
	statusCounts := make(map[AgentStatus]int)
	for _, agent := range agents {
		statusCounts[agent.Status]++
	}

	var parts []string

	// Format counts
	if count := statusCounts[StatusWorking]; count > 0 {
		parts = append(parts, fmt.Sprintf("%s%d",
			d.GetStatusIcon(StatusWorking), count))
	}
	if count := statusCounts[StatusThinking]; count > 0 {
		parts = append(parts, fmt.Sprintf("%s%d",
			d.GetStatusIcon(StatusThinking), count))
	}
	if count := statusCounts[StatusIdle]; count > 0 {
		parts = append(parts, fmt.Sprintf("%s%d",
			d.GetStatusIcon(StatusIdle), count))
	}
	if count := statusCounts[StatusError]; count > 0 {
		parts = append(parts, d.styles["error"].Render(
			fmt.Sprintf("%s%d", d.GetStatusIcon(StatusError), count)))
	}
	if count := statusCounts[StatusOffline]; count > 0 {
		parts = append(parts, d.styles["dim"].Render(
			fmt.Sprintf("%s%d", d.GetStatusIcon(StatusOffline), count)))
	}

	summary := fmt.Sprintf("Agents (%d): %s", len(agents), strings.Join(parts, " "))
	return summary
}

// GetStatusIcon returns an icon for the given status
func (d *agentDisplay) GetStatusIcon(status AgentStatus) string {
	switch status {
	case StatusIdle:
		return "🟢"
	case StatusThinking:
		return "🤔"
	case StatusWorking:
		return "⚙️"
	case StatusError:
		return "🔴"
	case StatusOffline:
		return "⚫"
	case StatusStarting:
		return "🔵"
	case StatusStopping:
		return "🟠"
	default:
		return "⚪"
	}
}

// GetStatusColor returns the color for the given status
func (d *agentDisplay) GetStatusColor(status AgentStatus) string {
	switch status {
	case StatusIdle:
		return "82" // Green
	case StatusThinking:
		return "226" // Yellow
	case StatusWorking:
		return "208" // Orange
	case StatusError:
		return "196" // Red
	case StatusOffline:
		return "240" // Gray
	case StatusStarting:
		return "141" // Purple
	case StatusStopping:
		return "202" // Light red
	default:
		return "254" // White
	}
}

// FormatWithIndicator formats agent info with an animated indicator
func (d *agentDisplay) FormatWithIndicator(info AgentInfo) string {
	// This would typically be called with an indicator frame
	// For now, just use the static icon
	return d.FormatAgentCompact(info)
}

// getStyleForStatus returns the lipgloss style for a status
func (d *agentDisplay) getStyleForStatus(status AgentStatus) lipgloss.Style {
	style, exists := d.styles[string(status)]
	if !exists {
		return d.styles["dim"]
	}
	return style
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		hours := int(d.Hours())
		minutes := int((d - time.Duration(hours)*time.Hour).Minutes())
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
}
