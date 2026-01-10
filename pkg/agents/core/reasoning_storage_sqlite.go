// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/storage"
	"github.com/lancekrogers/guild-core/pkg/storage/db"
)

// SQLiteReasoningStorage provides SQLite-backed storage for reasoning chains
type SQLiteReasoningStorage struct {
	db      *storage.Database
	queries *db.Queries
	config  ReasoningStorageConfig
}

// NewSQLiteReasoningStorage creates a new SQLite reasoning storage
func NewSQLiteReasoningStorage(ctx context.Context, database *storage.Database, config ReasoningStorageConfig) (*SQLiteReasoningStorage, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("NewSQLiteReasoningStorage")
	}

	if database == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "database cannot be nil", nil).
			WithComponent("sqlite_reasoning_storage").
			WithOperation("NewSQLiteReasoningStorage")
	}

	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid storage configuration").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("NewSQLiteReasoningStorage")
	}

	return &SQLiteReasoningStorage{
		db:      database,
		queries: database.Queries(),
		config:  config,
	}, nil
}

// Store saves a reasoning chain to SQLite
func (s *SQLiteReasoningStorage) Store(ctx context.Context, chain *ReasoningChain) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Store")
	}

	if chain == nil {
		return gerror.New(gerror.ErrCodeValidation, "chain cannot be nil", nil).
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Store")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("sqlite_reasoning_storage").
		WithOperation("Store").
		With("chain_id", chain.ID, "agent_id", chain.AgentID)

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(chain.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal metadata").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Store").
			WithDetails("chain_id", chain.ID)
	}

	// Execute insert
	query := `
		INSERT INTO reasoning_chains (
			id, agent_id, session_id, request_id, content, reasoning,
			confidence, task_type, success, tokens_used, duration_ms,
			created_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.DB().ExecContext(ctx, query,
		chain.ID,
		chain.AgentID,
		sql.NullString{String: chain.SessionID, Valid: chain.SessionID != ""},
		sql.NullString{String: chain.RequestID, Valid: chain.RequestID != ""},
		chain.Content,
		chain.Reasoning,
		chain.Confidence,
		sql.NullString{String: chain.TaskType, Valid: chain.TaskType != ""},
		chain.Success,
		chain.TokensUsed,
		chain.Duration.Milliseconds(),
		chain.CreatedAt,
		string(metadataJSON),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to insert reasoning chain").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Store").
			WithDetails("chain_id", chain.ID)
	}

	logger.DebugContext(ctx, "Reasoning chain stored successfully")

	// Update analytics asynchronously
	go s.updateAnalytics(context.Background(), chain.AgentID)

	return nil
}

// Get retrieves a reasoning chain by ID
func (s *SQLiteReasoningStorage) Get(ctx context.Context, id string) (*ReasoningChain, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Get")
	}

	query := `
		SELECT id, agent_id, session_id, request_id, content, reasoning,
			   confidence, task_type, success, tokens_used, duration_ms,
			   created_at, metadata
		FROM reasoning_chains
		WHERE id = ?
	`

	var chain ReasoningChain
	var sessionID, requestID, taskType sql.NullString
	var metadataJSON sql.NullString
	var durationMs int64

	err := s.db.DB().QueryRowContext(ctx, query, id).Scan(
		&chain.ID,
		&chain.AgentID,
		&sessionID,
		&requestID,
		&chain.Content,
		&chain.Reasoning,
		&chain.Confidence,
		&taskType,
		&chain.Success,
		&chain.TokensUsed,
		&durationMs,
		&chain.CreatedAt,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "reasoning chain not found: %s", id).
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Get").
			WithDetails("chain_id", id)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query reasoning chain").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Get").
			WithDetails("chain_id", id)
	}

	// Set nullable fields
	chain.SessionID = sessionID.String
	chain.RequestID = requestID.String
	chain.TaskType = taskType.String
	chain.Duration = time.Duration(durationMs) * time.Millisecond

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &chain.Metadata); err != nil {
			// Log warning but don't fail
			logger := observability.GetLogger(ctx)
			logger.WarnContext(ctx, "Failed to unmarshal metadata",
				"chain_id", id,
				"error", err.Error())
			chain.Metadata = make(map[string]interface{})
		}
	} else {
		chain.Metadata = make(map[string]interface{})
	}

	return &chain, nil
}

// Query searches for reasoning chains based on criteria
func (s *SQLiteReasoningStorage) Query(ctx context.Context, query *ReasoningQuery) ([]*ReasoningChain, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Query")
	}

	if query == nil {
		query = &ReasoningQuery{
			Limit:   s.config.MaxChainsPerQuery,
			OrderBy: "created_at",
		}
	}

	// Build dynamic query
	var conditions []string
	var args []interface{}

	if query.AgentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, query.AgentID)
	}

	if query.SessionID != "" {
		conditions = append(conditions, "session_id = ?")
		args = append(args, query.SessionID)
	}

	if query.TaskType != "" {
		conditions = append(conditions, "task_type = ?")
		args = append(args, query.TaskType)
	}

	if query.MinConfidence > 0 {
		conditions = append(conditions, "confidence >= ?")
		args = append(args, query.MinConfidence)
	}

	if query.MaxConfidence > 0 {
		conditions = append(conditions, "confidence <= ?")
		args = append(args, query.MaxConfidence)
	}

	if !query.StartTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, query.StartTime)
	}

	if !query.EndTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, query.EndTime)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderByClause := s.buildOrderByClause(query.OrderBy, query.Ascending)

	// Build final query with pagination
	sqlQuery := fmt.Sprintf(`
		SELECT id, agent_id, session_id, request_id, content, reasoning,
			   confidence, task_type, success, tokens_used, duration_ms,
			   created_at, metadata
		FROM reasoning_chains
		%s
		%s
		LIMIT ? OFFSET ?
	`, whereClause, orderByClause)

	// Add pagination args
	limit := query.Limit
	if limit == 0 || limit > s.config.MaxChainsPerQuery {
		limit = s.config.MaxChainsPerQuery
	}
	args = append(args, limit, query.Offset)

	// Execute query
	rows, err := s.db.DB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query reasoning chains").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Query")
	}
	defer rows.Close()

	// Parse results
	var chains []*ReasoningChain
	for rows.Next() {
		chain, err := s.scanChain(rows)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan reasoning chain").
				WithComponent("sqlite_reasoning_storage").
				WithOperation("Query")
		}
		chains = append(chains, chain)
	}

	if err := rows.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error iterating reasoning chains").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Query")
	}

	return chains, nil
}

// GetStats returns aggregated statistics
func (s *SQLiteReasoningStorage) GetStats(ctx context.Context, agentID string, startTime, endTime time.Time) (*ReasoningStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("GetStats")
	}

	// Check if we have cached stats
	cached, err := s.getCachedStats(ctx, agentID, startTime, endTime)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Build conditions
	var conditions []string
	var args []interface{}

	if agentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, agentID)
	}

	if !startTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, startTime)
	}

	if !endTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, endTime)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Query basic stats
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_chains,
			AVG(confidence) as avg_confidence,
			AVG(duration_ms) as avg_duration,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) * 1.0 / COUNT(*) as success_rate
		FROM reasoning_chains
		%s
	`, whereClause)

	var stats ReasoningStats
	var avgDurationMs sql.NullFloat64

	err = s.db.DB().QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalChains,
		&stats.AvgConfidence,
		&avgDurationMs,
		&stats.SuccessRate,
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query reasoning stats").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("GetStats")
	}

	if avgDurationMs.Valid {
		stats.AvgDuration = time.Duration(avgDurationMs.Float64) * time.Millisecond
	}

	// Get distributions
	stats.ConfidenceDistrib = s.getConfidenceDistribution(ctx, whereClause, args)
	stats.TaskTypeDistrib = s.getTaskTypeDistribution(ctx, whereClause, args)
	stats.TimeDistrib = s.getTimeDistribution(ctx, whereClause, args)

	// Cache the stats
	go s.cacheStats(context.Background(), agentID, startTime, endTime, &stats)

	return &stats, nil
}

