// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agents

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/backstory"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

// AgentInitializer manages the creation and initialization of enhanced agents
type AgentInitializer struct {
	creator           *DefaultAgentCreator
	backstoryManager  *backstory.BackstoryManager
	promptRegistry    layered.LayeredRegistry
}

// NewAgentInitializer creates a new agent initializer
func NewAgentInitializer(promptRegistry layered.LayeredRegistry) *AgentInitializer {
	backstoryManager := backstory.NewBackstoryManager(promptRegistry)
	
	return &AgentInitializer{
		creator:          NewDefaultAgentCreator(),
		backstoryManager: backstoryManager,
		promptRegistry:   promptRegistry,
	}
}

// InitializeDefaultAgents creates and saves default agents to a project
func (ai *AgentInitializer) InitializeDefaultAgents(ctx context.Context, projectPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("InitializeDefaultAgents")
	}

	// Create default agent set
	agents, err := ai.creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create default agent set").
			WithComponent("AgentInitializer").
			WithOperation("InitializeDefaultAgents")
	}

	// Ensure agents directory exists
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("AgentInitializer").
			WithOperation("InitializeDefaultAgents").
			WithDetails("dir", agentsDir)
	}

	// Save each agent and register with backstory manager
	for _, agent := range agents {
		// Save agent configuration to file
		if err := ai.saveAgentConfig(ctx, agentsDir, agent); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save agent config").
				WithComponent("AgentInitializer").
				WithOperation("InitializeDefaultAgents").
				WithDetails("agent_id", agent.ID)
		}

		// Register with backstory manager for personality enhancement
		if err := ai.backstoryManager.RegisterAgent(agent); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent with backstory manager").
				WithComponent("AgentInitializer").
				WithOperation("InitializeDefaultAgents").
				WithDetails("agent_id", agent.ID)
		}
	}

	return nil
}

// LoadAndEnhanceAgents loads agents from config and enhances them with backstory system
func (ai *AgentInitializer) LoadAndEnhanceAgents(ctx context.Context, guildConfig *config.GuildConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("LoadAndEnhanceAgents")
	}

	if guildConfig == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "guild config cannot be nil", nil).
			WithComponent("AgentInitializer").
			WithOperation("LoadAndEnhanceAgents")
	}

	// Register each agent with the backstory manager
	for _, agent := range guildConfig.Agents {
		agentCopy := agent // Create copy to avoid pointer issues
		
		if err := ai.backstoryManager.RegisterAgent(&agentCopy); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent with backstory manager").
				WithComponent("AgentInitializer").
				WithOperation("LoadAndEnhanceAgents").
				WithDetails("agent_id", agent.ID)
		}
	}

	return nil
}

// CreateElenaIfMissing checks if Elena exists and creates her if missing
func (ai *AgentInitializer) CreateElenaIfMissing(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("CreateElenaIfMissing")
	}

	// Check if Elena already exists
	for _, agent := range guildConfig.Agents {
		if agent.ID == "elena-guild-master" {
			return nil // Elena already exists
		}
	}

	// Create Elena
	elena, err := ai.creator.CreateElenaGuildMaster(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create Elena").
			WithComponent("AgentInitializer").
			WithOperation("CreateElenaIfMissing")
	}

	// Save Elena to agents directory
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := ai.saveAgentConfig(ctx, agentsDir, elena); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save Elena config").
			WithComponent("AgentInitializer").
			WithOperation("CreateElenaIfMissing")
	}

	// Add Elena to guild config
	guildConfig.Agents = append(guildConfig.Agents, *elena)

	// Set Elena as default manager if no manager is set
	if guildConfig.Manager.Default == "" {
		guildConfig.Manager.Default = elena.ID
	}

	// Register with backstory manager
	if err := ai.backstoryManager.RegisterAgent(elena); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register Elena with backstory manager").
			WithComponent("AgentInitializer").
			WithOperation("CreateElenaIfMissing")
	}

	return nil
}

// EnhanceExistingAgent enhances an existing agent with a specialist template
func (ai *AgentInitializer) EnhanceExistingAgent(ctx context.Context, agentID, specialistTemplate string, guildConfig *config.GuildConfig, projectPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("EnhanceExistingAgent")
	}

	// Find the agent in config
	var targetAgent *config.AgentConfig
	for i := range guildConfig.Agents {
		if guildConfig.Agents[i].ID == agentID {
			targetAgent = &guildConfig.Agents[i]
			break
		}
	}

	if targetAgent == nil {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found in guild config", agentID).
			WithComponent("AgentInitializer").
			WithOperation("EnhanceExistingAgent")
	}

	// Enhance with specialist template
	if err := ai.creator.EnhanceAgentWithBackstory(ctx, targetAgent, specialistTemplate); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enhance agent with backstory").
			WithComponent("AgentInitializer").
			WithOperation("EnhanceExistingAgent")
	}

	// Save enhanced agent
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := ai.saveAgentConfig(ctx, agentsDir, targetAgent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save enhanced agent").
			WithComponent("AgentInitializer").
			WithOperation("EnhanceExistingAgent")
	}

	// Re-register with backstory manager
	if err := ai.backstoryManager.RegisterAgent(targetAgent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to re-register enhanced agent").
			WithComponent("AgentInitializer").
			WithOperation("EnhanceExistingAgent")
	}

	return nil
}

