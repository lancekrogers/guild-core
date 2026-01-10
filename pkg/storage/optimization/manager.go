// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"database/sql"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// Manager coordinates all storage optimization components
type Manager struct {
	db                 *sql.DB
	dbPath             string
	metrics            *observability.MetricsRegistry
	queryOptimizer     *QueryOptimizer
	compressionManager *CompressionManager
	performanceMonitor *PerformanceMonitor
	maintenanceManager *MaintenanceManager
	config             Config
}

// Config configures the optimization manager
type Config struct {
	QueryOptimization QueryOptimizerConfig
	Compression       CompressionConfig
	Monitoring        MonitoringConfig
	Maintenance       MaintenanceConfig
	Enabled           bool
}

// DefaultConfig returns default optimization configuration
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		QueryOptimization: DefaultQueryOptimizerConfig(),
		Compression:       DefaultCompressionConfig(),
		Monitoring:        DefaultMonitoringConfig(),
		Maintenance:       DefaultMaintenanceConfig(),
	}
}

// QueryOptimizerConfig configures query optimization (extends QueryOptimizer)
type QueryOptimizerConfig struct {
	CacheConfig        QueryCacheConfig
	SlowQueryThreshold time.Duration
	PreparedStatements bool
}

// DefaultQueryOptimizerConfig returns default query optimizer settings
func DefaultQueryOptimizerConfig() QueryOptimizerConfig {
	return QueryOptimizerConfig{
		CacheConfig:        DefaultQueryCacheConfig(),
		SlowQueryThreshold: 100 * time.Millisecond,
		PreparedStatements: true,
	}
}

// NewManager creates a new optimization manager
func NewManager(ctx context.Context, db *sql.DB, dbPath string, metrics *observability.MetricsRegistry, config Config) (*Manager, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("OptimizationManager").
			WithOperation("NewManager")
	}

	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("OptimizationManager")
	}

	if dbPath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database path is required", nil).
			WithComponent("OptimizationManager")
	}

	manager := &Manager{
		db:      db,
		dbPath:  dbPath,
		metrics: metrics,
		config:  config,
	}

	if !config.Enabled {
		return manager, nil
	}

	// Initialize query optimizer
	queryOptimizer, err := NewQueryOptimizer(db, metrics)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create query optimizer").
			WithComponent("OptimizationManager")
	}
	manager.queryOptimizer = queryOptimizer

	// Initialize compression manager
	compressionManager, err := NewCompressionManager(db, metrics, config.Compression)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create compression manager").
			WithComponent("OptimizationManager")
	}
	manager.compressionManager = compressionManager

	// Initialize performance monitor
	performanceMonitor, err := NewPerformanceMonitor(db, metrics, queryOptimizer, config.Monitoring)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create performance monitor").
			WithComponent("OptimizationManager")
	}
	manager.performanceMonitor = performanceMonitor

	// Initialize maintenance manager
	maintenanceManager, err := NewMaintenanceManager(db, dbPath, metrics, config.Maintenance)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create maintenance manager").
			WithComponent("OptimizationManager")
	}
	manager.maintenanceManager = maintenanceManager

	return manager, nil
}

// QueryOptimizer returns the query optimizer
func (m *Manager) QueryOptimizer() *QueryOptimizer {
	return m.queryOptimizer
}

// CompressionManager returns the compression manager
func (m *Manager) CompressionManager() *CompressionManager {
	return m.compressionManager
}

// PerformanceMonitor returns the performance monitor
func (m *Manager) PerformanceMonitor() *PerformanceMonitor {
	return m.performanceMonitor
}

// MaintenanceManager returns the maintenance manager
func (m *Manager) MaintenanceManager() *MaintenanceManager {
	return m.maintenanceManager
}

// OptimizeQuery analyzes and optimizes a query
func (m *Manager) OptimizeQuery(ctx context.Context, query string) (*QueryPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("OptimizationManager").
			WithOperation("OptimizeQuery")
	}

	if m.queryOptimizer == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "query optimizer not initialized", nil).
			WithComponent("OptimizationManager").
			WithOperation("OptimizeQuery")
	}

	return m.queryOptimizer.AnalyzeQuery(ctx, query)
}

// RecordQueryExecution tracks query execution for monitoring
func (m *Manager) RecordQueryExecution(query string, duration time.Duration, rowsAffected int64, err error) {
	if m.performanceMonitor != nil {
		m.performanceMonitor.RecordQuery(query, duration, rowsAffected, err)
	}

	if m.queryOptimizer != nil {
		m.queryOptimizer.TrackQueryExecution(query, duration)
	}
}

