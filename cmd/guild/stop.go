// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/campaign"
	pkgDaemon "github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

var (
	stopCampaignID string
	stopAll        bool
	stopSession    int
	stopForce      bool
	stopTimeout    time.Duration
)

func init() {
	stopCmd.Flags().StringVar(&stopCampaignID, "campaign", "", "Stop specific campaign daemon")
	stopCmd.Flags().BoolVar(&stopAll, "all", false, "Stop all running daemons")
	stopCmd.Flags().IntVar(&stopSession, "session", -1, "Stop specific session (default: all sessions for campaign)")
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force kill processes (SIGKILL instead of SIGTERM)")
	stopCmd.Flags().DurationVar(&stopTimeout, "timeout", 5*time.Second, "Timeout for graceful shutdown")

	// Register completion functions
	stopCmd.RegisterFlagCompletionFunc("campaign", completeCampaignNames)
	stopCmd.RegisterFlagCompletionFunc("session", completeSessionIDs)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Guild daemon instances",
	Long: `Stop running Guild daemon instances.

By default, stops the daemon for the current campaign.
Use --campaign to stop a specific campaign's daemon.
Use --all to stop all running daemons.

Examples:
  guild stop                     # Stop daemon for current campaign
  guild stop --campaign shop     # Stop the 'shop' campaign daemon
  guild stop --session 2         # Stop session 2 of current campaign
  guild stop --all               # Stop all running daemons
  guild stop --all --force       # Force kill all daemons`,
	RunE: runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine what to stop
	if stopAll {
		return stopAllDaemons(ctx)
	}

	// Detect campaign if not specified
	if stopCampaignID == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
				WithComponent("cli").
				WithOperation("stop.run")
		}

		detectedCampaign, err := campaign.DetectCampaign(cwd, "")
		if err != nil {
			// If no campaign detected, try to find any running daemons
			allSessions, err := pkgDaemon.DiscoverAllRunningSessions()
			if err != nil || len(allSessions) == 0 {
				// No running daemons found
				fmt.Println("ℹ️  No running Guild daemons found")
				fmt.Println()
				fmt.Println("💡 Tips:")
				fmt.Println("  • Use 'guild status' to see running daemons")
				fmt.Println("  • Use 'guild stop --all' to stop all daemons")
				fmt.Println("  • Use 'guild stop --campaign <name>' to stop a specific campaign")
				return nil
			}

			// Show running daemons and suggest using --all
			fmt.Println("🔍 No campaign detected in current directory")
			fmt.Println()
			fmt.Printf("Found %d running daemon(s):\n", len(allSessions))
			for hash, sessions := range allSessions {
				for _, session := range sessions {
					fmt.Printf("  • %s (session %d)\n", hash, session.Session)
				}
			}
			fmt.Println()
			fmt.Println("💡 Use 'guild stop --all' to stop all daemons")
			fmt.Println("   or navigate to a campaign directory and run 'guild stop'")
			return nil
		}
		stopCampaignID = detectedCampaign
	}

	// Stop specific campaign
	return stopCampaignDaemons(ctx, stopCampaignID)
}

func stopAllDaemons(ctx context.Context) error {
	fmt.Println("🛑 Stopping all Guild daemons...")

	// Use lifecycle manager for graceful shutdown
	lifecycleManager := daemon.DefaultLifecycleManager
	if err := lifecycleManager.ShutdownAll(ctx, stopTimeout); err != nil {
		// If graceful shutdown failed, try using daemon manager
		manager := daemon.DefaultManager
		if err := manager.StopAll(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop all daemons").
				WithComponent("cli").
				WithOperation("stop.stopAllDaemons")
		}
	}

	// Also discover and stop any unmanaged sessions
	allSessions, err := pkgDaemon.DiscoverAllRunningSessions()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to discover sessions: %v\n", err)
	} else {
		stoppedCount := 0
		for campaignHash, sessions := range allSessions {
			for _, session := range sessions {
				fmt.Printf("  • Stopping %s (session %d)... ", campaignHash, session.Session)
				if err := pkgDaemon.StopSession(session.Socket); err != nil {
					if stopForce {
						// Force remove socket
						os.Remove(session.Socket)
						fmt.Println("✅ (forced)")
					} else {
						fmt.Printf("❌ %v\n", err)
					}
				} else {
					fmt.Println("✅")
					stoppedCount++
				}
			}
		}

		if stoppedCount > 0 {
			fmt.Printf("\n✨ Stopped %d daemon(s)\n", stoppedCount)
		} else if len(allSessions) == 0 {
			fmt.Println("ℹ️  No running daemons found")
		}
	}

	return nil
}

func stopCampaignDaemons(ctx context.Context, campaignName string) error {
	fmt.Printf("🛑 Stopping Guild daemon for campaign '%s'...\n", campaignName)

	// If specific session requested
	if stopSession >= 0 {
		return stopSpecificSession(ctx, campaignName, stopSession)
	}

	// Stop all sessions for the campaign
	manager := daemon.DefaultManager
	if err := manager.StopCampaign(campaignName); err != nil {
		gerr, ok := err.(*gerror.GuildError)
		if ok && gerr.Code == gerror.ErrCodeNotFound {
			fmt.Printf("ℹ️  No running sessions found for campaign '%s'\n", campaignName)
			return nil
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop campaign").
			WithComponent("cli").
			WithOperation("stop.stopCampaignDaemons").
			WithDetails("campaign", campaignName)
	}

	fmt.Printf("✨ Successfully stopped all sessions for campaign '%s'\n", campaignName)
	return nil
}

func stopSpecificSession(ctx context.Context, campaignName string, session int) error {
	fmt.Printf("🛑 Stopping session %d for campaign '%s'...\n", session, campaignName)

	// Get socket path for the session
	socketPath, err := paths.GetCampaignSocket(campaignName, session)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get socket path").
			WithComponent("cli").
			WithOperation("stop.stopSpecificSession").
			WithDetails("campaign", campaignName).
			WithDetails("session", session)
	}

	// Check if session is running
	if !pkgDaemon.CanConnect(socketPath) {
		fmt.Printf("ℹ️  Session %d is not running\n", session)
		return nil
	}

	// Stop the session
	if err := pkgDaemon.StopSession(socketPath); err != nil {
		if stopForce {
			// Force remove socket
			os.Remove(socketPath)
			fmt.Printf("✅ Forcefully stopped session %d\n", session)
		} else {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stop session").
				WithComponent("cli").
				WithOperation("stop.stopSpecificSession").
				WithDetails("campaign", campaignName).
				WithDetails("session", session)
		}
	} else {
		fmt.Printf("✅ Successfully stopped session %d\n", session)
	}

	return nil
}
