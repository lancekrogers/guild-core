// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/storage"
)

// memorySessionService provides an in-memory implementation of SessionService
type memorySessionService struct {
	pb.UnimplementedSessionServiceServer

	sessions   map[string]*storage.ChatSession
	messages   map[string][]*storage.ChatMessage // session_id -> messages
	sessionsMu sync.RWMutex
	messagesMu sync.RWMutex

	// Event channels for streaming
	eventChannels map[string]chan *pb.SessionEvent
	subscribers   map[string]map[string]chan *pb.Message // session_id -> subscriber_id -> channel
	eventMu       sync.RWMutex
	subMu         sync.RWMutex
}

// NewMemorySessionService creates a new in-memory session service
func NewMemorySessionService() pb.SessionServiceServer {
	return &memorySessionService{
		sessions:      make(map[string]*storage.ChatSession),
		messages:      make(map[string][]*storage.ChatMessage),
		eventChannels: make(map[string]chan *pb.SessionEvent),
		subscribers:   make(map[string]map[string]chan *pb.Message),
	}
}

// Session Management

func (s *memorySessionService) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.Session, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("CreateSession")

	sessionID := fmt.Sprintf("session_%s", uuid.New().String())

	session := &storage.ChatSession{
		ID:        sessionID,
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if req.CampaignId != nil && *req.CampaignId != "" {
		session.CampaignID = req.CampaignId
	}

	// Convert metadata
	for k, v := range req.Metadata {
		session.Metadata[k] = v
	}

	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	// Initialize empty message list
	s.messagesMu.Lock()
	s.messages[sessionID] = make([]*storage.ChatMessage, 0)
	s.messagesMu.Unlock()

	// Publish session created event
	s.publishEvent(&pb.SessionEvent{
		SessionId:   sessionID,
		Type:        pb.SessionEvent_SESSION_CREATED,
		Description: fmt.Sprintf("Session '%s' created", req.Name),
		Data:        req.Metadata,
		Timestamp:   timestamppb.Now(),
	})

	logger.Info("Session created successfully", "session_id", sessionID)

	return s.storageSessionToProto(session), nil
}

func (s *memorySessionService) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.Session, error) {
	s.sessionsMu.RLock()
	session, exists := s.sessions[req.Id]
	s.sessionsMu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "session not found: %s", req.Id)
	}

	return s.storageSessionToProto(session), nil
}

func (s *memorySessionService) ListSessions(ctx context.Context, req *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*pb.Session, 0)
	count := 0
	skipped := 0

	for _, session := range s.sessions {
		// Filter by campaign if specified
		if req.CampaignId != nil && *req.CampaignId != "" {
			if session.CampaignID == nil || *session.CampaignID != *req.CampaignId {
				continue
			}
		}

		// Handle pagination
		if skipped < int(req.Offset) {
			skipped++
			continue
		}

		if req.Limit > 0 && count >= int(req.Limit) {
			break
		}

		sessions = append(sessions, s.storageSessionToProto(session))
		count++
	}

	return &pb.ListSessionsResponse{
		Sessions:   sessions,
		TotalCount: int64(len(s.sessions)),
	}, nil
}

func (s *memorySessionService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.Session, error) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	session, exists := s.sessions[req.Id]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "session not found: %s", req.Id)
	}

	session.Name = req.Name
	session.UpdatedAt = time.Now()

	// Update metadata
	for k, v := range req.Metadata {
		session.Metadata[k] = v
	}

	// Publish session updated event
	s.publishEvent(&pb.SessionEvent{
		SessionId:   req.Id,
		Type:        pb.SessionEvent_SESSION_UPDATED,
		Description: "Session updated",
		Data:        req.Metadata,
		Timestamp:   timestamppb.Now(),
	})

	return s.storageSessionToProto(session), nil
}

func (s *memorySessionService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.DeleteSessionResponse, error) {
	s.sessionsMu.Lock()
	delete(s.sessions, req.Id)
	s.sessionsMu.Unlock()

	s.messagesMu.Lock()
	delete(s.messages, req.Id)
	s.messagesMu.Unlock()

	// Publish session deleted event
	s.publishEvent(&pb.SessionEvent{
		SessionId:   req.Id,
		Type:        pb.SessionEvent_SESSION_DELETED,
		Description: "Session deleted",
		Timestamp:   timestamppb.Now(),
	})

	return &pb.DeleteSessionResponse{
		Success: true,
		Message: "Session deleted successfully",
	}, nil
}

// Message Management

