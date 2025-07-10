// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// StreamEvent represents an event in the reasoning stream
type StreamEvent struct {
	Type      StreamEventType        `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StreamEventType categorizes stream events
type StreamEventType string

const (
	StreamEventThinkingStart    StreamEventType = "thinking_start"
	StreamEventThinkingUpdate   StreamEventType = "thinking_update"
	StreamEventThinkingComplete StreamEventType = "thinking_complete"
	StreamEventContentChunk     StreamEventType = "content_chunk"
	StreamEventToolCall         StreamEventType = "tool_call"
	StreamEventDecisionPoint    StreamEventType = "decision_point"
	StreamEventConfidenceUpdate StreamEventType = "confidence_update"
	StreamEventError            StreamEventType = "error"
	StreamEventInterrupted      StreamEventType = "interrupted"
)

// ThinkingUpdate represents a thinking progress update
type ThinkingUpdate struct {
	BlockID     string       `json:"block_id"`
	Type        ThinkingType `json:"type"`
	Content     string       `json:"content"`
	Confidence  float64      `json:"confidence"`
	TokensSoFar int          `json:"tokens_so_far"`
	IsPartial   bool         `json:"is_partial"`
}

// ContentChunk represents a piece of response content
type ContentChunk struct {
	Content  string `json:"content"`
	Position int    `json:"position"`
	IsFinal  bool   `json:"is_final"`
}

// ReasoningStreamer handles real-time reasoning streaming
type ReasoningStreamer struct {
	parser        *ThinkingBlockParser
	chainBuilder  *ReasoningChainBuilder
	eventChan     chan StreamEvent
	errorChan     chan error
	interruptChan chan struct{}

	// State management
	mu              sync.RWMutex
	currentBlock    *ThinkingBlock
	partialContent  strings.Builder
	inThinkingBlock bool
	blockStartTime  time.Time
	totalTokens     int

	// Configuration
	config  StreamConfig
	// metrics *observability.MetricsRegistry // TODO: Update to use MetricsRegistry
}

// StreamConfig configures the reasoning streamer
type StreamConfig struct {
	BufferSize       int           `json:"buffer_size"`
	FlushInterval    time.Duration `json:"flush_interval"`
	MaxBlockSize     int           `json:"max_block_size"`
	EnableMetrics    bool          `json:"enable_metrics"`
	InterruptTimeout time.Duration `json:"interrupt_timeout"`
}

// DefaultStreamConfig returns default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:       1024,
		FlushInterval:    100 * time.Millisecond,
		MaxBlockSize:     10000,
		EnableMetrics:    true,
		InterruptTimeout: 5 * time.Second,
	}
}

// NewReasoningStreamer creates a new reasoning streamer
func NewReasoningStreamer(parser *ThinkingBlockParser, chainBuilder *ReasoningChainBuilder) *ReasoningStreamer {
	config := DefaultStreamConfig()
	return &ReasoningStreamer{
		parser:        parser,
		chainBuilder:  chainBuilder,
		eventChan:     make(chan StreamEvent, config.BufferSize),
		errorChan:     make(chan error, 10),
		interruptChan: make(chan struct{}, 1),
		config:        config,
		// metrics:       metrics, // TODO: Add metrics registry
	}
}

// Stream processes a response stream and emits reasoning events
func (rs *ReasoningStreamer) Stream(ctx context.Context, reader io.Reader) error {
	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start background processors
	var wg sync.WaitGroup
	wg.Add(2)

	// Event flusher
	go func() {
		defer wg.Done()
		rs.eventFlusher(streamCtx)
	}()

	// Interrupt handler
	go func() {
		defer wg.Done()
		rs.interruptHandler(streamCtx, cancel)
	}()

	// Process stream
	err := rs.processStream(streamCtx, reader)

	// Wait for processors to finish
	cancel()
	wg.Wait()

	// Close channels
	close(rs.eventChan)
	close(rs.errorChan)

	return err
}

// processStream reads and processes the input stream
func (rs *ReasoningStreamer) processStream(ctx context.Context, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(rs.customSplitter) // Custom splitter for better control

	position := 0

	for scanner.Scan() {
		if ctx.Err() != nil {
			return gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "stream processing cancelled").
				WithComponent("reasoning_streamer")
		}

		chunk := scanner.Text()

		// Process chunk
		if err := rs.processChunk(ctx, chunk, position); err != nil {
			rs.sendError(err)
			// TODO: Update to use MetricsRegistry
			// if rs.config.EnableMetrics {
			// 	rs.metrics.RecordCounter("reasoning_stream_errors", 1)
			// }
		}

		position += len(chunk)
	}

	// Finalize any partial content
	rs.finalizeStream(ctx)

	if err := scanner.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "stream scanning error").
			WithComponent("reasoning_streamer")
	}

	return nil
}

// customSplitter provides intelligent stream splitting
func (rs *ReasoningStreamer) customSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Look for natural boundaries
	boundaries := []string{
		"</thinking>",
		"\n\n",
		". ",
		"\\n",
	}

	for _, boundary := range boundaries {
		if idx := strings.Index(string(data), boundary); idx >= 0 {
			return idx + len(boundary), data[:idx+len(boundary)], nil
		}
	}

	// If no boundary found, use default behavior
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// processChunk processes a single chunk of the stream
func (rs *ReasoningStreamer) processChunk(ctx context.Context, chunk string, position int) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Check for thinking block markers
	if strings.Contains(chunk, "<thinking") {
		rs.startThinkingBlock(chunk)
	} else if strings.Contains(chunk, "</thinking>") {
		rs.endThinkingBlock(ctx, chunk)
	} else if rs.inThinkingBlock {
		rs.updateThinkingBlock(chunk)
	} else {
		// Regular content
		rs.sendContentChunk(chunk, position)
	}

	return nil
}

// startThinkingBlock handles the start of a thinking block
func (rs *ReasoningStreamer) startThinkingBlock(chunk string) {
	rs.inThinkingBlock = true
	rs.blockStartTime = time.Now()
	rs.partialContent.Reset()

	// Extract type hint if present
	typePattern := `<thinking(?:\s+type="([^"]+)")?>`
	matches := findStringSubmatch(typePattern, chunk)

	blockID := generateID()
	thinkingType := ThinkingTypeAnalysis // default
	if len(matches) > 1 && matches[1] != "" {
		thinkingType = ThinkingType(matches[1])
	}

	// Send thinking start event
	rs.sendEvent(StreamEvent{
		Type:      StreamEventThinkingStart,
		Timestamp: time.Now(),
		Data: ThinkingUpdate{
			BlockID:   blockID,
			Type:      thinkingType,
			IsPartial: true,
		},
	})

	// Initialize current block
	rs.currentBlock = &ThinkingBlock{
		ID:        blockID,
		Type:      thinkingType,
		Timestamp: rs.blockStartTime,
	}

	// Add any content after the opening tag
	afterTag := strings.SplitN(chunk, ">", 2)
	if len(afterTag) > 1 {
		rs.partialContent.WriteString(afterTag[1])
	}
}

