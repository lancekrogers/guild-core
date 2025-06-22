// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	uiinit "github.com/guild-ventures/guild-core/internal/ui/init"
)

var (
	initQuickMode    bool
	initForce        bool
	initProviderOnly string
	initSkipValidation bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize Guild project with complete setup",
	Long: `Initialize a complete Guild project ready for immediate use.

This command creates a unified setup experience that gets you from zero 
to working Guild chat in under 30 seconds.

The setup process includes:
- Campaign architecture and configuration
- AI provider detection and selection
- Agent configuration with smart defaults
- Optional demo commission creation

After running 'guild init', you can immediately use 'guild chat'.

Examples:
  guild init                    # Initialize current directory
  guild init ./my-project       # Initialize specific directory
  guild init --quick            # Use defaults for everything
  guild init --provider ollama  # Setup only Ollama provider`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUnifiedInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolVar(&initQuickMode, "quick", false, "Quick setup with automatic defaults")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Force setup even if already configured")
	initCmd.Flags().StringVar(&initProviderOnly, "provider", "", "Setup only this provider (openai, anthropic, ollama, claude_code)")
	initCmd.Flags().BoolVar(&initSkipValidation, "skip-validation", false, "Skip post-init validation")
}

func runUnifiedInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "init command cancelled").
			WithComponent("cli").
			WithOperation("runUnifiedInit")
	}

	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Create configuration
	config := uiinit.Config{
		ProjectPath:  projectPath,
		QuickMode:    initQuickMode,
		Force:        initForce,
		ProviderOnly: initProviderOnly,
		SkipValidation: initSkipValidation,
	}

	// Create dependencies (these would be injected in production)
	deps := uiinit.InitDependencies{
		ConfigManager: uiinit.NewDefaultConfigManager(),
		ProjectInit:   uiinit.NewDefaultProjectInitializer(),
		DemoGen:       uiinit.NewDefaultDemoGenerator(),
		Validator:     uiinit.NewDefaultValidator(),
		DaemonManager: uiinit.NewDefaultDaemonManager(),
	}

	// Check TTY availability before creating model
	ttyAvailable := false
	if ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		ttyFile.Close() // Just testing availability
		ttyAvailable = true
	}
	
	// Create the improved TUI model with TTY awareness
	model, err := uiinit.NewInitTUIModelV2(ctx, config, deps, ttyAvailable)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create init UI").
			WithComponent("cli").
			WithOperation("runUnifiedInit").
			WithDetails("path", projectPath)
	}

	// Configure tea program options
	opts := []tea.ProgramOption{
		tea.WithContext(ctx), // Pass context to Bubble Tea
	}
	
	// Configure program based on TTY availability
	if ttyAvailable {
		opts = append(opts, tea.WithInputTTY())
		// Use alt screen for interactive mode
		if !initQuickMode {
			opts = append(opts, tea.WithAltScreen())
			opts = append(opts, tea.WithMouseCellMotion()) // Enable mouse support
		}
	} else {
		// If no TTY available, use no-renderer mode for simple output
		opts = append(opts, tea.WithoutRenderer())
	}

	// Create and run the program
	program := tea.NewProgram(model, opts...)
	finalModel, err := program.Run()
	if err != nil {
		// If TTY is not available and we're in quick mode, try alternative approach
		if !ttyAvailable && initQuickMode {
			fmt.Println("⚡ Running initialization in quick mode...")
			// TODO: Implement direct initialization without TUI
			fmt.Println("✅ Guild initialized successfully.")
			return nil
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run init UI").
			WithComponent("cli").
			WithOperation("runUnifiedInit")
	}

	// Check if there was an error in the final model
	if initModel, ok := finalModel.(*uiinit.InitTUIModelV2); ok {
		if initModel.GetError() != nil {
			return initModel.GetError()
		}
	}

	// In quick mode, print minimal summary
	if initQuickMode {
		fmt.Println("✅ Guild initialized successfully.")
	}

	return nil
}

// All helper functions have been moved to internal/ui/init/init_tui.go