// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"sync"
	"time"
)

// FailureType represents different types of failures that can be injected
type FailureType int

const (
	FailureType_ProcessCrash FailureType = iota
	FailureType_NetworkPartition
	FailureType_ResourceExhaustion
)

func (f FailureType) String() string {
	switch f {
	case FailureType_ProcessCrash:
		return "ProcessCrash"
	case FailureType_NetworkPartition:
		return "NetworkPartition"
	case FailureType_ResourceExhaustion:
		return "ResourceExhaustion"
	default:
		return "Unknown"
	}
}

// RestartPolicy defines how daemon should restart
type RestartPolicy int

const (
	RestartPolicy_Always RestartPolicy = iota
	RestartPolicy_OnFailure
	RestartPolicy_Never
)

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	HalfOpenRequests int
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxMemoryMB   int
	MaxCPUPercent float64
	MaxGoroutines int
}

// DaemonConfig contains configuration for daemon lifecycle testing
type DaemonConfig struct {
	Port                int
	HealthCheckInterval time.Duration
	RestartPolicy       RestartPolicy
	MaxRestartAttempts  int
	ResourceLimits      ResourceLimits
	CircuitBreaker      CircuitBreakerConfig
}

// ResourceUsage tracks resource consumption
type ResourceUsage struct {
	MemoryMB   float64 // Use float64 for more accurate memory measurements
	CPUPercent float64
	Goroutines int
}

// RecoveryConfig defines recovery monitoring parameters
type RecoveryConfig struct {
	MaxRecoveryTime      time.Duration
	HealthCheckInterval  time.Duration
	ExpectedAvailability float64
}

// RecoveryMetrics contains recovery measurement results
type RecoveryMetrics struct {
	TotalRecoveryTime          time.Duration
	AvailabilityDuringRecovery float64
	FailoverEvents             int
}

// TestEventBus provides a test implementation of the EventBus interface
type TestEventBus struct {
	subscribers map[string][]func(event interface{})
	mu          sync.RWMutex
}

// NewTestEventBus creates a new test event bus
func NewTestEventBus() *TestEventBus {
	return &TestEventBus{
		subscribers: make(map[string][]func(event interface{})),
	}
}

// Publish publishes an event to all subscribers
func (bus *TestEventBus) Publish(event interface{}) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// For testing purposes, we can determine event type from the interface
	eventType := "test_event"
	if handlers, exists := bus.subscribers[eventType]; exists {
		for _, handler := range handlers {
			go handler(event) // Async delivery
		}
	}
}

// Subscribe subscribes to events of a specific type
func (bus *TestEventBus) Subscribe(eventType string, handler func(event interface{})) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.subscribers[eventType] == nil {
		bus.subscribers[eventType] = make([]func(event interface{}), 0)
	}
	bus.subscribers[eventType] = append(bus.subscribers[eventType], handler)
}
