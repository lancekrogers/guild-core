package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StatusDisplay renders agent status information in the chat interface
type StatusDisplay struct {
	tracker       *AgentStatusTracker
	width         int
	height        int
	showDetails   bool              // Expanded vs compact view
	selectedAgent string            // Currently focused agent
	styles        StatusStyles      // Medieval-themed styling
	lastUpdate    time.Time         // Last update timestamp for refresh control
}

// StatusStyles contains all visual styling for status display
type StatusStyles struct {
	AgentPanel     lipgloss.Style   // Individual agent status panels
	ActivityBadge  lipgloss.Style   // Activity state indicators
	ProgressBar    lipgloss.Style   // Task progress visualization
	ToolIndicator  lipgloss.Style   // Active tool display
	HeaderStyle    lipgloss.Style   // Status section headers
	CompactStyle   lipgloss.Style   // Compact status line
	CoordStyle     lipgloss.Style   // Coordination events
	CostStyle      lipgloss.Style   // Cost information
	MetricsStyle   lipgloss.Style   // Metrics and statistics
}

// NewStatusDisplay creates a new status display with medieval styling
func NewStatusDisplay(tracker *AgentStatusTracker, width, height int) *StatusDisplay {
	display := &StatusDisplay{
		tracker:       tracker,
		width:         width,
		height:        height,
		showDetails:   true, // Start in detailed view
		selectedAgent: "",
		lastUpdate:    time.Now(),
	}

	// Initialize medieval-themed styles
	display.initializeStyles()

	return display
}

// initializeStyles sets up the medieval-themed visual styles
func (d *StatusDisplay) initializeStyles() {
	// Agent panel style - rounded border with Guild colors
	d.styles.AgentPanel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Purple border
		Padding(0, 1).
		Margin(0, 0, 1, 0).
		Width(d.width - 4)

	// Activity badge style - for status indicators
	d.styles.ActivityBadge = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Margin(0, 1, 0, 0)

	// Progress bar style
	d.styles.ProgressBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("76")). // Green
		Background(lipgloss.Color("8"))   // Dark gray

	// Tool indicator style
	d.styles.ToolIndicator = lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")). // Orange
		Bold(true)

	// Header style for sections
	d.styles.HeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Bright blue
		Bold(true).
		Underline(true).
		Margin(1, 0, 0, 0)

	// Compact status line style
	d.styles.CompactStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // Light gray
		Italic(true)

	// Coordination events style
	d.styles.CoordStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("219")). // Pink/magenta
		Bold(true)

	// Cost information style
	d.styles.CostStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")). // Yellow
		Bold(true)

	// Metrics style
	d.styles.MetricsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("144")). // Light purple
		Italic(true)
}

// RenderStatusPanel renders the full agent status panel
func (d *StatusDisplay) RenderStatusPanel() string {
	if d.tracker == nil {
		return d.styles.AgentPanel.Render("⚠️ Status tracker not available")
	}

	var content strings.Builder

	// Header with timestamp and summary
	content.WriteString(d.styles.HeaderStyle.Render("🏰 Guild Agent Status"))
	content.WriteString("\n")
	content.WriteString(d.renderSummaryLine())
	content.WriteString("\n\n")

	if d.showDetails {
		content.WriteString(d.renderDetailedStatus())
	} else {
		content.WriteString(d.RenderCompactStatus())
	}

	return d.styles.AgentPanel.Render(content.String())
}

// RenderCompactStatus renders a single line status summary
func (d *StatusDisplay) RenderCompactStatus() string {
	if d.tracker == nil {
		return d.styles.CompactStyle.Render("Status unavailable")
	}

	agents := d.tracker.GetAllAgents()
	activeCount := 0
	workingCount := 0

	for _, agent := range agents {
		if agent.State != AgentOffline {
			activeCount++
		}
		if agent.State == AgentWorking {
			workingCount++
		}
	}

	summary := fmt.Sprintf("🏰 %d agents (%d active, %d working)",
		len(agents), activeCount, workingCount)

	// Add tool count if any are active
	if len(d.tracker.activeTools) > 0 {
		summary += fmt.Sprintf(" | 🔨 %d tools", len(d.tracker.activeTools))
	}

	return d.styles.CompactStyle.Render(summary)
}

// renderSummaryLine creates a summary line with key metrics
func (d *StatusDisplay) renderSummaryLine() string {
	summary := d.tracker.GetCoordinationSummary()

	return d.styles.MetricsStyle.Render(fmt.Sprintf(
		"Active: %d/%d | Tools: %d | Cost: $%.3f | Updated: %s",
		summary["active_agents"],
		summary["total_agents"],
		summary["active_tools"],
		summary["total_cost"],
		d.lastUpdate.Format("15:04:05"),
	))
}

