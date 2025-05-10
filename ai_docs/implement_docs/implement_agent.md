## Implement Agent Component

Please help me implement the Agent component with these steps:

1. First, review the Agent specification at specs/features/agent-behavior.md
2. Then, check the implementation guide at ai_docs/architecture/agent_lifecycle.md
3. Follow the interface-first pattern from ai_docs/patterns/interface_first.md

## Implementation Requirements

The Agent component should:

1. Implement the Provider interface for LLM interactions
2. Support tools through a standardized interface
3. Maintain a personal Kanban board
4. Use cost-aware decision making
5. Execute tasks through prompt chains

## Implementation Approach

1. First, let's define the Agent interface
2. Then implement a BasicAgent that satisfies this interface
3. Add tests to verify the implementation
