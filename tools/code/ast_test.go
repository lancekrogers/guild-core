package code

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASTTool_NewASTTool(t *testing.T) {
	tool := NewASTTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "ast", tool.GetName())
	assert.Equal(t, "code", tool.GetCategory())
	assert.NotNil(t, tool.parsers)
}

func TestASTTool_Execute_GoFile(t *testing.T) {
	// Create a temporary Go file
	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

type User struct {
	Name string
	Age  int
}

func (u User) Greet() string {
	return "Hello, " + u.Name
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewASTTool()
	
	// Test basic analysis
	params := ASTParams{
		File:   tmpFile.Name(),
		Target: "functions",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Parse the result
	var astResult ASTResult
	err = json.Unmarshal([]byte(result.ExtraData["result"].(map[string]interface{})["functions"].(string)), &astResult.Functions)
	require.NoError(t, err)
	
	// Should find main and Greet functions
	assert.GreaterOrEqual(t, len(astResult.Functions), 2)
	
	// Check for main function
	foundMain := false
	for _, fn := range astResult.Functions {
		if fn.Name == "main" {
			foundMain = true
			assert.Equal(t, 4, fn.StartLine)
			break
		}
	}
	assert.True(t, foundMain)
}

func TestASTTool_Execute_PythonFile(t *testing.T) {
	// Create a temporary Python file
	pythonCode := `def main():
    print("Hello, World!")

class User:
    def __init__(self, name, age):
        self.name = name
        self.age = age
    
    def greet(self):
        return f"Hello, {self.name}"

if __name__ == "__main__":
    main()
`
	
	tmpFile, err := os.CreateTemp("", "test_*.py")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(pythonCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewASTTool()
	
	params := ASTParams{
		File:   tmpFile.Name(),
		Target: "classes",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestASTTool_Execute_InvalidFile(t *testing.T) {
	tool := NewASTTool()
	
	params := ASTParams{
		File:   "nonexistent.go",
		Target: "functions",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestASTTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewASTTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestASTTool_Execute_EmptyFile(t *testing.T) {
	tool := NewASTTool()
	
	params := ASTParams{
		File:   "",
		Target: "functions",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filename string
		expected Language
	}{
		{"main.go", LanguageGo},
		{"script.py", LanguagePython},
		{"app.ts", LanguageTypeScript},
		{"app.js", LanguageJavaScript},
		{"app.jsx", LanguageJavaScript},
		{"app.tsx", LanguageTypeScript},
		{"unknown.txt", LanguageUnknown},
	}
	
	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := DetectLanguage(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestASTTool_Execute_UnsupportedLanguage(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString("some content")
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewASTTool()
	
	params := ASTParams{
		File:   tmpFile.Name(),
		Target: "functions",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should return empty results for unsupported language
	assert.Contains(t, result.Content, "Language: unknown")
}

func TestASTTool_Execute_AllTargets(t *testing.T) {
	// Create a comprehensive Go file
	goCode := `package main

import (
	"fmt"
	"os"
)

type User struct {
	Name string
	Age  int
}

type Admin struct {
	User
	Permissions []string
}

func main() {
	fmt.Println("Hello, World!")
}

func (u User) Greet() string {
	return "Hello, " + u.Name
}

var GlobalVar = "test"
const MaxAge = 100
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(goCode)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewASTTool()
	
	targets := []string{"functions", "types", "variables", "imports", "all"}
	
	for _, target := range targets {
		t.Run("target_"+target, func(t *testing.T) {
			params := ASTParams{
				File:   tmpFile.Name(),
				Target: target,
			}
			
			input, err := json.Marshal(params)
			require.NoError(t, err)
			
			result, err := tool.Execute(context.Background(), string(input))
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content, "Language: go")
		})
	}
}