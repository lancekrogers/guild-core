// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Test data constants
const (
	testFileContent = `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func oldFunction() {
	fmt.Println("This is an old function")
	fmt.Println("It will be replaced")
}

func anotherFunction() {
	oldFunction()
	fmt.Println("Done")
}
`

	simpleTestContent = `line1
line2
line3
line4`

	jsonTestContent = `{
	"name": "test",
	"version": "1.0.0",
	"debug": false,
	"port": 8080
}`
)

func TestCraftMultiEditTool(t *testing.T) {
	tool := NewMultiEditTool()

	// Test basic tool properties
	if tool.Name() != "multi_edit" {
		t.Errorf("Expected tool name 'multi_edit', got '%s'", tool.Name())
	}

	if tool.Category() != "edit" {
		t.Errorf("Expected category 'edit', got '%s'", tool.Category())
	}

	if tool.RequiresAuth() {
		t.Error("Expected tool to not require auth")
	}

	// Test schema structure
	schema := tool.Schema()
	if schema == nil {
		t.Error("Expected non-nil schema")
	}

	examples := tool.Examples()
	if len(examples) == 0 {
		t.Error("Expected non-empty examples")
	}
}

func TestCraftMultiEditSingleEdit(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, simpleTestContent)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: "line2",
				NewString: "modified_line2",
			},
		},
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edit to be applied")
	}

	if result.TotalEdits != 1 {
		t.Errorf("Expected 1 edit, got %d", result.TotalEdits)
	}

	if result.AppliedEdits != 1 {
		t.Errorf("Expected 1 applied edit, got %d", result.AppliedEdits)
	}

	// Verify file content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := strings.Replace(simpleTestContent, "line2", "modified_line2", 1)
	if string(content) != expected {
		t.Errorf("Expected content:\n%s\nGot:\n%s", expected, string(content))
	}
}

func TestCraftMultiEditMultipleEdits(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, testFileContent)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString:  "oldFunction",
				NewString:  "newFunction",
				ReplaceAll: true,
			},
			{
				OldString: "Hello, World!",
				NewString: "Hello, Guild!",
			},
			{
				OldString: "This is an old function",
				NewString: "This is a new function",
			},
		},
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edits to be applied")
	}

	if result.TotalEdits != 3 {
		t.Errorf("Expected 3 edits, got %d", result.TotalEdits)
	}

	if result.AppliedEdits != 3 {
		t.Errorf("Expected 3 applied edits, got %d", result.AppliedEdits)
	}

	// Verify file content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "oldFunction") {
		t.Error("Expected all occurrences of 'oldFunction' to be replaced")
	}

	if !strings.Contains(contentStr, "newFunction") {
		t.Error("Expected 'newFunction' to be present")
	}

	if !strings.Contains(contentStr, "Hello, Guild!") {
		t.Error("Expected 'Hello, Guild!' to be present")
	}

	if !strings.Contains(contentStr, "This is a new function") {
		t.Error("Expected 'This is a new function' to be present")
	}
}

func TestCraftMultiEditReplaceAll(t *testing.T) {
	tool := NewMultiEditTool()
	content := "test test test"
	tempFile := createTestFile(t, content)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString:  "test",
				NewString:  "demo",
				ReplaceAll: true,
			},
		},
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edit to be applied")
	}

	// Check that all 3 occurrences were replaced
	if result.Stats.ReplacedOccurrences != 3 {
		t.Errorf("Expected 3 replaced occurrences, got %d", result.Stats.ReplacedOccurrences)
	}

	// Verify file content
	newContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "demo demo demo"
	if string(newContent) != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, string(newContent))
	}
}

func TestCraftMultiEditReplaceFirst(t *testing.T) {
	tool := NewMultiEditTool()
	content := "test test test"
	tempFile := createTestFile(t, content)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString:  "test",
				NewString:  "demo",
				ReplaceAll: false, // Only first occurrence
			},
		},
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edit to be applied")
	}

	// Check that only 1 occurrence was replaced
	if result.Stats.ReplacedOccurrences != 1 {
		t.Errorf("Expected 1 replaced occurrence, got %d", result.Stats.ReplacedOccurrences)
	}

	// Verify file content
	newContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "demo test test"
	if string(newContent) != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, string(newContent))
	}
}

