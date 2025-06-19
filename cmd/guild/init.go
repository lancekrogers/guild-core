// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/setup"
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

	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve path").
			WithComponent("cli").WithOperation("runUnifiedInit").WithDetails("path", projectPath)
	}

	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Welcome message
	if !initQuickMode {
		fmt.Println("🏰 Welcome to Guild Framework!")
		fmt.Println("Let's set up your first campaign to get started.")
		fmt.Println()
	}

	// Step 1: Campaign Setup
	campaignName, projectName, err := setupCampaign(ctx, absPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup campaign").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	if !initQuickMode {
		fmt.Printf("✅ Campaign: %s\n", campaignName)
		fmt.Printf("✅ Project: %s\n", projectName)
		fmt.Println()
	}

	// Step 2: Provider Detection and Selection
	if !initQuickMode {
		fmt.Println("🔍 Detecting available AI providers...")
	}

	setupConfig := &setup.Config{
		ProjectPath:  absPath,
		QuickMode:    initQuickMode,
		Force:        initForce,
		ProviderOnly: initProviderOnly,
	}

	wizard, err := setup.NewWizard(ctx, setupConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create setup wizard").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 3: Initialize project structure first
	if !project.IsProjectInitialized(absPath) {
		if !initQuickMode {
			fmt.Print("📁 Creating project directory structure... ")
		}

		// Check for cancellation before project initialization
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled before project setup").
				WithComponent("cli").WithOperation("runUnifiedInit")
		}

		if err := project.InitializeProject(absPath); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project structure").
				WithComponent("cli").WithOperation("runUnifiedInit").WithDetails("path", absPath)
		}

		if !initQuickMode {
			fmt.Println("✅")
		}
	}

	// Step 4: Create Phase 0 hierarchical configuration
	if !initQuickMode {
		fmt.Print("🎯 Creating Phase 0 configuration... ")
	}

	// Check for cancellation before config creation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled before config creation").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	if err := createPhase0Configuration(ctx, absPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create Phase 0 configuration").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	if !initQuickMode {
		fmt.Println("✅")
	}

	// Step 5: Run provider setup wizard and integrate with Phase 0
	if !initQuickMode {
		fmt.Println("⚙️  Setting up AI providers and agents...")
	}

	// Check for cancellation before wizard
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled before wizard").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	if err := wizard.Run(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run setup wizard").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 5.1: Integrate with Phase 0 hierarchical config
	if err := integrateWithPhase0Config(ctx, absPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to integrate with Phase 0 config").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 5.2: Create campaign reference for detection system
	if err := createCampaignReference(ctx, absPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create campaign reference").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 5.3: Create socket registry for daemon support
	if err := createSocketRegistry(ctx, absPath, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create socket registry").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 6: Create demo commission (optional)
	if !initQuickMode {
		if askYesNo(ctx, "🎯 Create a demo commission to get started?", true) {
			if err := createDemoCommission(ctx, absPath, campaignName); err != nil {
				fmt.Printf("⚠️  Warning: Could not create demo commission: %v\n", err)
			} else {
				fmt.Println("✅ Demo commission created")
			}
		}
	} else {
		// In quick mode, always create demo commission
		if err := createDemoCommission(ctx, absPath, campaignName); err != nil {
			// Don't fail the whole setup for demo commission
			fmt.Printf("⚠️  Warning: Could not create demo commission: %v\n", err)
		}
	}

	// Step 7: Run post-init validation
	if !initSkipValidation {
		if !initQuickMode {
			fmt.Println()
			fmt.Println("🔍 Validating setup...")
		}

		// Check for cancellation before validation
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled before validation").
				WithComponent("cli").WithOperation("runUnifiedInit")
		}

		validator := setup.NewInitValidator(absPath)
		if err := validator.Validate(ctx); err != nil {
			// Validation failed - show results anyway
			validator.PrintResults()
			return gerror.Wrap(err, gerror.ErrCodeValidation, "post-init validation failed").
				WithComponent("cli").WithOperation("runUnifiedInit")
		}

		// Show validation results if not in quick mode
		if !initQuickMode {
			validator.PrintResults()
		} else if validator.HasFailures() {
			// In quick mode, only show if there are failures
			validator.PrintResults()
		}
	}

	// Step 8: Success summary
	if !initQuickMode {
		fmt.Println()
	}
	fmt.Println("🎉 Guild is ready!")
	fmt.Println()
	fmt.Printf("✅ Campaign: %s\n", campaignName)
	fmt.Printf("✅ Project: %s\n", projectName)
	fmt.Printf("✅ Location: %s\n", absPath)
	fmt.Printf("✅ Database: Initialized (.guild/memory.db)\n")
	fmt.Printf("✅ Daemon: Ready to start\n")
	fmt.Println()

	// Step 9: Next steps
	fmt.Println("🚀 Try these commands:")
	fmt.Println()
	fmt.Println("  guild chat          # Start chatting with your agents")
	fmt.Println("  guild status        # Check system status")
	fmt.Println("  guild serve         # Start daemon manually (auto-starts with chat)")
	fmt.Println("  guild commission list # See your commissions")
	fmt.Println()
	fmt.Println("📚 Learn more:")
	fmt.Println()
	fmt.Println("  guild help          # See all commands")
	fmt.Println("  guild agent list    # View your agents")
	fmt.Println("  guild campaign list # See your campaigns")
	fmt.Println()
	fmt.Printf("Ready to start? Run: guild chat\n")

	return nil
}

// setupCampaign handles interactive campaign configuration with context support
func setupCampaign(ctx context.Context, projectPath string) (campaignName, projectName string, err error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return "", "", gerror.Wrap(err, gerror.ErrCodeCancelled, "campaign setup cancelled").
			WithComponent("cli").WithOperation("setupCampaign")
	}

	// Check if already in a campaign
	if _, err := campaign.DetectCampaign(projectPath, ""); err == nil {
		if !initForce {
			return "", "", gerror.New(gerror.ErrCodeValidation, "already in a campaign - use --force to reinitialize", nil).
				WithComponent("cli").WithOperation("setupCampaign")
		}
	}

	// Default campaign name
	defaultCampaign := "guild-demo"
	if !initQuickMode {
		fmt.Printf("Campaign name [%s]: ", defaultCampaign)
		campaignName = readInputWithContext(ctx, defaultCampaign)
	} else {
		campaignName = defaultCampaign
	}

	// Check for cancellation after user input
	if err := ctx.Err(); err != nil {
		return "", "", gerror.Wrap(err, gerror.ErrCodeCancelled, "campaign setup cancelled after name input").
			WithComponent("cli").WithOperation("setupCampaign")
	}

	// Default project name
	defaultProject := filepath.Base(projectPath)
	if !initQuickMode {
		fmt.Printf("Project name [%s]: ", defaultProject)
		projectName = readInputWithContext(ctx, defaultProject)
	} else {
		projectName = defaultProject
	}

	return campaignName, projectName, nil
}

