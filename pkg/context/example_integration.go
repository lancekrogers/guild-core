// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// This file demonstrates how the context system integrates with the Guild framework
// It shows practical usage patterns for context-aware operations

// ExampleUsage demonstrates the complete context system integration
func ExampleUsage() {
	// 1. Create a new Guild context for a user request
	ctx := NewGuildContext(context.Background())

	// 2. Add session and user information
	ctx = WithSessionID(ctx, "session-12345")
	ctx = WithCostBudget(ctx, 5.0, "USD")
	ctx = WithResourceLimit(ctx, 3, 1024*1024*1024) // 3 concurrent, 1GB

	// 3. Set up mock registry (in real usage, this would be the actual registry)
	mockRegistry := NewMockRegistryProvider()
	ctx = WithRegistryProvider(ctx, mockRegistry)

	// 4. Register some test components
	if err := mockRegistry.Providers().RegisterProvider("openai", &MockProvider{Name: "openai"}); err != nil {
		log.Printf("Failed to register openai provider: %v", err)
		return
	}
	if err := mockRegistry.Providers().RegisterProvider("anthropic", &MockProvider{Name: "anthropic"}); err != nil {
		log.Printf("Failed to register anthropic provider: %v", err)
		return
	}
	if err := mockRegistry.Providers().SetDefaultProvider("anthropic"); err != nil {
		log.Printf("Failed to set default provider: %v", err)
		return
	}

	if err := mockRegistry.Agents().RegisterAgent("coding-agent", &MockAgent{ID: "agent-001", Name: "coding-agent"}); err != nil {
		log.Printf("Failed to register agent: %v", err)
		return
	}
	if err := mockRegistry.Agents().SetDefaultAgent("coding-agent"); err != nil {
		log.Printf("Failed to set default agent: %v", err)
		return
	}

	if err := mockRegistry.Tools().RegisterTool("file-tool", &MockTool{Name: "file-tool"}); err != nil {
		log.Printf("Failed to register tool: %v", err)
		return
	}
	if err := mockRegistry.Tools().EnableTool("file-tool"); err != nil {
		log.Printf("Failed to enable tool: %v", err)
		return
	}

	// 5. Demonstrate context-aware operations
	fmt.Printf("=== Guild Context System Demo ===\n")
	fmt.Printf("Request ID: %s\n", GetRequestID(ctx))
	fmt.Printf("Session ID: %s\n", GetSessionID(ctx))
	fmt.Printf("Context Summary: %s\n\n", ContextSummary(ctx))

	// 6. Demonstrate component access through context
	demonstrateComponentAccess(ctx)

	// 7. Demonstrate context propagation through operations
	demonstrateContextPropagation(ctx)

	// 8. Demonstrate context-aware provider selection
	demonstrateProviderSelection(ctx)

	// 9. Demonstrate agent routing with context
	demonstrateAgentRouting(ctx)
}

func demonstrateComponentAccess(ctx context.Context) {
	fmt.Printf("=== Component Access Demo ===\n")

	// Access provider through context
	provider, err := GetProviderFromContext(ctx, "openai")
	if err != nil {
		fmt.Printf("Error getting provider: %v\n", err)
	} else {
		mockProvider := provider.(*MockProvider)
		fmt.Printf("Retrieved provider: %s\n", mockProvider.Name)
	}

	// Access agent through context
	agent, err := GetAgentFromContext(ctx, "coding-agent")
	if err != nil {
		fmt.Printf("Error getting agent: %v\n", err)
	} else {
		mockAgent := agent.(*MockAgent)
		fmt.Printf("Retrieved agent: %s (ID: %s)\n", mockAgent.Name, mockAgent.ID)
	}

	// Access tool through context
	tool, err := GetToolFromContext(ctx, "file-tool")
	if err != nil {
		fmt.Printf("Error getting tool: %v\n", err)
	} else {
		mockTool := tool.(*MockTool)
		fmt.Printf("Retrieved tool: %s\n", mockTool.Name)
	}

	fmt.Printf("\n")
}

