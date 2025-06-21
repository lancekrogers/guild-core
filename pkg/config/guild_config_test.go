// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

func TestGuildConfigFile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *GuildConfigFile
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "valid configuration",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild": {
						Purpose:     "Handle backend development tasks",
						Description: "Guild specialized in API and database work",
						Agents:      []string{"api-developer", "database-expert"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty guilds map",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name:    "nil guilds map",
			config:  &GuildConfigFile{},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "guild with invalid definition",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"invalid-guild": {
						Purpose:     "", // Missing purpose
						Description: "Description without purpose",
						Agents:      []string{"agent1"},
					},
				},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "guild with coordination settings",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"coordinated-guild": {
						Purpose:     "Coordinated development",
						Description: "Guild with coordination settings",
						Agents:      []string{"agent1", "agent2", "agent3"},
						Coordination: &CoordinationSettings{
							MaxParallelTasks: 3,
							ReviewRequired:   true,
							AutoHandoff:      false,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("Validate() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
		})
	}
}

func TestGuildDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		guild   GuildDefinition
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "valid guild",
			guild: GuildDefinition{
				Purpose:     "Backend development",
				Description: "Handle API and database tasks",
				Agents:      []string{"agent1", "agent2"},
			},
			wantErr: false,
		},
		{
			name: "missing purpose",
			guild: GuildDefinition{
				Description: "Description without purpose",
				Agents:      []string{"agent1"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "missing description",
			guild: GuildDefinition{
				Purpose: "Purpose without description",
				Agents:  []string{"agent1"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "no agents",
			guild: GuildDefinition{
				Purpose:     "Empty guild",
				Description: "Guild with no agents",
				Agents:      []string{},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "nil agents",
			guild: GuildDefinition{
				Purpose:     "Nil agents guild",
				Description: "Guild with nil agents slice",
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "negative max parallel tasks",
			guild: GuildDefinition{
				Purpose:     "Invalid coordination",
				Description: "Guild with invalid coordination settings",
				Agents:      []string{"agent1"},
				Coordination: &CoordinationSettings{
					MaxParallelTasks: -1,
				},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "valid coordination settings",
			guild: GuildDefinition{
				Purpose:     "Valid coordination",
				Description: "Guild with valid coordination settings",
				Agents:      []string{"agent1", "agent2"},
				Coordination: &CoordinationSettings{
					MaxParallelTasks: 0, // 0 is valid (means unlimited)
					ReviewRequired:   true,
					AutoHandoff:      true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.guild.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("Validate() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
		})
	}
}

func TestLoadGuildConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func() error
		wantErr    bool
		errCode    gerror.ErrorCode
		wantGuilds int
	}{
		{
			name: "valid guild config",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				config := &GuildConfigFile{
					Guilds: map[string]GuildDefinition{
						"backend-guild": {
							Purpose:     "Backend development",
							Description: "API and database work",
							Agents:      []string{"api-dev", "db-expert"},
						},
						"frontend-guild": {
							Purpose:     "Frontend development",
							Description: "UI and UX work",
							Agents:      []string{"ui-dev", "ux-designer"},
						},
					},
				}
				return saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), config)
			},
			wantErr:    false,
			wantGuilds: 2,
		},
		{
			name: "missing guild config",
			setup: func() error {
				return os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)
			},
			wantErr: true,
			errCode: gerror.ErrCodeNotFound,
		},
		{
			name: "invalid yaml",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(guildDir, "guild.yml"), []byte("invalid: yaml: : content"), 0644)
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "invalid guild data",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				// Guild with no agents
				yamlContent := `guilds:
  invalid-guild:
    purpose: "Test"
    description: "Test"
    agents: []`
				return os.WriteFile(filepath.Join(guildDir, "guild.yml"), []byte(yamlContent), 0644)
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			config, err := LoadGuildConfigFile(ctx, tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadGuildConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("LoadGuildConfigFile() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if config != nil && len(config.Guilds) != tt.wantGuilds {
				t.Errorf("LoadGuildConfigFile() loaded %d guilds, want %d", len(config.Guilds), tt.wantGuilds)
			}
		})
	}
}

