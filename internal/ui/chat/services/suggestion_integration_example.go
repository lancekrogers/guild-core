// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/suggestions"
)

// ChatWithSuggestions demonstrates how to integrate SuggestionService with ChatService
type ChatWithSuggestions struct {
	chatService       *ChatService
	suggestionService *SuggestionService
	context           context.Context

	// Configuration
	enableSuggestions bool
	suggestionMode    string // "pre", "post", "both"
}

// NewChatWithSuggestions creates an integrated chat service with suggestions
func NewChatWithSuggestions(
	ctx context.Context,
	chatService *ChatService,
	enhancedAgent core.EnhancedGuildArtisan,
) (*ChatWithSuggestions, error) {
	if chatService == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "chat service cannot be nil", nil).
			WithComponent("services.chat_with_suggestions").
			WithOperation("NewChatWithSuggestions")
	}

	// Create suggestion handler and service
	handler := core.NewChatSuggestionHandler(enhancedAgent)
	suggestionService, err := NewSuggestionService(ctx, handler)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create suggestion service").
			WithComponent("services.chat_with_suggestions").
			WithOperation("NewChatWithSuggestions")
	}

	return &ChatWithSuggestions{
		chatService:       chatService,
		suggestionService: suggestionService,
		context:           ctx,
		enableSuggestions: true,
		suggestionMode:    "both",
	}, nil
}

// SendMessageWithSuggestions sends a message and provides intelligent suggestions
func (cws *ChatWithSuggestions) SendMessageWithSuggestions(agentID, message string) tea.Cmd {
	return tea.Batch(
		// Get pre-execution suggestions
		cws.getPreSuggestions(message),
		// Send the actual message
		cws.chatService.SendMessage(agentID, message),
	)
}

// getPreSuggestions gets suggestions before executing the message
func (cws *ChatWithSuggestions) getPreSuggestions(message string) tea.Cmd {
	if !cws.enableSuggestions || cws.suggestionMode == "post" {
		return nil
	}

	return func() tea.Msg {
		logger := observability.GetLogger(cws.context).
			WithComponent("services.chat_with_suggestions").
			WithOperation("getPreSuggestions")

		// Build context for suggestions
		context := &SuggestionContext{
			ConversationID: fmt.Sprintf("chat-%d", time.Now().Unix()),
			// Add more context as needed
		}

		// Get suggestions
		cmd := cws.suggestionService.GetSuggestions(message, context)
		msg := cmd()

		// Convert to pre-suggestion message
		if sugMsg, ok := msg.(SuggestionsReceivedMsg); ok {
			logger.Debug("Got pre-execution suggestions",
				"count", len(sugMsg.Suggestions),
				"from_cache", sugMsg.FromCache)

			return PreExecutionSuggestionsMsg{
				Suggestions: sugMsg.Suggestions,
				Metadata:    sugMsg.Metadata,
			}
		}

		return msg
	}
}

// HandleAgentResponse processes agent responses and generates follow-up suggestions
func (cws *ChatWithSuggestions) HandleAgentResponse(response AgentResponseMsg) tea.Cmd {
	if !cws.enableSuggestions || cws.suggestionMode == "pre" {
		return nil
	}

	// Generate follow-up suggestions based on the response
	return cws.suggestionService.GetFollowUpSuggestions("", response.Content)
}

// SetSuggestionMode configures when suggestions are provided
func (cws *ChatWithSuggestions) SetSuggestionMode(mode string) error {
	switch mode {
	case "pre", "post", "both", "none":
		cws.suggestionMode = mode
		if mode == "none" {
			cws.enableSuggestions = false
		}
		return nil
	default:
		return gerror.Newf(gerror.ErrCodeInvalidInput, "invalid suggestion mode: %s", mode).
			WithComponent("services.chat_with_suggestions").
			WithOperation("SetSuggestionMode")
	}
}

// OptimizeMessageForAgent optimizes a message before sending to an agent
func (cws *ChatWithSuggestions) OptimizeMessageForAgent(message string) string {
	// Use the suggestion service's context optimization
	return cws.suggestionService.OptimizeContext(message)
}

// GetIntegratedStats returns combined statistics from both services
func (cws *ChatWithSuggestions) GetIntegratedStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Get chat service stats
	chatStats := cws.chatService.GetStats()
	for k, v := range chatStats {
		stats["chat_"+k] = v
	}

	// Get suggestion service stats
	suggestionStats := cws.suggestionService.GetStats()
	for k, v := range suggestionStats {
		stats["suggestion_"+k] = v
	}

	// Add integration-specific stats
	stats["suggestions_enabled"] = cws.enableSuggestions
	stats["suggestion_mode"] = cws.suggestionMode

	return stats
}

// Message types for integrated service

// PreExecutionSuggestionsMsg contains suggestions before execution
type PreExecutionSuggestionsMsg struct {
	Suggestions []suggestions.Suggestion
	Metadata    map[string]interface{}
}

// PostExecutionSuggestionsMsg contains follow-up suggestions
type PostExecutionSuggestionsMsg struct {
	Suggestions []suggestions.Suggestion
	AgentID     string
	OriginalMsg string
}
