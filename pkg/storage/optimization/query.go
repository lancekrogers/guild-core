// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// QueryOptimizer analyzes and optimizes database queries
type QueryOptimizer struct {
	db      *sql.DB
	cache   *QueryCache
	metrics *observability.MetricsRegistry
	mu      sync.RWMutex

	// Query analysis
	slowQueryThreshold time.Duration
	queryStats         map[string]*QueryStats

	// Prepared statements
	preparedStatements map[string]*sql.Stmt
}

// QueryStats tracks statistics for a specific query
type QueryStats struct {
	Query          string
	ExecutionCount int64
	TotalDuration  time.Duration
	AvgDuration    time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
	LastExecuted   time.Time
	IndexesUsed    []string
}

// QueryPlan represents an analyzed query execution plan
type QueryPlan struct {
	Query       string
	Plan        string
	Cost        float64
	IndexesUsed []string
	Suggestions []string
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *sql.DB, metrics *observability.MetricsRegistry) (*QueryOptimizer, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("QueryOptimizer")
	}

	cache, err := NewQueryCache(DefaultQueryCacheConfig())
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create query cache").
			WithComponent("QueryOptimizer")
	}

	qo := &QueryOptimizer{
		db:                 db,
		cache:              cache,
		slowQueryThreshold: 100 * time.Millisecond,
		queryStats:         make(map[string]*QueryStats),
		preparedStatements: make(map[string]*sql.Stmt),
	}
	if metrics != nil {
		qo.metrics = metrics
	}
	return qo, nil
}

// AnalyzeQuery analyzes a query's execution plan
func (q *QueryOptimizer) AnalyzeQuery(ctx context.Context, query string) (*QueryPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("QueryOptimizer").
			WithOperation("AnalyzeQuery")
	}

	// Normalize query for analysis
	normalizedQuery := q.normalizeQuery(query)

	// Execute EXPLAIN QUERY PLAN
	rows, err := q.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+normalizedQuery)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to explain query").
			WithComponent("QueryOptimizer").
			WithOperation("AnalyzeQuery").
			WithDetails("query", normalizedQuery)
	}
	defer rows.Close()

	plan := &QueryPlan{
		Query:       normalizedQuery,
		Suggestions: []string{},
		IndexesUsed: []string{},
	}

	var planLines []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan query plan").
				WithComponent("QueryOptimizer").
				WithOperation("AnalyzeQuery")
		}

		planLines = append(planLines, detail)

		// Extract index usage
		if strings.Contains(detail, "USING INDEX") {
			start := strings.Index(detail, "USING INDEX") + 12
			end := strings.IndexAny(detail[start:], " )")
			if end == -1 {
				end = len(detail) - start
			}
			indexName := strings.TrimSpace(detail[start : start+end])
			plan.IndexesUsed = append(plan.IndexesUsed, indexName)
		}

		// Analyze for optimization opportunities
		if strings.Contains(detail, "SCAN TABLE") && !strings.Contains(detail, "USING INDEX") {
			tableName := q.extractTableName(detail)
			plan.Suggestions = append(plan.Suggestions, fmt.Sprintf("Consider adding an index for table %s", tableName))
			plan.Cost += 100.0 // High cost for table scans
		}

		if strings.Contains(detail, "TEMP B-TREE") {
			plan.Suggestions = append(plan.Suggestions, "Query requires temporary B-tree for sorting, consider adding appropriate indexes")
			plan.Cost += 50.0
		}
	}

	plan.Plan = strings.Join(planLines, "\n")

	// Add general suggestions based on query pattern
	q.addQueryPatternSuggestions(normalizedQuery, plan)

	return plan, nil
}

// OptimizeIndexes suggests index optimizations based on query patterns
func (q *QueryOptimizer) OptimizeIndexes(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("QueryOptimizer").
			WithOperation("OptimizeIndexes")
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	var suggestions []string

	// Analyze frequently executed queries
	for query, stats := range q.queryStats {
		if stats.ExecutionCount > 100 && stats.AvgDuration > 50*time.Millisecond {
			plan, err := q.AnalyzeQuery(ctx, query)
			if err != nil {
				continue
			}

			if len(plan.IndexesUsed) == 0 && strings.Contains(strings.ToUpper(query), "WHERE") {
				suggestions = append(suggestions, fmt.Sprintf("Frequently executed slow query needs indexing: %s", query))
			}
		}
	}

	// Check existing indexes for redundancy
	redundantIndexes, err := q.findRedundantIndexes(ctx)
	if err == nil && len(redundantIndexes) > 0 {
		for _, idx := range redundantIndexes {
			suggestions = append(suggestions, fmt.Sprintf("Consider removing redundant index: %s", idx))
		}
	}

	// Check for missing foreign key indexes
	missingFKIndexes, err := q.findMissingForeignKeyIndexes(ctx)
	if err == nil && len(missingFKIndexes) > 0 {
		for _, fk := range missingFKIndexes {
			suggestions = append(suggestions, fmt.Sprintf("Add index for foreign key: %s", fk))
		}
	}

	return suggestions, nil
}

// PrepareStatement prepares and caches a statement for repeated use
func (q *QueryOptimizer) PrepareStatement(ctx context.Context, name, query string) (*sql.Stmt, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("QueryOptimizer").
			WithOperation("PrepareStatement")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if already prepared
	if stmt, exists := q.preparedStatements[name]; exists {
		return stmt, nil
	}

	// Prepare new statement
	stmt, err := q.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare statement").
			WithComponent("QueryOptimizer").
			WithOperation("PrepareStatement").
			WithDetails("name", name).
			WithDetails("query", query)
	}

	q.preparedStatements[name] = stmt
	return stmt, nil
}

