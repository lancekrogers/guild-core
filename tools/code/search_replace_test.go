package code

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchReplaceTool_NewSearchReplaceTool(t *testing.T) {
	tool := NewSearchReplaceTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "search_replace", tool.GetName())
	assert.Equal(t, "code", tool.GetCategory())
}

func TestSearchReplaceTool_Execute_SearchOnly(t *testing.T) {
	// Create a temporary file with content to search
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Printf("Testing search functionality")
	log.Info("This should not match")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern: "fmt.Println",
		Files:   []string{tmpFile.Name()},
		Context: 1,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find the match
	assert.Contains(t, result.Content, "1 matches in 1 files")
	assert.Contains(t, result.Content, "fmt.Println")
}

func TestSearchReplaceTool_Execute_SearchAndReplace(t *testing.T) {
	// Create a temporary file
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("Testing replacement")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:     "fmt.Println",
		Replacement: "log.Info",
		Files:       []string{tmpFile.Name()},
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should apply replacement
	assert.Contains(t, result.Content, "Modified 1 files")
	
	// Verify file was actually modified
	modifiedContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(modifiedContent), "log.Info")
	assert.NotContains(t, string(modifiedContent), "fmt.Println")
}

func TestSearchReplaceTool_Execute_Preview(t *testing.T) {
	// Create a temporary file
	content := `package main

func main() {
	oldFunction()
	oldFunction("param")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:     "oldFunction",
		Replacement: "newFunction",
		Files:       []string{tmpFile.Name()},
		Preview:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should show preview
	assert.Contains(t, result.Content, "Preview of changes:")
	assert.Contains(t, result.Content, "oldFunction")
	assert.Contains(t, result.Content, "newFunction")
	
	// File should not be modified
	originalContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(originalContent), "oldFunction")
}

func TestSearchReplaceTool_Execute_RegexPattern(t *testing.T) {
	// Create a temporary file
	content := `package main

func main() {
	var user1 = "John"
	var user2 = "Jane"
	var data = "some data"
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:     `var user(\d+)`,
		Replacement: `user$1 :=`,
		Files:       []string{tmpFile.Name()},
		Regex:       true,
		Preview:     true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find regex matches
	assert.Contains(t, result.Content, "user1")
	assert.Contains(t, result.Content, "user2")
}

func TestSearchReplaceTool_Execute_CaseSensitive(t *testing.T) {
	// Create a temporary file
	content := `package main

func main() {
	Function()
	function()
	FUNCTION()
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	// Case sensitive search
	params := SearchReplaceParams{
		Pattern:       "function",
		Files:         []string{tmpFile.Name()},
		CaseSensitive: true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should only find lowercase match
	assert.Contains(t, result.Content, "1 matches in 1 files")
}

func TestSearchReplaceTool_Execute_WholeWord(t *testing.T) {
	// Create a temporary file
	content := `package main

func main() {
	test()
	testing()
	contest()
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:   "test",
		Files:     []string{tmpFile.Name()},
		WholeWord: true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should only match whole word "test", not "testing" or "contest"
	assert.Contains(t, result.Content, "1 matches in 1 files")
}

func TestSearchReplaceTool_Execute_MultipleFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create multiple files
	files := []string{"file1.go", "file2.go", "file3.txt"}
	for i, filename := range files {
		content := `package main

func main() {
	fmt.Println("Hello from file %d")
}
`
		err = os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:   "fmt.Println",
		Files:     []string{filepath.Join(tmpDir, "*.go")},
		Recursive: false,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find matches in Go files only
	assert.Contains(t, result.Content, "2 matches in 2 files")
}

func TestSearchReplaceTool_Execute_RecursiveSearch(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Create files in both directories
	content := `package main
func main() {
	fmt.Println("Hello")
}
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644)
	require.NoError(t, err)
	
	err = os.WriteFile(filepath.Join(subDir, "sub.go"), []byte(content), 0644)
	require.NoError(t, err)

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:   "fmt.Println",
		Files:     []string{"*.go"},
		Recursive: true,
	}
	
	// Change to temp directory for recursive search
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find matches in both directories
	assert.Contains(t, result.Content, "2 matches in 2 files")
}

func TestSearchReplaceTool_Execute_InvalidPattern(t *testing.T) {
	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern: "",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSearchReplaceTool_Execute_InvalidRegex(t *testing.T) {
	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern: "[invalid regex",
		Regex:   true,
		Files:   []string{"*.go"},
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSearchReplaceTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewSearchReplaceTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSearchReplaceTool_Execute_NoFiles(t *testing.T) {
	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern: "test",
		Files:   []string{"nonexistent_*.go"},
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should return no matches
	assert.Contains(t, result.Content, "0 matches in 0 files")
}

func TestSearchReplaceTool_Execute_MaxResults(t *testing.T) {
	// Create a file with many matches
	content := `package main

func main() {
	test()
	test()
	test()
	test()
	test()
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern:    "test",
		Files:      []string{tmpFile.Name()},
		MaxResults: 3,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should limit results
	assert.Contains(t, result.Content, "3 matches in 1 files")
}

func TestSearchReplaceTool_Execute_LanguageBreakdown(t *testing.T) {
	// Create temporary directory with different file types
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create Go file
	goContent := `package main
func main() {
	test()
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goContent), 0644)
	require.NoError(t, err)

	// Create Python file
	pythonContent := `def main():
    test()
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(pythonContent), 0644)
	require.NoError(t, err)

	tool := NewSearchReplaceTool()
	
	params := SearchReplaceParams{
		Pattern: "test",
		Files:   []string{filepath.Join(tmpDir, "*")},
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should show language breakdown
	assert.Contains(t, result.Content, "Language breakdown:")
	assert.Contains(t, result.Content, "go:")
	assert.Contains(t, result.Content, "python:")
}