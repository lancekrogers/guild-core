// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/prompts/layered"
)

// TaskComplexityAnalyzer analyzes task complexity with proper error handling,
// validation, observability, and registry integration
type TaskComplexityAnalyzer struct {
	promptManager layered.LayeredManager
	artisanClient ArtisanClient
	agentRegistry core.AgentRegistry
	logger        *slog.Logger
	timeout       time.Duration
}

// ComplexityAnalysisRequest represents a request for task complexity analysis
type ComplexityAnalysisRequest struct {
	TaskDescription string
	TaskDomain      string
	TaskPriority    string
	TaskDeadline    *time.Time
	TokenBudget     int
	QualityLevel    string
	RiskTolerance   string
	TimeConstraint  string
}

// ComplexityAnalysisResult represents the result of complexity analysis
type ComplexityAnalysisResult struct {
	ComplexityScore     int                `json:"complexity_score"`
	RecommendedApproach string             `json:"recommended_approach"`
	Reasoning           string             `json:"reasoning"`
	AgentRequirements   []AgentRequirement `json:"agent_requirements"`
	ExecutionStrategy   ExecutionStrategy  `json:"execution_strategy"`
	CostEstimate        CostEstimate       `json:"cost_estimate"`
	QualityAssurance    QualityAssurance   `json:"quality_assurance"`
}

