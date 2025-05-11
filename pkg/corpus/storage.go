package corpus

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// MetadataSeparator defines the marker for YAML frontmatter
	MetadataSeparator = "---"

	// DefaultMaxSizeMB defines the default maximum corpus size in MB
	DefaultMaxSizeMB = 1024 // 1GB default

	// DefaultCategory is used when no category is specified
	DefaultCategory = "general"

	// GraphDirName is the directory name for graph data
	GraphDirName = "_graph"

	// ViewLogDirName is the directory name for view logs
	ViewLogDirName = "_viewlog"

	// DefaultPerms are the default permissions for new files and directories
	DefaultPerms = 0755
)

// ErrCorpusTooLarge is returned when a save would exceed the configured maximum size
var ErrCorpusTooLarge = errors.New("corpus exceeds maximum configured size")

// ErrDocNotFound is returned when a document cannot be found
var ErrDocNotFound = errors.New("document not found in corpus")

// Ensure creates the corpus directory structure if it doesn't exist
func Ensure(cfg Config) error {
	if cfg.Location == "" {
		return errors.New("corpus location not specified")
	}

	// Create main corpus directory
	if err := os.MkdirAll(cfg.Location, DefaultPerms); err != nil {
		return fmt.Errorf("failed to create corpus directory: %w", err)
	}

	// Create graph directory
	graphDir := filepath.Join(cfg.Location, GraphDirName)
	if err := os.MkdirAll(graphDir, DefaultPerms); err != nil {
		return fmt.Errorf("failed to create graph directory: %w", err)
	}

	// Create view log directory
	viewLogDir := filepath.Join(cfg.Location, ViewLogDirName)
	if err := os.MkdirAll(viewLogDir, DefaultPerms); err != nil {
		return fmt.Errorf("failed to create view log directory: %w", err)
	}

	// If default category is specified, create that directory
	if cfg.DefaultCategory != "" {
		defaultDir := filepath.Join(cfg.Location, cfg.DefaultCategory)
		if err := os.MkdirAll(defaultDir, DefaultPerms); err != nil {
			return fmt.Errorf("failed to create default category directory: %w", err)
		}
	}

	return nil
}

// Save stores a CorpusDoc to the filesystem in markdown format with YAML frontmatter
func Save(doc *CorpusDoc, cfg Config) error {
	if doc == nil {
		return errors.New("document cannot be nil")
	}

	// Create the corpus directory structure if needed
	if err := Ensure(cfg); err != nil {
		return err
	}

	// Set update time to now
	doc.UpdatedAt = time.Now()

	// Determine category from first tag, or use default
	category := cfg.DefaultCategory
	if len(doc.Tags) > 0 && doc.Tags[0] != "" {
		category = sanitizeFileName(doc.Tags[0])
	}

	// Ensure category directory exists
	categoryDir := filepath.Join(cfg.Location, category)
	if err := os.MkdirAll(categoryDir, DefaultPerms); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	// Create a sanitized filename based on the title
	fileName := sanitizeFileName(doc.Title) + ".md"
	filePath := filepath.Join(categoryDir, fileName)
	doc.FilePath = filePath

	// Build metadata for frontmatter
	metadata := Metadata{
		Title:   doc.Title,
		Source:  doc.Source,
		Tags:    doc.Tags,
		Created: doc.CreatedAt,
		Updated: doc.UpdatedAt,
		Author:  fmt.Sprintf("%s:%s", doc.GuildID, doc.AgentID),
	}

	// Marshal metadata to YAML
	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Format the document content with frontmatter
	content := fmt.Sprintf("%s\n%s\n%s\n\n%s", 
		MetadataSeparator, 
		string(yamlData), 
		MetadataSeparator,
		doc.Body)

	// Check corpus size constraints before writing
	if cfg.MaxSizeMB > 0 {
		if err := checkCorpusSize(cfg, len(content)); err != nil {
			return err
		}
	}

	// Write the file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write document: %w", err)
	}

	return nil
}

