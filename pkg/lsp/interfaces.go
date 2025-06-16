// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"context"
	"time"
)

// ClientInterface defines the contract for LSP client implementations
// This allows for mock implementations in tests
type ClientInterface interface {
	// Start initializes the LSP client
	Start(ctx context.Context) error

	// Stop shuts down the LSP client
	Stop(ctx context.Context) error

	// Initialize sends the initialize request to the LSP server
	Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error)

	// IsReady returns whether the client is ready to handle requests
	IsReady() bool

	// GetCapabilities returns the server capabilities
	GetCapabilities() ServerCapabilities

	// Completion sends a completion request
	Completion(ctx context.Context, params CompletionParams) (*CompletionList, error)

	// Hover sends a hover request
	Hover(ctx context.Context, params HoverParams) (*Hover, error)

	// Definition sends a definition request
	Definition(ctx context.Context, params DefinitionParams) ([]Location, error)

	// References sends a references request
	References(ctx context.Context, params ReferenceParams) ([]Location, error)

	// DocumentSymbols sends a document symbols request
	DocumentSymbols(ctx context.Context, params DocumentSymbolParams) ([]DocumentSymbol, error)

	// CodeAction sends a code action request
	CodeAction(ctx context.Context, params CodeActionParams) ([]CodeAction, error)

	// ExecuteCommand sends an execute command request
	ExecuteCommand(ctx context.Context, params ExecuteCommandParams) (interface{}, error)

	// WorkspaceSymbol sends a workspace symbol request
	WorkspaceSymbol(ctx context.Context, params WorkspaceSymbolParams) ([]SymbolInformation, error)

	// SignatureHelp sends a signature help request
	SignatureHelp(ctx context.Context, params SignatureHelpParams) (*SignatureHelp, error)

	// CodeLens sends a code lens request
	CodeLens(ctx context.Context, params CodeLensParams) ([]CodeLens, error)

	// DocumentHighlight sends a document highlight request
	DocumentHighlight(ctx context.Context, params DocumentHighlightParams) ([]DocumentHighlight, error)

	// Rename sends a rename request
	Rename(ctx context.Context, params RenameParams) (*WorkspaceEdit, error)

	// Formatting sends a formatting request
	Formatting(ctx context.Context, params DocumentFormattingParams) ([]TextEdit, error)
}

// ServerManagerInterface defines the contract for LSP server management
type ServerManagerInterface interface {
	// GetServer gets or creates a language server for the given language and workspace
	GetServer(ctx context.Context, language string, workspace string) (*Server, error)

	// StopServer stops a specific language server
	StopServer(ctx context.Context, language string, workspace string) error

	// StopAll stops all running language servers
	StopAll(ctx context.Context) error

	// GetActiveServers returns information about all active servers
	GetActiveServers() []ActiveServerInfo

	// CleanupIdleServers removes servers that have been idle for too long
	CleanupIdleServers(ctx context.Context, idleTimeout time.Duration) error

	// CheckServerHealth verifies a server is healthy
	CheckServerHealth(ctx context.Context, language string, workspace string) (bool, error)
}

// ProcessLauncherInterface abstracts the process launching functionality
// This allows mocking the exec.Cmd functionality in tests
type ProcessLauncherInterface interface {
	// LaunchServer starts an LSP server process
	LaunchServer(ctx context.Context, command string, args []string, workDir string) (ClientInterface, error)
}
