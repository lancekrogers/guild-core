// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/campaign"
	pkgDaemon "github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/spf13/cobra"
)

var (
	statusAll bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Guild server status",
	Long: `Display the current status of the Guild gRPC server and related information.

This command shows:
- Current campaign and daemon status
- Transport type (Unix socket or TCP)
- Connection details (socket path or port)
- Log file locations

Multi-Instance Support:
- Use --all to show all running Guild daemon instances
- Default view shows status for current directory's campaign
- Detects campaign from current working directory

Examples:
  guild status          # Show status for current campaign
  guild status --all    # Show all running daemon instances`,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Show status of all running Guild daemons")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusAll {
		return runStatusAll(cmd)
	}
	return runStatusSingle(cmd)
}

func runStatusSingle(cmd *cobra.Command) error {
	fmt.Println("● Guild Server Status")
	fmt.Println()

	// Detect current campaign
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	
	campaignName, err := campaign.DetectCampaign(cwd, "")
	if err != nil {
		fmt.Println("  Campaign: Not in a campaign directory")
		fmt.Println("  Status: No campaign detected")
		fmt.Println("\n💡 Initialize a campaign with 'guild init campaign'")
		return nil
	}

	// Get daemon configuration
	daemonConfig, err := daemon.GetDaemonConfig(campaignName, 0)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get daemon config").
			WithComponent("cli").
			WithOperation("status.run")
	}

	fmt.Printf("  Campaign: %s\n", campaignName)
	fmt.Printf("  Transport: Unix Socket\n")

	// Check if daemon is running
	isReachable := pkgDaemon.CanConnect(daemonConfig.SocketPath)
	isRunning := isReachable

	if isRunning {
		fmt.Printf("  Status: %sRunning%s\n", "\033[32m", "\033[0m") // Green
		fmt.Printf("  Socket: %s\n", daemonConfig.SocketPath)

		// Read PID from file if available
		if daemonConfig.PIDFile != "" {
			if data, err := os.ReadFile(daemonConfig.PIDFile); err == nil {
				fmt.Printf("  PID: %s\n", strings.TrimSpace(string(data)))
			}
		}

		if daemonConfig.LogFile != "" {
			fmt.Printf("  Logs: %s\n", daemonConfig.LogFile)
		}

		if isReachable {
			fmt.Printf("  Reachable: %sYes%s\n", "\033[32m", "\033[0m") // Green
		} else {
			fmt.Printf("  Reachable: %sNo%s (process exists but not responding)\n", "\033[31m", "\033[0m") // Red
		}
	} else {
		fmt.Printf("  Status: %sStopped%s\n", "\033[31m", "\033[0m") // Red

		fmt.Println("\n💡 Start the server with any of these methods:")
		fmt.Printf("   • Run 'guild chat --campaign %s'\n", campaignName)
		fmt.Printf("   • Start manually: 'guild serve --campaign %s'\n", campaignName)
		fmt.Printf("   • Start as daemon: 'guild serve --campaign %s --daemon'\n", campaignName)
	}

	// Show guild directory info
	homeDir, _ := os.UserHomeDir()
	guildDir := filepath.Join(homeDir, ".guild")
	if info, err := os.Stat(guildDir); err == nil {
		fmt.Println()
		fmt.Println("● Guild Directory")
		fmt.Printf("  Path: %s\n", guildDir)
		fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runStatusAll(cmd *cobra.Command) error {
	fmt.Println("● All Guild Daemon Instances")
	fmt.Println()

	// Check legacy daemon
	legacyRunning := daemon.IsRunning()
	if legacyRunning {
		fmt.Println("🏰 Legacy Daemon (TCP Mode)")
		fmt.Printf("  Campaign: default\n")
		fmt.Printf("  Transport: TCP\n")
		fmt.Printf("  Status: %sRunning%s\n", "\033[32m", "\033[0m")
		fmt.Printf("  Port: 9090\n")
		fmt.Printf("  Logs: %s\n", daemon.GetLogFilePath())
		fmt.Println()
	}

	// Discover all running campaign daemons
	manager := daemon.DefaultManager
	allSessions, err := manager.ListRunning()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to discover running daemons").
			WithComponent("cli").
			WithOperation("status.all")
	}

	if len(allSessions) == 0 && !legacyRunning {
		fmt.Println("ℹ️ No Guild daemon instances are currently running")
		fmt.Println()
		fmt.Println("💡 Start a daemon with:")
		fmt.Println("   • guild serve --campaign <campaign-name>")
		fmt.Println("   • guild chat --campaign <campaign-name>")
		fmt.Println("   • guild serve (for legacy mode)")
		return nil
	}

	// Display campaign-specific daemons
	for campaignHash, sessions := range allSessions {
		fmt.Printf("🏰 Campaign: %s\n", campaignHash) // TODO: Map hash back to campaign name
		for _, session := range sessions {
			fmt.Printf("  Session %d:\n", session.Session)
			fmt.Printf("    Status: %s%s%s\n", "\033[32m", session.Status, "\033[0m")
			fmt.Printf("    Socket: %s\n", session.Socket)
			fmt.Printf("    Transport: Unix Socket\n")
			fmt.Println()
		}
	}

	// Show summary
	totalInstances := len(allSessions)
	if legacyRunning {
		totalInstances++
	}
	fmt.Printf("📊 Total instances: %d\n", totalInstances)

	// Show guild directory info
	homeDir, _ := os.UserHomeDir()
	guildDir := filepath.Join(homeDir, ".guild")
	if info, err := os.Stat(guildDir); err == nil {
		fmt.Println()
		fmt.Println("● Guild Directory")
		fmt.Printf("  Path: %s\n", guildDir)
		fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	}

	return nil
}

func getTransportType(config *daemon.DaemonConfig) string {
	if config.UseSocket {
		return "Unix Socket"
	}
	return "TCP"
}
