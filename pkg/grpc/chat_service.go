// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/registry"
)

// ChatService implements real-time bidirectional communication with Guild agents
// Following the registry pattern for proper dependency management
type ChatService struct {
	pb.UnimplementedChatServiceServer

	// Registry pattern - use main component registry
	registry   registry.ComponentRegistry
	logger     *slog.Logger
	sessions   map[string]*ChatSession
	sessionsMu sync.RWMutex

	// Event broadcasting
	eventBus    EventBus
	subscribers map[string][]chan *pb.ChatResponse
	subsMu      sync.RWMutex
}

// ChatSession represents an active chat session
type ChatSession struct {
	ID           string
	Name         string
	AgentIDs     []string
	CampaignID   string
	Status       pb.ChatSession_SessionStatus
	CreatedAt    time.Time
	LastActivity time.Time
	Context      *pb.SessionContext
	Metadata     map[string]string

	// Active agents in this session
	agents   map[string]core.Agent
	agentsMu sync.RWMutex

	// Message history
	messages   []*pb.ChatMessage
	messagesMu sync.RWMutex

	// Tool execution tracking
	toolExecutions map[string]*pb.ToolExecution
	toolsMu        sync.RWMutex

	// Stream management
	streams   map[string]pb.ChatService_ChatServer
	streamsMu sync.RWMutex
}

// NewChatService creates a new chat service following the registry pattern
func NewChatService(registry registry.ComponentRegistry, eventBus EventBus) *ChatService {
	logger := slog.With("component", "chat_service")

	service := &ChatService{
		registry:    registry,
		logger:      logger,
		sessions:    make(map[string]*ChatSession),
		eventBus:    eventBus,
		subscribers: make(map[string][]chan *pb.ChatResponse),
	}

	logger.Info("chat service initialized", "service_type", "bidirectional_streaming")
	return service
}

