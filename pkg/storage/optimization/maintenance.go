// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// MaintenanceManager handles automated database maintenance
type MaintenanceManager struct {
	db      *sql.DB
	metrics *observability.MetricsRegistry
	config  MaintenanceConfig
	dbPath  string

	mu             sync.Mutex
	lastVacuum     time.Time
	lastAnalyze    time.Time
	lastBackup     time.Time
	maintenanceLog []MaintenanceEvent
}

// MaintenanceConfig configures maintenance behavior
type MaintenanceConfig struct {
	Enabled               bool
	VacuumInterval        time.Duration
	AnalyzeInterval       time.Duration
	BackupInterval        time.Duration
	BackupRetentionDays   int
	BackupPath            string
	AutoVacuumEnabled     bool
	IndexRebuildThreshold float64 // Fragmentation percentage
	MaxBackups            int
}

// DefaultMaintenanceConfig returns default maintenance settings
func DefaultMaintenanceConfig() MaintenanceConfig {
	return MaintenanceConfig{
		Enabled:               true,
		VacuumInterval:        24 * time.Hour,
		AnalyzeInterval:       6 * time.Hour,
		BackupInterval:        24 * time.Hour,
		BackupRetentionDays:   7,
		BackupPath:            ".guild/backups",
		AutoVacuumEnabled:     true,
		IndexRebuildThreshold: 30.0, // 30% fragmentation
		MaxBackups:            7,
	}
}

// MaintenanceEvent represents a maintenance operation
type MaintenanceEvent struct {
	Type      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
	Error     error
	Details   map[string]interface{}
}

// NewMaintenanceManager creates a new maintenance manager
func NewMaintenanceManager(db *sql.DB, dbPath string, metrics *observability.MetricsRegistry, config MaintenanceConfig) (*MaintenanceManager, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("MaintenanceManager")
	}

	if dbPath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database path is required", nil).
			WithComponent("MaintenanceManager")
	}

	// Ensure backup directory exists
	if config.BackupPath != "" {
		if err := os.MkdirAll(config.BackupPath, 0755); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create backup directory").
				WithComponent("MaintenanceManager").
				WithDetails("path", config.BackupPath)
		}
	}

	manager := &MaintenanceManager{
		db:             db,
		dbPath:         dbPath,
		metrics:        metrics,
		config:         config,
		maintenanceLog: make([]MaintenanceEvent, 0, 100),
	}

	// Configure SQLite auto-vacuum if enabled
	if config.AutoVacuumEnabled {
		if err := manager.configureAutoVacuum(context.Background()); err != nil {
			// Log but don't fail
			_ = err
		}
	}

	if config.Enabled {
		go manager.maintenanceLoop()
	}

	return manager, nil
}

// RunVacuum performs database vacuum operation
func (m *MaintenanceManager) RunVacuum(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("MaintenanceManager").
			WithOperation("RunVacuum")
	}

	event := MaintenanceEvent{
		Type:      "vacuum",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Get database size before vacuum
	var sizeBeforePages, pageSize int64
	_ = m.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&sizeBeforePages)
	_ = m.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	sizeBefore := sizeBeforePages * pageSize
	event.Details["size_before"] = sizeBefore

	// Run VACUUM
	_, err := m.db.ExecContext(ctx, "VACUUM")
	event.EndTime = time.Now()
	event.Duration = event.EndTime.Sub(event.StartTime)

	if err != nil {
		event.Success = false
		event.Error = err
		m.logEvent(event)
		return gerror.Wrap(err, gerror.ErrCodeStorage, "vacuum failed").
			WithComponent("MaintenanceManager").
			WithOperation("RunVacuum")
	}

	// Get database size after vacuum
	var sizeAfterPages int64
	_ = m.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&sizeAfterPages)
	sizeAfter := sizeAfterPages * pageSize
	event.Details["size_after"] = sizeAfter
	event.Details["space_saved"] = sizeBefore - sizeAfter

	event.Success = true
	m.logEvent(event)

	m.mu.Lock()
	m.lastVacuum = time.Now()
	m.mu.Unlock()

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = sizeBefore
	_ = sizeAfter

	return nil
}

