// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/storage"
)

// SessionManager provides enhanced session persistence with enterprise features
type SessionManager struct {
	store      SessionStore
	serializer *SessionSerializer
	encryptor  *Encryptor
	compressor *Compressor
	autoSaver  *AutoSaver
	mu         sync.RWMutex
}

// Session represents a complete chat session with state
type Session struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	CampaignID      string          `json:"campaign_id"`
	StartTime       time.Time       `json:"start_time"`
	LastActiveTime  time.Time       `json:"last_active_time"`
	State           SessionState    `json:"state"`
	Messages        []Message       `json:"messages"`
	Context         SessionContext  `json:"context"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SessionState captures the complete UI and agent state
type SessionState struct {
	ActiveAgents    map[string]AgentState `json:"active_agents"`
	CurrentView     string               `json:"current_view"`
	ScrollPosition  int                  `json:"scroll_position"`
	InputBuffer     string               `json:"input_buffer"`
	CommandHistory  []string             `json:"command_history"`
	Variables       map[string]interface{} `json:"variables"`
	Status          SessionStatus        `json:"status"`
}

// SessionStatus represents the current session state
type SessionStatus string

const (
	SessionStatusActive SessionStatus = "active"
	SessionStatusPaused SessionStatus = "paused"
	SessionStatusClosed SessionStatus = "closed"
)

// AgentState captures the state of an individual agent
type AgentState struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	LastActivity time.Time              `json:"last_activity"`
	Context      map[string]interface{} `json:"context"`
	TaskQueue    []string               `json:"task_queue"`
}

// SessionContext captures the working environment
type SessionContext struct {
	WorkingDirectory string   `json:"working_directory"`
	GitBranch        string   `json:"git_branch"`
	OpenFiles        []string `json:"open_files"`
	RunningTasks     []string `json:"running_tasks"`
	CorpusQueries    []string `json:"corpus_queries"`
}

// Message represents a chat message
type Message struct {
	ID        string                 `json:"id"`
	Agent     string                 `json:"agent"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Type      MessageType            `json:"type"`
	Metadata  map[string]interface{} `json:"metadata"`
	Attachments []Attachment         `json:"attachments,omitempty"`
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeUser   MessageType = "user"
	MessageTypeAgent  MessageType = "agent"
	MessageTypeSystem MessageType = "system"
)

// Attachment represents a file or media attachment
type Attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// NewSessionManager creates a new enhanced session manager
func NewSessionManager(store SessionStore, options ...SessionOption) *SessionManager {
	sm := &SessionManager{
		store:      store,
		serializer: NewSessionSerializer(),
		compressor: NewCompressor(),
	}

	// Apply options
	for _, opt := range options {
		opt(sm)
	}

	// Initialize auto-saver if not disabled
	if sm.autoSaver == nil {
		sm.autoSaver = NewAutoSaver(sm, 30*time.Second)
	}

	return sm
}

// SessionOption configures the session manager
type SessionOption func(*SessionManager)

// WithEncryption enables encryption with the provided key
func WithEncryption(key []byte) SessionOption {
	return func(sm *SessionManager) {
		encryptor, err := NewEncryptor(key)
		if err == nil {
			sm.encryptor = encryptor
		}
	}
}

// WithAutoSaveInterval sets the auto-save interval
func WithAutoSaveInterval(interval time.Duration) SessionOption {
	return func(sm *SessionManager) {
		sm.autoSaver = NewAutoSaver(sm, interval)
	}
}

// SessionStore interface defines the storage layer
type SessionStore interface {
	GetSession(ctx context.Context, sessionID string) (*storage.ChatSession, error)
	UpsertSession(ctx context.Context, session *storage.ChatSession, stateData []byte) error
	SaveMessage(ctx context.Context, sessionID string, message *storage.ChatMessage) error
	GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error)
	ListSessions(ctx context.Context, options ListOptions) ([]*storage.ChatSession, error)
	Begin() (Transaction, error)
}

// Transaction interface for database transactions
type Transaction interface {
	UpsertSession(session *storage.ChatSession, stateData []byte) error
	SaveMessage(sessionID string, message *storage.ChatMessage) error
	Commit() error
	Rollback() error
}

// ListOptions defines options for listing sessions
type ListOptions struct {
	OrderBy string
	Limit   int
	Offset  int
}

