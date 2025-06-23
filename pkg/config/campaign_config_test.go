// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

func TestCampaignConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CampaignConfig
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "valid configuration",
			config: &CampaignConfig{
				Name:        "e-commerce-migration",
				Description: "Migrate legacy e-commerce platform to modern architecture",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &CampaignConfig{
				Description: "A campaign without a name",
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "missing description",
			config: &CampaignConfig{
				Name: "nameless-campaign",
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "empty name",
			config: &CampaignConfig{
				Name:        "",
				Description: "A campaign with empty name",
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "with commission mappings",
			config: &CampaignConfig{
				Name:        "multi-guild-campaign",
				Description: "Campaign with multiple guilds and commissions",
				CommissionMappings: map[string][]string{
					"backend-guild":  {"api-refactor", "database-migration"},
					"frontend-guild": {"ui-redesign"},
				},
			},
			wantErr: false,
		},
		{
			name: "with project settings",
			config: &CampaignConfig{
				Name:        "configured-campaign",
				Description: "Campaign with project settings",
				ProjectSettings: map[string]interface{}{
					"environment": "production",
					"timeout":     300,
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

func TestLoadCampaignConfig(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "campaign-test")
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
		wantConfig *CampaignConfig
	}{
		{
			name: "valid campaign config",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				config := &CampaignConfig{
					Name:              "test-campaign",
					Description:       "Test campaign description",
					LastSelectedGuild: "backend-guild",
					CommissionMappings: map[string][]string{
						"backend-guild": {"commission1", "commission2"},
					},
				}
				return saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yaml"), config)
			},
			wantErr: false,
			wantConfig: &CampaignConfig{
				Name:              "test-campaign",
				Description:       "Test campaign description",
				LastSelectedGuild: "backend-guild",
				CommissionMappings: map[string][]string{
					"backend-guild": {"commission1", "commission2"},
				},
			},
		},
		{
			name: "missing campaign config",
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
				return os.WriteFile(filepath.Join(guildDir, "campaign.yaml"), []byte("invalid: yaml: content:"), 0644)
			},
			wantErr: true,
			errCode: gerror.ErrCodeValidation,
		},
		{
			name: "invalid campaign data",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				// Missing required fields
				return os.WriteFile(filepath.Join(guildDir, "campaign.yaml"), []byte("name: test\n"), 0644)
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

			config, err := LoadCampaignConfig(ctx, tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadCampaignConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("LoadCampaignConfig() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}
			if config != nil && tt.wantConfig != nil {
				if config.Name != tt.wantConfig.Name {
					t.Errorf("LoadCampaignConfig() name = %v, want %v", config.Name, tt.wantConfig.Name)
				}
				if config.Description != tt.wantConfig.Description {
					t.Errorf("LoadCampaignConfig() description = %v, want %v", config.Description, tt.wantConfig.Description)
				}
			}
		})
	}
}

func TestLoadCampaignConfig_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "campaign-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid config
	guildDir := filepath.Join(tempDir, ".campaign")
	os.MkdirAll(guildDir, 0755)
	config := &CampaignConfig{
		Name:        "test",
		Description: "test",
	}
	saveCampaignConfigYAML(filepath.Join(guildDir, "campaign.yaml"), config)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = LoadCampaignConfig(ctx, tempDir)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestSaveCampaignConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "campaign-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	tests := []struct {
		name    string
		config  *CampaignConfig
		setup   func() error
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "save valid config",
			config: &CampaignConfig{
				Name:        "save-test",
				Description: "Testing save functionality",
				CommissionMappings: map[string][]string{
					"guild1": {"commission1"},
				},
			},
			wantErr: false,
		},
		{
			name: "save with existing directory",
			config: &CampaignConfig{
				Name:        "existing-dir-test",
				Description: "Testing save with existing directory",
			},
			setup: func() error {
				return os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)
			},
			wantErr: false,
		},
		{
			name: "save with last selected guild",
			config: &CampaignConfig{
				Name:              "last-selected-test",
				Description:       "Testing last selected guild persistence",
				LastSelectedGuild: "frontend-guild",
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

			err := SaveCampaignConfig(ctx, tempDir, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveCampaignConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errCode != "" {
				if gerror.GetCode(err) != tt.errCode {
					t.Errorf("SaveCampaignConfig() error code = %v, want %v", gerror.GetCode(err), tt.errCode)
				}
			}

			// Verify the file was created and can be loaded
			if !tt.wantErr {
				loaded, err := LoadCampaignConfig(ctx, tempDir)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
				}
				if loaded.Name != tt.config.Name {
					t.Errorf("Loaded name = %v, want %v", loaded.Name, tt.config.Name)
				}
			}
		})
	}
}

