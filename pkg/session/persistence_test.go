// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/storage"
)

// mockSessionStore implements SessionStore for testing
type mockSessionStore struct {
	sessions map[string]*storage.ChatSession
	messages map[string][]*storage.ChatMessage
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*storage.ChatSession),
		messages: make(map[string][]*storage.ChatMessage),
	}
}

func (m *mockSessionStore) GetSession(ctx context.Context, sessionID string) (*storage.ChatSession, error) {
	if session, exists := m.sessions[sessionID]; exists {
		return session, nil
	}
	return nil, nil
}

func (m *mockSessionStore) UpsertSession(ctx context.Context, session *storage.ChatSession, stateData []byte) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) SaveMessage(ctx context.Context, sessionID string, message *storage.ChatMessage) error {
	m.messages[sessionID] = append(m.messages[sessionID], message)
	return nil
}

func (m *mockSessionStore) GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error) {
	return m.messages[sessionID], nil
}

func (m *mockSessionStore) ListSessions(ctx context.Context, options ListOptions) ([]*storage.ChatSession, error) {
	var sessions []*storage.ChatSession
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (m *mockSessionStore) Begin() (Transaction, error) {
	return &mockTransaction{store: m}, nil
}

// mockTransaction implements Transaction for testing
type mockTransaction struct {
	store *mockSessionStore
}

func (m *mockTransaction) UpsertSession(session *storage.ChatSession, stateData []byte) error {
	m.store.sessions[session.ID] = session
	return nil
}

func (m *mockTransaction) SaveMessage(sessionID string, message *storage.ChatMessage) error {
	m.store.messages[sessionID] = append(m.store.messages[sessionID], message)
	return nil
}

func (m *mockTransaction) Commit() error {
	return nil
}

func (m *mockTransaction) Rollback() error {
	return nil
}

func TestCraftSessionPersistence(t *testing.T) {
	ctx := context.Background()
	store := newMockSessionStore()
	manager := NewSessionManager(store)

	// Create a test session
	session := &Session{
		ID:             "test-session-1",
		UserID:         "test-user",
		CampaignID:     "test-campaign",
		StartTime:      time.Now().Add(-1 * time.Hour),
		LastActiveTime: time.Now(),
		State: SessionState{
			ActiveAgents:   make(map[string]AgentState),
			CurrentView:    "chat",
			ScrollPosition: 42,
			InputBuffer:    "test input",
			CommandHistory: []string{"help", "status"},
			Variables:      map[string]interface{}{"theme": "dark"},
			Status:         SessionStatusActive,
		},
		Messages: []Message{
			{
				ID:        "msg-1",
				Agent:     "elena",
				Content:   "Hello, I'm here to help!",
				Timestamp: time.Now().Add(-30 * time.Minute),
				Type:      MessageTypeAgent,
			},
			{
				ID:        "msg-2",
				Agent:     "user",
				Content:   "Can you help me with a task?",
				Timestamp: time.Now().Add(-25 * time.Minute),
				Type:      MessageTypeUser,
			},
		},
		Context: SessionContext{
			WorkingDirectory: "/tmp/test",
			GitBranch:        "main",
			OpenFiles:        []string{"test.go", "README.md"},
			RunningTasks:     []string{"task-1", "task-2"},
		},
	}

	// Test saving session
	err := manager.SaveSession(ctx, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Test loading session
	loadedSession, err := manager.LoadSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	// Verify basic properties
	if loadedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, loadedSession.ID)
	}

	if loadedSession.UserID != session.UserID {
		t.Errorf("Expected user ID %s, got %s", session.UserID, loadedSession.UserID)
	}

	// Test listing sessions
	sessions, err := manager.ListSessions(ctx, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestCraftSessionSerialization(t *testing.T) {
	serializer := NewSessionSerializer()

	state := SessionState{
		ActiveAgents: map[string]AgentState{
			"elena": {
				ID:           "elena",
				Name:         "Elena",
				Status:       "active",
				LastActivity: time.Now(),
				Context:      map[string]interface{}{"mode": "helpful"},
				TaskQueue:    []string{"task-1"},
			},
		},
		CurrentView:    "chat",
		ScrollPosition: 100,
		InputBuffer:    "test command",
		CommandHistory: []string{"help", "status", "list"},
		Variables: map[string]interface{}{
			"theme":     "dark",
			"debug":     true,
			"max_items": 50,
		},
		Status: SessionStatusActive,
	}

	// Test serialization
	data, err := serializer.SerializeState(state)
	if err != nil {
		t.Fatalf("Failed to serialize state: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data is empty")
	}

	// Test deserialization
	deserializedState, err := serializer.DeserializeState(data)
	if err != nil {
		t.Fatalf("Failed to deserialize state: %v", err)
	}

	// Verify deserialized state
	if deserializedState.CurrentView != state.CurrentView {
		t.Errorf("Expected current view %s, got %s", state.CurrentView, deserializedState.CurrentView)
	}

	if deserializedState.ScrollPosition != state.ScrollPosition {
		t.Errorf("Expected scroll position %d, got %d", state.ScrollPosition, deserializedState.ScrollPosition)
	}

	if len(deserializedState.ActiveAgents) != len(state.ActiveAgents) {
		t.Errorf("Expected %d active agents, got %d", len(state.ActiveAgents), len(deserializedState.ActiveAgents))
	}
}

func TestCraftEncryption(t *testing.T) {
	key := make([]byte, 32) // 256-bit key
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Test data
	plaintext := []byte("This is sensitive session data that should be encrypted")

	// Test encryption
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("Encrypted data is empty")
	}

	// Ensure ciphertext is different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Error("Ciphertext should be different from plaintext")
	}

	// Test decryption
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	// Verify decrypted data matches original
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data doesn't match original. Expected: %s, got: %s", string(plaintext), string(decrypted))
	}
}

