package manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// AgentRouter routes tasks to optimal agents using manager agent and layered prompts
type AgentRouter struct {
	promptManager prompts.LayeredManager
	artisanClient ArtisanClient
	agentRegistry registry.AgentRegistry
}

// NewAgentRouter creates a new agent router
func NewAgentRouter(
	promptManager prompts.LayeredManager,
	artisanClient ArtisanClient,
	agentRegistry registry.AgentRegistry,
) *AgentRouter {
	return &AgentRouter{
		promptManager: promptManager,
		artisanClient: artisanClient,
		agentRegistry: agentRegistry,
	}
}

// RoutingRequest represents a request for agent assignment
type RoutingRequest struct {
	TaskDescription     string
	ComplexityScore     int
	RecommendedApproach string
	AgentRequirements   []AgentRequirement
	AvailableAgents     []AgentInfo
}

// RoutingResult represents the result of agent routing
type RoutingResult struct {
	RoutingDecision RoutingDecision `json:"routing_decision"`
	CostAnalysis    CostAnalysis    `json:"cost_analysis"`
	ExecutionPlan   ExecutionPlan   `json:"execution_plan"`
	SuccessMetrics  SuccessMetrics  `json:"success_metrics"`
}

// RoutingDecision specifies which agents to assign
type RoutingDecision struct {
	PrimaryAgent     AgentAssignment   `json:"primary_agent"`
	SupportingAgents []AgentAssignment `json:"supporting_agents"`
}

// AgentAssignment represents a specific agent assignment
type AgentAssignment struct {
	AgentID              string `json:"agent_id"`
	Role                 string `json:"role"`
	AssignmentConfidence int    `json:"assignment_confidence"`
	EstimatedTokens      int    `json:"estimated_tokens"`
	Rationale            string `json:"rationale"`
	Responsibility       string `json:"responsibility,omitempty"`
	CoordinationType     string `json:"coordination_type,omitempty"`
}

// CostAnalysis provides detailed cost breakdown and alternatives
type CostAnalysis struct {
	TotalEstimatedTokens   int                     `json:"total_estimated_tokens"`
	CostBreakdown          []AgentCostBreakdown    `json:"cost_breakdown"`
	AlternativeApproaches  []AlternativeApproach   `json:"alternative_approaches"`
}

// AgentCostBreakdown shows cost per agent
type AgentCostBreakdown struct {
	Agent  string `json:"agent"`
	Tokens int    `json:"tokens"`
	Cost   string `json:"cost"`
}

// AlternativeApproach suggests different agent configurations
type AlternativeApproach struct {
	Approach        string   `json:"approach"`
	Agents          []string `json:"agents"`
	Tokens          int      `json:"tokens"`
	CostDifference  string   `json:"cost_difference"`
	QualityTradeOff string   `json:"quality_trade_off"`
}

// ExecutionPlan defines how agents will coordinate
type ExecutionPlan struct {
	CoordinationStrategy string               `json:"coordination_strategy"`
	TaskDistribution     []TaskDistribution   `json:"task_distribution"`
	QualityGates         []QualityGate        `json:"quality_gates"`
	RiskMitigation       []RiskMitigation     `json:"risk_mitigation"`
}

// TaskDistribution specifies tasks for each agent
type TaskDistribution struct {
	Agent        string   `json:"agent"`
	Tasks        []string `json:"tasks"`
	Dependencies []string `json:"dependencies"`
	Deliverables []string `json:"deliverables"`
}

// QualityGate defines review checkpoints
type QualityGate struct {
	Checkpoint string   `json:"checkpoint"`
	Reviewer   string   `json:"reviewer"`
	Criteria   []string `json:"criteria"`
}

// RiskMitigation defines risk management strategies
type RiskMitigation struct {
	Risk       string `json:"risk"`
	Mitigation string `json:"mitigation"`
	Fallback   string `json:"fallback"`
}

// SuccessMetrics defines how to measure execution quality
type SuccessMetrics struct {
	CompletionCriteria     []string          `json:"completion_criteria"`
	QualityThresholds      QualityThresholds `json:"quality_thresholds"`
	PerformanceIndicators  []string          `json:"performance_indicators"`
}

// QualityThresholds defines quality score expectations
type QualityThresholds struct {
	MinimumQuality int `json:"minimum_quality"`
	TargetQuality  int `json:"target_quality"`
}

