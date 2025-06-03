package protocol

import (
	"fmt"
)

// MCPError represents an MCP-specific error
type MCPError struct {
	Code    int
	Message string
	Data    interface{}
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("MCP error %d: %s (%v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// ToJSONRPCError converts to a JSON-RPC error
func (e *MCPError) ToJSONRPCError() *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Data:    nil, // Will be marshaled separately
	}
}

// Common MCP errors
var (
	ErrParseError = &MCPError{
		Code:    ErrorCodeParse,
		Message: "Parse error",
	}
	
	ErrInvalidRequest = &MCPError{
		Code:    ErrorCodeInvalidRequest,
		Message: "Invalid request",
	}
	
	ErrMethodNotFound = &MCPError{
		Code:    ErrorCodeMethodNotFound,
		Message: "Method not found",
	}
	
	ErrInvalidParams = &MCPError{
		Code:    ErrorCodeInvalidParams,
		Message: "Invalid params",
	}
	
	ErrInternalError = &MCPError{
		Code:    ErrorCodeInternal,
		Message: "Internal error",
	}
	
	ErrToolNotFound = &MCPError{
		Code:    ErrorCodeToolNotFound,
		Message: "Tool not found",
	}
	
	ErrToolUnavailable = &MCPError{
		Code:    ErrorCodeToolUnavailable,
		Message: "Tool unavailable",
	}
	
	ErrCostLimitExceeded = &MCPError{
		Code:    ErrorCodeCostLimitExceed,
		Message: "Cost limit exceeded",
	}
	
	ErrAuthFailed = &MCPError{
		Code:    ErrorCodeAuthFailed,
		Message: "Authentication failed",
	}
	
	ErrTimeout = &MCPError{
		Code:    ErrorCodeTimeout,
		Message: "Request timeout",
	}
)

// NewMCPError creates a new MCP error
func NewMCPError(code int, message string, data interface{}) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// WrapError wraps a standard error as an MCP error
func WrapError(err error, code int) *MCPError {
	if err == nil {
		return nil
	}
	
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr
	}
	
	return &MCPError{
		Code:    code,
		Message: err.Error(),
	}
}