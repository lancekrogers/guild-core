// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// MetadataStore manages corpus metadata in SQLite
type MetadataStore struct {
	db *sql.DB
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore(db *sql.DB) (*MetadataStore, error) {
	if db == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "database connection is required", nil).
			WithComponent("corpus.metadata").
			WithOperation("NewMetadataStore")
	}

	return &MetadataStore{db: db}, nil
}

// CreateTables creates the necessary database tables
func (ms *MetadataStore) CreateTables(ctx context.Context) error {
	queries := []string{
		// Corpus documents table
		`CREATE TABLE IF NOT EXISTS corpus_documents (
			id TEXT PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			type TEXT NOT NULL,
			title TEXT,
			description TEXT,
			checksum TEXT NOT NULL,
			last_modified TIMESTAMP NOT NULL,
			last_indexed TIMESTAMP,
			metadata JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index on path for fast lookups
		`CREATE INDEX IF NOT EXISTS idx_corpus_documents_path ON corpus_documents(path)`,

		// Index on type for filtering
		`CREATE INDEX IF NOT EXISTS idx_corpus_documents_type ON corpus_documents(type)`,

		// Index on last_modified for change detection
		`CREATE INDEX IF NOT EXISTS idx_corpus_documents_modified ON corpus_documents(last_modified)`,

		// Corpus chunks table
		`CREATE TABLE IF NOT EXISTS corpus_chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			start_offset INTEGER NOT NULL,
			end_offset INTEGER NOT NULL,
			content_hash TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (document_id) REFERENCES corpus_documents(id) ON DELETE CASCADE,
			UNIQUE(document_id, chunk_index)
		)`,

		// Index for finding chunks by document
		`CREATE INDEX IF NOT EXISTS idx_corpus_chunks_document ON corpus_chunks(document_id)`,

		// Corpus knowledge table
		`CREATE TABLE IF NOT EXISTS corpus_knowledge (
			id TEXT PRIMARY KEY,
			source_type TEXT NOT NULL, -- 'chat', 'code_analysis', 'manual'
			source_id TEXT,
			content TEXT NOT NULL,
			confidence REAL DEFAULT 1.0,
			tags TEXT, -- JSON array
			metadata JSON,
			extracted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			validated_at TIMESTAMP,
			validation_status TEXT DEFAULT 'pending' -- 'pending', 'validated', 'rejected'
		)`,

		// Index on source for tracking origin
		`CREATE INDEX IF NOT EXISTS idx_corpus_knowledge_source ON corpus_knowledge(source_type, source_id)`,

		// Index on validation status
		`CREATE INDEX IF NOT EXISTS idx_corpus_knowledge_status ON corpus_knowledge(validation_status)`,

		// Sync states table
		`CREATE TABLE IF NOT EXISTS corpus_sync_states (
			document_id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			file_checksum TEXT,
			file_modified TIMESTAMP,
			db_checksum TEXT,
			db_modified TIMESTAMP,
			last_synced_at TIMESTAMP,
			status TEXT DEFAULT 'pending',
			conflict_details TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Document tags table (normalized)
		`CREATE TABLE IF NOT EXISTS corpus_document_tags (
			document_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (document_id, tag),
			FOREIGN KEY (document_id) REFERENCES corpus_documents(id) ON DELETE CASCADE
		)`,

		// Index on tags for filtering
		`CREATE INDEX IF NOT EXISTS idx_corpus_document_tags_tag ON corpus_document_tags(tag)`,
	}

	// Execute queries in a transaction
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to begin transaction").
			WithComponent("corpus.metadata").
			WithOperation("CreateTables")
	}
	defer tx.Rollback()

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create table").
				WithComponent("corpus.metadata").
				WithOperation("CreateTables").
				WithDetails("query", query[:50]+"...")
		}
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to commit transaction").
			WithComponent("corpus.metadata").
			WithOperation("CreateTables")
	}

	return nil
}

// StoredDocument represents document metadata in the database
type StoredDocument struct {
	ID           string
	Path         string
	Type         ContentType
	Title        string
	Description  string
	Tags         []string
	Checksum     string
	LastModified time.Time
	LastIndexed  time.Time
	Metadata     map[string]interface{}
}

