// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"gopkg.in/yaml.v3"
)

// ContentType represents the type of content in a document
type ContentType string

const (
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypeYAML     ContentType = "yaml"
	ContentTypeGo       ContentType = "go"
	ContentTypeJSON     ContentType = "json"
	ContentTypeText     ContentType = "text"
	ContentTypeUnknown  ContentType = "unknown"
)

// DocumentMetadata contains extracted metadata from a document
type DocumentMetadata struct {
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	Language        string    `json:"language,omitempty"`
	WordCount       int       `json:"word_count"`
	CodeBlockCount  int       `json:"code_block_count"`
	LinkCount       int       `json:"link_count"`
	LastModified    time.Time `json:"last_modified"`
	FileSize        int64     `json:"file_size"`
	Checksum        string    `json:"checksum"`
	ExtractedTags   []string  `json:"extracted_tags,omitempty"`
	HeadingCount    int       `json:"heading_count"`
	TODOCount       int       `json:"todo_count"`
}

// ScannedDocument represents a document discovered by the scanner
type ScannedDocument struct {
	ID           string
	Path         string
	Type         ContentType
	Content      string
	Metadata     DocumentMetadata
	LastModified time.Time
	Checksum     string
}

// ScanResult contains the results of a document scan
type ScanResult struct {
	Document *ScannedDocument
	Error    error
}

// DocumentScanner discovers and analyzes documents in the filesystem
type DocumentScanner struct {
	basePath       string
	filePatterns   []string
	ignorePatterns []string
	contentTypes   map[string]ContentType
	workers        int
	mu             sync.RWMutex
}

// ScannerOption configures the document scanner
type ScannerOption func(*DocumentScanner)

// WithFilePatterns sets the file patterns to scan
func WithFilePatterns(patterns []string) ScannerOption {
	return func(s *DocumentScanner) {
		s.filePatterns = patterns
	}
}

// WithIgnorePatterns sets patterns to ignore during scanning
func WithIgnorePatterns(patterns []string) ScannerOption {
	return func(s *DocumentScanner) {
		s.ignorePatterns = patterns
	}
}

// WithWorkers sets the number of concurrent workers
func WithWorkers(count int) ScannerOption {
	return func(s *DocumentScanner) {
		if count > 0 {
			s.workers = count
		}
	}
}

// NewDocumentScanner creates a new document scanner
func NewDocumentScanner(basePath string, opts ...ScannerOption) (*DocumentScanner, error) {
	if basePath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "base path cannot be empty", nil).
			WithComponent("corpus.scanner").
			WithOperation("NewDocumentScanner")
	}

	// Verify base path exists
	if _, err := os.Stat(basePath); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "invalid base path").
			WithComponent("corpus.scanner").
			WithOperation("NewDocumentScanner").
			WithDetails("path", basePath)
	}

	scanner := &DocumentScanner{
		basePath: basePath,
		filePatterns: []string{
			"*.md", "*.markdown",
			"*.yaml", "*.yml",
			"*.go",
			"*.json",
			"*.txt",
		},
		ignorePatterns: []string{
			".git", "node_modules", "vendor", "dist", "build",
			"*.tmp", "*.temp", "*.log", "*.cache",
			".DS_Store", "Thumbs.db",
		},
		contentTypes: map[string]ContentType{
			".md":       ContentTypeMarkdown,
			".markdown": ContentTypeMarkdown,
			".yaml":     ContentTypeYAML,
			".yml":      ContentTypeYAML,
			".go":       ContentTypeGo,
			".json":     ContentTypeJSON,
			".txt":      ContentTypeText,
		},
		workers: 4, // Default worker count
	}

	// Apply options
	for _, opt := range opts {
		opt(scanner)
	}

	return scanner, nil
}

// ScanStream returns a channel of scan results for streaming processing
func (s *DocumentScanner) ScanStream(ctx context.Context) (<-chan ScanResult, error) {
	results := make(chan ScanResult, s.workers*2)

	// Create file path channel
	paths := make(chan string, 100)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanWorker(ctx, paths, results)
		}()
	}

	// Start path discovery in background
	go func() {
		defer close(paths)
		_ = s.discoverFiles(ctx, paths)
	}()

	// Close results channel when all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}

