// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// ShellTool provides command-line execution capabilities for agents
type ShellTool struct {
	*tools.BaseTool
	allowedCommands []string // List of allowed commands (empty means all are allowed)
	blockedCommands []string // List of blocked commands
	workingDir      string   // Working directory for commands
}

// ShellToolInput represents the input for shell command execution
type ShellToolInput struct {
	Command    string   `json:"command"`               // Command to execute
	Args       []string `json:"args,omitempty"`        // Command arguments
	Timeout    int      `json:"timeout,omitempty"`     // Timeout in seconds
	WorkingDir string   `json:"working_dir,omitempty"` // Working directory for the command
}

// ShellToolOptions contains options for the shell tool
type ShellToolOptions struct {
	AllowedCommands []string // List of allowed commands (empty means all are allowed)
	BlockedCommands []string // List of blocked commands
	WorkingDir      string   // Working directory for commands
}

// NewShellTool creates a new shell command tool
func NewShellTool(options ShellToolOptions) *ShellTool {
	// Set default working directory if not provided
	if options.WorkingDir == "" {
		options.WorkingDir, _ = os.Getwd()
	}

	// Add default blocked commands for safety
	defaultBlockedCommands := []string{
		"rm -rf /", "rm -rf /*", "rm -rf ~", "rm -rf ~/", "rm -rf ~/*",
		"mkfs", "dd", ">", ">>",
	}

	for _, cmd := range defaultBlockedCommands {
		if !contains(options.BlockedCommands, cmd) {
			options.BlockedCommands = append(options.BlockedCommands, cmd)
		}
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute",
			},
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Command arguments",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds",
				"default":     30,
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for the command",
			},
		},
		"required": []string{"command"},
	}

	examples := []string{
		`{"command": "ls", "args": ["-la"]}`,
		`{"command": "echo", "args": ["Hello, world!"]}`,
		`{"command": "pwd"}`,
		`{"command": "cat", "args": ["file.txt"]}`,
	}

	baseTool := tools.NewBaseTool(
		"shell",
		"Execute shell commands (with safety restrictions)",
		schema,
		"system",
		false,
		examples,
	)

	return &ShellTool{
		BaseTool:        baseTool,
		allowedCommands: options.AllowedCommands,
		blockedCommands: options.BlockedCommands,
		workingDir:      options.WorkingDir,
	}
}

// Execute runs the shell tool with the given input
func (t *ShellTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params ShellToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("shell_tool").
			WithOperation("execute")
	}

	// Validate command
	if params.Command == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "command is required", nil).
			WithComponent("shell_tool").
			WithOperation("execute")
	}

	// Check if command is allowed
	if !t.isCommandAllowed(params.Command) {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "command not allowed: %s", params.Command).
			WithComponent("shell_tool").
			WithOperation("execute")
	}

	// Check for blocked commands
	fullCmd := params.Command
	if len(params.Args) > 0 {
		fullCmd += " " + strings.Join(params.Args, " ")
	}

	if t.isCommandBlocked(fullCmd) {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "command is blocked for safety reasons: %s", fullCmd).
			WithComponent("shell_tool").
			WithOperation("execute")
	}

	// Set working directory
	workingDir := t.workingDir
	if params.WorkingDir != "" {
		workingDir = params.WorkingDir
	}

	// Set timeout if specified
	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, params.Command, params.Args...)
	cmd.Dir = workingDir

	// Capture output
	output, err := cmd.CombinedOutput()

	// Prepare metadata
	metadata := map[string]string{
		"command":     params.Command,
		"args":        strings.Join(params.Args, " "),
		"working_dir": workingDir,
		"exit_code":   "0",
		"timeout":     fmt.Sprintf("%d", params.Timeout),
	}

	// Handle errors
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			metadata["exit_code"] = "timeout"
			return tools.NewToolResult(string(output), metadata, gerror.Newf(gerror.ErrCodeInternal, "command timed out after %d seconds", params.Timeout).
				WithComponent("shell_tool").
				WithOperation("execute"), nil), nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			metadata["exit_code"] = fmt.Sprintf("%d", exitErr.ExitCode())
		}

		return tools.NewToolResult(string(output), metadata, gerror.Wrap(err, gerror.ErrCodeInternal, "command execution failed").
			WithComponent("shell_tool").
			WithOperation("execute"), nil), nil
	}

	return tools.NewToolResult(string(output), metadata, nil, nil), nil
}

// isCommandAllowed checks if a command is allowed
func (t *ShellTool) isCommandAllowed(command string) bool {
	// If no allowed commands are specified, all are allowed (except blocked ones)
	if len(t.allowedCommands) == 0 {
		return true
	}

	return contains(t.allowedCommands, command)
}

// isCommandBlocked checks if a command is blocked
func (t *ShellTool) isCommandBlocked(command string) bool {
	for _, blockedCmd := range t.blockedCommands {
		if strings.Contains(command, blockedCmd) {
			return true
		}
	}
	return false
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
