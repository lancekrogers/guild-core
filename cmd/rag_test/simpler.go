package main

import (
	"fmt"
	"strings"
)

type RetrievalConfig struct {
	MaxResults    int
	MinScore      float32
	IncludeCorpus bool
}

func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		MaxResults:    5,
		MinScore:      0.7,
		IncludeCorpus: true,
	}
}

type SearchMatch struct {
	Content string
	Score   float32
	Source  string
}

func formatSearchResults(matches []SearchMatch) string {
	var builder strings.Builder
	builder.WriteString("# Relevant Context\n\n")

	for i, match := range matches {
		builder.WriteString(fmt.Sprintf("## Source %d: %s (Relevance: %.2f)\n\n", 
			i+1, match.Source, match.Score))
		
		// Add content
		builder.WriteString(match.Content)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func main() {
	// Create test matches
	matches := []SearchMatch{
		{
			Content: "This is the first match content.",
			Score:   0.95,
			Source:  "Document 1",
		},
		{
			Content: "This is the second match content.",
			Score:   0.85,
			Source:  "Document 2",
		},
		{
			Content: "This is the third match content.",
			Score:   0.75,
			Source:  "Document 3",
		},
	}

	// Format the results
	formattedResults := formatSearchResults(matches)
	
	// Print formatted results
	fmt.Println(formattedResults)
	
	// Test retrieval config
	config := DefaultRetrievalConfig()
	fmt.Printf("Default retrieval config: MaxResults=%d, MinScore=%.2f, IncludeCorpus=%v\n", 
		config.MaxResults, config.MinScore, config.IncludeCorpus)
}