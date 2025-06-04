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
	Long: `Guild is a fellowship of specialized agents collaborating
through an orchestrated workflow to complete complex tasks.

Within these hallowed halls, artisans of various disciplines coordinate
their efforts through a shared system of scrolls (objectives),
ledgers (kanban boards), and archives (memory systems).`,
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

// objectiveCmd represents the objective command
var objectiveCmd = &cobra.Command{
	Use:   "objective [subcommand]",
	Short: "Manage objectives through UI or subcommands",
	Long:  `Create, list, view, and manage objectives for your Guild agents.

When run without subcommands, launches the interactive UI for objective management.
Subcommands are available for command-line operations without the UI.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch the objective UI by default when just "guild objective" is run
		// TODO: Implement objective UI
		fmt.Println("Objective UI not yet implemented. Use 'guild objective --help' to see available commands.")
	},
}

// objectiveCreateCmd represents the objective create command
var objectiveCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new objective",
	Long:  `Create a new objective for Guild agents to work on.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Creating new objective...")
		fmt.Println("This feature is not yet implemented.")
	},
}

// objectiveListCmd represents the objective list command
var objectiveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all objectives",
	Long:  `List all available objectives.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing objectives...")
		fmt.Println("This feature is not yet implemented.")
	},
}

// objectiveViewCmd represents the objective view command
var objectiveViewCmd = &cobra.Command{
	Use:   "view [id]",
	Short: "View a specific objective",
	Long:  `View details of a specific objective by ID.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Viewing objective %s...\n", args[0])
		fmt.Println("This feature is not yet implemented.")
	},
}

// objectiveUICmd represents the objective UI command
var objectiveUICmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the objective UI",
	Long:  `Launch the interactive terminal user interface for managing objectives.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Launching objective UI...")
		fmt.Println("This feature is not yet implemented.")
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
	rootCmd.AddCommand(objectiveCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(orchestratorCmd)
	rootCmd.AddCommand(kanbanDemoCmd)
	// rootCmd.AddCommand(campaignCmd)  // TODO: implement
	// rootCmd.AddCommand(chatCmd)      // TODO: implement

	// Register objective subcommands
	objectiveCmd.AddCommand(objectiveCreateCmd)
	objectiveCmd.AddCommand(objectiveListCmd)
	objectiveCmd.AddCommand(objectiveViewCmd)
	objectiveCmd.AddCommand(objectiveUICmd)

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