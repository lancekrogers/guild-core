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

	// Test init command
	cmd := rootCmd
	cmd.SetArgs([]string{"init"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	// Verify project was initialized
	if !project.IsInitialized(".") {
		t.Error("Project was not initialized")
	}

	// Check expected Phase 0 hierarchical configuration files exist
	expectedFiles := []string{
		".guild/campaign.yml",       // Phase 0 campaign configuration
		".guild/guild.yml",          // Phase 0 guild definitions
		".guild/project.yaml",       // Provider and agent configuration from wizard
		".guild/memory.db",          // SQLite database
		".guild/.gitignore",         // Git ignore rules
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected Phase 0 file %s was not created", file)
		}
	}

	// Check that agents directory exists and has agent files
	agentsDir := ".guild/agents"
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

	// Try to initialize again
	cmd := rootCmd
	cmd.SetArgs([]string{"init"})

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

	// Run init with path
	cmd := rootCmd
	cmd.SetArgs([]string{"init", projectDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Init command with path failed: %v", err)
	}

	// Verify project was initialized at the specified path
	if !project.IsInitialized(projectDir) {
		t.Error("Project was not initialized at specified path")
	}
}
