// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/paths"
)

// GuildConfigFile represents the guild configuration file structure
// This is stored in .campaign/guild.yml
type GuildConfigFile struct {
	// Map of guild name to guild definition
	Guilds map[string]GuildDefinition `yaml:"guilds"`
}

// GuildDefinition defines a single guild (team of agents)
type GuildDefinition struct {
	Purpose     string   `yaml:"purpose"`     // Team goal/purpose
	Description string   `yaml:"description"` // Detailed description
	Agents      []string `yaml:"agents"`      // Agent names (references to agents/*.yml)

	// Optional coordination settings
	Coordination *CoordinationSettings `yaml:"coordination,omitempty"`
}

// CoordinationSettings defines how the guild coordinates work
type CoordinationSettings struct {
	MaxParallelTasks int  `yaml:"max_parallel_tasks,omitempty"`
	ReviewRequired   bool `yaml:"review_required,omitempty"`
	AutoHandoff      bool `yaml:"auto_handoff,omitempty"`
}

// LoadGuildConfigFile loads the guild configuration from a project directory
func LoadGuildConfigFile(ctx context.Context, projectPath string) (*GuildConfigFile, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildConfigFile").
			WithOperation("LoadGuildConfigFile")
	}
	configPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "guild.yml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "guild configuration not found at %s", configPath).
			WithComponent("GuildConfigFile").
			WithOperation("LoadGuildConfigFile")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read guild config").
			WithComponent("GuildConfigFile").
			WithOperation("LoadGuildConfigFile").
			WithDetails("path", configPath)
	}

	var config GuildConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse guild config").
			WithComponent("GuildConfigFile").
			WithOperation("LoadGuildConfigFile").
			WithDetails("path", configPath)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid guild configuration").
			WithComponent("GuildConfigFile").
			WithOperation("LoadGuildConfigFile")
	}

	return &config, nil
}

// SaveGuildConfigFile saves the guild configuration to a project directory
func SaveGuildConfigFile(ctx context.Context, projectPath string, config *GuildConfigFile) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildConfigFile").
			WithOperation("SaveGuildConfigFile")
	}
	configPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "guild.yml")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guild directory").
			WithComponent("GuildConfigFile").
			WithOperation("SaveGuildConfigFile")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("GuildConfigFile").
			WithOperation("SaveGuildConfigFile")
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("GuildConfigFile").
			WithOperation("SaveGuildConfigFile").
			WithDetails("path", configPath)
	}

	return nil
}

// Validate validates the guild configuration file
func (g *GuildConfigFile) Validate() error {
	if g.Guilds == nil || len(g.Guilds) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "at least one guild must be defined", nil).
			WithComponent("GuildConfigFile").
			WithOperation("Validate")
	}

	// Validate each guild
	for name, guild := range g.Guilds {
		if err := guild.Validate(); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeValidation, "guild '%s' validation failed", name).
				WithComponent("GuildConfigFile").
				WithOperation("Validate")
		}
	}

	return nil
}

// Validate validates a guild definition
func (g *GuildDefinition) Validate() error {
	if g.Purpose == "" {
		return gerror.New(gerror.ErrCodeValidation, "guild purpose is required", nil).
			WithComponent("GuildDefinition").
			WithOperation("Validate")
	}

	if g.Description == "" {
		return gerror.New(gerror.ErrCodeValidation, "guild description is required", nil).
			WithComponent("GuildDefinition").
			WithOperation("Validate")
	}

	if len(g.Agents) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "guild must have at least one agent", nil).
			WithComponent("GuildDefinition").
			WithOperation("Validate")
	}

	// Validate coordination settings if present
	if g.Coordination != nil {
		if g.Coordination.MaxParallelTasks < 0 {
			return gerror.New(gerror.ErrCodeValidation, "max_parallel_tasks must be non-negative", nil).
				WithComponent("GuildDefinition").
				WithOperation("Validate")
		}
	}

	return nil
}

// GetGuild returns a specific guild definition by name
func (g *GuildConfigFile) GetGuild(name string) (*GuildDefinition, error) {
	guild, exists := g.Guilds[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "guild '%s' not found", name).
			WithComponent("GuildConfigFile").
			WithOperation("GetGuild")
	}
	return &guild, nil
}

// AddGuild adds a new guild to the configuration
func (g *GuildConfigFile) AddGuild(name string, guild GuildDefinition) error {
	if g.Guilds == nil {
		g.Guilds = make(map[string]GuildDefinition)
	}

	if _, exists := g.Guilds[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "guild '%s' already exists", name).
			WithComponent("GuildConfigFile").
			WithOperation("AddGuild")
	}

	// Validate the guild before adding
	if err := guild.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid guild definition").
			WithComponent("GuildConfigFile").
			WithOperation("AddGuild")
	}

	g.Guilds[name] = guild
	return nil
}

// ListGuildNames returns all guild names
func (g *GuildConfigFile) ListGuildNames() []string {
	names := make([]string, 0, len(g.Guilds))
	for name := range g.Guilds {
		names = append(names, name)
	}
	return names
}

// HasAgent checks if any guild contains the specified agent
func (g *GuildConfigFile) HasAgent(agentName string) bool {
	for _, guild := range g.Guilds {
		for _, agent := range guild.Agents {
			if agent == agentName {
				return true
			}
		}
	}
	return false
}

// GetGuildForAgent returns the guild name that contains the specified agent
func (g *GuildConfigFile) GetGuildForAgent(agentName string) (string, error) {
	for guildName, guild := range g.Guilds {
		for _, agent := range guild.Agents {
			if agent == agentName {
				return guildName, nil
			}
		}
	}
	return "", gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found in any guild", agentName).
		WithComponent("GuildConfigFile").
		WithOperation("GetGuildForAgent")
}