// RouteToAgents determines optimal agent assignments for a task
func (ar *AgentRouter) RouteToAgents(
	ctx context.Context,
	request RoutingRequest,
) (*RoutingResult, error) {
	// Enhance agent info with current status and performance metrics
	enhancedAgents, err := ar.enhanceAgentInfo(ctx, request.AvailableAgents)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance agent info: %w", err)
	}

	// Build context for routing prompt
	promptContext := map[string]interface{}{
		"TaskDescription":     request.TaskDescription,
		"ComplexityScore":     request.ComplexityScore,
		"RecommendedApproach": request.RecommendedApproach,
		"AgentRequirements":   request.AgentRequirements,
		"AvailableAgents":     enhancedAgents,
	}

	// Generate layered prompt for agent routing
	turnCtx := prompts.TurnContext{
		UserMessage:  request.TaskDescription,
		TaskID:       "agent-routing",
		CommissionID: "task-routing",
		Urgency:      "medium",
		Instructions: []string{"Route task to optimal agents based on capabilities and cost"},
		Metadata:     promptContext,
	}
	
	layeredPrompt, err := ar.promptManager.BuildLayeredPrompt(ctx, "manager-agent", "routing-session", turnCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble routing prompt: %w", err)
	}

	// Make request to artisan (manager agent)
	artisanRequest := ArtisanRequest{
		SystemPrompt: layeredPrompt.Compiled,
		UserPrompt:   turnCtx.UserMessage,
		Temperature:  0.2, // Very low temperature for routing decisions
		MaxTokens:    3000,
	}

	response, err := ar.artisanClient.Complete(ctx, artisanRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing decision from artisan: %w", err)
	}

	// Parse the JSON response
	var result RoutingResult
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		// If JSON parsing fails, try to extract JSON from markdown code blocks
		if extractedJSON := extractJSONFromMarkdown(response.Content); extractedJSON != "" {
			if err := json.Unmarshal([]byte(extractedJSON), &result); err != nil {
				return nil, fmt.Errorf("failed to parse routing response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse routing response: %w", err)
		}
	}

	return &result, nil
}

// enhanceAgentInfo adds current status and performance data to agent info
func (ar *AgentRouter) enhanceAgentInfo(ctx context.Context, agents []AgentInfo) ([]EnhancedAgentInfo, error) {
	var enhanced []EnhancedAgentInfo
	
	for _, agent := range agents {
		// TODO: Get real agent status from registry
		enhancedAgent := EnhancedAgentInfo{
			AgentInfo:          agent,
			TokenCost:          ar.calculateTokenCost(agent),
			QualityScore:       ar.calculateQualityScore(agent),
			AvgCompletionTime:  "2.5 minutes", // TODO: Get from metrics
			RecentTaskCount:    5,             // TODO: Get from task history
			CurrentTasks:       2,             // TODO: Get from active tasks
			AvailabilityStatus: "available",   // TODO: Get from agent status
		}
		enhanced = append(enhanced, enhancedAgent)
	}
	
	return enhanced, nil
}

// EnhancedAgentInfo includes runtime status and performance data
type EnhancedAgentInfo struct {
	AgentInfo          `json:",inline"`
	TokenCost          float64 `json:"token_cost"`
	QualityScore       float64 `json:"quality_score"`
	AvgCompletionTime  string  `json:"avg_completion_time"`
	RecentTaskCount    int     `json:"recent_task_count"`
	CurrentTasks       int     `json:"current_tasks"`
	AvailabilityStatus string  `json:"availability_status"`
}

// calculateTokenCost estimates cost per 1K tokens for the agent
func (ar *AgentRouter) calculateTokenCost(agent AgentInfo) float64 {
	// Provider-based cost estimation (simplified)
	switch agent.Provider {
	case "anthropic":
		return 0.015 // Claude 3.5 Sonnet approximate cost
	case "openai":
		return 0.03 // GPT-4 approximate cost
	case "ollama":
		return 0.0 // Local models are free
	case "deepseek":
		return 0.002 // DeepSeek competitive pricing
	default:
		return 0.01 // Default estimate
	}
}

// calculateQualityScore estimates agent quality based on success rate and specializations
func (ar *AgentRouter) calculateQualityScore(agent AgentInfo) float64 {
	// Base score from success rate
	baseScore := agent.SuccessRate / 10.0 // Convert percentage to 0-10 scale
	
	// Bonus for specialization depth (more specializations = higher versatility)
	specializationBonus := float64(len(agent.Specializations)) * 0.1
	if specializationBonus > 1.0 {
		specializationBonus = 1.0 // Cap at +1 point
	}
	
	// Context window bonus (larger windows handle complex tasks better)
	contextBonus := 0.0
	if agent.ContextWindow > 100000 {
		contextBonus = 0.5
	} else if agent.ContextWindow > 32000 {
		contextBonus = 0.2
	}
	
	totalScore := baseScore + specializationBonus + contextBonus
	if totalScore > 10.0 {
		totalScore = 10.0
	}
	
	return totalScore
}

// GetAgentCapabilities returns detailed capabilities for all available agents
func (ar *AgentRouter) GetAgentCapabilities(ctx context.Context) ([]EnhancedAgentInfo, error) {
	// TODO: Integrate with actual agent registry to get real agents
	// For now, return the mock data with enhancements
	mockAgents := []AgentInfo{
		{
			Name:            "backend-artisan",
			Role:            "Backend Developer",
			Provider:        "anthropic",
			Model:           "claude-3-5-sonnet-20241022",
			CostMagnitude:   3,
			ContextWindow:   200000,
			Specializations: []string{"Go", "APIs", "databases", "microservices"},
			Tools:           []string{"file", "shell", "http"},
			SuccessRate:     92.5,
		},
		{
			Name:            "frontend-artisan",
			Role:            "Frontend Developer",
			Provider:        "openai",
			Model:           "gpt-4",
			CostMagnitude:   2,
			ContextWindow:   128000,
			Specializations: []string{"React", "TypeScript", "CSS", "UI/UX"},
			Tools:           []string{"file", "shell", "http"},
			SuccessRate:     89.2,
		},
	}
	
	return ar.enhanceAgentInfo(ctx, mockAgents)
}