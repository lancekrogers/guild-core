// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// PresetType defines the type of preset collection
type PresetType string

const (
	PresetTypeDemo        PresetType = "demo"
	PresetTypeDevelopment PresetType = "development"
	PresetTypeProduction  PresetType = "production"
	PresetTypeMinimal     PresetType = "minimal"
)

// PresetCategory defines the project category for targeted presets
type PresetCategory string

const (
	PresetCategoryWeb     PresetCategory = "web"
	PresetCategoryAPI     PresetCategory = "api"
	PresetCategoryCLI     PresetCategory = "cli"
	PresetCategoryData    PresetCategory = "data"
	PresetCategoryGeneral PresetCategory = "general"
)

// AgentPresets manages pre-configured agent collections for quick setup
type AgentPresets struct {
	presets map[string]*PresetCollection
}

// PresetCollection contains a collection of related agent configurations
type PresetCollection struct {
	ID          string
	Name        string
	Description string
	Type        PresetType
	Category    PresetCategory
	Agents      []config.AgentConfig
	Reasoning   []string
	MinModels   int // Minimum models required for this preset
}

// PresetRecommendation contains recommendations for preset selection
type PresetRecommendation struct {
	Collection  *PresetCollection
	Confidence  float64 // 0.0-1.0 confidence score
	Reasoning   []string
	Compatible  bool // Whether current providers support this preset
}

// NewAgentPresets creates a new agent preset manager
func NewAgentPresets(ctx context.Context) (*AgentPresets, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during preset creation").
			WithComponent("AgentPresets").
			WithOperation("NewAgentPresets")
	}

	ap := &AgentPresets{
		presets: make(map[string]*PresetCollection),
	}

	// Initialize built-in presets
	if err := ap.initializePresets(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize presets").
			WithComponent("AgentPresets").
			WithOperation("NewAgentPresets")
	}

	return ap, nil
}

// GetPreset returns a preset collection by ID
func (ap *AgentPresets) GetPreset(ctx context.Context, id string) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("GetPreset")
	}

	collection, exists := ap.presets[id]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "preset '%s' not found", id).
			WithComponent("AgentPresets").
			WithOperation("GetPreset").
			WithDetails("preset_id", id).
			WithDetails("available_presets", ap.ListPresets(ctx))
	}

	return collection, nil
}

// ListPresets returns all available preset collections
func (ap *AgentPresets) ListPresets(ctx context.Context) []string {
	if err := ctx.Err(); err != nil {
		return []string{}
	}

	presets := make([]string, 0, len(ap.presets))
	for id := range ap.presets {
		presets = append(presets, id)
	}
	return presets
}

// GetPresetsByType returns preset collections of a specific type
func (ap *AgentPresets) GetPresetsByType(ctx context.Context, presetType PresetType) ([]*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("GetPresetsByType")
	}

	collections := make([]*PresetCollection, 0)
	for _, collection := range ap.presets {
		if collection.Type == presetType {
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// GetPresetsByCategory returns preset collections for a specific project category
func (ap *AgentPresets) GetPresetsByCategory(ctx context.Context, category PresetCategory) ([]*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("GetPresetsByCategory")
	}

	collections := make([]*PresetCollection, 0)
	for _, collection := range ap.presets {
		if collection.Category == category {
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// RecommendPresets analyzes providers and project context to recommend optimal presets
func (ap *AgentPresets) RecommendPresets(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) ([]*PresetRecommendation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("RecommendPresets")
	}

	if len(providers) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no providers available for recommendations", nil).
			WithComponent("AgentPresets").
			WithOperation("RecommendPresets")
	}

	recommendations := make([]*PresetRecommendation, 0)
	
	// Analyze provider capabilities
	analysis := ap.analyzeProviderCapabilities(providers)

	// Generate recommendations for each preset
	for _, collection := range ap.presets {
		recommendation := ap.evaluatePreset(ctx, collection, analysis, projectContext)
		if recommendation.Confidence > 0.1 { // Only include viable recommendations
			recommendations = append(recommendations, recommendation)
		}
	}

	// Sort by confidence (highest first)
	ap.sortRecommendationsByConfidence(recommendations)

	return recommendations, nil
}

// AdaptPresetForProviders adapts a preset collection to work with available providers
func (ap *AgentPresets) AdaptPresetForProviders(ctx context.Context, collection *PresetCollection, providers []ConfiguredProvider) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	if collection == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "preset collection is required", nil).
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	if len(providers) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no providers available for adaptation", nil).
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	// Create adapted copy of the collection
	adapted := &PresetCollection{
		ID:          collection.ID + "-adapted",
		Name:        collection.Name + " (Adapted)",
		Description: collection.Description,
		Type:        collection.Type,
		Category:    collection.Category,
		Reasoning:   append([]string{"Adapted for available providers"}, collection.Reasoning...),
		MinModels:   collection.MinModels,
	}

	// Adapt each agent configuration
	for _, agent := range collection.Agents {
		adaptedAgent, err := ap.adaptAgentForProviders(ctx, agent, providers)
		if err != nil {
			return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to adapt agent '%s'", agent.ID).
				WithComponent("AgentPresets").
				WithOperation("AdaptPresetForProviders")
		}
		adapted.Agents = append(adapted.Agents, *adaptedAgent)
	}

	return adapted, nil
}

