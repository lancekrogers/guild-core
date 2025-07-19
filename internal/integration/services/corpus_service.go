// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// Placeholder interfaces until corpus package is fully implemented
// These should be moved to pkg/corpus when ready

// DocumentIndex provides indexing and search capabilities
type DocumentIndex interface {
	Open() error
	Close() error
	Health(ctx context.Context) error
	Index(ctx context.Context, doc *corpus.ScannedDocument) error
	Search(ctx context.Context, query string, options SearchOptions) ([]SearchResult, error)
}

// SearchOptions configures document search
type SearchOptions struct {
	Limit      int
	Offset     int
	FileTypes  []string
	SortBy     string
	MaxResults int
}

// SearchResult represents a search result
type SearchResult struct {
	Document *corpus.ScannedDocument
	Score    float64
	Snippet  string
}

// IndexOptions configures the document index
type IndexOptions struct {
	EnableFullText bool
	EnableVector   bool
}

// placeholderIndex is a temporary implementation
type placeholderIndex struct{}

func (p *placeholderIndex) Open() error                                                  { return nil }
func (p *placeholderIndex) Close() error                                                 { return nil }
func (p *placeholderIndex) Health(ctx context.Context) error                             { return nil }
func (p *placeholderIndex) Index(ctx context.Context, doc *corpus.ScannedDocument) error { return nil }
func (p *placeholderIndex) Search(ctx context.Context, query string, options SearchOptions) ([]SearchResult, error) {
	return []SearchResult{}, nil
}

// NewDocumentIndex creates a new document index (placeholder)
func NewDocumentIndex(path string, options IndexOptions) (DocumentIndex, error) {
	// TODO: Implement actual index when corpus package is ready
	return &placeholderIndex{}, nil
}

// CorpusService wraps corpus scanning operations to integrate with the service framework
type CorpusService struct {
	scanner  *corpus.DocumentScanner
	index    DocumentIndex
	registry registry.ComponentRegistry
	eventBus events.EventBus
	logger   observability.Logger
	config   CorpusServiceConfig

	// Service state
	started    bool
	scanning   bool
	scanCtx    context.Context
	scanCancel context.CancelFunc
	mu         sync.RWMutex

	// Scan progress
	currentScan *scanProgress

	// Metrics
	totalScans   uint64
	filesScanned uint64
	filesIndexed uint64
	scanErrors   uint64
	lastScanTime time.Time
	avgScanTime  time.Duration
}

// scanProgress tracks the progress of a scan
type scanProgress struct {
	startTime      time.Time
	totalFiles     int32
	processedFiles int32
	errors         int32
	currentPath    string
	mu             sync.RWMutex
}

// CorpusServiceConfig configures the corpus service
type CorpusServiceConfig struct {
	// Scan configuration
	BasePath       string
	FilePatterns   []string
	IgnorePatterns []string
	MaxWorkers     int
	ScanOnStart    bool
	RescanInterval time.Duration

	// Index configuration
	IndexPath          string
	EnableFullText     bool
	EnableVectorSearch bool

	// Resource limits
	MaxFileSize     int64
	MaxScanDuration time.Duration
	MemoryLimit     int64
}

// DefaultCorpusServiceConfig returns default configuration
func DefaultCorpusServiceConfig() CorpusServiceConfig {
	return CorpusServiceConfig{
		BasePath: ".",
		FilePatterns: []string{
			"*.md",
			"*.yaml",
			"*.yml",
			"*.go",
			"*.js",
			"*.ts",
			"*.py",
		},
		IgnorePatterns: []string{
			".git/**",
			"node_modules/**",
			"vendor/**",
			"*.test",
			"*.tmp",
		},
		MaxWorkers:         4,
		ScanOnStart:        false,
		RescanInterval:     30 * time.Minute,
		IndexPath:          ".guild/corpus.db",
		EnableFullText:     true,
		EnableVectorSearch: false,
		MaxFileSize:        10 * 1024 * 1024, // 10MB
		MaxScanDuration:    5 * time.Minute,
		MemoryLimit:        512 * 1024 * 1024, // 512MB
	}
}

