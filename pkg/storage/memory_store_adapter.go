// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/memory"
)

// MemoryStoreAdapter adapts SQLite repositories to implement memory.Store interface
// This allows the kanban system to work with SQLite through the memory.Store abstraction
type MemoryStoreAdapter struct {
	database *Database
}

// NewMemoryStoreAdapter creates a new memory store adapter using SQLite database
func NewMemoryStoreAdapter(database *Database) *MemoryStoreAdapter {
	return &MemoryStoreAdapter{
		database: database,
	}
}

// Put stores a value with the given key
func (m *MemoryStoreAdapter) Put(ctx context.Context, bucket, key string, value []byte) error {
	if m.database == nil {
		return gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("Put")
	}

	// Store as a simple key-value mapping in a dedicated table
	query := `
		INSERT OR REPLACE INTO memory_store (bucket, key, value)
		VALUES (?, ?, ?)
	`

	_, err := m.database.DB().ExecContext(ctx, query, bucket, key, value)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store value").
			WithComponent("MemoryStoreAdapter").
			WithOperation("Put").
			WithDetails("bucket", bucket).
			WithDetails("key", key)
	}

	return nil
}

// Get retrieves a value by key
func (m *MemoryStoreAdapter) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	if m.database == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("Get")
	}

	query := `SELECT value FROM memory_store WHERE bucket = ? AND key = ?`

	var value []byte
	err := m.database.DB().QueryRowContext(ctx, query, bucket, key).Scan(&value)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, memory.ErrNotFound
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get value").
			WithComponent("MemoryStoreAdapter").
			WithOperation("Get").
			WithDetails("bucket", bucket).
			WithDetails("key", key)
	}

	return value, nil
}

// Delete removes a value by key
func (m *MemoryStoreAdapter) Delete(ctx context.Context, bucket, key string) error {
	if m.database == nil {
		return gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("Delete")
	}

	query := `DELETE FROM memory_store WHERE bucket = ? AND key = ?`

	_, err := m.database.DB().ExecContext(ctx, query, bucket, key)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete value").
			WithComponent("MemoryStoreAdapter").
			WithOperation("Delete").
			WithDetails("bucket", bucket).
			WithDetails("key", key)
	}

	return nil
}

// List returns all keys in a bucket
func (m *MemoryStoreAdapter) List(ctx context.Context, bucket string) ([]string, error) {
	if m.database == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("List")
	}

	query := `SELECT key FROM memory_store WHERE bucket = ? ORDER BY key`

	rows, err := m.database.DB().QueryContext(ctx, query, bucket)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list keys").
			WithComponent("MemoryStoreAdapter").
			WithOperation("List").
			WithDetails("bucket", bucket)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log close error
			_ = gerror.Wrap(closeErr, gerror.ErrCodeStorage, "failed to close rows").
				WithComponent("MemoryStoreAdapter").
				WithOperation("List")
		}
	}()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan key").
				WithComponent("MemoryStoreAdapter").
				WithOperation("List")
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error during key listing").
			WithComponent("MemoryStoreAdapter").
			WithOperation("List")
	}

	return keys, nil
}

// ListKeys returns keys with the given prefix in a bucket
func (m *MemoryStoreAdapter) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if m.database == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("ListKeys")
	}

	query := `SELECT key FROM memory_store WHERE bucket = ? AND key LIKE ? ORDER BY key`

	rows, err := m.database.DB().QueryContext(ctx, query, bucket, prefix+"%")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list keys with prefix").
			WithComponent("MemoryStoreAdapter").
			WithOperation("ListKeys").
			WithDetails("bucket", bucket).
			WithDetails("prefix", prefix)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log close error
			_ = gerror.Wrap(closeErr, gerror.ErrCodeStorage, "failed to close rows").
				WithComponent("MemoryStoreAdapter").
				WithOperation("ListKeys")
		}
	}()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan key").
				WithComponent("MemoryStoreAdapter").
				WithOperation("ListKeys")
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error during key listing").
			WithComponent("MemoryStoreAdapter").
			WithOperation("ListKeys")
	}

	return keys, nil
}

// Close closes the store
func (m *MemoryStoreAdapter) Close() error {
	// Database is managed externally, so this is a no-op
	return nil
}

// Helper function to ensure memory_store table exists for tests
func (m *MemoryStoreAdapter) EnsureMemoryStoreTable(ctx context.Context) error {
	if m.database == nil {
		return gerror.New(gerror.ErrCodeInternal, "database not initialized", nil).
			WithComponent("MemoryStoreAdapter").
			WithOperation("EnsureMemoryStoreTable")
	}

	query := `
		CREATE TABLE IF NOT EXISTS memory_store (
			bucket TEXT NOT NULL,
			key TEXT NOT NULL,
			value BLOB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (bucket, key)
		)
	`

	_, err := m.database.DB().ExecContext(ctx, query)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create memory_store table").
			WithComponent("MemoryStoreAdapter").
			WithOperation("EnsureMemoryStoreTable")
	}

	return nil
}
