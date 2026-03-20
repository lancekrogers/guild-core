// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
		opts       []ClientOption
		wantErr    bool
		validate   func(*testing.T, *Client)
	}{
		{
			name:       "valid socket path",
			socketPath: "/tmp/test.sock",
			wantErr:    false,
			validate: func(t *testing.T, c *Client) {
				assert.Equal(t, "/tmp/test.sock", c.GetSocketPath())
				assert.NotNil(t, c.httpClient)
				assert.Equal(t, 30*time.Second, c.httpClient.Timeout)
			},
		},
		{
			name:       "with custom timeout",
			socketPath: "/tmp/test.sock",
			opts:       []ClientOption{WithTimeout(5 * time.Second)},
			wantErr:    false,
			validate: func(t *testing.T, c *Client) {
				assert.Equal(t, 5*time.Second, c.httpClient.Timeout)
			},
		},
		{
			name:       "empty socket path",
			socketPath: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.socketPath, tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				if tt.validate != nil {
					tt.validate(t, client)
				}
			}
		})
	}
}

func TestClient_HTTPMethods(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	// Create test server
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back method and body
		w.Header().Set("X-Method", r.Method)
		w.Header().Set("X-URL", r.URL.Path)
		if r.Header.Get("Content-Type") != "" {
			w.Header().Set("X-Content-Type", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create client
	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	t.Run("Get method", func(t *testing.T) {
		resp, err := client.Get(ctx, "http://unix/test/path")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET", resp.Header.Get("X-Method"))
		assert.Equal(t, "/test/path", resp.Header.Get("X-URL"))
	})

	t.Run("Post method", func(t *testing.T) {
		body := strings.NewReader("test body")
		resp, err := client.Post(ctx, "http://unix/test/post", "application/json", body)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "POST", resp.Header.Get("X-Method"))
		assert.Equal(t, "application/json", resp.Header.Get("X-Content-Type"))

		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "test body", string(respBody))
	})

	t.Run("Do method with custom request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "PUT", "http://unix/custom", strings.NewReader("custom body"))
		require.NoError(t, err)

		resp, err := client.Do(ctx, req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "PUT", resp.Header.Get("X-Method"))
	})
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	// Create slow server
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create client with short timeout
	client, err := NewClient(socketPath, WithTimeout(100*time.Millisecond))
	require.NoError(t, err)
	defer client.Close()

	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		resp, err := client.Get(ctx, "http://unix/test")
		require.Error(t, err)
		assert.Nil(t, resp)

		gerr, ok := err.(*gerror.GuildError)
		require.True(t, ok, "expected gerror.Error")
		assert.Equal(t, gerror.ErrCodeCancelled, gerr.Code)
	})

	t.Run("immediate cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		resp, err := client.Get(ctx, "http://unix/test")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "canceled")
	})
}

func TestClient_IsHealthy(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	tests := []struct {
		name        string
		setupServer func() func()
		wantHealthy bool
	}{
		{
			name: "healthy server",
			setupServer: func() func() {
				listener, _ := net.Listen("unix", socketPath)
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" {
						w.WriteHeader(http.StatusOK)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				})
				server := &http.Server{Handler: handler}
				go server.Serve(listener)
				time.Sleep(50 * time.Millisecond)
				return func() {
					server.Close()
					listener.Close()
				}
			},
			wantHealthy: true,
		},
		{
			name: "unhealthy server",
			setupServer: func() func() {
				listener, _ := net.Listen("unix", socketPath)
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				})
				server := &http.Server{Handler: handler}
				go server.Serve(listener)
				time.Sleep(50 * time.Millisecond)
				return func() {
					server.Close()
					listener.Close()
				}
			},
			wantHealthy: false,
		},
		{
			name: "no server",
			setupServer: func() func() {
				// No server setup
				return func() {}
			},
			wantHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupServer()
			defer cleanup()

			client, err := NewClient(socketPath)
			require.NoError(t, err)
			defer client.Close()

			ctx := context.Background()
			healthy := client.IsHealthy(ctx)
			assert.Equal(t, tt.wantHealthy, healthy)
		})
	}
}

func TestClient_ErrorWrapping(t *testing.T) {
	// Test with non-existent socket
	socketPath := "/tmp/nonexistent.sock"

	client, err := NewClient(socketPath)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	t.Run("Get error", func(t *testing.T) {
		resp, err := client.Get(ctx, "http://unix/test")
		require.Error(t, err)
		assert.Nil(t, resp)

		gerr, ok := err.(*gerror.GuildError)
		require.True(t, ok, "expected gerror.Error")
		assert.Equal(t, gerror.ErrCodeConnection, gerr.Code)
		assert.Contains(t, gerr.Error(), "request failed")
	})

	t.Run("Post error", func(t *testing.T) {
		resp, err := client.Post(ctx, "http://unix/test", "application/json", strings.NewReader("test"))
		require.Error(t, err)
		assert.Nil(t, resp)

		gerr, ok := err.(*gerror.GuildError)
		require.True(t, ok, "expected gerror.Error")
		assert.Equal(t, gerror.ErrCodeConnection, gerr.Code)
	})
}
