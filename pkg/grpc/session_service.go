// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/storage"
)

// sessionService implements the SessionService gRPC service with enterprise-grade quality
type sessionService struct {
	v1.UnimplementedSessionServiceServer
	repository storage.SessionRepository

	// Connection management and health tracking
	healthy    bool
	lastHealth time.Time
	mu         sync.RWMutex
}

// NewSessionService creates a new SessionService with comprehensive validation
func NewSessionService(repository storage.SessionRepository) v1.SessionServiceServer {
	if repository == nil {
		panic("repository cannot be nil")
	}

	s := &sessionService{
		repository: repository,
		healthy:    true,
		lastHealth: time.Now(),
	}

	// Start health monitoring
	go s.healthMonitor()

	return s
}

// healthMonitor periodically checks service health
func (s *sessionService) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := s.repository.CountSessions(ctx)
		cancel()

		s.mu.Lock()
		s.healthy = (err == nil)
		s.lastHealth = time.Now()
		s.mu.Unlock()
	}
}

// isHealthy returns current service health status
func (s *sessionService) isHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

// Validation constants for enterprise-grade input validation
const (
	maxSessionNameLength   = 255
	maxMetadataEntries     = 50
	maxMetadataKeyLength   = 100
	maxMetadataValueLength = 1000
	maxContentLength       = 1_000_000 // 1MB
)

// validateSessionName validates session name according to business rules
func (s *sessionService) validateSessionName(name string) error {
	if strings.TrimSpace(name) == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "session name cannot be empty", nil).
			WithComponent("grpc").
			WithOperation("validateSessionName")
	}
	if len(name) > maxSessionNameLength {
		return gerror.New(gerror.ErrCodeInvalidInput,
			fmt.Sprintf("session name too long: %d > %d", len(name), maxSessionNameLength), nil).
			WithComponent("grpc").
			WithOperation("validateSessionName")
	}
	return nil
}

// validateMetadata validates metadata entries
func (s *sessionService) validateMetadata(metadata map[string]string) error {
	if len(metadata) > maxMetadataEntries {
		return gerror.New(gerror.ErrCodeInvalidInput,
			fmt.Sprintf("too many metadata entries: %d > %d", len(metadata), maxMetadataEntries), nil).
			WithComponent("grpc").
			WithOperation("validateMetadata")
	}

	for key, value := range metadata {
		if len(key) > maxMetadataKeyLength {
			return gerror.New(gerror.ErrCodeInvalidInput,
				fmt.Sprintf("metadata key too long: %d > %d", len(key), maxMetadataKeyLength), nil).
				WithComponent("grpc").
				WithOperation("validateMetadata")
		}
		if len(value) > maxMetadataValueLength {
			return gerror.New(gerror.ErrCodeInvalidInput,
				fmt.Sprintf("metadata value too long: %d > %d", len(value), maxMetadataValueLength), nil).
				WithComponent("grpc").
				WithOperation("validateMetadata")
		}
	}
	return nil
}

// validateMessageContent validates message content
func (s *sessionService) validateMessageContent(content string) error {
	if len(content) > maxContentLength {
		return gerror.New(gerror.ErrCodeInvalidInput,
			fmt.Sprintf("message content too long: %d > %d", len(content), maxContentLength), nil).
			WithComponent("grpc").
			WithOperation("validateMessageContent")
	}
	return nil
}

// convertToProtoSession converts storage.ChatSession to proto Session with nil safety
func (s *sessionService) convertToProtoSession(cs *storage.ChatSession) *v1.Session {
	if cs == nil {
		return nil
	}

	protoSession := &v1.Session{
		Id:        cs.ID,
		Name:      cs.Name,
		CreatedAt: timestamppb.New(cs.CreatedAt),
		UpdatedAt: timestamppb.New(cs.UpdatedAt),
		Metadata:  make(map[string]string),
	}

	if cs.CampaignID != nil {
		protoSession.CampaignId = cs.CampaignID
	}

	// Convert metadata with nil safety and type validation
	if cs.Metadata != nil {
		for k, v := range cs.Metadata {
			if str, ok := v.(string); ok {
				protoSession.Metadata[k] = str
			}
		}
	}

	return protoSession
}

// convertFromProtoSession converts proto Session to storage.ChatSession
func (s *sessionService) convertFromProtoSession(ps *v1.Session) *storage.ChatSession {
	cs := &storage.ChatSession{
		ID:        ps.Id,
		Name:      ps.Name,
		CreatedAt: ps.CreatedAt.AsTime(),
		UpdatedAt: ps.UpdatedAt.AsTime(),
		Metadata:  make(map[string]interface{}),
	}

	if ps.CampaignId != nil {
		cs.CampaignID = ps.CampaignId
	}

	// Convert metadata
	for k, v := range ps.Metadata {
		cs.Metadata[k] = v
	}

	return cs
}

