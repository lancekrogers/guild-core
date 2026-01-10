// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// EventStore provides persistence and replay capabilities for events
type EventStore interface {
	// Store persists an event to the store
	Store(ctx context.Context, event Event) error

	// Replay replays events from a given timestamp
	Replay(ctx context.Context, from time.Time, handler func(Event) error) error

	// ReplayByType replays events of a specific type
	ReplayByType(ctx context.Context, eventType string, from time.Time, handler func(Event) error) error

	// GetSnapshot retrieves the latest snapshot for an aggregate
	GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error)

	// SaveSnapshot saves a snapshot for an aggregate
	SaveSnapshot(ctx context.Context, snapshot *Snapshot) error

	// Archive moves old events to archive storage
	Archive(ctx context.Context, before time.Time) error

	// Close closes the event store
	Close() error
}

// Snapshot represents a point-in-time state of an aggregate
type Snapshot struct {
	AggregateID string
	Version     int64
	State       json.RawMessage
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// SQLEventStore implements EventStore using SQLite
type SQLEventStore struct {
	db              *sql.DB
	storeStmt       *sql.Stmt
	replayStmt      *sql.Stmt
	replayTypeStmt  *sql.Stmt
	snapshotGetStmt *sql.Stmt
	snapshotSetStmt *sql.Stmt
	archiveStmt     *sql.Stmt
	mu              sync.RWMutex
	closed          bool
}

// NewSQLEventStore creates a new SQL-based event store
func NewSQLEventStore(db *sql.DB) (*SQLEventStore, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "database is required", nil)
	}

	// Create tables if they don't exist
	if err := createEventStoreTables(db); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create event store tables")
	}

	store := &SQLEventStore{
		db: db,
	}

	// Prepare statements
	if err := store.prepareStatements(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare statements")
	}

	return store, nil
}

// Store persists an event to the store
func (s *SQLEventStore) Store(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	// Serialize event data
	data, err := json.Marshal(event)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal event").
			WithDetails("event_id", event.GetID())
	}

	// Extract correlation and parent IDs from metadata
	var correlationID, parentID string
	if metadata := event.GetMetadata(); metadata != nil {
		if corrID, ok := metadata["correlation_id"].(string); ok {
			correlationID = corrID
		}
		if parID, ok := metadata["parent_id"].(string); ok {
			parentID = parID
		}
	}

	// Store event
	_, err = s.storeStmt.ExecContext(ctx,
		event.GetID(),
		event.GetType(),
		event.GetTimestamp(),
		event.GetSource(),
		data,
		correlationID,
		parentID,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store event").
			WithDetails("event_id", event.GetID())
	}

	return nil
}

// Replay replays events from a given timestamp
func (s *SQLEventStore) Replay(ctx context.Context, from time.Time, handler func(Event) error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	rows, err := s.replayStmt.QueryContext(ctx, from)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query events").
			WithDetails("from", from)
	}
	defer rows.Close()

	return s.processEventRows(ctx, rows, handler)
}

// ReplayByType replays events of a specific type
func (s *SQLEventStore) ReplayByType(ctx context.Context, eventType string, from time.Time, handler func(Event) error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	rows, err := s.replayTypeStmt.QueryContext(ctx, eventType, from)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query events by type").
			WithDetails("type", eventType).
			WithDetails("from", from)
	}
	defer rows.Close()

	return s.processEventRows(ctx, rows, handler)
}

// GetSnapshot retrieves the latest snapshot for an aggregate
func (s *SQLEventStore) GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	var snapshot Snapshot
	var metadata []byte

	err := s.snapshotGetStmt.QueryRowContext(ctx, aggregateID).Scan(
		&snapshot.AggregateID,
		&snapshot.Version,
		&snapshot.State,
		&snapshot.Timestamp,
		&metadata,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get snapshot").
			WithDetails("aggregate_id", aggregateID)
	}

	if metadata != nil {
		if err := json.Unmarshal(metadata, &snapshot.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal snapshot metadata")
		}
	}

	return &snapshot, nil
}

// SaveSnapshot saves a snapshot for an aggregate
func (s *SQLEventStore) SaveSnapshot(ctx context.Context, snapshot *Snapshot) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	metadata, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal snapshot metadata")
	}

	_, err = s.snapshotSetStmt.ExecContext(ctx,
		snapshot.AggregateID,
		snapshot.Version,
		snapshot.State,
		snapshot.Timestamp,
		metadata,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save snapshot").
			WithDetails("aggregate_id", snapshot.AggregateID)
	}

	return nil
}

// Archive moves old events to archive storage
func (s *SQLEventStore) Archive(ctx context.Context, before time.Time) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "event store is closed", nil)
	}
	s.mu.RUnlock()

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Copy events to archive
	copyQuery := `
		INSERT INTO archived_events 
		SELECT * FROM events WHERE timestamp < ?
	`
	if _, err := tx.ExecContext(ctx, copyQuery, before); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to copy events to archive").
			WithDetails("before", before)
	}

	// Delete archived events
	if _, err := tx.ExecContext(ctx, "DELETE FROM events WHERE timestamp < ?", before); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete archived events").
			WithDetails("before", before)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to commit archive transaction")
	}

	return nil
}

