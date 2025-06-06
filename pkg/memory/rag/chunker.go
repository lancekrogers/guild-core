package rag

import (
	"strings"
)

// ChunkStrategy defines the strategy for chunking text
type ChunkStrategy string

const (
	// ChunkByParagraph chunks by paragraphs
	ChunkByParagraph ChunkStrategy = "paragraph"
	
	// ChunkBySentence chunks by sentences
	ChunkBySentence ChunkStrategy = "sentence"
	
	// ChunkByFixedSize chunks by fixed size
	ChunkByFixedSize ChunkStrategy = "fixed"
	
	// ChunkByMarkdownHeader chunks by markdown headers
	ChunkByMarkdownHeader ChunkStrategy = "markdown_header"
)

// ChunkerConfig defines the configuration for the chunker
type ChunkerConfig struct {
	// ChunkSize is the target size of each chunk in tokens
	ChunkSize int
	
	// ChunkOverlap is the number of tokens to overlap between chunks
	ChunkOverlap int
	
	// Strategy is the chunking strategy to use
	Strategy ChunkStrategy
}

// Chunker breaks text into chunks for embedding
type Chunker struct {
	Config ChunkerConfig
}

// newChunker creates a new Chunker (private constructor)
func newChunker(config ChunkerConfig) *Chunker {
	// Default config values
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	
	if config.ChunkOverlap <= 0 {
		config.ChunkOverlap = 200
	}
	
	if config.Strategy == "" {
		config.Strategy = ChunkByParagraph
	}
	
	return &Chunker{
		Config: config,
	}
}

// ChunkDocument breaks a document into chunks based on the configured strategy
func (c *Chunker) ChunkDocument(text string) []string {
	switch c.Config.Strategy {
	case ChunkByParagraph:
		return c.chunkByParagraph(text)
	case ChunkBySentence:
		return c.chunkBySentence(text)
	case ChunkByFixedSize:
		return c.chunkByFixedSize(text)
	case ChunkByMarkdownHeader:
		return c.chunkByMarkdownHeader(text)
	default:
		return c.chunkByParagraph(text)
	}
}

// chunkByParagraph chunks text by paragraphs
func (c *Chunker) chunkByParagraph(text string) []string {
	// Split by double newline (paragraph)
	paragraphs := strings.Split(text, "\n\n")
	
	// Group paragraphs into chunks of appropriate size
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int
	
	for _, para := range paragraphs {
		// Simple token count estimation (words)
		paraSize := len(strings.Fields(para))
		
		if currentSize+paraSize > c.Config.ChunkSize && currentSize > 0 {
			// Current paragraph would make chunk too large, start a new chunk
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentSize = 0
		}
		
		// Add paragraph to current chunk
		if currentSize > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
		currentSize += paraSize
	}
	
	// Add final chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// chunkBySentence chunks text by sentences
func (c *Chunker) chunkBySentence(text string) []string {
	// Simple sentence splitting
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	
	// Group sentences into chunks of appropriate size
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int
	
	for i, sentence := range sentences {
		// Clean up sentence
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		// Add sentence terminator back
		if i < len(text) && (text[i] == '.' || text[i] == '!' || text[i] == '?') {
			sentence += string(text[i])
		} else {
			sentence += "."
		}
		
		// Simple token count estimation (words)
		sentSize := len(strings.Fields(sentence))
		
		if currentSize+sentSize > c.Config.ChunkSize && currentSize > 0 {
			// Current sentence would make chunk too large, start a new chunk
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentSize = 0
		}
		
		// Add sentence to current chunk
		if currentSize > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		currentSize += sentSize
	}
	
	// Add final chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// chunkByFixedSize chunks text by fixed size
func (c *Chunker) chunkByFixedSize(text string) []string {
	words := strings.Fields(text)
	var chunks []string
	
	for i := 0; i < len(words); i += c.Config.ChunkSize - c.Config.ChunkOverlap {
		end := i + c.Config.ChunkSize
		if end > len(words) {
			end = len(words)
		}
		
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		
		// If we've reached the end, stop
		if end == len(words) {
			break
		}
	}
	
	return chunks
}

// chunkByMarkdownHeader chunks text by markdown headers
func (c *Chunker) chunkByMarkdownHeader(text string) []string {
	// Split by markdown headers (# Header)
	lines := strings.Split(text, "\n")
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int
	
	for _, line := range lines {
		// Check if line is a header
		isHeader := false
		if len(line) > 0 && line[0] == '#' {
			headerLevel := 0
			for i := 0; i < len(line) && line[i] == '#'; i++ {
				headerLevel++
			}
			
			// Check that its properly formatted with a space after #
			if headerLevel < len(line) && line[headerLevel] == ' ' {
				isHeader = true
				
				// If we have content in the current chunk, add it to chunks
				if currentSize > 0 {
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
					currentSize = 0
				}
			}
		}
		
		// Add line to current chunk
		if currentSize > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		currentSize += len(strings.Fields(line))
		
		// If chunk is too large, split it (unless we just started a new chunk)
		if !isHeader && currentSize > c.Config.ChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentSize = 0
		}
	}
	
	// Add final chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// ChunkWithMeta represents a chunk with metadata
type ChunkWithMeta struct {
	Content  string
	Index    int
	Metadata map[string]interface{}
}

// ChunkWithMetadata chunks a document and returns chunks with metadata
func (c *Chunker) ChunkWithMetadata(content string) []ChunkWithMeta {
	chunks := c.ChunkDocument(content)
	result := make([]ChunkWithMeta, len(chunks))
	
	for i, chunk := range chunks {
		result[i] = ChunkWithMeta{
			Content: chunk,
			Index:   i,
			Metadata: map[string]interface{}{
				"chunk_index":  i,
				"total_chunks": len(chunks),
				"strategy":     string(c.Config.Strategy),
				"chunk_size":   c.Config.ChunkSize,
				"overlap":      c.Config.ChunkOverlap,
			},
		}
	}
	
	return result
}

// GetConfig returns the chunker configuration
func (c *Chunker) GetConfig() ChunkerConfig {
	return c.Config
}

// DefaultChunkerFactory creates a chunker for registry use
func DefaultChunkerFactory(config ChunkerConfig) (ChunkerInterface, error) {
	return newChunker(config), nil
}