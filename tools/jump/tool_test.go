// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package jump

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestJumpToolExecute(t *testing.T) {
	// Set up test environment
	tmpDir, err := ioutil.TempDir("", "jump-tool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the default database location for testing
	originalFactory := defaultJumpFactory
	defaultJumpFactory = func() (*Jump, error) {
		dbPath := filepath.Join(tmpDir, "test.db")
		return New(dbPath)
	}
	defer func() {
		defaultJumpFactory = originalFactory
	}()

	// Create jump tool
	tool, err := NewJumpTool()
	if err != nil {
		t.Fatal(err)
	}
	defer tool.Close()

	// Create test directories
	testDirs := []string{
		filepath.Join(tmpDir, "projects", "guild-framework"),
		filepath.Join(tmpDir, "documents"),
		filepath.Join(tmpDir, "downloads"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()

	// Test tracking directories
	for _, dir := range testDirs {
		input := fmt.Sprintf(`{"query": "%s", "track": true}`, dir)
		result, err := tool.Execute(ctx, input)
		if err != nil {
			t.Errorf("Failed to track %s: %v", dir, err)
			continue
		}
		if result.Output != "ok" {
			t.Errorf("Track result = %s, want 'ok'", result.Output)
		}
		if result.Metadata["operation"] != "track" {
			t.Errorf("Operation = %s, want 'track'", result.Metadata["operation"])
		}
	}

	// Test finding directories
	tests := []struct {
		input    string
		contains string
	}{
		{`{"query": "guild"}`, "guild-framework"},
		{`{"query": "doc"}`, "documents"},
		{`{"query": "down"}`, "downloads"},
	}

	for _, test := range tests {
		result, err := tool.Execute(ctx, test.input)
		if err != nil {
			t.Errorf("Failed to execute %s: %v", test.input, err)
			continue
		}

		// Unquote the result
		var path string
		if err := json.Unmarshal([]byte(result.Output), &path); err != nil {
			t.Errorf("Failed to unmarshal result %s: %v", result.Output, err)
			continue
		}

		if !filepath.IsAbs(path) {
			t.Errorf("Result is not absolute path: %s", path)
		}

		if !contains(path, test.contains) {
			t.Errorf("Result %s does not contain %s", path, test.contains)
		}

		if result.Metadata["operation"] != "find" {
			t.Errorf("Operation = %s, want 'find'", result.Metadata["operation"])
		}
	}

	// Test recent directories
	recentInput := `{"recent": 2}`
	result, err := tool.Execute(ctx, recentInput)
	if err != nil {
		t.Errorf("Failed to get recent: %v", err)
	} else {
		var dirs []string
		if err := json.Unmarshal([]byte(result.Output), &dirs); err != nil {
			t.Errorf("Failed to unmarshal recent result: %v", err)
		} else {
			if len(dirs) != 2 {
				t.Errorf("Recent returned %d directories, want 2", len(dirs))
			}
			for _, dir := range dirs {
				if !filepath.IsAbs(dir) {
					t.Errorf("Recent directory is not absolute: %s", dir)
				}
			}
		}
		if result.Metadata["operation"] != "recent" {
			t.Errorf("Operation = %s, want 'recent'", result.Metadata["operation"])
		}
	}

	// Test tracking current directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	cwdInput := `{"query": ".", "track": true}`
	result, err = tool.Execute(ctx, cwdInput)
	if err != nil {
		t.Errorf("Failed to track current directory: %v", err)
	} else {
		if result.Output != "ok" {
			t.Errorf("Track current dir result = %s, want 'ok'", result.Output)
		}
		// Resolve paths to handle symlinks (e.g., /var vs /private/var on macOS)
		resolvedTracked, _ := filepath.EvalSymlinks(result.Metadata["path"])
		resolvedExpected, _ := filepath.EvalSymlinks(tmpDir)
		if resolvedTracked != resolvedExpected {
			t.Errorf("Tracked path = %s, want %s", result.Metadata["path"], tmpDir)
		}
	}
}

func TestJumpToolErrors(t *testing.T) {
	// Set up test environment
	tmpDir, err := ioutil.TempDir("", "jump-tool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the default database location for testing
	originalFactory := defaultJumpFactory
	defaultJumpFactory = func() (*Jump, error) {
		dbPath := filepath.Join(tmpDir, "test.db")
		return New(dbPath)
	}
	defer func() {
		defaultJumpFactory = originalFactory
	}()

	// Create jump tool
	tool, err := NewJumpTool()
	if err != nil {
		t.Fatal(err)
	}
	defer tool.Close()

	ctx := context.Background()

	// Test invalid JSON
	_, err = tool.Execute(ctx, "not json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Test empty query
	_, err = tool.Execute(ctx, `{"query": ""}`)
	if err == nil {
		t.Error("Expected error for empty query")
	}

	// Test no parameters
	_, err = tool.Execute(ctx, `{}`)
	if err == nil {
		t.Error("Expected error for no parameters")
	}

	// Test tracking non-existent directory
	_, err = tool.Execute(ctx, `{"query": "/does/not/exist", "track": true}`)
	if err == nil {
		t.Error("Expected error when tracking non-existent directory")
	}

	// Test finding with no tracked directories
	_, err = tool.Execute(ctx, `{"query": "anything"}`)
	if err == nil {
		t.Error("Expected error when finding with no tracked directories")
	}
}

func TestJumpToolMetadata(t *testing.T) {
	tool, err := NewJumpTool()
	if err != nil {
		t.Fatal(err)
	}
	defer tool.Close()

	// Check tool metadata
	if tool.Name() != "jump" {
		t.Errorf("Name = %s, want 'jump'", tool.Name())
	}

	if tool.Category() != "navigation" {
		t.Errorf("Category = %s, want 'navigation'", tool.Category())
	}

	if tool.RequiresAuth() {
		t.Error("Jump tool should not require auth")
	}

	// Check examples
	examples := tool.Examples()
	if len(examples) == 0 {
		t.Error("Tool should have examples")
	}

	// Check schema
	schema := tool.Schema()
	if schema["type"] != "object" {
		t.Error("Schema should be object type")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[len(s)-len(substr):] == substr ||
		(len(substr) > 0 && len(s) > 0 && filepath.Base(s) == substr || filepath.Dir(s) == substr ||
			filepath.Base(filepath.Dir(s)) == substr || filepath.Base(s) == filepath.Base(substr))))
}
