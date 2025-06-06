package rag

import (
	"context"
	"strings"
	
	"github.com/guild-ventures/guild-core/internal/corpus"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// SearchCorpus searches the corpus for documents matching a query
func SearchCorpus(ctx context.Context, query string, corpusConfig corpus.Config, maxResults int) ([]SearchResult, error) {
	// This is a stub implementation without actual vector search
	var results []SearchResult
	
	// List documents
	docs, err := corpus.List(ctx, corpusConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage).
			WithComponent("memory").
			WithOperation("SearchCorpus").
			WithDetails("failed to list corpus documents")
	}
	
	// Simple matching (not using vectors)
	count := 0
	for _, docPath := range docs {
		if count >= maxResults {
			break
		}
		
		// Load document
		doc, err := corpus.Load(ctx, docPath)
		if err != nil {
			continue
		}
		
		// Simple check if query appears in document
		if containsIgnoreCase(doc.Title, query) || containsIgnoreCase(doc.Body, query) {
			result := SearchResult{
				Content: doc.Body,
				Source:  doc.FilePath,
				Score:   0.9, // Arbitrary high score
			}
			results = append(results, result)
			count++
		}
	}
	
	return results, nil
}

// containsIgnoreCase checks if a string contains a substring, ignoring case
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}