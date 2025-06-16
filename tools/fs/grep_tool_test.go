// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftGrepTool tests the creation of a new grep tool
func TestCraftGrepTool(t *testing.T) {
	tests := []struct {
		name         string
		basePath     string
		expectExists bool
	}{
		{
			name:         "with valid base path",
			basePath:     t.TempDir(),
			expectExists: true,
		},
		{
			name:         "with empty base path uses cwd",
			basePath:     "",
			expectExists: true,
		},
		{
			name:         "with non-existent base path creates it",
			basePath:     filepath.Join(t.TempDir(), "new", "path"),
			expectExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewGrepTool(tt.basePath)

			assert.NotNil(t, tool)
			assert.Equal(t, "grep", tool.Name())
			assert.Equal(t, "filesystem", tool.Category())
			assert.False(t, tool.RequiresAuth())
			assert.NotEmpty(t, tool.Description())
			assert.NotEmpty(t, tool.Examples())
			assert.NotNil(t, tool.Schema())

			// Check schema structure
			schema := tool.Schema()
			assert.Equal(t, "object", schema["type"])

			props, ok := schema["properties"].(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, props, "pattern")
			assert.Contains(t, props, "include")
			assert.Contains(t, props, "path")

			required, ok := schema["required"].([]string)
			require.True(t, ok)
			assert.Contains(t, required, "pattern")
		})
	}
}

