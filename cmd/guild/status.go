package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Guild server status",
	Long:  "Display the current status of the Guild gRPC server and related information",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("● Guild Server Status")
	fmt.Println()

	// Get status from daemon package
	status, err := daemon.Status()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get server status").
			WithComponent("cli").
			WithOperation("status.run")
	}

	// Check if running
	if daemon.IsRunning() {
		fmt.Printf("  Status: %sRunning%s\n", "\033[32m", "\033[0m") // Green
		
		// Read PID
		pidFile := daemon.GetPIDFilePath()
		if data, err := os.ReadFile(pidFile); err == nil {
			fmt.Printf("  PID: %s\n", string(data))
		}
		
		fmt.Printf("  Port: 9090\n")
		fmt.Printf("  Logs: %s\n", daemon.GetLogFilePath())
		
		// Check if reachable
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		if daemon.IsReachable(ctx) {
			fmt.Printf("  Reachable: %sYes%s\n", "\033[32m", "\033[0m") // Green
		} else {
			fmt.Printf("  Reachable: %sNo%s (process exists but not responding)\n", "\033[31m", "\033[0m") // Red
		}
	} else {
		fmt.Printf("  Status: %sStopped%s\n", "\033[31m", "\033[0m") // Red
		
		// Check for stale PID file
		if _, err := os.Stat(daemon.GetPIDFilePath()); err == nil {
			fmt.Printf("  Note: Stale PID file detected, will be cleaned on next start\n")
		}
		
		fmt.Println("\n💡 Start the server with any of these methods:")
		fmt.Println("   • Run any Guild command (e.g., 'guild chat')")
		fmt.Println("   • Start manually: 'guild serve'")
		fmt.Println("   • Start as daemon: 'guild serve --daemon'")
	}

	fmt.Println()
	fmt.Printf("📊 %s\n", status)

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