// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project/global"
	"github.com/guild-ventures/guild-core/pkg/project/local"
)

// EnhancedGuildConfig represents the merged configuration from global and local
type EnhancedGuildConfig struct {
	// Local project configuration
	*GuildConfig

	// Global configuration overlays
	GlobalProviders *global.ProvidersConfig           `yaml:"-"`
	GlobalTools     *global.ToolsConfig               `yaml:"-"`
	GlobalUI        *global.UIConfig                  `yaml:"-"`
	GlobalSecurity  *global.GlobalSecurityConfig      `yaml:"-"`
	GlobalLSP       map[string]global.LSPServerConfig `yaml:"-"`
}

// LoadEnhancedConfig loads both global and local configurations and merges them
func LoadEnhancedConfig(ctx context.Context, projectPath string) (*EnhancedGuildConfig, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config").
			WithOperation("load_enhanced")
	}
	// Ensure global config exists
	if err := global.EnsureGlobalInitialized(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to ensure global config").
			WithComponent("config").
			WithOperation("load_enhanced")
	}

	// Ensure local config exists
	if err := local.EnsureLocalInitialized(projectPath); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to ensure local config").
			WithComponent("config").
			WithOperation("load_enhanced")
	}

	// Load global configuration
	globalConfig, err := loadGlobalConfig()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load global config").
			WithComponent("config").
			WithOperation("load_enhanced")
	}

	// Load local configuration
	localConfig, err := LoadGuildConfig(ctx, projectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load local config").
			WithComponent("config").
			WithOperation("load_enhanced")
	}

	// Merge configurations
	enhanced := &EnhancedGuildConfig{
		GuildConfig:     localConfig,
		GlobalProviders: &globalConfig.Providers,
		GlobalTools:     &globalConfig.Tools,
		GlobalUI:        &globalConfig.UI,
		GlobalSecurity:  &globalConfig.Security,
		GlobalLSP:       globalConfig.LSPServers,
	}

	// Apply global defaults where local is not specified
	applyGlobalDefaults(enhanced, globalConfig)

	return enhanced, nil
}

// loadGlobalConfig loads the global configuration
func loadGlobalConfig() (*global.GlobalConfig, error) {
	configPath := global.GlobalConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default global config if not exists
			return getDefaultGlobalConfig(), nil
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read global config")
	}

	var config global.GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse global config")
	}

	return &config, nil
}

// applyGlobalDefaults applies global defaults to the enhanced config
func applyGlobalDefaults(enhanced *EnhancedGuildConfig, globalConfig *global.GlobalConfig) {
	// Apply default provider if not specified locally
	if enhanced.Providers.OpenAI.BaseURL == "" && globalConfig.Providers.Default == "openai" {
		// Load provider-specific config
		loadProviderConfig(enhanced, "openai")
	}
	if enhanced.Providers.Anthropic.BaseURL == "" && globalConfig.Providers.Default == "anthropic" {
		loadProviderConfig(enhanced, "anthropic")
	}

	// Apply global tool settings
	// Tools are additive - local tools are added to global enabled tools
	for _, agent := range enhanced.Agents {
		if len(agent.Tools) == 0 && len(globalConfig.Tools.Enabled) > 0 {
			agent.Tools = globalConfig.Tools.Enabled
		}
	}
}

// loadProviderConfig loads provider-specific configuration from global
func loadProviderConfig(enhanced *EnhancedGuildConfig, provider string) {
	providerConfigPath := global.GlobalProviderConfigPath(provider)

	data, err := os.ReadFile(providerConfigPath)
	if err != nil {
		return // Ignore errors, use defaults
	}

	var providerConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &providerConfig); err != nil {
		return
	}

	// Apply provider-specific settings
	switch provider {
	case "openai":
		if baseURL, ok := providerConfig["base_url"].(string); ok && enhanced.Providers.OpenAI.BaseURL == "" {
			enhanced.Providers.OpenAI.BaseURL = baseURL
		}
	case "anthropic":
		if baseURL, ok := providerConfig["base_url"].(string); ok && enhanced.Providers.Anthropic.BaseURL == "" {
			enhanced.Providers.Anthropic.BaseURL = baseURL
		}
	}
}

// getDefaultGlobalConfig returns default global configuration
func getDefaultGlobalConfig() *global.GlobalConfig {
	return &global.GlobalConfig{
		Providers: global.ProvidersConfig{
			Default:  "anthropic",
			Fallback: []string{"openai", "ollama"},
		},
		Tools: global.ToolsConfig{
			Enabled:  []string{"git", "code", "lsp", "file", "web"},
			Disabled: []string{},
		},
		Cache: global.CacheConfig{
			Embeddings: global.EmbeddingsCacheConfig{
				MaxSizeGB: 10,
				TTLDays:   30,
			},
		},
		Logging: global.LoggingConfig{
			Level:     "info",
			MaxSizeMB: 100,
			MaxFiles:  5,
		},
		UI: global.UIConfig{
			VimMode: true,
			Theme:   "monokai",
		},
		Security: global.GlobalSecurityConfig{
			APIKeys: global.APIKeysConfig{
				Source: "environment",
			},
		},
		LSPServers: make(map[string]global.LSPServerConfig),
	}
}

// GetProjectPaths returns all relevant paths for a project
func GetProjectPaths(projectPath string) map[string]string {
	return map[string]string{
		// Global paths
		"global_dir":    global.GlobalGuildDir(),
		"global_config": global.GlobalConfigPath(),
		"providers_dir": filepath.Join(global.GlobalGuildDir(), "providers"),
		"tools_dir":     filepath.Join(global.GlobalGuildDir(), "tools"),
		"templates_dir": filepath.Join(global.GlobalGuildDir(), "templates"),
		"lsp_dir":       filepath.Join(global.GlobalGuildDir(), "lsp"),

		// Local paths
		"local_dir":    local.LocalGuildDir(projectPath),
		"local_config": local.LocalConfigPath(projectPath),
		"database":     local.LocalDatabasePath(projectPath),
		"corpus":       local.LocalCorpusPath(projectPath),
		"commissions":  local.LocalCommissionsPath(projectPath), // User objectives/goals
		"local_tools":  local.LocalToolsPath(projectPath),
		"workspaces":   local.LocalWorkspacesPath(projectPath),
	}
}
