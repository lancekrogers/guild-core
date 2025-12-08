// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// FileAuditStorage implements audit storage using file system
// This is a simple implementation for demonstration - production would use a database
type FileAuditStorage struct {
	baseDir    string
	logger     observability.Logger
	mu         sync.RWMutex
	entries    []AuditEntry
	maxEntries int
}

// NewFileAuditStorage creates a new file-based audit storage
func NewFileAuditStorage(ctx context.Context, baseDir string, maxEntries int) (*FileAuditStorage, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("FileAuditStorage")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("NewFileAuditStorage")
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create audit directory").
			WithComponent("FileAuditStorage").
			WithOperation("NewFileAuditStorage").
			WithDetails("base_dir", baseDir)
	}

	storage := &FileAuditStorage{
		baseDir:    baseDir,
		logger:     logger,
		entries:    make([]AuditEntry, 0),
		maxEntries: maxEntries,
	}

	// Load existing entries
	if err := storage.loadEntries(ctx); err != nil {
		logger.WithError(err).Warn("Failed to load existing audit entries")
	}

	logger.Info("File audit storage initialized",
		"base_dir", baseDir,
		"max_entries", maxEntries,
		"loaded_entries", len(storage.entries),
	)

	return storage, nil
}

// Store saves an audit entry
func (fas *FileAuditStorage) Store(ctx context.Context, entry AuditEntry) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Store")
	}

	fas.mu.Lock()
	defer fas.mu.Unlock()

	// Add entry to in-memory storage
	fas.entries = append(fas.entries, entry)

	// Trim if we exceed max entries
	if len(fas.entries) > fas.maxEntries {
		fas.entries = fas.entries[len(fas.entries)-fas.maxEntries:]
	}

	// Persist to file
	return fas.persistEntries(ctx)
}

// Retrieve gets audit entries based on filter
func (fas *FileAuditStorage) Retrieve(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Retrieve")
	}

	fas.mu.RLock()
	defer fas.mu.RUnlock()

	var filtered []AuditEntry

	for _, entry := range fas.entries {
		if fas.matchesFilter(entry, filter) {
			filtered = append(filtered, entry)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply limit and offset
	if filter.Offset > 0 {
		if filter.Offset >= len(filtered) {
			return []AuditEntry{}, nil
		}
		filtered = filtered[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(filtered) {
		filtered = filtered[:filter.Limit]
	}

	fas.logger.Debug("Retrieved audit entries",
		"total_entries", len(fas.entries),
		"filtered_entries", len(filtered),
	)

	return filtered, nil
}

// Count returns the number of entries matching the filter
func (fas *FileAuditStorage) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Count")
	}

	fas.mu.RLock()
	defer fas.mu.RUnlock()

	count := int64(0)
	for _, entry := range fas.entries {
		if fas.matchesFilter(entry, filter) {
			count++
		}
	}

	return count, nil
}

// Delete removes audit entries (typically for archival)
func (fas *FileAuditStorage) Delete(ctx context.Context, before time.Time) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Delete")
	}

	fas.mu.Lock()
	defer fas.mu.Unlock()

	originalCount := len(fas.entries)
	var remaining []AuditEntry

	for _, entry := range fas.entries {
		if entry.Timestamp.After(before) {
			remaining = append(remaining, entry)
		}
	}

	fas.entries = remaining

	fas.logger.Info("Deleted old audit entries",
		"before", before,
		"deleted_count", originalCount-len(remaining),
		"remaining_count", len(remaining),
	)

	return fas.persistEntries(ctx)
}

// Backup creates a backup of audit data
func (fas *FileAuditStorage) Backup(ctx context.Context, destination string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Backup")
	}

	fas.mu.RLock()
	defer fas.mu.RUnlock()

	// Create backup directory
	backupDir := filepath.Dir(destination)
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create backup directory").
			WithComponent("FileAuditStorage").
			WithOperation("Backup")
	}

	// Write entries to backup file
	data, err := json.MarshalIndent(fas.entries, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal audit entries").
			WithComponent("FileAuditStorage").
			WithOperation("Backup")
	}

	if err := os.WriteFile(destination, data, 0o600); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write backup file").
			WithComponent("FileAuditStorage").
			WithOperation("Backup").
			WithDetails("destination", destination)
	}

	fas.logger.Info("Audit backup completed",
		"destination", destination,
		"entries_count", len(fas.entries),
	)

	return nil
}

// Health checks storage health
func (fas *FileAuditStorage) Health(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FileAuditStorage").
			WithOperation("Health")
	}

	// Check if base directory is accessible
	if _, err := os.Stat(fas.baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "audit storage directory not accessible").
			WithComponent("FileAuditStorage").
			WithOperation("Health")
	}

	// Check if we can write to the directory
	testFile := filepath.Join(fas.baseDir, ".health_check")
	if err := os.WriteFile(testFile, []byte("health_check"), 0o600); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "cannot write to audit storage directory").
			WithComponent("FileAuditStorage").
			WithOperation("Health")
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log but don't fail health check for cleanup issues
		fas.logger.WithError(err).Debug("Failed to remove health check test file")
	}

	return nil
}

// Helper methods

func (fas *FileAuditStorage) loadEntries(ctx context.Context) error {
	auditFile := filepath.Join(fas.baseDir, "audit.json")

	data, err := os.ReadFile(auditFile) // #nosec G304 - auditFile is constructed from validated baseDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing file, start fresh
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read audit file")
	}

	var entries []AuditEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal audit entries")
	}

	fas.entries = entries
	return nil
}

func (fas *FileAuditStorage) persistEntries(ctx context.Context) error {
	auditFile := filepath.Join(fas.baseDir, "audit.json")

	data, err := json.MarshalIndent(fas.entries, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal audit entries")
	}

	if err := os.WriteFile(auditFile, data, 0o600); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write audit file")
	}

	return nil
}

func (fas *FileAuditStorage) matchesFilter(entry AuditEntry, filter AuditFilter) bool {
	// Time range filter
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}

	// Agent ID filter
	if filter.AgentID != "" && entry.AgentID != filter.AgentID {
		return false
	}

	// User ID filter
	if filter.UserID != "" && entry.UserID != filter.UserID {
		return false
	}

	// Session ID filter
	if filter.SessionID != "" && entry.SessionID != filter.SessionID {
		return false
	}

	// Resource filter (supports wildcard matching)
	if filter.Resource != "" && !fas.matchesPattern(entry.Resource, filter.Resource) {
		return false
	}

	// Action filter
	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}

	// Result filter
	if filter.Result != nil && entry.Result != *filter.Result {
		return false
	}

	// IP Address filter
	if filter.IPAddress != "" && entry.IPAddress != filter.IPAddress {
		return false
	}

	// Risk score filter
	if filter.MinRiskScore > 0 && entry.RiskScore < filter.MinRiskScore {
		return false
	}

	// Compliance filter
	if len(filter.Compliance) > 0 {
		found := false
		for _, reqCompliance := range filter.Compliance {
			for _, entryCompliance := range entry.Compliance {
				if entryCompliance == reqCompliance {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (fas *FileAuditStorage) matchesPattern(value, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	}

	return value == pattern
}
