// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// ContextManager handles context window management for agents
type ContextManager struct {
	agentID       string
	maxTokens     int
	currentTokens int
	resetStrategy string // "truncate" or "summarize"

	// Message history
	messages     []ContextMessage
	compressions int // Number of times context has been compressed

	// Token tracking
	totalTokensUsed int64
	avgTokensPerMsg int
	lastCompression time.Time

	// Cost tracking
	totalCost   float64
	costProfile CostEstimate
}

// ContextMessage represents a message in the context window
type ContextMessage struct {
	ID        string
	Role      string // "system", "user", "assistant"
	Content   string
	Tokens    int
	Timestamp time.Time
	Priority  int // 1-10, higher = more important to preserve
	Metadata  map[string]interface{}
}

// NewContextManager creates a new context manager for an agent
func NewContextManager(config *config.EnhancedAgentConfig, costProfile CostEstimate) *ContextManager {
	resetStrategy := "truncate"
	switch config.Type {
	case "manager":
		resetStrategy = "summarize" // Managers need to preserve context
	case "worker":
		resetStrategy = "truncate" // Workers can restart fresh
	case "specialist":
		resetStrategy = "summarize" // Specialists need context for expertise
	}

	return &ContextManager{
		agentID:         config.ID,
		maxTokens:       config.GetEffectiveContextWindow(),
		currentTokens:   0,
		resetStrategy:   resetStrategy,
		messages:        make([]ContextMessage, 0),
		compressions:    0,
		lastCompression: time.Time{},
		costProfile:     costProfile,
	}
}

// AddMessage adds a message to the context window
func (cm *ContextManager) AddMessage(ctx context.Context, role, content string, priority int) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ContextManager").
			WithOperation("AddMessage")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ContextManager")
	ctx = observability.WithOperation(ctx, "AddMessage")

	// Estimate tokens for the message
	tokens := cm.estimateTokens(content)

	logger.DebugContext(ctx, "Adding message to context",
		"agent_id", cm.agentID,
		"role", role,
		"tokens", tokens,
		"priority", priority,
		"current_tokens", cm.currentTokens,
		"max_tokens", cm.maxTokens)

	message := ContextMessage{
		ID:        generateMessageID(),
		Role:      role,
		Content:   content,
		Tokens:    tokens,
		Timestamp: time.Now(),
		Priority:  priority,
		Metadata:  make(map[string]interface{}),
	}

	// Check if adding this message would exceed the context window
	if cm.currentTokens+tokens > cm.maxTokens {
		logger.InfoContext(ctx, "Context window limit reached, applying reset strategy",
			"agent_id", cm.agentID,
			"strategy", cm.resetStrategy,
			"current_tokens", cm.currentTokens,
			"new_message_tokens", tokens,
			"max_tokens", cm.maxTokens)

		if err := cm.manageContextOverflow(ctx, tokens); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to manage context overflow").
				WithComponent("ContextManager").
				WithOperation("AddMessage").
				WithDetails("agent_id", cm.agentID)
		}
	}

	// Add the message
	cm.messages = append(cm.messages, message)
	cm.currentTokens += tokens
	cm.totalTokensUsed += int64(tokens)

	// Update average tokens per message
	if len(cm.messages) > 0 {
		cm.avgTokensPerMsg = int(cm.totalTokensUsed) / len(cm.messages)
	}

	logger.DebugContext(ctx, "Message added to context",
		"agent_id", cm.agentID,
		"message_id", message.ID,
		"new_current_tokens", cm.currentTokens)

	return nil
}

// manageContextOverflow handles context window overflow
func (cm *ContextManager) manageContextOverflow(ctx context.Context, newMessageTokens int) error {

	switch cm.resetStrategy {
	case "truncate":
		return cm.truncateContext(ctx, newMessageTokens)
	case "summarize":
		return cm.summarizeContext(ctx, newMessageTokens)
	default:
		return cm.truncateContext(ctx, newMessageTokens) // Default fallback
	}
}

// truncateContext removes old messages to make room for new ones
func (cm *ContextManager) truncateContext(ctx context.Context, newMessageTokens int) error {
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Truncating context window",
		"agent_id", cm.agentID,
		"target_free_tokens", newMessageTokens)

	// Calculate how many tokens we need to free
	targetTokens := cm.maxTokens - newMessageTokens
	tokensToFree := cm.currentTokens - targetTokens

	// Always preserve system messages and high-priority messages
	preservedMessages := make([]ContextMessage, 0)
	removableMessages := make([]ContextMessage, 0)

	for _, msg := range cm.messages {
		if msg.Role == "system" || msg.Priority >= 8 {
			preservedMessages = append(preservedMessages, msg)
		} else {
			removableMessages = append(removableMessages, msg)
		}
	}

	// Remove oldest messages first (FIFO)
	freedTokens := 0
	finalMessages := make([]ContextMessage, 0, len(preservedMessages))
	finalMessages = append(finalMessages, preservedMessages...)

	// Add back as many removable messages as we can, starting from the newest
	for i := len(removableMessages) - 1; i >= 0 && freedTokens < tokensToFree; i-- {
		msg := removableMessages[i]
		if cm.currentTokens-freedTokens-msg.Tokens >= targetTokens {
			// This message can fit
			finalMessages = append(finalMessages, msg)
		} else {
			// Remove this message
			freedTokens += msg.Tokens
		}
	}

	// Update context
	cm.messages = finalMessages
	cm.currentTokens -= freedTokens
	cm.compressions++
	cm.lastCompression = time.Now()

	logger.InfoContext(ctx, "Context truncation completed",
		"agent_id", cm.agentID,
		"freed_tokens", freedTokens,
		"remaining_messages", len(cm.messages),
		"remaining_tokens", cm.currentTokens)

	return nil
}

