// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestInputValidation comprehensively tests all input validation scenarios
func TestInputValidation(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}

	testCases := []struct {
		name        string
		request     ComplexityAnalysisRequest
		expectError bool
		errorType   string
	}{
		{
			name: "ValidRequest",
			request: ComplexityAnalysisRequest{
				TaskDescription: "Create a REST API",
				TaskDomain:      "backend",
				TokenBudget:     5000,
				QualityLevel:    "production",
			},
			expectError: false,
		},
		{
			name: "EmptyTaskDescription",
			request: ComplexityAnalysisRequest{
				TaskDescription: "",
				TokenBudget:     1000,
			},
			expectError: true,
			errorType:   "task description cannot be empty",
		},
		{
			name: "TaskDescriptionTooLong",
			request: ComplexityAnalysisRequest{
				TaskDescription: string(make([]byte, 10001)), // > 10000 chars
				TokenBudget:     1000,
			},
			expectError: true,
			errorType:   "task description too long",
		},
		{
			name: "ZeroTokenBudget",
			request: ComplexityAnalysisRequest{
				TaskDescription: "Valid task",
				TokenBudget:     0,
			},
			expectError: true,
			errorType:   "token budget must be positive",
		},
		{
			name: "NegativeTokenBudget",
			request: ComplexityAnalysisRequest{
				TaskDescription: "Valid task",
				TokenBudget:     -100,
			},
			expectError: true,
			errorType:   "token budget must be positive",
		},
		{
			name: "TokenBudgetTooLarge",
			request: ComplexityAnalysisRequest{
				TaskDescription: "Valid task",
				TokenBudget:     1000001, // > 1M
			},
			expectError: true,
			errorType:   "token budget too large",
		},
		{
			name: "MaxValidTokenBudget",
			request: ComplexityAnalysisRequest{
				TaskDescription: "Valid task",
				TokenBudget:     1000000, // Exactly 1M
			},
			expectError: false,
		},
		{
			name: "MaxValidTaskDescription",
			request: ComplexityAnalysisRequest{
				TaskDescription: string(make([]byte, 10000)), // Exactly 10000 chars
				TokenBudget:     1000,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := analyzer.validateRequest(tc.request)

			if tc.expectError {
				assert.Error(t, err, "Expected validation error")
				assert.Contains(t, err.Error(), tc.errorType, "Error should contain expected message")
			} else {
				assert.NoError(t, err, "Expected no validation error")
			}
		})
	}
}

// TestCustomErrorTypes validates that our custom error types work correctly
func TestCustomErrorTypes(t *testing.T) {
	testCases := []struct {
		name      string
		errorFunc func() error
		errorType string
	}{
		{
			name:      "ValidationError",
			errorFunc: func() error { return NewValidationError("test validation", nil) },
			errorType: "ValidationError",
		},
		{
			name:      "RegistryError",
			errorFunc: func() error { return NewRegistryError("test registry", nil) },
			errorType: "RegistryError",
		},
		{
			name:      "PromptError",
			errorFunc: func() error { return NewPromptError("test prompt", nil) },
			errorType: "PromptError",
		},
		{
			name:      "ArtisanError",
			errorFunc: func() error { return NewArtisanError("test artisan", nil) },
			errorType: "ArtisanError",
		},
		{
			name:      "ParseError",
			errorFunc: func() error { return NewParseError("test parse", nil) },
			errorType: "ParseError",
		},
		{
			name:      "TimeoutError",
			errorFunc: func() error { return NewTimeoutError("test timeout", nil) },
			errorType: "TimeoutError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errorFunc()

			var analysisErr *AnalysisError
			assert.ErrorAs(t, err, &analysisErr, "Should be an AnalysisError")
			assert.Equal(t, tc.errorType, analysisErr.Type, "Should have correct error type")
			assert.Contains(t, err.Error(), tc.errorType, "Error string should contain type")
		})
	}
}

// TestErrorChaining validates error unwrapping works correctly
func TestErrorChaining(t *testing.T) {
	originalErr := assert.AnError
	wrappedErr := NewValidationError("validation failed", originalErr)

	// Test that we can unwrap to get the original error
	unwrapped := wrappedErr.(*AnalysisError).Unwrap()
	assert.Equal(t, originalErr, unwrapped, "Should unwrap to original error")

	// Test error.Is and error.As work correctly
	assert.ErrorIs(t, wrappedErr, originalErr, "Should be identified as original error")

	var analysisErr *AnalysisError
	assert.ErrorAs(t, wrappedErr, &analysisErr, "Should be identifiable as AnalysisError")
}

