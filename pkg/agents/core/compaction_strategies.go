// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/observability"
)

// SummarizationStrategy uses LLM to summarize conversation segments
type SummarizationStrategy struct {
	summarizer  MessageSummarizer
	minMessages int
	targetRatio float64
}

// MessageSummarizer interface for summarization
type MessageSummarizer interface {
	Summarize(ctx context.Context, messages []WindowMessage) (string, int, error)
}

// NewSummarizationStrategy creates a new summarization strategy
func NewSummarizationStrategy(summarizer MessageSummarizer) *SummarizationStrategy {
	return &SummarizationStrategy{
		summarizer:  summarizer,
		minMessages: 10,
		targetRatio: 0.3, // Target 30% of original size
	}
}

// Name returns the strategy name
func (s *SummarizationStrategy) Name() string {
	return "summarization"
}

// Priority returns the strategy priority
func (s *SummarizationStrategy) Priority() int {
	return 100 // Highest priority
}

// CanApply checks if strategy can be applied
func (s *SummarizationStrategy) CanApply(window *ContextWindow, analysis *WindowAnalysis) bool {
	return analysis.CompressibleMessages >= s.minMessages &&
		analysis.EstimatedSavings > 1000
}

// Apply applies the summarization strategy
func (s *SummarizationStrategy) Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	logger := observability.GetLogger(ctx)

	// Group messages for summarization
	groups := s.groupMessages(window.Messages)

	result := &CompactedWindow{
		Messages: make([]WindowMessage, 0),
		Strategy: s.Name(),
	}

	totalOriginal := 0
	totalCompacted := 0

	for _, group := range groups {
		if len(group) < s.minMessages || !s.shouldSummarize(group) {
			// Keep messages as-is
			result.Messages = append(result.Messages, group...)
			for _, msg := range group {
				totalOriginal += msg.TokenCount
				totalCompacted += msg.TokenCount
			}
			continue
		}

		// Summarize group
		summary, tokenCount, err := s.summarizer.Summarize(ctx, group)
		if err != nil {
			logger.WarnContext(ctx, "Failed to summarize message group",
				"error", err,
				"group_size", len(group))
			// Keep original on error
			result.Messages = append(result.Messages, group...)
			for _, msg := range group {
				totalOriginal += msg.TokenCount
				totalCompacted += msg.TokenCount
			}
			continue
		}

		// Calculate token savings
		originalTokens := 0
		for _, msg := range group {
			originalTokens += msg.TokenCount
		}
		totalOriginal += originalTokens
		totalCompacted += tokenCount

		// Create summary message
		summaryMsg := WindowMessage{
			ID:           generateMessageID(),
			Role:         "assistant",
			Content:      fmt.Sprintf("[Summary of %d messages]\n%s", len(group), summary),
			TokenCount:   tokenCount,
			Timestamp:    time.Now(),
			Compressible: false, // Don't compress summaries
			Priority:     PriorityNormal,
			Metadata: map[string]interface{}{
				"summarized_count":  len(group),
				"original_tokens":   originalTokens,
				"compression_ratio": float64(tokenCount) / float64(originalTokens),
			},
		}

		result.Messages = append(result.Messages, summaryMsg)
		result.Summary += fmt.Sprintf("Summarized %d messages into %d tokens. ", len(group), tokenCount)
	}

	result.TotalTokens = totalCompacted
	result.TokensSaved = totalOriginal - totalCompacted

	return result, nil
}

