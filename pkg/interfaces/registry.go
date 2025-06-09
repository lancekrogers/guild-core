// Package interfaces provides shared interfaces to break circular dependencies
package interfaces

import (
	"context"
)

// ComponentRegistry is the main registry interface for all components
type ComponentRegistry interface {
	// Initialize sets up all registries with the provided configuration
	Initialize(ctx context.Context, config interface{}) error

	// Shutdown cleanly shuts down all registries and their components
	Shutdown(ctx context.Context) error

	// Additional methods will be added as needed
}

// ProjectContext represents a project's context
type ProjectContext interface {
	GetRootPath() string
	GetGuildPath() string
	GetCorpusPath() string
	GetEmbeddingsPath() string
	GetConfigPath() string
	GetAgentsPath() string
	GetObjectivesPath() string
}

// ProjectManager provides project management capabilities
type ProjectManager interface {
	// FindProjectRoot finds the project root from a starting path
	FindProjectRoot(startPath string) (string, error)
	// IsInitialized checks if a project exists at the given path
	IsInitialized(path string) bool
	// GetContext returns project context from current directory
	GetContext() (ProjectContext, error)
	// GetContextFromPath returns project context from a specific path
	GetContextFromPath(path string) (ProjectContext, error)
	// Initialize creates a new project structure
	Initialize(path string) error
}

// ConfigLoader provides configuration loading capabilities
type ConfigLoader interface {
	Load(path string) error
	Get(key string) interface{}
	Set(key string, value interface{})
}

// Agent is the interface for all Guild agents
// This is defined here to break the circular dependency between agent and registry packages
type Agent interface {
	// Execute runs a task
	Execute(ctx context.Context, request string) (string, error)

	// GetID returns the agent's ID
	GetID() string

	// GetName returns the agent's name
	GetName() string

	// GetType returns the agent's type
	GetType() string

	// GetCapabilities returns the agent's capabilities
	GetCapabilities() []string
}
