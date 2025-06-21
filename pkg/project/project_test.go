// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "myproject")
	subDir := filepath.Join(projectDir, "subdir", "deep")
	guildDir := filepath.Join(projectDir, ".campaign")

	// Create directories
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}
	if err := os.MkdirAll(guildDir, 0755); err != nil {
		t.Fatalf("Failed to create .campaign directory: %v", err)
	}

	tests := []struct {
		name      string
		startPath string
		wantRoot  string
		wantErr   bool
	}{
		{
			name:      "In project root",
			startPath: projectDir,
			wantRoot:  projectDir,
			wantErr:   false,
		},
		{
			name:      "In project subdirectory",
			startPath: subDir,
			wantRoot:  projectDir,
			wantErr:   false,
		},
		{
			name:      "Not in project",
			startPath: tempDir,
			wantRoot:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoot, err := FindProjectRoot(tt.startPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && gotRoot != tt.wantRoot {
				t.Errorf("FindProjectRoot() = %v, want %v", gotRoot, tt.wantRoot)
			}
		})
	}
}

func TestIsInitialized(t *testing.T) {
	tempDir := t.TempDir()

	// Test non-initialized directory
	if IsInitialized(tempDir) {
		t.Error("IsInitialized() returned true for non-initialized directory")
	}

	// Create .campaign directory
	guildDir := filepath.Join(tempDir, ".campaign")
	if err := os.MkdirAll(guildDir, 0755); err != nil {
		t.Fatalf("Failed to create .campaign directory: %v", err)
	}

	// Test initialized directory
	if !IsInitialized(tempDir) {
		t.Error("IsInitialized() returned false for initialized directory")
	}
}

func TestInitialize(t *testing.T) {
	tempDir := t.TempDir()

	// Test initialization
	ctx := context.Background()
	if _, err := Initialize(ctx, tempDir, InitOptions{}); err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Check that all expected directories were created
	expectedDirs := []string{
		".campaign",
		".campaign/commissions",
		".campaign/commissions/refined",
		".campaign/campaigns",
		".campaign/kanban",
		".campaign/corpus",
		".campaign/corpus/index",
		".campaign/prompts",
		".campaign/tools",
		".campaign/workspaces",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(tempDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", dir)
		}
	}

	// Check that expected files were created
	expectedFiles := []string{
		".campaign/guild.yaml",
		".campaign/memory.db",
		".campaign/.gitignore",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}

	// Test that re-initialization succeeds (idempotent)
	if _, err := Initialize(ctx, tempDir, InitOptions{}); err != nil {
		t.Errorf("Re-initialization failed: %v", err)
	}
}

func TestValidateProjectPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid directory",
			path:    tempDir,
			wantErr: false,
		},
		{
			name:    "Non-existent path",
			path:    filepath.Join(tempDir, "nonexistent"),
			wantErr: true,
		},
		{
			name:    "File instead of directory",
			path:    createTempFile(t, tempDir, "test.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContextMethods(t *testing.T) {
	tempDir := t.TempDir()
	guildDir := filepath.Join(tempDir, ".campaign")

	ctx, err := NewContext(tempDir)
	if err != nil {
		t.Fatalf("NewContext() failed: %v", err)
	}

	// Test all getter methods
	tests := []struct {
		name   string
		getter func() string
		want   string
	}{
		{"GetRootPath", ctx.GetRootPath, tempDir},
		{"GetGuildPath", ctx.GetGuildPath, guildDir},
		{"GetCorpusPath", ctx.GetCorpusPath, filepath.Join(guildDir, "corpus")},
		{"GetEmbeddingsPath", ctx.GetEmbeddingsPath, filepath.Join(guildDir, "embeddings")},
		{"GetConfigPath", ctx.GetConfigPath, filepath.Join(guildDir, "config.yaml")},
		{"GetAgentsPath", ctx.GetAgentsPath, filepath.Join(guildDir, "agents")},
		{"GetCommissionsPath", ctx.GetCommissionsPath, filepath.Join(guildDir, "commissions")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.getter(); got != tt.want {
				t.Errorf("%s() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestContextPropagation(t *testing.T) {
	tempDir := t.TempDir()

	projCtx, err := NewContext(tempDir)
	if err != nil {
		t.Fatalf("NewContext() failed: %v", err)
	}

	// Test adding to context
	ctx := context.Background()
	ctxWithProject := WithContext(ctx, projCtx)

	// Test retrieving from context
	retrieved, ok := FromContext(ctxWithProject)
	if !ok {
		t.Error("FromContext() failed to retrieve project context")
	}

	if retrieved.GetRootPath() != projCtx.GetRootPath() {
		t.Errorf("Retrieved context root path = %v, want %v",
			retrieved.GetRootPath(), projCtx.GetRootPath())
	}

	// Test MustFromContext with valid context
	mustCtx := MustFromContext(ctxWithProject)
	if mustCtx.GetRootPath() != projCtx.GetRootPath() {
		t.Errorf("MustFromContext() root path = %v, want %v",
			mustCtx.GetRootPath(), projCtx.GetRootPath())
	}

	// Test MustFromContext with invalid context (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromContext() did not panic with invalid context")
		}
	}()
	_ = MustFromContext(ctx)
}

// Helper function to create a temporary file
func createTempFile(t *testing.T, dir, name string) string {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f.Close()
	return path
}