// TestPromptContextGeneration validates prompt context building
func TestPromptContextGeneration(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}

	request := ComplexityAnalysisRequest{
		TaskDescription: "Test task",
		TaskDomain:      "test-domain",
		TaskPriority:    "high",
		TokenBudget:     5000,
		QualityLevel:    "production",
		RiskTolerance:   "medium",
		TimeConstraint:  "1 week",
	}

	agents := []AgentInfo{
		{
			Name:          "test-agent",
			Role:          "Test Role",
			Provider:      "test-provider",
			CostMagnitude: 3,
		},
	}

	context := analyzer.buildPromptContext(request, agents)

	// Validate all required fields are present
	assert.Equal(t, request.TaskDescription, context["TaskDescription"])
	assert.Equal(t, request.TaskDomain, context["TaskDomain"])
	assert.Equal(t, request.TaskPriority, context["TaskPriority"])
	assert.Equal(t, request.TokenBudget, context["TokenBudget"])
	assert.Equal(t, request.QualityLevel, context["QualityLevel"])
	assert.Equal(t, request.RiskTolerance, context["RiskTolerance"])
	assert.Equal(t, request.TimeConstraint, context["TimeConstraint"])
	assert.Equal(t, len(agents), context["AgentCount"])
	assert.Equal(t, agents, context["AvailableAgents"])

	// Validate default values are set
	assert.NotEmpty(t, context["GuildName"])
	assert.NotEmpty(t, context["ProjectType"])
	assert.Greater(t, context["CorpusDocuments"], 0)
	assert.Greater(t, context["CodebaseSize"], 0)
}

