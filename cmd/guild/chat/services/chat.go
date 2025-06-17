// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// ChatService handles communication with Guild agents via gRPC
type ChatService struct {
	ctx    context.Context
	client pb.GuildClient
	registry registry.ComponentRegistry
	
	// State
	activeStreams map[string]interface{} // Simplified for now
	agents        []string
	
	// Configuration
	timeout time.Duration
}

// NewChatService creates a new chat service
func NewChatService(ctx context.Context, client pb.GuildClient, registry registry.ComponentRegistry) (*ChatService, error) {
	if client == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "gRPC client cannot be nil", nil).
			WithComponent("services.chat").
			WithOperation("NewChatService")
	}
	
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "registry cannot be nil", nil).
			WithComponent("services.chat").
			WithOperation("NewChatService")
	}
	
	return &ChatService{
		ctx:           ctx,
		client:        client,
		registry:      registry,
		activeStreams: make(map[string]interface{}),
		agents:        make([]string, 0),
		timeout:       30 * time.Second,
	}, nil
}

// Start initializes the chat service
func (cs *ChatService) Start() tea.Cmd {
	return func() tea.Msg {
		// Discover available agents
		agents, err := cs.discoverAgents()
		if err != nil {
			return ChatServiceErrorMsg{
				Operation: "discover_agents",
				Error:     err,
			}
		}
		
		cs.agents = agents
		
		return ChatServiceStartedMsg{
			Agents: agents,
		}
	}
}

// SendMessage sends a message to an agent or all agents
func (cs *ChatService) SendMessage(agentID, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(cs.ctx, cs.timeout)
		defer cancel()
		
		if agentID == "all" {
			return cs.broadcastMessage(ctx, message)
		}
		
		return cs.sendToAgent(ctx, agentID, message)
	}
}

// StreamChat establishes a streaming chat connection with an agent
func (cs *ChatService) StreamChat(agentID string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement streaming when gRPC interface is available
		return ChatStreamStartedMsg{
			AgentID: agentID,
		}
	}
}

// GetAgents returns the list of available agents
func (cs *ChatService) GetAgents() []string {
	return cs.agents
}

// GetAgentStatus gets the current status of an agent
func (cs *ChatService) GetAgentStatus(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(cs.ctx, 5*time.Second)
		defer cancel()
		
		resp, err := cs.client.GetAgentStatus(ctx, &pb.GetAgentStatusRequest{
			AgentId: agentID,
		})
		if err != nil {
			return ChatServiceErrorMsg{
				Operation: "get_agent_status",
				Error: gerror.Wrap(err, gerror.ErrCodeConnection, "failed to get agent status").
					WithComponent("services.chat").
					WithOperation("GetAgentStatus").
					WithDetails("agent_id", agentID),
			}
		}
		
		return AgentStatusUpdateMsg{
			AgentID: agentID,
			Status:  resp,
		}
	}
}

// ExecuteTool executes a tool via the chat service
func (cs *ChatService) ExecuteTool(toolName string, parameters map[string]string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement tool execution when gRPC interface is available
		return ToolExecutionCompleteMsg{
			ExecutionID: fmt.Sprintf("exec-%d", time.Now().UnixNano()),
			ToolName:    toolName,
			Result:      "Tool execution completed",
		}
	}
}

// StopStream stops a streaming chat connection
func (cs *ChatService) StopStream(agentID string) tea.Cmd {
	return func() tea.Msg {
		if _, exists := cs.activeStreams[agentID]; exists {
			delete(cs.activeStreams, agentID)
		}
		
		return ChatStreamStoppedMsg{
			AgentID: agentID,
		}
	}
}

// discoverAgents discovers available agents from the registry
func (cs *ChatService) discoverAgents() ([]string, error) {
	// TODO: Implement agent discovery when gRPC interface is available
	// For now, return mock agents
	return []string{"developer", "writer", "researcher", "tester"}, nil
}

// sendToAgent sends a message to a specific agent
func (cs *ChatService) sendToAgent(ctx context.Context, agentID, message string) tea.Msg {
	// TODO: Implement actual message sending when gRPC interface is available
	return AgentResponseMsg{
		AgentID: agentID,
		Content: fmt.Sprintf("Agent %s received: %s", agentID, message),
		Done:    true,
	}
}

// broadcastMessage sends a message to all available agents
func (cs *ChatService) broadcastMessage(ctx context.Context, message string) tea.Msg {
	responses := make([]AgentResponseMsg, 0)
	
	for _, agentID := range cs.agents {
		responses = append(responses, AgentResponseMsg{
			AgentID: agentID,
			Content: fmt.Sprintf("Agent %s received broadcast: %s", agentID, message),
			Done:    true,
		})
	}
	
	return BroadcastResponseMsg{
		Responses: responses,
		Errors:    []error{},
	}
}

// listenToStream listens for streaming responses from an agent
// TODO: Implement when streaming gRPC interface is available

// GetActiveStreams returns the number of active streaming connections
func (cs *ChatService) GetActiveStreams() int {
	return len(cs.activeStreams)
}

// IsStreamActive checks if a stream is active for an agent
func (cs *ChatService) IsStreamActive(agentID string) bool {
	_, exists := cs.activeStreams[agentID]
	return exists
}

// SetTimeout sets the timeout for chat operations
func (cs *ChatService) SetTimeout(timeout time.Duration) {
	cs.timeout = timeout
}

// GetStats returns statistics about the chat service
func (cs *ChatService) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["agent_count"] = len(cs.agents)
	stats["active_streams"] = len(cs.activeStreams)
	stats["timeout"] = cs.timeout.String()
	
	// Stream details
	streamInfo := make(map[string]interface{})
	for agentID := range cs.activeStreams {
		streamInfo[agentID] = "active"
	}
	stats["streams"] = streamInfo
	
	return stats
}

// Message types for chat service communication

// ChatServiceStartedMsg indicates the chat service has started
type ChatServiceStartedMsg struct {
	Agents []string
}

// ChatServiceErrorMsg represents a chat service error
type ChatServiceErrorMsg struct {
	Operation string
	Error     error
}

// AgentResponseMsg represents a response from an agent
type AgentResponseMsg struct {
	AgentID string
	Content string
	Done    bool
}

// BroadcastResponseMsg represents responses from a broadcast message
type BroadcastResponseMsg struct {
	Responses []AgentResponseMsg
	Errors    []error
}

// ChatStreamStartedMsg indicates a chat stream has started
type ChatStreamStartedMsg struct {
	AgentID string
}

// ChatStreamStoppedMsg indicates a chat stream has stopped
type ChatStreamStoppedMsg struct {
	AgentID string
}

// AgentStatusUpdateMsg represents an agent status update
type AgentStatusUpdateMsg struct {
	AgentID string
	Status  *pb.AgentStatus
}

// ToolExecutionCompleteMsg represents completed tool execution
type ToolExecutionCompleteMsg struct {
	ExecutionID string
	ToolName    string
	Result      string
}

// ToolExecutionErrorMsg represents a tool execution error
type ToolExecutionErrorMsg struct {
	ToolName string
	Error    error
}