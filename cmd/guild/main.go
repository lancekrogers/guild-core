// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// cmd/guild/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/observability"
)

// rootCmd represents the Great Hall command (root command in standard terminology)
// The Great Hall is where Guild teams gather and missions are coordinated
var rootCmd = &cobra.Command{
	Use:   "guild",
	Short: "Guild - Teams of Specialized Agents Working in Concert",
	Long: `Guild coordinates specialized artisans (agents) to complete strategic work.

🏰 COMMISSION specialized work to your Guild:
   guild commission "Build a REST API" --assign
   guild commission "Research caching strategy" --campaign performance

🔨 MONITOR the workshop and artisan progress:
   guild kanban view                 # Interactive kanban board
   guild commission status           # Commission progress
   guild campaign watch              # Watch campaign execution

🎯 COORDINATE campaigns and strategy:
   guild campaign start "Q1 Goals"
   guild chat                        # Interactive coordination

Each commission automatically decomposes work, assigns capable artisans,
and coordinates their collaboration through the workshop board.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

// versionCmd shows the current version of Guild
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the Guild version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Guild v0.1.0 - Agent Framework")
	},
}

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
	Long:  `Create, list, start, and stop agents in your Guild.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// agentStartCmd represents the agent start command
var agentStartCmd = &cobra.Command{
	Use:               "start [agent-id]",
	Short:             "Start an agent",
	Long:              `Start a specific agent or all agents if no ID is provided.`,
	ValidArgsFunction: completeAgentIDs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Printf("Starting agent %s...\n", args[0])
		} else {
			fmt.Println("Starting all agents...")
		}
		// TODO: Implement actual start functionality when agent orchestration is ready
		fmt.Println("Agent orchestration functionality coming soon.")
		fmt.Println("\nFor now, agents are automatically managed by the orchestrator when running commissions.")
		fmt.Println("Try: guild commission create \"Build a REST API\"")
	},
}

func init() {
	// Register commands that are defined in this file
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(campaignCmd)
	rootCmd.AddCommand(kanbanCmd)
	rootCmd.AddCommand(completionCmd)

	// Note: The following commands are registered in their respective files:
	// - initCmd (init.go)
	// - migrateCmd (migrate.go)
	// - corpusCmd (corpus.go)
	// - commissionCmd (commission.go)
	// - promptCmd (prompt.go)

	// Demo commands are not registered in production builds
	// - kanbanDemoCmd (kanban_demo.go) - for testing kanban UI
	// - costCmd (cost_demo.go) - for demonstrating cost-based selection

	// Register agent subcommands
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(newAgentTemplateCmd())

	// Note: Additional agent subcommands (list, stop, status) are registered in agent.go
}

// Execute summons the Guild and its artisans (standard: launches the CLI application)
func Execute() {
	// Suppress default error printing for better UX
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "The Guild regrets to inform you of an error: %v\n", err)
		os.Exit(1)
	}
}

// initializeGuild sets up environment and logging for Guild
func initializeGuild() context.Context {
	// Load .env file if it exists (for local development)
	if err := loadEnvironment(); err != nil {
		// Not a fatal error - continue without .env
		fmt.Fprintf(os.Stderr, "Note: Could not load .env file: %v\n", err)
	}

	// Initialize observability system
	logger := observability.NewLogger(nil)
	ctx := context.Background()
	ctx = observability.WithLogger(ctx, logger)
	
	// Set up request context for tracing
	ctx = observability.EnsureRequestContext(ctx)
	ctx = observability.WithComponent(ctx, "guild-cli")
	
	logger.InfoContext(ctx, "Guild CLI starting", 
		"version", "dev-local",
		"log_file_enabled", os.Getenv("GUILD_LOG_FILE") == "true",
	)
	
	return ctx
}

// loadEnvironment attempts to load .env files in order of preference
func loadEnvironment() error {
	// Try loading .env from current directory first (for development)
	if err := godotenv.Load(); err != nil {
		// Try loading from the executable's directory
		if execPath, err := os.Executable(); err == nil {
			envPath := filepath.Join(filepath.Dir(execPath), ".env")
			if err := godotenv.Load(envPath); err != nil {
				// Try loading from home directory
				if homeDir, err := os.UserHomeDir(); err == nil {
					envPath := filepath.Join(homeDir, ".guild", ".env")
					return godotenv.Load(envPath)
				}
				return err
			}
		}
	}
	return nil
}

func main() {
	// Initialize Guild environment and logging
	ctx := initializeGuild()
	
	// Store context for use in commands (you might want to pass this through somehow)
	_ = ctx
	
	// Assemble the Guild teams (standard: start the application)
	Execute()
}
