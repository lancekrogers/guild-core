// Package bootstrap provides application startup and shutdown orchestration
package bootstrap

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/guild-framework/guild-core/internal/integration/bridges"
	"github.com/guild-framework/guild-core/internal/integration/services"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/registry"
	"github.com/guild-framework/guild-core/pkg/session"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// Application represents the main application with all integrated components
type Application struct {
	// Core components
	Config            *Config
	Logger            observability.Logger
	EventBus          events.EventBus
	Storage           StorageInterface
	ComponentRegistry registry.ComponentRegistry
	ServiceRegistry   *services.DefaultServiceRegistry

	// Bridges
	EventLoggerBridge          *bridges.EventLoggerBridge
	PersistenceEventBridge     *bridges.PersistenceEventBridge
	UIEventBridge              *bridges.UIEventBridge
	OrchestratorCampaignBridge *bridges.OrchestratorCampaignBridge
	AgentRegistrationBridge    *bridges.AgentRegistrationBridge
	CommissionProcessorBridge  *bridges.CommissionProcessorBridge

	// Monitoring
	Monitor MonitorInterface

	// State
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	mu      sync.RWMutex
}

// Config represents application configuration
type Config struct {
	Version    string
	Storage    StorageConfig
	Bridges    BridgesConfig
	Monitoring MonitoringConfig
}

// StorageConfig configures storage
type StorageConfig struct {
	DatabasePath   string
	MigrationsPath string
}

// BridgesConfig configures integration bridges
type BridgesConfig struct {
	EventLogging      EventLoggingConfig
	PersistenceEvents PersistenceEventsConfig
	UIEvents          UIEventsConfig
}

// EventLoggingConfig configures event logging
type EventLoggingConfig struct {
	Enabled     bool
	LogLevel    string
	IncludeData bool
}

// PersistenceEventsConfig configures persistence events
type PersistenceEventsConfig struct {
	Enabled         bool
	EmitCRUD        bool
	EmitQuery       bool
	EmitTransaction bool
	IncludePayload  bool
}

// UIEventsConfig configures UI events
type UIEventsConfig struct {
	Enabled          bool
	BatchEvents      bool
	BatchIntervalMs  int
	MaxBatchSize     int
	UIEventTypes     []string
	SystemEventTypes []string
}

// MonitoringConfig configures monitoring
type MonitoringConfig struct {
	MetricsEnabled   bool
	TracingEnabled   bool
	ProfilingEnabled bool
}

// StorageInterface represents the storage layer
type StorageInterface interface {
	Close() error
}

// MonitorInterface represents the monitoring layer
type MonitorInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Options configures the application
type Options struct {
	// ConfigPath is the path to the configuration file
	ConfigPath string

	// LogLevel sets the logging level
	LogLevel string

	// EnableEventLogging enables event logging bridge
	EnableEventLogging bool

	// EnablePersistenceEvents enables persistence event bridge
	EnablePersistenceEvents bool

	// EnableUIEvents enables UI event bridge
	EnableUIEvents bool

	// StartTimeout is the maximum time to wait for startup
	StartTimeout time.Duration

	// ShutdownTimeout is the maximum time to wait for shutdown
	ShutdownTimeout time.Duration
}

// DefaultOptions returns default application options
func DefaultOptions() Options {
	return Options{
		ConfigPath:              "guild.yaml",
		LogLevel:                "info",
		EnableEventLogging:      true,
		EnablePersistenceEvents: true,
		EnableUIEvents:          true,
		StartTimeout:            30 * time.Second,
		ShutdownTimeout:         30 * time.Second,
	}
}

