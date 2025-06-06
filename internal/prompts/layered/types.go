package layered

import (
	"time"
	
	"github.com/guild-ventures/guild-core/internal/prompts"
)

// Re-export types from main prompts package for convenience
type (
	PromptLayer = prompts.PromptLayer
	LayerConfig = prompts.LayerConfig
)

// Re-export constants
const (
	LayerPlatform = prompts.LayerPlatform
	LayerGuild    = prompts.LayerGuild
	LayerRole     = prompts.LayerRole
	LayerDomain   = prompts.LayerDomain
	LayerSession  = prompts.LayerSession
	LayerTurn     = prompts.LayerTurn
)

// SystemPrompt represents a single layer in the Guild's layered prompt system
type SystemPrompt struct {
	Layer     PromptLayer            `json:"layer"`
	ArtisanID string                 `json:"artisan_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Content   string                 `json:"content"`
	Version   int                    `json:"version"`
	Priority  int                    `json:"priority"`
	Updated   time.Time              `json:"updated"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LayeredPrompt represents a fully assembled Guild prompt with all layers
type LayeredPrompt struct {
	Layers      []SystemPrompt         `json:"layers"`
	Compiled    string                 `json:"compiled"`
	TokenCount  int                    `json:"token_count"`
	Truncated   bool                   `json:"truncated"`
	CacheKey    string                 `json:"cache_key"`
	ArtisanID   string                 `json:"artisan_id"`
	SessionID   string                 `json:"session_id"`
	AssembledAt time.Time              `json:"assembled_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TurnContext represents ephemeral context for a single Guild interaction
type TurnContext struct {
	UserMessage   string                 `json:"user_message"`
	TaskID        string                 `json:"task_id,omitempty"`
	CommissionID  string                 `json:"commission_id,omitempty"`
	Urgency       string                 `json:"urgency,omitempty"`
	Instructions  []string               `json:"instructions,omitempty"`
	Context       Context                `json:"context,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}