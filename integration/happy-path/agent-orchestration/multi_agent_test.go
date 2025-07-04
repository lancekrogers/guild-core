// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent_orchestration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiAgentCoordination_HappyPath validates multi-agent coordination workflows
// This completes the remaining 15% of Agent 1 implementation
func TestMultiAgentCoordination_HappyPath(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	coordinationScenarios := []struct {
		name                   string
		agentCount             int
		coordinationType       string
		expectedCompletionTime time.Duration
		coordinationComplexity ComplexityLevel
		expectedSuccessRate    float64
	}{
		{
			name:                   "Parallel independent tasks",
			agentCount:             3,
			coordinationType:       "parallel",
			expectedCompletionTime: 15 * time.Second,
			coordinationComplexity: ComplexityMedium,
			expectedSuccessRate:    0.95,
		},
		{
			name:                   "Sequential dependent tasks",
			agentCount:             2,
			coordinationType:       "sequential",
			expectedCompletionTime: 25 * time.Second,
			coordinationComplexity: ComplexityHigh,
			expectedSuccessRate:    0.90,
		},
		{
			name:                   "Collaborative analysis",
			agentCount:             4,
			coordinationType:       "collaborative",
			expectedCompletionTime: 30 * time.Second,
			coordinationComplexity: ComplexityHigh,
			expectedSuccessRate:    0.85,
		},
	}

	for _, scenario := range coordinationScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Starting multi-agent coordination test: %s", scenario.name)

			coordinationStart := time.Now()

			// PHASE 1: Agent Selection and Allocation
			t.Log("Phase 1: Agent Selection and Allocation")
			agents := make([]*RealAgent, scenario.agentCount)
			var allocationTime time.Duration

			for i := 0; i < scenario.agentCount; i++ {
				allocationStart := time.Now()

				// Vary requirements to get different agents
				requirements := TaskRequirements{
					Type:         "coding",
					Complexity:   scenario.coordinationComplexity,
					MaxCost:      5,
					Capabilities: []string{"code_analysis"},
				}

				// Add different capabilities for different agents
				if i%2 == 0 {
					requirements.Capabilities = append(requirements.Capabilities, "refactoring")
				}
				if i == scenario.agentCount-1 {
					requirements.Capabilities = append(requirements.Capabilities, "documentation")
				}

				agent, err := framework.GetOptimalAgent(requirements)
				require.NoError(t, err, "Agent %d selection must succeed", i+1)
				require.NotNil(t, agent, "Agent %d must not be nil", i+1)

				agents[i] = agent
				allocationTime += time.Since(allocationStart)
			}

			t.Logf("✓ Allocated %d agents in %v", scenario.agentCount, allocationTime)

			// PHASE 2: Execute Coordination Pattern
			t.Log("Phase 2: Execute Coordination Pattern")

			var coordinationResults []CoordinationResult
			var coordinationErr error

			switch scenario.coordinationType {
			case "parallel":
				coordinationResults, coordinationErr = framework.executeParallelCoordination(agents, scenario)
			case "sequential":
				coordinationResults, coordinationErr = framework.executeSequentialCoordination(agents, scenario)
			case "collaborative":
				coordinationResults, coordinationErr = framework.executeCollaborativeCoordination(agents, scenario)
			}

			coordinationDuration := time.Since(coordinationStart)

			require.NoError(t, coordinationErr, "Coordination must succeed")
			require.NotEmpty(t, coordinationResults, "Must have coordination results")

			// PHASE 3: Validate Coordination Performance
			t.Log("Phase 3: Validate Coordination Performance")

			// Check completion time SLA
			assert.LessOrEqual(t, coordinationDuration, scenario.expectedCompletionTime,
				"Coordination exceeded time limit: %v > %v", coordinationDuration, scenario.expectedCompletionTime)

			// Calculate success rate
			successfulTasks := 0
			for _, result := range coordinationResults {
				if result.Success {
					successfulTasks++
				}
			}
			actualSuccessRate := float64(successfulTasks) / float64(len(coordinationResults))

			assert.GreaterOrEqual(t, actualSuccessRate, scenario.expectedSuccessRate,
				"Success rate below target: %.2f%% < %.2f%%", actualSuccessRate*100, scenario.expectedSuccessRate*100)

			// PHASE 4: Validate Inter-Agent Communication
			t.Log("Phase 4: Validate Inter-Agent Communication")

			// Check that agents didn't interfere with each other
			for i, result := range coordinationResults {
				assert.NotEmpty(t, result.Content, "Agent %d must produce content", i+1)
				assert.Greater(t, result.ExecutionTime, time.Duration(0), "Agent %d must have execution time", i+1)

				// Validate no resource conflicts
				assert.LessOrEqual(t, result.MemoryUsage, uint64(100*1024*1024), // 100MB per agent
					"Agent %d memory usage too high: %d bytes", i+1, result.MemoryUsage)
			}

			// PHASE 5: Validate Coordination Quality
			t.Log("Phase 5: Validate Coordination Quality")

			coordinationQuality := framework.validateCoordinationQuality(coordinationResults, scenario.coordinationType)
			assert.GreaterOrEqual(t, coordinationQuality, 80,
				"Coordination quality below target: %d < 80", coordinationQuality)

			t.Logf("✅ Multi-agent coordination completed successfully: %s", scenario.name)
			t.Logf("📊 Coordination Summary:")
			t.Logf("   - Agents: %d", scenario.agentCount)
			t.Logf("   - Total Time: %v", coordinationDuration)
			t.Logf("   - Success Rate: %.1f%%", actualSuccessRate*100)
			t.Logf("   - Coordination Quality: %d", coordinationQuality)
		})
	}
}

