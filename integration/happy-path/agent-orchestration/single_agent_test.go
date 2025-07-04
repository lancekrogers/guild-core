// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent_orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSingleAgentExecution_HappyPath validates core agent execution workflow
// This test ensures that single agent execution meets all performance SLAs
// and provides excellent user experience quality.
func TestSingleAgentExecution_HappyPath(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	testCases := []struct {
		name          string
		taskInput     string
		expectedTime  time.Duration
		qualityTarget int
		slaTarget     time.Duration
		capabilities  []string
		complexity    ComplexityLevel
	}{
		{
			name:          "Simple code analysis",
			taskInput:     "Analyze the main.go file and identify potential improvements",
			expectedTime:  10 * time.Second,
			qualityTarget: 85,
			slaTarget:     2 * time.Second, // Agent selection SLA
			capabilities:  []string{"code_analysis"},
			complexity:    ComplexityMedium,
		},
		{
			name:          "Documentation generation",
			taskInput:     "Generate documentation for the registry package",
			expectedTime:  15 * time.Second,
			qualityTarget: 90,
			slaTarget:     2 * time.Second,
			capabilities:  []string{"documentation"},
			complexity:    ComplexityMedium,
		},
		{
			name:          "Complex refactoring analysis",
			taskInput:     "Suggest refactoring opportunities for the entire codebase",
			expectedTime:  30 * time.Second,
			qualityTarget: 80,
			slaTarget:     2 * time.Second,
			capabilities:  []string{"code_analysis", "refactoring"},
			complexity:    ComplexityHigh,
		},
		{
			name:          "Low complexity task",
			taskInput:     "Check code formatting in utils.go",
			expectedTime:  5 * time.Second,
			qualityTarget: 85,
			slaTarget:     2 * time.Second,
			capabilities:  []string{"code_analysis"},
			complexity:    ComplexityLow,
		},
		{
			name:          "Critical performance task",
			taskInput:     "Optimize database queries for production performance",
			expectedTime:  45 * time.Second,
			qualityTarget: 95,
			slaTarget:     2 * time.Second,
			capabilities:  []string{"code_analysis", "refactoring"},
			complexity:    ComplexityCritical,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Start performance monitoring
			perfMonitor := framework.performanceLog.StartOperation("single_agent_execution")
			defer perfMonitor.End()

			// PHASE 1: Agent Selection (Critical SLA: ≤2 seconds)
			t.Log("Phase 1: Agent Selection")
			selectionStart := time.Now()

			agent, err := framework.GetOptimalAgent(TaskRequirements{
				Type:         "coding",
				Complexity:   tc.complexity,
				MaxCost:      5, // Fibonacci scale
				Capabilities: tc.capabilities,
			})
			selectionDuration := time.Since(selectionStart)

			// Validate agent selection SLA
			require.NoError(t, err, "Agent selection must succeed")
			require.NotNil(t, agent, "Selected agent must not be nil")
			assert.LessOrEqual(t, selectionDuration, tc.slaTarget,
				"Agent selection exceeded SLA: %v > %v", selectionDuration, tc.slaTarget)

			perfMonitor.RecordMetric("agent_selection_time", selectionDuration)

			t.Logf("✓ Agent selected: %s (time: %v)", agent.GetName(), selectionDuration)

			// PHASE 2: Task Execution with Streaming
			t.Log("Phase 2: Task Execution")
			executionStart := time.Now()

			// Simulate real user interaction with context
			userContext := framework.userSimulator.CreateRealisticContext(map[string]interface{}{
				"projectType":    "go-service",
				"codebaseSize":   "medium",
				"userExperience": "senior_developer",
			})

			// Track streaming metrics
			streamingChunks := 0
			streamingLatencies := []time.Duration{}
			chunkStart := executionStart

			result, err := agent.Execute(framework.testContext, ExecutionInput{
				Message: tc.taskInput,
				Context: userContext,
				StreamingCallback: func(chunk string) {
					// Validate streaming response time (should be immediate)
					chunkReceived := time.Now()
					streamLatency := chunkReceived.Sub(chunkStart)
					streamingLatencies = append(streamingLatencies, streamLatency)
					streamingChunks++
					chunkStart = chunkReceived

					assert.LessOrEqual(t, streamLatency, 200*time.Millisecond,
						"Streaming latency too high: %v", streamLatency)

					// Record streaming metrics
					perfMonitor.RecordMetric("streaming_latency", streamLatency)
				},
				Requirements: TaskRequirements{
					Type:         "coding",
					Complexity:   tc.complexity,
					Capabilities: tc.capabilities,
				},
			})
			executionDuration := time.Since(executionStart)

			// Validate execution success
			require.NoError(t, err, "Agent execution must succeed for task: %s", tc.taskInput)
			require.NotNil(t, result, "Execution result must not be nil")

			// Validate execution time within expectations
			assert.LessOrEqual(t, executionDuration, tc.expectedTime,
				"Execution time exceeded expectation: %v > %v", executionDuration, tc.expectedTime)

			t.Logf("✓ Task executed successfully (time: %v)", executionDuration)

			// PHASE 3: Quality Validation
			t.Log("Phase 3: Quality Validation")
			qualityScore := framework.validateResponseQuality(result.Content, tc.taskInput)
			assert.GreaterOrEqual(t, qualityScore, tc.qualityTarget,
				"Response quality below target: %d < %d", qualityScore, tc.qualityTarget)

			t.Logf("✓ Quality score: %d (target: %d)", qualityScore, tc.qualityTarget)

			// PHASE 4: Resource Validation
			t.Log("Phase 4: Resource Validation")
			memUsage := framework.measureMemoryUsage()
			assert.LessOrEqual(t, memUsage, uint64(50*1024*1024), // 50MB limit
				"Memory usage too high: %d bytes", memUsage)

			t.Logf("✓ Memory usage: %.2f MB", float64(memUsage)/(1024*1024))

			// PHASE 5: Context Persistence Validation
			t.Log("Phase 5: Context Validation")
			contextAfter := agent.GetContext()
			assert.NotNil(t, contextAfter, "Agent context must be preserved")
			assert.Equal(t, agent.GetID(), contextAfter.AgentID, "Agent ID must be preserved in context")

			// PHASE 6: Streaming Performance Validation
			t.Log("Phase 6: Streaming Performance")
			if streamingChunks > 0 {
				assert.Greater(t, streamingChunks, 1, "Response should be streamed in multiple chunks")

				// Calculate average streaming latency
				var totalLatency time.Duration
				for _, latency := range streamingLatencies {
					totalLatency += latency
				}
				averageLatency := totalLatency / time.Duration(len(streamingLatencies))

				assert.LessOrEqual(t, averageLatency, 150*time.Millisecond,
					"Average streaming latency too high: %v", averageLatency)

				t.Logf("✓ Streaming: %d chunks, avg latency: %v", streamingChunks, averageLatency)
			}

			// PHASE 7: Cost Validation
			t.Log("Phase 7: Cost Validation")
			assert.Greater(t, result.TokensUsed, 0, "Token usage must be tracked")
			assert.Greater(t, result.CostIncurred, 0.0, "Cost must be tracked")

			// Validate cost is reasonable (not more than $0.50 for test cases)
			assert.LessOrEqual(t, result.CostIncurred, 0.50,
				"Cost too high for test case: $%.4f", result.CostIncurred)

			t.Logf("✓ Cost: $%.4f (%d tokens)", result.CostIncurred, result.TokensUsed)

			// Record comprehensive performance metrics
			perfMonitor.RecordMetrics(map[string]interface{}{
				"execution_time":   executionDuration,
				"quality_score":    qualityScore,
				"memory_usage":     memUsage,
				"tokens_used":      result.TokensUsed,
				"cost_incurred":    result.CostIncurred,
				"streaming_chunks": streamingChunks,
				"complexity_level": tc.complexity,
				"agent_type":       agent.GetType(),
				"agent_provider":   agent.GetProvider(),
			})

			t.Logf("✅ Task completed successfully: %s (time: %v, quality: %d, cost: $%.4f)",
				tc.name, executionDuration, qualityScore, result.CostIncurred)
		})
	}
}

