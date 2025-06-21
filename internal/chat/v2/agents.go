// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package v2

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
)

// AgentTarget represents a parsed agent mention with routing information
type AgentTarget struct {
	ID      string
	Message string
	IsBroadcast bool
}

// AgentInfo holds agent status and metadata
type AgentInfo struct {
	ID           string
	Name         string
	Status       pb.AgentStatus_State
	LastActivity time.Time
	Capabilities []string
}

// AgentRouter handles agent communication routing and @mention parsing
type AgentRouter struct {
	ctx         context.Context
	guildClient pb.GuildClient
	agents      map[string]*AgentInfo
}

// NewAgentRouter creates a new agent router
func NewAgentRouter(ctx context.Context, guildClient pb.GuildClient) *AgentRouter {
	return &AgentRouter{
		ctx:         ctx,
		guildClient: guildClient,
		agents:      make(map[string]*AgentInfo),
	}
}

// ParseInput analyzes user input and determines if it contains agent mentions
func (ar *AgentRouter) ParseInput(input string) (*AgentTarget, error) {
	input = strings.TrimSpace(input)
	
	// Check for @mention at start of message
	if !strings.HasPrefix(input, "@") {
		return nil, nil // Not an agent mention
	}
	
	// Split into mention and message parts
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 1 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid agent mention format", nil).
			WithComponent("chat.v2.agents").
			WithOperation("ParseInput")
	}
	
	mention := strings.TrimPrefix(parts[0], "@")
	message := ""
	if len(parts) > 1 {
		message = parts[1]
	}
	
	// Handle special cases
	switch mention {
	case "all":
		return &AgentTarget{
			ID:          "all",
			Message:     message,
			IsBroadcast: true,
		}, nil
	default:
		// Validate agent exists
		if !ar.AgentExists(mention) {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found", mention).
				WithComponent("chat.v2.agents").
				WithOperation("ParseInput").
				WithDetails("available_agents", ar.GetAvailableAgents())
		}
		
		return &AgentTarget{
			ID:          mention,
			Message:     message,
			IsBroadcast: false,
		}, nil
	}
}

// SendToAgent sends a message to a specific agent via gRPC
func (ar *AgentRouter) SendToAgent(agentID, message string) tea.Cmd {
	return func() tea.Msg {
		// Make gRPC call to send message to specific agent
		req := &pb.AgentMessageRequest{
			AgentId: agentID,
			Message: message,
		}
		
		resp, err := ar.guildClient.SendMessageToAgent(ar.ctx, req)
		if err != nil {
			return AgentErrorMsg{
				AgentID: agentID,
				Error:   gerror.Wrap(err, gerror.ErrCodeConnection, "failed to send message to agent").
					WithComponent("chat.v2.agents").
					WithOperation("SendToAgent").
					WithDetails("agent_id", agentID),
			}
		}
		
		return AgentResponseMsg{
			AgentID:   agentID,
			Content:   resp.Response,
			MessageID: generateMessageID(), // Generate our own ID since proto doesn't have one
			Timestamp: time.Now(),
		}
	}
}

// BroadcastToAll sends a message to all available agents
func (ar *AgentRouter) BroadcastToAll(message string) tea.Cmd {
	return func() tea.Msg {
		// Since there's no direct broadcast method, send to all agents individually
		// First get the list of available agents
		listResp, err := ar.guildClient.ListAvailableAgents(ar.ctx, &pb.ListAgentsRequest{})
		if err != nil {
			return AgentErrorMsg{
				AgentID: "all",
				Error:   gerror.Wrap(err, gerror.ErrCodeConnection, "failed to list agents for broadcast").
					WithComponent("chat.v2.agents").
					WithOperation("BroadcastToAll"),
			}
		}
		
		// Send message to each agent (simplified for now - in a full implementation 
		// this would be done concurrently and collected)
		var responses []*pb.AgentMessageResponse
		for _, agent := range listResp.Agents {
			req := &pb.AgentMessageRequest{
				AgentId: agent.Id,
				Message: message,
			}
			
			resp, err := ar.guildClient.SendMessageToAgent(ar.ctx, req)
			if err == nil {
				responses = append(responses, resp)
			}
		}
		
		return BroadcastResponseMsg{
			Responses: responses,
			MessageID: generateMessageID(),
			Timestamp: time.Now(),
		}
	}
}

// RefreshAgentList retrieves current agent list from the guild daemon
func (ar *AgentRouter) RefreshAgentList() tea.Cmd {
	return func() tea.Msg {
		resp, err := ar.guildClient.ListAvailableAgents(ar.ctx, &pb.ListAgentsRequest{})
		if err != nil {
			return AgentErrorMsg{
				AgentID: "system",
				Error:   gerror.Wrap(err, gerror.ErrCodeConnection, "failed to refresh agent list").
					WithComponent("chat.v2.agents").
					WithOperation("RefreshAgentList"),
			}
		}
		
		// Update local agent cache
		for _, agent := range resp.Agents {
			ar.agents[agent.Id] = &AgentInfo{
				ID:           agent.Id,
				Name:         agent.Name,
				Status:       pb.AgentStatus_IDLE, // Default status, will be updated separately
				LastActivity: time.Now(),
				Capabilities: agent.Capabilities,
			}
		}
		
		return AgentListUpdatedMsg{
			Agents: resp.Agents,
		}
	}
}

