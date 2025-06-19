// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// TestValidatorBasic tests basic validator functionality
func TestValidatorBasic(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create validator
	validator := NewInitValidator(tmpDir)
	assert.NotNil(t, validator)
	assert.Equal(t, tmpDir, validator.projectPath)
	assert.Empty(t, validator.results)
	assert.False(t, validator.hasFailures)
	assert.False(t, validator.hasWarnings)
}

// TestValidateProjectStructure tests project structure validation
func TestValidateProjectStructure(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error
		expectSuccess bool
		expectWarning bool
	}{
		{
			name: "complete structure",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				dirs := []string{
					"agents", "archives", "campaigns", "corpus",
					"guilds", "kanban", "objectives", "prompts",
				}
				for _, dir := range dirs {
					if err := os.MkdirAll(filepath.Join(guildDir, dir), 0755); err != nil {
						return err
					}
				}
				return nil
			},
			expectSuccess: true,
			expectWarning: false,
		},
		{
			name: "missing some directories",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				// Only create some directories
				dirs := []string{"agents", "archives", "campaigns"}
				for _, dir := range dirs {
					if err := os.MkdirAll(filepath.Join(guildDir, dir), 0755); err != nil {
						return err
					}
				}
				return nil
			},
			expectSuccess: true,
			expectWarning: true,
		},
		{
			name:          "no .guild directory",
			setup:         func(tmpDir string) error { return nil },
			expectSuccess: false,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, tt.setup(tmpDir))

			validator := NewInitValidator(tmpDir)
			result := validator.validateProjectStructure(context.Background())

			assert.Equal(t, "Project Structure", result.Name)
			assert.Equal(t, tt.expectSuccess, result.Success)
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warning)
			} else {
				assert.Empty(t, result.Warning)
			}
		})
	}
}

// TestValidateCampaignConfiguration tests campaign configuration validation
func TestValidateCampaignConfiguration(t *testing.T) {
	// Skip this test as it requires global campaign setup
	t.Skip("Requires global campaign configuration")
}

// TestValidateGuildConfiguration tests guild configuration validation
func TestValidateGuildConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error
		expectSuccess bool
		expectWarning bool
	}{
		{
			name: "valid guild config",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				
				guildConfig := &config.GuildConfigFile{
					Guilds: map[string]config.GuildDefinition{
						"test-guild": {
							Purpose:     "Test guild",
							Description: "A test guild",
							Agents:      []string{"agent1", "agent2"},
						},
					},
				}
				
				return config.SaveGuildConfigFile(context.Background(), tmpDir, guildConfig)
			},
			expectSuccess: true,
			expectWarning: false,
		},
		{
			name: "guild with no agents",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				
				guildConfig := &config.GuildConfigFile{
					Guilds: map[string]config.GuildDefinition{
						"empty-guild": {
							Purpose:     "Empty guild",
							Description: "A guild with no agents",
							Agents:      []string{},
						},
					},
				}
				
				return config.SaveGuildConfigFile(context.Background(), tmpDir, guildConfig)
			},
			expectSuccess: true,
			expectWarning: true,
		},
		{
			name: "no guild config file",
			setup: func(tmpDir string) error {
				return os.MkdirAll(filepath.Join(tmpDir, ".guild"), 0755)
			},
			expectSuccess: false,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, tt.setup(tmpDir))

			validator := NewInitValidator(tmpDir)
			result := validator.validateGuildConfiguration(context.Background())

			assert.Equal(t, "Guild Configuration", result.Name)
			assert.Equal(t, tt.expectSuccess, result.Success)
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warning)
			} else {
				assert.Empty(t, result.Warning)
			}
		})
	}
}

// TestValidateWithContextCancellation tests validation with context cancellation
func TestValidateWithContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewInitValidator(tmpDir)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := validator.Validate(ctx)
	require.Error(t, err)
	assert.Equal(t, gerror.ErrCodeCancelled, gerror.GetCode(err))
}

// TestValidateFullSuccess tests a successful validation scenario
func TestValidateFullSuccess(t *testing.T) {
	// Skip this test as it requires full environment setup
	t.Skip("Requires full environment setup including database and providers")
}

