// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// RegistryProvider defines the interface for accessing registry from context
// This avoids import cycles while providing strong typing
type RegistryProvider interface {
	// Agents returns the agent registry
	Agents() AgentRegistry

	// Tools returns the tool registry
	Tools() ToolRegistry

	// Providers returns the provider registry
	Providers() ProviderRegistry

	// Memory returns the memory registry
	Memory() MemoryRegistry
}

// AgentRegistry defines the interface for agent management
type AgentRegistry interface {
	RegisterAgent(name string, agent interface{}) error
	GetAgent(name string) (interface{}, error)
	ListAgents() []string
	SetDefaultAgent(name string) error
	GetDefaultAgent() (interface{}, error)
}

// ToolRegistry defines the interface for tool management
type ToolRegistry interface {
	RegisterTool(name string, tool interface{}) error
	GetTool(name string) (interface{}, error)
	ListTools() []string
	EnableTool(name string) error
	DisableTool(name string) error
	IsToolEnabled(name string) bool
}

// ProviderRegistry defines the interface for provider management
type ProviderRegistry interface {
	RegisterProvider(name string, provider interface{}) error
	GetProvider(name string) (interface{}, error)
	ListProviders() []string
	SetDefaultProvider(name string) error
	GetDefaultProvider() (interface{}, error)
}

// MemoryRegistry defines the interface for memory component management
type MemoryRegistry interface {
	RegisterMemoryStore(name string, store interface{}) error
	GetMemoryStore(name string) (interface{}, error)
	RegisterVectorStore(name string, store interface{}) error
	GetVectorStore(name string) (interface{}, error)
	ListMemoryStores() []string
	ListVectorStores() []string
}

// ConfigProvider defines the interface for accessing configuration
type ConfigProvider interface {
	GetProviderConfig(providerName string) (map[string]interface{}, error)
	GetMemoryStoreConfig(storeName string) (map[string]interface{}, error)
	GetToolConfig(toolName string) (map[string]interface{}, error)
	IsToolEnabled(toolName string) bool
}

// ==============================================================================
// Context-aware Registry Functions
// ==============================================================================

// WithRegistryProvider adds a registry provider to the context
func WithRegistryProvider(ctx context.Context, registry RegistryProvider) context.Context {
	return context.WithValue(ctx, RegistryKey, registry)
}

// GetRegistryProvider retrieves the registry provider from context
func GetRegistryProvider(ctx context.Context) (RegistryProvider, error) {
	if registry, ok := ctx.Value(RegistryKey).(RegistryProvider); ok {
		return registry, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "no registry provider found in context", nil).WithComponent("context").WithOperation("GetRegistryProvider")
}

// WithConfigProvider adds a configuration provider to the context
func WithConfigProvider(ctx context.Context, config ConfigProvider) context.Context {
	return context.WithValue(ctx, ConfigKey, config)
}

// GetConfigProvider retrieves the configuration provider from context
func GetConfigProvider(ctx context.Context) (ConfigProvider, error) {
	if config, ok := ctx.Value(ConfigKey).(ConfigProvider); ok {
		return config, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "no configuration provider found in context", nil).WithComponent("context").WithOperation("GetConfigProvider")
}

// ==============================================================================
// Context-aware Component Access
// ==============================================================================

// GetAgentFromContext retrieves an agent by name from the context registry
func GetAgentFromContext(ctx context.Context, name string) (interface{}, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetAgentFromContext")
	}

	agent, err := registry.Agents().GetAgent(name)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get agent").WithComponent("context").WithOperation("GetAgentFromContext").WithDetails("agent_name", name)
	}

	return agent, nil
}

// GetProviderFromContext retrieves a provider by name from the context registry
func GetProviderFromContext(ctx context.Context, name string) (interface{}, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetProviderFromContext")
	}

	provider, err := registry.Providers().GetProvider(name)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get provider").WithComponent("context").WithOperation("GetProviderFromContext").WithDetails("provider_name", name)
	}

	return provider, nil
}

