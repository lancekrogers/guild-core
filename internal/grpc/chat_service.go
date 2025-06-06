package grpc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// ChatService implements real-time bidirectional communication with Guild agents
// Following the registry pattern for proper dependency management
type ChatService struct {
	pb.UnimplementedChatServiceServer
	
	// Registry pattern - use main component registry
	registry     registry.ComponentRegistry
	logger       *slog.Logger
	sessions     map[string]*ChatSession
	sessionsMu   sync.RWMutex
	
	// Event broadcasting
	eventBus    EventBus
	subscribers map[string][]chan *pb.ChatResponse
	subsMu      sync.RWMutex
}

// ChatSession represents an active chat session
type ChatSession struct {
	ID          string
	Name        string
	AgentIDs    []string
	CampaignID  string
	Status      pb.ChatSession_SessionStatus
	CreatedAt   time.Time
	LastActivity time.Time
	Context     *pb.SessionContext
	Metadata    map[string]string
	
	// Active agents in this session
	agents    map[string]agent.Agent
	agentsMu  sync.RWMutex
	
	// Message history
	messages    []*pb.ChatMessage
	messagesMu  sync.RWMutex
	
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
	
	s.logger.Info("new chat stream started", 
		"remote_addr", getRemoteAddr(ctx),
		"user_agent", getUserAgent(ctx))
	
	defer func() {
		s.logger.Info("chat stream ended", "session_id", sessionID)
	}()
	
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("chat stream context cancelled", 
				"session_id", sessionID, 
				"reason", ctx.Err())
			return ctx.Err()
		default:
		}
		
		req, err := stream.Recv()
		if err == io.EOF {
			s.logger.Debug("chat stream closed by client", "session_id", sessionID)
			return nil
		}
		if err != nil {
			s.logger.Error("stream receive error", 
				"error", err,
				"session_id", sessionID)
			return status.Errorf(codes.Internal, "stream receive error: %v", err)
		}
		
		// Extract session ID for logging
		if sessionID == "unknown" {
			sessionID = s.extractSessionID(req)
		}
		
		if err := s.handleChatRequest(ctx, req, stream); err != nil {
			s.logger.Error("failed to handle chat request", 
				"error", err,
				"session_id", sessionID,
				"request_type", fmt.Sprintf("%T", req.Request))
			
			// Send error response but continue streaming
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
				s.logger.Error("failed to send error response", 
					"send_error", sendErr,
					"original_error", err,
					"session_id", sessionID)
				return status.Errorf(codes.Internal, "failed to send error response: %v", sendErr)
			}
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
	
	// Process message with agents
	go s.processWithAgents(ctx, session, msg, stream)
	
	return nil
}

// processWithAgents handles message processing by agents asynchronously
func (s *ChatService) processWithAgents(ctx context.Context, session *ChatSession, msg *pb.ChatMessage, stream pb.ChatService_ChatServer) {
	session.agentsMu.RLock()
	agents := make([]agent.Agent, 0, len(session.agents))
	for _, ag := range session.agents {
		agents = append(agents, ag)
	}
	session.agentsMu.RUnlock()
	
	for _, ag := range agents {
		go s.processWithSingleAgent(ctx, session, ag, msg, stream)
	}
}

// processWithSingleAgent processes a message with a specific agent
func (s *ChatService) processWithSingleAgent(ctx context.Context, session *ChatSession, ag agent.Agent, msg *pb.ChatMessage, stream pb.ChatService_ChatServer) {
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

// executeAgentResponse simulates agent response execution
func (s *ChatService) executeAgentResponse(ctx context.Context, ag agent.Agent, msg *pb.ChatMessage) (string, error) {
	// This would integrate with the actual agent execution system
	// For now, return a demo response
	return fmt.Sprintf("Agent %s (%s) received: %s", ag.GetName(), ag.GetID(), msg.Content), nil
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
	
	// Simulate tool execution (this would integrate with the actual tool system)
	time.Sleep(time.Second) // Simulate work
	
	toolExec.Status = pb.ToolExecution_COMPLETED
	toolExec.Progress = 1.0
	toolExec.Result = "Tool execution completed successfully"
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
		ID:         sessionID,
		Name:       req.Name,
		AgentIDs:   req.AgentIds,
		CampaignID: req.CampaignId,
		Status:     pb.ChatSession_ACTIVE,
		CreatedAt:  time.Now(),
		LastActivity: time.Now(),
		Context:    req.Context,
		Metadata:   req.Metadata,
		agents:     make(map[string]agent.Agent),
		messages:   make([]*pb.ChatMessage, 0),
		toolExecutions: make(map[string]*pb.ToolExecution),
		streams:    make(map[string]pb.ChatService_ChatServer),
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
		TotalMessages:   int32(len(session.messages)),
		ToolsExecuted:   int32(len(session.toolExecutions)),
		AgentsInvolved:  session.AgentIDs,
		Outcome:         req.Reason,
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