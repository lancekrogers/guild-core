// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild-core/pkg/corpus"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/memory/rag"
	"github.com/lancekrogers/guild-core/pkg/memory/vector"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// corpusScanCmd represents the corpus scan command
var corpusScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan corpus and update RAG embeddings",
	Long: `Scans the corpus directory for new or modified documents and updates the RAG system embeddings.

This command is used to synchronize human-added documents in the corpus directory with the
RAG (Retrieval-Augmented Generation) system. It detects new files, modified files, and
deleted files, updating the vector embeddings accordingly.

The scan command ensures that all human-curated knowledge in the corpus is available
to AI agents through the RAG system's semantic search capabilities.`,
	Run: runCorpusScan,
}

// ScanResult holds the results of a corpus scan
type ScanResult struct {
	NewFiles      []string
	ModifiedFiles []string
	DeletedFiles  []string
	Errors        []error
	StartTime     time.Time
	EndTime       time.Time
}

func runCorpusScan(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Get flags
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	forceRebuild, _ := cmd.Flags().GetBool("force")
	providerType, _ := cmd.Flags().GetString("provider")
	embeddingModel, _ := cmd.Flags().GetString("model")
	useGlobal, _ := cmd.Flags().GetBool("global")

	// Get corpus configuration
	var cfg corpus.Config
	var err error

	if useGlobal {
		// Use global config when --global flag is set
		cfg, err = corpus.GetGlobalConfig()
	} else {
		// Try project config first, fall back to global
		cfg, err = corpus.GetConfigWithFallback(ctx)
	}

	if err != nil {
		fmt.Printf("Error getting corpus configuration: %v\n", err)
		fmt.Println("Run 'guild init' to initialize a project")
		return
	}

	if verbose {
		fmt.Printf("Corpus path: %s\n", cfg.CorpusPath)
		fmt.Printf("Scanning for changes...\n\n")
	}

	// Initialize RAG system
	ragSystem, err := initializeRAGSystem(ctx, cfg, providerType, embeddingModel, verbose)
	if err != nil {
		fmt.Printf("Error initializing RAG system: %v\n", err)
		return
	}

	// Perform the scan
	result := performCorpusScan(ctx, cfg, ragSystem, dryRun, verbose, forceRebuild)

	// Display results
	displayScanResults(result, dryRun)
}

func initializeRAGSystem(ctx context.Context, cfg corpus.Config, providerType, embeddingModel string, verbose bool) (*rag.Retriever, error) {
	// Create provider based on type or auto-detect
	var provider interfaces.AIProvider
	var err error

	factory := providers.NewFactoryV2()

	if providerType != "" {
		// Use specified provider
		var pType providers.ProviderType
		switch strings.ToLower(providerType) {
		case "ollama":
			pType = providers.ProviderOllama
		case "openai":
			pType = providers.ProviderOpenAI
		case "anthropic":
			pType = providers.ProviderAnthropic
		default:
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported provider type", nil).
				WithComponent("cli").
				WithOperation("initializeRAGSystem").
				WithDetails("provider_type", providerType)
		}

		// Get API key or base URL from environment
		apiKey := ""
		if pType == providers.ProviderOllama {
			apiKey = os.Getenv("OLLAMA_HOST")
			if apiKey == "" {
				apiKey = "http://localhost:11434"
			}
		} else {
			envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerType))
			apiKey = os.Getenv(envKey)
			if apiKey == "" {
				return nil, gerror.New(gerror.ErrCodeMissingRequired, "missing API key", nil).
					WithComponent("cli").
					WithOperation("initializeRAGSystem").
					WithDetails("env_key", envKey)
			}
		}

		provider, err = factory.CreateAIProvider(pType, apiKey)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider").
				WithComponent("cli").
				WithOperation("initializeRAGSystem").
				WithDetails("provider_type", providerType)
		}
	} else {
		// Auto-detect will be handled by vector factory
		if verbose {
			fmt.Println("Auto-detecting available AI provider...")
		}
	}

	// Create vector store configuration
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: provider,
		EmbeddingModel:    embeddingModel,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   filepath.Join(cfg.CorpusPath, "..", "embeddings"),
			DefaultCollection: "corpus",
		},
	}

	// Create vector store
	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create vector store").
			WithComponent("cli").
			WithOperation("initializeRAGSystem")
	}

	// Create RAG configuration
	ragConfig := rag.Config{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		MaxResults:   10,
		UseCorpus:    true,
		CorpusPath:   cfg.CorpusPath,
	}

	// Create retriever with existing vector store
	retriever := rag.NewRetrieverWithStore(vectorStore, ragConfig)

	return retriever, nil
}