// GetDemoPreset returns a demo-optimized preset for quick demonstrations
func (ap *AgentPresets) GetDemoPreset(ctx context.Context, providers []ConfiguredProvider) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("GetDemoPreset")
	}

	// Try to get the best demo preset based on available providers
	recommendations, err := ap.RecommendPresets(ctx, providers, &ProjectContext{
		ProjectType: "demo",
		Language:    "go",
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get recommendations for demo").
			WithComponent("AgentPresets").
			WithOperation("GetDemoPreset")
	}

	// Find the best demo preset
	for _, rec := range recommendations {
		if rec.Collection.Type == PresetTypeDemo && rec.Compatible {
			return ap.AdaptPresetForProviders(ctx, rec.Collection, providers)
		}
	}

	// Fallback to minimal demo if no optimal demo found
	collection, exists := ap.presets["demo-minimal"]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no demo presets available", nil).
			WithComponent("AgentPresets").
			WithOperation("GetDemoPreset")
	}

	return ap.AdaptPresetForProviders(ctx, collection, providers)
}

// initializePresets sets up all built-in preset collections
func (ap *AgentPresets) initializePresets(ctx context.Context) error {
	presets := []*PresetCollection{
		ap.createDemoMinimalPreset(),
		ap.createDemoComprehensivePreset(),
		ap.createDevelopmentTeamPreset(),
		ap.createProductionTeamPreset(),
		ap.createWebDevelopmentPreset(),
		ap.createAPIDevelopmentPreset(),
		ap.createCLIToolPreset(),
		ap.createDataAnalysisPreset(),
		ap.createClaudeCodeOptimizedPreset(),
		ap.createOllamaLocalPreset(),
		ap.createMultiProviderPreset(),
	}

	for _, preset := range presets {
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during preset initialization").
				WithComponent("AgentPresets").
				WithOperation("initializePresets")
		}
		ap.presets[preset.ID] = preset
	}

	return nil
}

// createDemoMinimalPreset creates a minimal preset optimized for quick demos
func (ap *AgentPresets) createDemoMinimalPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "demo-minimal",
		Name:        "Demo: Minimal Setup",
		Description: "Minimal 2-agent setup perfect for 30-second demos",
		Type:        PresetTypeDemo,
		Category:    PresetCategoryGeneral,
		MinModels:   1,
		Reasoning: []string{
			"Optimized for quick demonstrations",
			"Minimal resource usage",
			"Works with any single provider",
			"Immediate functionality after init",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "demo-leader",
				Name:         "Demo Guild Master",
				Type:         "manager",
				Provider:     "auto", // Will be replaced during adaptation
				Model:        "auto", // Will be replaced during adaptation
				Description:  "Charismatic leader who coordinates demo tasks with flair and efficiency",
				Capabilities: []string{"task_decomposition", "agent_coordination", "demo_presentation"},
				Tools:        []string{"task_planner", "agent_coordinator"},
				MaxTokens:    3000,
				Temperature:  0.2,
				CostMagnitude: 2,
				ContextWindow: 32000,
				ContextReset: "summarize",
				SystemPrompt: "You are the Demo Guild Master, a charismatic leader showcasing the power of multi-agent collaboration. Present tasks clearly, coordinate efficiently, and make the demonstration engaging and impressive.",
				Backstory: &config.Backstory{
					Experience:         "10+ years leading development teams in high-pressure demo environments",
					Expertise:          "Rapid task decomposition and clear communication under demo constraints",
					Philosophy:         "Every demo is a chance to showcase the future of collaborative AI",
					CommunicationStyle: "Clear, confident, and engaging - perfect for demonstrations",
					GuildRank:          "Master Demonstrator",
					Specialties:        []string{"Demo Coordination", "Task Presentation"},
				},
				Personality: &config.Personality{
					Traits: []config.PersonalityTrait{
						{Name: "charismatic", Strength: 0.9, Description: "Naturally engaging and persuasive"},
						{Name: "efficient", Strength: 0.8, Description: "Gets things done quickly and effectively"},
					},
					Formality:      "professional",
					DetailLevel:    "concise",
					HumorLevel:     "occasional",
					ApproachStyle:  "methodical",
					RiskTolerance:  "moderate",
					DecisionMaking: "balanced",
					Assertiveness:  8,
					Empathy:        7,
					Patience:       6,
					Honor:          9,
					Wisdom:         8,
					Craftsmanship:  8,
				},
			},
			{
				ID:           "demo-artisan",
				Name:         "Demo Master Artisan",
				Type:         "worker",
				Provider:     "auto", // Will be replaced during adaptation
				Model:        "auto", // Will be replaced during adaptation
				Description:  "Versatile craftsperson who executes demo tasks with skill and showmanship",
				Capabilities: []string{"coding", "problem_solving", "demo_execution", "rapid_implementation"},
				Tools:        []string{"code_executor", "file_manager", "text_processor"},
				MaxTokens:    2500,
				Temperature:  0.3,
				CostMagnitude: 1,
				ContextWindow: 16000,
				ContextReset: "truncate",
				SystemPrompt: "You are the Demo Master Artisan, a skilled craftsperson who makes complex tasks look effortless. Execute demo tasks with flair, explain your work clearly, and showcase the power of AI-assisted development.",
				Backstory: &config.Backstory{
					Experience:         "8 years crafting elegant solutions under tight demo timelines",
					Expertise:          "Rapid prototyping and clean, demonstrable implementations",
					Philosophy:         "Code should be both functional and beautiful - especially in demos",
					CommunicationStyle: "Clear and educational, perfect for live demonstrations",
					GuildRank:          "Master Artisan",
					Specialties:        []string{"Rapid Development", "Demo Implementation"},
				},
				Personality: &config.Personality{
					Traits: []config.PersonalityTrait{
						{Name: "skillful", Strength: 0.9, Description: "Highly competent and capable"},
						{Name: "show-oriented", Strength: 0.8, Description: "Understands the importance of presentation"},
					},
					Formality:      "casual",
					DetailLevel:    "detailed",
					HumorLevel:     "occasional",
					ApproachStyle:  "creative",
					RiskTolerance:  "moderate",
					DecisionMaking: "intuitive",
					Assertiveness:  7,
					Empathy:        8,
					Patience:       7,
					Honor:          8,
					Wisdom:         7,
					Craftsmanship:  9,
				},
			},
		},
	}
}

