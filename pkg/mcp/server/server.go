// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package server implements the MCP server
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/mcp/cost"
	"github.com/lancekrogers/guild/pkg/mcp/prompt"
	"github.com/lancekrogers/guild/pkg/mcp/protocol"
	"github.com/lancekrogers/guild/pkg/mcp/tools"
	"github.com/lancekrogers/guild/pkg/mcp/transport"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// Server represents the MCP server
type Server struct {
	config         *Config
	transport      transport.Transport
	toolRegistry   tools.Registry
	costObserver   cost.Observer
	promptAnalyzer prompt.Analyzer
	guildRegistry  registry.ComponentRegistry
	handlers       map[string]HandlerFunc
	middleware     []Middleware
	mu             sync.RWMutex
	started        bool
	stopCh         chan struct{}
}

// Config holds server configuration
type Config struct {
	// Server identification
	ServerID   string
	ServerName string
	Version    string

	// Transport configuration
	TransportConfig *transport.TransportConfig

	// Security settings
	EnableTLS      bool
	TLSCertFile    string
	TLSKeyFile     string
	EnableAuth     bool
	JWTSecret      string
	AllowedOrigins []string

	// Performance settings
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	ShutdownTimeout       time.Duration

	// Feature flags
	EnableMetrics      bool
	EnableTracing      bool
	EnableCostTracking bool
}

// HandlerFunc handles MCP messages
type HandlerFunc func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error)

// Middleware wraps handlers with additional functionality
type Middleware func(HandlerFunc) HandlerFunc

// NewServer creates a new MCP server
func NewServer(config *Config, guildRegistry registry.ComponentRegistry) (*Server, error) {
	if config == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "mcp_server", nil).WithComponent("new_server").WithOperation("config cannot be nil")
	}
	if guildRegistry == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "mcp_server", nil).WithComponent("new_server").WithOperation("guild registry cannot be nil")
	}

	// Create transport
	transport, err := transport.NewTransport(config.TransportConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").WithComponent("new_server").WithOperation("failed to create transport")
	}

	// Create components
	toolRegistry := tools.NewMemoryRegistry()
	costObserver := cost.NewMemoryObserver(10000)
	promptAnalyzer := prompt.NewAnalyzer()

	server := &Server{
		config:         config,
		transport:      transport,
		toolRegistry:   toolRegistry,
		costObserver:   costObserver,
		promptAnalyzer: promptAnalyzer,
		guildRegistry:  guildRegistry,
		handlers:       make(map[string]HandlerFunc),
		middleware:     make([]Middleware, 0),
		stopCh:         make(chan struct{}),
	}

	// Register default handlers
	server.registerDefaultHandlers()

	// Apply default middleware
	server.applyDefaultMiddleware()

	return server, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeInternal, "mcp_server", nil).WithComponent("start").WithOperation("server already started")
	}

	// Connect transport
	if err := s.transport.Connect(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").WithComponent("start").WithOperation("failed to connect transport")
	}

	// Start message processing
	go s.processMessages(ctx)

	s.started = true
	return nil
}

// Stop stops the MCP server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeInternal, "mcp_server", nil).WithComponent("stop").WithOperation("server not started")
	}

	// Signal stop
	close(s.stopCh)

	// Disconnect transport
	if err := s.transport.Disconnect(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").WithComponent("stop").WithOperation("failed to disconnect transport")
	}

	s.started = false
	return nil
}

// RegisterHandler registers a message handler
func (s *Server) RegisterHandler(method string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply middleware to handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	s.handlers[method] = handler
}

// UseMiddleware adds middleware to the server
func (s *Server) UseMiddleware(mw Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.middleware = append(s.middleware, mw)
}

// processMessages processes incoming messages
func (s *Server) processMessages(ctx context.Context) {
	msgCh := s.transport.Receive(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case msgBytes := <-msgCh:
			go s.handleMessage(ctx, msgBytes)
		}
	}
}