// GetToolFromContext retrieves a tool by name from the context registry
func GetToolFromContext(ctx context.Context, name string) (interface{}, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetToolFromContext")
	}

	tool, err := registry.Tools().GetTool(name)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get tool").WithComponent("context").WithOperation("GetToolFromContext").WithDetails("tool_name", name)
	}

	return tool, nil
}

// GetMemoryStoreFromContext retrieves a memory store by name from the context registry
func GetMemoryStoreFromContext(ctx context.Context, name string) (interface{}, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetMemoryStoreFromContext")
	}

	store, err := registry.Memory().GetMemoryStore(name)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get memory store").WithComponent("context").WithOperation("GetMemoryStoreFromContext").WithDetails("store_name", name)
	}

	return store, nil
}

// GetVectorStoreFromContext retrieves a vector store by name from the context registry
func GetVectorStoreFromContext(ctx context.Context, name string) (interface{}, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("GetVectorStoreFromContext")
	}

	store, err := registry.Memory().GetVectorStore(name)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get vector store").WithComponent("context").WithOperation("GetVectorStoreFromContext").WithDetails("store_name", name)
	}

	return store, nil
}

// ==============================================================================
// Context-aware Configuration Access
// ==============================================================================

// GetProviderConfigFromContext retrieves provider configuration from context
func GetProviderConfigFromContext(ctx context.Context, providerName string) (map[string]interface{}, error) {
	config, err := GetConfigProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get config from context").WithComponent("context").WithOperation("GetProviderConfigFromContext")
	}

	providerConfig, err := config.GetProviderConfig(providerName)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get provider config").WithComponent("context").WithOperation("GetProviderConfigFromContext").WithDetails("provider_name", providerName)
	}

	return providerConfig, nil
}

// GetToolConfigFromContext retrieves tool configuration from context
func GetToolConfigFromContext(ctx context.Context, toolName string) (map[string]interface{}, error) {
	config, err := GetConfigProvider(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get config from context").WithComponent("context").WithOperation("GetToolConfigFromContext")
	}

	toolConfig, err := config.GetToolConfig(toolName)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get tool config").WithComponent("context").WithOperation("GetToolConfigFromContext").WithDetails("tool_name", toolName)
	}

	return toolConfig, nil
}

// IsToolEnabledInContext checks if a tool is enabled via context configuration
func IsToolEnabledInContext(ctx context.Context, toolName string) bool {
	config, err := GetConfigProvider(ctx)
	if err != nil {
		return false // Default to disabled if no config available
	}

	return config.IsToolEnabled(toolName)
}

// ==============================================================================
// Context Enhancement Utilities
// ==============================================================================

// EnhanceContextWithComponent adds component information to context for tracing
func EnhanceContextWithComponent(ctx context.Context, componentType, componentName string) context.Context {
	switch componentType {
	case "agent":
		return WithAgentID(ctx, componentName)
	case "provider":
		return WithProvider(ctx, componentName)
	case "tool":
		return WithTool(ctx, componentName)
	default:
		// For unknown component types, add to metadata
		requestInfo := GetRequestInfo(ctx)
		if requestInfo != nil && requestInfo.Metadata != nil {
			requestInfo.Metadata[componentType] = componentName
		}
		return ctx
	}
}

// CreateComponentContext creates a new context for component operations
func CreateComponentContext(parentCtx context.Context, componentType, componentName, operation string) context.Context {
	// Create operation context
	ctx := WithOperation(parentCtx, operation)

	// Add component information
	ctx = EnhanceContextWithComponent(ctx, componentType, componentName)

	// Generate new span ID for this component operation
	ctx = WithSpanID(ctx, generateSpanID())

	return ctx
}

// generateSpanID creates a simple span ID (using uuid would require import)
func generateSpanID() string {
	// Simple span ID generation - in production you might want something more sophisticated
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}
