// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/lancekrogers/guild/internal/daemon"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// DaemonService wraps the daemon server to integrate with the service framework
type DaemonService struct {
	registry registry.ComponentRegistry
	eventBus events.EventBus
	logger   observability.Logger
	config   DaemonServiceConfig

	// gRPC server components
	grpcServer   *grpc.Server
	grpcManager  *GRPCServiceManager
	healthServer *health.Server
	listener     net.Listener

	// HTTP/WebSocket servers (future)
	// httpServer     *http.Server
	// wsServer       *websocket.Server

	// Service state
	started bool
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex

	// Metrics
	requestsHandled   uint64
	activeConnections int32
	errors            uint64
	startTime         time.Time
}

// DaemonServiceConfig configures the daemon service
type DaemonServiceConfig struct {
	// Network configuration
	GRPCPort    int
	HTTPPort    int
	MetricsPort int

	// TLS configuration
	TLSEnabled bool
	CertPath   string
	KeyPath    string

	// Server options
	MaxConnections      int
	ConnectionTimeout   time.Duration
	KeepAliveInterval   time.Duration
	GracefulStopTimeout time.Duration

	// Features
	EnableReflection bool
	EnableMetrics    bool
	EnableProfiling  bool
}

// DefaultDaemonServiceConfig returns default configuration
func DefaultDaemonServiceConfig() DaemonServiceConfig {
	return DaemonServiceConfig{
		GRPCPort:            9090,
		HTTPPort:            8080,
		MetricsPort:         9091,
		TLSEnabled:          false,
		MaxConnections:      1000,
		ConnectionTimeout:   30 * time.Second,
		KeepAliveInterval:   30 * time.Second,
		GracefulStopTimeout: 30 * time.Second,
		EnableReflection:    true,
		EnableMetrics:       true,
		EnableProfiling:     false,
	}
}

// NewDaemonService creates a new daemon service wrapper
func NewDaemonService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config DaemonServiceConfig,
) (*DaemonService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("DaemonService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("DaemonService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("DaemonService")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DaemonService{
		registry:     registry,
		eventBus:     eventBus,
		logger:       logger,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		healthServer: health.NewServer(),
	}, nil
}

// Name returns the service name
func (s *DaemonService) Name() string {
	return "daemon-service"
}

// Start initializes and starts the service
func (s *DaemonService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("DaemonService")
	}

	// Check if another daemon is already running
	if daemon.IsRunning() {
		return gerror.New(gerror.ErrCodeAlreadyExists, "daemon already running on port", nil).
			WithComponent("DaemonService").
			WithDetails("port", s.config.GRPCPort)
	}

	// Create gRPC server with interceptors
	s.grpcServer = s.createGRPCServer()

	// Create gRPC service manager
	var err error
	s.grpcManager, err = NewGRPCServiceManager(s.registry, s.eventBus, s.logger, GRPCServiceManagerConfig{
		EnableHealth:     true,
		EnableReflection: s.config.EnableReflection,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create gRPC service manager").
			WithComponent("DaemonService")
	}

	// Register all gRPC services
	if err := s.grpcManager.RegisterServices(s.grpcServer); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register gRPC services").
			WithComponent("DaemonService")
	}

	// Register health service
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Enable reflection if configured
	if s.config.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	// Create listener
	addr := fmt.Sprintf(":%d", s.config.GRPCPort)
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create listener").
			WithComponent("DaemonService").
			WithDetails("address", addr)
	}

	// Start gRPC server in background
	go s.serve()

	// Write PID file for daemon management
	if err := s.writePIDFile(); err != nil {
		s.logger.WarnContext(ctx, "Failed to write PID file", "error", err)
	}

	s.started = true
	s.startTime = time.Now()

	// Set health status
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"daemon-service-started",
		"service.started",
		"daemon",
		map[string]interface{}{
			"grpc_port":          s.config.GRPCPort,
			"http_port":          s.config.HTTPPort,
			"metrics_port":       s.config.MetricsPort,
			"tls_enabled":        s.config.TLSEnabled,
			"reflection_enabled": s.config.EnableReflection,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Daemon service started",
		"grpc_port", s.config.GRPCPort,
		"tls_enabled", s.config.TLSEnabled)

	return nil
}

// Stop gracefully shuts down the service
func (s *DaemonService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("DaemonService")
	}

	// Set health status to not serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Cancel context to signal shutdown
	s.cancel()

	// Graceful shutdown with timeout
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Graceful stop completed
	case <-time.After(s.config.GracefulStopTimeout):
		// Force stop after timeout
		s.grpcServer.Stop()
	}

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Remove PID file
	if err := s.removePIDFile(); err != nil {
		s.logger.WarnContext(ctx, "Failed to remove PID file", "error", err)
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"daemon-service-stopped",
		"service.stopped",
		"daemon",
		map[string]interface{}{
			"requests_handled":   s.requestsHandled,
			"active_connections": s.activeConnections,
			"errors":             s.errors,
			"uptime_seconds":     uptime.Seconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false
	s.running = false

	s.logger.InfoContext(ctx, "Daemon service stopped",
		"requests_handled", s.requestsHandled,
		"uptime", uptime)

	return nil
}

// Health checks if the service is healthy
func (s *DaemonService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("DaemonService")
	}

	// Check if gRPC server is serving
	if !s.running {
		return gerror.New(gerror.ErrCodeResourceExhausted, "gRPC server not running", nil).
			WithComponent("DaemonService")
	}

	// Check sub-services health
	if s.grpcManager != nil {
		if err := s.grpcManager.Health(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "gRPC services unhealthy").
				WithComponent("DaemonService")
		}
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *DaemonService) Ready(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started || !s.running {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not ready", nil).
			WithComponent("DaemonService")
	}

	// Check if listener is active
	if s.listener == nil {
		return gerror.New(gerror.ErrCodeResourceExhausted, "no active listener", nil).
			WithComponent("DaemonService")
	}

	return nil
}

