# Prompt Chain Memory Patterns

This document explains how prompt chains are managed and stored in the Guild framework.

## Overview

Prompt chains are sequences of prompt-response pairs that represent the interaction history between an agent and an LLM. Guild stores and manages these chains to:

1. **Maintain context** across interactions
2. **Enable resumption** of interrupted tasks
3. **Analyze patterns** in LLM usage
4. **Optimize costs** by reusing results
5. **Provide traceability** for debugging

## Prompt Chain Structure

```go
// pkg/memory/interface.go
package memory

import (
 "context"
 "time"
)

// PromptChain represents a sequence of prompt-response pairs
type PromptChain struct {
 // ID is the unique identifier
 ID string `json:"id"`

 // TaskID is the associated task
 TaskID string `json:"task_id"`

 // AgentID is the agent that generated this chain
 AgentID string `json:"agent_id"`

 // Entries is the list of prompt-response pairs
 Entries []PromptEntry `json:"entries"`

 // CreatedAt is when the chain was created
 CreatedAt time.Time `json:"created_at"`

 // UpdatedAt is when the chain was last updated
 UpdatedAt time.Time `json:"updated_at"`

 // Tags are searchable labels
 Tags []string `json:"tags,omitempty"`
}

// PromptEntry represents a single prompt-response pair
type PromptEntry struct {
 // ID is the unique identifier
 ID string `json:"id"`

 // Prompt is the input to the LLM
 Prompt string `json:"prompt"`

 // Response is the output from the LLM
 Response string `json:"response"`

 // TokensUsed is the total tokens used
 TokensUsed int `json:"tokens_used"`

 // ToolsUsed is the list of tools used
 ToolsUsed []string `json:"tools_used,omitempty"`

 // Timestamp is when this entry was created
 Timestamp time.Time `json:"timestamp"`

 // Cost is the estimated cost of this interaction
 Cost float64 `json:"cost"`

 // Metadata contains additional information
 Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## Storage Interface

```go
// pkg/memory/interface.go
package memory

import (
 "context"
)

// Store defines the interface for memory storage
type Store interface {
 // SavePromptChain persists a prompt chain
 SavePromptChain(ctx context.Context, chain PromptChain) error

 // GetPromptChain retrieves a prompt chain by ID
 GetPromptChain(ctx context.Context, id string) (PromptChain, error)

 // GetPromptChainsByTask retrieves prompt chains by task ID
 GetPromptChainsByTask(ctx context.Context, taskID string) ([]PromptChain, error)

 // GetPromptChainsByAgent retrieves prompt chains by agent ID
 GetPromptChainsByAgent(ctx context.Context, agentID string) ([]PromptChain, error)

 // GetPromptChainsByTags retrieves prompt chains by tags
 GetPromptChainsByTags(ctx context.Context, tags []string) ([]PromptChain, error)

 // SearchPromptChains searches prompt chains by content
 SearchPromptChains(ctx context.Context, query string, limit int) ([]PromptChain, error)

 // Close closes the store
 Close() error
}
```

## Chain Management

Guild uses a `ChainManager` to handle prompt chain operations:

```go
// pkg/memory/chain_manager.go
package memory

import (
 "context"
 "fmt"
 "time"

 "github.com/google/uuid"
)

// ChainManager manages prompt chains
type ChainManager struct {
 store Store
}

// NewChainManager creates a new chain manager
func NewChainManager(store Store) *ChainManager {
 return &ChainManager{
  store: store,
 }
}

// CreateChain creates a new prompt chain
func (m *ChainManager) CreateChain(ctx context.Context, taskID, agentID string, tags []string) (PromptChain, error) {
 chain := PromptChain{
  ID:        uuid.New().String(),
  TaskID:    taskID,
  AgentID:   agentID,
  Entries:   []PromptEntry{},
  CreatedAt: time.Now(),
  UpdatedAt: time.Now(),
  Tags:      tags,
 }

 err := m.store.SavePromptChain(ctx, chain)
 if err != nil {
  return PromptChain{}, fmt.Errorf("failed to save chain: %w", err)
 }

 return chain, nil
}

