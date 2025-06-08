# Agent Specialization Routing - Execution Layer

## Routing Request

Select optimal agent(s) for the following task assignment:

**Task**: {{.TaskDescription}}
**Complexity**: {{.ComplexityScore}}/10
**Approach**: {{.RecommendedApproach}}
**Requirements**: {{.AgentRequirements}}

## Available Agents Analysis

{{range .AvailableAgents}}

### {{.Name}} - {{.Role}}

- **Provider**: {{.Provider}} ({{.Model}})
- **Cost**: {{.CostMagnitude}}/10 magnitude ({{.TokenCost}} per 1K tokens)
- **Context**: {{.ContextWindow}} tokens
- **Specializations**: {{join .Specializations ", "}}
- **Tools**: {{join .Tools ", "}}
- **Performance**:
  - Success Rate: {{.SuccessRate}}%
  - Avg Quality: {{.QualityScore}}/10
  - Avg Speed: {{.AvgCompletionTime}}
  - Recent Tasks: {{.RecentTaskCount}}
- **Current Load**: {{.CurrentTasks}} active tasks
- **Availability**: {{.AvailabilityStatus}}

{{end}}

## Routing Decision Required

Based on the task requirements and agent capabilities, provide your routing decision:

```json
{
  "routing_decision": {
    "primary_agent": {
      "agent_id": "<selected agent>",
      "role": "<agent role>",
      "assignment_confidence": <1-10>,
      "estimated_tokens": <token estimate>,
      "rationale": "<why this agent is optimal>"
    },
    "supporting_agents": [
      {
        "agent_id": "<agent id>",
        "role": "<agent role>",
        "responsibility": "<specific task area>",
        "coordination_type": "<parallel|sequential|review>",
        "estimated_tokens": <token estimate>
      }
    ]
  },
  "cost_analysis": {
    "total_estimated_tokens": <total>,
    "cost_breakdown": [
      {"agent": "<agent>", "tokens": <estimate>, "cost": "<$amount>"}
    ],
    "alternative_approaches": [
      {
        "approach": "<description>",
        "agents": ["<agent1>", "<agent2>"],
        "tokens": <estimate>,
        "cost_difference": "<±percentage>",
        "quality_trade_off": "<description>"
      }
    ]
  },
  "execution_plan": {
    "coordination_strategy": "<how agents will work together>",
    "task_distribution": [
      {
        "agent": "<agent>",
        "tasks": ["<task1>", "<task2>"],
        "dependencies": ["<prerequisite>"],
        "deliverables": ["<output1>", "<output2>"]
      }
    ],
    "quality_gates": [
      {
        "checkpoint": "<review point>",
        "reviewer": "<agent or process>",
        "criteria": ["<quality measure>"]
      }
    ],
    "risk_mitigation": [
      {
        "risk": "<potential issue>",
        "mitigation": "<preventive action>",
        "fallback": "<backup plan>"
      }
    ]
  },
  "success_metrics": {
    "completion_criteria": ["<criterion1>", "<criterion2>"],
    "quality_thresholds": {
      "minimum_quality": <score>,
      "target_quality": <score>
    },
    "performance_indicators": ["<metric1>", "<metric2>"]
  }
}
```

## Decision Factors to Consider

1. **Capability Match**: How well do agent specializations align with task requirements?
2. **Cost Efficiency**: What's the optimal balance of quality and token cost?
3. **Context Utilization**: Will the task fit within agent context windows?
4. **Workload Balance**: Are you distributing work fairly across available agents?
5. **Quality Prediction**: Which assignment is most likely to succeed?
6. **Coordination Overhead**: How much management will multi-agent approaches require?

Make your selection based on data-driven analysis of agent capabilities and task requirements.
