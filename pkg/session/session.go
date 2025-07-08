// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package session provides comprehensive session management for Guild Framework
//
// This package implements the session management requirements for Product Vision performance optimization,
// Agent 2 task, providing:
//   - Session persistence with encryption, compression, and auto-save
//   - Conversation resume functionality with UI state restoration
//   - Export/import capabilities supporting JSON, Markdown, HTML, and PDF formats
//   - Session analytics with usage tracking and productivity insights
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Registry pattern for dependency injection
//   - Observability integration
//
// Example usage:
//
//	// Create session manager
//	store := NewSQLiteSessionStore(db)
//	manager := NewSessionManager(store, WithEncryption(key), WithAutoSaveInterval(30*time.Second))
//
//	// Create and save session
//	session := &Session{
//		ID:         "session-123",
//		UserID:     "user-456",
//		CampaignID: "campaign-789",
//		StartTime:  time.Now(),
//		State:      SessionState{Status: SessionStatusActive},
//	}
//	err := manager.SaveSession(ctx, session)
//
//	// Resume session
//	resumer := NewSessionResumer(manager, ui, orchestrator, corpus)
//	err = resumer.ResumeSession(ctx, "session-123")
//
//	// Export session
//	exporter := NewSessionExporter()
//	data, err := exporter.Export(session, ExportOptions{
//		Format: ExportFormatMarkdown,
//		IncludeMetadata: true,
//	})
//
//	// Analyze session
//	analytics := NewSessionAnalytics(analyticsStore)
//	report, err := analytics.AnalyzeSession(ctx, session)
package session

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "session"
)

// DefaultSessionRegistry provides a default implementation of the session registry
type DefaultSessionRegistry struct {
	sessionManager     SessionManagerInterface
	sessionResumer     SessionResumerInterface
	sessionExporter    SessionExporterInterface
	sessionAnalytics   SessionAnalyticsInterface
	orchestratorInteg  OrchestratorIntegration
	uiIntegration      UIIntegration
	storageIntegration StorageIntegration
	configProvider     ConfigurationProvider
	mu                 sync.RWMutex
}

// NewDefaultSessionRegistry creates a new session registry
func NewDefaultSessionRegistry() *DefaultSessionRegistry {
	return &DefaultSessionRegistry{}
}

// Registry implementation
func (r *DefaultSessionRegistry) RegisterSessionManager(manager SessionManagerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionManager = manager
}

func (r *DefaultSessionRegistry) RegisterSessionResumer(resumer SessionResumerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionResumer = resumer
}

func (r *DefaultSessionRegistry) RegisterSessionExporter(exporter SessionExporterInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionExporter = exporter
}

func (r *DefaultSessionRegistry) RegisterSessionAnalytics(analytics SessionAnalyticsInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionAnalytics = analytics
}

func (r *DefaultSessionRegistry) RegisterOrchestratorIntegration(integration OrchestratorIntegration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orchestratorInteg = integration
}

func (r *DefaultSessionRegistry) RegisterUIIntegration(integration UIIntegration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uiIntegration = integration
}

func (r *DefaultSessionRegistry) RegisterStorageIntegration(integration StorageIntegration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storageIntegration = integration
}

func (r *DefaultSessionRegistry) RegisterConfigurationProvider(provider ConfigurationProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configProvider = provider
}

func (r *DefaultSessionRegistry) GetSessionManager() SessionManagerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionManager
}

func (r *DefaultSessionRegistry) GetSessionResumer() SessionResumerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionResumer
}

func (r *DefaultSessionRegistry) GetSessionExporter() SessionExporterInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionExporter
}

func (r *DefaultSessionRegistry) GetSessionAnalytics() SessionAnalyticsInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionAnalytics
}

// SessionService provides a high-level facade for session operations
type SessionService struct {
	registry    SessionRegistry
	middlewares []SessionMiddleware
	logger      observability.Logger
	mu          sync.RWMutex
}

