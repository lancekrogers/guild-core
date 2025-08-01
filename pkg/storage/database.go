// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/storage/db"
)

//go:embed all:migrations/*.sql
var migrations embed.FS

// Database represents a SQLite database connection with migrations
// Following Guild's pattern of encapsulating database operations
type Database struct {
	db      *sql.DB
	queries *db.Queries
	dbPath  string
}

// newDatabase creates a new database connection and runs migrations (private constructor)
// Following Guild's constructor pattern with proper error wrapping
func newDatabase(ctx context.Context, dbPath string) (*Database, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Database").
			WithOperation("newDatabase")
	}

	// Check if this is a test database
	isTestDb := strings.Contains(dbPath, "guild-test-") || 
		strings.Contains(dbPath, "/tmp/") || 
		strings.Contains(dbPath, "Test") ||
		strings.Contains(dbPath, "/T/") ||
		strings.Contains(dbPath, ":memory:")

	// For test databases, remove any existing database files to ensure clean state
	if isTestDb && dbPath != ":memory:" {
		// Remove all database-related files
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-shm")
		_ = os.Remove(dbPath + "-wal")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database directory").
			WithComponent("Database").
			WithOperation("newDatabase").
			WithDetails("db_path", dbPath)
	}

	// Check context before opening database
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before database open").
			WithComponent("Database").
			WithOperation("newDatabase")
	}

	// Open SQLite database
	sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to open database").
			WithComponent("Database").
			WithOperation("newDatabase").
			WithDetails("db_path", dbPath)
	}

	// Test connection
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to ping database").
			WithComponent("Database").
			WithOperation("newDatabase").
			WithDetails("db_path", dbPath)
	}

	database := &Database{
		db:      sqlDB,
		queries: db.New(sqlDB),
		dbPath:  dbPath,
	}

	return database, nil
}

// DefaultDatabaseFactory creates a database instance for registry use
func DefaultDatabaseFactory(ctx context.Context, dbPath string) (*Database, error) {
	return newDatabase(ctx, dbPath)
}

// Migrate runs database migrations following Guild's error handling patterns
func (d *Database) Migrate(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Database").
			WithOperation("Migrate")
	}

	// Create migration driver
	driver, err := sqlite3.WithInstance(d.db, &sqlite3.Config{})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create migration driver").
			WithComponent("Database").
			WithOperation("Migrate")
	}

	// Never skip migrations - they are required for proper database setup
	// Test environments should handle cleanup properly instead of skipping migrations

	// Create migration source from embedded filesystem
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create embedded migrations source").
			WithComponent("Database").
			WithOperation("Migrate")
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("file", source, "sqlite3", driver)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create migrate instance").
			WithComponent("Database").
			WithOperation("Migrate")
	}

	// Check if this is a test database
	isTestDb := strings.Contains(d.dbPath, "guild-test-") || 
		strings.Contains(d.dbPath, "/tmp/") || 
		strings.Contains(d.dbPath, "Test") ||
		strings.Contains(d.dbPath, "/T/") ||
		strings.Contains(d.dbPath, ":memory:")

	// For test databases, always start fresh by dropping schema_migrations
	if isTestDb {
		// Drop the schema_migrations table to force clean state
		_, dropErr := d.db.ExecContext(ctx, "DROP TABLE IF EXISTS schema_migrations")
		if dropErr == nil {
			// Recreate migration instance after dropping table
			driver, err = sqlite3.WithInstance(d.db, &sqlite3.Config{})
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to recreate migration driver after drop").
					WithComponent("Database").
					WithOperation("Migrate")
			}
			m, err = migrate.NewWithInstance("file", source, "sqlite3", driver)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to recreate migrate instance after drop").
					WithComponent("Database").
					WithOperation("Migrate")
			}
		}
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// Check if it's a dirty database error
		if strings.Contains(err.Error(), "Dirty database") && isTestDb {
			// Get current version and dirty state
			version, dirty, vErr := m.Version()
			if vErr == nil && dirty {
				// Force clean the dirty state
				if version > 0 {
					_ = m.Force(int(version - 1))
				} else {
					// If at version 0, force to -1 to reset completely
					_ = m.Force(-1)
				}
				// Retry migrations after forcing
				err = m.Up()
			}
		}
		
		if err != nil && err != migrate.ErrNoChange {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to run migrations").
				WithComponent("Database").
				WithOperation("Migrate").
				WithDetails("db_path", d.dbPath).
				WithDetails("is_test_db", isTestDb)
		}
	}

	return nil
}

