# 📚 Guild Terminology Reference

This document serves as a comprehensive glossary of guild-themed terminology used throughout the Guild framework.

## Core Guild Terminology

| Term | Technical Equivalent | Description |
|------|---------------------|-------------|
| **Guild** | Agent Organization | A coordinated group of AI agents working together toward common objectives with shared resources and governance. |
| **GuildHall** | Project Root/Runtime | The central location where Guild operations take place, containing all resources needed by the Guild members. |
| **GuildArtisan** | Agent Interface | The base interface for all AI agents in the system, defining the core capabilities of guild members. |
| **GuildMember** | Base Agent | The foundational implementation of an agent, containing common functionality for all guild artisans. |
| **GuildMaster** | Manager Agent | A senior artisan responsible for coordinating other Guild members, planning work, and managing resources. |
| **Craftsman** | Worker Agent | A specialized artisan that performs specific tasks using tools and LLM capabilities. |
| **Guild Charter** | Objective | A formal document describing the purpose, goals, and requirements for the Guild's work. |
| **Commission** | Task | A specific piece of work assigned to a Guild member. |
| **Guild Ledger** | Kanban Board | The system for tracking all commissions, their status, and assignments. |
| **Guild Archives** | Memory System | The persistent storage system that maintains the Guild's knowledge and history. |
| **Guild Tool** | Tool Interface | Specialized implements that Guild members can use to accomplish their tasks. |
| **Tool Registry** | Tool Registry | The system that manages available tools and their usage. |
| **Guild Seal** | API Key | Authentication credentials that allow access to external services. |
| **Tradecraft** | Domain Knowledge | Specialized knowledge and techniques used by Guild members. |
| **Guild Journal** | Chain Memory | The record of interactions and reasoning for a specific commission. |

## Cost-Related Terminology

| Term | Technical Equivalent | Description |
|------|---------------------|-------------|
| **Cost Ledger** | Cost Manager | The system for tracking and managing expenses incurred by Guild members. |
| **Resource Budget** | Cost Budget | Allocated funds or resources for specific cost types or commissions. |
| **Treasury** | Cost Report | Comprehensive accounting of all costs incurred by the Guild. |
| **Material Costs** | LLM Costs | Expenses related to using language models. |
| **Tool Tithe** | Tool Costs | Expenses related to using specialized tools. |
| **Treasury Warden** | Cost-Aware Behavior | The capability to operate within budget constraints and optimize for cost efficiency. |

## Communication Terminology

| Term | Technical Equivalent | Description |
|------|---------------------|-------------|
| **Guild Messenger** | ZeroMQ | The communication system used for event distribution between Guild members. |
| **Summons** | Event | A notification sent to Guild members about tasks or status changes. |
| **Guild Council** | Orchestrator | The governing body that manages the overall operation of the Guild. |
| **Proclamation** | Publish Message | A message broadcast to multiple Guild members simultaneously. |
| **Petition** | Subscribe Message | A request to receive specific types of messages or notifications. |

## Memory and Knowledge Terminology

| Term | Technical Equivalent | Description |
|------|---------------------|-------------|
| **Archivist** | Chain Manager | The component responsible for managing memory chains and context. |
| **Guild Chronicles** | Prompt Chain | The sequential record of interactions for a specific commission. |
| **Knowledge Repository** | Vector Store | The system for storing and retrieving information based on semantic meaning. |
| **Tome** | Document | A specific piece of stored knowledge or documentation. |
| **Guild Codex** | Embeddings | Vector representations of knowledge that allow for semantic search. |

## Task Management Terminology

| Term | Technical Equivalent | Description |
|------|---------------------|-------------|
| **Commission Board** | Kanban Board | The visual representation of all current Guild commissions. |
| **Work Queue** | Todo List | The list of pending commissions awaiting assignment. |
| **Workbench** | In-Progress Tasks | Commissions currently being worked on by Guild members. |
| **Inspection Chamber** | Blocked Tasks | Commissions that require review or are blocked on dependencies. |
| **Hall of Completed Works** | Completed Tasks | Commissions that have been successfully completed. |
| **Commission Priority** | Task Priority | The relative importance assigned to a commission. |

## Implementation Guidance

When extending the Guild system, consider using terminology consistent with these guild metaphors. For example:

- When adding new agent types, use terms like "Apprentice", "Journeyman", or "Specialist"
- When adding new tool capabilities, use craft-related verbs like "forge", "inscribe", or "appraise"
- When adding new messaging features, use terms like "herald", "decree", or "scroll"

```go
// Example code using guild-themed naming
func (g *GuildMaster) CommissionWork(ctx context.Context, charter *Commission) error {
    // Implementation details
}

func (c *Craftsman) CraftSolution(ctx context.Context) (Craftsmanship, error) {
    // Implementation details
}

func (l *CostLedger) RecordMaterialExpense(materialType, quantity, cost) {
    // Implementation details
}
```

## Benefits of Consistent Terminology

1. **Cohesive Mental Model**: A consistent metaphor helps developers understand the system's components and their relationships.
2. **Memorable API**: Guild-themed naming creates distinctive, memorable interfaces that are easier to recall.
3. **Engaging Documentation**: The metaphor makes technical documentation more engaging and accessible.
4. **Framework Identity**: The guild theme gives the framework a unique identity in the crowded AI agent ecosystem.

## References

- [Guild Lore and Naming Conventions](lore.md)
- [Medieval Guild Structures](https://en.wikipedia.org/wiki/Guild)