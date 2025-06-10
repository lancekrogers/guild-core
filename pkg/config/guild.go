package config

import (
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project/global"
)

// GuildConfig represents the configuration for a Guild (team of agents)
type GuildConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Version     string          `yaml:"version,omitempty"`
	Manager     ManagerConfig   `yaml:"manager"`
	Storage     StorageConfig   `yaml:"storage,omitempty"`
	Providers   ProvidersConfig `yaml:"providers,omitempty"`
	Agents      []AgentConfig   `yaml:"agents"`
	Metadata    Metadata        `yaml:"metadata,omitempty"`
}

// ManagerConfig configures the manager agent selection
type ManagerConfig struct {
	Default  string            `yaml:"default"`            // Default manager agent ID
	Override string            `yaml:"override,omitempty"` // User-specified override
	Fallback []string          `yaml:"fallback,omitempty"` // Fallback chain if default unavailable
	Settings map[string]string `yaml:"settings,omitempty"` // Manager-specific settings
}

// StorageConfig configures the storage backend
type StorageConfig struct {
	Backend string       `yaml:"backend,omitempty"` // "sqlite" (default: "sqlite")
	SQLite  SQLiteConfig `yaml:"sqlite,omitempty"`  // SQLite-specific configuration
}

// SQLiteConfig configures SQLite storage backend
type SQLiteConfig struct {
	Path string `yaml:"path,omitempty"` // Path to SQLite database file (default: ".guild/guild.db")
}

// ProvidersConfig contains settings for each provider (no API keys - use environment variables)
type ProvidersConfig struct {
	OpenAI     ProviderSettings `yaml:"openai,omitempty"`
	Anthropic  ProviderSettings `yaml:"anthropic,omitempty"`
	Ollama     ProviderSettings `yaml:"ollama,omitempty"`
	ClaudeCode ProviderSettings `yaml:"claude_code,omitempty"`
	DeepSeek   ProviderSettings `yaml:"deepseek,omitempty"`
	DeepInfra  ProviderSettings `yaml:"deepinfra,omitempty"`
	Ora        ProviderSettings `yaml:"ora,omitempty"`
}

// ProviderSettings contains configuration for a specific provider
// Note: API keys are NOT stored here - use environment variables for security
type ProviderSettings struct {
	BaseURL  string            `yaml:"base_url,omitempty"` // Custom base URL (for self-hosted)
	Settings map[string]string `yaml:"settings,omitempty"` // Additional provider settings
}

// AgentConfig represents configuration for a single agent
type AgentConfig struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"` // manager, worker, specialist
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model"`
	Description  string   `yaml:"description,omitempty"`
	Capabilities []string `yaml:"capabilities"`
	Tools        []string `yaml:"tools,omitempty"`
	MaxTokens    int      `yaml:"max_tokens,omitempty"`
	Temperature  float64  `yaml:"temperature,omitempty"`

	// Prompt configuration
	SystemPrompt   string            `yaml:"system_prompt,omitempty"`   // Direct system prompt
	PromptTemplate string            `yaml:"prompt_template,omitempty"` // Template name for layered prompts
	PromptLayers   map[string]string `yaml:"prompt_layers,omitempty"`   // Layer overrides for layered prompts

	// Enhanced configuration for intelligent assignment
	CostMagnitude int    `yaml:"cost_magnitude,omitempty"` // Fibonacci cost scale: 0=bash, 1=cheap API, 2,3,5,8=expensive models
	ContextWindow int    `yaml:"context_window,omitempty"` // Context window size in tokens (auto-detected if 0)
	ContextReset  string `yaml:"context_reset,omitempty"`  // "truncate" or "summarize" when context exceeds window

	Settings map[string]string `yaml:"settings,omitempty"` // Provider-specific settings
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

// LoadGuildConfig loads guild configuration with global -> project hierarchy
func LoadGuildConfig(projectPath string) (*GuildConfig, error) {
	// Ensure local config exists (global is handled by enhanced loader)
	if err := ensureLocalConfig(projectPath); err != nil {
		return nil, err
	}

	configPath := filepath.Join(projectPath, ".guild", "guild.yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Also check for guild.yml
		altPath := filepath.Join(projectPath, ".guild", "guild.yml")
		if _, err := os.Stat(altPath); err == nil {
			configPath = altPath
		} else {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "guild configuration not found at %s", configPath).
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig")
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read guild config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig")
	}

	var config GuildConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse guild config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig")
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid guild configuration").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig")
	}

	return &config, nil
}

// SaveGuildConfig saves guild configuration to a project directory
func SaveGuildConfig(projectPath string, config *GuildConfig) error {
	configPath := filepath.Join(projectPath, ".guild", "guild.yaml")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guild directory").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	return nil
}

// Validate validates the guild configuration
func (g *GuildConfig) Validate() error {
	if g.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "guild name is required", nil).
			WithComponent("GuildConfig").
			WithOperation("Validate")
	}

	if len(g.Agents) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "at least one agent must be configured", nil).
			WithComponent("GuildConfig").
			WithOperation("Validate")
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
			return gerror.Newf(gerror.ErrCodeValidation, "default manager agent '%s' not found in agents list", g.Manager.Default).
				WithComponent("GuildConfig").
				WithOperation("Validate")
		}
	}

	// Validate each agent
	agentIDs := make(map[string]bool)
	for i, agent := range g.Agents {
		if err := agent.Validate(); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeValidation, "agent[%d] validation failed", i).
				WithComponent("GuildConfig").
				WithOperation("Validate")
		}
		if agentIDs[agent.ID] {
			return gerror.Newf(gerror.ErrCodeValidation, "duplicate agent ID: %s", agent.ID).
				WithComponent("GuildConfig").
				WithOperation("Validate")
		}
		agentIDs[agent.ID] = true
	}

	return nil
}

