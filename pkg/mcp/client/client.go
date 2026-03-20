// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package client implements the MCP client
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/mcp/protocol"
	"github.com/lancekrogers/guild-core/pkg/mcp/transport"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// Client represents an MCP client
type Client struct {
	config        *Config
	transport     transport.Transport
	mu            sync.RWMutex
	connected     bool
	requestID     int64
	pendingReqs   map[string]chan *protocol.MCPMessage
	eventHandlers map[string]EventHandler
}

// Config holds client configuration
type Config struct {
	// Client identification
	ClientID   string
	ClientName string
	Version    string

	// Transport configuration
	TransportConfig *transport.TransportConfig

	// Security settings
	AuthToken   string
	EnableTLS   bool
	TLSInsecure bool

	// Performance settings
	RequestTimeout time.Duration
	ConnectTimeout time.Duration
	ReconnectDelay time.Duration
	MaxReconnects  int

	// Features
	EnableMetrics bool
	EnableTracing bool
}

// EventHandler handles server events
type EventHandler func(ctx context.Context, event *protocol.MCPMessage) error

// NewClient creates a new MCP client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "config cannot be nil", nil).
			WithComponent("MCPClient").
			WithOperation("NewClient")
	}

	// Set defaults
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10 * time.Second
	}
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}

	// Create transport
	transport, err := transport.NewTransport(config.TransportConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create transport").
			WithComponent("MCPClient").
			WithOperation("NewClient")
	}

	return &Client{
		config:        config,
		transport:     transport,
		pendingReqs:   make(map[string]chan *protocol.MCPMessage),
		eventHandlers: make(map[string]EventHandler),
	}, nil
}

// Connect connects to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return gerror.New(gerror.ErrCodeValidation, "client already connected", nil).
			WithComponent("MCPClient").
			WithOperation("Connect")
	}

	// Connect transport with timeout
	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	if err := c.transport.Connect(connectCtx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect transport").
			WithComponent("MCPClient").
			WithOperation("Connect")
	}

	// Start message processing
	go c.processMessages(ctx)

	c.connected = true
	return nil
}

// Disconnect disconnects from the MCP server
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return gerror.New(gerror.ErrCodeValidation, "client not connected", nil).
			WithComponent("MCPClient").
			WithOperation("Disconnect")
	}

	// Cancel pending requests
	for _, ch := range c.pendingReqs {
		close(ch)
	}
	c.pendingReqs = make(map[string]chan *protocol.MCPMessage)

	// Disconnect transport
	if err := c.transport.Disconnect(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to disconnect transport").
			WithComponent("MCPClient").
			WithOperation("Disconnect")
	}

	c.connected = false
	return nil
}

// RegisterTool registers a tool with the server
func (c *Client) RegisterTool(ctx context.Context, tool *protocol.ToolDefinition) error {
	req := &protocol.ToolRegistrationRequest{
		Tool: *tool,
	}

	response, err := c.sendRequest(ctx, "tool.register", req)
	if err != nil {
		return err
	}

	var resp protocol.ToolRegistrationResponse
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("RegisterTool")
	}

	if !resp.Success {
		return gerror.New(gerror.ErrCodeProviderAPI, "tool registration failed", nil).
			WithComponent("MCPClient").
			WithOperation("RegisterTool")
	}

	return nil
}

// DeregisterTool removes a tool from the server
func (c *Client) DeregisterTool(ctx context.Context, toolID string) error {
	req := map[string]string{"tool_id": toolID}

	_, err := c.sendRequest(ctx, "tool.deregister", req)
	return err
}

// DiscoverTools discovers available tools
func (c *Client) DiscoverTools(ctx context.Context, query *protocol.ToolQuery) (*protocol.ToolDiscoveryResponse, error) {
	response, err := c.sendRequest(ctx, "tool.discover", query)
	if err != nil {
		return nil, err
	}

	var resp protocol.ToolDiscoveryResponse
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("DiscoverTools")
	}

	return &resp, nil
}

// ExecuteTool executes a tool on the server
func (c *Client) ExecuteTool(ctx context.Context, req *protocol.ToolExecutionRequest) (*protocol.ToolExecutionResponse, error) {
	response, err := c.sendRequest(ctx, "tool.execute", req)
	if err != nil {
		return nil, err
	}

	var resp protocol.ToolExecutionResponse
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("ExecuteTool")
	}

	return &resp, nil
}

// CheckToolHealth checks if a tool is healthy
func (c *Client) CheckToolHealth(ctx context.Context, toolID string) (bool, error) {
	req := map[string]string{"tool_id": toolID}

	response, err := c.sendRequest(ctx, "tool.health", req)
	if err != nil {
		return false, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("CheckToolHealth")
	}

	healthy, ok := resp["healthy"].(bool)
	if !ok {
		return false, gerror.New(gerror.ErrCodeInternal, "invalid health response", nil).
			WithComponent("MCPClient").
			WithOperation("CheckToolHealth")
	}

	return healthy, nil
}

// ReportCost reports cost to the server
func (c *Client) ReportCost(ctx context.Context, cost *protocol.CostReport) error {
	_, err := c.sendRequest(ctx, "cost.report", cost)
	return err
}

// QueryCosts queries cost analysis from the server
func (c *Client) QueryCosts(ctx context.Context, query *protocol.CostQuery) (*protocol.CostAnalysis, error) {
	response, err := c.sendRequest(ctx, "cost.query", query)
	if err != nil {
		return nil, err
	}

	var analysis protocol.CostAnalysis
	if err := json.Unmarshal(response.Payload, &analysis); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("QueryCosts")
	}

	return &analysis, nil
}

