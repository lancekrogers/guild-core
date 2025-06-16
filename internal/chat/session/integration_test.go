// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FullSessionLifecycle(t *testing.T) {
	// Setup database
	db := setupTestDB(t)
	defer db.Close()

	// Create store and manager
	store := NewSQLiteStore(db)
	manager := NewManager(store)

	// Test 1: Create new session
	// Don't use a campaign ID since we haven't created the campaign
	session, err := manager.NewSession("Integration Test Session", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)

	// Test 2: Add messages
	msg1, err := manager.AppendMessage(session.ID, RoleUser, "What's the capital of France?", nil)
	require.NoError(t, err)

	// Test 3: Stream a response
	stream, err := manager.StreamMessage(session.ID, RoleAssistant)
	require.NoError(t, err)

	err = stream.Write("The capital of France is ")
	assert.NoError(t, err)
	err = stream.Write("Paris.")
	assert.NoError(t, err)

	msg2, err := stream.Close()
	require.NoError(t, err)
	assert.Equal(t, "The capital of France is Paris.", msg2.Content)

	// Test 4: Get context
	messages, err := manager.GetContext(session.ID, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, msg1.ID, messages[0].ID)
	assert.Equal(t, msg2.ID, messages[1].ID)

	// Test 5: Create bookmark
	msgToBookmark := messages[1] // Bookmark the assistant's response
	bookmark := &Bookmark{
		SessionID: session.ID,
		MessageID: msgToBookmark.ID,
		Name:      "Capital of France",
	}
	err = store.CreateBookmark(context.Background(), bookmark)
	require.NoError(t, err)

	// Test 6: Fork session
	forked, err := manager.ForkSession(session.ID, "Forked Geography Session")
	require.NoError(t, err)
	assert.NotEqual(t, session.ID, forked.ID)

	// Verify forked messages
	forkedMessages, err := store.GetMessages(context.Background(), forked.ID)
	require.NoError(t, err)
	assert.Len(t, forkedMessages, 2)

	// Test 7: Export session
	jsonData, err := manager.ExportSession(session.ID, ExportFormatJSON)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	markdownData, err := manager.ExportSession(session.ID, ExportFormatMarkdown)
	require.NoError(t, err)
	assert.Contains(t, string(markdownData), "Integration Test Session")
	assert.Contains(t, string(markdownData), "What's the capital of France?")
	assert.Contains(t, string(markdownData), "Paris")

	// Test 8: Search functionality
	searchResults, err := store.SearchMessages(context.Background(), "France", 10, 0)
	require.NoError(t, err)
	assert.Len(t, searchResults, 4) // Two messages in each session (original + forked)

	// Test 9: Session statistics
	count, err := store.CountMessages(context.Background(), session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	sessionCount, err := store.CountSessions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), sessionCount) // Original + forked
}

func TestIntegration_MessageWithToolCalls(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSQLiteStore(db)
	manager := NewManager(store)

	// Create session
	session, err := manager.NewSession("Tool Test Session", nil)
	require.NoError(t, err)

	// Add message with tool calls
	toolCalls := []ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: ToolFunction{
				Name: "get_weather",
				Arguments: map[string]interface{}{
					"location": "Paris",
					"units":    "celsius",
				},
			},
		},
		{
			ID:   "call-2",
			Type: "function",
			Function: ToolFunction{
				Name: "get_time",
				Arguments: map[string]interface{}{
					"timezone": "Europe/Paris",
				},
			},
		},
	}

	msg, err := manager.AppendMessage(session.ID, RoleAssistant, "I'll check the weather and time for you.", toolCalls)
	require.NoError(t, err)

	// Retrieve and verify
	ctx := context.Background()
	retrieved, err := store.GetMessage(ctx, msg.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.ToolCalls, 2)
	assert.Equal(t, "get_weather", retrieved.ToolCalls[0].Function.Name)
	assert.Equal(t, "Paris", retrieved.ToolCalls[0].Function.Arguments["location"])
}

