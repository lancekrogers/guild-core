// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
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
	Path string `yaml:"path,omitempty"` // Path to SQLite database file (default: "memory.db")
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

	// Agent personality and backstory (Step 7 enhancement)
	Backstory      *Backstory      `yaml:"backstory,omitempty"`
	Personality    *Personality    `yaml:"personality,omitempty"`
	Specialization *Specialization `yaml:"specialization,omitempty"`

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

// Backstory contains the professional background and narrative for an agent
type Backstory struct {
	// Professional background
	Experience    string   `yaml:"experience"`     // "15 years in distributed systems"
	PreviousRoles []string `yaml:"previous_roles"` // ["CTO at startup", "Google SRE"]
	Expertise     string   `yaml:"expertise"`      // Core expertise description
	Achievements  []string `yaml:"achievements"`   // Notable accomplishments

	// Personal touches
	Philosophy string   `yaml:"philosophy"` // Engineering philosophy
	Interests  []string `yaml:"interests"`  // Technical interests
	Background string   `yaml:"background"` // Educational/cultural background

	// Communication style
	CommunicationStyle string `yaml:"communication_style"` // How they communicate
	TeachingStyle      string `yaml:"teaching_style"`      // How they explain things

	// Medieval guild identity
	GuildRank   string   `yaml:"guild_rank"`  // "Master Artisan", "Journeyman", "Apprentice"
	Specialties []string `yaml:"specialties"` // ["Cryptography", "Performance"]

	// Generated dynamically
	CurrentMood string    `yaml:"-"` // Changes based on context
	LastUpdated time.Time `yaml:"-"`
}

// Personality defines the behavioral traits and communication patterns of an agent
type Personality struct {
	// Core traits
	Traits []PersonalityTrait `yaml:"traits"`

	// Communication preferences
	Formality   string `yaml:"formality"`    // formal, casual, adaptive
	DetailLevel string `yaml:"detail_level"` // concise, detailed, adaptive
	HumorLevel  string `yaml:"humor_level"`  // none, occasional, frequent

	// Working style
	ApproachStyle  string `yaml:"approach_style"`  // methodical, creative, balanced
	RiskTolerance  string `yaml:"risk_tolerance"`  // conservative, moderate, aggressive
	DecisionMaking string `yaml:"decision_making"` // data-driven, intuitive, hybrid

	// Interaction patterns
	Assertiveness int `yaml:"assertiveness"` // 1-10 scale
	Empathy       int `yaml:"empathy"`       // 1-10 scale
	Patience      int `yaml:"patience"`      // 1-10 scale

	// Medieval personality traits
	Honor         int `yaml:"honor"`         // 1-10 scale
	Wisdom        int `yaml:"wisdom"`        // 1-10 scale
	Craftsmanship int `yaml:"craftsmanship"` // 1-10 scale
}

// PersonalityTrait represents a specific behavioral trait
type PersonalityTrait struct {
	Name        string  `yaml:"name"`
	Strength    float64 `yaml:"strength"` // 0.0-1.0
	Description string  `yaml:"description"`
}

// Specialization defines the domain expertise and technical focus of an agent
type Specialization struct {
	Domain         string   `yaml:"domain"`          // fintech, healthcare, etc
	SubDomains     []string `yaml:"sub_domains"`     // payment processing, compliance
	ExpertiseLevel string   `yaml:"expertise_level"` // novice, intermediate, expert, master

	// Knowledge areas
	CoreKnowledge []string `yaml:"core_knowledge"` // Deep expertise areas
	Familiar      []string `yaml:"familiar"`       // Working knowledge
	Learning      []string `yaml:"learning"`       // Currently learning

	// Preferred approaches
	Methodologies []string `yaml:"methodologies"` // Agile, TDD, DDD
	Technologies  []string `yaml:"technologies"`  // Go, K8s, PostgreSQL
	Principles    []string `yaml:"principles"`    // SOLID, 12-factor

	// Medieval specialization
	Craft     string   `yaml:"craft"`     // "Blacksmithing", "Woodworking"
	Tools     []string `yaml:"tools"`     // Medieval tools they use
	Materials []string `yaml:"materials"` // What they work with
}

