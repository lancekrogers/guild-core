package registry

import (
	"fmt"
	"sync"
)

// DefaultAgentRegistry implements the AgentRegistry interface
type DefaultAgentRegistry struct {
	factories   map[string]func(config AgentConfig) (Agent, error)
	defaultType string
	mu          sync.RWMutex
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() AgentRegistry {
	return &DefaultAgentRegistry{
		factories: make(map[string]func(config AgentConfig) (Agent, error)),
	}
}

// RegisterAgentType registers a new agent type with its factory function
func (r *DefaultAgentRegistry) RegisterAgentType(name string, factory AgentFactory) error {
	if name == "" {
		return fmt.Errorf("agent type name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("agent factory cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("agent type '%s' already registered", name)
	}

	// Store the factory directly
	r.factories[name] = factory

	return nil
}

// GetAgent creates an agent instance of the specified type
func (r *DefaultAgentRegistry) GetAgent(agentType string) (Agent, error) {
	r.mu.RLock()
	factory, exists := r.factories[agentType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent type '%s' not registered", agentType)
	}

	// Create agent with default configuration
	// In practice, you'd want to pass in the actual configuration
	config := AgentConfig{
		DefaultType: agentType,
	}

	return factory(config)
}

// ListAgentTypes returns all registered agent types
func (r *DefaultAgentRegistry) ListAgentTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for name := range r.factories {
		types = append(types, name)
	}
	return types
}

// HasAgentType checks if an agent type is registered
func (r *DefaultAgentRegistry) HasAgentType(agentType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[agentType]
	return exists
}

// SetDefaultAgentType sets the default agent type
func (r *DefaultAgentRegistry) SetDefaultAgentType(agentType string) error {
	if !r.HasAgentType(agentType) {
		return fmt.Errorf("agent type '%s' not registered", agentType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultType = agentType
	return nil
}

// GetDefaultAgentType returns the default agent type
func (r *DefaultAgentRegistry) GetDefaultAgentType() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultType
}

// CreateAgent creates an agent using the default type if no type is specified
func (r *DefaultAgentRegistry) CreateAgent(agentType string) (Agent, error) {
	if agentType == "" {
		agentType = r.GetDefaultAgentType()
		if agentType == "" {
			return nil, fmt.Errorf("no agent type specified and no default type set")
		}
	}

	return r.GetAgent(agentType)
}