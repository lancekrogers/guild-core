// Package rag provides example usage of the RAG system.
// This file demonstrates how to integrate the RAG system with the Guild framework.
package rag

import (
	"context"
	"fmt"
	"log"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// ExampleBasicUsage demonstrates basic RAG system usage
func ExampleBasicUsage() {
	ctx := context.Background()

	// Create an embedder (would use real implementation in production)
	embedder := &vector.MockEmbedder{Dimension: 1536}

	// Configure the RAG system
	config := Config{
		ChunkSize:       1000,
		ChunkOverlap:    200,
		MaxResults:      5,
		UseCorpus:       true,
		CorpusPath:      "./guild_memory/corpus",
		VectorStorePath: "./data/vectors",
	}

	// Create a retriever
	retriever, err := NewRetriever(ctx, embedder, config)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	defer retriever.Close()

	// Index some documents
	err = retriever.AddDocument(ctx, "doc1",
		"The Guild Framework orchestrates AI agents to work together on complex tasks. "+
			"Agents are called Artisans and work in teams called Guilds.",
		"documentation")
	if err != nil {
		log.Printf("Failed to add document: %v", err)
	}

	// Index a corpus document
	corpusDoc := &corpus.CorpusDoc{
		Title: "Agent Communication Protocol",
		Body: "Agents communicate through the event bus system. "+
			"Messages are passed using a publish-subscribe pattern.",
		Tags:    []string{"agents", "communication", "architecture"},
		GuildID: "guild1",
		AgentID: "agent1",
	}
	err = retriever.AddCorpusDocument(ctx, corpusDoc)
	if err != nil {
		log.Printf("Failed to add corpus document: %v", err)
	}

	// Retrieve context for a query
	retrievalConfig := RetrievalConfig{
		MaxResults:      3,
		MinScore:        0.1,
		IncludeMetadata: true,
		UseCorpus:       true,
	}

	results, err := retriever.RetrieveContext(ctx, "How do agents communicate?", retrievalConfig)
	if err != nil {
		log.Printf("Failed to retrieve context: %v", err)
	}

	// Display results
	fmt.Printf("Found %d relevant results:\n", len(results.Results))
	for i, result := range results.Results {
		fmt.Printf("\n%d. Source: %s (Score: %.3f)\n", i+1, result.Source, result.Score)
		fmt.Printf("   Content: %s\n", result.Content)
		if result.Metadata != nil {
			fmt.Printf("   Metadata: %v\n", result.Metadata)
		}
	}

	// Enhance a prompt with context
	enhancedPrompt, err := retriever.EnhancePrompt(ctx,
		"Explain how agents work together in the Guild framework",
		retrievalConfig)
	if err != nil {
		log.Printf("Failed to enhance prompt: %v", err)
	}

	fmt.Printf("\nEnhanced Prompt:\n%s\n", enhancedPrompt)
}

// ExampleWithVectorStoreFactory demonstrates using the vector store factory
func ExampleWithVectorStoreFactory() {
	ctx := context.Background()

	// Create vector store using factory
	storeConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		OpenAIApiKey:      "your-api-key", // In production, use env var
		DefaultCollection: "agent_memories",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:  "./data/vectors",
			DefaultDimension: 1536,
		},
	}

	vectorStore, err := vector.NewVectorStore(ctx, storeConfig)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// Use the vector store for embeddings
	embedding := vector.Embedding{
		Text:   "Guild agents collaborate through shared objectives",
		Source: "example",
		Metadata: map[string]interface{}{
			"type": "example",
		},
	}

	err = vectorStore.SaveEmbedding(ctx, embedding)
	if err != nil {
		log.Printf("Failed to save embedding: %v", err)
	}

	// Query embeddings
	matches, err := vectorStore.QueryEmbeddings(ctx, "How do agents collaborate?", 5)
	if err != nil {
		log.Printf("Failed to query embeddings: %v", err)
	}

	fmt.Printf("Found %d matches\n", len(matches))
}

// ExampleRAGAgentIntegration demonstrates integrating RAG with an agent
func ExampleRAGAgentIntegration() {
	ctx := context.Background()

	// Create embedder and retriever
	embedder := &vector.MockEmbedder{Dimension: 1536}
	config := DefaultConfig()
	retriever, err := NewRetriever(ctx, embedder, config)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	defer retriever.Close()

	// In a real agent's Execute method:
	agentRequest := "What are the best practices for agent communication?"

	// Enhance the request with RAG context
	retrievalConfig := DefaultRetrievalConfig()
	retrievalConfig.MaxResults = 3

	enhancedRequest, err := retriever.EnhancePrompt(ctx, agentRequest, retrievalConfig)
	if err != nil {
		// Fall back to original request on error
		enhancedRequest = agentRequest
	}

	// The agent would then use the enhanced request with its LLM
	fmt.Printf("Original request: %s\n", agentRequest)
	fmt.Printf("Enhanced request length: %d characters\n", len(enhancedRequest))
}

// ExampleCorpusIntegration demonstrates corpus and RAG integration
func ExampleCorpusIntegration() {
	ctx := context.Background()

	// Configure RAG with corpus support
	config := Config{
		ChunkSize:       500,
		ChunkOverlap:    100,
		UseCorpus:       true,
		CorpusPath:      "./guild_memory/corpus",
		CorpusMaxSizeMB: 100,
	}

	embedder := &vector.MockEmbedder{Dimension: 1536}
	retriever, err := NewRetriever(ctx, embedder, config)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	defer retriever.Close()

	// The retriever will now search both vector embeddings and corpus documents
	retrievalConfig := RetrievalConfig{
		MaxResults:          5,
		UseCorpus:           true,
		DisableVectorSearch: false, // Use both vector and corpus search
	}

	results, err := retriever.RetrieveContext(ctx, "agent architecture", retrievalConfig)
	if err != nil {
		log.Printf("Failed to retrieve context: %v", err)
	}

	// Results will include both vector matches and corpus documents
	for _, result := range results.Results {
		fmt.Printf("Source: %s, Score: %.3f\n", result.Source, result.Score)
	}
}