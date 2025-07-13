// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_users_email ON users(email);
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			title TEXT,
			content TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	require.NoError(t, err)

	// Insert test data
	for i := 0; i < 10; i++ {
		_, err = db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
			fmt.Sprintf("User%d", i),
			fmt.Sprintf("user%d@example.com", i))
		require.NoError(t, err)
	}

	return db
}

func TestNewQueryOptimizer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		db      *sql.DB
		wantErr bool
	}{
		{
			name:    "Valid database",
			db:      db,
			wantErr: false,
		},
		{
			name:    "Nil database",
			db:      nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer, err := NewQueryOptimizer(tt.db, nil)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, optimizer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, optimizer)
				if optimizer != nil {
					optimizer.Close()
				}
			}
		})
	}
}

func TestQueryOptimizer_AnalyzeQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	optimizer, err := NewQueryOptimizer(db, nil)
	require.NoError(t, err)
	defer optimizer.Close()

	ctx := context.Background()

	tests := []struct {
		name            string
		query           string
		wantIndexUse    bool
		wantSuggestions bool
	}{
		{
			name:            "Query with index",
			query:           "SELECT * FROM users WHERE email = 'test@example.com'",
			wantIndexUse:    true,
			wantSuggestions: true, // SELECT * suggestion
		},
		{
			name:            "Query without index",
			query:           "SELECT * FROM users WHERE name = 'Test'",
			wantIndexUse:    false,
			wantSuggestions: true, // Table scan + SELECT *
		},
		{
			name:            "Query with LIKE wildcard",
			query:           "SELECT id FROM users WHERE email LIKE '%@example.com'",
			wantIndexUse:    false,
			wantSuggestions: true, // Leading wildcard
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := optimizer.AnalyzeQuery(ctx, tt.query)
			assert.NoError(t, err)
			assert.NotNil(t, plan)

			if tt.wantIndexUse {
				assert.NotEmpty(t, plan.IndexesUsed)
			}

			if tt.wantSuggestions {
				assert.NotEmpty(t, plan.Suggestions)
			}
		})
	}
}

func TestQueryOptimizer_PrepareStatement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	optimizer, err := NewQueryOptimizer(db, nil)
	require.NoError(t, err)
	defer optimizer.Close()

	ctx := context.Background()

	// Prepare a statement
	stmt1, err := optimizer.PrepareStatement(ctx, "get_user", "SELECT * FROM users WHERE id = ?")
	assert.NoError(t, err)
	assert.NotNil(t, stmt1)

	// Prepare same statement again - should return cached
	stmt2, err := optimizer.PrepareStatement(ctx, "get_user", "SELECT * FROM users WHERE id = ?")
	assert.NoError(t, err)
	assert.Equal(t, stmt1, stmt2)

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = optimizer.PrepareStatement(cancelledCtx, "test", "SELECT 1")
	assert.Error(t, err)
}

func TestQueryOptimizer_TrackQueryExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	optimizer, err := NewQueryOptimizer(db, nil)
	require.NoError(t, err)
	defer optimizer.Close()

	// Track several executions
	query := "SELECT * FROM users WHERE id = ?"
	durations := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, duration := range durations {
		optimizer.TrackQueryExecution(query, duration)
	}

	// Get stats
	stats := optimizer.GetQueryStats()
	assert.Len(t, stats, 1)

	queryStats, exists := stats[optimizer.normalizeQuery(query)]
	assert.True(t, exists)
	assert.Equal(t, int64(3), queryStats.ExecutionCount)
	assert.Equal(t, 50*time.Millisecond, queryStats.MinDuration)
	assert.Equal(t, 150*time.Millisecond, queryStats.MaxDuration)
	assert.Equal(t, 100*time.Millisecond, queryStats.AvgDuration)
}

func TestQueryOptimizer_OptimizeIndexes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	optimizer, err := NewQueryOptimizer(db, nil)
	require.NoError(t, err)
	defer optimizer.Close()

	ctx := context.Background()

	// Track some slow queries without indexes
	for i := 0; i < 200; i++ {
		optimizer.TrackQueryExecution(
			"SELECT * FROM posts WHERE title = 'test'",
			200*time.Millisecond,
		)
	}

	// Get optimization suggestions
	suggestions, err := optimizer.OptimizeIndexes(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, suggestions)
}

func TestQueryCache(t *testing.T) {
	cache, err := NewQueryCache(DefaultQueryCacheConfig())
	require.NoError(t, err)
	require.NotNil(t, cache)

	ctx := context.Background()

	// Test Set and Get
	query := "SELECT * FROM users WHERE id = ?"
	args := []interface{}{1}
	result := "test result"

	err = cache.Set(ctx, query, args, result, 5*time.Minute)
	assert.NoError(t, err)

	// Get from cache
	cached, found := cache.Get(ctx, query, args)
	assert.True(t, found)
	assert.Equal(t, result, cached)

	// Test cache miss
	_, found = cache.Get(ctx, "different query", args)
	assert.False(t, found)

	// Test invalidation
	count := cache.Invalidate("*")
	assert.Greater(t, count, 0)

	// Verify cache is empty
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Entries)

	// Test Clear
	cache.Set(ctx, query, args, result, 5*time.Minute)
	cache.Clear()

	_, found = cache.Get(ctx, query, args)
	assert.False(t, found)
}

func TestQueryCache_Expiration(t *testing.T) {
	config := DefaultQueryCacheConfig()
	config.DefaultTTL = 100 * time.Millisecond
	config.CleanupInterval = 50 * time.Millisecond

	cache, err := NewQueryCache(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Set with short TTL
	err = cache.Set(ctx, "test", []interface{}{}, "value", 100*time.Millisecond)
	assert.NoError(t, err)

	// Should be found immediately
	_, found := cache.Get(ctx, "test", []interface{}{})
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	_, found = cache.Get(ctx, "test", []interface{}{})
	assert.False(t, found)
}

func TestQueryCache_LRU(t *testing.T) {
	config := DefaultQueryCacheConfig()
	config.MaxEntries = 3

	cache, err := NewQueryCache(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		query := fmt.Sprintf("query%d", i)
		err = cache.Set(ctx, query, nil, i, 5*time.Minute)
		assert.NoError(t, err)
	}

	// Access first query to make it most recently used
	cache.Get(ctx, "query0", nil)

	// Add one more - should evict query1 (LRU)
	err = cache.Set(ctx, "query3", nil, 3, 5*time.Minute)
	assert.NoError(t, err)

	// Check what's in cache
	_, found := cache.Get(ctx, "query0", nil)
	assert.True(t, found, "query0 should be in cache (recently accessed)")

	_, found = cache.Get(ctx, "query1", nil)
	assert.False(t, found, "query1 should be evicted (LRU)")

	_, found = cache.Get(ctx, "query2", nil)
	assert.True(t, found, "query2 should be in cache")

	_, found = cache.Get(ctx, "query3", nil)
	assert.True(t, found, "query3 should be in cache (just added)")
}
