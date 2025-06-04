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
	agentRepo := NewSQLiteAgentRepository(database)

	storageRegistry.RegisterTaskRepository(taskRepo)
	storageRegistry.RegisterCampaignRepository(campaignRepo)
	storageRegistry.RegisterCommissionRepository(commissionRepo)
	storageRegistry.RegisterAgentRepository(agentRepo)

	// Create SQLite store adapter for memory.Store interface compatibility
	storeAdapter := NewSQLiteStoreAdapter(storageRegistry)

	// Return both the storage registry and the memory store adapter
	return storageRegistry, storeAdapter, nil
}

// ShutdownSQLiteStorage properly shuts down SQLite storage components
func ShutdownSQLiteStorage(storageRegistry StorageRegistry) error {
	// The database connection is managed by the repositories
	// For now, this is a no-op, but could be extended for cleanup
	return nil
}