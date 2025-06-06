package storage

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// InitializeSQLiteStorageForRegistry initializes SQLite storage and returns configured components
// This function is designed to be called by the registry to avoid circular imports
func InitializeSQLiteStorageForRegistry(ctx context.Context, dbPath string) (StorageRegistry, interface{}, error) {
	// Create database connection
	database, err := DefaultDatabaseFactory(ctx, dbPath)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("DefaultDatabaseFactory").
			WithDetails("db_path", dbPath)
	}

	// Run migrations
	if err := database.Migrate(ctx); err != nil {
		database.Close()
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to run migrations").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("Migrate")
	}

	// Create storage registry
	storageRegistry := DefaultStorageRegistryFactory()

	// Create and register repositories
	taskRepo := DefaultTaskRepositoryFactory(database)
	campaignRepo := DefaultCampaignRepositoryFactory(database)
	commissionRepo := DefaultCommissionRepositoryFactory(database)
	boardRepo := DefaultBoardRepositoryFactory(database)
	agentRepo := DefaultAgentRepositoryFactory(database)
	promptChainRepo := DefaultPromptChainRepositoryFactory(database.DB())

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)
	storageRegistry.RegisterPromptChainRepository(promptChainRepo)

	// Return the storage registry only - no more adapter needed
	return storageRegistry, nil, nil
}

// InitializeSQLiteStorageForTests initializes SQLite storage without migrations for testing
// This creates an in-memory database and manually creates the schema
func InitializeSQLiteStorageForTests(ctx context.Context) (StorageRegistry, interface{}, error) {
	// Use in-memory SQLite database for tests
	database, err := NewDatabase(ctx, ":memory:")
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test database").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("NewDatabase")
	}

	// Manually create schema instead of running migrations
	if err := createTestSchema(database); err != nil {
		database.Close()
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test schema").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("createTestSchema")
	}

	// Create storage registry
	storageRegistry := DefaultStorageRegistryFactory()

	// Create and register repositories
	taskRepo := DefaultTaskRepositoryFactory(database)
	campaignRepo := DefaultCampaignRepositoryFactory(database)
	commissionRepo := DefaultCommissionRepositoryFactory(database)
	boardRepo := DefaultBoardRepositoryFactory(database)
	agentRepo := DefaultAgentRepositoryFactory(database)
	promptChainRepo := DefaultPromptChainRepositoryFactory(database.DB())

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)
	storageRegistry.RegisterPromptChainRepository(promptChainRepo)

	// Return the storage registry only - no more adapter needed
	return storageRegistry, nil, nil
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
		commission_id TEXT NOT NULL REFERENCES commissions(id),
		assigned_agent_id TEXT REFERENCES agents(id),
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'blocked', 'pending_review', 'done')),
		column TEXT NOT NULL DEFAULT 'backlog',
		story_points INTEGER DEFAULT 1,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		board_id TEXT REFERENCES boards(id)
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
	CREATE INDEX idx_tasks_commission ON tasks(commission_id);
	CREATE INDEX idx_tasks_board ON tasks(board_id);
	CREATE INDEX idx_tasks_agent ON tasks(assigned_agent_id);
	CREATE INDEX idx_task_events_task ON task_events(task_id);
	CREATE INDEX idx_commissions_campaign ON commissions(campaign_id);
	CREATE INDEX idx_boards_commission ON boards(commission_id);

	-- Add prompt chains table
	CREATE TABLE prompt_chains (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		task_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Add prompt chain messages table  
	CREATE TABLE prompt_chain_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chain_id TEXT NOT NULL REFERENCES prompt_chains(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
		content TEXT NOT NULL,
		name TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		token_usage INTEGER DEFAULT 0,
		FOREIGN KEY (chain_id) REFERENCES prompt_chains(id)
	);

	-- Create indexes for efficient lookups
	CREATE INDEX idx_prompt_chains_agent ON prompt_chains(agent_id);
	CREATE INDEX idx_prompt_chains_task ON prompt_chains(task_id);
	CREATE INDEX idx_prompt_chain_messages_chain ON prompt_chain_messages(chain_id);
	CREATE INDEX idx_prompt_chain_messages_timestamp ON prompt_chain_messages(timestamp);
	`

	_, err := database.DB().Exec(schema)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test schema").
			WithComponent("createTestSchema").
			WithOperation("Exec")
	}
	return nil
}

// ShutdownSQLiteStorage properly shuts down SQLite storage components
func ShutdownSQLiteStorage(storageRegistry StorageRegistry) error {
	// The database connection is managed by the repositories
	// For now, this is a no-op, but could be extended for cleanup
	return nil
}