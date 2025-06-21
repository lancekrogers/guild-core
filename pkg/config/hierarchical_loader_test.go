// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

func TestNewHierarchicalLoader(t *testing.T) {
	loader := NewHierarchicalLoader()
	if loader == nil {
		t.Fatal("NewHierarchicalLoader() returned nil")
	}
	if loader.cache == nil {
		t.Error("NewHierarchicalLoader() created loader with nil cache")
	}
}

func TestHierarchicalLoader_LoadHierarchicalConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	loader := NewHierarchicalLoader()

	tests := []struct {
		name    string
		setup   func() error
		wantErr bool
		errCode gerror.ErrorCode
		verify  func(t *testing.T, config *HierarchicalConfig)
	}{
		{
			name: "valid complete hierarchy",
			setup: func() error {
				if err := setupCompleteHierarchy(tempDir); err != nil {
					return err
				}
				return nil
			},
			wantErr: false,
			verify: func(t *testing.T, config *HierarchicalConfig) {
				if config.Campaign == nil {
					t.Error("Campaign config is nil")
				}
				if config.Guilds == nil {
					t.Error("Guilds config is nil")
				}
				if len(config.Agents) == 0 {
					t.Error("No agents loaded")
				}
			},
		},
		{
			name: "missing campaign config",
			setup: func() error {
				// Only create guild config, no campaign
				guildDir := filepath.Join(tempDir, ".campaign")
				os.MkdirAll(guildDir, 0755)
				return saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"test-guild": {
							Purpose:     "Test",
							Description: "Test",
							Agents:      []string{"test-agent"},
						},
					},
				})
			},
			wantErr: true,
			errCode: gerror.ErrCodeInternal,
		},
		{
			name: "missing guild config",
			setup: func() error {
				// Only create campaign config, no guild
				guildDir := filepath.Join(tempDir, ".campaign")
				os.MkdirAll(guildDir, 0755)
				return saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yml"), &CampaignConfig{
					Name:        "test",
					Description: "test",
				})
			},
			wantErr: true,
			errCode: gerror.ErrCodeInternal,
		},
		{
			name: "missing agent files",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				os.MkdirAll(guildDir, 0755)
				
				// Create campaign
				saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yml"), &CampaignConfig{
					Name:        "test",
					Description: "test",
				})
				
				// Create guild referencing non-existent agent
				return saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"test-guild": {
							Purpose:     "Test",
							Description: "Test",
							Agents:      []string{"missing-agent"},
						},
					},
				})
			},
			wantErr: true,
			errCode: gerror.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			loader.cache = make(map[string]*HierarchicalConfig) // Clear cache
			
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			config, err := loader.LoadHierarchicalConfig(ctx, tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadHierarchicalConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("LoadHierarchicalConfig() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if tt.verify != nil && config != nil {
				tt.verify(t, config)
			}
		})
	}
}

func TestHierarchicalLoader_LoadHierarchicalConfig_Caching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	loader := NewHierarchicalLoader()

	// Setup complete hierarchy
	if err := setupCompleteHierarchy(tempDir); err != nil {
		t.Fatalf("Failed to setup hierarchy: %v", err)
	}

	// First load
	config1, err := loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Second load should use cache
	config2, err := loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	// Should be the same instance (cached)
	if config1 != config2 {
		t.Error("Second load did not return cached instance")
	}

	// Verify cache contains the entry
	loader.mu.RLock()
	cached, exists := loader.cache[tempDir]
	loader.mu.RUnlock()
	
	if !exists {
		t.Error("Cache does not contain loaded config")
	}
	if cached != config1 {
		t.Error("Cached config is not the same as loaded config")
	}
}

