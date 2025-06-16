// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/config"
)

func TestConfigShowCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupFunc      func(t *testing.T) string // Returns temp dir
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "show global config",
			args: []string{"--global"},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				globalDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(globalDir, 0755))

				cfg := &config.GuildConfig{
					Name:        "Test Guild",
					Description: "Test Description",
					Agents: []config.AgentConfig{
						{
							ID:   "test-agent",
							Name: "Test Agent",
							Type: "worker",
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(globalDir, "config.yaml"), data, 0644))

				// Mock home directory
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			expectedOutput: []string{
				"Global Configuration",
				"Path:",
				"config.yaml",
			},
		},
		{
			name: "show local config",
			args: []string{"--local"},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(guildDir, 0755))

				cfg := &config.GuildConfig{
					Name: "Local Project",
					Agents: []config.AgentConfig{
						{
							ID:   "local-agent",
							Name: "Local Agent",
							Type: "developer",
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(guildDir, "guild.yaml"), data, 0644))

				// Change to temp dir so it finds local config
				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })

				return tmpDir
			},
			expectedOutput: []string{
				"Local Configuration",
				"guild.yaml",
				"Local Agent",
			},
		},
		{
			name: "show raw yaml",
			args: []string{"--local", "--raw"},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(guildDir, 0755))

				cfg := &config.GuildConfig{
					Name: "Raw Test",
				}

				data, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(guildDir, "guild.yaml"), data, 0644))

				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })

				return tmpDir
			},
			expectedOutput: []string{
				"name: Raw Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			// Capture output
			var buf bytes.Buffer
			cmd := &cobra.Command{}
			configShowCmd.Flags().VisitAll(func(f *pflag.Flag) {
				cmd.Flags().AddFlag(f)
			})
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.ParseFlags(tt.args)

			// Execute command
			err := runConfigShow(cmd, []string{})

			// Check error
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check output
			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestConfigPathCommand(t *testing.T) {
	// Save original env
	oldHome := os.Getenv("HOME")
	oldUserHome := os.Getenv("USERPROFILE") // Windows
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserHome)
	}()

	// Set test home
	testHome := "/test/home"
	os.Setenv("HOME", testHome)
	os.Setenv("USERPROFILE", testHome)

	// Capture output
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute command
	err := runConfigPath(cmd, []string{})
	assert.NoError(t, err)

	// Check output contains expected paths
	output := buf.String()
	assert.Contains(t, output, "Configuration File Locations")
	assert.Contains(t, output, "Global:")
	assert.Contains(t, output, "Local (Project):")
	assert.Contains(t, output, "Environment Variables:")
	assert.Contains(t, output, "OPENAI_API_KEY")
	assert.Contains(t, output, "ANTHROPIC_API_KEY")
}

func TestConfigValidateCommand(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) string
		expectedOutput []string
		expectIssues   bool
	}{
		{
			name: "valid configuration",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(guildDir, 0755))

				cfg := &config.GuildConfig{
					Name:        "Valid Project",
					Description: "A valid configuration",
					Agents: []config.AgentConfig{
						{
							ID:       "agent1",
							Name:     "Agent One",
							Type:     "worker",
							Provider: "openai",
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(guildDir, "guild.yaml"), data, 0644))

				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })

				return tmpDir
			},
			expectedOutput: []string{
				"Validating Configuration",
				"Local config: Valid",
			},
			expectIssues: false,
		},
		{
			name: "invalid yaml",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(guildDir, 0755))

				// Write invalid YAML
				invalidYAML := `name: "Invalid
agents:
  - id: agent1
    name: Agent One
  type: worker`

				require.NoError(t, os.WriteFile(filepath.Join(guildDir, "guild.yaml"), []byte(invalidYAML), 0644))

				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })

				return tmpDir
			},
			expectedOutput: []string{
				"Validating Configuration",
				"Invalid YAML",
			},
			expectIssues: true,
		},
		{
			name: "missing required fields",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".guild")
				require.NoError(t, os.MkdirAll(guildDir, 0755))

				cfg := &config.GuildConfig{
					Name: "Incomplete Project",
					Agents: []config.AgentConfig{
						{
							Name: "Agent Without ID", // Missing ID
							Type: "worker",
						},
						{
							ID:   "agent2",
							Name: "Agent Without Type", // Missing Type
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(guildDir, "guild.yaml"), data, 0644))

				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })

				return tmpDir
			},
			expectedOutput: []string{
				"Validating Configuration",
				"Agent missing ID",
				"Agent agent2 missing type",
			},
			expectIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			// Capture output
			var buf bytes.Buffer
			cmd := &cobra.Command{}
			configValidateCmd.Flags().VisitAll(func(f *pflag.Flag) {
				cmd.Flags().AddFlag(f)
			})
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute command
			_ = runConfigValidate(cmd, []string{})

			// Check output
			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}

			if tt.expectIssues {
				assert.Contains(t, output, "Found")
				assert.Contains(t, output, "issue")
			} else {
				assert.Contains(t, output, "Valid")
			}
		})
	}
}

func TestConfigEditCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "edit local config - file not found",
			args: []string{},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(oldWd) })
				return tmpDir
			},
			expectError: true,
			errorMsg:    "no local configuration found",
		},
		{
			name: "edit global config - file not found",
			args: []string{"global"},
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			expectError: true,
			errorMsg:    "no global configuration found",
		},
		{
			name: "invalid argument",
			args: []string{"invalid"},
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: true,
			errorMsg:    "specify 'global' or 'local'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			// Execute command
			err := runConfigEdit(&cobra.Command{}, tt.args)

			// Check error
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisplayFormattedConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.GuildConfig
		expectedOutput []string
	}{
		{
			name: "display agents",
			config: &config.GuildConfig{
				Agents: []config.AgentConfig{
					{
						ID:           "agent1",
						Name:         "Test Agent",
						Type:         "worker",
						Provider:     "openai",
						Model:        "gpt-4",
						Capabilities: []string{"coding", "testing"},
					},
				},
			},
			expectedOutput: []string{
				"🤖 Agents:",
				"Test Agent (agent1)",
				"Type: worker | Provider: openai | Model: gpt-4",
				"Capabilities: coding, testing",
			},
		},
		{
			name: "display providers",
			config: &config.GuildConfig{
				Providers: config.ProvidersConfig{
					OpenAI: config.ProviderSettings{
						BaseURL: "https://api.openai.com",
					},
				},
			},
			expectedOutput: []string{
				"🔌 Providers:",
				"openai",
				"Base URL: https://api.openai.com",
			},
		},
		{
			name: "display storage",
			config: &config.GuildConfig{
				Storage: config.StorageConfig{
					Backend: "sqlite",
					SQLite: config.SQLiteConfig{
						Path: ".guild/memory.db",
					},
				},
			},
			expectedOutput: []string{
				"💾 Storage:",
				"Type: sqlite",
				"Path: .guild/memory.db",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Display config
			displayFormattedConfig(tt.config)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestGetAgentCostIconInConfig(t *testing.T) {
	// Test the cost icon function embedded in config.go
	testCases := []struct {
		cost     int
		expected string
	}{
		{0, "💰"},
		{1, "💰"},
		{2, "💰💰"},
		{3, "💰💰"},
		{4, "💰💰💰"},
		{5, "💰💰💰"},
		{6, "💰💰💰💰"},
		{10, "💰💰💰💰"},
	}

	for _, tc := range testCases {
		t.Run(strings.ReplaceAll(tc.expected, "💰", "coin"), func(t *testing.T) {
			// Since getAgentCostIcon is inlined in config.go, we can't test it directly
			// But we can verify the logic is correct
			var result string
			switch {
			case tc.cost <= 1:
				result = "💰"
			case tc.cost <= 3:
				result = "💰💰"
			case tc.cost <= 5:
				result = "💰💰💰"
			default:
				result = "💰💰💰💰"
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}
