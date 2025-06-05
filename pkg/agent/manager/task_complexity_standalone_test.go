package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestComplexityAnalysisValidationStandalone tests validation without import dependencies
func TestComplexityAnalysisValidationStandalone(t *testing.T) {
	// Create a minimal analyzer for testing validation only
	analyzer := &TaskComplexityAnalyzer{
		timeout: 30 * time.Second,
	}

	t.Run("ValidRequest", func(t *testing.T) {
		request := ComplexityAnalysisRequest{
			TaskDescription: "Create a REST API for user management",
			TaskDomain:      "backend",
			TokenBudget:     5000,
			QualityLevel:    "production",
		}

		err := analyzer.validateRequest(request)
		assert.NoError(t, err, "Valid request should pass validation")
	})

	t.Run("EmptyTaskDescription", func(t *testing.T) {
		request := ComplexityAnalysisRequest{
			TaskDescription: "",
			TokenBudget:     1000,
		}

		err := analyzer.validateRequest(request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task description cannot be empty")
	})

	t.Run("InvalidTokenBudget", func(t *testing.T) {
		request := ComplexityAnalysisRequest{
			TaskDescription: "Valid task",
			TokenBudget:     -100,
		}

		err := analyzer.validateRequest(request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token budget must be positive")
	})
}

// TestPromptContextBuildingStandalone tests context building without dependencies
func TestPromptContextBuildingStandalone(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}

	request := ComplexityAnalysisRequest{
		TaskDescription: "Build user authentication system",
		TaskDomain:      "security",
		TaskPriority:    "high",
		TokenBudget:     3000,
		QualityLevel:    "production",
		RiskTolerance:   "low",
		TimeConstraint:  "1 week",
	}

	agents := []AgentInfo{
		{
			Name:            "security-specialist",
			Role:            "Security Engineer",
			Provider:        "anthropic",
			Model:           "claude-3-5-sonnet",
			CostMagnitude:   4,
			ContextWindow:   200000,
			Specializations: []string{"authentication", "security", "cryptography"},
			Tools:           []string{"file", "shell", "security-scanner"},
			SuccessRate:     96.0,
		},
	}

	context := analyzer.buildPromptContext(request, agents)

	// Verify all request fields are included
	assert.Equal(t, request.TaskDescription, context["TaskDescription"])
	assert.Equal(t, request.TaskDomain, context["TaskDomain"])
	assert.Equal(t, request.TaskPriority, context["TaskPriority"])
	assert.Equal(t, request.TokenBudget, context["TokenBudget"])
	assert.Equal(t, request.QualityLevel, context["QualityLevel"])
	assert.Equal(t, request.RiskTolerance, context["RiskTolerance"])
	assert.Equal(t, request.TimeConstraint, context["TimeConstraint"])

	// Verify agent information is included
	assert.Equal(t, len(agents), context["AgentCount"])
	assert.Equal(t, agents, context["AvailableAgents"])

	// Verify default context values are set
	assert.NotEmpty(t, context["GuildName"])
	assert.NotEmpty(t, context["ProjectType"])
	assert.Greater(t, context["CorpusDocuments"], 0)
	assert.Greater(t, context["CodebaseSize"], 0)
}

// TestErrorTypesStandalone tests custom error types without external dependencies
func TestErrorTypesStandalone(t *testing.T) {
	testCases := []struct {
		name        string
		errorFunc   func() error
		expectedType string
	}{
		{
			name:        "ValidationError",
			errorFunc:   func() error { return NewValidationError("test validation error", nil) },
			expectedType: "ValidationError",
		},
		{
			name:        "RegistryError",
			errorFunc:   func() error { return NewRegistryError("test registry error", nil) },
			expectedType: "RegistryError",
		},
		{
			name:        "PromptError",
			errorFunc:   func() error { return NewPromptError("test prompt error", nil) },
			expectedType: "PromptError",
		},
		{
			name:        "ArtisanError",
			errorFunc:   func() error { return NewArtisanError("test artisan error", nil) },
			expectedType: "ArtisanError",
		},
		{
			name:        "ParseError",
			errorFunc:   func() error { return NewParseError("test parse error", nil) },
			expectedType: "ParseError",
		},
		{
			name:        "TimeoutError",
			errorFunc:   func() error { return NewTimeoutError("test timeout error", nil) },
			expectedType: "TimeoutError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.errorFunc()
			
			// Test that it's an AnalysisError
			var analysisErr *AnalysisError
			assert.ErrorAs(t, err, &analysisErr)
			assert.Equal(t, tc.expectedType, analysisErr.Type)
			
			// Test error string contains type
			assert.Contains(t, err.Error(), tc.expectedType)
		})
	}
}

