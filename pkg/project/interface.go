package project

import "context"

// Detector provides project detection capabilities
type Detector interface {
	// FindProjectRoot finds the project root from a starting path
	FindProjectRoot(startPath string) (string, error)
	// IsInitialized checks if a project exists at the given path
	IsInitialized(path string) bool
}

// ContextProvider provides project context
type ContextProvider interface {
	// GetContext returns project context from current directory
	GetContext() (*Context, error)
	// GetContextFromPath returns project context from a specific path
	GetContextFromPath(path string) (*Context, error)
	// Load loads a project context from the given path with context support
	Load(ctx context.Context, path string) (*Context, error)
	// GetContextWithFallback attempts to get project context with a fallback
	GetContextWithFallback(ctx context.Context, defaultPath string) (*Context, error)
}

// Initializer handles project initialization
type Initializer interface {
	// Initialize creates a new project with options and returns context
	Initialize(ctx context.Context, path string, opts InitOptions) (*Context, error)
	// InitializeWithConfig creates a new project with custom configuration
	InitializeWithConfig(path string, config interface{}) error
}

// Migrator handles project migration
type Migrator interface {
	// MigrateFromGlobal migrates from global to project-local
	MigrateFromGlobal(ctx context.Context, projectPath string, globalPath string, opts MigrationOptions) (*MigrationResult, error)
}

// Manager combines all project management interfaces
type Manager interface {
	Detector
	ContextProvider
	Initializer
	Migrator
}

// defaultManager implements the Manager interface
type defaultManager struct{}

// NewManager creates a new project manager
func NewManager() Manager {
	return &defaultManager{}
}

// FindProjectRoot implements Detector
func (m *defaultManager) FindProjectRoot(startPath string) (string, error) {
	return FindProjectRoot(startPath)
}

// IsInitialized implements Detector
func (m *defaultManager) IsInitialized(path string) bool {
	return IsInitialized(path)
}

// GetContext implements ContextProvider
func (m *defaultManager) GetContext() (*Context, error) {
	return GetContext()
}

// GetContextFromPath implements ContextProvider
func (m *defaultManager) GetContextFromPath(path string) (*Context, error) {
	return GetContextFromPath(path)
}

// Load implements ContextProvider
func (m *defaultManager) Load(ctx context.Context, path string) (*Context, error) {
	return Load(ctx, path)
}

// GetContextWithFallback implements ContextProvider
func (m *defaultManager) GetContextWithFallback(ctx context.Context, defaultPath string) (*Context, error) {
	return GetContextWithFallback(ctx, defaultPath)
}

// Initialize implements Initializer
func (m *defaultManager) Initialize(ctx context.Context, path string, opts InitOptions) (*Context, error) {
	return Initialize(ctx, path, opts)
}

// InitializeWithConfig implements Initializer
func (m *defaultManager) InitializeWithConfig(path string, config interface{}) error {
	return InitializeWithConfig(path, config)
}

// MigrateFromGlobal implements Migrator
func (m *defaultManager) MigrateFromGlobal(ctx context.Context, projectPath string, globalPath string, opts MigrationOptions) (*MigrationResult, error) {
	return MigrateFromGlobal(ctx, projectPath, globalPath, opts)
}