// NewSessionService creates a new session service
func NewSessionService(registry SessionRegistry) *SessionService {
	return &SessionService{
		registry:    registry,
		middlewares: make([]SessionMiddleware, 0),
		logger:      observability.GetLogger(context.Background()).WithComponent("session"),
	}
}

// AddMiddleware adds a session middleware
func (s *SessionService) AddMiddleware(middleware SessionMiddleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middlewares = append(s.middlewares, middleware)
}

// CreateSession creates a new session with full middleware support
func (s *SessionService) CreateSession(ctx context.Context, userID, campaignID string) (*Session, error) {
	logger := s.logger.WithOperation("CreateSession").
		With("user_id", userID, "campaign_id", campaignID)

	startTime := time.Now()
	defer func() {
		logger.With("duration_ms", time.Since(startTime).Milliseconds()).
			Info("CreateSession completed")
	}()

	// Create session object
	session := &Session{
		ID:             generateSessionID(),
		UserID:         userID,
		CampaignID:     campaignID,
		StartTime:      time.Now(),
		LastActiveTime: time.Now(),
		State: SessionState{
			ActiveAgents: make(map[string]AgentState),
			Variables:    make(map[string]interface{}),
			Status:       SessionStatusActive,
		},
		Context: SessionContext{
			WorkingDirectory: "/tmp", // Default, should be configurable
		},
		Metadata: make(map[string]interface{}),
	}

	// Run before middlewares
	for _, middleware := range s.middlewares {
		if err := middleware.BeforeSessionCreate(ctx, session); err != nil {
			logger.WithError(err).Error("Middleware BeforeSessionCreate failed")
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "session creation middleware failed")
		}
	}

	// Create session using manager
	manager := s.registry.GetSessionManager()
	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "session manager not registered", nil)
	}

	createdSession, err := manager.CreateSession(ctx, userID, campaignID)
	if err != nil {
		logger.WithError(err).Error("Failed to create session")
		return nil, err
	}

	// Run after middlewares
	for _, middleware := range s.middlewares {
		if err := middleware.AfterSessionCreate(ctx, createdSession); err != nil {
			logger.WithError(err).Warn("Middleware AfterSessionCreate failed")
			// Don't fail the operation for after middlewares
		}
	}

	logger.With("session_id", createdSession.ID).Info("Session created successfully")
	return createdSession, nil
}

// ResumeSession resumes a session with full orchestration
func (s *SessionService) ResumeSession(ctx context.Context, sessionID string) error {
	logger := s.logger.WithOperation("ResumeSession").
		With("session_id", sessionID)

	startTime := time.Now()
	defer func() {
		logger.With("duration_ms", time.Since(startTime).Milliseconds()).
			Info("ResumeSession completed")
	}()

	resumer := s.registry.GetSessionResumer()
	if resumer == nil {
		return gerror.New(gerror.ErrCodeConfiguration, "session resumer not registered", nil)
	}

	err := resumer.ResumeSession(ctx, sessionID)
	if err != nil {
		logger.WithError(err).Error("Failed to resume session")
		return err
	}

	logger.Info("Session resumed successfully")
	return nil
}

// ExportSession exports a session with the specified options
func (s *SessionService) ExportSession(ctx context.Context, sessionID string, options ExportOptions) ([]byte, error) {
	logger := s.logger.WithOperation("ExportSession").
		With("session_id", sessionID, "format", options.Format.String())

	startTime := time.Now()
	defer func() {
		logger.With("duration_ms", time.Since(startTime).Milliseconds()).
			Info("ExportSession completed")
	}()

	// Load session
	manager := s.registry.GetSessionManager()
	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "session manager not registered", nil)
	}

	session, err := manager.LoadSession(ctx, sessionID)
	if err != nil {
		logger.WithError(err).Error("Failed to load session for export")
		return nil, err
	}

	// Export session
	exporter := s.registry.GetSessionExporter()
	if exporter == nil {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "session exporter not registered", nil)
	}

	data, err := exporter.ExportSession(session, options.Format, options)
	if err != nil {
		logger.WithError(err).Error("Failed to export session")
		return nil, err
	}

	logger.With("export_size", len(data)).Info("Session exported successfully")
	return data, nil
}

