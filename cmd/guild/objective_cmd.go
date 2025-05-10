// cmd/guild/objective_cmd.go
//
// Guild Terminology Reference:
// ===========================
// - Scroll: Objective or specification document
// - Commission: Create or initiate
// - Refine: Edit or modify
// - Craft: Generate or produce content
// - Inspect: View or examine
// - Ledger: List or registry
// - Master Craftsman: LLM model
// - Guild: Team of specialized agents working together
// - Parchment: Format (markdown, json, yaml)
// - Archive: Storage or repository
// - Workbench: UI interface
//
// For debugging purposes, standard software engineering terms are
// included in comments alongside Guild terminology.
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
	// Using Guild-themed variable names with comments for standard terms
	createFlag   bool   // commission a scroll (create objective)
	listFlag     bool   // view the ledger (list objectives)
	editFlag     bool   // refine a scroll (edit objective)
	generateFlag bool   // craft content (generate content)
	viewFlag     bool   // inspect a scroll (view objective)
	modelFlag    string // master craftsman to consult (LLM model)
	providerFlag string // hall of knowledge to request assistance from (provider)
	formatFlag   string // parchment style (output format)
)

// scrollCmd represents the scroll command (objective in standard terminology)
// Scrolls are the Guild's formal documents defining work to be done
var objectiveCmd = &cobra.Command{
	Use:   "scroll [scrollPath]",
	Short: "Craft and manage Guild scrolls",
	Long:  `Commission, refine, inspect, and oversee scroll-based charters for Guild projects`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		
		// Prepare the Guild's resources (Setup components)
		store, objManager := setupObjectiveStore()
		llmClient, generator := setupGenerator()
		
		// Interpret the master's instructions (Process flags)
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
		
		// Default behavior - interactive crafting session (interactive mode)
		if len(args) > 0 {
			// Summon workbench with existing scroll (Launch UI with existing objective)
			launchObjectiveUI(ctx, args[0], objManager, generator)
		} else {
			// Summon workbench to commission new scroll (Launch UI to create new objective)
			launchCreateUI(ctx, objManager, generator)
		}
	},
}

func init() {
	rootCmd.AddCommand(objectiveCmd)
	
	// Add flags
	// These flags use Guild terminology with comments explaining the standard software terms
	objectiveCmd.Flags().BoolVarP(&createFlag, "commission", "c", false, "Commission a new scroll (create an objective)")
	objectiveCmd.Flags().BoolVarP(&listFlag, "ledger", "l", false, "View the ledger of all scrolls (list objectives)")
	objectiveCmd.Flags().BoolVarP(&editFlag, "refine", "r", false, "Refine an existing scroll (edit objective)")
	objectiveCmd.Flags().BoolVarP(&generateFlag, "craft", "g", false, "Craft content for a scroll (generate content)")
	objectiveCmd.Flags().BoolVarP(&viewFlag, "inspect", "i", false, "Inspect a scroll (view objective)")
	objectiveCmd.Flags().StringVarP(&modelFlag, "master", "m", "gpt-4", "Master craftsman to consult (LLM model)")
	objectiveCmd.Flags().StringVarP(&providerFlag, "hall", "p", "openai", "Hall of Knowledge to request assistance from (provider: openai, anthropic, ollama)")
	objectiveCmd.Flags().StringVarP(&formatFlag, "parchment", "f", "markdown", "Parchment style (output format: markdown, json, yaml)")
}

// setupArchiveLedger creates and configures the scroll archive and ledger
// (standard terminology: objective storage manager)
func setupObjectiveStore() (memory.Store, *objective.Manager) {
	// Locate the Guild's archive chambers (Get the data directory)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	
	dataDir := filepath.Join(homeDir, ".guild")
	dbPath := filepath.Join(dataDir, "guild.db")
	objectivesDir := filepath.Join(dataDir, "objectives")
	
	// Establish archive chambers if they don't exist (Create directories)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.MkdirAll(objectivesDir, 0755); err != nil {
		fmt.Printf("Error creating objectives directory: %v\n", err)
		os.Exit(1)
	}
	
	// Establish the ledger system (Create the store)
	store, err := boltdb.NewStore(dbPath)
	if err != nil {
		fmt.Printf("Error creating BoltDB store: %v\n", err)
		os.Exit(1)
	}
	
	// Appoint the Archive Master (Create the objective manager)
	objManager, err := objective.NewManager(store, objectivesDir)
	if err != nil {
		fmt.Printf("Error creating objective manager: %v\n", err)
		os.Exit(1)
	}
	
	// Prepare the Archive Master for duty (Initialize the manager)
	if err := objManager.Init(context.Background()); err != nil {
		fmt.Printf("Error initializing objective manager: %v\n", err)
		os.Exit(1)
	}
	
	return store, objManager
}

