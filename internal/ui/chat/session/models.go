// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/storage/db"
)

// Ensure our models implement the necessary interfaces
var (
	_ driver.Valuer = (*jsonMetadata)(nil)
	_ driver.Valuer = (*jsonToolCalls)(nil)
)

// jsonMetadata wraps map[string]interface{} for database storage
type jsonMetadata map[string]interface{}

// Value implements driver.Valuer for jsonMetadata
func (m jsonMetadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for jsonMetadata
func (m *jsonMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("cannot scan type %T into jsonMetadata", value)
	}
}

// jsonToolCalls wraps []ToolCall for database storage
type jsonToolCalls []ToolCall

// Value implements driver.Valuer for jsonToolCalls
func (tc jsonToolCalls) Value() (driver.Value, error) {
	if tc == nil {
		return nil, nil
	}
	return json.Marshal(tc)
}

// Scan implements sql.Scanner for jsonToolCalls
func (tc *jsonToolCalls) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, tc)
	case string:
		return json.Unmarshal([]byte(v), tc)
	default:
		return fmt.Errorf("cannot scan type %T into jsonToolCalls", value)
	}
}

// toDBSession converts a Session to the database model
func toDBSession(s *Session) db.ChatSession {
	var metadata []byte
	if s.Metadata != nil {
		metadata, _ = json.Marshal(s.Metadata)
	}

	return db.ChatSession{
		ID:         s.ID,
		Name:       s.Name,
		CampaignID: s.CampaignID,
		CreatedAt:  &s.CreatedAt,
		UpdatedAt:  &s.UpdatedAt,
		Metadata:   metadata,
	}
}

// fromDBSession converts a database model to Session
func fromDBSession(dbSession db.ChatSession) *Session {
	s := &Session{
		ID:         dbSession.ID,
		Name:       dbSession.Name,
		CampaignID: dbSession.CampaignID,
		CreatedAt:  time.Now(), // Default to now if nil
		UpdatedAt:  time.Now(), // Default to now if nil
	}

	// Set actual timestamps if available
	if dbSession.CreatedAt != nil {
		s.CreatedAt = *dbSession.CreatedAt
	}
	if dbSession.UpdatedAt != nil {
		s.UpdatedAt = *dbSession.UpdatedAt
	}

	if dbSession.Metadata != nil {
		var metadata map[string]interface{}
		switch v := dbSession.Metadata.(type) {
		case []byte:
			if err := json.Unmarshal(v, &metadata); err == nil {
				s.Metadata = metadata
			}
		case json.RawMessage:
			if err := json.Unmarshal(v, &metadata); err == nil {
				s.Metadata = metadata
			}
		}
	}

	return s
}

// toDBMessage converts a Message to the database model
func toDBMessage(m *Message) db.ChatMessage {
	var toolCalls []byte
	if m.ToolCalls != nil {
		toolCalls, _ = json.Marshal(m.ToolCalls)
	}

	var metadata []byte
	if m.Metadata != nil {
		metadata, _ = json.Marshal(m.Metadata)
	}

	return db.ChatMessage{
		ID:        m.ID,
		SessionID: m.SessionID,
		Role:      string(m.Role),
		Content:   m.Content,
		CreatedAt: &m.CreatedAt,
		ToolCalls: toolCalls,
		Metadata:  metadata,
	}
}

// fromDBMessage converts a database model to Message
func fromDBMessage(dbMessage db.ChatMessage) *Message {
	m := &Message{
		ID:        dbMessage.ID,
		SessionID: dbMessage.SessionID,
		Role:      MessageRole(dbMessage.Role),
		Content:   dbMessage.Content,
		CreatedAt: time.Now(), // Default to now if nil
	}

	if dbMessage.CreatedAt != nil {
		m.CreatedAt = *dbMessage.CreatedAt
	}

	if dbMessage.ToolCalls != nil {
		var toolCalls []ToolCall
		switch v := dbMessage.ToolCalls.(type) {
		case []byte:
			if err := json.Unmarshal(v, &toolCalls); err == nil {
				m.ToolCalls = toolCalls
			}
		case json.RawMessage:
			if err := json.Unmarshal(v, &toolCalls); err == nil {
				m.ToolCalls = toolCalls
			}
		}
	}

	if dbMessage.Metadata != nil {
		var metadata map[string]interface{}
		switch v := dbMessage.Metadata.(type) {
		case []byte:
			if err := json.Unmarshal(v, &metadata); err == nil {
				m.Metadata = metadata
			}
		case json.RawMessage:
			if err := json.Unmarshal(v, &metadata); err == nil {
				m.Metadata = metadata
			}
		}
	}

	return m
}

// toDBBookmark converts a Bookmark to the database model
func toDBBookmark(b *Bookmark) db.SessionBookmark {
	return db.SessionBookmark{
		ID:        b.ID,
		SessionID: b.SessionID,
		MessageID: b.MessageID,
		Name:      b.Name,
		CreatedAt: &b.CreatedAt,
	}
}

// fromDBBookmark converts a database model to Bookmark
func fromDBBookmark(dbBookmark db.SessionBookmark) *Bookmark {
	b := &Bookmark{
		ID:        dbBookmark.ID,
		SessionID: dbBookmark.SessionID,
		MessageID: dbBookmark.MessageID,
		Name:      dbBookmark.Name,
		CreatedAt: time.Now(), // Default to now if nil
	}

	if dbBookmark.CreatedAt != nil {
		b.CreatedAt = *dbBookmark.CreatedAt
	}

	return b
}

// sessionStats holds session statistics
type sessionStats struct {
	MessageCount  int64
	LastMessage   *time.Time
	BookmarkCount int64
}

// sessionFilter defines filtering options for sessions
type sessionFilter struct {
	CampaignID *string
	StartDate  *time.Time
	EndDate    *time.Time
	NameQuery  *string
}

// messageFilter defines filtering options for messages
type messageFilter struct {
	Role      *MessageRole
	StartDate *time.Time
	EndDate   *time.Time
	HasTools  *bool
}
