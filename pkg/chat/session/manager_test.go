package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore implements SessionStore for testing
type mockStore struct {
	sessions  map[string]*Session
	messages  map[string][]*Message
	bookmarks map[string][]*Bookmark
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions:  make(map[string]*Session),
		messages:  make(map[string][]*Message),
		bookmarks: make(map[string][]*Bookmark),
	}
}

func (m *mockStore) CreateSession(ctx context.Context, session *Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockStore) GetSession(ctx context.Context, id string) (*Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (m *mockStore) ListSessions(ctx context.Context, limit, offset int32) ([]*Session, error) {
	var sessions []*Session
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (m *mockStore) ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*Session, error) {
	var sessions []*Session
	for _, s := range m.sessions {
		if s.CampaignID != nil && *s.CampaignID == campaignID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (m *mockStore) UpdateSession(ctx context.Context, session *Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockStore) DeleteSession(ctx context.Context, id string) error {
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *mockStore) SearchSessions(ctx context.Context, query string, limit, offset int32) ([]*Session, error) {
	var results []*Session
	for _, s := range m.sessions {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(query)) {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *mockStore) CountSessions(ctx context.Context) (int64, error) {
	return int64(len(m.sessions)), nil
}

func (m *mockStore) SaveMessage(ctx context.Context, message *Message) error {
	m.messages[message.SessionID] = append(m.messages[message.SessionID], message)
	return nil
}

func (m *mockStore) GetMessage(ctx context.Context, id string) (*Message, error) {
	for _, messages := range m.messages {
		for _, msg := range messages {
			if msg.ID == id {
				return msg, nil
			}
		}
	}
	return nil, fmt.Errorf("message not found")
}

func (m *mockStore) GetMessages(ctx context.Context, sessionID string) ([]*Message, error) {
	return m.messages[sessionID], nil
}

func (m *mockStore) GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*Message, error) {
	messages := m.messages[sessionID]
	start := int(offset)
	end := start + int(limit)
	if start >= len(messages) {
		return []*Message{}, nil
	}
	if end > len(messages) {
		end = len(messages)
	}
	return messages[start:end], nil
}

func (m *mockStore) GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*Message, error) {
	var results []*Message
	for _, msg := range m.messages[sessionID] {
		if msg.CreatedAt.After(after) {
			results = append(results, msg)
		}
	}
	return results, nil
}

func (m *mockStore) CountMessages(ctx context.Context, sessionID string) (int64, error) {
	return int64(len(m.messages[sessionID])), nil
}

func (m *mockStore) DeleteMessage(ctx context.Context, id string) error {
	for sessionID, messages := range m.messages {
		for i, msg := range messages {
			if msg.ID == id {
				// Create a new slice to avoid issues with slice reuse
				newMessages := make([]*Message, 0, len(messages)-1)
				newMessages = append(newMessages, messages[:i]...)
				newMessages = append(newMessages, messages[i+1:]...)
				m.messages[sessionID] = newMessages
				return nil
			}
		}
	}
	return nil
}

func (m *mockStore) SearchMessages(ctx context.Context, query string, limit, offset int32) ([]*MessageSearchResult, error) {
	var results []*MessageSearchResult
	for sessionID, messages := range m.messages {
		session := m.sessions[sessionID]
		for _, msg := range messages {
			if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(query)) {
				results = append(results, &MessageSearchResult{
					Message:     msg,
					SessionName: session.Name,
					CampaignID:  session.CampaignID,
				})
			}
		}
	}
	return results, nil
}

func (m *mockStore) CreateBookmark(ctx context.Context, bookmark *Bookmark) error {
	m.bookmarks[bookmark.SessionID] = append(m.bookmarks[bookmark.SessionID], bookmark)
	return nil
}

func (m *mockStore) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	for _, bookmarks := range m.bookmarks {
		for _, b := range bookmarks {
			if b.ID == id {
				return b, nil
			}
		}
	}
	return nil, fmt.Errorf("bookmark not found")
}

func (m *mockStore) GetBookmarks(ctx context.Context, sessionID string) ([]*BookmarkWithDetails, error) {
	var results []*BookmarkWithDetails
	for _, b := range m.bookmarks[sessionID] {
		// Find the message
		for _, msg := range m.messages[sessionID] {
			if msg.ID == b.MessageID {
				results = append(results, &BookmarkWithDetails{
					Bookmark:       b,
					MessageContent: msg.Content,
					MessageRole:    msg.Role,
				})
				break
			}
		}
	}
	return results, nil
}

