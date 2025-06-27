// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lancekrogers/guild/internal/ui/chat/panes"
	"github.com/lancekrogers/guild/pkg/gerror"
	orchinterfaces "github.com/lancekrogers/guild/pkg/orchestrator/interfaces"
)

// StatusIntegration connects agent status tracking to the UI
type StatusIntegration struct {
	tracker    StatusTracker
	display    AgentDisplay
	indicators *IndicatorManager
	statusPane panes.StatusPane
	ctx        context.Context
}

// NewStatusIntegration creates a new status integration
func NewStatusIntegration(ctx context.Context, statusPane panes.StatusPane) (*StatusIntegration, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if statusPane == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "status pane cannot be nil", nil).
			WithComponent("StatusIntegration").
			WithOperation("NewStatusIntegration")
	}

	return &StatusIntegration{
		tracker:    NewStatusTracker(ctx),
		display:    NewAgentDisplay(),
		indicators: NewIndicatorManager(),
		statusPane: statusPane,
		ctx:        ctx,
	}, nil
}

// HandleOrchestratorEvent processes orchestrator events
func (si *StatusIntegration) HandleOrchestratorEvent(event orchinterfaces.Event) error {
	if err := si.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("StatusIntegration").
			WithOperation("HandleOrchestratorEvent")
	}

	switch event.Type {
	case orchinterfaces.EventTypeAgentCreated:
		return si.handleAgentCreated(event)
	case orchinterfaces.EventTypeAgentStarted:
		return si.handleAgentStarted(event)
	case orchinterfaces.EventTypeAgentStopped:
		return si.handleAgentStopped(event)
	case orchinterfaces.EventTypeTaskAssigned:
		return si.handleTaskAssigned(event)
	case orchinterfaces.EventTypeTaskStarted:
		return si.handleTaskStarted(event)
	case orchinterfaces.EventTypeTaskCompleted:
		return si.handleTaskCompleted(event)
	case orchinterfaces.EventTypeTaskFailed:
		return si.handleTaskFailed(event)
	}

	return nil
}

// Update handles Bubble Tea messages
func (si *StatusIntegration) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case time.Time:
		// Update animations
		si.indicators.Update()
		return si.updateStatusDisplay()

	case AgentRegisteredMsg:
		si.handleAgentRegisteredMsg(msg)
		return si.updateStatusDisplay()

	case AgentStatusChangedMsg:
		si.handleAgentStatusChangedMsg(msg)
		return si.updateStatusDisplay()

	case AgentUnregisteredMsg:
		si.handleAgentUnregisteredMsg(msg)
		return si.updateStatusDisplay()
	}

	return nil
}

// handleAgentCreated handles agent creation events
func (si *StatusIntegration) handleAgentCreated(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	agentName, _ := event.Data["agent_name"].(string)
	agentType, _ := event.Data["agent_type"].(string)

	if agentID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent ID missing from event", nil).
			WithComponent("StatusIntegration").
			WithOperation("handleAgentCreated")
	}

	info := AgentInfo{
		ID:        agentID,
		Name:      agentName,
		Type:      agentType,
		Status:    StatusStarting,
		StartTime: event.Timestamp,
		LastSeen:  event.Timestamp,
	}

	return si.tracker.RegisterAgent(info)
}

// handleAgentStarted handles agent start events
func (si *StatusIntegration) handleAgentStarted(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	if agentID == "" {
		return nil
	}

	err := si.tracker.UpdateAgentStatus(agentID, StatusIdle, "Agent started")
	if err != nil {
		return err
	}

	// Set spinning indicator for newly started agent
	si.indicators.SetIndicator(agentID, IndicatorSpinner, StatusIdle)

	// Update UI
	si.updateAgentInUI(agentID)

	return nil
}

// handleAgentStopped handles agent stop events
func (si *StatusIntegration) handleAgentStopped(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	if agentID == "" {
		return nil
	}

	err := si.tracker.UpdateAgentStatus(agentID, StatusOffline, "Agent stopped")
	if err != nil {
		return err
	}

	// Remove indicator
	si.indicators.RemoveIndicator(agentID)

	// Update UI
	si.updateAgentInUI(agentID)

	return nil
}

// handleTaskAssigned handles task assignment events
func (si *StatusIntegration) handleTaskAssigned(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	taskID, _ := event.Data["task_id"].(string)

	if agentID == "" {
		return nil
	}

	// Update agent info
	if info, err := si.tracker.GetAgentInfo(agentID); err == nil {
		info.CurrentTask = taskID
		info.TaskCount++
		si.tracker.RegisterAgent(*info) // Update
	}

	return nil
}

