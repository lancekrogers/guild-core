// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// SyncDirection defines the direction of synchronization
type SyncDirection string

const (
	SyncFromFiles     SyncDirection = "from_files"    // Filesystem → Database
	SyncToFiles       SyncDirection = "to_files"      // Database → Filesystem
	SyncBidirectional SyncDirection = "bidirectional" // Two-way sync
)

// ConflictStrategy defines how to handle conflicts during sync
type ConflictStrategy string

const (
	ConflictStrategyNewest   ConflictStrategy = "newest"    // Keep newest version
	ConflictStrategyOldest   ConflictStrategy = "oldest"    // Keep oldest version
	ConflictStrategyFilesWin ConflictStrategy = "files_win" // Filesystem takes precedence
	ConflictStrategyDBWins   ConflictStrategy = "db_wins"   // Database takes precedence
	ConflictStrategyManual   ConflictStrategy = "manual"    // Require manual resolution
)

// SyncStatus represents the synchronization status of a document
type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "synced"
	SyncStatusPending  SyncStatus = "pending"
	SyncStatusConflict SyncStatus = "conflict"
	SyncStatusError    SyncStatus = "error"
)

// SyncState tracks the state of a document for synchronization
type SyncState struct {
	DocumentID      string
	FilePath        string
	FileChecksum    string
	FileModified    time.Time
	DBChecksum      string
	DBModified      time.Time
	LastSyncedAt    time.Time
	Status          SyncStatus
	ConflictDetails string
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	DocumentsScanned int
	DocumentsSynced  int
	DocumentsSkipped int
	DocumentsErrored int
	Conflicts        []SyncConflict
	Errors           []error
	Duration         time.Duration
}

// SyncConflict represents a synchronization conflict
type SyncConflict struct {
	DocumentID   string
	FilePath     string
	FileModified time.Time
	DBModified   time.Time
	Resolution   string
}

// SyncManagerConfig configures the sync manager
type SyncManagerConfig struct {
	Scanner          *DocumentScanner
	VectorStore      *CorpusVectorStore
	MetadataStore    *MetadataStore
	Direction        SyncDirection
	ConflictStrategy ConflictStrategy
	BatchSize        int
	MaxRetries       int
}

// SyncManager handles bidirectional synchronization
type SyncManager struct {
	scanner          *DocumentScanner
	vectorStore      *CorpusVectorStore
	metaStore        *MetadataStore
	direction        SyncDirection
	conflictStrategy ConflictStrategy
	batchSize        int
	maxRetries       int

	// Sync state tracking
	syncStates map[string]*SyncState
	mu         sync.RWMutex

	// Error queue for retries
	errorQueue []SyncError
	errorMu    sync.Mutex

	// Sync control
	syncInProgress bool
	cancelFunc     context.CancelFunc
}

// SyncError represents an error that can be retried
type SyncError struct {
	DocumentID  string
	Operation   string
	Error       error
	Attempts    int
	LastAttempt time.Time
}

// NewSyncManager creates a new sync manager
func NewSyncManager(config SyncManagerConfig) (*SyncManager, error) {
	if config.Scanner == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "scanner is required", nil).
			WithComponent("corpus.sync").
			WithOperation("NewSyncManager")
	}
	if config.VectorStore == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "vector store is required", nil).
			WithComponent("corpus.sync").
			WithOperation("NewSyncManager")
	}
	if config.MetadataStore == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "metadata store is required", nil).
			WithComponent("corpus.sync").
			WithOperation("NewSyncManager")
	}

	// Set defaults
	if config.Direction == "" {
		config.Direction = SyncFromFiles
	}
	if config.ConflictStrategy == "" {
		config.ConflictStrategy = ConflictStrategyNewest
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &SyncManager{
		scanner:          config.Scanner,
		vectorStore:      config.VectorStore,
		metaStore:        config.MetadataStore,
		direction:        config.Direction,
		conflictStrategy: config.ConflictStrategy,
		batchSize:        config.BatchSize,
		maxRetries:       config.MaxRetries,
		syncStates:       make(map[string]*SyncState),
		errorQueue:       make([]SyncError, 0),
	}, nil
}

