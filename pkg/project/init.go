// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package project

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/storage"
)

// InitOptions contains options for project initialization
type InitOptions struct {
	Name        string `yaml:"name,omitempty"`        // Project name
	Description string `yaml:"description,omitempty"` // Project description
	Template    string `yaml:"template,omitempty"`    // Template to use for initialization
}

// directoryStructure defines the directory structure for a Guild project
var directoryStructure = []string{
	"corpus",
	"corpus/docs",
	"corpus/.activities",
	"embeddings",
	"agents",
	"commissions",
}

// Initialize creates a new Guild project with the given options and returns the project context
// This is the modern API that journey tests expect
func Initialize(ctx context.Context, path string, opts InitOptions) (*Context, error) {
	// Call the refactored initialization function (from init_refactored.go)
	if err := InitializeProject(path); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize project").
			WithComponent("project").
			WithOperation("initialize").
			WithDetails("path", path)
	}

	// Create and return the project context
	projCtx, err := NewContext(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create project context").
			WithComponent("project").
			WithOperation("initialize").
			WithDetails("path", path)
	}

	// TODO: Apply InitOptions (name, description, template) to the project
	// For now, we create a default project, but in the future we could:
	// - Set project name/description in config.yaml
	// - Apply different templates based on opts.Template
	// - Customize the initialization based on options

	return projCtx, nil
}

// createStructure creates the directory structure and initial files
func createStructure(baseDir string) error {
	// Create directories
	for _, dir := range directoryStructure {
		dirPath := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create directory %s", dir).
				WithComponent("project").
				WithOperation("create_structure")
		}
	}

	// Create default config file
	if err := createDefaultConfig(baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create default config").
			WithComponent("project").
			WithOperation("create_structure")
	}

	// Create .gitignore
	if err := createGitignore(baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create .gitignore").
			WithComponent("project").
			WithOperation("create_structure")
	}

	// Create README
	if err := createReadme(baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create README").
			WithComponent("project").
			WithOperation("create_structure")
	}

	// Create default guild configuration
	if err := createDefaultGuildConfig(baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create guild config").
			WithComponent("project").
			WithOperation("create_structure")
	}

	// Initialize database
	if err := initializeDatabase(baseDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize database").
			WithComponent("project").
			WithOperation("create_structure")
	}

	return nil
}

// createDefaultConfig creates a default config.yaml file
func createDefaultConfig(baseDir string) error {
	configPath := filepath.Join(baseDir, "config.yaml")
	content := `# Guild Project Configuration
project:
  name: "Guild Project"
  description: "AI-assisted development project"

corpus:
  max_size_mb: 100
  chunk_size: 1000
  chunk_overlap: 200

rag:
  # Universal embedder configuration
  embedder:
    provider: "ollama"  # Can be any provider in the registry
    model: "nomic-embed-text"  # Or any model that supports embeddings
    dimensions: 768  # Will be auto-detected if not specified

agents:
  default_provider: "ollama"
  default_model: "llama2"

# Registry configuration
registry:
  # Providers will be auto-discovered from the registry
  # Additional provider-specific config can be added here
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

// createGitignore creates a .gitignore file for Guild-specific files
func createGitignore(baseDir string) error {
	gitignorePath := filepath.Join(baseDir, ".gitignore")
	content := `# Guild-specific ignores
embeddings/
corpus/.activities/
*.tmp
*.log

# Keep the structure but ignore embeddings
!embeddings/.gitkeep
`

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// createReadme creates a README for the campaign directory
func createReadme(baseDir string) error {
	readmePath := filepath.Join(baseDir, "README.md")
	content := `# Guild Project

This directory contains Guild-specific files for this project.

## Directory Structure

- ` + "`corpus/`" + ` - Project knowledge base (human-readable Markdown files)
- ` + "`embeddings/`" + ` - Vector embeddings for semantic search (auto-generated)
- ` + "`agents/`" + ` - Agent configurations
- ` + "`commissions/`" + ` - Project commissions and goals
- ` + "`config.yaml`" + ` - Project configuration
- ` + "`guild.yaml`" + ` - Guild agent and provider configuration

## Provider API Keys

**Important**: Guild is configured to use environment variables for API keys by default.
This is the recommended approach for security and team collaboration.

### Required Environment Variables

Set these environment variables before running Guild commands:

` + "```bash" + `
export ANTHROPIC_API_KEY="your-anthropic-api-key"
export OPENAI_API_KEY="your-openai-api-key"
export DEEPSEEK_API_KEY="your-deepseek-api-key"  # If using DeepSeek
export DEEPINFRA_API_KEY="your-deepinfra-api-key"  # If using DeepInfra
export ORA_API_KEY="your-ora-api-key"  # If using Ora
` + "```" + `

### Why Environment Variables?

- **Security**: API keys are not stored in files that might be committed to version control
- **Team Collaboration**: Each team member can use their own API keys
- **Safe Sharing**: You can safely commit ` + "`guild.yaml`" + ` without exposing secrets

### Configuration in guild.yaml

The ` + "`guild.yaml`" + ` file contains agent configurations and provider settings (like base URLs for self-hosted services), but **never API keys**:

` + "```yaml" + `
providers:
  ollama:
    base_url: "http://localhost:11434"  # Custom base URL for self-hosted Ollama
  # API keys are only read from environment variables
` + "```" + `

## Usage

Guild commands will automatically use this project context when run from
anywhere within the project directory tree.

## Version Control

### Safe to Commit
- ` + "`corpus/docs/`" + ` - Your project documentation
- ` + "`agents/`" + ` - Agent configurations
- ` + "`commissions/`" + ` - Project commissions
- ` + "`config.yaml`" + ` - Project configuration
- ` + "`guild.yaml`" + ` - Guild configuration (safe to commit - no API keys stored)

### Should Not Be Committed
- ` + "`embeddings/`" + ` - Can be regenerated from corpus
- ` + "`corpus/.activities/`" + ` - Local activity tracking

## Getting Started

1. Set your API keys as environment variables (see above)
2. Run: ` + "`guild commission \"Your first task\" --assign`" + `
3. Monitor progress: ` + "`guild workshop`" + `
`

	return os.WriteFile(readmePath, []byte(content), 0644)
}

// InitializeWithConfig creates a new Guild project with custom configuration
func InitializeWithConfig(path string, config interface{}) error {
	// First do standard initialization using the refactored function
	if err := InitializeProject(path); err != nil {
		return err
	}

	// TODO: Apply custom configuration
	// This would update the config.yaml with provided settings

	return nil
}

// createDefaultGuildConfig creates a default guild.yaml file
func createDefaultGuildConfig(baseDir string) error {
	// Get the default template
	guildConfig := DefaultGuildTemplate()

	// Save it to the campaign directory
	guildPath := filepath.Join(baseDir, "guild.yaml")
	data, err := yaml.Marshal(guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("project").
			WithOperation("create_default_guild_config")
	}

	return os.WriteFile(guildPath, data, 0644)
}

// initializeDatabase creates and migrates the SQLite database
// Following Guild's context-aware pattern
func initializeDatabase(baseDir string) error {
	ctx := context.Background()

	// Create database path
	dbPath := filepath.Join(baseDir, "memory.db")

	// Create database
	db, err := storage.DefaultDatabaseFactory(ctx, dbPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create database").
			WithComponent("project").
			WithOperation("initialize_database")
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run database migrations").
			WithComponent("project").
			WithOperation("initialize_database")
	}

	return nil
}