// NewApplication creates a new application instance
func NewApplication(opts Options) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Load configuration
	// For now, create a minimal config
	// In a real implementation, this would load from file
	cfg := &Config{
		Version: "1.0.0",
		Storage: StorageConfig{
			DatabasePath:   ".guild/memory.db",
			MigrationsPath: "migrations",
		},
		Bridges: BridgesConfig{
			EventLogging: EventLoggingConfig{
				Enabled:     true,
				LogLevel:    "info",
				IncludeData: true,
			},
			PersistenceEvents: PersistenceEventsConfig{
				Enabled:         true,
				EmitCRUD:        true,
				EmitQuery:       false,
				EmitTransaction: false,
				IncludePayload:  false,
			},
			UIEvents: UIEventsConfig{
				Enabled:         true,
				BatchEvents:     true,
				BatchIntervalMs: 100,
				MaxBatchSize:    100,
			},
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled:   true,
			TracingEnabled:   false,
			ProfilingEnabled: false,
		},
	}

	// Initialize logger
	logger := observability.GetLogger(ctx).WithComponent("bootstrap")

	// Create application
	app := &Application{
		Config:          cfg,
		Logger:          logger,
		ServiceRegistry: services.NewServiceRegistry(ctx),
		ctx:             ctx,
		cancel:          cancel,
	}

	return app, nil
}

// Initialize initializes all application components
func (app *Application) Initialize(ctx context.Context) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "application already initialized", nil).
			WithComponent("bootstrap")
	}

	app.Logger.InfoContext(ctx, "Initializing application components")

	// Initialize core components
	if err := app.initializeCoreComponents(ctx); err != nil {
		return err
	}

	// Initialize bridges
	if err := app.initializeBridges(ctx); err != nil {
		return err
	}

	// Initialize monitoring
	if err := app.initializeMonitoring(ctx); err != nil {
		return err
	}

	// Register all services
	if err := app.registerServices(ctx); err != nil {
		return err
	}

	// Set up dependencies
	if err := app.setupDependencies(ctx); err != nil {
		return err
	}

	app.Logger.InfoContext(ctx, "Application initialized successfully")
	return nil
}

// Start starts all application components
func (app *Application) Start(ctx context.Context) error {
	app.mu.Lock()
	if app.started {
		app.mu.Unlock()
		return gerror.New(gerror.ErrCodeAlreadyExists, "application already started", nil).
			WithComponent("bootstrap")
	}
	app.started = true
	app.mu.Unlock()

	app.Logger.InfoContext(ctx, "Starting application")

	// Start all services
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := app.ServiceRegistry.Start(startCtx); err != nil {
		app.mu.Lock()
		app.started = false
		app.mu.Unlock()
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start services").
			WithComponent("bootstrap")
	}

	// Emit startup event
	startupEvent := events.NewBaseEvent(
		"app-startup",
		"application.started",
		"bootstrap",
		map[string]interface{}{
			"version":    app.Config.Version,
			"start_time": time.Now(),
		},
	)

	if err := app.EventBus.Publish(ctx, startupEvent); err != nil {
		app.Logger.ErrorContext(ctx, "Failed to publish startup event", "error", err)
	}

	app.Logger.InfoContext(ctx, "Application started successfully")
	return nil
}

// Stop gracefully shuts down all application components
func (app *Application) Stop(ctx context.Context) error {
	app.mu.Lock()
	if !app.started {
		app.mu.Unlock()
		return gerror.New(gerror.ErrCodeValidation, "application not started", nil).
			WithComponent("bootstrap")
	}
	app.started = false
	app.mu.Unlock()

	app.Logger.InfoContext(ctx, "Stopping application")

	// Emit shutdown event
	shutdownEvent := events.NewBaseEvent(
		"app-shutdown",
		"application.stopping",
		"bootstrap",
		map[string]interface{}{
			"stop_time": time.Now(),
		},
	)

	if err := app.EventBus.Publish(ctx, shutdownEvent); err != nil {
		app.Logger.ErrorContext(ctx, "Failed to publish shutdown event", "error", err)
	}

	// Stop all services
	stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := app.ServiceRegistry.Stop(stopCtx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop services").
			WithComponent("bootstrap")
	}

	// Cancel application context
	app.cancel()

	app.Logger.InfoContext(ctx, "Application stopped successfully")
	return nil
}

