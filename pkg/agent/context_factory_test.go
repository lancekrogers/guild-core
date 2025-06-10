package agent

import (
	"context"
	"testing"

	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test newContextAgentFactory
func TestNewContextAgentFactory(t *testing.T) {
	factory := newContextAgentFactory()
	assert.NotNil(t, factory)
}

// Test CreateAgent
func TestContextAgentFactory_CreateAgent(t *testing.T) {
	tests := []struct {
		name          string
		agentConfig   AgentConfig
		expectError   bool
		errorContains string
		validateAgent func(t *testing.T, agent guildcontext.AgentClient)
	}{
		{
			name: "successful worker agent creation",
			agentConfig: AgentConfig{
				ID:              "test-agent",
				Name:            "Test Agent",
				Type:            "worker",
				Capabilities:    []string{"code", "analysis"},
				DefaultProvider: "provider1",
				Enabled:         true,
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent guildcontext.AgentClient) {
				assert.Equal(t, "test-agent", agent.GetID())
				assert.Equal(t, "Test Agent", agent.GetName())
				assert.Equal(t, 2, len(agent.GetCapabilities()))
			},
		},
		{
			name: "successful manager agent creation",
			agentConfig: AgentConfig{
				ID:              "manager-agent",
				Name:            "Manager Agent",
				Type:            "manager",
				Capabilities:    []string{"coordination", "planning"},
				DefaultProvider: "provider1",
				SystemPrompt:    "You are a manager agent",
				Enabled:         true,
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent guildcontext.AgentClient) {
				assert.Equal(t, "manager-agent", agent.GetID())
				assert.Equal(t, "Manager Agent", agent.GetName())
				contextAgent, ok := agent.(*ContextAwareAgent)
				require.True(t, ok)
				assert.Equal(t, "manager", contextAgent.AgentType)
				assert.Equal(t, "You are a manager agent", contextAgent.systemPrompt)
			},
		},
		{
			name: "successful specialist agent creation",
			agentConfig: AgentConfig{
				ID:              "specialist-agent",
				Name:            "Specialist Agent",
				Type:            "specialist",
				Capabilities:    []string{"research", "analysis"},
				DefaultProvider: "provider2",
				Enabled:         true,
				Settings: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  2000,
				},
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent guildcontext.AgentClient) {
				assert.Equal(t, "specialist-agent", agent.GetID())
				contextAgent, ok := agent.(*ContextAwareAgent)
				require.True(t, ok)
				assert.Equal(t, "specialist", contextAgent.AgentType)
				assert.Equal(t, "provider2", contextAgent.defaultProvider)
				// Check metadata was added
				assert.Equal(t, 0.7, contextAgent.GetMetadata("temperature"))
				assert.Equal(t, 2000, contextAgent.GetMetadata("max_tokens"))
			},
		},
		{
			name: "disabled agent creation",
			agentConfig: AgentConfig{
				ID:           "disabled-agent",
				Name:         "Disabled Agent",
				Type:         "worker",
				Capabilities: []string{"general"},
				Enabled:      false,
			},
			expectError:   true,
			errorContains: "is disabled",
		},
		{
			name: "unknown agent type",
			agentConfig: AgentConfig{
				ID:           "unknown-agent",
				Name:         "Unknown Agent",
				Type:         "unknown-type",
				Capabilities: []string{"general"},
				Enabled:      true,
			},
			expectError:   true,
			errorContains: "unknown agent type",
		},
		{
			name: "empty type defaults to worker",
			agentConfig: AgentConfig{
				ID:           "default-type-agent",
				Name:         "Default Type Agent",
				Type:         "",
				Capabilities: []string{"general"},
				Enabled:      true,
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent guildcontext.AgentClient) {
				contextAgent, ok := agent.(*ContextAwareAgent)
				require.True(t, ok)
				assert.Equal(t, "worker", contextAgent.AgentType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			factory := newContextAgentFactory()

			agent, err := factory.CreateAgent(ctx, tt.agentConfig)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				if tt.validateAgent != nil {
					tt.validateAgent(t, agent)
				}
			}
		})
	}
}

