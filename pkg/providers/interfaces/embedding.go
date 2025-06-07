package interfaces

// LegacyEmbeddingRequest represents a legacy request for text embeddings
// Deprecated: Use EmbeddingRequest from ai_provider.go instead
type LegacyEmbeddingRequest struct {
	Input []string `json:"input"`
	Text  string   `json:"text,omitempty"`   // Single text version
	Texts []string `json:"texts,omitempty"`  // Multiple texts version
	Model string   `json:"model"`
	User  string   `json:"user,omitempty"`
}

// LegacyEmbeddingResponse represents a legacy response from an embedding request
// Deprecated: Use EmbeddingResponse from ai_provider.go instead
type LegacyEmbeddingResponse struct {
	Object    string            `json:"object"`
	Data      []EmbeddingData   `json:"data"`
	Model     string            `json:"model"`
	Usage     EmbeddingUsage    `json:"usage"`
	Embedding []float32         `json:"embedding,omitempty"`     // Single embedding vector
	Embeddings [][]float32      `json:"embeddings,omitempty"`    // Multiple embedding vectors
	Dimensions int              `json:"dimensions,omitempty"`    // Dimensions of the embedding vector
	TokensUsed int              `json:"tokens_used,omitempty"`   // Number of tokens used
}

// EmbeddingData represents a single embedding vector
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage contains token usage statistics for embeddings
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
