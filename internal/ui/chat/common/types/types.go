// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"time"
)

// Message types for the chat interface
type MessageType int

const (
	MsgUser MessageType = iota
	MsgAgent
	MsgSystem
	MsgError
	MsgToolStart
	MsgToolProgress
	MsgToolComplete
	MsgAgentThinking
	MsgAgentWorking
	MsgPrompt
	MsgToolError
	MsgToolAuth
)

// ChatMessage represents a single message in the chat history
type ChatMessage struct {
	Type      MessageType
	Content   string
	AgentID   string
	Timestamp time.Time
	Metadata  map[string]string
}

// Tool execution tracking
type ToolExecution struct {
	ID        string
	ToolName  string
	AgentID   string
	StartTime time.Time
	EndTime   *time.Time
	Status    string
	Progress  float32
	Result    string
	Output    string
	Error     string
	Metadata  map[string]string
}

// View modes for the chat interface
type ViewMode int

const (
	ViewChat ViewMode = iota
	ViewSearch
	ViewHelp
	ViewSettings
)

// Chat configuration
type ChatConfig struct {
	Campaign        string
	MaxMessageWidth int
	EnableMarkdown  bool
	EnableColors    bool
	AutoScroll      bool
	Debug           bool
}