// Run runs the application until interrupted
func (app *Application) Run() error {
	// Initialize
	if err := app.Initialize(app.ctx); err != nil {
		return err
	}

	// Start
	if err := app.Start(app.ctx); err != nil {
		return err
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt
	select {
	case sig := <-sigChan:
		app.Logger.InfoContext(app.ctx, "Received signal", "signal", sig)
	case <-app.ctx.Done():
		app.Logger.InfoContext(app.ctx, "Context cancelled")
	}

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return app.Stop(shutdownCtx)
}

// initializeCoreComponents initializes core application components
func (app *Application) initializeCoreComponents(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Initializing core components")

	// Initialize event bus
	eventBus := events.NewMemoryEventBusWithDefaults()
	app.EventBus = eventBus

	// Initialize component registry first
	componentReg := registry.NewComponentRegistry()
	app.ComponentRegistry = componentReg

	// Initialize storage using the factory method
	_, memoryStore, err := storage.InitializeSQLiteStorageForRegistry(ctx, app.Config.Storage.DatabasePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create storage").
			WithComponent("bootstrap")
	}

	// Create storage wrapper that implements our interface
	app.Storage = &storageAdapter{memoryStore: memoryStore}

	// Register storage in component registry
	// The storage registry needs to implement registry.StorageRegistry interface
	// For now, we'll set it using SetStorageRegistry
	if defaultReg, ok := componentReg.(*registry.DefaultComponentRegistry); ok {
		// Need to create a MemoryStore adapter that implements registry.MemoryStore
		defaultReg.SetStorageRegistry(nil, nil) // TODO: Implement proper adapter
	}

	return nil
}

// initializeBridges initializes integration bridges
func (app *Application) initializeBridges(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Initializing bridges")

	// Event logger bridge
	if app.Config.Bridges.EventLogging.Enabled {
		// Convert string log level to EventLogLevel
		var logLevel bridges.EventLogLevel
		switch app.Config.Bridges.EventLogging.LogLevel {
		case "debug":
			logLevel = bridges.LogLevelDebug
		case "info":
			logLevel = bridges.LogLevelInfo
		case "warn":
			logLevel = bridges.LogLevelWarn
		case "error":
			logLevel = bridges.LogLevelError
		default:
			logLevel = bridges.LogLevelInfo
		}

		config := bridges.EventLoggerConfig{
			LogLevel:         logLevel,
			IncludeEventData: app.Config.Bridges.EventLogging.IncludeData,
			BufferSize:       1000,
			FlushInterval:    100 * time.Millisecond,
			MaxBatchSize:     100,
		}
		app.EventLoggerBridge = bridges.NewEventLoggerBridge(app.EventBus, app.Logger, config)
	}

	// Persistence event bridge
	if app.Config.Bridges.PersistenceEvents.Enabled {
		// Note: PersistenceEventBridge expects a StorageRegistry, not our StorageInterface
		// For now, we'll skip persistence event bridge until we have proper integration
		// config := bridges.PersistenceEventConfig{
		//     EmitCRUDEvents:        app.Config.Bridges.PersistenceEvents.EmitCRUD,
		//     EmitQueryEvents:       app.Config.Bridges.PersistenceEvents.EmitQuery,
		//     EmitTransactionEvents: app.Config.Bridges.PersistenceEvents.EmitTransaction,
		//     IncludePayload:        app.Config.Bridges.PersistenceEvents.IncludePayload,
		// }
		// app.PersistenceEventBridge = bridges.NewPersistenceEventBridge(app.EventBus, app.Storage, app.Logger, config)
	}

	// UI event bridge
	if app.Config.Bridges.UIEvents.Enabled {
		config := bridges.UIEventConfig{
			BatchEvents:      app.Config.Bridges.UIEvents.BatchEvents,
			BatchInterval:    time.Duration(app.Config.Bridges.UIEvents.BatchIntervalMs) * time.Millisecond,
			MaxBatchSize:     app.Config.Bridges.UIEvents.MaxBatchSize,
			UIEventTypes:     app.Config.Bridges.UIEvents.UIEventTypes,
			SystemEventTypes: app.Config.Bridges.UIEvents.SystemEventTypes,
		}
		app.UIEventBridge = bridges.NewUIEventBridge(app.EventBus, app.Logger, config)
	}

	// Orchestrator Campaign Bridge - Always enable for multi-agent orchestration
	orchestratorConfig := bridges.OrchestratorCampaignConfig{
		Enabled:                true,
		ProcessCommissionsSync: true,
		MaxConcurrentAgents:    5,
	}

	// Create minimal adapters for the dependencies
	// The actual wiring will be done after services are registered
	app.OrchestratorCampaignBridge = bridges.NewOrchestratorCampaignBridge(
		app.EventBus,
		app.Logger,
		orchestratorConfig,
		nil, // Will be set after campaign manager is created
		nil, // Will be set after commission manager is created
		nil, // Will be set after task dispatcher is created
		nil, // Will be set after agent registry is available
	)

	// Agent Registration Bridge - Manages agent registration with task dispatcher
	agentRegConfig := bridges.AgentRegistrationConfig{
		Enabled:               true,
		AutoRegisterOnStartup: true,
		LoadFromGuildConfig:   true,
		GuildConfigPath:       "guild.yaml", // TODO: Get from actual config
		MaxAgents:             10,
	}

	app.AgentRegistrationBridge = bridges.NewAgentRegistrationBridge(
		app.EventBus,
		app.Logger,
		agentRegConfig,
		nil, // Will be set after agent registry is available
		nil, // Will be set after agent factory is created
		nil, // Will be set after task dispatcher is available
	)

	// Commission Processor Bridge - Handles commission to task conversion
	app.CommissionProcessorBridge = bridges.NewCommissionProcessorBridge(
		app.ComponentRegistry,
		app.Logger,
	)

	return nil
}

// initializeMonitoring initializes monitoring components
func (app *Application) initializeMonitoring(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Initializing monitoring")

	// For now, create a simple no-op monitor
	// TODO: Implement full monitoring when the package is available
	app.Monitor = &noOpMonitor{}

	return nil
}

// registerServices registers all services with the service registry
func (app *Application) registerServices(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Registering services")

	// Register bridges as services
	if app.EventLoggerBridge != nil {
		if err := app.ServiceRegistry.Register(app.EventLoggerBridge); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register event logger bridge").
				WithComponent("bootstrap")
		}
	}

	if app.PersistenceEventBridge != nil {
		if err := app.ServiceRegistry.Register(app.PersistenceEventBridge); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register persistence event bridge").
				WithComponent("bootstrap")
		}
	}

	if app.UIEventBridge != nil {
		if err := app.ServiceRegistry.Register(app.UIEventBridge); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register UI event bridge").
				WithComponent("bootstrap")
		}
	}

	if app.OrchestratorCampaignBridge != nil {
		if err := app.ServiceRegistry.Register(app.OrchestratorCampaignBridge); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register orchestrator campaign bridge").
				WithComponent("bootstrap")
		}
	}

	if app.AgentRegistrationBridge != nil {
		if err := app.ServiceRegistry.Register(app.AgentRegistrationBridge); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent registration bridge").
				WithComponent("bootstrap")
		}
	}

	// Note: CommissionProcessorBridge is not a service, it's just a helper bridge
	// It doesn't need to be registered with the service registry

	// Register core services

	// Register Kanban service
	kanbanConfig := services.KanbanServiceConfig{
		BoardPath:    ".guild/kanban",
		BoardName:    "Guild Tasks",
		Description:  "Task management board for Guild operations",
		AutoSave:     true,
		SaveInterval: 30 * time.Second,
	}

	kanbanService, err := services.NewKanbanService(
		app.ComponentRegistry,
		app.EventBus,
		app.Logger.WithComponent("KanbanService"),
		kanbanConfig,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban service").
			WithComponent("bootstrap")
	}

	if err := app.ServiceRegistry.Register(kanbanService); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register kanban service").
			WithComponent("bootstrap")
	}

	// Register Memory service
	memoryConfig := services.DefaultMemoryServiceConfig()

	memoryService, err := services.NewMemoryService(
		app.ComponentRegistry,
		app.EventBus,
		app.Logger.WithComponent("MemoryService"),
		memoryConfig,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create memory service").
			WithComponent("bootstrap")
	}

	if err := app.ServiceRegistry.Register(memoryService); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register memory service").
			WithComponent("bootstrap")
	}

	// Register Session service
	// Get session repository from storage registry
	storageReg := app.ComponentRegistry.Storage()
	if storageReg == nil {
		return gerror.New(gerror.ErrCodeInternal, "storage registry not available", nil).
			WithComponent("bootstrap")
	}

	sessionRepo := storageReg.GetSessionRepository()
	if sessionRepo == nil {
		return gerror.New(gerror.ErrCodeInternal, "session repository not available", nil).
			WithComponent("bootstrap")
	}

	// Create session store adapter that wraps the session repository
	sessionStore := &sessionStoreAdapter{repo: sessionRepo}

	// Create session manager with default options
	sessionManager := session.NewSessionManager(sessionStore)

	// Create session service
	sessionConfig := services.DefaultSessionServiceConfig()
	sessionService := services.NewSessionService(sessionManager, app.Logger.WithComponent("SessionService"), sessionConfig)

	if err := app.ServiceRegistry.Register(sessionService); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register session service").
			WithComponent("bootstrap")
	}

	// Register Orchestrator service
	orchestratorConfig := services.DefaultOrchestratorServiceConfig()

	orchestratorService, err := services.NewOrchestratorService(
		app.ComponentRegistry,
		app.EventBus,
		app.Logger.WithComponent("OrchestratorService"),
		orchestratorConfig,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create orchestrator service").
			WithComponent("bootstrap")
	}

	if err := app.ServiceRegistry.Register(orchestratorService); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register orchestrator service").
			WithComponent("bootstrap")
	}

	// Register Agent Manager service
	agentManagerConfig := services.DefaultAgentManagerServiceConfig()

	agentManagerService, err := services.NewAgentManagerService(
		app.ComponentRegistry,
		app.EventBus,
		app.Logger.WithComponent("AgentManagerService"),
		agentManagerConfig,
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent manager service").
			WithComponent("bootstrap")
	}

	if err := app.ServiceRegistry.Register(agentManagerService); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent manager service").
			WithComponent("bootstrap")
	}

	// Register Reasoning service
	// TODO: Implement reasoning service when ready
	// reasoningConfig := services.DefaultReasoningServiceConfig()

	// Get database from storage
	_, err = app.getDatabase()
	if err != nil {
		// Log but don't fail - reasoning service is optional for now
		app.Logger.WarnContext(ctx, "Failed to get database for reasoning service",
			"error", err)
	}

	// reasoningService, err := services.NewReasoningService(
	// 	app.ComponentRegistry,
	// 	app.EventBus,
	// 	app.Logger.WithComponent("ReasoningService"),
	// 	db,
	// 	reasoningConfig,
	// )
	// if err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create reasoning service").
	// 		WithComponent("bootstrap")
	// }

	// if err := app.ServiceRegistry.Register(reasoningService); err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register reasoning service").
	// 		WithComponent("bootstrap")
	// }

	// Register Chat UI service (optional, only if enabled)
	// Chat UI service will be registered conditionally based on the CLI command
	// For now, we'll add a placeholder comment
	// TODO: Add conditional Chat UI service registration based on runtime mode

	return nil
}

