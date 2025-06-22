// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
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
	
	// Suggestion integration
	suggestionService *SuggestionService
	suggestionMode    SuggestionMode
	enableSuggestions bool
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
		suggestionMode: SuggestionModeBoth,
		enableSuggestions: true,
	}, nil
}

// NewChatServiceWithSuggestions creates a chat service with integrated suggestions
func NewChatServiceWithSuggestions(
	ctx context.Context,
	client pb.GuildClient,
	registry registry.ComponentRegistry,
	enhancedAgent agent.EnhancedGuildArtisan,
) (*ChatService, error) {
	// Create base chat service
	chatService, err := NewChatService(ctx, client, registry)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chat service").
			WithComponent("services.chat").
			WithOperation("NewChatServiceWithSuggestions")
	}
	
	// Create and attach suggestion service if agent provided
	if enhancedAgent != nil {
		handler := agent.NewChatSuggestionHandler(enhancedAgent)
		suggestionService, err := NewSuggestionService(ctx, handler)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create suggestion service").
				WithComponent("services.chat").
				WithOperation("NewChatServiceWithSuggestions")
		}
		
		chatService.suggestionService = suggestionService
		chatService.enableSuggestions = true
		chatService.suggestionMode = SuggestionModeBoth
	}
	
	return chatService, nil
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
		
		// Optimize message if suggestions are enabled
		optimizedMessage := message
		if cs.enableSuggestions && cs.suggestionService != nil {
			optimizedMessage = cs.suggestionService.OptimizeContext(message)
			// Message optimized for better efficiency
		}
		
		if agentID == "all" {
			return cs.broadcastMessage(ctx, optimizedMessage)
		}
		
		return cs.sendToAgent(ctx, agentID, optimizedMessage)
	}
}

// SendMessageWithSuggestions sends a message with pre/post suggestions
func (cs *ChatService) SendMessageWithSuggestions(agentID, message, conversationID string) tea.Cmd {
	cmds := []tea.Cmd{}
	
	// Get pre-execution suggestions if enabled
	if cs.enableSuggestions && (cs.suggestionMode == SuggestionModePre || cs.suggestionMode == SuggestionModeBoth) {
		if preCmd := cs.GetPreExecutionSuggestions(message, conversationID); preCmd != nil {
			cmds = append(cmds, preCmd)
		}
	}
	
	// Send the message
	cmds = append(cmds, cs.SendMessage(agentID, message))
	
	// Return batch command
	if len(cmds) > 1 {
		return tea.Batch(cmds...)
	}
	return cmds[0]
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
	logger := observability.GetLogger(ctx).
		WithComponent("services.chat").
		WithOperation("sendToAgent")
	
	// TODO: Implement actual message sending when gRPC interface is available
	response := fmt.Sprintf("Agent %s received: %s", agentID, message)
	
	// Log token usage for analytics
	if cs.enableSuggestions {
		logger.Debug("Message sent with token optimization",
			"agent_id", agentID,
			"original_length", len(message),
			"message_sent", true)
	}
	
	return AgentResponseMsg{
		AgentID: agentID,
		Content: response,
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

// SetSuggestionService sets the suggestion service for integration
func (cs *ChatService) SetSuggestionService(service *SuggestionService) error {
	if service == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "suggestion service cannot be nil", nil).
			WithComponent("services.chat").
			WithOperation("SetSuggestionService")
	}
	
	cs.suggestionService = service
	return nil
}

// SetSuggestionMode sets when suggestions are generated
func (cs *ChatService) SetSuggestionMode(mode SuggestionMode) {
	cs.suggestionMode = mode
	cs.enableSuggestions = mode != SuggestionModeNone
}

// GetPreExecutionSuggestions retrieves suggestions before sending a message
func (cs *ChatService) GetPreExecutionSuggestions(message string, conversationID string) tea.Cmd {
	if !cs.enableSuggestions || cs.suggestionService == nil {
		return nil
	}
	
	if cs.suggestionMode != SuggestionModePre && cs.suggestionMode != SuggestionModeBoth {
		return nil
	}
	
	return func() tea.Msg {
		context := &SuggestionContext{
			ConversationID: conversationID,
		}
		
		cmd := cs.suggestionService.GetSuggestions(message, context)
		return cmd()
	}
}

// GetPostExecutionSuggestions retrieves follow-up suggestions after response
func (cs *ChatService) GetPostExecutionSuggestions(originalMessage, response string) tea.Cmd {
	if !cs.enableSuggestions || cs.suggestionService == nil {
		return nil
	}
	
	if cs.suggestionMode != SuggestionModePost && cs.suggestionMode != SuggestionModeBoth {
		return nil
	}
	
	return cs.suggestionService.GetFollowUpSuggestions(originalMessage, response)
}

// ProcessAgentResponse processes an agent response and generates suggestions
func (cs *ChatService) ProcessAgentResponse(response AgentResponseMsg, originalMessage string) tea.Cmd {
	if !cs.enableSuggestions || cs.suggestionService == nil {
		return nil
	}
	
	return func() tea.Msg {
		// Get post-execution suggestions if enabled
		if cs.suggestionMode == SuggestionModePost || cs.suggestionMode == SuggestionModeBoth {
			cmd := cs.GetPostExecutionSuggestions(originalMessage, response.Content)
			if cmd != nil {
				msg := cmd()
				if sugMsg, ok := msg.(SuggestionsReceivedMsg); ok {
					return AgentResponseWithSuggestionsMsg{
						AgentID:     response.AgentID,
						Content:     response.Content,
						Done:        response.Done,
						Suggestions: sugMsg.Suggestions,
						TokensUsed:  0,
					}
				}
			}
		}
		
		// Return response without suggestions
		return response
	}
}

// ConfigureSuggestions enables or disables the suggestion system
func (cs *ChatService) ConfigureSuggestions(enabled bool) {
	cs.enableSuggestions = enabled
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
	
	// Suggestion integration stats
	if cs.enableSuggestions {
		stats["suggestions_enabled"] = true
		stats["suggestion_mode"] = string(cs.suggestionMode)
		
		// Include suggestion service stats if available
		if cs.suggestionService != nil {
			suggestionStats := cs.suggestionService.GetStats()
			for k, v := range suggestionStats {
				stats["suggestion_"+k] = v
			}
		}
	} else {
		stats["suggestions_enabled"] = false
	}
	
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

// SuggestionMode defines when suggestions are generated
type SuggestionMode string

const (
	SuggestionModeNone SuggestionMode = "none"
	SuggestionModePre  SuggestionMode = "pre"
	SuggestionModePost SuggestionMode = "post"
	SuggestionModeBoth SuggestionMode = "both"
)

// ChatMessageWithSuggestionsMsg represents a message with suggestions
type ChatMessageWithSuggestionsMsg struct {
	AgentID        string
	Message        string
	ConversationID string
	Suggestions    []suggestions.Suggestion
}

// AgentResponseWithSuggestionsMsg represents agent response with follow-up suggestions
type AgentResponseWithSuggestionsMsg struct {
	AgentID     string
	Content     string
	Done        bool
	Suggestions []suggestions.Suggestion
	TokensUsed  int
}