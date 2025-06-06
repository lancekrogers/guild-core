package corpus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/internal/project"
)

// GetProjectConfig returns corpus configuration for the current project context
func GetProjectConfig(ctx context.Context) (Config, error) {
	// Try to get project context from context.Context
	if projCtx, ok := project.FromContext(ctx); ok {
		return Config{
			CorpusPath:      projCtx.GetCorpusPath(),
			ActivitiesPath:  filepath.Join(projCtx.GetCorpusPath(), ".activities"),
			MaxSizeBytes:    100 * 1024 * 1024, // 100MB default for projects
			DefaultTags:     []string{},
			DefaultCategory: "general",
			Location:        projCtx.GetCorpusPath(),
			MaxSizeMB:       100,
		}, nil
	}

	// Try to get project context from current directory
	projCtx, err := project.GetContext()
	if err != nil {
		// Fallback to global config for backward compatibility
		return GetGlobalConfig()
	}

	return Config{
		CorpusPath:      projCtx.GetCorpusPath(),
		ActivitiesPath:  filepath.Join(projCtx.GetCorpusPath(), ".activities"),
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB default for projects
		DefaultTags:     []string{},
		DefaultCategory: "general",
		Location:        projCtx.GetCorpusPath(),
		MaxSizeMB:       100,
	}, nil
}

// GetGlobalConfig returns the global corpus configuration (for backward compatibility)
func GetGlobalConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to get home directory: %w", err)
	}

	corpusPath := filepath.Join(home, ".guild", "corpus")
	return Config{
		CorpusPath:      corpusPath,
		ActivitiesPath:  filepath.Join(corpusPath, ".activities"),
		MaxSizeBytes:    10 * 1024 * 1024, // 10MB for global
		DefaultTags:     []string{},
		DefaultCategory: "general",
		Location:        corpusPath,
		MaxSizeMB:       10,
	}, nil
}

// GetConfigWithFallback attempts to get project config, falling back to global if needed
func GetConfigWithFallback(ctx context.Context) (Config, error) {
	// First try project config
	cfg, err := GetProjectConfig(ctx)
	if err == nil {
		return cfg, nil
	}

	// If no project found, use global config
	if err == project.ErrNotInProject {
		return GetGlobalConfig()
	}

	return cfg, err
}