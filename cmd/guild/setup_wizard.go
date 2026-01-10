// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild-core/internal/daemon"
	uiinit "github.com/lancekrogers/guild-core/internal/ui/init"
	"github.com/lancekrogers/guild-core/pkg/agents/creation"
	"github.com/lancekrogers/guild-core/pkg/campaign"
	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/providers"
)

var (
	initQuickMode      bool
	initForce          bool
	initProviderOnly   string
	initSkipValidation bool
	initNoDaemon       bool
)

// setupWizardCmd represents the setup-wizard command
var setupWizardCmd = &cobra.Command{
	Use:   "setup-wizard [path]",
	Short: "Initialize Guild project with interactive setup wizard",
	Long: `Initialize a complete Guild project with an interactive TUI wizard.

This command provides a comprehensive setup experience with full control
over campaign configuration, provider selection, and agent customization.

The interactive wizard includes:
- Campaign architecture and configuration
- AI provider detection and selection
- Agent configuration with customizable options
- Optional demo commission creation
- Validation of all settings

For a faster, non-interactive setup experience, use 'guild init'.

Examples:
  guild setup-wizard                    # Start interactive wizard for current directory
  guild setup-wizard ./my-project       # Start wizard for specific directory  
  guild setup-wizard --provider ollama  # Setup only Ollama provider
  guild setup-wizard --no-daemon        # Initialize without starting daemon`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUnifiedInit,
}

func init() {
	rootCmd.AddCommand(setupWizardCmd)

	// Add flags
	setupWizardCmd.Flags().BoolVar(&initQuickMode, "quick", false, "Quick setup with automatic defaults")
	setupWizardCmd.Flags().BoolVar(&initForce, "force", false, "Force setup even if already configured")
	setupWizardCmd.Flags().StringVar(&initProviderOnly, "provider", "", "Setup only this provider (openai, anthropic, ollama, claude_code)")
	setupWizardCmd.Flags().BoolVar(&initSkipValidation, "skip-validation", false, "Skip post-init validation")
	setupWizardCmd.Flags().BoolVar(&initNoDaemon, "no-daemon", false, "Don't auto-start the Guild server after initialization")
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
		ProjectPath:    projectPath,
		QuickMode:      initQuickMode,
		Force:          initForce,
		ProviderOnly:   initProviderOnly,
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
	var ttyFile *os.File
	if file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		ttyFile = file
		ttyAvailable = true
		defer ttyFile.Close()
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
		opts = append(opts, tea.WithInput(ttyFile))
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

			// Perform direct initialization with all enhancements
			if err := runDirectInitialization(ctx, config, deps); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "direct initialization failed").
					WithComponent("cli").
					WithOperation("runUnifiedInit")
			}

			fmt.Println("✅ Guild initialized successfully with Elena and enhanced agents!")
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
		// In quick mode, ensure files were created
		if initQuickMode {
			// Check if .campaign directory exists
			campaignPath := filepath.Join(config.ProjectPath, ".campaign")

			if _, err := os.Stat(campaignPath); os.IsNotExist(err) {
				// Campaign directory doesn't exist, initialization failed silently
				fmt.Println("⚠️  Warning: Initialization completed but no directories were created")
				fmt.Println("Running direct initialization as fallback...")

				// Run direct initialization
				if err := runDirectInitialization(ctx, config, deps); err != nil {
					return gerror.Wrap(err, gerror.ErrCodeInternal, "fallback initialization failed").
						WithComponent("cli").
						WithOperation("runUnifiedInit")
				}
			}
		}
	}

	// Auto-start daemon unless --no-daemon flag is set
	if !initNoDaemon {
		fmt.Println("🚀 Starting Guild daemon...")

		// Detect the campaign we just created
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not start daemon - failed to get working directory: %v\n", err)
		} else {
			campaignName, err := campaign.DetectCampaign(cwd, "")
			if err != nil {
				fmt.Printf("⚠️  Warning: Could not start daemon - failed to detect campaign: %v\n", err)
			} else {
				// Use the lifecycle manager for auto-start
				lifecycleManager := daemon.DefaultLifecycleManager
				_, err := lifecycleManager.AutoStartDaemon(ctx, campaignName)
				if err != nil {
					fmt.Printf("⚠️  Warning: Failed to start daemon: %v\n", err)
					fmt.Printf("💡 You can start it manually with: guild serve --campaign %s --daemon\n", campaignName)
				} else {
					// Give the server a moment to fully initialize
					time.Sleep(500 * time.Millisecond)
					fmt.Printf("✅ Guild daemon started successfully for campaign '%s'\n", campaignName)
					fmt.Printf("💬 You can now run: guild chat\n")
				}
			}
		}
	}

	// In quick mode, print minimal summary
	if initQuickMode {
		fmt.Println("✅ Guild initialized successfully.")
	}

	return nil
}

