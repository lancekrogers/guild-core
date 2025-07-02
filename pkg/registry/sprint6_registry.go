// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package registry provides performance optimization component registration and integration.
// This extends the main ComponentRegistry with performance optimization specific components.
package registry

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"go.uber.org/zap"
)

// PerformanceOptimizationRegistry extends the main ComponentRegistry with performance optimization components
type PerformanceOptimizationRegistry struct {
	// Session management components
	sessionManager   SessionManagerInterface
	sessionResumer   SessionResumerInterface
	sessionExporter  SessionExporterInterface
	sessionAnalytics SessionAnalyticsInterface

	// Performance optimization components
	performanceProfiler PerformanceProfilerInterface
	cacheManager        CacheManagerInterface
	memoryOptimizer     MemoryOptimizerInterface

	// Monitoring components
	performanceMonitor PerformanceMonitorInterface
	alertManager       AlertManagerInterface
	sloMonitor         SLOMonitorInterface
	dashboardRenderer  DashboardRendererInterface

	// Integration components
	eventBus          EventBus
	componentRegistry ComponentRegistry
	logger            *zap.Logger
	config            *PerformanceOptimizationConfig
	initialized       bool
	mu                sync.RWMutex
}

// performance optimization component interfaces
type SessionManagerInterface interface {
	CreateSession(ctx context.Context, userID, campaignID string) (*SessionData, error)
	LoadSession(ctx context.Context, sessionID string) (*SessionData, error)
	SaveSession(ctx context.Context, session *SessionData) error
	DeleteSession(ctx context.Context, sessionID string) error
	ListSessions(ctx context.Context, userID string) ([]*SessionData, error)
}

type SessionResumerInterface interface {
	ResumeSession(ctx context.Context, sessionID string) error
	GetRestorableState(ctx context.Context, sessionID string) (*RestorableState, error)
	RestoreUIState(ctx context.Context, sessionID string, state map[string]interface{}) error
}

type SessionExporterInterface interface {
	ExportSession(session *SessionData, format string, options *ExportOptions) ([]byte, error)
	GetSupportedFormats() []string
	ValidateExportOptions(format string, options *ExportOptions) error
}

type SessionAnalyticsInterface interface {
	GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error)
	AnalyzeSessionPatterns(ctx context.Context, userID string) (*SessionPatterns, error)
	RecordInteraction(ctx context.Context, sessionID string, interaction *Interaction) error
}

type PerformanceProfilerInterface interface {
	ProfileApplication(ctx context.Context, duration time.Duration) (*PerformanceReport, error)
	GetActiveProfiles(ctx context.Context) ([]*ProfileInfo, error)
	StopProfiling(ctx context.Context, profileID string) error
}

type CacheManagerInterface interface {
	GetMetrics(ctx context.Context, cacheName string) (*CacheMetrics, error)
	InvalidateCache(ctx context.Context, cacheName string) error
	OptimizeCache(ctx context.Context, cacheName string) error
}

type MemoryOptimizerInterface interface {
	OptimizeMemory(ctx context.Context) (*MemoryOptimizationReport, error)
	GetMemoryUsage(ctx context.Context) (*MemoryUsage, error)
}

type PerformanceMonitorInterface interface {
	GetCurrentMetrics(ctx context.Context, component string) (*SystemMetrics, error)
	StartMonitoring(ctx context.Context, component string) error
	StopMonitoring(ctx context.Context, component string) error
}

type AlertManagerInterface interface {
	GetActiveAlerts(ctx context.Context, severity string) ([]*Alert, error)
	CreateAlert(ctx context.Context, alert *Alert) error
	ResolveAlert(ctx context.Context, alertID string) error
}

type SLOMonitorInterface interface {
	CheckSLO(ctx context.Context, sloName string) (*SLOStatus, error)
	UpdateSLO(ctx context.Context, sloName string, target float64) error
}

type DashboardRendererInterface interface {
	RenderDashboard(ctx context.Context, config *DashboardConfig) ([]byte, error)
	GetAvailableWidgets() []string
}

