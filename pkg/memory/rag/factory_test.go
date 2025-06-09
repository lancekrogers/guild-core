package rag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// MockEmbedder implements vector.Embedder for testing
type MockEmbedder struct {
	embedFunc func(ctx context.Context, text string) ([]float32, error)
	closeFunc func() error
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbedder) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewFactory(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name      string
		embedder  vector.Embedder
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name:     "Successful factory creation",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test_collection",
				ChunkSize:      500,
				ChunkOverlap:   50,
				ChunkStrategy:  "paragraph",
			},
			wantError: false,
		},
		{
			name:     "Nil embedder",
			embedder: nil,
			config: Config{
				CollectionName: "test_collection",
			},
			wantError: true,
			errorMsg:  "embedder is required",
		},
		{
			name:     "Empty config uses defaults",
			embedder: &MockEmbedder{},
			config:   Config{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := newFactory(ctx, tt.embedder, tt.config)
			
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, factory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, factory)
				assert.NotNil(t, factory.retriever)
				assert.Equal(t, tt.embedder, factory.embedder)
			}
		})
	}
}

func TestFactory_GetRetriever(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_collection",
		ChunkSize:      500,
		ChunkOverlap:   50,
	}
	
	factory, err := newFactory(ctx, embedder, config)
	require.NoError(t, err)
	require.NotNil(t, factory)
	
	retriever := factory.GetRetriever()
	assert.NotNil(t, retriever)
	assert.Equal(t, factory.retriever, retriever)
}

func TestFactory_GetEmbedder(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_collection",
		ChunkSize:      500,
		ChunkOverlap:   50,
	}
	
	factory, err := newFactory(ctx, embedder, config)
	require.NoError(t, err)
	require.NotNil(t, factory)
	
	retrievedEmbedder := factory.GetEmbedder()
	assert.NotNil(t, retrievedEmbedder)
	assert.Equal(t, embedder, retrievedEmbedder)
}

func TestFactory_Close(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name        string
		setupFunc   func() *Factory
		expectError bool
	}{
		{
			name: "Close with retriever",
			setupFunc: func() *Factory {
				embedder := &MockEmbedder{}
				config := Config{
					CollectionName: "test_collection",
				}
				factory, _ := newFactory(ctx, embedder, config)
				return factory
			},
			expectError: false,
		},
		{
			name: "Close with nil retriever",
			setupFunc: func() *Factory {
				return &Factory{
					retriever: nil,
					embedder:  &MockEmbedder{},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := tt.setupFunc()
			err := factory.Close()
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultFactoryFactory(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name      string
		embedder  vector.Embedder
		config    Config
		wantError bool
	}{
		{
			name:     "Successful factory creation",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test_factory",
				ChunkSize:      750,
				ChunkOverlap:   100,
				ChunkStrategy:  "sentence",
			},
			wantError: false,
		},
		{
			name:     "Default config",
			embedder: &MockEmbedder{},
			config:   Config{},
			wantError: false,
		},
		{
			name:      "Nil embedder",
			embedder:  nil,
			config:    Config{CollectionName: "test_factory"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := DefaultFactoryFactory(ctx, tt.embedder, tt.config)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, factory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, factory)
				
				// Verify it implements the interface
				_, ok := factory.(FactoryInterface)
				assert.True(t, ok)
				
				// Verify components are accessible
				assert.NotNil(t, factory.GetRetriever())
				assert.NotNil(t, factory.GetEmbedder())
				
				// Clean up
				err = factory.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestFactory_InterfaceCompliance(t *testing.T) {
	// This test ensures that Factory implements FactoryInterface
	var _ FactoryInterface = (*Factory)(nil)
	
	// Also test with actual instance
	ctx := context.Background()
	embedder := &MockEmbedder{}
	config := Config{
		CollectionName: "test_interface",
	}
	
	factory, err := newFactory(ctx, embedder, config)
	require.NoError(t, err)
	
	// Test all interface methods
	var fi FactoryInterface = factory
	assert.NotNil(t, fi.GetRetriever())
	assert.NotNil(t, fi.GetEmbedder())
	assert.NoError(t, fi.Close())
}