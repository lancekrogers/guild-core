package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/config"
)

// ProjectType represents different types of projects that can be detected
type ProjectType struct {
	Name        string   `json:"name"`
	Language    string   `json:"language"`
	Framework   string   `json:"framework,omitempty"`
	Indicators  []string `json:"indicators"`
	Description string   `json:"description"`
}

// ProjectDetector provides intelligent project detection and configuration generation
type ProjectDetector struct{}

// NewProjectDetector creates a new project detector
func NewProjectDetector() *ProjectDetector {
	return &ProjectDetector{}
}

// knownProjectTypes defines the project types we can detect
var knownProjectTypes = []ProjectType{
	{
		Name:        "go-web",
		Language:    "go",
		Framework:   "web",
		Indicators:  []string{"go.mod", "main.go", "cmd/", "pkg/", "internal/"},
		Description: "Go web application or microservice",
	},
	{
		Name:        "go-cli",
		Language:    "go",
		Framework:   "cli",
		Indicators:  []string{"go.mod", "main.go", "cmd/"},
		Description: "Go command-line application",
	},
	{
		Name:        "go-lib",
		Language:    "go",
		Framework:   "library",
		Indicators:  []string{"go.mod", "*.go"},
		Description: "Go library or package",
	},
	{
		Name:        "node-web",
		Language:    "javascript",
		Framework:   "web",
		Indicators:  []string{"package.json", "src/", "public/", "index.html"},
		Description: "Node.js web application",
	},
	{
		Name:        "node-api",
		Language:    "javascript",
		Framework:   "api",
		Indicators:  []string{"package.json", "server.js", "app.js", "routes/"},
		Description: "Node.js API server",
	},
	{
		Name:        "python-web",
		Language:    "python",
		Framework:   "web",
		Indicators:  []string{"requirements.txt", "app.py", "main.py", "wsgi.py"},
		Description: "Python web application",
	},
	{
		Name:        "python-data",
		Language:    "python",
		Framework:   "data-science",
		Indicators:  []string{"requirements.txt", "*.ipynb", "data/", "models/"},
		Description: "Python data science project",
	},
	{
		Name:        "rust-app",
		Language:    "rust",
		Framework:   "application",
		Indicators:  []string{"Cargo.toml", "src/main.rs"},
		Description: "Rust application",
	},
	{
		Name:        "rust-lib",
		Language:    "rust",
		Framework:   "library",
		Indicators:  []string{"Cargo.toml", "src/lib.rs"},
		Description: "Rust library",
	},
	{
		Name:        "generic",
		Language:    "unknown",
		Framework:   "",
		Indicators:  []string{},
		Description: "Generic project (language not detected)",
	},
}

// DetectProjectType analyzes the given path and returns the most likely project type
func (d *ProjectDetector) DetectProjectType(path string) (*ProjectType, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Create a map of files and directories for quick lookup
	fileMap := make(map[string]bool)
	dirMap := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			dirMap[name+"/"] = true
		} else {
			fileMap[name] = true
			// Also add by extension for pattern matching
			if ext := filepath.Ext(name); ext != "" {
				fileMap["*"+ext] = true
			}
		}
	}

	// Score each project type based on indicator matches
	bestMatch := &knownProjectTypes[len(knownProjectTypes)-1] // Default to generic
	bestScore := 0

	for i := range knownProjectTypes {
		projectType := &knownProjectTypes[i]
		if projectType.Name == "generic" {
			continue // Skip generic, it's the fallback
		}

		score := d.scoreProjectType(projectType, fileMap, dirMap)
		if score > bestScore {
			bestScore = score
			bestMatch = projectType
		}
	}

	// If no strong match, return generic
	if bestScore == 0 {
		bestMatch = &knownProjectTypes[len(knownProjectTypes)-1]
	}

	return bestMatch, nil
}

// scoreProjectType calculates how well a project type matches the directory contents
func (d *ProjectDetector) scoreProjectType(projectType *ProjectType, fileMap, dirMap map[string]bool) int {
	score := 0
	for _, indicator := range projectType.Indicators {
		if strings.HasSuffix(indicator, "/") {
			// Directory indicator
			if dirMap[indicator] {
				score += 2 // Directories are worth more
			}
		} else if strings.HasPrefix(indicator, "*") {
			// Pattern indicator (e.g., "*.go")
			if fileMap[indicator] {
				score += 1
			}
		} else {
			// Exact file indicator
			if fileMap[indicator] {
				score += 3 // Exact matches are worth the most
			}
		}
	}
	return score
}

// GenerateGuildConfig creates an appropriate guild configuration for the detected project type
func (d *ProjectDetector) GenerateGuildConfig(projectType *ProjectType, projectPath string) (*config.GuildConfig, error) {
	projectName := filepath.Base(projectPath)

	guildConfig := &config.GuildConfig{
		Name:        projectName,
		Description: fmt.Sprintf("%s project initialized with Guild", projectType.Description),
		Version:     "1.0.0",
		Manager: config.ManagerConfig{
			Default: "architect", // Default to architect agent as manager
		},
		Agents:    d.generateAgentsForProjectType(projectType),
		Providers: d.generateProvidersConfig(),
	}

	return guildConfig, nil
}

