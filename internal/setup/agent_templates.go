// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

// AgentTemplate defines a lightweight template for generating agent configurations
type AgentTemplate struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model,omitempty"`
	Description  string   `yaml:"description"`
	Capabilities []string `yaml:"capabilities"`
	Tools        []string `yaml:"tools,omitempty"`

	// Optional fields with defaults
	MaxTokens     int     `yaml:"max_tokens,omitempty"`
	Temperature   float64 `yaml:"temperature,omitempty"`
	CostMagnitude int     `yaml:"cost_magnitude,omitempty"`
	ContextWindow int     `yaml:"context_window,omitempty"`

	// Simple backstory fields - all optional
	Experience string `yaml:"experience,omitempty"`
	Expertise  string `yaml:"expertise,omitempty"`
	Philosophy string `yaml:"philosophy,omitempty"`
}

// AgentTemplateGenerator handles generation of agent configuration files from templates
type AgentTemplateGenerator struct {
	templates map[string]AgentTemplate
}

// NewAgentTemplateGenerator creates a new template generator
func NewAgentTemplateGenerator() *AgentTemplateGenerator {
	return &AgentTemplateGenerator{
		templates: getBuiltInTemplates(),
	}
}

// GenerateAgentConfig converts a template to a full agent configuration
func (g *AgentTemplateGenerator) GenerateAgentConfig(template AgentTemplate) (*config.AgentConfig, error) {
	// Validate required fields
	if template.ID == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent ID is required", nil).
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentConfig")
	}
	if template.Name == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent name is required", nil).
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentConfig")
	}
	if template.Type == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent type is required", nil).
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentConfig")
	}
	if template.Provider == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent provider is required", nil).
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentConfig")
	}

	// Create base configuration
	agentConfig := &config.AgentConfig{
		ID:           template.ID,
		Name:         template.Name,
		Type:         template.Type,
		Provider:     template.Provider,
		Model:        template.Model,
		Description:  template.Description,
		Capabilities: template.Capabilities,
		Tools:        template.Tools,
	}

	// Apply defaults
	agentConfig.MaxTokens = template.MaxTokens
	if agentConfig.MaxTokens == 0 {
		agentConfig.MaxTokens = getDefaultMaxTokens(template.Type)
	}

	agentConfig.Temperature = template.Temperature
	if agentConfig.Temperature == 0 {
		agentConfig.Temperature = getDefaultTemperature(template.Type)
	}

	agentConfig.CostMagnitude = template.CostMagnitude
	agentConfig.ContextWindow = template.ContextWindow

	// Only create backstory if any fields are provided
	if template.Experience != "" || template.Expertise != "" || template.Philosophy != "" {
		agentConfig.Backstory = &config.Backstory{
			Experience: template.Experience,
			Expertise:  template.Expertise,
			Philosophy: template.Philosophy,
			// Leave other fields empty - user can fill them later if desired
		}
	}

	// Generate appropriate system prompt based on type and fields
	agentConfig.SystemPrompt = g.generateSystemPrompt(template)

	return agentConfig, nil
}

// GenerateAgentFile creates an agent configuration file from a template
func (g *AgentTemplateGenerator) GenerateAgentFile(ctx context.Context, projectPath string, template AgentTemplate) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentFile")
	}
	// Generate the configuration
	agentConfig, err := g.GenerateAgentConfig(template)
	if err != nil {
		return err
	}

	// Ensure agents directory exists
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentFile")
	}

	// Generate filename from agent ID
	filename := fmt.Sprintf("%s.yml", strings.ToLower(template.ID))
	filePath := filepath.Join(agentsDir, filename)

	// Marshal to YAML
	data, err := yaml.Marshal(agentConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentFile")
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent file").
			WithComponent("AgentTemplateGenerator").
			WithOperation("GenerateAgentFile")
	}

	return nil
}

// GetTemplate returns a template by name
func (g *AgentTemplateGenerator) GetTemplate(name string) (AgentTemplate, bool) {
	template, exists := g.templates[name]
	return template, exists
}

