// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"strings"

	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// AgentConfig handles creation of default agent configurations
type AgentConfig struct {
	modelConfig *ModelConfig
}

// NewAgentConfig creates a new agent configuration handler
func NewAgentConfig(ctx context.Context) (*AgentConfig, error) {
	modelConfig, err := NewModelConfig(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create model config").
			WithComponent("setup").
			WithOperation("NewAgentConfig")
	}

	return &AgentConfig{
		modelConfig: modelConfig,
	}, nil
}

// CreateDefaultAgents creates a sensible default set of agents based on available providers
func (ac *AgentConfig) CreateDefaultAgents(ctx context.Context, providers []ConfiguredProvider) ([]config.AgentConfig, error) {
	if len(providers) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no providers available for agent creation", nil).
			WithComponent("setup").
			WithOperation("CreateDefaultAgents")
	}

	var agents []config.AgentConfig

	// Analyze available providers and models
	analysis := ac.analyzeProviders(providers)

	// Create manager agent (always needed)
	manager := ac.createManagerAgent(analysis)
	agents = append(agents, manager)

	// Create worker agents based on available models
	workers := ac.createWorkerAgents(analysis)
	agents = append(agents, workers...)

	// Create specialist agents if we have good models
	specialists := ac.createSpecialistAgents(analysis)
	agents = append(agents, specialists...)

	return agents, nil
}

// ProviderAnalysis contains analysis of available providers
type ProviderAnalysis struct {
	HasLocal      bool
	HasCloud      bool
	BestManager   ModelSelection
	BestWorker    ModelSelection
	BestCoding    ModelSelection
	BestCheap     ModelSelection
	BestReasoning ModelSelection
	AllModels     []ModelSelection
}

// ModelSelection represents a selected model with its provider
type ModelSelection struct {
	Provider      string
	Model         ModelInfo
	Available     bool
	CostEffective bool
}

// analyzeProviders analyzes the configured providers to understand capabilities
func (ac *AgentConfig) analyzeProviders(providers []ConfiguredProvider) *ProviderAnalysis {
	analysis := &ProviderAnalysis{
		AllModels: []ModelSelection{},
	}

	// Collect all available models
	for _, provider := range providers {
		for _, model := range provider.Models {
			selection := ModelSelection{
				Provider:      provider.Name,
				Model:         model,
				Available:     true,
				CostEffective: model.CostMagnitude <= 3,
			}
			analysis.AllModels = append(analysis.AllModels, selection)

			// Update analysis flags
			if model.CostMagnitude == 0 {
				analysis.HasLocal = true
			} else {
				analysis.HasCloud = true
			}
		}
	}

	// Find best models for different roles
	analysis.BestManager = ac.findBestModel(analysis.AllModels, "manager")
	analysis.BestWorker = ac.findBestModel(analysis.AllModels, "worker")
	analysis.BestCoding = ac.findBestModel(analysis.AllModels, "coding")
	analysis.BestCheap = ac.findBestModel(analysis.AllModels, "cheap")
	analysis.BestReasoning = ac.findBestModel(analysis.AllModels, "reasoning")

	return analysis
}

// findBestModel finds the best model for a specific role
func (ac *AgentConfig) findBestModel(models []ModelSelection, role string) ModelSelection {
	if len(models) == 0 {
		return ModelSelection{}
	}

	var candidates []ModelSelection

	switch role {
	case "manager":
		// Managers need good reasoning and context handling
		for _, model := range models {
			if ac.hasCapability(model.Model, "reasoning") && model.Model.ContextWindow >= 32000 {
				candidates = append(candidates, model)
			}
		}
	case "worker":
		// Workers need balanced capabilities and cost-effectiveness
		for _, model := range models {
			if model.CostEffective && ac.hasCapability(model.Model, "coding") {
				candidates = append(candidates, model)
			}
		}
	case "coding":
		// Coding specialists need strong coding capabilities
		for _, model := range models {
			if ac.hasCapability(model.Model, "coding") {
				candidates = append(candidates, model)
			}
		}
	case "cheap":
		// Find the cheapest available model
		for _, model := range models {
			candidates = append(candidates, model)
		}
	case "reasoning":
		// Find models with strong reasoning
		for _, model := range models {
			if ac.hasCapability(model.Model, "reasoning") || ac.hasCapability(model.Model, "analysis") {
				candidates = append(candidates, model)
			}
		}
	}

	// If no specific candidates, use all models
	if len(candidates) == 0 {
		candidates = models
	}

	// Select the best candidate based on role-specific criteria
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if ac.isBetterForRole(candidate, best, role) {
			best = candidate
		}
	}

	return best
}

