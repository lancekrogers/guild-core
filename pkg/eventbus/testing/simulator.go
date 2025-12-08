// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testing

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Type aliases for convenience
type EventBus = events.EventBus
type Event = events.CoreEvent

// EventSimulator generates test events for load testing
type EventSimulator struct {
	eventTypes       []string
	sources          []string
	rate             int // events per second
	burstSize        int
	burstProbability float64

	// Stats
	generated atomic.Int64
	errors    atomic.Int64

	// Control
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// SimulatorConfig configures the event simulator
type SimulatorConfig struct {
	EventTypes       []string
	Sources          []string
	Rate             int
	BurstSize        int
	BurstProbability float64
}

// DefaultSimulatorConfig returns default simulator configuration
func DefaultSimulatorConfig() *SimulatorConfig {
	return &SimulatorConfig{
		EventTypes: []string{
			"task.created",
			"task.updated",
			"task.completed",
			"agent.started",
			"agent.stopped",
			"system.heartbeat",
		},
		Sources: []string{
			"agent-1",
			"agent-2",
			"agent-3",
			"orchestrator",
			"ui",
		},
		Rate:             100,
		BurstSize:        1000,
		BurstProbability: 0.1,
	}
}

// NewEventSimulator creates a new event simulator
func NewEventSimulator(config *SimulatorConfig) *EventSimulator {
	if config == nil {
		config = DefaultSimulatorConfig()
	}

	return &EventSimulator{
		eventTypes:       config.EventTypes,
		sources:          config.Sources,
		rate:             config.Rate,
		burstSize:        config.BurstSize,
		burstProbability: config.BurstProbability,
		stopCh:           make(chan struct{}),
	}
}

// Start starts generating events
func (es *EventSimulator) Start(ctx context.Context, bus EventBus) error {
	if bus == nil {
		return gerror.New(gerror.ErrCodeValidation, "event bus is required", nil)
	}

	es.wg.Add(1)
	go es.generateEvents(ctx, bus)

	return nil
}

// Stop stops the event simulator
func (es *EventSimulator) Stop() {
	close(es.stopCh)
	es.wg.Wait()
}

// GetStats returns simulator statistics
func (es *EventSimulator) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"generated": es.generated.Load(),
		"errors":    es.errors.Load(),
		"rate":      es.rate,
	}
}

// generateEvents generates events at the configured rate
func (es *EventSimulator) generateEvents(ctx context.Context, bus EventBus) {
	defer es.wg.Done()

	ticker := time.NewTicker(time.Second / time.Duration(es.rate))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-es.stopCh:
			return
		case <-ticker.C:
			// Check for burst
			if rand.Float64() < es.burstProbability {
				es.generateBurst(ctx, bus)
			} else {
				es.generateSingleEvent(ctx, bus)
			}
		}
	}
}

// generateSingleEvent generates a single event
func (es *EventSimulator) generateSingleEvent(ctx context.Context, bus EventBus) {
	event := es.createRandomEvent()

	if err := bus.Publish(ctx, event); err != nil {
		es.errors.Add(1)
	} else {
		es.generated.Add(1)
	}
}

// generateBurst generates a burst of events
func (es *EventSimulator) generateBurst(ctx context.Context, bus EventBus) {
	var wg sync.WaitGroup

	for i := 0; i < es.burstSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.generateSingleEvent(ctx, bus)
		}()
	}

	wg.Wait()
}