// ListTemplates returns all available template names
func (g *AgentTemplateGenerator) ListTemplates() []string {
	names := make([]string, 0, len(g.templates))
	for name := range g.templates {
		names = append(names, name)
	}
	return names
}

// CreateCustomTemplate creates a minimal template with user-provided fields
func (g *AgentTemplateGenerator) CreateCustomTemplate(
	id, name, agentType, provider, model, description string,
	capabilities []string,
) AgentTemplate {
	return AgentTemplate{
		ID:           id,
		Name:         name,
		Type:         agentType,
		Provider:     provider,
		Model:        model,
		Description:  description,
		Capabilities: capabilities,
		// All other fields left empty for user to optionally fill
	}
}

// generateSystemPrompt creates a simple system prompt based on template fields
func (g *AgentTemplateGenerator) generateSystemPrompt(template AgentTemplate) string {
	var parts []string

	// Base prompt
	parts = append(parts, fmt.Sprintf("You are %s, a %s agent.", template.Name, template.Type))

	// Add description if provided
	if template.Description != "" {
		parts = append(parts, template.Description)
	}

	// Add capabilities
	if len(template.Capabilities) > 0 {
		parts = append(parts, fmt.Sprintf("\nYour capabilities include: %s.",
			strings.Join(template.Capabilities, ", ")))
	}

	// Add experience if provided
	if template.Experience != "" {
		parts = append(parts, fmt.Sprintf("\nExperience: %s", template.Experience))
	}

	// Add expertise if provided
	if template.Expertise != "" {
		parts = append(parts, fmt.Sprintf("\nExpertise: %s", template.Expertise))
	}

	// Add philosophy if provided
	if template.Philosophy != "" {
		parts = append(parts, fmt.Sprintf("\nPhilosophy: %s", template.Philosophy))
	}

	// Add standard closing
	parts = append(parts, "\nApproach tasks methodically and communicate clearly.")

	return strings.Join(parts, " ")
}

// getDefaultMaxTokens returns sensible defaults based on agent type
func getDefaultMaxTokens(agentType string) int {
	switch agentType {
	case "manager":
		return 4000
	case "specialist":
		return 3500
	case "worker":
		return 3000
	default:
		return 2500
	}
}

// getDefaultTemperature returns sensible defaults based on agent type
func getDefaultTemperature(agentType string) float64 {
	switch agentType {
	case "manager":
		return 0.1 // Very consistent for management decisions
	case "specialist":
		return 0.3 // Some creativity within expertise
	case "worker":
		return 0.4 // Balanced creativity
	default:
		return 0.5
	}
}

