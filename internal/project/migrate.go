package project

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
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
		return nil, gerror.New(gerror.InvalidArgument, "project", "migrate_from_global", "project not initialized at %s", projectPath)
	}

	// Get project context
	projCtx, err := NewContext(projectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.Internal, "project", "migrate_from_global", "failed to get project context")
	}

	// Migrate corpus documents
	if err := migrateCorpus(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, gerror.Wrap(err, gerror.Internal, "project", "migrate_from_global", "failed to migrate corpus")
	}

	// Migrate embeddings if requested
	if opts.IncludeEmbeddings {
		if err := migrateEmbeddings(ctx, globalPath, projCtx, opts, result); err != nil {
			return result, gerror.Wrap(err, gerror.Internal, "project", "migrate_from_global", "failed to migrate embeddings")
		}
	}

	// Migrate agent configurations
	if err := migrateAgents(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, gerror.Wrap(err, gerror.Internal, "project", "migrate_from_global", "failed to migrate agents")
	}

	// Migrate objectives
	if err := migrateObjectives(ctx, globalPath, projCtx, opts, result); err != nil {
		return result, gerror.Wrap(err, gerror.Internal, "project", "migrate_from_global", "failed to migrate objectives")
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
			result.Errors = append(result.Errors, gerror.Wrap(err, gerror.Internal, "project", "migrate_directory", "error accessing %s", path))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			result.Errors = append(result.Errors, gerror.Wrap(err, gerror.Internal, "project", "migrate_directory", "failed to get relative path for %s", path))
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
			result.Errors = append(result.Errors, gerror.Wrap(err, gerror.Internal, "project", "migrate_directory", "failed to create directory %s", dstDir))
			return nil
		}

		// Copy file
		if err := copyFile(path, dstPath); err != nil {
			result.Errors = append(result.Errors, gerror.Wrap(err, gerror.Internal, "project", "migrate_directory", "failed to copy %s to %s", path, dstPath))
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
		return "", gerror.Wrap(err, gerror.Internal, "project", "get_global_guild_path", "failed to get home directory")
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