// runDirectInitialization performs enhanced initialization without TUI for non-TTY environments
func runDirectInitialization(ctx context.Context, config uiinit.Config, deps uiinit.InitDependencies) error {
	// Use default campaign and project names
	campaignName := "guild-demo"
	projectName := filepath.Base(config.ProjectPath)
	if projectName == "." {
		projectName = "my-project"
	}

	fmt.Printf("📋 Campaign: %s\n", campaignName)
	fmt.Printf("📁 Project: %s\n", projectName)
	fmt.Printf("📍 Location: %s\n", config.ProjectPath)
	fmt.Println()

	// Step 1: Initialize project structure
	fmt.Print("🏗️  Initializing project structure... ")
	if !deps.ProjectInit.IsProjectInitialized(config.ProjectPath) {
		if err := deps.ProjectInit.InitializeProject(ctx, config.ProjectPath); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project").
				WithComponent("directInit").
				WithOperation("runDirectInitialization")
		}
	}
	fmt.Println("✅")

	// Step 2: Detect AI providers
	fmt.Print("🤖 Detecting AI providers... ")
	detector := providers.NewAutoDetector(10 * time.Second)
	providerResults, err := detector.DetectAll(ctx)
	if err != nil {
		fmt.Printf("⚠️ (continuing with defaults)\n")
		providerResults = []providers.DetectionResult{} // Empty results, will use defaults
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

	// Step 3: Create configuration
	fmt.Print("⚙️  Creating configuration... ")
	if err := deps.ConfigManager.EstablishGuildFoundation(ctx, config.ProjectPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create configuration").
			WithComponent("directInit").
			WithOperation("runDirectInitialization")
	}
	fmt.Println("✅")

	// Step 4: Create Elena and enhanced agents
	fmt.Print("👥 Creating Elena and specialist agents... ")
	agentCount, err := createDirectEnhancedAgents(ctx, config.ProjectPath, providerResults)
	if err != nil {
		fmt.Printf("⚠️ (using basic agents: %v)\n", err)
		agentCount = 3 // Default count
	} else {
		fmt.Printf("✅ (%d agents)\n", agentCount)
	}

	// Step 5: Integration and validation
	fmt.Print("🔗 Integrating configuration... ")
	if err := deps.ConfigManager.FinalizeGuildCharter(ctx, config.ProjectPath, campaignName, projectName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to integrate configuration").
			WithComponent("directInit").
			WithOperation("runDirectInitialization")
	}
	fmt.Println("✅")

	// Step 6: Socket registry
	fmt.Print("🔧 Setting up daemon registry... ")
	if err := deps.DaemonManager.SaveSocketRegistry(config.ProjectPath, campaignName); err != nil {
		fmt.Printf("⚠️ (manual start required)\n")
	} else {
		fmt.Println("✅")
	}

	fmt.Println()
	fmt.Println("🏰 Guild successfully established!")
	fmt.Printf("👑 Elena the Guild Master is ready to lead your team\n")
	fmt.Printf("⚔️  Marcus the Code Artisan stands ready to craft solutions\n")
	fmt.Printf("🛡️  Vera the Quality Guardian protects your software excellence\n")
	fmt.Println()
	fmt.Println("🚀 Start your adventure:")
	fmt.Printf("   guild chat                # Meet Elena and begin\n")
	fmt.Printf("   guild chat --agent elena-guild-master  # Talk to Elena directly\n")
	fmt.Printf("   guild status              # Check guild status\n")
	fmt.Println()

	return nil
}

// createDirectEnhancedAgents creates enhanced agents for direct initialization
func createDirectEnhancedAgents(ctx context.Context, projectPath string, providerResults []providers.DetectionResult) (int, error) {
	// Create enhanced agent creator
	creator := creation.NewDefaultAgentCreator()

	// Create default agent set
	agentConfigs, err := creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent set").
			WithComponent("directInit").
			WithOperation("createDirectEnhancedAgents")
	}

	// Optimize providers based on detection results
	optimizeAgentProvidersForDirect(agentConfigs, providerResults)

	// Ensure agents directory exists
	agentsDir := filepath.Join(projectPath, ".campaign", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("directInit").
			WithOperation("createDirectEnhancedAgents")
	}

	// Save each agent configuration
	for _, agentConfig := range agentConfigs {
		if err := saveDirectAgentConfig(ctx, agentsDir, agentConfig); err != nil {
			return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save agent config").
				WithComponent("directInit").
				WithOperation("createDirectEnhancedAgents")
		}
	}

	return len(agentConfigs), nil
}

