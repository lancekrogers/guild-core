// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package panes

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guild-framework/guild-core/internal/ui/chat/common/layout"
	"github.com/guild-framework/guild-core/internal/ui/chat/components"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// StatusPane displays system status, agent information, and notifications
type StatusPane interface {
	layout.PaneInterface

	// Status management
	UpdateStatus(message, level string)
	SetAgentStatus(agentID, status string)
	SetConnectionStatus(connected bool)

	// Notifications
	AddNotification(message, level string)
	ClearNotifications()

	// System info
	SetSessionInfo(sessionID, campaignID string)
	SetSystemStats(stats SystemStats)

	// Display modes
	SetCompactMode(compact bool)
	SetShowAgents(show bool)
	SetShowStats(show bool)

	// Completions display
	ShowCompletions(completions []string, selectedIndex int)
	HideCompletions()

	// Mode indicators
	SetVimMode(mode string)
	SetAutoAcceptMode(enabled bool)
	SetInputMode(mode string) // normal, insert, visual, etc.

	// Commission progress
	ShowCommissionProgress(progress *CommissionProgress)
	HideCommissionProgress()
	UpdateCommissionProgress(stage PlanningStage, progress float64, status string)
}

// SystemStats represents system statistics
type SystemStats struct {
	ActiveAgents    int
	TotalMessages   int
	ActiveTools     int
	Uptime          time.Duration
	MemoryUsage     string
	ConnectionState string
}

// Notification represents a status notification
type Notification struct {
	Message   string
	Level     string // info, warning, error, success
	Timestamp time.Time
}

// AgentStatus represents an agent's current status
type AgentStatus struct {
	ID        string
	Status    string // idle, thinking, working, error, offline
	LastSeen  time.Time
	TaskCount int
}

// PlanningStage represents commission planning stages
type PlanningStage int

const (
	StageIntroduction PlanningStage = iota
	StageProjectType
	StageRequirements
	StageTechnology
	StageConstraints
	StageSummary
)

// CommissionProgress represents commission planning progress
type CommissionProgress struct {
	Stage    PlanningStage
	Progress float64
	Status   string
	Details  map[string]interface{}
}

// String returns a human-readable stage name
func (ps PlanningStage) String() string {
	stages := []string{
		"Introduction",
		"Project Type",
		"Requirements",
		"Technology",
		"Constraints",
		"Summary",
	}
	if int(ps) < len(stages) {
		return stages[ps]
	}
	return "Unknown"
}

// Icon returns an emoji icon for the stage
func (ps PlanningStage) Icon() string {
	icons := []string{
		"📋",  // Introduction
		"🎯",  // Project Type
		"📝",  // Requirements
		"⚙️", // Technology
		"⚠️", // Constraints
		"✅",  // Summary
	}
	if int(ps) < len(icons) {
		return icons[ps]
	}
	return "❓"
}

// statusPaneImpl implements the StatusPane interface
type statusPaneImpl struct {
	*layout.BasePane

	// Status information
	currentStatus    string
	currentLevel     string
	notifications    []Notification
	agentStatuses    map[string]AgentStatus
	connectionStatus bool

	// Session information
	sessionID   string
	campaignID  string
	systemStats SystemStats

	// Display settings
	compactMode bool
	showAgents  bool
	showStats   bool

	// Update tracking
	lastUpdate time.Time

	// Styling
	statusStyles map[string]lipgloss.Style

	// Context
	ctx context.Context

	// Completions
	showingCompletions bool
	completions        []string
	completionIndex    int

	// Mode indicators
	vimMode        string
	autoAcceptMode bool
	inputMode      string

	// Commission progress
	showingCommissionProgress bool
	commissionProgress        *CommissionProgress

	// Status transitions
	statusAnimator    *components.StatusAnimator
	transitionManager *components.TransitionManager
}

