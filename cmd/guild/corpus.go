package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/blockhead-consulting/guild/pkg/corpus"
	corpus_ui "github.com/blockhead-consulting/guild/pkg/ui/corpus"
)

// corpusCmd represents the corpus command
var corpusCmd = &cobra.Command{
	Use:   "corpus [subcommand]",
	Short: "Manage the Guild's research corpus",
	Long: `Guild corpus is a knowledge repository system for storing research findings, 
	summaries, and generated insights in a structured, human-navigable format.
	
	When run without subcommands, launches the interactive UI for corpus browsing.
	Subcommands are available for command-line operations without the UI.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch the corpus UI by default when just "guild corpus" is run
		if err := runCorpusUI(""); err != nil {
			fmt.Printf("Error running corpus UI: %v\n", err)
		}
	},
}

// corpusCreateCmd represents the corpus create command
var corpusCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new document in the corpus",
	Long:  `Create a new document for the Guild's research corpus.`,
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		source, _ := cmd.Flags().GetString("source")
		guildID, _ := cmd.Flags().GetString("guild")
		agentID, _ := cmd.Flags().GetString("agent")
		
		// Check if we have a title
		if title == "" {
			fmt.Println("Error: Title is required")
			return
		}
		
		// Create the document
		doc := corpus.CorpusDoc{
			Title:     title,
			Tags:      tags,
			Source:    source,
			GuildID:   guildID,
			AgentID:   agentID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Body:      "",
		}
		
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}
		
		// Save the document
		err = corpus.Save(&doc, cfg)
		if err != nil {
			fmt.Printf("Error creating document: %v\n", err)
			return
		}
		
		fmt.Printf("Document '%s' created successfully at %s\n", doc.Title, doc.FilePath)
	},
}

// corpusListCmd represents the corpus list command
var corpusListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all documents in the corpus",
	Long:  `List all available documents in the Guild's research corpus.`,
	Run: func(cmd *cobra.Command, args []string) {
		tag, _ := cmd.Flags().GetString("tag")
		
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}
		
		// Get all documents
		docs, err := corpus.List(cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}
		
		// Filter by tag if provided
		if tag != "" {
			var filtered []corpus.CorpusDoc
			for _, doc := range docs {
				for _, t := range doc.Tags {
					if strings.EqualFold(t, tag) {
						filtered = append(filtered, doc)
						break
					}
				}
			}
			docs = filtered
		}
		
		// Display the documents
		if len(docs) == 0 {
			fmt.Println("No documents found")
			return
		}
		
		fmt.Printf("Found %d documents:\n\n", len(docs))
		for i, doc := range docs {
			fmt.Printf("%d. %s\n", i+1, doc.Title)
			if len(doc.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(doc.Tags, ", "))
			}
			fmt.Printf("   Created: %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05"))
			if !doc.UpdatedAt.IsZero() && doc.UpdatedAt != doc.CreatedAt {
				fmt.Printf("   Updated: %s\n", doc.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Println()
		}
	},
}

// corpusViewCmd represents the corpus view command
var corpusViewCmd = &cobra.Command{
	Use:   "view [title]",
	Short: "View a specific document",
	Long:  `View details of a specific document in the Guild's research corpus by title.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]
		
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}
		
		// Get all documents
		docs, err := corpus.List(cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}
		
		// Find the document by title
		var targetDoc *corpus.CorpusDoc
		for _, doc := range docs {
			if strings.EqualFold(doc.Title, title) {
				targetDoc = &doc
				break
			}
		}
		
		if targetDoc == nil {
			fmt.Printf("Document '%s' not found\n", title)
			return
		}
		
		// Load the document with content
		doc, err := corpus.Load(targetDoc.FilePath)
		if err != nil {
			fmt.Printf("Error loading document: %v\n", err)
			return
		}
		
		// Track the view
		if username := os.Getenv("USER"); username != "" {
			_ = corpus.TrackUserView(username, doc.FilePath, cfg)
		}
		
		// Display the document
		fmt.Printf("# %s\n\n", doc.Title)
		if len(doc.Tags) > 0 {
			fmt.Printf("Tags: %s\n\n", strings.Join(doc.Tags, ", "))
		}
		if doc.Source != "" {
			fmt.Printf("Source: %s\n\n", doc.Source)
		}
		fmt.Printf("Created: %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05"))
		if !doc.UpdatedAt.IsZero() && doc.UpdatedAt != doc.CreatedAt {
			fmt.Printf("Updated: %s\n", doc.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println("\n---\n")
		fmt.Println(doc.Body)
	},
}

// corpusDeleteCmd represents the corpus delete command
var corpusDeleteCmd = &cobra.Command{
	Use:   "delete [title]",
	Short: "Delete a document",
	Long:  `Delete a document from the Guild's research corpus by title.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := args[0]
		
		// Get confirmation
		fmt.Printf("Are you sure you want to delete document '%s'? (y/N): ", title)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Operation cancelled")
			return
		}
		
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}
		
		// Get all documents
		docs, err := corpus.List(cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}
		
		// Find the document by title
		var targetDoc *corpus.CorpusDoc
		for _, doc := range docs {
			if strings.EqualFold(doc.Title, title) {
				targetDoc = &doc
				break
			}
		}
		
		if targetDoc == nil {
			fmt.Printf("Document '%s' not found\n", title)
			return
		}
		
		// Delete the document
		err = corpus.Delete(targetDoc.FilePath)
		if err != nil {
			fmt.Printf("Error deleting document: %v\n", err)
			return
		}
		
		fmt.Printf("Document '%s' deleted successfully\n", title)
	},
}