// Chat implements bidirectional streaming chat with proper error handling and logging
func (s *ChatService) Chat(stream pb.ChatService_ChatServer) error {
	ctx := stream.Context()
	sessionID := "unknown" // Will be populated from first message

	// Initialize comprehensive observability for gRPC chat stream
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("Chat")

	streamStart := time.Now()
	var messageCount int64
	var requestCount int64
	var responseCount int64
	var errorCount int64

	// Enhanced stream initialization observability
	remoteAddr := getRemoteAddr(ctx)
	userAgent := getUserAgent(ctx)
	logger.Info("New gRPC chat stream started",
		"remote_addr", remoteAddr,
		"user_agent", userAgent,
		"stream_type", "bidirectional",
	)

	// Add context values for distributed tracing
	ctx = context.WithValue(ctx, "stream_id", fmt.Sprintf("chat_%d", time.Now().UnixNano()))
	ctx = context.WithValue(ctx, "remote_addr", remoteAddr)

	defer func() {
		streamDuration := time.Since(streamStart)

		// Comprehensive stream completion observability
		logger.Info("gRPC chat stream ended",
			"session_id", sessionID,
			"stream_duration_ms", streamDuration.Milliseconds(),
			"total_messages_processed", messageCount,
			"requests_received", requestCount,
			"responses_sent", responseCount,
			"errors_encountered", errorCount,
			"remote_addr", remoteAddr,
		)

		// Log performance metrics for monitoring
		logger.Duration("grpc.chat_stream", streamDuration,
			"session_id", sessionID,
			"message_count", messageCount,
			"request_count", requestCount,
			"response_count", responseCount,
			"error_count", errorCount,
			"success", errorCount == 0,
		)
	}()

	for {
		iterationStart := time.Now()

		select {
		case <-ctx.Done():
			contextDuration := time.Since(iterationStart)

			// Enhanced context cancellation observability
			logger.Warn("gRPC chat stream context cancelled",
				"session_id", sessionID,
				"cancellation_reason", ctx.Err().Error(),
				"stream_duration_ms", time.Since(streamStart).Milliseconds(),
				"messages_processed", messageCount,
				"context_check_duration_ms", contextDuration.Milliseconds(),
			)
			return ctx.Err()
		default:
		}

		// Enhanced request reception with comprehensive error observability
		recvStart := time.Now()
		req, err := stream.Recv()
		recvDuration := time.Since(recvStart)
		requestCount++

		if err == io.EOF {
			// Enhanced EOF observability
			logger.Info("gRPC chat stream closed by client",
				"session_id", sessionID,
				"stream_duration_ms", time.Since(streamStart).Milliseconds(),
				"total_messages_processed", messageCount,
				"final_recv_duration_ms", recvDuration.Milliseconds(),
			)
			return nil
		}
		if err != nil {
			errorCount++

			// Enhanced stream receive error observability
			logger.WithError(err).Error("gRPC stream receive error",
				"session_id", sessionID,
				"request_count", requestCount,
				"stream_duration_ms", time.Since(streamStart).Milliseconds(),
				"recv_duration_ms", recvDuration.Milliseconds(),
				"remote_addr", remoteAddr,
			)

			// Log error metrics
			logger.Duration("grpc.stream_recv", recvDuration,
				"success", false,
				"session_id", sessionID,
				"error_type", fmt.Sprintf("%T", err),
			)

			return status.Errorf(codes.Internal, "stream receive error: %v", err)
		}

		// Log successful request reception
		logger.Debug("gRPC request received successfully",
			"session_id", sessionID,
			"recv_duration_ms", recvDuration.Milliseconds(),
			"request_type", fmt.Sprintf("%T", req.Request),
		)

		// Extract session ID for logging with enhanced observability
		if sessionID == "unknown" {
			extractStart := time.Now()
			sessionID = s.extractSessionID(req)
			extractDuration := time.Since(extractStart)

			logger.Debug("Session ID extracted from request",
				"session_id", sessionID,
				"extraction_duration_ms", extractDuration.Milliseconds(),
			)
		}

		// Enhanced request handling with comprehensive error observability
		handleStart := time.Now()
		if err := s.handleChatRequest(ctx, req, stream); err != nil {
			errorCount++
			handleDuration := time.Since(handleStart)

			// Enhanced request handling error observability
			logger.WithError(err).Error("Failed to handle gRPC chat request",
				"session_id", sessionID,
				"request_type", fmt.Sprintf("%T", req.Request),
				"handle_duration_ms", handleDuration.Milliseconds(),
				"stream_duration_ms", time.Since(streamStart).Milliseconds(),
				"total_errors", errorCount,
				"remote_addr", remoteAddr,
			)

			// Send error response but continue streaming with enhanced observability
			errorRespStart := time.Now()
			errorResp := &pb.ChatResponse{
				Response: &pb.ChatResponse_Error{
					Error: &pb.ChatError{
						Code:      pb.ChatError_UNKNOWN,
						Message:   err.Error(),
						Timestamp: time.Now().Unix(),
					},
				},
			}

			if sendErr := stream.Send(errorResp); sendErr != nil {
				errorRespDuration := time.Since(errorRespStart)

				// Enhanced error response send failure observability
				logger.WithError(sendErr).Error("Failed to send error response over gRPC stream",
					"session_id", sessionID,
					"original_error", err.Error(),
					"send_error", sendErr.Error(),
					"error_resp_duration_ms", errorRespDuration.Milliseconds(),
					"handle_duration_ms", handleDuration.Milliseconds(),
					"total_errors", errorCount,
				)

				return status.Errorf(codes.Internal, "failed to send error response: %v", sendErr)
			}

			errorRespDuration := time.Since(errorRespStart)
			logger.Debug("Error response sent successfully",
				"session_id", sessionID,
				"error_resp_duration_ms", errorRespDuration.Milliseconds(),
			)
			responseCount++

		} else {
			handleDuration := time.Since(handleStart)
			messageCount++
			responseCount++

			// Log successful request handling
			logger.Debug("gRPC chat request handled successfully",
				"session_id", sessionID,
				"request_type", fmt.Sprintf("%T", req.Request),
				"handle_duration_ms", handleDuration.Milliseconds(),
				"total_messages_processed", messageCount,
			)

			// Log performance metrics for successful request
			logger.Duration("grpc.request_handling", handleDuration,
				"success", true,
				"session_id", sessionID,
				"request_type", fmt.Sprintf("%T", req.Request),
			)
		}
	}
}