func performCorpusScan(ctx context.Context, cfg corpus.Config, ragSystem *rag.Retriever, dryRun, verbose, forceRebuild bool) ScanResult {
	result := ScanResult{
		StartTime: time.Now(),
	}

	// Get current corpus documents
	corpusFilePaths, err := corpus.List(ctx, cfg)
	if err != nil {
		result.Errors = append(result.Errors, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list corpus documents").
			WithComponent("cli").
			WithOperation("performCorpusScan"))
		result.EndTime = time.Now()
		return result
	}

	// Build a map of file paths to modification times
	corpusFiles := make(map[string]time.Time)
	for _, filePath := range corpusFilePaths {
		info, err := os.Stat(filePath)
		if err == nil {
			corpusFiles[filePath] = info.ModTime()
		}
	}

	// Get embeddings metadata to track what's already indexed
	embeddingsMeta := getEmbeddingsMetadata(cfg)

	// Check each corpus file
	for _, filePath := range corpusFilePaths {
		needsUpdate := false
		modTime := corpusFiles[filePath]

		if forceRebuild {
			needsUpdate = true
			// When force rebuilding, treat existing files as modified
			if _, exists := embeddingsMeta[filePath]; exists {
				result.ModifiedFiles = append(result.ModifiedFiles, filePath)
			} else {
				result.NewFiles = append(result.NewFiles, filePath)
			}
			if verbose {
				fmt.Printf("Force rebuilding: %s\n", filepath.Base(filePath))
			}
		} else {
			// Check if file is new or modified
			if lastIndexed, exists := embeddingsMeta[filePath]; !exists {
				result.NewFiles = append(result.NewFiles, filePath)
				needsUpdate = true
				if verbose {
					fmt.Printf("New file: %s\n", filepath.Base(filePath))
				}
			} else if modTime.After(lastIndexed) {
				result.ModifiedFiles = append(result.ModifiedFiles, filePath)
				needsUpdate = true
				if verbose {
					fmt.Printf("Modified file: %s\n", filepath.Base(filePath))
				}
			}
		}

		if needsUpdate && !dryRun {
			// Load full document content
			fullDoc, err := corpus.Load(ctx, filePath)
			if err != nil {
				result.Errors = append(result.Errors, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load corpus document").
					WithComponent("cli").
					WithOperation("performCorpusScan").
					WithDetails("file_path", filePath))
				continue
			}

			// Add to RAG system
			if err := addDocumentToRAG(ctx, ragSystem, fullDoc); err != nil {
				result.Errors = append(result.Errors, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add document to RAG").
					WithComponent("cli").
					WithOperation("performCorpusScan").
					WithDetails("file_path", filePath))
				continue
			}

			// Update metadata
			updateEmbeddingMetadata(cfg, filePath, time.Now())
		}
	}

	// Check for deleted files
	for filePath := range embeddingsMeta {
		if _, exists := corpusFiles[filePath]; !exists {
			result.DeletedFiles = append(result.DeletedFiles, filePath)
			if verbose {
				fmt.Printf("Deleted file: %s\n", filePath)
			}

			if !dryRun {
				// Remove from RAG system
				if err := removeDocumentFromRAG(ctx, ragSystem, filePath); err != nil {
					result.Errors = append(result.Errors, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to remove document from RAG").
						WithComponent("cli").
						WithOperation("performCorpusScan").
						WithDetails("file_path", filePath))
				}
			}
		}
	}

	result.EndTime = time.Now()
	return result
}