// setupDependencies sets up service dependencies
func (app *Application) setupDependencies(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Setting up service dependencies")

	// Bridges depend on event bus being available
	// Storage services depend on database being initialized
	// UI depends on event bridge being started

	// Core service dependencies
	// Memory service has no dependencies (uses registry components)
	// Kanban service depends on memory service for storage
	if err := app.ServiceRegistry.SetDependency("kanban-service", "memory-service"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set kanban->memory dependency").
			WithComponent("bootstrap")
	}

	// Orchestrator depends on kanban for task management
	if err := app.ServiceRegistry.SetDependency("orchestrator-service", "kanban-service"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set orchestrator->kanban dependency").
			WithComponent("bootstrap")
	}

	// Agent manager depends on memory for persistence
	if err := app.ServiceRegistry.SetDependency("agent-manager-service", "memory-service"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set agent-manager->memory dependency").
			WithComponent("bootstrap")
	}

	// Wire the orchestrator campaign bridge with actual services
	if app.OrchestratorCampaignBridge != nil {
		adapter := bridges.NewServiceRegistryAdapter(app.ServiceRegistry)
		if err := bridges.WireOrchestratorCampaignBridge(
			app.OrchestratorCampaignBridge,
			adapter,
		); err != nil {
			app.Logger.WithError(err).WarnContext(ctx, "Failed to wire orchestrator campaign bridge")
			// Don't fail - bridge will still emit events
		}
	}

	// Wire the agent registration bridge with actual services
	if app.AgentRegistrationBridge != nil {
		adapter := bridges.NewServiceRegistryAdapter(app.ServiceRegistry)
		if err := bridges.WireAgentRegistrationBridge(
			app.AgentRegistrationBridge,
			adapter,
			app.ComponentRegistry,
		); err != nil {
			app.Logger.WithError(err).WarnContext(ctx, "Failed to wire agent registration bridge")
			// Don't fail - bridge will still emit events
		}
	}

	// Initialize and wire the commission processor bridge
	if app.CommissionProcessorBridge != nil {
		// Initialize the commission processor (this loads the integration service)
		if err := app.CommissionProcessorBridge.Initialize(ctx); err != nil {
			app.Logger.WithError(err).WarnContext(ctx, "Failed to initialize commission processor bridge")
			// Don't fail - the orchestrator campaign bridge will emit events instead
		}

		// Wire commission processing to the orchestrator campaign bridge
		if app.OrchestratorCampaignBridge != nil {
			if err := bridges.WireCommissionProcessing(
				app.OrchestratorCampaignBridge,
				app.CommissionProcessorBridge,
			); err != nil {
				app.Logger.WithError(err).WarnContext(ctx, "Failed to wire commission processing")
				// Don't fail - the bridge will emit events for other components to handle
			} else {
				app.Logger.InfoContext(ctx, "Commission processing wired to campaign orchestration")
			}
		}
	}

	// TODO: Add more dependencies as services are registered

	return nil
}

