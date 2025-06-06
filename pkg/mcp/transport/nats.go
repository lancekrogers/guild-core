package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/nats-io/nats.go"
)

// NATSTransport implements Transport using NATS
type NATSTransport struct {
	conn          *nats.Conn
	js            nats.JetStreamContext
	config        *TransportConfig
	mu            sync.RWMutex
	closed        bool
	subscriptions map[string]*natsSubscription
	receiveCh     chan []byte
}

// natsSubscription holds subscription info
type natsSubscription struct {
	sub    *nats.Subscription
	dataCh chan []byte
	cancel context.CancelFunc
}

// NewNATSTransport creates a new NATS transport
func NewNATSTransport(config *TransportConfig) (*NATSTransport, error) {
	if config == nil {
		config = &TransportConfig{
			Address:        nats.DefaultURL,
			ConnectTimeout: 10 * time.Second,
			MaxReconnects:  -1,
			ReconnectWait:  time.Second,
		}
	}

	transport := &NATSTransport{
		config:        config,
		subscriptions: make(map[string]*natsSubscription),
		receiveCh:     make(chan []byte, 100),
	}

	return transport, nil
}

// Connect establishes a connection to NATS
func (t *NATSTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil && t.conn.IsConnected() {
		return nil // Already connected
	}

	// Set up connection options
	opts := t.buildConnectionOptions()

	// Connect to NATS
	conn, err := nats.Connect(t.config.Address, opts...)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_transport").WithComponent("connect").WithOperation("failed to connect to NATS")
	}

	t.conn = conn

	// Get JetStream context if available
	js, err := conn.JetStream()
	if err == nil {
		t.js = js
		// Create MCP stream if it doesn't exist
		t.ensureStream()
	}

	return nil
}

// Disconnect closes the connection
func (t *NATSTransport) Disconnect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true

	// Cancel all subscriptions
	for _, sub := range t.subscriptions {
		if sub.cancel != nil {
			sub.cancel()
		}
		if sub.sub != nil {
			sub.sub.Unsubscribe()
		}
		close(sub.dataCh)
	}
	t.subscriptions = make(map[string]*natsSubscription)

	// Close connection
	if t.conn != nil {
		t.conn.Close()
	}

	close(t.receiveCh)

	return nil
}

// Send sends a message to a topic
func (t *NATSTransport) Send(ctx context.Context, topic string, data []byte) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("send").WithOperation("transport is closed")
	}
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil || !conn.IsConnected() {
		return gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("send").WithOperation("not connected to NATS")
	}

	// Create NATS message
	msg := &nats.Msg{
		Subject: topic,
		Data:    data,
		Header:  nats.Header{},
	}

	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		msg.Header.Set("X-Trace-ID", fmt.Sprintf("%v", traceID))
	}

	// Send with context
	return conn.PublishMsg(msg)
}

// Receive returns a channel for receiving messages
func (t *NATSTransport) Receive(ctx context.Context) <-chan []byte {
	return t.receiveCh
}

// Request sends a request and waits for response
func (t *NATSTransport) Request(ctx context.Context, topic string, data []byte) ([]byte, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return nil, gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("receive").WithOperation("transport is closed")
	}
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil || !conn.IsConnected() {
		return nil, gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("receive").WithOperation("not connected to NATS")
	}

	// Create timeout from context
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	// Send request
	reply, err := conn.Request(topic, data, timeout)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_transport").WithComponent("request").WithOperation("request failed")
	}

	return reply.Data, nil
}

// Subscribe subscribes to a topic and returns a channel
func (t *NATSTransport) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("receive").WithOperation("transport is closed")
	}

	if t.conn == nil || !t.conn.IsConnected() {
		return nil, gerror.New(gerror.ErrCodeInternal, "mcp_transport", nil).WithComponent("receive").WithOperation("not connected to NATS")
	}

	// Check if already subscribed
	if existing, exists := t.subscriptions[topic]; exists {
		return existing.dataCh, nil
	}

	// Create channel for this subscription
	dataCh := make(chan []byte, 100)
	
	// Create context for managing subscription
	subCtx, cancel := context.WithCancel(context.Background())

	// Subscribe
	sub, err := t.conn.Subscribe(topic, func(msg *nats.Msg) {
		select {
		case dataCh <- msg.Data:
		case <-subCtx.Done():
			// Subscription cancelled, drop message
		default:
			// Channel full, drop message
		}
	})
	if err != nil {
		cancel()
		close(dataCh)
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_transport").WithComponent("subscribe").WithOperation("failed to subscribe")
	}

	// Store subscription
	t.subscriptions[topic] = &natsSubscription{
		sub:    sub,
		dataCh: dataCh,
		cancel: cancel,
	}

	// Handle context cancellation
	go func() {
		select {
		case <-ctx.Done():
			t.Unsubscribe(context.Background(), topic)
		case <-subCtx.Done():
			// Already cancelled
		}
	}()

	return dataCh, nil
}

// Unsubscribe unsubscribes from a topic
func (t *NATSTransport) Unsubscribe(ctx context.Context, topic string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	sub, exists := t.subscriptions[topic]
	if !exists {
		return nil // Not subscribed
	}

	// Cancel context
	if sub.cancel != nil {
		sub.cancel()
	}

	// Unsubscribe
	if sub.sub != nil {
		if err := sub.sub.Unsubscribe(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_transport").WithComponent("unsubscribe").WithOperation("failed to unsubscribe")
		}
	}

	// Close channel
	close(sub.dataCh)

	// Remove from map
	delete(t.subscriptions, topic)

	return nil
}

// Publish publishes to a topic
func (t *NATSTransport) Publish(ctx context.Context, topic string, data []byte) error {
	// For NATS, Publish is the same as Send
	return t.Send(ctx, topic, data)
}

// buildConnectionOptions builds NATS connection options from config
func (t *NATSTransport) buildConnectionOptions() []nats.Option {
	opts := []nats.Option{
		nats.Name(t.config.ClientID),
		nats.Timeout(t.config.ConnectTimeout),
		nats.MaxReconnects(t.config.MaxReconnects),
		nats.ReconnectWait(t.config.ReconnectWait),
		nats.ReconnectJitter(time.Second, 10*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				// Log disconnection
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			// Log reconnection
		}),
	}

	// Add TLS if configured
	if t.config.TLSConfig != nil {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if t.config.TLSConfig.CAFile != "" {
			opts = append(opts, nats.RootCAs(t.config.TLSConfig.CAFile))
		}

		if t.config.TLSConfig.CertFile != "" && t.config.TLSConfig.KeyFile != "" {
			opts = append(opts, nats.ClientCert(t.config.TLSConfig.CertFile, t.config.TLSConfig.KeyFile))
		}

		opts = append(opts, nats.Secure(tlsConfig))
	}

	// Add authentication
	switch t.config.AuthType {
	case "token":
		if t.config.AuthToken != "" {
			opts = append(opts, nats.Token(t.config.AuthToken))
		}
	}

	return opts
}

// ensureStream creates the MCP JetStream stream if it doesn't exist
func (t *NATSTransport) ensureStream() error {
	if t.js == nil {
		return nil
	}

	_, err := t.js.StreamInfo("MCP")
	if err == nil {
		return nil // Stream exists
	}

	// Create stream
	_, err = t.js.AddStream(&nats.StreamConfig{
		Name:      "MCP",
		Subjects:  []string{"mcp.>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		MaxBytes:  1024 * 1024 * 1024, // 1GB
	})

	return err
}

// IsConnected returns whether the transport is connected
func (t *NATSTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return !t.closed && t.conn != nil && t.conn.IsConnected()
}