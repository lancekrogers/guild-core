package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCraftAgTool tests creating a new ag tool
func TestCraftAgTool(t *testing.T) {
	workingDir := "/tmp/test"
	tool := NewAgTool(workingDir)

	if tool == nil {
		t.Fatal("Expected tool to be created, got nil")
	}

	if tool.Name() != "ag" {
		t.Errorf("Expected tool name to be 'ag', got '%s'", tool.Name())
	}

	if tool.Category() != "search" {
		t.Errorf("Expected tool category to be 'search', got '%s'", tool.Category())
	}

	if tool.RequiresAuth() {
		t.Error("Expected tool to not require auth")
	}

	description := tool.Description()
	if !strings.Contains(description, "Silver Searcher") {
		t.Errorf("Expected description to contain 'Silver Searcher', got '%s'", description)
	}
}

// TestAgToolSchema tests the JSON schema of the ag tool
func TestAgToolSchema(t *testing.T) {
	tool := NewAgTool("")
	schema := tool.Schema()

	if schema == nil {
		t.Fatal("Expected schema to be defined")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be defined in schema")
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required fields to be defined")
	}

	if len(required) != 1 || required[0] != "pattern" {
		t.Errorf("Expected required fields to be ['pattern'], got %v", required)
	}

	// Check pattern property
	pattern, ok := properties["pattern"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pattern property to be defined")
	}

	if pattern["type"] != "string" {
		t.Errorf("Expected pattern type to be 'string', got '%v'", pattern["type"])
	}

	// Check file_types property
	fileTypes, ok := properties["file_types"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected file_types property to be defined")
	}

	if fileTypes["type"] != "array" {
		t.Errorf("Expected file_types type to be 'array', got '%v'", fileTypes["type"])
	}
}

// TestAgToolExamples tests the examples provided by the ag tool
func TestAgToolExamples(t *testing.T) {
	tool := NewAgTool("")
	examples := tool.Examples()

	if len(examples) == 0 {
		t.Fatal("Expected examples to be provided")
	}

	// Validate that examples are valid JSON
	for i, example := range examples {
		var input AgToolInput
		if err := json.Unmarshal([]byte(example), &input); err != nil {
			t.Errorf("Example %d is not valid JSON: %v", i, err)
		}

		if input.Pattern == "" {
			t.Errorf("Example %d missing required pattern field", i)
		}
	}
}

// TestAgToolValidation tests input validation
func TestAgToolValidation(t *testing.T) {
	tool := NewAgTool("")
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid input",
			input:       `{"pattern": "test"}`,
			expectError: false,
		},
		{
			name:        "missing pattern",
			input:       `{"path": "/tmp"}`,
			expectError: true,
			errorText:   "pattern is required",
		},
		{
			name:        "empty pattern",
			input:       `{"pattern": ""}`,
			expectError: true,
			errorText:   "pattern is required",
		},
		{
			name:        "invalid json",
			input:       `{invalid json}`,
			expectError: true,
			errorText:   "invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)

			if tt.expectError {
				if err == nil && (result == nil || result.Error == "") {
					t.Errorf("Expected error for test '%s', but got none", tt.name)
				}
				if err != nil && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorText, err.Error())
				}
				if result != nil && result.Error != "" && !strings.Contains(result.Error, tt.errorText) {
					t.Errorf("Expected result error to contain '%s', got '%s'", tt.errorText, result.Error)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
				}
			}
		})
	}
}

// TestAgToolExecuteWithoutAg tests execution when ag is not installed
func TestAgToolExecuteWithoutAg(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ag_tool_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewAgTool(tempDir)
	ctx := context.Background()

	// Test with ag not available (assuming it's not in PATH during test)
	input := `{"pattern": "test"}`
	result, err := tool.Execute(ctx, input)

	// Should return a result with error metadata, not an error
	if err != nil {
		t.Errorf("Expected no error when ag is not installed, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result when ag is not installed")
	}

	// Check if the error indicates ag is not installed
	if result.Metadata["error"] != "ag_not_installed" {
		// This test might pass if ag is actually installed on the system
		t.Logf("ag appears to be installed on the system, skipping not-installed test")
		return
	}

	if !strings.Contains(result.Error, "not installed") {
		t.Errorf("Expected error message about ag not being installed, got: %s", result.Error)
	}
}

