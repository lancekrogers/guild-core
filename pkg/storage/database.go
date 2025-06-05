package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/storage/db"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// Database represents a SQLite database connection with migrations
// Following Guild's pattern of encapsulating database operations
type Database struct {
	db      *sql.DB
	queries *db.Queries
	dbPath  string
}

// NewDatabase creates a new database connection and runs migrations
// Following Guild's constructor pattern with proper error wrapping
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database
	sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:      sqlDB,
		queries: db.New(sqlDB),
		dbPath:  dbPath,
	}

	return database, nil
}

// Migrate runs database migrations following Guild's error handling patterns
func (d *Database) Migrate(ctx context.Context) error {
	// Create migration driver
	driver, err := sqlite3.WithInstance(d.db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Get migrations directory - try multiple locations
	var migrationsPath string
	
	// 1. Try current working directory
	if _, err := os.Stat("db/migrations"); err == nil {
		migrationsPath = "file://db/migrations"
	} else {
		// 2. Try walking up from current directory to find db/migrations
		currentDir, err := os.Getwd()
		if err == nil {
			for i := 0; i < 10; i++ { // Try up to 10 levels up
				potentialPath := filepath.Join(currentDir, "db", "migrations")
				if _, err := os.Stat(potentialPath); err == nil {
					migrationsPath = fmt.Sprintf("file://%s", potentialPath)
					break
				}
				parentDir := filepath.Dir(currentDir)
				if parentDir == currentDir {
					break // Reached filesystem root
				}
				currentDir = parentDir
			}
		}
		
		// 3. Try relative to executable as fallback
		if migrationsPath == "" {
			execPath, err := os.Executable()
			if err == nil {
				execDir := filepath.Dir(execPath)
				potentialPath := filepath.Join(execDir, "db", "migrations")
				if _, err := os.Stat(potentialPath); err == nil {
					migrationsPath = fmt.Sprintf("file://%s", potentialPath)
				} else {
					// Try going up from executable to find db/migrations
					for i := 0; i < 5; i++ { // Try up to 5 levels up
						execDir = filepath.Dir(execDir)
						potentialPath = filepath.Join(execDir, "db", "migrations")
						if _, err := os.Stat(potentialPath); err == nil {
							migrationsPath = fmt.Sprintf("file://%s", potentialPath)
							break
						}
					}
				}
			}
		}
	}
	
	if migrationsPath == "" {
		return fmt.Errorf("could not find database migrations directory")
	}

	// Create file source
	source, err := (&file.File{}).Open(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to open migrations source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("file", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
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
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}

// Transaction executes a function within a database transaction
// Following Guild's context-aware transaction pattern
func (d *Database) Transaction(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}