// renderDetailedStatus renders detailed agent status information
func (d *StatusDisplay) renderDetailedStatus() string {
	var content strings.Builder

	// Get all agents and sort by activity level
	agents := d.tracker.GetAllAgents()
	if len(agents) == 0 {
		content.WriteString("No agents configured.\n")
		return content.String()
	}

	// Render each agent
	for i, agent := range agents {
		if i > 0 {
			content.WriteString("\n")
		}
		content.WriteString(d.RenderAgentCard(agent))
	}

	// Add coordination events if any
	content.WriteString("\n")
	content.WriteString(d.renderCoordinationSection())

	return content.String()
}

// RenderAgentCard renders status information for a single agent
func (d *StatusDisplay) RenderAgentCard(agent *AgentStatus) string {
	var card strings.Builder

	// Agent name and type with status indicator
	statusIcon := d.getStatusIcon(agent.State)
	agentHeader := fmt.Sprintf("%s %s (%s)", statusIcon, agent.Name, agent.Type)

	// Highlight selected agent
	if d.selectedAgent == agent.ID {
		agentHeader = d.styles.ActivityBadge.
			Background(lipgloss.Color("57")). // Highlight background
			Render(agentHeader)
	} else {
		agentHeader = d.styles.ActivityBadge.
			Foreground(d.getStatusColor(agent.State)).
			Render(agentHeader)
	}

	card.WriteString(agentHeader)
	card.WriteString("\n")

	// Current task and progress
	if agent.CurrentTask != "" {
		card.WriteString(fmt.Sprintf("   Task: %s", agent.CurrentTask))
		card.WriteString("\n")

		if agent.Progress > 0 {
			progressBar := d.RenderProgressBar(agent.Progress)
			card.WriteString(fmt.Sprintf("   %s %.1f%%", progressBar, agent.Progress*100))
			card.WriteString("\n")
		}
	}

	// Active tools
	if len(agent.ActiveTools) > 0 {
		toolStr := strings.Join(agent.ActiveTools, ", ")
		card.WriteString(d.styles.ToolIndicator.Render(fmt.Sprintf("   🔨 Tools: %s", toolStr)))
		card.WriteString("\n")
	}

	// Agent metrics
	metrics := d.renderAgentMetrics(agent)
	if metrics != "" {
		card.WriteString(metrics)
	}

	return card.String()
}

// renderAgentMetrics creates a metrics line for an agent
func (d *StatusDisplay) renderAgentMetrics(agent *AgentStatus) string {
	var metrics []string

	// Add capabilities (first 2 only for space)
	if len(agent.Capabilities) > 0 {
		capStr := strings.Join(agent.Capabilities[:min(2, len(agent.Capabilities))], ", ")
		if len(agent.Capabilities) > 2 {
			capStr += "..."
		}
		metrics = append(metrics, fmt.Sprintf("Cap: %s", capStr))
	}

	// Add cost info
	if agent.TotalCost > 0 {
		costStr := d.styles.CostStyle.Render(fmt.Sprintf("$%.3f", agent.TotalCost))
		metrics = append(metrics, fmt.Sprintf("Cost: %s", costStr))
	}

	// Add task count
	if agent.TasksCompleted > 0 {
		metrics = append(metrics, fmt.Sprintf("Tasks: %d", agent.TasksCompleted))
	}

	// Add uptime
	uptime := time.Since(agent.StartTime)
	if uptime > time.Minute {
		uptimeStr := uptime.Round(time.Minute).String()
		metrics = append(metrics, fmt.Sprintf("Up: %s", uptimeStr))
	}

	if len(metrics) > 0 {
		return d.styles.MetricsStyle.Render(fmt.Sprintf("   %s\n", strings.Join(metrics, " | ")))
	}
	return ""
}

// renderCoordinationSection renders recent coordination events
func (d *StatusDisplay) renderCoordinationSection() string {
	var content strings.Builder

	recentEvents := d.tracker.GetRecentActivity(5) // Last 5 events
	if len(recentEvents) == 0 {
		return ""
	}

	content.WriteString(d.styles.HeaderStyle.Render("📊 Recent Activity"))
	content.WriteString("\n")

	for _, event := range recentEvents {
		timestamp := event.Timestamp.Format("15:04:05")

		// Format event based on type
		var eventStr string
		switch event.EventType {
		case ActivityCoordination:
			eventStr = d.styles.CoordStyle.Render(fmt.Sprintf("🔗 %s", event.Description))
		case ActivityTaskStarted:
			eventStr = fmt.Sprintf("▶️ %s: %s", event.AgentID, event.Description)
		case ActivityTaskCompleted:
			eventStr = fmt.Sprintf("✅ %s: %s", event.AgentID, event.Description)
		case ActivityToolStarted:
			eventStr = d.styles.ToolIndicator.Render(fmt.Sprintf("🔨 %s: %s", event.AgentID, event.Description))
		case ActivityStateChanged:
			eventStr = fmt.Sprintf("🔄 %s: %s", event.AgentID, event.Description)
		default:
			eventStr = fmt.Sprintf("📝 %s: %s", event.AgentID, event.Description)
		}

		content.WriteString(fmt.Sprintf("   [%s] %s\n", timestamp, eventStr))
	}

	return content.String()
}

