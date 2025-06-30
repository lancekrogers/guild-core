// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/ui/chat/common/config"
	"github.com/lancekrogers/guild/internal/ui/chat/panes"
	"github.com/lancekrogers/guild/pkg/corpus"
	"github.com/lancekrogers/guild/pkg/observability"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGuildClient provides a mock implementation of the gRPC client
type MockGuildClient struct {
	pb.GuildClient
}

// TestCorpusHandler_Handle tests the main corpus command handler
func TestCorpusHandler_Handle(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "corpus_test")
	
	tests := []struct {
		name           string
		args           []string
		expectedMsgType interface{}
		expectedError  bool
		setupCorpus    func(t *testing.T) string
	}{
		{
			name:           "no args shows list",
			args:           []string{},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "list command",
			args:           []string{"list"},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupTestCorpus,
		},
		{
			name:           "search command with query",
			args:           []string{"search", "authentication", "patterns"},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupTestCorpus,
		},
		{
			name:           "search command without query",
			args:           []string{"search"},
			expectedMsgType: panes.StatusUpdateMsg{},
			expectedError:  true,
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "add command with content",
			args:           []string{"add", "pattern", "Use repository pattern for data access"},
			expectedMsgType: panes.StatusUpdateMsg{},
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "add command insufficient args",
			args:           []string{"add", "pattern"},
			expectedMsgType: panes.StatusUpdateMsg{},
			expectedError:  true,
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "stats command",
			args:           []string{"stats"},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupTestCorpus,
		},
		{
			name:           "config command", 
			args:           []string{"config"},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "help command",
			args:           []string{"help"},
			expectedMsgType: panes.PaneUpdateMsg{},
			setupCorpus:    setupEmptyCorpus,
		},
		{
			name:           "unknown command",
			args:           []string{"unknown"},
			expectedMsgType: panes.StatusUpdateMsg{},
			expectedError:  true,
			setupCorpus:    setupEmptyCorpus,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup corpus directory
			corpusPath := tt.setupCorpus(t)
			defer os.RemoveAll(corpusPath)
			
			// Set environment variable for corpus location
			oldPath := os.Getenv("GUILD_CORPUS_PATH")
			os.Setenv("GUILD_CORPUS_PATH", corpusPath)
			defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
			
			// Create handler
			chatConfig := &config.ChatConfig{}
			client := &MockGuildClient{}
			handler := NewCorpusHandler(chatConfig, client)
			
			// Execute command
			cmd := handler.Handle(ctx, tt.args)
			require.NotNil(t, cmd)
			
			// Get result
			msg := cmd()
			
			// Verify message type
			switch tt.expectedMsgType.(type) {
			case panes.PaneUpdateMsg:
				_, ok := msg.(panes.PaneUpdateMsg)
				assert.True(t, ok, "Expected PaneUpdateMsg, got %T", msg)
			case panes.StatusUpdateMsg:
				statusMsg, ok := msg.(panes.StatusUpdateMsg)
				assert.True(t, ok, "Expected StatusUpdateMsg, got %T", msg)
				
				if tt.expectedError {
					assert.Equal(t, "error", statusMsg.Level)
				} else {
					assert.NotEqual(t, "error", statusMsg.Level)
				}
			}
		})
	}
}

// TestCorpusHandler_Search tests search functionality in detail
func TestCorpusHandler_Search(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "corpus_search_test")
	
	// Setup corpus with test documents
	corpusPath := setupTestCorpus(t)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	tests := []struct {
		name          string
		query         []string
		expectResults bool
		expectKeyword string
	}{
		{
			name:          "search existing content",
			query:         []string{"authentication"},
			expectResults: true,
			expectKeyword: "authentication",
		},
		{
			name:          "search non-existing content",
			query:         []string{"nonexistent"},
			expectResults: false,
			expectKeyword: "",
		},
		{
			name:          "search multiple words",
			query:         []string{"database", "pattern"},
			expectResults: true,
			expectKeyword: "database",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"search"}, tt.query...)
			cmd := handler.Handle(ctx, args)
			require.NotNil(t, cmd)
			
			msg := cmd()
			paneMsg, ok := msg.(panes.PaneUpdateMsg)
			require.True(t, ok)
			
			content := paneMsg.Content
			
			if tt.expectResults {
				assert.Contains(t, content, "Search Results")
				assert.Contains(t, strings.ToLower(content), strings.ToLower(tt.expectKeyword))
				assert.NotContains(t, content, "No results found")
			} else {
				assert.Contains(t, content, "No results found")
			}
		})
	}
}