// Scan performs a complete scan and returns all documents
func (s *DocumentScanner) Scan(ctx context.Context) ([]*ScannedDocument, error) {
	results, err := s.ScanStream(ctx)
	if err != nil {
		return nil, err
	}

	var documents []*ScannedDocument
	var scanErrors []error

	for result := range results {
		if result.Error != nil {
			scanErrors = append(scanErrors, result.Error)
			continue
		}
		documents = append(documents, result.Document)
	}

	// If there were errors, return them aggregated
	if len(scanErrors) > 0 {
		return documents, gerror.New(gerror.ErrCodeInternal, 
			fmt.Sprintf("scan completed with %d errors", len(scanErrors)), nil).
			WithComponent("corpus.scanner").
			WithOperation("Scan").
			WithDetails("error_count", len(scanErrors))
	}

	return documents, nil
}

// scanWorker processes files from the path channel
func (s *DocumentScanner) scanWorker(ctx context.Context, paths <-chan string, results chan<- ScanResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case path, ok := <-paths:
			if !ok {
				return
			}

			doc, err := s.scanFile(ctx, path)
			if err != nil {
				results <- ScanResult{Error: err}
			} else if doc != nil {
				results <- ScanResult{Document: doc}
			}
		}
	}
}

// discoverFiles walks the filesystem and sends file paths to the channel
func (s *DocumentScanner) discoverFiles(ctx context.Context, paths chan<- string) error {
	return filepath.WalkDir(s.basePath, func(path string, d fs.DirEntry, err error) error {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories
		if d.IsDir() {
			// Check if directory should be ignored
			for _, pattern := range s.ignorePatterns {
				if matched, _ := filepath.Match(pattern, d.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches patterns
		if !s.shouldScanFile(path) {
			return nil
		}

		// Send path to workers
		select {
		case paths <- path:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}

// shouldScanFile checks if a file should be scanned
func (s *DocumentScanner) shouldScanFile(path string) bool {
	base := filepath.Base(path)

	// Check ignore patterns
	for _, pattern := range s.ignorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return false
		}
		// Also check if the path contains ignored directories
		if strings.Contains(path, string(filepath.Separator)+pattern+string(filepath.Separator)) {
			return false
		}
	}

	// Check file patterns
	for _, pattern := range s.filePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}

	return false
}

// scanFile processes a single file
func (s *DocumentScanner) scanFile(ctx context.Context, path string) (*ScannedDocument, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to stat file").
			WithComponent("corpus.scanner").
			WithOperation("scanFile").
			WithDetails("path", path)
	}

	// Skip very large files (>10MB)
	if info.Size() > 10*1024*1024 {
		return nil, nil // Skip silently
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read file").
			WithComponent("corpus.scanner").
			WithOperation("scanFile").
			WithDetails("path", path)
	}

	// Calculate checksum
	checksum := s.calculateChecksum(content)

	// Detect content type
	contentType := s.detectContentType(path)

	// Extract metadata
	metadata := s.extractMetadata(ctx, content, contentType, info)
	metadata.Checksum = checksum

	// Generate document ID based on relative path
	relPath, _ := filepath.Rel(s.basePath, path)
	docID := generateDocumentID(relPath)

	return &ScannedDocument{
		ID:           docID,
		Path:         path,
		Type:         contentType,
		Content:      string(content),
		Metadata:     metadata,
		LastModified: info.ModTime(),
		Checksum:     checksum,
	}, nil
}

// detectContentType determines the content type based on file extension
func (s *DocumentScanner) detectContentType(path string) ContentType {
	ext := strings.ToLower(filepath.Ext(path))
	if ct, ok := s.contentTypes[ext]; ok {
		return ct
	}
	return ContentTypeUnknown
}

// calculateChecksum computes SHA256 checksum of content
func (s *DocumentScanner) calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// extractMetadata analyzes content to extract metadata
func (s *DocumentScanner) extractMetadata(ctx context.Context, content []byte, contentType ContentType, info fs.FileInfo) DocumentMetadata {
	metadata := DocumentMetadata{
		LastModified: info.ModTime(),
		FileSize:     info.Size(),
		WordCount:    countWords(string(content)),
	}

	switch contentType {
	case ContentTypeMarkdown:
		s.extractMarkdownMetadata(&metadata, content)
	case ContentTypeYAML:
		s.extractYAMLMetadata(&metadata, content)
	case ContentTypeGo:
		s.extractGoMetadata(&metadata, content)
	}

	return metadata
}

// extractMarkdownMetadata extracts metadata specific to markdown files
func (s *DocumentScanner) extractMarkdownMetadata(metadata *DocumentMetadata, content []byte) {
	text := string(content)
	lines := strings.Split(text, "\n")

	// Extract title from first heading
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			metadata.Title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	// Count various elements
	for _, line := range lines {
		// Count headings
		if strings.HasPrefix(line, "#") {
			metadata.HeadingCount++
		}

		// Count TODOs
		if strings.Contains(strings.ToUpper(line), "TODO") {
			metadata.TODOCount++
		}

		// Count code blocks
		if strings.HasPrefix(line, "```") {
			metadata.CodeBlockCount++
		}
	}
	metadata.CodeBlockCount /= 2 // Each code block has opening and closing ```

	// Count links
	metadata.LinkCount = len(ExtractLinks(text))

	// Extract tags from hashtags
	metadata.ExtractedTags = extractHashtags(text)

	// Check for YAML frontmatter
	if strings.HasPrefix(text, "---\n") {
		if endIdx := strings.Index(text[4:], "\n---\n"); endIdx > 0 {
			var frontmatter map[string]interface{}
			yamlContent := text[4 : endIdx+4]
			if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err == nil {
				// Extract title from frontmatter if not found
				if metadata.Title == "" {
					if title, ok := frontmatter["title"].(string); ok {
						metadata.Title = title
					}
				}
				// Extract description
				if desc, ok := frontmatter["description"].(string); ok {
					metadata.Description = desc
				}
				// Extract tags
				if tags, ok := frontmatter["tags"].([]interface{}); ok {
					for _, tag := range tags {
						if tagStr, ok := tag.(string); ok {
							metadata.ExtractedTags = append(metadata.ExtractedTags, tagStr)
						}
					}
				}
			}
		}
	}
}

// extractYAMLMetadata extracts metadata from YAML files
func (s *DocumentScanner) extractYAMLMetadata(metadata *DocumentMetadata, content []byte) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err == nil {
		// Try to extract common fields
		if name, ok := data["name"].(string); ok {
			metadata.Title = name
		} else if title, ok := data["title"].(string); ok {
			metadata.Title = title
		}

		if desc, ok := data["description"].(string); ok {
			metadata.Description = desc
		}
	}
}

// extractGoMetadata extracts metadata from Go files
func (s *DocumentScanner) extractGoMetadata(metadata *DocumentMetadata, content []byte) {
	text := string(content)
	lines := strings.Split(text, "\n")

	metadata.Language = "go"

	// Extract package name as title
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			metadata.Title = strings.TrimPrefix(line, "package ")
			break
		}
	}

	// Count TODOs and FIXMEs
	for _, line := range lines {
		if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") {
			metadata.TODOCount++
		}
	}
}