// UpsertDocument inserts or updates a document
func (ms *MetadataStore) UpsertDocument(ctx context.Context, doc *StoredDocument) error {
	if doc == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "document cannot be nil", nil).
			WithComponent("corpus.metadata").
			WithOperation("UpsertDocument")
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal metadata").
			WithComponent("corpus.metadata").
			WithOperation("UpsertDocument")
	}

	// Begin transaction
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to begin transaction").
			WithComponent("corpus.metadata").
			WithOperation("UpsertDocument")
	}
	defer tx.Rollback()

	// Upsert document
	query := `
		INSERT INTO corpus_documents (
			id, path, type, title, description, checksum, 
			last_modified, last_indexed, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path = excluded.path,
			type = excluded.type,
			title = excluded.title,
			description = excluded.description,
			checksum = excluded.checksum,
			last_modified = excluded.last_modified,
			last_indexed = excluded.last_indexed,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err = tx.ExecContext(ctx, query,
		doc.ID, doc.Path, string(doc.Type), doc.Title, doc.Description,
		doc.Checksum, doc.LastModified, doc.LastIndexed, metadataJSON,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to upsert document").
			WithComponent("corpus.metadata").
			WithOperation("UpsertDocument").
			WithDetails("document_id", doc.ID)
	}

	// Update tags
	if len(doc.Tags) > 0 {
		// Delete existing tags
		if _, err := tx.ExecContext(ctx, "DELETE FROM corpus_document_tags WHERE document_id = ?", doc.ID); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete existing tags").
				WithComponent("corpus.metadata").
				WithOperation("UpsertDocument")
		}

		// Insert new tags
		tagQuery := "INSERT INTO corpus_document_tags (document_id, tag) VALUES (?, ?)"
		for _, tag := range doc.Tags {
			if _, err := tx.ExecContext(ctx, tagQuery, doc.ID, tag); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to insert tag").
					WithComponent("corpus.metadata").
					WithOperation("UpsertDocument").
					WithDetails("tag", tag)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to commit transaction").
			WithComponent("corpus.metadata").
			WithOperation("UpsertDocument")
	}

	return nil
}

// GetDocument retrieves a document by ID
func (ms *MetadataStore) GetDocument(ctx context.Context, id string) (*StoredDocument, error) {
	query := `
		SELECT id, path, type, title, description, checksum, 
		       last_modified, last_indexed, metadata
		FROM corpus_documents
		WHERE id = ?
	`

	var doc StoredDocument
	var metadataJSON []byte
	var lastIndexed sql.NullTime
	var description sql.NullString

	err := ms.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.Path, &doc.Type, &doc.Title, &description,
		&doc.Checksum, &doc.LastModified, &lastIndexed, &metadataJSON,
	)
	if err == sql.ErrNoRows {
		return nil, gerror.New(gerror.ErrCodeNotFound, "document not found", nil).
			WithComponent("corpus.metadata").
			WithOperation("GetDocument").
			WithDetails("document_id", id)
	}
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query document").
			WithComponent("corpus.metadata").
			WithOperation("GetDocument")
	}

	if description.Valid {
		doc.Description = description.String
	}
	if lastIndexed.Valid {
		doc.LastIndexed = lastIndexed.Time
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal metadata").
				WithComponent("corpus.metadata").
				WithOperation("GetDocument")
		}
	}

	// Get tags
	tags, err := ms.getDocumentTags(ctx, id)
	if err != nil {
		return nil, err
	}
	doc.Tags = tags

	return &doc, nil
}

// ListDocuments returns all documents
func (ms *MetadataStore) ListDocuments(ctx context.Context) ([]*StoredDocument, error) {
	query := `
		SELECT id, path, type, title, description, checksum, 
		       last_modified, last_indexed, metadata
		FROM corpus_documents
		ORDER BY last_modified DESC
	`

	rows, err := ms.db.QueryContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query documents").
			WithComponent("corpus.metadata").
			WithOperation("ListDocuments")
	}
	defer rows.Close()

	var documents []*StoredDocument
	for rows.Next() {
		var doc StoredDocument
		var metadataJSON []byte
		var lastIndexed sql.NullTime
		var description sql.NullString

		err := rows.Scan(
			&doc.ID, &doc.Path, &doc.Type, &doc.Title, &description,
			&doc.Checksum, &doc.LastModified, &lastIndexed, &metadataJSON,
		)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan document").
				WithComponent("corpus.metadata").
				WithOperation("ListDocuments")
		}

		if description.Valid {
			doc.Description = description.String
		}
		if lastIndexed.Valid {
			doc.LastIndexed = lastIndexed.Time
		}

		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &doc.Metadata); err != nil {
				// Log error but continue
				doc.Metadata = map[string]interface{}{}
			}
		}

		documents = append(documents, &doc)
	}

	// Get tags for all documents
	for _, doc := range documents {
		tags, err := ms.getDocumentTags(ctx, doc.ID)
		if err != nil {
			// Log error but continue
			doc.Tags = []string{}
		} else {
			doc.Tags = tags
		}
	}

	return documents, nil
}

// DeleteDocument removes a document and its chunks
func (ms *MetadataStore) DeleteDocument(ctx context.Context, id string) error {
	// Foreign key constraints will handle cascade deletion
	query := "DELETE FROM corpus_documents WHERE id = ?"
	
	result, err := ms.db.ExecContext(ctx, query, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete document").
			WithComponent("corpus.metadata").
			WithOperation("DeleteDocument").
			WithDetails("document_id", id)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get rows affected").
			WithComponent("corpus.metadata").
			WithOperation("DeleteDocument")
	}

	if rowsAffected == 0 {
		return gerror.New(gerror.ErrCodeNotFound, "document not found", nil).
			WithComponent("corpus.metadata").
			WithOperation("DeleteDocument").
			WithDetails("document_id", id)
	}

	return nil
}

// RecordChunk records a vector chunk mapping
func (ms *MetadataStore) RecordChunk(ctx context.Context, chunk *DocumentChunk) error {
	query := `
		INSERT INTO corpus_chunks (
			id, document_id, chunk_index, start_offset, end_offset, content_hash
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(document_id, chunk_index) DO UPDATE SET
			start_offset = excluded.start_offset,
			end_offset = excluded.end_offset,
			content_hash = excluded.content_hash
	`

	// Calculate content hash
	contentHash := calculateStringChecksum(chunk.Content)

	_, err := ms.db.ExecContext(ctx, query,
		chunk.ID, chunk.DocumentID, chunk.ChunkIndex,
		chunk.StartOffset, chunk.EndOffset, contentHash,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to record chunk").
			WithComponent("corpus.metadata").
			WithOperation("RecordChunk").
			WithDetails("chunk_id", chunk.ID)
	}

	return nil
}

// GetDocumentChunks retrieves chunks for a document
func (ms *MetadataStore) GetDocumentChunks(ctx context.Context, documentID string) ([]*ChunkMetadata, error) {
	query := `
		SELECT id, chunk_index, start_offset, end_offset, content_hash, created_at
		FROM corpus_chunks
		WHERE document_id = ?
		ORDER BY chunk_index
	`

	rows, err := ms.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query chunks").
			WithComponent("corpus.metadata").
			WithOperation("GetDocumentChunks")
	}
	defer rows.Close()

	var chunks []*ChunkMetadata
	for rows.Next() {
		var chunk ChunkMetadata
		err := rows.Scan(
			&chunk.ID, &chunk.ChunkIndex, &chunk.StartOffset,
			&chunk.EndOffset, &chunk.ContentHash, &chunk.CreatedAt,
		)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan chunk").
				WithComponent("corpus.metadata").
				WithOperation("GetDocumentChunks")
		}
		chunk.DocumentID = documentID
		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

// AddKnowledge adds extracted knowledge
func (ms *MetadataStore) AddKnowledge(ctx context.Context, knowledge *CorpusKnowledge) error {
	if knowledge.ID == "" {
		knowledge.ID = generateID()
	}

	tagsJSON, err := json.Marshal(knowledge.Tags)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal tags").
			WithComponent("corpus.metadata").
			WithOperation("AddKnowledge")
	}

	metadataJSON, err := json.Marshal(knowledge.Metadata)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal metadata").
			WithComponent("corpus.metadata").
			WithOperation("AddKnowledge")
	}

	query := `
		INSERT INTO corpus_knowledge (
			id, source_type, source_id, content, confidence, tags, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = ms.db.ExecContext(ctx, query,
		knowledge.ID, knowledge.SourceType, knowledge.SourceID,
		knowledge.Content, knowledge.Confidence, tagsJSON, metadataJSON,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to insert knowledge").
			WithComponent("corpus.metadata").
			WithOperation("AddKnowledge")
	}

	return nil
}

// GetSyncStates retrieves all sync states
func (ms *MetadataStore) GetSyncStates(ctx context.Context) ([]*SyncState, error) {
	query := `
		SELECT document_id, file_path, file_checksum, file_modified,
		       db_checksum, db_modified, last_synced_at, status, conflict_details
		FROM corpus_sync_states
	`

	rows, err := ms.db.QueryContext(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query sync states").
			WithComponent("corpus.metadata").
			WithOperation("GetSyncStates")
	}
	defer rows.Close()

	var states []*SyncState
	for rows.Next() {
		var state SyncState
		var fileModified, dbModified, lastSynced sql.NullTime
		var conflictDetails sql.NullString

		err := rows.Scan(
			&state.DocumentID, &state.FilePath, &state.FileChecksum, &fileModified,
			&state.DBChecksum, &dbModified, &lastSynced, &state.Status, &conflictDetails,
		)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan sync state").
				WithComponent("corpus.metadata").
				WithOperation("GetSyncStates")
		}

		if fileModified.Valid {
			state.FileModified = fileModified.Time
		}
		if dbModified.Valid {
			state.DBModified = dbModified.Time
		}
		if lastSynced.Valid {
			state.LastSyncedAt = lastSynced.Time
		}
		if conflictDetails.Valid {
			state.ConflictDetails = conflictDetails.String
		}

		states = append(states, &state)
	}

	return states, nil
}