func demonstrateContextPropagation(ctx context.Context) {
	fmt.Printf("=== Context Propagation Demo ===\n")

	// Create operation context
	opCtx, cancel := NewOperationContext(ctx, "code-generation", 30*time.Second)
	defer cancel()

	// Enhance context with component information
	componentCtx := CreateComponentContext(opCtx, "agent", "coding-agent", "execute")

	fmt.Printf("Original Request ID: %s\n", GetRequestID(ctx))
	fmt.Printf("Operation Context Request ID: %s\n", GetRequestID(opCtx))
	fmt.Printf("Component Context Request ID: %s\n", GetRequestID(componentCtx))
	fmt.Printf("Operation: %s\n", GetOperation(componentCtx))
	fmt.Printf("Agent ID: %s\n", GetAgentID(componentCtx))
	fmt.Printf("Span ID: %s\n", GetSpanID(componentCtx))

	// Demonstrate log fields extraction
	fields := LogFields(componentCtx)
	fmt.Printf("Log Fields: %v\n", fields)

	fmt.Printf("\n")
}

func demonstrateProviderSelection(ctx context.Context) {
	fmt.Printf("=== Provider Selection Demo ===\n")

	// Test provider selection for different task types
	taskTypes := []string{"coding", "reasoning", "fast", "local"}

	for _, taskType := range taskTypes {
		provider, err := SelectBestProvider(ctx, taskType, nil)
		if err != nil {
			fmt.Printf("Error selecting provider for %s: %v\n", taskType, err)
		} else {
			fmt.Printf("Best provider for %s: %s\n", taskType, provider)
		}
	}

	fmt.Printf("\n")
}

func demonstrateAgentRouting(ctx context.Context) {
	fmt.Printf("=== Agent Routing Demo ===\n")

	// Test agent selection for different task types
	taskTypes := []string{"coding", "analysis", "general"}

	for _, taskType := range taskTypes {
		agent, err := SelectBestAgent(ctx, taskType, nil)
		if err != nil {
			fmt.Printf("Error selecting agent for %s: %v\n", taskType, err)
		} else {
			fmt.Printf("Best agent for %s: %s\n", taskType, agent)
		}
	}

	fmt.Printf("\n")
}

// Mock implementations for demonstration

// MockAgentRegistry is a mock implementation of AgentRegistry for examples
type MockAgentRegistry struct {
	agents       map[string]interface{}
	defaultAgent string
}

func (m *MockAgentRegistry) RegisterAgent(name string, agent interface{}) error {
	if m.agents == nil {
		m.agents = make(map[string]interface{})
	}
	m.agents[name] = agent
	return nil
}

