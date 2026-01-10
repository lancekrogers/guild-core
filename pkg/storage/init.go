// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/storage/optimization"
)

// InitializeSQLiteStorageForRegistry initializes SQLite storage and returns configured components
// This function is designed to be called by the registry to avoid circular imports
func InitializeSQLiteStorageForRegistry(ctx context.Context, dbPath string) (StorageRegistry, interface{}, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("Initialize")
	}

	// Create database connection
	database, err := DefaultDatabaseFactory(ctx, dbPath)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("DefaultDatabaseFactory").
			WithDetails("db_path", dbPath)
	}

	// Check context before migrations
	if err := ctx.Err(); err != nil {
		database.Close()
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before migration").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("Migrate")
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
	sessionRepo := DefaultSessionRepositoryFactory(database)
	preferencesRepo := DefaultPreferencesRepositoryFactory(database)

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)
	storageRegistry.RegisterPromptChainRepository(promptChainRepo)
	storageRegistry.RegisterSessionRepository(sessionRepo)
	storageRegistry.RegisterPreferencesRepository(preferencesRepo)

	// Create memory store adapter that implements memory.Store interface
	memoryStoreAdapter := NewMemoryStoreAdapter(database)

	// Register the memory store adapter in the storage registry
	storageRegistry.RegisterMemoryStore(memoryStoreAdapter)

	// Create optimization manager
	// Note: In production, would get metrics registry from main registry
	// For now, passing nil is acceptable as metrics are optional
	optimizationConfig := optimization.DefaultConfig()
	optimizationManager, err := optimization.NewManager(ctx, database.DB(), dbPath, nil, optimizationConfig)
	if err != nil {
		// Log error but don't fail - optimization is optional
		_ = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create optimization manager").
			WithComponent("InitializeSQLiteStorageForRegistry").
			WithOperation("NewOptimizationManager")
		// Continue without optimization
	} else {
		storageRegistry.RegisterOptimizationManager(optimizationManager)
	}

	// Return the storage registry and memory store adapter
	return storageRegistry, memoryStoreAdapter, nil
}

// InitializeSQLiteStorageForTests initializes SQLite storage without migrations for testing
// This creates an in-memory database and manually creates the schema
func InitializeSQLiteStorageForTests(ctx context.Context) (StorageRegistry, interface{}, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("Initialize")
	}

	// Use in-memory SQLite database for tests
	database, err := DefaultDatabaseFactory(ctx, ":memory:")
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create test database").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("NewDatabase")
	}

	// Reset migrations if needed (for in-memory this is a no-op but keeps consistency)
	if err := database.ResetMigrations(ctx); err != nil {
		// Log but don't fail - in-memory databases don't have migration state
		_ = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to reset migrations").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("ResetMigrations")
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
	sessionRepo := DefaultSessionRepositoryFactory(database)
	preferencesRepo := DefaultPreferencesRepositoryFactory(database)

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterBoardRepository(boardRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)
	storageRegistry.RegisterPromptChainRepository(promptChainRepo)
	storageRegistry.RegisterSessionRepository(sessionRepo)
	storageRegistry.RegisterPreferencesRepository(preferencesRepo)

	// Create memory store adapter that implements memory.Store interface
	memoryStoreAdapter := NewMemoryStoreAdapter(database)
	if err := memoryStoreAdapter.EnsureMemoryStoreTable(ctx); err != nil {
		database.Close()
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create memory store table").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("EnsureMemoryStoreTable")
	}

	// Register the memory store adapter in the storage registry
	storageRegistry.RegisterMemoryStore(memoryStoreAdapter)

	// Create optimization manager
	// Note: In production, would get metrics registry from main registry
	// For now, passing nil is acceptable as metrics are optional
	optimizationConfig := optimization.DefaultConfig()
	optimizationManager, err := optimization.NewManager(ctx, database.DB(), ":memory:", nil, optimizationConfig)
	if err != nil {
		// Log error but don't fail - optimization is optional
		_ = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create optimization manager").
			WithComponent("InitializeSQLiteStorageForTests").
			WithOperation("NewOptimizationManager")
		// Continue without optimization
	} else {
		storageRegistry.RegisterOptimizationManager(optimizationManager)
	}

	// Return the storage registry and memory store adapter
	return storageRegistry, memoryStoreAdapter, nil
}