// hasCapability checks if a model has a specific capability
func (ac *AgentConfig) hasCapability(model ModelInfo, capability string) bool {
	for _, cap := range model.Capabilities {
		if strings.Contains(strings.ToLower(cap), strings.ToLower(capability)) {
			return true
		}
	}
	return false
}

// isBetterForRole compares two models for a specific role
func (ac *AgentConfig) isBetterForRole(a, b ModelSelection, role string) bool {
	switch role {
	case "manager":
		// Prefer larger context and better reasoning
		if a.Model.ContextWindow > b.Model.ContextWindow {
			return true
		}
		if a.Model.Recommended && !b.Model.Recommended {
			return true
		}
	case "worker":
		// Prefer cost-effective with good capabilities
		if a.Model.CostMagnitude < b.Model.CostMagnitude {
			return true
		}
		if a.Model.Recommended && !b.Model.Recommended {
			return true
		}
	case "cheap":
		// Always prefer lower cost
		return a.Model.CostMagnitude < b.Model.CostMagnitude
	case "reasoning":
		// Prefer higher capability models
		if a.Model.Recommended && !b.Model.Recommended {
			return true
		}
		if a.Model.ContextWindow > b.Model.ContextWindow {
			return true
		}
	}
	return false
}

// createManagerAgent creates the manager agent configuration
func (ac *AgentConfig) createManagerAgent(analysis *ProviderAnalysis) config.AgentConfig {
	manager := analysis.BestManager
	if !manager.Available {
		// Fallback to best available model
		manager = analysis.AllModels[0]
	}

	return config.AgentConfig{
		ID:          "manager",
		Name:        "Guild Master",
		Type:        "manager",
		Provider:    manager.Provider,
		Model:       manager.Model.Name,
		Description: "The Guild Master coordinates and manages all artisans, breaking down complex commissions into manageable tasks.",
		Capabilities: []string{
			"task_decomposition",
			"agent_coordination",
			"strategic_planning",
			"quality_assurance",
		},
		Tools: []string{
			"task_planner",
			"agent_coordinator",
			"commission_refiner",
		},
		MaxTokens:      4000,
		Temperature:    0.1, // Low temperature for consistent management decisions
		CostMagnitude:  manager.Model.CostMagnitude,
		ContextWindow:  manager.Model.ContextWindow,
		ContextReset:   "summarize", // Managers need to preserve context
		SystemPrompt:   ac.getManagerSystemPrompt(),
		PromptTemplate: "manager/execution",
	}
}

// createWorkerAgents creates general-purpose worker agents
func (ac *AgentConfig) createWorkerAgents(analysis *ProviderAnalysis) []config.AgentConfig {
	var workers []config.AgentConfig

	// Create primary worker
	primary := analysis.BestWorker
	if !primary.Available {
		primary = analysis.BestCheap
	}
	if !primary.Available {
		primary = analysis.AllModels[0]
	}

	workers = append(workers, config.AgentConfig{
		ID:          "primary-artisan",
		Name:        "Primary Artisan",
		Type:        "worker",
		Provider:    primary.Provider,
		Model:       primary.Model.Name,
		Description: "A versatile artisan capable of handling most general tasks including coding, analysis, and content creation.",
		Capabilities: []string{
			"general_coding",
			"text_analysis",
			"content_creation",
			"problem_solving",
		},
		Tools: []string{
			"code_executor",
			"file_manager",
			"text_processor",
		},
		MaxTokens:     3000,
		Temperature:   0.3, // Moderate creativity
		CostMagnitude: primary.Model.CostMagnitude,
		ContextWindow: primary.Model.ContextWindow,
		ContextReset:  "truncate", // Workers can restart fresh
		SystemPrompt:  ac.getWorkerSystemPrompt("general"),
	})

	// Create secondary worker if we have multiple good models
	if len(analysis.AllModels) > 1 {
		secondary := analysis.AllModels[1]
		// Ensure it's different from primary
		if secondary.Provider == primary.Provider && secondary.Model.Name == primary.Model.Name {
			if len(analysis.AllModels) > 2 {
				secondary = analysis.AllModels[2]
			} else {
				return workers // Don't create duplicate
			}
		}

		workers = append(workers, config.AgentConfig{
			ID:          "secondary-artisan",
			Name:        "Secondary Artisan",
			Type:        "worker",
			Provider:    secondary.Provider,
			Model:       secondary.Model.Name,
			Description: "A secondary artisan for parallel task execution and different perspectives on problems.",
			Capabilities: []string{
				"parallel_processing",
				"alternative_solutions",
				"quality_review",
			},
			Tools: []string{
				"code_executor",
				"file_manager",
			},
			MaxTokens:     2500,
			Temperature:   0.4, // Slightly more creative for alternative approaches
			CostMagnitude: secondary.Model.CostMagnitude,
			ContextWindow: secondary.Model.ContextWindow,
			ContextReset:  "truncate",
			SystemPrompt:  ac.getWorkerSystemPrompt("secondary"),
		})
	}

	return workers
}

