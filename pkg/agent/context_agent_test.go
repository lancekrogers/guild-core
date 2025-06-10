package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test newContextAwareAgent
func TestNewContextAwareAgent(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		agentName     string
		agentType     string
		capabilities  []string
		validateAgent func(t *testing.T, agent *ContextAwareAgent)
	}{
		{
			name:         "successful creation with basic fields",
			id:           "test-agent-1",
			agentName:    "Test Agent 1",
			agentType:    "worker",
			capabilities: []string{"code", "analysis", "translation"},
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, "test-agent-1", agent.ID)
				assert.Equal(t, "Test Agent 1", agent.Name)
				assert.Equal(t, "worker", agent.AgentType)
				assert.Equal(t, 3, len(agent.Capabilities))
				assert.Equal(t, "idle", agent.status.State)
				assert.NotNil(t, agent.status.Metadata)
			},
		},
		{
			name:         "creation with manager type",
			id:           "manager-agent",
			agentName:    "Manager Agent",
			agentType:    "manager",
			capabilities: []string{"coordination", "planning"},
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, "manager-agent", agent.ID)
				assert.Equal(t, "Manager Agent", agent.Name)
				assert.Equal(t, "manager", agent.AgentType)
				assert.Equal(t, 2, len(agent.Capabilities))
			},
		},
		{
			name:         "creation with empty capabilities",
			id:           "test-agent-3",
			agentName:    "Test Agent 3",
			agentType:    "specialist",
			capabilities: []string{},
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, 0, len(agent.Capabilities))
				assert.Equal(t, "specialist", agent.AgentType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := newContextAwareAgent(tt.id, tt.agentName, tt.agentType, tt.capabilities)
			require.NotNil(t, agent)
			
			if tt.validateAgent != nil {
				tt.validateAgent(t, agent)
			}
		})
	}
}

// Test Execute method
func TestContextAwareAgent_Execute(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() *ContextAwareAgent
		request     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "execution updates status",
			setupAgent: func() *ContextAwareAgent {
				agent := newContextAwareAgent("test-1", "Test Agent", "worker", []string{"general"})
				return agent
			},
			request:     "test request",
			expectError: true, // Will error because no provider is configured
			errorMsg:    "failed to select provider",
		},
		{
			name: "execution with system prompt",
			setupAgent: func() *ContextAwareAgent {
				agent := newContextAwareAgent("test-2", "Test Agent", "manager", []string{"planning"})
				agent.systemPrompt = "You are a planning agent"
				return agent
			},
			request:     "create a plan",
			expectError: true, // Will error because no provider is configured
			errorMsg:    "failed to select provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			agent := tt.setupAgent()
			
			// Check initial status
			assert.Equal(t, "idle", agent.status.State)
			assert.Equal(t, int64(0), agent.taskCount)
			
			// Execute
			result, err := agent.Execute(ctx, tt.request)
			
			// Check error
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Equal(t, int64(1), agent.errorCount)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.Equal(t, int64(1), agent.status.SuccessCount)
			}
			
			// Check status was updated
			assert.Equal(t, int64(1), agent.taskCount)
			assert.NotEqual(t, time.Time{}, agent.status.LastActive)
		})
	}
}

// Test GetID method
func TestContextAwareAgent_GetID(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", []string{"general"})
	assert.Equal(t, "test-id", agent.GetID())
}

// Test GetName method
func TestContextAwareAgent_GetName(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent Name", "worker", []string{"general"})
	assert.Equal(t, "Test Agent Name", agent.GetName())
}

// Test GetCapabilities method
func TestContextAwareAgent_GetCapabilities(t *testing.T) {
	capabilities := []string{"code", "analysis", "documentation"}
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", capabilities)
	
	agentCaps := agent.GetCapabilities()
	assert.Equal(t, len(capabilities), len(agentCaps))
	for i, cap := range capabilities {
		assert.Equal(t, cap, agentCaps[i])
	}
}

// Test GetType method
func TestContextAwareAgent_GetType(t *testing.T) {
	tests := []struct {
		agentType string
		expected  string
	}{
		{"worker", "worker"},
		{"manager", "manager"},
		{"specialist", "specialist"},
		{"", ""}, // Empty type
	}
	
	for _, tt := range tests {
		t.Run("type_"+tt.agentType, func(t *testing.T) {
			agent := newContextAwareAgent("test-id", "Test Agent", tt.agentType, []string{})
			assert.Equal(t, tt.expected, agent.GetType())
		})
	}
}

// Test GetStatus method
func TestContextAwareAgent_GetStatus(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", []string{"general"})
	
	// Initial status
	status := agent.GetStatus()
	
	assert.Equal(t, "idle", status.State)
	assert.Equal(t, "", status.CurrentTask)
	assert.Equal(t, int64(0), status.TaskCount)
	assert.Equal(t, int64(0), status.SuccessCount)
	assert.Equal(t, int64(0), status.ErrorCount)
	
	// Update some status fields
	agent.status.State = "busy"
	agent.status.CurrentTask = "processing request"
	agent.status.TaskCount = 5
	agent.status.SuccessCount = 3
	agent.status.ErrorCount = 2
	
	// Check updated status
	status = agent.GetStatus()
	
	assert.Equal(t, "busy", status.State)
	assert.Equal(t, "processing request", status.CurrentTask)
	assert.Equal(t, int64(5), status.TaskCount)
	assert.Equal(t, int64(3), status.SuccessCount)
	assert.Equal(t, int64(2), status.ErrorCount)
}

// Test AddMetadata method
func TestContextAwareAgent_AddMetadata(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", []string{"general"})
	
	// Add metadata
	agent.AddMetadata("custom_field", "custom_value")
	agent.AddMetadata("metric", 42)
	
	// Check metadata was updated
	assert.Equal(t, "custom_value", agent.status.Metadata["custom_field"])
	assert.Equal(t, 42, agent.status.Metadata["metric"])
}

// Test AddMetadata and GetMetadata methods
func TestContextAwareAgent_Metadata(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", []string{"general"})
	
	// Add metadata
	agent.AddMetadata("key1", "value1")
	agent.AddMetadata("key2", 123)
	agent.AddMetadata("key3", true)
	
	// Get metadata
	assert.Equal(t, "value1", agent.GetMetadata("key1"))
	assert.Equal(t, 123, agent.GetMetadata("key2"))
	assert.Equal(t, true, agent.GetMetadata("key3"))
	assert.Nil(t, agent.GetMetadata("nonexistent"))
}

// Test additional methods
func TestContextAwareAgent_AdditionalMethods(t *testing.T) {
	agent := newContextAwareAgent("test-id", "Test Agent", "worker", []string{"general"})
	
	// Test SetDefaultProvider
	agent.SetDefaultProvider("openai")
	assert.Equal(t, "openai", agent.defaultProvider)
	
	// Test SetSystemPrompt
	agent.SetSystemPrompt("You are a helpful assistant")
	assert.Equal(t, "You are a helpful assistant", agent.systemPrompt)
	
	// Test UpdateCapabilities
	newCaps := []string{"coding", "testing", "documentation"}
	agent.UpdateCapabilities(newCaps)
	assert.Equal(t, newCaps, agent.Capabilities)
	
	// Test GetAgentType
	assert.Equal(t, "worker", agent.GetAgentType())
	
	// Test Reset
	agent.Reset()
	assert.Equal(t, "idle", agent.status.State)
	assert.Equal(t, "", agent.status.CurrentTask)
	assert.Equal(t, int64(0), agent.taskCount)
	assert.Equal(t, int64(0), agent.errorCount)
}