// TestJSONExtractionStandalone tests JSON extraction without dependencies
func TestJSONExtractionStandalone(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "ValidJSONBlock",
			input: `Here's the analysis:

` + "```json" + `
{"complexity_score": 7, "approach": "multi-agent"}
` + "```" + `

End of analysis.`,
			expected: `
{"complexity_score": 7, "approach": "multi-agent"}
`,
		},
		{
			name: "MultipleBlocks",
			input: `First block:
` + "```json" + `
{"first": "result"}
` + "```" + `

Second block:
` + "```json" + `
{"second": "result"}
` + "```",
			expected: `
{"first": "result"}
`,
		},
		{
			name:     "NoJSONBlock",
			input:    "This is just text without any JSON code blocks",
			expected: "",
		},
		{
			name: "EmptyJSONBlock",
			input: `Empty:
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

// TestResponseParsingStandalone tests response parsing without external dependencies
func TestResponseParsingStandalone(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}

	t.Run("ValidJSONResponse", func(t *testing.T) {
		response := &ArtisanResponse{
			Content: `{
				"complexity_score": 6,
				"recommended_approach": "multi-agent",
				"reasoning": "Task requires multiple specializations",
				"agent_requirements": [
					{
						"role": "Backend Developer",
						"priority": "high",
						"estimated_tokens": 2000,
						"rationale": "API development needed"
					}
				],
				"execution_strategy": {
					"parallel_tasks": ["Backend API", "Frontend UI"],
					"sequential_tasks": ["Integration Testing"],
					"dependencies": [{"from": "Backend API", "to": "Integration Testing"}]
				},
				"cost_estimate": {
					"single_agent_tokens": 4000,
					"multi_agent_tokens": 3200,
					"recommended_savings": "20%"
				},
				"quality_assurance": {
					"review_points": ["API design review", "UI/UX review"],
					"testing_strategy": "Component and integration testing",
					"risk_mitigation": ["API versioning", "Cross-browser testing"]
				}
			}`,
		}

		result, err := analyzer.parseAnalysisResponse(response)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 6, result.ComplexityScore)
		assert.Equal(t, "multi-agent", result.RecommendedApproach)
		assert.Equal(t, "Task requires multiple specializations", result.Reasoning)
		assert.Len(t, result.AgentRequirements, 1)
		assert.Equal(t, "Backend Developer", result.AgentRequirements[0].Role)
	})

	t.Run("JSONInMarkdown", func(t *testing.T) {
		response := &ArtisanResponse{
			Content: `Here's my analysis:

` + "```json" + `
{
	"complexity_score": 4,
	"recommended_approach": "single-agent",
	"reasoning": "Simple CRUD operations",
	"agent_requirements": [
		{
			"role": "Backend Developer",
			"priority": "medium",
			"estimated_tokens": 1500,
			"rationale": "Standard backend work"
		}
	],
	"execution_strategy": {
		"parallel_tasks": [],
		"sequential_tasks": ["Design schema", "Implement endpoints", "Add tests"],
		"dependencies": []
	},
	"cost_estimate": {
		"single_agent_tokens": 1500,
		"multi_agent_tokens": 2000,
		"recommended_savings": "-33%"
	},
	"quality_assurance": {
		"review_points": ["Schema review"],
		"testing_strategy": "Unit testing",
		"risk_mitigation": ["Input validation"]
	}
}
` + "```" + `

That's my assessment.`,
		}

		result, err := analyzer.parseAnalysisResponse(response)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 4, result.ComplexityScore)
		assert.Equal(t, "single-agent", result.RecommendedApproach)
	})

	t.Run("InvalidResponse", func(t *testing.T) {
		response := &ArtisanResponse{
			Content: "This is not JSON at all",
		}

		result, err := analyzer.parseAnalysisResponse(response)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no JSON found in response")
	})

	t.Run("NilResponse", func(t *testing.T) {
		result, err := analyzer.parseAnalysisResponse(nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "response is nil")
	})
}

