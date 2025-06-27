// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// MultiFormatExporter implements the Exporter interface with support for multiple formats
type MultiFormatExporter struct {
	formatters map[ExportFormat]Formatter
}

// Formatter interface for format-specific export implementations
type Formatter interface {
	Format(ctx context.Context, content ExportContent) ([]byte, error)
	GetMimeType() string
	GetFileExtension() string
	GetOptions() []ExportOption
	ValidateOptions(options ExportOptions) error
}

// NewMultiFormatExporter creates a new exporter with all supported formats
func NewMultiFormatExporter() *MultiFormatExporter {
	exporter := &MultiFormatExporter{
		formatters: make(map[ExportFormat]Formatter),
	}

	// Register all format handlers
	exporter.formatters[FormatMarkdown] = NewMarkdownFormatter()
	exporter.formatters[FormatHTML] = NewHTMLFormatter()
	exporter.formatters[FormatJSON] = NewJSONFormatter()
	exporter.formatters[FormatPlainText] = NewPlainTextFormatter()
	exporter.formatters[FormatCSV] = NewCSVFormatter()
	// PDF formatter would require additional dependencies

	return exporter
}

// Export exports content to the specified format
func (e *MultiFormatExporter) Export(ctx context.Context, content ExportContent, format ExportFormat) ([]byte, error) {
	formatter, exists := e.formatters[format]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported export format: %s", format), nil).
			WithComponent("export").
			WithOperation("Export").
			WithDetails("format", string(format))
	}

	// Validate content
	if err := e.ValidateContent(content); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "content validation failed").
			WithComponent("export").
			WithOperation("Export")
	}

	// Validate format-specific options
	if err := formatter.ValidateOptions(content.Options); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "options validation failed").
			WithComponent("export").
			WithOperation("Export").
			WithDetails("format", string(format))
	}

	// Set default metadata if not provided
	if content.Metadata.ExportedAt.IsZero() {
		content.Metadata.ExportedAt = time.Now()
	}
	if content.Metadata.Version == "" {
		content.Metadata.Version = "1.0"
	}

	// Perform the export
	data, err := formatter.Format(ctx, content)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "formatting failed").
			WithComponent("export").
			WithOperation("Export").
			WithDetails("format", string(format))
	}

	return data, nil
}

// SupportedFormats returns all supported export formats
func (e *MultiFormatExporter) SupportedFormats() []ExportFormat {
	formats := make([]ExportFormat, 0, len(e.formatters))
	for format := range e.formatters {
		formats = append(formats, format)
	}
	return formats
}

// ValidateContent validates the export content
func (e *MultiFormatExporter) ValidateContent(content ExportContent) error {
	if len(content.Messages) == 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "no messages to export", nil).
			WithComponent("export").
			WithOperation("ValidateContent")
	}

	if content.Metadata.Title == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "export title is required", nil).
			WithComponent("export").
			WithOperation("ValidateContent")
	}

	// Validate selection if provided
	if content.Selection != nil {
		if content.Selection.StartIndex < 0 || content.Selection.EndIndex >= len(content.Messages) {
			return gerror.New(gerror.ErrCodeInvalidInput, "invalid message selection range", nil).
				WithComponent("export").
				WithOperation("ValidateContent").
				WithDetails("start_index", fmt.Sprintf("%d", content.Selection.StartIndex)).
				WithDetails("end_index", fmt.Sprintf("%d", content.Selection.EndIndex))
		}
		if content.Selection.StartIndex > content.Selection.EndIndex {
			return gerror.New(gerror.ErrCodeInvalidInput, "start index must be less than or equal to end index", nil).
				WithComponent("export").
				WithOperation("ValidateContent")
		}
	}

	// Validate messages
	for i, msg := range content.Messages {
		if msg.Content == "" {
			return gerror.New(gerror.ErrCodeInvalidInput, "message content cannot be empty", nil).
				WithComponent("export").
				WithOperation("ValidateContent").
				WithDetails("message_index", fmt.Sprintf("%d", i))
		}
		if msg.Role == "" {
			return gerror.New(gerror.ErrCodeInvalidInput, "message role is required", nil).
				WithComponent("export").
				WithOperation("ValidateContent").
				WithDetails("message_index", fmt.Sprintf("%d", i))
		}
	}

	return nil
}

// GetFormatOptions returns available options for the specified format
func (e *MultiFormatExporter) GetFormatOptions(format ExportFormat) []ExportOption {
	formatter, exists := e.formatters[format]
	if !exists {
		return nil
	}
	return formatter.GetOptions()
}

// GetFormatter returns the formatter for the specified format
func (e *MultiFormatExporter) GetFormatter(format ExportFormat) Formatter {
	return e.formatters[format]
}

// ExportToResult performs export and returns a complete result
func (e *MultiFormatExporter) ExportToResult(ctx context.Context, content ExportContent, format ExportFormat) (*ExportResult, error) {
	data, err := e.Export(ctx, content, format)
	if err != nil {
		return nil, err
	}

	formatter := e.formatters[format]
	filename := generateFilename(content.Metadata.Title, formatter.GetFileExtension())

	return &ExportResult{
		Data:       data,
		Format:     format,
		Filename:   filename,
		MimeType:   formatter.GetMimeType(),
		Size:       int64(len(data)),
		ExportedAt: time.Now(),
		Metadata: map[string]interface{}{
			"message_count": len(content.Messages),
			"title":         content.Metadata.Title,
			"campaign":      content.Metadata.Campaign,
		},
	}, nil
}

// generateFilename creates a filename from title and extension
func generateFilename(title, extension string) string {
	// Sanitize title for filename
	filename := strings.ReplaceAll(title, " ", "_")
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename = strings.ReplaceAll(filename, "*", "-")
	filename = strings.ReplaceAll(filename, "?", "-")
	filename = strings.ReplaceAll(filename, "\"", "-")
	filename = strings.ReplaceAll(filename, "<", "-")
	filename = strings.ReplaceAll(filename, ">", "-")
	filename = strings.ReplaceAll(filename, "|", "-")

	// Limit length
	if len(filename) > 50 {
		filename = filename[:50]
	}

	// Add timestamp if no meaningful title
	if filename == "" || filename == "_" {
		filename = fmt.Sprintf("export_%s", time.Now().Format("2006-01-02_15-04-05"))
	}

	return fmt.Sprintf("%s.%s", filename, extension)
}
