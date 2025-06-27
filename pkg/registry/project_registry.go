// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/project"
)

// DefaultProjectRegistry is the default implementation of ProjectRegistry
type DefaultProjectRegistry struct {
	manager ProjectManager
	mu      sync.RWMutex
}

// NewProjectRegistry creates a new project registry
func NewProjectRegistry() ProjectRegistry {
	return &DefaultProjectRegistry{
		// Initialize with default project manager
		manager: &projectManagerAdapter{
			manager: project.NewManager(),
		},
	}
}

// GetProjectManager returns the project manager instance
func (r *DefaultProjectRegistry) GetProjectManager() ProjectManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.manager
}

// SetProjectManager sets the project manager instance
func (r *DefaultProjectRegistry) SetProjectManager(manager ProjectManager) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if manager == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "project manager cannot be nil", nil).
			WithComponent("registry").
			WithOperation("SetProjectManager")
	}

	r.manager = manager
	return nil
}

// GetCurrentContext returns the current project context
func (r *DefaultProjectRegistry) GetCurrentContext(ctx context.Context) (*ProjectContext, error) {
	r.mu.RLock()
	manager := r.manager
	r.mu.RUnlock()

	if manager == nil {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "no project manager configured", nil).
			WithComponent("registry").
			WithOperation("GetCurrentContext")
	}

	projCtx, err := manager.GetContext()
	if err != nil {
		return nil, err
	}

	// The manager already returns a *ProjectContext
	return projCtx, nil
}

// WithProjectContext returns a new context with project information
func (r *DefaultProjectRegistry) WithProjectContext(ctx context.Context) (context.Context, error) {
	projCtx, err := r.GetCurrentContext(ctx)
	if err != nil {
		// If no project context found, return original context
		// This allows the system to work without a project
		return ctx, nil
	}

	// Store in context using a registry-specific key
	return context.WithValue(ctx, projectRegistryContextKey, projCtx), nil
}

// Context key for project context storage
type projectRegistryKey string

const projectRegistryContextKey projectRegistryKey = "guild.registry.project.context"

// GetProjectContextFromContext retrieves project context from a context.Context
func GetProjectContextFromContext(ctx context.Context) (*ProjectContext, bool) {
	projCtx, ok := ctx.Value(projectRegistryContextKey).(*ProjectContext)
	return projCtx, ok
}

// projectManagerAdapter adapts project.Manager to registry.ProjectManager
type projectManagerAdapter struct {
	manager project.Manager
}

func (a *projectManagerAdapter) FindProjectRoot(startPath string) (string, error) {
	return a.manager.FindProjectRoot(startPath)
}

func (a *projectManagerAdapter) IsInitialized(path string) bool {
	return a.manager.IsInitialized(path)
}

func (a *projectManagerAdapter) GetContext() (*ProjectContext, error) {
	ctx, err := a.manager.GetContext()
	if err != nil {
		return nil, err
	}
	adapted := projectContextAdapter{ctx: ctx}
	var result ProjectContext = adapted
	return &result, nil
}

func (a *projectManagerAdapter) GetContextFromPath(path string) (*ProjectContext, error) {
	ctx, err := a.manager.GetContextFromPath(path)
	if err != nil {
		return nil, err
	}
	adapted := projectContextAdapter{ctx: ctx}
	var result ProjectContext = adapted
	return &result, nil
}

func (a *projectManagerAdapter) Initialize(path string) error {
	ctx := context.Background()
	_, err := a.manager.Initialize(ctx, path, project.InitOptions{})
	return err
}

// projectContextAdapter adapts project.Context to registry.ProjectContext
type projectContextAdapter struct {
	ctx *project.Context
}

func (a projectContextAdapter) GetRootPath() string {
	return a.ctx.GetRootPath()
}

func (a projectContextAdapter) GetGuildPath() string {
	return a.ctx.GetGuildPath()
}

func (a projectContextAdapter) GetCorpusPath() string {
	return a.ctx.GetCorpusPath()
}

func (a projectContextAdapter) GetEmbeddingsPath() string {
	return a.ctx.GetEmbeddingsPath()
}

func (a projectContextAdapter) GetConfigPath() string {
	return a.ctx.GetConfigPath()
}

func (a projectContextAdapter) GetAgentsPath() string {
	return a.ctx.GetAgentsPath()
}

func (a projectContextAdapter) GetCommissionsPath() string {
	return a.ctx.GetCommissionsPath()
}
