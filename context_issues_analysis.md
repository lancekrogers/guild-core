# Context Propagation Issues Analysis

## Summary
After reviewing the provided files, I found several instances where context is not properly propagated. The main issues are:

1. **Context.Background() usage instead of passed context**
2. **Missing context propagation to child functions**
3. **Corpus package functions don't accept context parameters**

## Detailed Findings

### 1. `/pkg/memory/vector/universal_embedder.go`
- **No issues found** - All context usage is properly propagated

### 2. `/pkg/memory/vector/chromem.go`
- **Issue on line 293**: In `batchEmbed()`, when creating individual embeddings in the fallback loop (line 284), it calls `e.Embed(ctx, text)` which is correct, but the embedder itself needs context propagation through the chromem embedding function.

### 3. `/pkg/memory/rag/retriever.go`
- **Issue on line 299**: In `searchCorpus()`, the function calls `corpus.List(*r.corpusConfig)` and `corpus.Load(docPath)` without passing the context. These corpus functions should accept context as their first parameter.

### 4. `/cmd/guild/corpus_scan.go`
- **Issue on line 88**: Uses `context.Background()` in `initializeRAGSystem()` when creating vector store, but should propagate the context from the caller
- **Issue on line 143**: `vector.NewVectorStore(context.Background(), vectorConfig)` - should use the passed context
- **Issue on line 167**: `corpus.List(cfg)` - should be `corpus.List(ctx, cfg)`
- **Issue on line 210**: `corpus.Load(doc.FilePath)` - should be `corpus.Load(ctx, doc.FilePath)`
- **Issue on lines 223, 239, 268, 270**: Various corpus operations don't propagate context

### 5. `/pkg/agent/corpus_agent.go`
- **Issue on line 138**: `corpus.Save(doc, a.corpusConfig)` - should be `corpus.Save(ctx, doc, a.corpusConfig)`

## Recommendations

### 1. Update Corpus Package API
The corpus package functions need to be updated to accept context as their first parameter:
```go
// Current
func List(config Config) ([]*CorpusDoc, error)
func Load(path string) (*CorpusDoc, error)
func Save(doc *CorpusDoc, config Config) error

// Should be
func List(ctx context.Context, config Config) ([]*CorpusDoc, error)
func Load(ctx context.Context, path string) (*CorpusDoc, error)
func Save(ctx context.Context, doc *CorpusDoc, config Config) error
```

### 2. Fix Context.Background() Usage
Replace `context.Background()` with properly propagated context in:
- `corpus_scan.go` line 143

### 3. Propagate Context in Retriever
Update the retriever's `searchCorpus` method to pass context to corpus operations.

### 4. Consider Context in Embedder Functions
The chromem store's embedding function wrapper should ensure context is properly used when creating embeddings.

## Priority
1. **High**: Fix corpus package API to accept context
2. **High**: Fix context.Background() usage in corpus_scan.go
3. **Medium**: Update all callers of corpus functions to pass context
4. **Low**: Review other packages for similar issues

These changes will ensure proper context propagation throughout the codebase, enabling:
- Proper cancellation handling
- Request-scoped values (like tracing)
- Timeout management
- Graceful shutdown