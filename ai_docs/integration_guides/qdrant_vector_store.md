# Qdrant Vector Store Integration

This document explains how to integrate with Qdrant for vector storage and similarity search in Guild.

## Overview

Qdrant is a vector similarity search engine that Guild uses for storing embeddings of prompts, responses, and objectives. It enables semantic search and context retrieval for RAG (Retrieval-Augmented Generation).

## Key Concepts

1. **Collections** - Named sets of vectors (e.g., `prompt_embeddings`, `objective_embeddings`)
2. **Points** - Individual vector entries with payloads
3. **Vectors** - The embedding vectors themselves
4. **Payloads** - Metadata associated with vectors (e.g., text content, timestamps)
5. **Searches** - Similarity queries against collections

## Installation

### Docker

The simplest way to run Qdrant:

```bash
docker run -p 6333:6333 -p 6334:6334 \
    -v $(pwd)/qdrant_data:/qdrant/storage \
    qdrant/qdrant
```

### Go Client

```bash
go get github.com/qdrant/go-client
```

## Implementation

```go
// pkg/memory/vector/qdrant.go
package vector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/your-username/guild/pkg/memory"
)

// QdrantStore implements the VectorStore interface using Qdrant
type QdrantStore struct {
	client        qdrant.QdrantClient
	collection    string
	vectorSize    uint64
	embedder      Embedder
}

// Embedder generates embeddings from text
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Config contains Qdrant configuration
type Config struct {
	Address      string
	Collection   string
	VectorSize   uint64
	Embedder     Embedder
	UseSSL       bool
}

// NewQdrantStore creates a new Qdrant store
func NewQdrantStore(config Config) (*QdrantStore, error) {
	// Set defaults
	if config.Collection == "" {
		config.Collection = "guild_embeddings"
	}
	if config.VectorSize == 0 {
		config.VectorSize = 1536 // Default for OpenAI embeddings
	}
	if config.Address == "" {
		config.Address = "localhost:6334"
	}

	// Create gRPC connection
	opts := []grpc.DialOption{
		grpc.WithBlock(),
	}
	if !config.UseSSL {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	// Create client
	client := qdrant.NewQdrantClient(conn)

	// Ensure collection exists
	err = createCollectionIfNotExists(context.Background(), client, config.Collection, config.VectorSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &QdrantStore{
		client:        client,
		collection:    config.Collection,
		vectorSize:    config.VectorSize,
		embedder:      config.Embedder,
	}, nil
}

// createCollectionIfNotExists creates a collection if it doesn't exist
func createCollectionIfNotExists(ctx context.Context, client qdrant.QdrantClient, name string, size uint64) error {
	// Check if collection exists
	collections, err := client.ListCollections(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return err
	}

	for _, collection := range collections.Collections {
		if collection.Name == name {
			return nil
		}
	}

	// Create collection
	_, err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: &qdrant.VectorsConfig{
			Params: &qdrant.VectorsParams{
				Size:     size,
				Distance: qdrant.Distance_Cosine,
			},
		},
	})

	return err
}

// SaveEmbedding stores a vector embedding
func (s *QdrantStore) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Generate vector if not provided
	vector := embedding.Vector
	if len(vector) == 0 && s.embedder != nil {
		var err error
		vector, err = s.embedder.Embed(ctx, embedding.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Create payload
	payload := make(map[string]interface{})
	payload["text"] = embedding.Text
	payload["source"] = embedding.Source
	payload["timestamp"] = embedding.Timestamp.Format(time.RFC3339)

	for k, v := range embedding.Metadata {
		payload[k] = v
	}

	// Convert to Qdrant payload
	qdrantPayload := make(map[string]*qdrant.Value)
	for k, v := range payload {
		switch val := v.(type) {
		case string:
			qdrantPayload[k] = &qdrant.Value{
				Kind: &qdrant.Value_StringValue{
					StringValue: val,
				},
			}
		case int:
			qdrantPayload[k] = &qdrant.Value{
				Kind: &qdrant.Value_IntegerValue{
					IntegerValue: int64(val),
				},
			}
		case float64:
			qdrantPayload[k] = &qdrant.Value{
				Kind: &qdrant.Value_DoubleValue{
					DoubleValue: val,
				},
			}
		case bool:
			qdrantPayload[k] = &qdrant.Value{
				Kind: &qdrant.Value_BoolValue{
					BoolValue: val,
				},
			}
		}
	}

	// Create point ID
	pointId := &qdrant.PointId{
		PointIdOptions: &qdrant.PointId_Uuid{
			Uuid: embedding.ID,
		},
	}

	// Create vector
	vectorData := &qdrant.Vector{
		Data: vector,
	}

	// Upsert the point
	_, err := s.client.UpsertPoints(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points: []*qdrant.PointStruct{
			{
				Id:      pointId,
				Vector:  vectorData,
				Payload: qdrantPayload,
			},
		},
	})

	return err
}

// QueryEmbeddings performs a similarity search
func (s *QdrantStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	// Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Search
	response, err := s.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: s.collection,
		Vector:         vector,
		Limit:          uint64(limit),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// Convert results
	matches := make([]memory.EmbeddingMatch, 0, len(response.Result))
	for _, point := range response.Result {
		match := memory.EmbeddingMatch{
			ID:       point.Id.GetUuid(),
			Score:    point.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract fields from payload
		if text, ok := point.Payload["text"]; ok {
			match.Text = text.GetStringValue()
		}
		if source, ok := point.Payload["source"]; ok {
			match.Source = source.GetStringValue()
		}
		if timestamp, ok := point.Payload["timestamp"]; ok {
			ts, err := time.Parse(time.RFC3339, timestamp.GetStringValue())
			if err == nil {
				match.Timestamp = ts
			}
		}

		// Extract other metadata
		for k, v := range point.Payload {
			if k == "text" || k == "source" || k == "timestamp" {
				continue
			}

			switch {
			case v.GetStringValue() != "":
				match.Metadata[k] = v.GetStringValue()
			case v.GetIntegerValue() != 0:
				match.Metadata[k] = v.GetIntegerValue()
			case v.GetDoubleValue() != 0:
				match.Metadata[k] = v.GetDoubleValue()
			case v.GetBoolValue():
				match.Metadata[k] = v.GetBoolValue()
			}
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// DeleteEmbedding removes an embedding
func (s *QdrantStore) DeleteEmbedding(ctx context.Context, id string) error {
	_, err := s.client.DeletePoints(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{
							PointIdOptions: &qdrant.PointId_Uuid{
								Uuid: id,
							},
						},
					},
				},
			},
		},
	})

	return err
}

// Close closes the connection
func (s *QdrantStore) Close() error {
	// No explicit close for the client
	return nil
}
```