func TestHierarchicalLoader_LoadHierarchicalConfig_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewHierarchicalLoader()
	setupCompleteHierarchy(tempDir)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = loader.LoadHierarchicalConfig(ctx, tempDir)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestHierarchicalLoader_RefreshConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-refresh-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	loader := NewHierarchicalLoader()

	// Setup initial hierarchy
	if err := setupCompleteHierarchy(tempDir); err != nil {
		t.Fatalf("Failed to setup hierarchy: %v", err)
	}

	// Initial load
	_, err = loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Initial load failed: %v", err)
	}

	// Modify campaign config
	campaignPath := filepath.Join(tempDir, ".campaign", "campaign.yml")
	updatedCampaign := &CampaignConfig{
		Name:              "updated-campaign",
		Description:       "Updated description",
		LastSelectedGuild: "new-guild",
	}
	saveCampaignConfigYAML(campaignPath, updatedCampaign)

	// Refresh config
	if err := loader.RefreshConfig(ctx, tempDir); err != nil {
		t.Errorf("RefreshConfig() error = %v", err)
	}

	// Load again - should get updated config
	config2, err := loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Load after refresh failed: %v", err)
	}

	// Verify update
	if config2.Campaign.Name != "updated-campaign" {
		t.Errorf("Campaign name not updated: got %v, want %v", config2.Campaign.Name, "updated-campaign")
	}
}

func TestHierarchicalLoader_RefreshConfig_ValidationFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-refresh-fail-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	loader := NewHierarchicalLoader()

	// Setup initial valid hierarchy
	if err := setupCompleteHierarchy(tempDir); err != nil {
		t.Fatalf("Failed to setup hierarchy: %v", err)
	}

	// Initial load
	config1, err := loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Initial load failed: %v", err)
	}
	originalCampaignName := config1.Campaign.Name

	// Create invalid campaign config
	campaignPath := filepath.Join(tempDir, ".campaign", "campaign.yml")
	invalidCampaign := `name: ""  # Invalid - empty name
description: "test"`
	os.WriteFile(campaignPath, []byte(invalidCampaign), 0644)

	// Refresh should fail
	err = loader.RefreshConfig(ctx, tempDir)
	if err == nil {
		t.Error("Expected error when refreshing with invalid config")
	}
	if gerror.GetCode(err) != gerror.ErrCodeValidation {
		t.Errorf("Expected validation error, got %v", gerror.GetCode(err))
	}

	// Original config should still be cached
	config2, err := loader.LoadHierarchicalConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Load after failed refresh failed: %v", err)
	}
	if config2.Campaign.Name != originalCampaignName {
		t.Error("Cache was corrupted after failed refresh")
	}
}

func TestHierarchicalLoader_RefreshConfig_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-refresh-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewHierarchicalLoader()
	setupCompleteHierarchy(tempDir)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = loader.RefreshConfig(ctx, tempDir)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestHierarchicalLoader_ValidateCrossReferences(t *testing.T) {
	loader := NewHierarchicalLoader()

	tests := []struct {
		name     string
		campaign *CampaignConfig
		guilds   *GuildConfigFile
		agents   map[string]*AgentConfig
		wantErr  bool
		errCode  gerror.ErrorCode
	}{
		{
			name: "valid cross references",
			campaign: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"backend-guild": {"commission1"},
				},
			},
			guilds: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild": {
						Agents: []string{"agent1"},
					},
				},
			},
			agents: map[string]*AgentConfig{
				"agent1": {Name: "agent1"},
			},
			wantErr: false,
		},
		{
			name: "commission mapping references non-existent guild",
			campaign: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"missing-guild": {"commission1"},
				},
			},
			guilds: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild": {
						Agents: []string{"agent1"},
					},
				},
			},
			agents: map[string]*AgentConfig{
				"agent1": {Name: "agent1"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "guild references non-existent agent",
			campaign: &CampaignConfig{},
			guilds: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild": {
						Agents: []string{"missing-agent"},
					},
				},
			},
			agents: map[string]*AgentConfig{
				"agent1": {Name: "agent1"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name:     "nil commission mappings (valid)",
			campaign: &CampaignConfig{},
			guilds: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild": {
						Agents: []string{"agent1"},
					},
				},
			},
			agents: map[string]*AgentConfig{
				"agent1": {Name: "agent1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validateCrossReferences(tt.campaign, tt.guilds, tt.agents)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCrossReferences() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("validateCrossReferences() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
		})
	}
}

