// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// CompressionManager handles data compression and archival
type CompressionManager struct {
	db          *sql.DB
	metrics     *observability.MetricsRegistry
	config      CompressionConfig
	archivePath string
}

// CompressionConfig configures compression behavior
type CompressionConfig struct {
	Enabled               bool
	CompressionLevel      int   // gzip compression level (1-9)
	MinSizeForCompression int64 // Minimum size in bytes to compress
	ArchiveAfterDays      int   // Days before archiving
	CompressJSONFields    bool  // Compress JSON fields automatically
	CompressBLOBs         bool  // Compress BLOB fields
}

// DefaultCompressionConfig returns default compression settings
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Enabled:               true,
		CompressionLevel:      gzip.DefaultCompression,
		MinSizeForCompression: 1024, // 1KB
		ArchiveAfterDays:      90,
		CompressJSONFields:    true,
		CompressBLOBs:         true,
	}
}

// CompressionStats tracks compression statistics
type CompressionStats struct {
	TotalCompressed        int64
	TotalDecompressed      int64
	BytesSaved             int64
	CompressionRatio       float64
	AverageCompressionTime time.Duration
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(db *sql.DB, metrics *observability.MetricsRegistry, config CompressionConfig) (*CompressionManager, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database is required", nil).
			WithComponent("CompressionManager")
	}

	if config.CompressionLevel < gzip.NoCompression || config.CompressionLevel > gzip.BestCompression {
		config.CompressionLevel = gzip.DefaultCompression
	}

	return &CompressionManager{
		db:      db,
		metrics: metrics,
		config:  config,
	}, nil
}

// CompressData compresses data using gzip
func (c *CompressionManager) CompressData(ctx context.Context, data []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("CompressData")
	}

	// Skip compression for small data
	if int64(len(data)) < c.config.MinSizeForCompression {
		return data, nil
	}

	start := time.Now()
	var buf bytes.Buffer

	writer, err := gzip.NewWriterLevel(&buf, c.config.CompressionLevel)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create gzip writer").
			WithComponent("CompressionManager").
			WithOperation("CompressData")
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compress data").
			WithComponent("CompressionManager").
			WithOperation("CompressData")
	}

	if err := writer.Close(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close gzip writer").
			WithComponent("CompressionManager").
			WithOperation("CompressData")
	}

	compressed := buf.Bytes()

	// Track metrics
	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = start

	return compressed, nil
}

// DecompressData decompresses gzip data
func (c *CompressionManager) DecompressData(ctx context.Context, data []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("DecompressData")
	}

	// Check if data is actually compressed (gzip magic number)
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		// Not compressed, return as-is
		return data, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create gzip reader").
			WithComponent("CompressionManager").
			WithOperation("DecompressData")
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decompress data").
			WithComponent("CompressionManager").
			WithOperation("DecompressData")
	}

	return decompressed, nil
}

// CompressJSON compresses JSON data efficiently
func (c *CompressionManager) CompressJSON(ctx context.Context, v interface{}) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("CompressJSON")
	}

	// Marshal to JSON
	data, err := json.Marshal(v)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").
			WithComponent("CompressionManager").
			WithOperation("CompressJSON")
	}

	// Compress
	return c.CompressData(ctx, data)
}

// DecompressJSON decompresses and unmarshals JSON data
func (c *CompressionManager) DecompressJSON(ctx context.Context, data []byte, v interface{}) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("DecompressJSON")
	}

	// Decompress
	decompressed, err := c.DecompressData(ctx, data)
	if err != nil {
		return err
	}

	// Unmarshal
	if err := json.Unmarshal(decompressed, v); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal JSON").
			WithComponent("CompressionManager").
			WithOperation("DecompressJSON")
	}

	return nil
}