// GetPatterns returns learned reasoning patterns
func (s *SQLiteReasoningStorage) GetPatterns(ctx context.Context, taskType string, limit int) ([]*ReasoningPattern, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("GetPatterns")
	}

	query := `
		SELECT id, pattern, task_type, occurrences, avg_success, examples,
			   created_at, updated_at, metadata
		FROM reasoning_patterns
		WHERE task_type = ? OR ? = ''
		ORDER BY occurrences DESC
		LIMIT ?
	`

	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.DB().QueryContext(ctx, query, taskType, taskType, limit)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query reasoning patterns").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("GetPatterns")
	}
	defer rows.Close()

	var patterns []*ReasoningPattern
	for rows.Next() {
		pattern, err := s.scanPattern(rows)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan reasoning pattern").
				WithComponent("sqlite_reasoning_storage").
				WithOperation("GetPatterns")
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// UpdatePattern updates or creates a reasoning pattern
func (s *SQLiteReasoningStorage) UpdatePattern(ctx context.Context, pattern *ReasoningPattern) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("UpdatePattern")
	}

	if pattern == nil || pattern.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "pattern and pattern ID cannot be empty", nil).
			WithComponent("sqlite_reasoning_storage").
			WithOperation("UpdatePattern")
	}

	// Marshal examples and metadata
	examplesJSON, _ := json.Marshal(pattern.Examples)
	metadataJSON, _ := json.Marshal(pattern.Metadata)

	query := `
		INSERT INTO reasoning_patterns (
			id, pattern, task_type, occurrences, avg_success,
			examples, created_at, updated_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			occurrences = excluded.occurrences,
			avg_success = excluded.avg_success,
			examples = excluded.examples,
			updated_at = excluded.updated_at,
			metadata = excluded.metadata
	`

	_, err := s.db.DB().ExecContext(ctx, query,
		pattern.ID,
		pattern.Pattern,
		sql.NullString{String: pattern.TaskType, Valid: pattern.TaskType != ""},
		pattern.Occurrences,
		pattern.AvgSuccess,
		string(examplesJSON),
		pattern.CreatedAt,
		pattern.UpdatedAt,
		string(metadataJSON),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update reasoning pattern").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("UpdatePattern").
			WithDetails("pattern_id", pattern.ID)
	}

	return nil
}