func (s *memorySessionService) SaveMessage(ctx context.Context, req *pb.SaveMessageRequest) (*pb.SaveMessageResponse, error) {
	msg := req.Message
	if msg == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	// Generate message ID if not provided
	if msg.Id == "" {
		msg.Id = fmt.Sprintf("msg_%s", uuid.New().String())
	}

	// Create storage message
	storageMsg := &storage.ChatMessage{
		ID:        msg.Id,
		SessionID: msg.SessionId,
		Role:      s.protoRoleToString(msg.Role),
		Content:   msg.Content,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Convert metadata
	for k, v := range msg.Metadata {
		storageMsg.Metadata[k] = v
	}

	s.messagesMu.Lock()
	if _, exists := s.messages[msg.SessionId]; !exists {
		s.messages[msg.SessionId] = make([]*storage.ChatMessage, 0)
	}
	s.messages[msg.SessionId] = append(s.messages[msg.SessionId], storageMsg)
	s.messagesMu.Unlock()

	// Update session timestamp
	s.sessionsMu.Lock()
	if session, exists := s.sessions[msg.SessionId]; exists {
		session.UpdatedAt = time.Now()
	}
	s.sessionsMu.Unlock()

	// Publish message added event
	s.publishEvent(&pb.SessionEvent{
		SessionId:   msg.SessionId,
		Type:        pb.SessionEvent_MESSAGE_ADDED,
		Description: fmt.Sprintf("Message added by %s", storageMsg.Role),
		Data:        map[string]string{"message_id": msg.Id},
		Timestamp:   timestamppb.Now(),
	})

	// Notify message stream subscribers
	s.notifyMessageSubscribers(msg.SessionId, msg)

	return &pb.SaveMessageResponse{
		Success:   true,
		MessageId: msg.Id,
	}, nil
}

func (s *memorySessionService) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.Message, error) {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == req.Id {
				return s.storageMessageToProto(msg), nil
			}
		}
	}

	return nil, status.Errorf(codes.NotFound, "message not found: %s", req.Id)
}

func (s *memorySessionService) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	messages, exists := s.messages[req.SessionId]
	if !exists {
		return &pb.GetMessagesResponse{
			Messages:   []*pb.Message{},
			TotalCount: 0,
			HasMore:    false,
		}, nil
	}

	protoMessages := make([]*pb.Message, len(messages))
	for i, msg := range messages {
		protoMessages[i] = s.storageMessageToProto(msg)
	}

	return &pb.GetMessagesResponse{
		Messages:   protoMessages,
		TotalCount: int64(len(messages)),
		HasMore:    false,
	}, nil
}

func (s *memorySessionService) GetMessagesPaginated(ctx context.Context, req *pb.GetMessagesPaginatedRequest) (*pb.GetMessagesResponse, error) {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	messages, exists := s.messages[req.SessionId]
	if !exists {
		return &pb.GetMessagesResponse{
			Messages:   []*pb.Message{},
			TotalCount: 0,
			HasMore:    false,
		}, nil
	}

	// Apply pagination
	start := int(req.Offset)
	end := start + int(req.Limit)
	if start >= len(messages) {
		return &pb.GetMessagesResponse{
			Messages:   []*pb.Message{},
			TotalCount: int64(len(messages)),
			HasMore:    false,
		}, nil
	}
	if end > len(messages) {
		end = len(messages)
	}

	paginatedMessages := messages[start:end]
	protoMessages := make([]*pb.Message, len(paginatedMessages))
	for i, msg := range paginatedMessages {
		protoMessages[i] = s.storageMessageToProto(msg)
	}

	return &pb.GetMessagesResponse{
		Messages:   protoMessages,
		TotalCount: int64(len(messages)),
		HasMore:    end < len(messages),
	}, nil
}

func (s *memorySessionService) GetMessagesAfter(ctx context.Context, req *pb.GetMessagesAfterRequest) (*pb.GetMessagesResponse, error) {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	messages, exists := s.messages[req.SessionId]
	if !exists {
		return &pb.GetMessagesResponse{
			Messages:   []*pb.Message{},
			TotalCount: 0,
			HasMore:    false,
		}, nil
	}

	afterTime := req.After.AsTime()
	filteredMessages := make([]*pb.Message, 0)

	for _, msg := range messages {
		if msg.CreatedAt.After(afterTime) {
			filteredMessages = append(filteredMessages, s.storageMessageToProto(msg))
		}
	}

	return &pb.GetMessagesResponse{
		Messages:   filteredMessages,
		TotalCount: int64(len(messages)),
		HasMore:    false,
	}, nil
}

func (s *memorySessionService) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.DeleteMessageResponse, error) {
	s.messagesMu.Lock()
	defer s.messagesMu.Unlock()

	for sessionID, messages := range s.messages {
		for i, msg := range messages {
			if msg.ID == req.Id {
				// Remove message from slice
				s.messages[sessionID] = append(messages[:i], messages[i+1:]...)

				// Publish message deleted event
				s.publishEvent(&pb.SessionEvent{
					SessionId:   sessionID,
					Type:        pb.SessionEvent_MESSAGE_DELETED,
					Description: "Message deleted",
					Data:        map[string]string{"message_id": req.Id},
					Timestamp:   timestamppb.Now(),
				})

				return &pb.DeleteMessageResponse{
					Success: true,
				}, nil
			}
		}
	}

	return nil, status.Errorf(codes.NotFound, "message not found: %s", req.Id)
}

