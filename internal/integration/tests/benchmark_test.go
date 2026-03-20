package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/internal/integration/bridges"
	"github.com/lancekrogers/guild-core/internal/integration/services"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// BenchmarkServiceRegistry benchmarks service lifecycle operations
func BenchmarkServiceRegistry(b *testing.B) {
	ctx := context.Background()

	b.Run("Register", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry := services.NewServiceRegistry(ctx)
			service := &MockService{
				name:    fmt.Sprintf("service-%d", i),
				healthy: true,
				ready:   true,
			}
			registry.Register(service)
		}
	})

	b.Run("Start/Stop", func(b *testing.B) {
		registry := services.NewServiceRegistry(ctx)
		for i := 0; i < 10; i++ {
			service := &MockService{
				name:    fmt.Sprintf("service-%d", i),
				healthy: true,
				ready:   true,
			}
			registry.Register(service)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry.Start(ctx)
			registry.Stop(ctx)
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		registry := services.NewServiceRegistry(ctx)
		for i := 0; i < 10; i++ {
			service := &MockService{
				name:    fmt.Sprintf("service-%d", i),
				healthy: true,
				ready:   true,
			}
			registry.Register(service)
		}
		registry.Start(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry.Health(ctx)
		}
	})

	b.Run("DependencyResolution", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			registry := services.NewServiceRegistry(ctx)

			// Create chain of dependencies
			for j := 0; j < 20; j++ {
				service := &MockService{
					name:    fmt.Sprintf("service-%d", j),
					healthy: true,
					ready:   true,
				}
				registry.Register(service)
				if j > 0 {
					registry.SetDependency(fmt.Sprintf("service-%d", j), fmt.Sprintf("service-%d", j-1))
				}
			}

			b.StartTimer()
			registry.Start(ctx)
			registry.Stop(ctx)
		}
	})
}

// BenchmarkEventBridge benchmarks event bridge performance
func BenchmarkEventBridge(b *testing.B) {
	ctx := context.Background()
	eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())
	logger := observability.NewLogger(&observability.Config{
		Level:  observability.LevelError, // Reduce logging overhead
		Format: "json",
	})

	b.Run("EventLogger/SingleEvent", func(b *testing.B) {
		config := bridges.DefaultEventLoggerConfig()
		config.BufferSize = 10000
		bridge := bridges.NewEventLoggerBridge(eventBus, logger, config)
		bridge.Start(ctx)
		defer bridge.Stop(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := events.NewBaseEvent(
				fmt.Sprintf("event-%d", i),
				"benchmark.event",
				"benchmark",
				map[string]interface{}{"index": i},
			)
			eventBus.Publish(ctx, event)
		}
	})

	b.Run("EventLogger/BatchedEvents", func(b *testing.B) {
		config := bridges.DefaultEventLoggerConfig()
		config.BufferSize = 10000
		config.MaxBatchSize = 100
		bridge := bridges.NewEventLoggerBridge(eventBus, logger, config)
		bridge.Start(ctx)
		defer bridge.Stop(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := events.NewBaseEvent(
				fmt.Sprintf("event-%d", i),
				"benchmark.event",
				"benchmark",
				map[string]interface{}{"index": i},
			)
			eventBus.Publish(ctx, event)
		}
	})

	b.Run("UIBridge/EventConversion", func(b *testing.B) {
		config := bridges.DefaultUIEventConfig()
		bridge := bridges.NewUIEventBridge(eventBus, logger, config)
		bridge.Start(ctx)
		defer bridge.Stop(ctx)

		// Subscribe to events
		eventCh := bridge.EventChannel()
		go func() {
			for range eventCh {
				// Drain events
			}
		}()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bridge.PublishUIEvent(ctx, "ui.benchmark", map[string]interface{}{
				"index": i,
			})
		}
	})
}

