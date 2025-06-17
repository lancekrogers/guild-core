// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// ChatConfig holds configuration specific to the chat interface
type ChatConfig struct {
	// Basic settings
	CampaignID   string
	SessionID    string
	GuildConfig  *config.GuildConfig
	ProjectRoot  string
	
	// UI settings
	Width            int
	Height           int
	MarkdownEnabled  bool
	VimModeEnabled   bool
	ShowLineNumbers  bool
	WrapLines        bool
	
	// Feature flags
	EnableCompletion    bool
	EnableHistory       bool
	EnableStatusDisplay bool
	EnableRichContent   bool
	
	// Paths
	DatabasePath   string
	HistoryPath    string
	ConfigPath     string
}

// ConfigManager handles chat configuration loading and validation
type ConfigManager struct {
	ctx context.Context
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(ctx context.Context) *ConfigManager {
	return &ConfigManager{
		ctx: ctx,
	}
}

// LoadChatConfig loads and validates the chat configuration
func (cm *ConfigManager) LoadChatConfig(campaignID, sessionID string) (*ChatConfig, error) {
	// Load guild configuration
	guildConfig, err := cm.loadGuildConfig()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to load guild configuration").
			WithComponent("chat.config").
			WithOperation("LoadChatConfig")
	}

	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load project context").
			WithComponent("chat.config").
			WithOperation("LoadChatConfig")
	}

	// Build configuration
	chatConfig := &ChatConfig{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		GuildConfig: guildConfig,
		ProjectRoot: projCtx.GetRootPath(),
		
		// UI defaults
		Width:           80,
		Height:          24,
		MarkdownEnabled: true,
		VimModeEnabled:  false,
		ShowLineNumbers: false,
		WrapLines:       true,
		
		// Feature defaults (all enabled)
		EnableCompletion:    true,
		EnableHistory:       true,
		EnableStatusDisplay: true,
		EnableRichContent:   true,
		
		// Paths
		DatabasePath: filepath.Join(".guild", "memory.db"),
		HistoryPath:  filepath.Join(".guild", "chat_history.txt"),
		ConfigPath:   filepath.Join(".guild", "guild.yaml"),
	}
	
	// Apply any overrides from guild config
	cm.applyConfigOverrides(chatConfig, guildConfig)
	
	return chatConfig, nil
}

// ValidateConfig validates the chat configuration
func (cm *ConfigManager) ValidateConfig(config *ChatConfig) error {
	if config == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "chat config is nil", nil).
			WithComponent("chat.config").
			WithOperation("ValidateConfig")
	}
	
	if config.GuildConfig == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "guild config is nil", nil).
			WithComponent("chat.config").
			WithOperation("ValidateConfig")
	}
	
	if config.SessionID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "session ID is required", nil).
			WithComponent("chat.config").
			WithOperation("ValidateConfig")
	}
	
	if config.ProjectRoot == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "project root is required", nil).
			WithComponent("chat.config").
			WithOperation("ValidateConfig")
	}
	
	// Validate dimensions
	if config.Width < 40 {
		config.Width = 40 // Minimum usable width
	}
	if config.Height < 10 {
		config.Height = 10 // Minimum usable height
	}
	
	return nil
}

// UpdateDimensions updates the chat dimensions
func (cm *ConfigManager) UpdateDimensions(config *ChatConfig, width, height int) {
	config.Width = width
	config.Height = height
	
	// Ensure minimum dimensions
	if config.Width < 40 {
		config.Width = 40
	}
	if config.Height < 10 {
		config.Height = 10
	}
}

// ToggleFeature toggles a feature flag
func (cm *ConfigManager) ToggleFeature(config *ChatConfig, feature string) {
	switch feature {
	case "completion":
		config.EnableCompletion = !config.EnableCompletion
	case "history":
		config.EnableHistory = !config.EnableHistory
	case "status":
		config.EnableStatusDisplay = !config.EnableStatusDisplay
	case "rich_content":
		config.EnableRichContent = !config.EnableRichContent
	case "vim_mode":
		config.VimModeEnabled = !config.VimModeEnabled
	case "line_numbers":
		config.ShowLineNumbers = !config.ShowLineNumbers
	case "wrap_lines":
		config.WrapLines = !config.WrapLines
	}
}

// GetCampaignDisplay returns a user-friendly campaign display name
func (cm *ConfigManager) GetCampaignDisplay(config *ChatConfig) string {
	if config.CampaignID == "" {
		return "default"
	}
	return config.CampaignID
}

// loadGuildConfig loads the guild configuration from the project
func (cm *ConfigManager) loadGuildConfig() (*config.GuildConfig, error) {
	// Load from current directory (LoadGuildConfig will add .guild/guild.yaml)
	return config.LoadGuildConfig(".")
}

// applyConfigOverrides applies any overrides from the guild configuration
func (cm *ConfigManager) applyConfigOverrides(chatConfig *ChatConfig, guildConfig *config.GuildConfig) {
	// Apply UI preferences if they exist in guild config
	// This is extensible for future UI customization options
	
	// For now, keep defaults but this can be expanded
	// to read from guildConfig.Chat.* settings when they're added
}