package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
)

// SimpleAgentDemo demonstrates the core agent system functionality
func main() {
	fmt.Println("=== Guild Agent System Simple Demo ===")
	fmt.Println("This demo shows the context-aware agent system")
	fmt.Println()

	if err := runSimpleDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}
}

func runSimpleDemo() error {
	// 1. Create Guild context
	ctx := guildcontext.NewGuildContext(context.Background())
	ctx = guildcontext.WithSessionID(ctx, "simple-demo-001")
	ctx = guildcontext.WithCostBudget(ctx, 5.0, "USD")
	
	fmt.Printf("Created Guild context: %s\n", guildcontext.ContextSummary(ctx))
	fmt.Println()

	// 2. Set up a simple mock registry
	mockRegistry := &SimpleMockRegistry{
		agents:    make(map[string]guildcontext.AgentClient),
		providers: make(map[string]interface{}),
	}
	
	ctx = guildcontext.WithRegistryProvider(ctx, mockRegistry)
	fmt.Println("✓ Mock registry initialized")

	// 3. Create and register agents
	if err := setupSimpleAgents(ctx, mockRegistry); err != nil {
		return fmt.Errorf("failed to setup agents: %w", err)
	}
	
	fmt.Println("✓ Agents configured")

	// 4. Set up mock provider
	mockProvider := &SimpleMockProvider{Name: "demo-provider"}
	mockRegistry.providers["mock-provider"] = mockProvider
	mockRegistry.defaultProvider = "mock-provider"
	
	fmt.Println("✓ Provider configured")
	fmt.Println()

	// 5. Demonstrate agent operations
	if err := demonstrateSimpleAgentOperations(ctx); err != nil {
		return fmt.Errorf("agent demonstration failed: %w", err)
	}

	// 6. Show system status
	showSimpleSystemStatus(ctx, mockRegistry)

	fmt.Println("\n=== Simple Demo Complete ===")
	return nil
}

func setupSimpleAgents(ctx context.Context, registry *SimpleMockRegistry) error {
	// Create different types of agents
	agents := []struct {
		name         string
		agentType    string
		capabilities []string
		systemPrompt string
	}{
		{
			name:         "worker",
			agentType:    "worker",
			capabilities: []string{"general", "completion"},
			systemPrompt: "You are a general-purpose worker agent.",
		},
		{
			name:         "coding-agent",
			agentType:    "specialist",
			capabilities: []string{"coding", "development", "debugging"},
			systemPrompt: "You are a specialized coding agent focused on software development.",
		},
		{
			name:         "analysis-agent",
			agentType:    "specialist",
			capabilities: []string{"analysis", "reasoning", "research"},
			systemPrompt: "You are a specialized analysis agent focused on research and reasoning.",
		},
	}

	for _, agentConfig := range agents {
		agent := agent.NewContextAwareAgent(
			agentConfig.name,
			agentConfig.name,
			agentConfig.agentType,
			agentConfig.capabilities,
		)
		agent.SetSystemPrompt(agentConfig.systemPrompt)
		
		registry.agents[agentConfig.name] = agent
		fmt.Printf("  ✓ Registered agent: %s (%s)\n", agentConfig.name, agentConfig.agentType)
	}

	// Set default agent
	registry.defaultAgent = "worker"
	
	return nil
}

func demonstrateSimpleAgentOperations(ctx context.Context) error {
	fmt.Println("=== Agent Operations Demo ===")

	tests := []struct {
		name     string
		agent    string
		request  string
		taskType string
	}{
		{
			name:     "General Task",
			agent:    "worker",
			request:  "Explain the concept of context-aware programming",
			taskType: "general",
		},
		{
			name:     "Coding Task",
			agent:    "coding-agent",
			request:  "Write a function to calculate the factorial of a number",
			taskType: "coding",
		},
		{
			name:     "Analysis Task",
			agent:    "analysis-agent",
			request:  "Analyze the benefits of microservices architecture",
			taskType: "analysis",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Agent: %s\n", test.agent)
		fmt.Printf("Request: %s\n", test.request)

		// Create operation context
		opCtx, cancel := guildcontext.NewOperationContext(ctx, test.taskType, 30*time.Second)
		defer cancel()

		// Execute with specific agent
		startTime := time.Now()
		result, err := guildcontext.ExecuteWithAgent(opCtx, test.agent, test.request)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Result: %s\n", result)
			fmt.Printf("⏱️  Duration: %v\n", duration)
		}

		// Show context information
		fmt.Printf("📊 Context: %s\n", guildcontext.ContextSummary(opCtx))
	}

	return nil
}

