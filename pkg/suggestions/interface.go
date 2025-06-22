// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"time"
)

// SuggestionProvider provides context-aware suggestions for the chat interface
type SuggestionProvider interface {
	// Get suggestions based on current context
	GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error)

	// Update context with new information
	UpdateContext(ctx context.Context, context SuggestionContext) error

	// Get suggestion types this provider handles
	SupportedTypes() []SuggestionType

	// Get provider metadata
	GetMetadata() ProviderMetadata
}

// SuggestionContext contains all context needed for generating suggestions
type SuggestionContext struct {
	CurrentMessage      string                 `json:"current_message"`
	ConversationHistory []ChatMessage          `json:"conversation_history"`
	ProjectContext      ProjectContext         `json:"project_context"`
	AvailableTools      []Tool                 `json:"available_tools"`
	UserPreferences     UserPreferences        `json:"user_preferences"`
	CampaignID          string                 `json:"campaign_id,omitempty"`
	SessionID           string                 `json:"session_id,omitempty"`
	FileContext         *FileContext           `json:"file_context,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// FileContext contains information about the current file position for LSP-based suggestions
type FileContext struct {
	FilePath         string `json:"file_path"`                   // Absolute path to the current file
	Line             int    `json:"line"`                        // Current line number (0-based)
	Column           int    `json:"column"`                      // Current column number (0-based)
	TriggerCharacter string `json:"trigger_character,omitempty"` // Optional trigger character (e.g., ".", "->")
	SymbolAtCursor   string `json:"symbol_at_cursor,omitempty"`  // Optional symbol name at the cursor position
	FileContent      string `json:"file_content,omitempty"`      // Optional file content for context
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	ToolCalls []string               `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProjectContext contains information about the current project
type ProjectContext struct {
	ProjectPath     string            `json:"project_path"`
	ProjectType     string            `json:"project_type"`
	Language        string            `json:"language,omitempty"`
	Framework       string            `json:"framework,omitempty"`
	RecentFiles     []string          `json:"recent_files,omitempty"`
	OpenFiles       []string          `json:"open_files,omitempty"`
	CurrentFile     string            `json:"current_file,omitempty"`
	ProjectMetadata map[string]string `json:"project_metadata,omitempty"`
}

// Tool represents an available tool or command
type Tool struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Category    string   `json:"category"`
	Keywords    []string `json:"keywords,omitempty"`
}

// UserPreferences contains user-specific preferences
type UserPreferences struct {
	SuggestionFrequency string                 `json:"suggestion_frequency"` // "always", "smart", "minimal", "never"
	PreferredTypes      []SuggestionType       `json:"preferred_types"`
	BlacklistedPatterns []string               `json:"blacklisted_patterns,omitempty"`
	FavoriteTemplates   []string               `json:"favorite_templates,omitempty"`
	CustomTriggers      map[string]string      `json:"custom_triggers,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// Suggestion represents a single suggestion
type Suggestion struct {
	ID          string                 `json:"id"`
	Type        SuggestionType         `json:"type"`
	Content     string                 `json:"content"`
	Display     string                 `json:"display"` // How to display in UI
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"` // 0.0 to 1.0
	Priority    int                    `json:"priority"`   // Higher = more important
	Action      SuggestionAction       `json:"action"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Source      string                 `json:"source"` // Which provider generated this
	CreatedAt   time.Time              `json:"created_at"`
}

// SuggestionType defines the type of suggestion
type SuggestionType string

const (
	SuggestionTypeCommand    SuggestionType = "command"
	SuggestionTypeTemplate   SuggestionType = "template"
	SuggestionTypeFollowUp   SuggestionType = "followup"
	SuggestionTypeCode       SuggestionType = "code"
	SuggestionTypeProject    SuggestionType = "project"
	SuggestionTypeTool       SuggestionType = "tool"
	SuggestionTypeCorrection SuggestionType = "correction"
	SuggestionTypeContext    SuggestionType = "context"
)

// SuggestionAction defines what happens when a suggestion is selected
type SuggestionAction struct {
	Type       ActionType             `json:"type"`
	Target     string                 `json:"target"` // Command, template ID, etc.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Preview    string                 `json:"preview,omitempty"`
}

// ActionType defines types of actions
type ActionType string

const (
	ActionTypeInsert   ActionType = "insert"   // Insert text
	ActionTypeReplace  ActionType = "replace"  // Replace current input
	ActionTypeExecute  ActionType = "execute"  // Execute command
	ActionTypeNavigate ActionType = "navigate" // Navigate to file/location
	ActionTypeTemplate ActionType = "template" // Apply template
	ActionTypeInfo     ActionType = "info"     // Show information
	ActionTypeCommand  ActionType = "command"  // Execute LSP or tool command
)

// ProviderMetadata contains information about a suggestion provider
type ProviderMetadata struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

// SuggestionHistory tracks suggestion usage
type SuggestionHistory struct {
	ID           string                 `json:"id"`
	SuggestionID string                 `json:"suggestion_id"`
	Type         SuggestionType         `json:"type"`
	Content      string                 `json:"content"`
	Accepted     bool                   `json:"accepted"`
	UsedAt       time.Time              `json:"used_at"`
	Context      SuggestionContext      `json:"context"`
	UserFeedback *UserFeedback          `json:"user_feedback,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UserFeedback represents user feedback on a suggestion
type UserFeedback struct {
	Helpful    bool      `json:"helpful"`
	Rating     int       `json:"rating,omitempty"` // 1-5
	Comment    string    `json:"comment,omitempty"`
	ReportedAt time.Time `json:"reported_at"`
}

// SuggestionFilter for querying suggestions
type SuggestionFilter struct {
	Types         []SuggestionType `json:"types,omitempty"`
	MinConfidence float64          `json:"min_confidence,omitempty"`
	MaxResults    int              `json:"max_results,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	Sources       []string         `json:"sources,omitempty"`
}

// SuggestionManager manages multiple providers and history
type SuggestionManager interface {
	// Register a suggestion provider
	RegisterProvider(provider SuggestionProvider) error

	// Get suggestions from all providers
	GetSuggestions(ctx context.Context, context SuggestionContext, filter *SuggestionFilter) ([]Suggestion, error)

	// Record suggestion usage
	RecordUsage(ctx context.Context, suggestionID string, accepted bool) error

	// Get suggestion history
	GetHistory(ctx context.Context, sessionID string, limit int) ([]SuggestionHistory, error)

	// Provide feedback on a suggestion
	ProvideFeedback(ctx context.Context, historyID string, feedback UserFeedback) error

	// Get suggestion analytics
	GetAnalytics(ctx context.Context) (*SuggestionAnalytics, error)
}

// SuggestionAnalytics provides analytics on suggestion usage
type SuggestionAnalytics struct {
	TotalSuggestions    int64                    `json:"total_suggestions"`
	AcceptedSuggestions int64                    `json:"accepted_suggestions"`
	AcceptanceRate      float64                  `json:"acceptance_rate"`
	TypeBreakdown       map[SuggestionType]int64 `json:"type_breakdown"`
	ProviderBreakdown   map[string]int64         `json:"provider_breakdown"`
	TopSuggestions      []SuggestionStat         `json:"top_suggestions"`
	UserSatisfaction    float64                  `json:"user_satisfaction"` // Average rating
}

// SuggestionStat represents statistics for a specific suggestion pattern
type SuggestionStat struct {
	Pattern        string         `json:"pattern"`
	Type           SuggestionType `json:"type"`
	UsageCount     int64          `json:"usage_count"`
	AcceptanceRate float64        `json:"acceptance_rate"`
	AverageRating  float64        `json:"average_rating"`
}
