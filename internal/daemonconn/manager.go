// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemonconn

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Manager handles daemon connections with automatic reconnection
type Manager struct {
	mu           sync.RWMutex
	conn         *grpc.ClientConn
	info         *ConnectionInfo
	ctx          context.Context
	cancel       context.CancelFunc
	reconnectC   chan struct{}
	maintainOnce sync.Once

	// Discovery parameters (used for reconnect)
	useCampaign bool
	campaign    string

	// Reconnection parameters
	minBackoff          time.Duration
	maxBackoff          time.Duration
	healthCheckInterval time.Duration
}

// NewManager creates a new connection manager
func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	return &Manager{
		ctx:                 ctx,
		cancel:              cancel,
		reconnectC:          make(chan struct{}, 1),
		minBackoff:          1 * time.Second,
		maxBackoff:          30 * time.Second,
		healthCheckInterval: 5 * time.Second,
	}
}

// Connect establishes initial connection to daemon
func (m *Manager) Connect(ctx context.Context) error {
	conn, info, err := Discover(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to discover daemon").
			WithComponent("daemonconn.Manager").
			WithOperation("Connect")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing connection if any
	if m.conn != nil {
		m.conn.Close()
	}

	m.conn = conn
	m.info = info
	m.useCampaign = false
	m.campaign = ""

	// Start connection maintenance goroutine
	m.maintainOnce.Do(func() { go m.maintainConnection() })

	return nil
}

// ConnectForCampaign establishes initial connection to a campaign daemon.
func (m *Manager) ConnectForCampaign(ctx context.Context, campaign string) error {
	conn, info, err := DiscoverForCampaign(ctx, campaign)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to discover daemon").
			WithComponent("daemonconn.Manager").
			WithOperation("ConnectForCampaign").
			WithDetails("campaign", campaign)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing connection if any
	if m.conn != nil {
		m.conn.Close()
	}

	m.conn = conn
	m.info = info
	m.useCampaign = true
	m.campaign = campaign

	// Start connection maintenance goroutine
	m.maintainOnce.Do(func() { go m.maintainConnection() })

	return nil
}

// GetConnection returns the current connection (may be nil)
func (m *Manager) GetConnection() (*grpc.ClientConn, *ConnectionInfo) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conn, m.info
}

// IsConnected checks if connection is available and healthy
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.conn == nil {
		return false
	}

	state := m.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// GetLatency measures connection latency (simple ping test)
func (m *Manager) GetLatency(ctx context.Context) time.Duration {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()

	if conn == nil {
		return 0
	}

	start := time.Now()

	// Simple connectivity check as ping
	state := conn.GetState()
	if state == connectivity.Ready || state == connectivity.Idle {
		return time.Since(start)
	}

	return 0
}

// TriggerReconnect manually triggers a reconnection attempt
func (m *Manager) TriggerReconnect() {
	select {
	case m.reconnectC <- struct{}{}:
	default:
		// Channel already has pending reconnect
	}
}

// Close shuts down the manager and closes connections
func (m *Manager) Close() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		err := m.conn.Close()
		m.conn = nil
		m.info = nil
		return err
	}

	return nil
}

// maintainConnection runs connection health checks and reconnection logic
func (m *Manager) maintainConnection() {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	backoff := m.minBackoff

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-m.reconnectC:
			// Manual reconnect trigger
			m.attemptReconnect()
			backoff = m.minBackoff // Reset backoff on manual trigger

		case <-ticker.C:
			// Regular health check
			if !m.IsConnected() {
				// Connection is unhealthy, attempt reconnect
				if m.attemptReconnect() {
					backoff = m.minBackoff // Reset backoff on success
				} else {
					// Exponential backoff
					time.Sleep(backoff)
					backoff = min(backoff*2, m.maxBackoff)
				}
			}
		}
	}
}

// attemptReconnect tries to establish a new connection
func (m *Manager) attemptReconnect() bool {
	ctx, cancel := context.WithTimeout(m.ctx, DefaultTimeout)
	defer cancel()

	m.mu.RLock()
	useCampaign := m.useCampaign
	campaign := m.campaign
	m.mu.RUnlock()

	var (
		conn *grpc.ClientConn
		info *ConnectionInfo
		err  error
	)
	if useCampaign {
		conn, info, err = DiscoverForCampaign(ctx, campaign)
	} else {
		conn, info, err = Discover(ctx)
	}
	if err != nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close old connection
	if m.conn != nil {
		m.conn.Close()
	}

	m.conn = conn
	m.info = info

	return true
}