// handleChatRequest processes different types of chat requests
func (s *ChatService) handleChatRequest(ctx context.Context, req *pb.ChatRequest, stream pb.ChatService_ChatServer) error {
	switch r := req.Request.(type) {
	case *pb.ChatRequest_Message:
		return s.handleChatMessage(ctx, r.Message, stream)
	case *pb.ChatRequest_Control:
		return s.handleChatControl(ctx, r.Control, stream)
	case *pb.ChatRequest_ToolApproval:
		return s.handleToolApproval(ctx, r.ToolApproval, stream)
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "unknown request type", nil).
			WithComponent("grpc").
			WithOperation("handleChatRequest").
			FromContext(ctx)
	}
}

// handleChatMessage processes user messages and routes them to agents
func (s *ChatService) handleChatMessage(ctx context.Context, msg *pb.ChatMessage, stream pb.ChatService_ChatServer) error {
	session, err := s.getSession(msg.SessionId)
	if err != nil {
		return err
	}

	// Add message to session history
	session.addMessage(msg)

	// Send thinking indicator
	for _, agentID := range session.AgentIDs {
		thinkingResp := &pb.ChatResponse{
			Response: &pb.ChatResponse_Thinking{
				Thinking: &pb.AgentThinking{
					AgentId:     agentID,
					AgentName:   s.getAgentName(agentID),
					SessionId:   msg.SessionId,
					State:       pb.AgentThinking_ANALYZING,
					Description: "Analyzing your message...",
					Timestamp:   time.Now().Unix(),
				},
			},
		}
		if err := stream.Send(thinkingResp); err != nil {
			return err
		}
	}

	// Process message with agents - ensure we always send a response
	if len(session.AgentIDs) == 0 {
		// No agents configured, send a default response
		defaultResp := &pb.ChatResponse{
			Response: &pb.ChatResponse_Message{
				Message: &pb.ChatMessage{
					SessionId:  msg.SessionId,
					SenderId:   "system",
					SenderName: "Guild System",
					Content:    "No agents are configured for this session. Please check your guild configuration.",
					Type:       pb.ChatMessage_SYSTEM_MESSAGE,
					Timestamp:  time.Now().Unix(),
				},
			},
		}
		session.addMessage(defaultResp.GetMessage())
		return stream.Send(defaultResp)
	}

	// Process message with agents
	go s.processWithAgents(ctx, session, msg, stream)

	return nil
}

// processWithAgents handles message processing by agents asynchronously
func (s *ChatService) processWithAgents(ctx context.Context, session *ChatSession, msg *pb.ChatMessage, stream pb.ChatService_ChatServer) {
	session.agentsMu.RLock()
	agents := make([]core.Agent, 0, len(session.agents))
	for _, ag := range session.agents {
		agents = append(agents, ag)
	}
	session.agentsMu.RUnlock()

	// If no agents are loaded, send an error response
	if len(agents) == 0 {
		errorResp := &pb.ChatResponse{
			Response: &pb.ChatResponse_Error{
				Error: &pb.ChatError{
					Code:      pb.ChatError_AGENT_UNAVAILABLE,
					Message:   "No agents are currently available for this session",
					Timestamp: time.Now().Unix(),
				},
			},
		}
		stream.Send(errorResp)
		return
	}

	for _, ag := range agents {
		go s.processWithSingleAgent(ctx, session, ag, msg, stream)
	}
}

// processWithSingleAgent processes a message with a specific agent
func (s *ChatService) processWithSingleAgent(ctx context.Context, session *ChatSession, ag core.Agent, msg *pb.ChatMessage, stream pb.ChatService_ChatServer) {
	agentID := ag.GetID()

	// Send planning state
	planningResp := &pb.ChatResponse{
		Response: &pb.ChatResponse_Thinking{
			Thinking: &pb.AgentThinking{
				AgentId:     agentID,
				AgentName:   ag.GetName(),
				SessionId:   msg.SessionId,
				State:       pb.AgentThinking_PLANNING,
				Description: "Planning response...",
				Timestamp:   time.Now().Unix(),
			},
		},
	}
	stream.Send(planningResp)

	// Execute agent processing (this would integrate with the actual agent execution)
	response, err := s.executeAgentResponse(ctx, ag, msg)
	if err != nil {
		errorResp := &pb.ChatResponse{
			Response: &pb.ChatResponse_Error{
				Error: &pb.ChatError{
					Code:      pb.ChatError_AGENT_UNAVAILABLE,
					Message:   fmt.Sprintf("Agent %s error: %v", agentID, err),
					Timestamp: time.Now().Unix(),
				},
			},
		}
		stream.Send(errorResp)
		return
	}

	// Send agent response
	agentResp := &pb.ChatResponse{
		Response: &pb.ChatResponse_Message{
			Message: &pb.ChatMessage{
				SessionId:  msg.SessionId,
				SenderId:   agentID,
				SenderName: ag.GetName(),
				Content:    response,
				Type:       pb.ChatMessage_AGENT_RESPONSE,
				Timestamp:  time.Now().Unix(),
			},
		},
	}

	session.addMessage(agentResp.GetMessage())
	stream.Send(agentResp)
}

