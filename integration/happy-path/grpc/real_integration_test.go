// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealDaemonIntegration tests the real gRPC daemon integration
func TestRealDaemonIntegration(t *testing.T) {
	// Create test framework with real daemon enabled
	framework := NewGRPCTestFramework(t).WithRealDaemon()
	defer framework.Cleanup()

	// Test daemon startup and basic functionality
	t.Run("RealDaemonStartup", func(t *testing.T) {
		config := DaemonConfig{
			Port:                framework.GetAvailablePort(),
			HealthCheckInterval: 1 * time.Second,
			RestartPolicy:       RestartPolicy_Always,
			MaxRestartAttempts:  3,
			ResourceLimits: ResourceLimits{
				MaxMemoryMB:   500,
				MaxCPUPercent: 50,
				MaxGoroutines: 1000,
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold: 5,
				RecoveryTimeout:  30 * time.Second,
				HalfOpenRequests: 3,
			},
		}

		// Start real daemon
		daemon, err := framework.StartDaemon(config)
		require.NoError(t, err, "Failed to start real daemon")
		require.NotNil(t, daemon, "Daemon should not be nil")

		// Verify daemon is healthy and responding
		assert.True(t, daemon.IsHealthy(), "Daemon should be healthy")
		assert.NotEmpty(t, daemon.Address(), "Daemon should have an address")

		// Check resource usage
		usage := daemon.GetResourceUsage()
		assert.Greater(t, usage.MemoryMB, 0.0, "Memory usage should be positive")
		assert.Greater(t, usage.CPUPercent, 0.0, "CPU usage should be positive")
		assert.Greater(t, usage.Goroutines, 0, "Goroutine count should be positive")

		// Wait for daemon health confirmation
		err = framework.WaitForDaemonHealth(daemon, 5*time.Second)
		assert.NoError(t, err, "Daemon should become healthy within timeout")

		t.Logf("✅ Real daemon started successfully at %s", daemon.Address())
		t.Logf("📊 Resource usage: Memory=%.1fMB, CPU=%.1f%%, Goroutines=%d",
			usage.MemoryMB, usage.CPUPercent, usage.Goroutines)
	})

	// Test real daemon failure injection and recovery
	t.Run("RealDaemonFailureRecovery", func(t *testing.T) {
		config := DaemonConfig{
			Port:                framework.GetAvailablePort(),
			HealthCheckInterval: 500 * time.Millisecond,
			RestartPolicy:       RestartPolicy_Always,
			MaxRestartAttempts:  3,
			ResourceLimits: ResourceLimits{
				MaxMemoryMB:   500,
				MaxCPUPercent: 50,
				MaxGoroutines: 1000,
			},
		}

		// Start real daemon
		daemon, err := framework.StartDaemon(config)
		require.NoError(t, err, "Failed to start real daemon")

		// Wait for initial health
		err = framework.WaitForDaemonHealth(daemon, 5*time.Second)
		require.NoError(t, err, "Daemon should be healthy initially")

		// Test failure injection with real daemon
		failureInjector := &FailureInjector{
			failureType: FailureType_ProcessCrash,
		}

		// Inject failure
		err = failureInjector.Inject(daemon)
		assert.NoError(t, err, "Failure injection should succeed")

		// Monitor recovery
		recoveryConfig := RecoveryConfig{
			MaxRecoveryTime:      10 * time.Second,
			HealthCheckInterval:  100 * time.Millisecond,
			ExpectedAvailability: 0.8, // 80% availability during recovery
		}

		metrics, err := framework.MonitorRecovery(daemon, recoveryConfig)
		assert.NoError(t, err, "Recovery monitoring should succeed")
		assert.NotNil(t, metrics, "Recovery metrics should be available")

		// Validate recovery metrics
		assert.LessOrEqual(t, metrics.TotalRecoveryTime, recoveryConfig.MaxRecoveryTime,
			"Recovery time should be within target")
		assert.GreaterOrEqual(t, metrics.AvailabilityDuringRecovery, recoveryConfig.ExpectedAvailability,
			"Availability during recovery should meet target")

		t.Logf("✅ Real daemon recovery completed in %v", metrics.TotalRecoveryTime)
		t.Logf("📈 Recovery metrics: Availability=%.1f%%, Failovers=%d",
			metrics.AvailabilityDuringRecovery*100, metrics.FailoverEvents)
	})

	// Test concurrent client connections to real daemon
	t.Run("RealDaemonConcurrentClients", func(t *testing.T) {
		config := DaemonConfig{
			Port:                framework.GetAvailablePort(),
			HealthCheckInterval: 1 * time.Second,
			RestartPolicy:       RestartPolicy_Always,
			MaxRestartAttempts:  3,
			ResourceLimits: ResourceLimits{
				MaxMemoryMB:   500,
				MaxCPUPercent: 50,
				MaxGoroutines: 1000,
			},
		}

		// Start real daemon
		daemon, err := framework.StartDaemon(config)
		require.NoError(t, err, "Failed to start real daemon")

		// Wait for daemon to be healthy
		err = framework.WaitForDaemonHealth(daemon, 5*time.Second)
		require.NoError(t, err, "Daemon should be healthy")

		// Simulate multiple concurrent clients
		clientCount := 5
		clientMetrics := make([]*ClientMetrics, clientCount)

		// Start concurrent clients
		for i := 0; i < clientCount; i++ {
			clientMetrics[i] = NewClientMetrics(i)

			// Simulate client operations
			go func(clientID int) {
				metrics := clientMetrics[clientID]

				// Simulate client requests
				for j := 0; j < 10; j++ {
					metrics.RequestsSent++

					// Check if daemon is still healthy (simulates gRPC call)
					if daemon.IsHealthy() {
						metrics.Responses++
					} else {
						metrics.Errors++
					}

					time.Sleep(50 * time.Millisecond)
				}
			}(i)
		}

		// Wait for clients to complete
		time.Sleep(2 * time.Second)

		// Validate client metrics
		totalRequests := 0
		totalResponses := 0
		totalErrors := 0

		for i, metrics := range clientMetrics {
			summary := metrics.GetSummary()
			totalRequests += metrics.RequestsSent
			totalResponses += metrics.Responses
			totalErrors += metrics.Errors

			t.Logf("Client %d: Requests=%d, Responses=%d, Success Rate=%.1f%%",
				i, metrics.RequestsSent, metrics.Responses, summary.SuccessRate*100)

			assert.Greater(t, summary.SuccessRate, 0.8,
				"Client %d should have >80%% success rate", i)
		}

		overallSuccessRate := float64(totalResponses) / float64(totalRequests)
		assert.Greater(t, overallSuccessRate, 0.9,
			"Overall success rate should be >90%%")

		t.Logf("✅ Concurrent client test completed")
		t.Logf("📊 Overall: Requests=%d, Responses=%d, Success Rate=%.1f%%",
			totalRequests, totalResponses, overallSuccessRate*100)

		// Check final daemon resource usage under load
		finalUsage := daemon.GetResourceUsage()
		t.Logf("📈 Final resource usage: Memory=%.1fMB, CPU=%.1f%%, Goroutines=%d",
			finalUsage.MemoryMB, finalUsage.CPUPercent, finalUsage.Goroutines)

		assert.LessOrEqual(t, finalUsage.MemoryMB, float64(config.ResourceLimits.MaxMemoryMB),
			"Memory usage should stay within limits")
	})
}

