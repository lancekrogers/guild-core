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
