package session

import (
	"context"
	"encoding/json"
	"time"
)

// SessionStore defines the contract for persistent chat session storage
type SessionStore interface {
	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	ListSessions(ctx context.Context, limit, offset int32) ([]*Session, error)
	ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, id string) error
	SearchSessions(ctx context.Context, query string, limit, offset int32) ([]*Session, error)
	CountSessions(ctx context.Context) (int64, error)

	// Message operations
	SaveMessage(ctx context.Context, message *Message) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	GetMessages(ctx context.Context, sessionID string) ([]*Message, error)
	GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*Message, error)
	GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*Message, error)
	CountMessages(ctx context.Context, sessionID string) (int64, error)
	DeleteMessage(ctx context.Context, id string) error
	SearchMessages(ctx context.Context, query string, limit, offset int32) ([]*MessageSearchResult, error)

	// Bookmark operations
	CreateBookmark(ctx context.Context, bookmark *Bookmark) error
	GetBookmark(ctx context.Context, id string) (*Bookmark, error)
	GetBookmarks(ctx context.Context, sessionID string) ([]*BookmarkWithDetails, error)
	DeleteBookmark(ctx context.Context, id string) error
	GetBookmarksByMessage(ctx context.Context, messageID string) ([]*Bookmark, error)
}

// Session represents a chat conversation
type Session struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	CampaignID *string                `json:"campaign_id,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a single message in a chat session
type Message struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      MessageRole            `json:"role"`
	Content   string                 `json:"content"`
	CreatedAt time.Time              `json:"created_at"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageRole represents the role of the message sender
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// ToolCall represents a tool invocation in a message
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ToolFunction           `json:"function"`
	Result   *ToolResult            `json:"result,omitempty"`
}

// ToolFunction represents the function called by a tool
type ToolFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content string                 `json:"content"`
	Error   *string                `json:"error,omitempty"`
}

// Bookmark represents a marked message for easy retrieval
type Bookmark struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	MessageID string    `json:"message_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// BookmarkWithDetails includes bookmark with message content
type BookmarkWithDetails struct {
	*Bookmark
	MessageContent string      `json:"message_content"`
	MessageRole    MessageRole `json:"message_role"`
}

// MessageSearchResult includes message with session context
type MessageSearchResult struct {
	*Message
	SessionName string  `json:"session_name"`
	CampaignID  *string `json:"campaign_id,omitempty"`
}

// SessionManager provides high-level session operations
type SessionManager interface {
	// Session lifecycle
	NewSession(name string, campaignID *string) (*Session, error)
	LoadSession(id string) (*Session, error)
	SaveSession(session *Session) error
	ForkSession(sourceID string, newName string) (*Session, error)

	// Message handling
	AppendMessage(sessionID string, role MessageRole, content string, toolCalls []ToolCall) (*Message, error)
	StreamMessage(sessionID string, role MessageRole) (MessageStream, error)

	// Context management
	GetContext(sessionID string, messageCount int) ([]*Message, error)
	ClearContext(sessionID string) error

	// Export/Import
	ExportSession(sessionID string, format ExportFormat) ([]byte, error)
	ImportSession(data []byte, format ExportFormat) (*Session, error)
}

// MessageStream allows for streaming message content
type MessageStream interface {
	Write(chunk string) error
	SetToolCalls(toolCalls []ToolCall) error
	Close() (*Message, error)
}

// ExportFormat defines the format for session export
type ExportFormat string

const (
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatHTML     ExportFormat = "html"
)

// MarshalJSON implements custom JSON marshaling for ToolCall
func (tc ToolCall) MarshalJSON() ([]byte, error) {
	type Alias ToolCall
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(tc),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for ToolCall
func (tc *ToolCall) UnmarshalJSON(data []byte) error {
	type Alias ToolCall
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(tc),
	}
	return json.Unmarshal(data, &aux)
}