// createPhase0Configuration creates the Phase 0 hierarchical configuration structure
func createPhase0Configuration(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "Phase 0 configuration creation cancelled").
			WithComponent("cli").WithOperation("createPhase0Configuration")
	}

	// Step 1: Create campaign.yml (Phase 0 campaign configuration)
	campaignConfig := &config.CampaignConfig{
		Name:        campaignName,
		Description: fmt.Sprintf("Campaign %s - automated multi-agent development", campaignName),
		ProjectSettings: map[string]interface{}{
			"project_name": projectName,
			"created_at":   time.Now().Format(time.RFC3339),
			"version":      "1.0.0",
		},
		CommissionMappings: make(map[string][]string),
		LastSelectedGuild:  "default", // Will be created next
	}

	if err := config.SaveCampaignConfig(ctx, projectPath, campaignConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save campaign configuration").
			WithComponent("cli").WithOperation("createPhase0Configuration").WithDetails("campaign", campaignName)
	}

	// Step 2: Create default guild.yml structure
	guildConfig := &config.GuildConfigFile{
		Guilds: map[string]config.GuildDefinition{
			"default": {
				Purpose:     "General development tasks and project management",
				Description: "Default guild for handling various development tasks, code generation, testing, and project coordination",
				Agents:      []string{"manager", "developer", "tester"}, // Will be created in agents/
				Coordination: &config.CoordinationSettings{
					MaxParallelTasks: 3,
					ReviewRequired:   false,
					AutoHandoff:      true,
				},
			},
		},
	}

	if err := config.SaveGuildConfigFile(ctx, projectPath, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save guild configuration").
			WithComponent("cli").WithOperation("createPhase0Configuration").WithDetails("guild_file", "guild.yml")
	}

	// Step 3: Create agents directory structure
	agentsDir := filepath.Join(projectPath, ".guild", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("cli").WithOperation("createPhase0Configuration").WithDetails("dir", agentsDir)
	}

	return nil
}

