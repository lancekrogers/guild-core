## Implement Memory System

Please help me implement the Memory system with these steps:

1. Review the Memory specification at specs/features/memory.md
2. Check the implementation guide at ai_docs/patterns/prompt_chain_memory.md
3. Consider the vector store integration at ai_docs/integration_guides/qdrant_vector_store.md

## Implementation Requirements

The Memory system should:

1. Store prompt chains in BoltDB
2. Maintain vector embeddings in Qdrant
3. Provide RAG for context restoration
4. Support efficient querying by task and agent

## Implementation Approach

1. First, let's define the Memory store interface
2. Then implement BoltDB and Qdrant backends
3. Create a RAG implementation for context retrieval