// LoadGuildConfig loads modular guild configuration created by guild init
// This loads from the new modular structure:
// - .campaign/campaign.yaml (campaign metadata)
// - .campaign/guilds/*.yaml (guild definitions)
// - .campaign/agents/*.yaml (agent configurations)
func LoadGuildConfig(ctx context.Context, projectPath string) (*GuildConfig, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "GuildConfig")
	ctx = observability.WithOperation(ctx, "LoadGuildConfig")

	logger.InfoContext(ctx, "Loading modular guild configuration", "project_path", projectPath)

	// Step 1: Load campaign configuration
	campaignPath := filepath.Join(projectPath, ".campaign", "campaign.yaml")
	if _, err := os.Stat(campaignPath); os.IsNotExist(err) {
		logger.WarnContext(ctx, "Campaign configuration not found", "path", campaignPath)
		
		// Check if .guild directory exists (old format)
		legacyGuildPath := filepath.Join(projectPath, ".guild", "guild.yaml")
		if _, err := os.Stat(legacyGuildPath); err == nil {
			return nil, gerror.New(gerror.ErrCodeNotFound, "legacy .guild/guild.yaml found - please run 'guild migrate' to upgrade to the new campaign format", nil).
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig").
				WithDetails("legacy_path", legacyGuildPath).
				WithDetails("expected_path", campaignPath)
		}
		
		// Check current directory for obvious indicators
		currentDir := filepath.Base(projectPath)
		helpMsg := "Guild campaign not initialized. Run 'guild init' to set up a new Guild project.\n\n" +
			"Current directory: " + currentDir + "\n" +
			"Expected file: " + campaignPath + "\n\n" +
			"If you're in the wrong directory, navigate to your project root first."
			
		return nil, gerror.New(gerror.ErrCodeNotFound, helpMsg, nil).
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("path", campaignPath)
	}

	// Read campaign.yaml
	campaignData, err := os.ReadFile(campaignPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read campaign config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("path", campaignPath)
	}

	// Parse campaign.yaml to get guild info
	var campaignInfo struct {
		Name     string   `yaml:"name"`
		Guilds   []string `yaml:"guilds"`
		Settings struct {
			DefaultGuild string `yaml:"default_guild"`
		} `yaml:"settings"`
	}
	if err := yaml.Unmarshal(campaignData, &campaignInfo); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse campaign config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("path", campaignPath)
	}

	// Determine which guild to load
	guildName := campaignInfo.Settings.DefaultGuild
	if guildName == "" && len(campaignInfo.Guilds) > 0 {
		guildName = campaignInfo.Guilds[0]
	}
	if guildName == "" {
		// Provide detailed error about what's missing
		var errorMsg string
		if len(campaignInfo.Guilds) == 0 {
			errorMsg = "campaign.yaml is missing guilds list - expected format:\n" +
				"name: my-campaign\n" +
				"guilds:\n" +
				"  - my-guild\n" +
				"settings:\n" +
				"  default_guild: my-guild"
		} else {
			errorMsg = "campaign.yaml is missing settings.default_guild - expected format:\n" +
				"settings:\n" +
				"  default_guild: " + campaignInfo.Guilds[0]
		}
		
		return nil, gerror.New(gerror.ErrCodeValidation, errorMsg, nil).
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("campaign_path", campaignPath).
			WithDetails("available_guilds", strings.Join(campaignInfo.Guilds, ", "))
	}

	logger.InfoContext(ctx, "Loading guild", "guild_name", guildName)

	// Step 2: Load guild definition
	guildPath := filepath.Join(projectPath, ".campaign", "guilds", guildName+".yaml")
	guildData, err := os.ReadFile(guildPath)
	if err != nil {
		if os.IsNotExist(err) {
			// List available guild files to help user
			guildsDir := filepath.Join(projectPath, ".campaign", "guilds")
			availableGuilds := []string{}
			if entries, dirErr := os.ReadDir(guildsDir); dirErr == nil {
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
						name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
						availableGuilds = append(availableGuilds, name)
					}
				}
			}
			
			errorMsg := fmt.Sprintf("guild file '%s.yaml' not found in .campaign/guilds/\n\n", guildName)
			if len(availableGuilds) > 0 {
				errorMsg += "Available guilds: " + strings.Join(availableGuilds, ", ") + "\n\n"
				errorMsg += "To fix: Update campaign.yaml settings.default_guild to one of the available guilds"
			} else {
				errorMsg += "No guild files found in .campaign/guilds/\n\n"
				errorMsg += "To fix: Run 'guild init' to create guild configuration files"
			}
			
			return nil, gerror.New(gerror.ErrCodeNotFound, errorMsg, nil).
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig").
				WithDetails("guild_name", guildName).
				WithDetails("path", guildPath).
				WithDetails("available_guilds", strings.Join(availableGuilds, ", "))
		}
		
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read guild config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("path", guildPath)
	}

	// Parse guild definition
	var guildDef struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description"`
		Purpose      string   `yaml:"purpose"`
		Manager      string   `yaml:"manager"`
		Agents       []string `yaml:"agents"`
		Coordination struct {
			MaxParallelTasks int  `yaml:"max_parallel_tasks"`
			ReviewRequired   bool `yaml:"review_required"`
			AutoHandoff      bool `yaml:"auto_handoff"`
		} `yaml:"coordination"`
	}
	if err := yaml.Unmarshal(guildData, &guildDef); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse guild config").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig").
			WithDetails("path", guildPath)
	}

	// Step 3: Load agent configurations
	agents := make([]AgentConfig, 0, len(guildDef.Agents))
	for _, agentID := range guildDef.Agents {
		// Check context in loop
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during agent loading").
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig")
		}

		agentPath := filepath.Join(projectPath, ".campaign", "agents", agentID+".yaml")
		agentData, err := os.ReadFile(agentPath)
		if err != nil {
			if os.IsNotExist(err) {
				// List available agent files to help user
				agentsDir := filepath.Join(projectPath, ".campaign", "agents")
				availableAgents := []string{}
				if entries, dirErr := os.ReadDir(agentsDir); dirErr == nil {
					for _, entry := range entries {
						if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
							name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
							availableAgents = append(availableAgents, name)
						}
					}
				}
				
				errorMsg := fmt.Sprintf("agent file '%s.yaml' not found in .campaign/agents/\n\n", agentID)
				if len(availableAgents) > 0 {
					errorMsg += "Available agents: " + strings.Join(availableAgents, ", ") + "\n\n"
					errorMsg += "To fix: Update guild file to reference existing agents, or create missing agent files"
				} else {
					errorMsg += "No agent files found in .campaign/agents/\n\n"
					errorMsg += "To fix: Run 'guild init' to create agent configuration files"
				}
				
				return nil, gerror.New(gerror.ErrCodeNotFound, errorMsg, nil).
					WithComponent("GuildConfig").
					WithOperation("LoadGuildConfig").
					WithDetails("agent_id", agentID).
					WithDetails("path", agentPath).
					WithDetails("available_agents", strings.Join(availableAgents, ", "))
			}
			
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read agent config").
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig").
				WithDetails("agent_id", agentID).
				WithDetails("path", agentPath)
		}

		var agent AgentConfig
		if err := yaml.Unmarshal(agentData, &agent); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse agent config").
				WithComponent("GuildConfig").
				WithOperation("LoadGuildConfig").
				WithDetails("agent_id", agentID).
				WithDetails("path", agentPath)
		}

		agents = append(agents, agent)
	}

	// Step 4: Construct GuildConfig from the loaded data
	config := &GuildConfig{
		Name:        guildDef.Name,
		Description: guildDef.Description,
		Manager: ManagerConfig{
			Default: guildDef.Manager,
		},
		Agents: agents,
		Metadata: Metadata{
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
			Version:   "1.0",
		},
	}

	logger.InfoContext(ctx, "Parsing configuration completed", "agents_count", len(config.Agents), "name", config.Name)

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid guild configuration").
			WithComponent("GuildConfig").
			WithOperation("LoadGuildConfig")
	}

	logger.InfoContext(ctx, "Guild configuration loaded successfully", "name", config.Name)

	return config, nil
}