// integrateWithPhase0Config integrates setup wizard results with Phase 0 hierarchical configuration
func integrateWithPhase0Config(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "Phase 0 integration cancelled").
			WithComponent("cli").WithOperation("integrateWithPhase0Config")
	}

	// Load the guild config created by the setup wizard
	guildConfigPath := filepath.Join(projectPath, ".guild", "guild.yaml")
	if _, err := os.Stat(guildConfigPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "guild config from wizard not found").
			WithComponent("cli").WithOperation("integrateWithPhase0Config").WithDetails("path", guildConfigPath)
	}

	// Read the wizard-generated config
	wizardConfigData, err := os.ReadFile(guildConfigPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read wizard config").
			WithComponent("cli").WithOperation("integrateWithPhase0Config")
	}

	// Parse the wizard config to extract agent configurations
	var wizardConfig config.GuildConfig
	if err := yaml.Unmarshal(wizardConfigData, &wizardConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "failed to parse wizard config").
			WithComponent("cli").WithOperation("integrateWithPhase0Config")
	}

	// Create individual agent files in agents/ directory
	for _, agent := range wizardConfig.Agents {
		if err := createAgentConfig(ctx, projectPath, &agent); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create agent config for %s", agent.Name).
				WithComponent("cli").WithOperation("integrateWithPhase0Config")
		}
	}

	// Move the wizard config to project.yaml to preserve it
	projectConfigPath := filepath.Join(projectPath, ".guild", "project.yaml")
	if err := os.WriteFile(projectConfigPath, wizardConfigData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save project config").
			WithComponent("cli").WithOperation("integrateWithPhase0Config")
	}

	// Note: We do NOT remove guild.yaml here as it may contain the campaign reference
	// The campaign reference will be created in a separate step

	return nil
}

// createAgentConfig creates an individual agent configuration file
func createAgentConfig(ctx context.Context, projectPath string, agent *config.AgentConfig) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "agent config creation cancelled").
			WithComponent("cli").WithOperation("createAgentConfig").WithDetails("agent", agent.Name)
	}

	agentPath := filepath.Join(projectPath, ".guild", "agents", agent.Name+".yml")

	// Validate agent config
	if err := agent.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid agent configuration").
			WithComponent("cli").WithOperation("createAgentConfig").WithDetails("agent", agent.Name)
	}

	// Convert to YAML
	agentData, err := yaml.Marshal(agent)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithComponent("cli").WithOperation("createAgentConfig").WithDetails("agent", agent.Name)
	}

	// Write agent file
	if err := os.WriteFile(agentPath, agentData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config").
			WithComponent("cli").WithOperation("createAgentConfig").WithDetails("path", agentPath)
	}

	return nil
}

// createCampaignReference creates the campaign reference file for detection system
func createCampaignReference(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "campaign reference creation cancelled").
			WithComponent("cli").WithOperation("createCampaignReference")
	}

	// Create campaign reference structure
	ref := campaign.CampaignReference{
		Campaign:    campaignName,
		Project:     projectName,
		Description: fmt.Sprintf("Project %s in campaign %s", projectName, campaignName),
	}

	// Create the reference file path (.guild/guild.yaml - note .yaml extension)
	refPath := filepath.Join(projectPath, ".guild", "guild.yaml")
	
	// Marshal to YAML
	refData, err := yaml.Marshal(ref)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign reference").
			WithComponent("cli").WithOperation("createCampaignReference")
	}

	// Write the reference file
	if err := os.WriteFile(refPath, refData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign reference").
			WithComponent("cli").WithOperation("createCampaignReference").
			WithDetails("path", refPath)
	}

	return nil
}

