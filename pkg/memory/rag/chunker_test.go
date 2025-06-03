package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChunker(t *testing.T) {
	// Test default values
	chunker := NewChunker(ChunkerConfig{})
	assert.Equal(t, 1000, chunker.Config.ChunkSize)
	assert.Equal(t, 200, chunker.Config.ChunkOverlap)
	assert.Equal(t, ChunkByParagraph, chunker.Config.Strategy)

	// Test with custom config
	config := ChunkerConfig{
		ChunkSize:    500,
		ChunkOverlap: 50,
		Strategy:     ChunkBySentence,
	}
	chunker = NewChunker(config)
	assert.Equal(t, 500, chunker.Config.ChunkSize)
	assert.Equal(t, 50, chunker.Config.ChunkOverlap)
	assert.Equal(t, ChunkBySentence, chunker.Config.Strategy)
}

func TestChunkByParagraph(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    20, // Word count, not character count
		ChunkOverlap: 0,
		Strategy:     ChunkByParagraph,
	}
	chunker := NewChunker(config)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name: "Single paragraph within chunk size",
			text: "This is a single paragraph that fits within the chunk size.",
			expected: []string{
				"This is a single paragraph that fits within the chunk size.",
			},
		},
		{
			name: "Multiple paragraphs within chunk size",
			text: "This is the first paragraph.\n\nThis is the second paragraph.",
			expected: []string{
				"This is the first paragraph.\n\nThis is the second paragraph.",
			},
		},
		{
			name: "Multiple paragraphs exceeding chunk size",
			text: "This is the first paragraph with some text that will make it exceed the chunk size when combined with the second paragraph.\n\nThis is the second paragraph.",
			expected: []string{
				"This is the first paragraph with some text that will make it exceed the chunk size when combined with the second paragraph.",
				"This is the second paragraph.",
			},
		},
		{
			name: "Long paragraph exceeding chunk size",
			text: "This is a very long paragraph that exceeds the chunk size on its own. It contains a lot of text that will need to be split into multiple chunks.",
			expected: []string{
				"This is a very long paragraph that exceeds the chunk size on its own. It contains a lot of text that will need to be split into multiple chunks.",
			},
		},
		{
			name:     "Empty text",
			text:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.ChunkDocument(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkBySentence(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    20, // Word count, not character count
		ChunkOverlap: 0,
		Strategy:     ChunkBySentence,
	}
	chunker := NewChunker(config)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name: "Single sentence within chunk size",
			text: "This is a single sentence that fits within the chunk size.",
			expected: []string{
				"This is a single sentence that fits within the chunk size.",
			},
		},
		{
			name: "Multiple sentences within chunk size",
			text: "This is the first sentence. This is the second sentence.",
			expected: []string{
				"This is the first sentence. This is the second sentence.",
			},
		},
		{
			name: "Multiple sentences exceeding chunk size",
			text: "This is the first sentence with a lot of text that will make it exceed the chunk size when combined with other sentences. This is the second sentence.",
			expected: []string{
				"This is the first sentence with a lot of text that will make it exceed the chunk size when combined with other sentences.",
				"This is the second sentence.",
			},
		},
		{
			name: "Long sentence exceeding chunk size",
			text: "This is a very long sentence that exceeds the chunk size on its own and it contains a lot of text that will need to be split into multiple chunks.",
			expected: []string{
				"This is a very long sentence that exceeds the chunk size on its own and it contains a lot of text that will need to be split into multiple chunks.",
			},
		},
		{
			name: "Mixed sentence endings",
			text: "This is a statement. This is a question? This is an exclamation!",
			expected: []string{
				"This is a statement. This is a question. This is an exclamation.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.ChunkDocument(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkByFixedSize(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    20,
		ChunkOverlap: 5,
		Strategy:     ChunkByFixedSize,
	}
	chunker := NewChunker(config)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name: "Text smaller than chunk size",
			text: "Small text.",
			expected: []string{
				"Small text.",
			},
		},
		{
			name: "Text exactly chunk size",
			text: "This is twenty chars",
			expected: []string{
				"This is twenty chars",
			},
		},
		{
			name: "Text larger than chunk size with word count",
			text: "This text is longer than twenty characters and should be split into multiple chunks based on word count not character count",
			expected: []string{
				"This text is longer than twenty characters and should be split into multiple chunks based on word count not character count",
			},
		},
		{
			name:     "Empty text",
			text:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.ChunkDocument(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkByMarkdownHeader(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:    20, // Word count, not character count
		ChunkOverlap: 0,
		Strategy:     ChunkByMarkdownHeader,
	}
	chunker := NewChunker(config)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name: "Single section within chunk size",
			text: "# Header\nThis is content under the header.",
			expected: []string{
				"# Header\nThis is content under the header.",
			},
		},
		{
			name: "Multiple sections within chunk size",
			text: "# Header 1\nContent 1.\n\n## Header 2\nContent 2.",
			expected: []string{
				"# Header 1\nContent 1.\n\n## Header 2\nContent 2.",
			},
		},
		{
			name: "Multiple sections exceeding chunk size",
			text: "# Header 1\nThis is the content under header 1 which has enough text to exceed the chunk size when combined with header 2.\n\n# Header 2\nThis is content under header 2.",
			expected: []string{
				"# Header 1\nThis is the content under header 1 which has enough text to exceed the chunk size when combined with header 2.",
				"# Header 2\nThis is content under header 2.",
			},
		},
		{
			name: "Different header levels",
			text: "# H1\nContent.\n\n## H2\nMore content.\n\n### H3\nEven more content.",
			expected: []string{
				"# H1\nContent.\n\n## H2\nMore content.\n\n### H3\nEven more content.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.ChunkDocument(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}