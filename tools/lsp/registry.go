// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"github.com/lancekrogers/guild/pkg/lsp"
	"github.com/lancekrogers/guild/tools"
)

// RegisterAll registers all LSP tools with the given registry
func RegisterAll(registry interface{ Register(tool tools.Tool) error }, lspManager *lsp.Manager) error {
	lspTools := []tools.Tool{
		NewDocumentSymbolsTool(lspManager),
		NewWorkspaceSymbolsTool(lspManager),
		NewCodeActionsTool(lspManager),
		NewRenameSymbolTool(lspManager),
		NewFormatDocumentTool(lspManager),
	}

	for _, tool := range lspTools {
		if err := registry.Register(tool); err != nil {
			return err
		}
	}

	return nil
}

// GetAllTools returns all available LSP tools
func GetAllTools(lspManager *lsp.Manager) []tools.Tool {
	return []tools.Tool{
		NewDocumentSymbolsTool(lspManager),
		NewWorkspaceSymbolsTool(lspManager),
		NewCodeActionsTool(lspManager),
		NewRenameSymbolTool(lspManager),
		NewFormatDocumentTool(lspManager),
	}
}