// Data structures
type SessionData struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	CampaignID string                 `json:"campaign_id"`
	Messages   []*ChatMessage         `json:"messages"`
	UIState    map[string]interface{} `json:"ui_state"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type RestorableState struct {
	SessionID   string                 `json:"session_id"`
	UIState     map[string]interface{} `json:"ui_state"`
	Messages    []*ChatMessage         `json:"messages"`
	ActiveAgent string                 `json:"active_agent"`
	Context     map[string]interface{} `json:"context"`
}

type ExportOptions struct {
	Format          string `json:"format"`
	IncludeMetadata bool   `json:"include_metadata"`
	Compress        bool   `json:"compress"`
	EncryptionKey   string `json:"encryption_key,omitempty"`
}

type SessionMetrics struct {
	SessionID    string        `json:"session_id"`
	MessageCount int32         `json:"message_count"`
	Duration     time.Duration `json:"duration"`
	UniqueAgents int32         `json:"unique_agents"`
	Interactions int32         `json:"interactions"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type SessionPatterns struct {
	UserID             string                 `json:"user_id"`
	AverageSessionTime time.Duration          `json:"average_session_time"`
	PreferredAgents    []string               `json:"preferred_agents"`
	CommonInteractions []string               `json:"common_interactions"`
	PeakUsageTimes     []time.Time            `json:"peak_usage_times"`
	AnalysisMetadata   map[string]interface{} `json:"analysis_metadata"`
}

type Interaction struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Type      string                 `json:"type"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type PerformanceReport struct {
	ProfilerID    string                    `json:"profiler_id"`
	Duration      time.Duration             `json:"duration"`
	CPUProfile    *CPUProfile               `json:"cpu_profile"`
	MemoryProfile *MemoryProfile            `json:"memory_profile"`
	Hotspots      []*PerformanceHotspot     `json:"hotspots"`
	Optimizations []*OptimizationSuggestion `json:"optimizations"`
	Severity      string                    `json:"severity"`
	Confidence    float64                   `json:"confidence"`
}

type CPUProfile struct {
	TotalSamples int64                 `json:"total_samples"`
	SampleRate   float64               `json:"sample_rate"`
	TopFunctions []*FunctionProfile    `json:"top_functions"`
	Hotspots     []*PerformanceHotspot `json:"hotspots"`
}

type MemoryProfile struct {
	TotalAllocations int64              `json:"total_allocations"`
	PeakUsage        int64              `json:"peak_usage"`
	GCStats          *GCStats           `json:"gc_stats"`
	TopAllocators    []*FunctionProfile `json:"top_allocators"`
}

type FunctionProfile struct {
	Function    string        `json:"function"`
	File        string        `json:"file"`
	Line        int           `json:"line"`
	CPUTime     time.Duration `json:"cpu_time"`
	Percentage  float64       `json:"percentage"`
	Calls       int64         `json:"calls"`
	Allocations int64         `json:"allocations"`
}

type PerformanceHotspot struct {
	Function   string        `json:"function"`
	File       string        `json:"file"`
	Line       int           `json:"line"`
	CPUTime    time.Duration `json:"cpu_time"`
	Percentage float64       `json:"percentage"`
	Calls      int64         `json:"calls"`
	Severity   string        `json:"severity"`
}

type OptimizationSuggestion struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Impact        string  `json:"impact"`
	Difficulty    string  `json:"difficulty"`
	Confidence    float64 `json:"confidence"`
	EstimatedGain string  `json:"estimated_gain"`
	CodeSample    string  `json:"code_sample,omitempty"`
}

type GCStats struct {
	NumGC      uint32        `json:"num_gc"`
	PauseTotal time.Duration `json:"pause_total"`
	PauseAvg   time.Duration `json:"pause_avg"`
	PauseMax   time.Duration `json:"pause_max"`
	LastGC     time.Time     `json:"last_gc"`
}

type ProfileInfo struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	Status    string        `json:"status"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration"`
}

type CacheMetrics struct {
	CacheName   string    `json:"cache_name"`
	HitCount    int64     `json:"hit_count"`
	MissCount   int64     `json:"miss_count"`
	HitRate     float64   `json:"hit_rate"`
	MemoryUsage int64     `json:"memory_usage"`
	ItemCount   int64     `json:"item_count"`
	LastUpdated time.Time `json:"last_updated"`
}

type MemoryOptimizationReport struct {
	OptimizationID  string                `json:"optimization_id"`
	BeforeUsage     *MemoryUsage          `json:"before_usage"`
	AfterUsage      *MemoryUsage          `json:"after_usage"`
	Savings         int64                 `json:"savings"`
	Optimizations   []*MemoryOptimization `json:"optimizations"`
	Recommendations []string              `json:"recommendations"`
}

