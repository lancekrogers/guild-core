// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/lsp"
	"github.com/guild-framework/guild-core/tools"
)

// ReferencesTool finds all references to a symbol using LSP
type ReferencesTool struct {
	tools.BaseTool
	manager *lsp.Manager
}

// ReferencesParams represents the parameters for the references tool
type ReferencesParams struct {
	File               string `json:"file" description:"The file path"`
	Line               int    `json:"line" description:"The line number (0-based)"`
	Column             int    `json:"column" description:"The column number (0-based)"`
	IncludeDeclaration bool   `json:"include_declaration,omitempty" description:"Whether to include the declaration in results"`
}

// ReferencesResult represents the result of a references request
type ReferencesResult struct {
	References []LocationResult `json:"references"`
	TotalCount int              `json:"total_count"`
}

// NewReferencesTool creates a new references tool
func NewReferencesTool(manager *lsp.Manager) *ReferencesTool {
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
			"include_declaration": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to include the declaration in results",
				"default":     true,
			},
		},
		"required": []string{"file", "line", "column"},
	}

	examples := []string{
		`{"file": "/path/to/main.go", "line": 10, "column": 15}`,
		`{"file": "/path/to/app.ts", "line": 25, "column": 8, "include_declaration": false}`,
	}

	return &ReferencesTool{
		BaseTool: *tools.NewBaseTool(
			"lsp_references",
			"Find all references to a symbol at a specific position using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the references tool
func (t *ReferencesTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params ReferencesParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.references_tool").
			WithOperation("execute")
	}

	// Default to including declaration
	if input != "" && !strings.Contains(input, "include_declaration") {
		params.IncludeDeclaration = true
	}

	// Get references from LSP
	locations, err := t.manager.GetReferences(ctx, params.File, params.Line, params.Column, params.IncludeDeclaration)
	if err != nil {
		return nil, err
	}

	// Convert to our format
	result := ReferencesResult{
		References: make([]LocationResult, 0, len(locations)),
		TotalCount: len(locations),
	}

	// Group references by file for better organization
	fileGroups := make(map[string][]LocationResult)

	for _, loc := range locations {
		// Convert URI to file path
		filePath := strings.TrimPrefix(loc.URI, "file://")

		locResult := LocationResult{
			File:      filePath,
			Line:      loc.Range.Start.Line,
			Column:    loc.Range.Start.Character,
			EndLine:   loc.Range.End.Line,
			EndColumn: loc.Range.End.Character,
		}

		result.References = append(result.References, locResult)
		fileGroups[filePath] = append(fileGroups[filePath], locResult)
	}

	// Add file group count to metadata
	extraData := map[string]interface{}{
		"files_with_references": len(fileGroups),
		"file_groups":           fileGroups,
	}

	// Convert result to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.references_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":            params.File,
		"position":        fmt.Sprintf("%d:%d", params.Line, params.Column),
		"reference_count": fmt.Sprintf("%d", len(result.References)),
		"file_count":      fmt.Sprintf("%d", len(fileGroups)),
	}

	return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}

// FormatReferencesAsText formats references as human-readable text
func FormatReferencesAsText(result *ReferencesResult) string {
	if len(result.References) == 0 {
		return "No references found"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d references:\n\n", result.TotalCount))

	// Group by file for better readability
	fileGroups := make(map[string][]LocationResult)
	for _, ref := range result.References {
		fileGroups[ref.File] = append(fileGroups[ref.File], ref)
	}

	fileNum := 1
	for file, refs := range fileGroups {
		builder.WriteString(fmt.Sprintf("%d. %s (%d references)\n", fileNum, file, len(refs)))
		for _, ref := range refs {
			builder.WriteString(fmt.Sprintf("   - Line %d, Column %d\n", ref.Line+1, ref.Column+1))
		}
		builder.WriteString("\n")
		fileNum++
	}

	return builder.String()
}
