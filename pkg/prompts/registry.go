package prompts

import (
	"context"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ManagerFactory is a function that creates a prompt manager
type ManagerFactory func(ctx context.Context, config ManagerConfig) (Manager, error)

// FormatterFactory is a function that creates a context formatter
type FormatterFactory func() (Formatter, error)

// PromptRegistry manages runtime selection of prompt strategies
type PromptRegistry struct {
	mu                   sync.RWMutex
	managerFactories     map[ManagerType]ManagerFactory
	formatterFactories   map[string]FormatterFactory
	defaultManagerType   ManagerType
	defaultFormatterType string
}

// NewPromptRegistry creates a new prompt registry with default strategies
func NewPromptRegistry() *PromptRegistry {
	registry := &PromptRegistry{
		managerFactories:     make(map[ManagerType]ManagerFactory),
		formatterFactories:   make(map[string]FormatterFactory),
		defaultManagerType:   TypeLayered, // Default to layered for Guild
		defaultFormatterType: "xml",
	}

	// Register default strategies
	registry.registerDefaults()

	return registry
}

// registerDefaults registers the built-in prompt strategies
func (r *PromptRegistry) registerDefaults() {
	// Register manager strategies
	r.managerFactories[TypeStandard] = newStandardManager
	r.managerFactories[TypeLayered] = newLayeredManager

	// Register formatter strategies
	r.formatterFactories["xml"] = defaultXMLFormatterFactory
	r.formatterFactories["markdown"] = defaultMarkdownFormatterFactory
}

// RegisterManagerStrategy registers a new prompt manager strategy
func (r *PromptRegistry) RegisterManagerStrategy(managerType ManagerType, factory ManagerFactory) error {
	if factory == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "factory cannot be nil", nil).
			WithComponent("prompts").
			WithOperation("RegisterManagerStrategy").
			WithDetails("manager_type", string(managerType))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.managerFactories[managerType] = factory
	return nil
}

// RegisterFormatterStrategy registers a new context formatter strategy
func (r *PromptRegistry) RegisterFormatterStrategy(formatterType string, factory FormatterFactory) error {
	if factory == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "factory cannot be nil", nil).
			WithComponent("prompts").
			WithOperation("RegisterFormatterStrategy").
			WithDetails("formatter_type", formatterType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.formatterFactories[formatterType] = factory
	return nil
}

// CreateManager creates a prompt manager using the registered strategy
func (r *PromptRegistry) CreateManager(ctx context.Context, config ManagerConfig) (Manager, error) {
	r.mu.RLock()
	factory, exists := r.managerFactories[config.Type]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "unknown prompt manager type", nil).
			WithComponent("prompts").
			WithOperation("CreateManager").
			WithDetails("manager_type", string(config.Type))
	}

	return factory(ctx, config)
}

// CreateFormatter creates a context formatter using the registered strategy
func (r *PromptRegistry) CreateFormatter(formatterType string) (Formatter, error) {
	r.mu.RLock()
	factory, exists := r.formatterFactories[formatterType]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "unknown formatter type", nil).
			WithComponent("prompts").
			WithOperation("CreateFormatter").
			WithDetails("formatter_type", formatterType)
	}

	return factory()
}

// GetDefaultManager creates a manager with default configuration
func (r *PromptRegistry) GetDefaultManager(ctx context.Context) (Manager, error) {
	config := ManagerConfig{
		Type: r.defaultManagerType,
		LayeredConfig: &LayeredManagerConfig{
			DefaultPlatformPrompt: "You are a Guild artisan, part of an advanced AI agent framework...",
			DefaultGuildPrompt:    "Follow Guild conventions and medieval naming patterns...",
		},
	}

	return r.CreateManager(ctx, config)
}

// GetDefaultFormatter creates a formatter with default configuration
func (r *PromptRegistry) GetDefaultFormatter() (Formatter, error) {
	return r.CreateFormatter(r.defaultFormatterType)
}

// SetDefaultManagerType sets the default prompt manager type
func (r *PromptRegistry) SetDefaultManagerType(managerType ManagerType) error {
	r.mu.RLock()
	_, exists := r.managerFactories[managerType]
	r.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "manager type not registered", nil).
			WithComponent("prompts").
			WithOperation("SetDefaultManagerType").
			WithDetails("manager_type", string(managerType))
	}

	r.mu.Lock()
	r.defaultManagerType = managerType
	r.mu.Unlock()

	return nil
}

