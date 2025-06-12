package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test GetAgentCapabilities
func TestAgentRouter_GetAgentCapabilities(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	// Test successful retrieval
	agents, err := router.GetAgentCapabilities(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, agents)
	assert.Greater(t, len(agents), 0, "Should return at least one agent")

	// Verify agent properties
	for _, agent := range agents {
		assert.NotEmpty(t, agent.Name, "Agent should have a name")
		assert.NotEmpty(t, agent.Role, "Agent should have a role")
		assert.NotEmpty(t, agent.Provider, "Agent should have a provider")
		assert.NotEmpty(t, agent.Model, "Agent should have a model")
		assert.Greater(t, agent.CostMagnitude, 0, "Agent should have a cost magnitude")
		assert.Greater(t, agent.ContextWindow, 0, "Agent should have a context window")
		assert.NotEmpty(t, agent.Specializations, "Agent should have specializations")
		assert.NotEmpty(t, agent.Tools, "Agent should have tools")
		assert.GreaterOrEqual(t, agent.SuccessRate, 0.0, "Success rate should be non-negative")
		assert.LessOrEqual(t, agent.SuccessRate, 100.0, "Success rate should not exceed 100")
	}
}

// Test specific agent types are present
func TestAgentRouter_SpecificAgentTypes(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	agents, err := router.GetAgentCapabilities(ctx)
	require.NoError(t, err)

	// Check for expected agent types
	expectedRoles := map[string]bool{
		"Backend Developer":  false,
		"Frontend Developer": false,
		"DevOps Engineer":    false,
		"Security Analyst":   false,
	}

	for _, agent := range agents {
		if _, exists := expectedRoles[agent.Role]; exists {
			expectedRoles[agent.Role] = true
		}
	}

	// Verify all expected roles were found
	for role, found := range expectedRoles {
		assert.True(t, found, "Expected to find agent with role: %s", role)
	}
}

// Test agent capabilities structure
func TestAgentRouter_AgentCapabilitiesStructure(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	agents, err := router.GetAgentCapabilities(ctx)
	require.NoError(t, err)
	require.Greater(t, len(agents), 0)

	// Test first agent in detail
	agent := agents[0]

	// Verify EnhancedAgentInfo fields
	assert.NotEmpty(t, agent.Name)
	assert.NotEmpty(t, agent.Role)
	assert.NotEmpty(t, agent.Provider)
	assert.NotEmpty(t, agent.Model)
	assert.Greater(t, agent.CostMagnitude, 0)
	assert.Greater(t, agent.ContextWindow, 0)
	assert.NotEmpty(t, agent.Specializations)
	assert.NotEmpty(t, agent.Tools)
	assert.GreaterOrEqual(t, agent.SuccessRate, 0.0)

	// Check enhanced fields
	assert.GreaterOrEqual(t, agent.TokenCost, 0.0)
	assert.GreaterOrEqual(t, agent.QualityScore, 0.0)
	assert.LessOrEqual(t, agent.QualityScore, 10.0)
	assert.NotEmpty(t, agent.AvgCompletionTime)
	assert.GreaterOrEqual(t, agent.RecentTaskCount, 0)
	assert.GreaterOrEqual(t, agent.CurrentTasks, 0)
	assert.NotEmpty(t, agent.AvailabilityStatus)
}

// Test concurrent access
func TestAgentRouter_ConcurrentAccess(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	// Run concurrent requests
	numGoroutines := 10
	results := make(chan []EnhancedAgentInfo, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			agents, err := router.GetAgentCapabilities(ctx)
			if err != nil {
				errors <- err
			} else {
				results <- agents
			}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("Concurrent access failed: %v", err)
		case agents := <-results:
			assert.NotNil(t, agents)
			assert.Greater(t, len(agents), 0)
		}
	}
}

// Test agent specializations
func TestAgentRouter_AgentSpecializations(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	agents, err := router.GetAgentCapabilities(ctx)
	require.NoError(t, err)

	// Check backend developer specializations
	for _, agent := range agents {
		if agent.Role == "Backend Developer" {
			expectedSpecs := []string{"Go", "APIs", "databases", "microservices"}
			for _, expected := range expectedSpecs {
				found := false
				for _, spec := range agent.Specializations {
					if spec == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "Backend Developer should have specialization: %s", expected)
			}
		}
	}
}

// Test cost magnitude ranges
func TestAgentRouter_CostMagnitudes(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	agents, err := router.GetAgentCapabilities(ctx)
	require.NoError(t, err)

	// Verify cost magnitudes are in reasonable range
	for _, agent := range agents {
		assert.GreaterOrEqual(t, agent.CostMagnitude, 1,
			"Cost magnitude should be at least 1 for agent %s", agent.Name)
		assert.LessOrEqual(t, agent.CostMagnitude, 5,
			"Cost magnitude should not exceed 5 for agent %s", agent.Name)
	}
}

// Test context window sizes
func TestAgentRouter_ContextWindows(t *testing.T) {
	router := &AgentRouter{}
	ctx := context.Background()

	agents, err := router.GetAgentCapabilities(ctx)
	require.NoError(t, err)

	// Verify context windows are reasonable
	for _, agent := range agents {
		assert.GreaterOrEqual(t, agent.ContextWindow, 4000,
			"Context window should be at least 4000 for agent %s", agent.Name)
		assert.LessOrEqual(t, agent.ContextWindow, 1000000,
			"Context window should not exceed 1M for agent %s", agent.Name)
	}
}
