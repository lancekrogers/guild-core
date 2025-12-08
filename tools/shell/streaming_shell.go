// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package shell

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// StreamingShellTool provides real-time streaming command execution
type StreamingShellTool struct {
	*tools.BaseTool
	allowedCommands []string
	blockedCommands []string
	workingDir      string
	activeProcesses sync.Map // Store active processes for cancellation
}

// StreamingShellInput represents input for streaming shell commands
type StreamingShellInput struct {
	Command         string            `json:"command"`                 // Command to execute
	Args            []string          `json:"args,omitempty"`          // Command arguments
	Timeout         int               `json:"timeout,omitempty"`       // Timeout in seconds
	WorkingDir      string            `json:"working_dir,omitempty"`   // Working directory
	StreamOutput    bool              `json:"stream_output,omitempty"` // Enable real-time streaming
	BufferSize      int               `json:"buffer_size,omitempty"`   // Output buffer size
	Environment     map[string]string `json:"environment,omitempty"`   // Environment variables
	InteractiveMode bool              `json:"interactive,omitempty"`   // Support interactive commands
}

// StreamingShellResult represents the result of a streaming command
type StreamingShellResult struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	ExitCode     int               `json:"exit_code"`
	Duration     time.Duration     `json:"duration"`
	Output       string            `json:"output"`        // Complete output
	OutputChunks []OutputChunk     `json:"output_chunks"` // Streaming chunks
	Error        string            `json:"error,omitempty"`
	ProcessID    int               `json:"process_id,omitempty"`
	Metadata     map[string]string `json:"metadata"`
}

// OutputChunk represents a chunk of streaming output
type OutputChunk struct {
	Type      string    `json:"type"`      // "stdout", "stderr"
	Data      string    `json:"data"`      // The actual output data
	Timestamp time.Time `json:"timestamp"` // When this chunk was received
	LineNo    int       `json:"line_no"`   // Line number in output
}

// ProcessHandle manages an active process
type ProcessHandle struct {
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	stdin     io.WriteCloser
	startTime time.Time
	done      chan bool
	output    *strings.Builder
	chunks    []OutputChunk
	mutex     sync.Mutex
}

// NewStreamingShellTool creates a new streaming shell tool
func NewStreamingShellTool(options ShellToolOptions) *StreamingShellTool {
	// Use same safety measures as regular shell tool
	if options.WorkingDir == "" {
		options.WorkingDir, _ = os.Getwd()
	}

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
				"default":     300,
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for the command",
			},
			"stream_output": map[string]interface{}{
				"type":        "boolean",
				"description": "Enable real-time output streaming",
				"default":     true,
			},
			"buffer_size": map[string]interface{}{
				"type":        "integer",
				"description": "Output buffer size in bytes",
				"default":     4096,
			},
			"environment": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables",
			},
			"interactive": map[string]interface{}{
				"type":        "boolean",
				"description": "Support interactive commands",
				"default":     false,
			},
		},
		"required": []string{"command"},
	}

	examples := []string{
		`{"command": "npm", "args": ["install"], "stream_output": true}`,
		`{"command": "make", "args": ["build"], "timeout": 600}`,
		`{"command": "docker", "args": ["build", "-t", "myapp", "."], "stream_output": true}`,
		`{"command": "go", "args": ["test", "-v", "./..."], "stream_output": true, "timeout": 300}`,
	}

	baseTool := tools.NewBaseTool(
		"streaming_shell",
		"Execute shell commands with real-time output streaming",
		schema,
		"system",
		false,
		examples,
	)

	return &StreamingShellTool{
		BaseTool:        baseTool,
		allowedCommands: options.AllowedCommands,
		blockedCommands: options.BlockedCommands,
		workingDir:      options.WorkingDir,
	}
}

// Execute runs the streaming shell tool
func (t *StreamingShellTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params StreamingShellInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("streaming_shell_tool").
			WithOperation("execute")
	}

	// Validate command
	if params.Command == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "command is required", nil).
			WithComponent("streaming_shell_tool").
			WithOperation("execute")
	}

	// Security checks (reuse from base shell tool)
	if !t.isCommandAllowed(params.Command) {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "command not allowed: %s", params.Command).
			WithComponent("streaming_shell_tool").
			WithOperation("execute")
	}

	fullCmd := params.Command
	if len(params.Args) > 0 {
		fullCmd += " " + strings.Join(params.Args, " ")
	}

	if t.isCommandBlocked(fullCmd) {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "command is blocked for safety reasons: %s", fullCmd).
			WithComponent("streaming_shell_tool").
			WithOperation("execute")
	}

	// Execute command with streaming
	result, err := t.executeWithStreaming(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert result to ToolResult
	resultJSON, _ := json.Marshal(result)
	metadata := map[string]string{
		"command":      result.Command,
		"args":         strings.Join(result.Args, " "),
		"exit_code":    fmt.Sprintf("%d", result.ExitCode),
		"duration":     result.Duration.String(),
		"streaming":    fmt.Sprintf("%t", params.StreamOutput),
		"chunks_count": fmt.Sprintf("%d", len(result.OutputChunks)),
	}

	var toolErr error
	if result.ExitCode != 0 {
		toolErr = gerror.Newf(gerror.ErrCodeInternal, "command failed with exit code %d", result.ExitCode).
			WithComponent("streaming_shell_tool").
			WithOperation("execute")
	}

	return tools.NewToolResult(string(resultJSON), metadata, toolErr, nil), nil
}

