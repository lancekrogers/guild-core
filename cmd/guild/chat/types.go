// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"github.com/guild-ventures/guild-core/cmd/guild/chat/common"
)

// Re-export common types for convenience
type MessageType = common.MessageType
type ChatMessage = common.ChatMessage
type ToolExecution = common.ToolExecution
type ViewMode = common.ViewMode
type CompletionResult = common.CompletionResult

// Re-export constants
const (
	MsgUser         = common.MsgUser
	MsgAgent        = common.MsgAgent
	MsgSystem       = common.MsgSystem
	MsgError        = common.MsgError
	MsgToolStart    = common.MsgToolStart
	MsgToolProgress = common.MsgToolProgress
	MsgToolComplete = common.MsgToolComplete
	MsgAgentThinking = common.MsgAgentThinking
	MsgAgentWorking = common.MsgAgentWorking
	MsgPrompt       = common.MsgPrompt
	MsgToolError    = common.MsgToolError
	MsgToolAuth     = common.MsgToolAuth
)

const (
	ViewChat        = common.ViewChat
	ViewSearch      = common.ViewSearch
	ViewHelp        = common.ViewHelp
	ViewSettings    = common.ViewSettings
	ViewModeNormal  = common.ViewChat  // Alias for compatibility
)