// handleTaskStarted handles task start events
func (si *StatusIntegration) handleTaskStarted(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	taskID, _ := event.Data["task_id"].(string)

	if agentID == "" {
		return nil
	}

	err := si.tracker.UpdateAgentStatus(agentID, StatusWorking, "Task started: "+taskID)
	if err != nil {
		return err
	}

	// Set working indicator
	si.indicators.SetIndicator(agentID, IndicatorProgress, StatusWorking)

	// Update UI
	si.updateAgentInUI(agentID)

	return nil
}

// handleTaskCompleted handles task completion events
func (si *StatusIntegration) handleTaskCompleted(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)

	if agentID == "" {
		return nil
	}

	err := si.tracker.UpdateAgentStatus(agentID, StatusIdle, "Task completed")
	if err != nil {
		return err
	}

	// Update agent info
	if info, err := si.tracker.GetAgentInfo(agentID); err == nil {
		info.CurrentTask = ""
		si.tracker.RegisterAgent(*info) // Update
	}

	// Set idle indicator
	si.indicators.RemoveIndicator(agentID)

	// Update UI
	si.updateAgentInUI(agentID)

	return nil
}

// handleTaskFailed handles task failure events
func (si *StatusIntegration) handleTaskFailed(event orchinterfaces.Event) error {
	agentID, _ := event.Data["agent_id"].(string)
	errorMsg, _ := event.Data["error"].(string)

	if agentID == "" {
		return nil
	}

	err := si.tracker.UpdateAgentStatus(agentID, StatusError, "Task failed: "+errorMsg)
	if err != nil {
		return err
	}

	// Remove indicator
	si.indicators.RemoveIndicator(agentID)

	// Update UI
	si.updateAgentInUI(agentID)

	return nil
}

// updateAgentInUI updates the agent status in the UI
func (si *StatusIntegration) updateAgentInUI(agentID string) {
	info, err := si.tracker.GetAgentInfo(agentID)
	if err != nil {
		return
	}

	// Update status pane
	si.statusPane.SetAgentStatus(agentID, string(info.Status))
}

// updateStatusDisplay updates the complete status display
func (si *StatusIntegration) updateStatusDisplay() tea.Cmd {
	agents, err := si.tracker.GetAllAgents()
	if err != nil {
		return nil
	}

	// Update each agent in the status pane
	for _, agent := range agents {
		// Get animated display
		display := si.indicators.FormatAgentWithIndicator(agent, si.display)
		si.statusPane.SetAgentStatus(agent.ID, display)
	}

	// Update stats
	stats := si.tracker.GetStats()
	si.statusPane.SetSystemStats(panes.SystemStats{
		ActiveAgents:  stats.ActiveAgents,
		TotalMessages: stats.TotalTasks,
	})

	return nil
}

// GetTracker returns the status tracker
func (si *StatusIntegration) GetTracker() StatusTracker {
	return si.tracker
}

// GetDisplay returns the agent display
func (si *StatusIntegration) GetDisplay() AgentDisplay {
	return si.display
}

// GetIndicators returns the indicator manager
func (si *StatusIntegration) GetIndicators() *IndicatorManager {
	return si.indicators
}

// Message types for UI communication

// AgentRegisteredMsg indicates a new agent was registered
type AgentRegisteredMsg struct {
	Info AgentInfo
}

// AgentStatusChangedMsg indicates an agent's status changed
type AgentStatusChangedMsg struct {
	AgentID   string
	OldStatus AgentStatus
	NewStatus AgentStatus
	Reason    string
}

// AgentUnregisteredMsg indicates an agent was unregistered
type AgentUnregisteredMsg struct {
	AgentID string
}

// handleAgentRegisteredMsg handles agent registration messages
func (si *StatusIntegration) handleAgentRegisteredMsg(msg AgentRegisteredMsg) {
	si.tracker.RegisterAgent(msg.Info)

	// Set initial indicator if agent is active
	if msg.Info.Status == StatusWorking || msg.Info.Status == StatusThinking {
		si.indicators.SetIndicator(msg.Info.ID, IndicatorSpinner, msg.Info.Status)
	}
}

// handleAgentStatusChangedMsg handles status change messages
func (si *StatusIntegration) handleAgentStatusChangedMsg(msg AgentStatusChangedMsg) {
	si.tracker.UpdateAgentStatus(msg.AgentID, msg.NewStatus, msg.Reason)

	// Update indicator based on new status
	switch msg.NewStatus {
	case StatusWorking:
		si.indicators.SetIndicator(msg.AgentID, IndicatorProgress, StatusWorking)
	case StatusThinking:
		si.indicators.SetIndicator(msg.AgentID, IndicatorDots, StatusThinking)
	case StatusIdle, StatusOffline, StatusError:
		si.indicators.RemoveIndicator(msg.AgentID)
	}
}

// handleAgentUnregisteredMsg handles agent unregistration messages
func (si *StatusIntegration) handleAgentUnregisteredMsg(msg AgentUnregisteredMsg) {
	si.tracker.UnregisterAgent(msg.AgentID)
	si.indicators.RemoveIndicator(msg.AgentID)
}