// createRandomEvent creates a random test event
func (es *EventSimulator) createRandomEvent() Event {
	eventType := es.eventTypes[rand.Intn(len(es.eventTypes))]
	source := es.sources[rand.Intn(len(es.sources))]

	// Create event based on type
	switch {
	case eventType == "task.created" || eventType == "task.updated" || eventType == "task.completed":
		taskData := map[string]interface{}{
			"task_id":   fmt.Sprintf("task-%d", rand.Int63()),
			"task_name": fmt.Sprintf("Test Task %d", rand.Intn(1000)),
			"status":    getRandomTaskStatus(),
		}
		return events.NewTaskEvent(eventType, taskData["task_id"].(string), taskData)

	case eventType == "agent.started" || eventType == "agent.stopped":
		agentID := fmt.Sprintf("agent-%d", rand.Intn(10))
		agentData := map[string]interface{}{
			"agent_name": fmt.Sprintf("Test Agent %d", rand.Intn(10)),
		}
		return events.NewAgentEvent(eventType, agentID, agentData)

	default:
		systemData := map[string]interface{}{
			"message": "Test event",
		}
		return events.NewSystemEvent(eventType, source, "info", systemData)
	}
}

// getRandomTaskStatus returns a random task status
func getRandomTaskStatus() string {
	statuses := []string{"pending", "in_progress", "completed", "failed"}
	return statuses[rand.Intn(len(statuses))]
}

// LoadTester performs load testing on the event bus
type LoadTester struct {
	duration           time.Duration
	goroutines         int
	eventsPerGoroutine int

	// Results
	totalEvents   atomic.Int64
	totalErrors   atomic.Int64
	totalDuration atomic.Int64 // nanoseconds
}

// LoadTestConfig configures load testing
type LoadTestConfig struct {
	Duration           time.Duration
	Goroutines         int
	EventsPerGoroutine int
}

// NewLoadTester creates a new load tester
func NewLoadTester(config *LoadTestConfig) *LoadTester {
	return &LoadTester{
		duration:           config.Duration,
		goroutines:         config.Goroutines,
		eventsPerGoroutine: config.EventsPerGoroutine,
	}
}

// Run runs the load test
func (lt *LoadTester) Run(ctx context.Context, bus EventBus) (*LoadTestResult, error) {
	if bus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus is required", nil)
	}

	ctx, cancel := context.WithTimeout(ctx, lt.duration)
	defer cancel()

	startTime := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < lt.goroutines; i++ {
		wg.Add(1)
		go lt.runWorker(ctx, bus, &wg)
	}

	wg.Wait()

	elapsed := time.Since(startTime)

	return &LoadTestResult{
		TotalEvents:     lt.totalEvents.Load(),
		TotalErrors:     lt.totalErrors.Load(),
		Duration:        elapsed,
		EventsPerSecond: float64(lt.totalEvents.Load()) / elapsed.Seconds(),
		AverageLatency:  time.Duration(lt.totalDuration.Load() / lt.totalEvents.Load()),
		ErrorRate:       float64(lt.totalErrors.Load()) / float64(lt.totalEvents.Load()),
	}, nil
}

// runWorker runs a single load test worker
func (lt *LoadTester) runWorker(ctx context.Context, bus EventBus, wg *sync.WaitGroup) {
	defer wg.Done()

	simulator := NewEventSimulator(nil)

	for i := 0; i < lt.eventsPerGoroutine; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			event := simulator.createRandomEvent()

			start := time.Now()
			err := bus.Publish(ctx, event)
			duration := time.Since(start)

			lt.totalDuration.Add(int64(duration))

			if err != nil {
				lt.totalErrors.Add(1)
			} else {
				lt.totalEvents.Add(1)
			}
		}
	}
}

// LoadTestResult contains load test results
type LoadTestResult struct {
	TotalEvents     int64
	TotalErrors     int64
	Duration        time.Duration
	EventsPerSecond float64
	AverageLatency  time.Duration
	ErrorRate       float64
}

// String returns a string representation of the results
func (r *LoadTestResult) String() string {
	return fmt.Sprintf(
		"Load Test Results:\n"+
			"  Total Events: %d\n"+
			"  Total Errors: %d (%.2f%%)\n"+
			"  Duration: %s\n"+
			"  Events/Second: %.2f\n"+
			"  Average Latency: %s",
		r.TotalEvents,
		r.TotalErrors, r.ErrorRate*100,
		r.Duration,
		r.EventsPerSecond,
		r.AverageLatency,
	)
}
