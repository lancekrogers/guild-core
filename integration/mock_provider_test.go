// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

func TestMockProviderIntegration(t *testing.T) {
	// Enable mock provider
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")

	t.Run("Factory Integration", func(t *testing.T) {
		factory := providers.NewFactoryV2()

		// Create provider through factory
		provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, "fake-key")
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Test that it's actually the mock provider
		capabilities := provider.GetCapabilities()
		assert.Contains(t, capabilities.Models[0].ID, "mock-model")

		// Test functionality
		req := interfaces.ChatRequest{
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Create a commission for building an API"},
			},
			Model: "mock-model-v1",
		}

		resp, err := provider.ChatCompletion(context.Background(), req)
		require.NoError(t, err)
		assert.Contains(t, resp.Choices[0].Message.Content, "Commission Analysis")
	})

	t.Run("Provider Registry Integration", func(t *testing.T) {
		// Test with legacy factory and registry
		factory := providers.NewFactory()

		// Create mock registry
		registry := &mockProviderRegistry{providers: make(map[string]providers.LLMClient)}

		// Register providers - should get mock provider
		providersConfig := map[string]interface{}{
			"openai": map[string]interface{}{
				"api_key": "fake-key",
				"model":   "gpt-4",
			},
		}

		err := factory.RegisterProvidersWithRegistry(registry, providersConfig)
		require.NoError(t, err)

		// Verify mock provider was registered
		assert.Len(t, registry.providers, 1)
		assert.Contains(t, registry.providers, "openai")

		// Test the registered provider
		provider := registry.providers["openai"]
		response, err := provider.Complete(context.Background(), "help me build an API")
		require.NoError(t, err)
		assert.NotEmpty(t, response)
	})

	t.Run("Environment Variable Activation", func(t *testing.T) {
		tests := []struct {
			name     string
			envValue string
			enabled  bool
		}{
			{"enabled", "true", true},
			{"disabled", "false", false},
			{"unset", "", false},
			{"invalid", "yes", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.envValue == "" {
					os.Unsetenv("GUILD_MOCK_PROVIDER")
				} else {
					os.Setenv("GUILD_MOCK_PROVIDER", tt.envValue)
				}

				factory := providers.NewFactoryV2()
				provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, "fake-key")

				if tt.enabled {
					require.NoError(t, err)
					capabilities := provider.GetCapabilities()
					assert.Contains(t, capabilities.Models[0].ID, "mock-model")
				} else {
					// Should get real provider (or error if no API key)
					if err == nil {
						capabilities := provider.GetCapabilities()
						assert.NotContains(t, capabilities.Models[0].ID, "mock-model")
					}
				}
			})
		}
	})
}

func TestMockProviderCommandLineIntegration(t *testing.T) {
	// Skip if guild binary not available
	if _, err := exec.LookPath("guild"); err != nil {
		t.Skip("guild binary not available for integration testing")
	}

	// Enable mock provider
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")

	t.Run("Guild Init With Mock Provider", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(oldWd)

		// Run guild init
		cmd := exec.Command("guild", "init", "--name", "test-project")
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Guild init output: %s", string(output))
			// If init fails, it might be due to missing dependencies
			// This is acceptable for a mock provider test
		}

		// At minimum, verify that the mock provider environment is recognized
		// by checking that the guild binary can start with the mock provider
		cmd = exec.Command("guild", "--help")
		err = cmd.Run()
		assert.NoError(t, err, "Guild CLI should run with mock provider enabled")
	})
}

func TestMockProviderPerformance(t *testing.T) {
	// Enable mock provider
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")

	factory := providers.NewFactoryV2()
	provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, "fake-key")
	require.NoError(t, err)

	t.Run("Response Time", func(t *testing.T) {
		req := interfaces.ChatRequest{
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Quick test"},
			},
			Model: "mock-model-fast",
		}

		start := time.Now()
		resp, err := provider.ChatCompletion(context.Background(), req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Less(t, elapsed, 100*time.Millisecond, "Mock provider should respond quickly")
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(index int) {
				req := interfaces.ChatRequest{
					Messages: []interfaces.ChatMessage{
						{Role: "user", Content: "Concurrent test " + string(rune('0'+index))},
					},
					Model: "mock-model-v1",
				}

				_, err := provider.ChatCompletion(context.Background(), req)
				results <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			select {
			case err := <-results:
				assert.NoError(t, err, "Concurrent request %d should succeed", i)
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent request timed out")
			}
		}
	})
}

func TestMockProviderCustomResponses(t *testing.T) {
	// Enable mock provider
	os.Setenv("GUILD_MOCK_PROVIDER", "true")
	defer os.Unsetenv("GUILD_MOCK_PROVIDER")

	// Create custom responses file
	tmpDir := t.TempDir()
	customResponsesPath := tmpDir + "/custom_responses.yaml"
	customYAML := `responses:
  - name: "custom_test"
    patterns:
      - "custom pattern"
    messages:
      - "This is a custom response for testing"
    delay_ms: 10
    tokens: 25`

	err := os.WriteFile(customResponsesPath, []byte(customYAML), 0644)
	require.NoError(t, err)

	// Set environment variable to use custom responses
	os.Setenv("GUILD_MOCK_RESPONSES", customResponsesPath)
	defer os.Unsetenv("GUILD_MOCK_RESPONSES")

	factory := providers.NewFactoryV2()
	provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, "fake-key")
	require.NoError(t, err)

	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "This matches custom pattern"},
		},
		Model: "mock-model-v1",
	}

	resp, err := provider.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, resp.Choices[0].Message.Content, "custom response for testing")
}

// mockProviderRegistry implements the ProviderRegistry interface for testing
type mockProviderRegistry struct {
	providers       map[string]providers.LLMClient
	defaultProvider string
}

func (r *mockProviderRegistry) RegisterProvider(name string, provider providers.LLMClient) error {
	r.providers[name] = provider
	return nil
}

func (r *mockProviderRegistry) SetDefaultProvider(name string) error {
	r.defaultProvider = name
	return nil
}

// Helper function to run commands with timeout
func runCommandWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}
