// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"github.com/guild-framework/guild-core/pkg/paths"
	"github.com/guild-framework/guild-core/pkg/project"
)

func TestCreateEnhancedCampaignStructure(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Test creating campaign structure
	err := createEnhancedCampaignStructure(ctx, tmpDir)
	require.NoError(t, err)

	// Verify campaign directories
	campaignDirs := []string{
		"agents",
		"guilds",
		"memory",
		"prompts",
		"tools",
		"workspaces",
	}

	for _, dir := range campaignDirs {
		path := filepath.Join(tmpDir, paths.DefaultCampaignDir, dir)
		assert.DirExists(t, path, "Campaign directory %s should exist", dir)
	}

	// Verify user-facing directories
	userDirs := []string{
		"commissions",
		"commissions/refined",
		"corpus",
		"corpus/index",
		"kanban",
	}

	for _, dir := range userDirs {
		path := filepath.Join(tmpDir, dir)
		assert.DirExists(t, path, "User directory %s should exist", dir)
	}
}

func TestGenerateCampaignHash(t *testing.T) {
	// Test hash generation
	hash1 := generateCampaignHash("/path1")
	hash2 := generateCampaignHash("/path2")
	hash3 := generateCampaignHash("/path1") // Same path but different time

	// Hashes should be unique
	assert.NotEqual(t, hash1, hash2, "Different paths should generate different hashes")
	assert.NotEqual(t, hash1, hash3, "Same path at different times should generate different hashes")

	// Hash should be 16 characters (8 bytes hex encoded)
	assert.Len(t, hash1, 16, "Hash should be 16 characters")
}

func TestCreateCampaignConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create campaign directory first
	campaignDir := filepath.Join(tmpDir, paths.DefaultCampaignDir)
	err := os.MkdirAll(campaignDir, 0755)
	require.NoError(t, err)

	// Test creating campaign config
	err = createCampaignConfig(ctx, tmpDir, "test-campaign", "test-project", "go")
	require.NoError(t, err)

	// Verify campaign.yaml exists
	campaignPath := filepath.Join(campaignDir, "campaign.yaml")
	assert.FileExists(t, campaignPath)

	// Load and verify content
	data, err := os.ReadFile(campaignPath)
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify campaign section
	campaign, ok := config["campaign"].(map[string]interface{})
	require.True(t, ok, "Should have campaign section")
	assert.Equal(t, "test-campaign", campaign["name"])
	assert.Equal(t, "test-project", campaign["project_name"])
	assert.Equal(t, "go", campaign["project_type"])
	assert.Contains(t, campaign, "hash")
	assert.Contains(t, campaign, "created_at")

	// Verify daemon section
	daemon, ok := config["daemon"].(map[string]interface{})
	require.True(t, ok, "Should have daemon section")
	assert.Contains(t, daemon, "socket_path")
	assert.Equal(t, "info", daemon["log_level"])

	// Verify storage section
	storage, ok := config["storage"].(map[string]interface{})
	require.True(t, ok, "Should have storage section")
	assert.Equal(t, "memory.db", storage["database"])
	assert.Equal(t, "sqlite", storage["backend"])

	// Verify .hash file exists
	hashPath := filepath.Join(campaignDir, paths.CampaignHashFile)
	assert.FileExists(t, hashPath)
}

func TestCreateSocketRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create campaign directory
	campaignDir := filepath.Join(tmpDir, paths.DefaultCampaignDir)
	err := os.MkdirAll(campaignDir, 0755)
	require.NoError(t, err)

	// Test creating socket registry
	campaignHash := "test1234hash5678"
	err = createSocketRegistry(ctx, tmpDir, "test-campaign", campaignHash)
	require.NoError(t, err)

	// Verify socket-registry.yaml exists
	registryPath := filepath.Join(campaignDir, paths.SocketRegistryFile)
	assert.FileExists(t, registryPath)

	// Load and verify content
	data, err := os.ReadFile(registryPath)
	require.NoError(t, err)

	var registry map[string]interface{}
	err = yaml.Unmarshal(data, &registry)
	require.NoError(t, err)

	assert.Equal(t, campaignHash, registry["campaign_hash"])
	assert.Equal(t, "test-campaign", registry["campaign_name"])
	assert.Equal(t, "/tmp/guild-test1234hash5678.sock", registry["socket_path"])
	assert.Contains(t, registry, "created_at")

	// Verify daemon section
	daemon, ok := registry["daemon"].(map[string]interface{})
	require.True(t, ok, "Should have daemon section")
	assert.Equal(t, "stopped", daemon["status"])
	assert.Equal(t, 0, daemon["pid"])
}

func TestCreateDefaultGuildConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create required directories
	guildsDir := filepath.Join(tmpDir, paths.DefaultCampaignDir, "guilds")
	err := os.MkdirAll(guildsDir, 0755)
	require.NoError(t, err)

	// Test creating guild config
	err = createDefaultGuildConfig(ctx, tmpDir, "test-project")
	require.NoError(t, err)

	// Verify elena_guild.yaml exists (actual filename used by implementation)
	guildPath := filepath.Join(guildsDir, "elena_guild.yaml")
	assert.FileExists(t, guildPath)

	// Load and verify content
	data, err := os.ReadFile(guildPath)
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	// Verify guild section
	guild, ok := config["guild"].(map[string]interface{})
	require.True(t, ok, "Should have guild section")
	assert.Equal(t, "test-project", guild["name"])
	assert.Contains(t, guild, "description")
	assert.Contains(t, guild, "created_at")

	// Verify manager section
	manager, ok := config["manager"].(map[string]interface{})
	require.True(t, ok, "Should have manager section")
	assert.Equal(t, "elena-guild-master", manager["default"])

	// Verify agents list
	agents, ok := config["agents"].([]interface{})
	require.True(t, ok, "Should have agents list")
	assert.Len(t, agents, 3)
	assert.Contains(t, agents, "elena-guild-master")
	assert.Contains(t, agents, "marcus-developer")
	assert.Contains(t, agents, "vera-tester")

	// Verify cost optimization
	cost, ok := config["cost_optimization"].(map[string]interface{})
	require.True(t, ok, "Should have cost_optimization section")
	assert.Equal(t, true, cost["enabled"])
	// Note: YAML unmarshaling may return int or float64 depending on the value
	maxCost := cost["max_cost"]
	assert.True(t, maxCost == 100 || maxCost == 100.0, "max_cost should be 100 (int or float64)")
}

func TestCreateEnhancedAgentConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create agents directory
	agentsDir := filepath.Join(tmpDir, paths.DefaultCampaignDir, "agents")
	err := os.MkdirAll(agentsDir, 0755)
	require.NoError(t, err)

	// Create project type
	projectType := &project.ProjectType{
		Name:        "go",
		Language:    "go",
		Framework:   "stdlib",
		Description: "Go project",
	}

	// Test creating agent configs
	err = createEnhancedAgentConfigs(ctx, tmpDir, projectType)
	require.NoError(t, err)

	// Verify all three agent files exist
	agents := []string{
		"elena-guild-master.yaml",
		"marcus-developer.yaml",
		"vera-tester.yaml",
	}

	for _, agent := range agents {
		path := filepath.Join(agentsDir, agent)
		assert.FileExists(t, path, "Agent config %s should exist", agent)
	}

	// Test Elena config content
	elenaPath := filepath.Join(agentsDir, "elena-guild-master.yaml")
	data, err := os.ReadFile(elenaPath)
	require.NoError(t, err)

	var elenaConfig map[string]interface{}
	err = yaml.Unmarshal(data, &elenaConfig)
	require.NoError(t, err)

	assert.Equal(t, "elena-guild-master", elenaConfig["id"])
	assert.Equal(t, "Elena", elenaConfig["name"])
	assert.Equal(t, "manager", elenaConfig["type"])
	assert.Equal(t, "anthropic", elenaConfig["provider"])
	assert.Contains(t, elenaConfig, "capabilities")
	assert.Contains(t, elenaConfig, "backstory")
}

func TestDetectLanguageExpertise(t *testing.T) {
	tests := []struct {
		name     string
		projType *project.ProjectType
		expected string
	}{
		{
			name: "Go project",
			projType: &project.ProjectType{
				Language: "go",
			},
			expected: "Go, concurrency patterns, error handling, testing",
		},
		{
			name: "Python project",
			projType: &project.ProjectType{
				Language: "python",
			},
			expected: "Python, Django/Flask, async programming, data science libraries",
		},
		{
			name: "JavaScript project",
			projType: &project.ProjectType{
				Language: "javascript",
			},
			expected: "JavaScript/TypeScript, React/Vue/Angular, Node.js, modern web development",
		},
		{
			name: "Unknown project",
			projType: &project.ProjectType{
				Language: "unknown",
			},
			expected: "Multiple programming languages and frameworks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguageExpertise(tt.projType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptAgentConfigsToProjectType(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create agents directory and Marcus config
	agentsDir := filepath.Join(tmpDir, paths.DefaultCampaignDir, "agents")
	err := os.MkdirAll(agentsDir, 0755)
	require.NoError(t, err)

	// Create a basic Marcus config
	marcusConfig := map[string]interface{}{
		"id":           "marcus-developer",
		"name":         "Marcus",
		"type":         "worker",
		"provider":     "anthropic",
		"capabilities": []string{"coding"},
		"tools":        []string{"code", "edit"},
	}

	marcusPath := filepath.Join(agentsDir, "marcus-developer.yaml")
	data, err := yaml.Marshal(marcusConfig)
	require.NoError(t, err)
	err = os.WriteFile(marcusPath, data, 0644)
	require.NoError(t, err)

	// Test adaptation for Go project
	projectType := &project.ProjectType{
		Name:      "go",
		Language:  "go",
		Framework: "stdlib",
	}

	err = adaptAgentConfigsToProjectType(ctx, tmpDir, projectType)
	require.NoError(t, err)

	// Read updated config
	updatedData, err := os.ReadFile(marcusPath)
	require.NoError(t, err)

	var updatedConfig map[string]interface{}
	err = yaml.Unmarshal(updatedData, &updatedConfig)
	require.NoError(t, err)

	// Verify tools were added
	tools, ok := updatedConfig["tools"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, tools, "go_test")
	assert.Contains(t, tools, "go_build")

	// Verify capabilities were added
	capabilities, ok := updatedConfig["capabilities"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, capabilities, "goroutines")
	assert.Contains(t, capabilities, "channels")
}