// AddEntry adds an entry to a prompt chain
func (m *ChainManager) AddEntry(ctx context.Context, chainID string, prompt, response string,
 tokensUsed int, toolsUsed []string, cost float64, metadata map[string]interface{}) error {

 // Get existing chain
 chain, err := m.store.GetPromptChain(ctx, chainID)
 if err != nil {
  return fmt.Errorf("failed to get chain: %w", err)
 }

 // Create entry
 entry := PromptEntry{
  ID:         uuid.New().String(),
  Prompt:     prompt,
  Response:   response,
  TokensUsed: tokensUsed,
  ToolsUsed:  toolsUsed,
  Timestamp:  time.Now(),
  Cost:       cost,
  Metadata:   metadata,
 }

 // Add entry to chain
 chain.Entries = append(chain.Entries, entry)
 chain.UpdatedAt = time.Now()

 // Save updated chain
 err = m.store.SavePromptChain(ctx, chain)
 if err != nil {
  return fmt.Errorf("failed to save chain: %w", err)
 }

 return nil
}

// GetLastEntry gets the last entry from a chain
func (m *ChainManager) GetLastEntry(ctx context.Context, chainID string) (PromptEntry, error) {
 // Get chain
 chain, err := m.store.GetPromptChain(ctx, chainID)
 if err != nil {
  return PromptEntry{}, fmt.Errorf("failed to get chain: %w", err)
 }

 // Check for entries
 if len(chain.Entries) == 0 {
  return PromptEntry{}, fmt.Errorf("chain has no entries")
 }

 // Return last entry
 return chain.Entries[len(chain.Entries)-1], nil
}

// BuildContextFromChain builds context from a prompt chain
func (m *ChainManager) BuildContextFromChain(ctx context.Context, chainID string, maxEntries int) (string, error) {
 // Get chain
 chain, err := m.store.GetPromptChain(ctx, chainID)
 if err != nil {
  return "", fmt.Errorf("failed to get chain: %w", err)
 }

 // Check for entries
 if len(chain.Entries) == 0 {
  return "", nil
 }

 // Determine start index
 startIdx := 0
 if maxEntries > 0 && len(chain.Entries) > maxEntries {
  startIdx = len(chain.Entries) - maxEntries
 }

 // Build context
 var context strings.Builder
 context.WriteString("# Previous Interactions\n\n")

 for i := startIdx; i < len(chain.Entries); i++ {
  entry := chain.Entries[i]
  context.WriteString(fmt.Sprintf("## Prompt %d\n\n%s\n\n", i+1, entry.Prompt))
  context.WriteString(fmt.Sprintf("## Response %d\n\n%s\n\n", i+1, entry.Response))
 }

 return context.String(), nil
}
```

## Integration with Agents

Agents use the `ChainManager` to maintain context:

```go
// pkg/agent/basic_agent.go
package agent

import (
 "context"
 "fmt"
 "strings"

 "github.com/your-username/guild/pkg/kanban"
 "github.com/your-username/guild/pkg/memory"
 "github.com/your-username/guild/pkg/providers"
)

// BasicAgent implements the Agent interface
type BasicAgent struct {
 id        string
 provider  providers.Provider
 board     kanban.Board
 chainMgr  *memory.ChainManager
 chainByTask map[string]string // Maps taskID to chainID
}