// generateAgentsForProjectType creates appropriate agent configurations based on project type
func (d *ProjectDetector) generateAgentsForProjectType(projectType *ProjectType) []config.AgentConfig {
	baseAgents := []config.AgentConfig{
		{
			ID:           "architect",
			Name:         "System Architect",
			Type:         "manager",
			Capabilities: []string{"system_design", "architecture", "planning"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 8, // High-level strategic work
		},
	}

	// Add language-specific agents
	switch projectType.Language {
	case "go":
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "go-dev",
			Name:         "Go Developer",
			Type:         "worker",
			Capabilities: []string{"golang", "backend", "testing", "performance"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 3, // Mid-level development work
		})
	case "javascript":
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "js-dev",
			Name:         "JavaScript Developer",
			Type:         "worker",
			Capabilities: []string{"javascript", "nodejs", "frontend", "testing"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 3,
		})
	case "python":
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "python-dev",
			Name:         "Python Developer",
			Type:         "worker",
			Capabilities: []string{"python", "backend", "data", "testing"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 3,
		})
	case "rust":
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "rust-dev",
			Name:         "Rust Developer",
			Type:         "worker",
			Capabilities: []string{"rust", "systems", "performance", "safety"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 5,
		})
	}

	// Add framework-specific agents
	if projectType.Framework == "web" {
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "frontend-dev",
			Name:         "Frontend Developer",
			Type:         "specialist",
			Capabilities: []string{"ui", "ux", "css", "html", "responsive"},
			Model:        "claude-3-haiku-20240307",
			Provider:     "anthropic",
			CostMagnitude: 2,
		})
	}

	if projectType.Framework == "data-science" {
		baseAgents = append(baseAgents, config.AgentConfig{
			ID:           "data-scientist",
			Name:         "Data Scientist",
			Type:         "specialist",
			Capabilities: []string{"data_analysis", "machine_learning", "visualization", "statistics"},
			Model:        "claude-3-sonnet-20240229",
			Provider:     "anthropic",
			CostMagnitude: 5,
		})
	}

	return baseAgents
}

// generateProvidersConfig creates default provider configuration with environment variable detection
func (d *ProjectDetector) generateProvidersConfig() config.ProvidersConfig {
	providers := config.ProvidersConfig{}

	// Check for API keys in environment and configure accordingly
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		providers.Anthropic = config.ProviderSettings{
			Settings: map[string]string{
				"enabled": "true",
			},
		}
	}

	if os.Getenv("OPENAI_API_KEY") != "" {
		providers.OpenAI = config.ProviderSettings{
			Settings: map[string]string{
				"enabled": "true",
			},
		}
	}

	// Add Ollama as a fallback for local development
	providers.Ollama = config.ProviderSettings{
		BaseURL: "http://localhost:11434",
		Settings: map[string]string{
			"enabled": "false", // Disabled by default
		},
	}

	return providers
}

// CorpusConfig represents corpus configuration to avoid import cycles
type CorpusConfig struct {
	CorpusPath      string   `yaml:"corpus_path"`
	ActivitiesPath  string   `yaml:"activities_path"`
	MaxSizeBytes    int64    `yaml:"max_size_bytes"`
	DefaultTags     []string `yaml:"default_tags"`
	DefaultCategory string   `yaml:"default_category"`
}

// GenerateCorpusConfig creates appropriate corpus configuration for the detected project type
func (d *ProjectDetector) GenerateCorpusConfig(projectType *ProjectType, projectPath string) CorpusConfig {
	// Update paths to be project-relative
	guildPath := filepath.Join(projectPath, ".guild")

	corpusConfig := CorpusConfig{
		CorpusPath:      filepath.Join(guildPath, "corpus"),
		ActivitiesPath:  filepath.Join(guildPath, "corpus", ".activities"),
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB default
		DefaultCategory: projectType.Language,
	}

	// Set appropriate tags based on project type
	tags := []string{projectType.Language}
	if projectType.Framework != "" {
		tags = append(tags, projectType.Framework)
	}
	corpusConfig.DefaultTags = tags

	return corpusConfig
}

// SeedCorpusFromProject analyzes the project and suggests files to add to the corpus
func (d *ProjectDetector) SeedCorpusFromProject(projectType *ProjectType, projectPath string) ([]string, error) {
	var suggestions []string

	// Walk the project directory and suggest documentation files
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common non-documentation directories
		skipDirs := []string{"node_modules", "vendor", "target", "build", "dist", ".git"}
		for _, skipDir := range skipDirs {
			if info.IsDir() && info.Name() == skipDir {
				return filepath.SkipDir
			}
		}

		// Look for documentation files
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			name := strings.ToLower(info.Name())

			// Documentation file patterns
			isDoc := ext == ".md" || ext == ".txt" || ext == ".rst" ||
				strings.Contains(name, "readme") ||
				strings.Contains(name, "changelog") ||
				strings.Contains(name, "license") ||
				strings.Contains(name, "contributing") ||
				strings.Contains(name, "doc")

			if isDoc {
				relPath, err := filepath.Rel(projectPath, path)
				if err == nil {
					suggestions = append(suggestions, relPath)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan project for corpus files: %w", err)
	}

	return suggestions, nil
}