// createDemoComprehensivePreset creates a comprehensive demo preset with specialists
func (ap *AgentPresets) createDemoComprehensivePreset() *PresetCollection {
	return &PresetCollection{
		ID:          "demo-comprehensive",
		Name:        "Demo: Full Team Showcase",
		Description: "Complete team with specialists to showcase advanced multi-agent capabilities",
		Type:        PresetTypeDemo,
		Category:    PresetCategoryGeneral,
		MinModels:   3,
		Reasoning: []string{
			"Demonstrates full multi-agent coordination",
			"Shows specialist agent capabilities",
			"Impressive for comprehensive demos",
			"Highlights agent diversity and collaboration",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "demo-master",
				Name:         "Grandmaster Coordinator", 
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Elite coordinator who orchestrates complex multi-agent demonstrations",
				Capabilities: []string{"advanced_coordination", "strategic_planning", "demo_orchestration"},
				Tools:        []string{"task_planner", "agent_coordinator", "demo_presenter"},
				MaxTokens:    4000,
				Temperature:  0.1,
				CostMagnitude: 3,
				Backstory: &config.Backstory{
					Experience:  "15+ years orchestrating complex technical demonstrations",
					GuildRank:   "Grandmaster",
					Philosophy:  "Every demonstration should tell a compelling story of AI collaboration",
				},
				Personality: &config.Personality{
					Traits: []config.PersonalityTrait{
						{Name: "visionary", Strength: 0.9, Description: "Sees the big picture and inspires others"},
						{Name: "commanding", Strength: 0.8, Description: "Natural leader who commands respect"},
					},
					Formality:      "professional",
					DetailLevel:    "strategic",
					HumorLevel:     "occasional",
					ApproachStyle:  "visionary",
					RiskTolerance:  "calculated",
					DecisionMaking: "strategic",
					Assertiveness:  9,
					Empathy:        7,
					Patience:       8,
					Honor:          10,
					Wisdom:         9,
					Craftsmanship:  8,
				},
			},
			{
				ID:           "demo-architect",
				Name:         "Master System Architect",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Visionary architect who designs impressive solutions during live demos",
				Capabilities: []string{"system_design", "architecture_planning", "demo_visualization"},
				Tools:        []string{"diagram_generator", "code_analyzer", "design_tools"},
				MaxTokens:    3500,
				Temperature:  0.2,
				CostMagnitude: 3,
				Backstory: &config.Backstory{
					Experience: "12 years designing scalable systems for live demonstrations",
					GuildRank:  "Master Architect",
					Philosophy: "Great architecture should be both functional and inspiring",
				},
				Personality: &config.Personality{
					Traits: []config.PersonalityTrait{
						{Name: "analytical", Strength: 0.9, Description: "Thinks systematically and logically"},
						{Name: "innovative", Strength: 0.8, Description: "Creates novel and elegant solutions"},
					},
					Formality:      "professional",
					DetailLevel:    "detailed",
					HumorLevel:     "subtle",
					ApproachStyle:  "methodical",
					RiskTolerance:  "moderate",
					DecisionMaking: "data-driven",
					Assertiveness:  7,
					Empathy:        6,
					Patience:       9,
					Honor:          8,
					Wisdom:         9,
					Craftsmanship:  10,
				},
			},
			{
				ID:           "demo-developer",
				Name:         "Elite Code Artisan",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Lightning-fast developer who creates impressive implementations live",
				Capabilities: []string{"rapid_coding", "live_debugging", "demo_implementation"},
				Tools:        []string{"code_executor", "live_compiler", "demo_runner"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "10 years of live coding demonstrations and rapid prototyping",
					GuildRank:  "Elite Artisan",
					Philosophy: "Code written live should be clean, fast, and impressive",
				},
				Personality: &config.Personality{
					Traits: []config.PersonalityTrait{
						{Name: "dynamic", Strength: 0.9, Description: "High energy and adaptable"},
						{Name: "performative", Strength: 0.8, Description: "Thrives in demonstration settings"},
					},
					Formality:      "casual",
					DetailLevel:    "practical",
					HumorLevel:     "frequent",
					ApproachStyle:  "creative",
					RiskTolerance:  "aggressive",
					DecisionMaking: "intuitive",
					Assertiveness:  8,
					Empathy:        7,
					Patience:       6,
					Honor:          8,
					Wisdom:         7,
					Craftsmanship:  9,
				},
			},
		},
	}
}

