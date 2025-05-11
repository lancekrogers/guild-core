package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChunker(t *testing.T) {
	// Test default values
	chunker := NewChunker()
	assert.Equal(t, 1000, chunker.ChunkSize)
	assert.Equal(t, 100, chunker.ChunkOverlap)
	assert.Equal(t, ChunkByParagraph, chunker.SplitStrategy)

	// Test with options
	chunker = NewChunker(
		WithChunkSize(500),
		WithChunkOverlap(50),
		WithSplitStrategy(ChunkBySentence),
	)
	assert.Equal(t, 500, chunker.ChunkSize)
	assert.Equal(t, 50, chunker.ChunkOverlap)
	assert.Equal(t, ChunkBySentence, chunker.SplitStrategy)
}

func TestChunkByParagraph(t *testing.T) {
	chunker := NewChunker(
		WithChunkSize(100),
		WithChunkOverlap(0),
		WithSplitStrategy(ChunkByParagraph),
	)

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
				"This is a very long paragraph that exceeds the chunk size on its own. It contains a lot of text that will need to be split ",
				"into multiple chunks.",
			},
		},
		{
			name: "Empty text",
			text: "",
			expected: []string{},
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
	chunker := NewChunker(
		WithChunkSize(100),
		WithChunkOverlap(0),
		WithSplitStrategy(ChunkBySentence),
	)

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
				"This is the first sentence This is the second sentence",
			},
		},
		{
			name: "Multiple sentences exceeding chunk size",
			text: "This is the first sentence with a lot of text that will make it exceed the chunk size when combined with other sentences. This is the second sentence.",
			expected: []string{
				"This is the first sentence with a lot of text that will make it exceed the chunk size when combined with other sentences",
				"This is the second sentence",
			},
		},
		{
			name: "Long sentence exceeding chunk size",
			text: "This is a very long sentence that exceeds the chunk size on its own and it contains a lot of text that will need to be split into multiple chunks.",
			expected: []string{
				"This is a very long sentence that exceeds the chunk size on its own and it contains a lot of text that will need to be ",
				"split into multiple chunks",
			},
		},
		{
			name: "Mixed sentence endings",
			text: "This is a statement. This is a question? This is an exclamation!",
			expected: []string{
				"This is a statement This is a question This is an exclamation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.chunkBySentence(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkByFixedSize(t *testing.T) {
	chunker := NewChunker(
		WithChunkSize(20),
		WithChunkOverlap(5),
		WithSplitStrategy(ChunkByFixedSize),
	)

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
			name: "Text larger than chunk size",
			text: "This text is longer than twenty characters and should be split.",
			expected: []string{
				"This text is longer ",
				"than twenty charact",
				"acters and should ",
				"be split.",
			},
		},
		{
			name: "Empty text",
			text: "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.chunkByFixedSize(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkByMarkdownHeader(t *testing.T) {
	chunker := NewChunker(
		WithChunkSize(100),
		WithChunkOverlap(0),
		WithSplitStrategy(ChunkByMarkdownHeader),
	)

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
			chunks := chunker.chunkByMarkdownHeader(tt.text)
			assert.Equal(t, tt.expected, chunks)
		})
	}
}

func TestChunkDocumentWithMetadata(t *testing.T) {
	chunker := NewChunker(
		WithChunkSize(100),
		WithChunkOverlap(0),
		WithSplitStrategy(ChunkByParagraph),
	)

	text := "This is paragraph one.\n\nThis is paragraph two."
	source := "test document"
	metadata := map[string]interface{}{
		"author": "test author",
		"date":   "2023-01-01",
	}

	chunks := chunker.ChunkDocumentWithMetadata(text, source, metadata)

	// Should be a single chunk since the text fits in one chunk
	assert.Len(t, chunks, 1)

	// Check content and source
	assert.Equal(t, "This is paragraph one.\n\nThis is paragraph two.", chunks[0].Content)
	assert.Equal(t, "test document", chunks[0].Source)

	// Check metadata
	assert.Equal(t, "test author", chunks[0].Metadata["author"])
	assert.Equal(t, "2023-01-01", chunks[0].Metadata["date"])
	assert.Equal(t, 0, chunks[0].Metadata["chunk_index"])
	assert.Equal(t, 1, chunks[0].Metadata["chunk_count"])

	// Test with multiple chunks
	longText := "This is paragraph one with a lot of text that will make it exceed the chunk size when combined with other paragraphs.\n\nThis is paragraph two with some additional text."
	chunks = chunker.ChunkDocumentWithMetadata(longText, source, metadata)

	// Should be two chunks
	assert.Len(t, chunks, 2)

	// Check content of first chunk
	assert.Equal(t, "This is paragraph one with a lot of text that will make it exceed the chunk size when combined with other paragraphs.", chunks[0].Content)
	assert.Equal(t, 0, chunks[0].Metadata["chunk_index"])
	assert.Equal(t, 2, chunks[0].Metadata["chunk_count"])

	// Check content of second chunk
	assert.Equal(t, "This is paragraph two with some additional text.", chunks[1].Content)
	assert.Equal(t, 1, chunks[1].Metadata["chunk_index"])
	assert.Equal(t, 2, chunks[1].Metadata["chunk_count"])
}