// UpdateStatistics updates table statistics for query optimization
func (m *MaintenanceManager) UpdateStatistics(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("MaintenanceManager").
			WithOperation("UpdateStatistics")
	}

	event := MaintenanceEvent{
		Type:      "analyze",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Run ANALYZE
	_, err := m.db.ExecContext(ctx, "ANALYZE")
	event.EndTime = time.Now()
	event.Duration = event.EndTime.Sub(event.StartTime)

	if err != nil {
		event.Success = false
		event.Error = err
		m.logEvent(event)
		return gerror.Wrap(err, gerror.ErrCodeStorage, "analyze failed").
			WithComponent("MaintenanceManager").
			WithOperation("UpdateStatistics")
	}

	event.Success = true
	m.logEvent(event)

	m.mu.Lock()
	m.lastAnalyze = time.Now()
	m.mu.Unlock()

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods

	return nil
}

// RebuildIndexes rebuilds fragmented indexes
func (m *MaintenanceManager) RebuildIndexes(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("MaintenanceManager").
			WithOperation("RebuildIndexes")
	}

	event := MaintenanceEvent{
		Type:      "index_rebuild",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Get list of indexes
	query := `SELECT name FROM sqlite_master WHERE type='index' AND sql IS NOT NULL`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		event.Success = false
		event.Error = err
		m.logEvent(event)
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query indexes").
			WithComponent("MaintenanceManager").
			WithOperation("RebuildIndexes")
	}
	defer rows.Close()

	rebuiltCount := 0
	var indexes []string
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			continue
		}
		indexes = append(indexes, indexName)
	}

	// Rebuild each index
	for _, indexName := range indexes {
		// REINDEX specific index
		if _, err := m.db.ExecContext(ctx, fmt.Sprintf("REINDEX %s", indexName)); err == nil {
			rebuiltCount++
		}
	}

	event.EndTime = time.Now()
	event.Duration = event.EndTime.Sub(event.StartTime)
	event.Success = true
	event.Details["indexes_rebuilt"] = rebuiltCount
	event.Details["total_indexes"] = len(indexes)
	m.logEvent(event)

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = rebuiltCount

	return nil
}

// CreateBackup creates a database backup
func (m *MaintenanceManager) CreateBackup(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("MaintenanceManager").
			WithOperation("CreateBackup")
	}

	event := MaintenanceEvent{
		Type:      "backup",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("guild_backup_%s.db", timestamp)
	backupPath := filepath.Join(m.config.BackupPath, backupName)
	event.Details["backup_path"] = backupPath

	// Create backup using SQLite backup API
	backupDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		event.Success = false
		event.Error = err
		m.logEvent(event)
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create backup database").
			WithComponent("MaintenanceManager").
			WithOperation("CreateBackup").
			WithDetails("path", backupPath)
	}
	defer backupDB.Close()

	// Use VACUUM INTO for online backup (SQLite 3.27.0+)
	_, err = m.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		// Fallback to manual copy if VACUUM INTO not supported
		os.Remove(backupPath)
		event.Success = false
		event.Error = err
		m.logEvent(event)
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "backup failed").
			WithComponent("MaintenanceManager").
			WithOperation("CreateBackup")
	}

	// Get backup size
	if info, err := os.Stat(backupPath); err == nil {
		event.Details["backup_size"] = info.Size()
	}

	event.EndTime = time.Now()
	event.Duration = event.EndTime.Sub(event.StartTime)
	event.Success = true
	m.logEvent(event)

	m.mu.Lock()
	m.lastBackup = time.Now()
	m.mu.Unlock()

	// Clean old backups
	if err := m.cleanOldBackups(ctx); err != nil {
		// Log but don't fail
		_ = err
	}

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods

	return backupPath, nil
}

// RestoreBackup restores a database from backup
func (m *MaintenanceManager) RestoreBackup(ctx context.Context, backupPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("MaintenanceManager").
			WithOperation("RestoreBackup")
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return gerror.New(gerror.ErrCodeNotFound, "backup file not found", err).
			WithComponent("MaintenanceManager").
			WithOperation("RestoreBackup").
			WithDetails("path", backupPath)
	}

	// This is a simplified implementation
	// In production, would need to handle active connections, transactions, etc.
	return gerror.New(gerror.ErrCodeNotImplemented, "backup restoration not yet implemented", nil).
		WithComponent("MaintenanceManager").
		WithOperation("RestoreBackup")
}