// Delete removes old reasoning chains
func (s *SQLiteReasoningStorage) Delete(ctx context.Context, beforeTime time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Delete")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("sqlite_reasoning_storage").
		WithOperation("Delete")

	result, err := s.db.DB().ExecContext(ctx,
		"DELETE FROM reasoning_chains WHERE created_at < ?",
		beforeTime,
	)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete old reasoning chains").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Delete")
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get rows affected").
			WithComponent("sqlite_reasoning_storage").
			WithOperation("Delete")
	}

	logger.InfoContext(ctx, "Deleted old reasoning chains",
		"deleted_count", deleted,
		"before_time", beforeTime)

	// Clean up old analytics
	go s.cleanupAnalytics(context.Background(), beforeTime)

	return deleted, nil
}

// Close closes the storage connection
func (s *SQLiteReasoningStorage) Close(ctx context.Context) error {
	// Database closing is handled by the storage package
	return nil
}

// Helper methods

func (s *SQLiteReasoningStorage) buildOrderByClause(orderBy string, ascending bool) string {
	// Validate orderBy to prevent SQL injection
	validColumns := map[string]string{
		"created_at": "created_at",
		"confidence": "confidence",
		"duration":   "duration_ms",
	}

	column, ok := validColumns[orderBy]
	if !ok {
		column = "created_at"
	}

	direction := "DESC"
	if ascending {
		direction = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", column, direction)
}

func (s *SQLiteReasoningStorage) scanChain(rows *sql.Rows) (*ReasoningChain, error) {
	var chain ReasoningChain
	var sessionID, requestID, taskType sql.NullString
	var metadataJSON sql.NullString
	var durationMs int64

	err := rows.Scan(
		&chain.ID,
		&chain.AgentID,
		&sessionID,
		&requestID,
		&chain.Content,
		&chain.Reasoning,
		&chain.Confidence,
		&taskType,
		&chain.Success,
		&chain.TokensUsed,
		&durationMs,
		&chain.CreatedAt,
		&metadataJSON,
	)
	if err != nil {
		return nil, err
	}

	// Set nullable fields
	chain.SessionID = sessionID.String
	chain.RequestID = requestID.String
	chain.TaskType = taskType.String
	chain.Duration = time.Duration(durationMs) * time.Millisecond

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		json.Unmarshal([]byte(metadataJSON.String), &chain.Metadata)
	} else {
		chain.Metadata = make(map[string]interface{})
	}

	return &chain, nil
}

