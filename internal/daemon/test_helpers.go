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
	
	// First, try to find guild in PATH (equivalent to 'which guild')
	if guildPath, err := exec.LookPath("guild"); err == nil {
		return guildPath, nil
	}
	
	// Check GOPATH/bin if GOPATH is set
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		gopathBin := filepath.Join(gopath, "bin", "guild")
		if _, err := os.Stat(gopathBin); err == nil {
			return gopathBin, nil
		}
	}
	
	// Check ~/go/bin (default GOPATH location)
	defaultGoBin := filepath.Join(os.Getenv("HOME"), "go", "bin", "guild")
	if _, err := os.Stat(defaultGoBin); err == nil {
		return defaultGoBin, nil
	}
	
	// Check ~/.guild/bin (guild's own bin directory)
	guildBin := filepath.Join(os.Getenv("HOME"), ".guild", "bin", "guild")
	if _, err := os.Stat(guildBin); err == nil {
		return guildBin, nil
	}
	
	// Last resort: use the current executable path
	// This might fail on macOS due to security restrictions
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("guild not found in PATH, GOPATH/bin, or ~/.guild/bin: %w", err)
	}
	
	// Clean and return the path
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
