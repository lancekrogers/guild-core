// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

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
`

	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	// Create some Go files
	mainGo := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello")
}
`

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath: tmpDir,
		Format:      "tree",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should detect Go project
	assert.Contains(t, result.Output, "(go)")
	assert.Contains(t, result.Output, "Dependencies for")
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

	err = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirements), 0o644)
	require.NoError(t, err)

	// Create Python file
	pythonFile := `import requests
import flask
import numpy as np

def main():
    pass
`

	err = os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(pythonFile), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath: tmpDir,
		Format:      "list",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should detect Python project
	assert.Contains(t, result.Output, "(python)")
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

	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath: tmpDir,
		Format:      "json",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should detect Node.js project in JSON output
	assert.Contains(t, result.Output, `"project_type": "node"`)
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
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("Hello world"), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath: tmpDir,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDependenciesTool_Execute_WithFilters(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21
`

	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath: tmpDir,
		Format:      "summary",
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should detect Go project
	assert.Contains(t, result.Output, "(go)")
}

func TestDependenciesTool_Execute_Outdated(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21
`

	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	params := DependenciesParams{
		ProjectPath:  tmpDir,
		CheckUpdates: true,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(input))
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should include outdated check information
	assert.Contains(t, result.Output, "(go)")
}

func TestDependenciesTool_Execute_AllFormats(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir, err := os.MkdirTemp("", "deps_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	goMod := `module test-project

go 1.21
`

	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	tool := NewDependenciesTool()

	formats := []string{"list", "tree", "json", "graph"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			params := DependenciesParams{
				ProjectPath: tmpDir,
				Format:      format,
			}

			input, err := json.Marshal(params)
			require.NoError(t, err)

			result, err := tool.Execute(context.Background(), string(input))
			require.NoError(t, err)
			assert.NotNil(t, result)

			if format == "json" {
				assert.Contains(t, result.Output, `"project_type": "go"`)
			} else {
				assert.Contains(t, result.Output, "(go)")
			}
		})
	}
}
