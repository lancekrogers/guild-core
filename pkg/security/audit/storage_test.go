// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftFileAuditStorage_NewFileAuditStorage(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, tempDir, storage.baseDir)
	assert.Equal(t, 1000, storage.maxEntries)

	// Verify directory was created
	_, err = os.Stat(tempDir)
	assert.NoError(t, err)
}

func TestGuildFileAuditStorage_Store_Retrieve(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Create test entry
	entry := AuditEntry{
		ID:         "test-entry-1",
		Timestamp:  time.Now(),
		AgentID:    "agent1",
		UserID:     "user1",
		SessionID:  "session1",
		Resource:   "file:/project/main.go",
		Action:     "read",
		Result:     ResultAllowed,
		Reason:     "permission granted",
		Duration:   100 * time.Millisecond,
		IPAddress:  "192.168.1.100",
		RiskScore:  2,
		Compliance: []string{"SOC2", "ISO27001"},
	}

	// Store entry
	err = storage.Store(ctx, entry)
	require.NoError(t, err)

	// Retrieve entry
	filter := AuditFilter{
		AgentID: "agent1",
		Limit:   10,
	}
	entries, err := storage.Retrieve(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entry.ID, entries[0].ID)
	assert.Equal(t, entry.AgentID, entries[0].AgentID)
	assert.Equal(t, entry.Resource, entries[0].Resource)
}

