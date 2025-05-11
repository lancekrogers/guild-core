# Retrieval-Augmented Generation with Chromem-go

## Overview

Guild's Retrieval-Augmented Generation (RAG) system enhances agent capabilities by retrieving relevant context from stored knowledge before generating responses. This system significantly reduces hallucinations, provides specific knowledge, and improves the quality of agent interactions.

## Architecture

The RAG system consists of these key components:

1. **Vector Store**: Chromem-go embedded database for storing and retrieving embeddings
2. **Document Chunking**: Processes large documents into appropriate segments for embedding
3. **Embedding Generation**: Converts text to vector embeddings using provider APIs
4. **Retrieval Logic**: Finds the most relevant content for a given query
5. **Context Integration**: Inserts retrieved context into prompts
6. **Agent Enhancement**: Wraps agents with RAG capabilities

## Vector Store Selection: Chromem-go vs. Qdrant

After careful analysis, we've selected **Chromem-go** as our vector database. Here's the comparison:

### Chromem-go

**Pros**:
- Pure Go implementation with zero dependencies
- Embeddable directly in your application (like SQLite vs PostgreSQL)
- Simple API inspired by Chroma (4 core commands)
- No separate service to maintain
- Native Go interface (no CGO or REST API calls)
- Good performance for small to medium datasets
- Aligns with Guild's dependency reduction philosophy

**Cons**:
- Still in beta, "under heavy construction"
- May introduce breaking changes before v1.0.0
- Limited to in-memory operation with optional persistence
- Not designed for massive scale (millions of documents)
- Fewer features than established vector DBs
- Less battle-tested than alternatives
- No distributed deployment options

### Qdrant

**Pros**:
- Production-ready, mature solution
- Built from ground up in Rust for performance
- Excellent benchmarks showing high RPS and low latency
- Supports horizontal scaling for large datasets
- Robust filtering capabilities
- Well-documented with strong community support
- Has official Go client library
- Supports complex payload filtering

**Cons**:
- Separate service to maintain/deploy
- Not embedded in your application code
- Requires REST API calls from Go code
- More complex setup and management

## Decision Rationale

Chromem-go was selected because:

1. **Simplicity**: Aligns with Guild's philosophy of simplifying setup and reducing maintenance overhead
2. **Native Go Integration**: Provides seamless integration without REST calls or service deployment
3. **Dependency Reduction**: Follows our recent work on reducing external dependencies
4. **Sufficient Scale**: Handles Guild's typical usage patterns effectively
5. **Embeddability**: Enables single-binary distribution without complex setup
6. **Developer Experience**: Simplifies development workflow with no separate service

Guild's typical use cases involve moderate-sized knowledge bases that fit well within Chromem-go's performance envelope. The simplicity benefits outweigh the maturity concerns for our current needs.

## Data Flow

1. **Ingestion**:
   - Documents from the Corpus System are chunked
   - Each chunk is embedded and stored in Chromem-go
   - Metadata and source information are preserved

2. **Retrieval**:
   - Agent query is embedded using the same model
   - Vector similarity search finds relevant chunks
   - Results are ranked by relevance score
   - Top N chunks are formatted as context

3. **Generation**:
   - Retrieved context is prepended to the prompt
   - Enhanced prompt is sent to the LLM for completion
   - Response maintains knowledge consistency

## Usage Examples

```go
// Initialize RAG system
vectorStore := memory.NewChromemStore(config)
retriever := rag.NewRetriever(vectorStore, corpusConfig)

// Enhance a prompt with context
enhancedPrompt, _ := retriever.EnhancePrompt(ctx, 
    "What tools can agents use?", 
    "agent tools capabilities", 
    rag.DefaultRetrievalConfig())

// Create RAG-enhanced agent
ragAgent := retriever.EnhanceAgent(originalAgent)
```

## Performance Considerations

- Embedding generation is typically the bottleneck (API call to provider)
- Chromem-go query performance: ~0.3ms for 1,000 documents, ~40ms for 100,000 documents
- Embed documents in batches during ingestion
- Consider caching common queries
- Monitor memory usage with large document collections

## Configuration

RAG system is configurable through:

- Vector store type (Chromem-go by default, with Qdrant as fallback)
- Embedding model selection (OpenAI by default)
- Retrieval parameters (minimum relevance score, maximum results)
- Chunking strategy and size
- Persistence options for Chromem-go

## Scalability Considerations

While Chromem-go meets our current needs, we maintain the option to switch to Qdrant for use cases requiring:

- Very large document collections (millions of vectors)
- Distributed deployment across multiple servers
- Advanced filtering capabilities beyond simple metadata matching

The interface-based design allows for swapping implementations as requirements evolve.

## Future Enhancements

1. **Hybrid Search**: Combine vector similarity with keyword-based search
2. **Query Rewriting**: Improve retrieval by transforming user queries
3. **Multi-stage Retrieval**: Use iterative searches with context refinement
4. **Feedback Integration**: Learn from agent usage patterns
5. **Cross-document Reasoning**: Connect insights across multiple documents

## Related Documents

- [RAG Implementation Guide](../../ai_docs/implement_rag_with_chromemgo.md)
- [Corpus System Specification](./corpus_system.md)
- [Agent Behavior Specification](./agent-behavior.md)