package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MigrationOptions configures the migration behavior
type MigrationOptions struct {
	// IncludeEmbeddings specifies whether to migrate embeddings
	IncludeEmbeddings bool
	// IncludeActivities specifies whether to migrate activity logs
	IncludeActivities bool
	// OverwriteExisting specifies whether to overwrite existing files
	OverwriteExisting bool
	// DryRun performs a dry run without making changes
	DryRun bool
}

// MigrationResult contains the results of a migration
type MigrationResult struct {
	// FilesCopied is the number of files successfully copied
	FilesCopied int
	// FilesSkipped is the number of files skipped
	FilesSkipped int
	// Errors contains any errors encountered
	Errors []error
}

// MigrateFromGlobal migrates data from global Guild configuration to project-local
func MigrateFromGlobal(ctx context.Context, projectPath string, globalPath string, opts MigrationOptions) (*MigrationResult, error) {
	result := &MigrationResult{}

	// Ensure project is initialized
	if !IsInitialized(projectPath) {
		return nil, fmt.Errorf("project not initialized at %s", projectPath)
	}

	// Get project context
	projCtx, err := NewContext(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get project context: %w", err)
	}

	// Migrate corpus documents
	if err := migrateCorpus(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, fmt.Errorf("failed to migrate corpus: %w", err)
	}

	// Migrate embeddings if requested
	if opts.IncludeEmbeddings {
		if err := migrateEmbeddings(ctx, globalPath, projCtx, opts, result); err != nil {
			return result, fmt.Errorf("failed to migrate embeddings: %w", err)
		}
	}

	// Migrate agent configurations
	if err := migrateAgents(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, fmt.Errorf("failed to migrate agents: %w", err)
	}

	// Migrate objectives
	if err := migrateObjectives(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, fmt.Errorf("failed to migrate objectives: %w", err)
	}

	return result, nil
}

// migrateCorpus migrates corpus documents
func migrateCorpus(ctx context.Context, globalPath string, projCtx *Context, opts MigrationOptions, result *MigrationResult) error {
	globalCorpusPath := filepath.Join(globalPath, "corpus", "docs")
	projectCorpusPath := filepath.Join(projCtx.GetCorpusPath(), "docs")

	return migrateDirectory(globalCorpusPath, projectCorpusPath, opts, result)
}

// migrateEmbeddings migrates embeddings
func migrateEmbeddings(ctx context.Context, globalPath string, projCtx *Context, opts MigrationOptions, result *MigrationResult) error {
	globalEmbeddingsPath := filepath.Join(globalPath, "embeddings")
	projectEmbeddingsPath := projCtx.GetEmbeddingsPath()

	return migrateDirectory(globalEmbeddingsPath, projectEmbeddingsPath, opts, result)
}

// migrateAgents migrates agent configurations
func migrateAgents(ctx context.Context, globalPath string, projCtx *Context, opts MigrationOptions, result *MigrationResult) error {
	globalAgentsPath := filepath.Join(globalPath, "agents")
	projectAgentsPath := projCtx.GetAgentsPath()

	return migrateDirectory(globalAgentsPath, projectAgentsPath, opts, result)
}

// migrateObjectives migrates objectives
func migrateObjectives(ctx context.Context, globalPath string, projCtx *Context, opts MigrationOptions, result *MigrationResult) error {
	globalObjectivesPath := filepath.Join(globalPath, "objectives")
	projectObjectivesPath := projCtx.GetObjectivesPath()

	return migrateDirectory(globalObjectivesPath, projectObjectivesPath, opts, result)
}

// migrateDirectory migrates files from source to destination directory
func migrateDirectory(src, dst string, opts MigrationOptions, result *MigrationResult) error {
	// Check if source exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		// Source doesn't exist, nothing to migrate
		return nil
	}

	// Walk source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to get relative path for %s: %w", path, err))
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		// Skip if destination exists and overwrite is not enabled
		if _, err := os.Stat(dstPath); err == nil && !opts.OverwriteExisting {
			result.FilesSkipped++
			return nil
		}

		// Dry run - just count
		if opts.DryRun {
			result.FilesCopied++
			return nil
		}

		// Create destination directory
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to create directory %s: %w", dstDir, err))
			return nil
		}

		// Copy file
		if err := copyFile(path, dstPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to copy %s to %s: %w", path, dstPath, err))
			return nil
		}

		result.FilesCopied++
		return nil
	})
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// GetGlobalGuildPath returns the default global Guild path
func GetGlobalGuildPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".guild"), nil
}

// FormatMigrationSummary formats a migration result for display
func FormatMigrationSummary(result *MigrationResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Migration Summary:\n"))
	sb.WriteString(fmt.Sprintf("  Files copied: %d\n", result.FilesCopied))
	sb.WriteString(fmt.Sprintf("  Files skipped: %d\n", result.FilesSkipped))

	if len(result.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("  Errors: %d\n", len(result.Errors)))
		for i, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("    %d. %v\n", i+1, err))
		}
	}

	return sb.String()
}