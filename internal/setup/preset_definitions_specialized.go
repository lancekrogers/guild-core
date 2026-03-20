// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"github.com/lancekrogers/guild-core/pkg/config"
)

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
				ID:            "prod-director",
				Name:          "Technical Director",
				Type:          "manager",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Senior technical leader overseeing production systems",
				Capabilities:  []string{"strategic_planning", "risk_management", "technical_leadership"},
				MaxTokens:     5000,
				Temperature:   0.1,
				CostMagnitude: 5,
				Backstory: &config.Backstory{
					Experience: "15+ years in enterprise technical leadership",
					GuildRank:  "Guild Director",
					Philosophy: "Production systems must be reliable, secure, and maintainable",
				},
			},
			{
				ID:            "prod-architect",
				Name:          "Principal Architect",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Senior architect focused on scalable, production-ready systems",
				Capabilities:  []string{"enterprise_architecture", "scalability_design", "system_integration"},
				MaxTokens:     4000,
				Temperature:   0.2,
				CostMagnitude: 5,
				Backstory: &config.Backstory{
					Experience: "12+ years designing enterprise systems",
					GuildRank:  "Principal Architect",
					Philosophy: "Architecture should enable growth while maintaining stability",
				},
			},
			{
				ID:            "prod-security",
				Name:          "Security Specialist",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "Security expert ensuring robust protection and compliance",
				Capabilities:  []string{"security_analysis", "compliance_checking", "threat_modeling"},
				MaxTokens:     3500,
				Temperature:   0.1,
				CostMagnitude: 3,
				Backstory: &config.Backstory{
					Experience: "10+ years in cybersecurity and compliance",
					GuildRank:  "Master Guardian",
					Philosophy: "Security is not optional - it's foundational",
				},
			},
			{
				ID:            "prod-ops",
				Name:          "Operations Engineer",
				Type:          "specialist",
				Provider:      "auto",
				Model:         "auto",
				Description:   "DevOps specialist managing deployments and infrastructure",
				Capabilities:  []string{"infrastructure_management", "ci_cd", "monitoring"},
				MaxTokens:     3000,
				Temperature:   0.2,
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
				ID:            "claude-strategic-lead",
				Name:          "Claude Strategic Lead",
				Type:          "manager",
				Provider:      "claude_code",
				Model:         "claude-3-opus-20240229",
				Description:   "Strategic leader leveraging Claude's advanced reasoning capabilities",
				Capabilities:  []string{"strategic_reasoning", "complex_planning", "claude_integration"},
				MaxTokens:     4000,
				Temperature:   0.1,
				CostMagnitude: 8,
				ContextWindow: 200000,
				Backstory: &config.Backstory{
					Experience: "Advanced AI reasoning with deep strategic thinking",
					Philosophy: "Leverage the full power of Claude's reasoning for optimal outcomes",
					GuildRank:  "Claude Master",
				},
			},
			{
				ID:            "claude-development-expert",
				Name:          "Claude Development Expert",
				Type:          "specialist",
				Provider:      "claude_code",
				Model:         "claude-3-sonnet-20240229",
				Description:   "Expert developer with Claude Code integration and advanced coding skills",
				Capabilities:  []string{"advanced_coding", "claude_code_integration", "code_generation"},
				MaxTokens:     3500,
				Temperature:   0.2,
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
				ID:            "local-coordinator",
				Name:          "Local Development Coordinator",
				Type:          "manager",
				Provider:      "ollama",
				Model:         "llama3:latest",
				Description:   "Coordinator using local models for privacy-sensitive projects",
				Capabilities:  []string{"local_coordination", "privacy_focused_planning", "offline_development"},
				MaxTokens:     2000,
				Temperature:   0.2,
				CostMagnitude: 0,
				Backstory: &config.Backstory{
					Experience: "Specialized in local development environments",
					Philosophy: "Privacy and control are paramount in development",
					GuildRank:  "Privacy Guardian",
				},
			},
			{
				ID:            "local-coder",
				Name:          "Local Code Artisan",
				Type:          "worker",
				Provider:      "ollama",
				Model:         "deepseek-coder:latest",
				Description:   "Local coding specialist for privacy-sensitive development",
				Capabilities:  []string{"local_coding", "offline_development", "privacy_focused_implementation"},
				MaxTokens:     2000,
				Temperature:   0.3,
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
				ID:            "multi-strategist",
				Name:          "Multi-Provider Strategist",
				Type:          "manager",
				Provider:      "anthropic", // Prefer Claude for strategic thinking
				Model:         "claude-3-opus-20240229",
				Description:   "Strategic leader leveraging the best reasoning capabilities",
				Capabilities:  []string{"strategic_planning", "provider_optimization", "multi_agent_coordination"},
				MaxTokens:     4000,
				Temperature:   0.1,
				CostMagnitude: 8,
			},
			{
				ID:            "multi-coder",
				Name:          "Versatile Developer",
				Type:          "worker",
				Provider:      "openai", // GPT for coding tasks
				Model:         "gpt-4-turbo-preview",
				Description:   "Versatile developer using optimal models for coding",
				Capabilities:  []string{"coding", "debugging", "implementation"},
				MaxTokens:     3000,
				Temperature:   0.3,
				CostMagnitude: 5,
			},
			{
				ID:            "multi-local",
				Name:          "Local Privacy Specialist",
				Type:          "specialist",
				Provider:      "ollama", // Local for privacy
				Model:         "deepseek-coder:latest",
				Description:   "Local specialist for privacy-sensitive tasks",
				Capabilities:  []string{"privacy_tasks", "local_processing", "sensitive_data_handling"},
				MaxTokens:     2000,
				Temperature:   0.3,
				CostMagnitude: 0,
			},
		},
	}
}