// handleMessage handles a single message
func (s *Server) handleMessage(ctx context.Context, msgBytes []byte) {
	// Parse message
	var msg protocol.MCPMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		s.sendError(ctx, "", protocol.ParseError, "Invalid message format", nil)
		return
	}

	// Add request context
	ctx = context.WithValue(ctx, "request_id", msg.ID)
	ctx = context.WithValue(ctx, "method", msg.Method)

	// Get handler
	s.mu.RLock()
	handler, exists := s.handlers[msg.Method]
	s.mu.RUnlock()

	if !exists {
		s.sendError(ctx, msg.ID, protocol.MethodNotFound,
			fmt.Sprintf("Method %s not found", msg.Method), nil)
		return
	}

	// Handle request
	response, err := handler(ctx, &msg)
	if err != nil {
		// Check if it's already an MCP error
		if mcpErr, ok := err.(*protocol.Error); ok {
			s.sendError(ctx, msg.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data)
		} else {
			s.sendError(ctx, msg.ID, protocol.ErrorCodeInternal, err.Error(), nil)
		}
		return
	}

	// Send response
	if response != nil {
		response.ID = msg.ID // Ensure response has same ID
		if err := s.sendMessage(ctx, response); err != nil {
			gerr := gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
				WithComponent("send_response").
				WithOperation("sendMessage").
				FromContext(ctx)
			observability.GetLogger(ctx).WithError(gerr).ErrorContext(ctx, "failed to send response")
		}
	}
}

// sendMessage sends a message via transport
func (s *Server) sendMessage(ctx context.Context, msg *protocol.MCPMessage) error {
	// Set defaults
	if msg.Version == "" {
		msg.Version = "1.0"
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Marshal message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").WithComponent("send_message").WithOperation("failed to marshal message")
	}

	// Send via transport
	return s.transport.Send(ctx, msg.ID, msgBytes)
}

// sendError sends an error response
func (s *Server) sendError(ctx context.Context, id string, code int, message string, data interface{}) {
	var dataBytes json.RawMessage
	if data != nil {
		if rawMsg, ok := data.(json.RawMessage); ok {
			dataBytes = rawMsg
		} else {
			dataBytes, _ = json.Marshal(data)
		}
	}

	payload, err := mustMarshal(&protocol.ErrorResponse{
		Error: &protocol.Error{
			Code:    code,
			Message: message,
			Data:    dataBytes,
		},
	})
	if err != nil {
		gerr := gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("send_error").
			WithOperation("marshal error response").
			FromContext(ctx)
		observability.GetLogger(ctx).WithError(gerr).ErrorContext(ctx, "failed to marshal error response")
		return
	}

	errMsg := &protocol.MCPMessage{
		ID:          id,
		Version:     "1.0",
		MessageType: protocol.ErrorMessage,
		Timestamp:   time.Now(),
		Payload:     payload,
	}

	if err := s.sendMessage(ctx, errMsg); err != nil {
		gerr := gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("send_error").
			WithOperation("sendMessage").
			FromContext(ctx)
		observability.GetLogger(ctx).WithError(gerr).ErrorContext(ctx, "failed to send error response")
	}
}

// registerDefaultHandlers registers default message handlers
func (s *Server) registerDefaultHandlers() {
	// Tool registration
	s.RegisterHandler("tool.register", s.handleToolRegister)
	s.RegisterHandler("tool.deregister", s.handleToolDeregister)
	s.RegisterHandler("tool.discover", s.handleToolDiscover)
	s.RegisterHandler("tool.execute", s.handleToolExecute)
	s.RegisterHandler("tool.health", s.handleToolHealth)

	// Cost tracking
	s.RegisterHandler("cost.report", s.handleCostReport)
	s.RegisterHandler("cost.query", s.handleCostQuery)

	// Prompt processing
	s.RegisterHandler("prompt.process", s.handlePromptProcess)
	s.RegisterHandler("prompt.analyze", s.handlePromptAnalyze)

	// System
	s.RegisterHandler("system.ping", s.handleSystemPing)
	s.RegisterHandler("system.info", s.handleSystemInfo)
}

// Tool handlers

