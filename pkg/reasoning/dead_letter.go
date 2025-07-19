// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// DeadLetterEntry represents a failed reasoning extraction
type DeadLetterEntry struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Content      string                 `json:"content"`
	Error        string                 `json:"error"`
	ErrorCode    string                 `json:"error_code"`
	Attempts     int                    `json:"attempts"`
	FirstAttempt time.Time              `json:"first_attempt"`
	LastAttempt  time.Time              `json:"last_attempt"`
	Metadata     map[string]interface{} `json:"metadata"`
	Processed    bool                   `json:"processed"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty"`
}

// DeadLetterQueue stores failed reasoning extractions for later processing
type DeadLetterQueue struct {
	mu       sync.RWMutex
	db       *sql.DB
	maxSize  int
	entries  []DeadLetterEntry // In-memory cache
	onChange func(entry DeadLetterEntry)
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(db *sql.DB, maxSize int) *DeadLetterQueue {
	if maxSize <= 0 {
		maxSize = 1000
	}

	return &DeadLetterQueue{
		db:      db,
		maxSize: maxSize,
		entries: make([]DeadLetterEntry, 0),
	}
}

// Add adds a failed extraction to the queue
func (dlq *DeadLetterQueue) Add(ctx context.Context, agentID, content string, err error, attempts int, metadata map[string]interface{}) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Extract error details
	errorMsg := err.Error()
	errorCode := string(gerror.ErrCodeInternal)
	if gerr, ok := err.(*gerror.GuildError); ok {
		errorCode = string(gerr.Code)
	}

	entry := DeadLetterEntry{
		ID:           uuid.New().String(),
		AgentID:      agentID,
		Content:      content,
		Error:        errorMsg,
		ErrorCode:    errorCode,
		Attempts:     attempts,
		FirstAttempt: time.Now(),
		LastAttempt:  time.Now(),
		Metadata:     metadata,
		Processed:    false,
	}

	// Store in database
	if err := dlq.persistEntry(ctx, entry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to persist dead letter entry").
			WithComponent("dead_letter_queue")
	}

	// Add to in-memory cache
	dlq.entries = append(dlq.entries, entry)

	// Enforce size limit (FIFO)
	if len(dlq.entries) > dlq.maxSize {
		dlq.entries = dlq.entries[1:]
	}

	// Notify listener if configured
	if dlq.onChange != nil {
		go dlq.onChange(entry)
	}

	return nil
}

// Get retrieves unprocessed entries
func (dlq *DeadLetterQueue) Get(ctx context.Context, limit int) ([]DeadLetterEntry, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	// Query from database
	query := `
		SELECT id, agent_id, content, error, error_code, attempts,
		       first_attempt, last_attempt, metadata
		FROM reasoning_failures
		WHERE processed = false
		ORDER BY last_attempt DESC
		LIMIT $1
	`

	rows, err := dlq.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to query dead letter entries").
			WithComponent("dead_letter_queue")
	}
	defer rows.Close()

	var entries []DeadLetterEntry
	for rows.Next() {
		var entry DeadLetterEntry
		var metadataJSON string

		err := rows.Scan(
			&entry.ID,
			&entry.AgentID,
			&entry.Content,
			&entry.Error,
			&entry.ErrorCode,
			&entry.Attempts,
			&entry.FirstAttempt,
			&entry.LastAttempt,
			&metadataJSON,
		)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan dead letter entry").
				WithComponent("dead_letter_queue")
		}

		// Unmarshal metadata
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &entry.Metadata); err != nil {
				// Log error but don't fail
				entry.Metadata = map[string]interface{}{
					"error": "failed to unmarshal metadata",
				}
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "error iterating dead letter entries").
			WithComponent("dead_letter_queue")
	}

	return entries, nil
}

// MarkProcessed marks an entry as processed
func (dlq *DeadLetterQueue) MarkProcessed(ctx context.Context, id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	now := time.Now()
	query := `
		UPDATE reasoning_failures
		SET processed = true, processed_at = $1
		WHERE id = $2
	`

	result, err := dlq.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to mark entry as processed").
			WithComponent("dead_letter_queue").
			WithDetails("entry_id", id)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get rows affected").
			WithComponent("dead_letter_queue")
	}

	if rows == 0 {
		return gerror.New(gerror.ErrCodeNotFound, "entry not found", nil).
			WithComponent("dead_letter_queue").
			WithDetails("entry_id", id)
	}

	// Update in-memory cache
	for i, entry := range dlq.entries {
		if entry.ID == id {
			dlq.entries[i].Processed = true
			dlq.entries[i].ProcessedAt = &now
			break
		}
	}

	return nil
}

