package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/blockhead-consulting/guild/tools"
)

// FileTool provides file system operations for agents
type FileTool struct {
	*tools.BaseTool
	basePath string // Base path to restrict file operations to
}

// FileToolInput represents the input for file system operations
type FileToolInput struct {
	Operation string `json:"operation"` // read, write, list, exists, delete
	Path      string `json:"path"`      // File path (relative to base path)
	Content   string `json:"content,omitempty"`
}

// NewFileTool creates a new file system tool
func NewFileTool(basePath string) *FileTool {
	if basePath == "" {
		// Default to current directory if none provided
		basePath, _ = os.Getwd()
	}

	// Ensure the base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		os.MkdirAll(basePath, 0755)
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{"read", "write", "list", "exists", "delete"},
				"description": "File operation to perform",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path (relative to base path)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to file (only for write operation)",
			},
		},
		"required": []string{"operation", "path"},
	}

	examples := []string{
		`{"operation": "read", "path": "example.txt"}`,
		`{"operation": "write", "path": "example.txt", "content": "Hello, world!"}`,
		`{"operation": "list", "path": "."}`,
		`{"operation": "exists", "path": "example.txt"}`,
		`{"operation": "delete", "path": "example.txt"}`,
	}

	baseTool := tools.NewBaseTool(
		"file",
		"Perform file system operations like reading, writing, listing files, etc.",
		schema,
		"filesystem",
		false,
		examples,
	)

	return &FileTool{
		BaseTool: baseTool,
		basePath: basePath,
	}
}

// Execute runs the file tool with the given input
func (t *FileTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params FileToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate operation
	switch params.Operation {
	case "read", "write", "list", "exists", "delete":
		// Valid operations
	default:
		return nil, fmt.Errorf("invalid operation: %s", params.Operation)
	}

	// Ensure path is provided
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Get absolute path while ensuring it doesn't escape the base path
	path := t.sanitizePath(params.Path)
	if path == "" {
		return nil, fmt.Errorf("invalid path: %s", params.Path)
	}

	var output string
	var err error
	metadata := map[string]string{
		"operation": params.Operation,
		"path":      params.Path,
	}

	// Perform the requested operation
	switch params.Operation {
	case "read":
		output, err = t.readFile(path)
	case "write":
		output, err = t.writeFile(path, params.Content)
	case "list":
		output, err = t.listFiles(path)
	case "exists":
		output, err = t.fileExists(path)
	case "delete":
		output, err = t.deleteFile(path)
	}

	if err != nil {
		return tools.NewToolResult("", metadata, err, nil), err
	}

	return tools.NewToolResult(output, metadata, nil, nil), nil
}

// sanitizePath ensures the path doesn't escape the base path
func (t *FileTool) sanitizePath(path string) string {
	// Convert to absolute path
	absPath := filepath.Join(t.basePath, path)
	
	// Ensure the path is within the base path
	if !strings.HasPrefix(absPath, t.basePath) {
		return ""
	}
	
	return absPath
}

// readFile reads the content of a file
func (t *FileTool) readFile(path string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	// Read file content
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// writeFile writes content to a file
func (t *FileTool) writeFile(path string, content string) (string, error) {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

	// Write content to file
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

// listFiles lists files in a directory
func (t *FileTool) listFiles(path string) (string, error) {
	// Check if directory exists
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", path)
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	// List directory contents
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var result strings.Builder
	for _, file := range files {
		fileType := "file"
		if file.IsDir() {
			fileType = "directory"
		}
		result.WriteString(fmt.Sprintf("%s (%s, %d bytes)\n", file.Name(), fileType, file.Size()))
	}

	return result.String(), nil
}

// fileExists checks if a file exists
func (t *FileTool) fileExists(path string) (string, error) {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "false", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to check file: %w", err)
	}

	fileType := "file"
	if fileInfo.IsDir() {
		fileType = "directory"
	}

	return fmt.Sprintf("true (%s, %d bytes)", fileType, fileInfo.Size()), nil
}

// deleteFile deletes a file or directory
func (t *FileTool) deleteFile(path string) (string, error) {
	// Check if file exists
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	// Delete file or directory
	if fileInfo.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return "", fmt.Errorf("failed to delete directory: %w", err)
		}
		return fmt.Sprintf("Successfully deleted directory: %s", path), nil
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	return fmt.Sprintf("Successfully deleted file: %s", path), nil
}