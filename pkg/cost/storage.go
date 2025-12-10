// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	_ "modernc.org/sqlite"
)

// CostStorage handles persistent storage of cost data
type CostStorage struct {
	db *sql.DB
}

// NewCostStorage creates a new cost storage instance
func NewCostStorage(ctx context.Context) (*CostStorage, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "NewCostStorage")

	// Use in-memory SQLite for cost storage (modernc driver)
	db, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize database").
			WithComponent("cost.storage").
			WithOperation("NewCostStorage")
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to ping database").
			WithComponent("cost.storage").
			WithOperation("NewCostStorage")
	}

	storage := &CostStorage{
		db: db,
	}

	// Initialize cost tables
	if err := storage.initializeTables(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize tables").
			WithComponent("cost.storage").
			WithOperation("NewCostStorage")
	}

	return storage, nil
}

// StoreUsage stores a usage record
func (cs *CostStorage) StoreUsage(ctx context.Context, usage Usage) error {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "StoreUsage")

	// Serialize metadata
	metadataJSON, err := json.Marshal(usage.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to serialize metadata").
			WithComponent("cost.storage").
			WithOperation("StoreUsage")
	}

	query := `
		INSERT INTO cost_usage (
			agent_id, session_id, provider, resource, quantity, unit, 
			timestamp, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = cs.db.ExecContext(ctx, query,
		usage.AgentID,
		usage.SessionID,
		usage.Provider,
		usage.Resource,
		usage.Quantity,
		usage.Unit,
		usage.Timestamp,
		string(metadataJSON),
	)

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store usage").
			WithComponent("cost.storage").
			WithOperation("StoreUsage").
			WithDetails("agent_id", usage.AgentID).
			WithDetails("provider", usage.Provider)
	}

	return nil
}

// GetHistoricalCosts retrieves historical cost data
func (cs *CostStorage) GetHistoricalCosts(ctx context.Context, period TimePeriod) (*HistoricalCosts, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "GetHistoricalCosts")

	query := `
		SELECT 
			timestamp,
			SUM(CAST(JSON_EXTRACT(metadata, '$.total_cost') AS REAL)) as cost
		FROM cost_usage 
		WHERE timestamp BETWEEN ? AND ?
		GROUP BY strftime('%Y-%m-%d %H:%M', timestamp)
		ORDER BY timestamp
	`

	rows, err := cs.db.QueryContext(ctx, query, period.Start, period.End)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query historical costs").
			WithComponent("cost.storage").
			WithOperation("GetHistoricalCosts")
	}
	defer rows.Close()

	var dataPoints []CostDataPoint
	totalCost := 0.0

	for rows.Next() {
		var timestamp time.Time
		var cost sql.NullFloat64

		if err := rows.Scan(&timestamp, &cost); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan historical cost row").
				WithComponent("cost.storage").
				WithOperation("GetHistoricalCosts")
		}

		costValue := 0.0
		if cost.Valid {
			costValue = cost.Float64
		}

		dataPoints = append(dataPoints, CostDataPoint{
			Timestamp: timestamp,
			Cost:      costValue,
		})

		totalCost += costValue
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating historical cost rows").
			WithComponent("cost.storage").
			WithOperation("GetHistoricalCosts")
	}

	return &HistoricalCosts{
		Period:     period,
		DataPoints: dataPoints,
		TotalCost:  totalCost,
		Currency:   "USD",
	}, nil
}

// GetCostsByAgent retrieves cost breakdown by agent
func (cs *CostStorage) GetCostsByAgent(ctx context.Context, period TimePeriod) (map[string]float64, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "GetCostsByAgent")

	query := `
		SELECT 
			agent_id,
			SUM(CAST(JSON_EXTRACT(metadata, '$.total_cost') AS REAL)) as total_cost
		FROM cost_usage 
		WHERE timestamp BETWEEN ? AND ?
		  AND JSON_EXTRACT(metadata, '$.total_cost') IS NOT NULL
		GROUP BY agent_id
	`

	rows, err := cs.db.QueryContext(ctx, query, period.Start, period.End)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query costs by agent").
			WithComponent("cost.storage").
			WithOperation("GetCostsByAgent")
	}
	defer rows.Close()

	costs := make(map[string]float64)

	for rows.Next() {
		var agentID string
		var cost sql.NullFloat64

		if err := rows.Scan(&agentID, &cost); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan agent cost row").
				WithComponent("cost.storage").
				WithOperation("GetCostsByAgent")
		}

		if cost.Valid {
			costs[agentID] = cost.Float64
		}
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating agent cost rows").
			WithComponent("cost.storage").
			WithOperation("GetCostsByAgent")
	}

	return costs, nil
}

// GetCostsByProvider retrieves cost breakdown by provider
func (cs *CostStorage) GetCostsByProvider(ctx context.Context, period TimePeriod) (map[string]float64, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "GetCostsByProvider")

	query := `
		SELECT 
			provider,
			SUM(CAST(JSON_EXTRACT(metadata, '$.total_cost') AS REAL)) as total_cost
		FROM cost_usage 
		WHERE timestamp BETWEEN ? AND ?
		  AND JSON_EXTRACT(metadata, '$.total_cost') IS NOT NULL
		GROUP BY provider
	`

	rows, err := cs.db.QueryContext(ctx, query, period.Start, period.End)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query costs by provider").
			WithComponent("cost.storage").
			WithOperation("GetCostsByProvider")
	}
	defer rows.Close()

	costs := make(map[string]float64)

	for rows.Next() {
		var provider string
		var cost sql.NullFloat64

		if err := rows.Scan(&provider, &cost); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan provider cost row").
				WithComponent("cost.storage").
				WithOperation("GetCostsByProvider")
		}

		if cost.Valid {
			costs[provider] = cost.Float64
		}
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating provider cost rows").
			WithComponent("cost.storage").
			WithOperation("GetCostsByProvider")
	}

	return costs, nil
}

// GetCostsByModel retrieves cost breakdown by model
func (cs *CostStorage) GetCostsByModel(ctx context.Context, period TimePeriod) (map[string]float64, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "GetCostsByModel")

	query := `
		SELECT 
			JSON_EXTRACT(metadata, '$.model') as model,
			SUM(CAST(JSON_EXTRACT(metadata, '$.total_cost') AS REAL)) as total_cost
		FROM cost_usage 
		WHERE timestamp BETWEEN ? AND ?
		  AND JSON_EXTRACT(metadata, '$.model') IS NOT NULL
		  AND JSON_EXTRACT(metadata, '$.total_cost') IS NOT NULL
		GROUP BY JSON_EXTRACT(metadata, '$.model')
	`

	rows, err := cs.db.QueryContext(ctx, query, period.Start, period.End)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query costs by model").
			WithComponent("cost.storage").
			WithOperation("GetCostsByModel")
	}
	defer rows.Close()

	costs := make(map[string]float64)

	for rows.Next() {
		var model sql.NullString
		var cost sql.NullFloat64

		if err := rows.Scan(&model, &cost); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan model cost row").
				WithComponent("cost.storage").
				WithOperation("GetCostsByModel")
		}

		if model.Valid && cost.Valid {
			costs[model.String] = cost.Float64
		}
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating model cost rows").
			WithComponent("cost.storage").
			WithOperation("GetCostsByModel")
	}

	return costs, nil
}

// GetUsageByAgent retrieves usage records for a specific agent
func (cs *CostStorage) GetUsageByAgent(ctx context.Context, agentID string, period TimePeriod) ([]Usage, error) {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "GetUsageByAgent")

	query := `
		SELECT agent_id, session_id, provider, resource, quantity, unit, timestamp, metadata
		FROM cost_usage 
		WHERE agent_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`

	rows, err := cs.db.QueryContext(ctx, query, agentID, period.Start, period.End)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query usage by agent").
			WithComponent("cost.storage").
			WithOperation("GetUsageByAgent").
			WithDetails("agent_id", agentID)
	}
	defer rows.Close()

	var usages []Usage

	for rows.Next() {
		var usage Usage
		var sessionID sql.NullString
		var metadataJSON string

		if err := rows.Scan(
			&usage.AgentID,
			&sessionID,
			&usage.Provider,
			&usage.Resource,
			&usage.Quantity,
			&usage.Unit,
			&usage.Timestamp,
			&metadataJSON,
		); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan usage row").
				WithComponent("cost.storage").
				WithOperation("GetUsageByAgent")
		}

		if sessionID.Valid {
			usage.SessionID = sessionID.String
		}

		// Deserialize metadata
		if err := json.Unmarshal([]byte(metadataJSON), &usage.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to deserialize metadata").
				WithComponent("cost.storage").
				WithOperation("GetUsageByAgent")
		}

		usages = append(usages, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating usage rows").
			WithComponent("cost.storage").
			WithOperation("GetUsageByAgent")
	}

	return usages, nil
}

// CleanupOldUsage removes usage records older than the retention period
func (cs *CostStorage) CleanupOldUsage(ctx context.Context, retentionPeriod time.Duration) error {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "CleanupOldUsage")

	cutoffTime := time.Now().Add(-retentionPeriod)

	query := `DELETE FROM cost_usage WHERE timestamp < ?`

	result, err := cs.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to cleanup old usage").
			WithComponent("cost.storage").
			WithOperation("CleanupOldUsage")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get rows affected").
			WithComponent("cost.storage").
			WithOperation("CleanupOldUsage")
	}

	if rowsAffected > 0 {
		// Log cleanup for observability
		// Note: rowsAffected logged separately for observability
		_ = rowsAffected
	}

	return nil
}

// initializeTables creates cost tracking tables if they don't exist
func (cs *CostStorage) initializeTables(ctx context.Context) error {
	ctx = observability.WithComponent(ctx, "cost.storage")
	ctx = observability.WithOperation(ctx, "initializeTables")

	// Create cost_usage table
	usageTableSQL := `
		CREATE TABLE IF NOT EXISTS cost_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			session_id TEXT,
			provider TEXT NOT NULL,
			resource TEXT NOT NULL,
			quantity REAL NOT NULL,
			unit TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			metadata TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := cs.db.ExecContext(ctx, usageTableSQL); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create cost_usage table").
			WithComponent("cost.storage").
			WithOperation("initializeTables")
	}

	// Create indexes for better query performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_cost_usage_agent_timestamp ON cost_usage(agent_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_cost_usage_provider_timestamp ON cost_usage(provider, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_cost_usage_timestamp ON cost_usage(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_cost_usage_session ON cost_usage(session_id)`,
	}

	for _, indexSQL := range indexes {
		if _, err := cs.db.ExecContext(ctx, indexSQL); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create index").
				WithComponent("cost.storage").
				WithOperation("initializeTables").
				WithDetails("index_sql", indexSQL)
		}
	}

	// Create cost_budgets table for budget tracking
	budgetTableSQL := `
		CREATE TABLE IF NOT EXISTS cost_budgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			period_type TEXT NOT NULL, -- 'daily', 'monthly', 'yearly'
			period_start DATE NOT NULL,
			period_end DATE NOT NULL,
			budget_limit REAL NOT NULL,
			current_spend REAL DEFAULT 0,
			currency TEXT DEFAULT 'USD',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := cs.db.ExecContext(ctx, budgetTableSQL); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create cost_budgets table").
			WithComponent("cost.storage").
			WithOperation("initializeTables")
	}

	// Create cost_alerts table for alert tracking
	alertTableSQL := `
		CREATE TABLE IF NOT EXISTS cost_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			alert_type TEXT NOT NULL,
			threshold_value REAL NOT NULL,
			current_value REAL NOT NULL,
			severity TEXT NOT NULL,
			message TEXT NOT NULL,
			triggered_at DATETIME NOT NULL,
			resolved_at DATETIME,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := cs.db.ExecContext(ctx, alertTableSQL); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create cost_alerts table").
			WithComponent("cost.storage").
			WithOperation("initializeTables")
	}

	return nil
}
