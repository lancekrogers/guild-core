// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"fmt"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/prompts/layered"
)

// ManagerIntelligenceService provides intelligent task analysis and agent routing
// using the manager agent with layered prompts for decision making
type ManagerIntelligenceService struct {
	complexityAnalyzer *TaskComplexityAnalyzer
	agentRouter        *AgentRouter
	promptManager      layered.LayeredManager
	artisanClient      ArtisanClient
	agentRegistry      core.AgentRegistry
}

// NewManagerIntelligenceService creates a new manager intelligence service
func NewManagerIntelligenceService(
	promptManager layered.LayeredManager,
	artisanClient ArtisanClient,
	agentRegistry core.AgentRegistry,
) *ManagerIntelligenceService {
	return &ManagerIntelligenceService{
		complexityAnalyzer: NewTaskComplexityAnalyzer(promptManager, artisanClient, agentRegistry),
		agentRouter:        NewAgentRouter(promptManager, artisanClient, agentRegistry),
		promptManager:      promptManager,
		artisanClient:      artisanClient,
		agentRegistry:      agentRegistry,
	}
}

// IntelligenceRequest represents a complete request for task analysis and routing
type IntelligenceRequest struct {
	TaskDescription string
	TaskDomain      string
	TaskPriority    string
	TokenBudget     int
	QualityLevel    string
	RiskTolerance   string
	TimeConstraint  string
}

// IntelligenceResult provides complete analysis and routing recommendations
type IntelligenceResult struct {
	// Complexity Analysis Results
	ComplexityAnalysis *ComplexityAnalysisResult `json:"complexity_analysis"`

	// Agent Routing Results
	AgentRouting *RoutingResult `json:"agent_routing"`

	// Executive Summary
	ExecutiveSummary ExecutiveSummary `json:"executive_summary"`
}

// ExecutiveSummary provides high-level recommendations
type ExecutiveSummary struct {
	RecommendedApproach string              `json:"recommended_approach"`
	PrimaryAgent        string              `json:"primary_agent"`
	SupportingAgents    []string            `json:"supporting_agents"`
	EstimatedCost       string              `json:"estimated_cost"`
	EstimatedDuration   string              `json:"estimated_duration"`
	ConfidenceLevel     string              `json:"confidence_level"`
	KeyRisks            []string            `json:"key_risks"`
	SuccessFactors      []string            `json:"success_factors"`
	AlternativeOptions  []AlternativeOption `json:"alternative_options"`
}

// AlternativeOption represents different execution strategies
type AlternativeOption struct {
	Strategy     string `json:"strategy"`
	Description  string `json:"description"`
	CostDelta    string `json:"cost_delta"`
	TimeDelta    string `json:"time_delta"`
	QualityDelta string `json:"quality_delta"`
}

// AnalyzeAndRoute performs complete task analysis and agent routing
func (mis *ManagerIntelligenceService) AnalyzeAndRoute(
	ctx context.Context,
	request IntelligenceRequest,
) (*IntelligenceResult, error) {
	// Step 1: Analyze task complexity
	complexityRequest := ComplexityAnalysisRequest{
		TaskDescription: request.TaskDescription,
		TaskDomain:      request.TaskDomain,
		TaskPriority:    request.TaskPriority,
		TokenBudget:     request.TokenBudget,
		QualityLevel:    request.QualityLevel,
		RiskTolerance:   request.RiskTolerance,
		TimeConstraint:  request.TimeConstraint,
	}

	complexityResult, err := mis.complexityAnalyzer.AnalyzeComplexity(ctx, complexityRequest)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to analyze task complexity").
			WithComponent("manager").
			WithOperation("AnalyzeAndRoute").
			WithDetails("task_domain", request.TaskDomain)
	}

	// Step 2: Get available agents
	availableAgents, err := mis.getAvailableAgentInfo(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to get available agents").
			WithComponent("manager").
			WithOperation("AnalyzeAndRoute")
	}

	// Step 3: Route to optimal agents
	routingRequest := RoutingRequest{
		TaskDescription:     request.TaskDescription,
		ComplexityScore:     complexityResult.ComplexityScore,
		RecommendedApproach: complexityResult.RecommendedApproach,
		AgentRequirements:   complexityResult.AgentRequirements,
		AvailableAgents:     availableAgents,
	}

	routingResult, err := mis.agentRouter.RouteToAgents(ctx, routingRequest)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to route to agents").
			WithComponent("manager").
			WithOperation("AnalyzeAndRoute").
			WithDetails("complexity_score", complexityResult.ComplexityScore)
	}

	// Step 4: Generate executive summary
	executiveSummary := mis.generateExecutiveSummary(complexityResult, routingResult)

	return &IntelligenceResult{
		ComplexityAnalysis: complexityResult,
		AgentRouting:       routingResult,
		ExecutiveSummary:   executiveSummary,
	}, nil
}

// AnalyzeComplexityOnly performs just complexity analysis
func (mis *ManagerIntelligenceService) AnalyzeComplexityOnly(
	ctx context.Context,
	request ComplexityAnalysisRequest,
) (*ComplexityAnalysisResult, error) {
	return mis.complexityAnalyzer.AnalyzeComplexity(ctx, request)
}