func (m *MockAgentRegistry) GetAgent(name string) (interface{}, error) {
	if agent, exists := m.agents[name]; exists {
		return agent, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found", name).WithComponent("context").WithOperation("GetAgent")
}

func (m *MockAgentRegistry) ListAgents() []string {
	var names []string
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

func (m *MockAgentRegistry) SetDefaultAgent(name string) error {
	m.defaultAgent = name
	return nil
}

func (m *MockAgentRegistry) GetDefaultAgent() (interface{}, error) {
	if m.defaultAgent == "" {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no default agent set", nil).WithComponent("context").WithOperation("GetDefaultAgent")
	}
	return m.GetAgent(m.defaultAgent)
}

// MockToolRegistry is a mock implementation of ToolRegistry for examples
type MockToolRegistry struct {
	tools   map[string]interface{}
	enabled map[string]bool
}

func (m *MockToolRegistry) RegisterTool(name string, tool interface{}) error {
	if m.tools == nil {
		m.tools = make(map[string]interface{})
	}
	if m.enabled == nil {
		m.enabled = make(map[string]bool)
	}
	m.tools[name] = tool
	m.enabled[name] = true
	return nil
}

func (m *MockToolRegistry) GetTool(name string) (interface{}, error) {
	if tool, exists := m.tools[name]; exists {
		return tool, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "tool '%s' not found", name).WithComponent("context").WithOperation("GetTool")
}

func (m *MockToolRegistry) ListTools() []string {
	var names []string
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

func (m *MockToolRegistry) EnableTool(name string) error {
	if m.enabled == nil {
		m.enabled = make(map[string]bool)
	}
	m.enabled[name] = true
	return nil
}

func (m *MockToolRegistry) DisableTool(name string) error {
	if m.enabled == nil {
		m.enabled = make(map[string]bool)
	}
	m.enabled[name] = false
	return nil
}

func (m *MockToolRegistry) IsToolEnabled(name string) bool {
	if m.enabled == nil {
		return false
	}
	return m.enabled[name]
}

// MockProviderRegistry is a mock implementation of ProviderRegistry for examples
type MockProviderRegistry struct {
	providers       map[string]interface{}
	defaultProvider string
}

func (m *MockProviderRegistry) RegisterProvider(name string, provider interface{}) error {
	if m.providers == nil {
		m.providers = make(map[string]interface{})
	}
	m.providers[name] = provider
	return nil
}

func (m *MockProviderRegistry) GetProvider(name string) (interface{}, error) {
	if provider, exists := m.providers[name]; exists {
		return provider, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not found", name).WithComponent("context").WithOperation("GetProvider")
}

func (m *MockProviderRegistry) ListProviders() []string {
	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

func (m *MockProviderRegistry) SetDefaultProvider(name string) error {
	m.defaultProvider = name
	return nil
}

func (m *MockProviderRegistry) GetDefaultProvider() (interface{}, error) {
	if m.defaultProvider == "" {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no default provider set", nil).WithComponent("context").WithOperation("GetDefaultProvider")
	}
	return m.GetProvider(m.defaultProvider)
}

// MockMemoryRegistry is a mock implementation of MemoryRegistry for examples
type MockMemoryRegistry struct {
	memoryStores map[string]interface{}
	vectorStores map[string]interface{}
}

func (m *MockMemoryRegistry) RegisterMemoryStore(name string, store interface{}) error {
	if m.memoryStores == nil {
		m.memoryStores = make(map[string]interface{})
	}
	m.memoryStores[name] = store
	return nil
}

func (m *MockMemoryRegistry) GetMemoryStore(name string) (interface{}, error) {
	if store, exists := m.memoryStores[name]; exists {
		return store, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not found", name).WithComponent("context").WithOperation("GetMemoryStore")
}

func (m *MockMemoryRegistry) RegisterVectorStore(name string, store interface{}) error {
	if m.vectorStores == nil {
		m.vectorStores = make(map[string]interface{})
	}
	m.vectorStores[name] = store
	return nil
}

func (m *MockMemoryRegistry) GetVectorStore(name string) (interface{}, error) {
	if store, exists := m.vectorStores[name]; exists {
		return store, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "vector store '%s' not found", name).WithComponent("context").WithOperation("GetVectorStore")
}

func (m *MockMemoryRegistry) ListMemoryStores() []string {
	var names []string
	for name := range m.memoryStores {
		names = append(names, name)
	}
	return names
}

func (m *MockMemoryRegistry) ListVectorStores() []string {
	var names []string
	for name := range m.vectorStores {
		names = append(names, name)
	}
	return names
}

// MockRegistryProvider is a mock implementation of RegistryProvider for examples
type MockRegistryProvider struct {
	agentRegistry    *MockAgentRegistry
	toolRegistry     *MockToolRegistry
	providerRegistry *MockProviderRegistry
	memoryRegistry   *MockMemoryRegistry
}

// NewMockRegistryProvider creates a new mock registry provider for examples
func NewMockRegistryProvider() *MockRegistryProvider {
	return &MockRegistryProvider{
		agentRegistry:    &MockAgentRegistry{},
		toolRegistry:     &MockToolRegistry{},
		providerRegistry: &MockProviderRegistry{},
		memoryRegistry:   &MockMemoryRegistry{},
	}
}

func (m *MockRegistryProvider) Agents() AgentRegistry {
	return m.agentRegistry
}

func (m *MockRegistryProvider) Tools() ToolRegistry {
	return m.toolRegistry
}

func (m *MockRegistryProvider) Providers() ProviderRegistry {
	return m.providerRegistry
}

func (m *MockRegistryProvider) Memory() MemoryRegistry {
	return m.memoryRegistry
}

type MockProvider struct {
	Name string
}

func (m *MockProvider) Complete(ctx context.Context, prompt string) (string, error) {
	// Log the context information
	fmt.Printf("Provider %s executing with context: %s\n", m.Name, ContextSummary(ctx))
	return fmt.Sprintf("Response from %s: %s", m.Name, prompt), nil
}

type MockAgent struct {
	ID   string
	Name string
}

func (m *MockAgent) Execute(ctx context.Context, request string) (string, error) {
	// Log the context information
	fmt.Printf("Agent %s executing with context: %s\n", m.Name, ContextSummary(ctx))
	return fmt.Sprintf("Agent %s result: %s", m.Name, request), nil
}

func (m *MockAgent) GetID() string {
	return m.ID
}

func (m *MockAgent) GetName() string {
	return m.Name
}

func (m *MockAgent) GetCapabilities() []string {
	switch m.Name {
	case "coding-agent":
		return []string{"coding", "development", "debugging"}
	default:
		return []string{"general"}
	}
}

func (m *MockAgent) GetStatus() AgentStatus {
	return AgentStatus{
		State:      "idle",
		LastActive: time.Now(),
		Metadata:   make(map[string]interface{}),
	}
}

type MockTool struct {
	Name string
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	fmt.Printf("Tool %s executing with context: %s\n", m.Name, ContextSummary(ctx))
	return fmt.Sprintf("Tool %s result", m.Name), nil
}

// ExampleIntegrationWithRegistry shows how to integrate the context system with a real registry
func ExampleIntegrationWithRegistry() {
	// This would be used in real integration code:

	/*
		// 1. Initialize the system
		config, err := registry.LoadConfig("config.yaml")
		if err != nil {
			log.Fatal(err)
		}

		componentRegistry := registry.NewComponentRegistry()
		err = componentRegistry.Initialize(context.Background(), *config)
		if err != nil {
			log.Fatal(err)
		}

		// 2. Create context with registry
		ctx := context.NewGuildContext(context.Background())
		ctx = context.WithRegistryProvider(ctx, componentRegistry)
		ctx = context.WithConfigProvider(ctx, config)

		// 3. Use context-aware operations
		result, err := context.CompleteWithDefaultProvider(ctx, "Write a function to sort an array")
		if err != nil {
			log.Error("Completion failed", context.LogFields(ctx)...)
		}

		// 4. Route to appropriate agent
		result, err = context.RouteToAgent(ctx, "coding", "Implement binary search", nil)
		if err != nil {
			log.Error("Agent routing failed", context.LogFields(ctx)...)
		}
	*/
}

// ExampleErrorHandling shows how to handle errors with context
func ExampleErrorHandling() {
	ctx := NewGuildContext(context.Background())
	ctx = WithSessionID(ctx, "error-demo-session")

	// Simulate an error with context
	err := gerror.New(gerror.ErrCodeConnection, "provider connection failed", nil).WithComponent("context").WithOperation("ExampleErrorHandling")

	// Log with context
	logger := GetLogger(ctx)
	if logger != nil {
		logger.Error("Operation failed", append(LogFields(ctx), "error", err.Error())...)
	} else {
		// Fallback logging
		fmt.Printf("ERROR [%s]: %v\n", ContextSummary(ctx), err)
	}

	// Error wrapping with context
	contextErr := gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to complete request %s", GetRequestID(ctx)).WithComponent("context").WithOperation("ExampleErrorHandling")
	fmt.Printf("Wrapped error: %v\n", contextErr)
}

// ExampleCostTracking shows how to track costs with context
func ExampleCostTracking() {
	ctx := NewGuildContext(context.Background())
	ctx = WithCostBudget(ctx, 10.0, "USD")

	// Simulate operations that consume budget
	costInfo := GetCostInfo(ctx)
	if costInfo != nil {
		fmt.Printf("Initial budget: $%.2f\n", costInfo.Budget)

		// Simulate cost consumption
		costInfo.Used += 2.50
		fmt.Printf("After operation 1: $%.2f used, $%.2f remaining\n",
			costInfo.Used, costInfo.Budget-costInfo.Used)

		costInfo.Used += 1.75
		fmt.Printf("After operation 2: $%.2f used, $%.2f remaining\n",
			costInfo.Used, costInfo.Budget-costInfo.Used)

		// Check budget
		if costInfo.Used >= costInfo.Budget {
			fmt.Printf("Budget exceeded! Operations should be limited.\n")
		}
	}
}