// NewStatusPane creates a new status pane
func NewStatusPane(width, height int) (StatusPane, error) {
	if width < 20 || height < 1 {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "status pane dimensions too small: %dx%d", width, height).
			WithComponent("panes.status").
			WithOperation("NewStatusPane")
	}

	ctx := context.Background()
	basePane := layout.NewBasePane(ctx, "status", width, height)
	basePane.SetConstraints(layout.StatusPaneConstraints())
	basePane.ApplyMinimalStyling() // Status bar doesn't need borders

	pane := &statusPaneImpl{
		BasePane:          basePane,
		currentStatus:     "Ready",
		currentLevel:      "info",
		notifications:     make([]Notification, 0),
		agentStatuses:     make(map[string]AgentStatus),
		connectionStatus:  true,
		compactMode:       true,
		showAgents:        true,
		showStats:         false,
		lastUpdate:        time.Now(),
		statusStyles:      createStatusStyles(),
		ctx:               ctx,
		statusAnimator:    components.NewStatusAnimator(ctx),
		transitionManager: components.NewTransitionManager(ctx),
	}

	return pane, nil
}

// createStatusStyles creates styling for different status levels
func createStatusStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"info": lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")), // Purple
		"success": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
		"warning": lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
		"agent_idle": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
		"agent_thinking": lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		"agent_working": lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange
		"agent_error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
		"agent_offline": lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Gray
		"connection_ok": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
		"connection_error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
	}
}

// UpdateStatus updates the main status message with smooth transitions
func (sp *statusPaneImpl) UpdateStatus(message, level string) {
	// Start transition if status actually changed
	if sp.currentStatus != message {
		// Trigger smooth transition to new status
		if err := sp.statusAnimator.TransitionTo(message, 300*time.Millisecond); err != nil {
			// If transition fails, just update directly
			sp.currentStatus = message
		}
	}

	sp.currentLevel = level
	sp.lastUpdate = time.Now()
}

// SetAgentStatus updates the status of a specific agent with smooth transitions
func (sp *statusPaneImpl) SetAgentStatus(agentID, status string) {
	existing, exists := sp.agentStatuses[agentID]
	if exists {
		// Start transition for agent status change
		if existing.Status != status {
			transitionKey := fmt.Sprintf("agent_%s", agentID)
			if err := sp.transitionManager.StartTransition(transitionKey, existing.Status, status, 250*time.Millisecond); err != nil {
				// If transition fails, update directly
			}
		}
		existing.Status = status
		existing.LastSeen = time.Now()
		sp.agentStatuses[agentID] = existing
	} else {
		sp.agentStatuses[agentID] = AgentStatus{
			ID:        agentID,
			Status:    status,
			LastSeen:  time.Now(),
			TaskCount: 0,
		}
	}
}

// SetConnectionStatus updates the connection status
func (sp *statusPaneImpl) SetConnectionStatus(connected bool) {
	sp.connectionStatus = connected
}

// AddNotification adds a new notification
func (sp *statusPaneImpl) AddNotification(message, level string) {
	notification := Notification{
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
	}

	sp.notifications = append(sp.notifications, notification)

	// Keep only recent notifications (last 10)
	maxNotifications := 10
	if len(sp.notifications) > maxNotifications {
		sp.notifications = sp.notifications[len(sp.notifications)-maxNotifications:]
	}
}

// ClearNotifications removes all notifications
func (sp *statusPaneImpl) ClearNotifications() {
	sp.notifications = make([]Notification, 0)
}

// SetSessionInfo updates session information
func (sp *statusPaneImpl) SetSessionInfo(sessionID, campaignID string) {
	sp.sessionID = sessionID
	sp.campaignID = campaignID
}

// SetSystemStats updates system statistics
func (sp *statusPaneImpl) SetSystemStats(stats SystemStats) {
	sp.systemStats = stats
}

// SetCompactMode toggles compact display mode
func (sp *statusPaneImpl) SetCompactMode(compact bool) {
	sp.compactMode = compact
}

// SetShowAgents toggles agent status display
func (sp *statusPaneImpl) SetShowAgents(show bool) {
	sp.showAgents = show
}

// SetShowStats toggles system stats display
func (sp *statusPaneImpl) SetShowStats(show bool) {
	sp.showStats = show
}

// ShowCompletions shows completion suggestions
func (sp *statusPaneImpl) ShowCompletions(completions []string, selectedIndex int) {
	sp.showingCompletions = true
	sp.completions = completions
	sp.completionIndex = selectedIndex
}

// HideCompletions hides completion suggestions
func (sp *statusPaneImpl) HideCompletions() {
	sp.showingCompletions = false
	sp.completions = []string{}
	sp.completionIndex = 0
}

