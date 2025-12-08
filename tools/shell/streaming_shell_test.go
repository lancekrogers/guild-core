// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package shell_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/tools/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamingShellTool_Basic(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test basic command execution
	input := `{
		"command": "echo",
		"args": ["Hello, World!"],
		"stream_output": true,
		"timeout": 10
	}`

	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, "echo", shellResult.Command)
	assert.Equal(t, []string{"Hello, World!"}, shellResult.Args)
	assert.Equal(t, 0, shellResult.ExitCode)
	assert.Contains(t, shellResult.Output, "Hello, World!")
	assert.Greater(t, len(shellResult.OutputChunks), 0)
}

func TestStreamingShellTool_CommandFailure(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test command that should fail
	input := `{
		"command": "false",
		"timeout": 5
	}`

	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err) // Tool execution succeeds even if command fails

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, "false", shellResult.Command)
	assert.Equal(t, 1, shellResult.ExitCode) // false command returns exit code 1
}

func TestStreamingShellTool_Timeout(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test command with very short timeout
	input := `{
		"command": "sleep",
		"args": ["2"],
		"timeout": 1
	}`

	start := time.Now()
	result, err := tool.Execute(context.Background(), input)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 2*time.Second) // Should timeout before 2 seconds

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, -1, shellResult.ExitCode) // Timeout should result in -1
	assert.Contains(t, shellResult.Error, "timeout")
}

func TestStreamingShellTool_SecurityBlocking(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test blocked command
	input := `{
		"command": "rm",
		"args": ["-rf", "/"]
	}`

	result, err := tool.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked for safety reasons")
	assert.Nil(t, result)
}

func TestStreamingShellTool_WorkingDirectory(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test working directory change
	input := `{
		"command": "pwd",
		"working_dir": "/tmp"
	}`

	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, 0, shellResult.ExitCode)
	assert.Contains(t, shellResult.Output, "/tmp")
}

func TestStreamingShellTool_Environment(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test environment variable
	input := `{
		"command": "sh",
		"args": ["-c", "echo $TEST_VAR"],
		"environment": {
			"TEST_VAR": "test_value"
		}
	}`

	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, 0, shellResult.ExitCode)
	assert.Contains(t, shellResult.Output, "test_value")
}

func TestStreamingShellTool_InvalidInput(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test invalid JSON
	result, err := tool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid input")
	assert.Nil(t, result)

	// Test empty command
	input := `{
		"command": ""
	}`

	result, err = tool.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
	assert.Nil(t, result)
}

func TestStreamingShellTool_OutputChunks(t *testing.T) {
	tool := shell.NewStreamingShellTool(shell.ShellToolOptions{})

	// Test command that produces multiple lines of output
	input := `{
		"command": "sh",
		"args": ["-c", "echo line1; echo line2; echo line3"],
		"stream_output": true
	}`

	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	// Parse the result
	var shellResult shell.StreamingShellResult
	err = json.Unmarshal([]byte(result.Output), &shellResult)
	require.NoError(t, err)

	assert.Equal(t, 0, shellResult.ExitCode)
	assert.GreaterOrEqual(t, len(shellResult.OutputChunks), 3) // Should have at least 3 chunks

	// Check that chunks have proper timestamps and line numbers
	for i, chunk := range shellResult.OutputChunks {
		assert.NotZero(t, chunk.Timestamp)
		assert.Equal(t, i+1, chunk.LineNo)
		assert.Equal(t, "stdout", chunk.Type)
	}
}