// ProcessPrompt processes a prompt through the server
func (c *Client) ProcessPrompt(ctx context.Context, prompt *protocol.PromptMessage) (*protocol.PromptResponse, error) {
	response, err := c.sendRequest(ctx, "prompt.process", prompt)
	if err != nil {
		return nil, err
	}

	var resp protocol.PromptResponse
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("ProcessPrompt")
	}

	return &resp, nil
}

// Ping sends a ping to the server
func (c *Client) Ping(ctx context.Context) (time.Time, error) {
	response, err := c.sendRequest(ctx, "system.ping", map[string]bool{"ping": true})
	if err != nil {
		return time.Time{}, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(response.Payload, &resp); err != nil {
		return time.Time{}, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("Ping")
	}

	timestampStr, ok := resp["timestamp"].(string)
	if !ok {
		return time.Time{}, gerror.New(gerror.ErrCodeInternal, "invalid ping response", nil).
			WithComponent("MCPClient").
			WithOperation("Ping")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse timestamp").
			WithComponent("MCPClient").
			WithOperation("Ping")
	}

	return timestamp, nil
}

// GetSystemInfo gets server information
func (c *Client) GetSystemInfo(ctx context.Context) (map[string]interface{}, error) {
	response, err := c.sendRequest(ctx, "system.info", map[string]bool{"info": true})
	if err != nil {
		return nil, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(response.Payload, &info); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("MCPClient").
			WithOperation("GetSystemInfo")
	}

	return info, nil
}

// OnEvent registers an event handler
func (c *Client) OnEvent(eventType string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.eventHandlers[eventType] = handler
}

// sendRequest sends a request and waits for response
func (c *Client) sendRequest(ctx context.Context, method string, payload interface{}) (*protocol.MCPMessage, error) {
	if !c.connected {
		return nil, gerror.New(gerror.ErrCodeValidation, "client not connected", nil).
			WithComponent("MCPClient").
			WithOperation("sendRequest")
	}

	// Generate request ID
	c.mu.Lock()
	c.requestID++
	requestID := fmt.Sprintf("%s-%d", c.config.ClientID, c.requestID)
	c.mu.Unlock()

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal payload").
			WithComponent("MCPClient").
			WithOperation("sendRequest")
	}

	// Create message
	metadata := protocol.Metadata{
		CustomFields: make(map[string]string),
	}

	// Add auth token if configured
	if c.config.AuthToken != "" {
		metadata.CustomFields["authorization"] = c.config.AuthToken
	}

	// Add tracing if enabled
	if c.config.EnableTracing {
		metadata.TraceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}

	msg := &protocol.MCPMessage{
		ID:          requestID,
		Version:     "1.0",
		MessageType: protocol.RequestMessage,
		Method:      method,
		Timestamp:   time.Now(),
		Payload:     json.RawMessage(payloadBytes),
		Metadata:    metadata,
	}

	// Create response channel
	responseCh := make(chan *protocol.MCPMessage, 1)

	c.mu.Lock()
	c.pendingReqs[requestID] = responseCh
	c.mu.Unlock()

	// Cleanup on exit
	defer func() {
		c.mu.Lock()
		delete(c.pendingReqs, requestID)
		c.mu.Unlock()
		close(responseCh)
	}()

	// Send message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal message").
			WithComponent("MCPClient").
			WithOperation("sendRequest")
	}

	if err := c.transport.Send(ctx, requestID, msgBytes); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to send message").
			WithComponent("MCPClient").
			WithOperation("sendRequest")
	}

	// Wait for response with timeout
	timeout := c.config.RequestTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseCh:
		if response == nil {
			return nil, gerror.New(gerror.ErrCodeConnection, "connection closed", nil).
				WithComponent("MCPClient").
				WithOperation("sendRequest")
		}

		// Check for error response
		if response.MessageType == protocol.ErrorMessage {
			var errorResp protocol.ErrorResponse
			if err := json.Unmarshal(response.Payload, &errorResp); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse error response").
					WithComponent("MCPClient").
					WithOperation("sendRequest")
			}
			return nil, errorResp.Error
		}

		return response, nil
	case <-time.After(timeout):
		return nil, gerror.Newf(gerror.ErrCodeTimeout, "request timeout after %v", timeout).
			WithComponent("MCPClient").
			WithOperation("sendRequest")
	}
}

// processMessages processes incoming messages
func (c *Client) processMessages(ctx context.Context) {
	msgCh := c.transport.Receive(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case msgBytes := <-msgCh:
			if msgBytes == nil {
				// Connection closed
				return
			}

			go c.handleMessage(ctx, msgBytes)
		}
	}
}

// handleMessage handles a single incoming message
func (c *Client) handleMessage(ctx context.Context, msgBytes []byte) {
	var msg protocol.MCPMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		observability.GetLogger(ctx).Warn("failed to unmarshal MCP message", "error", err)
		return
	}

	// Handle response messages
	if msg.MessageType == protocol.ResponseMessage || msg.MessageType == protocol.ErrorMessage {
		c.mu.RLock()
		responseCh, exists := c.pendingReqs[msg.ID]
		c.mu.RUnlock()

		if exists {
			select {
			case responseCh <- &msg:
			default:
				// Channel full or closed
			}
		}
		return
	}

	// Handle event messages
	if msg.MessageType == protocol.EventMessage {
		c.mu.RLock()
		handler, exists := c.eventHandlers[msg.Method]
		c.mu.RUnlock()

		if exists {
			go func() {
				if err := handler(ctx, &msg); err != nil {
					observability.GetLogger(ctx).Warn("MCP event handler error", "error", err, "method", msg.Method)
				}
			}()
		}
		return
	}
}
