// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGoProject creates a test Go project structure
func setupGoProject(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create go.mod
	goMod := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644))

	// Create main.go
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0644))

	return tempDir
}

// TestCraftInitCommand tests the comprehensive init command functionality
func TestCraftInitCommand(t *testing.T) {
	tests := []struct {
		name          string
		projectPath   string
		expectError   bool
		checkFiles    []string
		checkContents map[string]string
		setupFunc     func(t *testing.T) string
	}{
		{
			name:        "fresh_project",
			setupFunc:   func(t *testing.T) string { return t.TempDir() },
			expectError: false,
			checkFiles: []string{
				".campaign/campaign.yaml",
				".campaign/agents/elena-guild-master.yaml",
				".campaign/memory.db",
				"commissions/.gitkeep",
			},
		},
		{
			name:        "go_project",
			setupFunc:   setupGoProject,
			expectError: false,
			checkFiles: []string{
				".campaign/campaign.yaml",
				".campaign/agents/marcus-developer.yaml",
				".campaign/memory.db",
			},
			checkContents: map[string]string{
				".campaign/agents/marcus-developer.yaml": "goroutines",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test directory
			projectPath := tt.setupFunc(t)

			// Change to project directory
			oldCwd, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(oldCwd)
				require.NoError(t, err)
			}()

			err = os.Chdir(projectPath)
			require.NoError(t, err)

			// Execute init command
			cmd := rootCmd
			cmd.SetArgs([]string{"init", "--force"})

			err = cmd.Execute()
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify project was initialized
			assert.True(t, project.IsInitialized("."), "Project should be initialized")

			// Check expected files exist
			for _, file := range tt.checkFiles {
				_, err := os.Stat(file)
				assert.NoError(t, err, "Expected file %s should exist", file)
			}

			// Check file contents
			for file, expectedContent := range tt.checkContents {
				data, err := os.ReadFile(file)
				require.NoError(t, err, "Should be able to read %s", file)
				assert.Contains(t, string(data), expectedContent, "File %s should contain expected content", file)
			}
		})
	}
}

// TestCraftInitIdempotency tests that init can be run multiple times safely
func TestCraftInitIdempotency(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldCwd)
		require.NoError(t, err)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// First initialization
	cmd1 := rootCmd
	cmd1.SetArgs([]string{"init", "--force"})
	err = cmd1.Execute()
	require.NoError(t, err)

	// Record initial state
	initialFiles := []string{
		".campaign/campaign.yaml",
		".campaign/memory.db",
	}

	initialStates := make(map[string]os.FileInfo)
	for _, file := range initialFiles {
		stat, err := os.Stat(file)
		require.NoError(t, err)
		initialStates[file] = stat
	}

	// Second initialization
	cmd2 := rootCmd
	cmd2.SetArgs([]string{"init", "--force"})
	err = cmd2.Execute()
	assert.NoError(t, err, "Second init should not error")

	// Verify files still exist and were handled properly
	for _, file := range initialFiles {
		_, err := os.Stat(file)
		assert.NoError(t, err, "File %s should still exist after second init", file)
	}
}

// TestCraftInitPermissionErrors tests handling of permission errors
func TestCraftInitPermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0444)
	require.NoError(t, err)

	// Ensure cleanup even if test fails
	defer func() {
		_ = os.Chmod(readOnlyDir, 0755)
		_ = os.RemoveAll(readOnlyDir)
	}()

	// Change to read-only directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldCwd)
		require.NoError(t, err)
	}()

	err = os.Chdir(readOnlyDir)
	if err != nil {
		// If we can't chdir to the read-only directory, skip the test
		t.Skipf("Cannot chdir to read-only directory: %v", err)
	}

	// Try to initialize in read-only directory
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--force"})
	err = cmd.Execute()

	// Should handle permission error gracefully
	assert.Error(t, err, "Init should fail in read-only directory")
}

// TestCraftInitExistingCampaign tests behavior with existing campaign
func TestCraftInitExistingCampaign(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldCwd)
		require.NoError(t, err)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create existing campaign manually
	err = os.MkdirAll(".campaign", 0755)
	require.NoError(t, err)

	existingConfig := `name: existing-campaign
version: 1.0.0
`
	err = os.WriteFile(".campaign/campaign.yaml", []byte(existingConfig), 0644)
	require.NoError(t, err)

	// Initialize with force flag
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--force"})
	err = cmd.Execute()
	assert.NoError(t, err, "Init with force should succeed with existing campaign")

	// Verify campaign file exists
	_, err = os.Stat(".campaign/campaign.yaml")
	assert.NoError(t, err, "Campaign file should exist")
}