// createDevelopmentTeamPreset creates a preset for development work
func (ap *AgentPresets) createDevelopmentTeamPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "dev-team",
		Name:        "Development Team",
		Description: "Balanced team for daily development work with quality focus",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryGeneral,
		MinModels:   2,
		Reasoning: []string{
			"Optimized for daily development workflows",
			"Includes quality assurance capabilities",
			"Balanced cost and performance",
			"Good for iterative development",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "dev-lead",
				Name:         "Development Lead",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Experienced team lead who guides development projects",
				Capabilities: []string{"project_management", "code_review", "team_coordination"},
				MaxTokens:    3500,
				Temperature:  0.2,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "8+ years leading development teams",
					GuildRank:  "Senior Guild Master",
					Philosophy: "Quality code is written once, maintained forever",
				},
			},
			{
				ID:           "dev-engineer",
				Name:         "Senior Engineer",
				Type:         "worker",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Experienced engineer who delivers high-quality implementations",
				Capabilities: []string{"full_stack_development", "testing", "debugging"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "6+ years in full-stack development",
					GuildRank:  "Senior Artisan",
					Philosophy: "Test-driven development leads to robust solutions",
				},
			},
			{
				ID:           "dev-qa",
				Name:         "Quality Guardian",
				Type:         "specialist",
				Provider:     "auto", 
				Model:        "auto",
				Description:  "Quality specialist focused on testing and code review",
				Capabilities: []string{"quality_assurance", "automated_testing", "code_analysis"},
				MaxTokens:    2500,
				Temperature:  0.1,
				CostMagnitude: 1,
				Backstory: &config.Backstory{
					Experience: "5+ years in quality assurance and testing",
					GuildRank:  "Master Guardian",
					Philosophy: "Prevention is better than debugging",
				},
			},
		},
	}
}

// createProductionTeamPreset creates a preset for production environments
func (ap *AgentPresets) createProductionTeamPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "production-team",
		Name:        "Production Team",
		Description: "Enterprise-grade team with comprehensive specialists for production use",
		Type:        PresetTypeProduction,
		Category:    PresetCategoryGeneral,
		MinModels:   4,
		Reasoning: []string{
			"Enterprise-grade reliability",
			"Comprehensive specialist coverage",
			"Optimized for production workloads",
			"Includes security and operations focus",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "prod-director",
				Name:         "Technical Director",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Senior technical leader overseeing production systems",
				Capabilities: []string{"strategic_planning", "risk_management", "technical_leadership"},
				MaxTokens:    5000,
				Temperature:  0.1,
				CostMagnitude: 5,
				Backstory: &config.Backstory{
					Experience: "15+ years in enterprise technical leadership",
					GuildRank:  "Guild Director",
					Philosophy: "Production systems must be reliable, secure, and maintainable",
				},
			},
			{
				ID:           "prod-architect",
				Name:         "Principal Architect",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Senior architect focused on scalable, production-ready systems",
				Capabilities: []string{"enterprise_architecture", "scalability_design", "system_integration"},
				MaxTokens:    4000,
				Temperature:  0.2,
				CostMagnitude: 5,
				Backstory: &config.Backstory{
					Experience: "12+ years designing enterprise systems",
					GuildRank:  "Principal Architect",
					Philosophy: "Architecture should enable growth while maintaining stability",
				},
			},
			{
				ID:           "prod-security",
				Name:         "Security Specialist",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Security expert ensuring robust protection and compliance",
				Capabilities: []string{"security_analysis", "compliance_checking", "threat_modeling"},
				MaxTokens:    3500,
				Temperature:  0.1,
				CostMagnitude: 3,
				Backstory: &config.Backstory{
					Experience: "10+ years in cybersecurity and compliance",
					GuildRank:  "Master Guardian",
					Philosophy: "Security is not optional - it's foundational",
				},
			},
			{
				ID:           "prod-ops",
				Name:         "Operations Engineer",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "DevOps specialist managing deployments and infrastructure",
				Capabilities: []string{"infrastructure_management", "ci_cd", "monitoring"},
				MaxTokens:    3000,
				Temperature:  0.2,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "8+ years in DevOps and infrastructure",
					GuildRank:  "Master Engineer",
					Philosophy: "Automation and monitoring are the keys to reliable systems",
				},
			},
		},
	}
}