// Test parseAgentConfig
func TestParseAgentConfig(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		configMap map[string]interface{}
		expected  AgentConfig
	}{
		{
			name:      "parse basic config",
			agentName: "test-agent",
			configMap: map[string]interface{}{
				"type":    "worker",
				"name":    "Test Agent",
				"enabled": true,
			},
			expected: AgentConfig{
				ID:       "test-agent",
				Name:     "Test Agent",
				Type:     "worker",
				Enabled:  true,
				Settings: make(map[string]interface{}),
			},
		},
		{
			name:      "parse config with capabilities",
			agentName: "cap-agent",
			configMap: map[string]interface{}{
				"type":         "specialist",
				"capabilities": []interface{}{"code", "analysis"},
				"enabled":      true,
			},
			expected: AgentConfig{
				ID:           "cap-agent",
				Name:         "cap-agent",
				Type:         "specialist",
				Enabled:      true,
				Capabilities: []string{"code", "analysis"},
				Settings:     make(map[string]interface{}),
			},
		},
		{
			name:      "parse config with single capability string",
			agentName: "single-cap",
			configMap: map[string]interface{}{
				"capabilities": "general",
			},
			expected: AgentConfig{
				ID:           "single-cap",
				Name:         "single-cap",
				Type:         "worker",
				Enabled:      true,
				Capabilities: []string{"general"},
				Settings:     make(map[string]interface{}),
			},
		},
		{
			name:      "parse config with all fields",
			agentName: "full-agent",
			configMap: map[string]interface{}{
				"type":             "manager",
				"name":             "Full Agent",
				"enabled":          false,
				"default_provider": "openai",
				"system_prompt":    "You are a helpful assistant",
				"capabilities":     []interface{}{"planning", "coordination"},
				"settings": map[string]interface{}{
					"temperature": 0.5,
					"model":       "gpt-4",
				},
			},
			expected: AgentConfig{
				ID:              "full-agent",
				Name:            "Full Agent",
				Type:            "manager",
				Enabled:         false,
				DefaultProvider: "openai",
				SystemPrompt:    "You are a helpful assistant",
				Capabilities:    []string{"planning", "coordination"},
				Settings: map[string]interface{}{
					"temperature": 0.5,
					"model":       "gpt-4",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAgentConfig(tt.agentName, tt.configMap)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Enabled, result.Enabled)
			assert.Equal(t, tt.expected.DefaultProvider, result.DefaultProvider)
			assert.Equal(t, tt.expected.SystemPrompt, result.SystemPrompt)
			assert.Equal(t, tt.expected.Capabilities, result.Capabilities)
			assert.Equal(t, tt.expected.Settings, result.Settings)
		})
	}
}

// Test GetDefaultAgentConfigs
func TestGetDefaultAgentConfigs(t *testing.T) {
	configs := GetDefaultAgentConfigs()

	assert.NotEmpty(t, configs)

	// Check for expected default agents
	expectedAgents := []string{"worker", "coding-agent", "analysis-agent", "manager"}

	for _, expectedID := range expectedAgents {
		config, exists := configs[expectedID]
		assert.True(t, exists, "Expected default agent %s not found", expectedID)
		assert.NotEmpty(t, config.Name)
		assert.NotEmpty(t, config.Type)
		assert.True(t, config.Enabled)
		assert.NotEmpty(t, config.Capabilities)
	}

	// Verify specific agent configurations
	workerConfig := configs["worker"]
	assert.Equal(t, "default-worker", workerConfig.ID)
	assert.Equal(t, "worker", workerConfig.Type)
	assert.Contains(t, workerConfig.Capabilities, "general")

	codingConfig := configs["coding-agent"]
	assert.Equal(t, "specialist", codingConfig.Type)
	assert.Contains(t, codingConfig.Capabilities, "coding")
	assert.NotEmpty(t, codingConfig.SystemPrompt)

	managerConfig := configs["manager"]
	assert.Equal(t, "manager", managerConfig.Type)
	assert.Contains(t, managerConfig.Capabilities, "coordination")
}

// Test DefaultContextAgentFactory
func TestDefaultContextAgentFactory(t *testing.T) {
	factory := DefaultContextAgentFactory()

	assert.NotNil(t, factory)
	// Just check that it's not nil
	// The concrete type assertion is not needed
}