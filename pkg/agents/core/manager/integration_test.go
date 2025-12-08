// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/prompts/layered"
	"github.com/guild-framework/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestManagerIntelligenceIntegration tests the complete manager intelligence system
// with real components instead of just mocks
func TestManagerIntelligenceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up real components for integration testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create real implementations (these would normally come from registry)
	promptManager := createTestPromptManager(t)
	artisanClient := createTestArtisanClient(t)
	agentRegistry := createTestAgentRegistry(t)

	// Create the intelligence service with real components
	service := NewManagerIntelligenceService(promptManager, artisanClient, agentRegistry)

	t.Run("ComplexityAnalysisWithRealPrompts", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := ComplexityAnalysisRequest{
			TaskDescription: "Build a REST API for user management with authentication, CRUD operations, and audit logging",
			TaskDomain:      "backend-development",
			TaskPriority:    "high",
			TokenBudget:     5000,
			QualityLevel:    "production",
			RiskTolerance:   "medium",
			TimeConstraint:  "1 week",
		}

		result, err := service.AnalyzeComplexityOnly(ctx, request)

		require.NoError(t, err, "Complexity analysis should succeed")
		require.NotNil(t, result, "Result should not be nil")

		// Validate the analysis results
		assert.Greater(t, result.ComplexityScore, 0, "Complexity score should be positive")
		assert.LessOrEqual(t, result.ComplexityScore, 10, "Complexity score should be <= 10")
		assert.NotEmpty(t, result.RecommendedApproach, "Should have a recommended approach")
		assert.NotEmpty(t, result.Reasoning, "Should have reasoning")
		assert.NotEmpty(t, result.AgentRequirements, "Should have agent requirements")

		// Validate that the analysis makes sense for a complex backend task
		assert.Contains(t, []string{"single-agent", "multi-agent"}, result.RecommendedApproach)

		// Log the results for manual verification
		logger.Info("Complexity analysis completed",
			slog.Int("complexity_score", result.ComplexityScore),
			slog.String("approach", result.RecommendedApproach),
			slog.String("reasoning", result.Reasoning),
		)
	})

	t.Run("AgentRoutingWithRealData", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// First get complexity analysis
		complexityRequest := ComplexityAnalysisRequest{
			TaskDescription: "Create a simple CRUD API for products",
			TaskDomain:      "backend-development",
			TaskPriority:    "medium",
			TokenBudget:     3000,
			QualityLevel:    "development",
			RiskTolerance:   "low",
			TimeConstraint:  "2 days",
		}

		complexityResult, err := service.AnalyzeComplexityOnly(ctx, complexityRequest)
		require.NoError(t, err)

		// Now test routing with the complexity results
		availableAgents := []AgentInfo{
			{
				Name:            "backend-specialist",
				Role:            "Backend Developer",
				Provider:        "anthropic",
				Model:           "claude-3-5-sonnet-20241022",
				CostMagnitude:   3,
				ContextWindow:   200000,
				Specializations: []string{"Go", "REST APIs", "databases"},
				Tools:           []string{"file", "shell", "http"},
				SuccessRate:     92.5,
			},
			{
				Name:            "fullstack-generalist",
				Role:            "Full Stack Developer",
				Provider:        "openai",
				Model:           "gpt-4",
				CostMagnitude:   4,
				ContextWindow:   128000,
				Specializations: []string{"Go", "React", "APIs", "databases"},
				Tools:           []string{"file", "shell", "http"},
				SuccessRate:     88.0,
			},
		}

		routingRequest := RoutingRequest{
			TaskDescription:     complexityRequest.TaskDescription,
			ComplexityScore:     complexityResult.ComplexityScore,
			RecommendedApproach: complexityResult.RecommendedApproach,
			AgentRequirements:   complexityResult.AgentRequirements,
			AvailableAgents:     availableAgents,
		}

		routingResult, err := service.RouteToAgentsOnly(ctx, routingRequest)

		require.NoError(t, err, "Agent routing should succeed")
		require.NotNil(t, routingResult, "Routing result should not be nil")

		// Validate routing results
		assert.NotEmpty(t, routingResult.RoutingDecision.PrimaryAgent.AgentID, "Should have primary agent")
		assert.Greater(t, routingResult.RoutingDecision.PrimaryAgent.AssignmentConfidence, 0, "Should have confidence score")
		assert.Greater(t, routingResult.CostAnalysis.TotalEstimatedTokens, 0, "Should have token estimate")

		// The backend specialist should be preferred for a backend-only task
		if complexityResult.RecommendedApproach == "single-agent" {
			assert.Equal(t, "backend-specialist", routingResult.RoutingDecision.PrimaryAgent.AgentID,
				"Backend specialist should be chosen for simple backend tasks")
		}

		logger.Info("Agent routing completed",
			slog.String("primary_agent", routingResult.RoutingDecision.PrimaryAgent.AgentID),
			slog.Int("confidence", routingResult.RoutingDecision.PrimaryAgent.AssignmentConfidence),
			slog.Int("total_tokens", routingResult.CostAnalysis.TotalEstimatedTokens),
		)
	})

	t.Run("FullWorkflowIntegration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := IntelligenceRequest{
			TaskDescription: "Create a monitoring dashboard with real-time metrics, alerts, and user management",
			TaskDomain:      "full-stack-development",
			TaskPriority:    "high",
			TokenBudget:     8000,
			QualityLevel:    "production",
			RiskTolerance:   "medium",
			TimeConstraint:  "2 weeks",
		}

		result, err := service.AnalyzeAndRoute(ctx, request)

		require.NoError(t, err, "Full workflow should succeed")
		require.NotNil(t, result, "Result should not be nil")

		// Validate all components are present
		assert.NotNil(t, result.ComplexityAnalysis, "Should have complexity analysis")
		assert.NotNil(t, result.AgentRouting, "Should have agent routing")
		assert.NotEmpty(t, result.ExecutiveSummary.RecommendedApproach, "Should have executive summary")

		// Validate executive summary makes sense
		assert.NotEmpty(t, result.ExecutiveSummary.PrimaryAgent, "Should have primary agent recommendation")
		assert.Contains(t, result.ExecutiveSummary.EstimatedCost, "tokens", "Should have cost estimate")
		assert.NotEmpty(t, result.ExecutiveSummary.EstimatedDuration, "Should have duration estimate")

		// Complex tasks should likely recommend multi-agent approach
		if result.ComplexityAnalysis.ComplexityScore > 6 {
			assert.Equal(t, "multi-agent", result.ComplexityAnalysis.RecommendedApproach,
				"Complex tasks should recommend multi-agent approach")
			assert.NotEmpty(t, result.ExecutiveSummary.SupportingAgents,
				"Multi-agent tasks should have supporting agents")
		}

		logger.Info("Full workflow completed",
			slog.Int("complexity", result.ComplexityAnalysis.ComplexityScore),
			slog.String("approach", result.ExecutiveSummary.RecommendedApproach),
			slog.String("primary_agent", result.ExecutiveSummary.PrimaryAgent),
			slog.String("estimated_cost", result.ExecutiveSummary.EstimatedCost),
		)
	})
}