// executeAgentResponse executes the agent with real task processing
func (s *ChatService) executeAgentResponse(ctx context.Context, ag core.Agent, msg *pb.ChatMessage) (string, error) {
	// Execute the agent with the message content
	response, err := ag.Execute(ctx, msg.Content)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "agent execution failed").
			WithComponent("grpc").
			WithOperation("executeAgentResponse").
			WithDetails("agent_id", ag.GetID()).
			WithDetails("message_content", msg.Content).
			FromContext(ctx)
	}

	return response, nil
}

// handleChatControl processes session control commands
func (s *ChatService) handleChatControl(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	switch control.Action {
	case pb.ChatControl_START_SESSION:
		return s.startSession(ctx, control, stream)
	case pb.ChatControl_END_SESSION:
		return s.endSession(ctx, control, stream)
	case pb.ChatControl_PAUSE_SESSION:
		return s.pauseSession(ctx, control, stream)
	case pb.ChatControl_RESUME_SESSION:
		return s.resumeSession(ctx, control, stream)
	case pb.ChatControl_INTERRUPT_AGENT:
		return s.interruptAgent(ctx, control, stream)
	case pb.ChatControl_REQUEST_STATUS:
		return s.requestStatus(ctx, control, stream)
	default:
		return gerror.Newf(gerror.ErrCodeInvalidInput, "unknown control action: %v", control.Action).
			WithComponent("grpc").
			WithOperation("handleChatControl").
			WithDetails("action", control.Action).
			FromContext(ctx)
	}
}

// handleToolApproval processes user approval/rejection of tool executions
func (s *ChatService) handleToolApproval(ctx context.Context, approval *pb.ToolApproval, stream pb.ChatService_ChatServer) error {
	session, err := s.getSession(approval.SessionId)
	if err != nil {
		return err
	}

	session.toolsMu.Lock()
	toolExec, exists := session.toolExecutions[approval.ToolExecutionId]
	if !exists {
		session.toolsMu.Unlock()
		return gerror.Newf(gerror.ErrCodeNotFound, "tool execution not found: %s", approval.ToolExecutionId).
			WithComponent("grpc").
			WithOperation("handleToolApproval").
			WithDetails("tool_execution_id", approval.ToolExecutionId).
			FromContext(ctx)
	}

	if approval.Approved {
		toolExec.Status = pb.ToolExecution_EXECUTING
		// Execute the tool with approved parameters
		go s.executeApprovedTool(ctx, session, toolExec, approval, stream)
	} else {
		toolExec.Status = pb.ToolExecution_CANCELLED
	}
	session.toolsMu.Unlock()

	// Send updated tool execution status
	toolResp := &pb.ChatResponse{
		Response: &pb.ChatResponse_ToolExecution{
			ToolExecution: toolExec,
		},
	}
	return stream.Send(toolResp)
}

