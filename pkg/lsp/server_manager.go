package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// ServerManager manages language server instances
type ServerManager struct {
	config  *Config
	servers map[string]*Server
	mu      sync.RWMutex
}

// Server represents a managed language server instance
type Server struct {
	Language  string
	Config    *ServerConfig
	Client    *Client
	Workspace string
	Ready     bool
	StartTime time.Time
	LastUsed  time.Time
}

// NewServerManager creates a new server manager
func NewServerManager(config *Config) *ServerManager {
	if config == nil {
		config = &Config{
			Servers: DefaultConfigs(),
		}
	}

	return &ServerManager{
		config:  config,
		servers: make(map[string]*Server),
	}
}

// GetServer gets or starts a language server for the given language
func (m *ServerManager) GetServer(ctx context.Context, language string, workspace string) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server already exists for this language and workspace
	key := fmt.Sprintf("%s:%s", language, workspace)
	if server, exists := m.servers[key]; exists {
		server.LastUsed = time.Now()
		if server.Ready {
			return server, nil
		}
		// Server exists but not ready, wait a bit
		if time.Since(server.StartTime) < 5*time.Second {
			m.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
			m.mu.Lock()
			if server.Ready {
				return server, nil
			}
		}
	}

	// Get server config
	config, exists := m.config.Servers[language]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no configuration for language: %s", language).
			WithComponent("lsp").
			WithOperation("get_server").
			WithDetails("language", language)
	}

	// Create and start new server
	server := &Server{
		Language:  language,
		Config:    config,
		Workspace: workspace,
		StartTime: time.Now(),
		LastUsed:  time.Now(),
	}

	// Create client
	rootURI := fmt.Sprintf("file://%s", workspace)
	server.Client = NewClient(language, config.Command, rootURI)

	// Start the server
	if err := server.Client.Start(ctx); err != nil {
		return nil, err
	}

	// Initialize the server
	processID := int64(os.Getpid())
	initParams := InitializeParams{
		ProcessID: &processID,
		ClientInfo: &ClientInfo{
			Name:    "guild-lsp-client",
			Version: "1.0.0",
		},
		RootURI:               rootURI,
		InitializationOptions: config.InitOptions,
		Capabilities: ClientCapabilities{
			Workspace: &WorkspaceClientCapabilities{
				ApplyEdit: true,
			},
			TextDocument: &TextDocumentClientCapabilities{
				Completion: &CompletionClientCapabilities{
					DynamicRegistration: false,
					CompletionItem: &CompletionItemCapabilities{
						SnippetSupport:      true,
						DeprecatedSupport:   true,
						PreselectSupport:    true,
						DocumentationFormat: []string{"plaintext", "markdown"},
					},
				},
			},
		},
		WorkspaceFolders: []WorkspaceFolder{
			{
				URI:  rootURI,
				Name: filepath.Base(workspace),
			},
		},
	}

	_, err := server.Client.Initialize(ctx, initParams)
	if err != nil {
		server.Client.Stop(ctx)
		return nil, gerror.Wrap(err, gerror.ErrCodeExternal, "failed to initialize language server").
			WithComponent("lsp").
			WithOperation("initialize").
			WithDetails("language", language)
	}

	server.Ready = true
	m.servers[key] = server

	observability.GetLogger(ctx).InfoContext(ctx, "Started language server",
		"language", language,
		"workspace", workspace,
		"command", config.Command)

	return server, nil
}

// StopServer stops a specific language server
func (m *ServerManager) StopServer(ctx context.Context, language string, workspace string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", language, workspace)
	server, exists := m.servers[key]
	if !exists {
		return nil // Already stopped
	}

	if server.Client != nil {
		if err := server.Client.Stop(ctx); err != nil {
			observability.GetLogger(ctx).ErrorContext(ctx, "Failed to stop language server",
				"language", language,
				"workspace", workspace,
				"error", err)
		}
	}

	delete(m.servers, key)

	observability.GetLogger(ctx).InfoContext(ctx, "Stopped language server",
		"language", language,
		"workspace", workspace)

	return nil
}

// StopAll stops all language servers
func (m *ServerManager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for key, server := range m.servers {
		if server.Client != nil {
			if err := server.Client.Stop(ctx); err != nil {
				lastErr = err
				observability.GetLogger(ctx).ErrorContext(ctx, "Failed to stop language server",
					"key", key,
					"error", err)
			}
		}
	}

	m.servers = make(map[string]*Server)

	return lastErr
}

// GetActiveServers returns information about active servers
func (m *ServerManager) GetActiveServers() []ActiveServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var servers []ActiveServerInfo
	for key, server := range m.servers {
		servers = append(servers, ActiveServerInfo{
			Key:       key,
			Language:  server.Language,
			Workspace: server.Workspace,
			Ready:     server.Ready,
			StartTime: server.StartTime,
			LastUsed:  server.LastUsed,
		})
	}

	return servers
}

// ActiveServerInfo represents information about an active server
type ActiveServerInfo struct {
	Key       string
	Language  string
	Workspace string
	Ready     bool
	StartTime time.Time
	LastUsed  time.Time
}

// CleanupIdleServers stops servers that haven't been used recently
func (m *ServerManager) CleanupIdleServers(ctx context.Context, idleTimeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger := observability.GetLogger(ctx)
	now := time.Now()
	var toRemove []string

	for key, server := range m.servers {
		if now.Sub(server.LastUsed) > idleTimeout {
			toRemove = append(toRemove, key)
		}
	}

	var lastErr error
	for _, key := range toRemove {
		server := m.servers[key]
		if server.Client != nil {
			if err := server.Client.Stop(ctx); err != nil {
				lastErr = err
				logger.ErrorContext(ctx, "Failed to stop idle language server",
					"key", key,
					"error", err)
			} else {
				logger.InfoContext(ctx, "Stopped idle language server",
					"key", key,
					"idle_time", now.Sub(server.LastUsed))
			}
		}
		delete(m.servers, key)
	}

	return lastErr
}

// CheckServerHealth checks if a server is healthy
func (m *ServerManager) CheckServerHealth(ctx context.Context, language string, workspace string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", language, workspace)
	server, exists := m.servers[key]
	if !exists {
		return false, nil
	}

	if !server.Ready {
		return false, nil
	}

	// TODO: Implement actual health check (e.g., send a simple request)
	return server.Client.IsReady(), nil
}