// Execute runs a task and returns the result
func (a *BasicAgent) Execute(ctx context.Context, task kanban.Task) (kanban.Result, error) {
 // Get or create chain for this task
 chainID, err := a.getOrCreateChain(ctx, task.ID)
 if err != nil {
  return kanban.Result{}, err
 }

 // Build context from previous interactions
 context, err := a.chainMgr.BuildContextFromChain(ctx, chainID, 5)
 if err != nil {
  // Non-fatal error, proceed without context
  context = ""
 }

 // Build prompt
 prompt := a.buildPrompt(task, context)

 // Generate response
 resp, err := a.provider.Generate(ctx, providers.GenerateRequest{
  Model:       a.providerModel,
  Prompt:      prompt,
  MaxTokens:   2000,
  Temperature: 0.7,
  SystemPrompt: a.buildSystemPrompt(task),
 })
 if err != nil {
  return kanban.Result{}, fmt.Errorf("failed to generate response: %w", err)
 }

 // Save to chain
 err = a.chainMgr.AddEntry(ctx, chainID, prompt, resp.Text, resp.TokensUsed, nil,
  a.provider.Cost(providers.GenerateRequest{
   Model:    a.providerModel,
   Prompt:   prompt,
   MaxTokens: resp.TokensUsed,
  }), nil)
 if err != nil {
  // Non-fatal error, log but continue
  fmt.Printf("Warning: Failed to save prompt chain entry: %v\n", err)
 }

 // Parse response and create result
 result := kanban.Result{
  TaskID:      task.ID,
  Success:     true,
  Output:      resp.Text,
  TokensUsed:  resp.TokensUsed,
  CompletedAt: time.Now(),
 }

 return result, nil
}

// getOrCreateChain gets an existing chain or creates a new one
func (a *BasicAgent) getOrCreateChain(ctx context.Context, taskID string) (string, error) {
 // Check if we already have a chain for this task
 if chainID, ok := a.chainByTask[taskID]; ok {
  return chainID, nil
 }

 // Check if task has existing chains
 chains, err := a.chainMgr.GetChainsByTask(ctx, taskID)
 if err != nil {
  return "", fmt.Errorf("failed to get chains for task: %w", err)
 }

 // Use most recent chain if available
 if len(chains) > 0 {
  chainID := chains[len(chains)-1].ID
  a.chainByTask[taskID] = chainID
  return chainID, nil
 }

 // Create new chain
 chain, err := a.chainMgr.CreateChain(ctx, taskID, a.id, []string{task.Status})
 if err != nil {
  return "", fmt.Errorf("failed to create chain: %w", err)
 }

 // Store chain ID for future reference
 a.chainByTask[taskID] = chain.ID
 return chain.ID, nil
}

// buildPrompt constructs a prompt for the LLM
func (a *BasicAgent) buildPrompt(task kanban.Task, context string) string {
 var prompt strings.Builder

 // Add task details
 prompt.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Title))
 prompt.WriteString(fmt.Sprintf("## Description\n\n%s\n\n", task.Description))

 // Add context if available
 if context != "" {
  prompt.WriteString(context)
 }

 // Add instruction
 prompt.WriteString("\n## Instructions\n\n")
 prompt.WriteString("Please complete the task described above.\n")

 return prompt.String()
}

// buildSystemPrompt constructs a system prompt for the LLM
func (a *BasicAgent) buildSystemPrompt(task kanban.Task) string {
 return fmt.Sprintf("You are an AI agent named %s. Your role is to complete tasks efficiently and accurately.", a.id)
}
```

## Retrieval-Augmented Generation (RAG)

Guild uses Retrieval-Augmented Generation to enhance prompts with relevant context:

```go
// pkg/memory/rag/retriever.go
package rag

import (
 "context"
 "fmt"
 "sort"
 "strings"

 "github.com/your-username/guild/pkg/memory"
 "github.com/your-username/guild/pkg/memory/vector"
)

// Retriever provides retrieval-augmented generation
type Retriever struct {
 chainMgr    *memory.ChainManager
 vectorStore vector.VectorStore
}

// NewRetriever creates a new RAG retriever
func NewRetriever(chainMgr *memory.ChainManager, vectorStore vector.VectorStore) *Retriever {
 return &Retriever{
  chainMgr:    chainMgr,
  vectorStore: vectorStore,
 }
}

// EnhancePrompt augments a prompt with relevant context
func (r *Retriever) EnhancePrompt(ctx context.Context, prompt, query string, taskID, agentID string) (string, error) {
 // Build context from multiple sources
 context, err := r.buildContext(ctx, query, taskID, agentID)
 if err != nil {
  return prompt, fmt.Errorf("failed to build context: %w", err)
 }

 // If no context was found, return the original prompt
 if context == "" {
  return prompt, nil
 }

 // Combine context with prompt
 return fmt.Sprintf("# Relevant Context\n\n%s\n\n# Original Prompt\n\n%s", context, prompt), nil
}

