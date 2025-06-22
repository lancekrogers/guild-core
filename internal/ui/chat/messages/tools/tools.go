// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

// ToolExecutionStartMsg represents the start of tool execution
type ToolExecutionStartMsg struct {
	ExecutionID string
	ToolName    string
	AgentID     string
	Parameters  map[string]string
}

// ToolExecutionProgressMsg represents tool execution progress updates
type ToolExecutionProgressMsg struct {
	ExecutionID string
	Progress    float32
}

// ToolExecutionCompleteMsg represents completed tool execution
type ToolExecutionCompleteMsg struct {
	ExecutionID string
	Result      string
}

// ToolExecutionErrorMsg represents tool execution errors
type ToolExecutionErrorMsg struct {
	ExecutionID string
	Err         error
}

// ToolAuthRequiredMsg represents tool authorization requirements
type ToolAuthRequiredMsg struct {
	ToolName string
	AuthURL  string
	Message  string
}
