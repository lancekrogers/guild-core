// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchInterface_Creation(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "search_interface_test")

	si := NewSearchInterface(ctx)

	assert.NotNil(t, si)
	assert.Equal(t, "", si.GetQuery())
	assert.Equal(t, 0, si.selected)
	assert.Equal(t, false, si.showDetails)
	assert.Equal(t, 80, si.width)
	assert.Equal(t, 24, si.height)
	assert.Equal(t, 0.5, si.filters.MinScore)
}

func TestSearchInterface_QueryManagement(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test setting query
	si.SetQuery("test query")
	assert.Equal(t, "test query", si.GetQuery())

	// Test setting query with whitespace
	si.SetQuery("  spaced query  ")
	assert.Equal(t, "spaced query", si.GetQuery())

	// Test empty query
	si.SetQuery("")
	assert.Equal(t, "", si.GetQuery())
}

func TestSearchInterface_ResultsManagement(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Create test results
	results := []SearchResult{
		{
			ID:        "1",
			Title:     "Authentication Guide",
			Path:      "auth/guide.md",
			Source:    "documentation",
			Tags:      []string{"auth", "guide"},
			Preview:   "This document explains authentication patterns",
			Score:     0.95,
			Type:      "guide",
			UpdatedAt: time.Now(),
		},
		{
			ID:        "2",
			Title:     "Database Patterns",
			Path:      "db/patterns.md",
			Source:    "documentation",
			Tags:      []string{"database", "pattern"},
			Preview:   "Repository pattern for data access",
			Score:     0.85,
			Type:      "pattern",
			UpdatedAt: time.Now().Add(-time.Hour),
		},
	}

	si.SetResults(results)

	assert.Len(t, si.results, 2)
	assert.Equal(t, 0, si.selected)
	assert.False(t, si.showDetails)

	// Test getting selected result
	selected := si.GetSelectedResult()
	require.NotNil(t, selected)
	assert.Equal(t, "1", selected.ID)
	assert.Equal(t, "Authentication Guide", selected.Title)
}

func TestSearchInterface_FilterManagement(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test adding filters
	err := si.AddFilter("type", "pattern")
	assert.NoError(t, err)

	err = si.AddFilter("tag", "authentication")
	assert.NoError(t, err)

	err = si.AddFilter("source", "documentation")
	assert.NoError(t, err)

	err = si.AddFilter("author", "test-author")
	assert.NoError(t, err)

	err = si.AddFilter("score", "0.8")
	assert.NoError(t, err)

	err = si.AddFilter("project", "current")
	assert.NoError(t, err)

	filters := si.GetFilters()
	assert.Contains(t, filters.Types, "pattern")
	assert.Contains(t, filters.Tags, "authentication")
	assert.Contains(t, filters.Sources, "documentation")
	assert.Equal(t, "test-author", filters.Author)
	assert.Equal(t, 0.8, filters.MinScore)
	assert.True(t, filters.InCurrentProject)

	// Test invalid filter type
	err = si.AddFilter("unknown", "value")
	assert.Error(t, err)

	// Test invalid score
	err = si.AddFilter("score", "invalid")
	assert.Error(t, err)

	// Test removing filters
	si.RemoveFilter("type", "pattern")
	si.RemoveFilter("tag", "authentication")
	si.RemoveFilter("author", "test-author")
	si.RemoveFilter("project", "current")

	filters = si.GetFilters()
	assert.NotContains(t, filters.Types, "pattern")
	assert.NotContains(t, filters.Tags, "authentication")
	assert.Equal(t, "", filters.Author)
	assert.False(t, filters.InCurrentProject)

	// Test clearing filters
	si.clearFilters()
	filters = si.GetFilters()
	assert.Empty(t, filters.Types)
	assert.Empty(t, filters.Tags)
	assert.Empty(t, filters.Sources)
	assert.Equal(t, "", filters.Author)
	assert.Equal(t, 0.5, filters.MinScore) // Default value
	assert.False(t, filters.InCurrentProject)
}