// updateThinkingBlock handles thinking block content updates
func (rs *ReasoningStreamer) updateThinkingBlock(chunk string) {
	rs.partialContent.WriteString(chunk)

	// Check size limit
	if rs.partialContent.Len() > rs.config.MaxBlockSize {
		rs.sendError(gerror.New(gerror.ErrCodeResourceLimit, "thinking block too large", nil).
			WithComponent("reasoning_streamer").
			WithDetails("size", rs.partialContent.Len()))
		return
	}

	// Extract current confidence if visible
	confidence := rs.parser.extractConfidence(rs.partialContent.String())

	// Send update
	rs.sendEvent(StreamEvent{
		Type:      StreamEventThinkingUpdate,
		Timestamp: time.Now(),
		Data: ThinkingUpdate{
			BlockID:     rs.currentBlock.ID,
			Type:        rs.currentBlock.Type,
			Content:     rs.partialContent.String(),
			Confidence:  confidence,
			TokensSoFar: estimateTokens(rs.partialContent.String()),
			IsPartial:   true,
		},
	})

	// Update confidence if changed significantly
	if abs(confidence-rs.currentBlock.Confidence) > 0.1 {
		rs.currentBlock.Confidence = confidence
		rs.sendEvent(StreamEvent{
			Type:      StreamEventConfidenceUpdate,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"block_id":   rs.currentBlock.ID,
				"confidence": confidence,
			},
		})
	}
}

