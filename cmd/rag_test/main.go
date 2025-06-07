package main

import (
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/memory/rag"
)

func main() {
	// Just a simple program to verify the RAG code can be imported
	fmt.Println("RAG implementation test")
	
	// Create a chunker with default config
	chunkerConfig := rag.ChunkerConfig{
		ChunkSize:    1000,
		ChunkOverlap: 100,
	}
	chunker, err := rag.DefaultChunkerFactory(chunkerConfig)
	if err != nil {
		fmt.Printf("Error creating chunker: %v\n", err)
		return
	}
	fmt.Printf("Created chunker with config: %+v\n", chunkerConfig)
	_ = chunker // Mark as used
	
	// Create a default retrieval config
	config := rag.DefaultRetrievalConfig()
	fmt.Printf("Default retrieval config: MaxResults=%d, MinScore=%.2f\n", 
		config.MaxResults, config.MinScore)
	
	fmt.Println("RAG implementation can be imported successfully!")
}