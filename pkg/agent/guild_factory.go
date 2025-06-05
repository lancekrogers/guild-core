package agent

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// GuildFactory creates agents from guild configuration
type GuildFactory struct {
	registry         registry.ComponentRegistry
	memoryManager    memory.ChainManager
	toolRegistry     *tools.ToolRegistry
	objectiveManager *objective.Manager
	guildConfig      *config.GuildConfig
}

// NewGuildFactory creates a new guild-aware agent factory
func NewGuildFactory(
	registry registry.ComponentRegistry,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveManager *objective.Manager,
	guildConfig *config.GuildConfig,
) *GuildFactory {
	return &GuildFactory{
		registry:         registry,
		memoryManager:    memoryManager,
		toolRegistry:     toolRegistry,
		objectiveManager: objectiveManager,
		guildConfig:      guildConfig,
	}
}

// CreateAgentFromConfig creates an agent from guild configuration
func (f *GuildFactory) CreateAgentFromConfig(ctx context.Context, agentID string) (Agent, error) {
	// Find agent config
	agentConfig, err := f.guildConfig.GetAgentByID(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found in guild config: %w", err)
	}

	// Get provider and create a client
	// For now, we'll use the default provider from registry
	// TODO: Update this when provider registry supports per-model clients
	provider, err := f.registry.Providers().GetDefaultProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	// Create an LLM client wrapper
	// TODO: This needs to be updated when providers support model selection
	llmClient, ok := provider.(providers.LLMClient)
	if !ok {
		return nil, fmt.Errorf("provider does not implement LLMClient interface")
	}

	// Create tool registry for this agent
	agentToolRegistry := tools.NewToolRegistry()
	
	// Register only the tools this agent has access to
	for _, toolName := range agentConfig.Tools {
		tool, exists := f.toolRegistry.GetTool(toolName)
		if !exists {
			// Log warning but continue - tool might not be registered yet
			continue
		}
		agentToolRegistry.RegisterTool(tool)
	}

	// Create the appropriate agent type
	var agent Agent
	switch agentConfig.Type {
	case "manager":
		agent = NewManagerAgent(
			agentConfig.ID,
			agentConfig.Name,
			llmClient,
			f.memoryManager,
			agentToolRegistry,
			f.objectiveManager,
		)
	case "worker", "specialist":
		agent = NewWorkerAgent(
			agentConfig.ID,
			agentConfig.Name,
			llmClient,
			f.memoryManager,
			agentToolRegistry,
			f.objectiveManager,
		)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentConfig.Type)
	}

	// Add capabilities metadata to agent
	switch a := agent.(type) {
	case *WorkerAgent:
		a.SetCapabilities(agentConfig.Capabilities)
		a.SetDescription(agentConfig.Description)
	case *ManagerAgent:
		a.SetCapabilities(agentConfig.Capabilities)
		a.SetDescription(agentConfig.Description)
	}

	return agent, nil
}

// CreateManagerAgent creates the guild's manager agent
func (f *GuildFactory) CreateManagerAgent(ctx context.Context) (Agent, error) {
	managerConfig, err := f.guildConfig.GetManagerAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to get manager agent: %w", err)
	}
	return f.CreateAgentFromConfig(ctx, managerConfig.ID)
}

// CreateAllAgents creates all agents in the guild
func (f *GuildFactory) CreateAllAgents(ctx context.Context) (map[string]Agent, error) {
	agents := make(map[string]Agent)
	
	for _, agentConfig := range f.guildConfig.Agents {
		agent, err := f.CreateAgentFromConfig(ctx, agentConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent %s: %w", agentConfig.ID, err)
		}
		agents[agentConfig.ID] = agent
	}
	
	return agents, nil
}

// CreateAgentsByCapability creates all agents with a specific capability
func (f *GuildFactory) CreateAgentsByCapability(ctx context.Context, capability string) ([]Agent, error) {
	agentConfigs := f.guildConfig.GetAgentsByCapability(capability)
	
	agents := make([]Agent, 0, len(agentConfigs))
	for _, agentConfig := range agentConfigs {
		agent, err := f.CreateAgentFromConfig(ctx, agentConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent %s: %w", agentConfig.ID, err)
		}
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// UpdateGuildConfig updates the guild configuration
func (f *GuildFactory) UpdateGuildConfig(config *config.GuildConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid guild config: %w", err)
	}
	f.guildConfig = config
	return nil
}

// GetGuildConfig returns the current guild configuration
func (f *GuildFactory) GetGuildConfig() *config.GuildConfig {
	return f.guildConfig
}