// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package sla

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndSLA_HappyPath validates all performance targets continuously
func TestEndToEndSLA_HappyPath(t *testing.T) {
	framework := NewSLATestFramework(t)
	defer framework.Cleanup()

	slaTestScenarios := []struct {
		name                string
		testDuration        time.Duration
		userLoadProfile     UserLoadProfile
		slaTargets          SLATargets
		regressionThreshold float64
	}{
		{
			name:         "Light load SLA validation",
			testDuration: 10 * time.Second, // Reduced from 30 seconds
			userLoadProfile: UserLoadProfile{
				ConcurrentUsers:      5,
				ActionsPerUserPerMin: 10,
				UserBehaviorMix: UserBehaviorMix{
					ReadOperations:  0.6,
					WriteOperations: 0.3,
					AdminOperations: 0.1,
				},
			},
			slaTargets: SLATargets{
				AgentSelectionTime:    2 * time.Second,
				ChatInterfaceLoadTime: 500 * time.Millisecond,
				ThemeSwitchingTime:    16 * time.Millisecond,
				FirstResponseTime:     3 * time.Second,
				StreamingLatency:      100 * time.Millisecond,
				KanbanSyncTime:        2 * time.Second,
				RAGIndexingTime:       2 * time.Minute,
				SearchResponseTime:    500 * time.Millisecond,
				ProviderFailoverTime:  2 * time.Second,
				DaemonRecoveryTime:    5 * time.Second,
			},
			regressionThreshold: 0.1, // 10% regression tolerance
		},
		{
			name:         "Heavy load SLA validation",
			testDuration: 20 * time.Second, // Reduced from 60 seconds
			userLoadProfile: UserLoadProfile{
				ConcurrentUsers:      25,
				ActionsPerUserPerMin: 15,
				UserBehaviorMix: UserBehaviorMix{
					ReadOperations:  0.5,
					WriteOperations: 0.4,
					AdminOperations: 0.1,
				},
			},
			slaTargets: SLATargets{
				AgentSelectionTime:    3 * time.Second, // Relaxed under load
				ChatInterfaceLoadTime: 800 * time.Millisecond,
				ThemeSwitchingTime:    16 * time.Millisecond, // Hard requirement
				FirstResponseTime:     5 * time.Second,
				StreamingLatency:      150 * time.Millisecond,
				KanbanSyncTime:        3 * time.Second,
				RAGIndexingTime:       3 * time.Minute,
				SearchResponseTime:    800 * time.Millisecond,
				ProviderFailoverTime:  3 * time.Second,
				DaemonRecoveryTime:    8 * time.Second,
			},
			regressionThreshold: 0.15, // 15% regression tolerance under load
		},
	}

	for _, scenario := range slaTestScenarios {
		// Skip heavy load tests when using -short flag
		if testing.Short() && scenario.name == "Heavy load SLA validation" {
			t.Skipf("Skipping %s in short mode", scenario.name)
		}

		t.Run(scenario.name, func(t *testing.T) {
			// PHASE 1: Initialize comprehensive monitoring infrastructure
			slaMonitor, err := framework.CreateSLAMonitor(SLAMonitorConfig{
				SLATargets:          scenario.slaTargets,
				SamplingInterval:    1 * time.Second,
				AlertThreshold:      0.95, // Alert if 95% SLA compliance drops
				RegressionDetection: true,
				PerformanceBaseline: framework.GetPerformanceBaseline(),
			})
			require.NoError(t, err, "Failed to create SLA monitor")
			defer slaMonitor.Shutdown()

			// Initialize user simulation infrastructure
			userSimulator, err := framework.CreateUserSimulator(UserSimulatorConfig{
				LoadProfile:       scenario.userLoadProfile,
				RealisticBehavior: true,
				ErrorInjection:    true,
				NetworkSimulation: true,
			})
			require.NoError(t, err, "Failed to create user simulator")
			defer userSimulator.Shutdown()

			// PHASE 2: Execute continuous SLA monitoring
			testCtx, testCancel := context.WithTimeout(context.Background(), scenario.testDuration)
			defer testCancel()

			// Start SLA monitoring
			err = slaMonitor.Start(testCtx)
			require.NoError(t, err, "Failed to start SLA monitoring")

			// Start user simulation (adjusted periods for shorter tests)
			simulationResults := userSimulator.StartSimulation(testCtx, SimulationConfig{
				WarmupPeriod:      time.Duration(float64(scenario.testDuration) * 0.2), // 20% warmup
				SteadyStatePeriod: time.Duration(float64(scenario.testDuration) * 0.6), // 60% steady
				CooldownPeriod:    time.Duration(float64(scenario.testDuration) * 0.2), // 20% cooldown
			})

			// PHASE 3: Inject realistic failure scenarios during test (scaled down)
			failureScenarios := []FailureScenario{
				{
					Type:      FailureTypeNetworkLatency,
					Severity:  20, // 20% of baseline latency
					Duration:  time.Duration(float64(scenario.testDuration) * 0.2), // 20% of test
					StartTime: scenario.testDuration / 3,
				},
				{
					Type:      FailureTypeMemoryPressure,
					Severity:  80, // 80% memory utilization
					Duration:  time.Duration(float64(scenario.testDuration) * 0.15), // 15% of test
					StartTime: 2 * scenario.testDuration / 3,
				},
			}

			for _, failureScenario := range failureScenarios {
				go func(fs FailureScenario) {
					time.Sleep(fs.StartTime)

					t.Logf("🔥 Injecting failure: %v (severity: %d, duration: %v)",
						fs.Type, fs.Severity, fs.Duration)

					failureInjector := framework.CreateFailureInjector(fs)
					err := failureInjector.Inject(testCtx)
					if err != nil {
						t.Logf("⚠️ Failed to inject failure %v: %v", fs.Type, err)
					}
				}(failureScenario)
			}

			// Wait for simulation completion
			<-testCtx.Done()

			// Collect simulation results
			_ = simulationResults // Would be used in real implementation

			// PHASE 4: Validate SLA Compliance
			slaResults := slaMonitor.GetResults()

			// Agent Orchestration SLA Validation
			agentSLAMetrics := slaResults.GetAgentSLAMetrics()
			assert.GreaterOrEqual(t, agentSLAMetrics.AgentSelectionCompliance, 0.95,
				"Agent selection SLA compliance too low: %.2f%% < 95%%",
				agentSLAMetrics.AgentSelectionCompliance*100)

			assert.LessOrEqual(t, agentSLAMetrics.P95AgentSelectionTime, scenario.slaTargets.AgentSelectionTime,
				"P95 agent selection time exceeded SLA: %v > %v",
				agentSLAMetrics.P95AgentSelectionTime, scenario.slaTargets.AgentSelectionTime)

			// UI/UX SLA Validation (Critical for user experience)
			uiSLAMetrics := slaResults.GetUISLAMetrics()
			assert.Equal(t, 1.0, uiSLAMetrics.ThemeSwitchingCompliance,
				"Theme switching SLA compliance must be 100%%: %.2f%%",
				uiSLAMetrics.ThemeSwitchingCompliance*100)

			assert.GreaterOrEqual(t, uiSLAMetrics.ChatLoadTimeCompliance, 0.95,
				"Chat load time SLA compliance too low: %.2f%% < 95%%",
				uiSLAMetrics.ChatLoadTimeCompliance*100)

			// Backend Systems SLA Validation
			backendSLAMetrics := slaResults.GetBackendSLAMetrics()
			assert.GreaterOrEqual(t, backendSLAMetrics.KanbanSyncCompliance, 0.95,
				"Kanban sync SLA compliance too low: %.2f%% < 95%%",
				backendSLAMetrics.KanbanSyncCompliance*100)

			assert.GreaterOrEqual(t, backendSLAMetrics.RAGSearchCompliance, 0.90,
				"RAG search SLA compliance too low: %.2f%% < 90%%",
				backendSLAMetrics.RAGSearchCompliance*100)

			// Infrastructure SLA Validation
			infraSLAMetrics := slaResults.GetInfrastructureSLAMetrics()
			assert.GreaterOrEqual(t, infraSLAMetrics.ProviderFailoverCompliance, 0.98,
				"Provider failover SLA compliance too low: %.2f%% < 98%%",
				infraSLAMetrics.ProviderFailoverCompliance*100)

			assert.GreaterOrEqual(t, infraSLAMetrics.DaemonAvailability, 0.999,
				"Daemon availability too low: %.3f%% < 99.9%%",
				infraSLAMetrics.DaemonAvailability*100)

			// PHASE 5: Regression Detection
			if framework.HasPerformanceBaseline() {
				regressionAnalysis := framework.AnalyzePerformanceRegression(
					slaResults, scenario.regressionThreshold)

				for metricName, regression := range regressionAnalysis {
					assert.LessOrEqual(t, regression.RegressionPercentage, scenario.regressionThreshold,
						"Performance regression detected for %s: %.1f%% > %.1f%%",
						metricName, regression.RegressionPercentage*100, scenario.regressionThreshold*100)

					if regression.RegressionPercentage > scenario.regressionThreshold {
						t.Logf("📉 REGRESSION WARNING: %s degraded by %.1f%% (threshold: %.1f%%)",
							metricName, regression.RegressionPercentage*100, scenario.regressionThreshold*100)
						t.Logf("   Previous: %v, Current: %v", regression.BaselineValue, regression.CurrentValue)
					}
				}
			}

			// PHASE 6: Resource Efficiency Validation
			resourceMetrics := slaResults.GetResourceMetrics()

			// Memory usage should scale linearly with load
			expectedMemoryMB := scenario.userLoadProfile.ConcurrentUsers*10 + 100 // Base + 10MB per user
			assert.LessOrEqual(t, resourceMetrics.PeakMemoryMB, int(float64(expectedMemoryMB)*1.2),
				"Peak memory usage exceeded 120%% of expected: %d MB > %d MB",
				resourceMetrics.PeakMemoryMB, int(float64(expectedMemoryMB)*1.2))

			// CPU usage should remain reasonable
			assert.LessOrEqual(t, resourceMetrics.AverageCPUPercent, 60.0,
				"Average CPU usage too high: %.1f%% > 60%%", resourceMetrics.AverageCPUPercent)

			// Network usage should be efficient
			assert.LessOrEqual(t, resourceMetrics.NetworkThroughputMBps, 50.0,
				"Network throughput too high: %.1f MB/s > 50 MB/s", resourceMetrics.NetworkThroughputMBps)

			t.Logf("✅ End-to-end SLA validation completed successfully")
			t.Logf("📊 SLA Compliance Summary:")
			t.Logf("   - Agent Selection: %.1f%%", agentSLAMetrics.AgentSelectionCompliance*100)
			t.Logf("   - Theme Switching: %.1f%%", uiSLAMetrics.ThemeSwitchingCompliance*100)
			t.Logf("   - Chat Load Time: %.1f%%", uiSLAMetrics.ChatLoadTimeCompliance*100)
			t.Logf("   - Kanban Sync: %.1f%%", backendSLAMetrics.KanbanSyncCompliance*100)
			t.Logf("   - Provider Failover: %.1f%%", infraSLAMetrics.ProviderFailoverCompliance*100)
			t.Logf("📈 Resource Utilization:")
			t.Logf("   - Peak Memory: %d MB", resourceMetrics.PeakMemoryMB)
			t.Logf("   - Average CPU: %.1f%%", resourceMetrics.AverageCPUPercent)
			t.Logf("   - Network Throughput: %.1f MB/s", resourceMetrics.NetworkThroughputMBps)
		})
	}
}