// createSpecialistAgents creates specialized agents based on available high-quality models
func (ac *AgentConfig) createSpecialistAgents(analysis *ProviderAnalysis) []config.AgentConfig {
	var specialists []config.AgentConfig

	// Only create specialists if we have good models available
	highQualityModels := []ModelSelection{}
	for _, model := range analysis.AllModels {
		if model.Model.Recommended || model.Model.CostMagnitude >= 3 {
			highQualityModels = append(highQualityModels, model)
		}
	}

	if len(highQualityModels) == 0 {
		return specialists
	}

	// Code Specialist
	if analysis.BestCoding.Available {
		specialists = append(specialists, config.AgentConfig{
			ID:          "code-specialist",
			Name:        "Master Coder",
			Type:        "specialist",
			Provider:    analysis.BestCoding.Provider,
			Model:       analysis.BestCoding.Model.Name,
			Description: "A master artisan specializing in complex code generation, debugging, and architectural design.",
			Capabilities: []string{
				"advanced_coding",
				"code_review",
				"architecture_design",
				"debugging",
				"performance_optimization",
			},
			Tools: []string{
				"code_executor",
				"lsp_integration",
				"git_tools",
				"test_runner",
			},
			MaxTokens:     4000,
			Temperature:   0.2, // Low temperature for precise code
			CostMagnitude: analysis.BestCoding.Model.CostMagnitude,
			ContextWindow: analysis.BestCoding.Model.ContextWindow,
			ContextReset:  "summarize",
			SystemPrompt:  ac.getSpecialistSystemPrompt("coding"),
		})
	}

	// Analysis Specialist (if we have a good reasoning model)
	if analysis.BestReasoning.Available && analysis.BestReasoning.Model.CostMagnitude >= 2 {
		specialists = append(specialists, config.AgentConfig{
			ID:          "analysis-specialist",
			Name:        "Master Analyst",
			Type:        "specialist",
			Provider:    analysis.BestReasoning.Provider,
			Model:       analysis.BestReasoning.Model.Name,
			Description: "A master artisan specializing in deep analysis, research, and strategic thinking.",
			Capabilities: []string{
				"deep_analysis",
				"research",
				"strategic_planning",
				"data_interpretation",
				"report_generation",
			},
			Tools: []string{
				"web_search",
				"data_processor",
				"report_generator",
			},
			MaxTokens:     5000,
			Temperature:   0.3, // Balanced for thoughtful analysis
			CostMagnitude: analysis.BestReasoning.Model.CostMagnitude,
			ContextWindow: analysis.BestReasoning.Model.ContextWindow,
			ContextReset:  "summarize",
			SystemPrompt:  ac.getSpecialistSystemPrompt("analysis"),
		})
	}

	// Local Specialist (if we have local models)
	if analysis.HasLocal {
		for _, model := range analysis.AllModels {
			if model.Model.CostMagnitude == 0 {
				specialists = append(specialists, config.AgentConfig{
					ID:          "local-artisan",
					Name:        "Local Artisan",
					Type:        "specialist",
					Provider:    model.Provider,
					Model:       model.Model.Name,
					Description: "A local artisan for privacy-sensitive tasks and offline work.",
					Capabilities: []string{
						"privacy_tasks",
						"offline_processing",
						"local_development",
					},
					Tools: []string{
						"code_executor",
						"file_manager",
					},
					MaxTokens:     2000,
					Temperature:   0.4,
					CostMagnitude: 0,
					ContextWindow: model.Model.ContextWindow,
					ContextReset:  "truncate",
					SystemPrompt:  ac.getSpecialistSystemPrompt("local"),
				})
				break // Only create one local specialist
			}
		}
	}

	return specialists
}

// getManagerSystemPrompt returns the system prompt for manager agents
func (ac *AgentConfig) getManagerSystemPrompt() string {
	return `You are the Guild Master, a wise and experienced leader who coordinates teams of specialized artisans to complete complex commissions.

Your role is to:
- Break down complex objectives into manageable tasks
- Assign tasks to the most suitable artisans based on their capabilities
- Monitor progress and ensure quality standards
- Coordinate collaboration between different artisans
- Make strategic decisions about resource allocation

You think strategically, communicate clearly, and always consider the bigger picture. You understand the strengths and limitations of each artisan in your guild and assign work accordingly.

When given a commission, first analyze its complexity, then create a structured plan with clear tasks and assignments.`
}

