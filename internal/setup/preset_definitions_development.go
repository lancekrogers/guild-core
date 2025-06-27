// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"github.com/lancekrogers/guild/pkg/config"
)

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
				ID:            "dev-lead",
				Name:          "Development Lead",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Experienced team lead who guides development projects",
				Capabilities:  []string{"project_management", "code_review", "team_coordination"},
				MaxTokens:     3500,
				Temperature:   0.2,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "8+ years leading development teams",
					GuildRank:  "Senior Guild Master",
					Philosophy: "Quality code is written once, maintained forever",
				},
			},
			{
				ID:            "dev-engineer",
				Name:          "Senior Engineer",
				Type:          "worker",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Experienced engineer who delivers high-quality implementations",
				Capabilities:  []string{"full_stack_development", "testing", "debugging"},
				MaxTokens:     3000,
				Temperature:   0.3,
				CostMagnitude: 2,
				Backstory: &config.Backstory{
					Experience: "6+ years in full-stack development",
					GuildRank:  "Senior Artisan",
					Philosophy: "Test-driven development leads to robust solutions",
				},
			},
			{
				ID:            "dev-qa",
				Name:          "Quality Guardian",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Quality specialist focused on testing and code review",
				Capabilities:  []string{"quality_assurance", "automated_testing", "code_analysis"},
				MaxTokens:     2500,
				Temperature:   0.1,
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
				ID:            "web-lead",
				Name:          "Web Development Lead",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Experienced web team lead with full-stack expertise",
				Capabilities:  []string{"web_architecture", "team_coordination", "project_planning"},
				MaxTokens:     3500,
				Temperature:   0.2,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:        "web_development",
					Technologies:  []string{"React", "Node.js", "TypeScript", "Next.js"},
					Methodologies: []string{"Agile", "TDD", "Component-driven"},
				},
			},
			{
				ID:            "web-frontend",
				Name:          "Frontend Specialist",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Expert in modern frontend development and user experience",
				Capabilities:  []string{"frontend_development", "ui_design", "responsive_design"},
				MaxTokens:     3000,
				Temperature:   0.3,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "frontend",
					Technologies: []string{"React", "Vue", "CSS", "JavaScript", "TypeScript"},
					Principles:   []string{"Mobile-first", "Accessibility", "Performance"},
				},
			},
			{
				ID:            "web-backend",
				Name:          "Backend Specialist",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Expert in backend services and API development",
				Capabilities:  []string{"backend_development", "api_design", "database_design"},
				MaxTokens:     3000,
				Temperature:   0.3,
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
				ID:            "api-architect",
				Name:          "API Architect",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Expert in API design and microservices architecture",
				Capabilities:  []string{"api_design", "microservices_architecture", "documentation"},
				MaxTokens:     3500,
				Temperature:   0.2,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:       "api_development",
					Technologies: []string{"REST", "GraphQL", "gRPC", "OpenAPI"},
					Principles:   []string{"API-first", "Idempotency", "Versioning"},
				},
			},
			{
				ID:            "api-developer",
				Name:          "API Developer",
				Type:          "worker",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Skilled developer focused on API implementation and testing",
				Capabilities:  []string{"api_implementation", "integration_testing", "performance_optimization"},
				MaxTokens:     3000,
				Temperature:   0.3,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:        "backend",
					Technologies:  []string{"Go", "Python", "Docker", "Kubernetes"},
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
				ID:            "cli-designer",
				Name:          "CLI Designer",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Expert in CLI design and user experience",
				Capabilities:  []string{"cli_design", "user_experience", "interface_planning"},
				MaxTokens:     3000,
				Temperature:   0.2,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:       "cli_development",
					Technologies: []string{"Go", "Cobra", "Viper", "Bash"},
					Principles:   []string{"Unix Philosophy", "User-centric Design"},
				},
			},
			{
				ID:            "cli-developer",
				Name:          "CLI Developer",
				Type:          "worker",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Skilled developer focused on CLI implementation",
				Capabilities:  []string{"cli_implementation", "cross_platform_development", "testing"},
				MaxTokens:     2500,
				Temperature:   0.3,
				CostMagnitude: 1,
				Specialization: &config.Specialization{
					Domain:        "systems_programming",
					Technologies:  []string{"Go", "Rust", "Shell Scripting"},
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
				ID:            "data-scientist",
				Name:          "Lead Data Scientist",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Expert data scientist leading analytics projects",
				Capabilities:  []string{"data_science", "statistical_analysis", "project_coordination"},
				MaxTokens:     4000,
				Temperature:   0.2,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:        "data_science",
					Technologies:  []string{"Python", "R", "Jupyter", "Pandas", "Scikit-learn"},
					Methodologies: []string{"CRISP-DM", "Agile Analytics"},
				},
			},
			{
				ID:            "ml-engineer",
				Name:          "ML Engineer",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Machine learning engineer focused on model development",
				Capabilities:  []string{"machine_learning", "model_training", "feature_engineering"},
				MaxTokens:     3500,
				Temperature:   0.3,
				CostMagnitude: 3,
				Specialization: &config.Specialization{
					Domain:       "machine_learning",
					Technologies: []string{"TensorFlow", "PyTorch", "MLflow", "Docker"},
					Principles:   []string{"Model Versioning", "A/B Testing", "Monitoring"},
				},
			},
			{
				ID:            "data-analyst",
				Name:          "Data Analyst",
				Type:          "worker",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Data analyst focused on insights and visualization",
				Capabilities:  []string{"data_analysis", "visualization", "reporting"},
				MaxTokens:     3000,
				Temperature:   0.4,
				CostMagnitude: 2,
				Specialization: &config.Specialization{
					Domain:        "analytics",
					Technologies:  []string{"SQL", "Tableau", "Power BI", "Excel"},
					Methodologies: []string{"Exploratory Data Analysis", "Statistical Testing"},
				},
			},
		},
	}
}
