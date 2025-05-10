// cmd/guild/objective_cmd.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/memory/boltdb"
	"github.com/blockhead-consulting/Guild/pkg/objective"
	"github.com/blockhead-consulting/Guild/pkg/providers"
	"github.com/blockhead-consulting/Guild/pkg/generator/objective"
	objectiveui "github.com/blockhead-consulting/Guild/pkg/ui/objective"
)

var (
	createFlag   bool
	listFlag     bool
	editFlag     bool
	generateFlag bool
	viewFlag     bool
	modelFlag    string
	providerFlag string
	formatFlag   string
)

// objectiveCmd represents the objective command
var objectiveCmd = &cobra.Command{
	Use:   "objective [objectivePath]",
	Short: "Manage Guild objectives",
	Long:  `Create, edit, view, and manage objective-based plans for Guild projects`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		
		// Setup components
		store, objManager := setupObjectiveStore()
		llmClient, generator := setupGenerator()
		
		// Process flags
		if listFlag {
			listObjectives(ctx, objManager)
			return
		}
		
		if createFlag {
			createObjective(ctx, cmd, args, objManager, generator)
			return
		}
		
		if generateFlag {
			generateObjectiveContent(ctx, cmd, args, objManager, generator)
			return
		}
		
		if viewFlag && len(args) > 0 {
			viewObjective(ctx, args[0], objManager)
			return
		}
		
		if editFlag && len(args) > 0 {
			editObjective(ctx, args[0], objManager, generator)
			return
		}
		
		// Default behavior - interactive mode
		if len(args) > 0 {
			// Launch UI with existing objective
			launchObjectiveUI(ctx, args[0], objManager, generator)
		} else {
			// Launch UI to create new objective
			launchCreateUI(ctx, objManager, generator)
		}
	},
}

func init() {
	rootCmd.AddCommand(objectiveCmd)
	
	// Add flags
	objectiveCmd.Flags().BoolVarP(&createFlag, "create", "c", false, "Create a new objective")
	objectiveCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List all objectives")
	objectiveCmd.Flags().BoolVarP(&editFlag, "edit", "e", false, "Edit an existing objective")
	objectiveCmd.Flags().BoolVarP(&generateFlag, "generate", "g", false, "Generate content for an objective")
	objectiveCmd.Flags().BoolVarP(&viewFlag, "view", "v", false, "View an objective")
	objectiveCmd.Flags().StringVarP(&modelFlag, "model", "m", "gpt-4", "LLM model to use")
	objectiveCmd.Flags().StringVarP(&providerFlag, "provider", "p", "openai", "LLM provider to use (openai, anthropic, ollama)")
	objectiveCmd.Flags().StringVarP(&formatFlag, "format", "f", "markdown", "Output format (markdown, json, yaml)")
}

