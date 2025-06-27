// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/project"
)

// Manager is the main LSP manager that coordinates language servers
type Manager struct {
	serverManager   *ServerManager
	config          *Config
	projectDetector *project.ProjectDetector
	mu              sync.RWMutex

	// File to server mapping
	fileServers map[string]*Server
}

// NewManager creates a new LSP manager
func NewManager(configPath string) (*Manager, error) {
	// Load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		serverManager:   NewServerManager(config),
		config:          config,
		projectDetector: project.NewProjectDetector(),
		fileServers:     make(map[string]*Server),
	}, nil
}

// GetServerForFile gets or starts a language server for the given file
func (m *Manager) GetServerForFile(ctx context.Context, filePath string) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check cache
	if server, exists := m.fileServers[filePath]; exists {
		return server, nil
	}

	// Detect language by checking configured file patterns
	language, serverConfig := m.detectLanguageFromConfig(filePath)
	if serverConfig == nil {
		// Fallback to extension-based detection
		language = DetectLanguage(filePath)
		if language == "" {
			return nil, gerror.Newf(gerror.ErrCodeValidation, "cannot detect language for file: %s", filePath).
				WithComponent("lsp").
				WithOperation("get_server_for_file").
				WithDetails("file", filePath).
				WithDetails("hint", "No LSP server configured for this file type. Add server configuration to ~/.guild/lsp/config.yaml")
		}

		// Check if we have config for detected language
		var exists bool
		serverConfig, exists = m.config.Servers[language]
		if !exists {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "no server configuration for language: %s", language).
				WithComponent("lsp").
				WithOperation("get_server_for_file").
				WithDetails("language", language).
				WithDetails("hint", fmt.Sprintf("Add %s server configuration to ~/.guild/lsp/config.yaml", language))
		}
	}

	// Find project root
	rootPath, err := FindRootPath(filePath, serverConfig.RootMarkers)
	if err != nil {
		return nil, err
	}

	// Get or start server
	server, err := m.serverManager.GetServer(ctx, language, rootPath)
	if err != nil {
		return nil, err
	}

	// Cache the mapping
	m.fileServers[filePath] = server

	return server, nil
}

// GetCompletion gets code completions for the given file and position
func (m *Manager) GetCompletion(ctx context.Context, filePath string, line, character int, triggerChar string) (*CompletionList, error) {
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

	// Create completion parameters
	params := &CompletionParams{
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

	if triggerChar != "" {
		params.Context = &CompletionContext{
			TriggerKind:      CompletionTriggerKindTriggerCharacter,
			TriggerCharacter: triggerChar,
		}
	}

	// Send completion request
	var result CompletionList
	if err := server.Client.request(ctx, "textDocument/completion", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "completion request failed").
			WithComponent("lsp").
			WithOperation("get_completion").
			WithDetails("file", filePath).
			WithDetails("position", fmt.Sprintf("%d:%d", line, character))
	}

	logger.DebugContext(ctx, "Got completions",
		"file", filePath,
		"position", fmt.Sprintf("%d:%d", line, character),
		"count", len(result.Items))

	return &result, nil
}

// GetDefinition gets the definition location for the symbol at the given position
func (m *Manager) GetDefinition(ctx context.Context, filePath string, line, character int) ([]Location, error) {
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
	params := &TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{
			URI: filePathToURI(filePath),
		},
		Position: Position{
			Line:      line,
			Character: character,
		},
	}

	// Send definition request
	var result []Location
	if err := server.Client.request(ctx, "textDocument/definition", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "definition request failed").
			WithComponent("lsp").
			WithOperation("get_definition").
			WithDetails("file", filePath).
			WithDetails("position", fmt.Sprintf("%d:%d", line, character))
	}

	return result, nil
}

// GetReferences finds all references to the symbol at the given position
func (m *Manager) GetReferences(ctx context.Context, filePath string, line, character int, includeDeclaration bool) ([]Location, error) {
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
	params := &ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: filePathToURI(filePath),
			},
			Position: Position{
				Line:      line,
				Character: character,
			},
		},
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	// Send references request
	var result []Location
	if err := server.Client.request(ctx, "textDocument/references", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "references request failed").
			WithComponent("lsp").
			WithOperation("get_references").
			WithDetails("file", filePath).
			WithDetails("position", fmt.Sprintf("%d:%d", line, character))
	}

	return result, nil
}

// GetHover gets hover information for the symbol at the given position
func (m *Manager) GetHover(ctx context.Context, filePath string, line, character int) (*Hover, error) {
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
	params := &TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{
			URI: filePathToURI(filePath),
		},
		Position: Position{
			Line:      line,
			Character: character,
		},
	}

	// Send hover request
	var result Hover
	if err := server.Client.request(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "hover request failed").
			WithComponent("lsp").
			WithOperation("get_hover").
			WithDetails("file", filePath).
			WithDetails("position", fmt.Sprintf("%d:%d", line, character))
	}

	return &result, nil
}

// ensureFileOpened ensures the file is opened in the language server
func (m *Manager) ensureFileOpened(ctx context.Context, server *Server, filePath string) error {
	// TODO: Track which files are opened per server
	// For now, always open the file

	// Read file content
	content, err := readFile(filePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read file").
			WithComponent("lsp").
			WithOperation("ensure_file_opened").
			WithDetails("file", filePath)
	}

	// Send didOpen notification
	params := &DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        filePathToURI(filePath),
			LanguageID: server.Language,
			Version:    1,
			Text:       content,
		},
	}

	if err := server.Client.notify(ctx, "textDocument/didOpen", params); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeExternal, "failed to open file in language server").
			WithComponent("lsp").
			WithOperation("ensure_file_opened").
			WithDetails("file", filePath)
	}

	return nil
}

// Shutdown shuts down all language servers
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear file cache
	m.fileServers = make(map[string]*Server)

	// Stop all servers
	return m.serverManager.StopAll(ctx)
}

// GetActiveServers returns information about active language servers
func (m *Manager) GetActiveServers() []ActiveServerInfo {
	return m.serverManager.GetActiveServers()
}

// filePathToURI converts a file path to a URI
func filePathToURI(path string) string {
	absPath, _ := filepath.Abs(path)
	return fmt.Sprintf("file://%s", absPath)
}

// readFile reads the content of a file
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// detectLanguageFromConfig detects language based on configured file patterns
func (m *Manager) detectLanguageFromConfig(filePath string) (string, *ServerConfig) {
	fileName := filepath.Base(filePath)

	// Check each configured server's file patterns
	for lang, config := range m.config.Servers {
		for _, pattern := range config.FilePatterns {
			matched, err := filepath.Match(pattern, fileName)
			if err == nil && matched {
				return lang, config
			}
		}
	}

	return "", nil
}
