package manager

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockArtisanClient for testing
type MockArtisanClient struct {
	mock.Mock
}

func (m *MockArtisanClient) Complete(ctx context.Context, request ArtisanRequest) (*ArtisanResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*ArtisanResponse), args.Error(1)
}

// MockPromptManager for testing - implements LayeredManager interface
type MockPromptManager struct {
	mock.Mock
}

// Manager interface methods
func (m *MockPromptManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	args := m.Called(ctx, role, domain)
	return args.String(0), args.Error(1)
}

func (m *MockPromptManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	args := m.Called(ctx, templateName)
	return args.String(0), args.Error(1)
}

func (m *MockPromptManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	args := m.Called(ctx, context)
	return args.String(0), args.Error(1)
}

func (m *MockPromptManager) ListRoles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPromptManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	args := m.Called(ctx, role)
	return args.Get(0).([]string), args.Error(1)
}

// LayeredManager interface methods
func (m *MockPromptManager) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	args := m.Called(ctx, artisanID, sessionID, turnCtx)
	return args.Get(0).(*layered.LayeredPrompt), args.Error(1)
}

func (m *MockPromptManager) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	args := m.Called(ctx, layer, artisanID, sessionID)
	return args.Get(0).(*layered.SystemPrompt), args.Error(1)
}

func (m *MockPromptManager) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	args := m.Called(ctx, prompt)
	return args.Error(0)
}

func (m *MockPromptManager) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	args := m.Called(ctx, layer, artisanID, sessionID)
	return args.Error(0)
}

func (m *MockPromptManager) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	args := m.Called(ctx, artisanID, sessionID)
	return args.Get(0).([]layered.SystemPrompt), args.Error(1)
}

func (m *MockPromptManager) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	args := m.Called(ctx, artisanID, sessionID)
	return args.Error(0)
}

// MockAgentRegistry for testing - implements AgentRegistry interface
type MockAgentRegistry struct {
	mock.Mock
}

func (m *MockAgentRegistry) GetAgent(agentID string) (registry.Agent, error) {
	args := m.Called(agentID)
	return args.Get(0).(registry.Agent), args.Error(1)
}

func (m *MockAgentRegistry) ListAgents() []registry.Agent {
	args := m.Called()
	return args.Get(0).([]registry.Agent)
}

func (m *MockAgentRegistry) RegisterAgent(agent registry.Agent) error {
	args := m.Called(agent)
	return args.Error(0)
}

func (m *MockAgentRegistry) GetAgentsByCapability(capability string) []registry.AgentInfo {
	args := m.Called(capability)
	return args.Get(0).([]registry.AgentInfo)
}

// Add other required methods to satisfy AgentRegistry interface
func (m *MockAgentRegistry) RegisterAgentType(name string, factory registry.AgentFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockAgentRegistry) ListAgentTypes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockAgentRegistry) HasAgentType(agentType string) bool {
	args := m.Called(agentType)
	return args.Bool(0)
}

func (m *MockAgentRegistry) GetAgentsByCost(maxCost int) []registry.AgentInfo {
	args := m.Called(maxCost)
	return args.Get(0).([]registry.AgentInfo)
}

func (m *MockAgentRegistry) GetCheapestAgentByCapability(capability string) (*registry.AgentInfo, error) {
	args := m.Called(capability)
	return args.Get(0).(*registry.AgentInfo), args.Error(1)
}

func (m *MockAgentRegistry) RegisterGuildAgent(config registry.GuildAgentConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockAgentRegistry) GetRegisteredAgents() []registry.GuildAgentConfig {
	args := m.Called()
	return args.Get(0).([]registry.GuildAgentConfig)
}

