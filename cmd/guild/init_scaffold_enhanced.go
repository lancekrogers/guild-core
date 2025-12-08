// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

var (
	scaffoldIntegration *ScaffoldIntegration
)

// initScaffoldEnhancedCmd represents the enhanced init command with scaffold integration
var initScaffoldEnhancedCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a Guild project with template-driven scaffolding",
	Long: `Initialize a Guild project using YAML-driven templates or legacy initialization.

This enhanced init command provides:
- Template-driven project scaffolding with customizable presets
- Interactive configuration mode with guided setup
- Backwards compatibility with legacy initialization
- Dry-run mode to preview changes before creating files
- Feature flag support for gradual rollout

Examples:
  guild init my-project                         # Use default template
  guild init my-project --template campaign     # Use specific template  
  guild init my-project --dry-run               # Preview without creating files
  guild init --list-templates                  # Show available templates
  guild init --interactive                     # Interactive configuration mode
  guild init --legacy                          # Force legacy initialization

Template Variables:
  guild init my-project --var project_type=research --var team_size=5
  
Provider Configuration:
  guild init my-project --provider anthropic --model claude-3-sonnet
  
Feature Flags:
  GUILD_SCAFFOLD_ENABLED=true/false    # Enable/disable scaffold integration
  GUILD_SCAFFOLD_INTERACTIVE=true      # Enable interactive mode by default
  GUILD_SCAFFOLD_VERBOSE=false         # Enable verbose output
  GUILD_LEGACY_FALLBACK=true           # Allow fallback to legacy init`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitScaffoldEnhanced,
}

var (
	// Scaffold flags
	initTemplate      string
	initDryRun        bool
	initListTemplates bool
	initForce         bool
	initOutputDir     string
	initConfigFile    string
	initVariables     []string
	initProvider      string
	initModel         string
	initInteractive   bool
	initVerbose       bool
	initLegacy        bool
)

func init() {
	// Initialize scaffold integration
	scaffoldIntegration = NewScaffoldIntegration()

	// Add flags to the command
	initScaffoldEnhancedCmd.Flags().StringVarP(&initTemplate, "template", "t", "",
		"Template to use for initialization (auto-detect if not specified)")
	initScaffoldEnhancedCmd.Flags().BoolVar(&initDryRun, "dry-run", false,
		"Show what would be created without actually creating files")
	initScaffoldEnhancedCmd.Flags().BoolVar(&initListTemplates, "list-templates", false,
		"List available templates and exit")
	initScaffoldEnhancedCmd.Flags().BoolVar(&initForce, "force", false,
		"Overwrite existing files (use with caution)")
	initScaffoldEnhancedCmd.Flags().StringVarP(&initOutputDir, "output", "o", ".",
		"Output directory for the project")
	initScaffoldEnhancedCmd.Flags().StringVar(&initConfigFile, "config", "",
		"Custom scaffold configuration file")
	initScaffoldEnhancedCmd.Flags().StringSliceVarP(&initVariables, "var", "v", nil,
		"Set template variables (format: key=value)")
	initScaffoldEnhancedCmd.Flags().StringVar(&initProvider, "provider", "",
		"Default LLM provider for agents (anthropic, openai, ollama)")
	initScaffoldEnhancedCmd.Flags().StringVar(&initModel, "model", "",
		"Default model for agents")
	initScaffoldEnhancedCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false,
		"Interactive mode with prompts for configuration")
	initScaffoldEnhancedCmd.Flags().BoolVar(&initVerbose, "verbose", false,
		"Enable verbose output")
	initScaffoldEnhancedCmd.Flags().BoolVar(&initLegacy, "legacy", false,
		"Force legacy initialization mode")

	// Register the enhanced command (will replace the existing init command)
	// rootCmd.AddCommand(initScaffoldEnhancedCmd)
}