func TestJourneymanFileAuditStorage_Count(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Store multiple entries
	entries := []AuditEntry{
		{ID: "1", AgentID: "agent1", Resource: "file:test1", Action: "read", Result: ResultAllowed, Timestamp: time.Now()},
		{ID: "2", AgentID: "agent1", Resource: "file:test2", Action: "write", Result: ResultDenied, Timestamp: time.Now()},
		{ID: "3", AgentID: "agent2", Resource: "file:test3", Action: "read", Result: ResultAllowed, Timestamp: time.Now()},
	}

	for _, entry := range entries {
		err = storage.Store(ctx, entry)
		require.NoError(t, err)
	}

	// Count all entries
	count, err := storage.Count(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Count entries for specific agent
	count, err = storage.Count(ctx, AuditFilter{AgentID: "agent1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Count denied entries
	deniedResult := ResultDenied
	count, err = storage.Count(ctx, AuditFilter{Result: &deniedResult})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestCraftFileAuditStorage_FilterByTimeRange(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Store entries with different timestamps
	entries := []AuditEntry{
		{ID: "past", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: past},
		{ID: "now", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: now},
		{ID: "future", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: future},
	}

	for _, entry := range entries {
		err = storage.Store(ctx, entry)
		require.NoError(t, err)
	}

	// Filter by time range
	startTime := past.Add(30 * time.Minute)
	endTime := future.Add(-30 * time.Minute)
	filter := AuditFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	retrieved, err := storage.Retrieve(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, retrieved, 1)
	assert.Equal(t, "now", retrieved[0].ID)
}

func TestGuildFileAuditStorage_MaxEntries(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 2) // Small max entries
	require.NoError(t, err)

	// Store more entries than max
	entries := []AuditEntry{
		{ID: "1", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: time.Now()},
		{ID: "2", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: time.Now()},
		{ID: "3", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: time.Now()},
	}

	for _, entry := range entries {
		err = storage.Store(ctx, entry)
		require.NoError(t, err)
	}

	// Should only have the last 2 entries
	retrieved, err := storage.Retrieve(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)

	// Should be the most recent entries (2 and 3)
	ids := make([]string, len(retrieved))
	for i, entry := range retrieved {
		ids[i] = entry.ID
	}
	assert.Contains(t, ids, "2")
	assert.Contains(t, ids, "3")
	assert.NotContains(t, ids, "1")
}

func TestJourneymanFileAuditStorage_Delete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	now := time.Now()
	past := now.Add(-1 * time.Hour)

	// Store entries with different timestamps
	entries := []AuditEntry{
		{ID: "old1", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: past},
		{ID: "old2", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: past.Add(-30 * time.Minute)},
		{ID: "new", AgentID: "agent1", Resource: "test", Action: "read", Result: ResultAllowed, Timestamp: now},
	}

	for _, entry := range entries {
		err = storage.Store(ctx, entry)
		require.NoError(t, err)
	}

	// Delete entries older than 30 minutes ago
	cutoff := now.Add(-30 * time.Minute)
	err = storage.Delete(ctx, cutoff)
	require.NoError(t, err)

	// Should only have the new entry
	retrieved, err := storage.Retrieve(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Len(t, retrieved, 1)
	assert.Equal(t, "new", retrieved[0].ID)
}

func TestCraftFileAuditStorage_Backup(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Store test entry
	entry := AuditEntry{
		ID:        "backup-test",
		AgentID:   "agent1",
		Resource:  "test",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}
	err = storage.Store(ctx, entry)
	require.NoError(t, err)

	// Create backup
	backupPath := filepath.Join(tempDir, "backup.json")
	err = storage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Verify backup file exists and has content
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	data, err := os.ReadFile(backupPath) // #nosec G304 - backupPath is test-controlled path in tempDir
	require.NoError(t, err)
	assert.Contains(t, string(data), "backup-test")
	assert.Contains(t, string(data), "agent1")
}

func TestGuildFileAuditStorage_Health(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Health check should pass
	err = storage.Health(ctx)
	assert.NoError(t, err)

	// Remove directory and health check should fail
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
	err = storage.Health(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not accessible")
}

func TestJourneymanFileAuditStorage_FilterComplexity(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Store diverse entries
	entries := []AuditEntry{
		{
			ID: "high-risk", AgentID: "agent1", UserID: "user1", SessionID: "session1",
			Resource: "file:/sensitive/data", Action: "read", Result: ResultDenied,
			RiskScore: 8, IPAddress: "192.168.1.100", Compliance: []string{"SOC2"},
			Timestamp: time.Now(),
		},
		{
			ID: "low-risk", AgentID: "agent2", UserID: "user2", SessionID: "session2",
			Resource: "file:/public/docs", Action: "read", Result: ResultAllowed,
			RiskScore: 2, IPAddress: "192.168.1.101", Compliance: []string{"GDPR"},
			Timestamp: time.Now(),
		},
		{
			ID: "medium-risk", AgentID: "agent1", UserID: "user1", SessionID: "session3",
			Resource: "git:main", Action: "push", Result: ResultAllowed,
			RiskScore: 5, IPAddress: "192.168.1.100", Compliance: []string{"SOC2", "ISO27001"},
			Timestamp: time.Now(),
		},
	}

	for _, entry := range entries {
		err = storage.Store(ctx, entry)
		require.NoError(t, err)
	}

	// Test complex filter
	filter := AuditFilter{
		UserID:       "user1",
		MinRiskScore: 5,
		Compliance:   []string{"SOC2"},
		Limit:        10,
	}

	retrieved, err := storage.Retrieve(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2) // high-risk and medium-risk match user1, risk>=5, and SOC2

	// Test resource wildcard matching
	filter = AuditFilter{
		Resource: "file:*",
		Limit:    10,
	}

	retrieved, err = storage.Retrieve(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2) // Only the file: entries

	// Test pagination
	filter = AuditFilter{
		Limit:  1,
		Offset: 1,
	}

	retrieved, err = storage.Retrieve(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, retrieved, 1)
}

func TestCraftFileAuditStorage_PatternMatching(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	tests := []struct {
		name    string
		value   string
		pattern string
		matches bool
	}{
		{"exact match", "file:/path/to/file", "file:/path/to/file", true},
		{"wildcard all", "anything", "*", true},
		{"prefix wildcard", "file:/path/to/file", "file:*", true},
		{"prefix no match", "git:repo", "file:*", false},
		{"suffix wildcard", "file:/path/to/file.go", "*.go", true},
		{"suffix no match", "file:/path/to/file.py", "*.go", false},
		{"no wildcard no match", "file:/different/path", "file:/path/to/file", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matches := storage.matchesPattern(test.value, test.pattern)
			assert.Equal(t, test.matches, matches)
		})
	}
}

func TestGuildFileAuditStorage_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 1000)
	require.NoError(t, err)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	entry := AuditEntry{
		ID:        "test",
		AgentID:   "agent1",
		Resource:  "test",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}

	// Operations should handle cancellation gracefully
	err = storage.Store(cancelledCtx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")

	_, err = storage.Retrieve(cancelledCtx, AuditFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")

	_, err = storage.Count(cancelledCtx, AuditFilter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

// Benchmark tests

func BenchmarkCraftFileAuditStorage_Store(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 10000)
	require.NoError(b, err)

	entry := AuditEntry{
		AgentID:   "agent1",
		Resource:  "file:/project/main.go",
		Action:    "read",
		Result:    ResultAllowed,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.ID = string(rune(i))
		err := storage.Store(ctx, entry)
		if err != nil {
			b.Fatalf("Store failed: %v", err)
		}
	}
}

func BenchmarkJourneymanFileAuditStorage_Retrieve(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	storage, err := NewFileAuditStorage(ctx, tempDir, 10000)
	require.NoError(b, err)

	// Pre-populate with entries
	for i := 0; i < 1000; i++ {
		entry := AuditEntry{
			ID:        string(rune(i)),
			AgentID:   "agent1",
			Resource:  "file:/project/main.go",
			Action:    "read",
			Result:    ResultAllowed,
			Timestamp: time.Now(),
		}
		err := storage.Store(ctx, entry)
		if err != nil {
			b.Fatalf("Store failed: %v", err)
		}
	}

	filter := AuditFilter{
		AgentID: "agent1",
		Limit:   100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.Retrieve(ctx, filter)
		if err != nil {
			b.Fatalf("Retrieve failed: %v", err)
		}
	}
}