// TestDefaultAgentsStandalone tests the default agent configuration
func TestDefaultAgentsStandalone(t *testing.T) {
	analyzer := &TaskComplexityAnalyzer{}
	
	agents := analyzer.getDefaultAgents()
	
	assert.NotEmpty(t, agents, "Should have default agents")
	assert.GreaterOrEqual(t, len(agents), 2, "Should have at least 2 default agents")
	
	// Verify agent structure
	for _, agent := range agents {
		assert.NotEmpty(t, agent.Name, "Agent should have name")
		assert.NotEmpty(t, agent.Role, "Agent should have role")
		assert.NotEmpty(t, agent.Provider, "Agent should have provider")
		assert.NotEmpty(t, agent.Model, "Agent should have model")
		assert.Greater(t, agent.CostMagnitude, 0, "Agent should have cost magnitude")
		assert.Greater(t, agent.ContextWindow, 0, "Agent should have context window")
		assert.NotEmpty(t, agent.Specializations, "Agent should have specializations")
		assert.Greater(t, agent.SuccessRate, 0.0, "Agent should have success rate")
	}
	
	// Look for expected roles
	roles := make(map[string]bool)
	for _, agent := range agents {
		roles[agent.Role] = true
	}
	
	assert.True(t, roles["Backend Developer"], "Should have backend developer")
	assert.True(t, roles["Frontend Developer"], "Should have frontend developer")
}

// TestUtilityFunctionsStandalone tests utility functions
func TestUtilityFunctionsStandalone(t *testing.T) {
	t.Run("MinFunction", func(t *testing.T) {
		assert.Equal(t, 1, min(1, 2))
		assert.Equal(t, 3, min(5, 3))
		assert.Equal(t, 0, min(0, 0))
		assert.Equal(t, -5, min(-5, -2))
	})

	t.Run("FindSubstring", func(t *testing.T) {
		assert.Equal(t, 6, findSubstring("hello world", "world"))
		assert.Equal(t, 0, findSubstring("hello world", "hello"))
		assert.Equal(t, -1, findSubstring("hello world", "xyz"))
		assert.Equal(t, -1, findSubstring("", "test"))
	})

	t.Run("FindSubstringFrom", func(t *testing.T) {
		assert.Equal(t, 6, findSubstringFrom("hello world world", "world", 0))
		assert.Equal(t, 12, findSubstringFrom("hello world world", "world", 7))
		assert.Equal(t, -1, findSubstringFrom("hello world world", "world", 13))
	})
}

// BenchmarkValidationStandalone benchmarks validation performance
func BenchmarkValidationStandalone(b *testing.B) {
	analyzer := &TaskComplexityAnalyzer{}
	
	request := ComplexityAnalysisRequest{
		TaskDescription: "Create a comprehensive microservices architecture with API gateway, service discovery, load balancing, monitoring, logging, and distributed tracing",
		TaskDomain:      "system-architecture",
		TaskPriority:    "high",
		TokenBudget:     15000,
		QualityLevel:    "production",
		RiskTolerance:   "low",
		TimeConstraint:  "6 weeks",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = analyzer.validateRequest(request)
	}
}

// BenchmarkContextBuildingStandalone benchmarks context building
func BenchmarkContextBuildingStandalone(b *testing.B) {
	analyzer := &TaskComplexityAnalyzer{}
	
	request := ComplexityAnalysisRequest{
		TaskDescription: "Large scale system implementation",
		TaskDomain:      "enterprise",
		TaskPriority:    "critical",
		TokenBudget:     20000,
		QualityLevel:    "enterprise",
		RiskTolerance:   "minimal",
		TimeConstraint:  "3 months",
	}
	
	agents := make([]AgentInfo, 10)
	for i := range agents {
		agents[i] = AgentInfo{
			Name:            fmt.Sprintf("agent-%d", i),
			Role:            "Specialist",
			Provider:        "anthropic",
			Model:           "claude-3-5-sonnet",
			CostMagnitude:   3,
			ContextWindow:   200000,
			Specializations: []string{"expertise-1", "expertise-2", "expertise-3"},
			Tools:           []string{"tool-1", "tool-2"},
			SuccessRate:     90.0,
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = analyzer.buildPromptContext(request, agents)
	}
}