// AnalyzeSession performs analytics on a session
func (s *SessionService) AnalyzeSession(ctx context.Context, sessionID string) (*AnalyticsData, error) {
	logger := s.logger.WithOperation("AnalyzeSession").
		With("session_id", sessionID)

	startTime := time.Now()
	defer func() {
		logger.With("duration_ms", time.Since(startTime).Milliseconds()).
			Info("AnalyzeSession completed")
	}()

	// Load session
	manager := s.registry.GetSessionManager()
	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "session manager not registered", nil)
	}

	session, err := manager.LoadSession(ctx, sessionID)
	if err != nil {
		logger.WithError(err).Error("Failed to load session for analysis")
		return nil, err
	}

	// Analyze session
	analytics := s.registry.GetSessionAnalytics()
	if analytics == nil {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "session analytics not registered", nil)
	}

	data, err := analytics.AnalyzeSession(ctx, session)
	if err != nil {
		logger.WithError(err).Error("Failed to analyze session")
		return nil, err
	}

	logger.With("productivity_score", data.ProductivityScore).
		Info("Session analyzed successfully")
	return data, nil
}

// HealthCheck performs a health check on all session components
func (s *SessionService) HealthCheck(ctx context.Context) error {
	logger := s.logger.WithOperation("HealthCheck")

	var errors []error

	// Check session manager
	if s.registry.GetSessionManager() == nil {
		errors = append(errors, fmt.Errorf("session manager not registered"))
	}

	// Check session resumer
	if s.registry.GetSessionResumer() == nil {
		errors = append(errors, fmt.Errorf("session resumer not registered"))
	}

	// Check session exporter
	if s.registry.GetSessionExporter() == nil {
		errors = append(errors, fmt.Errorf("session exporter not registered"))
	}

	// Check session analytics
	if s.registry.GetSessionAnalytics() == nil {
		errors = append(errors, fmt.Errorf("session analytics not registered"))
	}

	if len(errors) > 0 {
		logger.With("error_count", len(errors)).Error("Health check failed")
		return gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("health check failed with %d errors", len(errors)), nil)
	}

	logger.Info("Health check passed")
	return nil
}

// GetMetrics returns service metrics
func (s *SessionService) GetMetrics(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"version":          Version,
		"api_version":      APIVersion,
		"middleware_count": len(s.middlewares),
		"registry_status":  s.getRegistryStatus(),
	}
}

func (s *SessionService) getRegistryStatus() map[string]bool {
	return map[string]bool{
		"session_manager":   s.registry.GetSessionManager() != nil,
		"session_resumer":   s.registry.GetSessionResumer() != nil,
		"session_exporter":  s.registry.GetSessionExporter() != nil,
		"session_analytics": s.registry.GetSessionAnalytics() != nil,
	}
}

// Utility functions

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// InitializeSessionManagement sets up session management with default configurations
func InitializeSessionManagement(ctx context.Context, options InitOptions) (*SessionService, error) {
	logger := observability.GetLogger(ctx).WithComponent("session").WithOperation("Initialize")

	logger.Info("Initializing session management")

	// Create registry
	registry := NewDefaultSessionRegistry()

	// Initialize storage if provided
	if options.Database != nil {
		db, ok := options.Database.(*sql.DB)
		if !ok {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "database must be of type *sql.DB", nil)
		}
		store := NewSQLiteSessionStore(db)
		if err := store.InitSchema(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize session schema")
		}

		// Create session manager
		managerOptions := []SessionOption{}
		if options.EncryptionKey != nil {
			managerOptions = append(managerOptions, WithEncryption(options.EncryptionKey))
		}
		if options.AutoSaveInterval > 0 {
			managerOptions = append(managerOptions, WithAutoSaveInterval(options.AutoSaveInterval))
		}

		manager := NewSessionManager(store, managerOptions...)
		registry.RegisterSessionManager(manager)

		// Create session resumer (requires UI and orchestrator integrations)
		if options.UIIntegration != nil && options.OrchestratorIntegration != nil {
			resumer := NewSessionResumer(manager, options.UIIntegration, options.OrchestratorIntegration, options.CorpusIntegration)
			registry.RegisterSessionResumer(resumer)
		}

		// Create session exporter
		exporter := NewSessionExporter()
		registry.RegisterSessionExporter(exporter)

		// Create session analytics (requires analytics store)
		if options.AnalyticsStore != nil {
			analytics := NewSessionAnalytics(options.AnalyticsStore)
			registry.RegisterSessionAnalytics(analytics)
		}
	}

	// Create session service
	service := NewSessionService(registry)

	// Add default middlewares if specified
	if options.EnableLogging {
		service.AddMiddleware(&LoggingMiddleware{logger: logger})
	}

	if options.EnableMetrics {
		service.AddMiddleware(&MetricsMiddleware{})
	}

	logger.Info("Session management initialized successfully")
	return service, nil
}

