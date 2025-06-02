// Package project provides project-local Guild management functionality.
// It handles detection, initialization, and context management for Guild projects.
package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNotInProject indicates the current directory is not within a Guild project
	ErrNotInProject = errors.New("not in a guild project")
	// ErrAlreadyInitialized indicates a project is already initialized at the given path
	ErrAlreadyInitialized = errors.New("project already initialized")
	// ErrInvalidPath indicates the provided path is invalid
	ErrInvalidPath = errors.New("invalid project path")
)

// Context represents a Guild project's context with paths and configuration.
// It is immutable after creation to ensure thread safety.
type Context struct {
	rootPath       string // Project root (where .guild exists)
	guildPath      string // .guild directory
	corpusPath     string // .guild/corpus
	embeddingsPath string // .guild/embeddings
	configPath     string // .guild/config.yaml
	agentsPath     string // .guild/agents
	objectivesPath string // .guild/objectives
}

// NewContext creates a new project context from a root path
func NewContext(rootPath string) (*Context, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	guildPath := filepath.Join(abs, ".guild")
	
	return &Context{
		rootPath:       abs,
		guildPath:      guildPath,
		corpusPath:     filepath.Join(guildPath, "corpus"),
		embeddingsPath: filepath.Join(guildPath, "embeddings"),
		configPath:     filepath.Join(guildPath, "config.yaml"),
		agentsPath:     filepath.Join(guildPath, "agents"),
		objectivesPath: filepath.Join(guildPath, "objectives"),
	}, nil
}

// GetRootPath returns the project root directory
func (c *Context) GetRootPath() string {
	return c.rootPath
}

// GetGuildPath returns the .guild directory path
func (c *Context) GetGuildPath() string {
	return c.guildPath
}

// GetCorpusPath returns the corpus directory path
func (c *Context) GetCorpusPath() string {
	return c.corpusPath
}

// GetEmbeddingsPath returns the embeddings directory path
func (c *Context) GetEmbeddingsPath() string {
	return c.embeddingsPath
}

// GetConfigPath returns the config file path
func (c *Context) GetConfigPath() string {
	return c.configPath
}

// GetAgentsPath returns the agents directory path
func (c *Context) GetAgentsPath() string {
	return c.agentsPath
}

// GetObjectivesPath returns the objectives directory path
func (c *Context) GetObjectivesPath() string {
	return c.objectivesPath
}

// FindProjectRoot walks up the directory tree looking for a .guild directory
func FindProjectRoot(startPath string) (string, error) {
	abs, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	current := abs
	for {
		guildPath := filepath.Join(current, ".guild")
		if info, err := os.Stat(guildPath); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached the root of the filesystem
			return "", ErrNotInProject
		}
		current = parent
	}
}

// GetContext returns project context from current directory
func GetContext() (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return GetContextFromPath(cwd)
}

// GetContextFromPath returns project context from a specific path
func GetContextFromPath(path string) (*Context, error) {
	rootPath, err := FindProjectRoot(path)
	if err != nil {
		return nil, err
	}

	return NewContext(rootPath)
}

// GetContextWithFallback attempts to get project context, falling back to a default if not in a project
func GetContextWithFallback(ctx context.Context, defaultPath string) (*Context, error) {
	// Try to get project context
	projCtx, err := GetContext()
	if err == nil {
		return projCtx, nil
	}

	// If not in a project, create context from default path
	if errors.Is(err, ErrNotInProject) {
		return NewContext(defaultPath)
	}

	return nil, err
}

// IsInitialized checks if a Guild project exists at the given path
func IsInitialized(path string) bool {
	guildPath := filepath.Join(path, ".guild")
	info, err := os.Stat(guildPath)
	return err == nil && info.IsDir()
}

// ValidateProjectPath ensures the project path is safe and valid
func ValidateProjectPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Ensure .guild would be within the resolved path
	guildPath := filepath.Join(abs, ".guild")
	if !strings.HasPrefix(guildPath, abs) {
		return ErrInvalidPath
	}

	// Check if the path exists and is a directory
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", abs)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", abs)
	}

	return nil
}

// ContextKey is the key type for storing project context in context.Context
type contextKey string

const projectContextKey contextKey = "guild.project.context"

// WithContext adds project context to a context.Context
func WithContext(ctx context.Context, projCtx *Context) context.Context {
	return context.WithValue(ctx, projectContextKey, projCtx)
}

// FromContext retrieves project context from a context.Context
func FromContext(ctx context.Context) (*Context, bool) {
	projCtx, ok := ctx.Value(projectContextKey).(*Context)
	return projCtx, ok
}

// MustFromContext retrieves project context from a context.Context or panics
func MustFromContext(ctx context.Context) *Context {
	projCtx, ok := FromContext(ctx)
	if !ok {
		panic("project context not found in context")
	}
	return projCtx
}