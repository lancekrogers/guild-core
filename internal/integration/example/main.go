// Package main demonstrates the integrated Guild application
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/guild/internal/integration/bridges"
	"github.com/lancekrogers/guild/internal/integration/services"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/observability"
)

func main() {
	// Create a simple demo without full application bootstrap
	ctx := context.Background()

	// Create logger
	logger := observability.NewLogger(&observability.Config{
		Level:  observability.LevelInfo,
		Format: "json",
	})

	logger.InfoContext(ctx, "Starting Guild integration demo")

	// Create event bus
	eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())

	// Create service registry
	serviceRegistry := services.NewServiceRegistry(ctx)

	// Create and register event logger bridge
	eventLoggerConfig := bridges.DefaultEventLoggerConfig()
	eventLoggerBridge := bridges.NewEventLoggerBridge(eventBus, logger, eventLoggerConfig)

	if err := serviceRegistry.Register(eventLoggerBridge); err != nil {
		log.Fatalf("Failed to register event logger bridge: %v", err)
	}

	// Create a demo service
	demoService := &DemoService{
		eventBus: eventBus,
		logger:   logger,
	}

	if err := serviceRegistry.Register(demoService); err != nil {
		log.Fatalf("Failed to register demo service: %v", err)
	}

	// Set dependency - demo service depends on event logger
	if err := serviceRegistry.SetDependency("demo-service", "event-logger-bridge"); err != nil {
		log.Fatalf("Failed to set dependency: %v", err)
	}

	// Start all services
	logger.InfoContext(ctx, "Starting services...")
	if err := serviceRegistry.Start(ctx); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}

	// List running services
	services := serviceRegistry.List()
	logger.InfoContext(ctx, "Running services:")
	for _, svc := range services {
		logger.InfoContext(ctx, fmt.Sprintf("  - %s: %s", svc.Name, svc.State))
	}

	// Publish some test events
	logger.InfoContext(ctx, "Publishing test events...")

	for i := 0; i < 5; i++ {
		event := events.NewBaseEvent(
			fmt.Sprintf("demo-event-%d", i),
			"demo.test.event",
			"demo-service",
			map[string]interface{}{
				"index":   i,
				"message": fmt.Sprintf("Test event %d", i),
			},
		)

		if err := eventBus.Publish(ctx, event); err != nil {
			logger.ErrorContext(ctx, "Failed to publish event", "error", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Check health
	logger.InfoContext(ctx, "Checking service health...")
	health := serviceRegistry.Health(ctx)
	for svc, err := range health {
		if err != nil {
			logger.ErrorContext(ctx, "Service unhealthy", "service", svc, "error", err)
		} else {
			logger.InfoContext(ctx, "Service healthy", "service", svc)
		}
	}

	// Get metrics
	metrics := eventLoggerBridge.GetMetrics()
	logger.InfoContext(ctx, "Event logger metrics",
		"events_logged", metrics.EventsLogged,
		"events_filtered", metrics.EventsFiltered,
		"errors", metrics.Errors,
	)

	// Wait a bit for events to process
	time.Sleep(1 * time.Second)

	// Stop all services
	logger.InfoContext(ctx, "Stopping services...")
	if err := serviceRegistry.Stop(ctx); err != nil {
		logger.ErrorContext(ctx, "Error stopping services", "error", err)
	}

	logger.InfoContext(ctx, "Demo completed")
}

// DemoService is a simple demonstration service
type DemoService struct {
	eventBus events.EventBus
	logger   observability.Logger
	started  bool
}

func (s *DemoService) Name() string {
	return "demo-service"
}

func (s *DemoService) Start(ctx context.Context) error {
	s.started = true
	s.logger.InfoContext(ctx, "Demo service started")

	// Subscribe to events
	handler := func(ctx context.Context, event events.CoreEvent) error {
		s.logger.InfoContext(ctx, "Demo service received event",
			"type", event.GetType(),
			"id", event.GetID(),
		)
		return nil
	}

	// Subscribe to all events for demo
	subID, err := s.eventBus.SubscribeAll(ctx, handler)
	if err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "Demo service subscribed to events", "subscription", subID)
	return nil
}

func (s *DemoService) Stop(ctx context.Context) error {
	s.started = false
	s.logger.InfoContext(ctx, "Demo service stopped")
	return nil
}

func (s *DemoService) Health(ctx context.Context) error {
	if !s.started {
		return fmt.Errorf("service not started")
	}
	return nil
}

func (s *DemoService) Ready(ctx context.Context) error {
	return s.Health(ctx)
}
