// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// IndicatorManager manages animated indicators for agents
type IndicatorManager struct {
	indicators map[string]*AgentIndicator
	styles     map[string]lipgloss.Style
	frameRate  time.Duration
}

// NewIndicatorManager creates a new indicator manager
func NewIndicatorManager() *IndicatorManager {
	return &IndicatorManager{
		indicators: make(map[string]*AgentIndicator),
		styles:     createIndicatorStyles(),
		frameRate:  100 * time.Millisecond,
	}
}

// createIndicatorStyles creates styles for indicators
func createIndicatorStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"working": lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")), // Orange
		"thinking": lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow
		"error": lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red
		"success": lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")), // Green
	}
}

// SetIndicator sets or updates an indicator for an agent
func (m *IndicatorManager) SetIndicator(agentID string, indicatorType IndicatorType, status AgentStatus) {
	m.indicators[agentID] = &AgentIndicator{
		Type:       indicatorType,
		AgentID:    agentID,
		Status:     status,
		Frame:      0,
		LastUpdate: time.Now(),
	}
}

// RemoveIndicator removes an indicator for an agent
func (m *IndicatorManager) RemoveIndicator(agentID string) {
	delete(m.indicators, agentID)
}

// Update advances all indicator animations
func (m *IndicatorManager) Update() {
	now := time.Now()
	for _, indicator := range m.indicators {
		if now.Sub(indicator.LastUpdate) >= m.frameRate {
			indicator.Frame++
			indicator.LastUpdate = now
		}
	}
}

// GetIndicator returns the current visual indicator for an agent
func (m *IndicatorManager) GetIndicator(agentID string) string {
	indicator, exists := m.indicators[agentID]
	if !exists {
		return ""
	}

	switch indicator.Type {
	case IndicatorSpinner:
		return m.getSpinnerFrame(indicator)
	case IndicatorPulse:
		return m.getPulseFrame(indicator)
	case IndicatorProgress:
		return m.getProgressFrame(indicator)
	case IndicatorDots:
		return m.getDotsFrame(indicator)
	default:
		return ""
	}
}

// getSpinnerFrame returns a spinner animation frame
func (m *IndicatorManager) getSpinnerFrame(indicator *AgentIndicator) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[indicator.Frame%len(frames)]
	
	style := m.getStyleForStatus(indicator.Status)
	return style.Render(frame)
}

// getPulseFrame returns a pulse animation frame
func (m *IndicatorManager) getPulseFrame(indicator *AgentIndicator) string {
	frames := []string{"◯", "◉", "●", "◉"}
	frame := frames[indicator.Frame%len(frames)]
	
	style := m.getStyleForStatus(indicator.Status)
	return style.Render(frame)
}

// getProgressFrame returns a progress bar animation frame
func (m *IndicatorManager) getProgressFrame(indicator *AgentIndicator) string {
	width := 10
	position := indicator.Frame % (width * 2)
	
	if position >= width {
		position = (width * 2) - position - 1
	}
	
	bar := strings.Repeat("─", position) + "●" + strings.Repeat("─", width-position-1)
	
	style := m.getStyleForStatus(indicator.Status)
	return fmt.Sprintf("[%s]", style.Render(bar))
}

// getDotsFrame returns a dots animation frame
func (m *IndicatorManager) getDotsFrame(indicator *AgentIndicator) string {
	dots := (indicator.Frame % 4)
	text := strings.Repeat(".", dots)
	padding := strings.Repeat(" ", 3-dots)
	
	style := m.getStyleForStatus(indicator.Status)
	return style.Render(text + padding)
}

// getStyleForStatus returns the appropriate style for a status
func (m *IndicatorManager) getStyleForStatus(status AgentStatus) lipgloss.Style {
	switch status {
	case StatusWorking:
		return m.styles["working"]
	case StatusThinking:
		return m.styles["thinking"]
	case StatusError:
		return m.styles["error"]
	default:
		return m.styles["success"]
	}
}

// FormatAgentWithIndicator formats an agent with its animated indicator
func (m *IndicatorManager) FormatAgentWithIndicator(info AgentInfo, display AgentDisplay) string {
	indicator := m.GetIndicator(info.ID)
	if indicator == "" {
		// No indicator, use static icon
		return display.FormatAgentCompact(info)
	}
	
	// Replace icon with animated indicator
	base := fmt.Sprintf("%s %s", indicator, info.Name)
	
	// Add status if not idle
	if info.Status != StatusIdle {
		base += fmt.Sprintf(" [%s]", info.Status)
	}
	
	// Add task count if any
	if info.TaskCount > 0 {
		base += fmt.Sprintf(" (%d)", info.TaskCount)
	}
	
	return base
}

// AnimatedAgentList creates an animated list of agents
func (m *IndicatorManager) AnimatedAgentList(agents []AgentInfo, display AgentDisplay) string {
	if len(agents) == 0 {
		return "No active agents"
	}
	
	var lines []string
	
	// Active agents with indicators
	for _, agent := range agents {
		if agent.Status == StatusWorking || agent.Status == StatusThinking {
			lines = append(lines, m.FormatAgentWithIndicator(agent, display))
		}
	}
	
	// Idle agents without indicators
	for _, agent := range agents {
		if agent.Status == StatusIdle {
			lines = append(lines, display.FormatAgentCompact(agent))
		}
	}
	
	// Error/offline agents
	for _, agent := range agents {
		if agent.Status == StatusError || agent.Status == StatusOffline {
			lines = append(lines, display.FormatAgentCompact(agent))
		}
	}
	
	return strings.Join(lines, "\n")
}

// GetFrameRate returns the frame rate for animations
func (m *IndicatorManager) GetFrameRate() time.Duration {
	return m.frameRate
}

// SetFrameRate sets the frame rate for animations
func (m *IndicatorManager) SetFrameRate(rate time.Duration) {
	m.frameRate = rate
}

// Clear removes all indicators
func (m *IndicatorManager) Clear() {
	m.indicators = make(map[string]*AgentIndicator)
}