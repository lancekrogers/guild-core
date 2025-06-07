package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GuildTemplate provides default guild configurations for new projects
// Moved from pkg/config to break circular dependency

// GuildConfig represents the structure needed for templates
// This is a minimal version to avoid importing from config package
type GuildConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Version     string          `yaml:"version"`
	Manager     ManagerConfig   `yaml:"manager"`
	Providers   ProvidersConfig `yaml:"providers,omitempty"`
	Agents      []AgentConfig   `yaml:"agents"`
	Metadata    Metadata        `yaml:"metadata,omitempty"`
}

type ManagerConfig struct {
	Default  string   `yaml:"default"`
	Fallback []string `yaml:"fallback,omitempty"`
}

type ProvidersConfig struct {
	OpenAI     ProviderSettings `yaml:"openai,omitempty"`
	Anthropic  ProviderSettings `yaml:"anthropic,omitempty"`
	Ollama     ProviderSettings `yaml:"ollama,omitempty"`
	ClaudeCode ProviderSettings `yaml:"claude_code,omitempty"`
	DeepSeek   ProviderSettings `yaml:"deepseek,omitempty"`
	DeepInfra  ProviderSettings `yaml:"deepinfra,omitempty"`
	Ora        ProviderSettings `yaml:"ora,omitempty"`
}

type ProviderSettings struct {
	BaseURL  string            `yaml:"base_url,omitempty"`  // Custom base URL (for self-hosted)
	Settings map[string]string `yaml:"settings,omitempty"`  // Additional provider settings
}

type AgentConfig struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model"`
	Description  string   `yaml:"description,omitempty"`
	Capabilities []string `yaml:"capabilities"`
	Tools        []string `yaml:"tools,omitempty"`
	MaxTokens    int      `yaml:"max_tokens,omitempty"`
	Temperature  float64  `yaml:"temperature,omitempty"`

	// Enhanced configuration for intelligent assignment
	CostMagnitude  int    `yaml:"cost_magnitude,omitempty"`   // Fibonacci cost scale: 0=bash, 1=cheap API, 2,3,5,8=expensive models
	ContextWindow  int    `yaml:"context_window,omitempty"`   // Context window size in tokens (auto-detected if 0)
	ContextReset   string `yaml:"context_reset,omitempty"`    // "truncate" or "summarize" when context exceeds window
}

type Metadata struct {
	Tags []string `yaml:"tags,omitempty"`
}

// DefaultGuildTemplate returns a default guild configuration template
func DefaultGuildTemplate() *GuildConfig {
	return &GuildConfig{
		Name:        "MyGuild",
		Description: "A guild of AI agents working together",
		Version:     "1.0.0",
		Manager: ManagerConfig{
			Default: "orchestrator",
			Fallback: []string{"analyst", "coder"},
		},
		Providers: ProvidersConfig{
			Ollama: ProviderSettings{
				BaseURL: "http://localhost:11434", // Default Ollama URL
			},
			// Note: API keys are configured via environment variables only (OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.)
		},
		Agents: []AgentConfig{
			{
				ID:          "orchestrator",
				Name:        "Master Orchestrator",
				Type:        "manager",
				Provider:    "anthropic",
				Model:       "claude-3-opus-20240229",
				Description: "Primary manager agent for planning and task decomposition",
				Capabilities: []string{
					"planning",
					"task_decomposition",
					"architecture",
					"coordination",
					"analysis",
				},
				Tools: []string{
					"shell",
					"file_system",
				},
				MaxTokens:     4096,
				Temperature:   0.7,
				CostMagnitude: 8,
				ContextWindow: 200000,
				ContextReset:  "summarize",
			},
			{
				ID:          "coder",
				Name:        "Code Artisan",
				Type:        "worker",
				Provider:    "openai",
				Model:       "gpt-4-turbo-preview",
				Description: "Specialist in coding, debugging, and technical implementation",
				Capabilities: []string{
					"coding",
					"debugging",
					"testing",
					"refactoring",
					"code_review",
				},
				Tools: []string{
					"file_system",
					"shell",
				},
				MaxTokens:     4096,
				Temperature:   0.3,
				CostMagnitude: 5,
				ContextWindow: 128000,
				ContextReset:  "truncate",
			},
			{
				ID:          "analyst",
				Name:        "Research Analyst",
				Type:        "worker",
				Provider:    "anthropic",
				Model:       "claude-3-sonnet-20240229",
				Description: "Expert in research, analysis, and documentation",
				Capabilities: []string{
					"research",
					"analysis",
					"documentation",
					"summarization",
					"data_analysis",
				},
				Tools: []string{
					"corpus",
					"file_system",
				},
				MaxTokens:     4096,
				Temperature:   0.5,
				CostMagnitude: 3,
				ContextWindow: 200000,
				ContextReset:  "summarize",
			},
		},
		Metadata: Metadata{
			Tags: []string{"default", "template"},
		},
	}
}

// SaveGuildConfig saves a guild configuration to the specified project path
func SaveGuildConfig(projectPath string, config *GuildConfig) error {
	// For now, we'll use a simple YAML marshaling approach
	// This avoids importing the config package
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal guild config: %w", err)
	}

	guildPath := filepath.Join(projectPath, "guild.yaml")
	if err := os.WriteFile(guildPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write guild config: %w", err)
	}

	return nil
}
