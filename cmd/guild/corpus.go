// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/pkg/corpus"
	"github.com/lancekrogers/guild/pkg/gerror"
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
		err = corpus.Save(context.Background(), &doc, cfg)
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

		// Get all document paths
		docPaths, err := corpus.List(context.Background(), cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}

		// Load documents and filter by tag if provided
		var docs []corpus.CorpusDoc
		for _, path := range docPaths {
			doc, err := corpus.Load(context.Background(), path)
			if err != nil {
				continue // Skip documents that can't be loaded
			}

			// Filter by tag if provided
			if tag != "" {
				hasTag := false
				for _, t := range doc.Tags {
					if strings.EqualFold(t, tag) {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			docs = append(docs, *doc)
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

		// Get all document paths
		docPaths, err := corpus.List(context.Background(), cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}

		// Find the document by title
		var targetDoc *corpus.CorpusDoc
		for _, path := range docPaths {
			doc, err := corpus.Load(context.Background(), path)
			if err != nil {
				continue
			}
			if strings.EqualFold(doc.Title, title) {
				targetDoc = doc
				break
			}
		}

		if targetDoc == nil {
			fmt.Printf("Document '%s' not found\n", title)
			return
		}

		// Document is already loaded with content
		doc, err := corpus.Load(context.Background(), targetDoc.FilePath)
		if err != nil {
			fmt.Printf("Error loading document: %v\n", err)
			return
		}

		// Track the view
		if username := os.Getenv("USER"); username != "" {
			_ = corpus.TrackUserView(context.Background(), username, doc.FilePath, cfg)
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
		fmt.Print("\n---\n\n")
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

		// Get all document paths
		docPaths, err := corpus.List(context.Background(), cfg)
		if err != nil {
			fmt.Printf("Error listing documents: %v\n", err)
			return
		}

		// Find the document by title
		var targetDoc *corpus.CorpusDoc
		for _, path := range docPaths {
			doc, err := corpus.Load(context.Background(), path)
			if err != nil {
				continue
			}
			if strings.EqualFold(doc.Title, title) {
				targetDoc = doc
				break
			}
		}

		if targetDoc == nil {
			fmt.Printf("Document '%s' not found\n", title)
			return
		}

		// Delete the document
		err = corpus.Delete(context.Background(), targetDoc.FilePath)
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
		graph, err := corpus.BuildGraph(context.Background(), cfg)
		if err != nil {
			fmt.Printf("Error building graph: %v\n", err)
			return
		}

		// Display the graph
		fmt.Printf("Corpus Graph - %d documents, %d connections\n\n", len(graph.Nodes), len(graph.Edges))
		for nodeName, references := range graph.Nodes {
			fmt.Printf("Document: %s\n", nodeName)

			// Show outgoing references
			if len(references) > 0 {
				fmt.Println("Links to:")
				for _, ref := range references {
					fmt.Printf("  → %s\n", ref)
				}
			}

			// Show incoming references (backlinks)
			if backlinks, exists := graph.Backlinks[nodeName]; exists && len(backlinks) > 0 {
				fmt.Println("Linked from:")
				for _, backlink := range backlinks {
					fmt.Printf("  ← %s\n", backlink)
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
		configDir := filepath.Join(".campaign")
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
	_, err := getCorpusConfig()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get corpus configuration").
			WithComponent("cli").
			WithOperation("corpus.runUI")
	}

	// Get the current user
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}

	// Create the model with proper parameters
	// TODO: Create proper corpus manager interface
	// For now, comment out to fix build
	// model := corpus_ui.NewModel(ctx, corpusManager, corpusConfig)

	// Create and run the program
	// p := tea.NewProgram(model, tea.WithAltScreen())
	// if _, err := p.Run(); err != nil {
	// 	return gerror.Wrap(err, gerror.ErrCodeInternal, "error running corpus UI").
	// 		WithComponent("cli").
	// 		WithOperation("corpus.runUI")
	// }
	return gerror.New(gerror.ErrCodeInternal, "corpus UI temporarily disabled during build fixes", nil).
		WithComponent("cli").
		WithOperation("corpus.runUI")
}

// getCorpusConfig returns the configuration for the corpus system
// This is kept for backward compatibility and will use project-aware config
func getCorpusConfig() (corpus.Config, error) {
	ctx := context.Background()
	return corpus.GetConfigWithFallback(ctx)
}

// getCorpusConfigWithFlags returns the configuration based on command flags
func getCorpusConfigWithFlags(cmd *cobra.Command) (corpus.Config, error) {
	ctx := context.Background()
	useGlobal, _ := cmd.Flags().GetBool("global")

	if useGlobal {
		return corpus.GetGlobalConfig()
	}

	cfg, err := corpus.GetConfigWithFallback(ctx)
	if err != nil {
		return cfg, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get corpus config").
			WithComponent("cli").
			WithOperation("corpus.getCorpusConfig").
			WithDetails("help", "Run 'guild init' to initialize a project")
	}

	return cfg, nil
}

// setupCorpusFlags adds common flags to the corpus commands
func setupCorpusFlags() {
	// Common global flag for all corpus commands
	for _, cmd := range []*cobra.Command{
		corpusCreateCmd, corpusListCmd, corpusViewCmd, corpusDeleteCmd,
		corpusGraphCmd, corpusUICmd, corpusConfigCmd,
	} {
		cmd.Flags().Bool("global", false, "Use global corpus instead of project-local")
	}

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
