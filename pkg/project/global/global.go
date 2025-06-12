package global

import (
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// globalDirectoryStructure defines the directory structure for the global Guild configuration
var globalDirectoryStructure = []string{
	"providers",
	"tools", // Global tool installations (shared across all projects)
	"templates",
	"templates/golang",
	"templates/python",
	"templates/typescript",
	"cache",
	"logs",
}

// GlobalGuildDir returns the path to the global Guild directory
func GlobalGuildDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".guild"
	}
	return filepath.Join(homeDir, ".guild")
}

// InitializeGlobal creates the global Guild directory structure
func InitializeGlobal() error {
	globalDir := GlobalGuildDir()

	// Check if already initialized
	if _, err := os.Stat(globalDir); err == nil {
		// Directory exists, check if it's properly initialized
		configPath := filepath.Join(globalDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			// Already initialized
			return nil
		}
	}

	// Create global directory
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create global Guild directory").
			WithComponent("project").
			WithOperation("initialize_global").
			WithDetails("path", globalDir)
	}

	// Create subdirectories
	for _, dir := range globalDirectoryStructure {
		dirPath := filepath.Join(globalDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create directory %s", dir).
				WithComponent("project").
				WithOperation("initialize_global")
		}
	}

	// Create default global config
	if err := createDefaultGlobalConfig(globalDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create global config").
			WithComponent("project").
			WithOperation("initialize_global")
	}

	// Create provider configs
	if err := createProviderConfigs(globalDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider configs").
			WithComponent("project").
			WithOperation("initialize_global")
	}

	// Create template files
	if err := createTemplates(globalDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create templates").
			WithComponent("project").
			WithOperation("initialize_global")
	}

	return nil
}

