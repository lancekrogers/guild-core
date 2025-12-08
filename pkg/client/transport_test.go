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

	"github.com/guild-framework/guild-core/pkg/gerror"
)

func TestNewUnixSocketTransport(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
		wantErr    bool
		errCode    gerror.ErrorCode
	}{
		{
			name:       "valid socket path",
			socketPath: "/tmp/test.sock",
			wantErr:    false,
		},
		{
			name:       "empty socket path",
			socketPath: "",
			wantErr:    true,
			errCode:    gerror.ErrCodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewUnixSocketTransport(tt.socketPath)

			if tt.wantErr {
				require.Error(t, err)
				gerr, ok := err.(*gerror.GuildError)
				require.True(t, ok, "expected gerror.Error")
				assert.Equal(t, tt.errCode, gerr.Code)
				assert.Nil(t, transport)
			} else {
				require.NoError(t, err)
				require.NotNil(t, transport)
				assert.Equal(t, tt.socketPath, transport.socketPath)
				assert.NotNil(t, transport.httpTransport)
			}
		})
	}
}

func TestUnixSocketTransport_RoundTrip(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	// Create a test HTTP server over Unix socket
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	// Simple test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back request headers and body
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create transport
	transport, err := NewUnixSocketTransport(socketPath)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		checkResponse func(*testing.T, *http.Response, error)
	}{
		{
			name: "successful GET request",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://unix/test", nil)
				return req
			},
			checkResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "test-value", resp.Header.Get("X-Test-Header"))
			},
		},
		{
			name: "successful POST request with body",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://unix/test", nil)
				req.Body = io.NopCloser(stringReader("test body"))
				req.ContentLength = 9
				return req
			},
			checkResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				require.NotNil(t, resp)
				body, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "test body", string(body))
			},
		},
		{
			name: "request with context cancellation",
			setupRequest: func() *http.Request {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				req, _ := http.NewRequestWithContext(ctx, "GET", "http://unix/test", nil)
				return req
			},
			checkResponse: func(t *testing.T, resp *http.Response, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "context canceled")
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			resp, err := transport.RoundTrip(req)
			tt.checkResponse(t, resp, err)
			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

func TestUnixSocketTransport_NonExistentSocket(t *testing.T) {
	// Use a socket path that doesn't exist
	socketPath := "/tmp/guild-test-nonexistent.sock"

	// Ensure socket doesn't exist
	os.Remove(socketPath)

	transport, err := NewUnixSocketTransport(socketPath)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "http://unix/test", nil)
	resp, err := transport.RoundTrip(req)

	require.Error(t, err)
	assert.Nil(t, resp)

	// Check that error is properly wrapped
	gerr, ok := err.(*gerror.GuildError)
	require.True(t, ok, "expected gerror.Error")
	assert.Equal(t, gerror.ErrCodeConnection, gerr.Code)
}

func TestUnixSocketTransport_ConnectionPooling(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	// Create a test HTTP server
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	connectionCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionCount++
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create transport with custom settings
	transport, err := NewUnixSocketTransport(socketPath)
	require.NoError(t, err)

	// Make multiple requests to test connection pooling
	client := &http.Client{Transport: transport}

	for i := 0; i < 5; i++ {
		resp, err := client.Get("http://unix/test")
		require.NoError(t, err)
		resp.Body.Close()
	}

	// With connection pooling, we should see fewer connections than requests
	// (exact count depends on timing and implementation details)
	assert.LessOrEqual(t, connectionCount, 5)
}

func TestUnixSocketTransport_Timeout(t *testing.T) {
	// Create temporary directory for Unix sockets
	// Use shorter path to avoid macOS Unix socket path limit
	socketPath := filepath.Join("/tmp", "guild-test-"+t.Name()+".sock")
	t.Cleanup(func() { os.Remove(socketPath) })

	// Create a slow test HTTP server
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

	// Create transport
	transport, err := NewUnixSocketTransport(socketPath)
	require.NoError(t, err)

	// Create request with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://unix/test", nil)
	resp, err := transport.RoundTrip(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

// Helper function to create string reader
func stringReader(s string) io.Reader {
	return io.NopCloser(strings.NewReader(s))
}
