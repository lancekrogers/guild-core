package lsp

import (
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"gopkg.in/yaml.v3"
)

// Config represents the LSP configuration
type Config struct {
	Servers map[string]*ServerConfig `yaml:"servers"`
}

// ServerConfig represents configuration for a language server
type ServerConfig struct {
	Language     string                 `yaml:"language"`
	Command      []string               `yaml:"command"`
	InitOptions  map[string]interface{} `yaml:"init_options,omitempty"`
	FilePatterns []string               `yaml:"file_patterns,omitempty"`
	RootMarkers  []string               `yaml:"root_markers,omitempty"`
	Environment  map[string]string      `yaml:"environment,omitempty"`
}

// DefaultConfigs returns default language server configurations
func DefaultConfigs() map[string]*ServerConfig {
	return map[string]*ServerConfig{
		"go": {
			Language: "go",
			Command:  []string{"gopls", "serve"},
			InitOptions: map[string]interface{}{
				"usePlaceholders":    true,
				"completeUnimported": true,
				"deepCompletion":     true,
				"staticcheck":        true,
			},
			FilePatterns: []string{"*.go"},
			RootMarkers:  []string{"go.mod", "go.sum"},
		},
		"typescript": {
			Language: "typescript",
			Command:  []string{"typescript-language-server", "--stdio"},
			InitOptions: map[string]interface{}{
				"preferences": map[string]interface{}{
					"includeCompletionsWithSnippetText": true,
					"includeCompletionsForImportStatements": true,
				},
			},
			FilePatterns: []string{"*.ts", "*.tsx", "*.js", "*.jsx"},
			RootMarkers:  []string{"package.json", "tsconfig.json"},
		},
		"python": {
			Language: "python",
			Command:  []string{"pylsp"},
			InitOptions: map[string]interface{}{
				"plugins": map[string]interface{}{
					"jedi_completion": map[string]interface{}{
						"enabled": true,
						"include_params": true,
					},
					"jedi_hover": map[string]interface{}{
						"enabled": true,
					},
					"jedi_references": map[string]interface{}{
						"enabled": true,
					},
					"jedi_definition": map[string]interface{}{
						"enabled": true,
						"follow_imports": true,
					},
				},
			},
			FilePatterns: []string{"*.py"},
			RootMarkers:  []string{"setup.py", "requirements.txt", "pyproject.toml"},
		},
		"rust": {
			Language: "rust",
			Command:  []string{"rust-analyzer"},
			InitOptions: map[string]interface{}{
				"cargo": map[string]interface{}{
					"loadOutDirsFromCheck": true,
				},
				"procMacro": map[string]interface{}{
					"enable": true,
				},
			},
			FilePatterns: []string{"*.rs"},
			RootMarkers:  []string{"Cargo.toml"},
		},
		"java": {
			Language: "java",
			Command:  []string{"jdtls"},
			FilePatterns: []string{"*.java"},
			RootMarkers:  []string{"pom.xml", "build.gradle", ".project"},
		},
		"csharp": {
			Language: "csharp",
			Command:  []string{"omnisharp", "--languageserver", "--hostPID", "$$"},
			FilePatterns: []string{"*.cs"},
			RootMarkers:  []string{"*.csproj", "*.sln"},
		},
	}
}

// LoadConfig loads LSP configuration from file
func LoadConfig(path string) (*Config, error) {
	// If no path specified, use default path
	if path == "" {
		path = GetConfigPath()
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return config with default servers if file doesn't exist
			// This ensures basic functionality works out of the box
			return &Config{
				Servers: DefaultConfigs(),
			}, nil
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read LSP config file").
			WithComponent("lsp").
			WithOperation("load_config").
			WithDetails("path", path)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse LSP config").
			WithComponent("lsp").
			WithOperation("load_config").
			WithDetails("path", path)
	}

	// Initialize servers map if nil
	if config.Servers == nil {
		config.Servers = make(map[string]*ServerConfig)
	}

	return &config, nil
}

// SaveConfig saves LSP configuration to file
func SaveConfig(config *Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create config directory").
			WithComponent("lsp").
			WithOperation("save_config").
			WithDetails("dir", dir)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal LSP config").
			WithComponent("lsp").
			WithOperation("save_config")
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write LSP config file").
			WithComponent("lsp").
			WithOperation("save_config").
			WithDetails("path", path)
	}

	return nil
}

// GetConfigPath returns the default LSP config path
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".guild", "lsp", "config.yaml")
}

// DetectLanguage detects the language from a file path based on configuration
func DetectLanguage(filePath string) string {
	// This is now handled by the manager using configured file patterns
	// Kept for backward compatibility
	ext := filepath.Ext(filePath)
	
	// Common extension mappings as fallback
	commonMappings := map[string]string{
		".go":   "go",
		".ts":   "typescript",
		".tsx":  "typescript",
		".js":   "javascript",
		".jsx":  "javascript",
		".py":   "python",
		".rs":   "rust",
		".java": "java",
		".cs":   "csharp",
		".c":    "c",
		".cpp":  "cpp",
		".cc":   "cpp",
		".cxx":  "cpp",
		".rb":   "ruby",
		".php":  "php",
		".lua":  "lua",
		".kt":   "kotlin",
		".kts":  "kotlin",
		".swift": "swift",
		".zig":  "zig",
		".hs":   "haskell",
		".lhs":  "haskell",
		".ex":   "elixir",
		".exs":  "elixir",
		".vim":  "vim",
		".sh":   "bash",
		".bash": "bash",
		".zsh":  "zsh",
		".fish": "fish",
	}
	
	if lang, ok := commonMappings[ext]; ok {
		return lang
	}
	
	return ""
}

// FindRootPath finds the project root path based on root markers
func FindRootPath(startPath string, markers []string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "failed to get absolute path").
			WithComponent("lsp").
			WithOperation("find_root").
			WithDetails("path", startPath)
	}

	// If startPath is a file, start from its directory
	info, err := os.Stat(absPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "failed to stat path").
			WithComponent("lsp").
			WithOperation("find_root").
			WithDetails("path", absPath)
	}

	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	// Walk up the directory tree looking for markers
	for {
		for _, marker := range markers {
			markerPath := filepath.Join(absPath, marker)
			if _, err := os.Stat(markerPath); err == nil {
				return absPath, nil
			}
		}

		parent := filepath.Dir(absPath)
		if parent == absPath {
			// Reached the root directory
			break
		}
		absPath = parent
	}

	// If no markers found, return the original directory
	if info.IsDir() {
		return startPath, nil
	}
	return filepath.Dir(startPath), nil
}