// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/paths"
)

// CampaignReference represents a local project's reference to a global campaign
type CampaignReference struct {
	Campaign    string `yaml:"campaign"`              // Name of the global campaign
	Project     string `yaml:"project,omitempty"`     // Local project name within campaign
	Description string `yaml:"description,omitempty"` // Local project description
}

// CampaignConfig represents the global campaign configuration
type CampaignConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Created     string            `yaml:"created"`
	Projects    []ProjectInfo     `yaml:"projects,omitempty"`
	Settings    map[string]string `yaml:"settings,omitempty"`
}

// ProjectInfo represents a project within a campaign
type ProjectInfo struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// SocketRegistry contains campaign hash and metadata for fast detection
type SocketRegistry struct {
	CampaignHash string `yaml:"campaign_hash"`
	CampaignName string `yaml:"campaign_name"`
}

// GenerateCampaignHash creates a consistent hash from campaign name
func GenerateCampaignHash(campaignName string) string {
	h := sha256.Sum256([]byte(campaignName))
	return hex.EncodeToString(h[:6]) // 12 chars, 6 bytes
}

// DetectCampaign finds the campaign for the current working directory
// Uses optimized hash-based detection with fallbacks
func DetectCampaign(cwd string, flagValue string) (string, error) {
	// 1. Explicit flag takes precedence
	if flagValue != "" {
		return flagValue, nil
	}

	// 2. Try ultra-fast binary hash detection (1μs)
	if hash := readBinaryHash(cwd); hash != "" {
		// Try to lookup campaign name via socket registry in same directory
		// This provides hash validation and campaign name
		if registry := readSocketRegistry(cwd); registry != nil {
			// Verify hash matches for data consistency
			expectedHash := GenerateCampaignHash(registry.CampaignName)
			if expectedHash == hash {
				return registry.CampaignName, nil
			}
		}
		// Hash found but no valid registry - fall through to other methods
	}

	// 3. Try socket registry only (fast)
	if registry := readSocketRegistry(cwd); registry != nil {
		return registry.CampaignName, nil
	}

	// 4. Fallback to YAML parsing (slow)
	campaignRef, err := findCampaignReference(cwd)
	if err != nil {
		// No campaign found - this is not an error in the new architecture
		// Users should run 'guild init' to create a campaign
		return "", gerror.New(gerror.ErrCodeNotFound, "no campaign found", nil).
			WithComponent("campaign").
			WithOperation("DetectCampaign").
			WithDetails("directory", cwd).
			WithDetails("suggestion", "Run 'guild init' to initialize a campaign")
	}

	return campaignRef.Campaign, nil
}

// findCampaignReference walks up the directory tree to find a campaign reference
func findCampaignReference(cwd string) (*CampaignReference, error) {
	currentDir := cwd
	for {
		campaignFile := filepath.Join(currentDir, paths.DefaultCampaignDir, "campaign.yaml")
		if fileExists(campaignFile) {
			ref, err := parseCampaignReference(campaignFile)
			if err != nil {
				return nil, err
			}
			return ref, nil
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "no campaign reference found", nil).
		WithComponent("campaign").
		WithOperation("findCampaignReference")
}

// parseCampaignReference reads and parses a campaign reference file
func parseCampaignReference(filePath string) (*CampaignReference, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read campaign reference").
			WithComponent("campaign").
			WithOperation("parseCampaignReference").
			WithDetails("file", filePath)
	}

	var ref CampaignReference
	if err := yaml.Unmarshal(data, &ref); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid campaign reference format").
			WithComponent("campaign").
			WithOperation("parseCampaignReference").
			WithDetails("file", filePath)
	}

	if ref.Campaign == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidFormat, "campaign reference missing campaign name", nil).
			WithComponent("campaign").
			WithOperation("parseCampaignReference").
			WithDetails("file", filePath)
	}

	return &ref, nil
}

// GetCampaignRoot finds the root directory of the current campaign
func GetCampaignRoot(cwd string) (string, error) {
	currentDir := cwd
	for {
		guildDir := filepath.Join(currentDir, paths.DefaultCampaignDir)
		if dirExists(guildDir) {
			return currentDir, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", gerror.New(gerror.ErrCodeNotFound, "no campaign root found", nil).
				WithComponent("campaign").
				WithOperation("GetCampaignRoot").
				WithDetails("directory", cwd)
		}
		currentDir = parent
	}
}

// CreateCampaignReference creates a local campaign reference file
func CreateCampaignReference(projectDir string, campaignName string, projectName string) error {
	localGuildDir := filepath.Join(projectDir, paths.DefaultCampaignDir)
	if err := os.MkdirAll(localGuildDir, 0o700); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("campaign").
			WithOperation("CreateCampaignReference").
			WithDetails("directory", localGuildDir)
	}

	ref := CampaignReference{
		Campaign:    campaignName,
		Project:     projectName,
		Description: fmt.Sprintf("Project %s in campaign %s", projectName, campaignName),
	}

	refPath := filepath.Join(localGuildDir, "guild.yaml")
	refData, err := yaml.Marshal(ref)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign reference").
			WithComponent("campaign").
			WithOperation("CreateCampaignReference")
	}

	if err := os.WriteFile(refPath, refData, 0o600); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign reference").
			WithComponent("campaign").
			WithOperation("CreateCampaignReference").
			WithDetails("file", refPath)
	}

	return nil
}