// SetVimMode sets the current vim mode
func (sp *statusPaneImpl) SetVimMode(mode string) {
	sp.vimMode = mode
}

// SetAutoAcceptMode sets the auto-accept mode
func (sp *statusPaneImpl) SetAutoAcceptMode(enabled bool) {
	sp.autoAcceptMode = enabled
}

// SetInputMode sets the current input mode
func (sp *statusPaneImpl) SetInputMode(mode string) {
	sp.inputMode = mode
}

// Update handles Bubble Tea messages
func (sp *statusPaneImpl) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update base pane first
	_, cmd := sp.BasePane.Update(msg)

	// Update transitions and animations
	if sp.statusAnimator != nil {
		if err := sp.statusAnimator.Update(); err != nil {
			// Log error but continue
		}
	}
	if sp.transitionManager != nil {
		if err := sp.transitionManager.UpdateAll(); err != nil {
			// Log error but continue
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sp.Resize(msg.Width, msg.Height)
	case StatusUpdateMsg:
		sp.UpdateStatus(msg.Message, msg.Level)
	case AgentStatusMsg:
		sp.SetAgentStatus(msg.AgentID, msg.Status)
	case NotificationMsg:
		sp.AddNotification(msg.Message, msg.Level)
	case time.Time:
		// Periodic updates for uptime, etc.
		sp.updateSystemStats()
	}

	return sp, cmd
}

// View renders the status pane
func (sp *statusPaneImpl) View() string {
	rect := sp.GetRect()

	if sp.compactMode {
		return sp.renderCompactStatus(rect.Width)
	}

	return sp.renderDetailedStatus(rect.Width)
}

// renderCompactStatus renders a single-line status bar
func (sp *statusPaneImpl) renderCompactStatus(width int) string {
	var parts []string

	// Connection indicator
	if sp.connectionStatus {
		indicator := sp.statusStyles["connection_ok"].Render("●")
		parts = append(parts, indicator)
	} else {
		indicator := sp.statusStyles["connection_error"].Render("●")
		parts = append(parts, indicator)
	}

	// Main status with smooth transitions
	statusText := sp.currentStatus
	if sp.statusAnimator != nil && sp.statusAnimator.IsTransitioning() {
		// Use animated status if transitioning
		statusText = sp.statusAnimator.View()
	}

	statusStyle, exists := sp.statusStyles[sp.currentLevel]
	if !exists {
		statusStyle = sp.statusStyles["info"]
	}
	parts = append(parts, statusStyle.Render(statusText))

	// Agent summary
	if sp.showAgents && len(sp.agentStatuses) > 0 {
		agentSummary := sp.getAgentSummary()
		parts = append(parts, agentSummary)
	}

	// Session info
	if sp.sessionID != "" {
		sessionInfo := fmt.Sprintf("Session: %s", sp.sessionID[:8])
		if sp.campaignID != "" {
			sessionInfo = fmt.Sprintf("Campaign: %s | %s", sp.campaignID, sessionInfo)
		}
		parts = append(parts, sessionInfo)
	}

	// Time
	timeStr := time.Now().Format("15:04:05")
	parts = append(parts, timeStr)

	// Join and truncate to fit width
	status := strings.Join(parts, " | ")
	if len(status) > width-2 {
		status = status[:width-5] + "..."
	}

	// Apply background style
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark gray background
		Foreground(lipgloss.Color("254")). // Light text
		Padding(0, 1).
		Width(width)

	return style.Render(status)
}

