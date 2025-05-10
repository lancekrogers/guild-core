package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	
	"github.com/blockhead-consulting/guild/pkg/memory/boltdb"
	generator "github.com/blockhead-consulting/guild/pkg/generator/objective"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	objective_ui "github.com/blockhead-consulting/guild/pkg/ui/objective"
)

// objectiveUICmd represents the command that explicitly launches the Objective UI
var objectiveUICmd = &cobra.Command{
	Use:   "ui [objectivePath]",
	Short: "Launch the Guild Hall Objective UI",
	Long: `Launch the Guild Hall interactive terminal UI for objective management.
This is the same UI that opens when you run 'guild objective' with no subcommand.
With no arguments, it allows you to create a new objective.
With an objective path, it opens that specific objective for editing.`,
	Run: func(cmd *cobra.Command, args []string) {
		var objectivePath string
		if len(args) > 0 {
			objectivePath = args[0]
		}
		
		if err := runObjectiveUI(objectivePath); err \!= nil {
			fmt.Printf("Error running objective UI: %v\n", err)
		}
	},
}

// runObjectiveUI launches the Bubble Tea terminal UI for objectives
func runObjectiveUI(objectivePath string) error {
	// Initialize the memory store
	ctx := context.Background()
	dbPath := filepath.Join(os.TempDir(), "guild_objectives.db")
	store, err := boltdb.NewStore(dbPath)
	if err \!= nil {
		return fmt.Errorf("failed to create memory store: %w", err)
	}

	// Initialize the objective manager
	basePath, err := os.Getwd()
	if err \!= nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	objectivesPath := filepath.Join(basePath, "objectives")
	if err := os.MkdirAll(objectivesPath, 0755); err \!= nil {
		return fmt.Errorf("failed to create objectives directory: %w", err)
	}

	manager, err := objective.NewManager(store, objectivesPath)
	if err \!= nil {
		return fmt.Errorf("failed to create objective manager: %w", err)
	}

	// Initialize the lifecycle manager
	lifecycleManager := objective.NewLifecycleManager(manager, basePath)

	// Initialize the planner
	planner := objective.NewPlanner(manager, lifecycleManager)

	// Initialize the generator
	// We'll use a mock LLM client for the generator
	var gen *generator.Generator
	factory := providers.NewFactory()
	client, err := factory.GetClient(providers.ProviderMock)
	if err == nil {
		gen, _ = generator.NewGenerator(client)
	}

	// Create the model with dependencies
	model := objective_ui.NewModel(objectivePath, manager, planner, gen)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func init() {
	objectiveCmd.AddCommand(objectiveUICmd)
}
