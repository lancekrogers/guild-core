// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/lsp"
)

// MockLSPClient is a mock implementation of ClientInterface for testing
type MockLSPClient struct {
	mu           sync.RWMutex
	started      bool
	ready        bool
	capabilities lsp.ServerCapabilities

	// Error injection
	startError      error
	initializeError error

	// Response tracking
	completionCalls int
	hoverCalls      int
	definitionCalls int
	referencesCalls int
}

// NewMockLSPClient creates a new mock LSP client
func NewMockLSPClient() *MockLSPClient {
	return &MockLSPClient{
		ready: true,
		capabilities: lsp.ServerCapabilities{
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{".", ":", ">"},
			},
			HoverProvider:      true,
			DefinitionProvider: true,
			ReferencesProvider: true,
		},
	}
}

// SetStartError sets an error to be returned by Start
func (m *MockLSPClient) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startError = err
}

// SetInitializeError sets an error to be returned by Initialize
func (m *MockLSPClient) SetInitializeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initializeError = err
}

// SetReady sets the ready state of the client
func (m *MockLSPClient) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

// GetCompletionCalls returns the number of completion calls
func (m *MockLSPClient) GetCompletionCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.completionCalls
}

// GetHoverCalls returns the number of hover calls
func (m *MockLSPClient) GetHoverCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hoverCalls
}

// Start implements ClientInterface
func (m *MockLSPClient) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.startError != nil {
		return m.startError
	}

	m.started = true
	return nil
}

// Stop implements ClientInterface
func (m *MockLSPClient) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.started = false
	m.ready = false
	return nil
}

// IsReady implements ClientInterface
func (m *MockLSPClient) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// GetCapabilities implements ClientInterface
func (m *MockLSPClient) GetCapabilities() lsp.ServerCapabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities
}

// Initialize implements ClientInterface
func (m *MockLSPClient) Initialize(ctx context.Context, params lsp.InitializeParams) (*lsp.InitializeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initializeError != nil {
		return nil, m.initializeError
	}

	return &lsp.InitializeResult{
		Capabilities: m.capabilities,
	}, nil
}

// Completion implements ClientInterface
func (m *MockLSPClient) Completion(ctx context.Context, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.completionCalls++
	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        []lsp.CompletionItem{},
	}, nil
}

// Hover implements ClientInterface
func (m *MockLSPClient) Hover(ctx context.Context, params lsp.HoverParams) (*lsp.Hover, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hoverCalls++
	return &lsp.Hover{
		Contents: "Mock hover content",
	}, nil
}

// Definition implements ClientInterface
func (m *MockLSPClient) Definition(ctx context.Context, params lsp.DefinitionParams) ([]lsp.Location, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.definitionCalls++
	return []lsp.Location{}, nil
}

// References implements ClientInterface
func (m *MockLSPClient) References(ctx context.Context, params lsp.ReferenceParams) ([]lsp.Location, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.referencesCalls++
	return []lsp.Location{}, nil
}

// Method stubs for remaining interface methods
func (m *MockLSPClient) DocumentSymbols(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	return []lsp.DocumentSymbol{}, nil
}

func (m *MockLSPClient) CodeAction(ctx context.Context, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	return []lsp.CodeAction{}, nil
}

func (m *MockLSPClient) ExecuteCommand(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	return nil, nil
}

func (m *MockLSPClient) WorkspaceSymbol(ctx context.Context, params lsp.WorkspaceSymbolParams) ([]lsp.SymbolInformation, error) {
	return []lsp.SymbolInformation{}, nil
}

func (m *MockLSPClient) SignatureHelp(ctx context.Context, params lsp.SignatureHelpParams) (*lsp.SignatureHelp, error) {
	return &lsp.SignatureHelp{}, nil
}

func (m *MockLSPClient) CodeLens(ctx context.Context, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	return []lsp.CodeLens{}, nil
}

func (m *MockLSPClient) DocumentHighlight(ctx context.Context, params lsp.DocumentHighlightParams) ([]lsp.DocumentHighlight, error) {
	return []lsp.DocumentHighlight{}, nil
}

func (m *MockLSPClient) Rename(ctx context.Context, params lsp.RenameParams) (*lsp.WorkspaceEdit, error) {
	return &lsp.WorkspaceEdit{}, nil
}

func (m *MockLSPClient) Formatting(ctx context.Context, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	return []lsp.TextEdit{}, nil
}
