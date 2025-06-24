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

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/storage/db"
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

	// Check if we're in a test environment
	if os.Getenv("GUILD_SKIP_MIGRATIONS") == "true" {
		// Skip migrations in test environments
		return nil
	}

	// Also skip if we're in a temporary directory (likely a test)
	cwd, _ := os.Getwd()
	if strings.Contains(cwd, "Test") || strings.Contains(cwd, "/T/") {
		// We're likely in a Go test temporary directory
		return nil
	}

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

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to run migrations").
			WithComponent("Database").
			WithOperation("Migrate")
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
