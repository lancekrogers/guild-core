// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentScanner_New(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		opts        []ScannerOption
		expectError bool
	}{
		{
			name:        "Empty base path",
			basePath:    "",
			expectError: true,
		},
		{
			name:        "Non-existent path",
			basePath:    "/non/existent/path",
			expectError: true,
		},
		{
			name:        "Valid path",
			basePath:    os.TempDir(),
			expectError: false,
		},
		{
			name:     "With custom workers",
			basePath: os.TempDir(),
			opts:     []ScannerOption{WithWorkers(8)},
		},
		{
			name:     "With custom patterns",
			basePath: os.TempDir(),
			opts: []ScannerOption{
				WithFilePatterns([]string{"*.txt", "*.doc"}),
				WithIgnorePatterns([]string{"temp_*", "cache_*"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewDocumentScanner(tt.basePath, tt.opts...)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, scanner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scanner)
			}
		})
	}
}

func TestDocumentScanner_ContentTypeDetection(t *testing.T) {
	scanner, err := NewDocumentScanner(os.TempDir())
	require.NoError(t, err)

	tests := []struct {
		path     string
		expected ContentType
	}{
		{"file.md", ContentTypeMarkdown},
		{"file.markdown", ContentTypeMarkdown},
		{"file.yaml", ContentTypeYAML},
		{"file.yml", ContentTypeYAML},
		{"file.go", ContentTypeGo},
		{"file.json", ContentTypeJSON},
		{"file.txt", ContentTypeText},
		{"file.unknown", ContentTypeUnknown},
		{"FILE.MD", ContentTypeMarkdown}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := scanner.detectContentType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocumentScanner_ChecksumCalculation(t *testing.T) {
	scanner, err := NewDocumentScanner(os.TempDir())
	require.NoError(t, err)

	content1 := []byte("Hello, World!")
	content2 := []byte("Hello, World!")
	content3 := []byte("Different content")

	checksum1 := scanner.calculateChecksum(content1)
	checksum2 := scanner.calculateChecksum(content2)
	checksum3 := scanner.calculateChecksum(content3)

	// Same content should produce same checksum
	assert.Equal(t, checksum1, checksum2)
	// Different content should produce different checksum
	assert.NotEqual(t, checksum1, checksum3)
	// Checksum should be hex encoded SHA256 (64 characters)
	assert.Len(t, checksum1, 64)
}

func TestDocumentScanner_MetadataExtraction(t *testing.T) {
	ctx := context.Background()
	scanner, err := NewDocumentScanner(os.TempDir())
	require.NoError(t, err)

	t.Run("Markdown metadata", func(t *testing.T) {
		content := []byte(`---
title: Test Document
description: A test document
tags: [test, example]
---

# Main Heading

This is a test document with some content.

## Subheading

- TODO: Complete this section
- Another item

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

Check out [this link](https://example.com) and [[WikiLink]].

#golang #testing`)

		info := &mockFileInfo{
			name:    "test.md",
			size:    int64(len(content)),
			modTime: time.Now(),
		}

		metadata := scanner.extractMetadata(ctx, content, ContentTypeMarkdown, info)

		assert.Equal(t, "Test Document", metadata.Title)
		assert.Equal(t, "A test document", metadata.Description)
		assert.Equal(t, 2, metadata.HeadingCount)
		assert.Equal(t, 1, metadata.CodeBlockCount)
		assert.Equal(t, 1, metadata.TODOCount)
		assert.Equal(t, 2, metadata.LinkCount) // One URL, one WikiLink
		assert.Contains(t, metadata.ExtractedTags, "golang")
		assert.Contains(t, metadata.ExtractedTags, "testing")
		assert.True(t, metadata.WordCount > 10)
	})

	t.Run("Go metadata", func(t *testing.T) {
		content := []byte(`package main

import "fmt"

// TODO: Add error handling
func main() {
    fmt.Println("Hello, World!")
    // FIXME: This needs improvement
}`)

		info := &mockFileInfo{
			name:    "main.go",
			size:    int64(len(content)),
			modTime: time.Now(),
		}

		metadata := scanner.extractMetadata(ctx, content, ContentTypeGo, info)

		assert.Equal(t, "main", metadata.Title)
		assert.Equal(t, "go", metadata.Language)
		assert.Equal(t, 2, metadata.TODOCount) // TODO and FIXME
	})
}

func TestDocumentScanner_Scan(t *testing.T) {
	ctx := context.Background()

	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"doc1.md": `# Document 1
This is the first document.`,
		"doc2.md": `# Document 2
This is the second document with a TODO item.`,
		"code.go": `package main

func main() {}`,
		"ignore.tmp": "Should be ignored",
		"data.json":  `{"key": "value"}`,
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory to test
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	subFile := filepath.Join(subDir, "sub.md")
	err = os.WriteFile(subFile, []byte("# Subdirectory Document"), 0644)
	require.NoError(t, err)

	// Create scanner and scan
	scanner, err := NewDocumentScanner(tmpDir)
	require.NoError(t, err)

	documents, err := scanner.Scan(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, documents)

	// Verify results
	assert.Len(t, documents, 4) // Should not include .tmp file

	// Check document properties
	foundTypes := make(map[ContentType]bool)
	for _, doc := range documents {
		assert.NotEmpty(t, doc.ID)
		assert.NotEmpty(t, doc.Path)
		assert.NotEmpty(t, doc.Checksum)
		assert.NotZero(t, doc.LastModified)
		foundTypes[doc.Type] = true

		// Verify content was read
		assert.NotEmpty(t, doc.Content)
		assert.Contains(t, testFiles, filepath.Base(doc.Path))
	}

	// Should have found different content types
	assert.True(t, foundTypes[ContentTypeMarkdown])
	assert.True(t, foundTypes[ContentTypeGo])
	assert.True(t, foundTypes[ContentTypeJSON])
}

func TestDocumentScanner_StreamProcessing(t *testing.T) {
	ctx := context.Background()

	// Create temporary test directory with many files
	tmpDir := t.TempDir()

	// Create 20 test files
	for i := 0; i < 20; i++ {
		name := filepath.Join(tmpDir, fmt.Sprintf("doc%d.md", i))
		content := fmt.Sprintf("# Document %d\nContent for document %d", i, i)
		err := os.WriteFile(name, []byte(content), 0644)
		require.NoError(t, err)
	}

	scanner, err := NewDocumentScanner(tmpDir, WithWorkers(2))
	require.NoError(t, err)

	// Use streaming API
	results, err := scanner.ScanStream(ctx)
	require.NoError(t, err)

	var documents []*ScannedDocument
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			documents = append(documents, result.Document)
		}
	}

	assert.Empty(t, errors)
	assert.Len(t, documents, 20)
}