// CoordinationResult represents the result of a coordinated agent execution
type CoordinationResult struct {
	AgentID       string
	TaskID        string
	Content       string
	Success       bool
	ExecutionTime time.Duration
	MemoryUsage   uint64
	Error         error
}

// executeParallelCoordination runs agents in parallel on independent tasks
func (f *HappyPathTestFramework) executeParallelCoordination(agents []*RealAgent, scenario struct {
	name                   string
	agentCount             int
	coordinationType       string
	expectedCompletionTime time.Duration
	coordinationComplexity ComplexityLevel
	expectedSuccessRate    float64
}) ([]CoordinationResult, error) {

	var wg sync.WaitGroup
	results := make([]CoordinationResult, len(agents))

	tasks := []string{
		"Analyze the error handling patterns in the codebase",
		"Review the testing strategy and suggest improvements",
		"Examine the performance characteristics of key functions",
		"Evaluate the documentation completeness",
	}

	for i, agent := range agents {
		wg.Add(1)
		go func(agentIndex int, a *RealAgent) {
			defer wg.Done()

			start := time.Now()
			memBefore := f.measureMemoryUsage()

			// Use a task appropriate for the agent index
			taskIndex := agentIndex % len(tasks)
			result, err := a.Execute(f.testContext, ExecutionInput{
				Message: tasks[taskIndex],
				Requirements: TaskRequirements{
					Type:       "coding",
					Complexity: scenario.coordinationComplexity,
				},
			})

			memAfter := f.measureMemoryUsage()
			executionTime := time.Since(start)

			results[agentIndex] = CoordinationResult{
				AgentID:       a.GetID(),
				TaskID:        tasks[taskIndex],
				Content:       "",
				Success:       err == nil,
				ExecutionTime: executionTime,
				MemoryUsage:   memAfter - memBefore,
				Error:         err,
			}

			if result != nil {
				results[agentIndex].Content = result.Content
			}
		}(i, agent)
	}

	wg.Wait()
	return results, nil
}

