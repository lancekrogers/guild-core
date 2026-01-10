// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// Extractor extracts reasoning blocks from LLM responses
type Extractor struct {
	patterns []Pattern

	// Callbacks for integration
	OnExtraction func(blocks []ReasoningBlock, duration time.Duration, err error)
}

// Pattern defines a reasoning extraction pattern
type Pattern struct {
	Name       string
	StartTag   string
	EndTag     string
	Priority   int
	TokenScale float64 // Scale factor for token estimation
}

// NewExtractor creates a new reasoning extractor
func NewExtractor() *Extractor {
	return &Extractor{
		patterns: []Pattern{
			{
				Name:       "thinking_tags",
				StartTag:   "<thinking",
				EndTag:     "</thinking>",
				Priority:   1,
				TokenScale: 0.25, // ~4 chars per token
			},
			{
				Name:       "reasoning_tags",
				StartTag:   "<reasoning",
				EndTag:     "</reasoning>",
				Priority:   2,
				TokenScale: 0.25,
			},
			{
				Name:       "analysis_tags",
				StartTag:   "<analysis",
				EndTag:     "</analysis>",
				Priority:   3,
				TokenScale: 0.25,
			},
		},
	}
}

// ExtractFromResponse extracts reasoning blocks from a chat response
func (e *Extractor) ExtractFromResponse(ctx context.Context, response *interfaces.ChatResponse) ([]*interfaces.ReasoningBlock, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("reasoning.extractor")
	}

	if response == nil || len(response.Choices) == 0 {
		return nil, nil
	}

	var allBlocks []*interfaces.ReasoningBlock
	for _, choice := range response.Choices {
		blocks := e.ExtractFromContent(choice.Message.Content)
		allBlocks = append(allBlocks, blocks...)
	}

	// Update token counts if we have reasoning blocks
	if len(allBlocks) > 0 && response.Usage.ReasoningTokens == 0 {
		reasoningTokens := 0
		for _, block := range allBlocks {
			reasoningTokens += block.TokenCount
		}
		response.Usage.ReasoningTokens = reasoningTokens
	}

	return allBlocks, nil
}

// Extract extracts reasoning blocks from content with context support
func (e *Extractor) Extract(ctx context.Context, content string) ([]ReasoningBlock, error) {
	start := time.Now()

	if err := ctx.Err(); err != nil {
		err = gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled").
			WithComponent("reasoning_extractor")
		if e.OnExtraction != nil {
			e.OnExtraction(nil, time.Since(start), err)
		}
		return nil, err
	}

	// Extract blocks using patterns
	interfaceBlocks := e.ExtractFromContent(content)

	// Convert to local ReasoningBlock type
	blocks := make([]ReasoningBlock, len(interfaceBlocks))
	for i, ib := range interfaceBlocks {
		blocks[i] = ReasoningBlock{
			ID:         ib.ID,
			Type:       ib.Type,
			Content:    ib.Content,
			Timestamp:  ib.Timestamp,
			Duration:   ib.Duration,
			TokenCount: ib.TokenCount,
			Depth:      ib.Depth,
			ParentID:   ib.ParentID,
			Children:   ib.Children,
			Confidence: ib.Confidence,
			Metadata:   ib.Metadata,
		}
	}

	// Call callback if set
	if e.OnExtraction != nil {
		e.OnExtraction(blocks, time.Since(start), nil)
	}

	return blocks, nil
}

// ExtractStream performs streaming extraction from a reader
func (e *Extractor) ExtractStream(ctx context.Context, reader io.Reader) (<-chan ReasoningBlock, <-chan error) {
	blockCh := make(chan ReasoningBlock, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(blockCh)
		defer close(errCh)

		start := time.Now()
		var buffer strings.Builder

		// Read all content from reader
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			if err := ctx.Err(); err != nil {
				errCh <- gerror.Wrap(err, gerror.ErrCodeCanceled, "context cancelled during streaming").
					WithComponent("reasoning_extractor")
				if e.OnExtraction != nil {
					e.OnExtraction(nil, time.Since(start), err)
				}
				return
			}

			buffer.WriteString(scanner.Text())
			buffer.WriteString("\n")
		}

		if err := scanner.Err(); err != nil {
			errCh <- gerror.Wrap(err, gerror.ErrCodeInternal, "stream reading error").
				WithComponent("reasoning_extractor")
			if e.OnExtraction != nil {
				e.OnExtraction(nil, time.Since(start), err)
			}
			return
		}

		// Extract reasoning blocks from accumulated content
		content := buffer.String()
		if content != "" {
			blocks, err := e.Extract(ctx, content)
			if err != nil {
				errCh <- err
				if e.OnExtraction != nil {
					e.OnExtraction(nil, time.Since(start), err)
				}
				return
			}

			// Send all blocks
			for _, block := range blocks {
				select {
				case blockCh <- block:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}

			if e.OnExtraction != nil {
				e.OnExtraction(blocks, time.Since(start), nil)
			}
		}
	}()

	return blockCh, errCh
}