// GetOptimizationReport generates a comprehensive optimization report
func (m *Manager) GetOptimizationReport(ctx context.Context) (*OptimizationReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("OptimizationManager").
			WithOperation("GetOptimizationReport")
	}

	report := &OptimizationReport{
		GeneratedAt: time.Now(),
	}

	// Get database stats
	if m.performanceMonitor != nil {
		stats, err := m.performanceMonitor.CollectDatabaseStats(ctx)
		if err == nil {
			report.DatabaseStats = stats
		}

		report.SlowQueries = m.performanceMonitor.GetSlowQueries(20)
		report.GrowthMetrics = m.performanceMonitor.GetStorageGrowthMetrics()
	}

	// Get query statistics
	if m.queryOptimizer != nil {
		report.QueryStats = m.queryOptimizer.GetQueryStats()

		// Get index optimization suggestions
		suggestions, err := m.queryOptimizer.OptimizeIndexes(ctx)
		if err == nil {
			report.IndexSuggestions = suggestions
		}

		// Get cache stats
		if m.queryOptimizer.cache != nil {
			cacheStats := m.queryOptimizer.cache.Stats()
			report.CacheStats = &cacheStats
		}
	}

	// Get compression stats
	if m.compressionManager != nil {
		compressionStats, err := m.compressionManager.GetCompressionStats(ctx)
		if err == nil {
			report.CompressionStats = compressionStats
		}
	}

	// Get maintenance schedule
	if m.maintenanceManager != nil {
		schedule := m.maintenanceManager.GetMaintenanceSchedule()
		report.MaintenanceSchedule = &schedule
		report.MaintenanceHistory = m.maintenanceManager.GetMaintenanceHistory(10)
	}

	return report, nil
}

// OptimizationReport contains comprehensive optimization information
type OptimizationReport struct {
	GeneratedAt         time.Time
	DatabaseStats       *DatabaseStats
	QueryStats          map[string]*QueryStats
	SlowQueries         []SlowQuery
	GrowthMetrics       map[string]GrowthMetric
	IndexSuggestions    []string
	CacheStats          *CacheStats
	CompressionStats    *CompressionStats
	MaintenanceSchedule *MaintenanceSchedule
	MaintenanceHistory  []MaintenanceEvent
}

// RunOptimization performs a full optimization cycle
func (m *Manager) RunOptimization(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("OptimizationManager").
			WithOperation("RunOptimization")
	}

	// Update statistics first
	if m.maintenanceManager != nil {
		if err := m.maintenanceManager.UpdateStatistics(ctx); err != nil {
			// Log but continue
			_ = err
		}
	}

	// Analyze indexes
	if m.queryOptimizer != nil {
		if _, err := m.queryOptimizer.OptimizeIndexes(ctx); err != nil {
			// Log but continue
			_ = err
		}
	}

	// Archive old data
	tables := []string{"chat_messages", "agent_memories", "session_logs"}
	if m.compressionManager != nil && m.config.Compression.ArchiveAfterDays > 0 {
		for _, table := range tables {
			if err := m.compressionManager.ArchiveOldData(ctx, table); err != nil {
				// Log but continue
				_ = err
			}
		}
	}

	// Optimize BLOB storage
	if m.compressionManager != nil && m.config.Compression.CompressBLOBs {
		blobTables := map[string]string{
			"agent_memories": "content",
			"embeddings":     "vector",
		}

		for table, column := range blobTables {
			if err := m.compressionManager.OptimizeBLOBStorage(ctx, table, column); err != nil {
				// Log but continue
				_ = err
			}
		}
	}

	// Run vacuum if needed
	schedule := m.maintenanceManager.GetMaintenanceSchedule()
	if schedule.IsOverdue && m.maintenanceManager != nil {
		if err := m.maintenanceManager.RunVacuum(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "vacuum failed during optimization").
				WithComponent("OptimizationManager").
				WithOperation("RunOptimization")
		}
	}

	return nil
}

// Close shuts down all optimization components
func (m *Manager) Close() error {
	var errors []error

	if m.queryOptimizer != nil {
		if err := m.queryOptimizer.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	// Other components don't have explicit close methods
	// but we ensure clean shutdown

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to close optimization components", errors[0]).
			WithComponent("OptimizationManager").
			WithOperation("Close").
			WithDetails("errors", len(errors))
	}

	return nil
}
