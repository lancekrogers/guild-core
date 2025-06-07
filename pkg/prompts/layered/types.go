package layered

import (
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Sentinel errors for common cases
var (
	// ErrPromptNotFound is returned when a requested prompt does not exist
	ErrPromptNotFound = gerror.New(gerror.ErrCodeNotFound, "prompt not found", nil).
		WithComponent("prompts").WithOperation("GetPrompt")

	// ErrTemplateNotFound is returned when a requested template does not exist
	ErrTemplateNotFound = gerror.New(gerror.ErrCodeNotFound, "template not found", nil).
		WithComponent("prompts").WithOperation("GetTemplate")

	// ErrLayerNotFound is returned when a requested layer does not exist
	ErrLayerNotFound = gerror.New(gerror.ErrCodeNotFound, "layer not found", nil).
		WithComponent("prompts").WithOperation("GetPromptLayer")
)

// PromptLayer represents the hierarchical layers of Guild prompts
type PromptLayer string

const (
	// LayerPlatform contains core Guild platform rules (terms of service, safety)
	LayerPlatform PromptLayer = "platform"

	// LayerGuild contains project-wide goals and style guidelines
	LayerGuild PromptLayer = "guild"

	// LayerRole contains artisan role definitions (Guild Master, Code Artisan, etc.)
	LayerRole PromptLayer = "role"

	// LayerDomain contains project type specializations (web-app, cli-tool, etc.)
	LayerDomain PromptLayer = "domain"

	// LayerSession contains user preferences and session-specific context
	LayerSession PromptLayer = "session"

	// LayerTurn contains ephemeral instructions for single interactions
	LayerTurn PromptLayer = "turn"
)

// LayerConfig provides configuration for compiling layered prompts
type LayerConfig struct {
	// AgentID is the ID of the agent requesting the prompt
	AgentID string

	// SessionID is the current session ID
	SessionID string

	// Role is the agent's role (e.g., "guild_master", "code_artisan")
	Role string

	// Domain is the project domain (e.g., "web-app", "cli-tool")
	Domain string

	// IncludeLayers specifies which layers to include
	IncludeLayers []PromptLayer

	// MaxTokens limits the compiled prompt size
	MaxTokens int
}

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