// Health returns the health status of all components
func (app *Application) Health(ctx context.Context) map[string]error {
	return app.ServiceRegistry.Health(ctx)
}

// Ready checks if the application is ready to serve requests
func (app *Application) Ready(ctx context.Context) error {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if !app.started {
		return gerror.New(gerror.ErrCodeValidation, "application not started", nil).
			WithComponent("bootstrap")
	}

	// Check critical services
	health := app.ServiceRegistry.Health(ctx)
	for service, err := range health {
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "service unhealthy").
				WithComponent("bootstrap").
				WithDetails("service", service)
		}
	}

	return nil
}

// getDatabase returns the underlying database connection
func (app *Application) getDatabase() (*sql.DB, error) {
	// This is a temporary solution - we need to access the database
	// from the storage layer. In a proper implementation, the database
	// would be exposed through the StorageRegistry interface.

	// For now, we'll return an error indicating this needs implementation
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "database access not yet implemented", nil).
		WithComponent("bootstrap").
		WithDetails("operation", "getDatabase")
}

// storageAdapter adapts the memory store to our StorageInterface
type storageAdapter struct {
	memoryStore interface{}
}

// Close closes the storage
func (s *storageAdapter) Close() error {
	// The memory store adapter doesn't have a Close method
	// This is handled at the database level
	return nil
}

