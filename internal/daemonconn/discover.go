// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package daemonconn provides daemon discovery and gRPC connection management for the Guild CLI
package daemonconn

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

const (
	// Default connection parameters
	DefaultUnixSocket = "/tmp/guild.sock"
	DefaultTCPPort    = "7600"
	DefaultTimeout    = 2 * time.Second
)

// ConnectionInfo holds information about a daemon connection
type ConnectionInfo struct {
	Address string
	Type    string // "unix" or "tcp"
}

// Discover attempts to find and connect to a running Guild daemon
// It tries Unix socket first, then TCP, with optional env var override
func Discover(ctx context.Context) (*grpc.ClientConn, *ConnectionInfo, error) {
	// Check for environment override first
	if addr := os.Getenv("GUILD_DAEMON_ADDR"); addr != "" {
		conn, err := connectTCP(ctx, addr)
		if err != nil {
			return nil, nil, gerror.Wrap(err, gerror.ErrCodeConnection,
				"failed to connect to daemon at override address").
				WithComponent("daemonconn").
				WithOperation("Discover").
				WithDetails("address", addr)
		}

		info := &ConnectionInfo{
			Address: addr,
			Type:    "tcp",
		}
		return conn, info, nil
	}

	// Try Unix socket first
	if _, err := os.Stat(DefaultUnixSocket); err == nil {
		conn, err := connectUnix(ctx, DefaultUnixSocket)
		if err == nil {
			info := &ConnectionInfo{
				Address: DefaultUnixSocket,
				Type:    "unix",
			}
			return conn, info, nil
		}
		// Unix socket exists but connection failed - continue to TCP
	}

	// Fall back to TCP
	tcpAddr := "localhost:" + DefaultTCPPort
	conn, err := connectTCP(ctx, tcpAddr)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeConnection,
			"failed to connect to daemon via Unix socket or TCP").
			WithComponent("daemonconn").
			WithOperation("Discover").
			WithDetails("unix_socket", DefaultUnixSocket).
			WithDetails("tcp_address", tcpAddr)
	}

	info := &ConnectionInfo{
		Address: tcpAddr,
		Type:    "tcp",
	}
	return conn, info, nil
}

// connectUnix establishes connection to Unix socket
func connectUnix(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, "unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "Unix socket connection failed").
			WithComponent("daemonconn").
			WithOperation("connectUnix").
			WithDetails("socket_path", socketPath)
	}

	return conn, nil
}

// connectTCP establishes connection to TCP address
func connectTCP(ctx context.Context, address string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "TCP connection failed").
			WithComponent("daemonconn").
			WithOperation("connectTCP").
			WithDetails("address", address)
	}

	return conn, nil
}

// FormatConnectionStatus returns a human-readable connection status string
func FormatConnectionStatus(info *ConnectionInfo, latency time.Duration) string {
	if info == nil {
		return "🔴 Offline"
	}

	icon := "🟢"
	var displayAddr string

	switch info.Type {
	case "unix":
		displayAddr = "unix socket"
	case "tcp":
		displayAddr = info.Address
	default:
		displayAddr = info.Address
	}

	return fmt.Sprintf("%s Connected to %s (%dms)",
		icon, displayAddr, latency.Milliseconds())
}