// GetAgentStatus retrieves status for a specific agent
func (ar *AgentRouter) GetAgentStatus(agentID string) tea.Cmd {
	return func() tea.Msg {
		req := &pb.GetAgentStatusRequest{
			AgentId: agentID,
		}
		
		resp, err := ar.guildClient.GetAgentStatus(ar.ctx, req)
		if err != nil {
			return AgentErrorMsg{
				AgentID: agentID,
				Error:   gerror.Wrap(err, gerror.ErrCodeConnection, "failed to get agent status").
					WithComponent("chat.v2.agents").
					WithOperation("GetAgentStatus").
					WithDetails("agent_id", agentID),
			}
		}
		
		// Update local cache
		if info, exists := ar.agents[agentID]; exists {
			info.Status = resp.State
			info.LastActivity = time.Unix(resp.LastActivity, 0)
		}
		
		return AgentStatusMsg{
			AgentID: agentID,
			Status:  resp,
		}
	}
}

// AgentExists checks if an agent is available
func (ar *AgentRouter) AgentExists(agentID string) bool {
	_, exists := ar.agents[agentID]
	return exists
}

// GetAvailableAgents returns a list of all available agent IDs
func (ar *AgentRouter) GetAvailableAgents() []string {
	agents := make([]string, 0, len(ar.agents))
	for id := range ar.agents {
		agents = append(agents, id)
	}
	return agents
}

// GetAgentInfo returns detailed information about an agent
func (ar *AgentRouter) GetAgentInfo(agentID string) *AgentInfo {
	return ar.agents[agentID]
}

// FormatAgentMention creates a formatted agent mention for display
func (ar *AgentRouter) FormatAgentMention(agentID string) string {
	if info, exists := ar.agents[agentID]; exists {
		status := getStatusIcon(info.Status)
		return fmt.Sprintf("%s @%s", status, agentID)
	}
	return fmt.Sprintf("@%s", agentID)
}

// GetAgentCompletions returns agent IDs that match a partial input (for auto-completion)
func (ar *AgentRouter) GetAgentCompletions(partial string) []string {
	var matches []string
	
	// Add special broadcast option
	if strings.HasPrefix("all", partial) {
		matches = append(matches, "all")
	}
	
	// Add matching agent IDs
	for id := range ar.agents {
		if strings.HasPrefix(id, partial) {
			matches = append(matches, id)
		}
	}
	
	return matches
}

// StartAgentStream initiates a streaming conversation with an agent
func (ar *AgentRouter) StartAgentStream(agentID string) tea.Cmd {
	return func() tea.Msg {
		// This would implement bidirectional streaming in a full implementation
		// For now, return a placeholder that indicates streaming started
		return AgentStreamStartedMsg{
			AgentID: agentID,
		}
	}
}

// Helper functions

// generateMessageID creates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// getStatusIcon returns an emoji icon for agent status (copied from V1)
func getStatusIcon(status pb.AgentStatus_State) string {
	switch status {
	case pb.AgentStatus_IDLE:
		return "🟢"
	case pb.AgentStatus_THINKING:
		return "🤔"
	case pb.AgentStatus_WORKING:
		return "⚙️"
	case pb.AgentStatus_WAITING:
		return "⏳"
	case pb.AgentStatus_ERROR:
		return "🔴"
	case pb.AgentStatus_OFFLINE:
		return "⚫"
	default:
		return "⚪"
	}
}

// Message types for agent communication

// AgentResponseMsg represents a response from a specific agent
type AgentResponseMsg struct {
	AgentID   string
	Content   string
	MessageID string
	Timestamp time.Time
}

// BroadcastResponseMsg represents responses from multiple agents
type BroadcastResponseMsg struct {
	Responses []*pb.AgentMessageResponse
	MessageID string
	Timestamp time.Time
}

// AgentErrorMsg represents an error in agent communication
type AgentErrorMsg struct {
	AgentID string
	Error   error
}

// AgentListUpdatedMsg indicates the agent list has been refreshed  
type AgentListUpdatedMsg struct {
	Agents []*pb.AgentInfo
}

// AgentStatusMsg represents an agent status update
type AgentStatusMsg struct {
	AgentID string
	Status  *pb.AgentStatus
}

// AgentStreamStartedMsg indicates a streaming session has started
type AgentStreamStartedMsg struct {
	AgentID string
}