func (s *SQLiteReasoningStorage) scanPattern(rows *sql.Rows) (*ReasoningPattern, error) {
	var pattern ReasoningPattern
	var taskType sql.NullString
	var examplesJSON, metadataJSON string

	err := rows.Scan(
		&pattern.ID,
		&pattern.Pattern,
		&taskType,
		&pattern.Occurrences,
		&pattern.AvgSuccess,
		&examplesJSON,
		&pattern.CreatedAt,
		&pattern.UpdatedAt,
		&metadataJSON,
	)
	if err != nil {
		return nil, err
	}

	pattern.TaskType = taskType.String

	// Unmarshal JSON fields
	if examplesJSON != "" {
		json.Unmarshal([]byte(examplesJSON), &pattern.Examples)
	}
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &pattern.Metadata)
	}

	return &pattern, nil
}

func (s *SQLiteReasoningStorage) getConfidenceDistribution(ctx context.Context, whereClause string, args []interface{}) map[string]int {
	query := fmt.Sprintf(`
		SELECT 
			CASE 
				WHEN confidence >= 0.9 THEN 'very_high'
				WHEN confidence >= 0.7 THEN 'high'
				WHEN confidence >= 0.5 THEN 'medium'
				WHEN confidence >= 0.3 THEN 'low'
				ELSE 'very_low'
			END as bucket,
			COUNT(*) as count
		FROM reasoning_chains
		%s
		GROUP BY bucket
	`, whereClause)

	rows, err := s.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return make(map[string]int)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var bucket string
		var count int
		if err := rows.Scan(&bucket, &count); err == nil {
			distribution[bucket] = count
		}
	}

	return distribution
}

func (s *SQLiteReasoningStorage) getTaskTypeDistribution(ctx context.Context, whereClause string, args []interface{}) map[string]int {
	query := fmt.Sprintf(`
		SELECT task_type, COUNT(*) as count
		FROM reasoning_chains
		%s AND task_type IS NOT NULL
		GROUP BY task_type
	`, whereClause)

	// Adjust for empty whereClause
	if whereClause == "" {
		query = strings.Replace(query, " AND", " WHERE", 1)
	}

	rows, err := s.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return make(map[string]int)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var taskType string
		var count int
		if err := rows.Scan(&taskType, &count); err == nil {
			distribution[taskType] = count
		}
	}

	return distribution
}

func (s *SQLiteReasoningStorage) getTimeDistribution(ctx context.Context, whereClause string, args []interface{}) map[string]int {
	query := fmt.Sprintf(`
		SELECT strftime('%%Y-%%m-%%d %%H:00', created_at) as hour_bucket, COUNT(*) as count
		FROM reasoning_chains
		%s
		GROUP BY hour_bucket
		ORDER BY hour_bucket DESC
		LIMIT 168
	`, whereClause)

	rows, err := s.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return make(map[string]int)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var bucket string
		var count int
		if err := rows.Scan(&bucket, &count); err == nil {
			distribution[bucket] = count
		}
	}

	return distribution
}

// Analytics caching methods