// corpusGraphCmd represents the corpus graph command
var corpusGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Generate corpus document graph",
	Long:  `Generate and display a graph of document relationships in the Guild's research corpus.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}
		
		// Build the graph
		graph, err := corpus.BuildGraph(cfg)
		if err != nil {
			fmt.Printf("Error building graph: %v\n", err)
			return
		}
		
		// Display the graph
		fmt.Printf("Corpus Graph - %d documents, %d connections\n\n", len(graph.Nodes), len(graph.Edges))
		for _, node := range graph.Nodes {
			fmt.Printf("Document: %s\n", node.Title)
			
			// Find connections
			var connections []string
			for _, edge := range graph.Edges {
				if edge.From == node.Title {
					connections = append(connections, fmt.Sprintf("  → %s", edge.To))
				} else if edge.To == node.Title {
					connections = append(connections, fmt.Sprintf("  ← %s", edge.From))
				}
			}
			
			if len(connections) > 0 {
				fmt.Println("Connections:")
				for _, conn := range connections {
					fmt.Println(conn)
				}
			}
			fmt.Println()
		}
	},
}

// corpusUICmd represents the corpus UI command
var corpusUICmd = &cobra.Command{
	Use:   "ui [title]",
	Short: "Launch the corpus UI",
	Long:  `Launch the interactive terminal user interface for browsing and managing the corpus.`,
	Run: func(cmd *cobra.Command, args []string) {
		var title string
		if len(args) > 0 {
			title = args[0]
		}

		if err := runCorpusUI(title); err != nil {
			fmt.Printf("Error running corpus UI: %v\n", err)
		}
	},
}

// corpusConfigCmd represents the corpus config command
var corpusConfigCmd = &cobra.Command{
	Use:   "config [action]",
	Short: "Manage corpus configuration",
	Long:  `Manage configuration settings for the Guild's research corpus.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}

		// Display the current configuration
		fmt.Println("Current Corpus Configuration:")
		fmt.Printf("Corpus Path: %s\n", cfg.CorpusPath)
		fmt.Printf("Activities Path: %s\n", cfg.ActivitiesPath)
		fmt.Printf("Maximum Size: %d bytes (%d MB)\n", cfg.MaxSizeBytes, cfg.MaxSizeBytes/1024/1024)

		if len(cfg.DefaultTags) > 0 {
			fmt.Printf("Default Tags: %s\n", strings.Join(cfg.DefaultTags, ", "))
		} else {
			fmt.Println("Default Tags: None")
		}
	},
}

