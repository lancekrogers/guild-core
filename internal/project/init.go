package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/guild-ventures/guild-core/pkg/storage"
	"gopkg.in/yaml.v3"
)

// directoryStructure defines the directory structure for a Guild project
var directoryStructure = []string{
	"corpus",
	"corpus/docs",
	"corpus/.activities",
	"embeddings",
	"agents",
	"objectives",
}

// Initialize creates a new Guild project structure at the specified path
func Initialize(path string) error {
	// Validate the path
	if err := ValidateProjectPath(path); err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	// Check if already initialized
	if IsInitialized(path) {
		return ErrAlreadyInitialized
	}

	// Create project structure atomically
	tempDir := filepath.Join(path, ".guild.tmp")
	finalDir := filepath.Join(path, ".guild")

	// Clean up temp directory if it exists
	os.RemoveAll(tempDir)

	// Create temporary directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create structure in temp directory
	if err := createStructure(tempDir); err != nil {
		os.RemoveAll(tempDir) // Clean up on error
		return fmt.Errorf("failed to create project structure: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempDir, finalDir); err != nil {
		os.RemoveAll(tempDir) // Clean up on error
		return fmt.Errorf("failed to finalize project structure: %w", err)
	}

	return nil
}

// createStructure creates the directory structure and initial files
func createStructure(baseDir string) error {
	// Create directories
	for _, dir := range directoryStructure {
		dirPath := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default config file
	if err := createDefaultConfig(baseDir); err != nil {
		return fmt.Errorf("failed to create default config: %w", err)
	}

	// Create .gitignore
	if err := createGitignore(baseDir); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README
	if err := createReadme(baseDir); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	// Create default guild configuration
	if err := createDefaultGuildConfig(baseDir); err != nil {
		return fmt.Errorf("failed to create guild config: %w", err)
	}

	// Initialize database
	if err := initializeDatabase(baseDir); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
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

// createReadme creates a README for the .guild directory
func createReadme(baseDir string) error {
	readmePath := filepath.Join(baseDir, "README.md")
	content := `# Guild Project

This directory contains Guild-specific files for this project.

## Directory Structure

- ` + "`corpus/`" + ` - Project knowledge base (human-readable Markdown files)
- ` + "`embeddings/`" + ` - Vector embeddings for semantic search (auto-generated)
- ` + "`agents/`" + ` - Agent configurations
- ` + "`objectives/`" + ` - Project objectives and goals
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
- ` + "`objectives/`" + ` - Project objectives
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
	// First do standard initialization
	if err := Initialize(path); err != nil {
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
	
	// Save it to the .guild directory
	guildPath := filepath.Join(baseDir, "guild.yaml")
	data, err := yaml.Marshal(guildConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal guild config: %w", err)
	}

	return os.WriteFile(guildPath, data, 0644)
}

// initializeDatabase creates and migrates the SQLite database
// Following Guild's context-aware pattern
func initializeDatabase(baseDir string) error {
	ctx := context.Background()
	
	// Create database path
	dbPath := filepath.Join(baseDir, "guild.db")
	
	// Create database
	db, err := storage.NewDatabase(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()
	
	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	
	return nil
}