package main

import (
	"fmt"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	
	objective_ui "github.com/blockhead-consulting/guild/pkg/ui/objective"
)

// objectiveUICmd represents the command that launches the Objective UI
var objectiveUICmd = &cobra.Command{
	Use:   "ui [objectivePath]",
	Short: "Launch the Guild Hall Objective UI",
	Long: `Launch the Guild Hall interactive terminal UI for objective management.
With no arguments, it allows you to create a new objective.
With an objective path, it opens that specific objective for editing.`,
	Run: func(cmd *cobra.Command, args []string) {
		var objectivePath string
		if len(args) > 0 {
			objectivePath = args[0]
		}
		
		if err := runObjectiveUI(objectivePath); err != nil {
			fmt.Printf("Error running objective UI: %v\n", err)
		}
	},
}

// runObjectiveUI launches the Bubble Tea terminal UI for objectives
func runObjectiveUI(objectivePath string) error {
	model := objective_ui.NewModel(objectivePath)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func init() {
	objectiveCmd.AddCommand(objectiveUICmd)
}