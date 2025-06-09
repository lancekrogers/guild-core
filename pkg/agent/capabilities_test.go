package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test SetCapabilities and GetCapabilities
func TestWorkerAgent_Capabilities(t *testing.T) {
	agent := &WorkerAgent{
		ID:           "test-agent",
		Name:         "Test Agent",
		capabilities: []string{}, // Start with empty capabilities
	}

	// Test initial empty capabilities
	caps := agent.GetCapabilities()
	assert.Empty(t, caps)

	// Test setting capabilities
	newCaps := []string{"file-operations", "web-requests", "data-analysis"}
	agent.SetCapabilities(newCaps)

	// Test getting capabilities after setting
	retrievedCaps := agent.GetCapabilities()
	assert.Equal(t, newCaps, retrievedCaps)
	assert.Len(t, retrievedCaps, 3)
	assert.Contains(t, retrievedCaps, "file-operations")
	assert.Contains(t, retrievedCaps, "web-requests")
	assert.Contains(t, retrievedCaps, "data-analysis")

	// Test setting empty capabilities
	agent.SetCapabilities([]string{})
	emptyRetrieved := agent.GetCapabilities()
	assert.Empty(t, emptyRetrieved)

	// Test setting nil capabilities
	agent.SetCapabilities(nil)
	nilRetrieved := agent.GetCapabilities()
	assert.Nil(t, nilRetrieved)
}

// Test capabilities with different data types and edge cases
func TestWorkerAgent_CapabilitiesEdgeCases(t *testing.T) {
	agent := &WorkerAgent{
		ID:   "test-agent",
		Name: "Test Agent",
	}

	tests := []struct {
		name         string
		capabilities []string
		expected     []string
	}{
		{
			name:         "single capability",
			capabilities: []string{"single-cap"},
			expected:     []string{"single-cap"},
		},
		{
			name:         "duplicate capabilities",
			capabilities: []string{"dup", "unique", "dup"},
			expected:     []string{"dup", "unique", "dup"}, // No deduplication expected
		},
		{
			name:         "empty string capability",
			capabilities: []string{"valid", "", "also-valid"},
			expected:     []string{"valid", "", "also-valid"},
		},
		{
			name:         "very long capability name",
			capabilities: []string{"very-long-capability-name-that-might-test-string-handling"},
			expected:     []string{"very-long-capability-name-that-might-test-string-handling"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent.SetCapabilities(tt.capabilities)
			result := agent.GetCapabilities()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test capabilities don't affect other agent functionality
func TestWorkerAgent_CapabilitiesIndependence(t *testing.T) {
	agent := &WorkerAgent{
		ID:   "test-agent",
		Name: "Test Agent",
	}

	// Set capabilities
	caps := []string{"test-capability"}
	agent.SetCapabilities(caps)

	// Verify other methods still work correctly
	assert.Equal(t, "test-agent", agent.GetID())
	assert.Equal(t, "Test Agent", agent.GetName())

	// Verify capabilities are preserved
	assert.Equal(t, caps, agent.GetCapabilities())
}