func showSimpleSystemStatus(ctx context.Context, registry *SimpleMockRegistry) {
	fmt.Println("\n=== System Status ===")

	// Show agent statuses
	fmt.Println("\n📋 Agent Status:")
	for agentName, agent := range registry.agents {
		status := agent.GetStatus()
		fmt.Printf("  %s: %s (Tasks: %d, Success: %d, Errors: %d)\n",
			agentName, status.State, status.TaskCount, status.SuccessCount, status.ErrorCount)
	}

	// Show cost information
	costInfo := guildcontext.GetCostInfo(ctx)
	if costInfo != nil {
		fmt.Printf("\n💰 Cost Summary:\n")
		fmt.Printf("  Budget: $%.2f %s\n", costInfo.Budget, costInfo.Currency)
		fmt.Printf("  Used: $%.4f\n", costInfo.Used)
		fmt.Printf("  Remaining: $%.4f\n", costInfo.Budget-costInfo.Used)
	}

	// Show registry information
	fmt.Printf("\n🔧 Registry Status:\n")
	fmt.Printf("  Agents: %v\n", registry.ListAgents())
	fmt.Printf("  Default Agent: %s\n", registry.defaultAgent)
	fmt.Printf("  Providers: %v\n", registry.ListProviders())

	fmt.Printf("\n📈 Session Summary:\n")
	fmt.Printf("  Session ID: %s\n", guildcontext.GetSessionID(ctx))
	fmt.Printf("  Request ID: %s\n", guildcontext.GetRequestID(ctx))
	fmt.Printf("  Context: %s\n", guildcontext.ContextSummary(ctx))
}

// SimpleMockRegistry provides a basic registry implementation for demo
type SimpleMockRegistry struct {
	agents          map[string]guildcontext.AgentClient
	providers       map[string]interface{}
	defaultAgent    string
	defaultProvider string
}

func (r *SimpleMockRegistry) Agents() guildcontext.AgentRegistry {
	return &SimpleMockAgentRegistry{r}
}

func (r *SimpleMockRegistry) Tools() guildcontext.ToolRegistry {
	return &SimpleMockToolRegistry{}
}

func (r *SimpleMockRegistry) Providers() guildcontext.ProviderRegistry {
	return &SimpleMockProviderRegistry{r}
}

func (r *SimpleMockRegistry) Memory() guildcontext.MemoryRegistry {
	return &SimpleMockMemoryRegistry{}
}

