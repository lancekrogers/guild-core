// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// TestEnvironment provides isolated testing environment for E2E tests
type TestEnvironment struct {
	t          *testing.T
	workDir    string
	guildBin   string
	cleanup    []func()
	recordings []Recording
	env        []string
}

// Recording captures command execution details for debugging
type Recording struct {
	Name     string
	Command  []string
	Output   string
	Error    string
	ExitCode int
	Time     time.Duration
}

// CommandResult contains the result of executing a guild command
type CommandResult struct {
	Stdout   string
	Stderr   string
	Error    error
	ExitCode int
	Duration time.Duration
}

// NewTestEnvironment creates a new isolated test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	// Create temp directory for test isolation
	workDir, err := os.MkdirTemp("", "guild-e2e-*")
	require.NoError(t, err, "Failed to create temp directory")

	// Build guild binary in temp location
	guildBin := filepath.Join(workDir, "guild")
	if runtime.GOOS == "windows" {
		guildBin += ".exe"
	}

	// Build from the current directory (should be guild-core)
	buildCmd := exec.Command("go", "build", "-o", guildBin, "./cmd/guild")
	buildCmd.Dir = getCurrentModuleRoot()
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build guild binary: %s", string(buildOutput))

	env := &TestEnvironment{
		t:        t,
		workDir:  workDir,
		guildBin: guildBin,
		env: []string{
			"GUILD_MOCK_PROVIDER=true", // Enable mock provider
			"GUILD_TEST_MODE=true",     // Enable test mode features
			"NO_COLOR=1",               // Disable colors for easier assertions
			"GUILD_LOG_LEVEL=warn",     // Reduce log noise
			"HOME=" + workDir,          // Isolate home directory
		},
	}

	// Add current environment but override specific vars
	for _, envVar := range os.Environ() {
		key := strings.Split(envVar, "=")[0]
		switch key {
		case "GUILD_MOCK_PROVIDER", "GUILD_TEST_MODE", "NO_COLOR", "GUILD_LOG_LEVEL", "HOME":
			// Skip - we set these ourselves
		default:
			env.env = append(env.env, envVar)
		}
	}

	// Setup cleanup
	t.Cleanup(func() {
		for _, fn := range env.cleanup {
			fn()
		}
		// Clean up temp directory
		if err := os.RemoveAll(workDir); err != nil {
			t.Logf("Warning: failed to clean up temp dir %s: %v", workDir, err)
		}
	})

	return env
}

// RunGuild executes a guild command in the test environment
func (e *TestEnvironment) RunGuild(args ...string) *CommandResult {
	return e.RunGuildWithTimeout(30*time.Second, args...)
}

// RunGuildWithTimeout executes a guild command with a custom timeout
func (e *TestEnvironment) RunGuildWithTimeout(timeout time.Duration, args ...string) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.guildBin, args...)
	cmd.Dir = e.workDir
	cmd.Env = e.env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	result := &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Error:    err,
		ExitCode: exitCode,
		Duration: duration,
	}

	// Record for debugging
	e.recordings = append(e.recordings, Recording{
		Name:     strings.Join(args, " "),
		Command:  append([]string{e.guildBin}, args...),
		Output:   result.Stdout,
		Error:    result.Stderr,
		ExitCode: exitCode,
		Time:     duration,
	})

	return result
}

// StartGuildInteractive starts guild in interactive mode for testing
func (e *TestEnvironment) StartGuildInteractive(args ...string) (*InteractiveSession, error) {
	// Add --no-tui flag for text-mode testing
	fullArgs := append(args, "--no-tui")

	cmd := exec.Command(e.guildBin, fullArgs...)
	cmd.Dir = e.workDir
	cmd.Env = e.env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create stdin pipe").
			WithComponent("e2e").
			WithOperation("StartGuildInteractive")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create stdout pipe").
			WithComponent("e2e").
			WithOperation("StartGuildInteractive")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create stderr pipe").
			WithComponent("e2e").
			WithOperation("StartGuildInteractive")
	}

	if err := cmd.Start(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start guild process").
			WithComponent("e2e").
			WithOperation("StartGuildInteractive")
	}

	session := &InteractiveSession{
		env:    e,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
		stderr: bufio.NewScanner(stderr),
	}

	// Register cleanup
	e.cleanup = append(e.cleanup, func() {
		session.Stop()
	})

	return session, nil
}

