// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"

	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/tools"
)

// EnhancedAgentFactory creates agents with full reasoning support
type EnhancedAgentFactory struct {
	baseFactory      Factory
	reasoningConfig  ReasoningConfig
	reasoningStorage ReasoningStorage
}

// NewEnhancedAgentFactory creates a new enhanced agent factory
func NewEnhancedAgentFactory(
	baseFactory Factory,
	reasoningConfig ReasoningConfig,
	reasoningStorage ReasoningStorage,
) (*EnhancedAgentFactory, error) {
	if baseFactory == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "base factory cannot be nil", nil).
			WithComponent("agent_factory").
			WithOperation("NewEnhancedAgentFactory")
	}

	// Validate reasoning config
	if err := reasoningConfig.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid reasoning configuration").
			WithComponent("agent_factory").
			WithOperation("NewEnhancedAgentFactory")
	}

	return &EnhancedAgentFactory{
		baseFactory:      baseFactory,
		reasoningConfig:  reasoningConfig,
		reasoningStorage: reasoningStorage,
	}, nil
}

// CreateAgent creates an agent with enhanced reasoning capabilities
func (f *EnhancedAgentFactory) CreateAgent(
	ctx context.Context,
	config *config.EnhancedAgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
) (Agent, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("agent_factory").
			WithOperation("CreateAgent")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("agent_factory").
		WithOperation("CreateAgent").
		With("agent_id", config.ID, "agent_type", config.Type)

	// Create base agent using the base factory
	baseAgent, err := f.baseFactory.CreateAgent(ctx, config.ID, config.Name, config.Type)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create base agent").
			WithComponent("agent_factory").
			WithOperation("CreateAgent").
			WithDetails("agent_id", config.ID)
	}

	// If reasoning is disabled in config, return base agent
	if config.Reasoning.Enabled == false {
		logger.DebugContext(ctx, "Reasoning disabled for agent, returning base agent")
		return baseAgent, nil
	}

	// Enhance with reasoning capabilities
	if err := f.enhanceWithReasoning(ctx, baseAgent, config); err != nil {
		logger.WithError(err).WarnContext(ctx, "Failed to enhance agent with reasoning, returning base agent")
		return baseAgent, nil // Graceful degradation
	}

	logger.InfoContext(ctx, "Created enhanced agent with reasoning support",
		"reasoning_enabled", true,
		"has_storage", f.reasoningStorage != nil)

	return baseAgent, nil
}

// enhanceWithReasoning adds reasoning capabilities to an agent
func (f *EnhancedAgentFactory) enhanceWithReasoning(
	ctx context.Context,
	agent Agent,
	agentConfig *config.EnhancedAgentConfig,
) error {
	// Type assert to WorkerAgent or ManagerAgent
	var workerAgent *WorkerAgent

	switch a := agent.(type) {
	case *WorkerAgent:
		workerAgent = a
	case *ManagerAgent:
		workerAgent = &a.WorkerAgent
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "unsupported agent type: %T", agent).
			WithComponent("agent_factory").
			WithOperation("enhanceWithReasoning")
	}

	// Merge agent-specific reasoning config with factory defaults
	effectiveConfig := f.mergeReasoningConfig(agentConfig.Reasoning)

	// Create reasoning extractor
	extractor, err := NewReasoningExtractor(effectiveConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create reasoning extractor").
			WithComponent("agent_factory").
			WithOperation("enhanceWithReasoning").
			WithDetails("agent_id", agentConfig.ID)
	}

	// Set extractor and storage
	workerAgent.SetReasoningExtractor(extractor)
	if f.reasoningStorage != nil {
		workerAgent.SetReasoningStorage(f.reasoningStorage)
	}

	return nil
}

// mergeReasoningConfig merges agent-specific config with factory defaults
func (f *EnhancedAgentFactory) mergeReasoningConfig(agentConfig config.ReasoningConfig) ReasoningConfig {
	// Start with factory defaults
	merged := f.reasoningConfig

	// Override with agent-specific settings if provided
	if agentConfig.Enabled {
		merged.EnableCaching = agentConfig.ShowThinking
	}

	if agentConfig.MinConfidenceDisplay > 0 {
		merged.MinConfidence = agentConfig.MinConfidenceDisplay
	}

	if agentConfig.DeepReasoningMinComplexity > 0 {
		// This could influence max reasoning length or other parameters
		// For now, we'll use it as a multiplier for max length
		baseLength := merged.MaxReasoningLength
		if baseLength == 0 {
			baseLength = 10000
		}
		merged.MaxReasoningLength = int(float64(baseLength) * (1 + agentConfig.DeepReasoningMinComplexity))
	}

	return merged
}

// CreateFromRegistry creates multiple agents from registry with reasoning
func (f *EnhancedAgentFactory) CreateFromRegistry(
	ctx context.Context,
	registry AgentRegistry,
	providerFactory providers.Factory,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
) (map[string]Agent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("agent_factory").
			WithOperation("CreateFromRegistry")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("agent_factory").
		WithOperation("CreateFromRegistry")

	configs := registry.GetRegisteredAgents()
	agents := make(map[string]Agent, len(configs))

	for _, cfg := range configs {
		// Convert GuildAgentConfig to EnhancedAgentConfig
		enhancedConfig := &config.EnhancedAgentConfig{
			ID:            cfg.ID,
			Name:          cfg.Name,
			Type:          cfg.Type,
			Model:         cfg.Model,
			Provider:      cfg.Provider,
			ContextWindow: cfg.ContextWindow,
			CostMagnitude: cfg.CostMagnitude,
			Capabilities:  cfg.Capabilities,
			// Set default reasoning config if not present
			Reasoning: config.ReasoningConfig{
				Enabled:         true,
				ShowThinking:    true,
				IncludeInPrompt: true,
			},
		}

		// Create LLM client for this agent
		// Note: In real usage, provider would be resolved through dependency injection
		var llmClient providers.LLMClient // This would be injected

		// Create cost manager
		// Note: In real usage, cost manager would be created through dependency injection
		var costManager CostManagerInterface // This would be injected

		// Create agent with reasoning
		agent, err := f.CreateAgent(
			ctx,
			enhancedConfig,
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
			costManager,
		)
		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Failed to create agent",
				"agent_id", cfg.ID)
			continue // Skip this agent
		}

		agents[cfg.ID] = agent
		logger.InfoContext(ctx, "Created agent from registry",
			"agent_id", cfg.ID,
			"agent_type", cfg.Type,
			"reasoning_enabled", enhancedConfig.Reasoning.Enabled)
	}

	if len(agents) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no agents could be created from registry", nil).
			WithComponent("agent_factory").
			WithOperation("CreateFromRegistry")
	}

	logger.InfoContext(ctx, "Created agents from registry",
		"total_configs", len(configs),
		"successful", len(agents))

	return agents, nil
}
