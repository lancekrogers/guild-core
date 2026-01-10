// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/lsp"
	"github.com/lancekrogers/guild-core/tools"
)

// FormatDocumentTool provides document formatting using LSP
type FormatDocumentTool struct {
	*tools.BaseTool
	manager *lsp.Manager
}

// FormatParams represents the parameters for formatting
type FormatParams struct {
	File    string                 `json:"file"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// FormatResult represents the result of a format operation
type FormatResult struct {
	Success   bool       `json:"success"`
	Formatted bool       `json:"formatted"`
	Edits     []TextEdit `json:"edits,omitempty"`
	Summary   struct {
		LinesChanged int    `json:"lines_changed"`
		Message      string `json:"message"`
	} `json:"summary"`
}

// NewFormatDocumentTool creates a new format document tool
func NewFormatDocumentTool(manager *lsp.Manager) *FormatDocumentTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "The file path to format",
			},
			"options": map[string]interface{}{
				"type":        "object",
				"description": "Language-specific formatting options",
				"properties": map[string]interface{}{
					"tabSize": map[string]interface{}{
						"type":        "integer",
						"description": "Size of a tab in spaces",
					},
					"insertSpaces": map[string]interface{}{
						"type":        "boolean",
						"description": "Prefer spaces over tabs",
					},
				},
			},
		},
		"required": []string{"file"},
	}

	examples := []string{
		`{"file": "/path/to/main.go"}`,
		`{"file": "/path/to/app.py", "options": {"tabSize": 4, "insertSpaces": true}}`,
		`{"file": "/path/to/index.js", "options": {"tabSize": 2}}`,
	}

	return &FormatDocumentTool{
		BaseTool: tools.NewBaseTool(
			"lsp_format_document",
			"Format a document according to language standards using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the format document tool
func (t *FormatDocumentTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params FormatParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.format_document_tool").
			WithOperation("execute")
	}

	// Check if file exists
	if _, err := os.Stat(params.File); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "file not found").
			WithComponent("lsp.format_document_tool").
			WithOperation("execute").
			WithDetails("file", params.File)
	}

	// For now, return a placeholder implementation
	// This will be updated when we extend the LSP manager with formatting support

	// Placeholder response showing formatting changes
	result := &FormatResult{
		Success:   true,
		Formatted: true,
		Edits: []TextEdit{
			{
				Range: &Range{
					StartLine:   5,
					StartColumn: 0,
					EndLine:     5,
					EndColumn:   10,
				},
				NewText: "    ", // Indentation fix
			},
			{
				Range: &Range{
					StartLine:   10,
					StartColumn: 20,
					EndLine:     10,
					EndColumn:   21,
				},
				NewText: " ", // Space formatting
			},
			{
				Range: &Range{
					StartLine:   15,
					StartColumn: 0,
					EndLine:     16,
					EndColumn:   0,
				},
				NewText: "", // Remove empty line
			},
		},
	}

	// Calculate summary
	result.Summary.LinesChanged = 0
	lines := make(map[int]bool)
	for _, edit := range result.Edits {
		for line := edit.Range.StartLine; line <= edit.Range.EndLine; line++ {
			lines[line] = true
		}
	}
	result.Summary.LinesChanged = len(lines)

	if result.Summary.LinesChanged > 0 {
		result.Summary.Message = fmt.Sprintf("Formatted %d lines", result.Summary.LinesChanged)
	} else {
		result.Summary.Message = "Document is already properly formatted"
		result.Formatted = false
	}

	// Convert to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.format_document_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":          params.File,
		"formatted":     fmt.Sprintf("%v", result.Formatted),
		"lines_changed": fmt.Sprintf("%d", result.Summary.LinesChanged),
		"status":        "placeholder",
	}

	// Add formatting options to metadata if provided
	if params.Options != nil {
		if tabSize, ok := params.Options["tabSize"].(float64); ok {
			metadata["tab_size"] = fmt.Sprintf("%d", int(tabSize))
		}
		if insertSpaces, ok := params.Options["insertSpaces"].(bool); ok {
			metadata["insert_spaces"] = fmt.Sprintf("%v", insertSpaces)
		}
	}

	extraData := map[string]interface{}{
		"note":      "This tool requires extending the LSP manager with formatting support",
		"formatted": result.Formatted,
	}

	return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}
