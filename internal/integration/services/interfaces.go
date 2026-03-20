// Package services provides service lifecycle management for Guild components
package services

import (
	"context"
	"time"
)

// Service represents a managed service with lifecycle hooks
type Service interface {
	// Name returns the service name
	Name() string

	// Start initializes and starts the service
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service
	Stop(ctx context.Context) error

	// Health checks if the service is healthy
	Health(ctx context.Context) error

	// Ready checks if the service is ready to handle requests
	Ready(ctx context.Context) error
}

// ServiceState represents the current state of a service
type ServiceState string

const (
	StateUnknown      ServiceState = "unknown"
	StateInitializing ServiceState = "initializing"
	StateStarting     ServiceState = "starting"
	StateRunning      ServiceState = "running"
	StateStopping     ServiceState = "stopping"
	StateStopped      ServiceState = "stopped"
	StateError        ServiceState = "error"
)

// ServiceInfo contains metadata about a service
type ServiceInfo struct {
	Name         string
	State        ServiceState
	StartedAt    time.Time
	StoppedAt    time.Time
	LastHealthAt time.Time
	Healthy      bool
	Ready        bool
	Error        error
	Dependencies []string
}

// ServiceRegistry manages service lifecycle and discovery
type ServiceRegistry interface {
	// Register adds a service to the registry
	Register(service Service) error

	// Unregister removes a service from the registry
	Unregister(name string) error

	// Get retrieves a service by name
	Get(name string) (Service, error)

	// List returns all registered services
	List() []ServiceInfo

	// Start starts all services in dependency order
	Start(ctx context.Context) error

	// Stop stops all services in reverse dependency order
	Stop(ctx context.Context) error

	// Health checks health of all services
	Health(ctx context.Context) map[string]error

	// SetDependency declares that service A depends on service B
	SetDependency(serviceA, serviceB string) error
}

// ServiceOptions configures service behavior
type ServiceOptions struct {
	// StartTimeout is the maximum time to wait for service to start
	StartTimeout time.Duration

	// StopTimeout is the maximum time to wait for service to stop
	StopTimeout time.Duration

	// HealthCheckInterval is how often to check service health
	HealthCheckInterval time.Duration

	// ReadinessCheckInterval is how often to check service readiness
	ReadinessCheckInterval time.Duration

	// MaxRetries is the number of times to retry starting a service
	MaxRetries int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// Dependencies lists services this service depends on
	Dependencies []string
}

// DefaultServiceOptions returns sensible defaults
func DefaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		StartTimeout:           30 * time.Second,
		StopTimeout:            30 * time.Second,
		HealthCheckInterval:    10 * time.Second,
		ReadinessCheckInterval: 5 * time.Second,
		MaxRetries:             3,
		RetryDelay:             1 * time.Second,
	}
}

// ServiceManager provides lifecycle management for a group of services
type ServiceManager interface {
	// AddService adds a service to be managed
	AddService(service Service, opts ServiceOptions) error

	// RemoveService removes a service from management
	RemoveService(name string) error

	// StartAll starts all services in dependency order
	StartAll(ctx context.Context) error

	// StopAll stops all services in reverse dependency order
	StopAll(ctx context.Context) error

	// RestartService restarts a specific service
	RestartService(ctx context.Context, name string) error

	// GetServiceInfo returns information about a service
	GetServiceInfo(name string) (ServiceInfo, error)

	// ListServices returns information about all services
	ListServices() []ServiceInfo

	// WaitForReady waits for all services to be ready
	WaitForReady(ctx context.Context, timeout time.Duration) error
}

// ServiceHook allows intercepting service lifecycle events
type ServiceHook interface {
	// OnStart is called before a service starts
	OnStart(ctx context.Context, service Service) error

	// OnStarted is called after a service successfully starts
	OnStarted(ctx context.Context, service Service)

	// OnStop is called before a service stops
	OnStop(ctx context.Context, service Service) error

	// OnStopped is called after a service successfully stops
	OnStopped(ctx context.Context, service Service)

	// OnError is called when a service encounters an error
	OnError(ctx context.Context, service Service, err error)

	// OnHealthCheck is called after each health check
	OnHealthCheck(ctx context.Context, service Service, healthy bool, err error)
}