func TestDocumentScanner_ChangeDetection(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	scanner, err := NewDocumentScanner(tmpDir)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.md")
	content1 := []byte("Original content")
	err = os.WriteFile(testFile, content1, 0644)
	require.NoError(t, err)

	// Get initial checksum
	checksum1, err := scanner.GetChecksum(ctx, testFile)
	require.NoError(t, err)

	// Check no change
	changed, err := scanner.HasFileChanged(ctx, testFile, checksum1)
	require.NoError(t, err)
	assert.False(t, changed)

	// Modify file
	content2 := []byte("Modified content")
	err = os.WriteFile(testFile, content2, 0644)
	require.NoError(t, err)

	// Check change detected
	changed, err = scanner.HasFileChanged(ctx, testFile, checksum1)
	require.NoError(t, err)
	assert.True(t, changed)
}

func TestDocumentScanner_IgnorePatterns(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create test structure
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	nodeDir := filepath.Join(tmpDir, "node_modules")
	err = os.MkdirAll(nodeDir, 0755)
	require.NoError(t, err)

	// Create files in ignored directories
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte("{}"), 0644)
	require.NoError(t, err)

	// Create normal file
	err = os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644)
	require.NoError(t, err)

	scanner, err := NewDocumentScanner(tmpDir)
	require.NoError(t, err)

	documents, err := scanner.Scan(ctx)
	require.NoError(t, err)

	// Should only find the readme file
	assert.Len(t, documents, 1)
	assert.Equal(t, "readme.md", filepath.Base(documents[0].Path))
}

// mockFileInfo implements fs.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// Helper function
func TestHelpers(t *testing.T) {
	t.Run("Word counting", func(t *testing.T) {
		tests := []struct {
			text     string
			expected int
		}{
			{"Hello world", 2},
			{"  Multiple   spaces  ", 2},
			{"One", 1},
			{"", 0},
			{"Line\nbreak\nwords", 3},
		}

		for _, tt := range tests {
			count := countWords(tt.text)
			assert.Equal(t, tt.expected, count)
		}
	})

	t.Run("Hashtag extraction", func(t *testing.T) {
		text := "This has #golang and #testing tags. Also #golang again and #Testing!"
		tags := extractHashtags(text)

		assert.Len(t, tags, 3) // Should deduplicate
		assert.Contains(t, tags, "golang")
		assert.Contains(t, tags, "testing")
		assert.Contains(t, tags, "Testing") // Case preserved
	})

	t.Run("Document ID generation", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"docs/guide.md", "docs/guide"},
			{"my file.txt", "my_file"},
			{"path/to/file.go", "path/to/file"},
		}

		for _, tt := range tests {
			id := generateDocumentID(tt.path)
			assert.Equal(t, tt.expected, id)
		}
	})
}