func TestCraftMultiEditDryRun(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, simpleTestContent)
	defer os.Remove(tempFile)

	originalContent, _ := os.ReadFile(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: "line2",
				NewString: "modified_line2",
			},
		},
		DryRun: true,
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Applied {
		t.Error("Expected edit to not be applied in dry run mode")
	}

	if result.AppliedEdits != 1 {
		t.Errorf("Expected 1 applied edit (simulated), got %d", result.AppliedEdits)
	}

	if result.Preview == "" {
		t.Error("Expected preview to be generated")
	}

	// Verify file content unchanged
	currentContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(currentContent) != string(originalContent) {
		t.Error("Expected file content to be unchanged in dry run mode")
	}
}

func TestCraftMultiEditBackup(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, simpleTestContent)
	defer os.Remove(tempFile)

	originalContent, _ := os.ReadFile(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: "line2",
				NewString: "modified_line2",
			},
		},
		Backup: true,
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edit to be applied")
	}

	if result.BackupFile == "" {
		t.Error("Expected backup file to be created")
	}

	// Verify backup file exists and has original content
	backupContent, err := os.ReadFile(result.BackupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != string(originalContent) {
		t.Error("Expected backup file to contain original content")
	}

	// Clean up backup file
	defer os.Remove(result.BackupFile)
}

func TestCraftMultiEditValidation(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, simpleTestContent)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: "line2",
				NewString: "modified_line2",
			},
			{
				OldString: "nonexistent",
				NewString: "replacement",
			},
		},
		Validate: true,
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Applied {
		t.Error("Expected edits to not be applied due to validation failure")
	}

	if result.ValidationError == "" {
		t.Error("Expected validation error")
	}

	// Verify file content unchanged
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != simpleTestContent {
		t.Error("Expected file content to be unchanged after validation failure")
	}
}

func TestCraftMultiEditAtomicity(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, simpleTestContent)
	defer os.Remove(tempFile)

	// Create params where first edit works but second fails
	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: "line2",
				NewString: "modified_line2",
			},
			{
				OldString: "nonexistent",
				NewString: "replacement",
			},
		},
		Validate: false, // Don't validate upfront to test atomicity
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// With validation disabled, first edit should succeed, second should fail
	if result.AppliedEdits != 1 {
		t.Errorf("Expected 1 applied edit, got %d", result.AppliedEdits)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors for failed edits")
	}

	// Verify first edit was applied
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "modified_line2") {
		t.Error("Expected first edit to be applied")
	}
}

func TestGuildMultiEditFileNotFound(t *testing.T) {
	tool := NewMultiEditTool()

	params := MultiEditParams{
		FilePath: "/nonexistent/file.txt",
		Edits: []EditEntry{
			{
				OldString: "test",
				NewString: "demo",
			},
		},
	}

	input, _ := json.Marshal(params)
	_, err := tool.Execute(context.Background(), string(input))

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	var guildErr *gerror.GuildError
	if !gerror.As(err, &guildErr) {
		t.Error("Expected GuildError")
	}

	if guildErr.Code != gerror.ErrCodeNotFound {
		t.Errorf("Expected ErrCodeNotFound, got %s", guildErr.Code)
	}
}