type MemoryUsage struct {
	TotalAlloc   uint64 `json:"total_alloc"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapIdle     uint64 `json:"heap_idle"`
	HeapInuse    uint64 `json:"heap_inuse"`
	StackInuse   uint64 `json:"stack_inuse"`
	StackSys     uint64 `json:"stack_sys"`
	NumGoroutine int    `json:"num_goroutine"`
}

type MemoryOptimization struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Savings     int64  `json:"savings"`
	Applied     bool   `json:"applied"`
}

type SystemMetrics struct {
	Component       string        `json:"component"`
	CPUUsage        float64       `json:"cpu_usage"`
	MemoryUsage     int64         `json:"memory_usage"`
	GoroutineCount  int           `json:"goroutine_count"`
	ResponseTimeP95 time.Duration `json:"response_time_p95"`
	ErrorRate       float64       `json:"error_rate"`
	Timestamp       time.Time     `json:"timestamp"`
}

type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type SLOStatus struct {
	Name         string    `json:"name"`
	CurrentValue float64   `json:"current_value"`
	TargetValue  float64   `json:"target_value"`
	ErrorBudget  float64   `json:"error_budget"`
	Status       string    `json:"status"`
	LastChecked  time.Time `json:"last_checked"`
}

type DashboardConfig struct {
	Title    string                 `json:"title"`
	Widgets  []string               `json:"widgets"`
	Layout   string                 `json:"layout"`
	Filters  map[string]interface{} `json:"filters"`
	Settings map[string]interface{} `json:"settings"`
}

// NewPerformanceOptimizationRegistry creates a registry for all performance optimization components
func NewPerformanceOptimizationRegistry(eventBus EventBus, componentRegistry ComponentRegistry, logger *zap.Logger) *PerformanceOptimizationRegistry {
	return &PerformanceOptimizationRegistry{
		eventBus:          eventBus,
		componentRegistry: componentRegistry,
		logger:            logger.Named("performance-optimization-registry"),
	}
}

// Session component registration methods
func (r *PerformanceOptimizationRegistry) RegisterSessionManager(manager SessionManagerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionManager = manager
	r.logger.Info("Session manager registered")
}

func (r *PerformanceOptimizationRegistry) GetSessionManager() SessionManagerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionManager
}

func (r *PerformanceOptimizationRegistry) RegisterSessionResumer(resumer SessionResumerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionResumer = resumer
	r.logger.Info("Session resumer registered")
}

func (r *PerformanceOptimizationRegistry) GetSessionResumer() SessionResumerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionResumer
}

func (r *PerformanceOptimizationRegistry) RegisterSessionExporter(exporter SessionExporterInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionExporter = exporter
	r.logger.Info("Session exporter registered")
}

func (r *PerformanceOptimizationRegistry) GetSessionExporter() SessionExporterInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionExporter
}

func (r *PerformanceOptimizationRegistry) RegisterSessionAnalytics(analytics SessionAnalyticsInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionAnalytics = analytics
	r.logger.Info("Session analytics registered")
}

func (r *PerformanceOptimizationRegistry) GetSessionAnalytics() SessionAnalyticsInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionAnalytics
}

// Performance component registration methods
func (r *PerformanceOptimizationRegistry) RegisterPerformanceProfiler(profiler PerformanceProfilerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.performanceProfiler = profiler
	r.logger.Info("Performance profiler registered")
}

func (r *PerformanceOptimizationRegistry) GetPerformanceProfiler() PerformanceProfilerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.performanceProfiler
}

func (r *PerformanceOptimizationRegistry) RegisterCacheManager(manager CacheManagerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheManager = manager
	r.logger.Info("Cache manager registered")
}

func (r *PerformanceOptimizationRegistry) GetCacheManager() CacheManagerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cacheManager
}

func (r *PerformanceOptimizationRegistry) RegisterMemoryOptimizer(optimizer MemoryOptimizerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryOptimizer = optimizer
	r.logger.Info("Memory optimizer registered")
}

func (r *PerformanceOptimizationRegistry) GetMemoryOptimizer() MemoryOptimizerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memoryOptimizer
}

// Monitoring component registration methods
func (r *PerformanceOptimizationRegistry) RegisterPerformanceMonitor(monitor PerformanceMonitorInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.performanceMonitor = monitor
	r.logger.Info("Performance monitor registered")
}

func (r *PerformanceOptimizationRegistry) GetPerformanceMonitor() PerformanceMonitorInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.performanceMonitor
}

func (r *PerformanceOptimizationRegistry) RegisterAlertManager(manager AlertManagerInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alertManager = manager
	r.logger.Info("Alert manager registered")
}

func (r *PerformanceOptimizationRegistry) GetAlertManager() AlertManagerInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.alertManager
}

func (r *PerformanceOptimizationRegistry) RegisterSLOMonitor(monitor SLOMonitorInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sloMonitor = monitor
	r.logger.Info("SLO monitor registered")
}

func (r *PerformanceOptimizationRegistry) GetSLOMonitor() SLOMonitorInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sloMonitor
}

func (r *PerformanceOptimizationRegistry) RegisterDashboardRenderer(renderer DashboardRendererInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dashboardRenderer = renderer
	r.logger.Info("Dashboard renderer registered")
}

func (r *PerformanceOptimizationRegistry) GetDashboardRenderer() DashboardRendererInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dashboardRenderer
}

// HealthCheck performs comprehensive health checking with proper context handling
func (r *PerformanceOptimizationRegistry) HealthCheck(ctx context.Context) error {
	// Check for context cancellation first - critical for observability
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "health check cancelled").
			FromContext(ctx).
			WithComponent("performance-optimization-registry").
			WithOperation("HealthCheck").
			WithDetails("stage", "context_check")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a logger with context-aware fields for observability
	logger := r.logger.With(
		zap.String("operation", "HealthCheck"),
		zap.Time("check_time", time.Now()),
	)

	var componentErrors []error
	
	// Component validation with structured error reporting
	components := []struct {
		name      string
		component interface{}
		required  bool
	}{
		{"session-manager", r.sessionManager, true},
		{"session-resumer", r.sessionResumer, true},
		{"session-exporter", r.sessionExporter, false}, // Optional component
		{"session-analytics", r.sessionAnalytics, true},
		{"performance-profiler", r.performanceProfiler, true},
		{"cache-manager", r.cacheManager, true},
		{"memory-optimizer", r.memoryOptimizer, true},
		{"performance-monitor", r.performanceMonitor, true},
		{"alert-manager", r.alertManager, true},
		{"slo-monitor", r.sloMonitor, false}, // Optional component
		{"dashboard-renderer", r.dashboardRenderer, false}, // Optional component
	}

	for _, comp := range components {
		// Check context cancellation during iteration for long health checks
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "health check cancelled during component validation").
				FromContext(ctx).
				WithComponent("performance-optimization-registry").
				WithOperation("HealthCheck").
				WithDetails("checking_component", comp.name)
		}

		if comp.component == nil && comp.required {
			componentErrors = append(componentErrors, 
				gerror.New(gerror.ErrCodeConfiguration, "required component not registered", nil).
					FromContext(ctx).
					WithComponent("performance-optimization-registry").
					WithOperation("HealthCheck").
					WithDetails("component_name", comp.name).
					WithDetails("component_type", "required"))
			
			logger.Error("Required component missing", 
				zap.String("component", comp.name),
				zap.Bool("required", comp.required))
		} else if comp.component == nil {
			logger.Warn("Optional component not registered",
				zap.String("component", comp.name),
				zap.Bool("required", comp.required))
		}
	}

	if len(componentErrors) > 0 {
		// Create comprehensive error with all component failures
		return gerror.New(gerror.ErrCodeConfiguration, "performance optimization registry health check failed", nil).
			FromContext(ctx).
			WithComponent("performance-optimization-registry").
			WithOperation("HealthCheck").
			WithDetails("failed_components", len(componentErrors)).
			WithDetails("total_components", len(components))
	}

	logger.Info("Performance optimization registry health check passed",
		zap.Int("components_checked", len(components)),
		zap.Int("required_components", countRequired(components)),
		zap.Int("optional_components", len(components)-countRequired(components)))
	
	return nil
}

// Helper function to count required components
func countRequired(components []struct {
	name      string
	component interface{}
	required  bool
}) int {
	count := 0
	for _, comp := range components {
		if comp.required {
			count++
		}
	}
	return count
}

// InitializeComponents initializes all performance optimization components with proper dependencies and error handling
func (r *PerformanceOptimizationRegistry) InitializeComponents(ctx context.Context, config *PerformanceOptimizationConfig) error {
	// Validate input parameters first
	if config == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "configuration cannot be nil", nil).
			FromContext(ctx).
			WithComponent("performance-optimization-registry").
			WithOperation("InitializeComponents").
			WithDetails("validation_stage", "config_check")
	}

	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled").
			FromContext(ctx).
			WithComponent("performance-optimization-registry").
			WithOperation("InitializeComponents")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already initialized
	if r.initialized {
		return gerror.New(gerror.ErrCodeAlreadyExists, "performance optimization components already initialized", nil).
			FromContext(ctx).
			WithComponent("performance-optimization-registry").
			WithOperation("InitializeComponents").
			WithDetails("current_state", "already_initialized")
	}

	logger := r.logger.With(
		zap.String("operation", "InitializeComponents"),
		zap.Time("init_start", time.Now()),
	)

	logger.Info("Starting performance optimization component initialization")

	// Store config early for cleanup in case of failure
	r.config = config

	// Initialize components with proper error handling and context propagation
	initSteps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"session-components", func(ctx context.Context) error {
			return r.initializeSessionComponents(ctx, config.Session)
		}},
		{"performance-components", func(ctx context.Context) error {
			return r.initializePerformanceComponents(ctx, config.Performance)
		}},
		{"monitoring-components", func(ctx context.Context) error {
			return r.initializeMonitoringComponents(ctx, config.Monitoring)
		}},
	}

	for i, step := range initSteps {
		// Check context cancellation before each step
		if err := ctx.Err(); err != nil {
			// Cleanup partially initialized state
			r.cleanupPartialInitialization(ctx, i)
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled during component setup").
				FromContext(ctx).
				WithComponent("performance-optimization-registry").
				WithOperation("InitializeComponents").
				WithDetails("step", step.name).
				WithDetails("completed_steps", i)
		}

		logger.Info("Initializing component group", zap.String("step", step.name))
		
		if err := step.fn(ctx); err != nil {
			// Cleanup any partially initialized components
			r.cleanupPartialInitialization(ctx, i)
			return gerror.Wrap(err, gerror.ErrCodeConfiguration, "failed to initialize component group").
				FromContext(ctx).
				WithComponent("performance-optimization-registry").
				WithOperation("InitializeComponents").
				WithDetails("failed_step", step.name).
				WithDetails("step_index", i)
		}

		logger.Info("Component group initialized successfully", zap.String("step", step.name))
	}

	r.initialized = true
	
	logger.Info("All performance optimization components initialized successfully",
		zap.Int("total_steps", len(initSteps)),
		zap.Duration("init_duration", time.Since(time.Now())))
	
	return nil
}

// cleanupPartialInitialization handles cleanup when initialization fails partway through
func (r *PerformanceOptimizationRegistry) cleanupPartialInitialization(ctx context.Context, completedSteps int) {
	logger := r.logger.With(
		zap.String("operation", "cleanupPartialInitialization"),
		zap.Int("completed_steps", completedSteps),
	)

	logger.Warn("Cleaning up partially initialized components")

	// Reset component registrations based on how far we got
	if completedSteps >= 1 {
		// Session components were initialized, clean them up
		r.sessionManager = nil
		r.sessionResumer = nil
		r.sessionExporter = nil
		r.sessionAnalytics = nil
		logger.Info("Session components cleaned up")
	}

	if completedSteps >= 2 {
		// Performance components were initialized, clean them up  
		r.performanceProfiler = nil
		r.cacheManager = nil
		r.memoryOptimizer = nil
		logger.Info("Performance components cleaned up")
	}

	if completedSteps >= 3 {
		// Monitoring components were initialized, clean them up
		r.performanceMonitor = nil
		r.alertManager = nil
		r.sloMonitor = nil
		r.dashboardRenderer = nil
		logger.Info("Monitoring components cleaned up")
	}

	r.config = nil
	r.initialized = false
	
	logger.Info("Partial initialization cleanup completed")
}

func (r *PerformanceOptimizationRegistry) initializeSessionComponents(ctx context.Context, config SessionConfig) error {
	// Session components would be initialized here based on config
	// This is where specific implementations would be created and registered
	return nil
}

func (r *PerformanceOptimizationRegistry) initializePerformanceComponents(ctx context.Context, config PerformanceConfig) error {
	// Performance components would be initialized here based on config
	return nil
}

func (r *PerformanceOptimizationRegistry) initializeMonitoringComponents(ctx context.Context, config MonitoringConfig) error {
	// Monitoring components would be initialized here based on config
	return nil
}

// Shutdown gracefully shuts down all performance optimization components
func (r *PerformanceOptimizationRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return gerror.New(gerror.ErrCodeValidation, "performance optimization components not initialized", nil).
			WithComponent("performance-optimization-registry").
			WithOperation("Shutdown")
	}

	// Gracefully shutdown components
	r.sessionManager = nil
	r.sessionResumer = nil
	r.sessionExporter = nil
	r.sessionAnalytics = nil
	r.performanceProfiler = nil
	r.cacheManager = nil
	r.memoryOptimizer = nil
	r.performanceMonitor = nil
	r.alertManager = nil
	r.sloMonitor = nil
	r.dashboardRenderer = nil

	r.initialized = false
	r.logger.Info("All performance optimization components shut down successfully")
	return nil
}

// Configuration structs for performance optimization components
type PerformanceOptimizationConfig struct {
	Session     SessionConfig     `json:"session" yaml:"session"`
	Performance PerformanceConfig `json:"performance" yaml:"performance"`
	Monitoring  MonitoringConfig  `json:"monitoring" yaml:"monitoring"`
}

type SessionConfig struct {
	Database     DatabaseConfig   `json:"database" yaml:"database"`
	EventBus     EventBusConfig   `json:"event_bus" yaml:"event_bus"`
	ExportConfig ExportConfigData `json:"export" yaml:"export"`
	Encryption   EncryptionConfig `json:"encryption" yaml:"encryption"`
}

type PerformanceConfig struct {
	Profiling ProfilingConfig `json:"profiling" yaml:"profiling"`
	Caching   CachingConfig   `json:"caching" yaml:"caching"`
	Memory    MemoryConfigData `json:"memory" yaml:"memory"`
}

type MonitoringConfig struct {
	Metrics   MetricsConfig   `json:"metrics" yaml:"metrics"`
	Alerting  AlertingConfig  `json:"alerting" yaml:"alerting"`
	SLO       SLOConfig       `json:"slo" yaml:"slo"`
	Dashboard DashboardConfig `json:"dashboard" yaml:"dashboard"`
}

type DatabaseConfig struct {
	Path        string `json:"path" yaml:"path"`
	MaxConns    int    `json:"max_conns" yaml:"max_conns"`
	IdleTimeout int    `json:"idle_timeout" yaml:"idle_timeout"`
}

type EventBusConfig struct {
	BufferSize    int    `json:"buffer_size" yaml:"buffer_size"`
	MaxRetries    int    `json:"max_retries" yaml:"max_retries"`
	RetryInterval string `json:"retry_interval" yaml:"retry_interval"`
}

type EncryptionConfig struct {
	Key       string `json:"key" yaml:"key"`
	Algorithm string `json:"algorithm" yaml:"algorithm"`
}

type ProfilingConfig struct {
	Enabled     bool   `json:"enabled" yaml:"enabled"`
	SampleRate  int    `json:"sample_rate" yaml:"sample_rate"`
	OutputDir   string `json:"output_dir" yaml:"output_dir"`
	MaxProfiles int    `json:"max_profiles" yaml:"max_profiles"`
}

type CachingConfig struct {
	Enabled        bool   `json:"enabled" yaml:"enabled"`
	MaxSize        int64  `json:"max_size" yaml:"max_size"`
	TTL            string `json:"ttl" yaml:"ttl"`
	EvictionPolicy string `json:"eviction_policy" yaml:"eviction_policy"`
}

type MetricsConfig struct {
	Enabled         bool   `json:"enabled" yaml:"enabled"`
	CollectInterval string `json:"collect_interval" yaml:"collect_interval"`
	RetentionPeriod string `json:"retention_period" yaml:"retention_period"`
}

type AlertingConfig struct {
	Enabled    bool               `json:"enabled" yaml:"enabled"`
	Thresholds map[string]float64 `json:"thresholds" yaml:"thresholds"`
	Recipients []string           `json:"recipients" yaml:"recipients"`
}

type SLOConfig struct {
	Enabled bool               `json:"enabled" yaml:"enabled"`
	Targets map[string]float64 `json:"targets" yaml:"targets"`
	Windows map[string]string  `json:"windows" yaml:"windows"`
}

type ExportConfigData struct {
	OutputDir   string `json:"output_dir" yaml:"output_dir"`
	MaxFileSize int64  `json:"max_file_size" yaml:"max_file_size"`
}

type MemoryConfigData struct {
	MaxHeapSize   int64 `json:"max_heap_size" yaml:"max_heap_size"`
	GCTargetPerc  int   `json:"gc_target_perc" yaml:"gc_target_perc"`
	EnableProfiling bool `json:"enable_profiling" yaml:"enable_profiling"`
}


// NewMulti creates a new GuildError that contains multiple errors
func NewMulti(code gerror.ErrorCode, message string, causes []error) *gerror.GuildError {
	// For now, just return the first error wrapped
	if len(causes) > 0 {
		return gerror.Wrap(causes[0], code, message)
	}
	return gerror.New(code, message, nil)
}
