package fs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/guild-ventures/guild-core/tools"
)

// TestGuildGrepIntegration tests the grep tool in a realistic scenario
func TestGuildGrepIntegration(t *testing.T) {
	// Create a test project structure
	testDir := t.TempDir()
	
	// Create a realistic project structure
	projectFiles := map[string]string{
		"README.md": `# My Project

TODO: Add installation instructions
TODO: Add usage examples`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	// TODO: Add command line arguments
}`,
		"pkg/server/server.go": `package server

import (
	"net/http"
	"log"
)

// Server represents the HTTP server
type Server struct {
	port string
}

// Start starts the server
func (s *Server) Start() error {
	// TODO: Add graceful shutdown
	log.Printf("Starting server on port %s", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}`,
		"pkg/client/client.go": `package client

import "net/http"

// Client represents an HTTP client
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a new client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}`,
		"test/server_test.go": `package test

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestServerStart(t *testing.T) {
	// TODO: Implement server tests
	assert.True(t, true)
}`,
		".gitignore": `*.exe
*.dll
*.so
*.dylib
bin/
.test/`,
		"docs/api.md": `# API Documentation

## Endpoints

### GET /health
Returns the health status of the server.

TODO: Document authentication`,
	}

	// Create all files
	for path, content := range projectFiles {
		fullPath := filepath.Join(testDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create grep tool and tool registry
	grepTool := NewGrepTool(testDir)
	registry := tools.NewToolRegistry()
	err := registry.RegisterTool(grepTool)
	require.NoError(t, err)

	ctx := context.Background()

	// Test 1: Find all TODO comments
	t.Run("Find all TODOs", func(t *testing.T) {
		result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "TODO"}`)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)

		var grepResult GrepResult
		err = json.Unmarshal([]byte(result.Output), &grepResult)
		require.NoError(t, err)

		// Should find TODOs in multiple files
		assert.Equal(t, 5, grepResult.Count) // Files with TODOs: README.md, main.go, server.go, server_test.go, api.md
		fileNames := make([]string, len(grepResult.Files))
		for i, f := range grepResult.Files {
			fileNames[i] = f.RelativePath
		}
		assert.Contains(t, fileNames, "README.md")
		assert.Contains(t, fileNames, "main.go")
		assert.Contains(t, fileNames, "pkg/server/server.go")
		assert.Contains(t, fileNames, "test/server_test.go")
		assert.Contains(t, fileNames, "docs/api.md")
	})

	// Test 2: Find function definitions in Go files
	t.Run("Find Go functions", func(t *testing.T) {
		result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "^func\\s+", "include": "*.go"}`)
		require.NoError(t, err)
		require.NotNil(t, result)

		var grepResult GrepResult
		err = json.Unmarshal([]byte(result.Output), &grepResult)
		require.NoError(t, err)

		// Should find functions in Go files
		assert.Greater(t, grepResult.Count, 0)
		for _, file := range grepResult.Files {
			assert.Contains(t, file.RelativePath, ".go")
		}
	})

	// Test 3: Search for imports in specific directory
	t.Run("Find imports in pkg directory", func(t *testing.T) {
		result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "^import", "path": "./pkg"}`)
		require.NoError(t, err)
		require.NotNil(t, result)

		var grepResult GrepResult
		err = json.Unmarshal([]byte(result.Output), &grepResult)
		require.NoError(t, err)

		// Should only find imports in pkg directory
		assert.Equal(t, 2, grepResult.Count) // server.go and client.go
		// When searching in ./pkg, the relative paths are relative to pkg directory
		expectedFiles := []string{"server/server.go", "client/client.go"}
		actualFiles := make([]string, len(grepResult.Files))
		for i, file := range grepResult.Files {
			actualFiles[i] = file.RelativePath
		}
		assert.ElementsMatch(t, expectedFiles, actualFiles)
	})

	// Test 4: Use complex pattern with file filtering
	t.Run("Find types in Go source files", func(t *testing.T) {
		result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "type\\s+\\w+\\s+struct", "include": "*.go"}`)
		require.NoError(t, err)
		require.NotNil(t, result)

		var grepResult GrepResult
		err = json.Unmarshal([]byte(result.Output), &grepResult)
		require.NoError(t, err)

		// Should find struct definitions
		assert.Equal(t, 2, grepResult.Count) // Server and Client structs
	})

	// Test 5: Binary file exclusion
	t.Run("Binary files are excluded", func(t *testing.T) {
		// Create a binary file
		binaryPath := filepath.Join(testDir, "program.exe")
		err := os.WriteFile(binaryPath, []byte{0x4D, 0x5A, 0x90, 0x00, 0xFF, 0xFE}, 0755)
		require.NoError(t, err)

		// Search for a pattern that would match if we searched binary
		result, err := registry.ExecuteTool(ctx, "grep", `{"pattern": "."}`)
		require.NoError(t, err)
		require.NotNil(t, result)

		var grepResult GrepResult
		err = json.Unmarshal([]byte(result.Output), &grepResult)
		require.NoError(t, err)

		// Should not include the binary file
		for _, file := range grepResult.Files {
			assert.NotEqual(t, "program.exe", file.RelativePath)
		}
	})
}