func TestManagerIntelligenceService_AnalyzeComplexity(t *testing.T) {
	// Setup mocks
	mockArtisan := &MockArtisanClient{}
	mockPromptMgr := &MockPromptManager{}
	mockAgentReg := &MockAgentRegistry{}

	// Create service
	service := NewManagerIntelligenceService(mockPromptMgr, mockArtisan, mockAgentReg)

	// Mock complexity analysis response
	complexityResponse := &ArtisanResponse{
		Content: `{
			"complexity_score": 6,
			"recommended_approach": "multi-agent",
			"reasoning": "Task involves both frontend and backend work requiring specialized knowledge",
			"agent_requirements": [
				{
					"role": "Backend Developer",
					"priority": "high",
					"estimated_tokens": 1500,
					"rationale": "API development requires backend expertise"
				},
				{
					"role": "Frontend Developer",
					"priority": "high",
					"estimated_tokens": 1200,
					"rationale": "UI components need frontend specialization"
				}
			],
			"execution_strategy": {
				"parallel_tasks": ["API development", "UI component creation"],
				"sequential_tasks": ["Integration testing"],
				"dependencies": [
					{"from": "API development", "to": "Integration testing"}
				]
			},
			"cost_estimate": {
				"single_agent_tokens": 4000,
				"multi_agent_tokens": 2700,
				"recommended_savings": "32%"
			},
			"quality_assurance": {
				"review_points": ["API contract review", "UI/UX review"],
				"testing_strategy": "Component and integration testing",
				"risk_mitigation": ["API versioning", "Cross-browser testing"]
			}
		}`,
	}

	// Setup mock expectations
	// Add GetRegisteredAgents expectation
	mockAgentReg.On("GetRegisteredAgents").Return([]registry.GuildAgentConfig{
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
	})

	mockPromptMgr.On("BuildLayeredPrompt", mock.Anything, "manager-agent", "analysis-session", mock.AnythingOfType("layered.TurnContext")).Return(&layered.LayeredPrompt{
		Compiled:   "You are a Guild Master analyzing task complexity... Analyze this task: Build a user authentication system",
		TokenCount: 150,
		Truncated:  false,
		CacheKey:   "test-cache-key",
		ArtisanID:  "manager-agent",
		SessionID:  "analysis-session",
		AssembledAt: time.Now(),
	}, nil)

	mockArtisan.On("Complete", mock.Anything, mock.Anything).Return(complexityResponse, nil)

	// Test complexity analysis
	ctx := context.Background()
	request := ComplexityAnalysisRequest{
		TaskDescription: "Build a user authentication system with JWT tokens",
		TaskDomain:      "web-development",
		TaskPriority:    "high",
		TokenBudget:     5000,
		QualityLevel:    "production",
		RiskTolerance:   "medium",
		TimeConstraint:  "1 week",
	}

	result, err := service.AnalyzeComplexityOnly(ctx, request)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 6, result.ComplexityScore)
	assert.Equal(t, "multi-agent", result.RecommendedApproach)
	assert.Len(t, result.AgentRequirements, 2)
	assert.Equal(t, "Backend Developer", result.AgentRequirements[0].Role)
	assert.Equal(t, "Frontend Developer", result.AgentRequirements[1].Role)
	assert.Equal(t, "32%", result.CostEstimate.RecommendedSavings)

	// Verify mocks were called
	mockPromptMgr.AssertExpectations(t)
	mockArtisan.AssertExpectations(t)
}

