package lsp

import (
	"github.com/guild-ventures/guild-core/pkg/lsp"
	"github.com/guild-ventures/guild-core/tools"
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