func (m *mockStore) DeleteBookmark(ctx context.Context, id string) error {
	for sessionID, bookmarks := range m.bookmarks {
		for i, b := range bookmarks {
			if b.ID == id {
				m.bookmarks[sessionID] = append(bookmarks[:i], bookmarks[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (m *mockStore) GetBookmarksByMessage(ctx context.Context, messageID string) ([]*Bookmark, error) {
	var results []*Bookmark
	for _, bookmarks := range m.bookmarks {
		for _, b := range bookmarks {
			if b.MessageID == messageID {
				results = append(results, b)
			}
		}
	}
	return results, nil
}

// Tests

func TestManager_NewSession(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	campaignID := "test-campaign"
	session, err := manager.NewSession("Test Session", &campaignID)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "Test Session", session.Name)
	assert.Equal(t, &campaignID, session.CampaignID)
	assert.NotNil(t, session.Metadata)
	assert.Equal(t, "1.0", session.Metadata["version"])
	assert.Equal(t, "guild-chat", session.Metadata["created_by"])
}

func TestManager_LoadSession(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create a session directly in store
	session := &Session{
		ID:   "test-id",
		Name: "Test Session",
	}
	store.sessions[session.ID] = session

	// Load it through manager
	loaded, err := manager.LoadSession(session.ID)
	
	assert.NoError(t, err)
	assert.Equal(t, session.ID, loaded.ID)
	assert.Equal(t, session.Name, loaded.Name)
}

func TestManager_SaveSession(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	session := &Session{
		ID:   "test-id",
		Name: "Original Name",
	}
	store.sessions[session.ID] = session

	// Update and save
	session.Name = "Updated Name"
	err := manager.SaveSession(session)
	
	assert.NoError(t, err)
	assert.False(t, session.UpdatedAt.IsZero())
	
	// Verify in store
	stored := store.sessions[session.ID]
	assert.Equal(t, "Updated Name", stored.Name)
}

func TestManager_ForkSession(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create source session with messages
	source := &Session{
		ID:   "source-id",
		Name: "Source Session",
	}
	store.sessions[source.ID] = source

	// Add messages to source
	messages := []*Message{
		{ID: "msg-1", SessionID: source.ID, Role: RoleUser, Content: "Hello"},
		{ID: "msg-2", SessionID: source.ID, Role: RoleAssistant, Content: "Hi there"},
	}
	store.messages[source.ID] = messages

	// Fork the session
	forked, err := manager.ForkSession(source.ID, "Forked Session")
	
	assert.NoError(t, err)
	assert.NotEqual(t, source.ID, forked.ID)
	assert.Equal(t, "Forked Session", forked.Name)
	assert.Equal(t, source.ID, forked.Metadata["forked_from"])
	
	// Verify messages were copied
	forkedMessages := store.messages[forked.ID]
	assert.Len(t, forkedMessages, 2)
	assert.Equal(t, "Hello", forkedMessages[0].Content)
	assert.Equal(t, "Hi there", forkedMessages[1].Content)
}

func TestManager_AppendMessage(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	sessionID := "test-session"
	
	toolCalls := []ToolCall{
		{
			ID:   "tool-1",
			Type: "function",
			Function: ToolFunction{
				Name:      "test_tool",
				Arguments: map[string]interface{}{"key": "value"},
			},
		},
	}

	message, err := manager.AppendMessage(sessionID, RoleUser, "Test message", toolCalls)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, message.ID)
	assert.Equal(t, sessionID, message.SessionID)
	assert.Equal(t, RoleUser, message.Role)
	assert.Equal(t, "Test message", message.Content)
	assert.Len(t, message.ToolCalls, 1)
	
	// Verify in store
	stored := store.messages[sessionID]
	assert.Len(t, stored, 1)
	assert.Equal(t, message.ID, stored[0].ID)
}

func TestManager_StreamMessage(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	sessionID := "test-session"
	
	// Create stream
	stream, err := manager.StreamMessage(sessionID, RoleAssistant)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	// Write content
	err = stream.Write("Hello ")
	assert.NoError(t, err)
	err = stream.Write("world!")
	assert.NoError(t, err)

	// Set tool calls
	toolCalls := []ToolCall{
		{ID: "tool-1", Type: "function"},
	}
	err = stream.SetToolCalls(toolCalls)
	assert.NoError(t, err)

	// Close stream
	message, err := stream.Close()
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", message.Content)
	assert.Equal(t, RoleAssistant, message.Role)
	assert.Len(t, message.ToolCalls, 1)

	// Verify message was saved
	stored := store.messages[sessionID]
	assert.Len(t, stored, 1)
	assert.Equal(t, message.ID, stored[0].ID)

	// Verify can't write after close
	err = stream.Write("more")
	assert.Error(t, err)
}

func TestManager_GetContext(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	sessionID := "test-session"
	
	// Add multiple messages
	for i := 0; i < 10; i++ {
		msg := &Message{
			ID:        fmt.Sprintf("msg-%d", i),
			SessionID: sessionID,
			Role:      RoleUser,
			Content:   fmt.Sprintf("Message %d", i),
		}
		store.messages[sessionID] = append(store.messages[sessionID], msg)
	}

	// Get last 5 messages
	context, err := manager.GetContext(sessionID, 5)
	assert.NoError(t, err)
	assert.Len(t, context, 5)
	assert.Equal(t, "Message 5", context[0].Content)
	assert.Equal(t, "Message 9", context[4].Content)

	// Get more than available
	context, err = manager.GetContext(sessionID, 20)
	assert.NoError(t, err)
	assert.Len(t, context, 10)
}

func TestManager_ClearContext(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	sessionID := "test-session"
	
	// Add messages
	for i := 0; i < 3; i++ {
		msg := &Message{
			ID:        fmt.Sprintf("msg-%d", i),
			SessionID: sessionID,
			Role:      RoleUser,
			Content:   fmt.Sprintf("Message %d", i),
		}
		store.messages[sessionID] = append(store.messages[sessionID], msg)
	}

	// Clear context
	err := manager.ClearContext(sessionID)
	assert.NoError(t, err)

	// Verify messages are gone
	messages := store.messages[sessionID]
	assert.Empty(t, messages)
}

func TestManager_ExportJSON(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create session
	session := &Session{
		ID:   "test-session",
		Name: "Export Test",
	}
	store.sessions[session.ID] = session

	// Add messages
	messages := []*Message{
		{ID: "msg-1", SessionID: session.ID, Role: RoleUser, Content: "Hello"},
		{ID: "msg-2", SessionID: session.ID, Role: RoleAssistant, Content: "Hi there"},
	}
	store.messages[session.ID] = messages

	// Export as JSON
	data, err := manager.ExportSession(session.ID, ExportFormatJSON)
	assert.NoError(t, err)

	// Parse exported JSON
	var export map[string]interface{}
	err = json.Unmarshal(data, &export)
	assert.NoError(t, err)
	
	assert.Equal(t, "2.0", export["version"])
	assert.NotNil(t, export["session"])
	assert.NotNil(t, export["messages"])
	assert.NotNil(t, export["exported_at"])
}

func TestManager_ExportMarkdown(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create session
	campaignID := "test-campaign"
	session := &Session{
		ID:         "test-session",
		Name:       "Markdown Export Test",
		CampaignID: &campaignID,
	}
	store.sessions[session.ID] = session

	// Add messages
	messages := []*Message{
		{
			ID:        "msg-1",
			SessionID: session.ID,
			Role:      RoleUser,
			Content:   "What's the weather?",
			CreatedAt: time.Now(),
		},
		{
			ID:        "msg-2",
			SessionID: session.ID,
			Role:      RoleAssistant,
			Content:   "I'll check the weather for you.",
			CreatedAt: time.Now(),
			ToolCalls: []ToolCall{
				{ID: "tool-1", Type: "function", Function: ToolFunction{Name: "get_weather"}},
			},
		},
	}
	store.messages[session.ID] = messages

	// Export as Markdown
	data, err := manager.ExportSession(session.ID, ExportFormatMarkdown)
	assert.NoError(t, err)

	markdown := string(data)
	assert.Contains(t, markdown, "# Markdown Export Test")
	assert.Contains(t, markdown, "**Campaign:** test-campaign")
	assert.Contains(t, markdown, "## 👤 User")
	assert.Contains(t, markdown, "What's the weather?")
	assert.Contains(t, markdown, "## 🤖 Assistant")
	assert.Contains(t, markdown, "**🔧 Tool Calls:**")
	assert.Contains(t, markdown, "- **get_weather**")
}

func TestManager_ExportHTML(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create session
	session := &Session{
		ID:   "test-session",
		Name: "HTML Export Test",
	}
	store.sessions[session.ID] = session

	// Add messages
	messages := []*Message{
		{
			ID:        "msg-1",
			SessionID: session.ID,
			Role:      RoleUser,
			Content:   "Line 1\nLine 2",
			CreatedAt: time.Now(),
		},
	}
	store.messages[session.ID] = messages

	// Export as HTML
	data, err := manager.ExportSession(session.ID, ExportFormatHTML)
	assert.NoError(t, err)

	html := string(data)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<title>HTML Export Test</title>")
	assert.Contains(t, html, `class="message user"`)
	assert.Contains(t, html, "Line 1<br>Line 2")
}

func TestManager_ImportJSON(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	// Create export data
	export := map[string]interface{}{
		"version": "1.0",
		"session": map[string]interface{}{
			"id":   "original-id",
			"name": "Original Session",
		},
		"messages": []map[string]interface{}{
			{
				"id":         "msg-1",
				"session_id": "original-id",
				"role":       "user",
				"content":    "Hello",
				"created_at": time.Now().Format(time.RFC3339),
			},
		},
	}

	data, err := json.Marshal(export)
	require.NoError(t, err)

	// Import
	imported, err := manager.ImportSession(data, ExportFormatJSON)
	assert.NoError(t, err)
	assert.NotEqual(t, "original-id", imported.ID)
	assert.Equal(t, "Original Session (imported)", imported.Name)
	assert.Equal(t, "original-id", imported.Metadata["imported_from"])

	// Verify messages were imported
	messages := store.messages[imported.ID]
	assert.Len(t, messages, 1)
	assert.Equal(t, "Hello", messages[0].Content)
}