// sessionStoreAdapter adapts registry.SessionRepository to session.SessionStore
type sessionStoreAdapter struct {
	repo registry.SessionRepository
}

// GetSession retrieves a session by ID
func (a *sessionStoreAdapter) GetSession(ctx context.Context, sessionID string) (*storage.ChatSession, error) {
	// Convert from registry.ChatSession to storage.ChatSession
	regSession, err := a.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if regSession == nil {
		return nil, nil
	}

	// For now, do a simple conversion (assuming the types are compatible)
	// In a real implementation, we'd need proper mapping
	return &storage.ChatSession{
		ID:         regSession.ID,
		Name:       regSession.Name,
		CampaignID: regSession.CampaignID,
		CreatedAt:  regSession.CreatedAt,
		UpdatedAt:  regSession.UpdatedAt,
		Metadata:   regSession.Metadata,
	}, nil
}

// UpsertSession creates or updates a session
func (a *sessionStoreAdapter) UpsertSession(ctx context.Context, session *storage.ChatSession, stateData []byte) error {
	// For now, we'll ignore stateData as it's not supported by the repository interface
	// Convert from storage.ChatSession to registry.ChatSession
	regSession := &registry.ChatSession{
		ID:         session.ID,
		Name:       session.Name,
		CampaignID: session.CampaignID,
		CreatedAt:  session.CreatedAt,
		UpdatedAt:  session.UpdatedAt,
		Metadata:   session.Metadata,
	}

	if session.ID == "" {
		return a.repo.CreateSession(ctx, regSession)
	}
	return a.repo.UpdateSession(ctx, regSession)
}

