# 📚 Using Migrated Components

This document provides examples and best practices for using the newly migrated components in the Guild architecture.

## Cost Tracking System

The cost tracking system provides comprehensive monitoring of resource usage across different cost types.

### Basic Usage

```go
// Create a new cost manager for an agent
costManager := agent.NewCostManager()

// Set budgets for different cost types
costManager.SetBudget(agent.CostTypeLLM, 10.0)       // $10 for LLM usage
costManager.SetBudget(agent.CostTypeTool, 2.5)      // $2.50 for tool usage
costManager.SetBudget(agent.CostTypeStorage, 1.0)   // $1 for storage

// Check if you can afford an operation
estimatedCost := costManager.EstimateLLMCost("claude-3-opus", 1000, 500)
if costManager.CanAfford(agent.CostTypeLLM, estimatedCost) {
    // Proceed with the LLM call
}

// Record costs after operations
costManager.RecordLLMCost("claude-3-opus", 1000, 500, map[string]string{
    "agent_id": "agent-123",
    "task_id":  "task-456",
})

// Get cost report
report := costManager.GetCostReport()
fmt.Printf("Total LLM cost: $%.4f\n", report["total_costs"].(map[string]float64)["llm"])
```

### Integration with Agents

```go
// Example of integrating cost tracking into an agent
type CostAwareAgent struct {
    agent.WorkerAgent
    costManager *agent.CostManager
}

func (a *CostAwareAgent) Execute(ctx context.Context, request string) (string, error) {
    // Estimate cost before execution
    estimatedCost := a.costManager.EstimateLLMCost(
        "claude-3-opus", 
        len(request)/4,  // rough token estimate
        1000,           // max completion tokens
    )
    
    // Check budget
    if !a.costManager.CanAfford(agent.CostTypeLLM, estimatedCost) {
        return "", fmt.Errorf("LLM budget exceeded")
    }
    
    // Execute request
    response, err := a.WorkerAgent.Execute(ctx, request)
    if err != nil {
        return "", err
    }
    
    // Record actual cost (you would get actual token counts from the LLM response)
    a.costManager.RecordLLMCost("claude-3-opus", 
        len(request)/4,    // actual prompt tokens
        len(response)/4,   // actual completion tokens
        map[string]string{
            "agent_id": a.GetID(),
            "request":  request[:50], // truncated for metadata
        },
    )
    
    return response, nil
}
```

## RAG Agent Wrapper

The RAG agent wrapper enhances existing agents with retrieval-augmented generation capabilities.

### Basic Usage

```go
// Create a standard agent
baseAgent := agent.NewWorkerAgent(
    "worker-1",
    "Data Analyst",
    llmClient,
    memoryManager,
    toolRegistry,
    objectiveManager,
)

// Create RAG components
embedder, _ := vector.NewOpenAIEmbedder(apiKey, "text-embedding-ada-002")
retriever, _ := rag.NewRetriever(ctx, embedder, rag.Config{
    CollectionName: "knowledge-base",
    ChunkSize:      1000,
    ChunkOverlap:   200,
    MaxResults:     5,
})

// Wrap the agent with RAG capabilities
ragAgent := rag.NewAgentWrapper(baseAgent, retriever, rag.Config{
    CollectionName: "knowledge-base",
    MaxResults:     5,
})

// Use the enhanced agent
response, err := ragAgent.Execute(ctx, "What is the company's policy on remote work?")
// The agent will automatically retrieve relevant context before responding
```

### Custom Context Enhancement

```go
// You can also manually enhance prompts
enhancedPrompt, err := ragAgent.EnhancePrompt(
    ctx,
    "You are a helpful assistant.",
    "Tell me about the project architecture",
    rag.RetrievalConfig{
        MaxResults:      3,
        MinScore:        0.7,
        IncludeMetadata: true,
    },
)

// The enhanced prompt now includes relevant context from the knowledge base
```

## Tool Registry with Cost Tracking

The enhanced tool registry adds cost tracking to tool usage.

### Basic Usage

```go
// Create a tool registry with cost tracking
registry := tools.NewToolRegistry()

// Register tools with specific costs
shellTool := shell.NewShellTool()
registry.RegisterToolWithCost(shellTool, 0.01)  // $0.01 per use

webTool := http.NewHTTPTool()
registry.RegisterToolWithCost(webTool, 0.05)    // $0.05 per use

// Execute tools with cost tracking
result, cost, err := registry.ExecuteToolWithCostTracking(
    ctx,
    "shell",
    `{"command": "ls -la"}`,
)

fmt.Printf("Tool execution cost: $%.4f\n", cost)
```