// BenchmarkIntegrationOverhead measures the overhead of the integration layer
func BenchmarkIntegrationOverhead(b *testing.B) {
	ctx := context.Background()

	b.Run("DirectEventPublish", func(b *testing.B) {
		eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := events.NewBaseEvent(
				fmt.Sprintf("event-%d", i),
				"benchmark.direct",
				"benchmark",
				nil,
			)
			eventBus.Publish(ctx, event)
		}
	})

	b.Run("IntegratedEventPublish", func(b *testing.B) {
		eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())
		logger := observability.NewLogger(&observability.Config{
			Level:  observability.LevelError,
			Format: "json",
		})

		// Set up bridges
		eventLogger := bridges.NewEventLoggerBridge(eventBus, logger, bridges.DefaultEventLoggerConfig())
		uiBridge := bridges.NewUIEventBridge(eventBus, logger, bridges.DefaultUIEventConfig())

		registry := services.NewServiceRegistry(ctx)
		registry.Register(eventLogger)
		registry.Register(uiBridge)
		registry.Start(ctx)
		defer registry.Stop(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := events.NewBaseEvent(
				fmt.Sprintf("event-%d", i),
				"benchmark.integrated",
				"benchmark",
				nil,
			)
			eventBus.Publish(ctx, event)
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent integration operations
func BenchmarkConcurrentOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("ConcurrentServiceStarts", func(b *testing.B) {
		registry := services.NewServiceRegistry(ctx)

		// Register 50 services
		for i := 0; i < 50; i++ {
			service := &MockService{
				name:    fmt.Sprintf("service-%d", i),
				healthy: true,
				ready:   true,
			}
			registry.Register(service)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(4)

			// Concurrent operations
			go func() {
				defer wg.Done()
				registry.Start(ctx)
			}()

			go func() {
				defer wg.Done()
				registry.Health(ctx)
			}()

			go func() {
				defer wg.Done()
				registry.List()
			}()

			go func() {
				defer wg.Done()
				registry.Stop(ctx)
			}()

			wg.Wait()
		}
	})

	b.Run("ConcurrentEventFlow", func(b *testing.B) {
		eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())
		logger := observability.NewLogger(&observability.Config{
			Level:  observability.LevelError,
			Format: "json",
		})

		bridge := bridges.NewEventLoggerBridge(eventBus, logger, bridges.DefaultEventLoggerConfig())
		bridge.Start(ctx)
		defer bridge.Stop(ctx)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				event := events.NewBaseEvent(
					fmt.Sprintf("event-%d", i),
					"benchmark.concurrent",
					"benchmark",
					map[string]interface{}{"goroutine": i},
				)
				eventBus.Publish(ctx, event)
				i++
			}
		})
	})
}

// BenchmarkMemoryUsage measures memory allocation
func BenchmarkMemoryUsage(b *testing.B) {
	ctx := context.Background()

	b.Run("ServiceRegistryMemory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			registry := services.NewServiceRegistry(ctx)
			for j := 0; j < 100; j++ {
				service := &MockService{
					name:    fmt.Sprintf("service-%d", j),
					healthy: true,
					ready:   true,
				}
				registry.Register(service)
			}
			registry.Start(ctx)
			registry.Stop(ctx)
		}
	})

	b.Run("EventBridgeMemory", func(b *testing.B) {
		b.ReportAllocs()
		eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())
		logger := observability.NewLogger(&observability.Config{
			Level:  observability.LevelError,
			Format: "json",
		})

		for i := 0; i < b.N; i++ {
			config := bridges.DefaultEventLoggerConfig()
			bridge := bridges.NewEventLoggerBridge(eventBus, logger, config)
			bridge.Start(ctx)

			// Publish some events
			for j := 0; j < 10; j++ {
				event := events.NewBaseEvent(
					fmt.Sprintf("event-%d-%d", i, j),
					"memory.test",
					"benchmark",
					map[string]interface{}{"data": "test"},
				)
				eventBus.Publish(ctx, event)
			}

			bridge.Stop(ctx)
		}
	})
}

// Performance baseline constants (adjust based on actual measurements)
const (
	// Service operations
	MaxServiceStartTime = 100 * time.Millisecond
	MaxServiceStopTime  = 50 * time.Millisecond
	MaxHealthCheckTime  = 10 * time.Millisecond

	// Event operations
	MaxEventPublishTime  = 1 * time.Millisecond
	MaxEventDeliveryTime = 10 * time.Millisecond
	MinEventThroughput   = 10000 // events/second

	// Memory limits
	MaxServiceMemoryMB = 10
	MaxEventMemoryMB   = 50
)

// TestPerformanceBaselines ensures performance meets requirements
func TestPerformanceBaselines(t *testing.T) {
	ctx := context.Background()

	t.Run("ServiceStartupTime", func(t *testing.T) {
		registry := services.NewServiceRegistry(ctx)
		for i := 0; i < 10; i++ {
			service := &MockService{
				name:    fmt.Sprintf("service-%d", i),
				healthy: true,
				ready:   true,
			}
			registry.Register(service)
		}

		start := time.Now()
		err := registry.Start(ctx)
		duration := time.Since(start)

		AssertNoError(t, err, "Service startup failed")
		if duration > MaxServiceStartTime {
			t.Errorf("Service startup too slow: %v > %v", duration, MaxServiceStartTime)
		}
	})

	t.Run("EventThroughput", func(t *testing.T) {
		eventBus := events.NewMemoryEventBus(events.DefaultEventBusConfig())

		count := 10000
		start := time.Now()

		for i := 0; i < count; i++ {
			event := events.NewBaseEvent(
				fmt.Sprintf("event-%d", i),
				"throughput.test",
				"test",
				nil,
			)
			eventBus.Publish(ctx, event)
		}

		duration := time.Since(start)
		throughput := float64(count) / duration.Seconds()

		if throughput < MinEventThroughput {
			t.Errorf("Event throughput too low: %.0f/s < %d/s", throughput, MinEventThroughput)
		}
	})
}