func addDocumentToRAG(ctx context.Context, ragSystem *rag.Retriever, doc *corpus.CorpusDoc) error {
	// Use the AddCorpusDocument method which is designed for corpus documents
	return ragSystem.AddCorpusDocument(ctx, doc)
}

func removeDocumentFromRAG(ctx context.Context, ragSystem *rag.Retriever, filePath string) error {
	// Remove all chunks associated with this document
	return ragSystem.RemoveDocument(ctx, filePath)
}

func getEmbeddingsMetadata(cfg corpus.Config) map[string]time.Time {
	// Read metadata file that tracks when each document was last indexed
	metadataPath := filepath.Join(cfg.CorpusPath, "..", "embeddings", ".metadata.json")

	metadata := make(map[string]time.Time)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		// File doesn't exist yet, return empty map
		return metadata
	}

	// Parse JSON metadata
	// Format: {"file_path": "2024-01-01T00:00:00Z", ...}
	// This is a simplified version - in production, use proper JSON parsing
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				metadata[parts[0]] = t
			}
		}
	}

	return metadata
}

func updateEmbeddingMetadata(cfg corpus.Config, filePath string, indexTime time.Time) error {
	metadataPath := filepath.Join(cfg.CorpusPath, "..", "embeddings", ".metadata.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return err
	}

	// Read existing metadata
	metadata := getEmbeddingsMetadata(cfg)

	// Update entry
	metadata[filePath] = indexTime

	// Write back
	// This is a simplified version - in production, use proper JSON encoding
	var lines []string
	for path, t := range metadata {
		lines = append(lines, fmt.Sprintf("%s\t%s", path, t.Format(time.RFC3339)))
	}

	return os.WriteFile(metadataPath, []byte(strings.Join(lines, "\n")), 0o644)
}

func displayScanResults(result ScanResult, dryRun bool) {
	duration := result.EndTime.Sub(result.StartTime)

	fmt.Println("\n=== Corpus Scan Results ===")
	fmt.Printf("Scan duration: %v\n", duration)

	if dryRun {
		fmt.Println("\n[DRY RUN - No changes were made]")
	}

	if len(result.NewFiles) > 0 {
		fmt.Printf("\nNew files (%d):\n", len(result.NewFiles))
		for _, f := range result.NewFiles {
			fmt.Printf("  + %s\n", filepath.Base(f))
		}
	}

	if len(result.ModifiedFiles) > 0 {
		fmt.Printf("\nModified files (%d):\n", len(result.ModifiedFiles))
		for _, f := range result.ModifiedFiles {
			fmt.Printf("  ~ %s\n", filepath.Base(f))
		}
	}

	if len(result.DeletedFiles) > 0 {
		fmt.Printf("\nDeleted files (%d):\n", len(result.DeletedFiles))
		for _, f := range result.DeletedFiles {
			fmt.Printf("  - %s\n", filepath.Base(f))
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  ! %v\n", err)
		}
	}

	total := len(result.NewFiles) + len(result.ModifiedFiles) + len(result.DeletedFiles)
	if total == 0 && len(result.Errors) == 0 {
		fmt.Println("\nNo changes detected. Corpus and RAG system are in sync.")
	} else if !dryRun {
		fmt.Printf("\nProcessed %d changes.\n", total)
	}
}

func init() {
	// Add scan command to corpus
	corpusCmd.AddCommand(corpusScanCmd)

	// Add flags
	corpusScanCmd.Flags().BoolP("dry-run", "n", false, "Show what would be done without making changes")
	corpusScanCmd.Flags().BoolP("verbose", "v", false, "Show detailed progress")
	corpusScanCmd.Flags().BoolP("force", "f", false, "Force rebuild all embeddings")
	corpusScanCmd.Flags().StringP("provider", "p", "", "AI provider to use (ollama, openai, anthropic)")
	corpusScanCmd.Flags().StringP("model", "m", "", "Embedding model to use (e.g., nomic-embed-text)")
	corpusScanCmd.Flags().Bool("global", false, "Use global corpus instead of project-local")
}