// Streaming

func (s *memorySessionService) StreamMessages(req *pb.StreamMessagesRequest, stream pb.SessionService_StreamMessagesServer) error {
	ctx := stream.Context()
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("StreamMessages")

	subscriberID := uuid.New().String()
	msgChan := make(chan *pb.Message, 10)

	// Register subscriber
	s.addMessageSubscriber(req.SessionId, subscriberID, msgChan)
	defer s.removeMessageSubscriber(req.SessionId, subscriberID)

	logger.Info("Message stream started", "session_id", req.SessionId, "subscriber_id", subscriberID)

	// Send existing messages if requested
	if req.Since != nil {
		s.messagesMu.RLock()
		messages, exists := s.messages[req.SessionId]
		s.messagesMu.RUnlock()

		if exists {
			afterTime := req.Since.AsTime()
			for _, msg := range messages {
				if msg.CreatedAt.After(afterTime) {
					if err := stream.Send(s.storageMessageToProto(msg)); err != nil {
						return err
					}
				}
			}
		}
	}

	// Stream new messages
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgChan:
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}

func (s *memorySessionService) StreamSessionEvents(req *pb.StreamSessionEventsRequest, stream pb.SessionService_StreamSessionEventsServer) error {
	ctx := stream.Context()
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("StreamSessionEvents")

	eventChan := make(chan *pb.SessionEvent, 10)
	channelID := uuid.New().String()

	// Register event channel
	s.eventMu.Lock()
	s.eventChannels[channelID] = eventChan
	s.eventMu.Unlock()

	defer func() {
		s.eventMu.Lock()
		delete(s.eventChannels, channelID)
		s.eventMu.Unlock()
	}()

	logger.Info("Event stream started", "session_id", req.GetSessionId(), "channel_id", channelID)

	// Stream events
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-eventChan:
			// Filter by session if specified
			if req.SessionId != nil && *req.SessionId != "" && event.SessionId != *req.SessionId {
				continue
			}

			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

// Helper methods

func (s *memorySessionService) storageSessionToProto(session *storage.ChatSession) *pb.Session {
	proto := &pb.Session{
		Id:        session.ID,
		Name:      session.Name,
		CreatedAt: timestamppb.New(session.CreatedAt),
		UpdatedAt: timestamppb.New(session.UpdatedAt),
		Metadata:  make(map[string]string),
	}

	if session.CampaignID != nil {
		proto.CampaignId = session.CampaignID
	}

	// Convert metadata
	for k, v := range session.Metadata {
		if str, ok := v.(string); ok {
			proto.Metadata[k] = str
		}
	}

	return proto
}

func (s *memorySessionService) storageMessageToProto(msg *storage.ChatMessage) *pb.Message {
	proto := &pb.Message{
		Id:        msg.ID,
		SessionId: msg.SessionID,
		Role:      s.stringRoleToProto(msg.Role),
		Content:   msg.Content,
		CreatedAt: timestamppb.New(msg.CreatedAt),
		Metadata:  make(map[string]string),
	}

	// Convert metadata
	for k, v := range msg.Metadata {
		if str, ok := v.(string); ok {
			proto.Metadata[k] = str
		}
	}

	// TODO: Handle tool calls if needed

	return proto
}

func (s *memorySessionService) protoRoleToString(role pb.Message_MessageRole) string {
	switch role {
	case pb.Message_SYSTEM:
		return "system"
	case pb.Message_USER:
		return "user"
	case pb.Message_ASSISTANT:
		return "assistant"
	case pb.Message_TOOL:
		return "tool"
	default:
		return "user"
	}
}

func (s *memorySessionService) stringRoleToProto(role string) pb.Message_MessageRole {
	switch role {
	case "system":
		return pb.Message_SYSTEM
	case "user":
		return pb.Message_USER
	case "assistant":
		return pb.Message_ASSISTANT
	case "tool":
		return pb.Message_TOOL
	default:
		return pb.Message_USER
	}
}

func (s *memorySessionService) publishEvent(event *pb.SessionEvent) {
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()

	// Send to all registered channels
	for _, ch := range s.eventChannels {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

func (s *memorySessionService) addMessageSubscriber(sessionID, subscriberID string, ch chan *pb.Message) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	if s.subscribers[sessionID] == nil {
		s.subscribers[sessionID] = make(map[string]chan *pb.Message)
	}
	s.subscribers[sessionID][subscriberID] = ch
}

func (s *memorySessionService) removeMessageSubscriber(sessionID, subscriberID string) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	if subs, ok := s.subscribers[sessionID]; ok {
		delete(subs, subscriberID)
		if len(subs) == 0 {
			delete(s.subscribers, sessionID)
		}
	}
}

func (s *memorySessionService) notifyMessageSubscribers(sessionID string, msg *pb.Message) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	if subs, ok := s.subscribers[sessionID]; ok {
		for _, ch := range subs {
			select {
			case ch <- msg:
			default:
				// Channel full, skip
			}
		}
	}
}