// AgentRequirement specifies what type of agent is needed
type AgentRequirement struct {
	Role            string `json:"role"`
	Priority        string `json:"priority"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Rationale       string `json:"rationale"`
}

// ExecutionStrategy defines how the task should be executed
type ExecutionStrategy struct {
	ParallelTasks   []string         `json:"parallel_tasks"`
	SequentialTasks []string         `json:"sequential_tasks"`
	Dependencies    []TaskDependency `json:"dependencies"`
}

// TaskDependency represents a dependency between tasks
type TaskDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CostEstimate provides token usage estimates
type CostEstimate struct {
	SingleAgentTokens  int    `json:"single_agent_tokens"`
	MultiAgentTokens   int    `json:"multi_agent_tokens"`
	RecommendedSavings string `json:"recommended_savings"`
}

// QualityAssurance defines quality control measures
type QualityAssurance struct {
	ReviewPoints    []string `json:"review_points"`
	TestingStrategy string   `json:"testing_strategy"`
	RiskMitigation  []string `json:"risk_mitigation"`
}

// AgentInfo represents information about an available agent
type AgentInfo struct {
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	CostMagnitude   int      `json:"cost_magnitude"`
	ContextWindow   int      `json:"context_window"`
	Specializations []string `json:"specializations"`
	Tools           []string `json:"tools"`
	SuccessRate     float64  `json:"success_rate"`
}

// NewTaskComplexityAnalyzer creates a properly configured analyzer
func NewTaskComplexityAnalyzer(
	promptManager layered.LayeredManager,
	artisanClient ArtisanClient,
	agentRegistry core.AgentRegistry,
) *TaskComplexityAnalyzer {
	return &TaskComplexityAnalyzer{
		promptManager: promptManager,
		artisanClient: artisanClient,
		agentRegistry: agentRegistry,
		logger:        slog.Default(),
		timeout:       30 * time.Second, // Configurable timeout
	}
}

// AnalyzeComplexity analyzes task complexity with proper validation and error handling
func (tca *TaskComplexityAnalyzer) AnalyzeComplexity(
	ctx context.Context,
	request ComplexityAnalysisRequest,
) (*ComplexityAnalysisResult, error) {
	// Input validation
	if err := tca.validateRequest(request); err != nil {
		return nil, NewValidationError("invalid request", err)
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, tca.timeout)
	defer cancel()

	// Log start of analysis
	tca.logger.Info("Starting complexity analysis",
		slog.String("task", request.TaskDescription),
		slog.String("domain", request.TaskDomain),
		slog.Int("token_budget", request.TokenBudget),
	)

	// Get real agents from registry with error handling
	availableAgents, err := tca.getAvailableAgentsFromRegistry(ctx)
	if err != nil {
		tca.logger.Error("Failed to get agents from registry", slog.Any("error", err))
		return nil, NewRegistryError("failed to get available agents", err)
	}

	tca.logger.Debug("Retrieved agents from registry",
		slog.Int("agent_count", len(availableAgents)),
	)

	// Build context with actual agent data
	promptContext := tca.buildPromptContext(request, availableAgents)

	// Generate layered prompt with error handling
	layeredPrompt, err := tca.buildLayeredPrompt(ctx, request, promptContext)
	if err != nil {
		tca.logger.Error("Failed to build layered prompt", slog.Any("error", err))
		return nil, NewPromptError("failed to assemble complexity analysis prompt", err)
	}

	tca.logger.Debug("Built layered prompt",
		slog.Int("token_count", layeredPrompt.TokenCount),
		slog.Bool("truncated", layeredPrompt.Truncated),
	)

	// Make request to artisan with timeout and retries
	result, err := tca.executeAnalysis(ctx, layeredPrompt, request)
	if err != nil {
		tca.logger.Error("Failed to execute analysis", slog.Any("error", err))
		return nil, err
	}

	tca.logger.Info("Complexity analysis completed",
		slog.Int("complexity_score", result.ComplexityScore),
		slog.String("approach", result.RecommendedApproach),
		slog.Int("agent_requirements", len(result.AgentRequirements)),
	)

	return result, nil
}

// validateRequest validates input parameters
func (tca *TaskComplexityAnalyzer) validateRequest(request ComplexityAnalysisRequest) error {
	if request.TaskDescription == "" {
		return gerror.New(gerror.ErrCodeValidation, "task description cannot be empty", nil).
			WithComponent("manager").
			WithOperation("validateRequest")
	}
	if len(request.TaskDescription) > 10000 {
		return gerror.Newf(gerror.ErrCodeValidation, "task description too long (max 10000 chars)").
			WithComponent("manager").
			WithOperation("validateRequest").
			WithDetails("length", len(request.TaskDescription))
	}
	if request.TokenBudget <= 0 {
		return gerror.New(gerror.ErrCodeValidation, "token budget must be positive", nil).
			WithComponent("manager").
			WithOperation("validateRequest").
			WithDetails("token_budget", request.TokenBudget)
	}
	if request.TokenBudget > 1000000 {
		return gerror.Newf(gerror.ErrCodeValidation, "token budget too large (max 1M tokens)").
			WithComponent("manager").
			WithOperation("validateRequest").
			WithDetails("token_budget", request.TokenBudget)
	}
	return nil
}

// getAvailableAgentsFromRegistry gets real agent data from registry
func (tca *TaskComplexityAnalyzer) getAvailableAgentsFromRegistry(ctx context.Context) ([]AgentInfo, error) {
	// Get the component registry to access storage
	componentReg, ok := tca.agentRegistry.(interface{ GetComponentRegistry() interface{} })
	if !ok {
		// If we can't access the full registry, check if we have any registered guild agents
		registeredAgents := tca.agentRegistry.GetRegisteredAgents()
		if len(registeredAgents) == 0 {
			tca.logger.Warn("No agents found in registry")
			return nil, NewRegistryError("no agents available", nil)
		}

		// Convert registered guild agents to AgentInfo
		var agents []AgentInfo
		for _, guildAgent := range registeredAgents {
			agent := AgentInfo{
				Name:            guildAgent.Name,
				Role:            guildAgent.Type,
				Provider:        guildAgent.Provider,
				Model:           guildAgent.Model,
				CostMagnitude:   guildAgent.CostMagnitude,
				ContextWindow:   guildAgent.ContextWindow,
				Specializations: guildAgent.Capabilities,
				Tools:           guildAgent.Tools,
				SuccessRate:     0.85, // Default success rate
			}
			agents = append(agents, agent)
		}
		return agents, nil
	}

	// If we have access to the full registry, try to get agents from the database
	fullRegistry := componentReg.GetComponentRegistry()
	_, ok = fullRegistry.(interface{ Storage() interface{} })
	if !ok {
		// Fall back to registered agents
		registeredAgents := tca.agentRegistry.GetRegisteredAgents()
		if len(registeredAgents) == 0 {
			return nil, NewRegistryError("no agents available", nil)
		}

		// Convert as above
		var agents []AgentInfo
		for _, guildAgent := range registeredAgents {
			agent := AgentInfo{
				Name:            guildAgent.Name,
				Role:            guildAgent.Type,
				Provider:        guildAgent.Provider,
				Model:           guildAgent.Model,
				CostMagnitude:   guildAgent.CostMagnitude,
				ContextWindow:   guildAgent.ContextWindow,
				Specializations: guildAgent.Capabilities,
				Tools:           guildAgent.Tools,
				SuccessRate:     0.85,
			}
			agents = append(agents, agent)
		}
		return agents, nil
	}

	// For now, just return registered agents - SQLite integration is complex
	registeredAgents := tca.agentRegistry.GetRegisteredAgents()
	if len(registeredAgents) == 0 {
		return nil, NewRegistryError("no agents available", nil)
	}

	// Convert to AgentInfo
	var agents []AgentInfo
	for _, guildAgent := range registeredAgents {
		agent := AgentInfo{
			Name:            guildAgent.Name,
			Role:            guildAgent.Type,
			Provider:        guildAgent.Provider,
			Model:           guildAgent.Model,
			CostMagnitude:   guildAgent.CostMagnitude,
			ContextWindow:   guildAgent.ContextWindow,
			Specializations: guildAgent.Capabilities,
			Tools:           guildAgent.Tools,
			SuccessRate:     0.85,
		}
		agents = append(agents, agent)
	}

	tca.logger.Info("Loaded registered agents", slog.Int("count", len(agents)))
	return agents, nil
}

// getDefaultAgents provides fallback agent data
func (tca *TaskComplexityAnalyzer) getDefaultAgents() []AgentInfo {
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
	}
}

// buildPromptContext creates context for prompt generation
func (tca *TaskComplexityAnalyzer) buildPromptContext(
	request ComplexityAnalysisRequest,
	availableAgents []AgentInfo,
) map[string]interface{} {
	return map[string]interface{}{
		"GuildName":       "Current Guild",        // TODO: Get from config
		"ProjectType":     "Software Development", // TODO: Get from project context
		"AgentCount":      len(availableAgents),
		"TokenBudget":     request.TokenBudget,
		"TaskDescription": request.TaskDescription,
		"TaskDomain":      request.TaskDomain,
		"TaskPriority":    request.TaskPriority,
		"TaskDeadline":    request.TaskDeadline,
		"AvailableAgents": availableAgents,
		"CorpusDocuments": 100, // TODO: Get from corpus system
		"CodebaseSize":    500, // TODO: Get from project analysis
		"RecentTasks":     10,  // TODO: Get from task history
		"TeamVelocity":    5,   // TODO: Calculate from metrics
		"QualityLevel":    request.QualityLevel,
		"RiskTolerance":   request.RiskTolerance,
		"TimeConstraint":  request.TimeConstraint,
	}
}

// buildLayeredPrompt generates the layered prompt
func (tca *TaskComplexityAnalyzer) buildLayeredPrompt(
	ctx context.Context,
	request ComplexityAnalysisRequest,
	promptContext map[string]interface{},
) (*layered.LayeredPrompt, error) {
	// Note: turnCtx would be used with a layered.LayeredManager that has BuildLayeredPrompt
	// For now we're using the basic layered.LayeredManager with GetCompiledPrompt
	_ = layered.TurnContext{
		UserMessage:  request.TaskDescription,
		TaskID:       "complexity-analysis",
		CommissionID: "task-analysis",
		Urgency:      request.TaskPriority,
		Instructions: []string{"Analyze task complexity and provide structured recommendations"},
		Metadata:     promptContext,
	}

	// Build layered prompt using BuildLayeredPrompt
	turnCtx := layered.TurnContext{
		UserMessage: fmt.Sprintf(`Analyze task complexity:
Task: %s
Domain: %s
Priority: %s
Token Budget: %d
Quality Level: %s`,
			request.TaskDescription,
			request.TaskDomain,
			request.TaskPriority,
			request.TokenBudget,
			request.QualityLevel),
		TaskID:       "task-complexity-analysis",
		CommissionID: "analysis-session",
		Metadata:     promptContext,
	}

	layeredPrompt, err := tca.promptManager.BuildLayeredPrompt(ctx, "manager-agent", "analysis-session", turnCtx)
	if err != nil {
		return nil, err
	}

	return layeredPrompt, nil
}

// executeAnalysis executes the analysis with retry logic
func (tca *TaskComplexityAnalyzer) executeAnalysis(
	ctx context.Context,
	layeredPrompt *layered.LayeredPrompt,
	request ComplexityAnalysisRequest,
) (*ComplexityAnalysisResult, error) {
	artisanRequest := ArtisanRequest{
		SystemPrompt: layeredPrompt.Compiled,
		UserPrompt:   request.TaskDescription,
		Temperature:  0.3,
		MaxTokens:    2000,
	}

	// Execute with retry logic
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, NewTimeoutError("analysis cancelled or timed out", ctx.Err())
		default:
		}

		tca.logger.Debug("Executing analysis attempt",
			slog.Int("attempt", attempt),
			slog.Int("max_retries", maxRetries),
		)

		response, err := tca.artisanClient.Complete(ctx, artisanRequest)
		if err != nil {
			lastErr = err
			tca.logger.Warn("Analysis attempt failed",
				slog.Int("attempt", attempt),
				slog.Any("error", err),
			)

			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return nil, NewTimeoutError("analysis timed out", err)
			}

			// Wait before retry (exponential backoff)
			if attempt < maxRetries {
				waitTime := time.Duration(attempt) * time.Second
				time.Sleep(waitTime)
			}
			continue
		}

		// Log the raw response for debugging
		tca.logger.Debug("Received analysis response",
			slog.String("response", response.Content[:min(len(response.Content), 200)]),
			slog.Int("response_length", len(response.Content)),
		)

		// Parse the response
		result, err := tca.parseAnalysisResponse(response)
		if err != nil {
			lastErr = err
			tca.logger.Warn("Failed to parse response",
				slog.Int("attempt", attempt),
				slog.Any("error", err),
				slog.String("raw_response", response.Content),
			)

			// Don't retry parse errors - they're likely not transient
			return nil, NewParseError("failed to parse complexity analysis response", err)
		}

		return result, nil
	}

	return nil, NewArtisanError("failed to get complexity analysis after retries", lastErr)
}

// parseAnalysisResponse parses the artisan response with detailed error handling
func (tca *TaskComplexityAnalyzer) parseAnalysisResponse(response *ArtisanResponse) (*ComplexityAnalysisResult, error) {
	if response == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "response is nil", nil).
			WithComponent("manager").
			WithOperation("parseAnalysisResponse")
	}
	if response.Content == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "response content is empty", nil).
			WithComponent("manager").
			WithOperation("parseAnalysisResponse")
	}

	var result ComplexityAnalysisResult

	// Try direct JSON parsing first
	if err := json.Unmarshal([]byte(response.Content), &result); err == nil {
		return &result, nil
	}

	// Try extracting from markdown code blocks
	extractedJSON := extractJSONFromMarkdown(response.Content)
	if extractedJSON == "" {
		return nil, gerror.New(gerror.ErrCodeInternal, "no JSON found in response", nil).
			WithComponent("manager").
			WithOperation("parseAnalysisResponse").
			WithDetails("response_length", len(response.Content))
	}

	if err := json.Unmarshal([]byte(extractedJSON), &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse extracted JSON").
			WithComponent("manager").
			WithOperation("parseAnalysisResponse").
			WithDetails("json_length", len(extractedJSON))
	}

	return &result, nil
}

// Custom error types for better debugging
type AnalysisError struct {
	Type    string
	Message string
	Cause   error
}

func (e *AnalysisError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AnalysisError) Unwrap() error {
	return e.Cause
}

func NewValidationError(message string, cause error) error {
	return &AnalysisError{Type: "ValidationError", Message: message, Cause: cause}
}

func NewRegistryError(message string, cause error) error {
	return &AnalysisError{Type: "RegistryError", Message: message, Cause: cause}
}

func NewPromptError(message string, cause error) error {
	return &AnalysisError{Type: "PromptError", Message: message, Cause: cause}
}

func NewArtisanError(message string, cause error) error {
	return &AnalysisError{Type: "ArtisanError", Message: message, Cause: cause}
}

func NewParseError(message string, cause error) error {
	return &AnalysisError{Type: "ParseError", Message: message, Cause: cause}
}

func NewTimeoutError(message string, cause error) error {
	return &AnalysisError{Type: "TimeoutError", Message: message, Cause: cause}
}

// extractJSONFromMarkdown attempts to extract JSON from markdown code blocks
func extractJSONFromMarkdown(content string) string {
	// Look for ```json code blocks
	start := "```json"
	end := "```"

	startIdx := findSubstring(content, start)
	if startIdx == -1 {
		return ""
	}

	startIdx += len(start)
	endIdx := findSubstringFrom(content, end, startIdx)
	if endIdx == -1 {
		return ""
	}

	return content[startIdx:endIdx]
}

// Helper function to find substring
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function to find substring from a starting position
func findSubstringFrom(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