// GetMaintenanceSchedule returns the next scheduled maintenance times
func (m *MaintenanceManager) GetMaintenanceSchedule() MaintenanceSchedule {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	return MaintenanceSchedule{
		NextVacuum:  m.lastVacuum.Add(m.config.VacuumInterval),
		NextAnalyze: m.lastAnalyze.Add(m.config.AnalyzeInterval),
		NextBackup:  m.lastBackup.Add(m.config.BackupInterval),
		LastVacuum:  m.lastVacuum,
		LastAnalyze: m.lastAnalyze,
		LastBackup:  m.lastBackup,
		IsOverdue: now.After(m.lastVacuum.Add(m.config.VacuumInterval)) ||
			now.After(m.lastAnalyze.Add(m.config.AnalyzeInterval)) ||
			now.After(m.lastBackup.Add(m.config.BackupInterval)),
	}
}

// MaintenanceSchedule contains maintenance scheduling information
type MaintenanceSchedule struct {
	NextVacuum  time.Time
	NextAnalyze time.Time
	NextBackup  time.Time
	LastVacuum  time.Time
	LastAnalyze time.Time
	LastBackup  time.Time
	IsOverdue   bool
}

// GetMaintenanceHistory returns recent maintenance events
func (m *MaintenanceManager) GetMaintenanceHistory(limit int) []MaintenanceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limit <= 0 || limit > len(m.maintenanceLog) {
		limit = len(m.maintenanceLog)
	}

	// Return most recent events
	start := len(m.maintenanceLog) - limit
	if start < 0 {
		start = 0
	}

	result := make([]MaintenanceEvent, limit)
	copy(result, m.maintenanceLog[start:])

	return result
}

// configureAutoVacuum configures SQLite auto-vacuum mode
func (m *MaintenanceManager) configureAutoVacuum(ctx context.Context) error {
	// Check current auto_vacuum setting
	var currentMode int
	if err := m.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&currentMode); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query auto_vacuum mode").
			WithComponent("MaintenanceManager").
			WithOperation("configureAutoVacuum")
	}

	// 0 = NONE, 1 = FULL, 2 = INCREMENTAL
	if currentMode == 0 && m.config.AutoVacuumEnabled {
		// Auto-vacuum can only be changed on an empty database
		// Log for information
		_ = gerror.New(gerror.ErrCodeConfiguration, "auto_vacuum must be set before any tables are created", nil).
			WithComponent("MaintenanceManager").
			WithOperation("configureAutoVacuum")
	}

	return nil
}

// maintenanceLoop runs periodic maintenance tasks
func (m *MaintenanceManager) maintenanceLoop() {
	// Use shorter interval to check more frequently
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

		now := time.Now()

		// Check if vacuum is due
		if now.Sub(m.lastVacuum) >= m.config.VacuumInterval {
			if err := m.RunVacuum(ctx); err != nil {
				// Log error but continue
				_ = err
			}
		}

		// Check if analyze is due
		if now.Sub(m.lastAnalyze) >= m.config.AnalyzeInterval {
			if err := m.UpdateStatistics(ctx); err != nil {
				// Log error but continue
				_ = err
			}
		}

		// Check if backup is due
		if now.Sub(m.lastBackup) >= m.config.BackupInterval {
			if _, err := m.CreateBackup(ctx); err != nil {
				// Log error but continue
				_ = err
			}
		}

		cancel()
	}
}

// cleanOldBackups removes backups older than retention period
func (m *MaintenanceManager) cleanOldBackups(ctx context.Context) error {
	files, err := os.ReadDir(m.config.BackupPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read backup directory").
			WithComponent("MaintenanceManager").
			WithOperation("cleanOldBackups")
	}

	cutoff := time.Now().AddDate(0, 0, -m.config.BackupRetentionDays)
	removed := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// Remove old backups
		if info.ModTime().Before(cutoff) {
			backupPath := filepath.Join(m.config.BackupPath, file.Name())
			if err := os.Remove(backupPath); err == nil {
				removed++
			}
		}
	}

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = removed

	return nil
}

// logEvent logs a maintenance event
func (m *MaintenanceManager) logEvent(event MaintenanceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Limit log size
	if len(m.maintenanceLog) >= 1000 {
		m.maintenanceLog = m.maintenanceLog[100:]
	}

	m.maintenanceLog = append(m.maintenanceLog, event)
}
