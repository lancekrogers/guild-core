// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/tools"
)

// EnhancedFactory creates agent instances using the enhanced configuration system
type EnhancedFactory struct {
	providerFactory   *providers.Factory
	toolRegistry      tools.Registry
	memoryManager     memory.ChainManager
	commissionManager commission.CommissionManager
	costManager       CostManagerInterface
	providerSelector  *ProviderSelector
	toolFilterFactory *ToolFilterFactory
}

// NewEnhancedFactory creates a new enhanced agent factory
func NewEnhancedFactory(
	providerFactory *providers.Factory,
	toolRegistry tools.Registry,
	memoryManager memory.ChainManager,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
) *EnhancedFactory {
	return &EnhancedFactory{
		providerFactory:   providerFactory,
		toolRegistry:      toolRegistry,
		memoryManager:     memoryManager,
		commissionManager: commissionManager,
		costManager:       costManager,
		providerSelector:  NewProviderSelector(providerFactory),
		toolFilterFactory: NewToolFilterFactory(toolRegistry),
	}
}

// CreateFromConfig creates an agent from an enhanced configuration
func (f *EnhancedFactory) CreateFromConfig(ctx context.Context, config *config.EnhancedAgentConfig) (Agent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromConfig")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "EnhancedFactory")
	ctx = observability.WithOperation(ctx, "CreateFromConfig")

	logger.InfoContext(ctx, "Creating agent from enhanced configuration",
		"agent_id", config.ID,
		"agent_type", config.Type,
		"model", config.Model)

	// Validate configuration
	if err := config.Validate(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid agent configuration").
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromConfig").
			WithDetails("agent_id", config.ID)
	}

	// Select provider and create client
	providerSelection, err := f.providerSelector.SelectProvider(ctx, config)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to select provider").
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromConfig").
			WithDetails("agent_id", config.ID)
	}

	logger.InfoContext(ctx, "Provider selected successfully",
		"agent_id", config.ID,
		"provider", providerSelection.Provider,
		"model", providerSelection.Model,
		"cost_magnitude", providerSelection.CostProfile.Magnitude)

	// Create tool filter
	toolFilter := f.toolFilterFactory.CreateToolFilter(config)

	// Create context manager
	contextManager := NewContextManager(config, providerSelection.CostProfile)

	// Create agent based on type
	switch config.Type {
	case "manager":
		return f.createManagerAgent(ctx, config, providerSelection, toolFilter, contextManager)
	case "worker":
		return f.createWorkerAgent(ctx, config, providerSelection, toolFilter, contextManager)
	case "specialist":
		return f.createSpecialistAgent(ctx, config, providerSelection, toolFilter, contextManager)
	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unknown agent type: %s", config.Type).
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromConfig").
			WithDetails("agent_id", config.ID)
	}
}

// createManagerAgent creates a manager agent with enhanced configuration
func (f *EnhancedFactory) createManagerAgent(
	ctx context.Context,
	config *config.EnhancedAgentConfig,
	providerSelection *ProviderSelection,
	toolFilter *ToolFilter,
	contextManager *ContextManager,
) (Agent, error) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating manager agent", "agent_id", config.ID)

	// Manager agents typically have limited tool access but high reasoning capability
	agent := &ConfigBasedAgent{
		id:                config.ID,
		name:              config.Name,
		agentType:         config.Type,
		role:              config.Role,
		specialty:         config.Specialty,
		backstory:         config.Backstory,
		llmClient:         providerSelection.Client,
		memoryManager:     f.memoryManager,
		toolRegistry:      f.toolRegistry,
		commissionManager: f.commissionManager,
		costManager:       f.costManager,
		toolFilter:        toolFilter,
		contextManager:    contextManager,
		config:            config,
		providerSelection: providerSelection,
		capabilities:      config.Capabilities,
		prompts:           config.Prompts,
		metadata:          config.Metadata,
	}

	logger.InfoContext(ctx, "Manager agent created successfully", "agent_id", config.ID)
	return agent, nil
}

// createWorkerAgent creates a worker agent with enhanced configuration
func (f *EnhancedFactory) createWorkerAgent(
	ctx context.Context,
	config *config.EnhancedAgentConfig,
	providerSelection *ProviderSelection,
	toolFilter *ToolFilter,
	contextManager *ContextManager,
) (Agent, error) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating worker agent", "agent_id", config.ID)

	// Worker agents typically have broader tool access for task execution
	agent := &ConfigBasedAgent{
		id:                config.ID,
		name:              config.Name,
		agentType:         config.Type,
		role:              config.Role,
		specialty:         config.Specialty,
		backstory:         config.Backstory,
		llmClient:         providerSelection.Client,
		memoryManager:     f.memoryManager,
		toolRegistry:      f.toolRegistry,
		commissionManager: f.commissionManager,
		costManager:       f.costManager,
		toolFilter:        toolFilter,
		contextManager:    contextManager,
		config:            config,
		providerSelection: providerSelection,
		capabilities:      config.Capabilities,
		prompts:           config.Prompts,
		metadata:          config.Metadata,
	}

	logger.InfoContext(ctx, "Worker agent created successfully", "agent_id", config.ID)
	return agent, nil
}

