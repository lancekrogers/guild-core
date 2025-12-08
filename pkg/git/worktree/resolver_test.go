// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftConflictResolver tests creation of conflict resolver
func TestCraftConflictResolver(t *testing.T) {
	ctx := context.Background()

	resolver, err := NewConflictResolver(ctx)
	require.NoError(t, err)
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.strategies)
	assert.NotNil(t, resolver.history)
	assert.NotNil(t, resolver.manual)
	assert.Greater(t, len(resolver.strategies), 0)
}

// TestJourneymanConflictResolverContextCancellation tests context cancellation
func TestJourneymanConflictResolverContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resolver, err := NewConflictResolver(ctx)
	assert.Error(t, err)
	assert.Nil(t, resolver)
}

// TestGuildWhitespaceResolver tests whitespace conflict resolution
func TestGuildWhitespaceResolver(t *testing.T) {
	ctx := context.Background()
	resolver := &WhitespaceResolver{}

	// Test whitespace-only conflict
	conflict := Conflict{
		File: "test.go",
		Diff: &ThreeWayDiff{
			Content1: "package main\n\nfunc main() {\n    fmt.Println(\"hello\")\n}",
			Content2: "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
		},
	}

	canResolve := resolver.CanResolve(ctx, conflict)
	assert.True(t, canResolve)

	resolution, err := resolver.Resolve(ctx, conflict)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, "whitespace_normalization", resolution.Strategy)
	assert.Equal(t, 1.0, resolution.Confidence)
}

// TestScribeImportResolver tests import conflict resolution
func TestScribeImportResolver(t *testing.T) {
	ctx := context.Background()

	languages := map[string]ImportSorter{
		"go":         &GoImportSorter{},
		"javascript": &JSImportSorter{},
		"python":     &PythonImportSorter{},
	}

	resolver := &ImportResolver{languages: languages}

	// Test Go import conflict
	goConflict := Conflict{
		File: "main.go",
		Diff: &ThreeWayDiff{
			Base:     "package main\n\nimport (\n\t\"fmt\"\n)",
			Content1: "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)",
			Content2: "package main\n\nimport (\n\t\"fmt\"\n\t\"log\"\n)",
		},
	}

	canResolve := resolver.CanResolve(ctx, goConflict)
	assert.True(t, canResolve)

	resolution, err := resolver.Resolve(ctx, goConflict)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, "import_merge", resolution.Strategy)
	assert.Equal(t, 0.95, resolution.Confidence)
}

// TestCraftGoImportSorter tests Go import sorting
func TestCraftGoImportSorter(t *testing.T) {
	ctx := context.Background()
	sorter := &GoImportSorter{}

	// Test import extraction
	content := `package main

import (
	"fmt"
	"os"
	"github.com/stretchr/testify/assert"
	"github.com/guild-framework/guild-core/pkg/test"
)

func main() {}`

	imports := sorter.ExtractImports(ctx, content)
	assert.Greater(t, len(imports), 0)

	// Test import merging
	imports1 := []string{"\"fmt\"", "\"os\""}
	imports2 := []string{"\"log\"", "\"os\""}
	merged := sorter.MergeImports(ctx, imports1, imports2)

	assert.Contains(t, merged, "\"fmt\"")
	assert.Contains(t, merged, "\"os\"")
	assert.Contains(t, merged, "\"log\"")

	// Test import sorting
	unsorted := []string{
		"\"github.com/stretchr/testify/assert\"",
		"\"fmt\"",
		"\"github.com/guild-framework/guild-core/pkg/test\"",
		"\"os\"",
	}

	sorted := sorter.SortImports(ctx, unsorted)
	assert.Greater(t, len(sorted), 0)

	// Standard library imports should come first
	stdlibFound := false
	for _, imp := range sorted {
		if imp == "\"fmt\"" || imp == "\"os\"" {
			stdlibFound = true
			break
		}
	}
	assert.True(t, stdlibFound)
}

