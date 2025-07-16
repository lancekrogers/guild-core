// Package bootstrap provides application startup and shutdown orchestration
package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lancekrogers/guild/internal/integration/bridges"
	"github.com/lancekrogers/guild/internal/integration/services"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/logging"
	"github.com/lancekrogers/guild/pkg/monitoring"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
)

// Application represents the main application with all integrated components
type Application struct {
	// Core components
	Config            *config.Config
	Logger            observability.Logger
	EventBus          events.EventBus
	Storage           storage.Storage
	ComponentRegistry registry.ComponentRegistry
	ServiceRegistry   *services.DefaultServiceRegistry

	// Bridges
	EventLoggerBridge      *bridges.EventLoggerBridge
	PersistenceEventBridge *bridges.PersistenceEventBridge
	UIEventBridge          *bridges.UIEventBridge

	// Monitoring
	Monitor *monitoring.Monitor

	// State
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	mu      sync.RWMutex
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
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		cancel()
		return nil, gerror.Wrap(gerror.ErrCodeInternal, err, "failed to load configuration").
			WithComponent("bootstrap")
	}

	// Initialize logger
	logger := logging.NewLogger(logging.Config{
		Level:  opts.LogLevel,
		Format: "json",
	})

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
		return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to start services").
			WithComponent("bootstrap")
	}

	// Emit startup event
	startupEvent := events.NewEvent("application.started", map[string]interface{}{
		"version":    app.Config.Version,
		"start_time": time.Now(),
	}).WithSource("bootstrap")

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
	shutdownEvent := events.NewEvent("application.stopping", map[string]interface{}{
		"stop_time": time.Now(),
	}).WithSource("bootstrap")

	if err := app.EventBus.Publish(ctx, shutdownEvent); err != nil {
		app.Logger.ErrorContext(ctx, "Failed to publish shutdown event", "error", err)
	}

	// Stop all services
	stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := app.ServiceRegistry.Stop(stopCtx); err != nil {
		return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to stop services").
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
	eventBus, err := events.NewMemoryEventBus(ctx)
	if err != nil {
		return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to create event bus").
			WithComponent("bootstrap")
	}
	app.EventBus = eventBus

	// Initialize storage
	storageConfig := storage.Config{
		DatabasePath: app.Config.Storage.DatabasePath,
		Migrations:   app.Config.Storage.MigrationsPath,
	}

	store, err := storage.NewSQLiteStorage(ctx, storageConfig)
	if err != nil {
		return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to create storage").
			WithComponent("bootstrap")
	}
	app.Storage = store

	// Initialize component registry
	componentReg := registry.NewDefaultComponentRegistry()
	app.ComponentRegistry = componentReg

	return nil
}

// initializeBridges initializes integration bridges
func (app *Application) initializeBridges(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Initializing bridges")

	// Event logger bridge
	if app.Config.Bridges.EventLogging.Enabled {
		config := bridges.EventLoggerConfig{
			LogLevel:         bridges.EventLogLevel(app.Config.Bridges.EventLogging.LogLevel),
			IncludeEventData: app.Config.Bridges.EventLogging.IncludeData,
			BufferSize:       1000,
			FlushInterval:    100 * time.Millisecond,
			MaxBatchSize:     100,
		}
		app.EventLoggerBridge = bridges.NewEventLoggerBridge(app.EventBus, app.Logger, config)
	}

	// Persistence event bridge
	if app.Config.Bridges.PersistenceEvents.Enabled {
		config := bridges.PersistenceEventConfig{
			EmitCRUDEvents:        app.Config.Bridges.PersistenceEvents.EmitCRUD,
			EmitQueryEvents:       app.Config.Bridges.PersistenceEvents.EmitQuery,
			EmitTransactionEvents: app.Config.Bridges.PersistenceEvents.EmitTransaction,
			IncludePayload:        app.Config.Bridges.PersistenceEvents.IncludePayload,
		}
		app.PersistenceEventBridge = bridges.NewPersistenceEventBridge(app.EventBus, app.Storage, app.Logger, config)
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

	return nil
}

// initializeMonitoring initializes monitoring components
func (app *Application) initializeMonitoring(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Initializing monitoring")

	// Create monitor
	monitorConfig := monitoring.Config{
		MetricsEnabled:   app.Config.Monitoring.MetricsEnabled,
		TracingEnabled:   app.Config.Monitoring.TracingEnabled,
		ProfilingEnabled: app.Config.Monitoring.ProfilingEnabled,
	}

	monitor, err := monitoring.NewMonitor(ctx, monitorConfig)
	if err != nil {
		return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to create monitor").
			WithComponent("bootstrap")
	}

	app.Monitor = monitor
	return nil
}

// registerServices registers all services with the service registry
func (app *Application) registerServices(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Registering services")

	// Register bridges as services
	if app.EventLoggerBridge != nil {
		if err := app.ServiceRegistry.Register(app.EventLoggerBridge); err != nil {
			return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to register event logger bridge").
				WithComponent("bootstrap")
		}
	}

	if app.PersistenceEventBridge != nil {
		if err := app.ServiceRegistry.Register(app.PersistenceEventBridge); err != nil {
			return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to register persistence event bridge").
				WithComponent("bootstrap")
		}
	}

	if app.UIEventBridge != nil {
		if err := app.ServiceRegistry.Register(app.UIEventBridge); err != nil {
			return gerror.Wrap(gerror.ErrCodeInternal, err, "failed to register UI event bridge").
				WithComponent("bootstrap")
		}
	}

	// Register core services
	// TODO: Register other services like daemon, chat UI, etc.

	// Note: Session service would be registered here when session manager is available
	// Example:
	// if sessionManager := app.ComponentRegistry.GetSessionManager(); sessionManager != nil {
	//     sessionService := services.NewSessionService(sessionManager, app.Logger, services.DefaultSessionServiceConfig())
	//     app.ServiceRegistry.Register(sessionService)
	// }

	return nil
}

// setupDependencies sets up service dependencies
func (app *Application) setupDependencies(ctx context.Context) error {
	app.Logger.InfoContext(ctx, "Setting up service dependencies")

	// Bridges depend on event bus being available
	// Storage services depend on database being initialized
	// UI depends on event bridge being started

	// Example dependencies:
	// app.ServiceRegistry.SetDependency("ui-event-bridge", "event-bus")
	// app.ServiceRegistry.SetDependency("chat-ui", "ui-event-bridge")

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
		return gerror.New(gerror.ErrCodeUnavailable, "application not started", nil).
			WithComponent("bootstrap")
	}

	// Check critical services
	health := app.ServiceRegistry.Health(ctx)
	for service, err := range health {
		if err != nil {
			return gerror.Wrap(gerror.ErrCodeUnavailable, err, "service unhealthy").
				WithComponent("bootstrap").
				WithDetails("service", service)
		}
	}

	return nil
}
