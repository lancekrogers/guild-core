// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// PerformanceMonitor tracks database performance metrics
type PerformanceMonitor struct {
	db             *sql.DB
	metrics        *observability.MetricsRegistry
	config         MonitoringConfig
	queryOptimizer *QueryOptimizer

	mu              sync.RWMutex
	slowQueries     []SlowQuery
	dbStats         DatabaseStats
	growthMetrics   map[string]GrowthMetric
	lastCollectedAt time.Time
}

// MonitoringConfig configures performance monitoring
type MonitoringConfig struct {
	Enabled            bool
	CollectionInterval time.Duration
	SlowQueryThreshold time.Duration
	MaxSlowQueries     int
	EnableAlerts       bool
	AlertThresholds    AlertThresholds
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Enabled:            true,
		CollectionInterval: 1 * time.Minute,
		SlowQueryThreshold: 100 * time.Millisecond,
		MaxSlowQueries:     100,
		EnableAlerts:       true,
		AlertThresholds: AlertThresholds{
			SlowQueryRate:     0.1,  // 10% of queries are slow
			ErrorRate:         0.05, // 5% error rate
			StorageGrowthRate: 0.2,  // 20% growth per day
			ConnectionLimit:   0.9,  // 90% of max connections
		},
	}
}

// AlertThresholds defines thresholds for performance alerts
type AlertThresholds struct {
	SlowQueryRate     float64
	ErrorRate         float64
	StorageGrowthRate float64
	ConnectionLimit   float64
}

// SlowQuery represents a slow query
type SlowQuery struct {
	Query        string
	Duration     time.Duration
	ExecutedAt   time.Time
	RowsAffected int64
	Error        error
}

// DatabaseStats contains database performance statistics
type DatabaseStats struct {
	TotalQueries      int64
	SlowQueries       int64
	ErrorCount        int64
	AvgQueryTime      time.Duration
	ActiveConnections int
	DatabaseSize      int64
	TableSizes        map[string]int64
	IndexSizes        map[string]int64
	CacheHitRate      float64
}

// GrowthMetric tracks storage growth over time
type GrowthMetric struct {
	TableName        string
	SizeBytes        int64
	RowCount         int64
	Timestamp        time.Time
	DailyGrowthRate  float64
	WeeklyGrowthRate float64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(db *sql.DB, metrics *observability.MetricsRegistry, queryOptimizer *QueryOptimizer, config MonitoringConfig) (*PerformanceMonitor, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("PerformanceMonitor")
	}

	monitor := &PerformanceMonitor{
		db:             db,
		metrics:        metrics,
		config:         config,
		queryOptimizer: queryOptimizer,
		slowQueries:    make([]SlowQuery, 0, config.MaxSlowQueries),
		growthMetrics:  make(map[string]GrowthMetric),
	}

	if config.Enabled {
		go monitor.collectionLoop()
	}

	return monitor, nil
}

// RecordQuery records query execution metrics
func (p *PerformanceMonitor) RecordQuery(query string, duration time.Duration, rowsAffected int64, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.dbStats.TotalQueries++

	if err != nil {
		p.dbStats.ErrorCount++
	}

	// Update average query time
	p.dbStats.AvgQueryTime = (p.dbStats.AvgQueryTime*time.Duration(p.dbStats.TotalQueries-1) + duration) / time.Duration(p.dbStats.TotalQueries)

	// Track slow queries
	if duration > p.config.SlowQueryThreshold {
		p.dbStats.SlowQueries++

		slowQuery := SlowQuery{
			Query:        query,
			Duration:     duration,
			ExecutedAt:   time.Now(),
			RowsAffected: rowsAffected,
			Error:        err,
		}

		// Add to slow query list (with size limit)
		if len(p.slowQueries) >= p.config.MaxSlowQueries {
			p.slowQueries = p.slowQueries[1:]
		}
		p.slowQueries = append(p.slowQueries, slowQuery)

		// Alert if slow query rate is too high
		if p.config.EnableAlerts {
			slowRate := float64(p.dbStats.SlowQueries) / float64(p.dbStats.TotalQueries)
			if slowRate > p.config.AlertThresholds.SlowQueryRate {
				p.sendAlert("high_slow_query_rate", fmt.Sprintf("Slow query rate: %.2f%%", slowRate*100))
			}
		}
	}

	// Pass to query optimizer for analysis
	if p.queryOptimizer != nil {
		p.queryOptimizer.TrackQueryExecution(query, duration)
	}

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = err
}

