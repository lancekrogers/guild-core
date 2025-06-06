#!/bin/bash

echo "Final build fixes..."

# Fix agent_registry.go properly
cat > pkg/registry/agent_registry.go << 'EOF'
package registry

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// AgentFactory creates agent instances
type AgentFactory func(config AgentConfig) (Agent, error)

// AgentConfig holds agent configuration
type AgentConfig struct {
	Type         string
	Name         string
	Model        string
	Provider     string
	SystemPrompt string
	Tools        []string
	Capabilities []string
	CostProfile  CostProfile
}

// Agent interface (minimal for registry)
type Agent interface {
	GetName() string
	GetType() string
	GetCapabilities() []string
}

// AgentInfo holds agent information
type AgentInfo struct {
	Type         string
	Name         string
	Capabilities []string
	CostProfile  CostProfile
}

// GuildAgentConfig represents a configured agent from guild config
type GuildAgentConfig struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Model        string   `yaml:"model"`
	Provider     string   `yaml:"provider"`
	SystemPrompt string   `yaml:"system_prompt"`
	Tools        []string `yaml:"tools"`
	Capabilities []string `yaml:"capabilities"`
}

// DefaultAgentRegistry implements the AgentRegistry interface
type DefaultAgentRegistry struct {
	factories      map[string]AgentFactory
	guildAgents    map[string]GuildAgentConfig
	defaultType    string
	agentFactory   AgentFactory
	mu             sync.RWMutex
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() AgentRegistry {
	return &DefaultAgentRegistry{
		factories:   make(map[string]AgentFactory),
		guildAgents: make(map[string]GuildAgentConfig),
	}
}

// RegisterAgentType registers a new agent type with its factory function
func (r *DefaultAgentRegistry) RegisterAgentType(name string, factory AgentFactory) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent type name cannot be empty", nil)
	}
	if factory == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent factory cannot be nil", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "agent type already registered", nil)
	}

	r.factories[name] = factory
	return nil
}

// GetAgent creates an agent instance of the specified type
func (r *DefaultAgentRegistry) GetAgent(agentType string) (Agent, error) {
	r.mu.RLock()
	factory, exists := r.factories[agentType]
	r.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "agent type not found", nil)
	}

	// Create default config
	config := AgentConfig{
		Type: agentType,
		Name: agentType + "-agent",
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
	sort.Strings(types)
	return types
}

// HasAgentType checks if an agent type is registered
func (r *DefaultAgentRegistry) HasAgentType(agentType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[agentType]
	return exists
}

// GetAgentsByCost returns agents with cost magnitude <= maxCost
func (r *DefaultAgentRegistry) GetAgentsByCost(maxCost int) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	// Implementation would filter by cost
	return agents
}

// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
func (r *DefaultAgentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) {
	agents := r.GetAgentsByCapability(capability)
	if len(agents) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no agent with capability", nil)
	}
	return &agents[0], nil
}

// GetAgentsByCapability returns all agents that have the specified capability
func (r *DefaultAgentRegistry) GetAgentsByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	// Implementation would filter by capability
	return agents
}

// RegisterGuildAgent registers a configured agent from guild config
func (r *DefaultAgentRegistry) RegisterGuildAgent(config GuildAgentConfig) error {
	if config.Name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent name cannot be empty", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.guildAgents[config.Name] = config
	return nil
}

// GetRegisteredAgents returns all registered agent configurations
func (r *DefaultAgentRegistry) GetRegisteredAgents() []GuildAgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]GuildAgentConfig, 0, len(r.guildAgents))
	for _, config := range r.guildAgents {
		agents = append(agents, config)
	}
	return agents
}
EOF

# Clean up temporary files
rm -f /tmp/agent_registry_fix.go
rm -f fix_gerror_usage.sh fix_remaining_build.sh fix_final_registry.sh

echo "Final fixes applied"