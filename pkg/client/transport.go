// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package client provides HTTP client utilities for connecting to Guild daemon instances via Unix sockets
package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// UnixSocketTransport implements http.RoundTripper for Unix domain socket connections
type UnixSocketTransport struct {
	socketPath    string
	httpTransport *http.Transport
}

// NewUnixSocketTransport creates a new HTTP transport that connects via Unix socket
func NewUnixSocketTransport(socketPath string) (*UnixSocketTransport, error) {
	if socketPath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "socket path cannot be empty", nil).
			WithComponent("client").
			WithOperation("NewUnixSocketTransport")
	}

	// Create custom dialer for Unix sockets
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Configure HTTP transport with Unix socket dialer
	transport := &http.Transport{
		// Override DialContext to use Unix sockets
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Always use Unix socket regardless of the URL
			conn, err := dialer.DialContext(ctx, "unix", socketPath)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to Unix socket").
					WithComponent("client").
					WithOperation("UnixSocketTransport.DialContext").
					WithDetails("socket", socketPath).
					FromContext(ctx)
			}
			return conn, nil
		},
		// Connection pooling settings
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Response header timeout
		ResponseHeaderTimeout: 30 * time.Second,
		// Disable HTTP/2 for Unix sockets
		ForceAttemptHTTP2: false,
	}

	return &UnixSocketTransport{
		socketPath:    socketPath,
		httpTransport: transport,
	}, nil
}

// RoundTrip implements the http.RoundTripper interface
func (t *UnixSocketTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure context is propagated
	if req.Context() == nil {
		ctx := context.Background()
		req = req.WithContext(ctx)
	}

	// Perform the request
	resp, err := t.httpTransport.RoundTrip(req)
	if err != nil {
		// Check if error is due to context cancellation
		if req.Context().Err() != nil {
			return nil, gerror.Wrap(req.Context().Err(), gerror.ErrCodeCancelled, "request canceled").
				WithComponent("client").
				WithOperation("UnixSocketTransport.RoundTrip").
				WithDetails("socket", t.socketPath).
				WithDetails("url", req.URL.String()).
				FromContext(req.Context())
		}

		// Wrap other errors
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "HTTP request failed").
			WithComponent("client").
			WithOperation("UnixSocketTransport.RoundTrip").
			WithDetails("socket", t.socketPath).
			WithDetails("method", req.Method).
			WithDetails("url", req.URL.String())
	}

	return resp, nil
}

// Close closes idle connections
func (t *UnixSocketTransport) Close() error {
	t.httpTransport.CloseIdleConnections()
	return nil
}

// GetSocketPath returns the Unix socket path this transport connects to
func (t *UnixSocketTransport) GetSocketPath() string {
	return t.socketPath
}

// Client creates a new HTTP client configured to use this Unix socket transport
type Client struct {
	httpClient *http.Client
	socketPath string
}

// NewClient creates a new HTTP client for Unix socket connections
func NewClient(socketPath string, opts ...ClientOption) (*Client, error) {
	transport, err := NewUnixSocketTransport(socketPath)
	if err != nil {
		return nil, err
	}

	client := &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // Default timeout
		},
		socketPath: socketPath,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption configures a Client
type ClientOption func(*Client)

// WithTimeout sets the client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient allows using a custom http.Client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// Do performs an HTTP request with context support
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Ensure context is set
	if req.Context() == nil {
		req = req.WithContext(ctx)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check for context cancellation
		if ctx.Err() != nil {
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "request canceled").
				WithComponent("client").
				WithOperation("Client.Do").
				WithDetails("socket", c.socketPath).
				FromContext(ctx)
		}

		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "request failed").
			WithComponent("client").
			WithOperation("Client.Do").
			WithDetails("socket", c.socketPath).
			WithDetails("method", req.Method).
			WithDetails("url", req.URL.String())
	}

	return resp, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to create request").
			WithComponent("client").
			WithOperation("Client.Get").
			WithDetails("url", url)
	}

	return c.Do(ctx, req)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to create request").
			WithComponent("client").
			WithOperation("Client.Post").
			WithDetails("url", url)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.Do(ctx, req)
}

// Close closes idle connections
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetSocketPath returns the Unix socket path
func (c *Client) GetSocketPath() string {
	return c.socketPath
}

// IsHealthy checks if the daemon is responsive
func (c *Client) IsHealthy(ctx context.Context) bool {
	// Attempt a simple health check request
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.Get(ctx, "http://unix/health")
	if err != nil {
		return false
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return resp.StatusCode == http.StatusOK
}
