// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// CampaignConfig represents the campaign-level configuration
// This is stored in .guild/campaign.yml
type CampaignConfig struct {
	// Campaign identity
	Name        string `yaml:"name"`
	Description string `yaml:"description"` // High-level multi-project goal

	// Project settings
	ProjectSettings map[string]interface{} `yaml:"project_settings,omitempty"`

	// Guild to commission mappings
	// Example: "backend-guild": ["e-commerce-platform", "api-refactor"]
	CommissionMappings map[string][]string `yaml:"commission_mappings,omitempty"`

	// Runtime state
	LastSelectedGuild string `yaml:"last_selected_guild,omitempty"` // Persisted for UX
}

// LoadCampaignConfig loads the campaign configuration from a project directory
func LoadCampaignConfig(ctx context.Context, projectPath string) (*CampaignConfig, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CampaignConfig").
			WithOperation("LoadCampaignConfig")
	}
	configPath := filepath.Join(projectPath, ".guild", "campaign.yml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "campaign configuration not found at %s", configPath).
			WithComponent("CampaignConfig").
			WithOperation("LoadCampaignConfig")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read campaign config").
			WithComponent("CampaignConfig").
			WithOperation("LoadCampaignConfig").
			WithDetails("path", configPath)
	}

	var config CampaignConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse campaign config").
			WithComponent("CampaignConfig").
			WithOperation("LoadCampaignConfig").
			WithDetails("path", configPath)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid campaign configuration").
			WithComponent("CampaignConfig").
			WithOperation("LoadCampaignConfig")
	}

	return &config, nil
}

// SaveCampaignConfig saves the campaign configuration to a project directory
func SaveCampaignConfig(ctx context.Context, projectPath string, config *CampaignConfig) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CampaignConfig").
			WithOperation("SaveCampaignConfig")
	}
	configPath := filepath.Join(projectPath, ".guild", "campaign.yml")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guild directory").
			WithComponent("CampaignConfig").
			WithOperation("SaveCampaignConfig")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign config").
			WithComponent("CampaignConfig").
			WithOperation("SaveCampaignConfig")
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign config").
			WithComponent("CampaignConfig").
			WithOperation("SaveCampaignConfig").
			WithDetails("path", configPath)
	}

	return nil
}

// UpdateLastSelectedGuild updates the last selected guild and saves the config
func (c *CampaignConfig) UpdateLastSelectedGuild(ctx context.Context, projectPath string, guildName string) error {
	c.LastSelectedGuild = guildName
	return SaveCampaignConfig(ctx, projectPath, c)
}

// Validate validates the campaign configuration
func (c *CampaignConfig) Validate() error {
	if c.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "campaign name is required", nil).
			WithComponent("CampaignConfig").
			WithOperation("Validate")
	}

	if c.Description == "" {
		return gerror.New(gerror.ErrCodeValidation, "campaign description is required", nil).
			WithComponent("CampaignConfig").
			WithOperation("Validate")
	}

	return nil
}

// GetMappedCommissions returns the commissions mapped to a specific guild
func (c *CampaignConfig) GetMappedCommissions(guildName string) []string {
	if c.CommissionMappings == nil {
		return []string{}
	}
	return c.CommissionMappings[guildName]
}

// MapGuildToCommissions maps a guild to a set of commissions
func (c *CampaignConfig) MapGuildToCommissions(guildName string, commissions []string) {
	if c.CommissionMappings == nil {
		c.CommissionMappings = make(map[string][]string)
	}
	c.CommissionMappings[guildName] = commissions
}

// GetProjectSetting returns a project setting by key
func (c *CampaignConfig) GetProjectSetting(key string) (interface{}, bool) {
	if c.ProjectSettings == nil {
		return nil, false
	}
	val, ok := c.ProjectSettings[key]
	return val, ok
}

// SetProjectSetting sets a project setting
func (c *CampaignConfig) SetProjectSetting(key string, value interface{}) {
	if c.ProjectSettings == nil {
		c.ProjectSettings = make(map[string]interface{})
	}
	c.ProjectSettings[key] = value
}