// Helper functions

// countWords counts the number of words in text
func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// extractHashtags finds hashtags in text
func extractHashtags(text string) []string {
	var tags []string
	seen := make(map[string]bool)

	words := strings.Fields(text)
	for _, word := range words {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			tag := strings.TrimPrefix(word, "#")
			tag = strings.Trim(tag, ".,!?;:")
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}

// generateDocumentID creates a unique ID for a document based on its path
func generateDocumentID(path string) string {
	// Normalize path separators
	normalized := strings.ReplaceAll(path, string(filepath.Separator), "/")
	// Remove extension
	normalized = strings.TrimSuffix(normalized, filepath.Ext(normalized))
	// Replace special characters
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}

// GetChecksum computes the checksum for a file
func (s *DocumentScanner) GetChecksum(ctx context.Context, path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read file for checksum").
			WithComponent("corpus.scanner").
			WithOperation("GetChecksum").
			WithDetails("path", path)
	}

	return s.calculateChecksum(content), nil
}

// HasFileChanged checks if a file has changed based on checksum
func (s *DocumentScanner) HasFileChanged(ctx context.Context, path string, previousChecksum string) (bool, error) {
	currentChecksum, err := s.GetChecksum(ctx, path)
	if err != nil {
		return false, err
	}

	return currentChecksum != previousChecksum, nil
}