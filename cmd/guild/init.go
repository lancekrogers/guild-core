package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/spf13/cobra"
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
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if already initialized
	if project.IsInitialized(path) {
		fmt.Fprintf(os.Stderr, "Error: Project already initialized at %s\n", absPath)
		fmt.Fprintln(os.Stderr, "The .guild directory already exists.")
		return nil
	}

	// Initialize project
	if err := project.Initialize(path); err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	// Success message
	fmt.Println("✅ Initialized Guild project")
	fmt.Printf("Created .guild/ directory structure at: %s\n", absPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add documents to your corpus: guild corpus add <file>")
	fmt.Println("  2. Create an agent: guild agent create <name>")
	fmt.Println("  3. Define objectives: guild objective create")

	return nil
}