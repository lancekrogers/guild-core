// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/suggestions"
)

// ChatSuggestionHandler provides suggestion integration for chat interfaces
type ChatSuggestionHandler struct {
	agent EnhancedGuildArtisan
}

// NewChatSuggestionHandler creates a new chat suggestion handler
func NewChatSuggestionHandler(agent EnhancedGuildArtisan) *ChatSuggestionHandler {
	return &ChatSuggestionHandler{
		agent: agent,
	}
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
	Message        string                        `json:"message"`
	ConversationID string                        `json:"conversation_id,omitempty"`
	FileContext    *suggestions.FileContext      `json:"file_context,omitempty"`
	Filter         *suggestions.SuggestionFilter `json:"filter,omitempty"`
	MaxSuggestions int                           `json:"max_suggestions,omitempty"`
	MinConfidence  float64                       `json:"min_confidence,omitempty"`
}

// SuggestionResponse represents the response with suggestions
type SuggestionResponse struct {
	Suggestions []suggestions.Suggestion `json:"suggestions"`
	Metadata    map[string]interface{}   `json:"metadata"`
	Success     bool                     `json:"success"`
	Error       string                   `json:"error,omitempty"`
}

// GetSuggestions handles a suggestion request from the chat interface
func (h *ChatSuggestionHandler) GetSuggestions(ctx context.Context, request SuggestionRequest) (*SuggestionResponse, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("agent.chat_integration").
		WithOperation("GetSuggestions").
		With("agent_id", h.agent.GetID())

	logger.DebugContext(ctx, "Processing suggestion request",
		"message_length", len(request.Message),
		"has_file_context", request.FileContext != nil,
		"has_filter", request.Filter != nil)

	// Apply default limits
	if request.MaxSuggestions <= 0 {
		request.MaxSuggestions = 10
	}
	if request.MinConfidence <= 0 {
		request.MinConfidence = 0.3
	}

	// Build filter
	filter := request.Filter
	if filter == nil {
		filter = &suggestions.SuggestionFilter{}
	}
	if filter.MaxResults <= 0 {
		filter.MaxResults = request.MaxSuggestions
	}
	if filter.MinConfidence <= 0 {
		filter.MinConfidence = request.MinConfidence
	}

	// Get suggestions from the agent
	suggestions, err := h.agent.GetSuggestionsForContext(ctx, request.Message, filter)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get suggestions", "error", err)
		return &SuggestionResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	response := &SuggestionResponse{
		Suggestions: suggestions,
		Success:     true,
		Metadata: map[string]interface{}{
			"suggestion_count": len(suggestions),
			"agent_id":         h.agent.GetID(),
			"filter_applied":   filter,
		},
	}

	logger.InfoContext(ctx, "Successfully provided suggestions",
		"suggestion_count", len(suggestions))

	return response, nil
}

// ExecuteWithSuggestions handles execution with suggestion assistance
func (h *ChatSuggestionHandler) ExecuteWithSuggestions(ctx context.Context, request string, enableSuggestions bool) (*EnhancedExecutionResult, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("agent.chat_integration").
		WithOperation("ExecuteWithSuggestions").
		With("agent_id", h.agent.GetID(), "enable_suggestions", enableSuggestions)

	logger.InfoContext(ctx, "Executing request with suggestion support",
		"request_length", len(request))

	return h.agent.ExecuteWithSuggestions(ctx, request, enableSuggestions)
}

// GetAvailableProviders returns information about available suggestion providers
func (h *ChatSuggestionHandler) GetAvailableProviders(ctx context.Context) ([]suggestions.ProviderMetadata, error) {
	suggestionManager := h.agent.GetSuggestionManager()
	if suggestionManager == nil {
		return []suggestions.ProviderMetadata{}, nil
	}

	// This would require extending the SuggestionManager interface to expose provider metadata
	// For now, return empty list - this would be implemented when the interface is extended
	return []suggestions.ProviderMetadata{}, nil
}

// ChatSuggestionConfig represents configuration for chat suggestion integration
type ChatSuggestionConfig struct {
	EnableSuggestions    bool                         `json:"enable_suggestions" yaml:"enable_suggestions"`
	DefaultMaxResults    int                          `json:"default_max_results" yaml:"default_max_results"`
	DefaultMinConfidence float64                      `json:"default_min_confidence" yaml:"default_min_confidence"`
	EnabledTypes         []suggestions.SuggestionType `json:"enabled_types" yaml:"enabled_types"`
	DisabledProviders    []string                     `json:"disabled_providers" yaml:"disabled_providers"`
	AutoSuggestThreshold float64                      `json:"auto_suggest_threshold" yaml:"auto_suggest_threshold"`
}

// DefaultChatSuggestionConfig returns a default configuration
func DefaultChatSuggestionConfig() ChatSuggestionConfig {
	return ChatSuggestionConfig{
		EnableSuggestions:    true,
		DefaultMaxResults:    5,
		DefaultMinConfidence: 0.5,
		EnabledTypes: []suggestions.SuggestionType{
			suggestions.SuggestionTypeCommand,
			suggestions.SuggestionTypeTool,
			suggestions.SuggestionTypeTemplate,
			suggestions.SuggestionTypeFollowUp,
		},
		DisabledProviders:    []string{},
		AutoSuggestThreshold: 0.7,
	}
}

// ApplyConfig applies configuration to suggestion requests
func (h *ChatSuggestionHandler) ApplyConfig(config ChatSuggestionConfig, request *SuggestionRequest) {
	if request.MaxSuggestions <= 0 {
		request.MaxSuggestions = config.DefaultMaxResults
	}
	if request.MinConfidence <= 0 {
		request.MinConfidence = config.DefaultMinConfidence
	}

	// Apply type filtering
	if len(config.EnabledTypes) > 0 {
		if request.Filter == nil {
			request.Filter = &suggestions.SuggestionFilter{}
		}
		request.Filter.Types = config.EnabledTypes
	}
}

// SuggestionWebhook represents a webhook for suggestion events
type SuggestionWebhook struct {
	URL     string            `json:"url"`
	Events  []string          `json:"events"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SuggestionEvent represents an event in the suggestion system
type SuggestionEvent struct {
	Type       string                  `json:"type"`
	AgentID    string                  `json:"agent_id"`
	Timestamp  int64                   `json:"timestamp"`
	Suggestion *suggestions.Suggestion `json:"suggestion,omitempty"`
	Context    map[string]interface{}  `json:"context,omitempty"`
}

// EmitSuggestionEvent emits a suggestion event (placeholder for future webhook implementation)
func (h *ChatSuggestionHandler) EmitSuggestionEvent(ctx context.Context, event SuggestionEvent) {
	logger := observability.GetLogger(ctx).
		WithComponent("agent.chat_integration").
		WithOperation("EmitSuggestionEvent")

	logger.DebugContext(ctx, "Emitting suggestion event",
		"event_type", event.Type,
		"agent_id", event.AgentID)

	// Future implementation would send webhooks or publish to event bus
}