// summonMasterCraftsman creates and configures an LLM generator (standard: setup generator)
func setupGenerator() (providers.LLMClient, *objective.Generator) {
	// Establish the Great Hall for master craftsmen (Create provider factory)
	factory := providers.NewFactory()
	
	// Determine which Hall of Knowledge to access (Get provider type based on flag)
	provType := providers.ProviderType(providerFlag)
	
	// Prepare the summoning ritual specifications (Create config for the provider)
	config := providers.ProviderConfig{
		Type:  provType,
		Model: modelFlag,
	}
	
	// Register the Hall of Knowledge (Register the provider)
	if err := factory.RegisterProvider(config); err != nil {
		fmt.Printf("Error registering provider: %v\n", err)
		os.Exit(1)
	}
	
	// Designate as primary Hall of Knowledge (Set as default)
	factory.SetDefaultProvider(provType)
	
	// Establish communion with the master craftsman (Get the LLM client)
	llmClient, err := factory.GetDefaultClient()
	if err != nil {
		fmt.Printf("Error getting LLM client: %v\n", err)
		os.Exit(1)
	}
	
	// Initiate the crafting apparatus (Create the generator)
	generator := objective.NewGenerator(llmClient)
	
	return llmClient, generator
}

// ledgerInspection lists all scrolls (standard: list objectives)
func listObjectives(ctx context.Context, objManager *objective.Manager) {
	objectives, err := objManager.ListObjectives(ctx)
	if err != nil {
		fmt.Printf("Error listing objectives: %v\n", err)
		os.Exit(1)
	}
	
	if len(objectives) == 0 {
		fmt.Println("No scrolls found in the Guild archives")
		return
	}
	
	fmt.Println("Scrolls in the Guild Archives:")
	for _, obj := range objectives {
		fmt.Printf("- %s (%s) - %s\n", obj.Title, obj.ID, string(obj.Status))
	}
}

// commissionScroll creates a new scroll (standard: create objective)
func createObjective(ctx context.Context, cmd *cobra.Command, args []string, objManager *objective.Manager, generator *objective.Generator) {
	var description string
	
	if len(args) > 0 {
		description = args[0]
	} else {
		fmt.Println("Enter a description for your scroll commission:")
		fmt.Scanln(&description)
	}
	
	if description == "" {
		fmt.Println("Description cannot be empty")
		os.Exit(1)
	}
	
	// Generate the objective
	obj, err := generator.GenerateObjective(description)
	if err != nil {
		fmt.Printf("Error crafting scroll: %v\n", err)
		os.Exit(1)
	}
	
	// Save the objective
	if err := objManager.SaveObjective(ctx, obj); err != nil {
		fmt.Printf("Error preserving scroll in the archives: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Commissioned scroll '%s' (%s)\n", obj.Title, obj.ID)
	
	// Launch UI for editing
	launchObjectiveUI(ctx, obj.ID, objManager, generator)
}

// craftScrollContent generates additional content for a scroll (standard: generate objective content)
func generateObjectiveContent(ctx context.Context, cmd *cobra.Command, args []string, objManager *objective.Manager, generator *objective.Generator) {
	if len(args) == 0 {
		fmt.Println("Please specify a scroll ID or path in the archives")
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
		fmt.Printf("Error retrieving scroll from the archives: %v\n", err)
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
		fmt.Printf("Error preserving scroll in the archives: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Crafted %d tasks for scroll '%s'\n", len(tasks), obj.Title)
}

// inspectScroll displays a scroll (standard: view objective)
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
		fmt.Printf("Error retrieving scroll from the archives: %v\n", err)
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

// refineScroll opens a scroll for editing (standard: edit objective)
func editObjective(ctx context.Context, objectiveID string, objManager *objective.Manager, generator *objective.Generator) {
	// Just launch the UI for the objective
	launchObjectiveUI(ctx, objectiveID, objManager, generator)
}

// summonScrollWorkbench starts the Bubble Tea UI for a scroll (standard: launch objective UI)
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
		fmt.Printf("Error retrieving scroll from the archives: %v\n", err)
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

// summonCommissionWorkbench starts the Bubble Tea UI for creating a new scroll (standard: launch create UI)
func launchCreateUI(ctx context.Context, objManager *objective.Manager, generator *objective.Generator) {
	// Create an empty scroll (standard: empty objective)
	obj := objective.NewObjective("New Scroll", "Description of the commission goes here")
	
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