// Validate validates an agent configuration
func (a *AgentConfig) Validate() error {
	if a.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent ID is required", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}
	if a.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent name is required", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}
	if a.Type == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent type is required", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}
	if a.Provider == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent provider is required", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}
	// Model is required unless this is a tool-only agent (cost magnitude 0)
	if a.Model == "" && a.CostMagnitude != 0 {
		return gerror.New(gerror.ErrCodeValidation, "agent model is required (except for tool-only agents with cost_magnitude: 0)", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}

	// Validate type
	validTypes := map[string]bool{
		"manager":    true,
		"worker":     true,
		"specialist": true,
	}
	if !validTypes[a.Type] {
		return gerror.Newf(gerror.ErrCodeValidation, "invalid agent type: %s (must be manager, worker, or specialist)", a.Type).
			WithComponent("AgentConfig").
			WithOperation("Validate")
	}

	// Validate capabilities
	if len(a.Capabilities) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "agent must have at least one capability", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
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
			return gerror.Newf(gerror.ErrCodeValidation, "invalid cost_magnitude: %d (must be 0 for bash tools, or Fibonacci values: 1,2,3,5,8)", a.CostMagnitude).
				WithComponent("AgentConfig").
				WithOperation("Validate")
		}
	}

	// Validate context reset behavior
	if a.ContextReset != "" {
		validResets := map[string]bool{
			"truncate":  true,
			"summarize": true,
		}
		if !validResets[a.ContextReset] {
			return gerror.Newf(gerror.ErrCodeValidation, "invalid context_reset: %s (must be 'truncate' or 'summarize')", a.ContextReset).
				WithComponent("AgentConfig").
				WithOperation("Validate")
		}
	}

	// Validate context window
	if a.ContextWindow < 0 {
		return gerror.New(gerror.ErrCodeValidation, "context_window must be non-negative (0 for auto-detection)", nil).
			WithComponent("AgentConfig").
			WithOperation("Validate")
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
		return nil, gerror.New(gerror.ErrCodeValidation, "no manager agent configured", nil).
			WithComponent("GuildConfig").
			WithOperation("GetManagerAgent")
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

	return nil, gerror.Newf(gerror.ErrCodeNotFound, "manager agent '%s' not found", managerID).
		WithComponent("GuildConfig").
		WithOperation("GetManagerAgent")
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
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent with ID '%s' not found", id).
		WithComponent("GuildConfig").
		WithOperation("GetAgentByID")
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

// GetProviderAPIKey returns the API key for a specific provider from environment variables only
func (g *GuildConfig) GetProviderAPIKey(provider string) string {
	// Only use environment variables for security
	switch provider {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY")
	case "deepinfra":
		return os.Getenv("DEEPINFRA_API_KEY")
	case "ora":
		return os.Getenv("ORA_API_KEY")
	default:
		return ""
	}
}

// GetProviderBaseURL returns the base URL for a specific provider
func (g *GuildConfig) GetProviderBaseURL(provider string) string {
	switch provider {
	case "openai":
		return g.Providers.OpenAI.BaseURL
	case "anthropic":
		return g.Providers.Anthropic.BaseURL
	case "ollama":
		return g.Providers.Ollama.BaseURL
	case "claudecode", "claude_code":
		return g.Providers.ClaudeCode.BaseURL
	case "deepseek":
		return g.Providers.DeepSeek.BaseURL
	case "deepinfra":
		return g.Providers.DeepInfra.BaseURL
	case "ora":
		return g.Providers.Ora.BaseURL
	default:
		return ""
	}
}

// GetEffectiveStorageBackend returns the storage backend with smart defaults
func (g *GuildConfig) GetEffectiveStorageBackend() string {
	if g.Storage.Backend != "" {
		return g.Storage.Backend
	}
	return "sqlite" // Default to SQLite
}

// GetEffectiveSQLitePath returns the SQLite database path with smart defaults
func (g *GuildConfig) GetEffectiveSQLitePath() string {
	if g.Storage.SQLite.Path != "" {
		return g.Storage.SQLite.Path
	}
	return ".guild/guild.db" // Default path
}

// IsUsingSQLite returns true if the configuration is set to use SQLite storage
func (g *GuildConfig) IsUsingSQLite() bool {
	return g.GetEffectiveStorageBackend() == "sqlite"
}

// ensureLocalConfig ensures the local Guild configuration exists
func ensureLocalConfig(projectPath string) error {
	localDir := filepath.Join(projectPath, ".guild")
	configPath := filepath.Join(localDir, "guild.yaml")
	
	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	
	return gerror.New(gerror.ErrCodeNotFound, "local guild config not found", nil).
		WithComponent("config").
		WithOperation("ensureLocalConfig")
}


// LoadGlobalConfig is deprecated - use enhanced_loader.go instead
// Kept for backward compatibility
func LoadGlobalConfig() (*GlobalConfig, error) {
	// This function is deprecated - for compatibility only
	return nil, gerror.New(gerror.ErrCodeInternal, "LoadGlobalConfig is deprecated, use LoadEnhancedConfig instead", nil).
		WithComponent("config").
		WithOperation("LoadGlobalConfig")
}

// GlobalConfig is deprecated - types moved to pkg/project/global/interfaces.go
type GlobalConfig = global.GlobalConfig

// initializeGlobalConfig is deprecated - use pkg/project/global/InitializeGlobal
func initializeGlobalConfig() error {
	return global.InitializeGlobal()
}