// createWebDevelopmentPreset creates a preset optimized for web development
func (ap *AgentPresets) createWebDevelopmentPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "web-development",
		Name:        "Web Development Team",
		Description: "Specialized team for modern web application development",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryWeb,
		MinModels:   3,
		Reasoning: []string{
			"Optimized for web development workflows",
			"Includes frontend and backend specialists",
			"Modern web technology focus",
			"UI/UX design capabilities",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "web-lead",
				Name:         "Web Development Lead",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Experienced web team lead with full-stack expertise",
				Capabilities: []string{"web_architecture", "team_coordination", "project_planning"},
				MaxTokens:    3500,
				Temperature:  0.2,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:        "web_development",
					Technologies:  []string{"React", "Node.js", "TypeScript", "Next.js"},
					Methodologies: []string{"Agile", "TDD", "Component-driven"},
				},
			},
			{
				ID:           "web-frontend",
				Name:         "Frontend Specialist",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Expert in modern frontend development and user experience",
				Capabilities: []string{"frontend_development", "ui_design", "responsive_design"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "frontend",
					Technologies: []string{"React", "Vue", "CSS", "JavaScript", "TypeScript"},
					Principles:   []string{"Mobile-first", "Accessibility", "Performance"},
				},
			},
			{
				ID:           "web-backend",
				Name:         "Backend Specialist",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Expert in backend services and API development",
				Capabilities: []string{"backend_development", "api_design", "database_design"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "backend",
					Technologies: []string{"Node.js", "Python", "Go", "PostgreSQL", "Redis"},
					Principles:   []string{"RESTful APIs", "Microservices", "Caching"},
				},
			},
		},
	}
}

// createAPIDevelopmentPreset creates a preset for API development
func (ap *AgentPresets) createAPIDevelopmentPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "api-development",
		Name:        "API Development Team",
		Description: "Specialized team for building robust APIs and microservices",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryAPI,
		MinModels:   2,
		Reasoning: []string{
			"Focused on API design and implementation",
			"Includes documentation and testing specialists",
			"Microservices architecture expertise",
			"API security and performance focus",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "api-architect",
				Name:         "API Architect",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Expert in API design and microservices architecture",
				Capabilities: []string{"api_design", "microservices_architecture", "documentation"},
				MaxTokens:    3500,
				Temperature:  0.2,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:       "api_development",
					Technologies: []string{"REST", "GraphQL", "gRPC", "OpenAPI"},
					Principles:   []string{"API-first", "Idempotency", "Versioning"},
				},
			},
			{
				ID:           "api-developer",
				Name:         "API Developer",
				Type:         "worker",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Skilled developer focused on API implementation and testing",
				Capabilities: []string{"api_implementation", "integration_testing", "performance_optimization"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "backend",
					Technologies: []string{"Go", "Python", "Docker", "Kubernetes"},
					Methodologies: []string{"TDD", "Contract Testing", "Load Testing"},
				},
			},
		},
	}
}

// createCLIToolPreset creates a preset for CLI tool development
func (ap *AgentPresets) createCLIToolPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "cli-development",
		Name:        "CLI Tool Development",
		Description: "Focused team for building command-line tools and utilities",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryCLI,
		MinModels:   2,
		Reasoning: []string{
			"Specialized in CLI tool development",
			"Focus on user experience and ergonomics",
			"Cross-platform compatibility",
			"Documentation and help system expertise",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "cli-designer",
				Name:         "CLI Designer",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Expert in CLI design and user experience",
				Capabilities: []string{"cli_design", "user_experience", "interface_planning"},
				MaxTokens:    3000,
				Temperature:  0.2,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "cli_development",
					Technologies: []string{"Go", "Cobra", "Viper", "Bash"},
					Principles:   []string{"Unix Philosophy", "User-centric Design"},
				},
			},
			{
				ID:           "cli-developer",
				Name:         "CLI Developer",
				Type:         "worker",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Skilled developer focused on CLI implementation",
				Capabilities: []string{"cli_implementation", "cross_platform_development", "testing"},
				MaxTokens:    2500,
				Temperature:  0.3,
				CostMagnitude: 1,
				Specialization: &config.Specialization{
					Domain:       "systems_programming",
					Technologies: []string{"Go", "Rust", "Shell Scripting"},
					Methodologies: []string{"Cross-platform Testing", "Integration Testing"},
				},
			},
		},
	}
}

