package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// manager implements SessionManager with high-level operations
type manager struct {
	store SessionStore
	mu    sync.RWMutex
	
	// Active message streams
	streams map[string]*messageStream
}

// NewManager creates a new session manager
func NewManager(store SessionStore) SessionManager {
	return &manager{
		store:   store,
		streams: make(map[string]*messageStream),
	}
}

// NewSession creates a new chat session
func (m *manager) NewSession(name string, campaignID *string) (*Session, error) {
	ctx := context.Background()
	
	session := &Session{
		ID:         uuid.New().String(),
		Name:       name,
		CampaignID: campaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"version": "1.0",
			"created_by": "guild-chat",
		},
	}

	if err := m.store.CreateSession(ctx, session); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create new session")
	}

	return session, nil
}

// LoadSession loads an existing session
func (m *manager) LoadSession(id string) (*Session, error) {
	ctx := context.Background()
	
	session, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load session")
	}

	return session, nil
}

// SaveSession updates an existing session
func (m *manager) SaveSession(session *Session) error {
	ctx := context.Background()
	
	session.UpdatedAt = time.Now()
	if err := m.store.UpdateSession(ctx, session); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save session")
	}

	return nil
}

// ForkSession creates a new session branching from an existing one
func (m *manager) ForkSession(sourceID string, newName string) (*Session, error) {
	ctx := context.Background()
	
	// Load source session
	source, err := m.store.GetSession(ctx, sourceID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load source session")
	}

	// Create new session
	forked := &Session{
		ID:         uuid.New().String(),
		Name:       newName,
		CampaignID: source.CampaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"created_by":  "guild-chat",
			"forked_from": sourceID,
			"forked_at":   time.Now().Format(time.RFC3339),
		},
	}

	if err := m.store.CreateSession(ctx, forked); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create forked session")
	}

	// Copy messages from source
	messages, err := m.store.GetMessages(ctx, sourceID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load source messages")
	}

	for _, msg := range messages {
		// Create new message with new ID
		forkedMsg := &Message{
			ID:        uuid.New().String(),
			SessionID: forked.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
			ToolCalls: msg.ToolCalls,
			Metadata:  msg.Metadata,
		}
		
		if err := m.store.SaveMessage(ctx, forkedMsg); err != nil {
			// Log error but continue copying
			continue
		}
	}

	return forked, nil
}

// AppendMessage adds a new message to a session
func (m *manager) AppendMessage(sessionID string, role MessageRole, content string, toolCalls []ToolCall) (*Message, error) {
	ctx := context.Background()
	
	message := &Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
		ToolCalls: toolCalls,
	}

	if err := m.store.SaveMessage(ctx, message); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to append message")
	}

	return message, nil
}

// StreamMessage creates a message stream for progressive content
func (m *manager) StreamMessage(sessionID string, role MessageRole) (MessageStream, error) {
	streamID := uuid.New().String()
	
	stream := &messageStream{
		id:        streamID,
		sessionID: sessionID,
		role:      role,
		content:   &bytes.Buffer{},
		manager:   m,
		createdAt: time.Now(),
	}

	m.mu.Lock()
	m.streams[streamID] = stream
	m.mu.Unlock()

	return stream, nil
}

// GetContext retrieves recent messages for context
func (m *manager) GetContext(sessionID string, messageCount int) ([]*Message, error) {
	ctx := context.Background()
	
	// Get all messages (they're already ordered by created_at ASC)
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get context messages")
	}

	// Return last N messages
	if len(messages) > messageCount {
		return messages[len(messages)-messageCount:], nil
	}
	
	return messages, nil
}

// ClearContext removes all messages from a session
func (m *manager) ClearContext(sessionID string) error {
	ctx := context.Background()
	
	// Get all messages
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get messages for clearing")
	}

	// Delete each message
	for _, msg := range messages {
		if err := m.store.DeleteMessage(ctx, msg.ID); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// ExportSession exports a session in the specified format
func (m *manager) ExportSession(sessionID string, format ExportFormat) ([]byte, error) {
	ctx := context.Background()
	
	// Load session
	session, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load session for export")
	}

	// Load messages
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load messages for export")
	}

	switch format {
	case ExportFormatJSON:
		return m.exportJSON(session, messages)
	case ExportFormatMarkdown:
		return m.exportMarkdown(session, messages)
	case ExportFormatHTML:
		return m.exportHTML(session, messages)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported export format: %s", format), nil)
	}
}

// ImportSession imports a session from exported data
func (m *manager) ImportSession(data []byte, format ExportFormat) (*Session, error) {
	switch format {
	case ExportFormatJSON:
		return m.importJSON(data)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported import format: %s", format), nil)
	}
}

// messageStream implements MessageStream for streaming content
type messageStream struct {
	id        string
	sessionID string
	role      MessageRole
	content   *bytes.Buffer
	toolCalls []ToolCall
	manager   *manager
	createdAt time.Time
	mu        sync.Mutex
	closed    bool
}

// Write appends content to the stream
func (s *messageStream) Write(chunk string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return gerror.New(gerror.ErrCodeInvalidInput, "stream is closed", nil)
	}

	s.content.WriteString(chunk)
	return nil
}

// SetToolCalls sets the tool calls for the message
func (s *messageStream) SetToolCalls(toolCalls []ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return gerror.New(gerror.ErrCodeInvalidInput, "stream is closed", nil)
	}

	s.toolCalls = toolCalls
	return nil
}