// TestInitValidationResultsDisplay tests the display functionality
func TestInitValidationResultsDisplay(t *testing.T) {
	validator := NewInitValidator("/test/path")
	
	// Add some test results
	validator.results = []InitValidationResult{
		{
			Name:        "Test Success",
			Description: "A successful test",
			Success:     true,
			Details:     map[string]string{"key": "value"},
		},
		{
			Name:        "Test Warning",
			Description: "A test with warning",
			Success:     true,
			Warning:     "This is a warning",
		},
		{
			Name:        "Test Failure",
			Description: "A failed test",
			Success:     false,
			Error:       gerror.New(gerror.ErrCodeValidation, "test error", nil),
		},
	}
	validator.hasWarnings = true
	validator.hasFailures = true

	// Test that PrintResults doesn't panic
	validator.PrintResults()

	// Test result accessors
	assert.True(t, validator.HasFailures())
	assert.True(t, validator.HasWarnings())
	assert.Equal(t, 1, validator.countFailures())
	assert.Len(t, validator.GetResults(), 3)
}

// TestSocketRegistryValidation tests socket registry validation
func TestSocketRegistryValidation(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error
		expectSuccess bool
		expectWarning bool
	}{
		{
			name: "valid socket registry",
			setup: func(tmpDir string) error {
				// Create .guild directory
				if err := os.MkdirAll(filepath.Join(tmpDir, ".guild"), 0755); err != nil {
					return err
				}
				// Save socket registry
				return daemon.SaveSocketRegistry(tmpDir, "test-campaign")
			},
			expectSuccess: true,
			expectWarning: false,
		},
		{
			name: "missing socket registry",
			setup: func(tmpDir string) error {
				return os.MkdirAll(filepath.Join(tmpDir, ".guild"), 0755)
			},
			expectSuccess: true,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, tt.setup(tmpDir))

			validator := NewInitValidator(tmpDir)
			result := validator.validateSocketRegistry(context.Background())

			assert.Equal(t, "Socket Registry", result.Name)
			assert.Equal(t, tt.expectSuccess, result.Success)
			if tt.expectWarning {
				assert.NotEmpty(t, result.Warning)
			} else {
				assert.Empty(t, result.Warning)
			}
		})
	}
}

// TestDatabaseValidation tests database validation
func TestDatabaseValidation(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error
		expectSuccess bool
	}{
		{
			name: "missing database",
			setup: func(tmpDir string) error {
				return os.MkdirAll(filepath.Join(tmpDir, ".guild"), 0755)
			},
			expectSuccess: false,
		},
		// Note: Testing with actual database requires SQLite setup
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, tt.setup(tmpDir))

			validator := NewInitValidator(tmpDir)
			result := validator.validateDatabaseInitialization(context.Background())

			assert.Equal(t, "Database Initialization", result.Name)
			assert.Equal(t, tt.expectSuccess, result.Success)
		})
	}
}

// TestProviderValidation tests provider configuration validation
func TestProviderValidation(t *testing.T) {
	// Save current env vars
	oldOpenAI := os.Getenv("OPENAI_API_KEY")
	oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		os.Setenv("OPENAI_API_KEY", oldOpenAI)
		os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
	}()

	tests := []struct {
		name          string
		setup         func(string) error
		envSetup      func()
		expectSuccess bool
	}{
		{
			name: "providers with credentials",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				
				// Create guild config with OpenAI agent
				guildConfig := &config.GuildConfig{
					Name: "test",
					Agents: []config.AgentConfig{
						{
							ID:       "test-agent",
							Provider: "openai",
							Model:    "gpt-4",
						},
					},
				}
				
				return config.SaveGuildConfig(tmpDir, guildConfig)
			},
			envSetup: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
			},
			expectSuccess: true,
		},
		{
			name: "missing credentials",
			setup: func(tmpDir string) error {
				guildDir := filepath.Join(tmpDir, ".guild")
				if err := os.MkdirAll(guildDir, 0755); err != nil {
					return err
				}
				
				// Create guild config with OpenAI agent
				guildConfig := &config.GuildConfig{
					Name: "test",
					Agents: []config.AgentConfig{
						{
							ID:       "test-agent",
							Provider: "openai",
							Model:    "gpt-4",
						},
					},
				}
				
				return config.SaveGuildConfig(tmpDir, guildConfig)
			},
			envSetup: func() {
				os.Unsetenv("OPENAI_API_KEY")
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, tt.setup(tmpDir))
			tt.envSetup()

			validator := NewInitValidator(tmpDir)
			result := validator.validateProviderConfiguration(context.Background())

			assert.Equal(t, "Provider Configuration", result.Name)
			assert.Equal(t, tt.expectSuccess, result.Success)
		})
	}
}

// TestValidationTimeout tests validation with timeout
func TestValidationTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewInitValidator(tmpDir)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the context time to expire
	time.Sleep(10 * time.Millisecond)

	err := validator.Validate(ctx)
	require.Error(t, err)
	// Could be either cancelled or deadline exceeded
	code := gerror.GetCode(err)
	assert.True(t, code == gerror.ErrCodeCancelled || code == gerror.ErrCodeTimeout)
}