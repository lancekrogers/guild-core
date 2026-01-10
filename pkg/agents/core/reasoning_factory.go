// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"path/filepath"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/storage"
)

// ReasoningSystemConfig configures the complete reasoning system
type ReasoningSystemConfig struct {
	ExtractorConfig ReasoningConfig
	StorageConfig   ReasoningStorageConfig
	EnableAnalytics bool
	DatabasePath    string // If empty, uses default .guild/memory.db
}

// DefaultReasoningSystemConfig returns production-ready defaults
func DefaultReasoningSystemConfig() ReasoningSystemConfig {
	return ReasoningSystemConfig{
		ExtractorConfig: DefaultReasoningConfig(),
		StorageConfig:   DefaultReasoningStorageConfig(),
		EnableAnalytics: true,
		DatabasePath:    "", // Use default
	}
}

// DefaultReasoningStorageConfig returns production storage defaults
func DefaultReasoningStorageConfig() ReasoningStorageConfig {
	return ReasoningStorageConfig{
		RetentionDays:     90,
		MaxChainsPerQuery: 1000,
		EnableCompression: true,
		StorageBackend:    "sqlite",
	}
}

// ReasoningSystem encapsulates all reasoning components
type ReasoningSystem struct {
	Extractor *ReasoningExtractor
	Storage   ReasoningStorage
	Analyzer  ReasoningAnalyzer
	database  *storage.Database // Keep reference for cleanup
}

// NewReasoningSystem creates a complete reasoning system with SQLite storage
func NewReasoningSystem(ctx context.Context, config ReasoningSystemConfig) (*ReasoningSystem, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning_system").
			WithOperation("NewReasoningSystem")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_system").
		WithOperation("NewReasoningSystem")

	// Validate configuration
	if err := config.ExtractorConfig.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid extractor configuration").
			WithComponent("reasoning_system").
			WithOperation("NewReasoningSystem")
	}

	if err := config.StorageConfig.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid storage configuration").
			WithComponent("reasoning_system").
			WithOperation("NewReasoningSystem")
	}

	// Create reasoning extractor
	extractor, err := NewReasoningExtractor(config.ExtractorConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create reasoning extractor").
			WithComponent("reasoning_system").
			WithOperation("NewReasoningSystem")
	}

	// Create storage based on backend
	var reasoningStorage ReasoningStorage
	var database *storage.Database

	switch config.StorageConfig.StorageBackend {
	case "sqlite":
		// Determine database path
		dbPath := config.DatabasePath
		if dbPath == "" {
			// Use default campaign database
			dbPath = filepath.Join(".campaign", "memory.db")
		}

		// Create or open database
		database, err = storage.DefaultDatabaseFactory(ctx, dbPath)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database").
				WithComponent("reasoning_system").
				WithOperation("NewReasoningSystem").
				WithDetails("db_path", dbPath)
		}

		// Run migrations
		if err := database.Migrate(ctx); err != nil {
			database.Close()
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to run migrations").
				WithComponent("reasoning_system").
				WithOperation("NewReasoningSystem")
		}

		// Create SQLite storage
		reasoningStorage, err = NewSQLiteReasoningStorage(ctx, database, config.StorageConfig)
		if err != nil {
			database.Close()
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create SQLite reasoning storage").
				WithComponent("reasoning_system").
				WithOperation("NewReasoningSystem")
		}

		logger.InfoContext(ctx, "Created SQLite reasoning storage",
			"db_path", dbPath,
			"retention_days", config.StorageConfig.RetentionDays)

	case "memory":
		// Create in-memory storage (for testing)
		reasoningStorage, err = NewMemoryReasoningStorage(config.StorageConfig)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create memory reasoning storage").
				WithComponent("reasoning_system").
				WithOperation("NewReasoningSystem")
		}

		logger.InfoContext(ctx, "Created in-memory reasoning storage")

	default:
		return nil, gerror.Newf(gerror.ErrCodeValidation, "unsupported storage backend: %s", config.StorageConfig.StorageBackend).
			WithComponent("reasoning_system").
			WithOperation("NewReasoningSystem")
	}

	// Create analyzer if enabled
	var analyzer ReasoningAnalyzer
	if config.EnableAnalytics {
		analyzer, err = NewDefaultReasoningAnalyzer(reasoningStorage)
		if err != nil {
			// Don't fail system creation if analytics fails
			logger.WithError(err).WarnContext(ctx, "Failed to create reasoning analyzer, analytics disabled")
		} else {
			logger.InfoContext(ctx, "Created reasoning analyzer")
		}
	}

	system := &ReasoningSystem{
		Extractor: extractor,
		Storage:   reasoningStorage,
		Analyzer:  analyzer,
		database:  database,
	}

	logger.InfoContext(ctx, "Reasoning system initialized successfully",
		"storage_backend", config.StorageConfig.StorageBackend,
		"caching_enabled", config.ExtractorConfig.EnableCaching,
		"analytics_enabled", config.EnableAnalytics)

	return system, nil
}

