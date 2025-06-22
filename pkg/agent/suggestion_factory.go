// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/lsp"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/suggestions"
	"github.com/guild-ventures/guild-core/pkg/templates"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// SuggestionAwareAgentFactory creates agents with suggestion capabilities
type SuggestionAwareAgentFactory struct {
	llmClient         providers.LLMClient
	memoryManager     memory.ChainManager
	toolRegistry      tools.Registry
	commissionManager commission.CommissionManager
	costManager       CostManagerInterface
	suggestionManager suggestions.SuggestionManager
}

// NewSuggestionAwareAgentFactory creates a new suggestion-aware agent factory
func NewSuggestionAwareAgentFactory(
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
) *SuggestionAwareAgentFactory {

	// Create suggestion manager with all providers
	suggestionManager := createDefaultSuggestionManager(toolRegistry)

	return &SuggestionAwareAgentFactory{
		llmClient:         llmClient,
		memoryManager:     memoryManager,
		toolRegistry:      toolRegistry,
		commissionManager: commissionManager,
		costManager:       costManager,
		suggestionManager: suggestionManager,
	}
}

// NewSuggestionAwareAgentFactoryWithManager creates a factory with a custom suggestion manager
func NewSuggestionAwareAgentFactoryWithManager(
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
	suggestionManager suggestions.SuggestionManager,
) *SuggestionAwareAgentFactory {
	return &SuggestionAwareAgentFactory{
		llmClient:         llmClient,
		memoryManager:     memoryManager,
		toolRegistry:      toolRegistry,
		commissionManager: commissionManager,
		costManager:       costManager,
		suggestionManager: suggestionManager,
	}
}

// CreateWorkerAgent creates a new suggestion-aware worker agent
func (f *SuggestionAwareAgentFactory) CreateWorkerAgent(id, name string) EnhancedGuildArtisan {
	return NewSuggestionAwareWorkerAgent(
		id, name,
		f.llmClient,
		f.memoryManager,
		f.toolRegistry,
		f.commissionManager,
		f.costManager,
		f.suggestionManager,
	)
}

// CreateWorkerAgentWithCapabilities creates a worker agent with specific capabilities
func (f *SuggestionAwareAgentFactory) CreateWorkerAgentWithCapabilities(id, name string, capabilities []string) EnhancedGuildArtisan {
	agent := NewSuggestionAwareWorkerAgent(
		id, name,
		f.llmClient,
		f.memoryManager,
		f.toolRegistry,
		f.commissionManager,
		f.costManager,
		f.suggestionManager,
	)

	// Set capabilities on the base worker agent
	agent.WorkerAgent.capabilities = capabilities

	return agent
}

// ConfigureSuggestionProviders allows customization of suggestion providers
func (f *SuggestionAwareAgentFactory) ConfigureSuggestionProviders(
	templateManager templates.TemplateManager,
	lspManager *lsp.Manager,
	customProviders ...suggestions.SuggestionProvider,
) error {

	// Create new suggestion manager with custom configuration
	manager := suggestions.NewSuggestionManager()

	// Register core providers
	commandProvider := suggestions.NewCommandSuggestionProvider()
	if err := manager.RegisterProvider(commandProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register command provider").
			WithComponent("agent.suggestion_factory").
			WithOperation("ConfigureSuggestionProviders")
	}

	followUpProvider := suggestions.NewFollowUpSuggestionProvider()
	if err := manager.RegisterProvider(followUpProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register follow-up provider").
			WithComponent("agent.suggestion_factory").
			WithOperation("ConfigureSuggestionProviders")
	}

	// Register template provider if available
	if templateManager != nil {
		templateProvider := suggestions.NewTemplateSuggestionProvider(templateManager)
		if err := manager.RegisterProvider(templateProvider); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register template provider").
				WithComponent("agent.suggestion_factory").
				WithOperation("ConfigureSuggestionProviders")
		}
	}

	// Register LSP provider if available
	if lspManager != nil {
		lspProvider := suggestions.NewLSPSuggestionProvider(lspManager)
		if err := manager.RegisterProvider(lspProvider); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register LSP provider").
				WithComponent("agent.suggestion_factory").
				WithOperation("ConfigureSuggestionProviders")
		}
	}

	// Register tool provider if tool registry is available
	if f.toolRegistry != nil {
		// Cast to pkg/tools.ToolRegistry which embeds the concrete ToolRegistry
		if pkgRegistry, ok := f.toolRegistry.(*tools.ToolRegistry); ok {
			// Access the embedded concrete ToolRegistry for the suggestion provider
			toolProvider := suggestions.NewToolSuggestionProvider(pkgRegistry.ToolRegistry)
			if err := manager.RegisterProvider(toolProvider); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool provider").
					WithComponent("agent.suggestion_factory").
					WithOperation("ConfigureSuggestionProviders")
			}
		}
	}

	// Register custom providers
	for _, provider := range customProviders {
		if err := manager.RegisterProvider(provider); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register custom provider").
				WithComponent("agent.suggestion_factory").
				WithOperation("ConfigureSuggestionProviders").
				WithDetails("provider", provider.GetMetadata().Name)
		}
	}

	f.suggestionManager = manager
	return nil
}

// GetSuggestionManager returns the current suggestion manager
func (f *SuggestionAwareAgentFactory) GetSuggestionManager() suggestions.SuggestionManager {
	return f.suggestionManager
}

// createDefaultSuggestionManager creates a suggestion manager with default providers
func createDefaultSuggestionManager(toolRegistry tools.Registry) suggestions.SuggestionManager {
	manager := suggestions.NewSuggestionManager()

	// Register core providers
	commandProvider := suggestions.NewCommandSuggestionProvider()
	_ = manager.RegisterProvider(commandProvider)

	followUpProvider := suggestions.NewFollowUpSuggestionProvider()
	_ = manager.RegisterProvider(followUpProvider)

	// Register tool provider if tool registry is available
	if toolRegistry != nil {
		if pkgReg, ok := toolRegistry.(*tools.ToolRegistry); ok {
			toolProvider := suggestions.NewToolSuggestionProvider(pkgReg.ToolRegistry)
			_ = manager.RegisterProvider(toolProvider)
		}
	}

	return manager
}
