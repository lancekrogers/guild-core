// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// FilesystemIsolator implements filesystem-based isolation
type FilesystemIsolator struct {
	config     SandboxConfig
	tempDirs   map[string]string
	tempDirsMu sync.RWMutex
	logger     observability.Logger
}

// NewFilesystemIsolator creates a new filesystem isolator
func NewFilesystemIsolator(ctx context.Context, config SandboxConfig) (Isolator, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("FilesystemIsolator")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FilesystemIsolator").
			WithOperation("NewFilesystemIsolator")
	}

	isolator := &FilesystemIsolator{
		config:   config,
		tempDirs: make(map[string]string),
		logger:   logger,
	}

	// Validate project root
	if err := isolator.validateProjectRoot(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid project root").
			WithComponent("FilesystemIsolator").
			WithOperation("NewFilesystemIsolator")
	}

	logger.Info("Filesystem isolator initialized", "project_root", config.ProjectRoot)
	return isolator, nil
}

// Isolate prepares a command for secure execution
func (fi *FilesystemIsolator) Isolate(ctx context.Context, cmd Command) (IsolatedCommand, error) {
	logger := fi.logger.WithOperation("Isolate")

	if err := ctx.Err(); err != nil {
		return IsolatedCommand{}, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("FilesystemIsolator").
			WithOperation("Isolate")
	}

	// Create isolated command structure
	isolated := IsolatedCommand{
		Original:       cmd,
		Namespace:      "guild-sandbox",
		Capabilities:   []string{}, // Start with no capabilities
		Environment:    make(map[string]string),
		ResourceLimits: fi.config.ResourceLimits,
	}

	// Set working directory within project bounds
	workingDir, err := fi.determineWorkingDirectory(cmd)
	if err != nil {
		return isolated, gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "invalid working directory").
			WithComponent("FilesystemIsolator").
			WithOperation("Isolate")
	}
	isolated.WorkingDir = workingDir

	// Rewrite file paths in command arguments
	rewrittenCmd, err := fi.rewriteCommandPaths(cmd)
	if err != nil {
		return isolated, gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "failed to rewrite command paths").
			WithComponent("FilesystemIsolator").
			WithOperation("Isolate")
	}
	isolated.Original = rewrittenCmd

	// Set safe environment
	isolated.Environment = fi.buildSafeEnvironment()

	logger.Debug("Command isolated successfully",
		"original_dir", cmd.Dir,
		"isolated_dir", isolated.WorkingDir,
		"command", cmd.String(),
	)

	return isolated, nil
}

// ValidatePath checks if a path is allowed for the given operation
func (fi *FilesystemIsolator) ValidatePath(path string, operation PathOperation) error {
	// Expand and clean the path
	cleanPath, err := fi.cleanPath(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid path").
			WithComponent("FilesystemIsolator").
			WithOperation("ValidatePath").
			WithDetails("path", path)
	}

	// Check forbidden paths first
	for _, forbidden := range fi.config.ForbiddenPaths {
		if fi.matchesPattern(cleanPath, forbidden) {
			return gerror.New(gerror.ErrCodeSecurityViolation, "path is forbidden", nil).
				WithComponent("FilesystemIsolator").
				WithOperation("ValidatePath").
				WithDetails("path", cleanPath).
				WithDetails("forbidden_pattern", forbidden)
		}
	}

	// Check operation-specific permissions
	switch operation {
	case PathOperationRead:
		return fi.validateReadPath(cleanPath)
	case PathOperationWrite:
		return fi.validateWritePath(cleanPath)
	case PathOperationExecute:
		return fi.validateExecutePath(cleanPath)
	case PathOperationDelete:
		return fi.validateDeletePath(cleanPath)
	default:
		return gerror.New(gerror.ErrCodeValidation, "unknown path operation", nil).
			WithComponent("FilesystemIsolator").
			WithOperation("ValidatePath").
			WithDetails("operation", operation.String())
	}
}

