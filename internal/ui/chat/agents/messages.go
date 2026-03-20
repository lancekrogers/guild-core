// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2
package agents

import (
	"time"

	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
)

// Message types for agent communication

// AgentResponseMsg represents a response from a specific agent
type AgentResponseMsg struct {
	AgentID   string
	Content   string
	MessageID string
	Timestamp time.Time
}

// BroadcastResponseMsg represents responses from multiple agents
type BroadcastResponseMsg struct {
	Responses []*pb.AgentMessageResponse
	MessageID string
	Timestamp time.Time
}

// AgentErrorMsg represents an error in agent communication
type AgentErrorMsg struct {
	AgentID string
	Error   error
}

// AgentListUpdatedMsg indicates the agent list has been refreshed
type AgentListUpdatedMsg struct {
	Agents []*pb.AgentInfo
}

// AgentStatusMsg represents an agent status update
type AgentStatusMsg struct {
	AgentID string
	Status  *pb.AgentStatus
}

// AgentStreamStartedMsg indicates a streaming session has started
type AgentStreamStartedMsg struct {
	AgentID string
}

// AgentStreamMsg represents streaming content from an agent
type AgentStreamMsg struct {
	AgentID string
	Content string
	Done    bool
}