// Close closes the event store
func (s *SQLEventStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Close prepared statements
	if s.storeStmt != nil {
		s.storeStmt.Close()
	}
	if s.replayStmt != nil {
		s.replayStmt.Close()
	}
	if s.replayTypeStmt != nil {
		s.replayTypeStmt.Close()
	}
	if s.snapshotGetStmt != nil {
		s.snapshotGetStmt.Close()
	}
	if s.snapshotSetStmt != nil {
		s.snapshotSetStmt.Close()
	}
	if s.archiveStmt != nil {
		s.archiveStmt.Close()
	}

	return nil
}

// prepareStatements prepares SQL statements for reuse
func (s *SQLEventStore) prepareStatements() error {
	var err error

	// Store event statement
	s.storeStmt, err = s.db.Prepare(`
		INSERT INTO events (id, type, timestamp, source, data, correlation_id, parent_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare store statement")
	}

	// Replay events statement
	s.replayStmt, err = s.db.Prepare(`
		SELECT id, type, timestamp, source, data, correlation_id, parent_id
		FROM events
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare replay statement")
	}

	// Replay by type statement
	s.replayTypeStmt, err = s.db.Prepare(`
		SELECT id, type, timestamp, source, data, correlation_id, parent_id
		FROM events
		WHERE type = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare replay by type statement")
	}

	// Get snapshot statement
	s.snapshotGetStmt, err = s.db.Prepare(`
		SELECT aggregate_id, version, state, timestamp, metadata
		FROM snapshots
		WHERE aggregate_id = ?
		ORDER BY version DESC
		LIMIT 1
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare get snapshot statement")
	}

	// Save snapshot statement
	s.snapshotSetStmt, err = s.db.Prepare(`
		INSERT OR REPLACE INTO snapshots (aggregate_id, version, state, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare save snapshot statement")
	}

	return nil
}

// processEventRows processes rows from event queries
func (s *SQLEventStore) processEventRows(ctx context.Context, rows *sql.Rows, handler func(Event) error) error {
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during replay")
		}

		var (
			id            string
			eventType     string
			timestamp     time.Time
			source        string
			data          []byte
			correlationID sql.NullString
			parentID      sql.NullString
		)

		if err := rows.Scan(&id, &eventType, &timestamp, &source, &data, &correlationID, &parentID); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan event row")
		}

		// Deserialize event based on type
		event, err := deserializeEvent(eventType, data)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to deserialize event").
				WithDetails("event_id", id).
				WithDetails("event_type", eventType)
		}

		// Call handler
		if err := handler(event); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "handler failed for event").
				WithDetails("event_id", id)
		}
	}

	if err := rows.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating event rows")
	}

	return nil
}

// createEventStoreTables creates the necessary database tables
func createEventStoreTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			source TEXT NOT NULL,
			data BLOB NOT NULL,
			correlation_id TEXT,
			parent_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(type)`,
		`CREATE INDEX IF NOT EXISTS idx_events_correlation ON events(correlation_id)`,
		`CREATE TABLE IF NOT EXISTS archived_events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			source TEXT NOT NULL,
			data BLOB NOT NULL,
			correlation_id TEXT,
			parent_id TEXT,
			created_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			aggregate_id TEXT PRIMARY KEY,
			version INTEGER NOT NULL,
			state BLOB NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			metadata BLOB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create table").
				WithDetails("query", query)
		}
	}

	return nil
}

// deserializeEvent deserializes an event from JSON based on its type
func deserializeEvent(eventType string, data []byte) (Event, error) {
	// This is a simplified version - in production, you'd have a registry
	// of event types and their corresponding struct types
	var event map[string]interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal event data")
	}

	// For now, return a generic event wrapper
	// In a full implementation, you'd deserialize to specific event types
	return &genericEvent{
		eventType: eventType,
		data:      event,
	}, nil
}

// genericEvent is a temporary wrapper for deserialized events
type genericEvent struct {
	eventType string
	data      map[string]interface{}
}

func (e *genericEvent) GetID() string {
	if id, ok := e.data["id"].(string); ok {
		return id
	}
	return ""
}

func (e *genericEvent) GetType() string {
	return e.eventType
}

func (e *genericEvent) GetTimestamp() time.Time {
	if ts, ok := e.data["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (e *genericEvent) GetSource() string {
	if source, ok := e.data["source"].(string); ok {
		return source
	}
	return ""
}

func (e *genericEvent) GetData() map[string]interface{} {
	if data, ok := e.data["data"].(map[string]interface{}); ok {
		return data
	}
	return make(map[string]interface{})
}

func (e *genericEvent) GetTarget() string {
	if target, ok := e.data["target"].(string); ok {
		return target
	}
	return ""
}

func (e *genericEvent) GetMetadata() map[string]interface{} {
	if metadata, ok := e.data["metadata"].(map[string]interface{}); ok {
		return metadata
	}
	return nil
}

func (e *genericEvent) GetCorrelationID() string {
	if id, ok := e.data["correlation_id"].(string); ok {
		return id
	}
	return ""
}

func (e *genericEvent) GetParentID() string {
	if id, ok := e.data["parent_id"].(string); ok {
		return id
	}
	return ""
}

func (e *genericEvent) Clone() Event {
	// Deep copy the data
	dataCopy := make(map[string]interface{})
	for k, v := range e.data {
		dataCopy[k] = v
	}
	return &genericEvent{
		eventType: e.eventType,
		data:      dataCopy,
	}
}
