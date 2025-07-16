// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

// CompletionRequest represents a request for completion
type CompletionRequest struct {
	Prompt           string                 `json:"prompt"`
	MaxTokens        int                    `json:"max_tokens"`
	Temperature      float64                `json:"temperature"`
	TopP             float64                `json:"top_p"`
	FrequencyPenalty float64                `json:"frequency_penalty"`
	PresencePenalty  float64                `json:"presence_penalty"`
	Stop             []string               `json:"stop"`
	StopTokens       []string               `json:"stop_tokens,omitempty"`
	Model            string                 `json:"model"`
	Stream           bool                   `json:"stream"`
	ResponseFormat   string                 `json:"response_format"`
	Metadata         map[string]string      `json:"metadata"`
	Options          map[string]interface{} `json:"options,omitempty"`
}

// CompletionResponse represents a response from a completion request
type CompletionResponse struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	Created      int64             `json:"created"`
	Model        string            `json:"model"`
	ModelUsed    string            `json:"model_used,omitempty"`
	Content      string            `json:"content"`
	Text         string            `json:"text,omitempty"`
	FinishReason string            `json:"finish_reason"`
	Usage        UsageInfo         `json:"usage"`
	TokensUsed   int               `json:"tokens_used,omitempty"`
	TokensInput  int               `json:"tokens_input,omitempty"`
	TokensOutput int               `json:"tokens_output,omitempty"`
	Metadata     map[string]string `json:"metadata"`
}

