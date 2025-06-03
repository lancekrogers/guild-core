package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
)

// NATSTransport implements Transport using NATS
type NATSTransport struct {
	conn      *nats.Conn
	js        nats.JetStreamContext
	config    *TransportConfig
	codec     Codec
	mu        sync.RWMutex
	closed    bool
	
	// For request-response pattern
	inbox     string
	sub       *nats.Subscription
	responses chan *nats.Msg
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
		config:    config,
		codec:     NewDefaultCodec(),
		responses: make(chan *nats.Msg, 100),
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
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	
	t.conn = conn
	
	// Get JetStream context if available
	js, err := conn.JetStream()
	if err == nil {
		t.js = js
		// Create MCP stream if it doesn't exist
		t.ensureStream()
	}
	
	// Set up inbox for request-response
	t.inbox = nats.NewInbox()
	sub, err := conn.Subscribe(t.inbox, func(msg *nats.Msg) {
		select {
		case t.responses <- msg:
		default:
			// Drop message if channel is full
		}
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to subscribe to inbox: %w", err)
	}
	t.sub = sub
	
	return nil
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
		Name:     "MCP",
		Subjects: []string{"mcp.>"},
		Storage:  nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:   24 * time.Hour,
		MaxBytes: 1024 * 1024 * 1024, // 1GB
	})
	
	return err
}

// Send sends a message through NATS
func (t *NATSTransport) Send(ctx context.Context, msg interface{}) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return fmt.Errorf("transport is closed")
	}
	conn := t.conn
	t.mu.RUnlock()
	
	// Encode message
	data, err := t.codec.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	
	// Determine subject based on message type
	subject := t.getSubjectForMessage(msg)
	
	// Create NATS message
	natsMsg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	
	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		natsMsg.Header.Set("X-Trace-ID", fmt.Sprintf("%v", traceID))
	}
	
	// Send with context
	return conn.PublishMsg(natsMsg)
}

// Receive receives a message from NATS
func (t *NATSTransport) Receive(ctx context.Context) (interface{}, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.RUnlock()
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.responses:
		// Decode message
		decoded, err := t.codec.Decode(msg.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		return decoded, nil
	}
}

// Disconnect closes the NATS connection
func (t *NATSTransport) Disconnect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	t.closed = true
	
	if t.sub != nil {
		t.sub.Unsubscribe()
	}
	
	if t.conn != nil {
		t.conn.Close()
	}
	
	close(t.responses)
	
	return nil
}

// IsConnected returns whether the transport is connected
func (t *NATSTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return !t.closed && t.conn != nil && t.conn.IsConnected()
}

// Subscribe subscribes to a subject pattern
func (t *NATSTransport) Subscribe(subject string, handler MessageHandler) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return fmt.Errorf("transport is closed")
	}
	conn := t.conn
	t.mu.RUnlock()
	
	_, err := conn.Subscribe(subject, func(msg *nats.Msg) {
		// Decode message
		decoded, err := t.codec.Decode(msg.Data)
		if err != nil {
			// Log error
			return
		}
		
		// Create context with trace ID
		ctx := context.Background()
		if traceID := msg.Header.Get("X-Trace-ID"); traceID != "" {
			ctx = context.WithValue(ctx, "trace_id", traceID)
		}
		
		// Handle message
		if err := handler(ctx, decoded); err != nil {
			// Send error response if this was a request
			if msg.Reply != "" {
				errResp := &protocol.Response{
					JSONRPC: protocol.JSONRPCVersion,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInternal,
						Message: err.Error(),
					},
				}
				
				if data, err := json.Marshal(errResp); err == nil {
					msg.Respond(data)
				}
			}
		}
	})
	
	return err
}

// Request sends a request and waits for response
func (t *NATSTransport) Request(ctx context.Context, subject string, msg interface{}) (interface{}, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}
	conn := t.conn
	t.mu.RUnlock()
	
	// Encode message
	data, err := t.codec.Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	
	// Create timeout from context
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	
	// Send request
	reply, err := conn.Request(subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	// Decode response
	decoded, err := t.codec.Decode(reply.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return decoded, nil
}

// getSubjectForMessage determines the NATS subject for a message
func (t *NATSTransport) getSubjectForMessage(msg interface{}) string {
	switch m := msg.(type) {
	case *protocol.Request:
		return fmt.Sprintf("mcp.request.%s", m.Method)
	case *protocol.Response:
		return "mcp.response"
	case *protocol.Notification:
		return fmt.Sprintf("mcp.notification.%s", m.Method)
	case *protocol.MCPMessage:
		switch m.Method {
		case protocol.RequestTypeToolRegister:
			return "mcp.tools.register"
		case protocol.RequestTypeToolDiscover:
			return "mcp.tools.discover"
		case protocol.RequestTypePromptProcess:
			return "mcp.prompts.process"
		case protocol.RequestTypeCostReport:
			return "mcp.cost.report"
		default:
			return fmt.Sprintf("mcp.%s", m.MessageType)
		}
	default:
		return "mcp.message"
	}
}

// NATSServer implements a NATS-based MCP server
type NATSServer struct {
	transport *NATSTransport
	handlers  map[string]MessageHandler
	mu        sync.RWMutex
}

// NewNATSServer creates a new NATS server
func NewNATSServer(config *TransportConfig) (*NATSServer, error) {
	transport, err := NewNATSTransport(config)
	if err != nil {
		return nil, err
	}
	
	return &NATSServer{
		transport: transport,
		handlers:  make(map[string]MessageHandler),
	}, nil
}

// RegisterHandler registers a handler for a method
func (s *NATSServer) RegisterHandler(method string, handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// Start starts the server
func (s *NATSServer) Start(ctx context.Context) error {
	// Subscribe to all MCP subjects
	return s.transport.Subscribe("mcp.>", func(ctx context.Context, msg interface{}) error {
		// Route to appropriate handler
		s.mu.RLock()
		defer s.mu.RUnlock()
		
		// Extract method from message
		var method string
		switch m := msg.(type) {
		case *protocol.Request:
			method = m.Method
		case *protocol.Notification:
			method = m.Method
		case *protocol.MCPMessage:
			method = m.Method
		}
		
		if handler, ok := s.handlers[method]; ok {
			return handler(ctx, msg)
		}
		
		return fmt.Errorf("no handler for method: %s", method)
	})
}

// Stop stops the server
func (s *NATSServer) Stop(ctx context.Context) error {
	return s.transport.Disconnect(ctx)
}