// TestRealVsMockDaemonComparison compares real and mock daemon behavior
func TestRealVsMockDaemonComparison(t *testing.T) {
	config := DaemonConfig{
		Port:                8950, // Will be overridden
		HealthCheckInterval: 1 * time.Second,
		RestartPolicy:       RestartPolicy_Always,
		MaxRestartAttempts:  3,
		ResourceLimits: ResourceLimits{
			MaxMemoryMB:   500,
			MaxCPUPercent: 50,
			MaxGoroutines: 1000,
		},
	}

	// Test with mock daemon
	t.Run("MockDaemonBaseline", func(t *testing.T) {
		framework := NewGRPCTestFramework(t) // Default to mock
		defer framework.Cleanup()

		config.Port = framework.GetAvailablePort()
		daemon, err := framework.StartDaemon(config)
		require.NoError(t, err, "Mock daemon should start")

		assert.True(t, daemon.IsHealthy(), "Mock daemon should be healthy")
		usage := daemon.GetResourceUsage()

		t.Logf("🎭 Mock daemon - Memory: %.1fMB, CPU: %.1f%%, Goroutines: %d",
			usage.MemoryMB, usage.CPUPercent, usage.Goroutines)
	})

	// Test with real daemon
	t.Run("RealDaemonComparison", func(t *testing.T) {
		framework := NewGRPCTestFramework(t).WithRealDaemon()
		defer framework.Cleanup()

		config.Port = framework.GetAvailablePort()
		daemon, err := framework.StartDaemon(config)
		require.NoError(t, err, "Real daemon should start")

		assert.True(t, daemon.IsHealthy(), "Real daemon should be healthy")
		usage := daemon.GetResourceUsage()

		t.Logf("✅ Real daemon - Memory: %.1fMB, CPU: %.1f%%, Goroutines: %d",
			usage.MemoryMB, usage.CPUPercent, usage.Goroutines)

		// Real daemon should have more realistic resource usage
		assert.Greater(t, usage.MemoryMB, 10.0, "Real daemon should use more memory than mock")
		assert.Greater(t, usage.Goroutines, 10, "Real daemon should have more goroutines")
	})
}