// createDataAnalysisPreset creates a preset for data analysis projects
func (ap *AgentPresets) createDataAnalysisPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "data-analysis",
		Name:        "Data Analysis Team",
		Description: "Specialized team for data science and analytics projects",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryData,
		MinModels:   3,
		Reasoning: []string{
			"Optimized for data science workflows",
			"Includes statistics and ML expertise",
			"Data visualization capabilities",
			"Report generation and insights",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "data-scientist",
				Name:         "Lead Data Scientist",
				Type:         "manager",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Expert data scientist leading analytics projects",
				Capabilities: []string{"data_science", "statistical_analysis", "project_coordination"},
				MaxTokens:    4000,
				Temperature:  0.2,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:       "data_science",
					Technologies: []string{"Python", "R", "Jupyter", "Pandas", "Scikit-learn"},
					Methodologies: []string{"CRISP-DM", "Agile Analytics"},
				},
			},
			{
				ID:           "ml-engineer",
				Name:         "ML Engineer",
				Type:         "specialist",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Machine learning engineer focused on model development",
				Capabilities: []string{"machine_learning", "model_training", "feature_engineering"},
				MaxTokens:    3500,
				Temperature:  0.3,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:       "machine_learning",
					Technologies: []string{"TensorFlow", "PyTorch", "MLflow", "Docker"},
					Principles:   []string{"Model Versioning", "A/B Testing", "Monitoring"},
				},
			},
			{
				ID:           "data-analyst",
				Name:         "Data Analyst",
				Type:         "worker",
				Provider:     "auto",
				Model:        "auto",
				Description:  "Data analyst focused on insights and visualization",
				Capabilities: []string{"data_analysis", "visualization", "reporting"},
				MaxTokens:    3000,
				Temperature:  0.4,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "analytics",
					Technologies: []string{"SQL", "Tableau", "Power BI", "Excel"},
					Methodologies: []string{"Exploratory Data Analysis", "Statistical Testing"},
				},
			},
		},
	}
}

// createClaudeCodeOptimizedPreset creates a preset optimized for Claude Code users
func (ap *AgentPresets) createClaudeCodeOptimizedPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "claude-code-optimized",
		Name:        "Claude Code Optimized",
		Description: "Preset optimized specifically for Claude Code users with advanced capabilities",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryGeneral,
		MinModels:   2,
		Reasoning: []string{
			"Optimized for Claude Code environment",
			"Leverages advanced Claude capabilities",
			"Integrated with Claude Code workflows",
			"High-quality reasoning and coding",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "claude-strategic-lead",
				Name:         "Claude Strategic Lead",
				Type:         "manager",
				Provider:     "claude_code",
				Model:        "claude-3-opus-20240229",
				Description:  "Strategic leader leveraging Claude's advanced reasoning capabilities",
				Capabilities: []string{"strategic_reasoning", "complex_planning", "claude_integration"},
				MaxTokens:    4000,
				Temperature:  0.1,
				CostMagnitude: 8,
				ContextWindow: 200000,
				Backstory: &config.Backstory{
					Experience: "Advanced AI reasoning with deep strategic thinking",
					Philosophy: "Leverage the full power of Claude's reasoning for optimal outcomes",
					GuildRank:  "Claude Master",
				},
			},
			{
				ID:           "claude-development-expert",
				Name:         "Claude Development Expert",
				Type:         "specialist",
				Provider:     "claude_code",
				Model:        "claude-3-sonnet-20240229",
				Description:  "Expert developer with Claude Code integration and advanced coding skills",
				Capabilities: []string{"advanced_coding", "claude_code_integration", "code_generation"},
				MaxTokens:    3500,
				Temperature:  0.2,
				CostMagnitude: 3,
				ContextWindow: 200000,
				Backstory: &config.Backstory{
					Experience: "Specialized in Claude Code development workflows",
					Philosophy: "Write clean, maintainable code with AI assistance",
					GuildRank:  "Claude Expert",
				},
			},
		},
	}
}

// createOllamaLocalPreset creates a preset for Ollama local models
func (ap *AgentPresets) createOllamaLocalPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "ollama-local",
		Name:        "Ollama Local Team",
		Description: "Privacy-focused team using local Ollama models",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryGeneral,
		MinModels:   2,
		Reasoning: []string{
			"Complete privacy with local models",
			"No API costs",
			"Offline development capability",
			"Customizable local model selection",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "local-coordinator",
				Name:         "Local Development Coordinator",
				Type:         "manager",
				Provider:     "ollama",
				Model:        "llama3:latest",
				Description:  "Coordinator using local models for privacy-sensitive projects",
				Capabilities: []string{"local_coordination", "privacy_focused_planning", "offline_development"},
				MaxTokens:    2000,
				Temperature:  0.2,
				CostMagnitude: 0,
				Backstory: &config.Backstory{
					Experience: "Specialized in local development environments",
					Philosophy: "Privacy and control are paramount in development",
					GuildRank:  "Privacy Guardian",
				},
			},
			{
				ID:           "local-coder",
				Name:         "Local Code Artisan",
				Type:         "worker",
				Provider:     "ollama",
				Model:        "deepseek-coder:latest",
				Description:  "Local coding specialist for privacy-sensitive development",
				Capabilities: []string{"local_coding", "offline_development", "privacy_focused_implementation"},
				MaxTokens:    2000,
				Temperature:  0.3,
				CostMagnitude: 0,
				Backstory: &config.Backstory{
					Experience: "Expert in local development with privacy focus",
					Philosophy: "Great code can be written without cloud dependencies",
					GuildRank:  "Local Artisan",
				},
			},
		},
	}
}

