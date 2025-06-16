// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package rag provides example usage of the RAG system.
// This file demonstrates how to integrate the RAG system with the Guild framework.
// The new RAG system is provider-agnostic and supports offline operation.
package rag

import (
	"context"
	"fmt"
	"log"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// ExampleBasicUsage demonstrates basic RAG system usage with the new provider-agnostic design
func ExampleBasicUsage() {
	ctx := context.Background()

	// The new system auto-detects available providers, preferring Ollama for offline operation
	// Configure the RAG system
	config := Config{
		ChunkSize:       1000,
		ChunkOverlap:    200,
		MaxResults:      5,
		UseCorpus:       true,
		CorpusPath:      "./guild_memory/corpus",
		VectorStorePath: "./embeddings", // Updated path to match new design
	}

	// Create vector store config - the new system auto-detects providers
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		DefaultCollection: "rag_embeddings",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:  config.VectorStorePath,
			DefaultDimension: 768, // Common dimension for local models like nomic-embed-text
		},
		// EmbeddingProvider will be auto-detected if not specified
	}

	// Create vector store - it will auto-detect available embedding providers
	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// Create retriever with the new vector store
	retriever := NewRetrieverWithStore(vectorStore, config)

	// Index some documents
	err = retriever.AddDocument(ctx,
		"guild-intro",
		"The Guild Framework orchestrates AI agents to work together on complex tasks. "+
			"Agents are called Artisans and work in teams called Guilds.",
		"documentation")
	if err != nil {
		log.Printf("Failed to add document: %v", err)
	}

	// Query the system
	query := "What are agents called in Guild?"
	retrievalConfig := RetrievalConfig{
		MaxResults:      3,
		MinScore:        0.5,
		IncludeMetadata: true,
	}

	results, err := retriever.RetrieveContext(ctx, query, retrievalConfig)
	if err != nil {
		log.Printf("Failed to retrieve context: %v", err)
	}

	// Display results
	fmt.Printf("Found %d results for query: %s\n", len(results.Results), query)
	for i, result := range results.Results {
		fmt.Printf("\nResult %d (score: %.2f):\n", i+1, result.Score)
		fmt.Printf("Content: %s\n", result.Content)
		if result.Source != "" {
			fmt.Printf("Source: %s\n", result.Source)
		}
	}
}

// ExampleWithOllama demonstrates using the RAG system with Ollama for offline operation
func ExampleWithOllama() {
	ctx := context.Background()

	// Create Ollama provider for offline operation
	factory := providers.NewFactoryV2()
	ollamaProvider, err := factory.CreateAIProvider(providers.ProviderOllama, "http://localhost:11434")
	if err != nil {
		log.Fatalf("Failed to create Ollama provider: %v", err)
	}

	// Create vector store config with explicit Ollama provider
	storeConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: ollamaProvider,
		EmbeddingModel:    "nomic-embed-text", // Excellent local embedding model
		DefaultCollection: "guild_knowledge",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:  "./embeddings",
			DefaultDimension: 768, // nomic-embed-text dimension
		},
	}

	vectorStore, err := vector.NewVectorStore(ctx, storeConfig)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// Create RAG config
	ragConfig := Config{
		ChunkSize:       1000,
		ChunkOverlap:    200,
		MaxResults:      5,
		VectorStorePath: "./embeddings",
	}

	// Create retriever
	retriever := NewRetrieverWithStore(vectorStore, ragConfig)

	// Example: Add a document
	err = retriever.AddDocument(ctx,
		"ollama-intro",
		"Ollama enables running large language models locally on your machine.",
		"ollama-docs")
	if err != nil {
		log.Printf("Failed to add document: %v", err)
	}

	fmt.Println("RAG system initialized with Ollama for offline operation")
}

// ExampleCorpusIntegration demonstrates RAG with corpus integration using the new on-demand model
func ExampleCorpusIntegration() {
	ctx := context.Background()

	// Configure corpus
	corpusConfig := corpus.Config{
		CorpusPath:      "./guild_memory/corpus",
		ActivitiesPath:  "./guild_memory/corpus/.activities",
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB
		DefaultCategory: "general",
	}

	// Create vector store with auto-detection
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		DefaultCollection: "corpus_embeddings",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:  "./embeddings",
			DefaultDimension: 768,
		},
	}

	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// Create RAG config with corpus integration
	ragConfig := Config{
		ChunkSize:       1000,
		ChunkOverlap:    200,
		ChunkStrategy:   "paragraph",
		MaxResults:      10,
		UseCorpus:       true,
		CorpusPath:      corpusConfig.CorpusPath,
		CorpusMaxSizeMB: int(corpusConfig.MaxSizeBytes / 1024 / 1024),
	}

	// Create retriever
	retriever := NewRetrieverWithStore(vectorStore, ragConfig)

	// The new on-demand model means corpus documents are synced to RAG
	// Users run 'guild corpus scan' to update embeddings when corpus changes
	fmt.Println("Run 'guild corpus scan' to sync corpus documents to RAG system")

	// Search with corpus integration
	query := "How does the agent system work?"
	retrievalConfig := RetrievalConfig{
		MaxResults:      10,
		MinScore:        0.5, // Lower threshold for better recall
		UseCorpus:       true,
		IncludeMetadata: true,
	}

	results, err := retriever.RetrieveContext(ctx, query, retrievalConfig)
	if err != nil {
		log.Printf("Failed to retrieve context: %v", err)
	}

	// Display results
	fmt.Printf("Found %d results (including corpus) for: %s\n", len(results.Results), query)
	for i, result := range results.Results {
		fmt.Printf("\nResult %d (score: %.2f):\n", i+1, result.Score)
		fmt.Printf("Content snippet: %.100s...\n", result.Content)
		if result.Source != "" {
			fmt.Printf("Source: %s\n", result.Source)
		}
		if len(result.Metadata) > 0 {
			fmt.Printf("Metadata: %v\n", result.Metadata)
		}
	}
}

// ExampleProviderAutoDetection demonstrates the auto-detection feature
func ExampleProviderAutoDetection() {
	ctx := context.Background()

	// Create vector store without specifying a provider
	// The system will auto-detect available providers
	storeConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		DefaultCollection: "auto_detect_demo",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:  "./embeddings",
			DefaultDimension: 768, // Will adapt based on detected model
		},
		// No EmbeddingProvider specified - will auto-detect
	}

	vectorStore, err := vector.NewVectorStore(ctx, storeConfig)
	if err != nil {
		// If no providers are available, it will use NoOpEmbedder
		log.Printf("Vector store created with fallback: %v", err)
	}
	defer vectorStore.Close()

	fmt.Println("Vector store created with auto-detected provider")
	fmt.Println("Priority order: Ollama (local) → OpenAI → Anthropic → NoOp")
}
