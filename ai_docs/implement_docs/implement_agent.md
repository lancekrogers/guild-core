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
6. Utilize RAG capabilities with Chromem-go for knowledge retrieval

## Implementation Approach

1. First, let's define the Agent interface
2. Then implement a BasicAgent that satisfies this interface
3. Create a RAGAgent wrapper that enhances the BasicAgent with retrieval capabilities
4. Integrate with the Chromem-go vector store for knowledge retrieval
5. Add comprehensive tests to verify the implementation

## RAG Integration

For retrieval-augmented generation capabilities:

1. Review the RAG implementation guide at ai_docs/implement_rag_with_chromemgo.md
2. Use the Chromem-go vector store for document embeddings
3. Implement the Agent wrapper pattern to enhance prompts with relevant context
4. Connect the Agent to the Corpus system for knowledge retrieval
5. Ensure proper configuration through the vector store factory