// SaveGuildConfig saves guild configuration to a project directory
func SaveGuildConfig(ctx context.Context, projectPath string, config *GuildConfig) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "GuildConfig")
	ctx = observability.WithOperation(ctx, "SaveGuildConfig")

	logger.InfoContext(ctx, "Saving guild configuration", "project_path", projectPath, "config_name", config.Name)

	// Save to campaign directory
	configPath := filepath.Join(projectPath, ".campaign", "guild.yaml")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig").
			WithDetails("path", configPath)
	}

	// Check context before marshaling
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before marshal").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	logger.InfoContext(ctx, "Marshaling configuration to YAML")

	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig").
			WithDetails("path", configPath)
	}

	// Check context before file write
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before file write").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig")
	}

	logger.InfoContext(ctx, "Writing configuration file", "path", configPath)

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("GuildConfig").
			WithOperation("SaveGuildConfig").
			WithDetails("path", configPath)
	}

	logger.InfoContext(ctx, "Guild configuration saved successfully", "path", configPath)

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
	return "memory.db" // Default path
}

// IsUsingSQLite returns true if the configuration is set to use SQLite storage
func (g *GuildConfig) IsUsingSQLite() bool {
	return g.GetEffectiveStorageBackend() == "sqlite"
}

// ensureLocalConfig ensures the local Guild configuration exists
func ensureLocalConfig(projectPath string) error {
	// Check both .guild and .campaign directories
	locations := []struct {
		dir  string
		file string
	}{
		{filepath.Join(projectPath, ".guild"), "guild.yaml"},
		{filepath.Join(projectPath, ".guild"), "guild.yml"},
		{filepath.Join(projectPath, ".campaign"), "guild.yaml"},
		{filepath.Join(projectPath, ".campaign"), "guild.yml"},
	}

	for _, loc := range locations {
		configPath := filepath.Join(loc.dir, loc.file)
		if _, err := os.Stat(configPath); err == nil {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "local guild config not found", nil).
		WithComponent("config").
		WithOperation("ensureLocalConfig").
		WithDetails("info", "Run 'guild init' to create a new Guild project")
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