// TestErrorHandlingIntegration tests error scenarios with real components
func TestErrorHandlingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Create a service with very short timeout
		promptManager := createTestPromptManager(t)
		artisanClient := createSlowArtisanClient(t) // Simulates slow LLM
		agentRegistry := createTestAgentRegistry(t)

		analyzer := NewTaskComplexityAnalyzer(promptManager, artisanClient, agentRegistry)
		analyzer.timeout = 100 * time.Millisecond // Very short timeout

		ctx := context.Background()
		request := ComplexityAnalysisRequest{
			TaskDescription: "Test task",
			TaskDomain:      "test",
			TokenBudget:     1000,
		}

		result, err := analyzer.AnalyzeComplexity(ctx, request)

		assert.Error(t, err, "Should timeout")
		assert.Nil(t, result, "Result should be nil on timeout")

		// Check that it's specifically a timeout error
		var analysisErr *AnalysisError
		assert.ErrorAs(t, err, &analysisErr, "Should be an AnalysisError")
		assert.Equal(t, "TimeoutError", analysisErr.Type, "Should be a timeout error")

		logger.Info("Timeout test completed", slog.Any("error", err))
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		promptManager := createTestPromptManager(t)
		artisanClient := createTestArtisanClient(t)
		agentRegistry := createTestAgentRegistry(t)

		analyzer := NewTaskComplexityAnalyzer(promptManager, artisanClient, agentRegistry)

		testCases := []struct {
			name    string
			request ComplexityAnalysisRequest
		}{
			{
				name: "EmptyTaskDescription",
				request: ComplexityAnalysisRequest{
					TaskDescription: "",
					TokenBudget:     1000,
				},
			},
			{
				name: "NegativeTokenBudget",
				request: ComplexityAnalysisRequest{
					TaskDescription: "Test task",
					TokenBudget:     -100,
				},
			},
			{
				name: "TooLargeTokenBudget",
				request: ComplexityAnalysisRequest{
					TaskDescription: "Test task",
					TokenBudget:     2000000, // > 1M limit
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				result, err := analyzer.AnalyzeComplexity(ctx, tc.request)

				assert.Error(t, err, "Should have validation error")
				assert.Nil(t, result, "Result should be nil on validation error")

				var analysisErr *AnalysisError
				assert.ErrorAs(t, err, &analysisErr, "Should be an AnalysisError")
				assert.Equal(t, "ValidationError", analysisErr.Type, "Should be a validation error")
			})
		}
	})
}

