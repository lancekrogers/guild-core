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
	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/internal/daemon"
	uiinit "github.com/guild-ventures/guild-core/internal/ui/init"
	"github.com/guild-ventures/guild-core/pkg/agents"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

var (
	fastInitForce    bool
	fastInitNoDaemon bool
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
- Starts the daemon automatically

For an interactive setup experience with more control, use 'guild setup-wizard'.

Examples:
  guild init                    # Initialize current directory
  guild init ./my-project       # Initialize specific directory  
  guild init --no-daemon        # Initialize without starting daemon`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFastInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolVar(&fastInitForce, "force", false, "Force initialization even if already configured")
	initCmd.Flags().BoolVar(&fastInitNoDaemon, "no-daemon", false, "Don't auto-start the Guild server after initialization")
}

func runFastInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
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

	// Create dependencies
	deps := uiinit.InitDependencies{
		ConfigManager: uiinit.NewDefaultConfigManager(),
		ProjectInit:   uiinit.NewDefaultProjectInitializer(),
		DemoGen:       uiinit.NewDefaultDemoGenerator(),
		Validator:     uiinit.NewDefaultValidator(),
		DaemonManager: uiinit.NewDefaultDaemonManager(),
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

	// Step 1: Initialize project structure
	fmt.Print("🏗️  Creating project structure... ")
	if !deps.ProjectInit.IsProjectInitialized(projectPath) {
		if err := deps.ProjectInit.InitializeProject(ctx, projectPath); err != nil {
			fmt.Println("❌")
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project").
				WithComponent("cli").
				WithOperation("runFastInit")
		}
	}
	fmt.Println("✅")

	// Step 2: Auto-detect AI providers
	fmt.Print("🤖 Detecting AI providers... ")
	detector := providers.NewAutoDetector(10 * time.Second)
	providerResults, err := detector.DetectAll(ctx)
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

	// Step 3: Create configuration
	fmt.Print("⚙️  Creating configuration... ")
	if err := deps.ConfigManager.CreatePhase0Configuration(ctx, projectPath, campaignName, projectName); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create configuration").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")

	// Step 4: Create Elena and enhanced agents
	fmt.Print("👥 Creating Elena and specialist agents... ")
	agentCount, err := createEnhancedAgents(ctx, projectPath, providerResults)
	if err != nil {
		fmt.Printf("❌\n")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agents").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Printf("✅ (%d agents)\n", agentCount)

	// Step 5: Integration and validation
	fmt.Print("🔗 Integrating configuration... ")
	if err := deps.ConfigManager.IntegrateWithPhase0Config(ctx, projectPath, campaignName, projectName); err != nil {
		fmt.Println("❌")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to integrate configuration").
			WithComponent("cli").
			WithOperation("runFastInit")
	}
	fmt.Println("✅")

	// Step 6: Socket registry
	fmt.Print("🔧 Setting up daemon registry... ")
	if err := deps.DaemonManager.SaveSocketRegistry(projectPath, campaignName); err != nil {
		fmt.Printf("⚠️ (manual start required)\n")
	} else {
		fmt.Println("✅")
	}

	// Step 7: Auto-start daemon unless --no-daemon flag is set
	if !fastInitNoDaemon {
		fmt.Print("🚀 Starting Guild daemon")
		
		// Show progress during daemon startup
		done := make(chan struct{})
		go func() {
			for i := 0; i < 10; i++ {
				select {
				case <-done:
					return
				case <-time.After(300 * time.Millisecond):
					fmt.Print(".")
				}
			}
		}()
		
		// Use the lifecycle manager for auto-start with timeout context
		daemonCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		defer cancel()
		
		lifecycleManager := daemon.DefaultLifecycleManager
		_, err := lifecycleManager.AutoStartDaemon(daemonCtx, campaignName)
		close(done) // Stop progress dots
		
		if err != nil {
			fmt.Printf(" ⚠️\n")
			fmt.Printf("   Failed to start daemon: %v\n", err)
			fmt.Printf("   💡 Manual start: guild serve --campaign %s --daemon\n", campaignName)
			fmt.Printf("   🔍 Check logs: tail ~/.guild/logs/%s.log\n", campaignName)
			fmt.Printf("   🛠️  Debug mode: guild serve --campaign %s --debug\n", campaignName)
		} else {
			// Verify daemon is fully ready
			fmt.Print(" ✅")
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
			fmt.Println(" (ready)")
		}
	}

	fmt.Println()
	fmt.Println("🏰 Guild successfully initialized!")
	fmt.Println()
	fmt.Println("👑 Elena the Guild Master is ready to lead your team")
	fmt.Println("⚔️  Marcus the Code Artisan stands ready to craft solutions")
	fmt.Println("🛡️  Vera the Quality Guardian protects your software excellence")
	fmt.Println()
	fmt.Println("🚀 Start your adventure:")
	if !fastInitNoDaemon {
		fmt.Println("   guild chat                           # Meet Elena and begin")
		fmt.Println("   guild chat --agent elena-guild-master  # Talk to Elena directly")
	} else {
		fmt.Println("   guild serve --campaign guild-demo --daemon  # Start the daemon first")
		fmt.Println("   guild chat                           # Then meet Elena and begin")
	}
	fmt.Println("   guild status                         # Check guild status")
	fmt.Println()
	fmt.Println("💡 For more control over setup, use: guild setup-wizard")
	fmt.Println()

	return nil
}

// createEnhancedAgents creates enhanced agents with optimized provider selection
func createEnhancedAgents(ctx context.Context, projectPath string, providerResults []providers.DetectionResult) (int, error) {
	// Create enhanced agent creator
	creator := agents.NewDefaultAgentCreator()

	// Create default agent set
	agentConfigs, err := creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent set").
			WithComponent("cli").
			WithOperation("createEnhancedAgents")
	}

	// Optimize providers based on detection results
	optimizeAgentProviders(agentConfigs, providerResults)

	// Ensure agents directory exists
	agentsDir := filepath.Join(projectPath, ".campaign", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("cli").
			WithOperation("createEnhancedAgents")
	}

	// Save each agent configuration
	for _, agentConfig := range agentConfigs {
		if err := saveAgentConfig(ctx, agentsDir, agentConfig); err != nil {
			return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save agent config").
				WithComponent("cli").
				WithOperation("createEnhancedAgents").
				WithDetails("agent", agentConfig.ID)
		}
	}

	return len(agentConfigs), nil
}

// optimizeAgentProviders intelligently assigns providers to agents based on availability
func optimizeAgentProviders(agentConfigs []*config.AgentConfig, providerResults []providers.DetectionResult) {
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

	// Apply intelligent provider mapping
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

// saveAgentConfig saves agent configuration to YAML file
func saveAgentConfig(ctx context.Context, agentsDir string, agentConfig *config.AgentConfig) error {
	// Convert to YAML-friendly format
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
			WithComponent("cli").
			WithOperation("saveAgentConfig")
	}

	// Save to file
	filename := fmt.Sprintf("%s.yaml", agentConfig.ID)
	filepath := filepath.Join(agentsDir, filename)

	if err := os.WriteFile(filepath, yamlData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config file").
			WithComponent("cli").
			WithOperation("saveAgentConfig").
			WithDetails("file", filepath)
	}

	return nil
}