func (s *Server) handleToolRegister(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req protocol.ToolRegistrationRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid tool registration request",
		}
	}

	// Create tool from definition
	tool := tools.NewBaseTool(
		req.Tool.ID,
		req.Tool.Name,
		req.Tool.Description,
		req.Tool.Capabilities,
		req.Tool.CostProfile,
		req.Tool.Parameters,
		req.Tool.Returns,
		nil, // Executor will be set up separately
	)

	// Register tool
	if err := s.toolRegistry.RegisterTool(tool); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Failed to register tool: %v", err),
		}
	}

	// Create response
	response := &protocol.ToolRegistrationResponse{
		Success: true,
		ToolID:  req.Tool.ID,
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_tool_register").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleToolDeregister(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req struct {
		ToolID string `json:"tool_id"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid deregistration request",
		}
	}

	if err := s.toolRegistry.DeregisterTool(req.ToolID); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Failed to deregister tool: %v", err),
		}
	}

	response := map[string]bool{"success": true}
	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_tool_deregister").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleToolDiscover(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var query protocol.ToolQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid tool query",
		}
	}

	tools, err := s.toolRegistry.DiscoverTools(query)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Failed to discover tools: %v", err),
		}
	}

	// Convert tools to definitions
	var toolInfos []protocol.ToolInfo
	for _, tool := range tools {
		toolInfos = append(toolInfos, protocol.ToolInfo{
			ToolID:       tool.ID(),
			Name:         tool.Name(),
			Description:  tool.Description(),
			Capabilities: tool.Capabilities(),
			Available:    tool.HealthCheck() == nil,
			CostProfile:  tool.GetCostProfile(),
		})
	}

	response := &protocol.ToolDiscoveryResponse{
		Tools: toolInfos,
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_tool_discover").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleToolExecute(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req protocol.ToolExecutionRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid execution request",
		}
	}

	// Get tool
	tool, err := s.toolRegistry.GetTool(req.ToolID)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: fmt.Sprintf("Tool not found: %v", err),
		}
	}

	// Generate execution ID
	executionID := fmt.Sprintf("exec-%s-%d", req.ToolID, time.Now().UnixNano())

	// Execute tool
	startTime := time.Now()
	result, err := tool.Execute(ctx, req.Parameters)
	endTime := time.Now()

	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Tool execution failed: %v", err),
		}
	}

	// Record cost if enabled
	if s.config.EnableCostTracking {
		cost := protocol.CostReport{
			OperationID: executionID,
			StartTime:   startTime,
			EndTime:     endTime,
			LatencyCost: endTime.Sub(startTime),
		}
		s.costObserver.RecordCost(ctx, executionID, cost)
	}

	// Marshal result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Failed to marshal result: %v", err),
		}
	}

	response := &protocol.ToolExecutionResponse{
		Success:     true,
		ExecutionID: executionID,
		ToolID:      req.ToolID,
		Result:      json.RawMessage(resultBytes),
		Duration:    endTime.Sub(startTime),
		StartTime:   startTime,
		EndTime:     endTime,
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_tool_execute").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleToolHealth(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req struct {
		ToolID string `json:"tool_id"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid health check request",
		}
	}

	tool, err := s.toolRegistry.GetTool(req.ToolID)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: fmt.Sprintf("Tool not found: %v", err),
		}
	}

	healthy := tool.HealthCheck() == nil
	response := map[string]interface{}{
		"tool_id": req.ToolID,
		"healthy": healthy,
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_tool_health").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

// Cost handlers

func (s *Server) handleCostReport(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var report protocol.CostReport
	if err := json.Unmarshal(msg.Payload, &report); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid cost report",
		}
	}

	s.costObserver.RecordCost(ctx, report.OperationID, report)

	response := map[string]bool{"success": true}
	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_cost_report").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleCostQuery(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var query protocol.CostQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid cost query",
		}
	}

	analysis, err := s.costObserver.Analyze(ctx, query)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Cost analysis failed: %v", err),
		}
	}

	payload, err := mustMarshal(analysis)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_cost_query").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

// Prompt handlers