// createSocketRegistry creates the socket registry for daemon support
func createSocketRegistry(ctx context.Context, projectPath, campaignName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "socket registry creation cancelled").
			WithComponent("cli").WithOperation("createSocketRegistry")
	}

	// Create socket registry
	if err := daemon.SaveSocketRegistry(projectPath, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save socket registry").
			WithComponent("cli").WithOperation("createSocketRegistry")
	}

	return nil
}

// createDemoCommission creates a sample commission for new users with context support
func createDemoCommission(ctx context.Context, projectPath, campaignName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "demo commission creation cancelled").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	commissionsDir := filepath.Join(projectPath, ".guild", "objectives", "refined")
	if err := os.MkdirAll(commissionsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	// Create demo commission generator
	generator := setup.NewDemoCommissionGenerator()

	// Gather project info for recommendation
	projectInfo := map[string]interface{}{
		"project_name": filepath.Base(projectPath),
		"campaign_name": campaignName,
	}

	// Get recommended demo type
	recommendedType, reason := generator.GetRecommendedDemo(ctx, projectInfo)

	// In quick mode, use the recommended type directly
	demoType := recommendedType
	if !initQuickMode {
		fmt.Printf("\n🎯 Selecting demo commission type...\n")
		fmt.Printf("   Recommendation: %s (%s)\n\n", recommendedType, reason)

		// Show available options
		fmt.Println("Available demo types:")
		fmt.Println("  1. api         - RESTful API with testing and documentation")
		fmt.Println("  2. webapp      - Modern web dashboard with real-time features")
		fmt.Println("  3. cli         - Developer productivity CLI tool")
		fmt.Println("  4. data        - Data pipeline and analytics platform")
		fmt.Println("  5. microservices - Cloud-native e-commerce platform")
		fmt.Println("  6. ai          - AI-powered recommendation system")
		fmt.Printf("\nSelect demo type [%s]: ", recommendedType)

		input := readInputWithContext(ctx, string(recommendedType))
		// Map user input to demo type
		switch strings.ToLower(input) {
		case "1", "api":
			demoType = setup.DemoTypeAPIService
		case "2", "webapp", "web":
			demoType = setup.DemoTypeWebApp
		case "3", "cli":
			demoType = setup.DemoTypeCLITool
		case "4", "data", "analytics":
			demoType = setup.DemoTypeDataAnalysis
		case "5", "microservices", "micro":
			demoType = setup.DemoTypeMicroservices
		case "6", "ai", "ml":
			demoType = setup.DemoTypeAI
		default:
			// Use recommended if input is invalid
			demoType = recommendedType
		}
	}

	// Generate the commission content
	demoContent, err := generator.GenerateCommission(ctx, demoType)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate demo commission").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	// Create filename based on demo type
	fileName := fmt.Sprintf("demo-%s.md", string(demoType))
	if demoType == setup.DemoTypeDefault {
		fileName = "demo-api-development.md"
	}

	commissionPath := filepath.Join(commissionsDir, fileName)
	if err := os.WriteFile(commissionPath, []byte(demoContent), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write demo commission").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	return nil
}

// readInputWithContext reads user input with context cancellation support
func readInputWithContext(ctx context.Context, defaultValue string) string {
	// For now, we don't implement actual cancellation in user input
	// but having the context parameter allows for future enhancement
	// where we could use a separate goroutine to handle cancellation
	return readInput(defaultValue)
}

// Helper functions for user interaction

// readInput reads user input with a default value
func readInput(defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

// askYesNo asks a yes/no question with a default and context support
func askYesNo(ctx context.Context, question string, defaultYes bool) bool {
	// Check for cancellation
	if err := ctx.Err(); err != nil {
		// If cancelled, return the default
		return defaultYes
	}

	if initQuickMode {
		return defaultYes
	}

	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)
	input := readInputWithContext(ctx, "")

	switch strings.ToLower(input) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultYes
	}
}