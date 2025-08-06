// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package dev_tools

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeIntelligencePerformance_HappyPath validates real-time code intelligence
func TestCodeIntelligencePerformance_HappyPath(t *testing.T) {
	framework := NewDevToolsTestFramework(t)
	defer framework.Cleanup()

	intelligenceScenarios := []struct {
		name                     string
		codebaseSize             CodebaseSize
		expectedAutocompleteTime time.Duration
		expectedNavigationTime   time.Duration
		expectedDiagnosticsTime  time.Duration
		simulatedTypingSpeed     int // characters per second
		concurrentSessions       int
	}{
		{
			name:                     "Small project intelligence",
			codebaseSize:             CodebaseSizeSmall,
			expectedAutocompleteTime: 50 * time.Millisecond,
			expectedNavigationTime:   100 * time.Millisecond,
			expectedDiagnosticsTime:  200 * time.Millisecond,
			simulatedTypingSpeed:     5, // 5 chars/sec (realistic typing)
			concurrentSessions:       3,
		},
		{
			name:                     "Large project intelligence",
			codebaseSize:             CodebaseSizeLarge,
			expectedAutocompleteTime: 100 * time.Millisecond,
			expectedNavigationTime:   200 * time.Millisecond,
			expectedDiagnosticsTime:  500 * time.Millisecond,
			simulatedTypingSpeed:     8, // 8 chars/sec (fast typing)
			concurrentSessions:       8,
		},
		{
			name:                     "Enterprise project intelligence",
			codebaseSize:             CodebaseSizeEnterprise,
			expectedAutocompleteTime: 200 * time.Millisecond,
			expectedNavigationTime:   500 * time.Millisecond,
			expectedDiagnosticsTime:  1 * time.Second,
			simulatedTypingSpeed:     10, // 10 chars/sec (very fast typing)
			concurrentSessions:       15,
		},
	}

	for _, scenario := range intelligenceScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Initialize code intelligence engine
			codebase, err := framework.SetupCodebaseForSize(scenario.codebaseSize)
			require.NoError(t, err)

			engine, err := framework.CreateCodeIntelligenceEngine(CodeIntelligenceConfig{
				Codebase:             codebase,
				EnableAutocompletion: true,
				EnableNavigation:     true,
				EnableDiagnostics:    true,
				EnableRefactoring:    true,
				CacheSize:            200 * 1024 * 1024, // 200MB cache
				BackgroundIndexing:   true,
				IncrementalUpdates:   true,
			})
			require.NoError(t, err)
			defer engine.Shutdown()

			// Wait for initial indexing to complete
			err = framework.WaitForInitialIndexing(engine, 120*time.Second)
			require.NoError(t, err, "Initial indexing did not complete in time")

			// PHASE 1: Concurrent Developer Sessions
			var sessionWg sync.WaitGroup
			sessionMetrics := make([]*DeveloperSessionMetrics, scenario.concurrentSessions)

			for sessionIdx := 0; sessionIdx < scenario.concurrentSessions; sessionIdx++ {
				sessionWg.Add(1)
				go func(idx int) {
					defer sessionWg.Done()

					session := framework.CreateDeveloperSession(DeveloperSessionConfig{
						SessionID:       fmt.Sprintf("session-%d", idx),
						TypingSpeed:     scenario.simulatedTypingSpeed,
						WorkingFiles:    framework.SelectRealisticWorkingFiles(codebase, 5),
						EditingPatterns: framework.GetRealisticEditingPatterns(),
					})

					metrics := NewDeveloperSessionMetrics(idx)
					sessionMetrics[idx] = metrics

					// Simulate 10-second development session (fast test)
					sessionCtx, sessionCancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer sessionCancel()

					framework.SimulateDevelopmentSession(sessionCtx, session, DevelopmentSimulation{
						AutocompletionFrequency: 2 * time.Second, // Request autocompletion every 2 seconds
						NavigationFrequency:     5 * time.Second, // Navigate every 5 seconds
						DiagnosticsFrequency:    8 * time.Second, // Check diagnostics every 8 seconds
						CodeChanges: []CodeChangePattern{
							{Type: ChangeTypeAddFunction, Probability: 0.3},
							{Type: ChangeTypeModifyFunction, Probability: 0.4},
							{Type: ChangeTypeAddImport, Probability: 0.2},
							{Type: ChangeTypeRefactorCode, Probability: 0.1},
						},
					}, metrics)

				}(sessionIdx)
			}

			sessionWg.Wait()

			// PHASE 2: Validate Performance Requirements

			// Autocompletion Performance
			totalAutocompleteRequests := 0
			totalAutocompleteTime := time.Duration(0)
			for _, metrics := range sessionMetrics {
				summary := metrics.GetAutocompleteSummary()
				totalAutocompleteRequests += summary.RequestCount
				totalAutocompleteTime += summary.TotalTime

				assert.LessOrEqual(t, summary.P95Time, scenario.expectedAutocompleteTime,
					"P95 autocompletion time exceeded target for session: %v > %v",
					summary.P95Time, scenario.expectedAutocompleteTime)

				assert.GreaterOrEqual(t, summary.SuccessRate, 0.98,
					"Autocompletion success rate too low: %.2f%% < 98%%", summary.SuccessRate*100)
			}

			averageAutocompleteTime := totalAutocompleteTime / time.Duration(totalAutocompleteRequests)
			assert.LessOrEqual(t, averageAutocompleteTime, scenario.expectedAutocompleteTime,
				"Average autocompletion time exceeded target: %v > %v",
				averageAutocompleteTime, scenario.expectedAutocompleteTime)

			// Navigation Performance
			totalNavigationRequests := 0
			totalNavigationTime := time.Duration(0)
			for _, metrics := range sessionMetrics {
				summary := metrics.GetNavigationSummary()
				totalNavigationRequests += summary.RequestCount
				totalNavigationTime += summary.TotalTime

				assert.LessOrEqual(t, summary.P95Time, scenario.expectedNavigationTime,
					"P95 navigation time exceeded target for session: %v > %v",
					summary.P95Time, scenario.expectedNavigationTime)

				assert.GreaterOrEqual(t, summary.AccuracyRate, 0.95,
					"Navigation accuracy too low: %.2f%% < 95%%", summary.AccuracyRate*100)
			}

			averageNavigationTime := totalNavigationTime / time.Duration(totalNavigationRequests)
			assert.LessOrEqual(t, averageNavigationTime, scenario.expectedNavigationTime,
				"Average navigation time exceeded target: %v > %v",
				averageNavigationTime, scenario.expectedNavigationTime)

			// Diagnostics Performance
			totalDiagnosticsRequests := 0
			totalDiagnosticsTime := time.Duration(0)
			for _, metrics := range sessionMetrics {
				summary := metrics.GetDiagnosticsSummary()
				totalDiagnosticsRequests += summary.RequestCount
				totalDiagnosticsTime += summary.TotalTime

				assert.LessOrEqual(t, summary.P95Time, scenario.expectedDiagnosticsTime,
					"P95 diagnostics time exceeded target for session: %v > %v",
					summary.P95Time, scenario.expectedDiagnosticsTime)

				assert.LessOrEqual(t, summary.FalsePositiveRate, 0.05,
					"Diagnostics false positive rate too high: %.2f%% > 5%%", summary.FalsePositiveRate*100)
			}

			averageDiagnosticsTime := totalDiagnosticsTime / time.Duration(totalDiagnosticsRequests)
			assert.LessOrEqual(t, averageDiagnosticsTime, scenario.expectedDiagnosticsTime,
				"Average diagnostics time exceeded target: %v > %v",
				averageDiagnosticsTime, scenario.expectedDiagnosticsTime)

			// PHASE 3: Validate Resource Efficiency
			engineMetrics := engine.GetPerformanceMetrics()

			// Memory usage should be reasonable for codebase size
			expectedMemoryMB := codebase.GetExpectedMemoryUsage()
			actualMemoryMB := engineMetrics.MemoryUsageMB
			assert.LessOrEqual(t, float64(actualMemoryMB), expectedMemoryMB*1.5,
				"Memory usage exceeded 150%% of expected: %d MB > %.0f MB",
				actualMemoryMB, expectedMemoryMB*1.5)

			// CPU usage should remain reasonable during concurrent sessions
			assert.LessOrEqual(t, engineMetrics.AverageCPUPercent, 25.0,
				"Average CPU usage too high: %.1f%% > 25%%", engineMetrics.AverageCPUPercent)

			// Cache hit rate should be high for repeated operations
			assert.GreaterOrEqual(t, engineMetrics.CacheHitRate, 0.80,
				"Cache hit rate too low: %.2f%% < 80%%", engineMetrics.CacheHitRate*100)

			t.Logf("✅ Code intelligence performance test completed successfully")
			t.Logf("📊 Intelligence Performance Summary:")
			t.Logf("   - Avg Autocompletion Time: %v", averageAutocompleteTime)
			t.Logf("   - Avg Navigation Time: %v", averageNavigationTime)
			t.Logf("   - Avg Diagnostics Time: %v", averageDiagnosticsTime)
			t.Logf("   - Memory Usage: %d MB", actualMemoryMB)
			t.Logf("   - CPU Usage: %.1f%%", engineMetrics.AverageCPUPercent)
			t.Logf("   - Cache Hit Rate: %.1f%%", engineMetrics.CacheHitRate*100)
		})
	}
}
