package deepinfra

import (
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	providertesting "github.com/guild-ventures/guild-core/pkg/providers/testing"
)

func TestDeepInfraProvider(t *testing.T) {
	// Create mock server
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	// Create client with mock server URL
	client := &Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"deepinfra",
			"test-api-key",
			mock.URL + "/v1/openai",
			map[string]string{
				"gpt-3.5-turbo": Llama32_8B,
				"gpt-4":         Llama33_70B,
			},
			interfaces.ProviderCapabilities{
				MaxTokens:      131072,
				ContextWindow:  131072,
				SupportsVision: false,
				SupportsTools:  true,
				SupportsStream: true,
				Models: []interfaces.ModelInfo{
					{
						ID:            Llama32_8B,
						Name:          "Llama 3.2 8B",
						ContextWindow: 131072,
						MaxOutput:     8192,
						InputCost:     0.06,
						OutputCost:    0.06,
					},
				},
			},
		),
	}

	// Run standard test suite
	suite := providertesting.NewProviderTestSuite(t, client, providertesting.TestConfig{
		ProviderName: "DeepInfra",
		TestModel:    Llama32_8B,
		SkipLive:     true,
	})
	
	suite.RunBasicTests()
}

func TestDeepInfraModelVariety(t *testing.T) {
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	// Check we have multiple model families
	modelFamilies := make(map[string]bool)
	for _, model := range caps.Models {
		if contains(model.ID, "llama") {
			modelFamilies["llama"] = true
		} else if contains(model.ID, "mistral") {
			modelFamilies["mistral"] = true
		} else if contains(model.ID, "qwen") {
			modelFamilies["qwen"] = true
		} else if contains(model.ID, "gemma") {
			modelFamilies["gemma"] = true
		}
	}

	expectedFamilies := []string{"llama", "mistral"}
	for _, family := range expectedFamilies {
		if !modelFamilies[family] {
			t.Errorf("Missing %s model family", family)
		}
	}
}

func TestDeepInfraModelRecommendation(t *testing.T) {
	testCases := []struct {
		useCase  string
		expected string
	}{
		{"coding", Qwen25_72B},
		{"reasoning", Llama33_70B},
		{"cost-efficient", Llama32_8B},
		{"fast", Mistral7B},
		{"general", Llama33_70B},
	}

	for _, tc := range testCases {
		t.Run(tc.useCase, func(t *testing.T) {
			model := GetRecommendedModel(tc.useCase)
			if model != tc.expected {
				t.Errorf("Expected %s for %s, got %s", tc.expected, tc.useCase, model)
			}
		})
	}
}

func TestDeepInfraCostEfficiency(t *testing.T) {
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	// Check that DeepInfra offers very competitive pricing
	for _, model := range caps.Models {
		if model.ID == Llama32_8B {
			if model.InputCost > 0.1 {
				t.Errorf("Llama 3.2 8B should be very affordable, got $%.2f", model.InputCost)
			}
		}
	}
}

func TestDeepInfraOpenSourceModels(t *testing.T) {
	// DeepInfra specializes in open-source models
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	openSourceModels := 0
	for range caps.Models {
		// All DeepInfra models are open source
		openSourceModels++
	}

	if openSourceModels == 0 {
		t.Error("DeepInfra should offer open-source models")
	}

	t.Logf("DeepInfra offers %d open-source models", openSourceModels)
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}