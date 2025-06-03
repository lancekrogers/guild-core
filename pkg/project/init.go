package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/config"
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

## Usage

Guild commands will automatically use this project context when run from 
anywhere within the project directory tree.

## Version Control

The following should typically be committed to version control:
- ` + "`corpus/docs/`" + ` - Your project documentation
- ` + "`agents/`" + ` - Agent configurations
- ` + "`objectives/`" + ` - Project objectives
- ` + "`config.yaml`" + ` - Project configuration

The following should typically be ignored:
- ` + "`embeddings/`" + ` - Can be regenerated from corpus
- ` + "`corpus/.activities/`" + ` - Local activity tracking
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
	guildConfig := config.DefaultGuildTemplate()
	
	// Save it to the project
	return config.SaveGuildConfig(filepath.Dir(baseDir), guildConfig)
}