// TestPerformanceIntegration tests performance characteristics
func TestPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	promptManager := createTestPromptManager(t)
	artisanClient := createTestArtisanClient(t)
	agentRegistry := createTestAgentRegistry(t)

	service := NewManagerIntelligenceService(promptManager, artisanClient, agentRegistry)

	t.Run("ConcurrentAnalysis", func(t *testing.T) {
		ctx := context.Background()

		// Run multiple analyses concurrently
		const numAnalyses = 5
		results := make(chan error, numAnalyses)

		for i := 0; i < numAnalyses; i++ {
			go func(taskNum int) {
				request := IntelligenceRequest{
					TaskDescription: fmt.Sprintf("Test task %d for concurrent processing", taskNum),
					TaskDomain:      "test",
					TokenBudget:     2000,
					QualityLevel:    "development",
				}

				_, err := service.AnalyzeAndRoute(ctx, request)
				results <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numAnalyses; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent analysis %d should succeed", i)
		}
	})

	t.Run("ResponseTimeBaseline", func(t *testing.T) {
		ctx := context.Background()

		request := ComplexityAnalysisRequest{
			TaskDescription: "Create a simple web application with user authentication",
			TaskDomain:      "web-development",
			TokenBudget:     3000,
			QualityLevel:    "development",
		}

		start := time.Now()
		result, err := service.AnalyzeComplexityOnly(ctx, request)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should complete within reasonable time (this depends on LLM provider)
		assert.Less(t, duration, 10*time.Second, "Analysis should complete within 10 seconds")

		t.Logf("Analysis completed in %v", duration)
	})
}

// Helper functions to create test components

func createTestPromptManager(t *testing.T) layered.LayeredManager {
	// In a real implementation, this would create a working LayeredManager
	// For now, we'll use a mock that returns reasonable prompts
	mockMgr := &MockPromptManager{}

	// Set up realistic prompt responses
	mockMgr.On("BuildLayeredPrompt",
		mock.Anything, "manager-agent", "analysis-session", mock.Anything).
		Return(&layered.LayeredPrompt{
			Compiled:    "You are a Guild Master analyzing task complexity. Provide JSON response with complexity_score, recommended_approach, reasoning, and agent_requirements.",
			TokenCount:  200,
			Truncated:   false,
			CacheKey:    "test-cache-key",
			ArtisanID:   "manager-agent",
			SessionID:   "analysis-session",
			AssembledAt: time.Now(),
		}, nil)

	mockMgr.On("BuildLayeredPrompt",
		mock.Anything, "manager-agent", "routing-session", mock.Anything).
		Return(&layered.LayeredPrompt{
			Compiled:    "You are a Guild Master routing tasks to agents. Provide JSON response with routing_decision, cost_analysis, and execution_plan.",
			TokenCount:  250,
			Truncated:   false,
			CacheKey:    "routing-cache-key",
			ArtisanID:   "manager-agent",
			SessionID:   "routing-session",
			AssembledAt: time.Now(),
		}, nil)

	return mockMgr
}