// executeApprovedTool executes a tool that has been approved by the user
func (s *ChatService) executeApprovedTool(ctx context.Context, session *ChatSession, toolExec *pb.ToolExecution, approval *pb.ToolApproval, stream pb.ChatService_ChatServer) {
	// Update tool execution to show progress
	toolExec.Progress = 0.1
	toolExec.UpdatedAt = time.Now().Unix()

	progressResp := &pb.ChatResponse{
		Response: &pb.ChatResponse_ToolExecution{
			ToolExecution: toolExec,
		},
	}
	stream.Send(progressResp)

	// Get tool registry
	toolRegistry := s.registry.Tools()
	if toolRegistry == nil {
		toolExec.Status = pb.ToolExecution_FAILED
		toolExec.Error = "Tool registry not available"
		toolExec.UpdatedAt = time.Now().Unix()
		return
	}

	// Get the tool
	tool, err := toolRegistry.GetTool(toolExec.ToolName)
	if err != nil {
		toolExec.Status = pb.ToolExecution_FAILED
		toolExec.Error = fmt.Sprintf("Tool not found: %v", err)
		toolExec.UpdatedAt = time.Now().Unix()
		return
	}

	// Convert parameters to JSON string for tool execution
	paramsJSON, err := json.Marshal(toolExec.Parameters)
	if err != nil {
		toolExec.Status = pb.ToolExecution_FAILED
		toolExec.Error = fmt.Sprintf("Failed to marshal parameters: %v", err)
		toolExec.UpdatedAt = time.Now().Unix()
		return
	}

	// Execute the tool
	result, err := tool.Execute(ctx, string(paramsJSON))
	if err != nil {
		toolExec.Status = pb.ToolExecution_FAILED
		toolExec.Error = fmt.Sprintf("Tool execution failed: %v", err)
		toolExec.UpdatedAt = time.Now().Unix()
		return
	}

	toolExec.Status = pb.ToolExecution_COMPLETED
	toolExec.Progress = 1.0
	toolExec.Result = result.Output
	if result.Error != "" {
		toolExec.Error = result.Error
	}
	toolExec.UpdatedAt = time.Now().Unix()

	completionResp := &pb.ChatResponse{
		Response: &pb.ChatResponse_ToolExecution{
			ToolExecution: toolExec,
		},
	}
	stream.Send(completionResp)
}

// Session management methods

func (s *ChatService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.ChatSession, error) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	sessionID := generateSessionID()
	session := &ChatSession{
		ID:             sessionID,
		Name:           req.Name,
		AgentIDs:       req.AgentIds,
		CampaignID:     req.CampaignId,
		Status:         pb.ChatSession_ACTIVE,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		Context:        req.Context,
		Metadata:       req.Metadata,
		agents:         make(map[string]core.Agent),
		messages:       make([]*pb.ChatMessage, 0),
		toolExecutions: make(map[string]*pb.ToolExecution),
		streams:        make(map[string]pb.ChatService_ChatServer),
	}

	// Load agents from registry
	agentRegistry := s.registry.Agents()
	for _, agentID := range req.AgentIds {
		if agentRegistry != nil {
			if ag, err := agentRegistry.GetAgent(agentID); err == nil {
				session.agents[agentID] = ag
			} else {
				s.logger.Warn("failed to load agent for session",
					"agent_id", agentID,
					"session_id", sessionID,
					"error", err)
			}
		} else {
			s.logger.Warn("agent registry not available", "session_id", sessionID)
		}
	}

	s.sessions[sessionID] = session

	return &pb.ChatSession{
		Id:           sessionID,
		Name:         session.Name,
		AgentIds:     session.AgentIDs,
		CampaignId:   session.CampaignID,
		Status:       session.Status,
		CreatedAt:    session.CreatedAt.Unix(),
		LastActivity: session.LastActivity.Unix(),
		Metadata:     session.Metadata,
		Context:      session.Context,
	}, nil
}

func (s *ChatService) EndChatSession(ctx context.Context, req *pb.EndChatSessionRequest) (*pb.EndChatSessionResponse, error) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	session, exists := s.sessions[req.SessionId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "session not found: %s", req.SessionId)
	}

	session.Status = pb.ChatSession_ENDED

	// Close all streams for this session
	session.streamsMu.Lock()
	for _, stream := range session.streams {
		// Send session end event
		endEvent := &pb.ChatResponse{
			Response: &pb.ChatResponse_Event{
				Event: &pb.ChatEvent{
					SessionId:   req.SessionId,
					Type:        pb.ChatEvent_SESSION_ENDED,
					Description: req.Reason,
					Timestamp:   time.Now().Unix(),
				},
			},
		}
		stream.Send(endEvent)
	}
	session.streamsMu.Unlock()

	summary := &pb.ChatSessionSummary{
		TotalMessages:  int32(len(session.messages)),
		ToolsExecuted:  int32(len(session.toolExecutions)),
		AgentsInvolved: session.AgentIDs,
		Outcome:        req.Reason,
	}

	delete(s.sessions, req.SessionId)

	return &pb.EndChatSessionResponse{
		Success: true,
		Message: "Session ended successfully",
		Summary: summary,
	}, nil
}

