// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package daemon provides daemon management and socket utilities for multi-instance Guild support
package daemon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

// SocketRegistry stores the mapping between campaign names and their socket hashes
type SocketRegistry struct {
	CampaignHash string `yaml:"campaign_hash"`
	CampaignName string `yaml:"campaign_name"`
}

// SessionInfo represents information about a running daemon session
type SessionInfo struct {
	Campaign string `json:"campaign"`
	Session  int    `json:"session"`
	Socket   string `json:"socket"`
	Status   string `json:"status"`
}

// SaveSocketRegistry creates a local socket registry file for efficient socket discovery
func SaveSocketRegistry(projectRoot string, campaign string) error {
	guildDir := filepath.Join(projectRoot, ".guild")
	registryPath := filepath.Join(guildDir, "socket-registry.yaml")

	registry := SocketRegistry{
		CampaignHash: paths.GetCampaignHash(campaign),
		CampaignName: campaign,
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal socket registry").
			WithComponent("daemon").
			WithOperation("SaveSocketRegistry")
	}

	return os.WriteFile(registryPath, data, 0600)
}

// LoadSocketRegistry loads the socket registry from a project directory
func LoadSocketRegistry(projectRoot string) (*SocketRegistry, error) {
	registryPath := filepath.Join(projectRoot, ".guild", "socket-registry.yaml")

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read socket registry").
			WithComponent("daemon").
			WithOperation("LoadSocketRegistry").
			WithDetails("file", registryPath)
	}

	var registry SocketRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid socket registry format").
			WithComponent("daemon").
			WithOperation("LoadSocketRegistry").
			WithDetails("file", registryPath)
	}

	return &registry, nil
}

// FindAvailableSession finds the next available session for a campaign
func FindAvailableSession(campaign string) (int, string, error) {
	// Try primary session first (session 0)
	socketPath, err := paths.GetCampaignSocket(campaign, 0)
	if err != nil {
		return 0, "", err
	}

	if !CanConnect(socketPath) {
		return 0, socketPath, nil
	}

	// Find next available session (limit to 10 sessions per campaign)
	for session := 1; session < 10; session++ {
		socketPath, err := paths.GetCampaignSocket(campaign, session)
		if err != nil {
			return 0, "", err
		}

		if !CanConnect(socketPath) {
			return session, socketPath, nil
		}
	}

	return 0, "", gerror.New(gerror.ErrCodeResourceExhausted, "maximum sessions reached").
		WithComponent("daemon").
		WithOperation("FindAvailableSession").
		WithDetails("campaign", campaign).
		WithDetails("max_sessions", 10)
}

// ListCampaignSessions lists all running sessions for a campaign
func ListCampaignSessions(campaign string) ([]SessionInfo, error) {
	var sessions []SessionInfo

	// Check primary session and additional sessions (0-9)
	for session := 0; session < 10; session++ {
		socketPath, err := paths.GetCampaignSocket(campaign, session)
		if err != nil {
			continue // Skip if socket path generation fails
		}

		if CanConnect(socketPath) {
			sessions = append(sessions, SessionInfo{
				Campaign: campaign,
				Session:  session,
				Socket:   socketPath,
				Status:   "running",
			})
		}
	}

	return sessions, nil
}

// FindSocketsByCampaign finds all socket files for a campaign using hash prefix
func FindSocketsByCampaign(campaign string) ([]string, error) {
	runDir, err := paths.GuildRunDir()
	if err != nil {
		return nil, err
	}

	campaignHash := paths.GetCampaignHash(campaign)
	pattern := filepath.Join(runDir, campaignHash+"*.sock")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to find campaign sockets").
			WithComponent("daemon").
			WithOperation("FindSocketsByCampaign").
			WithDetails("pattern", pattern)
	}

	return matches, nil
}

// CanConnect tests if a Unix socket is responsive
func CanConnect(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// UnlinkIfStale removes a socket file if it's not connected to a running process
func UnlinkIfStale(socketPath string) error {
	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil // No file to clean
	}

	// Try to connect
	if CanConnect(socketPath) {
		return nil // Socket is alive, don't remove
	}

	// Socket is stale, remove it
	if err := os.Remove(socketPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to remove stale socket").
			WithComponent("daemon").
			WithOperation("UnlinkIfStale").
			WithDetails("socket", socketPath)
	}

	return nil
}