// LoadGlobalCampaignConfig loads the global campaign configuration
func LoadGlobalCampaignConfig(campaignName string) (*CampaignConfig, error) {
	campaignDir, err := paths.GetCampaignDir(campaignName)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(campaignDir, "config.yaml")
	if !fileExists(configPath) {
		return nil, gerror.New(gerror.ErrCodeNotFound, "campaign config not found", nil).
			WithComponent("campaign").
			WithOperation("LoadGlobalCampaignConfig").
			WithDetails("campaign", campaignName).
			WithDetails("file", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read campaign config").
			WithComponent("campaign").
			WithOperation("LoadGlobalCampaignConfig").
			WithDetails("file", configPath)
	}

	var config CampaignConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "invalid campaign config format").
			WithComponent("campaign").
			WithOperation("LoadGlobalCampaignConfig").
			WithDetails("file", configPath)
	}

	return &config, nil
}

// SaveGlobalCampaignConfig saves the global campaign configuration
func SaveGlobalCampaignConfig(campaignName string, config *CampaignConfig) error {
	campaignDir, err := paths.EnsureCampaignDir(campaignName)
	if err != nil {
		return err
	}

	configPath := filepath.Join(campaignDir, "config.yaml")
	configData, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign config").
			WithComponent("campaign").
			WithOperation("SaveGlobalCampaignConfig")
	}

	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign config").
			WithComponent("campaign").
			WithOperation("SaveGlobalCampaignConfig").
			WithDetails("file", configPath)
	}

	return nil
}

// ListCampaigns returns all available campaigns from global storage
func ListCampaigns() ([]string, error) {
	configDir, err := paths.GetGuildConfigDir()
	if err != nil {
		return nil, err
	}

	campaignsDir := filepath.Join(configDir, "campaigns")
	if !dirExists(campaignsDir) {
		return []string{}, nil // No campaigns yet
	}

	entries, err := os.ReadDir(campaignsDir)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read campaigns directory").
			WithComponent("campaign").
			WithOperation("ListCampaigns").
			WithDetails("directory", campaignsDir)
	}

	var campaigns []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Validate that it's a real campaign
			if ValidateCampaign(entry.Name()) == nil {
				campaigns = append(campaigns, entry.Name())
			}
		}
	}

	return campaigns, nil
}

// ValidateCampaign checks if a campaign exists and has required structure
func ValidateCampaign(campaignName string) error {
	campaignDir, err := paths.GetCampaignDir(campaignName)
	if err != nil {
		return err
	}

	if !dirExists(campaignDir) {
		return gerror.New(gerror.ErrCodeNotFound, "campaign directory does not exist", nil).
			WithComponent("campaign").
			WithOperation("ValidateCampaign").
			WithDetails("campaign", campaignName).
			WithDetails("directory", campaignDir)
	}

	// Check if config file exists
	configPath := filepath.Join(campaignDir, "config.yaml")
	if !fileExists(configPath) {
		return gerror.New(gerror.ErrCodeNotFound, "campaign missing config file", nil).
			WithComponent("campaign").
			WithOperation("ValidateCampaign").
			WithDetails("campaign", campaignName).
			WithDetails("file", configPath)
	}

	return nil
}

// readBinaryHash tries to read campaign hash from .hash file (ultra-fast)
func readBinaryHash(cwd string) string {
	currentDir := cwd
	for {
		hashFile := filepath.Join(currentDir, paths.DefaultCampaignDir, paths.CampaignHashFile)
		if fileExists(hashFile) {
			data, err := os.ReadFile(hashFile)
			if err == nil && len(data) == 6 {
				hash := hex.EncodeToString(data)
				// TODO: Map hash back to campaign name via ~/.guild/campaigns/<hash>/config.yaml
				return hash // For now return hash, implement mapping later
			}
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}
	return ""
}

// readSocketRegistry tries to read campaign info from socket-registry.yaml (fast)
func readSocketRegistry(cwd string) *SocketRegistry {
	currentDir := cwd
	for {
		registryFile := filepath.Join(currentDir, paths.DefaultCampaignDir, paths.SocketRegistryFile)
		if fileExists(registryFile) {
			data, err := os.ReadFile(registryFile)
			if err == nil {
				var registry SocketRegistry
				if err := yaml.Unmarshal(data, &registry); err == nil {
					return &registry
				}
			}
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}
	return nil
}

// WriteCampaignHash writes the binary hash file for ultra-fast detection
func WriteCampaignHash(projectPath, campaignName string) error {
	hash := GenerateCampaignHash(campaignName)
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decode campaign hash").
			WithComponent("campaign").
			WithOperation("WriteCampaignHash")
	}

	hashFile := filepath.Join(projectPath, paths.DefaultCampaignDir, paths.CampaignHashFile)
	if err := os.WriteFile(hashFile, hashBytes, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign hash").
			WithComponent("campaign").
			WithOperation("WriteCampaignHash").
			WithDetails("file", hashFile)
	}

	return nil
}

// WriteSocketRegistry writes the socket registry for fast detection
func WriteSocketRegistry(projectPath, campaignName string) error {
	registry := SocketRegistry{
		CampaignHash: GenerateCampaignHash(campaignName),
		CampaignName: campaignName,
	}

	registryData, err := yaml.Marshal(registry)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal socket registry").
			WithComponent("campaign").
			WithOperation("WriteSocketRegistry")
	}

	registryFile := filepath.Join(projectPath, paths.DefaultCampaignDir, paths.SocketRegistryFile)
	if err := os.WriteFile(registryFile, registryData, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write socket registry").
			WithComponent("campaign").
			WithOperation("WriteSocketRegistry").
			WithDetails("file", registryFile)
	}

	return nil
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