func TestManagerIntelligenceService_RouteToAgents(t *testing.T) {
	// Setup mocks
	mockArtisan := &MockArtisanClient{}
	mockPromptMgr := &MockPromptManager{}
	mockAgentReg := &MockAgentRegistry{}

	// Create service
	service := NewManagerIntelligenceService(mockPromptMgr, mockArtisan, mockAgentReg)

	// Mock routing response
	routingResponse := &ArtisanResponse{
		Content: `{
			"routing_decision": {
				"primary_agent": {
					"agent_id": "backend-artisan",
					"role": "Backend Developer",
					"assignment_confidence": 9,
					"estimated_tokens": 1500,
					"rationale": "Optimal match for API development with high context window"
				},
				"supporting_agents": [
					{
						"agent_id": "frontend-artisan",
						"role": "Frontend Developer",
						"responsibility": "UI components",
						"coordination_type": "parallel",
						"estimated_tokens": 1200
					}
				]
			},
			"cost_analysis": {
				"total_estimated_tokens": 2700,
				"cost_breakdown": [
					{"agent": "backend-artisan", "tokens": 1500, "cost": "$0.023"},
					{"agent": "frontend-artisan", "tokens": 1200, "cost": "$0.036"}
				],
				"alternative_approaches": [
					{
						"approach": "Single architect agent",
						"agents": ["architect-artisan"],
						"tokens": 4000,
						"cost_difference": "+48%",
						"quality_trade_off": "Higher architecture quality, slower execution"
					}
				]
			},
			"execution_plan": {
				"coordination_strategy": "Parallel development with integration checkpoints",
				"task_distribution": [
					{
						"agent": "backend-artisan",
						"tasks": ["API design", "Authentication logic", "Database schema"],
						"dependencies": [],
						"deliverables": ["API endpoints", "Database migrations"]
					},
					{
						"agent": "frontend-artisan",
						"tasks": ["Login components", "User dashboard", "Auth flow"],
						"dependencies": ["API endpoints"],
						"deliverables": ["React components", "UI tests"]
					}
				],
				"quality_gates": [
					{
						"checkpoint": "API contract review",
						"reviewer": "architect-artisan",
						"criteria": ["REST compliance", "Security standards"]
					}
				],
				"risk_mitigation": [
					{
						"risk": "API breaking changes",
						"mitigation": "Versioned API contracts",
						"fallback": "Rollback to previous version"
					}
				]
			},
			"success_metrics": {
				"completion_criteria": ["All tests passing", "Security audit passed"],
				"quality_thresholds": {
					"minimum_quality": 8,
					"target_quality": 9
				},
				"performance_indicators": ["Response time < 200ms", "UI load time < 1s"]
			}
		}`,
	}

	// Setup mock expectations
	// Add GetRegisteredAgents expectation (not used directly in routing but needed by the service)
	mockAgentReg.On("GetRegisteredAgents").Return([]registry.GuildAgentConfig{})

	mockPromptMgr.On("BuildLayeredPrompt", mock.Anything, "manager-agent", "routing-session", mock.AnythingOfType("layered.TurnContext")).Return(&layered.LayeredPrompt{
		Compiled:   "You are a Guild Master routing tasks to agents... Route this task to optimal agents...",
		TokenCount: 200,
		Truncated:  false,
		CacheKey:   "routing-cache-key",
		ArtisanID:  "manager-agent",
		SessionID:  "routing-session",
		AssembledAt: time.Now(),
	}, nil)

	mockArtisan.On("Complete", mock.Anything, mock.Anything).Return(routingResponse, nil)

	// Test agent routing
	ctx := context.Background()
	request := RoutingRequest{
		TaskDescription:     "Build user authentication system",
		ComplexityScore:     6,
		RecommendedApproach: "multi-agent",
		AgentRequirements: []AgentRequirement{
			{Role: "Backend Developer", Priority: "high"},
			{Role: "Frontend Developer", Priority: "high"},
		},
		AvailableAgents: []AgentInfo{
			{
				Name:            "backend-artisan",
				Role:            "Backend Developer",
				Provider:        "anthropic",
				CostMagnitude:   3,
				ContextWindow:   200000,
				Specializations: []string{"Go", "APIs"},
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
		},
	}

	result, err := service.RouteToAgentsOnly(ctx, request)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend-artisan", result.RoutingDecision.PrimaryAgent.AgentID)
	assert.Equal(t, 9, result.RoutingDecision.PrimaryAgent.AssignmentConfidence)
	assert.Len(t, result.RoutingDecision.SupportingAgents, 1)
	assert.Equal(t, "frontend-artisan", result.RoutingDecision.SupportingAgents[0].AgentID)
	assert.Equal(t, 2700, result.CostAnalysis.TotalEstimatedTokens)
	assert.Len(t, result.ExecutionPlan.TaskDistribution, 2)

	// Verify mocks were called
	mockPromptMgr.AssertExpectations(t)
	mockArtisan.AssertExpectations(t)
}

func TestManagerIntelligenceService_FullAnalysisAndRouting(t *testing.T) {
	// This test demonstrates the complete end-to-end workflow
	mockArtisan := &MockArtisanClient{}
	mockPromptMgr := &MockPromptManager{}
	mockAgentReg := &MockAgentRegistry{}

	service := NewManagerIntelligenceService(mockPromptMgr, mockArtisan, mockAgentReg)

	// Mock both complexity analysis and routing responses
	complexityResponse := &ArtisanResponse{
		Content: `{
			"complexity_score": 4,
			"recommended_approach": "single-agent",
			"reasoning": "Simple CRUD operations can be handled by one backend specialist",
			"agent_requirements": [
				{"role": "Backend Developer", "priority": "high", "estimated_tokens": 2000, "rationale": "Database operations and API endpoints"}
			],
			"execution_strategy": {"parallel_tasks": [], "sequential_tasks": ["Design schema", "Implement CRUD", "Add validation"]},
			"cost_estimate": {"single_agent_tokens": 2000, "multi_agent_tokens": 2500, "recommended_savings": "-25%"},
			"quality_assurance": {"review_points": ["Schema review"], "testing_strategy": "Unit testing", "risk_mitigation": ["Data validation"]}
		}`,
	}

	routingResponse := &ArtisanResponse{
		Content: `{
			"routing_decision": {
				"primary_agent": {"agent_id": "backend-artisan", "role": "Backend Developer", "assignment_confidence": 10, "estimated_tokens": 2000, "rationale": "Perfect match for backend work"},
				"supporting_agents": []
			},
			"cost_analysis": {"total_estimated_tokens": 2000, "cost_breakdown": [{"agent": "backend-artisan", "tokens": 2000, "cost": "$0.030"}], "alternative_approaches": []},
			"execution_plan": {"coordination_strategy": "Single agent execution", "task_distribution": [{"agent": "backend-artisan", "tasks": ["CRUD implementation"], "dependencies": [], "deliverables": ["API endpoints"]}], "quality_gates": [], "risk_mitigation": []},
			"success_metrics": {"completion_criteria": ["Tests passing"], "quality_thresholds": {"minimum_quality": 8, "target_quality": 9}, "performance_indicators": ["API response time"]}
		}`,
	}

	// Setup expectations for both calls
	// Add GetRegisteredAgents expectation
	mockAgentReg.On("GetRegisteredAgents").Return([]registry.GuildAgentConfig{
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
	})

	mockPromptMgr.On("BuildLayeredPrompt", mock.Anything, "manager-agent", "analysis-session", mock.AnythingOfType("layered.TurnContext")).Return(&layered.LayeredPrompt{
		Compiled:   "Complexity analysis prompt... Analyze complexity...",
		TokenCount: 150,
		Truncated:  false,
		CacheKey:   "complexity-cache-key",
		ArtisanID:  "manager-agent",
		SessionID:  "analysis-session",
		AssembledAt: time.Now(),
	}, nil).Once()

	mockPromptMgr.On("BuildLayeredPrompt", mock.Anything, "manager-agent", "routing-session", mock.AnythingOfType("layered.TurnContext")).Return(&layered.LayeredPrompt{
		Compiled:   "Routing prompt... Route to agents...",
		TokenCount: 200,
		Truncated:  false,
		CacheKey:   "routing-cache-key",
		ArtisanID:  "manager-agent",
		SessionID:  "routing-session",
		AssembledAt: time.Now(),
	}, nil).Once()

	mockArtisan.On("Complete", mock.Anything, mock.MatchedBy(func(req ArtisanRequest) bool {
		// Complexity analysis call (first)
		return req.MaxTokens == 2000
	})).Return(complexityResponse, nil).Once()

	mockArtisan.On("Complete", mock.Anything, mock.MatchedBy(func(req ArtisanRequest) bool {
		// Routing call (second)
		return req.MaxTokens == 3000
	})).Return(routingResponse, nil).Once()

	// Test full analysis and routing
	ctx := context.Background()
	request := IntelligenceRequest{
		TaskDescription: "Create a simple user CRUD API",
		TaskDomain:      "backend",
		TaskPriority:    "medium",
		TokenBudget:     3000,
		QualityLevel:    "development",
		RiskTolerance:   "low",
		TimeConstraint:  "2 days",
	}

	result, err := service.AnalyzeAndRoute(ctx, request)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Check complexity analysis
	assert.Equal(t, 4, result.ComplexityAnalysis.ComplexityScore)
	assert.Equal(t, "single-agent", result.ComplexityAnalysis.RecommendedApproach)

	// Check routing
	assert.Equal(t, "backend-artisan", result.AgentRouting.RoutingDecision.PrimaryAgent.AgentID)
	assert.Equal(t, 10, result.AgentRouting.RoutingDecision.PrimaryAgent.AssignmentConfidence)

	// Check executive summary
	assert.Equal(t, "single-agent", result.ExecutiveSummary.RecommendedApproach)
	assert.Equal(t, "backend-artisan", result.ExecutiveSummary.PrimaryAgent)
	assert.Empty(t, result.ExecutiveSummary.SupportingAgents)
	assert.Contains(t, result.ExecutiveSummary.EstimatedCost, "2000 tokens")

	// Verify all mocks were called correctly
	mockPromptMgr.AssertExpectations(t)
	mockArtisan.AssertExpectations(t)
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid JSON in markdown",
			input:    "Here's the analysis:\n\n```json\n{\"complexity_score\": 5, \"approach\": \"multi-agent\"}\n```\n\nThat's the result.",
			expected: "\n{\"complexity_score\": 5, \"approach\": \"multi-agent\"}\n",
		},
		{
			name:     "No JSON block",
			input:    "Just some text without JSON",
			expected: "",
		},
		{
			name:     "Multiple code blocks",
			input:    "First block:\n```json\n{\"first\": \"block\"}\n```\n\nSecond block:\n```json\n{\"second\": \"block\"}\n```",
			expected: "\n{\"first\": \"block\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test demonstrating real usage patterns
func TestManagerIntelligenceService_UsagePatterns(t *testing.T) {
	t.Run("Simple task should recommend single agent", func(t *testing.T) {
		// Test that simple tasks get routed to single agents
		// This would be a real integration test in practice
	})

	t.Run("Complex task should recommend multi-agent", func(t *testing.T) {
		// Test that complex tasks get multi-agent recommendations
		// This would verify the complexity scoring works correctly
	})

	t.Run("Cost optimization should prefer cheaper agents for simple tasks", func(t *testing.T) {
		// Test that cost-conscious routing works
		// Should prefer Ollama for simple tasks, Claude for complex ones
	})
}