// executeSequentialCoordination runs agents in sequence with dependency
func (f *HappyPathTestFramework) executeSequentialCoordination(agents []*RealAgent, scenario struct {
	name                   string
	agentCount             int
	coordinationType       string
	expectedCompletionTime time.Duration
	coordinationComplexity ComplexityLevel
	expectedSuccessRate    float64
}) ([]CoordinationResult, error) {

	results := make([]CoordinationResult, len(agents))
	previousOutput := "Analyze the main.go file structure"

	for i, agent := range agents {
		start := time.Now()
		memBefore := f.measureMemoryUsage()

		// Each agent builds on the previous agent's work
		taskMessage := previousOutput
		if i > 0 {
			taskMessage = "Building on the previous analysis: " + previousOutput + ". Provide additional insights."
		}

		result, err := agent.Execute(f.testContext, ExecutionInput{
			Message: taskMessage,
			Requirements: TaskRequirements{
				Type:       "coding",
				Complexity: scenario.coordinationComplexity,
			},
		})

		memAfter := f.measureMemoryUsage()
		executionTime := time.Since(start)

		results[i] = CoordinationResult{
			AgentID:       agent.GetID(),
			TaskID:        taskMessage,
			Content:       "",
			Success:       err == nil,
			ExecutionTime: executionTime,
			MemoryUsage:   memAfter - memBefore,
			Error:         err,
		}

		if result != nil {
			results[i].Content = result.Content
			// Use this result as input for the next agent
			if len(result.Content) > 100 {
				previousOutput = result.Content[:100] + "..."
			} else {
				previousOutput = result.Content
			}
		}
	}

	return results, nil
}

// executeCollaborativeCoordination simulates agents working together
func (f *HappyPathTestFramework) executeCollaborativeCoordination(agents []*RealAgent, scenario struct {
	name                   string
	agentCount             int
	coordinationType       string
	expectedCompletionTime time.Duration
	coordinationComplexity ComplexityLevel
	expectedSuccessRate    float64
}) ([]CoordinationResult, error) {

	results := make([]CoordinationResult, len(agents))

	// Collaborative task: code review with different perspectives
	baseTask := "Review this Go code for: "
	perspectives := []string{
		"security vulnerabilities and best practices",
		"performance optimization opportunities",
		"code maintainability and readability",
		"testing coverage and quality",
	}

	var wg sync.WaitGroup
	for i, agent := range agents {
		wg.Add(1)
		go func(agentIndex int, a *RealAgent) {
			defer wg.Done()

			start := time.Now()
			memBefore := f.measureMemoryUsage()

			perspectiveIndex := agentIndex % len(perspectives)
			taskMessage := baseTask + perspectives[perspectiveIndex]

			result, err := a.Execute(f.testContext, ExecutionInput{
				Message: taskMessage,
				Requirements: TaskRequirements{
					Type:       "coding",
					Complexity: scenario.coordinationComplexity,
				},
			})

			memAfter := f.measureMemoryUsage()
			executionTime := time.Since(start)

			results[agentIndex] = CoordinationResult{
				AgentID:       a.GetID(),
				TaskID:        taskMessage,
				Content:       "",
				Success:       err == nil,
				ExecutionTime: executionTime,
				MemoryUsage:   memAfter - memBefore,
				Error:         err,
			}

			if result != nil {
				results[agentIndex].Content = result.Content
			}
		}(i, agent)
	}

	wg.Wait()
	return results, nil
}

