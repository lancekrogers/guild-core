# Chat Session Store

The session package provides persistent storage and management for chat conversations in the Guild Framework. It implements the session persistence layer described in Sprint 3, enabling chat history, session management, and rich export capabilities.

## Architecture

The package follows a layered architecture:

```
┌─────────────────────────────────┐
│      SessionManager             │  High-level operations
├─────────────────────────────────┤
│      SessionStore               │  Storage interface
├─────────────────────────────────┤
│      SQLite Implementation      │  Database layer
└─────────────────────────────────┘
```

## Core Components

### SessionStore Interface
Defines the contract for persistent storage operations:
- Session CRUD operations
- Message storage and retrieval
- Bookmark management
- Search functionality

### SessionManager Interface
Provides high-level session operations:
- Session lifecycle management
- Message streaming
- Context management
- Export/Import functionality

### Data Models
- **Session**: Represents a chat conversation
- **Message**: Individual messages with role, content, and tool calls
- **Bookmark**: Marked messages for easy retrieval
- **ToolCall**: Function invocations within messages

## Usage

### Creating a Session Store

```go
// Create SQLite database connection
db, err := sql.Open("sqlite3", "path/to/guild.db")
if err != nil {
    log.Fatal(err)
}

// Create store and manager
store := session.NewSQLiteStore(db)
manager := session.NewManager(store)
```

### Basic Session Operations

```go
// Create new session
session, err := manager.NewSession("Project Discussion", &campaignID)

// Append messages
msg, err := manager.AppendMessage(session.ID, session.RoleUser, "Hello!", nil)

// Stream assistant response
stream, err := manager.StreamMessage(session.ID, session.RoleAssistant)
stream.Write("I'm processing")
stream.Write(" your request...")
response, err := stream.Close()

// Get context (last N messages)
context, err := manager.GetContext(session.ID, 10)

// Fork session
forked, err := manager.ForkSession(session.ID, "Alternative Branch")
```

### Working with Tool Calls

```go
toolCalls := []session.ToolCall{
    {
        ID:   "call-1",
        Type: "function",
        Function: session.ToolFunction{
            Name: "get_weather",
            Arguments: map[string]interface{}{
                "location": "Paris",
                "units":    "celsius",
            },
        },
    },
}

msg, err := manager.AppendMessage(
    session.ID, 
    session.RoleAssistant, 
    "I'll check the weather for you.",
    toolCalls,
)
```

### Bookmarking Messages

```go
// Create bookmark
bookmark := &session.Bookmark{
    SessionID: session.ID,
    MessageID: messageID,
    Name:      "Important Decision",
}
err := store.CreateBookmark(ctx, bookmark)

// Get bookmarks with message details
bookmarks, err := store.GetBookmarks(ctx, session.ID)
for _, b := range bookmarks {
    fmt.Printf("Bookmark: %s\nContent: %s\n", b.Name, b.MessageContent)
}
```

### Search Operations

```go
// Search sessions by name
sessions, err := store.SearchSessions(ctx, "project", 10, 0)

// Search messages across all sessions
results, err := store.SearchMessages(ctx, "deadline", 20, 0)
for _, r := range results {
    fmt.Printf("Session: %s\nMessage: %s\n", r.SessionName, r.Content)
}
```

### Export/Import

```go
// Export session as JSON
jsonData, err := manager.ExportSession(session.ID, session.ExportFormatJSON)

// Export as Markdown
markdownData, err := manager.ExportSession(session.ID, session.ExportFormatMarkdown)

// Export as HTML
htmlData, err := manager.ExportSession(session.ID, session.ExportFormatHTML)

// Import from JSON
imported, err := manager.ImportSession(jsonData, session.ExportFormatJSON)
```

## Database Schema

The package uses the following SQLite schema (defined in migrations):

```sql
-- Chat sessions
CREATE TABLE chat_sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    campaign_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
);

-- Chat messages
CREATE TABLE chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tool_calls JSON,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- Session bookmarks
CREATE TABLE session_bookmarks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE
);
```

## Features

- **Persistent Storage**: All chat sessions and messages are stored in SQLite
- **Message Streaming**: Support for progressive message content
- **Session Forking**: Create branches from existing conversations
- **Rich Export**: Export sessions as JSON, Markdown, or HTML
- **Full-Text Search**: Search across sessions and messages
- **Bookmarking**: Mark important messages for easy retrieval
- **Tool Call Support**: Store and retrieve function invocations
- **Metadata Support**: Flexible metadata storage for sessions and messages
- **Cascade Deletes**: Automatic cleanup of related data
- **Auto-Timestamps**: Automatic session update tracking

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/chat/session/...

# Run with coverage
go test -cover ./pkg/chat/session/...

# Run integration tests only
go test -run Integration ./pkg/chat/session/...
```

## Performance Considerations

- Messages are indexed by session_id and created_at for fast retrieval
- Sessions are indexed by campaign_id and updated_at
- Bookmarks are indexed for quick lookup
- The updated_at trigger ensures efficient session sorting
- Consider pagination for large message histories

## Error Handling

All errors use the gerror package with appropriate error codes:
- `ErrCodeNotFound`: Session/message/bookmark not found
- `ErrCodeStorage`: Database operation failed
- `ErrCodeInternal`: Internal errors (e.g., JSON marshaling)
- `ErrCodeInvalidInput`: Invalid parameters or closed streams