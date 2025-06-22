// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"time"
)

// Exporter provides export capabilities for chat sessions and content
type Exporter interface {
	// Export content to specified format
	Export(ctx context.Context, content ExportContent, format ExportFormat) ([]byte, error)

	// Get supported export formats
	SupportedFormats() []ExportFormat

	// Validate export content before processing
	ValidateContent(content ExportContent) error

	// Get format-specific options
	GetFormatOptions(format ExportFormat) []ExportOption
}

// ExportContent represents content to be exported
type ExportContent struct {
	Messages  []ChatMessage     `json:"messages"`
	Metadata  ExportMetadata    `json:"metadata"`
	Selection *ContentSelection `json:"selection,omitempty"`
	Options   ExportOptions     `json:"options,omitempty"`
}

// ChatMessage represents a chat message for export
type ChatMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"` // "user", "assistant", "system"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Author      string    `json:"author,omitempty"`
	Campaign    string    `json:"campaign,omitempty"`
	ExportedAt  time.Time `json:"exported_at"`
	Version     string    `json:"version,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// ContentSelection allows exporting selected portions
type ContentSelection struct {
	StartIndex int      `json:"start_index"`
	EndIndex   int      `json:"end_index"`
	MessageIDs []string `json:"message_ids,omitempty"`
}

// ExportOptions contains format-specific options
type ExportOptions struct {
	IncludeMetadata   bool                   `json:"include_metadata"`
	IncludeTimestamps bool                   `json:"include_timestamps"`
	FormatSpecific    map[string]interface{} `json:"format_specific,omitempty"`
	Theme             string                 `json:"theme,omitempty"`
	CustomTemplate    string                 `json:"custom_template,omitempty"`
}

// ExportFormat defines supported export formats
type ExportFormat string

const (
	FormatMarkdown  ExportFormat = "markdown"
	FormatHTML      ExportFormat = "html"
	FormatJSON      ExportFormat = "json"
	FormatPDF       ExportFormat = "pdf"
	FormatPlainText ExportFormat = "text"
	FormatCSV       ExportFormat = "csv"
)

// ExportOption represents a configurable option for exports
type ExportOption struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type"` // "boolean", "string", "number", "select"
	Default     interface{} `json:"default"`
	Options     []string    `json:"options,omitempty"` // For select type
	Required    bool        `json:"required"`
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	Data       []byte                 `json:"data"`
	Format     ExportFormat           `json:"format"`
	Filename   string                 `json:"filename"`
	MimeType   string                 `json:"mime_type"`
	Size       int64                  `json:"size"`
	ExportedAt time.Time              `json:"exported_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ExportHistory tracks export operations
type ExportHistory struct {
	ID           string       `json:"id"`
	Format       ExportFormat `json:"format"`
	Title        string       `json:"title"`
	Filename     string       `json:"filename"`
	Size         int64        `json:"size"`
	ExportedAt   time.Time    `json:"exported_at"`
	Campaign     string       `json:"campaign,omitempty"`
	MessageCount int          `json:"message_count"`
}

// BatchExportRequest for exporting multiple sessions
type BatchExportRequest struct {
	Sessions []string      `json:"sessions"`
	Format   ExportFormat  `json:"format"`
	Options  ExportOptions `json:"options"`
	Archive  bool          `json:"archive"` // Create zip archive
}

// BatchExportResult contains results of batch export
type BatchExportResult struct {
	Results    []ExportResult `json:"results"`
	Archive    []byte         `json:"archive,omitempty"` // ZIP archive if requested
	TotalSize  int64          `json:"total_size"`
	ExportedAt time.Time      `json:"exported_at"`
	Errors     []string       `json:"errors,omitempty"`
}
