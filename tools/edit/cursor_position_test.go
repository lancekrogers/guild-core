package edit

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorPositionTool_NewCursorPositionTool(t *testing.T) {
	tool := NewCursorPositionTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "cursor_position", tool.Name())
	assert.Equal(t, "edit", tool.Category())
	assert.NotNil(t, tool.marks)
}

func TestCursorPositionTool_Execute_LineColumn(t *testing.T) {
	// Create a temporary file
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Line:   6,
		Column: 10,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should navigate to position
	assert.Contains(t, result.Output, "Position: Line 6, Column 10")
	assert.Contains(t, result.Output, "Language: go")
	assert.Contains(t, result.Output, "Context: function")
}

func TestCursorPositionTool_Execute_FindSymbol(t *testing.T) {
	// Create a temporary Go file
	content := `package main

import "fmt"

type User struct {
	Name string
	Age  int
}

func main() {
	fmt.Println("Hello, World!")
}

func (u User) Greet() string {
	return "Hello, " + u.Name
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Symbol: "Greet",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find the Greet function
	assert.Contains(t, result.Output, "Line 14")
	assert.Contains(t, result.Output, "Function: Greet")
}

func TestCursorPositionTool_Execute_SetMark(t *testing.T) {
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

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:    tmpFile.Name(),
		Line:    3,
		Column:  1,
		SetMark: "important_line",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should set mark
	assert.Contains(t, result.Output, "Mark 'important_line' set")
	assert.Contains(t, result.Output, "Position: Line 3, Column 1")
	
	// Verify mark was stored
	assert.Contains(t, tool.marks, "important_line")
	assert.Equal(t, 3, tool.marks["important_line"].Line)
}

func TestCursorPositionTool_Execute_GoToMark(t *testing.T) {
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

	tool := NewCursorPositionTool()
	
	// First, set a mark
	tool.marks["test_mark"] = &Position{
		Line:   4,
		Column: 5,
	}
	
	params := CursorParams{
		File:     tmpFile.Name(),
		GoToMark: "test_mark",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should go to mark
	assert.Contains(t, result.Output, "Jumped to mark 'test_mark'")
	assert.Contains(t, result.Output, "Position: Line 4, Column 5")
}

func TestCursorPositionTool_Execute_PatternSearch(t *testing.T) {
	// Create a temporary file
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	// TODO: Add error handling
	fmt.Printf("Testing")
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:    tmpFile.Name(),
		Pattern: "TODO",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find the TODO comment
	assert.Contains(t, result.Output, "Line 7")
	assert.Contains(t, result.Output, "TODO")
}

func TestCursorPositionTool_Execute_PythonSymbol(t *testing.T) {
	// Create a temporary Python file
	content := `def main():
    print("Hello, World!")

class User:
    def __init__(self, name):
        self.name = name
    
    def greet(self):
        return f"Hello, {self.name}"

if __name__ == "__main__":
    main()
`
	
	tmpFile, err := os.CreateTemp("", "test_*.py")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Symbol: "User",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find the User class
	assert.Contains(t, result.Output, "Line 4")
	assert.Contains(t, result.Output, "Language: python")
}

func TestCursorPositionTool_Execute_InvalidFile(t *testing.T) {
	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File: "nonexistent.go",
		Line: 1,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCursorPositionTool_Execute_EmptyFile(t *testing.T) {
	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File: "",
		Line: 1,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCursorPositionTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewCursorPositionTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCursorPositionTool_Execute_NonexistentMark(t *testing.T) {
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

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:     tmpFile.Name(),
		GoToMark: "nonexistent_mark",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCursorPositionTool_Execute_OutOfBounds(t *testing.T) {
	// Create a small file
	content := `package main
func main() {}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Line:   100, // Beyond file length
		Column: 1,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle out of bounds gracefully
	assert.Contains(t, result.Output, "Target not found")
}

func TestCursorPositionTool_Execute_NoOperation(t *testing.T) {
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

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File: tmpFile.Name(),
		// No operation specified
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCursorPositionTool_Execute_GenericSymbolSearch(t *testing.T) {
	// Create a file with unknown extension
	content := `function main() {
    console.log("Hello, World!");
}

function helper() {
    return "test";
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.js")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Symbol: "helper",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should find symbol using generic search
	assert.Contains(t, result.Output, "Line 5")
	assert.Contains(t, result.Output, "helper")
}

func TestCursorPositionTool_Execute_SymbolNotFound(t *testing.T) {
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

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Symbol: "nonexistent_function",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle symbol not found
	assert.Contains(t, result.Output, "Target not found")
}

func TestCursorPositionTool_Execute_GoContextDetection(t *testing.T) {
	// Create a Go file with nested functions and structs
	content := `package main

import "fmt"

type Server struct {
	Port int
	Host string
}

func (s *Server) Start() error {
	fmt.Printf("Starting server on %s:%d\n", s.Host, s.Port)
	return nil
}

func main() {
	server := &Server{
		Port: 8080,
		Host: "localhost",
	}
	server.Start()
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:   tmpFile.Name(),
		Line:   10, // Inside Start method
		Column: 5,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect function and struct context
	assert.Contains(t, result.Output, "Function: Start")
	assert.Contains(t, result.Output, "Context: function")
}

func TestCursorPositionTool_Execute_DirectionalNavigation(t *testing.T) {
	// Create a temporary file
	content := `package main

func main() {
	println("test")
}

func helper() {
	return
}
`
	
	tmpFile, err := os.CreateTemp("", "test_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	tool := NewCursorPositionTool()
	
	params := CursorParams{
		File:      tmpFile.Name(),
		Direction: "next",
		Target:    "function",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should handle directional navigation (even if not fully implemented)
	assert.Contains(t, result.Output, "not yet fully implemented")
}