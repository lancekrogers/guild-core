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

// UsageInfo contains token usage statistics
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