func TestLoadGuildConfigFile_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid config
	guildDir := filepath.Join(tempDir, ".campaign")
	os.MkdirAll(guildDir, 0755)
	config := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"test-guild": {
				Purpose:     "Test",
				Description: "Test",
				Agents:      []string{"agent1"},
			},
		},
	}
	saveGuildConfigYAML(filepath.Join(guildDir, "guild.yml"), config)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = LoadGuildConfigFile(ctx, tempDir)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestSaveGuildConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	tests := []struct {
		name    string
		config  *GuildConfigFile
		setup   func() error
		wantErr bool
	}{
		{
			name: "save valid config",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"save-test-guild": {
						Purpose:     "Testing save",
						Description: "Guild for testing save functionality",
						Agents:      []string{"agent1", "agent2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "save with coordination settings",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"coordinated-guild": {
						Purpose:     "Coordinated work",
						Description: "Guild with coordination",
						Agents:      []string{"agent1", "agent2"},
						Coordination: &CoordinationSettings{
							MaxParallelTasks: 5,
							ReviewRequired:   true,
							AutoHandoff:      true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "save multiple guilds",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"guild1": {
						Purpose:     "First guild",
						Description: "First test guild",
						Agents:      []string{"agent1"},
					},
					"guild2": {
						Purpose:     "Second guild",
						Description: "Second test guild",
						Agents:      []string{"agent2"},
					},
					"guild3": {
						Purpose:     "Third guild",
						Description: "Third test guild",
						Agents:      []string{"agent3"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			err := SaveGuildConfigFile(ctx, tempDir, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveGuildConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify the file was created and can be loaded
			if !tt.wantErr {
				loaded, err := LoadGuildConfigFile(ctx, tempDir)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
				}
				if len(loaded.Guilds) != len(tt.config.Guilds) {
					t.Errorf("Loaded %d guilds, expected %d", len(loaded.Guilds), len(tt.config.Guilds))
				}
			}
		})
	}
}

func TestSaveGuildConfigFile_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-save-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"test": {
				Purpose:     "Test",
				Description: "Test",
				Agents:      []string{"agent1"},
			},
		},
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = SaveGuildConfigFile(ctx, tempDir, config)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestGuildConfigFile_GetGuild(t *testing.T) {
	config := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"backend-guild": {
				Purpose:     "Backend work",
				Description: "Handle backend tasks",
				Agents:      []string{"api-dev", "db-expert"},
			},
			"frontend-guild": {
				Purpose:     "Frontend work",
				Description: "Handle frontend tasks",
				Agents:      []string{"ui-dev"},
			},
		},
	}

	tests := []struct {
		name      string
		guildName string
		wantErr   bool
		errCode   gerror.ErrorCode
		wantGuild string
	}{
		{
			name:      "existing guild",
			guildName: "backend-guild",
			wantErr:   false,
			wantGuild: "backend-guild",
		},
		{
			name:      "non-existent guild",
			guildName: "missing-guild",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
		{
			name:      "empty guild name",
			guildName: "",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guild, err := config.GetGuild(tt.guildName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGuild() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("GetGuild() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if guild != nil && guild.Purpose != config.Guilds[tt.wantGuild].Purpose {
				t.Errorf("GetGuild() returned wrong guild")
			}
		})
	}
}

func TestGuildConfigFile_AddGuild(t *testing.T) {
	tests := []struct {
		name      string
		config    *GuildConfigFile
		guildName string
		guild     GuildDefinition
		wantErr   bool
		errCode   gerror.ErrorCode
	}{
		{
			name:      "add to empty config",
			config:    &GuildConfigFile{},
			guildName: "new-guild",
			guild: GuildDefinition{
				Purpose:     "New guild purpose",
				Description: "New guild description",
				Agents:      []string{"agent1"},
			},
			wantErr: false,
		},
		{
			name: "add to existing config",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"existing": {
						Purpose:     "Existing",
						Description: "Existing guild",
						Agents:      []string{"agent1"},
					},
				},
			},
			guildName: "new-guild",
			guild: GuildDefinition{
				Purpose:     "New guild",
				Description: "Another guild",
				Agents:      []string{"agent2"},
			},
			wantErr: false,
		},
		{
			name: "add duplicate guild",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"duplicate": {
						Purpose:     "Original",
						Description: "Original guild",
						Agents:      []string{"agent1"},
					},
				},
			},
			guildName: "duplicate",
			guild: GuildDefinition{
				Purpose:     "Duplicate",
				Description: "Duplicate guild",
				Agents:      []string{"agent2"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeAlreadyExists,
		},
		{
			name:      "add invalid guild",
			config:    &GuildConfigFile{},
			guildName: "invalid",
			guild: GuildDefinition{
				Purpose:     "", // Invalid - missing purpose
				Description: "Invalid guild",
				Agents:      []string{"agent1"},
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.AddGuild(tt.guildName, tt.guild)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddGuild() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("AddGuild() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if !tt.wantErr {
				// Verify guild was added
				if _, exists := tt.config.Guilds[tt.guildName]; !exists {
					t.Error("Guild was not added to config")
				}
			}
		})
	}
}