// setupObjectiveStore creates and configures the objective storage
func setupObjectiveStore() (memory.Store, *objective.Manager) {
	// Get the data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	
	dataDir := filepath.Join(homeDir, ".guild")
	dbPath := filepath.Join(dataDir, "guild.db")
	objectivesDir := filepath.Join(dataDir, "objectives")
	
	// Create the directories if they don't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.MkdirAll(objectivesDir, 0755); err != nil {
		fmt.Printf("Error creating objectives directory: %v\n", err)
		os.Exit(1)
	}
	
	// Create the store
	store, err := boltdb.NewStore(dbPath)
	if err != nil {
		fmt.Printf("Error creating BoltDB store: %v\n", err)
		os.Exit(1)
	}
	
	// Create the objective manager
	objManager, err := objective.NewManager(store, objectivesDir)
	if err != nil {
		fmt.Printf("Error creating objective manager: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize the manager
	if err := objManager.Init(context.Background()); err != nil {
		fmt.Printf("Error initializing objective manager: %v\n", err)
		os.Exit(1)
	}
	
	return store, objManager
}

// setupGenerator creates and configures an LLM generator
func setupGenerator() (providers.LLMClient, *objective.Generator) {
	// Create provider factory
	factory := providers.NewFactory()
	
	// Get provider type based on flag
	provType := providers.ProviderType(providerFlag)
	
	// Create config for the provider
	config := providers.ProviderConfig{
		Type:  provType,
		Model: modelFlag,
	}
	
	// Register the provider
	if err := factory.RegisterProvider(config); err != nil {
		fmt.Printf("Error registering provider: %v\n", err)
		os.Exit(1)
	}
	
	// Set as default
	factory.SetDefaultProvider(provType)
	
	// Get the LLM client
	llmClient, err := factory.GetDefaultClient()
	if err != nil {
		fmt.Printf("Error getting LLM client: %v\n", err)
		os.Exit(1)
	}
	
	// Create the generator
	generator := objective.NewGenerator(llmClient)
	
	return llmClient, generator
}

// listObjectives lists all objectives
func listObjectives(ctx context.Context, objManager *objective.Manager) {
	objectives, err := objManager.ListObjectives(ctx)
	if err != nil {
		fmt.Printf("Error listing objectives: %v\n", err)
		os.Exit(1)
	}
	
	if len(objectives) == 0 {
		fmt.Println("No objectives found")
		return
	}
	
	fmt.Println("Objectives:")
	for _, obj := range objectives {
		fmt.Printf("- %s (%s) - %s\n", obj.Title, obj.ID, string(obj.Status))
	}
}

// createObjective creates a new objective
func createObjective(ctx context.Context, cmd *cobra.Command, args []string, objManager *objective.Manager, generator *objective.Generator) {
	var description string
	
	if len(args) > 0 {
		description = args[0]
	} else {
		fmt.Println("Enter a description for your objective:")
		fmt.Scanln(&description)
	}
	
	if description == "" {
		fmt.Println("Description cannot be empty")
		os.Exit(1)
	}
	
	// Generate the objective
	obj, err := generator.GenerateObjective(description)
	if err != nil {
		fmt.Printf("Error generating objective: %v\n", err)
		os.Exit(1)
	}
	
	// Save the objective
	if err := objManager.SaveObjective(ctx, obj); err != nil {
		fmt.Printf("Error saving objective: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Created objective '%s' (%s)\n", obj.Title, obj.ID)
	
	// Launch UI for editing
	launchObjectiveUI(ctx, obj.ID, objManager, generator)
}

// generateObjectiveContent generates additional content for an objective
func generateObjectiveContent(ctx context.Context, cmd *cobra.Command, args []string, objManager *objective.Manager, generator *objective.Generator) {
	if len(args) == 0 {
		fmt.Println("Please specify an objective ID or path")
		os.Exit(1)
	}
	
	objectiveID := args[0]
	
	// Check if it's a file path or ID
	var obj *objective.Objective
	var err error
	
	if _, err := os.Stat(objectiveID); err == nil {
		// It's a file path
		obj, err = objManager.LoadObjectiveFromFile(ctx, objectiveID)
	} else {
		// It's an ID
		obj, err = objManager.GetObjective(ctx, objectiveID)
	}
	
	if err != nil {
		fmt.Printf("Error loading objective: %v\n", err)
		os.Exit(1)
	}
	
	// Generate tasks for the objective
	tasks, err := generator.GenerateTasks(obj)
	if err != nil {
		fmt.Printf("Error generating tasks: %v\n", err)
		os.Exit(1)
	}
	
	// Add tasks to the objective
	for _, task := range tasks {
		obj.Tasks = append(obj.Tasks, task)
	}
	
	// Save the objective
	if err := objManager.SaveObjective(ctx, obj); err != nil {
		fmt.Printf("Error saving objective: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Generated %d tasks for objective '%s'\n", len(tasks), obj.Title)
}

// viewObjective displays an objective
func viewObjective(ctx context.Context, objectiveID string, objManager *objective.Manager) {
	// Check if it's a file path or ID
	var obj *objective.Objective
	var err error
	
	if _, err := os.Stat(objectiveID); err == nil {
		// It's a file path
		obj, err = objManager.LoadObjectiveFromFile(ctx, objectiveID)
	} else {
		// It's an ID
		obj, err = objManager.GetObjective(ctx, objectiveID)
	}
	
	if err != nil {
		fmt.Printf("Error loading objective: %v\n", err)
		os.Exit(1)
	}
	
	// Print the objective
	fmt.Printf("Title: %s\n", obj.Title)
	fmt.Printf("Status: %s\n", obj.Status)
	fmt.Printf("Priority: %s\n", obj.Priority)
	fmt.Printf("Description: %s\n\n", obj.Description)
	
	// Print sections
	for _, part := range obj.Parts {
		fmt.Printf("## %s\n\n%s\n\n", part.Title, part.Content)
	}
	
	// Print tasks
	fmt.Println("## Tasks")
	for _, task := range obj.Tasks {
		status := "[ ]"
		if task.Status == "done" {
			status = "[x]"
		}
		fmt.Printf("%s %s\n", status, task.Title)
	}
}

// editObjective opens an objective for editing
func editObjective(ctx context.Context, objectiveID string, objManager *objective.Manager, generator *objective.Generator) {
	// Just launch the UI for the objective
	launchObjectiveUI(ctx, objectiveID, objManager, generator)
}

// launchObjectiveUI starts the Bubble Tea UI for an objective
func launchObjectiveUI(ctx context.Context, objectiveID string, objManager *objective.Manager, generator *objective.Generator) {
	// Check if it's a file path or ID
	var obj *objective.Objective
	var err error
	
	if _, err := os.Stat(objectiveID); err == nil {
		// It's a file path
		obj, err = objManager.LoadObjectiveFromFile(ctx, objectiveID)
	} else {
		// It's an ID
		obj, err = objManager.GetObjective(ctx, objectiveID)
	}
	
	if err != nil {
		fmt.Printf("Error loading objective: %v\n", err)
		os.Exit(1)
	}
	
	// Create planner
	planner := objectiveui.NewPlanner(obj, generator, objManager)
	
	// Start UI
	model := objectiveui.NewModel(planner)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}

// launchCreateUI starts the Bubble Tea UI for creating a new objective
func launchCreateUI(ctx context.Context, objManager *objective.Manager, generator *objective.Generator) {
	// Create an empty objective
	obj := objective.NewObjective("New Objective", "Description goes here")
	
	// Create planner
	planner := objectiveui.NewPlanner(obj, generator, objManager)
	
	// Start UI
	model := objectiveui.NewModel(planner)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}
}