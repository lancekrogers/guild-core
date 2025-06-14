package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/lsp"
	"github.com/guild-ventures/guild-core/tools"
)

// DocumentSymbolsTool provides document symbols (outline) using LSP
type DocumentSymbolsTool struct {
	*tools.BaseTool
	manager *lsp.Manager
}

// DocumentParams represents the parameters for document-based tools
type DocumentParams struct {
	File string `json:"file"`
}

// SymbolInfo represents information about a symbol
type SymbolInfo struct {
	Name           string         `json:"name"`
	Kind           string         `json:"kind"`
	Range          *Range         `json:"range"`
	SelectionRange *Range         `json:"selection_range,omitempty"`
	Detail         string         `json:"detail,omitempty"`
	Children       []*SymbolInfo `json:"children,omitempty"`
}

// Range represents a text range  
type Range struct {
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
}

// NewDocumentSymbolsTool creates a new document symbols tool
func NewDocumentSymbolsTool(manager *lsp.Manager) *DocumentSymbolsTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "The file path to get symbols from",
			},
		},
		"required": []string{"file"},
	}

	examples := []string{
		`{"file": "/path/to/main.go"}`,
		`{"file": "/path/to/app.py"}`,
		`{"file": "/path/to/index.ts"}`,
	}

	return &DocumentSymbolsTool{
		BaseTool: tools.NewBaseTool(
			"lsp_document_symbols",
			"Get all symbols (functions, classes, variables) in a document using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the document symbols tool
func (t *DocumentSymbolsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params DocumentParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.document_symbols_tool").
			WithOperation("execute")
	}

	// For now, return a placeholder implementation
	// This will be updated when we extend the LSP manager with document symbols support
	
	// Read file to check it exists
	if _, err := os.Stat(params.File); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "file not found").
			WithComponent("lsp.document_symbols_tool").
			WithOperation("execute").
			WithDetails("file", params.File)
	}

	// Placeholder response showing the structure
	symbols := []*SymbolInfo{
		{
			Name: "Example",
			Kind: "Info",
			Detail: "Document symbols will be available when LSP manager is extended",
			Range: &Range{
				StartLine: 0,
				StartColumn: 0,
				EndLine: 0,
				EndColumn: 0,
			},
		},
	}

	// Convert to JSON
	output, err := json.MarshalIndent(symbols, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.document_symbols_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":         params.File,
		"symbol_count": fmt.Sprintf("%d", len(symbols)),
		"status":       "placeholder",
	}

	extraData := map[string]interface{}{
		"note": "This tool requires extending the LSP manager with document symbols support",
	}

	return tools.NewToolResult(string(output), metadata, nil, extraData), nil
}