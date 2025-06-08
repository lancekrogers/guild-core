// Package protocol defines the core MCP message types and structures
package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType defines the type of MCP message
type MessageType string

const (
	// Core message types
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeNotification MessageType = "notification"
	MessageTypeError        MessageType = "error"
	MessageTypeEvent        MessageType = "event"

	// Aliases for backward compatibility
	RequestMessage  = MessageTypeRequest
	ResponseMessage = MessageTypeResponse
	ErrorMessage    = MessageTypeError
	EventMessage    = MessageTypeEvent

	// Specific request types
	RequestTypeToolRegister   = "tool.register"
	RequestTypeToolDeregister = "tool.deregister"
	RequestTypeToolDiscover   = "tool.discover"
	RequestTypeToolExecute    = "tool.execute"
	RequestTypePromptProcess  = "prompt.process"
	RequestTypePromptAnalyze  = "prompt.analyze"
	RequestTypeCostReport     = "cost.report"
	RequestTypeCostQuery      = "cost.query"
)

// MCPMessage represents the base message structure
type MCPMessage struct {
	ID          string          `json:"id"`
	Version     string          `json:"version"`
	MessageType MessageType     `json:"message_type"`
	Method      string          `json:"method,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    Metadata        `json:"metadata,omitempty"`
}

// Metadata contains contextual information about the message
type Metadata struct {
	TraceID      string            `json:"trace_id"`
	SessionID    string            `json:"session_id"`
	Source       string            `json:"source"`
	Destination  string            `json:"destination"`
	Priority     int               `json:"priority"`
	TTL          int               `json:"ttl"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if len(e.Data) > 0 {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Standard error codes
const (
	ErrorCodeParse          = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternal       = -32603

	// MCP-specific error codes
	ErrorCodeToolNotFound    = -32001
	ErrorCodeToolUnavailable = -32002
	ErrorCodeCostLimitExceed = -32003
	ErrorCodeAuthFailed      = -32004
	ErrorCodeTimeout         = -32005
	ErrorCodeTooManyRequests = -32006
)

// Tool-related messages

// ToolRegistrationMessage for registering tools with the MCP
type ToolRegistrationMessage struct {
	ToolID           string           `json:"tool_id"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Version          string           `json:"version"`
	Capabilities     []string         `json:"capabilities"`
	Parameters       []ToolParameter  `json:"parameters"`
	Returns          []ToolParameter  `json:"returns"`
	CostProfile      CostProfile      `json:"cost_profile"`
	AuthRequirements AuthRequirements `json:"auth_requirements"`
}

// ToolParameter describes a tool parameter or return value
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// CostProfile defines the cost characteristics of a tool
type CostProfile struct {
	ComputeCost   float64       `json:"compute_cost"`   // CPU/GPU units
	MemoryCost    int64         `json:"memory_cost"`    // Bytes
	LatencyCost   time.Duration `json:"latency_cost"`   // Expected latency
	FinancialCost float64       `json:"financial_cost"` // Dollar cost per call
}

// AuthRequirements specifies authentication needs
type AuthRequirements struct {
	Type        string   `json:"type"` // "none", "api_key", "oauth", "jwt"
	Scopes      []string `json:"scopes,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ToolQuery for discovering tools
type ToolQuery struct {
	RequiredCapabilities []string          `json:"required_capabilities,omitempty"`
	Tags                 map[string]string `json:"tags,omitempty"`
	MaxLatency           time.Duration     `json:"max_latency,omitempty"`
	MaxCost              float64           `json:"max_cost,omitempty"`
}

// ToolList contains discovered tools
type ToolList struct {
	Tools []ToolInfo `json:"tools"`
}

// ToolInfo provides information about a discovered tool
type ToolInfo struct {
	ToolID       string      `json:"tool_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Capabilities []string    `json:"capabilities"`
	Available    bool        `json:"available"`
	CostProfile  CostProfile `json:"cost_profile"`
}

// Prompt-related messages

// PromptMessage for sending prompts through the MCP
type PromptMessage struct {
	Text        string                 `json:"text"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	HistoryID   string                 `json:"history_id,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	CostLimit   CostLimit              `json:"cost_limit,omitempty"`
}

// CostLimit defines spending limits
type CostLimit struct {
	MaxComputeCost   float64       `json:"max_compute_cost,omitempty"`
	MaxFinancialCost float64       `json:"max_financial_cost,omitempty"`
	MaxLatency       time.Duration `json:"max_latency,omitempty"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
}

// PromptResponse contains the result of prompt processing
type PromptResponse struct {
	Text      string            `json:"text"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CostUsed  CostReport        `json:"cost_used"`
	ToolsUsed []string          `json:"tools_used,omitempty"`
}

// Cost-related messages

// CostReport for tracking and optimizing resource usage
type CostReport struct {
	ComputeCost   float64       `json:"compute_cost"`
	MemoryCost    int64         `json:"memory_cost"`
	LatencyCost   time.Duration `json:"latency_cost"`
	TokensCost    int           `json:"tokens_cost"`
	APICallsCost  int           `json:"api_calls_cost"`
	FinancialCost float64       `json:"financial_cost"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	OperationID   string        `json:"operation_id"`
}

// CostQuery for querying cost information
type CostQuery struct {
	OperationIDs []string  `json:"operation_ids,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
	GroupBy      string    `json:"group_by,omitempty"` // "operation", "tool", "user"
	Limit        int       `json:"limit,omitempty"`
}

// CostAnalysis contains cost analysis results
type CostAnalysis struct {
	TotalCost       CostReport   `json:"total_cost"`
	BreakdownBy     string       `json:"breakdown_by"`
	Breakdown       []CostReport `json:"breakdown"`
	Recommendations []string     `json:"recommendations,omitempty"`
}

// ErrorResponse represents an error response message
type ErrorResponse struct {
	Error   *Error      `json:"error"`
	ID      interface{} `json:"id,omitempty"`
	Version string      `json:"version,omitempty"`
}