// endThinkingBlock handles the end of a thinking block
func (rs *ReasoningStreamer) endThinkingBlock(ctx context.Context, chunk string) {
	// Add any content before the closing tag
	beforeTag := strings.Split(chunk, "</thinking>")[0]
	rs.partialContent.WriteString(beforeTag)

	// Finalize block
	rs.currentBlock.Content = rs.partialContent.String()
	rs.currentBlock.Duration = time.Since(rs.blockStartTime)
	rs.currentBlock.TokenCount = estimateTokens(rs.currentBlock.Content)

	// Parse structured data
	if structured, err := rs.parser.structureExtractor.Extract(ctx, rs.currentBlock.Type, rs.currentBlock.Content); err == nil {
		rs.currentBlock.StructuredData = structured
	}

	// Extract decision points
	rs.parser.extractDecisionPoints([]*ThinkingBlock{rs.currentBlock})

	// Add to chain
	if err := rs.chainBuilder.AddBlock(rs.currentBlock); err != nil {
		rs.sendError(err)
	}

	// Send completion event
	rs.sendEvent(StreamEvent{
		Type:      StreamEventThinkingComplete,
		Timestamp: time.Now(),
		Data: ThinkingUpdate{
			BlockID:     rs.currentBlock.ID,
			Type:        rs.currentBlock.Type,
			Content:     rs.currentBlock.Content,
			Confidence:  rs.currentBlock.Confidence,
			TokensSoFar: rs.currentBlock.TokenCount,
			IsPartial:   false,
		},
	})

	// Send decision point events
	for _, dp := range rs.currentBlock.DecisionPoints {
		rs.sendEvent(StreamEvent{
			Type:      StreamEventDecisionPoint,
			Timestamp: time.Now(),
			Data:      dp,
			Metadata: map[string]interface{}{
				"block_id": rs.currentBlock.ID,
			},
		})
	}

	// Reset state
	rs.inThinkingBlock = false
	rs.currentBlock = nil
	rs.partialContent.Reset()

	// Process any remaining content after closing tag
	afterTag := strings.SplitN(chunk, "</thinking>", 2)
	if len(afterTag) > 1 && afterTag[1] != "" {
		rs.sendContentChunk(afterTag[1], 0)
	}
}

// sendContentChunk sends a regular content chunk event
func (rs *ReasoningStreamer) sendContentChunk(content string, position int) {
	rs.sendEvent(StreamEvent{
		Type:      StreamEventContentChunk,
		Timestamp: time.Now(),
		Data: ContentChunk{
			Content:  content,
			Position: position,
			IsFinal:  false,
		},
	})
}

// sendEvent sends an event to the channel
func (rs *ReasoningStreamer) sendEvent(event StreamEvent) {
	select {
	case rs.eventChan <- event:
		// TODO: Update to use MetricsRegistry
		// if rs.config.EnableMetrics {
		// 	rs.metrics.RecordCounter("reasoning_stream_events", 1,
		// 		"type", string(event.Type))
		// }
	default:
		// Channel full, log warning
		logger := observability.GetLogger(context.Background())
		logger.WarnContext(context.Background(), "Event channel full, dropping event",
			"type", event.Type)
	}
}

// sendError sends an error event
func (rs *ReasoningStreamer) sendError(err error) {
	select {
	case rs.errorChan <- err:
	default:
		logger := observability.GetLogger(context.Background())
		logger.WithError(err).ErrorContext(context.Background(), "Error channel full")
	}

	rs.sendEvent(StreamEvent{
		Type:      StreamEventError,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"error": err.Error(),
		},
	})
}

// eventFlusher periodically flushes events
func (rs *ReasoningStreamer) eventFlusher(ctx context.Context) {
	ticker := time.NewTicker(rs.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Flush logic if needed
		}
	}
}

