package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display Guild project information",
	Long: `Shows information about the current Guild project including:
- Project location
- Corpus statistics
- Embedding status
- Active configurations`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Try to get project context
	projCtx, err := project.GetContext()
	if err != nil {
		if err == project.ErrNotInProject {
			fmt.Println("Not in a Guild project")
			fmt.Println("\nTo initialize a project: guild init")
			fmt.Println("To use global Guild: add --global flag to commands")
			return nil
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("cli").
			WithOperation("info.run")
	}
	
	fmt.Println("=== Guild Project Info ===")
	fmt.Printf("Project Root: %s\n", projCtx.GetRootPath())
	fmt.Printf("Guild Path: %s\n", projCtx.GetGuildPath())
	fmt.Println()
	
	// Get corpus info
	cfg, err := corpus.GetProjectConfig(ctx)
	if err != nil {
		fmt.Printf("Error getting corpus config: %v\n", err)
	} else {
		fmt.Println("Corpus:")
		displayCorpusInfo(ctx, cfg)
	}
	
	// Get embeddings info
	fmt.Println("\nEmbeddings:")
	displayEmbeddingsInfo(projCtx.GetEmbeddingsPath())
	
	// Get agents info
	fmt.Println("\nAgents:")
	displayDirectoryInfo(projCtx.GetAgentsPath(), "*.yaml")
	
	// Get objectives info
	fmt.Println("\nObjectives:")
	displayDirectoryInfo(projCtx.GetObjectivesPath(), "*.md")
	
	// Display config info
	fmt.Println("\nConfiguration:")
	if _, err := os.Stat(projCtx.GetConfigPath()); err == nil {
		fmt.Printf("  Config file: %s\n", filepath.Base(projCtx.GetConfigPath()))
		// Could parse and display key settings here
	} else {
		fmt.Println("  No custom configuration (using defaults)")
	}
	
	return nil
}

func displayCorpusInfo(ctx context.Context, cfg corpus.Config) {
	// Count documents
	docs, err := corpus.List(ctx, cfg)
	if err != nil {
		fmt.Printf("  Error listing documents: %v\n", err)
		return
	}
	
	fmt.Printf("  Documents: %d\n", len(docs))
	fmt.Printf("  Path: %s\n", cfg.CorpusPath)
	
	// Calculate total size
	var totalSize int64
	for _, docPath := range docs {
		if info, err := os.Stat(docPath); err == nil {
			totalSize += info.Size()
		}
	}
	
	fmt.Printf("  Total size: %s\n", formatBytes(totalSize))
	fmt.Printf("  Max size: %s\n", formatBytes(cfg.MaxSizeBytes))
}

func displayEmbeddingsInfo(embeddingsPath string) {
	// Check if embeddings exist
	if info, err := os.Stat(embeddingsPath); err == nil && info.IsDir() {
		// Count files and calculate size
		var fileCount int
		var totalSize int64
		
		filepath.Walk(embeddingsPath, func(path string, info fs.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				fileCount++
				totalSize += info.Size()
			}
			return nil
		})
		
		if fileCount > 0 {
			fmt.Printf("  Status: Initialized\n")
			fmt.Printf("  Files: %d\n", fileCount)
			fmt.Printf("  Size: %s\n", formatBytes(totalSize))
		} else {
			fmt.Printf("  Status: Empty (run 'guild corpus scan' to generate)\n")
		}
	} else {
		fmt.Printf("  Status: Not initialized\n")
	}
}

func displayDirectoryInfo(dirPath string, pattern string) {
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		matches, _ := filepath.Glob(filepath.Join(dirPath, pattern))
		if len(matches) > 0 {
			fmt.Printf("  Count: %d\n", len(matches))
			// Show first few items
			for i, match := range matches {
				if i >= 3 {
					fmt.Printf("  ... and %d more\n", len(matches)-3)
					break
				}
				fmt.Printf("  - %s\n", filepath.Base(match))
			}
		} else {
			fmt.Printf("  None found\n")
		}
	} else {
		fmt.Printf("  Directory not found\n")
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}