func TestCraftCompression(t *testing.T) {
	compressor := NewCompressor()

	// Large data that should benefit from compression
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 10) // Repetitive pattern for good compression
	}

	// Test compression
	compressed := compressor.Compress(data)
	if len(compressed) == 0 {
		t.Error("Compressed data is empty")
	}

	// Compression should reduce size for repetitive data
	if len(compressed) >= len(data) {
		t.Error("Compression didn't reduce data size")
	}

	// Test compression detection
	if !compressor.IsCompressed(compressed) {
		t.Error("Failed to detect compressed data")
	}

	if compressor.IsCompressed(data) {
		t.Error("Incorrectly detected uncompressed data as compressed")
	}

	// Test decompression
	decompressed := compressor.Decompress(compressed)
	if len(decompressed) != len(data) {
		t.Errorf("Decompressed data size mismatch. Expected: %d, got: %d", len(data), len(decompressed))
	}

	// Verify data integrity
	for i, b := range decompressed {
		if b != data[i] {
			t.Errorf("Data integrity check failed at position %d. Expected: %d, got: %d", i, data[i], b)
			break
		}
	}
}

func TestCraftChangeBuffer(t *testing.T) {
	buffer := NewChangeBuffer()

	// Test initial state
	if buffer.HasChanges() {
		t.Error("New buffer should not have changes")
	}

	if buffer.ShouldFlush() {
		t.Error("New buffer should not need flushing")
	}

	// Add some changes
	for i := 0; i < 5; i++ {
		buffer.Add(map[string]interface{}{
			"type": "test_change",
			"id":   i,
		})
	}

	if !buffer.HasChanges() {
		t.Error("Buffer should have changes after adding")
	}

	// Add more changes to trigger flush threshold
	for i := 5; i < 12; i++ {
		buffer.Add(map[string]interface{}{
			"type": "test_change",
			"id":   i,
		})
	}

	if !buffer.ShouldFlush() {
		t.Error("Buffer should need flushing after reaching threshold")
	}

	// Test clearing
	buffer.Clear()
	if buffer.HasChanges() {
		t.Error("Buffer should not have changes after clearing")
	}
}

func TestCraftAutoSaveConfiguration(t *testing.T) {
	ctx := context.Background()
	store := newMockSessionStore()

	// Test with encryption
	key := make([]byte, 32)
	manager := NewSessionManager(store, WithEncryption(key))

	if manager.encryptor == nil {
		t.Error("Encryption should be enabled when key is provided")
	}

	// Test with custom auto-save interval
	interval := 5 * time.Second
	manager = NewSessionManager(store, WithAutoSaveInterval(interval))

	if manager.autoSaver.interval != interval {
		t.Errorf("Expected auto-save interval %v, got %v", interval, manager.autoSaver.interval)
	}

	// Test session creation and basic operations
	session := &Session{
		ID:             "test-session",
		UserID:         "test-user",
		CampaignID:     "test-campaign",
		StartTime:      time.Now(),
		LastActiveTime: time.Now(),
		State: SessionState{
			Status: SessionStatusActive,
		},
	}

	err := manager.SaveSession(ctx, session)
	if err != nil {
		t.Fatalf("Failed to save session with configuration: %v", err)
	}
}
