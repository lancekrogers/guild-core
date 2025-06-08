package config

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLoadSaveGuildConfig(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "guild-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			t.Logf("Failed to cleanup temp dir: %v", rmErr)
		}
	}()

	// Create .guild directory
	guildDir := filepath.Join(tempDir, ".guild")
	if err := os.MkdirAll(guildDir, 0755); err != nil {
		t.Fatalf("Failed to create .guild dir: %v", err)
	}

	// Test config
	config := DefaultGuildTemplate()

	// Save config
	if err := SaveGuildConfig(tempDir, config); err != nil {
		t.Fatalf("SaveGuildConfig() error = %v", err)
	}

	// Load config
	loaded, err := LoadGuildConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadGuildConfig() error = %v", err)
	}

	// Verify loaded config
	if loaded.Name != config.Name {
		t.Errorf("Loaded name = %s, want %s", loaded.Name, config.Name)
	}
	if len(loaded.Agents) != len(config.Agents) {
		t.Errorf("Loaded agents = %d, want %d", len(loaded.Agents), len(config.Agents))
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