// Load retrieves a CorpusDoc from the filesystem
func Load(path string) (*CorpusDoc, error) {
	// Ensure the file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDocNotFound
		}
		return nil, fmt.Errorf("failed to access document: %w", err)
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	// Split the content into frontmatter and body
	parts := strings.Split(string(content), MetadataSeparator)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid document format, missing frontmatter")
	}

	// Extract YAML frontmatter
	var metadata Metadata
	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse author field (guildID:agentID)
	guildID, agentID := parseAuthor(metadata.Author)

	// Extract body (everything after the second separator)
	bodyStart := strings.Index(string(content), MetadataSeparator) + len(MetadataSeparator)
	bodyStart = strings.Index(string(content[bodyStart:]), MetadataSeparator) + bodyStart + len(MetadataSeparator)
	body := strings.TrimSpace(string(content[bodyStart:]))

	// Create and return the document
	doc := &CorpusDoc{
		Title:     metadata.Title,
		Source:    metadata.Source,
		Tags:      metadata.Tags,
		Body:      body,
		Links:     extractLinks(body),
		GuildID:   guildID,
		AgentID:   agentID,
		CreatedAt: metadata.Created,
		UpdatedAt: metadata.Updated,
		FilePath:  path,
	}

	return doc, nil
}

// List returns all corpus documents
func List(cfg Config) ([]string, error) {
	if cfg.Location == "" {
		return nil, errors.New("corpus location not specified")
	}

	var docs []string
	err := filepath.WalkDir(cfg.Location, func(path string, d fs.DirEntry, err error) error {
		// Skip errors, directories, and non-markdown files
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Skip files in special directories
		dir := filepath.Dir(path)
		if strings.Contains(dir, GraphDirName) || strings.Contains(dir, ViewLogDirName) {
			return nil
		}

		docs = append(docs, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list corpus documents: %w", err)
	}

	return docs, nil
}

// ListByTag returns all corpus documents with a specific tag
func ListByTag(cfg Config, tag string) ([]string, error) {
	if cfg.Location == "" {
		return nil, errors.New("corpus location not specified")
	}

	var docs []string
	allDocs, err := List(cfg)
	if err != nil {
		return nil, err
	}

	for _, path := range allDocs {
		doc, err := Load(path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}

		for _, docTag := range doc.Tags {
			if docTag == tag {
				docs = append(docs, path)
				break
			}
		}
	}

	return docs, nil
}

// Delete removes a corpus document
func Delete(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ErrDocNotFound
		}
		return fmt.Errorf("failed to access document: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetSize returns the total size of the corpus in bytes
func GetSize(cfg Config) (int64, error) {
	if cfg.Location == "" {
		return 0, errors.New("corpus location not specified")
	}

	var size int64
	err := filepath.WalkDir(cfg.Location, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil
		}

		size += info.Size()
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate corpus size: %w", err)
	}

	return size, nil
}

// Helper functions

// sanitizeFileName makes a string safe for use as a filename
func sanitizeFileName(s string) string {
	// Replace spaces and special characters
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")

	// Convert to lowercase
	s = strings.ToLower(s)

	// Limit to 100 characters
	if len(s) > 100 {
		s = s[:100]
	}

	return s
}

// parseAuthor splits the author field into guildID and agentID
func parseAuthor(author string) (string, string) {
	parts := strings.Split(author, ":")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// checkCorpusSize checks if adding a document would exceed the maximum corpus size
func checkCorpusSize(cfg Config, newContentSize int) error {
	if cfg.MaxSizeMB <= 0 {
		return nil // No size limit
	}

	// Get current corpus size
	currentSize, err := GetSize(cfg)
	if err != nil {
		return err
	}

	// Calculate maximum size in bytes
	maxBytes := int64(cfg.MaxSizeMB) * 1024 * 1024

	// Check if adding the new content would exceed the limit
	if currentSize+int64(newContentSize) > maxBytes {
		return ErrCorpusTooLarge
	}

	return nil
}

// extractLinks is a placeholder for the function that will be defined in links.go
func extractLinks(content string) []string {
	// This is a placeholder - actual implementation in links.go
	return []string{}
}