## OpenAI Embedder Implementation

```go
// pkg/memory/vector/openai_embedder.go
package vector

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(apiKey string, model string) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if model == "" {
		model = openai.AdaEmbeddingV2
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbedder{
		client: client,
		model:  model,
	}, nil
}

// Embed generates an embedding from text
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	response, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return response.Data[0].Embedding, nil
}
```

## Usage Examples

### Initialization

```go
// Create embedder
embedder, err := vector.NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"), "")
if err != nil {
	log.Fatalf("Failed to create embedder: %v", err)
}

// Create vector store
config := vector.Config{
	Address:    "localhost:6334",
	Collection: "guild_embeddings",
	VectorSize: 1536,
	Embedder:   embedder,
}

store, err := vector.NewQdrantStore(config)
if err != nil {
	log.Fatalf("Failed to create vector store: %v", err)
}
```

### Storing Embeddings

```go
// Create a new embedding
embedding := memory.Embedding{
	Text:   "Guild is a framework for orchestrating AI agents",
	Source: "documentation",
	Metadata: map[string]interface{}{
		"type": "definition",
		"tags": []string{"framework", "agents"},
	},
	Timestamp: time.Now(),
}

// Save embedding
ctx := context.Background()
err := store.SaveEmbedding(ctx, embedding)
if err != nil {
	log.Printf("Failed to save embedding: %v", err)
}
```

### Searching for Similar Content

```go
// Perform a similarity search
ctx := context.Background()
query := "How do agents collaborate?"
matches, err := store.QueryEmbeddings(ctx, query, 5)
if err != nil {
	log.Printf("Failed to query embeddings: %v", err)
}

// Process results
for i, match := range matches {
	fmt.Printf("%d. %s (score: %.4f)\n", i+1, match.Text, match.Score)
	fmt.Printf("   Source: %s\n", match.Source)
}
```

## RAG Implementation

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
	vectorStore *vector.QdrantStore
}

// NewRetriever creates a new RAG retriever
func NewRetriever(vectorStore *vector.QdrantStore) *Retriever {
	return &Retriever{
		vectorStore: vectorStore,
	}
}

// RetrieveContext gets relevant context for a query
func (r *Retriever) RetrieveContext(ctx context.Context, query string, limit int) (string, error) {
	// Set default limit
	if limit <= 0 {
		limit = 5
	}

	// Search for relevant content
	matches, err := r.vectorStore.QueryEmbeddings(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("failed to query embeddings: %w", err)
	}

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Build context
	var builder strings.Builder
	builder.WriteString("# Relevant Context\n\n")

	for i, match := range matches {
		builder.WriteString(fmt.Sprintf("## Source %d: %s (Relevance: %.2f)\n\n", i+1, match.Source, match.Score))
		builder.WriteString(match.Text)
		builder.WriteString("\n\n")
	}

	return builder.String(), nil
}

// EnhancePrompt adds retrieved context to a prompt
func (r *Retriever) EnhancePrompt(ctx context.Context, prompt, query string) (string, error) {
	context, err := r.RetrieveContext(ctx, query, 3)
	if err != nil {
		return prompt, err
	}

	enhanced := fmt.Sprintf(`
# Retrieved Context
%s

# Original Prompt
%s
`, context, prompt)

	return enhanced, nil
}
```

## Best Practices

1. **Collection Organization**

   - Create separate collections for different types of content
   - Use payload fields for efficient filtering

2. **Vector Size Selection**

   - Use model-appropriate vector sizes:
     - OpenAI ada-002: 1536 dimensions
     - Claude: 1024 dimensions
     - BERT: 768 dimensions

3. **Filtering and Caching**

   - Use metadata filters to narrow search scope
   - Implement caching for frequently accessed embeddings

4. **Performance Optimization**

   - Use batch operations for multiple embeddings
   - Implement connection pooling for high-traffic applications

5. **Storage Considerations**
   - Plan for vector storage growth
   - Implement TTL (time-to-live) for temporary embeddings

## Common Patterns

1. **Semantic Search**

   - Searching for conceptually similar content
   - Finding related objectives or tasks

2. **Contextual Enhancement**

   - Adding relevant context to prompts
   - Retrieving past agent interactions

3. **Clustering and Classification**
   - Grouping similar content
   - Auto-categorizing objectives or tasks

## Related Documentation

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
- [../architecture/task_execution_flow.md](../architecture/task_execution_flow.md)