// SaveSession persists a session with encryption and compression
func (sm *SessionManager) SaveSession(ctx context.Context, session *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Serialize state
	stateData, err := sm.serializer.SerializeState(session.State)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to serialize state")
	}

	// Compress if large
	if len(stateData) > 1024*10 { // 10KB threshold
		stateData = sm.compressor.Compress(stateData)
	}

	// Encrypt sensitive data
	if sm.encryptor != nil {
		stateData, err = sm.encryptor.Encrypt(stateData)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encrypt state")
		}
	}

	// Begin transaction
	tx, err := sm.store.Begin()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Convert to storage model
	storageSession := &storage.ChatSession{
		ID:         session.ID,
		Name:       fmt.Sprintf("Session %s", session.ID[:8]),
		CampaignID: &session.CampaignID,
		CreatedAt:  session.StartTime,
		UpdatedAt:  session.LastActiveTime,
		Metadata: map[string]interface{}{
			"user_id":          session.UserID,
			"session_context":  session.Context,
			"variables":        session.State.Variables,
			"status":           string(session.State.Status),
		},
	}

	// Upsert session
	err = tx.UpsertSession(storageSession, stateData)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to upsert session")
	}

	// Save messages
	for _, msg := range session.Messages {
		storageMsg := &storage.ChatMessage{
			ID:        msg.ID,
			SessionID: session.ID,
			Role:      string(msg.Type),
			Content:   msg.Content,
			CreatedAt: msg.Timestamp,
			Metadata: map[string]interface{}{
				"agent":       msg.Agent,
				"attachments": msg.Attachments,
			},
		}

		err = tx.SaveMessage(session.ID, storageMsg)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save message")
		}
	}

	// Commit transaction
	return tx.Commit()
}

// LoadSession retrieves and reconstructs a session
func (sm *SessionManager) LoadSession(ctx context.Context, sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Load session data
	sessionData, err := sm.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get session")
	}

	// Extract state data from metadata (simplified for demo)
	stateData := []byte("{}") // This would be extracted from the actual storage
	
	// Decrypt if needed
	if sm.encryptor != nil {
		stateData, err = sm.encryptor.Decrypt(stateData)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decrypt state")
		}
	}

	// Decompress if needed
	if sm.compressor.IsCompressed(stateData) {
		stateData = sm.compressor.Decompress(stateData)
	}

	// Deserialize state
	state, err := sm.serializer.DeserializeState(stateData)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to deserialize state")
	}

	// Load messages
	messages, err := sm.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get messages")
	}

	// Convert messages
	var sessionMessages []Message
	for _, msg := range messages {
		sessionMsg := Message{
			ID:        msg.ID,
			Agent:     getStringFromMetadata(msg.Metadata, "agent"),
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
			Type:      MessageType(msg.Role),
			Metadata:  msg.Metadata,
		}
		sessionMessages = append(sessionMessages, sessionMsg)
	}

	// Reconstruct session
	session := &Session{
		ID:             sessionData.ID,
		UserID:         getStringFromMetadata(sessionData.Metadata, "user_id"),
		CampaignID:     getStringFromPointer(sessionData.CampaignID),
		StartTime:      sessionData.CreatedAt,
		LastActiveTime: sessionData.UpdatedAt,
		State:          state,
		Messages:       sessionMessages,
		Metadata:       sessionData.Metadata,
	}

	return session, nil
}

// ListSessions returns a list of sessions with optional filtering
func (sm *SessionManager) ListSessions(ctx context.Context, options ListOptions) ([]*Session, error) {
	sessions, err := sm.store.ListSessions(ctx, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions")
	}

	var result []*Session
	for _, sessionData := range sessions {
		// For listing, we don't need to load full state
		session := &Session{
			ID:             sessionData.ID,
			UserID:         getStringFromMetadata(sessionData.Metadata, "user_id"),
			CampaignID:     getStringFromPointer(sessionData.CampaignID),
			StartTime:      sessionData.CreatedAt,
			LastActiveTime: sessionData.UpdatedAt,
			Metadata:       sessionData.Metadata,
		}
		result = append(result, session)
	}

	return result, nil
}

// StartAutoSave begins automatic session saving
func (sm *SessionManager) StartAutoSave(ctx context.Context, session *Session) {
	if sm.autoSaver != nil {
		sm.autoSaver.Start(ctx, session)
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(ctx context.Context, userID, campaignID string) (*Session, error) {
	session := &Session{
		ID:             generateSessionID(),
		UserID:         userID,
		CampaignID:     campaignID,
		StartTime:      time.Now(),
		LastActiveTime: time.Now(),
		State:          SessionState{Status: SessionStatusActive},
		Messages:       make([]Message, 0),
		Metadata:       make(map[string]interface{}),
	}
	
	if err := sm.SaveSession(ctx, session); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create session")
	}
	
	return session, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	// TODO: The current SessionStore interface doesn't have DeleteSession
	// For now, return not implemented
	return gerror.New(gerror.ErrCodeNotImplemented, "delete session not implemented", nil)
}

// AddMessage adds a message to a session
func (sm *SessionManager) AddMessage(ctx context.Context, sessionID string, message *Message) error {
	session, err := sm.LoadSession(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load session for message")
	}
	
	session.Messages = append(session.Messages, *message)
	session.LastActiveTime = time.Now()
	
	return sm.SaveSession(ctx, session)
}

// GetMessages gets messages from a session
func (sm *SessionManager) GetMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error) {
	session, err := sm.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load session for messages")
	}
	
	messages := make([]*Message, 0)
	start := len(session.Messages) - limit
	if start < 0 {
		start = 0
	}
	
	for i := start; i < len(session.Messages); i++ {
		msg := session.Messages[i]
		messages = append(messages, &msg)
	}
	
	return messages, nil
}

