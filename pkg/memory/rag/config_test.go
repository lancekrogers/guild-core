package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// Verify all default values
	assert.Equal(t, "rag_embeddings", config.CollectionName)
	assert.Equal(t, 1000, config.ChunkSize)
	assert.Equal(t, 200, config.ChunkOverlap)
	assert.Equal(t, "paragraph", config.ChunkStrategy)
	assert.Equal(t, 5, config.MaxResults)
	assert.False(t, config.UseCorpus)
	assert.Empty(t, config.VectorStorePath)
	assert.Empty(t, config.CorpusPath)
	assert.Equal(t, 1000, config.CorpusMaxSizeMB)
}

func TestDefaultRetrievalConfig(t *testing.T) {
	config := DefaultRetrievalConfig()
	
	// Verify all default values
	assert.Equal(t, 5, config.MaxResults)
	assert.Equal(t, float32(0.0), config.MinScore)
	assert.False(t, config.IncludeMetadata)
	assert.False(t, config.UseCorpus)
	assert.False(t, config.DisableVectorSearch)
	assert.Empty(t, config.Query)
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		verify func(t *testing.T, config Config)
	}{
		{
			name: "Custom configuration",
			config: Config{
				CollectionName:  "custom_collection",
				ChunkSize:       1500,
				ChunkOverlap:    150,
				ChunkStrategy:   "sentence",
				MaxResults:      20,
				VectorStorePath: "/path/to/vector/store",
				UseCorpus:       true,
				CorpusPath:      "/path/to/corpus",
				CorpusMaxSizeMB: 2000,
			},
			verify: func(t *testing.T, config Config) {
				assert.Equal(t, "custom_collection", config.CollectionName)
				assert.Equal(t, 1500, config.ChunkSize)
				assert.Equal(t, 150, config.ChunkOverlap)
				assert.Equal(t, "sentence", config.ChunkStrategy)
				assert.Equal(t, 20, config.MaxResults)
				assert.Equal(t, "/path/to/vector/store", config.VectorStorePath)
				assert.True(t, config.UseCorpus)
				assert.Equal(t, "/path/to/corpus", config.CorpusPath)
				assert.Equal(t, 2000, config.CorpusMaxSizeMB)
			},
		},
		{
			name: "Zero values",
			config: Config{
				CollectionName:  "",
				ChunkSize:       0,
				ChunkOverlap:    0,
				ChunkStrategy:   "",
				MaxResults:      0,
				VectorStorePath: "",
				UseCorpus:       false,
				CorpusPath:      "",
				CorpusMaxSizeMB: 0,
			},
			verify: func(t *testing.T, config Config) {
				assert.Empty(t, config.CollectionName)
				assert.Equal(t, 0, config.ChunkSize)
				assert.Equal(t, 0, config.ChunkOverlap)
				assert.Empty(t, config.ChunkStrategy)
				assert.Equal(t, 0, config.MaxResults)
				assert.Empty(t, config.VectorStorePath)
				assert.False(t, config.UseCorpus)
				assert.Empty(t, config.CorpusPath)
				assert.Equal(t, 0, config.CorpusMaxSizeMB)
			},
		},
		{
			name: "Negative values",
			config: Config{
				CollectionName:  "test_collection",
				ChunkSize:       -100,
				ChunkOverlap:    -10,
				ChunkStrategy:   "fixed",
				MaxResults:      -5,
				VectorStorePath: "relative/path",
				UseCorpus:       true,
				CorpusPath:      "relative/corpus",
				CorpusMaxSizeMB: -500,
			},
			verify: func(t *testing.T, config Config) {
				// Config should store negative values as-is
				// Validation happens at usage time
				assert.Equal(t, -100, config.ChunkSize)
				assert.Equal(t, -10, config.ChunkOverlap)
				assert.Equal(t, -5, config.MaxResults)
				assert.Equal(t, -500, config.CorpusMaxSizeMB)
			},
		},
		{
			name: "Markdown header strategy",
			config: Config{
				CollectionName:  "markdown_collection",
				ChunkSize:       500,
				ChunkOverlap:    50,
				ChunkStrategy:   "markdown_header",
				MaxResults:      10,
				VectorStorePath: "/absolute/path/to/store",
				UseCorpus:       true,
				CorpusPath:      "/absolute/path/to/corpus",
				CorpusMaxSizeMB: 5000,
			},
			verify: func(t *testing.T, config Config) {
				assert.Equal(t, "markdown_header", config.ChunkStrategy)
				assert.Equal(t, "/absolute/path/to/corpus", config.CorpusPath)
				assert.Equal(t, 5000, config.CorpusMaxSizeMB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.config)
		})
	}
}

func TestRetrievalConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config RetrievalConfig
		verify func(t *testing.T, config RetrievalConfig)
	}{
		{
			name: "Custom retrieval config",
			config: RetrievalConfig{
				Query:               "test query",
				MaxResults:          15,
				MinScore:            0.85,
				IncludeMetadata:     true,
				UseCorpus:           true,
				DisableVectorSearch: false,
			},
			verify: func(t *testing.T, config RetrievalConfig) {
				assert.Equal(t, "test query", config.Query)
				assert.Equal(t, 15, config.MaxResults)
				assert.Equal(t, float32(0.85), config.MinScore)
				assert.True(t, config.IncludeMetadata)
				assert.True(t, config.UseCorpus)
				assert.False(t, config.DisableVectorSearch)
			},
		},
		{
			name: "Corpus disabled",
			config: RetrievalConfig{
				MaxResults:          3,
				MinScore:            0.5,
				IncludeMetadata:     false,
				UseCorpus:           false,
				DisableVectorSearch: false,
			},
			verify: func(t *testing.T, config RetrievalConfig) {
				assert.Equal(t, 3, config.MaxResults)
				assert.Equal(t, float32(0.5), config.MinScore)
				assert.False(t, config.IncludeMetadata)
				assert.False(t, config.UseCorpus)
			},
		},
		{
			name: "Edge case values",
			config: RetrievalConfig{
				MaxResults:          1000, // Very high
				MinScore:            1.0,  // Maximum score
				IncludeMetadata:     true,
				UseCorpus:           true,
				DisableVectorSearch: true, // Only corpus search
			},
			verify: func(t *testing.T, config RetrievalConfig) {
				assert.Equal(t, 1000, config.MaxResults)
				assert.Equal(t, float32(1.0), config.MinScore)
				assert.True(t, config.DisableVectorSearch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.config)
		})
	}
}

func TestConfig_DeepCopy(t *testing.T) {
	original := Config{
		CollectionName:  "original_collection",
		ChunkSize:       750,
		ChunkOverlap:    100,
		ChunkStrategy:   "sentence",
		MaxResults:      25,
		VectorStorePath: "/original/vector/path",
		UseCorpus:       true,
		CorpusPath:      "/original/corpus/path",
		CorpusMaxSizeMB: 1500,
	}
	
	// Create a copy
	copy := original
	
	// Modify the copy
	copy.CollectionName = "modified_collection"
	copy.ChunkSize = 1000
	copy.CorpusPath = "/modified/corpus/path"
	
	// Verify original is unchanged (structs are value types in Go)
	assert.Equal(t, "original_collection", original.CollectionName)
	assert.Equal(t, 750, original.ChunkSize)
	assert.Equal(t, "/original/corpus/path", original.CorpusPath)
	
	// Verify copy has new values
	assert.Equal(t, "modified_collection", copy.CollectionName)
	assert.Equal(t, 1000, copy.ChunkSize)
	assert.Equal(t, "/modified/corpus/path", copy.CorpusPath)
}