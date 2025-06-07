package interfaces

import "context"

// LLMClient defines the interface for LLM clients
type LLMClient interface {
	// Complete generates completions for the given prompt
	Complete(ctx context.Context, prompt string) (string, error)
}