// renderDetailedStatus renders a multi-line detailed status
func (sp *statusPaneImpl) renderDetailedStatus(width int) string {
	var lines []string

	// Commission progress (highest priority, shows at the top)
	if sp.showingCommissionProgress && sp.commissionProgress != nil {
		progressLines := sp.renderCommissionProgress(width)
		lines = append(lines, progressLines...)
	}

	// Completions (second priority, shows at the top of status area)
	if sp.showingCompletions && len(sp.completions) > 0 && !sp.showingCommissionProgress {
		completionLines := sp.renderCompletions(width)
		lines = append(lines, completionLines...)
	}

	// Mode indicators line
	modeLine := sp.renderModeLine(width)
	lines = append(lines, modeLine)

	// Main status line
	statusLine := sp.renderCompactStatus(width)
	lines = append(lines, statusLine)

	// Only show other details if we have space and no completions
	if !sp.showingCompletions {
		// Agent details
		if sp.showAgents && len(sp.agentStatuses) > 0 && len(lines) < 3 {
			agentLines := sp.renderAgentDetails(width)
			if len(lines)+len(agentLines) <= 4 {
				lines = append(lines, agentLines...)
			}
		}

		// System stats
		if sp.showStats && len(lines) < 3 {
			statsLines := sp.renderSystemStats(width)
			if len(lines)+len(statsLines) <= 4 {
				lines = append(lines, statsLines...)
			}
		}

		// Recent notifications
		if len(sp.notifications) > 0 && len(lines) < 3 {
			notificationLines := sp.renderNotifications(width)
			if len(lines)+len(notificationLines) <= 4 {
				lines = append(lines, notificationLines...)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// getAgentSummary returns a summary of agent statuses
func (sp *statusPaneImpl) getAgentSummary() string {
	if len(sp.agentStatuses) == 0 {
		return "No agents"
	}

	statusCounts := make(map[string]int)
	for _, agent := range sp.agentStatuses {
		statusCounts[agent.Status]++
	}

	var parts []string
	for status, count := range statusCounts {
		icon := sp.getAgentStatusIcon(status)
		parts = append(parts, fmt.Sprintf("%s%d", icon, count))
	}

	return fmt.Sprintf("Agents: %s", strings.Join(parts, " "))
}

// renderAgentDetails renders detailed agent information with smooth transitions
func (sp *statusPaneImpl) renderAgentDetails(width int) []string {
	var lines []string

	for _, agent := range sp.agentStatuses {
		icon := sp.getAgentStatusIcon(agent.Status)
		style, exists := sp.statusStyles["agent_"+agent.Status]
		if !exists {
			style = sp.statusStyles["info"]
		}

		// Check for agent status transition
		statusText := agent.Status
		if sp.transitionManager != nil {
			transitionKey := fmt.Sprintf("agent_%s", agent.ID)
			if sp.transitionManager.IsTransitioning(transitionKey) {
				statusText = sp.transitionManager.GetTransition(transitionKey)
			}
		}

		agentLine := fmt.Sprintf("%s %s: %s", icon, agent.ID, statusText)
		if agent.TaskCount > 0 {
			agentLine += fmt.Sprintf(" (%d tasks)", agent.TaskCount)
		}

		styledLine := style.Render(agentLine)
		lines = append(lines, styledLine)
	}

	return lines
}

// renderSystemStats renders system statistics
func (sp *statusPaneImpl) renderSystemStats(width int) []string {
	var lines []string

	statsLine := fmt.Sprintf("📊 Messages: %d | Tools: %d | Uptime: %s",
		sp.systemStats.TotalMessages,
		sp.systemStats.ActiveTools,
		sp.formatDuration(sp.systemStats.Uptime))

	if sp.systemStats.MemoryUsage != "" {
		statsLine += fmt.Sprintf(" | Memory: %s", sp.systemStats.MemoryUsage)
	}

	lines = append(lines, statsLine)

	return lines
}

// renderNotifications renders recent notifications
func (sp *statusPaneImpl) renderNotifications(width int) []string {
	var lines []string

	// Show only the most recent notifications (last 3 in detailed mode)
	maxNotifications := 3
	start := len(sp.notifications) - maxNotifications
	if start < 0 {
		start = 0
	}

	for i := start; i < len(sp.notifications); i++ {
		notification := sp.notifications[i]
		style, exists := sp.statusStyles[notification.Level]
		if !exists {
			style = sp.statusStyles["info"]
		}

		timeStr := notification.Timestamp.Format("15:04")
		notificationLine := fmt.Sprintf("[%s] %s", timeStr, notification.Message)

		styledLine := style.Render(notificationLine)
		lines = append(lines, styledLine)
	}

	return lines
}

// getAgentStatusIcon returns an icon for the agent status
func (sp *statusPaneImpl) getAgentStatusIcon(status string) string {
	switch status {
	case "idle":
		return "🟢"
	case "thinking":
		return "🤔"
	case "working":
		return "⚙️"
	case "error":
		return "🔴"
	case "offline":
		return "⚫"
	default:
		return "⚪"
	}
}

// formatDuration formats a duration for display
func (sp *statusPaneImpl) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		hours := int(d.Hours())
		minutes := int((d - time.Duration(hours)*time.Hour).Minutes())
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}

// updateSystemStats updates internal system statistics
func (sp *statusPaneImpl) updateSystemStats() {
	// Update uptime
	sp.systemStats.Uptime = time.Since(sp.lastUpdate)

	// Update active agent count
	sp.systemStats.ActiveAgents = 0
	for _, agent := range sp.agentStatuses {
		if agent.Status != "offline" {
			sp.systemStats.ActiveAgents++
		}
	}
}

// GetNotificationCount returns the number of unread notifications
func (sp *statusPaneImpl) GetNotificationCount() int {
	return len(sp.notifications)
}

// GetAgentCount returns the number of tracked agents
func (sp *statusPaneImpl) GetAgentCount() int {
	return len(sp.agentStatuses)
}

// GetActiveAgentCount returns the number of active agents
func (sp *statusPaneImpl) GetActiveAgentCount() int {
	count := 0
	for _, agent := range sp.agentStatuses {
		if agent.Status != "offline" {
			count++
		}
	}
	return count
}

// GetStats returns statistics about the status pane
func (sp *statusPaneImpl) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["current_status"] = sp.currentStatus
	stats["current_level"] = sp.currentLevel
	stats["notification_count"] = len(sp.notifications)
	stats["agent_count"] = len(sp.agentStatuses)
	stats["active_agent_count"] = sp.GetActiveAgentCount()
	stats["connection_status"] = sp.connectionStatus
	stats["compact_mode"] = sp.compactMode
	stats["show_agents"] = sp.showAgents
	stats["show_stats"] = sp.showStats
	stats["session_id"] = sp.sessionID
	stats["campaign_id"] = sp.campaignID
	stats["last_update"] = sp.lastUpdate

	// Agent status breakdown
	statusCounts := make(map[string]int)
	for _, agent := range sp.agentStatuses {
		statusCounts[agent.Status]++
	}
	stats["agent_status_counts"] = statusCounts

	return stats
}

// SetTheme applies a visual theme to the status pane
func (sp *statusPaneImpl) SetTheme(theme string) {
	switch theme {
	case "medieval":
		sp.statusStyles = createMedievalStatusStyles()
	case "minimal":
		sp.statusStyles = createMinimalStatusStyles()
	default:
		sp.statusStyles = createStatusStyles()
	}
}

// createMedievalStatusStyles creates medieval-themed status styles
func createMedievalStatusStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"info": lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange/amber
		"success": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
		"warning": lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
		// Add more medieval-themed styles...
	}
}

