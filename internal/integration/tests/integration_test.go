package tests

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/integration/bridges"
	"github.com/lancekrogers/guild/internal/integration/services"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// TestServiceLifecycle tests the service registry lifecycle management
func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()

	// Create service registry
	registry := services.NewServiceRegistry(ctx)

	// Create a mock service
	mockService := &MockService{
		name:    "test-service",
		healthy: true,
		ready:   true,
	}

	// Test registration
	err := registry.Register(mockService)
	AssertNoError(t, err, "Failed to register service")

	// Test duplicate registration
	err = registry.Register(mockService)
	AssertError(t, err, "Expected error for duplicate registration")

	// Test service retrieval
	service, err := registry.Get("test-service")
	AssertNoError(t, err, "Failed to get service")
	AssertEqual(t, "test-service", service.Name(), "Service name mismatch")

	// Test service list
	services := registry.List()
	AssertEqual(t, 1, len(services), "Expected 1 service")
	AssertEqual(t, "test-service", services[0].Name, "Service name mismatch in list")

	// Test service start
	err = registry.Start(ctx)
	AssertNoError(t, err, "Failed to start services")
	AssertEqual(t, true, mockService.started, "Service should be started")

	// Test health check
	health := registry.Health(ctx)
	AssertEqual(t, 0, len(health), "Expected no health errors")

	// Test service stop
	err = registry.Stop(ctx)
	AssertNoError(t, err, "Failed to stop services")
	AssertEqual(t, false, mockService.started, "Service should be stopped")

	// Clean up
	registry.Close()
}

// TestEventLoggerBridge tests the event logger bridge
func TestEventLoggerBridge(t *testing.T) {
	ctx := context.Background()
	harness := NewTestHarness(ctx)

	// Create event bus
	eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())

	// Create logger
	logger := observability.NewLogger(&observability.Config{
		Level:  observability.LevelDebug,
		Format: "json",
	})

	// Create bridge
	config := bridges.DefaultEventLoggerConfig()
	bridge := bridges.NewEventLoggerBridge(eventBus, logger, config)

	// Register and start bridge
	err := harness.StartService(ctx, "event-logger-bridge", func(ctx context.Context) error {
		return bridge.Start(ctx)
	})
	AssertNoError(t, err, "Failed to start event logger bridge")

	// Wait for bridge to be ready
	err = harness.WaitForService(ctx, "event-logger-bridge", func(ctx context.Context) error {
		return bridge.Ready(ctx)
	}, 5*time.Second)
	AssertNoError(t, err, "Bridge not ready")

	// Publish test events
	testEvent := events.NewBaseEvent("test-event-id", "test.event", "integration-test", map[string]interface{}{
		"message": "Hello, World!",
	})

	err = eventBus.Publish(ctx, testEvent)
	AssertNoError(t, err, "Failed to publish event")

	// Give time for event to be logged
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := bridge.GetMetrics()
	AssertEqual(t, true, metrics.Running, "Bridge should be running")
	// Note: EventsLogged might be 0 due to async processing

	// Stop bridge
	err = harness.StopService(ctx, "event-logger-bridge", func(ctx context.Context) error {
		return bridge.Stop(ctx)
	})
	AssertNoError(t, err, "Failed to stop bridge")
}

// TestServiceDependencies tests service dependency management
func TestServiceDependencies(t *testing.T) {
	ctx := context.Background()
	registry := services.NewServiceRegistry(ctx)

	// Create services
	serviceA := &MockService{name: "service-a", healthy: true, ready: true}
	serviceB := &MockService{name: "service-b", healthy: true, ready: true}
	serviceC := &MockService{name: "service-c", healthy: true, ready: true}

	// Register services
	registry.Register(serviceA)
	registry.Register(serviceB)
	registry.Register(serviceC)

	// Set dependencies: C depends on B, B depends on A
	err := registry.SetDependency("service-c", "service-b")
	AssertNoError(t, err, "Failed to set dependency C->B")

	err = registry.SetDependency("service-b", "service-a")
	AssertNoError(t, err, "Failed to set dependency B->A")

	// Test circular dependency detection
	err = registry.SetDependency("service-a", "service-c")
	AssertError(t, err, "Expected error for circular dependency")

	// Start services - should start in order A, B, C
	err = registry.Start(ctx)
	AssertNoError(t, err, "Failed to start services")

	// Verify start order
	if serviceA.startTime.After(serviceB.startTime) {
		t.Error("Service A should start before B")
	}
	if serviceB.startTime.After(serviceC.startTime) {
		t.Error("Service B should start before C")
	}

	// Stop services - should stop in reverse order C, B, A
	err = registry.Stop(ctx)
	AssertNoError(t, err, "Failed to stop services")

	// Verify stop order
	if serviceC.stopTime.After(serviceB.stopTime) {
		t.Error("Service C should stop before B")
	}
	if serviceB.stopTime.After(serviceA.stopTime) {
		t.Error("Service B should stop before A")
	}

	// Clean up
	registry.Close()
}

// TestIntegrationSuite runs a full integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite := IntegrationSuite{
		Name:        "Core Integration Tests",
		Description: "Tests core integration components",
		Tests: []IntegrationTest{
			{
				Name:        "EventBus Integration",
				Description: "Tests event bus publish/subscribe",
				Test: func(ctx context.Context, t *testing.T) error {
					eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())

					received := make(chan events.CoreEvent, 1)
					handler := func(ctx context.Context, event events.CoreEvent) error {
						received <- event
						return nil
					}

					// Subscribe to all events
					subID, err := eventBus.SubscribeAll(ctx, handler)
					if err != nil {
						return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe")
					}
					defer eventBus.Unsubscribe(ctx, subID)

					// Publish event
					event := events.NewBaseEvent("test-event-id", "test.event", "integration-test", map[string]interface{}{
						"test": true,
					})

					err = eventBus.Publish(ctx, event)
					if err != nil {
						return err
					}

					// Wait for event
					select {
					case e := <-received:
						if e.GetType() != "test.event" {
							return gerror.New(gerror.ErrCodeValidation, "wrong event type", nil).
								WithDetails("expected", "test.event").
								WithDetails("actual", e.GetType())
						}
					case <-time.After(1 * time.Second):
						return gerror.New(gerror.ErrCodeTimeout, "timeout waiting for event", nil)
					}

					return nil
				},
				Timeout: 5 * time.Second,
			},
		},
		Parallel:    false,
		StopOnError: true,
		RetryCount:  1,
		RetryDelay:  1 * time.Second,
	}

	RunSuite(t, suite)
}

// MockService is a test service implementation
type MockService struct {
	name      string
	started   bool
	healthy   bool
	ready     bool
	startTime time.Time
	stopTime  time.Time
}

func (s *MockService) Name() string {
	return s.name
}

func (s *MockService) Start(ctx context.Context) error {
	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil)
	}
	s.started = true
	s.startTime = time.Now()
	return nil
}

func (s *MockService) Stop(ctx context.Context) error {
	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil)
	}
	s.started = false
	s.stopTime = time.Now()
	return nil
}

func (s *MockService) Health(ctx context.Context) error {
	if !s.healthy {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service unhealthy", nil)
	}
	return nil
}

func (s *MockService) Ready(ctx context.Context) error {
	if !s.ready {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not ready", nil)
	}
	return nil
}