// groupMessages groups messages for summarization
func (s *SummarizationStrategy) groupMessages(messages []WindowMessage) [][]WindowMessage {
	groups := make([][]WindowMessage, 0)
	currentGroup := make([]WindowMessage, 0)

	for _, msg := range messages {
		// Start new group on high-priority or recent messages
		if msg.Priority >= PriorityHigh ||
			time.Since(msg.Timestamp) < 10*time.Minute ||
			len(currentGroup) >= 20 {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = make([]WindowMessage, 0)
			}
		}

		currentGroup = append(currentGroup, msg)
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// shouldSummarize determines if a group should be summarized
func (s *SummarizationStrategy) shouldSummarize(group []WindowMessage) bool {
	// Don't summarize if contains critical messages
	for _, msg := range group {
		if msg.Priority == PriorityCritical {
			return false
		}
		if msg.ReasoningBlockID != nil {
			return false // Keep reasoning blocks
		}
	}

	// Check age - don't summarize very recent messages
	for _, msg := range group {
		if time.Since(msg.Timestamp) < 5*time.Minute {
			return false
		}
	}

	return true
}

// PriorityFilterStrategy removes low-priority messages
type PriorityFilterStrategy struct {
	threshold         MessagePriority
	minMessagesToKeep int
}

// NewPriorityFilterStrategy creates a new priority filter
func NewPriorityFilterStrategy() *PriorityFilterStrategy {
	return &PriorityFilterStrategy{
		threshold:         PriorityNormal,
		minMessagesToKeep: 10,
	}
}

// Name returns the strategy name
func (p *PriorityFilterStrategy) Name() string {
	return "priority_filter"
}

// Priority returns the strategy priority
func (p *PriorityFilterStrategy) Priority() int {
	return 80
}

// CanApply checks if strategy can be applied
func (p *PriorityFilterStrategy) CanApply(window *ContextWindow, analysis *WindowAnalysis) bool {
	return analysis.LowPriorityMessages > 5 &&
		len(window.Messages) > p.minMessagesToKeep*2
}

// Apply applies the priority filter strategy
func (p *PriorityFilterStrategy) Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	// Sort messages by priority and timestamp
	sorted := make([]WindowMessage, len(window.Messages))
	copy(sorted, window.Messages)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	// Keep high-priority and recent messages
	kept := make([]WindowMessage, 0)
	removed := 0
	removedTokens := 0

	for _, msg := range sorted {
		if msg.Priority >= p.threshold ||
			len(kept) < p.minMessagesToKeep ||
			time.Since(msg.Timestamp) < 5*time.Minute {
			kept = append(kept, msg)
		} else {
			removed++
			removedTokens += msg.TokenCount
		}
	}

	// Re-sort by timestamp
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].Timestamp.Before(kept[j].Timestamp)
	})

	totalTokens := 0
	for _, msg := range kept {
		totalTokens += msg.TokenCount
	}

	return &CompactedWindow{
		Messages:    kept,
		TotalTokens: totalTokens,
		TokensSaved: removedTokens,
		Strategy:    p.Name(),
		Summary:     fmt.Sprintf("Removed %d low-priority messages", removed),
	}, nil
}

// RedundancyRemovalStrategy removes redundant messages
type RedundancyRemovalStrategy struct {
	similarityThreshold float64
	detector            SimilarityDetector
}

// SimilarityDetector interface for detecting similar messages
type SimilarityDetector interface {
	Similarity(msg1, msg2 WindowMessage) float64
}

// NewRedundancyRemovalStrategy creates a new redundancy removal strategy
func NewRedundancyRemovalStrategy() *RedundancyRemovalStrategy {
	return &RedundancyRemovalStrategy{
		similarityThreshold: 0.85,
		detector:            &SimpleSimilarityDetector{},
	}
}

// Name returns the strategy name
func (r *RedundancyRemovalStrategy) Name() string {
	return "redundancy_removal"
}

// Priority returns the strategy priority
func (r *RedundancyRemovalStrategy) Priority() int {
	return 70
}

// CanApply checks if strategy can be applied
func (r *RedundancyRemovalStrategy) CanApply(window *ContextWindow, analysis *WindowAnalysis) bool {
	return analysis.RedundantMessages > 3
}

// Apply applies the redundancy removal strategy
func (r *RedundancyRemovalStrategy) Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	kept := make([]WindowMessage, 0)
	removed := 0
	removedTokens := 0

	// Track seen content
	seen := make(map[string]bool)

	for _, msg := range window.Messages {
		// Check if similar message already kept
		isDuplicate := false

		for _, keptMsg := range kept {
			similarity := r.detector.Similarity(msg, keptMsg)
			if similarity > r.similarityThreshold {
				isDuplicate = true
				break
			}
		}

		// Simple hash check for exact duplicates
		contentHash := hashContent(msg.Content)
		if seen[contentHash] {
			isDuplicate = true
		}

		if !isDuplicate {
			kept = append(kept, msg)
			seen[contentHash] = true
		} else {
			removed++
			removedTokens += msg.TokenCount
		}
	}

	totalTokens := 0
	for _, msg := range kept {
		totalTokens += msg.TokenCount
	}

	return &CompactedWindow{
		Messages:    kept,
		TotalTokens: totalTokens,
		TokensSaved: removedTokens,
		Strategy:    r.Name(),
		Summary:     fmt.Sprintf("Removed %d redundant messages", removed),
	}, nil
}