// createMinimalStatusStyles creates minimal status styles
func createMinimalStatusStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"info": lipgloss.NewStyle().
			Foreground(lipgloss.Color("254")), // Light gray
		"success": lipgloss.NewStyle().
			Foreground(lipgloss.Color("254")),
		"warning": lipgloss.NewStyle().
			Foreground(lipgloss.Color("254")),
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("254")),
		// All minimal - same color
	}
}

// renderCompletions renders completion suggestions
func (sp *statusPaneImpl) renderCompletions(width int) []string {
	var lines []string

	// Show up to 3 completions
	maxCompletions := 3
	if len(sp.completions) < maxCompletions {
		maxCompletions = len(sp.completions)
	}

	completionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("247")) // Gray
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")). // Green
		Bold(true)

	for i := 0; i < maxCompletions; i++ {
		prefix := "  "
		if i == sp.completionIndex {
			prefix = "▸ "
		}

		completion := sp.completions[i]
		if len(completion) > width-4 {
			completion = completion[:width-7] + "..."
		}

		var line string
		if i == sp.completionIndex {
			line = selectedStyle.Render(prefix + completion)
		} else {
			line = completionStyle.Render(prefix + completion)
		}

		lines = append(lines, line)
	}

	// Add indicator if there are more completions
	if len(sp.completions) > maxCompletions {
		moreText := fmt.Sprintf("  ... %d more", len(sp.completions)-maxCompletions)
		lines = append(lines, completionStyle.Render(moreText))
	}

	return lines
}