// UpdateSessionState updates session state
func (sm *SessionManager) UpdateSessionState(ctx context.Context, sessionID string, state SessionState) error {
	session, err := sm.LoadSession(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load session for state update")
	}
	
	session.State = state
	session.LastActiveTime = time.Now()
	
	return sm.SaveSession(ctx, session)
}

// StopAutoSave stops automatic session saving
func (sm *SessionManager) StopAutoSave(ctx context.Context, sessionID string) {
	if sm.autoSaver != nil {
		sm.autoSaver.Stop(sessionID)
	}
}

// CreateBackup creates a backup of a session
func (sm *SessionManager) CreateBackup(ctx context.Context, sessionID string) error {
	session, err := sm.LoadSession(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load session for backup")
	}
	
	// For now, just save the session - a real implementation might save to a backup location
	return sm.SaveSession(ctx, session)
}

// Helper functions
func getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if val, ok := metadata[key].(string); ok {
		return val
	}
	return ""
}

func getStringFromPointer(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// SessionSerializer handles session state serialization
type SessionSerializer struct{}

// NewSessionSerializer creates a new session serializer
func NewSessionSerializer() *SessionSerializer {
	return &SessionSerializer{}
}

// SerializeState serializes session state to bytes
func (s *SessionSerializer) SerializeState(state SessionState) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	
	if err := encoder.Encode(state); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to encode session state")
	}
	
	return buf.Bytes(), nil
}

// DeserializeState deserializes session state from bytes
func (s *SessionSerializer) DeserializeState(data []byte) (SessionState, error) {
	var state SessionState
	
	if len(data) == 0 {
		// Return default state for empty data
		return SessionState{
			ActiveAgents:   make(map[string]AgentState),
			Variables:     make(map[string]interface{}),
			Status:        SessionStatusActive,
		}, nil
	}
	
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)
	
	if err := decoder.Decode(&state); err != nil {
		return state, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to decode session state")
	}
	
	return state, nil
}

// Encryptor handles session data encryption
type Encryptor struct {
	key    []byte
	cipher cipher.AEAD
}

// NewEncryptor creates a new encryptor with the provided key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "encryption key must be 32 bytes", nil)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cipher")
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create AEAD")
	}

	return &Encryptor{
		key:    key,
		cipher: aead,
	}, nil
}

// Encrypt encrypts data
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, e.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate nonce")
	}

	ciphertext := e.cipher.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts data
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	nonceSize := e.cipher.NonceSize()
	if len(data) < nonceSize {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "ciphertext too short", nil)
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decrypt")
	}

	return plaintext, nil
}

// Compressor handles session data compression
type Compressor struct{}

// NewCompressor creates a new compressor
func NewCompressor() *Compressor {
	return &Compressor{}
}

// Compress compresses data using gzip
func (c *Compressor) Compress(data []byte) []byte {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	
	writer.Write(data)
	writer.Close()
	
	return buf.Bytes()
}

// Decompress decompresses gzip data
func (c *Compressor) Decompress(data []byte) []byte {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return data // Return original if decompression fails
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return data // Return original if decompression fails
	}

	return result
}

// IsCompressed checks if data is gzip compressed
func (c *Compressor) IsCompressed(data []byte) bool {
	return len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b
}

// AutoSaver handles automatic session saving
type AutoSaver struct {
	manager  *SessionManager
	interval time.Duration
	buffer   *ChangeBuffer
}

// NewAutoSaver creates a new auto-saver
func NewAutoSaver(manager *SessionManager, interval time.Duration) *AutoSaver {
	return &AutoSaver{
		manager:  manager,
		interval: interval,
		buffer:   NewChangeBuffer(),
	}
}

