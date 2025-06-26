// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

var (
	cleanupForce bool
	cleanupAll   bool
)

func init() {
	cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false, "Force cleanup even if daemons appear to be running")
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Clean up all socket files (not just stale ones)")
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up stale Guild resources",
	Long: `Clean up stale Guild resources like socket files.

This command removes socket files that are no longer connected to running daemons.
Use --force to remove all socket files regardless of daemon status.

Examples:
  guild cleanup          # Clean up stale socket files
  guild cleanup --all    # Clean up all socket files
  guild cleanup --force  # Force cleanup even if daemons appear running`,
	RunE: runCleanup,
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("🧹 Cleaning up Guild resources...")

	// Clean up socket files
	if err := cleanupSockets(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to clean up sockets").
			WithComponent("cli").
			WithOperation("cleanup.run")
	}

	fmt.Println("✨ Cleanup complete!")
	return nil
}

func cleanupSockets(ctx context.Context) error {
	runDir, err := paths.GuildRunDir()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get run directory").
			WithComponent("cli").
			WithOperation("cleanup.cleanupSockets")
	}

	// Find all socket files
	pattern := filepath.Join(runDir, "*.sock")
	socketFiles, err := filepath.Glob(pattern)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to find socket files").
			WithComponent("cli").
			WithOperation("cleanup.cleanupSockets").
			WithDetails("pattern", pattern)
	}

	if len(socketFiles) == 0 {
		fmt.Println("ℹ️  No socket files found")
		return nil
	}

	fmt.Printf("📊 Found %d socket file(s)\n", len(socketFiles))

	staleCount := 0
	activeCount := 0

	for _, socketFile := range socketFiles {
		basename := filepath.Base(socketFile)

		if cleanupAll || cleanupForce {
			// Remove all sockets
			fmt.Printf("  • Removing %s... ", basename)
			if err := os.Remove(socketFile); err != nil {
				fmt.Printf("❌ %v\n", err)
			} else {
				fmt.Println("✅")
				staleCount++
			}
		} else {
			// Check if socket is active
			if daemon.CanConnect(socketFile) {
				fmt.Printf("  • %s [active] ", basename)
				if cleanupForce {
					if err := os.Remove(socketFile); err != nil {
						fmt.Printf("❌ %v\n", err)
					} else {
						fmt.Println("✅ (forced)")
						staleCount++
					}
				} else {
					fmt.Println("⏭️  (skipped)")
					activeCount++
				}
			} else {
				// Stale socket, remove it
				fmt.Printf("  • %s [stale] ", basename)
				if err := os.Remove(socketFile); err != nil {
					fmt.Printf("❌ %v\n", err)
				} else {
					fmt.Println("✅")
					staleCount++
				}
			}
		}
	}

	fmt.Println()
	if staleCount > 0 {
		fmt.Printf("🗑️  Removed %d stale socket(s)\n", staleCount)
	}
	if activeCount > 0 {
		fmt.Printf("🔌 Skipped %d active socket(s)\n", activeCount)
		fmt.Println("💡 Use --force to remove active sockets")
	}

	return nil
}
