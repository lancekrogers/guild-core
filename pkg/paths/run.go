// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package paths provides path management for Guild runtime directories and socket files
package paths

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// GuildRunDir returns the directory for Guild runtime files (sockets, PIDs)
// Creates the directory with proper permissions if it doesn't exist
func GuildRunDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get user home directory").
			WithComponent("paths").
			WithOperation("GuildRunDir")
	}

	runDir := filepath.Join(homeDir, ".guild", "run")

	// Ensure directory exists with proper permissions (user-only access)
	if err := os.MkdirAll(runDir, 0700); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create run directory").
			WithComponent("paths").
			WithOperation("GuildRunDir").
			WithDetails("directory", runDir)
	}

	// Verify permissions for security
	if err := os.Chmod(runDir, 0700); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to set run directory permissions").
			WithComponent("paths").
			WithOperation("GuildRunDir").
			WithDetails("directory", runDir)
	}

	return runDir, nil
}

// GetCampaignSocket returns the Unix socket path for a campaign
// Handles long campaign names by hashing to avoid socket path length limits (108 bytes on Unix)
func GetCampaignSocket(campaign string, session int) (string, error) {
	runDir, err := GuildRunDir()
	if err != nil {
		return "", err
	}

	// Hash campaign names for consistent socket naming and path length safety
	socketName := GetCampaignHash(campaign)

	// Add session suffix if not primary session
	if session > 0 {
		socketName = fmt.Sprintf("%s-%d", socketName, session)
	}

	socketPath := filepath.Join(runDir, socketName+".sock")
	
	// Verify the path isn't too long (Unix socket limit is 108 bytes)
	if len(socketPath) > 100 { // Leave some safety margin
		return "", gerror.New(gerror.ErrCodeInvalidInput, "socket path too long", nil).
			WithComponent("paths").
			WithOperation("GetCampaignSocket").
			WithDetails("path", socketPath).
			WithDetails("length", len(socketPath))
	}

	return socketPath, nil
}

// GetCampaignHash returns a consistent hash for campaign names
// Uses SHA1 truncated to 12 characters for uniqueness while keeping paths short
func GetCampaignHash(campaign string) string {
	hasher := sha1.New()
	hasher.Write([]byte(campaign))
	return hex.EncodeToString(hasher.Sum(nil))[:12] // 12 chars provides good uniqueness
}

// GetGuildConfigDir returns the global Guild configuration directory
func GetGuildConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get user home directory").
			WithComponent("paths").
			WithOperation("GetGuildConfigDir")
	}

	configDir := filepath.Join(homeDir, ".guild")
	
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create config directory").
			WithComponent("paths").
			WithOperation("GetGuildConfigDir").
			WithDetails("directory", configDir)
	}

	return configDir, nil
}

// GetCampaignDir returns the global campaign storage directory
func GetCampaignDir(campaignName string) (string, error) {
	configDir, err := GetGuildConfigDir()
	if err != nil {
		return "", err
	}

	campaignDir := filepath.Join(configDir, "campaigns", campaignName)
	return campaignDir, nil
}

// EnsureCampaignDir creates the campaign directory structure if it doesn't exist
func EnsureCampaignDir(campaignName string) (string, error) {
	campaignDir, err := GetCampaignDir(campaignName)
	if err != nil {
		return "", err
	}

	// Create campaign directory with subdirectories
	dirs := []string{
		campaignDir,
		filepath.Join(campaignDir, "objectives"),
		filepath.Join(campaignDir, "archives"),
		filepath.Join(campaignDir, "kanban"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
				WithComponent("paths").
				WithOperation("EnsureCampaignDir").
				WithDetails("directory", dir)
		}
	}

	return campaignDir, nil
}