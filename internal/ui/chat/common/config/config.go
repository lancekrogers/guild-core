// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"path/filepath"

	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/preferences"
	"github.com/guild-framework/guild-core/pkg/project"
)

// ChatConfig holds configuration specific to the chat interface
type ChatConfig struct {
	// Basic settings
	CampaignID  string
	SessionID   string
	GuildConfig *config.GuildConfig
	ProjectRoot string
	UserID      string // Added for preference loading

	// UI settings
	Width           int
	Height          int
	MarkdownEnabled bool
	VimModeEnabled  bool
	ShowLineNumbers bool
	WrapLines       bool

	// Feature flags
	EnableCompletion    bool
	EnableHistory       bool
	EnableStatusDisplay bool
	EnableRichContent   bool

	// Theme and appearance (from preferences)
	Theme          string
	FontSize       int
	ColorScheme    string
	ShowTimestamps bool
	CompactMode    bool

	// AI settings (from preferences)
	DefaultProvider  string
	DefaultModel     string
	Temperature      float64
	MaxTokens        int
	StreamingEnabled bool

	// Paths
	DatabasePath string
	HistoryPath  string
	ConfigPath   string
}

// ConfigManager handles chat configuration loading and validation
type ConfigManager struct {
	ctx         context.Context
	prefService *preferences.Service
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(ctx context.Context) *ConfigManager {
	return &ConfigManager{
		ctx: ctx,
	}
}

// NewConfigManagerWithPreferences creates a new configuration manager with preferences service
func NewConfigManagerWithPreferences(ctx context.Context, prefService *preferences.Service) *ConfigManager {
	return &ConfigManager{
		ctx:         ctx,
		prefService: prefService,
	}
}

// LoadChatConfig loads and validates the chat configuration
func (cm *ConfigManager) LoadChatConfig(ctx context.Context, userID, campaignID, sessionID string) (*ChatConfig, error) {
	// Load guild configuration
	guildConfig, err := cm.loadGuildConfig(ctx)
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

	// Build configuration with defaults
	chatConfig := &ChatConfig{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		UserID:      userID,
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

		// Theme defaults
		Theme:          "dark",
		FontSize:       12,
		ColorScheme:    "default",
		ShowTimestamps: true,
		CompactMode:    false,

		// AI defaults
		DefaultProvider:  "anthropic",
		DefaultModel:     "claude-3-sonnet-20240229",
		Temperature:      0.7,
		MaxTokens:        4096,
		StreamingEnabled: true,

		// Paths
		DatabasePath: filepath.Join(".campaign", "memory.db"),
		HistoryPath:  filepath.Join(".campaign", "chat_history.txt"),
		ConfigPath:   filepath.Join(".campaign", "campaign.yaml"),
	}

	// Apply any overrides from guild config
	cm.applyConfigOverrides(chatConfig, guildConfig)

	// Load and apply user preferences if service is available
	if cm.prefService != nil && userID != "" {
		if err := cm.loadUserPreferences(ctx, chatConfig); err != nil {
			// Log but don't fail - preferences are optional
			// In production, would use proper logger
			_ = err
		}
	}

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
func (cm *ConfigManager) loadGuildConfig(ctx context.Context) (*config.GuildConfig, error) {
	// Load from current directory (LoadGuildConfig will add .guild/guild.yaml)
	return config.LoadGuildConfig(ctx, ".")
}

// applyConfigOverrides applies any overrides from the guild configuration
func (cm *ConfigManager) applyConfigOverrides(chatConfig *ChatConfig, guildConfig *config.GuildConfig) {
	// Apply UI preferences if they exist in guild config
	// This is extensible for future UI customization options

	// For now, keep defaults but this can be expanded
	// to read from guildConfig.Chat.* settings when they're added
}

// loadUserPreferences loads and applies user preferences to the chat configuration
func (cm *ConfigManager) loadUserPreferences(ctx context.Context, chatConfig *ChatConfig) error {
	// Load preferences with hierarchical resolution
	// Start with system defaults, then user, then campaign specific

	// Load user-level UI preferences
	uiTheme, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ui.theme")
	if err == nil && uiTheme != nil {
		if theme, ok := uiTheme.(string); ok {
			chatConfig.Theme = theme
		}
	}

	fontSize, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ui.font_size")
	if err == nil && fontSize != nil {
		if size, ok := fontSize.(float64); ok {
			chatConfig.FontSize = int(size)
		}
	}

	vimMode, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ui.vim_mode")
	if err == nil && vimMode != nil {
		if enabled, ok := vimMode.(bool); ok {
			chatConfig.VimModeEnabled = enabled
		}
	}

	// Load AI preferences
	provider, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ai.default_provider")
	if err == nil && provider != nil {
		if p, ok := provider.(string); ok {
			chatConfig.DefaultProvider = p
		}
	}

	model, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ai.default_model")
	if err == nil && model != nil {
		if m, ok := model.(string); ok {
			chatConfig.DefaultModel = m
		}
	}

	temperature, err := cm.prefService.GetUserPreference(ctx, chatConfig.UserID, "ai.temperature")
	if err == nil && temperature != nil {
		if temp, ok := temperature.(float64); ok {
			chatConfig.Temperature = temp
		}
	}

	// Load campaign-specific overrides if campaign is set
	if chatConfig.CampaignID != "" {
		campaignTheme, err := cm.prefService.GetCampaignPreference(ctx, chatConfig.CampaignID, "ui.theme")
		if err == nil && campaignTheme != nil {
			if theme, ok := campaignTheme.(string); ok {
				chatConfig.Theme = theme
			}
		}

		// Campaign-specific AI settings
		campaignModel, err := cm.prefService.GetCampaignPreference(ctx, chatConfig.CampaignID, "ai.default_model")
		if err == nil && campaignModel != nil {
			if m, ok := campaignModel.(string); ok {
				chatConfig.DefaultModel = m
			}
		}
	}

	return nil
}

// SaveUserPreferences saves the current UI state back to preferences
func (cm *ConfigManager) SaveUserPreferences(ctx context.Context, chatConfig *ChatConfig) error {
	if cm.prefService == nil || chatConfig.UserID == "" {
		return nil // No preference service or user ID
	}

	// Save UI preferences
	if err := cm.prefService.SetUserPreference(ctx, chatConfig.UserID, "ui.theme", chatConfig.Theme); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save theme preference")
	}

	if err := cm.prefService.SetUserPreference(ctx, chatConfig.UserID, "ui.font_size", float64(chatConfig.FontSize)); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save font size preference")
	}

	if err := cm.prefService.SetUserPreference(ctx, chatConfig.UserID, "ui.vim_mode", chatConfig.VimModeEnabled); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save vim mode preference")
	}

	// Save AI preferences
	if err := cm.prefService.SetUserPreference(ctx, chatConfig.UserID, "ai.default_provider", chatConfig.DefaultProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save provider preference")
	}

	if err := cm.prefService.SetUserPreference(ctx, chatConfig.UserID, "ai.temperature", chatConfig.Temperature); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save temperature preference")
	}

	return nil
}
