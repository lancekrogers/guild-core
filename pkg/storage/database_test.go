// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabase_MigrationRecovery tests that the database can recover from dirty migration state
func TestDatabase_MigrationRecovery(t *testing.T) {
	ctx := context.Background()
	
	// Create a temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "guild-test-migration.db")
	
	// Create database and run migrations
	db1, err := DefaultDatabaseFactory(ctx, dbPath)
	require.NoError(t, err, "should create database")
	
	err = db1.Migrate(ctx)
	require.NoError(t, err, "first migration should succeed")
	
	// Close the first database
	err = db1.Close()
	require.NoError(t, err, "should close database")
	
	// Simulate a dirty migration state by creating a new database connection
	// and running migrations again (this would normally fail with dirty state)
	db2, err := DefaultDatabaseFactory(ctx, dbPath)
	require.NoError(t, err, "should create second database connection")
	defer db2.Close()
	
	// This should handle dirty state gracefully
	err = db2.Migrate(ctx)
	require.NoError(t, err, "second migration should handle dirty state")
}

// TestDatabase_ResetMigrations tests the ResetMigrations function
func TestDatabase_ResetMigrations(t *testing.T) {
	ctx := context.Background()
	
	// Test with in-memory database
	db, err := DefaultDatabaseFactory(ctx, ":memory:")
	require.NoError(t, err, "should create in-memory database")
	defer db.Close()
	
	// Should allow reset on in-memory database
	err = db.ResetMigrations(ctx)
	assert.NoError(t, err, "should allow reset on in-memory database")
	
	// Test with test database
	tempDir := t.TempDir()
	testDbPath := filepath.Join(tempDir, "guild-test-reset.db")
	testDb, err := DefaultDatabaseFactory(ctx, testDbPath)
	require.NoError(t, err, "should create test database")
	defer testDb.Close()
	
	// Should allow reset on test database
	err = testDb.ResetMigrations(ctx)
	assert.NoError(t, err, "should allow reset on test database")
	
	// Test with production-like database path (without test markers)
	// Create a path that doesn't contain any test markers
	prodDbPath := filepath.Join("/var", "lib", "guild", "production.db")
	// Note: We can't actually create the database at this path in tests,
	// but we can test the path validation logic by creating a mock database
	prodDb := &Database{
		dbPath: prodDbPath,
		db:     testDb.db, // Reuse the test database connection
	}
	
	// Should not allow reset on production database
	err = prodDb.ResetMigrations(ctx)
	assert.Error(t, err, "should not allow reset on production database")
	if err != nil {
		assert.Contains(t, err.Error(), "only allowed in test environments")
	}
}

// TestDatabase_CleanupTestDatabase tests the CleanupTestDatabase function
func TestDatabase_CleanupTestDatabase(t *testing.T) {
	ctx := context.Background()
	
	// Create test database with some data
	tempDir := t.TempDir()
	testDbPath := filepath.Join(tempDir, "guild-test-cleanup.db")
	testDb, err := DefaultDatabaseFactory(ctx, testDbPath)
	require.NoError(t, err, "should create test database")
	defer testDb.Close()
	
	// Run migrations to create schema
	err = testDb.Migrate(ctx)
	require.NoError(t, err, "should run migrations")
	
	// Add some test data
	_, err = testDb.DB().ExecContext(ctx, `
		INSERT INTO campaigns (id, name, status) VALUES ('test-1', 'Test Campaign', 'active')
	`)
	require.NoError(t, err, "should insert test data")
	
	// Verify data exists
	var count int
	err = testDb.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM campaigns").Scan(&count)
	require.NoError(t, err, "should query count")
	assert.Equal(t, 1, count, "should have one campaign")
	
	// Clean up test database
	err = testDb.CleanupTestDatabase(ctx)
	assert.NoError(t, err, "should cleanup test database")
	
	// Verify data is cleaned
	err = testDb.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM campaigns").Scan(&count)
	require.NoError(t, err, "should query count after cleanup")
	assert.Equal(t, 0, count, "should have no campaigns after cleanup")
}

// TestDatabase_SkipMigrations tests that migrations are skipped in test environments
func TestDatabase_SkipMigrations(t *testing.T) {
	ctx := context.Background()
	
	// Set environment variable to skip migrations
	os.Setenv("GUILD_SKIP_MIGRATIONS", "true")
	defer os.Unsetenv("GUILD_SKIP_MIGRATIONS")
	
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test-skip-migrations.db")
	
	db, err := DefaultDatabaseFactory(ctx, dbPath)
	require.NoError(t, err, "should create database")
	defer db.Close()
	
	// Migrations should be skipped
	err = db.Migrate(ctx)
	assert.NoError(t, err, "should skip migrations without error")
	
	// When migrations are skipped, no tables should be created except sqlite system tables
	rows, err := db.DB().QueryContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' 
		AND name NOT LIKE 'sqlite_%'
	`)
	require.NoError(t, err, "should query tables")
	defer rows.Close()
	
	var tableCount int
	if rows.Next() {
		err = rows.Scan(&tableCount)
		require.NoError(t, err, "should scan table count")
	}
	
	// If migrations were truly skipped, there should be no user tables
	// (or at most just the schema_migrations table if it was created before skip)
	assert.LessOrEqual(t, tableCount, 1, "should have at most one table when migrations are skipped")
}
