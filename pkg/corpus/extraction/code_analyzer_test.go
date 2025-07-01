// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCraftCodeAnalyzer tests the creation of a new code analyzer
func TestCraftCodeAnalyzer(t *testing.T) {
	ctx := context.Background()
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-repo-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Initialize a git repository in the temp directory
	err = exec.Command("git", "init", tempDir).Run()
	require.NoError(t, err)
	
	analyzer, err := NewCodeAnalyzer(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.gitClient)
	assert.NotNil(t, analyzer.codeParser)
	assert.NotNil(t, analyzer.diffAnalyzer)
	assert.NotNil(t, analyzer.patternDetector)
}

// TestJourneymanCodeAnalyzerContextCancellation tests context cancellation handling
func TestJourneymanCodeAnalyzerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	// Should handle cancelled context gracefully
	analyzer, err := NewCodeAnalyzer(ctx, "/tmp/test-repo")
	assert.Error(t, err)
	assert.Nil(t, analyzer)
}