func TestSearchInterface_Navigation(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Set up test results
	results := make([]SearchResult, 5)
	for i := 0; i < 5; i++ {
		results[i] = SearchResult{
			ID:    string(rune('A' + i)),
			Title: "Result " + string(rune('A'+i)),
			Score: 1.0 - float64(i)*0.1,
		}
	}
	si.SetResults(results)

	// Test initial state
	assert.Equal(t, 0, si.selected)

	// Test moving down
	si.moveSelection(1)
	assert.Equal(t, 1, si.selected)

	si.moveSelection(1)
	assert.Equal(t, 2, si.selected)

	// Test moving up
	si.moveSelection(-1)
	assert.Equal(t, 1, si.selected)

	// Test wrap around - move past end
	si.selected = 4 // Last item
	si.moveSelection(1)
	assert.Equal(t, 0, si.selected) // Should wrap to first

	// Test wrap around - move before start
	si.selected = 0 // First item
	si.moveSelection(-1)
	assert.Equal(t, 4, si.selected) // Should wrap to last

	// Test navigation with empty results
	si.SetResults([]SearchResult{})
	si.moveSelection(1)
	assert.Equal(t, 0, si.selected) // Should not change
}

func TestSearchInterface_DetailsToggle(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// No results - toggle should not change anything
	si.toggleDetails()
	assert.False(t, si.showDetails)

	// With results
	results := []SearchResult{
		{ID: "1", Title: "Test Result"},
	}
	si.SetResults(results)

	// Toggle on
	si.toggleDetails()
	assert.True(t, si.showDetails)

	// Toggle off
	si.toggleDetails()
	assert.False(t, si.showDetails)
}

func TestSearchInterface_Rendering(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)
	si.SetSize(100, 30)

	// Test empty state
	view := si.View()
	assert.Contains(t, view, "Guild Corpus Search")
	assert.Contains(t, view, "Welcome to Guild Corpus Search")
	assert.Contains(t, view, "Getting Started")

	// Test with query but no results
	si.SetQuery("test query")
	view = si.View()
	assert.Contains(t, view, "test query")
	assert.Contains(t, view, "No results found")
	assert.Contains(t, view, "Suggestions")

	// Test with query and results
	results := []SearchResult{
		{
			ID:        "1",
			Title:     "Test Result",
			Path:      "test/path.md",
			Source:    "test",
			Tags:      []string{"test", "example"},
			Preview:   "This is a test result preview",
			Score:     0.95,
			Type:      "example",
			UpdatedAt: time.Now(),
			Metadata:  map[string]string{"category": "testing"},
		},
	}
	si.SetResults(results)

	view = si.View()
	assert.Contains(t, view, "Found 1 results")
	assert.Contains(t, view, "Test Result (95%)")
	assert.Contains(t, view, "test result preview")
	assert.Contains(t, view, "test/path.md")

	// Test with details expanded
	si.showDetails = true
	view = si.View()
	assert.Contains(t, view, "Full Details")
	assert.Contains(t, view, "Extended Preview")
	assert.Contains(t, view, "Metadata")
	assert.Contains(t, view, "category: testing")
}

func TestSearchInterface_ActiveFilters(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// No active filters
	assert.False(t, si.hasActiveFilters())

	// Add some filters
	si.AddFilter("type", "pattern")
	si.AddFilter("tag", "auth")
	si.filters.MinScore = 0.8

	assert.True(t, si.hasActiveFilters())

	// Test rendering active filters
	view := si.View()
	assert.Contains(t, view, "Active filters")
	assert.Contains(t, view, "types:pattern")
	assert.Contains(t, view, "tags:auth")
	assert.Contains(t, view, "score:>0.8")
}