// getBuiltInTemplates returns pre-defined lightweight templates
func getBuiltInTemplates() map[string]AgentTemplate {
	return map[string]AgentTemplate{
		// Claude Code templates
		"claude-code-manager": {
			ID:           "claude-manager",
			Name:         "Claude Manager",
			Type:         "manager",
			Provider:     "claude_code",
			Model:        "claude-3-sonnet-20240229",
			Description:  "Strategic manager for coordinating development tasks",
			Capabilities: []string{"task_decomposition", "agent_coordination", "strategic_planning"},
			Tools:        []string{"task_planner", "agent_coordinator"},
		},
		"claude-code-developer": {
			ID:           "claude-developer",
			Name:         "Claude Developer",
			Type:         "worker",
			Provider:     "claude_code",
			Model:        "claude-3-sonnet-20240229",
			Description:  "Full-stack developer for general coding tasks",
			Capabilities: []string{"coding", "debugging", "testing", "documentation"},
			Tools:        []string{"code_executor", "file_manager", "git_tools"},
		},
		"claude-code-architect": {
			ID:           "claude-architect",
			Name:         "Claude Architect",
			Type:         "specialist",
			Provider:     "claude_code",
			Model:        "claude-3-opus-20240229",
			Description:  "System architect for complex design decisions",
			Capabilities: []string{"architecture_design", "system_analysis", "performance_optimization"},
			Tools:        []string{"diagram_generator", "code_analyzer"},
			Philosophy:   "Build systems that are maintainable, scalable, and elegant",
		},

		// Ollama templates
		"ollama-coder": {
			ID:            "local-coder",
			Name:          "Local Code Assistant",
			Type:          "worker",
			Provider:      "ollama",
			Model:         "deepseek-coder:latest",
			Description:   "Local model for privacy-sensitive coding tasks",
			Capabilities:  []string{"coding", "code_review", "refactoring"},
			Tools:         []string{"code_executor", "file_manager"},
			CostMagnitude: 0, // Free local model
		},
		"ollama-analyst": {
			ID:            "local-analyst",
			Name:          "Local Analyst",
			Type:          "specialist",
			Provider:      "ollama",
			Model:         "llama3:latest",
			Description:   "Local model for data analysis and research",
			Capabilities:  []string{"data_analysis", "research", "summarization"},
			CostMagnitude: 0,
		},

		// OpenAI templates
		"openai-developer": {
			ID:           "gpt-developer",
			Name:         "GPT Developer",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-4-turbo-preview",
			Description:  "Versatile developer powered by GPT-4",
			Capabilities: []string{"coding", "problem_solving", "documentation"},
			Tools:        []string{"code_executor", "web_search"},
		},
		"openai-creative": {
			ID:           "gpt-creative",
			Name:         "GPT Creative",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-4",
			Description:  "Creative specialist for design and content",
			Capabilities: []string{"creative_writing", "ui_design", "brainstorming"},
			Temperature:  0.7, // Higher for creativity
		},

		// Anthropic templates
		"anthropic-researcher": {
			ID:           "claude-researcher",
			Name:         "Claude Researcher",
			Type:         "specialist",
			Provider:     "anthropic",
			Model:        "claude-3-opus-20240229",
			Description:  "Deep research and analysis specialist",
			Capabilities: []string{"research", "analysis", "report_generation"},
			Tools:        []string{"web_search", "document_analyzer"},
			Expertise:    "Thorough research with attention to detail and accuracy",
		},

		// Generic templates
		"generic-manager": {
			ID:           "manager",
			Name:         "Task Manager",
			Type:         "manager",
			Provider:     "", // User must specify
			Model:        "", // User must specify
			Description:  "Coordinates tasks and manages agent teams",
			Capabilities: []string{"task_decomposition", "agent_coordination"},
		},
		"generic-worker": {
			ID:           "worker",
			Name:         "General Worker",
			Type:         "worker",
			Provider:     "", // User must specify
			Model:        "", // User must specify
			Description:  "Handles general development tasks",
			Capabilities: []string{"general_tasks", "implementation"},
		},
	}
}

// ProviderTemplates returns all templates for a specific provider
func (g *AgentTemplateGenerator) ProviderTemplates(provider string) []AgentTemplate {
	var templates []AgentTemplate
	for _, template := range g.templates {
		if template.Provider == provider || template.Provider == "" {
			templates = append(templates, template)
		}
	}
	return templates
}

// QuickSetup generates a minimal set of agents for a provider
func (g *AgentTemplateGenerator) QuickSetup(ctx context.Context, projectPath, provider, managerModel, workerModel string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentTemplateGenerator").
			WithOperation("QuickSetup")
	}
	// Create manager
	managerTemplate := g.CreateCustomTemplate(
		"manager",
		"Guild Manager",
		"manager",
		provider,
		managerModel,
		"Manages tasks and coordinates agent work",
		[]string{"task_decomposition", "agent_coordination", "planning"},
	)

	if err := g.GenerateAgentFile(ctx, projectPath, managerTemplate); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create manager agent").
			WithComponent("AgentTemplateGenerator").
			WithOperation("QuickSetup")
	}

	// Create worker
	workerTemplate := g.CreateCustomTemplate(
		"worker-1",
		"Primary Worker",
		"worker",
		provider,
		workerModel,
		"Executes tasks assigned by the manager",
		[]string{"coding", "testing", "documentation", "general_tasks"},
	)

	if err := g.GenerateAgentFile(ctx, projectPath, workerTemplate); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create worker agent").
			WithComponent("AgentTemplateGenerator").
			WithOperation("QuickSetup")
	}

	return nil
}