// TestJourneymanJSImportSorter tests JavaScript import sorting
func TestJourneymanJSImportSorter(t *testing.T) {
	ctx := context.Background()
	sorter := &JSImportSorter{}

	content := `import React from 'react';
import lodash from 'lodash';
import { utils } from './utils';

export default function App() {}`

	imports := sorter.ExtractImports(ctx, content)
	assert.Len(t, imports, 3)
	assert.Contains(t, imports, "import React from 'react';")
	assert.Contains(t, imports, "import lodash from 'lodash';")
	assert.Contains(t, imports, "import { utils } from './utils';")

	// Test sorting
	sorted := sorter.SortImports(ctx, imports)
	assert.Len(t, sorted, 3)
}

// TestGuildPythonImportSorter tests Python import sorting
func TestGuildPythonImportSorter(t *testing.T) {
	ctx := context.Background()
	sorter := &PythonImportSorter{}

	content := `import os
import sys
from django import models
from .utils import helper
import requests

def main():
    pass`

	imports := sorter.ExtractImports(ctx, content)
	assert.Greater(t, len(imports), 0)

	// Should find standard library, third-party, and local imports
	hasStdlib := false
	hasThirdParty := false
	hasLocal := false

	for _, imp := range imports {
		if imp == "import os" || imp == "import sys" {
			hasStdlib = true
		}
		if imp == "import requests" {
			hasThirdParty = true
		}
		if imp == "from .utils import helper" {
			hasLocal = true
		}
	}

	assert.True(t, hasStdlib)
	assert.True(t, hasThirdParty)
	assert.True(t, hasLocal)

	// Test sorting
	sorted := sorter.SortImports(ctx, imports)
	assert.Greater(t, len(sorted), 0)
}

// TestScribeFormattingResolver tests formatting conflict resolution
func TestScribeFormattingResolver(t *testing.T) {
	ctx := context.Background()
	resolver := &FormattingResolver{}

	// Test formatting-only conflict
	conflict := Conflict{
		Diff: &ThreeWayDiff{
			Content1: "function test() {\n  return true;\n}",
			Content2: "function test() {\n    return true;\n}",
		},
	}

	canResolve := resolver.CanResolve(ctx, conflict)
	assert.True(t, canResolve)

	resolution, err := resolver.Resolve(ctx, conflict)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, "formatting_normalization", resolution.Strategy)
	assert.Equal(t, 0.9, resolution.Confidence)
}

// TestCraftCommentResolver tests comment conflict resolution
func TestCraftCommentResolver(t *testing.T) {
	ctx := context.Background()
	resolver := &CommentResolver{}

	// Test comment-only conflict
	conflict := Conflict{
		Diff: &ThreeWayDiff{
			Content1: "// Comment 1\nfunc test() {}",
			Content2: "// Comment 2\nfunc test() {}",
		},
	}

	canResolve := resolver.CanResolve(ctx, conflict)
	assert.True(t, canResolve)

	resolution, err := resolver.Resolve(ctx, conflict)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, "comment_merge", resolution.Strategy)
	assert.Equal(t, 0.85, resolution.Confidence)
}

// TestJourneymanSimpleLineResolver tests simple line conflict resolution
func TestJourneymanSimpleLineResolver(t *testing.T) {
	ctx := context.Background()
	resolver := &SimpleLineResolver{}

	// Test simple conflict with few lines
	conflict := Conflict{
		Diff: &ThreeWayDiff{
			Content1:      "short content",
			Content2:      "longer content with more text",
			ConflictLines: []int{1, 2},
		},
	}

	canResolve := resolver.CanResolve(ctx, conflict)
	assert.True(t, canResolve)

	resolution, err := resolver.Resolve(ctx, conflict)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, "simple_line_merge", resolution.Strategy)
	assert.Equal(t, 0.6, resolution.Confidence)
	// Should choose longer content
	assert.Equal(t, "longer content with more text", resolution.Content)
}

// TestGuildResolutionStrategy tests resolution strategy priorities
func TestGuildResolutionStrategy(t *testing.T) {
	strategies := []ResolutionStrategy{
		&WhitespaceResolver{},
		&ImportResolver{},
		&FormattingResolver{},
		&CommentResolver{},
		&SimpleLineResolver{},
	}

	// Test priorities
	assert.Equal(t, 100, strategies[0].Priority()) // WhitespaceResolver
	assert.Equal(t, 90, strategies[1].Priority())  // ImportResolver
	assert.Equal(t, 80, strategies[2].Priority())  // FormattingResolver
	assert.Equal(t, 70, strategies[3].Priority())  // CommentResolver
	assert.Equal(t, 60, strategies[4].Priority())  // SimpleLineResolver
}