// buildContext retrieves relevant context from multiple sources
func (r *Retriever) buildContext(ctx context.Context, query, taskID, agentID string) (string, error) {
 var contexts []string

 // 1. Get task-specific prompt chain context
 if taskID != "" {
  chains, err := r.chainMgr.GetChainsByTask(ctx, taskID)
  if err == nil && len(chains) > 0 {
   // Get most recent chain
   chain := chains[len(chains)-1]

   // Build context from last 3 entries at most
   var chainContext strings.Builder
   chainContext.WriteString("## Previous Task Interactions\n\n")

   startIdx := 0
   if len(chain.Entries) > 3 {
    startIdx = len(chain.Entries) - 3
   }

   for i := startIdx; i < len(chain.Entries); i++ {
    entry := chain.Entries[i]
    chainContext.WriteString(fmt.Sprintf("### Exchange %d\n\n", i-startIdx+1))
    chainContext.WriteString(fmt.Sprintf("**Prompt**: %s\n\n", truncateText(entry.Prompt, 200)))
    chainContext.WriteString(fmt.Sprintf("**Response**: %s\n\n", truncateText(entry.Response, 300)))
   }

   contexts = append(contexts, chainContext.String())
  }
 }

 // 2. Get similar context from vector store
 if r.vectorStore != nil && query != "" {
  matches, err := r.vectorStore.QueryEmbeddings(ctx, query, 3)
  if err == nil && len(matches) > 0 {
   var vectorContext strings.Builder
   vectorContext.WriteString("## Related Information\n\n")

   for i, match := range matches {
    if match.Score < 0.7 {
     // Skip matches with low relevance
     continue
    }

    vectorContext.WriteString(fmt.Sprintf("### Source %d (%.2f relevance)\n\n", i+1, match.Score))
    vectorContext.WriteString(truncateText(match.Text, 500))
    vectorContext.WriteString("\n\n")
   }

   if vectorContext.Len() > 0 {
    contexts = append(contexts, vectorContext.String())
   }
  }
 }

 // 3. Get agent-specific context
 if agentID != "" && taskID != "" {
  // Get other recent chains for this agent (but not this task)
  chains, err := r.chainMgr.GetChainsByAgent(ctx, agentID)
  if err == nil && len(chains) > 0 {
   // Filter out chains for the current task
   var otherChains []memory.PromptChain
   for _, chain := range chains {
    if chain.TaskID != taskID {
     otherChains = append(otherChains, chain)
    }
   }

   // Sort by recency
   sort.Slice(otherChains, func(i, j int) bool {
    return otherChains[i].UpdatedAt.After(otherChains[j].UpdatedAt)
   })

   // Get the most recent chain
   if len(otherChains) > 0 {
    chain := otherChains[0]

    // Get the last entry
    if len(chain.Entries) > 0 {
     lastEntry := chain.Entries[len(chain.Entries)-1]

     // Add as context
     var agentContext strings.Builder
     agentContext.WriteString("## Recent Agent Activity\n\n")
     agentContext.WriteString(fmt.Sprintf("Task: %s\n\n", chain.TaskID))
     agentContext.WriteString(fmt.Sprintf("Last Response: %s\n\n", truncateText(lastEntry.Response, 300)))

     contexts = append(contexts, agentContext.String())
    }
   }
  }
 }

 // Combine all contexts
 return strings.Join(contexts, "\n\n"), nil
}

// truncateText shortens text to the specified length
func truncateText(text string, maxLen int) string {
 if len(text) <= maxLen {
  return text
 }
 return text[:maxLen-3] + "..."
}
```

## BoltDB Implementation

Guild uses BoltDB to persist prompt chains:

```go
// pkg/memory/boltdb/store.go
package boltdb

import (
 "context"
 "encoding/json"
 "fmt"
 "time"

 "github.com/boltdb/bolt"
 "github.com/your-username/guild/pkg/memory"
)

var (
 promptChainBucket = []byte("prompt_chains")
 tasksBucket      = []byte("tasks")
 agentsBucket     = []byte("agents")
 tagsBucket       = []byte("tags")
)

