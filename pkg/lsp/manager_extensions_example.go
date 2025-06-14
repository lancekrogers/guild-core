// +build example

// This file shows example implementations of the methods that need to be added
// to the Manager type to support the new LSP tools. These would go in manager.go

package lsp

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// GetDocumentSymbols returns all symbols in a document
func (m *Manager) GetDocumentSymbols(ctx context.Context, filePath string) ([]DocumentSymbol, error) {
	logger := observability.GetLogger(ctx)

	// Get server for file
	server, err := m.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Ensure file is opened in server
	if err := m.ensureFileOpened(ctx, server, filePath); err != nil {
		return nil, err
	}

	// Create parameters
	params := &DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{
			URI: filePathToURI(filePath),
		},
	}

	// Send document symbols request
	var result []DocumentSymbol
	if err := server.Client.request(ctx, "textDocument/documentSymbol", params, &result); err != nil {
		// Some servers return SymbolInformation instead of DocumentSymbol
		var symbolInfos []SymbolInformation
		if err2 := server.Client.request(ctx, "textDocument/documentSymbol", params, &symbolInfos); err2 == nil {
			// Convert SymbolInformation to DocumentSymbol
			result = convertSymbolInfoToDocumentSymbol(symbolInfos)
		} else {
			return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "document symbols request failed").
				WithComponent("lsp").
				WithOperation("get_document_symbols").
				WithDetails("file", filePath)
		}
	}

	logger.InfoContext(ctx, "Retrieved document symbols",
		"file", filePath,
		"symbol_count", len(result))

	return result, nil
}

// GetWorkspaceSymbols searches for symbols across the workspace
func (m *Manager) GetWorkspaceSymbols(ctx context.Context, query string) ([]SymbolInformation, error) {
	logger := observability.GetLogger(ctx)

	// For workspace symbols, we need to pick a server that has workspace capabilities
	// For now, use the first active server
	servers := m.GetActiveServers()
	if len(servers) == 0 {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no active language servers").
			WithComponent("lsp").
			WithOperation("get_workspace_symbols")
	}

	// Get the first server
	m.mu.RLock()
	server, exists := m.servers[fmt.Sprintf("%s:%s", servers[0].Language, servers[0].Workspace)]
	m.mu.RUnlock()

	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "server not found").
			WithComponent("lsp").
			WithOperation("get_workspace_symbols")
	}

	// Create parameters
	params := &WorkspaceSymbolParams{
		Query: query,
	}

	// Send workspace symbols request
	var result []SymbolInformation
	if err := server.Client.request(ctx, "workspace/symbol", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "workspace symbols request failed").
			WithComponent("lsp").
			WithOperation("get_workspace_symbols").
			WithDetails("query", query)
	}

	logger.InfoContext(ctx, "Retrieved workspace symbols",
		"query", query,
		"result_count", len(result))

	return result, nil
}

// GetCodeActions returns available code actions at the given range
func (m *Manager) GetCodeActions(ctx context.Context, filePath string, startLine, startChar, endLine, endChar int) ([]CodeAction, error) {
	// Get server for file
	server, err := m.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Ensure file is opened in server
	if err := m.ensureFileOpened(ctx, server, filePath); err != nil {
		return nil, err
	}

	// Create parameters
	params := &CodeActionParams{
		TextDocument: TextDocumentIdentifier{
			URI: filePathToURI(filePath),
		},
		Range: Range{
			Start: Position{Line: startLine, Character: startChar},
			End:   Position{Line: endLine, Character: endChar},
		},
		Context: CodeActionContext{
			// Could include diagnostics here if available
			Diagnostics: []Diagnostic{},
		},
	}

	// Send code actions request
	var result []CodeAction
	if err := server.Client.request(ctx, "textDocument/codeAction", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "code actions request failed").
			WithComponent("lsp").
			WithOperation("get_code_actions").
			WithDetails("file", filePath).
			WithDetails("range", fmt.Sprintf("%d:%d-%d:%d", startLine, startChar, endLine, endChar))
	}

	return result, nil
}

// RenameSymbol renames a symbol across all its usages
func (m *Manager) RenameSymbol(ctx context.Context, filePath string, line, character int, newName string) (*WorkspaceEdit, error) {
	// Get server for file
	server, err := m.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Ensure file is opened in server
	if err := m.ensureFileOpened(ctx, server, filePath); err != nil {
		return nil, err
	}

	// First, check if rename is valid at this position (optional)
	prepareParams := &PrepareRenameParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: filePathToURI(filePath),
			},
			Position: Position{
				Line:      line,
				Character: character,
			},
		},
	}

	var prepareResult interface{}
	if err := server.Client.request(ctx, "textDocument/prepareRename", prepareParams, &prepareResult); err != nil {
		// Some servers don't support prepareRename, continue anyway
		observability.GetLogger(ctx).WarnContext(ctx, "prepareRename not supported", "error", err)
	}

	// Create rename parameters
	params := &RenameParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: filePathToURI(filePath),
			},
			Position: Position{
				Line:      line,
				Character: character,
			},
		},
		NewName: newName,
	}

	// Send rename request
	var result WorkspaceEdit
	if err := server.Client.request(ctx, "textDocument/rename", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "rename request failed").
			WithComponent("lsp").
			WithOperation("rename_symbol").
			WithDetails("file", filePath).
			WithDetails("position", fmt.Sprintf("%d:%d", line, character)).
			WithDetails("new_name", newName)
	}

	return &result, nil
}

// FormatDocument formats an entire document
func (m *Manager) FormatDocument(ctx context.Context, filePath string, options FormattingOptions) ([]TextEdit, error) {
	// Get server for file
	server, err := m.GetServerForFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Ensure file is opened in server
	if err := m.ensureFileOpened(ctx, server, filePath); err != nil {
		return nil, err
	}

	// Create parameters
	params := &DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{
			URI: filePathToURI(filePath),
		},
		Options: options,
	}

	// Send formatting request
	var result []TextEdit
	if err := server.Client.request(ctx, "textDocument/formatting", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "formatting request failed").
			WithComponent("lsp").
			WithOperation("format_document").
			WithDetails("file", filePath)
	}

	return result, nil
}

// Helper function to convert SymbolInformation to DocumentSymbol
func convertSymbolInfoToDocumentSymbol(infos []SymbolInformation) []DocumentSymbol {
	var symbols []DocumentSymbol
	for _, info := range infos {
		symbols = append(symbols, DocumentSymbol{
			Name:           info.Name,
			Kind:           info.Kind,
			Range:          info.Location.Range,
			SelectionRange: info.Location.Range,
			// Note: SymbolInformation doesn't have children, so hierarchy is lost
		})
	}
	return symbols
}