// summarizeContext compresses old messages into a summary
func (cm *ContextManager) summarizeContext(ctx context.Context, newMessageTokens int) error {
	logger := observability.GetLogger(ctx)

	logger.InfoContext(ctx, "Summarizing context window",
		"agent_id", cm.agentID,
		"target_free_tokens", newMessageTokens)

	// For now, implement a simple form of summarization by keeping only recent and high-priority messages
	// In a full implementation, this would use an LLM to generate a summary

	// Calculate how many messages to keep
	_ = newMessageTokens // Reserve space for new message

	// Keep system messages, high-priority messages, and recent messages
	preservedMessages := make([]ContextMessage, 0)
	recentMessages := make([]ContextMessage, 0)
	summaryContent := make([]string, 0)

	now := time.Now()
	recentThreshold := now.Add(-30 * time.Minute) // Keep messages from last 30 minutes

	for _, msg := range cm.messages {
		if msg.Role == "system" || msg.Priority >= 8 {
			preservedMessages = append(preservedMessages, msg)
		} else if msg.Timestamp.After(recentThreshold) {
			recentMessages = append(recentMessages, msg)
		} else {
			// Add to summary
			summaryContent = append(summaryContent, msg.Content)
		}
	}

	// Create summary message if we have content to summarize
	finalMessages := preservedMessages
	if len(summaryContent) > 0 {
		summary := "CONTEXT SUMMARY: " + strings.Join(summaryContent, " | ")
		if len(summary) > 1000 {
			summary = summary[:1000] + "..."
		}

		summaryMessage := ContextMessage{
			ID:        generateMessageID(),
			Role:      "system",
			Content:   summary,
			Tokens:    cm.estimateTokens(summary),
			Timestamp: now,
			Priority:  9, // High priority to preserve summary
			Metadata:  map[string]interface{}{"type": "summary"},
		}
		finalMessages = append(finalMessages, summaryMessage)
	}

	// Add recent messages
	finalMessages = append(finalMessages, recentMessages...)

	// Recalculate tokens
	newTokenCount := 0
	for _, msg := range finalMessages {
		newTokenCount += msg.Tokens
	}

	cm.messages = finalMessages
	cm.currentTokens = newTokenCount
	cm.compressions++
	cm.lastCompression = time.Now()

	logger.InfoContext(ctx, "Context summarization completed",
		"agent_id", cm.agentID,
		"summary_created", len(summaryContent) > 0,
		"remaining_messages", len(cm.messages),
		"remaining_tokens", cm.currentTokens)

	return nil
}

// GetMessages returns the current context messages
func (cm *ContextManager) GetMessages() []ContextMessage {
	messages := make([]ContextMessage, len(cm.messages))
	copy(messages, cm.messages)
	return messages
}

// GetContextSummary returns a summary of the context manager state
func (cm *ContextManager) GetContextSummary(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":           cm.agentID,
		"max_tokens":         cm.maxTokens,
		"current_tokens":     cm.currentTokens,
		"utilization_pct":    float64(cm.currentTokens) / float64(cm.maxTokens) * 100,
		"reset_strategy":     cm.resetStrategy,
		"message_count":      len(cm.messages),
		"compressions":       cm.compressions,
		"total_tokens_used":  cm.totalTokensUsed,
		"avg_tokens_per_msg": cm.avgTokensPerMsg,
		"last_compression":   cm.lastCompression,
		"total_cost":         cm.totalCost,
		"cost_magnitude":     cm.costProfile.Magnitude,
	}
}

// UpdateCost updates the cost tracking
func (cm *ContextManager) UpdateCost(ctx context.Context, promptTokens, completionTokens int) {
	promptCost := float64(promptTokens) / 1000.0 * cm.costProfile.PromptCostPer1K
	completionCost := float64(completionTokens) / 1000.0 * cm.costProfile.OutputCostPer1K
	cm.totalCost += promptCost + completionCost

	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Updated context cost",
		"agent_id", cm.agentID,
		"prompt_tokens", promptTokens,
		"completion_tokens", completionTokens,
		"session_cost", promptCost+completionCost,
		"total_cost", cm.totalCost)
}

// Reset clears the context window
func (cm *ContextManager) Reset(ctx context.Context) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Resetting context window", "agent_id", cm.agentID)

	cm.messages = make([]ContextMessage, 0)
	cm.currentTokens = 0
	cm.compressions = 0
	cm.lastCompression = time.Time{}
}

// EstimateRemainingCapacity returns how many tokens can still be added
func (cm *ContextManager) EstimateRemainingCapacity() int {
	remaining := cm.maxTokens - cm.currentTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CanAddMessage checks if a message can be added without triggering compression
func (cm *ContextManager) CanAddMessage(content string) bool {
	tokens := cm.estimateTokens(content)
	return cm.currentTokens+tokens <= cm.maxTokens
}

// estimateTokens provides a rough estimate of token count for text
func (cm *ContextManager) estimateTokens(text string) int {
	// Simple heuristic: ~4 characters per token for English text
	// This is approximate - real tokenization would be more accurate
	return len(text) / 4
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
