// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/lsp"
	"github.com/lancekrogers/guild/tools"
)

// CompletionTool provides code completion using LSP
type CompletionTool struct {
	tools.BaseTool
	manager *lsp.Manager
}

// CompletionParams represents the parameters for the completion tool
type CompletionParams struct {
	File    string `json:"file" description:"The file path to get completions for"`
	Line    int    `json:"line" description:"The line number (0-based)"`
	Column  int    `json:"column" description:"The column number (0-based)"`
	Trigger string `json:"trigger,omitempty" description:"Optional trigger character"`
}

// CompletionResult represents the result of a completion request
type CompletionResult struct {
	IsIncomplete bool             `json:"is_incomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionItem represents a single completion item
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insert_text,omitempty"`
	SortText   string `json:"sort_text,omitempty"`
	FilterText string `json:"filter_text,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`
}

// NewCompletionTool creates a new completion tool
func NewCompletionTool(manager *lsp.Manager) *CompletionTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "The file path to get completions for",
			},
			"line": map[string]interface{}{
				"type":        "integer",
				"description": "The line number (0-based)",
			},
			"column": map[string]interface{}{
				"type":        "integer",
				"description": "The column number (0-based)",
			},
			"trigger": map[string]interface{}{
				"type":        "string",
				"description": "Optional trigger character",
			},
		},
		"required": []string{"file", "line", "column"},
	}

	examples := []string{
		`{"file": "/path/to/main.go", "line": 10, "column": 15}`,
		`{"file": "/path/to/app.ts", "line": 25, "column": 8, "trigger": "."}`,
	}

	return &CompletionTool{
		BaseTool: *tools.NewBaseTool(
			"lsp_completion",
			"Get code completions at a specific position in a file using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the completion tool
func (t *CompletionTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params CompletionParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.completion_tool").
			WithOperation("execute")
	}

	// Get completions from LSP
	completions, err := t.manager.GetCompletion(ctx, params.File, params.Line, params.Column, params.Trigger)
	if err != nil {
		return nil, err
	}

	// Convert to our format
	result := CompletionResult{
		IsIncomplete: completions.IsIncomplete,
		Items:        make([]CompletionItem, 0, len(completions.Items)),
	}

	for _, item := range completions.Items {
		// Convert completion kind to string
		kind := completionKindToString(item.Kind)

		// Use InsertText if available, otherwise use Label
		insertText := item.InsertText
		if insertText == "" {
			insertText = item.Label
		}

		result.Items = append(result.Items, CompletionItem{
			Label:      item.Label,
			Kind:       kind,
			Detail:     item.Detail,
			InsertText: insertText,
			SortText:   item.SortText,
			FilterText: item.FilterText,
			Deprecated: item.Deprecated,
		})
	}

	// Convert result to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.completion_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":             params.File,
		"position":         fmt.Sprintf("%d:%d", params.Line, params.Column),
		"completion_count": fmt.Sprintf("%d", len(result.Items)),
	}

	if params.Trigger != "" {
		metadata["trigger"] = params.Trigger
	}

	return tools.NewToolResult(string(output), metadata, nil, nil), nil
}

// completionKindToString converts LSP completion kind to string
func completionKindToString(kind lsp.CompletionItemKind) string {
	switch kind {
	case lsp.CompletionItemKindText:
		return "text"
	case lsp.CompletionItemKindMethod:
		return "method"
	case lsp.CompletionItemKindFunction:
		return "function"
	case lsp.CompletionItemKindConstructor:
		return "constructor"
	case lsp.CompletionItemKindField:
		return "field"
	case lsp.CompletionItemKindVariable:
		return "variable"
	case lsp.CompletionItemKindClass:
		return "class"
	case lsp.CompletionItemKindInterface:
		return "interface"
	case lsp.CompletionItemKindModule:
		return "module"
	case lsp.CompletionItemKindProperty:
		return "property"
	case lsp.CompletionItemKindUnit:
		return "unit"
	case lsp.CompletionItemKindValue:
		return "value"
	case lsp.CompletionItemKindEnum:
		return "enum"
	case lsp.CompletionItemKindKeyword:
		return "keyword"
	case lsp.CompletionItemKindSnippet:
		return "snippet"
	case lsp.CompletionItemKindColor:
		return "color"
	case lsp.CompletionItemKindFile:
		return "file"
	case lsp.CompletionItemKindReference:
		return "reference"
	case lsp.CompletionItemKindFolder:
		return "folder"
	case lsp.CompletionItemKindEnumMember:
		return "enum_member"
	case lsp.CompletionItemKindConstant:
		return "constant"
	case lsp.CompletionItemKindStruct:
		return "struct"
	case lsp.CompletionItemKindEvent:
		return "event"
	case lsp.CompletionItemKindOperator:
		return "operator"
	case lsp.CompletionItemKindTypeParameter:
		return "type_parameter"
	default:
		return "unknown"
	}
}

// FormatCompletionsAsText formats completions as human-readable text
func FormatCompletionsAsText(result *CompletionResult) string {
	if len(result.Items) == 0 {
		return "No completions available"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d completions:\n\n", len(result.Items)))

	for i, item := range result.Items {
		builder.WriteString(fmt.Sprintf("%d. %s (%s)", i+1, item.Label, item.Kind))
		if item.Detail != "" {
			builder.WriteString(fmt.Sprintf("\n   %s", item.Detail))
		}
		if item.Deprecated {
			builder.WriteString(" [DEPRECATED]")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
