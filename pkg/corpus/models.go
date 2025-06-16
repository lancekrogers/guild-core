// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package corpus provides functionality for storing and organizing research findings,
// summaries, and generated insights in a structured, human-navigable format.
package corpus

import (
	"time"
)

// CorpusDoc represents a document in the research corpus.
type CorpusDoc struct {
	// Title is the document's main heading
	Title string `json:"title" yaml:"title"`

	// Source indicates where the content originated from (e.g., "YouTube", "GitHub")
	Source string `json:"source" yaml:"source"`

	// Tags categorize the document for organization and filtering
	Tags []string `json:"tags" yaml:"tags"`

	// Body contains the main markdown content
	Body string `json:"body" yaml:"body,omitempty"`

	// Links contains the extracted wikilinks to other documents
	Links []string `json:"links" yaml:"links,omitempty"`

	// GuildID identifies the guild that created the document
	GuildID string `json:"guild_id" yaml:"guild_id"`

	// AgentID identifies the agent that created the document
	AgentID string `json:"agent_id" yaml:"agent_id"`

	// CreatedAt records when the document was created
	CreatedAt time.Time `json:"created_at" yaml:"created"`

	// UpdatedAt records when the document was last updated
	UpdatedAt time.Time `json:"updated_at" yaml:"updated,omitempty"`

	// FilePath stores the absolute path where the document is saved
	FilePath string `json:"file_path" yaml:"-"`
}

// Metadata represents the YAML frontmatter in a markdown document
type Metadata struct {
	Title    string    `yaml:"title"`
	Source   string    `yaml:"source"`
	Tags     []string  `yaml:"tags"`
	Created  time.Time `yaml:"created"`
	Updated  time.Time `yaml:"updated,omitempty"`
	Author   string    `yaml:"author"` // Format: "guildID:agentID"
	FilePath string    `yaml:"-"`
}

// Config represents corpus configuration
type Config struct {
	// CorpusPath is the directory path where corpus documents are stored
	CorpusPath string `yaml:"corpus_path" json:"corpus_path"`

	// ActivitiesPath is the directory where user activity logs are stored
	ActivitiesPath string `yaml:"activities_path" json:"activities_path"`

	// MaxSizeBytes is the maximum size of the corpus in bytes
	MaxSizeBytes int64 `yaml:"max_size_bytes" json:"max_size_bytes"`

	// DefaultTags are automatically added to new documents when none are provided
	DefaultTags []string `yaml:"default_tags" json:"default_tags"`

	// DefaultCategory is the category to use when none is specified
	DefaultCategory string `yaml:"default_category" json:"default_category"`

	// Location is an alias for CorpusPath to maintain backward compatibility
	// Deprecated: Use CorpusPath instead
	Location string `yaml:"-" json:"-"`

	// MaxSizeMB is an alias for MaxSizeBytes / 1024 / 1024
	// Deprecated: Use MaxSizeBytes instead
	MaxSizeMB int64 `yaml:"-" json:"-"`
}

// ViewLog represents a user's interaction with corpus documents
type ViewLog struct {
	// User identifies who viewed the document
	User string `json:"user"`

	// DocPath is the path to the viewed document
	DocPath string `json:"doc_path"`

	// Timestamp records when the document was viewed
	Timestamp time.Time `json:"timestamp"`
}

// NewCorpusDoc creates a new corpus document with sensible defaults
func NewCorpusDoc(title, source, body, guildID, agentID string, tags []string) *CorpusDoc {
	now := time.Now()
	return &CorpusDoc{
		Title:     title,
		Source:    source,
		Tags:      tags,
		Body:      body,
		Links:     []string{},
		GuildID:   guildID,
		AgentID:   agentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
