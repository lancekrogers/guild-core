// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
)

var initRefactoredCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a Guild project",
	Long: `Creates both global (~/.guild) and local (.guild) directory structures.

Global (~/.guild):
- Provider configurations
- Tool installations
- LSP servers
- Project templates
- Shared cache

Local (.guild):
- Project configuration (guild.yaml)
- SQLite database (memory.db)
- Corpus and RAG vector stores
- Commissions and objectives
- Agent workspaces`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitRefactored,
}

func runInitRefactored(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to resolve path").
			WithComponent("cli").WithOperation("init").WithDetails("path", path)
	}

	// Check if already initialized
	if project.IsProjectInitialized(path) {
		fmt.Fprintf(os.Stderr, "Error: Project already initialized at %s\n", absPath)
		fmt.Fprintln(os.Stderr, "The .guild directory already exists.")
		return gerror.New(gerror.ErrCodeAlreadyExists, "project already initialized", nil).
			WithComponent("cli").WithOperation("init")
	}

	fmt.Printf("Initializing Guild project at %s...\n", absPath)

	// Initialize both global and local structures
	if err := project.InitializeProject(path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize project").
			WithComponent("cli").WithOperation("init")
	}

	// Get project structure for display
	structure := project.GetProjectStructure(path)

	// Display what was created
	fmt.Println("\n✅ Global Guild directory initialized:")
	fmt.Printf("   📁 %s\n", structure.GlobalDir)
	fmt.Printf("   ├── 📄 config.yaml (global settings)\n")
	fmt.Printf("   ├── 📁 providers/ (API configurations)\n")
	fmt.Printf("   ├── 📁 tools/ (global tools)\n")
	fmt.Printf("   ├── 📁 templates/ (project templates)\n")
	fmt.Printf("   ├── 📁 lsp/ (language servers)\n")
	fmt.Printf("   ├── 📁 cache/ (shared embeddings)\n")
	fmt.Printf("   └── 📁 logs/ (guild logs)\n")

	fmt.Println("\n✅ Local Guild directory initialized:")
	fmt.Printf("   📁 %s\n", structure.LocalDir)
	fmt.Printf("   ├── 📄 guild.yaml (project config)\n")
	fmt.Printf("   ├── 🗄️  memory.db (SQLite database)\n")
	fmt.Printf("   ├── 📁 corpus/ (project documentation)\n")
	fmt.Printf("   ├── 📁 commissions/ (user objectives/goals)\n")
	fmt.Printf("   ├── 📁 campaigns/ (execution plans)\n")
	fmt.Printf("   ├── 📁 kanban/ (task tracking)\n")
	fmt.Printf("   ├── 📁 prompts/ (custom prompts)\n")
	fmt.Printf("   ├── 📁 tools/ (project-specific tools)\n")
	fmt.Printf("   └── 📁 workspaces/ (agent work areas)\n")

	// Load and display configuration summary
	enhancedConfig, err := config.LoadEnhancedConfig(ctx, path)
	if err == nil {
		fmt.Printf("\n📋 Project Configuration:\n")
		fmt.Printf("   Name: %s\n", enhancedConfig.Name)
		fmt.Printf("   Description: %s\n", enhancedConfig.Description)
		fmt.Printf("   Agents: %d configured\n", len(enhancedConfig.Agents))
		if enhancedConfig.GlobalProviders != nil {
			fmt.Printf("   Default Provider: %s\n", enhancedConfig.GlobalProviders.Default)
		}
	}

	fmt.Println("\n🚀 Next steps:")
	fmt.Println("   1. Review and customize .guild/guild.yaml")
	fmt.Println("   2. Set API keys as environment variables:")
	fmt.Println("      export ANTHROPIC_API_KEY=your-key")
	fmt.Println("      export OPENAI_API_KEY=your-key")
	fmt.Println("   3. Run 'guild corpus scan' to index your project")
	fmt.Println("   4. Run 'guild chat' to start working with your agents")

	return nil
}