func TestSearchInterface_HelperFunctions(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test truncate
	shortText := "Short text"
	assert.Equal(t, shortText, si.truncate(shortText, 20))

	longText := "This is a very long text that should be truncated at some point"
	truncated := si.truncate(longText, 20)
	assert.True(t, len(truncated) <= 20)
	assert.True(t, strings.HasSuffix(truncated, "..."))

	// Test formatTime
	now := time.Now()

	// Same day
	sameDay := now.Add(-2 * time.Hour)
	formatted := si.formatTime(sameDay)
	assert.Regexp(t, `^\d{2}:\d{2}$`, formatted) // HH:MM format

	// Different day, same year
	diffDay := now.Add(-48 * time.Hour)
	formatted = si.formatTime(diffDay)
	assert.Regexp(t, `^[A-Z][a-z]{2} \d{1,2}$`, formatted) // Jan 2 format

	// Different year
	diffYear := now.Add(-400 * 24 * time.Hour)
	formatted = si.formatTime(diffYear)
	assert.Regexp(t, `^[A-Z][a-z]{2} \d{1,2}, \d{4}$`, formatted) // Jan 2, 2006 format
}

func TestSearchInterface_InputMode(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test initial state
	assert.False(t, si.IsInputMode())

	// Test setting input mode
	si.SetInputMode(true)
	assert.True(t, si.IsInputMode())

	si.SetInputMode(false)
	assert.False(t, si.IsInputMode())
}

func TestSearchInterface_KeyHandling(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Set up test results
	results := []SearchResult{
		{ID: "1", Title: "Result 1"},
		{ID: "2", Title: "Result 2"},
		{ID: "3", Title: "Result 3"},
	}
	si.SetResults(results)

	// Test navigation keys (simulated)
	tests := []struct {
		name             string
		key              string
		expectedSelected int
		expectedDetails  bool
	}{
		{"move down", "j", 1, false},
		{"move down again", "j", 2, false},
		{"move up", "k", 1, false},
		{"toggle details", "space", 1, true},
		{"toggle details off", "space", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate key handling (we can't actually send tea.KeyMsg in unit tests easily)
			switch tt.key {
			case "j":
				si.moveSelection(1)
			case "k":
				si.moveSelection(-1)
			case "space":
				si.toggleDetails()
			}

			assert.Equal(t, tt.expectedSelected, si.selected)
			assert.Equal(t, tt.expectedDetails, si.showDetails)
		})
	}
}

func TestSearchInterface_Update(t *testing.T) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test window size message
	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	model, cmd := si.Update(sizeMsg)

	updatedSI, ok := model.(*SearchInterface)
	require.True(t, ok)
	assert.Equal(t, 120, updatedSI.width)
	assert.Equal(t, 40, updatedSI.height)
	assert.Nil(t, cmd)

	// Test search results message
	results := []SearchResult{
		{ID: "1", Title: "New Result"},
	}
	resultsMsg := SearchResultsMsg{
		Query:   "test",
		Results: results,
		Total:   1,
	}

	model, cmd = si.Update(resultsMsg)
	updatedSI, ok = model.(*SearchInterface)
	require.True(t, ok)
	assert.Len(t, updatedSI.results, 1)
	assert.Equal(t, "New Result", updatedSI.results[0].Title)
	assert.Nil(t, cmd)

	// Test error in search results
	errorMsg := SearchResultsMsg{
		Query: "test",
		Error: assert.AnError,
	}

	model, cmd = si.Update(errorMsg)
	_, ok = model.(*SearchInterface)
	require.True(t, ok)
	assert.Nil(t, cmd)
}