// SimpleSimilarityDetector provides basic similarity detection
type SimpleSimilarityDetector struct{}

// Similarity calculates similarity between messages
func (s *SimpleSimilarityDetector) Similarity(msg1, msg2 WindowMessage) float64 {
	// Simple word overlap similarity
	words1 := strings.Fields(strings.ToLower(msg1.Content))
	words2 := strings.Fields(strings.ToLower(msg2.Content))

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// Create word sets
	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	// Count overlaps
	overlap := 0
	for _, w := range words2 {
		if set1[w] {
			overlap++
		}
	}

	// Jaccard similarity
	union := len(set1) + len(words2) - overlap
	return float64(overlap) / float64(union)
}

// TemporalDecayStrategy removes old messages with decay function
type TemporalDecayStrategy struct {
	maxAge          time.Duration
	decayFunction   func(age time.Duration) float64
	importanceBoost map[MessagePriority]float64
}

// NewTemporalDecayStrategy creates a new temporal decay strategy
func NewTemporalDecayStrategy() *TemporalDecayStrategy {
	return &TemporalDecayStrategy{
		maxAge: 1 * time.Hour,
		decayFunction: func(age time.Duration) float64 {
			// Exponential decay
			hours := age.Hours()
			return exp(-hours / 2.0) // Half-life of 2 hours
		},
		importanceBoost: map[MessagePriority]float64{
			PriorityLow:      0.5,
			PriorityNormal:   1.0,
			PriorityHigh:     2.0,
			PriorityCritical: 10.0,
		},
	}
}

// Name returns the strategy name
func (t *TemporalDecayStrategy) Name() string {
	return "temporal_decay"
}

// Priority returns the strategy priority
func (t *TemporalDecayStrategy) Priority() int {
	return 60
}

// CanApply checks if strategy can be applied
func (t *TemporalDecayStrategy) CanApply(window *ContextWindow, analysis *WindowAnalysis) bool {
	// Check if we have old messages
	oldCount := 0
	for _, msg := range window.Messages {
		if time.Since(msg.Timestamp) > 30*time.Minute {
			oldCount++
		}
	}
	return oldCount > 5
}

// Apply applies the temporal decay strategy
func (t *TemporalDecayStrategy) Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	// Score each message
	type scoredMessage struct {
		msg   WindowMessage
		score float64
	}

	scored := make([]scoredMessage, 0, len(window.Messages))

	for _, msg := range window.Messages {
		age := time.Since(msg.Timestamp)

		// Calculate base score from age
		score := t.decayFunction(age)

		// Apply importance boost
		if boost, ok := t.importanceBoost[msg.Priority]; ok {
			score *= boost
		}

		// Boost reasoning blocks
		if msg.ReasoningBlockID != nil {
			score *= 3.0
		}

		scored = append(scored, scoredMessage{msg: msg, score: score})
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Keep messages above threshold
	threshold := 0.3
	kept := make([]WindowMessage, 0)
	removed := 0
	removedTokens := 0

	for _, sm := range scored {
		if sm.score >= threshold || len(kept) < 10 {
			kept = append(kept, sm.msg)
		} else {
			removed++
			removedTokens += sm.msg.TokenCount
		}
	}

	// Re-sort by timestamp
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].Timestamp.Before(kept[j].Timestamp)
	})

	totalTokens := 0
	for _, msg := range kept {
		totalTokens += msg.TokenCount
	}

	return &CompactedWindow{
		Messages:    kept,
		TotalTokens: totalTokens,
		TokensSaved: removedTokens,
		Strategy:    t.Name(),
		Summary:     fmt.Sprintf("Removed %d old messages using temporal decay", removed),
	}, nil
}

// OllamaCompactionStrategy uses Ollama for intelligent summarization
type OllamaCompactionStrategy struct {
	client    OllamaClient
	model     string
	chunkSize int
}

// OllamaClient interface for Ollama integration
type OllamaClient interface {
	Generate(ctx context.Context, prompt string, model string) (string, int, error)
}

// NewOllamaCompactionStrategy creates Ollama-based compaction
func NewOllamaCompactionStrategy(client OllamaClient, model string) *OllamaCompactionStrategy {
	return &OllamaCompactionStrategy{
		client:    client,
		model:     model,
		chunkSize: 10,
	}
}

// Name returns the strategy name
func (o *OllamaCompactionStrategy) Name() string {
	return "ollama_intelligent"
}