// InitOptions configures session management initialization
type InitOptions struct {
	Database                interface{}           // Database connection
	EncryptionKey           []byte                // Encryption key for session data
	AutoSaveInterval        time.Duration         // Auto-save interval
	UIIntegration           UIRestorer            // UI integration for resume functionality
	OrchestratorIntegration OrchestratorInterface // Orchestrator integration
	CorpusIntegration       CorpusInterface       // Corpus integration
	AnalyticsStore          AnalyticsStore        // Analytics storage
	EnableLogging           bool                  // Enable logging middleware
	EnableMetrics           bool                  // Enable metrics middleware
}

// Default middleware implementations

// LoggingMiddleware provides logging for session operations
type LoggingMiddleware struct {
	logger observability.Logger
}

func (m *LoggingMiddleware) BeforeSessionCreate(ctx context.Context, session *Session) error {
	m.logger.With("session_id", session.ID).Debug("Creating session")
	return nil
}

func (m *LoggingMiddleware) AfterSessionCreate(ctx context.Context, session *Session) error {
	m.logger.With("session_id", session.ID).Info("Session created")
	return nil
}

func (m *LoggingMiddleware) BeforeSessionLoad(ctx context.Context, sessionID string) error {
	m.logger.With("session_id", sessionID).Debug("Loading session")
	return nil
}

func (m *LoggingMiddleware) AfterSessionLoad(ctx context.Context, session *Session) error {
	m.logger.With("session_id", session.ID).Debug("Session loaded")
	return nil
}

func (m *LoggingMiddleware) BeforeSessionSave(ctx context.Context, session *Session) error {
	m.logger.With("session_id", session.ID).Debug("Saving session")
	return nil
}

func (m *LoggingMiddleware) AfterSessionSave(ctx context.Context, session *Session) error {
	m.logger.With("session_id", session.ID).Debug("Session saved")
	return nil
}

// MetricsMiddleware provides metrics collection for session operations
type MetricsMiddleware struct {
	createCount int64
	loadCount   int64
	saveCount   int64
	mu          sync.RWMutex
}

func (m *MetricsMiddleware) BeforeSessionCreate(ctx context.Context, session *Session) error {
	return nil
}

func (m *MetricsMiddleware) AfterSessionCreate(ctx context.Context, session *Session) error {
	m.mu.Lock()
	m.createCount++
	m.mu.Unlock()
	return nil
}

func (m *MetricsMiddleware) BeforeSessionLoad(ctx context.Context, sessionID string) error {
	return nil
}

func (m *MetricsMiddleware) AfterSessionLoad(ctx context.Context, session *Session) error {
	m.mu.Lock()
	m.loadCount++
	m.mu.Unlock()
	return nil
}

func (m *MetricsMiddleware) BeforeSessionSave(ctx context.Context, session *Session) error {
	return nil
}

func (m *MetricsMiddleware) AfterSessionSave(ctx context.Context, session *Session) error {
	m.mu.Lock()
	m.saveCount++
	m.mu.Unlock()
	return nil
}

func (m *MetricsMiddleware) GetMetrics() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]int64{
		"session_creates": m.createCount,
		"session_loads":   m.loadCount,
		"session_saves":   m.saveCount,
	}
}