// createDefaultGlobalConfig creates the default global configuration file
func createDefaultGlobalConfig(globalDir string) error {
	configPath := filepath.Join(globalDir, "config.yaml")
	content := `# Guild Global Configuration
# This file contains settings that apply to all Guild projects

# Default provider settings
providers:
  default: "anthropic"
  fallback:
    - "openai"
    - "ollama"

# Tool settings (global defaults - can be overridden per project)
tools:
  enabled:
    - git
    - code
    - lsp
    - file
    - web
  disabled: []

# Cache settings
cache:
  embeddings:
    max_size_gb: 10
    ttl_days: 30
  
# Logging settings
logging:
  level: "info"
  max_size_mb: 100
  max_files: 5

# UI settings
ui:
  vim_mode: true
  theme: "monokai"
  
# Security settings
security:
  api_keys:
    # NEVER store actual API keys here
    # Always use environment variables
    source: "environment"
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

// createProviderConfigs creates default provider configuration files
func createProviderConfigs(globalDir string) error {
	providersDir := filepath.Join(globalDir, "providers")

	// OpenAI config
	openaiConfig := `# OpenAI Provider Configuration
# API key should be set via OPENAI_API_KEY environment variable

base_url: "https://api.openai.com/v1"
default_model: "gpt-4-turbo-preview"
timeout_seconds: 60

models:
  - name: "gpt-4-turbo-preview"
    max_tokens: 4096
    cost_per_1k_input: 0.01
    cost_per_1k_output: 0.03
  - name: "gpt-3.5-turbo"
    max_tokens: 4096
    cost_per_1k_input: 0.0005
    cost_per_1k_output: 0.0015
`
	if err := os.WriteFile(filepath.Join(providersDir, "openai.yaml"), []byte(openaiConfig), 0644); err != nil {
		return err
	}

	// Anthropic config
	anthropicConfig := `# Anthropic Provider Configuration
# API key should be set via ANTHROPIC_API_KEY environment variable

base_url: "https://api.anthropic.com"
default_model: "claude-3-opus-20240229"
timeout_seconds: 60

models:
  - name: "claude-3-opus-20240229"
    max_tokens: 4096
    cost_per_1k_input: 0.015
    cost_per_1k_output: 0.075
  - name: "claude-3-sonnet-20240229"
    max_tokens: 4096
    cost_per_1k_input: 0.003
    cost_per_1k_output: 0.015
`
	if err := os.WriteFile(filepath.Join(providersDir, "anthropic.yaml"), []byte(anthropicConfig), 0644); err != nil {
		return err
	}

	// Ollama config
	ollamaConfig := `# Ollama Provider Configuration
# No API key required for local models

base_url: "http://localhost:11434"
default_model: "llama2"
timeout_seconds: 300

models:
  - name: "llama2"
    max_tokens: 4096
    cost_per_1k_input: 0
    cost_per_1k_output: 0
  - name: "codellama"
    max_tokens: 4096
    cost_per_1k_input: 0
    cost_per_1k_output: 0
  - name: "nomic-embed-text"
    embedding: true
    dimensions: 768
`
	if err := os.WriteFile(filepath.Join(providersDir, "ollama.yaml"), []byte(ollamaConfig), 0644); err != nil {
		return err
	}

	return nil
}

// createTemplates creates project template files
func createTemplates(globalDir string) error {
	templatesDir := filepath.Join(globalDir, "templates")

	// Go template
	goTemplate := `# Go Project Template

name: "golang"
description: "Template for Go projects"

agents:
  - name: "code-reviewer"
    type: "reviewer"
    specialties: ["go", "testing", "performance"]
  - name: "architect" 
    type: "designer"
    specialties: ["go", "microservices", "apis"]

tools:
  - "git"
  - "code" 
  - "lsp"
  - "go-test"
  - "go-fmt"

corpus:
  include:
    - "**/*.go"
    - "go.mod"
    - "go.sum"
    - "README.md"
    - "docs/**/*.md"
  exclude:
    - "vendor/**"
    - "**/*_test.go"

objectives:
  - "Maintain test coverage above 80%"
  - "Follow Go best practices and idioms"
  - "Optimize for performance and readability"
`
	if err := os.WriteFile(filepath.Join(templatesDir, "golang", "template.yaml"), []byte(goTemplate), 0644); err != nil {
		return err
	}

	// Python template
	pythonTemplate := `# Python Project Template

name: "python"
description: "Template for Python projects"

agents:
  - name: "code-reviewer"
    type: "reviewer"
    specialties: ["python", "testing", "type-hints"]
  - name: "data-scientist"
    type: "specialist"
    specialties: ["python", "pandas", "numpy", "ml"]

tools:
  - "git"
  - "code"
  - "lsp"
  - "pytest"
  - "black"
  - "mypy"

corpus:
  include:
    - "**/*.py"
    - "requirements.txt"
    - "pyproject.toml"
    - "README.md"
    - "docs/**/*.md"
  exclude:
    - "__pycache__/**"
    - "venv/**"
    - ".pytest_cache/**"

objectives:
  - "Maintain type hints for all public APIs"
  - "Follow PEP 8 style guidelines"
  - "Write comprehensive docstrings"
`
	if err := os.WriteFile(filepath.Join(templatesDir, "python", "template.yaml"), []byte(pythonTemplate), 0644); err != nil {
		return err
	}

	// TypeScript template
	tsTemplate := `# TypeScript Project Template

name: "typescript"
description: "Template for TypeScript projects"

agents:
  - name: "frontend-dev"
    type: "developer"
    specialties: ["typescript", "react", "css"]
  - name: "backend-dev"
    type: "developer"
    specialties: ["typescript", "node", "apis"]

tools:
  - "git"
  - "code"
  - "lsp"
  - "npm"
  - "eslint"
  - "prettier"

corpus:
  include:
    - "**/*.ts"
    - "**/*.tsx"
    - "package.json"
    - "tsconfig.json"
    - "README.md"
    - "docs/**/*.md"
  exclude:
    - "node_modules/**"
    - "dist/**"
    - "build/**"

objectives:
  - "Maintain strict TypeScript settings"
  - "Follow React best practices"
  - "Ensure accessibility compliance"
`
	if err := os.WriteFile(filepath.Join(templatesDir, "typescript", "template.yaml"), []byte(tsTemplate), 0644); err != nil {
		return err
	}

	return nil
}

// EnsureGlobalInitialized ensures the global Guild directory is initialized
// This should be called before any Guild operation
func EnsureGlobalInitialized() error {
	globalDir := GlobalGuildDir()
	configPath := filepath.Join(globalDir, "config.yaml")

	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Initialize if not exists
	return InitializeGlobal()
}

// GlobalConfigPath returns the path to the global config file
func GlobalConfigPath() string {
	return filepath.Join(GlobalGuildDir(), "config.yaml")
}

// GlobalProviderConfigPath returns the path to a provider's global config
func GlobalProviderConfigPath(provider string) string {
	return filepath.Join(GlobalGuildDir(), "providers", provider+".yaml")
}

// GlobalTemplatePath returns the path to a project template
func GlobalTemplatePath(templateName string) string {
	return filepath.Join(GlobalGuildDir(), "templates", templateName, "template.yaml")
}

// ListTemplates returns available project templates
func ListTemplates() ([]string, error) {
	templatesDir := filepath.Join(GlobalGuildDir(), "templates")

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list templates").
			WithComponent("project").
			WithOperation("list_templates")
	}

	var templates []string
	for _, entry := range entries {
		if entry.IsDir() {
			templateFile := filepath.Join(templatesDir, entry.Name(), "template.yaml")
			if _, err := os.Stat(templateFile); err == nil {
				templates = append(templates, entry.Name())
			}
		}
	}

	return templates, nil
}