// executeWithStreaming executes a command with real-time output streaming
func (t *StreamingShellTool) executeWithStreaming(ctx context.Context, params StreamingShellInput) (*StreamingShellResult, error) {
	// Set defaults
	if params.Timeout == 0 {
		params.Timeout = 300 // 5 minutes default
	}
	if params.BufferSize == 0 {
		params.BufferSize = 4096
	}

	// Set working directory
	workingDir := t.workingDir
	if params.WorkingDir != "" {
		workingDir = params.WorkingDir
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, params.Command, params.Args...)
	cmd.Dir = workingDir

	// Set environment variables
	if len(params.Environment) > 0 {
		env := append(cmd.Env, os.Environ()...)
		for key, value := range params.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stdout pipe").
			WithComponent("streaming_shell_tool").
			WithOperation("executeWithStreaming")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stderr pipe").
			WithComponent("streaming_shell_tool").
			WithOperation("executeWithStreaming")
	}

	var stdin io.WriteCloser
	if params.InteractiveMode {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stdin pipe").
				WithComponent("streaming_shell_tool").
				WithOperation("executeWithStreaming")
		}
	}

	// Start command
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start command").
			WithComponent("streaming_shell_tool").
			WithOperation("executeWithStreaming")
	}

	// Create process handle
	handle := &ProcessHandle{
		cmd:       cmd,
		cancel:    cancel,
		stdout:    stdout,
		stderr:    stderr,
		stdin:     stdin,
		startTime: startTime,
		done:      make(chan bool),
		output:    &strings.Builder{},
		chunks:    make([]OutputChunk, 0),
	}

	// Store process handle for potential cancellation
	processID := cmd.Process.Pid
	t.activeProcesses.Store(processID, handle)
	defer t.activeProcesses.Delete(processID)

	// Stream output if enabled
	var wg sync.WaitGroup
	if params.StreamOutput {
		wg.Add(2)
		go t.streamOutput(handle, stdout, "stdout", params.BufferSize, &wg)
		go t.streamOutput(handle, stderr, "stderr", params.BufferSize, &wg)
	}

	// Wait for command completion
	go func() {
		wg.Wait()
		close(handle.done)
	}()

	// Wait for completion or timeout
	err = cmd.Wait()
	<-handle.done

	duration := time.Since(startTime)
	exitCode := 0
	errorMsg := ""

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			exitCode = -1
			errorMsg = fmt.Sprintf("command timed out after %d seconds", params.Timeout)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			errorMsg = exitErr.Error()
		} else {
			exitCode = -1
			errorMsg = err.Error()
		}
	}

	result := &StreamingShellResult{
		Command:      params.Command,
		Args:         params.Args,
		ExitCode:     exitCode,
		Duration:     duration,
		Output:       handle.output.String(),
		OutputChunks: handle.chunks,
		Error:        errorMsg,
		ProcessID:    processID,
		Metadata: map[string]string{
			"working_dir": workingDir,
			"streaming":   fmt.Sprintf("%t", params.StreamOutput),
			"interactive": fmt.Sprintf("%t", params.InteractiveMode),
		},
	}

	return result, nil
}

// streamOutput reads output from a pipe and streams it in chunks
func (t *StreamingShellTool) streamOutput(handle *ProcessHandle, pipe io.ReadCloser, outputType string, bufferSize int, wg *sync.WaitGroup) {
	defer wg.Done()
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	lineNo := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		handle.mutex.Lock()

		// Add to complete output
		handle.output.WriteString(line + "\n")

		// Create output chunk
		chunk := OutputChunk{
			Type:      outputType,
			Data:      line,
			Timestamp: time.Now(),
			LineNo:    lineNo,
		}
		handle.chunks = append(handle.chunks, chunk)

		handle.mutex.Unlock()

		// TODO: In a real implementation, you might want to send these chunks
		// to a callback or channel for real-time processing
	}

	if err := scanner.Err(); err != nil {
		handle.mutex.Lock()
		errorChunk := OutputChunk{
			Type:      "error",
			Data:      fmt.Sprintf("Error reading %s: %v", outputType, err),
			Timestamp: time.Now(),
			LineNo:    lineNo + 1,
		}
		handle.chunks = append(handle.chunks, errorChunk)
		handle.mutex.Unlock()
	}
}

// CancelProcess cancels a running process by ID
func (t *StreamingShellTool) CancelProcess(processID int) error {
	if handle, ok := t.activeProcesses.Load(processID); ok {
		if h, ok := handle.(*ProcessHandle); ok {
			h.cancel()
			if h.cmd.Process != nil {
				return h.cmd.Process.Kill()
			}
		}
	}
	return gerror.Newf(gerror.ErrCodeNotFound, "process %d not found", processID).
		WithComponent("streaming_shell_tool").
		WithOperation("cancel_process")
}

// GetActiveProcesses returns a list of active process IDs
func (t *StreamingShellTool) GetActiveProcesses() []int {
	var processes []int
	t.activeProcesses.Range(func(key, value interface{}) bool {
		if pid, ok := key.(int); ok {
			processes = append(processes, pid)
		}
		return true
	})
	return processes
}

// Reuse security methods from base shell tool
func (t *StreamingShellTool) isCommandAllowed(command string) bool {
	if len(t.allowedCommands) == 0 {
		return true
	}
	return contains(t.allowedCommands, command)
}

func (t *StreamingShellTool) isCommandBlocked(command string) bool {
	for _, blockedCmd := range t.blockedCommands {
		if strings.Contains(command, blockedCmd) {
			return true
		}
	}
	return false
}