// renderModeLine renders the mode indicators line
func (sp *statusPaneImpl) renderModeLine(width int) string {
	var parts []string

	// Left side: Input mode and vim mode
	if sp.inputMode != "" {
		modeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")). // Purple
			Bold(true)

		modeText := sp.inputMode
		if sp.vimMode != "" && sp.vimMode != "normal" {
			modeText = fmt.Sprintf("%s:%s", sp.inputMode, sp.vimMode)
		}

		parts = append(parts, modeStyle.Render(modeText))
	}

	// Center: Auto-accept indicator
	if sp.autoAcceptMode {
		autoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")). // Green
			Bold(true)
		parts = append(parts, autoStyle.Render("AUTO"))
	}

	// Right side: Status messages (aligned to right)
	rightParts := []string{}

	// Add any additional status indicators here
	if sp.connectionStatus {
		rightParts = append(rightParts, "Connected")
	}

	// Join left parts
	leftContent := strings.Join(parts, " | ")
	rightContent := strings.Join(rightParts, " | ")

	// Calculate padding
	totalLen := len(leftContent) + len(rightContent)
	if totalLen < width {
		padding := width - totalLen - 2
		if padding > 0 {
			leftContent = leftContent + strings.Repeat(" ", padding) + rightContent
		}
	}

	// Apply background style
	lineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237")). // Slightly lighter than status bar
		Foreground(lipgloss.Color("254")).
		Width(width)

	return lineStyle.Render(leftContent)
}

// renderCommissionProgress renders commission planning progress
func (sp *statusPaneImpl) renderCommissionProgress(width int) []string {
	if sp.commissionProgress == nil {
		return []string{}
	}

	var lines []string
	cp := sp.commissionProgress

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141")) // Purple
	headerLine := headerStyle.Render("📋 Commission Planning")
	lines = append(lines, headerLine)

	// Progress bar
	progressWidth := width - 4 // Account for padding
	if progressWidth < 10 {
		progressWidth = 10
	}

	filled := int(cp.Progress * float64(progressWidth))
	if filled < 0 {
		filled = 0
	} else if filled > progressWidth {
		filled = progressWidth
	}

	progressBar := "["
	for i := 0; i < progressWidth; i++ {
		if i < filled {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}
	progressBar += fmt.Sprintf("] %.0f%%", cp.Progress*100)

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")) // Green
	lines = append(lines, progressStyle.Render(progressBar))

	// Stage indicators
	stages := []PlanningStage{
		StageIntroduction, StageProjectType, StageRequirements,
		StageTechnology, StageConstraints, StageSummary,
	}

	var stageLines []string
	for _, stage := range stages {
		icon := "○"
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray

		if stage < cp.Stage {
			icon = "✓"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("82")) // Green
		} else if stage == cp.Stage {
			icon = "▶"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Yellow
		}

		stageLine := fmt.Sprintf("%s %s %s", icon, stage.Icon(), stage.String())
		stageLines = append(stageLines, style.Render(stageLine))
	}

	// Only show 3 stages at a time for space efficiency
	startIdx := int(cp.Stage) - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(stageLines) {
		endIdx = len(stageLines)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}

	for i := startIdx; i < endIdx; i++ {
		lines = append(lines, stageLines[i])
	}

	// Current status
	if cp.Status != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("147")) // Light purple
		statusLine := statusStyle.Render(fmt.Sprintf("Status: %s", cp.Status))
		lines = append(lines, statusLine)
	}

	return lines
}

// ShowCommissionProgress shows commission planning progress
func (sp *statusPaneImpl) ShowCommissionProgress(progress *CommissionProgress) {
	sp.showingCommissionProgress = true
	sp.commissionProgress = progress
}

// HideCommissionProgress hides commission planning progress
func (sp *statusPaneImpl) HideCommissionProgress() {
	sp.showingCommissionProgress = false
	sp.commissionProgress = nil
}

// UpdateCommissionProgress updates commission planning progress
func (sp *statusPaneImpl) UpdateCommissionProgress(stage PlanningStage, progress float64, status string) {
	if sp.commissionProgress == nil {
		sp.commissionProgress = &CommissionProgress{}
	}
	sp.commissionProgress.Stage = stage
	sp.commissionProgress.Progress = progress
	sp.commissionProgress.Status = status
	sp.showingCommissionProgress = true
}