// SaveSyncStates saves sync states
func (ms *MetadataStore) SaveSyncStates(ctx context.Context, states []*SyncState) error {
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to begin transaction").
			WithComponent("corpus.metadata").
			WithOperation("SaveSyncStates")
	}
	defer tx.Rollback()

	query := `
		INSERT INTO corpus_sync_states (
			document_id, file_path, file_checksum, file_modified,
			db_checksum, db_modified, last_synced_at, status, conflict_details
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(document_id) DO UPDATE SET
			file_path = excluded.file_path,
			file_checksum = excluded.file_checksum,
			file_modified = excluded.file_modified,
			db_checksum = excluded.db_checksum,
			db_modified = excluded.db_modified,
			last_synced_at = excluded.last_synced_at,
			status = excluded.status,
			conflict_details = excluded.conflict_details,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, state := range states {
		_, err := tx.ExecContext(ctx, query,
			state.DocumentID, state.FilePath, state.FileChecksum, state.FileModified,
			state.DBChecksum, state.DBModified, state.LastSyncedAt, state.Status, state.ConflictDetails,
		)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save sync state").
				WithComponent("corpus.metadata").
				WithOperation("SaveSyncStates").
				WithDetails("document_id", state.DocumentID)
		}
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to commit transaction").
			WithComponent("corpus.metadata").
			WithOperation("SaveSyncStates")
	}

	return nil
}

// GetStats returns corpus statistics
func (ms *MetadataStore) GetStats(ctx context.Context) (*CorpusStats, error) {
	stats := &CorpusStats{}

	// Get document count by type
	typeQuery := `
		SELECT type, COUNT(*) 
		FROM corpus_documents 
		GROUP BY type
	`
	rows, err := ms.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query document types").
			WithComponent("corpus.metadata").
			WithOperation("GetStats")
	}
	defer rows.Close()

	stats.DocumentsByType = make(map[ContentType]int)
	for rows.Next() {
		var docType string
		var count int
		if err := rows.Scan(&docType, &count); err != nil {
			continue
		}
		stats.DocumentsByType[ContentType(docType)] = count
		stats.TotalDocuments += count
	}

	// Get total chunks
	err = ms.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM corpus_chunks").Scan(&stats.TotalChunks)
	if err != nil {
		// Ignore error, continue with other stats
		stats.TotalChunks = 0
	}

	// Get knowledge count
	err = ms.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM corpus_knowledge WHERE validation_status = 'validated'").Scan(&stats.TotalKnowledge)
	if err != nil {
		stats.TotalKnowledge = 0
	}

	// Get last sync time
	var lastSync sql.NullTime
	err = ms.db.QueryRowContext(ctx, "SELECT MAX(last_synced_at) FROM corpus_sync_states").Scan(&lastSync)
	if err == nil && lastSync.Valid {
		stats.LastSyncTime = lastSync.Time
	}

	return stats, nil
}

// Helper functions

func (ms *MetadataStore) getDocumentTags(ctx context.Context, documentID string) ([]string, error) {
	query := "SELECT tag FROM corpus_document_tags WHERE document_id = ? ORDER BY tag"
	
	rows, err := ms.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query tags").
			WithComponent("corpus.metadata").
			WithOperation("getDocumentTags")
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan tag").
				WithComponent("corpus.metadata").
				WithOperation("getDocumentTags")
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// ChunkMetadata represents metadata for a document chunk
type ChunkMetadata struct {
	ID          string
	DocumentID  string
	ChunkIndex  int
	StartOffset int
	EndOffset   int
	ContentHash string
	CreatedAt   time.Time
}

// CorpusKnowledge represents extracted knowledge
type CorpusKnowledge struct {
	ID               string
	SourceType       string
	SourceID         string
	Content          string
	Confidence       float64
	Tags             []string
	Metadata         map[string]interface{}
	ExtractedAt      time.Time
	ValidatedAt      *time.Time
	ValidationStatus string
}

// CorpusStats contains corpus statistics
type CorpusStats struct {
	TotalDocuments  int
	TotalChunks     int
	TotalKnowledge  int
	DocumentsByType map[ContentType]int
	LastSyncTime    time.Time
}

// Helper functions

func calculateStringChecksum(content string) string {
	return calculateChecksum([]byte(content))
}

func calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func generateID() string {
	return uuid.New().String()
}