// Close cleans up all resources
func (rs *ReasoningSystem) Close(ctx context.Context) error {
	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_system").
		WithOperation("Close")

	var errors []error

	// Close storage
	if rs.Storage != nil {
		if err := rs.Storage.Close(ctx); err != nil {
			errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close reasoning storage"))
		}
	}

	// Close database if we own it
	if rs.database != nil {
		if err := rs.database.Close(); err != nil {
			errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close database"))
		}
	}

	if len(errors) > 0 {
		return gerror.Newf(gerror.ErrCodeInternal, "failed to close reasoning system: %d errors", len(errors)).
			WithComponent("reasoning_system").
			WithOperation("Close")
	}

	logger.InfoContext(ctx, "Reasoning system closed successfully")
	return nil
}

// EnhanceAgent adds reasoning capabilities to an agent
func (rs *ReasoningSystem) EnhanceAgent(agent Agent) error {
	// Type assert to get worker agent
	var workerAgent *WorkerAgent

	switch a := agent.(type) {
	case *WorkerAgent:
		workerAgent = a
	case *ManagerAgent:
		workerAgent = &a.WorkerAgent
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "unsupported agent type: %T", agent).
			WithComponent("reasoning_system").
			WithOperation("EnhanceAgent")
	}

	// Set extractor and storage
	workerAgent.SetReasoningExtractor(rs.Extractor)
	workerAgent.SetReasoningStorage(rs.Storage)

	return nil
}

// GetInsights generates insights for an agent
func (rs *ReasoningSystem) GetInsights(ctx context.Context, agentID string) ([]string, error) {
	if rs.Analyzer == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "analytics not enabled", nil).
			WithComponent("reasoning_system").
			WithOperation("GetInsights")
	}

	// Get recent stats
	stats, err := rs.Storage.GetStats(ctx, agentID, time.Time{}, time.Now())
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get reasoning stats").
			WithComponent("reasoning_system").
			WithOperation("GetInsights").
			WithDetails("agent_id", agentID)
	}

	// Generate insights
	insights, err := rs.Analyzer.GenerateInsights(ctx, stats)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate insights").
			WithComponent("reasoning_system").
			WithOperation("GetInsights").
			WithDetails("agent_id", agentID)
	}

	return insights, nil
}

// StartMaintenance starts background maintenance tasks
func (rs *ReasoningSystem) StartMaintenance(ctx context.Context) {
	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_system").
		WithOperation("StartMaintenance")

	// Start retention cleanup
	go rs.runRetentionCleanup(ctx)

	// Start analytics aggregation
	if rs.Analyzer != nil {
		go rs.runAnalyticsAggregation(ctx)
	}

	logger.InfoContext(ctx, "Started reasoning system maintenance tasks")
}

func (rs *ReasoningSystem) runRetentionCleanup(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_system").
		WithOperation("runRetentionCleanup")

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "Stopping retention cleanup")
			return
		case <-ticker.C:
			// Calculate cutoff time based on retention config
			config, ok := rs.Storage.(*SQLiteReasoningStorage)
			if !ok {
				continue
			}

			cutoff := time.Now().AddDate(0, 0, -config.config.RetentionDays)
			deleted, err := rs.Storage.Delete(ctx, cutoff)

			if err != nil {
				logger.WithError(err).ErrorContext(ctx, "Failed to clean up old reasoning chains")
			} else {
				logger.InfoContext(ctx, "Cleaned up old reasoning chains",
					"deleted_count", deleted,
					"cutoff_date", cutoff)
			}
		}
	}
}

func (rs *ReasoningSystem) runAnalyticsAggregation(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	logger := observability.GetLogger(ctx).
		WithComponent("reasoning_system").
		WithOperation("runAnalyticsAggregation")

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "Stopping analytics aggregation")
			return
		case <-ticker.C:
			// Aggregate patterns for all agents
			query := &ReasoningQuery{
				StartTime: time.Now().Add(-24 * time.Hour),
				Limit:     1000,
			}

			chains, err := rs.Storage.Query(ctx, query)
			if err != nil {
				logger.WithError(err).ErrorContext(ctx, "Failed to query recent chains")
				continue
			}

			if len(chains) == 0 {
				continue
			}

			// Identify patterns
			patterns, err := rs.Analyzer.IdentifyPatterns(ctx, chains)
			if err != nil {
				logger.WithError(err).ErrorContext(ctx, "Failed to identify patterns")
				continue
			}

			// Update pattern storage
			for _, pattern := range patterns {
				if err := rs.Storage.UpdatePattern(ctx, pattern); err != nil {
					logger.WithError(err).ErrorContext(ctx, "Failed to update pattern",
						"pattern_id", pattern.ID)
				}
			}

			logger.InfoContext(ctx, "Completed analytics aggregation",
				"chains_analyzed", len(chains),
				"patterns_found", len(patterns))
		}
	}
}
