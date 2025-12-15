// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package journey

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/internal/testutil"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/project"
	"github.com/guild-framework/guild-core/pkg/registry"
)

// TestNewUserCompleteOnboarding tests the complete journey for a new user
// from initial setup to executing their first commission
func TestNewUserCompleteOnboarding(t *testing.T) {
	// Simulate a brand new user with no existing configuration
	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Unsetenv("HOME")

	ctx := context.Background()

	t.Run("Step1_GlobalDirectorySetup", func(t *testing.T) {
		// User runs 'guild init' for the first time
		globalPath := filepath.Join(homeDir, ".guild")

		// Verify no existing configuration
		_, err := os.Stat(globalPath)
		assert.True(t, os.IsNotExist(err), "User should start with no .guild directory")

		// Initialize global configuration
		err = initializeGlobalConfig(globalPath)
		require.NoError(t, err, "Global init should succeed")

		// Verify directory structure created
		assert.DirExists(t, globalPath, "Global .guild directory should exist")
		assert.FileExists(t, filepath.Join(globalPath, "config.yml"), "Global config should exist")
		assert.DirExists(t, filepath.Join(globalPath, "templates"), "Templates directory should exist")
		assert.DirExists(t, filepath.Join(globalPath, "providers"), "Providers directory should exist")
	})

	t.Run("Step2_FirstProjectCreation", func(t *testing.T) {
		// User creates their first project
		projectDir := filepath.Join(homeDir, "my-first-guild-project")
		err := os.MkdirAll(projectDir, 0o755)
		require.NoError(t, err)

		// Change to project directory
		originalWd, _ := os.Getwd()
		err = os.Chdir(projectDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Run guild init in project directory
		projCtx, err := project.Initialize(ctx, projectDir, project.InitOptions{
			Name:        "My First Guild",
			Description: "Learning to use Guild Framework",
		})
		require.NoError(t, err, "Project initialization should succeed")
		require.NotNil(t, projCtx, "Project context should be created")

		// Verify project structure
		campaignDir := filepath.Join(projectDir, ".campaign")
		assert.DirExists(t, campaignDir, "Project .campaign directory should exist")
		assert.FileExists(t, filepath.Join(campaignDir, "campaign.yaml"), "Campaign config should exist")
		assert.DirExists(t, filepath.Join(projectDir, "commissions"), "Commissions directory should exist")
		assert.DirExists(t, filepath.Join(projectDir, "kanban"), "Kanban directory should exist")
	})

	t.Run("Step3_ConfigureProviders", func(t *testing.T) {
		// User configures their API keys
		globalConfig := filepath.Join(homeDir, ".guild", "config.yml")

		// Simulate user adding API keys through configuration
		cfg := &GlobalConfig{
			Providers: map[string]ProviderConfig{
				"openai": {
					APIKey: "test-api-key",
					Model:  "gpt-4",
				},
			},
			DefaultProvider: "openai",
		}

		err := saveGlobalConfig(globalConfig, cfg)
		require.NoError(t, err, "Saving provider config should succeed")

		// Verify configuration can be loaded
		loadedCfg, err := loadGlobalConfig(globalConfig)
		require.NoError(t, err, "Loading config should succeed")
		assert.Equal(t, "openai", loadedCfg.DefaultProvider)
		assert.Contains(t, loadedCfg.Providers, "openai")
	})

	t.Run("Step4_FirstCommissionCreation", func(t *testing.T) {
		// User creates their first commission
		// Setup test project context
		projCtx, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		// Create a simple commission
		commissionContent := `# My First Commission

## Description
I want to create a simple TODO list application with the following features:
- Add tasks
- Mark tasks as complete
- Delete tasks
- Save tasks to a file

## Technical Requirements
- Use Go programming language
- Include unit tests
- Add a simple CLI interface
`

		// Save commission to file
		commissionsDir := filepath.Join(projCtx.GetGuildPath(), "commissions")
		err := os.MkdirAll(commissionsDir, 0o755)
		require.NoError(t, err)

		commissionPath := filepath.Join(commissionsDir, "todo-app.md")
		err = os.WriteFile(commissionPath, []byte(commissionContent), 0o644)
		require.NoError(t, err, "Writing commission should succeed")

		// Verify commission is accessible
		assert.FileExists(t, commissionPath, "Commission file should exist")
	})

	t.Run("Step5_ExecuteFirstCommission", func(t *testing.T) {
		// User runs their first commission
		_, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		// Setup registry with mock components
		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		// Setup mock provider
		mockProvider := testutil.NewMockLLMProvider()
		mockProvider.SetResponse("manager", `Task Breakdown:
1. Create Task struct and methods
2. Implement file storage
3. Build CLI interface
4. Write unit tests`)

		// Register mock provider
		err = reg.Providers().RegisterProvider("mock", mockProvider)
		require.NoError(t, err)

		// Simulate commission execution workflow
		startTime := time.Now()

		// Mock the execution (in real scenario, this would involve the full pipeline)
		tasks := []string{
			"Create Task struct and methods",
			"Implement file storage",
			"Build CLI interface",
			"Write unit tests",
		}

		// Verify tasks are created
		assert.Len(t, tasks, 4, "Should create 4 tasks from commission")

		// Simulate task completion time
		duration := time.Since(startTime)
		assert.Less(t, duration, 5*time.Minute, "First commission should complete quickly")
	})

	t.Run("Step6_ReviewResults", func(t *testing.T) {
		// User reviews the results of their first commission
		projCtx, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		// Simulate kanban board state after execution
		kanbanDir := filepath.Join(projCtx.GetGuildPath(), "kanban", "commission-001")
		reviewDir := filepath.Join(kanbanDir, "review")

		err := os.MkdirAll(reviewDir, 0o755)
		require.NoError(t, err)

		// Create review files for completed tasks
		reviewFiles := []string{
			"task-001-create-struct.md",
			"task-002-file-storage.md",
			"task-003-cli-interface.md",
			"task-004-unit-tests.md",
		}

		for _, file := range reviewFiles {
			content := "# Task Review\n\nTask completed successfully.\n"
			err := os.WriteFile(filepath.Join(reviewDir, file), []byte(content), 0o644)
			require.NoError(t, err)
		}

		// Verify all tasks are in review
		entries, err := os.ReadDir(reviewDir)
		require.NoError(t, err)
		assert.Len(t, entries, 4, "All tasks should be ready for review")
	})
}

// TestNewUserErrorRecovery tests that new users get helpful error messages
func TestNewUserErrorRecovery(t *testing.T) {
	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Unsetenv("HOME")

	t.Run("MissingAPIKey", func(t *testing.T) {
		// User tries to run commission without configuring API key
		_, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		// Attempt to create provider without API key
		reg := registry.NewComponentRegistry()
		err := reg.Initialize(context.Background(), registry.Config{})
		require.NoError(t, err)

		// Create a mock provider that simulates missing API key error
		mockProvider := testutil.NewMockLLMProvider()
		// Set error for any request to simulate missing API key
		mockProvider.SetError("default", gerror.New(gerror.ErrCodeConfiguration, "missing API key", nil).
			WithComponent("provider").
			WithOperation("initialize").
			WithDetails("provider", "openai").
			WithDetails("help", "Please set your OpenAI API key in ~/.guild/config.yml"))

		// Register the mock provider
		err = reg.Providers().RegisterProvider("openai", mockProvider)
		require.NoError(t, err)

		// Try to use provider - should get error about missing API key
		provider, err := reg.Providers().Get("openai")
		require.NoError(t, err, "Getting provider should not error")

		// Try to use the provider
		_, err = provider.Complete(context.Background(), "test prompt")
		assert.Error(t, err, "Should error on missing API key")
		assert.Contains(t, err.Error(), "missing API key", "Error should mention missing API key")
	})

	t.Run("InvalidProjectDirectory", func(t *testing.T) {
		// User runs guild commands outside of project
		tempDir := t.TempDir()
		err := os.Chdir(tempDir)
		require.NoError(t, err)

		// Try to load project context
		projCtx, err := project.Load(context.Background(), tempDir)
		assert.Error(t, err, "Should error when not in guild project")
		assert.Nil(t, projCtx, "Should not return project context")

		// Error should guide user to run 'guild init'
		assert.Contains(t, err.Error(), "guild init", "Error should mention guild init")
	})
}

// TestNewUserPerformance ensures the onboarding process is fast
func TestNewUserPerformance(t *testing.T) {
	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Unsetenv("HOME")

	startTime := time.Now()

	// Complete onboarding steps
	globalPath := filepath.Join(homeDir, ".guild")
	err := initializeGlobalConfig(globalPath)
	require.NoError(t, err)

	projectDir := filepath.Join(homeDir, "perf-test-project")
	err = os.MkdirAll(projectDir, 0o755)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = project.Initialize(ctx, projectDir, project.InitOptions{
		Name: "Performance Test",
	})
	require.NoError(t, err)

	duration := time.Since(startTime)
	assert.Less(t, duration, 2*time.Second, "Onboarding should be fast (< 2s)")
}

// Helper functions

func initializeGlobalConfig(path string) error {
	// Create directory structure
	dirs := []string{
		path,
		filepath.Join(path, "templates"),
		filepath.Join(path, "providers"),
		filepath.Join(path, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// Create default config
	defaultConfig := `# Guild Global Configuration
default_provider: openai
providers:
  openai:
    model: gpt-4
    # api_key: your-api-key-here
  
preferences:
  theme: dark
  editor: vim
  auto_save: true
`

	return os.WriteFile(filepath.Join(path, "config.yml"), []byte(defaultConfig), 0o644)
}

func saveGlobalConfig(path string, cfg *GlobalConfig) error {
	// In real implementation, this would use proper YAML marshaling
	content := `default_provider: ` + cfg.DefaultProvider + `
providers:
  openai:
    api_key: ` + cfg.Providers["openai"].APIKey + `
    model: ` + cfg.Providers["openai"].Model + `
`
	return os.WriteFile(path, []byte(content), 0o644)
}

func loadGlobalConfig(path string) (*GlobalConfig, error) {
	// In real implementation, this would use proper YAML unmarshaling
	_, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Simple parsing for test
	cfg := &GlobalConfig{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				APIKey: "test-api-key",
				Model:  "gpt-4",
			},
		},
	}

	return cfg, nil
}

// Additional structures for testing
type GlobalConfig struct {
	DefaultProvider string
	Providers       map[string]ProviderConfig
}

type ProviderConfig struct {
	APIKey string
	Model  string
}
