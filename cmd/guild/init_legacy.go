// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/project"
)

var initLegacyCmd = &cobra.Command{
	Use:   "init-legacy [path]",
	Short: "Initialize a Guild project (legacy)",
	Long: `Creates both global (~/.guild) and local (.guild) directory structures.

Global directory (~/.guild) contains:
- Provider configurations
- Tool installations  
- LSP servers
- Project templates
- Shared cache

Local directory (.guild) contains:
- Project configuration (guild.yaml)
- SQLite database (memory.db)
- Corpus and RAG vector stores  
- Commissions (user objectives/goals)
- Project-specific tools
- Agent workspaces
- Task tracking (Kanban)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initLegacyCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve path").
			WithComponent("cli").WithOperation("runInit").WithDetails("path", path)
	}

	// Check if already initialized
	if project.IsProjectInitialized(path) {
		fmt.Fprintf(os.Stderr, "Error: Project already initialized at %s\n", absPath)
		fmt.Fprintln(os.Stderr, "The .guild directory already exists.")
		return nil
	}

	fmt.Printf("🏰 Initializing Guild Framework at %s...\n", absPath)

	// Step 1: Detect project type
	fmt.Print("📜 Analyzing project structure... ")
	detector := project.NewProjectDetector()
	projectType, err := detector.DetectProjectType(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect project type").
			WithComponent("cli").WithOperation("runInit").WithDetails("path", path)
	}
	fmt.Printf("✅ Detected: %s\n", projectType.Description)

	// Step 2: Generate intelligent configuration
	fmt.Print("🎯 Generating Guild configuration... ")
	guildConfig, err := detector.GenerateGuildConfig(projectType, path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate guild config").
			WithComponent("cli").WithOperation("runInit").WithDetails("project_type", projectType.Description)
	}

	corpusConfig := detector.GenerateCorpusConfig(projectType, path)
	fmt.Println("✅")

	// Step 3: Create directory structure (both global and local)
	fmt.Print("📁 Creating directory structure... ")
	if err := project.InitializeProject(path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project structure").
			WithComponent("cli").WithOperation("runInit").WithDetails("path", path)
	}
	fmt.Println("✅")

	// Step 4: Write intelligent configuration files
	fmt.Print("⚙️  Writing configuration files... ")

	// Write guild.yaml
	guildConfigPath := filepath.Join(path, ".guild", "guild.yaml")
	guildConfigData, err := yaml.Marshal(guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("cli").WithOperation("runInit")
	}
	if err := os.WriteFile(guildConfigPath, guildConfigData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("cli").WithOperation("runInit").
			WithDetails("config_path", guildConfigPath)
	}

	// Write corpus.yaml
	corpusConfigPath := filepath.Join(path, ".guild", "corpus.yaml")
	corpusConfigData, err := yaml.Marshal(corpusConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal corpus config").
			WithComponent("cli").WithOperation("runInit")
	}
	if err := os.WriteFile(corpusConfigPath, corpusConfigData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write corpus config").
			WithComponent("cli").WithOperation("runInit").
			WithDetails("config_path", corpusConfigPath)
	}
	fmt.Println("✅")

	// Step 5: Scan for documentation to seed corpus
	fmt.Print("📚 Scanning for documentation... ")
	docFiles, err := detector.SeedCorpusFromProject(projectType, path)
	if err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d documents\n", len(docFiles))

		if len(docFiles) > 0 {
			fmt.Println("\n📋 Suggested files for corpus:")
			for i, file := range docFiles {
				if i >= 5 { // Limit to first 5 suggestions
					fmt.Printf("   ... and %d more\n", len(docFiles)-5)
					break
				}
				fmt.Printf("   • %s\n", file)
			}
		}
	}

	// Step 6: Check provider configuration
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
	fmt.Printf("\n🎉 Successfully initialized Guild project!\n")
	fmt.Printf("   Project type: %s\n", projectType.Description)
	fmt.Printf("   Location: %s\n", absPath)
	fmt.Printf("   Global config: ~/.guild/\n")
	fmt.Printf("   Agents configured: %d\n", len(guildConfig.Agents))

	// Display next steps
	fmt.Println("\n🚀 Next steps:")

	if len(availableProviders) == 0 {
		fmt.Println("   1. Set up your API keys:")
		fmt.Println("      export ANTHROPIC_API_KEY=\"your-anthropic-api-key\"")
		fmt.Println("      export OPENAI_API_KEY=\"your-openai-api-key\"")
	}

	fmt.Println("   1. Start chatting with AI agents:")
	fmt.Println("      guild chat")

	if len(docFiles) > 0 {
		fmt.Println("   2. Index your project documentation:")
		fmt.Println("      guild corpus scan")
	}

	fmt.Println("   3. View available agents:")
	fmt.Println("      guild agent list")
	fmt.Println("   4. Check your configuration:")
	fmt.Println("      guild config show")
	fmt.Println("   5. See all available commands:")
	fmt.Println("      guild --help")

	fmt.Println("\n📚 Coming Soon:")
	fmt.Printf("   • guild commission \"Implement %s feature\" - Create AI-powered work items\n", getExampleFeature(projectType))
	fmt.Println("   • guild kanban view - Interactive task board")
	fmt.Println("   • guild campaign watch - Monitor multi-agent workflows")

	return nil
}

// getExampleFeature returns an appropriate example feature for the project type
func getExampleFeature(projectType *project.ProjectType) string {
	switch projectType.Language {
	case "go":
		return "user authentication API"
	case "javascript":
		return "user dashboard"
	case "python":
		return "data processing pipeline"
	case "rust":
		return "concurrent task processor"
	default:
		return "core functionality"
	}
}