// corpusConfigSetCmd represents the corpus config set command
var corpusConfigSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long:  `Set a configuration value for the Guild's research corpus.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Get the corpus configuration
		cfg, err := getCorpusConfig()
		if err != nil {
			fmt.Printf("Error getting corpus configuration: %v\n", err)
			return
		}

		// Update the configuration
		switch key {
		case "corpus_path", "path":
			cfg.CorpusPath = value
		case "activities_path", "activities":
			cfg.ActivitiesPath = value
		case "max_size", "size":
			// Try to parse as MB first
			sizeInMB, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				fmt.Printf("Error parsing size value: %v\n", err)
				return
			}
			cfg.MaxSizeBytes = sizeInMB * 1024 * 1024
		case "default_tags", "tags":
			cfg.DefaultTags = strings.Split(value, ",")
		default:
			fmt.Printf("Unknown configuration key: %s\n", key)
			return
		}

		// Save the configuration
		configDir := filepath.Join(".guild")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}

		configPath := filepath.Join(configDir, "corpus.yml")
		if err := corpus.SaveConfig(cfg, configPath); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			return
		}

		fmt.Printf("Configuration updated: %s = %s\n", key, value)
	},
}

// runCorpusUI launches the Bubble Tea terminal UI for corpus browsing
func runCorpusUI(title string) error {
	// Get the corpus configuration
	cfg, err := getCorpusConfig()
	if err != nil {
		return fmt.Errorf("failed to get corpus configuration: %w", err)
	}
	
	// Get the current user
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}
	
	// Create the model
	model := corpus_ui.NewModel(cfg, username)
	
	// Create and run the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running corpus UI: %w", err)
	}
	
	return nil
}

// getCorpusConfig returns the configuration for the corpus system
func getCorpusConfig() (corpus.Config, error) {
	// Get the current working directory
	basePath, err := os.Getwd()
	if err != nil {
		return corpus.Config{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Look for config in standard locations
	configDir := filepath.Join(basePath, ".guild")
	configPath := filepath.Join(configDir, "corpus.yml")

	// Try to load from config file first
	cfg, err := corpus.LoadConfigFromFile(configPath)
	if err != nil {
		// Fall back to default config with environment overrides
		cfg = corpus.DefaultConfig()

		// Apply any environment variable overrides
		envVars := make(map[string]string)
		for _, env := range os.Environ() {
			kv := strings.SplitN(env, "=", 2)
			if len(kv) == 2 && strings.HasPrefix(kv[0], "GUILD_CORPUS_") {
				envVars[kv[0]] = kv[1]
			}
		}

		// Override with environment variables
		if path, ok := envVars[corpus.EnvCorpusPath]; ok && path != "" {
			cfg.CorpusPath = path
		}

		if path, ok := envVars[corpus.EnvActivitiesPath]; ok && path != "" {
			cfg.ActivitiesPath = path
		}

		if sizeStr, ok := envVars[corpus.EnvMaxSizeBytes]; ok && sizeStr != "" {
			if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
				cfg.MaxSizeBytes = size
			}
		}

		if tagsStr, ok := envVars[corpus.EnvDefaultTags]; ok && tagsStr != "" {
			// Parse comma-separated tags
			cfg.DefaultTags = strings.Split(tagsStr, ",")
		}
	}

	// Ensure the corpus directory exists
	if err := os.MkdirAll(cfg.CorpusPath, 0755); err != nil {
		return cfg, fmt.Errorf("failed to create corpus directory: %w", err)
	}

	// Ensure the activities directory exists
	if err := os.MkdirAll(cfg.ActivitiesPath, 0755); err != nil {
		return cfg, fmt.Errorf("failed to create activities directory: %w", err)
	}

	return cfg, nil
}

// setupCorpusFlags adds common flags to the corpus commands
func setupCorpusFlags() {
	// Flags for the create command
	corpusCreateCmd.Flags().String("title", "", "Title of the document (required)")
	corpusCreateCmd.MarkFlagRequired("title")
	corpusCreateCmd.Flags().StringSlice("tags", []string{}, "Tags for the document (comma-separated)")
	corpusCreateCmd.Flags().String("source", "", "Source of the document information")
	corpusCreateCmd.Flags().String("guild", "", "Guild ID associated with the document")
	corpusCreateCmd.Flags().String("agent", "", "Agent ID that created the document")
	
	// Flags for the list command
	corpusListCmd.Flags().String("tag", "", "Filter documents by tag")
}

func init() {
	// Register commands
	rootCmd.AddCommand(corpusCmd)

	// Register corpus subcommands
	corpusCmd.AddCommand(corpusCreateCmd)
	corpusCmd.AddCommand(corpusListCmd)
	corpusCmd.AddCommand(corpusViewCmd)
	corpusCmd.AddCommand(corpusDeleteCmd)
	corpusCmd.AddCommand(corpusGraphCmd)
	corpusCmd.AddCommand(corpusUICmd)
	corpusCmd.AddCommand(corpusConfigCmd)

	// Register corpus config subcommands
	corpusConfigCmd.AddCommand(corpusConfigSetCmd)

	// Setup flags
	setupCorpusFlags()
}