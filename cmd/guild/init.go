package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/internal/project"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a Guild project",
	Long: `Creates a .guild directory structure in the current or specified path.

This initializes a project-local Guild environment with:
- Corpus for project documentation
- Embeddings for semantic search
- Agent configurations
- Objective tracking
- Project-specific configuration`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, "failed to resolve path",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("path", path),
		)
	}

	// Check if already initialized
	if project.IsInitialized(path) {
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
		return gerror.Wrap(err, "failed to detect project type",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("path", path),
		)
	}
	fmt.Printf("✅ Detected: %s\n", projectType.Description)

	// Step 2: Generate intelligent configuration
	fmt.Print("🎯 Generating Guild configuration... ")
	guildConfig, err := detector.GenerateGuildConfig(projectType, path)
	if err != nil {
		return gerror.Wrap(err, "failed to generate guild config",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("project_type", projectType.Description),
		)
	}

	corpusConfig := detector.GenerateCorpusConfig(projectType, path)
	fmt.Println("✅")

	// Step 3: Create directory structure
	fmt.Print("📁 Creating directory structure... ")
	if err := project.Initialize(path); err != nil {
		return gerror.Wrap(err, "failed to initialize project structure",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("path", path),
		)
	}
	fmt.Println("✅")

	// Step 4: Write intelligent configuration files
	fmt.Print("⚙️  Writing configuration files... ")
	
	// Write guild.yaml
	guildConfigPath := filepath.Join(path, ".guild", "guild.yaml")
	guildConfigData, err := yaml.Marshal(guildConfig)
	if err != nil {
		return gerror.Wrap(err, "failed to marshal guild config",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
		)
	}
	if err := os.WriteFile(guildConfigPath, guildConfigData, 0644); err != nil {
		return gerror.Wrap(err, "failed to write guild config",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("config_path", guildConfigPath),
		)
	}

	// Write corpus.yaml
	corpusConfigPath := filepath.Join(path, ".guild", "corpus.yaml")
	corpusConfigData, err := yaml.Marshal(corpusConfig)
	if err != nil {
		return gerror.Wrap(err, "failed to marshal corpus config",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
		)
	}
	if err := os.WriteFile(corpusConfigPath, corpusConfigData, 0644); err != nil {
		return gerror.Wrap(err, "failed to write corpus config",
			gerror.WithComponent("cli"),
			gerror.WithOperation("runInit"),
			gerror.WithDetails("config_path", corpusConfigPath),
		)
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
	fmt.Printf("   Agents configured: %d\n", len(guildConfig.Agents))
	
	// Display next steps
	fmt.Println("\n🚀 Next steps:")
	
	if len(availableProviders) == 0 {
		fmt.Println("   1. Set up your API keys:")
		fmt.Println("      export ANTHROPIC_API_KEY=\"your-anthropic-api-key\"")
		fmt.Println("      export OPENAI_API_KEY=\"your-openai-api-key\"")
	}
	
	fmt.Println("   1. Start coding with agents:")
	fmt.Println("      guild chat")
	fmt.Println("   2. Create your first commission:")
	fmt.Printf("      guild commission \"Implement %s feature\"\n", getExampleFeature(projectType))
	fmt.Println("   3. Monitor agent progress:")
	fmt.Println("      guild campaign watch")
	
	if len(docFiles) > 0 {
		fmt.Println("   4. Add documentation to corpus:")
		fmt.Println("      guild corpus add README.md")
	}

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