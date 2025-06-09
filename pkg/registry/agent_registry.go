package registry

import (
	"sort"
	"strings"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/interfaces"
)

// CostProfile is an alias to agent.CostProfile
type CostProfile = agent.CostProfile

// AgentFactory creates agent instances
type AgentFactory func(config AgentConfig) (interfaces.Agent, error)

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

// Agent is an alias to the shared interface
type Agent = interfaces.Agent

// AgentInfo is an alias to agent.AgentInfo
type AgentInfo = agent.AgentInfo

// GuildAgentConfig is an alias to agent.GuildAgentConfig
type GuildAgentConfig = agent.GuildAgentConfig

// DefaultAgentRegistry implements the AgentRegistry interface
type DefaultAgentRegistry struct {
	factories    map[string]AgentFactory
	guildAgents  map[string]GuildAgentConfig
	defaultType  string
	agentFactory AgentFactory
	mu           sync.RWMutex
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
func (r *DefaultAgentRegistry) GetAgent(agentType string) (interfaces.Agent, error) {
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
	for _, config := range r.guildAgents {
		if config.CostMagnitude <= maxCost {
			agents = append(agents, AgentInfo{
				ID:            config.ID,
				Name:          config.Name,
				Type:          config.Type,
				Capabilities:  config.Capabilities,
				CostMagnitude: config.CostMagnitude,
				CostProfile: CostProfile{
					Magnitude:     config.CostMagnitude,
					ContextWindow: config.ContextWindow,
					Available:     true,
				},
			})
		}
	}

	// Sort by cost magnitude (cheapest first)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CostMagnitude < agents[j].CostMagnitude
	})

	return agents
}

// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
func (r *DefaultAgentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) {
	agents := r.GetAgentsByCapability(capability)
	if len(agents) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no agent found with capability: "+capability, nil)
	}
	return &agents[0], nil
}

// GetAgentsByCapability returns all agents that have the specified capability
func (r *DefaultAgentRegistry) GetAgentsByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	for _, config := range r.guildAgents {
		// Check if this agent has the required capability
		hasCapability := false
		for _, cap := range config.Capabilities {
			if cap == capability {
				hasCapability = true
				break
			}
		}

		if hasCapability {
			agents = append(agents, AgentInfo{
				ID:            config.ID,
				Name:          config.Name,
				Type:          config.Type,
				Capabilities:  config.Capabilities,
				CostMagnitude: config.CostMagnitude,
				CostProfile: CostProfile{
					Magnitude:     config.CostMagnitude,
					ContextWindow: config.ContextWindow,
					Available:     true,
				},
			})
		}
	}

	// Sort by cost magnitude (cheapest first)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CostMagnitude < agents[j].CostMagnitude
	})

	return agents
}

// RegisterGuildAgent registers a configured agent from guild config
func (r *DefaultAgentRegistry) RegisterGuildAgent(config GuildAgentConfig) error {
	if config.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent ID cannot be empty", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Auto-detect cost magnitude if not specified
	if config.CostMagnitude == 0 {
		config.CostMagnitude = r.getEffectiveCostMagnitude(config)
	}

	r.guildAgents[config.ID] = config
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

// getEffectiveCostMagnitude auto-detects cost magnitude based on model name
func (r *DefaultAgentRegistry) getEffectiveCostMagnitude(config GuildAgentConfig) int {
	// Auto-detect based on model name patterns
	switch {
	case contains(config.Model, "gpt-3.5"), contains(config.Model, "claude-3-haiku"):
		return 1 // Cheap models
	case contains(config.Model, "gpt-4"), contains(config.Model, "claude-3-sonnet"):
		return 3 // Mid-range models
	case contains(config.Model, "claude-3-opus"), contains(config.Model, "gpt-4-turbo"):
		return 8 // Expensive models
	case config.Provider == "local", config.Provider == "ollama":
		return 0 // Free/local models
	default:
		return 1 // Default to cheap if unknown
	}
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
