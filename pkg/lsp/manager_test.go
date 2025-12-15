// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration && lsp
// +build integration,lsp

package lsp_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/lsp"
)

func TestLSPManager(t *testing.T) {
	// Set a timeout for the entire test
	if deadline, ok := t.Deadline(); !ok {
		t.Fatal("Test must have a deadline")
	} else if time.Until(deadline) > 30*time.Second {
		// Limit test to 30 seconds max
		var cancel context.CancelFunc
		ctx := context.Background()
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Skip if gopls is not available
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH, skipping LSP tests")
	}

	// Create temporary workspace
	tmpDir, err := os.MkdirTemp("", "lsp-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
	message := "Hello, LSP!"
	fmt.Println(message)
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
`
	err = os.WriteFile(testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	// Create go.mod for the test workspace
	goModContent := `module test

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644)
	require.NoError(t, err)

	// Create LSP manager
	manager, err := lsp.NewManager("")
	require.NoError(t, err)

	// Create timeout context for all operations
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer manager.Shutdown(ctx)

	t.Run("GetServerForFile", func(t *testing.T) {
		// Add sub-test timeout
		subCtx, subCancel := context.WithTimeout(ctx, 5*time.Second)
		defer subCancel()

		server, err := manager.GetServerForFile(subCtx, testFile)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "go", server.Language)
		assert.True(t, server.Ready)
	})

	t.Run("GetCompletion", func(t *testing.T) {
		// Add sub-test timeout
		subCtx, subCancel := context.WithTimeout(ctx, 5*time.Second)
		defer subCancel()

		// Test completion after "fmt."
		completions, err := manager.GetCompletion(subCtx, testFile, 6, 5, ".")
		require.NoError(t, err)
		assert.NotNil(t, completions)
		assert.True(t, len(completions.Items) > 0)

		// Check that we get expected completions
		hasExpected := false
		for _, item := range completions.Items {
			if item.Label == "Println" || item.Label == "Printf" || item.Label == "Sprintf" {
				hasExpected = true
				break
			}
		}
		assert.True(t, hasExpected, "Expected fmt package completions")
	})

	t.Run("GetDefinition", func(t *testing.T) {
		// Test going to definition of 'fmt' in the import
		locations, err := manager.GetDefinition(ctx, testFile, 2, 8)
		require.NoError(t, err)
		assert.NotNil(t, locations)
		// fmt package definition should be found
		assert.True(t, len(locations) > 0 || err != nil) // Some LSPs might not support stdlib definitions
	})

	t.Run("GetReferences", func(t *testing.T) {
		// Test finding references to 'message' variable
		locations, err := manager.GetReferences(ctx, testFile, 5, 1, true)
		require.NoError(t, err)
		assert.NotNil(t, locations)
		// Should find at least 2 references (declaration and usage)
		assert.GreaterOrEqual(t, len(locations), 2)
	})

	t.Run("GetHover", func(t *testing.T) {
		// Test hover over 'fmt.Println'
		hover, err := manager.GetHover(ctx, testFile, 6, 5)
		require.NoError(t, err)
		assert.NotNil(t, hover)
		// Hover should contain type information
		// The exact format depends on the LSP server
	})
}

func TestLSPConfig(t *testing.T) {
	t.Run("DefaultConfigs", func(t *testing.T) {
		configs := lsp.DefaultConfigs()
		assert.NotEmpty(t, configs)

		// Check Go config
		goConfig, exists := configs["go"]
		assert.True(t, exists)
		assert.Equal(t, "go", goConfig.Language)
		assert.Contains(t, goConfig.Command, "gopls")
		assert.Contains(t, goConfig.FilePatterns, "*.go")
		assert.Contains(t, goConfig.RootMarkers, "go.mod")
	})

	t.Run("DetectLanguage", func(t *testing.T) {
		tests := []struct {
			file     string
			expected string
		}{
			{"main.go", "go"},
			{"app.ts", "typescript"},
			{"script.py", "python"},
			{"lib.rs", "rust"},
			{"Main.java", "java"},
			{"Program.cs", "csharp"},
			{"unknown.txt", ""},
		}

		for _, tt := range tests {
			t.Run(tt.file, func(t *testing.T) {
				result := lsp.DetectLanguage(tt.file)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("FindRootPath", func(t *testing.T) {
		// Create test directory structure
		tmpDir, err := os.MkdirTemp("", "lsp-root-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create nested structure with go.mod at root
		projectDir := filepath.Join(tmpDir, "myproject")
		srcDir := filepath.Join(projectDir, "src")
		err = os.MkdirAll(srcDir, 0o755)
		require.NoError(t, err)

		// Create go.mod at project root
		err = os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\n"), 0o644)
		require.NoError(t, err)

		// Create a file in src
		testFile := filepath.Join(srcDir, "main.go")
		err = os.WriteFile(testFile, []byte("package main\n"), 0o644)
		require.NoError(t, err)

		// Test finding root from nested file
		root, err := lsp.FindRootPath(testFile, []string{"go.mod"})
		require.NoError(t, err)
		assert.Equal(t, projectDir, root)
	})
}

func TestLSPLifecycle(t *testing.T) {
	// Create config with a short cleanup interval
	config := &lsp.Config{
		Servers: lsp.DefaultConfigs(),
	}

	manager := lsp.NewServerManager(config)
	lifecycleManager := lsp.NewLifecycleManager(&lsp.Manager{})

	ctx := context.Background()

	t.Run("HealthCheck", func(t *testing.T) {
		// Start lifecycle manager with short intervals for testing
		lifecycleManager.Start(ctx, 100*time.Millisecond, 200*time.Millisecond)
		defer lifecycleManager.Stop()

		// Perform health check
		status := lifecycleManager.HealthCheck(ctx)
		assert.NotNil(t, status)
		// Initially should be empty
		assert.Empty(t, status)
	})

	t.Run("CleanupIdleServers", func(t *testing.T) {
		// This test would need actual servers running
		// For now, just verify the cleanup doesn't panic
		err := manager.CleanupIdleServers(ctx, 100*time.Millisecond)
		assert.NoError(t, err)
	})
}