// TestJourneymanGrepExecution tests the execution of grep searches
func TestJourneymanGrepExecution(t *testing.T) {
	// Create test directory structure
	testDir := t.TempDir()

	// Create test files with various content
	testFiles := map[string]struct {
		content string
		modTime time.Time
	}{
		"hello.txt": {
			content: "Hello, World!\nThis is a test file.\nTODO: fix this later",
			modTime: time.Now().Add(-2 * time.Hour),
		},
		"code.js": {
			content: "function greet() {\n  console.log('Hello');\n  // TODO: add more features\n}\n\nfunction test() {\n  return 42;\n}",
			modTime: time.Now().Add(-1 * time.Hour),
		},
		"data.json": {
			content: `{"name": "test", "todo": "implement feature", "items": [1, 2, 3]}`,
			modTime: time.Now().Add(-30 * time.Minute),
		},
		"src/app.ts": {
			content: "import React from 'react';\n\nclass App {\n  constructor() {\n    // TODO: initialize\n  }\n}",
			modTime: time.Now().Add(-15 * time.Minute),
		},
		"src/utils.ts": {
			content: "export function calculate(x: number): number {\n  return x * 2;\n}",
			modTime: time.Now(),
		},
		"test/app_test.go": {
			content: "func TestScribeDocumentation(t *testing.T) {\n  // Test implementation\n}\n\nfunc TestGuildCoordination(t *testing.T) {\n  // Test coordination\n}",
			modTime: time.Now().Add(-45 * time.Minute),
		},
	}

	// Create test files
	for path, file := range testFiles {
		fullPath := filepath.Join(testDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(file.content), 0644)
		require.NoError(t, err)
		// Set modification time
		err = os.Chtimes(fullPath, file.modTime, file.modTime)
		require.NoError(t, err)
	}

	// Create a binary file to test exclusion
	binaryPath := filepath.Join(testDir, "binary.exe")
	err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02, 0x03, 0xFF}, 0644)
	require.NoError(t, err)

	tool := NewGrepTool(testDir)
	ctx := context.Background()

	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectCount   int
		validateFiles func(t *testing.T, files []GrepMatch)
	}{
		{
			name:        "search for TODO comments",
			input:       `{"pattern": "TODO"}`,
			expectError: false,
			expectCount: 3,
			validateFiles: func(t *testing.T, files []GrepMatch) {
				// Should find in hello.txt, code.js, and src/app.ts
				paths := make(map[string]bool)
				for _, f := range files {
					paths[f.RelativePath] = true
				}
				assert.True(t, paths["hello.txt"])
				assert.True(t, paths["code.js"])
				assert.True(t, paths["src/app.ts"])
			},
		},
		{
			name:        "search with regex pattern",
			input:       `{"pattern": "function\\s+\\w+"}`,
			expectError: false,
			expectCount: 2,
			validateFiles: func(t *testing.T, files []GrepMatch) {
				// Should find in code.js and src/utils.ts
				paths := make(map[string]bool)
				for _, f := range files {
					paths[f.RelativePath] = true
				}
				assert.True(t, paths["code.js"])
				assert.True(t, paths["src/utils.ts"])
			},
		},
		{
			name:        "search with file include pattern",
			input:       `{"pattern": "import", "include": "*.ts"}`,
			expectError: false,
			expectCount: 1,
			validateFiles: func(t *testing.T, files []GrepMatch) {
				// Should only find in *.ts files
				assert.Len(t, files, 1)
				assert.Equal(t, "src/app.ts", files[0].RelativePath)
			},
		},
		{
			name:        "search with brace expansion",
			input:       `{"pattern": "function|class", "include": "*.{js,ts}"}`,
			expectError: false,
			expectCount: 3,
			validateFiles: func(t *testing.T, files []GrepMatch) {
				paths := make(map[string]bool)
				for _, f := range files {
					paths[f.RelativePath] = true
				}
				assert.True(t, paths["code.js"])      // has function
				assert.True(t, paths["src/utils.ts"]) // has function
				assert.True(t, paths["src/app.ts"])   // has class
			},
		},
		{
			name:        "search in subdirectory",
			input:       `{"pattern": "Test", "path": "./test"}`,
			expectError: false,
			expectCount: 1,
			validateFiles: func(t *testing.T, files []GrepMatch) {
				assert.Equal(t, "app_test.go", files[0].RelativePath)
			},
		},
		{
			name:        "search with no matches",
			input:       `{"pattern": "NOTFOUND"}`,
			expectError: false,
			expectCount: 0,
		},
		{
			name:        "invalid regex pattern",
			input:       `{"pattern": "[invalid(regex"}`,
			expectError: true,
		},
		{
			name:        "empty pattern",
			input:       `{"pattern": ""}`,
			expectError: true,
		},
		{
			name:        "invalid JSON input",
			input:       `{invalid json}`,
			expectError: true,
		},
		{
			name:        "path escape attempt",
			input:       `{"pattern": "test", "path": "../../../etc"}`,
			expectError: true,
		},
		{
			name:        "non-existent directory",
			input:       `{"pattern": "test", "path": "./nonexistent"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.Success)

				// Parse the result
				var grepResult GrepResult
				err = json.Unmarshal([]byte(result.Output), &grepResult)
				require.NoError(t, err)

				assert.Equal(t, tt.expectCount, grepResult.Count)
				assert.Equal(t, tt.expectCount, len(grepResult.Files))

				if tt.validateFiles != nil {
					tt.validateFiles(t, grepResult.Files)
				}

				// Verify files are sorted by modification time (newest first)
				for i := 1; i < len(grepResult.Files); i++ {
					assert.True(t, grepResult.Files[i-1].ModTime.After(grepResult.Files[i].ModTime) ||
						grepResult.Files[i-1].ModTime.Equal(grepResult.Files[i].ModTime))
				}
			}
		})
	}
}

// TestGuildGrepBinaryExclusion tests that binary files are excluded
func TestGuildGrepBinaryExclusion(t *testing.T) {
	testDir := t.TempDir()

	// Create various binary files
	binaryFiles := map[string][]byte{
		"image.jpg":   {0xFF, 0xD8, 0xFF, 0xE0},             // JPEG header
		"program.exe": {0x4D, 0x5A, 0x90, 0x00},             // PE header
		"data.bin":    {0x00, 0x01, 0x02, 0x03, 0x00, 0x00}, // Null bytes
		"archive.zip": {0x50, 0x4B, 0x03, 0x04},             // ZIP header
	}

	// Create text files that should be searched
	textFiles := map[string]string{
		"readme.txt":  "This is a TODO item",
		"script.sh":   "#!/bin/bash\n# TODO: implement",
		"config.yaml": "# TODO: configure\nkey: value",
	}

	// Create binary files
	for name, content := range binaryFiles {
		path := filepath.Join(testDir, name)
		err := os.WriteFile(path, content, 0644)
		require.NoError(t, err)
	}

	// Create text files
	for name, content := range textFiles {
		path := filepath.Join(testDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	tool := NewGrepTool(testDir)
	ctx := context.Background()

	// Search for a pattern that would match in binary if we searched them
	result, err := tool.Execute(ctx, `{"pattern": "TODO"}`)
	require.NoError(t, err)
	require.NotNil(t, result)

	var grepResult GrepResult
	err = json.Unmarshal([]byte(result.Output), &grepResult)
	require.NoError(t, err)

	// Should only find matches in text files
	assert.Equal(t, len(textFiles), grepResult.Count)

	// Verify only text files are in results
	for _, match := range grepResult.Files {
		_, isText := textFiles[match.RelativePath]
		assert.True(t, isText, "Found unexpected file: %s", match.RelativePath)
	}
}

// TestScribeLargeFileHandling tests handling of large files
func TestScribeLargeFileHandling(t *testing.T) {
	testDir := t.TempDir()

	// Create a file larger than max size (10MB)
	largePath := filepath.Join(testDir, "large.txt")
	largeFile, err := os.Create(largePath)
	require.NoError(t, err)

	// Write 11MB of data
	pattern := []byte("This is a line with TODO item\n")
	for i := 0; i < 400000; i++ { // ~11.4MB
		_, err = largeFile.Write(pattern)
		require.NoError(t, err)
	}
	largeFile.Close()

	// Create a normal file
	normalPath := filepath.Join(testDir, "normal.txt")
	err = os.WriteFile(normalPath, []byte("This has a TODO too"), 0644)
	require.NoError(t, err)

	tool := NewGrepTool(testDir)
	ctx := context.Background()

	result, err := tool.Execute(ctx, `{"pattern": "TODO"}`)
	require.NoError(t, err)

	var grepResult GrepResult
	err = json.Unmarshal([]byte(result.Output), &grepResult)
	require.NoError(t, err)

	// Should only find the normal file
	assert.Equal(t, 1, grepResult.Count)
	assert.Equal(t, "normal.txt", grepResult.Files[0].RelativePath)
}

// TestCraftGrepContextCancellation tests context cancellation
func TestCraftGrepContextCancellation(t *testing.T) {
	testDir := t.TempDir()

	// Create many files to ensure we have time to cancel
	for i := 0; i < 1000; i++ {
		path := filepath.Join(testDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(path, []byte("TODO: test content"), 0644)
		require.NoError(t, err)
	}

	tool := NewGrepTool(testDir)

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := tool.Execute(ctx, `{"pattern": "TODO"}`)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestJourneymanGrepEdgeCases tests various edge cases
func TestJourneymanGrepEdgeCases(t *testing.T) {
	testDir := t.TempDir()

	// Create files with edge case content
	edgeCases := map[string]string{
		"empty.txt":           "",
		"unicode.txt":         "TODO: 你好世界 🌍",
		"long_line.txt":       "TODO: " + strings.Repeat("x", 5000) + " end",
		"windows_newline.txt": "Line 1\r\nTODO: Line 2\r\nLine 3",
		"no_newline.txt":      "TODO: no newline at end",
	}

	for name, content := range edgeCases {
		path := filepath.Join(testDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	tool := NewGrepTool(testDir)
	ctx := context.Background()

	result, err := tool.Execute(ctx, `{"pattern": "TODO"}`)
	require.NoError(t, err)

	var grepResult GrepResult
	err = json.Unmarshal([]byte(result.Output), &grepResult)
	require.NoError(t, err)

	// Should find matches in all files except empty.txt
	assert.Equal(t, len(edgeCases)-1, grepResult.Count)

	// Verify empty file is not in results
	for _, match := range grepResult.Files {
		assert.NotEqual(t, "empty.txt", match.RelativePath)
	}
}

// TestGuildGrepMetadata tests the metadata returned by grep
func TestGuildGrepMetadata(t *testing.T) {
	testDir := t.TempDir()

	// Create a simple test file
	err := os.WriteFile(filepath.Join(testDir, "test.txt"), []byte("TODO: test"), 0644)
	require.NoError(t, err)

	tool := NewGrepTool(testDir)
	ctx := context.Background()

	tests := []struct {
		name             string
		input            string
		expectedMetadata map[string]string
	}{
		{
			name:  "basic search",
			input: `{"pattern": "TODO"}`,
			expectedMetadata: map[string]string{
				"pattern":    "TODO",
				"search_dir": testDir,
				"count":      "1",
			},
		},
		{
			name:  "search with include",
			input: `{"pattern": "TODO", "include": "*.txt"}`,
			expectedMetadata: map[string]string{
				"pattern":    "TODO",
				"search_dir": testDir,
				"count":      "1",
				"include":    "*.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check metadata
			for key, expectedValue := range tt.expectedMetadata {
				assert.Equal(t, expectedValue, result.Metadata[key])
			}
		})
	}
}