// TestCorpusHandler_Add tests adding documents to corpus
func TestCorpusHandler_Add(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "corpus_add_test")
	
	// Setup empty corpus
	corpusPath := setupEmptyCorpus(t)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	// Add a document
	args := []string{"add", "pattern", "Use repository pattern for clean data access"}
	cmd := handler.Handle(ctx, args)
	require.NotNil(t, cmd)
	
	msg := cmd()
	statusMsg, ok := msg.(panes.StatusUpdateMsg)
	require.True(t, ok)
	
	// Should be success
	assert.Equal(t, "success", statusMsg.Level)
	assert.Contains(t, statusMsg.Message, "Added")
	assert.Contains(t, statusMsg.Message, "Pattern:")
	
	// Verify document was actually saved
	cfg, err := corpus.GetConfigWithFallback(ctx)
	require.NoError(t, err)
	
	docs, err := corpus.List(ctx, cfg)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	
	// Load and verify content
	doc, err := corpus.Load(ctx, docs[0])
	require.NoError(t, err)
	assert.Contains(t, doc.Body, "repository pattern")
	assert.Contains(t, doc.Tags, "pattern")
}

// TestCorpusHandler_Stats tests statistics functionality
func TestCorpusHandler_Stats(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "corpus_stats_test")
	
	// Setup corpus with known documents
	corpusPath := setupTestCorpus(t)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	// Get stats
	cmd := handler.Handle(ctx, []string{"stats"})
	require.NotNil(t, cmd)
	
	msg := cmd()
	paneMsg, ok := msg.(panes.PaneUpdateMsg)
	require.True(t, ok)
	
	content := paneMsg.Content
	
	// Should contain statistics
	assert.Contains(t, content, "Corpus Statistics")
	assert.Contains(t, content, "Documents:")
	assert.Contains(t, content, "Total Size:")
	assert.Contains(t, content, "Top Tags")
	assert.Contains(t, content, "Sources")
}

// TestKnowledgeHandler tests the knowledge command handler
func TestKnowledgeHandler(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "knowledge_test")
	
	handler := NewKnowledgeHandler()
	
	tests := []struct {
		name           string
		args           []string
		expectedMsgType interface{}
	}{
		{
			name:           "no args shows browse",
			args:           []string{},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "browse command",
			args:           []string{"browse"},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "validate command",
			args:           []string{"validate"},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "export command",
			args:           []string{"export"},
			expectedMsgType: panes.StatusUpdateMsg{},
		},
		{
			name:           "graph overview",
			args:           []string{"graph"},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "graph specific node",
			args:           []string{"graph", "node123"},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "unknown command",
			args:           []string{"unknown"},
			expectedMsgType: panes.StatusUpdateMsg{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := handler.Handle(ctx, tt.args)
			require.NotNil(t, cmd)
			
			msg := cmd()
			
			switch tt.expectedMsgType.(type) {
			case panes.PaneUpdateMsg:
				_, ok := msg.(panes.PaneUpdateMsg)
				assert.True(t, ok, "Expected PaneUpdateMsg, got %T", msg)
			case panes.StatusUpdateMsg:
				_, ok := msg.(panes.StatusUpdateMsg)
				assert.True(t, ok, "Expected StatusUpdateMsg, got %T", msg)
			}
		})
	}
}