// Reprocess attempts to reprocess an entry
func (dlq *DeadLetterQueue) Reprocess(ctx context.Context, id string, reprocessFn func(DeadLetterEntry) error) error {
	// Get the entry
	var entry DeadLetterEntry
	query := `
		SELECT id, agent_id, content, error, error_code, attempts,
		       first_attempt, last_attempt, metadata
		FROM reasoning_failures
		WHERE id = $1
	`

	var metadataJSON string
	err := dlq.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.AgentID,
		&entry.Content,
		&entry.Error,
		&entry.ErrorCode,
		&entry.Attempts,
		&entry.FirstAttempt,
		&entry.LastAttempt,
		&metadataJSON,
	)
	if err == sql.ErrNoRows {
		return gerror.New(gerror.ErrCodeNotFound, "entry not found", nil).
			WithComponent("dead_letter_queue").
			WithDetails("entry_id", id)
	}
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get entry for reprocessing").
			WithComponent("dead_letter_queue").
			WithDetails("entry_id", id)
	}

	// Unmarshal metadata
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &entry.Metadata); err != nil {
			entry.Metadata = map[string]interface{}{}
		}
	}

	// Attempt reprocessing
	if err := reprocessFn(entry); err != nil {
		// Update attempt count
		dlq.updateAttempts(ctx, id, entry.Attempts+1)
		return gerror.Wrap(err, gerror.ErrCodeInternal, "reprocessing failed").
			WithComponent("dead_letter_queue").
			WithDetails("entry_id", id)
	}

	// Mark as processed on success
	return dlq.MarkProcessed(ctx, id)
}

// Clean removes old processed entries
func (dlq *DeadLetterQueue) Clean(ctx context.Context, olderThan time.Duration) (int, error) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	query := `
		DELETE FROM reasoning_failures
		WHERE processed = true AND processed_at < $1
	`

	result, err := dlq.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to clean dead letter queue").
			WithComponent("dead_letter_queue")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get cleaned rows").
			WithComponent("dead_letter_queue")
	}

	// Clean in-memory cache
	newEntries := make([]DeadLetterEntry, 0)
	for _, entry := range dlq.entries {
		if !entry.Processed || entry.ProcessedAt == nil || entry.ProcessedAt.After(cutoff) {
			newEntries = append(newEntries, entry)
		}
	}
	dlq.entries = newEntries

	return int(rows), nil
}

// Statistics returns queue statistics
func (dlq *DeadLetterQueue) Statistics(ctx context.Context) (map[string]interface{}, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count total entries
	var total int
	err := dlq.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reasoning_failures").Scan(&total)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to count total entries").
			WithComponent("dead_letter_queue")
	}
	stats["total_entries"] = total

	// Count unprocessed entries
	var unprocessed int
	err = dlq.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reasoning_failures WHERE processed = false").Scan(&unprocessed)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to count unprocessed entries").
			WithComponent("dead_letter_queue")
	}
	stats["unprocessed_entries"] = unprocessed

	// Get error distribution
	query := `
		SELECT error_code, COUNT(*) as count
		FROM reasoning_failures
		WHERE processed = false
		GROUP BY error_code
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err := dlq.db.QueryContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get error distribution").
			WithComponent("dead_letter_queue")
	}
	defer rows.Close()

	errorDist := make(map[string]int)
	for rows.Next() {
		var errorCode string
		var count int
		if err := rows.Scan(&errorCode, &count); err != nil {
			continue
		}
		errorDist[errorCode] = count
	}
	stats["error_distribution"] = errorDist

	// Cache stats
	stats["cache_size"] = len(dlq.entries)
	stats["max_cache_size"] = dlq.maxSize

	return stats, nil
}

// SetOnChange sets a callback for when entries are added
func (dlq *DeadLetterQueue) SetOnChange(fn func(DeadLetterEntry)) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	dlq.onChange = fn
}

// persistEntry stores an entry in the database
func (dlq *DeadLetterQueue) persistEntry(ctx context.Context, entry DeadLetterEntry) error {
	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Use INSERT OR REPLACE to handle duplicates
	query := `
		INSERT INTO reasoning_failures (
			id, agent_id, provider, error_type, error_message,
			content_preview, stack_trace, timestamp, created_at
		) VALUES (
			$1, $2, 'default', $3, $4, $5, $6, $7, CURRENT_TIMESTAMP
		)
	`

	// Truncate content for preview
	contentPreview := entry.Content
	if len(contentPreview) > 200 {
		contentPreview = contentPreview[:200] + "..."
	}

	_, err = dlq.db.ExecContext(ctx, query,
		entry.ID,
		entry.AgentID,
		entry.ErrorCode,
		entry.Error,
		contentPreview,
		string(metadataJSON), // Using stack_trace field for metadata
		entry.LastAttempt,
	)

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to insert dead letter entry").
			WithComponent("dead_letter_queue")
	}

	return nil
}

// updateAttempts updates the attempt count for an entry
func (dlq *DeadLetterQueue) updateAttempts(ctx context.Context, id string, attempts int) error {
	query := `
		UPDATE reasoning_failures
		SET attempts = $1, last_attempt = $2
		WHERE id = $3
	`

	_, err := dlq.db.ExecContext(ctx, query, attempts, time.Now(), id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update attempts").
			WithComponent("dead_letter_queue")
	}

	return nil
}
