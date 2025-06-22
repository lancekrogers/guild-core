// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package completion

import (
	"time"
)

// CompletionRequestMsg represents a request for completions
type CompletionRequestMsg struct {
	Input string
	Type  string
}

// CompletionResultMsg represents completion results
type CompletionResultMsg struct {
	Results   []CompletionResult
	ForInput  string    // The input this completion is for
	Timestamp time.Time // When the completion was generated
}

// GetForInput returns the input this completion is for
func (msg CompletionResultMsg) GetForInput() string {
	return msg.ForInput
}

// SuggestionRequestMsg is sent when suggestions are requested
type SuggestionRequestMsg struct {
	Input     string
	Timestamp time.Time
}

// Add ForInput and Timestamp fields to CompletionResultMsg
type EnhancedCompletionResultMsg struct {
	CompletionResultMsg
	ForInput  string
	Timestamp time.Time
}

// CompletionResult represents an auto-completion suggestion
type CompletionResult struct {
	Content  string
	AgentID  string
	Metadata map[string]string
}
