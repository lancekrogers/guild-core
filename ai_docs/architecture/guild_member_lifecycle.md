# Guild Member Lifecycle

This document explains the lifecycle of artisans in the Guild system, from initiation to departure.

## Initiation and Commissioning

1. **Charter Loading**

   - Guild loads the artisan's charter from the YAML configuration
   - The GuildMaster summons the artisan using the member factory
   - Privileges and capabilities are assigned based on rank and specialization

2. **Provider Initiation**

   - The appropriate LLM provider is authorized and connected
   - Guild seals (API keys) are validated or local model availability is confirmed
   - Cost tracking ledgers are established

3. **Tool Provision**
   - Guild tools are furnished to the artisan's workbench
   - Tool privileges are granted according to artisan rank
   - Cost allocations for each tool are recorded in the ledger

## Crafting Phase

1. **Task Commission**

   - Work is assigned to the artisan via the Guild board
   - Artisan is summoned via the guild hall's messaging system
   - Cost budgets for the commission are established

2. **Context Gathering**

   - Artisan retrieves relevant tradecraft knowledge using the guild's archives (RAG)
   - Previous work on this commission is studied from memory chains
   - Guild charter details are incorporated

3. **Instruction Formulation**

   - Guild traditions (system prompt) are consulted to understand proper approach
   - Commission details are carefully examined
   - Gathered knowledge is arranged on the workbench
   - Available guild tools are inventoried

4. **Craft Execution**

   - Instructions are sent to the LLM provider with appropriate cost considerations
   - Response is examined and techniques are applied
   - Guild tools are wielded as needed
   - Progress is documented in the memory chain
   - Cost records are maintained for all resources used

5. **Commission Updates**
   - Task status is recorded in the Guild's ledger (Kanban)
   - Progress notifications are sent to the GuildMaster and patrons

## Apprenticeship and Mastery

1. **Knowledge Preservation**

   - Current crafting state is entrusted to the guild archives (BoltDB)
   - Instruction chains are preserved for future reference
   - Cost records are maintained for guild accounting

2. **Skill Advancement**
   - Task experiences enrich the artisan's capabilities
   - Context knowledge grows through each commission
   - Efficiency improves as cost-awareness develops

## Retirement

1. **Commission Completion**

   - Final craftsmanship is presented and recorded
   - Commission is marked as Complete in the Guild's ledger
   - Cost accounting is finalized and reported

2. **Errorcraft Handling**

   - Crafting issues are documented with full context
   - Commission may be marked as Blocked or returned to the planning stage
   - Cost anomalies are analyzed for future improvement

3. **Workshop Closure**
   - Provider connections are respectfully closed
   - Resources are returned to the guild stores
   - Final accounting is added to the guild's archives

## Implementation Guidelines

```go
// Example artisan commissioning
func NewGuildMember(config ArtisanCharter, providers ProviderRegistry, tools ToolRegistry) (GuildArtisan, error) {
    // Implementation details...
}

// Example commission execution loop
func (a *Craftsman) CraftSolution(ctx context.Context, commission Commission) (Craftsmanship, error) {
    // Implementation details...
}
```

## Related Tradecraft Documentation

- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
- [../integration_guides/agent_task_events.md](../integration_guides/agent_task_events.md)