// GetTempDir returns the temporary directory for an agent
func (fi *FilesystemIsolator) GetTempDir(agentID string) string {
	fi.tempDirsMu.RLock()
	if tempDir, exists := fi.tempDirs[agentID]; exists {
		fi.tempDirsMu.RUnlock()
		return tempDir
	}
	fi.tempDirsMu.RUnlock()

	fi.tempDirsMu.Lock()
	defer fi.tempDirsMu.Unlock()

	// Double-check after acquiring write lock
	if tempDir, exists := fi.tempDirs[agentID]; exists {
		return tempDir
	}

	// Create new temp directory
	pattern := strings.Replace(fi.config.TempDirPattern, "{agent_id}", agentID, -1)
	tempDir, err := os.MkdirTemp("", filepath.Base(pattern))
	if err != nil {
		fi.logger.WithError(err).Warn("Failed to create temp directory for agent", "agent_id", agentID)
		return filepath.Join(os.TempDir(), fmt.Sprintf("guild-%s", agentID))
	}

	fi.tempDirs[agentID] = tempDir
	return tempDir
}

// Close releases resources used by the isolator
func (fi *FilesystemIsolator) Close() error {
	fi.tempDirsMu.Lock()
	defer fi.tempDirsMu.Unlock()

	var errors []error

	// Clean up all temp directories
	for agentID, tempDir := range fi.tempDirs {
		if err := os.RemoveAll(tempDir); err != nil {
			fi.logger.WithError(err).Warn("Failed to remove temp directory", "agent_id", agentID, "path", tempDir)
			errors = append(errors, err)
		}
	}

	fi.tempDirs = make(map[string]string)

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to clean up temp directories", errors[0]).
			WithComponent("FilesystemIsolator").
			WithOperation("Close")
	}

	return nil
}

// Helper methods

func (fi *FilesystemIsolator) validateProjectRoot() error {
	if !filepath.IsAbs(fi.config.ProjectRoot) {
		return gerror.New(gerror.ErrCodeValidation, "project root must be absolute path", nil)
	}

	info, err := os.Stat(fi.config.ProjectRoot)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "project root does not exist")
	}

	if !info.IsDir() {
		return gerror.New(gerror.ErrCodeValidation, "project root must be a directory", nil)
	}

	return nil
}

func (fi *FilesystemIsolator) determineWorkingDirectory(cmd Command) (string, error) {
	if cmd.Dir == "" {
		return fi.config.ProjectRoot, nil
	}

	// Validate the requested directory
	if err := fi.ValidatePath(cmd.Dir, PathOperationExecute); err != nil {
		return "", err
	}

	// Ensure it's within project bounds or temp directory
	cleanDir, err := fi.cleanPath(cmd.Dir)
	if err != nil {
		return "", err
	}

	if fi.isWithinProject(cleanDir) {
		return cleanDir, nil
	}

	// Default to project root if outside bounds
	return fi.config.ProjectRoot, nil
}

func (fi *FilesystemIsolator) rewriteCommandPaths(cmd Command) (Command, error) {
	rewritten := cmd
	rewritten.Args = make([]string, len(cmd.Args))

	for i, arg := range cmd.Args {
		// Check if argument looks like a file path
		if fi.looksLikePath(arg) {
			rewrittenPath, err := fi.rewritePath(arg)
			if err != nil {
				return cmd, err
			}
			rewritten.Args[i] = rewrittenPath
		} else {
			rewritten.Args[i] = arg
		}
	}

	return rewritten, nil
}

func (fi *FilesystemIsolator) rewritePath(path string) (string, error) {
	// If path is relative, make it relative to project root
	if !filepath.IsAbs(path) {
		return filepath.Join(fi.config.ProjectRoot, path), nil
	}

	// If absolute path is within project, keep it
	if fi.isWithinProject(path) {
		return path, nil
	}

	// For paths outside project, check if they're allowed for reading
	if err := fi.validateReadPath(path); err == nil {
		return path, nil
	}

	// Otherwise, reject the path
	return "", gerror.New(gerror.ErrCodeSecurityViolation, "path outside allowed boundaries", nil).
		WithDetails("path", path)
}

