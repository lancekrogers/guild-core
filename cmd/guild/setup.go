// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-core/internal/setup"
	uisetup "github.com/guild-framework/guild-core/internal/ui/setup"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/project"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for Guild providers and configuration",
	Long: `Launch an interactive wizard to configure your Guild providers and agents.

The setup wizard will:
🔍 Auto-detect available providers (Claude Code, Ollama, environment variables)
🔑 Validate API keys before saving configuration  
💰 Show cost estimates for different models
🤖 Create sensible default agent configurations
⚙️  Support multi-provider setups

The wizard guides you through provider configuration in under 2 minutes.`,
	RunE: runSetup,
}

var (
	setupQuickMode bool
	setupForce     bool
	setupProvider  string
)

func init() {
	setupCmd.Flags().BoolVar(&setupQuickMode, "quick", false, "Quick setup with sensible defaults")
	setupCmd.Flags().BoolVar(&setupForce, "force", false, "Force setup even if already configured")
	setupCmd.Flags().StringVar(&setupProvider, "provider", "", "Setup specific provider only (openai, anthropic, ollama, etc.)")

	// Register the setup command
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get current working directory
	path := "."
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve path").
			WithComponent("setup").
			WithOperation("runSetup")
	}

	// Check if project is initialized
	if !project.IsProjectInitialized(path) {
		fmt.Println("🏰 Guild project not found. Initializing first...")
		if err := project.InitializeProject(path); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize project").
				WithComponent("setup").
				WithOperation("runSetup")
		}
		fmt.Printf("✅ Guild project initialized at %s\n\n", absPath)
	}

	// Check if already configured (unless force is used)
	if !setupForce {
		if isAlreadySetup, err := setup.IsProjectSetup(ctx, path); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to check setup status").
				WithComponent("setup").
				WithOperation("runSetup")
		} else if isAlreadySetup {
			fmt.Println("🏰 Guild is already configured!")
			fmt.Println("\nIf you want to reconfigure, use: guild setup --force")
			fmt.Println("To add more providers, use: guild setup --provider <name>")
			return nil
		}
	}

	// Create setup configuration
	setupConfig := &setup.Config{
		ProjectPath:  path,
		QuickMode:    setupQuickMode,
		Force:        setupForce,
		ProviderOnly: setupProvider,
	}

	// Welcome message
	if !setupQuickMode {
		fmt.Println("🏰 Welcome to the Guild Setup Wizard!")
		fmt.Println("═══════════════════════════════════════")
		fmt.Println()
		fmt.Println("This wizard will help you configure AI providers and create")
		fmt.Println("your first team of specialized artisans (agents) to tackle")
		fmt.Println("complex tasks through coordinated collaboration.")
		fmt.Println()
	}

	// Create wizard
	wizard, err := setup.NewWizard(ctx, setupConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create setup wizard").
			WithComponent("setup").
			WithOperation("runSetup")
	}

	// Run appropriate mode
	if setupQuickMode {
		// Run in quick mode without TUI
		if err := wizard.RunQuickMode(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "quick setup failed").
				WithComponent("setup").
				WithOperation("runSetup")
		}
	} else {
		// Run interactive TUI
		model := uisetup.NewWizardTUIModel(ctx, wizard)
		program := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := program.Run(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "setup wizard failed").
				WithComponent("setup").
				WithOperation("runSetup")
		}
	}

	// Success message
	fmt.Println("\n🎉 Guild setup complete!")
	fmt.Println("═════════════════════════")
	fmt.Println()
	fmt.Println("Your Guild is now ready for action. Try these commands:")
	fmt.Println("  guild chat                    # Interactive coordination")
	fmt.Println("  guild commission create       # Create new objectives")
	fmt.Println("  guild kanban view            # View task progress")
	fmt.Println()
	fmt.Println("💡 Pro tip: Use 'guild --help' to explore all commands")

	return nil
}

// setupListCmd shows current setup status
var setupListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current provider setup status",
	Long:  "Display the current configuration of providers and agents.",
	RunE:  runSetupList,
}

func runSetupList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	path := "."

	// Check if project is initialized
	if !project.IsProjectInitialized(path) {
		fmt.Println("❌ No Guild project found in current directory")
		fmt.Println("   Run 'guild init' first")
		return nil
	}

	// Get setup status
	status, err := setup.GetSetupStatus(ctx, path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get setup status").
			WithComponent("setup").
			WithOperation("runSetupList")
	}

	// Display status
	fmt.Println("🏰 Guild Setup Status")
	fmt.Println("═══════════════════════")
	fmt.Println()

	if !status.IsConfigured {
		fmt.Println("❌ Not configured - run 'guild setup' to configure")
		return nil
	}

	fmt.Printf("✅ Configured (%d providers, %d agents)\n\n",
		len(status.Providers), len(status.Agents))

	// Show providers
	fmt.Println("🔌 Providers:")
	for _, provider := range status.Providers {
		statusIcon := "✅"
		if !provider.Available {
			statusIcon = "❌"
		}
		fmt.Printf("  %s %s", statusIcon, provider.Name)
		if provider.Models > 0 {
			fmt.Printf(" (%d models)", provider.Models)
		}
		fmt.Println()
	}

	// Show agents
	fmt.Println("\n🤖 Agents:")
	for _, agent := range status.Agents {
		fmt.Printf("  • %s (%s) - %s/%s\n",
			agent.Name, agent.Type, agent.Provider, agent.Model)
	}

	return nil
}

func init() {
	setupCmd.AddCommand(setupListCmd)
}
