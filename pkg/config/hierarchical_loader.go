// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// HierarchicalConfig represents the complete hierarchical configuration
type HierarchicalConfig struct {
	Campaign *CampaignConfig
	Guilds   *GuildConfigFile
	Agents   map[string]*AgentConfig // Key is agent name

	// Internal state
	projectPath string
	mu          sync.RWMutex
}

// HierarchicalLoader loads and manages the hierarchical configuration
type HierarchicalLoader struct {
	cache map[string]*HierarchicalConfig
	mu    sync.RWMutex
}

// NewHierarchicalLoader creates a new hierarchical configuration loader
func NewHierarchicalLoader() *HierarchicalLoader {
	return &HierarchicalLoader{
		cache: make(map[string]*HierarchicalConfig),
	}
}

// LoadHierarchicalConfig loads the complete hierarchical configuration for a project
func (l *HierarchicalLoader) LoadHierarchicalConfig(ctx context.Context, projectPath string) (*HierarchicalConfig, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalLoader").
			WithOperation("LoadHierarchicalConfig")
	}

	l.mu.RLock()
	if cached, exists := l.cache[projectPath]; exists {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Load campaign configuration
	campaign, err := LoadCampaignConfig(ctx, projectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load campaign config").
			WithComponent("HierarchicalLoader").
			WithOperation("LoadHierarchicalConfig")
	}

	// Load guild configuration
	guilds, err := LoadGuildConfigFile(ctx, projectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("HierarchicalLoader").
			WithOperation("LoadHierarchicalConfig")
	}

	// Load all agent configurations referenced by guilds
	agents, err := l.loadAgentConfigs(ctx, projectPath, guilds)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load agent configs").
			WithComponent("HierarchicalLoader").
			WithOperation("LoadHierarchicalConfig")
	}

	config := &HierarchicalConfig{
		Campaign:    campaign,
		Guilds:      guilds,
		Agents:      agents,
		projectPath: projectPath,
	}

	// Cache the configuration
	l.mu.Lock()
	l.cache[projectPath] = config
	l.mu.Unlock()

	return config, nil
}

// RefreshConfig reloads the configuration with validation
func (l *HierarchicalLoader) RefreshConfig(ctx context.Context, projectPath string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalLoader").
			WithOperation("RefreshConfig")
	}

	// First, validate the new configuration without caching
	newConfig, err := l.validateConfiguration(ctx, projectPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "configuration validation failed").
			WithComponent("HierarchicalLoader").
			WithOperation("RefreshConfig")
	}

	// If validation passes, update the cache
	l.mu.Lock()
	l.cache[projectPath] = newConfig
	l.mu.Unlock()

	return nil
}

// validateConfiguration loads and validates configuration without caching
func (l *HierarchicalLoader) validateConfiguration(ctx context.Context, projectPath string) (*HierarchicalConfig, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalLoader").
			WithOperation("validateConfiguration")
	}

	// Load campaign configuration
	campaign, err := LoadCampaignConfig(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	// Load guild configuration
	guilds, err := LoadGuildConfigFile(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	// Load all agent configurations
	agents, err := l.loadAgentConfigs(ctx, projectPath, guilds)
	if err != nil {
		return nil, err
	}

	// Validate cross-references
	if err := l.validateCrossReferences(campaign, guilds, agents); err != nil {
		return nil, err
	}

	return &HierarchicalConfig{
		Campaign:    campaign,
		Guilds:      guilds,
		Agents:      agents,
		projectPath: projectPath,
	}, nil
}

// loadAgentConfigs loads all agent configurations referenced by guilds
func (l *HierarchicalLoader) loadAgentConfigs(ctx context.Context, projectPath string, guilds *GuildConfigFile) (map[string]*AgentConfig, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfigs")
	}
	agents := make(map[string]*AgentConfig)
	loaded := make(map[string]bool) // Track loaded agents to avoid duplicates

	agentsDir := filepath.Join(projectPath, ".guild", "agents")

	// Load all agents referenced by all guilds
	for guildName, guild := range guilds.Guilds {
		for _, agentName := range guild.Agents {
			if loaded[agentName] {
				continue // Already loaded
			}

			agentPath := filepath.Join(agentsDir, agentName+".yml")
			agent, err := l.loadAgentConfig(ctx, agentPath)
			if err != nil {
				return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, 
					"failed to load agent '%s' for guild '%s'", agentName, guildName).
					WithComponent("HierarchicalLoader").
					WithOperation("loadAgentConfigs")
			}

			// Ensure agent name matches filename
			if agent.Name != agentName {
				agent.Name = agentName // Use filename as authoritative name
			}

			agents[agentName] = agent
			loaded[agentName] = true
		}
	}

	return agents, nil
}