// NewCorpusService creates a new corpus service wrapper
func NewCorpusService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config CorpusServiceConfig,
) (*CorpusService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("CorpusService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("CorpusService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("CorpusService")
	}

	// Create scanner with options
	scanner, err := corpus.NewDocumentScanner(
		config.BasePath,
		corpus.WithFilePatterns(config.FilePatterns),
		corpus.WithIgnorePatterns(config.IgnorePatterns),
		corpus.WithWorkers(config.MaxWorkers),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create document scanner").
			WithComponent("CorpusService")
	}

	// Create index
	index, err := NewDocumentIndex(config.IndexPath, IndexOptions{
		EnableFullText: config.EnableFullText,
		EnableVector:   config.EnableVectorSearch,
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create document index").
			WithComponent("CorpusService")
	}

	return &CorpusService{
		scanner:  scanner,
		index:    index,
		registry: registry,
		eventBus: eventBus,
		logger:   logger,
		config:   config,
	}, nil
}

// Name returns the service name
func (s *CorpusService) Name() string {
	return "corpus-service"
}

// Start initializes and starts the service
func (s *CorpusService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("CorpusService")
	}

	// Open index
	if err := s.index.Open(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open index").
			WithComponent("CorpusService")
	}

	s.started = true

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"corpus-service-started",
		"service.started",
		"corpus",
		map[string]interface{}{
			"base_path":     s.config.BasePath,
			"file_patterns": s.config.FilePatterns,
			"scan_on_start": s.config.ScanOnStart,
			"index_path":    s.config.IndexPath,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	// Start initial scan if configured
	if s.config.ScanOnStart {
		go func() {
			if err := s.StartScan(context.Background()); err != nil {
				s.logger.ErrorContext(ctx, "Initial scan failed", "error", err)
			}
		}()
	}

	s.logger.InfoContext(ctx, "Corpus service started",
		"base_path", s.config.BasePath,
		"scan_on_start", s.config.ScanOnStart)

	return nil
}

// Stop gracefully shuts down the service
func (s *CorpusService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("CorpusService")
	}

	// Cancel any ongoing scan
	if s.scanCancel != nil {
		s.scanCancel()
	}

	// Close index
	if err := s.index.Close(); err != nil {
		s.logger.ErrorContext(ctx, "Failed to close index", "error", err)
	}

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"corpus-service-stopped",
		"service.stopped",
		"corpus",
		map[string]interface{}{
			"total_scans":   s.totalScans,
			"files_scanned": s.filesScanned,
			"files_indexed": s.filesIndexed,
			"scan_errors":   s.scanErrors,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false

	s.logger.InfoContext(ctx, "Corpus service stopped",
		"total_scans", s.totalScans,
		"files_indexed", s.filesIndexed)

	return nil
}

// Health checks if the service is healthy
func (s *CorpusService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("CorpusService")
	}

	// Check index health
	if err := s.index.Health(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "index unhealthy").
			WithComponent("CorpusService")
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *CorpusService) Ready(ctx context.Context) error {
	return s.Health(ctx)
}

// StartScan starts a corpus scan
func (s *CorpusService) StartScan(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("CorpusService")
	}
	if s.scanning {
		s.mu.Unlock()
		return gerror.New(gerror.ErrCodeAlreadyExists, "scan already in progress", nil).
			WithComponent("CorpusService")
	}

	// Create scan context
	s.scanCtx, s.scanCancel = context.WithTimeout(ctx, s.config.MaxScanDuration)
	s.scanning = true
	s.currentScan = &scanProgress{
		startTime: time.Now(),
	}
	s.mu.Unlock()

	// Emit scan started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"corpus-scan-started",
		"corpus.scan.started",
		"corpus",
		map[string]interface{}{
			"base_path":  s.config.BasePath,
			"start_time": s.currentScan.startTime,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish scan started event", "error", err)
	}

	// Run scan in background
	go s.runScan()

	s.logger.InfoContext(ctx, "Corpus scan started", "base_path", s.config.BasePath)
	return nil
}

