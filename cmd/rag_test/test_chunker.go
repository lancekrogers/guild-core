package main

import (
	"fmt"
	"strings"
)

// ChunkStrategy defines how to split documents into chunks
type ChunkStrategy string

const (
	// ChunkByParagraph splits by paragraphs (double newlines)
	ChunkByParagraph ChunkStrategy = "paragraph"

	// ChunkBySentence splits by sentences (periods, question marks, exclamation points)
	ChunkBySentence ChunkStrategy = "sentence"

	// ChunkByFixedSize splits by a fixed number of characters
	ChunkByFixedSize ChunkStrategy = "fixed"

	// ChunkByMarkdownHeader splits by markdown headers (# Header)
	ChunkByMarkdownHeader ChunkStrategy = "markdown-header"
)

// Chunker handles document chunking for RAG
type Chunker struct {
	ChunkSize     int          // Size of each chunk (characters/tokens)
	ChunkOverlap  int          // Number of characters/tokens to overlap between chunks
	SplitStrategy ChunkStrategy // Strategy for splitting text
}

// ChunkerOption is a functional option for configuring a Chunker
type ChunkerOption func(*Chunker)

// WithChunkSize sets the chunk size
func WithChunkSize(size int) ChunkerOption {
	return func(c *Chunker) {
		c.ChunkSize = size
	}
}

// WithChunkOverlap sets the chunk overlap
func WithChunkOverlap(overlap int) ChunkerOption {
	return func(c *Chunker) {
		c.ChunkOverlap = overlap
	}
}

// WithSplitStrategy sets the split strategy
func WithSplitStrategy(strategy ChunkStrategy) ChunkerOption {
	return func(c *Chunker) {
		c.SplitStrategy = strategy
	}
}

// NewChunker creates a new document chunker with provided options
func NewChunker(opts ...ChunkerOption) *Chunker {
	// Default values
	chunker := &Chunker{
		ChunkSize:     1000,         // Default to 1000 characters
		ChunkOverlap:  100,          // Default to 100 characters overlap
		SplitStrategy: ChunkByParagraph, // Default to splitting by paragraph
	}

	// Apply options
	for _, opt := range opts {
		opt(chunker)
	}

	return chunker
}

// ChunkDocument breaks a document into chunks based on the configured strategy
func (c *Chunker) ChunkDocument(text string) []string {
	switch c.SplitStrategy {
	case ChunkByParagraph:
		return c.chunkByParagraph(text)
	case ChunkByFixedSize:
		return c.chunkByFixedSize(text)
	default:
		return c.chunkByParagraph(text) // Default to paragraphs
	}
}

// chunkByParagraph splits text into chunks by paragraphs
func (c *Chunker) chunkByParagraph(text string) []string {
	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")
	
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0
	
	for _, para := range paragraphs {
		// Skip empty paragraphs
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		
		paraSize := len(para)
		
		// If paragraph is too large on its own, split it further
		if paraSize > c.ChunkSize {
			// Add current chunk if not empty
			if currentSize > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentSize = 0
			}
			
			// Split large paragraph into fixed-size chunks
			paraChunks := c.chunkByFixedSize(para)
			chunks = append(chunks, paraChunks...)
			continue
		}
		
		// If adding this paragraph would exceed chunk size, start a new chunk
		if currentSize + paraSize > c.ChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(para)
			currentSize = paraSize
		} else {
			// Add to current chunk
			if currentSize > 0 {
				currentChunk.WriteString("\n\n")
				currentSize += 2
			}
			currentChunk.WriteString(para)
			currentSize += paraSize
		}
	}
	
	// Add the last chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// chunkByFixedSize splits text into chunks of fixed size
func (c *Chunker) chunkByFixedSize(text string) []string {
	var chunks []string
	
	// Use runes to correctly handle UTF-8 characters
	textRunes := []rune(text)
	textLen := len(textRunes)
	
	for i := 0; i < textLen; i += c.ChunkSize - c.ChunkOverlap {
		// Calculate start and end positions
		start := i
		end := i + c.ChunkSize
		
		// Ensure we don't go beyond the text length
		if end > textLen {
			end = textLen
		}
		
		// Extract the chunk
		chunk := string(textRunes[start:end])
		chunks = append(chunks, chunk)
		
		// Break if we've reached the end
		if end == textLen {
			break
		}
	}
	
	return chunks
}

func main() {
	// Create a new chunker
	chunker := NewChunker(
		WithChunkSize(300),
		WithChunkOverlap(50),
		WithSplitStrategy(ChunkByParagraph),
	)
	
	// Test document
	testDoc := `# Document Title
	
	This is the first paragraph of the document. This paragraph contains multiple sentences.
	This is still part of the first paragraph.
	
	This is the second paragraph. It is separate from the first one.
	
	This is the third paragraph. The document is chunked by paragraphs by default.
	This means each paragraph will be a separate chunk, unless it exceeds the maximum chunk size.
	
	Very long paragraphs will be split into multiple chunks based on the maximum chunk size.
	This ensures that no chunk is too large for processing by the language model.`
	
	// Chunk the document
	chunks := chunker.ChunkDocument(testDoc)
	
	// Print the chunks
	fmt.Printf("Document chunked into %d parts:\n\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d (%d chars):\n%s\n\n", i+1, len(chunk), chunk)
	}
}