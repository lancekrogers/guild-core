//go:build integration

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/testutil"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGuildInitialization validates guild init flows and provider configuration
func TestGuildInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, dir string)
		args         []string
		validateFunc func(t *testing.T, dir string, result *testutil.CommandResult)
		wantError    bool
		wantInOutput string
	}{
		{
			name: "default_initialization",
			args: []string{"init"},
			validateFunc: func(t *testing.T, dir string, result *testutil.CommandResult) {
				// Verify .guild directory created
				guildDir := filepath.Join(dir, ".guild")
				require.DirExists(t, guildDir, ".guild directory should exist")

				// Verify default files created
				files := []string{
					".guild/guild.yaml",
					".guild/memory.db",
					".guild/objectives",
					".guild/kanban",
					".guild/archives",
					".guild/prompts",
				}
				for _, file := range files {
					path := filepath.Join(dir, file)
					assert.FileExists(t, path, "%s should exist", file)
				}

				// Verify config content
				cfg, err := config.LoadProject(filepath.Join(dir, ".guild", "guild.yaml"))
				require.NoError(t, err)
				assert.NotEmpty(t, cfg.Agents, "Should have default agents")
			},
		},
		{
			name: "init_with_provider",
			args: []string{"init", "--provider", "openai"},
			validateFunc: func(t *testing.T, dir string, result *testutil.CommandResult) {
				cfg, err := config.LoadProject(filepath.Join(dir, ".guild", "guild.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "openai", cfg.DefaultProvider)
			},
		},
		{
			name: "init_in_existing_project",
			setupFunc: func(t *testing.T, dir string) {
				// Create existing .guild directory
				err := os.MkdirAll(filepath.Join(dir, ".guild"), 0755)
				require.NoError(t, err)
			},
			args:         []string{"init"},
			wantError:    true,
			wantInOutput: "already initialized",
		},
		{
			name: "init_with_force",
			setupFunc: func(t *testing.T, dir string) {
				// Create existing .guild directory
				err := os.MkdirAll(filepath.Join(dir, ".guild"), 0755)
				require.NoError(t, err)
			},
			args: []string{"init", "--force"},
			validateFunc: func(t *testing.T, dir string, result *testutil.CommandResult) {
				guildDir := filepath.Join(dir, ".guild")
				assert.DirExists(t, guildDir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test environment
			projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
				Name:         tt.name,
				SkipDatabase: true, // We're testing init itself
			})
			defer cleanup()

			extCtx := testutil.ExtendProjectContext(projCtx)

			// Setup if needed
			if tt.setupFunc != nil {
				tt.setupFunc(t, projCtx.GetRootPath())
			}

			// Run command
			result := extCtx.RunGuild(tt.args...)

			// Check error expectation
			if tt.wantError {
				assert.Error(t, result.Error, "Expected error but got none")
				if tt.wantInOutput != "" {
					assert.Contains(t, result.Stderr, tt.wantInOutput)
				}
			} else {
				assert.NoError(t, result.Error, "Unexpected error: %v", result.Error)
			}

			// Run validation
			if tt.validateFunc != nil && !tt.wantError {
				tt.validateFunc(t, projCtx.GetRootPath(), result)
			}
		})
	}
}

// TestGuildInitPerformance validates initialization performance
func TestGuildInitPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:         "init-performance",
		SkipDatabase: true,
	})
	defer cleanup()

	extCtx := testutil.ExtendProjectContext(projCtx)

	// Measure init time
	start := time.Now()
	result := extCtx.RunGuild("init")
	duration := time.Since(start)

	require.NoError(t, result.Error)

	// Performance requirement: init should complete within 2 seconds
	assert.LessOrEqual(t, duration, 2*time.Second,
		"Guild init should complete within 2 seconds, took %v", duration)

	t.Logf("Guild init completed in %v", duration)
}

// TestProviderConfiguration validates provider setup during init
func TestProviderConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	providers := []string{"openai", "anthropic", "ollama", "deepseek"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
				Name:         "provider-" + provider,
				SkipDatabase: true,
			})
			defer cleanup()

			extCtx := testutil.ExtendProjectContext(projCtx)

			// Initialize with provider
			result := extCtx.RunGuild("init", "--provider", provider)
			require.NoError(t, result.Error)

			// Verify provider configuration
			cfg, err := config.LoadProject(filepath.Join(projCtx.GetRootPath(), ".guild", "guild.yaml"))
			require.NoError(t, err)
			assert.Equal(t, provider, cfg.DefaultProvider)

			// Test provider validation
			result = extCtx.RunGuild("status")
			assert.NoError(t, result.Error)
			assert.Contains(t, result.Stdout, provider)
		})
	}
}

// TestProjectDetection validates guild recognizes existing projects
func TestProjectDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test various project types
	projectMarkers := []struct {
		name    string
		files   []string
		wantMsg string
	}{
		{
			name:    "go_project",
			files:   []string{"go.mod", "main.go"},
			wantMsg: "Detected Go project",
		},
		{
			name:    "python_project",
			files:   []string{"requirements.txt", "setup.py"},
			wantMsg: "Detected Python project",
		},
		{
			name:    "node_project",
			files:   []string{"package.json", "index.js"},
			wantMsg: "Detected Node.js project",
		},
	}

	for _, pm := range projectMarkers {
		t.Run(pm.name, func(t *testing.T) {
			projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
				Name:         pm.name,
				SkipDatabase: true,
			})
			defer cleanup()

			extCtx := testutil.ExtendProjectContext(projCtx)

			// Create project files
			for _, file := range pm.files {
				path := filepath.Join(projCtx.GetRootPath(), file)
				err := os.WriteFile(path, []byte("test content"), 0644)
				require.NoError(t, err)
			}

			// Initialize
			result := extCtx.RunGuild("init")
			require.NoError(t, result.Error)

			// Check for project detection message
			if pm.wantMsg != "" {
				assert.Contains(t, result.Stdout, pm.wantMsg)
			}
		})
	}
}
