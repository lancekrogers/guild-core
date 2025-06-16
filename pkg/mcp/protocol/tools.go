// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package protocol defines tool-related MCP message types
package protocol

import (
	"encoding/json"
	"time"
)

// ToolDefinition describes a tool that can be registered with the MCP
type ToolDefinition struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Version          string            `json:"version"`
	Category         string            `json:"category,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Icon             string            `json:"icon,omitempty"`
	Author           string            `json:"author,omitempty"`
	License          string            `json:"license,omitempty"`
	Website          string            `json:"website,omitempty"`
	Documentation    string            `json:"documentation,omitempty"`
	Capabilities     []string          `json:"capabilities"`
	Parameters       []ToolParameter   `json:"parameters"`
	Returns          []ToolParameter   `json:"returns"`
	Examples         []ToolExample     `json:"examples,omitempty"`
	CostProfile      CostProfile       `json:"cost_profile"`
	AuthRequirements AuthRequirements  `json:"auth_requirements"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// ToolExample provides usage examples for a tool
type ToolExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output,omitempty"`
}

// ToolRegistrationRequest is sent to register a tool
type ToolRegistrationRequest struct {
	Tool    ToolDefinition          `json:"tool"`
	Options ToolRegistrationOptions `json:"options,omitempty"`
}

// ToolRegistrationOptions provides options for tool registration
type ToolRegistrationOptions struct {
	ReplaceExisting bool              `json:"replace_existing,omitempty"`
	EnableMetrics   bool              `json:"enable_metrics,omitempty"`
	Priority        int               `json:"priority,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// ToolRegistrationResponse is the response to a tool registration request
type ToolRegistrationResponse struct {
	Success      bool             `json:"success"`
	ToolID       string           `json:"tool_id,omitempty"`
	Message      string           `json:"message,omitempty"`
	Warnings     []string         `json:"warnings,omitempty"`
	Registration ToolRegistration `json:"registration,omitempty"`
}

// ToolRegistration contains information about a registered tool
type ToolRegistration struct {
	ToolID       string            `json:"tool_id"`
	RegisteredAt time.Time         `json:"registered_at"`
	LastHealthAt time.Time         `json:"last_health_at,omitempty"`
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Endpoint     string            `json:"endpoint,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ToolDiscoveryResponse contains discovered tools matching a query
type ToolDiscoveryResponse struct {
	Tools     []ToolInfo    `json:"tools"`
	Total     int           `json:"total"`
	Page      int           `json:"page,omitempty"`
	PageSize  int           `json:"page_size,omitempty"`
	HasMore   bool          `json:"has_more,omitempty"`
	QueryTime time.Duration `json:"query_time,omitempty"`
}

// ToolExecutionRequest requests execution of a tool
type ToolExecutionRequest struct {
	ToolID     string                 `json:"tool_id"`
	Parameters map[string]interface{} `json:"parameters"`
	Context    ExecutionContext       `json:"context,omitempty"`
	Options    ExecutionOptions       `json:"options,omitempty"`
}

// ExecutionContext provides context for tool execution
type ExecutionContext struct {
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ExecutionOptions provides options for tool execution
type ExecutionOptions struct {
	Timeout     time.Duration `json:"timeout,omitempty"`
	MaxRetries  int           `json:"max_retries,omitempty"`
	Priority    int           `json:"priority,omitempty"`
	CostLimit   CostLimit     `json:"cost_limit,omitempty"`
	AsyncMode   bool          `json:"async_mode,omitempty"`
	CallbackURL string        `json:"callback_url,omitempty"`
}

// ToolExecutionResponse contains the result of tool execution
type ToolExecutionResponse struct {
	Success     bool              `json:"success"`
	ExecutionID string            `json:"execution_id"`
	ToolID      string            `json:"tool_id"`
	Result      json.RawMessage   `json:"result,omitempty"`
	Error       *Error            `json:"error,omitempty"`
	Logs        []ExecutionLog    `json:"logs,omitempty"`
	Cost        CostReport        `json:"cost"`
	Duration    time.Duration     `json:"duration"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ExecutionLog represents a log entry from tool execution
type ExecutionLog struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Data      map[string]string `json:"data,omitempty"`
}
