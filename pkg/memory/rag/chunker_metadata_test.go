package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkWithMetadata(t *testing.T) {
	tests := []struct {
		name   string
		config ChunkerConfig
		text   string
		verify func(t *testing.T, chunks []ChunkWithMeta)
	}{
		{
			name: "Simple text with metadata",
			config: ChunkerConfig{
				ChunkSize:    3, // Small enough to force splitting
				ChunkOverlap: 1,
				Strategy:     ChunkByParagraph,
			},
			text: "This is paragraph one.\n\nThis is paragraph two.",
			verify: func(t *testing.T, chunks []ChunkWithMeta) {
				require.Len(t, chunks, 2)

				// Check first chunk
				assert.Equal(t, "This is paragraph one.", chunks[0].Content)
				assert.Equal(t, 0, chunks[0].Index)
				assert.Equal(t, 0, chunks[0].Metadata["chunk_index"])
				assert.Equal(t, 2, chunks[0].Metadata["total_chunks"])
				assert.Equal(t, "paragraph", chunks[0].Metadata["strategy"])
				assert.Equal(t, 3, chunks[0].Metadata["chunk_size"])
				assert.Equal(t, 1, chunks[0].Metadata["overlap"])

				// Check second chunk
				assert.Equal(t, "This is paragraph two.", chunks[1].Content)
				assert.Equal(t, 1, chunks[1].Index)
				assert.Equal(t, 1, chunks[1].Metadata["chunk_index"])
				assert.Equal(t, 2, chunks[1].Metadata["total_chunks"])
			},
		},
		{
			name: "Sentence strategy with metadata",
			config: ChunkerConfig{
				ChunkSize:    1, // Force each sentence into separate chunk
				ChunkOverlap: 1,
				Strategy:     ChunkBySentence,
			},
			text: "First sentence. Second sentence. Third sentence.",
			verify: func(t *testing.T, chunks []ChunkWithMeta) {
				require.Len(t, chunks, 3)
				for i, chunk := range chunks {
					assert.Equal(t, i, chunk.Index)
					assert.Equal(t, i, chunk.Metadata["chunk_index"])
					assert.Equal(t, 3, chunk.Metadata["total_chunks"])
					assert.Equal(t, "sentence", chunk.Metadata["strategy"])
					assert.Equal(t, 1, chunk.Metadata["chunk_size"])
					assert.Equal(t, 1, chunk.Metadata["overlap"])
				}
			},
		},
		{
			name: "Fixed size strategy with metadata",
			config: ChunkerConfig{
				ChunkSize:    10,
				ChunkOverlap: 2,
				Strategy:     ChunkByFixedSize,
			},
			text: "This is a text that will be chunked by fixed size with some overlap between chunks",
			verify: func(t *testing.T, chunks []ChunkWithMeta) {
				require.Greater(t, len(chunks), 1)
				for i, chunk := range chunks {
					assert.Equal(t, i, chunk.Index)
					assert.Equal(t, i, chunk.Metadata["chunk_index"])
					assert.Equal(t, len(chunks), chunk.Metadata["total_chunks"])
					assert.Equal(t, "fixed", chunk.Metadata["strategy"])
					assert.Equal(t, 10, chunk.Metadata["chunk_size"])
					assert.Equal(t, 2, chunk.Metadata["overlap"])
				}
			},
		},
		{
			name: "Markdown header strategy with metadata",
			config: ChunkerConfig{
				ChunkSize:    50,
				ChunkOverlap: 10,
				Strategy:     ChunkByMarkdownHeader,
			},
			text: "# Header 1\nContent for header 1\n\n## Header 2\nContent for header 2",
			verify: func(t *testing.T, chunks []ChunkWithMeta) {
				require.Len(t, chunks, 2)
				for i, chunk := range chunks {
					assert.Equal(t, i, chunk.Index)
					assert.Equal(t, i, chunk.Metadata["chunk_index"])
					assert.Equal(t, 2, chunk.Metadata["total_chunks"])
					assert.Equal(t, "markdown_header", chunk.Metadata["strategy"])
					assert.Equal(t, 50, chunk.Metadata["chunk_size"])
					assert.Equal(t, 10, chunk.Metadata["overlap"])
				}
			},
		},
		{
			name: "Empty text with metadata",
			config: ChunkerConfig{
				ChunkSize:    100,
				ChunkOverlap: 20,
				Strategy:     ChunkByParagraph,
			},
			text: "",
			verify: func(t *testing.T, chunks []ChunkWithMeta) {
				assert.Empty(t, chunks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker := newChunker(tt.config)
			chunks := chunker.ChunkWithMetadata(tt.text)
			tt.verify(t, chunks)
		})
	}
}

func TestGetConfig(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    500,
		ChunkOverlap: 100,
		Strategy:     ChunkByMarkdownHeader,
	}

	chunker := newChunker(config)
	retrievedConfig := chunker.GetConfig()

	assert.Equal(t, config.ChunkSize, retrievedConfig.ChunkSize)
	assert.Equal(t, config.ChunkOverlap, retrievedConfig.ChunkOverlap)
	assert.Equal(t, config.Strategy, retrievedConfig.Strategy)
}

func TestDefaultChunkerFactory(t *testing.T) {
	tests := []struct {
		name   string
		config ChunkerConfig
	}{
		{
			name:   "Default config",
			config: ChunkerConfig{},
		},
		{
			name: "Custom config",
			config: ChunkerConfig{
				ChunkSize:    750,
				ChunkOverlap: 150,
				Strategy:     ChunkBySentence,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker, err := DefaultChunkerFactory(tt.config)
			require.NoError(t, err)
			require.NotNil(t, chunker)

			// Verify it implements the interface
			_, ok := chunker.(ChunkerInterface)
			assert.True(t, ok)

			// Verify config is applied
			retrievedConfig := chunker.GetConfig()
			if tt.config.ChunkSize > 0 {
				assert.Equal(t, tt.config.ChunkSize, retrievedConfig.ChunkSize)
			} else {
				assert.Equal(t, 1000, retrievedConfig.ChunkSize) // default
			}
			if tt.config.ChunkOverlap > 0 {
				assert.Equal(t, tt.config.ChunkOverlap, retrievedConfig.ChunkOverlap)
			} else {
				assert.Equal(t, 200, retrievedConfig.ChunkOverlap) // default
			}
			if tt.config.Strategy != "" {
				assert.Equal(t, tt.config.Strategy, retrievedConfig.Strategy)
			} else {
				assert.Equal(t, ChunkByParagraph, retrievedConfig.Strategy) // default
			}
		})
	}
}

func TestChunkDocument_UnknownStrategy(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    5, // Small enough to force splitting
		ChunkOverlap: 1,
		Strategy:     ChunkStrategy("unknown_strategy"),
	}

	chunker := newChunker(config)
	text := "This is some text to be chunked.\n\nThis is another paragraph."

	// Should fall back to paragraph chunking
	chunks := chunker.ChunkDocument(text)
	assert.NotEmpty(t, chunks)
	assert.Equal(t, 2, len(chunks))
}