func TestSearchInterface_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Test complete workflow

	// 1. Set initial query
	si.SetQuery("authentication")
	assert.Equal(t, "authentication", si.GetQuery())

	// 2. Add filters
	err := si.AddFilter("type", "guide")
	require.NoError(t, err)

	err = si.AddFilter("tag", "security")
	require.NoError(t, err)

	// 3. Set results
	results := []SearchResult{
		{
			ID:        "auth-guide-1",
			Title:     "Authentication Best Practices",
			Path:      "guides/auth.md",
			Source:    "documentation",
			Tags:      []string{"authentication", "security", "guide"},
			Preview:   "This guide covers authentication patterns and security considerations...",
			Score:     0.95,
			Type:      "guide",
			UpdatedAt: time.Now(),
			Metadata: map[string]string{
				"author":   "security-team",
				"version":  "2.1",
				"reviewed": "2025-01-15",
			},
		},
		{
			ID:        "auth-example-1",
			Title:     "JWT Implementation Example",
			Path:      "examples/jwt.md",
			Source:    "code-examples",
			Tags:      []string{"authentication", "jwt", "example"},
			Preview:   "Example implementation of JWT authentication in Go...",
			Score:     0.88,
			Type:      "example",
			UpdatedAt: time.Now().Add(-time.Hour * 2),
		},
	}

	si.SetResults(results)

	// 4. Test navigation
	assert.Equal(t, 0, si.selected)
	selected := si.GetSelectedResult()
	require.NotNil(t, selected)
	assert.Equal(t, "auth-guide-1", selected.ID)

	si.moveSelection(1)
	assert.Equal(t, 1, si.selected)
	selected = si.GetSelectedResult()
	require.NotNil(t, selected)
	assert.Equal(t, "auth-example-1", selected.ID)

	// 5. Test details view - move back to first result to check metadata
	si.moveSelection(-1) // Move back to first result
	assert.Equal(t, 0, si.selected)
	si.toggleDetails()
	assert.True(t, si.showDetails)

	// 6. Test rendering with all features
	view := si.View()

	// Should contain query
	assert.Contains(t, view, "authentication")

	// Should contain active filters
	assert.Contains(t, view, "Active filters")
	assert.Contains(t, view, "types:guide")
	assert.Contains(t, view, "tags:security")

	// Should contain results
	assert.Contains(t, view, "Found 2 results")
	assert.Contains(t, view, "Authentication Best Practices")
	assert.Contains(t, view, "JWT Implementation Example")

	// Should contain scores
	assert.Contains(t, view, "95%")
	assert.Contains(t, view, "88%")

	// Should contain detailed view
	assert.Contains(t, view, "Full Details")
	assert.Contains(t, view, "author: security-team")

	// 7. Test clearing filters
	si.clearFilters()
	filters := si.GetFilters()
	assert.Empty(t, filters.Types)
	assert.Empty(t, filters.Tags)
	assert.False(t, si.hasActiveFilters())
}

func BenchmarkSearchInterface_Rendering(b *testing.B) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)
	si.SetSize(120, 40)

	// Create many results
	results := make([]SearchResult, 100)
	for i := 0; i < 100; i++ {
		results[i] = SearchResult{
			ID:        string(rune('A' + (i % 26))),
			Title:     "Test Result " + string(rune('A'+(i%26))),
			Path:      "test/path" + string(rune('A'+(i%26))) + ".md",
			Source:    "test",
			Tags:      []string{"test", "benchmark"},
			Preview:   "This is a test result preview that contains some content to render",
			Score:     1.0 - float64(i)*0.01,
			Type:      "test",
			UpdatedAt: time.Now(),
		}
	}

	si.SetResults(results)
	si.SetQuery("test query")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = si.View()
	}
}

func BenchmarkSearchInterface_Navigation(b *testing.B) {
	ctx := context.Background()
	si := NewSearchInterface(ctx)

	// Create many results
	results := make([]SearchResult, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = SearchResult{
			ID:    string(rune('A' + (i % 26))),
			Title: "Result " + string(rune('A'+(i%26))),
			Score: 1.0 - float64(i)*0.001,
		}
	}

	si.SetResults(results)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate navigation
		si.moveSelection(1)
		if si.selected >= len(results)-1 {
			si.selected = 0 // Reset to start
		}
	}
}
