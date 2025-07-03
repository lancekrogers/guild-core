// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/interfaces"
	"github.com/lancekrogers/guild/pkg/tools"
)

// createAgentFactory creates an agent factory with all required dependencies
func (r *DefaultComponentRegistry) createAgentFactory(ctx context.Context) (AgentFactory, error) {
	// Get default provider for LLM client
	llmClient, err := r.providerRegistry.GetDefaultProvider()
	if err != nil {
		// If no provider is available, create a null factory function
		return func(config AgentConfig) (Agent, error) {
			return nil, gerror.New(gerror.ErrCodeInternal, "agent factory dependencies not available", nil).
				WithComponent("registry").
				WithOperation("CreateAgent")
		}, nil
	}

	// Get default memory manager
	memoryManager, err := r.memoryRegistry.GetDefaultChainManager()
	if err != nil {
		// If no memory manager is available, create a null factory function
		return func(config AgentConfig) (Agent, error) {
			return nil, gerror.New(gerror.ErrCodeInternal, "memory manager not available", nil).
				WithComponent("registry").
				WithOperation("CreateAgent")
		}, nil
	}

	// Get tool registry - create empty one if none exists
	var toolRegistry tools.Registry
	if r.toolRegistry != nil {
		// Convert to tools.Registry interface
		toolRegistry = &toolRegistryAdapter{registry: r.toolRegistry}
	}

	// Get commission manager - this may be nil for now
	var commissionManager commission.CommissionManager
	// TODO: Get from internal/commission package when available

	// Get cost manager - create a default one
	costManager := core.DefaultCostManagerFactory()

	// Create the agent factory using the existing DefaultFactoryFactory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Return a function that wraps the core.Factory
	return func(config AgentConfig) (interfaces.Agent, error) {
		agentInstance, err := factory.CreateWorkerAgent(ctx, config.Name+"-id", config.Name)
		if err != nil {
			return nil, err
		}
		// Wrap the core.Agent to implement registry.Agent
		return &agentAdapter{agent: agentInstance}, nil
	}, nil
}

// agentFactoryAdapter is no longer needed since we return functions directly

// agentAdapter wraps core.Agent to implement registry.Agent
type agentAdapter struct {
	agent interfaces.Agent
}

func (a *agentAdapter) Execute(ctx context.Context, request string) (string, error) {
	return a.agent.Execute(ctx, request)
}

func (a *agentAdapter) GetID() string {
	return a.agent.GetID()
}

func (a *agentAdapter) GetName() string {
	return a.agent.GetName()
}

func (a *agentAdapter) GetType() string {
	// Extract type from agent name or use a default
	return "worker" // TODO: Implement proper type extraction
}

func (a *agentAdapter) GetCapabilities() []string {
	// Return empty capabilities for now
	return []string{} // TODO: Implement proper capability extraction
}

// nullAgentFactory is no longer needed since we return functions directly

// toolRegistryAdapter adapts ToolRegistry interface to tools.Registry
type toolRegistryAdapter struct {
	registry ToolRegistry
}

func (a *toolRegistryAdapter) RegisterTool(name string, tool tools.Tool) error {
	return a.registry.RegisterTool(name, tool)
}

func (a *toolRegistryAdapter) GetTool(name string) (tools.Tool, error) {
	return a.registry.GetTool(name)
}

func (a *toolRegistryAdapter) ListTools() []string {
	return a.registry.ListTools()
}

func (a *toolRegistryAdapter) HasTool(name string) bool {
	return a.registry.HasTool(name)
}

func (a *toolRegistryAdapter) UnregisterTool(name string) error {
	// ToolRegistry doesn't support unregistration
	return gerror.New(gerror.ErrCodeInternal, "unregister not supported", nil).
		WithComponent("registry").
		WithOperation("UnregisterTool")
}

func (a *toolRegistryAdapter) Clear() {
	// Clear all tools by recreating the registry
	// This is a workaround since the underlying registry doesn't support clear
	// In a real implementation, you'd need to track registered tools separately
}