func TestHierarchicalConfig_GetActiveGuild(t *testing.T) {
	config := &HierarchicalConfig{
		Guilds: &GuildConfigFile{
			Guilds: map[string]GuildDefinition{
				"backend-guild": {
					Purpose:     "Backend work",
					Description: "Backend development",
					Agents:      []string{"agent1"},
				},
			},
		},
	}

	tests := []struct {
		name      string
		guildName string
		wantErr   bool
		errCode   gerror.ErrorCode
	}{
		{
			name:      "existing guild",
			guildName: "backend-guild",
			wantErr:   false,
		},
		{
			name:      "non-existent guild",
			guildName: "missing-guild",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guild, err := config.GetActiveGuild(tt.guildName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetActiveGuild() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("GetActiveGuild() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if guild != nil && guild.Purpose != "Backend work" {
				t.Error("GetActiveGuild() returned wrong guild")
			}
		})
	}
}

func TestHierarchicalConfig_GetAgentByName(t *testing.T) {
	config := &HierarchicalConfig{
		Agents: map[string]*AgentConfig{
			"agent1": {
				Name:        "Agent One",
				ID:          "agent1",
				Type:        "worker",
				Provider:    "openai",
				Model:       "gpt-4",
				Capabilities: []string{"coding"},
			},
			"agent2": {
				Name:        "Agent Two",
				ID:          "agent2",
				Type:        "specialist",
				Provider:    "anthropic",
				Model:       "claude-3",
				Capabilities: []string{"analysis"},
			},
		},
	}

	tests := []struct {
		name      string
		agentName string
		wantErr   bool
		errCode   gerror.ErrorCode
		wantAgent string
	}{
		{
			name:      "existing agent",
			agentName: "agent1",
			wantErr:   false,
			wantAgent: "Agent One",
		},
		{
			name:      "another existing agent",
			agentName: "agent2",
			wantErr:   false,
			wantAgent: "Agent Two",
		},
		{
			name:      "non-existent agent",
			agentName: "missing-agent",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := config.GetAgentByName(tt.agentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgentByName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("GetAgentByName() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if agent != nil && agent.Name != tt.wantAgent {
				t.Errorf("GetAgentByName() returned wrong agent: got %v, want %v", agent.Name, tt.wantAgent)
			}
		})
	}
}

func TestHierarchicalConfig_GetGuildAgents(t *testing.T) {
	config := &HierarchicalConfig{
		Guilds: &GuildConfigFile{
			Guilds: map[string]GuildDefinition{
				"backend-guild": {
					Purpose:     "Backend",
					Description: "Backend work",
					Agents:      []string{"agent1", "agent2"},
				},
				"frontend-guild": {
					Purpose:     "Frontend",
					Description: "Frontend work",
					Agents:      []string{"agent3"},
				},
			},
		},
		Agents: map[string]*AgentConfig{
			"agent1": {ID: "agent1", Name: "Agent One"},
			"agent2": {ID: "agent2", Name: "Agent Two"},
			"agent3": {ID: "agent3", Name: "Agent Three"},
		},
	}

	tests := []struct {
		name       string
		guildName  string
		wantErr    bool
		errCode    gerror.ErrorCode
		wantAgents int
	}{
		{
			name:       "backend guild with 2 agents",
			guildName:  "backend-guild",
			wantErr:    false,
			wantAgents: 2,
		},
		{
			name:       "frontend guild with 1 agent",
			guildName:  "frontend-guild",
			wantErr:    false,
			wantAgents: 1,
		},
		{
			name:      "non-existent guild",
			guildName: "missing-guild",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents, err := config.GetGuildAgents(tt.guildName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGuildAgents() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("GetGuildAgents() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if len(agents) != tt.wantAgents {
				t.Errorf("GetGuildAgents() returned %d agents, want %d", len(agents), tt.wantAgents)
			}
		})
	}
}