// TestParseAgLine tests parsing individual ag output lines
func TestParseAgLine(t *testing.T) {
	tool := NewAgTool("/test/dir")

	tests := []struct {
		name     string
		line     string
		expected AgSearchResult
		hasError bool
	}{
		{
			name: "valid line",
			line: "file.go:10:5:func main() {",
			expected: AgSearchResult{
				File:    "file.go",
				Line:    10,
				Column:  5,
				Match:   "func main() {",
				Context: "func main() {",
			},
		},
		{
			name: "line with absolute path",
			line: "/test/dir/pkg/main.go:15:8:  return nil",
			expected: AgSearchResult{
				File:    "pkg/main.go",
				Line:    15,
				Column:  8,
				Match:   "return nil",
				Context: "  return nil",
			},
		},
		{
			name:     "invalid line format",
			line:     "invalid:format",
			hasError: true,
		},
		{
			name:     "invalid line number",
			line:     "file.go:abc:5:content",
			hasError: true,
		},
		{
			name:     "invalid column number",
			line:     "file.go:10:abc:content",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.parseAgLine(tt.line)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for line '%s', but got none", tt.line)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for line '%s': %v", tt.line, err)
				return
			}

			if result.File != tt.expected.File {
				t.Errorf("Expected file '%s', got '%s'", tt.expected.File, result.File)
			}

			if result.Line != tt.expected.Line {
				t.Errorf("Expected line %d, got %d", tt.expected.Line, result.Line)
			}

			if result.Column != tt.expected.Column {
				t.Errorf("Expected column %d, got %d", tt.expected.Column, result.Column)
			}

			if result.Match != tt.expected.Match {
				t.Errorf("Expected match '%s', got '%s'", tt.expected.Match, result.Match)
			}
		})
	}
}

// TestParseAgOutput tests parsing complete ag output
func TestParseAgOutput(t *testing.T) {
	tool := NewAgTool("/test/dir")

	output := `file1.go:10:5:func main() {
file2.go:20:8:  return nil
file3.go:30:12:  fmt.Println("test")`

	results, err := tool.parseAgOutput(output, 100)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check first result
	if results[0].File != "file1.go" || results[0].Line != 10 {
		t.Errorf("Unexpected first result: %+v", results[0])
	}

	// Test max results limit
	results, err = tool.parseAgOutput(output, 2)
	if err != nil {
		t.Fatalf("Unexpected error with max results: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with max limit, got %d", len(results))
	}
}

// TestAgToolCapabilities tests the capabilities reporting
func TestAgToolCapabilities(t *testing.T) {
	tool := NewAgTool("")
	capabilities := tool.GetCapabilities()

	expectedCapabilities := []string{"search", "text_search", "code_search", "pattern_matching", "file_filtering"}

	if len(capabilities) != len(expectedCapabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(capabilities))
	}

	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability '%s' not found in %v", expected, capabilities)
		}
	}
}

