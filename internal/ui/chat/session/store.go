// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/storage/db"
)

// sqliteStore implements SessionStore using SQLite
type sqliteStore struct {
	queries *db.Queries
}

// NewSQLiteStore creates a new SQLite-backed session store
func NewSQLiteStore(database *sql.DB) SessionStore {
	return &sqliteStore{
		queries: db.New(database),
	}
}

// CreateSession creates a new chat session
func (s *sqliteStore) CreateSession(ctx context.Context, session *Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = session.CreatedAt
	}

	var metadata []byte
	if session.Metadata != nil {
		var err error
		metadata, err = json.Marshal(session.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal session metadata")
		}
	}

	err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		ID:         session.ID,
		Name:       session.Name,
		CampaignID: session.CampaignID,
		Metadata:   json.RawMessage(metadata),
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create session")
	}

	return nil
}

// GetSession retrieves a session by ID
func (s *sqliteStore) GetSession(ctx context.Context, id string) (*Session, error) {
	dbSession, err := s.queries.GetSession(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("session not found: %s", id), nil)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get session")
	}

	return fromDBSession(dbSession), nil
}

// ListSessions returns a paginated list of sessions
func (s *sqliteStore) ListSessions(ctx context.Context, limit, offset int32) ([]*Session, error) {
	dbSessions, err := s.queries.ListSessions(ctx, db.ListSessionsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions")
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		sessions[i] = fromDBSession(dbSession)
	}

	return sessions, nil
}

// ListSessionsByCampaign returns all sessions for a campaign
func (s *sqliteStore) ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*Session, error) {
	dbSessions, err := s.queries.ListSessionsByCampaign(ctx, &campaignID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions by campaign")
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		sessions[i] = fromDBSession(dbSession)
	}

	return sessions, nil
}

// UpdateSession updates an existing session
func (s *sqliteStore) UpdateSession(ctx context.Context, session *Session) error {
	var metadata []byte
	if session.Metadata != nil {
		var err error
		metadata, err = json.Marshal(session.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal session metadata")
		}
	}

	err := s.queries.UpdateSession(ctx, db.UpdateSessionParams{
		Name:     session.Name,
		Metadata: json.RawMessage(metadata),
		ID:       session.ID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update session")
	}

	return nil
}

// DeleteSession deletes a session and all its messages
func (s *sqliteStore) DeleteSession(ctx context.Context, id string) error {
	err := s.queries.DeleteSession(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete session")
	}
	return nil
}

// SearchSessions searches for sessions by name
func (s *sqliteStore) SearchSessions(ctx context.Context, query string, limit, offset int32) ([]*Session, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)
	dbSessions, err := s.queries.SearchSessions(ctx, db.SearchSessionsParams{
		Name:   searchPattern,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to search sessions")
	}

	sessions := make([]*Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		sessions[i] = fromDBSession(dbSession)
	}

	return sessions, nil
}

// CountSessions returns the total number of sessions
func (s *sqliteStore) CountSessions(ctx context.Context) (int64, error) {
	count, err := s.queries.CountSessions(ctx)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to count sessions")
	}
	return count, nil
}

// SaveMessage saves a message to a session
func (s *sqliteStore) SaveMessage(ctx context.Context, message *Message) error {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	var toolCalls []byte
	if message.ToolCalls != nil {
		var err error
		toolCalls, err = json.Marshal(message.ToolCalls)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal tool calls")
		}
	}

	var metadata []byte
	if message.Metadata != nil {
		var err error
		metadata, err = json.Marshal(message.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal message metadata")
		}
	}

	err := s.queries.CreateMessage(ctx, db.CreateMessageParams{
		ID:        message.ID,
		SessionID: message.SessionID,
		Role:      string(message.Role),
		Content:   message.Content,
		ToolCalls: json.RawMessage(toolCalls),
		Metadata:  json.RawMessage(metadata),
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save message")
	}

	return nil
}

// GetMessage retrieves a single message by ID
func (s *sqliteStore) GetMessage(ctx context.Context, id string) (*Message, error) {
	dbMessage, err := s.queries.GetMessage(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("message not found: %s", id), nil)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get message")
	}

	return fromDBMessage(dbMessage), nil
}

// GetMessages retrieves all messages for a session
func (s *sqliteStore) GetMessages(ctx context.Context, sessionID string) ([]*Message, error) {
	dbMessages, err := s.queries.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get messages")
	}

	messages := make([]*Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i] = fromDBMessage(dbMessage)
	}

	return messages, nil
}