func (s *Server) handlePromptProcess(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req protocol.PromptMessage
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid prompt message",
		}
	}

	// Create prompt chain (simplified for now)
	chain := prompt.NewChain(
		prompt.NewValidationProcessor(func(input *prompt.Input) error {
			if input.Text == "" {
				return gerror.New(gerror.ErrCodeInvalidInput, "mcp_server", nil).WithComponent("handle_get_prompt").WithOperation("prompt text cannot be empty")
			}
			return nil
		}),
	)

	// Process prompt
	input := &prompt.Input{
		Text:       req.Text,
		Parameters: req.Parameters,
	}

	output, err := chain.Process(ctx, input)
	if err != nil {
		return nil, &protocol.Error{
			Code:    protocol.ErrorCodeInternal,
			Message: fmt.Sprintf("Prompt processing failed: %v", err),
		}
	}

	response := &protocol.PromptResponse{
		Text:     output.Text,
		Metadata: output.Metadata,
		CostUsed: protocol.CostReport{
			StartTime: time.Now(),
			EndTime:   time.Now(),
		},
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_prompt_process").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handlePromptAnalyze(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	var req struct {
		ChainID string `json:"chain_id,omitempty"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, &protocol.Error{
			Code:    protocol.InvalidParams,
			Message: "Invalid analyze request",
		}
	}

	var response interface{}
	if req.ChainID != "" {
		// Get specific chain analysis
		analysis, err := s.promptAnalyzer.GetChainAnalysis(req.ChainID)
		if err != nil {
			return nil, &protocol.Error{
				Code:    protocol.ErrorCodeInternal,
				Message: fmt.Sprintf("Failed to get chain analysis: %v", err),
			}
		}
		response = analysis
	} else {
		// Get aggregate analysis
		response = s.promptAnalyzer.GetAggregateAnalysis()
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_prompt_analyze").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

// System handlers

func (s *Server) handleSystemPing(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	response := map[string]interface{}{
		"pong":      true,
		"timestamp": time.Now(),
		"server_id": s.config.ServerID,
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_system_ping").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

func (s *Server) handleSystemInfo(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
	response := map[string]interface{}{
		"server_id":   s.config.ServerID,
		"server_name": s.config.ServerName,
		"version":     s.config.Version,
		"features": map[string]bool{
			"tls":           s.config.EnableTLS,
			"auth":          s.config.EnableAuth,
			"metrics":       s.config.EnableMetrics,
			"tracing":       s.config.EnableTracing,
			"cost_tracking": s.config.EnableCostTracking,
		},
		"tools_count": len(s.toolRegistry.ListTools()),
	}

	payload, err := mustMarshal(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("handle_system_info").
			WithOperation("marshal response")
	}

	return &protocol.MCPMessage{
		Version:     msg.Version,
		MessageType: protocol.ResponseMessage,
		Method:      msg.Method,
		Timestamp:   time.Now(),
		Payload:     payload,
	}, nil
}

// applyDefaultMiddleware applies default middleware
func (s *Server) applyDefaultMiddleware() {
	// Logging middleware
	s.UseMiddleware(loggingMiddleware)

	// Recovery middleware
	s.UseMiddleware(recoveryMiddleware)

	// Timeout middleware
	if s.config.RequestTimeout > 0 {
		s.UseMiddleware(timeoutMiddleware(s.config.RequestTimeout))
	}

	// Auth middleware
	if s.config.EnableAuth {
		s.UseMiddleware(authMiddleware(s.config.JWTSecret))
	}
}

// Accessor methods for testing and external integration

// GetConfig returns the server configuration
func (s *Server) GetConfig() *Config {
	return s.config
}

// GetToolRegistry returns the tool registry
func (s *Server) GetToolRegistry() tools.Registry {
	return s.toolRegistry
}

// GetCostObserver returns the cost observer
func (s *Server) GetCostObserver() cost.Observer {
	return s.costObserver
}

// GetPromptAnalyzer returns the prompt analyzer
func (s *Server) GetPromptAnalyzer() prompt.Analyzer {
	return s.promptAnalyzer
}

// Helper functions

func mustMarshal(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "mcp_server").
			WithComponent("marshal_payload").
			WithOperation("json_marshal")
	}
	return json.RawMessage(data), nil
}