// Sync performs a synchronization operation
func (sm *SyncManager) Sync(ctx context.Context) (*SyncResult, error) {
	// Check if sync is already in progress
	sm.mu.Lock()
	if sm.syncInProgress {
		sm.mu.Unlock()
		return nil, gerror.New(gerror.ErrCodeAgentBusy, "sync already in progress", nil).
			WithComponent("corpus.sync").
			WithOperation("Sync")
	}
	sm.syncInProgress = true
	sm.mu.Unlock()

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	sm.cancelFunc = cancel
	defer func() {
		sm.mu.Lock()
		sm.syncInProgress = false
		sm.cancelFunc = nil
		sm.mu.Unlock()
	}()

	startTime := time.Now()
	result := &SyncResult{
		Conflicts: make([]SyncConflict, 0),
		Errors:    make([]error, 0),
	}

	// Load existing sync states from database
	if err := sm.loadSyncStates(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load sync states").
			WithComponent("corpus.sync").
			WithOperation("Sync")
	}

	// Perform sync based on direction
	switch sm.direction {
	case SyncFromFiles:
		err := sm.syncFromFiles(ctx, result)
		if err != nil {
			return result, err
		}
	case SyncToFiles:
		err := sm.syncToFiles(ctx, result)
		if err != nil {
			return result, err
		}
	case SyncBidirectional:
		// First sync from files to catch new/modified files
		err := sm.syncFromFiles(ctx, result)
		if err != nil {
			return result, err
		}
		// Then sync from DB to files for any DB-only content
		err = sm.syncToFiles(ctx, result)
		if err != nil {
			return result, err
		}
	}

	// Process error queue for retries
	sm.processErrorQueue(ctx, result)

	// Save updated sync states
	if err := sm.saveSyncStates(ctx); err != nil {
		result.Errors = append(result.Errors, err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// syncFromFiles syncs from filesystem to database
func (sm *SyncManager) syncFromFiles(ctx context.Context, result *SyncResult) error {
	// Scan filesystem for documents
	scanResults, err := sm.scanner.ScanStream(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to scan files").
			WithComponent("corpus.sync").
			WithOperation("syncFromFiles")
	}

	batch := make([]*ScannedDocument, 0, sm.batchSize)

	for scanResult := range scanResults {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if scanResult.Error != nil {
			result.Errors = append(result.Errors, scanResult.Error)
			result.DocumentsErrored++
			continue
		}

		doc := scanResult.Document
		result.DocumentsScanned++

		// Check if document needs syncing
		needsSync, conflict := sm.checkNeedsSync(doc)
		if conflict != nil {
			result.Conflicts = append(result.Conflicts, *conflict)
			result.DocumentsSkipped++
			continue
		}

		if !needsSync {
			result.DocumentsSkipped++
			continue
		}

		// Add to batch
		batch = append(batch, doc)

		// Process batch when full
		if len(batch) >= sm.batchSize {
			if err := sm.processBatch(ctx, batch, result); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}

	// Process remaining documents
	if len(batch) > 0 {
		if err := sm.processBatch(ctx, batch, result); err != nil {
			return err
		}
	}

	return nil
}

// syncToFiles syncs from database to filesystem
func (sm *SyncManager) syncToFiles(ctx context.Context, result *SyncResult) error {
	// Get all documents from metadata store
	documents, err := sm.metaStore.ListDocuments(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list documents from database").
			WithComponent("corpus.sync").
			WithOperation("syncToFiles")
	}

	for _, docMeta := range documents {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result.DocumentsScanned++

		// Check if file exists and needs update
		state, exists := sm.syncStates[docMeta.ID]
		if !exists {
			// Document only in DB, write to filesystem
			if err := sm.writeDocumentToFile(ctx, docMeta); err != nil {
				sm.addToErrorQueue(docMeta.ID, "write_to_file", err)
				result.DocumentsErrored++
				continue
			}
			result.DocumentsSynced++
		} else if state.DBModified.After(state.FileModified) {
			// DB is newer, update file
			if err := sm.writeDocumentToFile(ctx, docMeta); err != nil {
				sm.addToErrorQueue(docMeta.ID, "update_file", err)
				result.DocumentsErrored++
				continue
			}
			result.DocumentsSynced++
		} else {
			result.DocumentsSkipped++
		}
	}

	return nil
}

// checkNeedsSync determines if a document needs syncing
func (sm *SyncManager) checkNeedsSync(doc *ScannedDocument) (bool, *SyncConflict) {
	sm.mu.RLock()
	state, exists := sm.syncStates[doc.ID]
	sm.mu.RUnlock()

	if !exists {
		// New document, needs sync
		return true, nil
	}

	// Check if file has changed
	if doc.Checksum == state.FileChecksum {
		// File hasn't changed
		return false, nil
	}

	// File has changed, check for conflicts
	if state.DBChecksum != state.FileChecksum && state.DBModified.After(state.LastSyncedAt) {
		// Both file and DB have changed since last sync - conflict!
		conflict := &SyncConflict{
			DocumentID:   doc.ID,
			FilePath:     doc.Path,
			FileModified: doc.LastModified,
			DBModified:   state.DBModified,
		}

		// Resolve conflict based on strategy
		switch sm.conflictStrategy {
		case ConflictStrategyNewest:
			if doc.LastModified.After(state.DBModified) {
				conflict.Resolution = "file_wins"
				return true, nil // Proceed with sync
			}
			conflict.Resolution = "db_wins"
			return false, conflict
		case ConflictStrategyFilesWin:
			conflict.Resolution = "file_wins"
			return true, nil
		case ConflictStrategyDBWins:
			conflict.Resolution = "db_wins"
			return false, conflict
		case ConflictStrategyManual:
			conflict.Resolution = "manual_required"
			return false, conflict
		default:
			conflict.Resolution = "skipped"
			return false, conflict
		}
	}

	// Only file has changed, safe to sync
	return true, nil
}

// processBatch processes a batch of documents
func (sm *SyncManager) processBatch(ctx context.Context, batch []*ScannedDocument, result *SyncResult) error {
	// Index documents in vector store
	if err := sm.vectorStore.IndexDocuments(ctx, batch); err != nil {
		// Add individual documents to error queue
		for _, doc := range batch {
			sm.addToErrorQueue(doc.ID, "index_document", err)
		}
		result.DocumentsErrored += len(batch)
		return nil // Don't fail entire sync
	}

	// Update metadata store
	for _, doc := range batch {
		docMeta := &StoredDocument{
			ID:           doc.ID,
			Path:         doc.Path,
			Type:         doc.Type,
			Title:        doc.Metadata.Title,
			Description:  doc.Metadata.Description,
			Tags:         doc.Metadata.ExtractedTags,
			Checksum:     doc.Checksum,
			LastModified: doc.LastModified,
			LastIndexed:  time.Now(),
			Metadata:     convertDocumentMetadata(doc.Metadata),
		}

		if err := sm.metaStore.UpsertDocument(ctx, docMeta); err != nil {
			sm.addToErrorQueue(doc.ID, "update_metadata", err)
			result.DocumentsErrored++
			continue
		}

		// Update sync state
		sm.updateSyncState(doc.ID, doc.Path, doc.Checksum, doc.LastModified, doc.Checksum, doc.LastModified)
		result.DocumentsSynced++
	}

	return nil
}

// writeDocumentToFile writes a document from DB to filesystem
func (sm *SyncManager) writeDocumentToFile(ctx context.Context, docMeta *StoredDocument) error {
	// Get full document content from somewhere (this would need to be implemented)
	// For now, return not implemented
	return gerror.New(gerror.ErrCodeInternal, "write to files not yet implemented", nil).
		WithComponent("corpus.sync").
		WithOperation("writeDocumentToFile")
}

// updateSyncState updates the sync state for a document
func (sm *SyncManager) updateSyncState(docID, filePath, fileChecksum string, fileModified time.Time, dbChecksum string, dbModified time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.syncStates[docID] = &SyncState{
		DocumentID:   docID,
		FilePath:     filePath,
		FileChecksum: fileChecksum,
		FileModified: fileModified,
		DBChecksum:   dbChecksum,
		DBModified:   dbModified,
		LastSyncedAt: time.Now(),
		Status:       SyncStatusSynced,
	}
}

// addToErrorQueue adds an error for retry
func (sm *SyncManager) addToErrorQueue(docID, operation string, err error) {
	sm.errorMu.Lock()
	defer sm.errorMu.Unlock()

	sm.errorQueue = append(sm.errorQueue, SyncError{
		DocumentID:  docID,
		Operation:   operation,
		Error:       err,
		Attempts:    1,
		LastAttempt: time.Now(),
	})
}

// processErrorQueue retries failed operations
func (sm *SyncManager) processErrorQueue(ctx context.Context, result *SyncResult) {
	sm.errorMu.Lock()
	queue := make([]SyncError, len(sm.errorQueue))
	copy(queue, sm.errorQueue)
	sm.errorQueue = sm.errorQueue[:0]
	sm.errorMu.Unlock()

	for _, syncErr := range queue {
		if syncErr.Attempts >= sm.maxRetries {
			result.Errors = append(result.Errors,
				gerror.Wrapf(syncErr.Error, gerror.ErrCodeInternal,
					"failed after %d attempts", syncErr.Attempts).
					WithComponent("corpus.sync").
					WithOperation("processErrorQueue").
					WithDetails("document_id", syncErr.DocumentID))
			continue
		}

		// Wait before retry (exponential backoff)
		waitTime := time.Duration(syncErr.Attempts) * time.Second
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return
		}

		// Retry operation (simplified - would need proper retry logic)
		syncErr.Attempts++
		syncErr.LastAttempt = time.Now()
		sm.errorQueue = append(sm.errorQueue, syncErr)
	}
}

// loadSyncStates loads sync states from the metadata store
func (sm *SyncManager) loadSyncStates(ctx context.Context) error {
	states, err := sm.metaStore.GetSyncStates(ctx)
	if err != nil {
		return err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.syncStates = make(map[string]*SyncState)
	for _, state := range states {
		sm.syncStates[state.DocumentID] = state
	}

	return nil
}

// saveSyncStates saves sync states to the metadata store
func (sm *SyncManager) saveSyncStates(ctx context.Context) error {
	sm.mu.RLock()
	states := make([]*SyncState, 0, len(sm.syncStates))
	for _, state := range sm.syncStates {
		states = append(states, state)
	}
	sm.mu.RUnlock()

	return sm.metaStore.SaveSyncStates(ctx, states)
}

// Stop cancels any ongoing sync operation
func (sm *SyncManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.cancelFunc != nil {
		sm.cancelFunc()
	}
}

// GetSyncStatus returns the current sync status
func (sm *SyncManager) GetSyncStatus(ctx context.Context) (map[string]interface{}, error) {
	sm.mu.RLock()
	inProgress := sm.syncInProgress
	stateCount := len(sm.syncStates)
	sm.mu.RUnlock()

	sm.errorMu.Lock()
	errorCount := len(sm.errorQueue)
	sm.errorMu.Unlock()

	// Count sync states by status
	statusCounts := make(map[SyncStatus]int)
	sm.mu.RLock()
	for _, state := range sm.syncStates {
		statusCounts[state.Status]++
	}
	sm.mu.RUnlock()

	return map[string]interface{}{
		"in_progress":       inProgress,
		"total_documents":   stateCount,
		"error_queue":       errorCount,
		"status_counts":     statusCounts,
		"direction":         string(sm.direction),
		"conflict_strategy": string(sm.conflictStrategy),
	}, nil
}

// StartContinuousSync starts continuous synchronization in the background
func (sm *SyncManager) StartContinuousSync(ctx context.Context, interval time.Duration) error {
	if interval < time.Minute {
		interval = time.Minute // Minimum interval
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Run sync
				_, err := sm.Sync(ctx)
				if err != nil {
					// Log error but continue
					fmt.Printf("Continuous sync error: %v\n", err)
				}
			}
		}
	}()

	return nil
}

// Helper function to convert DocumentMetadata to map
func convertDocumentMetadata(meta DocumentMetadata) map[string]interface{} {
	return map[string]interface{}{
		"title":            meta.Title,
		"description":      meta.Description,
		"language":         meta.Language,
		"word_count":       meta.WordCount,
		"code_block_count": meta.CodeBlockCount,
		"link_count":       meta.LinkCount,
		"heading_count":    meta.HeadingCount,
		"todo_count":       meta.TODOCount,
		"file_size":        meta.FileSize,
		"checksum":         meta.Checksum,
		"extracted_tags":   meta.ExtractedTags,
	}
}
