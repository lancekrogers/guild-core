// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package common

import (
	"github.com/lancekrogers/guild-core/internal/ui/chat/common/config"
	"github.com/lancekrogers/guild-core/internal/ui/chat/common/types"
)

// Re-export types types for convenience
type (
	// Common Types
	MessageType   = types.MessageType
	ChatMessage   = types.ChatMessage
	ToolExecution = types.ToolExecution
	ViewMode      = types.ViewMode

	// Chat Config
	ChatConfig = config.ChatConfig
)

// Re-export constants
const (
	MsgUser          = types.MsgUser
	MsgAgent         = types.MsgAgent
	MsgSystem        = types.MsgSystem
	MsgError         = types.MsgError
	MsgToolStart     = types.MsgToolStart
	MsgToolProgress  = types.MsgToolProgress
	MsgToolComplete  = types.MsgToolComplete
	MsgAgentThinking = types.MsgAgentThinking
	MsgAgentWorking  = types.MsgAgentWorking
	MsgPrompt        = types.MsgPrompt
	MsgToolError     = types.MsgToolError
	MsgToolAuth      = types.MsgToolAuth
)

const (
	ViewChat       = types.ViewChat
	ViewSearch     = types.ViewSearch
	ViewHelp       = types.ViewHelp
	ViewSettings   = types.ViewSettings
	ViewModeNormal = types.ViewChat // Alias for compatibility
)
