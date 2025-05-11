package vector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIEmbedder(t *testing.T) {
	t.Run("NewOpenAIEmbedder_MissingAPIKey", func(t *testing.T) {
		// Test that an error is returned when API key is missing
		embedder, err := NewOpenAIEmbedder("", "")
		assert.Error(t, err)
		assert.Nil(t, embedder)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("NewOpenAIEmbedder_DefaultModel", func(t *testing.T) {
		// Test that the default model is used when no model is provided
		embedder, err := NewOpenAIEmbedder("dummy-api-key", "")
		require.NoError(t, err)
		assert.NotNil(t, embedder)
		assert.Equal(t, "text-embedding-ada-002", embedder.model)
	})

	t.Run("NewOpenAIEmbedder_CustomModel", func(t *testing.T) {
		// Test that the provided model is used
		customModel := "custom-model"
		embedder, err := NewOpenAIEmbedder("dummy-api-key", customModel)
		require.NoError(t, err)
		assert.NotNil(t, embedder)
		assert.Equal(t, customModel, embedder.model)
	})
}

// Note: To test the actual Embed function, we would need to mock the OpenAI API.
// Since the go-openai library doesn't provide a simple way to mock the client,
// we'll skip testing the actual API calls in this test file.
// In a production environment, you might want to use a library that supports
// proper mocking or create a thin wrapper around the OpenAI client that can
// be easily mocked.