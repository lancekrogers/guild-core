// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/lsp"
	"github.com/lancekrogers/guild-core/tools"
)

// HoverTool provides hover information (type info, docs) using LSP
type HoverTool struct {
	tools.BaseTool
	manager *lsp.Manager
}

// HoverResult represents the result of a hover request
type HoverResult struct {
	Content  string `json:"content"`
	Language string `json:"language,omitempty"`
	Range    *Range `json:"range,omitempty"`
}

// Range represents a text range
type Range struct {
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
}

// NewHoverTool creates a new hover tool
func NewHoverTool(manager *lsp.Manager) *HoverTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "The file path",
			},
			"line": map[string]interface{}{
				"type":        "integer",
				"description": "The line number (0-based)",
			},
			"column": map[string]interface{}{
				"type":        "integer",
				"description": "The column number (0-based)",
			},
		},
		"required": []string{"file", "line", "column"},
	}

	examples := []string{
		`{"file": "/path/to/main.go", "line": 10, "column": 15}`,
		`{"file": "/path/to/app.ts", "line": 25, "column": 8}`,
	}

	return &HoverTool{
		BaseTool: *tools.NewBaseTool(
			"lsp_hover",
			"Get type information and documentation for a symbol using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the hover tool
func (t *HoverTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params LocationParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.hover_tool").
			WithOperation("execute")
	}

	// Get hover info from LSP
	hover, err := t.manager.GetHover(ctx, params.File, params.Line, params.Column)
	if err != nil {
		return nil, err
	}

	// Convert to our format
	result := HoverResult{}

	// Extract content based on type
	switch content := hover.Contents.(type) {
	case string:
		result.Content = content
	case map[string]interface{}:
		// Handle MarkupContent
		if kind, ok := content["kind"].(string); ok {
			result.Language = kind
		}
		if value, ok := content["value"].(string); ok {
			result.Content = value
		}
	case []interface{}:
		// Handle array of MarkedString
		var parts []string
		for _, item := range content {
			switch v := item.(type) {
			case string:
				parts = append(parts, v)
			case map[string]interface{}:
				if lang, ok := v["language"].(string); ok && result.Language == "" {
					result.Language = lang
				}
				if value, ok := v["value"].(string); ok {
					parts = append(parts, value)
				}
			}
		}
		result.Content = strings.Join(parts, "\n\n")
	}

	// Add range if present
	if hover.Range != nil {
		result.Range = &Range{
			StartLine:   hover.Range.Start.Line,
			StartColumn: hover.Range.Start.Character,
			EndLine:     hover.Range.End.Line,
			EndColumn:   hover.Range.End.Character,
		}
	}

	// Convert result to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.hover_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":     params.File,
		"position": fmt.Sprintf("%d:%d", params.Line, params.Column),
	}

	if result.Language != "" {
		metadata["language"] = result.Language
	}

	// Extract key information for metadata
	extraData := extractHoverInfo(result.Content)

	return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}

// extractHoverInfo extracts structured information from hover content
func extractHoverInfo(content string) map[string]interface{} {
	info := make(map[string]interface{})

	lines := strings.Split(content, "\n")

	// Try to extract type information
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Common patterns for type info
		if strings.HasPrefix(line, "type ") {
			info["type_definition"] = line
		} else if strings.HasPrefix(line, "func ") {
			info["function_signature"] = line
		} else if strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "const ") {
			info["variable_declaration"] = line
		} else if strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "interface ") {
			info["class_or_interface"] = line
		}

		// Extract package info for Go
		if strings.Contains(line, "package ") && strings.Contains(line, " ") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "package" && i+1 < len(parts) {
					info["package"] = parts[i+1]
					break
				}
			}
		}
	}

	// Check if content contains documentation
	if strings.Contains(content, "//") || strings.Contains(content, "/*") || strings.Contains(content, "/**") {
		info["has_documentation"] = true
	}

	return info
}

// FormatHoverAsText formats hover information as human-readable text
func FormatHoverAsText(result *HoverResult) string {
	if result.Content == "" {
		return "No hover information available"
	}

	var builder strings.Builder

	if result.Language != "" {
		builder.WriteString(fmt.Sprintf("Language: %s\n\n", result.Language))
	}

	builder.WriteString(result.Content)

	if result.Range != nil {
		builder.WriteString(fmt.Sprintf("\n\nRange: Line %d:%d to %d:%d",
			result.Range.StartLine+1, result.Range.StartColumn+1,
			result.Range.EndLine+1, result.Range.EndColumn+1))
	}

	return builder.String()
}
