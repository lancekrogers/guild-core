package layered

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/memory"
)

// Manager defines the interface for system prompt management
type Manager interface {
	// GetSystemPrompt retrieves a system prompt for a specific role and domain (legacy)
	GetSystemPrompt(ctx context.Context, role string, domain string) (string, error)

	// GetTemplate retrieves a named template
	GetTemplate(ctx context.Context, templateName string) (string, error)

	// FormatContext formats a context object into a string representation
	FormatContext(ctx context.Context, context Context) (string, error)

	// ListRoles returns all available roles
	ListRoles(ctx context.Context) ([]string, error)

	// ListDomains returns all available domains for a role
	ListDomains(ctx context.Context, role string) ([]string, error)
}

// LayeredManager extends Manager with layered prompt capabilities
type LayeredManager interface {
	Manager

	// BuildLayeredPrompt assembles a complete layered prompt for an artisan
	BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx TurnContext) (*LayeredPrompt, error)

	// GetPromptLayer retrieves a specific prompt layer
	GetPromptLayer(ctx context.Context, layer PromptLayer, artisanID, sessionID string) (*SystemPrompt, error)

	// SetPromptLayer sets or updates a specific prompt layer
	SetPromptLayer(ctx context.Context, prompt SystemPrompt) error

	// DeletePromptLayer removes a specific prompt layer
	DeletePromptLayer(ctx context.Context, layer PromptLayer, artisanID, sessionID string) error

	// ListPromptLayers returns all layers for an artisan/session
	ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]SystemPrompt, error)

	// InvalidateCache clears the layered prompt cache
	InvalidateCache(ctx context.Context, artisanID, sessionID string) error
}

// Context represents contextual information to be injected into prompts
type Context interface {
	// GetCommissionID returns the commission/objective ID
	GetCommissionID() string

	// GetCommissionTitle returns the commission title
	GetCommissionTitle() string

	// GetCurrentTask returns the current task information
	GetCurrentTask() TaskContext

	// GetRelevantSections returns relevant sections from the objective hierarchy
	GetRelevantSections() []Section

	// GetRelatedTasks returns related task information
	GetRelatedTasks() []TaskContext
}

// TaskContext represents information about a task
type TaskContext struct {
	ID            string
	Title         string
	Description   string
	SourceSection string
	Priority      string
	Estimate      string
	Dependencies  []string
	Capabilities  []string
}

// Section represents a section from the objective hierarchy
type Section struct {
	Level   int
	Path    string
	Title   string
	Content string
	Tasks   []TaskContext
}

// Formatter defines the interface for context formatting
type Formatter interface {
	// FormatAsXML formats context as XML for efficient token usage
	FormatAsXML(ctx Context) (string, error)

	// FormatAsMarkdown formats context as markdown
	FormatAsMarkdown(ctx Context) (string, error)

	// OptimizeForTokens optimizes content for token limits
	OptimizeForTokens(content string, maxTokens int) (string, error)
}

// Registry defines the interface for prompt registration
type Registry interface {
	// RegisterPrompt registers a system prompt (legacy)
	RegisterPrompt(role, domain, prompt string) error

	// RegisterTemplate registers a template
	RegisterTemplate(name, template string) error

	// GetPrompt retrieves a registered prompt (legacy)
	GetPrompt(role, domain string) (string, error)

	// GetTemplate retrieves a registered template
	GetTemplate(name string) (string, error)
}

// LayeredRegistry extends Registry with layered prompt support
type LayeredRegistry interface {
	Registry

	// RegisterLayeredPrompt registers a prompt in a specific layer
	RegisterLayeredPrompt(layer PromptLayer, identifier string, prompt SystemPrompt) error

	// GetLayeredPrompt retrieves a prompt from a specific layer
	GetLayeredPrompt(layer PromptLayer, identifier string) (*SystemPrompt, error)

	// ListLayeredPrompts returns all prompts in a layer
	ListLayeredPrompts(layer PromptLayer) ([]SystemPrompt, error)

	// DeleteLayeredPrompt removes a prompt from a layer
	DeleteLayeredPrompt(layer PromptLayer, identifier string) error

	// GetDefaultPrompts returns default prompts for a layer
	GetDefaultPrompts(layer PromptLayer) ([]SystemPrompt, error)
}

// LayeredStore extends memory.Store with layered prompt storage capabilities
type LayeredStore interface {
	memory.Store

	// SavePromptLayer stores a layered prompt in the Guild Archives
	SavePromptLayer(ctx context.Context, layer, identifier string, data []byte) error

	// GetPromptLayer retrieves a layered prompt from the Guild Archives
	GetPromptLayer(ctx context.Context, layer, identifier string) ([]byte, error)

	// DeletePromptLayer removes a layered prompt from the Guild Archives
	DeletePromptLayer(ctx context.Context, layer, identifier string) error

	// ListPromptLayers returns all prompts in a specific layer
	ListPromptLayers(ctx context.Context, layer string) ([]string, error)

	// CacheCompiledPrompt stores a compiled layered prompt for performance
	CacheCompiledPrompt(ctx context.Context, cacheKey string, data []byte) error

	// GetCachedPrompt retrieves a compiled layered prompt from cache
	GetCachedPrompt(ctx context.Context, cacheKey string) ([]byte, error)

	// InvalidatePromptCache removes cached prompts matching a pattern
	InvalidatePromptCache(ctx context.Context, keyPattern string) error

	// SavePromptMetrics stores Guild prompt performance metrics
	SavePromptMetrics(ctx context.Context, metricID string, data []byte) error

	// GetPromptMetrics retrieves Guild prompt performance metrics
	GetPromptMetrics(ctx context.Context, metricID string) ([]byte, error)
}
