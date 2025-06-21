// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package project provides project-local Guild management functionality.
// It handles detection, initialization, and context management for Guild projects.
package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

var (
	// ErrNotInProject indicates the current directory is not within a Guild project
	ErrNotInProject = gerror.New(gerror.ErrCodeNotFound, "not in a guild project - run 'guild init' to create one", nil).
			WithComponent("project").
			WithOperation("validate")
	// ErrAlreadyInitialized indicates a project is already initialized at the given path
	ErrAlreadyInitialized = gerror.New(gerror.ErrCodeAlreadyExists, "project already initialized", nil).
				WithComponent("project").
				WithOperation("initialize")
	// ErrInvalidPath indicates the provided path is invalid
	ErrInvalidPath = gerror.New(gerror.ErrCodeInvalidInput, "invalid project path", nil).
			WithComponent("project").
			WithOperation("validate")
)

// Context represents a Guild project's context with paths and configuration.
// It is immutable after creation to ensure thread safety.
type Context struct {
	rootPath        string // Project root (where .campaign exists)
	guildPath       string // .campaign directory
	corpusPath      string // .campaign/corpus
	embeddingsPath  string // .campaign/embeddings
	configPath      string // .campaign/config.yaml
	agentsPath      string // .campaign/agents
	commissionsPath string // .campaign/commissions
}

// NewContext creates a new project context from a root path
func NewContext(rootPath string) (*Context, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve absolute path").
			WithComponent("project").
			WithOperation("new_context")
	}

	guildPath := filepath.Join(abs, paths.DefaultCampaignDir)

	return &Context{
		rootPath:        abs,
		guildPath:       guildPath,
		corpusPath:      filepath.Join(guildPath, "corpus"),
		embeddingsPath:  filepath.Join(guildPath, "embeddings"),
		configPath:      filepath.Join(guildPath, "config.yaml"),
		agentsPath:      filepath.Join(guildPath, "agents"),
		commissionsPath: filepath.Join(guildPath, "commissions"),
	}, nil
}

// GetRootPath returns the project root directory
func (c *Context) GetRootPath() string {
	return c.rootPath
}

// GetGuildPath returns the .campaign directory path
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

// GetCommissionsPath returns the commissions directory path
func (c *Context) GetCommissionsPath() string {
	return c.commissionsPath
}

// FindProjectRoot walks up the directory tree looking for a campaign directory
func FindProjectRoot(startPath string) (string, error) {
	abs, err := filepath.Abs(startPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve absolute path").
			WithComponent("project").
			WithOperation("find_project_root")
	}

	current := abs
	for {
		guildPath := filepath.Join(current, paths.DefaultCampaignDir)
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
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("project").
			WithOperation("find_nearest_project")
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

// Load loads a project context from the given path with context support
// This is the context-aware version of GetContextFromPath for modern usage patterns
func Load(ctx context.Context, path string) (*Context, error) {
	// For now, we don't use the context, but having it allows for future
	// cancellation support and timeout handling during project loading
	return GetContextFromPath(path)
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
	guildPath := filepath.Join(path, paths.DefaultCampaignDir)
	info, err := os.Stat(guildPath)
	return err == nil && info.IsDir()
}

// ValidateProjectPath ensures the project path is safe and valid
func ValidateProjectPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve absolute path").
			WithComponent("project").
			WithOperation("validate_project_path")
	}

	// Ensure campaign directory would be within the resolved path
	guildPath := filepath.Join(abs, paths.DefaultCampaignDir)
	if !strings.HasPrefix(guildPath, abs) {
		return ErrInvalidPath
	}

	// Check if the path exists and is a directory
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return gerror.Newf(gerror.ErrCodeNotFound, "path does not exist: %s", abs).
				WithComponent("project").
				WithOperation("validate_project_path")
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stat path").
			WithComponent("project").
			WithOperation("validate_project_path")
	}

	if !info.IsDir() {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "path is not a directory: %s", abs).
			WithComponent("project").
			WithOperation("validate_project_path")
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
