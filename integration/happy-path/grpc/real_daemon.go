// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/grpc"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// RealDaemon wraps the actual gRPC server for integration testing
type RealDaemon struct {
	config    DaemonConfig
	server    *grpc.Server
	registry  registry.ComponentRegistry
	eventBus  grpc.EventBus
	address   string
	port      int
	running   bool
	startTime time.Time
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	listener  net.Listener
}

// NewRealDaemon creates a new real daemon instance for testing
func NewRealDaemon(config DaemonConfig) (*RealDaemon, error) {
	// Create registry for real backend integration
	reg := registry.NewComponentRegistry()

	// Initialize with minimal test configuration
	err := reg.Initialize(context.Background(), registry.Config{
		// Use memory-based configuration for testing
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Create event bus for real-time communication
	eventBus := NewTestEventBus()

	// Create the real gRPC server
	server := grpc.NewServer(reg, eventBus)

	// Create context for daemon lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Find available port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to listen on port %d: %w", config.Port, err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)

	return &RealDaemon{
		config:   config,
		server:   server,
		registry: reg,
		eventBus: eventBus,
		address:  address,
		port:     port,
		running:  false,
		ctx:      ctx,
		cancel:   cancel,
		listener: listener,
	}, nil
}

// Start starts the real gRPC daemon
func (d *RealDaemon) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("daemon already running")
	}

	// Start the real gRPC server
	go func() {
		if err := d.server.Start(d.ctx, d.address); err != nil {
			observability.GetLogger(d.ctx).ErrorContext(d.ctx, "Failed to start real gRPC server", "error", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	d.running = true
	d.startTime = time.Now()

	return nil
}

// Address returns the daemon's address (compatible with MockDaemon interface)
func (d *RealDaemon) Address() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.address
}

// Stop stops the daemon (compatible with MockDaemon interface)
func (d *RealDaemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	// Stop the real gRPC server
	if d.server != nil {
		d.server.Stop()
	}

	// Close listener
	if d.listener != nil {
		d.listener.Close()
	}

	// Cancel context
	if d.cancel != nil {
		d.cancel()
	}

	d.running = false
	return nil
}

// GetResourceUsage returns current resource usage (compatible with MockDaemon interface)
func (d *RealDaemon) GetResourceUsage() ResourceUsage {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Get actual resource usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memoryMB := float64(memStats.Alloc) / (1024 * 1024)
	goroutines := runtime.NumGoroutine()

	// For CPU, we'd need a more sophisticated measurement
	// For testing purposes, provide a reasonable estimate
	cpuPercent := 10.0 // Base CPU usage
	if d.running {
		cpuPercent += 5.0 // Add load when running
	}

	return ResourceUsage{
		MemoryMB:   memoryMB,
		CPUPercent: cpuPercent,
		Goroutines: goroutines,
	}
}

// IsHealthy checks if daemon is healthy (compatible with MockDaemon interface)
func (d *RealDaemon) IsHealthy() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.running {
		return false
	}

	// For real implementation, we could check:
	// - Server is responding to health checks
	// - Resource usage is within limits
	// - No critical errors in logs

	// Simple health check: running and started recently enough
	return time.Since(d.startTime) < 24*time.Hour
}

// Restart simulates a daemon restart for failure testing
func (d *RealDaemon) Restart() error {
	d.mu.Lock()
	wasRunning := d.running
	d.mu.Unlock()

	if wasRunning {
		if err := d.Stop(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		// Simulate restart delay
		time.Sleep(d.config.HealthCheckInterval)

		if err := d.Start(); err != nil {
			return fmt.Errorf("failed to restart daemon: %w", err)
		}
	}

	return nil
}

// SimulateFailure simulates various failure types for testing
func (d *RealDaemon) SimulateFailure(failureType FailureType) error {
	switch failureType {
	case FailureType_ProcessCrash:
		// Stop the daemon and restart after delay
		go func() {
			d.Stop()
			time.Sleep(2 * time.Second)
			d.Start()
		}()
		return nil

	case FailureType_NetworkPartition:
		// Close listener temporarily to simulate network issues
		if d.listener != nil {
			d.listener.Close()
			go func() {
				time.Sleep(5 * time.Second)
				// Create new listener for recovery
				listener, err := net.Listen("tcp", fmt.Sprintf(":%d", d.config.Port))
				if err == nil {
					d.mu.Lock()
					d.listener = listener
					d.mu.Unlock()
				}
			}()
		}
		return nil

	case FailureType_ResourceExhaustion:
		// Simulate resource exhaustion by consuming memory/CPU
		// For testing purposes, just mark as unhealthy temporarily
		go func() {
			d.mu.Lock()
			originalStartTime := d.startTime
			d.startTime = time.Now().Add(-25 * time.Hour) // Make it appear unhealthy
			d.mu.Unlock()

			time.Sleep(3 * time.Second) // Simulate recovery time

			d.mu.Lock()
			d.startTime = originalStartTime
			d.mu.Unlock()
		}()
		return nil

	default:
		return fmt.Errorf("unknown failure type: %v", failureType)
	}
}

// TestEventBus implementation is in types.go

// GetServer returns the underlying gRPC server for advanced testing
func (d *RealDaemon) GetServer() *grpc.Server {
	return d.server
}

// GetRegistry returns the component registry for testing integration
func (d *RealDaemon) GetRegistry() registry.ComponentRegistry {
	return d.registry
}

// WaitForReady waits for the daemon to be ready for connections
func (d *RealDaemon) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if d.IsHealthy() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon not ready within timeout %v", timeout)
}