// TestAgentSelection_PerformanceSLA validates that agent selection consistently meets SLA
func TestAgentSelection_PerformanceSLA(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	// Test multiple selection scenarios to ensure consistent performance
	scenarios := []struct {
		name         string
		requirements TaskRequirements
		iterations   int
	}{
		{
			name: "High frequency selections",
			requirements: TaskRequirements{
				Type:         "coding",
				Complexity:   ComplexityMedium,
				MaxCost:      5,
				Capabilities: []string{"code_analysis"},
			},
			iterations: 20,
		},
		{
			name: "Complex capability matching",
			requirements: TaskRequirements{
				Type:         "coding",
				Complexity:   ComplexityHigh,
				MaxCost:      3,
				Capabilities: []string{"code_analysis", "refactoring", "documentation"},
			},
			iterations: 10,
		},
		{
			name: "Cost-constrained selection",
			requirements: TaskRequirements{
				Type:         "documentation",
				Complexity:   ComplexityLow,
				MaxCost:      2,
				Capabilities: []string{"documentation"},
			},
			iterations: 15,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var totalTime time.Duration
			var maxTime time.Duration
			successCount := 0

			for i := 0; i < scenario.iterations; i++ {
				start := time.Now()
				agent, err := framework.GetOptimalAgent(scenario.requirements)
				duration := time.Since(start)

				if err == nil && agent != nil {
					successCount++
					totalTime += duration
					if duration > maxTime {
						maxTime = duration
					}

					// Each individual selection must meet SLA
					assert.LessOrEqual(t, duration, 2*time.Second,
						"Selection %d exceeded SLA: %v", i+1, duration)
				}
			}

			// Validate overall performance
			assert.Equal(t, scenario.iterations, successCount,
				"All selections should succeed")

			if successCount > 0 {
				avgTime := totalTime / time.Duration(successCount)
				t.Logf("Performance summary for %s:", scenario.name)
				t.Logf("  - Average: %v", avgTime)
				t.Logf("  - Maximum: %v", maxTime)
				t.Logf("  - Success rate: %.1f%%", float64(successCount)/float64(scenario.iterations)*100)

				// Average should be well under SLA
				assert.LessOrEqual(t, avgTime, 1*time.Second,
					"Average selection time too high: %v", avgTime)

				// Maximum should not exceed SLA
				assert.LessOrEqual(t, maxTime, 2*time.Second,
					"Maximum selection time exceeded SLA: %v", maxTime)
			}
		})
	}
}

