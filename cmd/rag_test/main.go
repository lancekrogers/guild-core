package main

import (
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/memory/rag"
)

func main() {
	// Just a simple program to verify the RAG code can be imported
	fmt.Println("RAG implementation test")
	
	// Create a chunker
	chunker := rag.NewChunker()
	fmt.Printf("Created chunker with chunk size: %d\n", chunker.ChunkSize)
	
	// Create a default retrieval config
	config := rag.DefaultRetrievalConfig()
	fmt.Printf("Default retrieval config: MaxResults=%d, MinScore=%.2f\n", 
		config.MaxResults, config.MinScore)
	
	fmt.Println("RAG implementation can be imported successfully!")
}