func (fi *FilesystemIsolator) buildSafeEnvironment() map[string]string {
	env := make(map[string]string)

	// Essential environment variables
	env["PATH"] = "/usr/local/bin:/usr/bin:/bin"
	env["HOME"] = fi.GetTempDir("default")
	env["USER"] = "guild-agent"
	env["SHELL"] = "/bin/bash"
	env["TERM"] = "xterm"
	env["PWD"] = fi.config.ProjectRoot

	// Add configured environment variables
	for key, value := range fi.config.Environment {
		env[key] = value
	}

	return env
}

func (fi *FilesystemIsolator) validateReadPath(path string) error {
	// Allow reading within project
	if fi.isWithinProject(path) {
		return nil
	}

	// Check explicit read permissions
	for _, allowedPath := range fi.config.AllowedReadPaths {
		if fi.matchesPattern(path, allowedPath) {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeSecurityViolation, "read access denied", nil).
		WithDetails("path", path)
}

func (fi *FilesystemIsolator) validateWritePath(path string) error {
	// Only allow writing within explicitly allowed write paths
	for _, allowedPath := range fi.config.AllowedWritePaths {
		if fi.matchesPattern(path, allowedPath) {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeSecurityViolation, "write access denied", nil).
		WithDetails("path", path)
}

func (fi *FilesystemIsolator) validateExecutePath(path string) error {
	// Allow execution within project and system paths
	if fi.isWithinProject(path) {
		return nil
	}

	// Allow execution from standard system paths
	systemPaths := []string{
		"/usr/bin/*",
		"/bin/*",
		"/usr/local/bin/*",
	}

	for _, sysPath := range systemPaths {
		if fi.matchesPattern(path, sysPath) {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeSecurityViolation, "execute access denied", nil).
		WithDetails("path", path)
}

func (fi *FilesystemIsolator) validateDeletePath(path string) error {
	// Only allow deletion within project and temp directories
	if fi.isWithinProject(path) {
		return nil
	}

	// Check if it's a temp directory
	fi.tempDirsMu.RLock()
	defer fi.tempDirsMu.RUnlock()

	for _, tempDir := range fi.tempDirs {
		if strings.HasPrefix(path, tempDir) {
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeSecurityViolation, "delete access denied", nil).
		WithDetails("path", path)
}

func (fi *FilesystemIsolator) isWithinProject(path string) bool {
	cleanPath, err := fi.cleanPath(path)
	if err != nil {
		return false
	}

	cleanProject, err := fi.cleanPath(fi.config.ProjectRoot)
	if err != nil {
		return false
	}

	return strings.HasPrefix(cleanPath, cleanProject)
}

func (fi *FilesystemIsolator) cleanPath(path string) (string, error) {
	// Expand home directory and environment variables
	expanded := os.ExpandEnv(path)
	if strings.HasPrefix(expanded, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		expanded = filepath.Join(homeDir, expanded[2:])
	}

	// Clean and absolute the path
	cleaned := filepath.Clean(expanded)
	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", err
		}
		cleaned = abs
	}

	return cleaned, nil
}

func (fi *FilesystemIsolator) matchesPattern(path, pattern string) bool {
	// Simple pattern matching with wildcards
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	// Check if path is under pattern directory (for "/**" patterns)
	if strings.HasSuffix(pattern, "/**") {
		dir := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, dir)
	}

	return false
}

func (fi *FilesystemIsolator) looksLikePath(arg string) bool {
	// Heuristic to determine if an argument is a file path
	return strings.Contains(arg, "/") ||
		strings.HasPrefix(arg, ".") ||
		strings.HasPrefix(arg, "~") ||
		strings.Contains(arg, ".")
}
