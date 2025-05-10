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