package agent

import (
	"context"

	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ContextAgentFactory creates agents with context-aware capabilities
type ContextAgentFactory struct {
	// Configuration for agent creation
}

// newContextAgentFactory creates a new context-aware agent factory (private constructor)
func newContextAgentFactory() *ContextAgentFactory {
	return &ContextAgentFactory{}
}

// AgentConfig represents configuration for creating an agent
type AgentConfig struct {
	ID              string                 `yaml:"id" json:"id"`
	Name            string                 `yaml:"name" json:"name"`
	Type            string                 `yaml:"type" json:"type"`                         // worker, manager, specialist
	Capabilities    []string               `yaml:"capabilities" json:"capabilities"`
	DefaultProvider string                 `yaml:"default_provider" json:"default_provider"`
	SystemPrompt    string                 `yaml:"system_prompt" json:"system_prompt"`
	Enabled         bool                   `yaml:"enabled" json:"enabled"`
	Settings        map[string]interface{} `yaml:"settings" json:"settings"`
}

// CreateAgent creates a new context-aware agent based on configuration
func (f *ContextAgentFactory) CreateAgent(ctx context.Context, config AgentConfig) (guildcontext.AgentClient, error) {
	if !config.Enabled {
		return nil, gerror.Newf(gerror.ErrCodeValidation, "agent '%s' is disabled", config.ID).
			WithComponent("agent").
			WithOperation("CreateAgent").
			WithDetails("agent_id", config.ID)
	}
	
	// Create the appropriate agent type
	var agent *ContextAwareAgent
	
	switch config.Type {
	case "worker", "":
		agent = newContextAwareAgent(config.ID, config.Name, "worker", config.Capabilities)
	case "manager":
		agent = newContextAwareAgent(config.ID, config.Name, "manager", config.Capabilities)
	case "specialist":
		agent = newContextAwareAgent(config.ID, config.Name, "specialist", config.Capabilities)
	default:
		return nil, gerror.Newf(gerror.ErrCodeValidation, "unknown agent type: %s", config.Type).
			WithComponent("agent").
			WithOperation("CreateAgent").
			WithDetails("agent_id", config.ID).
			WithDetails("agent_type", config.Type)
	}
	
	// Configure the agent
	if config.DefaultProvider != "" {
		agent.SetDefaultProvider(config.DefaultProvider)
	}
	
	if config.SystemPrompt != "" {
		agent.SetSystemPrompt(config.SystemPrompt)
	}
	
	// Apply additional settings
	for key, value := range config.Settings {
		agent.AddMetadata(key, value)
	}
	
	return agent, nil
}

// CreateAgentFromRegistry creates an agent using configuration from the registry
func (f *ContextAgentFactory) CreateAgentFromRegistry(ctx context.Context, agentName string) (guildcontext.AgentClient, error) {
	// Get configuration from context
	_, err := guildcontext.GetConfigProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get config from context").
			WithComponent("agent").
			WithOperation("CreateAgentFromRegistry").
			WithDetails("agent_name", agentName)
	}
	
	// This would typically load agent config from the configuration provider
	// For now, we'll create a default configuration
	agentConfig := AgentConfig{
		ID:           agentName,
		Name:         agentName,
		Type:         "worker",
		Capabilities: []string{"general"},
		Enabled:      true,
		Settings:     make(map[string]interface{}),
	}
	
	return f.CreateAgent(ctx, agentConfig)
}

// RegisterAgentsWithRegistry registers agents from configuration with the registry
func (f *ContextAgentFactory) RegisterAgentsWithRegistry(ctx context.Context, agentsConfig map[string]interface{}) error {
	registry, err := guildcontext.GetRegistryProvider(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get registry from context").
			WithComponent("agent").
			WithOperation("RegisterAgentsWithRegistry")
	}
	
	for agentName, agentConfigRaw := range agentsConfig {
		// Parse agent configuration
		agentConfigMap, ok := agentConfigRaw.(map[string]interface{})
		if !ok {
			return gerror.Newf(gerror.ErrCodeValidation, "invalid config for agent %s", agentName).
				WithComponent("agent").
				WithOperation("RegisterAgentsWithRegistry").
				WithDetails("agent_name", agentName)
		}
		
		agentConfig := parseAgentConfig(agentName, agentConfigMap)
		
		// Create the agent
		agent, err := f.CreateAgent(ctx, agentConfig)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeAgent, "failed to create agent %s", agentName).
				WithComponent("agent").
				WithOperation("RegisterAgentsWithRegistry").
				WithDetails("agent_name", agentName)
		}
		
		// Register with registry
		if err := registry.Agents().RegisterAgent(agentName, agent); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeAgent, "failed to register agent %s", agentName).
				WithComponent("agent").
				WithOperation("RegisterAgentsWithRegistry").
				WithDetails("agent_name", agentName)
		}
	}
	
	return nil
}

