package code

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGoParser is a simple parser for testing
type mockGoParser struct{}

func (p *mockGoParser) Parse(ctx context.Context, filename string, content []byte) (*ParseResult, error) {
	return &ParseResult{
		Language: LanguageGo,
		Filename: filename,
		Content:  content,
		AST:      "mock-ast", // Simple placeholder
		Errors:   []ParseError{},
		Metadata: map[string]interface{}{"parser": "mock"},
	}, nil
}

func (p *mockGoParser) GetFunctions(result *ParseResult) ([]*Function, error) {
	// Return mock functions based on the test content
	return []*Function{
		{Name: "main", StartLine: 5, EndLine: 7, Signature: "func main()"},
		{Name: "Greet", StartLine: 15, EndLine: 17, Signature: "func (u User) Greet() string"},
	}, nil
}

func (p *mockGoParser) GetClasses(result *ParseResult) ([]*Class, error) {
	// Return mock classes (structs in Go)
	return []*Class{
		{Name: "User", StartLine: 9, EndLine: 13},
	}, nil
}

func (p *mockGoParser) GetImports(result *ParseResult) ([]*Import, error) {
	// Return mock imports
	return []*Import{
		{Path: "fmt", Line: 3},
	}, nil
}

func (p *mockGoParser) FindSymbol(result *ParseResult, symbol string) ([]*Symbol, error) {
	// Simple symbol finding
	symbols := []*Symbol{}
	if symbol == "User" {
		symbols = append(symbols, &Symbol{Name: "User", Type: "struct", StartLine: 9, EndLine: 13})
	}
	return symbols, nil
}

func (p *mockGoParser) Language() Language {
	return LanguageGo
}

func (p *mockGoParser) Extensions() []string {
	return []string{".go"}
}

func TestASTTool_NewASTTool(t *testing.T) {
	tool := NewASTTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "ast", tool.Name())
	assert.Equal(t, "code", tool.Category())
	assert.NotNil(t, tool.registry)
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
	
	// Register a mock Go parser for testing
	tool.RegisterParser(LanguageGo, &mockGoParser{})

	// Test basic analysis
	params := ASTParams{
		File:  tmpFile.Name(),
		Query: "functions",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check the output contains expected functions
	assert.Contains(t, result.Output, "Functions")
	assert.Contains(t, result.Output, "main")
	assert.Contains(t, result.Output, "Greet")
}

func TestASTTool_Execute_PythonFile(t *testing.T) {
	t.Skip("Python parser not yet implemented")

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
		File:  tmpFile.Name(),
		Query: "classes",
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
		File:  "nonexistent.go",
		Query: "functions",
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
		File:  "",
		Query: "functions",
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
		File:  tmpFile.Name(),
		Query: "functions",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	_, err = tool.Execute(context.Background(), string(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language: unknown")
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
	
	// Register a mock Go parser for testing
	tool.RegisterParser(LanguageGo, &mockGoParser{})

	targets := []string{"functions", "classes", "imports", "all"}

	for _, target := range targets {
		t.Run("target_"+target, func(t *testing.T) {
			params := ASTParams{
				File:  tmpFile.Name(),
				Query: target,
			}

			input, err := json.Marshal(params)
			require.NoError(t, err)

			result, err := tool.Execute(context.Background(), string(input))
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Output, "(go)")
		})
	}
}
