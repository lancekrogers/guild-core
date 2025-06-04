package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectDetector_DetectProjectType(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string]string // filename -> content
		dirs           []string
		expectedName   string
		expectedLang   string
	}{
		{
			name: "go web project",
			files: map[string]string{
				"go.mod":  "module example.com/app\n\ngo 1.19",
				"main.go": "package main\n\nfunc main() {}",
			},
			dirs:         []string{"cmd", "pkg", "internal"},
			expectedName: "go-web",
			expectedLang: "go",
		},
		{
			name: "node.js web project",
			files: map[string]string{
				"package.json": `{"name": "my-app", "main": "index.js"}`,
				"index.html":   "<html></html>",
			},
			dirs:         []string{"src", "public"},
			expectedName: "node-web",
			expectedLang: "javascript",
		},
		{
			name: "python project",
			files: map[string]string{
				"requirements.txt": "flask==2.0.1\nrequests==2.25.1",
				"app.py":           "from flask import Flask\n\napp = Flask(__name__)",
			},
			expectedName: "python-web",
			expectedLang: "python",
		},
		{
			name: "rust application",
			files: map[string]string{
				"Cargo.toml": `[package]\nname = "my-app"\nversion = "0.1.0"`,
				"src/main.rs": "fn main() {\n    println!(\"Hello, world!\");\n}",
			},
			expectedName: "rust-app",
			expectedLang: "rust",
		},
		{
			name:         "generic project",
			files:        map[string]string{"README.md": "# My Project"},
			expectedName: "generic",
			expectedLang: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "project-detect-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create test files
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(filePath)
				if dir != tmpDir {
					err := os.MkdirAll(dir, 0755)
					require.NoError(t, err)
				}
				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Create test directories
			for _, dirName := range tt.dirs {
				err := os.MkdirAll(filepath.Join(tmpDir, dirName), 0755)
				require.NoError(t, err)
			}

			// Test detection
			detector := NewProjectDetector()
			projectType, err := detector.DetectProjectType(tmpDir)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedName, projectType.Name)
			assert.Equal(t, tt.expectedLang, projectType.Language)
		})
	}
}

func TestProjectDetector_GenerateGuildConfig(t *testing.T) {
	detector := NewProjectDetector()
	projectType := &ProjectType{
		Name:        "go-web",
		Language:    "go",
		Framework:   "web",
		Description: "Go web application",
	}

	tmpDir, err := os.MkdirTemp("", "guild-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config, err := detector.GenerateGuildConfig(projectType, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, filepath.Base(tmpDir), config.Name)
	assert.Contains(t, config.Description, "Go web application")
	assert.Equal(t, "architect", config.Manager.Default)

	// Check that we have at least architect and go-dev agents
	assert.GreaterOrEqual(t, len(config.Agents), 2)
	
	var hasArchitect, hasGoDev bool
	for _, agent := range config.Agents {
		if agent.ID == "architect" {
			hasArchitect = true
			assert.Equal(t, "manager", agent.Type)
		}
		if agent.ID == "go-dev" {
			hasGoDev = true
			assert.Equal(t, "worker", agent.Type)
			assert.Contains(t, agent.Capabilities, "golang")
		}
	}
	assert.True(t, hasArchitect, "Should have architect agent")
	assert.True(t, hasGoDev, "Should have Go developer agent")
}

func TestProjectDetector_GenerateCorpusConfig(t *testing.T) {
	detector := NewProjectDetector()
	projectType := &ProjectType{
		Name:        "python-data",
		Language:    "python",
		Framework:   "data-science",
		Description: "Python data science project",
	}

	tmpDir, err := os.MkdirTemp("", "corpus-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := detector.GenerateCorpusConfig(projectType, tmpDir)

	expectedCorpusPath := filepath.Join(tmpDir, ".guild", "corpus")
	expectedActivitiesPath := filepath.Join(tmpDir, ".guild", "corpus", ".activities")

	assert.Equal(t, expectedCorpusPath, config.CorpusPath)
	assert.Equal(t, expectedActivitiesPath, config.ActivitiesPath)
	assert.Equal(t, "python", config.DefaultCategory)
	assert.Contains(t, config.DefaultTags, "python")
	assert.Contains(t, config.DefaultTags, "data-science")
	assert.Equal(t, int64(100*1024*1024), config.MaxSizeBytes) // 100MB
}

func TestProjectDetector_SeedCorpusFromProject(t *testing.T) {
	// Create temporary directory with documentation files
	tmpDir, err := os.MkdirTemp("", "corpus-seed-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"README.md":           "# My Project",
		"CHANGELOG.md":        "## Version 1.0",
		"docs/api.md":         "# API Documentation",
		"docs/setup.md":       "# Setup Guide",
		"src/main.go":         "package main", // Should be ignored
		"node_modules/lib.js": "// Library", // Should be ignored
		".git/config":         "[core]",      // Should be ignored
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		dir := filepath.Dir(filePath)
		if dir != tmpDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	detector := NewProjectDetector()
	projectType := &ProjectType{Name: "go-web", Language: "go"}

	suggestions, err := detector.SeedCorpusFromProject(projectType, tmpDir)
	require.NoError(t, err)

	// Should find documentation files but not source code or ignored directories
	expectedFiles := []string{"README.md", "CHANGELOG.md", "docs/api.md", "docs/setup.md"}
	assert.ElementsMatch(t, expectedFiles, suggestions)
}

func TestProjectDetector_ProviderDetection(t *testing.T) {
	// Test with no API keys
	detector := NewProjectDetector()
	providers := detector.generateProvidersConfig()

	// Should have Ollama (disabled) but no enabled providers
	assert.Equal(t, "false", providers.Ollama.Settings["enabled"])

	// Test with ANTHROPIC_API_KEY set
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	providers = detector.generateProvidersConfig()
	assert.Equal(t, "true", providers.Anthropic.Settings["enabled"])

	// Test with both keys set
	os.Setenv("OPENAI_API_KEY", "test-key-2")
	defer os.Unsetenv("OPENAI_API_KEY")

	providers = detector.generateProvidersConfig()
	assert.Equal(t, "true", providers.Anthropic.Settings["enabled"])
	assert.Equal(t, "true", providers.OpenAI.Settings["enabled"])
}