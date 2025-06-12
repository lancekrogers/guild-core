// Package project implements Guild's dual-directory architecture for the registry pattern.
//
// Guild uses a sophisticated initialization system that separates:
// - Global resources (~/.guild/) - shared tools, providers, templates, LSP servers
// - Local resources (.guild/) - project-specific data, database, corpus, workspaces
//
// This design enables:
// - Component reusability across projects (registry pattern)
// - Project isolation and portability (like .git directories)
// - Clean separation between framework (global) and application (local) concerns
// - Future splitting of Guild into framework library and CLI tool
//
// The Initialize() function provides a convenient API that handles both global
// and local initialization while returning a project context for immediate use.
package project

import (
	"context"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project/global"
	"github.com/guild-ventures/guild-core/pkg/project/local"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// InitializeProject creates both global and local Guild structures
func InitializeProject(projectPath string) error {
	// Ensure global Guild directory exists
	if err := global.InitializeGlobal(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize global Guild").
			WithComponent("project").
			WithOperation("initialize_project")
	}

	// Initialize local project directory
	if err := local.InitializeLocal(projectPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize local Guild").
			WithComponent("project").
			WithOperation("initialize_project")
	}

	// Initialize the database
	dbPath := local.LocalDatabasePath(projectPath)
	if err := initializeDatabaseRefactored(dbPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize database").
			WithComponent("project").
			WithOperation("initialize_project")
	}

	return nil
}

// IsProjectInitialized checks if a project is initialized
func IsProjectInitialized(projectPath string) bool {
	localDir := local.LocalGuildDir(projectPath)
	configPath := filepath.Join(localDir, "guild.yaml")
	dbPath := filepath.Join(localDir, "memory.db")

	// Check both config and database exist
	if _, err := os.Stat(configPath); err != nil {
		return false
	}
	if _, err := os.Stat(dbPath); err != nil {
		return false
	}

	return true
}

// initializeDatabaseRefactored creates and migrates the SQLite database
func initializeDatabaseRefactored(dbPath string) error {
	ctx := context.Background()
	
	// Create database connection
	db, err := storage.DefaultDatabaseFactory(ctx, dbPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create database").
			WithComponent("project").
			WithOperation("initialize_database")
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run migrations").
			WithComponent("project").
			WithOperation("initialize_database")
	}

	return nil
}

// GetProjectStructure returns paths to all project directories
func GetProjectStructure(projectPath string) *ProjectStructure {
	localDir := local.LocalGuildDir(projectPath)
	globalDir := global.GlobalGuildDir()

	return &ProjectStructure{
		// Global paths
		GlobalDir:      globalDir,
		GlobalConfig:   global.GlobalConfigPath(),
		ProvidersDir:   filepath.Join(globalDir, "providers"),
		ToolsDir:       filepath.Join(globalDir, "tools"),
		TemplatesDir:   filepath.Join(globalDir, "templates"),
		GlobalCacheDir: filepath.Join(globalDir, "cache"),
		GlobalLogsDir:  filepath.Join(globalDir, "logs"),
		LSPServersDir:  filepath.Join(globalDir, "lsp"),

		// Local paths
		LocalDir:       localDir,
		LocalConfig:    local.LocalConfigPath(projectPath),
		DatabasePath:   local.LocalDatabasePath(projectPath),
		CorpusDir:      local.LocalCorpusPath(projectPath),
		CommissionsDir: local.LocalCommissionsPath(projectPath),
		WorkspacesDir:  local.LocalWorkspacesPath(projectPath),
		CampaignsDir:   filepath.Join(localDir, "campaigns"),
		KanbanDir:      filepath.Join(localDir, "kanban"),
		PromptsDir:     filepath.Join(localDir, "prompts"),
		LocalToolsDir:  local.LocalToolsPath(projectPath),
		// ArchivesDir: filepath.Join(localDir, "archives"), // TODO: pending ChromemGo deletion
	}
}

// ProjectStructure contains all paths for a Guild project
type ProjectStructure struct {
	// Global paths (~/.guild/)
	GlobalDir      string
	GlobalConfig   string
	ProvidersDir   string
	ToolsDir       string
	TemplatesDir   string
	GlobalCacheDir string
	GlobalLogsDir  string
	LSPServersDir  string

	// Local paths (.guild/)
	LocalDir       string
	LocalConfig    string
	DatabasePath   string
	CorpusDir      string
	CommissionsDir string // User objectives/goals
	WorkspacesDir  string
	CampaignsDir   string
	KanbanDir      string
	PromptsDir     string
	LocalToolsDir  string // Project-specific tools
	// ArchivesDir string // TODO: Agent memory (pending ChromemGo deletion)
}