func TestSaveCampaignConfig_ContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "campaign-save-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &CampaignConfig{
		Name:        "test",
		Description: "test",
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = SaveCampaignConfig(ctx, tempDir, config)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if gerror.GetCode(err) != gerror.ErrCodeCancelled {
		t.Errorf("Expected error code %v, got %v", gerror.ErrCodeCancelled, gerror.GetCode(err))
	}
}

func TestUpdateLastSelectedGuild(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "campaign-update-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create initial config
	guildDir := filepath.Join(tempDir, ".campaign")
	os.MkdirAll(guildDir, 0755)

	config := &CampaignConfig{
		Name:              "update-test",
		Description:       "Testing update functionality",
		LastSelectedGuild: "initial-guild",
	}

	// Save initial config
	if err := SaveCampaignConfig(ctx, tempDir, config); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Update last selected guild
	if err := config.UpdateLastSelectedGuild(ctx, tempDir, "new-guild"); err != nil {
		t.Errorf("UpdateLastSelectedGuild() error = %v", err)
	}

	// Verify update
	if config.LastSelectedGuild != "new-guild" {
		t.Errorf("LastSelectedGuild = %v, want %v", config.LastSelectedGuild, "new-guild")
	}

	// Load and verify persistence
	loaded, err := LoadCampaignConfig(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}
	if loaded.LastSelectedGuild != "new-guild" {
		t.Errorf("Loaded LastSelectedGuild = %v, want %v", loaded.LastSelectedGuild, "new-guild")
	}
}

