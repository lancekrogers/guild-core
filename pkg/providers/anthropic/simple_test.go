package anthropic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientSimple(t *testing.T) {
	// Test with explicit API key
	client := NewClient("test-key")
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)

	// Test with environment variable
	os.Setenv("ANTHROPIC_API_KEY", "env-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	client2 := NewClient("")
	assert.NotNil(t, client2)
}

func TestGetRecommendedModelSimple(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "claude-4-sonnet"},
		{"gpt-4", "claude-4-sonnet"},
		{"gpt-3.5-turbo", "claude-4-sonnet"},
		{"unknown", "claude-4-sonnet"},
	}

	for _, tt := range tests {
		result := GetRecommendedModel(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestParseErrorSimple(t *testing.T) {
	client := &Client{}

	// Test JSON error response
	err := client.parseError(400, []byte(`{"error": {"type": "invalid_request", "message": "Bad request"}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad request")

	// Test non-JSON error
	err = client.parseError(500, []byte("Server error"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Server error")
}

func TestCapabilitiesSimple(t *testing.T) {
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	assert.Equal(t, 200000, caps.MaxTokens)
	assert.Equal(t, 200000, caps.ContextWindow)
	assert.True(t, caps.SupportsVision)
	assert.True(t, caps.SupportsTools)
	assert.True(t, caps.SupportsStream)
	assert.Greater(t, len(caps.Models), 0)
}
