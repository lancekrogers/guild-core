// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/config"
)

func TestAgentTemplateGenerator_GenerateAgentConfig(t *testing.T) {
	generator := NewAgentTemplateGenerator()

	tests := []struct {
		name     string
		template AgentTemplate
		wantErr  bool
		validate func(t *testing.T, cfg *config.AgentConfig)
	}{
		{
			name: "minimal template",
			template: AgentTemplate{
				ID:           "test-agent",
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "openai",
				Model:        "gpt-4",
				Description:  "A test agent",
				Capabilities: []string{"testing"},
			},
			validate: func(t *testing.T, cfg *config.AgentConfig) {
				assert.Equal(t, "test-agent", cfg.ID)
				assert.Equal(t, "Test Agent", cfg.Name)
				assert.Equal(t, "worker", cfg.Type)
				assert.Equal(t, 3000, cfg.MaxTokens) // Default for worker
				assert.Equal(t, 0.4, cfg.Temperature) // Default for worker
				assert.Nil(t, cfg.Backstory) // No backstory fields provided
			},
		},
		{
			name: "template with optional backstory",
			template: AgentTemplate{
				ID:           "expert-agent",
				Name:         "Expert Agent",
				Type:         "specialist",
				Provider:     "anthropic",
				Model:        "claude-3-opus",
				Description:  "An expert specialist",
				Capabilities: []string{"expertise"},
				Experience:   "10 years in the field",
				Philosophy:   "Excellence in all things",
			},
			validate: func(t *testing.T, cfg *config.AgentConfig) {
				assert.Equal(t, "expert-agent", cfg.ID)
				assert.Equal(t, 3500, cfg.MaxTokens) // Default for specialist
				assert.NotNil(t, cfg.Backstory)
				assert.Equal(t, "10 years in the field", cfg.Backstory.Experience)
				assert.Equal(t, "Excellence in all things", cfg.Backstory.Philosophy)
				assert.Empty(t, cfg.Backstory.Expertise) // Not provided
			},
		},
		{
			name: "manager with custom values",
			template: AgentTemplate{
				ID:            "custom-manager",
				Name:          "Custom Manager",
				Type:          "manager",
				Provider:      "openai",
				Model:         "gpt-4-turbo",
				Description:   "Custom configuration",
				Capabilities:  []string{"management"},
				MaxTokens:     5000,
				Temperature:   0.2,
				CostMagnitude: 5,
				ContextWindow: 128000,
			},
			validate: func(t *testing.T, cfg *config.AgentConfig) {
				assert.Equal(t, 5000, cfg.MaxTokens) // Custom value
				assert.Equal(t, 0.2, cfg.Temperature) // Custom value
				assert.Equal(t, 5, cfg.CostMagnitude)
				assert.Equal(t, 128000, cfg.ContextWindow)
			},
		},
		{
			name: "missing required field",
			template: AgentTemplate{
				// Missing ID
				Name:         "Invalid Agent",
				Type:         "worker",
				Provider:     "openai",
				Capabilities: []string{"testing"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := generator.GenerateAgentConfig(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)
			
			// Validate system prompt is generated
			assert.NotEmpty(t, cfg.SystemPrompt)
			assert.Contains(t, cfg.SystemPrompt, cfg.Name)
			
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestAgentTemplateGenerator_GenerateAgentFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	generator := NewAgentTemplateGenerator()
	
	template := AgentTemplate{
		ID:           "file-test-agent",
		Name:         "File Test Agent",
		Type:         "worker",
		Provider:     "ollama",
		Model:        "llama3",
		Description:  "Agent for file generation test",
		Capabilities: []string{"testing", "validation"},
		Experience:   "Fresh out of training",
	}
	
	// Generate the file
	ctx := context.Background()
	err := generator.GenerateAgentFile(ctx, tmpDir, template)
	require.NoError(t, err)
	
	// Verify file exists
	expectedPath := filepath.Join(tmpDir, ".campaign", "agents", "file-test-agent.yml")
	assert.FileExists(t, expectedPath)
	
	// Read and validate the file
	data, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	
	var loadedConfig config.AgentConfig
	err = yaml.Unmarshal(data, &loadedConfig)
	require.NoError(t, err)
	
	// Validate content
	assert.Equal(t, "file-test-agent", loadedConfig.ID)
	assert.Equal(t, "File Test Agent", loadedConfig.Name)
	assert.Equal(t, "ollama", loadedConfig.Provider)
	assert.NotNil(t, loadedConfig.Backstory)
	assert.Equal(t, "Fresh out of training", loadedConfig.Backstory.Experience)
}

func TestAgentTemplateGenerator_BuiltInTemplates(t *testing.T) {
	generator := NewAgentTemplateGenerator()
	
	// Test listing templates
	templates := generator.ListTemplates()
	assert.NotEmpty(t, templates)
	assert.Contains(t, templates, "claude-code-manager")
	assert.Contains(t, templates, "ollama-coder")
	assert.Contains(t, templates, "openai-developer")
	
	// Test getting specific template
	claudeTemplate, exists := generator.GetTemplate("claude-code-developer")
	assert.True(t, exists)
	assert.Equal(t, "claude-developer", claudeTemplate.ID)
	assert.Equal(t, "claude_code", claudeTemplate.Provider)
	
	// Test provider filtering
	ollamaTemplates := generator.ProviderTemplates("ollama")
	assert.NotEmpty(t, ollamaTemplates)
	for _, tmpl := range ollamaTemplates {
		assert.True(t, tmpl.Provider == "ollama" || tmpl.Provider == "")
	}
}

func TestAgentTemplateGenerator_CreateCustomTemplate(t *testing.T) {
	generator := NewAgentTemplateGenerator()
	
	custom := generator.CreateCustomTemplate(
		"my-agent",
		"My Custom Agent",
		"specialist",
		"anthropic",
		"claude-3-haiku",
		"A custom specialist agent",
		[]string{"custom_capability", "special_skill"},
	)
	
	assert.Equal(t, "my-agent", custom.ID)
	assert.Equal(t, "My Custom Agent", custom.Name)
	assert.Equal(t, "specialist", custom.Type)
	assert.Empty(t, custom.Experience) // Optional fields are empty
	assert.Empty(t, custom.Philosophy)
	assert.Equal(t, 0, custom.MaxTokens) // Will get default when generated
}

func TestAgentTemplateGenerator_QuickSetup(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewAgentTemplateGenerator()
	
	// Run quick setup
	ctx := context.Background()
	err := generator.QuickSetup(ctx, tmpDir, "openai", "gpt-4", "gpt-3.5-turbo")
	require.NoError(t, err)
	
	// Verify manager file
	managerPath := filepath.Join(tmpDir, ".campaign", "agents", "manager.yml")
	assert.FileExists(t, managerPath)
	
	managerData, err := os.ReadFile(managerPath)
	require.NoError(t, err)
	
	var managerConfig config.AgentConfig
	err = yaml.Unmarshal(managerData, &managerConfig)
	require.NoError(t, err)
	
	assert.Equal(t, "manager", managerConfig.ID)
	assert.Equal(t, "Guild Manager", managerConfig.Name)
	assert.Equal(t, "manager", managerConfig.Type)
	assert.Equal(t, "openai", managerConfig.Provider)
	assert.Equal(t, "gpt-4", managerConfig.Model)
	
	// Verify worker file
	workerPath := filepath.Join(tmpDir, ".campaign", "agents", "worker-1.yml")
	assert.FileExists(t, workerPath)
	
	workerData, err := os.ReadFile(workerPath)
	require.NoError(t, err)
	
	var workerConfig config.AgentConfig
	err = yaml.Unmarshal(workerData, &workerConfig)
	require.NoError(t, err)
	
	assert.Equal(t, "worker-1", workerConfig.ID)
	assert.Equal(t, "Primary Worker", workerConfig.Name)
	assert.Equal(t, "worker", workerConfig.Type)
	assert.Equal(t, "openai", workerConfig.Provider)
	assert.Equal(t, "gpt-3.5-turbo", workerConfig.Model)
}

func TestAgentTemplateGenerator_SystemPromptGeneration(t *testing.T) {
	generator := NewAgentTemplateGenerator()
	
	tests := []struct {
		name           string
		template       AgentTemplate
		wantContains   []string
	}{
		{
			name: "basic prompt",
			template: AgentTemplate{
				ID:           "test",
				Name:         "Test Agent",
				Type:         "worker",
				Provider:     "test",
				Description:  "A helpful assistant",
				Capabilities: []string{"coding", "testing"},
			},
			wantContains: []string{
				"Test Agent",
				"worker agent",
				"A helpful assistant",
				"coding, testing",
			},
		},
		{
			name: "with backstory elements",
			template: AgentTemplate{
				ID:           "test",
				Name:         "Expert Agent",
				Type:         "specialist",
				Provider:     "test",
				Description:  "An expert",
				Capabilities: []string{"expertise"},
				Experience:   "20 years of experience",
				Expertise:    "Deep knowledge of systems",
				Philosophy:   "Quality above all",
			},
			wantContains: []string{
				"Expert Agent",
				"Experience: 20 years of experience",
				"Expertise: Deep knowledge of systems",
				"Philosophy: Quality above all",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := generator.GenerateAgentConfig(tt.template)
			require.NoError(t, err)
			
			for _, want := range tt.wantContains {
				assert.Contains(t, cfg.SystemPrompt, want)
			}
		})
	}
}