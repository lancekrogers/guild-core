# 💰 Cost-Aware Artisans

## Purpose

This document describes the cost awareness system in Guild, enabling artisans to operate under budget constraints and optimize their resource usage while completing commissions.

## Background

In traditional guilds, master craftsmen carefully tracked the cost of materials and labor to ensure profitability. Similarly, our Guild artisans must be mindful of computational and API costs that can quickly accumulate when using external LLM providers and specialized tools.

## Implementation

### Cost Management System

The Guild framework implements a comprehensive cost tracking and optimization system through the `CostManager` component. Each GuildArtisan is equipped with this capability, allowing for:

- Tracking costs across different categories (LLM, tools, storage)
- Setting and enforcing budgets for specific cost types
- Optimizing model selection based on cost considerations
- Reporting detailed cost information upon task completion

### Craftsman Cost Awareness

Craftsmen (worker agents) demonstrate cost-aware behavior through:

1. **Pre-execution Budget Checks**:
   - Before making LLM API calls, estimating potential costs
   - Ensuring there's sufficient budget for the operation
   - Declining expensive operations when budget constraints are tight

2. **Cost-optimized Prompt Construction**:
   - Including budget information in prompts
   - Providing cost-saving strategies to guide LLM reasoning
   - Encouraging efficiency through concise interactions

3. **Tool Usage Optimization**:
   - Evaluating tool costs before execution
   - Preferring lower-cost alternatives when available
   - Properly accounting for tool usage costs

4. **Comprehensive Cost Reporting**:
   - Recording costs in task metadata
   - Generating detailed cost breakdowns for completed tasks
   - Allowing GuildMasters to monitor expense patterns

### GuildMaster Oversight

GuildMasters (manager agents) provide cost governance through:

1. **Budget Setting**:
   - Assigning cost budgets to Craftsmen for specific tasks
   - Adjusting budgets based on task priority and complexity
   - Setting organization-wide cost policies

2. **Cost Monitoring**:
   - Tracking expenses across all supervised Craftsmen
   - Receiving cost reports after task completion
   - Identifying cost patterns and optimization opportunities

3. **Budget Allocation**:
   - Strategically distributing budgets across different tasks
   - Favoring cost-efficient Craftsmen for budget-sensitive work
   - Requesting budget increases when necessary

## Configuration

### Sample Budget Configuration

```yaml
costs:
  # Per-thousand token costs for different LLM models
  api_models:
    claude-3-opus: 15.00
    claude-3-sonnet: 3.00
    gpt-4: 10.00
    gpt-3.5-turbo: 0.50
  
  # Relative cost units for local models
  local_models:
    llama3-8b: 1
    mistral-7b: 0.8
  
  # Per-call costs for various tools
  cli_tools:
    default: 0.0
    search-web: 0.05
    image-generator: 0.10
    
  # Guild-wide budget settings
  budgets:
    daily_maximum: 50.00
    per_task_default: 2.00
    alert_threshold: 40.00
```

## Usage

### Setting Craftsman Budgets

GuildMasters can set budgets when commissioning work:

```go
// Set budget when assigning a task
guildMaster.SetCraftsmanCostBudgets(craftsmanID, 5.0, 1.0) // $5 for LLM, $1 for tools
```

### Checking Costs During Execution

Craftsmen automatically check costs during execution:

```go
// Before making an LLM call, check if it's affordable
estimatedCost := craftsman.costManager.EstimateLLMCost(model, promptTokens, maxTokens)
if !craftsman.costManager.CanAfford(CostTypeLLM, estimatedCost) {
    // Take alternative action or request budget increase
}
```

### Getting Cost Reports

After task completion, cost reports can be retrieved:

```go
// Get full cost report
costReport := craftsman.GetCostReport()

// Access specific cost type
llmCost := costReport["total_costs"].(map[string]float64)[string(CostTypeLLM)]
```

## Integration Points

- **Task Metadata**: Cost information is stored in task metadata
- **Event System**: Cost alerts can be triggered on budget thresholds
- **Kanban Board**: Tasks can display their accumulated costs
- **Objective System**: Objectives can specify cost constraints

## Benefits

- **Predictable Costs**: Avoid surprise expenses from runaway agents
- **Resource Efficiency**: Maximize value from limited API budgets
- **Optimization Insights**: Identify which tasks and agents are most cost-effective
- **Cost Accountability**: Trace expenses back to specific tasks and operations

## Future Enhancements

- Cost prediction and estimation before task execution
- Automated model switching based on budget constraints
- Cost-based prioritization of tasks in the Kanban system
- Integration with organizational billing systems