// CollectDatabaseStats collects current database statistics
func (p *PerformanceMonitor) CollectDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PerformanceMonitor").
			WithOperation("CollectDatabaseStats")
	}

	stats := &DatabaseStats{
		TableSizes: make(map[string]int64),
		IndexSizes: make(map[string]int64),
	}

	// Get database size
	var pageCount, pageSize int64
	if err := p.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err == nil {
		if err := p.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err == nil {
			stats.DatabaseSize = pageCount * pageSize
		}
	}

	// Get connection stats
	dbStats := p.db.Stats()
	stats.ActiveConnections = dbStats.OpenConnections

	// Get table sizes
	query := `
		SELECT name, SUM(pgsize) as size
		FROM dbstat
		WHERE name IN (SELECT name FROM sqlite_master WHERE type='table')
		GROUP BY name
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var name string
			var size int64
			if err := rows.Scan(&name, &size); err == nil {
				stats.TableSizes[name] = size
			}
		}
	}

	// Get index sizes
	indexQuery := `
		SELECT name, SUM(pgsize) as size
		FROM dbstat
		WHERE name IN (SELECT name FROM sqlite_master WHERE type='index')
		GROUP BY name
	`

	rows, err = p.db.QueryContext(ctx, indexQuery)
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var name string
			var size int64
			if err := rows.Scan(&name, &size); err == nil {
				stats.IndexSizes[name] = size
			}
		}
	}

	// Calculate cache hit rate from query cache if available
	if p.queryOptimizer != nil && p.queryOptimizer.cache != nil {
		cacheStats := p.queryOptimizer.cache.Stats()
		stats.CacheHitRate = cacheStats.HitRate
	}

	// Copy accumulated stats
	p.mu.RLock()
	stats.TotalQueries = p.dbStats.TotalQueries
	stats.SlowQueries = p.dbStats.SlowQueries
	stats.ErrorCount = p.dbStats.ErrorCount
	stats.AvgQueryTime = p.dbStats.AvgQueryTime
	p.mu.RUnlock()

	return stats, nil
}

// TrackStorageGrowth monitors storage growth patterns
func (p *PerformanceMonitor) TrackStorageGrowth(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PerformanceMonitor").
			WithOperation("TrackStorageGrowth")
	}

	stats, err := p.CollectDatabaseStats(ctx)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	for tableName, size := range stats.TableSizes {
		metric, exists := p.growthMetrics[tableName]
		if !exists {
			metric = GrowthMetric{
				TableName: tableName,
				Timestamp: now,
			}
		}

		// Calculate growth rates
		if exists && metric.SizeBytes > 0 {
			duration := now.Sub(metric.Timestamp)
			if duration > time.Hour {
				growthBytes := size - metric.SizeBytes
				dailyRate := float64(growthBytes) / (duration.Hours() / 24) / float64(metric.SizeBytes)
				metric.DailyGrowthRate = dailyRate

				// Alert on high growth rate
				if p.config.EnableAlerts && dailyRate > p.config.AlertThresholds.StorageGrowthRate {
					p.sendAlert("high_storage_growth", fmt.Sprintf("Table %s growing at %.2f%% per day", tableName, dailyRate*100))
				}
			}
		}

		metric.SizeBytes = size
		metric.Timestamp = now
		p.growthMetrics[tableName] = metric
	}

	return nil
}

// GetSlowQueries returns recent slow queries
func (p *PerformanceMonitor) GetSlowQueries(limit int) []SlowQuery {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 || limit > len(p.slowQueries) {
		limit = len(p.slowQueries)
	}

	// Return most recent queries
	start := len(p.slowQueries) - limit
	if start < 0 {
		start = 0
	}

	result := make([]SlowQuery, limit)
	copy(result, p.slowQueries[start:])

	return result
}

// GetStorageGrowthMetrics returns storage growth metrics
func (p *PerformanceMonitor) GetStorageGrowthMetrics() map[string]GrowthMetric {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]GrowthMetric)
	for k, v := range p.growthMetrics {
		result[k] = v
	}

	return result
}

// collectionLoop periodically collects metrics
func (p *PerformanceMonitor) collectionLoop() {
	ticker := time.NewTicker(p.config.CollectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Collect stats
		if _, err := p.CollectDatabaseStats(ctx); err != nil {
			// Log error but continue
			_ = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to collect database stats").
				WithComponent("PerformanceMonitor").
				WithOperation("collectionLoop")
		}

		// Track growth
		if err := p.TrackStorageGrowth(ctx); err != nil {
			// Log error but continue
			_ = gerror.Wrap(err, gerror.ErrCodeInternal, "failed to track storage growth").
				WithComponent("PerformanceMonitor").
				WithOperation("collectionLoop")
		}

		cancel()
	}
}

// sendAlert sends performance alerts
func (p *PerformanceMonitor) sendAlert(alertType, message string) {
	// TODO: Implement alerting when MetricsRegistry supports generic methods
	_ = alertType
	_ = message
}

// ExportMetrics exports monitoring data for analysis
func (p *PerformanceMonitor) ExportMetrics(ctx context.Context) (*MonitoringExport, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PerformanceMonitor").
			WithOperation("ExportMetrics")
	}

	stats, err := p.CollectDatabaseStats(ctx)
	if err != nil {
		return nil, err
	}

	export := &MonitoringExport{
		Timestamp:     time.Now(),
		DatabaseStats: stats,
		SlowQueries:   p.GetSlowQueries(50),
		GrowthMetrics: p.GetStorageGrowthMetrics(),
	}

	if p.queryOptimizer != nil {
		export.QueryStats = p.queryOptimizer.GetQueryStats()
	}

	return export, nil
}

// MonitoringExport contains exported monitoring data
type MonitoringExport struct {
	Timestamp     time.Time
	DatabaseStats *DatabaseStats
	SlowQueries   []SlowQuery
	GrowthMetrics map[string]GrowthMetric
	QueryStats    map[string]*QueryStats
}