// GetBackstoryManager returns the backstory manager for external use
func (ai *AgentInitializer) GetBackstoryManager() *backstory.BackstoryManager {
	return ai.backstoryManager
}

// GetAvailableSpecialists returns list of available specialist templates
func (ai *AgentInitializer) GetAvailableSpecialists() []string {
	return ai.creator.ListAvailableSpecialists()
}

// GeneratePersonalityPrompt generates an enhanced prompt using the backstory system
func (ai *AgentInitializer) GeneratePersonalityPrompt(ctx context.Context, agentID, basePrompt string, turnContext *layered.TurnContext) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("GeneratePersonalityPrompt")
	}

	return ai.backstoryManager.BuildPersonalityPrompt(ctx, agentID, basePrompt, turnContext)
}

// saveAgentConfig saves an agent configuration to the agents directory
func (ai *AgentInitializer) saveAgentConfig(ctx context.Context, agentsDir string, agent *config.AgentConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Create simplified config for file storage (compatible with existing system)
	agentFileConfig := map[string]interface{}{
		"id":           agent.ID,
		"name":         agent.Name,
		"type":         agent.Type,
		"description":  agent.Description,
		"provider":     agent.Provider,
		"model":        agent.Model,
		"capabilities": agent.Capabilities,
		"tools":        agent.Tools,
	}

	// Add cost magnitude if set
	if agent.CostMagnitude != 0 {
		agentFileConfig["cost_magnitude"] = agent.CostMagnitude
	}

	// Add backstory, personality, and specialization if present
	if agent.Backstory != nil {
		agentFileConfig["backstory"] = agent.Backstory
	}
	if agent.Personality != nil {
		agentFileConfig["personality"] = agent.Personality
	}
	if agent.Specialization != nil {
		agentFileConfig["specialization"] = agent.Specialization
	}

	// Marshal to YAML
	agentData, err := yaml.Marshal(agentFileConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithDetails("agent_id", agent.ID)
	}

	// Write to file
	agentFilePath := filepath.Join(agentsDir, agent.ID+".yaml")
	if err := os.WriteFile(agentFilePath, agentData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config file").
			WithDetails("path", agentFilePath)
	}

	return nil
}

// CreateGuildConfigWithElena creates a complete guild config with Elena as manager
func (ai *AgentInitializer) CreateGuildConfigWithElena(ctx context.Context, guildName string) (*config.GuildConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("CreateGuildConfigWithElena")
	}

	// Create default agent set
	agents, err := ai.creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create default agent set").
			WithComponent("AgentInitializer").
			WithOperation("CreateGuildConfigWithElena")
	}

	// Convert agent pointers to values for config
	agentConfigs := make([]config.AgentConfig, len(agents))
	for i, agent := range agents {
		agentConfigs[i] = *agent
	}

	guildConfig := &config.GuildConfig{
		Name:        guildName,
		Description: "Enhanced guild with Elena the Guild Master and rich agent personalities",
		Version:     "1.0",
		Manager: config.ManagerConfig{
			Default: "elena-guild-master", // Elena is the default manager
			Fallback: []string{"marcus-developer"}, // Marcus as fallback
		},
		Storage: config.StorageConfig{
			Backend: "sqlite",
			SQLite: config.SQLiteConfig{
				Path: ".guild/guild.db",
			},
		},
		Agents: agentConfigs,
	}

	return guildConfig, nil
}

// UpgradeExistingGuild upgrades an existing guild with enhanced agents
func (ai *AgentInitializer) UpgradeExistingGuild(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentInitializer").
			WithOperation("UpgradeExistingGuild")
	}

	// Ensure campaign directory structure exists
	campaignDir := filepath.Join(projectPath, paths.DefaultCampaignDir)
	agentsDir := filepath.Join(campaignDir, "agents")
	
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("AgentInitializer").
			WithOperation("UpgradeExistingGuild").
			WithDetails("dir", agentsDir)
	}

	// Create Elena if missing
	if err := ai.CreateElenaIfMissing(ctx, guildConfig, projectPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create Elena").
			WithComponent("AgentInitializer").
			WithOperation("UpgradeExistingGuild")
	}

	// Load and enhance all agents with backstory system
	if err := ai.LoadAndEnhanceAgents(ctx, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enhance agents").
			WithComponent("AgentInitializer").
			WithOperation("UpgradeExistingGuild")
	}

	return nil
}