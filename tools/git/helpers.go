// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// executeGitCommand runs a git command in the specified directory
func executeGitCommand(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "git command failed").
			WithComponent("tools.git").
			WithOperation("execute").
			WithDetails("command", "git "+strings.Join(args, " ")).
			WithDetails("output", string(output))
	}

	return string(output), nil
}

// validatePathWithBase ensures a path is within the workspace boundaries
func validatePathWithBase(basePath, path string) error {
	if path == "" {
		return nil // Empty path is valid (means workspace root)
	}

	// Check for absolute paths first
	if filepath.IsAbs(path) {
		return gerror.New(gerror.ErrCodeInvalidInput, "absolute paths not allowed", nil).
			WithComponent("tools.git").
			WithOperation("validate_path").
			WithDetails("path", path).
			WithDetails("base", basePath)
	}

	absPath, err := filepath.Abs(filepath.Join(basePath, path))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve path").
			WithComponent("tools.git").
			WithOperation("validate_path")
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve base path").
			WithComponent("tools.git").
			WithOperation("validate_path")
	}

	if !strings.HasPrefix(absPath, absBase) {
		return gerror.New(gerror.ErrCodeInvalidInput, "path outside workspace", nil).
			WithComponent("tools.git").
			WithOperation("validate_path").
			WithDetails("path", path).
			WithDetails("base", basePath)
	}

	return nil
}

// formatGitError formats git command errors for better readability
func formatGitError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if it's already a gerror
	if gerr, ok := err.(*gerror.GuildError); ok {
		return gerr
	}

	// Extract git error message
	errMsg := err.Error()
	if strings.Contains(errMsg, "exit status") {
		// Remove exit status prefix for cleaner error messages
		parts := strings.SplitN(errMsg, ":", 2)
		if len(parts) > 1 {
			errMsg = strings.TrimSpace(parts[1])
		}
	}

	return gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("git %s failed: %s", operation, errMsg), nil).
		WithComponent("tools.git").
		WithOperation(operation)
}

// truncateOutput limits output size to prevent overwhelming the system
func truncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	truncated := lines[:maxLines]
	truncated = append(truncated, fmt.Sprintf("... truncated %d lines ...", len(lines)-maxLines))
	return strings.Join(truncated, "\n")
}

// sanitizeGitOutput removes any sensitive information from git output
func sanitizeGitOutput(output string) string {
	// Remove potential email addresses in angle brackets
	// but keep the author name
	output = strings.ReplaceAll(output, "<", "&lt;")
	output = strings.ReplaceAll(output, ">", "&gt;")
	return output
}

// isGitRepository checks if the given path is a git repository
func isGitRepository(path string) bool {
	_, err := executeGitCommand(path, "rev-parse", "--git-dir")
	return err == nil
}

// getGitVersion returns the git version string
func getGitVersion() (string, error) {
	output, err := exec.Command("git", "--version").Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get git version").
			WithComponent("tools.git").
			WithOperation("get_version")
	}
	return strings.TrimSpace(string(output)), nil
}