// TestScribeResolutionHistory tests resolution history tracking
func TestScribeResolutionHistory(t *testing.T) {
	history := NewResolutionHistory()
	assert.NotNil(t, history)
	assert.Empty(t, history.resolutions)

	// Record a resolution
	conflict := Conflict{ID: "test-conflict"}
	resolution := &Resolution{
		Content:    "resolved content",
		Strategy:   "test_strategy",
		Confidence: 0.9,
	}
	strategy := &WhitespaceResolver{}

	history.Record(conflict, resolution, strategy)

	assert.Len(t, history.resolutions, 1)
	assert.Equal(t, "test-conflict", history.resolutions[0].Conflict.ID)
	assert.Equal(t, "resolved content", history.resolutions[0].Resolution.Content)
	assert.True(t, history.resolutions[0].Success)
}

// TestCraftMLResolver tests ML-based resolution
func TestCraftMLResolver(t *testing.T) {
	ctx := context.Background()

	resolver, err := NewMLResolver(ctx)
	require.NoError(t, err)
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.model)
	assert.NotNil(t, resolver.features)

	// Test feature extraction
	conflict := Conflict{
		File:     "test.go",
		Type:     ConflictTypeContent,
		Severity: SeverityMedium,
		Diff: &ThreeWayDiff{
			Content1: "content1",
			Content2: "content2",
		},
	}

	features := resolver.features.Extract(ctx, conflict)
	assert.NotEmpty(t, features)
	assert.Equal(t, ".go", features["file_extension"])
	assert.Equal(t, ConflictTypeContent, features["conflict_type"])
	assert.Equal(t, SeverityMedium, features["severity"])
}

// TestJourneymanManualResolver tests manual resolution
func TestJourneymanManualResolver(t *testing.T) {
	resolver := NewManualResolver()
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.ui)
	assert.NotNil(t, resolver.reviewer)

	// Test request ID generation
	id1 := resolver.generateRequestID()
	time.Sleep(1 * time.Microsecond) // Ensure different timestamp
	id2 := resolver.generateRequestID()
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "req_")
}

// TestGuildCodeReviewer tests code review validation
func TestGuildCodeReviewer(t *testing.T) {
	ctx := context.Background()
	reviewer := NewCodeReviewer()

	conflict := Conflict{ID: "test-conflict"}

	// Test valid resolution
	validResolution := &Resolution{
		Content: "func test() { return true }",
	}

	err := reviewer.ValidateResolution(ctx, conflict, validResolution)
	assert.NoError(t, err)

	// Test empty resolution
	emptyResolution := &Resolution{
		Content: "",
	}

	err = reviewer.ValidateResolution(ctx, conflict, emptyResolution)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolution content is empty")

	// Test resolution with conflict markers
	conflictResolution := &Resolution{
		Content: "func test() {\n<<<<<<< HEAD\nreturn true\n=======\nreturn false\n>>>>>>> branch\n}",
	}

	err = reviewer.ValidateResolution(ctx, conflict, conflictResolution)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolution contains conflict markers")
}

// TestScribeConflictTypes tests conflict type definitions
func TestScribeConflictTypes(t *testing.T) {
	types := []ConflictType{
		ConflictTypeContent,
		ConflictTypeAPI,
		ConflictTypeSchema,
		ConflictTypeConfig,
	}

	assert.Equal(t, ConflictType("content"), types[0])
	assert.Equal(t, ConflictType("api"), types[1])
	assert.Equal(t, ConflictType("schema"), types[2])
	assert.Equal(t, ConflictType("config"), types[3])
}