### Integration with Cost Manager

```go
// Integrate tool costs with the agent's cost manager
func (a *CostAwareAgent) ExecuteTool(ctx context.Context, toolName string, input string) (*tools.ToolResult, error) {
    // Check if we can afford the tool
    toolCost := a.toolRegistry.GetToolCost(toolName)
    if !a.costManager.CanAfford(agent.CostTypeTool, toolCost) {
        return nil, fmt.Errorf("tool budget exceeded for %s", toolName)
    }
    
    // Execute the tool
    result, cost, err := a.toolRegistry.ExecuteToolWithCostTracking(ctx, toolName, input)
    if err != nil {
        return nil, err
    }
    
    // Record the cost
    a.costManager.RecordToolCost(toolName, map[string]string{
        "agent_id":   a.GetID(),
        "tool_input": input[:100], // truncated for metadata
    })
    
    return result, nil
}
```

## Complete Example: Cost-Aware RAG Agent

Here's a complete example that combines all the migrated components:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/blockhead-consulting/guild/pkg/agent"
    "github.com/blockhead-consulting/guild/pkg/memory/rag"
    "github.com/blockhead-consulting/guild/pkg/memory/vector"
    "github.com/blockhead-consulting/guild/pkg/tools"
)

type EnhancedAgent struct {
    *rag.AgentWrapper
    costManager  *agent.CostManager
    toolRegistry *tools.ToolRegistry
}

func NewEnhancedAgent(
    baseAgent agent.GuildArtisan,
    retriever *rag.Retriever,
    ragConfig rag.Config,
) *EnhancedAgent {
    // Create cost manager
    costManager := agent.NewCostManager()
    costManager.SetBudget(agent.CostTypeLLM, 10.0)
    costManager.SetBudget(agent.CostTypeTool, 2.0)
    
    // Create tool registry
    toolRegistry := tools.NewToolRegistry()
    
    // Create RAG wrapper
    wrapper := rag.NewAgentWrapper(baseAgent, retriever, ragConfig)
    
    return &EnhancedAgent{
        AgentWrapper: wrapper,
        costManager:  costManager,
        toolRegistry: toolRegistry,
    }
}

func (a *EnhancedAgent) Execute(ctx context.Context, request string) (string, error) {
    // Estimate cost for the enhanced request
    enhancedRequest, _ := a.enhanceRequestWithRAG(ctx, request)
    estimatedCost := a.costManager.EstimateLLMCost(
        "claude-3-sonnet",
        len(enhancedRequest)/4,
        2000, // max tokens
    )
    
    // Check budget
    if !a.costManager.CanAfford(agent.CostTypeLLM, estimatedCost) {
        return "", fmt.Errorf("LLM budget exceeded")
    }
    
    // Execute with RAG enhancement
    response, err := a.AgentWrapper.Execute(ctx, request)
    if err != nil {
        return "", err
    }
    
    // Record cost
    a.costManager.RecordLLMCost(
        "claude-3-sonnet",
        len(enhancedRequest)/4,
        len(response)/4,
        map[string]string{
            "agent_id": a.GetID(),
            "enhanced": "true",
        },
    )
    
    return response, nil
}

func (a *EnhancedAgent) GetCostReport() map[string]interface{} {
    return a.costManager.GetCostReport()
}
```

## Best Practices

1. **Always Set Budgets**: Define budgets for all cost types to prevent runaway costs
2. **Estimate Before Executing**: Use estimation functions before expensive operations
3. **Record All Costs**: Track every operation to maintain accurate cost accounting
4. **Monitor Regularly**: Check cost reports periodically to identify optimization opportunities
5. **Graceful Degradation**: Handle budget exceeded scenarios gracefully
6. **Metadata Rich**: Include relevant metadata with cost records for better analysis

## Configuration Tips

```yaml
# Example configuration for cost-aware agents
agents:
  data_analyst:
    type: enhanced
    budgets:
      llm: 10.00
      tool: 2.50
      storage: 1.00
    rag_config:
      collection_name: company_knowledge
      max_results: 5
      min_score: 0.7
    tools:
      - name: shell
        cost: 0.01
      - name: web_search
        cost: 0.05
      - name: file_read
        cost: 0.001
```

## Conclusion

The migrated components provide powerful capabilities for building cost-aware, context-enhanced agents. By combining cost tracking, RAG enhancement, and tool management, you can create sophisticated agents that operate efficiently within defined budgets while leveraging relevant knowledge.