// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package local

import (
	"os"
	"path/filepath"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/paths"
)

// localDirectoryStructure defines the directory structure for a local Guild project
var localDirectoryStructure = []string{
	"commissions",         // User-defined objectives/goals
	"commissions/refined", // AI-refined versions
	"campaigns",           // Execution plans for commissions
	"kanban",              // Task tracking
	"corpus",              // Project documentation
	"corpus/index",        // Vector store indices
	"prompts",             // Custom prompt templates
	"tools",               // Project-specific tool installations/configs
	"workspaces",          // Temporary agent work areas
	// "archives",        // TODO: Agent memory - awaiting ChromemGo deletion support
}

// LocalGuildDir returns the path to the local Guild directory
func LocalGuildDir(projectPath string) string {
	return filepath.Join(projectPath, paths.DefaultCampaignDir)
}

// InitializeLocal creates the local Guild directory structure for a project
func InitializeLocal(projectPath string) error {
	localDir := LocalGuildDir(projectPath)

	// Check if already initialized
	if _, err := os.Stat(localDir); err == nil {
		// Directory exists, check if it's properly initialized
		configPath := filepath.Join(localDir, "guild.yaml")
		if _, err := os.Stat(configPath); err == nil {
			// Already initialized
			return nil
		}
	}

	// Create local directory
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create local Guild directory").
			WithComponent("project").
			WithOperation("initialize_local").
			WithDetails("path", localDir)
	}

	// Create subdirectories
	for _, dir := range localDirectoryStructure {
		dirPath := filepath.Join(localDir, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create directory %s", dir).
				WithComponent("project").
				WithOperation("initialize_local")
		}
	}

	// Create SQLite database file (touch it)
	dbPath := filepath.Join(localDir, "memory.db")
	if _, err := os.Create(dbPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create database file").
			WithComponent("project").
			WithOperation("initialize_local")
	}

	// Note: We don't create a guild.yaml file here because that should contain
	// the campaign reference and be created by campaign initialization.
	// Local project initialization only creates directory structure and database.

	// Create .gitignore
	if err := createGitIgnore(localDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create .gitignore").
			WithComponent("project").
			WithOperation("initialize_local")
	}

	return nil
}

// createDefaultLocalConfig creates the default local guild configuration
func createDefaultLocalConfig(localDir, projectPath string) error {
	configPath := filepath.Join(localDir, "guild.yaml")

	// Detect project type
	projectType := detectProjectType(projectPath)

	content := generateLocalConfig(projectType)

	return os.WriteFile(configPath, []byte(content), 0o644)
}

// detectProjectType detects the type of project based on files present
func detectProjectType(projectPath string) string {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		return "golang"
	}

	// Check for Python project
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "pyproject.toml")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "setup.py")); err == nil {
		return "python"
	}

	// Check for TypeScript/JavaScript project
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		return "typescript"
	}

	// Check for Rust project
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		return "rust"
	}

	return "generic"
}

// generateLocalConfig generates appropriate config based on project type
func generateLocalConfig(projectType string) string {
	switch projectType {
	case "golang":
		return goProjectConfig
	case "python":
		return pythonProjectConfig
	case "typescript":
		return typescriptProjectConfig
	case "rust":
		return rustProjectConfig
	default:
		return genericProjectConfig
	}
}

// createGitIgnore creates a .gitignore file for the Guild directory
func createGitIgnore(localDir string) error {
	gitignorePath := filepath.Join(localDir, ".gitignore")
	content := `# Guild Framework Files
# Database
memory.db
memory.db-shm
memory.db-wal

# Vector stores
corpus/index/*.bin
corpus/index/*.idx

# Workspaces (temporary agent work)
workspaces/

# Archives (agent memory)
archives/

# Logs
*.log

# API keys and secrets (should never be here anyway)
*.key
*.pem
secrets.yaml
`
	return os.WriteFile(gitignorePath, []byte(content), 0o644)
}

// LocalConfigPath returns the path to the local config file
func LocalConfigPath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "guild.yaml")
}

// LocalDatabasePath returns the path to the local SQLite database
func LocalDatabasePath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "memory.db")
}

// LocalCorpusPath returns the path to the local corpus directory
func LocalCorpusPath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "corpus")
}

// LocalCommissionsPath returns the path to the local commissions directory (objectives/goals)
func LocalCommissionsPath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "commissions")
}

// LocalToolsPath returns the path to the local tools directory
func LocalToolsPath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "tools")
}

// LocalWorkspacesPath returns the path to the local workspaces directory
func LocalWorkspacesPath(projectPath string) string {
	return filepath.Join(LocalGuildDir(projectPath), "workspaces")
}

// EnsureLocalInitialized ensures the local Guild directory is initialized
func EnsureLocalInitialized(projectPath string) error {
	localDir := LocalGuildDir(projectPath)
	configPath := filepath.Join(localDir, "guild.yaml")

	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Initialize if not exists
	return InitializeLocal(projectPath)
}