// GetMessagesPaginated retrieves messages with pagination
func (s *sqliteStore) GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*Message, error) {
	dbMessages, err := s.queries.GetMessagesPaginated(ctx, db.GetMessagesPaginatedParams{
		SessionID: sessionID,
		Limit:     int64(limit),
		Offset:    int64(offset),
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get paginated messages")
	}

	messages := make([]*Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i] = fromDBMessage(dbMessage)
	}

	return messages, nil
}

// GetMessagesAfter retrieves messages created after a specific time
func (s *sqliteStore) GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*Message, error) {
	dbMessages, err := s.queries.GetMessagesAfter(ctx, db.GetMessagesAfterParams{
		SessionID: sessionID,
		CreatedAt: &after,
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get messages after timestamp")
	}

	messages := make([]*Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i] = fromDBMessage(dbMessage)
	}

	return messages, nil
}

// CountMessages returns the number of messages in a session
func (s *sqliteStore) CountMessages(ctx context.Context, sessionID string) (int64, error) {
	count, err := s.queries.CountMessages(ctx, sessionID)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to count messages")
	}
	return count, nil
}

// DeleteMessage deletes a single message
func (s *sqliteStore) DeleteMessage(ctx context.Context, id string) error {
	err := s.queries.DeleteMessage(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete message")
	}
	return nil
}

// SearchMessages searches for messages containing a query string
func (s *sqliteStore) SearchMessages(ctx context.Context, query string, limit, offset int32) ([]*MessageSearchResult, error) {
	searchPattern := fmt.Sprintf("%%%s%%", query)
	rows, err := s.queries.SearchMessages(ctx, db.SearchMessagesParams{
		Content: searchPattern,
		Limit:   int64(limit),
		Offset:  int64(offset),
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to search messages")
	}

	results := make([]*MessageSearchResult, len(rows))
	for i, row := range rows {
		message := fromDBMessage(db.ChatMessage{
			ID:        row.ID,
			SessionID: row.SessionID,
			Role:      row.Role,
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
			ToolCalls: row.ToolCalls,
			Metadata:  row.Metadata,
		})

		results[i] = &MessageSearchResult{
			Message:     message,
			SessionName: row.SessionName,
			CampaignID:  row.CampaignID,
		}
	}

	return results, nil
}

// CreateBookmark creates a new bookmark
func (s *sqliteStore) CreateBookmark(ctx context.Context, bookmark *Bookmark) error {
	if bookmark.ID == "" {
		bookmark.ID = uuid.New().String()
	}

	if bookmark.CreatedAt.IsZero() {
		bookmark.CreatedAt = time.Now()
	}

	err := s.queries.CreateBookmark(ctx, db.CreateBookmarkParams{
		ID:        bookmark.ID,
		SessionID: bookmark.SessionID,
		MessageID: bookmark.MessageID,
		Name:      bookmark.Name,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create bookmark")
	}

	return nil
}

// GetBookmark retrieves a bookmark by ID
func (s *sqliteStore) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	dbBookmark, err := s.queries.GetBookmark(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("bookmark not found: %s", id), nil)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get bookmark")
	}

	return fromDBBookmark(dbBookmark), nil
}

// GetBookmarks retrieves all bookmarks for a session
func (s *sqliteStore) GetBookmarks(ctx context.Context, sessionID string) ([]*BookmarkWithDetails, error) {
	rows, err := s.queries.GetBookmarks(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get bookmarks")
	}

	bookmarks := make([]*BookmarkWithDetails, len(rows))
	for i, row := range rows {
		bookmark := fromDBBookmark(db.SessionBookmark{
			ID:        row.ID,
			SessionID: row.SessionID,
			MessageID: row.MessageID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
		})

		bookmarks[i] = &BookmarkWithDetails{
			Bookmark:       bookmark,
			MessageContent: row.MessageContent,
			MessageRole:    MessageRole(row.Role),
		}
	}

	return bookmarks, nil
}

// DeleteBookmark deletes a bookmark
func (s *sqliteStore) DeleteBookmark(ctx context.Context, id string) error {
	err := s.queries.DeleteBookmark(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete bookmark")
	}
	return nil
}

// GetBookmarksByMessage retrieves all bookmarks for a specific message
func (s *sqliteStore) GetBookmarksByMessage(ctx context.Context, messageID string) ([]*Bookmark, error) {
	dbBookmarks, err := s.queries.GetBookmarksByMessage(ctx, messageID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get bookmarks by message")
	}

	bookmarks := make([]*Bookmark, len(dbBookmarks))
	for i, dbBookmark := range dbBookmarks {
		bookmarks[i] = fromDBBookmark(dbBookmark)
	}

	return bookmarks, nil
}
