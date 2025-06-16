// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package fs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobTool_Name(t *testing.T) {
	tool := NewGlobTool("")
	assert.Equal(t, "glob", tool.Name())
}

func TestGlobTool_Description(t *testing.T) {
	tool := NewGlobTool("")
	desc := tool.Description()
	assert.Contains(t, desc, "file pattern matching")
	assert.Contains(t, desc, "glob patterns")
}

func TestGlobTool_Category(t *testing.T) {
	tool := NewGlobTool("")
	assert.Equal(t, "filesystem", tool.Category())
}

func TestGlobTool_RequiresAuth(t *testing.T) {
	tool := NewGlobTool("")
	assert.False(t, tool.RequiresAuth())
}

func TestGlobTool_Schema(t *testing.T) {
	tool := NewGlobTool("")
	schema := tool.Schema()

	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	// Check pattern property
	pattern, ok := properties["pattern"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", pattern["type"])
	assert.Contains(t, pattern["description"].(string), "Glob pattern")

	// Check path property
	path, ok := properties["path"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", path["type"])

	// Check exclude property
	exclude, ok := properties["exclude"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "array", exclude["type"])

	// Check required fields
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "pattern")
}

func TestGlobTool_Examples(t *testing.T) {
	tool := NewGlobTool("")
	examples := tool.Examples()

	assert.NotEmpty(t, examples)

	// Verify examples are valid JSON
	for _, example := range examples {
		var input GlobToolInput
		err := json.Unmarshal([]byte(example), &input)
		assert.NoError(t, err, "Example should be valid JSON: %s", example)
		assert.NotEmpty(t, input.Pattern, "Example should have pattern: %s", example)
	}
}

func TestGlobTool_Execute_InvalidInput(t *testing.T) {
	tool := NewGlobTool("")
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			input:       "invalid json",
			expectError: true,
		},
		{
			name:        "missing pattern",
			input:       `{"path": "/tmp"}`,
			expectError: true,
		},
		{
			name:        "empty pattern",
			input:       `{"pattern": ""}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if result != nil {
					assert.False(t, result.Success)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, result.Success)
			}
		})
	}
}

func TestGlobTool_Execute_WithTestData(t *testing.T) {
	// Create temporary directory with test files
	tempDir, err := os.MkdirTemp("", "glob_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test file structure
	testFiles := []string{
		"main.go",
		"utils.go",
		"src/app.js",
		"src/components/header.tsx",
		"src/components/footer.tsx",
		"tests/main_test.go",
		"tests/utils_test.go",
		"docs/README.md",
		"docs/api.md",
		"node_modules/package/index.js",
		".git/config",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Sleep briefly to ensure different modification times
		time.Sleep(1 * time.Millisecond)
	}

	tool := NewGlobTool(tempDir)
	ctx := context.Background()

	tests := []struct {
		name          string
		input         GlobToolInput
		expectedCount int
		shouldContain []string
		shouldExclude []string
	}{
		{
			name: "find all Go files",
			input: GlobToolInput{
				Pattern: "**/*.go",
			},
			expectedCount: 4,
			shouldContain: []string{"main.go", "utils.go", "tests/main_test.go", "tests/utils_test.go"},
		},
		{
			name: "find JavaScript files",
			input: GlobToolInput{
				Pattern: "**/*.js",
			},
			expectedCount: 2,
			shouldContain: []string{"src/app.js", "node_modules/package/index.js"},
		},
		{
			name: "find TypeScript React files",
			input: GlobToolInput{
				Pattern: "**/*.tsx",
			},
			expectedCount: 2,
			shouldContain: []string{"src/components/header.tsx", "src/components/footer.tsx"},
		},
		{
			name: "find Markdown files",
			input: GlobToolInput{
				Pattern: "**/*.md",
			},
			expectedCount: 2,
			shouldContain: []string{"docs/README.md", "docs/api.md"},
		},
		{
			name: "find all files in src directory",
			input: GlobToolInput{
				Pattern: "src/**/*",
			},
			expectedCount: 3,
			shouldContain: []string{"src/app.js", "src/components/header.tsx", "src/components/footer.tsx"},
		},
		{
			name: "find JavaScript files with exclusions",
			input: GlobToolInput{
				Pattern: "**/*.js",
				Exclude: []string{"node_modules/**"},
			},
			expectedCount: 1,
			shouldContain: []string{"src/app.js"},
			shouldExclude: []string{"node_modules/package/index.js"},
		},
		{
			name: "find test files",
			input: GlobToolInput{
				Pattern: "**/*_test.go",
			},
			expectedCount: 2,
			shouldContain: []string{"tests/main_test.go", "tests/utils_test.go"},
		},
		{
			name: "find files with multiple exclusions",
			input: GlobToolInput{
				Pattern: "**/*",
				Exclude: []string{"node_modules/**", ".git/**", "**/*.md"},
			},
			expectedCount: 7, // All files except node_modules, .git, and .md files
			shouldExclude: []string{"node_modules/package/index.js", ".git/config", "docs/README.md", "docs/api.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := tool.Execute(ctx, string(inputJSON))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.Success)

			// Parse the result
			var globResult GlobResult
			err = json.Unmarshal([]byte(result.Output), &globResult)
			require.NoError(t, err)

			// Check count
			assert.Equal(t, tt.expectedCount, globResult.Count, "Expected %d files, got %d", tt.expectedCount, globResult.Count)
			assert.Equal(t, tt.expectedCount, len(globResult.Files))

			// Check pattern and search dir
			assert.Equal(t, tt.input.Pattern, globResult.Pattern)
			assert.Equal(t, tempDir, globResult.SearchDir)

			// Collect relative paths for easier checking
			foundPaths := make([]string, len(globResult.Files))
			for i, file := range globResult.Files {
				foundPaths[i] = file.RelativePath
			}

			// Check that expected files are found
			for _, expectedFile := range tt.shouldContain {
				assert.Contains(t, foundPaths, expectedFile, "Expected to find %s", expectedFile)
			}

			// Check that excluded files are not found
			for _, excludedFile := range tt.shouldExclude {
				assert.NotContains(t, foundPaths, excludedFile, "Should not find %s", excludedFile)
			}

			// Verify files are sorted by modification time (newest first)
			if len(globResult.Files) > 1 {
				for i := 0; i < len(globResult.Files)-1; i++ {
					assert.True(t,
						globResult.Files[i].ModTime.After(globResult.Files[i+1].ModTime) ||
							globResult.Files[i].ModTime.Equal(globResult.Files[i+1].ModTime),
						"Files should be sorted by modification time (newest first)")
				}
			}

			// Verify metadata
			assert.Equal(t, tt.input.Pattern, result.Metadata["pattern"])
			assert.Equal(t, tempDir, result.Metadata["search_dir"])
			assert.Equal(t, string(rune(tt.expectedCount+'0')), result.Metadata["count"])
		})
	}
}

func TestGlobTool_Execute_WithCustomPath(t *testing.T) {
	// Create temporary directory with subdirectory
	tempDir, err := os.MkdirTemp("", "glob_tool_path_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create test files in subdirectory
	testFile := filepath.Join(subDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main"), 0644)
	require.NoError(t, err)

	tool := NewGlobTool(tempDir)
	ctx := context.Background()

	input := GlobToolInput{
		Pattern: "*.go",
		Path:    "subdir",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var globResult GlobResult
	err = json.Unmarshal([]byte(result.Output), &globResult)
	require.NoError(t, err)

	assert.Equal(t, 1, globResult.Count)
	assert.Equal(t, "test.go", globResult.Files[0].RelativePath)
}

func TestGlobTool_Execute_NonExistentPath(t *testing.T) {
	tool := NewGlobTool("")
	ctx := context.Background()

	input := GlobToolInput{
		Pattern: "*.go",
		Path:    "/nonexistent/path",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, string(inputJSON))
	assert.Error(t, err)
	if result != nil {
		assert.False(t, result.Success)
	}
}

func TestGlobTool_SanitizePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glob_tool_sanitize_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewGlobTool(tempDir)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "relative path",
			input:    "subdir",
			expected: filepath.Join(tempDir, "subdir"),
		},
		{
			name:     "current directory",
			input:    ".",
			expected: tempDir,
		},
		{
			name:     "path within base",
			input:    filepath.Join(tempDir, "subdir"),
			expected: filepath.Join(tempDir, "subdir"),
		},
		{
			name:     "path outside base (should be empty)",
			input:    "/etc/passwd",
			expected: "",
		},
		{
			name:     "path traversal attempt (should be empty)",
			input:    "../../../etc/passwd",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobTool_MatchPattern(t *testing.T) {
	tool := NewGlobTool("")

	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "simple match",
			path:     "test.go",
			pattern:  "*.go",
			expected: true,
		},
		{
			name:     "simple no match",
			path:     "test.js",
			pattern:  "*.go",
			expected: false,
		},
		{
			name:     "recursive match",
			path:     "src/main.go",
			pattern:  "**/*.go",
			expected: true,
		},
		{
			name:     "deep recursive match",
			path:     "src/components/app/main.go",
			pattern:  "**/*.go",
			expected: true,
		},
		{
			name:     "specific directory match",
			path:     "src/main.go",
			pattern:  "src/*.go",
			expected: true,
		},
		{
			name:     "specific directory no match",
			path:     "tests/main.go",
			pattern:  "src/*.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.matchPattern(tt.path, tt.pattern)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobTool_FileMatchMetadata(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "glob_tool_metadata_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.go")
	testContent := "package main\n\nfunc main() {}\n"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tool := NewGlobTool(tempDir)
	ctx := context.Background()

	input := GlobToolInput{
		Pattern: "*.go",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var globResult GlobResult
	err = json.Unmarshal([]byte(result.Output), &globResult)
	require.NoError(t, err)

	require.Equal(t, 1, len(globResult.Files))
	file := globResult.Files[0]

	assert.Equal(t, testFile, file.Path)
	assert.Equal(t, "test.go", file.RelativePath)
	assert.Equal(t, int64(len(testContent)), file.Size)
	assert.False(t, file.IsDir)
	assert.NotZero(t, file.ModTime)
}

func TestGlobTool_EmptyResult(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glob_tool_empty_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewGlobTool(tempDir)
	ctx := context.Background()

	input := GlobToolInput{
		Pattern: "*.nonexistent",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := tool.Execute(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var globResult GlobResult
	err = json.Unmarshal([]byte(result.Output), &globResult)
	require.NoError(t, err)

	assert.Equal(t, 0, globResult.Count)
	assert.Empty(t, globResult.Files)
	assert.Equal(t, "*.nonexistent", globResult.Pattern)
}

func TestGlobTool_Integration(t *testing.T) {
	// Test that we can use the tool with the tools.ToolRegistry
	registry := tools.NewToolRegistry()

	tool := NewGlobTool("")
	err := registry.RegisterTool(tool)
	require.NoError(t, err)

	// Verify tool is registered
	retrievedTool, exists := registry.GetTool("glob")
	assert.True(t, exists)
	assert.Equal(t, tool, retrievedTool)

	// Verify it's in the right category
	categoryTools := registry.ListToolsByCategory("filesystem")
	found := false
	for _, t := range categoryTools {
		if t.Name() == "glob" {
			found = true
			break
		}
	}
	assert.True(t, found, "Glob tool should be in filesystem category")
}