// RouteToAgentsOnly performs just agent routing
func (mis *ManagerIntelligenceService) RouteToAgentsOnly(
	ctx context.Context,
	request RoutingRequest,
) (*RoutingResult, error) {
	return mis.agentRouter.RouteToAgents(ctx, request)
}

// GetAvailableAgents returns enhanced information about available agents
func (mis *ManagerIntelligenceService) GetAvailableAgents(ctx context.Context) ([]EnhancedAgentInfo, error) {
	return mis.agentRouter.GetAgentCapabilities(ctx)
}

// getAvailableAgentInfo retrieves basic agent info for routing
func (mis *ManagerIntelligenceService) getAvailableAgentInfo(ctx context.Context) ([]AgentInfo, error) {
	// TODO: This should integrate with the actual agent registry
	// For now, return consistent mock data
	return []AgentInfo{
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
		{
			Name:            "devops-artisan",
			Role:            "DevOps Engineer",
			Provider:        "ollama",
			Model:           "llama3.1",
			CostMagnitude:   1,
			ContextWindow:   32000,
			Specializations: []string{"Docker", "Kubernetes", "CI/CD", "monitoring"},
			Tools:           []string{"shell", "file", "http"},
			SuccessRate:     87.8,
		},
		{
			Name:            "architect-artisan",
			Role:            "System Architect",
			Provider:        "anthropic",
			Model:           "claude-3-5-sonnet-20241022",
			CostMagnitude:   5,
			ContextWindow:   200000,
			Specializations: []string{"system design", "architecture", "patterns", "scalability"},
			Tools:           []string{"file", "corpus"},
			SuccessRate:     95.1,
		},
	}, nil
}

// generateExecutiveSummary creates a high-level summary of recommendations
func (mis *ManagerIntelligenceService) generateExecutiveSummary(
	complexity *ComplexityAnalysisResult,
	routing *RoutingResult,
) ExecutiveSummary {
	// Extract supporting agents
	var supportingAgents []string
	for _, agent := range routing.RoutingDecision.SupportingAgents {
		supportingAgents = append(supportingAgents, agent.AgentID)
	}

	// Calculate confidence based on complexity and agent match
	confidenceLevel := "High"
	if complexity.ComplexityScore > 7 {
		confidenceLevel = "Medium"
	}
	if len(routing.RoutingDecision.SupportingAgents) > 2 {
		confidenceLevel = "Medium"
	}

	// Extract key risks from complexity analysis
	keyRisks := complexity.QualityAssurance.RiskMitigation
	if len(keyRisks) == 0 {
		keyRisks = []string{"Standard development risks"}
	}

	// Determine success factors
	successFactors := []string{
		"Clear task decomposition",
		"Agent specialization alignment",
		"Defined quality gates",
	}
	if complexity.RecommendedApproach == "multi-agent" {
		successFactors = append(successFactors, "Effective agent coordination")
	}

	return ExecutiveSummary{
		RecommendedApproach: complexity.RecommendedApproach,
		PrimaryAgent:        routing.RoutingDecision.PrimaryAgent.AgentID,
		SupportingAgents:    supportingAgents,
		EstimatedCost:       mis.formatTokenCost(routing.CostAnalysis.TotalEstimatedTokens),
		EstimatedDuration:   mis.estimateDuration(complexity.ComplexityScore),
		ConfidenceLevel:     confidenceLevel,
		KeyRisks:            keyRisks,
		SuccessFactors:      successFactors,
		AlternativeOptions:  mis.generateAlternatives(routing.CostAnalysis.AlternativeApproaches),
	}
}

// formatTokenCost converts token count to cost estimate
func (mis *ManagerIntelligenceService) formatTokenCost(tokens int) string {
	// Average cost estimate across providers
	avgCostPer1K := 0.01
	totalCost := float64(tokens) / 1000.0 * avgCostPer1K
	return fmt.Sprintf("~$%.3f (%d tokens)", totalCost, tokens)
}

// estimateDuration provides time estimate based on complexity
func (mis *ManagerIntelligenceService) estimateDuration(complexity int) string {
	switch {
	case complexity <= 3:
		return "15-30 minutes"
	case complexity <= 6:
		return "1-2 hours"
	case complexity <= 8:
		return "4-8 hours"
	default:
		return "1-2 days"
	}
}

// generateAlternatives converts routing alternatives to executive format
func (mis *ManagerIntelligenceService) generateAlternatives(alternatives []AlternativeApproach) []AlternativeOption {
	var options []AlternativeOption

	for _, alt := range alternatives {
		option := AlternativeOption{
			Strategy:     alt.Approach,
			Description:  fmt.Sprintf("Use %v", alt.Agents),
			CostDelta:    alt.CostDifference,
			TimeDelta:    "Similar", // Could be enhanced with actual time analysis
			QualityDelta: alt.QualityTradeOff,
		}
		options = append(options, option)
	}

	return options
}