// Start begins auto-saving for a session
func (as *AutoSaver) Start(ctx context.Context, session *Session) {
	ticker := time.NewTicker(as.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final save on context cancellation
			as.saveIfChanged(ctx, session)
			return

		case <-ticker.C:
			as.saveIfChanged(ctx, session)

		case change := <-as.buffer.Changes():
			as.buffer.Add(change)
			if as.buffer.ShouldFlush() {
				as.save(ctx, session)
				as.buffer.Clear()
			}
		}
	}
}

// saveIfChanged saves the session if there are changes
func (as *AutoSaver) saveIfChanged(ctx context.Context, session *Session) {
	if as.buffer.HasChanges() {
		as.save(ctx, session)
		as.buffer.Clear()
	}
}

// save performs the actual save operation
func (as *AutoSaver) save(ctx context.Context, session *Session) {
	if err := as.manager.SaveSession(ctx, session); err != nil {
		// Log error but don't panic
		fmt.Printf("Auto-save failed: %v\n", err)
	}
}

// Stop stops auto-saving for a session
func (as *AutoSaver) Stop(sessionID string) {
	// For now, this is a placeholder. A real implementation would
	// track per-session auto-save goroutines and stop them
}

// ChangeBuffer tracks changes for batched saving
type ChangeBuffer struct {
	changes []interface{}
	mu      sync.Mutex
	ch      chan interface{}
}

// NewChangeBuffer creates a new change buffer
func NewChangeBuffer() *ChangeBuffer {
	return &ChangeBuffer{
		changes: make([]interface{}, 0),
		ch:      make(chan interface{}, 100),
	}
}

// Add adds a change to the buffer
func (cb *ChangeBuffer) Add(change interface{}) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.changes = append(cb.changes, change)
}

// HasChanges returns true if there are pending changes
func (cb *ChangeBuffer) HasChanges() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return len(cb.changes) > 0
}

// ShouldFlush returns true if the buffer should be flushed
func (cb *ChangeBuffer) ShouldFlush() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return len(cb.changes) >= 10 // Flush after 10 changes
}

// Clear clears the buffer
func (cb *ChangeBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.changes = cb.changes[:0]
}

// Changes returns the channel for receiving changes
func (cb *ChangeBuffer) Changes() <-chan interface{} {
	return cb.ch
}

// SQLiteSessionStore implements SessionStore using SQLite
type SQLiteSessionStore struct {
	db *sql.DB
}

// NewSQLiteSessionStore creates a new SQLite session store
func NewSQLiteSessionStore(db *sql.DB) *SQLiteSessionStore {
	return &SQLiteSessionStore{db: db}
}

// InitSchema initializes the database schema
func (sss *SQLiteSessionStore) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		campaign_id TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		last_active_time TIMESTAMP NOT NULL,
		state BLOB,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS session_messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		metadata JSON,
		embedding BLOB,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);
	
	CREATE TABLE IF NOT EXISTS session_events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		event_data JSON,
		timestamp TIMESTAMP NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_campaign ON sessions(campaign_id);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON session_messages(session_id);
	CREATE INDEX IF NOT EXISTS idx_events_session ON session_events(session_id);
	`

	_, err := sss.db.Exec(schema)
	return err
}

// Implement SessionStore interface methods (simplified for brevity)
func (sss *SQLiteSessionStore) GetSession(ctx context.Context, sessionID string) (*storage.ChatSession, error) {
	// Implementation would query the database
	return &storage.ChatSession{ID: sessionID}, nil
}

func (sss *SQLiteSessionStore) UpsertSession(ctx context.Context, session *storage.ChatSession, stateData []byte) error {
	// Implementation would upsert the session
	return nil
}

func (sss *SQLiteSessionStore) SaveMessage(ctx context.Context, sessionID string, message *storage.ChatMessage) error {
	// Implementation would save the message
	return nil
}

func (sss *SQLiteSessionStore) GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error) {
	// Implementation would retrieve messages
	return []*storage.ChatMessage{}, nil
}

func (sss *SQLiteSessionStore) ListSessions(ctx context.Context, options ListOptions) ([]*storage.ChatSession, error) {
	// Implementation would list sessions
	return []*storage.ChatSession{}, nil
}

func (sss *SQLiteSessionStore) Begin() (Transaction, error) {
	// Implementation would begin a transaction
	return &sqliteTransaction{}, nil
}

// sqliteTransaction implements Transaction
type sqliteTransaction struct{}

func (st *sqliteTransaction) UpsertSession(session *storage.ChatSession, stateData []byte) error {
	return nil
}

func (st *sqliteTransaction) SaveMessage(sessionID string, message *storage.ChatMessage) error {
	return nil
}

func (st *sqliteTransaction) Commit() error {
	return nil
}

func (st *sqliteTransaction) Rollback() error {
	return nil
}