// serve runs the gRPC server
func (s *DaemonService) serve() {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	s.logger.Info("gRPC server starting", "address", s.listener.Addr())

	// Emit server running event
	if err := s.eventBus.Publish(s.ctx, events.NewBaseEvent(
		"grpc-server-running",
		"server.running",
		"daemon",
		map[string]interface{}{
			"address": s.listener.Addr().String(),
		},
	)); err != nil {
		s.logger.Warn("Failed to publish server running event", "error", err)
	}

	// Serve blocks until stopped
	if err := s.grpcServer.Serve(s.listener); err != nil {
		s.mu.Lock()
		s.errors++
		s.mu.Unlock()

		s.logger.Error("gRPC server error", "error", err)

		// Emit server error event
		if err := s.eventBus.Publish(s.ctx, events.NewBaseEvent(
			"grpc-server-error",
			"server.error",
			"daemon",
			map[string]interface{}{
				"error": err.Error(),
			},
		)); err != nil {
			s.logger.Warn("Failed to publish server error event", "error", err)
		}
	}

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// createGRPCServer creates the gRPC server with interceptors
func (s *DaemonService) createGRPCServer() *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(uint32(s.config.MaxConnections)),
		grpc.ConnectionTimeout(s.config.ConnectionTimeout),
		// Add interceptors for logging, metrics, etc.
		grpc.UnaryInterceptor(s.unaryInterceptor),
		grpc.StreamInterceptor(s.streamInterceptor),
	}

	// TODO: Add TLS configuration if enabled
	// if s.config.TLSEnabled {
	//     creds, err := credentials.NewServerTLSFromFile(s.config.CertPath, s.config.KeyPath)
	//     if err != nil {
	//         s.logger.Error("Failed to load TLS credentials", "error", err)
	//     } else {
	//         opts = append(opts, grpc.Creds(creds))
	//     }
	// }

	return grpc.NewServer(opts...)
}

// unaryInterceptor handles unary RPC calls
func (s *DaemonService) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Increment active connections
	s.mu.Lock()
	s.activeConnections++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeConnections--
		s.requestsHandled++
		s.mu.Unlock()
	}()

	// Call handler
	resp, err := handler(ctx, req)

	// Log and emit metrics
	duration := time.Since(start)
	if err != nil {
		s.mu.Lock()
		s.errors++
		s.mu.Unlock()

		s.logger.ErrorContext(ctx, "gRPC request failed",
			"method", info.FullMethod,
			"duration", duration,
			"error", err)
	} else {
		s.logger.DebugContext(ctx, "gRPC request completed",
			"method", info.FullMethod,
			"duration", duration)
	}

	// Emit request event
	go func() {
		eventErr := s.eventBus.Publish(context.Background(), events.NewBaseEvent(
			"grpc-request",
			"grpc.request",
			"daemon",
			map[string]interface{}{
				"method":   info.FullMethod,
				"duration": duration.Milliseconds(),
				"success":  err == nil,
			},
		))
		if eventErr != nil {
			s.logger.Warn("Failed to publish request event", "error", eventErr)
		}
	}()

	return resp, err
}

// streamInterceptor handles streaming RPC calls
func (s *DaemonService) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()

	// Increment active connections
	s.mu.Lock()
	s.activeConnections++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeConnections--
		s.requestsHandled++
		s.mu.Unlock()
	}()

	// Call handler
	err := handler(srv, ss)

	// Log and emit metrics
	duration := time.Since(start)
	if err != nil {
		s.mu.Lock()
		s.errors++
		s.mu.Unlock()

		s.logger.ErrorContext(ss.Context(), "gRPC stream failed",
			"method", info.FullMethod,
			"duration", duration,
			"error", err)
	} else {
		s.logger.DebugContext(ss.Context(), "gRPC stream completed",
			"method", info.FullMethod,
			"duration", duration)
	}

	return err
}

// writePIDFile writes the daemon PID file
func (s *DaemonService) writePIDFile() error {
	pidFile := daemon.GetPIDFilePath()
	pid := os.Getpid()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create PID directory").
			WithComponent("DaemonService")
	}

	// Write PID
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write PID file").
			WithComponent("DaemonService")
	}

	return nil
}

// removePIDFile removes the daemon PID file
func (s *DaemonService) removePIDFile() error {
	pidFile := daemon.GetPIDFilePath()
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove PID file").
			WithComponent("DaemonService")
	}
	return nil
}

// GetMetrics returns service metrics
func (s *DaemonService) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uptime := time.Duration(0)
	if s.started {
		uptime = time.Since(s.startTime)
	}

	return map[string]interface{}{
		"requests_handled":   s.requestsHandled,
		"active_connections": s.activeConnections,
		"errors":             s.errors,
		"uptime_seconds":     uptime.Seconds(),
		"is_running":         s.running,
		"grpc_port":          s.config.GRPCPort,
	}
}