// createSpecialistAgent creates a specialist agent with enhanced configuration
func (f *EnhancedFactory) createSpecialistAgent(
	ctx context.Context,
	config *config.EnhancedAgentConfig,
	providerSelection *ProviderSelection,
	toolFilter *ToolFilter,
	contextManager *ContextManager,
) (Agent, error) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating specialist agent", "agent_id", config.ID)

	// Specialist agents have domain-specific capabilities and tools
	agent := &ConfigBasedAgent{
		id:                config.ID,
		name:              config.Name,
		agentType:         config.Type,
		role:              config.Role,
		specialty:         config.Specialty,
		backstory:         config.Backstory,
		llmClient:         providerSelection.Client,
		memoryManager:     f.memoryManager,
		toolRegistry:      f.toolRegistry,
		commissionManager: f.commissionManager,
		costManager:       f.costManager,
		toolFilter:        toolFilter,
		contextManager:    contextManager,
		config:            config,
		providerSelection: providerSelection,
		capabilities:      config.Capabilities,
		prompts:           config.Prompts,
		metadata:          config.Metadata,
	}

	logger.InfoContext(ctx, "Specialist agent created successfully", "agent_id", config.ID)
	return agent, nil
}

// CreateFromPath creates an agent from a configuration file path
func (f *EnhancedFactory) CreateFromPath(ctx context.Context, configPath string) (Agent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromPath")
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating agent from configuration file", "config_path", configPath)

	// Load configuration
	config, err := config.LoadAgentConfig(ctx, configPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to load agent configuration").
			WithComponent("EnhancedFactory").
			WithOperation("CreateFromPath").
			WithDetails("config_path", configPath)
	}

	// Create agent
	return f.CreateFromConfig(ctx, config)
}

// CreateMultipleFromDirectory creates multiple agents from configuration files in a directory
func (f *EnhancedFactory) CreateMultipleFromDirectory(ctx context.Context, configDir string) ([]Agent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("EnhancedFactory").
			WithOperation("CreateMultipleFromDirectory")
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Creating multiple agents from directory", "config_dir", configDir)

	// Load all configurations
	configs, err := config.LoadAgentConfigsFromDirectory(ctx, configDir)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to load agent configurations").
			WithComponent("EnhancedFactory").
			WithOperation("CreateMultipleFromDirectory").
			WithDetails("config_dir", configDir)
	}

	// Create agents
	agents := make([]Agent, 0, len(configs))
	var creationErrors []error

	for _, config := range configs {
		// Check context in loop
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during agent creation").
				WithComponent("EnhancedFactory").
				WithOperation("CreateMultipleFromDirectory")
		}

		agent, err := f.CreateFromConfig(ctx, config)
		if err != nil {
			logger.WarnContext(ctx, "Failed to create agent", "agent_id", config.ID, "error", err)
			creationErrors = append(creationErrors, err)
			continue
		}

		agents = append(agents, agent)
	}

	if len(agents) == 0 && len(creationErrors) > 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "no agents were successfully created", nil).
			WithComponent("EnhancedFactory").
			WithOperation("CreateMultipleFromDirectory").
			WithDetails("config_dir", configDir).
			WithDetails("error_count", len(creationErrors))
	}

	logger.InfoContext(ctx, "Multiple agents created",
		"config_dir", configDir,
		"successful_count", len(agents),
		"failed_count", len(creationErrors))

	return agents, nil
}

// ValidateConfiguration validates an agent configuration without creating the agent
func (f *EnhancedFactory) ValidateConfiguration(ctx context.Context, config *config.EnhancedAgentConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("EnhancedFactory").
			WithOperation("ValidateConfiguration")
	}

	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Validating agent configuration", "agent_id", config.ID)

	// Validate configuration
	if err := config.Validate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "agent configuration validation failed").
			WithComponent("EnhancedFactory").
			WithOperation("ValidateConfiguration").
			WithDetails("agent_id", config.ID)
	}

	// Validate provider selection
	if err := f.providerSelector.ValidateModelProvider(ctx, config.GetEffectiveProvider(), config.Model); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "provider validation failed").
			WithComponent("EnhancedFactory").
			WithOperation("ValidateConfiguration").
			WithDetails("agent_id", config.ID)
	}

	logger.DebugContext(ctx, "Agent configuration validation passed", "agent_id", config.ID)
	return nil
}

// GetProviderSelector returns the provider selector for advanced usage
func (f *EnhancedFactory) GetProviderSelector() *ProviderSelector {
	return f.providerSelector
}

// GetToolFilterFactory returns the tool filter factory for advanced usage
func (f *EnhancedFactory) GetToolFilterFactory() *ToolFilterFactory {
	return f.toolFilterFactory
}