// convertToProtoMessage converts storage.ChatMessage to proto Message
func (s *sessionService) convertToProtoMessage(cm *storage.ChatMessage) *v1.Message {
	protoMessage := &v1.Message{
		Id:        cm.ID,
		SessionId: cm.SessionID,
		Role:      s.convertToProtoRole(cm.Role),
		Content:   cm.Content,
		CreatedAt: timestamppb.New(cm.CreatedAt),
		Metadata:  make(map[string]string),
	}

	// Convert metadata
	for k, v := range cm.Metadata {
		if str, ok := v.(string); ok {
			protoMessage.Metadata[k] = str
		}
	}

	// TODO: Convert tool calls when needed
	// For now, tool calls are stored as JSON in the database
	// and can be extracted when needed

	return protoMessage
}

// convertFromProtoMessage converts proto Message to storage.ChatMessage
func (s *sessionService) convertFromProtoMessage(pm *v1.Message) *storage.ChatMessage {
	cm := &storage.ChatMessage{
		ID:        pm.Id,
		SessionID: pm.SessionId,
		Role:      s.convertFromProtoRole(pm.Role),
		Content:   pm.Content,
		CreatedAt: pm.CreatedAt.AsTime(),
		Metadata:  make(map[string]interface{}),
	}

	// Convert metadata
	for k, v := range pm.Metadata {
		cm.Metadata[k] = v
	}

	// TODO: Convert tool calls when needed
	// For now, tool calls are handled separately

	return cm
}

// convertToProtoRole converts string role to proto MessageRole
func (s *sessionService) convertToProtoRole(role string) v1.Message_MessageRole {
	switch role {
	case "system":
		return v1.Message_SYSTEM
	case "user":
		return v1.Message_USER
	case "assistant":
		return v1.Message_ASSISTANT
	case "tool":
		return v1.Message_TOOL
	default:
		return v1.Message_USER
	}
}

// convertFromProtoRole converts proto MessageRole to string
func (s *sessionService) convertFromProtoRole(role v1.Message_MessageRole) string {
	switch role {
	case v1.Message_SYSTEM:
		return "system"
	case v1.Message_USER:
		return "user"
	case v1.Message_ASSISTANT:
		return "assistant"
	case v1.Message_TOOL:
		return "tool"
	default:
		return "user"
	}
}

// Session operations with comprehensive validation and observability
func (s *sessionService) CreateSession(ctx context.Context, req *v1.CreateSessionRequest) (*v1.Session, error) {
	// Health check first
	if !s.isHealthy() {
		return nil, status.Error(codes.Unavailable, "service unhealthy")
	}

	// Context validation
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.DeadlineExceeded, "context cancelled")
	}

	// Request validation
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Comprehensive input validation
	if err := s.validateSessionName(req.Name); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.validateMetadata(req.Metadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Observability
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("CreateSession")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logger.Info("CreateSession completed",
			"duration_ms", duration.Milliseconds(),
			"session_name", req.Name,
		)
	}()

	logger.Info("Creating session",
		"session_name", req.Name,
		"campaign_id", func() string {
			if req.CampaignId != nil {
				return *req.CampaignId
			}
			return "none"
		}(),
	)

	// Create session with generated ID
	session := &storage.ChatSession{
		ID:        uuid.New().String(),
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if req.CampaignId != nil && *req.CampaignId != "" {
		session.CampaignID = req.CampaignId
	}

	// Convert metadata with validation
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			session.Metadata[k] = v
		}
	}

	if err := s.repository.CreateSession(ctx, session); err != nil {
		logger.WithError(err).Error("Failed to create session in repository",
			"session_id", session.ID,
		)
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	logger.Info("Session created successfully",
		"session_id", session.ID,
		"session_name", session.Name,
	)

	return s.convertToProtoSession(session), nil
}

