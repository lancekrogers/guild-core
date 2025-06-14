package fs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/tools"
)

func TestNewGrepTool(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		wantName string
	}{
		{
			name:     "CraftGrepToolWithBasePath",
			basePath: "/tmp",
			wantName: "grep",
		},
		{
			name:     "CraftGrepToolWithEmptyBasePath",
			basePath: "",
			wantName: "grep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewGrepTool(tt.basePath)
			if tool == nil {
				t.Fatal("Expected non-nil tool")
			}
			if tool.Name() != tt.wantName {
				t.Errorf("Expected name %s, got %s", tt.wantName, tool.Name())
			}
			if tool.Category() != "filesystem" {
				t.Errorf("Expected category filesystem, got %s", tool.Category())
			}
			if tool.RequiresAuth() {
				t.Error("Expected tool to not require auth")
			}
		})
	}
}

func TestGrepTool_Execute(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"test1.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
	// TODO: Add error handling
}`,
		"test2.js": `function hello() {
	console.log("Hello JavaScript");
	// TODO: Add TypeScript support
}

var x = 42;`,
		"test3.py": `def hello():
	print("Hello Python")
	# TODO: Add type hints

x = 42`,
		"subdir/test4.txt": `This is a test file.
TODO: Write more content
Error: Something went wrong`,
		"binary.bin": string([]byte{0x00, 0x01, 0x02, 0x03}), // Binary file
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(tempDir, relPath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	tool := NewGrepTool(tempDir)

	tests := []struct {
		name           string
		input          string
		wantErr        bool
		wantMatchCount int
		wantFileCount  int
		validate       func(t *testing.T, result *tools.ToolResult)
	}{
		{
			name:           "JourneymanSearchTODO",
			input:          `{"pattern": "TODO"}`,
			wantErr:        false,
			wantMatchCount: 3,
			wantFileCount:  3,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "TODO") {
					t.Error("Expected output to contain TODO matches")
				}
				if !strings.Contains(result.Output, "test1.go") {
					t.Error("Expected output to contain test1.go")
				}
			},
		},
		{
			name:           "JourneymanSearchWithIncludePattern",
			input:          `{"pattern": "Hello", "include": "*.go"}`,
			wantErr:        false,
			wantMatchCount: 1,
			wantFileCount:  1,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "test1.go") {
					t.Error("Expected output to contain test1.go")
				}
				if strings.Contains(result.Output, "test2.js") {
					t.Error("Expected output to not contain test2.js")
				}
			},
		},
		{
			name:           "JourneymanSearchWithBracePattern",
			input:          `{"pattern": "Hello", "include": "*.{go,js}"}`,
			wantErr:        false,
			wantMatchCount: 2,
			wantFileCount:  2,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "test1.go") {
					t.Error("Expected output to contain test1.go")
				}
				if !strings.Contains(result.Output, "test2.js") {
					t.Error("Expected output to contain test2.js")
				}
			},
		},
		{
			name:           "JourneymanSearchWithRegex",
			input:          `{"pattern": "\\d+"}`,
			wantErr:        false,
			wantMatchCount: 2,
			wantFileCount:  2,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "42") {
					t.Error("Expected output to contain numeric matches")
				}
			},
		},
		{
			name:           "JourneymanSearchWithPath",
			input:          `{"pattern": "Error", "path": "subdir"}`,
			wantErr:        false,
			wantMatchCount: 1,
			wantFileCount:  1,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "test4.txt") {
					t.Error("Expected output to contain test4.txt")
				}
			},
		},
		{
			name:           "CraftInvalidRegexPattern",
			input:          `{"pattern": "[invalid"}`,
			wantErr:        true,
			wantMatchCount: 0,
			wantFileCount:  0,
		},
		{
			name:           "CraftEmptyPattern",
			input:          `{"pattern": ""}`,
			wantErr:        true,
			wantMatchCount: 0,
			wantFileCount:  0,
		},
		{
			name:           "JourneymanSearchNoMatches",
			input:          `{"pattern": "NONEXISTENT"}`,
			wantErr:        false,
			wantMatchCount: 0,
			wantFileCount:  0,
			validate: func(t *testing.T, result *tools.ToolResult) {
				if !strings.Contains(result.Output, "No matches found") {
					t.Error("Expected output to indicate no matches found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := tool.Execute(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			// Check metadata
			if matchCountStr, ok := result.Metadata["match_count"]; ok {
				if matchCountStr != "0" && tt.wantMatchCount == 0 {
					t.Errorf("Expected 0 matches, got %s", matchCountStr)
				}
			}

			// Validate extra data
			if result.ExtraData != nil {
				if outputData, ok := result.ExtraData["output"]; ok {
					if grepOutput, ok := outputData.(GrepOutput); ok {
						if grepOutput.MatchCount != tt.wantMatchCount {
							t.Errorf("Expected %d matches, got %d", tt.wantMatchCount, grepOutput.MatchCount)
						}
						if grepOutput.FileCount != tt.wantFileCount {
							t.Errorf("Expected %d files with matches, got %d", tt.wantFileCount, grepOutput.FileCount)
						}
					}
				}
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGrepTool_InvalidInput(t *testing.T) {
	tool := NewGrepTool("")

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "CraftInvalidJSON",
			input:   `{"pattern": "test"`,
			wantErr: true,
		},
		{
			name:    "CraftMissingPattern",
			input:   `{"include": "*.go"}`,
			wantErr: true,
		},
		{
			name:    "CraftInvalidPath",
			input:   `{"pattern": "test", "path": "../../../etc/passwd"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tool.Execute(ctx, tt.input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGrepTool_ExpandBracePattern(t *testing.T) {
	tool := NewGrepTool("")

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "JourneymanSimpleBracePattern",
			pattern:  "*.{js,ts}",
			expected: []string{"*.js", "*.ts"},
		},
		{
			name:     "JourneymanComplexBracePattern",
			pattern:  "*.{ts,tsx,js,jsx}",
			expected: []string{"*.ts", "*.tsx", "*.js", "*.jsx"},
		},
		{
			name:     "JourneymanNoBraces",
			pattern:  "*.go",
			expected: []string{"*.go"},
		},
		{
			name:     "JourneymanEmptyBraces",
			pattern:  "*.{}",
			expected: []string{"*."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.expandBracePattern(tt.pattern)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d patterns, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected pattern %s at index %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}

func TestGrepTool_IsTextFile(t *testing.T) {
	tool := NewGrepTool("")

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "JourneymanGoFile",
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "JourneymanJavaScriptFile",
			filePath: "script.js",
			expected: true,
		},
		{
			name:     "JourneymanTypeScriptFile",
			filePath: "component.tsx",
			expected: true,
		},
		{
			name:     "JourneymanTextFile",
			filePath: "README.md",
			expected: true,
		},
		{
			name:     "JourneymanConfigFile",
			filePath: "config.yaml",
			expected: true,
		},
		{
			name:     "JourneymanDockerfile",
			filePath: "Dockerfile",
			expected: true,
		},
		{
			name:     "JourneymanMakefile",
			filePath: "Makefile",
			expected: true,
		},
		{
			name:     "JourneymanREADME",
			filePath: "README",
			expected: true,
		},
		{
			name:     "JourneymanLicense",
			filePath: "LICENSE",
			expected: true,
		},
		{
			name:     "GuildUnknownExtension",
			filePath: "file.unknown",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isTextFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("Expected %t for file %s, got %t", tt.expected, tt.filePath, result)
			}
		})
	}
}

