// Package transport provides transport layer abstractions for MCP
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
)

// Transport defines the interface for MCP transport implementations
type Transport interface {
	// Connect establishes a connection
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// Send sends a message through the transport
	Send(ctx context.Context, topic string, data []byte) error

	// Receive receives messages from the transport
	Receive(ctx context.Context) <-chan []byte

	// Request sends a request and waits for response
	Request(ctx context.Context, topic string, data []byte) ([]byte, error)

	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topic string) (<-chan []byte, error)

	// Unsubscribe unsubscribes from a topic
	Unsubscribe(ctx context.Context, topic string) error

	// Publish publishes to a topic
	Publish(ctx context.Context, topic string, data []byte) error
}

// StreamTransport supports streaming operations
type StreamTransport interface {
	Transport

	// SendStream sends a stream of messages
	SendStream(ctx context.Context) (StreamWriter, error)

	// ReceiveStream receives a stream of messages
	ReceiveStream(ctx context.Context) (StreamReader, error)
}

// StreamWriter writes messages to a stream
type StreamWriter interface {
	// Write writes a message to the stream
	Write(msg interface{}) error

	// Close closes the stream
	Close() error
}

// StreamReader reads messages from a stream
type StreamReader interface {
	// Read reads the next message from the stream
	Read() (interface{}, error)

	// Close closes the stream
	Close() error
}

// ConnectionInfo provides information about a transport connection
type ConnectionInfo struct {
	Protocol    string            // "nats", "grpc", "websocket", etc.
	LocalAddr   string            // Local address
	RemoteAddr  string            // Remote address
	Secure      bool              // Whether TLS is enabled
	Metadata    map[string]string // Additional metadata
}

// TransportConfig provides common configuration for transports
type TransportConfig struct {
	// Transport type
	Type           string            // "nats", "memory", "grpc"

	// Connection settings
	Address        string
	TLSConfig      *TLSConfig
	ConnectTimeout time.Duration

	// Retry settings
	MaxReconnects  int
	ReconnectWait  time.Duration

	// Message settings
	MaxMessageSize int
	Compression    bool

	// Authentication
	AuthType       string // "none", "token", "cert"
	AuthToken      string

	// Metadata
	ClientID       string
	Metadata       map[string]string

	// Transport-specific config
	Config         map[string]interface{}
}

// TLSConfig provides TLS configuration
type TLSConfig struct {
	CertFile       string
	KeyFile        string
	CAFile         string
	ServerName     string
	SkipVerify     bool
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx context.Context, msg interface{}) error

// Server represents a transport server
type Server interface {
	// Start starts the server
	Start(ctx context.Context) error

	// Stop stops the server
	Stop(ctx context.Context) error

	// Accept accepts incoming connections
	Accept() (Transport, error)

	// Addr returns the server address
	Addr() string
}

// Client represents a transport client
type Client interface {
	// Connect establishes a connection
	Connect(ctx context.Context) (Transport, error)

	// Close closes the client
	Close() error
}

// Codec handles message encoding/decoding for transports
type Codec interface {
	// Encode encodes a message
	Encode(msg interface{}) ([]byte, error)

	// Decode decodes a message
	Decode(data []byte) (interface{}, error)
}

// DefaultCodec uses JSON-RPC for message encoding
type DefaultCodec struct {
	jsonrpc *protocol.Codec
}

// NewDefaultCodec creates a new default codec
func NewDefaultCodec() *DefaultCodec {
	return &DefaultCodec{
		jsonrpc: protocol.NewCodec(nil),
	}
}

// Encode encodes a message using JSON
func (c *DefaultCodec) Encode(msg interface{}) ([]byte, error) {
	// For protocol messages, encode directly
	switch m := msg.(type) {
	case *protocol.Request, *protocol.Response, *protocol.Notification:
		return json.Marshal(m)
	case *protocol.MCPMessage:
		return json.Marshal(m)
	default:
		// For other types, wrap in MCPMessage
		payload, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}

		mcpMsg := &protocol.MCPMessage{
			ID:          generateID(),
			Version:     "1.0",
			MessageType: protocol.MessageTypeRequest,
			Timestamp:   time.Now(),
			Payload:     payload,
		}

		return json.Marshal(mcpMsg)
	}
}

// Decode decodes a message from JSON
func (c *DefaultCodec) Decode(data []byte) (interface{}, error) {
	// Try to decode as JSON-RPC message first
	msg, err := c.jsonrpc.DecodeMessage(data)
	if err == nil {
		return msg, nil
	}

	// Try to decode as MCPMessage
	var mcpMsg protocol.MCPMessage
	if err := json.Unmarshal(data, &mcpMsg); err == nil {
		return &mcpMsg, nil
	}

	return nil, fmt.Errorf("failed to decode message")
}

// Helpers

func generateID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// NewTransport creates a new transport based on configuration
func NewTransport(config *TransportConfig) (Transport, error) {
	if config == nil {
		return nil, fmt.Errorf("transport config cannot be nil")
	}

	switch config.Type {
	case "nats":
		return NewNATSTransport(config)
	case "memory":
		return NewMemoryTransport(config), nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.Type)
	}
}
