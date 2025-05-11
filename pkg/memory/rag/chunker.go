package rag

import (
	"fmt"
	"regexp"
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
	case ChunkBySentence:
		return c.chunkBySentence(text)
	case ChunkByFixedSize:
		return c.chunkByFixedSize(text)
	case ChunkByMarkdownHeader:
		return c.chunkByMarkdownHeader(text)
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

// chunkBySentence splits text into chunks by sentences
func (c *Chunker) chunkBySentence(text string) []string {
	// This is a simple sentence splitter that considers ., !, ? as sentence endings
	// It's not perfect - for real NLP, consider using a proper sentence tokenizer
	sentenceRegex := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentenceRegex.Split(text, -1)
	
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0
	
	for _, sentence := range sentences {
		// Skip empty sentences
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		sentenceSize := len(sentence)
		
		// If sentence is too large on its own, split it by fixed size
		if sentenceSize > c.ChunkSize {
			// Add current chunk if not empty
			if currentSize > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentSize = 0
			}
			
			// Split large sentence into fixed-size chunks
			sentenceChunks := c.chunkByFixedSize(sentence)
			chunks = append(chunks, sentenceChunks...)
			continue
		}
		
		// If adding this sentence would exceed chunk size, start a new chunk
		if currentSize + sentenceSize > c.ChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(sentence)
			currentSize = sentenceSize
		} else {
			// Add to current chunk
			if currentSize > 0 {
				currentChunk.WriteString(" ")
				currentSize++
			}
			currentChunk.WriteString(sentence)
			currentSize += sentenceSize
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

// chunkByMarkdownHeader splits text by markdown headers
func (c *Chunker) chunkByMarkdownHeader(text string) []string {
	// Split the text into lines
	lines := strings.Split(text, "\n")
	
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0
	
	// Regular expression to match Markdown headers (# Header)
	headerRegex := regexp.MustCompile(`^#{1,6}\s+.+$`)
	
	for _, line := range lines {
		lineSize := len(line)
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			if currentSize > 0 {
				currentChunk.WriteString("\n")
				currentSize++
			}
			continue
		}
		
		// If line is a header and we have content already, start a new chunk
		if headerRegex.MatchString(line) && currentSize > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(line)
			currentSize = lineSize
		} else if currentSize + lineSize > c.ChunkSize {
			// If adding this line would exceed chunk size, start a new chunk
			// But only if we have significant content already
			if currentSize > 100 { // Avoid empty chunks or tiny chunks
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentChunk.WriteString(line)
				currentSize = lineSize
			} else {
				// Just add to the current chunk even if it exceeds the limit a bit
				currentChunk.WriteString("\n")
				currentChunk.WriteString(line)
				currentSize += lineSize + 1
			}
		} else {
			// Add to current chunk
			if currentSize > 0 {
				currentChunk.WriteString("\n")
				currentSize++
			}
			currentChunk.WriteString(line)
			currentSize += lineSize
		}
	}
	
	// Add the last chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// ChunkWithMetadata represents a document chunk with metadata
type ChunkWithMetadata struct {
	// Content is the text content of the chunk
	Content string
	
	// Source is the source of the original document
	Source string
	
	// Metadata contains additional information about the chunk
	Metadata map[string]interface{}
}

// ChunkDocumentWithMetadata breaks a document into chunks and preserves metadata
func (c *Chunker) ChunkDocumentWithMetadata(text, source string, metadata map[string]interface{}) []ChunkWithMetadata {
	chunks := c.ChunkDocument(text)
	
	// Create chunks with metadata
	chunksWithMetadata := make([]ChunkWithMetadata, len(chunks))
	for i, chunk := range chunks {
		// Copy metadata for each chunk
		chunkMetadata := make(map[string]interface{})
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		
		// Add chunk-specific metadata
		chunkMetadata["chunk_index"] = i
		chunkMetadata["chunk_count"] = len(chunks)
		
		chunksWithMetadata[i] = ChunkWithMetadata{
			Content:  chunk,
			Source:   source,
			Metadata: chunkMetadata,
		}
	}
	
	return chunksWithMetadata
}