// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/corpus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCorpusDoc creates a mock corpus document for testing
func mockCorpusDoc(title, body, filePath string) *corpus.CorpusDoc {
	return &corpus.CorpusDoc{
		Title:    title,
		Body:     body,
		FilePath: filePath,
	}
}

func TestSearchCorpus(t *testing.T) {
	ctx := context.Background()

	// Create temporary corpus directory for testing
	tempDir := t.TempDir()
	corpusConfig := corpus.Config{
		CorpusPath: tempDir,
	}

	// Create test documents
	docs := []*corpus.CorpusDoc{
		mockCorpusDoc("Go Programming", "Go is a statically typed programming language", tempDir+"/go.md"),
		mockCorpusDoc("Python Tutorial", "Python is a dynamically typed programming language", tempDir+"/python.md"),
		mockCorpusDoc("JavaScript Guide", "JavaScript is used for web development", tempDir+"/js.md"),
		mockCorpusDoc("Database Design", "SQL databases use structured query language", tempDir+"/db.md"),
		mockCorpusDoc("Cloud Computing", "AWS and Azure are major cloud providers", tempDir+"/cloud.md"),
	}

	// Save documents to corpus
	for _, doc := range docs {
		err := corpus.Save(ctx, doc, corpusConfig)
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		query      string
		maxResults int
		wantCount  int
		wantDocs   []string // Expected document titles
	}{
		{
			name:       "Search for programming",
			query:      "programming",
			maxResults: 10,
			wantCount:  2,
			wantDocs:   []string{"Go Programming", "Python Tutorial"},
		},
		{
			name:       "Search for language",
			query:      "language",
			maxResults: 10,
			wantCount:  3,
			wantDocs:   []string{"Go Programming", "Python Tutorial", "Database Design"},
		},
		{
			name:       "Search with max results limit",
			query:      "language",
			maxResults: 2,
			wantCount:  2,
		},
		{
			name:       "Case insensitive search",
			query:      "PYTHON",
			maxResults: 10,
			wantCount:  1,
			wantDocs:   []string{"Python Tutorial"},
		},
		{
			name:       "No results",
			query:      "Rust",
			maxResults: 10,
			wantCount:  0,
			wantDocs:   []string{},
		},
		{
			name:       "Search in body content",
			query:      "statically typed",
			maxResults: 10,
			wantCount:  1,
			wantDocs:   []string{"Go Programming"},
		},
		{
			name:       "Partial word match",
			query:      "Java",
			maxResults: 10,
			wantCount:  1,
			wantDocs:   []string{"JavaScript Guide"},
		},
		{
			name:       "Max results of 1",
			query:      "programming",
			maxResults: 1,
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchCorpus(ctx, tt.query, corpusConfig, tt.maxResults)
			require.NoError(t, err)

			assert.Len(t, results, tt.wantCount)

			// Verify expected documents are found
			if len(tt.wantDocs) > 0 {
				foundDocs := make(map[string]bool)
				for _, result := range results {
					// Load the document to get the title
					doc, err := corpus.Load(ctx, result.Source)
					if err == nil {
						foundDocs[doc.Title] = true
					}
				}

				for _, expectedDoc := range tt.wantDocs {
					assert.True(t, foundDocs[expectedDoc], "Expected to find document: %s", expectedDoc)
				}
			}

			// Verify all results have high scores
			for _, result := range results {
				assert.Equal(t, float32(0.9), result.Score)
				assert.NotEmpty(t, result.Content)
				assert.NotEmpty(t, result.Source)
			}
		})
	}
}

func TestSearchCorpus_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		corpusConfig  corpus.Config
		query         string
		maxResults    int
		wantError     bool
		errorContains string
	}{
		{
			name: "Invalid corpus path",
			corpusConfig: corpus.Config{
				CorpusPath: "/nonexistent/path/that/should/not/exist",
			},
			query:      "test",
			maxResults: 10,
			wantError:  false, // corpus.List returns empty list for non-existent paths
		},
		{
			name: "Empty corpus directory",
			corpusConfig: corpus.Config{
				CorpusPath: t.TempDir(),
			},
			query:      "test",
			maxResults: 10,
			wantError:  false, // Should succeed with empty results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchCorpus(ctx, tt.query, tt.corpusConfig, tt.maxResults)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				// SearchCorpus returns empty slice, not nil
				if tt.name == "Empty corpus directory" {
					assert.Empty(t, results)
				} else {
					assert.NotNil(t, results)
				}
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "Exact match",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "Case insensitive match",
			s:        "Hello World",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "Mixed case match",
			s:        "HeLLo WoRLd",
			substr:   "HELLO",
			expected: true,
		},
		{
			name:     "Substring in middle",
			s:        "The quick brown fox",
			substr:   "quick",
			expected: true,
		},
		{
			name:     "No match",
			s:        "hello world",
			substr:   "goodbye",
			expected: false,
		},
		{
			name:     "Empty substring",
			s:        "hello world",
			substr:   "",
			expected: true,
		},
		{
			name:     "Empty string",
			s:        "",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "Both empty",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "Special characters",
			s:        "Hello, World!",
			substr:   "world!",
			expected: true,
		},
		{
			name:     "Unicode characters",
			s:        "Héllo Wörld",
			substr:   "HÉLLO",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSearchCorpus_CorruptedDocument(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	corpusConfig := corpus.Config{
		CorpusPath: tempDir,
	}

	// Create one valid document
	validDoc := mockCorpusDoc("Valid Document", "This is a valid document", tempDir+"/valid.md")
	err := corpus.Save(ctx, validDoc, corpusConfig)
	require.NoError(t, err)

	// Create a corrupted document file (this will fail to load)
	// We'll create a file that corpus.Load can't parse
	corruptedPath := tempDir + "/corrupted.md"
	err = corpus.Save(ctx, &corpus.CorpusDoc{
		FilePath: corruptedPath,
		// Missing required fields to cause load failure
	}, corpusConfig)

	// Search should still work and return the valid document
	results, err := SearchCorpus(ctx, "document", corpusConfig, 10)
	require.NoError(t, err)

	// Should find only the valid document
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "valid document")
}
