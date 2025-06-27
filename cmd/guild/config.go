// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/paths"
	"github.com/lancekrogers/guild/pkg/project/global"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Guild configuration",
	Long: `View and manage Guild configuration settings.

Guild uses a hierarchical configuration system:
- Global config: ~/.guild/config.yaml
- Project config: .campaign/guild.yaml
- Provider configs: ~/.guild/providers/*.yaml

API keys are read from environment variables for security.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Display the merged configuration from global and local settings.`,
	RunE:  runConfigShow,
}

// configPathCmd shows configuration file paths
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file locations",
	Long:  `Display the paths to all configuration files used by Guild.`,
	RunE:  runConfigPath,
}

// configValidateCmd validates configuration files
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long:  `Check that all configuration files are valid and properly formatted.`,
	RunE:  runConfigValidate,
}

// configEditCmd opens configuration in editor
var configEditCmd = &cobra.Command{
	Use:   "edit [global|local]",
	Short: "Edit configuration file",
	Long:  `Open the global or local configuration file in your default editor.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigEdit,
}

// configProvidersCmd lists configured providers
var configProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List configured providers",
	Long:  `Display all configured AI providers and their settings.`,
	RunE:  runConfigProviders,
}

// configAgentsCmd lists configured agents
var configAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List configured agents",
	Long:  `Display all configured agents from the project configuration.`,
	RunE:  runConfigAgents,
}

func init() {
	// Add flags
	configShowCmd.Flags().BoolP("global", "g", false, "Show only global configuration")
	configShowCmd.Flags().BoolP("local", "l", false, "Show only local configuration")
	configShowCmd.Flags().BoolP("raw", "r", false, "Show raw YAML without formatting")

	configValidateCmd.Flags().BoolP("fix", "f", false, "Attempt to fix common issues")

	// Register subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configProvidersCmd)
	configCmd.AddCommand(configAgentsCmd)

	// Register config command
	rootCmd.AddCommand(configCmd)
}

// runConfigShow displays the current configuration
func runConfigShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config show cancelled").
			WithComponent("cli").
			WithOperation("config.show")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "show")

	showGlobal, _ := cmd.Flags().GetBool("global")
	showLocal, _ := cmd.Flags().GetBool("local")
	showRaw, _ := cmd.Flags().GetBool("raw")

	logger.InfoContext(ctx, "Displaying configuration",
		"show_global", showGlobal,
		"show_local", showLocal,
		"raw_format", showRaw,
	)

	// If neither flag is set, show merged config
	if !showGlobal && !showLocal {
		return showMergedConfig(ctx, showRaw)
	}

	if showGlobal {
		return showGlobalConfig(ctx, showRaw)
	}

	if showLocal {
		return showLocalConfig(ctx, showRaw)
	}

	return nil
}

// showMergedConfig displays the merged configuration
func showMergedConfig(ctx context.Context, raw bool) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config").
			WithOperation("showMergedConfig")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Loading merged guild configuration")

	// Load the guild configuration
	cfg, err := config.LoadGuildConfig(ctx, ".")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load configuration").
			WithComponent("config").
			WithOperation("showMergedConfig")
	}

	if raw {
		// Marshal to YAML and display
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal config").
				WithComponent("config").
				WithOperation("show")
		}
		fmt.Print(string(data))
	} else {
		fmt.Println("🏰 Guild Configuration (Merged)")
		fmt.Println("═══════════════════════════════")
		fmt.Println()
		displayFormattedConfig(cfg)
	}

	return nil
}

// showGlobalConfig displays only the global configuration
func showGlobalConfig(ctx context.Context, raw bool) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config").
			WithOperation("showGlobalConfig")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)

	globalDir := global.GlobalGuildDir()
	configPath := filepath.Join(globalDir, "config.yaml")

	logger.InfoContext(ctx, "Loading global configuration", "path", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No global configuration found at:", configPath)
		fmt.Println("Run 'guild init --global' to create one.")
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read global config").
			WithComponent("config").
			WithOperation("show")
	}

	if raw {
		fmt.Print(string(data))
	} else {
		var cfg config.GuildConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse config").
				WithComponent("config").
				WithOperation("show")
		}

		fmt.Println("🌍 Global Configuration")
		fmt.Println("═════════════════════")
		fmt.Printf("Path: %s\n\n", configPath)
		displayFormattedConfig(&cfg)
	}

	return nil
}

// showLocalConfig displays only the local project configuration
func showLocalConfig(ctx context.Context, raw bool) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config").
			WithOperation("showLocalConfig")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)

	configPath := filepath.Join(paths.DefaultCampaignDir, "guild.yaml")

	logger.InfoContext(ctx, "Loading local configuration", "path", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No local configuration found at:", configPath)
		fmt.Println("Run 'guild init' in your project directory to create one.")
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read local config").
			WithComponent("config").
			WithOperation("show")
	}

	if raw {
		fmt.Print(string(data))
	} else {
		var cfg config.GuildConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse config").
				WithComponent("config").
				WithOperation("show")
		}

		fmt.Println("📁 Local Configuration")
		fmt.Println("════════════════════")
		fmt.Printf("Path: %s\n\n", configPath)
		displayFormattedConfig(&cfg)
	}

	return nil
}

// displayFormattedConfig shows configuration in a readable format
func displayFormattedConfig(cfg *config.GuildConfig) {
	// Display agents
	if len(cfg.Agents) > 0 {
		fmt.Println("🤖 Agents:")
		for _, agent := range cfg.Agents {
			fmt.Printf("  • %s (%s)\n", agent.Name, agent.ID)
			fmt.Printf("    Type: %s | Provider: %s | Model: %s\n",
				agent.Type, agent.Provider, agent.Model)
			if len(agent.Capabilities) > 0 {
				fmt.Printf("    Capabilities: %s\n", strings.Join(agent.Capabilities, ", "))
			}
		}
		fmt.Println()
	}

	// Display providers
	fmt.Println("🔌 Providers:")
	hasProviders := false

	if cfg.Providers.OpenAI.BaseURL != "" || len(cfg.Providers.OpenAI.Settings) > 0 {
		fmt.Printf("  • OpenAI\n")
		if cfg.Providers.OpenAI.BaseURL != "" {
			fmt.Printf("    Base URL: %s\n", cfg.Providers.OpenAI.BaseURL)
		}
		hasProviders = true
	}

	if cfg.Providers.Anthropic.BaseURL != "" || len(cfg.Providers.Anthropic.Settings) > 0 {
		fmt.Printf("  • Anthropic\n")
		if cfg.Providers.Anthropic.BaseURL != "" {
			fmt.Printf("    Base URL: %s\n", cfg.Providers.Anthropic.BaseURL)
		}
		hasProviders = true
	}

	if cfg.Providers.Ollama.BaseURL != "" || len(cfg.Providers.Ollama.Settings) > 0 {
		fmt.Printf("  • Ollama\n")
		if cfg.Providers.Ollama.BaseURL != "" {
			fmt.Printf("    Base URL: %s\n", cfg.Providers.Ollama.BaseURL)
		}
		hasProviders = true
	}

	if !hasProviders {
		fmt.Println("  No providers configured")
	}
	fmt.Println()

	// Tools would be displayed here if available in config

	// Display storage
	if cfg.Storage.Backend != "" {
		fmt.Println("💾 Storage:")
		fmt.Printf("  Type: %s\n", cfg.Storage.Backend)
		if cfg.Storage.SQLite.Path != "" {
			fmt.Printf("  Path: %s\n", cfg.Storage.SQLite.Path)
		}
		fmt.Println()
	}

	// UI settings would be displayed here if available in config
}

// runConfigPath shows configuration file paths
func runConfigPath(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config path cancelled").
			WithComponent("cli").
			WithOperation("config.path")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "path")

	logger.InfoContext(ctx, "Displaying configuration paths")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get home directory").
			WithComponent("config").
			WithOperation("path")
	}

	fmt.Println("📍 Configuration File Locations")
	fmt.Println("══════════════════════════════")
	fmt.Println()

	// Global paths
	fmt.Println("🌍 Global:")
	fmt.Printf("  Config:    %s\n", filepath.Join(homeDir, ".guild", "config.yaml"))
	fmt.Printf("  Providers: %s\n", filepath.Join(homeDir, ".guild", "providers"))
	fmt.Printf("  Templates: %s\n", filepath.Join(homeDir, ".guild", "templates"))
	fmt.Printf("  Tools:     %s\n", filepath.Join(homeDir, ".guild", "tools"))
	fmt.Println()

	// Local paths
	fmt.Println("📁 Local (Project):")
	fmt.Printf("  Config:    %s\n", filepath.Join(paths.DefaultCampaignDir, "guild.yaml"))
	fmt.Printf("  Database:  %s\n", filepath.Join(paths.DefaultCampaignDir, "memory.db"))
	fmt.Printf("  Corpus:    %s\n", filepath.Join(paths.DefaultCampaignDir, "corpus.yaml"))
	fmt.Printf("  Archives:  %s\n", filepath.Join(paths.DefaultCampaignDir, "archives"))
	fmt.Println()

	// Environment variables
	fmt.Println("🔐 Environment Variables:")
	providers := []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"DEEPSEEK_API_KEY",
		"OLLAMA_HOST",
	}

	for _, env := range providers {
		if os.Getenv(env) != "" {
			fmt.Printf("  %s: [SET]\n", env)
		} else {
			fmt.Printf("  %s: [NOT SET]\n", env)
		}
	}

	return nil
}

// runConfigValidate validates configuration files
func runConfigValidate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config validate cancelled").
			WithComponent("cli").
			WithOperation("config.validate")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "validate")

	attemptFix, _ := cmd.Flags().GetBool("fix")

	logger.InfoContext(ctx, "Validating configuration files", "attempt_fix", attemptFix)

	fmt.Println("🔍 Validating Configuration")
	fmt.Println("═════════════════════════")
	fmt.Println()

	issues := []string{}

	// Check global config
	homeDir, _ := os.UserHomeDir()
	globalConfigPath := filepath.Join(homeDir, ".guild", "config.yaml")

	if _, err := os.Stat(globalConfigPath); err == nil {
		data, err := os.ReadFile(globalConfigPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Cannot read global config: %v", err))
		} else {
			var cfg config.GuildConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				issues = append(issues, fmt.Sprintf("Invalid YAML in global config: %v", err))
			} else {
				fmt.Println("✅ Global config: Valid")
			}
		}
	} else {
		fmt.Println("ℹ️  Global config: Not found (optional)")
	}

	// Check local config
	localConfigPath := filepath.Join(paths.DefaultCampaignDir, "guild.yaml")

	if _, err := os.Stat(localConfigPath); err == nil {
		data, err := os.ReadFile(localConfigPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Cannot read local config: %v", err))
		} else {
			var cfg config.GuildConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				issues = append(issues, fmt.Sprintf("Invalid YAML in local config: %v", err))
			} else {
				fmt.Println("✅ Local config: Valid")

				// Validate agent configurations
				for _, agent := range cfg.Agents {
					if agent.ID == "" {
						issues = append(issues, fmt.Sprintf("Agent missing ID: %s", agent.Name))
					}
					if agent.Type == "" {
						issues = append(issues, fmt.Sprintf("Agent %s missing type", agent.ID))
					}
					if agent.Provider == "" {
						issues = append(issues, fmt.Sprintf("Agent %s missing provider", agent.ID))
					}
				}
			}
		}
	} else {
		fmt.Println("ℹ️  Local config: Not found")
	}

	// Check for required environment variables based on configured providers
	fmt.Println()
	fmt.Println("🔐 Checking API Keys:")

	requiredEnvs := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
	}

	// Load config to check which providers are used
	cfg, err := config.LoadGuildConfig(ctx, ".")
	if err == nil && cfg != nil {
		for _, agent := range cfg.Agents {
			if envVar, ok := requiredEnvs[agent.Provider]; ok {
				if os.Getenv(envVar) == "" {
					issues = append(issues, fmt.Sprintf("Provider %s used by agent %s but %s not set",
						agent.Provider, agent.ID, envVar))
				} else {
					fmt.Printf("  ✅ %s: Set\n", envVar)
				}
			}
		}
	}

	// Summary
	fmt.Println()
	if len(issues) == 0 {
		fmt.Println("✅ All configuration files are valid!")
	} else {
		fmt.Printf("❌ Found %d issue(s):\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}

		if attemptFix {
			fmt.Println("\n🔧 Attempting to fix issues...")
			// TODO: Implement auto-fix logic for common issues
			fmt.Println("Auto-fix not yet implemented.")
		}
	}

	return nil
}

// runConfigEdit opens configuration in editor
func runConfigEdit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config edit cancelled").
			WithComponent("cli").
			WithOperation("config.edit")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "edit")

	var configPath string

	if len(args) == 0 || args[0] == "local" {
		configPath = filepath.Join(paths.DefaultCampaignDir, "guild.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return gerror.New(gerror.ErrCodeNotFound, "no local configuration found", nil).
				WithComponent("config").
				WithOperation("edit").
				WithDetails("path", configPath)
		}
	} else if args[0] == "global" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get home directory").
				WithComponent("config").
				WithOperation("edit")
		}
		configPath = filepath.Join(homeDir, ".guild", "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return gerror.New(gerror.ErrCodeNotFound, "no global configuration found", nil).
				WithComponent("config").
				WithOperation("edit").
				WithDetails("path", configPath)
		}
	} else {
		return gerror.New(gerror.ErrCodeInvalidInput, "specify 'global' or 'local'", nil).
			WithComponent("config").
			WithOperation("edit")
	}

	// Try common editors
	editors := []string{
		os.Getenv("EDITOR"),
		os.Getenv("VISUAL"),
		"vim",
		"vi",
		"nano",
		"emacs",
		"code", // VS Code
	}

	logger.InfoContext(ctx, "Attempting to open config in editor", "path", configPath)

	for _, editor := range editors {
		if editor == "" {
			continue
		}

		logger.InfoContext(ctx, "Trying editor", "editor", editor)

		cmd := exec.Command(editor, configPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			logger.InfoContext(ctx, "Successfully opened config with editor", "editor", editor)
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeInternal, "no suitable editor found", nil).
		WithComponent("config").
		WithOperation("edit").
		WithDetails("tried", strings.Join(editors, ", "))
}

// runConfigProviders lists configured providers
func runConfigProviders(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config providers cancelled").
			WithComponent("cli").
			WithOperation("config.providers")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "providers")

	logger.InfoContext(ctx, "Loading configuration to list providers")

	cfg, err := config.LoadGuildConfig(ctx, ".")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load configuration").
			WithComponent("config").
			WithOperation("providers")
	}

	if cfg == nil {
		fmt.Println("No configuration found.")
		return nil
	}

	fmt.Println("🔌 Configured Providers")
	fmt.Println("════════════════════")
	fmt.Println()

	// Check each provider
	if cfg.Providers.OpenAI.BaseURL != "" || len(cfg.Providers.OpenAI.Settings) > 0 || os.Getenv("OPENAI_API_KEY") != "" {
		fmt.Printf("📡 OpenAI\n")
		if os.Getenv("OPENAI_API_KEY") != "" {
			fmt.Printf("   Status: ✅ API key set\n")
		} else {
			fmt.Printf("   Status: ❌ No API key (OPENAI_API_KEY)\n")
		}
		if cfg.Providers.OpenAI.BaseURL != "" {
			fmt.Printf("   Base URL: %s\n", cfg.Providers.OpenAI.BaseURL)
		}
		fmt.Println()
	}

	if cfg.Providers.Anthropic.BaseURL != "" || len(cfg.Providers.Anthropic.Settings) > 0 || os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Printf("📡 Anthropic\n")
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			fmt.Printf("   Status: ✅ API key set\n")
		} else {
			fmt.Printf("   Status: ❌ No API key (ANTHROPIC_API_KEY)\n")
		}
		if cfg.Providers.Anthropic.BaseURL != "" {
			fmt.Printf("   Base URL: %s\n", cfg.Providers.Anthropic.BaseURL)
		}
		fmt.Println()
	}

	if cfg.Providers.DeepSeek.BaseURL != "" || len(cfg.Providers.DeepSeek.Settings) > 0 || os.Getenv("DEEPSEEK_API_KEY") != "" {
		fmt.Printf("📡 DeepSeek\n")
		if os.Getenv("DEEPSEEK_API_KEY") != "" {
			fmt.Printf("   Status: ✅ API key set\n")
		} else {
			fmt.Printf("   Status: ❌ No API key (DEEPSEEK_API_KEY)\n")
		}
		if cfg.Providers.DeepSeek.BaseURL != "" {
			fmt.Printf("   Base URL: %s\n", cfg.Providers.DeepSeek.BaseURL)
		}
		fmt.Println()
	}

	if cfg.Providers.Ollama.BaseURL != "" || len(cfg.Providers.Ollama.Settings) > 0 {
		fmt.Printf("📡 Ollama\n")
		if cfg.Providers.Ollama.BaseURL != "" {
			fmt.Printf("   Base URL: %s\n", cfg.Providers.Ollama.BaseURL)
		} else {
			fmt.Printf("   Base URL: http://localhost:11434 (default)\n")
		}
		fmt.Println()
	}

	// Show Ollama status if available
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		fmt.Printf("🦙 Ollama\n")
		fmt.Printf("   Host: %s\n", ollamaHost)
		fmt.Printf("   Status: ✅ Configured\n")
	}

	return nil
}

// runConfigAgents lists configured agents
func runConfigAgents(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "config agents cancelled").
			WithComponent("cli").
			WithOperation("config.agents")
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config")
	ctx = observability.WithOperation(ctx, "agents")

	logger.InfoContext(ctx, "Loading configuration to list agents")

	cfg, err := config.LoadGuildConfig(ctx, ".")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load configuration").
			WithComponent("config").
			WithOperation("agents")
	}

	if cfg == nil || len(cfg.Agents) == 0 {
		fmt.Println("No agents configured in this project.")
		fmt.Println("\nTo configure agents, edit", filepath.Join(paths.DefaultCampaignDir, "guild.yaml"))
		return nil
	}

	fmt.Println("🤖 Configured Agents")
	fmt.Println("══════════════════")
	fmt.Println()

	for _, agent := range cfg.Agents {
		fmt.Printf("🎯 %s\n", agent.Name)
		fmt.Printf("   ID: %s\n", agent.ID)
		fmt.Printf("   Type: %s\n", agent.Type)
		fmt.Printf("   Provider: %s | Model: %s\n", agent.Provider, agent.Model)

		if agent.CostMagnitude > 0 {
			var costIcon string
			switch {
			case agent.CostMagnitude <= 1:
				costIcon = "💰" // Very cheap
			case agent.CostMagnitude <= 3:
				costIcon = "💰💰" // Moderate
			case agent.CostMagnitude <= 5:
				costIcon = "💰💰💰" // Expensive
			default:
				costIcon = "💰💰💰💰" // Very expensive
			}
			fmt.Printf("   Cost: %s %d\n", costIcon, agent.CostMagnitude)
		}

		if len(agent.Capabilities) > 0 {
			fmt.Printf("   Capabilities: %s\n", strings.Join(agent.Capabilities, ", "))
		}

		if len(agent.Tools) > 0 {
			fmt.Printf("   Tools: %s\n", strings.Join(agent.Tools, ", "))
		}

		if agent.SystemPrompt != "" {
			preview := agent.SystemPrompt
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Printf("   System Prompt: %s\n", preview)
		}

		fmt.Println()
	}

	fmt.Printf("Total: %d agents configured\n", len(cfg.Agents))

	return nil
}
