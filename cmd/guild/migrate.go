// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate global Guild data to project-local",
	Long: `Migrates Guild data from the global configuration to a project-local setup.

This command helps transition from the old global Guild setup to the new
project-local approach. It will:
- Copy corpus documents from global to project
- Optionally migrate embeddings (with --embeddings flag)
- Copy agent configurations
- Copy commissions`,
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Add flags
	migrateCmd.Flags().Bool("embeddings", false, "Also migrate embeddings (can be regenerated with 'guild corpus scan')")
	migrateCmd.Flags().Bool("overwrite", false, "Overwrite existing files in project")
	migrateCmd.Flags().Bool("dry-run", false, "Show what would be migrated without making changes")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	includeEmbeddings, _ := cmd.Flags().GetBool("embeddings")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Check if we're in a project
	if !project.IsInitialized(".") {
		return gerror.New(gerror.ErrCodeInvalidInput, "no Guild project found", nil).
			WithComponent("cli").
			WithOperation("migrate.run").
			WithDetails("help", "Run 'guild init' first")
	}

	// Get global path
	globalPath, err := project.GetGlobalGuildPath()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get global Guild path").
			WithComponent("cli").
			WithOperation("migrate.run")
	}

	// Check if global Guild exists
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		fmt.Println("No global Guild data found. Nothing to migrate.")
		return nil
	}

	// Create migration options
	opts := project.MigrationOptions{
		IncludeEmbeddings: includeEmbeddings,
		IncludeActivities: false, // Usually don't want to migrate activity logs
		OverwriteExisting: overwrite,
		DryRun:            dryRun,
	}

	if dryRun {
		fmt.Println("DRY RUN - No changes will be made")
		fmt.Println()
	}

	fmt.Printf("Migrating from: %s\n", globalPath)
	fmt.Printf("Migrating to: .guild/\n")
	fmt.Println()

	// Perform migration
	result, err := project.MigrateFromGlobal(ctx, ".", globalPath, opts)
	if err != nil {
		// Still show partial results even if there was an error
		if result != nil {
			fmt.Print(project.FormatMigrationSummary(result))
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "migration failed").
			WithComponent("cli").
			WithOperation("migrate.run")
	}

	// Show results
	fmt.Print(project.FormatMigrationSummary(result))

	if !dryRun && result.FilesCopied > 0 {
		fmt.Println("\nMigration complete!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Verify your project data: guild corpus list")
		if !includeEmbeddings {
			fmt.Println("  2. Regenerate embeddings: guild corpus scan")
		}
		fmt.Println("  3. (Optional) Remove global data: rm -rf " + globalPath)
	}

	return nil
}
