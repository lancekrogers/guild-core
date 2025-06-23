// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package daemon provides utilities for managing the Guild gRPC server as a background daemon
package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
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
	
	// Try to find guild in PATH (equivalent to 'which guild')
	if guildPath, err := exec.LookPath("guild"); err == nil {
		return guildPath, nil
	}
	
	// Fallback to current executable
	// Note: This may fail on macOS with "operation not permitted" when running
	// from non-standard locations. Install guild properly to avoid this issue.
	execPath, err := os.Executable()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "guild not found in PATH").
			WithComponent("daemon").
			WithOperation("getExecutablePath").
			WithDetails("help", "Run 'make install' to install guild properly")
	}
	
	return filepath.Clean(execPath), nil
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
