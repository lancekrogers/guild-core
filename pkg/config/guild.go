package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GuildConfig represents the configuration for a Guild (team of agents)
type GuildConfig struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Version     string        `yaml:"version,omitempty"`
	Manager     ManagerConfig `yaml:"manager"`
	Agents      []AgentConfig `yaml:"agents"`
	Metadata    Metadata      `yaml:"metadata,omitempty"`
}

// ManagerConfig configures the manager agent selection
type ManagerConfig struct {
	Default  string            `yaml:"default"`           // Default manager agent ID
	Override string            `yaml:"override,omitempty"` // User-specified override
	Fallback []string          `yaml:"fallback,omitempty"` // Fallback chain if default unavailable
	Settings map[string]string `yaml:"settings,omitempty"` // Manager-specific settings
}

// AgentConfig represents configuration for a single agent
type AgentConfig struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Type         string            `yaml:"type"` // manager, worker, specialist
	Provider     string            `yaml:"provider"`
	Model        string            `yaml:"model"`
	Description  string            `yaml:"description,omitempty"`
	Capabilities []string          `yaml:"capabilities"`
	Tools        []string          `yaml:"tools,omitempty"`
	MaxTokens    int               `yaml:"max_tokens,omitempty"`
	Temperature  float64           `yaml:"temperature,omitempty"`
	Settings     map[string]string `yaml:"settings,omitempty"` // Provider-specific settings
}

// Metadata contains optional metadata about the guild
type Metadata struct {
	Author      string   `yaml:"author,omitempty"`
	CreatedAt   string   `yaml:"created_at,omitempty"`
	UpdatedAt   string   `yaml:"updated_at,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// LoadGuildConfig loads guild configuration from a project directory
func LoadGuildConfig(projectPath string) (*GuildConfig, error) {
	configPath := filepath.Join(projectPath, ".guild", "guild.yaml")
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Also check for guild.yml
		altPath := filepath.Join(projectPath, ".guild", "guild.yml")
		if _, err := os.Stat(altPath); err == nil {
			configPath = altPath
		} else {
			return nil, fmt.Errorf("guild configuration not found at %s", configPath)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read guild config: %w", err)
	}

	var config GuildConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse guild config: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid guild configuration: %w", err)
	}

	return &config, nil
}

// SaveGuildConfig saves guild configuration to a project directory
func SaveGuildConfig(projectPath string, config *GuildConfig) error {
	configPath := filepath.Join(projectPath, ".guild", "guild.yaml")
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create guild directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal guild config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write guild config: %w", err)
	}

	return nil
}

// Validate validates the guild configuration
func (g *GuildConfig) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("guild name is required")
	}

	if len(g.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	// Validate manager configuration
	if g.Manager.Default != "" {
		found := false
		for _, agent := range g.Agents {
			if agent.ID == g.Manager.Default {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("default manager agent '%s' not found in agents list", g.Manager.Default)
		}
	}

	// Validate each agent
	agentIDs := make(map[string]bool)
	for i, agent := range g.Agents {
		if err := agent.Validate(); err != nil {
			return fmt.Errorf("agent[%d] validation failed: %w", i, err)
		}
		if agentIDs[agent.ID] {
			return fmt.Errorf("duplicate agent ID: %s", agent.ID)
		}
		agentIDs[agent.ID] = true
	}

	return nil
}

// Validate validates an agent configuration
func (a *AgentConfig) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.Type == "" {
		return fmt.Errorf("agent type is required")
	}
	if a.Provider == "" {
		return fmt.Errorf("agent provider is required")
	}
	if a.Model == "" {
		return fmt.Errorf("agent model is required")
	}
	
	// Validate type
	validTypes := map[string]bool{
		"manager":    true,
		"worker":     true,
		"specialist": true,
	}
	if !validTypes[a.Type] {
		return fmt.Errorf("invalid agent type: %s (must be manager, worker, or specialist)", a.Type)
	}

	// Validate capabilities
	if len(a.Capabilities) == 0 {
		return fmt.Errorf("agent must have at least one capability")
	}

	return nil
}

// GetManagerAgent returns the manager agent configuration
func (g *GuildConfig) GetManagerAgent() (*AgentConfig, error) {
	// Check for override first
	managerID := g.Manager.Default
	if g.Manager.Override != "" {
		managerID = g.Manager.Override
	}

	// If no manager specified, find first agent with type "manager"
	if managerID == "" {
		for _, agent := range g.Agents {
			if agent.Type == "manager" {
				return &agent, nil
			}
		}
		// If no manager type, use first agent
		if len(g.Agents) > 0 {
			return &g.Agents[0], nil
		}
		return nil, fmt.Errorf("no manager agent configured")
	}

	// Find specified manager
	for _, agent := range g.Agents {
		if agent.ID == managerID {
			return &agent, nil
		}
	}

	// Try fallback chain
	for _, fallbackID := range g.Manager.Fallback {
		for _, agent := range g.Agents {
			if agent.ID == fallbackID {
				return &agent, nil
			}
		}
	}

	return nil, fmt.Errorf("manager agent '%s' not found", managerID)
}

// GetAgentByCapability returns agents that have a specific capability
func (g *GuildConfig) GetAgentsByCapability(capability string) []AgentConfig {
	var agents []AgentConfig
	for _, agent := range g.Agents {
		for _, cap := range agent.Capabilities {
			if cap == capability {
				agents = append(agents, agent)
				break
			}
		}
	}
	return agents
}

// GetAgentByID returns an agent by its ID
func (g *GuildConfig) GetAgentByID(id string) (*AgentConfig, error) {
	for _, agent := range g.Agents {
		if agent.ID == id {
			return &agent, nil
		}
	}
	return nil, fmt.Errorf("agent with ID '%s' not found", id)
}

// HasCapability checks if an agent has a specific capability
func (a *AgentConfig) HasCapability(capability string) bool {
	for _, cap := range a.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// HasTool checks if an agent has access to a specific tool
func (a *AgentConfig) HasTool(tool string) bool {
	for _, t := range a.Tools {
		if t == tool {
			return true
		}
	}
	return false
}