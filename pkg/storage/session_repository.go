// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lancekrogers/guild/internal/ui/chat/session"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// sqliteSessionRepository implements SessionRepository by adapting the existing session.SessionStore
type sqliteSessionRepository struct {
	store session.SessionStore
}

// NewSQLiteSessionRepository creates a new SessionRepository using the existing session store
func NewSQLiteSessionRepository(store session.SessionStore) SessionRepository {
	return &sqliteSessionRepository{
		store: store,
	}
}

// convertToChatSession converts from session.Session to storage.ChatSession
func (r *sqliteSessionRepository) convertToChatSession(s *session.Session) *ChatSession {
	return &ChatSession{
		ID:         s.ID,
		Name:       s.Name,
		CampaignID: s.CampaignID,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
		Metadata:   s.Metadata,
	}
}

// convertFromChatSession converts from storage.ChatSession to session.Session
func (r *sqliteSessionRepository) convertFromChatSession(cs *ChatSession) *session.Session {
	return &session.Session{
		ID:         cs.ID,
		Name:       cs.Name,
		CampaignID: cs.CampaignID,
		CreatedAt:  cs.CreatedAt,
		UpdatedAt:  cs.UpdatedAt,
		Metadata:   cs.Metadata,
	}
}

// convertToChatMessage converts from session.Message to storage.ChatMessage
func (r *sqliteSessionRepository) convertToChatMessage(m *session.Message) (*ChatMessage, error) {
	// Convert tool calls to JSON
	var toolCallsJSON map[string]interface{}
	if len(m.ToolCalls) > 0 {
		toolCallsData, err := json.Marshal(m.ToolCalls)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to marshal tool calls")
		}
		if err := json.Unmarshal(toolCallsData, &toolCallsJSON); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to unmarshal tool calls")
		}
	}

	return &ChatMessage{
		ID:        m.ID,
		SessionID: m.SessionID,
		Role:      string(m.Role),
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		ToolCalls: toolCallsJSON,
		Metadata:  m.Metadata,
	}, nil
}

// convertFromChatMessage converts from storage.ChatMessage to session.Message
func (r *sqliteSessionRepository) convertFromChatMessage(cm *ChatMessage) (*session.Message, error) {
	// Convert tool calls from JSON
	var toolCalls []session.ToolCall
	if cm.ToolCalls != nil {
		toolCallsData, err := json.Marshal(cm.ToolCalls)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to marshal tool calls JSON")
		}
		if err := json.Unmarshal(toolCallsData, &toolCalls); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to unmarshal tool calls JSON")
		}
	}

	return &session.Message{
		ID:        cm.ID,
		SessionID: cm.SessionID,
		Role:      session.MessageRole(cm.Role),
		Content:   cm.Content,
		CreatedAt: cm.CreatedAt,
		ToolCalls: toolCalls,
		Metadata:  cm.Metadata,
	}, nil
}

// Session operations
func (r *sqliteSessionRepository) CreateSession(ctx context.Context, chatSession *ChatSession) error {
	sessionObj := r.convertFromChatSession(chatSession)
	return r.store.CreateSession(ctx, sessionObj)
}

func (r *sqliteSessionRepository) GetSession(ctx context.Context, id string) (*ChatSession, error) {
	sessionObj, err := r.store.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.convertToChatSession(sessionObj), nil
}

func (r *sqliteSessionRepository) ListSessions(ctx context.Context, limit, offset int32) ([]*ChatSession, error) {
	sessions, err := r.store.ListSessions(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*ChatSession, len(sessions))
	for i, s := range sessions {
		result[i] = r.convertToChatSession(s)
	}
	return result, nil
}

func (r *sqliteSessionRepository) ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*ChatSession, error) {
	sessions, err := r.store.ListSessionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	result := make([]*ChatSession, len(sessions))
	for i, s := range sessions {
		result[i] = r.convertToChatSession(s)
	}
	return result, nil
}

func (r *sqliteSessionRepository) UpdateSession(ctx context.Context, chatSession *ChatSession) error {
	sessionObj := r.convertFromChatSession(chatSession)
	return r.store.UpdateSession(ctx, sessionObj)
}

func (r *sqliteSessionRepository) DeleteSession(ctx context.Context, id string) error {
	return r.store.DeleteSession(ctx, id)
}

func (r *sqliteSessionRepository) CountSessions(ctx context.Context) (int64, error) {
	return r.store.CountSessions(ctx)
}

// Message operations
func (r *sqliteSessionRepository) SaveMessage(ctx context.Context, chatMessage *ChatMessage) error {
	messageObj, err := r.convertFromChatMessage(chatMessage)
	if err != nil {
		return err
	}
	return r.store.SaveMessage(ctx, messageObj)
}

func (r *sqliteSessionRepository) GetMessage(ctx context.Context, id string) (*ChatMessage, error) {
	messageObj, err := r.store.GetMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.convertToChatMessage(messageObj)
}

func (r *sqliteSessionRepository) GetMessages(ctx context.Context, sessionID string) ([]*ChatMessage, error) {
	messages, err := r.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	result := make([]*ChatMessage, len(messages))
	for i, m := range messages {
		converted, convErr := r.convertToChatMessage(m)
		if convErr != nil {
			return nil, convErr
		}
		result[i] = converted
	}
	return result, nil
}

func (r *sqliteSessionRepository) GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*ChatMessage, error) {
	messages, err := r.store.GetMessagesPaginated(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*ChatMessage, len(messages))
	for i, m := range messages {
		converted, convErr := r.convertToChatMessage(m)
		if convErr != nil {
			return nil, convErr
		}
		result[i] = converted
	}
	return result, nil
}

func (r *sqliteSessionRepository) GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*ChatMessage, error) {
	messages, err := r.store.GetMessagesAfter(ctx, sessionID, after)
	if err != nil {
		return nil, err
	}

	result := make([]*ChatMessage, len(messages))
	for i, m := range messages {
		converted, convErr := r.convertToChatMessage(m)
		if convErr != nil {
			return nil, convErr
		}
		result[i] = converted
	}
	return result, nil
}

func (r *sqliteSessionRepository) CountMessages(ctx context.Context, sessionID string) (int64, error) {
	return r.store.CountMessages(ctx, sessionID)
}

func (r *sqliteSessionRepository) DeleteMessage(ctx context.Context, id string) error {
	return r.store.DeleteMessage(ctx, id)
}

// StreamMessages implements real-time message streaming for the daemon
func (r *sqliteSessionRepository) StreamMessages(ctx context.Context, sessionID string, since time.Time) (<-chan *ChatMessage, error) {
	messageChan := make(chan *ChatMessage, 100) // Buffer for smooth streaming

	go func() {
		defer close(messageChan)

		ticker := time.NewTicker(100 * time.Millisecond) // Poll every 100ms
		defer ticker.Stop()

		lastTimestamp := since

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get new messages since last timestamp
				messages, err := r.store.GetMessagesAfter(ctx, sessionID, lastTimestamp)
				if err != nil {
					// Log error but continue streaming
					continue
				}

				// Convert and send new messages
				for _, msg := range messages {
					converted, convErr := r.convertToChatMessage(msg)
					if convErr != nil {
						continue
					}

					select {
					case messageChan <- converted:
						lastTimestamp = converted.CreatedAt
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return messageChan, nil
}
