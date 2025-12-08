//go:build integration

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/internal/testutil"
	"github.com/guild-framework/guild-core/pkg/config"
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
				// Verify .campaign directory created
				campaignDir := filepath.Join(dir, ".campaign")
				require.DirExists(t, campaignDir, ".campaign directory should exist")

				// Verify campaign.yaml created
				campaignFile := filepath.Join(campaignDir, "campaign.yaml")
				require.FileExists(t, campaignFile, "campaign.yaml should exist")

				// Verify guilds directory and elena_guild.yaml
				guildsDir := filepath.Join(campaignDir, "guilds")
				require.DirExists(t, guildsDir, "guilds directory should exist")
				elenaGuildFile := filepath.Join(guildsDir, "elena_guild.yaml")
				require.FileExists(t, elenaGuildFile, "elena_guild.yaml should exist")

				// Verify agents directory created
				agentsDir := filepath.Join(campaignDir, "agents")
				require.DirExists(t, agentsDir, "agents directory should exist")

				// Verify database created
				dbFile := filepath.Join(campaignDir, "memory.db")
				assert.FileExists(t, dbFile, "memory.db should exist")

				// Verify campaign.yaml contents (basic check)
				campaignData, err := os.ReadFile(campaignFile)
				require.NoError(t, err)
				assert.Contains(t, string(campaignData), "campaign:")
				assert.Contains(t, string(campaignData), "name:")

				// Verify at least one agent file exists
				agentFiles, err := filepath.Glob(filepath.Join(agentsDir, "*.yaml"))
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(agentFiles), 3, "Should have at least 3 agent configuration files")
			},
		},
		{
			name: "init_in_existing_project",
			setupFunc: func(t *testing.T, dir string) {
				// Create existing .campaign directory with campaign.yaml
				campaignDir := filepath.Join(dir, ".campaign")
				err := os.MkdirAll(campaignDir, 0755)
				require.NoError(t, err)
				// Create a dummy campaign.yaml
				err = os.WriteFile(filepath.Join(campaignDir, "campaign.yaml"), []byte("campaign:\n  name: test\n"), 0644)
				require.NoError(t, err)
			},
			args:         []string{"init"},
			wantError:    true,
			wantInOutput: "already initialized",
		},
		{
			name: "init_with_force",
			setupFunc: func(t *testing.T, dir string) {
				// Create existing .campaign directory with campaign.yaml
				campaignDir := filepath.Join(dir, ".campaign")
				err := os.MkdirAll(campaignDir, 0755)
				require.NoError(t, err)
				// Create a dummy campaign.yaml
				err = os.WriteFile(filepath.Join(campaignDir, "campaign.yaml"), []byte("campaign:\n  name: test\n"), 0644)
				require.NoError(t, err)
			},
			args: []string{"init", "--force"},
			validateFunc: func(t *testing.T, dir string, result *testutil.CommandResult) {
				campaignDir := filepath.Join(dir, ".campaign")
				assert.DirExists(t, campaignDir)
				// Verify the force flag worked by checking new files exist
				campaignFile := filepath.Join(campaignDir, "campaign.yaml")
				assert.FileExists(t, campaignFile)
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

			extCtx := testutil.ExtendProjectContext(t, projCtx)

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

	extCtx := testutil.ExtendProjectContext(t, projCtx)

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

			extCtx := testutil.ExtendProjectContext(t, projCtx)

			// Initialize with provider
			result := extCtx.RunGuild("init", "--provider", provider)
			require.NoError(t, result.Error)

			// Verify provider configuration
			cfg, err := config.LoadGuildConfig(context.Background(), projCtx.GetRootPath())
			require.NoError(t, err)
			// TODO: Check provider configuration once DefaultProvider is implemented
			// assert.Equal(t, provider, cfg.DefaultProvider)
			assert.NotNil(t, cfg.Providers)

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

			extCtx := testutil.ExtendProjectContext(t, projCtx)

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
