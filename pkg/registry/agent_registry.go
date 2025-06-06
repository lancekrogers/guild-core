package registry

import (
	"sort"
	"strings"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// DefaultAgentRegistry implements the AgentRegistry interface
type DefaultAgentRegistry struct {
	factories      map[string]func(config AgentConfig) (Agent, error)
	guildAgents    map[string]GuildAgentConfig // Guild-configured agents
	defaultType    string
	mu             sync.RWMutex
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() AgentRegistry {
	return &DefaultAgentRegistry{
		factories:   make(map[string]func(config AgentConfig) (Agent, error)),
		guildAgents: make(map[string]GuildAgentConfig),
	}
}

// RegisterAgentType registers a new agent type with its factory function
func (r *DefaultAgentRegistry) RegisterAgentType(name string, factory AgentFactory) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent type name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterAgentType")
	}
	if factory == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent factory cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterAgentType")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "agent type '%s' already registered", name).
			WithComponent("registry").
			WithOperation("RegisterAgentType").
			WithDetails("agentType", name)
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
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent type '%s' not registered", agentType).
			WithComponent("registry").
			WithOperation("GetAgent").
			WithDetails("agentType", agentType)
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
		return gerror.Newf(gerror.ErrCodeNotFound, "agent type '%s' not registered", agentType).
			WithComponent("registry").
			WithOperation("SetDefaultAgentType").
			WithDetails("agentType", agentType)
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
			return nil, gerror.New(gerror.ErrCodeMissingRequired, "no agent type specified and no default type set", nil).
				WithComponent("registry").
				WithOperation("CreateAgent")
		}
	}

	return r.GetAgent(agentType)
}

// Cost-based selection methods

// GetAgentsByCost returns agents with cost magnitude <= maxCost, sorted by cost
func (r *DefaultAgentRegistry) GetAgentsByCost(maxCost int) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	for _, config := range r.guildAgents {
		costMagnitude := r.getEffectiveCostMagnitude(config)
		if costMagnitude <= maxCost {
			agents = append(agents, r.configToAgentInfo(config))
		}
	}

	// Sort by cost magnitude (ascending)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CostMagnitude < agents[j].CostMagnitude
	})

	return agents
}

// GetCheapestAgentByCapability returns the lowest-cost agent with the given capability
func (r *DefaultAgentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var cheapestAgent *AgentInfo
	lowestCost := 999 // Higher than max Fibonacci value

	for _, config := range r.guildAgents {
		if r.hasCapability(config.Capabilities, capability) {
			costMagnitude := r.getEffectiveCostMagnitude(config)
			if costMagnitude < lowestCost {
				lowestCost = costMagnitude
				agentInfo := r.configToAgentInfo(config)
				cheapestAgent = &agentInfo
			}
		}
	}

	if cheapestAgent == nil {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no agent found with capability '%s'", capability).
			WithComponent("registry").
			WithOperation("GetCheapestAgentByCapability").
			WithDetails("capability", capability)
	}

	return cheapestAgent, nil
}

// GetAgentsByCapability returns all agents that have the specified capability
func (r *DefaultAgentRegistry) GetAgentsByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	for _, config := range r.guildAgents {
		if r.hasCapability(config.Capabilities, capability) {
			agents = append(agents, r.configToAgentInfo(config))
		}
	}

	// Sort by cost magnitude (ascending)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].CostMagnitude < agents[j].CostMagnitude
	})

	return agents
}

// RegisterGuildAgent registers a configured agent from guild config
func (r *DefaultAgentRegistry) RegisterGuildAgent(config GuildAgentConfig) error {
	if config.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent ID cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterGuildAgent")
	}
	if config.Name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterGuildAgent")
	}
	if len(config.Capabilities) == 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent must have at least one capability", nil).
			WithComponent("registry").
			WithOperation("RegisterGuildAgent")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.guildAgents[config.ID]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "agent with ID '%s' already registered", config.ID).
			WithComponent("registry").
			WithOperation("RegisterGuildAgent").
			WithDetails("agentID", config.ID)
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

	// Sort by cost magnitude for consistent ordering
	sort.Slice(agents, func(i, j int) bool {
		costI := r.getEffectiveCostMagnitude(agents[i])
		costJ := r.getEffectiveCostMagnitude(agents[j])
		return costI < costJ
	})

	return agents
}

// Helper methods

// configToAgentInfo converts GuildAgentConfig to AgentInfo
func (r *DefaultAgentRegistry) configToAgentInfo(config GuildAgentConfig) AgentInfo {
	return AgentInfo{
		ID:            config.ID,
		Name:          config.Name,
		Type:          config.Type,
		Provider:      config.Provider,
		Model:         config.Model,
		Capabilities:  config.Capabilities,
		Tools:         config.Tools,
		CostMagnitude: r.getEffectiveCostMagnitude(config),
		ContextWindow: r.getEffectiveContextWindow(config),
		ContextReset:  r.getEffectiveContextReset(config),
		Available:     true, // TODO: Add availability checking
	}
}

// hasCapability checks if the agent has a specific capability
func (r *DefaultAgentRegistry) hasCapability(capabilities []string, target string) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}

// getEffectiveCostMagnitude returns the cost magnitude with smart defaults
func (r *DefaultAgentRegistry) getEffectiveCostMagnitude(config GuildAgentConfig) int {
	if config.CostMagnitude != 0 {
		return config.CostMagnitude
	}
	
	// If no model specified, this is likely a tool-only agent
	if config.Model == "" {
		return 0
	}
	
	// Auto-assign based on model characteristics
	modelLower := strings.ToLower(config.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4"):
		return 5 // High cost
	case strings.Contains(modelLower, "gpt-3.5"):
		return 2 // Low-mid cost  
	case strings.Contains(modelLower, "claude-3-opus"):
		return 8 // Most expensive
	case strings.Contains(modelLower, "claude-3-sonnet"):
		return 3 // Mid cost
	case strings.Contains(modelLower, "claude-3-haiku"):
		return 1 // Cheap
	case strings.Contains(modelLower, "ollama") || strings.Contains(modelLower, "local"):
		return 0 // Free local models
	default:
		return 1 // Default to cheap for unknown models
	}
}

// getEffectiveContextWindow returns the context window with auto-detection
func (r *DefaultAgentRegistry) getEffectiveContextWindow(config GuildAgentConfig) int {
	if config.ContextWindow > 0 {
		return config.ContextWindow
	}
	
	// Auto-detect based on known models
	modelLower := strings.ToLower(config.Model)
	switch {
	case strings.Contains(modelLower, "gpt-4-turbo"):
		return 128000
	case strings.Contains(modelLower, "gpt-4"):
		return 32000
	case strings.Contains(modelLower, "gpt-3.5"):
		return 16000
	case strings.Contains(modelLower, "claude-3"):
		return 200000
	case strings.Contains(modelLower, "claude-2"):
		return 100000
	default:
		return 8000 // Conservative default
	}
}

// getEffectiveContextReset returns the context reset behavior with smart defaults
func (r *DefaultAgentRegistry) getEffectiveContextReset(config GuildAgentConfig) string {
	if config.ContextReset != "" {
		return config.ContextReset
	}
	
	// Default based on agent type
	switch config.Type {
	case "manager":
		return "summarize" // Managers need to preserve context
	case "worker":
		return "truncate" // Workers can restart fresh
	default:
		return "truncate" // Conservative default
	}
}