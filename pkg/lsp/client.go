// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// Client represents an LSP client that communicates with a language server
type Client struct {
	// Server info
	language string
	command  []string
	rootURI  string

	// Process management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// Transport
	transport *ClientTransport
	requestID int64

	// State
	initialized bool
	ready       bool
	mu          sync.RWMutex

	// Capabilities
	serverCapabilities ServerCapabilities
}

// NewClient creates a new LSP client for the given language and command
func NewClient(language string, command []string, rootURI string) *Client {
	return &Client{
		language:  language,
		command:   command,
		rootURI:   rootURI,
		requestID: 0,
	}
}

// Start starts the language server process
func (c *Client) Start(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return gerror.New(gerror.ErrCodeAlreadyExists, "language server already started", nil).
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language)
	}

	// Create command
	if len(c.command) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "no command specified for language server", nil).
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language)
	}

	c.cmd = exec.CommandContext(ctx, c.command[0], c.command[1:]...)
	c.cmd.Env = os.Environ()

	// Setup pipes
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stdin pipe").
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stdout pipe").
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stderr pipe").
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language)
	}

	// Start process
	if err := c.cmd.Start(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeExternal, "failed to start language server").
			WithComponent("lsp").
			WithOperation("start").
			WithDetails("language", c.language).
			WithDetails("command", fmt.Sprintf("%v", c.command))
	}

	// Create transport
	c.transport = NewClientTransport(c.stdout, c.stdin)

	// Start listening for server messages in background
	go c.transport.Listen(ctx)

	// Log stderr in background
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					logger.ErrorContext(ctx, "LSP server stderr read error",
						"language", c.language,
						"error", err)
				}
				return
			}
			if n > 0 {
				logger.InfoContext(ctx, "LSP server stderr",
					"language", c.language,
					"output", string(buf[:n]))
			}
		}
	}()

	logger.InfoContext(ctx, "Started language server",
		"language", c.language,
		"command", c.command)

	return nil
}

// Stop stops the language server process
func (c *Client) Stop(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return nil // Already stopped
	}

	// Send shutdown request
	if c.initialized {
		if err := c.shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "Failed to shutdown language server",
				"language", c.language,
				"error", err)
		}
	}

	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	// Kill process if still running
	if c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			logger.ErrorContext(ctx, "Failed to kill language server process",
				"language", c.language,
				"error", err)
		}
		c.cmd.Wait()
	}

	// Reset state
	c.cmd = nil
	c.transport = nil
	c.initialized = false
	c.ready = false

	logger.InfoContext(ctx, "Stopped language server",
		"language", c.language)

	return nil
}

// Initialize sends the initialize request to the language server
func (c *Client) Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil, gerror.New(gerror.ErrCodeAlreadyExists, "client already initialized", nil).
			WithComponent("lsp").
			WithOperation("initialize").
			WithDetails("language", c.language)
	}

	if c.transport == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "client not started", nil).
			WithComponent("lsp").
			WithOperation("initialize").
			WithDetails("language", c.language)
	}

	// Set default root URI if not provided
	if params.RootURI == "" {
		params.RootURI = c.rootURI
	}

	var result InitializeResult
	if err := c.request(ctx, "initialize", params, &result); err != nil {
		return nil, err
	}

	c.serverCapabilities = result.Capabilities
	c.initialized = true

	// Send initialized notification
	if err := c.notify(ctx, "initialized", &InitializedParams{}); err != nil {
		return nil, err
	}

	c.ready = true

	return &result, nil
}

// shutdown sends the shutdown request
func (c *Client) shutdown(ctx context.Context) error {
	var result interface{}
	return c.request(ctx, "shutdown", nil, &result)
}

// request sends a request and waits for response
func (c *Client) request(ctx context.Context, method string, params interface{}, result interface{}) error {
	if c.transport == nil {
		return gerror.New(gerror.ErrCodeValidation, "transport not initialized", nil).
			WithComponent("lsp").
			WithOperation("request").
			WithDetails("method", method)
	}

	return c.transport.Request(ctx, method, params, result)
}

// notify sends a notification (no response expected)
func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	if c.transport == nil {
		return gerror.New(gerror.ErrCodeValidation, "transport not initialized", nil).
			WithComponent("lsp").
			WithOperation("notify").
			WithDetails("method", method)
	}

	return c.transport.Notify(ctx, method, params)
}

// IsReady returns whether the client is ready to handle requests
func (c *Client) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// GetCapabilities returns the server capabilities
func (c *Client) GetCapabilities() ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities
}