// SetDefaultFormatterType sets the default formatter type
func (r *PromptRegistry) SetDefaultFormatterType(formatterType string) error {
	r.mu.RLock()
	_, exists := r.formatterFactories[formatterType]
	r.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "formatter type not registered", nil).
			WithComponent("prompts").
			WithOperation("SetDefaultFormatterType").
			WithDetails("formatter_type", formatterType)
	}

	r.mu.Lock()
	r.defaultFormatterType = formatterType
	r.mu.Unlock()

	return nil
}

// ListManagerTypes returns all registered manager types
func (r *PromptRegistry) ListManagerTypes() []ManagerType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ManagerType, 0, len(r.managerFactories))
	for managerType := range r.managerFactories {
		types = append(types, managerType)
	}

	return types
}

// ListFormatterTypes returns all registered formatter types
func (r *PromptRegistry) ListFormatterTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.formatterFactories))
	for formatterType := range r.formatterFactories {
		types = append(types, formatterType)
	}

	return types
}

// Formatter interface for context formatting (to avoid import cycle)
type Formatter interface {
	FormatAsXML(ctx Context) (string, error)
	FormatAsMarkdown(ctx Context) (string, error)
	OptimizeForTokens(content string, maxTokens int) (string, error)
}

// Context interface for prompt context (to avoid import cycle)
type Context interface {
	GetCommissionID() string
	GetCommissionTitle() string
	GetCurrentTask() TaskContext
	GetRelevantSections() []Section
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

// FormatterAdapter adapts the layered context formatter to the registry interface
type FormatterAdapter struct {
	xmlFormatter XMLFormatterImpl
}

// XMLFormatterImpl wraps the actual formatter to avoid import cycles
type XMLFormatterImpl interface {
	FormatAsXML(ctx interface{}) (string, error)
	FormatAsMarkdown(ctx interface{}) (string, error)
	OptimizeForTokens(content string, maxTokens int) (string, error)
}

// FormatAsXML implements the Formatter interface
func (f *FormatterAdapter) FormatAsXML(ctx Context) (string, error) {
	return f.xmlFormatter.FormatAsXML(ctx)
}

// FormatAsMarkdown implements the Formatter interface
func (f *FormatterAdapter) FormatAsMarkdown(ctx Context) (string, error) {
	return f.xmlFormatter.FormatAsMarkdown(ctx)
}

// OptimizeForTokens implements the Formatter interface
func (f *FormatterAdapter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	return f.xmlFormatter.OptimizeForTokens(content, maxTokens)
}

// defaultXMLFormatterFactory creates an XML formatter
func defaultXMLFormatterFactory() (Formatter, error) {
	// Create the actual formatter using the package function
	// Note: This creates a dependency but allows the registry to work
	return &FormatterAdapter{
		xmlFormatter: &defaultXMLFormatter{},
	}, nil
}

// defaultXMLFormatter provides a basic XML formatter implementation
type defaultXMLFormatter struct{}

func (f *defaultXMLFormatter) FormatAsXML(ctx interface{}) (string, error) {
	return fmt.Sprintf("<context>%v</context>", ctx), nil
}

func (f *defaultXMLFormatter) FormatAsMarkdown(ctx interface{}) (string, error) {
	return fmt.Sprintf("## Context\n\n%v", ctx), nil
}

func (f *defaultXMLFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	if len(content) > maxTokens*4 {
		return content[:maxTokens*4] + "...", nil
	}
	return content, nil
}

// defaultMarkdownFormatterFactory creates a Markdown formatter
func defaultMarkdownFormatterFactory() (Formatter, error) {
	// For now, use the same XML formatter which supports both formats
	return &FormatterAdapter{}, nil
}

// Global registry instance
var defaultRegistry *PromptRegistry

// init initializes the default registry
func init() {
	defaultRegistry = NewPromptRegistry()
}

// GetRegistry returns the global prompt registry
func GetRegistry() *PromptRegistry {
	return defaultRegistry
}

// DefaultPromptRegistryFactory creates a prompt registry for use in other registries
func DefaultPromptRegistryFactory() *PromptRegistry {
	return NewPromptRegistry()
}
