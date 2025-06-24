// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

func TestGuildConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *GuildConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &GuildConfig{
				Name: "TestGuild",
				Manager: ManagerConfig{
					Default: "agent1",
				},
				Agents: []AgentConfig{
					{
						ID:           "agent1",
						Name:         "Test Agent",
						Type:         "manager",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"planning"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &GuildConfig{
				Agents: []AgentConfig{
					{
						ID:           "agent1",
						Name:         "Test Agent",
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
			name: "no agents",
			config: &GuildConfig{
				Name:   "TestGuild",
				Agents: []AgentConfig{},
			},
			wantErr: true,
		},
		{
			name: "invalid manager reference",
			config: &GuildConfig{
				Name: "TestGuild",
				Manager: ManagerConfig{
					Default: "nonexistent",
				},
				Agents: []AgentConfig{
					{
						ID:           "agent1",
						Name:         "Test Agent",
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
			name: "duplicate agent IDs",
			config: &GuildConfig{
				Name: "TestGuild",
				Agents: []AgentConfig{
					{
						ID:           "agent1",
						Name:         "Test Agent 1",
						Type:         "worker",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"coding"},
					},
					{
						ID:           "agent1",
						Name:         "Test Agent 2",
						Type:         "worker",
						Provider:     "anthropic",
						Model:        "claude-3",
						Capabilities: []string{"analysis"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		agent   AgentConfig
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: AgentConfig{
				ID:           "test",
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding"},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			agent: AgentConfig{
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding"},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			agent: AgentConfig{
				ID:           "test",
				Name:         "Test Agent",
				Type:         "invalid",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding"},
			},
			wantErr: true,
		},
		{
			name: "no capabilities",
			agent: AgentConfig{
				ID:           "test",
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuildConfig_GetManagerAgent(t *testing.T) {
	config := &GuildConfig{
		Name: "TestGuild",
		Manager: ManagerConfig{
			Default:  "manager1",
			Fallback: []string{"manager2"},
		},
		Agents: []AgentConfig{
			{
				ID:           "manager1",
				Name:         "Primary Manager",
				Type:         "manager",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"planning"},
			},
			{
				ID:           "manager2",
				Name:         "Backup Manager",
				Type:         "manager",
				Provider:     "anthropic",
				Model:        "claude-3",
				Capabilities: []string{"planning"},
			},
			{
				ID:           "worker1",
				Name:         "Worker",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-3.5",
				Capabilities: []string{"coding"},
			},
		},
	}

	// Test default manager
	manager, err := config.GetManagerAgent()
	if err != nil {
		t.Fatalf("GetManagerAgent() error = %v", err)
	}
	if manager.ID != "manager1" {
		t.Errorf("Expected manager1, got %s", manager.ID)
	}

	// Test with override
	config.Manager.Override = "manager2"
	manager, err = config.GetManagerAgent()
	if err != nil {
		t.Fatalf("GetManagerAgent() with override error = %v", err)
	}
	if manager.ID != "manager2" {
		t.Errorf("Expected manager2, got %s", manager.ID)
	}

	// Test fallback
	config.Manager.Default = "nonexistent"
	config.Manager.Override = ""
	manager, err = config.GetManagerAgent()
	if err != nil {
		t.Fatalf("GetManagerAgent() with fallback error = %v", err)
	}
	if manager.ID != "manager2" {
		t.Errorf("Expected manager2 from fallback, got %s", manager.ID)
	}
}

func TestGuildConfig_GetAgentsByCapability(t *testing.T) {
	config := &GuildConfig{
		Name: "TestGuild",
		Agents: []AgentConfig{
			{
				ID:           "agent1",
				Name:         "Agent 1",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding", "testing"},
			},
			{
				ID:           "agent2",
				Name:         "Agent 2",
				Type:         "worker",
				Provider:     "anthropic",
				Model:        "claude-3",
				Capabilities: []string{"analysis", "documentation"},
			},
			{
				ID:           "agent3",
				Name:         "Agent 3",
				Type:         "specialist",
				Provider:     "openai",
				Model:        "gpt-4",
				Capabilities: []string{"coding", "review"},
			},
		},
	}

	// Test finding agents by capability
	codingAgents := config.GetAgentsByCapability("coding")
	if len(codingAgents) != 2 {
		t.Errorf("Expected 2 coding agents, got %d", len(codingAgents))
	}

	analysisAgents := config.GetAgentsByCapability("analysis")
	if len(analysisAgents) != 1 {
		t.Errorf("Expected 1 analysis agent, got %d", len(analysisAgents))
	}

	// Test non-existent capability
	mlAgents := config.GetAgentsByCapability("machine_learning")
	if len(mlAgents) != 0 {
		t.Errorf("Expected 0 ML agents, got %d", len(mlAgents))
	}
}

// TestLoadGuildConfig_ModularStructure tests the new modular config loading
// that reads from campaign.yaml + guilds/*.yaml + agents/*.yaml
func TestLoadGuildConfig_ModularStructure(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "guild-modular-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .campaign directory structure
	campaignDir := filepath.Join(tempDir, ".campaign")
	guildsDir := filepath.Join(campaignDir, "guilds")
	agentsDir := filepath.Join(campaignDir, "agents")

	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		t.Fatalf("Failed to create guilds dir: %v", err)
	}
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	// Create campaign.yaml
	campaignContent := `name: test-campaign
description: Test campaign for modular config
guilds:
  - test_guild
settings:
  default_guild: test_guild
`
	if err := os.WriteFile(filepath.Join(campaignDir, "campaign.yaml"), []byte(campaignContent), 0644); err != nil {
		t.Fatalf("Failed to write campaign.yaml: %v", err)
	}

	// Create guild yaml
	guildContent := `name: test_guild
description: Test guild for unit testing
purpose: Testing modular config loading
manager: manager-agent
agents:
  - manager-agent
  - worker-agent
coordination:
  max_parallel_tasks: 3
  review_required: true
  auto_handoff: false
`
	if err := os.WriteFile(filepath.Join(guildsDir, "test_guild.yaml"), []byte(guildContent), 0644); err != nil {
		t.Fatalf("Failed to write guild yaml: %v", err)
	}

	// Create manager agent yaml
	managerContent := `id: manager-agent
name: Test Manager
type: manager
provider: anthropic
model: claude-3-sonnet
description: Test manager agent
capabilities:
  - project-management
  - team-coordination
temperature: 0.1
backstory:
  experience: "10 years managing teams"
  guild_rank: "Guild Master"
  philosophy: "Lead by example"
personality:
  assertiveness: 8
  empathy: 9
  patience: 8
`
	if err := os.WriteFile(filepath.Join(agentsDir, "manager-agent.yaml"), []byte(managerContent), 0644); err != nil {
		t.Fatalf("Failed to write manager agent yaml: %v", err)
	}

	// Create worker agent yaml
	workerContent := `id: worker-agent
name: Test Worker
type: worker
provider: openai
model: gpt-4
description: Test worker agent
capabilities:
  - coding
  - testing
temperature: 0.2
backstory:
  experience: "5 years coding"
  guild_rank: "Journeyman"
`
	if err := os.WriteFile(filepath.Join(agentsDir, "worker-agent.yaml"), []byte(workerContent), 0644); err != nil {
		t.Fatalf("Failed to write worker agent yaml: %v", err)
	}

	// Test loading the modular config
	ctx := context.Background()
	config, err := LoadGuildConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("LoadGuildConfig() error = %v", err)
	}

	// Verify loaded config
	if config.Name != "test_guild" {
		t.Errorf("Expected guild name 'test_guild', got '%s'", config.Name)
	}

	if config.Description != "Test guild for unit testing" {
		t.Errorf("Expected guild description 'Test guild for unit testing', got '%s'", config.Description)
	}

	if config.Manager.Default != "manager-agent" {
		t.Errorf("Expected manager 'manager-agent', got '%s'", config.Manager.Default)
	}

	if len(config.Agents) != 2 {
		t.Fatalf("Expected 2 agents, got %d", len(config.Agents))
	}

	// Verify manager agent
	managerFound := false
	workerFound := false
	for _, agent := range config.Agents {
		switch agent.ID {
		case "manager-agent":
			managerFound = true
			if agent.Name != "Test Manager" {
				t.Errorf("Expected manager name 'Test Manager', got '%s'", agent.Name)
			}
			if agent.Type != "manager" {
				t.Errorf("Expected manager type 'manager', got '%s'", agent.Type)
			}
			if agent.Backstory == nil {
				t.Error("Expected manager to have backstory")
			} else if agent.Backstory.Experience != "10 years managing teams" {
				t.Errorf("Expected manager experience '10 years managing teams', got '%s'", agent.Backstory.Experience)
			}
			if agent.Personality == nil {
				t.Error("Expected manager to have personality")
			} else if agent.Personality.Empathy != 9 {
				t.Errorf("Expected manager empathy 9, got %d", agent.Personality.Empathy)
			}
		case "worker-agent":
			workerFound = true
			if agent.Name != "Test Worker" {
				t.Errorf("Expected worker name 'Test Worker', got '%s'", agent.Name)
			}
			if agent.Provider != "openai" {
				t.Errorf("Expected worker provider 'openai', got '%s'", agent.Provider)
			}
		}
	}

	if !managerFound {
		t.Error("Manager agent not found in loaded config")
	}
	if !workerFound {
		t.Error("Worker agent not found in loaded config")
	}
}

// TestLoadGuildConfig_MissingCampaign tests error when campaign.yaml is missing
func TestLoadGuildConfig_MissingCampaign(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-test-missing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	_, err = LoadGuildConfig(ctx, tempDir)
	if err == nil {
		t.Fatal("Expected error when campaign.yaml is missing")
	}

	if !strings.Contains(err.Error(), "campaign not initialized") {
		t.Errorf("Expected error about campaign not initialized, got: %v", err)
	}
}

// TestLoadGuildConfig_ContextCancellation tests context cancellation handling
func TestLoadGuildConfig_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := LoadGuildConfig(ctx, ".")
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}

	gerr, ok := err.(*gerror.GuildError)
	if !ok {
		t.Fatalf("Expected GuildError, got %T", err)
	}

	if gerr.Code != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %s, got %s", gerror.ErrCodeCancelled, gerr.Code)
	}
}

func TestAgentConfig_HasCapability(t *testing.T) {
	agent := AgentConfig{
		Capabilities: []string{"coding", "testing", "debugging"},
	}

	if !agent.HasCapability("coding") {
		t.Error("Expected agent to have coding capability")
	}
	if agent.HasCapability("documentation") {
		t.Error("Expected agent to not have documentation capability")
	}
}

func TestAgentConfig_HasTool(t *testing.T) {
	agent := AgentConfig{
		Tools: []string{"file_edit", "shell_execute"},
	}

	if !agent.HasTool("file_edit") {
		t.Error("Expected agent to have file_edit tool")
	}
	if agent.HasTool("web_search") {
		t.Error("Expected agent to not have web_search tool")
	}
}