func TestHierarchicalConfig_GetGuildAgents_MissingAgent(t *testing.T) {
	config := &HierarchicalConfig{
		Guilds: &GuildConfigFile{
			Guilds: map[string]GuildDefinition{
				"broken-guild": {
					Purpose:     "Broken",
					Description: "Guild with missing agent",
					Agents:      []string{"agent1", "missing-agent"},
				},
			},
		},
		Agents: map[string]*AgentConfig{
			"agent1": {ID: "agent1", Name: "Agent One"},
			// missing-agent is not in the agents map
		},
	}

	_, err := config.GetGuildAgents("broken-guild")
	if err == nil {
		t.Error("Expected error for guild with missing agent")
	}
	if gerror.GetCode(err) != gerror.ErrCodeInternal {
		t.Errorf("Expected internal error code, got %v", gerror.GetCode(err))
	}
}

func TestHierarchicalConfig_SaveAgentConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-save-agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	
	config := &HierarchicalConfig{
		Agents:      make(map[string]*AgentConfig),
		projectPath: tempDir,
	}

	// Create agents directory
	agentsDir := filepath.Join(tempDir, ".campaign", "agents")
	os.MkdirAll(agentsDir, 0755)

	newAgent := &AgentConfig{
		ID:           "new-agent",
		Name:         "New Agent",
		Type:         "worker",
		Provider:     "openai",
		Model:        "gpt-4",
		Capabilities: []string{"coding", "testing"},
	}

	// Save agent
	err = config.SaveAgentConfig(ctx, "new-agent", newAgent)
	if err != nil {
		t.Errorf("SaveAgentConfig() error = %v", err)
	}

	// Verify file was created
	agentPath := filepath.Join(agentsDir, "new-agent.yml")
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		t.Error("Agent file was not created")
	}

	// Verify agent was added to cache
	if _, exists := config.Agents["new-agent"]; !exists {
		t.Error("Agent was not added to cache")
	}

	// Test saving invalid agent
	invalidAgent := &AgentConfig{
		// Missing required fields
	}
	err = config.SaveAgentConfig(ctx, "invalid", invalidAgent)
	if err == nil {
		t.Error("Expected error when saving invalid agent")
	}
	if gerror.GetCode(err) != gerror.ErrCodeValidation {
		t.Errorf("Expected validation error, got %v", gerror.GetCode(err))
	}
}

