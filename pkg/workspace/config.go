// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package workspace

import (
	"time"
)

// Config holds configuration for workspace management
type Config struct {
	// BaseDir is the root directory for all workspaces
	BaseDir string

	// RepoPath is the path to the source repository
	RepoPath string

	// DefaultBranch is the default branch to create workspaces from
	DefaultBranch string

	// CleanupThreshold is the duration after which inactive workspaces are cleaned
	CleanupThreshold time.Duration

	// MaxWorkspaces is the maximum number of concurrent workspaces allowed
	MaxWorkspaces int

	// PreserveOnError indicates whether to preserve workspaces that encounter errors
	PreserveOnError bool

	// AutoCleanup enables automatic cleanup of inactive workspaces
	AutoCleanup bool

	// CleanupInterval is how often to run automatic cleanup
	CleanupInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		BaseDir:          "/var/guild/workspaces",
		DefaultBranch:    "main",
		CleanupThreshold: 2 * time.Hour,
		MaxWorkspaces:    10,
		PreserveOnError:  true,
		AutoCleanup:      true,
		CleanupInterval:  30 * time.Minute,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BaseDir == "" {
		return &WorkspaceError{
			Op:      "validate_config",
			Message: "BaseDir cannot be empty",
		}
	}

	if c.RepoPath == "" {
		return &WorkspaceError{
			Op:      "validate_config",
			Message: "RepoPath cannot be empty",
		}
	}

	if c.MaxWorkspaces < 1 {
		return &WorkspaceError{
			Op:      "validate_config",
			Message: "MaxWorkspaces must be at least 1",
		}
	}

	if c.CleanupThreshold < time.Minute {
		return &WorkspaceError{
			Op:      "validate_config",
			Message: "CleanupThreshold must be at least 1 minute",
		}
	}

	return nil
}