// TestIndexHandler tests the index command handler
func TestIndexHandler(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "index_test")
	
	handler := NewIndexHandler()
	
	tests := []struct {
		name           string
		args           []string
		expectedMsgType interface{}
	}{
		{
			name:           "no args shows status",
			args:           []string{},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "status command",
			args:           []string{"status"},
			expectedMsgType: panes.PaneUpdateMsg{},
		},
		{
			name:           "rebuild command",
			args:           []string{"rebuild"},
			expectedMsgType: panes.StatusUpdateMsg{},
		},
		{
			name:           "optimize command",
			args:           []string{"optimize"},
			expectedMsgType: panes.StatusUpdateMsg{},
		},
		{
			name:           "unknown command",
			args:           []string{"unknown"},
			expectedMsgType: panes.StatusUpdateMsg{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := handler.Handle(ctx, tt.args)
			require.NotNil(t, cmd)
			
			msg := cmd()
			
			switch tt.expectedMsgType.(type) {
			case panes.PaneUpdateMsg:
				_, ok := msg.(panes.PaneUpdateMsg)
				assert.True(t, ok, "Expected PaneUpdateMsg, got %T", msg)
			case panes.StatusUpdateMsg:
				_, ok := msg.(panes.StatusUpdateMsg)
				assert.True(t, ok, "Expected StatusUpdateMsg, got %T", msg)
			}
		})
	}
}

// TestCorpusIntegration tests the full corpus workflow
func TestCorpusIntegration(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "corpus_integration_test")
	
	// Setup corpus
	corpusPath := setupEmptyCorpus(t)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	// Test complete workflow: add -> list -> search -> stats
	
	// 1. Add documents
	documents := []struct {
		docType string
		content string
	}{
		{"pattern", "Use repository pattern for data access layer"},
		{"decision", "Decided to use JWT tokens for authentication"},
		{"tip", "Always validate user input on server side"},
		{"example", "func ValidateEmail(email string) bool { ... }"},
	}
	
	for _, doc := range documents {
		args := []string{"add", doc.docType, doc.content}
		cmd := handler.Handle(ctx, args)
		require.NotNil(t, cmd)
		
		msg := cmd()
		statusMsg, ok := msg.(panes.StatusUpdateMsg)
		require.True(t, ok)
		assert.Equal(t, "success", statusMsg.Level)
	}
	
	// 2. List documents
	cmd := handler.Handle(ctx, []string{"list"})
	require.NotNil(t, cmd)
	
	msg := cmd()
	paneMsg, ok := msg.(panes.PaneUpdateMsg)
	require.True(t, ok)
	
	content := paneMsg.Content
	assert.Contains(t, content, "Total Documents: 4")
	assert.Contains(t, content, "repository pattern")
	assert.Contains(t, content, "JWT tokens")
	
	// 3. Search documents
	cmd = handler.Handle(ctx, []string{"search", "authentication"})
	require.NotNil(t, cmd)
	
	msg = cmd()
	paneMsg, ok = msg.(panes.PaneUpdateMsg)
	require.True(t, ok)
	
	searchContent := paneMsg.Content
	assert.Contains(t, searchContent, "Search Results")
	assert.Contains(t, searchContent, "JWT")
	
	// 4. Get statistics
	cmd = handler.Handle(ctx, []string{"stats"})
	require.NotNil(t, cmd)
	
	msg = cmd()
	paneMsg, ok = msg.(panes.PaneUpdateMsg)
	require.True(t, ok)
	
	statsContent := paneMsg.Content
	assert.Contains(t, statsContent, "Documents: 4")
	assert.Contains(t, statsContent, "pattern")
	assert.Contains(t, statsContent, "decision")
	assert.Contains(t, statsContent, "tip")
	assert.Contains(t, statsContent, "example")
}

