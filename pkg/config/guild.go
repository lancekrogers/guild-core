package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	
	// Enhanced configuration for intelligent assignment
	CostMagnitude  int    `yaml:"cost_magnitude,omitempty"`   // Fibonacci cost scale: 0=bash, 1=cheap API, 2,3,5,8=expensive models
	ContextWindow  int    `yaml:"context_window,omitempty"`   // Context window size in tokens (auto-detected if 0)
	ContextReset   string `yaml:"context_reset,omitempty"`    // "truncate" or "summarize" when context exceeds window
	
	Settings       map[string]string `yaml:"settings,omitempty"` // Provider-specific settings
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
	// Model is required unless this is a tool-only agent (cost magnitude 0)
	if a.Model == "" && a.CostMagnitude != 0 {
		return fmt.Errorf("agent model is required (except for tool-only agents with cost_magnitude: 0)")
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

	// Validate cost magnitude (Fibonacci scale)
	if a.CostMagnitude != 0 {
		validCosts := map[int]bool{
			1: true, // Cheap API usage
			2: true, // Low-mid cost
			3: true, // Mid cost
			5: true, // High cost
			8: true, // Most expensive models
		}
		if !validCosts[a.CostMagnitude] {
			return fmt.Errorf("invalid cost_magnitude: %d (must be 0 for bash tools, or Fibonacci values: 1,2,3,5,8)", a.CostMagnitude)
		}
	}

	// Validate context reset behavior
	if a.ContextReset != "" {
		validResets := map[string]bool{
			"truncate":   true,
			"summarize":  true,
		}
		if !validResets[a.ContextReset] {
			return fmt.Errorf("invalid context_reset: %s (must be 'truncate' or 'summarize')", a.ContextReset)
		}
	}

	// Validate context window
	if a.ContextWindow < 0 {
		return fmt.Errorf("context_window must be non-negative (0 for auto-detection)")
	}

	return nil
}

// GetEffectiveCostMagnitude returns the cost magnitude with smart defaults
func (a *AgentConfig) GetEffectiveCostMagnitude() int {
	if a.CostMagnitude != 0 {
		return a.CostMagnitude
	}
	
	// If no model specified, this is likely a tool-only agent
	if a.Model == "" {
		return 0
	}
	
	// Auto-assign based on model characteristics if not specified
	modelLower := strings.ToLower(a.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4"):
		return 5 // High cost
	case strings.Contains(modelLower, "gpt-3.5"):
		return 2 // Low-mid cost  
	case strings.Contains(modelLower, "claude-3-opus"):
		return 8 // Most expensive
	case strings.Contains(modelLower, "claude-3-sonnet"):
		return 3 // Mid cost
	case strings.Contains(modelLower, "claude-3-haiku"):
		return 1 // Cheap
	case strings.Contains(modelLower, "ollama") || strings.Contains(modelLower, "local"):
		return 0 // Free local models
	default:
		return 1 // Default to cheap for unknown models
	}
}

// GetEffectiveContextWindow returns the context window with auto-detection
func (a *AgentConfig) GetEffectiveContextWindow() int {
	if a.ContextWindow > 0 {
		return a.ContextWindow
	}
	
	// Auto-detect based on known models
	modelLower := strings.ToLower(a.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4-turbo"):
		return 128000
	case strings.Contains(modelLower, "gpt-4"):
		return 32000
	case strings.Contains(modelLower, "gpt-3.5"):
		return 16000
	case strings.Contains(modelLower, "claude-3"):
		return 200000
	case strings.Contains(modelLower, "claude-2"):
		return 100000
	default:
		return 8000 // Conservative default
	}
}

// GetEffectiveContextReset returns the context reset behavior with smart defaults
func (a *AgentConfig) GetEffectiveContextReset() string {
	if a.ContextReset != "" {
		return a.ContextReset
	}
	
	// Default based on agent type and capabilities
	switch a.Type {
	case "manager":
		return "summarize" // Managers need to preserve context
	case "worker":
		return "truncate" // Workers can restart fresh
	default:
		return "truncate" // Conservative default
	}
}

// IsToolOnlyAgent returns true if this agent only uses tools (no LLM calls)
func (a *AgentConfig) IsToolOnlyAgent() bool {
	return a.CostMagnitude == 0
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