func TestGrepTool_ResolvePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grep_path_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	tool := NewGrepTool(tempDir)

	tests := []struct {
		name     string
		path     string
		wantNil  bool
		validate func(t *testing.T, result string)
	}{
		{
			name:    "JourneymanCurrentDirectory",
			path:    ".",
			wantNil: false,
			validate: func(t *testing.T, result string) {
				if result != tempDir {
					t.Errorf("Expected %s, got %s", tempDir, result)
				}
			},
		},
		{
			name:    "JourneymanSubdirectory", 
			path:    "subdir",
			wantNil: false,
			validate: func(t *testing.T, result string) {
				if result != subDir {
					t.Errorf("Expected %s, got %s", subDir, result)
				}
			},
		},
		{
			name:    "GuildInvalidPath",
			path:    "nonexistent",
			wantNil: true,
		},
		{
			name:    "GuildUnsafePath",
			path:    "../../../etc",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.resolvePath(tt.path)
			
			if tt.wantNil {
				if result != "" {
					t.Errorf("Expected empty result for invalid path, got %s", result)
				}
				return
			}

			if result == "" {
				t.Error("Expected non-empty result for valid path")
				return
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGrepTool_Schema(t *testing.T) {
	tool := NewGrepTool("")
	schema := tool.Schema()

	// Check that schema contains required fields
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Check type
	if schema["type"] != "object" {
		t.Error("Expected schema type to be object")
	}

	// Check properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check pattern property
	if _, ok := properties["pattern"]; !ok {
		t.Error("Expected pattern property in schema")
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required to be a string slice")
	}

	if len(required) != 1 || required[0] != "pattern" {
		t.Error("Expected pattern to be the only required field")
	}
}

func TestGrepTool_Examples(t *testing.T) {
	tool := NewGrepTool("")
	examples := tool.Examples()

	if len(examples) == 0 {
		t.Error("Expected at least one example")
	}

	// Validate that examples are valid JSON
	for i, example := range examples {
		var input GrepInput
		if err := json.Unmarshal([]byte(example), &input); err != nil {
			t.Errorf("Example %d is not valid JSON: %v", i, err)
		}
		if input.Pattern == "" {
			t.Errorf("Example %d missing required pattern field", i)
		}
	}
}

func TestGrepTool_CountUniqueFiles(t *testing.T) {
	tool := NewGrepTool("")

	results := []GrepResult{
		{FilePath: "file1.go", LineNum: 1},
		{FilePath: "file1.go", LineNum: 2},
		{FilePath: "file2.js", LineNum: 1},
		{FilePath: "file3.py", LineNum: 1},
		{FilePath: "file2.js", LineNum: 5},
	}

	count := tool.countUniqueFiles(results)
	if count != 3 {
		t.Errorf("Expected 3 unique files, got %d", count)
	}
}

func TestGrepTool_FormatOutput(t *testing.T) {
	tool := NewGrepTool("")

	output := GrepOutput{
		Pattern:      "test",
		Include:      "*.go",
		Path:         ".",
		MatchCount:   2,
		FileCount:    1,
		FilesScanned: 5,
		Duration:     "100ms",
		Results: []GrepResult{
			{
				FilePath: "test.go",
				LineNum:  1,
				Line:     "// This is a test",
				Match:    "test",
				Column:   15,
			},
			{
				FilePath: "test.go",
				LineNum:  5,
				Line:     "func test() {}",
				Match:    "test",
				Column:   6,
			},
		},
	}

	formatted := tool.formatOutput(output)

	// Check that output contains expected information
	expectedStrings := []string{
		"Grep Results for pattern: test",
		"File filter: *.go",
		"Found 2 matches in 1 files",
		"test.go:",
		"Line 1:15:",
		"Line 5:6:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(formatted, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, formatted)
		}
	}
}