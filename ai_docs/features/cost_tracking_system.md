# 💰 Guild Cost Tracking System

## Overview

The Cost Tracking System serves as the Guild's accounting office, ensuring that all Guild operations remain within allocated budgets while providing transparency into resource usage. This document provides a detailed explanation of the cost tracking features implemented throughout the Guild framework.

## Cost Manager (Treasury)

The central component of the cost tracking system is the `CostManager`, a dedicated ledger keeper that:

- Tracks expenses across different cost categories
- Manages budgets for each cost category
- Records individual expense items with metadata
- Provides cost reporting capabilities
- Enforces budget limits
- Analyzes cost efficiency

## Cost Types (Expense Categories)

The Guild accounting system tracks several types of expenses:

| Cost Type | Identifier | Description |
|-----------|------------|-------------|
| LLM Usage | `CostTypeLLM` | Costs associated with language model API calls |
| Embedding | `CostTypeEmbedding` | Costs for creating and storing vector embeddings |
| Tool Usage | `CostTypeTool` | Costs for using specialized tools |
| Storage | `CostTypeStorage` | Costs for data storage and retrieval |
| Compute | `CostTypeCompute` | Costs for computational resources |

## Budget Management

Each GuildArtisan maintains its own treasury to manage budgets:

```go
// Setting budgets for an artisan
artisan.SetCostBudget(CostTypeLLM, 10.0)  // $10 for LLM usage
artisan.SetCostBudget(CostTypeTool, 2.5)  // $2.50 for tool usage

// GuildMasters can set budgets for Craftsmen
guildMaster.SetCraftsmanCostBudgets(craftsmanID, 5.0, 1.0)
```

## Cost Recording

Costs are recorded whenever resources are used:

```go
// Recording LLM costs
costManager.RecordLLMCost(model, promptTokens, completionTokens, metadata)

// Recording tool costs
costManager.RecordToolCost(toolName, metadata)
```

Each cost record includes:
- Cost amount
- Cost unit (USD, tokens, etc.)
- Timestamp
- Description
- Associated metadata (agent ID, task ID, etc.)

## Cost-Aware Behavior

Guild members demonstrate cost awareness through:

### 1. Pre-execution Budget Verification

Before executing expensive operations, Guild members check if they have sufficient funds:

```go
// Estimate cost for an LLM call
estimatedCost := costManager.EstimateLLMCost(model, promptTokens, maxCompletionTokens)

// Check if we can afford it
if !costManager.CanAfford(CostTypeLLM, estimatedCost) {
    // Take alternative action or fail gracefully
}
```

### 2. Cost-Optimized Decision Making

Guild members can select the most cost-efficient options:

```go
// Select the most cost-efficient model for the task
bestModel, err := costManager.SelectCostEfficientModel(availableModels, promptTokens, requiredCompletionTokens)
```

### 3. Prompt Augmentation

Guild members include cost awareness in their prompts:

```
## Cost Awareness
You must optimize for cost efficiency while completing this task.
LLM Budget: $5.0000 (Used: $1.2345, Remaining: $3.7655)
Tool Budget: $1.0000 (Used: $0.1500, Remaining: $0.8500)

Cost-saving strategies:
1. Break complex tasks into smaller steps to avoid long completions
2. Use tools and memory efficiently to avoid redundant LLM calls
3. Be concise in your reasoning to minimize token usage
```

## Cost Reporting

### 1. Task-Level Reporting

When a task is completed, costs are recorded in the task metadata:

```go
task.Metadata["llm_cost"] = "1.2345"
task.Metadata["tool_cost"] = "0.1500"
task.Metadata["total_cost"] = "1.3845"
```

### 2. Agent-Level Reporting

Each Guild member can generate cost reports:

```go
costReport := guildMember.GetCostReport()
```

The report includes:
- Total costs by category
- Budget allocation and remaining funds
- Cost breakdown by model and tool
- Count of cost records

### 3. Guild-Level Reporting

The GuildMaster can generate consolidated reports:

```go
for agentID, agent := range guildMaster.workerAgents {
    agentCosts := agent.GetCostReport()
    // Aggregate costs
}
```

## Implementation Details

### CostManager Structure

```go
type CostManager struct {
    // Map of cost type to records
    costs map[CostType][]*CostRecord
    
    // Budgets by cost type
    budget map[CostType]float64
    
    // Total costs by type
    totalCost map[CostType]float64
    
    // Default unit for costs
    defaultUnit CostUnit
    
    // Model and tool costs
    modelCosts map[string]float64
    toolCosts map[string]float64
    
    // Thread safety
    mu sync.RWMutex
}
```

### Cost Record Structure

```go
type CostRecord struct {
    // Type of cost
    Type CostType
    
    // Amount of cost
    Amount float64
    
    // Unit of cost measurement
    Unit CostUnit
    
    // Description of the cost
    Description string
    
    // When the cost was incurred
    Timestamp time.Time
    
    // Additional information
    Metadata map[string]string
}
```

## Integration Points

### 1. GuildArtisan Interface

The cost tracking capabilities are exposed through the GuildArtisan interface:

```go
type GuildArtisan interface {
    // Other methods...
    
    // Set the budget for a specific cost type
    SetCostBudget(costType CostType, amount float64)
    
    // Get a report of all costs incurred
    GetCostReport() map[string]interface{}
}
```

### 2. Craftsman Implementation

The Craftsman implementation includes cost tracking in its execution loop:

```go
func (a *Craftsman) executeLoop(ctx context.Context, chainID string, promptTokens int) error {
    // Check budget before LLM call
    // Record costs after successful call
    // Check budget before tool execution
    // Record tool costs
}
```

### 3. GuildMaster Integration

The GuildMaster can manage budgets for its Craftsmen:

```go
func (a *GuildMaster) handleAssignTask(ctx context.Context, params map[string]interface{}) (string, error) {
    // Extract budget parameters
    // Set budgets for assigned agent
    // Include budget in task assignment
}
```

## Configuration

Cost-related configurations are specified in the Guild YAML:

```yaml
costs:
  api_models:
    claude-3-opus: 15.00
    gpt-4: 10.00
  local_models:
    llama3-8b: 1
  cli_tools:
    default: 0.0
    search-web: 0.05
  budgets:
    daily_maximum: 50.00
    per_task_default: 2.00
```

## Best Practices

1. **Set Default Budgets**: Always establish default budgets for all Guild members
2. **Track All Cost Types**: Ensure all cost types are tracked for complete visibility
3. **Budget Hierarchically**: Set budgets at Guild, GuildMaster, and Craftsman levels
4. **Monitor Regularly**: Check cost reports regularly to identify optimization opportunities
5. **Optimize High-Cost Operations**: Focus optimization efforts on the most expensive operations
6. **Use Metadata**: Include rich metadata with cost records for better analysis
7. **Adjust Dynamically**: Update budgets based on task complexity and priority

## Future Enhancements

1. **Historical Analysis**: Track cost trends over time
2. **Budget Alerts**: Notify when approaching budget thresholds
3. **Cost Forecasting**: Predict costs for planned operations
4. **Dynamic Budgeting**: Adjust budgets based on operational results
5. **Cost-Based Prioritization**: Prioritize tasks based on budget efficiency