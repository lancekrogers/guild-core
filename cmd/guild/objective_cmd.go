// cmd/guild/objective_cmd.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/yourusername/guild/pkg/objective"
	"github.com/yourusername/guild/pkg/providers/generator"
	"github.com/yourusername/guild/pkg/ui/objective"
)

// Additional initialization to connect the UI
var objectiveCmd = &cobra.Command{
	Use:   "objective [objectivePath]",
	Short: "Manage Guild objectives",
	Long:  `Create, edit, and manage objective-based plans for Guild projects`,
	Run: func(cmd *cobra.Command, args []string) {
		var planner *objective.Planner

		// Setup providers
		genClient := setupGenerator()

		if len(args) > 0 {
			// Load existing objective
			objectivePath := args[0]
			obj, err := objective.ParseObjectiveFile(objectivePath)
			if err != nil {
				fmt.Printf("Error loading objective: %v\n", err)
				os.Exit(1)
			}

			planner = objective.NewPlanner(obj, genClient)
		} else {
			// Interactive mode to create a new objective
			fmt.Println("Describe your objective:")
			var description string
			fmt.Scanln(&description)

			obj, err := genClient.GenerateObjective(cmd.Context(), description)
			if err != nil {
				fmt.Printf("Error generating objective: %v\n", err)
				os.Exit(1)
			}

			planner = objective.NewPlanner(obj, genClient)
		}

		// Start the Bubble Tea UI
		model := objective_ui.NewModel(planner, genClient)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running UI: %v\n", err)
			os.Exit(1)
		}
	},
}

// setupGenerator creates and configures an LLM generator
func setupGenerator() generator.LLMGenerator {
	// Implementation that connects to your provider system
	// ...
}