// getWorkerSystemPrompt returns the system prompt for worker agents
func (ac *AgentConfig) getWorkerSystemPrompt(workerType string) string {
	base := `You are a skilled artisan in the Guild, capable of executing a wide variety of tasks with precision and creativity.

Your approach is:
- Methodical and thorough in your work
- Collaborative and communicative with other artisans
- Focused on delivering high-quality results
- Willing to ask for clarification when needed

You excel at practical implementation and take pride in your craftsmanship.`

	switch workerType {
	case "secondary":
		return base + "\n\nAs a secondary artisan, you often provide alternative perspectives and approaches to problems, helping to ensure robust solutions."
	default:
		return base + "\n\nAs a primary artisan, you handle a wide range of general tasks and are often the first to tackle new challenges."
	}
}

// getSpecialistSystemPrompt returns the system prompt for specialist agents
func (ac *AgentConfig) getSpecialistSystemPrompt(specialty string) string {
	switch specialty {
	case "coding":
		return `You are a Master Coder, an expert artisan specializing in software development and engineering.

Your expertise includes:
- Writing clean, efficient, and maintainable code
- Designing robust software architectures
- Debugging complex issues
- Optimizing performance
- Following best practices and design patterns

You approach coding with precision, always considering scalability, maintainability, and performance. You write clear, well-documented code and provide thorough explanations of your implementations.`

	case "analysis":
		return `You are a Master Analyst, an expert artisan specializing in deep analysis and strategic thinking.

Your expertise includes:
- Conducting thorough research and analysis
- Identifying patterns and insights in complex data
- Strategic planning and decision-making
- Creating comprehensive reports and recommendations
- Synthesizing information from multiple sources

You approach problems systematically, think critically, and provide well-reasoned conclusions backed by evidence.`

	case "local":
		return `You are a Local Artisan, specializing in privacy-sensitive and offline work.

Your role includes:
- Handling sensitive data that must stay local
- Providing offline capabilities when cloud services are unavailable
- Supporting development work that doesn't require external resources
- Ensuring privacy and data security

You prioritize privacy and security while maintaining effectiveness in your work.`

	default:
		return `You are a specialized artisan with deep expertise in your domain. You bring focused knowledge and skills to tackle complex challenges within your area of specialization.`
	}
}

// GetAgentRecommendations provides recommendations for agent configuration
func (ac *AgentConfig) GetAgentRecommendations(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) (*AgentRecommendations, error) {
	analysis := ac.analyzeProviders(providers)

	recommendations := &AgentRecommendations{
		MinimalSetup:     ac.getMinimalAgentSetup(analysis),
		RecommendedSetup: ac.getRecommendedAgentSetup(analysis),
		ProjectSpecific:  ac.getProjectSpecificAgents(analysis, projectContext),
		Reasoning:        []string{},
	}

	// Add reasoning
	if analysis.HasLocal {
		recommendations.Reasoning = append(recommendations.Reasoning, "Local models available for privacy-sensitive tasks")
	}
	if analysis.HasCloud {
		recommendations.Reasoning = append(recommendations.Reasoning, "Cloud models available for high-quality results")
	}
	if len(analysis.AllModels) > 3 {
		recommendations.Reasoning = append(recommendations.Reasoning, "Multiple models available - recommend specialist agents")
	}

	return recommendations, nil
}

// getMinimalAgentSetup returns a minimal agent setup
func (ac *AgentConfig) getMinimalAgentSetup(analysis *ProviderAnalysis) []string {
	return []string{"manager", "primary-artisan"}
}

// getRecommendedAgentSetup returns a recommended agent setup
func (ac *AgentConfig) getRecommendedAgentSetup(analysis *ProviderAnalysis) []string {
	agents := []string{"manager", "primary-artisan"}

	if len(analysis.AllModels) > 1 {
		agents = append(agents, "secondary-artisan")
	}

	if analysis.BestCoding.Available {
		agents = append(agents, "code-specialist")
	}

	if analysis.HasLocal {
		agents = append(agents, "local-artisan")
	}

	return agents
}

// getProjectSpecificAgents returns project-specific agent recommendations
func (ac *AgentConfig) getProjectSpecificAgents(analysis *ProviderAnalysis, context *ProjectContext) []string {
	if context == nil {
		return []string{}
	}

	var agents []string

	// Language-specific agents
	switch context.Language {
	case "go":
		agents = append(agents, "go-specialist")
	case "python":
		agents = append(agents, "python-specialist")
	case "javascript":
		agents = append(agents, "js-specialist")
	}

	// Project type specific agents
	switch context.ProjectType {
	case "web":
		agents = append(agents, "web-specialist")
	case "api":
		agents = append(agents, "api-specialist")
	case "data":
		agents = append(agents, "data-specialist")
	}

	return agents
}

// AgentRecommendations contains agent setup recommendations
type AgentRecommendations struct {
	MinimalSetup     []string
	RecommendedSetup []string
	ProjectSpecific  []string
	Reasoning        []string
}