// RenderActivityStream renders a stream of recent activity events
func (d *StatusDisplay) RenderActivityStream(events []ActivityEvent) string {
	if len(events) == 0 {
		return "No recent activity."
	}

	var content strings.Builder
	content.WriteString(d.styles.HeaderStyle.Render("📈 Activity Stream"))
	content.WriteString("\n")

	for _, event := range events {
		timestamp := event.Timestamp.Format("15:04:05")
		content.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			timestamp, event.AgentID, event.Description))
	}

	return content.String()
}

// RenderProgressBar creates a visual progress bar
func (d *StatusDisplay) RenderProgressBar(progress float64) string {
	const barWidth = 20
	filled := int(progress * barWidth)

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}

	bar.WriteString("]")
	return d.styles.ProgressBar.Render(bar.String())
}

// getStatusIcon returns an appropriate icon for the agent state
func (d *StatusDisplay) getStatusIcon(state AgentState) string {
	switch state {
	case AgentIdle:
		return "🟢"
	case AgentThinking:
		return "🤔"
	case AgentWorking:
		return "⚙️"
	case AgentBlocked:
		return "⏳"
	case AgentOffline:
		return "⚫"
	default:
		return "⚪"
	}
}

// getStatusColor returns a color for the agent state
func (d *StatusDisplay) getStatusColor(state AgentState) lipgloss.Color {
	switch state {
	case AgentIdle:
		return lipgloss.Color("76")  // Green
	case AgentThinking:
		return lipgloss.Color("33")  // Blue
	case AgentWorking:
		return lipgloss.Color("214") // Orange
	case AgentBlocked:
		return lipgloss.Color("220") // Yellow
	case AgentOffline:
		return lipgloss.Color("8")   // Gray
	default:
		return lipgloss.Color("7")   // Light gray
	}
}

// ToggleDetailMode switches between compact and detailed view
func (d *StatusDisplay) ToggleDetailMode() {
	d.showDetails = !d.showDetails
}

// SelectAgent sets focus on a specific agent
func (d *StatusDisplay) SelectAgent(agentID string) {
	d.selectedAgent = agentID
}

// SetDimensions updates the display dimensions
func (d *StatusDisplay) SetDimensions(width, height int) {
	d.width = width
	d.height = height

	// Update agent panel width
	d.styles.AgentPanel = d.styles.AgentPanel.Width(width - 4)
}

// Update refreshes the display with current status
func (d *StatusDisplay) Update() {
	d.lastUpdate = time.Now()
}

// GetSelectedAgent returns the currently selected agent ID
func (d *StatusDisplay) GetSelectedAgent() string {
	return d.selectedAgent
}

// IsDetailMode returns true if showing detailed view
func (d *StatusDisplay) IsDetailMode() bool {
	return d.showDetails
}

// RenderCoordinationOverview renders a summary of multi-agent coordination
func (d *StatusDisplay) RenderCoordinationOverview() string {
	if d.tracker == nil {
		return "Coordination data unavailable"
	}

	summary := d.tracker.GetCoordinationSummary()

	var content strings.Builder
	content.WriteString(d.styles.HeaderStyle.Render("🤝 Coordination Overview"))
	content.WriteString("\n")

	content.WriteString(fmt.Sprintf("Total Agents: %d\n", summary["total_agents"]))
	content.WriteString(fmt.Sprintf("Active: %d\n", summary["active_agents"]))
	content.WriteString(fmt.Sprintf("Active Tools: %d\n", summary["active_tools"]))
	content.WriteString(fmt.Sprintf("Coordination Events: %d\n", summary["coordination_events"]))

	totalCost := summary["total_cost"].(float64)
	if totalCost > 0 {
		costStr := d.styles.CostStyle.Render(fmt.Sprintf("$%.4f", totalCost))
		content.WriteString(fmt.Sprintf("Total Cost: %s\n", costStr))
	}

	totalTasks := summary["total_tasks"].(int)
	if totalTasks > 0 {
		content.WriteString(fmt.Sprintf("Completed Tasks: %d\n", totalTasks))
	}

	return content.String()
}

// Note: min function is available from orchestrator_demo.go
