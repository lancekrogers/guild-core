# 🤖 Agent & Guild Behavior

## 🧠 Agent Behavior

- Execute tasks via LLM API
- Access tools via interfaces (email, HTTP, file, search, etc.)
- React to feedback from other agents or humans (event loop)
- Recursively break down tasks based on prompts and goals
- Maintain each agent’s personal Kanban board of tasks
- Move tasks through the board as they’re completed
- Refactor or rewrite task specs dynamically when appropriate

## 🛈 Guild Behavior

- Coordinate task execution among agents
- Ask the user for input when needed
- Aggregate results toward a shared guild-level goal
- Concurrent execution of agent actions using goroutines and channels

## 💰 Cost-Aware Agent Behavior

- Agents inspect configured `costs` when selecting tools or planning prompt chains
- They default to lower-cost tools when functionally equivalent
- High-cost actions are logged with reasoning justification
- Costs are used by managers to reassign or rebalance work
- Cost for models and actions need to configured by users with command line tools defaulting to 0 cost.
- Cost configuration should use fibonaci scales like in jira with an added 0 for command line tools and self hosted scripts (ex: 0, 1, 3, 5, 8)
- Cost for non-locally hosted models should be configured by the user with the exact pricing from the provider based on million token pricing
  - The magnitude cost should be determined through code on initial runs and config updates, stored in boltdb and memory. The magnitude pricing and real pricing should be stored in the database. Magnitude (0,1,3,5,8) should be calculated based on all available models. If all models have the same pricing, the price should be 8.
- Boltdb should contain the models and a system of determining a models capabilities needs to be designed

## 🧹 Cost Estimation Interface (Go)

```go
type CostEstimator interface {
    EstimateToolCost(toolName string) int
    EstimateModelCost(modelName string, tokens int) int
}
```

- Used internally by agents and managers to compare planning paths
- Default implementation reads from YAML config
- Can later support runtime heuristics, token estimation, or cost dashboards