// createTestSchema manually creates the database schema for tests
func createTestSchema(database *Database) error {
	// Note: This schema must match 000001_complete_schema.up.sql exactly
	schema := `
	-- Core campaign management
	CREATE TABLE campaigns (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Commissions (objectives) tied to campaigns
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

	-- Boards for organizing tasks within commissions
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

	-- AI agents configuration
	CREATE TABLE agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL, -- manager, worker, specialist
		provider TEXT,
		model TEXT,
		capabilities JSON,
		tools JSON,
		cost_magnitude INTEGER DEFAULT 2,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Tasks assigned to agents
	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		commission_id TEXT NOT NULL REFERENCES commissions(id),
		board_id TEXT REFERENCES boards(id),
		assigned_agent_id TEXT REFERENCES agents(id),
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'blocked', 'pending_review', 'done')),
		column TEXT NOT NULL DEFAULT 'backlog',
		story_points INTEGER DEFAULT 1,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Task history tracking
	CREATE TABLE task_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL REFERENCES tasks(id),
		agent_id TEXT REFERENCES agents(id),
		event_type TEXT NOT NULL, -- created, assigned, started, completed, blocked
		old_value TEXT,
		new_value TEXT,
		reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Prompt chains for agent conversations
	CREATE TABLE prompt_chains (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		task_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Messages within prompt chains
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

	-- Chat sessions for persistent conversations
	CREATE TABLE chat_sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		campaign_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		metadata JSON,
		FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
	);

	-- Individual messages in chat sessions
	CREATE TABLE chat_messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tool_calls JSON,
		metadata JSON,
		FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);

	-- Bookmarks for important messages
	CREATE TABLE session_bookmarks (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
		FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE
	);

	-- Memory store for general key-value storage
	CREATE TABLE memory_store (
		bucket TEXT NOT NULL,
		key TEXT NOT NULL,
		value BLOB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (bucket, key)
	);

	-- Performance indexes
	CREATE INDEX idx_commissions_campaign ON commissions(campaign_id);
	CREATE INDEX idx_boards_commission ON boards(commission_id);
	CREATE INDEX idx_tasks_status ON tasks(status);
	CREATE INDEX idx_tasks_commission ON tasks(commission_id);
	CREATE INDEX idx_tasks_board ON tasks(board_id);
	CREATE INDEX idx_tasks_agent ON tasks(assigned_agent_id);
	CREATE INDEX idx_task_events_task ON task_events(task_id);
	CREATE INDEX idx_prompt_chains_agent ON prompt_chains(agent_id);
	CREATE INDEX idx_prompt_chains_task ON prompt_chains(task_id);
	CREATE INDEX idx_prompt_chain_messages_chain ON prompt_chain_messages(chain_id);
	CREATE INDEX idx_prompt_chain_messages_timestamp ON prompt_chain_messages(timestamp);
	CREATE INDEX idx_chat_sessions_campaign ON chat_sessions(campaign_id);
	CREATE INDEX idx_chat_sessions_updated ON chat_sessions(updated_at);
	CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
	CREATE INDEX idx_chat_messages_created ON chat_messages(created_at);
	CREATE INDEX idx_chat_messages_role ON chat_messages(role);
	CREATE INDEX idx_session_bookmarks_session ON session_bookmarks(session_id);
	CREATE INDEX idx_session_bookmarks_message ON session_bookmarks(message_id);
	CREATE INDEX idx_session_bookmarks_created ON session_bookmarks(created_at);

	-- Preferences table for hierarchical preference management
	CREATE TABLE preferences (
		id TEXT PRIMARY KEY,
		scope TEXT NOT NULL CHECK (scope IN ('system', 'user', 'campaign', 'guild', 'agent')),
		scope_id TEXT,
		key TEXT NOT NULL,
		value JSON NOT NULL,
		version INTEGER DEFAULT 1,
		metadata JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_preference UNIQUE(scope, scope_id, key)
	);

	-- Preference inheritance relationships
	CREATE TABLE preference_inheritance (
		id TEXT PRIMARY KEY,
		child_scope TEXT NOT NULL,
		child_scope_id TEXT,
		parent_scope TEXT NOT NULL,
		parent_scope_id TEXT,
		priority INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_inheritance UNIQUE(child_scope, child_scope_id, parent_scope, parent_scope_id)
	);

	-- Indexes for preferences
	CREATE INDEX idx_preferences_scope ON preferences(scope);
	CREATE INDEX idx_preferences_scope_id ON preferences(scope_id);
	CREATE INDEX idx_preferences_key ON preferences(key);
	CREATE INDEX idx_preferences_updated_at ON preferences(updated_at);
	CREATE INDEX idx_preference_inheritance_child ON preference_inheritance(child_scope, child_scope_id);
	CREATE INDEX idx_preference_inheritance_parent ON preference_inheritance(parent_scope, parent_scope_id);

	-- Reasoning blocks table (from migration 7)
	CREATE TABLE reasoning_blocks (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		session_id TEXT,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		confidence REAL NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tokens_used INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0,
		reasoning_tokens INTEGER NOT NULL DEFAULT 0,
		content_tokens INTEGER NOT NULL DEFAULT 0,
		metadata JSON
	);

	CREATE INDEX idx_reasoning_blocks_agent_timestamp ON reasoning_blocks(agent_id, timestamp DESC);
	CREATE INDEX idx_reasoning_blocks_type_confidence ON reasoning_blocks(type, confidence);
	CREATE INDEX idx_reasoning_blocks_session_id ON reasoning_blocks(session_id);

	-- Trigger to update preferences.updated_at
	CREATE TRIGGER update_preferences_timestamp
	AFTER UPDATE ON preferences
	BEGIN
		UPDATE preferences SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

	-- Trigger to update chat_sessions.updated_at on new messages
	CREATE TRIGGER update_session_timestamp
	AFTER INSERT ON chat_messages
	BEGIN
		UPDATE chat_sessions 
		SET updated_at = CURRENT_TIMESTAMP 
		WHERE id = NEW.session_id;
	END;
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