// TestAgToolInputParameterHandling tests various input parameter combinations
func TestAgToolInputParameterHandling(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "ag_tool_param_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"test.go":     "package main\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
		"test.js":     "function hello() {\n\tconsole.log('Hello');\n}",
		"README.md":   "# Test Project\nThis is a test project.",
		"config.json": `{"name": "test", "version": "1.0.0"}`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	tool := NewAgTool(tempDir)
	ctx := context.Background()

	tests := []struct {
		name        string
		input       AgToolInput
		description string
	}{
		{
			name: "basic search",
			input: AgToolInput{
				Pattern: "Hello",
			},
			description: "Basic text search",
		},
		{
			name: "file type filtering",
			input: AgToolInput{
				Pattern:   "Hello",
				FileTypes: []string{"go"},
			},
			description: "Search with file type filter",
		},
		{
			name: "case sensitive search",
			input: AgToolInput{
				Pattern:       "hello",
				CaseSensitive: true,
			},
			description: "Case sensitive search",
		},
		{
			name: "literal search",
			input: AgToolInput{
				Pattern: "Hello",
				Literal: true,
			},
			description: "Literal pattern search",
		},
		{
			name: "search with context",
			input: AgToolInput{
				Pattern: "Hello",
				Context: 1,
			},
			description: "Search with context lines",
		},
		{
			name: "search with max results",
			input: AgToolInput{
				Pattern:    "Hello",
				MaxResults: 1,
			},
			description: "Search with result limit",
		},
		{
			name: "search with timeout",
			input: AgToolInput{
				Pattern: "Hello",
				Timeout: 5,
			},
			description: "Search with custom timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// Note: This test may fail if ag is not installed on the system
			// We'll check the result structure but not assert on specific matches
			result, err := tool.Execute(ctx, string(inputJSON))

			// If ag is not installed, skip the test
			if result != nil && result.Metadata["error"] == "ag_not_installed" {
				t.Skipf("Skipping test '%s' - ag not installed on system", tt.name)
				return
			}

			if err != nil {
				t.Errorf("Unexpected error in test '%s': %v", tt.name, err)
				return
			}

			if result == nil {
				t.Errorf("Expected result in test '%s', got nil", tt.name)
				return
			}

			// Validate metadata
			if result.Metadata["pattern"] != tt.input.Pattern {
				t.Errorf("Expected pattern metadata '%s', got '%s'", tt.input.Pattern, result.Metadata["pattern"])
			}

			// Validate that output is valid JSON if no error
			if result.Error == "" {
				var agResult AgToolResult
				if err := json.Unmarshal([]byte(result.Output), &agResult); err != nil {
					t.Errorf("Result output is not valid JSON in test '%s': %v", tt.name, err)
				} else {
					// Basic structure validation
					if agResult.Pattern != tt.input.Pattern {
						t.Errorf("Expected result pattern '%s', got '%s'", tt.input.Pattern, agResult.Pattern)
					}
					if agResult.Path == "" {
						t.Error("Expected result path to be set")
					}
					if agResult.Duration == "" {
						t.Error("Expected result duration to be set")
					}
				}
			}
		})
	}
}

// TestAgToolLargeResults tests handling of large result sets
func TestAgToolLargeResults(t *testing.T) {
	tool := NewAgTool("")

	// Simulate large ag output
	var outputLines []string
	for i := 0; i < 1000; i++ {
		outputLines = append(outputLines, fmt.Sprintf("file%d.go:%d:1:test line %d", i, i+1, i))
	}
	output := strings.Join(outputLines, "\n")

	// Test with max results limit
	results, err := tool.parseAgOutput(output, 50)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 50 {
		t.Errorf("Expected 50 results with limit, got %d", len(results))
	}

	// Test without limit (should process all)
	results, err = tool.parseAgOutput(output, 2000)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1000 {
		t.Errorf("Expected 1000 results without limit, got %d", len(results))
	}
}

// TestAgToolPathHandling tests path resolution and validation
func TestAgToolPathHandling(t *testing.T) {
	tool := NewAgTool("/tmp")
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{
			name:        "nonexistent path",
			input:       `{"pattern": "test", "path": "/nonexistent/path"}`,
			expectError: true,
			errorText:   "does not exist",
		},
		{
			name:  "relative path",
			input: `{"pattern": "test", "path": "."}`,
		},
		{
			name:  "absolute path",
			input: `{"pattern": "test", "path": "/tmp"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)

			if tt.expectError {
				if err == nil && (result == nil || result.Error == "") {
					t.Errorf("Expected error for test '%s', but got none", tt.name)
				}
				if err != nil && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				// If ag is not installed, that's OK for path handling tests
				if result != nil && result.Metadata["error"] == "ag_not_installed" {
					t.Skipf("Skipping test '%s' - ag not installed on system", tt.name)
					return
				}
				if err != nil {
					t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
				}
			}
		})
	}
}

// BenchmarkAgToolExecution benchmarks the ag tool execution
func BenchmarkAgToolExecution(b *testing.B) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "ag_tool_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test files
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("package test%d\nfunc Test%d() {\n\tfmt.Println(\"test %d\")\n}", i, i, i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	tool := NewAgTool(tempDir)
	ctx := context.Background()
	input := `{"pattern": "func", "file_types": ["go"]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := tool.Execute(ctx, input)
		// Skip if ag is not installed
		if result != nil && result.Metadata["error"] == "ag_not_installed" {
			b.Skip("ag not installed on system")
			return
		}
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}