package registry

import (
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts/standard"
)

// PromptProvider defines the interface for prompt providers
type PromptProvider interface {
	GetPrompt(id string, version string) (*Prompt, error)
	ListPrompts() ([]PromptMetadata, error)
	ValidatePrompt(id string, data interface{}) error
	RenderPrompt(id string, data interface{}) (string, error)
}

// Prompt represents a prompt with its content and metadata
type Prompt struct {
	ID       string
	Version  string
	Category string
	Content  string
	Metadata *PromptMetadata
}

// PromptMetadata contains metadata about a prompt
type PromptMetadata struct {
	ID                 string
	Version            string
	Category           string
	Complexity         int
	Tags               []string
	RequiredVariables  []string
	OptionalVariables  []string
	ModelCompatibility []string
}

// PromptRegistry manages prompt providers
type PromptRegistry struct {
	providers map[string]PromptProvider
	mu        sync.RWMutex
}

// NewPromptRegistry creates a new prompt registry
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		providers: make(map[string]PromptProvider),
	}
}

// Register registers a prompt provider
func (r *PromptRegistry) Register(name string, provider PromptProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "prompt provider %s already registered", name).
			WithComponent("registry").
			WithOperation("Register").
			WithDetails("provider", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a prompt provider by name
func (r *PromptRegistry) Get(name string) (PromptProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "prompt provider %s not found", name).
			WithComponent("registry").
			WithOperation("Get").
			WithDetails("provider", name)
	}

	return provider, nil
}

// List returns all registered prompt providers
func (r *PromptRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// DefaultPromptProvider implements PromptProvider using the internal prompt manager
type DefaultPromptProvider struct {
	manager *standard.EnhancedPromptManager
}

// NewDefaultPromptProvider creates a new default prompt provider
func NewDefaultPromptProvider() (*DefaultPromptProvider, error) {
	manager, err := standard.NewEnhancedPromptManager()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "error creating prompt manager").
			WithComponent("registry").
			WithOperation("NewDefaultPromptProvider")
	}

	return &DefaultPromptProvider{
		manager: manager,
	}, nil
}

// GetPrompt retrieves a prompt by ID and version
func (p *DefaultPromptProvider) GetPrompt(id string, version string) (*Prompt, error) {
	// For now, ignore version parameter as we don't have versioning yet
	metadata, err := p.manager.GetMetadata(id)
	if err != nil {
		return nil, err
	}

	// Get the raw template content (would need to expose this in manager)
	// For now, return a placeholder
	return &Prompt{
		ID:       id,
		Version:  metadata.Version,
		Category: metadata.Category,
		Content:  "", // Would need to expose raw content
		Metadata: convertMetadata(metadata),
	}, nil
}

// ListPrompts returns all available prompts
func (p *DefaultPromptProvider) ListPrompts() ([]PromptMetadata, error) {
	prompts := p.manager.ListPrompts()
	result := make([]PromptMetadata, 0, len(prompts))

	for _, meta := range prompts {
		result = append(result, *convertMetadata(meta))
	}

	return result, nil
}

// ValidatePrompt validates prompt data without rendering
func (p *DefaultPromptProvider) ValidatePrompt(id string, data interface{}) error {
	if dataMap, ok := data.(map[string]interface{}); ok {
		return p.manager.ValidatePrompt(id, dataMap)
	}
	return gerror.New(gerror.ErrCodeInvalidFormat, "data must be a map[string]interface{}", nil).
			WithComponent("registry").
			WithOperation("ValidatePrompt")
}

// RenderPrompt renders a prompt with the given data
func (p *DefaultPromptProvider) RenderPrompt(id string, data interface{}) (string, error) {
	return p.manager.RenderPrompt(id, data)
}

// convertMetadata converts internal metadata to registry metadata
func convertMetadata(internal *standard.PromptMetadata) *PromptMetadata {
	return &PromptMetadata{
		ID:                 internal.ID,
		Version:            internal.Version,
		Category:           internal.Category,
		Complexity:         internal.Complexity,
		Tags:               internal.Tags,
		RequiredVariables:  internal.Variables.Required,
		OptionalVariables:  internal.Variables.Optional,
		ModelCompatibility: internal.ModelCompatibility,
	}
}

// IntegratePromptRegistry adds prompt registry to the main registry
func (r *DefaultComponentRegistry) IntegratePromptRegistry() error {
	// Create prompt registry if it doesn't exist
	if r.promptRegistry == nil {
		r.promptRegistry = NewPromptRegistry()
	}

	// Register default provider
	defaultProvider, err := NewDefaultPromptProvider()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "error creating default prompt provider").
			WithComponent("registry").
			WithOperation("IntegratePromptRegistry")
	}

	if err := r.promptRegistry.Register("default", defaultProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "error registering default prompt provider").
			WithComponent("registry").
			WithOperation("IntegratePromptRegistry")
	}

	return nil
}

// GetPromptProvider retrieves a prompt provider from the registry
func (r *DefaultComponentRegistry) GetPromptProvider(name string) (PromptProvider, error) {
	if r.promptRegistry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "prompt registry not initialized", nil).
			WithComponent("registry").
			WithOperation("GetPromptProvider")
	}
	return r.promptRegistry.Get(name)
}