func TestGuildMultiEditInvalidInput(t *testing.T) {
	tool := NewMultiEditTool()

	tests := []struct {
		name   string
		params MultiEditParams
	}{
		{
			name: "empty file path",
			params: MultiEditParams{
				FilePath: "",
				Edits: []EditEntry{
					{OldString: "test", NewString: "demo"},
				},
			},
		},
		{
			name: "no edits",
			params: MultiEditParams{
				FilePath: "test.txt",
				Edits:    []EditEntry{},
			},
		},
		{
			name: "empty old string",
			params: MultiEditParams{
				FilePath: "test.txt",
				Edits: []EditEntry{
					{OldString: "", NewString: "demo"},
				},
			},
		},
		{
			name: "identical old and new strings",
			params: MultiEditParams{
				FilePath: "test.txt",
				Edits: []EditEntry{
					{OldString: "test", NewString: "test"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, _ := json.Marshal(tt.params)
			_, err := tool.Execute(context.Background(), string(input))

			if err == nil {
				t.Error("Expected error for invalid input")
			}

			var guildErr *gerror.GuildError
			if !gerror.As(err, &guildErr) {
				t.Error("Expected GuildError")
			}

			if guildErr.Code != gerror.ErrCodeInvalidInput {
				t.Errorf("Expected ErrCodeInvalidInput, got %s", guildErr.Code)
			}
		})
	}
}

func TestGuildMultiEditMalformedJSON(t *testing.T) {
	tool := NewMultiEditTool()

	_, err := tool.Execute(context.Background(), `{"file_path": "test.txt", "edits": [malformed json}`)

	if err == nil {
		t.Error("Expected error for malformed JSON")
	}

	var guildErr *gerror.GuildError
	if !gerror.As(err, &guildErr) {
		t.Error("Expected GuildError")
	}

	if guildErr.Code != gerror.ErrCodeInvalidInput {
		t.Errorf("Expected ErrCodeInvalidInput, got %s", guildErr.Code)
	}
}

func TestScribeMultiEditComplexJSON(t *testing.T) {
	tool := NewMultiEditTool()
	tempFile := createTestFile(t, jsonTestContent)
	defer os.Remove(tempFile)

	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString: `"debug": false`,
				NewString: `"debug": true`,
			},
			{
				OldString: `"port": 8080`,
				NewString: `"port": 3000`,
			},
			{
				OldString: `"version": "1.0.0"`,
				NewString: `"version": "2.0.0"`,
			},
		},
		Backup: true,
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edits to be applied")
	}

	if result.AppliedEdits != 3 {
		t.Errorf("Expected 3 applied edits, got %d", result.AppliedEdits)
	}

	// Verify changes
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"debug": true`) {
		t.Error("Expected debug to be set to true")
	}

	if !strings.Contains(contentStr, `"port": 3000`) {
		t.Error("Expected port to be set to 3000")
	}

	if !strings.Contains(contentStr, `"version": "2.0.0"`) {
		t.Error("Expected version to be set to 2.0.0")
	}

	// Clean up backup file
	if result.BackupFile != "" {
		defer os.Remove(result.BackupFile)
	}
}

func TestJourneymanMultiEditSequentialEdits(t *testing.T) {
	tool := NewMultiEditTool()
	content := "abc def abc ghi abc"
	tempFile := createTestFile(t, content)
	defer os.Remove(tempFile)

	// Test that sequential edits work correctly
	params := MultiEditParams{
		FilePath: tempFile,
		Edits: []EditEntry{
			{
				OldString:  "abc",
				NewString:  "xyz",
				ReplaceAll: false, // Only first occurrence
			},
			{
				OldString: "def",
				NewString: "uvw",
			},
			{
				OldString:  "abc", // Should still find remaining occurrences
				NewString:  "rst",
				ReplaceAll: true, // All remaining occurrences
			},
		},
	}

	result, err := executeMultiEdit(t, tool, params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Applied {
		t.Error("Expected edits to be applied")
	}

	if result.AppliedEdits != 3 {
		t.Errorf("Expected 3 applied edits, got %d", result.AppliedEdits)
	}

	// Verify final content
	finalContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "xyz uvw rst ghi rst"
	if string(finalContent) != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, string(finalContent))
	}
}

// Helper functions

func createTestFile(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "multi_edit_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tempFile.Name()
}

func executeMultiEdit(t *testing.T, tool *MultiEditTool, params MultiEditParams) (*MultiEditResult, error) {
	input, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	result, err := tool.Execute(context.Background(), string(input))
	if err != nil {
		return nil, err
	}

	// Extract the actual result from the tool result
	multiEditResult, ok := result.ExtraData["result"].(*MultiEditResult)
	if !ok {
		t.Fatal("Failed to extract MultiEditResult from tool result")
	}

	return multiEditResult, nil
}

// Benchmark tests

func BenchmarkMultiEditSingleEdit(b *testing.B) {
	tool := NewMultiEditTool()
	content := strings.Repeat("test content line\n", 1000)

	for i := 0; i < b.N; i++ {
		tempFile := createBenchFile(b, content)

		params := MultiEditParams{
			FilePath: tempFile,
			Edits: []EditEntry{
				{
					OldString:  "test",
					NewString:  "demo",
					ReplaceAll: true,
				},
			},
		}

		input, _ := json.Marshal(params)
		_, err := tool.Execute(context.Background(), string(input))
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}

		os.Remove(tempFile)
	}
}

func BenchmarkMultiEditMultipleEdits(b *testing.B) {
	tool := NewMultiEditTool()
	content := strings.Repeat("test content line with data\n", 1000)

	for i := 0; i < b.N; i++ {
		tempFile := createBenchFile(b, content)

		params := MultiEditParams{
			FilePath: tempFile,
			Edits: []EditEntry{
				{OldString: "test", NewString: "demo"},
				{OldString: "content", NewString: "material"},
				{OldString: "line", NewString: "row"},
				{OldString: "data", NewString: "info"},
			},
		}

		input, _ := json.Marshal(params)
		_, err := tool.Execute(context.Background(), string(input))
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}

		os.Remove(tempFile)
	}
}

func createBenchFile(b *testing.B, content string) string {
	tempFile, err := os.CreateTemp("", "multi_edit_bench_*.txt")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		b.Fatalf("Failed to close temp file: %v", err)
	}

	return tempFile.Name()
}
