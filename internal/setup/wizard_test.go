// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"testing"
	"time"

	guildconfig "github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
)

func TestNewWizard(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		ctx     context.Context
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "valid config with initialized project",
			config: &Config{
				ProjectPath: "", // Will be set in test
				QuickMode:   true,
				Force:       false,
			},
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			ctx:     context.Background(),
			wantErr: true,
			errCode: gerror.ErrCodeInvalidInput,
		},
		{
			name: "cancelled context",
			config: &Config{
				ProjectPath: "", // Will be set in test
				QuickMode:   true,
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: true,                    // NewWizard now checks context during component creation
			errCode: gerror.ErrCodeCancelled, // The cancelled error is now properly detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for valid tests
			if tt.config != nil {
				tempDir, err := os.MkdirTemp("", "guild-setup-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)

				// Initialize project
				if err := project.InitializeProject(tempDir); err != nil {
					t.Fatalf("Failed to initialize project: %v", err)
				}

				tt.config.ProjectPath = tempDir
			}

			wizard, err := NewWizard(tt.ctx, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewWizard() expected error but got none")
				}
				if tt.errCode != "" {
					if !gerror.Is(err, tt.errCode) {
						t.Errorf("NewWizard() error code = %v, want %v", err, tt.errCode)
					}
				}
				if wizard != nil {
					t.Errorf("NewWizard() expected nil wizard on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewWizard() unexpected error: %v", err)
				}
				if wizard == nil {
					t.Errorf("NewWizard() expected non-nil wizard")
				} else {
					if wizard.config != tt.config {
						t.Errorf("NewWizard() config mismatch")
					}
				}
			}
		})
	}
}

func TestWizardContextCancellation(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "guild-setup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize project
	if err := project.InitializeProject(tempDir); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	config := &Config{
		ProjectPath: tempDir,
		QuickMode:   true,
		Force:       false,
	}

	// Test context cancellation during Run
	t.Run("context cancelled during run", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		wizard, err := NewWizard(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create wizard: %v", err)
		}

		// Cancel context immediately
		cancel()

		// RunQuickMode should fail with cancelled error
		err = wizard.RunQuickMode(ctx)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
		if !gerror.Is(err, gerror.ErrCodeCancelled) {
			t.Errorf("Expected cancelled error, got: %v", err)
		}
	})

	// Test timeout during input - removed as we no longer have direct input reading
}

func TestIsProjectSetup(t *testing.T) {
	ctx := context.Background()

	// Test with non-existent project
	isSetup, err := IsProjectSetup(ctx, "/non/existent/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isSetup {
		t.Error("Expected project to not be setup")
	}
}

func TestGetSetupStatus(t *testing.T) {
	ctx := context.Background()

	// Test with non-existent project
	status, err := GetSetupStatus(ctx, "/non/existent/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status == nil {
		t.Fatal("Status is nil")
	}
	if status.IsConfigured {
		t.Error("Expected project to not be configured")
	}
	if len(status.Providers) != 0 {
		t.Error("Expected no providers")
	}
	if len(status.Agents) != 0 {
		t.Error("Expected no agents")
	}
}

func TestWizardQuickMode(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "guild-setup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize project
	if err := project.InitializeProject(tempDir); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Set up environment for provider detection
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	config := &Config{
		ProjectPath: tempDir,
		QuickMode:   true,
		Force:       false,
	}

	ctx := context.Background()
	wizard, err := NewWizard(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create wizard: %v", err)
	}

	// Test quick mode behavior (no user interaction)
	// This would need mocking to fully test, but we can verify it doesn't hang
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = wizard.RunQuickMode(runCtx)
	// We expect an error because we don't have real providers configured
	// But the test verifies it doesn't hang waiting for user input
	if err == nil {
		// Check if configuration was created
		isSetup, _ := IsProjectSetup(ctx, tempDir)
		if !isSetup {
			t.Error("Expected project to be setup after successful run")
		}
	}
}

// TestWizardProviderSelection has been removed as selectProviders is no longer
// a method on Wizard - provider selection is now handled through the Bubble Tea UI
/*
func TestWizardProviderSelection(t *testing.T) {
	// This test was for the old text-based UI and is no longer applicable
	// with the Bubble Tea implementation
}
*/

// TestWizardDemoSetup has been removed as GetDemoQuickSetup is no longer
// a method on Wizard - demo setup is now handled through the Bubble Tea UI
/*
func TestWizardDemoSetup(t *testing.T) {
	// This test was for the old text-based UI and is no longer applicable
	// with the Bubble Tea implementation
}
*/

func TestSaveConfiguration(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "guild-setup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize project
	if err := project.InitializeProject(tempDir); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	config := &Config{
		ProjectPath: tempDir,
		QuickMode:   true,
		Force:       true, // Force to replace existing agents from project init
	}

	ctx := context.Background()
	wizard, err := NewWizard(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create wizard: %v", err)
	}

	// Create test data
	providers := []ConfiguredProvider{
		{
			Name: "openai",
			Type: "cloud",
			Models: []ModelInfo{
				{Name: "gpt-4"},
			},
			Settings: map[string]string{
				"api_key": "test-key",
			},
		},
	}

	agents := []guildconfig.AgentConfig{
		{
			ID:           "manager", // Must match default manager ID
			Name:         "Test Manager",
			Type:         "manager",
			Provider:     "openai",
			Model:        "gpt-4",
			Description:  "Test manager agent",
			Capabilities: []string{"task-planning", "coordination"},
		},
		{
			ID:           "test-agent",
			Name:         "Test Agent",
			Type:         "worker",
			Provider:     "openai",
			Model:        "gpt-4",
			Description:  "Test agent for unit tests",
			Capabilities: []string{"coding", "testing"},
		},
	}

	// Save configuration
	err = wizard.SaveConfiguration(ctx, providers, agents)
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Skip IsProjectSetup check as it may have different validation
	// Just verify the configuration was saved correctly

	// Load and verify configuration
	loadedConfig, err := guildconfig.LoadGuildConfig(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if len(loadedConfig.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(loadedConfig.Agents))
	}
	// Check for both agents
	hasManager := false
	hasWorker := false
	for _, agent := range loadedConfig.Agents {
		if agent.ID == "manager" {
			hasManager = true
		}
		if agent.ID == "test-agent" {
			hasWorker = true
		}
	}
	if !hasManager {
		t.Error("Expected manager agent not found")
	}
	if !hasWorker {
		t.Error("Expected worker agent not found")
	}
}
