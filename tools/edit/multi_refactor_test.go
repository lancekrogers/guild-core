// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiFileRefactorTool_NewMultiFileRefactorTool(t *testing.T) {
	tool := NewMultiFileRefactorTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "multi_refactor", tool.Name())
	assert.Equal(t, "edit", tool.Category())
}

func TestMultiFileRefactorTool_Execute_RenameFunction(t *testing.T) {
	// Create a temporary Go file
	content := `package main

import "fmt"

func oldFunction() {
	fmt.Println("Hello from old function")
}

func main() {
	oldFunction()
	fmt.Println("Calling old function")
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "oldFunction",
		},
		NewName: "newFunction",
		Preview: true,
		Scope:   "file",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should show rename preview
	assert.Contains(t, result.Output, "Multi-File Refactoring Preview (Rename)")
	assert.Contains(t, result.Output, "oldFunction")
	assert.Contains(t, result.Output, "newFunction")
	assert.Contains(t, result.Output, "References found:")
}

func TestMultiFileRefactorTool_Execute_RenameApply(t *testing.T) {
	// Create a temporary Go file
	content := `package main

func testFunc() string {
	return "test"
}

func main() {
	result := testFunc()
	println(result)
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "testFunc",
		},
		NewName: "newTestFunc",
		Scope:   "file",
		Options: &RefactorOptions{
			UpdateReferences: true,
			BackupFiles:      true,
		},
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should apply rename
	assert.Contains(t, result.Output, "Multi-File Refactoring Applied (Rename)")
	assert.Contains(t, result.Output, "References updated:")

	// Verify file was modified
	modifiedContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(modifiedContent), "newTestFunc")
	assert.NotContains(t, string(modifiedContent), "testFunc")
}

func TestMultiFileRefactorTool_Execute_ExtractMethod(t *testing.T) {
	// Create a Go file with code to extract
	content := `package main

import "fmt"

func main() {
	name := "World"
	greeting := "Hello, " + name
	fmt.Println(greeting)
	fmt.Println("Have a nice day!")
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "extract",
		Target: &RefactorTarget{
			File:      tmpFile.Name(),
			StartLine: 6,
			EndLine:   8,
		},
		NewName: "printGreeting",
		Preview: true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should show extract preview
	assert.Contains(t, result.Output, "Multi-File Refactoring Preview (Extract)")
	assert.Contains(t, result.Output, "printGreeting")
	assert.Contains(t, result.Output, "3 lines into method")
}

func TestMultiFileRefactorTool_Execute_MoveRefactor(t *testing.T) {
	// Create source file
	sourceContent := `package main

func utilityFunction() string {
	return "utility"
}

func main() {
	result := utilityFunction()
	println(result)
}
`

	sourceFile, err := os.CreateTemp("", "source_*.go")
	require.NoError(t, err)
	defer os.Remove(sourceFile.Name())

	_, err = sourceFile.WriteString(sourceContent)
	require.NoError(t, err)
	sourceFile.Close()

	// Create destination file
	destContent := `package main

// Destination file for utility functions
`

	destFile, err := os.CreateTemp("", "dest_*.go")
	require.NoError(t, err)
	defer os.Remove(destFile.Name())

	_, err = destFile.WriteString(destContent)
	require.NoError(t, err)
	destFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "move",
		Target: &RefactorTarget{
			File:   sourceFile.Name(),
			Symbol: "utilityFunction",
		},
		Destination: destFile.Name(),
		Preview:     true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should handle move operation (placeholder implementation)
	assert.Contains(t, result.Output, "Move refactoring")
	assert.Contains(t, result.Output, "not fully implemented")
}

func TestMultiFileRefactorTool_Execute_InlineRefactor(t *testing.T) {
	// Create a Go file
	content := `package main

func simpleFunction() string {
	return "simple"
}

func main() {
	result := simpleFunction()
	println(result)
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "inline",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "simpleFunction",
		},
		Preview: true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should handle inline operation (placeholder implementation)
	assert.Contains(t, result.Output, "Inline refactoring")
	assert.Contains(t, result.Output, "not fully implemented")
}

func TestMultiFileRefactorTool_Execute_NamingConflict(t *testing.T) {
	// Create a Go file with existing function name
	content := `package main

func existingFunction() {
	println("existing")
}

func oldFunction() {
	println("old")
}

func main() {
	oldFunction()
	existingFunction()
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "oldFunction",
		},
		NewName: "existingFunction", // This should conflict
		Preview: true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should detect naming conflict
	assert.Contains(t, result.Output, "Conflicts")
	assert.Contains(t, result.Output, "name_collision")
}

func TestMultiFileRefactorTool_Execute_InvalidType(t *testing.T) {
	// Create a temporary file
	content := `package main
func main() {}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "invalid_type",
		Target: &RefactorTarget{
			File: tmpFile.Name(),
		},
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_MissingTarget(t *testing.T) {
	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		// Missing target
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_MissingFile(t *testing.T) {
	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File: "nonexistent.go",
		},
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewMultiFileRefactorTool()

	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_RenameWithoutNewName(t *testing.T) {
	// Create a temporary file
	content := `package main
func oldFunc() {}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "oldFunc",
		},
		// Missing NewName
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_ExtractWithoutLines(t *testing.T) {
	// Create a temporary file
	content := `package main
func main() {
	println("test")
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "extract",
		Target: &RefactorTarget{
			File: tmpFile.Name(),
			// Missing StartLine and EndLine
		},
		NewName: "extractedMethod",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_MoveWithoutDestination(t *testing.T) {
	// Create a temporary file
	content := `package main
func testFunc() {}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "move",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "testFunc",
		},
		// Missing Destination
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_ScopePackage(t *testing.T) {
	// Create a temporary Go file
	content := `package main

func testFunction() {
	println("test")
}

func main() {
	testFunction()
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "testFunction",
		},
		NewName: "renamedFunction",
		Scope:   "package", // Search entire package
		Preview: true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should search in package scope
	assert.Contains(t, result.Output, "References found:")
}

func TestMultiFileRefactorTool_Execute_ExtractOutOfBounds(t *testing.T) {
	// Create a small Go file
	content := `package main

func main() {
	println("test")
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "extract",
		Target: &RefactorTarget{
			File:      tmpFile.Name(),
			StartLine: 10, // Beyond file length
			EndLine:   15,
		},
		NewName: "extractedMethod",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMultiFileRefactorTool_Execute_DefaultOptions(t *testing.T) {
	// Create a Go file
	content := `package main

func oldFunc() {
	println("old")
}

func main() {
	oldFunc()
}
`

	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewMultiFileRefactorTool()

	params := RefactorParams{
		Type: "rename",
		Target: &RefactorTarget{
			File:   tmpFile.Name(),
			Symbol: "oldFunc",
		},
		NewName: "newFunc",
		Preview: true,
		// No Options specified - should use defaults
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should use default options
	assert.Contains(t, result.Output, "References found:")
}