// TestRelevanceCalculation tests the search relevance scoring
func TestRelevanceCalculation(t *testing.T) {
	tests := []struct {
		name          string
		doc           *corpus.CorpusDoc
		query         string
		expectedScore float64
		scoreRange    [2]float64 // min, max
	}{
		{
			name: "exact title match",
			doc: &corpus.CorpusDoc{
				Title: "Authentication Patterns",
				Body:  "Some content about auth",
				Tags:  []string{"pattern"},
			},
			query:      "authentication",
			scoreRange: [2]float64{1.0, 2.0}, // Title + body match
		},
		{
			name: "tag match",
			doc: &corpus.CorpusDoc{
				Title: "Some Title",
				Body:  "Some content",
				Tags:  []string{"authentication", "security"},
			},
			query:      "authentication",
			scoreRange: [2]float64{0.8, 1.0}, // Tag match only
		},
		{
			name: "body match multiple times",
			doc: &corpus.CorpusDoc{
				Title: "Some Title",
				Body:  "Authentication is important. Use authentication carefully. Authentication tokens expire.",
				Tags:  []string{"pattern"},
			},
			query:      "authentication",
			scoreRange: [2]float64{0.7, 1.0}, // Body match with frequency boost
		},
		{
			name: "no match",
			doc: &corpus.CorpusDoc{
				Title: "Database Patterns",
				Body:  "Content about databases",
				Tags:  []string{"database"},
			},
			query:      "authentication", 
			scoreRange: [2]float64{0.0, 0.0}, // No match
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateRelevance(tt.doc, strings.ToLower(tt.query))
			
			if tt.expectedScore > 0 {
				assert.Equal(t, tt.expectedScore, score)
			} else {
				assert.GreaterOrEqual(t, score, tt.scoreRange[0])
				assert.LessOrEqual(t, score, tt.scoreRange[1])
			}
		})
	}
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("extractPreview", func(t *testing.T) {
		content := "This is a long piece of content that should be truncated when we extract a preview from it. The query word appears here: authentication. And then continues with more text."
		query := "authentication"
		maxLength := 80
		
		preview := extractPreview(content, query, maxLength)
		
		assert.LessOrEqual(t, len(preview), maxLength+10) // Allow for "..." 
		assert.Contains(t, preview, query)
	})
	
	t.Run("extractTitle", func(t *testing.T) {
		tests := []struct {
			content  string
			expected string
		}{
			{
				content:  "Short title",
				expected: "Short title",
			},
			{
				content:  "This is a very long title that should be truncated at some reasonable point",
				expected: "This is a very long title that should be truncated...",
			},
			{
				content:  "Multi line\ncontent should use\nfirst line only",
				expected: "Multi line",
			},
		}
		
		for _, tt := range tests {
			result := extractTitle(tt.content)
			assert.Equal(t, tt.expected, result)
		}
	})
}

// BenchmarkSearch benchmarks search performance
func BenchmarkSearch(b *testing.B) {
	ctx := context.Background()
	
	// Setup corpus with many documents
	corpusPath := setupLargeCorpus(b)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	queries := []string{
		"authentication",
		"database pattern",
		"error handling",
		"testing strategy",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		args := []string{"search", query}
		
		cmd := handler.Handle(ctx, args)
		if cmd != nil {
			cmd() // Execute the command
		}
	}
}

// Helper functions for test setup

// setupEmptyCorpus creates an empty corpus directory
func setupEmptyCorpus(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "guild-corpus-test-*")
	require.NoError(t, err)
	
	return tempDir
}

