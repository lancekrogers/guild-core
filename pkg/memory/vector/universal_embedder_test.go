// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package vector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/lancekrogers/guild/pkg/providers/mock"
)

func TestUniversalEmbedder_NewUniversalEmbedder(t *testing.T) {
	provider := &mock.Provider{}

	tests := []struct {
		name     string
		opts     []EmbedderOption
		expected UniversalEmbedder
	}{
		{
			name: "default configuration",
			opts: nil,
			expected: UniversalEmbedder{
				strategy: StrategyAuto,
			},
		},
		{
			name: "with dedicated strategy",
			opts: []EmbedderOption{
				WithStrategy(StrategyDedicated),
			},
			expected: UniversalEmbedder{
				strategy: StrategyDedicated,
			},
		},
		{
			name: "with specific model",
			opts: []EmbedderOption{
				WithModel("nomic-embed-text"),
			},
			expected: UniversalEmbedder{
				model:    "nomic-embed-text",
				strategy: StrategyAuto,
			},
		},
		{
			name: "with config",
			opts: []EmbedderOption{
				WithConfig(&UniversalEmbedderConfig{
					PreferredModels:   []string{"model1", "model2"},
					DimensionHandling: "normalize",
					TargetDimension:   768,
				}),
			},
			expected: UniversalEmbedder{
				strategy: StrategyAuto,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := NewUniversalEmbedder(provider, tt.opts...)

			assert.Equal(t, tt.expected.strategy, embedder.strategy)
			if tt.expected.model != "" {
				assert.Equal(t, tt.expected.model, embedder.model)
			}
		})
	}
}

func TestUniversalEmbedder_Embed(t *testing.T) {
	ctx := context.Background()

	// Create mock providers for tests
	mockProvider1, err := mock.NewProvider()
	require.NoError(t, err)
	mockProvider2, err := mock.NewProvider()
	require.NoError(t, err)

	tests := []struct {
		name        string
		text        string
		strategy    EmbeddingStrategy
		provider    interfaces.AIProvider
		expectError bool
		expectNil   bool
	}{
		{
			name:        "successful dedicated embedding",
			text:        "test text",
			strategy:    StrategyDedicated,
			provider:    mockProvider1,
			expectError: false,
		},
		{
			name:      "graceful degradation with none strategy",
			text:      "test text",
			strategy:  StrategyNone,
			provider:  mockProvider2,
			expectNil: true,
		},
		{
			name:        "nil provider graceful degradation",
			text:        "test text",
			strategy:    StrategyAuto,
			provider:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := NewUniversalEmbedder(tt.provider, WithStrategy(tt.strategy))

			result, err := embedder.Embed(ctx, tt.text)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			if tt.expectNil {
				assert.NoError(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result) // Should have embeddings
		})
	}
}

func TestUniversalEmbedder_GetEmbeddings(t *testing.T) {
	ctx := context.Background()

	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	embedder := NewUniversalEmbedder(mockProvider, WithStrategy(StrategyDedicated))

	texts := []string{"text1", "text2", "text3"}
	results, err := embedder.GetEmbeddings(ctx, texts)

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Each result should have embeddings
	for _, result := range results {
		assert.NotNil(t, result)
		assert.NotEmpty(t, result)
	}
}

func TestNoOpEmbedder(t *testing.T) {
	ctx := context.Background()
	embedder := NewNoOpEmbedder(768)

	// Test single embed
	result, err := embedder.Embed(ctx, "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 768)

	// Test GetEmbedding
	result2, err := embedder.GetEmbedding(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, result, result2) // Should be deterministic

	// Test GetEmbeddings
	results, err := embedder.GetEmbeddings(ctx, []string{"test1", "test2"})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotEqual(t, results[0], results[1]) // Different texts should give different embeddings
}

func TestConvertToFloat32(t *testing.T) {
	input := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	expected := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	result := convertToFloat32(input)

	assert.Equal(t, expected, result)
}

func TestParseVectorString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []float32
		expectError bool
	}{
		{
			name:     "valid vector string",
			input:    "0.1, 0.2, 0.3",
			expected: []float32{0.1, 0.2, 0.3},
		},
		{
			name:     "vector with extra spaces",
			input:    " 0.1 , 0.2 , 0.3 ",
			expected: []float32{0.1, 0.2, 0.3},
		},
		{
			name:        "invalid number",
			input:       "0.1, abc, 0.3",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "only commas",
			input:       ",,,",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVectorString(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