func (s *SQLiteReasoningStorage) getCachedStats(ctx context.Context, agentID string, startTime, endTime time.Time) (*ReasoningStats, error) {
	// Generate cache key based on time range
	timeRange := s.generateTimeRangeKey(startTime, endTime)

	query := `
		SELECT total_chains, avg_confidence, avg_duration_ms, success_rate,
			   confidence_distribution, task_type_distribution, insights
		FROM reasoning_analytics
		WHERE agent_id = ? AND time_range = ?
		  AND created_at > datetime('now', '-1 hour')
	`

	var stats ReasoningStats
	var avgDurationMs int
	var confDistJSON, taskDistJSON, insightsJSON string

	err := s.db.DB().QueryRowContext(ctx, query, agentID, timeRange).Scan(
		&stats.TotalChains,
		&stats.AvgConfidence,
		&avgDurationMs,
		&stats.SuccessRate,
		&confDistJSON,
		&taskDistJSON,
		&insightsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	stats.AvgDuration = time.Duration(avgDurationMs) * time.Millisecond

	// Unmarshal JSON fields
	json.Unmarshal([]byte(confDistJSON), &stats.ConfidenceDistrib)
	json.Unmarshal([]byte(taskDistJSON), &stats.TaskTypeDistrib)

	return &stats, nil
}

func (s *SQLiteReasoningStorage) cacheStats(ctx context.Context, agentID string, startTime, endTime time.Time, stats *ReasoningStats) {
	timeRange := s.generateTimeRangeKey(startTime, endTime)

	// Marshal distributions
	confDistJSON, _ := json.Marshal(stats.ConfidenceDistrib)
	taskDistJSON, _ := json.Marshal(stats.TaskTypeDistrib)
	insightsJSON, _ := json.Marshal([]string{})

	query := `
		INSERT INTO reasoning_analytics (
			id, agent_id, time_range, total_chains, avg_confidence,
			avg_duration_ms, success_rate, confidence_distribution,
			task_type_distribution, insights
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			total_chains = excluded.total_chains,
			avg_confidence = excluded.avg_confidence,
			avg_duration_ms = excluded.avg_duration_ms,
			success_rate = excluded.success_rate,
			confidence_distribution = excluded.confidence_distribution,
			task_type_distribution = excluded.task_type_distribution,
			created_at = CURRENT_TIMESTAMP
	`

	id := fmt.Sprintf("%s_%s_%d", agentID, timeRange, time.Now().Unix())

	s.db.DB().ExecContext(ctx, query,
		id,
		agentID,
		timeRange,
		stats.TotalChains,
		stats.AvgConfidence,
		stats.AvgDuration.Milliseconds(),
		stats.SuccessRate,
		string(confDistJSON),
		string(taskDistJSON),
		string(insightsJSON),
	)
}

func (s *SQLiteReasoningStorage) generateTimeRangeKey(startTime, endTime time.Time) string {
	if startTime.IsZero() && endTime.IsZero() {
		return "all_time"
	}

	if startTime.IsZero() {
		return fmt.Sprintf("before_%s", endTime.Format("2006-01-02"))
	}

	if endTime.IsZero() {
		return fmt.Sprintf("after_%s", startTime.Format("2006-01-02"))
	}

	duration := endTime.Sub(startTime)
	if duration <= 24*time.Hour {
		return startTime.Format("2006-01-02_daily")
	} else if duration <= 7*24*time.Hour {
		return startTime.Format("2006-W01_weekly")
	} else if duration <= 31*24*time.Hour {
		return startTime.Format("2006-01_monthly")
	}

	return fmt.Sprintf("%s_to_%s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
}

func (s *SQLiteReasoningStorage) updateAnalytics(ctx context.Context, agentID string) {
	// Update daily analytics
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	stats, err := s.GetStats(ctx, agentID, startOfDay, now)
	if err == nil {
		s.cacheStats(ctx, agentID, startOfDay, now, stats)
	}
}

func (s *SQLiteReasoningStorage) cleanupAnalytics(ctx context.Context, beforeTime time.Time) {
	query := "DELETE FROM reasoning_analytics WHERE created_at < ?"
	s.db.DB().ExecContext(ctx, query, beforeTime)
}