// Pre-defined project configurations
const goProjectConfig = `# Guild Configuration for Go Project
name: "guild-project"
description: "Go project managed by Guild Framework"
version: "1.0.0"

manager:
  default: "guild-master"

agents:
  - id: "guild-master"
    name: "Guild Master"
    type: "manager"
    provider: "anthropic"
    model: "claude-3-opus-20240229"
    description: "Orchestrates the guild and assigns tasks"
    capabilities: ["task_routing", "planning", "coordination"]
    
  - id: "code-artisan"
    name: "Code Artisan"
    type: "worker"
    provider: "anthropic"
    model: "claude-3-sonnet-20240229"
    description: "Implements Go code with best practices"
    capabilities: ["golang", "testing", "refactoring"]
    tools: ["code", "edit", "lsp", "git"]
    
  - id: "test-scribe"
    name: "Test Scribe"
    type: "specialist"
    provider: "openai"
    model: "gpt-4-turbo-preview"
    description: "Writes comprehensive tests"
    capabilities: ["testing", "benchmarking", "coverage"]
    tools: ["code", "shell"]

storage:
  backend: "sqlite"
  sqlite:
    path: ".guild/memory.db"

metadata:
  tags: ["golang", "backend", "api"]
`

const pythonProjectConfig = `# Guild Configuration for Python Project
name: "guild-project"
description: "Python project managed by Guild Framework"
version: "1.0.0"

manager:
  default: "guild-master"

agents:
  - id: "guild-master"
    name: "Guild Master"
    type: "manager"
    provider: "anthropic"
    model: "claude-3-opus-20240229"
    description: "Orchestrates the guild and assigns tasks"
    capabilities: ["task_routing", "planning", "coordination"]
    
  - id: "python-artisan"
    name: "Python Artisan"
    type: "worker"
    provider: "anthropic"
    model: "claude-3-sonnet-20240229"
    description: "Implements Python code with best practices"
    capabilities: ["python", "async", "type-hints"]
    tools: ["code", "edit", "lsp", "git"]
    
  - id: "data-sage"
    name: "Data Sage"
    type: "specialist"
    provider: "openai"
    model: "gpt-4-turbo-preview"
    description: "Handles data processing and analysis"
    capabilities: ["pandas", "numpy", "scikit-learn", "visualization"]
    tools: ["code", "jupyter"]

storage:
  backend: "sqlite"
  sqlite:
    path: ".guild/memory.db"

metadata:
  tags: ["python", "data-science", "ml"]
`

const typescriptProjectConfig = `# Guild Configuration for TypeScript Project
name: "guild-project"
description: "TypeScript project managed by Guild Framework"
version: "1.0.0"

manager:
  default: "guild-master"

agents:
  - id: "guild-master"
    name: "Guild Master"
    type: "manager"
    provider: "anthropic"
    model: "claude-3-opus-20240229"
    description: "Orchestrates the guild and assigns tasks"
    capabilities: ["task_routing", "planning", "coordination"]
    
  - id: "frontend-artisan"
    name: "Frontend Artisan"
    type: "worker"
    provider: "anthropic"
    model: "claude-3-sonnet-20240229"
    description: "Implements React/Vue/Angular components"
    capabilities: ["typescript", "react", "vue", "css"]
    tools: ["code", "edit", "lsp", "git"]
    
  - id: "backend-artisan"
    name: "Backend Artisan"
    type: "worker"
    provider: "openai"
    model: "gpt-4-turbo-preview"
    description: "Implements Node.js backend services"
    capabilities: ["typescript", "node", "express", "graphql"]
    tools: ["code", "edit", "lsp", "git"]

storage:
  backend: "sqlite"
  sqlite:
    path: ".guild/memory.db"

metadata:
  tags: ["typescript", "fullstack", "web"]
`

const rustProjectConfig = `# Guild Configuration for Rust Project
name: "guild-project"
description: "Rust project managed by Guild Framework"
version: "1.0.0"

manager:
  default: "guild-master"

agents:
  - id: "guild-master"
    name: "Guild Master"
    type: "manager"
    provider: "anthropic"
    model: "claude-3-opus-20240229"
    description: "Orchestrates the guild and assigns tasks"
    capabilities: ["task_routing", "planning", "coordination"]
    
  - id: "rust-artisan"
    name: "Rust Artisan"
    type: "worker"
    provider: "anthropic"
    model: "claude-3-sonnet-20240229"
    description: "Implements Rust code with safety and performance"
    capabilities: ["rust", "async", "unsafe", "macros"]
    tools: ["code", "edit", "lsp", "git"]
    
  - id: "memory-sage"
    name: "Memory Sage"
    type: "specialist"
    provider: "openai"
    model: "gpt-4-turbo-preview"
    description: "Optimizes memory usage and performance"
    capabilities: ["profiling", "optimization", "benchmarking"]
    tools: ["code", "shell"]

storage:
  backend: "sqlite"
  sqlite:
    path: ".guild/memory.db"

metadata:
  tags: ["rust", "systems", "performance"]
`

const genericProjectConfig = `# Guild Configuration
name: "guild-project"
description: "Project managed by Guild Framework"
version: "1.0.0"

manager:
  default: "guild-master"

agents:
  - id: "guild-master"
    name: "Guild Master"
    type: "manager"
    provider: "anthropic"
    model: "claude-3-opus-20240229"
    description: "Orchestrates the guild and assigns tasks"
    capabilities: ["task_routing", "planning", "coordination"]
    
  - id: "code-artisan"
    name: "Code Artisan"
    type: "worker"
    provider: "anthropic"
    model: "claude-3-sonnet-20240229"
    description: "Implements code with best practices"
    capabilities: ["coding", "refactoring", "debugging"]
    tools: ["code", "edit", "git"]
    
  - id: "doc-scribe"
    name: "Documentation Scribe"
    type: "specialist"
    provider: "openai"
    model: "gpt-4-turbo-preview"
    description: "Writes clear documentation"
    capabilities: ["documentation", "markdown", "diagrams"]
    tools: ["edit", "git"]

storage:
  backend: "sqlite"
  sqlite:
    path: ".guild/memory.db"

metadata:
  tags: ["general"]
`