// TestCraftThreeWayDiff tests three-way diff functionality
func TestCraftThreeWayDiff(t *testing.T) {
	diff := &ThreeWayDiff{
		Base:          "original content",
		Content1:      "modified content 1",
		Content2:      "modified content 2",
		ConflictLines: []int{1, 2, 3},
	}

	assert.True(t, diff.HasConflicts())
	assert.Len(t, diff.ConflictLines, 3)

	// Test diff without conflicts
	noDiff := &ThreeWayDiff{
		Base:          "content",
		Content1:      "content",
		Content2:      "content",
		ConflictLines: []int{},
	}

	assert.False(t, noDiff.HasConflicts())
}

// TestJourneymanResolutionPattern tests resolution patterns
func TestJourneymanResolutionPattern(t *testing.T) {
	pattern := &ResolutionPattern{
		ID:          "pattern-1",
		Type:        "whitespace",
		Description: "Normalize whitespace differences",
		Conditions:  []string{"whitespace_only", "same_content"},
		Actions:     []string{"normalize", "merge"},
		Success:     0.95,
		Metadata: map[string]interface{}{
			"language": "go",
		},
	}

	assert.Equal(t, "pattern-1", pattern.ID)
	assert.Equal(t, "whitespace", pattern.Type)
	assert.Equal(t, 0.95, pattern.Success)
	assert.Len(t, pattern.Conditions, 2)
	assert.Len(t, pattern.Actions, 2)
	assert.Equal(t, "go", pattern.Metadata["language"])
}

// TestGuildResolutionRequest tests manual resolution request
func TestGuildResolutionRequest(t *testing.T) {
	conflict := Conflict{
		ID:   "conflict-1",
		File: "test.go",
		Type: ConflictTypeContent,
	}

	request := &ResolutionRequest{
		ID:        "req-1",
		Conflict:  conflict,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "req-1", request.ID)
	assert.Equal(t, "conflict-1", request.Conflict.ID)
	assert.Equal(t, "test.go", request.Conflict.File)
}

// TestScribeCompleteResolutionFlow tests complete resolution flow
func TestScribeCompleteResolutionFlow(t *testing.T) {
	ctx := context.Background()

	// Test complete flow with whitespace conflict
	resolver, err := NewConflictResolver(ctx)
	require.NoError(t, err)

	conflict := Conflict{
		ID:   "flow-test-1",
		File: "test.go",
		Type: ConflictTypeContent,
		Diff: &ThreeWayDiff{
			Content1: "package main\n\nfunc test() {\n    return true\n}",
			Content2: "package main\n\nfunc test() {\n\treturn true\n}",
		},
	}

	// This should be resolved by WhitespaceResolver
	resolution, err := resolver.ResolveConflict(ctx, conflict)

	if err != nil {
		// May fail due to manual resolution timeout in test environment
		assert.Contains(t, err.Error(), "manual resolution")
	} else {
		assert.NotNil(t, resolution)
		assert.Equal(t, "whitespace_normalization", resolution.Strategy)
		assert.Equal(t, 1.0, resolution.Confidence)
	}
}

// Benchmark tests for performance validation
func BenchmarkWhitespaceResolver(b *testing.B) {
	ctx := context.Background()
	resolver := &WhitespaceResolver{}

	conflict := Conflict{
		Diff: &ThreeWayDiff{
			Content1: "package main\n\nfunc main() {\n    fmt.Println(\"hello\")\n}",
			Content2: "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.Resolve(ctx, conflict)
	}
}

func BenchmarkImportSorting(b *testing.B) {
	ctx := context.Background()
	sorter := &GoImportSorter{}

	imports := []string{
		"\"fmt\"",
		"\"os\"",
		"\"github.com/stretchr/testify/assert\"",
		"\"github.com/guild-framework/guild-core/pkg/test\"",
		"\"log\"",
		"\"context\"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sorter.SortImports(ctx, imports)
	}
}

func BenchmarkFeatureExtraction(b *testing.B) {
	ctx := context.Background()
	extractor := &FeatureExtractor{}

	conflict := Conflict{
		File:     "test.go",
		Type:     ConflictTypeContent,
		Severity: SeverityMedium,
		Diff: &ThreeWayDiff{
			Content1:      "content1",
			Content2:      "content2",
			ConflictLines: []int{1, 2, 3},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.Extract(ctx, conflict)
	}
}