func TestHierarchicalConfig_SaveAgentConfig_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-save-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &HierarchicalConfig{
		Agents:      make(map[string]*AgentConfig),
		projectPath: tempDir,
	}

	agent := &AgentConfig{
		ID:           "test",
		Name:         "Test",
		Type:         "worker",
		Provider:     "openai",
		Model:        "gpt-4",
		Capabilities: []string{"coding"},
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = config.SaveAgentConfig(ctx, "test", agent)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestHierarchicalConfig_ValidateAll(t *testing.T) {
	tests := []struct {
		name    string
		config  *HierarchicalConfig
		wantErr bool
	}{
		{
			name: "valid complete config",
			config: &HierarchicalConfig{
				Campaign: &CampaignConfig{
					Name:        "valid-campaign",
					Description: "Valid campaign",
				},
				Guilds: &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"valid-guild": {
							Purpose:     "Valid",
							Description: "Valid guild",
							Agents:      []string{"agent1"},
						},
					},
				},
				Agents: map[string]*AgentConfig{
					"agent1": {
						ID:           "agent1",
						Name:         "Agent One",
						Type:         "worker",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"coding"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid campaign",
			config: &HierarchicalConfig{
				Campaign: &CampaignConfig{
					Name: "", // Invalid
				},
				Guilds: &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"guild": {
							Purpose:     "Test",
							Description: "Test",
							Agents:      []string{"agent1"},
						},
					},
				},
				Agents: map[string]*AgentConfig{
					"agent1": {
						ID:           "agent1",
						Name:         "Agent",
						Type:         "worker",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"coding"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid guild",
			config: &HierarchicalConfig{
				Campaign: &CampaignConfig{
					Name:        "campaign",
					Description: "Campaign",
				},
				Guilds: &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"guild": {
							Purpose:     "", // Invalid
							Description: "Test",
							Agents:      []string{"agent1"},
						},
					},
				},
				Agents: map[string]*AgentConfig{
					"agent1": {
						ID:           "agent1",
						Name:         "Agent",
						Type:         "worker",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"coding"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid agent",
			config: &HierarchicalConfig{
				Campaign: &CampaignConfig{
					Name:        "campaign",
					Description: "Campaign",
				},
				Guilds: &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"guild": {
							Purpose:     "Test",
							Description: "Test",
							Agents:      []string{"agent1"},
						},
					},
				},
				Agents: map[string]*AgentConfig{
					"agent1": {
						ID:       "agent1",
						Name:     "Agent",
						Type:     "invalid-type", // Invalid
						Provider: "openai",
						Model:    "gpt-4",
						Capabilities: []string{"coding"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to setup a complete valid hierarchy
func setupCompleteHierarchy(tempDir string) error {
	guildDir := filepath.Join(tempDir, ".campaign")
	agentsDir := filepath.Join(guildDir, "agents")
	
	// Create directories
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	// Create campaign config
	campaign := &CampaignConfig{
		Name:        "test-campaign",
		Description: "Test campaign for hierarchical loading",
		CommissionMappings: map[string][]string{
			"backend-guild": {"commission1", "commission2"},
		},
	}
	if err := saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yml"), campaign); err != nil {
		return err
	}

	// Create guild config
	guilds := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"backend-guild": {
				Purpose:     "Backend development",
				Description: "Handle backend tasks",
				Agents:      []string{"api-dev", "db-expert"},
			},
			"frontend-guild": {
				Purpose:     "Frontend development",
				Description: "Handle frontend tasks",
				Agents:      []string{"ui-dev"},
			},
		},
	}
	if err := saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), guilds); err != nil {
		return err
	}

	// Create agent configs
	agents := []struct {
		name  string
		agent *AgentConfig
	}{
		{
			name: "api-dev",
			agent: &AgentConfig{
				ID:           "api-dev",
				Name:         "API Developer",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding", "api-design"},
			},
		},
		{
			name: "db-expert",
			agent: &AgentConfig{
				ID:           "db-expert",
				Name:         "Database Expert",
				Type:         "specialist",
				Provider:     "anthropic",
				Model:        "claude-3",
				Capabilities: []string{"database", "optimization"},
			},
		},
		{
			name: "ui-dev",
			agent: &AgentConfig{
				ID:           "ui-dev",
				Name:         "UI Developer",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-3.5",
				Capabilities: []string{"frontend", "ui-design"},
			},
		},
	}

	for _, a := range agents {
		if err := saveAgentConfigYAML(filepath.Join(agentsDir, a.name+".yml"), a.agent); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to save agent config as YAML
func saveAgentConfigYAML(path string, agent *AgentConfig) error {
	yamlContent := "id: " + agent.ID + "\n"
	yamlContent += "name: \"" + agent.Name + "\"\n"
	yamlContent += "type: " + agent.Type + "\n"
	yamlContent += "provider: " + agent.Provider + "\n"
	yamlContent += "model: " + agent.Model + "\n"
	yamlContent += "capabilities:\n"
	for _, cap := range agent.Capabilities {
		yamlContent += "  - " + cap + "\n"
	}
	return os.WriteFile(path, []byte(yamlContent), 0644)
}

// Real-world scenario tests
func TestHierarchicalLoader_RealWorldScenarios(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hierarchical-realworld-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	loader := NewHierarchicalLoader()

	t.Run("complex multi-project campaign", func(t *testing.T) {
		// Setup a complex hierarchy representing a real microservices migration
		if err := setupComplexHierarchy(tempDir); err != nil {
			t.Fatalf("Failed to setup complex hierarchy: %v", err)
		}

		// Load the hierarchy
		config, err := loader.LoadHierarchicalConfig(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to load complex hierarchy: %v", err)
		}

		// Verify all components loaded
		if len(config.Guilds.Guilds) < 3 {
			t.Errorf("Expected at least 3 guilds, got %d", len(config.Guilds.Guilds))
		}
		if len(config.Agents) < 10 {
			t.Errorf("Expected at least 10 agents, got %d", len(config.Agents))
		}

		// Test cross-guild agent queries
		backendAgents, err := config.GetGuildAgents("backend-microservices")
		if err != nil {
			t.Errorf("Failed to get backend agents: %v", err)
		}
		if len(backendAgents) < 4 {
			t.Errorf("Expected at least 4 backend agents, got %d", len(backendAgents))
		}

		// Test commission mappings
		commissions := config.Campaign.GetMappedCommissions("backend-microservices")
		if len(commissions) < 3 {
			t.Errorf("Expected at least 3 backend commissions, got %d", len(commissions))
		}
	})

	t.Run("concurrent configuration updates", func(t *testing.T) {
		// Setup initial hierarchy
		if err := setupCompleteHierarchy(tempDir); err != nil {
			t.Fatalf("Failed to setup hierarchy: %v", err)
		}

		// Initial load
		config, err := loader.LoadHierarchicalConfig(ctx, tempDir)
		if err != nil {
			t.Fatalf("Initial load failed: %v", err)
		}

		// Simulate concurrent updates
		var wg sync.WaitGroup
		errors := make(chan error, 3)

		// Update campaign
		wg.Add(1)
		go func() {
			defer wg.Done()
			config.Campaign.LastSelectedGuild = "frontend-guild"
			config.Campaign.SetProjectSetting("concurrent-test", true)
			errors <- SaveCampaignConfig(ctx, tempDir, config.Campaign)
		}()

		// Add new agent
		wg.Add(1)
		go func() {
			defer wg.Done()
			newAgent := &AgentConfig{
				ID:           "concurrent-agent",
				Name:         "Concurrent Agent",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"testing"},
			}
			errors <- config.SaveAgentConfig(ctx, "concurrent-agent", newAgent)
		}()

		// Update guild
		wg.Add(1)
		go func() {
			defer wg.Done()
			config.Guilds.AddGuild("concurrent-guild", GuildDefinition{
				Purpose:     "Concurrent testing",
				Description: "Guild added concurrently",
				Agents:      []string{"concurrent-agent"},
			})
			errors <- SaveGuildConfigFile(ctx, tempDir, config.Guilds)
		}()

		// Wait for all updates
		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent update error: %v", err)
			}
		}

		// Refresh and verify
		if err := loader.RefreshConfig(ctx, tempDir); err != nil {
			t.Errorf("Failed to refresh after concurrent updates: %v", err)
		}

		// Load and verify updates
		updated, err := loader.LoadHierarchicalConfig(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to load after concurrent updates: %v", err)
		}

		if updated.Campaign.LastSelectedGuild != "frontend-guild" {
			t.Error("Campaign update was not persisted")
		}
		if _, exists := updated.Agents["concurrent-agent"]; !exists {
			t.Error("New agent was not added")
		}
		if _, err := updated.Guilds.GetGuild("concurrent-guild"); err != nil {
			t.Error("New guild was not added")
		}
	})
}

// Helper to setup a complex hierarchy for real-world testing
func setupComplexHierarchy(tempDir string) error {
	guildDir := filepath.Join(tempDir, ".campaign")
	agentsDir := filepath.Join(guildDir, "agents")
	
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	// Complex campaign with multiple guilds and mappings
	campaign := &CampaignConfig{
		Name:        "microservices-migration",
		Description: "Migrate monolithic e-commerce platform to microservices",
		ProjectSettings: map[string]interface{}{
			"target-architecture": "microservices",
			"deployment-strategy": "blue-green",
			"monitoring-stack":    []string{"prometheus", "grafana", "jaeger"},
		},
		CommissionMappings: map[string][]string{
			"backend-microservices": {
				"user-service-implementation",
				"product-catalog-service",
				"order-management-service",
				"payment-gateway-integration",
			},
			"frontend-modernization": {
				"spa-migration",
				"component-library",
				"mobile-app-development",
			},
			"infrastructure-automation": {
				"kubernetes-setup",
				"ci-cd-pipeline",
				"monitoring-implementation",
				"security-hardening",
			},
		},
		LastSelectedGuild: "backend-microservices",
	}
	
	if err := saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yml"), campaign); err != nil {
		return err
	}

	// Multiple specialized guilds
	guilds := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"backend-microservices": {
				Purpose:     "Design and implement microservices backend",
				Description: "Expert team for distributed systems and APIs",
				Agents: []string{
					"api-architect",
					"microservice-expert",
					"database-specialist",
					"message-broker-expert",
					"cache-specialist",
				},
				Coordination: &CoordinationSettings{
					MaxParallelTasks: 5,
					ReviewRequired:   true,
					AutoHandoff:      true,
				},
			},
			"frontend-modernization": {
				Purpose:     "Modernize frontend with SPA and mobile apps",
				Description: "Team for modern UI/UX development",
				Agents: []string{
					"react-expert",
					"vue-specialist",
					"mobile-developer",
					"ux-designer",
					"accessibility-expert",
				},
				Coordination: &CoordinationSettings{
					MaxParallelTasks: 3,
					ReviewRequired:   false,
					AutoHandoff:      true,
				},
			},
			"infrastructure-automation": {
				Purpose:     "Automate cloud infrastructure and deployments",
				Description: "DevOps and infrastructure team",
				Agents: []string{
					"kubernetes-expert",
					"terraform-specialist",
					"ci-cd-engineer",
					"monitoring-expert",
					"security-specialist",
				},
				Coordination: &CoordinationSettings{
					MaxParallelTasks: 2,
					ReviewRequired:   true,
					AutoHandoff:      false,
				},
			},
		},
	}

	if err := saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), guilds); err != nil {
		return err
	}

	// Create all the agent configs
	agentConfigs := map[string]*AgentConfig{
		// Backend agents
		"api-architect": {
			ID:           "api-architect",
			Name:         "API Architecture Specialist",
			Type:         "specialist",
			Provider:     "anthropic",
			Model:        "claude-3-opus",
			Capabilities: []string{"api-design", "architecture", "documentation"},
			CostMagnitude: 8,
			ContextWindow: 200000,
		},
		"microservice-expert": {
			ID:           "microservice-expert",
			Name:         "Microservices Developer",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"coding", "microservices", "distributed-systems"},
			CostMagnitude: 5,
			ContextWindow: 32000,
		},
		"database-specialist": {
			ID:           "database-specialist",
			Name:         "Database Architecture Expert",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"database", "sql", "nosql", "optimization"},
			CostMagnitude: 5,
		},
		"message-broker-expert": {
			ID:           "message-broker-expert",
			Name:         "Message Queue Specialist",
			Type:         "specialist",
			Provider:     "anthropic",
			Model:        "claude-3-sonnet",
			Capabilities: []string{"kafka", "rabbitmq", "event-driven"},
			CostMagnitude: 3,
		},
		"cache-specialist": {
			ID:           "cache-specialist",
			Name:         "Caching Strategy Expert",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			Capabilities: []string{"redis", "memcached", "caching-strategies"},
			CostMagnitude: 2,
		},
		// Frontend agents
		"react-expert": {
			ID:           "react-expert",
			Name:         "React Development Specialist",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"react", "redux", "frontend", "typescript"},
			CostMagnitude: 5,
		},
		"vue-specialist": {
			ID:           "vue-specialist",
			Name:         "Vue.js Developer",
			Type:         "worker",
			Provider:     "anthropic",
			Model:        "claude-3-haiku",
			Capabilities: []string{"vue", "vuex", "frontend", "javascript"},
			CostMagnitude: 1,
		},
		"mobile-developer": {
			ID:           "mobile-developer",
			Name:         "Mobile App Developer",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"react-native", "flutter", "mobile"},
			CostMagnitude: 5,
		},
		"ux-designer": {
			ID:           "ux-designer",
			Name:         "UX/UI Design Specialist",
			Type:         "specialist",
			Provider:     "anthropic",
			Model:        "claude-3-sonnet",
			Capabilities: []string{"ui-design", "ux", "figma", "accessibility"},
			CostMagnitude: 3,
		},
		"accessibility-expert": {
			ID:           "accessibility-expert",
			Name:         "Accessibility Specialist",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			Capabilities: []string{"wcag", "aria", "accessibility-testing"},
			CostMagnitude: 2,
		},
		// Infrastructure agents
		"kubernetes-expert": {
			ID:           "kubernetes-expert",
			Name:         "Kubernetes Platform Engineer",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"kubernetes", "helm", "container-orchestration"},
			CostMagnitude: 5,
		},
		"terraform-specialist": {
			ID:           "terraform-specialist",
			Name:         "Infrastructure as Code Expert",
			Type:         "worker",
			Provider:     "anthropic",
			Model:        "claude-3-sonnet",
			Capabilities: []string{"terraform", "cloudformation", "iac"},
			CostMagnitude: 3,
		},
		"ci-cd-engineer": {
			ID:           "ci-cd-engineer",
			Name:         "CI/CD Pipeline Engineer",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			Capabilities: []string{"jenkins", "github-actions", "gitlab-ci"},
			CostMagnitude: 2,
		},
		"monitoring-expert": {
			ID:           "monitoring-expert",
			Name:         "Observability Engineer",
			Type:         "specialist",
			Provider:     "openai",
			Model:        "gpt-4",
			Capabilities: []string{"prometheus", "grafana", "elk-stack", "tracing"},
			CostMagnitude: 5,
		},
		"security-specialist": {
			ID:           "security-specialist",
			Name:         "Security Engineer",
			Type:         "specialist",
			Provider:     "anthropic",
			Model:        "claude-3-opus",
			Capabilities: []string{"security", "penetration-testing", "compliance"},
			CostMagnitude: 8,
		},
	}

	// Save all agents
	for name, agent := range agentConfigs {
		if err := saveAgentConfigYAML(filepath.Join(agentsDir, name+".yml"), agent); err != nil {
			return err
		}
	}

	return nil
}