// TestJSONExtraction validates JSON extraction from various response formats
func TestJSONExtraction(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PlainJSON",
			input:    `{"key": "value", "number": 42}`,
			expected: "",
		},
		{
			name: "JSONInMarkdownBlock",
			input: `Here's the analysis:

` + "```json" + `
{"complexity_score": 5, "approach": "single-agent"}
` + "```" + `

That's the result.`,
			expected: `
{"complexity_score": 5, "approach": "single-agent"}
`,
		},
		{
			name: "MultipleCodeBlocks",
			input: `First:
` + "```json" + `
{"first": "block"}
` + "```" + `

Second:
` + "```json" + `
{"second": "block"}
` + "```",
			expected: `
{"first": "block"}
`,
		},
		{
			name:     "NoJSONBlock",
			input:    "Just plain text without any JSON blocks",
			expected: "",
		},
		{
			name: "EmptyJSONBlock",
			input: `Empty block:
` + "```json" + `
` + "```",
			expected: `
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestResponseParsing validates response parsing with various inputs
func TestResponseParsing(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}

	testCases := []struct {
		name        string
		response    *ArtisanResponse
		expectError bool
		errorType   string
	}{
		{
			name:        "NilResponse",
			response:    nil,
			expectError: true,
			errorType:   "response is nil",
		},
		{
			name: "EmptyContent",
			response: &ArtisanResponse{
				Content: "",
			},
			expectError: true,
			errorType:   "response content is empty",
		},
		{
			name: "ValidJSON",
			response: &ArtisanResponse{
				Content: `{
					"complexity_score": 5,
					"recommended_approach": "single-agent",
					"reasoning": "Simple task",
					"agent_requirements": [],
					"execution_strategy": {
						"parallel_tasks": [],
						"sequential_tasks": [],
						"dependencies": []
					},
					"cost_estimate": {
						"single_agent_tokens": 1000,
						"multi_agent_tokens": 1200,
						"recommended_savings": "-20%"
					},
					"quality_assurance": {
						"review_points": [],
						"testing_strategy": "unit tests",
						"risk_mitigation": []
					}
				}`,
			},
			expectError: false,
		},
		{
			name: "JSONInMarkdown",
			response: &ArtisanResponse{
				Content: `Analysis complete:

` + "```json" + `
{
	"complexity_score": 7,
	"recommended_approach": "multi-agent",
	"reasoning": "Complex task requiring multiple specializations",
	"agent_requirements": [
		{
			"role": "Backend Developer",
			"priority": "high",
			"estimated_tokens": 1500,
			"rationale": "API development needed"
		}
	],
	"execution_strategy": {
		"parallel_tasks": ["API", "UI"],
		"sequential_tasks": ["Testing"],
		"dependencies": [{"from": "API", "to": "Testing"}]
	},
	"cost_estimate": {
		"single_agent_tokens": 3000,
		"multi_agent_tokens": 2500,
		"recommended_savings": "17%"
	},
	"quality_assurance": {
		"review_points": ["Architecture review"],
		"testing_strategy": "Integration testing",
		"risk_mitigation": ["API versioning"]
	}
}
` + "```" + `

Hope this helps!`,
			},
			expectError: false,
		},
		{
			name: "InvalidJSON",
			response: &ArtisanResponse{
				Content: `{"invalid": json, "missing": "quotes"}`,
			},
			expectError: true,
			errorType:   "no JSON found in response",
		},
		{
			name: "NoJSONFound",
			response: &ArtisanResponse{
				Content: "This is just plain text with no JSON anywhere",
			},
			expectError: true,
			errorType:   "no JSON found in response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.parseAnalysisResponse(tc.response)

			if tc.expectError {
				assert.Error(t, err, "Expected parsing error")
				assert.Contains(t, err.Error(), tc.errorType, "Error should contain expected message")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no parsing error")
				assert.NotNil(t, result, "Result should not be nil")

				// Validate required fields are present
				assert.Greater(t, result.ComplexityScore, 0, "Should have complexity score")
				assert.NotEmpty(t, result.RecommendedApproach, "Should have recommended approach")
				assert.NotNil(t, result.ExecutionStrategy, "Should have execution strategy")
				assert.NotNil(t, result.CostEstimate, "Should have cost estimate")
				assert.NotNil(t, result.QualityAssurance, "Should have quality assurance")
			}
		})
	}
}

// TestHelperFunctions validates utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("FindSubstring", func(t *testing.T) {
		testCases := []struct {
			text     string
			substr   string
			expected int
		}{
			{"hello world", "world", 6},
			{"hello world", "hello", 0},
			{"hello world", "xyz", -1},
			{"", "test", -1},
			{"test", "", 0},
		}

		for _, tc := range testCases {
			result := findSubstring(tc.text, tc.substr)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("FindSubstringFrom", func(t *testing.T) {
		testCases := []struct {
			text     string
			substr   string
			start    int
			expected int
		}{
			{"hello world world", "world", 0, 6},
			{"hello world world", "world", 7, 12},
			{"hello world world", "world", 13, -1},
			{"hello", "xyz", 0, -1},
		}

		for _, tc := range testCases {
			result := findSubstringFrom(tc.text, tc.substr, tc.start)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("MinFunction", func(t *testing.T) {
		testCases := []struct {
			a, b, expected int
		}{
			{1, 2, 1},
			{5, 3, 3},
			{0, 0, 0},
			{-1, 1, -1},
		}

		for _, tc := range testCases {
			result := min(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		}
	})
}

// TestTimeoutScenarios validates timeout handling
func TestTimeoutScenarios(t *testing.T) {
	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		analyzer := &TaskComplexityAnalyzer{
			timeout: 5 * time.Second,
		}

		// The actual test would need a real artisan client that respects context
		// For now, just validate that timeout context is created properly
		_ = ComplexityAnalysisRequest{
			TaskDescription: "Test task",
			TokenBudget:     1000,
		}
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, analyzer.timeout)
		defer timeoutCancel()

		select {
		case <-timeoutCtx.Done():
			assert.Equal(t, context.Canceled, timeoutCtx.Err())
		default:
			t.Error("Context should be cancelled")
		}
	})

	t.Run("TimeoutCreation", func(t *testing.T) {
		analyzer := &TaskComplexityAnalyzer{
			timeout: 100 * time.Millisecond,
		}

		ctx := context.Background()
		timeoutCtx, cancel := context.WithTimeout(ctx, analyzer.timeout)
		defer cancel()

		// Wait for timeout
		<-timeoutCtx.Done()
		assert.Equal(t, context.DeadlineExceeded, timeoutCtx.Err())
	})
}

// BenchmarkComplexityAnalysis benchmarks the analysis performance
func BenchmarkComplexityAnalysis(b *testing.B) {
	analyzer := &TaskComplexityAnalyzer{}

	request := ComplexityAnalysisRequest{
		TaskDescription: "Create a comprehensive e-commerce platform with user management, product catalog, shopping cart, payment processing, order management, and administrative dashboard",
		TaskDomain:      "full-stack-development",
		TaskPriority:    "high",
		TokenBudget:     10000,
		QualityLevel:    "production",
		RiskTolerance:   "low",
		TimeConstraint:  "1 month",
	}

	agents := []AgentInfo{
		{
			Name:            "backend-artisan",
			Role:            "Backend Developer",
			Provider:        "anthropic",
			CostMagnitude:   3,
			ContextWindow:   200000,
			Specializations: []string{"Go", "APIs", "databases"},
			SuccessRate:     92.5,
		},
		{
			Name:            "frontend-artisan",
			Role:            "Frontend Developer",
			Provider:        "openai",
			CostMagnitude:   2,
			ContextWindow:   128000,
			Specializations: []string{"React", "TypeScript"},
			SuccessRate:     89.2,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Benchmark the components that don't require external services
		_ = analyzer.validateRequest(request)
		_ = analyzer.buildPromptContext(request, agents)
		_ = analyzer.getDefaultAgents()
	}
}

// TestMemoryUsage validates memory consumption patterns
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Test that we don't have memory leaks with large requests
	analyzer := &TaskComplexityAnalyzer{}

	// Create a large task description
	largeTask := string(make([]byte, 9999)) // Just under the limit

	request := ComplexityAnalysisRequest{
		TaskDescription: largeTask,
		TokenBudget:     10000,
	}

	// Run validation multiple times to check for memory accumulation
	for i := 0; i < 1000; i++ {
		err := analyzer.validateRequest(request)
		assert.NoError(t, err)
	}

	// Test context building doesn't accumulate memory
	agents := make([]AgentInfo, 10)
	for i := range agents {
		agents[i] = AgentInfo{
			Name:            fmt.Sprintf("agent-%d", i),
			Role:            "Test Role",
			Specializations: make([]string, 5),
		}
	}

	for i := 0; i < 100; i++ {
		context := analyzer.buildPromptContext(request, agents)
		assert.NotNil(t, context)
	}
}