func TestIntegration_SessionPersistenceAcrossRestart(t *testing.T) {
	// Simulate application restart by creating new manager instances
	db := setupTestDB(t)
	defer db.Close()

	// First "application run"
	store1 := NewSQLiteStore(db)
	manager1 := NewManager(store1)

	session, err := manager1.NewSession("Persistent Session", nil)
	require.NoError(t, err)
	sessionID := session.ID

	_, err = manager1.AppendMessage(sessionID, RoleUser, "Remember this message", nil)
	require.NoError(t, err)

	// Second "application run" - new instances but same DB
	store2 := NewSQLiteStore(db)
	manager2 := NewManager(store2)

	// Load previous session
	loaded, err := manager2.LoadSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, "Persistent Session", loaded.Name)

	// Get previous messages
	messages, err := manager2.GetContext(sessionID, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "Remember this message", messages[0].Content)

	// Continue conversation
	_, err = manager2.AppendMessage(sessionID, RoleAssistant, "I remember your message", nil)
	require.NoError(t, err)

	// Verify full conversation
	allMessages, err := store2.GetMessages(context.Background(), sessionID)
	require.NoError(t, err)
	assert.Len(t, allMessages, 2)
}

func TestIntegration_ConcurrentMessageStreams(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSQLiteStore(db)
	manager := NewManager(store)

	// Create two sessions
	session1, err := manager.NewSession("Session 1", nil)
	require.NoError(t, err)
	session2, err := manager.NewSession("Session 2", nil)
	require.NoError(t, err)

	// Start concurrent streams
	stream1, err := manager.StreamMessage(session1.ID, RoleAssistant)
	require.NoError(t, err)
	stream2, err := manager.StreamMessage(session2.ID, RoleAssistant)
	require.NoError(t, err)

	// Write to both streams
	err = stream1.Write("Stream 1 content")
	assert.NoError(t, err)
	err = stream2.Write("Stream 2 content")
	assert.NoError(t, err)

	// Close streams
	msg1, err := stream1.Close()
	require.NoError(t, err)
	msg2, err := stream2.Close()
	require.NoError(t, err)

	// Verify messages went to correct sessions
	assert.Equal(t, session1.ID, msg1.SessionID)
	assert.Equal(t, session2.ID, msg2.SessionID)
	assert.Equal(t, "Stream 1 content", msg1.Content)
	assert.Equal(t, "Stream 2 content", msg2.Content)
}

func TestIntegration_BookmarkWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSQLiteStore(db)
	ctx := context.Background()

	// Create session with messages
	session := &Session{ID: "bookmark-test", Name: "Bookmark Test"}
	err := store.CreateSession(ctx, session)
	require.NoError(t, err)

	// Add multiple messages
	importantMsg := &Message{
		ID:        "important-msg",
		SessionID: session.ID,
		Role:      RoleAssistant,
		Content:   "This is important information about the project deadline.",
	}
	err = store.SaveMessage(ctx, importantMsg)
	require.NoError(t, err)

	regularMsg := &Message{
		ID:        "regular-msg",
		SessionID: session.ID,
		Role:      RoleUser,
		Content:   "Thanks for the info.",
	}
	err = store.SaveMessage(ctx, regularMsg)
	require.NoError(t, err)

	// Bookmark the important message
	bookmark := &Bookmark{
		SessionID: session.ID,
		MessageID: importantMsg.ID,
		Name:      "Project Deadline Info",
	}
	err = store.CreateBookmark(ctx, bookmark)
	require.NoError(t, err)

	// Get bookmarks with details
	bookmarks, err := store.GetBookmarks(ctx, session.ID)
	require.NoError(t, err)
	assert.Len(t, bookmarks, 1)
	assert.Equal(t, "Project Deadline Info", bookmarks[0].Name)
	assert.Contains(t, bookmarks[0].MessageContent, "deadline")
	assert.Equal(t, RoleAssistant, bookmarks[0].MessageRole)
}

func TestIntegration_SessionUpdateTrigger(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSQLiteStore(db)
	ctx := context.Background()

	// Create session
	session := &Session{ID: "trigger-test", Name: "Trigger Test"}
	err := store.CreateSession(ctx, session)
	require.NoError(t, err)

	// Get initial timestamp
	initial, err := store.GetSession(ctx, session.ID)
	require.NoError(t, err)
	initialUpdated := initial.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(1 * time.Second)

	// Add a message (should trigger update)
	msg := &Message{
		ID:        "trigger-msg",
		SessionID: session.ID,
		Role:      RoleUser,
		Content:   "This should update the session timestamp",
	}
	err = store.SaveMessage(ctx, msg)
	require.NoError(t, err)

	// Verify session was updated
	updated, err := store.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(initialUpdated))
}