// runScan performs the actual scanning
func (s *CorpusService) runScan() {
	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.scanCancel = nil
		s.mu.Unlock()
	}()

	startTime := time.Now()
	var filesProcessed, errors uint64

	// Get result stream from scanner
	results, err := s.scanner.ScanStream(s.scanCtx)
	if err != nil {
		s.logger.ErrorContext(s.scanCtx, "Failed to start scan stream", "error", err)
		return
	}

	// Process results
	for result := range results {
		if result.Error != nil {
			atomic.AddUint64(&errors, 1)
			s.logger.ErrorContext(s.scanCtx, "Document scan error",
				"path", result.Document.Path,
				"error", result.Error)
			continue
		}

		// Update progress
		s.updateProgress(result.Document.Path)

		// Index document
		if err := s.index.Index(s.scanCtx, result.Document); err != nil {
			atomic.AddUint64(&errors, 1)
			s.logger.ErrorContext(s.scanCtx, "Failed to index document",
				"path", result.Document.Path,
				"error", err)

			// Emit indexing error event
			s.publishIndexEvent("corpus.index.error", result.Document, err)
		} else {
			atomic.AddUint64(&filesProcessed, 1)

			// Emit successful index event
			s.publishIndexEvent("corpus.file.indexed", result.Document, nil)
		}

		// Emit progress event periodically
		if atomic.LoadUint64(&filesProcessed)%10 == 0 {
			s.publishProgressEvent()
		}
	}

	// Update metrics
	duration := time.Since(startTime)
	s.mu.Lock()
	s.totalScans++
	s.filesScanned += filesProcessed
	s.filesIndexed += filesProcessed
	s.scanErrors += errors
	s.lastScanTime = time.Now()
	s.updateAvgScanTime(duration)
	s.mu.Unlock()

	// Emit scan completed event
	if err := s.eventBus.Publish(context.Background(), events.NewBaseEvent(
		"corpus-scan-completed",
		"corpus.scan.completed",
		"corpus",
		map[string]interface{}{
			"files_processed": filesProcessed,
			"errors":          errors,
			"duration":        duration.Seconds(),
		},
	)); err != nil {
		s.logger.Warn("Failed to publish scan completed event", "error", err)
	}

	s.logger.Info("Corpus scan completed",
		"files_processed", filesProcessed,
		"errors", errors,
		"duration", duration)
}

// updateProgress updates scan progress
func (s *CorpusService) updateProgress(path string) {
	if s.currentScan == nil {
		return
	}

	s.currentScan.mu.Lock()
	s.currentScan.processedFiles++
	s.currentScan.currentPath = path
	s.currentScan.mu.Unlock()
}

// publishProgressEvent publishes a scan progress event
func (s *CorpusService) publishProgressEvent() {
	if s.currentScan == nil {
		return
	}

	s.currentScan.mu.RLock()
	progress := map[string]interface{}{
		"processed_files": s.currentScan.processedFiles,
		"current_path":    s.currentScan.currentPath,
		"elapsed_time":    time.Since(s.currentScan.startTime).Seconds(),
	}
	s.currentScan.mu.RUnlock()

	if err := s.eventBus.Publish(context.Background(), events.NewBaseEvent(
		"corpus-scan-progress",
		"corpus.scan.progress",
		"corpus",
		progress,
	)); err != nil {
		s.logger.Warn("Failed to publish progress event", "error", err)
	}
}

// publishIndexEvent publishes an index event
func (s *CorpusService) publishIndexEvent(eventType string, doc *corpus.ScannedDocument, err error) {
	data := map[string]interface{}{
		"path": doc.Path,
		"type": doc.Type,
		"size": doc.Metadata.FileSize,
	}

	if err != nil {
		data["error"] = err.Error()
	}

	if err := s.eventBus.Publish(context.Background(), events.NewBaseEvent(
		"corpus-"+doc.ID,
		eventType,
		"corpus",
		data,
	)); err != nil {
		s.logger.Warn("Failed to publish index event", "error", err)
	}
}

// updateAvgScanTime updates the average scan time
func (s *CorpusService) updateAvgScanTime(duration time.Duration) {
	if s.avgScanTime == 0 {
		s.avgScanTime = duration
	} else {
		// Simple moving average
		s.avgScanTime = (s.avgScanTime*9 + duration) / 10
	}
}

// Search searches the corpus index
func (s *CorpusService) Search(ctx context.Context, query string, options SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("CorpusService")
	}
	s.mu.RUnlock()

	results, err := s.index.Search(ctx, query, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "search failed").
			WithComponent("CorpusService")
	}

	// Emit search event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"corpus-search",
		"corpus.search",
		"corpus",
		map[string]interface{}{
			"query":        query,
			"result_count": len(results),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish search event", "error", err)
	}

	return results, nil
}

// GetScanProgress returns current scan progress
func (s *CorpusService) GetScanProgress() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.scanning || s.currentScan == nil {
		return map[string]interface{}{
			"scanning": false,
		}
	}

	s.currentScan.mu.RLock()
	defer s.currentScan.mu.RUnlock()

	return map[string]interface{}{
		"scanning":        true,
		"processed_files": s.currentScan.processedFiles,
		"current_path":    s.currentScan.currentPath,
		"elapsed_time":    time.Since(s.currentScan.startTime).Seconds(),
	}
}

// GetMetrics returns service metrics
func (s *CorpusService) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_scans":    s.totalScans,
		"files_scanned":  s.filesScanned,
		"files_indexed":  s.filesIndexed,
		"scan_errors":    s.scanErrors,
		"last_scan_time": s.lastScanTime,
		"avg_scan_time":  s.avgScanTime.Seconds(),
		"is_scanning":    s.scanning,
	}
}
