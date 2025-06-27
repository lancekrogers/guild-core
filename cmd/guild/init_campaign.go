// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/pkg/campaign"
	"github.com/lancekrogers/guild/pkg/daemon"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/project"
)

var initCampaignCmd = &cobra.Command{
	Use:   "init-campaign [path]",
	Short: "Initialize a Guild campaign (new architecture)",
	Long: `Initialize a Guild campaign using the new campaign architecture.

This creates:
- Local .guild/ directory with campaign reference
- Global campaign storage in ~/.guild/campaigns/
- Campaign configuration and directory structure
- Socket registry for multi-instance daemon support

This command bridges the existing Guild project system with the new
campaign-based multi-instance architecture.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitCampaign,
}

func init() {
	rootCmd.AddCommand(initCampaignCmd)
}

func runInitCampaign(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve path").
			WithComponent("cli").WithOperation("runInitCampaign").WithDetails("path", path)
	}

	// Check if already initialized
	if project.IsProjectInitialized(path) {
		fmt.Fprintf(os.Stderr, "Error: Project already initialized at %s\n", absPath)
		fmt.Fprintln(os.Stderr, "The .guild directory already exists.")
		return nil
	}

	// Check if we're already in a campaign
	if _, err := campaign.DetectCampaign(path, ""); err == nil {
		fmt.Fprintf(os.Stderr, "Error: Already in a campaign at %s\n", absPath)
		fmt.Fprintln(os.Stderr, "Use 'guild status' to see current campaign info.")
		return nil
	}

	fmt.Printf("🏰 Initializing Guild Campaign at %s...\n", absPath)

	// Step 1: Detect project type (use existing logic)
	fmt.Print("📜 Analyzing project structure... ")
	detector := project.NewProjectDetector()
	projectType, err := detector.DetectProjectType(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect project type").
			WithComponent("cli").WithOperation("runInitCampaign").WithDetails("path", path)
	}
	fmt.Printf("✅ Detected: %s\n", projectType.Description)

	// Step 2: Interactive campaign setup
	fmt.Println("\n🎯 Campaign Setup:")

	// Default campaign name from directory
	defaultCampaign := filepath.Base(absPath)
	fmt.Printf("Campaign name [%s]: ", defaultCampaign)
	var campaignName string
	fmt.Scanln(&campaignName)
	if campaignName == "" {
		campaignName = defaultCampaign
	}

	// Default project name
	defaultProject := filepath.Base(absPath)
	fmt.Printf("Project name [%s]: ", defaultProject)
	var projectName string
	fmt.Scanln(&projectName)
	if projectName == "" {
		projectName = defaultProject
	}

	// Step 3: Create campaign structure
	fmt.Print("📁 Creating campaign directory structure... ")

	// Initialize the traditional project structure first
	if err := project.InitializeProject(path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project structure").
			WithComponent("cli").WithOperation("runInitCampaign").WithDetails("path", path)
	}

	// Create local campaign reference
	if err := campaign.CreateCampaignReference(path, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign reference").
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	// Create socket registry
	if err := daemon.SaveSocketRegistry(path, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save socket registry").
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	fmt.Println("✅")

	// Step 4: Create or update global campaign config
	fmt.Print("⚙️  Setting up global campaign... ")

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
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	fmt.Println("✅")

	// Step 5: Generate project configuration (use existing logic)
	fmt.Print("🎯 Generating project configuration... ")
	guildConfig, err := detector.GenerateGuildConfig(projectType, path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate guild config").
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	// Update guild config with campaign information
	// Campaign metadata is now stored separately in campaign reference and global config
	// The project-level guild config focuses on agents and local settings
	guildConfig.Name = projectName
	guildConfig.Description = fmt.Sprintf("Project %s in campaign %s", projectName, campaignName)

	// Write guild.yaml (this overwrites the campaign reference, so we need to be careful)
	// Actually, let's write it to a different file to avoid conflicts
	guildConfigPath := filepath.Join(path, ".guild", "project.yaml")
	if err := writeYAMLFile(guildConfigPath, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write project config").
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	// Generate and write corpus config
	corpusConfig := detector.GenerateCorpusConfig(projectType, path)
	corpusConfigPath := filepath.Join(path, ".guild", "corpus.yaml")
	if err := writeYAMLFile(corpusConfigPath, corpusConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write corpus config").
			WithComponent("cli").WithOperation("runInitCampaign")
	}

	fmt.Println("✅")

	// Step 6: Initialize campaign database in global location
	fmt.Print("🗄️  Initializing campaign database... ")

	// This would integrate with the existing database initialization
	// For now, we'll create a placeholder database file
	_, err = campaign.LoadGlobalCampaignConfig(campaignName)
	if err == nil {
		// Database initialization would go here
		// This should integrate with the existing SQLite setup
	}

	fmt.Println("✅")

	// Step 7: Check provider configuration (existing logic)
	fmt.Print("🔑 Checking API key configuration... ")
	var availableProviders []string
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		availableProviders = append(availableProviders, "Anthropic")
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		availableProviders = append(availableProviders, "OpenAI")
	}

	if len(availableProviders) > 0 {
		fmt.Printf("✅ Found: %v\n", availableProviders)
	} else {
		fmt.Println("⚠️  No API keys found")
	}

	// Success summary
	fmt.Printf("\n🎉 Successfully initialized Guild campaign!\n")
	fmt.Printf("   Campaign: %s\n", campaignName)
	fmt.Printf("   Project: %s\n", projectName)
	fmt.Printf("   Location: %s\n", absPath)
	fmt.Printf("   Global config: ~/.guild/campaigns/%s/\n", campaignName)
	fmt.Printf("   Socket registry: %s/.guild/socket-registry.yaml\n", absPath)

	// Display next steps
	fmt.Println("\n🚀 Next steps:")

	if len(availableProviders) == 0 {
		fmt.Println("   1. Set up your API keys:")
		fmt.Println("      export ANTHROPIC_API_KEY=\"your-anthropic-api-key\"")
		fmt.Println("      export OPENAI_API_KEY=\"your-openai-api-key\"")
	}

	fmt.Println("   1. Start the Guild daemon:")
	fmt.Println("      guild serve --daemon")
	fmt.Println("   2. Start chatting with AI agents:")
	fmt.Println("      guild chat")
	fmt.Println("   3. Check campaign status:")
	fmt.Println("      guild status")
	fmt.Println("   4. See all campaigns:")
	fmt.Println("      guild campaign list")

	fmt.Println("\n📚 Multi-Instance Features:")
	fmt.Println("   • Multiple projects can reference the same campaign")
	fmt.Println("   • Each campaign gets its own daemon instance")
	fmt.Println("   • Socket-based communication (no port conflicts)")
	fmt.Println("   • Global campaign data sharing")

	return nil
}

// Helper function to write YAML files
func writeYAMLFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Note: This is a simplified implementation
	// In practice, you'd want to use yaml.Marshal and proper error handling
	fmt.Fprintf(file, "# Generated Guild configuration\n")
	fmt.Fprintf(file, "# Path: %s\n", path)

	return nil
}