func TestCampaignConfig_GetMappedCommissions(t *testing.T) {
	tests := []struct {
		name      string
		config    *CampaignConfig
		guildName string
		want      []string
	}{
		{
			name: "existing guild with commissions",
			config: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"backend-guild":  {"api-refactor", "database-migration"},
					"frontend-guild": {"ui-redesign"},
				},
			},
			guildName: "backend-guild",
			want:      []string{"api-refactor", "database-migration"},
		},
		{
			name: "existing guild with no commissions",
			config: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"backend-guild": {},
				},
			},
			guildName: "backend-guild",
			want:      []string{},
		},
		{
			name: "non-existent guild",
			config: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"backend-guild": {"commission1"},
				},
			},
			guildName: "frontend-guild",
			want:      []string{},
		},
		{
			name:      "nil commission mappings",
			config:    &CampaignConfig{},
			guildName: "any-guild",
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetMappedCommissions(tt.guildName)
			if len(got) != len(tt.want) {
				t.Errorf("GetMappedCommissions() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if i < len(tt.want) && got[i] != tt.want[i] {
					t.Errorf("GetMappedCommissions()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCampaignConfig_MapGuildToCommissions(t *testing.T) {
	tests := []struct {
		name        string
		config      *CampaignConfig
		guildName   string
		commissions []string
		verify      func(t *testing.T, c *CampaignConfig)
	}{
		{
			name:        "map to new guild",
			config:      &CampaignConfig{},
			guildName:   "backend-guild",
			commissions: []string{"commission1", "commission2"},
			verify: func(t *testing.T, c *CampaignConfig) {
				mapped := c.GetMappedCommissions("backend-guild")
				if len(mapped) != 2 {
					t.Errorf("Expected 2 commissions, got %d", len(mapped))
				}
			},
		},
		{
			name: "overwrite existing mapping",
			config: &CampaignConfig{
				CommissionMappings: map[string][]string{
					"backend-guild": {"old-commission"},
				},
			},
			guildName:   "backend-guild",
			commissions: []string{"new-commission1", "new-commission2"},
			verify: func(t *testing.T, c *CampaignConfig) {
				mapped := c.GetMappedCommissions("backend-guild")
				if len(mapped) != 2 {
					t.Errorf("Expected 2 commissions, got %d", len(mapped))
				}
				if mapped[0] != "new-commission1" {
					t.Errorf("Expected first commission to be 'new-commission1', got %v", mapped[0])
				}
			},
		},
		{
			name:        "empty commissions list",
			config:      &CampaignConfig{},
			guildName:   "empty-guild",
			commissions: []string{},
			verify: func(t *testing.T, c *CampaignConfig) {
				mapped := c.GetMappedCommissions("empty-guild")
				if len(mapped) != 0 {
					t.Errorf("Expected 0 commissions, got %d", len(mapped))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.MapGuildToCommissions(tt.guildName, tt.commissions)
			if tt.verify != nil {
				tt.verify(t, tt.config)
			}
		})
	}
}

func TestCampaignConfig_ProjectSettings(t *testing.T) {
	config := &CampaignConfig{}

	// Test getting from nil map
	val, ok := config.GetProjectSetting("key")
	if ok {
		t.Error("Expected false for non-existent key in nil map")
	}
	if val != nil {
		t.Error("Expected nil value for non-existent key")
	}

	// Test setting creates map
	config.SetProjectSetting("environment", "production")
	val, ok = config.GetProjectSetting("environment")
	if !ok {
		t.Error("Expected true for existing key")
	}
	if val != "production" {
		t.Errorf("Expected 'production', got %v", val)
	}

	// Test overwriting value
	config.SetProjectSetting("environment", "staging")
	val, ok = config.GetProjectSetting("environment")
	if val != "staging" {
		t.Errorf("Expected 'staging', got %v", val)
	}

	// Test different types
	config.SetProjectSetting("timeout", 300)
	config.SetProjectSetting("enabled", true)
	config.SetProjectSetting("tags", []string{"tag1", "tag2"})

	if timeout, _ := config.GetProjectSetting("timeout"); timeout != 300 {
		t.Errorf("Expected 300, got %v", timeout)
	}
	if enabled, _ := config.GetProjectSetting("enabled"); enabled != true {
		t.Errorf("Expected true, got %v", enabled)
	}
}

// Helper function to save campaign config as YAML
func saveCampaignConfigYAML(path string, config *CampaignConfig) error {
	data, err := os.ReadFile("testdata/campaign_config_template.yml")
	if err != nil {
		// If template doesn't exist, use simple YAML
		simpleYAML := `name: ` + config.Name + `
description: ` + config.Description
		if config.LastSelectedGuild != "" {
			simpleYAML += "\nlast_selected_guild: " + config.LastSelectedGuild
		}
		if len(config.CommissionMappings) > 0 {
			simpleYAML += "\ncommission_mappings:"
			for guild, commissions := range config.CommissionMappings {
				simpleYAML += "\n  " + guild + ":"
				for _, c := range commissions {
					simpleYAML += "\n    - " + c
				}
			}
		}
		data = []byte(simpleYAML)
	}
	return os.WriteFile(path, data, 0644)
}

// TestCampaignConfig_RealWorldScenarios tests more complex real-world scenarios
func TestCampaignConfig_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "campaign-realworld-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("multi-guild campaign workflow", func(t *testing.T) {
		// Create a campaign with multiple guilds and commissions
		config := &CampaignConfig{
			Name:        "e-commerce-migration",
			Description: "Migrate legacy e-commerce platform to microservices",
			ProjectSettings: map[string]interface{}{
				"environment": "production",
				"version":     "2.0",
				"features":    []string{"api-gateway", "service-mesh", "monitoring"},
			},
		}

		// Save initial config
		if err := SaveCampaignConfig(ctx, tempDir, config); err != nil {
			t.Fatalf("Failed to save campaign: %v", err)
		}

		// Map guilds to commissions
		config.MapGuildToCommissions("backend-guild", []string{
			"api-gateway-implementation",
			"service-mesh-setup",
			"database-migration",
		})
		config.MapGuildToCommissions("frontend-guild", []string{
			"ui-component-library",
			"responsive-design",
		})
		config.MapGuildToCommissions("devops-guild", []string{
			"ci-cd-pipeline",
			"monitoring-setup",
		})

		// Update last selected guild
		config.UpdateLastSelectedGuild(ctx, tempDir, "backend-guild")

		// Verify the complete configuration
		loaded, err := LoadCampaignConfig(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to load campaign: %v", err)
		}

		// Check all mappings are preserved
		backendCommissions := loaded.GetMappedCommissions("backend-guild")
		if len(backendCommissions) != 3 {
			t.Errorf("Expected 3 backend commissions, got %d", len(backendCommissions))
		}

		// Check project settings
		env, ok := loaded.GetProjectSetting("environment")
		if !ok || env != "production" {
			t.Errorf("Expected environment=production, got %v", env)
		}

		// Verify last selected guild
		if loaded.LastSelectedGuild != "backend-guild" {
			t.Errorf("Expected LastSelectedGuild=backend-guild, got %v", loaded.LastSelectedGuild)
		}
	})

	t.Run("concurrent updates", func(t *testing.T) {
		config := &CampaignConfig{
			Name:        "concurrent-test",
			Description: "Test concurrent updates",
		}

		if err := SaveCampaignConfig(ctx, tempDir, config); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Simulate concurrent updates
		done := make(chan bool, 3)
		errors := make(chan error, 3)

		go func() {
			c, _ := LoadCampaignConfig(ctx, tempDir)
			c.SetProjectSetting("worker1", "value1")
			errors <- SaveCampaignConfig(ctx, tempDir, c)
			done <- true
		}()

		go func() {
			c, _ := LoadCampaignConfig(ctx, tempDir)
			c.SetProjectSetting("worker2", "value2")
			errors <- SaveCampaignConfig(ctx, tempDir, c)
			done <- true
		}()

		go func() {
			c, _ := LoadCampaignConfig(ctx, tempDir)
			c.MapGuildToCommissions("concurrent-guild", []string{"commission1"})
			errors <- SaveCampaignConfig(ctx, tempDir, c)
			done <- true
		}()

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		// Check for errors
		close(errors)
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent update error: %v", err)
			}
		}
	})
}

// Benchmark tests
func BenchmarkCampaignConfig_LoadSave(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "campaign-bench")
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	config := &CampaignConfig{
		Name:        "benchmark-campaign",
		Description: "Campaign for benchmarking",
		CommissionMappings: map[string][]string{
			"guild1": {"c1", "c2", "c3"},
			"guild2": {"c4", "c5"},
		},
		ProjectSettings: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	// Initial save
	SaveCampaignConfig(ctx, tempDir, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loaded, _ := LoadCampaignConfig(ctx, tempDir)
		loaded.LastSelectedGuild = "guild1"
		SaveCampaignConfig(ctx, tempDir, loaded)
	}
}

func BenchmarkCampaignConfig_GetMappedCommissions(b *testing.B) {
	config := &CampaignConfig{
		CommissionMappings: make(map[string][]string),
	}

	// Create many guilds with commissions
	for i := 0; i < 100; i++ {
		guildName := "guild" + string(rune(i))
		commissions := make([]string, 10)
		for j := 0; j < 10; j++ {
			commissions[j] = "commission" + string(rune(j))
		}
		config.CommissionMappings[guildName] = commissions
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetMappedCommissions("guild50")
	}
}
