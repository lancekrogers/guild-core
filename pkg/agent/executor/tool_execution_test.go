package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/guild-ventures/guild-core/tools/fs"
	"github.com/guild-ventures/guild-core/tools/shell"
)

func TestTaskExecutor_ToolExecution(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "executor-tool-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create tool registry with real tools
	toolRegistry := tools.NewToolRegistry()
	
	// Register file tool
	fileTool := fs.NewFileTool(tmpDir)
	err = toolRegistry.RegisterTool(fileTool.Name(), fileTool)
	require.NoError(t, err)
	
	// Register shell tool with safety restrictions
	shellTool := shell.NewShellTool(shell.ShellToolOptions{
		WorkingDir: tmpDir,
		BlockedCommands: []string{"rm -rf /"},
	})
	err = toolRegistry.RegisterTool(shellTool.Name(), shellTool)
	require.NoError(t, err)

	// Create test context
	execContext := &ExecutionContext{
		WorkspaceDir: tmpDir,
		ProjectRoot:  tmpDir,
		AgentID:      "test-agent",
		AgentType:    "worker",
		Capabilities: []string{"coding", "testing"},
		Tools:        []string{"file", "shell"},
		Objective:    "Test tool execution",
	}

	// Create mock agent
	mockAgent := &mockAgent{
		id:   "test-agent",
		name: "Test Artisan",
	}

	// Create executor with tools
	executor, err := NewBasicTaskExecutor(mockAgent, nil, toolRegistry, execContext, nil)
	require.NoError(t, err)

	// Create test task
	task := &kanban.Task{
		ID:          "test-tool-task",
		Title:       "Test Tool Execution",
		Description: "Verify tools work correctly",
		Status:      kanban.StatusTodo,
		Priority:    kanban.PriorityHigh,
	}

	// Execute task
	ctx := context.Background()
	result, err := executor.Execute(ctx, task)
	
	// Basic checks
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, StatusCompleted, result.Status)

	// Verify tool usage was tracked
	assert.NotEmpty(t, result.ToolUsage)
	
	// Check for both tools
	var hasFileTool, hasShellTool bool
	for _, usage := range result.ToolUsage {
		if usage.ToolName == "file" {
			hasFileTool = true
			assert.Greater(t, usage.Invocations, 0)
		}
		if usage.ToolName == "shell" {
			hasShellTool = true
			assert.Greater(t, usage.Invocations, 0)
		}
	}
	assert.True(t, hasFileTool, "File tool should have been used")
	assert.True(t, hasShellTool, "Shell tool should have been used")

	// Verify artifacts were created
	assert.NotEmpty(t, result.Artifacts)
	
	// Check specific artifacts
	var hasReadme, hasSolution bool
	for _, artifact := range result.Artifacts {
		if artifact.Name == "README.md" {
			hasReadme = true
			assert.Equal(t, "documentation", artifact.Type)
		}
		if artifact.Name == "solution.sh" {
			hasSolution = true
			assert.Equal(t, "script", artifact.Type)
		}
	}
	assert.True(t, hasReadme, "README.md should have been created")
	assert.True(t, hasSolution, "solution.sh should have been created")

	// Verify physical files exist
	taskDir := filepath.Join(tmpDir, "task_"+task.ID)
	assert.DirExists(t, taskDir)
	assert.FileExists(t, filepath.Join(taskDir, "README.md"))
	assert.FileExists(t, filepath.Join(taskDir, "solution.sh"))
	assert.FileExists(t, filepath.Join(taskDir, "verification.md"))

	// Read and verify README content
	readmeContent, err := os.ReadFile(filepath.Join(taskDir, "README.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(readmeContent), task.Title)
	assert.Contains(t, string(readmeContent), task.Description)
}

func TestTaskExecutor_ToolSafety(t *testing.T) {
	// Create tool registry
	toolRegistry := tools.NewToolRegistry()
	
	// Register shell tool with restrictions
	shellTool := shell.NewShellTool(shell.ShellToolOptions{
		BlockedCommands: []string{"rm -rf /", "sudo"},
		AllowedCommands: []string{"echo", "ls", "pwd"}, // Whitelist mode
	})
	err := toolRegistry.RegisterTool(shellTool.Name(), shellTool)
	require.NoError(t, err)

	// Create executor
	execContext := &ExecutionContext{
		AgentID: "safety-test",
	}
	executor := &BasicTaskExecutor{
		agent:        &mockAgent{id: "safety-test"},
		toolRegistry: toolRegistry,
		execContext:  execContext,
		result: &ExecutionResult{
			ToolUsage: []ToolUsage{},
			Metadata:  make(map[string]interface{}),
		},
	}

	ctx := context.Background()

	// Test blocked command
	result, err := executor.executeToolCall(ctx, "shell", map[string]interface{}{
		"command": "rm",
		"args":    []string{"-rf", "/"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not allowed")

	// Test non-whitelisted command
	result, err = executor.executeToolCall(ctx, "shell", map[string]interface{}{
		"command": "cat",
		"args":    []string{"/etc/passwd"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Test allowed command
	result, err = executor.executeToolCall(ctx, "shell", map[string]interface{}{
		"command": "echo",
		"args":    []string{"hello", "world"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "hello world")
}

func TestTaskExecutor_FileToolRestrictions(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "file-tool-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create file tool restricted to tmpDir
	fileTool := fs.NewFileTool(tmpDir)
	toolRegistry := tools.NewToolRegistry()
	err = toolRegistry.RegisterTool(fileTool.Name(), fileTool)
	require.NoError(t, err)

	// Create executor
	executor := &BasicTaskExecutor{
		agent:        &mockAgent{id: "file-test"},
		toolRegistry: toolRegistry,
		execContext:  &ExecutionContext{AgentID: "file-test"},
		result: &ExecutionResult{
			ToolUsage: []ToolUsage{},
			Metadata:  make(map[string]interface{}),
		},
	}

	ctx := context.Background()

	// Test path traversal prevention
	result, err := executor.executeToolCall(ctx, "file", map[string]interface{}{
		"operation": "read",
		"path":      "../../../etc/passwd",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")

	// Test writing within allowed directory
	result, err = executor.executeToolCall(ctx, "file", map[string]interface{}{
		"operation": "write",
		"path":      "test.txt",
		"content":   "Hello, world!",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify file was created
	assert.FileExists(t, filepath.Join(tmpDir, "test.txt"))
}

func TestExecutor_BuildPromptDataWithTools(t *testing.T) {
	// Create tool registry with tools
	toolRegistry := tools.NewToolRegistry()
	
	fileTool := fs.NewFileTool("/tmp")
	toolRegistry.RegisterTool(fileTool.Name(), fileTool)
	
	shellTool := shell.NewShellTool(shell.ShellToolOptions{})
	toolRegistry.RegisterTool(shellTool.Name(), shellTool)

	// Create executor
	executor := &BasicTaskExecutor{
		agent:        &mockAgent{id: "prompt-test", name: "Test Agent"},
		toolRegistry: toolRegistry,
		execContext: &ExecutionContext{
			WorkspaceDir: "/tmp/workspace",
			Capabilities: []string{"testing"},
		},
		currentTask: &kanban.Task{
			ID:    "test-task",
			Title: "Test Task",
		},
		result: &ExecutionResult{
			StartTime: time.Now(),
			Metadata:  make(map[string]interface{}),
		},
	}

	// Build prompt data
	promptData := executor.buildPromptData()

	// Verify tools are included
	assert.Len(t, promptData.Tools, 2)
	
	// Check file tool
	var fileToolFound bool
	for _, tool := range promptData.Tools {
		if tool.Name == "file" {
			fileToolFound = true
			assert.Equal(t, "filesystem", tool.Category)
			assert.NotEmpty(t, tool.Description)
			assert.NotEmpty(t, tool.Parameters)
			assert.NotEmpty(t, tool.Examples)
		}
	}
	assert.True(t, fileToolFound, "File tool should be in prompt data")

	// Check shell tool
	var shellToolFound bool
	for _, tool := range promptData.Tools {
		if tool.Name == "shell" {
			shellToolFound = true
			assert.Equal(t, "system", tool.Category)
			assert.NotEmpty(t, tool.Description)
			assert.NotEmpty(t, tool.Parameters)
			assert.NotEmpty(t, tool.Examples)
		}
	}
	assert.True(t, shellToolFound, "Shell tool should be in prompt data")
}