// Store implements memory.Store using BoltDB
type Store struct {
 db *bolt.DB
}

// NewStore creates a new BoltDB store
func NewStore(path string) (*Store, error) {
 // Open database
 db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
 if err != nil {
  return nil, fmt.Errorf("failed to open database: %w", err)
 }

 // Create buckets
 err = db.Update(func(tx *bolt.Tx) error {
  buckets := [][]byte{promptChainBucket, tasksBucket, agentsBucket, tagsBucket}

  for _, bucket := range buckets {
   _, err := tx.CreateBucketIfNotExists(bucket)
   if err != nil {
    return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
   }
  }

  return nil
 })
 if err != nil {
  db.Close()
  return nil, err
 }

 return &Store{db: db}, nil
}

// SavePromptChain persists a prompt chain
func (s *Store) SavePromptChain(ctx context.Context, chain memory.PromptChain) error {
 return s.db.Update(func(tx *bolt.Tx) error {
  // Marshal chain to JSON
  data, err := json.Marshal(chain)
  if err != nil {
   return fmt.Errorf("failed to marshal chain: %w", err)
  }

  // Save chain
  b := tx.Bucket(promptChainBucket)
  err = b.Put([]byte(chain.ID), data)
  if err != nil {
   return err
  }

  // Index by task
  if chain.TaskID != "" {
   tb := tx.Bucket(tasksBucket)
   key := []byte(fmt.Sprintf("%s:%s", chain.TaskID, chain.ID))
   err = tb.Put(key, []byte(chain.ID))
   if err != nil {
    return err
   }
  }

  // Index by agent
  if chain.AgentID != "" {
   ab := tx.Bucket(agentsBucket)
   key := []byte(fmt.Sprintf("%s:%s", chain.AgentID, chain.ID))
   err = ab.Put(key, []byte(chain.ID))
   if err != nil {
    return err
   }
  }

  // Index by tags
  if len(chain.Tags) > 0 {
   tb := tx.Bucket(tagsBucket)
   for _, tag := range chain.Tags {
    key := []byte(fmt.Sprintf("%s:%s", tag, chain.ID))
    err = tb.Put(key, []byte(chain.ID))
    if err != nil {
     return err
    }
   }
  }

  return nil
 })
}

// GetPromptChain retrieves a prompt chain by ID
func (s *Store) GetPromptChain(ctx context.Context, id string) (memory.PromptChain, error) {
 var chain memory.PromptChain

 err := s.db.View(func(tx *bolt.Tx) error {
  b := tx.Bucket(promptChainBucket)
  data := b.Get([]byte(id))
  if data == nil {
   return fmt.Errorf("prompt chain not found: %s", id)
  }

  return json.Unmarshal(data, &chain)
 })

 return chain, err
}

// GetPromptChainsByTask retrieves prompt chains by task ID
func (s *Store) GetPromptChainsByTask(ctx context.Context, taskID string) ([]memory.PromptChain, error) {
 var chains []memory.PromptChain

 err := s.db.View(func(tx *bolt.Tx) error {
  // Get task bucket
  tb := tx.Bucket(tasksBucket)
  if tb == nil {
   return nil
  }

  // Get prompt chains bucket
  cb := tx.Bucket(promptChainBucket)
  if cb == nil {
   return nil
  }

  // Get all chains for this task
  prefix := []byte(fmt.Sprintf("%s:", taskID))
  cursor := tb.Cursor()

  for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
   // Get chain data
   data := cb.Get(v)
   if data == nil {
    continue
   }

   // Unmarshal chain
   var chain memory.PromptChain
   if err := json.Unmarshal(data, &chain); err != nil {
    return err
   }

   chains = append(chains, chain)
  }

  return nil
 })

 return chains, err
}

// Additional methods for GetPromptChainsByAgent, GetPromptChainsByTags, SearchPromptChains...