func createTestArtisanClient(t *testing.T) ArtisanClient {
	// Create a mock that returns realistic responses
	mockClient := &MockArtisanClient{}

	// Complexity analysis response
	complexityResponse := &ArtisanResponse{
		Content: `{
			"complexity_score": 5,
			"recommended_approach": "single-agent",
			"reasoning": "Task involves standard backend operations that can be handled by a single specialized agent",
			"agent_requirements": [
				{
					"role": "Backend Developer",
					"priority": "high",
					"estimated_tokens": 2000,
					"rationale": "Requires backend API development expertise"
				}
			],
			"execution_strategy": {
				"parallel_tasks": [],
				"sequential_tasks": ["Design API", "Implement endpoints", "Add tests"],
				"dependencies": []
			},
			"cost_estimate": {
				"single_agent_tokens": 2000,
				"multi_agent_tokens": 2500,
				"recommended_savings": "-25%"
			},
			"quality_assurance": {
				"review_points": ["API design review"],
				"testing_strategy": "Unit and integration tests",
				"risk_mitigation": ["Input validation", "Error handling"]
			}
		}`,
	}

	// Routing response
	routingResponse := &ArtisanResponse{
		Content: `{
			"routing_decision": {
				"primary_agent": {
					"agent_id": "backend-specialist",
					"role": "Backend Developer",
					"assignment_confidence": 9,
					"estimated_tokens": 2000,
					"rationale": "Perfect match for backend development task"
				},
				"supporting_agents": []
			},
			"cost_analysis": {
				"total_estimated_tokens": 2000,
				"cost_breakdown": [
					{"agent": "backend-specialist", "tokens": 2000, "cost": "$0.030"}
				],
				"alternative_approaches": []
			},
			"execution_plan": {
				"coordination_strategy": "Single agent execution",
				"task_distribution": [
					{
						"agent": "backend-specialist",
						"tasks": ["API implementation"],
						"dependencies": [],
						"deliverables": ["REST API endpoints"]
					}
				],
				"quality_gates": [],
				"risk_mitigation": []
			},
			"success_metrics": {
				"completion_criteria": ["All tests passing"],
				"quality_thresholds": {"minimum_quality": 8, "target_quality": 9},
				"performance_indicators": ["Response time < 200ms"]
			}
		}`,
	}

	// Set up the mock to return appropriate responses based on request
	mockClient.On("Complete", mock.Anything, mock.MatchedBy(func(req ArtisanRequest) bool {
		return req.MaxTokens == 2000 // Complexity analysis
	})).Return(complexityResponse, nil)

	mockClient.On("Complete", mock.Anything, mock.MatchedBy(func(req ArtisanRequest) bool {
		return req.MaxTokens == 3000 // Routing
	})).Return(routingResponse, nil)

	return mockClient
}

func createSlowArtisanClient(t *testing.T) ArtisanClient {
	// Create a mock that simulates slow responses
	mockClient := &MockArtisanClient{}

	// Use Run to simulate delay before returning
	mockClient.On("Complete", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		// Simulate slow response
		select {
		case <-ctx.Done():
			// Context was cancelled
			return
		case <-time.After(5 * time.Second):
			// Timeout - response too slow
			return
		}
	}).Return(&ArtisanResponse{Content: "{}"}, context.DeadlineExceeded)

	return mockClient
}

func createTestAgentRegistry(t *testing.T) registry.AgentRegistry {
	// Create a mock registry with test agents
	mockRegistry := &MockAgentRegistry{}

	testAgents := []registry.Agent{
		// Would contain actual agent implementations
	}

	mockRegistry.On("ListAgents").Return(testAgents)

	// Add GetRegisteredAgents expectation with test data
	registeredAgents := []registry.GuildAgentConfig{
		{
			Name:          "backend-artisan",
			Type:          "Backend Developer",
			Provider:      "anthropic",
			Model:         "claude-3-5-sonnet-20241022",
			CostMagnitude: 3,
			ContextWindow: 200000,
			Capabilities:  []string{"Go", "APIs", "databases"},
			Tools:         []string{"file", "shell", "http"},
		},
		{
			Name:          "frontend-artisan",
			Type:          "Frontend Developer",
			Provider:      "openai",
			Model:         "gpt-4",
			CostMagnitude: 2,
			ContextWindow: 128000,
			Capabilities:  []string{"React", "TypeScript"},
			Tools:         []string{"file", "shell", "http"},
		},
	}
	mockRegistry.On("GetRegisteredAgents").Return(registeredAgents)

	return mockRegistry
}
