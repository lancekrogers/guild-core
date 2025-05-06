# 🧠 Meta-Coordination Protocol (MCP) + RAG

Guild implements a system-level protocol for minimizing redundant LLM usage, improving tool adoption, and maintaining high-quality results at low cost. This behavior is governed by two cooperating mechanisms:

- **MCP (Meta-Coordination Protocol)** — enables manager agents to detect inefficiencies, flag optimization opportunities, and enforce coordination rules
- **RAG (Retrieval-Augmented Generation)** — enables all agents to recall past decisions, tool usage, and prompt chains

---

## 🛁 MCP: Coordination & Oversight Layer

Manager agents:

- Detect repeated prompt chains or inefficient LLM use
- Flag repetitive patterns as `tool_candidate` records
- Recommend CLI tools for automation
- Apply heuristics for task cost, similarity, and outcome

### Example MCP Record

```json
{
  "type": "tool_candidate",
  "task_signature": "format markdown tables",
  "example_prompt": "...",
  "estimated_cost_per_use": 8,
  "recommended_tool_name": "format-md"
}
```

---

## 🔍 RAG: Retrieval-Augmented Generation

Used by agents to:

- Recover past prompt chains and outputs
- Retrieve project objectives or known tool usage
- Load examples or summaries from memory instead of re-generating

### Memory Sources

- Qdrant (vector store)
- Markdown objectives
- Completed task summaries
- Tagged decisions or architecture notes

---

## 🔄 Cost-Aware Context Restoration

When resuming or executing a task:

1. RAG restores task-relevant memory and prompt chains
2. MCP metadata informs available tools and costs
3. Agent reasons with cost in mind

   - Prefer efficient tools
   - Justify higher-cost decisions with output trace

---

## 📊 Optimization Loop

1. Agents complete repeated high-cost tasks
2. Manager detects the pattern
3. Flagged as `tool_candidate`
4. User builds CLI tool
5. Tool added to agent config
6. Future prompts invoke CLI instead of LLM

This enables compounding efficiency gains over time and gives users insight into where automation or tooling can save money and latency.