// ExtractFromContent extracts reasoning blocks from raw content
func (e *Extractor) ExtractFromContent(content string) []*interfaces.ReasoningBlock {
	var blocks []*interfaces.ReasoningBlock

	for _, pattern := range e.patterns {
		extracted := e.extractWithPattern(content, pattern)
		blocks = append(blocks, extracted...)
	}

	return blocks
}

// extractWithPattern extracts blocks using a specific pattern
func (e *Extractor) extractWithPattern(content string, pattern Pattern) []*interfaces.ReasoningBlock {
	var blocks []*interfaces.ReasoningBlock

	startIdx := 0
	for {
		// Find start of block
		idx := strings.Index(content[startIdx:], pattern.StartTag)
		if idx == -1 {
			break
		}
		idx += startIdx

		// Find end of opening tag
		tagEnd := strings.Index(content[idx:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd += idx + 1

		// Find closing tag
		endIdx := strings.Index(content[tagEnd:], pattern.EndTag)
		if endIdx == -1 {
			break
		}
		endIdx += tagEnd

		// Extract content
		blockContent := content[tagEnd:endIdx]

		// Create reasoning block
		block := &interfaces.ReasoningBlock{
			ID:         uuid.New().String(),
			Type:       strings.TrimSuffix(strings.TrimPrefix(pattern.Name, "<"), "_tags"),
			Content:    strings.TrimSpace(blockContent),
			Timestamp:  time.Now(),
			TokenCount: e.estimateTokens(blockContent, pattern.TokenScale),
		}

		blocks = append(blocks, block)

		// Move past this block
		startIdx = endIdx + len(pattern.EndTag)
	}

	return blocks
}

// estimateTokens estimates token count for content
func (e *Extractor) estimateTokens(content string, scale float64) int {
	return int(float64(len(content)) * scale)
}

// StreamExtractor handles reasoning extraction from streaming responses
type StreamExtractor struct {
	extractor        *Extractor
	buffer           *strings.Builder
	inReasoningBlock bool
	currentPattern   *Pattern
	blockStartTime   time.Time
	reasoningChan    chan *interfaces.ReasoningBlock
}

// NewStreamExtractor creates a new stream extractor
func NewStreamExtractor(extractor *Extractor) *StreamExtractor {
	return &StreamExtractor{
		extractor:     extractor,
		buffer:        &strings.Builder{},
		reasoningChan: make(chan *interfaces.ReasoningBlock, 100),
	}
}

// ProcessChunk processes a streaming chunk for reasoning
func (s *StreamExtractor) ProcessChunk(ctx context.Context, chunk string) {
	s.buffer.WriteString(chunk)

	// Process buffer for complete blocks
	for {
		bufferContent := s.buffer.String()

		if !s.inReasoningBlock {
			// Look for start of a reasoning block
			found := false
			for _, pattern := range s.extractor.patterns {
				if strings.Contains(bufferContent, pattern.StartTag) {
					s.startReasoningBlock(ctx, bufferContent, &pattern)
					found = true
					break
				}
			}
			if !found {
				break // No more blocks to process
			}
		} else if s.currentPattern != nil && strings.Contains(bufferContent, s.currentPattern.EndTag) {
			// Process end of current block
			s.endReasoningBlock(ctx, bufferContent)
			// Continue loop to check for more blocks
		} else {
			break // Waiting for more content
		}
	}
}

// startReasoningBlock handles the start of a reasoning block
func (s *StreamExtractor) startReasoningBlock(ctx context.Context, content string, pattern *Pattern) {
	s.inReasoningBlock = true
	s.currentPattern = pattern
	s.blockStartTime = time.Now()

	// Find the start of the tag
	idx := strings.Index(content, pattern.StartTag)
	if idx >= 0 {
		// Keep only content from tag onwards
		s.buffer.Reset()
		s.buffer.WriteString(content[idx:])
	}
}

// endReasoningBlock handles the end of a reasoning block
func (s *StreamExtractor) endReasoningBlock(ctx context.Context, content string) {
	if s.currentPattern == nil {
		return
	}

	// Extract the complete block
	endIdx := strings.Index(content, s.currentPattern.EndTag)
	if endIdx < 0 {
		return
	}

	fullBlock := content[:endIdx+len(s.currentPattern.EndTag)]

	// Extract blocks
	blocks := s.extractor.extractWithPattern(fullBlock, *s.currentPattern)

	// Send blocks to channel
	for _, block := range blocks {
		block.Duration = time.Since(s.blockStartTime)
		select {
		case s.reasoningChan <- block:
		case <-ctx.Done():
			return
		}
	}

	// Keep any content after the closing tag
	remaining := content[endIdx+len(s.currentPattern.EndTag):]

	// Reset state
	s.inReasoningBlock = false
	s.currentPattern = nil
	s.buffer.Reset()

	if remaining != "" {
		s.buffer.WriteString(remaining)
	}
}

// Channel returns the reasoning block channel
func (s *StreamExtractor) Channel() <-chan *interfaces.ReasoningBlock {
	return s.reasoningChan
}

// Close closes the stream extractor
func (s *StreamExtractor) Close() {
	close(s.reasoningChan)
}
