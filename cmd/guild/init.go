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

	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/setup"
)

var (
	initQuickMode    bool
	initForce        bool
	initProviderOnly string
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
}

func runUnifiedInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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

	// Welcome message
	if !initQuickMode {
		fmt.Println("🏰 Welcome to Guild Framework!")
		fmt.Println("Let's set up your first campaign to get started.")
		fmt.Println()
	}

	// Step 1: Campaign Setup
	campaignName, projectName, err := setupCampaign(absPath)
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

		if err := project.InitializeProject(absPath); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project structure").
				WithComponent("cli").WithOperation("runUnifiedInit").WithDetails("path", absPath)
		}

		if !initQuickMode {
			fmt.Println("✅")
		}
	}

	// Step 4: Create global campaign config (but not local reference yet)
	if !initQuickMode {
		fmt.Print("🎯 Creating global campaign... ")
	}

	if err := createGlobalCampaignConfig(absPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create global campaign config").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	if !initQuickMode {
		fmt.Println("✅")
	}

	// Step 5: Run provider setup wizard (but save to project.yaml to preserve campaign reference)
	if !initQuickMode {
		fmt.Println("⚙️  Setting up AI providers and agents...")
	}

	if err := wizard.Run(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run setup wizard").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 5.1: Move guild config from guild.yaml to project.yaml to preserve campaign reference
	if err := moveGuildConfigToProject(absPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to move guild config").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 5.2: Now create the campaign reference and socket registry
	if err := createLocalCampaignReference(absPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create campaign reference").
			WithComponent("cli").WithOperation("runUnifiedInit")
	}

	// Step 6: Create demo commission (optional)
	if !initQuickMode {
		if askYesNo("🎯 Create a demo commission to get started?", true) {
			if err := createDemoCommission(absPath, campaignName); err != nil {
				fmt.Printf("⚠️  Warning: Could not create demo commission: %v\n", err)
			} else {
				fmt.Println("✅ Demo commission created")
			}
		}
	} else {
		// In quick mode, always create demo commission
		if err := createDemoCommission(absPath, campaignName); err != nil {
			// Don't fail the whole setup for demo commission
			fmt.Printf("⚠️  Warning: Could not create demo commission: %v\n", err)
		}
	}

	// Step 7: Success summary
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

	// Step 8: Next steps
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

// setupCampaign handles interactive campaign configuration
func setupCampaign(projectPath string) (campaignName, projectName string, err error) {
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
		campaignName = readInput(defaultCampaign)
	} else {
		campaignName = defaultCampaign
	}

	// Default project name
	defaultProject := filepath.Base(projectPath)
	if !initQuickMode {
		fmt.Printf("Project name [%s]: ", defaultProject)
		projectName = readInput(defaultProject)
	} else {
		projectName = defaultProject
	}

	return campaignName, projectName, nil
}

// createGlobalCampaignConfig creates the global campaign configuration
func createGlobalCampaignConfig(projectPath, campaignName, projectName string) error {
	// Create or update global campaign config
	var globalConfig *campaign.CampaignConfig

	// Check if campaign already exists
	existingConfig, err := campaign.LoadGlobalCampaignConfig(campaignName)
	if err != nil {
		// Campaign doesn't exist, create new one
		globalConfig = &campaign.CampaignConfig{
			Name:        campaignName,
			Description: fmt.Sprintf("Campaign %s", campaignName),
			Created:     time.Now().Format(time.RFC3339),
			Settings:    make(map[string]string),
		}
	} else {
		// Campaign exists, update it
		globalConfig = existingConfig
	}

	// Add project to campaign if not already present
	absPath, _ := filepath.Abs(projectPath)
	projectExists := false
	for _, proj := range globalConfig.Projects {
		if proj.Path == absPath {
			projectExists = true
			break
		}
	}

	if !projectExists {
		globalConfig.Projects = append(globalConfig.Projects, campaign.ProjectInfo{
			Name: projectName,
			Path: absPath,
		})
	}

	// Save global campaign config
	if err := campaign.SaveGlobalCampaignConfig(campaignName, globalConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save global campaign config").
			WithComponent("cli").WithOperation("createGlobalCampaignConfig")
	}

	return nil
}

// createLocalCampaignReference creates the local campaign reference and socket registry
func createLocalCampaignReference(projectPath, campaignName, projectName string) error {
	// Create local campaign reference
	if err := campaign.CreateCampaignReference(projectPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign reference").
			WithComponent("cli").WithOperation("createLocalCampaignReference")
	}

	// Create socket registry
	if err := daemon.SaveSocketRegistry(projectPath, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save socket registry").
			WithComponent("cli").WithOperation("createLocalCampaignReference")
	}

	return nil
}

// createDemoCommission creates a sample commission for new users
func createDemoCommission(projectPath, campaignName string) error {
	commissionsDir := filepath.Join(projectPath, ".guild", "objectives", "refined")
	if err := os.MkdirAll(commissionsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	demoCommission := `# Simple API Development Task

## Objective
Create a basic REST API with essential endpoints to demonstrate Guild's code generation and testing capabilities.

## Requirements

### Core API Features
- Create a simple Go HTTP server
- Implement basic CRUD operations for a "tasks" resource
- Add proper error handling and HTTP status codes
- Include basic logging

### Technical Specifications
- Use Go's standard library (net/http)
- Implement JSON request/response handling
- Add input validation
- Follow REST conventions

### Endpoints Required
1. GET /tasks - List all tasks
2. POST /tasks - Create a new task  
3. GET /tasks/{id} - Get specific task
4. PUT /tasks/{id} - Update task
5. DELETE /tasks/{id} - Delete task

### Testing Requirements
- Write unit tests for each endpoint
- Include integration tests
- Test error scenarios
- Achieve >80% test coverage

## Success Criteria
- All endpoints respond correctly
- Tests pass and have good coverage
- Code follows Go best practices
- API is well-documented

## Notes
This is a demo commission designed to showcase Guild's multi-agent development workflow. The Manager will break this down into smaller tasks and assign them to appropriate specialized agents.
`

	commissionPath := filepath.Join(commissionsDir, "demo-api-development.md")
	if err := os.WriteFile(commissionPath, []byte(demoCommission), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write demo commission").
			WithComponent("cli").WithOperation("createDemoCommission")
	}

	return nil
}

// moveGuildConfigToProject moves the guild config to project.yaml to preserve campaign reference
func moveGuildConfigToProject(projectPath string) error {
	guildYaml := filepath.Join(projectPath, ".guild", "guild.yaml") 
	projectYaml := filepath.Join(projectPath, ".guild", "project.yaml")
	
	// Read the guild config that was just created
	guildData, err := os.ReadFile(guildYaml)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read guild config").
			WithComponent("cli").WithOperation("moveGuildConfigToProject")
	}
	
	// Write it to project.yaml
	if err := os.WriteFile(projectYaml, guildData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write project config").
			WithComponent("cli").WithOperation("moveGuildConfigToProject")
	}
	
	// Remove the guild.yaml file so it can be replaced with campaign reference
	if err := os.Remove(guildYaml); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to remove guild config").
			WithComponent("cli").WithOperation("moveGuildConfigToProject")
	}
	
	return nil
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

// askYesNo asks a yes/no question with a default
func askYesNo(question string, defaultYes bool) bool {
	if initQuickMode {
		return defaultYes
	}

	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)
	input := readInput("")

	switch strings.ToLower(input) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultYes
	}
}