// EnsureSocketClean ensures a socket path is ready for use by cleaning stale sockets
func EnsureSocketClean(socketPath string) error {
	if err := UnlinkIfStale(socketPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to clean socket path").
			WithComponent("daemon").
			WithOperation("EnsureSocketClean").
			WithDetails("socket", socketPath)
	}
	return nil
}

// CleanupStaleSessionSockets removes all stale sockets for a campaign
func CleanupStaleSessionSockets(campaign string) error {
	for session := 0; session < 10; session++ {
		socketPath, err := paths.GetCampaignSocket(campaign, session)
		if err != nil {
			continue // Skip if socket path generation fails
		}

		// Clean up any stale sockets (ignore errors for individual cleanups)
		UnlinkIfStale(socketPath)
	}
	return nil
}

// DiscoverAllRunningSessions discovers all currently running Guild daemon sessions
func DiscoverAllRunningSessions() (map[string][]SessionInfo, error) {
	runDir, err := paths.GuildRunDir()
	if err != nil {
		return nil, err
	}

	// Find all .sock files
	pattern := filepath.Join(runDir, "*.sock")
	sockFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to discover socket files").
			WithComponent("daemon").
			WithOperation("DiscoverAllRunningSessions").
			WithDetails("pattern", pattern)
	}

	sessions := make(map[string][]SessionInfo)

	for _, sockFile := range sockFiles {
		// Parse socket filename to extract campaign hash and session
		basename := filepath.Base(sockFile)
		basename = strings.TrimSuffix(basename, ".sock")

		// Extract session number if present
		session := 0
		if idx := strings.LastIndex(basename, "-"); idx != -1 {
			if sessionStr := basename[idx+1:]; sessionStr != "" {
				if parsedSession, err := strconv.Atoi(sessionStr); err == nil {
					session = parsedSession
					basename = basename[:idx] // Remove session suffix to get hash
				}
			}
		}

		campaignHash := basename

		// Test if socket is responsive
		if CanConnect(sockFile) {
			// For now, we don't have a reverse mapping from hash to campaign name
			// This would require either storing metadata or querying all known campaigns
			// For MVP, we'll use the hash as the campaign identifier
			sessionInfo := SessionInfo{
				Campaign: campaignHash, // TODO: Map hash back to campaign name
				Session:  session,
				Socket:   sockFile,
				Status:   "running",
			}

			sessions[campaignHash] = append(sessions[campaignHash], sessionInfo)
		} else {
			// Clean up stale socket
			os.Remove(sockFile)
		}
	}

	return sessions, nil
}

// GetCampaignFromSocketPath attempts to extract campaign information from socket path
// This is a best-effort function since socket paths use hashes
func GetCampaignFromSocketPath(socketPath string) (string, int, error) {
	basename := filepath.Base(socketPath)
	basename = strings.TrimSuffix(basename, ".sock")

	// Extract session number
	session := 0
	if idx := strings.LastIndex(basename, "-"); idx != -1 {
		if sessionStr := basename[idx+1:]; sessionStr != "" {
			if parsedSession, err := strconv.Atoi(sessionStr); err == nil {
				session = parsedSession
				basename = basename[:idx]
			}
		}
	}

	// basename now contains the campaign hash
	campaignHash := basename

	return campaignHash, session, nil
}

// StopSession sends a shutdown signal to a specific daemon session
func StopSession(socketPath string) error {
	if !CanConnect(socketPath) {
		return gerror.New(gerror.ErrCodeNotFound, "daemon session not running").
			WithComponent("daemon").
			WithOperation("StopSession").
			WithDetails("socket", socketPath)
	}

	// Connect to socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon").
			WithOperation("StopSession").
			WithDetails("socket", socketPath)
	}
	defer conn.Close()

	// Send shutdown command (this would need to be implemented in the daemon's socket handler)
	_, err = conn.Write([]byte("SHUTDOWN\n"))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to send shutdown command").
			WithComponent("daemon").
			WithOperation("StopSession").
			WithDetails("socket", socketPath)
	}

	return nil
}