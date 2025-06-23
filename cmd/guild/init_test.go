// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/project"
)

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