// Priority returns the strategy priority
func (o *OllamaCompactionStrategy) Priority() int {
	return 90
}

// CanApply checks if strategy can be applied
func (o *OllamaCompactionStrategy) CanApply(window *ContextWindow, analysis *WindowAnalysis) bool {
	return o.client != nil && len(window.Messages) > 20
}

// Apply applies Ollama-based compaction
func (o *OllamaCompactionStrategy) Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	logger := observability.GetLogger(ctx)

	// Group messages into chunks
	chunks := o.createChunks(window.Messages)

	result := &CompactedWindow{
		Messages: make([]WindowMessage, 0),
		Strategy: o.Name(),
	}

	totalOriginal := 0
	totalCompacted := 0

	for i, chunk := range chunks {
		// Skip recent chunks
		if o.isRecentChunk(chunk) {
			result.Messages = append(result.Messages, chunk...)
			for _, msg := range chunk {
				totalOriginal += msg.TokenCount
				totalCompacted += msg.TokenCount
			}
			continue
		}

		// Create summarization prompt
		prompt := o.createSummarizationPrompt(chunk)

		// Call Ollama
		summary, tokenCount, err := o.client.Generate(ctx, prompt, o.model)
		if err != nil {
			logger.WarnContext(ctx, "Ollama summarization failed",
				"error", err,
				"chunk", i)
			// Keep original on error
			result.Messages = append(result.Messages, chunk...)
			for _, msg := range chunk {
				totalOriginal += msg.TokenCount
				totalCompacted += msg.TokenCount
			}
			continue
		}

		// Calculate savings
		originalTokens := 0
		for _, msg := range chunk {
			originalTokens += msg.TokenCount
		}
		totalOriginal += originalTokens
		totalCompacted += tokenCount

		// Create summary message
		summaryMsg := WindowMessage{
			ID:           generateMessageID(),
			Role:         "system",
			Content:      fmt.Sprintf("[Ollama Summary of conversation segment %d]\n%s", i+1, summary),
			TokenCount:   tokenCount,
			Timestamp:    time.Now(),
			Compressible: false,
			Priority:     PriorityNormal,
			Metadata: map[string]interface{}{
				"chunk_index":       i,
				"original_messages": len(chunk),
				"original_tokens":   originalTokens,
				"model":             o.model,
			},
		}

		result.Messages = append(result.Messages, summaryMsg)
	}

	result.TotalTokens = totalCompacted
	result.TokensSaved = totalOriginal - totalCompacted
	result.Summary = fmt.Sprintf("Used Ollama to compress %d chunks", len(chunks))

	return result, nil
}

// createChunks groups messages into chunks for summarization
func (o *OllamaCompactionStrategy) createChunks(messages []WindowMessage) [][]WindowMessage {
	chunks := make([][]WindowMessage, 0)

	for i := 0; i < len(messages); i += o.chunkSize {
		end := i + o.chunkSize
		if end > len(messages) {
			end = len(messages)
		}
		chunks = append(chunks, messages[i:end])
	}

	return chunks
}

// isRecentChunk checks if chunk contains recent messages
func (o *OllamaCompactionStrategy) isRecentChunk(chunk []WindowMessage) bool {
	for _, msg := range chunk {
		if time.Since(msg.Timestamp) < 15*time.Minute {
			return true
		}
	}
	return false
}

// createSummarizationPrompt creates prompt for Ollama
func (o *OllamaCompactionStrategy) createSummarizationPrompt(chunk []WindowMessage) string {
	var sb strings.Builder

	sb.WriteString("Please provide a concise summary of the following conversation segment.\n")
	sb.WriteString("Preserve key decisions, important information, and action items.\n")
	sb.WriteString("Keep the summary under 200 words.\n\n")
	sb.WriteString("Conversation:\n")

	for _, msg := range chunk {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04"),
			msg.Role,
			msg.Content))
	}

	sb.WriteString("\nSummary:")

	return sb.String()
}

// Helper functions

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func hashContent(content string) string {
	// Simple hash for deduplication
	// In production, use proper hashing
	return fmt.Sprintf("%d_%d", len(content), strings.Count(content, " "))
}

// exp is a simple exponential function
func exp(x float64) float64 {
	// In production, use math.Exp
	if x >= 0 {
		return 1.0
	}
	// Simple approximation
	return 1.0 + x + (x*x)/2 + (x*x*x)/6
}
