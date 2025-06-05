package storage

import (
	"context"
	"fmt"
)

// InitializeSQLiteStorageForRegistry initializes SQLite storage and returns configured components
// This function is designed to be called by the registry to avoid circular imports
func InitializeSQLiteStorageForRegistry(ctx context.Context, dbPath string) (StorageRegistry, interface{}, error) {
	// Create database connection
	database, err := NewDatabase(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Run migrations
	if err := database.Migrate(ctx); err != nil {
		database.Close()
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create storage registry
	storageRegistry := NewStorageRegistry()

	// Create and register repositories
	taskRepo := NewSQLiteTaskRepository(database)
	campaignRepo := NewSQLiteCampaignRepository(database)
	commissionRepo := NewSQLiteCommissionRepository(database)
	boardRepo := NewSQLiteBoardRepository(database)
	agentRepo := NewSQLiteAgentRepository(database)

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)

	// Create SQLite store adapter for memory.Store interface compatibility
	storeAdapter := NewSQLiteStoreAdapter(storageRegistry)

	// Return both the storage registry and the memory store adapter
	return storageRegistry, storeAdapter, nil
}

// InitializeSQLiteStorageForTests initializes SQLite storage without migrations for testing
// This creates an in-memory database and manually creates the schema
func InitializeSQLiteStorageForTests(ctx context.Context) (StorageRegistry, interface{}, error) {
	// Use in-memory SQLite database for tests
	database, err := NewDatabase(":memory:")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create test database: %w", err)
	}

	// Manually create schema instead of running migrations
	if err := createTestSchema(database); err != nil {
		database.Close()
		return nil, nil, fmt.Errorf("failed to create test schema: %w", err)
	}

	// Create storage registry
	storageRegistry := NewStorageRegistry()

	// Create and register repositories
	taskRepo := NewSQLiteTaskRepository(database)
	campaignRepo := NewSQLiteCampaignRepository(database)
	commissionRepo := NewSQLiteCommissionRepository(database)
	boardRepo := NewSQLiteBoardRepository(database)
	agentRepo := NewSQLiteAgentRepository(database)

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)

	// Create SQLite store adapter for memory.Store interface compatibility
	storeAdapter := NewSQLiteStoreAdapter(storageRegistry)

	// Return both the storage registry and the memory store adapter
	return storageRegistry, storeAdapter, nil
}

// createTestSchema manually creates the database schema for tests
func createTestSchema(database *Database) error {
	schema := `
	CREATE TABLE campaigns (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE commissions (
		id TEXT PRIMARY KEY,
		campaign_id TEXT NOT NULL REFERENCES campaigns(id),
		title TEXT NOT NULL,
		description TEXT,
		domain TEXT,
		context JSON,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE boards (
		id TEXT PRIMARY KEY,
		commission_id TEXT NOT NULL REFERENCES commissions(id),
		name TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(commission_id) -- Ensures one board per commission
	);

	CREATE TABLE agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		provider TEXT,
		model TEXT,
		capabilities JSON,
		tools JSON,
		cost_magnitude INTEGER DEFAULT 2,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		board_id TEXT NOT NULL REFERENCES boards(id),
		assigned_agent_id TEXT REFERENCES agents(id),
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'blocked', 'pending_review', 'done')),
		story_points INTEGER DEFAULT 1,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE task_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL REFERENCES tasks(id),
		agent_id TEXT REFERENCES agents(id),
		event_type TEXT NOT NULL,
		old_value TEXT,
		new_value TEXT,
		reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX idx_tasks_status ON tasks(status);
	CREATE INDEX idx_tasks_board ON tasks(board_id);
	CREATE INDEX idx_tasks_agent ON tasks(assigned_agent_id);
	CREATE INDEX idx_task_events_task ON task_events(task_id);
	CREATE INDEX idx_commissions_campaign ON commissions(campaign_id);
	CREATE INDEX idx_boards_commission ON boards(commission_id);
	`

	_, err := database.DB().Exec(schema)
	return err
}

// ShutdownSQLiteStorage properly shuts down SQLite storage components
func ShutdownSQLiteStorage(storageRegistry StorageRegistry) error {
	// The database connection is managed by the repositories
	// For now, this is a no-op, but could be extended for cleanup
	return nil
}