// runInitScaffoldEnhanced executes the enhanced init command with scaffold integration
func runInitScaffoldEnhanced(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get logger from context
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "guild-init-scaffold")
	ctx = observability.WithOperation(ctx, "runInitScaffoldEnhanced")

	logger.InfoContext(ctx, "Starting Guild enhanced initialization",
		"args", args,
		"scaffold_enabled", scaffoldIntegration.IsEnabled(),
		"template", initTemplate,
		"dry_run", initDryRun,
		"interactive", initInteractive,
		"legacy", initLegacy,
	)

	// Check context early
	if err := ctx.Err(); err != nil {
		logger.ErrorContext(ctx, "Context cancelled during init", "error", err)
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "init command cancelled").
			WithComponent("cli").
			WithOperation("runInitScaffoldEnhanced")
	}

	// Handle list templates request
	if initListTemplates {
		return handleListTemplatesRequest(ctx)
	}

	// Determine whether to use scaffold or legacy initialization
	useScaffold := shouldUseScaffold(ctx, args)

	if useScaffold && scaffoldIntegration.IsEnabled() {
		logger.InfoContext(ctx, "Using scaffold-based initialization")
		return executeScaffoldInit(ctx, cmd, args)
	} else {
		logger.InfoContext(ctx, "Using legacy initialization")
		return executeLegacyInit(ctx, cmd, args)
	}
}

// shouldUseScaffold determines whether to use scaffold or legacy initialization
func shouldUseScaffold(ctx context.Context, args []string) bool {
	// If legacy is explicitly requested, use legacy
	if initLegacy {
		return false
	}

	// If scaffold integration is disabled, use legacy
	if !scaffoldIntegration.IsEnabled() {
		return false
	}

	// If any scaffold-specific flags are used, use scaffold
	if initTemplate != "" || initDryRun || initInteractive || len(initVariables) > 0 {
		return true
	}

	// Default behavior based on environment and feature flags
	return scaffoldIntegration.ShouldUseScaffold(ctx, os.Args)
}

// executeScaffoldInit executes scaffold-based initialization
func executeScaffoldInit(ctx context.Context, cmd *cobra.Command, args []string) error {
	return scaffoldIntegration.ExecuteScaffoldInit(ctx, cmd, args)
}

// executeLegacyInit executes legacy initialization
func executeLegacyInit(ctx context.Context, cmd *cobra.Command, args []string) error {
	// For backwards compatibility, delegate to the existing init implementation
	// This assumes the existing runFastInit function is available

	// Set force flag if it was specified
	fastInitForce = initForce

	// Call the original init function
	return runFastInit(cmd, args)
}

// handleListTemplatesRequest handles the --list-templates flag
func handleListTemplatesRequest(ctx context.Context) error {
	if !scaffoldIntegration.IsEnabled() {
		fmt.Println("❌ Scaffold integration is disabled")
		fmt.Println("   Set GUILD_SCAFFOLD_ENABLED=true to enable template listing")
		return nil
	}

	// Import and use the CLI package
	// This requires importing the scaffold CLI package
	fmt.Println("📋 Available Templates:")
	fmt.Println("(Template listing requires scaffold integration)")

	// TODO: Implement template listing by importing scaffold CLI
	return gerror.New("template listing not yet implemented").
		WithField("suggestion", "use scaffold CLI directly for now")
}

// IntegrateScaffoldWithExistingInit integrates scaffold with existing init command
func IntegrateScaffoldWithExistingInit() {
	if !scaffoldIntegration.IsEnabled() {
		return
	}

	// Add scaffold flags to existing init command
	scaffoldIntegration.AddScaffoldFlags(initCmd)

	// Wrap the existing RunE function
	originalRunE := initCmd.RunE

	initCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Check if scaffold should be used
		if shouldUseScaffold(ctx, args) {
			fmt.Println("🚀 Using enhanced scaffold-based initialization")
			return scaffoldIntegration.ExecuteScaffoldInit(ctx, cmd, args)
		}

		// Use legacy initialization
		fmt.Println("📋 Using legacy initialization")
		return originalRunE(cmd, args)
	}

	// Update command description to mention scaffold capabilities
	initCmd.Long = fmt.Sprintf(`%s

🚀 ENHANCED FEATURES (when scaffold integration is enabled):
- Template-driven project creation with customizable presets
- Interactive configuration mode with guided setup
- Dry-run mode to preview changes before creating files
- Advanced variable substitution and configuration options
- Support for multiple project types and templates

Use --legacy to force legacy initialization behavior.`, initCmd.Long)
}

// enableScaffoldIntegration enables scaffold integration by calling the integration function
func enableScaffoldIntegration() {
	if scaffoldIntegration != nil && scaffoldIntegration.IsEnabled() {
		IntegrateScaffoldWithExistingInit()
	}
}
