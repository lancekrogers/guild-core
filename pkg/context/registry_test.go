// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package context

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Mock implementations for testing
type TestMockAgentRegistry struct {
	agents       map[string]interface{}
	defaultAgent string
}

func (m *TestMockAgentRegistry) RegisterAgent(name string, agent interface{}) error {
	if m.agents == nil {
		m.agents = make(map[string]interface{})
	}
	m.agents[name] = agent
	return nil
}

func (m *TestMockAgentRegistry) GetAgent(name string) (interface{}, error) {
	if agent, exists := m.agents[name]; exists {
		return agent, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent '%s' not found", name).WithComponent("context").WithOperation("GetAgent")
}

func (m *TestMockAgentRegistry) ListAgents() []string {
	var names []string
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

func (m *TestMockAgentRegistry) SetDefaultAgent(name string) error {
	m.defaultAgent = name
	return nil
}

func (m *TestMockAgentRegistry) GetDefaultAgent() (interface{}, error) {
	if m.defaultAgent == "" {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no default agent set", nil).WithComponent("context").WithOperation("GetDefaultAgent")
	}
	return m.GetAgent(m.defaultAgent)
}

type TestMockToolRegistry struct {
	tools   map[string]interface{}
	enabled map[string]bool
}

func (m *TestMockToolRegistry) RegisterTool(name string, tool interface{}) error {
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

func (m *TestMockToolRegistry) GetTool(name string) (interface{}, error) {
	if tool, exists := m.tools[name]; exists {
		return tool, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "tool '%s' not found", name).WithComponent("context").WithOperation("GetTool")
}

func (m *TestMockToolRegistry) ListTools() []string {
	var names []string
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

func (m *TestMockToolRegistry) EnableTool(name string) error {
	if m.enabled == nil {
		m.enabled = make(map[string]bool)
	}
	m.enabled[name] = true
	return nil
}

func (m *TestMockToolRegistry) DisableTool(name string) error {
	if m.enabled == nil {
		m.enabled = make(map[string]bool)
	}
	m.enabled[name] = false
	return nil
}

func (m *TestMockToolRegistry) IsToolEnabled(name string) bool {
	if m.enabled == nil {
		return false
	}
	return m.enabled[name]
}

type TestMockProviderRegistry struct {
	providers       map[string]interface{}
	defaultProvider string
}

func (m *TestMockProviderRegistry) RegisterProvider(name string, provider interface{}) error {
	if m.providers == nil {
		m.providers = make(map[string]interface{})
	}
	m.providers[name] = provider
	return nil
}

func (m *TestMockProviderRegistry) GetProvider(name string) (interface{}, error) {
	if provider, exists := m.providers[name]; exists {
		return provider, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not found", name).WithComponent("context").WithOperation("GetProvider")
}

func (m *TestMockProviderRegistry) ListProviders() []string {
	var names []string
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

func (m *TestMockProviderRegistry) SetDefaultProvider(name string) error {
	m.defaultProvider = name
	return nil
}

func (m *TestMockProviderRegistry) GetDefaultProvider() (interface{}, error) {
	if m.defaultProvider == "" {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no default provider set", nil).WithComponent("context").WithOperation("GetDefaultProvider")
	}
	return m.GetProvider(m.defaultProvider)
}

type TestMockMemoryRegistry struct {
	memoryStores map[string]interface{}
	vectorStores map[string]interface{}
}

func (m *TestMockMemoryRegistry) RegisterMemoryStore(name string, store interface{}) error {
	if m.memoryStores == nil {
		m.memoryStores = make(map[string]interface{})
	}
	m.memoryStores[name] = store
	return nil
}

func (m *TestMockMemoryRegistry) GetMemoryStore(name string) (interface{}, error) {
	if store, exists := m.memoryStores[name]; exists {
		return store, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not found", name).WithComponent("context").WithOperation("GetMemoryStore")
}

func (m *TestMockMemoryRegistry) RegisterVectorStore(name string, store interface{}) error {
	if m.vectorStores == nil {
		m.vectorStores = make(map[string]interface{})
	}
	m.vectorStores[name] = store
	return nil
}

func (m *TestMockMemoryRegistry) GetVectorStore(name string) (interface{}, error) {
	if store, exists := m.vectorStores[name]; exists {
		return store, nil
	}
	return nil, gerror.Newf(gerror.ErrCodeNotFound, "vector store '%s' not found", name).WithComponent("context").WithOperation("GetVectorStore")
}

func (m *TestMockMemoryRegistry) ListMemoryStores() []string {
	var names []string
	for name := range m.memoryStores {
		names = append(names, name)
	}
	return names
}

func (m *TestMockMemoryRegistry) ListVectorStores() []string {
	var names []string
	for name := range m.vectorStores {
		names = append(names, name)
	}
	return names
}

type TestMockRegistryProvider struct {
	agentRegistry    *TestMockAgentRegistry
	toolRegistry     *TestMockToolRegistry
	providerRegistry *TestMockProviderRegistry
	memoryRegistry   *TestMockMemoryRegistry
}

func NewTestMockRegistryProvider() *TestMockRegistryProvider {
	return &TestMockRegistryProvider{
		agentRegistry:    &TestMockAgentRegistry{},
		toolRegistry:     &TestMockToolRegistry{},
		providerRegistry: &TestMockProviderRegistry{},
		memoryRegistry:   &TestMockMemoryRegistry{},
	}
}

func (m *TestMockRegistryProvider) Agents() AgentRegistry {
	return m.agentRegistry
}

func (m *TestMockRegistryProvider) Tools() ToolRegistry {
	return m.toolRegistry
}

func (m *TestMockRegistryProvider) Providers() ProviderRegistry {
	return m.providerRegistry
}

func (m *TestMockRegistryProvider) Memory() MemoryRegistry {
	return m.memoryRegistry
}

func TestRegistryProviderContext(t *testing.T) {
	ctx := context.Background()

	// Test getting registry when none exists
	_, err := GetRegistryProvider(ctx)
	if err == nil {
		t.Error("Expected error when getting registry from empty context")
	}

	// Test setting and getting registry
	mockRegistry := NewTestMockRegistryProvider()
	ctx = WithRegistryProvider(ctx, mockRegistry)

	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		t.Fatalf("Failed to get registry from context: %v", err)
	}

	if registry != mockRegistry {
		t.Error("Retrieved registry does not match the one we set")
	}
}

func TestComponentAccessFromContext(t *testing.T) {
	ctx := context.Background()
	mockRegistry := NewTestMockRegistryProvider()
	ctx = WithRegistryProvider(ctx, mockRegistry)

	// Test agent operations
	testAgent := "test-agent-instance"
	err := mockRegistry.Agents().RegisterAgent("test-agent", testAgent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	retrievedAgent, err := GetAgentFromContext(ctx, "test-agent")
	if err != nil {
		t.Fatalf("Failed to get agent from context: %v", err)
	}

	if retrievedAgent.(string) != testAgent {
		t.Error("Retrieved agent does not match registered agent")
	}

	// Test provider operations
	testProvider := "test-provider-instance"
	err = mockRegistry.Providers().RegisterProvider("test-provider", testProvider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	retrievedProvider, err := GetProviderFromContext(ctx, "test-provider")
	if err != nil {
		t.Fatalf("Failed to get provider from context: %v", err)
	}

	if retrievedProvider.(string) != testProvider {
		t.Error("Retrieved provider does not match registered provider")
	}

	// Test tool operations
	testTool := "test-tool-instance"
	err = mockRegistry.Tools().RegisterTool("test-tool", testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	retrievedTool, err := GetToolFromContext(ctx, "test-tool")
	if err != nil {
		t.Fatalf("Failed to get tool from context: %v", err)
	}

	if retrievedTool.(string) != testTool {
		t.Error("Retrieved tool does not match registered tool")
	}

	// Test memory store operations
	testMemoryStore := "test-memory-store-instance"
	err = mockRegistry.Memory().RegisterMemoryStore("test-memory", testMemoryStore)
	if err != nil {
		t.Fatalf("Failed to register memory store: %v", err)
	}

	retrievedMemoryStore, err := GetMemoryStoreFromContext(ctx, "test-memory")
	if err != nil {
		t.Fatalf("Failed to get memory store from context: %v", err)
	}

	if retrievedMemoryStore.(string) != testMemoryStore {
		t.Error("Retrieved memory store does not match registered memory store")
	}

	// Test vector store operations
	testVectorStore := "test-vector-store-instance"
	err = mockRegistry.Memory().RegisterVectorStore("test-vector", testVectorStore)
	if err != nil {
		t.Fatalf("Failed to register vector store: %v", err)
	}

	retrievedVectorStore, err := GetVectorStoreFromContext(ctx, "test-vector")
	if err != nil {
		t.Fatalf("Failed to get vector store from context: %v", err)
	}

	if retrievedVectorStore.(string) != testVectorStore {
		t.Error("Retrieved vector store does not match registered vector store")
	}
}

func TestEnhanceContextWithComponent(t *testing.T) {
	// Start with a Guild context that has proper request info
	ctx := NewGuildContext(context.Background())

	// Test agent enhancement
	ctx = EnhanceContextWithComponent(ctx, "agent", "test-agent")
	agentID := GetAgentID(ctx)
	if agentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got %s", agentID)
	}

	// Test provider enhancement
	ctx = EnhanceContextWithComponent(ctx, "provider", "test-provider")
	provider := GetProvider(ctx)
	if provider != "test-provider" {
		t.Errorf("Expected provider 'test-provider', got %s", provider)
	}

	// Test tool enhancement
	ctx = EnhanceContextWithComponent(ctx, "tool", "test-tool")
	tool := GetTool(ctx)
	if tool != "test-tool" {
		t.Errorf("Expected tool 'test-tool', got %s", tool)
	}

	// Test unknown component type (should be added to metadata)
	ctx = EnhanceContextWithComponent(ctx, "unknown", "test-unknown")
	requestInfo := GetRequestInfo(ctx)
	if requestInfo.Metadata["unknown"] != "test-unknown" {
		t.Errorf("Expected unknown component 'test-unknown' in metadata, got %v", requestInfo.Metadata["unknown"])
	}
}

func TestCreateComponentContext(t *testing.T) {
	parentCtx := NewGuildContext(context.Background())

	ctx := CreateComponentContext(parentCtx, "agent", "test-agent", "execute")

	// Check that operation is set
	operation := GetOperation(ctx)
	if operation != "execute" {
		t.Errorf("Expected operation 'execute', got %s", operation)
	}

	// Check that agent ID is set
	agentID := GetAgentID(ctx)
	if agentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got %s", agentID)
	}

	// Check that span ID is set (should be different from parent)
	spanID := GetSpanID(ctx)
	parentSpanID := GetSpanID(parentCtx)
	if spanID == parentSpanID {
		t.Error("Expected different span ID for component context")
	}

	// Check that parent context values are preserved
	requestID := GetRequestID(ctx)
	parentRequestID := GetRequestID(parentCtx)
	if requestID != parentRequestID {
		t.Error("Request ID should be preserved from parent context")
	}
}