// loadAgentConfig loads a single agent configuration
func (l *HierarchicalLoader) loadAgentConfig(ctx context.Context, agentPath string) (*AgentConfig, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfig")
	}
	// Check if file exists
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent configuration not found at %s", agentPath).
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfig")
	}

	data, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read agent config").
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfig").
			WithDetails("path", agentPath)
	}

	var agent AgentConfig
	if err := yaml.Unmarshal(data, &agent); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse agent config").
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfig").
			WithDetails("path", agentPath)
	}

	// Validate the agent configuration
	if err := agent.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid agent configuration").
			WithComponent("HierarchicalLoader").
			WithOperation("loadAgentConfig").
			WithDetails("path", agentPath)
	}

	return &agent, nil
}

// validateCrossReferences validates cross-references between configurations
func (l *HierarchicalLoader) validateCrossReferences(campaign *CampaignConfig, guilds *GuildConfigFile, agents map[string]*AgentConfig) error {
	// Validate commission mappings reference valid guilds
	if campaign.CommissionMappings != nil {
		for guildName := range campaign.CommissionMappings {
			if _, err := guilds.GetGuild(guildName); err != nil {
				return gerror.Newf(gerror.ErrCodeValidation, 
					"commission mapping references non-existent guild '%s'", guildName).
					WithComponent("HierarchicalLoader").
					WithOperation("validateCrossReferences")
			}
		}
	}

	// Validate all guild agent references exist
	for guildName, guild := range guilds.Guilds {
		for _, agentName := range guild.Agents {
			if _, exists := agents[agentName]; !exists {
				return gerror.Newf(gerror.ErrCodeValidation, 
					"guild '%s' references non-existent agent '%s'", guildName, agentName).
					WithComponent("HierarchicalLoader").
					WithOperation("validateCrossReferences")
			}
		}
	}

	return nil
}

// GetActiveGuild returns the configuration for the currently active guild
func (c *HierarchicalConfig) GetActiveGuild(guildName string) (*GuildDefinition, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Guilds.GetGuild(guildName)
}

// GetAgentByName returns an agent configuration by name
func (c *HierarchicalConfig) GetAgentByName(agentName string) (*AgentConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agent, exists := c.Agents[agentName]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found", agentName).
			WithComponent("HierarchicalConfig").
			WithOperation("GetAgentByName")
	}

	return agent, nil
}

// GetGuildAgents returns all agent configurations for a specific guild
func (c *HierarchicalConfig) GetGuildAgents(guildName string) ([]*AgentConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	guild, err := c.Guilds.GetGuild(guildName)
	if err != nil {
		return nil, err
	}

	agents := make([]*AgentConfig, 0, len(guild.Agents))
	for _, agentName := range guild.Agents {
		if agent, exists := c.Agents[agentName]; exists {
			agents = append(agents, agent)
		} else {
			return nil, gerror.Newf(gerror.ErrCodeInternal, 
				"guild '%s' references non-existent agent '%s'", guildName, agentName).
				WithComponent("HierarchicalConfig").
				WithOperation("GetGuildAgents")
		}
	}

	return agents, nil
}

// SaveAgentConfig saves an agent configuration
func (c *HierarchicalConfig) SaveAgentConfig(ctx context.Context, agentName string, agent *AgentConfig) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("HierarchicalConfig").
			WithOperation("SaveAgentConfig")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	agentPath := filepath.Join(c.projectPath, ".guild", "agents", agentName+".yml")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(agentPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("HierarchicalConfig").
			WithOperation("SaveAgentConfig")
	}

	// Validate before saving
	if err := agent.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid agent configuration").
			WithComponent("HierarchicalConfig").
			WithOperation("SaveAgentConfig")
	}

	data, err := yaml.Marshal(agent)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithComponent("HierarchicalConfig").
			WithOperation("SaveAgentConfig")
	}

	if err := os.WriteFile(agentPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config").
			WithComponent("HierarchicalConfig").
			WithOperation("SaveAgentConfig").
			WithDetails("path", agentPath)
	}

	// Update cache
	c.Agents[agentName] = agent

	return nil
}

// ValidateAll performs comprehensive validation of all configurations
func (c *HierarchicalConfig) ValidateAll() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Validate campaign
	if err := c.Campaign.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "campaign validation failed").
			WithComponent("HierarchicalConfig").
			WithOperation("ValidateAll")
	}

	// Validate guilds
	if err := c.Guilds.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "guild validation failed").
			WithComponent("HierarchicalConfig").
			WithOperation("ValidateAll")
	}

	// Validate each agent
	for name, agent := range c.Agents {
		if err := agent.Validate(); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeValidation, "agent '%s' validation failed", name).
				WithComponent("HierarchicalConfig").
				WithOperation("ValidateAll")
		}
	}

	return nil
}