// DB returns the underlying sql.DB for advanced operations
// Following Guild's pattern of exposing underlying connections when needed
func (d *Database) DB() *sql.DB {
	return d.db
}

// Queries returns the SQLC generated queries
// Following Guild's pattern of exposing typed query interfaces
func (d *Database) Queries() *db.Queries {
	return d.queries
}

// Close closes the database connection
// Following Guild's cleanup pattern
func (d *Database) Close() error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to close database").
				WithComponent("Database").
				WithOperation("Close")
		}
	}
	return nil
}

// ResetMigrations forces a clean migration state for testing
// This should only be used in test environments
func (d *Database) ResetMigrations(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Database").
			WithOperation("ResetMigrations")
	}

	// Only allow in test environments
	if !strings.Contains(d.dbPath, "guild-test-") && !strings.Contains(d.dbPath, "/tmp/") && !strings.Contains(d.dbPath, "Test") && !strings.Contains(d.dbPath, ":memory:") {
		return gerror.New(gerror.ErrCodeInvalidInput, "ResetMigrations is only allowed in test environments", nil).
			WithComponent("Database").
			WithOperation("ResetMigrations")
	}

	// Drop the schema_migrations table to force a clean state
	_, err := d.db.ExecContext(ctx, "DROP TABLE IF EXISTS schema_migrations")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to drop schema_migrations table").
			WithComponent("Database").
			WithOperation("ResetMigrations")
	}

	return nil
}

// CleanupTestDatabase removes all data from the database for testing
// This should only be used in test environments
func (d *Database) CleanupTestDatabase(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Database").
			WithOperation("CleanupTestDatabase")
	}

	// Only allow in test environments
	if !strings.Contains(d.dbPath, "guild-test-") && !strings.Contains(d.dbPath, "/tmp/") && !strings.Contains(d.dbPath, "Test") && !strings.Contains(d.dbPath, ":memory:") {
		return gerror.New(gerror.ErrCodeInvalidInput, "CleanupTestDatabase is only allowed in test environments", nil).
			WithComponent("Database").
			WithOperation("CleanupTestDatabase")
	}

	// Get all tables
	rows, err := d.db.QueryContext(ctx, `
		SELECT name FROM sqlite_master 
		WHERE type='table' 
		AND name NOT LIKE 'sqlite_%' 
		AND name != 'schema_migrations'
	`)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query tables").
			WithComponent("Database").
			WithOperation("CleanupTestDatabase")
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan table name").
				WithComponent("Database").
				WithOperation("CleanupTestDatabase")
		}
		tables = append(tables, table)
	}

	// Disable foreign keys temporarily
	if _, err := d.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to disable foreign keys").
			WithComponent("Database").
			WithOperation("CleanupTestDatabase")
	}

	// Delete all data from tables
	for _, table := range tables {
		if _, err := d.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to clean table").
				WithComponent("Database").
				WithOperation("CleanupTestDatabase").
				WithDetails("table", table)
		}
	}

	// Re-enable foreign keys
	if _, err := d.db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to re-enable foreign keys").
			WithComponent("Database").
			WithOperation("CleanupTestDatabase")
	}

	return nil
}

// Transaction executes a function within a database transaction
// Following Guild's context-aware transaction pattern
func (d *Database) Transaction(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeTransaction, "failed to begin transaction")
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeTransaction, "transaction function failed")
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeTransaction, "failed to commit transaction")
	}

	return nil
}