func (s *ChatService) ListChatSessions(ctx context.Context, req *pb.ListChatSessionsRequest) (*pb.ListChatSessionsResponse, error) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*pb.ChatSession, 0)
	for _, session := range s.sessions {
		if !req.IncludeEnded && session.Status == pb.ChatSession_ENDED {
			continue
		}
		if req.CampaignId != "" && session.CampaignID != req.CampaignId {
			continue
		}

		sessions = append(sessions, &pb.ChatSession{
			Id:           session.ID,
			Name:         session.Name,
			AgentIds:     session.AgentIDs,
			CampaignId:   session.CampaignID,
			Status:       session.Status,
			CreatedAt:    session.CreatedAt.Unix(),
			LastActivity: session.LastActivity.Unix(),
			Metadata:     session.Metadata,
			Context:      session.Context,
		})

		if req.Limit > 0 && len(sessions) >= int(req.Limit) {
			break
		}
	}

	return &pb.ListChatSessionsResponse{
		Sessions:   sessions,
		TotalCount: int32(len(sessions)),
	}, nil
}

func (s *ChatService) GetChatHistory(ctx context.Context, req *pb.GetChatHistoryRequest) (*pb.GetChatHistoryResponse, error) {
	session, err := s.getSession(req.SessionId)
	if err != nil {
		return nil, err
	}

	session.messagesMu.RLock()
	messages := make([]*pb.ChatMessage, 0)
	for _, msg := range session.messages {
		if req.SinceTimestamp > 0 && msg.Timestamp < req.SinceTimestamp {
			continue
		}
		if !req.IncludeSystemMessages && msg.Type == pb.ChatMessage_SYSTEM_MESSAGE {
			continue
		}
		messages = append(messages, msg)

		if req.Limit > 0 && len(messages) >= int(req.Limit) {
			break
		}
	}
	session.messagesMu.RUnlock()

	return &pb.GetChatHistoryResponse{
		Messages:   messages,
		TotalCount: int32(len(messages)),
		HasMore:    req.Limit > 0 && len(session.messages) > int(req.Limit),
	}, nil
}

// Helper methods

func (s *ChatService) getSession(sessionID string) (*ChatSession, error) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "session not found: %s", sessionID)
	}
	return session, nil
}

func (s *ChatService) getAgentName(agentID string) string {
	agentRegistry := s.registry.Agents()
	if agentRegistry != nil {
		if ag, err := agentRegistry.GetAgent(agentID); err == nil {
			return ag.GetName()
		}
	}
	return agentID
}

func (session *ChatSession) addMessage(msg *pb.ChatMessage) {
	session.messagesMu.Lock()
	defer session.messagesMu.Unlock()
	session.messages = append(session.messages, msg)
	session.LastActivity = time.Now()
}

// Session control implementations (stubs for now)
func (s *ChatService) startSession(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for session start
	return nil
}

func (s *ChatService) endSession(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for session end
	return nil
}

func (s *ChatService) pauseSession(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for session pause
	return nil
}

func (s *ChatService) resumeSession(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for session resume
	return nil
}

func (s *ChatService) interruptAgent(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for agent interruption
	return nil
}

func (s *ChatService) requestStatus(ctx context.Context, control *pb.ChatControl, stream pb.ChatService_ChatServer) error {
	// Implementation for status request
	return nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("chat_%d", time.Now().UnixNano())
}

// Helper functions for context extraction and debugging

func getRemoteAddr(ctx context.Context) string {
	// Extract remote address from gRPC context for debugging
	// This would use gRPC metadata to get peer information
	return "unknown" // Placeholder
}

func getUserAgent(ctx context.Context) string {
	// Extract user agent from gRPC context for debugging
	return "unknown" // Placeholder
}

func (s *ChatService) extractSessionID(req *pb.ChatRequest) string {
	switch r := req.Request.(type) {
	case *pb.ChatRequest_Message:
		if r.Message != nil {
			return r.Message.SessionId
		}
	case *pb.ChatRequest_Control:
		if r.Control != nil {
			return r.Control.SessionId
		}
	case *pb.ChatRequest_ToolApproval:
		if r.ToolApproval != nil {
			return r.ToolApproval.SessionId
		}
	}
	return "unknown"
}

// EventBus interface for broadcasting events
type EventBus interface {
	Publish(event interface{})
	Subscribe(eventType string, handler func(event interface{}))
}
