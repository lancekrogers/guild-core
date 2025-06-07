// cmd/guild/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
   guild workshop                    # Show active work
   guild commission status           # Commission progress

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
	Use:   "start [agent-id]",
	Short: "Start an agent",
	Long:  `Start a specific agent or all agents if no ID is provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Printf("Starting agent %s...\n", args[0])
		} else {
			fmt.Println("Starting all agents...")
		}
		fmt.Println("This feature is not yet implemented.")
	},
}

func init() {
	// Register commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(corpusCmd)
	rootCmd.AddCommand(commissionCmd)
	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(campaignCmd)

	// Register agent subcommands
	agentCmd.AddCommand(agentStartCmd)
}

// Execute summons the Guild and its artisans (standard: launches the CLI application)
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "The Guild regrets to inform you of an error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// Assemble the Guild teams (standard: start the application)
	Execute()
}