// SaveMessage saves a message to a session
func (a *sessionStoreAdapter) SaveMessage(ctx context.Context, sessionID string, message *storage.ChatMessage) error {
	// Convert from storage.ChatMessage to registry.ChatMessage
	regMessage := &registry.ChatMessage{
		ID:        message.ID,
		SessionID: sessionID,
		Role:      message.Role,
		Content:   message.Content,
		CreatedAt: message.CreatedAt,
		ToolCalls: message.ToolCalls,
		Metadata:  message.Metadata,
	}
	return a.repo.SaveMessage(ctx, regMessage)
}

// GetMessages retrieves all messages for a session
func (a *sessionStoreAdapter) GetMessages(ctx context.Context, sessionID string) ([]*storage.ChatMessage, error) {
	// Get messages from registry repository
	regMessages, err := a.repo.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Convert from registry.ChatMessage to storage.ChatMessage
	messages := make([]*storage.ChatMessage, len(regMessages))
	for i, regMsg := range regMessages {
		messages[i] = &storage.ChatMessage{
			ID:        regMsg.ID,
			SessionID: regMsg.SessionID,
			Role:      regMsg.Role,
			Content:   regMsg.Content,
			CreatedAt: regMsg.CreatedAt,
			ToolCalls: regMsg.ToolCalls,
			Metadata:  regMsg.Metadata,
		}
	}
	return messages, nil
}

// ListSessions lists sessions with options
func (a *sessionStoreAdapter) ListSessions(ctx context.Context, options session.ListOptions) ([]*storage.ChatSession, error) {
	// The repository interface uses limit and offset directly
	regSessions, err := a.repo.ListSessions(ctx, int32(options.Limit), int32(options.Offset))
	if err != nil {
		return nil, err
	}

	// Convert from registry.ChatSession to storage.ChatSession
	sessions := make([]*storage.ChatSession, len(regSessions))
	for i, regSession := range regSessions {
		sessions[i] = &storage.ChatSession{
			ID:         regSession.ID,
			Name:       regSession.Name,
			CampaignID: regSession.CampaignID,
			CreatedAt:  regSession.CreatedAt,
			UpdatedAt:  regSession.UpdatedAt,
			Metadata:   regSession.Metadata,
		}
	}
	return sessions, nil
}

// Begin starts a transaction (not supported by the repository interface)
func (a *sessionStoreAdapter) Begin() (session.Transaction, error) {
	// For now, return an error as transactions are not supported
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "transactions not supported", nil).
		WithComponent("sessionStoreAdapter")
}

// noOpMonitor is a simple no-op monitor implementation
type noOpMonitor struct{}

// Start starts the monitor (no-op)
func (m *noOpMonitor) Start(ctx context.Context) error {
	return nil
}

// Stop stops the monitor (no-op)
func (m *noOpMonitor) Stop(ctx context.Context) error {
	return nil
}
