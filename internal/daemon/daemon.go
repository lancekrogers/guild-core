// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package daemon provides utilities for managing the Guild gRPC server as a background daemon
package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

const (
	defaultPort    = "9090"
	pidFileName    = "daemon.pid"
	logFileName    = "daemon.log"
	maxStartupWait = 10 * time.Second
	checkInterval  = 100 * time.Millisecond
)

// GetPIDFilePath returns the path to the PID file
func GetPIDFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".guild", pidFileName)
}

// GetLogFilePath returns the path to the daemon log file
func GetLogFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".guild", logFileName)
}

// IsRunning checks if the daemon process is running by verifying both PID and port
func IsRunning() bool {
	// First check PID file
	pidFile := GetPIDFilePath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if alive
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, clean up PID file
		os.Remove(pidFile)
		return false
	}

	// Also check if port is actually listening
	// Use a short timeout context for this check
	checkCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return isPortListening(checkCtx, defaultPort)
}

// IsReachable checks if the gRPC server is reachable on the configured port
func IsReachable(ctx context.Context) bool {
	return isPortListening(ctx, defaultPort)
}

// EnsureRunning starts the daemon if not already running
func EnsureRunning(ctx context.Context) error {
	if IsReachable(ctx) {
		return nil // Already running
	}

	// Check for stale PID file
	if pidFileExists() && !IsRunning() {
		if err := os.Remove(GetPIDFilePath()); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove stale PID file").
				WithComponent("daemon").
				WithOperation("EnsureRunning")
		}
	}

	// Start the daemon
	return Start(ctx)
}

// Start launches the guild serve command in background
func Start(ctx context.Context) error {
	if IsRunning() {
		return gerror.New(gerror.ErrCodeAlreadyExists, "server already running", nil).
			WithComponent("daemon").
			WithOperation("Start")
	}

	// Get guild executable path
	guildPath, err := getExecutablePath()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find guild executable").
			WithComponent("daemon").
			WithOperation("Start")
	}

	// Ensure .guild directory exists
	guildDir := filepath.Dir(GetPIDFilePath())
	if err := os.MkdirAll(guildDir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create .guild directory").
			WithComponent("daemon").
			WithOperation("Start").
			WithDetails("directory", guildDir)
	}

	// Prepare command
	cmd := exec.Command(guildPath, "serve", "--daemon")

	// Set up for background execution
	// On macOS, setting Setsid can cause "operation not permitted" errors
	// when the binary is executed from certain locations
	if runtime.GOOS != "darwin" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // New session
		}
	}

	// Redirect output to log file
	logPath := GetLogFilePath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to open log file").
			WithComponent("daemon").
			WithOperation("Start").
			WithDetails("log_path", logPath)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start server").
			WithComponent("daemon").
			WithOperation("Start")
	}

	// Write PID file
	pidFile := GetPIDFilePath()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		cmd.Process.Kill()
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write PID file").
			WithComponent("daemon").
			WithOperation("Start").
			WithDetails("pid_file", pidFile)
	}

	// Release the process so it continues after parent exits
	if err := cmd.Process.Release(); err != nil {
		// Log warning but don't fail - process might still work
		// This is non-critical on some systems
	}

	// Wait for server to be ready
	waitCtx, cancel := context.WithTimeout(ctx, maxStartupWait)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			// Try to clean up
			Stop()
			return gerror.New(gerror.ErrCodeTimeout, "server failed to start within timeout", nil).
				WithComponent("daemon").
				WithOperation("Start").
				WithDetails("timeout", maxStartupWait.String())
		case <-ticker.C:
			if IsReachable(waitCtx) {
				return nil // Success!
			}
		}
	}
}

// Stop terminates the daemon process gracefully
func Stop() error {
	pidFile := GetPIDFilePath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return gerror.New(gerror.ErrCodeNotFound, "server not running", nil).
				WithComponent("daemon").
				WithOperation("Stop")
		}
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read PID file").
			WithComponent("daemon").
			WithOperation("Stop")
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid PID file").
			WithComponent("daemon").
			WithOperation("Stop").
			WithDetails("content", string(data))
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find process").
			WithComponent("daemon").
			WithOperation("Stop").
			WithDetails("pid", pid)
	}

	// Try graceful shutdown first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		os.Remove(pidFile)
		return nil
	}

	// Wait a bit for graceful shutdown
	for i := 0; i < 50; i++ {
		if !isProcessRunning(pid) {
			os.Remove(pidFile)
			return nil
		}
		time.Sleep(checkInterval)
	}

	// Force kill if still running
	if err := process.Kill(); err != nil {
		// Log error but continue to remove PID file
	}
	os.Remove(pidFile)

	return nil
}

// Status returns detailed status information about the daemon
func Status() (string, error) {
	if !IsRunning() {
		if pidFileExists() {
			return "Guild server has stale PID file (cleaning up)", nil
		}
		return "Guild server is stopped", nil
	}

	pidFile := GetPIDFilePath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "failed to read PID file").
			WithComponent("daemon").
			WithOperation("Status")
	}

	pid := string(data)
	port := defaultPort

	// Check if actually reachable
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if IsReachable(checkCtx) {
		return fmt.Sprintf("Guild server is running (PID: %s, Port: %s)", pid, port), nil
	}

	return fmt.Sprintf("Guild server process exists (PID: %s) but not reachable on port %s", pid, port), nil
}

// Helper functions

func isPortListening(ctx context.Context, port string) bool {
	// Create a context-aware dialer
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", "localhost:"+port)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func pidFileExists() bool {
	_, err := os.Stat(GetPIDFilePath())
	return err == nil
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// CleanupStaleFiles removes any stale PID files or locks
func CleanupStaleFiles() error {
	pidFile := GetPIDFilePath()

	// Check if PID file exists
	if !pidFileExists() {
		return nil
	}

	// Read the PID
	data, err := os.ReadFile(pidFile)
	if err != nil {
		// If we can't read it, just try to remove it
		os.Remove(pidFile)
		return nil
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		// Invalid PID file, remove it
		os.Remove(pidFile)
		return nil
	}

	// Check if process is still running
	if !isProcessRunning(pid) {
		// Process is dead, remove PID file
		if err := os.Remove(pidFile); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove stale PID file").
				WithComponent("daemon").
				WithOperation("CleanupStaleFiles")
		}
	}

	return nil
}