// TestCraftInitProjectTypeDetection tests automatic project type detection
func TestCraftInitProjectTypeDetection(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     map[string]string
		expectedAgents []string
	}{
		{
			name: "go_project_detection",
			setupFiles: map[string]string{
				"go.mod":  "module test\n\ngo 1.21\n",
				"main.go": "package main\n\nfunc main() {}\n",
			},
			expectedAgents: []string{"marcus-developer.yaml"},
		},
		{
			name: "javascript_project_detection",
			setupFiles: map[string]string{
				"package.json": `{"name": "test", "version": "1.0.0"}`,
				"index.js":     "console.log('hello');\n",
			},
			expectedAgents: []string{"marcus-developer.yaml"},
		},
		{
			name: "python_project_detection",
			setupFiles: map[string]string{
				"requirements.txt": "flask==2.0.0\n",
				"app.py":           "from flask import Flask\n",
			},
			expectedAgents: []string{"marcus-developer.yaml"},
		},
		{
			name: "generic_project",
			setupFiles: map[string]string{
				"README.md": "# Test Project\n",
			},
			expectedAgents: []string{"elena-guild-master.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup project files
			for file, content := range tt.setupFiles {
				filePath := filepath.Join(tempDir, file)
				err := os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			// Change to project directory
			oldCwd, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(oldCwd)
				require.NoError(t, err)
			}()

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			// Execute init command
			cmd := rootCmd
			cmd.SetArgs([]string{"init", "--force"})
			err = cmd.Execute()
			require.NoError(t, err)

			// Check that expected agents were created
			agentsDir := ".campaign/agents"
			entries, err := os.ReadDir(agentsDir)
			require.NoError(t, err)

			agentFiles := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					agentFiles = append(agentFiles, entry.Name())
				}
			}

			for _, expectedAgent := range tt.expectedAgents {
				assert.Contains(t, agentFiles, expectedAgent, "Expected agent %s should be created", expectedAgent)
			}
		})
	}
}

func TestInitCommand(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tempDir)

	// Test init command with available flags
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--force"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	// Verify project was initialized
	if !project.IsInitialized(".") {
		t.Error("Project was not initialized")
	}

	// Check expected files exist (based on actual init command behavior)
	expectedFiles := []string{
		".campaign/campaign.yaml", // Campaign configuration
		".campaign/memory.db",     // SQLite database
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected Phase 0 file %s was not created", file)
		}
	}

	// Check that agents directory exists and has agent files
	agentsDir := ".campaign/agents"
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		t.Errorf("Expected agents directory %s was not created", agentsDir)
	} else {
		// Check that at least one agent file exists
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			t.Errorf("Failed to read agents directory: %v", err)
		} else if len(entries) == 0 {
			t.Error("Expected agent files in agents directory, but found none")
		}
	}

	// Verify guild configuration exists in guilds directory (new architecture)
	guildConfigPath := ".campaign/guilds/elena_guild.yaml"
	if _, err := os.Stat(guildConfigPath); err != nil {
		t.Errorf("Guild configuration file %s was not created: %v", guildConfigPath, err)
	} else {
		// The elena_guild.yaml file exists and contains guild configuration
		// Just verify it can be read (structure may vary)
		if data, err := os.ReadFile(guildConfigPath); err != nil {
			t.Errorf("Failed to read guild configuration file: %v", err)
		} else if len(data) == 0 {
			t.Error("Guild configuration file is empty")
		}
	}
}

func TestInitCommandAlreadyInitialized(t *testing.T) {
	// Create temp directory and initialize it
	tempDir := t.TempDir()

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tempDir)

	// Initialize first time
	_, err := project.Initialize(context.Background(), ".", project.InitOptions{})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Try to initialize again with force flag
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--force"})

	// Should not error, but should print message
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Init command failed on already initialized project: %v", err)
	}
}

func TestInitCommandWithPath(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "myproject")

	// Create the project directory
	os.MkdirAll(projectDir, 0755)

	// Run init with path and force flag
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--force", projectDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Init command with path failed: %v", err)
	}

	// Verify project was initialized at the specified path
	if !project.IsInitialized(projectDir) {
		t.Error("Project was not initialized at specified path")
	}
}