// CreateFile creates a file in the test environment
func (e *TestEnvironment) CreateFile(relativePath, content string) error {
	fullPath := filepath.Join(e.workDir, relativePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directory").
			WithComponent("e2e").
			WithOperation("CreateFile").
			WithDetails("dir", dir)
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// FileExists checks if a file exists in the test environment
func (e *TestEnvironment) FileExists(relativePath string) bool {
	fullPath := filepath.Join(e.workDir, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads a file from the test environment
func (e *TestEnvironment) ReadFile(relativePath string) (string, error) {
	fullPath := filepath.Join(e.workDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "failed to read file").
			WithComponent("e2e").
			WithOperation("ReadFile").
			WithDetails("path", relativePath)
	}
	return string(content), nil
}

// SaveRecording saves the test recording for debugging
func (e *TestEnvironment) SaveRecording(name string) {
	recordingsDir := filepath.Join("recordings")
	if err := os.MkdirAll(recordingsDir, 0755); err != nil {
		e.t.Logf("Warning: failed to create recordings directory: %v", err)
		return
	}

	file, err := os.Create(filepath.Join(recordingsDir, name+".log"))
	if err != nil {
		e.t.Logf("Warning: failed to create recording file: %v", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "Guild E2E Test Recording: %s\n", name)
	fmt.Fprintf(file, "Time: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Work Dir: %s\n", e.workDir)
	fmt.Fprintf(file, "Guild Binary: %s\n", e.guildBin)
	fmt.Fprintf(file, "Environment:\n")
	for _, env := range e.env {
		if strings.HasPrefix(env, "GUILD_") || strings.HasPrefix(env, "NO_COLOR") {
			fmt.Fprintf(file, "  %s\n", env)
		}
	}
	fmt.Fprintf(file, "===============================================\n\n")

	for i, rec := range e.recordings {
		fmt.Fprintf(file, "Command %d: %s\n", i+1, strings.Join(rec.Command, " "))
		fmt.Fprintf(file, "Duration: %v\n", rec.Time)
		fmt.Fprintf(file, "Exit Code: %d\n", rec.ExitCode)
		fmt.Fprintf(file, "Stdout:\n%s\n", rec.Output)
		if rec.Error != "" {
			fmt.Fprintf(file, "Stderr:\n%s\n", rec.Error)
		}
		fmt.Fprintf(file, "-----------------------------------------------\n\n")
	}
}

// GetWorkDir returns the temporary work directory
func (e *TestEnvironment) GetWorkDir() string {
	return e.workDir
}

// InteractiveSession handles interactive guild sessions
type InteractiveSession struct {
	env    *TestEnvironment
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bufio.Scanner
}

// SendMessage sends a message to the interactive session
func (s *InteractiveSession) SendMessage(msg string) error {
	_, err := fmt.Fprintf(s.stdin, "%s\n", msg)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to send message").
			WithComponent("e2e").
			WithOperation("SendMessage")
	}
	return nil
}

// WaitForResponse waits for a response from the session
func (s *InteractiveSession) WaitForResponse(timeout time.Duration) (string, error) {
	responseC := make(chan string, 1)
	errorC := make(chan error, 1)

	go func() {
		var response strings.Builder
		for s.stdout.Scan() {
			line := s.stdout.Text()
			response.WriteString(line + "\n")

			// Look for end of response markers
			if strings.Contains(line, "> ") || strings.Contains(line, "guild>") {
				break
			}
		}

		if err := s.stdout.Err(); err != nil {
			errorC <- err
			return
		}

		responseC <- response.String()
	}()

	select {
	case response := <-responseC:
		return response, nil
	case err := <-errorC:
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "error reading response").
			WithComponent("e2e").
			WithOperation("WaitForResponse")
	case <-time.After(timeout):
		return "", gerror.New(gerror.ErrCodeTimeout, "timeout waiting for response", nil).
			WithComponent("e2e").
			WithOperation("WaitForResponse").
			WithDetails("timeout", timeout.String())
	}
}

// Stop stops the interactive session
func (s *InteractiveSession) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		s.stdin.Close()
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
}

// Command result assertion methods

// AssertSuccess asserts that the command succeeded
func (r *CommandResult) AssertSuccess(t *testing.T) {
	if r.Error != nil {
		t.Fatalf("Command failed: %v\nExit Code: %d\nStderr: %s\nStdout: %s",
			r.Error, r.ExitCode, r.Stderr, r.Stdout)
	}
	assert.Equal(t, 0, r.ExitCode, "Command should exit with code 0")
}

// AssertFailure asserts that the command failed
func (r *CommandResult) AssertFailure(t *testing.T) {
	assert.NotEqual(t, 0, r.ExitCode, "Command should fail")
}

// AssertContains asserts that stdout contains the expected string
func (r *CommandResult) AssertContains(t *testing.T, expected string) {
	assert.Contains(t, r.Stdout, expected,
		"Output should contain '%s'\nActual output:\n%s", expected, r.Stdout)
}

// AssertNotContains asserts that stdout does not contain the unexpected string
func (r *CommandResult) AssertNotContains(t *testing.T, unexpected string) {
	assert.NotContains(t, r.Stdout, unexpected,
		"Output should not contain '%s'\nActual output:\n%s", unexpected, r.Stdout)
}

// AssertStderrContains asserts that stderr contains the expected string
func (r *CommandResult) AssertStderrContains(t *testing.T, expected string) {
	assert.Contains(t, r.Stderr, expected,
		"Error output should contain '%s'\nActual stderr:\n%s", expected, r.Stderr)
}

// AssertFasterThan asserts that the command completed within the time limit
func (r *CommandResult) AssertFasterThan(t *testing.T, maxDuration time.Duration) {
	assert.Less(t, r.Duration, maxDuration,
		"Command took too long: %v (max: %v)", r.Duration, maxDuration)
}

// AssertSlowerThan asserts that the command took at least the minimum time
func (r *CommandResult) AssertSlowerThan(t *testing.T, minDuration time.Duration) {
	assert.Greater(t, r.Duration, minDuration,
		"Command completed too quickly: %v (min: %v)", r.Duration, minDuration)
}

// Helper functions

// getCurrentModuleRoot finds the root of the current Go module
func getCurrentModuleRoot() string {
	// Start from current directory and walk up looking for go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "." // fallback to current directory
}