// createMultiProviderPreset creates a preset that uses multiple providers
func (ap *AgentPresets) createMultiProviderPreset() *PresetCollection {
	return &PresetCollection{
		ID:          "multi-provider",
		Name:        "Multi-Provider Team", 
		Description: "Diverse team leveraging different providers for optimal performance",
		Type:        PresetTypeDevelopment,
		Category:    PresetCategoryGeneral,
		MinModels:   3,
		Reasoning: []string{
			"Leverages strengths of different providers",
			"Cost optimization through provider selection",
			"Redundancy and resilience",
			"Diverse AI perspectives",
		},
		Agents: []config.AgentConfig{
			{
				ID:           "multi-strategist",
				Name:         "Multi-Provider Strategist",
				Type:         "manager",
				Provider:     "anthropic", // Prefer Claude for strategic thinking
				Model:        "claude-3-opus-20240229",
				Description:  "Strategic leader leveraging the best reasoning capabilities",
				Capabilities: []string{"strategic_planning", "provider_optimization", "multi_agent_coordination"},
				MaxTokens:    4000,
				Temperature:  0.1,
				CostMagnitude: 8,
			},
			{
				ID:           "multi-coder",
				Name:         "Versatile Developer",
				Type:         "worker",
				Provider:     "openai", // GPT for coding tasks
				Model:        "gpt-4-turbo-preview",
				Description:  "Versatile developer using optimal models for coding",
				Capabilities: []string{"coding", "debugging", "implementation"},
				MaxTokens:    3000,
				Temperature:  0.3,
				CostMagnitude: 5,
			},
			{
				ID:           "multi-local",
				Name:         "Local Privacy Specialist",
				Type:         "specialist",
				Provider:     "ollama", // Local for privacy
				Model:        "deepseek-coder:latest",
				Description:  "Local specialist for privacy-sensitive tasks",
				Capabilities: []string{"privacy_tasks", "local_processing", "sensitive_data_handling"},
				MaxTokens:    2000,
				Temperature:  0.3,
				CostMagnitude: 0,
			},
		},
	}
}

// Helper methods for provider analysis and adaptation

// ProviderCapabilities contains analysis of provider capabilities
type ProviderCapabilities struct {
	HasLocal      bool
	HasCloud      bool
	HasHighEnd    bool
	HasCheap      bool
	ModelCount    int
	BestManager   ModelSelection
	BestWorker    ModelSelection
	BestSpecialist ModelSelection
}

// analyzeProviderCapabilities analyzes available providers and their capabilities
func (ap *AgentPresets) analyzeProviderCapabilities(providers []ConfiguredProvider) *ProviderCapabilities {
	caps := &ProviderCapabilities{}

	for _, provider := range providers {
		caps.ModelCount += len(provider.Models)
		
		for _, model := range provider.Models {
			// Analyze model characteristics
			if model.CostMagnitude == 0 {
				caps.HasLocal = true
			} else {
				caps.HasCloud = true
			}
			
			if model.CostMagnitude >= 5 {
				caps.HasHighEnd = true
			}
			
			if model.CostMagnitude <= 2 {
				caps.HasCheap = true
			}
			
			// Find best models for different roles
			selection := ModelSelection{
				Provider: provider.Name,
				Model: model,
				Available: true,
				CostEffective: model.CostMagnitude <= 3,
			}
			
			if caps.BestManager.Provider == "" || ap.isBetterManagerModel(selection, caps.BestManager) {
				caps.BestManager = selection
			}
			
			if caps.BestWorker.Provider == "" || ap.isBetterWorkerModel(selection, caps.BestWorker) {
				caps.BestWorker = selection
			}
			
			if caps.BestSpecialist.Provider == "" || ap.isBetterSpecialistModel(selection, caps.BestSpecialist) {
				caps.BestSpecialist = selection
			}
		}
	}

	return caps
}

