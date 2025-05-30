package vector

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	
	tests := []struct {
		name        string
		text        string
		strategy    EmbeddingStrategy
		provider    interfaces.AIProvider
		expectError bool
		expectNil   bool
	}{
		{
			name:     "successful dedicated embedding",
			text:     "test text",
			strategy: StrategyDedicated,
			provider: &mock.Provider{
				CreateEmbeddingFunc: func(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
					return &interfaces.EmbeddingResponse{
						Model: "test-model",
						Embeddings: []interfaces.Embedding{
							{
								Index:     0,
								Embedding: []float64{0.1, 0.2, 0.3},
							},
						},
					}, nil
				},
				GetCapabilitiesFunc: func() interfaces.ProviderCapabilities {
					return interfaces.ProviderCapabilities{
						SupportsEmbeddings: true,
					}
				},
			},
			expectError: false,
		},
		{
			name:        "empty text",
			text:        "",
			strategy:    StrategyAuto,
			provider:    &mock.Provider{},
			expectError: true,
		},
		{
			name:     "graceful degradation with none strategy",
			text:     "test text",
			strategy: StrategyNone,
			provider: &mock.Provider{},
			expectNil: true,
		},
		{
			name:     "provider without embedding support",
			text:     "test text",
			strategy: StrategyDedicated,
			provider: &mock.Provider{
				GetCapabilitiesFunc: func() interfaces.ProviderCapabilities {
					return interfaces.ProviderCapabilities{
						SupportsEmbeddings: false,
					}
				},
			},
			expectError: true,
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
			assert.Len(t, result, 3) // Based on mock response
		})
	}
}

func TestUniversalEmbedder_GetEmbeddings(t *testing.T) {
	ctx := context.Background()
	
	mockProvider := &mock.Provider{
		CreateEmbeddingFunc: func(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
			embeddings := make([]interfaces.Embedding, len(req.Input))
			for i := range req.Input {
				embeddings[i] = interfaces.Embedding{
					Index:     i,
					Embedding: []float64{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3},
				}
			}
			return &interfaces.EmbeddingResponse{
				Model:      "test-model",
				Embeddings: embeddings,
			}, nil
		},
		GetCapabilitiesFunc: func() interfaces.ProviderCapabilities {
			return interfaces.ProviderCapabilities{
				SupportsEmbeddings: true,
			}
		},
	}
	
	embedder := NewUniversalEmbedder(mockProvider, WithStrategy(StrategyDedicated))
	
	texts := []string{"text1", "text2", "text3"}
	results, err := embedder.GetEmbeddings(ctx, texts)
	
	require.NoError(t, err)
	assert.Len(t, results, 3)
	
	for i, result := range results {
		assert.Len(t, result, 3)
		assert.InDelta(t, float32(i)*0.1, result[0], 0.001)
		assert.InDelta(t, float32(i)*0.2, result[1], 0.001)
		assert.InDelta(t, float32(i)*0.3, result[2], 0.001)
	}
}

func TestNoOpEmbedder(t *testing.T) {
	ctx := context.Background()
	embedder := &NoOpEmbedder{}
	
	// Test single embed
	result, err := embedder.Embed(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)
	
	// Test GetEmbedding
	result, err = embedder.GetEmbedding(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)
	
	// Test GetEmbeddings
	results, err := embedder.GetEmbeddings(ctx, []string{"test1", "test2"})
	assert.NoError(t, err)
	assert.Nil(t, results)
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