// interruptHandler handles interruption requests
func (rs *ReasoningStreamer) interruptHandler(ctx context.Context, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
		return
	case <-rs.interruptChan:
		logger := observability.GetLogger(ctx)
		logger.InfoContext(ctx, "Reasoning stream interrupted")

		// Send interrupt event
		rs.sendEvent(StreamEvent{
			Type:      StreamEventInterrupted,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"reason": "user_interrupt",
			},
		})

		// Cancel stream processing
		cancel()

		// Give time for cleanup
		time.Sleep(100 * time.Millisecond)
	}
}

// finalizeStream completes any partial processing
func (rs *ReasoningStreamer) finalizeStream(ctx context.Context) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Handle any unclosed thinking block
	if rs.inThinkingBlock {
		logger := observability.GetLogger(ctx)
		logger.WarnContext(ctx, "Unclosed thinking block at stream end")
		rs.endThinkingBlock(ctx, "")
	}

	// Build final chain
	chain, err := rs.chainBuilder.Build(ctx)
	if err != nil {
		rs.sendError(err)
		return
	}

	// Send final content chunk
	rs.sendEvent(StreamEvent{
		Type:      StreamEventContentChunk,
		Timestamp: time.Now(),
		Data: ContentChunk{
			Content:  "",
			Position: -1,
			IsFinal:  true,
		},
		Metadata: map[string]interface{}{
			"chain_id":         chain.ID,
			"total_blocks":     len(chain.Blocks),
			"final_confidence": chain.FinalConfidence,
			"total_tokens":     chain.TotalTokens,
		},
	})
}

// Interrupt signals the streamer to stop processing
func (rs *ReasoningStreamer) Interrupt() {
	select {
	case rs.interruptChan <- struct{}{}:
	default:
		// Already interrupting
	}
}

// EventChannel returns the event channel for consumers
func (rs *ReasoningStreamer) EventChannel() <-chan StreamEvent {
	return rs.eventChan
}

// ErrorChannel returns the error channel for consumers
func (rs *ReasoningStreamer) ErrorChannel() <-chan error {
	return rs.errorChan
}

// GetChain returns the current reasoning chain (may be incomplete)
func (rs *ReasoningStreamer) GetChain(ctx context.Context) (*ReasoningChainEnhanced, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	return rs.chainBuilder.Build(ctx)
}

// StreamProcessor processes reasoning stream events for UI display
type StreamProcessor struct {
	handlers map[StreamEventType]StreamEventHandler
	mu       sync.RWMutex
}

// StreamEventHandler handles a specific type of stream event
type StreamEventHandler func(event StreamEvent) error

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		handlers: make(map[StreamEventType]StreamEventHandler),
	}
}

// RegisterHandler registers a handler for an event type
func (sp *StreamProcessor) RegisterHandler(eventType StreamEventType, handler StreamEventHandler) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.handlers[eventType] = handler
}

// ProcessEvent processes a single event
func (sp *StreamProcessor) ProcessEvent(event StreamEvent) error {
	sp.mu.RLock()
	handler, exists := sp.handlers[event.Type]
	sp.mu.RUnlock()

	if !exists {
		return nil // No handler registered
	}

	return handler(event)
}

// ProcessStream processes all events from a stream
func (sp *StreamProcessor) ProcessStream(ctx context.Context, eventChan <-chan StreamEvent, errorChan <-chan error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-eventChan:
			if !ok {
				return nil // Channel closed
			}

			if err := sp.ProcessEvent(event); err != nil {
				logger := observability.GetLogger(ctx)
				logger.WithError(err).ErrorContext(ctx, "Failed to process stream event",
					"type", event.Type)
			}

		case err, ok := <-errorChan:
			if !ok {
				continue
			}

			logger := observability.GetLogger(ctx)
			logger.WithError(err).ErrorContext(ctx, "Stream error")
		}
	}
}

// Helper functions

// findStringSubmatch is a helper for regex matching
func findStringSubmatch(pattern, text string) []string {
	// Implementation would use regexp package
	// Simplified for example
	return []string{}
}

// estimateTokens provides a rough token count estimate
func estimateTokens(text string) int {
	// Rough estimate: 1 token per 4 characters
	return len(text) / 4
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("block_%d", time.Now().UnixNano())
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