// Close closes the database
func (s *Store) Close() error {
 return s.db.Close()
}
```

## Usage Examples

### Creating and Using Chains

````go
// Create memory store
store, err := boltdb.NewStore("memory.db")
if err != nil {
 log.Fatalf("Failed to create store: %v", err)
}

// Create chain manager
chainMgr := memory.NewChainManager(store)

// Create a chain
ctx := context.Background()
chain, err := chainMgr.CreateChain(ctx, "task-123", "agent-456", []string{"code-generation"})
if err != nil {
 log.Fatalf("Failed to create chain: %v", err)
}

// Add an entry
err = chainMgr.AddEntry(ctx, chain.ID,
 "Generate a function to calculate Fibonacci numbers",
 "```go\nfunc fibonacci(n int) int {\n\tif n <= 1 {\n\t\treturn n\n\t}\n\treturn fibonacci(n-1) + fibonacci(n-2)\n}\n```",
 120, // tokens used
 nil, // tools used
 0.001, // cost
 map[string]interface{}{
  "language": "go",
 },
)
if err != nil {
 log.Printf("Failed to add entry: %v", err)
}

// Get context from chain
context, err := chainMgr.BuildContextFromChain(ctx, chain.ID, 5)
if err != nil {
 log.Printf("Failed to build context: %v", err)
}
````

### Using RAG for Enhanced Prompts

```go
// Create vector store
vectorStore, err := vector.NewQdrantStore(vector.Config{
 Address:    "localhost:6334",
 Collection: "guild_embeddings",
 VectorSize: 1536,
 Embedder:   openAIEmbedder,
})
if err != nil {
 log.Fatalf("Failed to create vector store: %v", err)
}

// Create RAG retriever
retriever := rag.NewRetriever(chainMgr, vectorStore)

// Enhance prompt with relevant context
enhancedPrompt, err := retriever.EnhancePrompt(ctx,
 "Implement a sorting algorithm",
 "sorting algorithm implementation",
 "task-123",
 "agent-456",
)
if err != nil {
 log.Printf("Failed to enhance prompt: %v", err)
}

// Use enhanced prompt with LLM
response, err := provider.Generate(ctx, providers.GenerateRequest{
 Model:  "gpt-4",
 Prompt: enhancedPrompt,
})
```

## Meta-Coordination Protocol (MCP)

The MCP uses prompt chains to optimize agent behavior:

```go
// pkg/mcp/optimizer.go
package mcp

import (
 "context"

 "github.com/your-username/guild/pkg/memory"
)

// Optimizer detects patterns in prompt chains
type Optimizer struct {
 chainMgr *memory.ChainManager
}

// NewOptimizer creates a new optimizer
func NewOptimizer(chainMgr *memory.ChainManager) *Optimizer {
 return &Optimizer{
  chainMgr: chainMgr,
 }
}

// DetectPatterns identifies repeated prompts and tool usage
func (o *Optimizer) DetectPatterns(ctx context.Context) ([]Pattern, error) {
 // Implementation details...
}

// RecommendTools suggests tools for common operations
func (o *Optimizer) RecommendTools(ctx context.Context) ([]ToolRecommendation, error) {
 // Implementation details...
}

// AnalyzeCosts calculates costs for different LLM operations
func (o *Optimizer) AnalyzeCosts(ctx context.Context) ([]CostAnalysis, error) {
 // Implementation details...
}
```

## Best Practices

1. **Chain Management**

   - Create one chain per task-agent combination
   - Store context from previous interactions
   - Use the most recent chain entries for context

2. **Storage Optimization**

   - Compress or archive old chains
   - Implement TTL for older entries
   - Consider storing large responses externally

3. **Retrieval Strategies**

   - Combine task-specific, semantic, and agent history
   - Use recency and relevance for ranking
   - Filter low-relevance matches

4. **Context Construction**

   - Truncate long entries to avoid token limits
   - Structure context with clear sections
   - Prioritize most relevant information

5. **Pattern Recognition**
   - Track common prompt patterns
   - Identify repetitive tasks
   - Measure token usage and costs

## Related Documentation

- [../integration_guides/qdrant_vector_store.md](../integration_guides/qdrant_vector_store.md)
- [../integration_guides/bolt_db_kanban.md](../integration_guides/bolt_db_kanban.md)
- [../architecture/coordination.md](../architecture/coordination.md)