// setupTestCorpus creates a corpus with test documents
func setupTestCorpus(t *testing.T) string {
	tempDir := setupEmptyCorpus(t)
	
	// Create test configuration
	cfg := corpus.Config{
		CorpusPath:      tempDir,
		MaxSizeBytes:    10 * 1024 * 1024, // 10MB
		DefaultCategory: "general",
	}
	
	// Create test documents
	testDocs := []struct {
		title   string
		content string
		tags    []string
		source  string
	}{
		{
			title:   "Authentication Guide",
			content: "This document explains authentication patterns and best practices. Use JWT tokens for stateless authentication. Always validate tokens on the server side.",
			tags:    []string{"authentication", "security", "guide"},
			source:  "documentation",
		},
		{
			title:   "Database Patterns",
			content: "Repository pattern provides a clean abstraction for data access. Use it to separate business logic from data access logic.",
			tags:    []string{"database", "pattern", "architecture"},
			source:  "documentation",
		},
		{
			title:   "Testing Strategies", 
			content: "Write unit tests first. Integration tests should cover the happy path and error scenarios. Use mocks for external dependencies.",
			tags:    []string{"testing", "strategy", "best-practice"},
			source:  "documentation",
		},
		{
			title:   "Error Handling",
			content: "Always handle errors gracefully. Return meaningful error messages. Log errors for debugging but don't expose internal details to users.",
			tags:    []string{"error", "handling", "best-practice"},
			source:  "documentation",
		},
	}
	
	ctx := context.Background()
	
	for _, docData := range testDocs {
		doc := corpus.NewCorpusDoc(
			docData.title,
			docData.source,
			docData.content,
			"test-guild",
			"test-agent",
			docData.tags,
		)
		
		err := corpus.Save(ctx, doc, cfg)
		require.NoError(t, err)
	}
	
	return tempDir
}

// setupLargeCorpus creates a corpus with many documents for benchmarking
func setupLargeCorpus(b *testing.B) string {
	tempDir, err := os.MkdirTemp("", "guild-corpus-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	
	cfg := corpus.Config{
		CorpusPath:      tempDir,
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB
		DefaultCategory: "general",
	}
	
	ctx := context.Background()
	
	// Generate many test documents
	topics := []string{"authentication", "database", "testing", "performance", "security", "architecture"}
	types := []string{"pattern", "decision", "tip", "example", "reference"}
	
	for i := 0; i < 100; i++ {
		topic := topics[i%len(topics)]
		docType := types[i%len(types)]
		
		doc := corpus.NewCorpusDoc(
			fmt.Sprintf("%s %s %d", strings.Title(topic), strings.Title(docType), i),
			"generated",
			fmt.Sprintf("This is document %d about %s. It contains information about %s patterns and best practices. Use this as a %s for your projects.", i, topic, topic, docType),
			"bench-guild",
			"bench-agent",
			[]string{topic, docType},
		)
		
		err := corpus.Save(ctx, doc, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	return tempDir
}

// TestSearchInterface_Integration tests the search interface integration
func TestSearchInterface_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// TODO: This test needs components package integration
	t.Skip("Search interface integration requires components package - skipping for now")
}

// TestErrorHandling tests error handling in corpus commands
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()
	
	// Test with invalid corpus path
	os.Setenv("GUILD_CORPUS_PATH", "/nonexistent/path")
	defer os.Unsetenv("GUILD_CORPUS_PATH")
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	handler := NewCorpusHandler(chatConfig, client)
	
	// Most operations should handle the error gracefully
	cmd := handler.Handle(ctx, []string{"list"})
	require.NotNil(t, cmd)
	
	msg := cmd()
	
	// Should get either an error status or an empty results message
	switch msg := msg.(type) {
	case panes.StatusUpdateMsg:
		assert.Equal(t, "error", msg.Level)
	case panes.PaneUpdateMsg:
		// Empty corpus should still work
		assert.Contains(t, msg.Content, "No documents found")
	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}

// TestConcurrentAccess tests concurrent access to corpus
func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	
	corpusPath := setupTestCorpus(t)
	defer os.RemoveAll(corpusPath)
	
	oldPath := os.Getenv("GUILD_CORPUS_PATH")
	os.Setenv("GUILD_CORPUS_PATH", corpusPath)
	defer os.Setenv("GUILD_CORPUS_PATH", oldPath)
	
	chatConfig := &config.ChatConfig{}
	client := &MockGuildClient{}
	
	// Test concurrent handlers
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			handler := NewCorpusHandler(chatConfig, client)
			
			// Perform various operations
			operations := [][]string{
				{"list"},
				{"search", "authentication"},
				{"stats"},
				{"config"},
			}
			
			for _, args := range operations {
				cmd := handler.Handle(ctx, args)
				if cmd != nil {
					cmd() // Execute command
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}