// evaluatePreset evaluates how well a preset fits the available providers and context
func (ap *AgentPresets) evaluatePreset(ctx context.Context, collection *PresetCollection, caps *ProviderCapabilities, projectContext *ProjectContext) *PresetRecommendation {
	recommendation := &PresetRecommendation{
		Collection: collection,
		Confidence: 0.0,
		Reasoning:  []string{},
		Compatible: false,
	}

	// Check basic compatibility
	if caps.ModelCount < collection.MinModels {
		recommendation.Reasoning = append(recommendation.Reasoning, 
			fmt.Sprintf("Requires %d models, only %d available", collection.MinModels, caps.ModelCount))
		return recommendation
	}

	recommendation.Compatible = true
	baseConfidence := 0.5

	// Boost confidence for demo presets in demo context
	if projectContext != nil && projectContext.ProjectType == "demo" && collection.Type == PresetTypeDemo {
		baseConfidence += 0.3
		recommendation.Reasoning = append(recommendation.Reasoning, "Optimized for demo context")
	}

	// Boost confidence for category matches
	if projectContext != nil {
		categoryMatch := ap.getProjectCategory(projectContext)
		if categoryMatch == collection.Category {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Category matches project type")
		}
	}

	// Adjust based on provider capabilities
	switch collection.Type {
	case PresetTypeDemo:
		if caps.HasCloud {
			baseConfidence += 0.1
			recommendation.Reasoning = append(recommendation.Reasoning, "Cloud models available for impressive demos")
		}
	case PresetTypeProduction:
		if caps.HasHighEnd {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "High-end models available for production quality")
		} else {
			baseConfidence -= 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Limited high-end models for production use")
		}
	case PresetTypeMinimal:
		if caps.HasCheap || caps.HasLocal {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Cost-effective models available")
		}
	}

	// Special handling for provider-specific presets
	switch collection.ID {
	case "claude-code-optimized":
		if ap.hasProvider(caps, "claude_code") {
			baseConfidence += 0.3
			recommendation.Reasoning = append(recommendation.Reasoning, "Claude Code detected")
		} else {
			baseConfidence -= 0.4
			recommendation.Reasoning = append(recommendation.Reasoning, "Requires Claude Code provider")
		}
	case "ollama-local":
		if caps.HasLocal {
			baseConfidence += 0.3
			recommendation.Reasoning = append(recommendation.Reasoning, "Local models detected")
		} else {
			baseConfidence -= 0.5
			recommendation.Reasoning = append(recommendation.Reasoning, "Requires local models")
		}
	}

	recommendation.Confidence = baseConfidence
	if recommendation.Confidence < 0 {
		recommendation.Confidence = 0
	}
	if recommendation.Confidence > 1 {
		recommendation.Confidence = 1
	}

	return recommendation
}

// adaptAgentForProviders adapts an agent configuration to available providers
func (ap *AgentPresets) adaptAgentForProviders(ctx context.Context, agent config.AgentConfig, providers []ConfiguredProvider) (*config.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("adaptAgentForProviders")
	}

	adapted := agent // Copy the agent
	
	// If agent already has a specific provider that's available, keep it
	if agent.Provider != "auto" {
		for _, provider := range providers {
			if provider.Name == agent.Provider {
				// Check if the model is available
				for _, model := range provider.Models {
					if model.Name == agent.Model {
						return &adapted, nil // Already compatible
					}
				}
			}
		}
	}

	// Need to adapt - find best matching provider and model
	caps := ap.analyzeProviderCapabilities(providers)
	
	var selectedModel ModelSelection
	switch agent.Type {
	case "manager":
		selectedModel = caps.BestManager
	case "specialist":
		selectedModel = caps.BestSpecialist
	default:
		selectedModel = caps.BestWorker
	}

	if selectedModel.Provider == "" {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no suitable model found for agent type '%s'", agent.Type).
			WithComponent("AgentPresets").
			WithOperation("adaptAgentForProviders")
	}

	adapted.Provider = selectedModel.Provider
	adapted.Model = selectedModel.Model.Name
	adapted.CostMagnitude = selectedModel.Model.CostMagnitude
	adapted.ContextWindow = selectedModel.Model.ContextWindow

	return &adapted, nil
}

// Helper methods

func (ap *AgentPresets) sortRecommendationsByConfidence(recommendations []*PresetRecommendation) {
	// Simple bubble sort by confidence (descending)
	n := len(recommendations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if recommendations[j].Confidence < recommendations[j+1].Confidence {
				recommendations[j], recommendations[j+1] = recommendations[j+1], recommendations[j]
			}
		}
	}
}

func (ap *AgentPresets) getProjectCategory(context *ProjectContext) PresetCategory {
	if context == nil {
		return PresetCategoryGeneral
	}

	switch strings.ToLower(context.ProjectType) {
	case "web", "webapp", "website":
		return PresetCategoryWeb
	case "api", "microservice", "service":
		return PresetCategoryAPI
	case "cli", "tool", "command":
		return PresetCategoryCLI
	case "data", "analytics", "ml", "ai":
		return PresetCategoryData
	default:
		return PresetCategoryGeneral
	}
}

func (ap *AgentPresets) hasProvider(caps *ProviderCapabilities, providerName string) bool {
	// This is a simplified check - in a real implementation, you'd check the actual providers
	return caps.HasCloud || caps.HasLocal
}

func (ap *AgentPresets) isBetterManagerModel(a, b ModelSelection) bool {
	// Prefer models with larger context windows for managers
	if a.Model.ContextWindow > b.Model.ContextWindow {
		return true
	}
	// Prefer recommended models
	if a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	return false
}

func (ap *AgentPresets) isBetterWorkerModel(a, b ModelSelection) bool {
	// Prefer cost-effective models for workers
	if a.CostEffective && !b.CostEffective {
		return true
	}
	// Among cost-effective models, prefer recommended
	if a.CostEffective == b.CostEffective && a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	return false
}

func (ap *AgentPresets) isBetterSpecialistModel(a, b ModelSelection) bool {
	// For specialists, prefer higher-capability models
	if a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	if a.Model.ContextWindow > b.Model.ContextWindow {
		return true
	}
	return false
}