func (r *SimpleMockRegistry) ListAgents() []string {
	var names []string
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

func (r *SimpleMockRegistry) ListProviders() []string {
	var names []string
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// SimpleMockAgentRegistry implements guildcontext.AgentRegistry
type SimpleMockAgentRegistry struct {
	registry *SimpleMockRegistry
}

func (r *SimpleMockAgentRegistry) RegisterAgent(name string, agent interface{}) error {
	if contextAgent, ok := agent.(guildcontext.AgentClient); ok {
		r.registry.agents[name] = contextAgent
		return nil
	}
	return fmt.Errorf("agent does not implement AgentClient interface")
}

func (r *SimpleMockAgentRegistry) GetAgent(name string) (interface{}, error) {
	if agent, exists := r.registry.agents[name]; exists {
		return agent, nil
	}
	return nil, fmt.Errorf("agent '%s' not found", name)
}

func (r *SimpleMockAgentRegistry) ListAgents() []string {
	return r.registry.ListAgents()
}

func (r *SimpleMockAgentRegistry) SetDefaultAgent(name string) error {
	r.registry.defaultAgent = name
	return nil
}

func (r *SimpleMockAgentRegistry) GetDefaultAgent() (interface{}, error) {
	if r.registry.defaultAgent == "" {
		return nil, fmt.Errorf("no default agent set")
	}
	return r.GetAgent(r.registry.defaultAgent)
}

// SimpleMockProviderRegistry implements guildcontext.ProviderRegistry
type SimpleMockProviderRegistry struct {
	registry *SimpleMockRegistry
}

func (r *SimpleMockProviderRegistry) RegisterProvider(name string, provider interface{}) error {
	r.registry.providers[name] = provider
	return nil
}

func (r *SimpleMockProviderRegistry) GetProvider(name string) (interface{}, error) {
	if provider, exists := r.registry.providers[name]; exists {
		return provider, nil
	}
	return nil, fmt.Errorf("provider '%s' not found", name)
}

func (r *SimpleMockProviderRegistry) ListProviders() []string {
	return r.registry.ListProviders()
}

func (r *SimpleMockProviderRegistry) SetDefaultProvider(name string) error {
	r.registry.defaultProvider = name
	return nil
}

func (r *SimpleMockProviderRegistry) GetDefaultProvider() (interface{}, error) {
	if r.registry.defaultProvider == "" {
		return nil, fmt.Errorf("no default provider set")
	}
	return r.GetProvider(r.registry.defaultProvider)
}

// SimpleMockToolRegistry implements guildcontext.ToolRegistry (minimal)
type SimpleMockToolRegistry struct{}

func (r *SimpleMockToolRegistry) RegisterTool(name string, tool interface{}) error { return nil }
func (r *SimpleMockToolRegistry) GetTool(name string) (interface{}, error)         { return nil, fmt.Errorf("not implemented") }
func (r *SimpleMockToolRegistry) ListTools() []string                              { return []string{} }
func (r *SimpleMockToolRegistry) EnableTool(name string) error                     { return nil }
func (r *SimpleMockToolRegistry) DisableTool(name string) error                    { return nil }
func (r *SimpleMockToolRegistry) IsToolEnabled(name string) bool                   { return false }

// SimpleMockMemoryRegistry implements guildcontext.MemoryRegistry (minimal)
type SimpleMockMemoryRegistry struct{}

func (r *SimpleMockMemoryRegistry) RegisterMemoryStore(name string, store interface{}) error { return nil }
func (r *SimpleMockMemoryRegistry) GetMemoryStore(name string) (interface{}, error)          { return nil, fmt.Errorf("not implemented") }
func (r *SimpleMockMemoryRegistry) RegisterVectorStore(name string, store interface{}) error { return nil }
func (r *SimpleMockMemoryRegistry) GetVectorStore(name string) (interface{}, error)          { return nil, fmt.Errorf("not implemented") }
func (r *SimpleMockMemoryRegistry) ListMemoryStores() []string                               { return []string{} }
func (r *SimpleMockMemoryRegistry) ListVectorStores() []string                               { return []string{} }

// SimpleMockProvider for demonstration purposes
type SimpleMockProvider struct {
	Name string
}

func (m *SimpleMockProvider) Complete(ctx context.Context, prompt string) (string, error) {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)
	
	// Log context for demonstration
	fmt.Printf("    🤖 Mock provider '%s' processing with context: %s\n", 
		m.Name, guildcontext.ContextSummary(ctx))
	
	// Generate a mock response based on the agent type
	agentID := guildcontext.GetAgentID(ctx)
	switch {
	case agentID == "coding-agent":
		return fmt.Sprintf("Mock coding response: Here's a solution for your request: %s", prompt), nil
	case agentID == "analysis-agent":
		return fmt.Sprintf("Mock analysis response: After analyzing your request: %s", prompt), nil
	default:
		return fmt.Sprintf("Mock response from %s: Processed your request about: %s", m.Name, prompt), nil
	}
}