// TrackQueryExecution tracks execution statistics for a query
func (q *QueryOptimizer) TrackQueryExecution(query string, duration time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	normalizedQuery := q.normalizeQuery(query)

	stats, exists := q.queryStats[normalizedQuery]
	if !exists {
		stats = &QueryStats{
			Query:       normalizedQuery,
			MinDuration: duration,
			MaxDuration: duration,
		}
		q.queryStats[normalizedQuery] = stats
	}

	stats.ExecutionCount++
	stats.TotalDuration += duration
	stats.AvgDuration = stats.TotalDuration / time.Duration(stats.ExecutionCount)
	stats.LastExecuted = time.Now()

	if duration < stats.MinDuration {
		stats.MinDuration = duration
	}
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	// Log slow queries
	if duration > q.slowQueryThreshold {
		// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	}
}

// GetQueryStats returns statistics for all tracked queries
func (q *QueryOptimizer) GetQueryStats() map[string]*QueryStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := make(map[string]*QueryStats)
	for k, v := range q.queryStats {
		statsCopy[k] = &QueryStats{
			Query:          v.Query,
			ExecutionCount: v.ExecutionCount,
			TotalDuration:  v.TotalDuration,
			AvgDuration:    v.AvgDuration,
			MinDuration:    v.MinDuration,
			MaxDuration:    v.MaxDuration,
			LastExecuted:   v.LastExecuted,
			IndexesUsed:    append([]string{}, v.IndexesUsed...),
		}
	}

	return statsCopy
}

// Close cleans up resources
func (q *QueryOptimizer) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Close all prepared statements
	for name, stmt := range q.preparedStatements {
		if err := stmt.Close(); err != nil {
			// Log error but continue closing others
			_ = gerror.Wrap(err, gerror.ErrCodeConnection, "failed to close prepared statement").
				WithComponent("QueryOptimizer").
				WithOperation("Close").
				WithDetails("statement", name)
		}
	}

	// Clear cache
	if q.cache != nil {
		q.cache.Clear()
	}

	return nil
}

// normalizeQuery normalizes a query for comparison and caching
func (q *QueryOptimizer) normalizeQuery(query string) string {
	// Remove extra whitespace
	query = strings.TrimSpace(query)
	query = strings.Join(strings.Fields(query), " ")

	// Convert to uppercase for consistency
	return strings.ToUpper(query)
}

// extractTableName extracts table name from query plan detail
func (q *QueryOptimizer) extractTableName(detail string) string {
	parts := strings.Fields(detail)
	for i, part := range parts {
		if part == "TABLE" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

// addQueryPatternSuggestions adds suggestions based on query patterns
func (q *QueryOptimizer) addQueryPatternSuggestions(query string, plan *QueryPlan) {
	upper := strings.ToUpper(query)

	// Check for SELECT *
	if strings.Contains(upper, "SELECT *") {
		plan.Suggestions = append(plan.Suggestions, "Consider selecting only required columns instead of SELECT *")
	}

	// Check for LIKE with leading wildcard
	if strings.Contains(upper, "LIKE '%") {
		plan.Suggestions = append(plan.Suggestions, "Leading wildcard in LIKE prevents index usage, consider full-text search")
	}

	// Check for OR conditions
	if strings.Count(upper, " OR ") > 2 {
		plan.Suggestions = append(plan.Suggestions, "Multiple OR conditions may prevent index usage, consider UNION")
	}

	// Check for NOT IN
	if strings.Contains(upper, "NOT IN") {
		plan.Suggestions = append(plan.Suggestions, "NOT IN can be inefficient, consider NOT EXISTS or LEFT JOIN")
	}
}

// findRedundantIndexes finds indexes that are redundant
func (q *QueryOptimizer) findRedundantIndexes(ctx context.Context) ([]string, error) {
	query := `
		SELECT name 
		FROM sqlite_master 
		WHERE type = 'index' 
		AND sql IS NOT NULL
		ORDER BY name
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query indexes").
			WithComponent("QueryOptimizer").
			WithOperation("findRedundantIndexes")
	}
	defer rows.Close()

	var redundant []string
	var indexes []string

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		indexes = append(indexes, name)
	}

	// Simple redundancy check - in production, this would be more sophisticated
	// checking for indexes with overlapping columns
	seen := make(map[string]bool)
	for _, idx := range indexes {
		base := strings.TrimSuffix(idx, "_idx")
		if seen[base] {
			redundant = append(redundant, idx)
		}
		seen[base] = true
	}

	return redundant, nil
}

// findMissingForeignKeyIndexes finds foreign keys without indexes
func (q *QueryOptimizer) findMissingForeignKeyIndexes(ctx context.Context) ([]string, error) {
	// SQLite doesn't have a direct way to query foreign keys
	// This is a simplified implementation
	query := `
		SELECT sql 
		FROM sqlite_master 
		WHERE type = 'table' 
		AND sql LIKE '%FOREIGN KEY%'
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query tables").
			WithComponent("QueryOptimizer").
			WithOperation("findMissingForeignKeyIndexes")
	}
	defer rows.Close()

	var missing []string
	// Simplified check - in production would parse SQL and check against existing indexes

	return missing, nil
}

// sanitizeQueryForMetrics removes sensitive data from queries for metrics
func (q *QueryOptimizer) sanitizeQueryForMetrics(query string) string {
	// Remove literal values
	// This is a simple implementation - production would be more sophisticated
	if len(query) > 50 {
		return query[:50] + "..."
	}
	return query
}