// parseAgentConfig parses agent configuration from a map
func parseAgentConfig(name string, configMap map[string]interface{}) AgentConfig {
	config := AgentConfig{
		ID:       name,
		Name:     name,
		Type:     "worker",
		Enabled:  true,
		Settings: make(map[string]interface{}),
	}
	
	// Parse configuration fields
	if agentType, ok := configMap["type"].(string); ok {
		config.Type = agentType
	}
	
	if displayName, ok := configMap["name"].(string); ok {
		config.Name = displayName
	}
	
	if enabled, ok := configMap["enabled"].(bool); ok {
		config.Enabled = enabled
	}
	
	if defaultProvider, ok := configMap["default_provider"].(string); ok {
		config.DefaultProvider = defaultProvider
	}
	
	if systemPrompt, ok := configMap["system_prompt"].(string); ok {
		config.SystemPrompt = systemPrompt
	}
	
	// Parse capabilities
	if capabilitiesRaw, ok := configMap["capabilities"]; ok {
		if capabilitiesList, ok := capabilitiesRaw.([]interface{}); ok {
			capabilities := make([]string, 0, len(capabilitiesList))
			for _, cap := range capabilitiesList {
				if capStr, ok := cap.(string); ok {
					capabilities = append(capabilities, capStr)
				}
			}
			config.Capabilities = capabilities
		} else if capabilitiesStr, ok := capabilitiesRaw.(string); ok {
			config.Capabilities = []string{capabilitiesStr}
		}
	}
	
	// Parse settings
	if settings, ok := configMap["settings"].(map[string]interface{}); ok {
		config.Settings = settings
	}
	
	return config
}

// GetDefaultAgentConfigs returns default agent configurations
func GetDefaultAgentConfigs() map[string]AgentConfig {
	return map[string]AgentConfig{
		"worker": {
			ID:           "default-worker",
			Name:         "Default Worker",
			Type:         "worker",
			Capabilities: []string{"general", "completion"},
			Enabled:      true,
			Settings:     make(map[string]interface{}),
		},
		"coding-agent": {
			ID:           "coding-agent",
			Name:         "Coding Specialist",
			Type:         "specialist",
			Capabilities: []string{"coding", "development", "debugging", "code-review"},
			SystemPrompt: "You are a specialized coding agent. You excel at writing, reviewing, and debugging code. Provide clear, well-documented solutions with best practices.",
			Enabled:      true,
			Settings:     make(map[string]interface{}),
		},
		"analysis-agent": {
			ID:           "analysis-agent",
			Name:         "Analysis Specialist",
			Type:         "specialist",
			Capabilities: []string{"analysis", "reasoning", "research", "data-analysis"},
			SystemPrompt: "You are a specialized analysis agent. You excel at breaking down complex problems, conducting research, and providing detailed analytical insights.",
			Enabled:      true,
			Settings:     make(map[string]interface{}),
		},
		"manager": {
			ID:           "manager-agent",
			Name:         "Task Manager",
			Type:         "manager",
			Capabilities: []string{"coordination", "planning", "task-management", "delegation"},
			SystemPrompt: "You are a manager agent responsible for coordinating tasks, planning workflows, and delegating work to other agents. Provide clear direction and comprehensive task breakdowns.",
			Enabled:      true,
			Settings:     make(map[string]interface{}),
		},
	}
}

// CreateDefaultAgents creates and registers default agents
func (f *ContextAgentFactory) CreateDefaultAgents(ctx context.Context) error {
	registry, err := guildcontext.GetRegistryProvider(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get registry from context").
			WithComponent("agent").
			WithOperation("CreateDefaultAgents")
	}
	
	defaultConfigs := GetDefaultAgentConfigs()
	
	for agentName, config := range defaultConfigs {
		agent, err := f.CreateAgent(ctx, config)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeAgent, "failed to create default agent %s", agentName).
				WithComponent("agent").
				WithOperation("CreateDefaultAgents").
				WithDetails("agent_name", agentName)
		}
		
		if err := registry.Agents().RegisterAgent(agentName, agent); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeAgent, "failed to register default agent %s", agentName).
				WithComponent("agent").
				WithOperation("CreateDefaultAgents").
				WithDetails("agent_name", agentName)
		}
	}
	
	// Set default agent
	if err := registry.Agents().SetDefaultAgent("worker"); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to set default agent").
			WithComponent("agent").
			WithOperation("CreateDefaultAgents").
			WithDetails("default_agent", "worker")
	}
	
	return nil
}

// DefaultContextAgentFactory creates a context-aware agent factory for registry use
func DefaultContextAgentFactory() *ContextAgentFactory {
	return newContextAgentFactory()
}