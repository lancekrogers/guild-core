// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"strings"

	"github.com/guild-framework/guild-core/pkg/corpus"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// SearchCorpus searches the corpus for documents matching a query
func SearchCorpus(ctx context.Context, query string, corpusConfig corpus.Config, maxResults int) ([]SearchResult, error) {
	// This is a stub implementation without actual vector search
	results := make([]SearchResult, 0)

	// List documents
	docs, err := corpus.List(ctx, corpusConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list corpus documents").
			WithComponent("memory").
			WithOperation("SearchCorpus")
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
