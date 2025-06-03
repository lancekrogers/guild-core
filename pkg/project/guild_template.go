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
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Version     string         `yaml:"version"`
	Manager     ManagerConfig  `yaml:"manager"`
	Agents      []AgentConfig  `yaml:"agents"`
	Metadata    Metadata       `yaml:"metadata,omitempty"`
}

type ManagerConfig struct {
	Default  string   `yaml:"default"`
	Fallback []string `yaml:"fallback,omitempty"`
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
				MaxTokens:   4096,
				Temperature: 0.7,
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
					"file_edit",
					"shell_execute",
					"file_read",
				},
				MaxTokens:   4096,
				Temperature: 0.3,
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
					"web_search",
					"corpus_query",
					"file_read",
				},
				MaxTokens:   4096,
				Temperature: 0.5,
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