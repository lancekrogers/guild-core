// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package daemon provides utilities for managing the Guild gRPC server as a background daemon
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testExecutablePath allows tests to override the executable path
var testExecutablePath string

// SetTestExecutable sets a custom executable path for testing
// This should only be used in tests
func SetTestExecutable(path string) {
	testExecutablePath = path
}

// ResetTestExecutable resets the test executable override
func ResetTestExecutable() {
	testExecutablePath = ""
}

// getExecutablePath returns the guild executable path, with test override support
func getExecutablePath() (string, error) {
	if testExecutablePath != "" {
		return testExecutablePath, nil
	}
	
	// First, check standard installation locations
	standardPaths := []string{
		"/usr/local/bin/guild",
		"/usr/bin/guild",
		filepath.Join(os.Getenv("HOME"), ".guild", "bin", "guild"),
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "guild"),
	}
	
	for _, path := range standardPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// Second, try to find guild in PATH
	if guildPath, err := exec.LookPath("guild"); err == nil {
		// Resolve to absolute path
		if absPath, err := filepath.Abs(guildPath); err == nil {
			return absPath, nil
		}
		return guildPath, nil
	}
	
	// Last resort: use the current executable path
	// This might fail on macOS due to security restrictions
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("guild not found in standard locations or PATH, and could not determine current executable: %w", err)
	}
	
	// Resolve any symlinks and clean up the path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		// If we can't resolve symlinks, at least clean the path
		return filepath.Clean(execPath), nil
	}
	
	// Return the absolute, cleaned path
	absPath, err := filepath.Abs(realPath)
	if err != nil {
		return realPath, nil
	}
	return absPath, nil
}

// SkipIfNoBinary skips the test if the guild binary doesn't exist
func SkipIfNoBinary(t *testing.T) {
	t.Helper()

	// Try to find guild binary in common locations
	var guildPath string
	possiblePaths := []string{
		"./bin/guild",
		"../bin/guild",
		"../../bin/guild",
		"../../../bin/guild",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				guildPath = absPath
				break
			}
		}
	}

	if guildPath == "" {
		t.Skip("Skipping test: guild binary not found (run 'make build' first)")
	}

	SetTestExecutable(guildPath)
	t.Cleanup(ResetTestExecutable)
}