func TestGuildConfigFile_ListGuildNames(t *testing.T) {
	tests := []struct {
		name     string
		config   *GuildConfigFile
		expected []string
	}{
		{
			name:     "empty config",
			config:   &GuildConfigFile{},
			expected: []string{},
		},
		{
			name: "single guild",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"solo-guild": {},
				},
			},
			expected: []string{"solo-guild"},
		},
		{
			name: "multiple guilds",
			config: &GuildConfigFile{
				Guilds: map[string]GuildDefinition{
					"backend-guild":  {},
					"frontend-guild": {},
					"devops-guild":   {},
				},
			},
			expected: []string{"backend-guild", "devops-guild", "frontend-guild"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := tt.config.ListGuildNames()
			sort.Strings(names) // Sort for consistent comparison
			sort.Strings(tt.expected)
			
			if len(names) != len(tt.expected) {
				t.Errorf("ListGuildNames() returned %d names, want %d", len(names), len(tt.expected))
			}
			for i := range names {
				if i < len(tt.expected) && names[i] != tt.expected[i] {
					t.Errorf("ListGuildNames()[%d] = %v, want %v", i, names[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGuildConfigFile_HasAgent(t *testing.T) {
	config := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"backend-guild": {
				Agents: []string{"api-dev", "db-expert", "cache-specialist"},
			},
			"frontend-guild": {
				Agents: []string{"ui-dev", "ux-designer"},
			},
			"devops-guild": {
				Agents: []string{"sre", "security-expert"},
			},
		},
	}

	tests := []struct {
		name      string
		agentName string
		want      bool
	}{
		{
			name:      "agent in backend guild",
			agentName: "api-dev",
			want:      true,
		},
		{
			name:      "agent in frontend guild",
			agentName: "ux-designer",
			want:      true,
		},
		{
			name:      "agent in devops guild",
			agentName: "sre",
			want:      true,
		},
		{
			name:      "non-existent agent",
			agentName: "ml-engineer",
			want:      false,
		},
		{
			name:      "empty agent name",
			agentName: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.HasAgent(tt.agentName); got != tt.want {
				t.Errorf("HasAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGuildConfigFile_GetGuildForAgent(t *testing.T) {
	config := &GuildConfigFile{
		Guilds: map[string]GuildDefinition{
			"backend-guild": {
				Agents: []string{"api-dev", "db-expert"},
			},
			"frontend-guild": {
				Agents: []string{"ui-dev", "ux-designer"},
			},
		},
	}

	tests := []struct {
		name       string
		agentName  string
		wantGuild  string
		wantErr    bool
		errCode    gerror.ErrorCode
	}{
		{
			name:      "agent in backend guild",
			agentName: "api-dev",
			wantGuild: "backend-guild",
			wantErr:   false,
		},
		{
			name:      "agent in frontend guild",
			agentName: "ui-dev",
			wantGuild: "frontend-guild",
			wantErr:   false,
		},
		{
			name:      "non-existent agent",
			agentName: "missing-agent",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
		{
			name:      "empty agent name",
			agentName: "",
			wantErr:   true,
			errCode:   gerror.ErrCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guild, err := config.GetGuildForAgent(tt.agentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGuildForAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("GetGuildForAgent() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if guild != tt.wantGuild {
				t.Errorf("GetGuildForAgent() = %v, want %v", guild, tt.wantGuild)
			}
		})
	}
}

// Helper function to save guild config as YAML
func saveGuildConfigYAML(path string, config *GuildConfigFile) error {
	yamlContent := "guilds:\n"
	for name, guild := range config.Guilds {
		yamlContent += "  " + name + ":\n"
		yamlContent += "    purpose: \"" + guild.Purpose + "\"\n"
		yamlContent += "    description: \"" + guild.Description + "\"\n"
		yamlContent += "    agents:\n"
		for _, agent := range guild.Agents {
			yamlContent += "      - " + agent + "\n"
		}
		if guild.Coordination != nil {
			yamlContent += "    coordination:\n"
			yamlContent += "      max_parallel_tasks: " + fmt.Sprintf("%d", guild.Coordination.MaxParallelTasks) + "\n"
			if guild.Coordination.ReviewRequired {
				yamlContent += "      review_required: true\n"
			}
			if guild.Coordination.AutoHandoff {
				yamlContent += "      auto_handoff: true\n"
			}
		}
	}
	return os.WriteFile(path, []byte(yamlContent), 0644)
}

// Real-world scenario tests
func TestGuildConfigFile_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "guild-realworld-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("enterprise multi-guild setup", func(t *testing.T) {
		// Create a complex multi-guild configuration
		config := &GuildConfigFile{
			Guilds: make(map[string]GuildDefinition),
		}

		// Backend guild with microservices focus
		config.Guilds["backend-microservices"] = GuildDefinition{
			Purpose:     "Design and implement microservices architecture",
			Description: "Guild specialized in microservices, APIs, and distributed systems",
			Agents: []string{
				"api-architect",
				"microservice-developer",
				"database-specialist",
				"message-queue-expert",
			},
			Coordination: &CoordinationSettings{
				MaxParallelTasks: 4,
				ReviewRequired:   true,
				AutoHandoff:      true,
			},
		}

		// Frontend guild with modern frameworks
		config.Guilds["frontend-modern"] = GuildDefinition{
			Purpose:     "Build modern reactive frontends",
			Description: "Guild specialized in React, Vue, and mobile development",
			Agents: []string{
				"react-specialist",
				"vue-developer",
				"mobile-developer",
				"ux-designer",
			},
			Coordination: &CoordinationSettings{
				MaxParallelTasks: 3,
				ReviewRequired:   false,
				AutoHandoff:      true,
			},
		}

		// DevOps guild
		config.Guilds["devops-automation"] = GuildDefinition{
			Purpose:     "Automate infrastructure and deployment",
			Description: "Guild for CI/CD, monitoring, and infrastructure as code",
			Agents: []string{
				"k8s-specialist",
				"terraform-expert",
				"monitoring-engineer",
				"security-auditor",
			},
			Coordination: &CoordinationSettings{
				MaxParallelTasks: 2,
				ReviewRequired:   true,
				AutoHandoff:      false,
			},
		}

		// Save and reload
		if err := SaveGuildConfigFile(ctx, tempDir, config); err != nil {
			t.Fatalf("Failed to save guild config: %v", err)
		}

		loaded, err := LoadGuildConfigFile(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to load guild config: %v", err)
		}

		// Verify all guilds loaded correctly
		if len(loaded.Guilds) != 3 {
			t.Errorf("Expected 3 guilds, got %d", len(loaded.Guilds))
		}

		// Test agent lookups
		guild, err := loaded.GetGuildForAgent("k8s-specialist")
		if err != nil || guild != "devops-automation" {
			t.Errorf("Failed to find correct guild for k8s-specialist: %v", err)
		}

		// Test listing
		names := loaded.ListGuildNames()
		if len(names) != 3 {
			t.Errorf("Expected 3 guild names, got %d", len(names))
		}
	})

	t.Run("guild evolution over time", func(t *testing.T) {
		// Start with a simple guild
		config := &GuildConfigFile{
			Guilds: map[string]GuildDefinition{
				"startup-guild": {
					Purpose:     "Build MVP quickly",
					Description: "Small team for rapid prototyping",
					Agents:      []string{"fullstack-dev"},
				},
			},
		}

		// Save initial config
		if err := SaveGuildConfigFile(ctx, tempDir, config); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Simulate growth - add more specialized agents
		guild := config.Guilds["startup-guild"]
		guild.Agents = append(guild.Agents, "frontend-specialist", "backend-specialist")
		guild.Coordination = &CoordinationSettings{
			MaxParallelTasks: 2,
			ReviewRequired:   true,
		}
		config.Guilds["startup-guild"] = guild

		// Add a new specialized guild
		config.AddGuild("data-guild", GuildDefinition{
			Purpose:     "Handle data pipeline and analytics",
			Description: "Guild for data engineering and ML",
			Agents:      []string{"data-engineer", "ml-specialist"},
		})

		// Save evolved config
		if err := SaveGuildConfigFile(ctx, tempDir, config); err != nil {
			t.Fatalf("Failed to save evolved config: %v", err)
		}

		// Verify evolution
		loaded, err := LoadGuildConfigFile(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to load evolved config: %v", err)
		}

		if len(loaded.Guilds) != 2 {
			t.Errorf("Expected 2 guilds after evolution, got %d", len(loaded.Guilds))
		}

		startupGuild, _ := loaded.GetGuild("startup-guild")
		if len(startupGuild.Agents) != 3 {
			t.Errorf("Expected 3 agents in evolved startup guild, got %d", len(startupGuild.Agents))
		}
	})
}

// Benchmark tests
func BenchmarkGuildConfigFile_LoadSave(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "guild-bench")
	defer os.RemoveAll(tempDir)
	
	ctx := context.Background()
	
	// Create a moderately complex config
	config := &GuildConfigFile{
		Guilds: make(map[string]GuildDefinition),
	}
	
	for i := 0; i < 10; i++ {
		guildName := "guild" + string(rune(i))
		agents := make([]string, 5)
		for j := 0; j < 5; j++ {
			agents[j] = "agent" + string(rune(j))
		}
		config.Guilds[guildName] = GuildDefinition{
			Purpose:     "Purpose " + guildName,
			Description: "Description for " + guildName,
			Agents:      agents,
			Coordination: &CoordinationSettings{
				MaxParallelTasks: i % 5,
				ReviewRequired:   i%2 == 0,
			},
		}
	}

	// Initial save
	SaveGuildConfigFile(ctx, tempDir, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loaded, _ := LoadGuildConfigFile(ctx, tempDir)
		loaded.AddGuild("bench-guild", GuildDefinition{
			Purpose:     "Benchmark",
			Description: "Benchmark guild",
			Agents:      []string{"bench-agent"},
		})
		SaveGuildConfigFile(ctx, tempDir, loaded)
		// Clean up the added guild
		delete(loaded.Guilds, "bench-guild")
		SaveGuildConfigFile(ctx, tempDir, loaded)
	}
}

func BenchmarkGuildConfigFile_HasAgent(b *testing.B) {
	config := &GuildConfigFile{
		Guilds: make(map[string]GuildDefinition),
	}
	
	// Create many guilds with many agents
	for i := 0; i < 50; i++ {
		guildName := "guild" + string(rune(i))
		agents := make([]string, 20)
		for j := 0; j < 20; j++ {
			agents[j] = "agent-" + string(rune(i)) + "-" + string(rune(j))
		}
		config.Guilds[guildName] = GuildDefinition{
			Agents: agents,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Search for an agent in the middle
		_ = config.HasAgent("agent-25-10")
	}
}

func BenchmarkGuildConfigFile_GetGuildForAgent(b *testing.B) {
	config := &GuildConfigFile{
		Guilds: make(map[string]GuildDefinition),
	}
	
	// Create many guilds with many agents
	for i := 0; i < 50; i++ {
		guildName := "guild" + string(rune(i))
		agents := make([]string, 20)
		for j := 0; j < 20; j++ {
			agents[j] = "agent-" + string(rune(i)) + "-" + string(rune(j))
		}
		config.Guilds[guildName] = GuildDefinition{
			Agents: agents,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Search for an agent in the middle
		_, _ = config.GetGuildForAgent("agent-25-10")
	}
}