// Close finalizes the stream and saves the message
func (s *messageStream) Close() (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "stream already closed", nil)
	}

	s.closed = true

	// Remove from active streams
	s.manager.mu.Lock()
	delete(s.manager.streams, s.id)
	s.manager.mu.Unlock()

	// Create and save message
	message := &Message{
		ID:        uuid.New().String(),
		SessionID: s.sessionID,
		Role:      s.role,
		Content:   s.content.String(),
		CreatedAt: s.createdAt,
		ToolCalls: s.toolCalls,
	}

	ctx := context.Background()
	if err := s.manager.store.SaveMessage(ctx, message); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save streamed message")
	}

	return message, nil
}

// Export helpers

func (m *manager) exportJSON(session *Session, messages []*Message) ([]byte, error) {
	export := map[string]interface{}{
		"session":  session,
		"messages": messages,
		"exported_at": time.Now().Format(time.RFC3339),
		"version": "1.0",
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON export")
	}

	return data, nil
}

func (m *manager) exportMarkdown(session *Session, messages []*Message) ([]byte, error) {
	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "# Chat Session: %s\n\n", session.Name)
	fmt.Fprintf(&buf, "**Session ID:** %s  \n", session.ID)
	fmt.Fprintf(&buf, "**Created:** %s  \n", session.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&buf, "**Updated:** %s  \n\n", session.UpdatedAt.Format(time.RFC3339))
	
	if session.CampaignID != nil {
		fmt.Fprintf(&buf, "**Campaign:** %s  \n\n", *session.CampaignID)
	}

	fmt.Fprintf(&buf, "---\n\n")

	// Messages
	for _, msg := range messages {
		fmt.Fprintf(&buf, "## %s\n", strings.Title(string(msg.Role)))
		fmt.Fprintf(&buf, "*%s*\n\n", msg.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&buf, "%s\n\n", msg.Content)

		if len(msg.ToolCalls) > 0 {
			fmt.Fprintf(&buf, "**Tool Calls:**\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&buf, "- %s\n", tc.Function.Name)
			}
			fmt.Fprintf(&buf, "\n")
		}

		fmt.Fprintf(&buf, "---\n\n")
	}

	return buf.Bytes(), nil
}

func (m *manager) exportHTML(session *Session, messages []*Message) ([]byte, error) {
	var buf bytes.Buffer

	// HTML header
	fmt.Fprintf(&buf, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Chat Session: %s</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .message { margin-bottom: 20px; padding: 15px; border-radius: 5px; }
        .user { background-color: #e3f2fd; }
        .assistant { background-color: #f5f5f5; }
        .system { background-color: #fff3e0; }
        .tool { background-color: #e8f5e9; }
        .role { font-weight: bold; color: #666; }
        .timestamp { font-size: 0.9em; color: #999; }
        .content { margin-top: 10px; white-space: pre-wrap; }
        .tool-calls { margin-top: 10px; font-style: italic; }
    </style>
</head>
<body>
`, session.Name)

	// Session info
	fmt.Fprintf(&buf, "<h1>Chat Session: %s</h1>\n", session.Name)
	fmt.Fprintf(&buf, "<p><strong>Session ID:</strong> %s<br>\n", session.ID)
	fmt.Fprintf(&buf, "<strong>Created:</strong> %s<br>\n", session.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&buf, "<strong>Updated:</strong> %s</p>\n", session.UpdatedAt.Format(time.RFC3339))
	
	if session.CampaignID != nil {
		fmt.Fprintf(&buf, "<p><strong>Campaign:</strong> %s</p>\n", *session.CampaignID)
	}

	fmt.Fprintf(&buf, "<hr>\n")

	// Messages
	for _, msg := range messages {
		fmt.Fprintf(&buf, `<div class="message %s">`, msg.Role)
		fmt.Fprintf(&buf, `<div class="role">%s</div>`, strings.Title(string(msg.Role)))
		fmt.Fprintf(&buf, `<div class="timestamp">%s</div>`, msg.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&buf, `<div class="content">%s</div>`, strings.ReplaceAll(msg.Content, "\n", "<br>"))
		
		if len(msg.ToolCalls) > 0 {
			fmt.Fprintf(&buf, `<div class="tool-calls">Tool Calls: `)
			for i, tc := range msg.ToolCalls {
				if i > 0 {
					fmt.Fprintf(&buf, ", ")
				}
				fmt.Fprintf(&buf, "%s", tc.Function.Name)
			}
			fmt.Fprintf(&buf, `</div>`)
		}
		
		fmt.Fprintf(&buf, "</div>\n")
	}

	// HTML footer
	fmt.Fprintf(&buf, "</body>\n</html>\n")

	return buf.Bytes(), nil
}

// Import helpers

func (m *manager) importJSON(data []byte) (*Session, error) {
	var export struct {
		Session  *Session   `json:"session"`
		Messages []*Message `json:"messages"`
		Version  string     `json:"version"`
	}

	if err := json.Unmarshal(data, &export); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to unmarshal JSON import")
	}

	if export.Version != "1.0" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported import version: %s", export.Version), nil)
	}

	ctx := context.Background()

	// Create new session with new ID
	newSession := &Session{
		ID:         uuid.New().String(),
		Name:       fmt.Sprintf("%s (imported)", export.Session.Name),
		CampaignID: export.Session.CampaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"imported_from": export.Session.ID,
			"imported_at":   time.Now().Format(time.RFC3339),
			"original_created": export.Session.CreatedAt.Format(time.RFC3339),
		},
	}

	if err := m.store.CreateSession(ctx, newSession); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create imported session")
	}

	// Import messages
	for _, msg := range export.Messages {
		newMsg := &Message{
			ID:        uuid.New().String(),
			SessionID: newSession.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
			ToolCalls: msg.ToolCalls,
			Metadata:  msg.Metadata,
		}

		if err := m.store.SaveMessage(ctx, newMsg); err != nil {
			// Log error but continue importing
			continue
		}
	}

	return newSession, nil
}