func (s *sessionService) GetSession(ctx context.Context, req *v1.GetSessionRequest) (*v1.Session, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	session, err := s.repository.GetSession(ctx, req.Id)
	if err != nil {
		if gerror.Is(err, gerror.ErrCodeNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	return s.convertToProtoSession(session), nil
}

func (s *sessionService) ListSessions(ctx context.Context, req *v1.ListSessionsRequest) (*v1.ListSessionsResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	var sessions []*storage.ChatSession
	var err error

	if req.CampaignId != nil {
		sessions, err = s.repository.ListSessionsByCampaign(ctx, *req.CampaignId)
	} else {
		sessions, err = s.repository.ListSessions(ctx, limit, req.Offset)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	totalCount, err := s.repository.CountSessions(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count sessions: %v", err)
	}

	protoSessions := make([]*v1.Session, len(sessions))
	for i, session := range sessions {
		protoSessions[i] = s.convertToProtoSession(session)
	}

	return &v1.ListSessionsResponse{
		Sessions:   protoSessions,
		TotalCount: totalCount,
	}, nil
}

func (s *sessionService) UpdateSession(ctx context.Context, req *v1.UpdateSessionRequest) (*v1.Session, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	// Get existing session
	existingSession, err := s.repository.GetSession(ctx, req.Id)
	if err != nil {
		if gerror.Is(err, gerror.ErrCodeNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Update fields
	existingSession.Name = req.Name
	existingSession.UpdatedAt = time.Now()

	// Update metadata
	for k, v := range req.Metadata {
		existingSession.Metadata[k] = v
	}

	if err := s.repository.UpdateSession(ctx, existingSession); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update session: %v", err)
	}

	return s.convertToProtoSession(existingSession), nil
}

func (s *sessionService) DeleteSession(ctx context.Context, req *v1.DeleteSessionRequest) (*v1.DeleteSessionResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	if err := s.repository.DeleteSession(ctx, req.Id); err != nil {
		if gerror.Is(err, gerror.ErrCodeNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", err)
	}

	return &v1.DeleteSessionResponse{
		Success: true,
		Message: "Session deleted successfully",
	}, nil
}

// Message operations
func (s *sessionService) SaveMessage(ctx context.Context, req *v1.SaveMessageRequest) (*v1.SaveMessageResponse, error) {
	if req.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	message := s.convertFromProtoMessage(req.Message)

	// Generate ID if not provided
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	// Set created timestamp if not provided
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	if err := s.repository.SaveMessage(ctx, message); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save message: %v", err)
	}

	return &v1.SaveMessageResponse{
		Success:   true,
		MessageId: message.ID,
	}, nil
}

func (s *sessionService) GetMessage(ctx context.Context, req *v1.GetMessageRequest) (*v1.Message, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "message ID is required")
	}

	message, err := s.repository.GetMessage(ctx, req.Id)
	if err != nil {
		if gerror.Is(err, gerror.ErrCodeNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get message: %v", err)
	}

	return s.convertToProtoMessage(message), nil
}

func (s *sessionService) GetMessages(ctx context.Context, req *v1.GetMessagesRequest) (*v1.GetMessagesResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	messages, err := s.repository.GetMessages(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	totalCount, err := s.repository.CountMessages(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count messages: %v", err)
	}

	protoMessages := make([]*v1.Message, len(messages))
	for i, message := range messages {
		protoMessages[i] = s.convertToProtoMessage(message)
	}

	return &v1.GetMessagesResponse{
		Messages:   protoMessages,
		TotalCount: totalCount,
		HasMore:    false, // For non-paginated requests
	}, nil
}

func (s *sessionService) GetMessagesPaginated(ctx context.Context, req *v1.GetMessagesPaginatedRequest) (*v1.GetMessagesResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	messages, err := s.repository.GetMessagesPaginated(ctx, req.SessionId, limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	totalCount, err := s.repository.CountMessages(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count messages: %v", err)
	}

	protoMessages := make([]*v1.Message, len(messages))
	for i, message := range messages {
		protoMessages[i] = s.convertToProtoMessage(message)
	}

	hasMore := int64(req.Offset+limit) < totalCount

	return &v1.GetMessagesResponse{
		Messages:   protoMessages,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

func (s *sessionService) GetMessagesAfter(ctx context.Context, req *v1.GetMessagesAfterRequest) (*v1.GetMessagesResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	after := req.After.AsTime()
	messages, err := s.repository.GetMessagesAfter(ctx, req.SessionId, after)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get messages after timestamp: %v", err)
	}

	totalCount, err := s.repository.CountMessages(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count messages: %v", err)
	}

	protoMessages := make([]*v1.Message, len(messages))
	for i, message := range messages {
		protoMessages[i] = s.convertToProtoMessage(message)
	}

	return &v1.GetMessagesResponse{
		Messages:   protoMessages,
		TotalCount: totalCount,
		HasMore:    false, // For timestamp-based requests
	}, nil
}

func (s *sessionService) DeleteMessage(ctx context.Context, req *v1.DeleteMessageRequest) (*v1.DeleteMessageResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "message ID is required")
	}

	if err := s.repository.DeleteMessage(ctx, req.Id); err != nil {
		if gerror.Is(err, gerror.ErrCodeNotFound) {
			return nil, status.Error(codes.NotFound, "message not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete message: %v", err)
	}

	return &v1.DeleteMessageResponse{
		Success: true,
	}, nil
}

// Streaming operations
func (s *sessionService) StreamMessages(req *v1.StreamMessagesRequest, stream v1.SessionService_StreamMessagesServer) error {
	if req.SessionId == "" {
		return status.Error(codes.InvalidArgument, "session ID is required")
	}

	since := req.Since.AsTime()
	messageChan, err := s.repository.StreamMessages(stream.Context(), req.SessionId, since)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to start message stream: %v", err)
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case message, ok := <-messageChan:
			if !ok {
				return nil // Stream closed
			}
			protoMessage := s.convertToProtoMessage(message)
			if err := stream.Send(protoMessage); err != nil {
				return status.Errorf(codes.Internal, "failed to send message: %v", err)
			}
		}
	}
}

func (s *sessionService) StreamSessionEvents(req *v1.StreamSessionEventsRequest, stream v1.SessionService_StreamSessionEventsServer) error {
	// TODO: Implement session event streaming
	// This would require an event bus or similar mechanism
	// For now, return not implemented
	return status.Error(codes.Unimplemented, "session event streaming not yet implemented")
}

// memorySessionRepository provides a simple in-memory implementation of SessionRepository
type memorySessionRepository struct {
	sessions map[string]*storage.ChatSession
	messages map[string]*storage.ChatMessage
	mu       sync.RWMutex
}

// Session operations
func (m *memorySessionRepository) CreateSession(ctx context.Context, session *storage.ChatSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

func (m *memorySessionRepository) GetSession(ctx context.Context, id string) (*storage.ChatSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if session, ok := m.sessions[id]; ok {
		return session, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "session not found", nil)
}

func (m *memorySessionRepository) ListSessions(ctx context.Context, limit, offset int32) ([]*storage.ChatSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sessions []*storage.ChatSession
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *memorySessionRepository) ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*storage.ChatSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var sessions []*storage.ChatSession
	for _, session := range m.sessions {
		if session.CampaignID != nil && *session.CampaignID == campaignID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (m *memorySessionRepository) UpdateSession(ctx context.Context, session *storage.ChatSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

func (m *memorySessionRepository) DeleteSession(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *memorySessionRepository) CountSessions(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.sessions)), nil
}

// Message operations
func (m *memorySessionRepository) SaveMessage(ctx context.Context, message *storage.ChatMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[message.ID] = message
	return nil
}

func (m *memorySessionRepository) GetMessage(ctx context.Context, id string) (*storage.ChatMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if message, ok := m.messages[id]; ok {
		return message, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "message not found", nil)
}

func (m *memorySessionRepository) GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var messages []*storage.ChatMessage
	for _, message := range m.messages {
		if message.SessionID == sessionID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (m *memorySessionRepository) GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*storage.ChatMessage, error) {
	messages, err := m.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// Simple pagination
	start := int(offset)
	end := start + int(limit)
	if start >= len(messages) {
		return []*storage.ChatMessage{}, nil
	}
	if end > len(messages) {
		end = len(messages)
	}
	return messages[start:end], nil
}

func (m *memorySessionRepository) GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*storage.ChatMessage, error) {
	messages, err := m.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var filtered []*storage.ChatMessage
	for _, message := range messages {
		if message.CreatedAt.After(after) {
			filtered = append(filtered, message)
		}
	}
	return filtered, nil
}

func (m *memorySessionRepository) CountMessages(ctx context.Context, sessionID string) (int64, error) {
	messages, err := m.GetMessages(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int64(len(messages)), nil
}

func (m *memorySessionRepository) DeleteMessage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.messages, id)
	return nil
}

func (m *memorySessionRepository) StreamMessages(ctx context.Context, sessionID string, since time.Time) (<-chan *storage.ChatMessage, error) {
	messageChan := make(chan *storage.ChatMessage, 10)
	go func() {
		defer close(messageChan)
		// For memory implementation, just return existing messages and close
		messages, err := m.GetMessagesAfter(ctx, sessionID, since)
		if err != nil {
			return
		}
		for _, message := range messages {
			select {
			case messageChan <- message:
			case <-ctx.Done():
				return
			}
		}
	}()
	return messageChan, nil
}