// ArchiveOldData moves old data to compressed archive tables
func (c *CompressionManager) ArchiveOldData(ctx context.Context, tableName string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("ArchiveOldData")
	}

	// Begin transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeTransaction, "failed to begin transaction").
			WithComponent("CompressionManager").
			WithOperation("ArchiveOldData")
	}
	defer tx.Rollback()

	// Create archive table if it doesn't exist
	archiveTable := fmt.Sprintf("%s_archive", tableName)
	if err := c.createArchiveTable(ctx, tx, tableName, archiveTable); err != nil {
		return err
	}

	// Move old data
	cutoffDate := time.Now().AddDate(0, 0, -c.config.ArchiveAfterDays)
	query := fmt.Sprintf(`
		INSERT INTO %s 
		SELECT * FROM %s 
		WHERE created_at < ?
	`, archiveTable, tableName)

	result, err := tx.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to archive data").
			WithComponent("CompressionManager").
			WithOperation("ArchiveOldData").
			WithDetails("table", tableName)
	}

	rowsArchived, _ := result.RowsAffected()

	// Delete archived data from main table
	deleteQuery := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE created_at < ?
	`, tableName)

	if _, err := tx.ExecContext(ctx, deleteQuery, cutoffDate); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete archived data").
			WithComponent("CompressionManager").
			WithOperation("ArchiveOldData").
			WithDetails("table", tableName)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeTransaction, "failed to commit transaction").
			WithComponent("CompressionManager").
			WithOperation("ArchiveOldData")
	}

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = rowsArchived

	return nil
}

// OptimizeBLOBStorage optimizes BLOB storage by compressing large values
func (c *CompressionManager) OptimizeBLOBStorage(ctx context.Context, tableName, blobColumn string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("OptimizeBLOBStorage")
	}

	if !c.config.CompressBLOBs {
		return nil
	}

	// Query for uncompressed BLOBs
	query := fmt.Sprintf(`
		SELECT id, %s 
		FROM %s 
		WHERE length(%s) > ? 
		AND substr(%s, 1, 2) != X'1f8b'  -- Not already gzipped
		LIMIT 100
	`, blobColumn, tableName, blobColumn, blobColumn)

	rows, err := c.db.QueryContext(ctx, query, c.config.MinSizeForCompression)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query BLOBs").
			WithComponent("CompressionManager").
			WithOperation("OptimizeBLOBStorage").
			WithDetails("table", tableName).
			WithDetails("column", blobColumn)
	}
	defer rows.Close()

	updateQuery := fmt.Sprintf(`UPDATE %s SET %s = ? WHERE id = ?`, tableName, blobColumn)
	updateStmt, err := c.db.PrepareContext(ctx, updateQuery)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to prepare update statement").
			WithComponent("CompressionManager").
			WithOperation("OptimizeBLOBStorage")
	}
	defer updateStmt.Close()

	compressed := 0
	for rows.Next() {
		var id string
		var data []byte

		if err := rows.Scan(&id, &data); err != nil {
			continue
		}

		// Compress the data
		compressedData, err := c.CompressData(ctx, data)
		if err != nil {
			continue
		}

		// Only update if compression saved space
		if len(compressedData) < len(data) {
			if _, err := updateStmt.ExecContext(ctx, compressedData, id); err == nil {
				compressed++
			}
		}
	}

	// TODO: Implement metrics tracking when MetricsRegistry supports generic methods
	_ = compressed

	return nil
}

// GetCompressionStats returns compression statistics
func (c *CompressionManager) GetCompressionStats(ctx context.Context) (*CompressionStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CompressionManager").
			WithOperation("GetCompressionStats")
	}

	// This is a simplified implementation
	// In production, would track detailed statistics
	stats := &CompressionStats{
		CompressionRatio: 0.7, // Example: 30% compression
	}

	return stats, nil
}

// createArchiveTable creates an archive table with the same schema
func (c *CompressionManager) createArchiveTable(ctx context.Context, tx *sql.Tx, sourceTable, archiveTable string) error {
	// Get source table schema
	var createSQL string
	query := `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`

	if err := tx.QueryRowContext(ctx, query, sourceTable).Scan(&createSQL); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get table schema").
			WithComponent("CompressionManager").
			WithOperation("createArchiveTable").
			WithDetails("table", sourceTable)
	}

	// Replace table name in CREATE statement
	archiveSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s", archiveTable)
	if idx := bytes.Index([]byte(createSQL), []byte("(")); idx > 0 {
		archiveSQL += string(createSQL[idx:])
	}

	// Create archive table
	if _, err := tx.ExecContext(ctx, archiveSQL); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create archive table").
			WithComponent("CompressionManager").
			WithOperation("createArchiveTable").
			WithDetails("table", archiveTable)
	}

	return nil
}