// Benchmark tests
func BenchmarkHierarchicalLoader_LoadHierarchicalConfig(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "hierarchical-bench")
	defer os.RemoveAll(tempDir)
	
	setupComplexHierarchy(tempDir)
	loader := NewHierarchicalLoader()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear cache to force reload
		loader.mu.Lock()
		delete(loader.cache, tempDir)
		loader.mu.Unlock()
		
		_, _ = loader.LoadHierarchicalConfig(ctx, tempDir)
	}
}

func BenchmarkHierarchicalLoader_LoadHierarchicalConfig_Cached(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "hierarchical-bench-cached")
	defer os.RemoveAll(tempDir)
	
	setupComplexHierarchy(tempDir)
	loader := NewHierarchicalLoader()
	ctx := context.Background()
	
	// Prime the cache
	loader.LoadHierarchicalConfig(ctx, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loader.LoadHierarchicalConfig(ctx, tempDir)
	}
}

func BenchmarkHierarchicalConfig_GetGuildAgents(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "hierarchical-bench-agents")
	defer os.RemoveAll(tempDir)
	
	setupComplexHierarchy(tempDir)
	loader := NewHierarchicalLoader()
	ctx := context.Background()
	
	config, _ := loader.LoadHierarchicalConfig(ctx, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.GetGuildAgents("backend-microservices")
	}
}