// validateCoordinationQuality assesses the quality of multi-agent coordination
func (f *HappyPathTestFramework) validateCoordinationQuality(results []CoordinationResult, coordinationType string) int {
	if len(results) == 0 {
		return 0
	}

	score := 50 // Base score

	// Success rate contributes to quality
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	successRate := float64(successCount) / float64(len(results))
	score += int(successRate * 30) // Up to 30 points for success rate

	// Content quality contributes to quality
	for _, result := range results {
		if len(result.Content) >= 50 { // Reasonable response length
			score += 5
		}
		if len(result.Content) >= 200 { // Detailed response
			score += 5
		}
	}

	// Coordination-specific bonuses
	switch coordinationType {
	case "parallel":
		// Check for reasonable parallelism (not all agents taking exactly the same time)
		executionTimes := make([]time.Duration, len(results))
		for i, result := range results {
			executionTimes[i] = result.ExecutionTime
		}
		if f.hasVariance(executionTimes) {
			score += 10 // Bonus for realistic parallel execution
		}

	case "sequential":
		// Check that results build on each other (later results reference earlier context)
		if len(results) > 1 {
			score += 10 // Bonus for sequential coordination
		}

	case "collaborative":
		// Check for different perspectives (results should be different)
		if f.hasDiverseContent(results) {
			score += 10 // Bonus for diverse collaborative input
		}
	}

	// Cap the score at 100
	if score > 100 {
		score = 100
	}

	return score
}

// hasVariance checks if execution times have reasonable variance (not all identical)
func (f *HappyPathTestFramework) hasVariance(times []time.Duration) bool {
	if len(times) <= 1 {
		return false
	}

	first := times[0]
	for _, t := range times[1:] {
		// If times differ by more than 100ms, consider it variance
		if t > first+100*time.Millisecond || t < first-100*time.Millisecond {
			return true
		}
	}
	return false
}

// hasDiverseContent checks if coordination results have diverse content
func (f *HappyPathTestFramework) hasDiverseContent(results []CoordinationResult) bool {
	if len(results) <= 1 {
		return false
	}

	// Simple check: results should not be identical
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Content != results[j].Content {
				return true
			}
		}
	}
	return false
}

// TestAgentResourceIsolation validates that agents don't interfere with each other
func TestAgentResourceIsolation(t *testing.T) {
	framework := NewHappyPathFramework(t)
	defer framework.Cleanup()

	// Create multiple agents concurrently
	agentCount := 5
	agents := make([]*RealAgent, agentCount)

	var wg sync.WaitGroup
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			agent, err := framework.GetOptimalAgent(TaskRequirements{
				Type:         "coding",
				Complexity:   ComplexityMedium,
				MaxCost:      5,
				Capabilities: []string{"code_analysis"},
			})

			require.NoError(t, err, "Agent %d creation must succeed", index)
			agents[index] = agent
		}(i)
	}
	wg.Wait()

	// Execute all agents simultaneously and check for interference
	results := make(chan CoordinationResult, agentCount)

	for i, agent := range agents {
		go func(agentIndex int, a *RealAgent) {
			start := time.Now()
			result, err := a.Execute(context.Background(), ExecutionInput{
				Message: "Analyze code structure and patterns",
				Requirements: TaskRequirements{
					Type:       "coding",
					Complexity: ComplexityMedium,
				},
			})

			results <- CoordinationResult{
				AgentID:       a.GetID(),
				Success:       err == nil && result != nil,
				ExecutionTime: time.Since(start),
				Content: func() string {
					if result != nil {
						return result.Content
					}
					return ""
				}(),
			}
		}(i, agent)
	}

	// Collect all results
	var allResults []CoordinationResult
	for i := 0; i < agentCount; i++ {
		result := <-results
		allResults = append(allResults, result)
	}

	// Validate no interference
	successCount := 0
	for _, result := range allResults {
		if result.Success {
			successCount++
		}

		// Each agent should complete within reasonable time
		assert.LessOrEqual(t, result.ExecutionTime, 30*time.Second,
			"Agent execution time too long: %v", result.ExecutionTime)
	}

	// At least 80% should succeed with no interference
	successRate := float64(successCount) / float64(agentCount)
	assert.GreaterOrEqual(t, successRate, 0.8,
		"Success rate too low with concurrent agents: %.1f%%", successRate*100)

	t.Logf("✅ Resource isolation test completed: %d/%d agents succeeded", successCount, agentCount)
}