// optimizeAgentProvidersForDirect optimizes agent providers for direct init
func optimizeAgentProvidersForDirect(agentConfigs []*config.AgentConfig, providerResults []providers.DetectionResult) {
	// Create provider availability map
	availableProviders := make(map[string]providers.DetectionResult)
	var bestProvider *providers.DetectionResult
	highestConfidence := 0.0

	for _, result := range providerResults {
		if result.Available {
			availableProviders[string(result.Provider)] = result
			if result.Confidence > highestConfidence {
				highestConfidence = result.Confidence
				bestProvider = &result
			}
		}
	}

	// If no providers available, use defaults
	if len(availableProviders) == 0 {
		return
	}

	// Apply the same intelligent mapping logic
	for _, agent := range agentConfigs {
		originalProvider := agent.Provider

		switch agent.Type {
		case "manager":
			// Elena benefits from Claude Code's planning capabilities
			if claudeCode, exists := availableProviders["claude_code"]; exists && claudeCode.Confidence > 0.7 {
				agent.Provider = "claude_code"
			} else if _, exists := availableProviders["anthropic"]; exists {
				agent.Provider = "anthropic"
			}
		case "worker":
			// Marcus benefits from Claude Code's coding features
			if agent.ID == "marcus-developer" {
				if claudeCode, exists := availableProviders["claude_code"]; exists && claudeCode.Confidence > 0.7 {
					agent.Provider = "claude_code"
				} else if _, exists := availableProviders["anthropic"]; exists {
					agent.Provider = "anthropic"
				}
			}
		case "specialist":
			// Vera can use any high-quality provider
			if _, exists := availableProviders["anthropic"]; exists {
				agent.Provider = "anthropic"
			} else if _, exists := availableProviders["claude_code"]; exists {
				agent.Provider = "claude_code"
			}
		}

		// Fallback to best available provider if chosen provider unavailable
		if _, exists := availableProviders[agent.Provider]; !exists {
			if bestProvider != nil {
				agent.Provider = string(bestProvider.Provider)
			} else {
				agent.Provider = originalProvider
			}
		}
	}
}

// saveDirectAgentConfig saves agent config for direct initialization
func saveDirectAgentConfig(ctx context.Context, agentsDir string, agentConfig *config.AgentConfig) error {
	// Convert to YAML-friendly format (reuse the same logic from TUI version)
	configData := map[string]interface{}{
		"id":           agentConfig.ID,
		"name":         agentConfig.Name,
		"type":         agentConfig.Type,
		"description":  agentConfig.Description,
		"provider":     agentConfig.Provider,
		"model":        agentConfig.Model,
		"capabilities": agentConfig.Capabilities,
		"tools":        agentConfig.Tools,
	}

	// Add backstory information if available
	if agentConfig.Backstory != nil {
		configData["backstory"] = map[string]interface{}{
			"experience":     agentConfig.Backstory.Experience,
			"previous_roles": agentConfig.Backstory.PreviousRoles,
			"expertise":      agentConfig.Backstory.Expertise,
			"achievements":   agentConfig.Backstory.Achievements,
			"philosophy":     agentConfig.Backstory.Philosophy,
			"guild_rank":     agentConfig.Backstory.GuildRank,
			"specialties":    agentConfig.Backstory.Specialties,
		}
	}

	// Add personality information if available
	if agentConfig.Personality != nil {
		configData["personality"] = map[string]interface{}{
			"formality":      agentConfig.Personality.Formality,
			"detail_level":   agentConfig.Personality.DetailLevel,
			"humor_level":    agentConfig.Personality.HumorLevel,
			"approach_style": agentConfig.Personality.ApproachStyle,
			"assertiveness":  agentConfig.Personality.Assertiveness,
			"empathy":        agentConfig.Personality.Empathy,
			"patience":       agentConfig.Personality.Patience,
			"honor":          agentConfig.Personality.Honor,
			"wisdom":         agentConfig.Personality.Wisdom,
			"craftsmanship":  agentConfig.Personality.Craftsmanship,
		}
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithComponent("directInit").
			WithOperation("saveDirectAgentConfig")
	}

	// Save to file
	filename := fmt.Sprintf("%s.yaml", agentConfig.ID)
	filepath := filepath.Join(agentsDir, filename)

	if err := os.WriteFile(filepath, yamlData, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config file").
			WithComponent("directInit").
			WithOperation("saveDirectAgentConfig")
	}

	return nil
}

// All helper functions have been moved to internal/ui/init/init_tui.go
