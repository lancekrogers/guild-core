// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/lancekrogers/guild-core/pkg/observability"
)

// HealthService implements the gRPC health checking protocol
type HealthService struct {
	grpc_health_v1.UnimplementedHealthServer

	services map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
	watchers map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus
	mu       sync.RWMutex
}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{
		services: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
		watchers: make(map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
}

// Check implements health checking for individual services
func (h *HealthService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("HealthCheck")

	service := req.GetService()
	logger.Debug("Health check requested", "service", service)

	h.mu.RLock()
	status, exists := h.services[service]
	h.mu.RUnlock()

	if !exists {
		// If service not registered, default to serving for backwards compatibility
		status = grpc_health_v1.HealthCheckResponse_SERVING
	}

	logger.Info("Health check result",
		"service", service,
		"status", status.String(),
	)

	return &grpc_health_v1.HealthCheckResponse{
		Status: status,
	}, nil
}

// Watch implements health status streaming
func (h *HealthService) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	service := req.GetService()

	logger := observability.GetLogger(stream.Context()).
		WithComponent("grpc").
		WithOperation("HealthWatch")

	logger.Info("Health watch started", "service", service)

	// Create watcher channel
	watcher := make(chan grpc_health_v1.HealthCheckResponse_ServingStatus, 10)

	// Register watcher
	h.mu.Lock()
	h.watchers[service] = append(h.watchers[service], watcher)
	currentStatus := h.services[service]
	if currentStatus == 0 {
		currentStatus = grpc_health_v1.HealthCheckResponse_SERVING
	}
	h.mu.Unlock()

	// Send initial status
	if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: currentStatus,
	}); err != nil {
		h.removeWatcher(service, watcher)
		return err
	}

	// Stream status updates
	for {
		select {
		case <-stream.Context().Done():
			h.removeWatcher(service, watcher)
			logger.Info("Health watch ended", "service", service)
			return stream.Context().Err()
		case status := <-watcher:
			if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
				Status: status,
			}); err != nil {
				h.removeWatcher(service, watcher)
				logger.WithError(err).Warn("Failed to send health status", "service", service)
				return err
			}
		}
	}
}

// SetServingStatus sets the health status for a service
func (h *HealthService) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.services[service] = status

	// Notify watchers
	watchers := h.watchers[service]
	for _, watcher := range watchers {
		select {
		case watcher <- status:
		default:
			// Channel full, skip
		}
	}
}

// Shutdown gracefully shuts down all health watchers
func (h *HealthService) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all watcher channels
	for service, watchers := range h.watchers {
		for _, watcher := range watchers {
			close(watcher)
		}
		delete(h.watchers, service)
	}

	// Set all services to not serving
	for service := range h.services {
		h.services[service] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return nil
}

// removeWatcher removes a watcher from the service's watcher list
func (h *HealthService) removeWatcher(service string, target chan grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	watchers := h.watchers[service]
	for i, watcher := range watchers {
		if watcher == target {
			// Remove from slice
			h.watchers[service] = append(watchers[:i], watchers[i+1:]...)
			close(target)
			break
		}
	}
}

// RegisterServices registers all Guild services with initial health status
func (h *HealthService) RegisterServices() {
	services := []string{
		"guild.v1.Guild",
		"guild.v1.SessionService",
		"guild.v1.EventService",
		"prompts.v1.PromptsService",
	}

	for _, service := range services {
		h.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
	}

	// Register overall server health
	h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
}

// PeriodicHealthCheck runs periodic health checks on dependent services
func (h *HealthService) PeriodicHealthCheck(ctx context.Context, checker func() error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("PeriodicHealthCheck")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := checker(); err != nil {
				logger.WithError(err).Warn("Health check failed")
				h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			} else {
				h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
			}
		}
	}
}
