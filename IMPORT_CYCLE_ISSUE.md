# Import Cycle Issue - RESOLVED

## Problem (FIXED)
There was a circular import dependency:
- `pkg/memory/rag` imports `pkg/agent` (in rag_agent.go)
- `pkg/agent` imports `pkg/memory/rag` (in corpus_agent.go)

## Cause
The CorpusAgent was placed in the agent package but it depends on the RAG system, while the RAG system has an AgentWrapper that depends on the agent package.

## Solution Implemented
Moved CorpusAgent to `pkg/corpus/agent` package. This follows the correct architecture where:
- `pkg/agent` - Contains core agent framework, interfaces, and factories
- `pkg/corpus/agent` - Contains the specific corpus agent implementation
- Other agent implementations should follow this pattern (e.g., `pkg/research/agent`)

## Result
The circular dependency is now resolved. The dependency graph is:
```
pkg/corpus/agent → pkg/memory/rag → pkg/agent (no cycles!)
```

## Files Changed
1. Moved `pkg/agent/corpus_agent.go` → `pkg/corpus/agent/corpus_agent.go`
2. Updated imports in `cmd/guild/corpus_query.go` to use the new package location