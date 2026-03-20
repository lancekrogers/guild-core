// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/corpus"
	"github.com/lancekrogers/guild-core/pkg/project"
)

// TestProjectIntegration tests the full project workflow
func TestProjectIntegration(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Test 1: Initialize project
	t.Run("Initialize", func(t *testing.T) {
		ctx := context.Background()
		_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
		if err != nil {
			t.Fatalf("Failed to initialize project: %v", err)
		}

		// Verify structure was created
		expectedDirs := []string{
			".campaign",
			".campaign/agents",
			".campaign/guilds",
			".campaign/memory",
			".campaign/prompts",
		}

		for _, dir := range expectedDirs {
			path := filepath.Join(tempDir, dir)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Expected directory %s was not created", dir)
			}
		}
	})

	// Test 2: Context detection
	t.Run("ContextDetection", func(t *testing.T) {
		// Change to project directory
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tempDir)

		// Get context
		ctx, err := project.GetContext()
		if err != nil {
			t.Fatalf("Failed to get project context: %v", err)
		}

		// Resolve symlinks for comparison (macOS /var -> /private/var)
		expectedPath, _ := filepath.EvalSymlinks(tempDir)
		actualPath, _ := filepath.EvalSymlinks(ctx.GetRootPath())
		if actualPath != expectedPath {
			t.Errorf("Expected root path %s, got %s", expectedPath, actualPath)
		}
	})

	// Test 3: Corpus integration
	t.Run("CorpusIntegration", func(t *testing.T) {
		ctx := context.Background()

		// Change to project directory
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tempDir)

		// Get project corpus config
		cfg, err := corpus.GetProjectConfig(ctx)
		if err != nil {
			t.Fatalf("Failed to get corpus config: %v", err)
		}

		expectedCorpusPath := filepath.Join(tempDir, ".campaign", "corpus")
		// Resolve symlinks for comparison
		expectedPath, _ := filepath.EvalSymlinks(expectedCorpusPath)
		actualPath, _ := filepath.EvalSymlinks(cfg.CorpusPath)
		if actualPath != expectedPath {
			t.Errorf("Expected corpus path %s, got %s", expectedPath, actualPath)
		}

		// Create a test document
		doc := &corpus.CorpusDoc{
			Title: "Test Document",
			Body:  "This is a test document for project integration",
			Tags:  []string{"test", "integration"},
		}

		// Save document
		err = corpus.Save(ctx, doc, cfg)
		if err != nil {
			t.Fatalf("Failed to save document: %v", err)
		}

		// List documents
		docs, err := corpus.List(ctx, cfg)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	// Test 4: Context propagation
	t.Run("ContextPropagation", func(t *testing.T) {
		// Change to project directory
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tempDir)

		// Get project context
		projCtx, err := project.GetContext()
		if err != nil {
			t.Fatalf("Failed to get project context: %v", err)
		}

		// Add to context
		ctx := context.Background()
		ctxWithProject := project.WithContext(ctx, projCtx)

		// Retrieve and verify
		retrieved, ok := project.FromContext(ctxWithProject)
		if !ok {
			t.Error("Failed to retrieve project context from context.Context")
		}

		if retrieved.GetRootPath() != projCtx.GetRootPath() {
			t.Errorf("Context propagation failed: expected %s, got %s",
				projCtx.GetRootPath(), retrieved.GetRootPath())
		}
	})

	// Test 5: Subdirectory detection
	t.Run("SubdirectoryDetection", func(t *testing.T) {
		// Create subdirectory
		subDir := filepath.Join(tempDir, "src", "components")
		os.MkdirAll(subDir, 0o755)

		// Change to subdirectory
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(subDir)

		// Should still detect project
		ctx, err := project.GetContext()
		if err != nil {
			t.Fatalf("Failed to get project context from subdirectory: %v", err)
		}

		// Resolve symlinks for comparison
		expectedPath, _ := filepath.EvalSymlinks(tempDir)
		actualPath, _ := filepath.EvalSymlinks(ctx.GetRootPath())
		if actualPath != expectedPath {
			t.Errorf("Expected root path %s from subdirectory, got %s",
				expectedPath, actualPath)
		}
	})
}

// TestProjectMigration tests the migration functionality
func TestProjectMigration(t *testing.T) {
	// Create temp directories
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Set up fake global corpus
	globalCorpusDir := filepath.Join(globalDir, "corpus", "docs")
	os.MkdirAll(globalCorpusDir, 0o755)

	// Create test files in global
	testFile := filepath.Join(globalCorpusDir, "test.md")
	content := []byte("# Test Document\n\nThis is a test.")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Initialize project
	ctx := context.Background()
	if _, err := project.Initialize(ctx, projectDir, project.InitOptions{}); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Run migration
	opts := project.MigrationOptions{
		IncludeEmbeddings: false,
		OverwriteExisting: false,
		DryRun:            false,
	}

	result, err := project.MigrateFromGlobal(ctx, projectDir, globalDir, opts)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify migration
	if result.FilesCopied != 1 {
		t.Errorf("Expected 1 file copied, got %d", result.FilesCopied)
	}

	// Check file exists in project
	projectFile := filepath.Join(projectDir, ".campaign", "corpus", "docs", "test.md")
	if _, err := os.Stat(projectFile); os.IsNotExist(err) {
		t.Error("Migrated file not found in project")
	}
}
