// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

import (
	"context"
)

// AgentExecutor defines the interface for executing tasks on agents
type AgentExecutor interface {
	// Execute runs a task and returns the result
	Execute(ctx context.Context, taskID string, payload interface{}) (interface{}, error)
	
	// GetAgentID returns the ID of the agent
	GetAgentID() string
	
	// GetCapabilities returns what types of tasks this executor can handle
	GetCapabilities() []string
	
	// IsAvailable checks if the executor can accept new tasks
	IsAvailable() bool
}