// TestAgentExecution_ErrorRecovery validates graceful error handling
func TestAgentExecution_ErrorRecovery(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	// Test error scenarios to ensure graceful handling
	errorScenarios := []struct {
		name         string
		requirements TaskRequirements
		expectError  bool
		errorType    string
	}{
		{
			name: "No agents available for capability",
			requirements: TaskRequirements{
				Type:         "nonexistent",
				Complexity:   ComplexityMedium,
				MaxCost:      5,
				Capabilities: []string{"quantum_computing"}, // Non-existent capability
			},
			expectError: true,
			errorType:   "no_agent_available",
		},
		{
			name: "Cost budget too low",
			requirements: TaskRequirements{
				Type:         "coding",
				Complexity:   ComplexityHigh,
				MaxCost:      1, // Too low for available agents
				Capabilities: []string{"code_analysis"},
			},
			expectError: true,
			errorType:   "budget_exceeded",
		},
		{
			name: "Valid requirements should succeed",
			requirements: TaskRequirements{
				Type:         "coding",
				Complexity:   ComplexityMedium,
				MaxCost:      5,
				Capabilities: []string{"code_analysis"},
			},
			expectError: false,
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			start := time.Now()
			agent, err := framework.GetOptimalAgent(scenario.requirements)
			duration := time.Since(start)

			if scenario.expectError {
				assert.Error(t, err, "Expected error for scenario: %s", scenario.name)
				assert.Nil(t, agent, "Agent should be nil when error occurs")

				// Even error cases should respond quickly
				assert.LessOrEqual(t, duration, 1*time.Second,
					"Error response took too long: %v", duration)
			} else {
				assert.NoError(t, err, "Unexpected error for scenario: %s", scenario.name)
				assert.NotNil(t, agent, "Agent should not be nil for valid scenario")

				// Valid selections should meet SLA
				assert.LessOrEqual(t, duration, 2*time.Second,
					"Valid selection exceeded SLA: %v", duration)
			}

			t.Logf("Scenario '%s' completed in %v", scenario.name, duration)
		})
	}
}

// TestConcurrentAgentSelection validates concurrent agent selection performance
func TestConcurrentAgentSelection(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	const numGoroutines = 10
	const selectionsPerGoroutine = 5

	results := make(chan time.Duration, numGoroutines*selectionsPerGoroutine)
	errors := make(chan error, numGoroutines*selectionsPerGoroutine)

	requirements := TaskRequirements{
		Type:         "coding",
		Complexity:   ComplexityMedium,
		MaxCost:      5,
		Capabilities: []string{"code_analysis"},
	}

	// Launch concurrent selections
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < selectionsPerGoroutine; j++ {
				start := time.Now()
				agent, err := framework.GetOptimalAgent(requirements)
				duration := time.Since(start)

				if err != nil {
					errors <- err
				} else if agent == nil {
					errors <- assert.AnError
				} else {
					results <- duration
				}
			}
		}(i)
	}

	// Collect results
	var durations []time.Duration
	var errorCount int
	totalOperations := numGoroutines * selectionsPerGoroutine

	for i := 0; i < totalOperations; i++ {
		select {
		case duration := <-results:
			durations = append(durations, duration)
		case err := <-errors:
			errorCount++
			t.Logf("Concurrent selection error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent selections")
		}
	}

	// Validate results
	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent selections")
	assert.Len(t, durations, totalOperations, "All selections should complete successfully")

	if len(durations) > 0 {
		// Calculate statistics
		var total time.Duration
		var maxDuration time.Duration
		for _, d := range durations {
			total += d
			if d > maxDuration {
				maxDuration = d
			}
		}
		avgDuration := total / time.Duration(len(durations))

		t.Logf("Concurrent selection performance:")
		t.Logf("  - Operations: %d", len(durations))
		t.Logf("  - Average: %v", avgDuration)
		t.Logf("  - Maximum: %v", maxDuration)
		t.Logf("  - Success rate: %.1f%%", float64(len(durations))/float64(totalOperations)*100)

		// All selections should meet SLA even under concurrent load
		for i, duration := range durations {
			assert.LessOrEqual(t, duration, 3*time.Second, // Slightly higher SLA for concurrent operations
				"Concurrent selection %d exceeded SLA: %v", i+1, duration)
		}

		// Average should still be reasonable
		assert.LessOrEqual(t, avgDuration, 2*time.Second,
			"Average concurrent selection time too high: %v", avgDuration)
	}
}
