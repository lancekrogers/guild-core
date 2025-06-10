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

func TestDependenciesTool_NewDependenciesTool(t *testing.T) {
	tool := NewDependenciesTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "dependencies", tool.Name())
	assert.Equal(t, "code", tool.Category())
}

func TestDependenciesTool_Execute_GoMod(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	goMod := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	github.com/spf13/cobra v1.7.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create some Go files
	mainGo := `package main

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

func main() {
	fmt.Println("Hello")
}
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath:   tmpDir,
		Format: "tree",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect Go project
	assert.Contains(t, result.Output, "Project Type: go")
	assert.Contains(t, result.Output, "Dependencies found:")
}

func TestDependenciesTool_Execute_PythonRequirements(t *testing.T) {
	// Create a temporary directory with requirements.txt
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create requirements.txt
	requirements := `requests==2.28.1
flask>=2.0.0,<3.0.0
numpy
pandas==1.5.2
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirements), 0644)
	require.NoError(t, err)

	// Create Python file
	pythonFile := `import requests
import flask
import numpy as np

def main():
    pass
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(pythonFile), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath:   tmpDir,
		Format: "list",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect Python project
	assert.Contains(t, result.Output, "Project Type: python")
	assert.Contains(t, result.Output, "requests")
	assert.Contains(t, result.Output, "flask")
}

func TestDependenciesTool_Execute_NodeJS(t *testing.T) {
	// Create a temporary directory with package.json
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create package.json
	packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0",
    "typescript": "^4.8.0"
  }
}`
	
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath:   tmpDir,
		Format: "json",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect Node.js project
	assert.Contains(t, result.Output, "Project Type: nodejs")
	assert.Contains(t, result.Output, "express")
	assert.Contains(t, result.Output, "lodash")
}

func TestDependenciesTool_Execute_InvalidPath(t *testing.T) {
	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath: "/nonexistent/path",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDependenciesTool_Execute_EmptyPath(t *testing.T) {
	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath: "",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDependenciesTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewDependenciesTool()
	
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDependenciesTool_Execute_UnknownProject(t *testing.T) {
	// Create a temporary directory without dependency files
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a random text file
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("Hello world"), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath: tmpDir,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should detect unknown project type
	assert.Contains(t, result.Output, "Project Type: unknown")
}

func TestDependenciesTool_Execute_WithFilters(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	github.com/spf13/cobra v1.7.0
	github.com/gin-gonic/gin v1.9.0
)
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath:    tmpDir,
		Format:  "summary",
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should include filtered dependencies
	assert.Contains(t, result.Output, "testify")
	assert.Contains(t, result.Output, "cobra")
}

func TestDependenciesTool_Execute_Outdated(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	params := DependenciesParams{
		ProjectPath:     tmpDir,
		CheckUpdates: true,
	}
	
	input, err := json.Marshal(params)
	require.NoError(t, err)
	
	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should include outdated check information
	assert.Contains(t, result.Output, "Project Type: go")
}

func TestDependenciesTool_Execute_AllFormats(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
	
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	tool := NewDependenciesTool()
	
	formats := []string{"list", "tree", "json", "graph"}
	
	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			params := DependenciesParams{
				ProjectPath:   tmpDir,
				Format: format,
			}
			
			input, err := json.Marshal(params)
			require.NoError(t, err)
			
			result, err := tool.Execute(context.Background(), string(input))
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Output, "Project Type: go")
		})
	}
}