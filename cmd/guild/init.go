// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/project"
	"github.com/lancekrogers/guild/pkg/providers"
)

var (
	fastInitForce bool
)

// initCmd represents the fast init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Fast initialization of Guild project",
	Long: `Initialize a Guild project quickly with sensible defaults.

This command provides a fast, non-interactive setup that:
- Auto-detects available AI providers
- Creates Elena (Guild Master) and specialist agents
- Generates optimized configuration
- Sets up daemon registry for automatic startup

For an interactive setup experience with more control, use 'guild setup-wizard'.

Examples:
  guild init                    # Initialize current directory
  guild init ./my-project       # Initialize specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFastInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolVar(&fastInitForce, "force", false, "Force initialization even if already configured")
}

func runFastInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get logger from context
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "guild-init")
	ctx = observability.WithOperation(ctx, "runFastInit")

	logger.InfoContext(ctx, "Starting Guild fast initialization",
		"args", args,
		"force", fastInitForce,
	)

	// Check context early
	if err := ctx.Err(); err != nil {
		logger.ErrorContext(ctx, "Context cancelled during init", "error", err)
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "init command cancelled").
			WithComponent("cli").
			WithOperation("runFastInit")
	}

	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get absolute path").
			WithComponent("cli").
			WithOperation("runFastInit").
			WithDetails("path", projectPath)
	}
	projectPath = absPath

	// Check if already initialized (unless --force)
	if !fastInitForce {
		campaignPath := filepath.Join(projectPath, ".campaign", "campaign.yaml")

		if _, err := os.Stat(campaignPath); err == nil {
			return gerror.New(gerror.ErrCodeAlreadyExists, "project already initialized", nil).
				WithComponent("cli").
				WithOperation("runFastInit").
				WithDetails("path", projectPath).
				WithDetails("hint", "use --force to reinitialize")
		}
	}

	// Use default campaign and project names
	campaignName := "guild-demo"
	projectName := filepath.Base(projectPath)
	if projectName == "." || projectName == "/" {
		// If we're in the root or current directory, use a better name
		if cwd, err := os.Getwd(); err == nil {
			projectName = filepath.Base(cwd)
		} else {
			projectName = "my-project"
		}
	}

	fmt.Println("🏰 Guild Fast Initialization")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📋 Campaign: %s\n", campaignName)
	fmt.Printf("📁 Project: %s\n", projectName)
	fmt.Printf("📍 Location: %s\n", projectPath)
	fmt.Println()

	// Step 1: Create enhanced campaign structure
	fmt.Print("🏗️  Creating campaign structure... ")
	if err := createEnhancedCampaignStructure(ctx, projectPath); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign structure").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")

	// Step 2: Detect project type
	fmt.Print("🔍 Analyzing project type... ")
	detector := project.NewProjectDetector()
	projectType, err := detector.DetectProjectType(projectPath)
	if err != nil {
		fmt.Printf("⚠️ (using generic)\n")
		projectType = &project.ProjectType{
			Name:        "generic",
			Language:    "multiple",
			Framework:   "",
			Description: "Generic project",
		}
	} else {
		fmt.Printf("✅ (%s)\n", projectType.Description)
	}

	// Step 3: Auto-detect AI providers
	fmt.Print("🤖 Detecting AI providers... ")
	providerDetector := providers.NewAutoDetector(5 * time.Second)
	providerResults, err := providerDetector.DetectAll(ctx)
	if err != nil {
		fmt.Printf("⚠️ (continuing with defaults)\n")
		providerResults = []providers.DetectionResult{}
	} else {
		availableCount := 0
		for _, result := range providerResults {
			if result.Available {
				availableCount++
			}
		}
		if availableCount > 0 {
			fmt.Printf("✅ (%d found)\n", availableCount)
		} else {
			fmt.Printf("⚠️ (none detected, using defaults)\n")
		}
	}

	// Step 4: Create campaign configuration
	fmt.Print("⚙️  Creating campaign configuration... ")
	campaignHash := generateCampaignHash(projectPath)
	if err := createCampaignConfig(ctx, projectPath, campaignName, projectName, projectType.Name); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create campaign config").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")

	// Step 5: Create default agent configurations
	fmt.Print("👥 Creating Elena and specialist agents... ")
	if err := createEnhancedAgentConfigs(ctx, projectPath, projectType); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent configs").
			WithComponent("cli").
			WithOperation("runFastInit")
	}

	// Adapt agents to project type
	if err := adaptAgentConfigsToProjectType(ctx, projectPath, projectType); err != nil {
		logger.WarnContext(ctx, "Failed to adapt agent configs", "error", err)
	}
	fmt.Println("✅ (3 agents)")

	// Step 6: Create guild configuration
	fmt.Print("🏰 Creating guild configuration... ")
	if err := createDefaultGuildConfig(ctx, projectPath, projectName); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create guild config").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")


	// Step 7: Initialize database
	fmt.Print("🗄️  Initializing database... ")
	if err := initializeCampaignDatabase(ctx, projectPath, campaignName); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize database").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")

	// Step 8: Create socket registry
	fmt.Print("🔧 Setting up socket registry... ")
	if err := createSocketRegistry(ctx, projectPath, campaignName, campaignHash); err != nil {
		fmt.Printf("⚠️ (manual daemon start required)\n")
	} else {
		fmt.Println("✅")
	}

	// Daemon will start automatically when needed (e.g., guild chat)

	fmt.Println()
	fmt.Println("🏰 Guild successfully initialized!")
	fmt.Println()
	fmt.Println("👑 Elena the Guild Master is ready to lead your team")
	fmt.Println("⚔️  Marcus the Code Artisan stands ready to craft solutions")
	fmt.Println("🛡️  Vera the Quality Guardian protects your software excellence")
	fmt.Println()
	fmt.Println("🚀 Start your adventure:")
	fmt.Println("   guild serve                          # Start daemon (run in separate terminal)")
	fmt.Println("   guild chat                           # Meet Elena and begin chatting")
	fmt.Println("   guild status                         # Check guild status")
	fmt.Println()
	fmt.Println("💡 For more control over setup, use: guild setup-wizard")
	fmt.Println()

	return nil
}
